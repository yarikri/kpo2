package gradefiles

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"pos/internal/application"
)

type Files struct{}

func New() *Files { return &Files{} }

func (*Files) WriteReport(path string, report application.Report) error {
	extension := strings.ToLower(filepath.Ext(path))
	if extension != ".json" && extension != ".csv" {
		return fmt.Errorf("отчёт должен быть файлом .json или .csv")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	if err := errors.Join(writeReport(file, extension, report), file.Close()); err != nil {
		return errors.Join(err, os.Remove(path))
	}
	return nil
}

func writeReport(writer io.Writer, extension string, report application.Report) error {
	if extension == ".json" {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	}
	return csv.NewWriter(writer).WriteAll([][]string{
		{"student_id", "student", "method", "percent"},
		{report.StudentID, report.Student, report.Method, strconv.FormatFloat(report.Percent, 'f', -1, 64)},
	})
}
