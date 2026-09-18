package ahdruntime

import (
	"math"
	"math/bits"
	"strconv"
	"time"
)

// This file holds the v1.7 standard-library fundamentals shared by compiled
// programs and the evaluator: the added Math functions, strict ISO 8601 text
// and instant arithmetic for Time, Vector and Matrix accessors, two-list
// Statistics, and Table concatenation and joins. Every helper is a pure
// function that returns its result or an AhdFault, so the evaluator raises
// exactly the class and message a compiled program raises. It uses only the
// standard library.

// AhdFault is a failure computed by a shared helper: the AhdCode error class
// name and the message. A nil fault means success.
type AhdFault struct {
	Class   string
	Message string
}

func ahdFault(class, message string) *AhdFault { return &AhdFault{Class: class, Message: message} }

// AhdRaiseFault raises fault in a compiled program; nil is ignored.
func AhdRaiseFault(fault *AhdFault) {
	if fault == nil {
		return
	}
	class := AhdClassValueError
	switch fault.Class {
	case "DomainError":
		class = AhdClassDomainError
	case "OverflowError":
		class = AhdClassOverflowError
	case "IndexError":
		class = AhdClassIndexError
	case "NumericError":
		class = AhdClassNumericError
	case "StatisticsError":
		class = AhdClassStatisticsError
	case "DataError":
		class = AhdClassDataError
	case "SecurityError":
		class = AhdClassSecurityError
	}
	AhdRaiseClass(class, fault.Message)
}

// ---- Math ----

// AhdMathUnary evaluates one of the v1.7 single-argument Math functions.
// Inputs outside a function's domain raise DomainError; a result that is not a
// finite Real raises DomainError (undefined) or OverflowError (too large), so
// NaN and infinity never reach a program.
func AhdMathUnary(name string, value float64) (float64, *AhdFault) {
	if fault := ahdMathFinite(name, value, "received"); fault != nil {
		return 0, fault
	}
	var result float64
	switch name {
	case "asin", "acos":
		if value < -1 || value > 1 {
			return 0, ahdFault("DomainError", "Math."+name+" requires a value between -1 and 1")
		}
		if name == "asin" {
			result = math.Asin(value)
		} else {
			result = math.Acos(value)
		}
	case "atan":
		result = math.Atan(value)
	case "sinh":
		result = math.Sinh(value)
	case "cosh":
		result = math.Cosh(value)
	case "tanh":
		result = math.Tanh(value)
	case "log2":
		if value <= 0 {
			return 0, ahdFault("DomainError", "Math.log2 requires a value greater than zero")
		}
		result = math.Log2(value)
	case "cbrt":
		result = math.Cbrt(value)
	case "radians":
		result = value * math.Pi / 180
	case "degrees":
		result = value * 180 / math.Pi
	default:
		return 0, ahdFault("ValueError", "unsupported Math function "+name)
	}
	return result, ahdMathFinite(name, result, "produced")
}

// AhdMathBinary evaluates Math.atan2(y, x) or Math.hypot(x, y).
func AhdMathBinary(name string, first, second float64) (float64, *AhdFault) {
	if fault := ahdMathFinite(name, first, "received"); fault != nil {
		return 0, fault
	}
	if fault := ahdMathFinite(name, second, "received"); fault != nil {
		return 0, fault
	}
	var result float64
	switch name {
	case "atan2":
		result = math.Atan2(first, second)
	case "hypot":
		result = math.Hypot(first, second)
	default:
		return 0, ahdFault("ValueError", "unsupported Math function "+name)
	}
	return result, ahdMathFinite(name, result, "produced")
}

func ahdMathFinite(name string, value float64, verb string) *AhdFault {
	if math.IsNaN(value) {
		if verb == "received" {
			return ahdFault("DomainError", "Math."+name+" received NaN")
		}
		return ahdFault("DomainError", "Math."+name+" produced an undefined result")
	}
	if math.IsInf(value, 0) {
		if verb == "received" {
			return ahdFault("OverflowError", "Math."+name+" received a non-finite value")
		}
		return ahdFault("OverflowError", "Math."+name+" result exceeds finite Real range")
	}
	return nil
}

