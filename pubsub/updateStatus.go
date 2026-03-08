package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/config"
	"github.com/ariandi/gocom/pubsub"
	"github.com/oklog/ulid/v2"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/client"
	"gitlab.com/anti_metter/switching_common/sender"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/logger"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/pubsub/ai"
	"gitlab.com/bot3342545/il-dashboard/pubsub/cs"
	"gitlab.com/bot3342545/il-dashboard/pubsub/payment"
	"gitlab.com/bot3342545/il-dashboard/pubsub/waimage"
	"gitlab.com/bot3342545/il-dashboard/repositories/customerServiceCases"
	"gitlab.com/bot3342545/il-dashboard/repositories/messageConversation"
	"gitlab.com/bot3342545/il-dashboard/services"
	"golang.org/x/sync/singleflight"
)

var (
	insertFailedCount        int64
	insertConsumedCount      int64
	processedMessages        sync.Map           // Cache for deduplication of message IDs
	processedStatusUpdates   sync.Map           // Cache for deduplication of status update IDs
	messageProcessingGroup   singleflight.Group // Prevents duplicate processing of same message
	statusUpdateProcessGroup singleflight.Group // Prevents duplicate processing of same status update
)

type WhatsappWebHook interface {
}

type WhatsappWebHookImpl struct {
	messageRouter  *MessageRouter
	aiHandler      *ai.Handler
	paymentHandler *payment.Handler
	csHandler      *cs.Handler
	imageHandler   *waimage.Handler
}

func NewWhatsappWebHook() *WhatsappWebHookImpl {
	impl := &WhatsappWebHookImpl{
		messageRouter: NewMessageRouter(),
	}
	impl.aiHandler = ai.NewHandler(impl.getClientFromPhoneNumber)
	impl.paymentHandler = payment.NewHandler()
	impl.csHandler = cs.NewHandler()
	impl.imageHandler = waimage.NewHandler(impl.messageRouter, impl.getClientFromPhoneNumber)
	return impl
}

func (o *WhatsappWebHookImpl) Init() {
	workerNo := config.GetInt(constans.WaWebHookWorkerNum, 1)

	for i := 1; i <= workerNo; i++ {
		pubsub.Get().Subscribe(constans.WaWebHookWorkerSubName, o.handleActionWaWebHook(i))

		// priority high latter
		//pubsub.Get().Subscribe(constans.WaWebHookWorkerSubName, o.handleActionWaWebHook(i))
	}
}

func (o *WhatsappWebHookImpl) handleActionWaWebHook(workerNo int) pubsub.PubSubEventHandler {
	customLogger.InfoWithData("Worker started subscribing", map[string]interface{}{
		"component":  "handleActionWaWebHook",
		"worker_id":  workerNo,
		"topic_name": constans.WaWebHookWorkerSubName,
	})
	return func(name, msg string) {
		o.processWebhookMessage(workerNo, msg)
	}
}

// processWebhookMessage handles the main webhook message processing with panic recovery
func (o *WhatsappWebHookImpl) processWebhookMessage(workerNo int, msg string) {
	// Generate unique request ID for tracing this entire request flow
	ctx := customLogger.WithRequestID(context.Background(), ulid.Make().String())

	consumed := atomic.AddInt64(&insertConsumedCount, 1)
	// Only log every 100 messages to reduce bloat
	if consumed%100 == 0 {
		customLogger.InfoWithContext(ctx, "Worker processing messages", map[string]interface{}{
			"component":     "handleActionWaWebHook",
			"worker_id":     workerNo,
			"message_count": consumed,
		})
	}

	defer o.recoverFromPanic(ctx)

	req, err := o.unmarshalWebhookRequest(ctx, msg)
	if err != nil {
		return
	}

	o.logWebhookRequest(ctx, req)
	o.processWebhookEntries(ctx, req)
}

// recoverFromPanic handles panic recovery and increments failure counter
func (o *WhatsappWebHookImpl) recoverFromPanic(ctx context.Context) {
	if r := recover(); r != nil {
		atomic.AddInt64(&insertFailedCount, 1)
		customLogger.ErrorWithContext(ctx, "Panic recovered in webhook handler", map[string]interface{}{
			"component":    "handleActionWaWebHook",
			"panic_value":  fmt.Sprintf("%+v", r),
			"failed_count": atomic.LoadInt64(&insertFailedCount),
		})
	}
}

// unmarshalWebhookRequest unmarshals the webhook message into a request object
func (o *WhatsappWebHookImpl) unmarshalWebhookRequest(ctx context.Context, msg string) (*dtos.WhatsAppWebhookRequest, error) {
	var req dtos.WhatsAppWebhookRequest
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		atomic.AddInt64(&insertFailedCount, 1)
		customLogger.ErrorWithContext(ctx, "Failed to unmarshal webhook request", map[string]interface{}{
			"component":      "handleActionWaWebHook",
			"error":          err.Error(),
			"message_length": len(msg),
			"failed_count":   atomic.LoadInt64(&insertFailedCount),
		})
		return nil, err
	}
	return &req, nil
}

// logWebhookRequest logs the webhook request for debugging
func (o *WhatsappWebHookImpl) logWebhookRequest(ctx context.Context, req *dtos.WhatsAppWebhookRequest) {
	// Only log at Debug level to reduce noise
	customLogger.DebugWithContext(ctx, "Webhook received", map[string]interface{}{
		"component":   "CallbackServiceImpl",
		"entry_count": len(req.Entry),
		"request":     logger.JsonToString(req),
	})
}

// processWebhookEntries processes all entries in the webhook request
func (o *WhatsappWebHookImpl) processWebhookEntries(ctx context.Context, req *dtos.WhatsAppWebhookRequest) {
	for _, entry := range req.Entry {
		for _, change := range entry.Changes {
			o.processWebhookChange(ctx, change)
		}
	}
}

