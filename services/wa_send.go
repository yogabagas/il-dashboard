package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/config"
	"github.com/ariandi/gocom/pubsub"
	"github.com/oklog/ulid/v2"
	"gitlab.com/anti_metter/switching_common"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/sender"
	"gitlab.com/anti_metter/switching_common/waTemplate"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/utils"
)

type WASendSvc interface {
	SendMessage(req dtos.WASendMessageReq, authInfo auth.AuthInfo) (*dtos.WASendMessageResp, *gocom.CodedError)
	SendBulkMessage(req dtos.WASendBulkMessageReq, authInfo auth.AuthInfo) (*dtos.WASendBulkMessageResp, *gocom.CodedError)
	SendTextMessage(to, text string) (string, *gocom.CodedError)
	SendTextMessageWithLog(to, text, clientId, senderIdentifier string) (string, string, *gocom.CodedError) // Returns: messageId, messageLogId, error
	SendInteractiveButtonMessage(to, bodyText string, buttons []map[string]string, clientId, senderIdentifier string) (string, string, *gocom.CodedError)
	SendInteractiveListMessage(to, bodyText, buttonText string, sections []map[string]interface{}, clientId, senderIdentifier string) (string, string, *gocom.CodedError)
	UploadFile(file *multipart.FileHeader, authInfo auth.AuthInfo) (string, *gocom.CodedError)
	GetFiles(fileType, filename string, authInfo auth.AuthInfo) ([]byte, *gocom.CodedError)
}

type WASendSvcImpl struct{}

// getSenderCredentialsByIdentifier gets WhatsApp credentials (phoneNumberId, accessToken) by sender identifier (business phone number)
func (o *WASendSvcImpl) getSenderCredentialsByIdentifier(identifier string) (phoneNumberId, accessToken string, err *gocom.CodedError) {
	// Lookup sender by identifier (business phone number)
	senderData := sender.GetRepo().GetByIdentifier(identifier)
	if senderData == nil {
		customLogger.WarnWithData("No sender found, using default config", map[string]interface{}{
			"component":  "WASendService",
			"function":   "getSenderCredentialsByIdentifier",
			"identifier": identifier,
			"fallback":   "default_config",
		})
		// Fallback to global config
		phoneNumberId = config.Get(constans.WaPhoneNumberId)
		accessToken = config.Get(constans.WaAuthToken)
	} else {
		// Unmarshal sender config to get WhatsApp credentials
		var senderConfig dtos.WASenderConfig
		unmarshalErr := json.Unmarshal([]byte(senderData.Config), &senderConfig)
		if unmarshalErr != nil {
			customLogger.WarnWithData("Failed to unmarshal sender config, using default", map[string]interface{}{
				"component":  "WASendService",
				"function":   "getSenderCredentialsByIdentifier",
				"identifier": identifier,
				"error":      unmarshalErr.Error(),
				"fallback":   "default_config",
			})
			// Fallback to global config
			phoneNumberId = config.Get(constans.WaPhoneNumberId)
			accessToken = config.Get(constans.WaAuthToken)
		} else {
			phoneNumberId = senderConfig.PhoneNumberId
			accessToken = senderConfig.AccessToken
			customLogger.InfoWithData("Using sender config", map[string]interface{}{
				"component":       "WASendService",
				"function":        "getSenderCredentialsByIdentifier",
				"identifier":      identifier,
				"phone_number_id": phoneNumberId,
			})
		}
	}

	if phoneNumberId == "" {
		return "", "", &gocom.CodedError{Code: 500, Message: "WhatsApp Phone Number ID not configured"}
	}

	if accessToken == "" {
		return "", "", &gocom.CodedError{Code: 500, Message: "WhatsApp Access Token not configured"}
	}

	return phoneNumberId, accessToken, nil
}

// TODO: getSenderCredentialsByClientId would lookup first WhatsApp sender for clientId
// This requires a proper sender.List() method or similar
// For now, functions using clientId will continue to use global config
// Future refactor: update function signatures to accept senderIdentifier (business phone number)
// and use getSenderCredentialsByIdentifier() helper

