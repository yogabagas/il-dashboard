package services

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"path/filepath"
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
	"gitlab.com/anti_metter/switching_common/contact"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/utils"
)

type ContactsService interface {
	Create(req dtos.ContactReq, authInfo auth.AuthInfo) (*dtos.Contact, *gocom.CodedError)
	Update(contactId string, req dtos.ContactUpdateReq, authInfo auth.AuthInfo) (*dtos.Contact, *gocom.CodedError)
	GetById(contactId string, authInfo auth.AuthInfo) (*dtos.Contact, *gocom.CodedError)
	Search(filter, phone, email, status, clientId string, tags []string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.Contact, bool, int)
	Delete(contactId string, authInfo auth.AuthInfo) *gocom.CodedError
	ImportFromCSV(req dtos.ContactImportReq, authInfo auth.AuthInfo) (*dtos.ContactImportResp, *gocom.CodedError)
	UploadContactsFile(file *multipart.FileHeader, campaignId string, authInfo auth.AuthInfo) (*dtos.ContactUploadFileResp, *gocom.CodedError)
	BulkDelete(contactIds []string, authInfo auth.AuthInfo) *gocom.CodedError
	BulkUpdateStatus(contactIds []string, status string, authInfo auth.AuthInfo) *gocom.CodedError
}

type ContactsSvcImpl struct{}

var contactsService ContactsService
var onceContactsService sync.Once

func GetContactsService() ContactsService {
	onceContactsService.Do(func() {
		contactsService = &ContactsSvcImpl{}
	})
	return contactsService
}

func marshalCustomFields(m map[string]interface{}) string {
	if m == nil {
		return ""
	}
	b, err := json.Marshal(m)
	if err != nil {
		customLogger.DebugWithData("Unable to marshal custom fields", map[string]interface{}{
			"component": "ContactsService",
			"function":  "marshalCustomFields",
			"error":     err.Error(),
		})
		return ""
	}
	return string(b)
}

func unmarshalCustomFields(s string) map[string]interface{} {
	if s == "" {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		customLogger.DebugWithData("Unable to unmarshal custom fields", map[string]interface{}{
			"component": "ContactsService",
			"function":  "unmarshalCustomFields",
			"error":     err.Error(),
		})
		return nil
	}
	return m
}

