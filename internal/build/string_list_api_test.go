package build

import (
	"path/filepath"
	"strings"
	"testing"
)

// listCallbacks is the shared preamble of the callback-typed List operations.
// v0.1 has no lambda syntax, so every callback is a declared Function.
const listCallbacks = `double: Function := (
    x: Int
) -> Int {
    return x * 2
}

describe: Function := (
    x: Int
) -> String {
    return "Sayi: {x}"
}

isEven: Function := (
    x: Int
) -> Bool {
    return x % 2 == 0
}

`

func TestStringOperationsRunAsNativeExecutables(t *testing.T) {
	cases := []program{
		{
			name:     "trim removes ASCII whitespace at both ends only",
			sources:  map[string]string{"main.ahd": "write(\"[{\"  Ali  \".trim()}]\")\nwrite(\"[{\"  Ali Harun  \".trim()}]\")\n"},
			expected: "[Ali]\n[Ali Harun]\n",
		},
		{
			name:     "trim removes tabs, newlines, and Unicode spaces",
			sources:  map[string]string{"main.ahd": "write(\"[{\"\\t Ali Harun \\n\".trim()}]\")\nwrite(\"[{\"\u00a0Ali\u3000\".trim()}]\")\nwrite(\"[{\"\".trim()}]\")\n"},
			expected: "[Ali Harun]\n[Ali]\n[]\n",
		},
		{
			name:     "lower is locale-independent Unicode",
			sources:  map[string]string{"main.ahd": "write(\"AhdCode\".lower())\nwrite(\"ÇĞİÖŞÜ\".lower())\n"},
			expected: "ahdcode\nçğiöşü\n",
		},
		{
			name:     "upper is locale-independent Unicode",
			sources:  map[string]string{"main.ahd": "write(\"AhdCode\".upper())\nwrite(\"çğıöşü\".upper())\n"},
			expected: "AHDCODE\nÇĞIÖŞÜ\n",
		},
		{
			name:     "capitalize uppercases only the first character",
			sources:  map[string]string{"main.ahd": "write(\"ali HARUN\".capitalize())\nwrite(\"aHD\".capitalize())\nwrite(\"[{\"\".capitalize()}]\")\nwrite(\"ünlü kişi\".capitalize())\n"},
			expected: "Ali HARUN\nAHD\n[]\nÜnlü kişi\n",
		},
		{
			name:     "capitalize composes with lower for full normalization",
			sources:  map[string]string{"main.ahd": "write(\"ali HARUN\".lower().capitalize())\n"},
			expected: "Ali harun\n",
		},
		{
			name:     "split divides on every occurrence",
			sources:  map[string]string{"main.ahd": "write(\"a,b,c\".split(\",\"))\nwrite(\"a--b\".split(\"--\"))\n"},
			expected: "[\"a\", \"b\", \"c\"]\n[\"a\", \"b\"]\n",
		},
		{
			name:     "split preserves empty fields",
			sources:  map[string]string{"main.ahd": "write(\"a,,b,\".split(\",\"))\nwrite(\"\".split(\",\"))\nwrite(len(\"a,,b,\".split(\",\")))\n"},
			expected: "[\"a\", \"\", \"b\", \"\"]\n[\"\"]\n4\n",
		},
		{
			name:       "an empty separator is a DomainError",
			sources:    map[string]string{"main.ahd": "write(\"abc\".split(\"\"))\n"},
			exitCode:   1,
			errorClass: "DomainError",
		},
		{
			name:     "replace rewrites every non-overlapping occurrence",
			sources:  map[string]string{"main.ahd": "write(\"banana\".replace(\"na\", \"X\"))\nwrite(\"abc\".replace(\"b\", \"\"))\nwrite(\"aaa\".replace(\"aa\", \"b\"))\n"},
			expected: "baXX\nac\nba\n",
		},
		{
			name:     "replace leaves the receiver untouched",
			sources:  map[string]string{"main.ahd": "text: String := \"banana\"\nwrite(text.replace(\"na\", \"X\"))\nwrite(text)\n"},
			expected: "baXX\nbanana\n",
		},
		{
			name:       "an empty search text is a DomainError for replace",
			sources:    map[string]string{"main.ahd": "write(\"abc\".replace(\"\", \"x\"))\n"},
			exitCode:   1,
			errorClass: "DomainError",
		},
		{
			name:     "contains, startsWith, and endsWith follow String mathematics",
			sources:  map[string]string{"main.ahd": "write(\"abc\".contains(\"\"))\nwrite(\"abc\".contains(\"bc\"))\nwrite(\"abc\".contains(\"x\"))\nwrite(\"abc\".startsWith(\"\"))\nwrite(\"abc\".startsWith(\"ab\"))\nwrite(\"abc\".endsWith(\"\"))\nwrite(\"abc\".endsWith(\"bc\"))\nwrite(\"abc\".endsWith(\"ab\"))\n"},
			expected: "true\ntrue\nfalse\ntrue\ntrue\ntrue\ntrue\nfalse\n",
		},
		{
			name:     "count counts non-overlapping occurrences",
			sources:  map[string]string{"main.ahd": "write(\"banana\".count(\"a\"))\nwrite(\"banana\".count(\"na\"))\nwrite(\"banana\".count(\"x\"))\nwrite(\"aaaa\".count(\"aa\"))\n"},
			expected: "3\n2\n0\n2\n",
		},
		{
			name:       "an empty needle has no count",
			sources:    map[string]string{"main.ahd": "write(\"abc\".count(\"\"))\n"},
			exitCode:   1,
			errorClass: "DomainError",
		},
		{
			name:     "index is a character index, not a byte offset",
			sources:  map[string]string{"main.ahd": "write(\"banana\".index(\"na\"))\nwrite(\"a✓b✓\".index(\"✓\"))\nwrite(\"a✓b✓\".index(\"b\"))\nwrite(\"çğü\".index(\"ü\"))\n"},
			expected: "2\n1\n2\n2\n",
		},
		{
			name:       "a missing search text is a DomainError rather than -1",
			sources:    map[string]string{"main.ahd": "write(\"banana\".index(\"xyz\"))\n"},
			exitCode:   1,
			errorClass: "DomainError",
		},
		{
			name:       "an empty needle has no index",
			sources:    map[string]string{"main.ahd": "write(\"abc\".index(\"\"))\n"},
			exitCode:   1,
			errorClass: "DomainError",
		},
		{
			name:     "String operation errors are catchable",
			sources:  map[string]string{"main.ahd": "attempt {\n    write(\"abc\".index(\"z\"))\n} except DomainError as error {\n    write(error.message)\n}\n"},
			expected: "index did not find the search text\n",
		},
		{
			name:     "operations compose without mutating the receiver",
			sources:  map[string]string{"main.ahd": "text: String := \"  Ali,Veli  \"\nwrite(text.trim().lower().split(\",\"))\nwrite(\"[{text}]\")\n"},
			expected: "[\"ali\", \"veli\"]\n[  Ali,Veli  ]\n",
		},
	}
	runProgramCases(t, cases)
}

