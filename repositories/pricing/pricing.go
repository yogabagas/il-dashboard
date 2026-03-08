package pricing

import (
	"strings"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

type Pricing struct {
	ID          string         `gorm:"primaryKey;size:36"`
	ClientId    *string        `gorm:"size:36;index"`            // NULL for default pricing, specific client_id for custom pricing
	ServiceType string         `gorm:"size:50;not null;index"`   // whatsapp, sms, email
	Category    string         `gorm:"size:50;default:'';index"` // For whatsapp: MARKETING, UTILITY, AUTHENTICATION
	Cost        float64        `gorm:"type:decimal(10,4);not null;default:0"`
	Currency    string         `gorm:"size:3;default:'IDR'"`
	IsActive    bool           `gorm:"default:true;index"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (c *Pricing) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = ulid.Make().String()
	}
	return nil
}

type PricingRepo struct {
	gocom.BaseRepo
}

func (r *PricingRepo) GetById(id string) *Pricing {
	ret := &Pricing{}
	err := r.Where("id = ?", id).First(ret).Error
	if err != nil {
		return nil
	}
	return ret
}

// GetByServiceAndCategory retrieves pricing for a specific service type and category
// Returns client-specific pricing if clientId is provided and exists, otherwise returns default pricing
func (r *PricingRepo) GetByServiceAndCategory(serviceType, category string, clientId *string) *Pricing {
	ret := &Pricing{}
	tx := r.Where("LOWER(service_type) = ? AND is_active = ?", strings.ToLower(serviceType), true)

	// Normalize category to lowercase for consistent comparison
	normalizedCategory := strings.ToLower(strings.TrimSpace(category))
	tx = tx.Where("LOWER(category) = ?", normalizedCategory)

	// Try client-specific pricing first if clientId is provided
	if clientId != nil && *clientId != "" {
		err := tx.Where("client_id = ?", *clientId).First(ret).Error
		if err == nil {
			return ret
		}
	}

	// Fall back to default pricing (client_id IS NULL)
	err := tx.Where("client_id IS NULL").First(ret).Error
	if err != nil {
		return nil
	}
	return ret
}

// GetAllByServiceType retrieves all active pricing for a specific service type
func (r *PricingRepo) GetAllByServiceType(serviceType string) []Pricing {
	ret := []Pricing{}
	r.Where("service_type = ? AND is_active = ? AND client_id IS NULL", strings.ToLower(serviceType), true).
		Order("category").
		Find(&ret)
	return ret
}

// GetAllActive retrieves all active pricing entries
func (r *PricingRepo) GetAllActive() []Pricing {
	ret := []Pricing{}
	r.Where("is_active = ? AND client_id IS NULL", true).
		Order("service_type, category").
		Find(&ret)
	return ret
}

// Search retrieves pricing with pagination and filters
func (r *PricingRepo) Search(filter, serviceType, category, clientId string, isActive *bool, pageNo, rowPerPage int) ([]Pricing, bool, int64) {
	ret := []Pricing{}
	tx := r.Model(Pricing{})
	count := int64(0)

	// Only show default pricing (client_id IS NULL) in search
	//tx = tx.Where("client_id IS NULL")

	// Filter by service type
	if serviceType != "" {
		tx = tx.Where("LOWER(service_type) = ?", strings.ToLower(serviceType))
	}

	// Filter by category
	if category != "" {
		tx = tx.Where("LOWER(category) = ?", strings.ToLower(category))
	}

	// Filter by active status
	if isActive != nil {
		tx = tx.Where("is_active = ?", *isActive)
	}

	if clientId != "" {
		tx = tx.Where("client_id = ?", clientId)
	}

	// General filter (searches in service_type or category)
	if filter != "" {
		filter = "%" + strings.ToLower(filter) + "%"
		tx = tx.Where("LOWER(service_type) LIKE ? OR LOWER(category) LIKE ?", filter, filter)
	}

	// Count total
	tx.Count(&count)

	// Apply pagination
	if rowPerPage > 0 {
		tx = tx.Offset((pageNo - 1) * rowPerPage).Limit(rowPerPage)
	}

	// Order and fetch
	tx.Order("service_type, category").Find(&ret)

	// Calculate haveNext
	haveNext := false
	if rowPerPage > 0 {
		totalPage := int((count + int64(rowPerPage) - 1) / int64(rowPerPage))
		if pageNo < totalPage {
			haveNext = true
		}
	}

	return ret, haveNext, count
}

func (r *PricingRepo) Update(mdl *Pricing) error {
	return r.BaseRepo.Update(mdl).Error
}

func (r *PricingRepo) Create(mdl *Pricing) error {
	return r.BaseRepo.Create(mdl).Error
}

func (r *PricingRepo) Delete(id string) error {
	return r.Where("id = ?", id).Delete(&Pricing{}).Error
}

// Singleton instantiation
var pricingRepo *PricingRepo
var pricingRepoOnce sync.Once

func GetRepo() *PricingRepo {
	if pricingRepo == nil {
		pricingRepoOnce.Do(func() {
			pricingRepo = &PricingRepo{}
			_ = pricingRepo.AutoMigrate(Pricing{})
		})
	}
	return pricingRepo
}
