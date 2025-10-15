package database

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// RunMigrations uses GORM AutoMigrate to create/update tables.
func RunMigrations(db *gorm.DB) {
	type User struct {
		ID        string    `gorm:"primaryKey;type:text"`
		Name      string    `gorm:"not null"`
		Email     string    `gorm:"uniqueIndex;not null"`
		Password  string    `gorm:"not null"`
		CreatedAt time.Time `gorm:"autoCreateTime"`
	}

	type Report struct {
		ID          string `gorm:"primaryKey;type:text"`
		UserID      string `gorm:"not null;index"`
		Title       string `gorm:"not null"`
		Description string
		Status      string    `gorm:"default:pending;index"`
		CreatedAt   time.Time `gorm:"autoCreateTime"`
		// Foreign keys
		User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;foreignKey:UserID;references:ID"`
	}

	type Evidence struct {
		ID        string    `gorm:"primaryKey;type:text"`
		ReportID  string    `gorm:"not null;index"`
		FilePath  string    `gorm:"not null"`
		CreatedAt time.Time `gorm:"autoCreateTime"`
		// Foreign keys
		Report Report `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:ReportID;references:ID"`
	}

	type Message struct {
		ID        string    `gorm:"primaryKey;type:text"`
		ReportID  string    `gorm:"not null;index"`
		SenderID  string    `gorm:"not null;index"`
		Message   string    `gorm:"not null"`
		CreatedAt time.Time `gorm:"autoCreateTime"`
		// Foreign keys
		Report Report `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:ReportID;references:ID"`
		User   User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;foreignKey:SenderID;references:ID"`
	}

	if err := db.AutoMigrate(&User{}, &Report{}, &Evidence{}, &Message{}); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	log.Println("✅ Database migrated successfully")
}
