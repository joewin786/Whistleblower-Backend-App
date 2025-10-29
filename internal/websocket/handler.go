package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"whistleblower_REST/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // ⚠️ allow all origins (development only)
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

	// ✅ Get user ID from context (set by AuthMiddleware)
	rawID := r.Context().Value("id")
	userID, ok := rawID.(string)
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	client := &Client{
		ReportID: reportID,
		UserID:   userID,
		Send:     make(chan []byte, 256),
	}
	h.Hub.Register(reportID, client)
	defer h.Hub.Unregister(reportID, client)

	// Writer goroutine
	go func() {
		for msg := range client.Send {
			conn.WriteMessage(websocket.TextMessage, msg)
		}
	}()

	// Reader loop
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Read error:", err)
			break
		}

		var input struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(message, &input); err != nil {
			fmt.Println("Invalid JSON:", err)
			continue
		}

		// ✅ Extract sender from the request context (set by AuthMiddleware)
		senderID, _ := r.Context().Value("id").(string)

		// ✅ Save to database
		msg := models.Message{
			ID:       uuid.NewString(),
			ReportID: reportID,
			SenderID: &senderID,
			Message:  input.Message,
		}

		if err := h.DB.Create(&msg).Error; err != nil {
			fmt.Println("DB save error:", err)
			continue
		}

		// ✅ Broadcast the new message (as JSON) to all clients in same report
		msgJSON, _ := json.Marshal(msg)
		h.Hub.Broadcast(reportID, msgJSON)
	}
}
