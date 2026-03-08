package pubsub

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/ariandi/gocom"
	"github.com/oklog/ulid/v2"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/sender"
	csDtos "gitlab.com/bot3342545/il-dashboard/customer_service/dtos"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/repositories/customerServiceCases"
	"gitlab.com/bot3342545/il-dashboard/repositories/messageConversation"
	"gitlab.com/bot3342545/il-dashboard/services"
)

// ============================================================================
// Message Router - Route WA messages to appropriate topics
// ============================================================================

// MessageRouter routes WhatsApp messages to different handlers via pubsub
type MessageRouter struct {
}

// NewMessageRouter creates new message router
func NewMessageRouter() *MessageRouter {
	return &MessageRouter{}
}

// RouteMessage routes incoming WhatsApp message to appropriate pubsub topic
func (r *MessageRouter) RouteMessage(fromNumber, toNumber, message, messageType, contactName, timestamp, waMessageID string) {
	customLogger.InfoWithData("Routing incoming message", map[string]interface{}{
		"component":   "MessageRouter",
		"from_number": fromNumber,
		"to_number":   toNumber,
		"message":     message,
		"message_id":  waMessageID,
	})

	// Get client ID from business phone number
	clientID := r.getClientIdFromPhoneNumber(toNumber)

	// Create session ID
	sessionID := ulid.Make().String()

	// PRIORITY: Check if user clicked CS escalation button
	if r.isCSEscalationButtonClick(message) {
		customLogger.InfoWithData("CS escalation button clicked", map[string]interface{}{
			"component":   "MessageRouter",
			"from_number": fromNumber,
			"action":      "auto_escalate",
		})
		r.autoEscalateToCS(sessionID, fromNumber, toNumber, message, contactName, clientID)
		return
	}

	// Detect message characteristics
	isGreeting := r.isGreeting(message)
	isPDAMRelated := r.isPDAMRelated(message)
	isNumberOnly := r.isNumberOnly(message)
	isConversationActive := messageConversation.GetRepo().IsConversationActive(fromNumber)

	// CRITICAL: Check if user has active CS case (regardless of escalation timeout)
	activeCase := r.getActiveCaseForUser(clientID, fromNumber)
	if activeCase != nil {
		customLogger.InfoWithData("User has active CS case", map[string]interface{}{
			"component":   "MessageRouter",
			"from_number": fromNumber,
			"case_number": activeCase.CaseNumber,
			"case_status": activeCase.Status,
			"routing":     "CS Hub",
		})
		r.routeToCSHub(fromNumber, toNumber, message, messageType, contactName, timestamp, waMessageID, clientID)
		return
	}

	// Determine routing category
	routingCategory := r.determineRoutingCategory(message, isGreeting, isPDAMRelated, isNumberOnly, isConversationActive)

	customLogger.InfoWithData("Routing decision made", map[string]interface{}{
		"component":              "MessageRouter",
		"category":               routingCategory,
		"is_greeting":            isGreeting,
		"is_pdam_related":        isPDAMRelated,
		"is_number_only":         isNumberOnly,
		"is_conversation_active": isConversationActive,
		"from_number":            fromNumber,
	})

	// Create message event
	event := dtos.WAMessageEvent{
		SessionID:         sessionID,
		FromNumber:        fromNumber,
		ToNumber:          toNumber,
		Message:           message,
		MessageType:       messageType,
		ClientID:          clientID,
		ContactName:       contactName,
		Timestamp:         timestamp,
		WhatsAppMessageID: waMessageID,
		RoutingCategory:   routingCategory,
		IsGreeting:        isGreeting,
		IsPDAMRelated:     isPDAMRelated,
		IsNumberOnly:      isNumberOnly,
	}

	// Route to appropriate topic
	r.publishToTopic(event)
}

