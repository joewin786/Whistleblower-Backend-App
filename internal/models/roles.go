package models

import(
	"time"
)

type Role struct{
	ID uint `json:"id" gorm:"primaryKey;autoIcnrement"`
	Name string `json:"name" gorm:"unique;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"update_at"`
}