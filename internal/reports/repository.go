package reports

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Repository interface {
	Create(ctx context.Context, r *Report) error
	GetAll(ctx context.Context, status, category string) ([]Report, error)
	GetByUser(ctx context.Context, userID int64) ([]Report, error)
	GetByID(ctx context.Context, id int64) (*Report, error)
	UpdatePartial(ctx context.Context, id int64, u UpdateReportRequest) (*Report, error)
	Delete(ctx context.Context, id int64) error
}

type sqliteRepo struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) Repository {
	return &sqliteRepo{db: db}
}

func (r *sqliteRepo) Create(ctx context.Context, rp *Report) error {
	q := `INSERT INTO reports (title, description, category, status, user_id)
	      VALUES (?,?,?,?,?) RETURNING id, created_at, status`
	return r.db.QueryRowContext(ctx, q,
		rp.Title, rp.Description, rp.Category, "OPEN", rp.UserID,
	).Scan(&rp.ID, &rp.CreatedAt, &rp.Status)
}

func (r *sqliteRepo) GetAll(ctx context.Context, status, category string) ([]Report, error) {
	sb := strings.Builder{}
	args := []any{}
	sb.WriteString(`SELECT id,title,description,category,status,user_id,created_at FROM reports`)
	clauses := []string{}
	if status != "" {
		clauses = append(clauses, "status=?")
		args = append(args, status)
	}
	if category != "" {
		clauses = append(clauses, "category=?")
		args = append(args, category)
	}
	if len(clauses) > 0 {
		sb.WriteString(" WHERE " + strings.Join(clauses, " AND "))
	}
	sb.WriteString(" ORDER BY created_at DESC")

	rows, err := r.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Report
	for rows.Next() {
		var rp Report
		if err := rows.Scan(&rp.ID, &rp.Title, &rp.Description, &rp.Category, &rp.Status, &rp.UserID, &rp.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rp)
	}
	return out, rows.Err()
}

func (r *sqliteRepo) GetByUser(ctx context.Context, userID int64) ([]Report, error) {
	q := `SELECT id,title,description,category,status,user_id,created_at
	      FROM reports WHERE user_id=? ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Report
	for rows.Next() {
		var rp Report
		if err := rows.Scan(&rp.ID, &rp.Title, &rp.Description, &rp.Category, &rp.Status, &rp.UserID, &rp.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rp)
	}
	return out, rows.Err()
}

func (r *sqliteRepo) GetByID(ctx context.Context, id int64) (*Report, error) {
	q := `SELECT id,title,description,category,status,user_id,created_at
	      FROM reports WHERE id=?`
	var rp Report
	if err := r.db.QueryRowContext(ctx, q, id).
		Scan(&rp.ID, &rp.Title, &rp.Description, &rp.Category, &rp.Status, &rp.UserID, &rp.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rp, nil
}

func (r *sqliteRepo) UpdatePartial(ctx context.Context, id int64, u UpdateReportRequest) (*Report, error) {
	q := `
	UPDATE reports SET
		title       = COALESCE(?, title),
		description = COALESCE(?, description),
		category    = COALESCE(?, category),
		status      = COALESCE(?, status)
	WHERE id=?`
	res, err := r.db.ExecContext(ctx, q, u.Title, u.Description, u.Category, u.Status, id)
	if err != nil {
		return nil, err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return nil, fmt.Errorf("not found")
	}
	return r.GetByID(ctx, id)
}

func (r *sqliteRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM reports WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}
