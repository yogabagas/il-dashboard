package csControllers

import (
	"strconv"
	"sync"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	common "gitlab.com/anti_metter/switching_common"
	a "gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/customer_service/dtos"
	"gitlab.com/bot3342545/il-dashboard/customer_service/pubsub"
	"gitlab.com/bot3342545/il-dashboard/customer_service/services"
)

type CustomerServiceController struct {
	service       services.CustomerServiceSvc
	pubsubHandler pubsub.CSPubSubHandler
}

func (o *CustomerServiceController) Init() {
	logger.Infof("[CustomerServiceController] Registering CS Hub routes...")

	// Initialize pubsub handlers (Init already called inside GetCSPubSubHandler)
	o.service = services.GetCustomerServiceSvc()
	o.pubsubHandler = pubsub.GetCSPubSubHandler(o.service)

	// Case management endpoints (ADMIN, OWNER, STAFF can manage cases)
	gocom.GET("/api/v1/cs/cases", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.getCases)
	gocom.GET("/api/v1/cs/cases/:id", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.getCaseDetails)
	gocom.POST("/api/v1/cs/cases/:id/claim", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.claimCase)
	gocom.POST("/api/v1/cs/cases/:id/messages", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.sendMessage)
	gocom.POST("/api/v1/cs/cases/:id/notes", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.addNote)
	gocom.POST("/api/v1/cs/cases/:id/resolve", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.resolveCase)
	gocom.POST("/api/v1/cs/cases/:id/close", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.closeCase)
	gocom.POST("/api/v1/cs/cases/:id/escalate", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.escalateCase)
	gocom.GET("/api/v1/cs/statistics", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.getStatistics)

	logger.Infof("[CustomerServiceController] CS Hub routes registered successfully")
}

func (o *CustomerServiceController) getCases(ctx gocom.Context) error {
	logger.Infof("[CustomerServiceController getCases] Getting cases list")

	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPerPage, _ := strconv.ParseInt(ctx.Query("rowPerPage"), 10, 32)

	if pageNo <= 0 {
		pageNo = 1
	}
	if rowPerPage <= 0 {
		rowPerPage = 20
	}

	filters := &dtos.CaseFilters{
		Status:     ctx.Query("status"),
		Severity:   ctx.Query("severity"),
		Category:   ctx.Query("category"),
		AssignedTo: ctx.Query("assigned_to"),
		Sort:       ctx.Query("sort"),
		SortOrder:  ctx.Query("sort_order"),
		Page:       int(pageNo),
		Limit:      int(rowPerPage),
	}

	// Get client_id from auth
	authInfo := a.Get(ctx)
	if authInfo.ClientId != "" {
		filters.ClientId = authInfo.ClientId
	}

	// SLA breach filter
	if ctx.Query("sla_breached") != "" {
		slaBreached := ctx.Query("sla_breached") == "true"
		filters.SlaBreached = &slaBreached
	}

	// Get cases
	response, err := o.service.GetCasesList(filters)
	if err != nil {
		logger.Errorf("[CustomerServiceController getCases] Failed: %v", err)
		return ctx.SendError(err)
	}

	// Send paginated response with stats
	haveNext := response.Pagination.Page < response.Pagination.TotalPages
	return common.SendPaged(ctx, response.Cases, int(pageNo), haveNext, int64(response.Pagination.Total))
}

func (o *CustomerServiceController) getCaseDetails(ctx gocom.Context) error {
	caseId := ctx.Param("id")
	logger.Infof("[CustomerServiceController getCaseDetails] Case: %s", caseId)

	response, err := o.service.GetCaseDetails(caseId)
	if err != nil {
		logger.Errorf("[CustomerServiceController getCaseDetails] Failed: %v", err)
		return ctx.SendError(err)
	}

	return ctx.SendResult(response)
}

func (o *CustomerServiceController) claimCase(ctx gocom.Context) error {
	caseId := ctx.Param("id")
	logger.Infof("[CustomerServiceController claimCase] Case: %s", caseId)

	var req dtos.ClaimCaseRequest
	_ = ctx.Bind(&req)

	authInfo := a.Get(ctx)
	csUserId := authInfo.UserId

	caseResp, err := o.service.ClaimCase(caseId, csUserId, req.Severity, req.Priority, req.Notes)
	if err != nil {
		logger.Errorf("[CustomerServiceController claimCase] Failed: %v", err)
		return ctx.SendError(err)
	}

	return ctx.SendResult(caseResp)
}

