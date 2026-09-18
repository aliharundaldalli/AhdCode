package ahdruntime

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func nearReal(actual, expected float64) bool {
	return math.Abs(actual-expected) <= 1e-12*math.Max(1, math.Abs(expected))
}

func requireFault(t *testing.T, fault *AhdFault, class, fragment string) {
	t.Helper()
	if fault == nil {
		t.Fatalf("expected %s containing %q, got success", class, fragment)
	}
	if fault.Class != class || !strings.Contains(fault.Message, fragment) {
		t.Fatalf("expected %s containing %q, got %s: %s", class, fragment, fault.Class, fault.Message)
	}
}

func requireNoFault(t *testing.T, fault *AhdFault) {
	t.Helper()
	if fault != nil {
		t.Fatalf("unexpected %s: %s", fault.Class, fault.Message)
	}
}

// ---- Math ----

func TestMathUnaryValuesAndIdentities(t *testing.T) {
	cases := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"asin", 1, math.Pi / 2}, {"asin", -1, -math.Pi / 2}, {"asin", 0, 0},
		{"acos", 1, 0}, {"acos", -1, math.Pi}, {"acos", 0, math.Pi / 2},
		{"atan", 1, math.Pi / 4}, {"atan", -1e300, -math.Pi / 2},
		{"sinh", 0, 0}, {"sinh", 1, math.Sinh(1)},
		{"cosh", 0, 1}, {"cosh", -1, math.Cosh(1)},
		{"tanh", 0, 0}, {"tanh", 1000, 1}, {"tanh", -1000, -1},
		{"log2", 1, 0}, {"log2", 8, 3}, {"log2", 0.5, -1},
		{"cbrt", 27, 3}, {"cbrt", -27, -3}, {"cbrt", 0, 0},
		{"radians", 180, math.Pi}, {"radians", -90, -math.Pi / 2}, {"radians", 720, 4 * math.Pi},
		{"degrees", math.Pi, 180}, {"degrees", -math.Pi / 2, -90}, {"degrees", 4 * math.Pi, 720},
	}
	for _, testCase := range cases {
		result, fault := AhdMathUnary(testCase.name, testCase.input)
		requireNoFault(t, fault)
		if !nearReal(result, testCase.expected) {
			t.Errorf("Math.%s(%v) = %v, want %v", testCase.name, testCase.input, result, testCase.expected)
		}
	}
	// radians and degrees are inverse conversions with no normalization.
	for _, angle := range []float64{0, 1, -1, 45, 359.5, 1e6} {
		radians, _ := AhdMathUnary("radians", angle)
		back, _ := AhdMathUnary("degrees", radians)
		if !nearReal(back, angle) {
			t.Errorf("degrees(radians(%v)) = %v", angle, back)
		}
	}
	// Inverse functions undo their forward functions on their domains.
	for _, value := range []float64{-0.9, -0.5, 0, 0.3, 0.99} {
		asin, _ := AhdMathUnary("asin", value)
		if !nearReal(math.Sin(asin), value) {
			t.Errorf("sin(asin(%v)) = %v", value, math.Sin(asin))
		}
		acos, _ := AhdMathUnary("acos", value)
		if !nearReal(math.Cos(acos), value) {
			t.Errorf("cos(acos(%v)) = %v", value, math.Cos(acos))
		}
	}
}

func TestMathDomainsAndNonFiniteResults(t *testing.T) {
	for _, value := range []float64{1.0000001, -1.0000001, 2, -1e300} {
		_, fault := AhdMathUnary("asin", value)
		requireFault(t, fault, "DomainError", "Math.asin requires a value between -1 and 1")
		_, fault = AhdMathUnary("acos", value)
		requireFault(t, fault, "DomainError", "Math.acos requires a value between -1 and 1")
	}
	for _, value := range []float64{0, -1, -1e-300} {
		_, fault := AhdMathUnary("log2", value)
		requireFault(t, fault, "DomainError", "Math.log2 requires a value greater than zero")
	}
	for _, name := range []string{"sinh", "cosh"} {
		_, fault := AhdMathUnary(name, 1000)
		requireFault(t, fault, "OverflowError", "Math."+name+" result exceeds finite Real range")
	}
	_, fault := AhdMathUnary("radians", math.MaxFloat64)
	requireFault(t, fault, "OverflowError", "Math.radians")
	_, fault = AhdMathUnary("degrees", math.MaxFloat64)
	requireFault(t, fault, "OverflowError", "Math.degrees")
	_, fault = AhdMathUnary("cbrt", math.NaN())
	requireFault(t, fault, "DomainError", "received NaN")
	_, fault = AhdMathUnary("atan", math.Inf(1))
	requireFault(t, fault, "OverflowError", "non-finite")
	_, fault = AhdMathBinary("hypot", math.MaxFloat64, math.MaxFloat64)
	requireFault(t, fault, "OverflowError", "Math.hypot result exceeds finite Real range")
}

