package build

import (
	"path/filepath"
	"testing"
)

// TestStatisticsStandardLibraryRunsNatively covers the descriptive statistics
// surface end to end, including the deliberate empty-input and boundary rules.
func TestStatisticsStandardLibraryRunsNatively(t *testing.T) {
	preamble := "bring Statistics\nfrom Statistics bring StatisticsError\n\n"
	runProgramCases(t, []program{
		{
			name: "Int statistics keep Int where the answer is an input value",
			sources: map[string]string{"main.ahd": preamble + `values: List<Int> := [3, 1, 4, 1, 5]
write(Statistics.sum(values))
write(Statistics.min(values))
write(Statistics.max(values))
write(Statistics.range(values))
write(Statistics.mode(values))
write(Statistics.mean(values))
write(Statistics.median(values))
`},
			expected: "14\n1\n5\n4\n1\n2.8\n3.0\n",
		},
		{
			name: "Real statistics",
			sources: map[string]string{"main.ahd": preamble + `values: List<Real> := [2.5, 1.5, 3.5]
write(Statistics.sum(values))
write(Statistics.min(values))
write(Statistics.max(values))
write(Statistics.mean(values))
write(Statistics.median(values))
`},
			expected: "7.5\n1.5\n3.5\n2.5\n2.5\n",
		},
		{
			name: "population and sample dispersion are separate functions",
			sources: map[string]string{"main.ahd": preamble + `values: List<Int> := [3, 1, 4, 1, 5]
write(Statistics.variance(values))
write(Statistics.sampleVariance(values))
write(Statistics.stdDev(values))
`},
			expected: "2.56\n3.2\n1.6\n",
		},
		{
			name: "median averages the two middle values for an even count",
			sources: map[string]string{"main.ahd": preamble + `write(Statistics.median([1, 2, 3, 4]))
write(Statistics.median([1, 2, 3]))
`},
			expected: "2.5\n2.0\n",
		},
		{
			name: "quantile interpolates linearly between order statistics",
			sources: map[string]string{"main.ahd": preamble + `values: List<Int> := [1, 2, 3, 4]
write(Statistics.quantile(values, 0.0))
write(Statistics.quantile(values, 0.25))
write(Statistics.quantile(values, 0.5))
write(Statistics.quantile(values, 1.0))
write(Statistics.quantile([7], 0.5))
`},
			expected: "1.0\n1.75\n2.5\n4.0\n7.0\n",
		},
		{
			name: "negatives duplicates and a single value behave normally",
			sources: map[string]string{"main.ahd": preamble + `values: List<Int> := [-5, -1, -5, -1, 3]
write(Statistics.min(values))
write(Statistics.mean(values))
write(Statistics.mode(values))
write(Statistics.mean([7]))
write(Statistics.variance([7]))
`},
			expected: "-5\n-1.8\n-5\n7.0\n0.0\n",
		},
		{
			name: "a mode tie is broken by first occurrence",
			sources: map[string]string{"main.ahd": preamble + `write(Statistics.mode([2, 3, 3, 2]))
write(Statistics.mode([3, 2, 2, 3]))
`},
			expected: "2\n3\n",
		},
		{
			name: "the empty sum is defined and every other statistic is not",
			sources: map[string]string{"main.ahd": preamble + `empty: List<Int> := []
write(Statistics.sum(empty))
attempt { Statistics.mean(empty) } except StatisticsError as error { write(error.message) }
attempt { Statistics.median(empty) } except StatisticsError as error { write(error.message) }
attempt { Statistics.min(empty) } except StatisticsError as error { write(error.message) }
attempt { Statistics.max(empty) } except StatisticsError as error { write(error.message) }
attempt { Statistics.range(empty) } except StatisticsError as error { write(error.message) }
attempt { Statistics.variance(empty) } except StatisticsError as error { write(error.message) }
attempt { Statistics.mode(empty) } except StatisticsError as error { write(error.message) }
attempt { Statistics.quantile(empty, 0.5) } except StatisticsError as error { write(error.message) }
`},
			expected: "0\nmean is undefined for an empty List\nmedian is undefined for an empty List\n" +
				"min is undefined for an empty List\nmax is undefined for an empty List\n" +
				"range is undefined for an empty List\nvariance is undefined for an empty List\n" +
				"mode is undefined for an empty List\nquantile is undefined for an empty List\n",
		},
		{
			name: "sample dispersion needs two values and quantile needs a probability",
			sources: map[string]string{"main.ahd": preamble + `one: List<Int> := [7]
attempt { Statistics.sampleVariance(one) } except StatisticsError as error { write(error.message) }
attempt { Statistics.sampleStdDev(one) } except StatisticsError as error { write(error.message) }
attempt { Statistics.quantile(one, 1.5) } except StatisticsError as error { write(error.message) }
attempt { Statistics.quantile(one, -0.1) } except StatisticsError as error { write(error.message) }
`},
			expected: "sampleVariance requires at least two values\nsampleVariance requires at least two values\n" +
				"quantile probability must be between 0.0 and 1.0\nquantile probability must be between 0.0 and 1.0\n",
		},
		{
			name: "the input List is never reordered",
			sources: map[string]string{"main.ahd": preamble + `values: List<Int> := [3, 1, 2]
write(Statistics.median(values))
write(Statistics.quantile(values, 0.5))
write(values)
`},
			expected: "2.0\n2.0\n[3, 1, 2]\n",
		},
		{
			name: "Data supplies numbers only through an explicit conversion",
			sources: map[string]string{"main.ahd": "bring Data\n" + preamble + `from Data bring Table

students: Table := Data.fromCSV("name,score\nAli,91\nAyse,78\nMehmet,84\n")
scores: List<Real> := students.column("score").map(lambda (value: String) -> real(value))
write(Statistics.mean(scores))
write(Statistics.median(scores))
`},
			expected: "84.33333333333333\n84.0\n",
		},
	})
}

// TestStatisticsRejectsNonNumericListsStatically keeps String cells out: a
// numeric statistic never coerces text, so Data users convert explicitly.
func TestStatisticsRejectsNonNumericListsStatically(t *testing.T) {
	for _, source := range []string{
		"bring Statistics\nwrite(Statistics.mean([\"10\", \"20\"]))\n",
		"bring Statistics\nwrite(Statistics.sum([true, false]))\n",
	} {
		directory := writeSources(t, map[string]string{"main.ahd": source})
		path, result := BuildProgram(filepath.Join(directory, "main.ahd"), filepath.Join(t.TempDir(), "program"))
		if path != "" || !result.HasErrors() {
			t.Fatalf("a non-numeric List must not compile: %s", source)
		}
	}
}
