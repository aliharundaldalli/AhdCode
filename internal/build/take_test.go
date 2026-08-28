package build

import (
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildExecutable compiles one source to a native executable and returns its
// path, so a test can drive the program's real terminal streams.
func buildExecutable(t *testing.T, source string) string {
	t.Helper()
	directory := writeSources(t, map[string]string{"main.ahd": source})
	output := filepath.Join(t.TempDir(), "program")
	path, result := BuildProgram(filepath.Join(directory, "main.ahd"), output)
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	return path
}

// TestTakeReadsOneLineOfTerminalInput covers the v0.1 terminal input contract
// against a real executable rather than semantic metadata.
func TestTakeReadsOneLineOfTerminalInput(t *testing.T) {
	cases := []program{
		{
			name:     "take reads a line",
			sources:  map[string]string{"main.ahd": "name: String := take()\nwrite(\"[{name}]\")\n"},
			stdin:    "Ali\n",
			expected: "[Ali]\n",
		},
		{
			name:     "take with a prompt reads a line",
			sources:  map[string]string{"main.ahd": "name: String := take(\"Name: \")\nwrite(\"[{name}]\")\n"},
			stdin:    "Ali\n",
			expected: "Name: [Ali]\n",
		},
		{
			name:     "a prompt adds no newline of its own",
			sources:  map[string]string{"main.ahd": "value: String := take(\"Value: \")\nwrite(\"[{value}]\")\n"},
			stdin:    "x\n",
			expected: "Value: [x]\n",
		},
		{
			name:     "an empty prompt writes nothing",
			sources:  map[string]string{"main.ahd": "value: String := take(\"\")\nwrite(\"[{value}]\")\n"},
			stdin:    "x\n",
			expected: "[x]\n",
		},
		{
			name:     "an LF terminator is removed",
			sources:  map[string]string{"main.ahd": "value: String := take()\nwrite(\"[{value}]\")\n"},
			stdin:    "Ali\n",
			expected: "[Ali]\n",
		},
		{
			name:     "a CRLF terminator is removed",
			sources:  map[string]string{"main.ahd": "value: String := take()\nwrite(\"[{value}]\")\n"},
			stdin:    "Ali\r\n",
			expected: "[Ali]\n",
		},
		{
			name:     "ordinary whitespace is preserved",
			sources:  map[string]string{"main.ahd": "value: String := take()\nwrite(\"[{value}]\")\n"},
			stdin:    "  Ali\t \n",
			expected: "[  Ali\t ]\n",
		},
		{
			name:     "an empty entered line yields an empty String",
			sources:  map[string]string{"main.ahd": "value: String := take()\nwrite(\"[{value}]\")\nwrite(len(value))\n"},
			stdin:    "\n",
			expected: "[]\n0\n",
		},
		{
			name:     "end of input yields an empty String",
			sources:  map[string]string{"main.ahd": "value: String := take()\nwrite(\"[{value}]\")\nwrite(len(value))\n"},
			stdin:    "",
			expected: "[]\n0\n",
		},
		{
			name:     "a final line without a terminator is still read",
			sources:  map[string]string{"main.ahd": "value: String := take()\nwrite(\"[{value}]\")\n"},
			stdin:    "Ali",
			expected: "[Ali]\n",
		},
		{
			name:     "consecutive calls read consecutive lines",
			sources:  map[string]string{"main.ahd": "first: String := take()\nsecond: String := take()\nwrite(\"[{first}][{second}]\")\n"},
			stdin:    "one\ntwo\n",
			expected: "[one][two]\n",
		},
		{
			name:     "reading past the input yields empty Strings",
			sources:  map[string]string{"main.ahd": "first: String := take()\nsecond: String := take()\nwrite(\"[{first}][{second}]\")\n"},
			stdin:    "one\n",
			expected: "[one][]\n",
		},
		{
			name: "prompts and writes interleave in order",
			sources: map[string]string{"main.ahd": `ask: Function := (
) -> Nothing {
    write("before")
    value: Local String := take("Prompt: ")
    write("after {value}")
}

ask()
`},
			stdin:    "x\n",
			expected: "before\nPrompt: after x\n",
		},
		{
			name:     "take feeds the numeric conversions",
			sources:  map[string]string{"main.ahd": "number: Int := int(take(\"Number: \"))\ndecimal: Real := real(take())\nwrite(number + 1)\nwrite(decimal)\n"},
			stdin:    "41\n2.5\n",
			expected: "Number: 42\n2.5\n",
		},
		{
			name: "take drives a lazy range bound",
			sources: map[string]string{"main.ahd": `limit: Int := int(take("Limit: "))

for value: Int in between(1, limit) {
    if value == 3 {
        continue
    }

    write(value)
}
`},
			stdin:    "5\n",
			expected: "Limit: 1\n2\n4\n",
		},
	}
	runAcceptance(t, cases)
}

// TestTakePromptIsVisibleBeforeInputIsConsumed drives the program's real
// terminal streams: the prompt must reach standard output while the process is
// still blocked on standard input, so it cannot sit in an unflushed buffer.
func TestTakePromptIsVisibleBeforeInputIsConsumed(t *testing.T) {
	executable := buildExecutable(t, "name: String := take(\"Name: \")\nwrite(\"Hello {name}\")\n")
	command := exec.Command(executable)
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatalf("could not open stdin: %v", err)
	}
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatalf("could not open stdout: %v", err)
	}
	if err := command.Start(); err != nil {
		t.Fatalf("could not start the program: %v", err)
	}
	defer func() { _ = command.Wait() }()

	// Nothing has been written to stdin yet, so a prompt that arrives now can
	// only have been flushed before the read blocked.
	prompt := make(chan string, 1)
	go func() {
		buffer := make([]byte, len("Name: "))
		if _, readError := io.ReadFull(output, buffer); readError != nil {
			prompt <- ""
			return
		}
		prompt <- string(buffer)
	}()
	select {
	case received := <-prompt:
		if received != "Name: " {
			t.Fatalf("prompt was %q before any input was supplied", received)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the prompt was still buffered while the program waited for input")
	}

	if _, err := io.WriteString(input, "Ali\n"); err != nil {
		t.Fatalf("could not write input: %v", err)
	}
	_ = input.Close()
	rest, err := io.ReadAll(output)
	if err != nil {
		t.Fatalf("could not read output: %v", err)
	}
	if string(rest) != "Hello Ali\n" {
		t.Fatalf("output after the prompt was %q", string(rest))
	}
}

