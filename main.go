// jf — JSON formatter and filter for daily use.
//
// Install:
//
//	go install github.com/roy0x01/jf@latest
//
// Usage:
//
//	jf <input> [-o output] [--filter path[=value]]
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ── ANSI colors ───────────────────────────────────────────────────────────────

const (
	reset      = "\033[0m"
	colorBrace = "\033[38;5;220m" // gold       – { } [ ]
	colorKey   = "\033[38;5;117m" // sky blue   – "key"
	colorStr   = "\033[38;5;114m" // sage green – string values
	colorNum   = "\033[38;5;209m" // coral      – numbers
	colorBool  = "\033[38;5;213m" // violet     – true/false
	colorNull  = "\033[38;5;240m" // grey       – null
	colorPunct = "\033[38;5;247m" // light grey – : ,
)

func colorize(src []byte) []byte {
	var b bytes.Buffer
	b.Grow(len(src) * 2)
	i, n := 0, len(src)
	afterColon := false
	for i < n {
		c := src[i]
		switch {
		// string
		case c == '"':
			j := i + 1
			for j < n {
				if src[j] == '\\' {
					j += 2
					continue
				}
				if src[j] == '"' {
					j++
					break
				}
				j++
			}
			// key if next non-space is ':'
			isKey := false
			if !afterColon {
				k := j
				for k < n && (src[k] == ' ' || src[k] == '\t') {
					k++
				}
				isKey = k < n && src[k] == ':'
			}
			if isKey {
				b.WriteString(colorKey)
			} else {
				b.WriteString(colorStr)
			}
			b.Write(src[i:j])
			b.WriteString(reset)
			afterColon = false
			i = j
		// punctuation
		case c == ':':
			b.WriteString(colorPunct)
			b.WriteByte(c)
			b.WriteString(reset)
			afterColon = true
			i++
		case c == ',':
			b.WriteString(colorPunct)
			b.WriteByte(c)
			b.WriteString(reset)
			afterColon = false
			i++
		// braces / brackets
		case c == '{' || c == '}' || c == '[' || c == ']':
			b.WriteString(colorBrace)
			b.WriteByte(c)
			b.WriteString(reset)
			afterColon = false
			i++
		// keywords
		case i+4 <= n && string(src[i:i+4]) == "null":
			b.WriteString(colorNull)
			b.WriteString("null")
			b.WriteString(reset)
			afterColon = false
			i += 4
		case i+4 <= n && string(src[i:i+4]) == "true":
			b.WriteString(colorBool)
			b.WriteString("true")
			b.WriteString(reset)
			afterColon = false
			i += 4
		case i+5 <= n && string(src[i:i+5]) == "false":
			b.WriteString(colorBool)
			b.WriteString("false")
			b.WriteString(reset)
			afterColon = false
			i += 5
		// number
		case c == '-' || (c >= '0' && c <= '9'):
			j := i
			for j < n && src[j] != ',' && src[j] != '\n' &&
				src[j] != ' ' && src[j] != ']' && src[j] != '}' {
				j++
			}
			b.WriteString(colorNum)
			b.Write(src[i:j])
			b.WriteString(reset)
			afterColon = false
			i = j
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.Bytes()
}

// ── Path tokenizer ────────────────────────────────────────────────────────────
//
// Syntax:
//   key            object key
//   key1.key2      nested key
//   key1.0.key2    array index then key
//   key1[*].key2   wildcard — every array element

type seg struct {
	key     string
	idx     int
	isIdx   bool
	isWild  bool
}

func parsePath(raw string) []seg {
	// Replace [*] with a sentinel so we can split purely on '.'
	const wild = "\x00W\x00"
	s := strings.ReplaceAll(raw, "[*]", "."+wild+".")
	var out []seg
	for _, tok := range strings.Split(s, ".") {
		tok = strings.TrimSpace(tok)
		switch {
		case tok == "":
		case tok == wild:
			out = append(out, seg{isWild: true})
		default:
			if n, err := strconv.Atoi(tok); err == nil {
				out = append(out, seg{idx: n, isIdx: true})
			} else {
				out = append(out, seg{key: tok})
			}
		}
	}
	return out
}

// ── Deep resolve ──────────────────────────────────────────────────────────────
//
// Returns all values reachable from node by following segs.
// [*] fans out across every array element.

func resolve(node interface{}, segs []seg) []interface{} {
	if len(segs) == 0 {
		return []interface{}{node}
	}
	s, rest := segs[0], segs[1:]
	switch {
	case s.isWild:
		arr, ok := node.([]interface{})
		if !ok {
			return nil
		}
		var out []interface{}
		for _, el := range arr {
			out = append(out, resolve(el, rest)...)
		}
		return out
	case s.isIdx:
		arr, ok := node.([]interface{})
		if !ok || s.idx < 0 || s.idx >= len(arr) {
			return nil
		}
		return resolve(arr[s.idx], rest)
	default:
		obj, ok := node.(map[string]interface{})
		if !ok {
			return nil
		}
		v, exists := obj[s.key]
		if !exists {
			return nil
		}
		return resolve(v, rest)
	}
}

// ── Filter ────────────────────────────────────────────────────────────────────

type filter struct {
	segs     []seg
	val      string
	hasVal   bool
}

// parseFilter splits "path=value" at first '=' that is outside brackets.
func parseFilter(s string) (filter, error) {
	depth, eqAt := 0, -1
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
		case '=':
			if depth == 0 {
				eqAt = i
			}
		}
		if eqAt >= 0 {
			break
		}
	}
	path, val, hasVal := s, "", false
	if eqAt >= 0 {
		path, val, hasVal = s[:eqAt], s[eqAt+1:], true
	}
	if path == "" {
		return filter{}, fmt.Errorf("empty filter path")
	}
	segs := parsePath(path)
	if len(segs) == 0 {
		return filter{}, fmt.Errorf("invalid filter path: %q", path)
	}
	return filter{segs: segs, val: val, hasVal: hasVal}, nil
}

