package main

import (
	"fmt"
	"log"
	"os"
	"whistleblower_REST/internal/database"
	"whistleblower_REST/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️  No .env file found, using system environment variables")
	}

	// Initialize the database
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}
	defer db.Close()

	// Run auto migration
	database.RunMigrations(db)

	// Setup Gin router
	r := gin.Default()
	routes.RegisterRoutes(r, db)

	// Run the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)

	log.Printf("🚀 Server running on http://localhost%s", addr)
	r.Run(addr)

}
