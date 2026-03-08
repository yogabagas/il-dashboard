package workers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ariandi/gocom/config"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/waTemplate"
	"gitlab.com/bot3342545/il-dashboard/constans"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/services"
)

type TemplateStatusChecker struct {
	stopChan chan bool
	wg       sync.WaitGroup
}

var templateStatusChecker *TemplateStatusChecker
var onceTemplateStatusChecker sync.Once

func GetTemplateStatusChecker() *TemplateStatusChecker {
	onceTemplateStatusChecker.Do(func() {
		templateStatusChecker = &TemplateStatusChecker{
			stopChan: make(chan bool),
		}
	})
	return templateStatusChecker
}

// Start begins the template status checker worker
func (tsc *TemplateStatusChecker) Start() {
	customLogger.InfoWithData("Starting template status checker worker", map[string]interface{}{
		"component": "TemplateStatusChecker",
		"action":    "start",
	})
	tsc.wg.Add(1)
	go tsc.run()
}

// Stop gracefully stops the template status checker worker
func (tsc *TemplateStatusChecker) Stop() {
	customLogger.InfoWithData("Stopping template status checker worker", map[string]interface{}{
		"component": "TemplateStatusChecker",
		"action":    "stop",
	})
	close(tsc.stopChan)
	tsc.wg.Wait()
	customLogger.InfoWithData("Template status checker worker stopped", map[string]interface{}{
		"component": "TemplateStatusChecker",
		"status":    "stopped",
	})
}

// run is the main worker loop - checks every 1 hour
func (tsc *TemplateStatusChecker) run() {
	defer tsc.wg.Done()

	// Check templates every 1 hour
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	customLogger.InfoWithData("Worker started", map[string]interface{}{
		"component":      "TemplateStatusChecker",
		"check_interval": "1 hour",
		"status":         "running",
	})

	// Run immediately on startup
	tsc.checkAllTemplateStatuses()

	for {
		select {
		case <-tsc.stopChan:
			customLogger.InfoWithData("Received stop signal", map[string]interface{}{
				"component": "TemplateStatusChecker",
				"action":    "stopping",
			})
			return
		case <-ticker.C:
			tsc.checkAllTemplateStatuses()
		}
	}
}

// checkAllTemplateStatuses checks all templates with WATemplateId
func (tsc *TemplateStatusChecker) checkAllTemplateStatuses() {
	customLogger.InfoWithData("Starting template status check cycle", map[string]interface{}{
		"component": "TemplateStatusChecker",
		"action":    "check_cycle_start",
	})

	// Get access token and WABA ID from config
	accessToken := config.Get(constans.WaAuthToken, "")
	wabaId := config.Get(constans.WaBaId, "")

	if accessToken == "" || wabaId == "" {
		customLogger.ErrorWithData("Access token or WABA ID not configured", map[string]interface{}{
			"component":        "TemplateStatusChecker",
			"has_access_token": accessToken != "",
			"has_waba_id":      wabaId != "",
			"action":           "skipping_check",
		})
		return
	}

	// Get all templates that have WATemplateId (meaning they were submitted to WhatsApp)
	// We search for SUBMITTED and APPROVED templates
	templates := tsc.getTemplatesWithWATemplateId()
	if len(templates) == 0 {
		customLogger.InfoWithData("No templates to check", map[string]interface{}{
			"component":       "TemplateStatusChecker",
			"templates_count": 0,
		})
		return
	}

	customLogger.InfoWithData("Found templates to check", map[string]interface{}{
		"component":       "TemplateStatusChecker",
		"templates_count": len(templates),
	})

	updatedCount := 0
	for _, template := range templates {
		if tsc.checkAndUpdateTemplate(&template, wabaId, accessToken) {
			updatedCount++
		}
		// Add small delay between API calls to avoid rate limiting
		time.Sleep(500 * time.Millisecond)
	}

	customLogger.InfoWithData("Template check cycle completed", map[string]interface{}{
		"component":     "TemplateStatusChecker",
		"updated_count": updatedCount,
		"checked_count": len(templates),
	})
}