// ahdMagnitude is |value| as an unsigned number, exact even for the minimum
// Int, whose magnitude does not fit Int.
func ahdMagnitude(value int64) uint64 {
	if value < 0 {
		return uint64(-(value + 1)) + 1
	}
	return uint64(value)
}

func ahdGCDMagnitude(first, second uint64) uint64 {
	for second != 0 {
		first, second = second, first%second
	}
	return first
}

func ahdMagnitudeInt(name string, value uint64) (int64, *AhdFault) {
	if value > math.MaxInt64 {
		return 0, ahdFault("OverflowError", "Math."+name+" result does not fit Int")
	}
	return int64(value), nil
}

// AhdMathGCD is the non-negative greatest common divisor; gcd(0, 0) is 0.
func AhdMathGCD(first, second int64) (int64, *AhdFault) {
	return ahdMagnitudeInt("gcd", ahdGCDMagnitude(ahdMagnitude(first), ahdMagnitude(second)))
}

// AhdMathLCM is the non-negative least common multiple, |(a / gcd) * b| with
// checked arithmetic; it is 0 when either argument is 0.
func AhdMathLCM(first, second int64) (int64, *AhdFault) {
	if first == 0 || second == 0 {
		return 0, nil
	}
	a, b := ahdMagnitude(first), ahdMagnitude(second)
	high, low := bits.Mul64(a/ahdGCDMagnitude(a, b), b)
	if high != 0 {
		return 0, ahdFault("OverflowError", "Math.lcm result does not fit Int")
	}
	return ahdMagnitudeInt("lcm", low)
}

// ---- Time ----

func ahdISODigits(text string, start, count int) (int, bool) {
	if start+count > len(text) {
		return 0, false
	}
	value := 0
	for index := start; index < start+count; index++ {
		character := text[index]
		if character < '0' || character > '9' {
			return 0, false
		}
		value = value*10 + int(character-'0')
	}
	return value, true
}

func ahdISOInvalid(text string) *AhdFault {
	if len(text) > 64 {
		text = text[:64] + "..."
	}
	return ahdFault("ValueError", "Time.parseISO expects YYYY-MM-DDTHH:MM:SS[.fff] followed by Z or ±HH:MM, got "+strconv.Quote(text))
}

// AhdTimeParseISO parses the strict RFC 3339 subset AhdCode accepts:
// YYYY-MM-DDTHH:MM:SS, an optional fraction of one to three digits, and a
// required Z or ±HH:MM designator. Anything else, and any impossible civil
// value, raises ValueError. The result keeps the written offset.
func AhdTimeParseISO(text string) (time.Time, *AhdFault) {
	if len(text) < 20 || text[4] != '-' || text[7] != '-' || text[10] != 'T' || text[13] != ':' || text[16] != ':' {
		return time.Time{}, ahdISOInvalid(text)
	}
	fields := [6]int{}
	for index, position := range [6]int{0, 5, 8, 11, 14, 17} {
		width := 2
		if index == 0 {
			width = 4
		}
		value, ok := ahdISODigits(text, position, width)
		if !ok {
			return time.Time{}, ahdISOInvalid(text)
		}
		fields[index] = value
	}
	position, millisecond := 19, 0
	if text[position] == '.' {
		digits := 0
		for position+1+digits < len(text) && text[position+1+digits] >= '0' && text[position+1+digits] <= '9' {
			digits++
		}
		if digits < 1 || digits > 3 {
			return time.Time{}, ahdISOInvalid(text)
		}
		millisecond, _ = ahdISODigits(text, position+1, digits)
		for scale := digits; scale < 3; scale++ {
			millisecond *= 10
		}
		position += 1 + digits
	}
	offset := 0
	switch {
	case position == len(text)-1 && text[position] == 'Z':
	case position == len(text)-6 && (text[position] == '+' || text[position] == '-') && text[position+3] == ':':
		hours, okHours := ahdISODigits(text, position+1, 2)
		minutes, okMinutes := ahdISODigits(text, position+4, 2)
		if !okHours || !okMinutes || minutes > 59 {
			return time.Time{}, ahdISOInvalid(text)
		}
		offset = hours*60 + minutes
		if offset > 840 {
			return time.Time{}, ahdFault("ValueError", "Time.parseISO offset "+text[position:]+" is outside -14:00..+14:00")
		}
		if text[position] == '-' {
			offset = -offset
		}
	default:
		return time.Time{}, ahdISOInvalid(text)
	}
	year, month, day, hour, minute, second := fields[0], fields[1], fields[2], fields[3], fields[4], fields[5]
	switch {
	case year < 1:
		return time.Time{}, ahdFault("ValueError", "Time.parseISO year 0000 is outside 1..9999")
	case month < 1 || month > 12:
		return time.Time{}, ahdFault("ValueError", "Time.parseISO month "+strconv.Itoa(month)+" is outside 1..12")
	case day < 1 || int64(day) > AhdCalendarDaysInMonth(int64(year), int64(month)):
		return time.Time{}, ahdFault("ValueError", "Time.parseISO day "+strconv.Itoa(day)+" does not exist in "+text[:7])
	case hour > 23:
		return time.Time{}, ahdFault("ValueError", "Time.parseISO hour "+strconv.Itoa(hour)+" is outside 0..23")
	case minute > 59:
		return time.Time{}, ahdFault("ValueError", "Time.parseISO minute "+strconv.Itoa(minute)+" is outside 0..59")
	case second > 59:
		return time.Time{}, ahdFault("ValueError", "Time.parseISO second "+strconv.Itoa(second)+" is outside 0..59")
	}
	return time.Date(year, time.Month(month), day, hour, minute, second, millisecond*1e6, time.FixedZone("", offset*60)), nil
}

