package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"whistleblower_REST/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // ⚠️ For development only; restrict in production
	},
}

type WSHandler struct {
	DB  *gorm.DB
	Hub *Hub
}

func NewWSHandler(db *gorm.DB, hub *Hub) *WSHandler {
	return &WSHandler{DB: db, Hub: hub}
}

func (h *WSHandler) HandleConnections(w http.ResponseWriter, r *http.Request) {
	reportIDStr := chi.URLParam(r, "reportId")
	reportID64, _ := strconv.ParseUint(reportIDStr, 10, 64)
	reportID := uint(reportID64)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	client := &Client{
		ReportID: reportID,
		Send:     make(chan []byte, 256),
	}
	h.Hub.Register(reportID, client)
	defer h.Hub.Unregister(reportID, client)

	// Start writer goroutine
	go func() {
		for msg := range client.Send {
			conn.WriteMessage(websocket.TextMessage, msg)
		}
	}()

	// Read messages from this client
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Read error:", err)
			break
		}

		// Parse incoming JSON message
		var input struct {
			SenderID string `json:"sender_id"`
			Message  string `json:"message"`
		}
		if err := json.Unmarshal(message, &input); err != nil {
			fmt.Println("Invalid JSON:", err)
			continue
		}

		// Save to database
		msg := models.Message{
			ID:       fmt.Sprintf("%v", reportIDStr+"_"+input.SenderID),
			ReportID: reportID,
			SenderID: &input.SenderID,
			Message:  input.Message,
		}
		h.DB.Create(&msg)

		// Broadcast to all clients in same report
		h.Hub.Broadcast(reportID, message)
	}
}
