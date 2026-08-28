package build

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestNumericFundamentalsRunAsNativeExecutables covers abs, sum, min, and max
// against real programs rather than semantic metadata.
func TestNumericFundamentalsRunAsNativeExecutables(t *testing.T) {
	cases := []program{
		{
			name:     "abs of a positive Int",
			sources:  map[string]string{"main.ahd": "write(abs(5))\n"},
			expected: "5\n",
		},
		{
			name:     "abs of a negative Int",
			sources:  map[string]string{"main.ahd": "write(abs(-5))\n"},
			expected: "5\n",
		},
		{
			name:     "abs of a positive Real",
			sources:  map[string]string{"main.ahd": "write(abs(2.5))\n"},
			expected: "2.5\n",
		},
		{
			name:     "abs of a negative Real",
			sources:  map[string]string{"main.ahd": "write(abs(-2.5))\n"},
			expected: "2.5\n",
		},
		{
			name:     "abs of negative zero is positive zero",
			sources:  map[string]string{"main.ahd": "write(abs(-0.0))\n"},
			expected: "0.0\n",
		},
		{
			name:       "abs of the minimum Int overflows",
			sources:    map[string]string{"main.ahd": "write(abs(-9223372036854775808))\n"},
			exitCode:   1,
			errorClass: "OverflowError",
		},
		{
			name:     "the minimum Int overflow is catchable",
			sources:  map[string]string{"main.ahd": "attempt {\n    write(abs(-9223372036854775808))\n} except OverflowError as error {\n    write(\"caught\")\n}\n"},
			expected: "caught\n",
		},
		{
			name:     "sum of List<Int>",
			sources:  map[string]string{"main.ahd": "values: List<Int> := [1, 2, 3]\nwrite(sum(values))\n"},
			expected: "6\n",
		},
		{
			name:     "sum of List<Real>",
			sources:  map[string]string{"main.ahd": "values: List<Real> := [1.5, 2.0, 3.5]\nwrite(sum(values))\n"},
			expected: "7.0\n",
		},
		{
			name:     "sum of an empty List<Int> is the Int identity",
			sources:  map[string]string{"main.ahd": "values: List<Int> := []\nwrite(sum(values))\n"},
			expected: "0\n",
		},
		{
			name:     "sum of an empty List<Real> is the Real identity",
			sources:  map[string]string{"main.ahd": "values: List<Real> := []\nwrite(sum(values))\n"},
			expected: "0.0\n",
		},
		{
			name:       "Int summation overflow is reported",
			sources:    map[string]string{"main.ahd": "values: List<Int> := [9223372036854775807, 1]\nwrite(sum(values))\n"},
			exitCode:   1,
			errorClass: "OverflowError",
		},
		{
			name:       "Real summation never produces a non-finite total",
			sources:    map[string]string{"main.ahd": "values: List<Real> := [1.0e308, 1.0e308]\nwrite(sum(values))\n"},
			exitCode:   1,
			errorClass: "OverflowError",
		},
		{
			name:       "a null element is not treated as zero",
			sources:    map[string]string{"main.ahd": "values: List<Int> := [1, null]\nwrite(sum(values))\n"},
			exitCode:   1,
			errorClass: "NullError",
		},
		{
			name:     "min and max of List<Int>",
			sources:  map[string]string{"main.ahd": "values: List<Int> := [8, 3, 12, 5]\nwrite(min(values))\nwrite(max(values))\n"},
			expected: "3\n12\n",
		},
		{
			name:     "min and max of List<Real>",
			sources:  map[string]string{"main.ahd": "values: List<Real> := [3.5, -2.0, 8.25]\nwrite(min(values))\nwrite(max(values))\n"},
			expected: "-2.0\n8.25\n",
		},
		{
			name:     "min and max of a single element",
			sources:  map[string]string{"main.ahd": "values: List<Int> := [7]\nwrite(min(values))\nwrite(max(values))\n"},
			expected: "7\n7\n",
		},
		{
			name:       "min of an empty List is a DomainError",
			sources:    map[string]string{"main.ahd": "values: List<Int> := []\nwrite(min(values))\n"},
			exitCode:   1,
			errorClass: "DomainError",
		},
		{
			name:       "max of an empty List is a DomainError",
			sources:    map[string]string{"main.ahd": "values: List<Real> := []\nwrite(max(values))\n"},
			exitCode:   1,
			errorClass: "DomainError",
		},
		{
			name:     "the empty-List DomainError is catchable",
			sources:  map[string]string{"main.ahd": "values: List<Int> := []\nattempt {\n    write(min(values))\n} except DomainError as error {\n    write(error.message)\n}\n"},
			expected: "min requires a non-empty List\n",
		},
		{
			name:       "min rejects a null element",
			sources:    map[string]string{"main.ahd": "values: List<Int> := [3, null, 1]\nwrite(min(values))\n"},
			exitCode:   1,
			errorClass: "NullError",
		},
		{
			name:       "max rejects a null element",
			sources:    map[string]string{"main.ahd": "values: List<Real> := [1.5, null]\nwrite(max(values))\n"},
			exitCode:   1,
			errorClass: "NullError",
		},
		{
			name:     "a Constant List is a valid read-only argument",
			sources:  map[string]string{"main.ahd": "values: Constant List<Int> := [4, 1, 9]\nwrite(sum(values))\nwrite(min(values))\nwrite(max(values))\nwrite(values)\n"},
			expected: "14\n1\n9\n[4, 1, 9]\n",
		},
		{
			name:     "the argument collection is unchanged",
			sources:  map[string]string{"main.ahd": "values: List<Int> := [3, 1, 2]\nalias: List<Int> := values\nwrite(sum(values))\nwrite(min(values))\nwrite(max(values))\nwrite(values)\nwrite(len(values))\nwrite(alias same values)\n"},
			expected: "6\n1\n3\n[3, 1, 2]\n3\ntrue\n",
		},
		{
			name:     "nested calls compose",
			sources:  map[string]string{"main.ahd": "values: List<Int> := [-40, 12, -3]\nwrite(abs(min(values)))\nwrite(abs(max(values)) + sum(values))\n"},
			expected: "40\n-19\n",
		},
		{
			name:     "module-root and Function-body calls agree",
			sources:  map[string]string{"main.ahd": "report: Function := (\n    values: List<Int>\n) -> Int {\n    return sum(values) + abs(min(values)) + max(values)\n}\n\ndata: List<Int> := [3, -7, 5]\nwrite(report(data))\nwrite(sum(data) + abs(min(data)) + max(data))\n"},
			expected: "13\n13\n",
		},
		{
			name:     "a refined nullable List is accepted",
			sources:  map[string]string{"main.ahd": "values: List<Int> := null\nvalues = [2, 4]\nif values != null {\n    write(sum(values))\n}\n"},
			expected: "6\n",
		},
		{
			name:     "reductions read a List built by iteration",
			sources:  map[string]string{"main.ahd": "values: List<Int> := []\nfor i: Int in between(1, 5) {\n    values.add(i * i)\n}\nwrite(sum(values))\nwrite(min(values))\nwrite(max(values))\n"},
			expected: "30\n1\n16\n",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			directory := writeSources(t, testCase.sources)
			out, errorOutput, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), testCase.stdin)
			if out != testCase.expected {
				t.Fatalf("stdout mismatch\n want %q\n have %q\n stderr: %s", testCase.expected, out, errorOutput)
			}
			if code != testCase.exitCode {
				t.Fatalf("exit code mismatch: want %d, have %d (stderr: %s)", testCase.exitCode, code, errorOutput)
			}
			if testCase.errorClass != "" && !strings.HasPrefix(errorOutput, testCase.errorClass+": ") {
				t.Fatalf("expected an uncaught %s on stderr; received %q", testCase.errorClass, errorOutput)
			}
		})
	}
}

