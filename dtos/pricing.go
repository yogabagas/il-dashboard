package dtos

import "time"

type Pricing struct {
	ID          string    `json:"id"`
	ClientId    *string   `json:"clientId,omitempty"`
	ServiceType string    `json:"serviceType"`
	Category    string    `json:"category"`
	Cost        float64   `json:"cost"`
	Currency    string    `json:"currency"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type PricingReq struct {
	ClientId    *string `json:"clientId"`
	ServiceType string  `json:"serviceType"`
	Category    string  `json:"category"`
	Cost        float64 `json:"cost"`
	Currency    string  `json:"currency"`
	IsActive    bool    `json:"isActive"`
}
