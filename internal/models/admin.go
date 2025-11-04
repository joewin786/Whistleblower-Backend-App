package models

import "time"

// Admin merepresentasikan user/staff internal (bisa admin, investigator, dsb)
type Admin struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	FullName    string    `json:"full_name" gorm:"type:varchar(100);not null"`
	Email       string    `json:"email" gorm:"type:varchar(100);unique;not null"`
	Department  string    `json:"department" gorm:"type:varchar(100);not null"`
	Role        string    `json:"role" gorm:"type:varchar(50);default:'admin'"` // misalnya admin/investigator
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
