package slogparse_test

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/soypat/slogparse"
)

func ExampleTextParser() {
	const filename = "slog.log"
	fp, err := os.Create(filename)
	if err != nil {
		panic(err.Error())
	}
	log := slog.New(slog.NewTextHandler(fp, nil))
	log.Info("Hello", slog.String("name", "World"))
	log.Info("Bye", slog.String("name", "Welt"))
	if err != nil {
		panic(err.Error())
	}
	fp.Seek(0, 0)
	p := slogparse.NewTextParser(fp, slogparse.ParserConfig{})
	for {
		record, err := p.Next()
		if err != nil {
			break
		}
		message := record.Get("msg")
		level := record.Get("level")
		name := record.Get("name")
		fmt.Println(level, message, "name:", name)
	}
	// Output:
	// INFO Hello name: World
	// INFO Bye name: Welt
}
