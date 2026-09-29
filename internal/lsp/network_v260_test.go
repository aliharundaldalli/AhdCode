package lsp

import (
	"strings"
	"testing"
)

const networkV260LSPSource = `bring DNS
bring TLS
from DNS bring DNSResult
from TLS bring TLSInfo
found: DNSResult := DNS.lookup(host: "example.com", timeoutSeconds: 5)
info: TLSInfo := TLS.inspect(host: "example.com", port: 443, timeoutSeconds: 5)
`

func TestCompletionOffersTheV260NetworkSurfaceOverTheWire(t *testing.T) {
	cases := []struct {
		prefix string
		labels []string
	}{
		{"DNS.", []string{"lookup", "DNSResult", "DNSError"}},
		{"TLS.", []string{"inspect", "TLSInfo", "TLSError"}},
		{"found.", []string{"host", "addresses", "ipv4", "ipv6"}},
		{"info.", []string{"host", "port", "valid", "verificationStatus", "subject", "issuer", "dnsNames", "notBefore", "notAfter", "protocol", "cipherSuite"}},
	}
	for _, testCase := range cases {
		source := networkV260LSPSource + testCase.prefix
		items := completionAt(t, source, "file:///main.ahd", len(source))
		for _, label := range testCase.labels {
			if !hasCompletionLabel(items, label) {
				t.Fatalf("%s offers no %q; got %#v", testCase.prefix, label, items)
			}
		}
	}
	source := "bring "
	items := completionAt(t, source, "file:///main.ahd", len(source))
	for _, module := range []string{"DNS", "TLS"} {
		if !hasCompletionLabel(items, module) {
			t.Fatalf("bring offers no %s module", module)
		}
	}
}

func TestHoverAndSignatureHelpForV260NetworkOverTheWire(t *testing.T) {
	source := networkV260LSPSource + `DNS.lookup("a.example")
TLS.inspect("a.example")
expires := info.notAfter()
`
	hover := hoverAt(t, source, "file:///main.ahd", "TLS.inspect(host", len("TLS.")+1)
	for _, want := range []string{"host: String", "port: Int", "timeoutSeconds: Int", "TLSInfo"} {
		if !strings.Contains(hover.Contents.Value, want) {
			t.Fatalf("TLS.inspect hover lacks %q: %q", want, hover.Contents.Value)
		}
	}
	help, found := signatureHelpAt(t, source, "file:///main.ahd", `DNS.lookup("a.example")`, len("DNS.lookup("))
	if !found || len(help.Signatures) == 0 || !strings.Contains(help.Signatures[0].Label, "host: String") ||
		!strings.Contains(help.Signatures[0].Label, "timeoutSeconds: Int") {
		t.Fatalf("DNS.lookup signature help = %+v, found=%v", help, found)
	}
	hover = hoverAt(t, source, "file:///main.ahd", "info.notAfter()", len("info.")+1)
	if !strings.Contains(hover.Contents.Value, "notAfter") || !strings.Contains(hover.Contents.Value, "DateTime") {
		t.Fatalf("notAfter hover = %q", hover.Contents.Value)
	}
}