// AhdTimeFormatISO writes value as YYYY-MM-DDTHH:MM:SS.mmm followed by Z for a
// zero offset or ±HH:MM otherwise. A historical offset with a seconds part
// cannot be written in this form and raises ValueError.
func AhdTimeFormatISO(value time.Time) (string, *AhdFault) {
	_, offset := value.Zone()
	if offset%60 != 0 {
		return "", ahdFault("ValueError", "DateTime.toISO cannot write a UTC offset with a seconds part; convert with toUTC() first")
	}
	text := ahdPad(int64(value.Year()), 4) + "-" + ahdPad(int64(value.Month()), 2) + "-" + ahdPad(int64(value.Day()), 2) + "T" +
		ahdPad(int64(value.Hour()), 2) + ":" + ahdPad(int64(value.Minute()), 2) + ":" + ahdPad(int64(value.Second()), 2) + "." +
		ahdPad(int64(value.Nanosecond()/1e6), 3)
	if offset == 0 {
		return text + "Z", nil
	}
	sign := "+"
	if offset < 0 {
		sign, offset = "-", -offset
	}
	return text + sign + ahdPad(int64(offset/3600), 2) + ":" + ahdPad(int64(offset%3600/60), 2), nil
}

// AhdTimeShift moves value by milliseconds (backwards when subtract is true).
// It is instant arithmetic: the result keeps value's UTC offset, so a fixed
// offset stays fixed and UTC stays UTC. Leaving years 1..9999 raises
// ValueError.
func AhdTimeShift(value time.Time, milliseconds int64, subtract bool) (time.Time, *AhdFault) {
	name := "add"
	if subtract {
		name = "subtract"
	}
	start := value.UnixMilli()
	var result int64
	var ok bool
	if subtract {
		result, ok = ahdCheckedSubtract(start, milliseconds)
	} else {
		result, ok = ahdCheckedAdd(start, milliseconds)
	}
	outside := ahdFault("ValueError", "DateTime."+name+" result is outside the supported DateTime range")
	// 400,000 years of milliseconds bounds every representable DateTime with a
	// wide margin while keeping the conversion below well inside time.Time.
	const limit = 400000 * 366 * 24 * 3600 * 1000
	if !ok || result < -limit || result > limit {
		return time.Time{}, outside
	}
	shifted := time.UnixMilli(result).In(value.Location())
	if shifted.Year() < 1 || shifted.Year() > 9999 {
		return time.Time{}, outside
	}
	return shifted, nil
}

func ahdCheckedAdd(left, right int64) (int64, bool) {
	sum := left + right
	return sum, (sum > left) == (right > 0)
}

