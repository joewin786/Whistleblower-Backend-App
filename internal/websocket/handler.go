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
	"whistleblower_REST/internal/auth"
	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/notifications" // ✅ TAMBAHKAN IMPORT INI

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // ⚠ allow all origins (development only)
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

	userID, ok := auth.GetIDFromContext(r.Context())
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("❌ Upgrade error:", err)
		return
	}
	defer conn.Close()

	client := &Client{
		ReportID: reportID,
		UserID:   userID,
		Send:     make(chan []byte, 256),
		Conn:     conn,
	}
	h.Hub.Register(reportID, client)
	defer h.Hub.Unregister(reportID, client)

	fmt.Printf("✅ WS connected: userID=%s reportID=%d\n", userID, reportID)

	// Auto-mark messages as read when client connects
	go func() {
		time.Sleep(500 * time.Millisecond)

		now := time.Now()
		result := h.DB.Model(&models.Message{}).
			Where("report_id = ? AND sender_id != ? AND is_read = ?", reportID, userID, false).
			Updates(map[string]interface{}{
				"is_read": true,
				"read_at": now,
			})

		if result.Error == nil && result.RowsAffected > 0 {
			fmt.Printf("✅ Auto-marked %d messages as read on connect (user: %s)\n", result.RowsAffected, userID)

			payload := map[string]any{
				"type":      "messages_read_all",
				"report_id": reportID,
				"reader_id": userID,
				"read_at":   now.Format(time.RFC3339),
				"count":     result.RowsAffected,
			}
			data, _ := json.Marshal(payload)
			h.Hub.Broadcast(reportID, data)
			fmt.Printf("📤 [AUTO-READ] Broadcasted to ALL clients (reader: %s)\n", userID)
		}
	}()

	// Writer goroutine
	go func() {
		for msg := range client.Send {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				fmt.Printf("❌ Write error for user %s: %v\n", userID, err)
				break
			}
		}
	}()

	// Single Reader loop
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("🔌 Read error for user %s: %v\n", userID, err)
			break
		}

		var input map[string]interface{}
		if err := json.Unmarshal(message, &input); err != nil {
			fmt.Println("❌ Invalid JSON:", err)
			continue
		}

		eventType, _ := input["type"].(string)
		text, _ := input["message"].(string)

		fmt.Printf("📩 Received from %s: type=%s\n", userID, eventType)

		// 1️⃣ Abaikan event kosong
		if eventType == "" && text == "" {
			fmt.Println("⚠️ Ignored empty message")
			continue
		}

		// 2️⃣ Event typing
		if eventType == "typing" {
			payload := map[string]any{
				"type":      "typing",
				"user_id":   userID,
				"report_id": reportID,
				"timestamp": time.Now().Unix(),
			}
			data, _ := json.Marshal(payload)
			h.Hub.Broadcast(reportID, data)
			continue
		}

		// 3️⃣ Event read_all
		if eventType == "read_all" {
			now := time.Now()

			result := h.DB.Model(&models.Message{}).
				Where("report_id = ? AND sender_id != ? AND is_read = ?", reportID, userID, false).
				Updates(map[string]interface{}{
					"is_read": true,
					"read_at": now,
				})

			if result.Error != nil {
				fmt.Printf("❌ DB read_all update error: %v\n", result.Error)
				continue
			}

			fmt.Printf("✅ [READ_ALL] Marked %d messages as read by %s (reportID=%d)\n", result.RowsAffected, userID, reportID)

			payload := map[string]any{
				"type":      "messages_read_all",
				"report_id": reportID,
				"reader_id": userID,
				"read_at":   now.Format(time.RFC3339),
				"count":     result.RowsAffected,
			}
			data, _ := json.Marshal(payload)

			h.Hub.Broadcast(reportID, data)
			fmt.Printf("📤 [READ_ALL] Broadcasted to ALL clients (reader: %s)\n", userID)
			continue
		}

		// 4️⃣ Event read_message
		if eventType == "read_message" {
			messageID, _ := input["message_id"].(string)
			if messageID == "" {
				fmt.Println("⚠️ Missing message_id for read_message event")
				continue
			}

			now := time.Now()
			result := h.DB.Model(&models.Message{}).
				Where("id = ? AND report_id = ? AND sender_id != ? AND is_read = ?", messageID, reportID, userID, false).
				Updates(map[string]interface{}{
					"is_read": true,
					"read_at": now,
				})

			if result.Error != nil {
				fmt.Printf("❌ DB read_message update error: %v\n", result.Error)
				continue
			}

			if result.RowsAffected > 0 {
				fmt.Printf("✅ Message %s marked as read by %s\n", messageID, userID)

				payload := map[string]any{
					"type":       "read_status",
					"message_id": messageID,
					"report_id":  reportID,
					"reader_id":  userID,
					"is_read":    true,
					"read_at":    now.Format(time.RFC3339),
				}
				data, _ := json.Marshal(payload)
				h.Hub.BroadcastExcept(reportID, client, data)
				fmt.Printf("📤 Broadcasted message_read to other clients (message: %s, reader: %s)\n", messageID, userID)
			}
			continue
		}

		// 5️⃣ Pesan teks kosong → skip
		if text == "" {
			fmt.Println("⚠️ Empty text message ignored")
			continue
		}

		// 6️⃣ Simpan pesan valid ke database
		msg := models.Message{
			ID:         uuid.NewString(),
			ReportID:   reportID,
			SenderID:   &userID,
			SenderRole: "user",
			Message:    text,
			IsRead:     false,
			CreatedAt:  time.Now(),
		}

		if err := h.DB.Create(&msg).Error; err != nil {
			fmt.Printf("❌ DB save error: %v\n", err)
			continue
		}

		fmt.Printf("✅ [NEW MSG] Saved: id=%s, sender=%s\n", msg.ID, userID)

		// 7️⃣ Broadcast pesan baru ke semua client via WebSocket
		msgJSON, _ := json.Marshal(msg)
		h.Hub.Broadcast(reportID, msgJSON)

		// 8️⃣ Auto-mark pesan sebagai delivered
		go func() {
			time.Sleep(100 * time.Millisecond)

			h.DB.Model(&models.Message{}).
				Where("id = ?", msg.ID).
				Update("is_delivered", true)

			deliveryPayload := map[string]any{
				"type":         "message_delivered",
				"message_id":   msg.ID,
				"report_id":    reportID,
				"is_delivered": true,
			}
			deliveryData, _ := json.Marshal(deliveryPayload)
			h.Hub.Broadcast(reportID, deliveryData)
		}()

		// ✅ 9️⃣ TAMBAHAN: Kirim Push Notification & Pusher
		go h.sendMessageNotifications(reportID, userID, text)
	}

	fmt.Printf("🔌 WS disconnected: userID=%s reportID=%d\n", userID, reportID)
}

