package models

import "time"

type Message struct {
	ID        string    `gorm:"primaryKey;type:char(36)"`
	ReportID  uint      `gorm:"index;not null"`
	SenderID  *string   `gorm:"type:char(36);index;null"`
	Message   string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	Report    Report    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:ReportID;references:ID"`
	User      User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:SenderID;references:ID"`
}
