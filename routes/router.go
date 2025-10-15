package routes

import (
	"net/http"
	"whistleblower_REST/internal/auth"
	"whistleblower_REST/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"
)

func RegisterRoutes(db *gorm.DB) *chi.Mux {
	r := chi.NewRouter()

	// === Global middleware ===
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	authHandler := &auth.AuthHandler{DB: db}
	authMiddleware := auth.AuthMiddleware()

	// ===== AUTH ROUTES =====
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", func(w http.ResponseWriter, r *http.Request) {
			utils.RespondWithJSON(w, 200, map[string]string{"message": "logout"})
		})
		r.Post("/reset-password", func(w http.ResponseWriter, r *http.Request) {
			utils.RespondWithJSON(w, 200, map[string]string{"message": "reset password"})
		})

		// protected: /auth/me
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/me", authHandler.Me)
		})
	})

	// ===== PROTECTED ROUTES =====
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)

		// ==== USERS ====
		r.Route("/users", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "get all users"})
			})
			r.Get("/{userId}", func(w http.ResponseWriter, r *http.Request) {
				userId := chi.URLParam(r, "userId")
				utils.RespondWithJSON(w, 200, map[string]string{"message": "get user by id", "userId": userId})
			})
			r.Patch("/{userId}", func(w http.ResponseWriter, r *http.Request) {
				userId := chi.URLParam(r, "userId")
				utils.RespondWithJSON(w, 200, map[string]string{"message": "update user", "userId": userId})
			})
		})

		// ==== REPORTS ====
		r.Route("/reports", func(r chi.Router) {
			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "create func(w http.ResponseWriter, r *http.Request) {report"})
			})
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "get all reports"})
			})
			r.Get("/my", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "get my reports"})
			})
			r.Get("/{reportId}", func(w http.ResponseWriter, r *http.Request) {
				reportId := chi.URLParam(r, "reportId")
				utils.RespondWithJSON(w, 200, map[string]string{"message": "get report by id", "reportId": reportId})
			})
			r.Patch("/{reportId}", func(w http.ResponseWriter, r *http.Request) {
				reportId := chi.URLParam(r, "reportId")
				utils.RespondWithJSON(w, 200, map[string]string{"message": "update report", "reportId": reportId})
			})
			r.Delete("/{reportId}", func(w http.ResponseWriter, r *http.Request) {
				reportId := chi.URLParam(r, "reportId")
				utils.RespondWithJSON(w, 200, map[string]string{"message": "delete report", "reportId": reportId})
			})

			// === EVIDENCE nested routes ===
			r.Route("/{reportId}/evidence", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					utils.RespondWithJSON(w, 200, map[string]string{"message": "get evidence list"})
				})
				r.Post("/", func(w http.ResponseWriter, r *http.Request) {
					utils.RespondWithJSON(w, 200, map[string]string{"message": "upload evidence"})
				})
				r.Get("/{evidenceId}", func(w http.ResponseWriter, r *http.Request) {
					utils.RespondWithJSON(w, 200, map[string]string{"message": "get evidence by id"})
				})
				r.Delete("/{evidenceId}", func(w http.ResponseWriter, r *http.Request) {
					utils.RespondWithJSON(w, 200, map[string]string{"message": "delete evidence"})
				})
			})

			// === MESSAGES nested routes ===
			r.Route("/{reportId}/messages", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					utils.RespondWithJSON(w, 200, map[string]string{"message": "get messages"})
				})
				r.Post("/", func(w http.ResponseWriter, r *http.Request) {
					utils.RespondWithJSON(w, 200, map[string]string{"message": "create message"})
				})
			})
		})

		// ==== ANALYTICS ====
		r.Route("/analytics", func(r chi.Router) {
			r.Get("/overview", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "analytics overview"})
			})
			r.Get("/trends", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "analytics trends"})
			})
			r.Get("/by-categories", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "analytics by categories"})
			})
			r.Get("/by-status", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "analytics by status"})
			})
			r.Get("/investigator-performance", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "analytics performance"})
			})
			r.Post("/reports/generate", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "generate report"})
			})
		})

		// ==== ADMIN CONFIG ====
		r.Route("/admin/config", func(r chi.Router) {
			r.Get("/categories", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "get categories"})
			})
			r.Post("/categories", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "create category"})
			})
			r.Patch("/categories/{catId}", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "update category"})
			})
			r.Delete("/categories/{catId}", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "delete category"})
			})

			r.Get("/roles", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "get roles"})
			})
			r.Post("/roles", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "create role"})
			})
			r.Patch("/roles/{roleId}", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "update role"})
			})

			r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "get settings"})
			})
			r.Put("/settings", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "update settings"})
			})

			r.Get("/workflows", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "get workflows"})
			})
			r.Put("/workflows", func(w http.ResponseWriter, r *http.Request) {
				utils.RespondWithJSON(w, 200, map[string]string{"message": "update workflows"})
			})
		})
	})

	// ===== HEALTH CHECK =====
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		utils.RespondWithJSON(w, 200, map[string]string{"status": "ok"})
	})

	return r
}
