package build

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestErrorHandlingProgramsRunAsNativeExecutables covers the attempt, except,
// ultimately, and toss runtime contract end to end.
func TestErrorHandlingProgramsRunAsNativeExecutables(t *testing.T) {
	cases := []program{
		{
			name: "caught custom error",
			sources: map[string]string{"main.ahd": `attempt {
    toss Error(
        message: "boom"
    )
}
except Error as error {
    write(error.message)
}
`},
			expected: "boom\n",
		},
		{
			name: "derived error class",
			sources: map[string]string{"main.ahd": `InvalidAgeError: Class<Error> := {
    structure: Attributes := (
        SuperClass.attributes
        age: Int
    )
}

attempt {
    toss InvalidAgeError(message: "too young", age: 3)
}
except InvalidAgeError as error {
    write("{error.message} {error.age}")
}
`},
			expected: "too young 3\n",
		},
		{
			name: "built-in arithmetic error is catchable",
			sources: map[string]string{"main.ahd": `attempt {
    x: Local Int := 1 % 0
    write(x)
}
except DivisionByZeroError as error {
    write("caught")
}
`},
			expected: "caught\n",
		},
		{
			name: "built-in index error is catchable",
			sources: map[string]string{"main.ahd": `values: List<Int> := [1]
attempt {
    write(str(values[5]))
}
except IndexError as error {
    write("index: {error.message}")
}
`},
			expected: "index: index 5 is out of range for length 1\n",
		},
		{
			name: "a base handler catches a derived error",
			sources: map[string]string{"main.ahd": `Specific: Class<Error> := {
    structure: Attributes := (
        SuperClass.attributes
    )
}

attempt {
    toss Specific(message: "specific")
}
except Error as error {
    write("base handler {error.message}")
}
`},
			expected: "base handler specific\n",
		},
		{
			name: "the first matching handler in source order runs",
			sources: map[string]string{"main.ahd": `Specific: Class<Error> := {
    structure: Attributes := (
        SuperClass.attributes
    )
}

attempt {
    toss Specific(message: "x")
}
except Specific as error {
    write("specific")
}
except Error as error {
    write("general")
}
`},
			expected: "specific\n",
		},
		{
			name: "ultimately runs before a pending return",
			sources: map[string]string{"main.ahd": `example: Function := (
) -> Int {
    attempt {
        return 5
    }
    ultimately {
        write("cleanup")
    }
}

write(example())
`},
			expected: "cleanup\n5\n",
		},
		{
			name: "ultimately runs on the normal, handled, and propagating paths",
			sources: map[string]string{"main.ahd": `run: Function := (
    mode: Int
) -> Nothing {
    attempt {
        if mode == 1 {
            toss Error(message: "handled")
        }

        if mode == 2 {
            toss Error(message: "propagated")
        }

        write("normal")
    }
    except Error as error {
        if mode == 2 {
            toss error
        }

        write("handled {error.message}")
    }
    ultimately {
        write("ultimately {mode}")
    }
}

run(0)
run(1)
attempt {
    run(2)
}
except Error as error {
    write("outer {error.message}")
}
`},
			expected: "normal\nultimately 0\nhandled handled\nultimately 1\nultimately 2\nouter propagated\n",
		},
		{
			name: "ultimately runs before break and continue",
			sources: map[string]string{"main.ahd": `i: Int := 0
while i < 3 {
    i++
    attempt {
        if i == 2 {
            continue
        }

        if i == 3 {
            break
        }

        write("body {i}")
    }
    ultimately {
        write("ultimately {i}")
    }
}

write("after {i}")
`},
			expected: "body 1\nultimately 1\nultimately 2\nultimately 3\nafter 3\n",
		},
		{
			name: "an error tossed by ultimately replaces the active error",
			sources: map[string]string{"main.ahd": `attempt {
    attempt {
        toss Error(message: "original")
    }
    ultimately {
        toss Error(message: "replacement")
    }
}
except Error as error {
    write(error.message)
}
`},
			expected: "replacement\n",
		},
		{
			name: "nested attempt inside a handler and an ultimately",
			sources: map[string]string{"main.ahd": `attempt {
    toss Error(message: "outer")
}
except Error as error {
    attempt {
        toss Error(message: "inside handler")
    }
    except Error as inner {
        write("handler nested {inner.message}")
    }
}
ultimately {
    attempt {
        toss Error(message: "inside ultimately")
    }
    except Error as inner {
        write("ultimately nested {inner.message}")
    }
}
`},
			expected: "handler nested inside handler\nultimately nested inside ultimately\n",
		},
		{
			name: "return through nested attempts evaluates once",
			sources: map[string]string{"main.ahd": `calls: Int := 0

value: Function := (
) -> Int {
    calls: Global Int
    calls = calls + 1
    return 42
}

example: Function := (
) -> Int {
    attempt {
        attempt {
            return value()
        }
        ultimately {
            write("inner")
        }
    }
    ultimately {
        write("outer")
    }
}

write(example())
write(calls)
`},
			expected: "inner\nouter\n42\n1\n",
		},
		{
			name:       "an uncaught error reports its Class and message",
			sources:    map[string]string{"main.ahd": "write(\"before\")\ntoss Error(message: \"uncaught boom\")\n"},
			expected:   "before\n",
			exitCode:   1,
			errorClass: "Error",
		},
	}
	runAcceptance(t, cases)
}