// getTemplatesWithWATemplateId retrieves templates that have WATemplateId
func (tsc *TemplateStatusChecker) getTemplatesWithWATemplateId() []waTemplate.WATemplate {
	// Get templates with status SUBMITTED or APPROVED
	submittedTemplates, _, _ := waTemplate.GetRepo().Search("", "", "SUBMITTED", "", "", "name", "ASC", 1, 1000)
	approvedTemplates, _, _ := waTemplate.GetRepo().Search("", "", "APPROVED", "", "", "name", "ASC", 1, 1000)

	// Filter only templates that have WATemplateId
	var templates []waTemplate.WATemplate
	for _, tmpl := range submittedTemplates {
		if tmpl.WATemplateId != "" {
			templates = append(templates, tmpl)
		}
	}
	for _, tmpl := range approvedTemplates {
		if tmpl.WATemplateId != "" {
			templates = append(templates, tmpl)
		}
	}

	return templates
}

// WhatsAppTemplateResponse represents the response from WhatsApp Graph API
type WhatsAppTemplateResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Category string `json:"category"`
	Language string `json:"language"`
}

// checkAndUpdateTemplate checks a single template status from WhatsApp API
func (tsc *TemplateStatusChecker) checkAndUpdateTemplate(template *waTemplate.WATemplate, wabaId, accessToken string) bool {
	customLogger.InfoWithData("Checking template", map[string]interface{}{
		"component":      "TemplateStatusChecker",
		"template_id":    template.ID,
		"template_name":  template.Name,
		"wa_template_id": template.WATemplateId,
	})

	// Fetch template details from WhatsApp API
	url := fmt.Sprintf("%s/%s", config.Get(constans.BaseURLMeta), template.WATemplateId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		customLogger.ErrorWithData("Failed to create request", map[string]interface{}{
			"component":   "TemplateStatusChecker",
			"template_id": template.ID,
			"error":       err.Error(),
		})
		return false
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		customLogger.ErrorWithData("Failed to fetch template from WhatsApp", map[string]interface{}{
			"component":   "TemplateStatusChecker",
			"template_id": template.ID,
			"error":       err.Error(),
		})
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		customLogger.ErrorWithData("Failed to read response", map[string]interface{}{
			"component":   "TemplateStatusChecker",
			"template_id": template.ID,
			"error":       err.Error(),
		})
		return false
	}

	// Handle different HTTP status codes
	if resp.StatusCode == 404 {
		customLogger.WarnWithData("Template not found in WhatsApp (404)", map[string]interface{}{
			"component":      "TemplateStatusChecker",
			"template_id":    template.ID,
			"wa_template_id": template.WATemplateId,
			"status_code":    404,
			"note":           "Template might have been deleted from WhatsApp",
		})
		return false
	}

	if resp.StatusCode != 200 {
		customLogger.WarnWithData("WhatsApp API returned error status", map[string]interface{}{
			"component":   "TemplateStatusChecker",
			"template_id": template.ID,
			"status_code": resp.StatusCode,
			"body":        string(body),
		})

		// Try to parse error message
		var errorResp map[string]interface{}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			if errorMsg, ok := errorResp["error"].(map[string]interface{}); ok {
				customLogger.WarnWithData("Error details from WhatsApp", map[string]interface{}{
					"component":     "TemplateStatusChecker",
					"template_id":   template.ID,
					"error_details": errorMsg,
				})
			}
		}
		return false
	}

	// Parse response
	var waTemplateResp WhatsAppTemplateResponse
	if err := json.Unmarshal(body, &waTemplateResp); err != nil {
		customLogger.ErrorWithData("Failed to parse WhatsApp response", map[string]interface{}{
			"component":     "TemplateStatusChecker",
			"template_id":   template.ID,
			"error":         err.Error(),
			"response_body": string(body),
		})
		return false
	}

	// Validate response has required fields
	if waTemplateResp.ID == "" || waTemplateResp.Name == "" {
		customLogger.ErrorWithData("Invalid response from WhatsApp - missing required fields", map[string]interface{}{
			"component":   "TemplateStatusChecker",
			"template_id": template.ID,
			"response":    waTemplateResp,
		})
		return false
	}

	// Normalize values for comparison
	waStatus := strings.ToUpper(waTemplateResp.Status)
	currentWAStatus := strings.ToUpper(template.WAStatus)
	waCategory := strings.ToUpper(waTemplateResp.Category)
	currentCategory := strings.ToUpper(template.Category)

	customLogger.InfoWithData("WhatsApp API response received", map[string]interface{}{
		"component":     "TemplateStatusChecker",
		"template_id":   template.ID,
		"template_name": waTemplateResp.Name,
		"wa_status":     waTemplateResp.Status,
		"wa_category":   waTemplateResp.Category,
	})

	customLogger.InfoWithData("Current DB values", map[string]interface{}{
		"component":        "TemplateStatusChecker",
		"template_id":      template.ID,
		"current_status":   currentWAStatus,
		"current_category": currentCategory,
	})

	// Check if status or category changed
	statusChanged := waStatus != currentWAStatus
	categoryChanged := waCategory != currentCategory

	// Skip update if nothing changed
	if !statusChanged && !categoryChanged {
		customLogger.DebugWithData("No changes detected", map[string]interface{}{
			"component":   "TemplateStatusChecker",
			"template_id": template.ID,
			"status":      currentWAStatus,
			"category":    currentCategory,
			"action":      "skipping_update",
		})
		return false
	}

	// Log what changed
	if statusChanged {
		customLogger.InfoWithData("Status changed", map[string]interface{}{
			"component":   "TemplateStatusChecker",
			"template_id": template.ID,
			"old_status":  currentWAStatus,
			"new_status":  waStatus,
			"change_type": "status",
		})
	}
	if categoryChanged {
		customLogger.InfoWithData("Category changed", map[string]interface{}{
			"component":    "TemplateStatusChecker",
			"template_id":  template.ID,
			"old_category": currentCategory,
			"new_category": waCategory,
			"change_type":  "category",
		})
	}

	// Update template
	providerAuth := auth.AuthInfo{
		ClientId: common.PROVIDER_ID,
	}

	// If status changed to APPROVED, use Approve service method
	if statusChanged && waStatus == "APPROVED" && template.Status != "APPROVED" {
		_, err := services.GetWATemplateSvc().Approve(template.ID, template.WATemplateId, providerAuth)
		if err != nil {
			customLogger.ErrorWithData("Failed to approve template", map[string]interface{}{
				"component":   "TemplateStatusChecker",
				"template_id": template.ID,
				"error":       err.Message,
				"new_status":  "APPROVED",
			})
			return false
		}
		customLogger.InfoWithData("Template approved successfully", map[string]interface{}{
			"component":   "TemplateStatusChecker",
			"template_id": template.ID,
			"status":      "APPROVED",
		})
	}

	// If status changed to REJECTED, use Reject service method
	if statusChanged && waStatus == "REJECTED" && template.Status != "REJECTED" {
		_, err := services.GetWATemplateSvc().Reject(template.ID, "Rejected by WhatsApp", providerAuth)
		if err != nil {
			customLogger.ErrorWithData("Failed to reject template", map[string]interface{}{
				"component":   "TemplateStatusChecker",
				"template_id": template.ID,
				"error":       err.Message,
				"new_status":  "REJECTED",
			})
			return false
		}
		customLogger.InfoWithData("Template rejected successfully", map[string]interface{}{
			"component":   "TemplateStatusChecker",
			"template_id": template.ID,
			"status":      "REJECTED",
		})
	}

	// If only category changed (or status is already correct), just update the fields
	if categoryChanged {
		// Fetch fresh template data
		freshTemplate := waTemplate.GetRepo().GetById(template.ID)
		if freshTemplate == nil {
			customLogger.ErrorWithData("Template not found", map[string]interface{}{
				"component":   "TemplateStatusChecker",
				"template_id": template.ID,
				"action":      "update_category",
			})
			return false
		}

		freshTemplate.Category = waCategory
		freshTemplate.WAStatus = waStatus

		if err := waTemplate.GetRepo().Update(freshTemplate); err != nil {
			customLogger.ErrorWithData("Failed to update category", map[string]interface{}{
				"component":    "TemplateStatusChecker",
				"template_id":  template.ID,
				"error":        err.Error(),
				"new_category": waCategory,
			})
			return false
		}

		customLogger.InfoWithData("Template category updated", map[string]interface{}{
			"component":   "TemplateStatusChecker",
			"template_id": template.ID,
			"category":    waCategory,
		})
	}

	return true
}
