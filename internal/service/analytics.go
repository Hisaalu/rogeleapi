package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/repository/postgres"
)

type AnalyticsService struct {
	repo *postgres.AnalyticsRepo
}

func NewAnalyticsService(repo *postgres.AnalyticsRepo) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

type DateRange struct {
	From time.Time
	To   time.Time
}

func DefaultRange() DateRange {
	to := time.Now().UTC()
	from := to.AddDate(0, -6, 0)
	return DateRange{From: from, To: to}
}

func (s *AnalyticsService) EnrollmentTrend(
	ctx context.Context,
	interval string,
	r DateRange,
) (*domain.EnrollmentTrend, error) {
	if err := validateInterval(interval); err != nil {
		return nil, err
	}

	points, err := s.repo.EnrollmentTrend(ctx, interval, r.From, r.To)
	if err != nil {
		return nil, err
	}

	total := 0
	for _, p := range points {
		total += p.Count
	}

	return &domain.EnrollmentTrend{
		Interval: interval,
		From:     r.From,
		To:       r.To,
		Points:   points,
		Total:    total,
	}, nil
}

func (s *AnalyticsService) RevenueSeries(
	ctx context.Context,
	interval string,
	r DateRange,
) (*domain.RevenueSeries, error) {
	if err := validateInterval(interval); err != nil {
		return nil, err
	}

	points, err := s.repo.RevenueSeries(ctx, interval, r.From, r.To)
	if err != nil {
		return nil, err
	}

	var total float64
	totalCount := 0
	for _, p := range points {
		total += p.Amount
		totalCount += p.Count
	}

	return &domain.RevenueSeries{
		Interval:   interval,
		From:       r.From,
		To:         r.To,
		Points:     points,
		Total:      total,
		TotalCount: totalCount,
	}, nil
}

func (s *AnalyticsService) ScoreDistribution(ctx context.Context) (*domain.ScoreDistribution, error) {
	return s.repo.ScoreDistribution(ctx)
}

func validateInterval(interval string) error {
	switch interval {
	case "day", "week", "month":
		return nil
	default:
		return fmt.Errorf("%w: interval must be day, week, or month", domain.ErrInvalidInput)
	}
}

func (s *AnalyticsService) UserEvents(
	ctx context.Context,
	r DateRange,
	limit int,
) ([]*domain.UserEvent, error) {
	return s.repo.UserEvents(ctx, r.From, r.To, limit)
}

func (s *AnalyticsService) SignupDates(ctx context.Context) ([]*domain.SignupDate, error) {
	return s.repo.SignupDates(ctx)
}

func (s *AnalyticsService) LearnerSignals(ctx context.Context) ([]*domain.LearnerSignal, error) {
	return s.repo.LearnerSignals(ctx)
}
