package evidence

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository interface {
	Create(ctx context.Context, e *Evidence) error
	GetByReport(ctx context.Context, reportID string) ([]Evidence, error)
	Delete(ctx context.Context, id string) error
}

type sqliteRepo struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) Repository {
	return &sqliteRepo{db: db}
}

func (r *sqliteRepo) Create(ctx context.Context, e *Evidence) error {
	q := `
		INSERT INTO evidence (id, report_id, file_path)
		VALUES (?, ?, ?)
		RETURNING created_at;
	`
	return r.db.QueryRowContext(ctx, q, e.ID, e.ReportID, e.FilePath).
		Scan(&e.CreatedAt)
}

func (r *sqliteRepo) GetByReport(ctx context.Context, reportID string) ([]Evidence, error) {
	q := `
		SELECT id, report_id, file_path, created_at
		FROM evidence
		WHERE report_id = ?
		ORDER BY created_at DESC;
	`
	rows, err := r.db.QueryContext(ctx, q, reportID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Evidence
	for rows.Next() {
		var e Evidence
		if err := rows.Scan(&e.ID, &e.ReportID, &e.FilePath, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

func (r *sqliteRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM evidence WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}
