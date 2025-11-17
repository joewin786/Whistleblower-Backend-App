package auth

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"regexp"
	"time"
    "bytes"
    "io"
	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"

	"github.com/google/uuid"
)

type ForgotPasswordRequest struct {
    Email string `json:"email"`
}



// Validate email format
func isValidEmail(email string) bool {
    emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    return emailRegex.MatchString(email)
}



func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
    var req ForgotPasswordRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    // Validate email format
    if req.Email == "" {
        utils.RespondWithError(w, http.StatusBadRequest, "Email is required")
        return
    }

    // Cek apakah email terdaftar
    var user models.User
    if err := h.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
        // Security: jangan kasih tau email tidak terdaftar
        utils.RespondWithJSON(w, http.StatusOK, map[string]string{
            "message": "Kode verifikasi telah dikirim ke email kamu",
        })
        return
    }

    // Generate secure OTP
    code, err := generateSecureOTP()
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate code")
        return
    }

    // Delete old reset codes
    h.DB.Where("email = ?", req.Email).Delete(&models.PasswordReset{})

    // Simpan ke database
    reset := models.PasswordReset{
        ID:        uuid.NewString(),
        Email:     req.Email,
        Code:      code,
        Type:      "forgot_password",
        ExpiresAt: time.Now().Add(10 * time.Minute),
        Used:      false,
    }
    
    if err := h.DB.Create(&reset).Error; err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create reset code")
        return
    }

    // ✅ KIRIM EMAIL DI BACKGROUND (tidak blocking response)
    go func() {
        msg := fmt.Sprintf("Kode verifikasi lupa password kamu adalah: %s (berlaku 10 menit)", code)
        if err := utils.SendEmail(req.Email, "Kode Verifikasi Reset Password", msg); err != nil {
            log.Printf("[FORGOT PASSWORD] Failed to send email to %s: %v", req.Email, err)
        }
    }()

    // ✅ LANGSUNG RETURN RESPONSE (tidak tunggu email terkirim)
    utils.RespondWithJSON(w, http.StatusOK, map[string]string{
        "message": "Kode verifikasi telah dikirim ke email kamu",
    })
}

