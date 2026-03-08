package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/config"
	"gitlab.com/anti_metter/switching_common/client"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/repositories/aiKnowledgeBase"
	"gitlab.com/bot3342545/il-dashboard/repositories/messageConversation"
)

type GroqSvc interface {
	Chat(message string) (string, *gocom.CodedError)
	ChatWithKnowledge(message, clientId, fromNumber string) (string, *gocom.CodedError)
	ChatWithContext(messages []dtos.GroqMessage) (string, *gocom.CodedError)
	SendConversationMessage(sessionID, from, userMessage, clientId string) (string, string, *gocom.CodedError)
	ChatWithKnowledgeAndFunctions(message, clientId, from string) (string, *dtos.PPOBInquiryResp, *gocom.CodedError)
	ChatWithKnowledgeAndPaymentFunction(message, clientId, from string, inquiryResp *dtos.PPOBInquiryResp) (string, *dtos.PPOBPaymentResp, *gocom.CodedError)
	ChatWithEscalation(userMessage, clientId, from string) (string, *EscalationInfo, *gocom.CodedError)
}

type GroqSvcImpl struct {
	apiKey  string
	baseURL string
}

// Chat sends a message to Groq AI and returns the response
func (o *GroqSvcImpl) Chat(userMessage string) (string, *gocom.CodedError) {
	customLogger.InfoWithData("Chat started", map[string]interface{}{
		"component":      "GroqService",
		"function":       "Chat",
		"message_length": len(userMessage),
	})

	if o.apiKey == "" {
		customLogger.WarnWithData("GROQ_API_KEY not set", map[string]interface{}{
			"component": "GroqService",
			"function":  "Chat",
		})
		return "", gocom.NewError(500, "GROQ_API_KEY not set in environment")
	}

	// Get AI name from config
	aiName := config.Get(constans.GroqAIName, "Asisten")

	// Prepare system prompt with AI name
	systemPrompt := fmt.Sprintf("Kamu adalah %s, asisten WhatsApp yang membantu dan ramah. Jawab pertanyaan dengan singkat dan jelas dalam bahasa Indonesia.", aiName)

	// Prepare request body
	reqBody := dtos.GroqRequest{
		Model: "meta-llama/llama-4-scout-17b-16e-instruct", // Llama 4 Scout with function calling support
		Messages: []dtos.GroqMessageInternal{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userMessage,
			},
		},
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		customLogger.ErrorWithData("Failed to marshal request", map[string]interface{}{
			"component": "GroqService",
			"function":  "Chat",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to marshal request: %v", err))
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", o.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		customLogger.ErrorWithData("Failed to create request", map[string]interface{}{
			"component": "GroqService",
			"function":  "Chat",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		customLogger.ErrorWithData("Failed to send request", map[string]interface{}{
			"component": "GroqService",
			"function":  "Chat",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to send request: %v", err))
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		customLogger.ErrorWithData("Failed to read response", map[string]interface{}{
			"component": "GroqService",
			"function":  "Chat",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to read response: %v", err))
	}

	customLogger.DebugWithData("Groq API response received", map[string]interface{}{
		"component":   "GroqService",
		"function":    "Chat",
		"status_code": resp.StatusCode,
	})

	if resp.StatusCode != http.StatusOK {
		customLogger.ErrorWithData("Groq API error", map[string]interface{}{
			"component":   "GroqService",
			"function":    "Chat",
			"status_code": resp.StatusCode,
			"body":        string(body),
		})
		return "", gocom.NewError(resp.StatusCode, fmt.Sprintf("Groq API error: %s", string(body)))
	}

	// Parse response
	var groqResp dtos.GroqResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		customLogger.ErrorWithData("Failed to unmarshal response", map[string]interface{}{
			"component": "GroqService",
			"function":  "Chat",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to unmarshal response: %v", err))
	}

	if len(groqResp.Choices) == 0 {
		customLogger.WarnWithData("No response from Groq AI", map[string]interface{}{
			"component": "GroqService",
			"function":  "Chat",
		})
		return "", gocom.NewError(500, "No response from Groq AI")
	}

	customLogger.InfoWithData("Chat success", map[string]interface{}{
		"component":       "GroqService",
		"function":        "Chat",
		"response_length": len(groqResp.Choices[0].Message.Content),
	})
	return groqResp.Choices[0].Message.Content, nil
}

// ChatWithKnowledge sends a message to Groq AI with knowledge base injection and conversation history
func (o *GroqSvcImpl) ChatWithKnowledge(userMessage, clientId, fromNumber string) (string, *gocom.CodedError) {
	customLogger.InfoWithData("ChatWithKnowledge started", map[string]interface{}{
		"component":      "GroqService",
		"function":       "ChatWithKnowledge",
		"client_id":      clientId,
		"from_number":    fromNumber,
		"message_length": len(userMessage),
		"message":        userMessage,
	})

	if o.apiKey == "" {
		customLogger.WarnWithData("GROQ_API_KEY not set", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledge",
		})
		return "", gocom.NewError(500, "GROQ_API_KEY not set in environment")
	}

	// // Get AI name from config
	// aiName := config.Get(constans.GroqAIName, "Asisten")

	// Load knowledge base from cache
	knowledgeBase, codedErr := GetAIKnowledgeCacheSvc().GetKnowledge(clientId)
	if codedErr != nil {
		customLogger.ErrorWithData("Failed to load knowledge base", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledge",
			"error":     codedErr.Message,
			"client_id": clientId,
		})
		// Continue without knowledge base
		knowledgeBase = []aiKnowledgeBase.AiKnowledgeBase{}
	}

	customLogger.InfoWithData("Knowledge base loaded", map[string]interface{}{
		"component":       "GroqService",
		"function":        "ChatWithKnowledge",
		"client_id":       clientId,
		"knowledge_count": len(knowledgeBase),
	})

	if len(knowledgeBase) == 0 {
		customLogger.WarnWithData("Knowledge base is EMPTY", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledge",
			"client_id": clientId,
		})
	}

	// Format knowledge base into system prompt
	categoryList, knowledges := o.formatKnowledgeBase(knowledgeBase)
	customLogger.InfoWithData("Formatted knowledge prompt", map[string]interface{}{
		"component":     "GroqService",
		"function":      "ChatWithKnowledge",
		"knowledges":    knowledges,
		"category_list": categoryList,
	})

	clientInfo := client.GetRepo().GetById(clientId)
	if clientInfo == nil {
		customLogger.ErrorWithData("Failed to get client info", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledge",
			"error":     codedErr.Message,
		})
		return "", gocom.NewError(500, "Failed to get client info")
	}

	greetingTrigger := strings.Join(strings.Split(clientInfo.GreetingTrigger, "|"), ", ")

	// Format category list as bullet points for rejection message
	var categoryBullets strings.Builder
	for _, cat := range categoryList {
		categoryBullets.WriteString(fmt.Sprintf("- %s\n", cat))
	}

	rejectionMessage := fmt.Sprintf(constans.AIGeneralRejectionMessage, clientInfo.Name, categoryBullets.String(), clientInfo.Name)

	// Prepare system prompt using strict template from constants
	systemPrompt := fmt.Sprintf(constans.AIGeneralSystemPromptTemplate,
		clientInfo.Name, greetingTrigger, clientInfo.Greeting, knowledges,
		clientInfo.Name, clientInfo.Greeting, rejectionMessage)

	customLogger.InfoWithData("System prompt prepared", map[string]interface{}{
		"component":         "GroqService",
		"function":          "ChatWithKnowledge",
		"system_prompt":     systemPrompt,
		"client_id":         clientId,
		"from_number":       fromNumber,
		"message":           userMessage,
		"greeting_trigger":  greetingTrigger,
		"greeting":          clientInfo.Greeting,
		"rejection_message": rejectionMessage,
	})

	// Load conversation history for context
	conversationHistory := messageConversation.GetRepo().GetConversationHistory(fromNumber, 10)
	customLogger.InfoWithData("Conversation history loaded", map[string]interface{}{
		"component":     "GroqService",
		"function":      "ChatWithKnowledge",
		"history_count": len(conversationHistory),
		"from_number":   fromNumber,
	})

	// Build messages array with history
	messages := []dtos.GroqMessageInternal{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	// Add conversation history (excluding system messages)
	for _, conv := range conversationHistory {
		if conv.Role == "user" || conv.Role == "assistant" {
			messages = append(messages, dtos.GroqMessageInternal{
				Role:    conv.Role,
				Content: conv.Message,
			})
		}
	}

	// Add current user message
	messages = append(messages, dtos.GroqMessageInternal{
		Role:    "user",
		Content: userMessage,
	})

	customLogger.DebugWithData("Messages context built", map[string]interface{}{
		"component":      "GroqService",
		"function":       "ChatWithKnowledge",
		"total_messages": len(messages),
		"history_count":  len(conversationHistory),
	})

	// Prepare request body
	reqBody := dtos.GroqRequest{
		Model:       "meta-llama/llama-4-scout-17b-16e-instruct",
		Messages:    messages,
		Temperature: 0.2, // Low temperature for strict knowledge-base following
		MaxTokens:   1500,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		customLogger.ErrorWithData("Failed to marshal request", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledge",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to marshal request: %v", err))
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", o.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		customLogger.ErrorWithData("Failed to create request", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledge",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		customLogger.ErrorWithData("Failed to send request", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledge",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to send request: %v", err))
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		customLogger.ErrorWithData("Failed to read response", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledge",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to read response: %v", err))
	}

	customLogger.DebugWithData("Groq API response received", map[string]interface{}{
		"component":   "GroqService",
		"function":    "ChatWithKnowledge",
		"status_code": resp.StatusCode,
	})

	if resp.StatusCode != http.StatusOK {
		customLogger.ErrorWithData("Groq API error", map[string]interface{}{
			"component":   "GroqService",
			"function":    "ChatWithKnowledge",
			"status_code": resp.StatusCode,
			"body":        string(body),
		})
		return "", gocom.NewError(resp.StatusCode, fmt.Sprintf("Groq API error: %s", string(body)))
	}

	// Parse response
	var groqResp dtos.GroqResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		customLogger.ErrorWithData("Failed to unmarshal response", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledge",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to unmarshal response: %v", err))
	}

	if len(groqResp.Choices) == 0 {
		customLogger.WarnWithData("No response from Groq AI", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledge",
		})
		return "", gocom.NewError(500, "No response from Groq AI")
	}

	customLogger.InfoWithData("ChatWithKnowledge success", map[string]interface{}{
		"component":       "GroqService",
		"function":        "ChatWithKnowledge",
		"knowledge_count": len(knowledgeBase),
		"response_length": len(groqResp.Choices[0].Message.Content),
	})
	return groqResp.Choices[0].Message.Content, nil
}

