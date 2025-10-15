package reports

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"whistleblower_REST/internal/auth"
)

// Handler mengatur semua endpoint /reports
type Handler struct {
	repo Repository
}

func NewHandler(r Repository) *Handler {
	return &Handler{repo: r}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("", h.GetAll)
	rg.POST("", auth.AuthMiddleware(), h.Create)
	rg.GET("/my", auth.AuthMiddleware(), h.GetMy)
	rg.GET("/:reportId", h.GetByID)
	rg.PATCH("/:reportId", auth.AuthMiddleware(), h.Patch)
	rg.DELETE("/:reportId", auth.AuthMiddleware(), h.Delete)
}

// GET /reports
func (h *Handler) GetAll(c *gin.Context) {
	status := c.Query("status")
	category := c.Query("category")
	items, err := h.repo.GetAll(c, status, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list"})
		return
	}
	c.JSON(http.StatusOK, items)
}

// POST /reports
func (h *Handler) Create(c *gin.Context) {
	var req CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rp := &Report{
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
		Status:      "OPEN",
		UserID:      c.GetInt64("userID"),
	}
	if err := h.repo.Create(c, rp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create"})
		return
	}
	c.JSON(http.StatusCreated, rp)
}

// GET /reports/my
func (h *Handler) GetMy(c *gin.Context) {
	uid := c.GetInt64("userID")
	items, err := h.repo.GetByUser(c, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list"})
		return
	}
	c.JSON(http.StatusOK, items)
}

// GET /reports/{reportId}
func (h *Handler) GetByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("reportId"), 10, 64)
	item, err := h.repo.GetByID(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get"})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// PATCH /reports/{reportId}
func (h *Handler) Patch(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("reportId"), 10, 64)
	var req UpdateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.repo.UpdatePartial(c, id, req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found or update failed"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// DELETE /reports/{reportId}
func (h *Handler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("reportId"), 10, 64)
	if err := h.repo.Delete(c, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
