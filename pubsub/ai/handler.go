package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ariandi/gocom"
	"gitlab.com/anti_metter/switching_common/sender"
	"gitlab.com/bot3342545/il-dashboard/constans"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/pubsub/wautil"
	"gitlab.com/bot3342545/il-dashboard/repositories/messageConversation"
	"gitlab.com/bot3342545/il-dashboard/services"
)

// ClientResolver resolves client ID from business phone number.
type ClientResolver func(phoneNumber string) *sender.Sender

// Handler contains AI-related helpers for WhatsApp webhook.
type Handler struct {
	resolveClientID ClientResolver
}

// NewHandler constructs AI handler with required dependencies.
func NewHandler(resolveClientID ClientResolver) *Handler {
	return &Handler{
		resolveClientID: resolveClientID,
	}
}

// ShouldTriggerAI checks if message should trigger AI conversation (initial trigger).
func (h *Handler) ShouldTriggerAI(messageText string) bool {
	lowerText := strings.ToLower(strings.TrimSpace(messageText))

	for _, trigger := range constans.PDAMGreetingWords {
		if strings.Contains(lowerText, trigger) {
			return true
		}
	}
	return false
}

// IsPDAMRelatedMessage checks if message contains PDAM-related keywords.
func (h *Handler) IsPDAMRelatedMessage(messageText string) bool {
	lowerText := strings.ToLower(strings.TrimSpace(messageText))

	for _, keyword := range constans.PDAMKeywords {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}

	trimmed := strings.TrimSpace(messageText)
	if len(trimmed) >= constans.CustomerNumberMinLength && len(trimmed) <= constans.CustomerNumberMaxLength {
		alphanumericCount := 0
		for _, r := range trimmed {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				alphanumericCount++
			}
		}
		if float64(alphanumericCount)/float64(len(trimmed)) > constans.AlphanumericThreshold {
			return true
		}
	}

	return false
}

// IsClosingConfirmation checks if user is confirming to close conversation.
func (h *Handler) IsClosingConfirmation(messageText string) bool {
	lowerText := strings.ToLower(strings.TrimSpace(messageText))

	exactConfirmations := []string{
		"iya", "ya", "yes", "yup", "yap", "ok", "oke", "okay", "okey",
		"baik", "siap", "betul", "benar", "setuju", "done", "selesai",
		"sudah", "cukup", "thanks", "thank you", "terima kasih",
		"makasih", "makasi",
	}

	for _, confirm := range exactConfirmations {
		if lowerText == confirm {
			return true
		}
	}

	confirmationPhrases := []string{
		"iya boleh",
		"ya boleh",
		"boleh ditutup",
		"silakan tutup",
		"sudah selesai",
		"sudah cukup",
		"iya terima kasih",
		"ya terima kasih",
	}

	for _, phrase := range confirmationPhrases {
		if strings.Contains(lowerText, phrase) {
			return true
		}
	}

	return false
}

