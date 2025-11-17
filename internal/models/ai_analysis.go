package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// StringArray is a custom type for storing string arrays
// Compatible with both PostgreSQL (JSONB) and SQLite (TEXT)
type StringArray []string

// Scan implements sql.Scanner interface for reading from database
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("unsupported type for StringArray")
	}

	return json.Unmarshal(bytes, a)
}

// Value implements driver.Valuer interface for writing to database
func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// AIAnalysis represents the result of AI-powered verification
type AIAnalysis struct {
	ID                uint        `json:"id" gorm:"primaryKey;autoIncrement"`
	ReportID          uint        `json:"report_id" gorm:"not null;uniqueIndex"` // One analysis per report
	
	// AI Verdict
	Verdict           string      `json:"verdict" gorm:"type:varchar(20);not null"`
	Confidence        float64     `json:"confidence" gorm:"not null"`
	
	// Analysis Details
	Reasoning         string      `json:"reasoning" gorm:"type:text;not null"`
	RedFlags          StringArray `json:"red_flags" gorm:"type:text;serializer:json"` // ✅ Changed for SQLite
	SupportingFactors StringArray `json:"supporting_factors" gorm:"type:text;serializer:json"` // ✅ Changed for SQLite
	Recommendation    string      `json:"recommendation" gorm:"type:text"`
	
	// Raw AI Response (for debugging/audit)
	RawResponse       string      `json:"raw_response,omitempty" gorm:"type:text"`
	
	// Metadata
	AnalyzedAt        time.Time   `json:"analyzed_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time   `json:"updated_at" gorm:"autoUpdateTime"`
	
	// Model information
	AIModel           string      `json:"ai_model" gorm:"type:varchar(100);default:'mistral-large-latest'"`
	
	// Relations
	Report            *Report     `json:"report,omitempty" gorm:"foreignKey:ReportID;constraint:OnDelete:CASCADE"`
}

// AIAnalysisResponse is the expected JSON response from Mistral AI
type AIAnalysisResponse struct {
	Verdict           string   `json:"verdict"`
	Confidence        float64  `json:"confidence"`
	Reasoning         string   `json:"reasoning"`
	RedFlags          []string `json:"redFlags"`
	SupportingFactors []string `json:"supportingFactors"`
	Recommendation    string   `json:"recommendation"`
}

// TableName specifies the table name for AIAnalysis model
func (AIAnalysis) TableName() string {
	return "ai_analyses"
}

// Constants for AI verdict types
const (
	VerdictVerified    = "verified"
	VerdictHoax        = "hoax"
	VerdictUnconfirmed = "unconfirmed"
)

// IsValid checks if the verdict is valid
func (a *AIAnalysis) IsValid() bool {
	return a.Verdict == VerdictVerified || 
	       a.Verdict == VerdictHoax || 
	       a.Verdict == VerdictUnconfirmed
}

// MarshalJSON implements custom JSON marshaling
func (a *AIAnalysis) MarshalJSON() ([]byte, error) {
	type Alias AIAnalysis
	return json.Marshal(&struct {
		*Alias
		RedFlags          []string `json:"red_flags"`
		SupportingFactors []string `json:"supporting_factors"`
	}{
		Alias:             (*Alias)(a),
		RedFlags:          []string(a.RedFlags),
		SupportingFactors: []string(a.SupportingFactors),
	})
}