// determineRoutingCategory determines which topic to route the message to
func (r *MessageRouter) determineRoutingCategory(message string, isGreeting, isPDAMRelated, isNumberOnly, isConversationActive bool) string {
	// Priority 1: PPOB flow (number only)
	if isNumberOnly {
		return "ppob"
	}

	// Priority 2: Check if user explicitly requests CS
	if r.isCSRequest(message) {
		return "cs"
	}

	// Priority 3: Active conversation with PDAM topic
	if isConversationActive && isPDAMRelated {
		// Check if it's a complex complaint that might need CS
		if r.isComplexComplaint(message) {
			return "cs"
		}
		return "knowledge"
	}

	// Priority 4: Greeting or first message
	if isGreeting {
		return "knowledge"
	}

	// Priority 5: PDAM related but no active conversation
	if isPDAMRelated {
		return "knowledge"
	}

	// Default: General AI with potential CS escalation
	return "knowledge"
}

// publishToTopic publishes message event to appropriate pubsub topic
func (r *MessageRouter) publishToTopic(event dtos.WAMessageEvent) {
	var topic string

	switch event.RoutingCategory {
	case "ppob":
		topic = dtos.TopicWAMessageAIPPOB
	case "cs":
		topic = dtos.TopicWAMessageAICS
	case "knowledge":
		topic = dtos.TopicWAMessageAIKnowledge
	default:
		topic = dtos.TopicWAMessageAIKnowledge
	}

	// Marshal event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		customLogger.ErrorWithData("Failed to marshal event", map[string]interface{}{
			"component":   "MessageRouter",
			"error":       err.Error(),
			"from_number": event.FromNumber,
		})
		return
	}

	// Publish to pubsub
	if err := gocom.PubSub().Publish(topic, string(eventJSON)); err != nil {
		customLogger.ErrorWithData("Failed to publish to topic", map[string]interface{}{
			"component":   "MessageRouter",
			"topic":       topic,
			"error":       err.Error(),
			"from_number": event.FromNumber,
		})
		return
	}

	customLogger.InfoWithData("Message published to topic", map[string]interface{}{
		"component":   "MessageRouter",
		"topic":       topic,
		"category":    event.RoutingCategory,
		"from_number": event.FromNumber,
	})
}

// isGreeting checks if message is a greeting
func (r *MessageRouter) isGreeting(message string) bool {
	lowerMsg := strings.ToLower(strings.TrimSpace(message))
	greetings := []string{
		"halo", "hello", "hai", "hi", "hey",
		"pagi", "siang", "sore", "malam",
		"permisi", "assalamualaikum", "salam",
	}

	for _, greeting := range greetings {
		if strings.Contains(lowerMsg, greeting) {
			return true
		}
	}
	return false
}

// isPDAMRelated checks if message is PDAM related
func (r *MessageRouter) isPDAMRelated(message string) bool {
	lowerMsg := strings.ToLower(strings.TrimSpace(message))
	pdamKeywords := []string{
		"air", "pdam", "tagihan", "bayar", "pembayaran",
		"meter", "meteran", "pelanggan", "rekening",
		"bocor", "mati", "keruh", "bau", "gangguan",
		"pasang", "tutup", "balik nama", "kantor",
	}

	for _, keyword := range pdamKeywords {
		if strings.Contains(lowerMsg, keyword) {
			return true
		}
	}
	return false
}

// isNumberOnly checks if message contains only customer number
func (r *MessageRouter) isNumberOnly(message string) bool {
	trimmed := strings.TrimSpace(message)

	// Check if it's alphanumeric (customer number format)
	matched, _ := regexp.MatchString(`^[A-Za-z0-9]{6,15}$`, trimmed)
	return matched
}

// isCSRequest checks if user explicitly requests customer service
func (r *MessageRouter) isCSRequest(message string) bool {
	lowerMsg := strings.ToLower(strings.TrimSpace(message))
	csKeywords := []string{
		"customer service", "cs", "operator", "manusia",
		"bicara dengan", "hubungkan dengan", "minta tolong",
		"komplain", "lapor", "pengaduan serius",
	}

	for _, keyword := range csKeywords {
		if strings.Contains(lowerMsg, keyword) {
			return true
		}
	}
	return false
}

