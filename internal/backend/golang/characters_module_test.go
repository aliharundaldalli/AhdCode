package golang

import (
	"go/format"
	"strings"
	"testing"
)

func TestCharactersRuntimeEmittedExactlyOnce(t *testing.T) {
	program := generate(t, "write(\"hi\")\n")
	seen := 0
	var content string
	for _, file := range program.Files {
		if file.Name == charactersRuntimeFile {
			seen++
			content = file.Content
		}
	}
	if seen != 1 {
		t.Fatalf("%s emitted %d times, want exactly 1", charactersRuntimeFile, seen)
	}
	if !strings.Contains(content, "package main") || strings.Contains(content, "package ahdruntime") {
		t.Fatal("the Characters runtime package clause was not rewritten")
	}
	if _, err := format.Source([]byte(content)); err != nil {
		t.Fatalf("generated Characters runtime is not gofmt-valid: %v", err)
	}
	// Standard library only: no helper binary, no third-party Unicode package.
	for _, forbidden := range []string{"golang.org/x", "github.com/", "os/exec", "net/"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("Characters runtime must not import %s", forbidden)
		}
	}
}

func TestProgramWithoutCharactersDoesNotReferenceIt(t *testing.T) {
	source := programSource(t, generate(t, "write(\"hi\")\n"))
	for _, symbol := range []string{"AhdCharactersList", "AhdCharactersIsLetter", "cd_CharactersError"} {
		if strings.Contains(source, symbol) {
			t.Fatalf("program that does not use Characters references %s", symbol)
		}
	}
}

func TestCharactersLoweringEmitsRuntimeCalls(t *testing.T) {
	source := programSource(t, generate(t, `bring Characters
write(Characters.list("Aş"))
write(str(Characters.count("Aş")))
write(str(Characters.codePoint("A")))
write(Characters.fromCodePoint(351))
write(str(Characters.isLetter("ş")))
write(str(Characters.isDigit("7")))
write(str(Characters.isWhitespace(" ")))
write(str(Characters.isUpper("İ")))
write(str(Characters.isLower("ı")))
write(str(Characters.isAlphaNumeric("a")))
write(str(Characters.isPunctuation("!")))
write(str(Characters.isSymbol("+")))
`))
	for _, helper := range []string{
		"AhdCharactersList(", "AhdCharactersCount(", "AhdCharactersCodePoint(", "AhdCharactersFromCodePoint(",
		"AhdCharactersIsLetter(", "AhdCharactersIsDigit(", "AhdCharactersIsWhitespace(", "AhdCharactersIsUpper(",
		"AhdCharactersIsLower(", "AhdCharactersIsAlphaNumeric(", "AhdCharactersIsPunctuation(", "AhdCharactersIsSymbol(",
	} {
		if !strings.Contains(source, helper) {
			t.Fatalf("generated program does not call %s", helper)
		}
	}
}
