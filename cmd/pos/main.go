package main

import (
	"flag"
	"fmt"
	"os"

	"go.uber.org/dig"

	"pos/internal/adapters/console"
	"pos/internal/adapters/gradefiles"
	"pos/internal/adapters/jsonstore"
	"pos/internal/application"
)

func main() {
	data := flag.String("data", "data/pos.json", "путь к файлу данных")
	flag.Parse()
	if err := run(*data); err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка запуска:", err)
		os.Exit(1)
	}
}

func run(path string) error {
	container := dig.New()
	if err := container.Provide(func() (*jsonstore.Store, error) {
		return jsonstore.Open(path)
	}, dig.As(new(application.Repository))); err != nil {
		return err
	}
	if err := container.Provide(gradefiles.New, dig.As(new(application.ReportExporter))); err != nil {
		return err
	}
	if err := container.Provide(application.NewService); err != nil {
		return err
	}
	if err := container.Provide(console.New); err != nil {
		return err
	}
	return container.Invoke(func(menu *console.Menu) error {
		return menu.Run()
	})
}