func ahdCheckedSubtract(left, right int64) (int64, bool) {
	difference := left - right
	return difference, (difference < left) == (right > 0)
}

// AhdDurationArithmetic implements Duration.add, subtract, negate, and abs
// over Int milliseconds; a result outside Int raises OverflowError.
func AhdDurationArithmetic(name string, first, second int64) (int64, *AhdFault) {
	var result int64
	ok := true
	switch name {
	case "add":
		result, ok = ahdCheckedAdd(first, second)
	case "subtract":
		result, ok = ahdCheckedSubtract(first, second)
	case "negate":
		result, ok = -first, first != math.MinInt64
	case "abs":
		result, ok = first, first != math.MinInt64
		if first < 0 {
			result = -first
		}
	}
	if !ok {
		return 0, ahdFault("OverflowError", "Duration."+name+" overflowed Int milliseconds")
	}
	return result, nil
}

// ---- Numeric ----

func ahdNumericIndex(index int64, length int, what string) (int, *AhdFault) {
	position := index
	if position < 0 {
		position += int64(length)
	}
	if position < 0 || position >= int64(length) {
		return 0, ahdFault("IndexError", what+" index "+strconv.FormatInt(index, 10)+" is out of range for length "+strconv.Itoa(length))
	}
	return int(position), nil
}

func ahdNumericFinite(operation string, values ...float64) *AhdFault {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return ahdFault("NumericError", operation+" produced a non-finite value")
		}
	}
	return nil
}

// AhdVectorAt reads one component; a negative index counts from the end.
func AhdVectorAt(values []float64, index int64) (float64, *AhdFault) {
	position, fault := ahdNumericIndex(index, len(values), "Vector")
	if fault != nil {
		return 0, fault
	}
	return values[position], nil
}

// ahdNumericL2 is the Euclidean length, scaled by the largest magnitude so an
// intermediate square cannot overflow or underflow for a representable result.
func ahdNumericL2(values []float64) float64 {
	scale := float64(0)
	for _, value := range values {
		scale = math.Max(scale, math.Abs(value))
	}
	if scale == 0 {
		return 0
	}
	total := float64(0)
	for _, value := range values {
		ratio := value / scale
		total += ratio * ratio
	}
	return scale * math.Sqrt(total)
}

// AhdVectorNorm is the Euclidean (L2) norm; the empty Vector has norm 0.
func AhdVectorNorm(values []float64) (float64, *AhdFault) {
	result := ahdNumericL2(values)
	return result, ahdNumericFinite("Vector.norm", result)
}

// AhdVectorOuter is the len(a)-by-len(b) outer product a bᵀ.
func AhdVectorOuter(first, second []float64) ([][]float64, *AhdFault) {
	if len(first) == 0 || len(second) == 0 {
		return nil, ahdFault("NumericError", "Vector.outer requires non-empty vectors")
	}
	rows := make([][]float64, len(first))
	for i, left := range first {
		rows[i] = make([]float64, len(second))
		for j, right := range second {
			rows[i][j] = left * right
		}
		if fault := ahdNumericFinite("Vector.outer", rows[i]...); fault != nil {
			return nil, fault
		}
	}
	return rows, nil
}

// AhdVectorCross is the right-handed cross product of two 3-D vectors.
func AhdVectorCross(first, second []float64) ([]float64, *AhdFault) {
	if len(first) != 3 || len(second) != 3 {
		return nil, ahdFault("NumericError", "Vector.cross requires two vectors of length 3, got "+strconv.Itoa(len(first))+" and "+strconv.Itoa(len(second)))
	}
	result := []float64{
		first[1]*second[2] - first[2]*second[1],
		first[2]*second[0] - first[0]*second[2],
		first[0]*second[1] - first[1]*second[0],
	}
	return result, ahdNumericFinite("Vector.cross", result...)
}

// AhdMatrixAt reads one entry; negative indices count from the end.
func AhdMatrixAt(rows [][]float64, row, column int64) (float64, *AhdFault) {
	r, fault := ahdNumericIndex(row, len(rows), "Matrix row")
	if fault != nil {
		return 0, fault
	}
	c, fault := ahdNumericIndex(column, len(rows[r]), "Matrix column")
	if fault != nil {
		return 0, fault
	}
	return rows[r][c], nil
}

