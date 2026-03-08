package controllers

import (
	"strconv"
	"sync"

	"github.com/ariandi/gocom"
	common "gitlab.com/anti_metter/switching_common"
	a "gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/services"
	"gitlab.com/bot3342545/il-dashboard/utils"
)

type ContactsController struct {
}

func (o *ContactsController) Init() {
	gocom.POST("/api/v1/instant-link/contacts", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.create)
	gocom.PUT("/api/v1/instant-link/contacts/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.update)
	gocom.GET("/api/v1/instant-link/contacts/:id", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.getById)
	gocom.GET("/api/v1/instant-link/contacts", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.search)
	gocom.DELETE("/api/v1/instant-link/contacts/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.delete)
	gocom.POST("/api/v1/instant-link/contacts/import", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.importCSV)
	gocom.POST("/api/v1/instant-link/contacts/upload", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.uploadFile)
	gocom.POST("/api/v1/instant-link/contacts/bulk-delete", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.bulkDelete)
	gocom.POST("/api/v1/instant-link/contacts/bulk-update-status", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.bulkUpdateStatus)
}

func (o *ContactsController) create(ctx gocom.Context) error {
	req := dtos.ContactReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetContactsService().Create(req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *ContactsController) update(ctx gocom.Context) error {
	req := dtos.ContactUpdateReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetContactsService().Update(ctx.Param("id"), req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *ContactsController) getById(ctx gocom.Context) error {
	ret, err := services.GetContactsService().GetById(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *ContactsController) search(ctx gocom.Context) error {
	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPerPage, _ := strconv.ParseInt(ctx.Query("rowPerPage"), 10, 32)

	filter := ctx.Query("filter")
	phone := ctx.Query("phone")
	email := ctx.Query("email")
	status := ctx.Query("status")
	clientId := ctx.Query("clientId")

	var tags []string
	tagsStr := ctx.Query("tags")
	if tagsStr != "" {
		tags = utils.SplitAndTrim(tagsStr, ",")
	}

	ret, haveNext, count := services.GetContactsService().Search(filter, phone, email, status, clientId, tags, int(pageNo), int(rowPerPage), a.Get(ctx))

	return common.SendPaged(ctx, ret, int(pageNo), haveNext, int64(count))
}

func (o *ContactsController) delete(ctx gocom.Context) error {
	err := services.GetContactsService().Delete(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(map[string]interface{}{"success": true, "id": ctx.Param("id")})
}

func (o *ContactsController) importCSV(ctx gocom.Context) error {
	req := dtos.ContactImportReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetContactsService().ImportFromCSV(req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *ContactsController) uploadFile(ctx gocom.Context) error {
	campaignId := ctx.FormValue("campaignId")
	if campaignId == "" {
		return ctx.SendError(common.ERR_INVALID_REQUEST)
	}

	file, formErr := ctx.FormFile("file")
	if formErr != nil {
		return ctx.SendError(common.ERR_INVALID_REQUEST)
	}
	if file == nil {
		return ctx.SendError(common.ERR_INVALID_REQUEST)
	}

	ret, err := services.GetContactsService().UploadContactsFile(file, campaignId, a.Get(ctx))
	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *ContactsController) bulkDelete(ctx gocom.Context) error {
	var req struct {
		ContactIds []string `json:"contactIds"`
	}
	_ = ctx.Bind(&req)

	err := services.GetContactsService().BulkDelete(req.ContactIds, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(map[string]interface{}{"success": true, "count": len(req.ContactIds)})
}

func (o *ContactsController) bulkUpdateStatus(ctx gocom.Context) error {
	var req struct {
		ContactIds []string `json:"contactIds"`
		Status     string   `json:"status"`
	}
	_ = ctx.Bind(&req)

	err := services.GetContactsService().BulkUpdateStatus(req.ContactIds, req.Status, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(map[string]interface{}{"success": true, "count": len(req.ContactIds)})
}

//---------------------------------------

var contactsController *ContactsController
var contactsControllerOnce sync.Once

func GetContactsController() *ContactsController {

	if contactsController == nil {

		contactsControllerOnce.Do(func() {
			contactsController = &ContactsController{}
		})
	}

	return contactsController
}
