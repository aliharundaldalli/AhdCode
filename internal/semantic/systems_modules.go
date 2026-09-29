package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

// The v2.5.0 deployment and systems primitives. Archive gains list/extract
// and the ArchiveEntry value, File gains copy, atomic writes and moves,
// symbolic links, permissions, and walk with the FileEntry value, SQLite gains
// Database.backupTo, and the new Process module runs one executable with an
// explicit argument list -- never through a shell.
//
// ArchiveEntry, FileEntry, and ProcessResult are immutable, opaque values that
// only their module produces. Their members are accessor methods that publish
// real signatures, so completion, hover, and signature help list them the way
// they list Canvas and Chart members.

const processModuleID = "builtin:Process"

var (
	archiveEntryClass  = &types.ClassSymbol{ModuleID: archiveModuleID, Name: "ArchiveEntry"}
	fileEntryClass     = &types.ClassSymbol{ModuleID: fileModuleID, Name: "FileEntry"}
	processResultClass = &types.ClassSymbol{ModuleID: processModuleID, Name: "ProcessResult"}
	processErrorClass  = &types.ClassSymbol{ModuleID: processModuleID, Name: "ProcessError",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
			Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}}
)

// Identities exposed to lowering without coupling the public module
// interfaces to a backend.
func ArchiveEntryIdentity() *types.ClassSymbol  { return archiveEntryClass }
func FileEntryIdentity() *types.ClassSymbol     { return fileEntryClass }
func ProcessResultIdentity() *types.ClassSymbol { return processResultClass }
func ProcessErrorIdentity() *types.ClassSymbol  { return processErrorClass }

const (
	ArchiveEntryPath TypeOperation = "ArchiveEntry.path"
	ArchiveEntryKind TypeOperation = "ArchiveEntry.kind"
	ArchiveEntrySize TypeOperation = "ArchiveEntry.size"

	FileEntryPath         TypeOperation = "FileEntry.path"
	FileEntryRelativePath TypeOperation = "FileEntry.relativePath"
	FileEntryKind         TypeOperation = "FileEntry.kind"
	FileEntrySize         TypeOperation = "FileEntry.size"
	FileEntryIsSymlink    TypeOperation = "FileEntry.isSymlink"

	ProcessResultExitCode TypeOperation = "ProcessResult.exitCode"
	ProcessResultStdout   TypeOperation = "ProcessResult.stdout"
	ProcessResultStderr   TypeOperation = "ProcessResult.stderr"
)

// The members each value publishes, in documentation order. Lowering builds
// the IR Classes from these same lists.
var (
	ArchiveEntryOperations  = []string{"path", "kind", "size"}
	FileEntryOperations     = []string{"path", "relativePath", "kind", "size", "isSymlink"}
	ProcessResultOperations = []string{"exitCode", "stdout", "stderr"}
)

var systemsMembers = map[TypeOperation]*Symbol{
	ArchiveEntryPath: completionMember(archiveModuleID, "path", types.String),
	ArchiveEntryKind: completionMember(archiveModuleID, "kind", types.String),
	ArchiveEntrySize: completionMember(archiveModuleID, "size", types.Int),

	FileEntryPath:         completionMember(fileModuleID, "path", types.String),
	FileEntryRelativePath: completionMember(fileModuleID, "relativePath", types.String),
	FileEntryKind:         completionMember(fileModuleID, "kind", types.String),
	FileEntrySize:         completionMember(fileModuleID, "size", types.Int),
	FileEntryIsSymlink:    completionMember(fileModuleID, "isSymlink", types.Bool),

	ProcessResultExitCode: completionMember(processModuleID, "exitCode", types.Int),
	ProcessResultStdout:   completionMember(processModuleID, "stdout", types.String),
	ProcessResultStderr:   completionMember(processModuleID, "stderr", types.String),
}

func systemsIdentity(identity *types.ClassSymbol) bool {
	return identity == archiveEntryClass || identity == fileEntryClass || identity == processResultClass ||
		identity == dnsResultClass || identity == tlsInfoClass || identity == diskInfoClass || identity == serviceInfoClass
}

// systemsOperationFor names the member an ArchiveEntry, FileEntry, or
// ProcessResult publishes. Only the compiler-supplied identities match, so a
// user Class with the same name is never affected.
func systemsOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || !systemsIdentity(class.Symbol) {
		return "", false
	}
	operation := TypeOperation(class.Symbol.Name + "." + name)
	_, known := systemsMembers[operation]
	return operation, known
}

