package chatagent

import (
	"encoding/json"
	"fmt"
	"net/http"

	"whistleblower_REST/internal/notifications"
	"whistleblower_REST/internal/utils"
)

// ============ STRUCT REQUEST ============

type AdminReplyRequest struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

// ============ ADMIN REPLY HANDLER ============
//
// Route: POST /admin/chat-agent/reply
//
// =======================================

func AdminReplyHandler(w http.ResponseWriter, r *http.Request) {
	var req AdminReplyRequest

	// Parse body JSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	// Validation
	if req.UserID == "" || req.Message == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "user_id & message are required")
		return
	}

	UpdateActivity(req.UserID)

	// ============ SEND TO USER (via Pusher) ============

	channel := fmt.Sprintf("user-%s", req.UserID)

	err := notifications.Client.Trigger(
		channel,
		"chatagent-admin-reply",
		map[string]any{
			"user_id": req.UserID,
			"message": req.Message,
			"from":    "admin",
		},
	)

	if err != nil {
		fmt.Println("❌ Failed to trigger Pusher to user:", err)
		utils.RespondWithError(w, 500, "Failed to send message to user")
		return
	}

	fmt.Println("📨 Admin message sent to:", channel)

	// Response to admin web
	utils.RespondWithJSON(w, 200, map[string]any{
		"status":  "ok",
		"user_id": req.UserID,
		"message": req.Message,
	})
}

func AdminEndSessionHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// ✅ End session di backend
	EndSession(req.UserID)

	// ✅ Notify user bahwa session berakhir
	channel := fmt.Sprintf("user-%s", req.UserID)
	err := notifications.Client.Trigger(
		channel,
		"chatagent-session-ended",  // ✅ Event name yang akan di-listen Flutter
		map[string]any{
			"message": "Sesi chat dengan admin telah berakhir. Terima kasih.",
			"ended_by": "admin",
		},
	)

	if err != nil {
		fmt.Printf("❌ Failed to notify user about session end: %v\n", err)
		utils.RespondWithError(w, 500, "Failed to notify user")
		return
	}

	fmt.Printf("✅ Session ended for user: %s\n", req.UserID)

	utils.RespondWithJSON(w, 200, map[string]any{
		"status":  "ok",
		"message": "Session ended successfully",
		"user_id": req.UserID,
	})
}