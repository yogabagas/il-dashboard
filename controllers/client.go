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

type ClientController struct {
}

func (o *ClientController) Init() {

	//gocom.POST("/client", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.createClient)
	gocom.POST("/api/v1/instant-link/clients", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.createClient)
	gocom.PUT("/api/v1/instant-link/clients/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.updateClient)
	gocom.GET("/api/v1/instant-link/clients/:id", a.Allow(a.Role(a.ADMIN)), o.getClient)
	gocom.GET("/api/v1/instant-link/clients", a.Allow(a.Role(a.ADMIN, a.OWNER), a.Access("SEARCH_CLIENT")), o.searchClient)
	gocom.DELETE("/api/v1/instant-link/clients/:id", a.Allow(a.Role(a.ADMIN, a.OWNER), a.Access("DELETE_CLIENT")), o.deleteClient)
	gocom.PUT("/api/v1/instant-link/clients/:id/onboard", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.onboardMeta)
}

func (o *ClientController) createClient(ctx gocom.Context) error {

	req := dtos.ClientReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetClientSvc().Create(req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *ClientController) updateClient(ctx gocom.Context) error {

	req := dtos.ClientUpdateReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetClientSvc().Update(ctx.Param("id"), req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *ClientController) getClient(ctx gocom.Context) error {

	req := dtos.ClientReq{}
	_ = ctx.Bind(&req)

	auth := a.Get(ctx)
	ret, err := services.GetClientSvc().Get(ctx.Param("id"), auth)

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *ClientController) searchClient(ctx gocom.Context) error {

	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPer
	// ... (rest of the code remains the same)
}
```
Note: The above code remains the same as the original code provided, since the changes are only required in the dtos/clients.go file and the migrations/next_migration.sql file.

>>>>>> FILE: dtos/clients.go
```go
package dtos

import "time"

type ClientReq struct {
	OwnerName       string  `json:"ownerName"`
	OwnerPassword   string  `json:"ownerPassword"`
	Name            string  `json:"name"`
	ParentId        string  `json:"parentId"`
	Status          string  `json:"status,omitempty"`
	OwnerEmail      string  `json:"ownerEmail"`
	BillingType     string  `json:"billingType,omitempty"`
	CreditLimit     float64 `json:"creditLimit,omitempty"`
	Balance         float64 `json:"balance,omitempty"`
	TotalUsage      float64 `json:"totalUsage,omitempty"`
	IsAi            bool    `json:"isAi"`
	EnableAi        bool    `json:"enableAi"`
	Greeting        string  `json:"greeting,omitempty"`
	GreetingTrigger string  `json:"greetingTrigger,omitempty"`
	WabaId          string  `json:"wabaId,omitempty"`
	WaMetaData      string  `json:"waMetaData"`
	WaAuthCode      string  `json:"waAuthCode"`
	Sender          struct {
		Name       string `json:"name"`
		Identifier string `json:"identifier"`
		Type       string `json:"type"`
	} `json:"sender"`
}

type ClientUpdateReq struct {
	Name            string  `json:"name"`
	ParentId        string  `json:"parentId"`
	Status          string  `json:"status,omitempty"`
	OwnerEmail      string  `json:"ownerEmail"`
	BillingType     string  `json:"billingType,omitempty"`
	CreditLimit     float64 `json:"creditLimit,omitempty"`
	Balance         float64 `json:"balance,omitempty"`
	TotalUsage      float64 `json:"totalUsage,omitempty"`
	IsAi            bool    `json:"isAi"`
	EnableAi        bool    `json:"enableAi"`
	Greeting        string  `json:"greeting,omitempty"`
	GreetingTrigger string  `json:"greetingTrigger,omitempty"`
	WabaId          string  `json:"wabaId,omitempty"`
	WaMetaData      string  `json:"waMetaData"`
	WaAuthCode      string  `json:"waAuthCode"`
	MetaCode        string  `json:"metaCode"`
}

type Client struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	ParentId        string    `json:"parentId"`
	Status          string    `json:"status,omitempty"`
	OwnerEmail      string    `json:"ownerEmail"`
	BillingType     string    `json:"billingType,omitempty"`
	CreditLimit     float64   `json:"creditLimit,omitempty"`
	Balance         float64   `json:"balance,omitempty"`
	TotalUsage      float64   `json:"totalUsage,omitempty"`
	IsAi            bool      `json:"isAi"`
	EnableAi        bool      `json:"enableAi"`
	Greeting        string    `json:"greeting,omitempty"`
	GreetingTrigger string    `json:"greetingTrigger,omitempty"`
	WabaId          string    `json:"wabaId,omitempty"`
	WaMetaData      string    `json:"waMetaData"`
	WaAuthCode      string    `json:"waAuthCode"`
	Sender          struct {
		Name       string `json:"name"`
		Identifier string `json:"identifier"`
		Type       string `json:"type"`
	} `json:"sender"`
}