package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/config"
	"github.com/jinzhu/copier"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/client"
	"gitlab.com/anti_metter/switching_common/waTemplate"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
)

type WATemplateSvc interface {
	Create(req dtos.WATemplateReq, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError)
	Get(id string, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError)
	GetByWaTemplateId(waTemplateId string, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError)
	Update(id string, req dtos.WATemplateUpdateReq, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError)
	Delete(id string, authInfo auth.AuthInfo) *gocom.CodedError
	Search(filter, clientId, status, waStatus, category, sortBy, sortOrder string, pageNo, rowPerPage int) ([]dtos.WATemplate, bool, int64)
	Submit(id string, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError)
	Approve(id string, waTemplateId string, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError)
	Reject(id string, reason string, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError)
	SubmitToWhatsApp(id string, wabaId string, accessToken string, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError)
	UploadMediaToWhatsApp(imageData []byte, fileName string, authInfo auth.AuthInfo) (string, *gocom.CodedError)
}

type WATemplateSvcImpl struct {
}

func (o *WATemplateSvcImpl) Create(req dtos.WATemplateReq, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError) {
	customLogger.InfoWithData("Create template started", map[string]interface{}{
		"component":   "WATemplateService",
		"function":    "Create",
		"name":        req.Name,
		"client_id":   req.ClientId,
		"auth_client": authInfo.ClientId,
	})

	// Validasi input
	if req.Name == "" {
		customLogger.WarnWithData("Template name is required", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Create",
			"error":     "missing_name",
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Template name is required"}
	}

	if req.Category == "" {
		customLogger.WarnWithData("Category is required", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Create",
			"error":     "missing_category",
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Category is required"}
	}

	if req.Language == "" {
		customLogger.WarnWithData("Language is required", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Create",
			"error":     "missing_language",
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Language is required"}
	}

	if len(req.Components) == 0 {
		customLogger.WarnWithData("Components are required", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Create",
			"error":     "missing_components",
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Components are required"}
	}

	if common.PROVIDER_ID != authInfo.ClientId {
		req.ClientId = authInfo.ClientId
	}

	if req.ClientId == "" {
		req.ClientId = authInfo.ClientId
		customLogger.InfoWithData("Using auth clientId", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Create",
			"client_id": req.ClientId,
		})
	}

	clientModel := client.GetRepo().GetById(req.ClientId)
	if clientModel == nil {
		customLogger.WarnWithData("Client not found", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Create",
			"client_id": req.ClientId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if authInfo.ClientId != common.PROVIDER_ID && authInfo.ClientId != req.ClientId {
		customLogger.WarnWithData("Access denied", map[string]interface{}{
			"component":           "WATemplateService",
			"function":            "Create",
			"auth_client_id":      authInfo.ClientId,
			"requested_client_id": req.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Check duplicate name per client
	req.Name = strings.Trim(req.Name, " ")
	req.WATemplate = strings.Trim(req.WATemplate, " ")

	// Validate WATemplate field
	if req.WATemplate == "" {
		customLogger.WarnWithData("waTemplate is required", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Create",
			"error":     "missing_wa_template",
		})
		return nil, &gocom.CodedError{Code: 400, Message: "WhatsApp template name (waTemplate) is required"}
	}

	mdlReq := &waTemplate.WATemplate{
		Name:     req.Name,
		ClientId: req.ClientId,
		Media:    req.Media,
	}
	existing := waTemplate.GetRepo().GetByNameAndClient(mdlReq)
	if existing != nil {
		customLogger.WarnWithData("Template name already exists", map[string]interface{}{
			"component":     "WATemplateService",
			"function":      "Create",
			"template_name": req.Name,
			"client_id":     req.ClientId,
		})
		return nil, common.ERR_ALREADY_EXIST
	}

	// Process components - handle base64 image in HEADER and set default headerHandle
	for i := range req.Components {
		comp := &req.Components[i]
		compType := strings.ToUpper(comp.Type)
		compFormat := strings.ToUpper(comp.Format)
		isMediaHeader := compType == "HEADER" && (compFormat == "IMAGE" || compFormat == "VIDEO" || compFormat == "DOCUMENT")

		// Check if HEADER has base64 image in text field
		if isMediaHeader && strings.HasPrefix(comp.Text, "data:") {
			customLogger.InfoWithData("Detected base64 in HEADER text, clearing", map[string]interface{}{
				"component":   "WATemplateService",
				"function":    "Create",
				"text_length": len(comp.Text),
				"index":       i,
			})

			// Clear base64 data from text field - WhatsApp API doesn't accept text in media header
			comp.Text = ""
		}

		// Validate header_handle for media headers - REQUIRED
		if isMediaHeader {
			if comp.Example == nil || comp.Example.HeaderHandle == "" {
				customLogger.ErrorWithData("header_handle is required but not provided", map[string]interface{}{
					"component": "WATemplateService",
					"function":  "Create",
					"format":    compFormat,
					"index":     i,
				})
				return nil, &gocom.CodedError{
					Code:    400,
					Message: fmt.Sprintf("Header handle is required for %s header. Please upload media using /upload-img endpoint first and set the returned handle in example.header_handle field", compFormat),
				}
			}
			customLogger.InfoWithData("header_handle found", map[string]interface{}{
				"component":     "WATemplateService",
				"function":      "Create",
				"format":        compFormat,
				"header_handle": comp.Example.HeaderHandle,
			})
		}
	}

	// Convert components to JSON
	componentsJSON, err := json.Marshal(req.Components)
	if err != nil {
		customLogger.ErrorWithData("Unable to marshal components", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Create",
			"error":     err.Error(),
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Invalid components format"}
	}

	// Log component details
	customLogger.DebugWithData("Components after processing", map[string]interface{}{
		"component":       "WATemplateService",
		"function":        "Create",
		"components_json": string(componentsJSON),
		"count":           len(req.Components),
	})

	mdl := &waTemplate.WATemplate{
		Name:       req.Name,
		WATemplate: req.WATemplate,
		Category:   req.Category,
		Language:   req.Language,
		ClientId:   req.ClientId,
		Components: string(componentsJSON),
		Status:     "DRAFT",
		WAStatus:   "",
		Media:      req.Media,
	}

	if req.Status != "" {
		mdl.Status = req.Status
	}

	errCreate := waTemplate.GetRepo().Create(mdl).Error
	if errCreate != nil {
		customLogger.ErrorWithData("Unable to create template", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Create",
			"error":     errCreate.Error(),
			"name":      req.Name,
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	customLogger.InfoWithData("Template created successfully", map[string]interface{}{
		"component":   "WATemplateService",
		"function":    "Create",
		"template_id": mdl.ID,
		"name":        mdl.Name,
	})
	return o.toDTO(mdl), nil
}

func (o *WATemplateSvcImpl) Get(id string, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError) {
	customLogger.InfoWithData("Get template started", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "Get",
		"id":        id,
		"client_id": authInfo.ClientId,
	})

	mdl := waTemplate.GetRepo().GetById(id)
	if mdl == nil {
		customLogger.WarnWithData("Template not found", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Get",
			"id":        id,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if authInfo.ClientId != common.PROVIDER_ID && authInfo.ClientId != mdl.ClientId {
		customLogger.WarnWithData("Access denied", map[string]interface{}{
			"component":          "WATemplateService",
			"function":           "Get",
			"auth_client_id":     authInfo.ClientId,
			"template_client_id": mdl.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	customLogger.InfoWithData("Template found", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "Get",
		"id":        mdl.ID,
		"name":      mdl.Name,
	})
	return o.toDTO(mdl), nil
}

func (o *WATemplateSvcImpl) GetByWaTemplateId(waTemplateId string, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError) {
	customLogger.InfoWithData("GetByWaTemplateId started", map[string]interface{}{
		"component":      "WATemplateService",
		"function":       "GetByWaTemplateId",
		"wa_template_id": waTemplateId,
		"client_id":      authInfo.ClientId,
	})

	mdl := waTemplate.GetRepo().GetByWATemplateId(waTemplateId)
	if mdl == nil {
		customLogger.WarnWithData("Template not found", map[string]interface{}{
			"component":      "WATemplateService",
			"function":       "GetByWaTemplateId",
			"wa_template_id": waTemplateId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if authInfo.ClientId != common.PROVIDER_ID && authInfo.ClientId != mdl.ClientId {
		customLogger.WarnWithData("Access denied", map[string]interface{}{
			"component":          "WATemplateService",
			"function":           "GetByWaTemplateId",
			"auth_client_id":     authInfo.ClientId,
			"template_client_id": mdl.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	customLogger.InfoWithData("Template found", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "GetByWaTemplateId",
		"id":        mdl.ID,
		"name":      mdl.Name,
	})
	return o.toDTO(mdl), nil
}

func (o *WATemplateSvcImpl) Update(id string, req dtos.WATemplateUpdateReq, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError) {
	customLogger.InfoWithData("Update template started", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "Update",
		"id":        id,
		"client_id": authInfo.ClientId,
	})

	mdl := waTemplate.GetRepo().GetById(id)
	if mdl == nil {
		customLogger.WarnWithData("Template not found", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Update",
			"id":        id,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if authInfo.ClientId != common.PROVIDER_ID && authInfo.ClientId != mdl.ClientId {
		customLogger.WarnWithData("Access denied", map[string]interface{}{
			"component":          "WATemplateService",
			"function":           "Update",
			"auth_client_id":     authInfo.ClientId,
			"template_client_id": mdl.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Check if status allows update
	if mdl.Status == "SUBMITTED" || mdl.Status == "APPROVED" {
		customLogger.WarnWithData("Cannot update template", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Update",
			"id":        id,
			"status":    mdl.Status,
			"reason":    "status_locked",
		})
		return nil, &gocom.CodedError{
			Code:    400,
			Message: "Cannot update template in " + mdl.Status + " status",
		}
	}

	// Update fields
	if req.Name != "" {
		req.Name = strings.Trim(req.Name, " ")
		// Check duplicate name
		checkMdl := &waTemplate.WATemplate{
			Name: req.Name,
			//ClientId: mdl.ClientId,
			Media: req.Media,
		}
		existing := waTemplate.GetRepo().GetByNameAndClient(checkMdl)
		if existing != nil && existing.ID != id {
			customLogger.WarnWithData("Template name already exists", map[string]interface{}{
				"component": "WATemplateService",
				"function":  "Update",
				"name":      req.Name,
			})
			return nil, common.ERR_ALREADY_EXIST
		}
		mdl.Name = req.Name
	}

	if req.WATemplate != "" {
		req.WATemplate = strings.Trim(req.WATemplate, " ")
		mdl.WATemplate = req.WATemplate
	}

	if req.Category != "" {
		mdl.Category = req.Category
	}

	if req.Language != "" {
		mdl.Language = req.Language
	}

	if len(req.Components) > 0 {
		componentsJSON, err := json.Marshal(req.Components)
		if err != nil {
			customLogger.ErrorWithData("Unable to marshal components", map[string]interface{}{
				"component": "WATemplateService",
				"function":  "Update",
				"error":     err.Error(),
			})
			return nil, &gocom.CodedError{Code: 400, Message: "Invalid components format"}
		}
		mdl.Components = string(componentsJSON)
	}

	if req.Status != "" {
		mdl.Status = req.Status
	}

	errUpdate := waTemplate.GetRepo().Update(mdl)
	if errUpdate != nil {
		customLogger.ErrorWithData("Unable to update template", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Update",
			"error":     errUpdate.Error(),
			"id":        id,
		})
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	customLogger.InfoWithData("Template updated successfully", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "Update",
		"id":        mdl.ID,
		"name":      mdl.Name,
	})
	return o.toDTO(mdl), nil
}

func (o *WATemplateSvcImpl) Delete(id string, authInfo auth.AuthInfo) *gocom.CodedError {
	customLogger.InfoWithData("Delete template started", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "Delete",
		"id":        id,
		"client_id": authInfo.ClientId,
	})

	mdl := waTemplate.GetRepo().GetById(id)
	if mdl == nil {
		customLogger.WarnWithData("Template not found", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Delete",
			"id":        id,
		})
		return common.ERR_NOT_FOUND
	}

	// Check permission
	if authInfo.ClientId != common.PROVIDER_ID && authInfo.ClientId != mdl.ClientId {
		customLogger.WarnWithData("Access denied", map[string]interface{}{
			"component":          "WATemplateService",
			"function":           "Delete",
			"auth_client_id":     authInfo.ClientId,
			"template_client_id": mdl.ClientId,
		})
		return common.ERR_NOT_ALLOWED
	}

	// Check if status allows delete
	if mdl.Status == "APPROVED" {
		customLogger.WarnWithData("Cannot delete approved template", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Delete",
			"id":        id,
			"status":    mdl.Status,
		})
		return &gocom.CodedError{
			Code:    400,
			Message: "Cannot delete approved template",
		}
	}

	err := waTemplate.GetRepo().Delete(id)
	if err != nil {
		customLogger.ErrorWithData("Unable to delete template", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Delete",
			"error":     err.Error(),
			"id":        id,
		})
		return common.ERR_UNABLE_TO_DELETE
	}

	customLogger.InfoWithData("Template deleted successfully", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "Delete",
		"id":        id,
	})
	return nil
}

func (o *WATemplateSvcImpl) Search(filter, clientId, status, waStatus, category, sortBy, sortOrder string, pageNo, rowPerPage int) ([]dtos.WATemplate, bool, int64) {
	customLogger.DebugWithData("Search templates started", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "Search",
		"filter":    filter,
		"client_id": clientId,
		"status":    status,
		"wa_status": waStatus,
		"category":  category,
		"page":      pageNo,
		"rows":      rowPerPage,
	})

	ret := make([]dtos.WATemplate, 0)

	list, haveNext, count := waTemplate.GetRepo().Search(filter, clientId, status, waStatus, category, sortBy, sortOrder, pageNo, rowPerPage)

	customLogger.InfoWithData("Search completed", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "Search",
		"found":     len(list),
		"total":     count,
	})

	for _, item := range list {
		ret = append(ret, *o.toDTO(&item))
	}

	return ret, haveNext, count
}

func (o *WATemplateSvcImpl) Submit(id string, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError) {
	customLogger.InfoWithData("Submit template started", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "Submit",
		"id":        id,
		"client_id": authInfo.ClientId,
	})

	mdl := waTemplate.GetRepo().GetById(id)
	if mdl == nil {
		customLogger.WarnWithData("Template not found", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Submit",
			"id":        id,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if authInfo.ClientId != common.PROVIDER_ID && authInfo.ClientId != mdl.ClientId {
		customLogger.WarnWithData("Access denied", map[string]interface{}{
			"component":          "WATemplateService",
			"function":           "Submit",
			"auth_client_id":     authInfo.ClientId,
			"template_client_id": mdl.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Check status
	if mdl.Status != "DRAFT" && mdl.Status != "REJECTED" {
		customLogger.WarnWithData("Cannot submit template", map[string]interface{}{
			"component":      "WATemplateService",
			"function":       "Submit",
			"id":             id,
			"current_status": mdl.Status,
		})
		return nil, &gocom.CodedError{
			Code:    400,
			Message: "Can only submit template in DRAFT or REJECTED status",
		}
	}

	now := time.Now()
	mdl.Status = "SUBMITTED"
	mdl.WAStatus = "PENDING"
	mdl.SubmittedAt = &now

	err := waTemplate.GetRepo().Update(mdl)
	if err != nil {
		customLogger.ErrorWithData("Unable to update template", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Submit",
			"error":     err.Error(),
			"id":        id,
		})
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	customLogger.InfoWithData("Template submitted successfully", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "Submit",
		"id":        mdl.ID,
		"name":      mdl.Name,
	})
	return o.toDTO(mdl), nil
}

func (o *WATemplateSvcImpl) Approve(id string, waTemplateId string, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError) {
	customLogger.InfoWithData("Approve template started", map[string]interface{}{
		"component":      "WATemplateService",
		"function":       "Approve",
		"id":             id,
		"wa_template_id": waTemplateId,
		"client_id":      authInfo.ClientId,
	})

	mdl := waTemplate.GetRepo().GetById(id)
	if mdl == nil {
		customLogger.WarnWithData("Template not found", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Approve",
			"id":        id,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Only PROVIDER can approve
	if authInfo.ClientId != common.PROVIDER_ID {
		customLogger.WarnWithData("Only provider can approve", map[string]interface{}{
			"component":      "WATemplateService",
			"function":       "Approve",
			"auth_client_id": authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Check status
	if mdl.Status != "SUBMITTED" {
		customLogger.WarnWithData("Cannot approve template", map[string]interface{}{
			"component":      "WATemplateService",
			"function":       "Approve",
			"id":             id,
			"current_status": mdl.Status,
		})
		return nil, &gocom.CodedError{
			Code:    400,
			Message: "Can only approve submitted template",
		}
	}

	if waTemplateId == "" {
		customLogger.WarnWithData("waTemplateId is required", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Approve",
			"id":        id,
		})
		return nil, &gocom.CodedError{Code: 400, Message: "WhatsApp Template ID is required"}
	}

	now := time.Now()
	mdl.Status = "APPROVED"
	mdl.WAStatus = "APPROVED"
	mdl.WATemplateId = waTemplateId
	mdl.ApprovedAt = &now

	err := waTemplate.GetRepo().Update(mdl)
	if err != nil {
		customLogger.ErrorWithData("Unable to update template", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Approve",
			"error":     err.Error(),
			"id":        id,
		})
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	customLogger.InfoWithData("Template approved successfully", map[string]interface{}{
		"component":      "WATemplateService",
		"function":       "Approve",
		"id":             mdl.ID,
		"name":           mdl.Name,
		"wa_template_id": waTemplateId,
	})
	return o.toDTO(mdl), nil
}

func (o *WATemplateSvcImpl) Reject(id string, reason string, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError) {
	customLogger.InfoWithData("Reject template started", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "Reject",
		"id":        id,
		"client_id": authInfo.ClientId,
	})

	mdl := waTemplate.GetRepo().GetById(id)
	if mdl == nil {
		customLogger.WarnWithData("Template not found", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Reject",
			"id":        id,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Only PROVIDER can reject
	if authInfo.ClientId != common.PROVIDER_ID {
		customLogger.WarnWithData("Only provider can reject", map[string]interface{}{
			"component":      "WATemplateService",
			"function":       "Reject",
			"auth_client_id": authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Check status
	if mdl.Status != "SUBMITTED" {
		customLogger.WarnWithData("Cannot reject template", map[string]interface{}{
			"component":      "WATemplateService",
			"function":       "Reject",
			"id":             id,
			"current_status": mdl.Status,
		})
		return nil, &gocom.CodedError{
			Code:    400,
			Message: "Can only reject submitted template",
		}
	}

	if reason == "" {
		customLogger.WarnWithData("Rejection reason is required", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Reject",
			"id":        id,
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Rejection reason is required"}
	}

	mdl.Status = "REJECTED"
	mdl.WAStatus = "REJECTED"
	mdl.RejectedReason = reason

	err := waTemplate.GetRepo().Update(mdl)
	if err != nil {
		customLogger.ErrorWithData("Unable to update template", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "Reject",
			"error":     err.Error(),
			"id":        id,
		})
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	customLogger.InfoWithData("Template rejected successfully", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "Reject",
		"id":        mdl.ID,
		"name":      mdl.Name,
		"reason":    reason,
	})
	return o.toDTO(mdl), nil
}

func (o *WATemplateSvcImpl) SubmitToWhatsApp(id string, wabaId string, accessToken string, authInfo auth.AuthInfo) (*dtos.WATemplate, *gocom.CodedError) {
	customLogger.InfoWithData("Start", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "SubmitToWhatsApp",
		"id":        id,
		"client_id": authInfo.ClientId,
	})

	clientMdl := client.GetRepo().GetById(authInfo.ClientId)
	if clientMdl == nil {
		customLogger.WarnWithData("Client not found", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "SubmitToWhatsApp",
			"client_id": authInfo.ClientId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	wabaId = clientMdl.WabaId
	if clientMdl.WabaId == "" {
		wabaId = config.Get(constans.WaBaId, "")
	}
	accessToken = config.Get(constans.WaAuthToken, "")
	customLogger.InfoWithData("SubmitToWhatsApp started", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "SubmitToWhatsApp",
		"id":        id,
		"waba_id":   wabaId,
		"client_id": authInfo.ClientId,
	})

	mdl := waTemplate.GetRepo().GetById(id)
	if mdl == nil {
		customLogger.WarnWithData("Template not found", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "SubmitToWhatsApp",
			"id":        id,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if authInfo.ClientId != common.PROVIDER_ID && authInfo.ClientId != mdl.ClientId {
		customLogger.WarnWithData("Access denied", map[string]interface{}{
			"component":          "WATemplateService",
			"function":           "SubmitToWhatsApp",
			"auth_client_id":     authInfo.ClientId,
			"template_client_id": mdl.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Validation
	if wabaId == "" {
		return nil, &gocom.CodedError{Code: 400, Message: "WhatsApp Business Account ID is required"}
	}

	if accessToken == "" {
		return nil, &gocom.CodedError{Code: 400, Message: "Access Token is required"}
	}

	if mdl.Status != "SUBMITTED" {
		customLogger.WarnWithData("Invalid status for WhatsApp submission", map[string]interface{}{
			"component":      "WATemplateService",
			"function":       "SubmitToWhatsApp",
			"id":             id,
			"current_status": mdl.Status,
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Can only submit to WhatsApp after internal submission (SUBMITTED status)"}
	}

	// Parse components
	var components []dtos.WATemplateComponent
	if mdl.Components != "" {
		err := json.Unmarshal([]byte(mdl.Components), &components)
		if err != nil {
			customLogger.ErrorWithData("Unable to parse components", map[string]interface{}{
				"component": "WATemplateService",
				"function":  "SubmitToWhatsApp",
				"error":     err.Error(),
				"id":        id,
			})
			return nil, &gocom.CodedError{Code: 400, Message: "Invalid components format"}
		}
	}

	customLogger.DebugWithData("Components parsed from DB", map[string]interface{}{
		"component":       "WATemplateService",
		"function":        "SubmitToWhatsApp",
		"components_json": mdl.Components,
		"count":           len(components),
	})

	// Build WhatsApp API payload
	waComponents := make([]map[string]interface{}, 0)
	for _, comp := range components {
		waComp := map[string]interface{}{
			"type": strings.ToUpper(comp.Type),
		}

		if strings.ToUpper(mdl.Category) == "AUTHENTICATION" && strings.ToUpper(comp.Type) == "BODY" {
			waComp["add_security_recommendation"] = true
		}

		if comp.Format != "" {
			waComp["format"] = strings.ToUpper(comp.Format)
		}

		// Only add text field for non-media headers and other components
		// For HEADER with media format (IMAGE, VIDEO, DOCUMENT), do not include text field
		compType := strings.ToUpper(comp.Type)
		compFormat := strings.ToUpper(comp.Format)
		isMediaHeader := compType == "HEADER" && (compFormat == "IMAGE" || compFormat == "VIDEO" || compFormat == "DOCUMENT")

		if comp.Text != "" && !isMediaHeader {
			waComp["text"] = comp.Text
		}

		if len(comp.Buttons) > 0 {
			buttons := make([]map[string]interface{}, 0)
			for _, btn := range comp.Buttons {
				button := map[string]interface{}{
					"type": strings.ToUpper(btn.Type),
					"text": btn.Text,
				}
				if strings.ToUpper(mdl.Category) == "AUTHENTICATION" {
					button = map[string]interface{}{
						"type":     strings.ToUpper(btn.Type),
						"otp_type": "COPY_CODE",
						"text":     btn.Text,
					}
				}

				if btn.URL != "" {
					button["url"] = btn.URL
				}
				if btn.PhoneNumber != "" {
					button["phone_number"] = btn.PhoneNumber
				}
				buttons = append(buttons, button)
			}
			waComp["buttons"] = buttons
		}

		// Add examples for BODY (text parameters)
		if len(comp.Examples) > 0 && strings.ToUpper(comp.Type) == "BODY" {
			waComp["example"] = map[string]interface{}{
				"body_text": [][]string{comp.Examples},
			}
		}

		// Add header media example (image/video/document)
		if compType == "HEADER" && isMediaHeader {
			var headerHandle string
			if comp.Example != nil {
				headerHandle = comp.Example.HeaderHandle
			}

			// Validate that headerHandle is provided for media headers
			if headerHandle == "" {
				customLogger.ErrorWithData("headerHandle is required but not provided", map[string]interface{}{
					"component": "WATemplateService",
					"function":  "SubmitToWhatsApp",
					"format":    compFormat,
					"id":        id,
				})
				return nil, &gocom.CodedError{
					Code:    400,
					Message: fmt.Sprintf("Header handle is required for %s header. Please upload media using /upload-img endpoint first and set the returned handle in example.header_handle field", compFormat),
				}
			}

			// WhatsApp API expects header_handle in example object as array
			waComp["example"] = map[string]interface{}{
				"header_handle": []string{headerHandle},
			}

			customLogger.DebugWithData("Added example for header", map[string]interface{}{
				"component":     "WATemplateService",
				"function":      "SubmitToWhatsApp",
				"format":        compFormat,
				"header_handle": headerHandle,
			})
		}

		waComponents = append(waComponents, waComp)
	}

	waTemplateName := fmt.Sprintf("%s", strings.ReplaceAll(strings.ToLower(mdl.Name), " ", "_"))
	if mdl.WATemplate != "" {
		waTemplateName = mdl.WATemplate
	}

	payload := map[string]interface{}{
		"name":       waTemplateName,
		"category":   strings.ToUpper(mdl.Category),
		"language":   mdl.Language,
		"components": waComponents,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		customLogger.ErrorWithData("Unable to marshal payload", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "SubmitToWhatsApp",
			"error":     err.Error(),
			"id":        id,
		})
		return nil, &gocom.CodedError{Code: 500, Message: "Failed to create payload"}
	}

	baseURL := fmt.Sprintf("%s/%s/message_templates", config.Get(constans.BaseURLMeta), wabaId)
	customLogger.InfoWithData("Submitting to WhatsApp API", map[string]interface{}{
		"component":    "WATemplateService",
		"function":     "SubmitToWhatsApp",
		"url":          baseURL,
		"payload_size": len(payloadBytes),
	})

	// Send to WhatsApp API
	httpReq, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		customLogger.ErrorWithData("Unable to create request", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "SubmitToWhatsApp",
			"error":     err.Error(),
			"url":       baseURL,
		})
		return nil, &gocom.CodedError{Code: 500, Message: "Failed to create request"}
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		customLogger.ErrorWithData("Request failed", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "SubmitToWhatsApp",
			"error":     err.Error(),
			"url":       baseURL,
		})
		return nil, &gocom.CodedError{Code: 500, Message: "Failed to submit to WhatsApp"}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		customLogger.ErrorWithData("Unable to read response", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "SubmitToWhatsApp",
			"error":     err.Error(),
			"id":        id,
		})
		return nil, &gocom.CodedError{Code: 500, Message: "Failed to read response"}
	}

	customLogger.InfoWithData("WhatsApp API response received", map[string]interface{}{
		"component":   "WATemplateService",
		"function":    "SubmitToWhatsApp",
		"status_code": resp.StatusCode,
		"body_length": len(body),
	})

	if resp.StatusCode != 200 {
		customLogger.ErrorWithData("WhatsApp API error", map[string]interface{}{
			"component":   "WATemplateService",
			"function":    "SubmitToWhatsApp",
			"status_code": resp.StatusCode,
			"body":        string(body),
		})
		return nil, &gocom.CodedError{Code: resp.StatusCode, Message: "WhatsApp API error: " + string(body)}
	}

	// Parse response to get template ID
	var waResp map[string]interface{}
	err = json.Unmarshal(body, &waResp)
	if err != nil {
		customLogger.ErrorWithData("Unable to parse response", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "SubmitToWhatsApp",
			"error":     err.Error(),
			"body":      string(body),
		})
		return nil, &gocom.CodedError{Code: 500, Message: "Failed to parse response"}
	}

	// Update template status
	now := time.Now()
	mdl.Status = "SUBMITTED"
	mdl.WAStatus = "PENDING"
	mdl.SubmittedAt = &now

	if waTemplateIdVal, ok := waResp["id"].(string); ok {
		mdl.WATemplateId = waTemplateIdVal
	}

	errUpdate := waTemplate.GetRepo().Update(mdl)
	if errUpdate != nil {
		customLogger.ErrorWithData("Unable to update template", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "SubmitToWhatsApp",
			"error":     errUpdate.Error(),
			"id":        id,
		})
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	customLogger.InfoWithData("Template submitted to WhatsApp successfully", map[string]interface{}{
		"component":      "WATemplateService",
		"function":       "SubmitToWhatsApp",
		"id":             mdl.ID,
		"wa_template_id": mdl.WATemplateId,
	})
	return o.toDTO(mdl), nil
}

func (o *WATemplateSvcImpl) toDTO(mdl *waTemplate.WATemplate) *dtos.WATemplate {
	ret := &dtos.WATemplate{}
	_ = copier.Copy(ret, mdl)

	// Parse components JSON
	var components []dtos.WATemplateComponent
	if mdl.Components != "" {
		err := json.Unmarshal([]byte(mdl.Components), &components)
		if err != nil {
			customLogger.WarnWithData("Unable to unmarshal components", map[string]interface{}{
				"component": "WATemplateService",
				"function":  "toDTO",
				"error":     err.Error(),
				"id":        mdl.ID,
			})
		} else {
			ret.Components = components
		}
	}

	// Get client name
	clientModel := client.GetRepo().GetById(mdl.ClientId)
	if clientModel != nil {
		ret.ClientName = clientModel.Name
	}

	// Format timestamps
	if !mdl.CreatedAt.IsZero() {
		ret.CreatedAt = mdl.CreatedAt.Format(time.RFC3339)
	}
	if !mdl.UpdatedAt.IsZero() {
		ret.UpdatedAt = mdl.UpdatedAt.Format(time.RFC3339)
	}
	if mdl.SubmittedAt != nil && !mdl.SubmittedAt.IsZero() {
		ret.SubmittedAt = mdl.SubmittedAt.Format(time.RFC3339)
	}
	if mdl.ApprovedAt != nil && !mdl.ApprovedAt.IsZero() {
		ret.ApprovedAt = mdl.ApprovedAt.Format(time.RFC3339)
	}

	return ret
}

// UploadImageToWhatsApp uploads image to WhatsApp Cloud API using Resumable Upload
// Returns the file handle that can be used in template header_handle
func (o *WATemplateSvcImpl) UploadMediaToWhatsApp(data []byte, fileName string, authInfo auth.AuthInfo) (string, *gocom.CodedError) {
	customLogger.InfoWithData("UploadMediaToWhatsApp started", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "UploadMediaToWhatsApp",
		"file_name": fileName,
		"file_size": len(data),
		"client_id": authInfo.ClientId,
	})

	appId := config.Get(constans.AppId, "")
	accessToken := config.Get(constans.WaAuthToken, "")

	if appId == "" {
		customLogger.ErrorWithData("WABA ID not configured", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "UploadMediaToWhatsApp",
		})
		return "", &gocom.CodedError{Code: 500, Message: "WhatsApp Business Account ID not configured"}
	}

	if accessToken == "" {
		customLogger.ErrorWithData("Access Token not configured", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "UploadMediaToWhatsApp",
		})
		return "", &gocom.CodedError{Code: 500, Message: "WhatsApp Access Token not configured"}
	}

	// Determine content type from file name
	contentType := o.getContentType(fileName)
	if contentType == "" {
		customLogger.ErrorWithData("Invalid file type", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "UploadMediaToWhatsApp",
			"file_name": fileName,
		})
		return "", &gocom.CodedError{Code: 400, Message: "Unsupported file type"}
	}

	fileLength := len(data)

	// Step 1: Create Upload Session
	customLogger.InfoWithData("Creating upload session", map[string]interface{}{
		"component":    "WATemplateService",
		"function":     "UploadMediaToWhatsApp",
		"file_length":  fileLength,
		"content_type": contentType,
		"file_name":    fileName,
	})

	sessionURL := fmt.Sprintf("%s/%s/uploads?file_length=%d&file_type=%s&file_name=%s", config.Get(constans.BaseURLMeta), appId, fileLength, url.QueryEscape(contentType), url.QueryEscape(fileName))

	sessionReq, err := http.NewRequest("POST", sessionURL, nil)
	if err != nil {
		customLogger.ErrorWithData("Failed to create session request", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "UploadMediaToWhatsApp",
			"error":     err.Error(),
		})
		return "", common.ERR_INTERNAL_SERVER_ERROR
	}

	sessionReq.Header.Set("Authorization", "Bearer "+accessToken)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	sessionResp, err := httpClient.Do(sessionReq)
	if err != nil {
		customLogger.ErrorWithData("Failed to create upload session", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "UploadMediaToWhatsApp",
			"error":     err.Error(),
		})
		return "", common.ERR_INTERNAL_SERVER_ERROR
	}
	defer sessionResp.Body.Close()

	sessionBody, _ := io.ReadAll(sessionResp.Body)
	customLogger.DebugWithData("Upload session response", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "UploadMediaToWhatsApp",
		"response":  string(sessionBody),
	})

	if sessionResp.StatusCode != 200 {
		customLogger.ErrorWithData("Failed to create session", map[string]interface{}{
			"component":   "WATemplateService",
			"function":    "UploadMediaToWhatsApp",
			"status_code": sessionResp.StatusCode,
			"body":        string(sessionBody),
		})
		return "", &gocom.CodedError{Code: 400, Message: fmt.Sprintf("Failed to create upload session: %s", string(sessionBody))}
	}

	var sessionResult map[string]interface{}
	if err := json.Unmarshal(sessionBody, &sessionResult); err != nil {
		customLogger.ErrorWithData("Failed to parse session response", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "UploadMediaToWhatsApp",
			"error":     err.Error(),
		})
		return "", common.ERR_INTERNAL_SERVER_ERROR
	}

	uploadSessionId, ok := sessionResult["id"].(string)
	if !ok || uploadSessionId == "" {
		customLogger.ErrorWithData("No upload session ID in response", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "UploadMediaToWhatsApp",
		})
		return "", common.ERR_INTERNAL_SERVER_ERROR
	}

	customLogger.InfoWithData("Upload session created", map[string]interface{}{
		"component":         "WATemplateService",
		"function":          "UploadMediaToWhatsApp",
		"upload_session_id": uploadSessionId,
	})

	// Step 2: Upload Binary Data
	customLogger.InfoWithData("Uploading binary data", map[string]interface{}{
		"component":         "WATemplateService",
		"function":          "UploadMediaToWhatsApp",
		"upload_session_id": uploadSessionId,
		"file_size":         len(data),
	})

	uploadURL := fmt.Sprintf("https://graph.facebook.com/v24.0/%s", uploadSessionId)

	uploadReq, err := http.NewRequest("POST", uploadURL, bytes.NewReader(data))
	if err != nil {
		customLogger.ErrorWithData("Failed to create upload request", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "UploadMediaToWhatsApp",
			"error":     err.Error(),
		})
		return "", common.ERR_INTERNAL_SERVER_ERROR
	}

	uploadReq.Header.Set("Authorization", "Bearer "+accessToken)
	uploadReq.Header.Set("file_offset", "0")
	uploadReq.Header.Set("Content-Type", contentType)

	uploadResp, err := httpClient.Do(uploadReq)
	if err != nil {
		customLogger.ErrorWithData("Failed to upload binary", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "UploadMediaToWhatsApp",
			"error":     err.Error(),
		})
		return "", common.ERR_INTERNAL_SERVER_ERROR
	}
	defer uploadResp.Body.Close()

	uploadBody, _ := io.ReadAll(uploadResp.Body)
	customLogger.DebugWithData("Upload response received", map[string]interface{}{
		"component": "WATemplateService",
		"function":  "UploadMediaToWhatsApp",
		"response":  string(uploadBody),
	})

	if uploadResp.StatusCode != 200 {
		customLogger.ErrorWithData("Failed to upload binary", map[string]interface{}{
			"component":   "WATemplateService",
			"function":    "UploadMediaToWhatsApp",
			"status_code": uploadResp.StatusCode,
			"body":        string(uploadBody),
		})
		return "", &gocom.CodedError{Code: 400, Message: fmt.Sprintf("Failed to upload media: %s", string(uploadBody))}
	}

	var uploadResult map[string]interface{}
	if err := json.Unmarshal(uploadBody, &uploadResult); err != nil {
		customLogger.ErrorWithData("Failed to parse upload response", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "UploadMediaToWhatsApp",
			"error":     err.Error(),
		})
		return "", common.ERR_INTERNAL_SERVER_ERROR
	}

	fileHandle, ok := uploadResult["h"].(string)
	if !ok || fileHandle == "" {
		customLogger.ErrorWithData("No file handle in response", map[string]interface{}{
			"component": "WATemplateService",
			"function":  "UploadMediaToWhatsApp",
		})
		return "", common.ERR_INTERNAL_SERVER_ERROR
	}

	customLogger.InfoWithData("Image uploaded to WhatsApp successfully", map[string]interface{}{
		"component":   "WATemplateService",
		"function":    "UploadMediaToWhatsApp",
		"file_handle": fileHandle,
		"file_name":   fileName,
	})
	return fileHandle, nil
}

func (o *WATemplateSvcImpl) getContentType(fileName string) string {
	extension := filepath.Ext(fileName)
	switch extension {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".csv":
		return "text/csv"
	}
	return ""
}

//----------------------------------------

var waTemplateSvc WATemplateSvc
var waTemplateSvcOnce sync.Once

func SetWATemplateSvc(svc WATemplateSvc) {
	waTemplateSvc = svc
}

func GetWATemplateSvc() WATemplateSvc {
	if waTemplateSvc == nil {
		waTemplateSvcOnce.Do(func() {
			tmp := &WATemplateSvcImpl{}
			waTemplateSvc = tmp
		})
	}
	return waTemplateSvc
}
