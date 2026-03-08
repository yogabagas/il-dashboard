package dtos

import "time"

// CaseRequest represents request to create a new CS case
type CaseRequest struct {
	ClientId    string                 `json:"client_id"`
	UserPhone   string                 `json:"user_phone"`
	UserName    string                 `json:"user_name,omitempty"`
	Category    string                 `json:"category"` // initial_question, gangguan, inquiry_payment, campaign, other
	Subject     string                 `json:"subject"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
}

// ClaimCaseRequest represents request to claim a case
type ClaimCaseRequest struct {
	Severity string `json:"severity"` // low, normal, high, urgent
	Priority int    `json:"priority"` // 1=Urgent, 2=High, 3=Normal, 4=Low
	Notes    string `json:"notes,omitempty"`
}

// SendMessageRequest represents request to send message to user
type SendMessageRequest struct {
	MessageContent string `json:"message_content" binding:"required"`
	MessageType    string `json:"message_type,omitempty"` // text, image, document
}

// AddNoteRequest represents request to add internal note
type AddNoteRequest struct {
	NoteType    string `json:"note_type"` // general, resolution, escalation, internal
	NoteContent string `json:"note_content" binding:"required"`
	IsInternal  bool   `json:"is_internal"`
}

// ResolveCaseRequest represents request to resolve case
type ResolveCaseRequest struct {
	ResolutionNotes string `json:"resolution_notes" binding:"required"`
}

// CloseCaseRequest represents request to close case
type CloseCaseRequest struct {
	CloseNotes string `json:"close_notes,omitempty"`
}

// EscalateCaseRequest represents request to escalate case
type EscalateCaseRequest struct {
	EscalateTo       string `json:"escalate_to" binding:"required"`
	EscalationReason string `json:"escalation_reason" binding:"required"`
	Severity         string `json:"severity"`
	Priority         int    `json:"priority"`
}

// CaseResponse represents single case response
type CaseResponse struct {
	ID                string                 `json:"id"`
	CaseNumber        string                 `json:"case_number"`
	ClientId          string                 `json:"client_id"`
	UserPhone         string                 `json:"user_phone"`
	UserName          string                 `json:"user_name,omitempty"`
	Category          string                 `json:"category"`
	Subject           string                 `json:"subject"`
	Description       string                 `json:"description,omitempty"`
	Status            string                 `json:"status"`
	Severity          string                 `json:"severity"`
	Priority          int                    `json:"priority"`
	AssignedTo        string                 `json:"assigned_to,omitempty"`
	AssignedToName    string                 `json:"assigned_to_name,omitempty"`
	AssignedAt        *time.Time             `json:"assigned_at,omitempty"`
	SlaDeadline       time.Time              `json:"sla_deadline"`
	SlaBreached       bool                   `json:"sla_breached"`
	FirstResponseAt   *time.Time             `json:"first_response_at,omitempty"`
	FirstResponseTime *int                   `json:"first_response_time,omitempty"`
	ResolvedAt        *time.Time             `json:"resolved_at,omitempty"`
	ClosedAt          *time.Time             `json:"closed_at,omitempty"`
	TotalMessages     int                    `json:"total_messages"`
	CsMessages        int                    `json:"cs_messages"`
	UserMessages      int                    `json:"user_messages"`
	SessionDuration   *int                   `json:"session_duration,omitempty"`
	LastMessageAt     *time.Time             `json:"last_message_at,omitempty"`
	LastMessageFrom   string                 `json:"last_message_from,omitempty"`
	WaitTimeSeconds   int64                  `json:"wait_time_seconds"`
	WaitTimeFormatted string                 `json:"wait_time_formatted"`
	Tags              []string               `json:"tags,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	BillingCost       float64                `json:"billing_cost"`
	BillingReference  string                 `json:"billing_reference,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// CaseDetailsResponse represents detailed case with conversation
type CaseDetailsResponse struct {
	Case                *CaseResponse        `json:"case"`
	ConversationHistory []MessageResponse    `json:"conversation_history"`
	Notes               []NoteResponse       `json:"notes"`
	Assignments         []AssignmentResponse `json:"assignments"`
}

// MessageResponse represents CS message
type MessageResponse struct {
	ID                string     `json:"id"`
	CaseId            string     `json:"case_id"`
	SenderType        string     `json:"sender_type"` // user or cs
	SenderId          string     `json:"sender_id,omitempty"`
	SenderName        string     `json:"sender_name,omitempty"`
	MessageType       string     `json:"message_type"`
	MessageContent    string     `json:"message_content"`
	WhatsappMessageId string     `json:"whatsapp_message_id,omitempty"`
	Status            string     `json:"status"`
	ErrorMessage      string     `json:"error_message,omitempty"`
	SentAt            time.Time  `json:"sent_at"`
	DeliveredAt       *time.Time `json:"delivered_at,omitempty"`
	ReadAt            *time.Time `json:"read_at,omitempty"`
}

// NoteResponse represents internal CS note
type NoteResponse struct {
	ID            string    `json:"id"`
	CaseId        string    `json:"case_id"`
	NoteType      string    `json:"note_type"`
	NoteContent   string    `json:"note_content"`
	CreatedBy     string    `json:"created_by"`
	CreatedByName string    `json:"created_by_name"`
	IsInternal    bool      `json:"is_internal"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AssignmentResponse represents assignment history
type AssignmentResponse struct {
	ID              string    `json:"id"`
	CaseId          string    `json:"case_id"`
	Action          string    `json:"action"` // claimed, released, escalated, reassigned
	FromUserId      string    `json:"from_user_id,omitempty"`
	FromUserName    string    `json:"from_user_name,omitempty"`
	ToUserId        string    `json:"to_user_id,omitempty"`
	ToUserName      string    `json:"to_user_name,omitempty"`
	SeverityBefore  string    `json:"severity_before,omitempty"`
	SeverityAfter   string    `json:"severity_after,omitempty"`
	Notes           string    `json:"notes,omitempty"`
	PerformedBy     string    `json:"performed_by"`
	PerformedByName string    `json:"performed_by_name"`
	PerformedAt     time.Time `json:"performed_at"`
}

// CasesListResponse represents list of cases with pagination
type CasesListResponse struct {
	Cases      []CaseResponse `json:"cases"`
	Pagination PaginationInfo `json:"pagination"`
	Stats      CaseStats      `json:"stats"`
}

// PaginationInfo represents pagination information
type PaginationInfo struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

// CaseStats represents case statistics
type CaseStats struct {
	NewCount         int `json:"new_count"`
	AssignedCount    int `json:"assigned_count"`
	ResolvedCount    int `json:"resolved_count"`
	ClosedCount      int `json:"closed_count"`
	EscalatedCount   int `json:"escalated_count"`
	SlaBreachedCount int `json:"sla_breached_count"`
}

// CaseFilters represents filters for querying cases
type CaseFilters struct {
	ClientId    string `form:"client_id"`
	Status      string `form:"status"`   // new, assigned, resolved, escalated, closed
	Severity    string `form:"severity"` // low, normal, high, urgent
	Category    string `form:"category"` // initial_question, gangguan, inquiry_payment, campaign
	AssignedTo  string `form:"assigned_to"`
	SlaBreached *bool  `form:"sla_breached"`
	Page        int    `form:"page"`
	Limit       int    `form:"limit"`
	Sort        string `form:"sort"`       // created_at, severity, sla_deadline
	SortOrder   string `form:"sort_order"` // asc, desc
}

// StatisticsResponse represents dashboard statistics
type StatisticsResponse struct {
	Overview    StatsOverview    `json:"overview"`
	Performance StatsPerformance `json:"performance"`
	TopCSAgents []TopCSAgent     `json:"top_cs_agents"`
	Breakdown   StatsBreakdown   `json:"breakdown"`
}

// StatsOverview represents overview stats
type StatsOverview struct {
	TotalCases     int `json:"total_cases"`
	NewCases       int `json:"new_cases"`
	AssignedCases  int `json:"assigned_cases"`
	ResolvedCases  int `json:"resolved_cases"`
	ClosedCases    int `json:"closed_cases"`
	EscalatedCases int `json:"escalated_cases"`
	SlaBreached    int `json:"sla_breached"`
}

// StatsPerformance represents performance metrics
type StatsPerformance struct {
	AvgFirstResponseTime int     `json:"average_first_response_time"` // seconds
	AvgResolutionTime    int     `json:"average_resolution_time"`     // seconds
	AvgSessionDuration   int     `json:"average_session_duration"`    // seconds
	CasesPerCS           float64 `json:"cases_per_cs"`
	MessagesPerCase      float64 `json:"messages_per_case"`
}

// TopCSAgent represents top performing CS agent
type TopCSAgent struct {
	UserId               string  `json:"user_id"`
	Name                 string  `json:"name"`
	CasesHandled         int     `json:"cases_handled"`
	CasesResolved        int     `json:"cases_resolved"`
	AvgResponseTime      int     `json:"average_response_time"`
	CustomerSatisfaction float64 `json:"customer_satisfaction,omitempty"`
}

// StatsBreakdown represents breakdown by category/severity
type StatsBreakdown struct {
	BySeverity map[string]int `json:"by_severity"`
	ByCategory map[string]int `json:"by_category"`
	ByStatus   map[string]int `json:"by_status"`
}
