package semantic

import (
	"reflect"
	"testing"

	"ahdcode/internal/types"
)

const inspectionPreamble = `bring Disk
bring Service
from Disk bring (DiskInfo, DiskError)
from Service bring (ServiceInfo, ServiceError)
`

func TestSystemInspectionTypeChecks(t *testing.T) {
	requireSemanticClean(t, analyzeWithStandardModules(t, inspectionPreamble+`
disk: DiskInfo := Disk.inspect("/")
named: DiskInfo := Disk.inspect(path: "/var/www")
where: String := disk.path()
total: Int := disk.totalBytes()
used: Int := disk.usedBytes()
free: Int := disk.freeBytes()
available: Int := disk.availableBytes()
percent: Real := disk.usedPercent()
unit: ServiceInfo := Service.status("nginx.service")
byName: ServiceInfo := Service.status(name: "caddy.service")
unitName: String := unit.name()
active: String := unit.activeState()
sub: String := unit.subState()
running: Bool := unit.running()
enabled: Bool := unit.enabled()
attempt {
    Disk.inspect("/missing")
}
except DiskError as failure {
    write(failure.message)
}
attempt {
    Service.status("x")
}
except ServiceError as failure {
    write(failure.message)
}
`))
}

func TestSystemInspectionRejectsWrongUseAtCompileTime(t *testing.T) {
	setup := "disk: DiskInfo := Disk.inspect(\"/\")\nunit: ServiceInfo := Service.status(\"a\")\n"
	for _, source := range []string{
		`Disk.inspect(42)`,
		`Disk.inspect()`,
		`Disk.inspect("/", "/tmp")`,
		`Disk.usage("/")`,
		`Service.status(42)`,
		`Service.status()`,
		`Service.start("nginx")`,
		`Service.stop("nginx")`,
		`Service.restart("nginx")`,
		`Service.enable("nginx")`,
		`Service.disable("nginx")`,
		`Service.reload("nginx")`,
		`Service.status(name: "a", manager: "launchd")`,
		`text: String := disk.totalBytes()`,
		`whole: Int := disk.usedPercent()`,
		`disk.totalBytes(1)`,
		`unit.restart()`,
		`flag: String := unit.running()`,
		`DiskInfo("x")`,
		`ServiceInfo("x")`,
	} {
		t.Run(source, func(t *testing.T) {
			requireSemanticFailure(t, analyzeWithStandardModules(t, inspectionPreamble+setup+source+"\n"))
		})
	}
}

func TestSystemInspectionModulesExposeExactSurface(t *testing.T) {
	modules := StandardModuleInterfaces()
	if want := []string{"DiskError", "DiskInfo", "inspect"}; !reflect.DeepEqual(modules["Disk"].ExportNames, want) {
		t.Fatalf("Disk exports = %v", modules["Disk"].ExportNames)
	}
	if want := []string{"ServiceError", "ServiceInfo", "status"}; !reflect.DeepEqual(modules["Service"].ExportNames, want) {
		t.Fatalf("Service exports = %v (no start/stop/restart/enable/disable may exist)", modules["Service"].ExportNames)
	}
	for _, identity := range []*types.ClassSymbol{diskErrorClass, serviceErrorClass} {
		if identity.Parent == nil || identity.Parent.Name != "Error" {
			t.Fatalf("%s must derive from Error", identity.Name)
		}
	}
	for identity, want := range map[*types.ClassSymbol][]string{diskInfoClass: DiskInfoOperations, serviceInfoClass: ServiceInfoOperations} {
		members := BuiltinClassMembers(identity)
		if len(members) != len(want) {
			t.Fatalf("%s publishes %d members", identity.Name, len(members))
		}
		for _, member := range members {
			if member == nil || member.Callable == nil {
				t.Fatalf("%s has a member without a signature", identity.Name)
			}
		}
	}
}