// processWebhookChange routes different types of webhook changes to appropriate handlers
func (o *WhatsappWebHookImpl) processWebhookChange(ctx context.Context, change dtos.WhatsAppChangeData) {
	if len(change.Value.Messages) > 0 {
		o.handleIncomingMessages(ctx, change)
	}

	if len(change.Value.Statuses) > 0 {
		o.handleMessageStatusUpdates(ctx, change)
	}

	if change.Field == "message_template_status_update" {
		o.handleTemplateStatusChange(ctx, change.Value)
	}
}

// handleIncomingMessages processes incoming messages with race condition protection
func (o *WhatsappWebHookImpl) handleIncomingMessages(ctx context.Context, change dtos.WhatsAppChangeData) {
	contactName := o.extractContactName(change.Value.Contacts)

	for _, message := range change.Value.Messages {
		// Quick check first (fast path) - check if already fully processed
		if o.isMessageAlreadyProcessed(ctx, message.ID) {
			continue
		}

		// Use singleflight to ensure only ONE goroutine processes this message
		// If multiple goroutines try to process the same message.ID simultaneously,
		// only the first one will execute, others will wait and receive the same result
		messageID := message.ID
		_, err, shared := messageProcessingGroup.Do(messageID, func() (interface{}, error) {
			// Double-check inside singleflight to handle the case where message
			// was processed between the first check and entering singleflight
			if o.isMessageAlreadyProcessed(ctx, messageID) {
				return nil, nil
			}

			// Process the message
			o.processIncomingMessage(ctx, message, contactName, change.Value.Metadata)

			// Mark as processed AFTER successful processing
			processedMessages.Store(messageID, true)

			customLogger.DebugWithContext(ctx, "Message processed successfully", map[string]interface{}{
				"component":  "handleIncomingMessages",
				"message_id": messageID,
			})

			return nil, nil
		})

		// Log if this was a shared call (indicates duplicate was prevented)
		if shared {
			customLogger.InfoWithContext(ctx, "Duplicate message processing prevented by singleflight", map[string]interface{}{
				"component":  "handleIncomingMessages",
				"message_id": messageID,
			})
		}

		if err != nil {
			customLogger.ErrorWithContext(ctx, "Error processing message", map[string]interface{}{
				"component":  "handleIncomingMessages",
				"message_id": messageID,
				"error":      err.Error(),
			})
		}
	}
}

// isMessageAlreadyProcessed checks if a message has already been fully processed (deduplication)
func (o *WhatsappWebHookImpl) isMessageAlreadyProcessed(ctx context.Context, messageID string) bool {
	if _, exists := processedMessages.Load(messageID); exists {
		customLogger.DebugWithContext(ctx, "Message already processed, skipping", map[string]interface{}{
			"component":  "handleActionWaWebHook",
			"message_id": messageID,
		})
		return true
	}
	return false
}

// extractContactName extracts contact name from contacts array
func (o *WhatsappWebHookImpl) extractContactName(contacts []dtos.WhatsAppContact) string {
	if len(contacts) > 0 {
		return contacts[0].Profile.Name
	}
	return ""
}

// processIncomingMessage processes a single incoming message
func (o *WhatsappWebHookImpl) processIncomingMessage(ctx context.Context, message dtos.WhatsAppMessage, contactName string, metadata dtos.WhatsAppMetadata) {
	// Single log per message - key info only
	customLogger.InfoWithContext(ctx, "Processing message", map[string]interface{}{
		"component":   "handleActionWaWebHook",
		"from":        message.From,
		"type":        message.Type,
		"business_ph": metadata.DisplayPhoneNumber,
	})

	messageContent := o.extractMessageContent(message)
	if messageContent == "" {
		customLogger.WarnWithContext(ctx, "Empty message content", map[string]interface{}{
			"component": "handleActionWaWebHook",
			"type":      message.Type,
			"msg_id":    message.ID,
		})
		return
	}

	// Track user-initiated conversation window
	sender := o.getClientFromPhoneNumber(metadata.DisplayPhoneNumber)

	client := o.getClientById(sender.ClientId)
	if client != nil && !client.IsAi {
		customLogger.WarnWithContext(ctx, "Client is not using AI Service", map[string]interface{}{
			"component":      "handleActionWaWebHook",
			"client_id":      sender.ClientId,
			"user_phone":     message.From,
			"business_phone": metadata.DisplayPhoneNumber,
		})
		return
	}

	o.trackUserInitiatedConversation(sender, message.From)

	// Save message
	sessionID := ulid.Make().String()
	_ = o.saveIncomingMessage(message, messageContent, contactName, sessionID, metadata)

	// Route message based on type
	o.routeMessageByType(message, contactName, sessionID, metadata)
}

// trackUserInitiatedConversation tracks user-initiated conversation for free business replies
func (o *WhatsappWebHookImpl) trackUserInitiatedConversation(client *sender.Sender, userPhone string) {
	if client == nil {
		return
	}

	conversationKey := fmt.Sprintf("wa_conversation:%s:%s", client.ClientId, userPhone)
	conversationWindow := 24 * time.Hour
	userInitiatedValue := fmt.Sprintf("user_initiated:%s", time.Now().Format(time.RFC3339))

	if err := gocom.KeyVal().Set(conversationKey, userInitiatedValue, conversationWindow); err != nil {
		customLogger.ErrorWithData("Failed to set user conversation window", map[string]interface{}{
			"component": "handleActionWaWebHook",
			"error":     err.Error(),
			"phone":     userPhone,
			"client_id": client.ClientId,
		})
	}
	// Success case: no log needed, reduces bloat
}

// routeMessageByType routes messages to appropriate handlers based on message type
func (o *WhatsappWebHookImpl) routeMessageByType(message dtos.WhatsAppMessage, contactName, sessionID string, metadata dtos.WhatsAppMetadata) {
	switch message.Type {
	case "interactive":
		o.handleInteractiveMessage(message, contactName, sessionID, metadata)
	case "image":
		o.handleImageMessage(message, contactName, sessionID, metadata)
	case "text":
		o.handleTextMessage(message, contactName, metadata)
	}
}

