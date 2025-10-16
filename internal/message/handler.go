package messages

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"whistleblower_REST/internal/utils"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

// POST /reports/{reportId}/messages
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")
	uid, _ := r.Context().Value("uid").(string)
	if uid == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var in CreateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	msg := &Message{
		ID:       uuid.NewString(),
		ReportID: reportID,
		SenderID: uid,
		Message:  in.Message,
	}

	if err := h.DB.Create(msg).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, msg)
}

// GET /reports/{reportId}/messages
func (h *Handler) GetByReport(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")

	var messages []Message
	if err := h.DB.Where("report_id = ?", reportID).
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, messages)
}

// GET /reports/{reportId}/messages/{messageId}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	messageID := chi.URLParam(r, "messageId")

	var msg Message
	if err := h.DB.First(&msg, "id = ?", messageID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "message not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, msg)
}

// DELETE /reports/{reportId}/messages/{messageId}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	messageID := chi.URLParam(r, "messageId")

	res := h.DB.Delete(&Message{}, "id = ?", messageID)
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