func (o *ContactsSvcImpl) Create(req dtos.ContactReq, authInfo auth.AuthInfo) (*dtos.Contact, *gocom.CodedError) {
	customLogger.InfoWithData("Create contact started", map[string]interface{}{
		"component": "ContactsService",
		"function":  "Create",
		"name":      req.Name,
		"phone":     req.Phone,
		"client_id": authInfo.ClientId,
	})

	clientId := req.ClientId
	if common.PROVIDER_ID != authInfo.ClientId {
		clientId = authInfo.ClientId
	}

	if req.ClientId == "" {
		customLogger.WarnWithData("Empty clientId", map[string]interface{}{
			"component": "ContactsService",
			"function":  "Create",
		})
		return nil, common.ERR_INVALID_REQUEST
	}

	// Validation
	if req.Name == "" {
		customLogger.WarnWithData("Name is required", map[string]interface{}{
			"component": "ContactsService",
			"function":  "Create",
		})
		return nil, common.ERR_INVALID_REQUEST
	}

	if req.Phone == "" && req.Email == "" {
		customLogger.WarnWithData("At least phone or email is required", map[string]interface{}{
			"component": "ContactsService",
			"function":  "Create",
		})
		return nil, gocom.NewError(400, "At least phone or email is required")
	}

	// Check duplicate phone or email
	if req.Phone != "" {
		existing := contact.GetRepo().GetByClientIdAndPhone(clientId, req.Phone)
		if existing != nil {
			customLogger.WarnWithData("Phone already exists", map[string]interface{}{
				"component": "ContactsService",
				"function":  "Create",
				"phone":     req.Phone,
			})
			return nil, gocom.NewError(409, "Phone already exists")
		}
	}

	if req.Email != "" {
		existing := contact.GetRepo().GetByClientIdAndEmail(clientId, req.Email)
		if existing != nil {
			customLogger.WarnWithData("Email already exists", map[string]interface{}{
				"component": "ContactsService",
				"function":  "Create",
				"email":     req.Email,
			})
			return nil, gocom.NewError(409, "Email already exists")
		}
	}

	// Set default tags if empty
	if len(req.Tags) == 0 {
		req.Tags = []string{"Instant Link"}
	}

	// Set default customFields if empty
	if req.CustomFields == nil || len(req.CustomFields) == 0 {
		req.CustomFields = map[string]interface{}{
			"city": "Tangerang Selatan",
		}
	}

	// Create model: store tags as JSON array
	tagsBytes := []byte{}
	if len(req.Tags) > 0 {
		tagsBytes, _ = json.Marshal(req.Tags)
	}

	mdl := &contact.Contact{
		ID:           ulid.Make().String(),
		ClientId:     clientId,
		Phone:        req.Phone,
		Email:        req.Email,
		Name:         req.Name,
		Tags:         string(tagsBytes),
		CustomFields: marshalCustomFields(req.CustomFields),
		Status:       "active",
		CreatedBy:    authInfo.UserId,
		UpdatedBy:    authInfo.UserId,
	}

	// Save to DB
	err := contact.GetRepo().Create(mdl).Error
	if err != nil {
		customLogger.ErrorWithData("Failed to create contact", map[string]interface{}{
			"component":  "ContactsService",
			"function":   "Create",
			"contact_id": mdl.ID,
			"error":      err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	customLogger.InfoWithData("Contact created successfully", map[string]interface{}{
		"component":  "ContactsService",
		"function":   "Create",
		"contact_id": mdl.ID,
		"name":       mdl.Name,
	})
	return o.toDTO(mdl), nil
}

func (o *ContactsSvcImpl) Update(contactId string, req dtos.ContactUpdateReq, authInfo auth.AuthInfo) (*dtos.Contact, *gocom.CodedError) {
	customLogger.InfoWithData("Update contact started", map[string]interface{}{
		"component":  "ContactsService",
		"function":   "Update",
		"contact_id": contactId,
		"client_id":  authInfo.ClientId,
	})

	clientId := req.ClientId
	if common.PROVIDER_ID != authInfo.ClientId {
		clientId = authInfo.ClientId
	}

	if clientId == "" {
		customLogger.WarnWithData("Empty clientId", map[string]interface{}{
			"component": "ContactsService",
			"function":  "Update",
		})
		return nil, common.ERR_INVALID_REQUEST
	}

	// Get existing contact
	mdl := contact.GetRepo().GetById(contactId)
	if mdl == nil {
		customLogger.WarnWithData("Contact not found", map[string]interface{}{
			"component":  "ContactsService",
			"function":   "Update",
			"contact_id": contactId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check duplicate phone or email if changed
	if req.Phone != "" && req.Phone != mdl.Phone {
		existing := contact.GetRepo().GetByClientIdAndPhone(authInfo.ClientId, req.Phone)
		if existing != nil && existing.ID != contactId {
			customLogger.WarnWithData("Phone already exists", map[string]interface{}{
				"component": "ContactsService",
				"function":  "Update",
				"phone":     req.Phone,
			})
			return nil, gocom.NewError(409, "Phone already exists")
		}
		mdl.Phone = req.Phone
	}

	if req.Email != "" && req.Email != mdl.Email {
		existing := contact.GetRepo().GetByClientIdAndEmail(authInfo.ClientId, req.Email)
		if existing != nil && existing.ID != contactId {
			customLogger.WarnWithData("Email already exists", map[string]interface{}{
				"component": "ContactsService",
				"function":  "Update",
				"email":     req.Email,
			})
			return nil, gocom.NewError(409, "Email already exists")
		}
		mdl.Email = req.Email
	}

	// Update fields
	if req.Name != "" {
		mdl.Name = req.Name
	}
	if req.Tags != nil {
		tagsBytes, _ := json.Marshal(req.Tags)
		mdl.Tags = string(tagsBytes)
	}
	if req.CustomFields != nil {
		mdl.CustomFields = marshalCustomFields(req.CustomFields)
	}
	if req.Status != "" {
		mdl.Status = req.Status
	}
	mdl.UpdatedBy = authInfo.UserId
	mdl.ClientId = clientId

	err := contact.GetRepo().Update(mdl)
	if err != nil {
		customLogger.ErrorWithData("Failed to update contact", map[string]interface{}{
			"component":  "ContactsService",
			"function":   "Update",
			"contact_id": contactId,
			"error":      err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	customLogger.InfoWithData("Contact updated successfully", map[string]interface{}{
		"component":  "ContactsService",
		"function":   "Update",
		"contact_id": mdl.ID,
		"name":       mdl.Name,
	})
	return o.toDTO(mdl), nil
}

func (o *ContactsSvcImpl) GetById(contactId string, authInfo auth.AuthInfo) (*dtos.Contact, *gocom.CodedError) {
	customLogger.DebugWithData("GetById started", map[string]interface{}{
		"component":  "ContactsService",
		"function":   "GetById",
		"contact_id": contactId,
	})

	mdl := contact.GetRepo().GetById(contactId)
	if mdl == nil {
		customLogger.WarnWithData("Contact not found", map[string]interface{}{
			"component":  "ContactsService",
			"function":   "GetById",
			"contact_id": contactId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if common.PROVIDER_ID != authInfo.ClientId {
		if mdl.ClientId != authInfo.ClientId {
			customLogger.WarnWithData("Unauthorized", map[string]interface{}{
				"component":         "ContactsService",
				"function":          "GetById",
				"contact_client_id": mdl.ClientId,
				"auth_client_id":    authInfo.ClientId,
			})
			return nil, common.ERR_NOT_ALLOWED
		}
	}

	customLogger.DebugWithData("Contact found", map[string]interface{}{
		"component":  "ContactsService",
		"function":   "GetById",
		"contact_id": mdl.ID,
		"name":       mdl.Name,
	})
	return o.toDTO(mdl), nil
}

func (o *ContactsSvcImpl) Search(filter, phone, email, status, clientId string, tags []string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.Contact, bool, int) {
	customLogger.DebugWithData("Search started", map[string]interface{}{
		"component": "ContactsService",
		"function":  "Search",
		"filter":    filter,
		"phone":     phone,
		"email":     email,
		"status":    status,
		"page":      pageNo,
		"rows":      rowPerPage,
	})

	if common.PROVIDER_ID != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized, using auth clientId", map[string]interface{}{
			"component": "ContactsService",
			"function":  "Search",
			"client_id": authInfo.ClientId,
		})
		clientId = authInfo.ClientId
	}

	//if clientId == "" {
	//	logger.Warnf("ContactsService.Search - Empty clientId")
	//	return nil, false, 0
	//}

	// silence unused parameter for now
	_ = tags

	if rowPerPage <= 0 {
		rowPerPage = config.GetInt(constans.StaticDefaultEnv)
	}
	if pageNo <= 0 {
		pageNo = 1
	}

	var tagsStr string
	if len(tags) > 0 {
		tagsStr = tags[0] // Use the first tag for searching
	}

	mdls, haveNext, count := contact.GetRepo().Search(filter, clientId, status, tagsStr, pageNo, rowPerPage)

	ret := make([]*dtos.Contact, len(mdls))
	for i, mdl := range mdls {
		ret[i] = o.toDTO(&mdl)
	}

	customLogger.InfoWithData("Search completed", map[string]interface{}{
		"component": "ContactsService",
		"function":  "Search",
		"count":     len(ret),
		"have_next": haveNext,
		"total":     count,
	})
	return ret, haveNext, int(count)
}

func (o *ContactsSvcImpl) Delete(contactId string, authInfo auth.AuthInfo) *gocom.CodedError {
	customLogger.InfoWithData("Delete contact started", map[string]interface{}{
		"component":  "ContactsService",
		"function":   "Delete",
		"contact_id": contactId,
	})

	// Get existing contact
	mdl := contact.GetRepo().GetById(contactId)
	if mdl == nil {
		customLogger.WarnWithData("Contact not found", map[string]interface{}{
			"component":  "ContactsService",
			"function":   "Delete",
			"contact_id": contactId,
		})
		return common.ERR_NOT_FOUND
	}

	// Check permission
	if common.PROVIDER_ID != authInfo.ClientId {
		if mdl.ClientId != authInfo.ClientId {
			customLogger.WarnWithData("Unauthorized", map[string]interface{}{
				"component":         "ContactsService",
				"function":          "Delete",
				"contact_client_id": mdl.ClientId,
				"auth_client_id":    authInfo.ClientId,
			})
			return common.ERR_NOT_ALLOWED
		}
	}

	err := contact.GetRepo().Delete(contactId)
	if err != nil {
		customLogger.ErrorWithData("Failed to delete contact", map[string]interface{}{
			"component":  "ContactsService",
			"function":   "Delete",
			"contact_id": contactId,
			"error":      err.Error(),
		})
		return common.ERR_UNABLE_TO_DELETE
	}

	customLogger.InfoWithData("Contact deleted successfully", map[string]interface{}{
		"component":  "ContactsService",
		"function":   "Delete",
		"contact_id": contactId,
	})
	return nil
}

func (o *ContactsSvcImpl) ImportFromCSV(req dtos.ContactImportReq, authInfo auth.AuthInfo) (*dtos.ContactImportResp, *gocom.CodedError) {
	customLogger.InfoWithData("ImportFromCSV started", map[string]interface{}{
		"component": "ContactsService",
		"function":  "ImportFromCSV",
		"row_count": len(req.FileData),
		"client_id": authInfo.ClientId,
	})

	result := &dtos.ContactImportResp{
		TotalRows:      len(req.FileData),
		SuccessCount:   0,
		FailedCount:    0,
		FailedRows:     []int{},
		FailedMessages: []string{},
	}

	for rowNum, row := range req.FileData {
		// Validation
		if row.Name == "" {
			result.FailedCount++
			result.FailedRows = append(result.FailedRows, rowNum+2)
			result.FailedMessages = append(result.FailedMessages, "Name is required")
			continue
		}

		if row.Phone == "" && row.Email == "" {
			result.FailedCount++
			result.FailedRows = append(result.FailedRows, rowNum+2)
			result.FailedMessages = append(result.FailedMessages, "At least phone or email is required")
			continue
		}

		// Check duplicate
		isDuplicate := false
		if row.Phone != "" {
			existing := contact.GetRepo().GetByClientIdAndPhone(authInfo.ClientId, row.Phone)
			if existing != nil {
				isDuplicate = true
			}
		}
		if !isDuplicate && row.Email != "" {
			existing := contact.GetRepo().GetByClientIdAndEmail(authInfo.ClientId, row.Email)
			if existing != nil {
				isDuplicate = true
			}
		}

		if isDuplicate {
			result.FailedCount++
			result.FailedRows = append(result.FailedRows, rowNum+2)
			result.FailedMessages = append(result.FailedMessages, "Contact already exists")
			continue
		}

		// Parse tags
		var tags []string
		if row.Tags != "" {
			tags = strings.Split(row.Tags, ",")
			for i, tag := range tags {
				tags[i] = strings.TrimSpace(tag)
			}
		}
		// Set default tags if empty
		if len(tags) == 0 {
			tags = []string{"Instant Link"}
		}

		// Set default customFields if empty
		customFields := row.CustomFields
		if customFields == nil || len(customFields) == 0 {
			customFields = map[string]interface{}{
				"city": "Tangerang Selatan",
			}
		}

		// Create contact
		tagsBytes, _ := json.Marshal(tags)
		mdl := &contact.Contact{
			ID:           ulid.Make().String(),
			ClientId:     authInfo.ClientId,
			Phone:        row.Phone,
			Email:        row.Email,
			Name:         row.Name,
			Tags:         string(tagsBytes),
			CustomFields: marshalCustomFields(customFields),
			Status:       "active",
			CreatedBy:    authInfo.UserId,
			UpdatedBy:    authInfo.UserId,
		}

		err := contact.GetRepo().Create(mdl)
		if err != nil {
			customLogger.ErrorWithData("Failed to create contact during import", map[string]interface{}{
				"component": "ContactsService",
				"function":  "ImportFromCSV",
				"row":       rowNum + 2,
				"phone":     row.Phone,
				"error":     err.Error,
			})
			result.FailedCount++
			result.FailedRows = append(result.FailedRows, rowNum+2)
			result.FailedMessages = append(result.FailedMessages, "Failed to save to database")
			continue
		}

		result.SuccessCount++
	}

	customLogger.InfoWithData("ImportFromCSV completed", map[string]interface{}{
		"component":     "ContactsService",
		"function":      "ImportFromCSV",
		"success_count": result.SuccessCount,
		"failed_count":  result.FailedCount,
	})
	return result, nil
}

func (o *ContactsSvcImpl) BulkDelete(contactIds []string, authInfo auth.AuthInfo) *gocom.CodedError {
	customLogger.InfoWithData("BulkDelete started", map[string]interface{}{
		"component":     "ContactsService",
		"function":      "BulkDelete",
		"contact_count": len(contactIds),
	})

	if len(contactIds) == 0 {
		customLogger.WarnWithData("No contact IDs provided", map[string]interface{}{
			"component": "ContactsService",
			"function":  "BulkDelete",
		})
		return gocom.NewError(400, "No contact IDs provided")
	}

	// Verify all contacts belong to client
	for _, contactId := range contactIds {
		mdl := contact.GetRepo().GetById(contactId)
		if mdl == nil {
			customLogger.WarnWithData("Contact not found", map[string]interface{}{
				"component":  "ContactsService",
				"function":   "BulkDelete",
				"contact_id": contactId,
			})
			return gocom.NewError(404, fmt.Sprintf("Contact %s not found", contactId))
		}
		if mdl.ClientId != authInfo.ClientId {
			customLogger.WarnWithData("Unauthorized", map[string]interface{}{
				"component":         "ContactsService",
				"function":          "BulkDelete",
				"contact_client_id": mdl.ClientId,
				"auth_client_id":    authInfo.ClientId,
			})
			return common.ERR_NOT_ALLOWED
		}
	}

	// Delete all
	for _, contactId := range contactIds {
		err := contact.GetRepo().Delete(contactId)
		if err != nil {
			customLogger.ErrorWithData("Failed to delete contact", map[string]interface{}{
				"component":  "ContactsService",
				"function":   "BulkDelete",
				"contact_id": contactId,
				"error":      err.Error(),
			})
			return common.ERR_UNABLE_TO_DELETE
		}
	}

	customLogger.InfoWithData("BulkDelete completed", map[string]interface{}{
		"component":     "ContactsService",
		"function":      "BulkDelete",
		"contact_count": len(contactIds),
	})
	return nil
}

func (o *ContactsSvcImpl) BulkUpdateStatus(contactIds []string, status string, authInfo auth.AuthInfo) *gocom.CodedError {
	customLogger.InfoWithData("BulkUpdateStatus started", map[string]interface{}{
		"component":     "ContactsService",
		"function":      "BulkUpdateStatus",
		"contact_count": len(contactIds),
		"status":        status,
	})

	if len(contactIds) == 0 {
		customLogger.WarnWithData("No contact IDs provided", map[string]interface{}{
			"component": "ContactsService",
			"function":  "BulkUpdateStatus",
		})
		return gocom.NewError(400, "No contact IDs provided")
	}

	if status != "active" && status != "inactive" && status != "blocked" {
		customLogger.WarnWithData("Invalid status", map[string]interface{}{
			"component": "ContactsService",
			"function":  "BulkUpdateStatus",
			"status":    status,
		})
		return gocom.NewError(400, "Invalid status. Must be: active, inactive, or blocked")
	}

	// Verify all contacts belong to client and update
	for _, contactId := range contactIds {
		mdl := contact.GetRepo().GetById(contactId)
		if mdl == nil {
			customLogger.WarnWithData("Contact not found", map[string]interface{}{
				"component":  "ContactsService",
				"function":   "BulkUpdateStatus",
				"contact_id": contactId,
			})
			return gocom.NewError(404, fmt.Sprintf("Contact %s not found", contactId))
		}
		if mdl.ClientId != authInfo.ClientId {
			customLogger.WarnWithData("Unauthorized", map[string]interface{}{
				"component":         "ContactsService",
				"function":          "BulkUpdateStatus",
				"contact_client_id": mdl.ClientId,
				"auth_client_id":    authInfo.ClientId,
			})
			return common.ERR_NOT_ALLOWED
		}

		mdl.Status = status
		mdl.UpdatedBy = authInfo.UserId
		err := contact.GetRepo().Update(mdl)
		if err != nil {
			customLogger.ErrorWithData("Failed to update contact", map[string]interface{}{
				"component":  "ContactsService",
				"function":   "BulkUpdateStatus",
				"contact_id": contactId,
				"error":      err.Error(),
			})
			return common.ERR_UNABLE_TO_UPDATE
		}
	}

	customLogger.InfoWithData("BulkUpdateStatus completed", map[string]interface{}{
		"component":     "ContactsService",
		"function":      "BulkUpdateStatus",
		"contact_count": len(contactIds),
		"status":        status,
	})
	return nil
}

func (o *ContactsSvcImpl) UploadContactsFile(file *multipart.FileHeader, campaignId string, authInfo auth.AuthInfo) (*dtos.ContactUploadFileResp, *gocom.CodedError) {
	customLogger.InfoWithData("UploadContactsFile started", map[string]interface{}{
		"component":   "ContactsService",
		"function":    "UploadContactsFile",
		"filename":    file.Filename,
		"size":        file.Size,
		"campaign_id": campaignId,
		"client_id":   authInfo.ClientId,
	})

	// Validate file type
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".csv" {
		customLogger.WarnWithData("Invalid file type", map[string]interface{}{
			"component": "ContactsService",
			"function":  "UploadContactsFile",
			"extension": ext,
			"filename":  file.Filename,
		})
		return nil, &gocom.CodedError{Code: 400, Message: "File must be CSV format"}
	}

	// Validate campaign exists and status is draft
	campaignMdl := campaign.GetRepo().GetById(campaignId)
	if campaignMdl == nil {
		customLogger.WarnWithData("Campaign not found", map[string]interface{}{
			"component":   "ContactsService",
			"function":    "UploadContactsFile",
			"campaign_id": campaignId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if campaignMdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized campaign access", map[string]interface{}{
			"component":          "ContactsService",
			"function":           "UploadContactsFile",
			"campaign_client_id": campaignMdl.ClientId,
			"auth_client_id":     authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Validate campaign status
	if campaignMdl.Status != "draft" {
		customLogger.WarnWithData("Campaign is not in draft status", map[string]interface{}{
			"component":   "ContactsService",
			"function":    "UploadContactsFile",
			"campaign_id": campaignId,
			"status":      campaignMdl.Status,
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Campaign must be in draft status"}
	}

	// Parse CSV file
	rows, parseErr := o.parseCSV(file)
	if parseErr != nil {
		customLogger.ErrorWithData("Failed to parse file", map[string]interface{}{
			"component": "ContactsService",
			"function":  "UploadContactsFile",
			"error":     parseErr.Error(),
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Failed to parse file: " + parseErr.Error()}
	}

	// Validate headers
	if len(rows) == 0 {
		return nil, &gocom.CodedError{Code: 400, Message: "File is empty"}
	}

	// Check required headers
	firstRow := rows[0]
	requiredHeaders := []string{"name", "phone_number"}

	for _, header := range requiredHeaders {
		if _, exists := firstRow[header]; !exists {
			customLogger.WarnWithData("Missing required header", map[string]interface{}{
				"component": "ContactsService",
				"function":  "UploadContactsFile",
				"header":    header,
			})
			return nil, &gocom.CodedError{Code: 400, Message: fmt.Sprintf("Missing required header: %s", header)}
		}
	}

	customLogger.InfoWithData("File parsed successfully", map[string]interface{}{
		"component": "ContactsService",
		"function":  "UploadContactsFile",
		"row_count": len(rows),
	})

	// Process rows concurrently
	result := &dtos.ContactUploadFileResp{
		TotalRows:       len(rows),
		ContactsCreated: 0,
		TargetsCreated:  0,
		FailedCount:     0,
		FailedRows:      []int{},
		FailedMessages:  []string{},
		CampaignId:      campaignId,
	}

	// Use channels for concurrent processing
	type processResult struct {
		rowNum    int
		contactId string
		err       string
	}

	resultChan := make(chan processResult, len(rows))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrent goroutines to 10

	for rowNum, row := range rows {
		wg.Add(1)
		go func(idx int, data map[string]string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			// Validate row data
			name := strings.TrimSpace(data["name"])
			phoneNumber := strings.TrimSpace(data["phone_number"])
			email := strings.TrimSpace(data["email"])
			address := strings.TrimSpace(data["address"])

			if name == "" {
				resultChan <- processResult{rowNum: idx + 2, err: "Name is required"}
				return
			}

			if phoneNumber == "" {
				resultChan <- processResult{rowNum: idx + 2, err: "Phone number is required"}
				return
			}

			// Check if contact already exists
			existing := contact.GetRepo().GetByClientIdAndPhone(authInfo.ClientId, phoneNumber)
			var contactId string

			if existing != nil {
				// Contact exists, use existing ID
				contactId = existing.ID
				customLogger.DebugWithData("Contact already exists, using existing", map[string]interface{}{
					"component":  "ContactsService",
					"function":   "UploadContactsFile",
					"contact_id": contactId,
					"phone":      phoneNumber,
				})
			} else {
				// Create new contact
				contactId = ulid.Make().String()
				now := time.Now()

				customFieldsMap := make(map[string]interface{})
				if address != "" {
					customFieldsMap["address"] = address
				}

				newContact := &contact.Contact{
					ID:           contactId,
					ClientId:     authInfo.ClientId,
					Phone:        phoneNumber,
					Email:        email,
					Name:         name,
					Tags:         "[]",
					CustomFields: marshalCustomFields(customFieldsMap),
					Status:       "active",
					CreatedBy:    authInfo.UserId,
					CreatedAt:    now,
					UpdatedAt:    now,
				}

				err := contact.GetRepo().Create(newContact).Error
				if err != nil {
					customLogger.ErrorWithData("Failed to create contact", map[string]interface{}{
						"component": "ContactsService",
						"function":  "UploadContactsFile",
						"error":     err.Error(),
						"phone":     phoneNumber,
					})
					resultChan <- processResult{rowNum: idx + 2, err: "Failed to create contact"}
					return
				}
			}

			resultChan <- processResult{rowNum: idx + 2, contactId: contactId}
		}(rowNum, row)
	}

	// Wait for all goroutines to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var contactIds []string
	for res := range resultChan {
		if res.err != "" {
			result.FailedCount++
			result.FailedRows = append(result.FailedRows, res.rowNum)
			result.FailedMessages = append(result.FailedMessages, res.err)
		} else {
			result.ContactsCreated++
			contactIds = append(contactIds, res.contactId)
		}
	}

	// Insert into campaign_recipients
	customLogger.InfoWithData("Creating campaign recipients", map[string]interface{}{
		"component":     "ContactsService",
		"function":      "UploadContactsFile",
		"campaign_id":   campaignId,
		"contact_count": len(contactIds),
	})

	recipientType := ""
	recipientValueField := ""
	switch campaignMdl.Type {
	case "whatsapp", "sms":
		recipientType = "phone"
		recipientValueField = "phone"
	case "email":
		recipientType = "email"
		recipientValueField = "email"
	}

	campaignRecipients := []string{}
	for _, contactId := range contactIds {
		contactMdl := contact.GetRepo().GetById(contactId)
		if contactMdl == nil {
			result.FailedCount++
			result.FailedMessages = append(result.FailedMessages, fmt.Sprintf("Contact %s not found", contactId))
			continue
		}

		recipientValue := ""
		switch recipientValueField {
		case "phone":
			recipientValue = contactMdl.Phone
		case "email":
			recipientValue = contactMdl.Email
		}

		if recipientValue == "" {
			result.FailedCount++
			result.FailedMessages = append(result.FailedMessages, fmt.Sprintf("Contact %s has no %s", contactId, recipientType))
			continue
		}

		recipient := &campaignRecipient.CampaignRecipient{
			ID:             ulid.Make().String(),
			CampaignId:     campaignId,
			ContactId:      contactId,
			RecipientType:  recipientType,
			RecipientValue: recipientValue,
			Status:         "pending",
			Parameters:     campaignMdl.Parameters,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := campaignRecipient.GetRepo().Create(recipient).Error
		if err != nil {
			customLogger.ErrorWithData("Failed to create campaign recipient", map[string]interface{}{
				"component":  "ContactsService",
				"function":   "UploadContactsFile",
				"error":      err.Error(),
				"contact_id": contactId,
			})
			continue
		}

		result.TargetsCreated++
		campaignRecipients = append(campaignRecipients, contactId)
	}

	// Update campaign total_recipients
	campaignMdl.TotalRecipients = len(campaignRecipients)
	if updateErr := campaign.GetRepo().Update(campaignMdl); updateErr != nil {
		customLogger.ErrorWithData("Failed to update campaign total_recipients", map[string]interface{}{
			"component":        "ContactsService",
			"function":         "UploadContactsFile",
			"campaign_id":      campaignId,
			"total_recipients": len(campaignRecipients),
			"error":            updateErr.Error(),
		})
	}

	// Put into cache
	cacheKey := utils.GetCacheKey(fmt.Sprintf("campaign-recipient:campaign:%s", campaignId))
	cacheData, _ := json.Marshal(campaignRecipients)
	cacheErr := gocom.KeyVal().Set(cacheKey, string(cacheData), time.Hour*24) // Cache for 24 hours

	if cacheErr != nil {
		customLogger.WarnWithData("Failed to cache campaign recipients", map[string]interface{}{
			"component":   "ContactsService",
			"function":    "UploadContactsFile",
			"campaign_id": campaignId,
			"cache_key":   cacheKey,
			"error":       cacheErr.Error(),
		})
	} else {
		customLogger.InfoWithData("Campaign recipients cached", map[string]interface{}{
			"component":       "ContactsService",
			"function":        "UploadContactsFile",
			"campaign_id":     campaignId,
			"cache_key":       cacheKey,
			"recipient_count": len(campaignRecipients),
		})
	}

	result.CacheKey = cacheKey

	customLogger.InfoWithData("UploadContactsFile completed", map[string]interface{}{
		"component":        "ContactsService",
		"function":         "UploadContactsFile",
		"contacts_created": result.ContactsCreated,
		"targets_created":  result.TargetsCreated,
		"failed_count":     result.FailedCount,
	})

	return result, nil
}

func (o *ContactsSvcImpl) parseCSV(file *multipart.FileHeader) ([]map[string]string, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	reader := csv.NewReader(src)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("file must have at least header and one data row")
	}

	// Parse headers (first row)
	headers := rows[0]
	for i := range headers {
		headers[i] = strings.ToLower(strings.TrimSpace(headers[i]))
	}

	// Parse data rows
	var result []map[string]string
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		rowMap := make(map[string]string)
		for j, cell := range row {
			if j < len(headers) {
				rowMap[headers[j]] = strings.TrimSpace(cell)
			}
		}
		result = append(result, rowMap)
	}

	return result, nil
}

func (o *ContactsSvcImpl) toDTO(mdl *contact.Contact) *dtos.Contact {
	if mdl == nil {
		return nil
	}

	var createdAtStr, updatedAtStr string
	if !mdl.CreatedAt.IsZero() {
		createdAtStr = mdl.CreatedAt.Format(time.RFC3339)
	}
	if !mdl.UpdatedAt.IsZero() {
		updatedAtStr = mdl.UpdatedAt.Format(time.RFC3339)
	}

	// convert tags JSON to []string
	var tags []string
	if mdl.Tags != "" {
		if err := json.Unmarshal([]byte(mdl.Tags), &tags); err != nil {
			customLogger.DebugWithData("Unable to unmarshal tags", map[string]interface{}{
				"component": "ContactsService",
				"function":  "toDTO",
				"error":     err.Error(),
			})
			tags = []string{}
		}
	}

	return &dtos.Contact{
		ID:           mdl.ID,
		ClientId:     mdl.ClientId,
		Phone:        mdl.Phone,
		Email:        mdl.Email,
		Name:         mdl.Name,
		Tags:         tags,
		CustomFields: unmarshalCustomFields(mdl.CustomFields),
		Status:       mdl.Status,
		CreatedBy:    mdl.CreatedBy,
		UpdatedBy:    mdl.UpdatedBy,
		CreatedAt:    createdAtStr,
		UpdatedAt:    updatedAtStr,
	}
}
