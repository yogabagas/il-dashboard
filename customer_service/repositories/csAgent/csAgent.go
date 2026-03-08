package csAgent

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

// ============================================================================
// CS Agent Repository - Manage Customer Service Agents
// ============================================================================

// CSAgent represents a customer service agent/operator
type CSAgent struct {
	ID                   string     `gorm:"type:varchar(50);primaryKey" json:"id"`
	UserID               string     `gorm:"type:varchar(50);not null;index" json:"user_id"` // FK to users table
	ClientID             string     `gorm:"type:varchar(50);not null;index" json:"client_id"`
	Name                 string     `gorm:"type:varchar(255);not null" json:"name"`
	Email                string     `gorm:"type:varchar(255)" json:"email"`
	Phone                string     `gorm:"type:varchar(50)" json:"phone"`
	Status               string     `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`           // active, inactive, offline, busy, away
	AvailabilityStatus   string     `gorm:"type:varchar(20);not null;default:'available'" json:"availability_status"` // available, busy, offline, break
	CurrentCasesCount    int        `gorm:"default:0" json:"current_cases_count"`
	MaxConcurrentCases   int        `gorm:"default:5" json:"max_concurrent_cases"`
	Skills               string     `gorm:"type:text" json:"skills"`         // JSON: ["billing", "technical", "general", "complaint"]
	ShiftSchedule        string     `gorm:"type:text" json:"shift_schedule"` // JSON: {"monday": {"start": "09:00", "end": "17:00"}, ...}
	Timezone             string     `gorm:"type:varchar(50);default:'Asia/Jakarta'" json:"timezone"`
	LastActiveAt         *time.Time `json:"last_active_at"`
	LastCaseAssignedAt   *time.Time `json:"last_case_assigned_at"`
	TotalCasesHandled    int        `gorm:"default:0" json:"total_cases_handled"`
	TotalCasesResolved   int        `gorm:"default:0" json:"total_cases_resolved"`
	AvgResponseTimeMin   float64    `gorm:"default:0" json:"avg_response_time_min"`   // in minutes
	AvgResolutionTimeMin float64    `gorm:"default:0" json:"avg_resolution_time_min"` // in minutes
	SatisfactionScore    float64    `gorm:"default:0" json:"satisfaction_score"`      // 0-5 rating
	Metadata             string     `gorm:"type:text" json:"metadata"`                // JSON metadata
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	DeletedAt            *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName specifies the table name for CSAgent
func (CSAgent) TableName() string {
	return "cs_agents"
}

// BeforeCreate hook to generate ID
func (a *CSAgent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = ulid.Make().String()
	}
	return nil
}

// CSAgentRepo repository implementation
type CSAgentRepo struct {
	gocom.BaseRepo
}

var (
	repoInstance *CSAgentRepo
	repoOnce     sync.Once
)

// GetRepo returns singleton instance of CSAgentRepo
func GetRepo() *CSAgentRepo {
	repoOnce.Do(func() {
		repoInstance = &CSAgentRepo{}
		// Auto migrate
		if err := repoInstance.AutoMigrate(&CSAgent{}); err != nil {
			logger.Errorf("[CSAgentRepo] Failed to auto migrate: %v", err)
		} else {
			logger.Info("[CSAgentRepo] Auto migration completed successfully")
		}
	})
	return repoInstance
}

// Create creates a new CS agent
func (r *CSAgentRepo) Create(agent *CSAgent) error {
	if err := r.BaseRepo.Create(agent).Error; err != nil {
		logger.Errorf("[CSAgentRepo Create] Error: %v", err)
		return err
	}

	logger.Infof("[CSAgentRepo Create] CS Agent created successfully: %s (%s)", agent.Name, agent.ID)
	return nil
}

// Update updates CS agent
func (r *CSAgentRepo) Update(agent *CSAgent) error {
	if err := r.BaseRepo.Update(agent).Error; err != nil {
		logger.Errorf("[CSAgentRepo Update] Error: %v", err)
		return err
	}

	logger.Infof("[CSAgentRepo Update] CS Agent updated successfully: %s", agent.ID)
	return nil
}

