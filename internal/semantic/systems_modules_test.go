package semantic

import (
	"reflect"
	"strings"
	"testing"

	"ahdcode/internal/types"
)

const systemsPreamble = `bring Archive
bring File
bring Process
bring SQLite
from Archive bring (ArchiveEntry, ArchiveError)
from File bring (FileEntry, FileError)
from Process bring (ProcessResult, ProcessError)
from SQLite bring Database
`

func TestSystemsPrimitivesTypeCheck(t *testing.T) {
	result := analyzeWithStandardModules(t, systemsPreamble+`
entries: List<ArchiveEntry> := Archive.list("release.zip")
limited: List<ArchiveEntry> := Archive.list(archive: "release.zip", maxFiles: 50)
for entry in entries {
    path: Local String := entry.path()
    kind: Local String := entry.kind()
    size: Local Int := entry.size()
}
Archive.extract("release.zip", "staging")
Archive.extract(archive: "release.zip", destination: "staging", maxFiles: 5000, maxBytes: 1073741824)
Archive.extract(archive: "release.tar.gz", destination: "staging2", maxBytes: 1024)

File.copy("app.bin", "backup/app.bin")
File.copy(source: "app.bin", destination: "backup/app2.bin")
File.atomicWrite("config.json", r'{}')
File.atomicMove("staging/app", "current/app")
File.symlink(target: "releases/42", link: "current")
target: String := File.readLink("current")
linked: Bool := File.isSymlink("current")
mode: String := File.permissions("run.sh")
File.setPermissions("run.sh", "0755")
walked: List<FileEntry> := File.walk("site")
bounded: List<FileEntry> := File.walk(path: "site", maxEntries: 500)
for item in walked {
    a: Local String := item.path()
    b: Local String := item.relativePath()
    c: Local String := item.kind()
    d: Local Int := item.size()
    e: Local Bool := item.isSymlink()
}

result: ProcessResult := Process.run(
    command: "/usr/bin/systemctl",
    args: ["status", "example.service"],
    timeoutSeconds: 15,
    maxOutputBytes: 1048576
)
bare: ProcessResult := Process.run("/usr/bin/true")
positional: ProcessResult := Process.run("/bin/echo", ["hi"], 5, 1024)
code: Int := result.exitCode()
out: String := result.stdout()
err: String := result.stderr()
attempt {
    Process.run("/bin/false")
} except ProcessError as failure {
    write(failure.message)
}

db: Database := SQLite.open("panel.db")
db.backupTo("backup/panel.db")
`)
	requireSemanticClean(t, result)
}

func TestSystemsPrimitivesRejectWrongTypesAtCompileTime(t *testing.T) {
	setup := `entry: ArchiveEntry := Archive.list("a.zip")[0]
item: FileEntry := File.walk("site")[0]
result: ProcessResult := Process.run("/bin/true")
db: Database := SQLite.open(":memory:")
missing: String? := null
`
	tests := []string{
		`Process.run(command: 42)`,
		`Process.run(command: "/bin/echo", args: "hi")`,
		`Process.run(command: "/bin/echo", args: [1, 2])`,
		`Process.run(command: "/bin/echo", timeoutSeconds: "15")`,
		`Process.run(command: "/bin/echo", maxOutputBytes: 1.5)`,
		// Mixing positional and named arguments is refused earlier, by the
		// parser (PAR004), for every call including Process.run.
		`Process.run(command: "/bin/echo", shell: true)`,
		`Process.run()`,
		`Process.run(missing)`,
		`Process.shell("ls")`,
		`Archive.extract(archive: "a.zip", destination: "out", maxBytes: "large")`,
		`Archive.extract(archive: "a.zip", destination: "out", maxFiles: 1.5)`,
		`Archive.extract(archive: "a.zip", destination: "out", allowSymlinks: true)`,
		`Archive.extract("a.zip")`,
		`Archive.list(42)`,
		`Archive.unsafeExtract("a.zip", "out")`,
		`db.backupTo(42)`,
		`db.backupTo()`,
		`db.backupTo("a.db", "b.db")`,
		`db.backupTo(path: "a.db")`,
		`File.copy("a")`,
		`File.copy(1, 2)`,
		`File.copy(missing, "b")`,
		`File.atomicWrite("a", 1)`,
		`File.setPermissions("run.sh", 755)`,
		`File.setPermissions("run.sh", 0)`,
		`mode: Int := File.permissions("run.sh")`,
		`File.walk("site", "10")`,
		`File.walk(path: "site", followSymlinks: true)`,
		`File.symlink("target")`,
		`entry.path(1)`,
		`size: String := entry.size()`,
		`item.follow()`,
		`linked: String := item.isSymlink()`,
		`result.exitCode("x")`,
		`code: String := result.exitCode()`,
		`ProcessResult("x")`,
		`ArchiveEntry("x")`,
		`FileEntry("x")`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, systemsPreamble+setup+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestSystemsModulesExposeExactSurface(t *testing.T) {
	modules := StandardModuleInterfaces()
	if want := []string{"ProcessError", "ProcessResult", "run"}; !reflect.DeepEqual(modules["Process"].ExportNames, want) {
		t.Fatalf("Process exports = %v", modules["Process"].ExportNames)
	}
	if want := []string{"ArchiveEntry", "ArchiveError", "extract", "list", "tar", "tarGzip", "zip"}; !reflect.DeepEqual(modules["Archive"].ExportNames, want) {
		t.Fatalf("Archive exports = %v", modules["Archive"].ExportNames)
	}
	run := modules["Process"].Exports["run"].Callable.Signature
	var names []string
	for _, parameter := range run.Parameters {
		names = append(names, parameter.Name)
	}
	if strings.Join(names, ",") != "command,args,timeoutSeconds,maxOutputBytes" || run.Parameters[0].HasDefault || !run.Parameters[1].HasDefault {
		t.Fatalf("Process.run signature = %+v", run.Parameters)
	}
	if !types.Equal(run.Parameters[1].Type, types.List{Element: types.String}) {
		t.Fatalf("Process.run args type = %s", types.Display(run.Parameters[1].Type))
	}
	if processErrorClass.Parent == nil || processErrorClass.Parent.Name != "Error" {
		t.Fatal("ProcessError must derive from Error")
	}
	for _, identity := range []*types.ClassSymbol{archiveEntryClass, fileEntryClass, processResultClass} {
		members := BuiltinClassMembers(identity)
		if len(members) == 0 || len(members) != len(systemsMemberNames(identity)) {
			t.Fatalf("%s publishes %d members", identity.Name, len(members))
		}
		for _, member := range members {
			if member == nil || member.Callable == nil {
				t.Fatalf("%s has a member without a signature", identity.Name)
			}
		}
	}
	database := BuiltinClassMembers(sqliteDatabaseClass)
	found := false
	for _, member := range database {
		if member == nil {
			t.Fatal("Database completion has a nil member")
		}
		if member.Name == "backupTo" {
			found = true
		}
	}
	if !found {
		t.Fatal("Database completion lacks backupTo")
	}
}
