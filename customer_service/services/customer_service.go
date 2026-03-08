package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	"github.com/oklog/ulid/v2"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/bot3342545/il-dashboard/customer_service/dtos"
	"gitlab.com/bot3342545/il-dashboard/repositories/customerServiceCases"
	"gitlab.com/bot3342545/il-dashboard/repositories/messageConversation"
	mainServices "gitlab.com/bot3342545/il-dashboard/services"
)

// ============================================================================
// Customer Service Interface
// ============================================================================

type CustomerServiceSvc interface {
	CreateCase(clientId, userPhone, userName, sessionId, category, subject string) (*dtos.CaseResponse, *gocom.CodedError)
	ClaimCase(caseId, csUserId, severity string, priority int, notes string) (*dtos.CaseResponse, *gocom.CodedError)
	SendMessage(caseId, csUserId, message string) *gocom.CodedError
	SaveUserMessage(caseId, userPhone, message, messageType string) *gocom.CodedError
	AddNote(caseId, csUserId, noteType, content string, isInternal bool) *gocom.CodedError
	ResolveCase(caseId, csUserId, resolutionNotes string) *gocom.CodedError
	CloseCase(caseId, csUserId, closeNotes string) *gocom.CodedError
	EscalateCase(caseId, csUserId, escalateTo, reason string, severity string, priority int) *gocom.CodedError
	GetCaseDetails(caseId string) (*dtos.CaseDetailsResponse, *gocom.CodedError)
	GetCasesList(filters *dtos.CaseFilters) (*dtos.CasesListResponse, *gocom.CodedError)
	GetStatistics(clientId string, period string) (*dtos.StatisticsResponse, *gocom.CodedError)
}

type CustomerServiceSvcImpl struct{}

// ============================================================================
// Case Management
// ============================================================================