func (o *WASendSvcImpl) SendMessage(req dtos.WASendMessageReq, authInfo auth.AuthInfo) (*dtos.WASendMessageResp, *gocom.CodedError) {
	customLogger.InfoWithData("SendMessage started", map[string]interface{}{
		"component":   "WASendService",
		"function":    "SendMessage",
		"template_id": req.TemplateId,
		"to":          req.To,
		"client_id":   authInfo.ClientId,
	})

	// Validation
	if req.TemplateId == "" {
		customLogger.WarnWithData("Template ID is required", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendMessage",
			"error":     "missing_template_id",
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Template ID is required"}
	}

	if req.To == "" {
		customLogger.WarnWithData("Recipient phone number is required", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendMessage",
			"error":     "missing_recipient",
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Recipient phone number is required"}
	}

	// Get template
	mdl := waTemplate.GetRepo().GetById(req.TemplateId)
	if mdl == nil {
		customLogger.WarnWithData("Template not found", map[string]interface{}{
			"component":   "WASendService",
			"function":    "SendMessage",
			"template_id": req.TemplateId,
		})
		return nil, switching_common.ERR_NOT_FOUND
	}

	// Check permission
	if authInfo.ClientId != switching_common.PROVIDER_ID && authInfo.ClientId != mdl.ClientId {
		customLogger.WarnWithData("Access denied", map[string]interface{}{
			"component":          "WASendService",
			"function":           "SendMessage",
			"auth_client_id":     authInfo.ClientId,
			"template_client_id": mdl.ClientId,
		})
		return nil, switching_common.ERR_NOT_ALLOWED
	}

	// Check if template is approved
	if mdl.Status != "APPROVED" || mdl.WATemplateId == "" {
		customLogger.WarnWithData("Template not approved", map[string]interface{}{
			"component":   "WASendService",
			"function":    "SendMessage",
			"template_id": req.TemplateId,
			"status":      mdl.Status,
			"has_wa_id":   mdl.WATemplateId != "",
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Template must be approved before sending"}
	}

	// Get template category for pricing
	templateCategory := mdl.Category
	if templateCategory == "" {
		templateCategory = "utility" // Default to utility if not specified
	}

	// Check balance before sending with category-based pricing
	messageCost := GetBillingService().CalculateMessageCostWithCategory("whatsapp", templateCategory, authInfo.ClientId)
	customLogger.InfoWithData("Calculated message cost", map[string]interface{}{
		"component": "WASendService",
		"function":  "SendMessage",
		"category":  templateCategory,
		"cost":      messageCost,
		"client_id": authInfo.ClientId,
	})

	canSend, balanceErr := GetBillingService().CanClientSend(authInfo.ClientId, messageCost)
	if balanceErr != nil {
		customLogger.ErrorWithData("Failed to check balance", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendMessage",
			"error":     balanceErr.Error(),
			"client_id": authInfo.ClientId,
		})
		return nil, &gocom.CodedError{Code: 500, Message: "Failed to check balance"}
	}
	if !canSend {
		customLogger.WarnWithData("Insufficient balance", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendMessage",
			"client_id": authInfo.ClientId,
			"cost":      messageCost,
		})
		return nil, &gocom.CodedError{Code: 402, Message: "Insufficient balance"}
	}

	phoneNumberId := ""
	accessToken := ""
	senderMdl := sender.GetRepo().GetById(req.SenderId)
	if senderMdl == nil {
		customLogger.WarnWithData("Sender not found, using default", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendMessage",
			"sender_id": req.SenderId,
			"fallback":  "default_config",
		})
		phoneNumberId = config.Get(constans.WaPhoneNumberId)
		accessToken = config.Get(constans.WaAuthToken)
	} else {
		// Unmarshal sender config to get WhatsApp credentials
		var senderConfig dtos.WASenderConfig
		err := json.Unmarshal([]byte(senderMdl.Config), &senderConfig)
		if err != nil {
			customLogger.WarnWithData("Failed to unmarshal sender config", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendMessage",
				"sender_id": req.SenderId,
				"error":     err.Error(),
				"fallback":  "default_config",
			})
			phoneNumberId = config.Get(constans.WaPhoneNumberId)
			accessToken = config.Get(constans.WaAuthToken)
		} else {
			phoneNumberId = senderConfig.PhoneNumberId
			accessToken = senderConfig.AccessToken
			customLogger.InfoWithData("Using sender config", map[string]interface{}{
				"component":       "WASendService",
				"function":        "SendMessage",
				"sender_id":       req.SenderId,
				"phone_number_id": phoneNumberId,
			})
		}
	}

	if phoneNumberId == "" {
		customLogger.ErrorWithData("Phone Number ID not configured", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendMessage",
		})
		return nil, &gocom.CodedError{Code: 500, Message: "WhatsApp Phone Number ID not configured"}
	}

	if accessToken == "" {
		customLogger.ErrorWithData("Access Token not configured", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendMessage",
		})
		return nil, &gocom.CodedError{Code: 500, Message: "WhatsApp Access Token not configured"}
	}

	// Prepare template name
	templateName := mdl.WATemplate
	if templateName == "" {
		templateName = mdl.Name
	}

	// Generate message log ID (will be used to track the message)
	logId := ulid.Make().String()
	now := time.Now()

	// Create message log with status "sent" immediately
	messageLog := &dtos.MessageLog{
		ID:             logId,
		ClientId:       authInfo.ClientId,
		SenderId:       req.SenderId,
		CampaignId:     req.CampaignId,
		Type:           "whatsapp",
		Direction:      "outbound",
		RecipientType:  "phone",
		RecipientValue: req.To,
		TemplateId:     req.TemplateId,
		MessageId:      "", // Will be filled by microservice after actual send
		MessageContent: fmt.Sprintf("Template: %s", mdl.Name),
		Status:         "sent", // Set as sent immediately, will be updated by microservice
		Cost:           messageCost,
		SentAt:         now.Format(time.RFC3339),
	}

	// Save message log
	_, logErr := GetMessageLogsService().Create(messageLog, authInfo)
	if logErr != nil {
		customLogger.ErrorWithData("Failed to save message log", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendMessage",
			"error":     logErr.Message,
			"log_id":    logId,
		})
		return nil, &gocom.CodedError{Code: 500, Message: "Failed to create message log"}
	}

	// Publish to pubsub for async processing
	subscribeCacheKey := utils.GetCacheKey(constans.SingleSendMessageSubscribe)
	sendMessageProcess := dtos.SendMessageProcess{
		MessageLogId:   logId,
		MessageType:    "whatsapp",
		PhoneNumberId:  phoneNumberId,
		AccessToken:    accessToken,
		TemplateName:   templateName,
		Language:       mdl.Language,
		To:             req.To,
		Parameters:     req.Parameters,
		Category:       mdl.Category,
		ClientId:       authInfo.ClientId,
		SenderId:       req.SenderId,
		CampaignId:     req.CampaignId,
		TemplateId:     req.TemplateId,
		MessageContent: fmt.Sprintf("Template: %s", mdl.Name),
		Cost:           messageCost,
		UserId:         authInfo.UserId,
	}

	customLogger.InfoWithData("Publishing to pubsub", map[string]interface{}{
		"component": "WASendService",
		"function":  "SendMessage",
		"key":       subscribeCacheKey,
		"log_id":    logId,
		"to":        req.To,
	})
	pubErr := pubsub.Get().Publish(subscribeCacheKey, sendMessageProcess)
	if pubErr != nil {
		customLogger.ErrorWithData("Failed to publish to pubsub", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendMessage",
			"error":     pubErr.Error(),
			"log_id":    logId,
		})
		// Update log to failed status
		_ = GetMessageLogsService().UpdateStatus(logId, "failed", "", "Failed to publish to queue")
		return nil, &gocom.CodedError{Code: 500, Message: "Failed to queue message for sending"}
	}

	customLogger.InfoWithData("Message queued successfully", map[string]interface{}{
		"component": "WASendService",
		"function":  "SendMessage",
		"log_id":    logId,
		"to":        req.To,
		"cost":      messageCost,
	})

	// Return response immediately (actual send will be done by microservice)
	return &dtos.WASendMessageResp{
		MessageId:        logId, // Return logId as temporary identifier
		To:               req.To,
		Status:           "sent",
		MessagingProduct: "whatsapp",
	}, nil
}

