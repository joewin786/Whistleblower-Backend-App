package models

import "time"

// UserDevice stores FCM tokens for push notifications
type UserDevice struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       string    `json:"user_id" gorm:"index"`
	FCMToken     string    `json:"fcm_token" gorm:"uniqueIndex"`
	DeviceType   string    `json:"device_type"` // "android", "ios", "web"
	DeviceName   string    `json:"device_name,omitempty"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	LastActiveAt time.Time `json:"last_active_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}