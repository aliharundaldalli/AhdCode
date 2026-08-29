package build

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const timeImports = "bring Time\nfrom Time bring DateTime\nfrom Time bring Duration\n\n"

// TestTimeStandardLibraryRunsAsNativeExecutables covers the deterministic part
// of the Time surface end to end.
func TestTimeStandardLibraryRunsAsNativeExecutables(t *testing.T) {
	cases := []program{
		{
			name:     "an ordinary civil date exposes its fields",
			sources:  map[string]string{"main.ahd": timeImports + "value: DateTime := Time.dateTime(year: 2026, month: 5, day: 4, hour: 6, minute: 7, second: 8, millisecond: 9)\nwrite(value.year)\nwrite(value.month)\nwrite(value.day)\nwrite(value.hour)\nwrite(value.minute)\nwrite(value.second)\nwrite(value.millisecond)\n"},
			expected: "2026\n5\n4\n6\n7\n8\n9\n",
		},
		{
			name:     "omitted clock arguments default to zero",
			sources:  map[string]string{"main.ahd": timeImports + "value: DateTime := Time.dateTime(year: 2026, month: 1, day: 1)\nwrite(value.hour)\nwrite(value.minute)\nwrite(value.second)\nwrite(value.millisecond)\n"},
			expected: "0\n0\n0\n0\n",
		},
		{
			name:     "the leap day of a leap year is valid",
			sources:  map[string]string{"main.ahd": timeImports + "value: DateTime := Time.dateTime(year: 2028, month: 2, day: 29)\nwrite(value.toString())\n"},
			expected: "2028-02-29 00:00:00\n",
		},
		{
			name:       "the leap day of an ordinary year is rejected",
			sources:    map[string]string{"main.ahd": timeImports + "value: DateTime := Time.dateTime(year: 2026, month: 2, day: 29)\nwrite(value.year)\n"},
			exitCode:   1,
			errorClass: "ValueError",
		},
		{
			name:       "a day past the end of the month is rejected",
			sources:    map[string]string{"main.ahd": timeImports + "value: DateTime := Time.dateTime(year: 2026, month: 2, day: 30)\nwrite(value.year)\n"},
			exitCode:   1,
			errorClass: "ValueError",
		},
		{
			name:       "an out-of-range month is rejected",
			sources:    map[string]string{"main.ahd": timeImports + "value: DateTime := Time.dateTime(year: 2026, month: 13, day: 1)\nwrite(value.year)\n"},
			exitCode:   1,
			errorClass: "ValueError",
		},
		{
			name:       "an out-of-range hour is rejected",
			sources:    map[string]string{"main.ahd": timeImports + "value: DateTime := Time.dateTime(year: 2026, month: 1, day: 1, hour: 25)\nwrite(value.year)\n"},
			exitCode:   1,
			errorClass: "ValueError",
		},
		{
			name:     "every out-of-range component is a catchable ValueError",
			sources:  map[string]string{"main.ahd": timeImports + "reject: Function := (\n    year: Int\n    month: Int\n    day: Int\n    hour: Int\n    minute: Int\n    second: Int\n    millisecond: Int\n) -> Nothing {\n    attempt {\n        value: Local DateTime := Time.dateTime(year: year, month: month, day: day, hour: hour, minute: minute, second: second, millisecond: millisecond)\n        write(\"accepted {value.year}\")\n    } except ValueError as error {\n        write(\"rejected\")\n    }\n}\n\nreject(0, 1, 1, 0, 0, 0, 0)\nreject(2026, 0, 1, 0, 0, 0, 0)\nreject(2026, 1, 0, 0, 0, 0, 0)\nreject(2026, 1, 1, 24, 0, 0, 0)\nreject(2026, 1, 1, 0, 60, 0, 0)\nreject(2026, 1, 1, 0, 0, 60, 0)\nreject(2026, 1, 1, 0, 0, 0, 1000)\nreject(2026, 1, 1, 23, 59, 59, 999)\n"},
			expected: "rejected\nrejected\nrejected\nrejected\nrejected\nrejected\nrejected\naccepted 2026\n",
		},
		{
			name:     "toString is stable and zero padded",
			sources:  map[string]string{"main.ahd": timeImports + "write(Time.dateTime(year: 2026, month: 1, day: 2, hour: 3, minute: 4, second: 5).toString())\nwrite(Time.dateTime(year: 999, month: 12, day: 31, hour: 23, minute: 59, second: 59).toString())\n"},
			expected: "2026-01-02 03:04:05\n0999-12-31 23:59:59\n",
		},
		{
			name:     "before, after, and sameMoment order two moments",
			sources:  map[string]string{"main.ahd": timeImports + "a: DateTime := Time.dateTime(year: 2026, month: 1, day: 1)\nb: DateTime := Time.dateTime(year: 2026, month: 1, day: 2)\nwrite(a.before(b))\nwrite(b.before(a))\nwrite(a.after(b))\nwrite(b.after(a))\nwrite(a.sameMoment(a))\nwrite(a.sameMoment(b))\n"},
			expected: "true\nfalse\nfalse\ntrue\ntrue\nfalse\n",
		},
		{
			name:     "two separately built equal moments are the same moment",
			sources:  map[string]string{"main.ahd": timeImports + "a: DateTime := Time.dateTime(year: 2026, month: 3, day: 4, hour: 5, minute: 6, second: 7, millisecond: 8)\nb: DateTime := Time.dateTime(year: 2026, month: 3, day: 4, hour: 5, minute: 6, second: 7, millisecond: 8)\nwrite(a.sameMoment(b))\nwrite(a.before(b))\nwrite(a.after(b))\nwrite(a.toString() == b.toString())\n"},
			expected: "true\nfalse\nfalse\ntrue\n",
		},
		{
			name:     "duration carries a signed millisecond count",
			sources:  map[string]string{"main.ahd": timeImports + "zero: Duration := Time.duration(milliseconds: 0)\npositive: Duration := Time.duration(milliseconds: 1500)\nnegative: Duration := Time.duration(milliseconds: -2500)\nwrite(zero.milliseconds)\nwrite(zero.seconds)\nwrite(positive.milliseconds)\nwrite(positive.seconds)\nwrite(negative.milliseconds)\nwrite(negative.seconds)\n"},
			expected: "0\n0.0\n1500\n1.5\n-2500\n-2.5\n",
		},
		{
			name:     "between is second minus first",
			sources:  map[string]string{"main.ahd": timeImports + "a: DateTime := Time.dateTime(year: 2026, month: 1, day: 1)\nb: DateTime := Time.dateTime(year: 2026, month: 1, day: 2)\nwrite(Time.between(a, b).milliseconds)\nwrite(Time.between(b, a).milliseconds)\nwrite(Time.between(a, a).milliseconds)\nwrite(Time.between(a, b).seconds)\n"},
			expected: "86400000\n-86400000\n0\n86400.0\n",
		},
		{
			name:     "calendar applies the Gregorian leap rule",
			sources:  map[string]string{"main.ahd": "bring Time\n\nwrite(Time.Calendar.isLeapYear(2028))\nwrite(Time.Calendar.isLeapYear(2026))\nwrite(Time.Calendar.isLeapYear(2100))\nwrite(Time.Calendar.isLeapYear(2000))\nwrite(Time.Calendar.isLeapYear(1900))\n"},
			expected: "true\nfalse\nfalse\ntrue\nfalse\n",
		},
		{
			name:     "calendar reports month lengths",
			sources:  map[string]string{"main.ahd": "bring Time\n\nwrite(Time.Calendar.daysInMonth(2026, 2))\nwrite(Time.Calendar.daysInMonth(2028, 2))\nwrite(Time.Calendar.daysInMonth(2026, 4))\nwrite(Time.Calendar.daysInMonth(2026, 12))\nwrite(Time.Calendar.daysInMonth(2026, 1))\nwrite(Time.Calendar.daysInMonth(2026, 9))\n"},
			expected: "28\n29\n30\n31\n31\n30\n",
		},
		{
			name:     "calendar weekdays run Monday to Sunday",
			sources:  map[string]string{"main.ahd": "bring Time\n\nwrite(Time.Calendar.weekday(2026, 8, 24))\nwrite(Time.Calendar.weekday(2026, 8, 25))\nwrite(Time.Calendar.weekday(2026, 8, 26))\nwrite(Time.Calendar.weekday(2026, 8, 27))\nwrite(Time.Calendar.weekday(2026, 8, 28))\nwrite(Time.Calendar.weekday(2026, 8, 29))\nwrite(Time.Calendar.weekday(2026, 8, 30))\n"},
			expected: "1\n2\n3\n4\n5\n6\n7\n",
		},
		{
			name:     "a DateTime reports the same weekday as the calendar",
			sources:  map[string]string{"main.ahd": timeImports + "value: DateTime := Time.dateTime(year: 2026, month: 8, day: 29)\nwrite(value.weekday)\nwrite(value.weekday == Time.Calendar.weekday(2026, 8, 29))\n"},
			expected: "6\ntrue\n",
		},
		{
			name:       "an invalid calendar month is rejected",
			sources:    map[string]string{"main.ahd": "bring Time\n\nwrite(Time.Calendar.daysInMonth(2026, 13))\n"},
			exitCode:   1,
			errorClass: "ValueError",
		},
		{
			name:       "an invalid calendar date is rejected",
			sources:    map[string]string{"main.ahd": "bring Time\n\nwrite(Time.Calendar.weekday(2026, 2, 30))\n"},
			exitCode:   1,
			errorClass: "ValueError",
		},
		{
			name:     "calendar errors are catchable",
			sources:  map[string]string{"main.ahd": "bring Time\n\nattempt {\n    write(Time.Calendar.daysInMonth(2026, 0))\n} except ValueError as error {\n    write(error.message)\n}\n"},
			expected: "month 0 is outside 1..12\n",
		},
		{
			name:     "sleep accepts zero and a small positive wait",
			sources:  map[string]string{"main.ahd": "bring Time\n\nTime.sleep(0)\nTime.sleep(5)\nwrite(\"slept\")\n"},
			expected: "slept\n",
		},
		{
			name:       "a negative sleep is rejected rather than clamped",
			sources:    map[string]string{"main.ahd": "bring Time\n\nTime.sleep(-1)\n"},
			exitCode:   1,
			errorClass: "ValueError",
		},
		{
			name:     "a negative sleep is catchable",
			sources:  map[string]string{"main.ahd": "bring Time\n\nattempt {\n    Time.sleep(-1)\n} except ValueError as error {\n    write(error.message)\n}\n"},
			expected: "sleep requires a non-negative number of milliseconds\n",
		},
		{
			name:     "monotonic never moves backwards",
			sources:  map[string]string{"main.ahd": "bring Time\n\nfirst: Real := Time.monotonic()\nTime.sleep(1)\nsecond: Real := Time.monotonic()\nThird: Real := Time.monotonic()\nwrite(second >= first)\nwrite(Third >= second)\nwrite(first >= 0.0)\n"},
			expected: "true\ntrue\ntrue\n",
		},
		{
			name:     "monotonic measures an elapsed sleep",
			sources:  map[string]string{"main.ahd": "bring Time\n\nstart: Real := Time.monotonic()\nTime.sleep(120)\nelapsed: Real := Time.monotonic() - start\nwrite(elapsed >= 0.1)\n"},
			expected: "true\n",
		},
		{
			name:     "now reports coherent local calendar fields",
			sources:  map[string]string{"main.ahd": timeImports + "current: DateTime := Time.now()\nwrite(current.month >= 1 and current.month <= 12)\nwrite(current.day >= 1 and current.day <= Time.Calendar.daysInMonth(current.year, current.month))\nwrite(current.hour >= 0 and current.hour <= 23)\nwrite(current.minute >= 0 and current.minute <= 59)\nwrite(current.second >= 0 and current.second <= 59)\nwrite(current.millisecond >= 0 and current.millisecond <= 999)\nwrite(current.weekday == Time.Calendar.weekday(current.year, current.month, current.day))\n"},
			expected: "true\ntrue\ntrue\ntrue\ntrue\ntrue\ntrue\n",
		},
		{
			name:     "member existence reports the real Time members",
			sources:  map[string]string{"main.ahd": timeImports + "current: DateTime := Time.now()\nwait: Duration := Time.duration(milliseconds: 1)\nwrite(current has year)\nwrite(current has weekday)\nwrite(current has before)\nwrite(current has toString)\nwrite(current has milliseconds)\nwrite(current has missing)\nwrite(wait has milliseconds)\nwrite(wait has seconds)\nwrite(wait has year)\nwrite(current has not missing)\n"},
			expected: "true\ntrue\ntrue\ntrue\nfalse\nfalse\ntrue\ntrue\nfalse\ntrue\n",
		},
		{
			name:     "Time values participate in ordinary Class rules",
			sources:  map[string]string{"main.ahd": timeImports + "a: DateTime := Time.dateTime(year: 2026, month: 1, day: 1)\nb: DateTime := a\nwrite(a same b)\nwrite(a == b)\nwrite(str(a))\n"},
			expected: "true\ntrue\n<DateTime>\n",
		},
	}
	runProgramCases(t, cases)
}

