package models

import (
    "time"
)

type PasswordReset struct {
    ID        string    `gorm:"primaryKey"`
    UserID    string    `gorm:"index"` // ✅ Tambahkan untuk change password
    Email     string    `gorm:"index;not null"`
    Code      string    `gorm:"not null"`
    Type      string    `gorm:"not null;default:'forgot_password'"` // ✅ Tambahkan type
    ExpiresAt time.Time `gorm:"not null"`
    Used      bool      `gorm:"default:false"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
}

// Type values: "forgot_password" atau "change_password"


type VerifyCodeRequest struct {
    Email string `json:"email"`
    Code  string `json:"code"`
}

type ResetPasswordRequest struct {
    Token       string `json:"token"`
    NewPassword string `json:"new_password"`
}

type ChangePasswordRequest struct {
    OldPassword string `json:"old_password"` // Wajib verify password lama
    NewPassword string `json:"new_password"`
}

type RequestChangePasswordRequest struct {
    OldPassword string `json:"old_password"`
}

type VerifyChangePasswordCodeRequest struct {
    Code string `json:"code"`
}
