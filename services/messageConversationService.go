package services

import (
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/config"
	"github.com/ariandi/gocom/logger"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/repositories/messageConversation"
)

// ============================================================================
// Message Conversation Service
// ============================================================================

type MessageConversationService interface {
	CreateUserMessage(clientId, sessionId, fromNumber, message, messageLogId string) (*messageConversation.MessageConversation, error)
	CreateAssistantMessage(clientId, sessionId, fromNumber, toNumber, message, messageLogId, model string, tokensUsed int) (*messageConversation.MessageConversation, error)
	CreateSystemMessage(clientId, sessionId, fromNumber, message string) (*messageConversation.MessageConversation, error)
	UpdateMessageStatus(messageLogId, status string) error
	UpdateConversationStatus(fromNumber, conversationStatus string) error
	GetConversationHistory(fromNumber string, limit int) []messageConversation.MessageConversation
	CloseConversation(fromNumber string) error
	IsConversationActive(fromNumber string) bool
	IsConversationEscalated(fromNumber string) bool
	Search(req dtos.MessageConversationSearchReq, authInfo auth.AuthInfo) ([]*dtos.MessageConversation, bool, int64)
	Reply(req dtos.ConversationReplyReq, authInfo auth.AuthInfo) (*dtos.MessageConversation, *gocom.CodedError)
}

type MessageConversationServiceImpl struct {
	repo *messageConversation.MessageConversationRepo
}

var messageConversationService MessageConversationService
var onceMessageConversationService sync.Once

func GetMessageConversationService() MessageConversationService {
	onceMessageConversationService.Do(func() {
		messageConversationService = &MessageConversationServiceImpl{
			repo: messageConversation.GetRepo(),
		}
	})
	return messageConversationService
}

