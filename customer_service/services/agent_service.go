package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	"gitlab.com/bot3342545/il-dashboard/customer_service/dtos"
	"gitlab.com/bot3342545/il-dashboard/customer_service/repositories/csAgent"
)

// ============================================================================
// CS Agent Service - Business logic for CS Agent management
// ============================================================================

type CSAgentService interface {
	CreateAgent(req *dtos.CreateAgentRequest) (*dtos.AgentResponse, *gocom.CodedError)
	UpdateAgent(agentID string, req *dtos.UpdateAgentRequest) (*dtos.AgentResponse, *gocom.CodedError)
	DeleteAgent(agentID string) *gocom.CodedError
	GetAgent(agentID string) (*dtos.AgentResponse, *gocom.CodedError)
	GetAgentByUserID(userID string) (*dtos.AgentResponse, *gocom.CodedError)
	GetAgents(filters *dtos.AgentFilters) (*dtos.AgentListResponse, *gocom.CodedError)
	UpdateAvailability(agentID string, req *dtos.UpdateAgentAvailabilityRequest) *gocom.CodedError
	GetAvailableAgents(clientID, skillRequired string) ([]*dtos.AgentAvailabilityResponse, *gocom.CodedError)
	AssignCaseToAgent(clientID, caseID string, strategy *dtos.AssignmentStrategy) (string, *gocom.CodedError)
	UnassignCaseFromAgent(agentID, caseID string) *gocom.CodedError
	UpdateAgentStats(agentID string) *gocom.CodedError
}

type CSAgentServiceImpl struct {
	repo *csAgent.CSAgentRepo
}

var (
	agentSvcInstance *CSAgentServiceImpl
	agentSvcOnce     sync.Once
)

// GetAgentService returns singleton instance of CSAgentService
func GetAgentService() CSAgentService {
	agentSvcOnce.Do(func() {
		agentSvcInstance = &CSAgentServiceImpl{
			repo: csAgent.GetRepo(),
		}
	})
	return agentSvcInstance
}

