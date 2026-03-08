package workers

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gitlab.com/bot3342545/il-dashboard/constans"
	csServices "gitlab.com/bot3342545/il-dashboard/customer_service/services"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/repositories/customerServiceCases"
	"gitlab.com/bot3342545/il-dashboard/repositories/messageConversation"
	"gitlab.com/bot3342545/il-dashboard/services"
)

type CSTimeoutChecker struct {
	stopChan chan bool
	wg       sync.WaitGroup
}

var csTimeoutChecker *CSTimeoutChecker
var onceCSTimeoutChecker sync.Once

func GetCSTimeoutChecker() *CSTimeoutChecker {
	onceCSTimeoutChecker.Do(func() {
		csTimeoutChecker = &CSTimeoutChecker{
			stopChan: make(chan bool),
		}
	})
	return csTimeoutChecker
}

// Start begins the CS timeout checker worker
func (ctc *CSTimeoutChecker) Start() {
	customLogger.InfoWithData("Starting CS timeout checker worker", map[string]interface{}{
		"component": "CSTimeoutChecker",
		"action":    "start",
	})
	ctc.wg.Add(1)
	go ctc.run()
}

// Stop gracefully stops the CS timeout checker worker
func (ctc *CSTimeoutChecker) Stop() {
	customLogger.InfoWithData("Stopping CS timeout checker worker", map[string]interface{}{
		"component": "CSTimeoutChecker",
		"action":    "stop",
	})
	close(ctc.stopChan)
	ctc.wg.Wait()
	customLogger.InfoWithData("CS timeout checker worker stopped", map[string]interface{}{
		"component": "CSTimeoutChecker",
		"status":    "stopped",
	})
}

// run is the main worker loop - checks every 10 minutes
func (ctc *CSTimeoutChecker) run() {
	defer ctc.wg.Done()

	// Check every 10 minutes
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	customLogger.InfoWithData("Worker started", map[string]interface{}{
		"component":      "CSTimeoutChecker",
		"check_interval": "10 minutes",
		"status":         "running",
	})

	// Run immediately on startup
	ctc.checkTimeouts()

	for {
		select {
		case <-ctc.stopChan:
			customLogger.InfoWithData("Received stop signal", map[string]interface{}{
				"component": "CSTimeoutChecker",
				"action":    "stopping",
			})
			return
		case <-ticker.C:
			ctc.checkTimeouts()
		}
	}
}