func TestMathAtan2AndHypot(t *testing.T) {
	cases := []struct {
		name          string
		first, second float64
		expected      float64
	}{
		{"atan2", 1, 1, math.Pi / 4}, {"atan2", 1, -1, 3 * math.Pi / 4},
		{"atan2", -1, -1, -3 * math.Pi / 4}, {"atan2", -1, 1, -math.Pi / 4},
		{"atan2", 0, -1, math.Pi}, {"atan2", 1, 0, math.Pi / 2}, {"atan2", 0, 0, 0},
		{"hypot", 3, 4, 5}, {"hypot", -3, -4, 5}, {"hypot", 0, 0, 0},
		{"hypot", 1e200, 1e200, math.Sqrt2 * 1e200},
	}
	for _, testCase := range cases {
		result, fault := AhdMathBinary(testCase.name, testCase.first, testCase.second)
		requireNoFault(t, fault)
		if !nearReal(result, testCase.expected) {
			t.Errorf("Math.%s(%v, %v) = %v, want %v", testCase.name, testCase.first, testCase.second, result, testCase.expected)
		}
	}
}

func TestMathGCDAndLCM(t *testing.T) {
	gcd := []struct{ first, second, expected int64 }{
		{0, 0, 0}, {12, 18, 6}, {-12, 18, 6}, {12, -18, 6}, {-12, -18, 6},
		{0, 7, 7}, {7, 0, 7}, {0, -7, 7}, {17, 5, 1},
		{math.MinInt64, 2, 2}, {math.MaxInt64, math.MaxInt64, math.MaxInt64},
		{math.MinInt64, math.MaxInt64, 1},
	}
	for _, testCase := range gcd {
		result, fault := AhdMathGCD(testCase.first, testCase.second)
		requireNoFault(t, fault)
		if result != testCase.expected {
			t.Errorf("gcd(%d, %d) = %d, want %d", testCase.first, testCase.second, result, testCase.expected)
		}
	}
	for _, pair := range [][2]int64{{math.MinInt64, 0}, {0, math.MinInt64}, {math.MinInt64, math.MinInt64}} {
		_, fault := AhdMathGCD(pair[0], pair[1])
		requireFault(t, fault, "OverflowError", "Math.gcd result does not fit Int")
	}
	lcm := []struct{ first, second, expected int64 }{
		{0, 0, 0}, {5, 0, 0}, {0, -5, 0}, {math.MinInt64, 0, 0},
		{4, 6, 12}, {-4, 6, 12}, {4, -6, 12}, {-4, -6, 12}, {7, 7, 7}, {1, math.MaxInt64, math.MaxInt64},
		{-1, math.MaxInt64, math.MaxInt64}, {1 << 31, 1 << 31, 1 << 31},
	}
	for _, testCase := range lcm {
		result, fault := AhdMathLCM(testCase.first, testCase.second)
		requireNoFault(t, fault)
		if result != testCase.expected {
			t.Errorf("lcm(%d, %d) = %d, want %d", testCase.first, testCase.second, result, testCase.expected)
		}
	}
	for _, pair := range [][2]int64{{math.MaxInt64, 2}, {math.MinInt64, 1}, {math.MinInt64, -1}, {math.MinInt64, 3}, {1 << 62, 3}, {3037000500, 3037000501}} {
		_, fault := AhdMathLCM(pair[0], pair[1])
		requireFault(t, fault, "OverflowError", "Math.lcm result does not fit Int")
	}
}

// ---- Time ----

func TestTimeParseISOAcceptsTheStrictSubset(t *testing.T) {
	cases := []struct {
		text                        string
		year, month, day, hour, min int
		second, millisecond, offset int
	}{
		{"2026-09-18T10:30:00Z", 2026, 9, 18, 10, 30, 0, 0, 0},
		{"2026-09-18T10:30:05.3Z", 2026, 9, 18, 10, 30, 5, 300, 0},
		{"2026-09-18T10:30:05.35Z", 2026, 9, 18, 10, 30, 5, 350, 0},
		{"2026-09-18T10:30:05.007Z", 2026, 9, 18, 10, 30, 5, 7, 0},
		{"2026-09-18T13:30:00+03:00", 2026, 9, 18, 13, 30, 0, 0, 180},
		{"2026-09-18T05:00:00-05:30", 2026, 9, 18, 5, 0, 0, 0, -330},
		{"2026-09-18T05:00:00+00:00", 2026, 9, 18, 5, 0, 0, 0, 0},
		{"2026-09-18T05:00:00-00:00", 2026, 9, 18, 5, 0, 0, 0, 0},
		{"2024-02-29T23:59:59.999+14:00", 2024, 2, 29, 23, 59, 59, 999, 840},
		{"0001-01-01T00:00:00Z", 1, 1, 1, 0, 0, 0, 0, 0},
		{"9999-12-31T23:59:59.999-14:00", 9999, 12, 31, 23, 59, 59, 999, -840},
	}
	for _, testCase := range cases {
		value, fault := AhdTimeParseISO(testCase.text)
		requireNoFault(t, fault)
		_, offset := value.Zone()
		got := []int{value.Year(), int(value.Month()), value.Day(), value.Hour(), value.Minute(), value.Second(), value.Nanosecond() / 1e6, offset / 60}
		want := []int{testCase.year, testCase.month, testCase.day, testCase.hour, testCase.min, testCase.second, testCase.millisecond, testCase.offset}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("parseISO(%q) = %v, want %v", testCase.text, got, want)
		}
	}
}

