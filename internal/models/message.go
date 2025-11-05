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


	// File Upload fields
	FileURL   *string    `json:"file_url,omitempty"   gorm:"type:varchar(500)"`
	FileName  *string    `json:"file_name,omitempty"  gorm:"type:varchar(255)"`
	FileType  *string    `json:"file_type,omitempty"  gorm:"type:varchar(100)"` // image/png, application/pdf, etc.
	FileSize  *int64     `json:"file_size,omitempty"`

	IsDelivered bool       `json:"is_delivered" gorm:"not null;default:true"`  // ✓ (centang 1)
	IsRead      bool       `json:"is_read"      gorm:"not null;default:false"` // ✓✓ (centang 2)
	ReadAt      *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	

	// Associations
	Report Report `json:"-" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:ReportID;references:ID"`
	User   auth.User      `json:"-" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:SenderID;references:ID"`
}

type CreateMessageRequest struct {
	Message string `json:"message" binding:"required"`
}
