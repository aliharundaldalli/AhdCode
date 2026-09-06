package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const identityModulePrefix = "builtin:Identity::"

var identityErrorClass = ir.ClassID("builtin:Identity::class::IdentityError")

func (generator *generator) identityCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), identityModulePrefix)
	errorClass := generator.descriptorName(identityErrorClass)
	switch name {
	case "id":
		return "AhdIdentityID(" + errorClass + ")"
	default:
		return generator.unsupported("Identity function "+name, meta.Span)
	}
}