func (o *WASendSvcImpl) SendBulkMessage(req dtos.WASendBulkMessageReq, authInfo auth.AuthInfo) (*dtos.WASendBulkMessageResp, *gocom.CodedError) {
	customLogger.InfoWithData("SendBulkMessage started", map[string]interface{}{
		"component":        "WASendService",
		"function":         "SendBulkMessage",
		"template_id":      req.TemplateId,
		"recipients_count": len(req.Recipients),
		"client_id":        authInfo.ClientId,
	})

	// Validation
	if req.TemplateId == "" {
		customLogger.WarnWithData("Template ID is required", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendBulkMessage",
			"error":     "missing_template_id",
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Template ID is required"}
	}

	if len(req.Recipients) == 0 {
		customLogger.WarnWithData("Recipients are required", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendBulkMessage",
			"error":     "missing_recipients",
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Recipients are required"}
	}

	// Limit bulk send to prevent abuse
	if len(req.Recipients) > 100 {
		customLogger.WarnWithData("Too many recipients", map[string]interface{}{
			"component":        "WASendService",
			"function":         "SendBulkMessage",
			"recipients_count": len(req.Recipients),
			"limit":            100,
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Maximum 100 recipients per bulk request"}
	}

	// Get template
	mdl := waTemplate.GetRepo().GetById(req.TemplateId)
	if mdl == nil {
		customLogger.WarnWithData("Template not found", map[string]interface{}{
			"component":   "WASendService",
			"function":    "SendBulkMessage",
			"template_id": req.TemplateId,
		})
		return nil, switching_common.ERR_NOT_FOUND
	}

	// Check permission
	if authInfo.ClientId != switching_common.PROVIDER_ID && authInfo.ClientId != mdl.ClientId {
		customLogger.WarnWithData("Access denied", map[string]interface{}{
			"component":          "WASendService",
			"function":           "SendBulkMessage",
			"auth_client_id":     authInfo.ClientId,
			"template_client_id": mdl.ClientId,
		})
		return nil, switching_common.ERR_NOT_ALLOWED
	}

	// Check if template is approved
	if mdl.Status != "APPROVED" || mdl.WATemplateId == "" {
		customLogger.WarnWithData("Template not approved", map[string]interface{}{
			"component":   "WASendService",
			"function":    "SendBulkMessage",
			"template_id": req.TemplateId,
			"status":      mdl.Status,
			"has_wa_id":   mdl.WATemplateId != "",
		})
		return nil, &gocom.CodedError{Code: 400, Message: "Template must be approved before sending"}
	}

	// Get WhatsApp Phone Number ID and Access Token from config
	phoneNumberId := config.Get(constans.WaPhoneNumberId)
	accessToken := config.Get(constans.WaAuthToken)

	if phoneNumberId == "" {
		customLogger.ErrorWithData("Phone Number ID not configured", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendBulkMessage",
		})
		return nil, &gocom.CodedError{Code: 500, Message: "WhatsApp Phone Number ID not configured"}
	}

	if accessToken == "" {
		customLogger.ErrorWithData("Access Token not configured", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendBulkMessage",
		})
		return nil, &gocom.CodedError{Code: 500, Message: "WhatsApp Access Token not configured"}
	}

	// Send messages in parallel with rate limiting
	results := make([]dtos.WASendMessageResp, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrent requests to prevent rate limit
	semaphore := make(chan struct{}, 10) // Max 10 concurrent requests

	successCount := 0
	failedCount := 0

	for _, recipient := range req.Recipients {
		wg.Add(1)
		go func(r dtos.WASendBulkRecipient) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			templateName := mdl.WATemplate
			if templateName == "" {
				templateName = mdl.Name
			}
			messageId, err := o.sendToWhatsApp(phoneNumberId, accessToken, templateName, mdl.Language, r.To, r.Parameters, mdl.Category)

			// Create message log
			logId := ulid.Make().String()
			now := time.Now()
			messageLog := &dtos.MessageLog{
				ID:             logId,
				ClientId:       authInfo.ClientId,
				SenderId:       req.SenderId,
				Type:           "whatsapp",
				Direction:      "outbound",
				RecipientType:  "phone",
				RecipientValue: r.To,
				TemplateId:     req.TemplateId,
				MessageId:      messageId,
				MessageContent: fmt.Sprintf("Template: %s", mdl.Name),
				Status:         "sent",
				SentAt:         now.Format(time.RFC3339),
			}

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				failedCount++
				results = append(results, dtos.WASendMessageResp{
					MessageId: "",
					To:        r.To,
					Status:    "failed",
				})

				// Update message log for failed message
				messageLog.Status = "failed"
				messageLog.ErrorMessage = err.Message
				messageLog.FailedAt = now.Format(time.RFC3339)
				messageLog.SentAt = ""

				customLogger.WarnWithData("Failed to send to recipient", map[string]interface{}{
					"component": "WASendService",
					"function":  "SendBulkMessage",
					"to":        r.To,
					"error":     err.Message,
				})
			} else {
				successCount++
				results = append(results, dtos.WASendMessageResp{
					MessageId:        messageId,
					To:               r.To,
					Status:           "sent",
					MessagingProduct: "whatsapp",
				})
				customLogger.InfoWithData("Message sent to recipient", map[string]interface{}{
					"component":  "WASendService",
					"function":   "SendBulkMessage",
					"to":         r.To,
					"message_id": messageId,
				})
			}

			// Save message log (outside of mutex to avoid blocking)
			go func(log *dtos.MessageLog) {
				_, logErr := GetMessageLogsService().Create(log, authInfo)
				if logErr != nil {
					customLogger.ErrorWithData("Failed to save message log", map[string]interface{}{
						"component": "WASendService",
						"function":  "SendBulkMessage",
						"to":        r.To,
						"error":     logErr.Message,
					})
				}
			}(messageLog)
		}(recipient)
	}

	wg.Wait()

	customLogger.InfoWithData("SendBulkMessage completed", map[string]interface{}{
		"component":     "WASendService",
		"function":      "SendBulkMessage",
		"total":         len(req.Recipients),
		"success_count": successCount,
		"failed_count":  failedCount,
	})

	return &dtos.WASendBulkMessageResp{
		TotalRequested: len(req.Recipients),
		TotalSuccess:   successCount,
		TotalFailed:    failedCount,
		Results:        results,
	}, nil
}

func (o *WASendSvcImpl) sendToWhatsApp(phoneNumberId, accessToken, templateName, language, to string, parameters []dtos.WAMessageParameter, templateCategory string) (string, *gocom.CodedError) {
	customLogger.InfoWithData("sendToWhatsApp started", map[string]interface{}{
		"component":    "WASendService",
		"function":     "sendToWhatsApp",
		"template":     templateName,
		"language":     language,
		"to":           to,
		"category":     templateCategory,
		"params_count": len(parameters),
	})

	baseURL := fmt.Sprintf(config.Get(constans.BaseURLMeta), phoneNumberId)

	// Build template message payload
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "template",
		"template": map[string]interface{}{
			"name":     templateName,
			"language": map[string]string{"code": language},
		},
	}

	// Add parameters if provided
	if len(parameters) > 0 {
		components := make([]map[string]interface{}, 0)

		// Separate header and body parameters
		headerParams := make([]map[string]interface{}, 0)
		bodyParams := make([]map[string]interface{}, 0)

		for _, p := range parameters {
			param := make(map[string]interface{})
			param["type"] = p.Type

			// Handle different parameter types
			switch p.Type {
			case "text":
				param["text"] = p.Text
				bodyParams = append(bodyParams, param)
			case "image":
				if p.Image != nil {
					imageParam := make(map[string]string)
					if p.Image.Link != "" {
						imageParam["link"] = p.Image.Link
					} else if p.Image.ID != "" {
						imageParam["id"] = p.Image.ID
					}
					param["image"] = imageParam
					headerParams = append(headerParams, param)
				}
			case "video":
				if p.Video != nil {
					videoParam := make(map[string]string)
					if p.Video.Link != "" {
						videoParam["link"] = p.Video.Link
					} else if p.Video.ID != "" {
						videoParam["id"] = p.Video.ID
					}
					param["video"] = videoParam
					headerParams = append(headerParams, param)
				}
			case "document":
				if p.Document != nil {
					docParam := make(map[string]string)
					if p.Document.Link != "" {
						docParam["link"] = p.Document.Link
					} else if p.Document.ID != "" {
						docParam["id"] = p.Document.ID
					}
					param["document"] = docParam
					headerParams = append(headerParams, param)
				}
			}
		}

		// Add header component if there are header parameters
		if len(headerParams) > 0 {
			components = append(components, map[string]interface{}{
				"type":       "header",
				"parameters": headerParams,
			})
		}

		// Add body component if there are body parameters
		if len(bodyParams) > 0 {
			components = append(components, map[string]interface{}{
				"type":       "body",
				"parameters": bodyParams,
			})
		}

		// For AUTHENTICATION templates, add button component with first body parameter
		// This is required for OTP templates with URL buttons that need dynamic parameters
		if strings.ToUpper(templateCategory) == "AUTHENTICATION" && len(bodyParams) > 0 {
			// Add button component with the first body parameter (OTP code)
			buttonComponent := map[string]interface{}{
				"type":     "button",
				"sub_type": "url",
				"index":    "0",
				"parameters": []map[string]interface{}{
					bodyParams[0], // Use the first body parameter (OTP code)
				},
			}
			components = append(components, buttonComponent)
			customLogger.InfoWithData("Added button component for AUTHENTICATION template", map[string]interface{}{
				"component": "WASendService",
				"function":  "sendToWhatsApp",
				"template":  templateName,
				"category":  templateCategory,
				"parameter": bodyParams[0],
			})
		}

		if len(components) > 0 {
			payload["template"].(map[string]interface{})["components"] = components
		}
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		customLogger.ErrorWithData("Unable to marshal payload", map[string]interface{}{
			"component": "WASendService",
			"function":  "sendToWhatsApp",
			"error":     err.Error(),
			"to":        to,
		})
		return "", &gocom.CodedError{Code: 500, Message: "Failed to create message payload"}
	}

	customLogger.InfoWithData("Sending to WhatsApp API", map[string]interface{}{
		"component":    "WASendService",
		"function":     "sendToWhatsApp",
		"url":          baseURL,
		"to":           to,
		"template":     templateName,
		"payload_size": len(payloadBytes),
	})

	// Send to WhatsApp API
	httpReq, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		customLogger.ErrorWithData("Unable to create HTTP request", map[string]interface{}{
			"component": "WASendService",
			"function":  "sendToWhatsApp",
			"error":     err.Error(),
			"to":        to,
		})
		return "", &gocom.CodedError{Code: 500, Message: "Failed to create request"}
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		customLogger.ErrorWithData("HTTP request failed", map[string]interface{}{
			"component": "WASendService",
			"function":  "sendToWhatsApp",
			"error":     err.Error(),
			"to":        to,
		})
		return "", &gocom.CodedError{Code: 500, Message: "Failed to send message to WhatsApp"}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		customLogger.ErrorWithData("Unable to read response body", map[string]interface{}{
			"component": "WASendService",
			"function":  "sendToWhatsApp",
			"error":     err.Error(),
			"to":        to,
		})
		return "", &gocom.CodedError{Code: 500, Message: "Failed to read response"}
	}

	customLogger.InfoWithData("WhatsApp API response received", map[string]interface{}{
		"component":   "WASendService",
		"function":    "sendToWhatsApp",
		"status_code": resp.StatusCode,
		"to":          to,
		"body_size":   len(body),
	})

	if resp.StatusCode != 200 {
		customLogger.ErrorWithData("WhatsApp API error", map[string]interface{}{
			"component":   "WASendService",
			"function":    "sendToWhatsApp",
			"status_code": resp.StatusCode,
			"body":        string(body),
			"to":          to,
		})
		return "", &gocom.CodedError{Code: resp.StatusCode, Message: "WhatsApp API error: " + string(body)}
	}

	var waResp dtos.WAApiSendResponse
	err = json.Unmarshal(body, &waResp)
	if err != nil {
		customLogger.ErrorWithData("Unable to parse WhatsApp response", map[string]interface{}{
			"component": "WASendService",
			"function":  "sendToWhatsApp",
			"error":     err.Error(),
			"body":      string(body),
			"to":        to,
		})
		return "", &gocom.CodedError{Code: 500, Message: "Failed to parse response"}
	}

	if len(waResp.Messages) == 0 {
		customLogger.ErrorWithData("No message ID in WhatsApp response", map[string]interface{}{
			"component": "WASendService",
			"function":  "sendToWhatsApp",
			"body":      string(body),
			"to":        to,
		})
		return "", &gocom.CodedError{Code: 500, Message: "No message ID returned from WhatsApp"}
	}

	messageId := waResp.Messages[0].Id
	customLogger.InfoWithData("Message sent successfully", map[string]interface{}{
		"component":  "WASendService",
		"function":   "sendToWhatsApp",
		"message_id": messageId,
		"to":         to,
		"template":   templateName,
	})

	return messageId, nil
}

