package ai

import (
	
	"fmt"
	"net/http"
	"strconv"
	"time"

	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/notifications"
	"whistleblower_REST/internal/utils"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type Handler struct {
	DB            *gorm.DB
	GeminiService *GeminiService
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		DB:            db,
		GeminiService: NewGeminiService(),
	}
}

// AnalyzeReport triggers AI analysis for a reviewed report
func (h *Handler) AnalyzeReport(w http.ResponseWriter, r *http.Request) {
	// Check admin role
	role, _ := r.Context().Value("role").(string)
	if role != "admin" {
		utils.RespondWithError(w, http.StatusForbidden, "forbidden: only admins can trigger AI analysis")
		return
	}

	// ⚠️ REMOVED: adminID variable (not needed anymore)
	// adminID, _ := r.Context().Value("id").(string)

	reportIDStr := chi.URLParam(r, "reportId")
	reportID, err := strconv.ParseUint(reportIDStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report ID")
		return
	}

	// Validate API key
	if err := h.GeminiService.ValidateAPIKey(); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Gemini AI not configured: "+err.Error())
		return
	}

	// Get report with review
	var report models.Report
	if err := h.DB.Preload("User").First(&report, reportID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "report not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check if review exists
	var review models.Review
	if err := h.DB.Preload("Admin").Where("report_id = ?", reportID).First(&review).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusBadRequest, "report must be reviewed before AI analysis")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check if AI analysis already exists
	var existingAnalysis models.AIAnalysis
	if err := h.DB.Where("report_id = ?", reportID).First(&existingAnalysis).Error; err == nil {
		utils.RespondWithError(w, http.StatusConflict, "AI analysis already exists for this report. Use update endpoint to re-analyze.")
		return
	}

	fmt.Printf("[INFO] 🤖 Starting AI analysis for Report #%d...\n", reportID)

	// Call Gemini AI
	aiResult, rawResponse, err := h.GeminiService.AnalyzeReport(&report, &review)
	if err != nil {
		fmt.Printf("[ERROR] ❌ AI analysis failed: %v\n", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "AI analysis failed: "+err.Error())
		return
	}

	// Save AI analysis to database
	aiAnalysis := models.AIAnalysis{
		ReportID:          uint(reportID),
		Verdict:           aiResult.Verdict,
		Confidence:        aiResult.Confidence,
		Reasoning:         aiResult.Reasoning,
		RedFlags:          models.StringArray(aiResult.RedFlags),
		SupportingFactors: models.StringArray(aiResult.SupportingFactors),
		Recommendation:    aiResult.Recommendation,
		RawResponse:       rawResponse,
		AIModel:           "gemini-pro",
	}

	if err := h.DB.Create(&aiAnalysis).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to save AI analysis")
		return
	}

	// ⚠️ REMOVED: AI no longer auto-updates report status
	// Status tetap di "under_review" atau status sebelumnya
	// Admin yang akan update status secara manual

	fmt.Printf("[INFO] ✅ AI Analysis completed for Report #%d\n", reportID)
	fmt.Printf("[INFO] 📊 Verdict: %s (Confidence: %.1f%%)\n", aiResult.Verdict, aiResult.Confidence)

	// Send notification about AI analysis completion
	go func() {
		if err := notifications.NotifyAIAnalysisComplete(h.DB, uint(reportID), aiResult.Verdict, aiResult.Confidence); err != nil {
			fmt.Printf("[WARN] ⚠️ Failed to send AI analysis notification: %v\n", err)
		}
	}()

	// Reload with relations
	h.DB.Preload("Report").First(&aiAnalysis, aiAnalysis.ID)

	utils.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"message":  "AI analysis completed successfully",
		"analysis": aiAnalysis,
		"note":     "AI verdict is advisory only. Admin should manually update report status.",
	})
}

// GetByReportID gets AI analysis for a specific report
func (h *Handler) GetByReportID(w http.ResponseWriter, r *http.Request) {
	reportIDStr := chi.URLParam(r, "reportId")
	reportID, err := strconv.ParseUint(reportIDStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report ID")
		return
	}

	var analysis models.AIAnalysis
	if err := h.DB.Preload("Report").Where("report_id = ?", reportID).First(&analysis).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "AI analysis not found for this report")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, analysis)
}

