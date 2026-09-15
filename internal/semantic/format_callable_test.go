package semantic

import (
	"strings"
	"testing"

	"ahdcode/internal/types"
)

func TestFormatCallableShowsNullability(t *testing.T) {
	signature := &types.Signature{
		Parameters: []types.Parameter{{Name: "name", Type: types.String}, {Name: "database", Type: types.String, HasDefault: true}},
		Return:     types.String,
	}
	callable := &Callable{Signature: signature, ParameterNull: []NullState{NonNull, MaybeNull}, ReturnNull: MaybeNull}
	if got, want := FormatCallable(callable), "(name: String, database: String? := default) -> String?"; got != want {
		t.Fatalf("FormatCallable = %q, want %q", got, want)
	}
	if got, want := strings.Join(FormatCallableParameters(callable), ", "), "name: String, database: String? := default"; got != want {
		t.Fatalf("FormatCallableParameters = %q, want %q", got, want)
	}
	// Diagnostics keep the unchanged rendering.
	if got, want := FormatSignature(signature), "(name: String, database: String := default) -> String"; got != want {
		t.Fatalf("FormatSignature changed: %q, want %q", got, want)
	}
	nonNull := &Callable{Signature: signature, ParameterNull: []NullState{NonNull, NonNull}, ReturnNull: NonNull}
	if got := FormatCallable(nonNull); got != FormatSignature(signature) {
		t.Fatalf("a non-null callable renders %q, want the FormatSignature text %q", got, FormatSignature(signature))
	}
	nothing := &Callable{Signature: &types.Signature{Return: types.Nothing}, ReturnNull: MaybeNull}
	if got := FormatCallable(nothing); got != "() -> Nothing" {
		t.Fatalf("a Nothing return renders %q", got)
	}
	if got := FormatCallable(nil); got != "Function<?>" {
		t.Fatalf("a nil callable renders %q", got)
	}
	// Missing parameter metadata never invents a ?.
	sparse := &Callable{Signature: signature, ReturnNull: NonNull}
	if got := FormatCallable(sparse); got != FormatSignature(signature) {
		t.Fatalf("a callable without parameter metadata renders %q", got)
	}
}

// Only the documented nullable standard-module signatures show a ?.
func TestFormatCallableStandardModuleSignatures(t *testing.T) {
	modules := StandardModuleInterfaces()
	for _, testCase := range []struct{ module, name, want string }{
		{"Terminal", "width", "() -> Int?"},
		{"Terminal", "height", "() -> Int?"},
		{"Terminal", "isInteractive", "() -> Bool"},
		{"Terminal", "emit", "(parts: List<String>, separator: String := default, ending: String := default) -> Nothing"},
		{"Terminal", "style", "(text: String, foreground: String := default, background: String := default, bold: Bool := default, underline: Bool := default) -> String"},
		{"Env", "get", "(name: String) -> String?"},
		{"Env", "secret", "(name: String) -> String?"},
		{"Env", "getOr", "(name: String, fallback: String) -> String"},
	} {
		symbol := modules[testCase.module].Exports[testCase.name]
		if symbol == nil || symbol.Callable == nil {
			t.Fatalf("%s.%s has no callable", testCase.module, testCase.name)
		}
		if got := FormatCallable(symbol.Callable); got != testCase.want {
			t.Fatalf("%s.%s = %q, want %q", testCase.module, testCase.name, got, testCase.want)
		}
	}
	connect := FormatCallable(modules["PostgreSQL"].Exports["connect"].Callable)
	if !strings.Contains(connect, "database: String? := default") || strings.Contains(connect, "host: String?") {
		t.Fatalf("PostgreSQL.connect = %q", connect)
	}
}
