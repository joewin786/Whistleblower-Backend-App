package chatai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Struktur request/response untuk Gemini API
type GeminiChatRequest struct {
	Contents         []GeminiContent           `json:"contents"`
	GenerationConfig GeminiGenerationConfig    `json:"generationConfig"`
	SafetySettings   []GeminiSafetySetting     `json:"safetySettings,omitempty"`
}

type GeminiChatResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"` // "user" atau "model"
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	TopK            int     `json:"topK"`
	TopP            float64 `json:"topP"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// =============================================
//  CHAT AGENT FUNCTION (GEMINI 2.0/2.5)
// =============================================

func ChatAgentReply(systemPrompt, userMsg string) (string, error) {
	// Rate limiting: tunggu jika sudah melebihi quota
	geminiRateLimiter.WaitForRateLimit()

	// ✅ GUNAKAN API KEY TERPISAH UNTUK CHAT
	apiKey := os.Getenv("GEMINI_CHAT_API_KEY")
	if apiKey == "" {
		// Fallback ke API key utama jika tidak ada
		apiKey = os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return "", fmt.Errorf("GEMINI_CHAT_API_KEY or GEMINI_API_KEY not found")
		}
		fmt.Println("⚠️ Using fallback API key (GEMINI_API_KEY)")
	} else {
		fmt.Println("✅ Using dedicated chat API key (GEMINI_CHAT_API_KEY)")
	}

	// ✅ GUNAKAN MODEL YANG TERSEDIA (dari list Anda sebelumnya)
	// RECOMMENDED: gemini-2.5-flash (Stable, Fast, Smart)
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=" + apiKey
	
	// Alternative Options (uncomment jika mau ganti):
	// url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=" + apiKey
	// url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-flash-lite-latest:generateContent?key=" + apiKey

	// GABUNGKAN SYSTEM PROMPT + USER MESSAGE
	fullPrompt := systemPrompt + "\n\nUser: " + userMsg

	reqBody := GeminiChatRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: fullPrompt},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:     0.2,  // Konsisten untuk customer service
			TopK:            40,
			TopP:            0.9,
			MaxOutputTokens: 500,  // Cukup untuk jawaban singkat
		},
		SafetySettings: []GeminiSafetySetting{
			{
				Category:  "HARM_CATEGORY_HARASSMENT",
				Threshold: "BLOCK_NONE",
			},
			{
				Category:  "HARM_CATEGORY_HATE_SPEECH",
				Threshold: "BLOCK_NONE",
			},
			{
				Category:  "HARM_CATEGORY_SEXUALLY_EXPLICIT",
				Threshold: "BLOCK_NONE",
			},
			{
				Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
				Threshold: "BLOCK_NONE",
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// DEBUG LOG (boleh dimatikan jika tidak perlu)
	fmt.Println("ChatAgent Raw Response:", string(body))

	// Parse response
	var gemResp GeminiChatResponse
	if err := json.Unmarshal(body, &gemResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if gemResp.Error != nil {
		return "", fmt.Errorf("Gemini API error [%d]: %s", gemResp.Error.Code, gemResp.Error.Message)
	}

	// Check response status
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Gemini error (status %d): %s", resp.StatusCode, string(body))
	}

	// Extract reply
	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return "Maaf, saya tidak mengerti pertanyaan Anda.", nil
	}

	reply := gemResp.Candidates[0].Content.Parts[0].Text
	return reply, nil
}