package models


type OverviewStats struct{
	TotalReports int64 `json:"total_reports"`
	UnderReviewReports int64 `json:"under_review_reports"`
	ResolvedReports int64 `json:"resolved_reports"`
	DismissedReports int64 `json:"dismissed_reports"`
	TotalInvestigators int64 `json:"total_investigators"`
}

type TrendData struct{
	Month string `json:"month"`
	Count int `json:"count"`
}

type CategoryStats struct{
	Category string `json:"category"`
	Count int `json:"count"`
}

type StatusStats struct{
	Status string `json:"status"`
	Count int `json:"count"`
}

type InvestigatorPerformance struct{
	InvestigatorID uint `json:"investigator_id"`
	InvestigatorName string `json:"investigator_name"`
	HandledReports int `json:"handled_reports"`
	ResolvedReports int `json:"resolved_reports"`
	AvgResponseTime string `json:"avg_response_time"`
}