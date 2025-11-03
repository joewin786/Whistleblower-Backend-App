package models

import "time"

type Notification struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

type UserNotification struct {
    ID        uint      `json:"id" gorm:"primaryKey"`
    UserID    string    `json:"user_id"`
    Title     string    `json:"title"`
    Message   string    `json:"message"`
    Type      string    `json:"type"`
    Read      bool      `json:"read" gorm:"default:false"`
    CreatedAt time.Time `json:"created_at"`
}
