package reports

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

func NewHandler(db *gorm.DB) *Handler { return &Handler{DB: db} }

// POST /reports
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	uid, _ := r.Context().Value("uid").(string)
	if uid == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	rp := &Report{
		ID:          uuid.NewString(),
		UserUID:     uid,
		Title:       in.Title,
		Description: in.Description,
		Category:    in.Category,
		Status:      "OPEN",
	}
	if err := h.DB.Create(rp).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusCreated, rp)
}

// GET /reports?status=&category=
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	category := r.URL.Query().Get("category")

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

// GET /reports/my
func (h *Handler) GetMy(w http.ResponseWriter, r *http.Request) {
	uid, _ := r.Context().Value("uid").(string)
	if uid == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var list []Report
	if err := h.DB.Where("user_uid = ?", uid).Order("created_at DESC").Find(&list).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, list)
}

// GET /reports/{reportId}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "reportId")
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
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "reportId")
	var in UpdateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	updates := map[string]any{}
	if in.Title != nil {
		updates["title"] = *in.Title
	}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.Category != nil {
		updates["category"] = *in.Category
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if len(updates) == 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	if err := h.DB.Model(&Report{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "report updated"})
}

// DELETE /reports/{reportId}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "reportId")
	res := h.DB.Delete(&Report{}, "id = ?", id)
	if res.Error != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, res.Error.Error())
		return
	}
	if res.RowsAffected == 0 {
		utils.RespondWithError(w, http.StatusNotFound, "not found")
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "report deleted"})
}
