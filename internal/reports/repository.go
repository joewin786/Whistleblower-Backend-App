package reports

import (
	"context"
	"database/sql"
	"fmt"
)

// Repository interface mendefinisikan semua operasi database untuk tabel reports
type Repository interface {
	Create(ctx context.Context, r *Report) error
	GetAll(ctx context.Context, status, category string) ([]Report, error)
	GetByUser(ctx context.Context, userUID string) ([]Report, error)
	GetByID(ctx context.Context, id string) (*Report, error)
	UpdatePartial(ctx context.Context, id string, req UpdateReportRequest) error
	Delete(ctx context.Context, id string) error
}

// sqliteRepo adalah implementasi repository untuk SQLite
type sqliteRepo struct {
	db *sql.DB
}

// NewSQLiteRepository membuat instance repository baru
func NewSQLiteRepository(db *sql.DB) Repository {
	return &sqliteRepo{db: db}
}

// Create menambahkan laporan baru ke database
func (r *sqliteRepo) Create(ctx context.Context, rp *Report) error {
	q := `
		INSERT INTO reports (id, user_uid, title, description, category, status)
		VALUES (?, ?, ?, ?, ?, 'OPEN')
		RETURNING created_at;
	`
	return r.db.QueryRowContext(ctx, q,
		rp.ID, rp.UserUID, rp.Title, rp.Description, rp.Category,
	).Scan(&rp.CreatedAt)
}

// GetAll mengambil semua laporan (bisa difilter status & category)
func (r *sqliteRepo) GetAll(ctx context.Context, status, category string) ([]Report, error) {
	query := `
		SELECT id, user_uid, title, description, category, status, created_at
		FROM reports WHERE 1=1
	`
	args := []interface{}{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Report
	for rows.Next() {
		var rp Report
		if err := rows.Scan(
			&rp.ID, &rp.UserUID, &rp.Title, &rp.Description,
			&rp.Category, &rp.Status, &rp.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, rp)
	}
	return list, rows.Err()
}

// GetByUser mengambil semua laporan milik user tertentu
func (r *sqliteRepo) GetByUser(ctx context.Context, userUID string) ([]Report, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_uid, title, description, category, status, created_at
		FROM reports WHERE user_uid = ?
		ORDER BY created_at DESC;
	`, userUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []Report
	for rows.Next() {
		var rp Report
		if err := rows.Scan(
			&rp.ID, &rp.UserUID, &rp.Title, &rp.Description,
			&rp.Category, &rp.Status, &rp.CreatedAt,
		); err != nil {
			return nil, err
		}
		reports = append(reports, rp)
	}
	return reports, rows.Err()
}

// GetByID mengambil 1 laporan berdasarkan ID
func (r *sqliteRepo) GetByID(ctx context.Context, id string) (*Report, error) {
	var rp Report
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_uid, title, description, category, status, created_at
		FROM reports WHERE id = ?;
	`, id).Scan(
		&rp.ID, &rp.UserUID, &rp.Title, &rp.Description,
		&rp.Category, &rp.Status, &rp.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rp, nil
}

// UpdatePartial memperbarui sebagian field laporan
func (r *sqliteRepo) UpdatePartial(ctx context.Context, id string, req UpdateReportRequest) error {
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
		return fmt.Errorf("no fields to update")
	}

	query += joinStrings(updates, ", ") + " WHERE id = ?"
	args = append(args, id)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// Delete menghapus laporan berdasarkan ID
func (r *sqliteRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM reports WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

// helper sederhana untuk menggabungkan string SQL
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, s := range parts[1:] {
		out += sep + s
	}
	return out
}