func TestTimeParseISORejectsEverythingElse(t *testing.T) {
	rejected := []string{
		"", "2026-09-18", "2026-09-18T10:30", "2026-09-18T10:30:00", "2026-09-18T10:30:00.123",
		"2026-09-18 10:30:00Z", "2026-09-18t10:30:00Z", "2026-09-18T10:30:00z",
		"2026-09-18T10:30:00.Z", "2026-09-18T10:30:00.1234Z", "2026-09-18T10:30:00,5Z",
		"2026-09-18T10:30:00+03", "2026-09-18T10:30:00+0300", "2026-09-18T10:30:00+3:00",
		"2026-09-18T10:30:00 +03:00", "2026-09-18T10:30:00Z ", " 2026-09-18T10:30:00Z",
		"2026-09-18T10:30:00ZZ", "2026-09-18T10:30:00UTC", "2026-09-18T10:30:00+03:00Z",
		"26-09-18T10:30:00Z", "+2026-09-18T10:30:00Z", "2026-9-18T10:30:00Z", "2026-09-18T1:30:00Z",
		"２０２６-09-18T10:30:00Z", "2026-09-18T10:30:00.٣Z",
		"Sep 18 2026", "18/09/2026", "UTC+3", "Europe/Istanbul", "now", "tomorrow at noon",
		"2026-09-18T10:30:00[Europe/Istanbul]", "2026-W38-5T10:30:00Z", "2026-261T10:30:00Z",
	}
	for _, text := range rejected {
		_, fault := AhdTimeParseISO(text)
		requireFault(t, fault, "ValueError", "Time.parseISO")
	}
	impossible := map[string]string{
		"0000-01-01T00:00:00Z":      "year 0000",
		"2026-00-10T00:00:00Z":      "month 0",
		"2026-13-10T00:00:00Z":      "month 13",
		"2026-02-29T00:00:00Z":      "day 29 does not exist in 2026-02",
		"2026-04-31T00:00:00Z":      "day 31",
		"2026-01-00T00:00:00Z":      "day 0",
		"2026-01-01T24:00:00Z":      "hour 24",
		"2026-01-01T00:60:00Z":      "minute 60",
		"2026-12-31T23:59:60Z":      "second 60",
		"2026-01-01T00:00:00+14:01": "outside -14:00..+14:00",
		"2026-01-01T00:00:00-15:00": "outside -14:00..+14:00",
		"2026-01-01T00:00:00+05:60": "Time.parseISO expects",
	}
	for text, fragment := range impossible {
		_, fault := AhdTimeParseISO(text)
		requireFault(t, fault, "ValueError", fragment)
	}
}