// isComplexComplaint checks if message is a complex complaint
func (r *MessageRouter) isComplexComplaint(message string) bool {
	lowerMsg := strings.ToLower(strings.TrimSpace(message))
	complexIndicators := []string{
		"sudah lapor", "belum ditangani", "tidak ada respon",
		"berapa lama", "kapan selesai", "sudah berapa",
		"masih belum", "kenapa belum", "sudah berhari",
	}

	for _, indicator := range complexIndicators {
		if strings.Contains(lowerMsg, indicator) {
			return true
		}
	}
	return false
}

// getClientIdFromPhoneNumber maps business phone number to client_id
func (r *MessageRouter) getClientIdFromPhoneNumber(phoneNumber string) string {
	if phoneNumber == "" {
		customLogger.WarnWithData("Empty phone number provided", map[string]interface{}{
			"component": "MessageRouter",
			"function":  "getClientIdFromPhoneNumber",
			"fallback":  common.PROVIDER_ID,
		})
		return common.PROVIDER_ID // Fallback to default
	}

	// Lookup sender by phone number identifier
	senderData := sender.GetRepo().GetByIdentifier(phoneNumber)
	if senderData == nil {
		customLogger.WarnWithData("No sender found for phone number", map[string]interface{}{
			"component":    "MessageRouter",
			"function":     "getClientIdFromPhoneNumber",
			"phone_number": phoneNumber,
			"fallback":     common.PROVIDER_ID,
		})
		return common.PROVIDER_ID // Fallback to default
	}

	customLogger.InfoWithData("Found sender", map[string]interface{}{
		"component":    "MessageRouter",
		"function":     "getClientIdFromPhoneNumber",
		"phone_number": phoneNumber,
		"client_id":    senderData.ClientId,
		"sender_name":  senderData.Name,
	})
	return senderData.ClientId
}

// routeToCSHub routes user message to CS Hub when conversation is escalated
func (r *MessageRouter) routeToCSHub(fromNumber, toNumber, message, messageType, contactName, timestamp, waMessageID, clientID string) {
	customLogger.InfoWithData("Routing message to CS Hub", map[string]interface{}{
		"component":   "MessageRouter",
		"function":    "routeToCSHub",
		"from_number": fromNumber,
		"client_id":   clientID,
	})

	// Get active CS case for this user
	activeCase := r.getActiveCaseForUser(clientID, fromNumber)
	if activeCase == nil {
		customLogger.WarnWithData("No active CS case found", map[string]interface{}{
			"component":   "MessageRouter",
			"function":    "routeToCSHub",
			"from_number": fromNumber,
			"client_id":   clientID,
		})
		return
	}

	// Create CS user message event
	event := csDtos.CSCaseMessageReceivedEvent{
		MessageID:      waMessageID,
		CaseID:         activeCase.ID,
		CaseNumber:     activeCase.CaseNumber,
		ClientId:       clientID,
		UserPhone:      fromNumber,
		UserName:       contactName,
		MessageContent: message,
		MessageType:    messageType,
		ReceivedAt:     time.Now(),
	}

	// Marshal to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		customLogger.ErrorWithData("Failed to marshal CS event", map[string]interface{}{
			"component":   "MessageRouter",
			"function":    "routeToCSHub",
			"error":       err.Error(),
			"case_id":     activeCase.ID,
			"case_number": activeCase.CaseNumber,
		})
		return
	}

	// Publish to CS Hub topic
	if err := gocom.PubSub().Publish(csDtos.TopicCSCaseMessageReceived, string(eventJSON)); err != nil {
		customLogger.ErrorWithData("Failed to publish to CS Hub topic", map[string]interface{}{
			"component":   "MessageRouter",
			"function":    "routeToCSHub",
			"topic":       csDtos.TopicCSCaseMessageReceived,
			"error":       err.Error(),
			"case_id":     activeCase.ID,
		})
		return
	}

	customLogger.InfoWithData("User message routed to CS case", map[string]interface{}{
		"component":   "MessageRouter",
		"function":    "routeToCSHub",
		"case_id":     activeCase.ID,
		"case_number": activeCase.CaseNumber,
		"from_number": fromNumber,
	})
}

