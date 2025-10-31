package auth

import (
	"context"
	"net/http"
	"strings"
	"whistleblower_REST/internal/utils"
)

// AuthMiddleware validates JWT and injects user id and role into request context
// Supports token from Authorization header (mobile) OR query param (browser WebSocket)
func AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string

			// ✅ Try to get token from query parameter first (for browser WebSocket)
			token = r.URL.Query().Get("token")

			// ✅ Fallback to Authorization header (for regular HTTP & mobile WebSocket)
			if token == "" {
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					utils.RespondWithError(w, http.StatusUnauthorized, "missing token")
					return
				}
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}

			// Validate token
			id, role, err := ValidateToken(token)
			if err != nil {
				utils.RespondWithError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			// Inject user data into context
			ctx := context.WithValue(r.Context(), "id", id)
			ctx = context.WithValue(ctx, "role", role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware injects user id and role when a valid token is present; otherwise passes through
func OptionalAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string

			// Try query parameter first
			token = r.URL.Query().Get("token")

			// Fallback to Authorization header
			if token == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.TrimSpace(authHeader) == "" {
					// No token provided: proceed anonymously
					next.ServeHTTP(w, r)
					return
				}
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}

			// Token provided: validate
			id, role, err := ValidateToken(token)
			if err != nil {
				utils.RespondWithError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), "id", id)
			ctx = context.WithValue(ctx, "role", role)
			next.ServeHTTP(w, r.WithContext(ctx))
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
			role, _ := r.Context().Value("role").(string)
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