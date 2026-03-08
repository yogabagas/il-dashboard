package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/config"
	"github.com/oklog/ulid/v2"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/campaign"
	"gitlab.com/anti_metter/switching_common/campaignRecipient"
	"gitlab.com/anti_metter/switching_common/campaignTarget"
	"gitlab.com/anti_metter/switching_common/contact"
	"gitlab.com/anti_metter/switching_common/contactGroupMember"
	"gitlab.com/anti_metter/switching_common/messageLog"
	"gitlab.com/anti_metter/switching_common/sender"
	"gitlab.com/anti_metter/switching_common/waTemplate"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
)

type CampaignsService interface {
	Create(req dtos.CampaignReq, authInfo auth.AuthInfo) (*dtos.Campaign, *gocom.CodedError)
	Update(campaignId string, req dtos.CampaignUpdateReq, authInfo auth.AuthInfo) (*dtos.Campaign, *gocom.CodedError)
	GetById(campaignId string, authInfo auth.AuthInfo) (*dtos.Campaign, *gocom.CodedError)
	Search(filter, clientId, senderId, campaignType, status, dateFrom, dateTo string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.Campaign, bool, int)
	Delete(campaignId string, authInfo auth.AuthInfo) *gocom.CodedError
	GetRecipients(campaignId string, status string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.CampaignRecipient, bool, int)
	GetStats(campaignId string, authInfo auth.AuthInfo) (*dtos.CampaignStats, *gocom.CodedError)
	GenerateRecipients(campaignId string, authInfo auth.AuthInfo) (int, *gocom.CodedError)
	Start(campaignId string, authInfo auth.AuthInfo) (*dtos.Campaign, *gocom.CodedError)
	Pause(campaignId string, authInfo auth.AuthInfo) (*dtos.Campaign, *gocom.CodedError)
	Resume(campaignId string, authInfo auth.AuthInfo) (*dtos.Campaign, *gocom.CodedError)
	GetRecipientByMessageId(messageId string) (*dtos.CampaignRecipient, *gocom.CodedError)
	UpdateRecipient(recipient *dtos.CampaignRecipient) *gocom.CodedError
}

type CampaignsSvcImpl struct{}

var campaignsService CampaignsService
var onceCampaignsService sync.Once
var recipientUpdateMutex sync.Map // Map of recipient ID to mutex for preventing race conditions

func GetCampaignsService() CampaignsService {
	onceCampaignsService.Do(func() {
		campaignsService = &CampaignsSvcImpl{}
	})
	return campaignsService
}

func normalizeParameters(p interface{}) string {
	if p == nil {
		return ""
	}

	switch v := p.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		// If already JSON (object or array), return as-is
		if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
			return v
		}
		parts := strings.Split(v, ",")
		res := make([]string, 0, len(parts))
		for _, s := range parts {
			t := strings.TrimSpace(s)
			if t != "" {
				res = append(res, t)
			}
		}
		b, _ := json.Marshal(res)
		return string(b)
	case []string:
		b, _ := json.Marshal(v)
		return string(b)
	case []interface{}:
		// Try to coerce to []string
		res := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				res = append(res, s)
			}
		}
		if len(res) > 0 {
			b, _ := json.Marshal(res)
			return string(b)
		}
		b, _ := json.Marshal(v)
		return string(b)
	case map[string]interface{}:
		b, _ := json.Marshal(v)
		return string(b)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func parseParameters(p string) map[string]interface{} {
	if p == "" {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(p), &result); err != nil {
		return nil
	}
	return result
}

