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
func SendNotification(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		// Simpan ke DB (untuk admin dashboard)
		db.Create(&models.Notification{
			Title:     notif.Title,
			Message:   notif.Message,
			Type:      notif.Type,
			CreatedAt: time.Now(),
		})

		// Kirim realtime via Pusher
		if err := Client.Trigger(channel, event, notif); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

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
	var report models.Report
	if err := db.Preload("User").First(&report, reportID).Error; err != nil {
		fmt.Printf("[ERROR] Report #%d tidak ditemukan: %v\n", reportID, err)
		return err
	}

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

	// 🔹 Pastikan hanya user login yang dikirimi notifikasi
	if report.UserID == nil || *report.UserID == "" {
		fmt.Printf("[INFO] ⚠️ Report #%d tidak memiliki UserID (anonymous), notifikasi dilewati\n", reportID)
		return nil
	}

	channel := fmt.Sprintf("user-%s", *report.UserID)
	event := "status-updated"

	payload := map[string]any{
		"title":      statusInfo.Title,
		"message":    statusInfo.Message,
		"type":       statusInfo.Type,
		"report_id":  reportID,
		"old_status": oldStatus,
		"new_status": newStatus,
		"timestamp":  time.Now().Unix(),
	}

	// 🔹 1. Kirim realtime via Pusher
	if err := Client.Trigger(channel, event, payload); err != nil {
		fmt.Printf("[ERROR] ❌ Gagal kirim notifikasi status change ke %s: %v\n", channel, err)
		return err
	}

	// 🔹 2. Simpan ke database
	notif := models.UserNotification{
		UserID:    *report.UserID,
		Title:     statusInfo.Title,
		Message:   statusInfo.Message,
		Type:      statusInfo.Type,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&notif).Error; err != nil {
		fmt.Printf("[WARN] ⚠️ Gagal simpan notifikasi user ke DB: %v\n", err)
	} else {
		fmt.Printf("[INFO] ✅ Notifikasi user disimpan ke DB untuk user_id=%s\n", *report.UserID)
	}

	return nil
}


// ✅ FUNGSI TAMBAHAN: Notifikasi saat ada laporan baru (untuk admin)
func NotifyNewReport(db *gorm.DB, reportID uint, title string) error {
	channel := "admin-notifications"
	event := "new-report"

	data := map[string]any{
		"title":     "🆕 Laporan Baru Masuk",
		"message":   fmt.Sprintf("Laporan baru: %s", title),
		"type":      "info",
		"report_id": reportID,
		"timestamp": time.Now().Unix(),
	}

	// 🔹 1. Kirim notifikasi realtime ke admin
	if err := Client.Trigger(channel, event, data); err != nil {
		fmt.Printf("[ERROR] ❌ Gagal kirim notifikasi report baru: %v\n", err)
		return err
	}

	// 🔹 2. Simpan ke database supaya bisa ditampilkan di halaman notifikasi
	notif := models.Notification{
		Title:     "🆕 Laporan Baru Masuk",
		Message:   fmt.Sprintf("Laporan baru: %s", title),
		Type:      "info",
		CreatedAt: time.Now(),
	}
	if err := db.Create(&notif).Error; err != nil {
		fmt.Printf("[WARN] ⚠️ Gagal simpan notifikasi ke DB: %v\n", err)
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

func GetAllAdminNotifications(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var notifications []models.Notification

		// Ambil dari database, urutkan terbaru
		if err := db.Order("created_at DESC").Find(&notifications).Error; err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to fetch notifications")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, notifications)
	}
}

func GetUserNotifications(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := r.Context().Value("id").(string)
		if !ok || uid == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "missing user id in token")
			return
		}

		var list []models.UserNotification
		if err := db.Where("user_id = ?", uid).
			Order("created_at DESC").
			Find(&list).Error; err != nil {
			utils.RespondWithError(w, 500, "failed to load user notifications")
			return
		}
		utils.RespondWithJSON(w, 200, list)
	}
}