// checkTimeouts checks for cases that have been inactive for 1 hour
func (ctc *CSTimeoutChecker) checkTimeouts() {
	customLogger.InfoWithData("Starting timeout check cycle", map[string]interface{}{
		"component": "CSTimeoutChecker",
		"action":    "check_cycle_start",
	})

	// Step 1: Find cases inactive for 1 hour (send warning)
	oneHourThreshold := time.Now().Add(-1 * time.Hour)
	timedOutCases := customerServiceCases.GetRepo().GetTimedOutCases(oneHourThreshold)

	warningCount := 0
	closedCount := 0

	for _, caseData := range timedOutCases {
		// Check metadata for warning timestamp
		var metadata map[string]interface{}
		if caseData.Metadata != "" {
			json.Unmarshal([]byte(caseData.Metadata), &metadata)
		} else {
			metadata = make(map[string]interface{})
		}

		warningSentAt, hasWarning := metadata["timeout_warning_sent_at"].(string)

		if !hasWarning {
			// Step 1: Send warning (1 hour timeout)
			if ctc.sendTimeoutWarning(&caseData, metadata) {
				warningCount++
			}
		} else {
			// Step 2: Check if 5 minutes passed since warning
			warningTime, err := time.Parse(time.RFC3339, warningSentAt)
			if err == nil && time.Since(warningTime) >= 5*time.Minute {
				// Close case after 5 minutes from warning
				if ctc.closeTimedOutCase(&caseData) {
					closedCount++
				}
			} else {
				customLogger.DebugWithData("Warning sent, waiting for grace period", map[string]interface{}{
					"component":   "CSTimeoutChecker",
					"case_number": caseData.CaseNumber,
					"grace_period": "5 minutes",
				})
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	customLogger.InfoWithData("Timeout check completed", map[string]interface{}{
		"component":      "CSTimeoutChecker",
		"warnings_sent":  warningCount,
		"cases_closed":   closedCount,
	})
}

// sendTimeoutWarning sends warning message after 1 hour of inactivity
func (ctc *CSTimeoutChecker) sendTimeoutWarning(caseData *customerServiceCases.CustomerServiceCase, metadata map[string]interface{}) bool {
	customLogger.InfoWithData("Sending timeout warning", map[string]interface{}{
		"component":   "CSTimeoutChecker",
		"case_id":     caseData.ID,
		"case_number": caseData.CaseNumber,
		"user_phone":  caseData.UserPhone,
	})

	// Warning message
	warningMessage := constans.CSTimeoutWarningMessage

	_, _, err := services.GetWASendSvc().SendTextMessageWithLog(
		caseData.UserPhone,
		warningMessage,
		caseData.ClientId,
		"", // businessPhone not available in worker - will use fallback config
	)

	if err != nil {
		customLogger.ErrorWithData("Failed to send warning", map[string]interface{}{
			"component":   "CSTimeoutChecker",
			"case_number": caseData.CaseNumber,
			"error":       err.Message,
			"user_phone":  caseData.UserPhone,
		})
		return false
	}

	// Save warning timestamp to metadata
	metadata["timeout_warning_sent_at"] = time.Now().Format(time.RFC3339)
	metadataJSON, _ := json.Marshal(metadata)
	caseData.Metadata = string(metadataJSON)

	if err := customerServiceCases.GetRepo().Update(caseData); err != nil {
		customLogger.ErrorWithData("Failed to update metadata", map[string]interface{}{
			"component":   "CSTimeoutChecker",
			"case_number": caseData.CaseNumber,
			"error":       err.Error(),
		})
		return false
	}

	customLogger.InfoWithData("Warning sent successfully", map[string]interface{}{
		"component":   "CSTimeoutChecker",
		"case_number": caseData.CaseNumber,
		"user_phone":  caseData.UserPhone,
		"status":      "warning_sent",
	})
	return true
}

// closeTimedOutCase closes case after grace period (5 minutes after warning)
func (ctc *CSTimeoutChecker) closeTimedOutCase(caseData *customerServiceCases.CustomerServiceCase) bool {
	customLogger.InfoWithData("Closing timed out case", map[string]interface{}{
		"component":   "CSTimeoutChecker",
		"case_id":     caseData.ID,
		"case_number": caseData.CaseNumber,
		"user_phone":  caseData.UserPhone,
	})

	// Send closing message
	closingMessage := constans.CSTimeoutMessage

	_, _, err := services.GetWASendSvc().SendTextMessageWithLog(
		caseData.UserPhone,
		closingMessage,
		caseData.ClientId,
		"", // businessPhone not available in worker - will use fallback config
	)

	if err != nil {
		customLogger.ErrorWithData("Failed to send closing message", map[string]interface{}{
			"component":   "CSTimeoutChecker",
			"case_number": caseData.CaseNumber,
			"error":       err.Message,
			"user_phone":  caseData.UserPhone,
		})
		// Continue with closing even if message fails
	} else {
		customLogger.InfoWithData("Closing message sent", map[string]interface{}{
			"component":   "CSTimeoutChecker",
			"case_number": caseData.CaseNumber,
			"user_phone":  caseData.UserPhone,
		})
	}

	// Close the case
	closeErr := csServices.GetCustomerServiceSvc().CloseCase(
		caseData.ID,
		"SYSTEM",
		fmt.Sprintf("Auto-closed due to inactivity (1 hour + 5 min grace period). Last activity: %s",
			caseData.LastMessageAt.Format("2006-01-02 15:04:05")),
	)

	if closeErr != nil {
		customLogger.ErrorWithData("Failed to close case", map[string]interface{}{
			"component":   "CSTimeoutChecker",
			"case_number": caseData.CaseNumber,
			"error":       closeErr.Message,
		})
		return false
	}

	// Close the conversation
	if err := messageConversation.GetRepo().CloseConversation(caseData.UserPhone); err != nil {
		customLogger.ErrorWithData("Failed to close conversation", map[string]interface{}{
			"component":  "CSTimeoutChecker",
			"user_phone": caseData.UserPhone,
			"error":      err.Error(),
		})
	} else {
		customLogger.InfoWithData("Conversation closed", map[string]interface{}{
			"component":  "CSTimeoutChecker",
			"user_phone": caseData.UserPhone,
		})
	}

	customLogger.InfoWithData("Case auto-closed successfully", map[string]interface{}{
		"component":   "CSTimeoutChecker",
		"case_number": caseData.CaseNumber,
		"case_id":     caseData.ID,
		"user_phone":  caseData.UserPhone,
		"status":      "auto_closed",
	})
	return true
}
