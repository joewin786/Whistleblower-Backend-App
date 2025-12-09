package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"whistleblower_REST/internal/utils"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	contextKeyID      contextKey = "id"
	contextKeyRole    contextKey = "role"
	contextKeyAdminID contextKey = "admin_id"
)

// Helper functions to retrieve values from context
// These should be used instead of directly accessing context values

// GetIDFromContext retrieves the user/admin ID from context (as string)
func GetIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(contextKeyID).(string)
	return id, ok
}

// GetRoleFromContext retrieves the role from context
func GetRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(contextKeyRole).(string)
	return role, ok
}

// GetAdminIDFromContext retrieves the admin ID from context (as uint)
func GetAdminIDFromContext(ctx context.Context) (uint, bool) {
	adminID, ok := ctx.Value(contextKeyAdminID).(uint)
	return adminID, ok
}

// AuthMiddleware validates JWT and injects user/admin id and role into request context
// Supports both user tokens (string id) and admin tokens (uint id)
func AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string

			// Try to get token from query parameter first (for browser WebSocket)
			token = r.URL.Query().Get("token")

			// Fallback to Authorization header
			if token == "" {
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					utils.RespondWithError(w, http.StatusUnauthorized, "missing token")
					return
				}
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}

			// ✅ Try validate as USER token first (string id)
			id, role, err := ValidateToken(token)
			if err == nil {
				// Success - it's a user token
				ctx := context.WithValue(r.Context(), contextKeyID, id)
				ctx = context.WithValue(ctx, contextKeyRole, role)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// ✅ If user token fails, try validate as ADMIN token (uint id)
			adminID, adminRole, adminErr := ValidateAdminToken(token)
			if adminErr == nil {
				// Success - it's an admin token
				// Convert uint to string for compatibility
				ctx := context.WithValue(r.Context(), contextKeyID, fmt.Sprintf("%d", adminID))
				ctx = context.WithValue(ctx, contextKeyRole, adminRole)
				ctx = context.WithValue(ctx, contextKeyAdminID, adminID) // ✅ Store original uint
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Both validations failed
			utils.RespondWithError(w, http.StatusUnauthorized, "invalid token")
		})
	}
}

// OptionalAuthMiddleware injects user/admin id and role when a valid token is present
func OptionalAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string

			token = r.URL.Query().Get("token")
			if token == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.TrimSpace(authHeader) == "" {
					next.ServeHTTP(w, r)
					return
				}
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}

			// Try validate as user token
			id, role, err := ValidateToken(token)
			if err == nil {
				ctx := context.WithValue(r.Context(), contextKeyID, id)
				ctx = context.WithValue(ctx, contextKeyRole, role)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Try validate as admin token
			adminID, adminRole, adminErr := ValidateAdminToken(token)
			if adminErr == nil {
				ctx := context.WithValue(r.Context(), contextKeyID, fmt.Sprintf("%d", adminID))
				ctx = context.WithValue(ctx, contextKeyRole, adminRole)
				ctx = context.WithValue(ctx, contextKeyAdminID, adminID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Optional - proceed anyway even if token invalid
			utils.RespondWithError(w, http.StatusUnauthorized, "invalid token")
		})
	}
}

// RoleMiddleware restricts access to users with one of the allowed roles
func RoleMiddleware(allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(contextKeyRole).(string)
			if role == "" {
				utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			if _, ok := allowed[role]; !ok {
				utils.RespondWithError(w, http.StatusForbidden, "forbidden: insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func AdminOnlyMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(contextKeyRole).(string)
			if !ok || (role != "admin" && role != "superadmin") {
				utils.RespondWithError(w, http.StatusForbidden, "admin access only")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func SuperAdminOnlyMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(contextKeyRole).(string)
			if !ok || role != "superadmin" {
				utils.RespondWithError(w, http.StatusForbidden, "superadmin access only")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
