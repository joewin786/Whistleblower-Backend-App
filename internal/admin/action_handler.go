package admin

import (
	"encoding/json"
	"net/http"
	"time"
	"fmt"

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

	action := models.Action{
		ID:                 uuid.NewString(),
		ActionDescription:  in.ActionDescription,
		Department:         in.Department,
		ResponsiblePerson:  in.ResponsiblePerson,
		HandleAt:          in.HandleAt,
		EstimatedCompletion: in.EstimatedCompletion,
		ReportID:           parseUint(reportID),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := h.DB.Create(&action).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, action)
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