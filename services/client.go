package services

import (
	"context"
	"encoding/json"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	"github.com/ariandi/gocom/pubsub"
	"github.com/jinzhu/copier"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/billingTransaction"
	"gitlab.com/anti_metter/switching_common/client"
	"gitlab.com/anti_metter/switching_common/role"
	"gitlab.com/anti_metter/switching_common/sender"
	"gitlab.com/anti_metter/switching_common/user"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/outbound"
	"gitlab.com/bot3342545/il-dashboard/utils"

	"strings"
	"sync"
)

type ClientSvc interface {
	Create(req dtos.ClientReq, authInfo auth.AuthInfo) (*dtos.Client, *gocom.CodedError)
	Get(id string, authInfo auth.AuthInfo) (*dtos.Client, *gocom.CodedError)
	Update(id string, req dtos.ClientUpdateReq, authInfo auth.AuthInfo) (*dtos.Client, *gocom.CodedError)
	Delete(id string, authInfo auth.AuthInfo) *gocom.CodedError
	Search(filter, parentId, clientId string, pageNo, rowPerPage int) ([]dtos.Client, bool, int64)
	OnboardMeta(req dtos.ClientMetaOnboardReq) (*dtos.ClientMetaOnboardResp, *gocom.CodedError)
}

type ClientSvcImpl struct {
	MetaOutbound outbound.MetaOutboundService
}