// SendTextMessage sends a plain text message to WhatsApp (without template)
func (o *WASendSvcImpl) SendTextMessage(to, text string) (string, *gocom.CodedError) {
	customLogger.InfoWithData("SendTextMessage started", map[string]interface{}{
		"component":   "WASendService",
		"function":    "SendTextMessage",
		"to":          to,
		"text_length": len(text),
	})

	// Get WhatsApp Phone Number ID and Access Token from config
	phoneNumberId := config.Get(constans.WaPhoneNumberId)
	accessToken := config.Get(constans.WaAuthToken)

	if phoneNumberId == "" {
		customLogger.ErrorWithData("Phone Number ID not configured", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessage",
		})
		return "", &gocom.CodedError{Code: 500, Message: "WhatsApp Phone Number ID not configured"}
	}

	if accessToken == "" {
		customLogger.ErrorWithData("Access Token not configured", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessage",
		})
		return "", &gocom.CodedError{Code: 500, Message: "WhatsApp Access Token not configured"}
	}

	baseURL := fmt.Sprintf("%s/%s/messages", config.Get(constans.BaseURLMeta), phoneNumberId)

	// Build text message payload
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text": map[string]string{
			"body": text,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		customLogger.ErrorWithData("Unable to marshal payload", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessage",
			"error":     err.Error(),
			"to":        to,
		})
		return "", &gocom.CodedError{Code: 500, Message: "Failed to create message payload"}
	}

	customLogger.InfoWithData("Sending text message", map[string]interface{}{
		"component":   "WASendService",
		"function":    "SendTextMessage",
		"to":          to,
		"text_length": len(text),
	})

	// Send to WhatsApp API
	httpReq, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		customLogger.ErrorWithData("Unable to create request", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessage",
			"error":     err.Error(),
			"to":        to,
		})
		return "", &gocom.CodedError{Code: 500, Message: "Failed to create request"}
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		customLogger.ErrorWithData("Request failed", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessage",
			"error":     err.Error(),
			"to":        to,
		})
		return "", &gocom.CodedError{Code: 500, Message: "Failed to send message to WhatsApp"}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		customLogger.ErrorWithData("Unable to read response", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessage",
			"error":     err.Error(),
			"to":        to,
		})
		return "", &gocom.CodedError{Code: 500, Message: "Failed to read response"}
	}

	customLogger.InfoWithData("WhatsApp API response received", map[string]interface{}{
		"component":   "WASendService",
		"function":    "SendTextMessage",
		"status_code": resp.StatusCode,
		"to":          to,
	})

	if resp.StatusCode != 200 {
		customLogger.ErrorWithData("WhatsApp API error", map[string]interface{}{
			"component":   "WASendService",
			"function":    "SendTextMessage",
			"status_code": resp.StatusCode,
			"body":        string(body),
			"to":          to,
		})
		return "", &gocom.CodedError{Code: resp.StatusCode, Message: "WhatsApp API error: " + string(body)}
	}

	var waResp dtos.WAApiSendResponse
	err = json.Unmarshal(body, &waResp)
	if err != nil {
		customLogger.ErrorWithData("Unable to parse response", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessage",
			"error":     err.Error(),
			"to":        to,
		})
		return "", &gocom.CodedError{Code: 500, Message: "Failed to parse response"}
	}

	if len(waResp.Messages) == 0 {
		customLogger.ErrorWithData("No message ID in response", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessage",
			"to":        to,
		})
		return "", &gocom.CodedError{Code: 500, Message: "No message ID returned from WhatsApp"}
	}

	customLogger.InfoWithData("Text message sent successfully", map[string]interface{}{
		"component":  "WASendService",
		"function":   "SendTextMessage",
		"message_id": waResp.Messages[0].Id,
		"to":         to,
	})
	return waResp.Messages[0].Id, nil
}

