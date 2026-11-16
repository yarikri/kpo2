package jsonstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"pos/internal/application"
	"pos/internal/domain"
)

type Store struct {
	path             string
	mu               sync.RWMutex
	state            application.Snapshot
	rejectedCriteria []criterionRecord
}

type studentRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type criterionRecord struct {
	Name     string  `json:"name"`
	MaxScore float64 `json:"max_score"`
	Weight   float64 `json:"weight"`
}

type evaluationRecord struct {
	StudentID string   `json:"student_id"`
	Criterion string   `json:"criterion"`
	Score     *float64 `json:"score"`
}

type records struct {
	Students    []studentRecord    `json:"students"`
	Criteria    []criterionRecord  `json:"criteria"`
	Evaluations []evaluationRecord `json:"evaluations"`
}

func Open(path string) (*Store, error) {
	store := &Store{path: path}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var data *records
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("чтение %s: %w", path, err)
	}
	if data == nil {
		return nil, fmt.Errorf("в %s ожидается объект JSON", path)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("после объекта в %s есть лишние данные", path)
	}
	if err := store.load(*data); err != nil {
		return nil, fmt.Errorf("чтение %s: %w", path, err)
	}
	return store, nil
}

func (s *Store) load(data records) error {
	students := make(map[string]domain.Student, len(data.Students))
	for _, row := range data.Students {
		student, err := domain.NewStudent(row.ID, row.Name)
		if err != nil {
			return err
		}
		if _, exists := students[student.ID()]; exists {
			return fmt.Errorf("повторный ID студента %q", student.ID())
		}
		students[student.ID()] = student
		s.state.Students = append(s.state.Students, student)
	}

	criteria := make(map[string]bool, len(data.Criteria))
	for _, row := range data.Criteria {
		name := strings.TrimSpace(row.Name)
		if criteria[name] {
			return fmt.Errorf("повторный критерий %q", name)
		}
		criteria[name] = true
		criterion, err := domain.NewCriterion(row.Name, row.MaxScore, row.Weight)
		if err != nil {
			s.rejectedCriteria = append(s.rejectedCriteria, row)
			s.state.RejectedCriteria = append(s.state.RejectedCriteria, application.RejectedCriterion{Name: name, Reason: err.Error()})
			continue
		}
		s.state.Criteria = append(s.state.Criteria, criterion)
	}
	rubric, err := domain.NewRubric(s.state.Criteria)
	if err != nil {
		return err
	}
	return s.loadGrades(data.Evaluations, students, rubric)
}

func (s *Store) loadGrades(rows []evaluationRecord, students map[string]domain.Student, rubric domain.Rubric) error {
	type key struct{ studentID, criterion string }
	seen := make(map[key]bool, len(rows))
	for _, record := range rows {
		if record.Score == nil {
			return fmt.Errorf("у оценки %q / %q отсутствует балл", record.StudentID, record.Criterion)
		}
		row := application.GradeInput{
			StudentID: strings.TrimSpace(record.StudentID),
			Criterion: strings.TrimSpace(record.Criterion),
			Score:     *record.Score,
		}
		id := key{row.StudentID, row.Criterion}
		if seen[id] {
			return fmt.Errorf("повторная оценка студента %q по критерию %q", row.StudentID, row.Criterion)
		}
		seen[id] = true

		student, exists := students[row.StudentID]
		if !exists {
			s.state.RejectedGrades = append(s.state.RejectedGrades, application.RejectedGrade{GradeInput: row, Reason: "студент не найден"})
			continue
		}
		evaluation, err := domain.NewEvaluation(student, rubric, row.Criterion, row.Score)
		if err != nil {
			s.state.RejectedGrades = append(s.state.RejectedGrades, application.RejectedGrade{GradeInput: row, Reason: err.Error()})
			continue
		}
		s.state.Evaluations = append(s.state.Evaluations, evaluation)
	}
	return nil
}

func (s *Store) Read() application.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Clone()
}

func (s *Store) Update(change func(*application.Snapshot) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.state.Clone()
	if err := change(&next); err != nil {
		return err
	}
	if err := s.write(next); err != nil {
		return fmt.Errorf("сохранение %s: %w", s.path, err)
	}
	s.state = next
	return nil
}

func (s *Store) write(state application.Snapshot) error {
	data := records{
		Students:    make([]studentRecord, 0, len(state.Students)),
		Criteria:    make([]criterionRecord, 0, len(state.Criteria)+len(s.rejectedCriteria)),
		Evaluations: make([]evaluationRecord, 0, len(state.Evaluations)+len(state.RejectedGrades)),
	}
	for _, student := range state.Students {
		data.Students = append(data.Students, studentRecord{ID: student.ID(), Name: student.Name()})
	}
	for _, criterion := range state.Criteria {
		data.Criteria = append(data.Criteria, criterionRecord{Name: criterion.Name(), MaxScore: criterion.MaxScore(), Weight: criterion.Weight()})
	}
	data.Criteria = append(data.Criteria, s.rejectedCriteria...)
	for _, grade := range state.Evaluations {
		score := grade.Score().Points()
		data.Evaluations = append(data.Evaluations, evaluationRecord{StudentID: grade.StudentID(), Criterion: grade.Criterion(), Score: &score})
	}
	for _, grade := range state.RejectedGrades {
		data.Evaluations = append(data.Evaluations, evaluationRecord{StudentID: grade.StudentID, Criterion: grade.Criterion, Score: &grade.Score})
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".pos-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")
	writeErr := encoder.Encode(data)
	if writeErr == nil {
		writeErr = temp.Sync()
	}
	if err := errors.Join(writeErr, temp.Close()); err != nil {
		return err
	}
	return os.Rename(temp.Name(), s.path)
}
