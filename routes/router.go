package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, db *sql.DB) {
	// ===== Public Routes =====
	auth := r.Group("/auth")
	{
		auth.POST("/register", func(c *gin.Context) { c.JSON(200, gin.H{"message": "register"}) })
		auth.POST("/login", func(c *gin.Context) { c.JSON(200, gin.H{"message": "login"}) })
		auth.POST("/refresh", func(c *gin.Context) { c.JSON(200, gin.H{"message": "refresh"}) })
		auth.POST("/logout", func(c *gin.Context) { c.JSON(200, gin.H{"message": "logout"}) })
		auth.POST("/reset-password", func(c *gin.Context) { c.JSON(200, gin.H{"message": "reset password"}) })
		auth.GET("/me", func(c *gin.Context) { c.JSON(200, gin.H{"message": "me"}) })
	}

	// ===== Protected Routes (placeholder, add middleware later) =====
	users := r.Group("/users")
	{
		users.GET("", func(c *gin.Context) { c.JSON(200, gin.H{"message": "get all users"}) })
		users.GET("/:userId", func(c *gin.Context) { c.JSON(200, gin.H{"message": "get user by id"}) })
		users.PATCH("/:userId", func(c *gin.Context) { c.JSON(200, gin.H{"message": "update user"}) })
	}

	reports := r.Group("/reports")
	{
		reports.POST("", func(c *gin.Context) { c.JSON(200, gin.H{"message": "create report"}) })
		reports.GET("", func(c *gin.Context) { c.JSON(200, gin.H{"message": "get all reports"}) })
		reports.GET("/my", func(c *gin.Context) { c.JSON(200, gin.H{"message": "get my reports"}) })
		reports.GET("/:reportId", func(c *gin.Context) { c.JSON(200, gin.H{"message": "get report by id"}) })
		reports.PATCH("/:reportId", func(c *gin.Context) { c.JSON(200, gin.H{"message": "update report"}) })
		reports.DELETE("/:reportId", func(c *gin.Context) { c.JSON(200, gin.H{"message": "delete report"}) })

		// Evidence nested routes
		reports.GET("/:reportId/evidence", func(c *gin.Context) { c.JSON(200, gin.H{"message": "get evidence list"}) })
		reports.POST("/:reportId/evidence", func(c *gin.Context) { c.JSON(200, gin.H{"message": "upload evidence"}) })
		reports.GET("/:reportId/evidence/:evidenceId", func(c *gin.Context) { c.JSON(200, gin.H{"message": "get evidence by id"}) })
		reports.DELETE("/:reportId/evidence/:evidenceId", func(c *gin.Context) { c.JSON(200, gin.H{"message": "delete evidence"}) })

		// Messages nested routes
		reports.GET("/:reportId/messages", func(c *gin.Context) { c.JSON(200, gin.H{"message": "get messages"}) })
		reports.POST("/:reportId/messages", func(c *gin.Context) { c.JSON(200, gin.H{"message": "create message"}) })
	}

	// ===== Analytics =====
	analytics := r.Group("/analytics")
	{
		analytics.GET("/overview", func(c *gin.Context) { c.JSON(200, gin.H{"message": "analytics overview"}) })
		analytics.GET("/trends", func(c *gin.Context) { c.JSON(200, gin.H{"message": "analytics trends"}) })
		analytics.GET("/by-categories", func(c *gin.Context) { c.JSON(200, gin.H{"message": "analytics by categories"}) })
		analytics.GET("/by-status", func(c *gin.Context) { c.JSON(200, gin.H{"message": "analytics by status"}) })
		analytics.GET("/investigator-performance", func(c *gin.Context) { c.JSON(200, gin.H{"message": "analytics performance"}) })
		analytics.POST("/reports/generate", func(c *gin.Context) { c.JSON(200, gin.H{"message": "generate report"}) })
	}

	// ===== Admin Config =====
	admin := r.Group("/admin/config")
	{
		admin.GET("/categories", func(c *gin.Context) { c.JSON(200, gin.H{"message": "get categories"}) })
		admin.POST("/categories", func(c *gin.Context) { c.JSON(200, gin.H{"message": "create category"}) })
		admin.PATCH("/categories/:catId", func(c *gin.Context) { c.JSON(200, gin.H{"message": "update category"}) })
		admin.DELETE("/categories/:catId", func(c *gin.Context) { c.JSON(200, gin.H{"message": "delete category"}) })

		admin.GET("/roles", func(c *gin.Context) { c.JSON(200, gin.H{"message": "get roles"}) })
		admin.POST("/roles", func(c *gin.Context) { c.JSON(200, gin.H{"message": "create role"}) })
		admin.PATCH("/roles/:roleId", func(c *gin.Context) { c.JSON(200, gin.H{"message": "update role"}) })

		admin.GET("/settings", func(c *gin.Context) { c.JSON(200, gin.H{"message": "get settings"}) })
		admin.PUT("/settings", func(c *gin.Context) { c.JSON(200, gin.H{"message": "update settings"}) })

		admin.GET("/workflows", func(c *gin.Context) { c.JSON(200, gin.H{"message": "get workflows"}) })
		admin.PUT("/workflows", func(c *gin.Context) { c.JSON(200, gin.H{"message": "update workflows"}) })
	}
}