func systemsMemberNames(identity *types.ClassSymbol) []string {
	switch identity {
	case archiveEntryClass:
		return ArchiveEntryOperations
	case fileEntryClass:
		return FileEntryOperations
	case processResultClass:
		return ProcessResultOperations
	case dnsResultClass:
		return DNSResultOperations
	case tlsInfoClass:
		return TLSInfoOperations
	case diskInfoClass:
		return DiskInfoOperations
	case serviceInfoClass:
		return ServiceInfoOperations
	}
	return nil
}

func systemsConstructionHint(identity *types.ClassSymbol) (string, bool) {
	switch identity {
	case archiveEntryClass:
		return "list an archive's entries with Archive.list(archive)", true
	case fileEntryClass:
		return "list a directory tree with File.walk(path)", true
	case processResultClass:
		return "run a program with Process.run(command, args)", true
	case dnsResultClass:
		return "resolve a host with DNS.lookup(host)", true
	case tlsInfoClass:
		return "inspect a TLS endpoint with TLS.inspect(host)", true
	case diskInfoClass:
		return "inspect a filesystem with Disk.inspect(path)", true
	case serviceInfoClass:
		return "inspect a systemd unit with Service.status(name)", true
	}
	return "", false
}

func systemsValueSymbol(moduleID string, identity *types.ClassSymbol) *Symbol {
	symbol := &Symbol{
		Name: identity.Name, Kind: ClassSymbol, Class: identity,
		Type: types.Class{Symbol: identity, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: moduleID,
		Members: make(map[string]*Symbol),
	}
	if identity.Parent != nil {
		symbol.Constructor = builtinErrorConstructor()
	}
	return symbol
}

func addSystemsClass(module *ModuleInterface, moduleID string, identity *types.ClassSymbol) {
	symbol := systemsValueSymbol(moduleID, identity)
	module.Classes[moduleID+"\x00"+identity.Name] = symbol
	addStandardExport(module, symbol)
}

// addArchiveReading publishes Archive.list, Archive.extract, and ArchiveEntry.
func addArchiveReading(module *ModuleInterface) {
	addSystemsClass(module, archiveModuleID, archiveEntryClass)
	archive := types.Parameter{Name: "archive", Type: types.String}
	maxFiles := types.Parameter{Name: "maxFiles", Type: types.Int, HasDefault: true}
	maxBytes := types.Parameter{Name: "maxBytes", Type: types.Int, HasDefault: true}
	addStandardExport(module, standardFunction(archiveModuleID, "list",
		types.List{Element: types.Class{Symbol: archiveEntryClass}}, archive, maxFiles))
	addStandardExport(module, standardFunction(archiveModuleID, "extract", types.Nothing,
		archive, types.Parameter{Name: "destination", Type: types.String}, maxFiles, maxBytes))
}

// addFileDeployment publishes the v2.5.0 File primitives and FileEntry.
func addFileDeployment(module *ModuleInterface) {
	addSystemsClass(module, fileModuleID, fileEntryClass)
	text := func(name string) types.Parameter { return types.Parameter{Name: name, Type: types.String} }
	addStandardExport(module, standardFunction(fileModuleID, "copy", types.Nothing, text("source"), text("destination")))
	addStandardExport(module, standardFunction(fileModuleID, "atomicWrite", types.Nothing, text("path"), text("content")))
	addStandardExport(module, standardFunction(fileModuleID, "atomicMove", types.Nothing, text("source"), text("destination")))
	addStandardExport(module, standardFunction(fileModuleID, "symlink", types.Nothing, text("target"), text("link")))
	addStandardExport(module, standardFunction(fileModuleID, "readLink", types.String, text("path")))
	addStandardExport(module, standardFunction(fileModuleID, "isSymlink", types.Bool, text("path")))
	addStandardExport(module, standardFunction(fileModuleID, "permissions", types.String, text("path")))
	addStandardExport(module, standardFunction(fileModuleID, "setPermissions", types.Nothing, text("path"), text("mode")))
	addStandardExport(module, standardFunction(fileModuleID, "walk",
		types.List{Element: types.Class{Symbol: fileEntryClass}}, text("path"),
		types.Parameter{Name: "maxEntries", Type: types.Int, HasDefault: true}))
}

func processModuleInterface() *ModuleInterface {
	module := standardInterface(processModuleID, "Process")
	addSystemsClass(module, processModuleID, processErrorClass)
	addSystemsClass(module, processModuleID, processResultClass)
	addStandardExport(module, standardFunction(processModuleID, "run", types.Class{Symbol: processResultClass},
		types.Parameter{Name: "command", Type: types.String},
		types.Parameter{Name: "args", Type: types.List{Element: types.String}, HasDefault: true},
		types.Parameter{Name: "timeoutSeconds", Type: types.Int, HasDefault: true},
		types.Parameter{Name: "maxOutputBytes", Type: types.Int, HasDefault: true}))
	sort.Strings(module.ExportNames)
	return module
}