// formatKnowledgeBase formats knowledge base into readable prompt
func (o *GroqSvcImpl) formatKnowledgeBase(knowledge []aiKnowledgeBase.AiKnowledgeBase) ([]string, string) {
	if len(knowledge) == 0 {
		return []string{}, "Belum ada pengetahuan khusus yang ditambahkan. Jawab pertanyaan umum dengan ramah."
	}

	// Group by category
	categories := make(map[string][]aiKnowledgeBase.AiKnowledgeBase)
	for _, k := range knowledge {
		categories[k.Category] = append(categories[k.Category], k)
	}

	var result strings.Builder
	var categoryList []string

	for categoryKey, categoryValue := range categories {

		key := strings.ToUpper(categoryKey)

		categoryList = append(categoryList, key)

		result.WriteString(fmt.Sprintf("\n## %s\n", key))

		for _, item := range categoryValue {
			result.WriteString(fmt.Sprintf("\nQ: %s\nA: %s\n", item.Question, item.Answer))
		}
	}

	return categoryList, result.String()
}

// ChatWithContext sends a message with conversation context
func (o *GroqSvcImpl) ChatWithContext(messages []dtos.GroqMessage) (string, *gocom.CodedError) {
	customLogger.InfoWithData("ChatWithContext started", map[string]interface{}{
		"component":      "GroqService",
		"function":       "ChatWithContext",
		"messages_count": len(messages),
	})

	if o.apiKey == "" {
		customLogger.WarnWithData("GROQ_API_KEY not set", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithContext",
		})
		return "", gocom.NewError(500, "GROQ_API_KEY not set in environment")
	}

	// Convert dtos.GroqMessage to internal GroqMessageInternal
	var groqMessages []dtos.GroqMessageInternal
	for _, msg := range messages {
		groqMessages = append(groqMessages, dtos.GroqMessageInternal{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Prepare request body
	reqBody := dtos.GroqRequest{
		Model:       "meta-llama/llama-4-scout-17b-16e-instruct", // Llama 4 Scout with function calling support
		Messages:    groqMessages,
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		customLogger.ErrorWithData("Failed to marshal request", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithContext",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to marshal request: %v", err))
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", o.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		customLogger.ErrorWithData("Failed to create request", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithContext",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		customLogger.ErrorWithData("Failed to send request", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithContext",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to send request: %v", err))
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		customLogger.ErrorWithData("Failed to read response", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithContext",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to read response: %v", err))
	}

	customLogger.DebugWithData("Groq API response received", map[string]interface{}{
		"component":   "GroqService",
		"function":    "ChatWithContext",
		"status_code": resp.StatusCode,
	})

	if resp.StatusCode != http.StatusOK {
		customLogger.ErrorWithData("Groq API error", map[string]interface{}{
			"component":   "GroqService",
			"function":    "ChatWithContext",
			"status_code": resp.StatusCode,
			"body":        string(body),
		})
		return "", gocom.NewError(resp.StatusCode, fmt.Sprintf("Groq API error: %s", string(body)))
	}

	// Parse response
	var groqResp dtos.GroqResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		customLogger.ErrorWithData("Failed to unmarshal response", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithContext",
			"error":     err.Error(),
		})
		return "", gocom.NewError(500, fmt.Sprintf("Failed to unmarshal response: %v", err))
	}

	if len(groqResp.Choices) == 0 {
		customLogger.WarnWithData("No response from Groq AI", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithContext",
		})
		return "", gocom.NewError(500, "No response from Groq AI")
	}

	customLogger.InfoWithData("ChatWithContext success", map[string]interface{}{
		"component":       "GroqService",
		"function":        "ChatWithContext",
		"response_length": len(groqResp.Choices[0].Message.Content),
	})
	return groqResp.Choices[0].Message.Content, nil
}

