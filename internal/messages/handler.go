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


// Fungsi IsRead Style
func (h *Handler) MarkMessageAsRead(w http.ResponseWriter, r *http.Request) {
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

	// Get current user ID (recipient)
	rawID := r.Context().Value("id")
	currentUserID, ok := rawID.(string)
	if !ok || currentUserID == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var msg models.Message
	if err := h.DB.Where("id = ? AND report_id = ?", mid, rid).First(&msg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "message not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// ⚠️ Jangan mark as read jika user adalah pengirim pesan
	if msg.SenderID != nil && *msg.SenderID == currentUserID {
		utils.RespondWithError(w, http.StatusBadRequest, "cannot mark own message as read")
		return
	}

	// Update IsRead & ReadAt
	now := time.Now()
	msg.IsRead = true
	msg.ReadAt = &now

	if err := h.DB.Save(&msg).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update message")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, msg)
}

// ✅ Mark all messages in a report as read
func (h *Handler) MarkAllMessagesAsRead(w http.ResponseWriter, r *http.Request) {
	ridStr := chi.URLParam(r, "reportId")
	rid, ok := parseUintID(ridStr)
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report id")
		return
	}

	// Get current user ID
	rawID := r.Context().Value("id")
	currentUserID, ok := rawID.(string)
	if !ok || currentUserID == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	now := time.Now()

	// ✅ Hanya mark as read pesan yang BUKAN dari user sendiri
	result := h.DB.Model(&models.Message{}).
		Where("report_id = ? AND is_read = ? AND (sender_id IS NULL OR sender_id != ?)", 
			rid, false, currentUserID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		})

	if result.Error != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update messages")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":       "all messages marked as read",
		"rows_affected": result.RowsAffected,
	})
}

// ✅ Get unread message count for a report
func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	ridStr := chi.URLParam(r, "reportId")
	rid, ok := parseUintID(ridStr)
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report id")
		return
	}

	// Get current user ID
	rawID := r.Context().Value("id")
	currentUserID, ok := rawID.(string)
	if !ok || currentUserID == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var count int64
	// Hitung pesan yang belum dibaca dan BUKAN dari user sendiri
	if err := h.DB.Model(&models.Message{}).
		Where("report_id = ? AND is_read = ? AND (sender_id IS NULL OR sender_id != ?)", 
			rid, false, currentUserID).
		Count(&count).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to count messages")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"report_id":    rid,
		"unread_count": count,
	})
}

// ✅ Get read status for specific message (untuk realtime update)
func (h *Handler) GetMessageReadStatus(w http.ResponseWriter, r *http.Request) {
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
	if err := h.DB.Select("id, is_delivered, is_read, read_at").
		Where("id = ? AND report_id = ?", mid, rid).
		First(&msg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "message not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	status := map[string]interface{}{
		"message_id":   msg.ID,
		"is_delivered": msg.IsDelivered,
		"is_read":      msg.IsRead,
		"read_at":      msg.ReadAt,
	}

	utils.RespondWithJSON(w, http.StatusOK, status)
}

// 🆕 Update/Edit message
func (h *Handler) UpdateMessage(w http.ResponseWriter, r *http.Request) {
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

	// Get current user ID
	rawID := r.Context().Value("id")
	currentUserID, ok := rawID.(string)
	if !ok || currentUserID == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Find existing message
	var msg models.Message
	if err := h.DB.Where("id = ? AND report_id = ?", mid, rid).First(&msg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "message not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// ⚠️ Hanya sender yang bisa edit pesan
	if msg.SenderID == nil || *msg.SenderID != currentUserID {
		utils.RespondWithError(w, http.StatusForbidden, "you can only edit your own messages")
		return
	}

	// Parse request body
	var input struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Message == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "message cannot be empty")
		return
	}

	// Update message
	msg.Message = input.Message
	
	if err := h.DB.Save(&msg).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update message")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, msg)
}