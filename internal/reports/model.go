package reports

import "time"

const (
	StatusSubmitted = "submitted"
	StatusUnderReview = "under_review"
	StatusResolved = "resolved"
	StatusDismissed = "dismissed"

	ReporterAnonymous = "anonymous"
	ReporterAuthenticated = "authenticated"
)

type Report struct {
	ID           uint      `json:"id"            gorm:"primaryKey;autoIncrement"`
	UserID       *string   `json:"user_uid,omitempty" gorm:"type:char(36);index"`
	Title        string    `json:"title"         gorm:"not null"`
	Description  string    `json:"description"`
	Category     string    `json:"category"`
	Status       string    `json:"status"        gorm:"default:submitted;index"`
	ReporterType string    `json:"reporterType"  gorm:"not null;default:anonymous"`
	Email        *string   `json:"email,omitempty" gorm:"type:text"`
	CreatedAt    time.Time `json:"createdAt"     gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updatedAt"     gorm:"autoUpdateTime"`
}

// Request DTO
type CreateReportRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Email       *string `json:"email,omitempty"`
}

type UpdateReportRequest struct {
	Status *string `json:"status,omitempty"`
}
