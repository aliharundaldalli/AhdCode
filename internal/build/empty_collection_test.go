package build

import (
	"path/filepath"
	"testing"
)

// TestEmptyCollectionProgramsRunAsNativeExecutables keeps contextually typed
// empty literals working through the whole compile chain.
func TestEmptyCollectionProgramsRunAsNativeExecutables(t *testing.T) {
	cases := []program{
		{
			name:     "typed empty List",
			sources:  map[string]string{"main.ahd": "numbers: List<Int> := []\nwrite(len(numbers))\nwrite(str(numbers))\n"},
			expected: "0\n[]\n",
		},
		{
			name:     "typed empty Pair",
			sources:  map[string]string{"main.ahd": "scores: Pair<String, Int> := {}\nwrite(len(scores))\nwrite(str(scores))\n"},
			expected: "0\n{}\n",
		},
		{
			name: "nested empty collections",
			sources: map[string]string{"main.ahd": `rows: List<List<Int>> := [
    []
    []
]

maps: List<Pair<String, Int>> := [
    {}
    {}
]

write(len(rows))
write(len(maps))
write(str(rows))
write(str(maps))
`},
			expected: "2\n2\n[[], []]\n[{}, {}]\n",
		},
		{
			name: "empty collections through call and return context",
			sources: map[string]string{"main.ahd": `useList: Function := (
    values: List<Int>
) -> Nothing {
    write(len(values))
}

makeList: Function := (
) -> List<Int> {
    return []
}

makePair: Function := (
) -> Pair<String, Int> {
    return {}
}

useList([])
write(len(makeList()))
write(len(makePair()))
`},
			expected: "0\n0\n0\n",
		},
		{
			name: "an empty literal still produces a usable collection",
			sources: map[string]string{"main.ahd": `numbers: List<Int> := []
alias: List<Int> := numbers
write(len(alias))

scores: Pair<String, Int> := {}
scores["a"] = 1
write(len(scores))
write(str(scores))
`},
			expected: "0\n1\n{\"a\": 1}\n",
		},
	}
	runAcceptance(t, cases)
}

// TestUninferableEmptyCollectionsAreRejected keeps an empty literal with no
// type context a compile-time error rather than a backend failure.
func TestUninferableEmptyCollectionsAreRejected(t *testing.T) {
	for name, source := range map[string]string{
		"bare empty List":          "write([])\n",
		"bare empty Pair":          "write({})\n",
		"untyped List declaration": "values: List := []\nwrite(len(values))\n",
		"untyped Pair declaration": "scores: Pair := {}\nwrite(len(scores))\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			directory := writeSources(t, map[string]string{"main.ahd": source})
			result := Compile(filepath.Join(directory, "main.ahd"))
			if !result.HasErrors() || result.Program != nil {
				t.Fatal("expected the program to be rejected before code generation")
			}
			for _, diagnostic := range result.Diagnostics {
				if diagnostic.Code[0] == 'I' || diagnostic.Code[0] == 'B' {
					t.Fatalf("an uninferable literal must fail in semantic analysis, not later: %s %s", diagnostic.Code, diagnostic.Message)
				}
			}
		})
	}
}
