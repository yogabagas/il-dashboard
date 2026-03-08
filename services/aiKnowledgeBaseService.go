package services

import (
	"sync"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	"github.com/jinzhu/copier"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/repositories/aiKnowledgeBase"
)

type AIKnowledgeBaseService interface {
	Create(req dtos.AiKnowledgeBaseCreateReq, authInfo auth.AuthInfo) (*dtos.AiKnowledgeBaseResp, *gocom.CodedError)
	Update(id string, req dtos.AiKnowledgeBaseUpdateReq, authInfo auth.AuthInfo) (*dtos.AiKnowledgeBaseResp, *gocom.CodedError)
	GetById(id string, authInfo auth.AuthInfo) (*dtos.AiKnowledgeBaseResp, *gocom.CodedError)
	Search(req dtos.AiKnowledgeBaseSearchReq, authInfo auth.AuthInfo) ([]*dtos.AiKnowledgeBaseResp, bool, int)
	Delete(id string, authInfo auth.AuthInfo) *gocom.CodedError
}

type AIKnowledgeBaseSvcImpl struct{}

var aiKnowledgeBaseService AIKnowledgeBaseService
var onceAIKnowledgeBaseService sync.Once

func GetAIKnowledgeBaseService() AIKnowledgeBaseService {
	onceAIKnowledgeBaseService.Do(func() {
		aiKnowledgeBaseService = &AIKnowledgeBaseSvcImpl{}
	})
	return aiKnowledgeBaseService
}

func (o *AIKnowledgeBaseSvcImpl) Create(req dtos.AiKnowledgeBaseCreateReq, authInfo auth.AuthInfo) (*dtos.AiKnowledgeBaseResp, *gocom.CodedError) {
	logger.Infof("AIKnowledgeBaseService.Create - Start req=%+v", req)

	baseKnowledgeMdl := aiKnowledgeBase.GetRepo().GetByQuestion(req.Question)
	if baseKnowledgeMdl != nil {
		logger.Warnf("AIKnowledgeBaseService.Create - Knowledge with the same question already exists: %s", req.Question)

		ret := &dtos.AiKnowledgeBaseResp{}
		_ = copier.Copy(&ret, &baseKnowledgeMdl)
		return ret, nil
	}

	// Validation
	if req.ClientId == "" {
		logger.Warnf("AIKnowledgeBaseService.Create - ClientId is required")
		return nil, gocom.NewError(400, "ClientId is required")
	}
	if req.Question == "" {
		logger.Warnf("AIKnowledgeBaseService.Create - Question is required")
		return nil, gocom.NewError(400, "Question is required")
	}
	if req.Answer == "" {
		logger.Warnf("AIKnowledgeBaseService.Create - Answer is required")
		return nil, gocom.NewError(400, "Answer is required")
	}

	knowledge := &aiKnowledgeBase.AiKnowledgeBase{
		ClientId:  authInfo.ClientId,
		Category:  req.Category,
		Question:  req.Question,
		Answer:    req.Answer,
		Keywords:  req.Keywords,
		IsActive:  true,
		Priority:  req.Priority,
		CreatedBy: authInfo.UserId,
		UpdatedBy: authInfo.UserId,
	}

	err := aiKnowledgeBase.GetRepo().Create(knowledge).Error
	if err != nil {
		logger.Errorf("AIKnowledgeBaseService.Create - Create error: %v", err)
		return nil, gocom.NewError(500, "Failed to create knowledge")
	}

	// Auto refresh cache for this client after insert
	_ = GetAIKnowledgeCacheSvc().RefreshCache(req.ClientId)

	return &dtos.AiKnowledgeBaseResp{
		ID:        knowledge.ID,
		ClientId:  knowledge.ClientId,
		Category:  knowledge.Category,
		Question:  knowledge.Question,
		Answer:    knowledge.Answer,
		Keywords:  knowledge.Keywords,
		IsActive:  knowledge.IsActive,
		Priority:  knowledge.Priority,
		CreatedBy: knowledge.CreatedBy,
		UpdatedBy: knowledge.UpdatedBy,
		CreatedAt: knowledge.CreatedAt,
		UpdatedAt: knowledge.UpdatedAt,
	}, nil
}

