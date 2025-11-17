package chatagent

import (
	"encoding/json"
	"net/http"
	"strings"
	"fmt"

	"whistleblower_REST/internal/ai"
	"whistleblower_REST/internal/notifications"
)

type ChatRequest struct {
	Message string `json:"message"`
	UserID  uint   `json:"user_id"`
}

type ChatResponse struct {
	Reply       string `json:"reply"`
	NeedHandoff bool   `json:"need_handoff"`
}

func ChatHandler(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	json.NewDecoder(r.Body).Decode(&req)

	userMsg := strings.TrimSpace(req.Message)

	// === 1. RULE-BASED CHECK ===
	if ruleRes, ok := MatchRule(userMsg); ok {
		needAdmin := strings.Contains(ruleRes, "<handoff>true</handoff>")
		clean := strings.ReplaceAll(ruleRes, "<handoff>true</handoff>", "")

		respond(w, clean, needAdmin, req.UserID, userMsg)
		return
	}

	// === 2. AI FALLBACK (GEMINI) ===
	aiService := ai.NewGeminiService()

	aiReply, err := aiService.ChatAgentReply(SystemPrompt, userMsg)
	if err != nil {
		respond(w, "Maaf, terjadi gangguan sistem. Silakan coba beberapa saat lagi.", false, req.UserID, userMsg)
		return
	}

	needAdmin := strings.Contains(aiReply, "<handoff>true</handoff>")
	clean := strings.ReplaceAll(aiReply, "<handoff>true</handoff>", "")

	respond(w, clean, needAdmin, req.UserID, userMsg)
}

func respond(w http.ResponseWriter, reply string, needAdmin bool, userID uint, userMsg string) {
	if needAdmin {
		NotifyAdmin(userID, userMsg)
	}

	json.NewEncoder(w).Encode(ChatResponse{
		Reply:       reply,
		NeedHandoff: needAdmin,
	})
}

func NotifyAdmin(userID uint, userMessage string) {
	notifications.NotifyChatAgentHandoff(fmt.Sprintf("%d", userID), userMessage)
}
