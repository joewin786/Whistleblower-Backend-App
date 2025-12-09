package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"whistleblower_REST/internal/auth"
	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// RegisterDeviceRequest represents device registration payload
type RegisterDeviceRequest struct {
	FCMToken   string `json:"fcm_token"`
	DeviceType string `json:"device_type"` // "android", "ios", "web"
	DeviceName string `json:"device_name,omitempty"`
}

// RegisterDevice registers a user's device for push notifications
func RegisterDevice(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := auth.GetIDFromContext(r.Context())
		if !ok || uid == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "missing user id in token")
			return
		}

		var req RegisterDeviceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.FCMToken == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "fcm_token is required")
			return
		}

		// Check if device already exists
		var device models.UserDevice
		result := db.Where("fcm_token = ?", req.FCMToken).First(&device)

		if result.Error == gorm.ErrRecordNotFound {
			// Create new device
			device = models.UserDevice{
				UserID:       uid,
				FCMToken:     req.FCMToken,
				DeviceType:   req.DeviceType,
				DeviceName:   req.DeviceName,
				IsActive:     true,
				LastActiveAt: time.Now(),
			}
			if err := db.Create(&device).Error; err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, "failed to register device")
				return
			}
			fmt.Printf("[FCM] ✅ New device registered for user %s\n", uid)
		} else {
			// Update existing device
			device.UserID = uid
			device.DeviceType = req.DeviceType
			device.DeviceName = req.DeviceName
			device.IsActive = true
			device.LastActiveAt = time.Now()
			if err := db.Save(&device).Error; err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, "failed to update device")
				return
			}
			fmt.Printf("[FCM] ✅ Device updated for user %s\n", uid)
		}

		// Subscribe to user-specific topic
		ctx := context.Background()
		topic := fmt.Sprintf("user-%s", uid)
		if err := SubscribeToTopic(ctx, []string{req.FCMToken}, topic); err != nil {
			fmt.Printf("[FCM] ⚠️ Failed to subscribe to topic: %v\n", err)
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"message": "device registered successfully",
			"device":  device,
		})
	}
}

// UnregisterDevice removes a device token
func UnregisterDevice(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := auth.GetIDFromContext(r.Context())
		if !ok || uid == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "missing user id in token")
			return
		}

		var req struct {
			FCMToken string `json:"fcm_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Soft delete - mark as inactive
		result := db.Model(&models.UserDevice{}).
			Where("user_id = ? AND fcm_token = ?", uid, req.FCMToken).
			Update("is_active", false)

		if result.Error != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to unregister device")
			return
		}

		fmt.Printf("[FCM] ✅ Device unregistered for user %s\n", uid)
		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "device unregistered successfully",
		})
	}
}

// GetUserDevices returns all registered devices for a user
func GetUserDevices(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := auth.GetIDFromContext(r.Context())
		if !ok || uid == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "missing user id in token")
			return
		}

		var devices []models.UserDevice
		if err := db.Where("user_id = ? AND is_active = ?", uid, true).
			Order("last_active_at DESC").
			Find(&devices).Error; err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to fetch devices")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, devices)
	}
}

// SendPushToUser sends push notification to all user's devices
func SendPushToUser(db *gorm.DB, userID, title, message, notifType string, data map[string]string) error {
	ctx := context.Background()

	// Get all active devices for user
	var devices []models.UserDevice
	if err := db.Where("user_id = ? AND is_active = ?", userID, true).
		Find(&devices).Error; err != nil {
		return fmt.Errorf("failed to get user devices: %v", err)
	}

	if len(devices) == 0 {
		fmt.Printf("[FCM] ℹ️ No active devices for user %s\n", userID)
		return nil
	}

	// Collect all FCM tokens
	tokens := make([]string, 0, len(devices))
	for _, device := range devices {
		tokens = append(tokens, device.FCMToken)
	}

	// Prepare data payload
	if data == nil {
		data = make(map[string]string)
	}
	data["user_id"] = userID
	data["type"] = notifType

	// Send multicast notification
	if err := SendMulticastNotification(ctx, tokens, title, message, data); err != nil {
		return fmt.Errorf("failed to send push notification: %v", err)
	}

	fmt.Printf("[FCM] ✅ Push notification sent to %d devices for user %s\n", len(tokens), userID)
	return nil
}

// SendPushToAdmins sends notification to all admin devices
func SendPushToAdmins(db *gorm.DB, title, message string, data map[string]string) error {
	ctx := context.Background()

	// Get all admin users (assuming you have a role field)
	var adminUsers []models.User
	if err := db.Where("role = ?", "admin").Find(&adminUsers).Error; err != nil {
		return fmt.Errorf("failed to get admin users: %v", err)
	}

	if len(adminUsers) == 0 {
		fmt.Printf("[FCM] ℹ️ No admin users found\n")
		return nil
	}

	// Get admin user IDs
	adminIDs := make([]string, 0, len(adminUsers))
	for _, admin := range adminUsers {
		adminIDs = append(adminIDs, admin.ID)
	}

	// Get all active devices for admins
	var devices []models.UserDevice
	if err := db.Where("user_id IN ? AND is_active = ?", adminIDs, true).
		Find(&devices).Error; err != nil {
		return fmt.Errorf("failed to get admin devices: %v", err)
	}

	if len(devices) == 0 {
		fmt.Printf("[FCM] ℹ️ No active devices for admins\n")
		return nil
	}

	// Collect tokens
	tokens := make([]string, 0, len(devices))
	for _, device := range devices {
		tokens = append(tokens, device.FCMToken)
	}

	// Prepare data
	if data == nil {
		data = make(map[string]string)
	}
	data["target"] = "admin"

	// Send notification
	if err := SendMulticastNotification(ctx, tokens, title, message, data); err != nil {
		return fmt.Errorf("failed to send admin notification: %v", err)
	}

	fmt.Printf("[FCM] ✅ Push notification sent to %d admin devices\n", len(tokens))
	return nil
}

// TestPushNotification sends a test notification (for debugging)
func TestPushNotification(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := auth.GetIDFromContext(r.Context())
		if !ok || uid == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "missing user id in token")
			return
		}

		var req struct {
			Title   string            `json:"title"`
			Message string            `json:"message"`
			Data    map[string]string `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Title == "" {
			req.Title = "Test Notification"
		}
		if req.Message == "" {
			req.Message = "This is a test push notification"
		}

		if err := SendPushToUser(db, uid, req.Title, req.Message, "test", req.Data); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "test notification sent",
		})
	}
}

// DeleteDevice permanently removes a device
func DeleteDevice(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := auth.GetIDFromContext(r.Context())
		if !ok || uid == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "missing user id in token")
			return
		}

		deviceID := chi.URLParam(r, "deviceId")
		if deviceID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "missing device id")
			return
		}

		result := db.Where("id = ? AND user_id = ?", deviceID, uid).
			Delete(&models.UserDevice{})

		if result.Error != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to delete device")
			return
		}

		if result.RowsAffected == 0 {
			utils.RespondWithError(w, http.StatusNotFound, "device not found")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "device deleted successfully",
		})
	}
}