// SendTextMessageWithLog sends a plain text message and creates message_logs entry
func (o *WASendSvcImpl) SendTextMessageWithLog(to, text, clientId, senderIdentifier string) (string, string, *gocom.CodedError) {
	customLogger.InfoWithData("SendTextMessageWithLog started", map[string]interface{}{
		"component":         "WASendService",
		"function":          "SendTextMessageWithLog",
		"to":                to,
		"text_length":       len(text),
		"sender_identifier": senderIdentifier,
		"client_id":         clientId,
	})

	// Check if we're in an active conversation window (24 hours)
	// WhatsApp charges per conversation, not per message
	conversationKey := fmt.Sprintf("wa_conversation:%s:%s", clientId, to)
	activeConversation := gocom.KeyVal().Get(conversationKey)

	var messageCost float64
	var shouldCharge bool
	var conversationType string

	if activeConversation == "" {
		// No active conversation - this is a NEW business-initiated conversation
		// Charge for service conversation (utility category)
		messageCost = GetBillingService().CalculateMessageCostWithCategory("whatsapp", "utility", clientId)
		shouldCharge = true
		conversationType = "business_initiated"
		customLogger.InfoWithData("New business-initiated conversation", map[string]interface{}{
			"component":         "WASendService",
			"function":          "SendTextMessageWithLog",
			"conversation_type": "business_initiated",
			"cost":              messageCost,
			"should_charge":     true,
		})
	} else if strings.HasPrefix(activeConversation, "user_initiated:") {
		// User initiated conversation - business can reply for FREE
		messageCost = 0
		shouldCharge = false
		conversationType = "user_initiated"
		customLogger.InfoWithData("User-initiated conversation - FREE reply", map[string]interface{}{
			"component":         "WASendService",
			"function":          "SendTextMessageWithLog",
			"conversation_type": "user_initiated",
			"cost":              0,
		})
	} else {
		// Active business-initiated conversation - no additional charge
		messageCost = 0
		shouldCharge = false
		conversationType = "active_conversation"
		customLogger.InfoWithData("Active conversation window - no charge", map[string]interface{}{
			"component":         "WASendService",
			"function":          "SendTextMessageWithLog",
			"conversation_type": "active_conversation",
			"cost":              0,
		})
	}

	// Check balance only if we need to charge
	if shouldCharge {
		canSend, balanceErr := GetBillingService().CanClientSend(clientId, messageCost)
		if balanceErr != nil {
			customLogger.ErrorWithData("Failed to check balance", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendTextMessageWithLog",
				"error":     balanceErr.Error(),
				"client_id": clientId,
			})
			return "", "", &gocom.CodedError{Code: 500, Message: "Failed to check balance"}
		}
		if !canSend {
			customLogger.WarnWithData("Insufficient balance", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendTextMessageWithLog",
				"client_id": clientId,
				"cost":      messageCost,
			})
			return "", "", &gocom.CodedError{Code: 402, Message: "Insufficient balance"}
		}
	}

	// Get WhatsApp credentials - use senderIdentifier if provided, otherwise fallback to config
	var phoneNumberId, accessToken string
	var credErr *gocom.CodedError

	if senderIdentifier != "" {
		phoneNumberId, accessToken, credErr = o.getSenderCredentialsByIdentifier(senderIdentifier)
		if credErr != nil {
			return "", "", credErr
		}
	} else {
		phoneNumberId = config.Get(constans.WaPhoneNumberId)
		accessToken = config.Get(constans.WaAuthToken)
		customLogger.WarnWithData("No senderIdentifier provided, using global config", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessageWithLog",
			"fallback":  "global_config",
		})
	}

	if phoneNumberId == "" {
		customLogger.ErrorWithData("Phone Number ID not configured", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessageWithLog",
		})
		return "", "", &gocom.CodedError{Code: 500, Message: "WhatsApp Phone Number ID not configured"}
	}

	if accessToken == "" {
		customLogger.ErrorWithData("Access Token not configured", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessageWithLog",
		})
		return "", "", &gocom.CodedError{Code: 500, Message: "WhatsApp Access Token not configured"}
	}

	baseURL := fmt.Sprintf("%s/%s/messages", config.Get(constans.BaseURLMeta), phoneNumberId)

	// Build text message payload
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text": map[string]string{
			"body": text,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		customLogger.ErrorWithData("Unable to marshal payload", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessageWithLog",
			"error":     err.Error(),
			"to":        to,
		})
		return "", "", &gocom.CodedError{Code: 500, Message: "Failed to create message payload"}
	}

	customLogger.InfoWithData("Sending text message to WhatsApp", map[string]interface{}{
		"component": "WASendService",
		"function":  "SendTextMessageWithLog",
		"to":        to,
	})

	// Send to WhatsApp API
	httpReq, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		customLogger.ErrorWithData("Unable to create request", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessageWithLog",
			"error":     err.Error(),
			"to":        to,
		})
		return "", "", &gocom.CodedError{Code: 500, Message: "Failed to create request"}
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		customLogger.ErrorWithData("Request failed", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessageWithLog",
			"error":     err.Error(),
			"to":        to,
		})
		return "", "", &gocom.CodedError{Code: 500, Message: "Failed to send message to WhatsApp"}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		customLogger.ErrorWithData("Unable to read response", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessageWithLog",
			"error":     err.Error(),
			"to":        to,
		})
		return "", "", &gocom.CodedError{Code: 500, Message: "Failed to read response"}
	}

	customLogger.InfoWithData("WhatsApp API response received", map[string]interface{}{
		"component":   "WASendService",
		"function":    "SendTextMessageWithLog",
		"status_code": resp.StatusCode,
		"to":          to,
	})

	now := time.Now()
	messageLogId := ulid.Make().String()

	if resp.StatusCode != 200 {
		customLogger.ErrorWithData("WhatsApp API error", map[string]interface{}{
			"component":   "WASendService",
			"function":    "SendTextMessageWithLog",
			"status_code": resp.StatusCode,
			"body":        string(body),
			"to":          to,
		})

		// Create failed message log
		messageLog := &dtos.MessageLog{
			ID:             messageLogId,
			ClientId:       clientId,
			Type:           "whatsapp",
			Direction:      "outbound",
			RecipientType:  "phone",
			RecipientValue: to,
			MessageId:      "",
			MessageContent: text,
			Status:         "failed",
			ErrorMessage:   "WhatsApp API error: " + string(body),
			FailedAt:       now.Format(time.RFC3339),
		}

		// Save failed message log (best effort, don't fail if this fails)
		_, logErr := GetMessageLogsService().Create(messageLog, auth.AuthInfo{ClientId: clientId})
		if logErr != nil {
			customLogger.ErrorWithData("Failed to save failed message log", map[string]interface{}{
				"component":      "WASendService",
				"function":       "SendTextMessageWithLog",
				"error":          logErr.Message,
				"message_log_id": messageLogId,
			})
		}

		return "", messageLogId, &gocom.CodedError{Code: resp.StatusCode, Message: "WhatsApp API error: " + string(body)}
	}

	var waResp dtos.WAApiSendResponse
	err = json.Unmarshal(body, &waResp)
	if err != nil {
		customLogger.ErrorWithData("Unable to parse response", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendTextMessageWithLog",
			"error":     err.Error(),
			"to":        to,
		})

		// Create failed message log
		messageLog := &dtos.MessageLog{
			ID:             messageLogId,
			ClientId:       clientId,
			Type:           "whatsapp",
			Direction:      "outbound",
			RecipientType:  "phone",
			RecipientValue: to,
			MessageId:      "",
			MessageContent: text,
			Status:         "failed",
			ErrorMessage:   "Failed to parse WhatsApp response",
			FailedAt:       now.Format(time.RFC3339),
		}

		GetMessageLogsService().Create(messageLog, auth.AuthInfo{ClientId: clientId})

		return "", messageLogId, &gocom.CodedError{Code: 500, Message: "Failed to parse response"}
	}

	if len(waResp.Messages) == 0 {
		customLogger.ErrorWithData("No message ID in response", map[string]interface{}{
			"component":      "WASendService",
			"function":       "SendTextMessageWithLog",
			"to":             to,
			"message_log_id": messageLogId,
		})

		// Create failed message log
		messageLog := &dtos.MessageLog{
			ID:             messageLogId,
			ClientId:       clientId,
			Type:           "whatsapp",
			Direction:      "outbound",
			RecipientType:  "phone",
			RecipientValue: to,
			MessageId:      "",
			MessageContent: text,
			Status:         "failed",
			ErrorMessage:   "No message ID returned from WhatsApp",
			FailedAt:       now.Format(time.RFC3339),
		}

		GetMessageLogsService().Create(messageLog, auth.AuthInfo{ClientId: clientId})

		return "", messageLogId, &gocom.CodedError{Code: 500, Message: "No message ID returned from WhatsApp"}
	}

	waMessageId := waResp.Messages[0].Id

	// Create successful message log
	messageLog := &dtos.MessageLog{
		ID:             messageLogId,
		ClientId:       clientId,
		SenderId:       constans.AIDefaultSender, // AI auto-response sender (static/temporary)
		Type:           "whatsapp",
		Direction:      "outbound",
		RecipientType:  "phone",
		RecipientValue: to,
		MessageId:      waMessageId,
		MessageContent: text,
		Status:         "sent",
		Cost:           messageCost, // 0 if in active conversation, cost if new conversation
		SentAt:         now.Format(time.RFC3339),
	}

	// Save message log (best effort)
	_, logErr := GetMessageLogsService().Create(messageLog, auth.AuthInfo{ClientId: clientId})
	if logErr != nil {
		customLogger.ErrorWithData("Failed to save message log", map[string]interface{}{
			"component":      "WASendService",
			"function":       "SendTextMessageWithLog",
			"error":          logErr.Message,
			"message_log_id": messageLogId,
		})
		// Continue anyway, message was sent successfully
	}

	// Deduct balance and create conversation window only for new business-initiated conversations
	if shouldCharge {
		_, deductErr := GetBillingService().DeductBalance(
			clientId,
			messageCost,
			waMessageId,
			"whatsapp_conversation",
			fmt.Sprintf("WhatsApp %s conversation with %s", conversationType, to),
			"system", // AI conversation triggered by system
		)
		if deductErr != nil {
			customLogger.ErrorWithData("Failed to deduct balance", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendTextMessageWithLog",
				"error":     deductErr.Message,
				"client_id": clientId,
				"cost":      messageCost,
			})
			// Continue anyway - message already sent and logged
		}

		// Create 24-hour business-initiated conversation window in Redis
		conversationWindow := 24 * time.Hour
		businessInitiatedValue := fmt.Sprintf("business_initiated:%s", time.Now().Format(time.RFC3339))
		setErr := gocom.KeyVal().Set(conversationKey, businessInitiatedValue, conversationWindow)
		if setErr != nil {
			customLogger.ErrorWithData("Failed to set conversation window", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendTextMessageWithLog",
				"error":     setErr.Error(),
				"to":        to,
			})
		} else {
			customLogger.InfoWithData("Business-initiated conversation window created", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendTextMessageWithLog",
				"to":        to,
				"ttl":       "24h",
			})
		}
	}

	customLogger.InfoWithData("SendTextMessageWithLog success", map[string]interface{}{
		"component":         "WASendService",
		"function":          "SendTextMessageWithLog",
		"message_id":        waMessageId,
		"message_log_id":    messageLogId,
		"cost":              messageCost,
		"conversation_type": conversationType,
		"charged":           shouldCharge,
		"to":                to,
	})
	return waMessageId, messageLogId, nil
}

