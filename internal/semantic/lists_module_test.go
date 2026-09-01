package semantic

import "testing"

const listsPreamble = "bring Lists\nfrom Lists bring ListsError\n\n"

// listsClassPreamble adds a user Class, so the generic-preservation tests can
// prove a Class element type survives a structural transformation intact.
const listsClassPreamble = listsPreamble + `Student: Class<> := {
    structure: Attributes := (
        name: String
    )
}

`

// TestListsOperationsPreserveExactGenericTypes is the central type-safety
// proof: every result below is written with its full generic shape, so an
// erased element type or a silently widened one would fail to compile.
func TestListsOperationsPreserveExactGenericTypes(t *testing.T) {
	result := analyzeWithStandardModules(t, listsClassPreamble+`numbers: List<Int> := [1, 2, 3]
words: List<String> := ["a", "b"]
flags: List<Bool> := [true, false]
students: List<Student> := [Student(name: "Ali")]

intChunks: List<List<Int>> := Lists.chunk(numbers, 2)
stringChunks: List<List<String>> := Lists.chunk(words, 1)
boolChunks: List<List<Bool>> := Lists.chunk(flags, 1)
studentChunks: List<List<Student>> := Lists.chunk(students, 1)

grid: List<List<Int>> := [[1, 2], [3, 4]]
nestedChunks: List<List<List<Int>>> := Lists.chunk(grid, 1)
flattened: List<Int> := Lists.flatten(grid)
turned: List<List<Int>> := Lists.transpose(grid)

uniqueNumbers: List<Int> := Lists.unique(numbers)
uniqueStudents: List<Student> := Lists.unique(students)
uniqueRows: List<List<Int>> := Lists.unique(grid)

stringCounts: Pair<String, Int> := Lists.valueCounts(words)
intCounts: Pair<Int, Int> := Lists.valueCounts(numbers)
boolCounts: Pair<Bool, Int> := Lists.valueCounts(flags)

byInitial: Pair<String, List<String>> := Lists.groupBy(words, lambda (value: String) -> value[0])
byParity: Pair<Int, List<Int>> := Lists.groupBy(numbers, lambda (value: Int) -> value % 2)
byEven: Pair<Bool, List<Int>> := Lists.groupBy(numbers, lambda (value: Int) -> value % 2 == 0)
byName: Pair<String, List<Student>> := Lists.groupBy(students, lambda (value: Student) -> value.name)
`)
	requireSemanticClean(t, result)
}

// TestListsPreservesElementNullability proves structural nullability is
// carried through rather than erased or silently dropped.
func TestListsPreservesElementNullability(t *testing.T) {
	result := analyzeWithStandardModules(t, listsPreamble+`values: List<String?> := ["a", null]
chunks: List<List<String?>> := Lists.chunk(values, 2)
distinct: List<String?> := Lists.unique(values)

grid: List<List<Int?>> := [[1, null]]
flattened: List<Int?> := Lists.flatten(grid)
turned: List<List<Int?>> := Lists.transpose(grid)
`)
	requireSemanticClean(t, result)
}

// TestListsRejectsGenericWidening pins invariance: no result may be assigned
// to a wider or differently parameterized collection type.
func TestListsRejectsGenericWidening(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"chunk element does not widen Int to Real",
			"numbers: List<Int> := [1]\nbad: List<List<Real>> := Lists.chunk(numbers, 2)"},
		{"chunk does not flatten one nesting level",
			"numbers: List<Int> := [1]\nbad: List<Int> := Lists.chunk(numbers, 2)"},
		{"unique does not widen its element",
			"numbers: List<Int> := [1]\nbad: List<Real> := Lists.unique(numbers)"},
		{"flatten does not keep the outer nesting",
			"grid: List<List<Int>> := [[1]]\nbad: List<List<Int>> := Lists.flatten(grid)"},
		{"transpose does not drop a nesting level",
			"grid: List<List<Int>> := [[1]]\nbad: List<Int> := Lists.transpose(grid)"},
		{"valueCounts does not widen its key",
			"numbers: List<Int> := [1]\nbad: Pair<Real, Int> := Lists.valueCounts(numbers)"},
		{"groupBy does not widen its group element",
			"numbers: List<Int> := [1]\nbad: Pair<Int, List<Real>> := Lists.groupBy(numbers, lambda (value: Int) -> value)"},
		{"chunk does not erase element nullability",
			"values: List<String?> := [null]\nbad: List<List<String>> := Lists.chunk(values, 1)"},
		{"unique does not erase element nullability",
			"values: List<String?> := [null]\nbad: List<String> := Lists.unique(values)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requireSemanticFailure(t, analyzeWithStandardModules(t, listsPreamble+test.text+"\n"))
		})
	}
}

