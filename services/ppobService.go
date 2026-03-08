package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/config"
	"github.com/oklog/ulid/v2"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/utils"
)

type PPOBSvc interface {
	Inquiry(billID string) (*dtos.PPOBInquiryResp, *gocom.CodedError)
	Payment(inquiryResp *dtos.PPOBInquiryResp, bankCode *string) (*dtos.PPOBPaymentResp, *gocom.CodedError)
	Advice(txID, refID string) (*dtos.PPOBAdviceResp, *gocom.CodedError)
}

type PPOBSvcImpl struct{}

var ppobService PPOBSvc
var oncePPOBService sync.Once

const (
	PPOBAccessTokenTTL = 50 * time.Minute
)

func GetPPOBSvc() PPOBSvc {
	oncePPOBService.Do(func() {
		ppobService = &PPOBSvcImpl{}
	})
	return ppobService
}

// generateMerchantToken generates SHA256 hash for merchant token
// Formula: SHA256(trx_date + bill_id + prod_code + ref_id + secretCode)
func (o *PPOBSvcImpl) generateMerchantToken(trxDate, billID, prodCode, refID, secretCode string) string {
	concat := trxDate + billID + prodCode + refID + secretCode
	hash := sha256.Sum256([]byte(concat))
	return hex.EncodeToString(hash[:])
}

