package services

import (
	"sync"
	"time"

	"github.com/ariandi/gocom/logger"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/messageLog"
	"gitlab.com/bot3342545/il-dashboard/dtos"
)

type DashboardService interface {
	GetOverview(authInfo auth.AuthInfo, filterClientId string) (*dtos.DashboardOverviewResponse, error)
}

type DashboardServiceImpl struct{}

var dashboardService DashboardService
var onceDashboardService sync.Once

func GetDashboardService() DashboardService {
	onceDashboardService.Do(func() {
		dashboardService = &DashboardServiceImpl{}
	})
	return dashboardService
}

// GetOverview returns dashboard overview with stats for today, 7 days, and 30 days
func (s *DashboardServiceImpl) GetOverview(authInfo auth.AuthInfo, filterClientId string) (*dtos.DashboardOverviewResponse, error) {
	// Determine which clientId to use for filtering
	var targetClientId string

	if authInfo.ClientId == common.PROVIDER_ID {
		// ADMIN: can filter by specific client or see all clients
		if filterClientId != "" {
			targetClientId = filterClientId
			logger.Infof("[DashboardService GetOverview] Admin viewing client: %s", targetClientId)
		} else {
			targetClientId = "" // Empty = all clients
			logger.Infof("[DashboardService GetOverview] Admin viewing ALL clients")
		}
	} else {
		// OWNER/STAFF: can only see their own client data
		targetClientId = authInfo.ClientId
		logger.Infof("[DashboardService GetOverview] Owner/Staff viewing own client: %s", targetClientId)
	}

	now := time.Now()

	// Calculate time ranges
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)

	last7DaysStart := todayStart.Add(-7 * 24 * time.Hour)
	last30DaysStart := todayStart.Add(-30 * 24 * time.Hour)

	// Get stats for each period
	todayStats := s.getStatsForPeriod(targetClientId, todayStart, todayEnd)
	last7DaysStats := s.getStatsForPeriod(targetClientId, last7DaysStart, todayEnd)
	last30DaysStats := s.getStatsForPeriod(targetClientId, last30DaysStart, todayEnd)

	response := &dtos.DashboardOverviewResponse{
		Today:      todayStats,
		Last7Days:  last7DaysStats,
		Last30Days: last30DaysStats,
	}

	logger.Infof("[DashboardService GetOverview] Overview completed")
	return response, nil
}

// getStatsForPeriod calculates statistics for a specific time period
func (s *DashboardServiceImpl) getStatsForPeriod(clientId string, startTime, endTime time.Time) *dtos.DashboardStatsResponse {
	repo := messageLog.GetRepo()

	// Build base query
	baseQuery := repo.Model(&messageLog.MessageLog{}).
		Where("direction = ?", "outbound"). // Only count outbound messages
		Where("created_at >= ? AND created_at < ?", startTime, endTime)

	// Filter by client ID if specified (empty = all clients)
	if clientId != "" {
		baseQuery = baseQuery.Where("client_id = ?", clientId)
	}

	// Count total sent (all outbound messages)
	var totalSent int64
	baseQuery.Count(&totalSent)

	// Count delivered (status = 'delivered' or 'read')
	var totalDelivered int64
	deliveredQuery := repo.Model(&messageLog.MessageLog{}).
		Where("direction = ?", "outbound").
		Where("created_at >= ? AND created_at < ?", startTime, endTime).
		Where("status IN ?", []string{"delivered", "read"})

	if clientId != "" {
		deliveredQuery = deliveredQuery.Where("client_id = ?", clientId)
	}
	deliveredQuery.Count(&totalDelivered)

	// Count failed (status = 'failed')
	var totalFailed int64
	failedQuery := repo.Model(&messageLog.MessageLog{}).
		Where("direction = ?", "outbound").
		Where("created_at >= ? AND created_at < ?", startTime, endTime).
		Where("status = ?", "failed")

	if clientId != "" {
		failedQuery = failedQuery.Where("client_id = ?", clientId)
	}
	failedQuery.Count(&totalFailed)

	// Calculate success rate
	var successRate float64
	if totalSent > 0 {
		successRate = (float64(totalDelivered) / float64(totalSent)) * 100
	}

	logger.Debugf("[DashboardService getStatsForPeriod] ClientID: %s, Period %v to %v - Sent: %d, Delivered: %d, Failed: %d, Rate: %.2f%%",
		clientId, startTime, endTime, totalSent, totalDelivered, totalFailed, successRate)

	return &dtos.DashboardStatsResponse{
		TotalSent:      totalSent,
		TotalDelivered: totalDelivered,
		TotalFailed:    totalFailed,
		SuccessRate:    successRate,
	}
}
