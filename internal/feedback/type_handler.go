package feedback


import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type TypeHandler struct {
	DB *gorm.DB
}

func NewTypeHandler(db *gorm.DB) *TypeHandler {
	return &TypeHandler{DB: db}
}

// CreateFeedbackType creates a new feedback type (admin only)
func (h *TypeHandler) CreateFeedbackType(w http.ResponseWriter, r *http.Request) {
	var req models.CreateFeedbackTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Check duplicate
	var existing models.FeedbackType
	if err := h.DB.Where("LOWER(name) = LOWER(?)", req.Name).First(&existing).Error; err == nil {
		utils.RespondWithError(w, http.StatusConflict, "feedback type with this name already exists")
		return
	}

	feedbackType := models.FeedbackType{
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		IsActive:    true,
	}

	if err := h.DB.Create(&feedbackType).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to create feedback type")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"message":       "feedback type created successfully",
		"feedback_type": feedbackType,
	})
}

// GetAllFeedbackTypes gets all feedback types
func (h *TypeHandler) GetAllFeedbackTypes(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active_only") == "true"

	var feedbackTypes []models.FeedbackType
	query := h.DB.Order("name ASC")

	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&feedbackTypes).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to fetch feedback types")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"feedback_types": feedbackTypes,
		"total":          len(feedbackTypes),
	})
}

// GetFeedbackTypeByID gets a single feedback type
func (h *TypeHandler) GetFeedbackTypeByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid feedback type ID")
		return
	}

	var feedbackType models.FeedbackType
	if err := h.DB.First(&feedbackType, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "feedback type not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, feedbackType)
}

// UpdateFeedbackType updates a feedback type (admin only)
func (h *TypeHandler) UpdateFeedbackType(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid feedback type ID")
		return
	}

	var req models.UpdateFeedbackTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Check if exists
	var feedbackType models.FeedbackType
	if err := h.DB.First(&feedbackType, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "feedback type not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build updates map
	updates := make(map[string]interface{})
	updated := false

	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		// Check duplicate name
		var existing models.FeedbackType
		if err := h.DB.Where("LOWER(name) = LOWER(?) AND id != ?", *req.Name, id).First(&existing).Error; err == nil {
			utils.RespondWithError(w, http.StatusConflict, "feedback type with this name already exists")
			return
		}
		updates["name"] = strings.TrimSpace(*req.Name)
		updated = true
	}

	if req.Description != nil {
		updates["description"] = *req.Description
		updated = true
	}

	if req.Icon != nil {
		updates["icon"] = *req.Icon
		updated = true
	}

	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
		updated = true
	}

	if !updated {
		utils.RespondWithError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	updates["updated_at"] = time.Now()

	if err := h.DB.Model(&feedbackType).Updates(updates).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update feedback type")
		return
	}

	// Reload
	h.DB.First(&feedbackType, id)

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":       "feedback type updated successfully",
		"feedback_type": feedbackType,
	})
}

// DeleteFeedbackType deletes a feedback type (admin only)
func (h *TypeHandler) DeleteFeedbackType(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid feedback type ID")
		return
	}

	// Check if has related feedbacks
	var count int64
	h.DB.Model(&models.Feedback{}).Where("feedback_type_id = ?", id).Count(&count)
	if count > 0 {
		utils.RespondWithError(w, http.StatusConflict, "cannot delete feedback type with existing feedbacks. Set is_active to false instead.")
		return
	}

	result := h.DB.Delete(&models.FeedbackType{}, id)
	if result.Error != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to delete feedback type")
		return
	}

	if result.RowsAffected == 0 {
		utils.RespondWithError(w, http.StatusNotFound, "feedback type not found")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "feedback type deleted successfully",
	})
}