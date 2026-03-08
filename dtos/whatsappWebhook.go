package dtos

// WhatsAppWebhookVerification untuk GET request saat Meta melakukan verifikasi webhook
type WhatsAppWebhookVerification struct {
	Mode      string `query:"hub.mode"`
	Token     string `query:"hub.verify_token"`
	Challenge string `query:"hub.challenge"`
}

// WhatsAppWebhookRequest adalah struktur utama webhook dari Meta
type WhatsAppWebhookRequest struct {
	Object string              `json:"object"`
	Entry  []WhatsAppEntryData `json:"entry"`
}

type WhatsAppEntryData struct {
	ID      string               `json:"id"`
	Time    int64                `json:"time,omitempty"` // Unix timestamp
	Changes []WhatsAppChangeData `json:"changes"`
}

type WhatsAppChangeData struct {
	Value WhatsAppValueData `json:"value"`
	Field string            `json:"field"`
}

type WhatsAppValueData struct {
	MessagingProduct string            `json:"messaging_product"`
	Metadata         WhatsAppMetadata  `json:"metadata"`
	Contacts         []WhatsAppContact `json:"contacts,omitempty"`
	Messages         []WhatsAppMessage `json:"messages,omitempty"`
	Statuses         []WhatsAppStatus  `json:"statuses,omitempty"`

	// Template status update fields (flattened in value object) - Format 1
	Event                   string `json:"event,omitempty"`
	MessageTemplateID       int64  `json:"message_template_id,omitempty"`
	MessageTemplateName     string `json:"message_template_name,omitempty"`
	MessageTemplateLanguage string `json:"message_template_language,omitempty"`
	MessageTemplateCategory string `json:"message_template_category,omitempty"`
	Reason                  string `json:"reason,omitempty"`

	// Template status update fields (nested object) - Format 2
	MessageTemplateStatusUpdate *WhatsAppTemplateStatusEvent `json:"message_template_status_update,omitempty"`
}

type WhatsAppMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type WhatsappEmbeddedSignupData struct {
	Data Data `json:"data"`
}

type Data struct {
	BusinessId          string   `json:"business_id"`
	PhoneNumberId       string   `json:"phone_number_id"`
	WabaId              string   `json:"waba_id"`
	CatalogIds          []string `json:"catalog_ids"`
	DatasetIds          []string `json:"dataset_ids"`
	InstagramAccountIds []string `json:"instagram_account_ids"`
	PageIds             []string `json:"page_ids"`
}

type WhatsAppContact struct {
	Profile WhatsAppProfile `json:"profile"`
	WaID    string          `json:"wa_id"`
}

type WhatsAppProfile struct {
	Name string `json:"name"`
}

type WhatsAppMessage struct {
	From        string               `json:"from"`
	ID          string               `json:"id"`
	Timestamp   string               `json:"timestamp"`
	Type        string               `json:"type"`
	Text        *WhatsAppText        `json:"text,omitempty"`
	Image       *WhatsAppMedia       `json:"image,omitempty"`
	Audio       *WhatsAppMedia       `json:"audio,omitempty"`
	Video       *WhatsAppMedia       `json:"video,omitempty"`
	Document    *WhatsAppMedia       `json:"document,omitempty"`
	Location    *WhatsAppLocation    `json:"location,omitempty"`
	Button      *WhatsAppButton      `json:"button,omitempty"`
	Interactive *WhatsAppInteractive `json:"interactive,omitempty"`
}

type WhatsAppText struct {
	Body string `json:"body"`
}

type WhatsAppMedia struct {
	ID       string `json:"id,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type WhatsAppLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type WhatsAppButton struct {
	Text    string `json:"text"`
	Payload string `json:"payload"`
}

type WhatsAppInteractive struct {
	Type        string               `json:"type"`
	ButtonReply *WhatsAppButtonReply `json:"button_reply,omitempty"`
	ListReply   *WhatsAppListReply   `json:"list_reply,omitempty"`
}

type WhatsAppButtonReply struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type WhatsAppListReply struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type WhatsAppStatus struct {
	ID           string                `json:"id"`
	Status       string                `json:"status"`
	Timestamp    string                `json:"timestamp"`
	RecipientID  string                `json:"recipient_id"`
	Conversation *WhatsAppConversation `json:"conversation,omitempty"`
	Pricing      *WhatsAppPricing      `json:"pricing,omitempty"`
	Errors       []WhatsAppError       `json:"errors,omitempty"`
}

type WhatsAppConversation struct {
	ID                  string                     `json:"id"`
	ExpirationTimestamp string                     `json:"expiration_timestamp,omitempty"`
	Origin              WhatsAppConversationOrigin `json:"origin"`
}

type WhatsAppConversationOrigin struct {
	Type string `json:"type"`
}

type WhatsAppPricing struct {
	PricingModel string `json:"pricing_model"`
	Billable     bool   `json:"billable"`
	Category     string `json:"category"`
}

type WhatsAppError struct {
	Code    int    `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message,omitempty"`
	Details string `json:"error_data,omitempty"`
}

// WhatsAppTemplateStatusEvent untuk template status update webhook
type WhatsAppTemplateStatusEvent struct {
	MessageTemplateID       int64  `json:"message_template_id"`
	MessageTemplateName     string `json:"message_template_name"`
	MessageTemplateLanguage string `json:"message_template_language"`
	MessageTemplateCategory string `json:"message_template_category,omitempty"` // MARKETING, UTILITY, AUTHENTICATION
	Event                   string `json:"event"`                               // APPROVED, REJECTED, DISABLED, PENDING, etc.
	Reason                  string `json:"reason,omitempty"`
}
