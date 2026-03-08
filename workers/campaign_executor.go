package workers

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/campaign"
	"gitlab.com/anti_metter/switching_common/campaignRecipient"
	"gitlab.com/anti_metter/switching_common/sender"
	"gitlab.com/anti_metter/switching_common/waTemplate"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/services"
)

type CampaignExecutor struct {
	stopChan        chan bool
	wg              sync.WaitGroup
	processingLock  sync.Mutex
	processingCamps map[string]bool
}

var campaignExecutor *CampaignExecutor
var onceCampaignExecutor sync.Once

func GetCampaignExecutor() *CampaignExecutor {
	onceCampaignExecutor.Do(func() {
		campaignExecutor = &CampaignExecutor{
			stopChan:        make(chan bool),
			processingCamps: make(map[string]bool), // NEW: Initialize map
		}
	})
	return campaignExecutor
}

// Start begins the campaign executor worker
func (ce *CampaignExecutor) Start() {
	customLogger.InfoWithData("Starting campaign executor worker", map[string]interface{}{
		"component": "CampaignExecutor",
		"action":    "start",
	})
	ce.wg.Add(1)
	go ce.run()
}

// Stop gracefully stops the campaign executor worker
func (ce *CampaignExecutor) Stop() {
	customLogger.InfoWithData("Stopping campaign executor worker", map[string]interface{}{
		"component": "CampaignExecutor",
		"action":    "stop",
	})
	close(ce.stopChan)
	ce.wg.Wait()
	customLogger.InfoWithData("Campaign executor worker stopped", map[string]interface{}{
		"component": "CampaignExecutor",
		"status":    "stopped",
	})
}

// run is the main worker loop
func (ce *CampaignExecutor) run() {
	defer ce.wg.Done()

	// Check for running campaigns every 10 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	customLogger.InfoWithData("Worker started", map[string]interface{}{
		"component":      "CampaignExecutor",
		"check_interval": "10 seconds",
		"status":         "running",
	})

	for {
		select {
		case <-ce.stopChan:
			customLogger.InfoWithData("Received stop signal", map[string]interface{}{
				"component": "CampaignExecutor",
				"action":    "stopping",
			})
			return
		case <-ticker.C:
			// Process scheduled campaigns first (check if time to start)
			ce.processScheduledCampaigns()
			// Then process running campaigns
			ce.processRunningCampaigns()
		}
	}
}

// processRunningCampaigns finds and processes all running campaigns
func (ce *CampaignExecutor) processRunningCampaigns() {
	campaigns, _, _ := campaign.GetRepo().Search("", "", "", "", "running", time.Time{}, time.Time{}, 1, 50)

	if len(campaigns) == 0 {
		return
	}

	customLogger.InfoWithData("Found running campaigns", map[string]interface{}{
		"component":       "CampaignExecutor",
		"campaigns_count": len(campaigns),
	})

	for _, camp := range campaigns {
		campCopy := camp

		// NEW: Check if campaign is already being processed
		ce.processingLock.Lock()
		if ce.processingCamps[campCopy.ID] {
			customLogger.InfoWithData("Campaign already being processed, skipping", map[string]interface{}{
				"component":   "CampaignExecutor",
				"campaign_id": campCopy.ID,
				"action":      "skip",
			})
			ce.processingLock.Unlock()
			continue
		}
		// Mark as processing
		ce.processingCamps[campCopy.ID] = true
		ce.processingLock.Unlock()

		go ce.processCampaign(&campCopy)
	}
}

