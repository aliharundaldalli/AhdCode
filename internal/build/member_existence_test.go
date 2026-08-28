package build

import (
	"path/filepath"
	"testing"
)

// personStudent is the inheritance pair the runtime member tests share.
const personStudent = `Person: Class<> := {
    structure: Attributes := (
        name: String
    )

    describe: Function := (
    ) -> String {
        return attribute.name
    }
}

Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )

    study: Function := (
    ) -> Nothing {
        return
    }
}

`

// TestHasReadsTheRuntimeClass covers the defining property of has: the answer
// comes from the object's exact runtime Class and its inheritance chain, not
// from the static type of the expression it is written against.
func TestHasReadsTheRuntimeClass(t *testing.T) {
	cases := []program{
		{
			name:     "an own field is present",
			sources:  map[string]string{"main.ahd": personStudent + "student: Student := Student(name: \"Ali\", number: 42)\nwrite(student has number)\n"},
			expected: "true\n",
		},
		{
			name:     "an inherited field is present",
			sources:  map[string]string{"main.ahd": personStudent + "student: Student := Student(name: \"Ali\", number: 42)\nwrite(student has name)\n"},
			expected: "true\n",
		},
		{
			name:     "a missing member is absent",
			sources:  map[string]string{"main.ahd": personStudent + "student: Student := Student(name: \"Ali\", number: 42)\nwrite(student has nickname)\n"},
			expected: "false\n",
		},
		{
			name:     "an own method is present",
			sources:  map[string]string{"main.ahd": personStudent + "student: Student := Student(name: \"Ali\", number: 42)\nwrite(student has study)\n"},
			expected: "true\n",
		},
		{
			name:     "an inherited method is present",
			sources:  map[string]string{"main.ahd": personStudent + "student: Student := Student(name: \"Ali\", number: 42)\nwrite(student has describe)\n"},
			expected: "true\n",
		},
		{
			name:     "an overridden method is present on both Classes",
			sources:  map[string]string{"main.ahd": "Animal: Class<> := {\n    speak: Function := (\n    ) -> String {\n        return \"...\"\n    }\n}\n\nDog: Class<Animal> := {\n    speak: Override Function := (\n    ) -> String {\n        return \"woof\"\n    }\n}\n\nanimal: Animal := Animal()\ndog: Dog := Dog()\nupcast: Animal := Dog()\nwrite(animal has speak)\nwrite(dog has speak)\nwrite(upcast has speak)\n"},
			expected: "true\ntrue\ntrue\n",
		},
		{
			name:     "has not is the exact negation of has",
			sources:  map[string]string{"main.ahd": personStudent + "student: Student := Student(name: \"Ali\", number: 42)\nwrite(student has not nickname)\nwrite(student has not name)\nwrite(student has not number)\nwrite(student has not study)\n"},
			expected: "true\nfalse\nfalse\nfalse\n",
		},
		{
			name:     "a parent-typed child still reports its own field",
			sources:  map[string]string{"main.ahd": personStudent + "person: Person := Student(name: \"Ali\", number: 42)\nwrite(person has number)\n"},
			expected: "true\n",
		},
		{
			name:     "a parent-typed child still reports its own method",
			sources:  map[string]string{"main.ahd": personStudent + "person: Person := Student(name: \"Ali\", number: 42)\nwrite(person has study)\n"},
			expected: "true\n",
		},
		{
			name:     "a parent-typed child still reports inherited members",
			sources:  map[string]string{"main.ahd": personStudent + "person: Person := Student(name: \"Ali\", number: 42)\nwrite(person has name)\nwrite(person has describe)\nwrite(person has nickname)\n"},
			expected: "true\ntrue\nfalse\n",
		},
		{
			name:     "an ordinary parent instance gains no child member",
			sources:  map[string]string{"main.ahd": personStudent + "person: Person := Person(name: \"Ali\")\nwrite(person has name)\nwrite(person has describe)\nwrite(person has number)\nwrite(person has study)\n"},
			expected: "true\ntrue\nfalse\nfalse\n",
		},
		{
			name:     "the left expression is evaluated exactly once",
			sources:  map[string]string{"main.ahd": personStudent + "calls: Int := 0\n\nmakeStudent: Function := (\n) -> Student {\n    calls: Global Int\n    calls = calls + 1\n    return Student(name: \"Ali\", number: 42)\n}\n\nwrite(makeStudent() has number)\nwrite(makeStudent() has not nickname)\nwrite(calls)\n"},
			expected: "true\ntrue\n2\n",
		},
		{
			name:     "the member designator executes nothing",
			sources:  map[string]string{"main.ahd": personStudent + "number: Int := 7\nstudent: Student := Student(name: \"Ali\", number: 42)\nwrite(student has number)\nwrite(number)\n"},
			expected: "true\n7\n",
		},
		{
			name:     "a Class three levels deep sees every ancestor",
			sources:  map[string]string{"main.ahd": "A: Class<> := {\n    structure: Attributes := (\n        a: Int\n    )\n}\n\nB: Class<A> := {\n    structure: Attributes := (\n        SuperClass.attributes\n        b: Int\n    )\n}\n\nC: Class<B> := {\n    structure: Attributes := (\n        SuperClass.attributes\n        c: Int\n    )\n}\n\nvalue: A := C(a: 1, b: 2, c: 3)\nwrite(value has a)\nwrite(value has b)\nwrite(value has c)\nwrite(value has d)\n"},
			expected: "true\ntrue\ntrue\nfalse\n",
		},
		{
			name:     "a built-in Error still publishes message",
			sources:  map[string]string{"main.ahd": "attempt {\n    toss(ValueError(\"boom\"))\n} except Error as error {\n    write(error has message)\n    write(error has missing)\n}\n"},
			expected: "true\nfalse\n",
		},
	}
	runProgramCases(t, cases)
}

