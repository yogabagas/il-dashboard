package dtos

import "time"

type BillingTransactionReq struct {
	Type          string  `json:"type"` // topup, usage, payment, refund, adjustment
	ClientId      string  `json:"clientId"`
	Amount        float64 `json:"amount"`
	ReferenceId   string  `json:"referenceId,omitempty"`
	ReferenceType string  `json:"referenceType,omitempty"`
	Description   string  `json:"description"`
}

type BillingTransaction struct {
	ID            string    `json:"id"`
	ClientId      string    `json:"clientId"`
	Type          string    `json:"type"`
	Amount        float64   `json:"amount"`
	BalanceBefore float64   `json:"balanceBefore"`
	BalanceAfter  float64   `json:"balanceAfter"`
	ReferenceId   string    `json:"referenceId,omitempty"`
	ReferenceType string    `json:"referenceType,omitempty"`
	BillingName   string    `json:"billingName,omitempty"`
	Description   string    `json:"description,omitempty"`
	CreatedBy     string    `json:"createdBy,omitempty"`
	CreatedAt     time.Time `json:"createdAt,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt,omitempty"`
}

type ClientBillingInfo struct {
	ClientId         string  `json:"clientId"`
	ClientName       string  `json:"clientName"`
	BillingType      string  `json:"billingType"`
	Balance          float64 `json:"balance"`
	CreditLimit      float64 `json:"creditLimit,omitempty"`
	TotalUsage       float64 `json:"totalUsage"`
	Status           string  `json:"status"`
	CanSend          bool    `json:"canSend"`
	RemainingBalance float64 `json:"remainingBalance,omitempty"`
}

type UsageSummary struct {
	ClientId      string  `json:"clientId"`
	Period        string  `json:"period"`
	TotalMessages int     `json:"totalMessages"`
	TotalCost     float64 `json:"totalCost"`
	WhatsAppCount int     `json:"whatsappCount"`
	WhatsAppCost  float64 `json:"whatsappCost"`
	SmsCount      int     `json:"smsCount"`
	SmsCost       float64 `json:"smsCost"`
	EmailCount    int     `json:"emailCount"`
	EmailCost     float64 `json:"emailCost"`
}