func (o *AIKnowledgeBaseSvcImpl) Update(id string, req dtos.AiKnowledgeBaseUpdateReq, authInfo auth.AuthInfo) (*dtos.AiKnowledgeBaseResp, *gocom.CodedError) {
	logger.Infof("AIKnowledgeBaseService.Update - Start id=%s req=%+v", id, req)

	if id == "" {
		logger.Warnf("AIKnowledgeBaseService.Update - ID is required")
		return nil, common.ERR_INVALID_REQUEST
	}

	knowledge := aiKnowledgeBase.GetRepo().GetById(id)
	if knowledge == nil {
		logger.Warnf("AIKnowledgeBaseService.Update - Knowledge not found id=%s", id)
		return nil, common.ERR_NOT_FOUND
	}

	// OWNER can only update knowledge base for their own client
	if common.PROVIDER_ID != authInfo.ClientId {
		if knowledge.ClientId != authInfo.ClientId {
			logger.Warnf("AIKnowledgeBaseService.Update - Owner can only update their own client's knowledge. userId=%s clientId=%s knowledgeClientId=%s", authInfo.UserId, authInfo.ClientId, knowledge.ClientId)
			return nil, common.ERR_NOT_ALLOWED
		}
	}

	// Update only provided fields
	if req.Category != "" {
		knowledge.Category = req.Category
	}
	if req.Question != "" {
		knowledge.Question = req.Question
	}
	if req.Answer != "" {
		knowledge.Answer = req.Answer
	}
	if req.Keywords != "" {
		knowledge.Keywords = req.Keywords
	}
	if req.IsActive != nil {
		knowledge.IsActive = *req.IsActive
	}
	if req.Priority != nil {
		knowledge.Priority = *req.Priority
	}
	knowledge.UpdatedBy = authInfo.UserId

	err := aiKnowledgeBase.GetRepo().Update(knowledge)
	if err != nil {
		logger.Errorf("AIKnowledgeBaseService.Update - Update error: %v", err)
		return nil, gocom.NewError(500, "Failed to update knowledge")
	}

	// Auto refresh cache for this client after update
	_ = GetAIKnowledgeCacheSvc().RefreshCache(knowledge.ClientId)

	return &dtos.AiKnowledgeBaseResp{
		ID:        knowledge.ID,
		ClientId:  knowledge.ClientId,
		Category:  knowledge.Category,
		Question:  knowledge.Question,
		Answer:    knowledge.Answer,
		Keywords:  knowledge.Keywords,
		IsActive:  knowledge.IsActive,
		Priority:  knowledge.Priority,
		CreatedBy: knowledge.CreatedBy,
		UpdatedBy: knowledge.UpdatedBy,
		CreatedAt: knowledge.CreatedAt,
		UpdatedAt: knowledge.UpdatedAt,
	}, nil
}

func (o *AIKnowledgeBaseSvcImpl) GetById(id string, authInfo auth.AuthInfo) (*dtos.AiKnowledgeBaseResp, *gocom.CodedError) {
	logger.Infof("AIKnowledgeBaseService.GetById - Start id=%s", id)

	if id == "" {
		logger.Warnf("AIKnowledgeBaseService.GetById - ID is required")
		return nil, common.ERR_INVALID_REQUEST
	}

	knowledge := aiKnowledgeBase.GetRepo().GetById(id)
	if knowledge == nil {
		logger.Warnf("AIKnowledgeBaseService.GetById - Knowledge not found id=%s", id)
		return nil, common.ERR_NOT_FOUND
	}

	// OWNER can only get knowledge base for their own client
	//if common.PROVIDER_ID != authInfo.ClientId {
	//	logger.Warnf("AIKnowledgeBaseService.GetById - Owner can only get their own client's knowledge. userId=%s clientId=%s knowledgeClientId=%s", authInfo.UserId, authInfo.ClientId, knowledge.ClientId)
	//	return nil, common.ERR_NOT_ALLOWED
	//}

	return &dtos.AiKnowledgeBaseResp{
		ID:        knowledge.ID,
		ClientId:  knowledge.ClientId,
		Category:  knowledge.Category,
		Question:  knowledge.Question,
		Answer:    knowledge.Answer,
		Keywords:  knowledge.Keywords,
		IsActive:  knowledge.IsActive,
		Priority:  knowledge.Priority,
		CreatedBy: knowledge.CreatedBy,
		UpdatedBy: knowledge.UpdatedBy,
		CreatedAt: knowledge.CreatedAt,
		UpdatedAt: knowledge.UpdatedAt,
	}, nil
}

