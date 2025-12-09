package auth

import (
	"context"
	"net/http"
	"strings"
	"whistleblower_REST/internal/utils"
)

func AdminAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			var token string

			// 1️⃣ Coba ambil dari query param (untuk WebSocket)
			token = r.URL.Query().Get("token")

			// 2️⃣ Kalau kosong, ambil dari Authorization: Bearer xxx
			if token == "" {
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					utils.RespondWithError(w, http.StatusUnauthorized, "missing admin token")
					return
				}
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}

			// 3️⃣ Validate admin JWT
			adminID, role, err := ValidateAdminToken(token)
			if err != nil {
				utils.RespondWithError(w, http.StatusUnauthorized, "invalid admin token")
				return
			}

			// 4️⃣ Inject context
			ctx := context.WithValue(r.Context(), contextKeyAdminID, adminID)
			ctx = context.WithValue(ctx, contextKeyRole, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