func (o *CampaignsSvcImpl) Create(req dtos.CampaignReq, authInfo auth.AuthInfo) (*dtos.Campaign, *gocom.CodedError) {
	customLogger.InfoWithData("Create campaign started", map[string]interface{}{
		"component": "CampaignsService",
		"function":  "Create",
		"name":      req.Name,
		"type":      req.Type,
		"client_id": authInfo.ClientId,
	})

	// Validation
	if req.Name == "" {
		customLogger.WarnWithData("Name is required", map[string]interface{}{
			"component": "CampaignsService",
			"function":  "Create",
			"error":     "missing_name",
		})
		return nil, common.ERR_INVALID_REQUEST
	}

	if req.Type != "whatsapp" && req.Type != "sms" && req.Type != "email" {
		customLogger.WarnWithData("Invalid type", map[string]interface{}{
			"component": "CampaignsService",
			"function":  "Create",
			"type":      req.Type,
		})
		return nil, gocom.NewError(400, "Type must be: whatsapp, sms, or email")
	}

	if req.SenderId == "" {
		customLogger.WarnWithData("SenderID is required", map[string]interface{}{
			"component": "CampaignsService",
			"function":  "Create",
			"error":     "missing_sender_id",
		})
		return nil, common.ERR_INVALID_REQUEST
	}

	if req.TemplateId == "" {
		customLogger.WarnWithData("TemplateID is required", map[string]interface{}{
			"component": "CampaignsService",
			"function":  "Create",
			"error":     "missing_template_id",
		})
		return nil, common.ERR_INVALID_REQUEST
	}

	if len(req.Targets) == 0 {
		customLogger.WarnWithData("At least one target is required", map[string]interface{}{
			"component": "CampaignsService",
			"function":  "Create",
			"error":     "missing_targets",
		})
		return nil, gocom.NewError(400, "At least one target is required")
	}

	// Verify sender exists and belongs to client
	senderMdl := sender.GetRepo().GetById(req.SenderId)
	if senderMdl == nil {
		customLogger.WarnWithData("Sender not found", map[string]interface{}{
			"component": "CampaignsService",
			"function":  "Create",
			"sender_id": req.SenderId,
		})
		return nil, gocom.NewError(404, "Sender not found")
	}
	if senderMdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Sender unauthorized", map[string]interface{}{
			"component":        "CampaignsService",
			"function":         "Create",
			"sender_client_id": senderMdl.ClientId,
			"auth_client_id":   authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Verify sender type matches campaign type
	if senderMdl.Type != req.Type {
		customLogger.WarnWithData("Sender type mismatch", map[string]interface{}{
			"component":     "CampaignsService",
			"function":      "Create",
			"campaign_type": req.Type,
			"sender_type":   senderMdl.Type,
		})
		return nil, gocom.NewError(400, "Sender type does not match campaign type")
	}

	// Verify template exists and belongs to client
	templateMdl := waTemplate.GetRepo().GetById(req.TemplateId)
	if templateMdl == nil {
		customLogger.WarnWithData("Template not found", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Create",
			"template_id": req.TemplateId,
		})
		return nil, gocom.NewError(404, "Template not found")
	}
	if templateMdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Template unauthorized", map[string]interface{}{
			"component":          "CampaignsService",
			"function":           "Create",
			"template_client_id": templateMdl.ClientId,
			"auth_client_id":     authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Parse scheduled time if provided
	var scheduledAt *time.Time
	if req.ScheduledAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.ScheduledAt)
		if err != nil {
			customLogger.WarnWithData("Invalid scheduled time", map[string]interface{}{
				"component":    "CampaignsService",
				"function":     "Create",
				"scheduled_at": req.ScheduledAt,
				"error":        err.Error(),
			})
			return nil, gocom.NewError(400, "Invalid scheduled time format. Use RFC3339")
		}
		scheduledAt = &parsed
	}

	// Create campaign
	campaignMdl := &campaign.Campaign{
		ID:           ulid.Make().String(),
		ClientId:     authInfo.ClientId,
		SenderId:     req.SenderId,
		Name:         req.Name,
		Type:         req.Type,
		TemplateId:   req.TemplateId,
		Status:       "draft",
		ScheduledAt:  scheduledAt,
		BatchSize:    req.BatchSize,
		DelaySeconds: req.DelaySeconds,
		Parameters:   normalizeParameters(req.Parameters),
		CreatedBy:    authInfo.UserId,
		UpdatedBy:    authInfo.UserId,
	}

	if campaignMdl.BatchSize <= 0 {
		campaignMdl.BatchSize = 50 // Default batch size
	}
	if campaignMdl.DelaySeconds <= 0 {
		campaignMdl.DelaySeconds = 2 // Default delay
	}

	err := campaign.GetRepo().Create(campaignMdl).Error
	if err != nil {
		customLogger.ErrorWithData("Failed to create campaign", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Create",
			"campaign_id": campaignMdl.ID,
			"error":       err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	// Create campaign recipients based on targets
	recipients := []*campaignRecipient.CampaignRecipient{}
	contactMap := make(map[string]bool) // To avoid duplicates

	for _, target := range req.Targets {
		// Retrieve contacts and create campaign recipients based on target type
		switch target.TargetType {
		case "contact":
			// Single contact
			contactMdl := contact.GetRepo().GetById(target.TargetId)
			if contactMdl != nil && contactMdl.Status == "active" {
				if !contactMap[contactMdl.ID] {
					recipient := o.buildRecipient(campaignMdl, contactMdl)
					if recipient != nil {
						recipients = append(recipients, recipient)
						contactMap[contactMdl.ID] = true
					}
				}
			} else {
				customLogger.WarnWithData("Contact not found or inactive", map[string]interface{}{
					"component":  "CampaignsService",
					"function":   "Create",
					"contact_id": target.TargetId,
				})
			}

		case "group":
			// All contacts in group
			members, _, _ := contactGroupMember.GetRepo().GetContactsByGroupId(target.TargetId, 1, 10000) // Get all
			customLogger.InfoWithData("Retrieved contacts from group", map[string]interface{}{
				"component":    "CampaignsService",
				"function":     "Create",
				"group_id":     target.TargetId,
				"member_count": len(members),
			})

			for _, contactMdl := range members {
				if contactMdl.Status == "active" && !contactMap[contactMdl.ID] {
					recipient := o.buildRecipient(campaignMdl, &contactMdl)
					if recipient != nil {
						recipients = append(recipients, recipient)
						contactMap[contactMdl.ID] = true
					}
				}
			}

		case "all":
			// All active contacts for this client
			contacts, _, _ := contact.GetRepo().Search("", authInfo.ClientId, "active", "", 1, 10000) // Get all active
			customLogger.InfoWithData("Retrieved all contacts for client", map[string]interface{}{
				"component":     "CampaignsService",
				"function":      "Create",
				"client_id":     authInfo.ClientId,
				"contact_count": len(contacts),
			})

			for _, contactMdl := range contacts {
				if !contactMap[contactMdl.ID] {
					recipient := o.buildRecipient(campaignMdl, &contactMdl)
					if recipient != nil {
						recipients = append(recipients, recipient)
						contactMap[contactMdl.ID] = true
					}
				}
			}
		}
	}

	// Insert campaign recipients
	if len(recipients) > 0 {
		customLogger.InfoWithData("Creating campaign recipients", map[string]interface{}{
			"component":       "CampaignsService",
			"function":        "Create",
			"campaign_id":     campaignMdl.ID,
			"recipient_count": len(recipients),
		})

		// Insert recipients in batches for better performance
		batchSize := 100
		for i := 0; i < len(recipients); i += batchSize {
			end := i + batchSize
			if end > len(recipients) {
				end = len(recipients)
			}
			batch := recipients[i:end]

			for _, recipient := range batch {
				err := campaignRecipient.GetRepo().Create(recipient).Error
				if err != nil {
					customLogger.ErrorWithData("Failed to create recipient", map[string]interface{}{
						"component":   "CampaignsService",
						"function":    "Create",
						"campaign_id": campaignMdl.ID,
						"contact_id":  recipient.ContactId,
						"error":       err.Error(),
					})
				}
			}
		}

		// Update campaign total recipients count
		campaignMdl.TotalRecipients = len(recipients)
		_ = campaign.GetRepo().Update(campaignMdl)
	}

	customLogger.InfoWithData("Campaign created successfully", map[string]interface{}{
		"component":       "CampaignsService",
		"function":        "Create",
		"campaign_id":     campaignMdl.ID,
		"name":            campaignMdl.Name,
		"recipient_count": len(recipients),
	})
	return o.toDTO(campaignMdl, senderMdl, templateMdl), nil
}

func (o *CampaignsSvcImpl) Update(campaignId string, req dtos.CampaignUpdateReq, authInfo auth.AuthInfo) (*dtos.Campaign, *gocom.CodedError) {
	customLogger.InfoWithData("Update campaign started", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "Update",
		"campaign_id": campaignId,
		"client_id":   authInfo.ClientId,
	})

	// Get existing campaign
	mdl := campaign.GetRepo().GetById(campaignId)
	if mdl == nil {
		customLogger.WarnWithData("Campaign not found", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Update",
			"campaign_id": campaignId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if mdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":          "CampaignsService",
			"function":           "Update",
			"campaign_client_id": mdl.ClientId,
			"auth_client_id":     authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Cannot update campaign if running or completed
	if mdl.Status == "running" || mdl.Status == "completed" {
		customLogger.WarnWithData("Cannot update running/completed campaign", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Update",
			"campaign_id": campaignId,
			"status":      mdl.Status,
		})
		return nil, gocom.NewError(400, "Cannot update campaign that is running or completed")
	}

	// Update fields
	if req.Name != "" {
		mdl.Name = req.Name
	}
	if req.Status != "" {
		if req.Status != "draft" && req.Status != "scheduled" && req.Status != "paused" && req.Status != "cancelled" {
			customLogger.WarnWithData("Invalid status", map[string]interface{}{
				"component": "CampaignsService",
				"function":  "Update",
				"status":    req.Status,
			})
			return nil, gocom.NewError(400, "Invalid status")
		}
		mdl.Status = req.Status
	}
	if req.ScheduledAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.ScheduledAt)
		if err != nil {
			customLogger.WarnWithData("Invalid scheduled time", map[string]interface{}{
				"component":    "CampaignsService",
				"function":     "Update",
				"scheduled_at": req.ScheduledAt,
				"error":        err.Error(),
			})
			return nil, gocom.NewError(400, "Invalid scheduled time format. Use RFC3339")
		}
		mdl.ScheduledAt = &parsed
	}
	if req.BatchSize > 0 {
		mdl.BatchSize = req.BatchSize
	}
	if req.DelaySeconds > 0 {
		mdl.DelaySeconds = req.DelaySeconds
	}
	if req.Parameters != nil {
		mdl.Parameters = normalizeParameters(req.Parameters)
	}
	mdl.UpdatedBy = authInfo.UserId

	err := campaign.GetRepo().Update(mdl)
	if err != nil {
		customLogger.ErrorWithData("Failed to update campaign", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Update",
			"campaign_id": campaignId,
			"error":       err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	// Get sender and template for DTO
	senderMdl := sender.GetRepo().GetById(mdl.SenderId)
	templateMdl := waTemplate.GetRepo().GetById(mdl.TemplateId)

	customLogger.InfoWithData("Campaign updated successfully", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "Update",
		"campaign_id": campaignId,
		"name":        mdl.Name,
	})
	return o.toDTO(mdl, senderMdl, templateMdl), nil
}

