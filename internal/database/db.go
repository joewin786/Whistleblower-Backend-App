package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB() (*sql.DB, error) {
	dbDriver := os.Getenv("DB_DRIVER") // "postgres" or "sqlite3"
	dbSource := os.Getenv("DB_SOURCE")

	if dbDriver == "" {
		dbDriver = "sqlite3"
		dbSource = "./whistleblower.db"
		log.Println("ℹ️  Defaulting to SQLite database")
	}

	db, err := sql.Open(dbDriver, dbSource)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %v", err)
	}

	log.Println("✅ Database connected successfully")
	return db, nil
}