// getAccessToken retrieves access token from cache or login
func (o *PPOBSvcImpl) getAccessToken() (string, *gocom.CodedError) {
	// Try to get from cache first
	cacheKey := utils.GetCacheKey(constans.PPOBAccessTokenCache)
	cachedToken := gocom.KeyVal().Get(cacheKey)

	if cachedToken != "" {
		customLogger.InfoWithData("Using cached access token", map[string]interface{}{
			"component": "PPOBService",
			"function":  "getAccessToken",
			"cached":    true,
		})
		return cachedToken, nil
	}

	// Cache miss - need to login
	customLogger.InfoWithData("Access token not in cache, logging in", map[string]interface{}{
		"component": "PPOBService",
		"function":  "getAccessToken",
		"action":    "login",
	})

	authURL := config.Get(constans.PPOBAuthURL, "http://148.230.97.174:9301/auth/login")
	email := config.Get(constans.PPOBLoginEmail, "")
	password := config.Get(constans.PPOBLoginPassword, "")

	customLogger.DebugWithData("PPOB login configuration", map[string]interface{}{
		"component": "PPOBService",
		"function":  "getAccessToken",
		"auth_url":  authURL,
		"email":     email,
	})

	if email == "" || password == "" {
		customLogger.ErrorWithData("PPOB login credentials not configured", map[string]interface{}{
			"component":      "PPOBService",
			"function":       "getAccessToken",
			"email_empty":    email == "",
			"password_empty": password == "",
		})
		return "", &gocom.CodedError{
			Code:    500,
			Message: "PPOB credentials not configured",
		}
	}

	loginReq := dtos.PPOBLoginReq{
		Email:    email,
		Password: password,
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		customLogger.ErrorWithData("Failed to marshal login request", map[string]interface{}{
			"component": "PPOBService",
			"function":  "getAccessToken",
			"error":     err.Error(),
		})
		return "", &gocom.CodedError{
			Code:    500,
			Message: "Failed to prepare login request",
		}
	}

	// Log request (without password for security)
	customLogger.InfoWithData("Sending PPOB login request", map[string]interface{}{
		"component": "PPOBService",
		"function":  "getAccessToken",
		"url":       authURL,
		"email":     email,
	})

	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(jsonData))
	if err != nil {
		customLogger.ErrorWithData("Failed to create login request", map[string]interface{}{
			"component": "PPOBService",
			"function":  "getAccessToken",
			"error":     err.Error(),
			"url":       authURL,
		})
		return "", &gocom.CodedError{
			Code:    500,
			Message: "Failed to create login request",
		}
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		customLogger.ErrorWithData("Login request failed", map[string]interface{}{
			"component": "PPOBService",
			"function":  "getAccessToken",
			"error":     err.Error(),
			"url":       authURL,
		})
		return "", &gocom.CodedError{
			Code:    500,
			Message: "Login request failed",
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		customLogger.ErrorWithData("Failed to read login response", map[string]interface{}{
			"component": "PPOBService",
			"function":  "getAccessToken",
			"error":     err.Error(),
		})
		return "", &gocom.CodedError{
			Code:    500,
			Message: "Failed to read login response",
		}
	}

	customLogger.InfoWithData("PPOB login response received", map[string]interface{}{
		"component":   "PPOBService",
		"function":    "getAccessToken",
		"status_code": resp.StatusCode,
		"body_length": len(body),
	})

	if resp.StatusCode != http.StatusOK {
		customLogger.ErrorWithData("Login failed", map[string]interface{}{
			"component":   "PPOBService",
			"function":    "getAccessToken",
			"status_code": resp.StatusCode,
			"body":        string(body),
		})
		return "", &gocom.CodedError{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("Login failed: %s", string(body)),
		}
	}

	var loginResp dtos.PPOBLoginResp
	if err := json.Unmarshal(body, &loginResp); err != nil {
		customLogger.ErrorWithData("Failed to parse login response", map[string]interface{}{
			"component": "PPOBService",
			"function":  "getAccessToken",
			"error":     err.Error(),
			"body":      string(body),
		})
		return "", &gocom.CodedError{
			Code:    500,
			Message: "Failed to parse login response",
		}
	}

	customLogger.InfoWithData("Parsed PPOB login response", map[string]interface{}{
		"component":         "PPOBService",
		"function":          "getAccessToken",
		"code":              loginResp.Code,
		"message":           loginResp.Message,
		"token_length":      len(loginResp.Data.AccessToken),
	})

	if loginResp.Code != 0 || loginResp.Data.AccessToken == "" {
		customLogger.ErrorWithData("Login response invalid", map[string]interface{}{
			"component":   "PPOBService",
			"function":    "getAccessToken",
			"code":        loginResp.Code,
			"message":     loginResp.Message,
			"token_empty": loginResp.Data.AccessToken == "",
		})
		return "", &gocom.CodedError{
			Code:    500,
			Message: "Login response invalid",
		}
	}

	accessToken := loginResp.Data.AccessToken

	// Cache the access token for 50 minutes
	cacheErr := gocom.KeyVal().Set(cacheKey, accessToken, PPOBAccessTokenTTL)
	if cacheErr != nil {
		customLogger.WarnWithData("Failed to cache access token", map[string]interface{}{
			"component": "PPOBService",
			"function":  "getAccessToken",
			"error":     cacheErr.Error(),
		})
	} else {
		customLogger.InfoWithData("Access token cached successfully", map[string]interface{}{
			"component":  "PPOBService",
			"function":   "getAccessToken",
			"cache_key":  cacheKey,
			"ttl":        "50min",
		})
	}

	return accessToken, nil
}

