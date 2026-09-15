package ahdruntime

import (
	"os"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Terminal standard module (v1.5.0)
// ---------------------------------------------------------------------------
//
// Terminal adds the terminal-specific behaviour the Fundamentals write, take,
// and str deliberately do not expose: joining Strings onto standard output,
// writing to standard error, flushing, and asking whether standard output is
// an interactive terminal, how large it is, and whether styled text should be
// used. Nothing here keeps terminal state. Every styled String carries its own
// reset, and the only process state involved is the operating system's
// standard streams.
//
// The capability queries are shared with the evaluator, which applies them to
// its own output file. AhdTerminalIsTerminal, ahdTerminalSize, and
// AhdTerminalSequences are the only platform-specific pieces; they live in the
// terminal_<os>.go files, use the standard library's syscall package, and never
// start a process or read a file.

// AhdTerminalJoin is the exact text Terminal.emit writes: the parts in order
// with separator only between them, then ending once.
func AhdTerminalJoin(parts []string, separator, ending string) string {
	return strings.Join(parts, separator) + ending
}

// AhdTerminalEmit implements Terminal.emit for a generated program. It writes
// through the buffered standard output write uses, so the two keep their
// relative order.
func AhdTerminalEmit(parts *AhdList[string], separator, ending string) {
	var items []string
	if parts != nil {
		items = parts.items
	}
	_, _ = ahdOut.WriteString(AhdTerminalJoin(items, separator, ending))
}

// AhdTerminalError implements Terminal.error. Pending standard output is
// flushed first so a terminal shows both streams in program order; the text
// itself goes straight to the unbuffered standard error.
func AhdTerminalError(text, ending string) {
	AhdFlush()
	_, _ = os.Stderr.WriteString(text + ending)
}

// AhdTerminalFlush implements Terminal.flush. Standard error is unbuffered, so
// the buffered standard output is the only thing with pending text.
func AhdTerminalFlush(class *AhdClass) {
	if err := ahdOut.Flush(); err != nil {
		AhdRaiseClass(class, "standard output could not be flushed")
	}
}

// AhdTerminalColorPolicy is the complete v1.5 color policy. Styled text is
// used only when standard output is an interactive terminal that interprets
// escape sequences, NO_COLOR is unset or empty, and TERM is not "dumb". No
// other environment variable is consulted.
func AhdTerminalColorPolicy(interactive, sequences bool, lookup func(string) (string, bool)) bool {
	if !interactive || !sequences {
		return false
	}
	if value, present := lookup("NO_COLOR"); present && value != "" {
		return false
	}
	if value, _ := lookup("TERM"); value == "dumb" {
		return false
	}
	return true
}

// AhdTerminalDimensions reports the width and height of the terminal on one
// file as AhdCode Int? values: nil unless the file is a terminal and reports a
// positive size. There is no fallback size.
func AhdTerminalDimensions(fd uintptr) (*int64, *int64) {
	if !AhdTerminalIsTerminal(fd) {
		return nil, nil
	}
	columns, rows, ok := ahdTerminalSize(fd)
	if !ok {
		return nil, nil
	}
	return ahdTerminalPositive(columns), ahdTerminalPositive(rows)
}

func ahdTerminalPositive(value int) *int64 {
	if value <= 0 {
		return nil
	}
	result := int64(value)
	return &result
}

// AhdTerminalColorEnabled applies AhdTerminalColorPolicy to one output file
// and the process environment.
func AhdTerminalColorEnabled(fd uintptr) bool {
	interactive := AhdTerminalIsTerminal(fd)
	return AhdTerminalColorPolicy(interactive, interactive && AhdTerminalSequences(fd), os.LookupEnv)
}

// AhdTerminalIsInteractive implements Terminal.isInteractive: whether standard
// output is attached to an interactive terminal.
func AhdTerminalIsInteractive() bool { return AhdTerminalIsTerminal(os.Stdout.Fd()) }

// AhdTerminalWidth implements Terminal.width for standard output.
func AhdTerminalWidth() *int64 {
	width, _ := AhdTerminalDimensions(os.Stdout.Fd())
	return width
}

// AhdTerminalHeight implements Terminal.height for standard output.
func AhdTerminalHeight() *int64 {
	_, height := AhdTerminalDimensions(os.Stdout.Fd())
	return height
}

// AhdTerminalSupportsColor implements Terminal.supportsColor for standard
// output.
func AhdTerminalSupportsColor() bool { return AhdTerminalColorEnabled(os.Stdout.Fd()) }

const ahdTerminalReset = "\x1b[0m"

var ahdTerminalColorOffsets = map[string]int{
	"black": 0, "red": 1, "green": 2, "yellow": 3, "blue": 4, "magenta": 5, "cyan": 6, "white": 7,
}

// ahdTerminalColorCode maps one color name to its SGR parameter. "default"
// adds no parameter; an unknown name is a problem, never a silent default.
func ahdTerminalColorCode(name string, base int, role string) (string, string) {
	if name == "default" {
		return "", ""
	}
	offset, known := ahdTerminalColorOffsets[name]
	if !known {
		return "", "unsupported " + role + " color " + strconv.Quote(name) +
			"; use default, black, red, green, yellow, blue, magenta, cyan, or white"
	}
	return strconv.Itoa(base + offset), ""
}

// AhdTerminalStyle builds Terminal.style's result, or reports why the color
// names are invalid. Names are validated whether or not styling is enabled, so
// a misspelled name fails the same way on every machine. With styling enabled
// and at least one attribute requested, the text is wrapped in one SGR prefix
// and one reset. A reset already inside the text -- the end of a nested
// Terminal.style result -- is followed by this prefix again, so the outer style
// resumes after the inner one instead of silently ending there.
func AhdTerminalStyle(text, foreground, background string, bold, underline, enabled bool) (string, string) {
	foregroundCode, problem := ahdTerminalColorCode(foreground, 30, "foreground")
	if problem != "" {
		return "", problem
	}
	backgroundCode, problem := ahdTerminalColorCode(background, 40, "background")
	if problem != "" {
		return "", problem
	}
	var codes []string
	if bold {
		codes = append(codes, "1")
	}
	if underline {
		codes = append(codes, "4")
	}
	if foregroundCode != "" {
		codes = append(codes, foregroundCode)
	}
	if backgroundCode != "" {
		codes = append(codes, backgroundCode)
	}
	if !enabled || len(codes) == 0 {
		return text, ""
	}
	prefix := "\x1b[" + strings.Join(codes, ";") + "m"
	body := strings.ReplaceAll(text, ahdTerminalReset, ahdTerminalReset+prefix)
	if strings.HasSuffix(text, ahdTerminalReset) {
		body = strings.TrimSuffix(body, prefix)
	}
	return prefix + body + ahdTerminalReset, ""
}

// AhdTerminalStyleText implements Terminal.style against standard output's
// color capability.
func AhdTerminalStyleText(class *AhdClass, text, foreground, background string, bold, underline bool) string {
	styled, problem := AhdTerminalStyle(text, foreground, background, bold, underline, AhdTerminalSupportsColor())
	if problem != "" {
		AhdRaiseClass(class, problem)
	}
	return styled
}

// ahdTerminalIndent is one level of Terminal.pretty indentation.
const ahdTerminalIndent = "    "

// AhdTerminalPrettyLeaf lays out a value with no inner structure exactly as str
// renders it inside a collection.
func AhdTerminalPrettyLeaf[T any](render func(T) string) func(T, string) string {
	return func(value T, _ string) string { return render(value) }
}

// AhdTerminalPrettyList lays out a List one element per line. An empty List
// stays "[]"; indent is the indentation of the line the List starts on.
func AhdTerminalPrettyList[T any](render func(T, string) string) func(*AhdList[T], string) string {
	return func(list *AhdList[T], indent string) string {
		if list == nil {
			return "null"
		}
		if len(list.items) == 0 {
			return "[]"
		}
		inner := indent + ahdTerminalIndent
		var out strings.Builder
		out.WriteString("[\n")
		for index, item := range list.items {
			out.WriteString(inner)
			out.WriteString(render(item, inner))
			if index < len(list.items)-1 {
				out.WriteByte(',')
			}
			out.WriteByte('\n')
		}
		out.WriteString(indent)
		out.WriteByte(']')
		return out.String()
	}
}

// AhdTerminalPrettyPair lays out a Pair one entry per line in insertion order.
// An empty Pair stays "{}".
func AhdTerminalPrettyPair[K comparable, V any](renderKey func(K) string, renderValue func(V, string) string) func(*AhdPair[K, V], string) string {
	return func(pair *AhdPair[K, V], indent string) string {
		if pair == nil {
			return "null"
		}
		pair.require()
		if len(pair.keys) == 0 {
			return "{}"
		}
		inner := indent + ahdTerminalIndent
		var out strings.Builder
		out.WriteString("{\n")
		for index, key := range pair.keys {
			out.WriteString(inner)
			out.WriteString(renderKey(key))
			out.WriteString(": ")
			out.WriteString(renderValue(pair.values[key], inner))
			if index < len(pair.keys)-1 {
				out.WriteByte(',')
			}
			out.WriteByte('\n')
		}
		out.WriteString(indent)
		out.WriteByte('}')
		return out.String()
	}
}

// AhdTerminalPretty writes one laid-out value and a line break through the
// buffered standard output write uses.
func AhdTerminalPretty(text string) {
	_, _ = ahdOut.WriteString(text)
	_ = ahdOut.WriteByte('\n')
}
