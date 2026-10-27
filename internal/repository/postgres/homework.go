package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Hisaalu/rogele/internal/dberr"
	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository"
)

type HomeworkRepo struct {
	db *sql.DB
}

func NewHomeworkRepo(db *sql.DB) *HomeworkRepo {
	return &HomeworkRepo{db: db}
}

const homeworkColumns = `
	id, teacher_id, class_id, subject_id, title, description,
	due_date, is_active, created_at, updated_at
`

func (r *HomeworkRepo) Create(ctx context.Context, hw *domain.Homework) (*domain.Homework, error) {
	query := `
		INSERT INTO homework (
			teacher_id, class_id, subject_id, title, description,
			due_date, is_active
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING ` + homeworkColumns
	row := r.db.QueryRowContext(ctx, query,
		hw.TeacherID, hw.ClassID, hw.SubjectID, hw.Title, hw.Description,
		hw.DueDate, hw.IsActive,
	)
	result, err := scanHomework(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

func (r *HomeworkRepo) GetByID(ctx context.Context, id int64) (*domain.Homework, error) {
	query := `SELECT ` + homeworkColumns + ` FROM homework WHERE id = $1`
	return scanHomework(r.db.QueryRowContext(ctx, query, id))
}

func (r *HomeworkRepo) Update(ctx context.Context, hw *domain.Homework) error {
	query := `
		UPDATE homework SET
			class_id = $1, subject_id = $2, title = $3, description = $4,
			due_date = $5, is_active = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $7
	`
	res, err := r.db.ExecContext(ctx, query,
		hw.ClassID, hw.SubjectID, hw.Title, hw.Description,
		hw.DueDate, hw.IsActive, hw.ID,
	)
	if err != nil {
		return dberr.Translate(fmt.Errorf("update homework: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HomeworkRepo) Delete(ctx context.Context, id int64) error {
	_, _ = r.db.ExecContext(ctx, `
		DELETE FROM homework_submission_files 
		WHERE submission_id IN (SELECT id FROM homework_submissions WHERE homework_id = $1)
	`, id)

	_, _ = r.db.ExecContext(ctx, `DELETE FROM homework_submissions WHERE homework_id = $1`, id)

	_, _ = r.db.ExecContext(ctx, `DELETE FROM homework_attachments WHERE homework_id = $1`, id)

	res, err := r.db.ExecContext(ctx, `DELETE FROM homework WHERE id = $1`, id)
	if err != nil {
		return dberr.Translate(fmt.Errorf("delete homework: %w", err))
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HomeworkRepo) List(ctx context.Context, f repository.HomeworkFilter) ([]*domain.Homework, error) {
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
	if f.IsActive != nil {
		where = append(where, fmt.Sprintf("is_active = $%d", i))
		args = append(args, *f.IsActive)
		i++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", i, i))
		args = append(args, "%"+f.Search+"%")
		i++
	}

	query := `SELECT ` + homeworkColumns + ` FROM homework`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY due_date DESC"

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
		return nil, fmt.Errorf("list homework: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.Homework, 0, 16)
	for rows.Next() {
		hw, err := scanHomeworkRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, hw)
	}
	return results, rows.Err()
}

const homeworkAttachmentColumns = `
	id, homework_id, file_name, file_path, file_type, file_size, uploaded_at
`

func (r *HomeworkRepo) ListAttachments(ctx context.Context, homeworkID int64) ([]*domain.HomeworkAttachment, error) {
	query := `SELECT ` + homeworkAttachmentColumns + ` FROM homework_attachments WHERE homework_id = $1 ORDER BY uploaded_at DESC`
	rows, err := r.db.QueryContext(ctx, query, homeworkID)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.HomeworkAttachment, 0, 8)
	for rows.Next() {
		var (
			a        domain.HomeworkAttachment
			fileType sql.NullString
			fileSize sql.NullInt64
		)
		if err := rows.Scan(&a.ID, &a.HomeworkID, &a.FileName, &a.FilePath,
			&fileType, &fileSize, &a.UploadedAt); err != nil {
			return nil, err
		}
		if fileType.Valid {
			a.FileType = &fileType.String
		}
		if fileSize.Valid {
			s := int(fileSize.Int64)
			a.FileSize = &s
		}
		results = append(results, &a)
	}
	return results, rows.Err()
}

func (r *HomeworkRepo) CreateAttachment(ctx context.Context, a *domain.HomeworkAttachment) (*domain.HomeworkAttachment, error) {
	query := `
		INSERT INTO homework_attachments (homework_id, file_name, file_path, file_type, file_size)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING ` + homeworkAttachmentColumns

	var (
		out      domain.HomeworkAttachment
		fileType sql.NullString
		fileSize sql.NullInt64
	)
	err := r.db.QueryRowContext(ctx, query,
		a.HomeworkID, a.FileName, a.FilePath, a.FileType, a.FileSize,
	).Scan(&out.ID, &out.HomeworkID, &out.FileName, &out.FilePath,
		&fileType, &fileSize, &out.UploadedAt)
	if err != nil {
		return nil, dberr.Translate(fmt.Errorf("create attachment: %w", err))
	}
	if fileType.Valid {
		out.FileType = &fileType.String
	}
	if fileSize.Valid {
		s := int(fileSize.Int64)
		out.FileSize = &s
	}
	return &out, nil
}

func (r *HomeworkRepo) DeleteAttachment(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM homework_attachments WHERE id = $1`, id)
	if err != nil {
		return dberr.Translate(fmt.Errorf("delete attachment: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const submissionColumns = `
	id, homework_id, student_id, text_answer, grade, feedback, status,
	submitted_at, graded_at, created_at, updated_at
`

func (r *HomeworkRepo) CreateSubmission(ctx context.Context, s *domain.HomeworkSubmission) (*domain.HomeworkSubmission, error) {
	query := `
		INSERT INTO homework_submissions (
			homework_id, student_id, text_answer, status, submitted_at
		) VALUES ($1,$2,$3,$4,$5)
		RETURNING ` + submissionColumns
	row := r.db.QueryRowContext(ctx, query,
		s.HomeworkID, s.StudentID, s.TextAnswer, s.Status, s.SubmittedAt,
	)
	result, err := scanSubmission(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

func (r *HomeworkRepo) GetSubmissionByID(ctx context.Context, id int64) (*domain.HomeworkSubmission, error) {
	query := `SELECT ` + submissionColumns + ` FROM homework_submissions WHERE id = $1`
	return scanSubmission(r.db.QueryRowContext(ctx, query, id))
}

func (r *HomeworkRepo) GetSubmission(ctx context.Context, homeworkID, studentID int64) (*domain.HomeworkSubmission, error) {
	query := `
		SELECT id, homework_id, student_id, text_answer, grade, feedback, status, submitted_at
		FROM homework_submissions
		WHERE homework_id = $1 AND student_id = $2
	`
	var (
		s           domain.HomeworkSubmission
		textAnswer  sql.NullString
		grade       sql.NullFloat64
		feedback    sql.NullString
		submittedAt sql.NullTime
	)

	err := r.db.QueryRowContext(ctx, query, homeworkID, studentID).Scan(
		&s.ID, &s.HomeworkID, &s.StudentID, &textAnswer, &grade, &feedback, &s.Status, &submittedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &s, nil
}

func (r *HomeworkRepo) ListSubmissions(ctx context.Context, homeworkID int64) ([]*domain.HomeworkSubmission, error) {
	query := `SELECT ` + submissionColumns + ` FROM homework_submissions WHERE homework_id = $1 ORDER BY submitted_at DESC NULLS LAST`
	rows, err := r.db.QueryContext(ctx, query, homeworkID)
	if err != nil {
		return nil, fmt.Errorf("list submissions: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.HomeworkSubmission, 0, 16)
	for rows.Next() {
		s, err := scanSubmissionRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

func (r *HomeworkRepo) ListSubmissionsByStudent(ctx context.Context, studentID int64) ([]*domain.HomeworkSubmission, error) {
	query := `SELECT ` + submissionColumns + ` FROM homework_submissions WHERE student_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, studentID)
	if err != nil {
		return nil, fmt.Errorf("list student submissions: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.HomeworkSubmission, 0, 16)
	for rows.Next() {
		s, err := scanSubmissionRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

func (r *HomeworkRepo) UpdateSubmission(ctx context.Context, s *domain.HomeworkSubmission) error {
	query := `
		UPDATE homework_submissions SET
			text_answer = $1, grade = $2, feedback = $3, status = $4,
			submitted_at = $5, graded_at = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $7
	`
	res, err := r.db.ExecContext(ctx, query,
		s.TextAnswer, s.Grade, s.Feedback, s.Status,
		s.SubmittedAt, s.GradedAt, s.ID,
	)
	if err != nil {
		return dberr.Translate(fmt.Errorf("update submission: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HomeworkRepo) DeleteSubmission(ctx context.Context, id int64) error {
	_, _ = r.db.ExecContext(ctx, `DELETE FROM homework_submission_files WHERE submission_id = $1`, id)

	res, err := r.db.ExecContext(ctx, `DELETE FROM homework_submissions WHERE id = $1`, id)
	if err != nil {
		return dberr.Translate(fmt.Errorf("delete submission: %w", err))
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const submissionFileColumns = `
	id, submission_id, file_name, file_path, file_type, file_size, uploaded_at
`

func (r *HomeworkRepo) ListSubmissionFiles(ctx context.Context, submissionID int64) ([]*domain.HomeworkSubmissionFile, error) {
	query := `SELECT ` + submissionFileColumns + ` FROM homework_submission_files WHERE submission_id = $1 ORDER BY uploaded_at DESC`
	rows, err := r.db.QueryContext(ctx, query, submissionID)
	if err != nil {
		return nil, fmt.Errorf("list submission files: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.HomeworkSubmissionFile, 0, 8)
	for rows.Next() {
		var (
			f        domain.HomeworkSubmissionFile
			fileType sql.NullString
			fileSize sql.NullInt64
		)
		if err := rows.Scan(&f.ID, &f.SubmissionID, &f.FileName, &f.FilePath,
			&fileType, &fileSize, &f.UploadedAt); err != nil {
			return nil, err
		}
		if fileType.Valid {
			f.FileType = &fileType.String
		}
		if fileSize.Valid {
			s := int(fileSize.Int64)
			f.FileSize = &s
		}
		results = append(results, &f)
	}
	return results, rows.Err()
}

func (r *HomeworkRepo) CreateSubmissionFile(ctx context.Context, f *domain.HomeworkSubmissionFile) (*domain.HomeworkSubmissionFile, error) {
	query := `
		INSERT INTO homework_submission_files (submission_id, file_name, file_path, file_type, file_size)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING ` + submissionFileColumns

	var (
		out      domain.HomeworkSubmissionFile
		fileType sql.NullString
		fileSize sql.NullInt64
	)
	err := r.db.QueryRowContext(ctx, query,
		f.SubmissionID, f.FileName, f.FilePath, f.FileType, f.FileSize,
	).Scan(&out.ID, &out.SubmissionID, &out.FileName, &out.FilePath,
		&fileType, &fileSize, &out.UploadedAt)
	if err != nil {
		return nil, dberr.Translate(fmt.Errorf("create submission file: %w", err))
	}
	if fileType.Valid {
		out.FileType = &fileType.String
	}
	if fileSize.Valid {
		s := int(fileSize.Int64)
		out.FileSize = &s
	}
	return &out, nil
}

func (r *HomeworkRepo) DeleteSubmissionFile(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM homework_submission_files WHERE id = $1`, id)
	if err != nil {
		return dberr.Translate(fmt.Errorf("delete submission file: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type homeworkScanner interface {
	Scan(dest ...any) error
}

func scanHomework(row *sql.Row) (*domain.Homework, error) {
	hw, err := scanHomeworkRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return hw, err
}

func scanHomeworkRow(s homeworkScanner) (*domain.Homework, error) {
	var (
		hw        domain.Homework
		desc      sql.NullString
		updatedAt sql.NullTime
	)
	err := s.Scan(
		&hw.ID, &hw.TeacherID, &hw.ClassID, &hw.SubjectID, &hw.Title, &desc,
		&hw.DueDate, &hw.IsActive, &hw.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		hw.Description = &desc.String
	}
	if updatedAt.Valid {
		hw.UpdatedAt = &updatedAt.Time
	}
	return &hw, nil
}

func scanSubmission(row *sql.Row) (*domain.HomeworkSubmission, error) {
	s, err := scanSubmissionRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return s, err
}

func scanSubmissionRow(s homeworkScanner) (*domain.HomeworkSubmission, error) {
	var (
		sub         domain.HomeworkSubmission
		textAnswer  sql.NullString
		grade       sql.NullFloat64
		feedback    sql.NullString
		submittedAt sql.NullTime
		gradedAt    sql.NullTime
		updatedAt   sql.NullTime
	)
	err := s.Scan(
		&sub.ID, &sub.HomeworkID, &sub.StudentID, &textAnswer, &grade, &feedback,
		&sub.Status, &submittedAt, &gradedAt, &sub.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	if textAnswer.Valid {
		sub.TextAnswer = &textAnswer.String
	}
	if grade.Valid {
		sub.Grade = &grade.Float64
	}
	if feedback.Valid {
		sub.Feedback = &feedback.String
	}
	if submittedAt.Valid {
		sub.SubmittedAt = &submittedAt.Time
	}
	if gradedAt.Valid {
		sub.GradedAt = &gradedAt.Time
	}
	if updatedAt.Valid {
		sub.UpdatedAt = &updatedAt.Time
	}
	return &sub, nil
}

func (r *HomeworkRepo) GetAttachmentByID(ctx context.Context, id int64) (*domain.HomeworkAttachment, error) {
	query := `SELECT ` + homeworkAttachmentColumns + ` FROM homework_attachments WHERE id = $1`
	var (
		a        domain.HomeworkAttachment
		fileType sql.NullString
		fileSize sql.NullInt64
	)
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID, &a.HomeworkID, &a.FileName, &a.FilePath, &fileType, &fileSize, &a.UploadedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if fileType.Valid {
		a.FileType = &fileType.String
	}
	if fileSize.Valid {
		s := int(fileSize.Int64)
		a.FileSize = &s
	}
	return &a, nil
}

func (r *HomeworkRepo) GetSubmissionFileByID(ctx context.Context, id int64) (*domain.HomeworkSubmissionFile, error) {
	query := `SELECT ` + submissionFileColumns + ` FROM homework_submission_files WHERE id = $1`
	var (
		f        domain.HomeworkSubmissionFile
		fileType sql.NullString
		fileSize sql.NullInt64
	)
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&f.ID, &f.SubmissionID, &f.FileName, &f.FilePath, &fileType, &fileSize, &f.UploadedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if fileType.Valid {
		f.FileType = &fileType.String
	}
	if fileSize.Valid {
		s := int(fileSize.Int64)
		f.FileSize = &s
	}
	return &f, nil
}

var _ repository.HomeworkRepository = (*HomeworkRepo)(nil)
