package dtos

type MessageConversation struct {
	ID                 string `json:"id"`
	ClientId           string `json:"clientId"`
	SessionID          string `json:"sessionId"`
	FromNumber         string `json:"fromNumber"`
	ToNumber           string `json:"toNumber,omitempty"`
	Role               string `json:"role"`
	Message            string `json:"message"`
	MessageLogId       string `json:"messageLogId,omitempty"`
	TokensUsed         int    `json:"tokensUsed,omitempty"`
	Model              string `json:"model,omitempty"`
	Status             string `json:"status"`
	ConversationStatus string `json:"conversationStatus"`
	ErrorMessage       string `json:"errorMessage,omitempty"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

type MessageConversationSearchReq struct {
	Filter     string `json:"filter"`
	ClientId   string `json:"clientId"`   // Admin only: filter by client
	FromNumber string `json:"fromNumber"` // Search by sender phone number
	SessionId  string `json:"sessionId"`
	DateFrom   string `json:"dateFrom"`
	DateTo     string `json:"dateTo"`
	PageNo     int    `json:"pageNo"`
	RowPerPage int    `json:"rowPerPage"`
}

type ConversationReplyReq struct {
	FromNumber string `json:"fromNumber"` // Customer phone to reply to
	Message    string `json:"message"`    // Text message content
	ClientId   string `json:"clientId"`   // Admin only: override clientId
}
