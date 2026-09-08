package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const securityModulePrefix = "builtin:Security::"

var securityErrorClass = ir.ClassID("builtin:Security::class::SecurityError")

// securityCall lowers the Security module's functions. Like Env, Security
// publishes no data-carrying Class — every call maps to a plain String/Bool
// runtime function. The runtime implementation lives in ahdruntime/security.go
// which is embedded alongside the other runtime files.
func (generator *generator) securityCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), securityModulePrefix)
	errorClass := generator.descriptorName(securityErrorClass)
	text := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	number := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
	}
	switch name {
	case "passwordHash":
		return "AhdSecurityPasswordHash(" + errorClass + ", " + text(0) + ")"
	case "passwordVerify":
		return "AhdSecurityPasswordVerify(" + errorClass + ", " + text(0) + ", " + text(1) + ")"
	case "token":
		return "AhdSecurityToken(" + errorClass + ")"
	case "secureEqual":
		return "AhdSecuritySecureEqual(" + text(0) + ", " + text(1) + ")"
	case "sha256":
		return "AhdSecuritySHA256(" + text(0) + ")"
	case "sha512":
		return "AhdSecuritySHA512(" + text(0) + ")"
	case "hmacSHA256":
		return "AhdSecurityHMACSHA256(" + text(0) + ", " + text(1) + ")"
	case "hmacVerify":
		return "AhdSecurityHMACVerify(" + text(0) + ", " + text(1) + ", " + text(2) + ")"
	case "base64Encode":
		return "AhdSecurityBase64Encode(" + text(0) + ")"
	case "base64Decode":
		return "AhdSecurityBase64Decode(" + errorClass + ", " + text(0) + ")"
	case "base64UrlEncode":
		return "AhdSecurityBase64UrlEncode(" + text(0) + ")"
	case "base64UrlDecode":
		return "AhdSecurityBase64UrlDecode(" + errorClass + ", " + text(0) + ")"
	case "hexEncode":
		return "AhdSecurityHexEncode(" + text(0) + ")"
	case "hexDecode":
		return "AhdSecurityHexDecode(" + errorClass + ", " + text(0) + ")"
	case "randomHex":
		return "AhdSecurityRandomHex(" + errorClass + ", " + number(0) + ")"
	case "rsaSignSHA256":
		return "AhdSecurityRSASignSHA256(" + errorClass + ", " + text(0) + ", " + text(1) + ")"
	case "rsaVerifySHA256":
		return "AhdSecurityRSAVerifySHA256(" + errorClass + ", " + text(0) + ", " + text(1) + ", " + text(2) + ")"
	case "aesEncrypt":
		return "AhdSecurityAESGCMEncrypt(" + errorClass + ", " + text(0) + ", " + text(1) + ")"
	case "aesDecrypt":
		return "AhdSecurityAESGCMDecrypt(" + errorClass + ", " + text(0) + ", " + text(1) + ")"
	default:
		return generator.unsupported("Security function "+name, meta.Span)
	}
}
