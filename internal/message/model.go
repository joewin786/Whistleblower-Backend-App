package messages

import (
	"time"
	"whistleblower_REST/internal/reports"
)


type Message struct {
	ID        string    `json:"id" gorm:"primaryKey;type:text"`
	ReportID  string    `json:"report_id" gorm:"not null;index"`
	SenderID  string    `json:"sender_id" gorm:"not null;index"`
	Message   string    `json:"message" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`

	Report reports.Report `json:"-" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:ReportID;references:ID"`
}

type CreateMessageRequest struct {
	Message string `json:"message" binding:"required"`
}
