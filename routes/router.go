package routes

import (
	"net/http"
	"whistleblower_REST/internal/admin"
	"whistleblower_REST/internal/analytics"
	"whistleblower_REST/internal/auth"
	"whistleblower_REST/internal/chatagent"
	"whistleblower_REST/internal/evidence"
	"whistleblower_REST/internal/feedback"
	"whistleblower_REST/internal/utils"
	"whistleblower_REST/internal/websocket"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"

	"whistleblower_REST/internal/ai"
	"whistleblower_REST/internal/messages"
	"whistleblower_REST/internal/notifications"
	"whistleblower_REST/internal/reports"
	"whistleblower_REST/internal/reviews"
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
	messageHandler := messages.NewHandler(db, hub)
	analyticsHandler := &analytics.AnalyticsHandler{DB: db}
	roleHandler := &admin.RoleHandler{DB: db}
	settingsHandler := &admin.SettingsHandler{DB: db}
	workflowHandler := &admin.WorkflowHandler{DB: db}
	actionHandler := &admin.ActionHandler{DB: db}
	adminHandler := &admin.AdminHandler{DB: db}
	reviewHandler := reviews.NewHandler(db)
	aiHandler := ai.NewHandler(db)
	feedbackTypeHandler := feedback.NewTypeHandler(db)
	feedbackHandler := feedback.NewFeedbackHandler(db)
	
	

	// FCM Routes
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware()) // User must be authenticated

		// Register device for push notifications
		r.Post("/api/devices/register", notifications.RegisterDevice(db))
		
		// Unregister device
		r.Post("/api/devices/unregister", notifications.UnregisterDevice(db))
		
		// Get all user devices
		r.Get("/api/devices", notifications.GetUserDevices(db))
		
		// Delete specific device
		r.Delete("/api/devices/{deviceId}", notifications.DeleteDevice(db))
		
		// Test push notification
		r.Post("/api/notifications/test-push", notifications.TestPushNotification(db))
	})


	r.Route("/feedbacks", func(r chi.Router) {
		// Public: Submit feedback (anonymous or authenticated)
		r.With(auth.OptionalAuthMiddleware()).Post("/", feedbackHandler.CreateFeedback)
		
		// Public: Upload image for feedback
		r.Post("/{feedbackId}/image", feedbackHandler.UploadFeedbackImage)
		
		// Public: Get active feedback types
		r.Get("/types", feedbackTypeHandler.GetAllFeedbackTypes)
		
		// Protected: Get my feedbacks (user only)
		r.With(authMiddleware).Get("/my", feedbackHandler.GetMyFeedbacks)
	})




	// ===== AUTH ROUTES =====
	r.Route("/auth", func(r chi.Router) {
		// === Auth Basic === //
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/google", authHandler.GoogleAuth)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/validate", authHandler.ValidateToken)

		// === Password & Security === //
		r.Post("/forgot-password", authHandler.ForgotPassword)
		r.Post("/verify-code", authHandler.VerifyResetCode)
		r.Post("/reset-password",authHandler.ResetPassword)

		// === Change Password (with auth) === //
		r.Group(func(r chi.Router) {
			r.Use(auth.AuthMiddleware()) // ✅ PENTING: Apply middleware
			r.Post("/request-change-password", authHandler.RequestChangePassword)
			r.Post("/verify-change-code", authHandler.VerifyChangePasswordCode)
			r.Post("/change-password", authHandler.ChangePassword)
		})
		
		
		

		// === Logout === // 
		r.Post("/logout", func(w http.ResponseWriter, r *http.Request) {
			utils.RespondWithJSON(w, 200, map[string]string{"message": "logout"})
		})

		// === Protected Profile === //
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/me", authHandler.Me)
			r.Patch("/edit-profile", authHandler.EditProfile)
		})
	})


		
	

	// ===== PROTECTED ROUTES =====
	r.Group(func(r chi.Router) {

		 r.Post("/chat-agent", chatagent.ChatHandler)

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

		r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware())

		// Get user notifications
		r.Get("/api/notifications/user", notifications.GetUserNotifications(db))
		
		// Mark single notification as read
		r.Patch("/api/notifications/{notificationId}/read", notifications.MarkNotificationAsRead(db))
		
		// Mark all notifications as read
		r.Patch("/api/notifications/read-all", notifications.MarkAllNotificationsAsRead(db))
		
		// Delete notification
		r.Delete("/api/notifications/{notificationId}", notifications.DeleteNotification(db))
		
		// Delete all notifications
		r.Delete("/api/notifications/all", notifications.DeleteAllUserNotifications(db))
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
				r.Get("/{reportId}/actions", actionHandler.GetActionsByReport)
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

				r.With(authMiddleware).Post("/upload", wsHandler.UploadMessageFile)
				

				r.With(authMiddleware).Get("/unread-count", messageHandler.GetUnreadCount)
    			r.With(authMiddleware).Patch("/mark-all-read", messageHandler.MarkAllMessagesAsRead)
    			r.With(authMiddleware).Get("/{messageId}/status", messageHandler.GetMessageReadStatus)
    			r.With(authMiddleware).Patch("/{messageId}/read", messageHandler.MarkMessageAsRead)
    
    			// 🆕 Edit & Delete message (dengan auth)
				r.With(authMiddleware).Patch("/{messageId}", messageHandler.UpdateMessage)
    			r.With(authMiddleware).Delete("/{messageId}", messageHandler.Delete)
    			r.With(authMiddleware).Get("/{messageId}", messageHandler.GetByID)
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

			r.Route("/admins", func(r chi.Router) {
				r.Get("/", adminHandler.GetAdmins)     // GET /admin/config/admins?department=IT
				r.Post("/", adminHandler.CreateAdmin)  // POST /admin/config/admins
			})

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
			r.Get("/notifications", notifications.GetAllAdminNotifications(db)) 

			// Actions
			r.Route("/actions", func(r chi.Router) {
				r.Post("/{reportId}", actionHandler.CreateAction)      // Buat tindakan untuk report tertentu
				r.Get("/{reportId}", actionHandler.GetActionsByReport) // Ambil semua tindakan untuk report
				r.Patch("/{reportId}/complete", actionHandler.MarkActionCompleted) // Update Status Laporan
			})

			r.Route("/reviews", func(r chi.Router) {
				r.Post("/", reviewHandler.Create)                    // Create review untuk report
				r.Get("/", reviewHandler.GetAll)                     // Get all reviews
				r.Get("/report/{reportId}", reviewHandler.GetByReportID) // Get review by report ID
				r.Patch("/report/{reportId}", reviewHandler.Update)  // Update review
				r.Delete("/report/{reportId}", reviewHandler.Delete) // Delete review
			})
			r.Route("/ai", func(r chi.Router) {
				r.Post("/analyze/{reportId}", aiHandler.AnalyzeReport)    // Trigger AI analysis
				r.Post("/re-analyze/{reportId}", aiHandler.ReAnalyze)     // Re-analyze existing
				r.Get("/analysis/{reportId}", aiHandler.GetByReportID)    // Get AI analysis by report
				r.Get("/analyses", aiHandler.GetAllAnalyses)              // Get all AI analyses
				r.Get("/statistics", aiHandler.GetStatistics)             // AI statistics
				r.Get("/test-connection", aiHandler.TestGeminiConnection) // Test Mistral API
			})

			r.Route("/feedbacks", func(r chi.Router) {
				r.Use(authMiddleware)
				r.Use(auth.RoleMiddleware("admin"))

				// Feedback management
				r.Get("/", feedbackHandler.GetAllFeedbacks)                   // List all feedbacks
				r.Get("/{id}", feedbackHandler.GetFeedbackByID)              // Get feedback detail
				r.Post("/{id}/respond", feedbackHandler.RespondToFeedback)   // Admin respond to feedback
				r.Delete("/{id}", feedbackHandler.DeleteFeedback)            // Delete feedback

		// Feedback type management
			r.Route("/types", func(r chi.Router) {
				r.Post("/", feedbackTypeHandler.CreateFeedbackType)      // Create feedback type
				r.Get("/", feedbackTypeHandler.GetAllFeedbackTypes)      // List all types
				r.Get("/{id}", feedbackTypeHandler.GetFeedbackTypeByID)  // Get type by ID
				r.Patch("/{id}", feedbackTypeHandler.UpdateFeedbackType) // Update type
				r.Delete("/{id}", feedbackTypeHandler.DeleteFeedbackType)// Delete type
			})
		})
	})
	})

	r.Post("/notify", notifications.SendNotification(db))

	// ===== HEALTH CHECK =====
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		utils.RespondWithJSON(w, 200, map[string]string{"status": "ok"})
	})

	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	return r
}
