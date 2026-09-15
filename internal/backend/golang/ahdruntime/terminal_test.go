package ahdruntime

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestTerminalJoinIsExact(t *testing.T) {
	for _, testCase := range []struct {
		parts             []string
		separator, ending string
		want              string
	}{
		{nil, " ", "\n", "\n"},
		{[]string{}, " | ", "", ""},
		{[]string{"Ali"}, " ", "\n", "Ali\n"},
		{[]string{"Ali", "95", "Passed"}, " ", "\n", "Ali 95 Passed\n"},
		{[]string{"Ali", "95", "Passed"}, " | ", "\n", "Ali | 95 | Passed\n"},
		{[]string{"a", "b", "c"}, "", "", "abc"},
		{[]string{"Loading"}, " ", "", "Loading"},
		{[]string{"ç", "ğ", "🙂"}, " · ", "\r\n", "ç · ğ · 🙂\r\n"},
		{[]string{"line\nbreak", "x"}, "|", "!", "line\nbreak|x!"},
		{[]string{"", ""}, ",", "\n", ",\n"},
	} {
		if got := AhdTerminalJoin(testCase.parts, testCase.separator, testCase.ending); got != testCase.want {
			t.Fatalf("AhdTerminalJoin(%q, %q, %q) = %q, want %q", testCase.parts, testCase.separator, testCase.ending, got, testCase.want)
		}
	}
}

func TestTerminalColorPolicy(t *testing.T) {
	environment := func(values map[string]string) func(string) (string, bool) {
		return func(name string) (string, bool) {
			value, present := values[name]
			return value, present
		}
	}
	for _, testCase := range []struct {
		name                   string
		interactive, sequences bool
		values                 map[string]string
		want                   bool
	}{
		{"interactive terminal, no variables", true, true, nil, true},
		{"interactive terminal with a TERM", true, true, map[string]string{"TERM": "xterm-256color"}, true},
		{"redirected output", false, true, nil, false},
		{"redirected output ignores the environment", false, true, map[string]string{"TERM": "xterm"}, false},
		{"console without escape sequences", true, false, nil, false},
		{"TERM=dumb", true, true, map[string]string{"TERM": "dumb"}, false},
		{"NO_COLOR set", true, true, map[string]string{"NO_COLOR": "1"}, false},
		{"NO_COLOR set to any text", true, true, map[string]string{"NO_COLOR": "false"}, false},
		{"NO_COLOR present but empty", true, true, map[string]string{"NO_COLOR": ""}, true},
		{"unrelated variables are not consulted", true, true, map[string]string{"FORCE_COLOR": "0", "CLICOLOR": "0", "COLORTERM": ""}, true},
	} {
		if got := AhdTerminalColorPolicy(testCase.interactive, testCase.sequences, environment(testCase.values)); got != testCase.want {
			t.Fatalf("%s: color policy = %v, want %v", testCase.name, got, testCase.want)
		}
	}
}

func TestTerminalStyleColorsAndAttributes(t *testing.T) {
	names := []string{"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"}
	for index, name := range names {
		got, problem := AhdTerminalStyle("x", name, "default", false, false, true)
		if want := "\x1b[" + strconv.Itoa(30+index) + "mx\x1b[0m"; problem != "" || got != want {
			t.Fatalf("foreground %s = %q (%s), want %q", name, got, problem, want)
		}
		got, problem = AhdTerminalStyle("x", "default", name, false, false, true)
		if want := "\x1b[" + strconv.Itoa(40+index) + "mx\x1b[0m"; problem != "" || got != want {
			t.Fatalf("background %s = %q (%s), want %q", name, got, problem, want)
		}
	}
	for _, testCase := range []struct {
		text, foreground, background string
		bold, underline              bool
		want                         string
	}{
		{"OK", "default", "default", false, false, "OK"},
		{"OK", "default", "default", true, false, "\x1b[1mOK\x1b[0m"},
		{"OK", "default", "default", false, true, "\x1b[4mOK\x1b[0m"},
		{"OK", "green", "default", true, false, "\x1b[1;32mOK\x1b[0m"},
		{"OK", "green", "white", true, true, "\x1b[1;4;32;47mOK\x1b[0m"},
		{"çğ🙂", "red", "default", false, false, "\x1b[31mçğ🙂\x1b[0m"},
		{"", "default", "default", true, false, "\x1b[1m\x1b[0m"},
		{"two\nlines", "cyan", "default", false, false, "\x1b[36mtwo\nlines\x1b[0m"},
	} {
		got, problem := AhdTerminalStyle(testCase.text, testCase.foreground, testCase.background, testCase.bold, testCase.underline, true)
		if problem != "" || got != testCase.want {
			t.Fatalf("style(%q, %s, %s, bold=%v, underline=%v) = %q (%s), want %q", testCase.text, testCase.foreground, testCase.background,
				testCase.bold, testCase.underline, got, problem, testCase.want)
		}
	}
}

func TestTerminalStyleDegradesToPlainText(t *testing.T) {
	for _, foreground := range []string{"default", "red", "white"} {
		for _, background := range []string{"default", "blue"} {
			for _, bold := range []bool{false, true} {
				got, problem := AhdTerminalStyle("Başarılı ✓", foreground, background, bold, !bold, false)
				if problem != "" || got != "Başarılı ✓" {
					t.Fatalf("without color, style = %q (%s), want the plain text", got, problem)
				}
			}
		}
	}
}