// handleInteractiveMessage handles interactive button/list replies
func (o *WhatsappWebHookImpl) handleInteractiveMessage(message dtos.WhatsAppMessage, contactName, sessionID string, metadata dtos.WhatsAppMetadata) {
	if message.Interactive == nil {
		return
	}

	if message.Interactive.ButtonReply != nil {
		o.handleButtonReply(message, contactName, sessionID, metadata)
	}

	if message.Interactive.ListReply != nil {
		o.handleListReply(message, sessionID, metadata)
	}
}

// handleButtonReply processes button reply interactions
func (o *WhatsappWebHookImpl) handleButtonReply(message dtos.WhatsAppMessage, contactName, sessionID string, metadata dtos.WhatsAppMetadata) {
	buttonID := message.Interactive.ButtonReply.ID
	customLogger.InfoWithData("Interactive button clicked", map[string]interface{}{
		"component":    "handleActionWaWebHook",
		"button_id":    buttonID,
		"button_title": message.Interactive.ButtonReply.Title,
		"from_number":  message.From,
		"session_id":   sessionID,
	})

	client := o.getClientFromPhoneNumber(metadata.DisplayPhoneNumber)
	if client == nil {
		return
	}

	// Route to appropriate button handler
	switch {
	case strings.HasPrefix(buttonID, "pay_"):
		go o.paymentHandler.HandlePaymentButtonClick(sessionID, message.From, buttonID, client.ClientId)
	case strings.HasPrefix(buttonID, "copy_va_"):
		go o.paymentHandler.HandleCopyVAButtonClick(sessionID, message.From, buttonID, client.ClientId)
	case strings.HasPrefix(buttonID, "ocr_confirm_"):
		customLogger.InfoWithData("OCR result confirmed by user", map[string]interface{}{
			"component":  "handleActionWaWebHook",
			"user_phone": message.From,
			"button_id":  buttonID,
		})
		go o.imageHandler.HandleOCRConfirmation(message.From, buttonID, metadata.DisplayPhoneNumber)
	case strings.HasPrefix(buttonID, "ocr_reject_"):
		customLogger.InfoWithData("OCR result rejected by user", map[string]interface{}{
			"component":  "handleActionWaWebHook",
			"user_phone": message.From,
			"button_id":  buttonID,
		})
		go o.imageHandler.HandleOCRRejection(message.From)
	case buttonID == "btn_escalate_cs":
		customLogger.InfoWithData("CS escalation requested", map[string]interface{}{
			"component":    "handleActionWaWebHook",
			"user_phone":   message.From,
			"contact_name": contactName,
			"client_id":    client.ClientId,
			"session_id":   sessionID,
		})
		go o.csHandler.HandleCSEscalationButtonClick(sessionID, message.From, metadata.DisplayPhoneNumber, contactName, message.Timestamp, message.ID, client.ClientId)
	}
}

// handleListReply processes list reply interactions
func (o *WhatsappWebHookImpl) handleListReply(message dtos.WhatsAppMessage, sessionID string, metadata dtos.WhatsAppMetadata) {
	listID := message.Interactive.ListReply.ID
	customLogger.InfoWithData("List item selected", map[string]interface{}{
		"component":   "handleActionWaWebHook",
		"list_id":     listID,
		"list_title":  message.Interactive.ListReply.Title,
		"from_number": message.From,
		"session_id":  sessionID,
	})

	if strings.HasPrefix(listID, "bank_") {
		client := o.getClientFromPhoneNumber(metadata.DisplayPhoneNumber)
		if client != nil {
			go o.paymentHandler.HandleBankSelection(sessionID, message.From, listID, client.ClientId)
		}
	}
}

// handleImageMessage processes image messages for OCR
func (o *WhatsappWebHookImpl) handleImageMessage(message dtos.WhatsAppMessage, contactName, sessionID string, metadata dtos.WhatsAppMetadata) {
	if message.Image == nil {
		return
	}

	client := o.getClientFromPhoneNumber(metadata.DisplayPhoneNumber)
	if client == nil {
		return
	}

	customLogger.InfoWithData("Image message received", map[string]interface{}{
		"component":   "handleActionWaWebHook",
		"from_number": message.From,
		"image_id":    message.Image.ID,
		"mime_type":   message.Image.MimeType,
		"has_caption": message.Image.Caption != "",
		"client_id":   client.ClientId,
	})

	imageEvent := dtos.WAImageMessageEvent{
		SessionID:         sessionID,
		FromNumber:        message.From,
		ToNumber:          metadata.DisplayPhoneNumber,
		ImageID:           message.Image.ID,
		ImageMimeType:     message.Image.MimeType,
		ImageCaption:      message.Image.Caption,
		ClientID:          client.ClientId,
		ContactName:       contactName,
		Timestamp:         message.Timestamp,
		WhatsAppMessageID: message.ID,
	}

	o.publishImageEvent(imageEvent)
}

// publishImageEvent publishes image event to processing queue
func (o *WhatsappWebHookImpl) publishImageEvent(imageEvent dtos.WAImageMessageEvent) {
	eventJSON, err := json.Marshal(imageEvent)
	if err != nil {
		customLogger.ErrorWithData("Failed to marshal image event", map[string]interface{}{
			"component":   "handleActionWaWebHook",
			"error":       err.Error(),
			"image_id":    imageEvent.ImageID,
			"from_number": imageEvent.FromNumber,
		})
		return
	}

	if err := gocom.PubSub().Publish(dtos.TopicWAMessageImage, string(eventJSON)); err != nil {
		customLogger.ErrorWithData("Failed to publish image event", map[string]interface{}{
			"component": "handleActionWaWebHook",
			"error":     err.Error(),
			"topic":     dtos.TopicWAMessageImage,
			"image_id":  imageEvent.ImageID,
		})
		return
	}

	customLogger.InfoWithData("Image event published for processing", map[string]interface{}{
		"component":   "handleActionWaWebHook",
		"topic":       dtos.TopicWAMessageImage,
		"image_id":    imageEvent.ImageID,
		"from_number": imageEvent.FromNumber,
		"client_id":   imageEvent.ClientID,
	})
}