// TestNumericFundamentalsRejectionsStopBeforeCodeGeneration verifies that an
// invalid call is a frontend rejection with no derivative IR or backend
// diagnostic.
func TestNumericFundamentalsRejectionsStopBeforeCodeGeneration(t *testing.T) {
	cases := map[string]string{
		"abs of a String":          "write(abs(\"5\"))\n",
		"abs of a Bool":            "write(abs(true))\n",
		"sum of List<Bool>":        "write(sum([true, false]))\n",
		"sum of a String":          "write(sum(\"abc\"))\n",
		"min of a scalar":          "write(min(5))\n",
		"max of a Pair":            "write(max({\"a\": 1}))\n",
		"sum of a null List":       "values: List<Int> := null\nwrite(sum(values))\n",
		"a named Fundamentals arg": "write(sum(values: [1, 2]))\n",
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

// TestStudentGradeProgramUsesTheNumericFundamentals is the end-to-end program
// the Fundamentals surface is meant to make writable.
func TestStudentGradeProgramUsesTheNumericFundamentals(t *testing.T) {
	source := `count: Int := int(take("Ogrenci sayisi: "))
grades: List<Int> := []

for i: Int in between(count) {
    grade: Local Int := int(take("Not: "))
    grades.add(grade)
}

total: Int := sum(grades)
average: Real := total / len(grades)

write("Toplam: {total}")
write("En dusuk: {min(grades)}")
write("En yuksek: {max(grades)}")
write("Ortalama: {average}")
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	out, errorOutput, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "3\n80\n90\n100\n")
	expected := "Ogrenci sayisi: Not: Not: Not: Toplam: 270\nEn dusuk: 80\nEn yuksek: 100\nOrtalama: 90.0\n"
	if out != expected || code != 0 {
		t.Fatalf("grade program output\n want %q\n have %q (exit %d, stderr %s)", expected, out, code, errorOutput)
	}
	// With no students, sum is still the additive identity and the average
	// divides by zero; min and max report the empty-List DomainError.
	zeroOut, _, zeroCode := buildAndRun(t, filepath.Join(directory, "main.ahd"), "0\n")
	if zeroOut != "Ogrenci sayisi: " || zeroCode != 1 {
		t.Fatalf("zero-student output = %q (exit %d), want the division to fail", zeroOut, zeroCode)
	}
}

// TestEmptyGradeListReductionsAreCatchable separates the three empty-List
// outcomes the zero-student case combines.
func TestEmptyGradeListReductionsAreCatchable(t *testing.T) {
	source := `grades: List<Int> := []
write("Toplam: {sum(grades)}")

attempt {
    write(min(grades))
} except DomainError as error {
    write("min: {error.message}")
}

attempt {
    write(max(grades))
} except DomainError as error {
    write("max: {error.message}")
}

attempt {
    average: Local Real := sum(grades) / len(grades)
    write(average)
} except DivisionByZeroError as error {
    write("average: {error.message}")
}
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	out, errorOutput, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	expected := "Toplam: 0\nmin: min requires a non-empty List\nmax: max requires a non-empty List\naverage: division by zero\n"
	if out != expected || code != 0 {
		t.Fatalf("empty-grade output\n want %q\n have %q (exit %d, stderr %s)", expected, out, code, errorOutput)
	}
}
