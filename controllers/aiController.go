package controllers

import (
	"sync"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	a "gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/services"
)

type AIController struct {
	gocom.Controller
}

var aiController *AIController
var onceAIController sync.Once

func GetAIController() *AIController {
	onceAIController.Do(func() {
		aiController = &AIController{}
	})
	return aiController
}

func (o *AIController) Init() {
	gocom.POST("/api/v1/instant-link/ai/chat", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.chat)
	gocom.POST("/api/v1/instant-link/ai/chat-context", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.chatWithContext)
	gocom.POST("/api/v1/instant-link/ai/conversation", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.sendConversation)
	gocom.POST("/api/v1/instant-link/ai/knowledge/refresh", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.refreshKnowledge)
	gocom.POST("/api/v1/instant-link/ai/knowledge/refresh-all", a.Allow(a.Role(a.ADMIN)), o.refreshAllKnowledge)
}

// chat handles AI chat requests
// POST /api/v1/instant-link/ai/chat
func (o *AIController) chat(ctx gocom.Context) error {
	logger.Debug("AIController.chat - Start")

	var req dtos.AIChatReq
	if err := ctx.Bind(&req); err != nil {
		logger.Warnf("AIController.chat - Bind error: %v", err)
		return ctx.SendError(gocom.NewError(400, "Invalid request"))
	}

	ret, codedErr := services.GetGroqSvc().Chat(req.Message)
	if codedErr != nil {
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult(dtos.AIChatResp{
		Response: ret,
		From:     req.From,
	})
}

// chatWithContext handles AI chat with conversation context
// POST /api/v1/instant-link/ai/chat-context
func (o *AIController) chatWithContext(ctx gocom.Context) error {
	logger.Debug("AIController.chatWithContext - Start")

	var req dtos.AIChatContextReq
	if err := ctx.Bind(&req); err != nil {
		logger.Warnf("AIController.chatWithContext - Bind error: %v", err)
		return ctx.SendError(gocom.NewError(400, "Invalid request"))
	}

	ret, codedErr := services.GetGroqSvc().ChatWithContext(req.Messages)
	if codedErr != nil {
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult(dtos.AIChatResp{
		Response: ret,
		From:     req.From,
	})
}

// sendConversation handles AI conversation with WhatsApp auto-reply
// POST /api/v1/instant-link/ai/conversation
func (o *AIController) sendConversation(ctx gocom.Context) error {
	logger.Debug("AIController.sendConversation - Start")

	var req dtos.AIConversationReq
	if err := ctx.Bind(&req); err != nil {
		logger.Warnf("AIController.sendConversation - Bind error: %v", err)
		return ctx.SendError(gocom.NewError(400, "Invalid request"))
	}

	// Get clientId from auth context
	authInfo := a.Get(ctx)
	clientId := authInfo.ClientId

	// Get AI response and send to WhatsApp
	messageId, aiResponse, codedErr := services.GetGroqSvc().SendConversationMessage(req.SessionID, req.From, req.Message, clientId)
	if codedErr != nil {
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult(dtos.AIConversationResp{
		MessageId:  messageId,
		To:         req.From,
		Status:     "sent",
		Message:    "AI response sent successfully",
		AIResponse: aiResponse,
	})
}

// refreshKnowledge refreshes AI knowledge cache for specific client
// POST /api/v1/instant-link/ai/knowledge/refresh
func (o *AIController) refreshKnowledge(ctx gocom.Context) error {
	logger.Debug("AIController.refreshKnowledge - Start")

	var req struct {
		ClientId string `json:"client_id" binding:"required"`
	}

	if err := ctx.Bind(&req); err != nil {
		logger.Warnf("AIController.refreshKnowledge - Bind error: %v", err)
		return ctx.SendError(gocom.NewError(400, "Invalid request - client_id required"))
	}

	codedErr := services.GetAIKnowledgeCacheSvc().RefreshCache(req.ClientId)
	if codedErr != nil {
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult(map[string]interface{}{
		"status":    "success",
		"message":   "Knowledge cache refreshed successfully",
		"client_id": req.ClientId,
	})
}

// refreshAllKnowledge refreshes AI knowledge cache for all clients
// POST /api/v1/instant-link/ai/knowledge/refresh-all
func (o *AIController) refreshAllKnowledge(ctx gocom.Context) error {
	logger.Debug("AIController.refreshAllKnowledge - Start")

	codedErr := services.GetAIKnowledgeCacheSvc().RefreshAllClients()
	if codedErr != nil {
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult(map[string]interface{}{
		"status":  "success",
		"message": "All knowledge caches refreshed successfully",
	})
}