// AhdMatrixRow copies one row as a Vector.
func AhdMatrixRow(rows [][]float64, index int64) ([]float64, *AhdFault) {
	r, fault := ahdNumericIndex(index, len(rows), "Matrix row")
	if fault != nil {
		return nil, fault
	}
	return append([]float64(nil), rows[r]...), nil
}

// AhdMatrixColumn copies one column as a Vector.
func AhdMatrixColumn(rows [][]float64, index int64) ([]float64, *AhdFault) {
	width := 0
	if len(rows) > 0 {
		width = len(rows[0])
	}
	c, fault := ahdNumericIndex(index, width, "Matrix column")
	if fault != nil {
		return nil, fault
	}
	result := make([]float64, len(rows))
	for i, row := range rows {
		result[i] = row[c]
	}
	return result, nil
}

// AhdMatrixDiagonal is the main diagonal, of length min(rows, columns).
func AhdMatrixDiagonal(rows [][]float64) []float64 {
	result := []float64{}
	for i := 0; i < len(rows) && i < len(rows[i]); i++ {
		result = append(result, rows[i][i])
	}
	return result
}

// AhdMatrixNorm is the Frobenius norm.
func AhdMatrixNorm(rows [][]float64) (float64, *AhdFault) {
	all := []float64{}
	for _, row := range rows {
		all = append(all, row...)
	}
	result := ahdNumericL2(all)
	return result, ahdNumericFinite("Matrix.norm", result)
}

// AhdMatrixHadamard is the element-wise product of two same-shape matrices.
func AhdMatrixHadamard(first, second [][]float64) ([][]float64, *AhdFault) {
	if len(first) != len(second) || len(first) > 0 && len(first[0]) != len(second[0]) {
		return nil, ahdFault("NumericError", "Matrix.hadamard requires matrices of the same shape")
	}
	result := make([][]float64, len(first))
	for i := range first {
		result[i] = make([]float64, len(first[i]))
		for j := range first[i] {
			result[i][j] = first[i][j] * second[i][j]
		}
		if fault := ahdNumericFinite("Matrix.hadamard", result[i]...); fault != nil {
			return nil, fault
		}
	}
	return result, nil
}

// AhdMatrixVector multiplies an m-by-n matrix by a length-n Vector.
func AhdMatrixVector(rows [][]float64, vector []float64) ([]float64, *AhdFault) {
	if len(rows) == 0 || len(rows[0]) != len(vector) {
		width := 0
		if len(rows) > 0 {
			width = len(rows[0])
		}
		return nil, ahdFault("NumericError", "Matrix.matvec requires a Vector of length "+strconv.Itoa(width)+", got "+strconv.Itoa(len(vector)))
	}
	result := make([]float64, len(rows))
	for i, row := range rows {
		total := float64(0)
		for j, value := range row {
			total += value * vector[j]
		}
		result[i] = total
	}
	return result, ahdNumericFinite("Matrix.matvec", result...)
}

// ---- Statistics ----

func ahdStatisticsPair(name string, first, second []float64, minimum int) *AhdFault {
	if len(first) != len(second) {
		return ahdFault("StatisticsError", name+" requires Lists of the same length, got "+strconv.Itoa(len(first))+" and "+strconv.Itoa(len(second)))
	}
	if len(first) == 0 {
		return ahdFault("StatisticsError", name+" is undefined for empty Lists")
	}
	if len(first) < minimum {
		return ahdFault("StatisticsError", name+" requires at least two pairs of values")
	}
	return nil
}

func ahdStatisticsConstant(values []float64) bool {
	for _, value := range values {
		if value != values[0] {
			return false
		}
	}
	return true
}

func ahdStatisticsMeanOf(values []float64) float64 {
	total := float64(0)
	for _, value := range values {
		total += value
	}
	mean := total / float64(len(values))
	// A second pass removes the rounding error of the first sum.
	correction := float64(0)
	for _, value := range values {
		correction += value - mean
	}
	return mean + correction/float64(len(values))
}

