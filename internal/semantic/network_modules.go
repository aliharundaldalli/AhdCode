package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

// The v2.6.0 network-inspection modules. DNS.lookup resolves a host name to
// its addresses; TLS.inspect reports the certificate and handshake a TLS
// endpoint presents, verified explicitly against the system roots. Both are
// bounded by a timeout and return immutable, opaque values only their module
// produces. TLSInfo's certificate dates are the Time module's DateTime.

const (
	dnsModuleID = "builtin:DNS"
	tlsModuleID = "builtin:TLS"
)

var (
	networkErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	dnsResultClass = &types.ClassSymbol{ModuleID: dnsModuleID, Name: "DNSResult"}
	dnsErrorClass  = &types.ClassSymbol{ModuleID: dnsModuleID, Name: "DNSError", Parent: networkErrorParent}
	tlsInfoClass   = &types.ClassSymbol{ModuleID: tlsModuleID, Name: "TLSInfo"}
	tlsErrorClass  = &types.ClassSymbol{ModuleID: tlsModuleID, Name: "TLSError", Parent: networkErrorParent}
)

func DNSResultIdentity() *types.ClassSymbol { return dnsResultClass }
func DNSErrorIdentity() *types.ClassSymbol  { return dnsErrorClass }
func TLSInfoIdentity() *types.ClassSymbol   { return tlsInfoClass }
func TLSErrorIdentity() *types.ClassSymbol  { return tlsErrorClass }

// The members each value publishes, in documentation order. Lowering builds
// the IR Classes from these same lists.
var (
	DNSResultOperations = []string{"host", "addresses", "ipv4", "ipv6"}
	TLSInfoOperations   = []string{"host", "port", "valid", "verificationStatus", "subject", "issuer",
		"dnsNames", "notBefore", "notAfter", "protocol", "cipherSuite"}
)

func init() {
	strings := types.List{Element: types.String}
	dateTime := types.Class{Symbol: timeDateTimeClass}
	for name, result := range map[string]types.Type{"host": types.String, "addresses": strings, "ipv4": strings, "ipv6": strings} {
		systemsMembers[TypeOperation("DNSResult."+name)] = completionMember(dnsModuleID, name, result)
	}
	for name, result := range map[string]types.Type{
		"host": types.String, "port": types.Int, "valid": types.Bool, "verificationStatus": types.String,
		"subject": types.String, "issuer": types.String, "dnsNames": strings,
		"notBefore": dateTime, "notAfter": dateTime, "protocol": types.String, "cipherSuite": types.String,
	} {
		systemsMembers[TypeOperation("TLSInfo."+name)] = completionMember(tlsModuleID, name, result)
	}
}

func dnsModuleInterface() *ModuleInterface {
	module := standardInterface(dnsModuleID, "DNS")
	addSystemsClass(module, dnsModuleID, dnsErrorClass)
	addSystemsClass(module, dnsModuleID, dnsResultClass)
	addStandardExport(module, standardFunction(dnsModuleID, "lookup", types.Class{Symbol: dnsResultClass},
		types.Parameter{Name: "host", Type: types.String},
		types.Parameter{Name: "timeoutSeconds", Type: types.Int, HasDefault: true}))
	sort.Strings(module.ExportNames)
	return module
}

func tlsModuleInterface() *ModuleInterface {
	module := standardInterface(tlsModuleID, "TLS")
	addSystemsClass(module, tlsModuleID, tlsErrorClass)
	addSystemsClass(module, tlsModuleID, tlsInfoClass)
	addStandardExport(module, standardFunction(tlsModuleID, "inspect", types.Class{Symbol: tlsInfoClass},
		types.Parameter{Name: "host", Type: types.String},
		types.Parameter{Name: "port", Type: types.Int, HasDefault: true},
		types.Parameter{Name: "timeoutSeconds", Type: types.Int, HasDefault: true}))
	sort.Strings(module.ExportNames)
	return module
}
