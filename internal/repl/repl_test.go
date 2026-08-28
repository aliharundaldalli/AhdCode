package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestIncompleteUsesCompilerTokens(t *testing.T) {
	for _, text := range []string{"if true {\n", "add: Function := (\n", `text: String := """hello`} {
		if !Incomplete(text) {
			t.Fatalf("expected incomplete input: %q", text)
		}
	}
	for _, text := range []string{"x: Int := )\n", "write(1)\n", "if true {\n write(1)\n}\n\n"} {
		if Incomplete(text) {
			t.Fatalf("expected complete input: %q", text)
		}
	}
}

func TestPersistentSessionMutationErrorsAndDeclarations(t *testing.T) {
	input := strings.Join([]string{
		"x: Int := 5",
		"write(x ^ 2)",
		"x = 7",
		"write(x)",
		"x: Int := 9",
		"write(1 / 0)",
		"write(x)",
	}, "\n") + "\n"
	var output, errors bytes.Buffer
	if code := Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1"); code != 0 {
		t.Fatalf("REPL exit = %d", code)
	}
	if !strings.Contains(output.String(), "25\n") || strings.Count(output.String(), "7\n") != 2 {
		t.Fatalf("REPL output:\n%s", output.String())
	}
	if !strings.Contains(errors.String(), "already declared") || !strings.Contains(errors.String(), "DivisionByZeroError") {
		t.Fatalf("REPL errors:\n%s", errors.String())
	}
}

func TestMultilineFunctionAndClassPersist(t *testing.T) {
	input := `add: Function := (
    x: Int
    y: Int
) -> Int {
    return x + y
}
write(add(2, 3))
Box: Class<> := {
    structure: Attributes := (
        value: Int
    )
}
box: Box := Box(value: 9)
write(box.value)
if true {
    write("block")
}
text: String := """first
second"""
write(text)
if false {
    write("wrong")
}
else {
    write("else")
}
attempt {
    write(1 % 0)
}
except DivisionByZeroError as error {
    write("caught")
}
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1")
	if !strings.Contains(output.String(), "5\n") || !strings.Contains(output.String(), "9\n") || !strings.Contains(output.String(), "block\n") || !strings.Contains(output.String(), "first\nsecond\n") || !strings.Contains(output.String(), "else\n") || !strings.Contains(output.String(), "caught\n") {
		t.Fatalf("REPL output:\n%s\nerrors:\n%s", output.String(), errors.String())
	}
	if errors.Len() != 0 {
		t.Fatalf("REPL errors:\n%s", errors.String())
	}
}

func TestNumericConversionsAndPowerUseTheSharedPipeline(t *testing.T) {
	input := "write(real(2))\nwrite(real(2) ^ -3)\nwrite(2 ^ -3)\nwrite(int(3.7))\n"
	var output, errors bytes.Buffer
	if code := Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1"); code != 0 {
		t.Fatalf("REPL exit = %d", code)
	}
	if !strings.Contains(output.String(), "2.0\n") || !strings.Contains(output.String(), "0.125\n") || !strings.Contains(output.String(), "3\n") {
		t.Fatalf("REPL output:\n%s", output.String())
	}
	if !strings.Contains(errors.String(), "DomainError") {
		t.Fatalf("REPL errors:\n%s", errors.String())
	}
}
