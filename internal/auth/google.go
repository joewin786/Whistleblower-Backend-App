package auth

import (
    "encoding/json"
    "fmt"
    "net/http"
    "whistleblower_REST/internal/utils"
	"whistleblower_REST/internal/models"

    "github.com/google/uuid"
    "google.golang.org/api/idtoken"
    "gorm.io/gorm"
)

type GoogleAuthRequest struct {
    Token string `json:"token"`
}

func (h *AuthHandler) GoogleAuth(w http.ResponseWriter, r *http.Request) {
    var req GoogleAuthRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
        return
    }

    if req.Token == "" {
        utils.RespondWithError(w, http.StatusBadRequest, "Missing Google token")
        return
    }

    // ✅ Verifikasi token dari Google
    payload, err := idtoken.Validate(r.Context(), req.Token, "781729739869-7cil9dr0grb1q4l7kt1q7cshigq7hdv7.apps.googleusercontent.com")
    if err != nil {
        utils.RespondWithError(w, http.StatusUnauthorized, fmt.Sprintf("Invalid Google token: %v", err))
        return
    }

    email := fmt.Sprintf("%v", payload.Claims["email"])
    name := fmt.Sprintf("%v", payload.Claims["name"])

    // ✅ Cek apakah user sudah ada di DB
    var user models.User
    if err := h.DB.Where("email = ?", email).First(&user).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            // Buat akun baru
            user = models.User{
                ID:       uuid.NewString(),
                Name:     name,
                Email:    email,
                Role:     "user",
                Password: "-", // kosong karena login pakai Google
            }
            if err := h.DB.Create(&user).Error; err != nil {
                utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create user")
                return
            }
        } else {
            utils.RespondWithError(w, http.StatusInternalServerError, "Database error")
            return
        }
    }

    // ✅ Generate Access & Refresh Token (fungsi kamu sendiri)
    accessToken, err := GenerateToken(user.ID, user.Role)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate access token")
        return
    }

    refreshToken, err := GenerateRefreshToken(user.ID)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate refresh token")
        return
    }

    utils.RespondWithJSON(w, http.StatusOK, TokenResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
    })
}


