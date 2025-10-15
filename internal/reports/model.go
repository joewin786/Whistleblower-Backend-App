package reports

import "time"

type Report struct {
	ID          string    `json:"id"`
	UserUID     string    `json:"user_uid"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}
