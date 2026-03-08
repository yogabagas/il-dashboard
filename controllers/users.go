package controllers

import (
	"fmt"
	"strconv"
	"sync"

	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/services"

	"github.com/ariandi/gocom"
	a "gitlab.com/anti_metter/switching_common/auth"
)

type UserController struct {
}

func (o *UserController) Init() {

	gocom.GET("/api/v1/instant-link/users", a.Allow(a.Role(a.ADMIN, a.OWNER), a.Access("SEARCH_CLIENT")), o.searchUser)
	gocom.POST("/api/v1/instant-link/users", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.createUser)
	gocom.POST("/api/v1/instant-link/users/upload-media", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.UserMediaRequest)
	gocom.GET("/api/v1/instant-link/users/me", a.Allow(a.Logged), o.getMe)
	gocom.PUT("/api/v1/instant-link/users/:id", a.Allow(a.Role(a.ADMIN, a.OWNER), a.Access("UPDATE_CLIENT")), o.updateUser)
	gocom.GET("/api/v1/instant-link/users/:id", a.Allow(a.Logged, a.Access("GET_CLIENT")), o.getUserById)
	gocom.DELETE("/api/v1/instant-link/users/:id", a.Allow(a.Role(a.ADMIN, a.OWNER), a.Access("DELETE_CLIENT")), o.deleteUser)
}

func (o *UserController) getUserById(ctx gocom.Context) error {

	req := dtos.UserReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetUserService().Get(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *UserController) createUser(ctx gocom.Context) error {

	req := dtos.UserReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetUserService().Create(req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *UserController) UserMediaRequest(ctx gocom.Context) error {

	req := dtos.UserMediaRequest{}
	_ = ctx.Bind(&req)

	//ctx.FormValue()
	//fmt.Println("[IndexController healthCheck] start :", ctx.Body())
	//fmt.Println(string(ctx.Body())) // Outputs: This is the body content.
	fmt.Println("[IndexController healthCheck] start :", req)
	fmt.Println("[IndexController healthCheck] start :", ctx.FormValue("table_name"))
	fmt.Println("[IndexController healthCheck] start :", ctx.FormValue("ref_id"))
	fmt.Println("[IndexController healthCheck] start :", ctx.FormValue("media_name"))
	fmt.Println("[IndexController healthCheck] start :", ctx.FormValue("file_type"))
	fmt.Println("[IndexController healthCheck] start :", ctx.Get("media"))
	fmt.Println("[IndexController healthCheck] start :", ctx.Get("Content-Type"))
	fmt.Println("[IndexController healthCheck] start :", ctx.Get("Accept"))

	// Retrieve the file uploaded under the 'media' field
	file := ctx.FormValue("media")

	// File metadata (e.g., filename)
	fmt.Println("Uploaded file:", file)

	// auth := a.Get(ctx)

	// if auth.UserId != common.PROVIDER_ID {
	// 	if !client.GetSvc().IsAcessible(ctx.Param("id"), auth.UserId) {

	// 		return ctx.SendError(common.ERR_NOT_FOUND)
	// 	}
	// }

	return nil

	//ret, err := services.GetUserSvc().UploadUserMedia(req, a.Get(ctx))
	//
	//if err != nil {
	//	return ctx.SendError(err)
	//}
	//
	//return ctx.SendResult(ret)
}

func (o *UserController) updateUser(ctx gocom.Context) error {

	req := dtos.UserReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetUserService().Update(ctx.Param("id"), req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *UserController) searchUser(ctx gocom.Context) error {

	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPerPage, _ := strconv.ParseInt(ctx.Query("rowPerPage"), 10, 32)

	//parentId := ctx.Query("parentId")

	auth := a.Get(ctx)

	//if auth.UserId != common.PROVIDER_ID {
	//	parentId = auth.UserId
	//}

	filter := ctx.Query("filter")
	roleId := ctx.Query("roleId")
	clientId := ctx.Query("clientId")

	ret, haveNext, count := services.GetUserService().Search(filter, clientId, roleId, int(pageNo), int(rowPerPage), auth)

	return common.SendPaged(ctx, ret, int(pageNo), haveNext, count)
}

func (o *UserController) getMe(ctx gocom.Context) error {

	auth := a.Get(ctx)
	ret, err := services.GetUserService().Get(auth.UserId, auth)

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *UserController) deleteUser(ctx gocom.Context) error {

	err := services.GetUserService().Delete(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(true)
}

//---------------------------------------

var userController *UserController
var userControllerOnce sync.Once

func GetUserController() *UserController {

	if userController == nil {

		userControllerOnce.Do(func() {
			userController = &UserController{}
		})
	}

	return userController
}
