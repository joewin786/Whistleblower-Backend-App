package notifications

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"

	"github.com/go-chi/chi/v5"

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
			Title:   "Laporan Diterima",
			Message: "Laporan Anda telah diterima dan sedang menunggu peninjauan",
			Type:    "info",
		},
		"under_review": {
			Title:   "Laporan Sedang Ditinjau",
			Message: "Tim kami sedang menindaklanjuti laporan Anda",
			Type:    "info",
		},
		"resolved": {
			Title:   "Laporan Selesai",
			Message: "Laporan Anda telah diselesaikan. Terima kasih atas laporannya!",
			Type:    "success",
		},
		"dismissed": {
			Title:   "Laporan Ditolak",
			Message: "Maaf, laporan Anda tidak dapat diproses lebih lanjut",
			Type:    "error",
		},
		"need_info": {
			Title:   "Informasi Tambahan Diperlukan",
			Message: "Kami membutuhkan informasi tambahan terkait laporan Anda",
			Type:    "warning",
		},
	}

	statusInfo, exists := statusMessages[newStatus]
	if !exists {
		statusInfo = struct {
			Title, Message, Type string
		}{
			Title:   "📢 Status Laporan Diperbarui",
			Message: fmt.Sprintf("Status laporan Anda berubah menjadi: %s", newStatus),
			Type:    "info",
		}
	}

	if report.UserID == nil || *report.UserID == "" {
		fmt.Printf("[INFO] ⚠️ Report #%d tidak memiliki UserID, notifikasi dilewati\n", reportID)

		// ✅ Tambahan: kirim email ke reporter anonymous jika punya email
		if report.ReporterType == "anonymous" && report.Email != nil && *report.Email != "" {
			subject := fmt.Sprintf("📢 Status Laporan #%d Telah Berubah", reportID)
			body := fmt.Sprintf(`
				<h2>%s</h2>
				<p>%s</p>
				<hr>
				<p><strong>ID Laporan:</strong> #%d</p>
				<p><strong>Status Lama:</strong> %s</p>
				<p><strong>Status Baru:</strong> %s</p>
				<p style="color: gray; font-size: 12px;">Email ini dikirim otomatis oleh sistem Whistleblower.</p>
			`, statusInfo.Title, statusInfo.Message, reportID, oldStatus, newStatus)

			if err := utils.SendEmail(*report.Email, subject, body); err != nil {
				fmt.Printf("[EMAIL ERROR] gagal kirim ke %s: %v\n", *report.Email, err)
			} else {
				fmt.Printf("[EMAIL SENT] Notifikasi status #%d dikirim ke %s\n", reportID, *report.Email)
			}
		}

		return nil
	}

	channel := fmt.Sprintf("user-%s", *report.UserID)
	event := "status-updated"

	// 🆕 Perbarui judul & pesan agar menyertakan ID laporan
	enhancedTitle := fmt.Sprintf("Laporan #%d %s", reportID, statusInfo.Title[8:]) // Hapus kata "Laporan " dari awal
	enhancedMessage := statusInfo.Message

	payload := map[string]any{
		"title":      enhancedTitle,
		"message":    enhancedMessage,
		"type":       statusInfo.Type,
		"report_id":  reportID,
		"old_status": oldStatus,
		"new_status": newStatus,
		"timestamp":  time.Now().Unix(),
	}

	// 1️⃣ Kirim realtime via Pusher
	if err := Client.Trigger(channel, event, payload); err != nil {
		fmt.Printf("[ERROR] ❌ Gagal kirim notifikasi ke %s: %v\n", channel, err)
		return err
	}

	// 2️⃣ Simpan ke database
	notif := models.UserNotification{
		UserID:    *report.UserID,
		Title:     enhancedTitle,   // 🆕 sudah berisi "Laporan #ID"
		Message:   enhancedMessage, // 🆕 sudah berisi "ID: #ID"
		Type:      statusInfo.Type,
		ReportID:  &reportID,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&notif).Error; err != nil {
		fmt.Printf("[WARN] ⚠️ Gagal simpan notifikasi user ke DB: %v\n", err)
	} else {
		fmt.Printf("[INFO] ✅ Notifikasi user disimpan ke DB untuk user_id=%s (report #%d)\n", *report.UserID, reportID)
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

// ✅ Fungsi untuk ambil semua notifikasi admin dari database
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

// ✅ Fungsi untuk ambil notifikasi user dari database
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

// ✅ Fungsi untuk menandai semua notifikasi sebagai sudah dibaca
// Tambahkan fungsi-fungsi ini ke handler.go Anda
// Mark single notification as read
func MarkNotificationAsRead(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := r.Context().Value("id").(string)
		if !ok || uid == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "missing user id in token")
			return
		}

		notifID := chi.URLParam(r, "notificationId")
		
		var notif models.UserNotification
		if err := db.Where("id = ? AND user_id = ?", notifID, uid).First(&notif).Error; err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "notification not found")
			return
		}

		notif.IsRead = true
		if err := db.Save(&notif).Error; err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to update notification")
			return
		}

		fmt.Printf("[INFO] ✅ Notification #%s marked as read for user %s\n", notifID, uid)
		utils.RespondWithJSON(w, http.StatusOK, notif)
	}
}