// ahdStatisticsMoments returns the two-pass centered sums Σdx², Σdy², Σdxdy.
func ahdStatisticsMoments(first, second []float64) (meanX, meanY, sxx, syy, sxy float64) {
	meanX, meanY = ahdStatisticsMeanOf(first), ahdStatisticsMeanOf(second)
	for i := range first {
		dx, dy := first[i]-meanX, second[i]-meanY
		sxx += dx * dx
		syy += dy * dy
		sxy += dx * dy
	}
	return
}

func ahdStatisticsResult(name string, value float64) (float64, *AhdFault) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, ahdFault("StatisticsError", name+" has no finite Real value for this input")
	}
	return value, nil
}

// AhdStatisticsCovariance is the population (sample false) or sample
// covariance of two equal-length Lists.
func AhdStatisticsCovariance(first, second []float64, sample bool) (float64, *AhdFault) {
	name, minimum := "covariance", 1
	if sample {
		name, minimum = "sampleCovariance", 2
	}
	if fault := ahdStatisticsPair(name, first, second, minimum); fault != nil {
		return 0, fault
	}
	_, _, _, _, sxy := ahdStatisticsMoments(first, second)
	divisor := float64(len(first))
	if sample {
		divisor--
	}
	return ahdStatisticsResult(name, sxy/divisor)
}

// AhdStatisticsCorrelation is Pearson's correlation coefficient, in [-1, 1].
func AhdStatisticsCorrelation(first, second []float64) (float64, *AhdFault) {
	if fault := ahdStatisticsPair("correlation", first, second, 2); fault != nil {
		return 0, fault
	}
	if ahdStatisticsConstant(first) || ahdStatisticsConstant(second) {
		return 0, ahdFault("StatisticsError", "correlation is undefined when either List has zero variance")
	}
	_, _, sxx, syy, sxy := ahdStatisticsMoments(first, second)
	if sxx == 0 || syy == 0 {
		return 0, ahdFault("StatisticsError", "correlation is undefined when either List has zero variance")
	}
	result, fault := ahdStatisticsResult("correlation", sxy/(math.Sqrt(sxx)*math.Sqrt(syy)))
	// Rounding can overshoot ±1 by an ulp for perfectly linear data; the
	// coefficient is clamped to its mathematical range.
	return math.Max(-1, math.Min(1, result)), fault
}

// AhdStatisticsLinearRegression fits y = slope * x + intercept by ordinary
// least squares and returns slope, intercept.
func AhdStatisticsLinearRegression(x, y []float64) (float64, float64, *AhdFault) {
	if fault := ahdStatisticsPair("linearRegression", x, y, 2); fault != nil {
		return 0, 0, fault
	}
	if ahdStatisticsConstant(x) {
		return 0, 0, ahdFault("StatisticsError", "linearRegression is undefined when x has zero variance")
	}
	if ahdStatisticsConstant(y) {
		return 0, y[0], nil
	}
	meanX, meanY, sxx, _, sxy := ahdStatisticsMoments(x, y)
	if sxx == 0 {
		return 0, 0, ahdFault("StatisticsError", "linearRegression is undefined when x has zero variance")
	}
	slope, fault := ahdStatisticsResult("linearRegression", sxy/sxx)
	if fault != nil {
		return 0, 0, fault
	}
	intercept, fault := ahdStatisticsResult("linearRegression", meanY-slope*meanX)
	return slope, intercept, fault
}

// ---- Data ----

func ahdDataSameColumns(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for i := range first {
		if first[i] != second[i] {
			return false
		}
	}
	return true
}

func ahdDataCopyRows(rows [][]string) [][]string {
	result := make([][]string, len(rows))
	for i, row := range rows {
		result[i] = append([]string(nil), row...)
	}
	return result
}

// AhdDataConcat appends other's rows after the receiver's. Both Tables must
// have exactly the same column names in the same order.
func AhdDataConcat(columns []string, rows [][]string, otherColumns []string, otherRows [][]string) ([]string, [][]string, *AhdFault) {
	if !ahdDataSameColumns(columns, otherColumns) {
		return nil, nil, ahdFault("DataError", "Table.concat requires the same columns in the same order")
	}
	result := ahdDataCopyRows(rows)
	result = append(result, ahdDataCopyRows(otherRows)...)
	return append([]string(nil), columns...), result, nil
}

