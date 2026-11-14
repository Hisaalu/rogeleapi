package handler

import (
	"context"
	"database/sql"
	"strings"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/dto"
	"github.com/lib/pq"
)

type lessonEnricher struct {
	db *sql.DB
}

func newLessonEnricher(db *sql.DB) *lessonEnricher {
	return &lessonEnricher{db: db}
}

func (e *lessonEnricher) enrich(ctx context.Context, l *domain.Lesson, viewerID int64) dto.LessonResponse {
	out := e.enrichMany(ctx, []*domain.Lesson{l}, viewerID)
	if len(out) == 0 {
		return dto.LessonWithJoins(l, dto.LessonExtras{})
	}
	return out[0]
}

func (e *lessonEnricher) enrichMany(
	ctx context.Context,
	lessons []*domain.Lesson,
	viewerID int64,
) []dto.LessonResponse {
	if len(lessons) == 0 {
		return []dto.LessonResponse{}
	}

	classIDs := make([]int64, 0, len(lessons))
	subjectIDs := make([]int64, 0, len(lessons))
	teacherIDs := make([]int64, 0, len(lessons))
	lessonIDs := make([]int64, 0, len(lessons))
	seenClass := make(map[int64]struct{}, len(lessons))
	seenSubject := make(map[int64]struct{}, len(lessons))
	seenTeacher := make(map[int64]struct{}, len(lessons))
	seenLesson := make(map[int64]struct{}, len(lessons))

	for _, l := range lessons {
		if l == nil {
			continue
		}
		if l.ClassID != nil {
			if _, ok := seenClass[*l.ClassID]; !ok {
				seenClass[*l.ClassID] = struct{}{}
				classIDs = append(classIDs, *l.ClassID)
			}
		}
		if l.SubjectID != nil {
			if _, ok := seenSubject[*l.SubjectID]; !ok {
				seenSubject[*l.SubjectID] = struct{}{}
				subjectIDs = append(subjectIDs, *l.SubjectID)
			}
		}
		if l.TeacherID != nil {
			if _, ok := seenTeacher[*l.TeacherID]; !ok {
				seenTeacher[*l.TeacherID] = struct{}{}
				teacherIDs = append(teacherIDs, *l.TeacherID)
			}
		}
		if _, ok := seenLesson[l.ID]; !ok {
			seenLesson[l.ID] = struct{}{}
			lessonIDs = append(lessonIDs, l.ID)
		}
	}

	classNames := make(map[int64]string, len(classIDs))
	subjectNames := make(map[int64]string, len(subjectIDs))
	teacherNames := make(map[int64]string, len(teacherIDs))
	materialCounts := make(map[int64]int, len(lessonIDs))
	bookmarkedSet := make(map[int64]bool, len(lessonIDs))

	if len(classIDs) > 0 {
		rows, err := e.db.QueryContext(ctx,
			`SELECT id, name FROM classes WHERE id = ANY($1)`,
			pq.Array(classIDs),
		)
		if err == nil {
			for rows.Next() {
				var id int64
				var name string
				if rows.Scan(&id, &name) == nil && name != "" {
					classNames[id] = name
				}
			}
			rows.Close()
		}
	}

	if len(subjectIDs) > 0 {
		rows, err := e.db.QueryContext(ctx,
			`SELECT id, name FROM subjects WHERE id = ANY($1)`,
			pq.Array(subjectIDs),
		)
		if err == nil {
			for rows.Next() {
				var id int64
				var name string
				if rows.Scan(&id, &name) == nil && name != "" {
					subjectNames[id] = name
				}
			}
			rows.Close()
		}
	}

	if len(teacherIDs) > 0 {
		rows, err := e.db.QueryContext(ctx,
			`SELECT id, COALESCE(first_name,''), COALESCE(last_name,'')
			 FROM users WHERE id = ANY($1)`,
			pq.Array(teacherIDs),
		)
		if err == nil {
			for rows.Next() {
				var id int64
				var first, last string
				if rows.Scan(&id, &first, &last) == nil {
					name := strings.TrimSpace(first + " " + last)
					if name != "" {
						teacherNames[id] = name
					}
				}
			}
			rows.Close()
		}
	}

	if len(lessonIDs) > 0 {
		rows, err := e.db.QueryContext(ctx,
			`SELECT lesson_id, COUNT(*) FROM lesson_materials
			 WHERE lesson_id = ANY($1)
			 GROUP BY lesson_id`,
			pq.Array(lessonIDs),
		)
		if err == nil {
			for rows.Next() {
				var id int64
				var cnt int
				if rows.Scan(&id, &cnt) == nil {
					materialCounts[id] = cnt
				}
			}
			rows.Close()
		}
	}

	if viewerID > 0 && len(lessonIDs) > 0 {
		rows, err := e.db.QueryContext(ctx,
			`SELECT lesson_id FROM bookmarks
			 WHERE user_id = $1 AND lesson_id = ANY($2)`,
			viewerID, pq.Array(lessonIDs),
		)
		if err == nil {
			for rows.Next() {
				var id int64
				if rows.Scan(&id) == nil {
					bookmarkedSet[id] = true
				}
			}
			rows.Close()
		}
	}

	out := make([]dto.LessonResponse, 0, len(lessons))
	for _, l := range lessons {
		if l == nil {
			continue
		}
		extras := dto.LessonExtras{
			MaterialsCount: materialCounts[l.ID],
			IsBookmarked:   bookmarkedSet[l.ID],
		}
		if l.ClassID != nil {
			if name, ok := classNames[*l.ClassID]; ok {
				n := name
				extras.ClassName = &n
			}
		}
		if l.SubjectID != nil {
			if name, ok := subjectNames[*l.SubjectID]; ok {
				n := name
				extras.SubjectName = &n
			}
		}
		if l.TeacherID != nil {
			if name, ok := teacherNames[*l.TeacherID]; ok {
				n := name
				extras.TeacherName = &n
			}
		}
		out = append(out, dto.LessonWithJoins(l, extras))
	}
	return out
}