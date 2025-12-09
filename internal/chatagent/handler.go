package chatagent

import (
	"encoding/json"
	"net/http"
	"strings"
	"fmt"

	"whistleblower_REST/internal/chatai"
	"whistleblower_REST/internal/notifications"
	
)

type ChatRequest struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"` // ← Ubah dari uint ke string untuk support UUID
}

type ChatResponse struct {
	Reply       string `json:"reply"`
	NeedHandoff bool   `json:"need_handoff"`
}

func ChatHandler(w http.ResponseWriter, r *http.Request) {
	// Set content type
	w.Header().Set("Content-Type", "application/json")
	
	// LOG: Print semua request details
	fmt.Printf("\n=== INCOMING REQUEST ===\n")
	fmt.Printf("Method: %s\n", r.Method)
	fmt.Printf("URL: %s\n", r.URL.String())
	fmt.Printf("Headers: %+v\n", r.Header)
	fmt.Printf("Remote Addr: %s\n", r.RemoteAddr)

	// Validate request method
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("❌ Error decoding body: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	
	// LOG: Print decoded request
	fmt.Printf("Decoded Request: message='%s', user_id=%s\n", req.Message, req.UserID)
	fmt.Printf("========================\n\n")

	userMsg := strings.TrimSpace(req.Message)
	if userMsg == "" {
		respond(w, "Silakan masukkan pertanyaan Anda.", false, req.UserID, userMsg)
		return
	}

	// ✅ CHECK PERTAMA: Apakah user sedang terhubung dengan admin?
	if IsConnectedToAdmin(req.UserID) {
		fmt.Printf("👤 User %s is connected to admin, forwarding message\n", req.UserID)

		// Update activity timestamp
		UpdateActivity(req.UserID)

		// Kirim ke admin via Pusher
		go func() {
			err := notifications.NotifyAdminNewMessage(req.UserID, userMsg)
			if err != nil {
				fmt.Printf("❌ Failed to notify admin: %v\n", err)
			} else {
				fmt.Printf("✅ Message forwarded to admin from user: %s\n", req.UserID)
			}
		}()

		// Response ke user
		respond(w, "Pesan Anda telah dikirim ke admin.", false, req.UserID, userMsg)
		return
	}

	// === 1. RULE-BASED CHECK (Fast Response - PRIORITAS!) ===
	if ruleRes, ok := MatchRule(userMsg); ok {
		needAdmin := strings.Contains(ruleRes, "<handoff>true</handoff>")
		clean := strings.ReplaceAll(ruleRes, "<handoff>true</handoff>", "")
		
		// LOG untuk debugging
		fmt.Printf("✅ Rule matched for: %s\n", userMsg)

		// Jika handoff, buat session
		if needAdmin {
			CreateSession(req.UserID)
		}

		respond(w, clean, needAdmin, req.UserID, userMsg)
		return // PENTING: return di sini agar tidak lanjut ke AI
	}

	// === 2. AI FALLBACK (Gemini) - HANYA jika rule tidak match ===
	fmt.Printf("⚠️ No rule match, calling AI for: %s\n", userMsg)
	
	aiReply, err := chatai.ChatAgentReply(SystemPrompt, userMsg)
	if err != nil {
		fmt.Printf("❌ AI Error: %v\n", err)
		respond(w, "Maaf, terjadi gangguan sistem. Silakan coba beberapa saat lagi.", false, req.UserID, userMsg)
		return	
	}

	fmt.Printf("✅ AI Reply received\n")

	needAdmin := strings.Contains(aiReply, "<handoff>true</handoff>")
	clean := strings.ReplaceAll(aiReply, "<handoff>true</handoff>", "")

	// Jika handoff, buat session
	if needAdmin {
		CreateSession(req.UserID)
	}

	respond(w, clean, needAdmin, req.UserID, userMsg)
}


func respond(w http.ResponseWriter, reply string, needAdmin bool, userID string, userMsg string) {
	// PENTING: Kirim response dulu SEBELUM notifikasi
	// Jangan sampai error notifikasi menghalangi response ke user
	
	response := ChatResponse{
		Reply:       reply,
		NeedHandoff: needAdmin,
	}
	
	// Set header dan encode response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Printf("❌ Error encoding response: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Response sent: reply=%s, handoff=%v\n", reply[:min(50, len(reply))], needAdmin)
	
	// Notifikasi admin dilakukan SETELAH response terkirim
	if needAdmin {
		go func() {
			// Jalankan di goroutine terpisah agar tidak blocking
			if err := NotifyAdminSafe(userID, userMsg); err != nil {
				fmt.Printf("⚠️ Failed to notify admin: %v\n", err)
			}
		}()
	}
}

func NotifyAdminSafe(userID string, userMessage string) error {
    defer func() {
        if r := recover(); r != nil {
            fmt.Printf("❌ Panic in NotifyAdmin: %v\n", r)
        }
    }()

    fmt.Printf("📨 Sending Chat Agent Handoff for user %s\n", userID)

    // PENTING — panggil Pusher
    return notifications.NotifyChatAgentHandoff(userID, userMessage)
}


// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}