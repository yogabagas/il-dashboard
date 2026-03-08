package outbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ariandi/gocom/config"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/logger"
)

type MetaOutboundImpl struct {
}

type MetaOutboundService interface {
	RegisterPhoneNumberOnboard(ctx context.Context, req dtos.RegisterPhoneNumberOnboardReq) (dtos.RegisterPhoneNumberOnboardResp, error)
	CreatePhoneNumberToWABA(ctx context.Context, req dtos.MetaSenderReq) (dtos.MetaSenderResp, error)
	RequestOTPCode(ctx context.Context, req dtos.RequestOTPCodeReq) (dtos.RequestOTPCodeResp, error)
	VerifyOTPCode(ctx context.Context, req dtos.VerifyOTPCodeReq) (dtos.VerifyOTPCodeResp, error)
	ExchangeToken(ctx context.Context, req dtos.ExchangeTokenReq) (dtos.ExchangeTokenResp, error)
	SubscribeWebhook(ctx context.Context, req dtos.SubscribeWebhookReq) (dtos.SubscribeWebhookResp, error)
}

func NewMetaOutboundService() MetaOutboundService {
	return &MetaOutboundImpl{}
}

func (o *MetaOutboundImpl) RegisterPhoneNumberOnboard(ctx context.Context, req dtos.RegisterPhoneNumberOnboardReq) (dtos.RegisterPhoneNumberOnboardResp, error) {
	logger.InfoWithData("RegisterPhoneNumberOnboard started", map[string]interface{}{
		"component": "MetaOutboundService",
		"function":  "RegisterPhoneNumberOnboard",
		"req":       req,
	})

	baseURL := fmt.Sprintf("%s/%s/register", config.Get(constans.BaseURLMeta), req.PhoneNumberID)

	payload := map[string]interface{}{
		"messaging_product": req.MessagingProduct,
		"pin":               req.Pin,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return dtos.RegisterPhoneNumberOnboardResp{}, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return dtos.RegisterPhoneNumberOnboardResp{}, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+config.Get(constans.WaAuthToken))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.ErrorWithData("Failed to send request", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "RegisterPhoneNumberOnboard",
			"error":     err.Error(),
		})
		return dtos.RegisterPhoneNumberOnboardResp{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorWithData("Failed to read response", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "RegisterPhoneNumberOnboard",
			"error":     err.Error(),
		})
		return dtos.RegisterPhoneNumberOnboardResp{}, err
	}

	var respDto dtos.RegisterPhoneNumberOnboardResp
	err = json.Unmarshal(body, &respDto)
	if err != nil {
		logger.ErrorWithData("Failed to unmarshal response", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "RegisterPhoneNumberOnboard",
			"error":     err.Error(),
		})
		return dtos.RegisterPhoneNumberOnboardResp{}, err
	}

	return respDto, nil
}

func (o *MetaOutboundImpl) CreatePhoneNumberToWABA(ctx context.Context, req dtos.MetaSenderReq) (dtos.MetaSenderResp, error) {
	logger.InfoWithData("CreatePhoneNumberToWABA started", map[string]interface{}{
		"component": "MetaOutboundService",
		"function":  "CreatePhoneNumberToWABA",
		"req":       req,
	})

	baseURL := fmt.Sprintf("%s/%s/phone_numbers", config.Get(constans.BaseURLMeta), req.WabaID)

	payload := dtos.CreatePhoneNumberToWABAReq{
		CountryCode:        req.CountryCode,
		MigratePhoneNumber: req.MigratePhoneNumber,
		PhoneNumber:        req.PhoneNumber,
		PreverifiedID:      req.PreverifiedID,
		VerifiedName:       req.VerifiedName,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.ErrorWithData("Failed to marshal payload", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "CreatePhoneNumberToWABA",
			"error":     err.Error(),
		})
		return dtos.MetaSenderResp{}, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.ErrorWithData("Failed to create request", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "CreatePhoneNumberToWABA",
			"error":     err.Error(),
		})
		return dtos.MetaSenderResp{}, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+config.Get(constans.WaAuthToken))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.ErrorWithData("Failed to send request", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "CreatePhoneNumberToWABA",
			"error":     err.Error(),
		})
		return dtos.MetaSenderResp{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorWithData("Failed to read response", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "CreatePhoneNumberToWABA",
			"error":     err.Error(),
		})
		return dtos.MetaSenderResp{}, err
	}

	var respDto dtos.MetaSenderResp
	err = json.Unmarshal(body, &respDto)
	if err != nil {
		logger.ErrorWithData("Failed to unmarshal response", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "CreatePhoneNumberToWABA",
			"error":     err.Error(),
		})
		return dtos.MetaSenderResp{}, err
	}

	return respDto, nil
}

