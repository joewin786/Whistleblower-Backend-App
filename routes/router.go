package routes

import (
	"net/http"
	"whistleblower_REST/internal/admin"
	"whistleblower_REST/internal/analytics"
	"whistleblower_REST/internal/auth"
	"whistleblower_REST/internal/evidence"
	"whistleblower_REST/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"

	"whistleblower_REST/internal/messages"
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
	messageHandler := messages.NewHandler(db)
	analyticsHandler := &analytics.AnalyticsHandler{DB: db}

	// ===== AUTH ROUTES =====
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/validate", authHandler.ValidateToken)
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

		// ==== USERS ====
		r.Route("/users", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(auth.RoleMiddleware("admin"))
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
			r.Get("/{reportId}", reportHandler.GetByID)
			r.Get("/{reportId}/messages", messageHandler.GetByReportID)
			r.Get("/categories", categoryHandler.GetAllCategories)
			r.With(auth.OptionalAuthMiddleware()).Post("/", reportHandler.Create) // publik bisa kirim laporan (optional auth)
			r.Put("/{reportId}/assign", reportHandler.AssignAdmin)                // publik bisa assign admin (dengan email wajib)

			// --- Protected routes (login required) ---
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware)
				r.Get("/my", reportHandler.GetMy)             // laporan user login
				r.Patch("/{reportId}", reportHandler.Update)  // update status
				r.Delete("/{reportId}", reportHandler.Delete) // hapus laporan
			})

			// === EVIDENCE nested routes ===
			r.Route("/{reportId}/evidence", func(r chi.Router) {
				r.Get("/", evidenceHandler.GetByReport)
				r.Post("/", evidenceHandler.Create)
				r.Delete("/{evidenceId}", evidenceHandler.Delete)
			})

			// === MESSAGES nested routes ===
			r.Route("/{reportId}/messages", func(r chi.Router) {

				r.Group(func(r chi.Router) {
					r.Post("/", messageHandler.Create)
				})
			})
		})

		// ==== ANALYTICS ====
		r.Route("/analytics", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(auth.RoleMiddleware("admin"))
			r.Get("/overview", analyticsHandler.GetOverview)
			r.Get("/trends", analyticsHandler.GetTrends)
			r.Get("/by-categories", analyticsHandler.GetByCategories)
			r.Get("/by-status", analyticsHandler.GetByStatus)
			r.Get("/\t", analyticsHandler.GetInvestigatorPerformance)
			r.Post("/reports/generate", analyticsHandler.GenerateReport)
		})

		// ==== ADMIN CONFIG ====
		r.Route("/admin/config", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(auth.RoleMiddleware("admin"))
			// Removed GET /categories here to keep it public above
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
