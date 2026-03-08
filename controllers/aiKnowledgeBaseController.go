package controllers

import (
	"strconv"
	"sync"

	"github.com/ariandi/gocom"
	common "gitlab.com/anti_metter/switching_common"
	a "gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/services"
)

type AIKnowledgeBaseController struct {
	gocom.Controller
}

var aiKnowledgeBaseController *AIKnowledgeBaseController
var onceAIKnowledgeBaseController sync.Once

func GetAIKnowledgeBaseController() *AIKnowledgeBaseController {
	onceAIKnowledgeBaseController.Do(func() {
		aiKnowledgeBaseController = &AIKnowledgeBaseController{}
	})
	return aiKnowledgeBaseController
}

func (o *AIKnowledgeBaseController) Init() {
	gocom.POST("/api/v1/instant-link/ai/knowledge-base/create", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.create)
	gocom.POST("/api/v1/instant-link/ai/knowledge-base/update", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.update)
	gocom.POST("/api/v1/instant-link/ai/knowledge-base/delete", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.delete)
	gocom.GET("/api/v1/instant-link/ai/knowledge-base/:id", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.getById)
	gocom.GET("/api/v1/instant-link/ai/knowledge-base", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.search)
}

func (o *AIKnowledgeBaseController) create(ctx gocom.Context) error {
	req := dtos.AiKnowledgeBaseCreateReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetAIKnowledgeBaseService().Create(req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *AIKnowledgeBaseController) update(ctx gocom.Context) error {
	req := dtos.AiKnowledgeBaseUpdateReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetAIKnowledgeBaseService().Update(req.ID, req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *AIKnowledgeBaseController) delete(ctx gocom.Context) error {
	var req struct {
		ID string `json:"id" binding:"required"`
	}
	_ = ctx.Bind(&req)

	err := services.GetAIKnowledgeBaseService().Delete(req.ID, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(map[string]interface{}{"success": true, "id": req.ID})
}

func (o *AIKnowledgeBaseController) getById(ctx gocom.Context) error {
	id := ctx.Param("id")

	ret, err := services.GetAIKnowledgeBaseService().GetById(id, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *AIKnowledgeBaseController) search(ctx gocom.Context) error {
	// Parse all from query params (following billing template)
	filter := ctx.Query("filter")
	clientId := ctx.Query("clientId")
	category := ctx.Query("category")
	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPerPage, _ := strconv.ParseInt(ctx.Query("rowPerPage"), 10, 32)

	req := dtos.AiKnowledgeBaseSearchReq{
		Filter:     filter,
		ClientId:   clientId,
		Category:   category,
		PageNo:     int(pageNo),
		RowPerPage: int(rowPerPage),
	}

	ret, haveNext, count := services.GetAIKnowledgeBaseService().Search(req, a.Get(ctx))

	return common.SendPaged(ctx, ret, int(pageNo), haveNext, int64(count))
}
