package reports

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
	"whistleblower_REST/internal/utils"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {return &Handler{DB: db}}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	if in.Title == "" || in.Description == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "title and description are required")
		return
	}

	var uid string
	if v := r.Context().Value("uid"); v != nil {
		if s, ok := v.(string); ok {
			uid = s
		}
	}

	reporterType := ReporterAnonymous
	var userID *string
	if uid != "" {
		reporterType = ReporterAuthenticated
		userID = &uid
		in.Email = nil
	}

	rp := Report{
		Title: in.Title,
		Description: in.Description,
		Category: in.Category,
		Status: StatusSubmitted,
		ReporterType: reporterType,
		UserID: userID,
		Email: in.Email,
	}

	if err := h.DB.Create(&rp).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusCreated, rp)
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	category := strings.TrimSpace(r.URL.Query().Get("category"))

	var list []Report
	tx := h.DB.Model(&Report{})
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if category != "" {
		tx = tx.Where("category = ?", category)
	}
	if err := tx.Order("created_at DESC").Find(&list).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, list)
}

func (h *Handler) GetMy(w http.ResponseWriter, r *http.Request) {
	v := r.Context().Value("uid")
	uid, _ := v.(string)
	if uid == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var list []Report
	if err := h.DB.Where("user_id = ?", uid).Order("created_at DESC").Find(&list).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, list)

}


	func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUintID(chi.URLParam(r, "reportId"))
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report id")
		return
	}
	var rp Report
	if err := h.DB.First(&rp, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "report not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, rp)
}

// PATCH /reports/{reportId}
// Hanya update Status (sesuai OpenAPI). updated_at otomatis oleh GORM.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUintID(chi.URLParam(r, "reportId"))
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report id")
		return
	}
	var in UpdateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	updates := map[string]any{
		"updated_at": time.Now(),
	}
	if in.Status != nil {
		if !isValidStatus(*in.Status) {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid status value")
			return
		}
		updates["status"] = *in.Status
	}
	if len(updates) == 1 { // hanya updated_at terisi
		utils.RespondWithError(w, http.StatusBadRequest, "no updatable fields provided")
		return
	}

	res := h.DB.Model(&Report{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, res.Error.Error())
		return
	}
	if res.RowsAffected == 0 {
		utils.RespondWithError(w, http.StatusNotFound, "report not found")
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "report updated"})
}

// DELETE /reports/{reportId} (opsional; kalau router kamu pakai)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUintID(chi.URLParam(r, "reportId"))
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report id")
		return
	}
	res := h.DB.Delete(&Report{}, "id = ?", id)
	if res.Error != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, res.Error.Error())
		return
	}
	if res.RowsAffected == 0 {
		utils.RespondWithError(w, http.StatusNotFound, "report not found")
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "report deleted"})
}

// === helpers ===

func parseUintID(s string) (uint, bool) {
	if s == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(n), true
}

func isValidStatus(st string) bool {
	switch st {
	case StatusSubmitted, StatusUnderReview, StatusResolved, StatusDismissed:
		return true
	default:
		return false
	}
}