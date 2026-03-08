package services

import (
	"strings"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/config"
	"github.com/jinzhu/copier"
	"github.com/oklog/ulid/v2"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/billingTransaction"
	"gitlab.com/anti_metter/switching_common/client"
	"gitlab.com/anti_metter/switching_common/messageLog"
	"gitlab.com/anti_metter/switching_common/waTemplate"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/repositories/pricing"
)

type BillingService interface {
	GetClientBillingInfo(authInfo auth.AuthInfo) (*dtos.ClientBillingInfo, *gocom.CodedError)
	CanClientSend(clientId string, estimatedCost float64) (bool, *gocom.CodedError)
	CreateTransaction(req dtos.BillingTransactionReq, authInfo auth.AuthInfo) (*dtos.BillingTransaction, *gocom.CodedError)
	GetTransactionById(transactionId string, authInfo auth.AuthInfo) (*dtos.BillingTransaction, *gocom.CodedError)
	SearchTransactions(transactionType, dateFrom, dateTo string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.BillingTransaction, bool, int)
	DeductBalance(clientId string, amount float64, referenceId, referenceType, description, userId string) (*dtos.BillingTransaction, *gocom.CodedError)
	TopupBalance(clientId string, amount float64, description, userId string) (*dtos.BillingTransaction, *gocom.CodedError)
	GetUsageSummary(period string, authInfo auth.AuthInfo) (*dtos.UsageSummary, *gocom.CodedError)
	CalculateMessageCost(messageType, clientId string) float64
	CalculateMessageCostWithCategory(messageType, category, clientId string) float64
}

type BillingSvcImpl struct{}

var billingService BillingService
var onceBillingService sync.Once

func GetBillingService() BillingService {
	onceBillingService.Do(func() {
		billingService = &BillingSvcImpl{}
	})
	return billingService
}

