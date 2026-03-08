package pubsub

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	"github.com/ariandi/gocom/pubsub"
	"gitlab.com/bot3342545/il-dashboard/customer_service/dtos"
	"gitlab.com/bot3342545/il-dashboard/customer_service/services"
	"gitlab.com/bot3342545/il-dashboard/repositories/customerServiceCases"
)

// ============================================================================
// Pubsub Handler Implementation
// ============================================================================

type CSPubSubHandler interface {
	Init()
	PublishCaseCreated(event *dtos.CSCaseCreatedEvent) error
	PublishCaseClaimed(event *dtos.CSCaseClaimedEvent) error
	PublishCaseEscalated(event *dtos.CSCaseEscalatedEvent) error
	PublishCaseResolved(event *dtos.CSCaseResolvedEvent) error
	PublishCaseClosed(event *dtos.CSCaseClosedEvent) error
	PublishMessageSent(event *dtos.CSCaseMessageSentEvent) error
	PublishMessageReceived(event *dtos.CSCaseMessageReceivedEvent) error
	PublishSLABreached(event *dtos.CSCaseSLABreachedEvent) error
}

type CSPubSubHandlerImpl struct {
	caseService services.CustomerServiceSvc
}

func NewCSPubSubHandler(caseService services.CustomerServiceSvc) *CSPubSubHandlerImpl {
	return &CSPubSubHandlerImpl{
		caseService: caseService,
	}
}

// Init initializes all pubsub subscriptions
func (h *CSPubSubHandlerImpl) Init() {
	logger.Infof("[CSPubSubHandler] Initializing CS Hub pubsub handlers...")

	// Subscribe to AI escalation events
	pubsub.Get().Subscribe(dtos.TopicAIEscalationConfirmed, h.handleAIEscalationConfirmed())

	// Subscribe to user message events for CS case routing
	pubsub.Get().Subscribe(dtos.TopicUserMessageClassified, h.handleUserMessageClassified())

	// Subscribe to CS case message events
	pubsub.Get().Subscribe(dtos.TopicCSCaseMessageReceived, h.handleCSCaseMessageReceived())

	// Subscribe to case events for internal processing
	pubsub.Get().Subscribe(dtos.TopicCSCaseClosed, h.handleCaseClosed())
	pubsub.Get().Subscribe(dtos.TopicCSCaseSLABreached, h.handleSLABreached())

	logger.Infof("[CSPubSubHandler] CS Hub pubsub handlers initialized")
}

// ============================================================================
// Event Publishers - untuk publish events ke pubsub
// ============================================================================

func (h *CSPubSubHandlerImpl) PublishCaseCreated(event *dtos.CSCaseCreatedEvent) error {
	return h.publishEvent(dtos.TopicCSCaseCreated, event)
}

func (h *CSPubSubHandlerImpl) PublishCaseClaimed(event *dtos.CSCaseClaimedEvent) error {
	return h.publishEvent(dtos.TopicCSCaseClaimed, event)
}

func (h *CSPubSubHandlerImpl) PublishCaseEscalated(event *dtos.CSCaseEscalatedEvent) error {
	return h.publishEvent(dtos.TopicCSCaseEscalated, event)
}

func (h *CSPubSubHandlerImpl) PublishCaseResolved(event *dtos.CSCaseResolvedEvent) error {
	return h.publishEvent(dtos.TopicCSCaseResolved, event)
}

func (h *CSPubSubHandlerImpl) PublishCaseClosed(event *dtos.CSCaseClosedEvent) error {
	return h.publishEvent(dtos.TopicCSCaseClosed, event)
}

func (h *CSPubSubHandlerImpl) PublishMessageSent(event *dtos.CSCaseMessageSentEvent) error {
	return h.publishEvent(dtos.TopicCSCaseMessageSent, event)
}

