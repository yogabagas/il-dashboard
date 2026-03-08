package pubsub

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/ariandi/gocom"
	csdtos "gitlab.com/bot3342545/il-dashboard/customer_service/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/repositories/messageConversation"
	"gitlab.com/bot3342545/il-dashboard/services"
)

// ============================================================================
// CS Escalation Handler
// ============================================================================

// CSEscalationHandler handles AI → CS escalation flow
type CSEscalationHandler struct {
}

// NewCSEscalationHandler creates new CS escalation handler
func NewCSEscalationHandler() *CSEscalationHandler {
	return &CSEscalationHandler{}
}

// TriggerAIWithEscalation handles AI conversation with escalation support
func (h *CSEscalationHandler) TriggerAIWithEscalation(sessionID, fromNumber, userMessage, businessPhoneNumber string) {
	customLogger.InfoWithData("Processing message for AI with escalation", map[string]interface{}{
		"component":   "CSEscalationHandler",
		"function":    "TriggerAIWithEscalation",
		"from_number": fromNumber,
		"message":     userMessage,
		"session_id":  sessionID,
	})

	// Get client_id from business phone number
	clientId := h.getClientIdFromPhoneNumber(businessPhoneNumber)

	// Call AI service with escalation support
	aiResponse, escalationInfo, codedErr := services.GetGroqSvc().ChatWithEscalation(userMessage, clientId, fromNumber)
	if codedErr != nil {
		customLogger.ErrorWithData("Failed to get AI response", map[string]interface{}{
			"component":   "CSEscalationHandler",
			"function":    "TriggerAIWithEscalation",
			"error":       codedErr.Message,
			"from_number": fromNumber,
			"client_id":   clientId,
		})
		return
	}

	// Save user message to conversation
	userConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         fromNumber,
		Role:               "user",
		Message:            userMessage,
		Status:             "received",
		ConversationStatus: "active",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(userConv)

	// Handle CS escalation if detected
	if escalationInfo != nil {
		customLogger.InfoWithData("AI escalating to CS", map[string]interface{}{
			"component":   "CSEscalationHandler",
			"function":    "TriggerAIWithEscalation",
			"from_number": fromNumber,
			"reason":      escalationInfo.Reason,
			"category":    escalationInfo.Category,
		})

		// Publish escalation event to CS Hub
		h.publishEscalationEvent(sessionID, clientId, fromNumber, escalationInfo, userMessage)

		// Send escalation message to user
		h.sendEscalationResponse(sessionID, clientId, fromNumber, aiResponse, businessPhoneNumber)
		return
	}

	// Normal AI response (no escalation)
	waMessageID, messageLogID, sendErr := services.GetWASendSvc().SendTextMessageWithLog(fromNumber, aiResponse, clientId, businessPhoneNumber)
	if sendErr != nil {
		customLogger.ErrorWithData("Failed to send AI response", map[string]interface{}{
			"component":   "CSEscalationHandler",
			"function":    "TriggerAIWithEscalation",
			"error":       sendErr.Message,
			"from_number": fromNumber,
			"client_id":   clientId,
		})
		return
	}

	// Save AI response to conversation
	assistantConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         fromNumber,
		ToNumber:           fromNumber,
		Role:               "assistant",
		Message:            aiResponse,
		MessageLogId:       messageLogID,
		Status:             "sent",
		ConversationStatus: "active",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(assistantConv)

	customLogger.InfoWithData("AI response sent successfully", map[string]interface{}{
		"component":      "CSEscalationHandler",
		"function":       "TriggerAIWithEscalation",
		"message_id":     waMessageID,
		"from_number":    fromNumber,
		"message_log_id": messageLogID,
	})
}

