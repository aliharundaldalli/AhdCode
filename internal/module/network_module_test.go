package module

import (
	"strings"
	"testing"
)

// TestNetworkModulesCannotBeShadowedAndTLSBringsTimeImplicitly covers the
// v2.6.0 standard-module names: a sibling DNS.ahd or TLS.ahd is no longer
// what `bring DNS` / `bring TLS` loads, and TLS makes the Time Classes
// available (so TLSInfo.notAfter().year resolves) without binding any Time
// name in the program.
func TestNetworkModulesCannotBeShadowedAndTLSBringsTimeImplicitly(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd": "bring DNS\nbring TLS\ninfo := TLS.inspect(\"example.com\")\nyear: Int := info.notAfter().year\nfound := DNS.lookup(\"example.com\")\n",
		"/DNS.ahd":  `lookup: Bool := true`,
		"/TLS.ahd":  `inspect: Bool := true`,
	}, "/Main.ahd")
	requireClean(t, result)
	for _, name := range []string{"/DNS.ahd", "/TLS.ahd"} {
		if workspace.LoadCount(memoryIdentity(name).ID) != 0 {
			t.Fatalf("the sibling %s shadowed the standard module", name)
		}
	}
	for _, name := range []string{"DNS", "TLS", "Time"} {
		module := moduleNamed(t, result, name)
		if !module.Source.Builtin || string(module.ID) != "builtin:"+name {
			t.Fatalf("%s did not keep its built-in identity: %#v", name, module)
		}
	}

	// The implicit dependency binds no name: Time and DateTime still need
	// their own bring.
	_, result = compileMemory(t, map[string]string{
		"/Main.ahd": "bring TLS\nnow := Time.now()\n",
	}, "/Main.ahd")
	if !result.HasErrors() {
		t.Fatal("bring TLS made the Time namespace visible")
	}
	_, result = compileMemory(t, map[string]string{
		"/Main.ahd": "bring TLS\nwhen: DateTime := TLS.inspect(\"a\").notAfter()\n",
	}, "/Main.ahd")
	if !result.HasErrors() {
		t.Fatal("bring TLS bound the DateTime name")
	}
	var text []string
	for _, item := range result.Diagnostics {
		text = append(text, item.Diagnostic.Message)
	}
	if !strings.Contains(strings.Join(text, "\n"), "DateTime") {
		t.Fatalf("unexpected diagnostics: %v", text)
	}
}