func (o *MetaOutboundImpl) RequestOTPCode(ctx context.Context, req dtos.RequestOTPCodeReq) (dtos.RequestOTPCodeResp, error) {
	logger.InfoWithData("RequestOTPCode started", map[string]interface{}{
		"component": "MetaOutboundService",
		"function":  "RequestOTPCode",
		"req":       req,
	})

	baseURL := fmt.Sprintf("%s/%s/request_code", config.Get(constans.BaseURLMeta), req.PhoneNumberId)

	payload := dtos.RequestOTPCodeReq{
		CodeMethode: req.CodeMethode,
		Language:    req.Language,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.ErrorWithData("Failed to marshal payload", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "RequestOTPCode",
			"error":     err.Error(),
		})
		return dtos.RequestOTPCodeResp{}, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.ErrorWithData("Failed to create request", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "RequestOTPCode",
			"error":     err.Error(),
		})
		return dtos.RequestOTPCodeResp{}, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+config.Get(constans.WaAuthToken))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.ErrorWithData("Failed to send request", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "RequestOTPCode",
			"error":     err.Error(),
		})
		return dtos.RequestOTPCodeResp{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorWithData("Failed to read response", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "RequestOTPCode",
			"error":     err.Error(),
		})
		return dtos.RequestOTPCodeResp{}, err
	}

	var respDto dtos.RequestOTPCodeResp
	err = json.Unmarshal(body, &respDto)
	if err != nil {
		logger.ErrorWithData("Failed to unmarshal response", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "RequestOTPCode",
			"error":     err.Error(),
		})
		return dtos.RequestOTPCodeResp{}, err
	}

	return respDto, nil
}

func (o *MetaOutboundImpl) VerifyOTPCode(ctx context.Context, req dtos.VerifyOTPCodeReq) (dtos.VerifyOTPCodeResp, error) {
	logger.InfoWithData("VerifyOTPCode started", map[string]interface{}{
		"component": "MetaOutboundService",
		"function":  "VerifyOTPCode",
		"req":       req,
	})

	baseURL := fmt.Sprintf("%s/%s/verify_code", config.Get(constans.BaseURLMeta), req.PhoneNumberId)

	payload := dtos.VerifyOTPCodeReq{
		Code: req.Code,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.ErrorWithData("Failed to marshal payload", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "VerifyOTPCode",
			"error":     err.Error(),
		})
		return dtos.VerifyOTPCodeResp{}, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.ErrorWithData("Failed to create request", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "VerifyOTPCode",
			"error":     err.Error(),
		})
		return dtos.VerifyOTPCodeResp{}, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+config.Get(constans.WaAuthToken))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.ErrorWithData("Failed to send request", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "VerifyOTPCode",
			"error":     err.Error(),
		})
		return dtos.VerifyOTPCodeResp{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorWithData("Failed to read response", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "VerifyOTPCode",
			"error":     err.Error(),
		})
		return dtos.VerifyOTPCodeResp{}, err
	}

	var respDto dtos.VerifyOTPCodeResp
	err = json.Unmarshal(body, &respDto)
	if err != nil {
		logger.ErrorWithData("Failed to unmarshal response", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "VerifyOTPCode",
			"error":     err.Error(),
		})
		return dtos.VerifyOTPCodeResp{}, err
	}

	return respDto, nil
}

