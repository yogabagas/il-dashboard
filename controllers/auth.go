package controllers

import (
	"github.com/ariandi/gocom"
	a "gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/services"

	"sync"
)

type AuthController struct {
}

func (o *AuthController) Init() {

	gocom.POST("/api/v1/instant-link/auth/login", o.login)
	// gocom.POST("/auth/loginbytoken", o.loginByToken)
	gocom.POST("/api/v1/instant-link/auth/logout", a.Allow(a.Logged), o.logout)
	gocom.POST("/api/v1/instant-link/auth/refresh-token", o.refreshToken)
}

func (o *AuthController) login(ctx gocom.Context) error {

	req := dtos.LoginReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetAuthSvc().Login(req)

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *AuthController) logout(ctx gocom.Context) error {

	auth := a.Get(ctx)
	err := services.GetAuthSvc().Logout(auth)

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(true)
}

func (o *AuthController) refreshToken(ctx gocom.Context) error {

	req := dtos.RefreshTokenReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetAuthSvc().RefreshToken(req.RefreshToken)

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

//---------------------------------------

var authController *AuthController
var authControllerOnce sync.Once

func GetAuthController() *AuthController {

	if authController == nil {

		authControllerOnce.Do(func() {
			authController = &AuthController{}
		})
	}

	return authController
}
