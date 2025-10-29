package reports

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"whistleblower_REST/internal/models"
	"whistleblower_REST/internal/utils"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler { return &Handler{DB: db} }

// ==============================
// CREATE REPORT
// ==============================
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in models.CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Validasi basic
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	if in.Title == "" || in.Description == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "title and description are required")
		return
	}

	// Ambil ID user dari context (jika login)
	var uid string
	v := r.Context().Value("id")
	if idStr, ok := v.(string); ok && idStr != "" {
		uid = idStr
	}

	// Default: anonymous
	reporterType := models.ReporterAnonymous
	var userID *string
	if uid != "" {
		reporterType = models.ReporterAuthenticated
		userID = &uid
		in.Email = nil // abaikan email kalau user login
	} else {
		// Email wajib jika user tidak login
		if in.Email == nil || strings.TrimSpace(*in.Email) == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "email is required for anonymous reports")
			return
		}
	}

	report := models.Report{
		Title:        in.Title,
		Description:  in.Description,
		Category:     in.Category,
		Status:       models.StatusSubmitted,
		ReporterType: reporterType,
		UserID:       userID,
		Email:        in.Email,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.DB.Create(&report).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Log simulasi kirim email
	if report.Email != nil {
		fmt.Printf("[INFO] Simulasi kirim email ke %s (Laporan ID #%d, Status: %s)\n",
			*report.Email, report.ID, report.Status)
	}

	utils.RespondWithJSON(w, http.StatusCreated, report)
}

// ==============================
// GET ALL REPORTS (admin only)
// ==============================
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	reporterType := strings.TrimSpace(r.URL.Query().Get("reporter_type"))

	var reports []models.Report

	// ✅ Preload relasi ke tabel users agar bisa akses user.Name
	tx := h.DB.Preload("User").Model(&models.Report{})

	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if category != "" {
		tx = tx.Where("category = ?", category)
	}
	if reporterType != "" {
		tx = tx.Where("reporter_type = ?", reporterType)
	}

	if err := tx.Order("created_at DESC").Find(&reports).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// ✅ Bentuk response dengan nama pelapor sesuai tipe
	type ReportResponse struct {
		ID           uint       `json:"id"`
		Title        string     `json:"title"`
		Description  string     `json:"description"`
		Category     string     `json:"category"`
		Status       string     `json:"status"`
		ReporterType string     `json:"reporter_type"`
		ReporterName string     `json:"reporter_name"`
		Email        *string    `json:"email,omitempty"`
		CreatedAt    time.Time  `json:"created_at"`
		UpdatedAt    time.Time  `json:"updated_at"`
	}

	var out []ReportResponse
	for _, r := range reports {
		reporterName := "Anonymous"

		// Kalau user login, ambil nama user dari relasi
		if r.ReporterType == models.ReporterAuthenticated && r.User != nil {
			reporterName = r.User.Name
		}

		out = append(out, ReportResponse{
			ID:           r.ID,
			Title:        r.Title,
			Description:  r.Description,
			Category:     r.Category,
			Status:       r.Status,
			ReporterType: string(r.ReporterType),
			ReporterName: reporterName,
			Email:        r.Email,
			CreatedAt:    r.CreatedAt,
			UpdatedAt:    r.UpdatedAt,
		})
	}

	utils.RespondWithJSON(w, http.StatusOK, out)
}




// ==============================
// GET USER'S OWN REPORTS
// ==============================
func (h *Handler) GetMy(w http.ResponseWriter, r *http.Request) {
	v := r.Context().Value("id")
	uid, _ := v.(string)
	if uid == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var list []models.Report
	if err := h.DB.Where("user_id = ?", uid).Order("created_at DESC").Find(&list).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, list)
}

// ==============================
// GET REPORT BY ID
// ==============================
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUintID(chi.URLParam(r, "reportId"))
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report id")
		return
	}

	var rp models.Report
	if err := h.DB.First(&rp, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "report not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, rp)
}

