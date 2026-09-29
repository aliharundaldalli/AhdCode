package evaluator

// The v2.7.0 Disk and Service modules in the persistent evaluator. Every call
// goes to the same ahdruntime function a native program's wrapper calls. A
// relative Disk path is resolved against the session directory and reported
// as the program wrote it.

import (
	"strings"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

var (
	evaluatorDiskInfoClass    = ir.ClassID("builtin:Disk::class::DiskInfo")
	evaluatorServiceInfoClass = ir.ClassID("builtin:Service::class::ServiceInfo")
	evaluatorDiskInfoField    = ir.FieldID("builtin:Disk::class::DiskInfo::field::data")
	evaluatorServiceInfoField = ir.FieldID("builtin:Service::class::ServiceInfo::field::data")
)

func (session *Session) diskBuiltin(name string, args []any) any {
	if name != "inspect" {
		session.raise("Error", "unsupported Disk function "+name)
		return nil
	}
	path := session.systemsPathOf(args[0])
	data, err := ahdruntime.DiskInspectAs(path.resolved, path.written)
	session.systemsRaise("DiskError", err, path)
	return &Instance{Class: evaluatorDiskInfoClass, Fields: map[ir.FieldID]any{evaluatorDiskInfoField: data}}
}

func (session *Session) serviceBuiltin(name string, args []any) any {
	if name != "status" {
		session.raise("Error", "unsupported Service function "+name)
		return nil
	}
	data, err := ahdruntime.ServiceStatus(args[0].(string))
	session.systemsRaise("ServiceError", err)
	return &Instance{Class: evaluatorServiceInfoClass, Fields: map[ir.FieldID]any{evaluatorServiceInfoField: data}}
}

func isInspectionOperation(name string) bool {
	return strings.HasPrefix(name, "DiskInfo.") || strings.HasPrefix(name, "ServiceInfo.")
}

func (session *Session) inspectionOperation(name string, receiver any) any {
	var value any
	var err error
	if strings.HasPrefix(name, "DiskInfo.") {
		data := session.systemsDataOf(receiver, evaluatorDiskInfoClass, evaluatorDiskInfoField, "DiskError", "DiskInfo")
		switch name {
		case "DiskInfo.path":
			value, err = ahdruntime.DiskInfoPath(data)
		case "DiskInfo.totalBytes":
			value, err = ahdruntime.DiskInfoTotalBytes(data)
		case "DiskInfo.usedBytes":
			value, err = ahdruntime.DiskInfoUsedBytes(data)
		case "DiskInfo.freeBytes":
			value, err = ahdruntime.DiskInfoFreeBytes(data)
		case "DiskInfo.availableBytes":
			value, err = ahdruntime.DiskInfoAvailableBytes(data)
		case "DiskInfo.usedPercent":
			value, err = ahdruntime.DiskInfoUsedPercent(data)
		default:
			session.raise("Error", "unsupported Disk operation "+name)
		}
		session.systemsRaise("DiskError", err)
		return value
	}
	data := session.systemsDataOf(receiver, evaluatorServiceInfoClass, evaluatorServiceInfoField, "ServiceError", "ServiceInfo")
	switch name {
	case "ServiceInfo.name":
		value, err = ahdruntime.ServiceInfoName(data)
	case "ServiceInfo.activeState":
		value, err = ahdruntime.ServiceInfoActiveState(data)
	case "ServiceInfo.subState":
		value, err = ahdruntime.ServiceInfoSubState(data)
	case "ServiceInfo.running":
		value, err = ahdruntime.ServiceInfoRunning(data)
	case "ServiceInfo.enabled":
		value, err = ahdruntime.ServiceInfoEnabled(data)
	default:
		session.raise("Error", "unsupported Service operation "+name)
	}
	session.systemsRaise("ServiceError", err)
	return value
}
