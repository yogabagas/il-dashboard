package services

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/config"
	"github.com/oklog/ulid/v2"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/client"
	"gitlab.com/anti_metter/switching_common/sender"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/outbound"
)

type SendersService interface {
	Create(req dtos.SenderReq, authInfo auth.AuthInfo) (*dtos.Sender, *gocom.CodedError)
	Update(senderId string, req dtos.SenderUpdateReq, authInfo auth.AuthInfo) (*dtos.Sender, *gocom.CodedError)
	GetById(senderId string, authInfo auth.AuthInfo) (*dtos.Sender, *gocom.CodedError)
	Search(filter, senderType, status, clientId string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.Sender, bool, int)
	Delete(senderId string, authInfo auth.AuthInfo) *gocom.CodedError
	VerifySender(phoneNumberId string, code string, authInfo auth.AuthInfo) (bool, *gocom.CodedError)
}

type SendersSvcImpl struct {
	metaOutboundService outbound.MetaOutboundService
}

var sendersService SendersService
var onceSendersService sync.Once

func GetSendersService() SendersService {
	onceSendersService.Do(func() {
		sendersService = &SendersSvcImpl{
			metaOutboundService: outbound.NewMetaOutboundService(),
		}
	})
	return sendersService
}

func marshalSenderConfig(m map[string]interface{}) string {
	if m == nil {
		return ""
	}
	b, err := json.Marshal(m)
	if err != nil {
		customLogger.DebugWithData("Unable to marshal config", map[string]interface{}{
			"component": "SendersService",
			"function":  "marshalSenderConfig",
			"error":     err.Error(),
		})
		return ""
	}
	return string(b)
}

func unmarshalSenderConfig(s string) map[string]interface{} {
	if s == "" {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		customLogger.DebugWithData("Unable to unmarshal config", map[string]interface{}{
			"component": "SendersService",
			"function":  "unmarshalSenderConfig",
			"error":     err.Error(),
		})
		return nil
	}
	return m
}

