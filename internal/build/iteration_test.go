package build

import (
	"path/filepath"
	"testing"
)

// TestIterationProgramsRunAsNativeExecutables covers typed for bindings and the
// lazy between iteration end to end.
func TestIterationProgramsRunAsNativeExecutables(t *testing.T) {
	cases := []program{
		{
			name: "typed and inferred bindings over every iterable",
			sources: map[string]string{"main.ahd": `values: List<Int> := [1, 2]
scores: Pair<String, Int> := {
    "a": 1
}

for inferred in values {
    write(inferred)
}

for typed: Int in values {
    write(typed)
}

for character: String in "añb" {
    write(character)
}

for key: String in scores {
    write(key)
}
`},
			expected: "1\n2\n1\n2\na\nñ\nb\na\n",
		},
		{
			name: "between forms and defaults",
			sources: map[string]string{"main.ahd": `for a in between(5) {
    write(a)
}

for b in between(1, 5) {
    write(b)
}

for c in between(0, 10, 2) {
    write(c)
}
`},
			expected: "0\n1\n2\n3\n4\n1\n2\n3\n4\n0\n2\n4\n6\n8\n",
		},
		{
			name: "a negative step counts down and an unreachable stop is empty",
			sources: map[string]string{"main.ahd": `for a in between(5, 0, -1) {
    write(a)
}

write("--")

for b in between(0, 5, -1) {
    write(b)
}

for c in between(5, 0, 1) {
    write(c)
}

write("empty")
`},
			expected: "5\n4\n3\n2\n1\n--\nempty\n",
		},
		{
			name: "a zero step raises a catchable DomainError",
			sources: map[string]string{"main.ahd": `attempt {
    for value in between(0, 10, 0) {
        write(value)
    }
}
except DomainError as error {
    write("zero step rejected")
}
`},
			expected: "zero step rejected\n",
		},
		{
			name: "continue and break work over a lazy range",
			sources: map[string]string{"main.ahd": `for value: Int in between(1, 100) {
    if value == 3 {
        continue
    }

    if value == 6 {
        break
    }

    write(value)
}
`},
			expected: "1\n2\n4\n5\n",
		},
		{
			name: "a very large range is not materialized",
			sources: map[string]string{"main.ahd": `for value in between(1000000000) {
    break
}

write("immediate")

seen: Int := 0
for value in between(1, 100000000) {
    seen++
    if seen == 3 {
        break
    }
}

write(seen)
`},
			expected: "immediate\n3\n",
		},
		{
			name: "a range near the Int boundaries terminates",
			sources: map[string]string{"main.ahd": `maximum: Int := 9223372036854775807
minimum: Int := -9223372036854775808

for a in between(maximum - 2, maximum) {
    write(a)
}

for b in between(maximum - 1, maximum, 2) {
    write(b)
}

for c in between(minimum + 1, minimum, -2) {
    write(c)
}

for d in between(minimum, minimum + 2) {
    write(d)
}
`},
			expected: "9223372036854775805\n9223372036854775806\n9223372036854775806\n" +
				"-9223372036854775807\n-9223372036854775808\n-9223372036854775807\n",
		},
		{
			name: "collection iteration still snapshots",
			sources: map[string]string{"main.ahd": `values: List<Int> := [1, 2, 3]
seen: Int := 0
for value: Int in values {
    seen++
    clear(values)
}

write(seen)
write(len(values))
`},
			expected: "3\n0\n",
		},
	}
	runAcceptance(t, cases)
}

// TestIterationBindingProgramsAreRejected keeps a wrong explicit binding type a
// compile-time error with no backend fallout.
func TestIterationBindingProgramsAreRejected(t *testing.T) {
	cases := map[string]string{
		"wrong List element":        "values: List<String> := [\"A\"]\nfor value: Int in values {\n}\n",
		"wrong between value":       "for value: String in between(10) {\n}\n",
		"Real between bound":        "for value in between(1.5, 10) {\n}\n",
		"between with no arguments": "for value in between() {\n}\n",
		"scope modifier":            "for value: Local Int in between(3) {\n}\n",
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
				if diagnostic.Code[0] == 'I' || diagnostic.Code[0] == 'B' {
					t.Fatalf("a semantic root cause was followed by %s: %s", diagnostic.Code, diagnostic.Message)
				}
			}
		})
	}
}