// Delete soft deletes CS agent
func (r *CSAgentRepo) Delete(id string) error {
	if err := r.Where("id = ?", id).Delete(&CSAgent{}).Error; err != nil {
		logger.Errorf("[CSAgentRepo Delete] Error: %v", err)
		return err
	}

	logger.Infof("[CSAgentRepo Delete] CS Agent deleted successfully: %s", id)
	return nil
}

// GetByID gets CS agent by ID
func (r *CSAgentRepo) GetByID(id string) (*CSAgent, error) {
	var agent CSAgent
	if err := r.Where("id = ?", id).First(&agent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.Errorf("[CSAgentRepo GetByID] Error: %v", err)
		return nil, err
	}
	return &agent, nil
}

// GetByUserID gets CS agent by user ID
func (r *CSAgentRepo) GetByUserID(userID string) (*CSAgent, error) {
	var agent CSAgent
	if err := r.Where("user_id = ?", userID).First(&agent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.Errorf("[CSAgentRepo GetByUserID] Error: %v", err)
		return nil, err
	}
	return &agent, nil
}

// GetByClientID gets all CS agents for a client
func (r *CSAgentRepo) GetByClientID(clientID string) ([]*CSAgent, error) {
	var agents []*CSAgent
	if err := r.Where("client_id = ?", clientID).Order("name ASC").Find(&agents).Error; err != nil {
		logger.Errorf("[CSAgentRepo GetByClientID] Error: %v", err)
		return nil, err
	}
	return agents, nil
}

// GetAvailableAgents gets available CS agents for case assignment
func (r *CSAgentRepo) GetAvailableAgents(clientID string, skillRequired string) ([]*CSAgent, error) {
	var agents []*CSAgent

	query := r.Where("client_id = ? AND status = ? AND availability_status = ?",
		clientID, "active", "available").Debug()

	// Filter by skill if required
	if skillRequired != "" {
		query = query.Where("skills LIKE ?", "%"+skillRequired+"%")
	}

	// Get agents with capacity (current_cases_count < max_concurrent_cases)
	query = query.Where("current_cases_count < max_concurrent_cases")

	// Order by load (least loaded first)
	query = query.Order("current_cases_count ASC, last_case_assigned_at ASC")

	if err := query.Find(&agents).Error; err != nil {
		logger.Errorf("[CSAgentRepo GetAvailableAgents] Error: %v", err)
		return nil, err
	}

	return agents, nil
}

// UpdateStatus updates CS agent status
func (r *CSAgentRepo) UpdateStatus(id, status string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":         status,
		"last_active_at": now,
		"updated_at":     now,
	}

	if err := r.Model(&CSAgent{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		logger.Errorf("[CSAgentRepo UpdateStatus] Error: %v", err)
		return err
	}

	logger.Infof("[CSAgentRepo UpdateStatus] Status updated for agent %s: %s", id, status)
	return nil
}

// UpdateAvailability updates CS agent availability status
func (r *CSAgentRepo) UpdateAvailability(id, availabilityStatus string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"availability_status": availabilityStatus,
		"last_active_at":      now,
		"updated_at":          now,
	}

	if err := r.Model(&CSAgent{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		logger.Errorf("[CSAgentRepo UpdateAvailability] Error: %v", err)
		return err
	}

	logger.Infof("[CSAgentRepo UpdateAvailability] Availability updated for agent %s: %s", id, availabilityStatus)
	return nil
}

// IncrementCasesCount increments current cases count
func (r *CSAgentRepo) IncrementCasesCount(id string) error {
	now := time.Now()
	if err := r.Model(&CSAgent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"current_cases_count":   gorm.Expr("current_cases_count + ?", 1),
			"last_case_assigned_at": now,
			"updated_at":            now,
		}).Error; err != nil {
		logger.Errorf("[CSAgentRepo IncrementCasesCount] Error: %v", err)
		return err
	}

	logger.Infof("[CSAgentRepo IncrementCasesCount] Cases count incremented for agent: %s", id)
	return nil
}