func TestTimeFormatISOAndRoundTrip(t *testing.T) {
	cases := map[time.Time]string{
		time.Date(2026, 9, 18, 10, 30, 0, 0, time.UTC):                                     "2026-09-18T10:30:00.000Z",
		time.Date(2026, 9, 18, 10, 30, 0, 7e6, time.FixedZone("", 0)):                      "2026-09-18T10:30:00.007Z",
		time.Date(2026, 9, 18, 13, 30, 0, 350e6, time.FixedZone("", 3*3600)):               "2026-09-18T13:30:00.350+03:00",
		time.Date(1, 1, 1, 0, 0, 0, 0, time.FixedZone("", -(5*3600+30*60))):                "0001-01-01T00:00:00.000-05:30",
		time.Date(9999, 12, 31, 23, 59, 59, 999e6+999999, time.FixedZone("", 14*3600)):     "9999-12-31T23:59:59.999+14:00",
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.FixedZone("", -(9*3600+45*60))):             "2026-03-01T00:00:00.000-09:45",
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.FixedZone("Anywhere/Named", 5*3600+45*60)):  "2026-03-01T00:00:00.000+05:45",
		time.Date(2026, 3, 1, 12, 0, 0, 0, time.FixedZone("", -12*3600)):                   "2026-03-01T12:00:00.000-12:00",
		time.Date(2024, 2, 29, 23, 59, 59, 900e6, time.FixedZone("", -(5*3600+30*60))):     "2024-02-29T23:59:59.900-05:30",
		time.Date(2026, 9, 18, 10, 30, 0, 0, time.FixedZone("", 1*3600+0*60)).In(time.UTC): "2026-09-18T09:30:00.000Z",
	}
	for value, expected := range cases {
		text, fault := AhdTimeFormatISO(value)
		requireNoFault(t, fault)
		if text != expected {
			t.Errorf("toISO(%v) = %q, want %q", value, text, expected)
		}
		back, fault := AhdTimeParseISO(text)
		requireNoFault(t, fault)
		if !back.Equal(value.Truncate(time.Millisecond)) {
			t.Errorf("parseISO(toISO(%v)) = %v is not the same moment", value, back)
		}
		if again, _ := AhdTimeFormatISO(back); again != text {
			t.Errorf("toISO is not stable: %q then %q", text, again)
		}
	}
	// A historical offset with a seconds part cannot be written; toUTC first.
	istanbulLMT := time.Date(1870, 1, 1, 12, 0, 0, 0, time.FixedZone("LMT", 1*3600+55*60+52))
	_, fault := AhdTimeFormatISO(istanbulLMT)
	requireFault(t, fault, "ValueError", "toUTC() first")
	text, fault := AhdTimeFormatISO(istanbulLMT.UTC())
	requireNoFault(t, fault)
	if text != "1870-01-01T10:04:08.000Z" {
		t.Fatalf("LMT instant in UTC = %q", text)
	}
}

func TestTimeShiftKeepsTheOffsetAndTheRange(t *testing.T) {
	plus3 := time.FixedZone("", 3*3600)
	start := time.Date(2026, 9, 18, 23, 30, 0, 250e6, plus3)
	shifted, fault := AhdTimeShift(start, 3600000, false)
	requireNoFault(t, fault)
	if text, _ := AhdTimeFormatISO(shifted); text != "2026-09-19T00:30:00.250+03:00" {
		t.Fatalf("add one hour = %q", text)
	}
	back, fault := AhdTimeShift(shifted, 3600000, true)
	requireNoFault(t, fault)
	if !back.Equal(start) {
		t.Fatalf("subtract did not undo add: %v", back)
	}
	// Negative durations move the other way; UTC stays UTC.
	utc := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	earlier, _ := AhdTimeShift(utc, -86400000, false)
	if text, _ := AhdTimeFormatISO(earlier); text != "2024-02-29T00:00:00.000Z" {
		t.Fatalf("add a negative day = %q", text)
	}
	// A sub-minute historical offset is kept exactly.
	lmt := time.FixedZone("", 1*3600+55*60+52)
	historical, fault := AhdTimeShift(time.Date(1870, 1, 1, 0, 0, 0, 0, lmt), 1000, false)
	requireNoFault(t, fault)
	if _, offset := historical.Zone(); offset != 1*3600+55*60+52 || historical.Second() != 1 {
		t.Fatalf("historical offset changed: %v", historical)
	}
	last := time.Date(9999, 12, 31, 23, 59, 59, 999e6, time.UTC)
	_, fault = AhdTimeShift(last, 1, false)
	requireFault(t, fault, "ValueError", "DateTime.add result is outside the supported DateTime range")
	first := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	_, fault = AhdTimeShift(first, 1, true)
	requireFault(t, fault, "ValueError", "DateTime.subtract result is outside")
	for _, milliseconds := range []int64{math.MaxInt64, math.MinInt64, math.MinInt64 + 1} {
		_, fault = AhdTimeShift(start, milliseconds, false)
		requireFault(t, fault, "ValueError", "outside the supported DateTime range")
		_, fault = AhdTimeShift(start, milliseconds, true)
		requireFault(t, fault, "ValueError", "outside the supported DateTime range")
	}
}

func TestDurationArithmeticIsCheckedInt(t *testing.T) {
	cases := []struct {
		name                    string
		first, second, expected int64
	}{
		{"add", 1500, 500, 2000}, {"add", -1, 1, 0}, {"subtract", 1500, 2000, -500},
		{"negate", 1500, 0, -1500}, {"negate", 0, 0, 0}, {"negate", math.MaxInt64, 0, -math.MaxInt64},
		{"abs", -1500, 0, 1500}, {"abs", 1500, 0, 1500}, {"abs", math.MinInt64 + 1, 0, math.MaxInt64},
		{"add", math.MaxInt64 - 1, 1, math.MaxInt64}, {"subtract", math.MinInt64 + 1, 1, math.MinInt64},
	}
	for _, testCase := range cases {
		result, fault := AhdDurationArithmetic(testCase.name, testCase.first, testCase.second)
		requireNoFault(t, fault)
		if result != testCase.expected {
			t.Errorf("Duration.%s(%d, %d) = %d, want %d", testCase.name, testCase.first, testCase.second, result, testCase.expected)
		}
	}
	overflow := []struct {
		name          string
		first, second int64
	}{
		{"add", math.MaxInt64, 1}, {"add", math.MinInt64, -1}, {"subtract", math.MinInt64, 1},
		{"subtract", math.MaxInt64, -1}, {"subtract", 0, math.MinInt64}, {"negate", math.MinInt64, 0}, {"abs", math.MinInt64, 0},
	}
	for _, testCase := range overflow {
		_, fault := AhdDurationArithmetic(testCase.name, testCase.first, testCase.second)
		requireFault(t, fault, "OverflowError", "Duration."+testCase.name+" overflowed Int milliseconds")
	}
}