func (h *AuthHandler) VerifyResetCode(w http.ResponseWriter, r *http.Request) {
    var req models.VerifyCodeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
        return
    }

    // Validate email format
    if !isValidEmail(req.Email) {
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid email format")
        return
    }

    // Validate code format (6 digits)
    if len(req.Code) != 6 {
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid code format")
        return
    }

    // Use transaction to prevent race condition
    tx := h.DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    var reset models.PasswordReset
    if err := tx.Where("email = ? AND code = ? AND used = ?", req.Email, req.Code, false).
        First(&reset).Error; err != nil {
        tx.Rollback()
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid or expired code")
        return
    }

    // Check expiration
    if time.Now().After(reset.ExpiresAt) {
        tx.Rollback()
        utils.RespondWithError(w, http.StatusBadRequest, "Code expired")
        return
    }

    // Mark as used
    reset.Used = true
    if err := tx.Save(&reset).Error; err != nil {
        tx.Rollback()
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to verify code")
        return
    }

    if err := tx.Commit().Error; err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }

    // Generate reset token
    token, err := GenerateResetToken(req.Email)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate token")
        return
    }

    utils.RespondWithJSON(w, http.StatusOK, map[string]string{
        "reset_token": token,
    })
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
    log.Println("=== RESET PASSWORD START ===")
    
    // Read raw body first untuk debugging
    bodyBytes, err := io.ReadAll(r.Body)
    if err != nil {
        log.Printf("❌ Failed to read body: %v", err)
        utils.RespondWithError(w, http.StatusBadRequest, "Failed to read request")
        return
    }
    log.Printf("📦 Raw request body: %s", string(bodyBytes))
    
    // Restore body for decoder
    r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
    
    var req models.ResetPasswordRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("❌ Decode error: %v", err)
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
        return
    }
    
    log.Printf("✅ Decoded successfully")
    log.Printf("📧 Token length: %d", len(req.Token))
    log.Printf("📧 Token (first 30): %s...", req.Token[:min(30, len(req.Token))])
    log.Printf("🔑 New password length: %d", len(req.NewPassword))
    
    // Validate basic fields
    if req.Token == "" {
        log.Printf("❌ Token is empty")
        utils.RespondWithError(w, http.StatusBadRequest, "Token is required")
        return
    }
    
    if req.NewPassword == "" {
        log.Printf("❌ New password is empty")
        utils.RespondWithError(w, http.StatusBadRequest, "New password is required")
        return
    }
    
    // Validate password length
    if len(req.NewPassword) < 8 {
        log.Printf("❌ Password too short: %d characters", len(req.NewPassword))
        utils.RespondWithError(w, http.StatusBadRequest, "Password must be at least 8 characters")
        return
    }
    
    // Validate token
    email, role, err := ValidateResetToken(req.Token)
    if err != nil {
        log.Printf("❌ Token validation error: %v", err)
        utils.RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
        return
    }
    
    log.Printf("✅ Token valid - Email: %s, Role: %s", email, role)
    
    // Hash password
    hashed, err := HashFunction(req.NewPassword)
    if err != nil {
        log.Printf("❌ Hash error: %v", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to hash password")
        return
    }
    
    log.Printf("✅ Password hashed successfully")
    
    // Update password
    result := h.DB.Model(&models.User{}).Where("email = ?", email).Update("password", hashed)
    if result.Error != nil {
        log.Printf("❌ Database error: %v", result.Error)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to reset password")
        return
    }
    
    if result.RowsAffected == 0 {
        log.Printf("❌ User not found with email: %s", email)
        utils.RespondWithError(w, http.StatusNotFound, "User not found")
        return
    }
    
    log.Printf("✅ Database updated, rows affected: %d", result.RowsAffected)
    
    // Delete reset codes
    h.DB.Where("email = ? AND type = ?", email, "forgot_password").Delete(&models.PasswordReset{})
    
    log.Println("✅ Password reset successful")
    
    utils.RespondWithJSON(w, http.StatusOK, map[string]string{
        "message": "Password berhasil direset, silakan login kembali.",
    })
}




// ===== STEP 1: Request Change Password dengan Old Password =====


func (h *AuthHandler) RequestChangePassword(w http.ResponseWriter, r *http.Request) {
    // Get user ID dari JWT token (dari middleware)
    userID := r.Context().Value("id").(string)
    
    var req models.RequestChangePasswordRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
        return
    }

    // Validate input
    if req.OldPassword == "" {
        utils.RespondWithError(w, http.StatusBadRequest, "Old password is required")
        return
    }

    // Get user from database
    var user models.User
    if err := h.DB.Where("id = ?", userID).First(&user).Error; err != nil {
        utils.RespondWithError(w, http.StatusNotFound, "User not found")
        return
    }

    // Verify old password
    if !CheckPasswordHash(req.OldPassword, user.Password) {
        utils.RespondWithError(w, http.StatusUnauthorized, "Old password is incorrect")
        return
    }

    // Generate secure OTP
    code, err := generateSecureOTP()
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate code")
        return
    }

    // Delete old change password codes for this user
    h.DB.Where("user_id = ? AND type = ?", userID, "change_password").Delete(&models.PasswordReset{})

    // Save to database dengan type "change_password"
    reset := models.PasswordReset{
        ID:        uuid.NewString(),
        UserID:    userID,
        Email:     user.Email,
        Code:      code,
        Type:      "change_password", // Tandai sebagai change password
        ExpiresAt: time.Now().Add(10 * time.Minute),
        Used:      false,
    }
    
    if err := h.DB.Create(&reset).Error; err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create verification code")
        return
    }

    // Send OTP to email
    msg := fmt.Sprintf("Kode verifikasi untuk mengganti password adalah: %s (berlaku 10 menit)", code)
    if err := utils.SendEmail(user.Email, "Kode Verifikasi Ganti Password", msg); err != nil {
        fmt.Printf("Failed to send email to %s: %v\n", user.Email, err)
    }

    utils.RespondWithJSON(w, http.StatusOK, map[string]string{
        "message": "Kode verifikasi telah dikirim ke email kamu",
    })
}