// TestPowerRevisionRunsAsNativeExecutable locks the operand-type-only power
// rule and its checked Int runtime behavior through the full compiler chain.
func TestPowerRevisionRunsAsNativeExecutable(t *testing.T) {
	cases := []program{{
		name: "Int power typing, compound assignment, and runtime errors",
		sources: map[string]string{"main.ahd": `write(2 ^ 3)

base: Int := 5
exponent: Int := 7
total: Int := base ^ exponent
write(total)

constantBase: Constant Int := 5
constantExponent: Constant Int := 7
constantTotal: Int := constantBase ^ constantExponent
write(constantTotal)
write(base ^ constantExponent)
write(constantBase ^ exponent)

write(2.0 ^ -1)
write(2 ^ 3.0)

attempt {
    write(2 ^ -1)
}
except DomainError as error {
    write("literal domain")
}

negative: Int := -1
attempt {
    write(base ^ negative)
}
except DomainError as error {
    write("mutable domain")
}

largeBase: Int := 10
largeExponent: Int := 100
attempt {
    write(largeBase ^ largeExponent)
}
except OverflowError as error {
    write("overflow")
}

compound: Int := 2
compoundExponent: Int := 3
compound ^= compoundExponent
write(compound)

attempt {
    compound ^= negative
}
except DomainError as error {
    write("compound domain")
}

realCompound: Real := 2.0
realCompound ^= compoundExponent
write(realCompound)
realCompound = 4.0
realExponent: Real := 0.5
realCompound ^= realExponent
write(realCompound)
`},
		expected: "8\n78125\n78125\n78125\n78125\n0.5\n8.0\nliteral domain\nmutable domain\noverflow\n8\ncompound domain\n8.0\n2.0\n",
	}}
	runAcceptance(t, cases)
}

