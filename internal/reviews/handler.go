package reviews

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"whistleblower_REST/internal/auth"
	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

// Create creates a new review for a report
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	// Get admin ID from context
	var adminID uint
	if adminIDUint, ok := auth.GetAdminIDFromContext(r.Context()); ok {
		adminID = adminIDUint
	} else {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized: admin ID not found")
		return
	}

	// Check if user is admin
	role, _ := auth.GetRoleFromContext(r.Context())
	if role != "admin" && role != "superadmin" {
		utils.RespondWithError(w, http.StatusForbidden, "forbidden: only admins can create reviews")
		return
	}

	var req models.CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate report exists
	var report models.Report
	if err := h.DB.First(&report, req.ReportID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "report not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check if review already exists for this report
	var existingReview models.Review
	if err := h.DB.Where("report_id = ?", req.ReportID).First(&existingReview).Error; err == nil {
		utils.RespondWithError(w, http.StatusConflict, "review already exists for this report")
		return
	}

	// Create review
	review := models.Review{
		ReportID:          req.ReportID,
		AdminID:           adminID,
		CredibilityScore:  req.CredibilityScore,
		EvidenceQuality:   req.EvidenceQuality,
		ConsistencyScore:  req.ConsistencyScore,
		SourceReliability: req.SourceReliability,
		UrgencyLevel:      req.UrgencyLevel,
		ReviewNotes:       req.ReviewNotes,
	}

	// Calculate overall score
	review.CalculateOverallScore()

	if err := h.DB.Create(&review).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to create review")
		return
	}

	// Update report status to under_review if not already
	if report.Status != models.StatusUnderReview {
		h.DB.Model(&report).Updates(map[string]interface{}{
			"status":     models.StatusUnderReview,
			"updated_at": time.Now(),
		})
	}

	// Load admin info for response
	h.DB.Preload("Admin").First(&review, review.ID)

	fmt.Printf("[INFO] ✅ Review created for Report #%d by Admin %s (Overall Score: %.2f)\n",
		req.ReportID, adminID, review.OverallScore)

	utils.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "review created successfully",
		"review":  review,
	})
}

// GetByReportID gets the review for a specific report
func (h *Handler) GetByReportID(w http.ResponseWriter, r *http.Request) {
	reportIDStr := chi.URLParam(r, "reportId")
	reportID, err := strconv.ParseUint(reportIDStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report ID")
		return
	}

	var review models.Review
	if err := h.DB.Preload("Admin").Preload("Report").
		Where("report_id = ?", reportID).First(&review).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "review not found for this report")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, review)
}

// Update updates an existing review
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	// Get admin ID from context
	var adminID uint
	if adminIDUint, ok := auth.GetAdminIDFromContext(r.Context()); ok {
		adminID = adminIDUint
	} else {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized: admin ID not found")
		return
	}

	reportIDStr := chi.URLParam(r, "reportId")
	reportID, err := strconv.ParseUint(reportIDStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report ID")
		return
	}

	var req models.UpdateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get existing review
	var review models.Review
	if err := h.DB.Where("report_id = ?", reportID).First(&review).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "review not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check if admin owns this review or is super admin
	role, _ := auth.GetRoleFromContext(r.Context())
	if review.AdminID != adminID && role != "superadmin" {
		utils.RespondWithError(w, http.StatusForbidden, "you can only update your own reviews")
		return
	}

	// Update fields
	updates := make(map[string]interface{})
	updated := false

	if req.CredibilityScore != nil {
		review.CredibilityScore = *req.CredibilityScore
		updated = true
	}
	if req.EvidenceQuality != nil {
		review.EvidenceQuality = *req.EvidenceQuality
		updated = true
	}
	if req.ConsistencyScore != nil {
		review.ConsistencyScore = *req.ConsistencyScore
		updated = true
	}
	if req.SourceReliability != nil {
		review.SourceReliability = *req.SourceReliability
		updated = true
	}
	if req.UrgencyLevel != nil {
		review.UrgencyLevel = *req.UrgencyLevel
		updated = true
	}
	if req.ReviewNotes != nil {
		review.ReviewNotes = *req.ReviewNotes
		updated = true
	}

	if !updated {
		utils.RespondWithError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	// Recalculate overall score
	review.CalculateOverallScore()
	updates["overall_score"] = review.OverallScore
	updates["updated_at"] = time.Now()

	if err := h.DB.Model(&review).Updates(updates).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update review")
		return
	}

	// Reload with relations
	h.DB.Preload("Admin").First(&review, review.ID)

	fmt.Printf("[INFO] ✅ Review updated for Report #%d (New Score: %.2f)\n",
		reportID, review.OverallScore)

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "review updated successfully",
		"review":  review,
	})
}

// Delete deletes a review (admin only)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var adminID uint
	if adminIDUint, ok := auth.GetAdminIDFromContext(r.Context()); ok {
		adminID = adminIDUint
	} else {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized: admin ID not found")
		return
	}

	reportIDStr := chi.URLParam(r, "reportId")
	reportID, err := strconv.ParseUint(reportIDStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report ID")
		return
	}

	// Get existing review
	var review models.Review
	if err := h.DB.Where("report_id = ?", reportID).First(&review).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "review not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check ownership
	role, _ := auth.GetRoleFromContext(r.Context())
	if review.AdminID != adminID && role != "superadmin" {
		utils.RespondWithError(w, http.StatusForbidden, "you can only delete your own reviews")
		return
	}

	if err := h.DB.Delete(&review).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to delete review")
		return
	}

	fmt.Printf("[INFO] 🗑️ Review deleted for Report #%d by Admin %s\n", reportID, adminID)

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "review deleted successfully",
	})
}

// GetAll gets all reviews (admin only) with pagination
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	role, _ := auth.GetRoleFromContext(r.Context())
	if role != "admin" && role != "superadmin" {
		utils.RespondWithError(w, http.StatusForbidden, "forbidden: admin only")
		return
	}

	var reviews []models.Review
	if err := h.DB.Preload("Admin").Preload("Report").
		Order("reviewed_at DESC").Find(&reviews).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"reviews": reviews,
		"total":   len(reviews),
	})
}
