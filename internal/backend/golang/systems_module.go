package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

// The v2.5.0 systems primitives: Archive.list/extract, the File deployment
// functions, Process.run, SQLite's Database.backupTo, and the ArchiveEntry,
// FileEntry, and ProcessResult accessors. Each lowers to one call of the
// shared ahdruntime function the evaluator also uses. The three value Classes
// are hidden-String handles emitted by emitSMTPHelpers.

const processModulePrefix = "builtin:Process::"

var (
	archiveEntryClass       = ir.ClassID("builtin:Archive::class::ArchiveEntry")
	archiveEntryDataField   = ir.FieldID("builtin:Archive::class::ArchiveEntry::field::data")
	archiveErrorClass       = ir.ClassID("builtin:Archive::class::ArchiveError")
	fileEntryClass          = ir.ClassID("builtin:File::class::FileEntry")
	fileEntryDataField      = ir.FieldID("builtin:File::class::FileEntry::field::data")
	processResultClass      = ir.ClassID("builtin:Process::class::ProcessResult")
	processResultDataField  = ir.FieldID("builtin:Process::class::ProcessResult::field::data")
	processErrorClass       = ir.ClassID("builtin:Process::class::ProcessError")
	systemsDefaultArguments = map[string]string{
		"Archive.maxFiles":       "AhdArchiveDefaultMaxFiles",
		"Archive.maxBytes":       "AhdArchiveDefaultMaxBytes",
		"File.maxEntries":        "AhdFileWalkDefaultMaxEntries",
		"Process.timeoutSeconds": "AhdProcessDefaultTimeoutSeconds",
		"Process.maxOutputBytes": "AhdProcessDefaultMaxOutputBytes",
	}
)

func (generator *generator) systemsText(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return `""`
	}
	return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
}

// systemsInt renders an Int argument, or the documented runtime default when
// the call omitted it.
func (generator *generator) systemsInt(value *ir.CallExpr, index int, fallback string) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return systemsDefaultArguments[fallback]
	}
	return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
}

// systemsValueList wraps a runtime []string of encoded values into the
// List<Class> the program sees.
func (generator *generator) systemsValueList(class ir.ClassID, data string, meta ir.ExprBase) string {
	helper, ok := generator.smtpHelper(class)
	if !ok {
		return generator.unsupported("a systems value without its Class declaration", meta.Span)
	}
	element := generator.interfaceName(class)
	return "func(items []string) *AhdList[" + element + "] { result := make([]" + element + ", len(items)); " +
		"for index, item := range items { result[index] = " + helper + "(item) }; " +
		"return AhdNewList(result...) }(" + data + ")"
}

// archiveReadingCall lowers Archive.list and Archive.extract.
func (generator *generator) archiveReadingCall(name string, value *ir.CallExpr) (string, bool) {
	meta := value.ExprMeta()
	errorClass := generator.descriptorName(archiveErrorClass)
	switch name {
	case "list":
		return generator.systemsValueList(archiveEntryClass, "AhdArchiveList("+errorClass+", "+generator.systemsText(value, 0)+", "+
			generator.systemsInt(value, 1, "Archive.maxFiles")+")", meta), true
	case "extract":
		return "AhdArchiveExtract(" + errorClass + ", " + generator.systemsText(value, 0) + ", " + generator.systemsText(value, 1) + ", " +
			generator.systemsInt(value, 2, "Archive.maxFiles") + ", " + generator.systemsInt(value, 3, "Archive.maxBytes") + ")", true
	}
	return "", false
}

