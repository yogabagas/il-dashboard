package main

import (
	"github.com/ariandi/gocom"
	"gitlab.com/bot3342545/il-dashboard/controllers"
	"gitlab.com/bot3342545/il-dashboard/customer_service/csControllers"
	csRepos "gitlab.com/bot3342545/il-dashboard/customer_service/repositories/csAgent"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/pubsub"
	"gitlab.com/bot3342545/il-dashboard/repositories/customerServiceCases"
	"gitlab.com/bot3342545/il-dashboard/repositories/messageConversation"
	"gitlab.com/bot3342545/il-dashboard/services"
	"gitlab.com/bot3342545/il-dashboard/workers"
)

func main() {

	// Setup both traditional and JSON logging
	if err := customLogger.Setup("logs/il-dashboard"); err != nil {
		customLogger.FatalWithData("Failed to setup logger", map[string]interface{}{
			"component": "main",
			"error":     err.Error(),
		})
	}

	// Initialize repositories (trigger AutoMigrate)
	_ = messageConversation.GetRepo()
	_ = pubsub.GetWhatsappWebhook()
	_ = pubsub.GetMessageSubscribers()

	// Initialize CS repositories (trigger AutoMigrate for CS tables)
	_ = customerServiceCases.GetRepo()
	_ = csRepos.GetRepo()

	// Initialize message routing subscribers
	customLogger.InfoWithData("Initializing message routing subscribers", map[string]interface{}{
		"component": "STARTUP",
	})
	pubsub.InitMessageSubscribers()
	customLogger.InfoWithData("Message routing subscribers initialized successfully", map[string]interface{}{
		"component": "STARTUP",
		"status":    "success",
	})

	// Note: CS pubsub subscribers initialized via GetCSPubSubHandler singleton in controller

	// Auto-refresh AI knowledge cache on startup
	customLogger.InfoWithData("Auto-refreshing AI knowledge cache", map[string]interface{}{
		"component": "STARTUP",
		"action":    "refresh_cache",
	})
	if err := services.GetAIKnowledgeCacheSvc().RefreshAllClients(); err != nil {
		customLogger.ErrorWithData("Failed to refresh AI knowledge cache", map[string]interface{}{
			"component": "STARTUP",
			"error":     err.Message,
			"action":    "refresh_cache",
		})
	} else {
		customLogger.InfoWithData("AI knowledge cache refreshed successfully", map[string]interface{}{
			"component": "STARTUP",
			"action":    "refresh_cache",
			"status":    "success",
		})
	}

	// Start Campaign Executor Worker (background scheduler for campaigns)
	customLogger.InfoWithData("Starting Campaign Executor Worker", map[string]interface{}{
		"component":   "STARTUP",
		"worker_type": "campaign_executor",
		"action":      "starting",
	})
	workers.GetCampaignExecutor().Start()
	customLogger.InfoWithData("Campaign Executor Worker started successfully", map[string]interface{}{
		"component":   "STARTUP",
		"worker_type": "campaign_executor",
		"status":      "running",
	})

	// Start Template Status Checker Worker (checks template approval and category changes every 1 hour)
	customLogger.InfoWithData("Starting Template Status Checker Worker", map[string]interface{}{
		"component":      "STARTUP",
		"worker_type":    "template_status_checker",
		"check_interval": "1 hour",
		"action":         "starting",
	})
	workers.GetTemplateStatusChecker().Start()
	customLogger.InfoWithData("Template Status Checker Worker started successfully", map[string]interface{}{
		"component":   "STARTUP",
		"worker_type": "template_status_checker",
		"status":      "running",
	})

	// Start CS Timeout Checker Worker (auto-close CS conversations after 1 hour inactivity)
	customLogger.InfoWithData("Starting CS Timeout Checker Worker", map[string]interface{}{
		"component":        "STARTUP",
		"worker_type":      "cs_timeout_checker",
		"timeout_duration": "1 hour",
		"action":           "starting",
	})
	workers.GetCSTimeoutChecker().Start()
	customLogger.InfoWithData("CS Timeout Checker Worker started successfully", map[string]interface{}{
		"component":   "STARTUP",
		"worker_type": "cs_timeout_checker",
		"status":      "running",
	})

	gocom.AddCtrl(controllers.GetIndexController())
	gocom.AddCtrl(controllers.GetAuthController())
	gocom.AddCtrl(controllers.GetClientController())
	gocom.AddCtrl(controllers.GetUserController())
	gocom.AddCtrl(controllers.GetWATemplateController())
	gocom.AddCtrl(controllers.GetWASendController())
	gocom.AddCtrl(controllers.GetRoleController())

	// BSP Controllers
	gocom.AddCtrl(controllers.GetContactsController())
	gocom.AddCtrl(controllers.GetContactGroupsController())
	gocom.AddCtrl(controllers.GetSendersController())
	gocom.AddCtrl(controllers.GetCampaignsController())
	gocom.AddCtrl(controllers.GetMessageLogsController())
	gocom.AddCtrl(controllers.GetMessageConversationController())
	gocom.AddCtrl(controllers.GetBillingController())
	gocom.AddCtrl(controllers.GetPricingController())
	gocom.AddCtrl(controllers.GetDashboardController())

	// AI Controllers
	gocom.AddCtrl(controllers.GetAIController())
	gocom.AddCtrl(controllers.GetAIKnowledgeBaseController())

	// Customer Service Controllers
	gocom.AddCtrl(csControllers.GetAgentController())
	gocom.AddCtrl(csControllers.GetCustomerServiceController())

	gocom.Start()
}
