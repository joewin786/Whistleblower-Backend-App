package database

import (
	"log"
	"time"

	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) {
	type User struct {
		ID        string    `gorm:"primaryKey;type:char(36)"` // UUID generated in Go
		Name      string    `gorm:"not null"`
		Email     string    `gorm:"uniqueIndex;not null"`
		Password  string    `gorm:"not null"`
		Role      string    `gorm:"default:user"`
		CreatedAt time.Time `gorm:"autoCreateTime"`
		UpdatedAt time.Time `gorm:"autoUpdateTime"`
	}

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

	type Evidence struct {
		ID        string `gorm:"primaryKey;type:char(36)"`
		ReportID  uint   `gorm:"index;not null"`
		FilePath  string `gorm:"not null"`
		FileName  string
		CreatedAt time.Time `gorm:"autoCreateTime"`
		Report    Report    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:ReportID;references:ID"`
	}

	type Message struct {
		ID        string    `gorm:"primaryKey;type:char(36)"`
		ReportID  uint      `gorm:"index;not null"`
		SenderID  *string   `gorm:"type:char(36);index;null"`
		Message   string    `gorm:"not null"`
		CreatedAt time.Time `gorm:"autoCreateTime"`
		Report    Report    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:ReportID;references:ID"`
		User      User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:SenderID;references:ID"`
	}

	if err := db.AutoMigrate(&User{}, &Report{}, &Evidence{}, &Message{}); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}

	log.Println("✅ Database migrated successfully")
}