// getActiveCaseForUser gets active CS case for user
func (r *MessageRouter) getActiveCaseForUser(clientID, userPhone string) *customerServiceCases.CustomerServiceCase {
	customLogger.DebugWithData("Getting active case for user", map[string]interface{}{
		"component":  "MessageRouter",
		"function":   "getActiveCaseForUser",
		"user_phone": userPhone,
		"client_id":  clientID,
	})

	// Get active case from CS repository
	activeCase := customerServiceCases.GetRepo().GetActiveCase(clientID, userPhone)
	if activeCase == nil {
		customLogger.DebugWithData("No active case found", map[string]interface{}{
			"component":  "MessageRouter",
			"function":   "getActiveCaseForUser",
			"user_phone": userPhone,
			"client_id":  clientID,
		})
		return nil
	}

	customLogger.InfoWithData("Found active case", map[string]interface{}{
		"component":   "MessageRouter",
		"function":    "getActiveCaseForUser",
		"case_id":     activeCase.ID,
		"case_number": activeCase.CaseNumber,
		"user_phone":  userPhone,
		"status":      activeCase.Status,
	})
	return activeCase
}

// isCSEscalationButtonClick checks if message is from CS escalation button click
func (r *MessageRouter) isCSEscalationButtonClick(message string) bool {
	trimmed := strings.TrimSpace(message)
	// Exact match for button text
	return trimmed == "Hubungkan dengan CS"
}

// autoEscalateToCS automatically creates CS case when user clicks escalation button
func (r *MessageRouter) autoEscalateToCS(sessionID, fromNumber, toNumber, message, contactName, clientID string) {
	customLogger.InfoWithData("Auto-escalating user to CS", map[string]interface{}{
		"component":   "MessageRouter",
		"function":    "autoEscalateToCS",
		"from_number": fromNumber,
		"client_id":   clientID,
		"reason":      "button_click",
	})

	// Publish AI Escalation Confirmed event (bypass AI processing)
	event := csDtos.AIEscalationConfirmedEvent{
		SessionID:      sessionID,
		ClientId:       clientID,
		UserPhone:      fromNumber,
		UserName:       contactName,
		UserMessage:    message,
		ConfirmMessage: message,
		Category:       "umum", // General category for button escalation
		Metadata: map[string]interface{}{
			"reason":          "User clicked CS escalation button",
			"auto_escalate":   true,
			"escalation_type": "button_click",
		},
		ConfirmedAt: time.Now(),
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		customLogger.ErrorWithData("Failed to marshal escalation event", map[string]interface{}{
			"component":   "MessageRouter",
			"function":    "autoEscalateToCS",
			"error":       err.Error(),
			"from_number": fromNumber,
		})
		// Send fallback message
		services.GetWASendSvc().SendTextMessageWithLog(
			fromNumber,
			"Maaf, terjadi kesalahan saat menghubungkan dengan CS. Silakan coba lagi.",
			clientID,
			toNumber,
		)
		return
	}

	// Publish to AI Escalation Confirmed topic
	if err := gocom.PubSub().Publish(csDtos.TopicAIEscalationConfirmed, string(eventJSON)); err != nil {
		customLogger.ErrorWithData("Failed to publish escalation event", map[string]interface{}{
			"component":   "MessageRouter",
			"function":    "autoEscalateToCS",
			"topic":       csDtos.TopicAIEscalationConfirmed,
			"error":       err.Error(),
			"from_number": fromNumber,
		})
		// Send fallback message
		services.GetWASendSvc().SendTextMessageWithLog(
			fromNumber,
			"Maaf, terjadi kesalahan saat menghubungkan dengan CS. Silakan coba lagi.",
			clientID,
			toNumber,
		)
		return
	}

	customLogger.InfoWithData("CS escalation event published successfully", map[string]interface{}{
		"component":   "MessageRouter",
		"function":    "autoEscalateToCS",
		"from_number": fromNumber,
		"session_id":  sessionID,
		"status":      "success",
	})

	// Send confirmation message to user
	confirmMsg := "Baik, saya akan menghubungkan Anda dengan customer service kami. Mohon tunggu sebentar..."
	services.GetWASendSvc().SendTextMessageWithLog(fromNumber, confirmMsg, clientID, toNumber)
}
