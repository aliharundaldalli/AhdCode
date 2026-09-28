package evaluator

// The v2.5.0 systems primitives in the persistent evaluator: Archive.list and
// Archive.extract, the File deployment functions, Process.run, and the
// ArchiveEntry, FileEntry, and ProcessResult accessors. Every call goes to the
// same ahdruntime function a native program's wrapper calls, so validation,
// bounds, safety policy, and messages cannot diverge. The evaluator only
// resolves relative paths against its session directory and reports errors
// with the paths exactly as the program wrote them.

import (
	"strconv"
	"strings"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

var (
	evaluatorArchiveEntryClass  = ir.ClassID("builtin:Archive::class::ArchiveEntry")
	evaluatorFileEntryClass     = ir.ClassID("builtin:File::class::FileEntry")
	evaluatorProcessResultClass = ir.ClassID("builtin:Process::class::ProcessResult")

	evaluatorArchiveEntryField  = ir.FieldID("builtin:Archive::class::ArchiveEntry::field::data")
	evaluatorFileEntryField     = ir.FieldID("builtin:File::class::FileEntry::field::data")
	evaluatorProcessResultField = ir.FieldID("builtin:Process::class::ProcessResult::field::data")
)

// systemsPath is one path argument: as written, and as resolved.
type systemsPath struct{ written, resolved string }

func (session *Session) systemsPathOf(value any) systemsPath {
	written := value.(string)
	if written == "" {
		return systemsPath{written: written, resolved: written}
	}
	return systemsPath{written: written, resolved: session.sessionPath(written)}
}

// systemsRaise raises class with err's message, restoring the written form of
// every resolved path, so the message matches a native program's.
func (session *Session) systemsRaise(class string, err error, paths ...systemsPath) {
	if err == nil {
		return
	}
	message := err.Error()
	for _, path := range paths {
		if path.resolved != path.written {
			message = strings.ReplaceAll(message, strconv.Quote(path.resolved), strconv.Quote(path.written))
		}
	}
	session.raise(class, message)
}

func (session *Session) systemsIntArg(args []any, index int, fallback int64) int64 {
	if index >= len(args) || args[index] == nil {
		return fallback
	}
	return args[index].(int64)
}

func (session *Session) systemsValues(class ir.ClassID, field ir.FieldID, encoded []string) *List {
	items := make([]any, len(encoded))
	for index, data := range encoded {
		items[index] = &Instance{Class: class, Fields: map[ir.FieldID]any{field: data}}
	}
	return &List{Items: items}
}

func (session *Session) systemsDataOf(value any, class ir.ClassID, field ir.FieldID, errorClass, label string) string {
	instance := session.requireInstance(value)
	data, ok := instance.Fields[field].(string)
	if !ok || instance.Class != class {
		session.raise(errorClass, label+" storage is corrupted")
	}
	return data
}

// archiveReadingBuiltin handles Archive.list and Archive.extract.
func (session *Session) archiveReadingBuiltin(name string, args []any) (any, bool) {
	switch name {
	case "list":
		archive := session.systemsPathOf(args[0])
		entries, err := ahdruntime.ArchiveList(archive.resolved, session.systemsIntArg(args, 1, ahdruntime.AhdArchiveDefaultMaxFiles))
		session.systemsRaise("ArchiveError", err, archive)
		return session.systemsValues(evaluatorArchiveEntryClass, evaluatorArchiveEntryField, entries), true
	case "extract":
		archive := session.systemsPathOf(args[0])
		destination := session.systemsPathOf(args[1])
		err := ahdruntime.ArchiveExtract(archive.resolved, destination.resolved,
			session.systemsIntArg(args, 2, ahdruntime.AhdArchiveDefaultMaxFiles),
			session.systemsIntArg(args, 3, ahdruntime.AhdArchiveDefaultMaxBytes))
		session.systemsRaise("ArchiveError", err, archive, destination)
		return Nothing, true
	}
	return nil, false
}

// fileDeploymentBuiltin handles the v2.5.0 File functions.
func (session *Session) fileDeploymentBuiltin(name string, args []any) (any, bool) {
	const class = "FileError"
	switch name {
	case "copy", "atomicMove":
		source, destination := session.systemsPathOf(args[0]), session.systemsPathOf(args[1])
		operation := ahdruntime.FileCopy
		if name == "atomicMove" {
			operation = ahdruntime.FileAtomicMove
		}
		session.systemsRaise(class, operation(source.resolved, destination.resolved), source, destination)
		return Nothing, true
	case "atomicWrite":
		path := session.systemsPathOf(args[0])
		session.systemsRaise(class, ahdruntime.FileAtomicWrite(path.resolved, args[1].(string)), path)
		return Nothing, true
	case "symlink":
		// The target is stored verbatim: a relative target is relative to
		// the link's directory, never to the session directory.
		link := session.systemsPathOf(args[1])
		session.systemsRaise(class, ahdruntime.FileSymlink(args[0].(string), link.resolved), link)
		return Nothing, true
	case "readLink":
		path := session.systemsPathOf(args[0])
		target, err := ahdruntime.FileReadLink(path.resolved)
		session.systemsRaise(class, err, path)
		return target, true
	case "isSymlink":
		path := session.systemsPathOf(args[0])
		result, err := ahdruntime.FileIsSymlink(path.resolved)
		session.systemsRaise(class, err, path)
		return result, true
	case "permissions":
		path := session.systemsPathOf(args[0])
		mode, err := ahdruntime.FilePermissions(path.resolved)
		session.systemsRaise(class, err, path)
		return mode, true
	case "setPermissions":
		path := session.systemsPathOf(args[0])
		session.systemsRaise(class, ahdruntime.FileSetPermissions(path.resolved, args[1].(string)), path)
		return Nothing, true
	case "walk":
		path := session.systemsPathOf(args[0])
		entries, err := ahdruntime.FileWalkAs(path.resolved, path.written, session.systemsIntArg(args, 1, ahdruntime.AhdFileWalkDefaultMaxEntries))
		session.systemsRaise(class, err, path)
		return session.systemsValues(evaluatorFileEntryClass, evaluatorFileEntryField, entries), true
	}
	return nil, false
}

// processBuiltin handles Process.run. The child runs in the session
// directory, which is where a native program started from that directory
// would run it.
func (session *Session) processBuiltin(name string, args []any) any {
	if name != "run" {
		session.raise("Error", "unsupported Process function "+name)
		return nil
	}
	var arguments []string
	if len(args) > 1 && args[1] != nil {
		for _, item := range session.requireList(args[1]).Items {
			arguments = append(arguments, item.(string))
		}
	}
	data, err := ahdruntime.ProcessRun(args[0].(string), arguments,
		session.systemsIntArg(args, 2, ahdruntime.AhdProcessDefaultTimeoutSeconds),
		session.systemsIntArg(args, 3, ahdruntime.AhdProcessDefaultMaxOutputBytes), session.CWD)
	session.systemsRaise("ProcessError", err)
	return &Instance{Class: evaluatorProcessResultClass, Fields: map[ir.FieldID]any{evaluatorProcessResultField: data}}
}

// systemsOperation handles the ArchiveEntry, FileEntry, and ProcessResult
// accessors.
func (session *Session) systemsOperation(name string, receiver any) any {
	archive := func() string {
		return session.systemsDataOf(receiver, evaluatorArchiveEntryClass, evaluatorArchiveEntryField, "ArchiveError", "ArchiveEntry")
	}
	file := func() string {
		return session.systemsDataOf(receiver, evaluatorFileEntryClass, evaluatorFileEntryField, "FileError", "FileEntry")
	}
	process := func() string {
		return session.systemsDataOf(receiver, evaluatorProcessResultClass, evaluatorProcessResultField, "ProcessError", "ProcessResult")
	}
	var result any
	var err error
	class := "Error"
	switch name {
	case "ArchiveEntry.path":
		result, err = ahdruntime.ArchiveEntryPath(archive())
		class = "ArchiveError"
	case "ArchiveEntry.kind":
		result, err = ahdruntime.ArchiveEntryKind(archive())
		class = "ArchiveError"
	case "ArchiveEntry.size":
		result, err = ahdruntime.ArchiveEntrySize(archive())
		class = "ArchiveError"
	case "FileEntry.path":
		result, err = ahdruntime.FileEntryPath(file())
		class = "FileError"
	case "FileEntry.relativePath":
		result, err = ahdruntime.FileEntryRelativePath(file())
		class = "FileError"
	case "FileEntry.kind":
		result, err = ahdruntime.FileEntryKind(file())
		class = "FileError"
	case "FileEntry.size":
		result, err = ahdruntime.FileEntrySize(file())
		class = "FileError"
	case "FileEntry.isSymlink":
		result, err = ahdruntime.FileEntryIsSymlink(file())
		class = "FileError"
	case "ProcessResult.exitCode":
		result, err = ahdruntime.ProcessResultExitCode(process())
		class = "ProcessError"
	case "ProcessResult.stdout":
		result, err = ahdruntime.ProcessResultStdout(process())
		class = "ProcessError"
	case "ProcessResult.stderr":
		result, err = ahdruntime.ProcessResultStderr(process())
		class = "ProcessError"
	default:
		session.raise("Error", "unsupported systems operation "+name)
		return nil
	}
	session.systemsRaise(class, err)
	return result
}

func isSystemsOperation(name string) bool {
	return strings.HasPrefix(name, "ArchiveEntry.") || strings.HasPrefix(name, "FileEntry.") || strings.HasPrefix(name, "ProcessResult.")
}
