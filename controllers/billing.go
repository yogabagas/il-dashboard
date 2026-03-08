package controllers

import (
	"strconv"
	"sync"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	common "gitlab.com/anti_metter/switching_common"
	a "gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/services"
)

type BillingController struct {
	gocom.Controller
}

var billingController *BillingController
var onceBillingController sync.Once

func GetBillingController() *BillingController {
	onceBillingController.Do(func() {
		billingController = &BillingController{}
	})
	return billingController
}

func (o *BillingController) Init() {
	gocom.GET("/api/v1/instant-link/billing/info", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.getInfo)
	gocom.POST("/api/v1/instant-link/billing/transactions", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.createTransaction)
	gocom.GET("/api/v1/instant-link/billing/transactions/:id", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.getTransaction)
	gocom.GET("/api/v1/instant-link/billing/transactions", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.searchTransactions)
	gocom.POST("/api/v1/instant-link/billing/topup", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.topup)
	gocom.GET("/api/v1/instant-link/billing/usage-summary", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.usageSummary)
}

func (o *BillingController) getInfo(ctx gocom.Context) error {
	logger.Debug("BillingController.getInfo - Start")

	authInfo := a.Get(ctx)
	ret, codedErr := services.GetBillingService().GetClientBillingInfo(authInfo)
	//gocom.WriteResponse(ctx, ret, codedErr)
	if codedErr != nil {
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult(ret)
}

func (o *BillingController) createTransaction(ctx gocom.Context) error {
	logger.Debug("BillingController.createTransaction - Start")

	req := dtos.BillingTransactionReq{}
	_ = ctx.Bind(&req)

	authInfo := a.Get(ctx)

	ret, codedErr := services.GetBillingService().CreateTransaction(req, authInfo)
	if codedErr != nil {
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult(ret)
}

func (o *BillingController) getTransaction(ctx gocom.Context) error {
	logger.Debug("BillingController.getTransaction - Start")

	authInfo := a.Get(ctx)
	transactionId := ctx.Param("id")

	ret, codedErr := services.GetBillingService().GetTransactionById(transactionId, authInfo)
	if codedErr != nil {
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult(ret)
}

func (o *BillingController) searchTransactions(ctx gocom.Context) error {
	logger.Debug("BillingController.searchTransactions - Start")

	authInfo := a.Get(ctx)
	transactionType := ctx.Query("type")
	dateFrom := ctx.Query("dateFrom")
	dateTo := ctx.Query("dateTo")
	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPerPage, _ := strconv.ParseInt(ctx.Query("rowPerPage"), 10, 32)

	ret, haveNext, count := services.GetBillingService().SearchTransactions(transactionType, dateFrom, dateTo, int(pageNo), int(rowPerPage), authInfo)

	return common.SendPaged(ctx, ret, int(pageNo), haveNext, int64(count))
}

func (o *BillingController) topup(ctx gocom.Context) error {
	logger.Debug("BillingController.topup - Start")

	authInfo := a.Get(ctx)
	var req struct {
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
	}

	if req.Amount <= 0 {
		logger.Warnf("BillingController.topup - Amount must be positive %f", req.Amount)
		err := gocom.NewError(400, "Amount must be positive")
		return ctx.SendError(err)
	}

	ret, codedErr := services.GetBillingService().TopupBalance(authInfo.ClientId, req.Amount, req.Description, authInfo.UserId)
	if codedErr != nil {
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult(ret)
}

func (o *BillingController) usageSummary(ctx gocom.Context) error {
	logger.Debug("BillingController.usageSummary - Start")

	authInfo := a.Get(ctx)
	period := ctx.Query("period") // Format: YYYY-MM

	if period == "" {
		logger.Warn("BillingController.usageSummary - Period is required")
		err := gocom.NewError(400, "Period is required (format: YYYY-MM)")
		return ctx.SendError(err)
	}

	ret, codedErr := services.GetBillingService().GetUsageSummary(period, authInfo)
	if codedErr != nil {
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult(ret)
}