func (h *CSPubSubHandlerImpl) PublishMessageReceived(event *dtos.CSCaseMessageReceivedEvent) error {
	return h.publishEvent(dtos.TopicCSCaseMessageReceived, event)
}

func (h *CSPubSubHandlerImpl) PublishSLABreached(event *dtos.CSCaseSLABreachedEvent) error {
	return h.publishEvent(dtos.TopicCSCaseSLABreached, event)
}

// publishEvent is a generic publisher
func (h *CSPubSubHandlerImpl) publishEvent(topic string, event interface{}) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		logger.Errorf("[CSPubSubHandler] Failed to marshal event for topic %s: %v", topic, err)
		return err
	}

	if err := gocom.PubSub().Publish(topic, string(eventJSON)); err != nil {
		logger.Errorf("[CSPubSubHandler] Failed to publish to topic %s: %v", topic, err)
		return err
	}

	logger.Infof("[CSPubSubHandler] Published event to topic: %s", topic)
	return nil
}

// ============================================================================
// Event Subscribers - untuk handle events dari pubsub
// ============================================================================

// handleAIEscalationConfirmed handles when user confirms AI escalation
func (h *CSPubSubHandlerImpl) handleAIEscalationConfirmed() pubsub.PubSubEventHandler {
	logger.Infof("[CSPubSubHandler] Subscribing to topic: %s", dtos.TopicAIEscalationConfirmed)
	return func(topic, msg string) {
		logger.Infof("[CSPubSubHandler] Received AI escalation confirmation: %s", msg)

		var event dtos.AIEscalationConfirmedEvent
		if err := json.Unmarshal([]byte(msg), &event); err != nil {
			logger.Errorf("[CSPubSubHandler] Failed to unmarshal AIEscalationConfirmedEvent: %v", err)
			return
		}

		// IDEMPOTENCY CHECK: Check if user already has active case to prevent duplicates
		existingCase := customerServiceCases.GetRepo().GetActiveCase(event.ClientId, event.UserPhone)
		if existingCase != nil {
			logger.Warnf("[CSPubSubHandler] User %s already has active case %s (status: %s) - skipping duplicate case creation",
				event.UserPhone, existingCase.CaseNumber, existingCase.Status)

			// Publish existing case created event (for consistency)
			createdEvent := &dtos.CSCaseCreatedEvent{
				CaseID:      existingCase.ID,
				CaseNumber:  existingCase.CaseNumber,
				ClientId:    existingCase.ClientId,
				UserPhone:   existingCase.UserPhone,
				UserName:    existingCase.UserName,
				Category:    existingCase.Category,
				Subject:     existingCase.Subject,
				Status:      existingCase.Status,
				Severity:    existingCase.Severity,
				Priority:    existingCase.Priority,
				SlaDeadline: existingCase.SlaDeadline,
				SessionID:   event.SessionID,
				CreatedAt:   existingCase.CreatedAt,
			}
			h.PublishCaseCreated(createdEvent)
			return
		}

		// Create CS case
		logger.Infof("[CSPubSubHandler] Creating CS case for user: %s, session: %s", event.UserPhone, event.SessionID)

		caseData, err := h.caseService.CreateCase(
			event.ClientId,
			event.UserPhone,
			event.UserName,
			event.SessionID,
			event.Category,
			event.UserMessage,
		)

		if err != nil {
			logger.Errorf("[CSPubSubHandler] Failed to create CS case: %v", err)
			return
		}

		logger.Infof("[CSPubSubHandler] CS case created successfully: %s (Case #%s)", caseData.ID, caseData.CaseNumber)

		// Publish case created event
		createdEvent := &dtos.CSCaseCreatedEvent{
			CaseID:      caseData.ID,
			CaseNumber:  caseData.CaseNumber,
			ClientId:    caseData.ClientId,
			UserPhone:   caseData.UserPhone,
			UserName:    caseData.UserName,
			Category:    caseData.Category,
			Subject:     caseData.Subject,
			Status:      caseData.Status,
			Severity:    caseData.Severity,
			Priority:    caseData.Priority,
			SlaDeadline: caseData.SlaDeadline,
			SessionID:   event.SessionID,
			CreatedAt:   caseData.CreatedAt,
		}
		h.PublishCaseCreated(createdEvent)
	}
}

