package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"whistleblower_REST/internal/database"
	"whistleblower_REST/routes"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
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

	// ✅ Tambahkan middleware CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://192.168.150.152:3000",
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	// Jalankan server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf("0.0.0.0:%s", port) 

	log.Printf("🚀 Server running on http://%s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
