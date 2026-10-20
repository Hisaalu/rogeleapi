package handler

import (
	"context"
	"database/sql"
	"strings"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/dto"
)

type lessonEnricher struct {
	db *sql.DB
}

func newLessonEnricher(db *sql.DB) *lessonEnricher {
	return &lessonEnricher{db: db}
}

func (e *lessonEnricher) enrich(ctx context.Context, l *domain.Lesson, viewerID int64) dto.LessonResponse {
	extras := dto.LessonExtras{}

	if l.ClassID != nil {
		var name string
		err := e.db.QueryRowContext(ctx,
			`SELECT name FROM classes WHERE id = $1`, *l.ClassID,
		).Scan(&name)
		if err == nil && name != "" {
			extras.ClassName = &name
		}
	}

	if l.SubjectID != nil {
		var name string
		err := e.db.QueryRowContext(ctx,
			`SELECT name FROM subjects WHERE id = $1`, *l.SubjectID,
		).Scan(&name)
		if err == nil && name != "" {
			extras.SubjectName = &name
		}
	}

	if l.TeacherID != nil {
		var first, last sql.NullString
		err := e.db.QueryRowContext(ctx,
			`SELECT first_name, last_name FROM users WHERE id = $1`, *l.TeacherID,
		).Scan(&first, &last)
		if err == nil {
			name := strings.TrimSpace(first.String + " " + last.String)
			if name != "" {
				extras.TeacherName = &name
			}
		}
	}

	var count int
	if err := e.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM lesson_materials WHERE lesson_id = $1`, l.ID,
	).Scan(&count); err == nil {
		extras.MaterialsCount = count
	}

	if viewerID > 0 {
		var exists bool
		if err := e.db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM bookmarks WHERE user_id = $1 AND lesson_id = $2)`,
			viewerID, l.ID,
		).Scan(&exists); err == nil {
			extras.IsBookmarked = exists
		}
	}

	return dto.LessonWithJoins(l, extras)
}
