package reports

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"whistleblower_REST/internal/utils"
)

type ReportsHandler struct {
	DB *sql.DB
}

type CreateReportRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

type UpdateReportRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`
	Status      *string `json:"status,omitempty"`
}


func (h *ReportsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	userUID := r.Context().Value("uid").(string)
	reportID := uuid.NewString()

	_, err := h.DB.Exec(`
		INSERT INTO reports (id, user_uid, title, description, category, status)
		VALUES (?, ?, ?, ?, ?, 'OPEN')`,
		reportID, userUID, input.Title, input.Description, input.Category,
	)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, map[string]string{
		"message": "report created successfully",
		"id":      reportID,
	})
}

// GET /reports
func (h *ReportsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	category := r.URL.Query().Get("category")

	query := `SELECT id, user_uid, title, description, category, status, created_at
			  FROM reports WHERE 1=1`
	args := []interface{}{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var reports []map[string]interface{}
	for rows.Next() {
		var r Report
		if err := rows.Scan(&r.ID, &r.UserUID, &r.Title, &r.Description, &r.Category, &r.Status, &r.CreatedAt); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		reports = append(reports, map[string]interface{}{
			"id":          r.ID,
			"user_uid":    r.UserUID,
			"title":       r.Title,
			"description": r.Description,
			"category":    r.Category,
			"status":      r.Status,
			"created_at":  r.CreatedAt,
		})
	}

	utils.RespondWithJSON(w, http.StatusOK, reports)
}

// GET /reports/my
func (h *ReportsHandler) GetMy(w http.ResponseWriter, r *http.Request) {
	userUID := r.Context().Value("uid").(string)

	rows, err := h.DB.Query(`
		SELECT id, title, description, category, status, created_at
		FROM reports WHERE user_uid = ? ORDER BY created_at DESC`,
		userUID,
	)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var reports []map[string]interface{}
	for rows.Next() {
		var r Report
		if err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Category, &r.Status, &r.CreatedAt); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		reports = append(reports, map[string]interface{}{
			"id":          r.ID,
			"title":       r.Title,
			"description": r.Description,
			"category":    r.Category,
			"status":      r.Status,
			"created_at":  r.CreatedAt,
		})
	}

	utils.RespondWithJSON(w, http.StatusOK, reports)
}

// GET /reports/{id}
func (h *ReportsHandler) GetByID(w http.ResponseWriter, r *http.Request, id string) {
	var report Report
	err := h.DB.QueryRow(`
		SELECT id, user_uid, title, description, category, status, created_at
		FROM reports WHERE id = ?`, id).
		Scan(&report.ID, &report.UserUID, &report.Title, &report.Description, &report.Category, &report.Status, &report.CreatedAt)

	if err == sql.ErrNoRows {
		utils.RespondWithError(w, http.StatusNotFound, "report not found")
		return
	} else if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, report)
}

// PATCH /reports/{id}
func (h *ReportsHandler) Update(w http.ResponseWriter, r *http.Request, id string) {
	var req UpdateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	query := `UPDATE reports SET `
	args := []interface{}{}
	updates := []string{}

	if req.Title != nil {
		updates = append(updates, "title = ?")
		args = append(args, *req.Title)
	}
	if req.Description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *req.Description)
	}
	if req.Category != nil {
		updates = append(updates, "category = ?")
		args = append(args, *req.Category)
	}
	if req.Status != nil {
		updates = append(updates, "status = ?")
		args = append(args, *req.Status)
	}

	if len(updates) == 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	query += (stringJoin(updates, ", ") + " WHERE id = ?")
	args = append(args, id)

	_, err := h.DB.Exec(query, args...)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "report updated"})
}

// DELETE /reports/{id}
func (h *ReportsHandler) Delete(w http.ResponseWriter, r *http.Request, id string) {
	_, err := h.DB.Exec(`DELETE FROM reports WHERE id = ?`, id)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "report deleted"})
}

// Helper
func stringJoin(arr []string, sep string) string {
	if len(arr) == 0 {
		return ""
	}
	out := arr[0]
	for _, s := range arr[1:] {
		out += sep + s
	}
	return out
}
