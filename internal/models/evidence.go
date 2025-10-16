package models

import "time"

type Evidence struct {
	ID        string `gorm:"primaryKey;type:char(36)"`
	ReportID  uint   `gorm:"index;not null"`
	FilePath  string `gorm:"not null"`
	FileName  string
	CreatedAt time.Time `gorm:"autoCreateTime"`
	Report    Report    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:ReportID;references:ID"`
}
