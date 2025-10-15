package database

import (
	"fmt"
	"log"
	"os"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
)

// InitDB initializes and returns a GORM *gorm.DB.
// It supports Postgres when DB_DRIVER=postgres; otherwise defaults to SQLite.
func InitDB() (*gorm.DB, error) {
	dbDriver := os.Getenv("DB_DRIVER") // "postgres" or "sqlite"
	dbSource := os.Getenv("DB_SOURCE")

	var (
		db  *gorm.DB
		err error
	)

	switch dbDriver {
	case "postgres":
		if dbSource == "" {
			return nil, fmt.Errorf("DB_SOURCE is required for postgres")
		}
		db, err = gorm.Open(postgres.Open(dbSource), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
	default:
		// Default to SQLite
		if dbSource == "" {
			dbSource = "./whistleblower.db"
			log.Println("ℹ️  Defaulting to SQLite database")
		}
		db, err = gorm.Open(sqlite.Open(dbSource), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain sql.DB from gorm: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database connection failed: %v", err)
	}

	log.Println("✅ Database connected successfully")
	return db, nil
}