// handleTextMessage processes text messages and routes to AI/CS
func (o *WhatsappWebHookImpl) handleTextMessage(message dtos.WhatsAppMessage, contactName string, metadata dtos.WhatsAppMetadata) {
	if message.Text == nil {
		return
	}

	// Check if user is confirming conversation close
	if o.handleConversationClose(message) {
		return
	}

	// Route message through MessageRouter
	o.routeTextMessage(message, contactName, metadata)
}

// handleConversationClose checks and handles conversation close confirmation
func (o *WhatsappWebHookImpl) handleConversationClose(message dtos.WhatsAppMessage) bool {
	if !o.aiHandler.IsClosingConfirmation(message.Text.Body) {
		return false
	}

	lastConv := messageConversation.GetRepo().GetConversationHistory(message.From, 1)
	if len(lastConv) > 0 && lastConv[0].ConversationStatus == "pending_close" {
		customLogger.InfoWithData("User confirmed conversation close", map[string]interface{}{
			"component":       "handleActionWaWebHook",
			"user_phone":      message.From,
			"conversation_id": lastConv[0].ID,
		})
		if err := messageConversation.GetRepo().CloseConversation(message.From); err != nil {
			customLogger.ErrorWithData("Failed to close conversation", map[string]interface{}{
				"component":  "handleActionWaWebHook",
				"error":      err.Error(),
				"user_phone": message.From,
			})
		}
		return true
	}

	return false
}

// routeTextMessage routes text messages through MessageRouter
func (o *WhatsappWebHookImpl) routeTextMessage(message dtos.WhatsAppMessage, contactName string, metadata dtos.WhatsAppMetadata) {
	isGreeting := o.aiHandler.ShouldTriggerAI(message.Text.Body)
	isConversationActive := messageConversation.GetRepo().IsConversationActive(message.From)
	isPDAMRelated := o.aiHandler.IsPDAMRelatedMessage(message.Text.Body)

	if !isGreeting && !isConversationActive {
		customLogger.InfoWithData("Message ignored - no active conversation", map[string]interface{}{
			"component":   "handleActionWaWebHook",
			"from_number": message.From,
			"is_greeting": isGreeting,
			"is_active":   isConversationActive,
		})
		return
	}

	customLogger.InfoWithData("Message routing decision", map[string]interface{}{
		"component":       "handleActionWaWebHook",
		"from_number":     message.From,
		"is_greeting":     isGreeting,
		"is_active":       isConversationActive,
		"is_pdam_related": isPDAMRelated,
		"will_route":      true,
	})

	go o.messageRouter.RouteMessage(
		message.From,
		metadata.DisplayPhoneNumber,
		message.Text.Body,
		message.Type,
		contactName,
		message.Timestamp,
		message.ID,
	)
}

// handleMessageStatusUpdates processes message status updates (delivered, read, failed) with race condition protection
func (o *WhatsappWebHookImpl) handleMessageStatusUpdates(ctx context.Context, request dtos.WhatsAppChangeData) {
	for _, status := range request.Value.Statuses {
		// Create unique key for this status update: messageID + status + recipientID
		// This ensures we process each unique status change only once
		statusKey := fmt.Sprintf("%s:%s:%s", status.ID, status.Status, status.RecipientID)

		// Quick check first (fast path) - check if already fully processed
		if o.isStatusUpdateAlreadyProcessed(ctx, statusKey) {
			continue
		}

		// Use singleflight to ensure only ONE goroutine processes this status update
		// If multiple goroutines try to process the same status simultaneously,
		// only the first one will execute, others will wait and receive the same result
		_, err, shared := statusUpdateProcessGroup.Do(statusKey, func() (interface{}, error) {
			// Double-check inside singleflight to handle the case where status
			// was processed between the first check and entering singleflight
			if o.isStatusUpdateAlreadyProcessed(ctx, statusKey) {
				return nil, nil
			}

			customLogger.InfoWithContext(ctx, "Message status updated", map[string]interface{}{
				"component":    "handleActionWaWebHook",
				"message_id":   status.ID,
				"new_status":   status.Status,
				"recipient_id": status.RecipientID,
				"has_errors":   len(status.Errors) > 0,
			})

			// Process the status update
			o.updateMessageStatus(ctx, request.Value.Metadata, status)

			// Mark as processed AFTER successful processing
			processedStatusUpdates.Store(statusKey, true)

			customLogger.DebugWithContext(ctx, "Status update processed successfully", map[string]interface{}{
				"component":  "handleMessageStatusUpdates",
				"status_key": statusKey,
			})

			return nil, nil
		})

		// Log if this was a shared call (indicates duplicate was prevented)
		if shared {
			customLogger.InfoWithContext(ctx, "Duplicate status update prevented by singleflight", map[string]interface{}{
				"component":  "handleMessageStatusUpdates",
				"status_key": statusKey,
			})
		}

		if err != nil {
			customLogger.ErrorWithContext(ctx, "Error processing status update", map[string]interface{}{
				"component":  "handleMessageStatusUpdates",
				"status_key": statusKey,
				"error":      err.Error(),
			})
		}
	}
}

// isStatusUpdateAlreadyProcessed checks if a status update has already been fully processed (deduplication)
func (o *WhatsappWebHookImpl) isStatusUpdateAlreadyProcessed(ctx context.Context, statusKey string) bool {
	if _, exists := processedStatusUpdates.Load(statusKey); exists {
		customLogger.DebugWithContext(ctx, "Status update already processed, skipping", map[string]interface{}{
			"component":  "handleMessageStatusUpdates",
			"status_key": statusKey,
		})
		return true
	}
	return false
}

