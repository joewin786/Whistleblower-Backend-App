package auth

import (
	"context"
	"net/http"
	"strings"
	"whistleblower_REST/internal/utils"
)

func AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.RespondWithError(w, http.StatusUnauthorized, "missing token")
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			id, err := ValidateToken(token)
			if err != nil {
				utils.RespondWithError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), "id", id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