func TestListOperationsRunAsNativeExecutables(t *testing.T) {
	cases := []program{
		{
			name:     "reverse mutates in place and every alias observes it",
			sources:  map[string]string{"main.ahd": "values: List<Int> := [1, 2, 3]\nalias: List<Int> := values\nvalues.reverse()\nwrite(values)\nwrite(alias)\nwrite(alias same values)\n"},
			expected: "[3, 2, 1]\n[3, 2, 1]\ntrue\n",
		},
		{
			name:     "reverse of an empty or single-element List is stable",
			sources:  map[string]string{"main.ahd": "empty: List<Int> := []\nsingle: List<Int> := [7]\nempty.reverse()\nsingle.reverse()\nwrite(empty)\nwrite(single)\n"},
			expected: "[]\n[7]\n",
		},
		{
			name:     "natural sort orders Int, Real, and String ascending in place",
			sources:  map[string]string{"main.ahd": "ints: List<Int> := [8, 3, 12, 5]\nreals: List<Real> := [3.5, -2.0, 8.25]\nwords: List<String> := [\"pear\", \"apple\", \"fig\"]\nalias: List<Int> := ints\nints.sort()\nreals.sort()\nwords.sort()\nwrite(ints)\nwrite(reals)\nwrite(words)\nwrite(alias)\n"},
			expected: "[3, 5, 8, 12]\n[-2.0, 3.5, 8.25]\n[\"apple\", \"fig\", \"pear\"]\n[3, 5, 8, 12]\n",
		},
		{
			name:       "a null element has no natural order",
			sources:    map[string]string{"main.ahd": "values: List<Int> := [3, null, 1]\nvalues.sort()\n"},
			exitCode:   1,
			errorClass: "NullError",
		},
		{
			name:     "a rejected natural sort leaves the List unchanged",
			sources:  map[string]string{"main.ahd": "values: List<Int> := [3, null, 1]\nattempt {\n    values.sort()\n} except NullError as error {\n    write(\"caught\")\n}\nwrite(len(values))\n"},
			expected: "caught\n3\n",
		},
		{
			name:     "count uses deep == rather than identity",
			sources:  map[string]string{"main.ahd": "values: List<Int> := [5, 7, 5, 9]\nwrite(values.count(5))\nwrite(values.count(1))\nrows: List<List<Int>> := [\n    [1, 2],\n    [3],\n    [1, 2]\n]\nwrite(rows.count([1, 2]))\ntexts: List<String> := [\"a\", \"b\", \"a\"]\nwrite(texts.count(\"a\"))\n"},
			expected: "2\n0\n2\n2\n",
		},
		{
			name:     "index reports the first match",
			sources:  map[string]string{"main.ahd": "values: List<Int> := [5, 7, 5]\nwrite(values.index(5))\nwrite(values.index(7))\n"},
			expected: "0\n1\n",
		},
		{
			name:       "a missing value is a DomainError rather than -1",
			sources:    map[string]string{"main.ahd": "values: List<Int> := [5, 7]\nwrite(values.index(999))\n"},
			exitCode:   1,
			errorClass: "DomainError",
		},
		{
			name:     "count and index leave the List unchanged",
			sources:  map[string]string{"main.ahd": "values: List<Int> := [5, 7, 5]\nwrite(values.count(5))\nwrite(values.index(7))\nwrite(values)\nwrite(len(values))\n"},
			expected: "2\n1\n[5, 7, 5]\n3\n",
		},
		{
			name:     "map builds a new List of the callback result type",
			sources:  map[string]string{"main.ahd": listCallbacks + "numbers: List<Int> := [1, 2, 3]\ndoubled: List<Int> := numbers.map(double)\ntexts: List<String> := numbers.map(describe)\nwrite(doubled)\nwrite(texts)\nwrite(numbers)\nwrite(doubled same numbers)\n"},
			expected: "[2, 4, 6]\n[\"Sayi: 1\", \"Sayi: 2\", \"Sayi: 3\"]\n[1, 2, 3]\nfalse\n",
		},
		{
			name:     "a mapped List is mutable",
			sources:  map[string]string{"main.ahd": listCallbacks + "numbers: List<Int> := [1]\ndoubled: List<Int> := numbers.map(double)\ndoubled.add(9)\nwrite(doubled)\n"},
			expected: "[2, 9]\n",
		},
		{
			name:     "filter keeps the predicate's elements in order",
			sources:  map[string]string{"main.ahd": listCallbacks + "values: List<Int> := [1, 2, 3, 4]\nevens: List<Int> := values.filter(isEven)\nwrite(evens)\nwrite(values)\n"},
			expected: "[2, 4]\n[1, 2, 3, 4]\n",
		},
		{
			name:     "map and filter read a Constant List",
			sources:  map[string]string{"main.ahd": listCallbacks + "values: Constant List<Int> := [1, 2, 3, 4]\nwrite(values.map(double))\nwrite(values.filter(isEven))\nwrite(values.count(2))\nwrite(values.index(2))\nwrite(values)\n"},
			expected: "[2, 4, 6, 8]\n[2, 4]\n1\n1\n[1, 2, 3, 4]\n",
		},
		{
			name:     "map and filter iterate a shallow snapshot",
			sources:  map[string]string{"main.ahd": "values: List<Int> := [1, 2, 3]\n\ngrow: Function := (\n    x: Int\n) -> Int {\n    values: Global List<Int>\n    values.add(99)\n    return x\n}\n\nkeep: Function := (\n    x: Int\n) -> Bool {\n    values: Global List<Int>\n    values.add(50)\n    return true\n}\n\nwrite(values.map(grow))\nwrite(len(values))\nwrite(values.filter(keep))\nwrite(len(values))\n"},
			expected: "[1, 2, 3]\n6\n[1, 2, 3, 99, 99, 99]\n12\n",
		},
		{
			name:       "a callback error propagates out of map",
			sources:    map[string]string{"main.ahd": "fail: Function := (\n    x: Int\n) -> Int {\n    if x == 2 {\n        toss(ValueError(\"bad element\"))\n    }\n\n    return x\n}\n\nvalues: List<Int> := [1, 2, 3]\nwrite(values.map(fail))\n"},
			exitCode:   1,
			errorClass: "ValueError",
		},
		{
			name:     "a callback error is catchable and leaves the source unchanged",
			sources:  map[string]string{"main.ahd": "fail: Function := (\n    x: Int\n) -> Bool {\n    if x == 2 {\n        toss(ValueError(\"bad element\"))\n    }\n\n    return true\n}\n\nvalues: List<Int> := [1, 2, 3]\nattempt {\n    write(values.filter(fail))\n} except ValueError as error {\n    write(\"caught {error.message}\")\n}\nwrite(values)\n"},
			expected: "caught bad element\n[1, 2, 3]\n",
		},
		{
			name:     "a null element reaches a NonNull callback parameter as a NullError",
			sources:  map[string]string{"main.ahd": listCallbacks + "values: List<Int> := [1, null]\nattempt {\n    write(values.map(double))\n} except NullError as error {\n    write(\"caught\")\n}\n"},
			expected: "caught\n",
		},
		{
			name:     "a refined nullable List receiver is accepted",
			sources:  map[string]string{"main.ahd": "values: List<Int> := null\nvalues = [3, 1]\nif values != null {\n    values.sort()\n    write(values)\n    write(values.count(1))\n}\n"},
			expected: "[1, 3]\n1\n",
		},
	}
	runProgramCases(t, cases)
}

