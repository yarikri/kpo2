package application

import (
	"slices"

	"pos/internal/domain"
)

// Snapshot — текущее состояние курса. Не прошедшие проверку записи доступны
// отдельно, чтобы консоль могла показывать предупреждения.
type Snapshot struct {
	Students         []domain.Student
	Criteria         []domain.Criterion
	Evaluations      []domain.Evaluation
	RejectedCriteria []RejectedCriterion
	RejectedGrades   []RejectedGrade
}

func (s Snapshot) Clone() Snapshot {
	return Snapshot{
		Students: slices.Clone(s.Students), Criteria: slices.Clone(s.Criteria),
		Evaluations: slices.Clone(s.Evaluations), RejectedCriteria: slices.Clone(s.RejectedCriteria),
		RejectedGrades: slices.Clone(s.RejectedGrades),
	}
}

type RejectedCriterion struct {
	Name   string
	Reason string
}

type GradeInput struct {
	StudentID string
	Criterion string
	Score     float64
}

type RejectedGrade struct {
	GradeInput
	Reason string
}

type Report struct {
	StudentID string  `json:"student_id"`
	Student   string  `json:"student"`
	Method    string  `json:"method"`
	Percent   float64 `json:"percent"`
}

// Repository применяет изменение целиком. При ошибке прежние данные сохраняются.
type Repository interface {
	Read() Snapshot
	Update(change func(*Snapshot) error) error
}

type ReportExporter interface {
	WriteReport(path string, report Report) error
}