func (o *CampaignsSvcImpl) GetById(campaignId string, authInfo auth.AuthInfo) (*dtos.Campaign, *gocom.CodedError) {
	customLogger.InfoWithData("GetById started", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "GetById",
		"campaign_id": campaignId,
	})

	mdl := campaign.GetRepo().GetById(campaignId)
	if mdl == nil {
		customLogger.WarnWithData("Campaign not found", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "GetById",
			"campaign_id": campaignId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if mdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":          "CampaignsService",
			"function":           "GetById",
			"campaign_client_id": mdl.ClientId,
			"auth_client_id":     authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Get sender and template
	senderMdl := sender.GetRepo().GetById(mdl.SenderId)
	templateMdl := waTemplate.GetRepo().GetById(mdl.TemplateId)

	// Get campaign targets
	targetMdls := campaignTarget.GetRepo().GetByCampaignId(campaignId)

	customLogger.InfoWithData("GetById success", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "GetById",
		"campaign_id": mdl.ID,
		"name":        mdl.Name,
	})
	return o.toDTO(mdl, senderMdl, templateMdl, targetMdls), nil
}

func (o *CampaignsSvcImpl) Search(filter, clientId, senderId, campaignType, status, dateFrom, dateTo string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.Campaign, bool, int) {
	customLogger.DebugWithData("Search started", map[string]interface{}{
		"component": "CampaignsService",
		"function":  "Search",
		"filter":    filter,
		"type":      campaignType,
		"status":    status,
		"date_from": dateFrom,
		"date_to":   dateTo,
		"page":      pageNo,
		"rows":      rowPerPage,
	})

	if rowPerPage <= 0 {
		rowPerPage = config.GetInt(constans.StaticDefaultEnv)
	}
	if pageNo <= 0 {
		pageNo = 1
	}

	if common.PROVIDER_ID != authInfo.ClientId {
		clientId = authInfo.ClientId
	}

	// Parse dates
	var dateFromParsed, dateToParsed time.Time
	if dateFrom != "" {
		parsed, err := time.Parse("2006-01-02", dateFrom)
		if err == nil {
			dateFromParsed = parsed
		}
	}
	if dateTo != "" {
		parsed, err := time.Parse("2006-01-02", dateTo)
		if err == nil {
			// Add 1 day to include the whole day
			parsed = parsed.Add(24 * time.Hour)
			dateToParsed = parsed
		}
	}

	mdls, haveNext, count := campaign.GetRepo().Search(filter, clientId, senderId, campaignType, status, dateFromParsed, dateToParsed, pageNo, rowPerPage)

	ret := make([]*dtos.Campaign, len(mdls))
	for i, mdl := range mdls {
		senderMdl := sender.GetRepo().GetById(mdl.SenderId)
		templateMdl := waTemplate.GetRepo().GetById(mdl.TemplateId)
		ret[i] = o.toDTO(&mdl, senderMdl, templateMdl)
	}

	customLogger.InfoWithData("Search completed", map[string]interface{}{
		"component": "CampaignsService",
		"function":  "Search",
		"count":     len(ret),
		"have_next": haveNext,
		"total":     count,
	})
	return ret, haveNext, int(count)
}