// TestInheritanceProgramsRunAsNativeExecutables covers single inheritance,
// override dispatch, SuperClass, upcast identity, and type membership.
func TestInheritanceProgramsRunAsNativeExecutables(t *testing.T) {
	cases := []program{
		{
			name: "override dispatches through a parent-typed reference",
			sources: map[string]string{"main.ahd": `Person: Class<> := {
    structure: Attributes := (
        name: String
    )

    describe: Function := (
    ) -> String {
        return "Person: {attribute.name}"
    }
}

Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )

    describe: Override Function := (
    ) -> String {
        return "Student: {attribute.name} #{attribute.number}"
    }
}

student: Student := Student(
    name: "Ali"
    number: 1
)

person: Person := student
write(student.describe())
write(person.describe())
write(person.name)
`},
			expected: "Student: Ali #1\nStudent: Ali #1\nAli\n",
		},
		{
			name: "SuperClass calls the parent implementation on the same instance",
			sources: map[string]string{"main.ahd": `Person: Class<> := {
    structure: Attributes := (
        name: String
    )

    describe: Function := (
    ) -> String {
        return "Person {attribute.name}"
    }
}

Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )

    describe: Override Function := (
    ) -> String {
        return "{SuperClass.describe()} / Student #{attribute.number}"
    }
}

GraduateStudent: Class<Student> := {
    structure: Attributes := (
        SuperClass.attributes
        topic: String
    )

    describe: Override Function := (
    ) -> String {
        return "{SuperClass.describe()} / Graduate {attribute.topic}"
    }
}

graduate: GraduateStudent := GraduateStudent(
    name: "Ada"
    number: 7
    topic: "Types"
)

write(graduate.describe())
person: Person := graduate
write(person.describe())
`},
			expected: "Person Ada / Student #7 / Graduate Types\nPerson Ada / Student #7 / Graduate Types\n",
		},
		{
			name: "inheritance-aware type membership and identity",
			sources: map[string]string{"main.ahd": `Person: Class<> := {
    structure: Attributes := (
        name: String
    )
}

Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
    )
}

Teacher: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
    )
}

student: Student := Student(name: "Ada")
person: Person := student
other: Person := Person(name: "Bob")

write(str(student is Student))
write(str(student is Person))
write(str(student is Object))
write(str(student is Teacher))
write(str(person is Student))
write(str(other is Student))
write(str(person == student))
write(str(person same student))
write(str(person same other))
write(str(person))
`},
			expected: "true\ntrue\ntrue\nfalse\ntrue\nfalse\ntrue\ntrue\nfalse\n<Student>\n",
		},
		{
			name: "an inherited method runs on the derived instance",
			sources: map[string]string{"main.ahd": `Counter: Class<> := {
    structure: Attributes := (
        value: Int
    )

    bump: Function := (
    ) -> Nothing {
        attribute.value++
    }

    report: Function := (
    ) -> String {
        return "value={attribute.value}"
    }
}

Doubling: Class<Counter> := {
    structure: Attributes := (
        SuperClass.attributes
    )

    bump: Override Function := (
    ) -> Nothing {
        attribute.value = attribute.value + 2
    }
}

counter: Counter := Doubling(value: 1)
counter.bump()
write(counter.report())
`},
			expected: "value=3\n",
		},
		{
			name: "cross-module inheritance keeps one canonical Class identity",
			sources: map[string]string{
				"Models.ahd": `Person: Class<> := {
    structure: Attributes := (
        name: String
    )

    describe: Function := (
    ) -> String {
        return "Person {attribute.name}"
    }
}
`,
				"Students.ahd": `from Models bring Person

Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )

    describe: Override Function := (
    ) -> String {
        return "{SuperClass.describe()} #{attribute.number}"
    }
}
`,
				"main.ahd": `from Models bring Person
from Students bring Student

student: Student := Student(
    name: "Ada"
    number: 3
)

person: Person := student
write(student.describe())
write(person.describe())
write(person.name)
write(str(person is Student))
write(str(person same student))
`,
			},
			expected: "Person Ada #3\nPerson Ada #3\nAda\ntrue\ntrue\n",
		},
		{
			name: "a same-named Class in another module is a different runtime type",
			sources: map[string]string{
				"Other.ahd": `Shape: Class<> := {
    structure: Attributes := (
        tag: String
    )
}

make: Function := (
) -> Shape {
    return Shape(tag: "foreign")
}
`,
				"main.ahd": `from Other bring make

Shape: Class<> := {
    structure: Attributes := (
        tag: String
    )
}

check: Function := (
    value: Object
) -> Bool {
    return value is Shape
}

local: Shape := Shape(tag: "local")
write(str(check(local)))
write(str(check(make())))
write(make().tag)
`,
			},
			expected: "true\nfalse\nforeign\n",
		},
	}
	runAcceptance(t, cases)
}