// extractErrorMessage extracts error message from status errors
func (o *WhatsappWebHookImpl) extractErrorMessage(errors []dtos.WhatsAppError) string {
	if len(errors) == 0 {
		return ""
	}

	var errorMessage string
	for _, err := range errors {
		customLogger.ErrorWithData("WhatsApp message error received", map[string]interface{}{
			"component":     "handleActionWaWebHook",
			"error_code":    err.Code,
			"error_title":   err.Title,
			"error_message": err.Message,
		})
		errorMessage = fmt.Sprintf("Code: %d, Title: %s, Message: %s", err.Code, err.Title, err.Message)
	}

	return errorMessage
}

// handleTemplateStatusChange processes template status updates
func (o *WhatsappWebHookImpl) handleTemplateStatusChange(ctx context.Context, value dtos.WhatsAppValueData) {
	customLogger.InfoWithContext(ctx, "Template status update detected", map[string]interface{}{
		"component": "handleActionWaWebHook",
	})

	if value.Event == "" || value.MessageTemplateName == "" {
		customLogger.WarnWithData("Template status update missing required fields", map[string]interface{}{
			"component":     "handleActionWaWebHook",
			"event":         value.Event,
			"template_name": value.MessageTemplateName,
		})
		return
	}

	templateStatus := &dtos.WhatsAppTemplateStatusEvent{
		MessageTemplateID:       value.MessageTemplateID,
		MessageTemplateName:     value.MessageTemplateName,
		MessageTemplateLanguage: value.MessageTemplateLanguage,
		MessageTemplateCategory: value.MessageTemplateCategory,
		Event:                   value.Event,
		Reason:                  value.Reason,
	}

	customLogger.InfoWithData("Template status details", map[string]interface{}{
		"component":     "handleActionWaWebHook",
		"template_id":   templateStatus.MessageTemplateID,
		"template_name": templateStatus.MessageTemplateName,
		"event":         templateStatus.Event,
		"category":      templateStatus.MessageTemplateCategory,
		"language":      templateStatus.MessageTemplateLanguage,
		"has_reason":    templateStatus.Reason != "",
	})

	o.handleTemplateStatusUpdate(templateStatus)
}

// extractMessageContent extracts text content from different message types
func (o *WhatsappWebHookImpl) extractMessageContent(message dtos.WhatsAppMessage) string {
	switch message.Type {
	case "text":
		if message.Text != nil {
			return message.Text.Body
		}
	case "image":
		if message.Image != nil {
			caption := message.Image.Caption
			if caption == "" {
				caption = "[Image]"
			}
			return fmt.Sprintf("[Image ID: %s] %s", message.Image.ID, caption)
		}
	case "audio":
		if message.Audio != nil {
			return fmt.Sprintf("[Audio ID: %s]", message.Audio.ID)
		}
	case "video":
		if message.Video != nil {
			caption := message.Video.Caption
			if caption == "" {
				caption = "[Video]"
			}
			return fmt.Sprintf("[Video ID: %s] %s", message.Video.ID, caption)
		}
	case "document":
		if message.Document != nil {
			filename := message.Document.Filename
			if filename == "" {
				filename = "document"
			}
			return fmt.Sprintf("[Document: %s, ID: %s]", filename, message.Document.ID)
		}
	case "location":
		if message.Location != nil {
			return fmt.Sprintf("[Location: lat=%f, lng=%f, name=%s]",
				message.Location.Latitude, message.Location.Longitude, message.Location.Name)
		}
	case "button":
		if message.Button != nil {
			return fmt.Sprintf("[Button: %s, payload=%s]", message.Button.Text, message.Button.Payload)
		}
	case "interactive":
		if message.Interactive != nil {
			if message.Interactive.ButtonReply != nil {
				return fmt.Sprintf("[Button Reply: %s]", message.Interactive.ButtonReply.Title)
			}
			if message.Interactive.ListReply != nil {
				return fmt.Sprintf("[List Reply: %s]", message.Interactive.ListReply.Title)
			}
		}
	}
	return ""
}

// getClientIdFromPhoneNumber looks up client_id from senders table by phone number identifier
func (o *WhatsappWebHookImpl) getClientFromPhoneNumber(phoneNumber string) *sender.Sender {
	if phoneNumber == "" {
		customLogger.WarnWithData("Empty phone number provided for client lookup", map[string]interface{}{
			"component": "getClientIdFromPhoneNumber",
		})
		return nil // Fallback to default
	}

	// Lookup sender by phone number identifier
	senderData := sender.GetRepo().GetByIdentifier(phoneNumber)
	if senderData == nil {
		customLogger.WarnWithData("No sender found for phone number", map[string]interface{}{
			"component":    "getClientIdFromPhoneNumber",
			"phone_number": phoneNumber,
		})
		return nil // Fallback to default
	}

	// Success case: no log needed
	return senderData
}

