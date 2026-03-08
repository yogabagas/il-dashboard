package controllers

import (
	"sync"

	"github.com/ariandi/gocom"
	a "gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/services"
)

type DashboardController struct {
}

func (o *DashboardController) Init() {
	gocom.GET("/api/v1/instant-link/dashboard/overview", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.getOverview)
}

func (o *DashboardController) getOverview(ctx gocom.Context) error {
	authInfo := a.Get(ctx)
	filterClientId := ctx.Query("clientId") // Optional filter for admin to specify client

	ret, err := services.GetDashboardService().GetOverview(authInfo, filterClientId)

	if err != nil {
		return ctx.SendError(&gocom.CodedError{
			Code:    500,
			Message: err.Error(),
		})
	}

	return ctx.SendResult(ret)
}

//---------------------------------------

var dashboardController *DashboardController
var dashboardControllerOnce sync.Once

func GetDashboardController() *DashboardController {

	if dashboardController == nil {

		dashboardControllerOnce.Do(func() {
			dashboardController = &DashboardController{}
		})
	}

	return dashboardController
}