// TestNullableCallableProgramsRunAsNativeExecutables covers nullable Function
// parameters, nullable returns, and nullable callbacks.
func TestNullableCallableProgramsRunAsNativeExecutables(t *testing.T) {
	cases := []program{
		{
			name: "nullable Class and scalar returns",
			sources: map[string]string{"main.ahd": `Student: Class<> := {
    structure: Attributes := (
        name: String
    )
}

find: Function := (
    id: Int
) -> Student {
    if id == 1 {
        return Student(name: "Ali")
    }

    return null
}

maybeNumber: Function := (
    ok: Bool
) -> Int {
    if ok {
        return 5
    }

    return null
}

found: Student := find(1)
if found != null {
    write(found.name)
}

missing: Student := find(2)
if missing == null {
    write("missing")
}

present: Int := maybeNumber(true)
if present != null {
    write(present)
}

absent: Int := maybeNumber(false)
if absent == null {
    write("no number")
}
`},
			expected: "Ali\nmissing\n5\nno number\n",
		},
		{
			name: "nullable Function parameter with a default",
			sources: map[string]string{"main.ahd": `greet: Function := (
    name: String := null
) -> String {
    if name != null {
        return "Hello {name}"
    }

    return "Hello stranger"
}

write(greet("Ada"))
write(greet())
write(greet(null))
`},
			expected: "Hello Ada\nHello stranger\nHello stranger\n",
		},
		{
			name: "a callback keeps the nullable return of the callable assigned to it",
			sources: map[string]string{"main.ahd": `maybeDouble: Function := (
    n: Int
) -> Int {
    if n > 0 {
        return n * 2
    }

    return null
}

callback: Function := maybeDouble

present: Int := callback(4)
if present != null {
    write(present)
}

absent: Int := callback(-1)
if absent == null {
    write("null result")
}
`},
			expected: "8\nnull result\n",
		},
		{
			name: "a NonNull callback still satisfies an inferred callback contract",
			sources: map[string]string{"main.ahd": `apply: Function := (
    action: Function
    value: Int
) -> Int {
    return action(value)
}

double: Function := (
    n: Int
) -> Int {
    return n * 2
}

write(apply(double, 21))
`},
			expected: "42\n",
		},
		{
			name: "an imported callable keeps its null contract",
			sources: map[string]string{
				"Lookup.ahd": `find: Function := (
    id: Int
) -> String {
    if id == 1 {
        return "one"
    }

    return null
}
`,
				"main.ahd": `from Lookup bring find

hit: String := find(1)
if hit != null {
    write(hit)
}

miss: String := find(2)
if miss == null {
    write("miss")
}
`,
			},
			expected: "one\nmiss\n",
		},
		{
			name: "an overloaded callable selects the concrete nullable signature",
			sources: map[string]string{"main.ahd": `pick: Function := (
    value: Int
) -> String {
    if value > 0 {
        return "int"
    }

    return null
}

pick: Overload Function := (
    value: Real
) -> String {
    return "real"
}

hit: String := pick(1)
if hit != null {
    write(hit)
}

miss: String := pick(0)
if miss == null {
    write("null int")
}

write(pick(1.5))
`},
			expected: "int\nnull int\nreal\n",
		},
	}
	runAcceptance(t, cases)
}

