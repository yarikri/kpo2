package console

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"pos/internal/application"
)

type Menu struct {
	service *application.Service
	input   *bufio.Scanner
	output  io.Writer
}

func New(service *application.Service) *Menu {
	return &Menu{service: service, input: bufio.NewScanner(os.Stdin), output: os.Stdout}
}

// Команда связывает пункт меню с действием, которое выполнит консоль.
type command struct {
	key   string
	label string
	run   func() error
}

func (m *Menu) commands() []command {
	return []command{
		{"1", "Добавить студента", m.addStudent},
		{"2", "Добавить критерий", m.addCriterion},
		{"3", "Показать студентов и рубрикатор", m.showCatalog},
		{"4", "Выставить или исправить оценку", m.placeEvaluation},
		{"5", "Показать оценки студента", m.showEvaluations},
		{"6", "Отменить оценку", m.cancelEvaluation},
		{"7", "Рассчитать итог", m.showTotal},
		{"8", "Экспортировать итог в JSON или CSV", m.exportTotal},
	}
}

func (m *Menu) Run() error {
	fmt.Fprintln(m.output, "ПОС — Преподавательская оценка студентов")
	m.printWarnings(m.service.List())
	commands := m.commands()
	for {
		fmt.Fprintln(m.output)
		for _, command := range commands {
			fmt.Fprintf(m.output, "%s. %s\n", command.key, command.label)
		}
		fmt.Fprintln(m.output, "0. Выход")
		choice, err := m.ask("Команда")
		if err != nil || choice == "0" {
			return finish(err)
		}
		index := slices.IndexFunc(commands, func(item command) bool { return item.key == choice })
		if index < 0 {
			fmt.Fprintln(m.output, "Ошибка: выбери команду из меню")
			continue
		}
		if err := commands[index].run(); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			if m.input.Err() != nil {
				return m.input.Err()
			}
			fmt.Fprintf(m.output, "Ошибка: %v\n", err)
		}
	}
}

func finish(err error) error {
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func (m *Menu) addStudent() error {
	values, err := m.readFields("ID студента", "Имя")
	if err != nil {
		return err
	}
	return m.confirm(m.service.AddStudent(values[0], values[1]), "Студент добавлен.")
}

func (m *Menu) cancelEvaluation() error {
	values, err := m.readFields("ID студента", "Критерий")
	if err != nil {
		return err
	}
	return m.confirm(m.service.CancelEvaluation(values[0], values[1]), "Оценка отменена.")
}

func (m *Menu) showCatalog() error {
	m.printCatalog(m.service.List())
	return nil
}

func (m *Menu) addCriterion() error {
	values, err := m.readFields("Название", "Максимальный балл", "Вес")
	if err != nil {
		return err
	}
	maxScore, err := number(values[1], "максимальный балл")
	if err != nil {
		return err
	}
	weight, err := number(values[2], "вес")
	if err != nil {
		return err
	}
	return m.confirm(m.service.AddCriterion(values[0], maxScore, weight), "Критерий добавлен.")
}

func (m *Menu) placeEvaluation() error {
	values, err := m.readFields("ID студента", "Критерий", "Балл")
	if err != nil {
		return err
	}
	score, err := number(values[2], "балл")
	if err != nil {
		return err
	}
	input := application.GradeInput{StudentID: values[0], Criterion: values[1], Score: score}
	return m.confirm(m.service.PlaceEvaluation(input), "Оценка сохранена.")
}

func (m *Menu) showEvaluations() error {
	id, err := m.ask("ID студента")
	if err != nil {
		return err
	}
	state := m.service.List()
	studentName := ""
	for _, student := range state.Students {
		if student.ID() == id {
			studentName = student.Name()
			break
		}
	}
	if studentName == "" {
		return fmt.Errorf("студент %q не найден", id)
	}
	fmt.Fprintf(m.output, "Оценки студента %s [%s]:\n", studentName, id)
	count := 0
	for _, grade := range state.Evaluations {
		if grade.StudentID() == id {
			fmt.Fprintf(m.output, "  %s — %g\n", grade.Criterion(), grade.Score().Points())
			count++
		}
	}
	for _, grade := range state.RejectedGrades {
		if grade.StudentID == id {
			fmt.Fprintf(m.output, "  %s — %g. Предупреждение: %s\n", grade.Criterion, grade.Score, grade.Reason)
			count++
		}
	}
	if count == 0 {
		fmt.Fprintln(m.output, "  Оценок пока нет.")
	}
	return nil
}

func (m *Menu) exportTotal() error {
	values, err := m.readFields("ID студента", "Путь к новому отчёту")
	if err != nil {
		return err
	}
	return m.confirm(m.service.ExportTotal(values[0], values[1]), "Итог экспортирован.")
}

func (m *Menu) showTotal() error {
	id, err := m.ask("ID студента")
	if err != nil {
		return err
	}
	report, err := m.service.Total(id)
	if err != nil {
		return err
	}
	fmt.Fprintf(m.output, "Итог для %s: %.2f%%\n", report.Student, report.Percent)
	return nil
}

func (m *Menu) printCatalog(state application.Snapshot) {
	fmt.Fprintln(m.output, "Студенты:")
	for _, student := range state.Students {
		fmt.Fprintf(m.output, "  %s — %s\n", student.ID(), student.Name())
	}
	if len(state.Students) == 0 {
		fmt.Fprintln(m.output, "  Пока пусто.")
	}
	fmt.Fprintln(m.output, "Рубрикатор:")
	for _, criterion := range state.Criteria {
		fmt.Fprintf(m.output, "  %s — максимум %g, вес %g\n", criterion.Name(), criterion.MaxScore(), criterion.Weight())
	}
	if len(state.Criteria) == 0 {
		fmt.Fprintln(m.output, "  Пока пусто.")
	}
	m.printWarnings(state)
}

func (m *Menu) printWarnings(state application.Snapshot) {
	for _, criterion := range state.RejectedCriteria {
		fmt.Fprintf(m.output, "Предупреждение: критерий %q — %s\n", criterion.Name, criterion.Reason)
	}
	for _, grade := range state.RejectedGrades {
		fmt.Fprintf(m.output, "Предупреждение: оценка %q / %q — %s\n", grade.StudentID, grade.Criterion, grade.Reason)
	}
}

func (m *Menu) ask(label string) (string, error) {
	fmt.Fprintf(m.output, "%s: ", label)
	if m.input.Scan() {
		return strings.TrimSpace(m.input.Text()), nil
	}
	if err := m.input.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

func (m *Menu) readFields(labels ...string) ([]string, error) {
	values := make([]string, 0, len(labels))
	for _, label := range labels {
		value, err := m.ask(label)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (m *Menu) confirm(err error, message string) error {
	if err == nil {
		fmt.Fprintln(m.output, message)
	}
	return err
}

func number(input, label string) (float64, error) {
	value, err := strconv.ParseFloat(strings.ReplaceAll(input, ",", "."), 64)
	if err != nil {
		return 0, fmt.Errorf("%s: ожидается число", label)
	}
	return value, nil
}
