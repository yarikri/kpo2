package application

import (
	"fmt"
	"slices"
	"strings"

	"pos/internal/domain"
)

// Service — фасад сценариев, с которым работает консоль.
type Service struct {
	repository Repository
	exporter   ReportExporter
}

func NewService(repository Repository, exporter ReportExporter) *Service {
	return &Service{repository: repository, exporter: exporter}
}

func (s *Service) List() Snapshot {
	return s.repository.Read()
}

func (s *Service) AddStudent(id, name string) error {
	student, err := domain.NewStudent(id, name)
	if err != nil {
		return err
	}
	return s.repository.Update(func(state *Snapshot) error {
		if _, exists := findStudent(state.Students, student.ID()); exists {
			return fmt.Errorf("студент с ID %q уже есть", student.ID())
		}
		state.Students = append(state.Students, student)
		return nil
	})
}

func (s *Service) AddCriterion(name string, maxScore, weight float64) error {
	criterion, err := domain.NewCriterion(name, maxScore, weight)
	if err != nil {
		return err
	}
	return s.repository.Update(func(state *Snapshot) error {
		for _, rejected := range state.RejectedCriteria {
			if rejected.Name == criterion.Name() {
				return fmt.Errorf("критерий %q уже есть в файле и требует исправления", criterion.Name())
			}
		}
		criteria := append(state.Criteria, criterion)
		if _, err := domain.NewRubric(criteria); err != nil {
			return err
		}
		state.Criteria = criteria
		return nil
	})
}

func (s *Service) PlaceEvaluation(input GradeInput) error {
	return s.repository.Update(func(state *Snapshot) error {
		student, found := findStudent(state.Students, strings.TrimSpace(input.StudentID))
		if !found {
			return fmt.Errorf("студент %q не найден", input.StudentID)
		}
		rubric, err := domain.NewRubric(state.Criteria)
		if err != nil {
			return err
		}
		grade, err := domain.NewEvaluation(student, rubric, input.Criterion, input.Score)
		if err != nil {
			return err
		}

		index := slices.IndexFunc(state.Evaluations, func(current domain.Evaluation) bool {
			return current.StudentID() == grade.StudentID() && current.Criterion() == grade.Criterion()
		})
		if index >= 0 {
			state.Evaluations[index] = grade
		} else {
			state.Evaluations = append(state.Evaluations, grade)
		}
		state.RejectedGrades = slices.DeleteFunc(state.RejectedGrades, func(rejected RejectedGrade) bool {
			return rejected.StudentID == grade.StudentID() && rejected.Criterion == grade.Criterion()
		})
		return nil
	})
}

func (s *Service) CancelEvaluation(studentID, criterion string) error {
	studentID, criterion = strings.TrimSpace(studentID), strings.TrimSpace(criterion)
	return s.repository.Update(func(state *Snapshot) error {
		count := len(state.Evaluations) + len(state.RejectedGrades)
		state.Evaluations = slices.DeleteFunc(state.Evaluations, func(grade domain.Evaluation) bool {
			return grade.StudentID() == studentID && grade.Criterion() == criterion
		})
		state.RejectedGrades = slices.DeleteFunc(state.RejectedGrades, func(grade RejectedGrade) bool {
			return grade.StudentID == studentID && grade.Criterion == criterion
		})
		if len(state.Evaluations)+len(state.RejectedGrades) == count {
			return fmt.Errorf("оценка студента %q по критерию %q не найдена", studentID, criterion)
		}
		return nil
	})
}

func (s *Service) Total(studentID string) (Report, error) {
	state := s.repository.Read()
	student, found := findStudent(state.Students, strings.TrimSpace(studentID))
	if !found {
		return Report{}, fmt.Errorf("студент %q не найден", studentID)
	}
	if len(state.RejectedCriteria) > 0 {
		return Report{}, fmt.Errorf("исправьте некорректные критерии в файле данных перед расчётом")
	}

	rubric, err := domain.NewRubric(state.Criteria)
	if err != nil {
		return Report{}, err
	}
	percent, err := domain.CalculateTotal(student, rubric, state.Evaluations)
	if err != nil {
		return Report{}, err
	}
	return Report{StudentID: student.ID(), Student: student.Name(), Method: "weighted", Percent: percent}, nil
}

func (s *Service) ExportTotal(studentID, path string) error {
	report, err := s.Total(studentID)
	if err != nil {
		return err
	}
	return s.exporter.WriteReport(path, report)
}

func findStudent(students []domain.Student, id string) (domain.Student, bool) {
	for _, student := range students {
		if student.ID() == id {
			return student, true
		}
	}
	return domain.Student{}, false
}