// saveIncomingMessage saves message to both message_logs and message_conversations
func (o *WhatsappWebHookImpl) saveIncomingMessage(
	message dtos.WhatsAppMessage,
	content string,
	contactName string,
	sessionID string,
	metadata dtos.WhatsAppMetadata,
) string {
	// No entry log - reduces bloat, parent already logged

	// Get client_id from business phone number that received the message
	client := o.getClientFromPhoneNumber(metadata.DisplayPhoneNumber)

	// Save to message_conversations
	conv := &messageConversation.MessageConversation{
		ClientId:   client.ClientId,
		SessionID:  sessionID,
		FromNumber: message.From,
		ToNumber:   metadata.DisplayPhoneNumber, // Business phone number that received the message
		Role:       "user",
		Message:    content,
		Status:     "received",
	}

	err := messageConversation.GetRepo().Create(conv).Error
	if err != nil {
		customLogger.ErrorWithData("Failed to save conversation", map[string]interface{}{
			"component": "saveIncomingMessage",
			"error":     err.Error(),
			"msg_id":    message.ID,
			"from":      message.From,
		})
	}
	// Success case: no log needed

	// Save to message_logs (INBOUND message)
	now := time.Now()
	messageLog := &dtos.MessageLog{
		ID:             ulid.Make().String(),
		ClientId:       client.ClientId,
		SenderId:       message.From, // User's WhatsApp number who sent the message
		Type:           "whatsapp",
		Direction:      "inbound",
		RecipientType:  "phone",
		RecipientValue: metadata.DisplayPhoneNumber, // Business phone that received the message
		MessageId:      message.ID,
		SessionId:      sessionID,
		MessageContent: content,
		Status:         "received",
		SentAt:         now.Format(time.RFC3339), // When message was received
		DeliveredAt:    now.Format(time.RFC3339),
		Cost:           0,
		CreatedAt:      now.Format(time.RFC3339),
	}

	// Save message log (best effort, don't fail if this fails)
	_, logErr := services.GetMessageLogsService().Create(messageLog, auth.AuthInfo{ClientId: client.ClientId})
	if logErr != nil {
		customLogger.ErrorWithData("Failed to save message log", map[string]interface{}{
			"component": "saveIncomingMessage",
			"error":     logErr.Message,
			"msg_id":    message.ID,
			"client_id": client.ClientId,
		})
		// Continue anyway, already saved to conversations
	}
	// Success case: no log needed
	return sessionID
}

// triggerAIConversation delegates AI flow to AI handler.
func (o *WhatsappWebHookImpl) triggerAIConversation(sessionID, fromNumber, userMessage, businessPhoneNumber string) {
	o.aiHandler.TriggerAIConversation(sessionID, fromNumber, userMessage, businessPhoneNumber)
}

// sendRejectionMessage delegates rejection flow to AI handler.
func (o *WhatsappWebHookImpl) sendRejectionMessage(sessionID, fromNumber, userMessage, businessPhoneNumber string) {
	o.aiHandler.SendRejectionMessage(sessionID, fromNumber, userMessage, businessPhoneNumber)
}

// updateMessageStatus updates message status in message_logs and message_conversations
func (o *WhatsappWebHookImpl) updateMessageStatus(ctx context.Context, metadata dtos.WhatsAppMetadata, status dtos.WhatsAppStatus) {

	errorMessage := o.extractErrorMessage(status.Errors)

	// Only log errors or failures
	if status.Status == "failed" || errorMessage != "" {
		customLogger.WarnWithData("Message status update", map[string]interface{}{
			"component": "updateMessageStatus",
			"msg_id":    status.ID,
			"status":    status,
			"has_error": true,
		})

		clientSender := o.getClientFromPhoneNumber(metadata.DisplayPhoneNumber)
		if clientSender == nil {
			customLogger.WarnWithData("Client not found", map[string]interface{}{
				"component": "updateMessageStatus",
				"msg_id":    status.ID,
				"status":    status,
				"has_error": true,
			})
			return
		}

		messageLog, err := services.GetMessageLogsService().GetByMessageId(status.ID)
		if err != nil {
			customLogger.WarnWithData("Message log not found", map[string]interface{}{
				"component": "updateMessageStatus",
				"msg_id":    status.ID,
				"status":    status,
				"has_error": true,
			})
			return
		}

		if messageLog == nil {
			customLogger.WarnWithData("Message log not found", map[string]interface{}{
				"component": "updateMessageStatus",
				"msg_id":    status.ID,
				"status":    status,
				"has_error": true,
			})
			return
		}

		go services.GetBillingService().CreateTransaction(dtos.BillingTransactionReq{
			Type:          constans.BillingTransactionTypeRefund,
			ClientId:      clientSender.ClientId,
			Amount:        messageLog.Cost,
			ReferenceId:   messageLog.ID,
			ReferenceType: "message",
			Description:   fmt.Sprintf("Revert balance for failed message from %s", status.RecipientID),
		}, auth.AuthInfo{ClientId: clientSender.ClientId})

	}
	// Success updates: no log needed (reduces bloat)

	// Parse timestamp for status updates
	now := time.Now()
	var deliveredAt, readAt, failedAt *time.Time

	switch status.Status {
	case "delivered":
		deliveredAt = &now
	case "read":
		readAt = &now
	case "failed":
		failedAt = &now
	}

	// Update message_logs
	err := services.GetMessageLogsService().UpdateDeliveryStatus(
		status.ID,
		status.Status,
		deliveredAt,
		readAt,
		failedAt,
		errorMessage,
	)
	if err != nil {
		customLogger.WarnWithData("Failed to update message_logs", map[string]interface{}{
			"component":  "updateMessageStatus",
			"error":      err.Message,
			"message_id": status.ID,
			"status":     status,
		})
	} else {
		customLogger.DebugWithData("Updated message_logs", map[string]interface{}{
			"component":  "updateMessageStatus",
			"message_id": status.ID,
			"status":     status,
		})
	}

	// Update campaign recipient status if this message is from a campaign (async)
	go o.updateCampaignRecipientStatus(status.ID, status.Status, deliveredAt, readAt, failedAt)

	// Update message_conversations status if exists
	go o.updateMessageConversationStatus(status.ID, status.Status)

	// Update customer_service_messages status if exists
	go o.updateCSMessageStatus(status.ID, status.Status, deliveredAt, readAt)
}