func (o *AIKnowledgeBaseSvcImpl) Search(req dtos.AiKnowledgeBaseSearchReq, authInfo auth.AuthInfo) ([]*dtos.AiKnowledgeBaseResp, bool, int) {
	logger.Infof("AIKnowledgeBaseService.Search - Start req=%+v", req)

	// Default pagination
	if req.PageNo <= 0 {
		req.PageNo = 1
	}
	if req.RowPerPage <= 0 {
		req.RowPerPage = 10
	}

	// OWNER can only search knowledge base for their own client
	clientId := req.ClientId
	if common.PROVIDER_ID != authInfo.ClientId {
		clientId = authInfo.ClientId
		logger.Debugf("AIKnowledgeBaseService.Search - Owner restricted to clientId=%s", clientId)
	}

	results, haveNext, total := aiKnowledgeBase.GetRepo().Search(
		req.Filter,
		clientId,
		req.Category,
		req.IsActive,
		req.DateFrom,
		req.DateTo,
		req.PageNo,
		req.RowPerPage,
	)

	// Convert to response DTOs
	data := make([]*dtos.AiKnowledgeBaseResp, 0, len(results))
	for _, k := range results {
		data = append(data, &dtos.AiKnowledgeBaseResp{
			ID:        k.ID,
			ClientId:  k.ClientId,
			Category:  k.Category,
			Question:  k.Question,
			Answer:    k.Answer,
			Keywords:  k.Keywords,
			IsActive:  k.IsActive,
			Priority:  k.Priority,
			CreatedBy: k.CreatedBy,
			UpdatedBy: k.UpdatedBy,
			CreatedAt: k.CreatedAt,
			UpdatedAt: k.UpdatedAt,
		})
	}

	return data, haveNext, int(total)
}

func (o *AIKnowledgeBaseSvcImpl) Delete(id string, authInfo auth.AuthInfo) *gocom.CodedError {
	logger.Infof("AIKnowledgeBaseService.Delete - Start id=%s", id)

	if id == "" {
		logger.Warnf("AIKnowledgeBaseService.Delete - ID is required")
		return common.ERR_INVALID_REQUEST
	}

	knowledge := aiKnowledgeBase.GetRepo().GetById(id)
	if knowledge == nil {
		logger.Warnf("AIKnowledgeBaseService.Delete - Knowledge not found id=%s", id)
		return common.ERR_NOT_FOUND
	}

	// OWNER can only delete knowledge base for their own client
	if common.PROVIDER_ID != authInfo.ClientId {
		if knowledge.ClientId != authInfo.ClientId {
			logger.Warnf("AIKnowledgeBaseService.Delete - Owner can only delete their own client's knowledge. userId=%s clientId=%s knowledgeClientId=%s", authInfo.UserId, authInfo.ClientId, knowledge.ClientId)
			return common.ERR_NOT_ALLOWED
		}
	}

	err := aiKnowledgeBase.GetRepo().Delete(id, authInfo.UserId)
	if err != nil {
		logger.Errorf("AIKnowledgeBaseService.Delete - Delete error: %v", err)
		return gocom.NewError(500, "Failed to delete knowledge")
	}

	// Auto refresh cache for this client after delete
	_ = GetAIKnowledgeCacheSvc().RefreshCache(knowledge.ClientId)

	return nil
}
