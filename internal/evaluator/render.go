package evaluator

import (
	"math"
	"strconv"
	"strings"
)

type visit struct {
	kind byte
	ptr  any
}

// Render returns the canonical top-level AhdCode String representation used
// by REPL expression results and Fundamentals.str.
func (session *Session) Render(value any) string {
	return session.render(value, false, make(map[visit]bool))
}

func (session *Session) render(value any, nested bool, seen map[visit]bool) string {
	switch item := value.(type) {
	case nil:
		return "null"
	case nothing:
		return ""
	case int64:
		return strconv.FormatInt(item, 10)
	case float64:
		if math.IsNaN(item) {
			session.raise("DomainError", "Real value is not a number")
		}
		if math.IsInf(item, 0) {
			session.raise("OverflowError", "Real value is not finite")
		}
		return formatReal(item)
	case complex128:
		if math.IsNaN(real(item)) || math.IsNaN(imag(item)) {
			session.raise("DomainError", "Complex component is not a number")
		}
		if math.IsInf(real(item), 0) || math.IsInf(imag(item), 0) {
			session.raise("OverflowError", "Complex component is not finite")
		}
		sign := "+"
		imaginary := imag(item)
		if math.Signbit(imaginary) {
			sign = "-"
			imaginary = -imaginary
		}
		return formatReal(real(item)) + sign + formatReal(imaginary) + "I"
	case bool:
		return strconv.FormatBool(item)
	case string:
		if nested {
			return quote(item)
		}
		return item
	case *List:
		if item == nil {
			return "null"
		}
		key := visit{'l', item}
		if seen[key] {
			return "[...]"
		}
		seen[key] = true
		parts := make([]string, len(item.Items))
		for index, value := range item.Items {
			parts[index] = session.render(value, true, seen)
		}
		delete(seen, key)
		return "[" + strings.Join(parts, ", ") + "]"
	case *Pair:
		if item == nil {
			return "null"
		}
		key := visit{'p', item}
		if seen[key] {
			return "{...}"
		}
		seen[key] = true
		parts := make([]string, 0, len(item.Keys))
		for _, pairKey := range item.Keys {
			parts = append(parts, session.render(pairKey, true, seen)+": "+session.render(item.Values[pairKey], true, seen))
		}
		delete(seen, key)
		return "{" + strings.Join(parts, ", ") + "}"
	case *Instance:
		if item == nil {
			return "null"
		}
		return "<" + className(item.Class) + ">"
	case *FunctionValue:
		name := string(item.Callable)
		if function := session.functions[item.Callable]; function != nil {
			name = function.Name
		} else if index := strings.LastIndex(name, "::"); index >= 0 {
			name = name[index+2:]
		}
		return "<Function " + name + ">"
	case ClassValue:
		return "<Class " + className(item.Class) + ">"
	default:
		return "<value>"
	}
}

