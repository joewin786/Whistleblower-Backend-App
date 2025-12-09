package chatagent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"whistleblower_REST/internal/notifications"
	"whistleblower_REST/internal/utils"
)

// ============ USER MESSAGE HANDLER ============
// Route: POST /chat-agent/user-message
// 
// KHUSUS untuk pesan yang sudah di-handoff ke admin
// Handler ini TIDAK menggunakan AI/Rule-based
// Hanya meneruskan pesan user ke admin via notifikasi
// =======================================

func UserMessageHandler(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("❌ Error decoding body: %v\n", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if req.UserID == "" || req.Message == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "user_id & message are required")
		return
	}

	userMsg := strings.TrimSpace(req.Message)

	fmt.Printf("\n=== USER MESSAGE TO ADMIN ===\n")
	fmt.Printf("User ID: %s\n", req.UserID)
	fmt.Printf("Message: %s\n", userMsg)
	fmt.Printf("===========================\n\n")

	// Update activity
	UpdateActivity(req.UserID)

	// ✅ KIRIM KE ADMIN VIA PUSHER dengan event yang benar
	go func() {
		err := notifications.Client.Trigger(
			"admin-notifications",
			"admin-new-message",  // ← Pastikan event name sama dengan frontend
			map[string]any{
				"user_id": req.UserID,
				"message": userMsg,
			},
		)
		if err != nil {
			fmt.Printf("⚠️ Failed to notify admin: %v\n", err)
		} else {
			fmt.Printf("✅ Message forwarded to admin from user: %s\n", req.UserID)
		}
	}()

	response := ChatResponse{
		Reply:       "Pesan Anda telah dikirim ke admin. Mohon tunggu sebentar, admin akan segera membalas.",
		NeedHandoff: false,
	}

	utils.RespondWithJSON(w, 200, response)
}