// SendInteractiveButtonMessage sends interactive button message to WhatsApp
func (o *WASendSvcImpl) SendInteractiveButtonMessage(to, bodyText string, buttons []map[string]string, clientId, senderIdentifier string) (string, string, *gocom.CodedError) {
	customLogger.InfoWithData("SendInteractiveButtonMessage started", map[string]interface{}{
		"component":         "WASendService",
		"function":          "SendInteractiveButtonMessage",
		"to":                to,
		"buttons_count":     len(buttons),
		"sender_identifier": senderIdentifier,
		"client_id":         clientId,
	})

	// Check conversation window for pricing
	conversationKey := fmt.Sprintf("wa_conversation:%s:%s", clientId, to)
	activeConversation := gocom.KeyVal().Get(conversationKey)

	var messageCost float64
	var shouldCharge bool
	var conversationType string

	if activeConversation == "" {
		messageCost = GetBillingService().CalculateMessageCostWithCategory("whatsapp", "utility", clientId)
		shouldCharge = true
		conversationType = "business_initiated"
		customLogger.InfoWithData("New business-initiated conversation", map[string]interface{}{
			"component":         "WASendService",
			"function":          "SendInteractiveButtonMessage",
			"conversation_type": "business_initiated",
			"cost":              messageCost,
			"should_charge":     true,
		})
	} else if strings.HasPrefix(activeConversation, "user_initiated:") {
		messageCost = 0
		shouldCharge = false
		conversationType = "user_initiated"
		customLogger.InfoWithData("User-initiated conversation - FREE reply", map[string]interface{}{
			"component":         "WASendService",
			"function":          "SendInteractiveButtonMessage",
			"conversation_type": "user_initiated",
			"cost":              0,
		})
	} else {
		messageCost = 0
		shouldCharge = false
		conversationType = "active_conversation"
		customLogger.InfoWithData("Active conversation window - no charge", map[string]interface{}{
			"component":         "WASendService",
			"function":          "SendInteractiveButtonMessage",
			"conversation_type": "active_conversation",
			"cost":              0,
		})
	}

	if shouldCharge {
		canSend, balanceErr := GetBillingService().CanClientSend(clientId, messageCost)
		if balanceErr != nil {
			customLogger.ErrorWithData("Failed to check balance", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendInteractiveButtonMessage",
				"error":     balanceErr.Error(),
				"client_id": clientId,
			})
			return "", "", &gocom.CodedError{Code: 500, Message: "Failed to check balance"}
		}
		if !canSend {
			customLogger.WarnWithData("Insufficient balance", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendInteractiveButtonMessage",
				"client_id": clientId,
				"cost":      messageCost,
			})
			return "", "", &gocom.CodedError{Code: 402, Message: "Insufficient balance"}
		}
	}

	var phoneNumberId, accessToken string
	var credErr *gocom.CodedError

	if senderIdentifier != "" {
		phoneNumberId, accessToken, credErr = o.getSenderCredentialsByIdentifier(senderIdentifier)
		if credErr != nil {
			return "", "", credErr
		}
	} else {
		phoneNumberId = config.Get(constans.WaPhoneNumberId)
		accessToken = config.Get(constans.WaAuthToken)
		customLogger.WarnWithData("No senderIdentifier provided, using global config", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendInteractiveButtonMessage",
			"fallback":  "global_config",
		})
	}

	if phoneNumberId == "" || accessToken == "" {
		return "", "", &gocom.CodedError{Code: 500, Message: "WhatsApp configuration not set"}
	}

	baseURL := fmt.Sprintf("%s/%s/messages", config.Get(constans.BaseURLMeta), phoneNumberId)

	// Build interactive button payload
	buttonComponents := make([]map[string]interface{}, 0)
	for _, btn := range buttons {
		buttonComponents = append(buttonComponents, map[string]interface{}{
			"type": "reply",
			"reply": map[string]string{
				"id":    btn["id"],
				"title": btn["title"],
			},
		})
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "interactive",
		"interactive": map[string]interface{}{
			"type": "button",
			"body": map[string]string{
				"text": bodyText,
			},
			"action": map[string]interface{}{
				"buttons": buttonComponents,
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", &gocom.CodedError{Code: 500, Message: "Failed to marshal payload"}
	}

	customLogger.DebugWithData("Sending button message payload", map[string]interface{}{
		"component":    "WASendService",
		"function":     "SendInteractiveButtonMessage",
		"payload_size": len(payloadBytes),
		"to":           to,
	})

	httpReq, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", "", &gocom.CodedError{Code: 500, Message: "Failed to create request"}
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", "", &gocom.CodedError{Code: 500, Message: "Failed to send message"}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", &gocom.CodedError{Code: 500, Message: "Failed to read response"}
	}

	now := time.Now()
	messageLogId := ulid.Make().String()

	if resp.StatusCode != 200 {
		customLogger.ErrorWithData("WhatsApp API returned error", map[string]interface{}{
			"component":   "WASendService",
			"function":    "SendInteractiveButtonMessage",
			"status_code": resp.StatusCode,
			"body":        string(body),
			"to":          to,
		})

		// Create failed message log
		messageLog := &dtos.MessageLog{
			ID:             messageLogId,
			ClientId:       clientId,
			SenderId:       constans.AIDefaultSender,
			Type:           "whatsapp",
			Direction:      "outbound",
			RecipientType:  "phone",
			RecipientValue: to,
			MessageId:      "",
			MessageContent: fmt.Sprintf("[Interactive Button] %s", bodyText),
			Status:         "failed",
			ErrorMessage:   "WhatsApp API error: " + string(body),
			FailedAt:       now.Format(time.RFC3339),
		}

		GetMessageLogsService().Create(messageLog, auth.AuthInfo{ClientId: clientId})

		return "", messageLogId, &gocom.CodedError{Code: resp.StatusCode, Message: "WhatsApp API error: " + string(body)}
	}

	var waResp dtos.WAApiSendResponse
	if err := json.Unmarshal(body, &waResp); err != nil {
		return "", messageLogId, &gocom.CodedError{Code: 500, Message: "Failed to parse response"}
	}

	if len(waResp.Messages) == 0 {
		return "", messageLogId, &gocom.CodedError{Code: 500, Message: "No message ID returned"}
	}

	waMessageId := waResp.Messages[0].Id

	// Create message log
	messageLog := &dtos.MessageLog{
		ID:             messageLogId,
		ClientId:       clientId,
		SenderId:       constans.AIDefaultSender,
		Type:           "whatsapp",
		Direction:      "outbound",
		RecipientType:  "phone",
		RecipientValue: to,
		MessageId:      waMessageId,
		MessageContent: fmt.Sprintf("[Interactive Button] %s", bodyText),
		Status:         "sent",
		Cost:           messageCost,
		SentAt:         now.Format(time.RFC3339),
	}

	GetMessageLogsService().Create(messageLog, auth.AuthInfo{ClientId: clientId})

	// Deduct balance and create conversation window only for new business-initiated conversations
	if shouldCharge {
		_, deductErr := GetBillingService().DeductBalance(
			clientId,
			messageCost,
			waMessageId,
			"whatsapp_conversation",
			fmt.Sprintf("WhatsApp %s conversation with %s", conversationType, to),
			"system",
		)
		if deductErr != nil {
			customLogger.ErrorWithData("Failed to deduct balance", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendInteractiveButtonMessage",
				"error":     deductErr.Message,
				"client_id": clientId,
				"cost":      messageCost,
			})
		}

		conversationWindow := 24 * time.Hour
		businessInitiatedValue := fmt.Sprintf("business_initiated:%s", time.Now().Format(time.RFC3339))
		setErr := gocom.KeyVal().Set(conversationKey, businessInitiatedValue, conversationWindow)
		if setErr != nil {
			customLogger.ErrorWithData("Failed to set conversation window", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendInteractiveButtonMessage",
				"error":     setErr.Error(),
				"to":        to,
			})
		} else {
			customLogger.InfoWithData("Business-initiated conversation window created", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendInteractiveButtonMessage",
				"to":        to,
				"ttl":       "24h",
			})
		}
	}

	customLogger.InfoWithData("SendInteractiveButtonMessage success", map[string]interface{}{
		"component":         "WASendService",
		"function":          "SendInteractiveButtonMessage",
		"message_id":        waMessageId,
		"message_log_id":    messageLogId,
		"cost":              messageCost,
		"conversation_type": conversationType,
		"charged":           shouldCharge,
		"to":                to,
	})
	return waMessageId, messageLogId, nil
}