// SendConversationMessage - Get AI response and send to WhatsApp
func (o *GroqSvcImpl) SendConversationMessage(sessionID, from, userMessage, clientId string) (string, string, *gocom.CodedError) {
	customLogger.InfoWithData("SendConversationMessage started", map[string]interface{}{
		"component":  "GroqService",
		"function":   "SendConversationMessage",
		"from":       from,
		"session_id": sessionID,
		"client_id":  clientId,
	})

	// Save user message to conversation history
	userConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         from,
		Role:               "user",
		Message:            userMessage,
		Status:             "received",
		ConversationStatus: "active",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}
	messageConversation.GetRepo().Create(userConv)

	// Get AI response with knowledge base
	aiResponse, codedErr := o.ChatWithKnowledge(userMessage, clientId, from)
	if codedErr != nil {
		// Save failed AI response
		assistantConv := &messageConversation.MessageConversation{
			SessionID:    sessionID,
			ClientId:     clientId,
			FromNumber:   from,
			Role:         "assistant",
			Message:      "",
			Status:       "failed",
			ErrorMessage: codedErr.Message,
			Model:        "meta-llama/llama-4-scout-17b-16e-instruct",
		}
		messageConversation.GetRepo().Create(assistantConv)
		return "", "", codedErr
	}

	// Send text message to WhatsApp using WASendSvc with message_logs creation
	// Note: Pass empty senderIdentifier since this is called from API (will use global config)
	waMessageId, messageLogId, sendErr := GetWASendSvc().SendTextMessageWithLog(from, aiResponse, clientId, "")

	// Detect if AI wants to close conversation
	conversationStatus := "active"
	if o.shouldCloseConversation(aiResponse) {
		conversationStatus = "pending_close" // Wait for user confirmation
	}

	// Save AI response to conversation history
	assistantConv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientId,
		FromNumber:         from,
		ToNumber:           from, // Reply ke nomor yang sama
		Role:               "assistant",
		Message:            aiResponse,
		MessageLogId:       messageLogId, // Link to message_logs!
		Status:             "sent",
		ConversationStatus: conversationStatus,
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
	}

	if sendErr != nil {
		assistantConv.Status = "failed"
		assistantConv.ErrorMessage = sendErr.Message
		messageConversation.GetRepo().Create(assistantConv)
		return "", aiResponse, sendErr
	}

	messageConversation.GetRepo().Create(assistantConv)

	customLogger.InfoWithData("SendConversationMessage success", map[string]interface{}{
		"component":      "GroqService",
		"function":       "SendConversationMessage",
		"wa_message_id":  waMessageId,
		"message_log_id": messageLogId,
		"from":           from,
	})
	return waMessageId, aiResponse, nil
}

// shouldCloseConversation checks if AI response indicates conversation should be closed
func (o *GroqSvcImpl) shouldCloseConversation(aiResponse string) bool {
	lowerResponse := strings.ToLower(aiResponse)
	// Detect phrases indicating AI wants to close conversation (case insensitive)
	closingPhrases := []string{
		"bisa kami close",
		"bisa kami tutup",
		"dapat kami close",
		"dapat kami tutup",
		"bisa ditutup",
		"dapat ditutup",
		"close conversation",
		"tutup percakapan",
		"sudah selesai kah",
		"sudah selesaikah",
		"apakah sudah selesai",
		"mau di close",
		"mau ditutup",
		"ingin menutup",
		"sudah cukup",
		"ada lagi yang bisa",
		"ada yang bisa kami bantu lagi",
	}

	for _, phrase := range closingPhrases {
		if strings.Contains(lowerResponse, phrase) {
			return true
		}
	}
	return false
}