func ahdDataColumnIndex(columns []string, name, side string) (int, *AhdFault) {
	for i, column := range columns {
		if column == name {
			return i, nil
		}
	}
	return 0, ahdFault("DataError", "Table.innerJoin "+side+" key column "+strconv.Quote(name)+" does not exist")
}

// AhdDataInnerJoin joins rows whose leftKey and rightKey cells are equal
// Strings. The result has every left column followed by the right columns
// except rightKey; for each left row in order it emits the matching right rows
// in their order. A right column whose name is already a left column raises
// DataError instead of being renamed.
func AhdDataInnerJoin(leftColumns []string, leftRows [][]string, rightColumns []string, rightRows [][]string, leftKey, rightKey string) ([]string, [][]string, *AhdFault) {
	leftIndex, fault := ahdDataColumnIndex(leftColumns, leftKey, "left")
	if fault != nil {
		return nil, nil, fault
	}
	rightIndex, fault := ahdDataColumnIndex(rightColumns, rightKey, "right")
	if fault != nil {
		return nil, nil, fault
	}
	leftNames := make(map[string]bool, len(leftColumns))
	for _, column := range leftColumns {
		leftNames[column] = true
	}
	columns := append([]string(nil), leftColumns...)
	kept := []int{}
	for i, column := range rightColumns {
		if i == rightIndex {
			continue
		}
		if leftNames[column] {
			return nil, nil, ahdFault("DataError", "Table.innerJoin column "+strconv.Quote(column)+" exists in both Tables")
		}
		columns = append(columns, column)
		kept = append(kept, i)
	}
	matches := make(map[string][]int, len(rightRows))
	for i, row := range rightRows {
		matches[row[rightIndex]] = append(matches[row[rightIndex]], i)
	}
	result := [][]string{}
	for _, left := range leftRows {
		for _, match := range matches[left[leftIndex]] {
			row := make([]string, 0, len(columns))
			row = append(row, left...)
			for _, column := range kept {
				row = append(row, rightRows[match][column])
			}
			result = append(result, row)
		}
	}
	return columns, result, nil
}

// The Checked wrappers raise a helper's fault in a compiled program.

func AhdMathUnaryChecked(name string, value float64) float64 {
	result, fault := AhdMathUnary(name, value)
	AhdRaiseFault(fault)
	return result
}

func AhdMathBinaryChecked(name string, first, second float64) float64 {
	result, fault := AhdMathBinary(name, first, second)
	AhdRaiseFault(fault)
	return result
}

func AhdMathGCDChecked(first, second int64) int64 {
	result, fault := AhdMathGCD(first, second)
	AhdRaiseFault(fault)
	return result
}

func AhdMathLCMChecked(first, second int64) int64 {
	result, fault := AhdMathLCM(first, second)
	AhdRaiseFault(fault)
	return result
}

func AhdTimeParseISOChecked(text string) AhdCivilTime {
	value, fault := AhdTimeParseISO(text)
	AhdRaiseFault(fault)
	return ahdCivilFrom(value)
}

func AhdTimeFormatISOChecked(value time.Time) string {
	text, fault := AhdTimeFormatISO(value)
	AhdRaiseFault(fault)
	return text
}

func AhdTimeShiftChecked(value time.Time, milliseconds int64, subtract bool) AhdCivilTime {
	shifted, fault := AhdTimeShift(value, milliseconds, subtract)
	AhdRaiseFault(fault)
	return ahdCivilFrom(shifted)
}

func AhdDurationArithmeticChecked(name string, first, second int64) int64 {
	result, fault := AhdDurationArithmetic(name, first, second)
	AhdRaiseFault(fault)
	return result
}

func AhdVectorAtChecked(vector AhdVector, index int64) float64 {
	result, fault := AhdVectorAt(ahdNumericValues(vector), index)
	AhdRaiseFault(fault)
	return result
}

func AhdVectorNormChecked(vector AhdVector) float64 {
	result, fault := AhdVectorNorm(ahdNumericValues(vector))
	AhdRaiseFault(fault)
	return result
}

