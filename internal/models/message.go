package models

import (
	"time"
	
)


type Message struct {
    ID        string     `json:"id" gorm:"primaryKey;type:char(36)"`
    ReportID  uint       `json:"report_id" gorm:"not null;index"`

    // Tidak lagi memiliki foreign key ke users ataupun admins
    SenderID   *string    `json:"sender_id,omitempty" gorm:"type:varchar(50);index"`
    SenderRole string     `json:"sender_role" gorm:"type:varchar(20)"` // "user" atau "admin"

    Message    string     `json:"message" gorm:"not null"`

    FileURL    *string    `json:"file_url,omitempty"`
    FileName   *string    `json:"file_name,omitempty"`
    FileType   *string    `json:"file_type,omitempty"`
    FileSize   *int64     `json:"file_size,omitempty"`

    IsDelivered bool       `json:"is_delivered" gorm:"not null;default:true"`
    IsRead      bool       `json:"is_read" gorm:"not null;default:false"`
    ReadAt      *time.Time `json:"read_at,omitempty"`

    CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`

    // Associations
    Report Report `json:"-" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:ReportID;references:ID"`
}


type CreateMessageRequest struct {
	Message string `json:"message" binding:"required"`
}
