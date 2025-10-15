package evidence

import "time"

type Evidence struct {
	ID        string    `json:"id" gorm:"type:char(36);primaryKey"`
	ReportID  string    `json:"report_id" gorm:"type:char(36);index"`
	FilePath  string    `json:"file_path"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// optional kalau mau terima body JSON juga (selain path param)
type CreateEvidenceRequest struct {
	FilePath string `json:"file_path" binding:"required"`
}