func (o *CampaignsSvcImpl) Delete(campaignId string, authInfo auth.AuthInfo) *gocom.CodedError {
	customLogger.InfoWithData("Delete campaign started", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "Delete",
		"campaign_id": campaignId,
	})

	// Get existing campaign
	mdl := campaign.GetRepo().GetById(campaignId)
	if mdl == nil {
		customLogger.WarnWithData("Campaign not found", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Delete",
			"campaign_id": campaignId,
		})
		return common.ERR_NOT_FOUND
	}

	// Check permission
	if mdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":          "CampaignsService",
			"function":           "Delete",
			"campaign_client_id": mdl.ClientId,
			"auth_client_id":     authInfo.ClientId,
		})
		return common.ERR_NOT_ALLOWED
	}

	// Cannot delete running campaign
	if mdl.Status == "running" {
		customLogger.WarnWithData("Cannot delete running campaign", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Delete",
			"campaign_id": campaignId,
			"status":      mdl.Status,
		})
		return gocom.NewError(400, "Cannot delete running campaign")
	}

	// Delete recipients
	err := campaignRecipient.GetRepo().DeleteByCampaignId(campaignId)
	if err != nil {
		customLogger.ErrorWithData("Failed to delete recipients", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Delete",
			"campaign_id": campaignId,
			"error":       err.Error(),
		})
		return common.ERR_UNABLE_TO_DELETE
	}

	// Delete targets
	err = campaignTarget.GetRepo().DeleteByCampaignId(campaignId)
	if err != nil {
		customLogger.ErrorWithData("Failed to delete targets", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Delete",
			"campaign_id": campaignId,
			"error":       err.Error(),
		})
		return common.ERR_UNABLE_TO_DELETE
	}

	// Delete campaign
	err = campaign.GetRepo().Delete(campaignId)
	if err != nil {
		customLogger.ErrorWithData("Failed to delete campaign", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Delete",
			"campaign_id": campaignId,
			"error":       err.Error(),
		})
		return common.ERR_UNABLE_TO_DELETE
	}

	customLogger.InfoWithData("Campaign deleted successfully", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "Delete",
		"campaign_id": campaignId,
	})
	return nil
}

func (o *CampaignsSvcImpl) GetRecipients(campaignId string, status string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.CampaignRecipient, bool, int) {
	customLogger.DebugWithData("GetRecipients started", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "GetRecipients",
		"campaign_id": campaignId,
		"status":      status,
		"page":        pageNo,
		"rows":        rowPerPage,
	})

	// Verify campaign exists and belongs to client
	campaignMdl := campaign.GetRepo().GetById(campaignId)
	if campaignMdl == nil {
		customLogger.WarnWithData("Campaign not found", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "GetRecipients",
			"campaign_id": campaignId,
		})
		return nil, false, 0
	}
	if campaignMdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":          "CampaignsService",
			"function":           "GetRecipients",
			"campaign_client_id": campaignMdl.ClientId,
			"auth_client_id":     authInfo.ClientId,
		})
		return nil, false, 0
	}

	if rowPerPage <= 0 {
		rowPerPage = config.GetInt(constans.StaticDefaultEnv)
	}
	if pageNo <= 0 {
		pageNo = 1
	}

	mdls, haveNext, count := campaignRecipient.GetRepo().SearchByCampaignId(campaignId, status, pageNo, rowPerPage)

	ret := make([]*dtos.CampaignRecipient, len(mdls))
	for i, mdl := range mdls {
		ret[i] = o.toRecipientDTO(&mdl)
	}

	customLogger.InfoWithData("GetRecipients completed", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "GetRecipients",
		"campaign_id": campaignId,
		"count":       len(ret),
		"have_next":   haveNext,
		"total":       count,
	})
	return ret, haveNext, int(count)
}

func (o *CampaignsSvcImpl) GetStats(campaignId string, authInfo auth.AuthInfo) (*dtos.CampaignStats, *gocom.CodedError) {
	customLogger.InfoWithData("GetStats started", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "GetStats",
		"campaign_id": campaignId,
	})

	// Verify campaign exists and belongs to client
	campaignMdl := campaign.GetRepo().GetById(campaignId)
	if campaignMdl == nil {
		customLogger.WarnWithData("Campaign not found", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "GetStats",
			"campaign_id": campaignId,
		})
		return nil, common.ERR_NOT_FOUND
	}
	if campaignMdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":          "CampaignsService",
			"function":           "GetStats",
			"campaign_client_id": campaignMdl.ClientId,
			"auth_client_id":     authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	stats := &dtos.CampaignStats{
		CampaignId:      campaignId,
		TotalRecipients: campaignMdl.TotalRecipients,
		TotalSent:       campaignMdl.TotalSent,
		TotalDelivered:  campaignMdl.TotalDelivered,
		TotalFailed:     campaignMdl.TotalFailed,
		TotalRead:       campaignMdl.TotalRead,
	}

	// Calculate rates
	if stats.TotalRecipients > 0 {
		stats.SentRate = float64(stats.TotalSent) / float64(stats.TotalRecipients) * 100
		stats.DeliveredRate = float64(stats.TotalDelivered) / float64(stats.TotalRecipients) * 100
		stats.ReadRate = float64(stats.TotalRead) / float64(stats.TotalRecipients) * 100
		stats.FailedRate = float64(stats.TotalFailed) / float64(stats.TotalRecipients) * 100
	}

	customLogger.InfoWithData("GetStats success", map[string]interface{}{
		"component":        "CampaignsService",
		"function":         "GetStats",
		"campaign_id":      campaignId,
		"total_recipients": stats.TotalRecipients,
		"total_sent":       stats.TotalSent,
	})
	return stats, nil
}