// ===== STEP 2: Verify OTP Code untuk Change Password =====


func (h *AuthHandler) VerifyChangePasswordCode(w http.ResponseWriter, r *http.Request) {
    // Get user ID dari JWT token
    userID := r.Context().Value("id").(string)
    
    var req models.VerifyChangePasswordCodeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
        return
    }

    // Validate code format
    if len(req.Code) != 6 {
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid code format")
        return
    }

    // Use transaction to prevent race condition
    tx := h.DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    var reset models.PasswordReset
    if err := tx.Where("user_id = ? AND code = ? AND type = ? AND used = ?", 
        userID, req.Code, "change_password", false).First(&reset).Error; err != nil {
        tx.Rollback()
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid or expired code")
        return
    }

    // Check expiration
    if time.Now().After(reset.ExpiresAt) {
        tx.Rollback()
        utils.RespondWithError(w, http.StatusBadRequest, "Code expired")
        return
    }

    // Mark as used
    reset.Used = true
    if err := tx.Save(&reset).Error; err != nil {
        tx.Rollback()
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to verify code")
        return
    }

    if err := tx.Commit().Error; err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }

    // Generate change password token (JWT dengan role "change_password")
   	 token, err := GenerateChangePasswordToken(reset.Email)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate token")
        return
    }

    utils.RespondWithJSON(w, http.StatusOK, map[string]string{
        "change_token": token,
        "message": "Kode berhasil diverifikasi, silakan masukkan password baru",
    })
}

// ===== STEP 3: Change Password dengan New Password =====
type ChangePasswordRequest struct {
    ChangeToken string `json:"change_token"`
    NewPassword string `json:"new_password"`
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
    var req ChangePasswordRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
        return
    }

    // Validate change token
    email, role, err := ValidateChangePasswordToken(req.ChangeToken)
        if err != nil || role != "change_password" {
            utils.RespondWithError(w, http.StatusUnauthorized, "Invalid or expired change token")
            return
        }

    // Validate new password
    if len(req.NewPassword) < 8 {
        utils.RespondWithError(w, http.StatusBadRequest, "Password must be at least 8 characters")
        return
    }

    // Get user untuk check apakah password baru sama dengan lama
    var user models.User
    if err := h.DB.Where("email = ?", email).First(&user).Error; err != nil {
        utils.RespondWithError(w, http.StatusNotFound, "User not found")
        return
    }

    // Check if new password same as old password
    if CheckPasswordHash(req.NewPassword, user.Password) {
        utils.RespondWithError(w, http.StatusBadRequest, "New password must be different from old password")
        return
    }

    // Hash new password
    hashed, err := HashFunction(req.NewPassword)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to hash password")
        return
    }

    // Update password
    if err := h.DB.Model(&user).Update("password", hashed).Error; err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to change password")
        return
    }

    // Delete all change password codes untuk user ini
    h.DB.Where("user_id = ? AND type = ?", email, "change_password").Delete(&models.PasswordReset{})

    // Optional: Send notification email
    msg := "Password kamu berhasil diubah. Jika ini bukan kamu, segera hubungi admin."
    utils.SendEmail(user.Email, "Password Berhasil Diubah", msg)

    utils.RespondWithJSON(w, http.StatusOK, map[string]string{
        "message": "Password berhasil diubah",
    })
}

// ===== Helper Functions =====

// Generate cryptographically secure OTP
func generateSecureOTP() (string, error) {
    max := big.NewInt(1000000)
    n, err := rand.Int(rand.Reader, max)
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("%06d", n.Int64()), nil
}


