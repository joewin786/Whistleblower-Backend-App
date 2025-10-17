package messages

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"whistleblower_REST/internal/utils"
	"whistleblower_REST/internal/models"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler { return &Handler{DB: db} }


func parseUintID(s string) (uint, bool) {
	var out uint
	if s == "" {
		return 0, false
	}
	if _, err := fmt.Sscan(s, &out); err != nil {
		return 0, false
	}
	return out, true
}


func (h *Handler) GetByReportID(w http.ResponseWriter, r *http.Request) {
	ridStr := chi.URLParam(r, "reportId")
	rid, ok := parseUintID(ridStr)
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report id")
		return
	}

	var msgs []models.Message
	if err := h.DB.
		Where("report_id = ?", rid).
		Order("created_at ASC").
		Find(&msgs).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, msgs)
}


func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ridStr := chi.URLParam(r, "reportId")
	rid, ok := parseUintID(ridStr)
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report id")
		return
	}

	rawID := r.Context().Value("id") // ✅ konsisten: rawID
	fmt.Printf("🔍 Raw ID from context: %#v (type: %T)\n", rawID, rawID)

	uid, ok := rawID.(string)
	if !ok || uid == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized: invalid or missing user ID")
		return
	}

	var in models.CreateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Message == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "message is required")
		return
	}

	msg := models.Message{
		ID:        uuid.NewString(),
		ReportID:  rid,
		SenderID:  &uid,
		Message:   in.Message,
		CreatedAt: time.Now(),
	}

	if err := h.DB.Create(&msg).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusCreated, msg)
}


func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	ridStr := chi.URLParam(r, "reportId")
	rid, ok := parseUintID(ridStr)
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report id")
		return
	}
	mid := chi.URLParam(r, "messageId")
	if mid == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid message id")
		return
	}

	var msg models.Message
	if err := h.DB.
		Where("id = ? AND report_id = ?", mid, rid).
		First(&msg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "message not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, msg)
}


func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ridStr := chi.URLParam(r, "reportId")
	rid, ok := parseUintID(ridStr)
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report id")
		return
	}
	mid := chi.URLParam(r, "messageId")
	if mid == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid message id")
		return
	}

	

	res := h.DB.Where("id = ? AND report_id = ?", mid, rid).Delete(&models.Message{})
	if res.Error != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, res.Error.Error())
		return
	}
	if res.RowsAffected == 0 {
		utils.RespondWithError(w, http.StatusNotFound, "message not found")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "deleted successfully"})
}