// updateCampaignRecipientStatus updates campaign recipient status based on message delivery status
func (o *WhatsappWebHookImpl) updateCampaignRecipientStatus(messageID, status string, deliveredAt, readAt, failedAt *time.Time) {
	customLogger.DebugWithData("Checking campaign recipient", map[string]interface{}{
		"component":  "updateCampaignRecipientStatus",
		"message_id": messageID,
		"status":     status,
	})

	// Get message log to find if it's from a campaign
	mdl, err := services.GetMessageLogsService().GetByMessageId(messageID)
	if err != nil {
		customLogger.DebugWithData("Message log not found", map[string]interface{}{
			"component":  "updateCampaignRecipientStatus",
			"message_id": messageID,
		})
		return
	}

	// Check if this message is from a campaign
	if mdl.CampaignId == "" {
		customLogger.DebugWithData("Message not from campaign, skipping", map[string]interface{}{
			"component":  "updateCampaignRecipientStatus",
			"message_id": messageID,
		})
		return
	}

	customLogger.InfoWithData("Updating campaign recipient status", map[string]interface{}{
		"component":   "updateCampaignRecipientStatus",
		"campaign_id": mdl.CampaignId,
		"message_id":  messageID,
		"status":      status,
	})

	// Find campaign recipient by message_id
	recipient, recipientErr := services.GetCampaignsService().GetRecipientByMessageId(messageID)
	if recipientErr != nil {
		customLogger.WarnWithData("Campaign recipient not found", map[string]interface{}{
			"component":   "updateCampaignRecipientStatus",
			"message_id":  messageID,
			"campaign_id": mdl.CampaignId,
			"error":       recipientErr.Message,
		})
		return
	}

	// Update recipient status based on message status
	updated := false
	switch status {
	case "delivered":
		if recipient.Status == "sent" || recipient.Status == "pending" {
			recipient.Status = "delivered"
			if deliveredAt != nil {
				recipient.DeliveredAt = deliveredAt.Format(time.RFC3339)
			}
			updated = true
		}
	case "read":
		recipient.Status = "read"
		if readAt != nil {
			recipient.ReadAt = readAt.Format(time.RFC3339)
		}
		if recipient.DeliveredAt == "" && readAt != nil {
			recipient.DeliveredAt = readAt.Format(time.RFC3339) // Set delivered time if not set
		}
		updated = true
	case "failed":
		recipient.Status = "failed"
		if failedAt != nil {
			recipient.FailedAt = failedAt.Format(time.RFC3339)
		}
		updated = true
	}

	if updated {
		updateErr := services.GetCampaignsService().UpdateRecipient(recipient)
		if updateErr != nil {
			customLogger.ErrorWithData("Failed to update campaign recipient", map[string]interface{}{
				"component":    "updateCampaignRecipientStatus",
				"error":        updateErr.Message,
				"recipient_id": recipient.ID,
				"campaign_id":  mdl.CampaignId,
				"status":       status,
			})
		} else {
			customLogger.InfoWithData("Campaign recipient updated", map[string]interface{}{
				"component":    "updateCampaignRecipientStatus",
				"recipient_id": recipient.ID,
				"campaign_id":  mdl.CampaignId,
				"new_status":   recipient.Status,
			})
		}
	}
}

// updateMessageConversationStatus updates message_conversations status based on WhatsApp delivery status
func (o *WhatsappWebHookImpl) updateMessageConversationStatus(messageID, status string) {
	customLogger.DebugWithData("Checking message conversation", map[string]interface{}{
		"component":  "updateMessageConversationStatus",
		"message_id": messageID,
		"status":     status,
	})

	// Get message log first to find the message_log_id
	mdl, err := services.GetMessageLogsService().GetByMessageId(messageID)
	if err != nil {
		customLogger.DebugWithData("Message log not found", map[string]interface{}{
			"component":  "updateMessageConversationStatus",
			"message_id": messageID,
		})
		return
	}

	// Get message conversation by message_log_id
	conv := messageConversation.GetRepo().GetByMessageLogId(mdl.ID)
	if conv == nil {
		customLogger.DebugWithData("Message conversation not found", map[string]interface{}{
			"component": "updateMessageConversationStatus",
			"log_id":    mdl.ID,
		})
		return
	}

	// Only update if status changed
	if conv.Status == status {
		customLogger.DebugWithData("Status already set, skipping", map[string]interface{}{
			"component":       "updateMessageConversationStatus",
			"conversation_id": conv.ID,
			"status":          status,
		})
		return
	}

	// Update status
	conv.Status = status
	conv.UpdatedAt = time.Now()

	if err := messageConversation.GetRepo().Update(conv); err != nil {
		customLogger.ErrorWithData("Failed to update conversation status", map[string]interface{}{
			"component":       "updateMessageConversationStatus",
			"error":           err.Error(),
			"conversation_id": conv.ID,
			"new_status":      status,
		})
		return
	}

	customLogger.InfoWithData("Conversation status updated", map[string]interface{}{
		"component":       "updateMessageConversationStatus",
		"conversation_id": conv.ID,
		"old_status":      conv.Status,
		"new_status":      status,
	})
}

// updateCSMessageStatus updates customer_service_messages status based on WhatsApp delivery status
func (o *WhatsappWebHookImpl) updateCSMessageStatus(messageID, status string, deliveredAt, readAt *time.Time) {
	customLogger.DebugWithData("Checking CS message", map[string]interface{}{
		"component":  "updateCSMessageStatus",
		"message_id": messageID,
		"status":     status,
	})

	// Get CS message directly by WhatsApp message ID
	csRepo := customerServiceCases.GetRepo()
	csMessage := csRepo.GetMessageByWhatsAppMessageId(messageID)

	if csMessage == nil {
		customLogger.DebugWithData("CS message not found", map[string]interface{}{
			"component":  "updateCSMessageStatus",
			"message_id": messageID,
		})
		return
	}

	// Only update if status changed
	if csMessage.Status == status {
		customLogger.DebugWithData("CS message status already set", map[string]interface{}{
			"component":     "updateCSMessageStatus",
			"cs_message_id": csMessage.ID,
			"status":        status,
		})
		return
	}

	// Update status and timestamps
	csMessage.Status = status
	if deliveredAt != nil {
		csMessage.DeliveredAt = deliveredAt
	}
	if readAt != nil {
		csMessage.ReadAt = readAt
	}

	if err := csRepo.UpdateMessage(csMessage); err != nil {
		customLogger.ErrorWithData("Failed to update CS message status", map[string]interface{}{
			"component":     "updateCSMessageStatus",
			"error":         err.Error(),
			"cs_message_id": csMessage.ID,
			"new_status":    status,
		})
		return
	}

	customLogger.InfoWithData("CS message status updated", map[string]interface{}{
		"component":     "updateCSMessageStatus",
		"cs_message_id": csMessage.ID,
		"new_status":    status,
		"delivered":     deliveredAt != nil,
		"read":          readAt != nil,
	})
}