// handleUserMessageClassified handles classified user messages
func (h *CSPubSubHandlerImpl) handleUserMessageClassified() pubsub.PubSubEventHandler {
	logger.Infof("[CSPubSubHandler] Subscribing to topic: %s", dtos.TopicUserMessageClassified)
	return func(topic, msg string) {
		logger.Infof("[CSPubSubHandler] Received classified message: %s", msg)

		var classification dtos.MessageClassification
		if err := json.Unmarshal([]byte(msg), &classification); err != nil {
			logger.Errorf("[CSPubSubHandler] Failed to unmarshal MessageClassification: %v", err)
			return
		}

		// CRITICAL: Route message based on classification
		logger.Infof("[CSPubSubHandler] Message classification - Type: %s, HasActiveCSCase: %v, IsEscalation: %v",
			classification.Classification, classification.HasActiveCSCase, classification.IsEscalation)

		// If user has active CS case, route to that case
		if classification.HasActiveCSCase && classification.ActiveCaseID != "" {
			logger.Infof("[CSPubSubHandler] Routing message to active CS case: %s", classification.ActiveCaseID)
			h.routeMessageToCase(classification)
			return
		}

		// If message needs escalation and confidence is low
		if classification.IsEscalation {
			logger.Infof("[CSPubSubHandler] Message requires escalation to CS")
			// This will be handled by AI service to ask user confirmation
			return
		}

		// If message is from campaign - don't create case, just log
		if classification.IsCampaign {
			logger.Infof("[CSPubSubHandler] Message is from campaign, skipping CS case creation")
			return
		}

		// If message is part of payment flow - route to payment service
		if classification.IsPaymentFlow {
			logger.Infof("[CSPubSubHandler] Message is part of payment flow, routing to payment service")
			// TODO: Route to payment service
			return
		}

		// Otherwise, let AI handle it normally
		logger.Infof("[CSPubSubHandler] Message will be handled by AI")
	}
}

// routeMessageToCase routes user message to active CS case
func (h *CSPubSubHandlerImpl) routeMessageToCase(classification dtos.MessageClassification) {
	logger.Infof("[CSPubSubHandler] Saving user message to CS case: %s", classification.ActiveCaseID)

	// Save user message to CS case
	err := h.caseService.SaveUserMessage(
		classification.ActiveCaseID,
		classification.UserPhone,
		classification.MessageContent,
		classification.MessageType,
	)

	if err != nil {
		logger.Errorf("[CSPubSubHandler] Failed to save user message to case: %v", err)
		return
	}

	logger.Infof("[CSPubSubHandler] User message saved to CS case successfully")

	// Publish message received event
	event := &dtos.CSCaseMessageReceivedEvent{
		MessageID:      classification.MessageID,
		CaseID:         classification.ActiveCaseID,
		ClientId:       classification.ClientId,
		UserPhone:      classification.UserPhone,
		UserName:       classification.UserName,
		MessageContent: classification.MessageContent,
		MessageType:    classification.MessageType,
		ReceivedAt:     time.Now(),
	}
	h.PublishMessageReceived(event)
}