func (o *CampaignsSvcImpl) GenerateRecipients(campaignId string, authInfo auth.AuthInfo) (int, *gocom.CodedError) {
	customLogger.InfoWithData("GenerateRecipients started", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "GenerateRecipients",
		"campaign_id": campaignId,
	})

	// Get campaign
	campaignMdl := campaign.GetRepo().GetById(campaignId)
	if campaignMdl == nil {
		customLogger.WarnWithData("Campaign not found", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "GenerateRecipients",
			"campaign_id": campaignId,
		})
		return 0, common.ERR_NOT_FOUND
	}
	if campaignMdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":          "CampaignsService",
			"function":           "GenerateRecipients",
			"campaign_client_id": campaignMdl.ClientId,
			"auth_client_id":     authInfo.ClientId,
		})
		return 0, common.ERR_NOT_ALLOWED
	}

	// Get targets
	targets := campaignTarget.GetRepo().GetByCampaignId(campaignId)
	if len(targets) == 0 {
		customLogger.WarnWithData("No targets found", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "GenerateRecipients",
			"campaign_id": campaignId,
		})
		return 0, gocom.NewError(400, "No targets found for campaign")
	}

	// Delete existing recipients
	err := campaignRecipient.GetRepo().DeleteByCampaignId(campaignId)
	if err != nil {
		customLogger.ErrorWithData("Failed to delete existing recipients", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "GenerateRecipients",
			"campaign_id": campaignId,
			"error":       err.Error(),
		})
		return 0, common.ERR_UNABLE_TO_DELETE
	}

	// Generate recipients from targets - collect first, then batch insert
	recipients := []*campaignRecipient.CampaignRecipient{}
	contactMap := make(map[string]bool) // To avoid duplicates

	for _, target := range targets {
		switch target.TargetType {
		case "contact":
			// Single contact
			contactMdl := contact.GetRepo().GetById(target.TargetId)
			if contactMdl != nil && contactMdl.Status == "active" {
				if !contactMap[contactMdl.ID] {
					recipient := o.buildRecipient(campaignMdl, contactMdl)
					if recipient != nil {
						recipients = append(recipients, recipient)
						contactMap[contactMdl.ID] = true
					}
				}
			}

		case "group":
			// All contacts in group
			members, _, _ := contactGroupMember.GetRepo().GetContactsByGroupId(target.TargetId, 1, 10000) // Get all
			for _, contactMdl := range members {
				if contactMdl.Status == "active" && !contactMap[contactMdl.ID] {
					recipient := o.buildRecipient(campaignMdl, &contactMdl)
					if recipient != nil {
						recipients = append(recipients, recipient)
						contactMap[contactMdl.ID] = true
					}
				}
			}

		case "all":
			// All active contacts
			contacts, _, _ := contact.GetRepo().Search("", authInfo.ClientId, "active", "", 1, 10000) // Get all
			for _, contactMdl := range contacts {
				if !contactMap[contactMdl.ID] {
					recipient := o.buildRecipient(campaignMdl, &contactMdl)
					if recipient != nil {
						recipients = append(recipients, recipient)
						contactMap[contactMdl.ID] = true
					}
				}
			}
		}
	}

	recipientCount := len(recipients)

	// Check balance before finalizing
	messageCost := GetBillingService().CalculateMessageCost(campaignMdl.Type, authInfo.ClientId)
	estimatedCost := float64(recipientCount) * messageCost
	customLogger.InfoWithData("Checking balance", map[string]interface{}{
		"component":       "CampaignsService",
		"function":        "GenerateRecipients",
		"recipient_count": recipientCount,
		"message_cost":    messageCost,
		"estimated_cost":  estimatedCost,
	})

	canSend, balanceErr := GetBillingService().CanClientSend(authInfo.ClientId, estimatedCost)
	if balanceErr != nil {
		customLogger.ErrorWithData("Failed to check balance", map[string]interface{}{
			"component": "CampaignsService",
			"function":  "GenerateRecipients",
			"client_id": authInfo.ClientId,
			"error":     balanceErr.Error(),
		})
		// Continue anyway, just warn
	} else if !canSend {
		customLogger.WarnWithData("Insufficient balance", map[string]interface{}{
			"component":      "CampaignsService",
			"function":       "GenerateRecipients",
			"client_id":      authInfo.ClientId,
			"estimated_cost": estimatedCost,
		})
		return 0, gocom.NewError(400, "Insufficient balance. Estimated cost: Rp "+fmt.Sprintf("%.2f", estimatedCost))
	}

	// Batch insert recipients asynchronously
	customLogger.InfoWithData("Starting batch insert", map[string]interface{}{
		"component":       "CampaignsService",
		"function":        "GenerateRecipients",
		"recipient_count": recipientCount,
		"batch_size":      campaignMdl.BatchSize,
	})
	go o.batchInsertRecipients(recipients, campaignMdl)

	// Update campaign total recipients
	campaignMdl.TotalRecipients = recipientCount
	err = campaign.GetRepo().Update(campaignMdl)
	if err != nil {
		customLogger.ErrorWithData("Failed to update campaign", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "GenerateRecipients",
			"campaign_id": campaignId,
			"error":       err.Error(),
		})
		return recipientCount, common.ERR_UNABLE_TO_CREATE
	}

	customLogger.InfoWithData("Recipients generated successfully", map[string]interface{}{
		"component":       "CampaignsService",
		"function":        "GenerateRecipients",
		"campaign_id":     campaignId,
		"recipient_count": recipientCount,
		"estimated_cost":  estimatedCost,
	})
	return recipientCount, nil
}

// buildRecipient creates a recipient object without inserting to database
func (o *CampaignsSvcImpl) buildRecipient(campaignMdl *campaign.Campaign, contactMdl *contact.Contact) *campaignRecipient.CampaignRecipient {
	recipientType := ""
	recipientValue := ""

	// Determine recipient type based on campaign type
	switch campaignMdl.Type {
	case "whatsapp", "sms":
		recipientType = "phone"
		recipientValue = contactMdl.Phone
	case "email":
		recipientType = "email"
		recipientValue = contactMdl.Email
	}

	if recipientValue == "" {
		customLogger.DebugWithData("No valid recipient value", map[string]interface{}{
			"component":     "CampaignsService",
			"function":      "buildRecipient",
			"contact_id":    contactMdl.ID,
			"campaign_type": campaignMdl.Type,
		})
		return nil
	}

	recipient := &campaignRecipient.CampaignRecipient{
		ID:             ulid.Make().String(),
		CampaignId:     campaignMdl.ID,
		ContactId:      contactMdl.ID,
		RecipientType:  recipientType,
		RecipientValue: recipientValue,
		Status:         "pending",
		Parameters:     campaignMdl.Parameters, // Copy campaign parameters
	}

	return recipient
}

