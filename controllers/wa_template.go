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

type WATemplateController struct {
}

func (o *WATemplateController) Init() {
	// CRUD endpoints for WhatsApp templates
	// Upload image endpoint
	gocom.POST("/api/v1/instant-link/wa-templates/upload", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.uploadFileToWa)

	gocom.POST("/api/v1/instant-link/wa-templates", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.createTemplate)
	gocom.PUT("/api/v1/instant-link/wa-templates/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.updateTemplate)
	gocom.GET("/api/v1/instant-link/wa-templates/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.getTemplate)
	gocom.GET("/api/v1/instant-link/wa-templates", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.searchTemplate)
	gocom.DELETE("/api/v1/instant-link/wa-templates/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.deleteTemplate)

	// Template workflow endpoints
	gocom.POST("/api/v1/instant-link/wa-templates/:id/submit", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.submitTemplate)
	gocom.POST("/api/v1/instant-link/wa-templates/:id/approve", a.Allow(a.Role(a.ADMIN)), o.approveTemplate)
	gocom.POST("/api/v1/instant-link/wa-templates/:id/reject", a.Allow(a.Role(a.ADMIN)), o.rejectTemplate)

	// WhatsApp Business API integration
	gocom.POST("/api/v1/instant-link/wa-templates/:id/submit-to-whatsapp", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.submitToWhatsApp)
}

func (o *WATemplateController) createTemplate(ctx gocom.Context) error {
	req := dtos.WATemplateReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetWATemplateSvc().Create(req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *WATemplateController) uploadFileToWa(ctx gocom.Context) error {

	logger.Infof("[WATemplateController uploadFileToWa] start")

	req := dtos.WATemplateUploadFileReq{}
	if err := ctx.Bind(&req); err != nil {
		return ctx.SendError(common.ERR_INVALID_REQUEST)
	}

	ret, err := services.GetWATemplateSvc().UploadMediaToWhatsApp([]byte(req.File), req.FileName, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *WATemplateController) updateTemplate(ctx gocom.Context) error {
	req := dtos.WATemplateUpdateReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetWATemplateSvc().Update(ctx.Param("id"), req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *WATemplateController) getTemplate(ctx gocom.Context) error {
	auth := a.Get(ctx)
	ret, err := services.GetWATemplateSvc().Get(ctx.Param("id"), auth)

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *WATemplateController) searchTemplate(ctx gocom.Context) error {
	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPerPage, _ := strconv.ParseInt(ctx.Query("rowPerPage"), 10, 32)

	clientId := ctx.Query("clientId")
	status := ctx.Query("status")
	waStatus := ctx.Query("waStatus")
	category := ctx.Query("category")

	auth := a.Get(ctx)

	// Non-provider users can only see their own templates
	if auth.ClientId != common.PROVIDER_ID {
		clientId = auth.ClientId
	}

	// Get sort parameters from query
	sortBy := ctx.Query("sortBy")
	sortOrder := ctx.Query("sortOrder")

	ret, haveNext, count := services.GetWATemplateSvc().Search(
		ctx.Query("filter"),
		clientId,
		status,
		waStatus,
		category,
		sortBy,
		sortOrder,
		int(pageNo),
		int(rowPerPage),
	)

	return common.SendPaged(ctx, ret, int(pageNo), haveNext, count)
}

func (o *WATemplateController) deleteTemplate(ctx gocom.Context) error {
	err := services.GetWATemplateSvc().Delete(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(true)
}

func (o *WATemplateController) submitTemplate(ctx gocom.Context) error {
	ret, err := services.GetWATemplateSvc().Submit(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *WATemplateController) approveTemplate(ctx gocom.Context) error {
	type ApproveReq struct {
		WATemplateId string `json:"waTemplateId"`
	}

	req := ApproveReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetWATemplateSvc().Approve(ctx.Param("id"), req.WATemplateId, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *WATemplateController) rejectTemplate(ctx gocom.Context) error {
	type RejectReq struct {
		Reason string `json:"reason"`
	}

	req := RejectReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetWATemplateSvc().Reject(ctx.Param("id"), req.Reason, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *WATemplateController) submitToWhatsApp(ctx gocom.Context) error {
	type SubmitToWAReq struct {
		WabaId      string `json:"wabaId"`
		AccessToken string `json:"accessToken"`
	}

	req := SubmitToWAReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetWATemplateSvc().SubmitToWhatsApp(ctx.Param("id"), req.WabaId, req.AccessToken, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

//---------------------------------------

var waTemplateController *WATemplateController
var waTemplateControllerOnce sync.Once

func GetWATemplateController() *WATemplateController {
	if waTemplateController == nil {
		waTemplateControllerOnce.Do(func() {
			waTemplateController = &WATemplateController{}
		})
	}

	return waTemplateController
}
