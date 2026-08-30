// Package evaluator executes validated, typed AhdCode IR in a persistent
// process. It is used by the REPL; ordinary file execution continues to use
// the native Go backend.
package evaluator

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"ahdcode/internal/ir"
)

// Nothing is the sole runtime value of an expression whose static type is
// Nothing. It is never rendered as an AhdCode value.
type nothing struct{}

var Nothing any = nothing{}

type Cell struct {
	Value    any
	Constant bool
}

type List struct {
	Items  []any
	Frozen bool
}

type Pair struct {
	Keys   []any
	Values map[any]any
	Frozen bool
}

type Range struct {
	Current  int64
	Stop     int64
	Step     int64
	Finished bool
}

type Instance struct {
	Class  ir.ClassID
	Fields map[ir.FieldID]any
	Frozen bool
}

type FunctionValue struct {
	Callable ir.CallableID
	Receiver *Instance
	Direct   bool
}

type ClassValue struct{ Class ir.ClassID }

// RuntimeError is one catchable AhdCode error that escaped evaluation.
type RuntimeError struct {
	Class    ir.ClassID
	Name     string
	Message  string
	Instance *Instance
}

func (failure *RuntimeError) Error() string {
	if failure == nil {
		return "Error"
	}
	if failure.Message == "" {
		return failure.Name
	}
	return failure.Name + ": " + failure.Message
}

type raised struct{ failure *RuntimeError }

func (session *Session) raise(name, message string) {
	identity := session.errorClass(name)
	instance := &Instance{Class: identity, Fields: make(map[ir.FieldID]any)}
	if field := session.fieldNamed(identity, "message"); field != "" {
		instance.Fields[field] = message
	}
	panic(raised{failure: &RuntimeError{Class: identity, Name: name, Message: message, Instance: instance}})
}

func (session *Session) requireList(value any) *List {
	list, ok := value.(*List)
	if !ok || list == nil {
		session.raise("NullError", "List value is null")
	}
	return list
}

func (session *Session) requirePair(value any) *Pair {
	pair, ok := value.(*Pair)
	if !ok || pair == nil {
		session.raise("NullError", "Pair value is null")
	}
	if pair.Values == nil {
		pair.Values = make(map[any]any)
	}
	return pair
}

func (session *Session) requireInstance(value any) *Instance {
	instance, ok := value.(*Instance)
	if !ok || instance == nil {
		session.raise("NullError", "Class value is null")
	}
	return instance
}

func (session *Session) requireMutable(value any) {
	switch target := value.(type) {
	case *List:
		if target == nil {
			session.raise("NullError", "List value is null")
		}
		if target.Frozen {
			session.raise("ConstantError", "cannot mutate a Constant object")
		}
	case *Pair:
		if target == nil {
			session.raise("NullError", "Pair value is null")
		}
		if target.Frozen {
			session.raise("ConstantError", "cannot mutate a Constant object")
		}
	case *Instance:
		if target == nil {
			session.raise("NullError", "Class value is null")
		}
		if target.Frozen {
			session.raise("ConstantError", "cannot mutate a Constant object")
		}
	}
}

func freeze(value any) {
	freezeSeen(value, make(map[any]bool))
}

func freezeSeen(value any, seen map[any]bool) {
	switch target := value.(type) {
	case *List:
		if target == nil || seen[target] {
			return
		}
		seen[target] = true
		target.Frozen = true
		for _, item := range target.Items {
			freezeSeen(item, seen)
		}
	case *Pair:
		if target == nil || seen[target] {
			return
		}
		seen[target] = true
		target.Frozen = true
		for _, key := range target.Keys {
			freezeSeen(target.Values[key], seen)
		}
	case *Instance:
		if target == nil || seen[target] {
			return
		}
		seen[target] = true
		target.Frozen = true
		for _, item := range target.Fields {
			freezeSeen(item, seen)
		}
	}
}

func pairSet(pair *Pair, key, value any) {
	if _, exists := pair.Values[key]; !exists {
		pair.Keys = append(pair.Keys, key)
	}
	pair.Values[key] = value
}

func pairDelete(pair *Pair, key any) bool {
	if _, exists := pair.Values[key]; !exists {
		return false
	}
	delete(pair.Values, key)
	for index, existing := range pair.Keys {
		if existing == key {
			pair.Keys = append(pair.Keys[:index], pair.Keys[index+1:]...)
			break
		}
	}
	return true
}

func resolveIndex(session *Session, index int64, length int) int {
	resolved := index
	if resolved < 0 {
		resolved += int64(length)
	}
	if resolved < 0 || resolved >= int64(length) {
		session.raise("IndexError", fmt.Sprintf("index %d is outside length %d", index, length))
	}
	return int(resolved)
}

func resolveRange(start int64, hasStart bool, end int64, hasEnd bool, length int) (int, int) {
	low, high := int64(0), int64(length)
	if hasStart {
		low = start
		if low < 0 {
			low += int64(length)
		}
	}
	if hasEnd {
		high = end
		if high < 0 {
			high += int64(length)
		}
	}
	if low < 0 {
		low = 0
	}
	if high < 0 {
		high = 0
	}
	if low > int64(length) {
		low = int64(length)
	}
	if high > int64(length) {
		high = int64(length)
	}
	if high < low {
		high = low
	}
	return int(low), int(high)
}

func (session *Session) writeText(text string) {
	_, _ = io.WriteString(session.Output, text)
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func className(identity ir.ClassID) string {
	text := string(identity)
	if index := strings.LastIndex(text, "::class::"); index >= 0 {
		return text[index+len("::class::"):]
	}
	return text
}
