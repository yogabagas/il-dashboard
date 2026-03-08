package dtos

import "time"

// ============================================================================
// Pubsub Topic Names - untuk komunikasi microservice
// ============================================================================

const (
	// Topics untuk CS Case Events
	TopicCSCaseCreated         = "cs.case.created"          // Ketika case baru dibuat
	TopicCSCaseClaimed         = "cs.case.claimed"          // Ketika CS claim case
	TopicCSCaseEscalated       = "cs.case.escalated"        // Ketika case di-escalate
	TopicCSCaseResolved        = "cs.case.resolved"         // Ketika case resolved
	TopicCSCaseClosed          = "cs.case.closed"           // Ketika case closed (trigger billing)
	TopicCSCaseMessageSent     = "cs.case.message.sent"     // Ketika CS kirim message
	TopicCSCaseMessageReceived = "cs.case.message.received" // Ketika user reply ke CS case
	TopicCSCaseSLABreached     = "cs.case.sla.breached"     // Ketika SLA breach

	// Topics untuk AI Escalation
	TopicAIEscalationRequest   = "ai.escalation.request"   // AI request escalate ke CS
	TopicAIEscalationConfirmed = "ai.escalation.confirmed" // User confirm escalation
	TopicAIEscalationRejected  = "ai.escalation.rejected"  // User reject escalation

	// Topics untuk Message Routing (critical untuk distinguish type)
	TopicUserMessageReceived   = "user.message.received"   // Raw message dari user
	TopicUserMessageClassified = "user.message.classified" // Message sudah di-classify
)

// ============================================================================
// Message Classification - CRITICAL untuk distinguish message type
// ============================================================================

