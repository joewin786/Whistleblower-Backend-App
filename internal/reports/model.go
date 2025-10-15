package reports

import "time"

type Report struct {
	ID          string    `json:"id" gorm:"type:char(36);primaryKey"`
	UserUID     string    `json:"user_uid" gorm:"type:char(36);index"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category" gorm:"index"`
	Status      string    `json:"status" gorm:"index"` // OPEN | IN_PROGRESS | RESOLVED
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// Request DTO
type CreateReportRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	Category    string `json:"category" binding:"required"`
}

type UpdateReportRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`
	Status      *string `json:"status,omitempty"`
}
