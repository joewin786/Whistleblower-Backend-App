package models

import (
	"time"
)

type Setting struct{
	ID uint `json:"id" gorm:"primaryKey;autoIncrement"`
	Key string `json:"key" gorm:"unique;not null"`
	Value string `json:"value"`
	UpdateAt time.Time `json:"update_at"`
}