func matchVal(got interface{}, want string) bool {
	switch v := got.(type) {
	case string:
		return v == want
	case bool:
		return strconv.FormatBool(v) == want
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10) == want
		}
		return strconv.FormatFloat(v, 'g', -1, 64) == want
	case nil:
		return want == "null"
	default:
		b, _ := json.Marshal(v)
		return string(b) == want
	}
}

// match returns true when node satisfies f.
// For JSONL: node is one top-level record — filter matches the whole record.
// For JSON array roots: each element is tested individually (see filterArray).
func (f filter) match(node interface{}) bool {
	leaves := resolve(node, f.segs)
	if len(leaves) == 0 {
		return false
	}
	if !f.hasVal {
		return true // existence
	}
	for _, l := range leaves {
		if matchVal(l, f.val) {
			return true
		}
	}
	return false
}

// filterArray returns elements of arr that satisfy f when f's path is
// relative to each element. Used when the root JSON document is an array.
func (f filter) filterArray(arr []interface{}) []interface{} {
	var out []interface{}
	for _, el := range arr {
		if f.match(el) {
			out = append(out, el)
		}
	}
	return out
}

// ── Format detection ──────────────────────────────────────────────────────────

func detectFormat(data []byte) (string, error) {
	t := bytes.TrimSpace(data)
	if len(t) == 0 {
		return "", fmt.Errorf("file is empty")
	}
	// Try JSON first.
	var v interface{}
	if err := json.Unmarshal(t, &v); err == nil {
		return "json", nil
	}
	// Try JSONL: every non-empty line must parse.
	sc := bufio.NewScanner(bytes.NewReader(t))
	sc.Buffer(make([]byte, 1<<20), 1<<27)
	count := 0
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var lv interface{}
		if err := json.Unmarshal(line, &lv); err != nil {
			return "", fmt.Errorf("line %d is not valid JSON: %v", count+1, err)
		}
		count++
	}
	if count == 0 {
		return "", fmt.Errorf("no parseable content found")
	}
	return "jsonl", nil
}

// ── Pretty printer ────────────────────────────────────────────────────────────