// handleCSCaseMessageReceived handles user messages received for active CS cases
func (h *CSPubSubHandlerImpl) handleCSCaseMessageReceived() pubsub.PubSubEventHandler {
	logger.Infof("[CSPubSubHandler] Subscribing to topic: %s", dtos.TopicCSCaseMessageReceived)
	return func(topic, msg string) {
		logger.Infof("[CSPubSubHandler handleCSCaseMessageReceived] Received user message: %s", msg)

		var event dtos.CSCaseMessageReceivedEvent
		if err := json.Unmarshal([]byte(msg), &event); err != nil {
			logger.Errorf("[CSPubSubHandler handleCSCaseMessageReceived] Failed to unmarshal CSCaseMessageReceivedEvent: %v", err)
			return
		}

		logger.Infof("[CSPubSubHandler handleCSCaseMessageReceived] Processing message for case %s from user %s", event.CaseNumber, event.UserPhone)

		// Save user message to customer_service_messages table
		err := h.caseService.SaveUserMessage(
			event.CaseID,
			event.UserPhone,
			event.MessageContent,
			event.MessageType,
		)

		if err != nil {
			logger.Errorf("[CSPubSubHandler handleCSCaseMessageReceived] Failed to save user message: %v", err)
			return
		}

		logger.Infof("[CSPubSubHandler handleCSCaseMessageReceived] User message saved successfully to case %s (Case #%s)",
			event.CaseID, event.CaseNumber)
	}
}

// handleCaseClosed handles when CS case is closed - triggers billing
func (h *CSPubSubHandlerImpl) handleCaseClosed() pubsub.PubSubEventHandler {
	logger.Infof("[CSPubSubHandler] Subscribing to topic: %s", dtos.TopicCSCaseClosed)
	return func(topic, msg string) {
		logger.Infof("[CSPubSubHandler] Received case closed event: %s", msg)

		var event dtos.CSCaseClosedEvent
		if err := json.Unmarshal([]byte(msg), &event); err != nil {
			logger.Errorf("[CSPubSubHandler] Failed to unmarshal CSCaseClosedEvent: %v", err)
			return
		}

		// Log billing information
		logger.Infof("[CSPubSubHandler] Case %s closed - Billing: Rp %.2f, Reference: %s",
			event.CaseNumber, event.BillingCost, event.BillingReference)

		// TODO: Additional post-close actions (notifications, analytics, etc)
	}
}

// handleSLABreached handles when SLA is breached
func (h *CSPubSubHandlerImpl) handleSLABreached() pubsub.PubSubEventHandler {
	logger.Infof("[CSPubSubHandler] Subscribing to topic: %s", dtos.TopicCSCaseSLABreached)
	return func(topic, msg string) {
		logger.Infof("[CSPubSubHandler] Received SLA breach event: %s", msg)

		var event dtos.CSCaseSLABreachedEvent
		if err := json.Unmarshal([]byte(msg), &event); err != nil {
			logger.Errorf("[CSPubSubHandler] Failed to unmarshal CSCaseSLABreachedEvent: %v", err)
			return
		}

		logger.Warnf("[CSPubSubHandler] SLA BREACHED - Case: %s, Wait time: %d seconds", event.CaseNumber, event.WaitTime)

		// TODO: Send notifications to managers
		// TODO: Auto-escalate if needed
	}
}

// ============================================================================
// Message Classifier - CRITICAL untuk distinguish message types
// ============================================================================

type MessageClassifier interface {
	Classify(messageContent string, clientId string, userPhone string) (*dtos.MessageClassification, error)
}

type MessageClassifierImpl struct{}

func NewMessageClassifier() *MessageClassifierImpl {
	return &MessageClassifierImpl{}
}

