package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
	"whistleblower_REST/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB *gorm.DB
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ValidateTokenRequest struct {
	Token string `json:"token"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type MeResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type User struct {
	ID        string    `gorm:"primaryKey;type:text" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	Role      string    `gorm:"default:user" json:"role"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body. Please provide valid JSON data.")
		return
	}

	var existing User
	if err := h.DB.Where("email = ?", input.Email).First(&existing).Error; err == nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Email is already registered. Please use a different email.")
		return
	}

	hashed, err := HashFunction(input.Password)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to process password. Please try again later.")
		return
	}

	user := User{
		ID:       uuid.NewString(),
		Name:     input.Name,
		Email:    input.Email,
		Password: hashed,
	}

	if err := h.DB.Create(&user).Error; err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Failed to create user account. Please try again.")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, map[string]string{
		"message": "User registered successfully",
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body. Please provide valid JSON data.")
		return
	}

	var user User
	if err := h.DB.
		Select("id", "password").
		Where("email = ?", input.Email).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid email or password.")
			return
		}
		utils.RespondWithError(w, http.StatusBadRequest, "An unexpected error occurred. Please try again.")
		return
	}

	if !CheckPasswordHash(input.Password, user.Password) {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid email or password.")
		return
	}

	accessToken, err := GenerateToken(user.ID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate access token.")
		return
	}
	refreshToken, err := GenerateRefreshToken(user.ID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate refresh token.")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	id, ok := r.Context().Value("id").(string)
	if !ok || id == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var user MeResponse

	err := h.DB.
		Table("users").
		Select("id", "name", "email", "role", "created_at", "updated_at").
		Where("id = ?", id).
		Take(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.RespondWithError(w, http.StatusNotFound, "User not found.")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve user information.")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body. Refresh token is required.")
		return
	}

	id, err := ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid or expired refresh token.")
		return
	}

	accessToken, err := GenerateToken(id)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Invalid or expired refresh token.")
		return
	}
	newRefreshToken, err := GenerateRefreshToken(id)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate new refresh token.")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	})
}

func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	var req ValidateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body. Token is required.")
		return
	}

	_, err := ValidateToken(req.Token)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token.")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"valid": true,
	})
}
