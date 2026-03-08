package dtos

// PPOB Login Request/Response
type PPOBLoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type PPOBLoginResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		ID           string `json:"id"`
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	} `json:"data"`
}

// PPOB Inquiry Request
type PPOBInquiryReq struct {
	TrxDate       string `json:"trx_date"`       // YYYYMMDDHHmmss
	ProductCode   string `json:"product_code"`   // PDAMKABTGR
	BillID        string `json:"bill_id"`        // Nomor pelanggan
	RefID         string `json:"ref_id"`         // Unique reference
	MerchantToken string `json:"merchant_token"` // SHA256 hash
	User          string `json:"user"`           // Merchant user
	MerchantCode  string `json:"merchant_code"`  // Merchant code
	Category      string `json:"category"`       // PDAM0001
	Method        string `json:"method"`         // PDAMTKR001
}

// PPOB Inquiry Response
type PPOBInquiryResp struct {
	ResultCD        string                 `json:"result_cd"`
	ResultMsg       string                 `json:"result_msg"`
	TxID            *string                `json:"tx_id,omitempty"`
	TxDate          *string                `json:"tx_date,omitempty"`
	User            *string                `json:"user,omitempty"`
	BillID          *string                `json:"bill_id,omitempty"`
	ProductCode     *string                `json:"product_code,omitempty"`
	Category        *string                `json:"category,omitempty"`
	MerchantCode    *string                `json:"merchant_code,omitempty"`
	Method          *string                `json:"method,omitempty"`
	Amount          *float64               `json:"amount,omitempty"`
	Admin           *float64               `json:"admin,omitempty"`
	TotalAmount     *float64               `json:"total_amount,omitempty"`
	DeductedBalance *float64               `json:"deducted_balance,omitempty"`
	FirstBalance    *float64               `json:"first_balance,omitempty"`
	LastBalance     *float64               `json:"last_balance,omitempty"`
	Sign            *string                `json:"sign,omitempty"`
	RefID           *string                `json:"ref_id,omitempty"`
	Status          *string                `json:"status,omitempty"`
	Qty             *int                   `json:"qty,omitempty"`
	Detail          *PPOBInquiryRespDetail `json:"detail,omitempty"`
}

type PPOBInquiryRespDetail struct {
	ProductName     string                      `json:"product_name"`
	Period          *string                     `json:"period,omitempty"`
	CustomerName    *string                     `json:"customer_name,omitempty"`
	CustomerAddress *string                     `json:"customer_address,omitempty"`
	Type            *string                     `json:"type,omitempty"`
	FirstStand      *int64                      `json:"first_stand,omitempty"`
	LastStand       *int64                      `json:"last_stand,omitempty"`
	RefProvider     *string                     `json:"ref_provider,omitempty"`
	DetailBilling   *[]PPOBBillingInquiryDetail `json:"detail_billing,omitempty"`
	VaNo            string                      `json:"va_no,omitempty"`
	BankCode        string                      `json:"bank_code,omitempty"`
	ValidUntil      string                      `json:"valid_until,omitempty"`
	QrCode          string                      `json:"qr_code,omitempty"`
}

type PPOBBillingInquiryDetail struct {
	StandMeter      *string  `json:"stand_meter,omitempty"`
	Period          *string  `json:"period,omitempty"`
	BillAmount      *float64 `json:"bill_amount,omitempty"`
	Fine            *float64 `json:"fine,omitempty"`
	AdminMerchant   *float64 `json:"admin_merchant,omitempty"`
	AdminProvider   *float64 `json:"admin_provider,omitempty"`
	Total           *float64 `json:"total,omitempty"`
	AdditionalInfo  *string  `json:"additional_info,omitempty"`
	AdditionalInfo2 *string  `json:"additional_info2,omitempty"`
	Info            *string  `json:"info,omitempty"`
}

// PPOB Payment Request
type PPOBPaymentReq struct {
	TrxDate       string   `json:"trx_date"`
	ProductCode   string   `json:"product_code"`
	BillID        string   `json:"bill_id"`
	RefID         string   `json:"ref_id"`
	MerchantToken string   `json:"merchant_token"`
	User          string   `json:"user"`
	MerchantCode  string   `json:"merchant_code"`
	Category      string   `json:"category"`
	Method        string   `json:"method"`
	TxID          string   `json:"tx_id"`        // From inquiry response
	Amount        *float64 `json:"amount"`       // From inquiry response
	Admin         *float64 `json:"admin"`        // From inquiry response
	TotalAmount   *float64 `json:"total_amount"` // From inquiry response
	BankCode      *string  `json:"bank_code,omitempty"`
}

// PPOB Payment Response
type PPOBPaymentResp struct {
	ResultCD        string                 `json:"result_cd"`
	ResultMsg       string                 `json:"result_msg"`
	TxID            *string                `json:"tx_id,omitempty"`
	TxDate          *string                `json:"tx_date,omitempty"`
	User            *string                `json:"user,omitempty"`
	BillID          *string                `json:"bill_id,omitempty"`
	ProductCode     *string                `json:"product_code,omitempty"`
	Category        *string                `json:"category,omitempty"`
	MerchantCode    *string                `json:"merchant_code,omitempty"`
	Method          *string                `json:"method,omitempty"`
	Amount          *float64               `json:"amount,omitempty"`
	Admin           *float64               `json:"admin,omitempty"`
	TotalAmount     *float64               `json:"total_amount,omitempty"`
	DeductedBalance *float64               `json:"deducted_balance,omitempty"`
	FirstBalance    *float64               `json:"first_balance,omitempty"`
	LastBalance     *float64               `json:"last_balance,omitempty"`
	Sign            *string                `json:"sign,omitempty"`
	RefID           *string                `json:"ref_id,omitempty"`
	Status          *string                `json:"status,omitempty"`
	Qty             *int                   `json:"qty,omitempty"`
	VaNo            string                 `json:"va_no,omitempty"`
	BankCode        string                 `json:"bank_code,omitempty"`
	ValidUntil      string                 `json:"valid_until,omitempty"`
	QrCode          string                 `json:"qr_code"`
	Detail          *PPOBInquiryRespDetail `json:"detail,omitempty"`
}

// PPOB Advice Request
type PPOBAdviceReq struct {
	TxID  string `json:"tx_id"`
	RefID string `json:"ref_id"`
}

// PPOB Advice Response (same structure as inquiry/payment response)
type PPOBAdviceResp PPOBInquiryResp
