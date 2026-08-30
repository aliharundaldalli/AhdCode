package evaluator

import (
	"fmt"
	"regexp"

	"ahdcode/internal/ir"
)

var regexClassID = ir.ClassID("builtin:Regex::class::Pattern")
var regexPatternField = ir.FieldID(string(regexClassID) + "::field::pattern")

// regexCompiled compiles (and caches, on the Session) a pattern by its exact
// source text, so a Regex value used repeatedly across REPL commands or
// within one run pays the compilation cost only once. Mirrors
// ahdRegexCompiled in the native backend runtime.
func (session *Session) regexCompiled(pattern string) *regexp.Regexp {
	if session.regexCache == nil {
		session.regexCache = make(map[string]*regexp.Regexp)
	}
	if cached, ok := session.regexCache[pattern]; ok {
		return cached
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		session.raise("RegexError", fmt.Sprintf("invalid Regex pattern %q: %v", pattern, err))
	}
	session.regexCache[pattern] = compiled
	return compiled
}

func (session *Session) regexInstance(pattern string) *Instance {
	return &Instance{Class: regexClassID, Fields: map[ir.FieldID]any{regexPatternField: pattern}}
}

func (session *Session) regexBuiltin(name string, arguments []any) any {
	if name != "compile" {
		session.raise("Error", "unsupported Regex function "+name)
		return nil
	}
	pattern := arguments[0].(string)
	session.regexCompiled(pattern)
	return session.regexInstance(pattern)
}

func (session *Session) regexOperation(name string, receiver any, arguments []any) any {
	instance := session.requireInstance(receiver)
	pattern, _ := instance.Fields[regexPatternField].(string)
	compiled := session.regexCompiled(pattern)
	switch name {
	case "Regex.matches":
		return compiled.MatchString(arguments[0].(string))
	case "Regex.find":
		text := arguments[0].(string)
		location := compiled.FindStringIndex(text)
		if location == nil {
			return nil
		}
		return text[location[0]:location[1]]
	case "Regex.findAll":
		matches := compiled.FindAllString(arguments[0].(string), -1)
		items := make([]any, len(matches))
		for index, match := range matches {
			items[index] = match
		}
		return &List{Items: items}
	case "Regex.groups":
		match := compiled.FindStringSubmatch(arguments[0].(string))
		if match == nil {
			return nil
		}
		items := make([]any, len(match))
		for index, group := range match {
			items[index] = group
		}
		return &List{Items: items}
	case "Regex.replace":
		return compiled.ReplaceAllString(arguments[0].(string), arguments[1].(string))
	case "Regex.split":
		parts := compiled.Split(arguments[0].(string), -1)
		items := make([]any, len(parts))
		for index, part := range parts {
			items[index] = part
		}
		return &List{Items: items}
	}
	session.raise("Error", "unsupported Regex operation "+name)
	return nil
}
