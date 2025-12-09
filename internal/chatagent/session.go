package chatagent

import (
	"fmt"
	"sync"
	"time"
	"whistleblower_REST/internal/notifications"
)

type Session struct {
	UserID       string
	ConnectedAt  time.Time
	LastActivity time.Time
	IsActive     bool
}

var (
	sessions = make(map[string]*Session)
	mu       sync.RWMutex
)

// Buat session baru saat handoff
func CreateSession(userID string) {
	mu.Lock()
	defer mu.Unlock()

	sessions[userID] = &Session{
		UserID:       userID,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		IsActive:     true,
	}
}

// Update last activity
func UpdateActivity(userID string) {
	mu.Lock()
	defer mu.Unlock()

	if session, exists := sessions[userID]; exists {
		session.LastActivity = time.Now()
	}
}

// Check apakah user sedang terhubung dengan admin
func IsConnectedToAdmin(userID string) bool {
	mu.RLock()
	defer mu.RUnlock()

	session, exists := sessions[userID]
	return exists && session.IsActive
}

// End session
func EndSession(userID string) {
	mu.Lock()
	defer mu.Unlock()

	if session, exists := sessions[userID]; exists {
		session.IsActive = false
	}
}

// Auto cleanup inactive sessions (run in goroutine)
func StartSessionCleaner() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			mu.Lock()
			for userID, session := range sessions {
				// Jika tidak ada aktivitas selama 3 menit
				if session.IsActive && time.Since(session.LastActivity) > 3*time.Minute {
					session.IsActive = false

					// Notify admin bahwa user disconnected
					notifications.NotifyChatAgentDisconnect(userID)

					fmt.Printf("⏰ Session timeout for user: %s\n", userID)
				}
			}
			mu.Unlock()
		}
	}()
}