func (s *CustomerServiceSvcImpl) CreateCase(
	clientId, userPhone, userName, sessionId, category, subject string,
) (*dtos.CaseResponse, *gocom.CodedError) {
	logger.Infof("[CustomerServiceSvc CreateCase] clientId=%s userPhone=%s category=%s", clientId, userPhone, category)

	// Validation
	if clientId == "" || userPhone == "" {
		logger.Warnf("[CustomerServiceSvc CreateCase] Missing required fields")
		return nil, common.ERR_INVALID_REQUEST
	}

	// Generate case number: CS-YYYYMMDD-NNNN
	today := time.Now().Format("20060102")
	dailyCount := customerServiceCases.GetRepo().GetTodayCount(clientId) + 1
	caseNumber := fmt.Sprintf("CS-%s-%04d", today, dailyCount)

	// Get conversation history for context - filter by sessionId to get only current conversation
	var history []messageConversation.MessageConversation
	if sessionId != "" {
		history = messageConversation.GetRepo().GetConversationHistoryBySession(sessionId, 10)
	} else {
		// Fallback to phone number if no sessionId
		history = messageConversation.GetRepo().GetConversationHistory(userPhone, 10)
	}

	var description string
	if len(history) > 0 {
		description = "Conversation history:\n"
		for i, conv := range history {
			if i >= 5 {
				break
			}
			description += fmt.Sprintf("- %s: %s\n", conv.Role, conv.Message)
		}
	}

	// Determine severity and priority based on category
	severity := dtos.GetSeverityFromClassification(category)
	priority := dtos.GetPriorityFromClassification(category, 0.5)

	// Create case
	newCase := &customerServiceCases.CustomerServiceCase{
		ID:                    ulid.Make().String(),
		ClientId:              clientId,
		UserPhone:             userPhone,
		UserName:              userName,
		CaseNumber:            caseNumber,
		Category:              category,
		Subject:               subject,
		Description:           description,
		Status:                "new",
		Severity:              severity,
		Priority:              priority,
		ConversationSessionId: sessionId,
		SlaDeadline:           time.Now().Add(2 * time.Hour), // 2 hour SLA
		Tags:                  "[]",                          // Empty JSON array
		Metadata:              "{}",                          // Empty JSON object
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if err := customerServiceCases.GetRepo().Create(newCase); err != nil {
		logger.Errorf("[CustomerServiceSvc CreateCase] Failed to create case: %v", err)
		return nil, gocom.NewError(500, "Failed to create case")
	}

	logger.Infof("[CustomerServiceSvc CreateCase] Case created successfully id=%s caseNumber=%s", newCase.ID, newCase.CaseNumber)

	// Return case response
	caseResp := buildCaseResponse(newCase)
	return &caseResp, nil
}

func (s *CustomerServiceSvcImpl) ClaimCase(
	caseId, csUserId, severity string, priority int, notes string,
) (*dtos.CaseResponse, *gocom.CodedError) {
	logger.Infof("[CustomerServiceSvc ClaimCase] caseId=%s csUserId=%s", caseId, csUserId)

	caseData := customerServiceCases.GetRepo().GetById(caseId)
	if caseData == nil {
		logger.Warnf("[CustomerServiceSvc ClaimCase] Case not found caseId=%s", caseId)
		return nil, gocom.NewError(404, "Case not found")
	}

	if caseData.Status != "new" && caseData.Status != "pending_escalation" {
		logger.Warnf("[CustomerServiceSvc ClaimCase] Case already assigned caseId=%s status=%s", caseId, caseData.Status)
		return nil, gocom.NewError(400, fmt.Sprintf("Case already assigned or closed: status=%s", caseData.Status))
	}

	now := time.Now()
	caseData.Status = "assigned"
	caseData.AssignedTo = &csUserId
	caseData.AssignedAt = &now
	caseData.Severity = severity
	caseData.Priority = priority
	caseData.UpdatedAt = now

	if err := customerServiceCases.GetRepo().Update(caseData); err != nil {
		logger.Errorf("[CustomerServiceSvc ClaimCase] Failed to update case: %v", err)
		return nil, gocom.NewError(500, "Failed to update case")
	}

	// Create assignment record (audit trail)
	assignment := &customerServiceCases.CustomerServiceAssignment{
		ID:            ulid.Make().String(),
		CaseId:        caseId,
		Action:        "claimed",
		ToUserId:      &csUserId,
		SeverityAfter: severity,
		Notes:         notes,
		PerformedBy:   csUserId,
		PerformedAt:   now,
	}

	if err := customerServiceCases.GetRepo().CreateAssignment(assignment); err != nil {
		logger.Errorf("[CustomerServiceSvc ClaimCase] Failed to create assignment: %v", err)
	}

	logger.Infof("[CustomerServiceSvc ClaimCase] Case claimed successfully caseId=%s csUserId=%s", caseId, csUserId)

	// Return updated case data
	caseResp := buildCaseResponse(caseData)
	return &caseResp, nil
}

func (s *CustomerServiceSvcImpl) SendMessage(caseId, csUserId, message string) *gocom.CodedError {
	logger.Infof("[CustomerServiceSvc SendMessage] caseId=%s csUserId=%s", caseId, csUserId)

	caseData := customerServiceCases.GetRepo().GetById(caseId)
	if caseData == nil {
		logger.Warnf("[CustomerServiceSvc SendMessage] Case not found caseId=%s", caseId)
		return gocom.NewError(404, "Case not found")
	}

	if caseData.AssignedTo == nil || *caseData.AssignedTo != csUserId {
		logger.Warnf("[CustomerServiceSvc SendMessage] Case not assigned to this CS caseId=%s csUserId=%s", caseId, csUserId)
		return gocom.NewError(403, "Case not assigned to this CS agent")
	}

	// Send via WhatsApp
	waSvc := mainServices.GetWASendSvc()
	messageId, messageLogID, err := waSvc.SendTextMessageWithLog(caseData.UserPhone, message, caseData.ClientId, "")
	if err != nil {
		logger.Errorf("[CustomerServiceSvc SendMessage] Failed to send WhatsApp: %s", err.Message)
		return err
	}

	// Save to customer_service_messages
	now := time.Now()
	csMessage := &customerServiceCases.CustomerServiceMessage{
		ID:                ulid.Make().String(),
		CaseId:            caseId,
		SenderType:        "cs",
		SenderId:          &csUserId,
		MessageType:       "text",
		MessageContent:    message,
		WhatsappMessageId: &messageId,
		MessageLogId:      &messageLogID,
		Status:            "sent",
		SentAt:            now,
	}

	if err := customerServiceCases.GetRepo().CreateMessage(csMessage); err != nil {
		logger.Errorf("[CustomerServiceSvc SendMessage] Failed to save message: %v", err)
	}

	// Update case metrics
	caseData.CsMessages++
	caseData.TotalMessages++
	caseData.LastMessageAt = &now
	lastFrom := "cs"
	caseData.LastMessageFrom = &lastFrom

	if caseData.FirstResponseAt == nil {
		caseData.FirstResponseAt = &now
		firstResponseTime := int(now.Sub(caseData.CreatedAt).Seconds())
		caseData.FirstResponseTime = &firstResponseTime
	}

	customerServiceCases.GetRepo().Update(caseData)

	logger.Infof("[CustomerServiceSvc SendMessage] Message sent successfully caseId=%s", caseId)
	return nil
}

func (s *CustomerServiceSvcImpl) SaveUserMessage(caseId, userPhone, message, messageType string) *gocom.CodedError {
	logger.Infof("[CustomerServiceSvc SaveUserMessage] caseId=%s userPhone=%s", caseId, userPhone)

	caseData := customerServiceCases.GetRepo().GetById(caseId)
	if caseData == nil {
		logger.Warnf("[CustomerServiceSvc SaveUserMessage] Case not found caseId=%s", caseId)
		return gocom.NewError(404, "Case not found")
	}

	if caseData.UserPhone != userPhone {
		logger.Warnf("[CustomerServiceSvc SaveUserMessage] User phone mismatch caseId=%s", caseId)
		return gocom.NewError(400, "User phone mismatch")
	}

	now := time.Now()
	userMessage := &customerServiceCases.CustomerServiceMessage{
		ID:             ulid.Make().String(),
		CaseId:         caseId,
		SenderType:     "user",
		MessageType:    messageType,
		MessageContent: message,
		Status:         "received",
		SentAt:         now,
	}

	if err := customerServiceCases.GetRepo().CreateMessage(userMessage); err != nil {
		logger.Errorf("[CustomerServiceSvc SaveUserMessage] Failed to save: %v", err)
		return gocom.NewError(500, "Failed to save user message")
	}

	// Update case metrics
	caseData.UserMessages++
	caseData.TotalMessages++
	caseData.LastMessageAt = &now
	lastFrom := "user"
	caseData.LastMessageFrom = &lastFrom
	customerServiceCases.GetRepo().Update(caseData)

	logger.Infof("[CustomerServiceSvc SaveUserMessage] User message saved caseId=%s", caseId)
	return nil
}

func (s *CustomerServiceSvcImpl) AddNote(caseId, csUserId, noteType, content string, isInternal bool) *gocom.CodedError {
	logger.Infof("[CustomerServiceSvc AddNote] caseId=%s csUserId=%s", caseId, csUserId)

	caseData := customerServiceCases.GetRepo().GetById(caseId)
	if caseData == nil {
		logger.Warnf("[CustomerServiceSvc AddNote] Case not found caseId=%s", caseId)
		return gocom.NewError(404, "Case not found")
	}

	note := &customerServiceCases.CustomerServiceNote{
		ID:          ulid.Make().String(),
		CaseId:      caseId,
		NoteType:    noteType,
		NoteContent: content,
		CreatedBy:   csUserId,
		IsInternal:  isInternal,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := customerServiceCases.GetRepo().CreateNote(note); err != nil {
		logger.Errorf("[CustomerServiceSvc AddNote] Failed to create note: %v", err)
		return gocom.NewError(500, "Failed to create note")
	}

	logger.Infof("[CustomerServiceSvc AddNote] Note added caseId=%s", caseId)
	return nil
}

func (s *CustomerServiceSvcImpl) ResolveCase(caseId, csUserId, resolutionNotes string) *gocom.CodedError {
	logger.Infof("[CustomerServiceSvc ResolveCase] caseId=%s csUserId=%s", caseId, csUserId)

	caseData := customerServiceCases.GetRepo().GetById(caseId)
	if caseData == nil {
		logger.Warnf("[CustomerServiceSvc ResolveCase] Case not found caseId=%s", caseId)
		return gocom.NewError(404, "Case not found")
	}

	if caseData.AssignedTo == nil || *caseData.AssignedTo != csUserId {
		logger.Warnf("[CustomerServiceSvc ResolveCase] Case not assigned to this CS caseId=%s", caseId)
		return gocom.NewError(403, "Case not assigned to this CS agent")
	}

	now := time.Now()
	caseData.Status = "resolved"
	caseData.ResolvedAt = &now
	caseData.ResolutionNotes = resolutionNotes
	caseData.UpdatedAt = now

	if caseData.AssignedAt != nil {
		duration := int(now.Sub(*caseData.AssignedAt).Seconds())
		caseData.SessionDuration = &duration
	}

	if err := customerServiceCases.GetRepo().Update(caseData); err != nil {
		logger.Errorf("[CustomerServiceSvc ResolveCase] Failed to update: %v", err)
		return gocom.NewError(500, "Failed to update case")
	}

	logger.Infof("[CustomerServiceSvc ResolveCase] Case resolved caseId=%s", caseId)
	return nil
}

func (s *CustomerServiceSvcImpl) CloseCase(caseId, csUserId, closeNotes string) *gocom.CodedError {
	logger.Infof("[CustomerServiceSvc CloseCase] caseId=%s csUserId=%s", caseId, csUserId)

	caseData := customerServiceCases.GetRepo().GetById(caseId)
	if caseData == nil {
		logger.Warnf("[CustomerServiceSvc CloseCase] Case not found caseId=%s", caseId)
		return gocom.NewError(404, "Case not found")
	}

	now := time.Now()
	caseData.Status = "closed"
	caseData.ClosedAt = &now
	caseData.UpdatedAt = now

	if caseData.SessionDuration == nil && caseData.AssignedAt != nil {
		duration := int(now.Sub(*caseData.AssignedAt).Seconds())
		caseData.SessionDuration = &duration
	}

	if err := customerServiceCases.GetRepo().Update(caseData); err != nil {
		logger.Errorf("[CustomerServiceSvc CloseCase] Failed to update: %v", err)
		return gocom.NewError(500, "Failed to update case")
	}

	// CRITICAL: Close escalated conversation so user can start new AI conversation
	if err := messageConversation.GetRepo().CloseConversation(caseData.UserPhone); err != nil {
		logger.Errorf("[CustomerServiceSvc CloseCase] Failed to close conversation: %v", err)
	} else {
		logger.Infof("[CustomerServiceSvc CloseCase] Escalated conversation closed for user: %s", caseData.UserPhone)
	}

	// BILLING: Charge for CS conversation (ONCE per case)
	billingSvc := mainServices.GetBillingService()
	cost := billingSvc.CalculateMessageCostWithCategory("whatsapp", "cs_conversation", caseData.ClientId)

	billingTx, err := billingSvc.DeductBalance(
		caseData.ClientId,
		cost,
		caseId,
		"cs_conversation",
		fmt.Sprintf("Customer Service case %s with %s", caseData.CaseNumber, caseData.UserPhone),
		csUserId,
	)

	if err != nil {
		logger.Errorf("[CustomerServiceSvc CloseCase] Billing failed caseId=%s err=%s", caseId, err.Message)
	} else {
		caseData.BillingCost = cost
		caseData.BillingReference = &billingTx.ID
		customerServiceCases.GetRepo().Update(caseData)
		logger.Infof("[CustomerServiceSvc CloseCase] Billed caseId=%s cost=%.2f ref=%s", caseId, cost, billingTx.ID)
	}

	// Send closing message to user
	closingMessage := fmt.Sprintf(
		"Terima kasih telah menghubungi customer service PDAM.\n\n"+
			"✅ Tiket #%s telah ditutup.\n\n"+
			"Untuk memulai percakapan baru, silakan kirim \"Hai\". 🙏",
		caseData.CaseNumber,
	)

	_, _, sendErr := mainServices.GetWASendSvc().SendTextMessageWithLog(
		caseData.UserPhone,
		closingMessage,
		caseData.ClientId,
		"", // businessPhone not available yet - will use fallback config
	)

	if sendErr != nil {
		logger.Errorf("[CustomerServiceSvc CloseCase] Failed to send closing message: %s", sendErr.Message)
		// Don't fail the close operation if message sending fails
	} else {
		logger.Infof("[CustomerServiceSvc CloseCase] Closing message sent to user %s", caseData.UserPhone)
	}

	logger.Infof("[CustomerServiceSvc CloseCase] Case closed caseId=%s", caseId)
	return nil
}

func (s *CustomerServiceSvcImpl) EscalateCase(
	caseId, csUserId, escalateTo, reason string, severity string, priority int,
) *gocom.CodedError {
	logger.Infof("[CustomerServiceSvc EscalateCase] caseId=%s to=%s", caseId, escalateTo)

	caseData := customerServiceCases.GetRepo().GetById(caseId)
	if caseData == nil {
		logger.Warnf("[CustomerServiceSvc EscalateCase] Case not found caseId=%s", caseId)
		return gocom.NewError(404, "Case not found")
	}

	if caseData.AssignedTo == nil || *caseData.AssignedTo != csUserId {
		logger.Warnf("[CustomerServiceSvc EscalateCase] Case not assigned to this CS caseId=%s", caseId)
		return gocom.NewError(403, "Case not assigned to this CS agent")
	}

	oldSeverity := caseData.Severity
	now := time.Now()
	caseData.Status = "escalated"
	caseData.EscalatedTo = &escalateTo
	caseData.EscalatedAt = &now
	caseData.EscalationReason = reason
	caseData.Severity = severity
	caseData.Priority = priority
	caseData.UpdatedAt = now

	if err := customerServiceCases.GetRepo().Update(caseData); err != nil {
		logger.Errorf("[CustomerServiceSvc EscalateCase] Failed to update: %v", err)
		return gocom.NewError(500, "Failed to update case")
	}

	// Create assignment record
	assignment := &customerServiceCases.CustomerServiceAssignment{
		ID:             ulid.Make().String(),
		CaseId:         caseId,
		Action:         "escalated",
		FromUserId:     &csUserId,
		ToUserId:       &escalateTo,
		SeverityBefore: oldSeverity,
		SeverityAfter:  severity,
		Notes:          reason,
		PerformedBy:    csUserId,
		PerformedAt:    now,
	}

	customerServiceCases.GetRepo().CreateAssignment(assignment)

	logger.Infof("[CustomerServiceSvc EscalateCase] Case escalated caseId=%s to=%s", caseId, escalateTo)
	return nil
}

// ============================================================================
// Query Operations
// ============================================================================

func (s *CustomerServiceSvcImpl) GetCaseDetails(caseId string) (*dtos.CaseDetailsResponse, *gocom.CodedError) {
	logger.Infof("[CustomerServiceSvc GetCaseDetails] caseId=%s", caseId)

	caseData := customerServiceCases.GetRepo().GetById(caseId)
	if caseData == nil {
		logger.Warnf("[CustomerServiceSvc GetCaseDetails] Case not found caseId=%s", caseId)
		return nil, gocom.NewError(404, "Case not found")
	}

	// Get CS messages (after case created)
	csMessages := customerServiceCases.GetRepo().GetMessages(caseId)

	// Get conversation history (before case created) using session ID
	var conversationMessages []messageConversation.MessageConversation
	if caseData.ConversationSessionId != "" {
		conversationMessages = messageConversation.GetRepo().GetConversationHistoryBySession(caseData.ConversationSessionId, 50)
	}

	// Merge both message sources
	messageResponses := make([]dtos.MessageResponse, 0)

	// Add conversation messages (before escalation)
	for _, msg := range conversationMessages {
		// Only include user and assistant messages (not system)
		if msg.Role == "user" || msg.Role == "assistant" {
			senderType := "user"
			if msg.Role == "assistant" {
				senderType = "system" // AI assistant messages
			}

			messageResponses = append(messageResponses, dtos.MessageResponse{
				ID:             msg.ID,
				CaseId:         caseId,
				SenderType:     senderType,
				SenderName:     caseData.UserName,
				MessageType:    "text",
				MessageContent: msg.Message,
				Status:         msg.Status,
				SentAt:         msg.CreatedAt,
			})
		}
	}

	// Add CS messages (after escalation)
	for _, msg := range csMessages {
		messageResponses = append(messageResponses, dtos.MessageResponse{
			ID:                msg.ID,
			CaseId:            msg.CaseId,
			SenderType:        msg.SenderType,
			SenderId:          getStringValue(msg.SenderId),
			SenderName:        msg.SenderName,
			MessageType:       msg.MessageType,
			MessageContent:    msg.MessageContent,
			WhatsappMessageId: getStringValue(msg.WhatsappMessageId),
			Status:            msg.Status,
			ErrorMessage:      msg.ErrorMessage,
			SentAt:            msg.SentAt,
			DeliveredAt:       msg.DeliveredAt,
			ReadAt:            msg.ReadAt,
		})
	}

	notes := customerServiceCases.GetRepo().GetNotes(caseId)
	noteResponses := make([]dtos.NoteResponse, 0, len(notes))
	for _, note := range notes {
		noteResponses = append(noteResponses, dtos.NoteResponse{
			ID:            note.ID,
			CaseId:        note.CaseId,
			NoteType:      note.NoteType,
			NoteContent:   note.NoteContent,
			CreatedBy:     note.CreatedBy,
			CreatedByName: note.CreatedByName,
			IsInternal:    note.IsInternal,
			CreatedAt:     note.CreatedAt,
			UpdatedAt:     note.UpdatedAt,
		})
	}

	assignments := customerServiceCases.GetRepo().GetAssignments(caseId)
	assignmentResponses := make([]dtos.AssignmentResponse, 0, len(assignments))
	for _, asg := range assignments {
		assignmentResponses = append(assignmentResponses, dtos.AssignmentResponse{
			ID:             asg.ID,
			CaseId:         asg.CaseId,
			Action:         asg.Action,
			FromUserId:     getStringValue(asg.FromUserId),
			ToUserId:       getStringValue(asg.ToUserId),
			SeverityBefore: asg.SeverityBefore,
			SeverityAfter:  asg.SeverityAfter,
			Notes:          asg.Notes,
			PerformedBy:    asg.PerformedBy,
			PerformedAt:    asg.PerformedAt,
		})
	}

	caseResp := buildCaseResponse(caseData)

	return &dtos.CaseDetailsResponse{
		Case:                &caseResp,
		ConversationHistory: messageResponses,
		Notes:               noteResponses,
		Assignments:         assignmentResponses,
	}, nil
}

func (s *CustomerServiceSvcImpl) GetCasesList(filters *dtos.CaseFilters) (*dtos.CasesListResponse, *gocom.CodedError) {
	logger.Infof("[CustomerServiceSvc GetCasesList] Getting cases with filters")

	cases, total := customerServiceCases.GetRepo().GetByFilters(filters)

	caseResponses := make([]dtos.CaseResponse, 0, len(cases))
	for _, c := range cases {
		caseResponses = append(caseResponses, buildCaseResponse(&c))
	}

	stats := customerServiceCases.GetRepo().GetStats(filters.ClientId)

	page := filters.Page
	if page <= 0 {
		page = 1
	}
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	totalPages := (total + limit - 1) / limit

	return &dtos.CasesListResponse{
		Cases: caseResponses,
		Pagination: dtos.PaginationInfo{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
		Stats: dtos.CaseStats{
			NewCount:         stats["new"],
			AssignedCount:    stats["assigned"],
			ResolvedCount:    stats["resolved"],
			ClosedCount:      stats["closed"],
			EscalatedCount:   stats["escalated"],
			SlaBreachedCount: stats["sla_breached"],
		},
	}, nil
}

func (s *CustomerServiceSvcImpl) GetStatistics(clientId string, period string) (*dtos.StatisticsResponse, *gocom.CodedError) {
	logger.Infof("[CustomerServiceSvc GetStatistics] clientId=%s period=%s", clientId, period)

	// TODO: Implement statistics aggregation
	return &dtos.StatisticsResponse{
		Overview:    dtos.StatsOverview{},
		Performance: dtos.StatsPerformance{},
		TopCSAgents: []dtos.TopCSAgent{},
		Breakdown: dtos.StatsBreakdown{
			BySeverity: make(map[string]int),
			ByCategory: make(map[string]int),
			ByStatus:   make(map[string]int),
		},
	}, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func buildCaseResponse(c *customerServiceCases.CustomerServiceCase) dtos.CaseResponse {
	waitTime := time.Since(c.CreatedAt)
	if c.ClosedAt != nil {
		waitTime = c.ClosedAt.Sub(c.CreatedAt)
	}

	return dtos.CaseResponse{
		ID:                c.ID,
		CaseNumber:        c.CaseNumber,
		ClientId:          c.ClientId,
		UserPhone:         c.UserPhone,
		UserName:          c.UserName,
		Category:          c.Category,
		Subject:           c.Subject,
		Description:       c.Description,
		Status:            c.Status,
		Severity:          c.Severity,
		Priority:          c.Priority,
		AssignedTo:        getStringValue(c.AssignedTo),
		AssignedAt:        c.AssignedAt,
		SlaDeadline:       c.SlaDeadline,
		SlaBreached:       c.SlaBreached,
		FirstResponseAt:   c.FirstResponseAt,
		FirstResponseTime: c.FirstResponseTime,
		ResolvedAt:        c.ResolvedAt,
		ClosedAt:          c.ClosedAt,
		TotalMessages:     c.TotalMessages,
		CsMessages:        c.CsMessages,
		UserMessages:      c.UserMessages,
		SessionDuration:   c.SessionDuration,
		LastMessageAt:     c.LastMessageAt,
		LastMessageFrom:   getStringValue(c.LastMessageFrom),
		WaitTimeSeconds:   int64(waitTime.Seconds()),
		WaitTimeFormatted: formatDuration(waitTime),
		BillingCost:       c.BillingCost,
		BillingReference:  getStringValue(c.BillingReference),
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
}

func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm ago", hours, minutes)
	}
	return fmt.Sprintf("%dm ago", minutes)
}

// ============================================================================
// Singleton
// ============================================================================

var customerServiceSvc CustomerServiceSvc
var customerServiceSvcOnce sync.Once

func GetCustomerServiceSvc() CustomerServiceSvc {
	if customerServiceSvc == nil {
		customerServiceSvcOnce.Do(func() {
			customerServiceSvc = &CustomerServiceSvcImpl{}
		})
	}
	return customerServiceSvc
}