// SendInteractiveListMessage sends interactive list message to WhatsApp
func (o *WASendSvcImpl) SendInteractiveListMessage(to, bodyText, buttonText string, sections []map[string]interface{}, clientId, senderIdentifier string) (string, string, *gocom.CodedError) {
	customLogger.InfoWithData("SendInteractiveListMessage started", map[string]interface{}{
		"component":         "WASendService",
		"function":          "SendInteractiveListMessage",
		"to":                to,
		"sections_count":    len(sections),
		"sender_identifier": senderIdentifier,
		"client_id":         clientId,
	})

	// Check conversation window for pricing
	conversationKey := fmt.Sprintf("wa_conversation:%s:%s", clientId, to)
	activeConversation := gocom.KeyVal().Get(conversationKey)

	var messageCost float64
	var shouldCharge bool
	var conversationType string

	if activeConversation == "" {
		messageCost = GetBillingService().CalculateMessageCostWithCategory("whatsapp", "utility", clientId)
		shouldCharge = true
		conversationType = "business_initiated"
		customLogger.InfoWithData("New business-initiated conversation", map[string]interface{}{
			"component":         "WASendService",
			"function":          "SendInteractiveListMessage",
			"conversation_type": "business_initiated",
			"cost":              messageCost,
		})
	} else if strings.HasPrefix(activeConversation, "user_initiated:") {
		messageCost = 0
		shouldCharge = false
		conversationType = "user_initiated"
		customLogger.InfoWithData("User-initiated conversation - FREE reply", map[string]interface{}{
			"component":         "WASendService",
			"function":          "SendInteractiveListMessage",
			"conversation_type": "user_initiated",
		})
	} else {
		messageCost = 0
		shouldCharge = false
		conversationType = "active_conversation"
		customLogger.InfoWithData("Active conversation window - no charge", map[string]interface{}{
			"component":         "WASendService",
			"function":          "SendInteractiveListMessage",
			"conversation_type": "active_conversation",
		})
	}

	if shouldCharge {
		canSend, balanceErr := GetBillingService().CanClientSend(clientId, messageCost)
		if balanceErr != nil {
			customLogger.ErrorWithData("Failed to check balance", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendInteractiveListMessage",
				"error":     balanceErr.Error(),
				"client_id": clientId,
			})
			return "", "", &gocom.CodedError{Code: 500, Message: "Failed to check balance"}
		}
		if !canSend {
			customLogger.WarnWithData("Insufficient balance", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendInteractiveListMessage",
				"client_id": clientId,
				"cost":      messageCost,
			})
			return "", "", &gocom.CodedError{Code: 402, Message: "Insufficient balance"}
		}
	}

	// Get WhatsApp credentials - use senderIdentifier if provided, otherwise fallback to config
	var phoneNumberId, accessToken string
	var credErr *gocom.CodedError

	if senderIdentifier != "" {
		phoneNumberId, accessToken, credErr = o.getSenderCredentialsByIdentifier(senderIdentifier)
		if credErr != nil {
			return "", "", credErr
		}
	} else {
		phoneNumberId = config.Get(constans.WaPhoneNumberId)
		accessToken = config.Get(constans.WaAuthToken)
		customLogger.WarnWithData("No senderIdentifier provided, using global config", map[string]interface{}{
			"component": "WASendService",
			"function":  "SendInteractiveListMessage",
			"fallback":  "global_config",
		})
	}

	if phoneNumberId == "" || accessToken == "" {
		return "", "", &gocom.CodedError{Code: 500, Message: "WhatsApp configuration not set"}
	}

	baseURL := fmt.Sprintf("%s/%s/messages", config.Get(constans.BaseURLMeta), phoneNumberId)

	// Build interactive list payload
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "interactive",
		"interactive": map[string]interface{}{
			"type": "list",
			"body": map[string]string{
				"text": bodyText,
			},
			"action": map[string]interface{}{
				"button":   buttonText,
				"sections": sections,
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", &gocom.CodedError{Code: 500, Message: "Failed to marshal payload"}
	}

	customLogger.DebugWithData("Sending list message payload", map[string]interface{}{
		"component":    "WASendService",
		"function":     "SendInteractiveListMessage",
		"payload_size": len(payloadBytes),
		"to":           to,
	})

	httpReq, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", "", &gocom.CodedError{Code: 500, Message: "Failed to create request"}
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", "", &gocom.CodedError{Code: 500, Message: "Failed to send message"}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", &gocom.CodedError{Code: 500, Message: "Failed to read response"}
	}

	now := time.Now()
	messageLogId := ulid.Make().String()

	if resp.StatusCode != 200 {
		customLogger.ErrorWithData("WhatsApp API error", map[string]interface{}{
			"component":   "WASendService",
			"function":    "SendInteractiveListMessage",
			"status_code": resp.StatusCode,
			"body":        string(body),
			"to":          to,
		})

		// Create failed message log
		messageLog := &dtos.MessageLog{
			ID:             messageLogId,
			ClientId:       clientId,
			SenderId:       constans.AIDefaultSender,
			Type:           "whatsapp",
			Direction:      "outbound",
			RecipientType:  "phone",
			RecipientValue: to,
			MessageId:      "",
			MessageContent: fmt.Sprintf("[Interactive List] %s", bodyText),
			Status:         "failed",
			ErrorMessage:   "WhatsApp API error: " + string(body),
			FailedAt:       now.Format(time.RFC3339),
		}

		GetMessageLogsService().Create(messageLog, auth.AuthInfo{ClientId: clientId})

		return "", messageLogId, &gocom.CodedError{Code: resp.StatusCode, Message: "WhatsApp API error: " + string(body)}
	}

	var waResp dtos.WAApiSendResponse
	if err := json.Unmarshal(body, &waResp); err != nil {
		return "", messageLogId, &gocom.CodedError{Code: 500, Message: "Failed to parse response"}
	}

	if len(waResp.Messages) == 0 {
		return "", messageLogId, &gocom.CodedError{Code: 500, Message: "No message ID returned"}
	}

	waMessageId := waResp.Messages[0].Id

	// Create message log
	messageLog := &dtos.MessageLog{
		ID:             messageLogId,
		ClientId:       clientId,
		SenderId:       constans.AIDefaultSender,
		Type:           "whatsapp",
		Direction:      "outbound",
		RecipientType:  "phone",
		RecipientValue: to,
		MessageId:      waMessageId,
		MessageContent: fmt.Sprintf("[Interactive List] %s", bodyText),
		Status:         "sent",
		Cost:           messageCost,
		SentAt:         now.Format(time.RFC3339),
	}

	GetMessageLogsService().Create(messageLog, auth.AuthInfo{ClientId: clientId})

	// Deduct balance and create conversation window only for new business-initiated conversations
	if shouldCharge {
		_, deductErr := GetBillingService().DeductBalance(
			clientId,
			messageCost,
			waMessageId,
			"whatsapp_conversation",
			fmt.Sprintf("WhatsApp %s conversation with %s", conversationType, to),
			"system",
		)
		if deductErr != nil {
			customLogger.ErrorWithData("Failed to deduct balance", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendInteractiveListMessage",
				"error":     deductErr.Message,
				"client_id": clientId,
				"cost":      messageCost,
			})
		}

		conversationWindow := 24 * time.Hour
		businessInitiatedValue := fmt.Sprintf("business_initiated:%s", time.Now().Format(time.RFC3339))
		setErr := gocom.KeyVal().Set(conversationKey, businessInitiatedValue, conversationWindow)
		if setErr != nil {
			customLogger.ErrorWithData("Failed to set conversation window", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendInteractiveListMessage",
				"error":     setErr.Error(),
				"to":        to,
			})
		} else {
			customLogger.InfoWithData("Business-initiated conversation window created", map[string]interface{}{
				"component": "WASendService",
				"function":  "SendInteractiveListMessage",
				"to":        to,
				"ttl":       "24h",
			})
		}
	}

	customLogger.InfoWithData("SendInteractiveListMessage success", map[string]interface{}{
		"component":         "WASendService",
		"function":          "SendInteractiveListMessage",
		"message_id":        waMessageId,
		"message_log_id":    messageLogId,
		"cost":              messageCost,
		"conversation_type": conversationType,
		"charged":           shouldCharge,
		"to":                to,
	})
	return waMessageId, messageLogId, nil
}