// ChatWithKnowledgeAndFunctions uses function calling to detect and handle PPOB bill inquiry
func (o *GroqSvcImpl) ChatWithKnowledgeAndFunctions(userMessage, clientId, from string) (string, *dtos.PPOBInquiryResp, *gocom.CodedError) {
	customLogger.InfoWithData("ChatWithKnowledgeAndFunctions started", map[string]interface{}{
		"component": "GroqService",
		"function":  "ChatWithKnowledgeAndFunctions",
		"client_id": clientId,
		"from":      from,
		"message":   userMessage,
	})

	if o.apiKey == "" {
		return "", nil, gocom.NewError(500, "GROQ_API_KEY not set in environment")
	}

	// Get AI name from config
	aiName := config.Get(constans.GroqAIName, "Asisten")

	// Load knowledge base from cache
	knowledgeBase, _ := GetAIKnowledgeCacheSvc().GetKnowledge(clientId)
	categoryList, knowledges := o.formatKnowledgeBase(knowledgeBase)
	customLogger.DebugWithData("Formatted knowledge prompt", map[string]interface{}{
		"component":     "GroqService",
		"function":      "ChatWithKnowledgeAndFunctions",
		"knowledges":    knowledges,
		"category_list": categoryList,
	})

	// Define tools (functions) for AI to call
	tools := []dtos.GroqTool{
		{
			Type: "function",
			Function: dtos.GroqFunctionDefinition{
				Name:        "check_pdam_bill",
				Description: "Cek tagihan air PDAM berdasarkan nomor pelanggan. Gunakan fungsi ini ketika user memberikan nomor pelanggan atau meminta cek tagihan.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"customer_number": map[string]interface{}{
							"type":        "string",
							"description": "Nomor pelanggan PDAM yang ingin dicek tagihannya. Bisa berupa angka atau kombinasi huruf dan angka (contoh: A532159, 1234567890)",
						},
					},
					"required": []string{"customer_number"},
				},
			},
		},
		{
			Type: "function",
			Function: dtos.GroqFunctionDefinition{
				Name:        "escalate_to_cs",
				Description: "Escalate komplain atau pertanyaan ke customer service manusia. Gunakan fungsi ini HANYA ketika pertanyaan user terkait PDAM tapi tidak ada jawabannya di PENGETAHUAN, atau user explicitly meminta bicara dengan CS.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"reason": map[string]interface{}{
							"type":        "string",
							"description": "Alasan escalation. Contoh: 'Komplain gangguan air tidak ada di knowledge base', 'User request bicara dengan CS', 'Pertanyaan terlalu spesifik'",
						},
						"category": map[string]interface{}{
							"type":        "string",
							"description": "Kategori komplain: 'pengaduan', 'tagihan', 'layanan', 'administrasi', atau 'umum'",
							"enum":        []string{"pengaduan", "tagihan", "layanan", "administrasi", "umum"},
						},
						"user_message": map[string]interface{}{
							"type":        "string",
							"description": "Pesan asli dari user yang memicu escalation",
						},
					},
					"required": []string{"reason", "category", "user_message"},
				},
			},
		},
	}

	// Prepare system prompt from constants template with CRITICAL WARNING about markdown links
	systemPrompt := fmt.Sprintf(
		constans.AISystemPromptTemplate,
		aiName,
		knowledges,
		constans.AIGreetingMessage,
		constans.AIEscalationMessage,
		constans.AINoInfoEscalationMessage,
		constans.AIRejectionMessage,
	)

	// Prepare request
	reqBody := dtos.GroqRequest{
		Model: "meta-llama/llama-4-scout-17b-16e-instruct",
		Messages: []dtos.GroqMessageInternal{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userMessage,
			},
		},
		Temperature: 0.3,
		MaxTokens:   8000,
		Tools:       tools,
		ToolChoice:  "auto",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, gocom.NewError(500, fmt.Sprintf("Failed to marshal request: %v", err))
	}

	customLogger.DebugWithData("Request payload prepared", map[string]interface{}{
		"component":    "GroqService",
		"function":     "ChatWithKnowledgeAndFunctions",
		"payload_size": len(jsonData),
	})

	// Create HTTP request
	req, err := http.NewRequest("POST", o.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, gocom.NewError(500, fmt.Sprintf("Failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	// Send request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, gocom.NewError(500, fmt.Sprintf("Failed to send request: %v", err))
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, gocom.NewError(500, fmt.Sprintf("Failed to read response: %v", err))
	}

	customLogger.DebugWithData("Groq API response received", map[string]interface{}{
		"component":   "GroqService",
		"function":    "ChatWithKnowledgeAndFunctions",
		"status_code": resp.StatusCode,
		"body_length": len(body),
	})

	if resp.StatusCode != http.StatusOK {
		customLogger.ErrorWithData("Groq API error", map[string]interface{}{
			"component":   "GroqService",
			"function":    "ChatWithKnowledgeAndFunctions",
			"status_code": resp.StatusCode,
			"body":        string(body),
		})

		// If Groq returns tool_use_failed, it means AI got stuck - likely due to markdown syntax in knowledge base
		// DO NOT return partial failed_generation as it will be truncated and incomplete
		// Instead, return a generic error message
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}

		if err := json.Unmarshal(body, &errorResp); err == nil {
			if errorResp.Error.Code == "tool_use_failed" {
				customLogger.WarnWithData("tool_use_failed detected", map[string]interface{}{
					"component": "GroqService",
					"function":  "ChatWithKnowledgeAndFunctions",
					"note":      "knowledge base may contain problematic syntax (markdown links, etc)",
				})
				return "Maaf, terjadi kesalahan saat memproses permintaan Anda. Silakan coba lagi atau hubungi customer service kami.", nil, nil
			}
		}

		return "", nil, gocom.NewError(resp.StatusCode, fmt.Sprintf("Groq API error: %s", string(body)))
	}

	// Parse response
	var groqResp dtos.GroqResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		// If parsing fails, try to extract text response directly
		customLogger.WarnWithData("Failed to unmarshal response, trying to extract text", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledgeAndFunctions",
			"error":     err.Error(),
		})

		// Try alternative parsing for text-only response
		var simpleResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}

		if err2 := json.Unmarshal(body, &simpleResp); err2 == nil && len(simpleResp.Choices) > 0 {
			customLogger.InfoWithData("Successfully extracted text response", map[string]interface{}{
				"component": "GroqService",
				"function":  "ChatWithKnowledgeAndFunctions",
			})
			return simpleResp.Choices[0].Message.Content, nil, nil
		}

		return "", nil, gocom.NewError(500, fmt.Sprintf("Failed to unmarshal response: %v", err))
	}

	if len(groqResp.Choices) == 0 {
		return "", nil, gocom.NewError(500, "No response from Groq AI")
	}

	choice := groqResp.Choices[0]

	// Log finish reason to detect truncation
	customLogger.DebugWithData("Response finish reason", map[string]interface{}{
		"component":     "GroqService",
		"function":      "ChatWithKnowledgeAndFunctions",
		"finish_reason": choice.FinishReason,
	})
	if choice.FinishReason == "length" {
		customLogger.WarnWithData("Response truncated due to max_tokens limit", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledgeAndFunctions",
		})
	}

	// Check if AI wants to call a function
	if len(choice.Message.ToolCalls) > 0 {
		toolCall := choice.Message.ToolCalls[0]
		customLogger.InfoWithData("AI calling function", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledgeAndFunctions",
			"tool_name": toolCall.Function.Name,
			"tool_args": toolCall.Function.Arguments,
		})

		if toolCall.Function.Name == "check_pdam_bill" {
			// Parse function arguments
			var args struct {
				CustomerNumber string `json:"customer_number"`
			}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				customLogger.ErrorWithData("Failed to parse function arguments", map[string]interface{}{
					"component": "GroqService",
					"function":  "ChatWithKnowledgeAndFunctions",
					"error":     err.Error(),
				})
				return "Maaf, terjadi kesalahan saat memproses nomor pelanggan.", nil, nil
			}

			customLogger.InfoWithData("Checking PDAM bill", map[string]interface{}{
				"component":       "GroqService",
				"function":        "ChatWithKnowledgeAndFunctions",
				"customer_number": args.CustomerNumber,
			})

			// Call PPOB Inquiry API
			inquiryResp, codedErr := GetPPOBSvc().Inquiry(args.CustomerNumber)
			if codedErr != nil {
				customLogger.ErrorWithData("PPOB Inquiry failed", map[string]interface{}{
					"component":       "GroqService",
					"function":        "ChatWithKnowledgeAndFunctions",
					"error":           codedErr.Message,
					"customer_number": args.CustomerNumber,
				})

				// Parse error message format: [ERROR_CODE] message
				var errorMsg string
				if strings.HasPrefix(codedErr.Message, "[7011]") {
					// Bill already paid
					errorMsg = fmt.Sprintf(`Halo! 👋

Terima kasih telah melakukan pengecekan tagihan untuk Nomor Pelanggan *%s*.

✅ *Tagihan Sudah Lunas*

Kami menemukan bahwa tagihan untuk nomor pelanggan ini sudah dibayar.

📞 Jika Anda memerlukan informasi lebih lanjut atau ada pertanyaan terkait pembayaran, silakan hubungi:

*Call Center PDAM Kabupaten Tangerang*
☎️ (021) 5951234

Terima kasih! 🙏`, args.CustomerNumber)
				} else if strings.HasPrefix(codedErr.Message, "[7010]") {
					// Bill not found
					errorMsg = fmt.Sprintf(`Maaf, kami tidak dapat menemukan tagihan untuk Nomor Pelanggan *%s*.

❌ *Nomor Pelanggan Tidak Ditemukan*

Mohon periksa kembali nomor pelanggan Anda dan pastikan sudah benar.

📞 Jika Anda yakin nomor pelanggan sudah benar, silakan hubungi:

*Call Center PDAM Kabupaten Tangerang*
☎️ (021) 5951234

Terima kasih! 🙏`, args.CustomerNumber)
				} else if strings.Contains(codedErr.Message, "tidak ditemukan") || strings.Contains(codedErr.Message, "not found") {
					// Generic not found
					errorMsg = fmt.Sprintf("Maaf, tidak dapat mengecek tagihan untuk nomor pelanggan %s.\n\n❌ Nomor pelanggan tidak ditemukan\n\nPastikan nomor pelanggan Anda benar dan coba lagi.", args.CustomerNumber)
				} else {
					// Other errors
					errorMsg = fmt.Sprintf("Maaf, tidak dapat mengecek tagihan untuk nomor pelanggan %s.\n\n%s\n\nSilakan coba lagi nanti atau hubungi Call Center PDAM Kabupaten Tangerang di (021) 5951234.", args.CustomerNumber, codedErr.Message)
				}

				return errorMsg, nil, nil
			}

			// Format success response
			responseMsg := o.formatBillInquiryResponse(inquiryResp)
			customLogger.InfoWithData("Bill inquiry successful", map[string]interface{}{
				"component": "GroqService",
				"function":  "ChatWithKnowledgeAndFunctions",
				"tx_id": func() string {
					if inquiryResp.TxID != nil {
						return *inquiryResp.TxID
					}
					return ""
				}(),
			})

			return responseMsg, inquiryResp, nil
		}
	}

	// No function call, return normal AI response
	customLogger.InfoWithData("No function call, returning normal response", map[string]interface{}{
		"component": "GroqService",
		"function":  "ChatWithKnowledgeAndFunctions",
	})
	return choice.Message.Content, nil, nil
}

// EscalationInfo holds escalation data when AI escalates to CS
type EscalationInfo struct {
	Reason      string
	Category    string
	UserMessage string
}