func (o *BillingSvcImpl) GetClientBillingInfo(authInfo auth.AuthInfo) (*dtos.ClientBillingInfo, *gocom.CodedError) {
	clientMdl := client.GetRepo().GetById(authInfo.ClientId)
	if clientMdl == nil {
		customLogger.WarnWithData("Client not found", map[string]interface{}{
			"component": "BillingService",
			"function":  "GetClientBillingInfo",
			"client_id": authInfo.ClientId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	canSend, _ := o.CanClientSend(authInfo.ClientId, 0)

	info := &dtos.ClientBillingInfo{
		ClientId:    clientMdl.ID,
		ClientName:  clientMdl.Name,
		BillingType: clientMdl.BillingType,
		Balance:     clientMdl.Balance,
		TotalUsage:  clientMdl.TotalUsage,
		CanSend:     canSend,
	}

	if clientMdl.BillingType == "postpaid" {
		info.CreditLimit = clientMdl.CreditLimit
		info.RemainingBalance = clientMdl.CreditLimit + clientMdl.Balance // Balance is negative for postpaid
		info.Status = "active"
		if info.RemainingBalance < 0 {
			info.Status = "overlimit"
		}
	} else {
		// Prepaid
		info.RemainingBalance = clientMdl.Balance
		info.Status = "active"
		if info.RemainingBalance <= 0 {
			info.Status = "insufficient"
		}
	}

	customLogger.DebugWithData("Get billing info completed", map[string]interface{}{
		"component": "BillingService",
		"function":  "GetClientBillingInfo",
		"can_send":  canSend,
		"balance":   info.Balance,
	})
	return info, nil
}

func (o *BillingSvcImpl) CanClientSend(clientId string, estimatedCost float64) (bool, *gocom.CodedError) {
	clientMdl := client.GetRepo().GetById(clientId)
	if clientMdl == nil {
		customLogger.WarnWithData("Client not found", map[string]interface{}{
			"component": "BillingService",
			"function":  "CanClientSend",
			"client_id": clientId,
		})
		return false, common.ERR_NOT_FOUND
	}

	canSend := false

	if clientMdl.BillingType == "postpaid" {
		// For postpaid: balance is negative (usage), credit_limit is positive
		// Can send if: |balance| + estimatedCost <= credit_limit
		// Which is: -balance + estimatedCost <= credit_limit
		// Or: balance >= -credit_limit + estimatedCost
		remainingCredit := clientMdl.CreditLimit + clientMdl.Balance
		canSend = remainingCredit >= estimatedCost
	} else {
		// Prepaid: balance is positive
		canSend = clientMdl.Balance >= estimatedCost
	}

	customLogger.DebugWithData("CanClientSend check completed", map[string]interface{}{
		"component":      "BillingService",
		"function":       "CanClientSend",
		"client_id":      clientId,
		"can_send":       canSend,
		"balance":        clientMdl.Balance,
		"estimated_cost": estimatedCost,
	})
	return canSend, nil
}

func (o *BillingSvcImpl) CreateTransaction(req dtos.BillingTransactionReq, authInfo auth.AuthInfo) (*dtos.BillingTransaction, *gocom.CodedError) {
	customLogger.InfoWithData("CreateTransaction started", map[string]interface{}{
		"component": "BillingService",
		"function":  "CreateTransaction",
		"type":      req.Type,
		"amount":    req.Amount,
	})

	// Validation
	if req.Type == "" {
		customLogger.WarnWithData("Type is required", map[string]interface{}{
			"component": "BillingService",
			"function":  "CreateTransaction",
		})
		return nil, common.ERR_INVALID_REQUEST
	}

	validTypes := map[string]bool{"topup": true, "usage": true, "payment": true, "refund": true, "adjustment": true}
	if !validTypes[req.Type] {
		customLogger.WarnWithData("Invalid type", map[string]interface{}{
			"component": "BillingService",
			"function":  "CreateTransaction",
			"type":      req.Type,
		})
		return nil, gocom.NewError(400, "Invalid transaction type")
	}

	if req.Amount <= 0 {
		customLogger.WarnWithData("Amount must be positive", map[string]interface{}{
			"component": "BillingService",
			"function":  "CreateTransaction",
			"amount":    req.Amount,
		})
		return nil, gocom.NewError(400, "Amount must be positive")
	}

	if common.PROVIDER_ID != authInfo.ClientId {
		req.ClientId = authInfo.ClientId
	}

	if req.ClientId == "" {
		req.ClientId = authInfo.ClientId
	}

	clientMdl := client.GetRepo().GetById(authInfo.ClientId)
	if clientMdl == nil {
		customLogger.WarnWithData("Client not found", map[string]interface{}{
			"component": "BillingService",
			"function":  "CreateTransaction",
			"client_id": authInfo.ClientId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	balanceBefore := clientMdl.Balance
	balanceAfter := balanceBefore

	// Calculate new balance based on transaction type
	switch req.Type {
	case constans.BillingTransactionTypeTopup, constans.BillingTransactionTypeRefund:
		balanceAfter = balanceBefore + req.Amount
		clientMdl.TotalUsage -= req.Amount
	case constans.BillingTransactionTypeUsage, constans.BillingTransactionTypePayment:
		balanceAfter = balanceBefore - req.Amount
		clientMdl.TotalUsage += req.Amount
	case constans.BillingTransactionTypeAdjustment:
		// For adjustment, amount can be positive (add) or negative (subtract) based on description
		// But since we validate amount > 0, we'll treat adjustment as additive
		// If client wants to subtract, they should use negative in description and we adjust here
		balanceAfter = balanceBefore + req.Amount
	}

	// Create transaction
	txn := &billingTransaction.BillingTransaction{
		ClientId:      authInfo.ClientId,
		Type:          req.Type,
		Amount:        req.Amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		ReferenceId:   req.ReferenceId,
		ReferenceType: req.ReferenceType,
		Description:   req.Description,
		CreatedBy:     authInfo.UserId,
	}

	err := billingTransaction.GetRepo().Create(txn).Error
	if err != nil {
		customLogger.ErrorWithData("Failed to create transaction", map[string]interface{}{
			"component": "BillingService",
			"function":  "CreateTransaction",
			"type":      req.Type,
			"error":     err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	// Update client balance
	clientMdl.Balance = balanceAfter

	errUpdate := client.GetRepo().Update(clientMdl)
	if errUpdate != nil {
		customLogger.ErrorWithData("Failed to update client balance", map[string]interface{}{
			"component": "BillingService",
			"function":  "CreateTransaction",
			"client_id": authInfo.ClientId,
			"error":     errUpdate.Error(),
		})
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	customLogger.InfoWithData("Transaction created successfully", map[string]interface{}{
		"component":      "BillingService",
		"function":       "CreateTransaction",
		"transaction_id": txn.ID,
		"type":           txn.Type,
		"amount":         txn.Amount,
		"balance_after":  txn.BalanceAfter,
	})
	ret := dtos.BillingTransaction{}
	_ = copier.Copy(&ret, &txn)
	return &ret, nil
}

func (o *BillingSvcImpl) GetTransactionById(transactionId string, authInfo auth.AuthInfo) (*dtos.BillingTransaction, *gocom.CodedError) {
	mdl := billingTransaction.GetRepo().GetById(transactionId)
	if mdl == nil {
		customLogger.WarnWithData("Transaction not found", map[string]interface{}{
			"component":      "BillingService",
			"function":       "GetTransactionById",
			"transaction_id": transactionId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	// Check permission
	if mdl.ClientId != authInfo.ClientId {
		customLogger.WarnWithData("Unauthorized", map[string]interface{}{
			"component":      "BillingService",
			"function":       "GetTransactionById",
			"txn_client_id":  mdl.ClientId,
			"auth_client_id": authInfo.ClientId,
		})
		return nil, common.ERR_NOT_ALLOWED
	}

	ret := &dtos.BillingTransaction{}
	_ = copier.Copy(&ret, &mdl)

	customLogger.DebugWithData("Transaction retrieved", map[string]interface{}{
		"component":      "BillingService",
		"function":       "GetTransactionById",
		"transaction_id": mdl.ID,
	})
	return ret, nil
}

func (o *BillingSvcImpl) SearchTransactions(transactionType, dateFrom, dateTo string, pageNo, rowPerPage int, authInfo auth.AuthInfo) ([]*dtos.BillingTransaction, bool, int) {
	if rowPerPage <= 0 {
		rowPerPage = config.GetInt(constans.StaticDefaultEnv)
	}
	if pageNo <= 0 {
		pageNo = 1
	}

	// Parse dates
	var dateFromParsed, dateToParsed time.Time
	//dateFromParsed := time.Now()
	//dateToParsed := time.Now()

	if dateFrom != "" {
		parsed, err := time.Parse("2006-01-02", dateFrom)
		if err == nil {
			dateFromParsed = parsed
			dateFromParsed = parsed.Add(-7 * time.Hour)
		}
	}
	if dateTo != "" {
		parsed, err := time.Parse("2006-01-02", dateTo)
		if err == nil {
			dateToParsed = parsed.Add(-7 * time.Hour)
			dateToParsed = parsed
		}
	}

	mdls, haveNext, count := billingTransaction.GetRepo().Search("", authInfo.ClientId, transactionType, "", dateFromParsed, dateToParsed, pageNo, rowPerPage)

	ret := make([]*dtos.BillingTransaction, len(mdls))
	for i, mdl := range mdls {

		tmp := &dtos.BillingTransaction{}
		_ = copier.Copy(&tmp, &mdl)
		ret[i] = tmp

		msgLogMdl := messageLog.GetRepo().GetById(mdl.ReferenceId)
		if msgLogMdl != nil {
			waTemplateMdl := waTemplate.GetRepo().GetById(msgLogMdl.TemplateId)
			if waTemplateMdl != nil {
				ret[i].BillingName = waTemplateMdl.Category
			}
		}
	}

	customLogger.DebugWithData("SearchTransactions completed", map[string]interface{}{
		"component": "BillingService",
		"function":  "SearchTransactions",
		"count":     len(ret),
		"have_next": haveNext,
	})
	return ret, haveNext, int(count)
}

func (o *BillingSvcImpl) DeductBalance(clientId string, amount float64, referenceId, referenceType, description, userId string) (*dtos.BillingTransaction, *gocom.CodedError) {
	customLogger.InfoWithData("DeductBalance started", map[string]interface{}{
		"component":      "BillingService",
		"function":       "DeductBalance",
		"client_id":      clientId,
		"amount":         amount,
		"reference_type": referenceType,
	})

	clientMdl := client.GetRepo().GetById(clientId)
	if clientMdl == nil {
		customLogger.WarnWithData("Client not found", map[string]interface{}{
			"component": "BillingService",
			"function":  "DeductBalance",
			"client_id": clientId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	balanceBefore := clientMdl.Balance
	balanceAfter := balanceBefore - amount

	// Create transaction
	txn := &billingTransaction.BillingTransaction{
		ID:            ulid.Make().String(),
		ClientId:      clientId,
		Type:          constans.BillingTransactionTypeUsage,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		ReferenceId:   referenceId,
		ReferenceType: referenceType,
		Description:   description,
		CreatedBy:     userId,
	}

	err := billingTransaction.GetRepo().Create(txn).Error
	if err != nil {
		customLogger.ErrorWithData("Failed to create transaction", map[string]interface{}{
			"component": "BillingService",
			"function":  "DeductBalance",
			"client_id": clientId,
			"error":     err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	// Update client balance
	clientMdl.Balance = balanceAfter
	clientMdl.TotalUsage += amount
	errUpdate := client.GetRepo().Update(clientMdl)
	if errUpdate != nil {
		customLogger.ErrorWithData("Failed to update client balance", map[string]interface{}{
			"component": "BillingService",
			"function":  "DeductBalance",
			"client_id": clientId,
			"error":     errUpdate.Error(),
		})
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	ret := &dtos.BillingTransaction{}
	_ = copier.Copy(&ret, &txn)

	customLogger.InfoWithData("Balance deducted successfully", map[string]interface{}{
		"component":      "BillingService",
		"function":       "DeductBalance",
		"transaction_id": txn.ID,
		"amount":         amount,
		"balance_after":  balanceAfter,
	})
	return ret, nil
}

func (o *BillingSvcImpl) TopupBalance(clientId string, amount float64, description, userId string) (*dtos.BillingTransaction, *gocom.CodedError) {
	customLogger.InfoWithData("TopupBalance started", map[string]interface{}{
		"component": "BillingService",
		"function":  "TopupBalance",
		"client_id": clientId,
		"amount":    amount,
	})

	clientMdl := client.GetRepo().GetById(clientId)
	if clientMdl == nil {
		customLogger.WarnWithData("Client not found", map[string]interface{}{
			"component": "BillingService",
			"function":  "TopupBalance",
			"client_id": clientId,
		})
		return nil, common.ERR_NOT_FOUND
	}

	balanceBefore := clientMdl.Balance
	balanceAfter := balanceBefore + amount

	// Create transaction
	txn := &billingTransaction.BillingTransaction{
		ID:            ulid.Make().String(),
		ClientId:      clientId,
		Type:          constans.BillingTransactionTypeTopup,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		Description:   description,
		CreatedBy:     userId,
	}

	err := billingTransaction.GetRepo().Create(txn).Error
	if err != nil {
		customLogger.ErrorWithData("Failed to create transaction", map[string]interface{}{
			"component": "BillingService",
			"function":  "TopupBalance",
			"client_id": clientId,
			"error":     err.Error(),
		})
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	// Update client balance
	clientMdl.Balance = balanceAfter
	errUpdate := client.GetRepo().Update(clientMdl)
	if errUpdate != nil {
		customLogger.ErrorWithData("Failed to update client balance", map[string]interface{}{
			"component": "BillingService",
			"function":  "TopupBalance",
			"client_id": clientId,
			"error":     errUpdate.Error(),
		})
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	ret := &dtos.BillingTransaction{}
	_ = copier.Copy(&ret, &txn)

	customLogger.InfoWithData("Balance topped up successfully", map[string]interface{}{
		"component":      "BillingService",
		"function":       "TopupBalance",
		"transaction_id": txn.ID,
		"amount":         amount,
		"balance_after":  balanceAfter,
	})
	return ret, nil
}

func (o *BillingSvcImpl) GetUsageSummary(period string, authInfo auth.AuthInfo) (*dtos.UsageSummary, *gocom.CodedError) {
	// Parse period (format: YYYY-MM)
	if len(period) != 7 {
		customLogger.WarnWithData("Invalid period format", map[string]interface{}{
			"component": "BillingService",
			"function":  "GetUsageSummary",
			"period":    period,
		})
		return nil, gocom.NewError(400, "Invalid period format. Use YYYY-MM")
	}

	startDate, err := time.Parse("2006-01", period)
	if err != nil {
		customLogger.WarnWithData("Invalid period", map[string]interface{}{
			"component": "BillingService",
			"function":  "GetUsageSummary",
			"period":    period,
			"error":     err.Error(),
		})
		return nil, gocom.NewError(400, "Invalid period format. Use YYYY-MM")
	}
	endDate := startDate.AddDate(0, 1, 0) // Next month

	// Get all message logs for the period
	logs := messageLog.GetRepo().GetByClientIdAndDateRange(authInfo.ClientId, &startDate, &endDate)

	summary := &dtos.UsageSummary{
		ClientId:      authInfo.ClientId,
		Period:        period,
		TotalMessages: 0,
		TotalCost:     0,
		WhatsAppCount: 0,
		WhatsAppCost:  0,
		SmsCount:      0,
		SmsCost:       0,
		EmailCount:    0,
		EmailCost:     0,
	}

	for _, log := range logs {
		summary.TotalMessages++
		summary.TotalCost += log.Cost

		switch log.Type {
		case "whatsapp":
			summary.WhatsAppCount++
			summary.WhatsAppCost += log.Cost
		case "sms":
			summary.SmsCount++
			summary.SmsCost += log.Cost
		case "email":
			summary.EmailCount++
			summary.EmailCost += log.Cost
		}
	}

	customLogger.InfoWithData("GetUsageSummary completed", map[string]interface{}{
		"component":      "BillingService",
		"function":       "GetUsageSummary",
		"total_messages": summary.TotalMessages,
		"total_cost":     summary.TotalCost,
	})
	return summary, nil
}

// CalculateMessageCostWithCategory calculates cost based on service type and category
// For WhatsApp: category can be "MARKETING", "UTILITY", "AUTHENTICATION"
// For SMS/Email: category is ignored
// Reads pricing from database (pricing table) instead of hardcoded values
func (o *BillingSvcImpl) CalculateMessageCostWithCategory(messageType, category, clientId string) float64 {
	// Normalize inputs
	normalizedType := strings.ToLower(strings.TrimSpace(messageType))
	normalizedCategory := strings.ToLower(strings.TrimSpace(category))

	// For non-whatsapp services, category might be empty
	if normalizedType != "whatsapp" {
		normalizedCategory = ""
	}

	// Try to get pricing from database
	pricingMdl := pricing.GetRepo().GetByServiceAndCategory(normalizedType, normalizedCategory, &clientId)

	if pricingMdl != nil {
		customLogger.DebugWithData("Found pricing in database", map[string]interface{}{
			"component": "BillingService",
			"function":  "CalculateMessageCostWithCategory",
			"type":      normalizedType,
			"category":  normalizedCategory,
			"cost":      pricingMdl.Cost,
		})
		return pricingMdl.Cost
	}

	// Fallback to hardcoded values if not found in database (backward compatibility)
	customLogger.WarnWithData("Pricing not found in database, using fallback", map[string]interface{}{
		"component": "BillingService",
		"function":  "CalculateMessageCostWithCategory",
		"type":      normalizedType,
		"category":  normalizedCategory,
	})

	switch normalizedType {
	case "whatsapp":
		switch normalizedCategory {
		case "marketing":
			return 600.0
		case "authentication", "auth":
			return 175.0
		case "utility":
			return 350.0
		default:
			customLogger.WarnWithData("Unknown category, defaulting to UTILITY pricing", map[string]interface{}{
				"component": "BillingService",
				"function":  "CalculateMessageCostWithCategory",
				"category":  category,
			})
			return 350.0
		}
	case "sms":
		return 350.0
	case "email":
		return 50.0
	default:
		customLogger.WarnWithData("Unknown message type", map[string]interface{}{
			"component":    "BillingService",
			"function":     "CalculateMessageCostWithCategory",
			"message_type": messageType,
		})
		return 0
	}
}

// CalculateMessageCost calculates cost based on service type only
// For backward compatibility - defaults WhatsApp to utility pricing
func (o *BillingSvcImpl) CalculateMessageCost(messageType, clientId string) float64 {
	if messageType == "whatsapp" {
		return o.CalculateMessageCostWithCategory("whatsapp", "utility", clientId)
	}
	return o.CalculateMessageCostWithCategory(messageType, "", clientId)
}
