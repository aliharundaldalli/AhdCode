package evaluator

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// The Env standard module's REPL implementation. It mirrors the native
// backend's ahdruntime Env section function-for-function - the same
// dotenv grammar, the same validation-before-apply Load semantics - since
// Env has no data-carrying Class at all (unlike Word/JSON/XML), there is no
// receiver representation to keep in sync between the two runtimes, only
// this parsing/validation logic.

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type envEntry struct {
	Key   string
	Value string
}

func (s *Session) envValidateName(name string) {
	if name == "" {
		s.raise("EnvError", "environment variable name must not be empty")
	}
	if strings.IndexByte(name, 0) >= 0 {
		s.raise("EnvError", "environment variable name must not contain a NUL byte")
	}
	if strings.IndexByte(name, '=') >= 0 {
		s.raise("EnvError", "environment variable name must not contain '='")
	}
}

func (s *Session) envBuiltin(name string, args []any) any {
	switch name {
	case "get":
		value, present := os.LookupEnv(args[0].(string))
		if !present {
			return nil
		}
		return value
	case "getOr":
		if value, present := os.LookupEnv(args[0].(string)); present {
			return value
		}
		return args[1].(string)
	case "exists":
		_, present := os.LookupEnv(args[0].(string))
		return present
	case "set":
		name := args[0].(string)
		s.envValidateName(name)
		if err := os.Setenv(name, args[1].(string)); err != nil {
			s.raise("EnvError", "could not set the environment variable")
		}
		return Nothing
	case "unset":
		name := args[0].(string)
		s.envValidateName(name)
		if err := os.Unsetenv(name); err != nil {
			s.raise("EnvError", "could not unset the environment variable")
		}
		return Nothing
	case "read":
		entries := s.envReadFile(args[0].(string))
		pair := &Pair{Keys: make([]any, len(entries)), Values: make(map[any]any, len(entries))}
		for index, entry := range entries {
			pair.Keys[index] = entry.Key
			pair.Values[entry.Key] = entry.Value
		}
		return pair
	case "load":
		override := len(args) > 1 && args[1] != nil && args[1].(bool)
		entries := s.envReadFile(args[0].(string))
		for _, entry := range entries {
			if !override {
				if _, present := os.LookupEnv(entry.Key); present {
					continue
				}
			}
			if err := os.Setenv(entry.Key, entry.Value); err != nil {
				s.raise("EnvError", "could not set the environment variable")
			}
		}
		return Nothing
	}
	s.raise("Error", "unsupported Env function "+name)
	return nil
}

func (s *Session) envReadFile(path string) []envEntry {
	content, err := os.ReadFile(s.sessionPath(path))
	if err != nil {
		s.raise("EnvError", "could not read the .env file: "+err.Error())
	}
	return s.envParseFile(string(content))
}

func (s *Session) envParseFile(content string) []envEntry {
	var entries []envEntry
	seen := make(map[string]bool)
	for index, rawLine := range strings.Split(content, "\n") {
		lineNumber := index + 1
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || trimmed[0] == '#' {
			continue
		}
		key, value := s.envParseAssignment(trimmed, lineNumber)
		if seen[key] {
			s.raise("EnvError", fmt.Sprintf("line %d: duplicate key %q in .env file", lineNumber, key))
		}
		seen[key] = true
		entries = append(entries, envEntry{Key: key, Value: value})
	}
	return entries
}

func (s *Session) envParseAssignment(line string, lineNumber int) (string, string) {
	equals := strings.IndexByte(line, '=')
	if equals < 0 {
		s.raise("EnvError", fmt.Sprintf("line %d is not a valid KEY=value assignment", lineNumber))
	}
	key := line[:equals]
	if !envKeyPattern.MatchString(key) {
		s.raise("EnvError", fmt.Sprintf("line %d has an invalid key %q", lineNumber, key))
	}
	return key, s.envParseValue(line[equals+1:], lineNumber)
}

func (s *Session) envParseValue(rest string, lineNumber int) string {
	if len(rest) == 0 {
		return ""
	}
	switch rest[0] {
	case '"':
		return s.envParseDoubleQuoted(rest, lineNumber)
	case '\'':
		return s.envParseSingleQuoted(rest, lineNumber)
	default:
		return strings.TrimSpace(rest)
	}
}

func (s *Session) envParseDoubleQuoted(rest string, lineNumber int) string {
	var builder strings.Builder
	index := 1
	for index < len(rest) {
		character := rest[index]
		switch character {
		case '"':
			if strings.TrimSpace(rest[index+1:]) != "" {
				s.raise("EnvError", fmt.Sprintf("line %d has content after a closing quote", lineNumber))
			}
			return builder.String()
		case '\\':
			index++
			if index >= len(rest) {
				s.raise("EnvError", fmt.Sprintf("line %d has an incomplete escape sequence", lineNumber))
			}
			switch rest[index] {
			case '\\':
				builder.WriteByte('\\')
			case '"':
				builder.WriteByte('"')
			case 'n':
				builder.WriteByte('\n')
			case 'r':
				builder.WriteByte('\r')
			case 't':
				builder.WriteByte('\t')
			default:
				s.raise("EnvError", fmt.Sprintf("line %d has an invalid escape sequence", lineNumber))
			}
			index++
			continue
		default:
			builder.WriteByte(character)
		}
		index++
	}
	s.raise("EnvError", fmt.Sprintf("line %d has an unterminated double-quoted value", lineNumber))
	return ""
}

func (s *Session) envParseSingleQuoted(rest string, lineNumber int) string {
	closing := strings.IndexByte(rest[1:], '\'')
	if closing < 0 {
		s.raise("EnvError", fmt.Sprintf("line %d has an unterminated single-quoted value", lineNumber))
	}
	value := rest[1 : 1+closing]
	if strings.TrimSpace(rest[1+closing+1:]) != "" {
		s.raise("EnvError", fmt.Sprintf("line %d has content after a closing quote", lineNumber))
	}
	return value
}
