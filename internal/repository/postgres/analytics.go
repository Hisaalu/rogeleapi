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

// EnrollmentTrend returns signup counts bucketed by day, week, or month.
// `interval` must be one of: day, week, month.
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

	var points []*domain.EnrollmentPoint
	for rows.Next() {
		p := &domain.EnrollmentPoint{}
		if err := rows.Scan(&p.Period, &p.Count); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// RevenueSeries returns completed payments bucketed by day, week, or month.
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

	var points []*domain.RevenuePoint
	for rows.Next() {
		p := &domain.RevenuePoint{}
		if err := rows.Scan(&p.Period, &p.Amount, &p.Count); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// ScoreDistribution buckets completed quiz attempts into five ranges.
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

	// Passing threshold is hardcoded to 50 for the aggregate view.
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
	// Simple insertion sort — the dataset is small.
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