// ---- Numeric ----

func TestVectorMembers(t *testing.T) {
	values := []float64{3, 4, 12}
	for index, expected := range map[int64]float64{0: 3, 1: 4, 2: 12, -1: 12, -3: 3} {
		result, fault := AhdVectorAt(values, index)
		requireNoFault(t, fault)
		if result != expected {
			t.Errorf("at(%d) = %v", index, result)
		}
	}
	for _, index := range []int64{3, -4, math.MaxInt64, math.MinInt64} {
		_, fault := AhdVectorAt(values, index)
		requireFault(t, fault, "IndexError", "out of range for length 3")
	}
	_, fault := AhdVectorAt(nil, 0)
	requireFault(t, fault, "IndexError", "length 0")

	if norm, _ := AhdVectorNorm(values); norm != 13 {
		t.Errorf("norm = %v", norm)
	}
	if norm, _ := AhdVectorNorm(nil); norm != 0 {
		t.Errorf("empty norm = %v", norm)
	}
	// Scaling keeps a representable norm finite and small values exact.
	if norm, fault := AhdVectorNorm([]float64{3e300, 4e300}); fault != nil || !nearReal(norm, 5e300) {
		t.Errorf("large norm = %v %v", norm, fault)
	}
	if norm, _ := AhdVectorNorm([]float64{3e-300, 4e-300}); !nearReal(norm, 5e-300) {
		t.Errorf("tiny norm = %v", norm)
	}
	_, fault = AhdVectorNorm([]float64{math.MaxFloat64, math.MaxFloat64})
	requireFault(t, fault, "NumericError", "Vector.norm produced a non-finite value")

	x, y, z := []float64{1, 0, 0}, []float64{0, 1, 0}, []float64{0, 0, 1}
	if cross, _ := AhdVectorCross(x, y); !reflect.DeepEqual(cross, z) {
		t.Errorf("x × y = %v, want z (right-handed)", cross)
	}
	if cross, _ := AhdVectorCross(y, x); !reflect.DeepEqual(cross, []float64{0, 0, -1}) {
		t.Errorf("y × x = %v", cross)
	}
	if cross, _ := AhdVectorCross([]float64{2, 3, 4}, []float64{5, 6, 7}); !reflect.DeepEqual(cross, []float64{-3, 6, -3}) {
		t.Errorf("cross = %v", cross)
	}
	for _, pair := range [][2][]float64{{{1, 2}, {3, 4}}, {{1, 2, 3, 4}, {1, 2, 3, 4}}, {{1, 2, 3}, {1, 2}}, {nil, nil}} {
		_, fault := AhdVectorCross(pair[0], pair[1])
		requireFault(t, fault, "NumericError", "Vector.cross requires two vectors of length 3")
	}

	outer, fault := AhdVectorOuter([]float64{1, 2}, []float64{3, 4, 5})
	requireNoFault(t, fault)
	if !reflect.DeepEqual(outer, [][]float64{{3, 4, 5}, {6, 8, 10}}) {
		t.Errorf("outer = %v", outer)
	}
	_, fault = AhdVectorOuter(nil, []float64{1})
	requireFault(t, fault, "NumericError", "non-empty")
	_, fault = AhdVectorOuter([]float64{1e200}, []float64{1e200})
	requireFault(t, fault, "NumericError", "Vector.outer produced a non-finite value")
}

