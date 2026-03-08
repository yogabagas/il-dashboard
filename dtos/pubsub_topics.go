package dtos

// ============================================================================
// WhatsApp Message Routing Topics
// ============================================================================

const (
	// TopicWAMessageAIKnowledge - AI general conversation (no PPOB, no CS)
	TopicWAMessageAIKnowledge = "wa.message.ai.knowledge"

	// TopicWAMessageAIPPOB - PPOB inquiry & payment flow
	TopicWAMessageAIPPOB = "wa.message.ai.ppob"

	// TopicWAMessageAICS - CS escalation flow
	TopicWAMessageAICS = "wa.message.ai.cs"

	// TopicWAMessageImage - Image message processing (OCR, vision)
	TopicWAMessageImage = "wa.message.image"
)

// WAMessageEvent represents a WhatsApp message event for routing
type WAMessageEvent struct {
	SessionID          string `json:"session_id"`
	FromNumber         string `json:"from_number"`
	ToNumber           string `json:"to_number"` // Business phone number
	Message            string `json:"message"`
	MessageType        string `json:"message_type"` // text, image, video, etc
	ClientID           string `json:"client_id"`
	ContactName        string `json:"contact_name,omitempty"`
	Timestamp          string `json:"timestamp"` // WhatsApp timestamp as string
	WhatsAppMessageID  string `json:"whatsapp_message_id,omitempty"`
	RoutingCategory    string `json:"routing_category"` // knowledge, ppob, cs
	DetectedIntent     string `json:"detected_intent,omitempty"`
	IsGreeting         bool   `json:"is_greeting"`
	IsPDAMRelated      bool   `json:"is_pdam_related"`
	IsNumberOnly       bool   `json:"is_number_only"`      // For PPOB detection
	RequiresEscalation bool   `json:"requires_escalation"` // For CS detection
}

// WAImageMessageEvent represents a WhatsApp image message event for OCR processing
type WAImageMessageEvent struct {
	SessionID         string `json:"session_id"`
	FromNumber        string `json:"from_number"`
	ToNumber          string `json:"to_number"` // Business phone number
	ImageID           string `json:"image_id"`  // WhatsApp media ID
	ImageMimeType     string `json:"image_mime_type"`
	ImageCaption      string `json:"image_caption,omitempty"`
	ClientID          string `json:"client_id"`
	ContactName       string `json:"contact_name,omitempty"`
	Timestamp         string `json:"timestamp"`
	WhatsAppMessageID string `json:"whatsapp_message_id,omitempty"`
}