// TestTakeProgramsAreRejected keeps terminal input strictly typed: the read
// text is never implicitly converted.
func TestTakeProgramsAreRejected(t *testing.T) {
	cases := map[string]string{
		"Int from take":      "number: Int := take()\n",
		"Real from take":     "value: Real := take()\n",
		"Bool from take":     "flag: Bool := take()\n",
		"Int prompt":         "value: String := take(5)\n",
		"Bool prompt":        "value: String := take(true)\n",
		"two prompts":        "value: String := take(\"A\", \"B\")\n",
		"named prompt":       "value: String := take(prompt: \"A\")\n",
		"nullable prompt":    "prompt: String := null\nvalue: String := take(prompt)\n",
		"take is not a List": "values: List<String> := take()\n",
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			directory := writeSources(t, map[string]string{"main.ahd": source})
			result := Compile(filepath.Join(directory, "main.ahd"))
			if !result.HasErrors() || result.Program != nil {
				t.Fatal("expected the program to be rejected before code generation")
			}
			for _, diagnostic := range result.Diagnostics {
				if strings.HasPrefix(diagnostic.Code, "IR") || strings.HasPrefix(diagnostic.Code, "BCK") {
					t.Fatalf("a semantic root cause was followed by %s: %s", diagnostic.Code, diagnostic.Message)
				}
			}
		})
	}
}