func TestMatrixMembers(t *testing.T) {
	rows := [][]float64{{1, 2, 3}, {4, 5, 6}}
	original := [][]float64{{1, 2, 3}, {4, 5, 6}}
	if value, _ := AhdMatrixAt(rows, 1, 2); value != 6 {
		t.Errorf("at(1, 2) = %v", value)
	}
	if value, _ := AhdMatrixAt(rows, -1, -3); value != 4 {
		t.Errorf("at(-1, -3) = %v", value)
	}
	_, fault := AhdMatrixAt(rows, 2, 0)
	requireFault(t, fault, "IndexError", "Matrix row index 2 is out of range for length 2")
	_, fault = AhdMatrixAt(rows, 0, -4)
	requireFault(t, fault, "IndexError", "Matrix column index -4 is out of range for length 3")

	row, _ := AhdMatrixRow(rows, -1)
	column, _ := AhdMatrixColumn(rows, 1)
	if !reflect.DeepEqual(row, []float64{4, 5, 6}) || !reflect.DeepEqual(column, []float64{2, 5}) {
		t.Errorf("row = %v, column = %v", row, column)
	}
	row[0] = 99 // a copy: the matrix is unchanged
	_, fault = AhdMatrixRow(rows, 2)
	requireFault(t, fault, "IndexError", "Matrix row index 2")
	_, fault = AhdMatrixColumn(rows, 3)
	requireFault(t, fault, "IndexError", "Matrix column index 3")

	if diagonal := AhdMatrixDiagonal(rows); !reflect.DeepEqual(diagonal, []float64{1, 5}) {
		t.Errorf("wide diagonal = %v", diagonal)
	}
	if diagonal := AhdMatrixDiagonal([][]float64{{1, 2}, {3, 4}, {5, 6}}); !reflect.DeepEqual(diagonal, []float64{1, 4}) {
		t.Errorf("tall diagonal = %v", diagonal)
	}
	if diagonal := AhdMatrixDiagonal([][]float64{{7}}); !reflect.DeepEqual(diagonal, []float64{7}) {
		t.Errorf("1x1 diagonal = %v", diagonal)
	}
	if norm, _ := AhdMatrixNorm(rows); !nearReal(norm, math.Sqrt(91)) {
		t.Errorf("Frobenius norm = %v", norm)
	}

	product, fault := AhdMatrixHadamard(rows, [][]float64{{2, 2, 2}, {0, -1, 0.5}})
	requireNoFault(t, fault)
	if !reflect.DeepEqual(product, [][]float64{{2, 4, 6}, {0, -5, 3}}) {
		t.Errorf("hadamard = %v", product)
	}
	for _, other := range [][][]float64{{{1, 2}, {3, 4}}, {{1, 2, 3}}, {{1, 2, 3}, {4, 5, 6}, {7, 8, 9}}} {
		_, fault = AhdMatrixHadamard(rows, other)
		requireFault(t, fault, "NumericError", "Matrix.hadamard requires matrices of the same shape")
	}

	vector, fault := AhdMatrixVector(rows, []float64{1, 0, -1})
	requireNoFault(t, fault)
	if !reflect.DeepEqual(vector, []float64{-2, -2}) {
		t.Errorf("matvec = %v", vector)
	}
	for _, other := range [][]float64{{1, 2}, {1, 2, 3, 4}, nil} {
		_, fault = AhdMatrixVector(rows, other)
		requireFault(t, fault, "NumericError", "Matrix.matvec requires a Vector of length 3")
	}
	if !reflect.DeepEqual(rows, original) {
		t.Fatalf("a Matrix member mutated its input: %v", rows)
	}
}

// ---- Statistics ----

func TestStatisticsCovarianceAndCorrelation(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{2, 4.1, 5.9, 8.2, 9.8}
	xCopy, yCopy := append([]float64(nil), x...), append([]float64(nil), y...)
	population, fault := AhdStatisticsCovariance(x, y, false)
	requireNoFault(t, fault)
	sample, fault := AhdStatisticsCovariance(x, y, true)
	requireNoFault(t, fault)
	if !nearReal(population, 3.94) || !nearReal(sample, 4.925) {
		t.Errorf("covariance = %v, sampleCovariance = %v", population, sample)
	}
	if single, fault := AhdStatisticsCovariance([]float64{5}, []float64{9}, false); fault != nil || single != 0 {
		t.Errorf("covariance of one pair = %v %v", single, fault)
	}
	// covariance(x, x) is the variance.
	if variance, _ := AhdStatisticsCovariance(x, x, false); !nearReal(variance, 2) {
		t.Errorf("covariance(x, x) = %v", variance)
	}
	correlation, fault := AhdStatisticsCorrelation(x, y)
	requireNoFault(t, fault)
	if math.Abs(correlation-0.99882964932988) > 1e-12 {
		t.Errorf("correlation = %v", correlation)
	}
	for _, testCase := range []struct {
		second   []float64
		expected float64
	}{
		{[]float64{10, 8, 6, 4, 2}, -1}, {[]float64{3, 5, 7, 9, 11}, 1}, {[]float64{0.1, 0.2, 0.3, 0.4, 0.5}, 1},
	} {
		result, fault := AhdStatisticsCorrelation(x, testCase.second)
		requireNoFault(t, fault)
		if math.Abs(result-testCase.expected) > 1e-12 || result > 1 || result < -1 {
			t.Errorf("correlation(x, %v) = %v", testCase.second, result)
		}
	}
	// Perfectly linear data whose rounding would overshoot is clamped to ±1.
	for scale := 1; scale < 200; scale++ {
		first, second := make([]float64, 7), make([]float64, 7)
		for index := range first {
			first[index] = 0.1 * float64(index+scale)
			second[index] = -0.7*first[index] + 1e-3
		}
		result, _ := AhdStatisticsCorrelation(first, second)
		if result < -1 || result > 1 {
			t.Fatalf("correlation escaped [-1, 1]: %v", result)
		}
	}
	if !reflect.DeepEqual(x, xCopy) || !reflect.DeepEqual(y, yCopy) {
		t.Fatal("a Statistics function mutated its input")
	}
}

