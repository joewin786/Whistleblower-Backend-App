package models

import (
	"time"
)

type FeedbackType struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string    `json:"name" gorm:"not null;unique"`
	Description string    `json:"description" gorm:"type:text"`
	Icon        string    `json:"icon" gorm:"type:varchar(50)"` // emoji or icon name
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type Feedback struct {
	ID               uint          `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID           *string       `json:"user_id,omitempty" gorm:"type:char(36);index"` // null if anonymous
	FeedbackTypeID   uint          `json:"feedback_type_id" gorm:"not null;index"`
	Description      string        `json:"description" gorm:"type:text;not null"`
	ImagePath        *string       `json:"image_path,omitempty" gorm:"type:varchar(255)"` // optional screenshot
	Status           string        `json:"status" gorm:"default:'pending';index"` // pending, reviewed, resolved
	AdminResponse    *string       `json:"admin_response,omitempty" gorm:"type:text"`
	RespondedBy      *string       `json:"responded_by,omitempty" gorm:"type:char(36)"` // admin ID
	RespondedAt      *time.Time    `json:"responded_at,omitempty"`
	IsAnonymous      bool          `json:"is_anonymous" gorm:"default:false"`
	ContactEmail     *string       `json:"contact_email,omitempty" gorm:"type:varchar(255)"` // for anonymous feedback
	CreatedAt        time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time     `json:"updated_at" gorm:"autoUpdateTime"`
	
	// Relations
	FeedbackType     *FeedbackType `json:"feedback_type,omitempty" gorm:"foreignKey:FeedbackTypeID"`
	User             *User         `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID"`
	AdminUser        *User         `json:"admin_user,omitempty" gorm:"foreignKey:RespondedBy;references:ID"`
}

func (FeedbackType) TableName() string {
	return "feedback_types"
}

// TableName for Feedback
func (Feedback) TableName() string {
	return "feedbacks"
}

type CreateFeedbackTypeRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// DTO for updating feedback type
type UpdateFeedbackTypeRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// DTO for creating feedback
type CreateFeedbackRequest struct {
	FeedbackTypeID uint    `json:"feedback_type_id" binding:"required"`
	Description    string  `json:"description" binding:"required"`
	ContactEmail   *string `json:"contact_email,omitempty"` // required if anonymous
}

// DTO for admin response
type AdminResponseRequest struct {
	AdminResponse string `json:"admin_response" binding:"required"`
	Status        string `json:"status" binding:"required,oneof=reviewed resolved"`
}
