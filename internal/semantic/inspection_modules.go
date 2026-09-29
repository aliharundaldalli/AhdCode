package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

// The v2.7.0 read-only system-inspection modules. Disk.inspect reports the
// capacity of the filesystem that contains a path; Service.status reports
// the state of a Linux systemd unit. Both return immutable, opaque values
// only their module produces, and neither changes anything on the system.

const (
	diskModuleID    = "builtin:Disk"
	serviceModuleID = "builtin:Service"
)

var (
	diskInfoClass     = &types.ClassSymbol{ModuleID: diskModuleID, Name: "DiskInfo"}
	diskErrorClass    = &types.ClassSymbol{ModuleID: diskModuleID, Name: "DiskError", Parent: networkErrorParent}
	serviceInfoClass  = &types.ClassSymbol{ModuleID: serviceModuleID, Name: "ServiceInfo"}
	serviceErrorClass = &types.ClassSymbol{ModuleID: serviceModuleID, Name: "ServiceError", Parent: networkErrorParent}
)

func DiskInfoIdentity() *types.ClassSymbol     { return diskInfoClass }
func DiskErrorIdentity() *types.ClassSymbol    { return diskErrorClass }
func ServiceInfoIdentity() *types.ClassSymbol  { return serviceInfoClass }
func ServiceErrorIdentity() *types.ClassSymbol { return serviceErrorClass }

// The members each value publishes, in documentation order. Lowering builds
// the IR Classes from these same lists.
var (
	DiskInfoOperations    = []string{"path", "totalBytes", "usedBytes", "freeBytes", "availableBytes", "usedPercent"}
	ServiceInfoOperations = []string{"name", "activeState", "subState", "running", "enabled"}
)

func init() {
	for name, result := range map[string]types.Type{
		"path": types.String, "totalBytes": types.Int, "usedBytes": types.Int, "freeBytes": types.Int,
		"availableBytes": types.Int, "usedPercent": types.Real,
	} {
		systemsMembers[TypeOperation("DiskInfo."+name)] = completionMember(diskModuleID, name, result)
	}
	for name, result := range map[string]types.Type{
		"name": types.String, "activeState": types.String, "subState": types.String, "running": types.Bool, "enabled": types.Bool,
	} {
		systemsMembers[TypeOperation("ServiceInfo."+name)] = completionMember(serviceModuleID, name, result)
	}
}

func diskModuleInterface() *ModuleInterface {
	module := standardInterface(diskModuleID, "Disk")
	addSystemsClass(module, diskModuleID, diskErrorClass)
	addSystemsClass(module, diskModuleID, diskInfoClass)
	addStandardExport(module, standardFunction(diskModuleID, "inspect", types.Class{Symbol: diskInfoClass},
		types.Parameter{Name: "path", Type: types.String}))
	sort.Strings(module.ExportNames)
	return module
}

func serviceModuleInterface() *ModuleInterface {
	module := standardInterface(serviceModuleID, "Service")
	addSystemsClass(module, serviceModuleID, serviceErrorClass)
	addSystemsClass(module, serviceModuleID, serviceInfoClass)
	addStandardExport(module, standardFunction(serviceModuleID, "status", types.Class{Symbol: serviceInfoClass},
		types.Parameter{Name: "name", Type: types.String}))
	sort.Strings(module.ExportNames)
	return module
}