// ChatWithEscalation handles AI conversation with CS escalation support
// This is separate from ChatWithKnowledgeAndFunctions to avoid breaking existing PPOB flow
func (o *GroqSvcImpl) ChatWithEscalation(userMessage, clientId, from string) (string, *EscalationInfo, *gocom.CodedError) {
	customLogger.InfoWithData("ChatWithEscalation started", map[string]interface{}{
		"component": "GroqService",
		"function":  "ChatWithEscalation",
		"client_id": clientId,
		"from":      from,
		"message":   userMessage,
	})

	if o.apiKey == "" {
		return "", nil, gocom.NewError(500, "GROQ_API_KEY not set in environment")
	}

	// Get AI name from config
	aiName := config.Get(constans.GroqAIName, "Asisten")

	// Load knowledge base from cache
	knowledgeBase, _ := GetAIKnowledgeCacheSvc().GetKnowledge(clientId)
	categoryList, knowledges := o.formatKnowledgeBase(knowledgeBase)
	customLogger.DebugWithData("Formatted knowledge prompt", map[string]interface{}{
		"component":     "GroqService",
		"function":      "ChatWithEscalation",
		"knowledges":    knowledges,
		"category_list": categoryList,
	})

	// Define tools (functions) for AI to call - ONLY escalation, no PPOB
	tools := []dtos.GroqTool{
		{
			Type: "function",
			Function: dtos.GroqFunctionDefinition{
				Name:        "escalate_to_cs",
				Description: "Escalate komplain atau pertanyaan ke customer service manusia. Gunakan fungsi ini HANYA ketika pertanyaan user terkait PDAM tapi tidak ada jawabannya di PENGETAHUAN, atau user explicitly meminta bicara dengan CS.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"reason": map[string]interface{}{
							"type":        "string",
							"description": "Alasan escalation. Contoh: 'Komplain gangguan air tidak ada di knowledge base', 'User request bicara dengan CS', 'Pertanyaan terlalu spesifik'",
						},
						"category": map[string]interface{}{
							"type":        "string",
							"description": "Kategori komplain: 'pengaduan', 'tagihan', 'layanan', 'administrasi', atau 'umum'",
							"enum":        []string{"pengaduan", "tagihan", "layanan", "administrasi", "umum"},
						},
						"user_message": map[string]interface{}{
							"type":        "string",
							"description": "Pesan asli dari user yang memicu escalation",
						},
					},
					"required": []string{"reason", "category", "user_message"},
				},
			},
		},
	}

	// Prepare system prompt
	systemPrompt := fmt.Sprintf(
		constans.AISystemPromptTemplate,
		aiName,
		knowledges,
		constans.AIGreetingMessage,
		constans.AIEscalationMessage,
		constans.AINoInfoEscalationMessage,
		constans.AIRejectionMessage,
	)

	// Prepare request
	reqBody := dtos.GroqRequest{
		Model: "meta-llama/llama-4-scout-17b-16e-instruct",
		Messages: []dtos.GroqMessageInternal{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userMessage,
			},
		},
		Temperature: 0.3,
		MaxTokens:   8000,
		Tools:       tools,
		ToolChoice:  "auto",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, gocom.NewError(500, fmt.Sprintf("Failed to marshal request: %v", err))
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", o.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, gocom.NewError(500, fmt.Sprintf("Failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, gocom.NewError(500, fmt.Sprintf("Failed to send request: %v", err))
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, gocom.NewError(500, fmt.Sprintf("Failed to read response: %v", err))
	}

	if resp.StatusCode != http.StatusOK {
		customLogger.ErrorWithData("Groq API error", map[string]interface{}{
			"component":   "GroqService",
			"function":    "ChatWithEscalation",
			"status_code": resp.StatusCode,
			"body":        string(body),
		})
		return "Maaf, terjadi kesalahan. Silakan hubungi customer service kami di (021) 5951234.", nil, nil
	}

	// Parse response
	var groqResp dtos.GroqResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		customLogger.ErrorWithData("Failed to unmarshal response", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithEscalation",
			"error":     err.Error(),
		})
		return "Maaf, terjadi kesalahan. Silakan hubungi customer service kami di (021) 5951234.", nil, nil
	}

	if len(groqResp.Choices) == 0 {
		return "", nil, gocom.NewError(500, "No response from Groq AI")
	}

	choice := groqResp.Choices[0]

	// Check if AI wants to escalate to CS
	if len(choice.Message.ToolCalls) > 0 {
		toolCall := choice.Message.ToolCalls[0]
		customLogger.InfoWithData("AI calling function", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithEscalation",
			"tool_name": toolCall.Function.Name,
		})

		if toolCall.Function.Name == "escalate_to_cs" {
			// Parse escalation arguments
			var args struct {
				Reason      string `json:"reason"`
				Category    string `json:"category"`
				UserMessage string `json:"user_message"`
			}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				customLogger.ErrorWithData("Failed to parse escalation arguments", map[string]interface{}{
					"component": "GroqService",
					"function":  "ChatWithEscalation",
					"error":     err.Error(),
				})
				return "Maaf, terjadi kesalahan. Silakan hubungi customer service kami di (021) 5951234.", nil, nil
			}

			customLogger.InfoWithData("AI escalating to CS", map[string]interface{}{
				"component": "GroqService",
				"function":  "ChatWithEscalation",
				"reason":    args.Reason,
				"category":  args.Category,
			})

			// Return escalation info
			escalationInfo := &EscalationInfo{
				Reason:      args.Reason,
				Category:    args.Category,
				UserMessage: args.UserMessage,
			}

			// Return user-facing message
			escalationMsg := "Baik, saya akan menghubungkan Anda dengan customer service kami. Mohon tunggu sebentar..."

			return escalationMsg, escalationInfo, nil
		}
	}

	// No escalation, return normal AI response
	customLogger.InfoWithData("No escalation, returning normal response", map[string]interface{}{
		"component": "GroqService",
		"function":  "ChatWithEscalation",
	})
	return choice.Message.Content, nil, nil
}

// formatBillInquiryResponse formats PPOB inquiry response for WhatsApp
func (o *GroqSvcImpl) formatBillInquiryResponse(resp *dtos.PPOBInquiryResp) string {
	var msg strings.Builder

	msg.WriteString("✅ *INFORMASI TAGIHAN PDAM*\n\n")

	// Safely handle pointer fields
	if resp.Detail != nil {
		if resp.Detail.CustomerName != nil {
			msg.WriteString(fmt.Sprintf("👤 Nama: %s\n", *resp.Detail.CustomerName))
		}
		if resp.Detail.CustomerAddress != nil {
			msg.WriteString(fmt.Sprintf("📍 Alamat: %s\n", *resp.Detail.CustomerAddress))
		}
		msg.WriteString(fmt.Sprintf("📦 Produk: %s\n", resp.Detail.ProductName))
	}

	if resp.BillID != nil {
		msg.WriteString(fmt.Sprintf("🏷️ Nomor Pelanggan: %s\n\n", *resp.BillID))
	}

	// Detail billing
	if resp.Detail != nil && resp.Detail.DetailBilling != nil && len(*resp.Detail.DetailBilling) > 0 {
		msg.WriteString("📋 *Detail Tagihan:*\n")
		for i, detail := range *resp.Detail.DetailBilling {
			period := ""
			if detail.Period != nil {
				period = *detail.Period
			}
			msg.WriteString(fmt.Sprintf("\n*Periode %d: %s*\n", i+1, period))

			if detail.StandMeter != nil {
				msg.WriteString(fmt.Sprintf("   💧 Stand Meter: %s\n", *detail.StandMeter))
			}

			if detail.BillAmount != nil {
				msg.WriteString(fmt.Sprintf("   💵 Tagihan: Rp %s\n", formatRupiahFloat(*detail.BillAmount)))
			}

			if detail.Fine != nil && *detail.Fine > 0 {
				msg.WriteString(fmt.Sprintf("   ⚠️ Denda: Rp %s\n", formatRupiahFloat(*detail.Fine)))
			}

			// Calculate admin total
			adminTotal := float64(0)
			if detail.AdminMerchant != nil {
				adminTotal += *detail.AdminMerchant
			}
			if detail.AdminProvider != nil {
				adminTotal += *detail.AdminProvider
			}
			if adminTotal > 0 {
				msg.WriteString(fmt.Sprintf("   🏦 Admin: Rp %s\n", formatRupiahFloat(adminTotal)))
			}

			if detail.Total != nil {
				msg.WriteString(fmt.Sprintf("   ✅ Total: Rp %s\n", formatRupiahFloat(*detail.Total)))
			}
		}
	}

	if resp.TotalAmount != nil {
		msg.WriteString(fmt.Sprintf("\n💰 *TOTAL TAGIHAN: Rp %s*\n", formatRupiahFloat(*resp.TotalAmount)))
	}

	if resp.Detail != nil && resp.Detail.Period != nil {
		msg.WriteString(fmt.Sprintf("📅 Periode: %s\n", *resp.Detail.Period))
	}

	return msg.String()
}

