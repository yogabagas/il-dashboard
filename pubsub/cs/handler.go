package cs

import (
	"encoding/json"
	"time"

	"github.com/ariandi/gocom"
	csDtos "gitlab.com/bot3342545/il-dashboard/customer_service/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/repositories/messageConversation"
	"gitlab.com/bot3342545/il-dashboard/services"
)

// Handler manages CS escalation interactions from WhatsApp.
type Handler struct{}

// NewHandler constructs CS handler.
func NewHandler() *Handler {
	return &Handler{}
}

// HandleCSEscalationButtonClick handles CS escalation button click.
func (h *Handler) HandleCSEscalationButtonClick(sessionID, fromNumber, businessPhone, contactName, timestamp, waMessageID, clientId string) {
	customLogger.InfoWithData("Processing CS escalation button click", map[string]interface{}{
		"component":      "CSHandler",
		"function":       "HandleCSEscalationButtonClick",
		"from_number":    fromNumber,
		"client_id":      clientId,
		"session_id":     sessionID,
		"business_phone": businessPhone,
	})

	userConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         fromNumber,
		Role:               "user",
		Message:            "[User clicked: Hubungkan dengan CS]",
		Status:             "received",
		ConversationStatus: "escalated",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(userConv)

	event := csDtos.AIEscalationConfirmedEvent{
		SessionID:      sessionID,
		ClientId:       clientId,
		UserPhone:      fromNumber,
		UserName:       contactName,
		UserMessage:    "Hubungkan dengan CS",
		ConfirmMessage: "Hubungkan dengan CS",
		Category:       "umum",
		Metadata: map[string]interface{}{
			"reason":          "User clicked CS escalation button",
			"auto_escalate":   true,
			"escalation_type": "button_click",
		},
		ConfirmedAt: time.Now(),
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		customLogger.ErrorWithData("Failed to marshal CS escalation event", map[string]interface{}{
			"component":   "CSHandler",
			"function":    "HandleCSEscalationButtonClick",
			"error":       err.Error(),
			"from_number": fromNumber,
			"client_id":   clientId,
		})
		services.GetWASendSvc().SendTextMessageWithLog(
			fromNumber,
			"Maaf, terjadi kesalahan saat menghubungkan dengan CS. Silakan coba lagi.",
			clientId,
			businessPhone,
		)
		return
	}

	if err := gocom.PubSub().Publish(csDtos.TopicAIEscalationConfirmed, string(eventJSON)); err != nil {
		customLogger.ErrorWithData("Failed to publish CS escalation event", map[string]interface{}{
			"component":   "CSHandler",
			"function":    "HandleCSEscalationButtonClick",
			"topic":       csDtos.TopicAIEscalationConfirmed,
			"error":       err.Error(),
			"from_number": fromNumber,
		})
		services.GetWASendSvc().SendTextMessageWithLog(
			fromNumber,
			"Maaf, terjadi kesalahan saat menghubungkan dengan CS. Silakan coba lagi.",
			clientId,
			businessPhone,
		)
		return
	}

	customLogger.InfoWithData("CS escalation event published successfully", map[string]interface{}{
		"component":   "CSHandler",
		"function":    "HandleCSEscalationButtonClick",
		"from_number": fromNumber,
		"client_id":   clientId,
		"session_id":  sessionID,
	})

	confirmMsg := "Baik, saya akan menghubungkan Anda dengan customer service kami. Mohon tunggu sebentar..."
	_, messageLogID, sendErr := services.GetWASendSvc().SendTextMessageWithLog(fromNumber, confirmMsg, clientId, businessPhone)
	if sendErr != nil {
		customLogger.ErrorWithData("Failed to send CS confirmation message", map[string]interface{}{
			"component":   "CSHandler",
			"function":    "HandleCSEscalationButtonClick",
			"error":       sendErr.Message,
			"from_number": fromNumber,
		})
		return
	}

	assistantConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         fromNumber,
		ToNumber:           fromNumber,
		Role:               "assistant",
		Message:            confirmMsg,
		MessageLogId:       messageLogID,
		Status:             "sent",
		ConversationStatus: "escalated",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(assistantConv)

	customLogger.InfoWithData("CS escalation completed successfully", map[string]interface{}{
		"component":      "CSHandler",
		"function":       "HandleCSEscalationButtonClick",
		"from_number":    fromNumber,
		"client_id":      clientId,
		"session_id":     sessionID,
		"message_log_id": messageLogID,
		"status":         "escalated",
	})
}
