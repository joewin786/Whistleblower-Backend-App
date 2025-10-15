package evidence

import "time"

// Evidence mewakili satu file bukti laporan
type Evidence struct {
	ID        string    `json:"id"`
	ReportID  string    `json:"report_id"`
	FilePath  string    `json:"file_path"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateEvidenceRequest digunakan untuk tambah bukti baru
type CreateEvidenceRequest struct {
	ReportID string `json:"report_id" binding:"required"`
	FilePath string `json:"file_path" binding:"required"`
}