// publishEscalationEvent publishes AI escalation event to CS Hub
func (h *CSEscalationHandler) publishEscalationEvent(sessionID, clientId, userPhone string, escalationInfo *services.EscalationInfo, userMessage string) {
	customLogger.InfoWithData("Publishing escalation event", map[string]interface{}{
		"component":  "CSEscalationHandler",
		"function":   "publishEscalationEvent",
		"client_id":  clientId,
		"user_phone": userPhone,
		"category":   escalationInfo.Category,
		"session_id": sessionID,
	})

	// Create escalation event
	event := csdtos.AIEscalationConfirmedEvent{
		SessionID:      sessionID,
		ClientId:       clientId,
		UserPhone:      userPhone,
		UserName:       "", // Will be filled by CS Hub from contact
		UserMessage:    escalationInfo.UserMessage,
		ConfirmMessage: userMessage, // The message that triggered escalation
		Category:       escalationInfo.Category,
		Metadata: map[string]interface{}{
			"reason":        escalationInfo.Reason,
			"auto_escalate": true, // Automatic escalation by AI
		},
		ConfirmedAt: time.Now(),
	}

	// Marshal to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		customLogger.ErrorWithData("Failed to marshal escalation event", map[string]interface{}{
			"component":  "CSEscalationHandler",
			"function":   "publishEscalationEvent",
			"error":      err.Error(),
			"user_phone": userPhone,
			"category":   escalationInfo.Category,
		})
		return
	}

	// Publish to pubsub
	if err := gocom.PubSub().Publish(csdtos.TopicAIEscalationConfirmed, string(eventJSON)); err != nil {
		customLogger.ErrorWithData("Failed to publish escalation event", map[string]interface{}{
			"component":  "CSEscalationHandler",
			"function":   "publishEscalationEvent",
			"topic":      csdtos.TopicAIEscalationConfirmed,
			"error":      err.Error(),
			"user_phone": userPhone,
		})
		return
	}

	customLogger.InfoWithData("Escalation event published successfully", map[string]interface{}{
		"component":  "CSEscalationHandler",
		"function":   "publishEscalationEvent",
		"user_phone": userPhone,
		"category":   escalationInfo.Category,
		"session_id": sessionID,
	})
}

// sendEscalationResponse sends escalation message to user and saves to conversation
func (h *CSEscalationHandler) sendEscalationResponse(sessionID, clientId, fromNumber, message, businessPhoneNumber string) {
	customLogger.InfoWithData("Sending escalation response", map[string]interface{}{
		"component":   "CSEscalationHandler",
		"function":    "sendEscalationResponse",
		"from_number": fromNumber,
		"client_id":   clientId,
	})

	// Send escalation message to user
	waMessageID, messageLogID, sendErr := services.GetWASendSvc().SendTextMessageWithLog(fromNumber, message, clientId, businessPhoneNumber)
	if sendErr != nil {
		customLogger.ErrorWithData("Failed to send escalation message", map[string]interface{}{
			"component":   "CSEscalationHandler",
			"function":    "sendEscalationResponse",
			"error":       sendErr.Message,
			"from_number": fromNumber,
			"client_id":   clientId,
		})
		return
	}

	// Save AI response to conversation
	assistantConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         fromNumber,
		ToNumber:           fromNumber,
		Role:               "assistant",
		Message:            message,
		MessageLogId:       messageLogID,
		Status:             "sent",
		ConversationStatus: "escalated",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(assistantConv)

	customLogger.InfoWithData("Escalation message sent successfully", map[string]interface{}{
		"component":      "CSEscalationHandler",
		"function":       "sendEscalationResponse",
		"message_id":     waMessageID,
		"from_number":    fromNumber,
		"message_log_id": messageLogID,
		"status":         "escalated",
	})
}

