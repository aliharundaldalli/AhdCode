package evaluator

import (
	"testing"

	"ahdcode/internal/semantic"
)

// Expected values below come from the definition of each operation on a signed
// 64-bit two's-complement integer, not from the implementation itself.

func TestBitsLogicalOperations(t *testing.T) {
	session := newSecurityTestSession()
	cases := []struct {
		name string
		args []any
		want int64
	}{
		// 12 = 1100b, 10 = 1010b
		{"bitAnd", []any{int64(12), int64(10)}, 8}, // 1000b
		{"bitOr", []any{int64(12), int64(10)}, 14}, // 1110b
		{"bitXor", []any{int64(12), int64(10)}, 6}, // 0110b
		{"bitNot", []any{int64(0)}, -1},            // all ones
		{"bitNot", []any{int64(-1)}, 0},            // and back
		{"bitAnd", []any{int64(-1), int64(255)}, 255},
		{"bitXor", []any{int64(-1), int64(-1)}, 0},
		{"bitOr", []any{int64(0), int64(0)}, 0},
	}
	for _, test := range cases {
		got := session.bitsBuiltin(test.name, test.args).(int64)
		if got != test.want {
			t.Fatalf("%s%v = %d, want %d", test.name, test.args, got, test.want)
		}
	}
}

func TestBitsShiftsAndRotations(t *testing.T) {
	session := newSecurityTestSession()
	const minInt64 = int64(-9223372036854775808)
	cases := []struct {
		name string
		args []any
		want int64
	}{
		{"shiftLeft", []any{int64(1), int64(0)}, 1},
		{"shiftLeft", []any{int64(1), int64(10)}, 1024},
		// Shifting into the sign bit is defined behaviour, not an error.
		{"shiftLeft", []any{int64(1), int64(63)}, minInt64},
		// shiftRight is arithmetic: the sign bit is replicated.
		{"shiftRight", []any{int64(-8), int64(1)}, -4},
		{"shiftRight", []any{int64(-1), int64(63)}, -1},
		{"shiftRight", []any{int64(1024), int64(10)}, 1},
		// shiftRightUnsigned fills with zeros instead.
		{"shiftRightUnsigned", []any{int64(-1), int64(60)}, 15},
		{"shiftRightUnsigned", []any{int64(-1), int64(63)}, 1},
		{"shiftRightUnsigned", []any{int64(8), int64(3)}, 1},
		// Rotations lose no bits and round-trip.
		{"rotateLeft", []any{int64(1), int64(63)}, minInt64},
		{"rotateLeft", []any{int64(1), int64(0)}, 1},
		{"rotateRight", []any{int64(1), int64(1)}, minInt64},
		{"rotateLeft", []any{minInt64, int64(1)}, 1},
	}
	for _, test := range cases {
		got := session.bitsBuiltin(test.name, test.args).(int64)
		if got != test.want {
			t.Fatalf("%s%v = %d, want %d", test.name, test.args, got, test.want)
		}
	}
}

func TestBitsPopulationCounts(t *testing.T) {
	session := newSecurityTestSession()
	cases := []struct {
		name string
		arg  int64
		want int64
	}{
		{"count", 0, 0},
		{"count", 255, 8},
		{"count", -1, 64}, // every bit set
		{"leadingZeros", 0, 64},
		{"leadingZeros", 1, 63},
		{"leadingZeros", -1, 0}, // sign bit set
		{"trailingZeros", 8, 3},
		{"trailingZeros", 1, 0},
		{"trailingZeros", 0, 64},
	}
	for _, test := range cases {
		got := session.bitsBuiltin(test.name, []any{test.arg}).(int64)
		if got != test.want {
			t.Fatalf("%s(%d) = %d, want %d", test.name, test.arg, got, test.want)
		}
	}
}

// A distance outside 0..63 is a caller bug: raising beats silently returning
// zero, which would hide the mistake inside a larger calculation.
func TestBitsRejectsOutOfRangeDistance(t *testing.T) {
	session := newSecurityTestSession()
	for _, name := range []string{"shiftLeft", "shiftRight", "shiftRightUnsigned", "rotateLeft", "rotateRight"} {
		for _, distance := range []int64{-1, 64, 1000} {
			name, distance := name, distance
			expectEvaluatorRaise(t, "BitsError", func() {
				session.bitsBuiltin(name, []any{int64(1), distance})
			})
		}
	}
}

func TestBitsModuleRegisteredForEvaluator(t *testing.T) {
	modules := semantic.StandardModuleInterfaces()
	module, ok := modules["Bits"]
	if !ok {
		t.Fatal("Bits module not found in StandardModuleInterfaces")
	}
	if module.Exports["BitsError"] == nil {
		t.Fatal("Bits module missing BitsError export")
	}
}
