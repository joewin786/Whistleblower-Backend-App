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
			Title:   "Status Laporan Diperbarui",
			Message: fmt.Sprintf("Status laporan Anda berubah menjadi: %s", newStatus),
			Type:    "info",
		}
	}

	// Handle anonymous users
	if report.UserID == nil || *report.UserID == "" {
		fmt.Printf("[INFO] ⚠️ Report #%d tidak memiliki UserID, notifikasi dilewati\n", reportID)

		// Send email to anonymous reporter if email exists
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

	enhancedTitle := fmt.Sprintf("Laporan #%d - %s", reportID, statusInfo.Title)
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

	// 1️⃣ Kirim realtime via Pusher (untuk web app yang sedang aktif)
	if err := Client.Trigger(channel, event, payload); err != nil {
		fmt.Printf("[ERROR] ❌ Gagal kirim notifikasi Pusher ke %s: %v\n", channel, err)
	} else {
		fmt.Printf("[PUSHER] ✅ Notifikasi realtime terkirim ke %s\n", channel)
	}

	// 2️⃣ Simpan ke database
	notif := models.UserNotification{
		UserID:    *report.UserID,
		Title:     enhancedTitle,
		Message:   enhancedMessage,
		Type:      statusInfo.Type,
		ReportID:  &reportID,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&notif).Error; err != nil {
		fmt.Printf("[WARN] ⚠️ Gagal simpan notifikasi user ke DB: %v\n", err)
	} else {
		fmt.Printf("[DB] ✅ Notifikasi disimpan ke DB untuk user_id=%s (report #%d)\n", *report.UserID, reportID)
	}

	// 3️⃣ Kirim Push Notification via FCM (untuk mobile/background)
	fcmData := map[string]string{
		"report_id":  fmt.Sprintf("%d", reportID),
		"old_status": oldStatus,
		"new_status": newStatus,
		"type":       statusInfo.Type,
		"action":     "status_update",
	}

	if err := SendPushToUser(db, *report.UserID, enhancedTitle, enhancedMessage, statusInfo.Type, fcmData); err != nil {
		fmt.Printf("[FCM] ⚠️ Gagal kirim push notification: %v\n", err)
	} else {
		fmt.Printf("[FCM] ✅ Push notification terkirim untuk user_id=%s\n", *report.UserID)
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

	// 1. Kirim notifikasi realtime ke admin via Pusher
	if err := Client.Trigger(channel, event, data); err != nil {
		fmt.Printf("[ERROR] ❌ Gagal kirim notifikasi report baru: %v\n", err)
	}

	// 2. Simpan ke database
	notif := models.Notification{
		Title:     "🆕 Laporan Baru Masuk",
		Message:   fmt.Sprintf("Laporan baru: %s", title),
		Type:      "info",
		ReportID:  &reportID,
		CreatedAt: time.Now(),
	}
	if err := db.Create(&notif).Error; err != nil {
		fmt.Printf("[WARN] ⚠️ Gagal simpan notifikasi ke DB: %v\n", err)
	}

	// 3. Kirim Push Notification ke semua admin
	fcmData := map[string]string{
		"report_id": fmt.Sprintf("%d", reportID),
		"type":      "info",
		"action":    "new_report",
	}

	pushTitle := "🆕 Laporan Baru Masuk"
	pushMessage := fmt.Sprintf("Laporan baru: %s", title)

	if err := SendPushToAdmins(db, pushTitle, pushMessage, fcmData); err != nil {
		fmt.Printf("[FCM] ⚠️ Gagal kirim push ke admin: %v\n", err)
	} else {
		fmt.Printf("[FCM] ✅ Push notification terkirim ke admin\n")
	}

	fmt.Printf("[INFO] ✅ Admin di-notifikasi tentang report baru #%d\n", reportID)
	return nil
}


// ✅ FUNGSI TAMBAHAN: Notifikasi saat ada pesan chat baru
// ✅ FUNGSI TAMBAHAN: Notifikasi saat ada pesan chat baru
func NotifyNewChatMessage(db *gorm.DB, reportID uint, senderID, message string, isFromAdmin bool) error {
	var report models.Report
	if err := db.First(&report, reportID).Error; err != nil {
		return err
	}

	var channel string
	var title, notifMessage string
	var targetUserID string

	if isFromAdmin {
		// Notifikasi ke user (reporter)
		if report.UserID != nil && *report.UserID != "" {
			channel = fmt.Sprintf("user-%s", *report.UserID)
			targetUserID = *report.UserID
			title = "💬 Pesan Baru dari Admin"
			notifMessage = "Admin telah membalas laporan Anda"
		} else {
			// Skip untuk anonymous user
			fmt.Printf("[INFO] ℹ️ Report #%d adalah anonymous, skip notifikasi ke user\n", reportID)
			return nil
		}
	} else {
		// Notifikasi ke admin
		channel = "admin-notifications"
		title = "💬 Pesan Baru dari Reporter"
		notifMessage = fmt.Sprintf("Reporter mengirim pesan untuk laporan #%d", reportID)
	}

	event := "new-message"

	// Preview maksimal 50 karakter
	preview := message
	if len(message) > 50 {
		preview = message[:50] + "..."
	}

	data := map[string]any{
		"title":     title,
		"message":   notifMessage,
		"type":      "info",
		"report_id": reportID,
		"sender_id": senderID,
		"preview":   preview,
		"timestamp": time.Now().Unix(),
	}

	// 1️⃣ Kirim realtime via Pusher
	if err := Client.Trigger(channel, event, data); err != nil {
		fmt.Printf("[PUSHER ERROR] ❌ Gagal kirim notifikasi chat: %v\n", err)
	} else {
		fmt.Printf("[PUSHER] ✅ Notifikasi chat realtime terkirim ke %s\n", channel)
	}

	// 2️⃣ Simpan notifikasi ke database
	if isFromAdmin && targetUserID != "" {
		// Simpan ke UserNotification untuk user
		userNotif := models.UserNotification{
			UserID:    targetUserID,
			Title:     title,
			Message:   notifMessage,
			Type:      "info",
			ReportID:  &reportID,
			IsRead:    false,
			CreatedAt: time.Now(),
		}
		if err := db.Create(&userNotif).Error; err != nil {
			fmt.Printf("[DB WARN] ⚠️ Gagal simpan user notification: %v\n", err)
		} else {
			fmt.Printf("[DB] ✅ User notification disimpan untuk user %s\n", targetUserID)
		}
	} else if !isFromAdmin {
		// Simpan ke Notification untuk admin
		adminNotif := models.Notification{
			Title:     title,
			Message:   notifMessage,
			Type:      "info",
			ReportID:  &reportID,
			CreatedAt: time.Now(),
		}
		if err := db.Create(&adminNotif).Error; err != nil {
			fmt.Printf("[DB WARN] ⚠️ Gagal simpan admin notification: %v\n", err)
		} else {
			fmt.Printf("[DB] ✅ Admin notification disimpan\n")
		}
	}

	// 3️⃣ Kirim Push Notification via FCM
	fcmData := map[string]string{
		"report_id": fmt.Sprintf("%d", reportID),
		"sender_id": senderID,
		"type":      "info",
		"action":    "new_message",
		"preview":   preview,
	}

	if isFromAdmin && targetUserID != "" {
		// Push ke user
		if err := SendPushToUser(db, targetUserID, title, notifMessage, "info", fcmData); err != nil {
			fmt.Printf("[FCM] ⚠️ Gagal kirim push ke user: %v\n", err)
		} else {
			fmt.Printf("[FCM] ✅ Push notification terkirim ke user %s\n", targetUserID)
		}
	} else if !isFromAdmin {
		// Push ke admin
		if err := SendPushToAdmins(db, title, notifMessage, fcmData); err != nil {
			fmt.Printf("[FCM] ⚠️ Gagal kirim push ke admin: %v\n", err)
		} else {
			fmt.Printf("[FCM] ✅ Push notification terkirim ke admin\n")
		}
	}

	fmt.Printf("[INFO] ✅ Notifikasi chat lengkap terkirim untuk Report #%d\n", reportID)
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



// Tambahkan fungsi ini ke file: internal/notifications/handler.go

// NotifyAIAnalysisComplete sends notification when AI analysis is completed
func NotifyAIAnalysisComplete(db *gorm.DB, reportID uint, verdict string, confidence float64) error {
	var report models.Report
	if err := db.Preload("User").First(&report, reportID).Error; err != nil {
		fmt.Printf("[ERROR] Report #%d tidak ditemukan: %v\n", reportID, err)
		return err
	}

	verdictMessages := map[string]struct {
		Title   string
		Message string
		Type    string
		Emoji   string
	}{
		"verified": {
			Title:   "Laporan Terverifikasi",
			Message: "Laporan Anda telah diverifikasi oleh sistem AI sebagai valid dan kredibel",
			Type:    "success",
			Emoji:   "✅",
		},
		"hoax": {
			Title:   "Laporan Tidak Terverifikasi",
			Message: "Sistem AI mendeteksi laporan ini mengandung informasi yang tidak dapat diverifikasi",
			Type:    "warning",
			Emoji:   "⚠️",
		},
		"unconfirmed": {
			Title:   "Laporan Memerlukan Investigasi Lebih Lanjut",
			Message: "Laporan Anda memerlukan investigasi manual lebih lanjut oleh tim kami",
			Type:    "info",
			Emoji:   "🔍",
		},
	}

	verdictInfo, exists := verdictMessages[verdict]
	if !exists {
		verdictInfo = verdictMessages["unconfirmed"]
	}

	if report.UserID == nil || *report.UserID == "" {
		fmt.Printf("[INFO] ⚠️ Report #%d tidak memiliki UserID, notifikasi AI dilewati\n", reportID)

		// Send email to anonymous reporter
		if report.Email != nil && *report.Email != "" {
			subject := fmt.Sprintf("%s Hasil Analisis AI - Laporan #%d", verdictInfo.Emoji, reportID)
			body := fmt.Sprintf(`
				<h2>%s</h2>
				<p>%s</p>
				<hr>
				<p><strong>ID Laporan:</strong> #%d</p>
				<p><strong>Tingkat Keyakinan AI:</strong> %.1f%%</p>
				<p><strong>Hasil Verifikasi:</strong> %s</p>
				<p style="color: gray; font-size: 12px;">Email ini dikirim otomatis oleh sistem Whistleblower AI.</p>
			`, verdictInfo.Title, verdictInfo.Message, reportID, confidence, verdict)

			if err := utils.SendEmail(*report.Email, subject, body); err != nil {
				fmt.Printf("[EMAIL ERROR] gagal kirim ke %s: %v\n", *report.Email, err)
			}
		}

		return nil
	}

	channel := fmt.Sprintf("user-%s", *report.UserID)
	event := "ai-analysis-complete"

	enhancedTitle := fmt.Sprintf("%s Laporan #%d - %s", verdictInfo.Emoji, reportID, verdictInfo.Title)
	enhancedMessage := fmt.Sprintf("%s (Tingkat keyakinan AI: %.1f%%)", verdictInfo.Message, confidence)

	payload := map[string]any{
		"title":      enhancedTitle,
		"message":    enhancedMessage,
		"type":       verdictInfo.Type,
		"report_id":  reportID,
		"verdict":    verdict,
		"confidence": confidence,
		"timestamp":  time.Now().Unix(),
	}

	// Kirim realtime via Pusher
	if err := Client.Trigger(channel, event, payload); err != nil {
		fmt.Printf("[ERROR] ❌ Gagal kirim notifikasi AI ke %s: %v\n", channel, err)
	}

	// Simpan ke database
	notif := models.UserNotification{
		UserID:    *report.UserID,
		Title:     enhancedTitle,
		Message:   enhancedMessage,
		Type:      verdictInfo.Type,
		ReportID:  &reportID,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&notif).Error; err != nil {
		fmt.Printf("[WARN] ⚠️ Gagal simpan notifikasi AI ke DB: %v\n", err)
	}

	// Kirim Push Notification
	fcmData := map[string]string{
		"report_id":  fmt.Sprintf("%d", reportID),
		"verdict":    verdict,
		"confidence": fmt.Sprintf("%.1f", confidence),
		"type":       verdictInfo.Type,
		"action":     "ai_analysis",
	}

	if err := SendPushToUser(db, *report.UserID, enhancedTitle, enhancedMessage, verdictInfo.Type, fcmData); err != nil {
		fmt.Printf("[FCM] ⚠️ Gagal kirim push notification AI: %v\n", err)
	} else {
		fmt.Printf("[FCM] ✅ Push notification AI terkirim untuk user_id=%s\n", *report.UserID)
	}

	// Notify admins
	go notifyAdminsAboutAIResult(db, reportID, report.Title, verdict, confidence)

	return nil
}

// notifyAdminsAboutAIResult sends notification to admin channel about AI analysis result
func notifyAdminsAboutAIResult(db *gorm.DB, reportID uint, title, verdict string, confidence float64) {
	channel := "admin-notifications"
	event := "ai-analysis-result"

	var emoji string
	var message string

	switch verdict {
	case "verified":
		emoji = "✅"
		message = fmt.Sprintf("AI memverifikasi laporan: %s", title)
	case "hoax":
		emoji = "⚠️"
		message = fmt.Sprintf("AI mendeteksi hoax: %s", title)
	default:
		emoji = "🔍"
		message = fmt.Sprintf("AI memerlukan review manual: %s", title)
	}

	data := map[string]any{
		"title":      fmt.Sprintf("%s Hasil Analisis AI - Laporan #%d", emoji, reportID),
		"message":    message,
		"type":       "info",
		"report_id":  reportID,
		"verdict":    verdict,
		"confidence": confidence,
		"timestamp":  time.Now().Unix(),
	}

	if err := Client.Trigger(channel, event, data); err != nil {
		fmt.Printf("[ERROR] ❌ Gagal kirim notifikasi AI ke admin: %v\n", err)
		return
	}

	// Save to admin notifications table
	notif := models.Notification{
		Title:     fmt.Sprintf("%s Hasil Analisis AI", emoji),
		Message:   message,
		Type:      "info",
		ReportID:  &reportID,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&notif).Error; err != nil {
		fmt.Printf("[WARN] ⚠️ Gagal simpan notifikasi AI admin ke DB: %v\n", err)
	}

	fmt.Printf("[INFO] ✅ Admin notified tentang AI result #%d (verdict: %s)\n", reportID, verdict)
}

// Notify Admin untuk chat agent
// NotifyChatAgentHandoff notifies admin when a user requests human support
func NotifyChatAgentHandoff(userID string, message string) error {

	channel := "admin-notifications"
	event := "chatagent-handoff"

	// batasi preview max 80 char
	preview := message
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}

	payload := map[string]any{
		"title":     "👤 Pengguna Meminta Bantuan Admin",
		"message":   fmt.Sprintf("User meminta bantuan CS: \"%s\"", preview),
		"type":      "info",
		"user_id":   userID,
		"source":    "chat_agent",
		"timestamp": time.Now().Unix(),
	}

	// Kirim realtime ke admin
	if err := Client.Trigger(channel, event, payload); err != nil {
		fmt.Printf("[ERROR] ❌ Gagal kirim notifikasi Chat Agent ke admin: %v\n", err)
		return err
	}

	fmt.Printf("[CHAT AGENT] ⚡ Admin diberi tahu bahwa user %s meminta bantuan\n", userID)
	return nil
}
