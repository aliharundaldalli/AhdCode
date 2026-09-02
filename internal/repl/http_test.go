package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestHTTPCookieWorksInTheREPL(t *testing.T) {
	input := `bring HTTP
from HTTP bring Cookie
cookie := HTTP.cookie("theme", "dark")
cookie = cookie.withHttpOnly(true)
write("ok")
`
	var output, errorOutput bytes.Buffer
	Run(strings.NewReader(input), &output, &errorOutput, "AhdCode v0.5.0")
	if !strings.Contains(output.String(), "ok") {
		t.Fatalf("REPL cookie output:\n%s\nerrors:\n%s", output.String(), errorOutput.String())
	}
	if errorOutput.Len() != 0 {
		t.Fatalf("REPL reported errors:\n%s", errorOutput.String())
	}
}

func TestHTTPSessionsWorkInTheREPL(t *testing.T) {
	input := `bring HTTP
from HTTP bring SessionStore
store := HTTP.sessions()
write("ok")
`
	var output, errorOutput bytes.Buffer
	Run(strings.NewReader(input), &output, &errorOutput, "AhdCode v0.5.0")
	if !strings.Contains(output.String(), "ok") {
		t.Fatalf("REPL sessions output:\n%s\nerrors:\n%s", output.String(), errorOutput.String())
	}
	if errorOutput.Len() != 0 {
		t.Fatalf("REPL reported errors:\n%s", errorOutput.String())
	}
}
