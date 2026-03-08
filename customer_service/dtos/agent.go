package dtos

import "time"

// ============================================================================
// CS Agent DTOs - Data Transfer Objects for CS Agents
// ============================================================================

// CreateAgentRequest represents request to create a new CS agent
type CreateAgentRequest struct {
	UserID             string                 `json:"user_id" binding:"required"`
	ClientID           string                 `json:"client_id" binding:"required"`
	Name               string                 `json:"name" binding:"required"`
	Email              string                 `json:"email"`
	Phone              string                 `json:"phone"`
	MaxConcurrentCases int                    `json:"max_concurrent_cases"` // default: 5
	Skills             []string               `json:"skills"`               // ["billing", "technical", "general", "complaint"]
	ShiftSchedule      map[string]interface{} `json:"shift_schedule"`       // {"monday": {"start": "09:00", "end": "17:00"}, ...}
	Timezone           string                 `json:"timezone"`             // default: Asia/Jakarta
}

// UpdateAgentRequest represents request to update CS agent
type UpdateAgentRequest struct {
	Name               string                 `json:"name"`
	Email              string                 `json:"email"`
	Phone              string                 `json:"phone"`
	Status             string                 `json:"status"` // active, inactive
	MaxConcurrentCases *int                   `json:"max_concurrent_cases"`
	Skills             []string               `json:"skills"`
	ShiftSchedule      map[string]interface{} `json:"shift_schedule"`
	Timezone           string                 `json:"timezone"`
}

// UpdateAgentAvailabilityRequest represents request to update agent availability
type UpdateAgentAvailabilityRequest struct {
	AvailabilityStatus string `json:"availability_status" binding:"required"` // available, busy, offline, break
}

// AgentResponse represents CS agent response
type AgentResponse struct {
	ID                   string                 `json:"id"`
	UserID               string                 `json:"user_id"`
	ClientID             string                 `json:"client_id"`
	Name                 string                 `json:"name"`
	Email                string                 `json:"email"`
	Phone                string                 `json:"phone"`
	Status               string                 `json:"status"`
	AvailabilityStatus   string                 `json:"availability_status"`
	CurrentCasesCount    int                    `json:"current_cases_count"`
	MaxConcurrentCases   int                    `json:"max_concurrent_cases"`
	Skills               []string               `json:"skills"`
	ShiftSchedule        map[string]interface{} `json:"shift_schedule,omitempty"`
	Timezone             string                 `json:"timezone"`
	LastActiveAt         *time.Time             `json:"last_active_at,omitempty"`
	LastCaseAssignedAt   *time.Time             `json:"last_case_assigned_at,omitempty"`
	TotalCasesHandled    int                    `json:"total_cases_handled"`
	TotalCasesResolved   int                    `json:"total_cases_resolved"`
	AvgResponseTimeMin   float64                `json:"avg_response_time_min"`
	AvgResolutionTimeMin float64                `json:"avg_resolution_time_min"`
	SatisfactionScore    float64                `json:"satisfaction_score"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// AgentListResponse represents list of CS agents
type AgentListResponse struct {
	Agents []AgentResponse `json:"agents"`
	Total  int             `json:"total"`
}

// AgentPerformanceResponse represents agent performance metrics
type AgentPerformanceResponse struct {
	AgentID              string    `json:"agent_id"`
	AgentName            string    `json:"agent_name"`
	Period               string    `json:"period"` // today, week, month
	TotalCasesHandled    int       `json:"total_cases_handled"`
	TotalCasesResolved   int       `json:"total_cases_resolved"`
	TotalCasesEscalated  int       `json:"total_cases_escalated"`
	AvgResponseTimeMin   float64   `json:"avg_response_time_min"`
	AvgResolutionTimeMin float64   `json:"avg_resolution_time_min"`
	SatisfactionScore    float64   `json:"satisfaction_score"`
	FirstResponseRate    float64   `json:"first_response_rate"` // % of cases with first response
	ResolutionRate       float64   `json:"resolution_rate"`     // % of cases resolved
	OnlineTimeHours      float64   `json:"online_time_hours"`
	IdleTimeHours        float64   `json:"idle_time_hours"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// AgentAvailabilityResponse represents agent availability status
type AgentAvailabilityResponse struct {
	AgentID            string    `json:"agent_id"`
	AgentName          string    `json:"agent_name"`
	Status             string    `json:"status"`
	AvailabilityStatus string    `json:"availability_status"`
	CurrentCasesCount  int       `json:"current_cases_count"`
	MaxConcurrentCases int       `json:"max_concurrent_cases"`
	CanAcceptNewCase   bool      `json:"can_accept_new_case"`
	LastActiveAt       time.Time `json:"last_active_at"`
}

// AgentFilters represents filters for querying agents
type AgentFilters struct {
	ClientID           string `form:"client_id"`
	Status             string `form:"status"`              // active, inactive, offline, busy, away
	AvailabilityStatus string `form:"availability_status"` // available, busy, offline, break
	Skill              string `form:"skill"`               // Filter by specific skill
}

// AssignmentStrategy represents strategy for auto-assignment
type AssignmentStrategy struct {
	Type     string                 `json:"type"` // round_robin, least_loaded, skill_based, random
	Criteria map[string]interface{} `json:"criteria,omitempty"`
}
