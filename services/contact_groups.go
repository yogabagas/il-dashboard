package services

import (
	"sync"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/config"
	"github.com/oklog/ulid/v2"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/contact"
	"gitlab.com/anti_metter/switching_common/contactGroup"
	"gitlab.com/anti_metter/switching_common/contactGroupMember"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
)

type ContactGroupsService interface {
	Create(req dtos.ContactGroupReq, authInfo auth.AuthInfo) (*dtos.ContactGroup, *gocom.CodedError)
	Update(groupId string, req dtos.ContactGroupUpdateReq, authInfo auth.AuthInfo) (*dtos.ContactGroup, *gocom.CodedError)
	GetById(groupId string, authInfo auth.AuthInfo) (*dtos.ContactGroup, *gocom.CodedError)
	Search(filter, clientId string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.ContactGroup, bool, int)
	Delete(groupId string, authInfo auth.AuthInfo) *gocom.CodedError
	AddMembers(groupId string, contactIds []string, authInfo auth.AuthInfo) *gocom.CodedError
	RemoveMembers(groupId string, contactIds []string, authInfo auth.AuthInfo) *gocom.CodedError
	GetMembers(groupId string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.Contact, bool, int)
}

type ContactGroupsSvcImpl struct{}

var contactGroupsService ContactGroupsService
var onceContactGroupsService sync.Once

func GetContactGroupsService() ContactGroupsService {
	onceContactGroupsService.Do(func() {
		contactGroupsService = &ContactGroupsSvcImpl{}
	})
	return contactGroupsService
}

