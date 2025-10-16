package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryHandler struct {
	DB *gorm.DB
}

func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value("role").(string)
	if !ok || role != "admin" {
		utils.RespondWithError(w, http.StatusForbidden, "Access denied. Only admins can create categories.")
		return
	}

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body. Ensure JSON structure is correct.")
		return
	}

	if input.Name == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Category name is required.")
		return
	}

	var existing models.Category
	if err := h.DB.Where("name = ?", input.Name).First(&existing).Error; err == nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Category name already exists.")
		return
	}

	category := models.Category{
		ID:          uuid.NewString(),
		Name:        input.Name,
		Description: input.Description,
	}

	if err := h.DB.Create(&category).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create category.")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, category)
}

func (h *CategoryHandler) GetAllCategories(w http.ResponseWriter, r *http.Request) {
	var categories []models.Category
	if err := h.DB.Find(&categories).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch categories.")
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, categories)
}

func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value("role").(string)
	if !ok || role != "admin" {
		utils.RespondWithError(w, http.StatusForbidden, "Access denied. Only admins can update categories.")
		return
	}

	id := utils.GetParam(r, "id")
	if id == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Missing category ID.")
		return
	}

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}

	var category models.Category
	if err := h.DB.First(&category, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.RespondWithError(w, http.StatusNotFound, "Category not found.")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update category.")
		return
	}

	if input.Name != "" {
		category.Name = input.Name
	}
	category.Description = input.Description

	if err := h.DB.Save(&category).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update category.")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, category)
}

func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value("role").(string)
	if !ok || role != "admin" {
		utils.RespondWithError(w, http.StatusForbidden, "Access denied. Only admins can delete categories.")
		return
	}

	id := utils.GetParam(r, "id")
	if id == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Missing category ID.")
		return
	}

	if err := h.DB.Delete(&models.Category{}, "id = ?", id).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete category.")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Category deleted successfully."})
}