// Classify classifies user message into categories
func (c *MessageClassifierImpl) Classify(messageContent string, clientId string, userPhone string) (*dtos.MessageClassification, error) {
	logger.Infof("[MessageClassifier] Classifying message from %s: %s", userPhone, messageContent)

	lowerMsg := strings.ToLower(strings.TrimSpace(messageContent))

	classification := &dtos.MessageClassification{
		ClientId:       clientId,
		UserPhone:      userPhone,
		MessageContent: messageContent,
		MessageType:    "text",
		ClassifiedAt:   time.Now(),
	}

	// Check if user has active CS case
	activeCase := customerServiceCases.GetRepo().GetActiveCase(clientId, userPhone)
	if activeCase != nil {
		classification.HasActiveCSCase = true
		classification.ActiveCaseID = activeCase.ID
		classification.ConversationStatus = activeCase.Status
	}

	// Check for greeting (initial question)
	greetingWords := []string{"halo", "hello", "hai", "hi", "pagi", "siang", "sore", "malam", "permisi", "assalamualaikum"}
	for _, word := range greetingWords {
		if strings.Contains(lowerMsg, word) {
			classification.IsGreeting = true
			classification.Classification = "initial_question"
			classification.Confidence = 0.9
			break
		}
	}

	// Check for gangguan (complaint/outage)
	gangguanKeywords := []string{
		"air mati", "tidak ada air", "air tidak mengalir", "air macet",
		"gangguan", "rusak", "bocor", "pipa bocor", "meter rusak",
		"air keruh", "air bau", "tekanan rendah", "air kecil",
	}
	for _, keyword := range gangguanKeywords {
		if strings.Contains(lowerMsg, keyword) {
			classification.Classification = "gangguan"
			classification.IsPDAMRelated = true
			classification.Confidence = 0.95
			classification.Keywords = append(classification.Keywords, keyword)
			break
		}
	}

	// Check for inquiry (tagihan, cek meter, info)
	inquiryKeywords := []string{
		"tagihan", "cek tagihan", "berapa tagihan", "info tagihan",
		"nomor pelanggan", "meter", "cek meter", "baca meter",
		"biaya", "tarif", "harga air",
	}
	for _, keyword := range inquiryKeywords {
		if strings.Contains(lowerMsg, keyword) {
			classification.Classification = "inquiry"
			classification.IsPDAMRelated = true
			classification.Confidence = 0.85
			classification.Keywords = append(classification.Keywords, keyword)
			break
		}
	}

	// Check for payment
	paymentKeywords := []string{"bayar", "pembayaran", "virtual account", "va", "bank", "transfer"}
	for _, keyword := range paymentKeywords {
		if strings.Contains(lowerMsg, keyword) {
			classification.Classification = "payment"
			classification.IsPaymentFlow = true
			classification.IsPDAMRelated = true
			classification.Confidence = 0.9
			classification.Keywords = append(classification.Keywords, keyword)
			break
		}
	}

	// Check for campaign keywords
	campaignKeywords := []string{"promo", "diskon", "penawaran", "campaign"}
	for _, keyword := range campaignKeywords {
		if strings.Contains(lowerMsg, keyword) {
			classification.IsCampaign = true
			break
		}
	}

	// Default classification
	if classification.Classification == "" {
		classification.Classification = "other"
		classification.Confidence = 0.5
	}

	// Determine if escalation needed (low confidence or complex issue)
	if classification.Confidence < 0.6 {
		classification.IsEscalation = true
		classification.DetectedIntent = "needs_human_support"
	}

	logger.Infof("[MessageClassifier] Classification result - Type: %s, Confidence: %.2f, IsEscalation: %v",
		classification.Classification, classification.Confidence, classification.IsEscalation)

	return classification, nil
}

// ============================================================================
// Singleton
// ============================================================================

var (
	csPubSubHandler     CSPubSubHandler
	csPubSubHandlerOnce sync.Once
	messageClassifier   MessageClassifier
	classifierOnce      sync.Once
)

func GetCSPubSubHandler(caseService services.CustomerServiceSvc) CSPubSubHandler {
	if csPubSubHandler == nil {
		csPubSubHandlerOnce.Do(func() {
			handler := NewCSPubSubHandler(caseService)
			handler.Init()
			csPubSubHandler = handler
		})
	}
	return csPubSubHandler
}

func GetMessageClassifier() MessageClassifier {
	if messageClassifier == nil {
		classifierOnce.Do(func() {
			messageClassifier = NewMessageClassifier()
		})
	}
	return messageClassifier
}
