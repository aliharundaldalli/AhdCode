package semantic

import (
	"reflect"
	"strings"
	"testing"

	"ahdcode/internal/types"
)

const networkPreamble = `bring DNS
bring TLS
from DNS bring (DNSResult, DNSError)
from TLS bring (TLSInfo, TLSError)
`

func TestNetworkInspectionTypeChecks(t *testing.T) {
	result := analyzeWithStandardModules(t, networkPreamble+`bring Time
from Time bring DateTime
found: DNSResult := DNS.lookup("panel.example.com")
bounded: DNSResult := DNS.lookup(host: "panel.example.com", timeoutSeconds: 5)
host: String := found.host()
every: List<String> := found.addresses()
four: List<String> := found.ipv4()
six: List<String> := found.ipv6()

info: TLSInfo := TLS.inspect("example.com")
explicit: TLSInfo := TLS.inspect(host: "example.com", port: 8443, timeoutSeconds: 5)
positional: TLSInfo := TLS.inspect("example.com", 443, 5)
ok: Bool := info.valid()
status: String := info.verificationStatus()
subject: String := info.subject()
issuer: String := info.issuer()
names: List<String> := info.dnsNames()
starts: DateTime := info.notBefore()
ends: DateTime := info.notAfter()
protocol: String := info.protocol()
cipher: String := info.cipherSuite()
port: Int := info.port()
name: String := info.host()
attempt {
    DNS.lookup("x")
}
except DNSError as failure {
    write(failure.message)
}
attempt {
    TLS.inspect("x")
}
except TLSError as failure {
    write(failure.message)
}
`)
	requireSemanticClean(t, result)
}

func TestNetworkInspectionRejectsWrongTypesAtCompileTime(t *testing.T) {
	setup := "found: DNSResult := DNS.lookup(\"a\")\ninfo: TLSInfo := TLS.inspect(\"a\")\n"
	for _, source := range []string{
		`DNS.lookup(42)`,
		`DNS.lookup()`,
		`DNS.lookup(host: "a", timeoutSeconds: "5")`,
		`DNS.lookup(host: "a", recordType: "MX")`,
		`DNS.lookup("a", 5, 6)`,
		`DNS.resolveMX("a")`,
		`DNS.update("a", "1.2.3.4")`,
		`TLS.inspect(host: 42)`,
		`TLS.inspect(host: "a", port: "443")`,
		`TLS.inspect(host: "a", insecure: true)`,
		`TLS.inspect(host: "a", serverName: "b")`,
		`TLS.renew("a")`,
		`found.addresses(1)`,
		`every: String := found.addresses()`,
		`text: String := info.valid()`,
		`year: Int := info.notAfter()`,
		`info.privateKey()`,
		`DNSResult("x")`,
		`TLSInfo("x")`,
	} {
		t.Run(source, func(t *testing.T) {
			requireSemanticFailure(t, analyzeWithStandardModules(t, networkPreamble+setup+source+"\n"))
		})
	}
}

func TestNetworkModulesExposeExactSurface(t *testing.T) {
	modules := StandardModuleInterfaces()
	if want := []string{"DNSError", "DNSResult", "lookup"}; !reflect.DeepEqual(modules["DNS"].ExportNames, want) {
		t.Fatalf("DNS exports = %v", modules["DNS"].ExportNames)
	}
	if want := []string{"TLSError", "TLSInfo", "inspect"}; !reflect.DeepEqual(modules["TLS"].ExportNames, want) {
		t.Fatalf("TLS exports = %v", modules["TLS"].ExportNames)
	}
	parameters := func(module, name string) string {
		var names []string
		for _, parameter := range modules[module].Exports[name].Callable.Signature.Parameters {
			names = append(names, parameter.Name)
		}
		return strings.Join(names, ",")
	}
	if parameters("DNS", "lookup") != "host,timeoutSeconds" || parameters("TLS", "inspect") != "host,port,timeoutSeconds" {
		t.Fatalf("signatures: %s / %s", parameters("DNS", "lookup"), parameters("TLS", "inspect"))
	}
	for _, identity := range []*types.ClassSymbol{dnsErrorClass, tlsErrorClass} {
		if identity.Parent == nil || identity.Parent.Name != "Error" {
			t.Fatalf("%s must derive from Error", identity.Name)
		}
	}
	for identity, want := range map[*types.ClassSymbol][]string{dnsResultClass: DNSResultOperations, tlsInfoClass: TLSInfoOperations} {
		members := BuiltinClassMembers(identity)
		if len(members) != len(want) {
			t.Fatalf("%s publishes %d members", identity.Name, len(members))
		}
		for _, member := range members {
			if member == nil || member.Callable == nil {
				t.Fatalf("%s has a member without a signature", identity.Name)
			}
		}
	}
}