func pretty(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// ── Process ───────────────────────────────────────────────────────────────────

func process(input, output string, f *filter) error {
	raw, err := os.ReadFile(input)
	if err != nil {
		return err
	}

	ext := strings.ToLower(filepath.Ext(input))
	format := ""
	if ext == ".jsonl" {
		format = "jsonl"
	} else {
		format, err = detectFormat(raw)
		if err != nil {
			return err
		}
	}

	header(input, format)

	// Collect output records.
	var records [][]byte

	switch format {

	case "json":
		var root interface{}
		if err := json.Unmarshal(bytes.TrimSpace(raw), &root); err != nil {
			return fmt.Errorf("parse error: %v", err)
		}
		if f == nil {
			// No filter — pretty-print the whole document.
			b, err := pretty(root)
			if err != nil {
				return err
			}
			records = [][]byte{b}
		} else {
			// Filter: if root is an array, test each element.
			// If root is an object, test the whole document.
			switch rv := root.(type) {
			case []interface{}:
				matched := f.filterArray(rv)
				if len(matched) == 0 {
					return nil
				}
				for _, el := range matched {
					b, err := pretty(el)
					if err != nil {
						return err
					}
					records = append(records, b)
				}
			default:
				if !f.match(root) {
					return nil
				}
				b, err := pretty(root)
				if err != nil {
					return err
				}
				records = [][]byte{b}
			}
		}

	case "jsonl":
		sc := bufio.NewScanner(bytes.NewReader(raw))
		sc.Buffer(make([]byte, 1<<20), 1<<27)
		lineNum := 0
		for sc.Scan() {
			line := bytes.TrimSpace(sc.Bytes())
			if len(line) == 0 {
				continue
			}
			lineNum++
			var v interface{}
			if err := json.Unmarshal(line, &v); err != nil {
				return fmt.Errorf("line %d: %v", lineNum, err)
			}
			if f != nil && !f.match(v) {
				continue
			}
			b, err := pretty(v)
			if err != nil {
				return fmt.Errorf("line %d: %v", lineNum, err)
			}
			records = append(records, b)
		}
		if err := sc.Err(); err != nil {
			return err
		}
	}

	if len(records) == 0 {
		return nil
	}

	// Build plain output (no color) — used for -o file.
	plain := bytes.Join(records, []byte("\n"))
	plain = append(plain, '\n')

	// Always print colored to stdout.
	os.Stdout.Write(colorize(plain))

	// Write plain to file when -o given.
	if output != "" {
		if err := os.WriteFile(output, plain, 0644); err != nil {
			return fmt.Errorf("cannot write %s: %v", output, err)
		}
		ok(fmt.Sprintf("saved → %s", output))
	}

	return nil
}

// ── UI helpers ────────────────────────────────────────────────────────────────

func header(path, format string) {
	fmt.Fprintf(os.Stderr, "\033[1;38;5;220m▶ %s\033[0m  \033[38;5;240m[%s]\033[0m\n",
		filepath.Base(path), strings.ToUpper(format))
}
func ok(msg string)   { fmt.Fprintf(os.Stderr, "  \033[38;5;114m✔\033[0m %s\n", msg) }
func fail(msg string) { fmt.Fprintf(os.Stderr, "  \033[38;5;196m✘\033[0m %s\n", msg) }

// ── Usage ─────────────────────────────────────────────────────────────────────

func usage() {
	fmt.Fprint(os.Stderr, `
  jf — JSON formatter and filter

  Install:
    go install github.com/roy0x01/jf@latest

  Usage:
    jf <input> [-o output] [--filter path[=value]]

  Default:
    Pretty-prints with color to terminal. Input file is never modified.

  Flags:
    -o <file>              write plain output to file
    --filter <path>        show records where path exists
    --filter <path=value>  show records where path equals value

  Path syntax:
    key                top-level key
    a.b                nested key
    a.0.b              array index then key
    a[*].b             wildcard — all elements of array a, then key b
    a[*].b[*].c        nested wildcards

  Behaviour:
    JSONL  each line is a record; filter keeps matching lines
    JSON   if root is array, filter tests each element
           if root is object, filter tests the whole document

  Examples:
    jf data.json
    jf data.json -o pretty.json
    jf logs.jsonl --filter event=login
    jf logs.jsonl --filter user.role=admin
    jf scan.json --filter hosts[*].ip=10.0.0.1
    jf scan.json --filter hosts[*].ports[*].state=open
    jf dump.jsonl --filter results[*].severity=critical -o hits.jsonl

`)
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	var input, output string
	var f *filter

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-o":
			if i+1 >= len(args) {
				fail("-o requires a filename")
				os.Exit(1)
			}
			i++
			output = args[i]
		case strings.HasPrefix(a, "-o="):
			output = a[3:]
		case a == "--filter" || a == "-f":
			if i+1 >= len(args) {
				fail("--filter requires an argument")
				os.Exit(1)
			}
			i++
			pf, err := parseFilter(args[i])
			if err != nil {
				fail(err.Error())
				os.Exit(1)
			}
			f = &pf
		case strings.HasPrefix(a, "--filter="):
			pf, err := parseFilter(a[9:])
			if err != nil {
				fail(err.Error())
				os.Exit(1)
			}
			f = &pf
		case a == "--help" || a == "-h":
			usage()
			os.Exit(0)
		case strings.HasPrefix(a, "-"):
			fail("unknown flag: " + a)
			usage()
			os.Exit(1)
		default:
			if input != "" {
				fail("only one input file at a time")
				os.Exit(1)
			}
			input = a
		}
	}

	if input == "" {
		fail("no input file specified")
		usage()
		os.Exit(1)
	}

	if err := process(input, output, f); err != nil {
		fail(err.Error())
		os.Exit(1)
	}
}
