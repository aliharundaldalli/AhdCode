package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const (
	qrModulePrefix      = "builtin:QR::"
	barcodeModulePrefix = "builtin:Barcode::"
)

var (
	qrCodeClass          = ir.ClassID("builtin:QR::class::QRCode")
	qrErrorClass         = ir.ClassID("builtin:QR::class::QRError")
	qrCodeDataField      = ir.FieldID("builtin:QR::class::QRCode::field::data")
	barcodeCodeClass     = ir.ClassID("builtin:Barcode::class::BarcodeCode")
	barcodeErrorClass    = ir.ClassID("builtin:Barcode::class::BarcodeError")
	barcodeCodeDataField = ir.FieldID("builtin:Barcode::class::BarcodeCode::field::data")
)

// codesArgument renders one optional argument of the given kind, or fallback
// when the call omitted it.
func (generator *generator) codesArgument(value *ir.CallExpr, index int, kind ir.TypeKind, fallback string) string {
	if index >= len(value.Arguments) || value.Arguments[index].UsesDefault || value.Arguments[index].Value == nil {
		return fallback
	}
	return generator.value(value.Arguments[index].Value, ir.Type{Kind: kind}, false)
}

// qrCall lowers QR.create. A QRCode is a hidden String holding the validated
// level and value, built with the same constructor helper SMTPClient uses.
// Every function here reaches the vendored encoder, so the program receives
// the codes runtime.
func (generator *generator) qrCall(value *ir.CallExpr) string {
	generator.usesCodes = true
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), qrModulePrefix)
	errorClass := generator.descriptorName(qrErrorClass)
	switch name {
	case "create":
		return generator.smtpValueFrom(qrCodeClass, "AhdQRCreate("+errorClass+", "+
			generator.codesArgument(value, 0, ir.StringType, `""`)+", "+
			generator.codesArgument(value, 1, ir.StringType, `"M"`)+")", meta)
	default:
		return generator.unsupported("QR function "+name, meta.Span)
	}
}

// barcodeCall lowers Barcode.code128, Barcode.ean13, and Barcode.upca.
func (generator *generator) barcodeCall(value *ir.CallExpr) string {
	generator.usesCodes = true
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), barcodeModulePrefix)
	kinds := map[string]string{"code128": "Code128", "ean13": "EAN13", "upca": "UPCA"}
	kind, known := kinds[name]
	if !known {
		return generator.unsupported("Barcode function "+name, meta.Span)
	}
	return generator.smtpValueFrom(barcodeCodeClass, "AhdBarcodeCreate("+generator.descriptorName(barcodeErrorClass)+
		", \""+kind+"\", "+generator.codesArgument(value, 0, ir.StringType, `""`)+")", meta)
}

// codesOperation lowers the QRCode and BarcodeCode members. Omitted trailing
// arguments take the documented defaults: a 512-pixel or 5 cm QR image, and
// an 800 by 240 pixel or 8 by 2.4 cm barcode.
func (generator *generator) codesOperation(name string, value *ir.CallExpr) string {
	generator.usesCodes = true
	meta := value.ExprMeta()
	if strings.HasPrefix(name, "QRCode.") {
		errorClass := generator.descriptorName(qrErrorClass)
		data := generator.smtpDataOf(qrCodeClass, qrCodeDataField, value.Callee)
		switch name {
		case "QRCode.value":
			return "AhdQRValue(" + errorClass + ", " + data + ")"
		case "QRCode.level":
			return "AhdQRLevel(" + errorClass + ", " + data + ")"
		case "QRCode.size":
			return "AhdQRSize(" + errorClass + ", " + data + ")"
		case "QRCode.matrix":
			return "AhdQRMatrix(" + errorClass + ", " + data + ")"
		case "QRCode.savePNG":
			return "AhdQRSavePNG(" + errorClass + ", " + data + ", " + generator.codesArgument(value, 0, ir.StringType, `""`) + ", " +
				generator.codesArgument(value, 1, ir.IntType, "int64(512)") + ")"
		case "QRCode.saveSVG":
			return "AhdQRSaveSVG(" + errorClass + ", " + data + ", " + generator.codesArgument(value, 0, ir.StringType, `""`) + ", " +
				generator.codesArgument(value, 1, ir.RealType, "5.0") + ")"
		}
		return generator.unsupported("QRCode operation "+name, meta.Span)
	}
	errorClass := generator.descriptorName(barcodeErrorClass)
	data := generator.smtpDataOf(barcodeCodeClass, barcodeCodeDataField, value.Callee)
	switch name {
	case "BarcodeCode.kind":
		return "AhdBarcodeKind(" + errorClass + ", " + data + ")"
	case "BarcodeCode.value":
		return "AhdBarcodeValue(" + errorClass + ", " + data + ")"
	case "BarcodeCode.pattern":
		return "AhdBarcodePattern(" + errorClass + ", " + data + ")"
	case "BarcodeCode.savePNG":
		return "AhdBarcodeSavePNG(" + errorClass + ", " + data + ", " + generator.codesArgument(value, 0, ir.StringType, `""`) + ", " +
			generator.codesArgument(value, 1, ir.IntType, "int64(800)") + ", " + generator.codesArgument(value, 2, ir.IntType, "int64(240)") + ")"
	case "BarcodeCode.saveSVG":
		return "AhdBarcodeSaveSVG(" + errorClass + ", " + data + ", " + generator.codesArgument(value, 0, ir.StringType, `""`) + ", " +
			generator.codesArgument(value, 1, ir.RealType, "8.0") + ", " + generator.codesArgument(value, 2, ir.RealType, "2.4") + ")"
	}
	return generator.unsupported("BarcodeCode operation "+name, meta.Span)
}
