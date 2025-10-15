package database

import (
	"database/sql"
	"log"
)

// RunMigrations creates tables if they don't exist
func RunMigrations(db *sql.DB) {
	createUsers := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	createReports := `
	CREATE TABLE IF NOT EXISTS reports (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);`

	createEvidence := `
	CREATE TABLE IF NOT EXISTS evidence (
		id TEXT PRIMARY KEY,
		report_id TEXT NOT NULL,
		file_path TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(report_id) REFERENCES reports(id)
	);`

	createMessages := `
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		report_id TEXT NOT NULL,
		sender_id TEXT NOT NULL,
		message TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(report_id) REFERENCES reports(id),
		FOREIGN KEY(sender_id) REFERENCES users(id)
	);`

	stmts := []string{
		createUsers,
		createReports,
		createEvidence,
		createMessages,
	}

	for _, stmt := range stmts {
		_, err := db.Exec(stmt)
		if err != nil {
			log.Fatalf("migration error: %v\nSQL: %s", err, stmt)
		}
	}

	log.Println("✅ Database migrated successfully")
}