func TestStatisticsTwoListErrors(t *testing.T) {
	_, fault := AhdStatisticsCovariance(nil, nil, false)
	requireFault(t, fault, "StatisticsError", "covariance is undefined for empty Lists")
	_, fault = AhdStatisticsCovariance([]float64{1, 2}, []float64{1}, false)
	requireFault(t, fault, "StatisticsError", "covariance requires Lists of the same length, got 2 and 1")
	_, fault = AhdStatisticsCovariance([]float64{1}, []float64{1}, true)
	requireFault(t, fault, "StatisticsError", "sampleCovariance requires at least two pairs of values")
	_, fault = AhdStatisticsCorrelation([]float64{1}, []float64{1})
	requireFault(t, fault, "StatisticsError", "correlation requires at least two pairs of values")
	_, fault = AhdStatisticsCorrelation(nil, nil)
	requireFault(t, fault, "StatisticsError", "correlation is undefined for empty Lists")
	for _, pair := range [][2][]float64{{{1, 2, 3}, {4, 4, 4}}, {{2, 2}, {1, 5}}, {{0.1, 0.1, 0.1}, {1, 2, 3}}} {
		_, fault = AhdStatisticsCorrelation(pair[0], pair[1])
		requireFault(t, fault, "StatisticsError", "correlation is undefined when either List has zero variance")
	}
	_, fault = AhdStatisticsCovariance([]float64{math.MaxFloat64, -math.MaxFloat64}, []float64{math.MaxFloat64, -math.MaxFloat64}, false)
	requireFault(t, fault, "StatisticsError", "covariance has no finite Real value")
}

func TestStatisticsLinearRegression(t *testing.T) {
	slope, intercept, fault := AhdStatisticsLinearRegression([]float64{1, 2, 3, 4, 5}, []float64{2, 4.1, 5.9, 8.2, 9.8})
	requireNoFault(t, fault)
	if !nearReal(slope, 1.97) || math.Abs(intercept-0.09) > 1e-12 {
		t.Errorf("fit = %v, %v", slope, intercept)
	}
	slope, intercept, _ = AhdStatisticsLinearRegression([]float64{0, 10}, []float64{-3, 17})
	if !nearReal(slope, 2) || !nearReal(intercept, -3) {
		t.Errorf("exact line = %v, %v", slope, intercept)
	}
	// A constant y is slope 0 and intercept exactly that constant.
	slope, intercept, _ = AhdStatisticsLinearRegression([]float64{1, 2, 3}, []float64{0.1, 0.1, 0.1})
	if slope != 0 || intercept != 0.1 {
		t.Errorf("constant y = %v, %v", slope, intercept)
	}
	// A large offset in x does not destroy the slope (two-pass centering).
	slope, intercept, _ = AhdStatisticsLinearRegression([]float64{1e9 + 1, 1e9 + 2, 1e9 + 3}, []float64{1, 2, 3})
	if !nearReal(slope, 1) || math.Abs(intercept+1e9) > 1e-6 {
		t.Errorf("offset x = %v, %v", slope, intercept)
	}
	_, _, fault = AhdStatisticsLinearRegression([]float64{2, 2, 2}, []float64{1, 2, 3})
	requireFault(t, fault, "StatisticsError", "linearRegression is undefined when x has zero variance")
	_, _, fault = AhdStatisticsLinearRegression([]float64{1}, []float64{1})
	requireFault(t, fault, "StatisticsError", "linearRegression requires at least two pairs of values")
	_, _, fault = AhdStatisticsLinearRegression([]float64{1, 2}, []float64{1, 2, 3})
	requireFault(t, fault, "StatisticsError", "same length")
	_, _, fault = AhdStatisticsLinearRegression(nil, nil)
	requireFault(t, fault, "StatisticsError", "empty")
}

// ---- Data ----