// ==============================
// UPDATE REPORT STATUS
// ==============================
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUintID(chi.URLParam(r, "reportId"))
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report id")
		return
	}

	// Ambil role dari context
	role, _ := r.Context().Value("role").(string)
	if role != "admin" {
		utils.RespondWithError(w, http.StatusForbidden, "forbidden: only admin can update report status")
		return
	}

	var in models.UpdateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	updates := map[string]any{
		"updated_at": time.Now(),
	}

	if in.Status != nil {
		if !isValidStatus(*in.Status) {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid status value")
			return
		}
		updates["status"] = *in.Status
	}

	if len(updates) == 1 {
		utils.RespondWithError(w, http.StatusBadRequest, "no updatable fields provided")
		return
	}

	// === Update report status ===
	res := h.DB.Model(&models.Report{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, res.Error.Error())
		return
	}
	if res.RowsAffected == 0 {
		utils.RespondWithError(w, http.StatusNotFound, "report not found")
		return
	}

	// === Jika status berubah jadi 'under_review', set handle_at di tabel actions ===
	if in.Status != nil && *in.Status == models.StatusUnderReview {
		now := time.Now()
		h.DB.Model(&models.Action{}).
			Where("report_id = ?", id).
			Update("handle_at", now)
		fmt.Printf("[INFO] Laporan #%d mulai ditangani pada %s\n", id, now.Format(time.RFC3339))
	}

	fmt.Printf("[INFO] Status laporan #%d diperbarui menjadi %v oleh admin\n", id, in.Status)
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "report updated"})
}



// ==============================
// ASSIGN ADMIN TO REPORT (public allowed)
// ==============================
func (h *Handler) AssignAdmin(w http.ResponseWriter, r *http.Request) {
	reportID, ok := parseUintID(chi.URLParam(r, "reportId"))
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report id")
		return
	}

	var body struct {
		AdminID string  `json:"admin_id"`
		Email   *string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validasi email wajib untuk publik
	if body.Email == nil || strings.TrimSpace(*body.Email) == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "email is required for public assignment")
		return
	}

	// Validasi admin
	var admin models.User
	if err := h.DB.Where("id = ? AND role = ?", body.AdminID, "admin").First(&admin).Error; err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid admin id")
		return
	}

	// Cek report
	var report models.Report
	if err := h.DB.First(&report, reportID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.RespondWithError(w, http.StatusNotFound, "report not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	updates := map[string]any{
		"assigned_admin_id": admin.ID,
		"email":             body.Email,
		"status":            models.StatusUnderReview,
		"updated_at":        time.Now(),
	}

	if err := h.DB.Model(&models.Report{}).Where("id = ?", reportID).Updates(updates).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Simulasi kirim notifikasi
	fmt.Printf("[INFO] Admin %s (%s) ditugaskan ke laporan #%d (notifikasi seharusnya ke %s)\n",
		admin.Name, admin.Email, report.ID, *body.Email)

	utils.RespondWithJSON(w, http.StatusOK, map[string]any{
		"message": "report assigned successfully",
		"assignedAdmin": map[string]string{
			"id":    admin.ID,
			"name":  admin.Name,
			"email": admin.Email,
		},
	})
}

// ==============================
// DELETE REPORT
// ==============================
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUintID(chi.URLParam(r, "reportId"))
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid report id")
		return
	}

	res := h.DB.Delete(&models.Report{}, "id = ?", id)
	if res.Error != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, res.Error.Error())
		return
	}
	if res.RowsAffected == 0 {
		utils.RespondWithError(w, http.StatusNotFound, "report not found")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "report deleted"})
}

// ==============================
// HELPERS
// ==============================
func parseUintID(s string) (uint, bool) {
	if s == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(n), true
}

func isValidStatus(st string) bool {
	switch st {
	case models.StatusSubmitted, models.StatusUnderReview, models.StatusResolved, models.StatusDismissed:
		return true
	default:
		return false
	}
}
