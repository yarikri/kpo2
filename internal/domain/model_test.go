package domain

import (
	"math"
	"testing"
)

func TestEvaluationFactory(t *testing.T) {
	student, err := NewStudent("anna", "Анна")
	if err != nil {
		t.Fatal(err)
	}
	criterion := testCriterion(t, "Содержание", 10, 2)
	rubric := testRubric(t, criterion)

	for _, test := range []struct {
		name, criterion string
		score           float64
		wantError       bool
	}{
		{"zero", "Содержание", 0, false},
		{"maximum", "Содержание", 10, false},
		{"negative", "Содержание", -1, true},
		{"too high", "Содержание", 12, true},
		{"unknown criterion", "Оформление", 5, true},
		{"NaN", "Содержание", math.NaN(), true},
		{"infinity", "Содержание", math.Inf(1), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			evaluation, err := NewEvaluation(student, rubric, test.criterion, test.score)
			if (err != nil) != test.wantError {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && evaluation.Score().Points() != test.score {
				t.Fatalf("score = %v, want %v", evaluation.Score().Points(), test.score)
			}
		})
	}
}

func TestCalculateTotal(t *testing.T) {
	student, err := NewStudent("anna", "Анна")
	if err != nil {
		t.Fatal(err)
	}
	rubric := testRubric(t, testCriterion(t, "Содержание", 10, 3), testCriterion(t, "Оформление", 5, 1))
	content, err := NewEvaluation(student, rubric, "Содержание", 8)
	if err != nil {
		t.Fatal(err)
	}
	format, err := NewEvaluation(student, rubric, "Оформление", 2)
	if err != nil {
		t.Fatal(err)
	}
	grades := []Evaluation{content, format}
	total, err := CalculateTotal(student, rubric, grades)
	if err != nil || math.Abs(total-70) > 1e-9 {
		t.Fatalf("total = %g, error = %v, want 70", total, err)
	}
	if _, err := CalculateTotal(student, rubric, []Evaluation{content}); err == nil {
		t.Fatal("expected an error for missing grade")
	}
}

func testCriterion(t *testing.T, name string, maxScore, weight float64) Criterion {
	t.Helper()
	criterion, err := NewCriterion(name, maxScore, weight)
	if err != nil {
		t.Fatal(err)
	}
	return criterion
}

func testRubric(t *testing.T, criteria ...Criterion) Rubric {
	t.Helper()
	rubric, err := NewRubric(criteria)
	if err != nil {
		t.Fatal(err)
	}
	return rubric
}
