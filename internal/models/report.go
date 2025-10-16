package models

import "time"

type Report struct {
	ID           uint    `gorm:"primaryKey;autoIncrement"`
	UserID       *string `gorm:"type:char(36);index;null"`
	Title        string  `gorm:"not null"`
	Description  string
	Category     string
	Status       string    `gorm:"default:submitted;index"`
	ReporterType string    `gorm:"not null;default:anonymous"`
	Email        *string   `gorm:"type:text"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
	User         User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:UserID;references:ID"`
}
