package evaluator

import (
	"math"
	"sort"
)

// The Statistics standard module's REPL implementation. It mirrors the native
// runtime helper for helper: the same snapshots, the same definitions, the same
// undefined-input errors, so a statistic cannot mean one thing compiled and
// another interactively.

// statisticsNumbers snapshots a numeric List as float64 for the averaging
// statistics, and reports whether every element was an Int.
func (session *Session) statisticsNumbers(value any) ([]float64, bool) {
	list := session.requireList(value)
	numbers := make([]float64, len(list.Items))
	integral := true
	for index, item := range list.Items {
		switch number := item.(type) {
		case int64:
			numbers[index] = float64(number)
		case float64:
			numbers[index] = number
			integral = false
		default:
			session.raise("NullError", "numeric List element is null")
		}
	}
	return numbers, integral
}

func (session *Session) statisticsRequire(count int, statistic string) {
	if count == 0 {
		session.raise("StatisticsError", statistic+" is undefined for an empty List")
	}
}

func (session *Session) statisticsFinite(value float64, statistic string) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		session.raise("StatisticsError", statistic+" has no finite Real value for this input")
	}
	return value
}

// statisticsBuiltin dispatches one Statistics function. The frontend already
// selected the Int or Real overload, so the element kind of the snapshot
// decides only whether an order statistic is returned as Int or Real.
func (session *Session) statisticsBuiltin(name string, arguments []any) any {
	numbers, integral := session.statisticsNumbers(arguments[0])
	// An order statistic returns one of the input's own values, so an Int input
	// keeps Int; an averaging statistic is always Real.
	element := func(value float64) any {
		if integral {
			return int64(value)
		}
		return value
	}
	switch name {
	case "sum":
		total := float64(0)
		for _, value := range numbers {
			total += value
		}
		if integral {
			return session.statisticsIntSum(arguments[0])
		}
		return total
	case "min", "max", "range":
		session.statisticsRequire(len(numbers), name)
		smallest, largest := numbers[0], numbers[0]
		for _, value := range numbers {
			smallest = math.Min(smallest, value)
			largest = math.Max(largest, value)
		}
		switch name {
		case "min":
			return element(smallest)
		case "max":
			return element(largest)
		default:
			return element(largest - smallest)
		}
	case "mean":
		session.statisticsRequire(len(numbers), name)
		return session.statisticsFinite(statisticsMean(numbers), name)
	case "median":
		session.statisticsRequire(len(numbers), name)
		return session.statisticsFinite(statisticsMedian(numbers), name)
	case "variance", "sampleVariance", "stdDev", "sampleStdDev":
		return session.statisticsSpread(name, numbers)
	case "mode":
		session.statisticsRequire(len(numbers), name)
		return element(statisticsMode(numbers))
	case "quantile":
		session.statisticsRequire(len(numbers), name)
		probability, ok := arguments[1].(float64)
		if !ok {
			session.raise("NullError", "quantile probability is null")
		}
		if !(probability >= 0 && probability <= 1) {
			session.raise("StatisticsError", "quantile probability must be between 0.0 and 1.0")
		}
		return session.statisticsFinite(statisticsQuantile(numbers, probability), name)
	}
	session.raise("Error", "unsupported Statistics function "+name)
	return nil
}

// statisticsIntSum adds Int values through the language's checked arithmetic, so
// an overflowing total is an error rather than a rounded Real.
func (session *Session) statisticsIntSum(value any) int64 {
	total := int64(0)
	for _, item := range session.requireList(value).Items {
		total = session.intAdd(total, item.(int64))
	}
	return total
}

func (session *Session) statisticsSpread(name string, numbers []float64) any {
	sample := name == "sampleVariance" || name == "sampleStdDev"
	if sample && len(numbers) < 2 {
		session.raise("StatisticsError", "sampleVariance requires at least two values")
	}
	session.statisticsRequire(len(numbers), name)
	variance := statisticsVariance(numbers, sample)
	if name == "stdDev" || name == "sampleStdDev" {
		return session.statisticsFinite(math.Sqrt(variance), name)
	}
	return session.statisticsFinite(variance, name)
}

func statisticsMean(numbers []float64) float64 {
	total := float64(0)
	for _, value := range numbers {
		total += value
	}
	return total / float64(len(numbers))
}

// statisticsSorted orders a copy, so a caller's List is never reordered.
func statisticsSorted(numbers []float64) []float64 {
	ordered := append([]float64(nil), numbers...)
	sort.Float64s(ordered)
	return ordered
}

func statisticsMedian(numbers []float64) float64 {
	ordered := statisticsSorted(numbers)
	middle := len(ordered) / 2
	if len(ordered)%2 == 1 {
		return ordered[middle]
	}
	low, high := ordered[middle-1], ordered[middle]
	return low + (high-low)/2
}

func statisticsVariance(numbers []float64, sample bool) float64 {
	mean := statisticsMean(numbers)
	total := float64(0)
	for _, value := range numbers {
		deviation := value - mean
		total += deviation * deviation
	}
	divisor := float64(len(numbers))
	if sample {
		divisor = float64(len(numbers) - 1)
	}
	return total / divisor
}

// statisticsMode breaks a tie by first occurrence, so the result never depends
// on map iteration order.
func statisticsMode(numbers []float64) float64 {
	counts := make(map[float64]int, len(numbers))
	for _, value := range numbers {
		counts[value]++
	}
	best := numbers[0]
	for _, value := range numbers {
		if counts[value] > counts[best] {
			best = value
		}
	}
	return best
}

func statisticsQuantile(numbers []float64, probability float64) float64 {
	ordered := statisticsSorted(numbers)
	if len(ordered) == 1 {
		return ordered[0]
	}
	position := probability * float64(len(ordered)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return ordered[lower]
	}
	weight := position - float64(lower)
	return ordered[lower] + (ordered[upper]-ordered[lower])*weight
}