// TestTimeNowLiesBetweenTwoHostReadings is the stable way to check the clock:
// the AhdCode reading must fall between a host reading taken before and one
// taken after, rather than matching an exact instant.
func TestTimeNowLiesBetweenTwoHostReadings(t *testing.T) {
	source := timeImports + `current: DateTime := Time.now()
write(current.year)
write(current.month)
write(current.day)
write(current.hour)
write(current.minute)
write(current.second)
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	before := time.Now().Add(-2 * time.Second)
	out, errorOutput, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	after := time.Now().Add(2 * time.Second)
	if code != 0 {
		t.Fatalf("exit %d, stderr %s", code, errorOutput)
	}
	fields := strings.Fields(out)
	if len(fields) != 6 {
		t.Fatalf("output = %q, want six fields", out)
	}
	numbers := make([]int, 6)
	for index, field := range fields {
		value, err := strconv.Atoi(field)
		if err != nil {
			t.Fatalf("field %d = %q: %v", index, field, err)
		}
		numbers[index] = value
	}
	reported := time.Date(numbers[0], time.Month(numbers[1]), numbers[2],
		numbers[3], numbers[4], numbers[5], 0, time.Local)
	if reported.Before(before) || reported.After(after) {
		t.Fatalf("Time.now() reported %s, which is outside [%s, %s]", reported, before, after)
	}
}

// TestTimeProgramsAgreeBetweenRunAndBuild checks that the deterministic part of
// the Time surface behaves identically on both entry points.
func TestTimeProgramsAgreeBetweenRunAndBuild(t *testing.T) {
	source := timeImports + `a: DateTime := Time.dateTime(year: 2026, month: 1, day: 1)
b: DateTime := Time.dateTime(year: 2028, month: 2, day: 29, hour: 12, minute: 30)

write(a.toString())
write(b.toString())
write(b.weekday)
write(a.before(b))
write(Time.between(a, b).milliseconds)
write(Time.between(a, b).seconds)
write(Time.Calendar.isLeapYear(2028))
write(Time.Calendar.daysInMonth(2028, 2))
write(Time.Calendar.weekday(2026, 8, 29))
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	entry := filepath.Join(directory, "main.ahd")

	ran, errorOutput, code := buildAndRun(t, entry, "")
	if code != 0 {
		t.Fatalf("run exit %d, stderr %s", code, errorOutput)
	}
	built := buildExecutable(t, source)
	command := exec.Command(built)
	var produced strings.Builder
	command.Stdout = &produced
	if err := command.Run(); err != nil {
		t.Fatalf("the built executable failed: %v", err)
	}
	if ran != produced.String() {
		t.Fatalf("run and build disagree\n run   %q\n build %q", ran, produced.String())
	}
	if !strings.HasPrefix(ran, "2026-01-01 00:00:00\n2028-02-29 12:30:00\n") {
		t.Fatalf("unexpected output %q", ran)
	}
}

