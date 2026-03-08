package messageLog

import (
	"math"
	"strings"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

type MessageLog struct {
	ID             string     `gorm:"primaryKey;size:36"`
	ClientId       string     `gorm:"size:36;not null;index"`
	SenderId       string     `gorm:"size:36"`
	CampaignId     string     `gorm:"size:36;index"`
	Type           string     `gorm:"size:20;not null;index"`          // whatsapp, sms, email
	Direction      string     `gorm:"size:20;not null;index"`          // inbound, outbound
	RecipientType  string     `gorm:"size:20;not null"`                // phone, email
	RecipientValue string     `gorm:"size:255;not null;index"`         // phone number or email
	TemplateId     string     `gorm:"size:36"`                         // template ID (for outbound template messages)
	MessageId      string     `gorm:"size:255;index"`                  // WhatsApp message ID
	SessionId      string     `gorm:"size:36;index"`                   // Conversation session ID
	MessageContent string     `gorm:"type:text"`                       // Actual message content
	Status         string     `gorm:"size:20;default:'pending';index"` // sent, delivered, read, failed, received
	Cost           float64    `gorm:"type:decimal(10,4);default:0"`
	SentAt         *time.Time `gorm:"index"`
	DeliveredAt    *time.Time
	ReadAt         *time.Time
	FailedAt       *time.Time
	ErrorMessage   string         `gorm:"type:text"`
	Metadata       string         `gorm:"type:json"`
	CreatedAt      time.Time      `gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (c *MessageLog) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = ulid.Make().String()
	}
	return nil
}

type MessageLogRepo struct {
	gocom.BaseRepo
}

func (r *MessageLogRepo) GetById(id string) *MessageLog {
	ret := &MessageLog{}
	err := r.Where("id = ?", id).First(ret).Error
	if err != nil {
		return nil
	}
	return ret
}

func (r *MessageLogRepo) GetByMessageId(messageId string) *MessageLog {
	ret := &MessageLog{}
	err := r.Where("message_id = ?", messageId).First(ret).Error
	if err != nil {
		return nil
	}
	return ret
}

func (r *MessageLogRepo) GetByClientIdAndDateRange(clientId string, startDate, endDate *time.Time) []MessageLog {
	ret := []MessageLog{}
	tx := r.Where("client_id = ?", clientId)

	if startDate != nil {
		tx = tx.Where("created_at >= ?", startDate)
	}

	if endDate != nil {
		endOfDay := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, endDate.Location())
		tx = tx.Where("created_at <= ?", endOfDay)
	}

	tx.Order("created_at desc").Find(&ret)
	return ret
}

func (r *MessageLogRepo) Update(mdl *MessageLog) error {
	return r.BaseRepo.Update(mdl).Error
}

func (r *MessageLogRepo) Search(
	filter,
	clientId,
	campaignId,
	messageType,
	status,
	direction,
	recipientVal string,
	dateFrom,
	dateTo *time.Time,
	pageNo,
	rowPerPage int,
) ([]MessageLog, bool, int64) {
	ret := []MessageLog{}
	tx := r.Model(MessageLog{})
	count := int64(0)

	if filter != "" {
		filter = "%" + strings.ToUpper(filter) + "%"
		tx = tx.Where("upper(recipient_value) like ? or upper(message_id) like ? or upper(message_content) like ?", filter, filter, filter)
	}

	if clientId != "" {
		tx = tx.Where("client_id = ?", clientId)
	}

	if campaignId != "" {
		tx = tx.Where("campaign_id = ?", campaignId)
	}

	if messageType != "" {
		tx = tx.Where("type = ?", messageType)
	}

	if status != "" {
		tx = tx.Where("status = ?", status)
	}

	if direction != "" {
		tx = tx.Where("direction = ?", direction)
	}

	if recipientVal != "" {
		tx = tx.Where("recipient_value = ?", recipientVal)
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

func (r *MessageLogRepo) Delete(id string) error {
	return r.Where("id = ?", id).Delete(&MessageLog{}).Error
}

// Singleton instantiation
var messageLogRepo *MessageLogRepo
var messageLogRepoOnce sync.Once

func GetRepo() *MessageLogRepo {
	if messageLogRepo == nil {
		messageLogRepoOnce.Do(func() {
			messageLogRepo = &MessageLogRepo{}
			_ = messageLogRepo.AutoMigrate(MessageLog{})
		})
	}
	return messageLogRepo
}
