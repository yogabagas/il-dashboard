package dtos

type MessageLog struct {
	ID             string                 `json:"id"`
	ClientId       string                 `json:"clientId"`
	SenderId       string                 `json:"senderId,omitempty"`
	SenderName     string                 `json:"senderName,omitempty"`
	CampaignId     string                 `json:"campaignId,omitempty"`
	CampaignName   string                 `json:"campaignName,omitempty"`
	Type           string                 `json:"type"`                     // whatsapp, sms, email
	Direction      string                 `json:"direction"`                // inbound, outbound
	RecipientType  string                 `json:"recipientType"`            // phone, email
	RecipientValue string                 `json:"recipientValue"`           // phone number or email
	TemplateId     string                 `json:"templateId,omitempty"`     // template ID (for outbound template messages)
	TemplateName   string                 `json:"templateName,omitempty"`   // template name
	MessageId      string                 `json:"messageId,omitempty"`      // WhatsApp message ID
	SessionId      string                 `json:"sessionId,omitempty"`      // Conversation session ID
	MessageContent string                 `json:"messageContent,omitempty"` // Actual message content
	Status         string                 `json:"status"`                   // sent, delivered, read, failed, received
	Cost           float64                `json:"cost"`
	SentAt         string                 `json:"sentAt,omitempty"`
	DeliveredAt    string                 `json:"deliveredAt,omitempty"`
	ReadAt         string                 `json:"readAt,omitempty"`
	FailedAt       string                 `json:"failedAt,omitempty"`
	ErrorMessage   string                 `json:"errorMessage,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      string                 `json:"createdAt,omitempty"`
	UpdatedAt      string                 `json:"updatedAt,omitempty"`
}

type MessageLogSearchReq struct {
	Filter       string `json:"filter"`
	ClientId     string `json:"clientId"` // Filter by client ID
	CampaignId   string `json:"campaignId"`
	Type         string `json:"type"`
	Direction    string `json:"direction"` // inbound or outbound
	Status       string `json:"status"`
	RecipientVal string `json:"recipientVal"`
	DateFrom     string `json:"dateFrom"`
	DateTo       string `json:"dateTo"`
	PageNo       int    `json:"pageNo"`
	RowPerPage   int    `json:"rowPerPage"`
}