func TestKeyedSortIsStableAtomicAndEvaluatesEachKeyOnce(t *testing.T) {
	students := `Student: Class<> := {
    structure: Attributes := (
        name: String
        grade: Int
    )
}

gradeOf: Function := (
    student: Student
) -> Int {
    return student.grade
}

nameOf: Function := (
    student: Student
) -> String {
    return student.name
}

`
	cases := []program{
		{
			name:     "a keyed sort is stable and ascending",
			sources:  map[string]string{"main.ahd": students + "students: List<Student> := [\n    Student(\"Zeynep\", 70),\n    Student(\"Ali\", 90),\n    Student(\"Mehmet\", 70)\n]\nstudents.sort(gradeOf)\nfor s: Student in students {\n    write(\"{s.name} {s.grade}\")\n}\n"},
			expected: "Zeynep 70\nMehmet 70\nAli 90\n",
		},
		{
			name:     "a String key orders by text",
			sources:  map[string]string{"main.ahd": students + "students: List<Student> := [\n    Student(\"Zeynep\", 70),\n    Student(\"Ali\", 90),\n    Student(\"Mehmet\", 70)\n]\nstudents.sort(nameOf)\nfor s: Student in students {\n    write(s.name)\n}\n"},
			expected: "Ali\nMehmet\nZeynep\n",
		},
		{
			name:     "a Real key orders numerically",
			sources:  map[string]string{"main.ahd": "weightOf: Function := (\n    x: Int\n) -> Real {\n    return real(x) * -1.5\n}\n\nvalues: List<Int> := [1, 3, 2]\nvalues.sort(weightOf)\nwrite(values)\n"},
			expected: "[3, 2, 1]\n",
		},
		{
			name:     "each key is computed exactly once, left to right",
			sources:  map[string]string{"main.ahd": "trace: List<Int> := []\n\nkeyOf: Function := (\n    x: Int\n) -> Int {\n    trace: Global List<Int>\n    trace.add(x)\n    return -x\n}\n\nvalues: List<Int> := [3, 1, 2]\nvalues.sort(keyOf)\nwrite(values)\nwrite(trace)\nwrite(len(trace))\n"},
			expected: "[3, 2, 1]\n[3, 1, 2]\n3\n",
		},
		{
			name:     "a raising key Function leaves the original order unchanged",
			sources:  map[string]string{"main.ahd": "failing: Function := (\n    x: Int\n) -> Int {\n    if x == 1 {\n        toss(DomainError(\"bad key\"))\n    }\n\n    return x\n}\n\nvalues: List<Int> := [3, 1, 2]\nattempt {\n    values.sort(failing)\n} except DomainError as error {\n    write(\"caught {error.message}\")\n}\nwrite(values)\n"},
			expected: "caught bad key\n[3, 1, 2]\n",
		},
		{
			name:     "a Constant List rejects a keyed sort before any key runs",
			sources:  map[string]string{"main.ahd": "trace: List<Int> := []\n\nkeyOf: Function := (\n    x: Int\n) -> Int {\n    trace: Global List<Int>\n    trace.add(x)\n    return x\n}\n\nvalues: Constant List<Int> := [3, 1]\nfrozen: List<Int> := values\nattempt {\n    frozen.sort(keyOf)\n} except ConstantError as error {\n    write(error.message)\n}\nwrite(trace)\nwrite(frozen)\n"},
			expected: "cannot mutate a Constant object\n[]\n[3, 1]\n",
		},
		{
			name:       "a Constant List rejects reverse through an alias",
			sources:    map[string]string{"main.ahd": "values: Constant List<Int> := [3, 1]\nfrozen: List<Int> := values\nfrozen.reverse()\n"},
			exitCode:   1,
			errorClass: "ConstantError",
		},
	}
	runProgramCases(t, cases)
}