// Inquiry performs bill inquiry
func (o *PPOBSvcImpl) Inquiry(billID string) (*dtos.PPOBInquiryResp, *gocom.CodedError) {
	customLogger.InfoWithData("PPOB Inquiry started", map[string]interface{}{
		"component": "PPOBService",
		"function":  "Inquiry",
		"bill_id":   billID,
	})

	// Get access token
	accessToken, codedErr := o.getAccessToken()
	if codedErr != nil {
		return nil, codedErr
	}

	// Prepare request parameters
	trxDate := time.Now().Format("20060102150405") // YYYYMMDDHHmmss
	refID := "ref_" + ulid.Make().String()
	productCode := config.Get(constans.PPOBProductCode, "")
	secretCode := config.Get(constans.PPOBSecretCode, "")

	if secretCode == "" {
		customLogger.ErrorWithData("Secret code not configured", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Inquiry",
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "PPOB secret code not configured",
		}
	}

	// Generate merchant token
	merchantToken := o.generateMerchantToken(trxDate, billID, productCode, refID, secretCode)
	customLogger.DebugWithData("Generated merchant token", map[string]interface{}{
		"component":      "PPOBService",
		"function":       "Inquiry",
		"token_preview":  merchantToken[:10] + "...",
		"trx_date":       trxDate,
		"bill_id":        billID,
		"product_code":   productCode,
		"ref_id":         refID,
	})

	// Build inquiry request
	inquiryReq := dtos.PPOBInquiryReq{
		TrxDate:       trxDate,
		ProductCode:   productCode,
		BillID:        billID,
		RefID:         refID,
		MerchantToken: merchantToken,
		User:          config.Get(constans.PPOBMerchantUser, "testva001"),
		MerchantCode:  config.Get(constans.PPOBMerchantCode, "TESTVA001"),
		Category:      config.Get(constans.PPOBCategory, "PDAM0001"),
		Method:        config.Get(constans.PPOBMethod, "PDAMTKR001"),
	}

	customLogger.InfoWithData("PPOB inquiry request params", map[string]interface{}{
		"component":     "PPOBService",
		"function":      "Inquiry",
		"user":          inquiryReq.User,
		"merchant_code": inquiryReq.MerchantCode,
		"category":      inquiryReq.Category,
		"method":        inquiryReq.Method,
	})

	jsonData, err := json.Marshal(inquiryReq)
	if err != nil {
		customLogger.ErrorWithData("Failed to marshal request", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Inquiry",
			"error":     err.Error(),
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Failed to prepare inquiry request",
		}
	}

	baseURL := config.Get(constans.PPOBBaseURL, "http://148.230.97.174:9302/api/v1")
	reqURL := baseURL + "/inq"

	customLogger.InfoWithData("Sending PPOB inquiry request", map[string]interface{}{
		"component":    "PPOBService",
		"function":     "Inquiry",
		"url":          reqURL,
		"payload_size": len(jsonData),
	})

	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		customLogger.ErrorWithData("Failed to create request", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Inquiry",
			"error":     err.Error(),
			"url":       reqURL,
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Failed to create inquiry request",
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		customLogger.ErrorWithData("Inquiry request failed", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Inquiry",
			"error":     err.Error(),
			"url":       reqURL,
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Inquiry request failed",
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		customLogger.ErrorWithData("Failed to read response", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Inquiry",
			"error":     err.Error(),
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Failed to read inquiry response",
		}
	}

	customLogger.InfoWithData("PPOB inquiry response received", map[string]interface{}{
		"component":   "PPOBService",
		"function":    "Inquiry",
		"status_code": resp.StatusCode,
		"body_length": len(body),
		"headers":     resp.Header,
	})

	// Check if response is not 200 OK
	if resp.StatusCode != http.StatusOK {
		customLogger.ErrorWithData("Non-OK response status", map[string]interface{}{
			"component":   "PPOBService",
			"function":    "Inquiry",
			"status_code": resp.StatusCode,
			"body":        string(body),
		})
		return nil, &gocom.CodedError{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("Inquiry request failed with status %d: %s", resp.StatusCode, string(body)),
		}
	}

	var inquiryResp dtos.PPOBInquiryResp
	if err := json.Unmarshal(body, &inquiryResp); err != nil {
		customLogger.ErrorWithData("Failed to parse response", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Inquiry",
			"error":     err.Error(),
			"body":      string(body),
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Failed to parse inquiry response",
		}
	}

	// Check result code
	if inquiryResp.ResultCD != "0000" {
		customLogger.WarnWithData("Inquiry failed", map[string]interface{}{
			"component":  "PPOBService",
			"function":   "Inquiry",
			"result_code": inquiryResp.ResultCD,
			"result_msg":  inquiryResp.ResultMsg,
		})

		// Include error code in message for better error handling
		errorMessage := fmt.Sprintf("[%s] %s", inquiryResp.ResultCD, inquiryResp.ResultMsg)

		return &inquiryResp, &gocom.CodedError{
			Code:    400,
			Message: errorMessage,
		}
	}

	txID := ""
	if inquiryResp.TxID != nil {
		txID = *inquiryResp.TxID
	}
	amount := float64(0)
	if inquiryResp.Amount != nil {
		amount = *inquiryResp.Amount
	}
	totalAmount := float64(0)
	if inquiryResp.TotalAmount != nil {
		totalAmount = *inquiryResp.TotalAmount
	}

	customLogger.InfoWithData("PPOB Inquiry success", map[string]interface{}{
		"component":    "PPOBService",
		"function":     "Inquiry",
		"tx_id":        txID,
		"amount":       amount,
		"total_amount": totalAmount,
	})

	return &inquiryResp, nil
}

// Payment performs bill payment
func (o *PPOBSvcImpl) Payment(inquiryResp *dtos.PPOBInquiryResp, bankCode *string) (*dtos.PPOBPaymentResp, *gocom.CodedError) {
	txID := ""
	if inquiryResp.TxID != nil {
		txID = *inquiryResp.TxID
	}
	billID := ""
	if inquiryResp.BillID != nil {
		billID = *inquiryResp.BillID
	}
	refID := ""
	if inquiryResp.RefID != nil {
		refID = *inquiryResp.RefID
	}

	bankCodeStr := "NOT_SET"
	if bankCode != nil {
		bankCodeStr = *bankCode
	}

	amount := float64(0)
	if inquiryResp.Amount != nil {
		amount = *inquiryResp.Amount
	}
	totalAmount := float64(0)
	if inquiryResp.TotalAmount != nil {
		totalAmount = *inquiryResp.TotalAmount
	}

	customLogger.InfoWithData("PPOB Payment process started", map[string]interface{}{
		"component":    "PPOBService",
		"function":     "Payment",
		"tx_id":        txID,
		"bill_id":      billID,
		"ref_id":       refID,
		"bank_code":    bankCodeStr,
		"amount":       amount,
		"total_amount": totalAmount,
	})

	// Get access token
	accessToken, codedErr := o.getAccessToken()
	if codedErr != nil {
		return nil, codedErr
	}

	// Use same parameters from inquiry
	trxDate := time.Now().Format("20060102150405")
	productCode := config.Get(constans.PPOBProductCode, "")
	secretCode := config.Get(constans.PPOBSecretCode, "")

	// Generate new merchant token for payment
	merchantToken := o.generateMerchantToken(trxDate, billID, productCode, refID, secretCode)
	customLogger.DebugWithData("Generated merchant token for payment", map[string]interface{}{
		"component":      "PPOBService",
		"function":       "Payment",
		"token_preview":  merchantToken[:10] + "...",
		"trx_date":       trxDate,
		"bill_id":        billID,
		"product_code":   productCode,
		"ref_id":         refID,
	})

	// Build payment request
	paymentReq := dtos.PPOBPaymentReq{
		TrxDate:       trxDate,
		ProductCode:   productCode,
		BillID:        billID,
		RefID:         refID,
		MerchantToken: merchantToken,
		User:          config.Get(constans.PPOBMerchantUser, "testva001"),
		MerchantCode:  config.Get(constans.PPOBMerchantCode, "TESTVA001"),
		Category:      config.Get(constans.PPOBCategory, "PDAM0001"),
		Method:        config.Get(constans.PPOBMethod, "PDAMTKR001"),
		TxID:          txID,
		Amount:        inquiryResp.Amount,
		Admin:         inquiryResp.Admin,
		TotalAmount:   inquiryResp.TotalAmount,
		BankCode:      bankCode,
	}

	customLogger.InfoWithData("PPOB payment request params", map[string]interface{}{
		"component":     "PPOBService",
		"function":      "Payment",
		"user":          paymentReq.User,
		"merchant_code": paymentReq.MerchantCode,
		"tx_id":         paymentReq.TxID,
		"amount":        amount,
		"total_amount":  totalAmount,
	})

	jsonData, err := json.Marshal(paymentReq)
	if err != nil {
		customLogger.ErrorWithData("Failed to marshal request", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Payment",
			"error":     err.Error(),
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Failed to prepare payment request",
		}
	}

	baseURL := config.Get(constans.PPOBBaseURL, "http://148.230.97.174:9302/api/v1")
	reqURL := baseURL + "/pay"

	customLogger.InfoWithData("Sending PPOB payment request", map[string]interface{}{
		"component":    "PPOBService",
		"function":     "Payment",
		"url":          reqURL,
		"payload_size": len(jsonData),
	})

	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		customLogger.ErrorWithData("Failed to create request", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Payment",
			"error":     err.Error(),
			"url":       reqURL,
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Failed to create payment request",
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		customLogger.ErrorWithData("Payment request failed", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Payment",
			"error":     err.Error(),
			"url":       reqURL,
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Payment request failed",
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		customLogger.ErrorWithData("Failed to read response", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Payment",
			"error":     err.Error(),
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Failed to read payment response",
		}
	}

	customLogger.InfoWithData("PPOB payment response received", map[string]interface{}{
		"component":    "PPOBService",
		"function":     "Payment",
		"status_code":  resp.StatusCode,
		"body_length":  len(body),
	})

	// Check if response is not 200 OK
	if resp.StatusCode != http.StatusOK {
		customLogger.ErrorWithData("Non-OK response status", map[string]interface{}{
			"component":   "PPOBService",
			"function":    "Payment",
			"status_code": resp.StatusCode,
			"body":        string(body),
		})
		return nil, &gocom.CodedError{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("Payment request failed with status %d: %s", resp.StatusCode, string(body)),
		}
	}

	var paymentResp dtos.PPOBPaymentResp
	if err := json.Unmarshal(body, &paymentResp); err != nil {
		customLogger.ErrorWithData("Failed to parse response", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Payment",
			"error":     err.Error(),
			"body":      string(body),
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Failed to parse payment response",
		}
	}

	status := "nil"
	if paymentResp.Status != nil {
		status = *paymentResp.Status
	}
	
	customLogger.InfoWithData("Parsed PPOB payment response", map[string]interface{}{
		"component": "PPOBService",
		"function":  "Payment",
		"va_no":     paymentResp.VaNo,
		"bank_code": paymentResp.BankCode,
		"status":    status,
	})

	// Check result code
	if paymentResp.ResultCD != "0000" {
		customLogger.WarnWithData("Payment failed", map[string]interface{}{
			"component":   "PPOBService",
			"function":    "Payment",
			"result_code": paymentResp.ResultCD,
			"result_msg":  paymentResp.ResultMsg,
		})

		return &paymentResp, &gocom.CodedError{
			Code:    400,
			Message: paymentResp.ResultMsg,
		}
	}

	// Payment SUCCESS
	respTxID := ""
	if paymentResp.TxID != nil {
		respTxID = *paymentResp.TxID
	}
	status = ""
	if paymentResp.Status != nil {
		status = *paymentResp.Status
	}
	respBillID := ""
	if paymentResp.BillID != nil {
		respBillID = *paymentResp.BillID
	}

	paymentData := map[string]interface{}{
		"component": "PPOBService",
		"function":  "Payment",
		"tx_id":     respTxID,
		"status":    status,
		"bill_id":   respBillID,
	}

	if paymentResp.Amount != nil {
		paymentData["amount"] = *paymentResp.Amount
	}
	if paymentResp.Admin != nil {
		paymentData["admin"] = *paymentResp.Admin
	}
	if paymentResp.TotalAmount != nil {
		paymentData["total_amount"] = *paymentResp.TotalAmount
	}

	customLogger.InfoWithData("PPOB Payment SUCCESS", paymentData)

	// Log Virtual Account info
	if paymentResp.Detail != nil && paymentResp.Detail.VaNo != "" {
		vaData := map[string]interface{}{
			"component":   "PPOBService",
			"function":    "Payment",
			"va_number":   paymentResp.Detail.VaNo,
			"bank_code":   paymentResp.Detail.BankCode,
			"valid_until": paymentResp.Detail.ValidUntil,
		}
		if paymentResp.Detail.CustomerName != nil {
			vaData["customer_name"] = *paymentResp.Detail.CustomerName
		}
		if paymentResp.Detail.Period != nil {
			vaData["period"] = *paymentResp.Detail.Period
		}
		if paymentResp.QrCode != "" {
			vaData["qr_code"] = paymentResp.QrCode
		}
		customLogger.InfoWithData("Virtual Account created", vaData)
	} else {
		customLogger.WarnWithData("No Virtual Account info in response", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Payment",
		})
	}

	customLogger.InfoWithData("PPOB Payment process complete", map[string]interface{}{
		"component": "PPOBService",
		"function":  "Payment",
		"status":    "success",
	})

	return &paymentResp, nil
}

// Advice checks transaction status
func (o *PPOBSvcImpl) Advice(txID, refID string) (*dtos.PPOBAdviceResp, *gocom.CodedError) {
	customLogger.InfoWithData("PPOB Advice started", map[string]interface{}{
		"component": "PPOBService",
		"function":  "Advice",
		"tx_id":     txID,
		"ref_id":    refID,
	})

	// Get access token
	accessToken, codedErr := o.getAccessToken()
	if codedErr != nil {
		return nil, codedErr
	}

	// Build advice request
	adviceReq := dtos.PPOBAdviceReq{
		TxID:  txID,
		RefID: refID,
	}

	jsonData, err := json.Marshal(adviceReq)
	if err != nil {
		customLogger.ErrorWithData("Failed to marshal request", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Advice",
			"error":     err.Error(),
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Failed to prepare advice request",
		}
	}

	baseURL := config.Get(constans.PPOBBaseURL, "http://148.230.97.174:9302/api/v1")
	reqURL := baseURL + "/adv"

	customLogger.InfoWithData("Sending PPOB advice request", map[string]interface{}{
		"component":    "PPOBService",
		"function":     "Advice",
		"url":          reqURL,
		"payload_size": len(jsonData),
	})

	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		customLogger.ErrorWithData("Failed to create request", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Advice",
			"error":     err.Error(),
			"url":       reqURL,
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Failed to create advice request",
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		customLogger.ErrorWithData("Advice request failed", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Advice",
			"error":     err.Error(),
			"url":       reqURL,
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Advice request failed",
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		customLogger.ErrorWithData("Failed to read response", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Advice",
			"error":     err.Error(),
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Failed to read advice response",
		}
	}

	customLogger.InfoWithData("PPOB advice response received", map[string]interface{}{
		"component":   "PPOBService",
		"function":    "Advice",
		"status_code": resp.StatusCode,
		"body_length": len(body),
	})

	// Check if response is not 200 OK
	if resp.StatusCode != http.StatusOK {
		customLogger.ErrorWithData("Non-OK response status", map[string]interface{}{
			"component":   "PPOBService",
			"function":    "Advice",
			"status_code": resp.StatusCode,
			"body":        string(body),
		})
		return nil, &gocom.CodedError{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("Advice request failed with status %d: %s", resp.StatusCode, string(body)),
		}
	}

	var adviceResp dtos.PPOBAdviceResp
	if err := json.Unmarshal(body, &adviceResp); err != nil {
		customLogger.ErrorWithData("Failed to parse response", map[string]interface{}{
			"component": "PPOBService",
			"function":  "Advice",
			"error":     err.Error(),
			"body":      string(body),
		})
		return nil, &gocom.CodedError{
			Code:    500,
			Message: "Failed to parse advice response",
		}
	}

	// Check result code
	if adviceResp.ResultCD != "0000" {
		customLogger.WarnWithData("Advice failed", map[string]interface{}{
			"component":   "PPOBService",
			"function":    "Advice",
			"result_code": adviceResp.ResultCD,
			"result_msg":  adviceResp.ResultMsg,
		})
		return &adviceResp, &gocom.CodedError{
			Code:    400,
			Message: adviceResp.ResultMsg,
		}
	}

	respTxID := ""
	if adviceResp.TxID != nil {
		respTxID = *adviceResp.TxID
	}
	status := ""
	if adviceResp.Status != nil {
		status = *adviceResp.Status
	}

	customLogger.InfoWithData("PPOB Advice success", map[string]interface{}{
		"component": "PPOBService",
		"function":  "Advice",
		"tx_id":     respTxID,
		"status":    status,
	})

	return &adviceResp, nil
}