func (o *WASendSvcImpl) UploadFile(file *multipart.FileHeader, authInfo auth.AuthInfo) (string, *gocom.CodedError) {
	customLogger.InfoWithData("UploadFile started", map[string]interface{}{
		"component": "WASendService",
		"function":  "UploadFile",
		"filename":  file.Filename,
		"size":      file.Size,
		"client_id": authInfo.ClientId,
	})

	// Validate file type
	contentType := file.Header.Get("Content-Type")
	if !o.isValidFileType(file.Header.Get("Content-Type")) {
		customLogger.WarnWithData("Invalid file type", map[string]interface{}{
			"component":    "WASendService",
			"function":     "UploadFile",
			"content_type": contentType,
			"filename":     file.Filename,
		})
		return "", common.ERR_INVALID_REQUEST
	}

	// Validate file size (max: image 5MB, video 25MB, pdf 10MB)
	if !o.validateFileSize(o.getFileType(contentType), file.Size) {
		customLogger.WarnWithData("File size too big", map[string]interface{}{
			"component": "WASendService",
			"function":  "UploadFile",
			"size":      file.Size,
			"filename":  file.Filename,
		})
		return "", common.ERR_INVALID_REQUEST
	}

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		customLogger.ErrorWithData("Failed to open file", map[string]interface{}{
			"component": "WASendService",
			"function":  "UploadFile",
			"error":     err.Error(),
			"filename":  file.Filename,
		})
		return "", common.ERR_INTERNAL_SERVER_ERROR
	}
	defer func(src multipart.File) {
		errUpload := src.Close()
		if errUpload != nil {
			customLogger.ErrorWithData("Failed to close file", map[string]interface{}{
				"component": "WASendService",
				"function":  "UploadFile",
				"error":     errUpload.Error(),
				"filename":  file.Filename,
			})
		}
	}(src)

	uploadDir := fmt.Sprintf("./files/%s", o.getFileType(contentType))
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		customLogger.ErrorWithData("Failed to create directory", map[string]interface{}{
			"component": "WASendService",
			"function":  "UploadFile",
			"error":     err.Error(),
			"directory": uploadDir,
		})
		return "", common.ERR_INTERNAL_SERVER_ERROR
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	newFilename := fmt.Sprintf("%s%s", ulid.Make().String(), ext)
	filePath := filepath.Join(uploadDir, newFilename)

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		customLogger.ErrorWithData("Failed to create file", map[string]interface{}{
			"component": "WASendService",
			"function":  "UploadFile",
			"error":     err.Error(),
			"file_path": filePath,
		})
		return "", common.ERR_INTERNAL_SERVER_ERROR
	}
	defer dst.Close()

	// Copy file
	if _, err := io.Copy(dst, src); err != nil {
		customLogger.ErrorWithData("Failed to save file", map[string]interface{}{
			"component": "WASendService",
			"function":  "UploadFile",
			"error":     err.Error(),
			"file_path": filePath,
		})
		return "", common.ERR_INTERNAL_SERVER_ERROR
	}

	// Generate public URL
	baseURL := config.Get("base.url", "https://ilbedev.kitamandiri.com")
	if baseURL == "" {
		baseURL = "http://localhost:8080" // Default
	}
	//imageUrl := fmt.Sprintf("%s/images/%s", baseURL, newFilename)
	fileUrl := fmt.Sprintf("%s/api/v1/instant-link/wa-send/%s/%s", baseURL, o.getFileType(contentType), newFilename)

	customLogger.InfoWithData("File upload success", map[string]interface{}{
		"component": "WASendService",
		"function":  "UploadFile",
		"file_url":  fileUrl,
		"filename":  newFilename,
		"client_id": authInfo.ClientId,
	})
	return fileUrl, nil
}

func (o *WASendSvcImpl) isValidFileType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/jpg":
		return true
	case "video/mp4", "video/quicktime", "video/x-msvideo", "video/mpeg", "video/3gpp":
		return true
	case "application/pdf":
		return true
	}
	return false
}

func (o *WASendSvcImpl) getFileType(contentType string) string {
	switch contentType {
	case "image/jpeg", "image/png", "image/jpg":
		return "image"
	case "video/mp4", "video/quicktime", "video/x-msvideo", "video/mpeg", "video/3gpp":
		return "video"
	case "application/pdf":
		return "pdf"
	}
	return ""
}

func (o *WASendSvcImpl) validateFileSize(fileType string, fileSize int64) bool {
	switch fileType {
	case "image":
		return fileSize <= 5*1024*1024
	case "video":
		return fileSize <= 25*1024*1024
	case "pdf":
		return fileSize <= 10*1024*1024
	default:
		return false
	}
}

func (o *WASendSvcImpl) GetFiles(fileType, filename string, authInfo auth.AuthInfo) ([]byte, *gocom.CodedError) {
	customLogger.InfoWithData("GetFiles started", map[string]interface{}{
		"component": "WASendService",
		"function":  "GetFiles",
		"filename":  filename,
		"client_id": authInfo.ClientId,
	})

	// Validate filename (prevent directory traversal)
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		customLogger.WarnWithData("Invalid filename", map[string]interface{}{
			"component": "WASendService",
			"function":  "GetFiles",
			"filename":  filename,
			"reason":    "directory_traversal_attempt",
		})
		return nil, &gocom.CodedError{
			Code:    400,
			Message: "Invalid filename",
		}
	}

	filePath := filepath.Join(fmt.Sprintf("./files/%s", fileType), filename)
	customLogger.DebugWithData("Checking file path", map[string]interface{}{
		"component": "WASendService",
		"function":  "GetFiles",
		"file_path": filePath,
		"filename":  filename,
		"file_type": fileType,
	})

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		customLogger.ErrorWithData("File does not exist", map[string]interface{}{
			"component": "WASendService",
			"function":  "GetFiles",
			"file_path": filePath,
			"filename":  filename,
			"file_type": fileType,
		})
		return nil, &gocom.CodedError{
			Code:    404,
			Message: "File not found",
		}
	}
	customLogger.DebugWithData("File exists", map[string]interface{}{
		"component": "WASendService",
		"function":  "GetFiles",
		"filename":  filename,
		"file_type": fileType,
		"size":      fileInfo.Size(),
	})

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		customLogger.ErrorWithData("Failed to read file", map[string]interface{}{
			"component": "WASendService",
			"function":  "GetFiles",
			"error":     err.Error(),
			"file_path": filePath,
			"file_type": fileType,
		})
		return nil, common.ERR_INTERNAL_SERVER_ERROR
	}

	customLogger.InfoWithData("GetFiles success", map[string]interface{}{
		"component":   "WASendService",
		"function":    "GetFiles",
		"filename":    filename,
		"file_type":   fileType,
		"data_length": len(data),
	})
	return data, nil
}

//---------------------------------------

var waSendSvc *WASendSvcImpl
var waSendSvcOnce sync.Once

func GetWASendSvc() WASendSvc {
	if waSendSvc == nil {
		waSendSvcOnce.Do(func() {
			waSendSvc = &WASendSvcImpl{}
		})
	}

	return waSendSvc
}
