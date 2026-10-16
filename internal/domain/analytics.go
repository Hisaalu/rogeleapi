package domain

import "time"

type EnrollmentPoint struct {
	Period time.Time `json:"period"`
	Count  int       `json:"count"`
}

type RevenuePoint struct {
	Period time.Time `json:"period"`
	Amount float64   `json:"amount"`
	Count  int       `json:"count"`
}

type ScoreBucket struct {
	Label string `json:"label"`
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Count int    `json:"count"`
}

type ScoreDistribution struct {
	Buckets    []ScoreBucket `json:"buckets"`
	Total      int           `json:"total"`
	Average    float64       `json:"average"`
	Median     int           `json:"median"`
	PassCount  int           `json:"pass_count"`
	FailCount  int           `json:"fail_count"`
	PassingPct float64       `json:"passing_pct"`
}

type EnrollmentTrend struct {
	Interval string             `json:"interval"`
	From     time.Time          `json:"from"`
	To       time.Time          `json:"to"`
	Points   []*EnrollmentPoint `json:"points"`
	Total    int                `json:"total"`
}

type RevenueSeries struct {
	Interval   string          `json:"interval"`
	From       time.Time       `json:"from"`
	To         time.Time       `json:"to"`
	Points     []*RevenuePoint `json:"points"`
	Total      float64         `json:"total"`
	TotalCount int             `json:"total_count"`
}
