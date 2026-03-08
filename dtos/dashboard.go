package dtos

// DashboardStatsResponse represents dashboard statistics response
type DashboardStatsResponse struct {
	TotalSent      int64   `json:"totalSent"`
	TotalDelivered int64   `json:"totalDelivered"`
	TotalFailed    int64   `json:"totalFailed"`
	SuccessRate    float64 `json:"successRate"` // Percentage (0-100)
}

// DashboardOverviewResponse represents dashboard overview with time periods
type DashboardOverviewResponse struct {
	Today      *DashboardStatsResponse `json:"today"`
	Last7Days  *DashboardStatsResponse `json:"last7Days"`
	Last30Days *DashboardStatsResponse `json:"last30Days"`
}
