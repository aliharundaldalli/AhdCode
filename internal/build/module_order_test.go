package build

import (
	"path/filepath"
	"testing"
)

// TestModuleInitializationKeepsSourceOrder covers the v0.1 rule that executable
// module-root effects happen in source order. A module-root binding declares
// storage separately, but its initializer runs where it is written.
func TestModuleInitializationKeepsSourceOrder(t *testing.T) {
	cases := []program{
		{
			name: "a statement, an initializer, and a statement",
			sources: map[string]string{"main.ahd": `noisy: Function := (
    label: String
) -> Int {
    write("init {label}")
    return 1
}

write("statement A")
first: Int := noisy("first")
write("statement B")
`},
			expected: "statement A\ninit first\nstatement B\n",
		},
		{
			name: "interleaved initializers",
			sources: map[string]string{"main.ahd": `sideEffect: Function := (
    label: String
) -> Int {
    write(label)
    return 1
}

write("A")
x: Int := sideEffect("X")
write("B")
y: Int := sideEffect("Y")
write("C")
write(x + y)
`},
			expected: "A\nX\nB\nY\nC\n2\n",
		},
		{
			name: "Class construction interleaves with writes",
			sources: map[string]string{"main.ahd": `Label: Class<> := {
    structure: Attributes := (
        text: String
    )
}

announce: Function := (
    value: Label
) -> Label {
    write("constructed {value.text}")
    return value
}

write("A")
one: Label := announce(Label(text: "one"))
write("B")
two: Label := announce(Label(text: "two"))
write("C")
write("{one.text}{two.text}")
`},
			expected: "A\nconstructed one\nB\nconstructed two\nC\nonetwo\n",
		},
		{
			name: "an initializer that raises stops later module statements",
			sources: map[string]string{"main.ahd": `boom: Function := (
) -> Int {
    toss Error(message: "initializer failed")
    return 1
}

write("A")
value: Int := boom()
write("B must not run")
`},
			expected:   "A\n",
			exitCode:   1,
			errorClass: "Error",
		},
		{
			name: "an initializer error is catchable in source position",
			sources: map[string]string{"main.ahd": `boom: Function := (
) -> Int {
    toss Error(message: "boom")
    return 1
}

write("A")
attempt {
    value: Local Int := boom()
    write("unreachable")
}
except Error as error {
    write("caught {error.message}")
}

write("B")
`},
			expected: "A\ncaught boom\nB\n",
		},
		{
			name: "side-effect-free globals still initialize",
			sources: map[string]string{"main.ahd": `first: Int := 1
second: Int := first + 1
third: String := "value {second}"
write(first)
write(second)
write(third)
`},
			expected: "1\n2\nvalue 2\n",
		},
		{
			name: "Global reads and writes from Functions stay correct",
			sources: map[string]string{"main.ahd": `counter: Int := 0

bump: Function := (
) -> Int {
    counter: Global Int
    counter = counter + 1
    return counter
}

write("start {counter}")
first: Int := bump()
write("after first {counter}")
second: Int := bump()
write("{first} {second} {counter}")
`},
			expected: "start 0\nafter first 1\n1 2 2\n",
		},
		{
			name: "a Constant global still freezes its graph at its own position",
			sources: map[string]string{"main.ahd": `source: List<Int> := [1, 2, 3]
write("A")
frozen: Constant List<Int> := source
write("B")

attempt {
    clear(source)
}
except ConstantError as error {
    write("rejected")
}

write(len(frozen))
`},
			expected: "A\nB\nrejected\n3\n",
		},
		{
			name: "a dependency initializes fully before the entry module",
			sources: map[string]string{
				"Dep.ahd": `write("dep A")
value: Int := 1
write("dep B {value}")

provide: Function := (
) -> Int {
    return 42
}
`,
				"main.ahd": `from Dep bring provide

write("main A")
answer: Int := provide()
write("main B {answer}")
`,
			},
			expected: "dep A\ndep B 1\nmain A\nmain B 42\n",
		},
		{
			name:     "take in a module-root initializer keeps its source position",
			sources:  map[string]string{"main.ahd": "write(\"before\")\nname: String := take(\"Name: \")\nwrite(\"after {name}\")\n"},
			stdin:    "Ali\n",
			expected: "before\nName: after Ali\n",
		},
		{
			name: "consecutive module-root reads consume consecutive lines",
			sources: map[string]string{"main.ahd": `first: String := take()
write("[{first}]")
second: String := take()
write("[{second}]")
`},
			stdin:    "one\ntwo\n",
			expected: "[one]\n[two]\n",
		},
	}
	runAcceptance(t, cases)
}

// TestModuleInitializationIsDeterministic keeps repeated compilation of an
// interleaved module byte-identical.
func TestModuleInitializationIsDeterministic(t *testing.T) {
	sources := map[string]string{
		"Dep.ahd": "write(\"dep\")\nvalue: Int := 1\n",
		"main.ahd": `from Dep bring value

write("A")
first: Int := 1
write("B")
second: Int := first + 1
write(second)
`,
	}
	directory := writeSources(t, sources)
	first := Compile(filepath.Join(directory, "main.ahd"))
	second := Compile(filepath.Join(directory, "main.ahd"))
	if first.HasErrors() || second.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(first.Diagnostics))
	}
	for index := range first.Program.Files {
		if first.Program.Files[index].Content != second.Program.Files[index].Content {
			t.Fatalf("module initialization is not deterministic for %s", first.Program.Files[index].Name)
		}
	}
}
