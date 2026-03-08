package dtos

import "time"

type CampaignReq struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // whatsapp, sms, email
	SenderId     string                 `json:"senderId"`
	TemplateId   string                 `json:"templateId"`
	ScheduledAt  string                 `json:"scheduledAt,omitempty"`
	BatchSize    int                    `json:"batchSize"`
	DelaySeconds int                    `json:"delaySeconds"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	Targets      []CampaignTarget       `json:"targets"`
}

type CampaignTarget struct {
	TargetType string `json:"targetType"` // contact, group, all
	TargetId   string `json:"targetId"`
}

type CampaignUpdateReq struct {
	Name         string                 `json:"name,omitempty"`
	Status       string                 `json:"status,omitempty"`
	ScheduledAt  string                 `json:"scheduledAt,omitempty"`
	BatchSize    int                    `json:"batchSize,omitempty"`
	DelaySeconds int                    `json:"delaySeconds,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
}

type Campaign struct {
	ID              string                 `json:"id"`
	ClientId        string                 `json:"clientId"`
	SenderId        string                 `json:"senderId"`
	SenderName      string                 `json:"senderName,omitempty"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	TemplateId      string                 `json:"templateId,omitempty"`
	TemplateName    string                 `json:"templateName,omitempty"`
	Status          string                 `json:"status"`
	ScheduledAt     string                 `json:"scheduledAt,omitempty"`
	StartedAt       string                 `json:"startedAt,omitempty"`
	CompletedAt     string                 `json:"completedAt,omitempty"`
	TotalRecipients int                    `json:"totalRecipients"`
	TotalSent       int                    `json:"totalSent"`
	TotalDelivered  int                    `json:"totalDelivered"`
	TotalFailed     int                    `json:"totalFailed"`
	TotalRead       int                    `json:"totalRead"`
	BatchSize       int                    `json:"batchSize"`
	DelaySeconds    int                    `json:"delaySeconds"`
	Parameters      map[string]interface{} `json:"parameters,omitempty"`
	Targets         []CampaignTarget       `json:"targets,omitempty"`
	CreatedBy       string                 `json:"createdBy,omitempty"`
	UpdatedBy       string                 `json:"updatedBy,omitempty"`
	CreatedAt       time.Time              `json:"createdAt,omitempty"`
	UpdatedAt       time.Time              `json:"updatedAt,omitempty"`
}

type CampaignRecipient struct {
	ID             string                 `json:"id"`
	CampaignId     string                 `json:"campaignId"`
	ContactId      string                 `json:"contactId"`
	ContactName    string                 `json:"contactName,omitempty"`
	RecipientType  string                 `json:"recipientType"`
	RecipientValue string                 `json:"recipientValue"`
	Status         string                 `json:"status"`
	MessageId      string                 `json:"messageId,omitempty"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
	SentAt         string                 `json:"sentAt,omitempty"`
	DeliveredAt    string                 `json:"deliveredAt,omitempty"`
	ReadAt         string                 `json:"readAt,omitempty"`
	FailedAt       string                 `json:"failedAt,omitempty"`
	ErrorMessage   string                 `json:"errorMessage,omitempty"`
	RetryCount     int                    `json:"retryCount"`
	CreatedAt      string                 `json:"createdAt,omitempty"`
	UpdatedAt      string                 `json:"updatedAt,omitempty"`
}

type CampaignStats struct {
	CampaignId      string  `json:"campaignId"`
	TotalRecipients int     `json:"totalRecipients"`
	TotalSent       int     `json:"totalSent"`
	TotalDelivered  int     `json:"totalDelivered"`
	TotalFailed     int     `json:"totalFailed"`
	TotalRead       int     `json:"totalRead"`
	SentRate        float64 `json:"sentRate"`
	DeliveredRate   float64 `json:"deliveredRate"`
	ReadRate        float64 `json:"readRate"`
	FailedRate      float64 `json:"failedRate"`
}

type CampaignExecuteReq struct {
	CampaignId string `json:"campaignId"`
}