// formatRupiahFloat formats float64 to Rupiah format (e.g., 150000.50 -> "150.000")
func formatRupiahFloat(amount float64) string {
	// Convert to int (remove decimals)
	amountInt := int(amount)
	amountStr := fmt.Sprintf("%d", amountInt)
	var result strings.Builder

	for i, digit := range amountStr {
		if i > 0 && (len(amountStr)-i)%3 == 0 {
			result.WriteString(".")
		}
		result.WriteRune(digit)
	}

	return result.String()
}

// ChatWithKnowledgeAndPaymentFunction uses function calling to detect and handle PPOB bill payment
func (o *GroqSvcImpl) ChatWithKnowledgeAndPaymentFunction(userMessage, clientId, from string, inquiryResp *dtos.PPOBInquiryResp) (string, *dtos.PPOBPaymentResp, *gocom.CodedError) {
	customLogger.InfoWithData("ChatWithKnowledgeAndPaymentFunction started", map[string]interface{}{
		"component": "GroqService",
		"function":  "ChatWithKnowledgeAndPaymentFunction",
		"client_id": clientId,
		"from":      from,
		"message":   userMessage,
	})

	if o.apiKey == "" {
		return "", nil, gocom.NewError(500, "GROQ_API_KEY not set in environment")
	}

	// Validate inquiry response
	if inquiryResp == nil {
		return "Maaf, tidak ada tagihan yang ditemukan. Silakan lakukan pengecekan tagihan terlebih dahulu.", nil, nil
	}

	// Get AI name from config
	aiName := config.Get(constans.GroqAIName, "Asisten")

	// Load knowledge base from cache
	knowledgeBase, _ := GetAIKnowledgeCacheSvc().GetKnowledge(clientId)
	categoryList, knowledges := o.formatKnowledgeBase(knowledgeBase)
	customLogger.DebugWithData("Formatted knowledge prompt", map[string]interface{}{
		"component":     "GroqService",
		"function":      "ChatWithKnowledgeAndPaymentFunction",
		"knowledges":    knowledges,
		"category_list": categoryList,
	})

	// Define tools (functions) for AI to call
	tools := []dtos.GroqTool{
		{
			Type: "function",
			Function: dtos.GroqFunctionDefinition{
				Name:        "pay_pdam_bill",
				Description: "Bayar tagihan air PDAM. Gunakan fungsi ini ketika user konfirmasi untuk membayar atau memproses pembayaran tagihan.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"bank_code": map[string]interface{}{
							"type":        "string",
							"description": "Kode bank untuk pembayaran (optional). Contoh: BCA, BNI, MANDIRI",
						},
					},
					"required": []string{},
				},
			},
		},
	}

	// Prepare system prompt with payment context
	billInfo := ""
	if inquiryResp.BillID != nil {
		billInfo = fmt.Sprintf("Nomor Pelanggan: %s", *inquiryResp.BillID)
	}
	if inquiryResp.TotalAmount != nil {
		billInfo += fmt.Sprintf("\nTotal Tagihan: Rp %s", formatRupiahFloat(*inquiryResp.TotalAmount))
	}

	systemPrompt := fmt.Sprintf(`%s

KONTEKS PEMBAYARAN:
%s

Gunakan fungsi pay_pdam_bill ketika user konfirmasi untuk membayar tagihan.`,
		fmt.Sprintf(
			constans.AISystemPromptTemplate,
			aiName,
			knowledges,
			constans.AIGreetingMessage,
			constans.AIEscalationMessage,
			constans.AINoInfoEscalationMessage,
			constans.AIRejectionMessage,
		),
		billInfo)

	// Prepare request
	reqBody := dtos.GroqRequest{
		Model: "meta-llama/llama-4-scout-17b-16e-instruct",
		Messages: []dtos.GroqMessageInternal{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userMessage,
			},
		},
		Temperature: 0.3,
		MaxTokens:   8000,
		Tools:       tools,
		ToolChoice:  "auto",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, gocom.NewError(500, fmt.Sprintf("Failed to marshal request: %v", err))
	}

	customLogger.DebugWithData("Request payload prepared", map[string]interface{}{
		"component":    "GroqService",
		"function":     "ChatWithKnowledgeAndPaymentFunction",
		"payload_size": len(jsonData),
	})

	// Create HTTP request
	req, err := http.NewRequest("POST", o.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, gocom.NewError(500, fmt.Sprintf("Failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	// Send request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, gocom.NewError(500, fmt.Sprintf("Failed to send request: %v", err))
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, gocom.NewError(500, fmt.Sprintf("Failed to read response: %v", err))
	}

	customLogger.DebugWithData("Response received", map[string]interface{}{
		"component":   "GroqService",
		"function":    "ChatWithKnowledgeAndPaymentFunction",
		"status_code": resp.StatusCode,
		"body_length": len(body),
	})

	if resp.StatusCode != http.StatusOK {
		customLogger.ErrorWithData("Groq API error", map[string]interface{}{
			"component":   "GroqService",
			"function":    "ChatWithKnowledgeAndPaymentFunction",
			"status_code": resp.StatusCode,
			"body":        string(body),
		})

		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}

		if err := json.Unmarshal(body, &errorResp); err == nil {
			if errorResp.Error.Code == "tool_use_failed" {
				customLogger.WarnWithData("tool_use_failed detected", map[string]interface{}{
					"component": "GroqService",
					"function":  "ChatWithKnowledgeAndPaymentFunction",
				})
				return "Maaf, terjadi kesalahan saat memproses pembayaran Anda. Silakan coba lagi atau hubungi customer service kami.", nil, nil
			}
		}

		return "", nil, gocom.NewError(resp.StatusCode, fmt.Sprintf("Groq API error: %s", string(body)))
	}

	// Parse response
	var groqResp dtos.GroqResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		customLogger.WarnWithData("Failed to unmarshal response", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledgeAndPaymentFunction",
			"error":     err.Error(),
		})

		var simpleResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}

		if err2 := json.Unmarshal(body, &simpleResp); err2 == nil && len(simpleResp.Choices) > 0 {
			customLogger.InfoWithData("Successfully extracted text response", map[string]interface{}{
				"component": "GroqService",
				"function":  "ChatWithKnowledgeAndPaymentFunction",
			})
			return simpleResp.Choices[0].Message.Content, nil, nil
		}

		return "", nil, gocom.NewError(500, fmt.Sprintf("Failed to unmarshal response: %v", err))
	}

	if len(groqResp.Choices) == 0 {
		return "", nil, gocom.NewError(500, "No response from Groq AI")
	}

	choice := groqResp.Choices[0]

	customLogger.DebugWithData("Response finish reason", map[string]interface{}{
		"component":     "GroqService",
		"function":      "ChatWithKnowledgeAndPaymentFunction",
		"finish_reason": choice.FinishReason,
	})
	if choice.FinishReason == "length" {
		customLogger.WarnWithData("Response truncated", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledgeAndPaymentFunction",
			"reason":    "max_tokens_limit",
		})
	}

	// Check if AI wants to call a function
	if len(choice.Message.ToolCalls) > 0 {
		toolCall := choice.Message.ToolCalls[0]
		customLogger.InfoWithData("AI calling function", map[string]interface{}{
			"component": "GroqService",
			"function":  "ChatWithKnowledgeAndPaymentFunction",
			"tool_name": toolCall.Function.Name,
			"tool_args": toolCall.Function.Arguments,
		})

		if toolCall.Function.Name == "pay_pdam_bill" {
			// Parse function arguments
			var args struct {
				BankCode *string `json:"bank_code,omitempty"`
			}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				customLogger.ErrorWithData("Failed to parse function arguments", map[string]interface{}{
					"component": "GroqService",
					"function":  "ChatWithKnowledgeAndPaymentFunction",
					"error":     err.Error(),
				})
				return "Maaf, terjadi kesalahan saat memproses pembayaran.", nil, nil
			}

			customLogger.InfoWithData("Processing payment for inquiry", map[string]interface{}{
				"component": "GroqService",
				"function":  "ChatWithKnowledgeAndPaymentFunction",
				"bank_code": func() string {
					if args.BankCode != nil {
						return *args.BankCode
					}
					return "default"
				}(),
			})

			// Call PPOB Payment API
			paymentResp, codedErr := GetPPOBSvc().Payment(inquiryResp, args.BankCode)
			if codedErr != nil {
				customLogger.ErrorWithData("PPOB Payment failed", map[string]interface{}{
					"component": "GroqService",
					"function":  "ChatWithKnowledgeAndPaymentFunction",
					"error":     codedErr.Message,
				})

				// Parse error message
				var errorMsg string
				if strings.Contains(codedErr.Message, "insufficient") || strings.Contains(codedErr.Message, "saldo") {
					errorMsg = fmt.Sprintf(`Maaf, pembayaran tidak dapat diproses. ❌

⚠️ *Saldo tidak mencukupi*

Silakan hubungi customer service kami untuk informasi lebih lanjut.

📞 *Call Center PDAM Kabupaten Tangerang*
☎️ (021) 5951234`)
				} else if strings.Contains(codedErr.Message, "already paid") || strings.Contains(codedErr.Message, "sudah dibayar") {
					errorMsg = fmt.Sprintf(`Maaf, pembayaran tidak dapat diproses. ❌

✅ *Tagihan sudah dibayar sebelumnya*

Terima kasih! 🙏`)
				} else {
					errorMsg = fmt.Sprintf("Maaf, tidak dapat memproses pembayaran.\n\n%s\n\nSilakan coba lagi nanti atau hubungi Call Center PDAM Kabupaten Tangerang di (021) 5951234.", codedErr.Message)
				}

				return errorMsg, nil, nil
			}

			// Payment response details
			paymentStatus := "UNKNOWN"
			if paymentResp.Status != nil {
				paymentStatus = *paymentResp.Status
			}

			paymentLogData := map[string]interface{}{
				"component":   "GroqService",
				"function":    "ChatWithKnowledgeAndPaymentFunction",
				"status_code": paymentStatus,
			}

			if paymentResp.TxID != nil {
				paymentLogData["tx_id"] = *paymentResp.TxID
			}
			if paymentResp.TotalAmount != nil {
				paymentLogData["total_amount"] = *paymentResp.TotalAmount
			}

			// Status interpretation
			switch paymentStatus {
			case "0":
				paymentLogData["status_text"] = "PAYMENT_SUCCESS"
			case "1":
				paymentLogData["status_text"] = "PENDING_VA_CREATED"
			case "2":
				paymentLogData["status_text"] = "FAILED"
			case "3":
				paymentLogData["status_text"] = "INQUIRY"
			default:
				paymentLogData["status_text"] = "UNKNOWN"
			}

			if paymentResp.Detail != nil && paymentResp.Detail.VaNo != "" {
				paymentLogData["va_number"] = paymentResp.Detail.VaNo
				paymentLogData["bank_code"] = paymentResp.Detail.BankCode
				paymentLogData["valid_until"] = paymentResp.Detail.ValidUntil
				customLogger.InfoWithData("Payment response received with VA", paymentLogData)
			} else {
				customLogger.WarnWithData("VA NUMBER IS EMPTY in payment response", paymentLogData)
			}

			// Format response based on status
			responseMsg := o.formatPaymentResponse(paymentResp)
			customLogger.InfoWithData("Payment response formatted", map[string]interface{}{
				"component": "GroqService",
				"function":  "ChatWithKnowledgeAndPaymentFunction",
				"status":    paymentStatus,
			})

			return responseMsg, paymentResp, nil
		}
	}

	// No function call, return normal AI response
	customLogger.InfoWithData("No function call, returning normal response", map[string]interface{}{
		"component": "GroqService",
		"function":  "ChatWithKnowledgeAndPaymentFunction",
	})
	return choice.Message.Content, nil, nil
}