func (o *SendersSvcImpl) Create(req dtos.SenderReq, authInfo auth.AuthInfo) (*dtos.Sender, *gocom.CodedError) {
	customLogger.InfoWithData("Create sender started", map[string]interface{}{
		"component": "SendersService",
		"function":  "Create",
		"type":      req.Type,
		"name":      req.Name,
	})

	// Validation
	if req.Name == "" {
		customLogger.WarnWithData("Name is required", map[string]interface{}{
			"component": "SendersService",
			"function":  "Create",
		})
		return nil, common.ERR_INVALID_REQUEST
	}

	if req.Type != "whatsapp" && req.Type != "sms" && req.Type != "email" {
		customLogger.WarnWithData("Invalid type", map[string]interface{}{
			"component": "SendersService",
			"function":  "Create",
			"type":      req.Type,
		})
		return nil, gocom.NewError(400, "Type must be: whatsapp, sms, or email")
	}

	if req.Identifier == "" {
		customLogger.WarnWithData("Identifier is required", map[string]interface{}{
			"component": "SendersService",
			"function":  "Create",
		})
		return nil, common.ERR_INVALID_REQUEST
	}

	if len(req.Identifier) <= 2 {
		customLogger.WarnWithData("Identifier is too short", map[string]interface{}{
			"component":  "SendersService",
			"function":   "Create",
			"identifier": req.Identifier,
		})
		return nil, gocom.NewError(400, "Identifier must include country code and phone number")
	}

	// Validate config based on type
	codedErr := o.validateConfig(req.Type, req.Config)
	if codedErr != nil {
		return nil, codedErr
	}

	// Determine client ID based on user role
	// If user is PROVIDER, allow setting clientId from request, otherwise use authInfo.ClientId
	clientId := authInfo.ClientId
	if authInfo.ClientId == common.PROVIDER_ID {
		// Provider can specify clientId, but if empty, use their own clientId
		if req.ClientId != "" {
			clientId = req.ClientId
		}
	}

	// Check duplicate name
	existing := sender.GetRepo().GetByClientIdAndName(clientId, req.Name)
	if existing != nil {
		customLogger.WarnWithData("Sender name already exists", map[string]interface{}{
			"component": "SendersService",
			"function":  "Create",
			"name":      req.Name,
		})
		return nil, gocom.NewError(409, "Sender name already exists")
	}

	// Check duplicate identifier
	existing = sender.GetRepo().GetByClientIdTypeAndIdentifier(clientId, req.Type, req.Identifier)
	if existing != nil {
		customLogger.WarnWithData("Sender identifier already exists", map[string]interface{}{
			"component":  "SendersService",
			"function":   "Create",
			"identifier": req.Identifier,
			"type":       req.Type,
		})
		return nil, gocom.NewError(409, "Sender identifier already exists for this type")
	}

	clientData := client.GetRepo().GetById(clientId)
	if clientData == nil {
		customLogger.WarnWithData("Client not found", map[string]interface{}{
			"component": "SendersService",
			"function":  "Create",
			"client_id": clientId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	metaResp, err := o.metaOutboundService.CreatePhoneNumberToWABA(context.Background(), dtos.MetaSenderReq{
		WabaID:             clientData.WabaId,
		CountryCode:        req.Identifier[0:2],
		MigratePhoneNumber: false,
		PhoneNumber:        req.Identifier[2:],
		VerifiedName:       req.Name,
	})
	if err != nil {
		customLogger.ErrorWithData("Failed to create phone number to WABA", map[string]interface{}{
			"component": "SendersService",
			"function":  "Create",
			"error":     err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	userConfig := map[string]interface{}{
		"phone_number_id": metaResp.ID,
		"access_token":    config.Get(constans.WaAuthToken),
		"waba_id":         clientData.WabaId,
	}

	// Create model
	mdl := &sender.Sender{
		ID:           ulid.Make().String(),
		ClientId:     clientId,
		Type:         req.Type,
		Name:         req.Name,
		Identifier:   req.Identifier,
		Config:       marshalSenderConfig(userConfig),
		Status:       "active",
		IsVerified:   0,
		DailyLimit:   req.DailyLimit,
		MonthlyLimit: req.MonthlyLimit,
		CreatedBy:    authInfo.UserId,
		UpdatedBy:    authInfo.UserId,
	}

	err = sender.GetRepo().Create(mdl).Error
	if err != nil {
		customLogger.ErrorWithData("Failed to create sender", map[string]interface{}{
			"component": "SendersService",
			"function":  "Create",
			"sender_id": mdl.ID,
			"error":     err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	_, err = o.metaOutboundService.RequestOTPCode(context.Background(), dtos.RequestOTPCodeReq{
		CodeMethode: "sms",
		Language:    "id",
	})
	if err != nil {
		customLogger.ErrorWithData("Failed to request OTP code", map[string]interface{}{
			"component": "SendersService",
			"function":  "Create",
			"error":     err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	customLogger.InfoWithData("Sender created successfully", map[string]interface{}{
		"component": "SendersService",
		"function":  "Create",
		"sender_id": mdl.ID,
		"type":      mdl.Type,
	})
	return o.toDTO(metaResp.ID, mdl), nil
}

func (o *SendersSvcImpl) Update(senderId string, req dtos.SenderUpdateReq, authInfo auth.AuthInfo) (*dtos.Sender, *gocom.CodedError) {
	customLogger.InfoWithData("Update sender started", map[string]interface{}{
		"component": "SendersService",
		"function":  "Update",
		"sender_id": senderId,
	})

	// Get existing sender
	mdl := sender.GetRepo().GetById(senderId)
	if mdl == nil {
		customLogger.WarnWithData("Sender not found", map[string]interface{}{
			"component": "SendersService",
			"function":  "Update",
			"sender_id": senderId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission - Provider can access all senders, others only their own
	if authInfo.ClientId != common.PROVIDER_ID && mdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":        "SendersService",
			"function":         "Update",
			"sender_client_id": mdl.ClientId,
			"auth_client_id":   authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	// Determine target clientId for duplicate checks
	targetClientId := mdl.ClientId
	if authInfo.ClientId == common.PROVIDER_ID && req.ClientId != "" {
		// Provider can update clientId
		targetClientId = req.ClientId
		mdl.ClientId = req.ClientId
	}

	// Check duplicate name if changed
	if req.Name != "" && req.Name != mdl.Name {
		existing := sender.GetRepo().GetByClientIdAndName(targetClientId, req.Name)
		if existing != nil && existing.ID != senderId {
			customLogger.WarnWithData("Sender name already exists", map[string]interface{}{
				"component": "SendersService",
				"function":  "Update",
				"name":      req.Name,
			})
			return nil, gocom.NewError(409, "Sender name already exists")
		}
		mdl.Name = req.Name
	}

	// Check duplicate identifier if changed
	if req.Identifier != "" && req.Identifier != mdl.Identifier {
		existing := sender.GetRepo().GetByClientIdTypeAndIdentifier(targetClientId, mdl.Type, req.Identifier)
		if existing != nil && existing.ID != senderId {
			customLogger.WarnWithData("Sender identifier already exists", map[string]interface{}{
				"component":  "SendersService",
				"function":   "Update",
				"identifier": req.Identifier,
			})
			return nil, gocom.NewError(409, "Sender identifier already exists for this type")
		}
		mdl.Identifier = req.Identifier
	}

	// Update config if provided
	if req.Config != nil {
		codedErr := o.validateConfig(mdl.Type, req.Config)
		if codedErr != nil {
			return nil, codedErr
		}
		mdl.Config = marshalSenderConfig(req.Config)
	}

	// Update fields
	if req.Status != "" {
		if req.Status != "active" && req.Status != "inactive" {
			customLogger.WarnWithData("Invalid status", map[string]interface{}{
				"component": "SendersService",
				"function":  "Update",
				"status":    req.Status,
			})
			return nil, gocom.NewError(400, "Status must be: active or inactive")
		}
		mdl.Status = req.Status
	}

	mdl.IsVerified = 0
	if req.IsVerified {
		mdl.IsVerified = 1
	}

	if req.DailyLimit > 0 {
		mdl.DailyLimit = req.DailyLimit
	}
	if req.MonthlyLimit > 0 {
		mdl.MonthlyLimit = req.MonthlyLimit
	}

	mdl.UpdatedBy = authInfo.UserId

	err := sender.GetRepo().Update(mdl)
	if err != nil {
		customLogger.ErrorWithData("Failed to update sender", map[string]interface{}{
			"component": "SendersService",
			"function":  "Update",
			"sender_id": senderId,
			"error":     err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	customLogger.InfoWithData("Sender updated successfully", map[string]interface{}{
		"component": "SendersService",
		"function":  "Update",
		"sender_id": mdl.ID,
	})
	return o.toDTO("", mdl), nil
}

func (o *SendersSvcImpl) GetById(senderId string, authInfo auth.AuthInfo) (*dtos.Sender, *gocom.CodedError) {
	mdl := sender.GetRepo().GetById(senderId)
	if mdl == nil {
		customLogger.WarnWithData("Sender not found", map[string]interface{}{
			"component": "SendersService",
			"function":  "GetById",
			"sender_id": senderId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission - Provider can access all senders, others only their own
	if authInfo.ClientId != common.PROVIDER_ID && mdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":        "SendersService",
			"function":         "GetById",
			"sender_client_id": mdl.ClientId,
			"auth_client_id":   authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	return o.toDTO("", mdl), nil
}

func (o *SendersSvcImpl) Search(filter, senderType, status, clientId string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.Sender, bool, int) {
	if authInfo.ClientId != common.PROVIDER_ID {
		clientId = authInfo.ClientId
	}

	mdls, haveNext, count := sender.GetRepo().Search(filter, clientId, senderType, status, pageNo, rowPerPage)

	ret := make([]*dtos.Sender, len(mdls))
	for i, mdl := range mdls {
		ret[i] = o.toDTO("", &mdl)
	}

	customLogger.DebugWithData("Search completed", map[string]interface{}{
		"component": "SendersService",
		"function":  "Search",
		"count":     len(ret),
		"have_next": haveNext,
	})
	return ret, haveNext, int(count)
}

func (o *SendersSvcImpl) Delete(senderId string, authInfo auth.AuthInfo) *gocom.CodedError {
	customLogger.InfoWithData("Delete sender started", map[string]interface{}{
		"component": "SendersService",
		"function":  "Delete",
		"sender_id": senderId,
	})

	// Get existing sender
	mdl := sender.GetRepo().GetById(senderId)
	if mdl == nil {
		customLogger.WarnWithData("Sender not found", map[string]interface{}{
			"component": "SendersService",
			"function":  "Delete",
			"sender_id": senderId,
		})
		return common.ERR_NOT_FOUND
	}

	// Check permission - Provider can access all senders, others only their own
	if authInfo.ClientId != common.PROVIDER_ID && mdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":        "SendersService",
			"function":         "Delete",
			"sender_client_id": mdl.ClientId,
			"auth_client_id":   authInfo.ClientId,
		})
		return common.ERR_NOT_ALLOWED
	}

	// TODO: Check if sender is being used in any campaigns
	// If yes, prevent deletion or mark as inactive

	err := sender.GetRepo().Delete(senderId)
	if err != nil {
		customLogger.ErrorWithData("Failed to delete sender", map[string]interface{}{
			"component": "SendersService",
			"function":  "Delete",
			"sender_id": senderId,
			"error":     err.Error(),
		})
		return common.ERR_UNABLE_TO_DELETE
	}

	customLogger.InfoWithData("Sender deleted successfully", map[string]interface{}{
		"component": "SendersService",
		"function":  "Delete",
		"sender_id": senderId,
	})
	return nil
}

func (o *SendersSvcImpl) VerifySender(phoneNumberId string, code string, authInfo auth.AuthInfo) (bool, *gocom.CodedError) {
	customLogger.InfoWithData("Verify sender started", map[string]interface{}{
		"component":       "SendersService",
		"function":        "VerifySender",
		"phone_number_id": phoneNumberId,
	})

	metaResp, err := o.metaOutboundService.VerifyOTPCode(context.Background(), dtos.VerifyOTPCodeReq{
		PhoneNumberId: phoneNumberId,
		Code:          code,
	})
	if err != nil {
		return false, gocom.NewError(400, "Invalid OTP code")
	}

	if !metaResp.SuccessfulVerification.Value.Success {
		return false, gocom.NewError(400, "Invalid OTP code")
	}

	return true, nil
}

func (o *SendersSvcImpl) validateConfig(senderType string, config map[string]interface{}) *gocom.CodedError {
	if config == nil {
		return nil
	}

	switch senderType {
	case "whatsapp":
		// Validate WhatsApp config
		// Expected: phone_number_id, access_token, waba_id
		if config["phone_number_id"] == nil || config["phone_number_id"] == "" {
			return gocom.NewError(400, "WhatsApp config must have phone_number_id")
		}
		if config["access_token"] == nil || config["access_token"] == "" {
			return gocom.NewError(400, "WhatsApp config must have access_token")
		}
		if config["waba_id"] == nil || config["waba_id"] == "" {
			return gocom.NewError(400, "WhatsApp config must have waba_id")
		}

	case "sms":
		// Validate SMS config
		// Expected: provider (twilio, nexmo, etc), api_key, api_secret, etc
		if config["provider"] == nil || config["provider"] == "" {
			return gocom.NewError(400, "SMS config must have provider")
		}

	case "email":
		// Validate Email config
		// Expected: smtp_host, smtp_port, username, password, from_email
		if config["smtp_host"] == nil || config["smtp_host"] == "" {
			return gocom.NewError(400, "Email config must have smtp_host")
		}
		if config["from_email"] == nil || config["from_email"] == "" {
			return gocom.NewError(400, "Email config must have from_email")
		}
	}

	return nil
}

func (o *SendersSvcImpl) toDTO(phoneNumberId string, mdl *sender.Sender) *dtos.Sender {
	if mdl == nil {
		return nil
	}

	return &dtos.Sender{
		ID:            mdl.ID,
		ClientId:      mdl.ClientId,
		Type:          mdl.Type,
		Name:          mdl.Name,
		Identifier:    mdl.Identifier,
		Config:        unmarshalSenderConfig(mdl.Config),
		Status:        mdl.Status,
		IsVerified:    mdl.IsVerified != 0,
		DailyLimit:    mdl.DailyLimit,
		MonthlyLimit:  mdl.MonthlyLimit,
		CreatedBy:     mdl.CreatedBy,
		UpdatedBy:     mdl.UpdatedBy,
		PhoneNumberId: phoneNumberId,
	}
}