// processCampaign processes a single campaign
func (ce *CampaignExecutor) processCampaign(camp *campaign.Campaign) {
	// NEW: Ensure we remove from processing map when done
	defer func() {
		ce.processingLock.Lock()
		delete(ce.processingCamps, camp.ID)
		ce.processingLock.Unlock()
		customLogger.InfoWithData("Finished processing campaign", map[string]interface{}{
			"component":     "CampaignExecutor",
			"campaign_id":   camp.ID,
			"campaign_name": camp.Name,
			"status":        "completed",
		})
	}()

	customLogger.InfoWithData("Processing campaign", map[string]interface{}{
		"component":     "CampaignExecutor",
		"campaign_id":   camp.ID,
		"campaign_name": camp.Name,
	})

	// Get pending recipients for this campaign
	recipients, _, totalPending := campaignRecipient.GetRepo().SearchByCampaignId(camp.ID, "pending", 1, camp.BatchSize)

	if len(recipients) == 0 {
		// No more pending recipients, mark campaign as completed
		customLogger.InfoWithData("No pending recipients, marking as completed", map[string]interface{}{
			"component":   "CampaignExecutor",
			"campaign_id": camp.ID,
			"action":      "complete_campaign",
		})
		ce.completeCampaign(camp)
		return
	}

	customLogger.InfoWithData("Found pending recipients", map[string]interface{}{
		"component":        "CampaignExecutor",
		"campaign_id":      camp.ID,
		"recipients_count": len(recipients),
		"total_pending":    totalPending,
	})

	// Get template and sender
	templateMdl := waTemplate.GetRepo().GetById(camp.TemplateId)
	if templateMdl == nil {
		customLogger.ErrorWithData("Template not found", map[string]interface{}{
			"component":   "CampaignExecutor",
			"template_id": camp.TemplateId,
			"campaign_id": camp.ID,
		})
		return
	}

	senderMdl := sender.GetRepo().GetById(camp.SenderId)
	if senderMdl == nil {
		customLogger.ErrorWithData("Sender not found", map[string]interface{}{
			"component":   "CampaignExecutor",
			"sender_id":   camp.SenderId,
			"campaign_id": camp.ID,
		})
		return
	}

	// Process batch of recipients
	successCount := 0
	failCount := 0

	for _, recipient := range recipients {
		// Check if campaign is still running (might have been paused)
		freshCamp := campaign.GetRepo().GetById(camp.ID)
		if freshCamp == nil {
			customLogger.WarnWithData("Campaign not found anymore, stopping execution", map[string]interface{}{
				"component":   "CampaignExecutor",
				"campaign_id": camp.ID,
				"action":      "stop_execution",
			})
			return
		}
		if freshCamp.Status != "running" {
			customLogger.InfoWithData("Campaign no longer running, stopping execution", map[string]interface{}{
				"component":   "CampaignExecutor",
				"campaign_id": camp.ID,
				"status":      freshCamp.Status,
				"action":      "stop_execution",
			})
			return
		}

		// Send message to recipient
		success := ce.sendToRecipient(camp, &recipient, templateMdl, senderMdl)
		if success {
			successCount++
		} else {
			failCount++
		}

		// Apply delay between messages
		if camp.DelaySeconds > 0 {
			time.Sleep(time.Duration(camp.DelaySeconds) * time.Second)
		}
	}

	customLogger.InfoWithData("Batch completed", map[string]interface{}{
		"component":     "CampaignExecutor",
		"campaign_id":   camp.ID,
		"success_count": successCount,
		"fail_count":    failCount,
	})

	// Update campaign statistics
	ce.updateCampaignStats(camp)
}

// sendToRecipient sends a message to a single recipient
func (ce *CampaignExecutor) sendToRecipient(camp *campaign.Campaign, recipient *campaignRecipient.CampaignRecipient, templateMdl *waTemplate.WATemplate, senderMdl *sender.Sender) bool {
	customLogger.InfoWithData("Sending message to recipient", map[string]interface{}{
		"component":    "CampaignExecutor",
		"recipient_id": recipient.ID,
		"phone":        recipient.RecipientValue,
		"campaign_id":  camp.ID,
	})

	// Parse parameters
	var parameters map[string]interface{}
	if recipient.Parameters != "" {
		if err := json.Unmarshal([]byte(recipient.Parameters), &parameters); err != nil {
			customLogger.WarnWithData("Failed to parse recipient parameters", map[string]interface{}{
				"component":    "CampaignExecutor",
				"recipient_id": recipient.ID,
				"error":        err.Error(),
			})
			parameters = make(map[string]interface{})
		}
	} else {
		parameters = make(map[string]interface{})
	}

	// Build request based on campaign type
	switch camp.Type {
	case "whatsapp":
		return ce.sendWhatsAppMessage(camp, recipient, templateMdl, senderMdl, parameters)
	case "sms":
		customLogger.WarnWithData("SMS sending not implemented yet", map[string]interface{}{
			"component":   "CampaignExecutor",
			"campaign_id": camp.ID,
			"type":        "sms",
		})
		ce.markRecipientFailed(recipient, "SMS sending not implemented")
		return false
	case "email":
		customLogger.WarnWithData("Email sending not implemented yet", map[string]interface{}{
			"component":   "CampaignExecutor",
			"campaign_id": camp.ID,
			"type":        "email",
		})
		ce.markRecipientFailed(recipient, "Email sending not implemented")
		return false
	default:
		customLogger.ErrorWithData("Unknown campaign type", map[string]interface{}{
			"component":   "CampaignExecutor",
			"campaign_id": camp.ID,
			"type":        camp.Type,
		})
		ce.markRecipientFailed(recipient, fmt.Sprintf("Unknown campaign type: %s", camp.Type))
		return false
	}
}

