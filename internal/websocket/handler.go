package websocket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
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

		senderID, _ := r.Context().Value("id").(string)

		// Save to database
		msg := models.Message{
			ID:       uuid.NewString(),
			ReportID: reportID,
			SenderID: &senderID,
			Message:  input.Message,
			IsRead:   false,
		}

		if err := h.DB.Create(&msg).Error; err != nil {
			fmt.Println("DB save error:", err)
			continue
		}

		// Broadcast the new message
		msgJSON, _ := json.Marshal(msg)
		h.Hub.Broadcast(reportID, msgJSON)
	}
}

// 🆕 FUNGSI BARU: Upload file via HTTP (bukan WebSocket)
func (h *WSHandler) UploadMessageFile(w http.ResponseWriter, r *http.Request) {
	reportIDStr := chi.URLParam(r, "reportId")
	reportID64, _ := strconv.ParseUint(reportIDStr, 10, 64)
	reportID := uint(reportID64)

	// Get user ID from context
	rawID := r.Context().Value("id")
	userID, ok := rawID.(string)
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "file too large", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "no file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	message := r.FormValue("message") // Optional text message

	// Create upload directory if not exists
	uploadDir := "./uploads/messages"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		http.Error(w, "failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// Generate unique filename
	ext := filepath.Ext(handler.Filename)
	filename := fmt.Sprintf("%s_%d%s", uuid.NewString(), time.Now().Unix(), ext)
	filePath := filepath.Join(uploadDir, filename)

	// Save file to disk
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	// File URL (adjust sesuai domain Anda)
	fileURL := fmt.Sprintf("/uploads/messages/%s", filename)
	fileSize := handler.Size
	contentType := handler.Header.Get("Content-Type")

	// Save message to database
	msg := models.Message{
		ID:       uuid.NewString(),
		ReportID: reportID,
		SenderID: &userID,
		Message:  message,
		FileURL:  &fileURL,
		FileName: &handler.Filename,
		FileType: &contentType,
		FileSize: &fileSize,
		IsRead:   false,
	}

	if err := h.DB.Create(&msg).Error; err != nil {
		http.Error(w, "failed to save message", http.StatusInternalServerError)
		return
	}

	// Broadcast via WebSocket
	msgJSON, _ := json.Marshal(msg)
	h.Hub.Broadcast(reportID, msgJSON)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}