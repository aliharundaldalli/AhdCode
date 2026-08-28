package golang

import (
	"hash/fnv"
	"strconv"
	"strings"
	"unicode"
)

// Generated identifiers are derived from stable IR identities rather than from
// raw AhdCode spelling, so Go keywords, Unicode identifiers, and cross-module
// name collisions cannot corrupt the generated program.
const (
	globalPrefix      = "gv"
	localPrefix       = "lv"
	functionPrefix    = "fn"
	classPrefix       = "Cl"
	fieldPrefix       = "Fd"
	constructorPrefix = "ct"
	initPrefix        = "md"
)

// mangle builds a deterministic, collision-resistant Go identifier from a
// stable IR identity.
func mangle(prefix, identity string) string {
	return prefix + "_" + readable(identity) + "_" + digest(identity)
}

// mangleNamed keeps a caller-supplied readable fragment while still deriving
// uniqueness from the stable IR identity.
func mangleNamed(prefix, name, identity string) string {
	readableName := sanitize(name)
	if readableName == "" {
		readableName = "x"
	}
	return prefix + "_" + readableName + "_" + digest(identity)
}

func digest(identity string) string {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(identity))
	return strconv.FormatUint(hash.Sum64(), 16)
}

// readable extracts an ASCII-safe, human-recognizable fragment of an IR
// identity. It is decorative only; uniqueness comes from the digest.
func readable(identity string) string {
	segments := strings.Split(identity, "::")
	for index := len(segments) - 1; index >= 0; index-- {
		candidate := sanitize(segments[index])
		if candidate != "" {
			return candidate
		}
	}
	return "x"
}

func sanitize(segment string) string {
	if cut := strings.IndexAny(segment, "(@"); cut >= 0 {
		segment = segment[:cut]
	}
	var out strings.Builder
	for _, item := range segment {
		switch {
		case item >= 'a' && item <= 'z', item >= 'A' && item <= 'Z', item == '_':
			out.WriteRune(item)
		case unicode.IsDigit(item) && out.Len() > 0:
			out.WriteRune(item)
		}
		if out.Len() >= 24 {
			break
		}
	}
	return out.String()
}
