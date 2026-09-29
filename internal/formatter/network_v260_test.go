package formatter

import (
	"strings"
	"testing"
)

// TestV260NetworkCallsFormatCanonically guards the long-call class for the
// v2.6.0 DNS and TLS calls: tokens survive, output is idempotent, and one-line
// and multiline spellings format identically.
func TestV260NetworkCallsFormatCanonically(t *testing.T) {
	oneLine := `bring DNS
bring TLS
found:=DNS.lookup(host:"panel.production-cluster-eu-west.example-customer-domain.com",timeoutSeconds:5)
info:=TLS.inspect(host:"panel.production-cluster-eu-west.example-customer-domain.com",port:443,timeoutSeconds:5)
short:=TLS.inspect(host:"example.com",port:443,timeoutSeconds:5)
`
	multiline := `bring DNS
bring TLS
found := DNS.lookup(
    host: "panel.production-cluster-eu-west.example-customer-domain.com",
    timeoutSeconds: 5
)
info := TLS.inspect(
    host: "panel.production-cluster-eu-west.example-customer-domain.com",
    port: 443,
    timeoutSeconds: 5
)
short := TLS.inspect(host: "example.com", port: 443, timeoutSeconds: 5)
`
	first := formatText(t, oneLine)
	if withoutSpace(first) != withoutSpace(oneLine) {
		t.Fatalf("formatting changed the program's tokens:\n%s", first)
	}
	for _, wanted := range []string{"timeoutSeconds: 5", "port: 443", `short := TLS.inspect(host: "example.com", port: 443, timeoutSeconds: 5)`} {
		if !strings.Contains(first, wanted) {
			t.Fatalf("formatted output lacks %q:\n%s", wanted, first)
		}
	}
	if second := formatText(t, first); second != first {
		t.Fatalf("not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if fromMultiline := formatText(t, multiline); fromMultiline != first {
		t.Fatalf("spellings differ:\none-line:\n%s\nmultiline:\n%s", first, fromMultiline)
	}
}