// batchInsertRecipients inserts recipients in batches asynchronously
func (o *CampaignsSvcImpl) batchInsertRecipients(recipients []*campaignRecipient.CampaignRecipient, camp *campaign.Campaign) {
	// Use campaign batch size, default to 100 if not set
	batchSize := camp.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	totalRecipients := len(recipients)
	successCount := 0
	failCount := 0

	customLogger.InfoWithData("Batch insert started", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "batchInsertRecipients",
		"campaign_id": camp.ID,
		"total":       totalRecipients,
		"batch_size":  batchSize,
	})

	for i := 0; i < totalRecipients; i += batchSize {
		end := i + batchSize
		if end > totalRecipients {
			end = totalRecipients
		}

		batch := recipients[i:end]

		// Insert batch
		for _, recipient := range batch {
			err := campaignRecipient.GetRepo().Create(recipient).Error
			if err != nil {
				failCount++
				customLogger.ErrorWithData("Failed to insert recipient", map[string]interface{}{
					"component":    "CampaignsService",
					"function":     "batchInsertRecipients",
					"recipient_id": recipient.ID,
					"campaign_id":  camp.ID,
					"error":        err.Error(),
				})
			} else {
				successCount++
			}
		}

		customLogger.DebugWithData("Batch inserted", map[string]interface{}{
			"component":     "CampaignsService",
			"function":      "batchInsertRecipients",
			"campaign_id":   camp.ID,
			"batch_number":  (i / batchSize) + 1,
			"batch_size":    len(batch),
			"success_count": successCount,
			"fail_count":    failCount,
		})
	}

	customLogger.InfoWithData("Batch insert completed", map[string]interface{}{
		"component":     "CampaignsService",
		"function":      "batchInsertRecipients",
		"campaign_id":   camp.ID,
		"total":         totalRecipients,
		"success_count": successCount,
		"fail_count":    failCount,
	})
}

func (o *CampaignsSvcImpl) toDTO(mdl *campaign.Campaign, senderMdl *sender.Sender, templateMdl *waTemplate.WATemplate, targetMdls ...[]campaignTarget.CampaignTarget) *dtos.Campaign {
	if mdl == nil {
		return nil
	}

	// when building DTO, try to unmarshal Parameters JSON string into an interface{}
	dto := &dtos.Campaign{
		ID:              mdl.ID,
		ClientId:        mdl.ClientId,
		SenderId:        mdl.SenderId,
		Name:            mdl.Name,
		Type:            mdl.Type,
		TemplateId:      mdl.TemplateId,
		Status:          mdl.Status,
		TotalRecipients: mdl.TotalRecipients,
		TotalSent:       mdl.TotalSent,
		TotalDelivered:  mdl.TotalDelivered,
		TotalFailed:     mdl.TotalFailed,
		TotalRead:       mdl.TotalRead,
		BatchSize:       mdl.BatchSize,
		DelaySeconds:    mdl.DelaySeconds,
		CreatedBy:       mdl.CreatedBy,
		UpdatedBy:       mdl.UpdatedBy,
		CreatedAt:       mdl.CreatedAt,
		UpdatedAt:       mdl.UpdatedAt,
	}
	dto.Parameters = parseParameters(mdl.Parameters)

	if mdl.ScheduledAt != nil {
		dto.ScheduledAt = mdl.ScheduledAt.Format(time.RFC3339)
	}
	if mdl.StartedAt != nil {
		dto.StartedAt = mdl.StartedAt.Format(time.RFC3339)
	}
	if mdl.CompletedAt != nil {
		dto.CompletedAt = mdl.CompletedAt.Format(time.RFC3339)
	}

	if senderMdl != nil {
		dto.SenderName = senderMdl.Name
	}
	if templateMdl != nil {
		dto.TemplateName = templateMdl.Name
	}

	// Convert campaign targets to DTO targets
	if len(targetMdls) > 0 && targetMdls[0] != nil {
		dto.Targets = make([]dtos.CampaignTarget, len(targetMdls[0]))
		for i, target := range targetMdls[0] {
			dto.Targets[i] = dtos.CampaignTarget{
				TargetType: target.TargetType,
				TargetId:   target.TargetId,
			}
		}
	}

	return dto
}

func (o *CampaignsSvcImpl) toRecipientDTO(mdl *campaignRecipient.CampaignRecipient) *dtos.CampaignRecipient {
	if mdl == nil {
		return nil
	}

	dto := &dtos.CampaignRecipient{
		ID:             mdl.ID,
		CampaignId:     mdl.CampaignId,
		ContactId:      mdl.ContactId,
		RecipientType:  mdl.RecipientType,
		RecipientValue: mdl.RecipientValue,
		Status:         mdl.Status,
		MessageId:      mdl.MessageId,
		Parameters:     parseParameters(mdl.Parameters),
		ErrorMessage:   mdl.ErrorMessage,
		RetryCount:     mdl.RetryCount,
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
	}

	// Get contact name
	contactMdl := contact.GetRepo().GetById(mdl.ContactId)
	if contactMdl != nil {
		dto.ContactName = contactMdl.Name
	}

	return dto
}

func (o *CampaignsSvcImpl) Start(campaignId string, authInfo auth.AuthInfo) (*dtos.Campaign, *gocom.CodedError) {
	customLogger.InfoWithData("Start campaign", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "Start",
		"campaign_id": campaignId,
	})

	// Get campaign
	mdl := campaign.GetRepo().GetById(campaignId)
	if mdl == nil {
		customLogger.WarnWithData("Campaign not found", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Start",
			"campaign_id": campaignId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if mdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":          "CampaignsService",
			"function":           "Start",
			"campaign_client_id": mdl.ClientId,
			"auth_client_id":     authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Can only start draft or scheduled campaigns
	if mdl.Status != "draft" && mdl.Status != "scheduled" {
		customLogger.WarnWithData("Cannot start campaign", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Start",
			"campaign_id": campaignId,
			"status":      mdl.Status,
		})
		return nil, gocom.NewError(400, fmt.Sprintf("Cannot start campaign with status: %s. Only draft or scheduled campaigns can be started.", mdl.Status))
	}

	// Check if campaign has recipients
	if mdl.TotalRecipients == 0 {
		customLogger.WarnWithData("Campaign has no recipients", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Start",
			"campaign_id": campaignId,
		})
		return nil, gocom.NewError(400, "Campaign has no recipients. Please generate recipients first.")
	}

	// Check balance
	messageCost := GetBillingService().CalculateMessageCost(mdl.Type, authInfo.ClientId)
	_, _, pendingCount := campaignRecipient.GetRepo().SearchByCampaignId(campaignId, "pending", 1, 1)
	estimatedCost := float64(pendingCount) * messageCost

	canSend, balanceErr := GetBillingService().CanClientSend(authInfo.ClientId, estimatedCost)
	if balanceErr != nil {
		customLogger.ErrorWithData("Failed to check balance", map[string]interface{}{
			"component": "CampaignsService",
			"function":  "Start",
			"client_id": authInfo.ClientId,
			"error":     balanceErr.Error(),
		})
		return nil, gocom.NewError(500, "Failed to check balance")
	}
	if !canSend {
		customLogger.WarnWithData("Insufficient balance", map[string]interface{}{
			"component":      "CampaignsService",
			"function":       "Start",
			"client_id":      authInfo.ClientId,
			"estimated_cost": estimatedCost,
		})
		return nil, gocom.NewError(400, fmt.Sprintf("Insufficient balance. Estimated cost: Rp %.2f", estimatedCost))
	}

	// Update campaign status
	now := time.Now()
	mdl.Status = "running"
	mdl.StartedAt = &now
	mdl.UpdatedBy = authInfo.UserId

	err := campaign.GetRepo().Update(mdl)
	if err != nil {
		customLogger.ErrorWithData("Failed to update campaign", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Start",
			"campaign_id": campaignId,
			"error":       err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	// Get sender and template for DTO
	senderMdl := sender.GetRepo().GetById(mdl.SenderId)
	templateMdl := waTemplate.GetRepo().GetById(mdl.TemplateId)

	customLogger.InfoWithData("Campaign started successfully", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "Start",
		"campaign_id": campaignId,
	})
	return o.toDTO(mdl, senderMdl, templateMdl), nil
}

