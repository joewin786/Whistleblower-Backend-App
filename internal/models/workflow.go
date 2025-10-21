package models

import (
	"time"
)

type Workflow struct{
	ID uint `json:"id" gorm:"primaryKey;autoIncrement"`
	Name string `json:"name" gorm:"unique;not null"`
	Order int64 `json:"order"`
	IsActive bool `json:"isActive" gorm:"default:true"`
	UpdateAt time.Time `json:"update_at"`
}