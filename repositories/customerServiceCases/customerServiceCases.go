package customerServiceCases

import (
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	"github.com/oklog/ulid/v2"
	"gitlab.com/bot3342545/il-dashboard/customer_service/dtos"
	"gorm.io/gorm"
)

// ============================================================================
// Models
// ============================================================================

type CustomerServiceCase struct {
	ID                    string  `gorm:"primaryKey;size:26"`
	ClientId              string  `gorm:"size:26;not null;index:idx_client_status"`
	UserPhone             string  `gorm:"size:20;not null;index:idx_user_phone"`
	UserName              string  `gorm:"size:255"`
	CaseNumber            string  `gorm:"size:50;not null;uniqueIndex"`
	Category              string  `gorm:"size:50;default:'pdam_complaint'"`
	Subject               string  `gorm:"type:text"`
	Description           string  `gorm:"type:text"`
	Status                string  `gorm:"size:50;default:'new';index:idx_client_status,idx_status_created"`
	Severity              string  `gorm:"size:20;default:'normal'"`
	Priority              int     `gorm:"default:3"`
	AssignedTo            *string `gorm:"size:26;index:idx_assigned_to"`
	AssignedAt            *time.Time
	EscalatedTo           *string `gorm:"size:26"`
	EscalatedAt           *time.Time
	EscalationReason      string `gorm:"type:text"`
	ResolvedAt            *time.Time
	ResolutionNotes       string `gorm:"type:text"`
	ClosedAt              *time.Time
	SlaDeadline           time.Time `gorm:"not null;index:idx_sla_breach"`
	SlaBreached           bool      `gorm:"default:false;index:idx_sla_breach"`
	FirstResponseAt       *time.Time
	FirstResponseTime     *int
	TotalMessages         int `gorm:"default:0"`
	CsMessages            int `gorm:"default:0"`
	UserMessages          int `gorm:"default:0"`
	SessionDuration       *int
	ConversationSessionId string `gorm:"size:26"`
	LastMessageAt         *time.Time
	LastMessageFrom       *string   `gorm:"size:20"`
	BillingCost           float64   `gorm:"type:decimal(10,2);default:0.00"`
	BillingReference      *string   `gorm:"size:26"`
	Tags                  string    `gorm:"type:json"`
	Metadata              string    `gorm:"type:json"`
	CreatedAt             time.Time `gorm:"autoCreateTime;index:idx_status_created"`
	UpdatedAt             time.Time `gorm:"autoUpdateTime"`
}

func (c *CustomerServiceCase) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = ulid.Make().String()
	}
	return nil
}

type CustomerServiceMessage struct {
	ID                string    `gorm:"primaryKey;size:26"`
	CaseId            string    `gorm:"size:26;not null;index:idx_case_id"`
	SenderType        string    `gorm:"size:10;not null"` // user or cs
	SenderId          *string   `gorm:"size:26"`
	SenderName        string    `gorm:"size:255"`
	MessageType       string    `gorm:"size:20;default:'text'"`
	MessageContent    string    `gorm:"type:text;not null"`
	WhatsappMessageId *string   `gorm:"size:100;index"`
	MessageLogId      *string   `gorm:"size:26;index"`
	Status            string    `gorm:"size:20;default:'sent'"`
	ErrorMessage      string    `gorm:"type:text"`
	SentAt            time.Time `gorm:"autoCreateTime;index:idx_case_id"`
	DeliveredAt       *time.Time
	ReadAt            *time.Time
}

func (c *CustomerServiceMessage) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = ulid.Make().String()
	}
	return nil
}

type CustomerServiceAssignment struct {
	ID             string    `gorm:"primaryKey;size:26"`
	CaseId         string    `gorm:"size:26;not null;index:idx_case_id"`
	Action         string    `gorm:"size:50;not null"` // claimed, released, escalated, reassigned
	FromUserId     *string   `gorm:"size:26"`
	ToUserId       *string   `gorm:"size:26;index:idx_to_user"`
	SeverityBefore string    `gorm:"size:20"`
	SeverityAfter  string    `gorm:"size:20"`
	Notes          string    `gorm:"type:text"`
	PerformedBy    string    `gorm:"size:26;not null"`
	PerformedAt    time.Time `gorm:"autoCreateTime;index:idx_case_id"`
}

func (c *CustomerServiceAssignment) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = ulid.Make().String()
	}
	return nil
}

