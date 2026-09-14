package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const uuidModulePrefix = "builtin:UUID::"

var (
	uuidValueClass     = ir.ClassID("builtin:UUID::class::UUIDValue")
	uuidErrorClass     = ir.ClassID("builtin:UUID::class::UUIDError")
	uuidValueDataField = ir.FieldID("builtin:UUID::class::UUIDValue::field::data")
)

// uuidCall lowers the UUID module functions. A UUIDValue is a hidden String
// holding canonical lowercase text, built with the same constructor helper
// QRCode uses. The UUID runtime is standard library only, so no program
// requirement flag is set.
func (generator *generator) uuidCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), uuidModulePrefix)
	errorClass := generator.descriptorName(uuidErrorClass)
	switch name {
	case "v4":
		return generator.smtpValueFrom(uuidValueClass, "AhdUUIDV4("+errorClass+")", meta)
	case "v7":
		return generator.smtpValueFrom(uuidValueClass, "AhdUUIDV7("+errorClass+")", meta)
	case "zero":
		return generator.smtpValueFrom(uuidValueClass, "AhdUUIDZero()", meta)
	case "parse":
		return generator.smtpValueFrom(uuidValueClass, "AhdUUIDParse("+errorClass+", "+
			generator.codesArgument(value, 0, ir.StringType, `""`)+")", meta)
	case "isValid":
		return "AhdUUIDIsValid(" + generator.codesArgument(value, 0, ir.StringType, `""`) + ")"
	default:
		return generator.unsupported("UUID function "+name, meta.Span)
	}
}

// uuidOperation lowers the UUIDValue members. equals and compare read the
// hidden text of their UUIDValue argument the same way as the receiver's.
func (generator *generator) uuidOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	errorClass := generator.descriptorName(uuidErrorClass)
	data := generator.smtpDataOf(uuidValueClass, uuidValueDataField, value.Callee)
	switch name {
	case "UUIDValue.string":
		return "AhdUUIDString(" + errorClass + ", " + data + ")"
	case "UUIDValue.version":
		return "AhdUUIDVersion(" + errorClass + ", " + data + ")"
	case "UUIDValue.isZero":
		return "AhdUUIDIsZero(" + errorClass + ", " + data + ")"
	case "UUIDValue.equals", "UUIDValue.compare":
		if len(value.Arguments) != 1 || value.Arguments[0].Value == nil {
			return generator.unsupported("UUIDValue operation "+name+" with malformed arguments", meta.Span)
		}
		other := generator.smtpDataOf(uuidValueClass, uuidValueDataField, value.Arguments[0].Value)
		if name == "UUIDValue.equals" {
			return "AhdUUIDEquals(" + errorClass + ", " + data + ", " + other + ")"
		}
		return "AhdUUIDCompare(" + errorClass + ", " + data + ", " + other + ")"
	}
	return generator.unsupported("UUIDValue operation "+name, meta.Span)
}
