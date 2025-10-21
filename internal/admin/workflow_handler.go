package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"
	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"
)

type WorkflowHandler struct {
	DB *gorm.DB
}

func (h *WorkflowHandler) GetWorkflows(w http.ResponseWriter, r *http.Request) {
	var wf []models.Workflow
	if err := h.DB.Find(&wf).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, wf)
}

func (h *WorkflowHandler) UpdateWorkflows(w http.ResponseWriter, r *http.Request) {
	var in []models.Workflow
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	for _, wflow := range in {
		h.DB.Model(&models.Workflow{}).
			Where("id = ?", wflow.ID).
			Updates(map[string]any{
				"name":       wflow.Name,
				"order":      wflow.Order,
				"is_active":  wflow.IsActive,
				"updated_at": time.Now(),
			})
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Workflow updated"})
}
