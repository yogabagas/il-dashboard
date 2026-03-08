package services

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/config"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/campaign"
	"gitlab.com/anti_metter/switching_common/messageLog"
	"gitlab.com/anti_metter/switching_common/sender"
	"gitlab.com/anti_metter/switching_common/waTemplate"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
)

type MessageLogsService interface {
	Create(log *dtos.MessageLog, authInfo auth.AuthInfo) (*dtos.MessageLog, *gocom.CodedError)
	GetById(logId string, authInfo auth.AuthInfo) (*dtos.MessageLog, *gocom.CodedError)
	GetByMessageId(messageId string) (*dtos.MessageLog, *gocom.CodedError)
	Search(req dtos.MessageLogSearchReq, sortBy, sortOrder string, authInfo auth.AuthInfo) ([]*dtos.MessageLog, bool, int)
	UpdateStatus(logId, status, messageId, errorMessage string) *gocom.CodedError
	UpdateDeliveryStatus(messageId, status string, deliveredAt, readAt, failedAt *time.Time, errorMessage string) *gocom.CodedError
}

type MessageLogsSvcImpl struct{}

var messageLogsService MessageLogsService
var onceMessageLogsService sync.Once

func GetMessageLogsService() MessageLogsService {
	onceMessageLogsService.Do(func() {
		messageLogsService = &MessageLogsSvcImpl{}
	})
	return messageLogsService
}

func marshalMetadata(m map[string]interface{}) string {
	if m == nil || len(m) == 0 {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		customLogger.DebugWithData("Unable to marshal metadata", map[string]interface{}{
			"component": "MessageLogsService",
			"function":  "marshalMetadata",
			"error":     err.Error(),
		})
		return "{}"
	}
	return string(b)
}

func unmarshalMetadata(s string) map[string]interface{} {
	if s == "" {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		customLogger.DebugWithData("Unable to unmarshal metadata", map[string]interface{}{
			"component": "MessageLogsService",
			"function":  "unmarshalMetadata",
			"error":     err.Error(),
		})
		return nil
	}
	return m
}

