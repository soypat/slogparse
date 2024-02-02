# slogparse
[![go.dev reference](https://pkg.go.dev/badge/github.com/soypat/slogparse)](https://pkg.go.dev/github.com/soypat/slogparse)
[![Go Report Card](https://goreportcard.com/badge/github.com/soypat/slogparse)](https://goreportcard.com/report/github.com/soypat/slogparse)
[![codecov](https://codecov.io/gh/soypat/slogparse/branch/main/graph/badge.svg)](https://codecov.io/gh/soypat/slogparse)
[![Go](https://github.com/soypat/slogparse/actions/workflows/go.yml/badge.svg)](https://github.com/soypat/slogparse/actions/workflows/go.yml)
[![sourcegraph](https://sourcegraph.com/github.com/soypat/slogparse/-/badge.svg)](https://sourcegraph.com/github.com/soypat/slogparse?badge)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) 


How to install package with newer versions of Go (+1.16):
```sh
go mod download github.com/soypat/slogparse@latest
```


## Example

```go
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/soypat/slogparse"
)

func main() {
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
	fp.Seek(0, 0) // Reset file pointer to start to re-read with parser.
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
```