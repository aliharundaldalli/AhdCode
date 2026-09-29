package evaluator

// The v2.6.0 DNS and TLS modules in the persistent evaluator. Every call goes
// to the same ahdruntime function a native program's wrapper calls, so input
// validation, bounds, verification, and messages cannot diverge.

import (
	"strings"
	"time"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

var (
	evaluatorDNSResultClass = ir.ClassID("builtin:DNS::class::DNSResult")
	evaluatorTLSInfoClass   = ir.ClassID("builtin:TLS::class::TLSInfo")
	evaluatorDNSResultField = ir.FieldID("builtin:DNS::class::DNSResult::field::data")
	evaluatorTLSInfoField   = ir.FieldID("builtin:TLS::class::TLSInfo::field::data")
)

func (session *Session) dnsBuiltin(name string, args []any) any {
	if name != "lookup" {
		session.raise("Error", "unsupported DNS function "+name)
		return nil
	}
	data, err := ahdruntime.DNSLookup(args[0].(string), session.systemsIntArg(args, 1, ahdruntime.AhdDNSDefaultTimeoutSeconds))
	session.systemsRaise("DNSError", err)
	return &Instance{Class: evaluatorDNSResultClass, Fields: map[ir.FieldID]any{evaluatorDNSResultField: data}}
}

func (session *Session) tlsBuiltin(name string, args []any) any {
	if name != "inspect" {
		session.raise("Error", "unsupported TLS function "+name)
		return nil
	}
	data, err := ahdruntime.TLSInspect(args[0].(string),
		session.systemsIntArg(args, 1, ahdruntime.AhdTLSDefaultPort),
		session.systemsIntArg(args, 2, ahdruntime.AhdTLSDefaultTimeoutSeconds))
	session.systemsRaise("TLSError", err)
	return &Instance{Class: evaluatorTLSInfoClass, Fields: map[ir.FieldID]any{evaluatorTLSInfoField: data}}
}

func isNetworkOperation(name string) bool {
	return strings.HasPrefix(name, "DNSResult.") || strings.HasPrefix(name, "TLSInfo.")
}

func (session *Session) networkStrings(values []string) *List {
	items := make([]any, len(values))
	for index, value := range values {
		items[index] = value
	}
	return &List{Items: items}
}

func (session *Session) networkOperation(name string, receiver any) any {
	if strings.HasPrefix(name, "DNSResult.") {
		data := session.systemsDataOf(receiver, evaluatorDNSResultClass, evaluatorDNSResultField, "DNSError", "DNSResult")
		var values []string
		var err error
		switch name {
		case "DNSResult.host":
			host, hostErr := ahdruntime.DNSResultHost(data)
			session.systemsRaise("DNSError", hostErr)
			return host
		case "DNSResult.addresses":
			values, err = ahdruntime.DNSResultAddresses(data)
		case "DNSResult.ipv4":
			values, err = ahdruntime.DNSResultIPv4(data)
		case "DNSResult.ipv6":
			values, err = ahdruntime.DNSResultIPv6(data)
		default:
			session.raise("Error", "unsupported DNS operation "+name)
		}
		session.systemsRaise("DNSError", err)
		return session.networkStrings(values)
	}
	data := session.systemsDataOf(receiver, evaluatorTLSInfoClass, evaluatorTLSInfoField, "TLSError", "TLSInfo")
	text := func(read func(string) (string, error)) any {
		value, err := read(data)
		session.systemsRaise("TLSError", err)
		return value
	}
	instant := func(read func(string) (int64, error)) any {
		milliseconds, err := read(data)
		session.systemsRaise("TLSError", err)
		return session.dateTime(time.UnixMilli(milliseconds).UTC())
	}
	switch name {
	case "TLSInfo.host":
		return text(ahdruntime.TLSInfoHost)
	case "TLSInfo.verificationStatus":
		return text(ahdruntime.TLSInfoVerificationStatus)
	case "TLSInfo.subject":
		return text(ahdruntime.TLSInfoSubject)
	case "TLSInfo.issuer":
		return text(ahdruntime.TLSInfoIssuer)
	case "TLSInfo.protocol":
		return text(ahdruntime.TLSInfoProtocol)
	case "TLSInfo.cipherSuite":
		return text(ahdruntime.TLSInfoCipherSuite)
	case "TLSInfo.port":
		value, err := ahdruntime.TLSInfoPort(data)
		session.systemsRaise("TLSError", err)
		return value
	case "TLSInfo.valid":
		value, err := ahdruntime.TLSInfoValid(data)
		session.systemsRaise("TLSError", err)
		return value
	case "TLSInfo.dnsNames":
		values, err := ahdruntime.TLSInfoDNSNames(data)
		session.systemsRaise("TLSError", err)
		return session.networkStrings(values)
	case "TLSInfo.notBefore":
		return instant(ahdruntime.TLSInfoNotBefore)
	case "TLSInfo.notAfter":
		return instant(ahdruntime.TLSInfoNotAfter)
	}
	session.raise("Error", "unsupported TLS operation "+name)
	return nil
}
