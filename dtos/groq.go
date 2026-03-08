package dtos

// Groq API Internal Structures

type GroqMessageInternal struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCalls  []GroqToolCall `json:"tool_calls,omitempty"`
	ToolCallId string         `json:"tool_call_id,omitempty"`
}

type GroqToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"` // JSON string
	} `json:"function"`
}

type GroqRequest struct {
	Model       string                `json:"model"`
	Messages    []GroqMessageInternal `json:"messages"`
	Temperature float64               `json:"temperature,omitempty"`
	MaxTokens   int                   `json:"max_tokens,omitempty"`
	Tools       []GroqTool            `json:"tools,omitempty"`
	ToolChoice  interface{}           `json:"tool_choice,omitempty"` // Can be string "auto" or object
}

type GroqResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string         `json:"role"`
			Content   string         `json:"content"`
			ToolCalls []GroqToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}
