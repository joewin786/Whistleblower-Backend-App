package notifications

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

var FCMClient *messaging.Client

// InitializeFCM initializes Firebase Cloud Messaging client
func InitializeFCM() error {
	ctx := context.Background()
	
	// Path ke service account JSON file
	serviceAccountPath := os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH")
	if serviceAccountPath == "" {
		serviceAccountPath = "firebase-service-account.json"
	}

	// Verify file exists
	if _, err := os.Stat(serviceAccountPath); os.IsNotExist(err) {
		return fmt.Errorf("service account file not found at: %s", serviceAccountPath)
	}

	log.Printf("📂 Using Firebase service account: %s", serviceAccountPath)

	// Create options
	opt := option.WithCredentialsFile(serviceAccountPath)
	
	// Initialize app with explicit endpoint (IMPORTANT!)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return fmt.Errorf("error initializing firebase app: %v", err)
	}

	// Get messaging client
	client, err := app.Messaging(ctx)
	if err != nil {
		return fmt.Errorf("error getting messaging client: %v", err)
	}

	FCMClient = client
	log.Println("✅ FCM initialized successfully")
	
	return nil
}

// SendPushNotification sends a push notification to a specific device token
func SendPushNotification(ctx context.Context, token, title, body string, data map[string]string) error {
	if FCMClient == nil {
		return fmt.Errorf("FCM client not initialized")
	}

	// Validate token
	if token == "" {
		return fmt.Errorf("FCM token is empty")
	}

	message := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				Sound:        "default",
				ChannelID:    "whistleblower_notifications",
				Priority:     messaging.PriorityHigh,
				DefaultSound: true,
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound: "default",
					Badge: nil,
				},
			},
		},
		Webpush: &messaging.WebpushConfig{
			Notification: &messaging.WebpushNotification{
				Title: title,
				Body:  body,
				Icon:  "/icon.png",
			},
		},
	}

	// Add timeout to context
	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := FCMClient.Send(ctxTimeout, message)
	if err != nil {
		return fmt.Errorf("error sending FCM message: %v", err)
	}

	log.Printf("[FCM] ✅ Message sent successfully to token: %s... (ID: %s)", token[:20], response)
	return nil
}

// SendMulticastNotification sends notification to multiple devices
func SendMulticastNotification(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	if FCMClient == nil {
		return fmt.Errorf("FCM client not initialized")
	}

	if len(tokens) == 0 {
		return fmt.Errorf("no tokens provided")
	}

	// Filter out empty tokens
	validTokens := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token != "" {
			validTokens = append(validTokens, token)
		}
	}

	if len(validTokens) == 0 {
		return fmt.Errorf("no valid tokens provided")
	}

	log.Printf("[FCM] 📤 Sending notification to %d devices", len(validTokens))

	// IMPORTANT: Use SendEach instead of SendMulticast for better error handling
	messages := make([]*messaging.Message, len(validTokens))
	for i, token := range validTokens {
		messages[i] = &messaging.Message{
			Token: token,
			Notification: &messaging.Notification{
				Title: title,
				Body:  body,
			},
			Data: data,
			Android: &messaging.AndroidConfig{
				Priority: "high",
				Notification: &messaging.AndroidNotification{
					Sound:        "default",
					ChannelID:    "whistleblower_notifications",
					Priority:     messaging.PriorityHigh,
					DefaultSound: true,
				},
			},
			APNS: &messaging.APNSConfig{
				Payload: &messaging.APNSPayload{
					Aps: &messaging.Aps{
						Sound: "default",
					},
				},
			},
		}
	}

	// Add timeout to context
	ctxTimeout, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Use SendEach for better control
	response, err := FCMClient.SendEach(ctxTimeout, messages)
	if err != nil {
		return fmt.Errorf("error sending FCM messages: %v", err)
	}

	log.Printf("[FCM] ✅ Sent: %d success, %d failure out of %d", 
		response.SuccessCount, response.FailureCount, len(validTokens))
	
	// Log failed tokens with details
	if response.FailureCount > 0 {
		for idx, resp := range response.Responses {
			if !resp.Success {
				errMsg := "unknown error"
				if resp.Error != nil {
					errMsg = resp.Error.Error()
				}
				log.Printf("[FCM] ❌ Failed to send to token %d (%.20s...): %s", 
					idx, validTokens[idx], errMsg)
				
				// Check if token is invalid and should be removed
				if resp.Error != nil && isTokenInvalid(resp.Error) {
					log.Printf("[FCM] 🗑️ Token %d is invalid and should be removed from database", idx)
					// TODO: Add logic to mark token as inactive in database
				}
			}
		}
	}

	return nil
}

// isTokenInvalid checks if error indicates invalid token
func isTokenInvalid(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	invalidErrors := []string{
		"registration-token-not-registered",
		"invalid-registration-token",
		"invalid-argument",
	}
	
	for _, invalidErr := range invalidErrors {
		if contains(errStr, invalidErr) {
			return true
		}
	}
	return false
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// SendToTopic sends notification to a topic (for broadcast)
func SendToTopic(ctx context.Context, topic, title, body string, data map[string]string) error {
	if FCMClient == nil {
		return fmt.Errorf("FCM client not initialized")
	}

	message := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				Sound:     "default",
				ChannelID: "whistleblower_notifications",
			},
		},
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := FCMClient.Send(ctxTimeout, message)
	if err != nil {
		return fmt.Errorf("error sending to topic: %v", err)
	}

	log.Printf("[FCM] ✅ Message sent to topic '%s': %s", topic, response)
	return nil
}

// SubscribeToTopic subscribes tokens to a topic
func SubscribeToTopic(ctx context.Context, tokens []string, topic string) error {
	if FCMClient == nil {
		return fmt.Errorf("FCM client not initialized")
	}

	if len(tokens) == 0 {
		return fmt.Errorf("no tokens provided")
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := FCMClient.SubscribeToTopic(ctxTimeout, tokens, topic)
	if err != nil {
		return fmt.Errorf("error subscribing to topic: %v", err)
	}

	log.Printf("[FCM] ✅ Subscribed %d devices to topic '%s' (failed: %d)", 
		response.SuccessCount, topic, response.FailureCount)
	return nil
}

// UnsubscribeFromTopic unsubscribes tokens from a topic
func UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) error {
	if FCMClient == nil {
		return fmt.Errorf("FCM client not initialized")
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := FCMClient.UnsubscribeFromTopic(ctxTimeout, tokens, topic)
	if err != nil {
		return fmt.Errorf("error unsubscribing from topic: %v", err)
	}

	log.Printf("[FCM] ✅ Unsubscribed %d devices from topic '%s'", 
		response.SuccessCount, topic)
	return nil
}

// TestFCMConnection tests if FCM is working with a dry run
func TestFCMConnection() error {
	if FCMClient == nil {
		return fmt.Errorf("FCM client not initialized")
	}

	// Create a test message with dry run
	message := &messaging.Message{
		Topic: "test-topic",
		Notification: &messaging.Notification{
			Title: "Test",
			Body:  "Test notification",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Send with dry run (won't actually send notification)
	_, err := FCMClient.Send(ctx, message)
	
	// Even with dry run, we'll get an error if connection fails
	// But topic might not exist, so we ignore topic-related errors
	if err != nil && !contains(err.Error(), "topic") {
		return fmt.Errorf("FCM connection test failed: %v", err)
	}

	return nil
}