// TestExistingCollectionOperationsAreUnchanged re-checks the operations that
// were frozen before this batch.
func TestExistingCollectionOperationsAreUnchanged(t *testing.T) {
	cases := []program{
		{
			name:     "add, eject, and clear still mutate in place",
			sources:  map[string]string{"main.ahd": "values: List<Int> := [1, 2, 3]\nalias: List<Int> := values\nvalues.add(4)\nvalues.eject(0)\nwrite(alias)\nwrite(len(alias))\nclear(values)\nwrite(alias)\n"},
			expected: "[2, 3, 4]\n3\n[]\n",
		},
		{
			name:     "Pair eject still removes one key",
			sources:  map[string]string{"main.ahd": "scores: Pair<String, Int> := {\n    \"a\": 1,\n    \"b\": 2\n}\nscores.eject(\"a\")\nwrite(scores)\nwrite(len(scores))\n"},
			expected: "{\"b\": 2}\n1\n",
		},
		{
			name:     "for still iterates a shallow snapshot",
			sources:  map[string]string{"main.ahd": "values: List<Int> := [1, 2, 3]\nfor value: Int in values {\n    values.add(value)\n}\nwrite(values)\n"},
			expected: "[1, 2, 3, 1, 2, 3]\n",
		},
	}
	runProgramCases(t, cases)
}