// TestDeepFreezeProgramsRunAsNativeExecutables covers the Constant
// deep-freeze-through-alias runtime contract.
func TestDeepFreezeProgramsRunAsNativeExecutables(t *testing.T) {
	cases := []program{
		{
			name: "an alias cannot mutate a frozen List",
			sources: map[string]string{"main.ahd": `source: List<Int> := [1, 2, 3]
frozen: Constant List<Int> := source

attempt {
    clear(source)
    write("mutated")
}
except ConstantError as error {
    write("rejected")
}

write(len(source))
`},
			expected: "rejected\n3\n",
		},
		{
			name: "a frozen graph rejects nested mutation",
			sources: map[string]string{"main.ahd": `inner: List<Int> := [1, 2]
outer: List<List<Int>> := [inner]
frozen: Constant List<List<Int>> := outer

attempt {
    clear(inner)
}
except ConstantError as error {
    write("nested rejected")
}

write(len(inner))
`},
			expected: "nested rejected\n2\n",
		},
		{
			name: "a frozen Class rejects field and attribute-graph mutation",
			sources: map[string]string{"main.ahd": `Box: Class<> := {
    structure: Attributes := (
        label: String
        values: List<Int>
    )
}

box: Box := Box(
    label: "b"
    values: [1, 2, 3]
)

frozen: Constant Box := box

attempt {
    box.label = "changed"
}
except ConstantError as error {
    write("field rejected")
}

attempt {
    clear(box.values)
}
except ConstantError as error {
    write("graph rejected")
}

write(box.label)
write(len(box.values))
`},
			expected: "field rejected\ngraph rejected\nb\n3\n",
		},
		{
			name: "a frozen Pair rejects insertion and clearing",
			sources: map[string]string{"main.ahd": `scores: Pair<String, Int> := {
    "a": 1
}

frozen: Constant Pair<String, Int> := scores

attempt {
    scores["b"] = 2
}
except ConstantError as error {
    write("insert rejected")
}

attempt {
    clear(scores)
}
except ConstantError as error {
    write("clear rejected")
}

write(len(scores))
`},
			expected: "insert rejected\nclear rejected\n1\n",
		},
		{
			name: "an equivalent unfrozen program still mutates",
			sources: map[string]string{"main.ahd": `source: List<Int> := [1, 2, 3]
alias: List<Int> := source
clear(alias)
write(len(source))

scores: Pair<String, Int> := {
    "a": 1
}

scores["b"] = 2
write(len(scores))
`},
			expected: "0\n2\n",
		},
		{
			name: "a Constant local binding freezes its graph",
			sources: map[string]string{"main.ahd": `check: Function := (
) -> Nothing {
    source: Local List<Int> := [1, 2]
    frozen: Local Constant List<Int> := source
    attempt {
        clear(source)
    }
    except ConstantError as error {
        write("local rejected")
    }

    write(len(frozen))
}

check()
`},
			expected: "local rejected\n2\n",
		},
	}
	runAcceptance(t, cases)
}