// CreateUserMessage creates a user message in conversation
func (s *MessageConversationServiceImpl) CreateUserMessage(clientId, sessionId, fromNumber, message, messageLogId string) (*messageConversation.MessageConversation, error) {
	conv := &messageConversation.MessageConversation{
		ClientId:           clientId,
		SessionID:          sessionId,
		FromNumber:         fromNumber,
		Role:               "user",
		Message:            message,
		MessageLogId:       messageLogId,
		Status:             "received",
		ConversationStatus: "active",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.repo.Create(conv).Error; err != nil {
		logger.Errorf("[MessageConversationService CreateUserMessage] Failed to create: %v", err)
		return nil, err
	}

	logger.Infof("[MessageConversationService CreateUserMessage] Created user message: %s", conv.ID)
	return conv, nil
}

// CreateAssistantMessage creates an assistant message in conversation
func (s *MessageConversationServiceImpl) CreateAssistantMessage(clientId, sessionId, fromNumber, toNumber, message, messageLogId, model string, tokensUsed int) (*messageConversation.MessageConversation, error) {
	conv := &messageConversation.MessageConversation{
		ClientId:           clientId,
		SessionID:          sessionId,
		FromNumber:         fromNumber,
		ToNumber:           toNumber,
		Role:               "assistant",
		Message:            message,
		MessageLogId:       messageLogId,
		TokensUsed:         tokensUsed,
		Model:              model,
		Status:             "sent",
		ConversationStatus: "active",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.repo.Create(conv).Error; err != nil {
		logger.Errorf("[MessageConversationService CreateAssistantMessage] Failed to create: %v", err)
		return nil, err
	}

	logger.Infof("[MessageConversationService CreateAssistantMessage] Created assistant message: %s", conv.ID)
	return conv, nil
}

// CreateSystemMessage creates a system message in conversation
func (s *MessageConversationServiceImpl) CreateSystemMessage(clientId, sessionId, fromNumber, message string) (*messageConversation.MessageConversation, error) {
	conv := &messageConversation.MessageConversation{
		ClientId:           clientId,
		SessionID:          sessionId,
		FromNumber:         fromNumber,
		Role:               "system",
		Message:            message,
		Status:             "sent",
		ConversationStatus: "active",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.repo.Create(conv).Error; err != nil {
		logger.Errorf("[MessageConversationService CreateSystemMessage] Failed to create: %v", err)
		return nil, err
	}

	logger.Infof("[MessageConversationService CreateSystemMessage] Created system message: %s", conv.ID)
	return conv, nil
}

// UpdateMessageStatus updates message status based on WhatsApp webhook (sent, delivered, read)
func (s *MessageConversationServiceImpl) UpdateMessageStatus(messageLogId, status string) error {
	if messageLogId == "" {
		return nil // Skip if no message log ID
	}

	conv := s.repo.GetByMessageLogId(messageLogId)
	if conv == nil {
		logger.Debugf("[MessageConversationService UpdateMessageStatus] Message conversation not found for log ID: %s", messageLogId)
		return nil // Not an error - message might not have conversation record
	}

	// Only update if status changed
	if conv.Status == status {
		return nil
	}

	conv.Status = status
	conv.UpdatedAt = time.Now()

	if err := s.repo.Update(conv); err != nil {
		logger.Errorf("[MessageConversationService UpdateMessageStatus] Failed to update status for %s: %v", messageLogId, err)
		return err
	}

	logger.Infof("[MessageConversationService UpdateMessageStatus] Updated message %s status to: %s", messageLogId, status)
	return nil
}

// UpdateConversationStatus updates conversation status (active, escalated, closed)
func (s *MessageConversationServiceImpl) UpdateConversationStatus(fromNumber, conversationStatus string) error {
	// Get all messages from this user in active conversation
	messages := s.repo.GetByFromNumber(fromNumber, 100) // Get recent messages

	if len(messages) == 0 {
		logger.Warnf("[MessageConversationService UpdateConversationStatus] No messages found for: %s", fromNumber)
		return nil
	}

	// Update all active messages to new status
	for _, msg := range messages {
		if msg.ConversationStatus == conversationStatus {
			continue // Skip if already in desired status
		}

		msg.ConversationStatus = conversationStatus
		msg.UpdatedAt = time.Now()

		if err := s.repo.Update(&msg); err != nil {
			logger.Errorf("[MessageConversationService UpdateConversationStatus] Failed to update: %v", err)
		}
	}

	logger.Infof("[MessageConversationService UpdateConversationStatus] Updated %d messages for %s to status: %s",
		len(messages), fromNumber, conversationStatus)
	return nil
}

// GetConversationHistory gets conversation history for user
func (s *MessageConversationServiceImpl) GetConversationHistory(fromNumber string, limit int) []messageConversation.MessageConversation {
	return s.repo.GetConversationHistory(fromNumber, limit)
}

// CloseConversation closes conversation for user
func (s *MessageConversationServiceImpl) CloseConversation(fromNumber string) error {
	logger.Infof("[MessageConversationService CloseConversation] Closing conversation for: %s", fromNumber)
	return s.repo.CloseConversation(fromNumber)
}

// IsConversationActive checks if conversation is active
func (s *MessageConversationServiceImpl) IsConversationActive(fromNumber string) bool {
	return s.repo.IsConversationActive(fromNumber)
}

// IsConversationEscalated checks if conversation is escalated to CS
func (s *MessageConversationServiceImpl) IsConversationEscalated(fromNumber string) bool {
	return s.repo.IsConversationEscalated(fromNumber)
}

// Search returns paginated conversation history with role-based access control and 3-month max range
func (s *MessageConversationServiceImpl) Search(req dtos.MessageConversationSearchReq, authInfo auth.AuthInfo) ([]*dtos.MessageConversation, bool, int64) {
	if req.RowPerPage <= 0 {
		req.RowPerPage = config.GetInt(constans.StaticDefaultEnv)
	}
	if req.PageNo <= 0 {
		req.PageNo = 1
	}

	// Owner/Staff: force clientId to their own
	clientId := req.ClientId
	if authInfo.ClientId != common.PROVIDER_ID {
		clientId = authInfo.ClientId
	}

	// Enforce 3-month max date range
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	now := time.Now()

	var dateFrom, dateTo *time.Time

	if req.DateFrom != "" {
		t, err := time.Parse("2006-01-02", req.DateFrom)
		if err == nil {
			if t.Before(threeMonthsAgo) {
				t = threeMonthsAgo
			}
			dateFrom = &t
		}
	}
	if dateFrom == nil {
		dateFrom = &threeMonthsAgo
	}

	if req.DateTo != "" {
		t, err := time.Parse("2006-01-02", req.DateTo)
		if err == nil {
			endOfDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
			dateTo = &endOfDay
		}
	}
	if dateTo == nil {
		dateTo = &now
	}

	mdls, haveNext, count := s.repo.Search(req.Filter, clientId, req.FromNumber, req.SessionId, "", dateFrom, dateTo, req.PageNo, req.RowPerPage)

	ret := make([]*dtos.MessageConversation, 0, len(mdls))
	for _, m := range mdls {
		ret = append(ret, s.toDTO(&m))
	}
	return ret, haveNext, count
}

// Reply sends a text reply to a customer and saves it to conversation history
func (s *MessageConversationServiceImpl) Reply(req dtos.ConversationReplyReq, authInfo auth.AuthInfo) (*dtos.MessageConversation, *gocom.CodedError) {
	if req.FromNumber == "" {
		return nil, &gocom.CodedError{Code: 400, Message: "fromNumber is required"}
	}
	if req.Message == "" {
		return nil, &gocom.CodedError{Code: 400, Message: "message is required"}
	}

	// Lookup last user message to get business phone (toNumber) and clientId
	lastMsgs, _, _ := s.repo.Search("", "", req.FromNumber, "", "user", nil, nil, 1, 1)
	if len(lastMsgs) == 0 {
		return nil, &gocom.CodedError{Code: 404, Message: "No conversation found for this number"}
	}

	lastMsg := lastMsgs[0]
	senderIdentifier := lastMsg.ToNumber // Business phone that received the original message
	clientId := lastMsg.ClientId
	sessionID := lastMsg.SessionID

	// Owner/Staff: must belong to the same clientId
	if authInfo.ClientId != common.PROVIDER_ID && authInfo.ClientId != clientId {
		return nil, common.ERR_NOT_ALLOWED
	}

	// Admin: can override clientId if explicitly specified
	if authInfo.ClientId == common.PROVIDER_ID && req.ClientId != "" {
		clientId = req.ClientId
	}

	// Send via WhatsApp
	_, messageLogId, sendErr := GetWASendSvc().SendTextMessageWithLog(req.FromNumber, req.Message, clientId, senderIdentifier)
	if sendErr != nil {
		return nil, sendErr
	}

	// Save reply to conversation history
	conv := &messageConversation.MessageConversation{
		ClientId:           clientId,
		SessionID:          sessionID,
		FromNumber:         req.FromNumber,
		ToNumber:           req.FromNumber,
		Role:               "assistant",
		Message:            req.Message,
		MessageLogId:       messageLogId,
		Status:             "sent",
		ConversationStatus: "active",
	}

	if err := s.repo.Create(conv).Error; err != nil {
		logger.Errorf("[MessageConversationService Reply] Failed to save conversation: %v", err)
	}

	return s.toDTO(conv), nil
}

func (s *MessageConversationServiceImpl) toDTO(m *messageConversation.MessageConversation) *dtos.MessageConversation {
	return &dtos.MessageConversation{
		ID:                 m.ID,
		ClientId:           m.ClientId,
		SessionID:          m.SessionID,
		FromNumber:         m.FromNumber,
		ToNumber:           m.ToNumber,
		Role:               m.Role,
		Message:            m.Message,
		MessageLogId:       m.MessageLogId,
		TokensUsed:         m.TokensUsed,
		Model:              m.Model,
		Status:             m.Status,
		ConversationStatus: m.ConversationStatus,
		ErrorMessage:       m.ErrorMessage,
		CreatedAt:          m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          m.UpdatedAt.Format(time.RFC3339),
	}
}
