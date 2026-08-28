package token

import "testing"

func TestReservedKeywordTable(t *testing.T) {
	tests := map[string]Kind{
		"and": KeywordAnd, "null": KeywordNull, "Int": KeywordInt,
		"Function": KeywordFunction, "Confidential": KeywordConfidential,
		"Object": KeywordObject, "Error": KeywordError,
	}
	for text, want := range tests {
		got, ok := LookupKeyword(text)
		if !ok || got != want {
			t.Fatalf("LookupKeyword(%q) = %v, %v; want %v, true", text, got, ok, want)
		}
	}
	for _, contextual := range []string{"structure", "attribute", "SuperClass", "write", "IndexError"} {
		if kind, ok := LookupKeyword(contextual); ok {
			t.Fatalf("contextual/predeclared %q unexpectedly reserved as %v", contextual, kind)
		}
	}
}

func TestEveryKindHasName(t *testing.T) {
	for kind := Invalid; kind <= RightBrace; kind++ {
		if got := kind.String(); got == "" || got == "Kind(?)" {
			t.Fatalf("kind %d has no stable name", kind)
		}
	}
}
