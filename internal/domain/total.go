package domain

import (
	"fmt"
	"slices"
	"strings"
)

// CalculateTotal — доменный сервис. Итог считается по полному набору оценок.
func CalculateTotal(student Student, rubric Rubric, evaluations []Evaluation) (float64, error) {
	if len(rubric.criteria) == 0 {
		return 0, fmt.Errorf("в рубрикаторе пока нет критериев")
	}
	scores := make(map[string]Score)
	for _, evaluation := range evaluations {
		if evaluation.studentID != student.id {
			continue
		}
		checked, err := NewEvaluation(student, rubric, evaluation.criterion, evaluation.score.points)
		if err != nil {
			return 0, err
		}
		if _, exists := scores[checked.criterion]; exists {
			return 0, fmt.Errorf("повторная оценка по критерию %q", checked.criterion)
		}
		scores[checked.criterion] = checked.score
	}

	names := make([]string, 0, len(rubric.criteria))
	var maxWeight float64
	for name, criterion := range rubric.criteria {
		names = append(names, name)
		maxWeight = max(maxWeight, criterion.weight)
	}
	slices.Sort(names)

	var missing []string
	var earned, totalWeight float64
	for _, name := range names {
		score, exists := scores[name]
		if !exists {
			missing = append(missing, name)
			continue
		}
		criterion := rubric.criteria[name]
		weight := criterion.weight / maxWeight
		earned += (score.points / criterion.maxScore) * weight
		totalWeight += weight
	}
	if len(missing) > 0 {
		return 0, fmt.Errorf("нет оценок по критериям: %s", strings.Join(missing, ", "))
	}
	return 100 * (earned / totalWeight), nil
}
