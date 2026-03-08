package messageConversation

import (
	"math"
	"strings"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

type MessageConversation struct {
	ID                 string     `gorm:"primaryKey;size:36"`
	ClientId           string     `gorm:"size:36;not null;index"`
	SessionID          string     `gorm:"size:100;not null;index"`
	FromNumber         string     `gorm:"size:255;not null;index"` // User WhatsApp number
	ToNumber           string     `gorm:"size:255;index"`          // Destination WhatsApp number (for assistant messages)
	Role               string     `gorm:"size:20;not null"`        // user, assistant, system
	Message            string     `gorm:"type:text;not null"`
	MessageLogId       string     `gorm:"size:36;index"` // Foreign key to message_logs table
	TokensUsed         int        `gorm:"default:0"`
	Model              string     `gorm:"size:100"`
	Status             string     `gorm:"size:20;default:'sent';index"`
	ConversationStatus string     `gorm:"size:20;default:'active';index"` // active, closed
	ErrorMessage       string     `gorm:"type:text"`
	CreatedAt          time.Time  `gorm:"autoCreateTime"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime"`
	DeletedAt          *time.Time `gorm:"index"`
}

func (c *MessageConversation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = ulid.Make().String()
	}
	return nil
}

type MessageConversationRepo struct {
	gocom.BaseRepo
}

func (r *MessageConversationRepo) GetById(id string) *MessageConversation {
	ret := &MessageConversation{}
	err := r.Where("id = ?", id).First(ret).Error
	if err != nil {
		return nil
	}
	return ret
}

func (r *MessageConversationRepo) GetBySessionId(sessionId string) []MessageConversation {
	ret := []MessageConversation{}
	r.Where("session_id = ?", sessionId).Order("created_at asc").Find(&ret)
	return ret
}

func (r *MessageConversationRepo) GetByFromNumber(fromNumber string, limit int) []MessageConversation {
	ret := []MessageConversation{}
	tx := r.Where("from_number = ?", fromNumber).Order("created_at desc")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	tx.Find(&ret)
	return ret
}

func (r *MessageConversationRepo) GetConversationHistory(fromNumber string, limit int) []MessageConversation {
	ret := []MessageConversation{}
	tx := r.Where("from_number = ?", fromNumber).Order("created_at desc")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	tx.Find(&ret)

	// Reverse untuk urutan ascending (dari lama ke baru)
	for i, j := 0, len(ret)-1; i < j; i, j = i+1, j-1 {
		ret[i], ret[j] = ret[j], ret[i]
	}

	return ret
}

func (r *MessageConversationRepo) GetConversationHistoryBySession(sessionId string, limit int) []MessageConversation {
	ret := []MessageConversation{}
	tx := r.Where("session_id = ?", sessionId).Order("created_at desc")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	tx.Find(&ret)

	// Reverse untuk urutan ascending (dari lama ke baru)
	for i, j := 0, len(ret)-1; i < j; i, j = i+1, j-1 {
		ret[i], ret[j] = ret[j], ret[i]
	}

	return ret
}

func (r *MessageConversationRepo) GetByMessageLogId(messageLogId string) *MessageConversation {
	ret := &MessageConversation{}
	err := r.Where("message_log_id = ?", messageLogId).First(ret).Error
	if err != nil {
		return nil
	}
	return ret
}

func (r *MessageConversationRepo) Update(mdl *MessageConversation) error {
	return r.BaseRepo.Update(mdl).Error
}

