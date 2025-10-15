package evidence

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"whistleblower_REST/internal/utils"
)

type Handler struct {
	repo Repository
}

func NewHandler(r Repository) *Handler {
	return &Handler{repo: r}
}

// POST /reports/{reportId}/evidence
func (h *Handler) Create(w http.ResponseWriter, r *http.Request, reportID string) {
	var req CreateEvidenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	e := &Evidence{
		ID:       uuid.NewString(),
		ReportID: reportID,
		FilePath: req.FilePath,
	}

	if err := h.repo.Create(r.Context(), e); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, e)
}

// GET /reports/{reportId}/evidence
func (h *Handler) GetByReport(w http.ResponseWriter, r *http.Request, reportID string) {
	list, err := h.repo.GetByReport(r.Context(), reportID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, list)
}

// DELETE /reports/{reportId}/evidence/{evidenceId}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request, evidenceID string) {
	if err := h.repo.Delete(r.Context(), evidenceID); err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "not found")
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "evidence deleted"})
}