func (o *MetaOutboundImpl) ExchangeToken(ctx context.Context, req dtos.ExchangeTokenReq) (dtos.ExchangeTokenResp, error) {
	logger.InfoWithData("ExchangeToken started", map[string]interface{}{
		"component": "MetaOutboundService",
		"function":  "ExchangeToken",
		"req":       req,
	})

	baseURL := fmt.Sprintf("%s/oauth/access_token", config.Get(constans.BaseURLMeta))

	payload := map[string]interface{}{
		"client_id":     config.Get(constans.WaClientId),
		"client_secret": config.Get(constans.WaClientSecret),
		"code":          req.Code,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.ErrorWithData("Failed to marshal payload", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "ExchangeToken",
			"error":     err.Error(),
		})
		return dtos.ExchangeTokenResp{}, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.ErrorWithData("Failed to create request", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "ExchangeToken",
			"error":     err.Error(),
		})
		return dtos.ExchangeTokenResp{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.ErrorWithData("Failed to send request", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "ExchangeToken",
			"error":     err.Error(),
		})
		return dtos.ExchangeTokenResp{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorWithData("Failed to read response", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "ExchangeToken",
			"error":     err.Error(),
		})
		return dtos.ExchangeTokenResp{}, err
	}

	logger.InfoWithData("ExchangeToken response", map[string]interface{}{
		"component": "MetaOutboundService",
		"function":  "ExchangeToken",
		"body":      string(body),
	})

	var respDto dtos.ExchangeTokenResp
	err = json.Unmarshal(body, &respDto)
	if err != nil {
		logger.ErrorWithData("Failed to unmarshal response", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "ExchangeToken",
			"error":     err.Error(),
		})
		return dtos.ExchangeTokenResp{}, err
	}

	logger.InfoWithData("ExchangeToken successful", map[string]interface{}{
		"component": "MetaOutboundService",
		"function":  "ExchangeToken",
		"resp":      respDto,
	})

	return respDto, nil
}

func (o *MetaOutboundImpl) SubscribeWebhook(ctx context.Context, req dtos.SubscribeWebhookReq) (dtos.SubscribeWebhookResp, error) {
	logger.InfoWithData("SubscribeWebhook started", map[string]interface{}{
		"component": "MetaOutboundService",
		"function":  "SubscribeWebhook",
		"req":       req,
	})

	url := config.Get(constans.BaseURLMeta)
	lastSlash := strings.LastIndex(url, "/")

	baseURL := url[:lastSlash]   // https://graph.facebook.com
	version := url[lastSlash+1:] // v21.0

	uri := fmt.Sprintf("%s/%s/%s/subscribed_apps", baseURL, version, req.WabaId)

	payload := map[string]interface{}{
		"name": "whatsapp_business_account",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.ErrorWithData("Failed to marshal payload", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "SubscribeWebhook",
			"error":     err.Error(),
		})
		return dtos.SubscribeWebhookResp{}, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, uri, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.ErrorWithData("Failed to create request", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "SubscribeWebhook",
			"error":     err.Error(),
		})
		return dtos.SubscribeWebhookResp{}, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+req.BusinessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.ErrorWithData("Failed to send request", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "SubscribeWebhook",
			"error":     err.Error(),
		})
		return dtos.SubscribeWebhookResp{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorWithData("Failed to read response", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "SubscribeWebhook",
			"error":     err.Error(),
		})
		return dtos.SubscribeWebhookResp{}, err
	}

	var respDto dtos.SubscribeWebhookResp
	err = json.Unmarshal(body, &respDto)
	if err != nil {
		logger.ErrorWithData("Failed to unmarshal response", map[string]interface{}{
			"component": "MetaOutboundService",
			"function":  "SubscribeWebhook",
			"error":     err.Error(),
		})
		return dtos.SubscribeWebhookResp{}, err
	}

	logger.InfoWithData("SubscribeWebhook successful", map[string]interface{}{
		"component": "MetaOutboundService",
		"function":  "SubscribeWebhook",
		"resp":      respDto,
	})

	return respDto, nil
}
