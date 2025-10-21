package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"
)

type RoleHandler struct {
	DB *gorm.DB
}

func (h *RoleHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
	var roles []models.Role
	if err := h.DB.Find(&roles).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, roles)
}

func (h *RoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var role models.Role
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if role.Name == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "role name required")
		return
	}
	role.CreatedAt = time.Now()
	role.UpdateAt = time.Now()

	if err := h.DB.Create(&role).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusCreated, role)
}

func (h *RoleHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "roleId")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid role id")
		return
	}
	var in models.Role
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.DB.Model(&models.Role{}).
		Where("id = ?", id).
		Updates(map[string]any{"name": in.Name, "updated_at": time.Now()}).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Role updated"})
}