// DecrementCasesCount decrements current cases count
func (r *CSAgentRepo) DecrementCasesCount(id string) error {
	if err := r.Model(&CSAgent{}).
		Where("id = ? AND current_cases_count > 0", id).
		Updates(map[string]interface{}{
			"current_cases_count": gorm.Expr("current_cases_count - ?", 1),
			"updated_at":          time.Now(),
		}).Error; err != nil {
		logger.Errorf("[CSAgentRepo DecrementCasesCount] Error: %v", err)
		return err
	}

	logger.Infof("[CSAgentRepo DecrementCasesCount] Cases count decremented for agent: %s", id)
	return nil
}

// UpdateStats updates CS agent performance statistics
func (r *CSAgentRepo) UpdateStats(id string, totalHandled, totalResolved int, avgResponseTime, avgResolutionTime, satisfactionScore float64) error {
	updates := map[string]interface{}{
		"total_cases_handled":     totalHandled,
		"total_cases_resolved":    totalResolved,
		"avg_response_time_min":   avgResponseTime,
		"avg_resolution_time_min": avgResolutionTime,
		"satisfaction_score":      satisfactionScore,
		"updated_at":              time.Now(),
	}

	if err := r.Model(&CSAgent{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		logger.Errorf("[CSAgentRepo UpdateStats] Error: %v", err)
		return err
	}

	logger.Infof("[CSAgentRepo UpdateStats] Stats updated for agent: %s", id)
	return nil
}

// GetAll gets all CS agents with optional filters
func (r *CSAgentRepo) GetAll(filters map[string]interface{}) ([]*CSAgent, error) {
	var agents []*CSAgent

	query := r.Model(&CSAgent{})

	// Apply filters
	if clientID, ok := filters["client_id"].(string); ok && clientID != "" {
		query = query.Where("client_id = ?", clientID)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if availabilityStatus, ok := filters["availability_status"].(string); ok && availabilityStatus != "" {
		query = query.Where("availability_status = ?", availabilityStatus)
	}

	if err := query.Order("name ASC").Find(&agents).Error; err != nil {
		logger.Errorf("[CSAgentRepo GetAll] Error: %v", err)
		return nil, err
	}

	return agents, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// ParseSkills parses skills JSON to string slice
func (a *CSAgent) ParseSkills() []string {
	if a.Skills == "" {
		return []string{}
	}

	var skills []string
	if err := json.Unmarshal([]byte(a.Skills), &skills); err != nil {
		logger.Errorf("[CSAgent ParseSkills] Error: %v", err)
		return []string{}
	}
	return skills
}

// SetSkills sets skills from string slice
func (a *CSAgent) SetSkills(skills []string) error {
	skillsJSON, err := json.Marshal(skills)
	if err != nil {
		return err
	}
	a.Skills = string(skillsJSON)
	return nil
}

// ParseShiftSchedule parses shift schedule JSON
func (a *CSAgent) ParseShiftSchedule() map[string]interface{} {
	if a.ShiftSchedule == "" {
		return map[string]interface{}{}
	}

	var schedule map[string]interface{}
	if err := json.Unmarshal([]byte(a.ShiftSchedule), &schedule); err != nil {
		logger.Errorf("[CSAgent ParseShiftSchedule] Error: %v", err)
		return map[string]interface{}{}
	}
	return schedule
}

// SetShiftSchedule sets shift schedule from map
func (a *CSAgent) SetShiftSchedule(schedule map[string]interface{}) error {
	scheduleJSON, err := json.Marshal(schedule)
	if err != nil {
		return err
	}
	a.ShiftSchedule = string(scheduleJSON)
	return nil
}

// IsAvailableForAssignment checks if agent is available for new case assignment
func (a *CSAgent) IsAvailableForAssignment() bool {
	return a.Status == "active" &&
		a.AvailabilityStatus == "available" &&
		a.CurrentCasesCount < a.MaxConcurrentCases
}

// HasSkill checks if agent has specific skill
func (a *CSAgent) HasSkill(skill string) bool {
	skills := a.ParseSkills()
	for _, s := range skills {
		if s == skill {
			return true
		}
	}
	return false
}
