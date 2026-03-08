package aiKnowledgeBase

import (
	"math"
	"strings"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

type AiKnowledgeBase struct {
	ID        string    `gorm:"primaryKey;size:36"`
	ClientId  string    `gorm:"size:36;not null;index"`
	Category  string    `gorm:"size:100;not null;index"` // layanan, tagihan, pengaduan, administrasi, umum
	Question  string    `gorm:"type:text;not null"`      // Pertanyaan/topik
	Answer    string    `gorm:"type:text;not null"`      // Jawaban/penjelasan
	Keywords  string    `gorm:"type:text"`               // Keywords untuk pencarian (comma separated)
	IsActive  bool      `gorm:"default:true;index"`
	Priority  int       `gorm:"default:0"` // Prioritas tampil (semakin tinggi semakin penting)
	CreatedBy string    `gorm:"size:100"`
	UpdatedBy string    `gorm:"size:100"`
	DeletedBy string    `gorm:"size:100"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	DeletedAt *time.Time
}

func (a *AiKnowledgeBase) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = ulid.Make().String()
	}
	return nil
}

type AiKnowledgeBaseRepo struct {
	gocom.BaseRepo
}

func (r *AiKnowledgeBaseRepo) GetById(id string) *AiKnowledgeBase {
	ret := &AiKnowledgeBase{}
	err := r.Where("id = ?", id).Where("deleted_at IS NULL").First(ret).Error
	if err != nil {
		return nil
	}
	return ret
}

func (r *AiKnowledgeBaseRepo) GetByQuestion(question string) *AiKnowledgeBase {
	ret := &AiKnowledgeBase{}
	question = strings.ToLower(question)
	err := r.Where("lower(question) = ?", question).Where("deleted_at IS NULL").First(ret).Error
	if err != nil {
		return nil
	}
	return ret
}

func (r *AiKnowledgeBaseRepo) GetByClientId(clientId string) []AiKnowledgeBase {
	ret := []AiKnowledgeBase{}
	r.Where("client_id = ?", clientId).
		Where("is_active = ?", true).
		Where("deleted_at IS NULL").
		Order("priority desc, category asc, created_at asc").
		Find(&ret)
	return ret
}

func (r *AiKnowledgeBaseRepo) GetByClientIdAndCategory(clientId, category string) []AiKnowledgeBase {
	ret := []AiKnowledgeBase{}
	r.Where("client_id = ?", clientId).
		Where("category = ?", category).
		Where("is_active = ?", true).
		Where("deleted_at IS NULL").
		Order("priority desc, created_at asc").
		Find(&ret)
	return ret
}

func (r *AiKnowledgeBaseRepo) GetActiveKnowledge(clientId string) []AiKnowledgeBase {
	ret := []AiKnowledgeBase{}
	r.Where("client_id = ?", clientId).
		Where("is_active = ?", true).
		Where("deleted_at IS NULL").
		Order("priority desc, category asc").
		Find(&ret)
	return ret
}

func (r *AiKnowledgeBaseRepo) Update(mdl *AiKnowledgeBase) error {
	return r.BaseRepo.Update(mdl).Error
}

func (r *AiKnowledgeBaseRepo) Search(filter, clientId, category string, isActive *bool, dateFrom, dateTo *time.Time, pageNo, rowPerPage int) ([]AiKnowledgeBase, bool, int64) {
	ret := []AiKnowledgeBase{}
	tx := r.Model(AiKnowledgeBase{}).Where("deleted_at IS NULL")
	count := int64(0)

	if filter != "" {
		filter = "%" + strings.ToUpper(filter) + "%"
		tx = tx.Where("upper(question) like ? or upper(answer) like ? or upper(keywords) like ?", filter, filter, filter)
	}

	if clientId != "" {
		tx = tx.Where("client_id = ?", clientId)
	}

	if category != "" {
		tx = tx.Where("category = ?", category)
	}

	if isActive != nil {
		tx = tx.Where("is_active = ?", *isActive)
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

	tx.Order("priority desc, category asc, created_at desc").Find(&ret)

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

func (r *AiKnowledgeBaseRepo) Delete(id, deletedBy string) error {
	now := time.Now()
	return r.Model(AiKnowledgeBase{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"deleted_by": deletedBy,
		}).Error
}

func (r *AiKnowledgeBaseRepo) HardDelete(id string) error {
	return r.Where("id = ?", id).Delete(&AiKnowledgeBase{}).Error
}

// Singleton instantiation
var aiKnowledgeBaseRepo *AiKnowledgeBaseRepo
var aiKnowledgeBaseRepoOnce sync.Once

func GetRepo() *AiKnowledgeBaseRepo {
	if aiKnowledgeBaseRepo == nil {
		aiKnowledgeBaseRepoOnce.Do(func() {
			aiKnowledgeBaseRepo = &AiKnowledgeBaseRepo{}
			_ = aiKnowledgeBaseRepo.AutoMigrate(AiKnowledgeBase{})
		})
	}
	return aiKnowledgeBaseRepo
}
