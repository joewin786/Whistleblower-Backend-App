package database

import (
	"log"
	"whistleblower_REST/internal/models"

	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) {

	if err := db.AutoMigrate(&models.User{}, &models.Report{}, &models.Evidence{}, &models.Message{}, &models.Category{}); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}

	log.Println("✅ Database migrated successfully")
}