// TestStringAndListOperationRejectionsStopBeforeCodeGeneration verifies that
// an invalid operation is a frontend rejection with no derivative IR or
// backend diagnostic.
func TestStringAndListOperationRejectionsStopBeforeCodeGeneration(t *testing.T) {
	cases := map[string]string{
		"null String receiver": "text: String := null\nwrite(text.trim())\n",
		"null List receiver":   "values: List<Int> := null\nwrite(values.count(1))\n",
		"unsupported sort":     "values: List<Bool> := [true]\nvalues.sort()\n",
		"Constant sort":        "values: Constant List<Int> := [3, 1]\nvalues.sort()\n",
		"Constant reverse":     "values: Constant List<Int> := [3, 1]\nvalues.reverse()\n",
		"non-Bool predicate":   listCallbacks + "values: List<Int> := [1]\nwrite(values.filter(double))\n",
		"alias name":           "values: List<Int> := [1]\nvalues.append(2)\n",
		"named argument":       "values: List<Int> := [1]\nwrite(values.count(value: 1))\n",
		"String arity":         "write(\"a\".trim(\"b\"))\n",
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			directory := writeSources(t, map[string]string{"main.ahd": text})
			_, result := BuildProgram(filepath.Join(directory, "main.ahd"), filepath.Join(t.TempDir(), "program"))
			if !result.HasErrors() {
				t.Fatal("expected a compile-time rejection")
			}
			for _, diagnostic := range result.Diagnostics {
				if !strings.HasPrefix(diagnostic.Code, "SEM") {
					t.Fatalf("diagnostics = %s, want only frontend diagnostics", diagnosticText(result.Diagnostics))
				}
			}
		})
	}
}

// TestStudentReportUsesTheStringAndListAPI is the end-to-end program this
// batch is meant to make writable.
func TestStudentReportUsesTheStringAndListAPI(t *testing.T) {
	source := `Student: Class<> := {
    structure: Attributes := (
        name: String
        grade: Int
    )
}

gradeOf: Function := (
    student: Student
) -> Int {
    return student.grade
}

labelOf: Function := (
    student: Student
) -> String {
    return "{student.name} ({student.grade})"
}

passed: Function := (
    student: Student
) -> Bool {
    return student.grade >= 50
}

raw: String := take("Ogrenciler: ")
fields: List<String> := raw.trim().lower().split(",")
students: List<Student> := []

for field: String in fields {
    parts: Local List<String> := field.trim().split(":")
    name: Local String := parts[0]
    score: Local String := parts[1]

    if name != null {
        if score != null {
            students.add(Student(name.trim().capitalize(), int(score.trim())))
        }
    }
}

students.sort(gradeOf)

for label: String in students.map(labelOf) {
    write(label)
}

grades: List<Int> := students.map(gradeOf)
write("Gecen: {len(students.filter(passed))}")
write("En dusuk indeks: {grades.index(min(grades))}")
write("70 alan: {grades.count(70)}")
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	out, errorOutput, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "  Zeynep: 70 , ALI: 45, Mehmet: 90  \n")
	expected := "Ogrenciler: Ali (45)\nZeynep (70)\nMehmet (90)\nGecen: 2\nEn dusuk indeks: 0\n70 alan: 1\n"
	if out != expected || code != 0 {
		t.Fatalf("student report output\n want %q\n have %q (exit %d, stderr %s)", expected, out, code, errorOutput)
	}
}
