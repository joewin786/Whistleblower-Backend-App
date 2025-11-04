package models

import "time"

type Notification struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
    ReportID  *uint     `json:"report_id,omitempty"` 
    IsRead    bool      `json:"is_read" gorm:"default:false"`

}

type UserNotification struct {
    ID        uint      `json:"id" gorm:"primaryKey"`
    UserID    string    `json:"user_id"`
    Title     string    `json:"title"`
    Message   string    `json:"message"`
    Type      string    `json:"type"`
    ReportID  *uint     `json:"report_id,omitempty"`
    IsRead    bool      `json:"is_read" gorm:"default:false"`
    CreatedAt time.Time `json:"created_at"`
}
