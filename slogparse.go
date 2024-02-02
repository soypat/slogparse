package slogparse

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// TextParser implements parsing of a [slog.TextHandler] generated structured log file.
type TextParser struct {
	scanner     *bufio.Scanner
	lineNumber  int
	reuseRecord []kv
	lastRecord  Record
}

type ParserConfig struct {
	// ReuseRecord controls whether calls to Read may return a slice sharing
	// the backing array of the previous call's returned slice for performance.
	// By default, each call to Read returns newly allocated memory owned by the caller.
	ReuseRecord bool
}

// Record is a single parsed log line, equivalent to [slog.Record].
type Record struct {
	items []kv
}

type kv struct {
	key   string
	value string
}

// NewTextParser returns a new parser ready to parse TextHandler slog-formatted logs.
func NewTextParser(r io.Reader, cfg ParserConfig) *TextParser {

	p := &TextParser{
		scanner: bufio.NewScanner(r),
	}
	if cfg.ReuseRecord {
		p.reuseRecord = make([]kv, 0, 8)
	}
	return p
}

// Next reads the next log line from the input and returns it as a Record.
// Next returns [io.EOF] when the input ends.
func (p *TextParser) Next() (Record, error) {
	if err := p.scan(); err != nil {
		return Record{}, err
	}
	return p.lastRecord, nil
}

// Reset discards any buffered data and resets the parser to read from r.
func (p *TextParser) Reset(r io.Reader) {
	p.scanner = bufio.NewScanner(r)
	p.lineNumber = 0
	p.lastRecord = Record{}
}

func (p *TextParser) scan() error {
	scanner := p.scanner
	if !scanner.Scan() {
		err := scanner.Err()
		if err == nil {
			err = io.EOF // Ensure error returned to end the loop.
		}
		return err
	}
	p.lineNumber++
	line := scanner.Text()
	text := line
	var items []kv
	if p.reuseRecord != nil {
		items = p.reuseRecord[:0]
	}
	keyNumber := 0
	for len(text) > 0 {
		var err error
		var key, value string
		key, text, err = cutString(text, true)
		if err != nil {
			return p.abortf("%s at key %d: %v", err, keyNumber, line)
		}
		if len(text) <= 1 {
			return p.abortf("unterminated string key %d: %v", keyNumber, line)
		} else if len(key) == 0 {
			return p.abortf("malformed key %d: %v", keyNumber, line)
		}
		value, text, err = cutString(text, false)
		if err != nil {
			return p.abortf("%s at value %d: %v", err, keyNumber, line)
		}
		if len(value) > 0 && value[0] == ' ' {
			return p.abortf("value %d starts with forbidden char: %v", keyNumber, line)
		}
		items = append(items, kv{key: key, value: value})
		keyNumber++
	}
	p.lastRecord.items = items
	return nil
}

func (p *TextParser) abortf(msg string, a ...any) error {
	a = append([]any{p.lineNumber}, a...)
	return fmt.Errorf("line %d: "+msg, a...)
}

func cutString(s string, key bool) (result, rest string, err error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return "", "", nil
	}
	lookFor := byte(' ')
	if key {
		lookFor = '='
	}

	if s[0] != '"' {
		// Simplest case, is not quoted string.
		spaceIdx := strings.IndexByte(s, lookFor)
		if spaceIdx < 0 {
			// No space found, return entire string.
			return s, "", nil
		}
		return s[:spaceIdx], s[spaceIdx+1:], nil
	} else if len(s) > 1 && s[1] == '"' {
		// Empty string case.
		return "", s[2:], nil
	}

	// Parse quoted string case.
	maybeQuoteIdx := 1
	for {
		off := strings.IndexByte(s[maybeQuoteIdx:], '"')
		if off < 0 {
			return "", "", fmt.Errorf("unterminated quoted string: %v", s)
		}
		maybeQuoteIdx += off
		// We now count the number of backslashes before the quote.
		bsCount := 0
		for ; bsCount < maybeQuoteIdx && s[maybeQuoteIdx-1-bsCount] == '\\'; bsCount++ {
		}

		if bsCount%2 == 0 {
			// If the number of backslashes is even,
			// the quote is not escaped, we may terminate the string here.
			break
		}
		// This quote is escaped, continue searching for the next one.
		maybeQuoteIdx++
	}
	result, err = strconv.Unquote(s[:maybeQuoteIdx+1])
	rest = s[maybeQuoteIdx+1:]
	if len(rest) > 0 {
		rest = s[maybeQuoteIdx+2:]
	}
	return result, rest, err
}

// ForEach calls the given function for each key-value pair in the Record.
func (d Record) ForEach(fn func(key, value string)) {
	for i := range d.items {
		fn(d.items[i].key, d.items[i].value)
	}
}

// Contains returns true if the Record contains the given key.
func (d Record) ContainsK(key string) bool {
	for i := range d.items {
		if d.items[i].key == key {
			return true
		}
	}
	return false
}

// ContainsKV returns true if the Record contains the given key and value.
func (d Record) ContainsKV(key, value string) bool {
	for i := range d.items {
		if d.items[i].key == key && d.items[i].value == value {
			return true
		}
	}
	return false
}

// Get returns the value for the given key. If the key is not found, returns an empty string.
func (d Record) Get(key string) string {
	for i := range d.items {
		if d.items[i].key == key {
			return d.items[i].value
		}
	}
	return ""
}

// GetInt returns the value for the given key as an integer.
// If the key is not found or the value is not an integer, returns the argument default value.
func (d Record) GetInt(key string, defaultVal int) int {
	for i := range d.items {
		if d.items[i].key == key {
			v, err := strconv.Atoi(d.items[i].value)
			if err == nil {
				return v
			}
		}
	}
	return defaultVal
}

// LogTime returns time.Time from "time" key in the dictionary.
// If not found, returns time.Time{}.
func (d Record) LogTime() time.Time {
	const logLayout = "2006-01-02T15:04:05.999-07:00"
	return d.GetTime("time", logLayout)
}

// GetTime returns the value for the given key as a time.Time.
// If the key is not found or the value is not a valid time, returns time.Time{}.
func (d Record) GetTime(key string, layout string) time.Time {
	for i := range d.items {
		if d.items[i].key == key {
			t, err := time.Parse(layout, d.items[i].value)
			if err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

// GetDuration returns the value for the given key as a time.Duration.
// If the key is not found or the value is not a valid duration, returns the argument default value.
func (d Record) GetDuration(key string, defaultVal time.Duration) time.Duration {
	for i := range d.items {
		if d.items[i].key == key {
			v, err := time.ParseDuration(d.items[i].value)
			if err == nil {
				return v
			}
		}
	}
	return defaultVal
}

// getIfValueContains returns the key and value if the value contains the given
// substring. If not found, returns ("", "").
func (d Record) getIfValueContains(substr string) (key, value string) {
	for i := range d.items {
		value := d.items[i].value
		if strings.Contains(value, substr) {
			return d.items[i].key, value
		}
	}
	return "", ""
}
