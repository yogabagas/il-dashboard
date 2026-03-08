package payment

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ariandi/gocom"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/pubsub/wautil"
	"gitlab.com/bot3342545/il-dashboard/repositories/messageConversation"
	"gitlab.com/bot3342545/il-dashboard/services"
)

// Handler handles payment related WhatsApp interactions.
type Handler struct{}

// NewHandler constructs payment handler.
func NewHandler() *Handler {
	return &Handler{}
}

// HandlePaymentButtonClick handles "Bayar Sekarang" button click and shows bank selection.
func (h *Handler) HandlePaymentButtonClick(sessionID, fromNumber, buttonID, clientId string) {
	customLogger.InfoWithData("Processing payment button click", map[string]interface{}{
		"component":   "PaymentHandler",
		"function":    "HandlePaymentButtonClick",
		"session_id":  sessionID,
		"from_number": fromNumber,
		"button_id":   buttonID,
		"client_id":   clientId,
	})

	parts := strings.Split(buttonID, "_")
	if len(parts) < 3 {
		customLogger.ErrorWithData("Invalid button ID format", map[string]interface{}{
			"component":   "PaymentHandler",
			"function":    "HandlePaymentButtonClick",
			"button_id":   buttonID,
			"from_number": fromNumber,
		})
		return
	}

	billID := parts[1]

	userConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         fromNumber,
		Role:               "user",
		Message:            "[User clicked: Bayar Sekarang]",
		Status:             "received",
		ConversationStatus: "active",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(userConv)

	bodyText := "💳 Silakan pilih bank untuk mendapatkan Virtual Account:"
	buttonText := "Pilih Bank"

	sections := []map[string]interface{}{
		{
			"title": "Virtual Account",
			"rows": []map[string]interface{}{
				{
					"id":          fmt.Sprintf("bank_%s_bca", billID),
					"title":       "BCA Virtual Account",
					"description": "Pembayaran via VA BCA",
				},
				{
					"id":          fmt.Sprintf("bank_%s_mandiri", billID),
					"title":       "Mandiri Virtual Account",
					"description": "Pembayaran via VA Mandiri",
				},
				{
					"id":          fmt.Sprintf("bank_%s_bri", billID),
					"title":       "BRI Virtual Account",
					"description": "Pembayaran via VA BRI",
				},
			},
		},
	}

	listMessageID, listLogID, listErr := services.GetWASendSvc().SendInteractiveListMessage(
		fromNumber,
		bodyText,
		buttonText,
		sections,
		clientId,
		"", // businessPhone not available yet - will use fallback config
	)

	if listErr != nil {
		customLogger.ErrorWithData("Failed to send bank selection list", map[string]interface{}{
			"component":   "PaymentHandler",
			"function":    "HandlePaymentButtonClick",
			"error":       listErr.Message,
			"from_number": fromNumber,
			"bill_id":     billID,
		})
		return
	}

	listConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         fromNumber,
		ToNumber:           fromNumber,
		Role:               "assistant",
		Message:            "[Interactive List: Bank Selection]",
		MessageLogId:       listLogID,
		Status:             "sent",
		ConversationStatus: "active",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(listConv)

	customLogger.InfoWithData("Bank selection list sent", map[string]interface{}{
		"component":     "PaymentHandler",
		"function":      "HandlePaymentButtonClick",
		"message_id":    listMessageID,
		"from_number":   fromNumber,
		"bill_id":       billID,
		"message_log_id": listLogID,
	})
}

// HandleCopyVAButtonClick handles "Copy VA" button click and sends VA number as text.
func (h *Handler) HandleCopyVAButtonClick(sessionID, fromNumber, buttonID, clientId string) {
	customLogger.InfoWithData("Processing copy VA button click", map[string]interface{}{
		"component":   "PaymentHandler",
		"function":    "HandleCopyVAButtonClick",
		"session_id":  sessionID,
		"from_number": fromNumber,
		"button_id":   buttonID,
		"client_id":   clientId,
	})

	parts := strings.Split(buttonID, "_")
	if len(parts) < 3 {
		customLogger.ErrorWithData("Invalid button ID format", map[string]interface{}{
			"component":   "PaymentHandler",
			"function":    "HandleCopyVAButtonClick",
			"button_id":   buttonID,
			"from_number": fromNumber,
		})
		return
	}

	vaNumber := strings.Join(parts[2:], "_")
	customLogger.InfoWithData("Parsed VA Number", map[string]interface{}{
		"component":   "PaymentHandler",
		"function":    "HandleCopyVAButtonClick",
		"va_number":   vaNumber,
		"from_number": fromNumber,
	})

	userConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         fromNumber,
		Role:               "user",
		Message:            "[User clicked: Copy VA]",
		Status:             "received",
		ConversationStatus: "active",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(userConv)

	responseMsg := vaNumber

	waMessageID, messageLogID, sendErr := services.GetWASendSvc().SendTextMessageWithLog(fromNumber, responseMsg, clientId, "")
	if sendErr != nil {
		customLogger.ErrorWithData("Failed to send VA number", map[string]interface{}{
			"component":   "PaymentHandler",
			"function":    "HandleCopyVAButtonClick",
			"error":       sendErr.Message,
			"from_number": fromNumber,
			"va_number":   vaNumber,
		})
		return
	}

	assistantConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         fromNumber,
		ToNumber:           fromNumber,
		Role:               "assistant",
		Message:            responseMsg,
		MessageLogId:       messageLogID,
		Status:             "sent",
		ConversationStatus: "active",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(assistantConv)

	customLogger.InfoWithData("VA number sent", map[string]interface{}{
		"component":      "PaymentHandler",
		"function":       "HandleCopyVAButtonClick",
		"message_id":     waMessageID,
		"va_number":      vaNumber,
		"from_number":    fromNumber,
		"message_log_id": messageLogID,
	})
}