// handleTemplateStatusUpdate handles template status updates from WhatsApp
func (o *WhatsappWebHookImpl) handleTemplateStatusUpdate(templateStatus *dtos.WhatsAppTemplateStatusEvent) {
	customLogger.InfoWithData("Template status update received", map[string]interface{}{
		"component":     "handleTemplateStatusUpdate",
		"template_name": templateStatus.MessageTemplateName,
		"event":         templateStatus.Event,
		"reason":        templateStatus.Reason,
		"template_id":   templateStatus.MessageTemplateID,
	})

	// Get WATemplateSvc to update template status
	waTemplateID := fmt.Sprintf("%d", templateStatus.MessageTemplateID)

	// Create provider auth for approval/rejection - HARUS PROVIDER
	providerAuth := auth.AuthInfo{
		ClientId: common.PROVIDER_ID,
	}

	// Get template by WATemplateId to get the database ID
	template, err := services.GetWATemplateSvc().GetByWaTemplateId(waTemplateID, providerAuth)
	if err != nil {
		customLogger.ErrorWithData("Template not found by WATemplateId", map[string]interface{}{
			"component":      "handleTemplateStatusUpdate",
			"wa_template_id": waTemplateID,
			"error":          err.Message,
		})
		return
	}

	customLogger.InfoWithData("Template found in database", map[string]interface{}{
		"component":      "handleTemplateStatusUpdate",
		"template_id":    template.ID,
		"template_name":  template.Name,
		"current_status": template.Status,
		"wa_template_id": waTemplateID,
	})

	switch strings.ToUpper(templateStatus.Event) {
	case "APPROVED":
		// Skip if already approved
		if template.Status == "APPROVED" {
			customLogger.InfoWithData("Template already approved, skipping", map[string]interface{}{
				"component":     "handleTemplateStatusUpdate",
				"template_name": template.Name,
				"template_id":   template.ID,
			})
			return
		}

		_, err := services.GetWATemplateSvc().Approve(
			template.ID, // Use database ID
			waTemplateID,
			providerAuth,
		)
		if err != nil {
			customLogger.ErrorWithData("Failed to approve template", map[string]interface{}{
				"component":     "handleTemplateStatusUpdate",
				"error":         err.Message,
				"template_id":   template.ID,
				"template_name": template.Name,
			})
		} else {
			customLogger.InfoWithData("Template approved successfully", map[string]interface{}{
				"component":     "handleTemplateStatusUpdate",
				"template_name": template.Name,
				"template_id":   template.ID,
			})
		}

	case "REJECTED":
		// Skip if already rejected
		if template.Status == "REJECTED" {
			customLogger.InfoWithData("Template already rejected, skipping", map[string]interface{}{
				"component":     "handleTemplateStatusUpdate",
				"template_name": template.Name,
				"template_id":   template.ID,
			})
			return
		}

		reason := templateStatus.Reason
		if reason == "" {
			reason = "Rejected by WhatsApp"
		}
		_, err := services.GetWATemplateSvc().Reject(
			template.ID, // Use database ID
			reason,
			providerAuth,
		)
		if err != nil {
			customLogger.ErrorWithData("Failed to reject template", map[string]interface{}{
				"component":     "handleTemplateStatusUpdate",
				"error":         err.Message,
				"template_id":   template.ID,
				"template_name": template.Name,
			})
		} else {
			customLogger.InfoWithData("Template rejected successfully", map[string]interface{}{
				"component":     "handleTemplateStatusUpdate",
				"template_name": template.Name,
				"template_id":   template.ID,
				"reason":        reason,
			})
		}

	default:
		customLogger.InfoWithData("Unhandled template event", map[string]interface{}{
			"component":     "handleTemplateStatusUpdate",
			"event":         templateStatus.Event,
			"template_name": templateStatus.MessageTemplateName,
		})
	}
}

func (o *WhatsappWebHookImpl) getClientById(clientId string) *client.Client {
	client := client.GetRepo().GetById(clientId)
	if client == nil {
		customLogger.WarnWithData("Client not found", map[string]interface{}{
			"component": "getClientById",
			"client_id": clientId,
		})
		return nil
	}
	return client
}

// handleBankSelection delegates bank selection flow to payment handler
func (o *WhatsappWebHookImpl) handleBankSelection(sessionID, fromNumber, listID, clientId string) {
	o.paymentHandler.HandleBankSelection(sessionID, fromNumber, listID, clientId)
}

//---------------------------

var waWebhook WhatsappWebHook
var waWebhookOnce sync.Once

func GetWhatsappWebhook() WhatsappWebHook {
	if waWebhook == nil {
		waWebhookOnce.Do(func() {
			tmp := NewWhatsappWebHook()
			tmp.Init()
			waWebhook = tmp
		})
	}
	return waWebhook
}

// handleOCRConfirmation delegates OCR confirmation to image handler
func (o *WhatsappWebHookImpl) handleOCRConfirmation(fromNumber, buttonID, businessPhone string) {
	o.imageHandler.HandleOCRConfirmation(fromNumber, buttonID, businessPhone)
}

// handleOCRRejection delegates OCR rejection to image handler
func (o *WhatsappWebHookImpl) handleOCRRejection(fromNumber, businessPhone string) {
	_ = businessPhone
	o.imageHandler.HandleOCRRejection(fromNumber)
}

// handleCSEscalationButtonClick delegates CS escalation to CS handler
func (o *WhatsappWebHookImpl) handleCSEscalationButtonClick(sessionID, fromNumber, businessPhone, contactName, timestamp, waMessageID, clientId string) {
	o.csHandler.HandleCSEscalationButtonClick(sessionID, fromNumber, businessPhone, contactName, timestamp, waMessageID, clientId)
}
