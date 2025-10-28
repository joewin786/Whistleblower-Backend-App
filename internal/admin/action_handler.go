package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"
)

type ActionHandler struct {
	DB *gorm.DB
}

func (h *ActionHandler) CreateAction(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")

	var in models.CreateActionRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if in.ActionDescription == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "action_description is required")
		return
	}

	// jika HandleAt tidak diisi, isi otomatis waktu sekarang
	handleAt := in.HandleAt
	if handleAt == nil {
		now := time.Now()
		handleAt = &now
	}

	action := models.Action{
		ID:                  uuid.NewString(),
		ActionDescription:   in.ActionDescription,
		Department:          in.Department,
		ResponsiblePerson:   in.ResponsiblePerson,
		HandleAt:            handleAt,
		EstimatedCompletion: in.EstimatedCompletion,
		ReportID:            parseUint(reportID),
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// Simpan action baru
	if err := h.DB.Create(&action).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	
	if err := h.DB.Model(&models.Report{}).
	Where("id = ?", action.ReportID).
	Updates(map[string]any{
		"status":     "on_process", // ubah dari resolved ke on_process
		"updated_at": time.Now(),
	}).Error; err != nil {
	utils.RespondWithError(w, http.StatusInternalServerError, "failed to update report status")
	return
}

fmt.Printf("[INFO] Report #%d marked as 'on_process' after action creation\n", action.ReportID)
}

func (h *ActionHandler) MarkActionCompleted(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")

	// Cek apakah ada action untuk report ini
	var action models.Action
	if err := h.DB.Where("report_id = ?", reportID).First(&action).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "no action found for this report")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	now := time.Now()
	// Update waktu selesai tindakan
	if err := h.DB.Model(&models.Action{}).
		Where("report_id = ?", reportID).
		Updates(map[string]any{
			"estimated_completion": now,
			"updated_at":           now,
		}).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update action completion time")
		return
	}

	// ✅ Ubah status laporan jadi "resolved"
	if err := h.DB.Model(&models.Report{}).
		Where("id = ?", reportID).
		Updates(map[string]any{
			"status":     "resolved",
			"updated_at": now,
		}).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update report status to resolved")
		return
	}

	fmt.Printf("[INFO] Report #%s marked as resolved at %s\n", reportID, now.Format(time.RFC3339))

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "action completed and report marked as resolved",
	})
}


func (h *ActionHandler) GetActionsByReport(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")

	var list []models.Action
	if err := h.DB.Where("report_id = ?", reportID).Order("created_at DESC").Find(&list).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, list)
}

func parseUint(s string) uint {
	var id uint
	fmt.Sscan(s, &id)
	return id
}