// =========================================
//
//	ADMIN WEBSOCKET HANDLER
//
// =========================================
func (h *WSHandler) HandleAdminConnections(w http.ResponseWriter, r *http.Request) {
	reportIDStr := chi.URLParam(r, "reportId")
	reportID64, _ := strconv.ParseUint(reportIDStr, 10, 64)
	reportID := uint(reportID64)

	// ✅ FIX: Ambil admin_id sebagai uint dulu
	var adminIDStr string

	// Try get admin_id (uint) from context
	if adminID, ok := auth.GetAdminIDFromContext(r.Context()); ok {
		adminIDStr = fmt.Sprintf("%d", adminID)
	}

	// Fallback: try get id (string) from context
	if adminIDStr == "" {
		if id, ok := auth.GetIDFromContext(r.Context()); ok {
			adminIDStr = id
		}
	}

	if adminIDStr == "" {
		http.Error(w, "unauthorized admin ws", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("❌ WS Admin Upgrade error:", err)
		return
	}
	defer conn.Close()

	client := &Client{
		ReportID: reportID,
		UserID:   adminIDStr,
		Send:     make(chan []byte, 256),
		Conn:     conn,
	}

	h.Hub.Register(reportID, client)
	defer h.Hub.Unregister(reportID, client)

	fmt.Printf("👨‍💼 WS ADMIN connected: adminID=%s reportID=%d\n", adminIDStr, reportID)

	// Auto-mark messages as read
	go func() {
		time.Sleep(300 * time.Millisecond)

		now := time.Now()
		result := h.DB.Model(&models.Message{}).
			Where("report_id = ? AND sender_id != ? AND is_read = ?", reportID, adminIDStr, false).
			Updates(map[string]interface{}{
				"is_read": true,
				"read_at": now,
			})

		if result.Error == nil && result.RowsAffected > 0 {
			payload := map[string]any{
				"type":      "messages_read_all",
				"report_id": reportID,
				"reader_id": adminIDStr,
				"read_at":   now.Format(time.RFC3339),
				"count":     result.RowsAffected,
			}
			data, _ := json.Marshal(payload)
			h.Hub.Broadcast(reportID, data)

			fmt.Printf("📤 ADMIN auto-read broadcasted (%d messages)\n", result.RowsAffected)
		}
	}()

	// Writer goroutine
	go func() {
		for msg := range client.Send {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				fmt.Printf("❌ Admin WS Write error: %v\n", err)
				break
			}
		}
	}()

	// Reader loop - REUSE logic dari HandleConnections
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("🔌 WS admin read error: %v\n", err)
			break
		}

		var input map[string]interface{}
		if err := json.Unmarshal(message, &input); err != nil {
			fmt.Println("❌ Invalid JSON:", err)
			continue
		}

		eventType, _ := input["type"].(string)
		text, _ := input["message"].(string)

		fmt.Printf("📩 ADMIN Received: type=%s from=%s\n", eventType, adminIDStr)

		// Handle empty events
		if eventType == "" && text == "" {
			continue
		}

		// Handle typing event
		if eventType == "typing" {
			payload := map[string]any{
				"type":      "typing",
				"user_id":   adminIDStr,
				"report_id": reportID,
				"timestamp": time.Now().Unix(),
			}
			data, _ := json.Marshal(payload)
			h.Hub.Broadcast(reportID, data)
			continue
		}

		// Handle read_all event
		if eventType == "read_all" {
			now := time.Now()
			result := h.DB.Model(&models.Message{}).
				Where("report_id = ? AND sender_id != ? AND is_read = ?", reportID, adminIDStr, false).
				Updates(map[string]interface{}{
					"is_read": true,
					"read_at": now,
				})

			if result.Error != nil {
				fmt.Printf("❌ DB read_all error: %v\n", result.Error)
				continue
			}

			fmt.Printf("✅ [ADMIN READ_ALL] Marked %d messages as read\n", result.RowsAffected)

			payload := map[string]any{
				"type":      "messages_read_all",
				"report_id": reportID,
				"reader_id": adminIDStr,
				"read_at":   now.Format(time.RFC3339),
				"count":     result.RowsAffected,
			}
			data, _ := json.Marshal(payload)
			h.Hub.Broadcast(reportID, data)
			continue
		}

		// Handle read_message event
		if eventType == "read_message" {
			messageID, _ := input["message_id"].(string)
			if messageID == "" {
				continue
			}

			now := time.Now()
			result := h.DB.Model(&models.Message{}).
				Where("id = ? AND report_id = ? AND sender_id != ? AND is_read = ?", messageID, reportID, adminIDStr, false).
				Updates(map[string]interface{}{
					"is_read": true,
					"read_at": now,
				})

			if result.Error == nil && result.RowsAffected > 0 {
				payload := map[string]any{
					"type":       "read_status",
					"message_id": messageID,
					"report_id":  reportID,
					"reader_id":  adminIDStr,
					"is_read":    true,
					"read_at":    now.Format(time.RFC3339),
				}
				data, _ := json.Marshal(payload)
				h.Hub.BroadcastExcept(reportID, client, data)
			}
			continue
		}

		// Handle new text message
		if text == "" {
			continue
		}

		// Save message to database
		msg := models.Message{
			ID:         uuid.NewString(),
			ReportID:   reportID,
			SenderID:   &adminIDStr,
			SenderRole: "admin",
			Message:    text,
			IsRead:     false,
			CreatedAt:  time.Now(),
		}

		if err := h.DB.Create(&msg).Error; err != nil {
			fmt.Printf("❌ DB save error: %v\n", err)
			continue
		}

		fmt.Printf("✅ [ADMIN MSG] Saved: id=%s, sender=%s\n", msg.ID, adminIDStr)

		// Broadcast to all clients
		msgJSON, _ := json.Marshal(msg)
		h.Hub.Broadcast(reportID, msgJSON)

		// Auto-mark as delivered
		go func() {
			time.Sleep(100 * time.Millisecond)

			h.DB.Model(&models.Message{}).
				Where("id = ?", msg.ID).
				Update("is_delivered", true)

			deliveryPayload := map[string]any{
				"type":         "message_delivered",
				"message_id":   msg.ID,
				"report_id":    reportID,
				"is_delivered": true,
			}
			deliveryData, _ := json.Marshal(deliveryPayload)
			h.Hub.Broadcast(reportID, deliveryData)
		}()

		// Send notifications
		go h.sendMessageNotifications(reportID, adminIDStr, text)
	}

	fmt.Printf("🔌 WS ADMIN disconnected: %s\n", adminIDStr)
}