// HandleBankSelection handles bank selection from list and processes payment.
func (h *Handler) HandleBankSelection(sessionID, fromNumber, listID, clientId string) {
	customLogger.InfoWithData("Processing bank selection", map[string]interface{}{
		"component":   "PaymentHandler",
		"function":    "HandleBankSelection",
		"session_id":  sessionID,
		"from_number": fromNumber,
		"list_id":     listID,
		"client_id":   clientId,
	})

	parts := strings.Split(listID, "_")
	if len(parts) < 3 {
		customLogger.ErrorWithData("Invalid list ID format", map[string]interface{}{
			"component":   "PaymentHandler",
			"function":    "HandleBankSelection",
			"list_id":     listID,
			"from_number": fromNumber,
		})
		return
	}

	billID := parts[1]
	bankCode := strings.ToUpper(parts[2])

	inquiryKey := fmt.Sprintf("ppob_inquiry:%s:%s", fromNumber, billID)
	inquiryData := gocom.KeyVal().Get(inquiryKey)
	if inquiryData == "" {
		customLogger.ErrorWithData("Failed to retrieve inquiry from cache", map[string]interface{}{
			"component":   "PaymentHandler",
			"function":    "HandleBankSelection",
			"inquiry_key": inquiryKey,
			"from_number": fromNumber,
			"bill_id":     billID,
		})

		errorMsg := `Maaf, data tagihan tidak ditemukan atau sudah kadaluarsa. 

Silakan lakukan pengecekan tagihan kembali dengan mengirim nomor pelanggan Anda.`

		_, _, _ = services.GetWASendSvc().SendTextMessageWithLog(fromNumber, errorMsg, clientId, "")
		return
	}

	var inquiryResp dtos.PPOBInquiryResp
	if err := json.Unmarshal([]byte(inquiryData), &inquiryResp); err != nil {
		customLogger.ErrorWithData("Failed to unmarshal inquiry data", map[string]interface{}{
			"component":   "PaymentHandler",
			"function":    "HandleBankSelection",
			"error":       err.Error(),
			"from_number": fromNumber,
			"bill_id":     billID,
		})
		return
	}

	customLogger.InfoWithData("Retrieved inquiry data", map[string]interface{}{
		"component":    "PaymentHandler",
		"function":     "HandleBankSelection",
		"bill_id":      billID,
		"total_amount": *inquiryResp.TotalAmount,
		"from_number":  fromNumber,
	})

	userConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         fromNumber,
		Role:               "user",
		Message:            fmt.Sprintf("[User selected bank: %s for payment]", bankCode),
		Status:             "received",
		ConversationStatus: "active",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(userConv)

	bankName := bankCode
	switch bankCode {
	case "BCA":
		bankName = "BCA"
	case "MANDIRI", "BMRI":
		bankName = "Mandiri"
	case "BRI":
		bankName = "BRI"
	}

	processingMsg := fmt.Sprintf(`💳 *Memproses Pembayaran via %s*

⏳ Sedang memproses pembayaran untuk tagihan *%s*...

_Mohon tunggu sebentar..._`, bankName, billID)

	_, _, _ = services.GetWASendSvc().SendTextMessageWithLog(fromNumber, processingMsg, clientId, "")

	userMessage := fmt.Sprintf("bayar pakai %s", bankCode)
	paymentResponse, paymentResp, paymentErr := services.GetGroqSvc().ChatWithKnowledgeAndPaymentFunction(
		userMessage,
		clientId,
		fromNumber,
		&inquiryResp,
	)

	if paymentErr != nil {
		customLogger.ErrorWithData("Payment failed", map[string]interface{}{
			"component":   "PaymentHandler",
			"function":    "HandleBankSelection",
			"error":       paymentErr.Message,
			"from_number": fromNumber,
			"bill_id":     billID,
			"bank_code":   bankCode,
		})

		errorMsg := fmt.Sprintf(`❌ *Pembayaran Gagal*

Maaf, terjadi kesalahan saat memproses pembayaran.

Error: %s

Silakan coba lagi atau hubungi customer service kami.`, paymentErr.Message)

		waMessageID, messageLogID, _ := services.GetWASendSvc().SendTextMessageWithLog(fromNumber, errorMsg, clientId, "")

		errorConv := &messageConversation.MessageConversation{
			SessionID:          sessionID,
			ClientId:           clientId,
			FromNumber:         fromNumber,
			ToNumber:           fromNumber,
			Role:               "assistant",
			Message:            errorMsg,
			MessageLogId:       messageLogID,
			Status:             "sent",
			ConversationStatus: "active",
			Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
		}
		messageConversation.GetRepo().Create(errorConv)

		customLogger.InfoWithData("Error message sent", map[string]interface{}{
			"component":      "PaymentHandler",
			"function":       "HandleBankSelection",
			"message_id":     waMessageID,
			"from_number":    fromNumber,
			"message_log_id": messageLogID,
		})
		return
	}

	waMessageID, messageLogID, sendErr := services.GetWASendSvc().SendTextMessageWithLog(fromNumber, paymentResponse, clientId, "")
	if sendErr != nil {
		customLogger.ErrorWithData("Failed to send payment response", map[string]interface{}{
			"component":   "PaymentHandler",
			"function":    "HandleBankSelection",
			"error":       sendErr.Message,
			"from_number": fromNumber,
			"bill_id":     billID,
		})
		return
	}

	conversationStatus := "active"
	if paymentResp != nil {
		conversationStatus = "completed"

		customLogger.InfoWithData("Payment successful", map[string]interface{}{
			"component":    "PaymentHandler",
			"function":     "HandleBankSelection",
			"tx_id":        wautil.NormalizeString(paymentResp.TxID),
			"bill_id":      wautil.NormalizeString(paymentResp.BillID),
			"bank_code":    bankCode,
			"total_amount": wautil.NormalizeFloat(paymentResp.TotalAmount),
			"from_number":  fromNumber,
		})
	}

	assistantConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         fromNumber,
		ToNumber:           fromNumber,
		Role:               "assistant",
		Message:            paymentResponse,
		MessageLogId:       messageLogID,
		Status:             "sent",
		ConversationStatus: conversationStatus,
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(assistantConv)

	customLogger.InfoWithData("Payment response sent", map[string]interface{}{
		"component":            "PaymentHandler",
		"function":             "HandleBankSelection",
		"message_id":           waMessageID,
		"bank_code":            bankCode,
		"conversation_status":  conversationStatus,
		"from_number":          fromNumber,
		"message_log_id":       messageLogID,
	})

	if paymentResp != nil && paymentResp.Status != nil && *paymentResp.Status == "1" {
		if paymentResp.Detail != nil && paymentResp.Detail.VaNo != "" {
			vaNumber := paymentResp.Detail.VaNo

			buttonText := "📱 *Copy Nomor Virtual Account*"
			buttons := []map[string]string{
				{
					"id":    fmt.Sprintf("copy_va_%s", vaNumber),
					"title": "📋 Copy VA Dibawah",
				},
			}

			buttonMessageID, buttonLogID, buttonErr := services.GetWASendSvc().SendInteractiveButtonMessage(
				fromNumber,
				buttonText,
				buttons,
				clientId,
				"", // businessPhone not available yet - will use fallback config
			)

			if buttonErr != nil {
				customLogger.ErrorWithData("Failed to send Copy VA button", map[string]interface{}{
					"component":   "PaymentHandler",
					"function":    "HandleBankSelection",
					"error":       buttonErr.Message,
					"from_number": fromNumber,
					"va_number":   vaNumber,
				})
			} else {
				buttonConv := &messageConversation.MessageConversation{
					SessionID:          sessionID,
					ClientId:           clientId,
					FromNumber:         fromNumber,
					ToNumber:           fromNumber,
					Role:               "assistant",
					Message:            fmt.Sprintf("[Copy VA Button: %s]", vaNumber),
					MessageLogId:       buttonLogID,
					Status:             "sent",
					ConversationStatus: conversationStatus,
					Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
				}
				messageConversation.GetRepo().Create(buttonConv)

				customLogger.InfoWithData("Copy VA button sent", map[string]interface{}{
					"component":      "PaymentHandler",
					"function":       "HandleBankSelection",
					"message_id":     buttonMessageID,
					"va_number":      vaNumber,
					"from_number":    fromNumber,
					"message_log_id": buttonLogID,
				})
			}
		}
	}

	if paymentResp != nil {
		if err := gocom.KeyVal().Del(inquiryKey); err != nil {
			customLogger.WarnWithData("Failed to clear inquiry cache", map[string]interface{}{
				"component":   "PaymentHandler",
				"function":    "HandleBankSelection",
				"error":       err.Error(),
				"inquiry_key": inquiryKey,
			})
		} else {
			customLogger.InfoWithData("Cleared inquiry cache", map[string]interface{}{
				"component":   "PaymentHandler",
				"function":    "HandleBankSelection",
				"inquiry_key": inquiryKey,
				"bill_id":     billID,
			})
		}
	}
}