// sendWhatsAppMessage sends a WhatsApp message to a recipient
func (ce *CampaignExecutor) sendWhatsAppMessage(camp *campaign.Campaign, recipient *campaignRecipient.CampaignRecipient, templateMdl *waTemplate.WATemplate, _ *sender.Sender, parameters map[string]interface{}) bool {
	// Build parameters based on template structure
	var waParams []dtos.WAMessageParameter

	// Note: templateMdl.Components is from switching_common package
	// We need to check if template has header with media by looking at template name/structure
	// For now, we'll try to detect from parameters provided

	// Check if parameters contain header media keys
	hasHeaderImage := false
	hasHeaderVideo := false
	hasHeaderDocument := false

	if _, ok := parameters["header_image"]; ok {
		hasHeaderImage = true
	} else if _, ok := parameters["image"]; ok {
		hasHeaderImage = true
	} else if _, ok := parameters["image_id"]; ok {
		hasHeaderImage = true
	}

	if _, ok := parameters["header_video"]; ok {
		hasHeaderVideo = true
	} else if _, ok := parameters["video"]; ok {
		hasHeaderVideo = true
	}

	if _, ok := parameters["header_document"]; ok {
		hasHeaderDocument = true
	} else if _, ok := parameters["document"]; ok {
		hasHeaderDocument = true
	}

	// Add header media parameter if exists
	if hasHeaderImage {
		if imageURL, ok := parameters["header_image"].(string); ok && imageURL != "" {
			waParams = append(waParams, dtos.WAMessageParameter{
				Type:  "image",
				Image: &dtos.WAMessageMediaParam{Link: imageURL},
			})
		} else if imageURL, ok := parameters["image"].(string); ok && imageURL != "" {
			waParams = append(waParams, dtos.WAMessageParameter{
				Type:  "image",
				Image: &dtos.WAMessageMediaParam{Link: imageURL},
			})
		} else if imageID, ok := parameters["image_id"].(string); ok && imageID != "" {
			waParams = append(waParams, dtos.WAMessageParameter{
				Type:  "image",
				Image: &dtos.WAMessageMediaParam{ID: imageID},
			})
		}
	} else if hasHeaderVideo {
		if videoURL, ok := parameters["header_video"].(string); ok && videoURL != "" {
			waParams = append(waParams, dtos.WAMessageParameter{
				Type:  "video",
				Video: &dtos.WAMessageMediaParam{Link: videoURL},
			})
		} else if videoURL, ok := parameters["video"].(string); ok && videoURL != "" {
			waParams = append(waParams, dtos.WAMessageParameter{
				Type:  "video",
				Video: &dtos.WAMessageMediaParam{Link: videoURL},
			})
		}
	} else if hasHeaderDocument {
		if docURL, ok := parameters["header_document"].(string); ok && docURL != "" {
			waParams = append(waParams, dtos.WAMessageParameter{
				Type:     "document",
				Document: &dtos.WAMessageMediaParam{Link: docURL},
			})
		} else if docURL, ok := parameters["document"].(string); ok && docURL != "" {
			waParams = append(waParams, dtos.WAMessageParameter{
				Type:     "document",
				Document: &dtos.WAMessageMediaParam{Link: docURL},
			})
		}
	}

	// Add body text parameters (variables like {{1}}, {{2}}, etc)
	// Only add parameters that are not header media
	bodyParamKeys := []string{}
	for k := range parameters {
		// Skip header-related keys
		if k != "header_image" && k != "header_video" && k != "header_document" &&
			k != "image" && k != "video" && k != "document" && k != "image_id" &&
			k != "header" {
			bodyParamKeys = append(bodyParamKeys, k)
		}
	}

	// Sort keys to ensure consistent order (param1, param2, etc)
	// Assuming keys are like "param1", "param2", or "1", "2"
	for i := 1; i <= len(bodyParamKeys); i++ {
		key := fmt.Sprintf("param%d", i)
		if v, ok := parameters[key]; ok {
			waParams = append(waParams, dtos.WAMessageParameter{
				Type: "text",
				Text: fmt.Sprintf("%v", v),
			})
		} else {
			// Try numeric key
			key = fmt.Sprintf("%d", i)
			if v, ok := parameters[key]; ok {
				waParams = append(waParams, dtos.WAMessageParameter{
					Type: "text",
					Text: fmt.Sprintf("%v", v),
				})
			}
		}
	}

	hasHeaderMedia := hasHeaderImage || hasHeaderVideo || hasHeaderDocument
	customLogger.InfoWithData("Prepared parameters for template", map[string]interface{}{
		"component":        "CampaignExecutor",
		"recipient_id":     recipient.ID,
		"params_count":     len(waParams),
		"has_header_media": hasHeaderMedia,
		"has_image":        hasHeaderImage,
		"has_video":        hasHeaderVideo,
		"has_document":     hasHeaderDocument,
	})

	// Prepare WA send request
	req := dtos.WASendMessageReq{
		SenderId:   camp.SenderId,
		TemplateId: templateMdl.ID,
		To:         recipient.RecipientValue,
		Parameters: waParams,
		CampaignId: camp.ID, // Set campaign ID for tracking
	}

	// Create auth info for the client
	authInfo := auth.AuthInfo{
		ClientId: camp.ClientId,
		UserId:   camp.CreatedBy,
	}

	// Send message using WASendService
	resp, err := services.GetWASendSvc().SendMessage(req, authInfo)
	if err != nil {
		customLogger.ErrorWithData("Failed to send WhatsApp message", map[string]interface{}{
			"component":    "CampaignExecutor",
			"recipient_id": recipient.ID,
			"phone":        recipient.RecipientValue,
			"error":        fmt.Sprintf("%v", err),
		})
		ce.markRecipientFailed(recipient, fmt.Sprintf("%v", err))
		return false
	}

	// Mark recipient as sent
	ce.markRecipientSent(recipient, resp.MessageId)
	return true
}