func AhdVectorOuterChecked(first, second AhdVector) AhdMatrix {
	rows, fault := AhdVectorOuter(ahdNumericValues(first), ahdNumericValues(second))
	AhdRaiseFault(fault)
	return ahdNumericMatrix(rows)
}

func AhdVectorCrossChecked(first, second AhdVector) AhdVector {
	values, fault := AhdVectorCross(ahdNumericValues(first), ahdNumericValues(second))
	AhdRaiseFault(fault)
	return ahdNumericVector(values)
}

func AhdMatrixAtChecked(matrix AhdMatrix, row, column int64) float64 {
	result, fault := AhdMatrixAt(ahdNumericRows(matrix), row, column)
	AhdRaiseFault(fault)
	return result
}

func AhdMatrixRowChecked(matrix AhdMatrix, index int64) AhdVector {
	values, fault := AhdMatrixRow(ahdNumericRows(matrix), index)
	AhdRaiseFault(fault)
	return ahdNumericVector(values)
}

func AhdMatrixColumnChecked(matrix AhdMatrix, index int64) AhdVector {
	values, fault := AhdMatrixColumn(ahdNumericRows(matrix), index)
	AhdRaiseFault(fault)
	return ahdNumericVector(values)
}

func AhdMatrixDiagonalChecked(matrix AhdMatrix) AhdVector {
	return ahdNumericVector(AhdMatrixDiagonal(ahdNumericRows(matrix)))
}

func AhdMatrixNormChecked(matrix AhdMatrix) float64 {
	result, fault := AhdMatrixNorm(ahdNumericRows(matrix))
	AhdRaiseFault(fault)
	return result
}

func AhdMatrixHadamardChecked(first, second AhdMatrix) AhdMatrix {
	rows, fault := AhdMatrixHadamard(ahdNumericRows(first), ahdNumericRows(second))
	AhdRaiseFault(fault)
	return ahdNumericMatrix(rows)
}

func AhdMatrixVectorChecked(matrix AhdMatrix, vector AhdVector) AhdVector {
	values, fault := AhdMatrixVector(ahdNumericRows(matrix), ahdNumericValues(vector))
	AhdRaiseFault(fault)
	return ahdNumericVector(values)
}

// AhdStatisticsWiden snapshots an Int or Real List as Real values.
func AhdStatisticsWiden[T int64 | float64](list *AhdList[T]) []float64 {
	if list == nil {
		AhdRaiseClass(AhdClassNullError, "List value is null")
	}
	source := list.Snapshot()
	values := make([]float64, len(source))
	for i, value := range source {
		values[i] = float64(value)
	}
	return values
}

func AhdStatisticsCovarianceChecked(first, second []float64, sample bool) float64 {
	result, fault := AhdStatisticsCovariance(first, second, sample)
	AhdRaiseFault(fault)
	return result
}

func AhdStatisticsCorrelationChecked(first, second []float64) float64 {
	result, fault := AhdStatisticsCorrelation(first, second)
	AhdRaiseFault(fault)
	return result
}

func AhdStatisticsLinearRegressionChecked(x, y []float64) *AhdPair[string, float64] {
	slope, intercept, fault := AhdStatisticsLinearRegression(x, y)
	AhdRaiseFault(fault)
	result := AhdNewPair[string, float64]()
	result.Set("slope", slope)
	result.Set("intercept", intercept)
	return result
}

func AhdDataConcatChecked(first, second AhdTable) AhdTable {
	leftColumns, leftRows := ahdTableOf(first)
	rightColumns, rightRows := ahdTableOf(second)
	columns, rows, fault := AhdDataConcat(leftColumns, leftRows, rightColumns, rightRows)
	AhdRaiseFault(fault)
	return ahdTableValue(columns, rows)
}

func AhdDataInnerJoinChecked(first, second AhdTable, leftKey, rightKey string) AhdTable {
	leftColumns, leftRows := ahdTableOf(first)
	rightColumns, rightRows := ahdTableOf(second)
	columns, rows, fault := AhdDataInnerJoin(leftColumns, leftRows, rightColumns, rightRows, leftKey, rightKey)
	AhdRaiseFault(fault)
	return ahdTableValue(columns, rows)
}
