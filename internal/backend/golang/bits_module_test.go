package golang

import (
	"go/format"
	"strings"
	"testing"
)

// The Bits runtime must ship with every program exactly once, must be
// gofmt-valid, and must lower to the runtime helpers rather than to inline Go
// operators.

func TestBitsRuntimeEmittedExactlyOnce(t *testing.T) {
	program := generate(t, "write(\"hi\")\n")
	seen := 0
	var content string
	for _, file := range program.Files {
		if file.Name == bitsRuntimeFileName {
			seen++
			content = file.Content
		}
	}
	if seen != 1 {
		t.Fatalf("ahdcode_bits_runtime.go emitted %d times, want exactly 1", seen)
	}
	if !strings.Contains(content, "package main") {
		t.Fatal("bits runtime does not declare package main")
	}
	if strings.Contains(content, "package ahdruntime") {
		t.Fatal("the bits runtime package clause was not rewritten")
	}
	if _, err := format.Source([]byte(content)); err != nil {
		t.Fatalf("generated bits runtime is not gofmt-valid: %v", err)
	}
	// The runtime depends only on the standard library's math/bits.
	if !strings.Contains(content, `import "math/bits"`) {
		t.Fatal("bits runtime lost its math/bits import")
	}
}

// A program that never mentions Bits still carries the runtime file (every
// program does), but must not reference the Bits error class or helpers.
func TestProgramWithoutBitsDoesNotReferenceIt(t *testing.T) {
	program := generate(t, "write(\"hi\")\n")
	source := programSource(t, program)
	for _, symbol := range []string{"AhdBitsAnd", "AhdBitsShiftLeft", "cd_BitsError"} {
		if strings.Contains(source, symbol) {
			t.Fatalf("program that does not use Bits references %s", symbol)
		}
	}
}

func TestBitsLoweringEmitsRuntimeCalls(t *testing.T) {
	source := programSource(t, generate(t, `bring Bits
write(str(Bits.bitAnd(12, 10)))
write(str(Bits.bitOr(12, 10)))
write(str(Bits.bitXor(12, 10)))
write(str(Bits.bitNot(0)))
write(str(Bits.shiftLeft(1, 10)))
write(str(Bits.shiftRight(-8, 1)))
write(str(Bits.shiftRightUnsigned(-1, 60)))
write(str(Bits.rotateLeft(1, 63)))
write(str(Bits.rotateRight(1, 1)))
write(str(Bits.count(255)))
write(str(Bits.leadingZeros(1)))
write(str(Bits.trailingZeros(8)))
`))
	expected := []string{
		"AhdBitsAnd(", "AhdBitsOr(", "AhdBitsXor(", "AhdBitsNot(",
		"AhdBitsShiftLeft(", "AhdBitsShiftRight(", "AhdBitsShiftRightUnsigned(",
		"AhdBitsRotateLeft(", "AhdBitsRotateRight(",
		"AhdBitsCount(", "AhdBitsLeadingZeros(", "AhdBitsTrailingZeros(",
	}
	for _, call := range expected {
		if !strings.Contains(source, call) {
			t.Fatalf("generated program does not call %s", call)
		}
	}
	// Only the operations that can be given a bad distance take the error
	// class; the descriptor is emitted as cd_BitsError_<hash>.
	if !strings.Contains(source, "AhdBitsShiftLeft(cd_BitsError") {
		t.Fatal("shiftLeft lowering does not pass the Bits error class")
	}
	if !strings.Contains(source, "AhdBitsRotateRight(cd_BitsError") {
		t.Fatal("rotateRight lowering does not pass the Bits error class")
	}
	// Operations that cannot fail take no error class.
	if !strings.Contains(source, "AhdBitsAnd(int64(") {
		t.Fatal("bitAnd lowering should take its operands directly")
	}
	if strings.Contains(source, "AhdBitsAnd(cd_BitsError") {
		t.Fatal("bitAnd lowering should not take an error class")
	}
}