// formatPaymentResponse formats PPOB payment response for WhatsApp
func (o *GroqSvcImpl) formatPaymentResponse(resp *dtos.PPOBPaymentResp) string {
	var msg strings.Builder

	// Check payment status
	status := ""
	if resp.Status != nil {
		status = *resp.Status
	}

	// Status mapping (CORRECT):
	// "0" = SUCCESS (payment completed/lunas)
	// "1" = PENDING (VA created, menunggu pembayaran)
	// "2" = FAILED (pembayaran gagal)
	// "3" = INQUIRY (hanya inquiry)
	isPending := status == "1"
	isSuccess := status == "0"
	isFailed := status == "2"
	_ = status == "3" // isInquiry - for future use

	if isSuccess {
		// Status 0 = Payment Completed
		msg.WriteString("✅ *PEMBAYARAN BERHASIL*\n\n")
		msg.WriteString("🎉 Tagihan PDAM Anda telah lunas!\n\n")
	} else if isPending || resp.VaNo != "" {
		// Status 1 = VA Created - Pending Payment
		msg.WriteString("⏳ *MENUNGGU PEMBAYARAN*\n\n")
		msg.WriteString("✅ Virtual Account berhasil dibuat\n")
		msg.WriteString("💳 Silakan lakukan pembayaran\n\n")
	} else if isFailed {
		// Status 2 = Payment Failed
		msg.WriteString("❌ *PEMBAYARAN GAGAL*\n\n")
		msg.WriteString("Mohon maaf, pembayaran tidak dapat diproses.\n\n")
	} else {
		// Other status
		msg.WriteString("✅ *TRANSAKSI DITERIMA*\n\n")
	}

	// Virtual Account Info - Show for PENDING status
	if isPending {
		msg.WriteString("🏦 *INFORMASI VIRTUAL ACCOUNT*\n")
		msg.WriteString("━━━━━━━━━━━━━━━━━━━━\n\n")

		// VA info is in Detail object
		vaNo := ""
		bankCode := ""
		validUntil := ""
		if resp.Detail != nil {
			vaNo = resp.Detail.VaNo
			bankCode = resp.Detail.BankCode
			validUntil = resp.Detail.ValidUntil
		}

		if vaNo != "" {
			// VA Number exists
			// Bank name
			bankName := "Virtual Account"
			switch bankCode {
			case "BCA":
				bankName = "BCA Virtual Account"
			case "MANDIRI", "BMRI":
				bankName = "Mandiri Virtual Account"
			case "BRI":
				bankName = "BRI Virtual Account"
			default:
				if bankCode != "" {
					bankName = bankCode + " Virtual Account"
				}
			}
			msg.WriteString(fmt.Sprintf("🏧 Bank: *%s*\n\n", bankName))

			// VA Number - HIGHLIGHT dengan code block untuk copy
			msg.WriteString("📱 *NOMOR VIRTUAL ACCOUNT:*\n")
			msg.WriteString(fmt.Sprintf("```%s```\n", vaNo))
			msg.WriteString("_(Tap nomor di atas untuk copy)_\n\n")

			// Valid until - JELAS
			if validUntil != "" {
				msg.WriteString(fmt.Sprintf("⏰ Berlaku sampai: *%s*\n\n", validUntil))
			}
		} else {
			// VA Number not yet generated - show message
			txID := "N/A"
			if resp.TxID != nil {
				txID = *resp.TxID
			}
			customLogger.WarnWithData("VaNo is empty for PENDING status", map[string]interface{}{
				"component": "GroqService",
				"function":  "formatPaymentResponse",
				"tx_id":     txID,
			})

			msg.WriteString("⚠️ *Nomor Virtual Account sedang diproses*\n\n")
			msg.WriteString("Nomor VA akan dikirimkan dalam beberapa saat.\n")
			msg.WriteString("Atau hubungi customer service untuk informasi lebih lanjut.\n\n")
			msg.WriteString("📞 *Call Center PDAM Kabupaten Tangerang*\n")
			msg.WriteString("☎️ (021) 5951234\n\n")
		}

		msg.WriteString("━━━━━━━━━━━━━━━━━━━━\n\n")
	}

	// Transaction details
	msg.WriteString("📋 *DETAIL TRANSAKSI*\n\n")

	if resp.TxID != nil {
		msg.WriteString(fmt.Sprintf("🔖 ID Transaksi: %s\n", *resp.TxID))
	}

	// Show status explicitly
	if resp.Status != nil {
		statusText := *resp.Status
		statusEmoji := "📊"
		switch statusText {
		case "0":
			statusText = "Lunas"
			statusEmoji = "✅"
		case "1":
			statusText = "Menunggu Pembayaran"
			statusEmoji = "⏳"
		case "2":
			statusText = "Gagal"
			statusEmoji = "❌"
		case "3":
			statusText = "Inquiry"
			statusEmoji = "📋"
		}
		msg.WriteString(fmt.Sprintf("%s Status: *%s*\n", statusEmoji, statusText))
	}

	if resp.BillID != nil {
		msg.WriteString(fmt.Sprintf("🏷️ Nomor Pelanggan: %s\n", *resp.BillID))
	}

	// Customer details
	if resp.Detail != nil {
		if resp.Detail.CustomerName != nil {
			msg.WriteString(fmt.Sprintf("👤 Nama: %s\n", *resp.Detail.CustomerName))
		}
		if resp.Detail.Period != nil {
			msg.WriteString(fmt.Sprintf("📅 Periode: %s\n", *resp.Detail.Period))
		}
	}

	msg.WriteString("\n")

	// Payment amount
	msg.WriteString("💰 *TOTAL YANG HARUS DIBAYAR*\n")

	if resp.Amount != nil {
		msg.WriteString(fmt.Sprintf("   Tagihan: Rp %s\n", formatRupiahFloat(*resp.Amount)))
	}

	if resp.Admin != nil && *resp.Admin > 0 {
		msg.WriteString(fmt.Sprintf("   Admin: Rp %s\n", formatRupiahFloat(*resp.Admin)))
	}

	if resp.TotalAmount != nil {
		msg.WriteString(fmt.Sprintf("\n   *TOTAL: Rp %s*\n", formatRupiahFloat(*resp.TotalAmount)))
	}

	msg.WriteString("\n━━━━━━━━━━━━━━━━━━━━\n\n")

	// Payment instructions (only for pending status)
	if isPending && !isSuccess && !isFailed {
		// Get VA info from Detail
		vaNo := ""
		validUntil := ""
		if resp.Detail != nil {
			vaNo = resp.Detail.VaNo
			validUntil = resp.Detail.ValidUntil
		}

		msg.WriteString("📱 *CARA PEMBAYARAN:*\n\n")
		msg.WriteString("1️⃣ Buka aplikasi mobile banking Anda\n")
		msg.WriteString("2️⃣ Pilih menu *Transfer* atau *Pembayaran*\n")
		msg.WriteString("3️⃣ Pilih *Virtual Account*\n")
		if vaNo != "" {
			msg.WriteString(fmt.Sprintf("4️⃣ Masukkan nomor VA: *%s*\n", vaNo))
			msg.WriteString("5️⃣ Periksa detail pembayaran\n")
			msg.WriteString("6️⃣ Konfirmasi pembayaran\n\n")
		} else {
			msg.WriteString("4️⃣ Masukkan nomor VA yang akan dikirimkan\n")
			msg.WriteString("5️⃣ Periksa detail pembayaran\n")
			msg.WriteString("6️⃣ Konfirmasi pembayaran\n\n")
		}

		msg.WriteString("⚠️ *PENTING:*\n")
		msg.WriteString("• Pastikan nominal yang dibayar sesuai\n")
		msg.WriteString("• Simpan bukti pembayaran\n")
		if validUntil != "" {
			msg.WriteString(fmt.Sprintf("• Bayar sebelum *%s*\n", validUntil))
		}
	} else if isSuccess {
		// Success message (status 0)
		msg.WriteString("✅ *PEMBAYARAN ANDA TELAH DITERIMA*\n\n")
		msg.WriteString("Terima kasih telah melakukan pembayaran.\n")
		msg.WriteString("Tagihan Anda telah lunas.\n")
	} else if isFailed {
		// Failed message (status 2)
		msg.WriteString("❌ *PEMBAYARAN TIDAK DAPAT DIPROSES*\n\n")
		msg.WriteString("Silakan coba lagi atau hubungi customer service.\n")
		msg.WriteString("\n📞 *Call Center PDAM Kabupaten Tangerang*\n")
		msg.WriteString("☎️ (021) 5951234\n")
	}

	msg.WriteString("\nTerima kasih! 🙏")

	return msg.String()
}

//---------------------------------------

var groqSvc *GroqSvcImpl
var groqSvcOnce sync.Once

func GetGroqSvc() GroqSvc {
	if groqSvc == nil {
		groqSvcOnce.Do(func() {
			groqSvc = &GroqSvcImpl{
				apiKey:  config.Get(constans.GroqApiKey, ""),
				baseURL: "https://api.groq.com/openai/v1/chat/completions",
			}
		})
	}

	return groqSvc
}