func (r *MessageConversationRepo) Search(filter, clientId, fromNumber, sessionId, role string, dateFrom, dateTo *time.Time, pageNo, rowPerPage int) ([]MessageConversation, bool, int64) {
	ret := []MessageConversation{}
	tx := r.Model(MessageConversation{})
	count := int64(0)

	if filter != "" {
		filter = "%" + strings.ToUpper(filter) + "%"
		tx = tx.Where("upper(message) like ? or upper(from_number) like ? or upper(to_number) like ?", filter, filter, filter)
	}

	if clientId != "" {
		tx = tx.Where("client_id = ?", clientId)
	}

	if fromNumber != "" {
		tx = tx.Where("from_number = ?", fromNumber)
	}

	if sessionId != "" {
		tx = tx.Where("session_id = ?", sessionId)
	}

	if role != "" {
		tx = tx.Where("role = ?", role)
	}

	if dateFrom != nil {
		tx = tx.Where("created_at >= ?", dateFrom.Format("2006-01-02 15:04:05"))
	}

	if dateTo != nil {
		endOfDay := time.Date(dateTo.Year(), dateTo.Month(), dateTo.Day(), 23, 59, 59, 0, dateTo.Location())
		tx = tx.Where("created_at <= ?", endOfDay.Format("2006-01-02 15:04:05"))
	}

	tx.Count(&count)

	if rowPerPage > 0 {
		tx = tx.Offset((pageNo - 1) * rowPerPage).Limit(rowPerPage)
	}

	tx.Order("created_at desc").Find(&ret)

	haveNext := false
	totalPage := 1
	if rowPerPage > 0 {
		totalPage = int(math.Ceil(float64(count) / float64(rowPerPage)))
		if pageNo < totalPage {
			haveNext = true
		}
	}

	return ret, haveNext, count
}

func (r *MessageConversationRepo) Delete(id string) error {
	return r.Where("id = ?", id).Delete(&MessageConversation{}).Error
}

func (r *MessageConversationRepo) DeleteBySessionId(sessionId string) error {
	return r.Where("session_id = ?", sessionId).Delete(&MessageConversation{}).Error
}

// IsConversationActive checks if conversation is still active (not closed) and within 60 minutes timeout
func (r *MessageConversationRepo) IsConversationActive(fromNumber string) bool {
	var lastConv MessageConversation
	err := r.Model(MessageConversation{}).
		Where("from_number = ?", fromNumber).
		Where("conversation_status = ?", "active").
		Where("role = ?", "assistant"). // Check last assistant message
		Order("created_at desc").
		Limit(1).
		First(&lastConv).Error

	// No active conversation found
	if err != nil {
		logger.Debugf("[IsConversationActive] No active conversation found for: %s", fromNumber)
		return false
	}

	// Check if last conversation is older than 60 minutes
	timeout := 60 * time.Minute
	timeSinceLastMessage := time.Since(lastConv.CreatedAt)

	if timeSinceLastMessage > timeout {
		// Auto-close conversation due to timeout
		logger.Infof("[IsConversationActive] Auto-closing conversation for %s (inactive for %.0f minutes)", fromNumber, timeSinceLastMessage.Minutes())
		_ = r.CloseConversation(fromNumber)
		return false
	}

	logger.Debugf("[IsConversationActive] Conversation active for %s (last message %.0f minutes ago)", fromNumber, timeSinceLastMessage.Minutes())
	return true
}

// CloseConversation closes all active and escalated conversations for a user
func (r *MessageConversationRepo) CloseConversation(fromNumber string) error {
	return r.Model(MessageConversation{}).
		Where("from_number = ?", fromNumber).
		Where("conversation_status IN ?", []string{"active", "escalated"}).
		Update("conversation_status", "closed").Error
}

// IsConversationEscalated checks if conversation has been escalated to CS
func (r *MessageConversationRepo) IsConversationEscalated(fromNumber string) bool {
	var lastConv MessageConversation
	err := r.Model(MessageConversation{}).
		Where("from_number = ?", fromNumber).
		Where("conversation_status = ?", "escalated").
		Order("created_at desc").
		Limit(1).
		First(&lastConv).Error

	// No escalated conversation found
	if err != nil {
		return false
	}

	// Check if escalation is recent (within 1 hour)
	timeout := 1 * time.Hour
	timeSinceEscalation := time.Since(lastConv.CreatedAt)

	if timeSinceEscalation > timeout {
		logger.Infof("[IsConversationEscalated] Escalation expired for %s (escalated %.0f minutes ago)", fromNumber, timeSinceEscalation.Minutes())
		return false
	}

	logger.Infof("[IsConversationEscalated] Conversation is escalated for %s (%.0f minutes ago)", fromNumber, timeSinceEscalation.Minutes())
	return true
}

// Singleton instantiation
var messageConversationRepo *MessageConversationRepo
var messageConversationRepoOnce sync.Once

func GetRepo() *MessageConversationRepo {
	if messageConversationRepo == nil {
		messageConversationRepoOnce.Do(func() {
			messageConversationRepo = &MessageConversationRepo{}
			_ = messageConversationRepo.AutoMigrate(MessageConversation{})
		})
	}
	return messageConversationRepo
}
