package csControllers

import (
	"sync"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	a "gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/customer_service/dtos"
	"gitlab.com/bot3342545/il-dashboard/customer_service/services"
)

// ============================================================================
// CS Agent Controller - API endpoints for CS Agent management
// ============================================================================

type CSAgentController struct {
	gocom.Controller
	service services.CSAgentService
}

var (
	agentCtrlInstance *CSAgentController
	agentCtrlOnce     sync.Once
)

// GetAgentController returns singleton instance of CSAgentController
func GetAgentController() *CSAgentController {
	agentCtrlOnce.Do(func() {
		agentCtrlInstance = &CSAgentController{
			service: services.GetAgentService(),
		}
	})
	return agentCtrlInstance
}

// Init initializes routes for CS agent controller
func (ctrl *CSAgentController) Init() {
	logger.Info("[CSAgentController] Initializing routes...")

	// Availability management
	gocom.POST("/api/v1/cs/agents/availability/update", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), ctrl.updateAvailability)
	gocom.GET("/api/v1/cs/agents/available", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), ctrl.getAvailableAgents)

	// CS Agent Management Routes - require ADMIN or OWNER role
	gocom.POST("/api/v1/cs/agents/create", a.Allow(a.Role(a.ADMIN, a.OWNER)), ctrl.createAgent)
	gocom.GET("/api/v1/cs/agents/list", a.Allow(a.Role(a.ADMIN, a.OWNER)), ctrl.getAgents)
	gocom.GET("/api/v1/cs/agents/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), ctrl.getAgent)
	gocom.POST("/api/v1/cs/agents/update", a.Allow(a.Role(a.ADMIN, a.OWNER)), ctrl.updateAgent)
	gocom.POST("/api/v1/cs/agents/delete", a.Allow(a.Role(a.ADMIN, a.OWNER)), ctrl.deleteAgent)

	logger.Info("[CSAgentController] Routes initialized successfully")
}

// createAgent creates a new CS agent
// POST /api/v1/cs/agents/create
func (ctrl *CSAgentController) createAgent(ctx gocom.Context) error {
	logger.Info("[CSAgentController createAgent] Creating new CS agent")

	var req dtos.CreateAgentRequest
	if err := ctx.Bind(&req); err != nil {
		logger.Errorf("[CSAgentController createAgent] Invalid request: %v", err)
		return ctx.SendError(gocom.NewError(400, "Invalid request: "+err.Error()))
	}

	// Create agent
	agent, codedErr := ctrl.service.CreateAgent(&req)
	if codedErr != nil {
		logger.Errorf("[CSAgentController createAgent] Error: %s", codedErr.Message)
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult(agent)
}

// getAgents gets list of CS agents with filters
// GET /api/v1/cs/agents/list?client_id=xxx&status=active
func (ctrl *CSAgentController) getAgents(ctx gocom.Context) error {
	logger.Info("[CSAgentController getAgents] Getting CS agents list")

	clientID := ctx.Query("client_id")
	status := ctx.Query("status")
	availabilityStatus := ctx.Query("availability_status")

	filters := &dtos.AgentFilters{
		ClientID:           clientID,
		Status:             status,
		AvailabilityStatus: availabilityStatus,
	}

	// Get agents
	response, codedErr := ctrl.service.GetAgents(filters)
	if codedErr != nil {
		logger.Errorf("[CSAgentController getAgents] Error: %s", codedErr.Message)
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult(response)
}

// getAgent gets CS agent by ID
// GET /api/v1/cs/agents/:id
func (ctrl *CSAgentController) getAgent(ctx gocom.Context) error {
	agentID := ctx.Param("id")
	logger.Infof("[CSAgentController getAgent] Getting CS agent: %s", agentID)

	if agentID == "" {
		return ctx.SendError(gocom.NewError(400, "agent_id is required"))
	}

	agent, codedErr := ctrl.service.GetAgent(agentID)
	if codedErr != nil {
		logger.Errorf("[CSAgentController getAgent] Error: %s", codedErr.Message)
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult(agent)
}

// updateAgent updates CS agent information
// POST /api/v1/cs/agents/update
func (ctrl *CSAgentController) updateAgent(ctx gocom.Context) error {
	agentID := ctx.Query("agent_id")
	logger.Infof("[CSAgentController updateAgent] Updating CS agent: %s", agentID)

	if agentID == "" {
		return ctx.SendError(gocom.NewError(400, "agent_id is required"))
	}

	var req dtos.UpdateAgentRequest
	if err := ctx.Bind(&req); err != nil {
		logger.Errorf("[CSAgentController updateAgent] Invalid request: %v", err)
		return ctx.SendError(gocom.NewError(400, "Invalid request: "+err.Error()))
	}

	agent, codedErr := ctrl.service.UpdateAgent(agentID, &req)
	if codedErr != nil {
		logger.Errorf("[CSAgentController updateAgent] Error: %s", codedErr.Message)
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult(agent)
}

// deleteAgent deletes CS agent
// POST /api/v1/cs/agents/delete
func (ctrl *CSAgentController) deleteAgent(ctx gocom.Context) error {
	agentID := ctx.Query("agent_id")
	logger.Infof("[CSAgentController deleteAgent] Deleting CS agent: %s", agentID)

	if agentID == "" {
		return ctx.SendError(gocom.NewError(400, "agent_id is required"))
	}

	codedErr := ctrl.service.DeleteAgent(agentID)
	if codedErr != nil {
		logger.Errorf("[CSAgentController deleteAgent] Error: %s", codedErr.Message)
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult("CS Agent deleted successfully")
}

// updateAvailability updates CS agent availability status
// POST /api/v1/cs/agents/availability/update
func (ctrl *CSAgentController) updateAvailability(ctx gocom.Context) error {
	agentID := ctx.Query("agent_id")
	logger.Infof("[CSAgentController updateAvailability] Updating availability for agent: %s", agentID)

	if agentID == "" {
		return ctx.SendError(gocom.NewError(400, "agent_id is required"))
	}

	var req dtos.UpdateAgentAvailabilityRequest
	if err := ctx.Bind(&req); err != nil {
		logger.Errorf("[CSAgentController updateAvailability] Invalid request: %v", err)
		return ctx.SendError(gocom.NewError(400, "Invalid request: "+err.Error()))
	}

	codedErr := ctrl.service.UpdateAvailability(agentID, &req)
	if codedErr != nil {
		logger.Errorf("[CSAgentController updateAvailability] Error: %s", codedErr.Message)
		return ctx.SendError(codedErr)
	}

	return ctx.SendResult("Availability updated successfully")
}

// getAvailableAgents gets list of available CS agents
// GET /api/v1/cs/agents/available?client_id=xxx&skill=billing
func (ctrl *CSAgentController) getAvailableAgents(ctx gocom.Context) error {
	clientID := ctx.Query("client_id")
	skill := ctx.Query("skill")

	logger.Infof("[CSAgentController getAvailableAgents] Getting available agents for client: %s, skill: %s", clientID, skill)

	if clientID == "" {
		return ctx.SendError(gocom.NewError(400, "client_id is required"))
	}

	agents, codedErr := ctrl.service.GetAvailableAgents(clientID, skill)
	if codedErr != nil {
		logger.Errorf("[CSAgentController getAvailableAgents] Error: %s", codedErr.Message)
		return ctx.SendError(codedErr)
	}

	result := map[string]interface{}{
		"agents": agents,
		"total":  len(agents),
	}

	return ctx.SendResult(result)
}