// TriggerAIConversation triggers AI conversation response with function calling support.
func (h *Handler) TriggerAIConversation(sessionID, fromNumber, userMessage, businessPhoneNumber string) {
	customLogger.InfoWithData("Triggering AI conversation", map[string]interface{}{
		"component":      "AIHandler",
		"function":       "TriggerAIConversation",
		"session_id":     sessionID,
		"from_number":    fromNumber,
		"business_phone": businessPhoneNumber,
	})

	client := h.resolveClientID(businessPhoneNumber)

	aiResponse, inquiryResp, codedErr := services.GetGroqSvc().ChatWithKnowledgeAndFunctions(userMessage, client.ClientId, fromNumber)
	if codedErr != nil {
		customLogger.ErrorWithData("Failed to get AI response", map[string]interface{}{
			"component":   "AIHandler",
			"function":    "TriggerAIConversation",
			"error":       codedErr.Message,
			"from_number": fromNumber,
			"client_id":   client.ClientId,
		})
		return
	}

	if inquiryResp != nil {
		customLogger.InfoWithData("PPOB inquiry successful", map[string]interface{}{
			"component":   "AIHandler",
			"function":    "TriggerAIConversation",
			"from_number": fromNumber,
			"bill_id":     wautil.NormalizeBillID(inquiryResp.BillID),
			"action":      "sending_payment_button",
		})

		billID := wautil.NormalizeBillID(inquiryResp.BillID)
		if billID != "" {
			inquiryKey := fmt.Sprintf("ppob_inquiry:%s:%s", fromNumber, billID)
			inquiryData, _ := json.Marshal(inquiryResp)
			if err := gocom.KeyVal().Set(inquiryKey, string(inquiryData), 24*time.Hour); err != nil {
				customLogger.ErrorWithData("Failed to cache inquiry response", map[string]interface{}{
					"component":   "AIHandler",
					"function":    "TriggerAIConversation",
					"error":       err.Error(),
					"inquiry_key": inquiryKey,
				})
			} else {
				customLogger.InfoWithData("Cached inquiry response", map[string]interface{}{
					"component":   "AIHandler",
					"function":    "TriggerAIConversation",
					"bill_id":     billID,
					"inquiry_key": inquiryKey,
					"ttl":         "24h",
				})
			}
		}

		_, messageLogID, sendErr := services.GetWASendSvc().SendTextMessageWithLog(fromNumber, aiResponse, client.ClientId, businessPhoneNumber)
		if sendErr != nil {
			customLogger.ErrorWithData("Failed to send inquiry response", map[string]interface{}{
				"component":   "AIHandler",
				"function":    "TriggerAIConversation",
				"error":       sendErr.Message,
				"from_number": fromNumber,
			})
			return
		}

		assistantConv := &messageConversation.MessageConversation{
			SessionID:          sessionID,
			ClientId:           client.ClientId,
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

		billIDNormalized := wautil.NormalizeBillID(inquiryResp.BillID)
		if billIDNormalized == "" {
			customLogger.WarnWithData("Missing bill ID, skipping payment button", map[string]interface{}{
				"component":   "AIHandler",
				"function":    "TriggerAIConversation",
				"session_id":  sessionID,
				"from_number": fromNumber,
			})
			return
		}

		buttonText := "💳 Klik di bawah ini untuk melanjutkan pembayaran:"
		buttons := []map[string]string{
			{
				"id":    fmt.Sprintf("pay_%s_%.0f", billIDNormalized, wautil.NormalizeFloat(inquiryResp.TotalAmount)),
				"title": "💰 Bayar Sekarang",
			},
		}

		buttonMessageID, buttonLogID, buttonErr := services.GetWASendSvc().SendInteractiveButtonMessage(
			fromNumber,
			buttonText,
			buttons,
			client.ClientId,
			businessPhoneNumber,
		)
		if buttonErr != nil {
			customLogger.ErrorWithData("Failed to send payment button", map[string]interface{}{
				"component":   "AIHandler",
				"function":    "TriggerAIConversation",
				"error":       buttonErr.Message,
				"from_number": fromNumber,
				"bill_id":     billIDNormalized,
			})
			return
		}

		buttonConv := &messageConversation.MessageConversation{
			SessionID:          sessionID,
			ClientId:           client.ClientId,
			FromNumber:         fromNumber,
			ToNumber:           fromNumber,
			Role:               "assistant",
			Message:            "[Interactive Button: Bayar Sekarang]",
			MessageLogId:       buttonLogID,
			Status:             "sent",
			ConversationStatus: "active",
			Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
		}
		messageConversation.GetRepo().Create(buttonConv)

		customLogger.InfoWithData("Payment button sent", map[string]interface{}{
			"component":      "AIHandler",
			"function":       "TriggerAIConversation",
			"message_id":     buttonMessageID,
			"from_number":    fromNumber,
			"bill_id":        billIDNormalized,
			"message_log_id": buttonLogID,
		})
		return
	}

	waMessageID, messageLogID, sendErr := services.GetWASendSvc().SendTextMessageWithLog(fromNumber, aiResponse, client.ClientId, businessPhoneNumber)
	if sendErr != nil {
		customLogger.ErrorWithData("Failed to send AI response", map[string]interface{}{
			"component":   "AIHandler",
			"function":    "TriggerAIConversation",
			"error":       sendErr.Message,
			"from_number": fromNumber,
		})
		return
	}

	conversationStatus := "active"
	if h.shouldCloseConversation(aiResponse) {
		conversationStatus = "pending_close"
	}

	assistantConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           client.ClientId,
		FromNumber:         fromNumber,
		ToNumber:           fromNumber,
		Role:               "assistant",
		Message:            aiResponse,
		MessageLogId:       messageLogID,
		Status:             "sent",
		ConversationStatus: conversationStatus,
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(assistantConv)

	customLogger.InfoWithData("AI response sent successfully", map[string]interface{}{
		"component":           "AIHandler",
		"function":            "TriggerAIConversation",
		"message_id":          waMessageID,
		"from_number":         fromNumber,
		"conversation_status": conversationStatus,
		"message_log_id":      messageLogID,
	})
}

// SendRejectionMessage sends rejection message for non-PDAM questions.
func (h *Handler) SendRejectionMessage(sessionID, fromNumber, userMessage, businessPhoneNumber string) {
	customLogger.InfoWithData("Sending rejection for non-PDAM question", map[string]interface{}{
		"component":   "AIHandler",
		"function":    "SendRejectionMessage",
		"session_id":  sessionID,
		"from_number": fromNumber,
	})

	client := h.resolveClientID(businessPhoneNumber)

	rejectionMessage := constans.AIRejectionMessage
	waMessageID, messageLogID, sendErr := services.GetWASendSvc().SendTextMessageWithLog(fromNumber, rejectionMessage, client.ClientId, businessPhoneNumber)
	if sendErr != nil {
		customLogger.ErrorWithData("Failed to send rejection message", map[string]interface{}{
			"component":   "AIHandler",
			"function":    "SendRejectionMessage",
			"error":       sendErr.Message,
			"from_number": fromNumber,
			"client_id":   client.ClientId,
		})
		return
	}

	assistantConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           client.ClientId,
		FromNumber:         fromNumber,
		ToNumber:           fromNumber,
		Role:               "assistant",
		Message:            rejectionMessage,
		MessageLogId:       messageLogID,
		Status:             "sent",
		ConversationStatus: "active",
		Model:              constans.AIModel,
	}
	messageConversation.GetRepo().Create(assistantConv)

	customLogger.InfoWithData("Rejection message sent", map[string]interface{}{
		"component":      "AIHandler",
		"function":       "SendRejectionMessage",
		"message_id":     waMessageID,
		"from_number":    fromNumber,
		"message_log_id": messageLogID,
	})
}

func (h *Handler) shouldCloseConversation(aiResponse string) bool {
	lowerResponse := strings.ToLower(aiResponse)
	closingPhrases := []string{
		"bisa kami close",
		"bisa kami tutup",
		"dapat kami close",
		"dapat kami tutup",
		"bisa ditutup",
		"dapat ditutup",
		"sudah selesai kah",
		"ada lagi yang bisa",
	}

	for _, phrase := range closingPhrases {
		if strings.Contains(lowerResponse, phrase) {
			return true
		}
	}
	return false
}
