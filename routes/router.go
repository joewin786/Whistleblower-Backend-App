package routes

import (
	"net/http"
	"whistleblower_REST/internal/admin"
	"whistleblower_REST/internal/analytics"
	"whistleblower_REST/internal/auth"
	"whistleblower_REST/internal/evidence"
	"whistleblower_REST/internal/utils"
	"whistleblower_REST/internal/websocket"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"

	"whistleblower_REST/internal/messages"
	"whistleblower_REST/internal/notifications"
	"whistleblower_REST/internal/reports"
)

func RegisterRoutes(db *gorm.DB, hub *websocket.Hub) *chi.Mux {
	r := chi.NewRouter()

	
	wsHandler := websocket.NewWSHandler(db, hub)

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
	roleHandler := &admin.RoleHandler{DB: db}
	settingsHandler := &admin.SettingsHandler{DB: db}
	workflowHandler := &admin.WorkflowHandler{DB: db}
	actionHandler := &admin.ActionHandler{DB: db}

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

		r.Route("/ws/reports", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/{reportId}", wsHandler.HandleConnections)
		})

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
			

			// Public 
			r.Get("/{reportId}", reportHandler.GetByID)
			r.Get("/{reportId}/messages", messageHandler.GetByReportID)
			r.Get("/categories", categoryHandler.GetAllCategories)
			r.With(auth.OptionalAuthMiddleware()).Post("/", reportHandler.Create) // publik bisa kirim laporan (optional auth)
			r.Put("/{reportId}/assign", reportHandler.AssignAdmin)                // publik bisa assign admin (dengan email wajib)

			 

			// --- Protected routes (login required) ---
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware)
				r.Get("/my", reportHandler.GetMy)             // laporan user login
				r.With(auth.RoleMiddleware("admin")).Get("/", reportHandler.GetAll)
				r.With(auth.RoleMiddleware("admin")).Patch("/{reportId}", reportHandler.Update)
				r.Delete("/{reportId}", reportHandler.Delete) // hapus laporan
			})

			// === EVIDENCE nested routes ===
			r.Route("/{reportId}/evidence", func(r chi.Router) {
				r.Get("/", evidenceHandler.GetByReport)
				r.Post("/", evidenceHandler.Create)
				r.Delete("/{evidenceId}", evidenceHandler.Delete)
				r.Get("/file/{id}", evidenceHandler.DownloadEvidence)
			})

			// === MESSAGES nested routes ===
			r.Route("/{reportId}/messages", func(r chi.Router) {
				r.Get("/", messageHandler.GetByReportID)

				r.With(authMiddleware).Post("/", messageHandler.Create)
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
			r.Get("/investigator-performance", analyticsHandler.GetInvestigatorPerformance)
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

			// Roles
			r.Get("/roles", roleHandler.GetRoles)
			r.Post("/roles", roleHandler.CreateRole)
			r.Patch("/roles/{roleId}", roleHandler.UpdateRole)

			// Setting
			r.Get("/settings", settingsHandler.GetSettings)
			r.Put("/settings", settingsHandler.UpdateSettings)

			// Workflows
			r.Get("/workflows", workflowHandler.GetWorkflows)
			r.Put("/workflows", workflowHandler.UpdateWorkflows)

			// Notifications
			r.Post("/notify-by-report", notifications.SendFromAdminByReport(db))

			// Actions
			r.Route("/actions", func(r chi.Router) {
				r.Post("/{reportId}", actionHandler.CreateAction)      // Buat tindakan untuk report tertentu
				r.Get("/{reportId}", actionHandler.GetActionsByReport) // Ambil semua tindakan untuk report
				r.Patch("/{reportId}/complete", actionHandler.MarkActionCompleted) // Update Status Laporan
			})
		})
	})

	r.Post("/notify", notifications.SendNotification)

	// ===== HEALTH CHECK =====
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		utils.RespondWithJSON(w, 200, map[string]string{"status": "ok"})
	})

	return r
}