func (o *MessageLogsSvcImpl) Create(log *dtos.MessageLog, authInfo auth.AuthInfo) (*dtos.MessageLog, *gocom.CodedError) {
	customLogger.InfoWithData("Create message log started", map[string]interface{}{
		"component":  "MessageLogsService",
		"function":   "Create",
		"log_id":     log.ID,
		"type":       log.Type,
		"direction":  log.Direction,
		"status":     log.Status,
		"client_id":  authInfo.ClientId,
	})

	// Create model
	mdl := &messageLog.MessageLog{
		ID:             log.ID,
		ClientId:       authInfo.ClientId,
		SenderId:       log.SenderId,
		CampaignId:     log.CampaignId,
		Type:           log.Type,
		Direction:      log.Direction,
		RecipientType:  log.RecipientType,
		RecipientValue: log.RecipientValue,
		TemplateId:     log.TemplateId,
		MessageId:      log.MessageId,
		SessionId:      log.SessionId,
		MessageContent: log.MessageContent,
		Status:         log.Status,
		Cost:           log.Cost,
		Metadata:       marshalMetadata(log.Metadata),
	}

	if log.SentAt != "" {
		parsed, _ := time.Parse(time.RFC3339, log.SentAt)
		mdl.SentAt = &parsed
	}

	err := messageLog.GetRepo().Create(mdl).Error
	if err != nil {
		customLogger.ErrorWithData("Failed to create message log", map[string]interface{}{
			"component": "MessageLogsService",
			"function":  "Create",
			"log_id":    mdl.ID,
			"error":     err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	customLogger.InfoWithData("Message log created successfully", map[string]interface{}{
		"component": "MessageLogsService",
		"function":  "Create",
		"log_id":    mdl.ID,
	})
	return o.toDTO(mdl), nil
}

func (o *MessageLogsSvcImpl) GetById(logId string, authInfo auth.AuthInfo) (*dtos.MessageLog, *gocom.CodedError) {
	mdl := messageLog.GetRepo().GetById(logId)
	if mdl == nil {
		customLogger.WarnWithData("Message log not found", map[string]interface{}{
			"component": "MessageLogsService",
			"function":  "GetById",
			"log_id":    logId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if mdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized access to message log", map[string]interface{}{
			"component":       "MessageLogsService",
			"function":        "GetById",
			"log_client_id":   mdl.ClientId,
			"auth_client_id":  authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	return o.toDTO(mdl), nil
}

func (o *MessageLogsSvcImpl) GetByMessageId(messageId string) (*dtos.MessageLog, *gocom.CodedError) {
	mdl := messageLog.GetRepo().GetByMessageId(messageId)
	if mdl == nil {
		customLogger.WarnWithData("Message log not found", map[string]interface{}{
			"component":  "MessageLogsService",
			"function":   "GetByMessageId",
			"message_id": messageId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	return o.toDTO(mdl), nil
}

func (o *MessageLogsSvcImpl) Search(req dtos.MessageLogSearchReq, sortBy, sortOrder string, authInfo auth.AuthInfo) ([]*dtos.MessageLog, bool, int) {
	if req.RowPerPage <= 0 {
		req.RowPerPage = config.GetInt(constans.StaticDefaultEnv)
	}
	if req.PageNo <= 0 {
		req.PageNo = 1
	}

	// Parse dates
	var dateFrom, dateTo *time.Time
	if req.DateFrom != "" {
		parsed, err := time.Parse("2006-01-02", req.DateFrom)
		if err == nil {
			dateFrom = &parsed
		}
	}
	if req.DateTo != "" {
		parsed, err := time.Parse("2006-01-02", req.DateTo)
		if err == nil {
			// Add 1 day to include the whole day
			parsed = parsed.Add(24 * time.Hour)
			dateTo = &parsed
		}
	}

	mdls, haveNext, count := messageLog.GetRepo().Search(
		req.Filter,
		authInfo.ClientId,
		req.CampaignId,
		req.Type,
		req.Status,
		req.Direction,
		req.RecipientVal,
		sortBy,
		sortOrder,
		dateFrom,
		dateTo,
		req.PageNo,
		req.RowPerPage,
	)

	ret := make([]*dtos.MessageLog, len(mdls))
	for i, mdl := range mdls {
		ret[i] = o.toDTO(&mdl)
	}

	customLogger.DebugWithData("Message log search completed", map[string]interface{}{
		"component":  "MessageLogsService",
		"function":   "Search",
		"count":      len(ret),
		"have_next":  haveNext,
		"total":      count,
		"client_id":  authInfo.ClientId,
	})
	return ret, haveNext, int(count)
}

func (o *MessageLogsSvcImpl) UpdateStatus(logId, status, messageId, errorMessage string) *gocom.CodedError {
	customLogger.InfoWithData("Update message log status started", map[string]interface{}{
		"component":  "MessageLogsService",
		"function":   "UpdateStatus",
		"log_id":     logId,
		"status":     status,
		"message_id": messageId,
	})

	mdl := messageLog.GetRepo().GetById(logId)
	if mdl == nil {
		customLogger.WarnWithData("Message log not found", map[string]interface{}{
			"component": "MessageLogsService",
			"function":  "UpdateStatus",
			"log_id":    logId,
		})
		return common.ERR_NOT_FOUND
	}

	mdl.Status = status
	if messageId != "" {
		mdl.MessageId = messageId
	}
	if errorMessage != "" {
		mdl.ErrorMessage = errorMessage
	}

	now := time.Now()
	switch status {
	case "sent":
		mdl.SentAt = &now
	case "delivered":
		mdl.DeliveredAt = &now
	case "read":
		mdl.ReadAt = &now
	case "failed":
		mdl.FailedAt = &now
	}

	err := messageLog.GetRepo().Update(mdl)
	if err != nil {
		customLogger.ErrorWithData("Failed to update message log", map[string]interface{}{
			"component": "MessageLogsService",
			"function":  "UpdateStatus",
			"log_id":    logId,
			"error":     err.Error,
		})
		return common.ERR_UNABLE_TO_CREATE
	}

	customLogger.InfoWithData("Message log status updated", map[string]interface{}{
		"component": "MessageLogsService",
		"function":  "UpdateStatus",
		"log_id":    logId,
		"status":    status,
	})
	return nil
}

func (o *MessageLogsSvcImpl) UpdateDeliveryStatus(messageId, status string, deliveredAt, readAt, failedAt *time.Time, errorMessage string) *gocom.CodedError {
	customLogger.InfoWithData("Update delivery status started", map[string]interface{}{
		"component":  "MessageLogsService",
		"function":   "UpdateDeliveryStatus",
		"message_id": messageId,
		"status":     status,
	})

	mdl := messageLog.GetRepo().GetByMessageId(messageId)
	if mdl == nil {
		customLogger.WarnWithData("Message log not found by message ID", map[string]interface{}{
			"component":  "MessageLogsService",
			"function":   "UpdateDeliveryStatus",
			"message_id": messageId,
		})
		return common.ERR_NOT_FOUND
	}

	mdl.Status = status
	if deliveredAt != nil {
		mdl.DeliveredAt = deliveredAt
	}
	if readAt != nil {
		mdl.ReadAt = readAt
	}
	if failedAt != nil {
		mdl.FailedAt = failedAt
	}
	if errorMessage != "" {
		mdl.ErrorMessage = errorMessage
	}

	err := messageLog.GetRepo().Update(mdl)
	if err != nil {
		customLogger.ErrorWithData("Failed to update delivery status", map[string]interface{}{
			"component":  "MessageLogsService",
			"function":   "UpdateDeliveryStatus",
			"message_id": messageId,
			"error":      err.Error,
		})
		return common.ERR_UNABLE_TO_CREATE
	}

	customLogger.InfoWithData("Delivery status updated", map[string]interface{}{
		"component":  "MessageLogsService",
		"function":   "UpdateDeliveryStatus",
		"message_id": messageId,
		"status":     status,
	})
	return nil
}

func (o *MessageLogsSvcImpl) toDTO(mdl *messageLog.MessageLog) *dtos.MessageLog {
	if mdl == nil {
		return nil
	}

	dto := &dtos.MessageLog{
		ID:             mdl.ID,
		ClientId:       mdl.ClientId,
		SenderId:       mdl.SenderId,
		CampaignId:     mdl.CampaignId,
		Type:           mdl.Type,
		Direction:      mdl.Direction,
		RecipientType:  mdl.RecipientType,
		RecipientValue: mdl.RecipientValue,
		TemplateId:     mdl.TemplateId,
		MessageId:      mdl.MessageId,
		SessionId:      mdl.SessionId,
		MessageContent: mdl.MessageContent,
		Status:         mdl.Status,
		Cost:           mdl.Cost,
		ErrorMessage:   mdl.ErrorMessage,
		Metadata:       unmarshalMetadata(mdl.Metadata),
		CreatedAt:      mdl.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      mdl.UpdatedAt.Format(time.RFC3339),
	}

	// Populate sender name
	if mdl.SenderId != "" {
		senderMdl := sender.GetRepo().GetById(mdl.SenderId)
		if senderMdl != nil {
			dto.SenderName = senderMdl.Name
		}
	}

	// Populate campaign name
	if mdl.CampaignId != "" {
		campaignMdl := campaign.GetRepo().GetById(mdl.CampaignId)
		if campaignMdl != nil {
			dto.CampaignName = campaignMdl.Name
		}
	}

	// Populate template name
	if mdl.TemplateId != "" {
		templateMdl := waTemplate.GetRepo().GetById(mdl.TemplateId)
		if templateMdl != nil {
			dto.TemplateName = templateMdl.Name
		}
	}

	if mdl.SentAt != nil {
		dto.SentAt = mdl.SentAt.Format(time.RFC3339)
	}
	if mdl.DeliveredAt != nil {
		dto.DeliveredAt = mdl.DeliveredAt.Format(time.RFC3339)
	}
	if mdl.ReadAt != nil {
		dto.ReadAt = mdl.ReadAt.Format(time.RFC3339)
	}
	if mdl.FailedAt != nil {
		dto.FailedAt = mdl.FailedAt.Format(time.RFC3339)
	} else if mdl.Status == "failed" {
		// If status is failed but FailedAt is nil, use UpdatedAt
		dto.FailedAt = mdl.UpdatedAt.Format(time.RFC3339)
	}

	return dto
}
