package models

import (
	"time"
	"whistleblower_REST/internal/auth"
)


type Message struct {
	ID        string     `json:"id"         gorm:"primaryKey;type:char(36)"`
	ReportID  uint       `json:"report_id"  gorm:"not null;index"`                // match reports.Report.ID (uint)
	SenderID  *string    `json:"sender_id,omitempty" gorm:"type:char(36);index"`  // nullable → boleh anonim
	Message   string     `json:"message"    gorm:"not null"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`

	// Associations
	Report Report `json:"-" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:ReportID;references:ID"`
	User   auth.User      `json:"-" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:SenderID;references:ID"`
}

type CreateMessageRequest struct {
	Message string `json:"message" binding:"required"`
}
