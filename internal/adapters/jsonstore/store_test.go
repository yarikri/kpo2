package jsonstore_test

import (
	"os"
	"path/filepath"
	"testing"

	"pos/internal/adapters/gradefiles"
	"pos/internal/adapters/jsonstore"
	"pos/internal/application"
)

func TestEvaluationUpdatesAndPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "course.json")
	store, err := jsonstore.Open(path)
	must(t, err)
	files := gradefiles.New()
	service := application.NewService(store, files)
	must(t, service.AddStudent("anna", "Анна"))
	must(t, service.AddCriterion("Содержание", 10, 1))
	must(t, service.AddCriterion("Оформление", 5, 1))
	must(t, service.PlaceEvaluation(application.GradeInput{StudentID: "anna", Criterion: "Содержание", Score: 8}))

	if err := service.PlaceEvaluation(application.GradeInput{StudentID: "anna", Criterion: "Содержание", Score: 12}); err == nil {
		t.Fatal("expected invalid score to fail")
	}
	assertGrades(t, store, 8)

	must(t, service.PlaceEvaluation(application.GradeInput{StudentID: "anna", Criterion: "Содержание", Score: 9}))
	reopened, err := jsonstore.Open(path)
	must(t, err)
	assertGrades(t, reopened, 9)
}

func TestInvalidCriterionRemainsVisibleAfterSaving(t *testing.T) {
	path := filepath.Join(t.TempDir(), "course.json")
	must(t, os.WriteFile(path, []byte(`{
		"students":[{"id":"anna","name":"Анна"}],
		"criteria":[{"name":"Оформление","max_score":-5,"weight":1}],
		"evaluations":[{"student_id":"anna","criterion":"Оформление","score":4}]
	}`), 0600))
	store, err := jsonstore.Open(path)
	must(t, err)
	files := gradefiles.New()
	service := application.NewService(store, files)
	must(t, service.AddCriterion("Содержание", 10, 1))
	reopened, err := jsonstore.Open(path)
	must(t, err)
	state := reopened.Read()
	if len(state.Criteria) != 1 || len(state.RejectedCriteria) != 1 || len(state.RejectedGrades) != 1 {
		t.Fatalf("unexpected state: %+v", state)
	}
	if state.RejectedGrades[0].Score != 4 {
		t.Fatalf("unexpected rejected grade: %+v", state.RejectedGrades[0])
	}
}

func assertGrades(t *testing.T, store *jsonstore.Store, want float64) {
	t.Helper()
	grades := store.Read().Evaluations
	if len(grades) != 1 || grades[0].Score().Points() != want {
		t.Fatalf("grades = %+v, want one grade with score %g", grades, want)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