// TestExampleProgramsRunAsNativeExecutables keeps the shipped examples, and
// the manual stress test, working as real executables.
func TestExampleProgramsRunAsNativeExecutables(t *testing.T) {
	cases := []struct {
		name     string
		file     string
		expected string
	}{
		{name: "hello", file: "hello.ahd", expected: "Hello AhdCode\n"},
		{name: "clear", file: "clear.ahd", expected: "0\n0\n"},
		{name: "vector", file: "vector.ahd", expected: "23.0\n"},
		{name: "functions", file: "functions.ahd", expected: "12\n3628800\n"},
		{
			name: "arithmetic", file: "arithmetic.ahd",
			expected: "12\n-2\n35\n2\n0.7142857142857143\n25\n16807\n78125\n5\n",
		},
		{
			name: "errors", file: "errors.ahd",
			expected: "handled: too young\ncaught DivisionByZeroError\ncleanup\n42\nuncaught is reported by the runtime\n",
		},
		{
			name: "inheritance", file: "inheritance.ahd",
			expected: "Person: Ada\nStudent: Ada #7 (Person: Ada)\nStudent: Ada #7 (Person: Ada)\ntrue\ntrue\ntrue\n",
		},
		{
			name: "deep freeze", file: "deep_freeze.ahd",
			expected: "alias mutation rejected\nnested mutation rejected\n3\n2\n",
		},
		{
			name: "nullable function", file: "nullable_function.ahd",
			expected: "found Ada\nnothing found\n10\nno value\n",
		},
		{
			name: "class test", file: "class_test.ahd",
			expected: "Birinci dikdörtgen: 4.0 x 5.0\nİkinci dikdörtgen: 3.0 x 3.0\n" +
				"Birinci alan: 20.0\nBirinci çevre: 18.0\nİkinci alan: 9.0\nİkinci kare mi: true\n" +
				"Toplam alan: 29.0\nBüyük alan: 20.0\n--- Birinci nesne büyütülüyor ---\n" +
				"Birinci dikdörtgen: 8.0 x 10.0\nYeni alan: 80.0\nYeni çevre: 36.0\n" +
				"--- Reference semantics testi ---\nAlias: Birinci dikdörtgen: 4.0 x 5.0\n" +
				"Original: Birinci dikdörtgen: 4.0 x 5.0\nOriginal alan: 20.0\n",
		},
		{
			name: "stress test", file: "tester.ahd",
			expected: "=== CONSTANT / POWER ===\n78125\n16807\n" +
				"=== ARITHMETIC ===\n12\n-2\n35\n0.7142857142857143\n2\n25\n" +
				"=== RECURSION ===\n720\n" +
				"=== CALLBACK ===\n12\n" +
				"=== OVERLOAD ===\nInt: 25\nReal: 25.0\n" +
				"=== CLASS ===\nAna sayaç: 2\nAna sayaç: 7\nAna sayaç: 8\nAna sayaç: 8\n" +
				"=== CLASS + IF ===\nSayaç doğru: 8\n" +
				"=== STATE ===\nstate: eight\n" +
				"=== LIST ALIAS + CLEAR ===\n3\nsameNumbers: []\n0\n" +
				"=== UNTIL ===\nuntil: 1\nuntil: 2\nuntil: 3\n" +
				"=== STRING / UNICODE FOR ===\na\nñ\nb\n" +
				"=== INTERPOLATION ===\nAhdCode v0.1\nfactorial(5) = 120\ncounter = 8\n" +
				"=== DONE ===\n",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			entry, err := filepath.Abs(filepath.Join("..", "..", "examples", testCase.file))
			if err != nil {
				t.Fatalf("could not resolve the example path: %v", err)
			}
			out, errorOutput, code := buildAndRun(t, entry, "")
			if out != testCase.expected {
				t.Fatalf("stdout mismatch\n want %q\n have %q\n stderr: %s", testCase.expected, out, errorOutput)
			}
			if code != 0 {
				t.Fatalf("example exited with %d (stderr: %s)", code, errorOutput)
			}
		})
	}
}

// runAcceptance builds and runs every case as a real native executable.
func runAcceptance(t *testing.T, cases []program) {
	t.Helper()
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			entry := testCase.entry
			if entry == "" {
				entry = "main.ahd"
			}
			directory := writeSources(t, testCase.sources)
			out, errorOutput, code := buildAndRun(t, filepath.Join(directory, entry), testCase.stdin)
			if out != testCase.expected {
				t.Fatalf("stdout mismatch\n want %q\n have %q\n stderr: %s", testCase.expected, out, errorOutput)
			}
			if code != testCase.exitCode {
				t.Fatalf("exit code mismatch: want %d, have %d (stderr: %s)", testCase.exitCode, code, errorOutput)
			}
			if testCase.errorClass != "" && !strings.HasPrefix(errorOutput, testCase.errorClass+": ") {
				t.Fatalf("expected an uncaught %s on stderr; received %q", testCase.errorClass, errorOutput)
			}
			if strings.Contains(errorOutput, "goroutine ") {
				t.Fatalf("a Go stack trace leaked into program output: %q", errorOutput)
			}
		})
	}
}