// ✅ FUNGSI BARU: Kirim notifikasi untuk pesan baru
func (h *WSHandler) sendMessageNotifications(reportID uint, senderID string, message string) {
	// Ambil role dari kolom SenderRole langsung dari message terakhir
	var msg models.Message
	err := h.DB.Where("report_id = ? AND sender_id = ?", reportID, senderID).
		Order("created_at DESC").First(&msg).Error

	senderRole := "user"
	if err == nil {
		senderRole = msg.SenderRole
	}

	isAdmin := (senderRole == "admin")

	if err := notifications.NotifyNewChatMessage(h.DB, reportID, senderID, message, isAdmin); err != nil {
		fmt.Printf("[WS NOTIFY ERROR] ❌ Failed to send notification: %v\n", err)
	} else {
		fmt.Printf("[WS NOTIFY] ✅ Notification sent for message from %s (role=%s, report #%d)\n",
			senderID, senderRole, reportID)
	}
}

// Upload file via HTTP
func (h *WSHandler) UploadMessageFile(w http.ResponseWriter, r *http.Request) {
	reportIDStr := chi.URLParam(r, "reportId")
	reportID64, _ := strconv.ParseUint(reportIDStr, 10, 64)
	reportID := uint(reportID64)

	userID, ok := auth.GetIDFromContext(r.Context())
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

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

	message := r.FormValue("message")

	uploadDir := "./uploads/messages"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		http.Error(w, "failed to create upload directory", http.StatusInternalServerError)
		return
	}

	ext := filepath.Ext(handler.Filename)
	filename := fmt.Sprintf("%s_%d%s", uuid.NewString(), time.Now().Unix(), ext)
	filePath := filepath.Join(uploadDir, filename)

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

	fileURL := fmt.Sprintf("/uploads/messages/%s", filename)
	fileSize := handler.Size
	contentType := handler.Header.Get("Content-Type")

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

	// Auto-mark as delivered
	go func() {
		time.Sleep(100 * time.Millisecond)
		h.DB.Model(&models.Message{}).
			Where("id = ?", msg.ID).
			Update("is_delivered", true)

		deliveryPayload := map[string]any{
			"type":         "message_delivered",
			"message_id":   msg.ID,
			"report_id":    reportID,
			"is_delivered": true,
		}
		deliveryData, _ := json.Marshal(deliveryPayload)
		h.Hub.Broadcast(reportID, deliveryData)
	}()

	// ✅ TAMBAHAN: Kirim notifikasi untuk file upload
	go h.sendFileNotifications(reportID, userID, message, handler.Filename)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}

// ✅ FUNGSI BARU: Kirim notifikasi untuk file upload
func (h *WSHandler) sendFileNotifications(reportID uint, senderID, message, filename string) {
	// Check if sender is admin
	var user models.User
	isAdmin := false
	if err := h.DB.First(&user, "id = ?", senderID).Error; err == nil {
		isAdmin = (user.Role == "admin")
	}

	// Format message with file info
	notificationMessage := message
	if notificationMessage == "" {
		notificationMessage = fmt.Sprintf("Mengirim file: %s", filename)
	} else {
		notificationMessage = fmt.Sprintf("%s (File: %s)", message, filename)
	}

	// Send notification
	if err := notifications.NotifyNewChatMessage(h.DB, reportID, senderID, notificationMessage, isAdmin); err != nil {
		fmt.Printf("[FILE NOTIFY ERROR] ❌ Failed to send notification: %v\n", err)
	} else {
		fmt.Printf("[FILE NOTIFY] ✅ Notification sent for file upload from %s (report #%d)\n", senderID, reportID)
	}
}
