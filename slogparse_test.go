package slogparse_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

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

var ctx = context.Background()

func TestTextParser(t *testing.T) {
	const max64 = 0xfffffffffffffff
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	var tests = []struct {
		level slog.Level
		msg   string
		attrs []slog.Attr
	}{
		{slog.LevelInfo, "escape quote3\\\\\"", nil},
		{slog.LevelInfo, "escape quote2\\\"", nil},
		{slog.LevelInfo, "escape quote\\", nil},
		{slog.LevelInfo, "escape\n", nil},
		{slog.LevelInfo, "", nil},
		{slog.LevelInfo, "whitespace example", nil},
		{slog.LevelInfo, "simple_string", []slog.Attr{slog.String("name", "World")}},
		{slog.LevelInfo, "int+string", []slog.Attr{slog.Int64("name", 32), slog.String("xx", "yy")}},
	}
	for _, test := range tests {
		log.LogAttrs(ctx, test.level, test.msg, test.attrs...)
	}
	data := buf.String()
	p := slogparse.NewTextParser(strings.NewReader(data), slogparse.ParserConfig{})
	for _, test := range tests {
		record, err := p.Next()
		if err != nil {
			t.Fatalf("parser ended early: %s", err)
		}
		if record.Get("msg") != test.msg {
			t.Errorf("msg mismatch: %s != %s", record.Get("msg"), test.msg)
		}
		if record.Get("level") != test.level.String() {
			t.Errorf("level mismatch: %s != %s", record.Get("level"), test.level.String())
		}
		for _, attr := range test.attrs {
			switch attr.Value.Kind() {
			case slog.KindString:
				if record.Get(attr.Key) != attr.Value.String() {
					t.Errorf("attr mismatch: %s != %s", record.Get(attr.Key), attr.Value.String())
				}
			case slog.KindInt64:
				GOT := record.GetInt(attr.Key, max64)
				if int64(GOT) != attr.Value.Int64() {
					t.Errorf("attr mismatch: %d != %d", GOT, attr.Value.Int64())
				}
			case slog.KindDuration:
				GOT := record.GetDuration(attr.Key, max64)
				if GOT != attr.Value.Duration() {
					t.Errorf("attr mismatch: %s != %s", GOT, attr.Value.Duration())
				}
			}
		}
	}
}