func quote(text string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, current := range text {
		switch current {
		case '\\':
			builder.WriteString(`\\`)
		case '"':
			builder.WriteString(`\"`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		default:
			builder.WriteRune(current)
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

func formatReal(value float64) string {
	scientific := strconv.FormatFloat(value, 'e', -1, 64)
	exponent := 0
	if marker := strings.IndexByte(scientific, 'e'); marker >= 0 {
		exponent, _ = strconv.Atoi(scientific[marker+1:])
	}
	if exponent < -4 || exponent >= 21 {
		return scientific
	}
	text := strconv.FormatFloat(value, 'f', -1, 64)
	if !strings.Contains(text, ".") {
		text += ".0"
	}
	return text
}

func (session *Session) same(left, right any) bool {
	switch first := left.(type) {
	case *List:
		second, ok := right.(*List)
		return ok && first == second
	case *Pair:
		second, ok := right.(*Pair)
		return ok && first == second
	case *Instance:
		second, ok := right.(*Instance)
		return ok && first == second
	case *FunctionValue:
		second, ok := right.(*FunctionValue)
		return ok && first.Callable == second.Callable && first.Receiver == second.Receiver
	default:
		return session.equal(left, right)
	}
}

type equalityVisit struct{ left, right any }

func (session *Session) equal(left, right any) bool {
	return session.equalSeen(left, right, make(map[equalityVisit]bool))
}

func (session *Session) equalSeen(left, right any, seen map[equalityVisit]bool) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	switch first := left.(type) {
	case int64:
		second, ok := right.(int64)
		return ok && first == second
	case float64:
		second, ok := right.(float64)
		return ok && first == second
	case complex128:
		second, ok := right.(complex128)
		return ok && first == second
	case string:
		second, ok := right.(string)
		return ok && first == second
	case bool:
		second, ok := right.(bool)
		return ok && first == second
	case *Instance:
		second, ok := right.(*Instance)
		return ok && first == second
	case *FunctionValue:
		second, ok := right.(*FunctionValue)
		return ok && first.Callable == second.Callable && first.Receiver == second.Receiver
	case *List:
		second, ok := right.(*List)
		if !ok || first == nil || second == nil || len(first.Items) != len(second.Items) {
			return ok && first == nil && second == nil
		}
		key := equalityVisit{first, second}
		if seen[key] {
			return true
		}
		seen[key] = true
		for index := range first.Items {
			if !session.equalSeen(first.Items[index], second.Items[index], seen) {
				return false
			}
		}
		return true
	case *Pair:
		second, ok := right.(*Pair)
		if !ok || first == nil || second == nil || len(first.Values) != len(second.Values) {
			return ok && first == nil && second == nil
		}
		key := equalityVisit{first, second}
		if seen[key] {
			return true
		}
		seen[key] = true
		for key, value := range first.Values {
			other, exists := second.Values[key]
			if !exists || !session.equalSeen(value, other, seen) {
				return false
			}
		}
		return true
	}
	return false
}

func (session *Session) parseInt(value string) int64 {
	text := strings.TrimSpace(value)
	if !validIntText(text) {
		session.raise("DomainError", "String is not valid decimal Int text")
	}
	result, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		session.raise("OverflowError", "decimal String is outside signed 64-bit Int range")
	}
	return result
}

func validIntText(text string) bool {
	if text == "" {
		return false
	}
	if text[0] == '+' || text[0] == '-' {
		text = text[1:]
	}
	if text == "" {
		return false
	}
	for _, current := range text {
		if current < '0' || current > '9' {
			return false
		}
	}
	return true
}

func (session *Session) parseReal(value string) float64 {
	text := strings.TrimSpace(value)
	if !validRealText(text) {
		session.raise("DomainError", "String is not valid decimal Real text")
	}
	result, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsInf(result, 0) || result == 0 && nonzeroSignificand(text) {
		session.raise("OverflowError", "decimal String is outside finite Real range")
	}
	if math.IsNaN(result) {
		session.raise("DomainError", "String is not valid decimal Real text")
	}
	return result
}

func validRealText(text string) bool {
	if text == "" {
		return false
	}
	index := 0
	if text[index] == '+' || text[index] == '-' {
		index++
	}
	digits := false
	for index < len(text) && text[index] >= '0' && text[index] <= '9' {
		index++
		digits = true
	}
	if !digits {
		return false
	}
	if index < len(text) && text[index] == '.' {
		index++
		start := index
		for index < len(text) && text[index] >= '0' && text[index] <= '9' {
			index++
		}
		if index == start {
			return false
		}
	}
	if index < len(text) && (text[index] == 'e' || text[index] == 'E') {
		index++
		if index < len(text) && (text[index] == '+' || text[index] == '-') {
			index++
		}
		start := index
		for index < len(text) && text[index] >= '0' && text[index] <= '9' {
			index++
		}
		if index == start {
			return false
		}
	}
	return index == len(text)
}

func nonzeroSignificand(text string) bool {
	for _, current := range text {
		if current == 'e' || current == 'E' {
			break
		}
		if current >= '1' && current <= '9' {
			return true
		}
	}
	return false
}
