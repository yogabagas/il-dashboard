package dtos

type WASendMessageReq struct {
	SenderId   string               `json:"senderId,omitempty"`
	TemplateId string               `json:"templateId"`
	To         string               `json:"to"`
	Parameters []WAMessageParameter `json:"parameters,omitempty"`
	CampaignId string               `json:"campaignId,omitempty"` // Optional campaign ID
}

type WASendBulkMessageReq struct {
	SenderId   string                `json:"senderId,omitempty"`
	TemplateId string                `json:"templateId"`
	Recipients []WASendBulkRecipient `json:"recipients"`
}

type WASendBulkRecipient struct {
	To         string               `json:"to"`
	Parameters []WAMessageParameter `json:"parameters,omitempty"`
}

type WAMessageParameter struct {
	Type     string               `json:"type"` // text, image, video, document
	Text     string               `json:"text,omitempty"`
	Image    *WAMessageMediaParam `json:"image,omitempty"`
	Video    *WAMessageMediaParam `json:"video,omitempty"`
	Document *WAMessageMediaParam `json:"document,omitempty"`
}

type WAMessageMediaParam struct {
	Link string `json:"link,omitempty"` // URL of the media
	ID   string `json:"id,omitempty"`   // Media ID if already uploaded to Meta
}

type WASendMessageResp struct {
	MessageId        string `json:"messageId"`
	To               string `json:"to"`
	Status           string `json:"status"`
	MessagingProduct string `json:"messagingProduct,omitempty"`
}

type WASendBulkMessageResp struct {
	TotalRequested int                 `json:"totalRequested"`
	TotalSuccess   int                 `json:"totalSuccess"`
	TotalFailed    int                 `json:"totalFailed"`
	Results        []WASendMessageResp `json:"results"`
}

// WASenderConfig defines the configuration structure for WhatsApp sender
type WASenderConfig struct {
	AccessToken   string `json:"access_token"`
	PhoneNumberId string `json:"phone_number_id"`
	WabaId        string `json:"waba_id"`
}

// Response dari WhatsApp Business API
type WAApiSendResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaId  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		Id string `json:"id"`
	} `json:"messages"`
}

type SendMessageProcess struct {
	MessageLogId   string               `json:"messageLogId"`
	MessageType    string               `json:"messageType"`
	PhoneNumberId  string               `json:"phoneNumberId"`
	AccessToken    string               `json:"accessToken"`
	TemplateName   string               `json:"templateName"`
	TemplateId     string               `json:"templateId,omitempty"`
	Language       string               `json:"language"`
	To             string               `json:"to"`
	Parameters     []WAMessageParameter `json:"parameters,omitempty"`
	Category       string               `json:"category,omitempty"`
	ClientId       string               `json:"clientId"`
	SenderId       string               `json:"senderId"`
	CampaignId     string               `json:"campaignId,omitempty"`
	MessageContent string               `json:"messageContent,omitempty"`
	Cost           float64              `json:"cost,omitempty"`
	UserId         string               `json:"userId,omitempty"` // For billing deduction
}
