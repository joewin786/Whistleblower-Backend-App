package models

import (
	"time"
)

type Action struct{
	ID string `json:"id" gorm:"primaryKey"`
	ReportID uint `json:"report_id" gorm:"not null;index"`
	ActionDescription string `json:"action_description" gorm:"type:text;not null"`
	Department string `json:"department" gorm:"type:varchar(100)"`
	ResponsiblePerson string `json:"responsible_person" gorm:"type:varchar(100)"`
	HandleAt *time.Time `json:"handle_at"`
	EstimatedCompletion *time.Time `json:"estimated_completion"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateActionRequest struct {
	ActionDescription string `json:"action_description"`
	Department string `json:"department"`
	ResponsiblePerson string `json:"responsible_person"`
	HandleAt *time.Time `json:"handle_at"`
	EstimatedCompletion *time.Time `json:"estimated_completion"`
}