// ReAnalyze triggers re-analysis for a report (overwrites existing analysis)
func (h *Handler) ReAnalyze(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("role").(string)
	if role != "admin" {
		utils.RespondWithError(w, http.StatusForbidden, "forbidden: admin only")
		return
	}

	reportIDStr := chi.URLParam(r, "reportId")
	reportID, err := strconv.ParseUint(reportIDStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report ID")
		return
	}

	// Check if analysis exists
	var existingAnalysis models.AIAnalysis
	if err := h.DB.Where("report_id = ?", reportID).First(&existingAnalysis).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "no existing analysis found. Use analyze endpoint instead.")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get report and review
	var report models.Report
	if err := h.DB.First(&report, reportID).Error; err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "report not found")
		return
	}

	var review models.Review
	if err := h.DB.Where("report_id = ?", reportID).First(&review).Error; err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "review not found")
		return
	}

	fmt.Printf("[INFO] 🔄 Re-analyzing Report #%d...\n", reportID)

	// Call Gemini AI
	aiResult, rawResponse, err := h.GeminiService.AnalyzeReport(&report, &review)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "AI re-analysis failed: "+err.Error())
		return
	}

	// Update existing analysis
	existingAnalysis.Verdict = aiResult.Verdict
	existingAnalysis.Confidence = aiResult.Confidence
	existingAnalysis.Reasoning = aiResult.Reasoning
	existingAnalysis.RedFlags = models.StringArray(aiResult.RedFlags)
	existingAnalysis.SupportingFactors = models.StringArray(aiResult.SupportingFactors)
	existingAnalysis.Recommendation = aiResult.Recommendation
	existingAnalysis.RawResponse = rawResponse
	existingAnalysis.UpdatedAt = time.Now()

	if err := h.DB.Save(&existingAnalysis).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update AI analysis")
		return
	}

	fmt.Printf("[INFO] ✅ Re-analysis completed for Report #%d (Verdict: %s)\n", reportID, aiResult.Verdict)

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "AI re-analysis completed successfully",
		"analysis": existingAnalysis,
	})
}

// GetAllAnalyses gets all AI analyses with pagination (admin only)
func (h *Handler) GetAllAnalyses(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("role").(string)
	if role != "admin" {
		utils.RespondWithError(w, http.StatusForbidden, "forbidden: admin only")
		return
	}

	// Query parameters for filtering
	verdict := r.URL.Query().Get("verdict")

	query := h.DB.Preload("Report")

	if verdict != "" {
		query = query.Where("verdict = ?", verdict)
	}

	var analyses []models.AIAnalysis
	if err := query.Order("analyzed_at DESC").Find(&analyses).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Calculate statistics
	stats := map[string]interface{}{
		"total":       len(analyses),
		"verified":    0,
		"hoax":        0,
		"unconfirmed": 0,
		"avg_confidence": 0.0,
	}

	totalConfidence := 0.0
	for _, a := range analyses {
		switch a.Verdict {
		case models.VerdictVerified:
			stats["verified"] = stats["verified"].(int) + 1
		case models.VerdictHoax:
			stats["hoax"] = stats["hoax"].(int) + 1
		case models.VerdictUnconfirmed:
			stats["unconfirmed"] = stats["unconfirmed"].(int) + 1
		}
		totalConfidence += a.Confidence
	}

	if len(analyses) > 0 {
		stats["avg_confidence"] = totalConfidence / float64(len(analyses))
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"analyses": analyses,
		"stats":    stats,
	})
}

// GetStatistics returns AI analysis statistics
func (h *Handler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("role").(string)
	if role != "admin" {
		utils.RespondWithError(w, http.StatusForbidden, "forbidden: admin only")
		return
	}

	type Stats struct {
		TotalAnalyses    int64   `json:"total_analyses"`
		VerifiedCount    int64   `json:"verified_count"`
		HoaxCount        int64   `json:"hoax_count"`
		UnconfirmedCount int64   `json:"unconfirmed_count"`
		AvgConfidence    float64 `json:"avg_confidence"`
	}

	var stats Stats

	h.DB.Model(&models.AIAnalysis{}).Count(&stats.TotalAnalyses)
	h.DB.Model(&models.AIAnalysis{}).Where("verdict = ?", models.VerdictVerified).Count(&stats.VerifiedCount)
	h.DB.Model(&models.AIAnalysis{}).Where("verdict = ?", models.VerdictHoax).Count(&stats.HoaxCount)
	h.DB.Model(&models.AIAnalysis{}).Where("verdict = ?", models.VerdictUnconfirmed).Count(&stats.UnconfirmedCount)

	var avgResult struct {
		Avg float64
	}
	h.DB.Model(&models.AIAnalysis{}).Select("AVG(confidence) as avg").Scan(&avgResult)
	stats.AvgConfidence = avgResult.Avg

	utils.RespondWithJSON(w, http.StatusOK, stats)
}

// TestGeminiConnection tests if Gemini AI is properly configured
func (h *Handler) TestGeminiConnection(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("role").(string)
	if role != "admin" {
		utils.RespondWithError(w, http.StatusForbidden, "forbidden: admin only")
		return
	}

	if err := h.GeminiService.ValidateAPIKey(); err != nil {
		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"configured": false,
			"error":      err.Error(),
			"message":    "Please set GEMINI_API_KEY environment variable",
		})
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"configured": true,
		"message":    "Gemini AI is properly configured",
		"model":      "gemini-pro",
	})
}