// TriggerSimpleKnowledgeChat handles simple knowledge-based conversation WITHOUT function calling
// This is used for general PDAM questions that don't need bill inquiry or payment functions
func (h *CSEscalationHandler) TriggerSimpleKnowledgeChat(sessionID, fromNumber, userMessage, businessPhoneNumber string) {
	customLogger.InfoWithData("Processing simple knowledge chat", map[string]interface{}{
		"component":   "CSEscalationHandler",
		"function":    "TriggerSimpleKnowledgeChat",
		"from_number": fromNumber,
		"message":     userMessage,
		"session_id":  sessionID,
	})

	// Get client_id from business phone number
	clientId := h.getClientIdFromPhoneNumber(businessPhoneNumber)

	// Call AI service with simple knowledge chat (NO function calling)
	// Pass fromNumber to enable conversation history context
	aiResponse, codedErr := services.GetGroqSvc().ChatWithKnowledge(userMessage, clientId, fromNumber)
	if codedErr != nil {
		customLogger.ErrorWithData("Failed to get AI response", map[string]interface{}{
			"component":   "CSEscalationHandler",
			"function":    "TriggerSimpleKnowledgeChat",
			"error":       codedErr.Message,
			"from_number": fromNumber,
			"client_id":   clientId,
		})

		// Send fallback error message to user
		fallbackMsg := "Maaf, terjadi kesalahan. Silakan hubungi customer service kami di (021) 5951234."
		services.GetWASendSvc().SendTextMessageWithLog(fromNumber, fallbackMsg, clientId, businessPhoneNumber)
		return
	}

	// Save user message to conversation
	userConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         fromNumber,
		Role:               "user",
		Message:            userMessage,
		Status:             "received",
		ConversationStatus: "active",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(userConv)

	// Detect if this is a greeting (initial user message)
	isGreeting := h.isGreetingMessage(userMessage)

	var waMessageID, messageLogID string
	var sendErr *gocom.CodedError

	if isGreeting {
		// For greeting, send response WITHOUT button
		customLogger.InfoWithData("Greeting detected - sending without button", map[string]interface{}{
			"component":   "CSEscalationHandler",
			"function":    "TriggerSimpleKnowledgeChat",
			"from_number": fromNumber,
			"is_greeting": true,
		})
		waMessageID, messageLogID, sendErr = services.GetWASendSvc().SendTextMessageWithLog(fromNumber, aiResponse, clientId, businessPhoneNumber)
	} else {
		// For non-greeting, send response WITH CS escalation button
		customLogger.InfoWithData("Non-greeting - sending with CS button", map[string]interface{}{
			"component":   "CSEscalationHandler",
			"function":    "TriggerSimpleKnowledgeChat",
			"from_number": fromNumber,
			"is_greeting": false,
			"has_button":  true,
		})
		buttons := []map[string]string{
			{
				"id":    "btn_escalate_cs",
				"title": "Hubungkan dengan CS",
			},
		}
		waMessageID, messageLogID, sendErr = services.GetWASendSvc().SendInteractiveButtonMessage(fromNumber, aiResponse, buttons, clientId, businessPhoneNumber)
	}

	if sendErr != nil {
		customLogger.ErrorWithData("Failed to send AI response", map[string]interface{}{
			"component":   "CSEscalationHandler",
			"function":    "TriggerSimpleKnowledgeChat",
			"error":       sendErr.Message,
			"from_number": fromNumber,
			"is_greeting": isGreeting,
		})
		return
	}

	// Save AI response to conversation
	assistantConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         fromNumber,
		ToNumber:           fromNumber,
		Role:               "assistant",
		Message:            aiResponse,
		MessageLogId:       messageLogID,
		Status:             "sent",
		ConversationStatus: "active",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(assistantConv)

	customLogger.InfoWithData("Simple knowledge response sent successfully", map[string]interface{}{
		"component":      "CSEscalationHandler",
		"function":       "TriggerSimpleKnowledgeChat",
		"message_id":     waMessageID,
		"from_number":    fromNumber,
		"is_greeting":    isGreeting,
		"message_log_id": messageLogID,
	})
}

// isGreetingMessage checks if message is a greeting (initial contact)
func (h *CSEscalationHandler) isGreetingMessage(message string) bool {
	lowerMsg := strings.ToLower(strings.TrimSpace(message))

	greetingWords := []string{
		"halo", "hello", "hai", "hi",
		"pagi", "siang", "sore", "malam",
		"permisi", "assalamualaikum", "assalamu'alaikum",
	}

	for _, word := range greetingWords {
		if strings.Contains(lowerMsg, word) {
			return true
		}
	}

	return false
}

// getClientIdFromPhoneNumber maps business phone number to client_id
func (h *CSEscalationHandler) getClientIdFromPhoneNumber(phoneNumber string) string {
	// This should match the logic in updateStatus.go
	// For now, return a default client ID
	// TODO: Implement proper mapping from config or database
	return "01JPETM6DBJ1TPSKZNKNSE5HB6" // Default client ID
}
