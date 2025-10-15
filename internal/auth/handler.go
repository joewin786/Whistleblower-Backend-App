package auth

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	DB *sql.DB
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

func (h *AuthHandler) Register(ctx *gin.Context) {
	var input RegisterRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashed, _ := HashFunction(input.Password)
	userID := uuid.NewString()

	_, err := h.DB.Exec(`
		INSERT INTO users (uid, name, email, password)
		VALUES (?, ?, ?, ?)`,
		userID, input.Name, input.Email, hashed,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"uid": userID, "message": "user registered"})
}

func (h *AuthHandler) Login(ctx *gin.Context) {
	var input LoginRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var hashed string
	var userID string
	err := h.DB.QueryRow(`SELECT uid FROM users WHERE email = ?`, input.Email).Scan(&userID, &hashed)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !CheckPasswordHash(hashed, input.Password) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, _ := GenerateToken(userID)
	ctx.JSON(http.StatusOK, gin.H{"uid": userID, "token": token})
}

func (h *AuthHandler) Me(ctx *gin.Context) {
	uid := ctx.GetString("uid")
	var user struct {
		UID   string `json:"uid"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	err := h.DB.QueryRow("SELECT uid, name, email FROM users WHERE uid = ?", uid).
		Scan(&user.UID, &user.Name, &user.Email)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	ctx.JSON(http.StatusOK, user)
}
