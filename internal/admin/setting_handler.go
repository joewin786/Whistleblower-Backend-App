package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"
	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"
)

type SettingsHandler struct {
	DB *gorm.DB
}

func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	var settings []models.Setting
	if err := h.DB.Find(&settings).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, settings)
}

func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var in []models.Setting
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	for _, s := range in {
		h.DB.Model(&models.Setting{}).
			Where("key = ?", s.Key).
			Updates(map[string]any{"value": s.Value, "updated_at": time.Now()})
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Settings updated"})
}
