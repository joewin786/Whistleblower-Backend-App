package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"whistleblower_REST/internal/models"
)

// GeminiService handles interactions with Google Gemini AI API
type GeminiService struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// NewGeminiService creates a new instance of GeminiService
func NewGeminiService() *GeminiService {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("[WARN] ⚠️ GEMINI_API_KEY not set in environment variables")
	}

	return &GeminiService{
		APIKey:  apiKey,
		// ✅ Using gemini-pro (most stable and widely available)
		BaseURL: "https://generativelanguage.googleapis.com/v1/models/gemini-2.5-flash:generateContent",
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GeminiRequest represents the request body for Gemini AI
type GeminiRequest struct {
	Contents         []GeminiContent         `json:"contents"`
	GenerationConfig GeminiGenerationConfig  `json:"generationConfig"`
	SafetySettings   []GeminiSafetySetting   `json:"safetySettings,omitempty"`
}

// GeminiContent represents content in the request
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents a part of the content
type GeminiPart struct {
	Text string `json:"text"`
}


// GeminiGenerationConfig represents generation configuration
type GeminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	TopK            int     `json:"topK"`
	TopP            float64 `json:"topP"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

// GeminiSafetySetting represents safety settings
type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GeminiResponse represents the response from Gemini AI
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// AnalyzeReport analyzes a report using Gemini AI based on admin review
func (s *GeminiService) AnalyzeReport(report *models.Report, review *models.Review) (*models.AIAnalysisResponse, string, error) {
	if s.APIKey == "" {
		return nil, "", fmt.Errorf("Gemini API key not configured")
	}

	// Build the prompt
	prompt := s.buildPrompt(report, review)

	// Prepare request
	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{
						Text: prompt,
					},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:     0.3, // Lower temperature for more consistent results
			TopK:            40,
			TopP:            0.95,
			MaxOutputTokens: 2048,
		},
		SafetySettings: []GeminiSafetySetting{
			{
				Category:  "HARM_CATEGORY_HATE_SPEECH",
				Threshold: "BLOCK_NONE",
			},
			{
				Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
				Threshold: "BLOCK_NONE",
			},
			{
				Category:  "HARM_CATEGORY_SEXUALLY_EXPLICIT",
				Threshold: "BLOCK_NONE",
			},
			{
				Category:  "HARM_CATEGORY_HARASSMENT",
				Threshold: "BLOCK_NONE",
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request with API key in URL
	url := fmt.Sprintf("%s?key=%s", s.BaseURL, s.APIKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	fmt.Println("[INFO] 🤖 Sending request to Gemini AI...")
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, "", fmt.Errorf("no response from Gemini AI")
	}

	rawContent := geminiResp.Candidates[0].Content.Parts[0].Text
	fmt.Printf("[INFO] ✅ Received response from Gemini AI (%d tokens)\n", geminiResp.UsageMetadata.TotalTokenCount)

	// Clean and parse JSON response
	cleanedContent := s.cleanJSONResponse(rawContent)
	
	var aiResult models.AIAnalysisResponse
	if err := json.Unmarshal([]byte(cleanedContent), &aiResult); err != nil {
		fmt.Printf("[WARN] ⚠️ Failed to parse AI response as JSON: %v\n", err)
		// Return a default response if parsing fails
		return &models.AIAnalysisResponse{
			Verdict:           models.VerdictUnconfirmed,
			Confidence:        0,
			Reasoning:         "AI gagal memparse response. Response mentah tersimpan untuk review manual.",
			RedFlags:          []string{"AI parsing error"},
			SupportingFactors: []string{},
			Recommendation:    "Perlu review manual lebih lanjut oleh tim investigasi.",
		}, rawContent, nil
	}

	return &aiResult, rawContent, nil
}



// buildPrompt creates the prompt for Gemini AI
func (s *GeminiService) buildPrompt(report *models.Report, review *models.Review) string {
	// Get evidence count (if you have evidence relation)
	evidenceCount := 0
	// Note: Jika Anda punya relasi Evidence di Report, uncomment ini:
	// evidenceCount = len(report.Evidences)

	prompt := fmt.Sprintf(`Kamu adalah AI expert dalam mendeteksi hoax dan misinformasi untuk sistem Whistleblower. Analisis laporan berikut berdasarkan review admin yang telah dilakukan.

INFORMASI LAPORAN:
- ID Laporan: #%d
- Judul: %s
- Kategori: %s
- Deskripsi: %s
- Tipe Pelapor: %s
- Status Saat Ini: %s
- Bukti yang dilampirkan: %d file/dokumen
- Tanggal Laporan: %s

PENILAIAN ADMIN REVIEWER:
- Kredibilitas Sumber: %d/10
- Kualitas Bukti: %d/10
- Konsistensi Cerita: %d/10
- Reliabilitas Informasi: %d/10
- Tingkat Urgensi: %s
- Skor Keseluruhan: %.2f/10
- Catatan Admin: %s

KRITERIA PENILAIAN:
- "verified": Jika skor overall ≥ 7.0 dan tidak ada red flags mayor
- "hoax": Jika skor overall < 4.0 atau ada banyak red flags signifikan  
- "unconfirmed": Jika skor 4.0-6.9 atau perlu investigasi lebih lanjut

INSTRUKSI PENTING:
1. Berikan HANYA JSON yang valid, tanpa teks lain sama sekali
2. Jangan gunakan markdown code blocks ()
3. Jangan tambahkan penjelasan di luar JSON
4. Format HARUS persis seperti ini:

{"verdict":"verified","confidence":85.5,"reasoning":"Penjelasan lengkap mengapa verdict ini dipilih berdasarkan data yang ada. Minimal 3-4 kalimat yang menjelaskan analisis secara detail.","redFlags":["Red flag 1","Red flag 2"],"supportingFactors":["Faktor pendukung 1","Faktor pendukung 2"],"recommendation":"Rekomendasi spesifik dan actionable untuk tim investigasi"}

Berikan response JSON sekarang:`,
		report.ID,
		report.Title,
		report.Category,
		report.Description,
		report.ReporterType,
		report.Status,
		evidenceCount,
		report.CreatedAt.Format("2006-01-02 15:04:05"),
		review.CredibilityScore,
		review.EvidenceQuality,
		review.ConsistencyScore,
		review.SourceReliability,
		review.UrgencyLevel,
		review.OverallScore,
		review.ReviewNotes,
	)

	return prompt
}

// cleanJSONResponse removes markdown code blocks and extra whitespace
func (s *GeminiService) cleanJSONResponse(content string) string {
	// Remove markdown code blocks
	content = strings.TrimPrefix(content, "```json\n")
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```\n")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "\n```")
	content = strings.TrimSuffix(content, "```")
	
	// Trim whitespace
	content = strings.TrimSpace(content)
	
	return content
}

// ValidateAPIKey checks if the API key is configured
func (s *GeminiService) ValidateAPIKey() error {
	if s.APIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY is not set in environment variables")
	}
	return nil
}