// markRecipientSent updates recipient status to sent
func (ce *CampaignExecutor) markRecipientSent(recipient *campaignRecipient.CampaignRecipient, messageId string) {
	now := time.Now()
	recipient.Status = "sent"
	recipient.MessageId = messageId
	recipient.SentAt = &now

	// Update asynchronously to avoid blocking
	go func(r *campaignRecipient.CampaignRecipient) {
		err := campaignRecipient.GetRepo().Update(r)
		if err != nil {
			customLogger.ErrorWithData("Failed to update recipient status", map[string]interface{}{
				"component":    "CampaignExecutor",
				"recipient_id": r.ID,
				"error":        err.Error(),
				"status":       "sent",
			})
		} else {
			customLogger.InfoWithData("Recipient marked as sent", map[string]interface{}{
				"component":    "CampaignExecutor",
				"recipient_id": r.ID,
				"message_id":   messageId,
				"status":       "sent",
			})
		}
	}(recipient)
}

// markRecipientFailed updates recipient status to failed
func (ce *CampaignExecutor) markRecipientFailed(recipient *campaignRecipient.CampaignRecipient, errorMessage string) {
	now := time.Now()
	recipient.Status = "failed"
	recipient.ErrorMessage = errorMessage
	recipient.FailedAt = &now
	recipient.RetryCount++

	// Update asynchronously to avoid blocking
	go func(r *campaignRecipient.CampaignRecipient, errMsg string) {
		err := campaignRecipient.GetRepo().Update(r)
		if err != nil {
			customLogger.ErrorWithData("Failed to update recipient status", map[string]interface{}{
				"component":    "CampaignExecutor",
				"recipient_id": r.ID,
				"error":        err.Error(),
				"status":       "failed",
			})
		} else {
			customLogger.WarnWithData("Recipient marked as failed", map[string]interface{}{
				"component":     "CampaignExecutor",
				"recipient_id":  r.ID,
				"error_message": errMsg,
				"status":        "failed",
				"retry_count":   r.RetryCount,
			})
		}
	}(recipient, errorMessage)
}

// updateCampaignStats updates campaign statistics
func (ce *CampaignExecutor) updateCampaignStats(camp *campaign.Campaign) {
	// Count recipients by status
	_, _, sentCount := campaignRecipient.GetRepo().SearchByCampaignId(camp.ID, "sent", 1, 1)
	_, _, deliveredCount := campaignRecipient.GetRepo().SearchByCampaignId(camp.ID, "delivered", 1, 1)
	_, _, failedCount := campaignRecipient.GetRepo().SearchByCampaignId(camp.ID, "failed", 1, 1)
	_, _, readCount := campaignRecipient.GetRepo().SearchByCampaignId(camp.ID, "read", 1, 1)

	camp.TotalSent = int(sentCount + deliveredCount + readCount) // sent includes delivered and read
	camp.TotalDelivered = int(deliveredCount + readCount)
	camp.TotalFailed = int(failedCount)
	camp.TotalRead = int(readCount)

	err := campaign.GetRepo().Update(camp)
	if err != nil {
		customLogger.ErrorWithData("Failed to update campaign stats", map[string]interface{}{
			"component":   "CampaignExecutor",
			"campaign_id": camp.ID,
			"error":       err.Error(),
		})
	} else {
		customLogger.InfoWithData("Campaign stats updated", map[string]interface{}{
			"component":       "CampaignExecutor",
			"campaign_id":     camp.ID,
			"total_sent":      camp.TotalSent,
			"total_delivered": camp.TotalDelivered,
			"total_failed":    camp.TotalFailed,
			"total_read":      camp.TotalRead,
		})
	}
}