type CustomerServiceNote struct {
	ID            string    `gorm:"primaryKey;size:26"`
	CaseId        string    `gorm:"size:26;not null;index:idx_case_id"`
	NoteType      string    `gorm:"size:50;default:'general'"`
	NoteContent   string    `gorm:"type:text;not null"`
	CreatedBy     string    `gorm:"size:26;not null;index"`
	CreatedByName string    `gorm:"size:255"`
	IsInternal    bool      `gorm:"default:true"`
	CreatedAt     time.Time `gorm:"autoCreateTime;index:idx_case_id"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

func (c *CustomerServiceNote) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = ulid.Make().String()
	}
	return nil
}

// ============================================================================
// Repository
// ============================================================================

type CustomerServiceCaseRepo struct {
	gocom.BaseRepo
}

// Create creates a new case
func (r *CustomerServiceCaseRepo) Create(caseData *CustomerServiceCase) error {
	return r.BaseRepo.Create(caseData).Error
}

// GetById returns case by ID
func (r *CustomerServiceCaseRepo) GetById(id string) *CustomerServiceCase {
	ret := &CustomerServiceCase{}
	err := r.Where("id = ?", id).First(ret).Error
	if err != nil {
		return nil
	}
	return ret
}

// GetByCaseNumber returns case by case number
func (r *CustomerServiceCaseRepo) GetByCaseNumber(caseNumber string) *CustomerServiceCase {
	ret := &CustomerServiceCase{}
	err := r.Where("case_number = ?", caseNumber).First(ret).Error
	if err != nil {
		return nil
	}
	return ret
}

// GetActiveCase returns active case for user (NEW, PENDING_ESCALATION, ASSIGNED, RESOLVED)
func (r *CustomerServiceCaseRepo) GetActiveCase(clientId, userPhone string) *CustomerServiceCase {
	ret := &CustomerServiceCase{}
	err := r.Where("client_id = ? AND user_phone = ? AND status IN (?)",
		clientId,
		userPhone,
		[]string{"new", "pending_escalation", "assigned", "resolved"},
	).Order("created_at DESC").First(ret).Error

	if err != nil {
		return nil
	}
	return ret
}

// Update updates case
func (r *CustomerServiceCaseRepo) Update(caseData *CustomerServiceCase) error {
	return r.BaseRepo.Update(caseData).Error
}

// GetByStatus returns cases by status
func (r *CustomerServiceCaseRepo) GetByStatus(clientId, status string, limit int) []CustomerServiceCase {
	ret := []CustomerServiceCase{}
	tx := r.Where("client_id = ? AND status = ?", clientId, status).Order("created_at DESC")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	tx.Find(&ret)
	return ret
}

// GetByFilters returns cases with filters and pagination
func (r *CustomerServiceCaseRepo) GetByFilters(filters *dtos.CaseFilters) ([]CustomerServiceCase, int) {
	ret := []CustomerServiceCase{}
	query := r.Model(&CustomerServiceCase{})

	// Apply filters
	if filters.ClientId != "" {
		query = query.Where("client_id = ?", filters.ClientId)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.Severity != "" {
		query = query.Where("severity = ?", filters.Severity)
	}
	if filters.Category != "" {
		query = query.Where("category = ?", filters.Category)
	}
	if filters.AssignedTo != "" {
		query = query.Where("assigned_to = ?", filters.AssignedTo)
	}
	if filters.SlaBreached != nil {
		query = query.Where("sla_breached = ?", *filters.SlaBreached)
	}

	// Count total
	var total int64
	query.Model(&CustomerServiceCase{}).Count(&total)

	// Apply sorting
	sortField := "created_at"
	if filters.Sort != "" {
		sortField = filters.Sort
	}
	sortOrder := "DESC"
	if filters.SortOrder == "asc" {
		sortOrder = "ASC"
	}
	query = query.Order(sortField + " " + sortOrder)

	// Apply pagination
	page := filters.Page
	if page <= 0 {
		page = 1
	}
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	query.Offset(offset).Limit(limit).Find(&ret)

	return ret, int(total)
}

// GetSLABreachedCases returns cases with breached SLA
func (r *CustomerServiceCaseRepo) GetSLABreachedCases() []CustomerServiceCase {
	ret := []CustomerServiceCase{}
	r.Where("sla_breached = ? AND sla_deadline < ? AND status IN (?)",
		false,
		time.Now(),
		[]string{"new", "assigned", "pending_escalation"},
	).Find(&ret)
	return ret
}

// GetTimedOutCases returns cases that have been inactive for more than the timeout threshold
func (r *CustomerServiceCaseRepo) GetTimedOutCases(timeoutThreshold time.Time) []CustomerServiceCase {
	ret := []CustomerServiceCase{}
	r.Where("status = ? AND last_message_at < ? AND last_message_at IS NOT NULL",
		"assigned",
		timeoutThreshold,
	).Order("last_message_at ASC").Find(&ret)
	return ret
}

// GetTodayCount returns count of cases created today for case number generation
func (r *CustomerServiceCaseRepo) GetTodayCount(clientId string) int {
	var count int64
	today := time.Now().Format("2006-01-02")
	r.Where("client_id = ? AND DATE(created_at) = ?", clientId, today).
		Model(&CustomerServiceCase{}).
		Count(&count)
	return int(count)
}

// GetStats returns case statistics
func (r *CustomerServiceCaseRepo) GetStats(clientId string) map[string]int {
	stats := make(map[string]int)

	// Count by status
	statuses := []string{"new", "assigned", "resolved", "closed", "escalated"}
	for _, status := range statuses {
		var count int64
		r.Where("client_id = ? AND status = ?", clientId, status).
			Model(&CustomerServiceCase{}).
			Count(&count)
		stats[status] = int(count)
	}

	// Count SLA breached
	var breachedCount int64
	r.Where("client_id = ? AND sla_breached = ?", clientId, true).
		Model(&CustomerServiceCase{}).
		Count(&breachedCount)
	stats["sla_breached"] = int(breachedCount)

	return stats
}

// ============================================================================
// Message Repository Methods
// ============================================================================

// CreateMessage creates a new message
func (r *CustomerServiceCaseRepo) CreateMessage(message *CustomerServiceMessage) error {
	return r.BaseRepo.Create(message).Error
}

// GetMessages returns messages for a case
func (r *CustomerServiceCaseRepo) GetMessages(caseId string) []CustomerServiceMessage {
	ret := []CustomerServiceMessage{}
	r.Where("case_id = ?", caseId).Order("sent_at ASC").Find(&ret)
	return ret
}

// GetMessageById returns message by ID
func (r *CustomerServiceCaseRepo) GetMessageById(id string) *CustomerServiceMessage {
	ret := &CustomerServiceMessage{}
	err := r.Where("id = ?", id).First(ret).Error
	if err != nil {
		return nil
	}
	return ret
}

// GetMessageByWhatsAppMessageId returns message by WhatsApp message ID
func (r *CustomerServiceCaseRepo) GetMessageByWhatsAppMessageId(waMessageId string) *CustomerServiceMessage {
	ret := &CustomerServiceMessage{}
	err := r.Where("whatsapp_message_id = ?", waMessageId).First(ret).Error
	if err != nil {
		return nil
	}
	return ret
}

// UpdateMessage updates message
func (r *CustomerServiceCaseRepo) UpdateMessage(message *CustomerServiceMessage) error {
	return r.BaseRepo.Update(message).Error
}

// ============================================================================
// Assignment Repository Methods
// ============================================================================

// CreateAssignment creates assignment record (audit trail)
func (r *CustomerServiceCaseRepo) CreateAssignment(assignment *CustomerServiceAssignment) error {
	return r.BaseRepo.Create(assignment).Error
}

// GetAssignments returns assignments for a case
func (r *CustomerServiceCaseRepo) GetAssignments(caseId string) []CustomerServiceAssignment {
	ret := []CustomerServiceAssignment{}
	r.Where("case_id = ?", caseId).Order("performed_at ASC").Find(&ret)
	return ret
}

// ============================================================================
// Note Repository Methods
// ============================================================================

// CreateNote creates a new note
func (r *CustomerServiceCaseRepo) CreateNote(note *CustomerServiceNote) error {
	return r.BaseRepo.Create(note).Error
}

// GetNotes returns notes for a case
func (r *CustomerServiceCaseRepo) GetNotes(caseId string) []CustomerServiceNote {
	ret := []CustomerServiceNote{}
	r.Where("case_id = ?", caseId).Order("created_at ASC").Find(&ret)
	return ret
}

// GetNoteById returns note by ID
func (r *CustomerServiceCaseRepo) GetNoteById(id string) *CustomerServiceNote {
	ret := &CustomerServiceNote{}
	err := r.Where("id = ?", id).First(ret).Error
	if err != nil {
		return nil
	}
	return ret
}

// UpdateNote updates note
func (r *CustomerServiceCaseRepo) UpdateNote(note *CustomerServiceNote) error {
	return r.BaseRepo.Update(note).Error
}

// ============================================================================
// Singleton
// ============================================================================

var (
	customerServiceCaseRepo     *CustomerServiceCaseRepo
	customerServiceCaseRepoOnce sync.Once
)

func GetRepo() *CustomerServiceCaseRepo {
	if customerServiceCaseRepo == nil {
		customerServiceCaseRepoOnce.Do(func() {
			logger.Infof("[CustomerServiceCaseRepo] Initializing repository...")
			customerServiceCaseRepo = &CustomerServiceCaseRepo{}

			// Auto migrate tables
			_ = customerServiceCaseRepo.AutoMigrate(CustomerServiceCase{})
			_ = customerServiceCaseRepo.AutoMigrate(CustomerServiceMessage{})
			_ = customerServiceCaseRepo.AutoMigrate(CustomerServiceAssignment{})
			_ = customerServiceCaseRepo.AutoMigrate(CustomerServiceNote{})
			logger.Infof("[CustomerServiceCaseRepo] Tables migrated successfully")
		})
	}
	return customerServiceCaseRepo
}
