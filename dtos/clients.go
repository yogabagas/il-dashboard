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
	OwnerName       string    `json:"ownerName,omitempty"`
	ParentName      string    `json:"parentName,omitempty"`
	Children        []Client  `json:"children"`
	Roles           []Role    `json:"roles,omitempty"`
	BillingType     string    `json:"billingType,omitempty"`
	CreditLimit     float64   `json:"creditLimit,omitempty"`
	Balance         float64   `json:"balance,omitempty"`
	TotalUsage      float64   `json:"totalUsage,omitempty"`
	CreatedAt       time.Time `json:"createdAt,omitempty"`
	IsAi            bool      `json:"isAi"`
	Greeting        string    `json:"greeting,omitempty"`
	GreetingTrigger string    `json:"greetingTrigger,omitempty"`
	WabaId          string    `json:"wabaId,omitempty"`
	WaMetaData      string    `json:"waMetaData"`
	WaAuthCode      string    `json:"waAuthCode"`
	MetaCode        string    `json:"metaCode"`
}

type ClientMetaOnboardReq struct {
	ClientId string `json:"clientId"`
}

type ClientMetaOnboardResp struct {
	Success bool `json:"success"`
}
