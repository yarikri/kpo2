package domain

import (
	"fmt"
	"math"
	"strings"
)

type Student struct {
	id   string
	name string
}

func NewStudent(id, name string) (Student, error) {
	id, name = strings.TrimSpace(id), strings.TrimSpace(name)
	if id == "" || name == "" {
		return Student{}, fmt.Errorf("ID и имя студента обязательны")
	}
	return Student{id: id, name: name}, nil
}

func (s Student) ID() string   { return s.id }
func (s Student) Name() string { return s.name }

type Criterion struct {
	name     string
	maxScore float64
	weight   float64
}

func NewCriterion(name string, maxScore, weight float64) (Criterion, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Criterion{}, fmt.Errorf("название критерия обязательно")
	}
	if !finite(maxScore) || maxScore <= 0 {
		return Criterion{}, fmt.Errorf("максимум критерия должен быть положительным конечным числом")
	}
	if !finite(weight) || weight <= 0 {
		return Criterion{}, fmt.Errorf("вес критерия должен быть положительным конечным числом")
	}
	return Criterion{name: name, maxScore: maxScore, weight: weight}, nil
}

func (c Criterion) Name() string      { return c.name }
func (c Criterion) MaxScore() float64 { return c.maxScore }
func (c Criterion) Weight() float64   { return c.weight }

type Rubric struct {
	criteria map[string]Criterion
}

func NewRubric(criteria []Criterion) (Rubric, error) {
	rubric := Rubric{criteria: make(map[string]Criterion, len(criteria))}
	for _, criterion := range criteria {
		if criterion.name == "" {
			return Rubric{}, fmt.Errorf("пустой критерий")
		}
		if _, exists := rubric.criteria[criterion.name]; exists {
			return Rubric{}, fmt.Errorf("критерий %q уже есть в рубрикаторе", criterion.name)
		}
		rubric.criteria[criterion.name] = criterion
	}
	return rubric, nil
}

func (r Rubric) Criterion(name string) (Criterion, bool) {
	criterion, found := r.criteria[strings.TrimSpace(name)]
	return criterion, found
}

// Score — объект-значение, содержащий проверенный балл.
type Score struct {
	points float64
}

func (s Score) Points() float64 { return s.points }

type Evaluation struct {
	studentID string
	criterion string
	score     Score
}

// NewEvaluation — фабрика оценки. Она проверяет критерий и диапазон балла.
func NewEvaluation(student Student, rubric Rubric, criterionName string, points float64) (Evaluation, error) {
	if student.id == "" {
		return Evaluation{}, fmt.Errorf("студент не задан")
	}
	criterion, found := rubric.Criterion(criterionName)
	if !found {
		return Evaluation{}, fmt.Errorf("критерия %q нет в рубрикаторе", criterionName)
	}
	if !finite(points) || points < 0 || points > criterion.maxScore {
		return Evaluation{}, fmt.Errorf("балл по критерию %q должен быть от 0 до %g", criterion.name, criterion.maxScore)
	}
	return Evaluation{studentID: student.id, criterion: criterion.name, score: Score{points: points}}, nil
}

func (e Evaluation) StudentID() string { return e.studentID }
func (e Evaluation) Criterion() string { return e.criterion }
func (e Evaluation) Score() Score      { return e.score }

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
