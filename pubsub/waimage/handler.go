package waimage

import (
	"fmt"
	"strings"
	"time"

	"gitlab.com/anti_metter/switching_common/sender"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/repositories/messageConversation"
	"gitlab.com/bot3342545/il-dashboard/services"
)

// Router defines minimal route contract to avoid circular dependency.
type Router interface {
	RouteMessage(fromNumber, toNumber, message, messageType, contactName, timestamp, waMessageID string)
}

// ClientResolver resolves client ID from business phone number.
type ClientResolver func(phoneNumber string) *sender.Sender

// Handler manages image/OCR related WhatsApp interactions.
type Handler struct {
	router        Router
	resolveClient ClientResolver
}

// NewHandler constructs image handler.
func NewHandler(router Router, resolveClient ClientResolver) *Handler {
	return &Handler{
		router:        router,
		resolveClient: resolveClient,
	}
}

// HandleOCRConfirmation handles when user confirms OCR result is correct.
func (h *Handler) HandleOCRConfirmation(fromNumber, buttonID, businessPhone string) {
	customLogger.InfoWithData("Processing OCR confirmation", map[string]interface{}{
		"component":      "WAImageHandler",
		"function":       "HandleOCRConfirmation",
		"from_number":    fromNumber,
		"button_id":      buttonID,
		"business_phone": businessPhone,
	})

	sessionID := strings.TrimPrefix(buttonID, "ocr_confirm_")

	conversations := messageConversation.GetRepo().GetConversationHistoryBySession(sessionID, 1)
	if len(conversations) == 0 {
		customLogger.ErrorWithData("OCR session not found", map[string]interface{}{
			"component":   "WAImageHandler",
			"function":    "HandleOCRConfirmation",
			"session_id":  sessionID,
			"from_number": fromNumber,
		})
		services.GetWASendSvc().SendTextMessage(fromNumber, "Maaf, sesi OCR tidak ditemukan. Silakan kirim gambar ulang.")
		return
	}

	extractedText := conversations[0].Message
	customLogger.InfoWithData("Retrieved OCR text", map[string]interface{}{
		"component":      "WAImageHandler",
		"function":       "HandleOCRConfirmation",
		"from_number":    fromNumber,
		"extracted_text": extractedText,
		"text_length":    len(extractedText),
	})

	confirmMsg := "✅ Baik, saya akan memproses data tersebut..."
	services.GetWASendSvc().SendTextMessage(fromNumber, confirmMsg)

	client := h.resolveClient(businessPhone)
	h.router.RouteMessage(
		fromNumber,
		businessPhone,
		extractedText,
		"text",
		"",
		fmt.Sprintf("%d", time.Now().Unix()),
		"",
	)

	conversations[0].Status = "confirmed"
	messageConversation.GetRepo().Update(&conversations[0])

	clientID := ""
	if client != nil {
		clientID = client.ClientId
	}
	customLogger.InfoWithData("OCR confirmed and routed to AI", map[string]interface{}{
		"component":   "WAImageHandler",
		"function":    "HandleOCRConfirmation",
		"from_number": fromNumber,
		"client_id":   clientID,
		"session_id":  sessionID,
		"status":      "confirmed",
	})
}

// HandleOCRRejection handles when user says OCR result is wrong.
func (h *Handler) HandleOCRRejection(fromNumber string) {
	customLogger.InfoWithData("User rejected OCR", map[string]interface{}{
		"component":   "WAImageHandler",
		"function":    "HandleOCRRejection",
		"from_number": fromNumber,
	})

	rejectMsg := `❌ Baik, pembacaan gambar mungkin kurang akurat.

Silakan ketik informasi yang Anda butuhkan secara manual, dan saya akan membantu Anda. 😊`

	services.GetWASendSvc().SendTextMessage(fromNumber, rejectMsg)

	customLogger.InfoWithData("Rejection handled, user instructed to type manually", map[string]interface{}{
		"component":   "WAImageHandler",
		"function":    "HandleOCRRejection",
		"from_number": fromNumber,
		"status":      "rejected",
	})
}
