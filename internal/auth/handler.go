package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"whistleblower_REST/internal/utils"

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

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
	}

	hashed, _ := HashFunction(input.Password)
	userID := uuid.NewString()

	_, err := h.DB.Exec(`
		INSERT INTO users (id, name, email, password)
		VALUES (?, ?, ?, ?)`,
		userID, input.Name, input.Email, hashed,
	)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "User registered successfully"})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
	}

	var hashed string
	var userID string
	err := h.DB.QueryRow(`SELECT id, password FROM users WHERE email = ?`, input.Email).Scan(&userID, &hashed)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
	}

	if !CheckPasswordHash(input.Password, hashed) {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid password")
		return
	}

	token, _ := GenerateToken(userID)
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value("id").(string)

	var user struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	err := h.DB.QueryRow(`SELECT id, name, email FROM users WHERE id = ?`, id).
		Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, user)
}
