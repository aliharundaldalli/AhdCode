package build

import (
	"path/filepath"
	"testing"
)

// TestClassProtocolMethodsBuildAndRunNatively exercises the full v0.1.8 Class
// Protocol Method surface end to end through the real native executable:
// CEqual and its != derivation, all four CCompare-derived ordering operators
// with non-±1 sign values, every arithmetic protocol, overloaded CAdd,
// CNegate, CStr integration with str(), left-sided-only dispatch, compound
// assignment, inheritance/override with runtime type(), and id() identity
// stability across aliasing and mutation.
func TestClassProtocolMethodsBuildAndRunNatively(t *testing.T) {
	source := `Vector2: Class<> := {
    structure: Attributes := (
        x: Real
        y: Real
    )

    CEqual: Function := (
        other: Vector2
    ) -> Bool {
        return attribute.x == other.x and attribute.y == other.y
    }

    CAdd: Function := (
        other: Vector2
    ) -> Vector2 {
        return Vector2(x: attribute.x + other.x, y: attribute.y + other.y)
    }

    CAdd: Overload Function := (
        scalar: Real
    ) -> Vector2 {
        return Vector2(x: attribute.x + scalar, y: attribute.y + scalar)
    }

    CSubtract: Function := (
        other: Vector2
    ) -> Vector2 {
        return Vector2(x: attribute.x - other.x, y: attribute.y - other.y)
    }

    CMultiply: Function := (
        scalar: Real
    ) -> Vector2 {
        return Vector2(x: attribute.x * scalar, y: attribute.y * scalar)
    }

    CDivide: Function := (
        scalar: Real
    ) -> Vector2 {
        return Vector2(x: attribute.x / scalar, y: attribute.y / scalar)
    }

    CRemainder: Function := (
        other: Int
    ) -> Int {
        return int(attribute.x) % other
    }

    CPower: Function := (
        exponent: Int
    ) -> Vector2 {
        return Vector2(x: attribute.x ^ exponent, y: attribute.y ^ exponent)
    }

    CNegate: Function := (
    ) -> Vector2 {
        return Vector2(x: -attribute.x, y: -attribute.y)
    }

    CStr: Function := (
    ) -> String {
        return "Vector2({attribute.x}, {attribute.y})"
    }
}

Score: Class<> := {
    structure: Attributes := (value: Int)
    CCompare: Function := (
        other: Score
    ) -> Int {
        if attribute.value == other.value {
            return 0
        }
        if attribute.value < other.value {
            return -8
        }
        return 13
    }
}

Animal: Class<> := {
    structure: Attributes := (name: String)
    CStr: Function := () -> String { return "Animal({attribute.name})" }
}
Dog: Class<Animal> := {
    structure: Attributes := (SuperClass.attributes)
    CStr: Override Function := () -> String { return "Dog({attribute.name})" }
}

a: Vector2 := Vector2(x: 1.0, y: 2.0)
b: Vector2 := Vector2(x: 3.0, y: 4.0)
c: Vector2 := Vector2(x: 1.0, y: 2.0)

write(a + b)
write(a - b)
write(a + 10.0)
write(a * 2.0)
write(a / 2.0)
write(a % 3)
write(a ^ 2)
write(-a)
write(a == c)
write(a != b)
write(str(a))
write(10.0 + 5.0)

low: Score := Score(value: 10)
high: Score := Score(value: 90)
write(low < high)
write(low <= high)
write(low > high)
write(low >= high)

m: Vector2 := Vector2(x: 1.0, y: 1.0)
m += 4.0
write(m)

pet: Animal := Dog(name: "Rex")
write(str(pet))
write(type(pet))

first: Vector2 := Vector2(x: 0.0, y: 0.0)
second: Vector2 := first
write(id(first) == id(second))
third: Vector2 := Vector2(x: 0.0, y: 0.0)
write(id(first) == id(third))
first.x = 99.0
write(id(first) == id(second))
write(second.x)
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	out, errorOutput, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || errorOutput != "" {
		t.Fatalf("Class Protocol Method program failed: code=%d stderr=%s", code, errorOutput)
	}
	want := "Vector2(4.0, 6.0)\n" +
		"Vector2(-2.0, -2.0)\n" +
		"Vector2(11.0, 12.0)\n" +
		"Vector2(2.0, 4.0)\n" +
		"Vector2(0.5, 1.0)\n" +
		"1\n" +
		"Vector2(1.0, 4.0)\n" +
		"Vector2(-1.0, -2.0)\n" +
		"true\n" +
		"true\n" +
		"Vector2(1.0, 2.0)\n" +
		"15.0\n" +
		"true\n" +
		"true\n" +
		"false\n" +
		"false\n" +
		"Vector2(5.0, 5.0)\n" +
		"Dog(Rex)\n" +
		"Dog\n" +
		"true\n" +
		"false\n" +
		"true\n" +
		"99.0\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

// TestClassProtocolReverseDispatchIsRejected proves the left-sided-only rule:
// a primitive left operand never magically calls a Class's CAdd.
func TestClassProtocolReverseDispatchIsRejected(t *testing.T) {
	source := `Money: Class<> := {
    structure: Attributes := (amount: Real)
    CAdd: Function := (extra: Real) -> Money {
        return Money(amount: attribute.amount + extra)
    }
}
m: Money := Money(amount: 1.0)
n: Real := 2.0 + m
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	_, result := BuildProgram(filepath.Join(directory, "main.ahd"), filepath.Join(t.TempDir(), "program"))
	if !result.HasErrors() {
		t.Fatal("expected a compile-time error for 2.0 + m, got none")
	}
}

// TestClassProtocolMissingProtocolRemainsStaticError proves that an operator
// unavailable on a Class is a normal compile-time diagnostic, never a runtime
// panic.
func TestClassProtocolMissingProtocolRemainsStaticError(t *testing.T) {
	source := `Plain: Class<> := { structure: Attributes := (x: Int) }
a: Plain := Plain(x: 1)
b: Plain := Plain(x: 2)
c: Plain := a + b
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	_, result := BuildProgram(filepath.Join(directory, "main.ahd"), filepath.Join(t.TempDir(), "program"))
	if !result.HasErrors() {
		t.Fatal("expected a compile-time error for Plain + Plain, got none")
	}
}
