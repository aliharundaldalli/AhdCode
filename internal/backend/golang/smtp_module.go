package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const smtpModulePrefix = "builtin:SMTP::"

var (
	smtpClientClass       = ir.ClassID("builtin:SMTP::class::SMTPClient")
	smtpMessageClass      = ir.ClassID("builtin:SMTP::class::SMTPMessage")
	smtpErrorClass        = ir.ClassID("builtin:SMTP::class::SMTPError")
	smtpClientHandleField = ir.FieldID("builtin:SMTP::class::SMTPClient::field::handle")
	smtpMessageDataField  = ir.FieldID("builtin:SMTP::class::SMTPMessage::field::data")
)

func (generator *generator) smtpCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), smtpModulePrefix)
	errorClass := generator.descriptorName(smtpErrorClass)
	text := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	integer := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
	}
	switch name {
	case "client":
		return generator.smtpValueFrom(smtpClientClass, "AhdSMTPClient("+errorClass+", "+
			text(0, `""`)+", "+integer(1, "int64(0)")+", "+text(2, `"starttls"`)+", "+
			integer(3, "int64(30)")+")", meta)
	case "message":
		return generator.smtpValueFrom(smtpMessageClass, "AhdSMTPMessage("+errorClass+", "+
			text(0, `""`)+", "+generator.smtpStringList(value, 1)+", "+text(2, `""`)+")", meta)
	default:
		return generator.unsupported("SMTP function "+name, meta.Span)
	}
}

func (generator *generator) smtpOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	errorClass := generator.descriptorName(smtpErrorClass)
	text := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	switch name {
	case "SMTPClient.withPlainAuth":
		return generator.smtpValueFrom(smtpClientClass, "AhdSMTPClientWithPlainAuth("+errorClass+", "+
			generator.smtpDataOf(smtpClientClass, smtpClientHandleField, value.Callee)+", "+
			text(0)+", "+text(1)+")", meta)
	case "SMTPClient.send":
		return "AhdSMTPClientSend(" + errorClass + ", " +
			generator.smtpDataOf(smtpClientClass, smtpClientHandleField, value.Callee) + ", " +
			generator.smtpDataOf(smtpMessageClass, smtpMessageDataField, value.Arguments[0].Value) + ")"
	case "SMTPMessage.withCc":
		return generator.smtpValueFrom(smtpMessageClass, "AhdSMTPMessageWithCc("+errorClass+", "+
			generator.smtpDataOf(smtpMessageClass, smtpMessageDataField, value.Callee)+", "+
			generator.smtpStringList(value, 0)+")", meta)
	case "SMTPMessage.withBcc":
		return generator.smtpValueFrom(smtpMessageClass, "AhdSMTPMessageWithBcc("+errorClass+", "+
			generator.smtpDataOf(smtpMessageClass, smtpMessageDataField, value.Callee)+", "+
			generator.smtpStringList(value, 0)+")", meta)
	case "SMTPMessage.withReplyTo":
		return generator.smtpValueFrom(smtpMessageClass, "AhdSMTPMessageWithReplyTo("+errorClass+", "+
			generator.smtpDataOf(smtpMessageClass, smtpMessageDataField, value.Callee)+", "+
			text(0)+")", meta)
	case "SMTPMessage.withText":
		return generator.smtpValueFrom(smtpMessageClass, "AhdSMTPMessageWithText("+errorClass+", "+
			generator.smtpDataOf(smtpMessageClass, smtpMessageDataField, value.Callee)+", "+
			text(0)+")", meta)
	case "SMTPMessage.withHtml":
		return generator.smtpValueFrom(smtpMessageClass, "AhdSMTPMessageWithHtml("+errorClass+", "+
			generator.smtpDataOf(smtpMessageClass, smtpMessageDataField, value.Callee)+", "+
			text(0)+")", meta)
	default:
		return generator.unsupported("SMTP operation "+name, meta.Span)
	}
}

func (generator *generator) smtpStringList(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "nil"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	return "func(list *AhdList[string]) []string { if list == nil { return nil }; return append([]string(nil), list.Snapshot()...) }(" + rendered + ")"
}

func (generator *generator) smtpValueFrom(class ir.ClassID, data string, meta ir.ExprBase) string {
	helper, ok := generator.smtpHelper(class)
	if !ok {
		return generator.unsupported("an SMTP value without its Class declaration", meta.Span)
	}
	return helper + "(" + data + ")"
}

func (generator *generator) smtpDataOf(class ir.ClassID, field ir.FieldID, expression ir.Expr) string {
	rendered := generator.expr(expression)
	getter := generator.fieldName(field) + "_get()"
	return "func(value " + generator.interfaceName(class) + ") string { return value." + getter + " }(" + rendered + ")"
}

func (generator *generator) smtpHelper(class ir.ClassID) (string, bool) {
	if generator.layouts[class] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[class]; known {
		return name, true
	}
	name := mangleNamed("smtp_", generator.classDisplayName(class), string(class))
	generator.timeHelpers[class] = name
	return name, true
}

func (generator *generator) emitSMTPHelpers(writer *emitter) {
	for _, class := range []ir.ClassID{smtpClientClass, smtpMessageClass} {
		name, known := generator.timeHelpers[class]
		if !known {
			continue
		}
		layout := generator.layouts[class]
		if layout == nil {
			continue
		}
		constructor := generator.functions[layout.class.Constructor]
		if constructor == nil {
			continue
		}
		writer.open("func " + name + "(data string) " + generator.interfaceName(class) + " {")
		writer.line("return " + generator.callableName(constructor) + "(data)")
		writer.close("}")
		writer.blank()
	}
}