func TestTerminalStyleRejectsUnknownNames(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		for _, name := range []string{"purple", "Red", "RED", "", " red", "bright-red", "256", "#ff0000"} {
			got, problem := AhdTerminalStyle("x", name, "default", false, false, enabled)
			if got != "" || !strings.HasPrefix(problem, "unsupported foreground color "+strconv.Quote(name)+"; use default, black") {
				t.Fatalf("foreground %q (color %v) = %q, %q", name, enabled, got, problem)
			}
			got, problem = AhdTerminalStyle("x", "red", name, false, false, enabled)
			if got != "" || !strings.HasPrefix(problem, "unsupported background color "+strconv.Quote(name)+"; use default, black") {
				t.Fatalf("background %q (color %v) = %q, %q", name, enabled, got, problem)
			}
		}
	}
}

// A styled String never leaves styling active, and styling applied around an
// already styled String resumes after the inner reset.
func TestTerminalStyleNestingResumesOuterStyle(t *testing.T) {
	inner, _ := AhdTerminalStyle("warning", "yellow", "default", false, false, true)
	if inner != "\x1b[33mwarning\x1b[0m" {
		t.Fatalf("inner = %q", inner)
	}
	outer, _ := AhdTerminalStyle("Status: "+inner+" now", "default", "default", true, false, true)
	if want := "\x1b[1mStatus: \x1b[33mwarning\x1b[0m\x1b[1m now\x1b[0m"; outer != want {
		t.Fatalf("outer = %q, want %q", outer, want)
	}
	trailing, _ := AhdTerminalStyle("x "+inner, "default", "default", true, false, true)
	if want := "\x1b[1mx \x1b[33mwarning\x1b[0m\x1b[0m"; trailing != want {
		t.Fatalf("outer ending in a styled String = %q, want %q", trailing, want)
	}
	for _, value := range []string{inner, outer, trailing} {
		if !strings.HasSuffix(value, ahdTerminalReset) {
			t.Fatalf("styled text does not end with a reset: %q", value)
		}
	}
}

func TestTerminalDimensionsAreNullWithoutATerminal(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	file, err := os.Create(filepath.Join(t.TempDir(), "output.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer devNull.Close()
	for name, handle := range map[string]*os.File{"pipe reader": reader, "pipe writer": writer, "regular file": file, "null device": devNull} {
		fd := handle.Fd()
		if AhdTerminalIsTerminal(fd) {
			t.Fatalf("%s reported as a terminal", name)
		}
		if width, height := AhdTerminalDimensions(fd); width != nil || height != nil {
			t.Fatalf("%s has dimensions %v x %v, want null", name, width, height)
		}
		if AhdTerminalColorEnabled(fd) {
			t.Fatalf("%s reported color support", name)
		}
	}
}

func TestTerminalDimensionsNeverInventASize(t *testing.T) {
	for _, value := range []int{0, -1, -80} {
		if got := ahdTerminalPositive(value); got != nil {
			t.Fatalf("dimension %d became %d, want null", value, *got)
		}
	}
	if got := ahdTerminalPositive(132); got == nil || *got != 132 {
		t.Fatalf("dimension 132 became %v", got)
	}
}

func TestTerminalPrettyLayout(t *testing.T) {
	integers := AhdTerminalPrettyList[int64](AhdTerminalPrettyLeaf[int64](AhdStrInt))
	if got := integers(AhdNewList[int64](90, 95, 100), ""); got != "[\n    90,\n    95,\n    100\n]" {
		t.Fatalf("List<Int> = %q", got)
	}
	if got := integers(AhdNewList[int64](), ""); got != "[]" {
		t.Fatalf("empty List = %q", got)
	}
	if got := integers(nil, ""); got != "null" {
		t.Fatalf("null List = %q", got)
	}
	grades := AhdBuildPair([]string{"Ali", "Ayşe"}, []*AhdList[int64]{AhdNewList[int64](90, 95, 100), AhdNewList[int64](85, 91, 97)})
	table := AhdTerminalPrettyPair[string, *AhdList[int64]](AhdStrQuoted, integers)
	want := "{\n    \"Ali\": [\n        90,\n        95,\n        100\n    ],\n    \"Ayşe\": [\n        85,\n        91,\n        97\n    ]\n}"
	if got := table(grades, ""); got != want {
		t.Fatalf("Pair<String, List<Int>> =\n%s\nwant\n%s", got, want)
	}
	texts := AhdTerminalPrettyList[string](AhdTerminalPrettyLeaf[string](AhdStrQuoted))
	if got := texts(AhdNewList("a\"b", "line\nbreak", "tab\there"), ""); got != "[\n    \"a\\\"b\",\n    \"line\\nbreak\",\n    \"tab\\there\"\n]" {
		t.Fatalf("escaped Strings = %q", got)
	}
	empty := AhdTerminalPrettyPair[string, int64](AhdStrQuoted, AhdTerminalPrettyLeaf[int64](AhdStrInt))
	if got := empty(AhdBuildPair([]string{}, []int64{}), ""); got != "{}" {
		t.Fatalf("empty Pair = %q", got)
	}
	half := 1.5
	optional := AhdTerminalPrettyList[*float64](AhdTerminalPrettyLeaf[*float64](AhdStrNull[float64](AhdStrReal)))
	if got := optional(AhdNewList[*float64](&half, nil), ""); got != "[\n    1.5,\n    null\n]" {
		t.Fatalf("List<Real?> = %q", got)
	}
	flags := AhdTerminalPrettyPair[int64, bool](AhdStrInt, AhdTerminalPrettyLeaf[bool](AhdStrBool))
	if got := flags(AhdBuildPair([]int64{1, 2}, []bool{true, false}), "  "); got != "{\n      1: true,\n      2: false\n  }" {
		t.Fatalf("indented Pair<Int, Bool> = %q", got)
	}
}
