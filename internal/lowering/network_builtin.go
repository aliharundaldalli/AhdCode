package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// DNSModuleID and TLSModuleID are the compiler-supplied identities of the
// v2.6.0 network-inspection modules.
const (
	DNSModuleID = "builtin:DNS"
	TLSModuleID = "builtin:TLS"
)

const (
	dnsResultClassID = ir.ClassID(DNSModuleID + "::class::DNSResult")
	dnsErrorClassID  = ir.ClassID(DNSModuleID + "::class::DNSError")
	tlsInfoClassID   = ir.ClassID(TLSModuleID + "::class::TLSInfo")
	tlsErrorClassID  = ir.ClassID(TLSModuleID + "::class::TLSError")
)

// DNSResultDataFieldID and TLSInfoDataFieldID are the hidden Strings holding
// each value's encoded inspection result.
var (
	DNSResultDataFieldID = ir.FieldID(string(dnsResultClassID) + "::field::data")
	TLSInfoDataFieldID   = ir.FieldID(string(tlsInfoClassID) + "::field::data")
)

func networkModule(id ir.ModuleID, name, path string, valueID ir.ClassID, valueName string, field ir.FieldID, operations []string, errorID ir.ClassID, errorName string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	addHiddenStringValueClass(module, valueID, valueName, field, operations)
	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: errorID, Symbol: ir.SymbolID(string(errorID) + "::symbol"),
		Name: errorName, Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(errorID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}

func dnsModule(id ir.ModuleID, name, path string) *ir.Module {
	return networkModule(id, name, path, dnsResultClassID, "DNSResult", DNSResultDataFieldID, semantic.DNSResultOperations, dnsErrorClassID, "DNSError")
}

func tlsModule(id ir.ModuleID, name, path string) *ir.Module {
	return networkModule(id, name, path, tlsInfoClassID, "TLSInfo", TLSInfoDataFieldID, semantic.TLSInfoOperations, tlsErrorClassID, "TLSError")
}

// DiskModuleID and ServiceModuleID are the compiler-supplied identities of the
// v2.7.0 read-only system-inspection modules.
const (
	DiskModuleID    = "builtin:Disk"
	ServiceModuleID = "builtin:Service"
)

const (
	diskInfoClassID     = ir.ClassID(DiskModuleID + "::class::DiskInfo")
	diskErrorClassID    = ir.ClassID(DiskModuleID + "::class::DiskError")
	serviceInfoClassID  = ir.ClassID(ServiceModuleID + "::class::ServiceInfo")
	serviceErrorClassID = ir.ClassID(ServiceModuleID + "::class::ServiceError")
)

var (
	DiskInfoDataFieldID    = ir.FieldID(string(diskInfoClassID) + "::field::data")
	ServiceInfoDataFieldID = ir.FieldID(string(serviceInfoClassID) + "::field::data")
)

func diskModule(id ir.ModuleID, name, path string) *ir.Module {
	return networkModule(id, name, path, diskInfoClassID, "DiskInfo", DiskInfoDataFieldID, semantic.DiskInfoOperations, diskErrorClassID, "DiskError")
}

func serviceModule(id ir.ModuleID, name, path string) *ir.Module {
	return networkModule(id, name, path, serviceInfoClassID, "ServiceInfo", ServiceInfoDataFieldID, semantic.ServiceInfoOperations, serviceErrorClassID, "ServiceError")
}