// CreateAgent creates a new CS agent
func (s *CSAgentServiceImpl) CreateAgent(req *dtos.CreateAgentRequest) (*dtos.AgentResponse, *gocom.CodedError) {
	logger.Infof("[CSAgentService CreateAgent] Creating agent: %s for client: %s", req.Name, req.ClientID)

	// Check if agent already exists for this user
	existingAgent, err := s.repo.GetByUserID(req.UserID)
	if err != nil {
		logger.Errorf("[CSAgentService CreateAgent] Error checking existing agent: %v", err)
		return nil, gocom.NewError(500, "Failed to check existing agent")
	}
	if existingAgent != nil {
		return nil, gocom.NewError(400, "CS Agent already exists for this user")
	}

	// Create agent entity
	agent := &csAgent.CSAgent{
		UserID:             req.UserID,
		ClientID:           req.ClientID,
		Name:               req.Name,
		Email:              req.Email,
		Phone:              req.Phone,
		Status:             "active",
		AvailabilityStatus: "available",
		MaxConcurrentCases: req.MaxConcurrentCases,
		Timezone:           req.Timezone,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Set default max concurrent cases if not provided
	if agent.MaxConcurrentCases == 0 {
		agent.MaxConcurrentCases = 5
	}

	// Set default timezone if not provided
	if agent.Timezone == "" {
		agent.Timezone = "Asia/Jakarta"
	}

	// Set skills
	if len(req.Skills) > 0 {
		if err := agent.SetSkills(req.Skills); err != nil {
			logger.Errorf("[CSAgentService CreateAgent] Error setting skills: %v", err)
			return nil, gocom.NewError(400, "Invalid skills format")
		}
	} else {
		// Default skills
		if err := agent.SetSkills([]string{"general"}); err != nil {
			logger.Errorf("[CSAgentService CreateAgent] Error setting default skills: %v", err)
			return nil, gocom.NewError(500, "Failed to set default skills")
		}
	}

	// Set shift schedule
	if req.ShiftSchedule != nil {
		if err := agent.SetShiftSchedule(req.ShiftSchedule); err != nil {
			logger.Errorf("[CSAgentService CreateAgent] Error setting shift schedule: %v", err)
			return nil, gocom.NewError(400, "Invalid shift schedule format")
		}
	}

	// Save to database
	if err := s.repo.Create(agent); err != nil {
		logger.Errorf("[CSAgentService CreateAgent] Error creating agent: %v", err)
		return nil, gocom.NewError(500, "Failed to create CS agent")
	}

	// Convert to response
	response := s.toAgentResponse(agent)

	logger.Infof("[CSAgentService CreateAgent] Agent created successfully: %s", agent.ID)
	return response, nil
}

// UpdateAgent updates CS agent information
func (s *CSAgentServiceImpl) UpdateAgent(agentID string, req *dtos.UpdateAgentRequest) (*dtos.AgentResponse, *gocom.CodedError) {
	logger.Infof("[CSAgentService UpdateAgent] Updating agent: %s", agentID)

	// Get existing agent
	agent, err := s.repo.GetByID(agentID)
	if err != nil {
		logger.Errorf("[CSAgentService UpdateAgent] Error getting agent: %v", err)
		return nil, gocom.NewError(500, "Failed to get agent")
	}
	if agent == nil {
		return nil, gocom.NewError(404, "CS Agent not found")
	}

	// Update fields
	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.Email != "" {
		agent.Email = req.Email
	}
	if req.Phone != "" {
		agent.Phone = req.Phone
	}
	if req.Status != "" {
		agent.Status = req.Status
	}
	if req.MaxConcurrentCases != nil {
		agent.MaxConcurrentCases = *req.MaxConcurrentCases
	}
	if req.Timezone != "" {
		agent.Timezone = req.Timezone
	}

	// Update skills
	if len(req.Skills) > 0 {
		if err := agent.SetSkills(req.Skills); err != nil {
			logger.Errorf("[CSAgentService UpdateAgent] Error setting skills: %v", err)
			return nil, gocom.NewError(400, "Invalid skills format")
		}
	}

	// Update shift schedule
	if req.ShiftSchedule != nil {
		if err := agent.SetShiftSchedule(req.ShiftSchedule); err != nil {
			logger.Errorf("[CSAgentService UpdateAgent] Error setting shift schedule: %v", err)
			return nil, gocom.NewError(400, "Invalid shift schedule format")
		}
	}

	agent.UpdatedAt = time.Now()

	// Save updates
	if err := s.repo.Update(agent); err != nil {
		logger.Errorf("[CSAgentService UpdateAgent] Error updating agent: %v", err)
		return nil, gocom.NewError(500, "Failed to update CS agent")
	}

	// Convert to response
	response := s.toAgentResponse(agent)

	logger.Infof("[CSAgentService UpdateAgent] Agent updated successfully: %s", agentID)
	return response, nil
}

// DeleteAgent soft deletes CS agent
func (s *CSAgentServiceImpl) DeleteAgent(agentID string) *gocom.CodedError {
	logger.Infof("[CSAgentService DeleteAgent] Deleting agent: %s", agentID)

	// Check if agent exists
	agent, err := s.repo.GetByID(agentID)
	if err != nil {
		logger.Errorf("[CSAgentService DeleteAgent] Error getting agent: %v", err)
		return gocom.NewError(500, "Failed to get agent")
	}
	if agent == nil {
		return gocom.NewError(404, "CS Agent not found")
	}

	// Check if agent has active cases
	if agent.CurrentCasesCount > 0 {
		return gocom.NewError(400, fmt.Sprintf("Cannot delete agent with %d active cases", agent.CurrentCasesCount))
	}

	// Delete agent
	if err := s.repo.Delete(agentID); err != nil {
		logger.Errorf("[CSAgentService DeleteAgent] Error deleting agent: %v", err)
		return gocom.NewError(500, "Failed to delete CS agent")
	}

	logger.Infof("[CSAgentService DeleteAgent] Agent deleted successfully: %s", agentID)
	return nil
}

// GetAgent gets CS agent by ID
func (s *CSAgentServiceImpl) GetAgent(agentID string) (*dtos.AgentResponse, *gocom.CodedError) {
	agent, err := s.repo.GetByID(agentID)
	if err != nil {
		logger.Errorf("[CSAgentService GetAgent] Error getting agent: %v", err)
		return nil, gocom.NewError(500, "Failed to get agent")
	}
	if agent == nil {
		return nil, gocom.NewError(404, "CS Agent not found")
	}

	return s.toAgentResponse(agent), nil
}

// GetAgentByUserID gets CS agent by user ID
func (s *CSAgentServiceImpl) GetAgentByUserID(userID string) (*dtos.AgentResponse, *gocom.CodedError) {
	agent, err := s.repo.GetByUserID(userID)
	if err != nil {
		logger.Errorf("[CSAgentService GetAgentByUserID] Error getting agent: %v", err)
		return nil, gocom.NewError(500, "Failed to get agent")
	}
	if agent == nil {
		return nil, gocom.NewError(404, "CS Agent not found")
	}

	return s.toAgentResponse(agent), nil
}

// GetAgents gets list of CS agents with filters
func (s *CSAgentServiceImpl) GetAgents(filters *dtos.AgentFilters) (*dtos.AgentListResponse, *gocom.CodedError) {
	filterMap := make(map[string]interface{})

	if filters.ClientID != "" {
		filterMap["client_id"] = filters.ClientID
	}
	if filters.Status != "" {
		filterMap["status"] = filters.Status
	}
	if filters.AvailabilityStatus != "" {
		filterMap["availability_status"] = filters.AvailabilityStatus
	}

	agents, err := s.repo.GetAll(filterMap)
	if err != nil {
		logger.Errorf("[CSAgentService GetAgents] Error getting agents: %v", err)
		return nil, gocom.NewError(500, "Failed to get agents")
	}

	// Convert to response
	responses := make([]dtos.AgentResponse, len(agents))
	for i, agent := range agents {
		responses[i] = *s.toAgentResponse(agent)
	}

	return &dtos.AgentListResponse{
		Agents: responses,
		Total:  len(responses),
	}, nil
}

// UpdateAvailability updates CS agent availability status
func (s *CSAgentServiceImpl) UpdateAvailability(agentID string, req *dtos.UpdateAgentAvailabilityRequest) *gocom.CodedError {
	logger.Infof("[CSAgentService UpdateAvailability] Updating availability for agent %s: %s", agentID, req.AvailabilityStatus)

	// Validate availability status
	validStatuses := map[string]bool{
		"available": true,
		"busy":      true,
		"offline":   true,
		"break":     true,
	}
	if !validStatuses[req.AvailabilityStatus] {
		return gocom.NewError(400, "Invalid availability status")
	}

	// Update availability
	if err := s.repo.UpdateAvailability(agentID, req.AvailabilityStatus); err != nil {
		logger.Errorf("[CSAgentService UpdateAvailability] Error updating availability: %v", err)
		return gocom.NewError(500, "Failed to update availability")
	}

	logger.Infof("[CSAgentService UpdateAvailability] Availability updated successfully for agent: %s", agentID)
	return nil
}

// GetAvailableAgents gets list of available CS agents for case assignment
func (s *CSAgentServiceImpl) GetAvailableAgents(clientID, skillRequired string) ([]*dtos.AgentAvailabilityResponse, *gocom.CodedError) {
	agents, err := s.repo.GetAvailableAgents(clientID, skillRequired)
	if err != nil {
		logger.Errorf("[CSAgentService GetAvailableAgents] Error getting available agents: %v", err)
		return nil, gocom.NewError(500, "Failed to get available agents")
	}

	// Convert to response
	responses := make([]*dtos.AgentAvailabilityResponse, len(agents))
	for i, agent := range agents {
		lastActive := time.Now()
		if agent.LastActiveAt != nil {
			lastActive = *agent.LastActiveAt
		}

		responses[i] = &dtos.AgentAvailabilityResponse{
			AgentID:            agent.ID,
			AgentName:          agent.Name,
			Status:             agent.Status,
			AvailabilityStatus: agent.AvailabilityStatus,
			CurrentCasesCount:  agent.CurrentCasesCount,
			MaxConcurrentCases: agent.MaxConcurrentCases,
			CanAcceptNewCase:   agent.IsAvailableForAssignment(),
			LastActiveAt:       lastActive,
		}
	}

	return responses, nil
}

// AssignCaseToAgent assigns a case to an available CS agent
func (s *CSAgentServiceImpl) AssignCaseToAgent(clientID, caseID string, strategy *dtos.AssignmentStrategy) (string, *gocom.CodedError) {
	logger.Infof("[CSAgentService AssignCaseToAgent] Assigning case %s using strategy: %s", caseID, strategy.Type)

	// Get available agents
	agents, err := s.repo.GetAvailableAgents(clientID, "")
	if err != nil {
		logger.Errorf("[CSAgentService AssignCaseToAgent] Error getting available agents: %v", err)
		return "", gocom.NewError(500, "Failed to get available agents")
	}

	if len(agents) == 0 {
		logger.Warnf("[CSAgentService AssignCaseToAgent] No available agents for client: %s", clientID)
		return "", gocom.NewError(503, "No available CS agents at the moment")
	}

	// Select agent based on strategy
	var selectedAgent *csAgent.CSAgent
	switch strategy.Type {
	case "least_loaded":
		selectedAgent = agents[0] // Already sorted by current_cases_count ASC
	case "round_robin", "random":
		selectedAgent = agents[0] // For now, use first available
	default:
		selectedAgent = agents[0]
	}

	// Increment agent's cases count
	if err := s.repo.IncrementCasesCount(selectedAgent.ID); err != nil {
		logger.Errorf("[CSAgentService AssignCaseToAgent] Error incrementing cases count: %v", err)
		return "", gocom.NewError(500, "Failed to assign case to agent")
	}

	logger.Infof("[CSAgentService AssignCaseToAgent] Case %s assigned to agent %s (%s)", caseID, selectedAgent.ID, selectedAgent.Name)
	return selectedAgent.ID, nil
}

// UnassignCaseFromAgent unassigns a case from CS agent
func (s *CSAgentServiceImpl) UnassignCaseFromAgent(agentID, caseID string) *gocom.CodedError {
	logger.Infof("[CSAgentService UnassignCaseFromAgent] Unassigning case %s from agent %s", caseID, agentID)

	// Decrement agent's cases count
	if err := s.repo.DecrementCasesCount(agentID); err != nil {
		logger.Errorf("[CSAgentService UnassignCaseFromAgent] Error decrementing cases count: %v", err)
		return gocom.NewError(500, "Failed to unassign case")
	}

	logger.Infof("[CSAgentService UnassignCaseFromAgent] Case unassigned successfully")
	return nil
}

// UpdateAgentStats updates CS agent performance statistics
func (s *CSAgentServiceImpl) UpdateAgentStats(agentID string) *gocom.CodedError {
	// TODO: Calculate stats from cases and update
	// This will be called periodically or after case resolution
	logger.Infof("[CSAgentService UpdateAgentStats] Updating stats for agent: %s", agentID)

	// Placeholder - implement actual stats calculation
	// Get all cases for this agent
	// Calculate averages
	// Update stats

	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// toAgentResponse converts CSAgent entity to AgentResponse DTO
func (s *CSAgentServiceImpl) toAgentResponse(agent *csAgent.CSAgent) *dtos.AgentResponse {
	return &dtos.AgentResponse{
		ID:                   agent.ID,
		UserID:               agent.UserID,
		ClientID:             agent.ClientID,
		Name:                 agent.Name,
		Email:                agent.Email,
		Phone:                agent.Phone,
		Status:               agent.Status,
		AvailabilityStatus:   agent.AvailabilityStatus,
		CurrentCasesCount:    agent.CurrentCasesCount,
		MaxConcurrentCases:   agent.MaxConcurrentCases,
		Skills:               agent.ParseSkills(),
		ShiftSchedule:        agent.ParseShiftSchedule(),
		Timezone:             agent.Timezone,
		LastActiveAt:         agent.LastActiveAt,
		LastCaseAssignedAt:   agent.LastCaseAssignedAt,
		TotalCasesHandled:    agent.TotalCasesHandled,
		TotalCasesResolved:   agent.TotalCasesResolved,
		AvgResponseTimeMin:   agent.AvgResponseTimeMin,
		AvgResolutionTimeMin: agent.AvgResolutionTimeMin,
		SatisfactionScore:    agent.SatisfactionScore,
		CreatedAt:            agent.CreatedAt,
		UpdatedAt:            agent.UpdatedAt,
	}
}
