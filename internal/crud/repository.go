package crud

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Hisaalu/rogele/internal/dberr"
	"github.com/Hisaalu/rogele/internal/domain"
)

type ListOptions struct {
	Page     int
	PerPage  int
	Search   string
	Filters  map[string]string
	Sort     []string
	MaxLimit int
}

type ListResult struct {
	Rows    []map[string]any
	Total   int64
	Page    int
	PerPage int
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, res *Resource, opts ListOptions) (*ListResult, error) {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PerPage < 1 {
		opts.PerPage = 20
	}
	if opts.MaxLimit > 0 && opts.PerPage > opts.MaxLimit {
		opts.PerPage = opts.MaxLimit
	}
	offset := (opts.Page - 1) * opts.PerPage

	where, args := buildWhere(res, opts)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", res.Table, where)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count %s: %w", res.Table, err)
	}

	order := buildOrder(res, opts.Sort)

	cols := allColumns(res)
	selectCols := strings.Join(cols, ", ")

	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	query := fmt.Sprintf(
		"SELECT %s FROM %s%s%s LIMIT $%d OFFSET $%d",
		selectCols, res.Table, where, order, limitArg, offsetArg,
	)
	args = append(args, opts.PerPage, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", res.Table, err)
	}
	defer rows.Close()

	results, err := scanRows(rows, cols)
	if err != nil {
		return nil, err
	}

	return &ListResult{
		Rows:    results,
		Total:   total,
		Page:    opts.Page,
		PerPage: opts.PerPage,
	}, nil
}

func (r *Repository) Get(ctx context.Context, res *Resource, id any) (map[string]any, error) {
	cols := allColumns(res)
	selectCols := strings.Join(cols, ", ")
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", selectCols, res.Table, res.PrimaryKey)

	row := r.db.QueryRowContext(ctx, query, id)
	result, err := scanRow(row, cols)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", res.Table, err)
	}
	return result, nil
}

func (r *Repository) Create(ctx context.Context, res *Resource, data map[string]any) (map[string]any, error) {
	cols, vals := writableColumns(res, data)
	if len(cols) == 0 {
		return nil, fmt.Errorf("%w: no writable fields provided", domain.ErrInvalidInput)
	}

	placeholders := make([]string, len(cols))
	args := make([]any, len(cols))
	for i := range cols {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = vals[i]
	}

	allCols := allColumns(res)
	returning := strings.Join(allCols, ", ")

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
		res.Table,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
		returning,
	)

	row := r.db.QueryRowContext(ctx, query, args...)
	result, err := scanRow(row, allCols)
	if err != nil {
		return nil, dberr.Translate(fmt.Errorf("create %s: %w", res.Table, err))
	}
	return result, nil
}

func (r *Repository) Update(ctx context.Context, res *Resource, id any, data map[string]any) (map[string]any, error) {
	cols, vals := writableColumns(res, data)
	if len(cols) == 0 {
		return nil, fmt.Errorf("%w: no writable fields provided", domain.ErrInvalidInput)
	}

	sets := make([]string, len(cols))
	args := make([]any, len(cols))
	for i, c := range cols {
		sets[i] = fmt.Sprintf("%s = $%d", c, i+1)
		args[i] = vals[i]
	}

	idArg := len(args) + 1
	args = append(args, id)

	allCols := allColumns(res)
	returning := strings.Join(allCols, ", ")

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d RETURNING %s",
		res.Table,
		strings.Join(sets, ", "),
		res.PrimaryKey,
		idArg,
		returning,
	)

	row := r.db.QueryRowContext(ctx, query, args...)
	result, err := scanRow(row, allCols)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, dberr.Translate(fmt.Errorf("update %s: %w", res.Table, err))
	}
	return result, nil
}

func (r *Repository) Delete(ctx context.Context, res *Resource, id any) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", res.Table, res.PrimaryKey)
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return dberr.Translate(fmt.Errorf("delete %s: %w", res.Table, err))
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func buildWhere(res *Resource, opts ListOptions) (string, []any) {
	var clauses []string
	var args []any
	argN := 1

	if opts.Search != "" && len(res.Searchable) > 0 {
		var ors []string
		for _, col := range res.Searchable {
			ors = append(ors, fmt.Sprintf("%s::text ILIKE $%d", col, argN))
		}
		args = append(args, "%"+opts.Search+"%")
		argN++
		clauses = append(clauses, "("+strings.Join(ors, " OR ")+")")
	}

	for col, val := range opts.Filters {
		if !res.IsFilterable(col) {
			continue
		}
		clauses = append(clauses, fmt.Sprintf("%s::text = $%d", col, argN))
		args = append(args, val)
		argN++
	}

	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func buildOrder(res *Resource, sort []string) string {
	if len(sort) == 0 {
		sort = res.DefaultSort
	}

	var parts []string
	for _, s := range sort {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		dir := "ASC"
		col := s
		if strings.HasPrefix(s, "-") {
			dir = "DESC"
			col = s[1:]
		}
		if !res.IsKnownColumn(col) {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s", col, dir))
	}
	if len(parts) == 0 {
		return ""
	}
	return " ORDER BY " + strings.Join(parts, ", ")
}

func allColumns(res *Resource) []string {
	seen := make(map[string]struct{})
	var cols []string

	add := func(name string) {
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		cols = append(cols, name)
	}

	add(res.PrimaryKey)
	for name := range res.Columns {
		add(name)
	}
	for _, name := range builtinSystemColumns {
		add(name)
	}
	for _, name := range res.SystemColumns {
		add(name)
	}
	return cols
}

func writableColumns(res *Resource, data map[string]any) ([]string, []any) {
	var cols []string
	var vals []any
	for name, col := range res.Columns {
		if !col.Writable {
			continue
		}
		v, ok := data[name]
		if !ok {
			continue
		}
		cols = append(cols, name)
		vals = append(vals, v)
	}
	return cols, vals
}

func scanRow(row *sql.Row, cols []string) (map[string]any, error) {
	ptrs := make([]any, len(cols))
	bufs := make([]any, len(cols))
	for i := range cols {
		ptrs[i] = &bufs[i]
	}
	if err := row.Scan(ptrs...); err != nil {
		return nil, err
	}
	out := make(map[string]any, len(cols))
	for i, c := range cols {
		out[c] = normalizeValue(bufs[i])
	}
	return out, nil
}

func scanRows(rows *sql.Rows, cols []string) ([]map[string]any, error) {
	results := make([]map[string]any, 0, 16)
	for rows.Next() {
		ptrs := make([]any, len(cols))
		bufs := make([]any, len(cols))
		for i := range cols {
			ptrs[i] = &bufs[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		out := make(map[string]any, len(cols))
		for i, c := range cols {
			out[c] = normalizeValue(bufs[i])
		}
		results = append(results, out)
	}
	return results, rows.Err()
}

func normalizeValue(v any) any {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return v
}
