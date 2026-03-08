package dtos

// AI Chat DTOs
type AIChatReq struct {
	Message string `json:"message" binding:"required"`
	From    string `json:"from"`
}

type AIChatResp struct {
	Response string `json:"response"`
	From     string `json:"from,omitempty"`
}

// AI Webhook DTOs
type AIWebhookReq struct {
	SessionID string `json:"session_id" binding:"required"`
	From      string `json:"from" binding:"required"`
	Message   string `json:"message" binding:"required"`
	IsGroup   bool   `json:"is_group"`
}

type AIWebhookResp struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	From       string `json:"from"`
	AIResponse string `json:"ai_response"`
}

// AI Chat Context DTOs
type AIChatContextReq struct {
	Messages []GroqMessage `json:"messages" binding:"required"`
	From     string        `json:"from"`
}

// AI Conversation DTOs (untuk send message conversation dengan AI)
type AIConversationReq struct {
	SessionID  string `json:"session_id" binding:"required"`
	From       string `json:"from" binding:"required"`
	Message    string `json:"message" binding:"required"`
	TemplateId string `json:"template_id"` // Optional, kalau mau pakai template
}

type AIConversationResp struct {
	MessageId  string `json:"message_id"`
	To         string `json:"to"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	AIResponse string `json:"ai_response"`
}

// GroqMessage represents a message in conversation
type GroqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Groq Function Calling DTOs
type GroqTool struct {
	Type     string                 `json:"type"`
	Function GroqFunctionDefinition `json:"function"`
}

type GroqFunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type GroqToolChoice struct {
	Type string `json:"type"` // "auto", "none", atau specific function
}
