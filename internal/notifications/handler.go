package notifications

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"

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

// Kirim notifikasi dari admin ke user (manual)
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

		channel := fmt.Sprintf("user-%s", *report.UserID)
		event := "status-updated"

		// Kirim notifikasi ke user
		err := Client.Trigger(channel, event, map[string]any{
			"title":     payload.Title,
			"message":   payload.Message,
			"type":      payload.Type,
			"report_id": payload.ReportID,
			"timestamp": time.Now().Unix(),
		})
		if err != nil {
			fmt.Printf("[WARN] gagal kirim notifikasi ke %s: %v\n", channel, err)
			utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		fmt.Printf("[INFO] ✅ Notifikasi dikirim ke %s (report #%d)\n", channel, payload.ReportID)
		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"status":  "notification sent",
			"channel": channel,
		})
	}
}

// ✅ FUNGSI UTAMA: Kirim notifikasi otomatis saat status berubah
// ✅ FUNGSI UTAMA: Kirim notifikasi otomatis saat status berubah
// ✅ FUNGSI UTAMA: Kirim notifikasi otomatis saat status berubah
func NotifyStatusChange(db *gorm.DB, reportID uint, oldStatus, newStatus string) error {
	// Ambil data report
	var report models.Report
	if err := db.Preload("User").First(&report, reportID).Error; err != nil {
		fmt.Printf("[ERROR] Report #%d tidak ditemukan: %v\n", reportID, err)
		return err
	}

	// Mapping status ke pesan yang user-friendly (sesuaikan dengan models.Status*)
	statusMessages := map[string]struct {
		Title   string
		Message string
		Type    string
	}{
		"submitted": {
			Title:   "📋 Laporan Diterima",
			Message: "Laporan Anda telah diterima dan sedang menunggu peninjauan",
			Type:    "info",
		},
		"under_review": {
			Title:   "🔍 Laporan Sedang Ditinjau",
			Message: "Tim kami sedang menindaklanjuti laporan Anda",
			Type:    "info",
		},
		"resolved": {
			Title:   "✅ Laporan Selesai",
			Message: "Laporan Anda telah diselesaikan. Terima kasih atas laporannya!",
			Type:    "success",
		},
		"dismissed": {
			Title:   "❌ Laporan Ditolak",
			Message: "Maaf, laporan Anda tidak dapat diproses lebih lanjut",
			Type:    "error",
		},
		"need_info": {
			Title:   "📝 Informasi Tambahan Diperlukan",
			Message: "Kami membutuhkan informasi tambahan terkait laporan Anda",
			Type:    "warning",
		},
	}

	// Ambil pesan untuk status baru
	statusInfo, exists := statusMessages[newStatus]
	if !exists {
		statusInfo = struct {
			Title   string
			Message string
			Type    string
		}{
			Title:   "📢 Status Laporan Diperbarui",
			Message: fmt.Sprintf("Status laporan Anda berubah menjadi: %s", newStatus),
			Type:    "info",
		}
	}

	// Format channel berdasarkan UserID (UUID string)
	// Handle pointer dengan benar
	var channel string
	if report.UserID != nil && *report.UserID != "" {
		channel = fmt.Sprintf("user-%s", *report.UserID)
	} else if report.Email != nil {
		// Untuk anonymous user, skip notifikasi atau gunakan email
		fmt.Printf("[INFO] ⚠️ Report #%d tidak memiliki UserID (anonymous), notifikasi dilewati\n", reportID)
		return nil
	} else {
		fmt.Printf("[WARN] Report #%d tidak memiliki UserID atau Email\n", reportID)
		return nil
	}

	event := "status-updated"

	// Data notifikasi
	notifData := map[string]any{
		"title":      statusInfo.Title,
		"message":    statusInfo.Message,
		"type":       statusInfo.Type,
		"report_id":  reportID,
		"old_status": oldStatus,
		"new_status": newStatus,
		"timestamp":  time.Now().Unix(),
	}

	// Kirim ke Pusher
	err := Client.Trigger(channel, event, notifData)
	if err != nil {
		fmt.Printf("[ERROR] ❌ Gagal kirim notifikasi status change ke %s: %v\n", channel, err)
		return err
	}

	fmt.Printf("[INFO] ✅ Notifikasi status change terkirim ke %s (Report #%d: %s → %s)\n", 
		channel, reportID, oldStatus, newStatus)

	return nil
}

// ✅ FUNGSI TAMBAHAN: Notifikasi saat ada laporan baru (untuk admin)
func NotifyNewReport(reportID uint, title string) error {
	channel := "admin-notifications"
	event := "new-report"

	data := map[string]any{
		"title":     "🆕 Laporan Baru Masuk",
		"message":   fmt.Sprintf("Laporan baru: %s", title),
		"type":      "info",
		"report_id": reportID,
		"timestamp": time.Now().Unix(),
	}

	if err := Client.Trigger(channel, event, data); err != nil {
		fmt.Printf("[ERROR] ❌ Gagal notifikasi report baru: %v\n", err)
		return err
	}

	fmt.Printf("[INFO] ✅ Admin di-notifikasi tentang report baru #%d\n", reportID)
	return nil
}

// ✅ FUNGSI TAMBAHAN: Notifikasi saat ada pesan chat baru
func NotifyNewChatMessage(db *gorm.DB, reportID uint, senderID, message string, isFromAdmin bool) error {
	var report models.Report
	if err := db.First(&report, reportID).Error; err != nil {
		return err
	}

	var channel string
	var title, notifMessage string

	if isFromAdmin {
		// Notifikasi ke user (reporter)
		if report.UserID != nil && *report.UserID != "" {
			channel = fmt.Sprintf("user-%s", *report.UserID)
			title = "💬 Pesan Baru dari Admin"
			notifMessage = "Admin telah membalas laporan Anda"
		} else {
			// Skip untuk anonymous user
			return nil
		}
	} else {
		// Notifikasi ke admin
		channel = "admin-notifications"
		title = "💬 Pesan Baru dari Reporter"
		notifMessage = fmt.Sprintf("Reporter mengirim pesan untuk laporan #%d", reportID)
	}

	event := "new-message"

	// Ambil preview maksimal 50 karakter
	preview := message
	if len(message) > 50 {
		preview = message[:50]
	}

	data := map[string]any{
		"title":      title,
		"message":    notifMessage,
		"type":       "info",
		"report_id":  reportID,
		"sender_id":  senderID,
		"preview":    preview,
		"timestamp":  time.Now().Unix(),
	}

	if err := Client.Trigger(channel, event, data); err != nil {
		fmt.Printf("[ERROR] ❌ Gagal kirim notifikasi chat: %v\n", err)
		return err
	}

	fmt.Printf("[INFO] ✅ Notifikasi chat terkirim ke %s (Report #%d)\n", channel, reportID)
	return nil
}