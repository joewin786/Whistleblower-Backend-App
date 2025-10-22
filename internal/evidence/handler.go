package evidence

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler { return &Handler{DB: db} }

// GET /reports/{reportId}/evidence
func (h *Handler) GetByReport(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")
	var list []models.Evidence
	if err := h.DB.Where("report_id = ?", reportID).Order("created_at DESC").Find(&list).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, list)
}

// POST /reports/{reportId}/evidence
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")

	// limit in-memory upload to 20MB
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid form data: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "File missing: "+err.Error())
		return
	}
	defer file.Close()

	// Ensure upload directory exists
	if err := os.MkdirAll("uploads", os.ModePerm); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create upload dir: "+err.Error())
		return
	}

	// Generate unique filename to avoid collision
	filename := fmt.Sprintf("%s_%s", uuid.NewString(), header.Filename)
	dst := filepath.Join("uploads", filename)

	out, err := os.Create(dst)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to save file: "+err.Error())
		return
	}
	defer out.Close()

	if _, err = io.Copy(out, file); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to write file: "+err.Error())
		return
	}

	// Save record in DB
	ev := &models.Evidence{
		ID:       uuid.NewString(),
		ReportID: reportID,
		FilePath: dst,
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

	var ev models.Evidence
	if err := h.DB.First(&ev, "id = ?", evidenceID).Error; err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Evidence not found")
		return
	}

	// Delete file from disk
	if err := os.Remove(ev.FilePath); err != nil && !os.IsNotExist(err) {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete file: "+err.Error())
		return
	}

	// Delete from DB
	if err := h.DB.Delete(&ev).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Evidence deleted"})
}

// GET /reports/{reportId}/evidence/file/{id}
func (h *Handler) DownloadEvidence(w http.ResponseWriter, r *http.Request) {
	evidenceID := chi.URLParam(r, "id")

	var evidence models.Evidence
	if err := h.DB.First(&evidence, "id = ?", evidenceID).Error; err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "File not found")
		return
	}

	http.ServeFile(w, r, evidence.FilePath)
}
