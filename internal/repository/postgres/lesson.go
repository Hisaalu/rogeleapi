package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Hisaalu/rogele/internal/dberr"
	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
)

type LessonRepo struct {
	db *sql.DB
}

func NewLessonRepo(db *sql.DB) *LessonRepo {
	return &LessonRepo{db: db}
}

const lessonColumns = `
	id, title, content, subject_id, class_id, teacher_id,
	video_url, duration, is_published, is_approved,
	views, created_at, updated_at
`

func (r *LessonRepo) Create(ctx context.Context, l *domain.Lesson) (*domain.Lesson, error) {
	query := `
		INSERT INTO lessons (
			title, content, subject_id, class_id, teacher_id,
			video_url, duration, is_published, is_approved
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING ` + lessonColumns

	row := r.db.QueryRowContext(ctx, query,
		l.Title, l.Content, l.SubjectID, l.ClassID, l.TeacherID,
		l.VideoURL, l.Duration, l.IsPublished, l.IsApproved,
	)
	result, err := scanLesson(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

func (r *LessonRepo) GetByID(ctx context.Context, id int64) (*domain.Lesson, error) {
	query := `SELECT ` + lessonColumns + ` FROM lessons WHERE id = $1`
	return scanLesson(r.db.QueryRowContext(ctx, query, id))
}

func (r *LessonRepo) Update(ctx context.Context, l *domain.Lesson) error {
	query := `
		UPDATE lessons SET
			title = $1,
			content = $2,
			subject_id = $3,
			class_id = $4,
			video_url = $5,
			duration = $6,
			is_published = $7,
			is_approved = $8,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $9
	`
	res, err := r.db.ExecContext(ctx, query,
		l.Title, l.Content, l.SubjectID, l.ClassID,
		l.VideoURL, l.Duration, l.IsPublished, l.IsApproved, l.ID,
	)
	if err != nil {
		return dberr.Translate(fmt.Errorf("update lesson: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *LessonRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM lessons WHERE id = $1`, id)
	if err != nil {
		return dberr.Translate(fmt.Errorf("delete lesson: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *LessonRepo) List(ctx context.Context, f repository.LessonFilter) ([]*domain.Lesson, error) {
	where, args := buildLessonWhere(f)
	i := len(args) + 1

	query := `SELECT ` + lessonColumns + ` FROM lessons`
	if where != "" {
		query += " WHERE " + where
	}
	query += " ORDER BY created_at DESC"

	if f.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", i)
		args = append(args, f.Limit)
		i++
	}
	if f.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", i)
		args = append(args, f.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list lessons: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.Lesson, 0, 16)
	for rows.Next() {
		l, err := scanLessonRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, l)
	}
	return results, rows.Err()
}

func (r *LessonRepo) IncrementViews(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE lessons SET views = views + 1 WHERE id = $1`, id)
	return err
}

const materialColumns = `
	id, lesson_id, file_name, file_path, file_type, file_size,
	download_count, created_at
`

func (r *LessonRepo) ListMaterials(ctx context.Context, lessonID int64) ([]*domain.LessonMaterial, error) {
	query := `SELECT ` + materialColumns + ` FROM lesson_materials WHERE lesson_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, lessonID)
	if err != nil {
		return nil, fmt.Errorf("list materials: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.LessonMaterial, 0, 8)
	for rows.Next() {
		m, err := scanMaterial(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

func (r *LessonRepo) GetMaterialByID(ctx context.Context, id int64) (*domain.LessonMaterial, error) {
	query := `SELECT ` + materialColumns + ` FROM lesson_materials WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	m, err := scanMaterial(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

func (r *LessonRepo) CreateMaterial(ctx context.Context, m *domain.LessonMaterial) (*domain.LessonMaterial, error) {
	query := `
		INSERT INTO lesson_materials (lesson_id, file_name, file_path, file_type, file_size)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING ` + materialColumns

	row := r.db.QueryRowContext(ctx, query,
		m.LessonID, m.FileName, m.FilePath, m.FileType, m.FileSize,
	)
	result, err := scanMaterial(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

func (r *LessonRepo) DeleteMaterial(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM lesson_materials WHERE id = $1`, id)
	if err != nil {
		return dberr.Translate(fmt.Errorf("delete material: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *LessonRepo) IsBookmarked(ctx context.Context, userID, lessonID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM bookmarks WHERE user_id = $1 AND lesson_id = $2)`,
		userID, lessonID,
	).Scan(&exists)
	return exists, err
}

func (r *LessonRepo) AddBookmark(ctx context.Context, b *domain.Bookmark) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO bookmarks (user_id, lesson_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		b.UserID, b.LessonID,
	)
	return dberr.Translate(err)
}

func (r *LessonRepo) RemoveBookmark(ctx context.Context, userID, lessonID int64) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM bookmarks WHERE user_id = $1 AND lesson_id = $2`,
		userID, lessonID,
	)
	return err
}

func (r *LessonRepo) ListBookmarks(ctx context.Context, userID int64) ([]*domain.Lesson, error) {
	query := `
		SELECT l.id, l.title, l.content, l.subject_id, l.class_id, l.teacher_id,
		       l.video_url, l.duration, l.is_published, l.is_approved,
		       l.views, l.created_at, l.updated_at
		FROM lessons l
		INNER JOIN bookmarks b ON b.lesson_id = l.id
		WHERE b.user_id = $1
		ORDER BY b.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list bookmarks: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.Lesson, 0, 16)
	for rows.Next() {
		l, err := scanLessonRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, l)
	}
	return results, rows.Err()
}

type lessonScanner interface {
	Scan(dest ...any) error
}

func buildLessonWhere(f repository.LessonFilter) (string, []any) {
	var where []string
	var args []any
	i := 1

	if f.TeacherID != nil {
		where = append(where, fmt.Sprintf("teacher_id = $%d", i))
		args = append(args, *f.TeacherID)
		i++
	}
	if f.ClassID != nil {
		where = append(where, fmt.Sprintf("class_id = $%d", i))
		args = append(args, *f.ClassID)
		i++
	}
	if f.SubjectID != nil {
		where = append(where, fmt.Sprintf("subject_id = $%d", i))
		args = append(args, *f.SubjectID)
		i++
	}
	if f.PublishedAt != nil {
		where = append(where, fmt.Sprintf("is_published = $%d", i))
		args = append(args, *f.PublishedAt)
		i++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf("(title ILIKE $%d OR content ILIKE $%d)", i, i))
		args = append(args, "%"+f.Search+"%")
		i++
	}

	return joinAnd(where), args
}

func scanLesson(row *sql.Row) (*domain.Lesson, error) {
	l, err := scanLessonRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return l, err
}

func scanLessonRow(s lessonScanner) (*domain.Lesson, error) {
	var (
		l         domain.Lesson
		content   sql.NullString
		subjectID sql.NullInt64
		classID   sql.NullInt64
		teacherID sql.NullInt64
		videoURL  sql.NullString
		duration  sql.NullInt64
		updatedAt sql.NullTime
	)
	err := s.Scan(
		&l.ID, &l.Title, &content, &subjectID, &classID, &teacherID,
		&videoURL, &duration, &l.IsPublished, &l.IsApproved,
		&l.Views, &l.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	if content.Valid {
		l.Content = &content.String
	}
	if subjectID.Valid {
		l.SubjectID = &subjectID.Int64
	}
	if classID.Valid {
		l.ClassID = &classID.Int64
	}
	if teacherID.Valid {
		l.TeacherID = &teacherID.Int64
	}
	if videoURL.Valid {
		l.VideoURL = &videoURL.String
	}
	if duration.Valid {
		d := int(duration.Int64)
		l.Duration = &d
	}
	if updatedAt.Valid {
		l.UpdatedAt = &updatedAt.Time
	}
	return &l, nil
}

func scanMaterial(s lessonScanner) (*domain.LessonMaterial, error) {
	var (
		m        domain.LessonMaterial
		lessonID sql.NullInt64
		fileName sql.NullString
		filePath sql.NullString
		fileType sql.NullString
		fileSize sql.NullInt64
	)
	err := s.Scan(
		&m.ID, &lessonID, &fileName, &filePath, &fileType, &fileSize,
		&m.DownloadCount, &m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if lessonID.Valid {
		m.LessonID = &lessonID.Int64
	}
	if fileName.Valid {
		m.FileName = &fileName.String
	}
	if filePath.Valid {
		m.FilePath = &filePath.String
	}
	if fileType.Valid {
		m.FileType = &fileType.String
	}
	if fileSize.Valid {
		s := int(fileSize.Int64)
		m.FileSize = &s
	}
	return &m, nil
}

func (r *LessonRepo) Count(ctx context.Context, f repository.LessonFilter) (int, error) {
	where, args := buildLessonWhere(f)
	query := `SELECT COUNT(*) FROM lessons`
	if where != "" {
		query += " WHERE " + where
	}

	var n int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&n)
	return n, err
}

var _ repository.LessonRepository = (*LessonRepo)(nil)