func TestDataConcat(t *testing.T) {
	columns := []string{"id", "name"}
	left := [][]string{{"1", "Ada"}, {"2", "Alan"}}
	right := [][]string{{"3", "Grace"}}
	resultColumns, rows, fault := AhdDataConcat(columns, left, columns, right)
	requireNoFault(t, fault)
	if !reflect.DeepEqual(resultColumns, columns) || !reflect.DeepEqual(rows, [][]string{{"1", "Ada"}, {"2", "Alan"}, {"3", "Grace"}}) {
		t.Fatalf("concat = %v %v", resultColumns, rows)
	}
	rows[0][0] = "changed"
	if left[0][0] != "1" {
		t.Fatal("concat shares rows with its receiver")
	}
	_, rows, _ = AhdDataConcat(columns, nil, columns, nil)
	if len(rows) != 0 {
		t.Fatalf("header-only concat = %v", rows)
	}
	_, rows, _ = AhdDataConcat(columns, nil, columns, right)
	if !reflect.DeepEqual(rows, right) {
		t.Fatalf("empty receiver concat = %v", rows)
	}
	for _, other := range [][]string{{"name", "id"}, {"id"}, {"id", "name", "extra"}, {"id", "Name"}, nil} {
		_, _, fault = AhdDataConcat(columns, left, other, nil)
		requireFault(t, fault, "DataError", "Table.concat requires the same columns in the same order")
	}
}

func TestDataInnerJoin(t *testing.T) {
	people := [][]string{{"1", "Ada"}, {"2", "Alan"}, {"", "Blank"}, {"3", "Grace"}, {"2", "Alan II"}}
	orders := [][]string{{"2", "book"}, {"1", "pen"}, {"2", "lamp"}, {"", "ghost"}, {"9", "none"}}
	columns, rows, fault := AhdDataInnerJoin([]string{"id", "name"}, people, []string{"id", "item"}, orders, "id", "id")
	requireNoFault(t, fault)
	if !reflect.DeepEqual(columns, []string{"id", "name", "item"}) {
		t.Fatalf("columns = %v", columns)
	}
	want := [][]string{
		{"1", "Ada", "pen"},
		{"2", "Alan", "book"}, {"2", "Alan", "lamp"},
		{"", "Blank", "ghost"},
		{"2", "Alan II", "book"}, {"2", "Alan II", "lamp"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %v", rows)
	}
	// Different key names: the right key column is dropped; right column
	// order is kept; matching is exact String equality.
	columns, rows, _ = AhdDataInnerJoin([]string{"id", "name"}, people, []string{"city", "person", "zip"},
		[][]string{{"London", "1", "E1"}, {"Paris", "01", "75"}, {"Rome", "1", "00100"}}, "id", "person")
	if !reflect.DeepEqual(columns, []string{"id", "name", "city", "zip"}) ||
		!reflect.DeepEqual(rows, [][]string{{"1", "Ada", "London", "E1"}, {"1", "Ada", "Rome", "00100"}}) {
		t.Fatalf("leftKey/rightKey join = %v %v", columns, rows)
	}
	// No match keeps the full schema.
	columns, rows, _ = AhdDataInnerJoin([]string{"id", "name"}, people, []string{"id", "x"}, nil, "id", "id")
	if !reflect.DeepEqual(columns, []string{"id", "name", "x"}) || len(rows) != 0 {
		t.Fatalf("empty join = %v %v", columns, rows)
	}
	_, _, fault = AhdDataInnerJoin([]string{"id", "name"}, people, []string{"id", "name"}, nil, "id", "id")
	requireFault(t, fault, "DataError", `Table.innerJoin column "name" exists in both Tables`)
	_, _, fault = AhdDataInnerJoin([]string{"id", "name"}, people, []string{"key", "id"}, nil, "id", "key")
	requireFault(t, fault, "DataError", `column "id" exists in both Tables`)
	_, _, fault = AhdDataInnerJoin([]string{"id"}, nil, []string{"id"}, nil, "missing", "id")
	requireFault(t, fault, "DataError", `left key column "missing" does not exist`)
	_, _, fault = AhdDataInnerJoin([]string{"id"}, nil, []string{"id"}, nil, "id", "missing")
	requireFault(t, fault, "DataError", `right key column "missing" does not exist`)
	if people[0][0] != "1" || orders[0][1] != "book" {
		t.Fatal("innerJoin mutated an input")
	}
}

// TestDataInnerJoinUsesAKeyedIndex checks the join scales with the number of
// rows and matches, not with their product: 20,000 × 20,000 rows with one match
// each would be 4e8 comparisons for a nested scan.
func TestDataInnerJoinUsesAKeyedIndex(t *testing.T) {
	const size = 20000
	left, right := make([][]string, size), make([][]string, size)
	for index := range left {
		key := string(rune('a'+index%26)) + string(rune('A'+index/26%26)) + string(rune('0'+index/676))
		left[index] = []string{key, "L"}
		right[size-1-index] = []string{key, "R"}
	}
	start := time.Now()
	_, rows, fault := AhdDataInnerJoin([]string{"k", "l"}, left, []string{"k", "r"}, right, "k", "k")
	requireNoFault(t, fault)
	if len(rows) != size {
		t.Fatalf("rows = %d", len(rows))
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("join of %d rows took %v", size, elapsed)
	}
}
