package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
)

type AnalyticsRepo struct {
	db *sql.DB
}

func NewAnalyticsRepo(db *sql.DB) *AnalyticsRepo {
	return &AnalyticsRepo{db: db}
}

func (r *AnalyticsRepo) EnrollmentTrend(
	ctx context.Context,
	interval string,
	from, to time.Time,
) ([]*domain.EnrollmentPoint, error) {
	trunc, err := intervalTrunc(interval)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT date_trunc('%s', created_at) AS period, COUNT(*)::int
		FROM users
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY period
		ORDER BY period ASC
	`, trunc)

	rows, err := r.db.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("enrollment trend: %w", err)
	}
	defer rows.Close()

	points := make([]*domain.EnrollmentPoint, 0)
	for rows.Next() {
		p := &domain.EnrollmentPoint{}
		if err := rows.Scan(&p.Period, &p.Count); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (r *AnalyticsRepo) RevenueSeries(
	ctx context.Context,
	interval string,
	from, to time.Time,
) ([]*domain.RevenuePoint, error) {
	trunc, err := intervalTrunc(interval)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT
			date_trunc('%s', COALESCE(payment_date, created_at)) AS period,
			COALESCE(SUM(amount), 0)::float8 AS amount,
			COUNT(*)::int AS count
		FROM payments
		WHERE status = 'completed'
		  AND COALESCE(payment_date, created_at) >= $1
		  AND COALESCE(payment_date, created_at) <= $2
		GROUP BY period
		ORDER BY period ASC
	`, trunc)

	rows, err := r.db.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("revenue series: %w", err)
	}
	defer rows.Close()

	points := make([]*domain.RevenuePoint, 0)
	for rows.Next() {
		p := &domain.RevenuePoint{}
		if err := rows.Scan(&p.Period, &p.Amount, &p.Count); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (r *AnalyticsRepo) ScoreDistribution(ctx context.Context) (*domain.ScoreDistribution, error) {
	result := &domain.ScoreDistribution{
		Buckets: []domain.ScoreBucket{
			{Label: "0-20", Min: 0, Max: 20},
			{Label: "21-40", Min: 21, Max: 40},
			{Label: "41-60", Min: 41, Max: 60},
			{Label: "61-80", Min: 61, Max: 80},
			{Label: "81-100", Min: 81, Max: 100},
		},
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT score FROM quiz_attempts
		WHERE status = 'completed' AND score IS NOT NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("score distribution query: %w", err)
	}
	defer rows.Close()

	var scores []int
	for rows.Next() {
		var s sql.NullInt64
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		if s.Valid {
			scores = append(scores, int(s.Int64))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result.Total = len(scores)
	if result.Total == 0 {
		return result, nil
	}

	sum := 0
	for _, s := range scores {
		sum += s
		switch {
		case s <= 20:
			result.Buckets[0].Count++
		case s <= 40:
			result.Buckets[1].Count++
		case s <= 60:
			result.Buckets[2].Count++
		case s <= 80:
			result.Buckets[3].Count++
		default:
			result.Buckets[4].Count++
		}
	}
	result.Average = float64(sum) / float64(result.Total)
	result.Median = median(scores)

	for _, s := range scores {
		if s >= 50 {
			result.PassCount++
		} else {
			result.FailCount++
		}
	}
	result.PassingPct = float64(result.PassCount) / float64(result.Total) * 100

	return result, nil
}

func intervalTrunc(interval string) (string, error) {
	switch interval {
	case "day", "week", "month":
		return interval, nil
	default:
		return "", fmt.Errorf("invalid interval: %s", interval)
	}
}

func median(scores []int) int {
	if len(scores) == 0 {
		return 0
	}
	sorted := make([]int, len(scores))
	copy(sorted, scores)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

func (r *AnalyticsRepo) UserEvents(
	ctx context.Context,
	from, to time.Time,
	limit int,
) ([]*domain.UserEvent, error) {
	if limit <= 0 || limit > 100_000 {
		limit = 50_000
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id, action, created_at
		FROM activity_logs
		WHERE created_at >= $1
		  AND created_at <= $2
		  AND user_id IS NOT NULL
		ORDER BY created_at ASC
		LIMIT $3
	`, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("user events: %w", err)
	}
	defer rows.Close()

	events := make([]*domain.UserEvent, 0)
	for rows.Next() {
		var e domain.UserEvent
		if err := rows.Scan(&e.UserID, &e.Action, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

func (r *AnalyticsRepo) SignupDates(ctx context.Context) ([]*domain.SignupDate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, email, role, created_at, last_login
		FROM users
		WHERE is_active = true
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("signup dates: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.SignupDate, 0)
	for rows.Next() {
		var (
			s         domain.SignupDate
			lastLogin sql.NullTime
		)
		if err := rows.Scan(&s.UserID, &s.Email, &s.Role, &s.CreatedAt, &lastLogin); err != nil {
			return nil, err
		}
		if lastLogin.Valid {
			s.LastLogin = &lastLogin.Time
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepo) LearnerSignals(ctx context.Context) ([]*domain.LearnerSignal, error) {
	query := `
		WITH quiz_stats AS (
			SELECT
				user_id,
				COUNT(*)::int AS attempts,
				AVG(score)::float8 AS avg_score
			FROM quiz_attempts
			WHERE status = 'completed' AND score IS NOT NULL
			GROUP BY user_id
		),
		homework_stats AS (
			SELECT
				student_id,
				COUNT(*) FILTER (WHERE status IN ('submitted','graded','late'))::int AS submitted,
				COUNT(*) FILTER (WHERE status = 'graded')::int AS graded,
				AVG(grade) FILTER (WHERE grade IS NOT NULL)::float8 AS avg_grade
			FROM homework_submissions
			GROUP BY student_id
		),
		assignment_counts AS (
			SELECT
				u.id AS user_id,
				COUNT(h.id)::int AS active_assignments
			FROM users u
			LEFT JOIN homework h ON h.class_id = u.class_id AND h.is_active = true
			WHERE u.is_active = true
			  AND u.is_suspended = false
			  AND u.role IN ('learner', 'external')
			GROUP BY u.id
		),
		missing AS (
			SELECT
				u.id AS user_id,
				COUNT(h.id) FILTER (
					WHERE NOT EXISTS (
						SELECT 1 FROM homework_submissions s
						WHERE s.homework_id = h.id
						  AND s.student_id = u.id
						  AND s.status IN ('submitted','graded','late')
					)
				)::int AS missing_count
			FROM users u
			LEFT JOIN homework h ON h.class_id = u.class_id AND h.is_active = true
			WHERE u.is_active = true
			  AND u.is_suspended = false
			  AND u.role IN ('learner', 'external')
			GROUP BY u.id
		),
		last_activity AS (
			SELECT user_id, MAX(created_at) AS last_at
			FROM activity_logs
			WHERE user_id IS NOT NULL
			GROUP BY user_id
		)
		SELECT
			u.id,
			u.email,
			u.first_name,
			u.last_name,
			u.role,
			u.class_id,
			c.name,
			u.last_login,
			la.last_at,
			COALESCE(qs.attempts, 0),
			qs.avg_score,
			COALESCE(hs.submitted, 0),
			COALESCE(hs.graded, 0),
			hs.avg_grade,
			COALESCE(ac.active_assignments, 0),
			COALESCE(m.missing_count, 0)
		FROM users u
		LEFT JOIN classes c ON c.id = u.class_id
		LEFT JOIN quiz_stats qs ON qs.user_id = u.id
		LEFT JOIN homework_stats hs ON hs.student_id = u.id
		LEFT JOIN assignment_counts ac ON ac.user_id = u.id
		LEFT JOIN missing m ON m.user_id = u.id
		LEFT JOIN last_activity la ON la.user_id = u.id
		WHERE u.is_active = true
		  AND u.is_suspended = false
		  AND u.role IN ('learner', 'external')
		ORDER BY u.id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("learner signals: %w", err)
	}
	defer rows.Close()

	now := time.Now().UTC()
	out := make([]*domain.LearnerSignal, 0)

	for rows.Next() {
		var (
			s            domain.LearnerSignal
			firstName    sql.NullString
			lastName     sql.NullString
			classID      sql.NullInt64
			className    sql.NullString
			lastLogin    sql.NullTime
			lastActivity sql.NullTime
			quizAvg      sql.NullFloat64
			hwAvg        sql.NullFloat64
		)

		if err := rows.Scan(
			&s.UserID, &s.Email, &firstName, &lastName, &s.Role,
			&classID, &className, &lastLogin, &lastActivity,
			&s.QuizAttempts, &quizAvg,
			&s.HomeworkSubmitted, &s.HomeworkGraded, &hwAvg,
			&s.ActiveAssignmentsForUser, &s.MissingAssignments,
		); err != nil {
			return nil, err
		}

		if firstName.Valid {
			s.FirstName = &firstName.String
		}
		if lastName.Valid {
			s.LastName = &lastName.String
		}
		if classID.Valid {
			s.ClassID = &classID.Int64
		}
		if className.Valid {
			s.ClassName = &className.String
		}
		if lastLogin.Valid {
			s.LastLogin = &lastLogin.Time
		}
		if lastActivity.Valid {
			s.LastActivity = &lastActivity.Time
			days := int(now.Sub(lastActivity.Time).Hours() / 24)
			s.DaysSinceLastActivity = &days
		}
		if quizAvg.Valid {
			s.QuizAverage = &quizAvg.Float64
		}
		if hwAvg.Valid {
			s.HomeworkAverageGrade = &hwAvg.Float64
		}

		out = append(out, &s)
	}

	return out, rows.Err()
}