func (o *ClientSvcImpl) Create(req dtos.ClientReq, authInfo auth.AuthInfo) (*dtos.Client, *gocom.CodedError) {

	req.Name = strings.Trim(req.Name, " ")
	req.OwnerName = strings.Trim(req.OwnerName, " ")
	req.OwnerPassword = strings.Trim(req.OwnerPassword, " ")

	existing := client.GetRepo().GetByName(strings.Trim(req.Name, " "))
	if existing != nil {
		return nil, common.ERR_ALREADY_EXIST
	}

	// check owner
	existingOwner := user.GetRepo().GetByEmail(req.OwnerEmail)
	if existingOwner != nil {
		return nil, common.ERR_ALREADY_EXIST
	}

	if req.ParentId == "" {
		req.ParentId = authInfo.ClientId
	}

	if authInfo.ClientId != common.PROVIDER_ID {
		req.ParentId = authInfo.ClientId
		req.Balance = 0
		req.TotalUsage = 0
		req.BillingType = "prepaid"
		req.CreditLimit = 0
	}

	parent := client.GetSvc().GetById(req.ParentId)
	if parent == nil {
		return nil, common.ERR_NOT_FOUND
	}

	mdl := client.Client{}
	_ = copier.Copy(&mdl, req)

	mdl.Status = common.STATUS_ACTIVE
	mdl.OwnerId = req.OwnerEmail

	err := client.GetRepo().Create(&mdl).Error
	if err != nil {
		logger.Debugf("[ClientService Create] unable to create client %+v \n", err)
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	if authInfo.ClientId != common.PROVIDER_ID {
		authInfo.ClientId = mdl.ID
	}

	logger.Infof("[ClientService Create] create client %s \n", authInfo)

	err = billingTransaction.GetRepo().Create(&billingTransaction.BillingTransaction{
		ClientId:      mdl.ID,
		Type:          constans.BillingTransactionTypeTopup,
		Amount:        mdl.Balance,
		BalanceBefore: 0,
		BalanceAfter:  mdl.Balance,
		Description:   "Initial balance",
	}).Error
	if err != nil {
		logger.Debugf("[ClientService Create] unable to create initial balance %+v \n", err)
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	owner, errCreateUser := GetUserService().Create(dtos.UserReq{
		Name:     req.OwnerName,
		Email:    req.OwnerEmail,
		Password: req.OwnerPassword,
		ClientId: mdl.ID,
		Roles:    []string{auth.OWNER},
	}, authInfo)

	if errCreateUser != nil {
		logger.Debugf("[ClientService Create] unable to create user %+v \n", errCreateUser)
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	if req.Sender.Name == "" || req.Sender.Identifier == "" {
		return nil, common.ERR_INVALID_REQUEST
	}

	if req.Sender.Type == "" {
		req.Sender.Type = "whatsapp"
	}

	existingSender := sender.GetRepo().GetByClientIdTypeAndIdentifier(mdl.ID, req.Sender.Type, req.Sender.Identifier)
	if existingSender != nil {
		return nil, common.ERR_ALREADY_EXIST
	}

	_, senderError := sendersService.Create(dtos.SenderReq{
		ClientId:   mdl.ID,
		Name:       req.Sender.Name,
		Identifier: req.Sender.Identifier,
		Type:       req.Sender.Type,
	}, authInfo)

	if senderError != nil {
		logger.Debugf("[ClientService Create] unable to create sender %+v \n", senderError)
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	ret := &dtos.Client{}
	_ = copier.Copy(ret, mdl)

	roles := []dtos.Role{}
	if owner != nil {
		ret.OwnerName = owner.Name
		ret.OwnerEmail = owner.Email
		userRoles := user.GetRepo().ListRole(owner.ID)
		for _, ur := range userRoles {
			urByID := role.GetRepo().GetById(ur.ID)
			urTemp := dtos.Role{}
			_ = copier.Copy(&urTemp, urByID)
			roles = append(roles, urTemp)
		}
	}

	parent = client.GetSvc().GetById(mdl.ParentId)
	if parent != nil {
		ret.ParentName = parent.Name
		ret.ParentId = parent.ID
	}
	ret.Children = o.Children(mdl.ID)
	ret.Roles = roles

	_ = pubsub.Get().Publish("update_client", ret.ID)

	return ret, nil
}

func (o *ClientSvcImpl) Get(id string, authInfo auth.AuthInfo) (*dtos.Client, *gocom.CodedError) {
	logger.Infof("[ClientService Get] start \n")
	mdl := client.GetRepo().GetById(id)
	if mdl == nil {
		return nil, common.ERR_NOT_FOUND
	}

	ret := &dtos.Client{}
	_ = copier.Copy(ret, mdl)

	roles := []dtos.Role{}
	owner := user.GetSvc().GetByEmail(mdl.OwnerId)
	if owner != nil {
		ret.OwnerEmail = mdl.OwnerId
		ret.OwnerName = owner.Name
		userRoles := user.GetRepo().ListRole(owner.ID)
		for _, ur := range userRoles {
			urByID := role.GetRepo().GetById(ur.ID)
			urTemp := dtos.Role{}
			_ = copier.Copy(&urTemp, urByID)
			roles = append(roles, urTemp)
		}
	}

	parent := client.GetSvc().GetById(mdl.ParentId)
	if parent != nil {
		ret.ParentName = parent.Name
		ret.ParentId = parent.ID
	}
	ret.Children = o.Children(mdl.ID)
	ret.Roles = roles

	return ret, nil
}

func (o *ClientSvcImpl) Update(id string, req dtos.ClientUpdateReq, authInfo auth.AuthInfo) (*dtos.Client, *gocom.CodedError) {

	customLogger.InfoWithData("Update client started", map[string]interface{}{
		"component": "ClientService",
		"function":  "Update",
		"id":        id,
		"req":       req,
	})

	mdl := client.GetRepo().GetById(id)
	if mdl == nil {
		return nil, common.ERR_NOT_FOUND
	}

	if authInfo.ClientId != common.PROVIDER_ID {
		if !client.GetSvc().IsAcessible(mdl.ID, authInfo.ClientId) {
			return nil, common.ERR_NOT_FOUND
		}
		//return nil, common.ERR_NOT_FOUND
	}

	if mdl.Name != req.Name {
		existing := client.GetRepo().GetByName(req.Name)
		if existing != nil {
			return nil, common.ERR_ALREADY_EXIST
		}
	}

	mdl.Name = editVal(req.Name, mdl.Name)
	//mdl.OwnerId = editVal(req.OwnerEmail, mdl.OwnerId)

	mdl.Status = editVal(req.Status, mdl.Status)
	mdl.Greeting = editVal(req.Greeting, mdl.Greeting)
	mdl.IsAi = req.IsAi
	mdl.GreetingTrigger = editVal(req.GreetingTrigger, mdl.GreetingTrigger)
	mdl.WabaId = editVal(req.WabaId, mdl.WabaId)
	mdl.WaAuthCode = editVal(req.MetaCode, mdl.WaAuthCode)

	var waMetaData dtos.WhatsappEmbeddedSignupData
	err := json.Unmarshal([]byte(req.WaMetaData), &waMetaData)
	if err != nil {
		return nil, common.ERR_INVALID_REQUEST
	}

	customLogger.InfoWithData("WaMetaData", map[string]interface{}{
		"component":  "ClientService",
		"function":   "Update",
		"waMetaData": waMetaData,
	})

	mdl.PhoneNumberId = waMetaData.Data.PhoneNumberId
	mdl.BusinessId = waMetaData.Data.BusinessId
	// Validate required fields
	if mdl.PhoneNumberId == "" {
		customLogger.ErrorWithData("Missing phone_number_id in WaMetaData", map[string]interface{}{
			"component":  "ClientService",
			"function":   "Update",
			"waMetaData": waMetaData,
		})
		return nil, common.ERR_INVALID_REQUEST
	}

	if mdl.BusinessId == "" {
		customLogger.ErrorWithData("Missing business_id in WaMetaData", map[string]interface{}{
			"component":  "ClientService",
			"function":   "Update",
			"waMetaData": waMetaData,
		})
		return nil, common.ERR_INVALID_REQUEST
	}

	if authInfo.ClientId == common.PROVIDER_ID {
		if req.CreditLimit > 0 {
			mdl.CreditLimit = req.CreditLimit
		}

		if req.Balance > 0 {
			mdl.Balance = req.Balance
		}

		if req.TotalUsage > 0 {
			mdl.TotalUsage = req.TotalUsage
		}

		mdl.BillingType = editVal(req.BillingType, mdl.BillingType)
	}

	err = client.GetRepo().Update(mdl)
	if err != nil {
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	senders := sender.GetRepo().GetByClientId(mdl.ID)
	for _, senderMdl := range senders {
		senderMdl.PhoneNumberId = mdl.PhoneNumberId
		err = sender.GetRepo().Update(&senderMdl)
		if err != nil {
			customLogger.ErrorWithData("Failed to update sender", map[string]interface{}{
				"component":       "client",
				"function":        "Update",
				"sender_id":       senderMdl.ID,
				"phone_number_id": senderMdl.PhoneNumberId,
				"error":           err.Error(),
			})
			return nil, common.ERR_UNABLE_TO_UPDATE
		}
	}

	ret := &dtos.Client{}
	_ = copier.Copy(ret, mdl)

	roles := []dtos.Role{}
	owner := user.GetSvc().GetByEmail(mdl.OwnerId)
	if owner != nil {
		ret.OwnerName = owner.Name
		ret.OwnerEmail = owner.Email
		userRoles := user.GetRepo().ListRole(owner.ID)
		for _, ur := range userRoles {
			urByID := role.GetRepo().GetById(ur.ID)
			urTemp := dtos.Role{}
			_ = copier.Copy(&urTemp, urByID)
			roles = append(roles, urTemp)
		}
	}

	parent := client.GetSvc().GetById(mdl.ParentId)
	if parent != nil {
		ret.ParentName = parent.Name
		ret.ParentId = parent.ID
	}
	ret.Children = o.Children(mdl.ID)
	ret.Roles = roles

	err = billingTransaction.GetRepo().Create(&billingTransaction.BillingTransaction{
		ClientId:      mdl.ID,
		Type:          constans.BillingTransactionTypeTopup,
		Amount:        mdl.Balance,
		BalanceBefore: 0,
		BalanceAfter:  mdl.Balance,
		Description:   "Initial balance",
	}).Error
	if err != nil {
		logger.Debugf("[ClientService Update] unable to create initial balance %+v \n", err)
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	_ = pubsub.Get().Publish("update_client", ret.ID)
	return ret, nil
}

func (o *ClientSvcImpl) Delete(id string, authInfo auth.AuthInfo) *gocom.CodedError {

	mdl := client.GetRepo().GetById(id)

	if mdl == nil {
		return common.ERR_NOT_FOUND
	}

	if authInfo.ClientId != common.PROVIDER_ID && mdl.ParentId != authInfo.ClientId {
		return common.ERR_NOT_FOUND
	}

	o.DeleteSubChildren(mdl.ID)

	client.GetRepo().Delete(mdl.OwnerId)
	err := client.GetRepo().Delete(mdl).Error

	if err != nil {
		return common.ERR_UNABLE_TO_DELETE
	}

	_ = pubsub.Get().Publish("update_client", id)
	return nil
}

func (o *ClientSvcImpl) Search(filter, parentId, clientId string, pageNo, rowPerPage int) ([]dtos.Client, bool, int64) {
	customLogger.InfoWithData("search client start", map[string]interface{}{
		"component": "client",
		"function":  "Search",
		"filter":    filter,
		"parentId":  parentId,
		"clientId":  clientId,
	})

	ret := []dtos.Client{}
	list, haveNext, count := client.GetRepo().Search(filter, parentId, clientId, pageNo, rowPerPage)

	for _, item := range list {
		cl := dtos.Client{}
		_ = copier.Copy(&cl, item)

		roles := []dtos.Role{}
		owner := user.GetSvc().GetByEmail(item.OwnerId)
		if owner != nil {
			cl.OwnerEmail = item.OwnerId
			cl.OwnerName = item.Name
			userRoles := user.GetRepo().ListRole(owner.ID)
			for _, ur := range userRoles {
				urByID := role.GetRepo().GetById(ur.ID)
				urTemp := dtos.Role{}
				_ = copier.Copy(&urTemp, urByID)
				roles = append(roles, urTemp)
			}
		}

		parent := client.GetSvc().GetById(item.ParentId)
		if parent != nil {
			cl.ParentName = parent.Name
			cl.ParentId = parent.ID
		}
		cl.Children = o.Children(item.ID)
		cl.Roles = roles
		ret = append(ret, cl)
	}

	return ret, haveNext, count
}

func (o *ClientSvcImpl) Children(parentId string) []dtos.Client {

	ret := []dtos.Client{}
	list := client.GetRepo().List(parentId)

	for _, mdl := range list {
		item := dtos.Client{}
		_ = copier.Copy(&item, mdl)

		owner := user.GetSvc().GetByEmail(mdl.OwnerId)
		if owner != nil {
			item.OwnerEmail = owner.Email
		}

		parent := client.GetSvc().GetById(mdl.ParentId)

		if parent != nil {
			item.ParentName = parent.Name
			item.ParentId = parent.ID
		}

		item.Children = o.Children(mdl.ID)

		ret = append(ret, item)
	}

	return ret
}

func (o *ClientSvcImpl) DeleteSubChildren(id string) {

	list := client.GetRepo().List(id)

	for _, mdl := range list {

		o.DeleteSubChildren(mdl.ID)
		client.GetRepo().Delete(mdl.ID)
		user.GetRepo().Delete(mdl.OwnerId)
	}
}

func (o *ClientSvcImpl) OnboardMeta(req dtos.ClientMetaOnboardReq) (*dtos.ClientMetaOnboardResp, *gocom.CodedError) {
	customLogger.InfoWithData("OnboardMeta started", map[string]interface{}{
		"component": "client",
		"function":  "OnboardMeta",
		"req":       req,
	})

	mdl := client.GetRepo().GetById(req.ClientId)
	if mdl == nil {
		return nil, common.ERR_NOT_FOUND
	}

	if mdl.WaAuthCode == "" && mdl.WabaId == "" {
		return nil, common.ERR_NOT_FOUND
	}

	resp, err := o.MetaOutbound.ExchangeToken(context.Background(), dtos.ExchangeTokenReq{
		Code: mdl.WaAuthCode,
	})
	if err != nil {
		customLogger.ErrorWithData("Failed to exchange token", map[string]interface{}{
			"component": "client",
			"function":  "OnboardMeta",
			"error":     err.Error(),
		})
		return nil, common.ERR_INTERNAL_SERVER_ERROR
	}

	subscribeResp, err := o.MetaOutbound.SubscribeWebhook(context.Background(), dtos.SubscribeWebhookReq{
		BusinessToken: resp.AccessToken,
		WabaId:        mdl.WabaId,
	})
	if err != nil {
		customLogger.ErrorWithData("Failed to subscribe webhook", map[string]interface{}{
			"component": "client",
			"function":  "OnboardMeta",
			"error":     err.Error(),
		})
		return nil, common.ERR_INTERNAL_SERVER_ERROR
	}

	mdl.BusinessToken = resp.AccessToken
	mdl.Pin = utils.GenerateRandomPin()

	err = client.GetRepo().Update(mdl)
	if err != nil {
		customLogger.ErrorWithData("Failed to update client", map[string]interface{}{
			"component": "client",
			"function":  "OnboardMeta",
			"error":     err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	senders := sender.GetRepo().GetByClientIdAndPhoneNumberId(req.ClientId, mdl.PhoneNumberId)
	if senders != nil {
		_, err = o.MetaOutbound.RegisterPhoneNumberOnboard(context.Background(), dtos.RegisterPhoneNumberOnboardReq{
			MessagingProduct: "whatsapp",
			PhoneNumberID:    senders.PhoneNumberId,
			Pin:              mdl.Pin,
		})
		if err != nil {
			customLogger.ErrorWithData("Failed to register phone number onboard", map[string]interface{}{
				"component":       "client",
				"function":        "OnboardMeta",
				"sender_id":       senders.ID,
				"phone_number_id": senders.PhoneNumberId,
				"error":           err.Error(),
			})
			return nil, common.ERR_INTERNAL_SERVER_ERROR
		}
	}

	return &dtos.ClientMetaOnboardResp{
		Success: subscribeResp.Success,
	}, nil
}

//----------------------------------------

var clientSvc ClientSvc
var clientSvcOnce sync.Once

func SetClientSvc(svc ClientSvc) {
	clientSvc = svc
}

func GetClientSvc() ClientSvc {

	if clientSvc == nil {
		clientSvcOnce.Do(func() {
			tmp := &ClientSvcImpl{
				MetaOutbound: outbound.NewMetaOutboundService(),
			}
			clientSvc = tmp
			role.GetRepo()
		})
	}

	return clientSvc
}
