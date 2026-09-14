package evaluator

import (
	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

// The UUID standard module's REPL implementation. It calls the native runtime's
// ahdruntime/uuid.go directly, so the evaluator and a compiled program share one
// parser, one formatter, and -- within the compiler process -- one version 7
// clock.

const evaluatorUUIDValueClass = ir.ClassID("builtin:UUID::class::UUIDValue")

var evaluatorUUIDValueData = ir.FieldID("builtin:UUID::class::UUIDValue::field::data")

func (session *Session) uuidValue(data string) *Instance {
	return &Instance{Class: evaluatorUUIDValueClass, Fields: map[ir.FieldID]any{evaluatorUUIDValueData: data}}
}

func (session *Session) uuidBuiltin(name string, args []any) any {
	defer session.codesRecover("UUIDError")
	class := ahdruntime.AhdClassUUIDError
	switch name {
	case "v4":
		return session.uuidValue(ahdruntime.AhdUUIDV4(class))
	case "v7":
		return session.uuidValue(ahdruntime.AhdUUIDV7(class))
	case "zero":
		return session.uuidValue(ahdruntime.AhdUUIDZero())
	case "parse":
		return session.uuidValue(ahdruntime.AhdUUIDParse(class, args[0].(string)))
	case "isValid":
		return ahdruntime.AhdUUIDIsValid(args[0].(string))
	}
	session.raise("Error", "unsupported UUID function "+name)
	return nil
}

func (session *Session) uuidData(value any) string {
	instance := session.requireInstance(value)
	if instance.Class != evaluatorUUIDValueClass {
		session.raise("Error", "value is not a UUIDValue")
	}
	data, ok := instance.Fields[evaluatorUUIDValueData].(string)
	if !ok {
		session.raise("UUIDError", "UUIDValue storage is corrupted")
	}
	return data
}

func (session *Session) uuidOperation(name string, receiver any, args []any) any {
	data := session.uuidData(receiver)
	defer session.codesRecover("UUIDError")
	class := ahdruntime.AhdClassUUIDError
	switch name {
	case "UUIDValue.string":
		return ahdruntime.AhdUUIDString(class, data)
	case "UUIDValue.version":
		return ahdruntime.AhdUUIDVersion(class, data)
	case "UUIDValue.isZero":
		return ahdruntime.AhdUUIDIsZero(class, data)
	case "UUIDValue.equals":
		return ahdruntime.AhdUUIDEquals(class, data, session.uuidData(args[0]))
	case "UUIDValue.compare":
		return ahdruntime.AhdUUIDCompare(class, data, session.uuidData(args[0]))
	}
	session.raise("Error", "unsupported UUIDValue operation "+name)
	return nil
}
