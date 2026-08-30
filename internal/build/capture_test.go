package build

import "testing"

// TestExplicitLambdaCaptureRunsNatively exercises closures end to end through
// the Go backend. A capture is passed as a leading parameter of the lifted
// callable, so closure storage is ordinary typed parameter passing.
func TestExplicitLambdaCaptureRunsNatively(t *testing.T) {
	runProgramCases(t, []program{
		{
			name: "captures an enclosing Function parameter",
			sources: map[string]string{"main.ahd": `keep: Function := (
    minimum: Int
    scores: List<Int>
) -> List<Int> {
    return scores.filter(lambda [minimum] (score: Int) -> score >= minimum)
}

write(keep(70, [50, 70, 90, 65]))
write(keep(90, [50, 70, 90, 65]))
`},
			expected: "[70, 90]\n[90]\n",
		},
		{
			name: "captures an enclosing Local and calls the closure",
			sources: map[string]string{"main.ahd": `run: Function := (
) -> Bool {
    minimum: Local Int := 70
    check: Local Function := lambda [minimum] (score: Int) -> score >= minimum
    return check(80)
}

write(run())
`},
			expected: "true\n",
		},
		{
			name: "captures several bindings of different types",
			sources: map[string]string{"main.ahd": `band: Function := (
    low: Int
    high: Int
    label: String
    values: List<Int>
) -> List<String> {
    return values.filter(
        lambda [low, high] (v: Int) -> v >= low and v <= high
    ).map(lambda [label] (v: Int) -> "{label}{v}")
}

write(band(10, 20, "n", [5, 12, 25, 18]))
`},
			expected: "[\"n12\", \"n18\"]\n",
		},
		{
			name: "a captured value outlives the call that created the closure",
			sources: map[string]string{"main.ahd": `adders: Function := (
    base: Int
    values: List<Int>
) -> List<Int> {
    return values.map(lambda [base] (x: Int) -> x + base)
}

write(adders(100, [1, 2, 3]))
write(adders(1000, [1, 2, 3]))
`},
			expected: "[101, 102, 103]\n[1001, 1002, 1003]\n",
		},
		{
			// Capture is by value: the first closure keeps 1 and the second keeps
			// 101, so the sum is 102. Capturing the variable instead of its value
			// would make both read 101 and print 202.
			name: "each evaluation captures the value the binding held then",
			sources: map[string]string{"main.ahd": `bump: Function := (
    start: Int
) -> Int {
    step: Local Int := start
    first: Local Function := lambda [step] (x: Int) -> x + step
    step = step + 100
    second: Local Function := lambda [step] (x: Int) -> x + step
    return first(0) + second(0)
}

write(bump(1))
`},
			expected: "102\n",
		},
		{
			name: "a captured reference still shares the object it refers to",
			sources: map[string]string{"main.ahd": `collect: Function := (
    sink: List<Int>
    values: List<Int>
) -> Int {
    return len(values.map(lambda [sink] (v: Int) -> len(sink) + v))
}

target: List<Int> := [7]
write(collect(target, [1, 2]))
write(target)
`},
			expected: "2\n[7]\n",
		},
		{
			name: "existing lambda syntax and an empty capture list both work",
			sources: map[string]string{"main.ahd": `write([1, 2, 3].map(lambda (x: Int) -> x * 2))
write([1, 2, 3].map(lambda [] (x: Int) -> x * 3))
`},
			expected: "[2, 4, 6]\n[3, 6, 9]\n",
		},
		{
			name: "captures work with a keyed List sort",
			sources: map[string]string{"main.ahd": `rank: Function := (
    pivot: Int
    values: List<Int>
) -> List<Int> {
    values.sort(lambda [pivot] (v: Int) -> pivot - v)
    return values
}

write(rank(100, [10, 30, 20]))
`},
			expected: "[30, 20, 10]\n",
		},
		{
			name: "captures drive the Data callbacks",
			sources: map[string]string{"main.ahd": `bring Data
from Data bring Table

report: Function := (
    minimum: Int
    suffix: String
) -> Nothing {
    students: Local Table := Data.fromCSV("name,score\nAli,91\nAyse,78\nMehmet,84\n")
    passed: Local Table := students.filter(
        lambda [minimum] (row: Pair<String, String>) -> int(row["score"]) >= minimum
    )
    write(passed.column("name"))
    write(passed.sort(lambda [minimum] (row: Pair<String, String>) -> minimum - int(row["score"])).column("name"))
    write(passed.transform("name", lambda [suffix] (value: String) -> value + suffix).column("name"))
    write(passed.derive("band", lambda [minimum] (row: Pair<String, String>) -> str(int(row["score"]) - minimum)).column("band"))
}

report(80, "!")
`},
			expected: "[\"Ali\", \"Mehmet\"]\n[\"Ali\", \"Mehmet\"]\n[\"Ali!\", \"Mehmet!\"]\n[\"11\", \"4\"]\n",
		},
	})
}
