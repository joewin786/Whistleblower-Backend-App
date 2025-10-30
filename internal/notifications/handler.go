package notifications

import (
	"encoding/json"
	"fmt"
	"net/http"
	"whistleblower_REST/internal/utils"
	"whistleblower_REST/internal/models"
	"gorm.io/gorm"
)

type Notification struct {
	Channel string `json:"channel,omitempty"`
	Event   string `json:"event,omitempty"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

type AdminNotifyByReportRequest struct {
	ReportID uint   `json:"report_id"`
	Title    string `json:"title"`
	Message  string `json:"message"`
	Type     string `json:"type"`
}

// Broadcast umum (opsional)
func SendNotification(w http.ResponseWriter, r *http.Request) {
	var notif Notification
	if err := json.NewDecoder(r.Body).Decode(&notif); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	channel := notif.Channel
	if channel == "" {
		channel = "notification-channel"
	}
	event := notif.Event
	if event == "" {
		event = "new-notification"
	}

	if err := Client.Trigger(channel, event, notif); err != nil {
		fmt.Printf("[WARN] Gagal kirim notifikasi ke %s: %v\n", channel, err)
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	fmt.Printf("[INFO] Broadcast notifikasi ke %s: %s - %s\n", channel, notif.Title, notif.Message)
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "notification sent", "channel": channel})
}

// Kirim notifikasi dari admin ke user
func SendFromAdminByReport(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value("role").(string)
		if role != "admin" {
			utils.RespondWithError(w, http.StatusForbidden, "forbidden: only admin can send notifications")
			return
		}

		var payload AdminNotifyByReportRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if payload.ReportID == 0 {
			utils.RespondWithError(w, http.StatusBadRequest, "missing report_id")
			return
		}

		// Cari report terkait untuk dapatkan UserID
		var report models.Report
		if err := db.First(&report, payload.ReportID).Error; err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "report not found")
			return
		}

		channel := fmt.Sprintf("user-%d", report.UserID)
		event := "status-updated"

		// Kirim notifikasi ke user
		err := Client.Trigger(channel, event, map[string]any{
			"title":   payload.Title,
			"message": payload.Message,
			"type":    payload.Type,
			"report":  payload.ReportID,
		})
		if err != nil {
			fmt.Printf("[WARN] gagal kirim notifikasi ke %s: %v\n", channel, err)
			utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		fmt.Printf("[INFO] notifikasi dikirim ke %s (report #%d)\n", channel, payload.ReportID)
		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"status":  "notification sent",
			"channel": channel,
		})
	}
}