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
	switch name {
	case "passwordHash":
		return "AhdSecurityPasswordHash(" + errorClass + ", " + text(0) + ")"
	case "passwordVerify":
		return "AhdSecurityPasswordVerify(" + errorClass + ", " + text(0) + ", " + text(1) + ")"
	case "token":
		return "AhdSecurityToken(" + errorClass + ")"
	case "secureEqual":
		return "AhdSecuritySecureEqual(" + text(0) + ", " + text(1) + ")"
	default:
		return generator.unsupported("Security function "+name, meta.Span)
	}
}
