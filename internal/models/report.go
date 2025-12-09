package models

import "time"

const (
	StatusSubmitted = "submitted"
	StatusUnderReview = "under_review"
	StatusOnProcess = "on_process"
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
	AssignedAdminID *string  `json:"assignedAdminID,omitempty" gorm:"type:char(36);index"` // ✅ ini penting!
	AssignedAdmin   *User   `json:"assignedAdmin,omitempty" gorm:"foreignKey:AssignedAdminID;references:ID"`
	CreatedAt    time.Time `json:"createdAt"     gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updatedAt"     gorm:"autoUpdateTime"`
	User *User `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:UserID;references:ID"`
	Actions       []Action   `json:"actions" gorm:"foreignKey:ReportID;constraint:OnDelete:CASCADE;"`
	InvestigatorID  *uint      `json:"investigator_id,omitempty" gorm:"type:integer;index;constraint:OnDelete:SET NULL"`
	Investigator    *Admin     `json:"investigator,omitempty" gorm:"foreignKey:InvestigatorID;references:ID"`
	AssignedAt      *time.Time `json:"assigned_at,omitempty"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
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