// completeCampaign marks campaign as completed
func (ce *CampaignExecutor) completeCampaign(camp *campaign.Campaign) {
	now := time.Now()
	camp.Status = "completed"
	camp.CompletedAt = &now

	ce.updateCampaignStats(camp)

	err := campaign.GetRepo().Update(camp)
	if err != nil {
		customLogger.ErrorWithData("Failed to mark campaign as completed", map[string]interface{}{
			"component":   "CampaignExecutor",
			"campaign_id": camp.ID,
			"error":       err.Error(),
		})
	} else {
		customLogger.InfoWithData("Campaign completed", map[string]interface{}{
			"component":     "CampaignExecutor",
			"campaign_id":   camp.ID,
			"campaign_name": camp.Name,
			"total_sent":    camp.TotalSent,
			"total_failed":  camp.TotalFailed,
			"status":        "completed",
		})
	}
}

// processScheduledCampaigns checks for scheduled campaigns that are ready to start
func (ce *CampaignExecutor) processScheduledCampaigns() {
	now := time.Now()

	// Find campaigns with status "scheduled" and scheduled_at <= now
	campaigns, _, _ := campaign.GetRepo().Search("", "", "", "", "scheduled", time.Time{}, now, 1, 50)

	if len(campaigns) == 0 {
		return
	}

	customLogger.InfoWithData("Found scheduled campaigns ready to start", map[string]interface{}{
		"component":       "CampaignExecutor",
		"campaigns_count": len(campaigns),
		"current_time":    now.Format(time.RFC3339),
	})

	for _, camp := range campaigns {
		campCopy := camp

		// Check if campaign is already being processed
		ce.processingLock.Lock()
		if ce.processingCamps[campCopy.ID] {
			customLogger.InfoWithData("Scheduled campaign already being processed, skipping", map[string]interface{}{
				"component":   "CampaignExecutor",
				"campaign_id": campCopy.ID,
				"action":      "skip",
			})
			ce.processingLock.Unlock()
			continue
		}
		// Mark as processing
		ce.processingCamps[campCopy.ID] = true
		ce.processingLock.Unlock()

		// Start campaign asynchronously
		go ce.startScheduledCampaign(&campCopy)
	}
}

// startScheduledCampaign updates campaign status from scheduled to running
func (ce *CampaignExecutor) startScheduledCampaign(camp *campaign.Campaign) {
	defer func() {
		ce.processingLock.Lock()
		delete(ce.processingCamps, camp.ID)
		ce.processingLock.Unlock()
	}()

	customLogger.InfoWithData("Starting scheduled campaign", map[string]interface{}{
		"component":     "CampaignExecutor",
		"campaign_id":   camp.ID,
		"campaign_name": camp.Name,
		"scheduled_at":  camp.ScheduledAt,
	})

	// Double check status to prevent race condition
	freshCamp := campaign.GetRepo().GetById(camp.ID)
	if freshCamp == nil {
		customLogger.WarnWithData("Campaign not found", map[string]interface{}{
			"component":   "CampaignExecutor",
			"campaign_id": camp.ID,
		})
		return
	}

	if freshCamp.Status != "scheduled" {
		customLogger.InfoWithData("Campaign status changed, skipping", map[string]interface{}{
			"component":      "CampaignExecutor",
			"campaign_id":    camp.ID,
			"current_status": freshCamp.Status,
		})
		return
	}

	// Update status to running
	now := time.Now()
	freshCamp.Status = "running"
	freshCamp.StartedAt = &now

	err := campaign.GetRepo().Update(freshCamp)
	if err != nil {
		customLogger.ErrorWithData("Failed to update campaign status to running", map[string]interface{}{
			"component":   "CampaignExecutor",
			"campaign_id": camp.ID,
			"error":       err.Error(),
		})
		return
	}

	customLogger.InfoWithData("Campaign status updated to running", map[string]interface{}{
		"component":     "CampaignExecutor",
		"campaign_id":   camp.ID,
		"campaign_name": camp.Name,
		"status":        "running",
		"started_at":    now.Format(time.RFC3339),
	})
}
