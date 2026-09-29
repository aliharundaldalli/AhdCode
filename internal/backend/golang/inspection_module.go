package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

// The v2.7.0 Disk and Service modules. Each call lowers to the shared
// ahdruntime function the evaluator also uses; DiskInfo and ServiceInfo are
// hidden-String handles emitted by emitSMTPHelpers.

const (
	diskModulePrefix    = "builtin:Disk::"
	serviceModulePrefix = "builtin:Service::"
)

var (
	diskInfoClass        = ir.ClassID("builtin:Disk::class::DiskInfo")
	diskInfoDataField    = ir.FieldID("builtin:Disk::class::DiskInfo::field::data")
	diskErrorClass       = ir.ClassID("builtin:Disk::class::DiskError")
	serviceInfoClass     = ir.ClassID("builtin:Service::class::ServiceInfo")
	serviceInfoDataField = ir.FieldID("builtin:Service::class::ServiceInfo::field::data")
	serviceErrorClass    = ir.ClassID("builtin:Service::class::ServiceError")
)

func (generator *generator) diskCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	if name := strings.TrimPrefix(string(value.Callable), diskModulePrefix); name != "inspect" {
		return generator.unsupported("Disk function "+name, meta.Span)
	}
	call := "AhdDiskInspect(" + generator.descriptorName(diskErrorClass) + ", " + generator.systemsText(value, 0) + ")"
	return generator.smtpValueFrom(diskInfoClass, call, meta)
}

func (generator *generator) serviceCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	if name := strings.TrimPrefix(string(value.Callable), serviceModulePrefix); name != "status" {
		return generator.unsupported("Service function "+name, meta.Span)
	}
	call := "AhdServiceStatus(" + generator.descriptorName(serviceErrorClass) + ", " + generator.systemsText(value, 0) + ")"
	return generator.smtpValueFrom(serviceInfoClass, call, meta)
}

func isInspectionOperation(name string) bool {
	return strings.HasPrefix(name, "DiskInfo.") || strings.HasPrefix(name, "ServiceInfo.")
}

// inspectionOperation lowers the DiskInfo and ServiceInfo accessors.
func (generator *generator) inspectionOperation(name string, value *ir.CallExpr) string {
	accessors := map[string]struct{ wrapper, reader string }{
		"DiskInfo.path":           {"AhdSystemString", "DiskInfoPath"},
		"DiskInfo.totalBytes":     {"AhdSystemInt", "DiskInfoTotalBytes"},
		"DiskInfo.usedBytes":      {"AhdSystemInt", "DiskInfoUsedBytes"},
		"DiskInfo.freeBytes":      {"AhdSystemInt", "DiskInfoFreeBytes"},
		"DiskInfo.availableBytes": {"AhdSystemInt", "DiskInfoAvailableBytes"},
		"DiskInfo.usedPercent":    {"AhdSystemReal", "DiskInfoUsedPercent"},
		"ServiceInfo.name":        {"AhdSystemString", "ServiceInfoName"},
		"ServiceInfo.activeState": {"AhdSystemString", "ServiceInfoActiveState"},
		"ServiceInfo.subState":    {"AhdSystemString", "ServiceInfoSubState"},
		"ServiceInfo.running":     {"AhdSystemBool", "ServiceInfoRunning"},
		"ServiceInfo.enabled":     {"AhdSystemBool", "ServiceInfoEnabled"},
	}
	accessor, known := accessors[name]
	if !known {
		return generator.unsupported("inspection operation "+name, value.ExprMeta().Span)
	}
	class, field, errorClass := diskInfoClass, diskInfoDataField, diskErrorClass
	if strings.HasPrefix(name, "ServiceInfo.") {
		class, field, errorClass = serviceInfoClass, serviceInfoDataField, serviceErrorClass
	}
	return accessor.wrapper + "(" + generator.descriptorName(errorClass) + ", " +
		generator.smtpDataOf(class, field, value.Callee) + ", " + accessor.reader + ")"
}