// TestIsRemainsUnchangedAlongsideHas records that runtime-aware has did not
// alter Class membership testing.
func TestIsRemainsUnchangedAlongsideHas(t *testing.T) {
	cases := []program{
		{
			name:     "is still follows the runtime Class and its ancestry",
			sources:  map[string]string{"main.ahd": personStudent + "student: Student := Student(name: \"Ali\", number: 42)\nperson: Person := Student(name: \"Ali\", number: 42)\nplain: Person := Person(name: \"Ali\")\nwrite(student is Student)\nwrite(student is Person)\nwrite(person is Student)\nwrite(person is Person)\nwrite(plain is Student)\nwrite(plain is not Student)\n"},
			expected: "true\ntrue\ntrue\ntrue\nfalse\ntrue\n",
		},
	}
	runProgramCases(t, cases)
}

// TestHasWorksAcrossModuleBoundaries checks that a Class whose parent is
// imported from another module keeps one canonical runtime descriptor chain.
func TestHasWorksAcrossModuleBoundaries(t *testing.T) {
	sources := map[string]string{
		"Shapes.ahd": `Shape: Class<> := {
    structure: Attributes := (
        label: String
    )

    describe: Function := (
    ) -> String {
        return attribute.label
    }
}
`,
		"main.ahd": `from Shapes bring Shape

Circle: Class<Shape> := {
    structure: Attributes := (
        SuperClass.attributes
        radius: Real
    )

    area: Function := (
    ) -> Real {
        return attribute.radius * attribute.radius
    }
}

shape: Shape := Circle(label: "c", radius: 2.0)
write(shape has label)
write(shape has describe)
write(shape has radius)
write(shape has area)
write(shape has missing)
`,
	}
	directory := writeSources(t, sources)
	out, errorOutput, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	expected := "true\ntrue\ntrue\ntrue\nfalse\n"
	if out != expected || code != 0 {
		t.Fatalf("cross-module has output\n want %q\n have %q (exit %d, stderr %s)", expected, out, code, errorOutput)
	}
}

// TestHasReportsConfidentialMemberExistence records the behavior AhdCode has
// always had: has answers whether a member exists, and a Confidential member
// exists. Whether existence itself should be hidden is not settled by the
// current normative text, so this test pins the status quo rather than a new
// visibility rule.
func TestHasReportsConfidentialMemberExistence(t *testing.T) {
	cases := []program{
		{
			name:     "a Confidential attribute and method still exist",
			sources:  map[string]string{"main.ahd": "Account: Class<> := {\n    structure: Attributes := (\n        name: String\n        secret: Confidential String\n    )\n\n    check: Confidential Function := (\n    ) -> Bool {\n        return true\n    }\n}\n\naccount: Account := Account(name: \"Ali\", secret: \"s\")\nwrite(account has name)\nwrite(account has secret)\nwrite(account has check)\n"},
			expected: "true\ntrue\ntrue\n",
		},
	}
	runProgramCases(t, cases)
}
