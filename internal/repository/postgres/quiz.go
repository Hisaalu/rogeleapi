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

type QuizRepo struct {
	db *sql.DB
}

func NewQuizRepo(db *sql.DB) *QuizRepo {
	return &QuizRepo{db: db}
}

const quizColumns = `
	id, title, description, subject_id, class_id, teacher_id,
	time_limit, passing_score, max_attempts, is_published, is_approved,
	approved_at, approved_by, start_date, end_date, status,
	created_at, updated_at
`

func (r *QuizRepo) Create(ctx context.Context, q *domain.Quiz) (*domain.Quiz, error) {
	query := `
		INSERT INTO quizzes (
			title, description, subject_id, class_id, teacher_id,
			time_limit, passing_score, max_attempts, is_published,
			start_date, end_date, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING ` + quizColumns
	row := r.db.QueryRowContext(ctx, query,
		q.Title, q.Description, q.SubjectID, q.ClassID, q.TeacherID,
		q.TimeLimit, q.PassingScore, q.MaxAttempts, q.IsPublished,
		q.StartDate, q.EndDate, q.Status,
	)
	result, err := scanQuiz(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

func (r *QuizRepo) GetByID(ctx context.Context, id int64) (*domain.Quiz, error) {
	query := `SELECT ` + quizColumns + ` FROM quizzes WHERE id = $1`
	return scanQuiz(r.db.QueryRowContext(ctx, query, id))
}

func (r *QuizRepo) Update(ctx context.Context, q *domain.Quiz) error {
	query := `
		UPDATE quizzes SET
			title = $1, description = $2, subject_id = $3, class_id = $4,
			time_limit = $5, passing_score = $6, max_attempts = $7,
			is_published = $8, is_approved = $9, approved_at = $10,
			approved_by = $11, start_date = $12, end_date = $13, status = $14,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $15
	`
	res, err := r.db.ExecContext(ctx, query,
		q.Title, q.Description, q.SubjectID, q.ClassID,
		q.TimeLimit, q.PassingScore, q.MaxAttempts,
		q.IsPublished, q.IsApproved, q.ApprovedAt,
		q.ApprovedBy, q.StartDate, q.EndDate, q.Status,
		q.ID,
	)
	if err != nil {
		return dberr.Translate(fmt.Errorf("update quiz: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *QuizRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM quizzes WHERE id = $1`, id)
	if err != nil {
		return dberr.Translate(fmt.Errorf("delete quiz: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *QuizRepo) List(ctx context.Context, f repository.QuizFilter) ([]*domain.Quiz, error) {
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
	if f.Published != nil {
		where = append(where, fmt.Sprintf("is_published = $%d", i))
		args = append(args, *f.Published)
		i++
	}

	if f.Search != "" {
		where = append(where, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", i, i))
		args = append(args, "%"+f.Search+"%")
		i++
	}

	query := `SELECT ` + quizColumns + ` FROM quizzes`
	if len(where) > 0 {
		query += " WHERE " + joinAnd(where)
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
		return nil, fmt.Errorf("list quizzes: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.Quiz, 0, 16)
	for rows.Next() {
		q, err := scanQuizRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, q)
	}
	return results, rows.Err()
}

const questionColumns = `
	id, quiz_id, question, option_a, option_b, option_c, option_d,
	correct_answer, points, explanation, created_at
`

func (r *QuizRepo) ListQuestions(ctx context.Context, quizID int64) ([]*domain.QuizQuestion, error) {
	query := `SELECT ` + questionColumns + ` FROM quiz_questions WHERE quiz_id = $1 ORDER BY id`
	rows, err := r.db.QueryContext(ctx, query, quizID)
	if err != nil {
		return nil, fmt.Errorf("list questions: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.QuizQuestion, 0, 16)
	for rows.Next() {
		var (
			q       domain.QuizQuestion
			qid     sql.NullInt64
			optA    sql.NullString
			optB    sql.NullString
			optC    sql.NullString
			optD    sql.NullString
			correct sql.NullString
			expl    sql.NullString
		)
		if err := rows.Scan(&q.ID, &qid, &q.Question, &optA, &optB, &optC, &optD,
			&correct, &q.Points, &expl, &q.CreatedAt); err != nil {
			return nil, err
		}
		if qid.Valid {
			q.QuizID = &qid.Int64
		}
		if optA.Valid {
			q.OptionA = &optA.String
		}
		if optB.Valid {
			q.OptionB = &optB.String
		}
		if optC.Valid {
			q.OptionC = &optC.String
		}
		if optD.Valid {
			q.OptionD = &optD.String
		}
		if correct.Valid {
			q.CorrectAnswer = &correct.String
		}
		if expl.Valid {
			q.Explanation = &expl.String
		}
		results = append(results, &q)
	}
	return results, rows.Err()
}

func (r *QuizRepo) CreateQuestion(ctx context.Context, q *domain.QuizQuestion) (*domain.QuizQuestion, error) {
	query := `
		INSERT INTO quiz_questions (
			quiz_id, question, option_a, option_b, option_c, option_d,
			correct_answer, points, explanation
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING ` + questionColumns
	var (
		out     domain.QuizQuestion
		qid     sql.NullInt64
		optA    sql.NullString
		optB    sql.NullString
		optC    sql.NullString
		optD    sql.NullString
		correct sql.NullString
		expl    sql.NullString
	)
	err := r.db.QueryRowContext(ctx, query,
		q.QuizID, q.Question, q.OptionA, q.OptionB, q.OptionC, q.OptionD,
		q.CorrectAnswer, q.Points, q.Explanation,
	).Scan(&out.ID, &qid, &out.Question, &optA, &optB, &optC, &optD,
		&correct, &out.Points, &expl, &out.CreatedAt)
	if err != nil {
		return nil, dberr.Translate(fmt.Errorf("create question: %w", err))
	}
	if qid.Valid {
		out.QuizID = &qid.Int64
	}
	if optA.Valid {
		out.OptionA = &optA.String
	}
	if optB.Valid {
		out.OptionB = &optB.String
	}
	if optC.Valid {
		out.OptionC = &optC.String
	}
	if optD.Valid {
		out.OptionD = &optD.String
	}
	if correct.Valid {
		out.CorrectAnswer = &correct.String
	}
	if expl.Valid {
		out.Explanation = &expl.String
	}
	return &out, nil
}

func (r *QuizRepo) DeleteQuestion(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM quiz_questions WHERE id = $1`, id)
	if err != nil {
		return dberr.Translate(fmt.Errorf("delete question: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const attemptColumns = `
	id, quiz_id, user_id, score, total_questions, correct_answers,
	time_taken, status, started_at, completed_at
`

func (r *QuizRepo) CreateAttempt(ctx context.Context, a *domain.QuizAttempt) (*domain.QuizAttempt, error) {
	query := `
		INSERT INTO quiz_attempts (
			quiz_id, user_id, status, started_at
		) VALUES ($1,$2,$3,$4)
		RETURNING ` + attemptColumns
	row := r.db.QueryRowContext(ctx, query, a.QuizID, a.UserID, a.Status, a.StartedAt)
	result, err := scanAttempt(row)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return result, nil
}

func (r *QuizRepo) GetAttempt(ctx context.Context, id int64) (*domain.QuizAttempt, error) {
	query := `SELECT ` + attemptColumns + ` FROM quiz_attempts WHERE id = $1`
	return scanAttempt(r.db.QueryRowContext(ctx, query, id))
}

func (r *QuizRepo) GetInProgressAttempt(ctx context.Context, quizID, userID int64) (*domain.QuizAttempt, error) {
	query := `SELECT ` + attemptColumns + ` FROM quiz_attempts
	          WHERE quiz_id = $1 AND user_id = $2 AND status = 'in_progress'
	          ORDER BY started_at DESC LIMIT 1`
	return scanAttempt(r.db.QueryRowContext(ctx, query, quizID, userID))
}

func (r *QuizRepo) UpdateAttempt(ctx context.Context, a *domain.QuizAttempt) error {
	query := `
		UPDATE quiz_attempts SET
			score = $1, total_questions = $2, correct_answers = $3,
			time_taken = $4, status = $5, completed_at = $6
		WHERE id = $7
	`
	res, err := r.db.ExecContext(ctx, query,
		a.Score, a.TotalQuestions, a.CorrectAnswers,
		a.TimeTaken, a.Status, a.CompletedAt, a.ID,
	)
	if err != nil {
		return dberr.Translate(fmt.Errorf("update attempt: %w", err))
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *QuizRepo) CountAttempts(ctx context.Context, quizID, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM quiz_attempts WHERE quiz_id = $1 AND user_id = $2 AND status IN ('completed','expired')`,
		quizID, userID,
	).Scan(&n)
	return n, err
}

func (r *QuizRepo) ListAttemptsByUser(ctx context.Context, userID int64) ([]*domain.QuizAttempt, error) {
	query := `SELECT ` + attemptColumns + ` FROM quiz_attempts WHERE user_id = $1 ORDER BY started_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list attempts: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.QuizAttempt, 0, 16)
	for rows.Next() {
		a, err := scanAttemptRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, rows.Err()
}

func (r *QuizRepo) SaveAnswer(ctx context.Context, a *domain.QuizAttemptAnswer) error {
	query := `
		INSERT INTO quiz_attempt_answers (attempt_id, question_id, selected_answer, is_correct)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (attempt_id, question_id) DO UPDATE
		SET selected_answer = EXCLUDED.selected_answer,
		    is_correct = EXCLUDED.is_correct
	`
	_, err := r.db.ExecContext(ctx, query, a.AttemptID, a.QuestionID, a.SelectedAnswer, a.IsCorrect)
	if err != nil {
		return dberr.Translate(fmt.Errorf("save answer: %w", err))
	}
	return nil
}

func (r *QuizRepo) DeleteAttempt(ctx context.Context, attemptID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM quiz_attempts WHERE id = $1`, attemptID)
	return dberr.Translate(err)
}

func (r *QuizRepo) ListAnswers(ctx context.Context, attemptID int64) ([]*domain.QuizAttemptAnswer, error) {
	query := `SELECT id, attempt_id, question_id, selected_answer, is_correct, created_at
	          FROM quiz_attempt_answers WHERE attempt_id = $1 ORDER BY id`
	rows, err := r.db.QueryContext(ctx, query, attemptID)
	if err != nil {
		return nil, fmt.Errorf("list answers: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.QuizAttemptAnswer, 0, 16)
	for rows.Next() {
		var (
			a        domain.QuizAttemptAnswer
			selected sql.NullInt64
		)
		if err := rows.Scan(&a.ID, &a.AttemptID, &a.QuestionID,
			&selected, &a.IsCorrect, &a.CreatedAt); err != nil {
			return nil, err
		}
		if selected.Valid {
			v := int(selected.Int64)
			a.SelectedAnswer = &v
		}
		results = append(results, &a)
	}
	return results, rows.Err()
}

func scanQuiz(row *sql.Row) (*domain.Quiz, error) {
	q, err := scanQuizRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return q, err
}

type quizScanner interface {
	Scan(dest ...any) error
}

func scanQuizRow(s quizScanner) (*domain.Quiz, error) {
	var (
		q          domain.Quiz
		desc       sql.NullString
		subjectID  sql.NullInt64
		classID    sql.NullInt64
		teacherID  sql.NullInt64
		timeLimit  sql.NullInt64
		approvedAt sql.NullTime
		approvedBy sql.NullInt64
		startDate  sql.NullTime
		endDate    sql.NullTime
	)
	err := s.Scan(
		&q.ID, &q.Title, &desc, &subjectID, &classID, &teacherID,
		&timeLimit, &q.PassingScore, &q.MaxAttempts, &q.IsPublished, &q.IsApproved,
		&approvedAt, &approvedBy, &startDate, &endDate, &q.Status,
		&q.CreatedAt, &q.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		q.Description = &desc.String
	}
	if subjectID.Valid {
		q.SubjectID = &subjectID.Int64
	}
	if classID.Valid {
		q.ClassID = &classID.Int64
	}
	if teacherID.Valid {
		q.TeacherID = &teacherID.Int64
	}
	if timeLimit.Valid {
		v := int(timeLimit.Int64)
		q.TimeLimit = &v
	}
	if approvedAt.Valid {
		q.ApprovedAt = &approvedAt.Time
	}
	if approvedBy.Valid {
		q.ApprovedBy = &approvedBy.Int64
	}
	if startDate.Valid {
		q.StartDate = &startDate.Time
	}
	if endDate.Valid {
		q.EndDate = &endDate.Time
	}
	return &q, nil
}

func scanAttempt(row *sql.Row) (*domain.QuizAttempt, error) {
	a, err := scanAttemptRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return a, err
}

func scanAttemptRow(s quizScanner) (*domain.QuizAttempt, error) {
	var (
		a         domain.QuizAttempt
		quizID    sql.NullInt64
		userID    sql.NullInt64
		score     sql.NullInt64
		totalQs   sql.NullInt64
		correct   sql.NullInt64
		timeTaken sql.NullInt64
		completed sql.NullTime
	)
	err := s.Scan(
		&a.ID, &quizID, &userID, &score, &totalQs, &correct,
		&timeTaken, &a.Status, &a.StartedAt, &completed,
	)
	if err != nil {
		return nil, err
	}
	if quizID.Valid {
		a.QuizID = &quizID.Int64
	}
	if userID.Valid {
		a.UserID = &userID.Int64
	}
	if score.Valid {
		v := int(score.Int64)
		a.Score = &v
	}
	if totalQs.Valid {
		v := int(totalQs.Int64)
		a.TotalQuestions = &v
	}
	if correct.Valid {
		v := int(correct.Int64)
		a.CorrectAnswers = &v
	}
	if timeTaken.Valid {
		v := int(timeTaken.Int64)
		a.TimeTaken = &v
	}
	if completed.Valid {
		a.CompletedAt = &completed.Time
	}
	return &a, nil
}

func joinAnd(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += " AND " + p
	}
	return out
}

func (r *QuizRepo) ListAttemptsByQuiz(ctx context.Context, quizID int64) ([]*domain.QuizAttempt, error) {
	query := `SELECT ` + attemptColumns + ` FROM quiz_attempts WHERE quiz_id = $1 ORDER BY started_at DESC`
	rows, err := r.db.QueryContext(ctx, query, quizID)
	if err != nil {
		return nil, fmt.Errorf("list attempts by quiz: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.QuizAttempt, 0, 16)
	for rows.Next() {
		a, err := scanAttemptRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, rows.Err()
}

func (r *QuizRepo) ListAttemptsWithStudents(ctx context.Context, quizID int64) ([]*domain.QuizAttempt, error) {
	query := `
		SELECT
			a.id, a.quiz_id, a.user_id, a.score, a.total_questions, a.correct_answers,
			a.time_taken, a.status, a.started_at, a.completed_at,
			u.email, u.first_name, u.last_name, u.registration_number
		FROM quiz_attempts a
		LEFT JOIN users u ON u.id = a.user_id
		WHERE a.quiz_id = $1
		ORDER BY a.started_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, quizID)
	if err != nil {
		return nil, fmt.Errorf("list attempts with students: %w", err)
	}
	defer rows.Close()

	results := make([]*domain.QuizAttempt, 0, 16)
	for rows.Next() {
		var (
			a         domain.QuizAttempt
			quizID    sql.NullInt64
			userID    sql.NullInt64
			score     sql.NullInt64
			totalQs   sql.NullInt64
			correct   sql.NullInt64
			timeTaken sql.NullInt64
			completed sql.NullTime
			email     sql.NullString
			firstName sql.NullString
			lastName  sql.NullString
			regNo     sql.NullString
		)
		if err := rows.Scan(
			&a.ID, &quizID, &userID, &score, &totalQs, &correct,
			&timeTaken, &a.Status, &a.StartedAt, &completed,
			&email, &firstName, &lastName, &regNo,
		); err != nil {
			return nil, err
		}
		if quizID.Valid {
			a.QuizID = &quizID.Int64
		}
		if userID.Valid {
			a.UserID = &userID.Int64
		}
		if score.Valid {
			v := int(score.Int64)
			a.Score = &v
		}
		if totalQs.Valid {
			v := int(totalQs.Int64)
			a.TotalQuestions = &v
		}
		if correct.Valid {
			v := int(correct.Int64)
			a.CorrectAnswers = &v
		}
		if timeTaken.Valid {
			v := int(timeTaken.Int64)
			a.TimeTaken = &v
		}
		if completed.Valid {
			a.CompletedAt = &completed.Time
		}
		if email.Valid {
			a.StudentEmail = &email.String
		}
		if firstName.Valid {
			a.StudentFirstName = &firstName.String
		}
		if lastName.Valid {
			a.StudentLastName = &lastName.String
		}
		if regNo.Valid {
			a.StudentRegNo = &regNo.String
		}
		results = append(results, &a)
	}
	return results, rows.Err()
}

func (r *QuizRepo) DeleteAllAttempts(ctx context.Context, quizID int64) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM quiz_attempt_answers
		WHERE attempt_id IN (SELECT id FROM quiz_attempts WHERE quiz_id = $1)
	`, quizID)
	if err != nil {
		return fmt.Errorf("delete attempt answers: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM quiz_attempts WHERE quiz_id = $1`, quizID)
	if err != nil {
		return fmt.Errorf("delete attempts: %w", err)
	}
	return nil
}

func (r *QuizRepo) CountQuestions(ctx context.Context, quizID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM quiz_questions WHERE quiz_id = $1`, quizID).Scan(&n)
	return n, err
}

func (r *QuizRepo) CountAttemptsForQuiz(ctx context.Context, quizID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM quiz_attempts WHERE quiz_id = $1`, quizID).Scan(&n)
	return n, err
}

func (r *QuizRepo) GetQuizForUpdate(ctx context.Context, quizID int64) (*domain.Quiz, error) {
	query := `SELECT ` + quizColumns + ` FROM quizzes WHERE id = $1 FOR UPDATE`
	return scanQuiz(r.db.QueryRowContext(ctx, query, quizID))
}

func (r *QuizRepo) GetGlobalAverageScore(ctx context.Context) (int, error) {
	var avg float64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(score), 0)
		FROM quiz_attempts
		WHERE status = 'completed' AND score IS NOT NULL
	`).Scan(&avg)
	if err != nil {
		return 0, err
	}
	return int(avg), nil
}

type DailyPerformance struct {
	Date         string  `json:"date"`
	AvgScore     float64 `json:"avg_score"`
	AttemptCount int     `json:"attempt_count"`
}

func (r *QuizRepo) GetQuizPerformanceByDay(ctx context.Context, days int) ([]repository.DailyPerformance, error) {
	query := `
		SELECT 
			TO_CHAR(completed_at, 'YYYY-MM-DD') as date,
			COALESCE(AVG(score), 0) as avg_score,
			COUNT(*) as attempt_count
		FROM quiz_attempts
		WHERE status = 'completed' 
		  AND completed_at >= NOW() - ($1 || ' days')::INTERVAL
		GROUP BY TO_CHAR(completed_at, 'YYYY-MM-DD')
		ORDER BY date ASC
	`
	rows, err := r.db.QueryContext(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("get quiz performance by day: %w", err)
	}
	defer rows.Close()

	var results []repository.DailyPerformance
	for rows.Next() {
		var p repository.DailyPerformance
		if err := rows.Scan(&p.Date, &p.AvgScore, &p.AttemptCount); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

func buildQuizWhere(f repository.QuizFilter) (string, []any) {
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
	if f.Published != nil {
		where = append(where, fmt.Sprintf("is_published = $%d", i))
		args = append(args, *f.Published)
		i++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", i, i))
		args = append(args, "%"+f.Search+"%")
		i++
	}

	if f.MinQuestions > 0 {
		where = append(where, fmt.Sprintf(
			"(SELECT COUNT(*) FROM quiz_questions WHERE quiz_id = quizzes.id) >= $%d", i,
		))
		args = append(args, f.MinQuestions)
		i++
	}

	return joinAnd(where), args
}

func (r *QuizRepo) Count(ctx context.Context, f repository.QuizFilter) (int, error) {
	where, args := buildQuizWhere(f)
	query := `SELECT COUNT(*) FROM quizzes`
	if len(where) > 0 {
		query += " WHERE " + where
	}
	var n int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&n)
	return n, err
}

var _ repository.QuizRepository = (*QuizRepo)(nil)
