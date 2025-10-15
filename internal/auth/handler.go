package auth

import (
	"encoding/json"
	"net/http"
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

// Minimal User model for GORM usage within this package
type User struct {
	ID        string `gorm:"primaryKey;type:text"`
	Name      string `gorm:"not null"`
	Email     string `gorm:"uniqueIndex;not null"`
	Password  string `gorm:"not null"`
	CreatedAt int64  `gorm:"autoCreateTime"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	hashed, _ := HashFunction(input.Password)
	user := User{
		ID:       uuid.NewString(),
		Name:     input.Name,
		Email:    input.Email,
		Password: hashed,
	}

	if err := h.DB.Create(&user).Error; err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, map[string]string{"message": "User registered successfully"})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
	}

	var user User
	if err := h.DB.
		Select("id", "password").
		Where("email = ?", input.Email).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid credentials")
			return
		}
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !CheckPasswordHash(input.Password, user.Password) {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid password")
		return
	}

	accessToken, err := GenerateToken(user.ID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}
	refreshToken, err := GenerateRefreshToken(user.ID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value("id").(string)

	var user struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := h.DB.
		Table("users").
		Select("id", "name", "email").
		Where("id = ?", id).
		Take(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "user not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, user)
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request")
		return
	}

	id, err := ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	accessToken, err := GenerateToken(id)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}
	newRefreshToken, err := GenerateRefreshToken(id)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
	})
}
