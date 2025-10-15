package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"whistleblower_REST/internal/database"
	"whistleblower_REST/routes"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️  No .env file found, using system environment variables")
	}

	// Initialize the database (GORM)
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}

	// Run auto migration
	database.RunMigrations(db)

	// Setup router with GORM DB
	r := routes.RegisterRoutes(db)

	// Run the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)

	log.Printf("🚀 Server running on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