// TestTimeWorksAcrossModuleBoundaries checks that an imported module may use
// Time and hand its values back to the entry module.
func TestTimeWorksAcrossModuleBoundaries(t *testing.T) {
	sources := map[string]string{
		"Schedule.ahd": "bring Time\nfrom Time bring DateTime\n\nopening: Function := (\n) -> DateTime {\n    return Time.dateTime(year: 2026, month: 9, day: 1, hour: 9)\n}\n",
		"main.ahd":     "bring Time\nfrom Time bring DateTime\nfrom Schedule bring opening\n\nstart: DateTime := opening()\nwrite(start.toString())\nwrite(start.weekday)\nwrite(Time.Calendar.daysInMonth(start.year, start.month))\n",
	}
	directory := writeSources(t, sources)
	out, errorOutput, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	expected := "2026-09-01 09:00:00\n2\n30\n"
	if out != expected || code != 0 {
		t.Fatalf("cross-module Time output\n want %q\n have %q (exit %d, stderr %s)", expected, out, code, errorOutput)
	}
}

// TestTimeProgramsAreRejected keeps invalid Time use a frontend rejection with
// no derivative IR or backend diagnostic.
func TestTimeProgramsAreRejected(t *testing.T) {
	cases := map[string]string{
		"sleep with a String":        "bring Time\n\nTime.sleep(\"500\")\n",
		"isLeapYear with a String":   "bring Time\n\nwrite(Time.Calendar.isLeapYear(\"2028\"))\n",
		"daysInMonth with a String":  "bring Time\n\nwrite(Time.Calendar.daysInMonth(2028, \"2\"))\n",
		"between with Int arguments": timeImports + "gap: Duration := Time.between(10, 20)\n",
		"assigning a DateTime field": timeImports + "current: DateTime := Time.now()\ncurrent.year = 2030\n",
		"constructing a DateTime":    timeImports + "value: DateTime := DateTime(year: 2026, month: 1, day: 1)\n",
		"an unknown Time member":     "bring Time\n\nwrite(Time.epoch())\n",
		"a DateTime without import":  "bring Time\n\ncurrent: DateTime := Time.now()\n",
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