// MessageClassification represents classified user message
type MessageClassification struct {
	MessageID      string `json:"message_id"`
	SessionID      string `json:"session_id"`
	ClientId       string `json:"client_id"`
	UserPhone      string `json:"user_phone"`
	UserName       string `json:"user_name,omitempty"`
	MessageContent string `json:"message_content"`
	MessageType    string `json:"message_type"` // text, image, document, button, etc

	// Classification Results - CRITICAL
	Classification string  `json:"classification"` // initial_question, gangguan, inquiry, payment, campaign, other
	Confidence     float64 `json:"confidence"`     // 0.0 - 1.0
	IsPDAMRelated  bool    `json:"is_pdam_related"`
	IsGreeting     bool    `json:"is_greeting"`
	IsEscalation   bool    `json:"is_escalation"`   // Apakah perlu escalate ke CS
	IsPaymentFlow  bool    `json:"is_payment_flow"` // Apakah part dari payment flow
	IsCampaign     bool    `json:"is_campaign"`     // Apakah dari campaign

	// Context
	HasActiveCSCase    bool   `json:"has_active_cs_case"` // Apakah user punya active CS case
	ActiveCaseID       string `json:"active_case_id,omitempty"`
	ConversationStatus string `json:"conversation_status"` // active, pending_close, closed

	// Metadata
	Keywords       []string               `json:"keywords,omitempty"`
	DetectedIntent string                 `json:"detected_intent,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	ClassifiedAt   time.Time              `json:"classified_at"`
}

// ============================================================================
// CS Case Events - untuk pubsub communication
// ============================================================================

// CSCaseCreatedEvent published when new CS case is created
type CSCaseCreatedEvent struct {
	CaseID      string                 `json:"case_id"`
	CaseNumber  string                 `json:"case_number"`
	ClientId    string                 `json:"client_id"`
	UserPhone   string                 `json:"user_phone"`
	UserName    string                 `json:"user_name,omitempty"`
	Category    string                 `json:"category"` // initial_question, gangguan, inquiry_payment, campaign
	Subject     string                 `json:"subject"`
	Status      string                 `json:"status"` // always "new" for this event
	Severity    string                 `json:"severity"`
	Priority    int                    `json:"priority"`
	SlaDeadline time.Time              `json:"sla_deadline"`
	SessionID   string                 `json:"session_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// CSCaseClaimedEvent published when CS agent claims a case
type CSCaseClaimedEvent struct {
	CaseID         string    `json:"case_id"`
	CaseNumber     string    `json:"case_number"`
	ClientId       string    `json:"client_id"`
	AssignedTo     string    `json:"assigned_to"`
	AssignedToName string    `json:"assigned_to_name"`
	Severity       string    `json:"severity"`
	Priority       int       `json:"priority"`
	ClaimedAt      time.Time `json:"claimed_at"`
}

// CSCaseEscalatedEvent published when case is escalated
type CSCaseEscalatedEvent struct {
	CaseID           string    `json:"case_id"`
	CaseNumber       string    `json:"case_number"`
	ClientId         string    `json:"client_id"`
	FromUserId       string    `json:"from_user_id"`
	FromUserName     string    `json:"from_user_name"`
	EscalatedTo      string    `json:"escalated_to"`
	EscalatedToName  string    `json:"escalated_to_name"`
	EscalationReason string    `json:"escalation_reason"`
	OldSeverity      string    `json:"old_severity"`
	NewSeverity      string    `json:"new_severity"`
	EscalatedAt      time.Time `json:"escalated_at"`
}

// CSCaseResolvedEvent published when case is resolved
type CSCaseResolvedEvent struct {
	CaseID          string    `json:"case_id"`
	CaseNumber      string    `json:"case_number"`
	ClientId        string    `json:"client_id"`
	ResolvedBy      string    `json:"resolved_by"`
	ResolvedByName  string    `json:"resolved_by_name"`
	ResolutionNotes string    `json:"resolution_notes"`
	SessionDuration int       `json:"session_duration"` // seconds
	TotalMessages   int       `json:"total_messages"`
	ResolvedAt      time.Time `json:"resolved_at"`
}

// CSCaseClosedEvent published when case is closed - TRIGGERS BILLING
type CSCaseClosedEvent struct {
	CaseID            string `json:"case_id"`
	CaseNumber        string `json:"case_number"`
	ClientId          string `json:"client_id"`
	UserPhone         string `json:"user_phone"`
	ClosedBy          string `json:"closed_by"`
	ClosedByName      string `json:"closed_by_name"`
	CloseNotes        string `json:"close_notes,omitempty"`
	SessionDuration   int    `json:"session_duration"` // seconds
	TotalMessages     int    `json:"total_messages"`
	CsMessages        int    `json:"cs_messages"`
	UserMessages      int    `json:"user_messages"`
	FirstResponseTime int    `json:"first_response_time"` // seconds

	// Billing Info - CRITICAL
	BillingCost      float64 `json:"billing_cost"`
	BillingCategory  string  `json:"billing_category"` // cs_conversation
	BillingReference string  `json:"billing_reference,omitempty"`

	ClosedAt time.Time `json:"closed_at"`
}

// CSCaseMessageSentEvent published when CS sends message to user
type CSCaseMessageSentEvent struct {
	MessageID         string    `json:"message_id"`
	CaseID            string    `json:"case_id"`
	CaseNumber        string    `json:"case_number"`
	ClientId          string    `json:"client_id"`
	UserPhone         string    `json:"user_phone"`
	SenderId          string    `json:"sender_id"`
	SenderName        string    `json:"sender_name"`
	MessageContent    string    `json:"message_content"`
	MessageType       string    `json:"message_type"` // text, image, document
	WhatsappMessageId string    `json:"whatsapp_message_id,omitempty"`
	IsFirstResponse   bool      `json:"is_first_response"`
	SentAt            time.Time `json:"sent_at"`
}

// CSCaseMessageReceivedEvent published when user replies to CS case
type CSCaseMessageReceivedEvent struct {
	MessageID      string    `json:"message_id"`
	CaseID         string    `json:"case_id"`
	CaseNumber     string    `json:"case_number"`
	ClientId       string    `json:"client_id"`
	UserPhone      string    `json:"user_phone"`
	UserName       string    `json:"user_name,omitempty"`
	MessageContent string    `json:"message_content"`
	MessageType    string    `json:"message_type"`
	ReceivedAt     time.Time `json:"received_at"`
}

// CSCaseSLABreachedEvent published when SLA is breached
type CSCaseSLABreachedEvent struct {
	CaseID         string    `json:"case_id"`
	CaseNumber     string    `json:"case_number"`
	ClientId       string    `json:"client_id"`
	UserPhone      string    `json:"user_phone"`
	Status         string    `json:"status"`
	Severity       string    `json:"severity"`
	AssignedTo     string    `json:"assigned_to,omitempty"`
	AssignedToName string    `json:"assigned_to_name,omitempty"`
	SlaDeadline    time.Time `json:"sla_deadline"`
	WaitTime       int64     `json:"wait_time"` // seconds
	BreachedAt     time.Time `json:"breached_at"`
}

// ============================================================================
// AI Escalation Events
// ============================================================================

// AIEscalationRequestEvent published when AI wants to escalate to CS
type AIEscalationRequestEvent struct {
	SessionID    string                 `json:"session_id"`
	ClientId     string                 `json:"client_id"`
	UserPhone    string                 `json:"user_phone"`
	UserName     string                 `json:"user_name,omitempty"`
	UserMessage  string                 `json:"user_message"`
	AIResponse   string                 `json:"ai_response"`
	Reason       string                 `json:"reason"` // knowledge_gap, low_confidence, etc
	AIConfidence float64                `json:"ai_confidence"`
	Category     string                 `json:"category"` // auto-detected category
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	RequestedAt  time.Time              `json:"requested_at"`
}

// AIEscalationConfirmedEvent published when user confirms escalation
type AIEscalationConfirmedEvent struct {
	SessionID      string                 `json:"session_id"`
	ClientId       string                 `json:"client_id"`
	UserPhone      string                 `json:"user_phone"`
	UserName       string                 `json:"user_name,omitempty"`
	UserMessage    string                 `json:"user_message"`    // Original message that needs CS
	ConfirmMessage string                 `json:"confirm_message"` // User's "Ya" message
	Category       string                 `json:"category"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	ConfirmedAt    time.Time              `json:"confirmed_at"`
}

// AIEscalationRejectedEvent published when user rejects escalation
type AIEscalationRejectedEvent struct {
	SessionID     string    `json:"session_id"`
	ClientId      string    `json:"client_id"`
	UserPhone     string    `json:"user_phone"`
	RejectMessage string    `json:"reject_message"` // User's "Tidak" message
	RejectedAt    time.Time `json:"rejected_at"`
}

// ============================================================================
// Helper Functions
// ============================================================================

// GetCategoryFromClassification returns category based on classification
func GetCategoryFromClassification(classification string) string {
	switch classification {
	case "initial_question":
		return "pdam_inquiry"
	case "gangguan":
		return "pdam_complaint"
	case "inquiry", "payment":
		return "pdam_inquiry"
	case "campaign":
		return "campaign"
	default:
		return "pdam_complaint"
	}
}

// IsCriticalCategory checks if category requires immediate attention
func IsCriticalCategory(category string) bool {
	criticalCategories := map[string]bool{
		"gangguan":       true,
		"pdam_complaint": true,
		"urgent":         true,
	}
	return criticalCategories[category]
}

// GetPriorityFromClassification returns priority based on classification
func GetPriorityFromClassification(classification string, confidence float64) int {
	if classification == "gangguan" {
		return 2 // High priority
	}
	if confidence < 0.4 {
		return 2 // Low confidence = higher priority
	}
	return 3 // Normal priority
}

// GetSeverityFromClassification returns severity based on classification
func GetSeverityFromClassification(classification string) string {
	if classification == "gangguan" {
		return "high"
	}
	return "normal"
}