func (o *CustomerServiceController) sendMessage(ctx gocom.Context) error {
	caseId := ctx.Param("id")
	logger.Infof("[CustomerServiceController sendMessage] Case: %s", caseId)

	var req dtos.SendMessageRequest
	_ = ctx.Bind(&req)

	authInfo := a.Get(ctx)
	csUserId := authInfo.UserId

	err := o.service.SendMessage(caseId, csUserId, req.MessageContent)
	if err != nil {
		logger.Errorf("[CustomerServiceController sendMessage] Failed: %v", err)
		return ctx.SendError(err)
	}

	return ctx.SendResult(map[string]interface{}{
		"success": true,
		"case_id": caseId,
	})
}

func (o *CustomerServiceController) addNote(ctx gocom.Context) error {
	caseId := ctx.Param("id")
	logger.Infof("[CustomerServiceController addNote] Case: %s", caseId)

	var req dtos.AddNoteRequest
	_ = ctx.Bind(&req)

	authInfo := a.Get(ctx)
	csUserId := authInfo.UserId

	err := o.service.AddNote(caseId, csUserId, req.NoteType, req.NoteContent, req.IsInternal)
	if err != nil {
		logger.Errorf("[CustomerServiceController addNote] Failed: %v", err)
		return ctx.SendError(err)
	}

	return ctx.SendResult(map[string]interface{}{
		"success": true,
		"case_id": caseId,
	})
}

func (o *CustomerServiceController) resolveCase(ctx gocom.Context) error {
	caseId := ctx.Param("id")
	logger.Infof("[CustomerServiceController resolveCase] Case: %s", caseId)

	var req dtos.ResolveCaseRequest
	_ = ctx.Bind(&req)

	authInfo := a.Get(ctx)
	csUserId := authInfo.UserId

	err := o.service.ResolveCase(caseId, csUserId, req.ResolutionNotes)
	if err != nil {
		logger.Errorf("[CustomerServiceController resolveCase] Failed: %v", err)
		return ctx.SendError(err)
	}

	return ctx.SendResult(map[string]interface{}{
		"success": true,
		"case_id": caseId,
	})
}

func (o *CustomerServiceController) closeCase(ctx gocom.Context) error {
	caseId := ctx.Param("id")
	logger.Infof("[CustomerServiceController closeCase] Case: %s", caseId)

	var req dtos.CloseCaseRequest
	_ = ctx.Bind(&req)

	authInfo := a.Get(ctx)
	csUserId := authInfo.UserId

	err := o.service.CloseCase(caseId, csUserId, req.CloseNotes)
	if err != nil {
		logger.Errorf("[CustomerServiceController closeCase] Failed: %v", err)
		return ctx.SendError(err)
	}

	return ctx.SendResult(map[string]interface{}{
		"success": true,
		"case_id": caseId,
		"message": "Case closed and billing triggered",
	})
}

func (o *CustomerServiceController) escalateCase(ctx gocom.Context) error {
	caseId := ctx.Param("id")
	logger.Infof("[CustomerServiceController escalateCase] Case: %s", caseId)

	var req dtos.EscalateCaseRequest
	_ = ctx.Bind(&req)

	authInfo := a.Get(ctx)
	csUserId := authInfo.UserId

	err := o.service.EscalateCase(caseId, csUserId, req.EscalateTo, req.EscalationReason, req.Severity, req.Priority)
	if err != nil {
		logger.Errorf("[CustomerServiceController escalateCase] Failed: %v", err)
		return ctx.SendError(err)
	}

	return ctx.SendResult(map[string]interface{}{
		"success":      true,
		"case_id":      caseId,
		"escalated_to": req.EscalateTo,
	})
}

func (o *CustomerServiceController) getStatistics(ctx gocom.Context) error {
	logger.Infof("[CustomerServiceController getStatistics] Getting statistics")

	authInfo := a.Get(ctx)
	clientId := authInfo.ClientId
	period := ctx.Query("period")
	if period == "" {
		period = "today"
	}

	response, err := o.service.GetStatistics(clientId, period)
	if err != nil {
		logger.Errorf("[CustomerServiceController getStatistics] Failed: %v", err)
		return ctx.SendError(err)
	}

	return ctx.SendResult(response)
}

//---------------------------------------

var customerServiceController *CustomerServiceController
var customerServiceControllerOnce sync.Once

func GetCustomerServiceController() *CustomerServiceController {
	if customerServiceController == nil {
		customerServiceControllerOnce.Do(func() {
			customerServiceController = &CustomerServiceController{}
		})
	}
	return customerServiceController
}
