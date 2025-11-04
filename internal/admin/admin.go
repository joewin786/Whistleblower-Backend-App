package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"

	"gorm.io/gorm"
)

type AdminHandler struct {
	DB *gorm.DB
}

// GET /admin/list
// Bisa filter dengan query ?department=IT
func (h *AdminHandler) GetAdmins(w http.ResponseWriter, r *http.Request) {
	dept := strings.TrimSpace(r.URL.Query().Get("department"))
	var admins []models.Admin

	query := h.DB.Model(&models.Admin{}).Where("is_active = ?", true)
	if dept != "" {
		query = query.Where("department = ?", dept)
	}

	if err := query.Order("full_name ASC").Find(&admins).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, admins)
}

// POST /admin/create
func (h *AdminHandler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	var in models.Admin
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	in.FullName = strings.TrimSpace(in.FullName)
	in.Department = strings.TrimSpace(in.Department)
	in.Email = strings.TrimSpace(in.Email)

	if in.FullName == "" || in.Email == "" || in.Department == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "name, email, and department are required")
		return
	}

	if err := h.DB.Create(&in).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, in)
}
