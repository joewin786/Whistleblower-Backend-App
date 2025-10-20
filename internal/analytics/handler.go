package analytics

import(
	"encoding/json"
	"net/http"

	"fmt"

	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"

	"gorm.io/gorm"
)

type AnalyticsHandler struct {
	DB *gorm.DB
}

func (h *AnalyticsHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	var stats models.OverviewStats

	h.DB.Model(&models.Report{}).Count(&stats.TotalReports)
	h.DB.Model(&models.Report{}).Where("status = ?", "pending").Count(&stats.PendingReports)
	h.DB.Model(&models.Report{}).Where("status = ?", "resolved").Count(&stats.ResolvedReports)
	h.DB.Model(&models.Report{}).Where("status = ?", "rejected").Count(&stats.RejectedReports)
	h.DB.Model(&models.User{}).Where("role = ?", "investigator").Count(&stats.TotalInvestigators)

	utils.RespondWithJSON(w, http.StatusOK, stats)
}

func (h *AnalyticsHandler) GetTrends(w http.ResponseWriter, r *http.Request) {
	type result struct {
		Month string
		Count int
	}
	var rows []result

	h.DB.Model(&models.Report{}).
		Select("DATE_FORMAT(created_at, '%Y-%m') as month, COUNT(*) as count").
		Group("month").
		Order("month ASC").
		Scan(&rows)

	var trends []models.TrendData
	for _, r := range rows {
		trends = append(trends, models.TrendData{
			Month: r.Month,
			Count: r.Count,
		})
	}

	utils.RespondWithJSON(w, http.StatusOK, trends)
}

// ✅ GET /analytics/by-categories
func (h *AnalyticsHandler) GetByCategories(w http.ResponseWriter, r *http.Request) {
	type result struct {
		Category string
		Count    int
	}
	var rows []result

	h.DB.Model(&models.Report{}).
		Select("category, COUNT(*) as count").
		Group("category").
		Order("count DESC").
		Scan(&rows)

	var stats []models.CategoryStats
	for _, r := range rows {
		stats = append(stats, models.CategoryStats{
			Category: r.Category,
			Count:    r.Count,
		})
	}

	utils.RespondWithJSON(w, http.StatusOK, stats)
}

// ✅ GET /analytics/by-status
func (h *AnalyticsHandler) GetByStatus(w http.ResponseWriter, r *http.Request) {
	type result struct {
		Status string
		Count  int
	}
	var rows []result

	h.DB.Model(&models.Report{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Order("count DESC").
		Scan(&rows)

	var stats []models.StatusStats
	for _, r := range rows {
		stats = append(stats, models.StatusStats{
			Status: r.Status,
			Count:  r.Count,
		})
	}

	utils.RespondWithJSON(w, http.StatusOK, stats)
}

// ✅ GET /analytics/investigator-performance
func (h *AnalyticsHandler) GetInvestigatorPerformance(w http.ResponseWriter, r *http.Request) {
	type result struct {
		InvestigatorID   uint
		InvestigatorName string
		HandledReports   int
		ResolvedReports  int
		AvgResponseHours *float64
	}
	var rows []result

	h.DB.Table("users AS u").
		Select(`
			u.id AS investigator_id,
			u.full_name AS investigator_name,
			COUNT(r.id) AS handled_reports,
			SUM(CASE WHEN r.status = 'resolved' THEN 1 ELSE 0 END) AS resolved_reports,
			AVG(TIMESTAMPDIFF(HOUR, r.assigned_at, r.resolved_at)) AS avg_response_hours
		`).
		Joins("LEFT JOIN reports r ON r.investigator_id = u.id").
		Where("u.role = ?", "investigator").
		Group("u.id, u.full_name").
		Order("handled_reports DESC").
		Scan(&rows)

	var data []models.InvestigatorPerformance
	for _, r := range rows {
		avgTime := "-"
		if r.AvgResponseHours != nil {
			avgTime = formatResponseTime(*r.AvgResponseHours)
		}
		data = append(data, models.InvestigatorPerformance{
			InvestigatorID:   r.InvestigatorID,
			InvestigatorName: r.InvestigatorName,
			HandledReports:   r.HandledReports,
			ResolvedReports:  r.ResolvedReports,
			AvgResponseTime:  avgTime,
		})
	}

	utils.RespondWithJSON(w, http.StatusOK, data)
}

// ✅ POST /analytics/reports/generate
func (h *AnalyticsHandler) GenerateReport(w http.ResponseWriter, r *http.Request) {
	type GenerateRequest struct {
		Format string `json:"format"` // "pdf" or "excel"
	}
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Format != "pdf" && req.Format != "excel" {
		utils.RespondWithError(w, http.StatusBadRequest, "Format must be 'pdf' or 'excel'")
		return
	}

	// TODO: implement real export (pakai gofpdf/excelize)
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Report generated successfully in " + req.Format + " format",
	})
}

// 🕓 Helper — konversi jam ke format yang lebih manusiawi

func formatResponseTime(hours float64) string {
	if hours < 24 {
		return fmt.Sprintf("%.1f jam", hours)
	}
	days := hours / 24
	return fmt.Sprintf("%.1f hari", days)
}