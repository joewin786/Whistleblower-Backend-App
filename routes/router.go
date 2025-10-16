package routes

import (
	"net/http"
	"whistleblower_REST/internal/admin"
	"whistleblower_REST/internal/auth"
	"whistleblower_REST/internal/evidence"
	"whistleblower_REST/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"

	"whistleblower_REST/internal/reports"
)

func RegisterRoutes(db *gorm.DB) *chi.Mux {
	r := chi.NewRouter()

	// === Global middleware ===
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	authMiddleware := auth.AuthMiddleware()

	authHandler := &auth.AuthHandler{DB: db}
	reportHandler := reports.NewHandler(db)
	evidenceHandler := evidence.NewHandler(db)
	categoryHandler := &admin.CategoryHandler{DB: db}

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
			r.Post("/", reportHandler.Create)
			r.Get("/{reportId}", reportHandler.GetByID)

			r.Group(func(r chi.Router) {
				r.Use(authMiddleware)
				r.Get("/my", reportHandler.GetMy)
				r.Patch("/{reportId}", reportHandler.Update)
				r.Delete("/{reportId}", reportHandler.Delete)
			})

			// === EVIDENCE nested routes ===
			r.Route("/{reportId}/evidence", func(r chi.Router) {
				r.Get("/", evidenceHandler.GetByReport)
				r.Post("/", evidenceHandler.Create)
				r.Delete("/{evidenceId}", evidenceHandler.Delete)
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
			r.Get("/categories", categoryHandler.GetAllCategories)
			r.Post("/categories", categoryHandler.CreateCategory)
			r.Patch("/categories/{catId}", categoryHandler.UpdateCategory)
			r.Delete("/categories/{catId}", categoryHandler.DeleteCategory)

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
