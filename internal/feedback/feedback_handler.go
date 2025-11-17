package feedback

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FeedbackHandler struct {
	DB *gorm.DB
}

func NewFeedbackHandler(db *gorm.DB) *FeedbackHandler {
	return &FeedbackHandler{DB: db}
}

// CreateFeedback creates a new feedback (public or authenticated)
func (h *FeedbackHandler) CreateFeedback(w http.ResponseWriter, r *http.Request) {
	var req models.CreateFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate
	req.Description = strings.TrimSpace(req.Description)
	if req.Description == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "description is required")
		return
	}

	// Check if feedback type exists and is active
	var feedbackType models.FeedbackType
	if err := h.DB.First(&feedbackType, req.FeedbackTypeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "feedback type not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !feedbackType.IsActive {
		utils.RespondWithError(w, http.StatusBadRequest, "this feedback type is not active")
		return
	}

	// Get user ID from context (optional)
	var userID *string
	isAnonymous := true
	if uid, ok := r.Context().Value("id").(string); ok && uid != "" {
		userID = &uid
		isAnonymous = false
	}

	// If anonymous, email is required
	if isAnonymous && (req.ContactEmail == nil || strings.TrimSpace(*req.ContactEmail) == "") {
		utils.RespondWithError(w, http.StatusBadRequest, "contact email is required for anonymous feedback")
		return
	}

	feedback := models.Feedback{
		UserID:         userID,
		FeedbackTypeID: req.FeedbackTypeID,
		Description:    req.Description,
		IsAnonymous:    isAnonymous,
		ContactEmail:   req.ContactEmail,
	}

	if err := h.DB.Create(&feedback).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to create feedback")
		return
	}

	// Reload with relations
	h.DB.Preload("FeedbackType").First(&feedback, feedback.ID)

	utils.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"message":  "feedback submitted successfully",
		"feedback": feedback,
	})
}

// UploadFeedbackImage uploads image for feedback
func (h *FeedbackHandler) UploadFeedbackImage(w http.ResponseWriter, r *http.Request) {
	feedbackIDStr := chi.URLParam(r, "feedbackId")
	feedbackID, err := strconv.ParseUint(feedbackIDStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid feedback ID")
		return
	}

	// Check if feedback exists
	var feedback models.Feedback
	if err := h.DB.First(&feedback, feedbackID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "feedback not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "file too large (max 10MB)")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "image file is required")
		return
	}
	defer file.Close()

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		utils.RespondWithError(w, http.StatusBadRequest, "file must be an image")
		return
	}

	// Create upload directory if not exists
	uploadDir := "uploads/feedback"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to create upload directory")
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	filepath := filepath.Join(uploadDir, filename)

	// Save file
	dst, err := os.Create(filepath)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	// Update feedback with image path
	imagePath := "/" + filepath
	h.DB.Model(&feedback).Update("image_path", imagePath)

	// Reload
	h.DB.Preload("FeedbackType").First(&feedback, feedbackID)

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "image uploaded successfully",
		"image_path": imagePath,
		"feedback":   feedback,
	})
}

// GetAllFeedbacks gets all feedbacks (admin only)
func (h *FeedbackHandler) GetAllFeedbacks(w http.ResponseWriter, r *http.Request) {
	typeID := r.URL.Query().Get("type_id")

	query := h.DB.Preload("FeedbackType").Preload("User").Preload("AdminUser")

	

	if typeID != "" {
		query = query.Where("feedback_type_id = ?", typeID)
	}

	var feedbacks []models.Feedback
	if err := query.Order("created_at DESC").Find(&feedbacks).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to fetch feedbacks")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"feedbacks": feedbacks,
		"total":     len(feedbacks),
	})
}

// GetFeedbackByID gets a single feedback
func (h *FeedbackHandler) GetFeedbackByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid feedback ID")
		return
	}

	var feedback models.Feedback
	if err := h.DB.Preload("FeedbackType").Preload("User").Preload("AdminUser").
		First(&feedback, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "feedback not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, feedback)
}

// GetMyFeedbacks gets feedbacks by current user
func (h *FeedbackHandler) GetMyFeedbacks(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("id").(string)
	if !ok || userID == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var feedbacks []models.Feedback
	if err := h.DB.Preload("FeedbackType").Preload("AdminUser").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&feedbacks).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to fetch feedbacks")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"feedbacks": feedbacks,
		"total":     len(feedbacks),
	})
}

// RespondToFeedback allows admin to respond to feedback
func (h *FeedbackHandler) RespondToFeedback(w http.ResponseWriter, r *http.Request) {
	adminID, ok := r.Context().Value("id").(string)
	if !ok || adminID == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid feedback ID")
		return
	}

	var req models.AdminResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate
	req.AdminResponse = strings.TrimSpace(req.AdminResponse)
	if req.AdminResponse == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "admin response is required")
		return
	}

	// Check if feedback exists
	var feedback models.Feedback
	if err := h.DB.First(&feedback, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "feedback not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update feedback
	now := time.Now()
	updates := map[string]interface{}{
		"admin_response": req.AdminResponse,
		"responded_by":   adminID,
		"responded_at":   now,
		"updated_at":     now,
	}

	if err := h.DB.Model(&feedback).Updates(updates).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to respond to feedback")
		return
	}

	// Reload
	h.DB.Preload("FeedbackType").Preload("User").Preload("AdminUser").First(&feedback, id)

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "response submitted successfully",
		"feedback": feedback,
	})
}

// DeleteFeedback deletes a feedback (admin only)
func (h *FeedbackHandler) DeleteFeedback(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid feedback ID")
		return
	}

	// Get feedback to delete image file
	var feedback models.Feedback
	if err := h.DB.First(&feedback, id).Error; err == nil {
		// Delete image file if exists
		if feedback.ImagePath != nil && *feedback.ImagePath != "" {
			imagePath := strings.TrimPrefix(*feedback.ImagePath, "/")
			if err := os.Remove(imagePath); err != nil {
				fmt.Printf("[WARN] Failed to delete image file: %v\n", err)
			}
		}
	}

	result := h.DB.Delete(&models.Feedback{}, id)
	if result.Error != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to delete feedback")
		return
	}

	if result.RowsAffected == 0 {
		utils.RespondWithError(w, http.StatusNotFound, "feedback not found")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "feedback deleted successfully",
	})
}