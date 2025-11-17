package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"whistleblower_REST/internal/database"
	"whistleblower_REST/internal/notifications"
	"whistleblower_REST/internal/websocket"
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

	// ✅ Initialize FCM (Firebase Cloud Messaging)
	if err := notifications.InitializeFCM(); err != nil {
		log.Printf("⚠️  Warning: FCM initialization failed: %v\n", err)
		log.Println("Push notifications will not be available")
	} else {
		log.Println("✅ FCM (Firebase Cloud Messaging) initialized successfully")
	}

	hub := websocket.NewHub()
	log.Println("✅ WebSocket Hub initialized")


	// Setup router with GORM DB
	r := routes.RegisterRoutes(db, hub)

	// ✅ Tambahkan middleware CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://192.168.150.84:3000",
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
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
