package evaluator

import (
	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

// The QR and Barcode standard modules' REPL implementation. It calls the
// native runtime's ahdruntime/codes.go directly, so the evaluator and a
// compiled program share one encoder, one PNG writer, and one SVG writer.

const (
	evaluatorQRCodeClass      = ir.ClassID("builtin:QR::class::QRCode")
	evaluatorBarcodeCodeClass = ir.ClassID("builtin:Barcode::class::BarcodeCode")
)

var (
	evaluatorQRCodeData      = ir.FieldID("builtin:QR::class::QRCode::field::data")
	evaluatorBarcodeCodeData = ir.FieldID("builtin:Barcode::class::BarcodeCode::field::data")
)

func (session *Session) qrBuiltin(name string, args []any) any {
	defer session.codesRecover("QRError")
	if name != "create" {
		session.raise("Error", "unsupported QR function "+name)
	}
	level := "M"
	if len(args) > 1 && args[1] != nil {
		level = args[1].(string)
	}
	return &Instance{Class: evaluatorQRCodeClass, Fields: map[ir.FieldID]any{
		evaluatorQRCodeData: ahdruntime.AhdQRCreate(ahdruntime.AhdClassQRError, args[0].(string), level),
	}}
}

func (session *Session) barcodeBuiltin(name string, args []any) any {
	defer session.codesRecover("BarcodeError")
	kinds := map[string]string{"code128": "Code128", "ean13": "EAN13", "upca": "UPCA"}
	kind, known := kinds[name]
	if !known {
		session.raise("Error", "unsupported Barcode function "+name)
	}
	return &Instance{Class: evaluatorBarcodeCodeClass, Fields: map[ir.FieldID]any{
		evaluatorBarcodeCodeData: ahdruntime.AhdBarcodeCreate(ahdruntime.AhdClassBarcodeError, kind, args[0].(string)),
	}}
}

func (session *Session) codesOperation(name string, receiver any, args []any) any {
	instance := session.requireInstance(receiver)
	integer := func(index int, fallback int64) int64 {
		if index < len(args) && args[index] != nil {
			return args[index].(int64)
		}
		return fallback
	}
	real := func(index int, fallback float64) float64 {
		if index < len(args) && args[index] != nil {
			return numericFloat(args[index])
		}
		return fallback
	}
	if instance.Class == evaluatorQRCodeClass {
		defer session.codesRecover("QRError")
		class := ahdruntime.AhdClassQRError
		data, ok := instance.Fields[evaluatorQRCodeData].(string)
		if !ok {
			session.raise("QRError", "QRCode storage is corrupted")
		}
		switch name {
		case "QRCode.value":
			return ahdruntime.AhdQRValue(class, data)
		case "QRCode.level":
			return ahdruntime.AhdQRLevel(class, data)
		case "QRCode.size":
			return ahdruntime.AhdQRSize(class, data)
		case "QRCode.matrix":
			rows := ahdruntime.AhdQRMatrix(class, data).Snapshot()
			items := make([]any, len(rows))
			for index, row := range rows {
				items[index] = booleanList(row.Snapshot())
			}
			return &List{Items: items}
		case "QRCode.savePNG":
			ahdruntime.AhdQRSavePNG(class, data, args[0].(string), integer(1, 512))
			return Nothing
		case "QRCode.saveSVG":
			ahdruntime.AhdQRSaveSVG(class, data, args[0].(string), real(1, 5.0))
			return Nothing
		}
		session.raise("Error", "unsupported QRCode operation "+name)
	}
	if instance.Class == evaluatorBarcodeCodeClass {
		defer session.codesRecover("BarcodeError")
		class := ahdruntime.AhdClassBarcodeError
		data, ok := instance.Fields[evaluatorBarcodeCodeData].(string)
		if !ok {
			session.raise("BarcodeError", "BarcodeCode storage is corrupted")
		}
		switch name {
		case "BarcodeCode.kind":
			return ahdruntime.AhdBarcodeKind(class, data)
		case "BarcodeCode.value":
			return ahdruntime.AhdBarcodeValue(class, data)
		case "BarcodeCode.pattern":
			return booleanList(ahdruntime.AhdBarcodePattern(class, data).Snapshot())
		case "BarcodeCode.savePNG":
			ahdruntime.AhdBarcodeSavePNG(class, data, args[0].(string), integer(1, 800), integer(2, 240))
			return Nothing
		case "BarcodeCode.saveSVG":
			ahdruntime.AhdBarcodeSaveSVG(class, data, args[0].(string), real(1, 8.0), real(2, 2.4))
			return Nothing
		}
		session.raise("Error", "unsupported BarcodeCode operation "+name)
	}
	session.raise("Error", "value is not a QRCode or BarcodeCode")
	return nil
}

func booleanList(values []bool) *List {
	items := make([]any, len(values))
	for index, value := range values {
		items[index] = value
	}
	return &List{Items: items}
}

// codesRecover turns a runtime signal into the evaluator's catchable module
// error.
func (session *Session) codesRecover(errorName string) {
	recovered := recover()
	if recovered == nil {
		return
	}
	if signal, ok := recovered.(*ahdruntime.AhdSignal); ok {
		session.raise(errorName, signal.Message)
	}
	panic(recovered)
}