func (o *ContactGroupsSvcImpl) Create(req dtos.ContactGroupReq, authInfo auth.AuthInfo) (*dtos.ContactGroup, *gocom.CodedError) {
	customLogger.InfoWithData("Create contact group started", map[string]interface{}{
		"component": "ContactGroupsService",
		"function":  "Create",
		"name":      req.Name,
		"client_id": authInfo.ClientId,
	})

	// Validation
	if req.Name == "" {
		customLogger.WarnWithData("Name is required", map[string]interface{}{
			"component": "ContactGroupsService",
			"function":  "Create",
		})
		return nil, common.ERR_INVALID_REQUEST
	}

	// Check duplicate name
	existing := contactGroup.GetRepo().GetByClientIdAndName(authInfo.ClientId, req.Name)
	if existing != nil {
		customLogger.WarnWithData("Group name already exists", map[string]interface{}{
			"component": "ContactGroupsService",
			"function":  "Create",
			"name":      req.Name,
		})
		return nil, gocom.NewError(409, "Group name already exists")
	}

	// Create model
	mdl := &contactGroup.ContactGroup{
		ID:          ulid.Make().String(),
		ClientId:    authInfo.ClientId,
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   authInfo.UserId,
		UpdatedBy:   authInfo.UserId,
	}

	err := contactGroup.GetRepo().Create(mdl).Error
	if err != nil {
		customLogger.ErrorWithData("Failed to create group", map[string]interface{}{
			"component": "ContactGroupsService",
			"function":  "Create",
			"group_id":  mdl.ID,
			"error":     err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	customLogger.InfoWithData("Contact group created successfully", map[string]interface{}{
		"component": "ContactGroupsService",
		"function":  "Create",
		"group_id":  mdl.ID,
		"name":      mdl.Name,
	})
	return o.toDTO(mdl), nil
}

func (o *ContactGroupsSvcImpl) Update(groupId string, req dtos.ContactGroupUpdateReq, authInfo auth.AuthInfo) (*dtos.ContactGroup, *gocom.CodedError) {
	customLogger.InfoWithData("Update contact group started", map[string]interface{}{
		"component": "ContactGroupsService",
		"function":  "Update",
		"group_id":  groupId,
	})

	// Get existing group
	mdl := contactGroup.GetRepo().GetById(groupId)
	if mdl == nil {
		customLogger.WarnWithData("Group not found", map[string]interface{}{
			"component": "ContactGroupsService",
			"function":  "Update",
			"group_id":  groupId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if mdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":       "ContactGroupsService",
			"function":        "Update",
			"group_client_id": mdl.ClientId,
			"auth_client_id":  authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Check duplicate name if changed
	if req.Name != "" && req.Name != mdl.Name {
		existing := contactGroup.GetRepo().GetByClientIdAndName(authInfo.ClientId, req.Name)
		if existing != nil && existing.ID != groupId {
			customLogger.WarnWithData("Group name already exists", map[string]interface{}{
				"component": "ContactGroupsService",
				"function":  "Update",
				"name":      req.Name,
			})
			return nil, gocom.NewError(409, "Group name already exists")
		}
		mdl.Name = req.Name
	}

	// Update fields
	if req.Description != "" {
		mdl.Description = req.Description
	}
	mdl.UpdatedBy = authInfo.UserId

	err := contactGroup.GetRepo().Update(mdl)
	if err != nil {
		customLogger.ErrorWithData("Failed to update group", map[string]interface{}{
			"component": "ContactGroupsService",
			"function":  "Update",
			"group_id":  groupId,
			"error":     err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	customLogger.InfoWithData("Contact group updated successfully", map[string]interface{}{
		"component": "ContactGroupsService",
		"function":  "Update",
		"group_id":  mdl.ID,
		"name":      mdl.Name,
	})
	return o.toDTO(mdl), nil
}

func (o *ContactGroupsSvcImpl) GetById(groupId string, authInfo auth.AuthInfo) (*dtos.ContactGroup, *gocom.CodedError) {
	customLogger.DebugWithData("GetById started", map[string]interface{}{
		"component": "ContactGroupsService",
		"function":  "GetById",
		"group_id":  groupId,
	})

	mdl := contactGroup.GetRepo().GetById(groupId)
	if mdl == nil {
		customLogger.WarnWithData("Group not found", map[string]interface{}{
			"component": "ContactGroupsService",
			"function":  "GetById",
			"group_id":  groupId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if mdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":       "ContactGroupsService",
			"function":        "GetById",
			"group_client_id": mdl.ClientId,
			"auth_client_id":  authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	customLogger.DebugWithData("Group found", map[string]interface{}{
		"component": "ContactGroupsService",
		"function":  "GetById",
		"group_id":  mdl.ID,
		"name":      mdl.Name,
	})
	return o.toDTO(mdl), nil
}

func (o *ContactGroupsSvcImpl) Search(filter, clientId string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.ContactGroup, bool, int) {
	if rowPerPage <= 0 {
		rowPerPage = config.GetInt(constans.StaticDefaultEnv)
	}
	if pageNo <= 0 {
		pageNo = 1
	}

	if authInfo.ClientId != common.PROVIDER_ID {
		clientId = authInfo.ClientId
	}

	mdls, haveNext, count := contactGroup.GetRepo().Search(filter, clientId, pageNo, rowPerPage)

	ret := make([]*dtos.ContactGroup, len(mdls))
	for i, mdl := range mdls {
		ret[i] = o.toDTO(&mdl)
	}

	customLogger.DebugWithData("Search completed", map[string]interface{}{
		"component": "ContactGroupsService",
		"function":  "Search",
		"count":     len(ret),
		"have_next": haveNext,
	})
	return ret, haveNext, int(count)
}

func (o *ContactGroupsSvcImpl) Delete(groupId string, authInfo auth.AuthInfo) *gocom.CodedError {
	customLogger.InfoWithData("Delete group started", map[string]interface{}{
		"component": "ContactGroupsService",
		"function":  "Delete",
		"group_id":  groupId,
	})

	// Get existing group
	mdl := contactGroup.GetRepo().GetById(groupId)
	if mdl == nil {
		customLogger.WarnWithData("Group not found", map[string]interface{}{
			"component": "ContactGroupsService",
			"function":  "Delete",
			"group_id":  groupId,
		})
		return common.ERR_NOT_FOUND
	}

	// Check permission
	if mdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":       "ContactGroupsService",
			"function":        "Delete",
			"group_client_id": mdl.ClientId,
			"auth_client_id":  authInfo.ClientId,
		})
		return common.ERR_NOT_ALLOWED
	}

	// Delete all members first
	err := contactGroupMember.GetRepo().DeleteByGroupId(groupId)
	if err != nil {
		customLogger.ErrorWithData("Failed to delete group members", map[string]interface{}{
			"component": "ContactGroupsService",
			"function":  "Delete",
			"group_id":  groupId,
			"error":     err.Error(),
		})
		return common.ERR_UNABLE_TO_DELETE
	}

	// Delete group
	err = contactGroup.GetRepo().Delete(groupId)
	if err != nil {
		customLogger.ErrorWithData("Failed to delete group", map[string]interface{}{
			"component": "ContactGroupsService",
			"function":  "Delete",
			"group_id":  groupId,
			"error":     err.Error(),
		})
		return common.ERR_UNABLE_TO_DELETE
	}

	customLogger.InfoWithData("Group deleted successfully", map[string]interface{}{
		"component": "ContactGroupsService",
		"function":  "Delete",
		"group_id":  groupId,
	})
	return nil
}

func (o *ContactGroupsSvcImpl) AddMembers(groupId string, contactIds []string, authInfo auth.AuthInfo) *gocom.CodedError {
	customLogger.InfoWithData("AddMembers started", map[string]interface{}{
		"component":     "ContactGroupsService",
		"function":      "AddMembers",
		"group_id":      groupId,
		"contact_count": len(contactIds),
	})

	// Verify group exists and belongs to client
	group := contactGroup.GetRepo().GetById(groupId)
	if group == nil {
		customLogger.WarnWithData("Group not found", map[string]interface{}{
			"component": "ContactGroupsService",
			"function":  "AddMembers",
			"group_id":  groupId,
		})
		return common.ERR_NOT_FOUND
	}
	if group.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":       "ContactGroupsService",
			"function":        "AddMembers",
			"group_client_id": group.ClientId,
			"auth_client_id":  authInfo.ClientId,
		})
		return common.ERR_NOT_ALLOWED
	}

	// Verify all contacts exist and belong to client
	for _, contactId := range contactIds {
		contactMdl := contact.GetRepo().GetById(contactId)
		if contactMdl == nil {
			customLogger.WarnWithData("Contact not found", map[string]interface{}{
				"component":  "ContactGroupsService",
				"function":   "AddMembers",
				"contact_id": contactId,
			})
			return gocom.NewError(404, "Contact "+contactId+" not found")
		}
		if contactMdl.ClientId != authInfo.ClientId {
			customLogger.WarnWithData("Contact unauthorized", map[string]interface{}{
				"component":         "ContactGroupsService",
				"function":          "AddMembers",
				"contact_client_id": contactMdl.ClientId,
				"auth_client_id":    authInfo.ClientId,
			})
			return common.ERR_NOT_ALLOWED
		}

		// Check if already member
		existing := contactGroupMember.GetRepo().GetByGroupIdAndContactId(groupId, contactId)
		if existing != nil {
			customLogger.DebugWithData("Contact already member, skipping", map[string]interface{}{
				"component":  "ContactGroupsService",
				"function":   "AddMembers",
				"group_id":   groupId,
				"contact_id": contactId,
			})
			continue
		}

		// Add member
		member := &contactGroupMember.ContactGroupMember{
			ID:        ulid.Make().String(),
			GroupId:   groupId,
			ContactId: contactId,
		}
		err := contactGroupMember.GetRepo().Create(member).Error
		if err != nil {
			customLogger.ErrorWithData("Failed to add member", map[string]interface{}{
				"component":  "ContactGroupsService",
				"function":   "AddMembers",
				"group_id":   groupId,
				"contact_id": contactId,
				"error":      err.Error(),
			})
			return common.ERR_UNABLE_TO_UPDATE
		}
	}

	customLogger.InfoWithData("AddMembers completed", map[string]interface{}{
		"component":     "ContactGroupsService",
		"function":      "AddMembers",
		"group_id":      groupId,
		"contact_count": len(contactIds),
	})
	return nil
}

func (o *ContactGroupsSvcImpl) RemoveMembers(groupId string, contactIds []string, authInfo auth.AuthInfo) *gocom.CodedError {
	customLogger.InfoWithData("RemoveMembers started", map[string]interface{}{
		"component":     "ContactGroupsService",
		"function":      "RemoveMembers",
		"group_id":      groupId,
		"contact_count": len(contactIds),
	})

	// Verify group exists and belongs to client
	group := contactGroup.GetRepo().GetById(groupId)
	if group == nil {
		customLogger.WarnWithData("Group not found", map[string]interface{}{
			"component": "ContactGroupsService",
			"function":  "RemoveMembers",
			"group_id":  groupId,
		})
		return common.ERR_NOT_FOUND
	}
	if group.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":       "ContactGroupsService",
			"function":        "RemoveMembers",
			"group_client_id": group.ClientId,
			"auth_client_id":  authInfo.ClientId,
		})
		return common.ERR_NOT_ALLOWED
	}

	// Remove members
	for _, contactId := range contactIds {
		err := contactGroupMember.GetRepo().DeleteByGroupIdAndContactId(groupId, contactId)
		if err != nil {
			customLogger.ErrorWithData("Failed to remove member", map[string]interface{}{
				"component":  "ContactGroupsService",
				"function":   "RemoveMembers",
				"group_id":   groupId,
				"contact_id": contactId,
				"error":      err.Error(),
			})
			return common.ERR_UNABLE_TO_DELETE
		}
	}

	customLogger.InfoWithData("RemoveMembers completed", map[string]interface{}{
		"component":     "ContactGroupsService",
		"function":      "RemoveMembers",
		"group_id":      groupId,
		"contact_count": len(contactIds),
	})
	return nil
}

func (o *ContactGroupsSvcImpl) GetMembers(groupId string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.Contact, bool, int) {
	// Verify group exists and belongs to client
	group := contactGroup.GetRepo().GetById(groupId)
	if group == nil {
		customLogger.WarnWithData("Group not found", map[string]interface{}{
			"component": "ContactGroupsService",
			"function":  "GetMembers",
			"group_id":  groupId,
		})
		return nil, false, 0
	}
	if group.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":       "ContactGroupsService",
			"function":        "GetMembers",
			"group_client_id": group.ClientId,
			"auth_client_id":  authInfo.ClientId,
		})
		return nil, false, 0
	}

	if rowPerPage <= 0 {
		rowPerPage = config.GetInt(constans.StaticDefaultEnv)
	}
	if pageNo <= 0 {
		pageNo = 1
	}

	contactMdls, haveNext, count := contactGroupMember.GetRepo().GetContactsByGroupId(groupId, pageNo, rowPerPage)

	ret := make([]*dtos.Contact, len(contactMdls))
	for i, contactMdl := range contactMdls {
		ret[i] = GetContactsService().(*ContactsSvcImpl).toDTO(&contactMdl)
	}

	customLogger.DebugWithData("GetMembers completed", map[string]interface{}{
		"component": "ContactGroupsService",
		"function":  "GetMembers",
		"count":     len(ret),
		"have_next": haveNext,
	})
	return ret, haveNext, int(count)
}

func (o *ContactGroupsSvcImpl) toDTO(mdl *contactGroup.ContactGroup) *dtos.ContactGroup {
	if mdl == nil {
		return nil
	}

	// Get total contacts count
	totalContacts := contactGroupMember.GetRepo().CountByGroupId(mdl.ID)

	return &dtos.ContactGroup{
		ID:            mdl.ID,
		ClientId:      mdl.ClientId,
		Name:          mdl.Name,
		Description:   mdl.Description,
		TotalContacts: int(totalContacts),
		CreatedBy:     mdl.CreatedBy,
		UpdatedBy:     mdl.UpdatedBy,
	}
}
