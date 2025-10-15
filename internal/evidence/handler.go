package evidence

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

// GET /reports/{reportId}/evidence
func (h *Handler) GetByReport(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")
	var list []Evidence
	if err := h.DB.Where("report_id = ?", reportID).Order("created_at DESC").Find(&list).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, list)
}

// POST /reports/{reportId}/evidence
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")

	var in CreateEvidenceRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	ev := &Evidence{
		ID:       uuid.NewString(),
		ReportID: reportID,
		FilePath: in.FilePath,
	}

	if err := h.DB.Create(ev).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusCreated, ev)
}

// DELETE /reports/{reportId}/evidence/{evidenceId}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	evidenceID := chi.URLParam(r, "evidenceId")
	res := h.DB.Delete(&Evidence{}, "id = ?", evidenceID)
	if res.Error != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, res.Error.Error())
		return
	}
	if res.RowsAffected == 0 {
		utils.RespondWithError(w, http.StatusNotFound, "not found")
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "evidence deleted"})
}
