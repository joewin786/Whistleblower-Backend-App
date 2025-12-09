package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"whistleblower_REST/internal/utils"

	"github.com/go-chi/chi/v5"
)

type AdminAuthHandler struct {
	service *AdminAuthService
}

func NewAdminAuthHandler(service *AdminAuthService) *AdminAuthHandler {
	return &AdminAuthHandler{service: service}
}

// AdminLogin handles admin/superadmin login
func (h *AdminAuthHandler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Validasi input
	if req.Email == "" || req.Password == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "email and password required")
		return
	}

	token, admin, err := h.service.AdminLogin(req.Email, req.Password)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Response dengan token dan data admin (tanpa password)
	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"admin": map[string]interface{}{
			"id":         admin.ID,
			"full_name":  admin.FullName,
			"email":      admin.Email,
			"department": admin.Department,
			"role":       admin.Role,
		},
	})
}

// CreateAdmin - hanya superadmin
func (h *AdminAuthHandler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FullName   string `json:"full_name"`
		Email      string `json:"email"`
		Password   string `json:"password"`
		Department string `json:"department"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Validasi input
	if req.FullName == "" || req.Email == "" || req.Password == "" || req.Department == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "all fields are required")
		return
	}

	if err := h.service.CreateAdmin(req.FullName, req.Email, req.Password, req.Department); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, map[string]string{"message": "admin created successfully"})
}

// CreateInvestigator - superadmin dan admin
func (h *AdminAuthHandler) CreateInvestigator(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FullName   string `json:"full_name"`
		Email      string `json:"email"`
		Department string `json:"department"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Validasi input
	if req.FullName == "" || req.Email == "" || req.Department == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "full_name, email, and department are required")
		return
	}

	if err := h.service.CreateInvestigator(req.FullName, req.Email, req.Department); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, map[string]string{"message": "investigator created successfully"})
}

// GetAllAdmins - list semua admin dan investigator
func (h *AdminAuthHandler) GetAllAdmins(w http.ResponseWriter, r *http.Request) {
	admins, err := h.service.GetAllAdmins()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to fetch admins")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, admins)
}

// GetAdminByID - get admin by ID
func (h *AdminAuthHandler) GetAdminByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid id")
		return
	}

	admin, err := h.service.GetAdminByID(uint(id))
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "admin not found")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, admin)
}

// UpdateAdmin - update admin/investigator
func (h *AdminAuthHandler) UpdateAdmin(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		FullName   string `json:"full_name"`
		Email      string `json:"email"`
		Department string `json:"department"`
		IsActive   bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if err := h.service.UpdateAdmin(uint(id), req.FullName, req.Email, req.Department, req.IsActive); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update admin")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "admin updated successfully"})
}

// ChangePassword - ubah password admin
func (h *AdminAuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if req.NewPassword == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "new_password is required")
		return
	}

	if err := h.service.ChangeAdminPassword(uint(id), req.NewPassword); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to change password")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "password changed successfully"})
}

// DeleteAdmin - soft delete
func (h *AdminAuthHandler) DeleteAdmin(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.DeleteAdmin(uint(id)); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "admin deleted successfully"})
}

// GetMe - get current logged in admin profile
// GetMe - get current logged in admin profile
func (h *AdminAuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// ✅ Try get admin_id first (set by middleware for admin tokens)
	adminID, ok := GetAdminIDFromContext(r.Context())
	if ok {
		admin, err := h.service.GetAdminByID(adminID)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "admin not found")
			return
		}
		utils.RespondWithJSON(w, http.StatusOK, admin)
		return
	}

	// ✅ Fallback: try parse from string id
	idStr, ok := GetIDFromContext(r.Context())
	if !ok || idStr == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Convert string to uint
	var fallbackAdminID uint
	if _, err := fmt.Sscanf(idStr, "%d", &fallbackAdminID); err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "invalid admin id")
		return
	}

	admin, err := h.service.GetAdminByID(fallbackAdminID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "admin not found")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, admin)
}

// GetAllInvestigators - hanya mengembalikan role = investigator
func (h *AdminAuthHandler) GetAllInvestigators(w http.ResponseWriter, r *http.Request) {
	investigators, err := h.service.GetAllInvestigators()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to fetch investigators")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, investigators)
}
