package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

// The v2.6.0 DNS and TLS modules. Each call lowers to the shared ahdruntime
// function the evaluator also uses. DNSResult and TLSInfo are hidden-String
// handles emitted by emitSMTPHelpers; TLSInfo's certificate dates become the
// Time module's DateTime through the ordinary DateTime helper.

const (
	dnsModulePrefix = "builtin:DNS::"
	tlsModulePrefix = "builtin:TLS::"
)

var (
	dnsResultClass     = ir.ClassID("builtin:DNS::class::DNSResult")
	dnsResultDataField = ir.FieldID("builtin:DNS::class::DNSResult::field::data")
	dnsErrorClass      = ir.ClassID("builtin:DNS::class::DNSError")
	tlsInfoClass       = ir.ClassID("builtin:TLS::class::TLSInfo")
	tlsInfoDataField   = ir.FieldID("builtin:TLS::class::TLSInfo::field::data")
	tlsErrorClass      = ir.ClassID("builtin:TLS::class::TLSError")
)

func init() {
	systemsDefaultArguments["DNS.timeoutSeconds"] = "AhdDNSDefaultTimeoutSeconds"
	systemsDefaultArguments["TLS.port"] = "AhdTLSDefaultPort"
	systemsDefaultArguments["TLS.timeoutSeconds"] = "AhdTLSDefaultTimeoutSeconds"
}

func (generator *generator) dnsCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), dnsModulePrefix)
	if name != "lookup" {
		return generator.unsupported("DNS function "+name, meta.Span)
	}
	call := "AhdDNSLookup(" + generator.descriptorName(dnsErrorClass) + ", " + generator.systemsText(value, 0) + ", " +
		generator.systemsInt(value, 1, "DNS.timeoutSeconds") + ")"
	return generator.smtpValueFrom(dnsResultClass, call, meta)
}

func (generator *generator) tlsCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), tlsModulePrefix)
	if name != "inspect" {
		return generator.unsupported("TLS function "+name, meta.Span)
	}
	call := "AhdTLSInspect(" + generator.descriptorName(tlsErrorClass) + ", " + generator.systemsText(value, 0) + ", " +
		generator.systemsInt(value, 1, "TLS.port") + ", " + generator.systemsInt(value, 2, "TLS.timeoutSeconds") + ")"
	return generator.smtpValueFrom(tlsInfoClass, call, meta)
}

func isNetworkOperation(name string) bool {
	return strings.HasPrefix(name, "DNSResult.") || strings.HasPrefix(name, "TLSInfo.")
}

// networkOperation lowers the DNSResult and TLSInfo accessors.
func (generator *generator) networkOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	if strings.HasPrefix(name, "DNSResult.") {
		helpers := map[string]string{
			"DNSResult.host": "AhdDNSResultHost", "DNSResult.addresses": "AhdDNSResultAddresses",
			"DNSResult.ipv4": "AhdDNSResultIPv4", "DNSResult.ipv6": "AhdDNSResultIPv6",
		}
		helper, known := helpers[name]
		if !known {
			return generator.unsupported("DNS operation "+name, meta.Span)
		}
		return helper + "(" + generator.descriptorName(dnsErrorClass) + ", " +
			generator.smtpDataOf(dnsResultClass, dnsResultDataField, value.Callee) + ")"
	}
	helpers := map[string]string{
		"TLSInfo.host": "AhdTLSInfoHost", "TLSInfo.port": "AhdTLSInfoPort", "TLSInfo.valid": "AhdTLSInfoValid",
		"TLSInfo.verificationStatus": "AhdTLSInfoVerificationStatus", "TLSInfo.subject": "AhdTLSInfoSubject",
		"TLSInfo.issuer": "AhdTLSInfoIssuer", "TLSInfo.dnsNames": "AhdTLSInfoDNSNames",
		"TLSInfo.notBefore": "AhdTLSInfoNotBefore", "TLSInfo.notAfter": "AhdTLSInfoNotAfter",
		"TLSInfo.protocol": "AhdTLSInfoProtocol", "TLSInfo.cipherSuite": "AhdTLSInfoCipherSuite",
	}
	helper, known := helpers[name]
	if !known {
		return generator.unsupported("TLS operation "+name, meta.Span)
	}
	call := helper + "(" + generator.descriptorName(tlsErrorClass) + ", " +
		generator.smtpDataOf(tlsInfoClass, tlsInfoDataField, value.Callee) + ")"
	if name == "TLSInfo.notBefore" || name == "TLSInfo.notAfter" {
		return generator.dateTimeFrom(call, meta)
	}
	return call
}
