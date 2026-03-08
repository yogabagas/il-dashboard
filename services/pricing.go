package services

import (
	"sync"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	"github.com/jinzhu/copier"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/repositories/pricing"
)

type PricingService interface {
	Create(req dtos.PricingReq, authInfo auth.AuthInfo) (*dtos.Pricing, *gocom.CodedError)
	Update(pricingId string, req dtos.PricingReq, authInfo auth.AuthInfo) (*dtos.Pricing, *gocom.CodedError)
	GetById(pricingId string, authInfo auth.AuthInfo) (*dtos.Pricing, *gocom.CodedError)
	Search(filter, serviceType, category, clientId, status string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.Pricing, bool, int)
	Delete(pricingId string, authInfo auth.AuthInfo) *gocom.CodedError
}

type PricingSvcImpl struct{}

var pricingService PricingService
var oncePricingService sync.Once

func GetPricingService() PricingService {
	oncePricingService.Do(func() {
		pricingService = &PricingSvcImpl{}
	})
	return pricingService
}

func (o *PricingSvcImpl) Create(req dtos.PricingReq, authInfo auth.AuthInfo) (*dtos.Pricing, *gocom.CodedError) {
	logger.Infof("PricingService.Create - Start req=%+v", req)

	pricingMdl := pricing.GetRepo().GetByServiceAndCategory(req.ServiceType, req.Category, req.ClientId)
	if pricingMdl != nil {
		logger.Warnf("PricingService.Create - Pricing with the same service type already exists serviceType=%s clientId=%v", req.ServiceType, req.ClientId)
		return nil, gocom.NewError(400, "pricing with the same service type already exists")
	}

	// Validation
	if req.ServiceType == "" {
		logger.Warnf("PricingService.Create - ServiceType is required req=%+v", req)
		return nil, gocom.NewError(400, "service_type is required")
	}

	if req.Cost < 0 {
		logger.Warnf("PricingService.Create - Cost must be non-negative req=%+v", req)
		return nil, gocom.NewError(400, "cost must be non-negative")
	}

	// Access control: Non-PROVIDER users can only create pricing for their own clientId
	if authInfo.ClientId != common.PROVIDER_ID {
		req.ClientId = &authInfo.ClientId
		logger.Infof("PricingService.Create - Non-provider user, forcing clientId=%s", authInfo.ClientId)
	}

	// Create model
	mdl := &pricing.Pricing{
		ServiceType: req.ServiceType,
		Category:    req.Category,
		Cost:        req.Cost,
		Currency:    req.Currency,
		IsActive:    req.IsActive,
		ClientId:    req.ClientId,
	}

	// Set defaults
	if mdl.Currency == "" {
		mdl.Currency = "IDR"
	}

	err := pricing.GetRepo().Create(mdl)
	if err != nil {
		logger.Errorf("PricingService.Create - Failed to create pricing err=%v", err)
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	logger.Infof("PricingService.Create - mdl %+v Success", mdl)
	return o.toDTO(mdl), nil
}

func (o *PricingSvcImpl) Update(pricingId string, req dtos.PricingReq, authInfo auth.AuthInfo) (*dtos.Pricing, *gocom.CodedError) {
	logger.Infof("PricingService.Update - Start pricingId=%s req=%+v", pricingId, req)

	// Get existing pricing
	mdl := pricing.GetRepo().GetById(pricingId)
	if mdl == nil {
		logger.Warnf("PricingService.Update - Pricing not found pricingId=%s", pricingId)
		return nil, common.ERR_NOT_FOUND
	}

	// Access control: Non-PROVIDER users can only update their own pricing
	if authInfo.ClientId != common.PROVIDER_ID {
		// Check ownership
		if mdl.ClientId == nil || *mdl.ClientId != authInfo.ClientId {
			logger.Warnf("PricingService.Update - Unauthorized access pricingClientId=%v authClientId=%s", mdl.ClientId, authInfo.ClientId)
			return nil, common.ERR_NOT_ALLOWED
		}
		// Force clientId to remain the same
		req.ClientId = &authInfo.ClientId
	}

	// Update fields
	if req.ServiceType != "" {
		mdl.ServiceType = req.ServiceType
	}

	mdl.Category = req.Category
	mdl.Cost = req.Cost

	if req.Currency != "" {
		mdl.Currency = req.Currency
	}

	mdl.IsActive = req.IsActive

	if req.ClientId != nil {
		mdl.ClientId = req.ClientId
	}

	err := pricing.GetRepo().Update(mdl)
	if err != nil {
		logger.Errorf("PricingService.Update - Failed to update pricing pricingId=%s err=%v", pricingId, err)
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	logger.Infof("PricingService.Update - mdl %+v Success", mdl)
	return o.toDTO(mdl), nil
}

func (o *PricingSvcImpl) GetById(pricingId string, authInfo auth.AuthInfo) (*dtos.Pricing, *gocom.CodedError) {
	logger.Infof("PricingService.GetById - Start pricingId=%s", pricingId)

	mdl := pricing.GetRepo().GetById(pricingId)
	if mdl == nil {
		logger.Warnf("PricingService.GetById - Pricing not found pricingId=%s", pricingId)
		return nil, common.ERR_NOT_FOUND
	}

	// Access control: Non-PROVIDER users can only view their own pricing
	if authInfo.ClientId != common.PROVIDER_ID {
		if mdl.ClientId == nil || *mdl.ClientId != authInfo.ClientId {
			logger.Warnf("PricingService.GetById - Unauthorized access pricingClientId=%v authClientId=%s", mdl.ClientId, authInfo.ClientId)
			return nil, common.ERR_NOT_ALLOWED
		}
	}

	logger.Infof("PricingService.GetById - mdl %+v Success", mdl)
	return o.toDTO(mdl), nil
}

func (o *PricingSvcImpl) Search(filter, serviceType, category, clientId, status string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.Pricing, bool, int) {
	logger.Infof("PricingService.Search - Start filter=%s serviceType=%s category=%s status=%s pageNo=%d rowPerPage=%d", filter, serviceType, category, status, pageNo, rowPerPage)

	if authInfo.ClientId != common.PROVIDER_ID {
		clientId = authInfo.ClientId
	}

	// Parse status to a bool pointer
	var isActive *bool
	if status != "" {
		if status == "active" {
			active := true
			isActive = &active
		} else if status == "inactive" {
			inactive := false
			isActive = &inactive
		}
	}

	mdls, haveNext, count := pricing.GetRepo().Search(filter, serviceType, category, clientId, isActive, pageNo, rowPerPage)

	ret := make([]*dtos.Pricing, len(mdls))
	for i, mdl := range mdls {
		ret[i] = o.toDTO(&mdl)
	}

	logger.Infof("PricingService.Search - len=%d haveNext=%v count=%d Success", len(ret), haveNext, count)
	return ret, haveNext, int(count)
}

func (o *PricingSvcImpl) Delete(pricingId string, authInfo auth.AuthInfo) *gocom.CodedError {
	logger.Infof("PricingService.Delete - Start pricingId=%s", pricingId)

	// Get existing pricing
	mdl := pricing.GetRepo().GetById(pricingId)
	if mdl == nil {
		logger.Warnf("PricingService.Delete - Pricing not found pricingId=%s", pricingId)
		return common.ERR_NOT_FOUND
	}

	// Access control: Non-PROVIDER users can only delete their own pricing
	if authInfo.ClientId != common.PROVIDER_ID {
		if mdl.ClientId == nil || *mdl.ClientId != authInfo.ClientId {
			logger.Warnf("PricingService.Delete - Unauthorized access pricingClientId=%v authClientId=%s", mdl.ClientId, authInfo.ClientId)
			return common.ERR_NOT_ALLOWED
		}
	}

	err := pricing.GetRepo().Delete(pricingId)
	if err != nil {
		logger.Errorf("PricingService.Delete - Failed to delete pricing pricingId=%s err=%v", pricingId, err)
		return common.ERR_UNABLE_TO_DELETE
	}

	logger.Infof("PricingService.Delete - pricingId %s Success", pricingId)
	return nil
}

func (o *PricingSvcImpl) toDTO(mdl *pricing.Pricing) *dtos.Pricing {
	if mdl == nil {
		return nil
	}

	dto := &dtos.Pricing{}
	_ = copier.Copy(&dto, &mdl)
	return dto
}
