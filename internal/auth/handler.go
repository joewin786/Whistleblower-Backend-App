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

type LoginResponse struct {
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
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	var existing User
	if err := h.DB.Where("email = ?", input.Email).First(&existing).Error; err == nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Email already registered")
		return
	}

	hashed, err := HashFunction(input.Password)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

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
		return
	}

	var user User
	if err := h.DB.
		Select("id", "password").
		Where("email = ?", input.Email).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

	utils.RespondWithJSON(w, http.StatusOK, LoginResponse{
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