func (o *CampaignsSvcImpl) Pause(campaignId string, authInfo auth.AuthInfo) (*dtos.Campaign, *gocom.CodedError) {
	customLogger.InfoWithData("Pause campaign", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "Pause",
		"campaign_id": campaignId,
	})

	// Get campaign
	mdl := campaign.GetRepo().GetById(campaignId)
	if mdl == nil {
		customLogger.WarnWithData("Campaign not found", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Pause",
			"campaign_id": campaignId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if mdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":          "CampaignsService",
			"function":           "Pause",
			"campaign_client_id": mdl.ClientId,
			"auth_client_id":     authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Can only pause running campaigns
	if mdl.Status != "running" {
		customLogger.WarnWithData("Cannot pause campaign", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Pause",
			"campaign_id": campaignId,
			"status":      mdl.Status,
		})
		return nil, gocom.NewError(400, fmt.Sprintf("Cannot pause campaign with status: %s. Only running campaigns can be paused.", mdl.Status))
	}

	// Update campaign status
	mdl.Status = "paused"
	mdl.UpdatedBy = authInfo.UserId

	err := campaign.GetRepo().Update(mdl)
	if err != nil {
		customLogger.ErrorWithData("Failed to update campaign", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Pause",
			"campaign_id": campaignId,
			"error":       err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	// Get sender and template for DTO
	senderMdl := sender.GetRepo().GetById(mdl.SenderId)
	templateMdl := waTemplate.GetRepo().GetById(mdl.TemplateId)

	customLogger.InfoWithData("Campaign paused successfully", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "Pause",
		"campaign_id": campaignId,
	})
	return o.toDTO(mdl, senderMdl, templateMdl), nil
}

func (o *CampaignsSvcImpl) Resume(campaignId string, authInfo auth.AuthInfo) (*dtos.Campaign, *gocom.CodedError) {
	customLogger.InfoWithData("Resume campaign", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "Resume",
		"campaign_id": campaignId,
	})

	// Get campaign
	mdl := campaign.GetRepo().GetById(campaignId)
	if mdl == nil {
		customLogger.WarnWithData("Campaign not found", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Resume",
			"campaign_id": campaignId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if mdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":          "CampaignsService",
			"function":           "Resume",
			"campaign_client_id": mdl.ClientId,
			"auth_client_id":     authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Can only resume paused campaigns
	if mdl.Status != "paused" {
		customLogger.WarnWithData("Cannot resume campaign", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Resume",
			"campaign_id": campaignId,
			"status":      mdl.Status,
		})
		return nil, gocom.NewError(400, fmt.Sprintf("Cannot resume campaign with status: %s. Only paused campaigns can be resumed.", mdl.Status))
	}

	// Check balance again
	messageCost := GetBillingService().CalculateMessageCost(mdl.Type, authInfo.ClientId)
	_, _, pendingCount := campaignRecipient.GetRepo().SearchByCampaignId(campaignId, "pending", 1, 1)
	estimatedCost := float64(pendingCount) * messageCost

	canSend, balanceErr := GetBillingService().CanClientSend(authInfo.ClientId, estimatedCost)
	if balanceErr != nil {
		customLogger.ErrorWithData("Failed to check balance", map[string]interface{}{
			"component": "CampaignsService",
			"function":  "Resume",
			"client_id": authInfo.ClientId,
			"error":     balanceErr.Error(),
		})
		return nil, gocom.NewError(500, "Failed to check balance")
	}
	if !canSend {
		customLogger.WarnWithData("Insufficient balance", map[string]interface{}{
			"component":      "CampaignsService",
			"function":       "Resume",
			"client_id":      authInfo.ClientId,
			"estimated_cost": estimatedCost,
		})
		return nil, gocom.NewError(400, fmt.Sprintf("Insufficient balance. Estimated cost: Rp %.2f", estimatedCost))
	}

	// Update campaign status
	mdl.Status = "running"
	mdl.UpdatedBy = authInfo.UserId

	err := campaign.GetRepo().Update(mdl)
	if err != nil {
		customLogger.ErrorWithData("Failed to update campaign", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "Resume",
			"campaign_id": campaignId,
			"error":       err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	// Get sender and template for DTO
	senderMdl := sender.GetRepo().GetById(mdl.SenderId)
	templateMdl := waTemplate.GetRepo().GetById(mdl.TemplateId)

	customLogger.InfoWithData("Campaign resumed successfully", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "Resume",
		"campaign_id": campaignId,
	})
	return o.toDTO(mdl, senderMdl, templateMdl), nil
}

// GetRecipientByMessageId gets a campaign recipient by WhatsApp message ID
func (o *CampaignsSvcImpl) GetRecipientByMessageId(messageId string) (*dtos.CampaignRecipient, *gocom.CodedError) {
	customLogger.DebugWithData("GetRecipientByMessageId started", map[string]interface{}{
		"component":  "CampaignsService",
		"function":   "GetRecipientByMessageId",
		"message_id": messageId,
	})

	// Try to get recipient directly with retry mechanism
	var mdl *campaignRecipient.CampaignRecipient
	mdl = campaignRecipient.GetRepo().GetByMessageId(messageId)
	if mdl != nil {
		customLogger.InfoWithData("Found recipient", map[string]interface{}{
			"component":    "CampaignsService",
			"function":     "GetRecipientByMessageId",
			"recipient_id": mdl.ID,
			"campaign_id":  mdl.CampaignId,
			"message_id":   messageId,
		})
		return o.toRecipientDTO(mdl), nil
	}

	// Fallback: get campaign_id and recipient_value from message_logs
	customLogger.InfoWithData("Fallback to message_logs", map[string]interface{}{
		"component":  "CampaignsService",
		"function":   "GetRecipientByMessageId",
		"message_id": messageId,
	})

	msgLog := messageLog.GetRepo().GetByMessageId(messageId)
	if msgLog == nil {
		customLogger.WarnWithData("Message log not found", map[string]interface{}{
			"component":  "CampaignsService",
			"function":   "GetRecipientByMessageId",
			"message_id": messageId,
			"error":      common.ERR_NOT_FOUND.Message,
		})
		return nil, common.ERR_NOT_FOUND
	}

	mdl = campaignRecipient.GetRepo().GetByMessageId(msgLog.ID)
	if mdl != nil {
		customLogger.InfoWithData("Found recipient by message id from message_logs", map[string]interface{}{
			"component":    "CampaignsService",
			"function":     "GetRecipientByMessageId",
			"recipient_id": mdl.ID,
			"campaign_id":  mdl.CampaignId,
			"message_id":   messageId,
		})
		return o.toRecipientDTO(mdl), nil
	}

	return nil, common.ERR_NOT_FOUND
}

// UpdateRecipient updates a campaign recipient
func (o *CampaignsSvcImpl) UpdateRecipient(recipient *dtos.CampaignRecipient) *gocom.CodedError {
	customLogger.InfoWithData("UpdateRecipient started", map[string]interface{}{
		"component":    "CampaignsService",
		"function":     "UpdateRecipient",
		"recipient_id": recipient.ID,
		"status":       recipient.Status,
	})

	// Acquire lock for this specific recipient to prevent race conditions
	mutexInterface, _ := recipientUpdateMutex.LoadOrStore(recipient.ID, &sync.Mutex{})
	mu := mutexInterface.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	// Get existing recipient
	mdl := campaignRecipient.GetRepo().GetById(recipient.ID)
	if mdl == nil {
		customLogger.WarnWithData("Recipient not found", map[string]interface{}{
			"component":    "CampaignsService",
			"function":     "UpdateRecipient",
			"recipient_id": recipient.ID,
		})
		return common.ERR_NOT_FOUND
	}

	// Update fields
	mdl.Status = recipient.Status

	// Parse and update time fields
	if recipient.DeliveredAt != "" {
		t, err := time.Parse(time.RFC3339, recipient.DeliveredAt)
		if err == nil {
			mdl.DeliveredAt = &t
		}
	}
	if recipient.ReadAt != "" {
		t, err := time.Parse(time.RFC3339, recipient.ReadAt)
		if err == nil {
			mdl.ReadAt = &t
		}
	}
	if recipient.FailedAt != "" {
		t, err := time.Parse(time.RFC3339, recipient.FailedAt)
		if err == nil {
			mdl.FailedAt = &t
		}
	}

	if recipient.ErrorMessage != "" {
		mdl.ErrorMessage = recipient.ErrorMessage
	}

	// Save to database
	err := campaignRecipient.GetRepo().Update(mdl)
	if err != nil {
		customLogger.ErrorWithData("Failed to update recipient", map[string]interface{}{
			"component":    "CampaignsService",
			"function":     "UpdateRecipient",
			"recipient_id": recipient.ID,
			"error":        err.Error(),
		})
		return common.ERR_UNABLE_TO_CREATE
	}

	customLogger.InfoWithData("Recipient updated successfully", map[string]interface{}{
		"component":    "CampaignsService",
		"function":     "UpdateRecipient",
		"recipient_id": recipient.ID,
		"status":       recipient.Status,
	})

	// Update campaign statistics after recipient status change
	if mdl.CampaignId != "" {
		go o.updateCampaignStatsById(mdl.CampaignId)
	}

	return nil
}

// updateCampaignStatsById updates campaign statistics by campaign ID
func (o *CampaignsSvcImpl) updateCampaignStatsById(campaignId string) {
	customLogger.DebugWithData("updateCampaignStatsById started", map[string]interface{}{
		"component":   "CampaignsService",
		"function":    "updateCampaignStatsById",
		"campaign_id": campaignId,
	})

	camp := campaign.GetRepo().GetById(campaignId)
	if camp == nil {
		customLogger.WarnWithData("Campaign not found", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "updateCampaignStatsById",
			"campaign_id": campaignId,
		})
		return
	}

	// Count recipients by status
	_, _, sentCount := campaignRecipient.GetRepo().SearchByCampaignId(campaignId, "sent", 1, 1)
	_, _, deliveredCount := campaignRecipient.GetRepo().SearchByCampaignId(campaignId, "delivered", 1, 1)
	_, _, failedCount := campaignRecipient.GetRepo().SearchByCampaignId(campaignId, "failed", 1, 1)
	_, _, readCount := campaignRecipient.GetRepo().SearchByCampaignId(campaignId, "read", 1, 1)

	camp.TotalSent = int(sentCount + deliveredCount + readCount) // sent includes delivered and read
	camp.TotalDelivered = int(deliveredCount + readCount)
	camp.TotalFailed = int(failedCount)
	camp.TotalRead = int(readCount)

	err := campaign.GetRepo().Update(camp)
	if err != nil {
		customLogger.ErrorWithData("Failed to update campaign stats", map[string]interface{}{
			"component":   "CampaignsService",
			"function":    "updateCampaignStatsById",
			"campaign_id": campaignId,
			"error":       err.Error(),
		})
	} else {
		customLogger.InfoWithData("Campaign stats updated", map[string]interface{}{
			"component":       "CampaignsService",
			"function":        "updateCampaignStatsById",
			"campaign_id":     campaignId,
			"total_sent":      camp.TotalSent,
			"total_delivered": camp.TotalDelivered,
			"total_failed":    camp.TotalFailed,
			"total_read":      camp.TotalRead,
		})
	}
}
