package mailer

import (
	"context"
	"database/sql"
	"sync"
	"time"
)

type Branding struct {
	SchoolName   string
	Tagline      string
	LogoURL      string
	ColorPrimary string
	ColorAccent  string
	WebsiteURL   string
	SupportEmail string
}

var defaultBranding = Branding{
	SchoolName:   "Rogele",
	Tagline:      "E-Learning Platform",
	ColorPrimary: "#7f2677",
	ColorAccent:  "#f06724",
}

type BrandingProvider struct {
	db        *sql.DB
	mu        sync.RWMutex
	cached    Branding
	expiresAt time.Time
	ttl       time.Duration
}

func NewBrandingProvider(db *sql.DB) *BrandingProvider {
	return &BrandingProvider{
		db:  db,
		ttl: 60 * time.Second,
	}
}

func (p *BrandingProvider) Get(ctx context.Context) Branding {
	p.mu.RLock()
	if time.Now().Before(p.expiresAt) {
		b := p.cached
		p.mu.RUnlock()
		return b
	}
	p.mu.RUnlock()

	b := p.load(ctx)

	p.mu.Lock()
	p.cached = b
	p.expiresAt = time.Now().Add(p.ttl)
	p.mu.Unlock()

	return b
}

func (p *BrandingProvider) load(ctx context.Context) Branding {
	b := defaultBranding

	rows, err := p.db.QueryContext(ctx, `
		SELECT setting_key, setting_value
		FROM settings
		WHERE setting_key LIKE 'branding.%'
	`)
	if err != nil {
		return b
	}
	defer rows.Close()

	values := make(map[string]string)
	for rows.Next() {
		var (
			key   string
			value sql.NullString
		)
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		if value.Valid && value.String != "" {
			values[key] = value.String
		}
	}

	if v := values["branding.school_name"]; v != "" {
		b.SchoolName = v
	}
	if v := values["branding.tagline"]; v != "" {
		b.Tagline = v
	}
	if v := values["branding.logo_url"]; v != "" {
		b.LogoURL = v
	}
	if v := values["branding.color_primary"]; v != "" {
		b.ColorPrimary = v
	}
	if v := values["branding.color_accent"]; v != "" {
		b.ColorAccent = v
	}
	if v := values["branding.website_url"]; v != "" {
		b.WebsiteURL = v
	}
	if v := values["branding.support_email"]; v != "" {
		b.SupportEmail = v
	}

	return b
}