// TestListsValueCountsRestrictsPairKeys proves the existing Pair key rules are
// enforced rather than worked around by stringifying an unsupported key.
func TestListsValueCountsRestrictsPairKeys(t *testing.T) {
	tests := []string{
		"reals: List<Real> := [1.5]\nwrite(Lists.valueCounts(reals))",
		"nullable: List<String?> := [null]\nwrite(Lists.valueCounts(nullable))",
		"grid: List<List<Int>> := [[1]]\nwrite(Lists.valueCounts(grid))",
		"records: List<Pair<String, Int>> := [{\"a\": 1}]\nwrite(Lists.valueCounts(records))",
	}
	for _, text := range tests {
		t.Run(text, func(t *testing.T) {
			result := analyzeWithStandardModules(t, listsPreamble+text+"\n")
			requireSemanticCode(t, result, codeInvalidPairKey)
		})
	}
}

func TestListsValueCountsRejectsClassKeys(t *testing.T) {
	result := analyzeWithStandardModules(t, listsClassPreamble+`students: List<Student> := [Student(name: "Ali")]
write(Lists.valueCounts(students))
`)
	requireSemanticCode(t, result, codeInvalidPairKey)
}

// TestListsGroupByCallbackContract pins the key Function's typing: its
// parameter is exactly the element type and its result must be a non-null Pair
// key.
func TestListsGroupByCallbackContract(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"Int key accepted", "write(Lists.groupBy([1, 2], lambda (value: Int) -> value % 2))", true},
		{"Real key rejected", "write(Lists.groupBy([1, 2], lambda (value: Int) -> value * 1.5))", false},
		{"Nothing result rejected",
			"noop: Function := (value: Int) -> Nothing {\n    return\n}\nwrite(Lists.groupBy([1, 2], noop))", false},
		{"nullable key result rejected",
			"maybe: Function := (value: Int) -> String? {\n    return null\n}\nwrite(Lists.groupBy([1, 2], maybe))", false},
		{"mismatched parameter type rejected",
			"write(Lists.groupBy([1, 2], lambda (value: String) -> value))", false},
		{"nullable element with matching nullable parameter accepted",
			"values: List<String?> := [null]\nwrite(Lists.groupBy(values, lambda (value: String?) -> str(value)))", true},
		{"nullable element with a non-null parameter rejected",
			"values: List<String?> := [null]\nwrite(Lists.groupBy(values, lambda (value: String) -> value))", false},
		{"non-null element with a nullable parameter rejected",
			"write(Lists.groupBy([1, 2], lambda (value: Int?) -> str(value)))", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := analyzeWithStandardModules(t, listsPreamble+test.text+"\n")
			if test.ok {
				requireSemanticClean(t, result)
				return
			}
			requireSemanticFailure(t, result)
		})
	}
}

func TestListsRejectsWrongArityAndShapes(t *testing.T) {
	tests := []string{
		"write(Lists.chunk([1, 2]))",
		"write(Lists.chunk([1, 2], 2, 3))",
		"write(Lists.chunk(values: [1], size: 2))",
		"write(Lists.chunk([1, 2], \"two\"))",
		"write(Lists.chunk(\"not a List\", 2))",
		"write(Lists.flatten([1, 2, 3]))",
		"nested: List<List<Int>?> := [null]\nwrite(Lists.flatten(nested))",
		"write(Lists.transpose([1, 2, 3]))",
		"write(Lists.unique())",
		"write(Lists.valueCounts([1], 2))",
		"write(Lists.groupBy([1]))",
		"maybe: List<Int>? := null\nwrite(Lists.unique(maybe))",
	}
	for _, text := range tests {
		t.Run(text, func(t *testing.T) {
			requireSemanticFailure(t, analyzeWithStandardModules(t, listsPreamble+text+"\n"))
		})
	}
}

// TestListsOperationHasNoFunctionValue documents the one deliberate boundary:
// a type-directed operation is specialized at its call site and therefore has
// no unspecialized Function value.
func TestListsOperationHasNoFunctionValue(t *testing.T) {
	t.Run("namespace member", func(t *testing.T) {
		result := analyzeWithStandardModules(t, listsPreamble+"stored := Lists.chunk\n")
		requireSemanticCode(t, result, codeInvalidType)
	})
	t.Run("direct import", func(t *testing.T) {
		result := analyzeWithStandardModules(t, "bring Lists\nfrom Lists bring chunk\n\nstored := chunk\n")
		requireSemanticCode(t, result, codeInvalidType)
	})
}

// TestListsDirectImportCall proves the imported spelling reaches the same
// type-directed analysis as the qualified one.
func TestListsDirectImportCall(t *testing.T) {
	result := analyzeWithStandardModules(t, `bring Lists
from Lists bring chunk
from Lists bring valueCounts

parts: List<List<Int>> := chunk([1, 2, 3], 2)
counts: Pair<String, Int> := valueCounts(["a", "a"])
`)
	requireSemanticClean(t, result)
}

func TestListsErrorIsCatchable(t *testing.T) {
	result := analyzeWithStandardModules(t, listsPreamble+`attempt {
    write(Lists.chunk([1, 2], 0))
} except ListsError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}
