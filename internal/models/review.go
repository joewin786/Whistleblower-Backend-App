package models

import "time"

// Review represents admin's assessment of a report before AI analysis
type Review struct {
	ID                uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	ReportID          uint       `json:"report_id" gorm:"not null;uniqueIndex"` // One review per report
	AdminID uint `json:"admin_id" gorm:"not null"`
	
	// Scoring criteria (1-10)
	CredibilityScore  int        `json:"credibility_score" gorm:"not null;check:credibility_score >= 1 AND credibility_score <= 10"`
	EvidenceQuality   int        `json:"evidence_quality" gorm:"not null;check:evidence_quality >= 1 AND evidence_quality <= 10"`
	ConsistencyScore  int        `json:"consistency_score" gorm:"not null;check:consistency_score >= 1 AND consistency_score <= 10"`
	SourceReliability int        `json:"source_reliability" gorm:"not null;check:source_reliability >= 1 AND source_reliability <= 10"`
	
	// Additional assessment
	UrgencyLevel      string     `json:"urgency_level" gorm:"type:varchar(20);not null"` // low, medium, high, critical
	ReviewNotes       string     `json:"review_notes" gorm:"type:text"`
	OverallScore      float64    `json:"overall_score" gorm:"not null"`
	
	// Metadata
	ReviewedAt        time.Time  `json:"reviewed_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	
	// Relations
	Report            *Report    `json:"report,omitempty" gorm:"foreignKey:ReportID;constraint:OnDelete:CASCADE"`
	Admin *Admin `json:"admin,omitempty" gorm:"foreignKey:AdminID;references:ID"`
}

// CreateReviewRequest for creating a new review
type CreateReviewRequest struct {
	ReportID          uint    `json:"report_id" binding:"required"`
	CredibilityScore  int     `json:"credibility_score" binding:"required,min=1,max=10"`
	EvidenceQuality   int     `json:"evidence_quality" binding:"required,min=1,max=10"`
	ConsistencyScore  int     `json:"consistency_score" binding:"required,min=1,max=10"`
	SourceReliability int     `json:"source_reliability" binding:"required,min=1,max=10"`
	UrgencyLevel      string  `json:"urgency_level" binding:"required,oneof=low medium high critical"`
	ReviewNotes       string  `json:"review_notes"`
}

// UpdateReviewRequest for updating an existing review
type UpdateReviewRequest struct {
	CredibilityScore  *int    `json:"credibility_score,omitempty" binding:"omitempty,min=1,max=10"`
	EvidenceQuality   *int    `json:"evidence_quality,omitempty" binding:"omitempty,min=1,max=10"`
	ConsistencyScore  *int    `json:"consistency_score,omitempty" binding:"omitempty,min=1,max=10"`
	SourceReliability *int    `json:"source_reliability,omitempty" binding:"omitempty,min=1,max=10"`
	UrgencyLevel      *string `json:"urgency_level,omitempty" binding:"omitempty,oneof=low medium high critical"`
	ReviewNotes       *string `json:"review_notes,omitempty"`
}

// CalculateOverallScore calculates the average of all scores
func (r *Review) CalculateOverallScore() {
	r.OverallScore = float64(r.CredibilityScore+r.EvidenceQuality+r.ConsistencyScore+r.SourceReliability) / 4.0
}

// TableName specifies the table name for Review model
func (Review) TableName() string {
	return "reviews"
}