// Mark all notifications as read
func MarkAllNotificationsAsRead(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := r.Context().Value("id").(string)
		if !ok || uid == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "missing user id in token")
			return
		}

		fmt.Printf("[DEBUG] Marking all notifications as read for user: %s\n", uid)

		result := db.Model(&models.UserNotification{}).
			Where("user_id = ? AND is_read = ?", uid, false).
			Update("is_read", true)

		if result.Error != nil {
			fmt.Printf("[ERROR] Failed to update notifications: %v\n", result.Error)
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to update notifications")
			return
		}

		fmt.Printf("[DEBUG] Updated %d notifications\n", result.RowsAffected)

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"message":        "all notifications marked as read",
			"rows_affected": result.RowsAffected,
		})
	}
}

// Delete notification
func DeleteNotification(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := r.Context().Value("id").(string)
		if !ok || uid == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "missing user id in token")
			return
		}

		notifID := chi.URLParam(r, "notificationId")
		
		result := db.Where("id = ? AND user_id = ?", notifID, uid).Delete(&models.UserNotification{})
		if result.Error != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to delete notification")
			return
		}

		if result.RowsAffected == 0 {
			utils.RespondWithError(w, http.StatusNotFound, "notification not found")
			return
		}

		fmt.Printf("[INFO] ✅ Notification #%s deleted for user %s\n", notifID, uid)
		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "notification deleted",
		})
	}
}

// ✅ Fungsi untuk menghapus notifikasi user berdasarkan ID
func DeleteUserNotification(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := r.Context().Value("id").(string)
		if !ok || uid == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "missing user id")
			return
		}

		notifID := chi.URLParam(r, "notifId")
		if notifID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "missing notification id")
			return
		}

		res := db.Where("id = ? AND user_id = ?", notifID, uid).Delete(&models.UserNotification{})
		if res.Error != nil {
			utils.RespondWithError(w, 500, "failed to delete notification")
			return
		}
		if res.RowsAffected == 0 {
			utils.RespondWithError(w, 404, "notification not found or unauthorized")
			return
		}

		utils.RespondWithJSON(w, 200, map[string]string{"message": "Notification deleted successfully"})
	}
}

// ✅ Fungsi untuk menghapus semua notifikasi user
func DeleteAllUserNotifications(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := r.Context().Value("id").(string)
		if !ok || uid == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "missing user id")
			return
		}

		if err := db.Where("user_id = ?", uid).Delete(&models.UserNotification{}).Error; err != nil {
			utils.RespondWithError(w, 500, "failed to delete all notifications")
			return
		}

		utils.RespondWithJSON(w, 200, map[string]string{"message": "All notifications deleted successfully"})
	}
}