// fileDeploymentCall lowers the v2.5.0 File functions.
func (generator *generator) fileDeploymentCall(name string, value *ir.CallExpr) (string, bool) {
	meta := value.ExprMeta()
	errorClass := generator.descriptorName(fileErrorClass)
	text := func(index int) string { return generator.systemsText(value, index) }
	switch name {
	case "copy":
		return "AhdFileCopy(" + errorClass + ", " + text(0) + ", " + text(1) + ")", true
	case "atomicWrite":
		return "AhdFileAtomicWrite(" + errorClass + ", " + text(0) + ", " + text(1) + ")", true
	case "atomicMove":
		return "AhdFileAtomicMove(" + errorClass + ", " + text(0) + ", " + text(1) + ")", true
	case "symlink":
		return "AhdFileSymlink(" + errorClass + ", " + text(0) + ", " + text(1) + ")", true
	case "readLink":
		return "AhdFileReadLink(" + errorClass + ", " + text(0) + ")", true
	case "isSymlink":
		return "AhdFileIsSymlink(" + errorClass + ", " + text(0) + ")", true
	case "permissions":
		return "AhdFilePermissions(" + errorClass + ", " + text(0) + ")", true
	case "setPermissions":
		return "AhdFileSetPermissions(" + errorClass + ", " + text(0) + ", " + text(1) + ")", true
	case "walk":
		return generator.systemsValueList(fileEntryClass, "AhdFileWalk("+errorClass+", "+text(0)+", "+
			generator.systemsInt(value, 1, "File.maxEntries")+")", meta), true
	}
	return "", false
}

// processCall lowers Process.run. The argument list crosses as
// *AhdList[string]; an omitted list means no arguments.
func (generator *generator) processCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), processModulePrefix)
	if name != "run" {
		return generator.unsupported("Process function "+name, meta.Span)
	}
	errorClass := generator.descriptorName(processErrorClass)
	args := "AhdNewList[string]()"
	if len(value.Arguments) > 1 && value.Arguments[1].Value != nil {
		args = generator.expr(value.Arguments[1].Value)
	}
	call := "AhdProcessRun(" + errorClass + ", " + generator.systemsText(value, 0) + ", " + args + ", " +
		generator.systemsInt(value, 2, "Process.timeoutSeconds") + ", " + generator.systemsInt(value, 3, "Process.maxOutputBytes") + ")"
	return generator.smtpValueFrom(processResultClass, call, meta)
}

func isSystemsOperation(name string) bool {
	return strings.HasPrefix(name, "ArchiveEntry.") || strings.HasPrefix(name, "FileEntry.") || strings.HasPrefix(name, "ProcessResult.")
}

// systemsOperation lowers the ArchiveEntry, FileEntry, and ProcessResult
// accessors.
func (generator *generator) systemsOperation(name string, value *ir.CallExpr) string {
	accessors := map[string]struct {
		class      ir.ClassID
		field      ir.FieldID
		errorClass ir.ClassID
		helper     string
	}{
		"ArchiveEntry.path":      {archiveEntryClass, archiveEntryDataField, archiveErrorClass, "AhdArchiveEntryPath"},
		"ArchiveEntry.kind":      {archiveEntryClass, archiveEntryDataField, archiveErrorClass, "AhdArchiveEntryKind"},
		"ArchiveEntry.size":      {archiveEntryClass, archiveEntryDataField, archiveErrorClass, "AhdArchiveEntrySize"},
		"FileEntry.path":         {fileEntryClass, fileEntryDataField, fileErrorClass, "AhdFileEntryPath"},
		"FileEntry.relativePath": {fileEntryClass, fileEntryDataField, fileErrorClass, "AhdFileEntryRelativePath"},
		"FileEntry.kind":         {fileEntryClass, fileEntryDataField, fileErrorClass, "AhdFileEntryKind"},
		"FileEntry.size":         {fileEntryClass, fileEntryDataField, fileErrorClass, "AhdFileEntrySize"},
		"FileEntry.isSymlink":    {fileEntryClass, fileEntryDataField, fileErrorClass, "AhdFileEntryIsSymlink"},
		"ProcessResult.exitCode": {processResultClass, processResultDataField, processErrorClass, "AhdProcessResultExitCode"},
		"ProcessResult.stdout":   {processResultClass, processResultDataField, processErrorClass, "AhdProcessResultStdout"},
		"ProcessResult.stderr":   {processResultClass, processResultDataField, processErrorClass, "AhdProcessResultStderr"},
	}
	accessor, known := accessors[name]
	if !known {
		return generator.unsupported("systems operation "+name, value.ExprMeta().Span)
	}
	return accessor.helper + "(" + generator.descriptorName(accessor.errorClass) + ", " +
		generator.smtpDataOf(accessor.class, accessor.field, value.Callee) + ")"
}
