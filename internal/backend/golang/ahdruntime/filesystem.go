package ahdruntime

// The v2.5.0 File deployment primitives: binary copy, atomic write, atomic
// move, symbolic links, permissions, and a bounded, non-following recursive
// walk. They are built only on the Go standard library (os, io, path/filepath,
// syscall), so this file is emitted verbatim into native programs with only
// its package clause rewritten, and the persistent evaluator calls the exact
// same functions.
//
// Every exported function without the Ahd prefix returns a Go error instead
// of raising, so the evaluator can turn it into its own catchable FileError;
// the Ahd-prefixed wrappers raise through AhdRaiseClass for native programs.
// The returned errors already carry the operation and path context and a
// plain reason: raw Go error text (which repeats the path and the syscall
// name) never reaches an AhdCode program.

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"
)

// File.walk bounds. The listing is materialized, so it is bounded: the
// default suits a deployment tree, and the hard limit keeps an explicit
// raise finite.
const (
	AhdFileWalkDefaultMaxEntries = int64(100000)
	AhdFileWalkHardMaxEntries    = int64(1000000)
)

// ahdFSError is a filesystem failure already phrased for an AhdCode program.
type ahdFSError struct{ message string }

func (failure *ahdFSError) Error() string { return failure.message }

func ahdFSFail(operation, path, reason string) error {
	return &ahdFSError{message: operation + " " + strconv.Quote(path) + " failed: " + reason}
}

func ahdFSFail2(operation, source, destination, reason string) error {
	return &ahdFSError{message: operation + " " + strconv.Quote(source) + " to " + strconv.Quote(destination) + " failed: " + reason}
}

// ahdFSIsCrossDevice reports a rename that the operating system refused
// because source and destination are on different filesystems (EXDEV on Unix,
// ERROR_NOT_SAME_DEVICE on Windows).
func ahdFSIsCrossDevice(err error) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	if runtime.GOOS == "windows" {
		return uintptr(errno) == 17
	}
	return errors.Is(err, syscall.EXDEV)
}

// ahdFSReason turns an operating-system error into a short, stable reason.
// The well-known conditions get fixed wording; anything else keeps only the
// innermost system message, never the Go operation prefix or the path.
func ahdFSReason(err error) string {
	switch {
	case err == nil:
		return "unknown failure"
	case errors.Is(err, fs.ErrNotExist):
		return "no such file or directory"
	case errors.Is(err, fs.ErrExist):
		return "the path already exists"
	case errors.Is(err, fs.ErrPermission):
		return "permission denied"
	case ahdFSIsCrossDevice(err):
		return "source and destination are on different filesystems"
	}
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		return pathError.Err.Error()
	}
	var linkError *os.LinkError
	if errors.As(err, &linkError) {
		return linkError.Err.Error()
	}
	var syscallError *os.SyscallError
	if errors.As(err, &syscallError) {
		return syscallError.Err.Error()
	}
	return err.Error()
}

func ahdFSRandomSuffix() string {
	var buffer [8]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buffer[:])
}

// ahdFSTempFile creates a new, empty, exclusively created temporary file in
// dir -- the same directory (so the same filesystem) as its final path.
func ahdFSTempFile(dir, base string) (*os.File, string, error) {
	for attempt := 0; attempt < 16; attempt++ {
		name := filepath.Join(dir, "."+base+".ahdtmp-"+ahdFSRandomSuffix())
		file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			return file, name, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, "", err
		}
	}
	return nil, "", errors.New("could not create a unique temporary file")
}

// ahdFSSyncDir flushes a directory entry change (a rename) where the
// operating system supports it. Windows cannot open a directory for sync, so
// there the rename itself is the durability boundary.
func ahdFSSyncDir(dir string) {
	if runtime.GOOS == "windows" {
		return
	}
	if handle, err := os.Open(dir); err == nil {
		_ = handle.Sync()
		_ = handle.Close()
	}
}

// ahdFSPublishNoReplace moves the finished temporary path to its final name
// only if that name does not exist yet. A hard link is the atomic
// "create if absent" primitive on every supported filesystem; on a filesystem
// without hard links it falls back to a checked rename, whose only window is
// a concurrent creator of the very same path.
func ahdFSPublishNoReplace(temporary, final string) error {
	err := os.Link(temporary, final)
	if err == nil {
		_ = os.Remove(temporary)
		return nil
	}
	if errors.Is(err, fs.ErrExist) {
		return err
	}
	if _, statErr := os.Lstat(final); statErr == nil {
		return fs.ErrExist
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return statErr
	}
	return os.Rename(temporary, final)
}

// FileCopy copies one regular file byte for byte, streaming through a fixed
// buffer. The source must be a regular file (a symbolic link is refused, not
// followed), the destination must not exist, and the copy is published under
// its final name only once it is complete: a failure never leaves a partial
// destination behind. The destination receives the source's permission bits.
func FileCopy(source, destination string) error {
	const operation = "copy"
	info, err := os.Lstat(source)
	if err != nil {
		return ahdFSFail2(operation, source, destination, "source: "+ahdFSReason(err))
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return ahdFSFail2(operation, source, destination, "source is a symbolic link; File.copy copies regular files only")
	}
	if !info.Mode().IsRegular() {
		return ahdFSFail2(operation, source, destination, "source is not a regular file")
	}
	if _, err := os.Lstat(destination); err == nil {
		return ahdFSFail2(operation, source, destination, "destination already exists; File.copy never overwrites")
	} else if !errors.Is(err, fs.ErrNotExist) {
		return ahdFSFail2(operation, source, destination, "destination: "+ahdFSReason(err))
	}
	input, err := os.Open(source)
	if err != nil {
		return ahdFSFail2(operation, source, destination, "source: "+ahdFSReason(err))
	}
	defer input.Close()
	output, temporary, err := ahdFSTempFile(filepath.Dir(destination), filepath.Base(destination))
	if err != nil {
		return ahdFSFail2(operation, source, destination, "destination: "+ahdFSReason(err))
	}
	published := false
	defer func() {
		if !published {
			_ = output.Close()
			_ = os.Remove(temporary)
		}
	}()
	if _, err := io.CopyBuffer(output, input, make([]byte, 64*1024)); err != nil {
		return ahdFSFail2(operation, source, destination, ahdFSReason(err))
	}
	if err := output.Sync(); err != nil {
		return ahdFSFail2(operation, source, destination, ahdFSReason(err))
	}
	if err := output.Close(); err != nil {
		return ahdFSFail2(operation, source, destination, ahdFSReason(err))
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(temporary, info.Mode().Perm()); err != nil {
			return ahdFSFail2(operation, source, destination, ahdFSReason(err))
		}
	}
	if err := ahdFSPublishNoReplace(temporary, destination); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return ahdFSFail2(operation, source, destination, "destination already exists; File.copy never overwrites")
		}
		return ahdFSFail2(operation, source, destination, ahdFSReason(err))
	}
	published = true
	ahdFSSyncDir(filepath.Dir(destination))
	return nil
}

// FileAtomicWrite replaces path with content in one step: the text is
// written to a temporary file in the same directory, flushed to stable
// storage, and renamed over path. Readers observe either the old or the new
// complete file, never a partial one. An existing file keeps its permission
// bits; a new file is created with 0644 (Unix). A symbolic link at path is
// replaced by the new file, not followed; a directory at path is refused.
func FileAtomicWrite(path, content string) error {
	const operation = "atomic write"
	if !utf8.ValidString(content) {
		return ahdFSFail(operation, path, "content is not valid UTF-8")
	}
	mode := fs.FileMode(0o644)
	if info, err := os.Lstat(path); err == nil {
		if info.IsDir() {
			return ahdFSFail(operation, path, "the path is a directory")
		}
		if info.Mode().IsRegular() {
			mode = info.Mode().Perm()
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return ahdFSFail(operation, path, ahdFSReason(err))
	}
	output, temporary, err := ahdFSTempFile(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		return ahdFSFail(operation, path, ahdFSReason(err))
	}
	committed := false
	defer func() {
		if !committed {
			_ = output.Close()
			_ = os.Remove(temporary)
		}
	}()
	if _, err := io.WriteString(output, content); err != nil {
		return ahdFSFail(operation, path, ahdFSReason(err))
	}
	if err := output.Sync(); err != nil {
		return ahdFSFail(operation, path, ahdFSReason(err))
	}
	if err := output.Close(); err != nil {
		return ahdFSFail(operation, path, ahdFSReason(err))
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(temporary, mode); err != nil {
			return ahdFSFail(operation, path, ahdFSReason(err))
		}
	}
	if err := os.Rename(temporary, path); err != nil {
		return ahdFSFail(operation, path, ahdFSReason(err))
	}
	committed = true
	ahdFSSyncDir(filepath.Dir(path))
	return nil
}

// FileAtomicMove renames source to destination with the operating system's
// single rename. It never copies: a move across filesystems fails with a
// clear reason instead of silently becoming copy-and-delete. An existing
// file or symbolic link at destination is replaced atomically; an existing
// directory at destination is refused on every platform, so the result does
// not depend on whether that directory happens to be empty.
func FileAtomicMove(source, destination string) error {
	const operation = "atomic move"
	if _, err := os.Lstat(source); err != nil {
		return ahdFSFail2(operation, source, destination, "source: "+ahdFSReason(err))
	}
	if info, err := os.Lstat(destination); err == nil {
		if info.IsDir() {
			return ahdFSFail2(operation, source, destination, "destination is an existing directory; File.atomicMove replaces files and symbolic links only")
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return ahdFSFail2(operation, source, destination, "destination: "+ahdFSReason(err))
	}
	if err := os.Rename(source, destination); err != nil {
		if ahdFSIsCrossDevice(err) {
			return ahdFSFail2(operation, source, destination, "source and destination are on different filesystems; File.atomicMove never copies")
		}
		return ahdFSFail2(operation, source, destination, ahdFSReason(err))
	}
	ahdFSSyncDir(filepath.Dir(destination))
	return nil
}

// FileSymlink creates the symbolic link `link` whose content is `target`,
// in the same order as `ln -s target link`. The target is stored verbatim
// (a relative target is resolved by the operating system relative to the
// link's own directory) and need not exist. The link must not exist yet.
func FileSymlink(target, link string) error {
	const operation = "create symbolic link"
	if target == "" {
		return ahdFSFail(operation, link, "the link target is empty")
	}
	if err := os.Symlink(target, link); err != nil {
		reason := ahdFSReason(err)
		if runtime.GOOS == "windows" && (errors.Is(err, fs.ErrPermission) || strings.Contains(strings.ToLower(reason), "privilege")) {
			reason = "Windows requires Developer Mode or the SeCreateSymbolicLinkPrivilege to create symbolic links"
		}
		return ahdFSFail(operation, link, reason)
	}
	return nil
}

// FileReadLink returns the stored target of the symbolic link at path,
// verbatim and unresolved.
func FileReadLink(path string) (string, error) {
	const operation = "read symbolic link"
	info, err := os.Lstat(path)
	if err != nil {
		return "", ahdFSFail(operation, path, ahdFSReason(err))
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		return "", ahdFSFail(operation, path, "the path is not a symbolic link")
	}
	target, err := os.Readlink(path)
	if err != nil {
		return "", ahdFSFail(operation, path, ahdFSReason(err))
	}
	return target, nil
}

// FileIsSymlink reports whether path itself is a symbolic link. A missing
// path is not a link (false), the same way File.exists reports false.
func FileIsSymlink(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, ahdFSFail("inspect", path, ahdFSReason(err))
	}
	return info.Mode()&fs.ModeSymlink != 0, nil
}

func ahdFSPermissionsSupported(operation, path string) error {
	if runtime.GOOS == "windows" {
		return ahdFSFail(operation, path, "Unix permission bits are not supported on Windows; its access control lists are not represented by File permissions")
	}
	return nil
}

// FilePermissions returns the Unix permission bits of path as a four-digit
// octal String such as "0755". Symbolic links are refused rather than
// followed, so a link can never redirect the answer to another file.
func FilePermissions(path string) (string, error) {
	const operation = "read permissions of"
	if err := ahdFSPermissionsSupported(operation, path); err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", ahdFSFail(operation, path, ahdFSReason(err))
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return "", ahdFSFail(operation, path, "the path is a symbolic link; File permissions never follow links")
	}
	return fmt.Sprintf("%04o", uint32(info.Mode().Perm())), nil
}

// ahdFSParseMode accepts exactly three octal digits, optionally preceded by
// one 0: "755", "0755", "0640". setuid, setgid, and sticky bits are
// deliberately outside the accepted range.
func ahdFSParseMode(mode string) (fs.FileMode, bool) {
	digits := mode
	if len(digits) == 4 && digits[0] == '0' {
		digits = digits[1:]
	}
	if len(digits) != 3 {
		return 0, false
	}
	var value fs.FileMode
	for _, digit := range digits {
		if digit < '0' || digit > '7' {
			return 0, false
		}
		value = value*8 + fs.FileMode(digit-'0')
	}
	return value, true
}

// FileSetPermissions sets the Unix permission bits of path from an octal
// String ("0755", "640"). Symbolic links are refused rather than followed.
func FileSetPermissions(path, mode string) error {
	const operation = "set permissions of"
	parsed, ok := ahdFSParseMode(mode)
	if !ok {
		return ahdFSFail(operation, path, "mode "+strconv.Quote(mode)+" is not three octal digits such as \"0755\" or \"640\"")
	}
	if err := ahdFSPermissionsSupported(operation, path); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return ahdFSFail(operation, path, ahdFSReason(err))
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return ahdFSFail(operation, path, "the path is a symbolic link; File permissions never follow links")
	}
	if err := os.Chmod(path, parsed); err != nil {
		return ahdFSFail(operation, path, ahdFSReason(err))
	}
	return nil
}

// ahdFileEntryData is the whole public surface of one FileEntry.
type ahdFileEntryData struct {
	Path         string `json:"path"`
	RelativePath string `json:"relativePath"`
	Kind         string `json:"kind"`
	Size         int64  `json:"size"`
}

func ahdFSKind(mode fs.FileMode) string {
	switch {
	case mode&fs.ModeSymlink != 0:
		return "symlink"
	case mode.IsDir():
		return "directory"
	case mode.IsRegular():
		return "file"
	}
	return "other"
}

// FileWalk lists every entry below root (root itself excluded) in
// deterministic lexical order. Symbolic links are reported with kind
// "symlink" and are never followed or descended into, so a link cycle cannot
// loop the walk and a link cannot lead it outside root. The result is
// materialized and bounded by maxEntries. Each entry is returned in its
// encoded form; FileEntry accessors decode it.
func FileWalk(root string, maxEntries int64) ([]string, error) {
	return FileWalkAs(root, root, maxEntries)
}

// FileWalkAs walks resolved but reports every entry path (and any error)
// under display, the path exactly as the program wrote it. The evaluator
// resolves relative paths against its session directory; this keeps its
// results identical to a native program's.
func FileWalkAs(resolved, display string, maxEntries int64) ([]string, error) {
	const operation = "walk"
	root := resolved
	if maxEntries < 1 || maxEntries > AhdFileWalkHardMaxEntries {
		return nil, ahdFSFail(operation, display, fmt.Sprintf("maxEntries must be between 1 and %d; received %d", AhdFileWalkHardMaxEntries, maxEntries))
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, ahdFSFail(operation, display, ahdFSReason(err))
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return nil, ahdFSFail(operation, display, "the root is a symbolic link; File.walk never follows links (read it with File.readLink)")
	}
	if !info.IsDir() {
		return nil, ahdFSFail(operation, display, "the root is not a directory")
	}
	entries := make([]string, 0)
	count := int64(0)
	var limitReached bool
	walkErr := filepath.WalkDir(root, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if current == root {
			return nil
		}
		count++
		if count > maxEntries {
			limitReached = true
			return fs.SkipAll
		}
		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		data := ahdFileEntryData{Path: filepath.Join(display, relative), RelativePath: filepath.ToSlash(relative), Kind: ahdFSKind(entryInfo.Mode())}
		if data.Kind == "file" {
			data.Size = entryInfo.Size()
		}
		encoded, err := json.Marshal(data)
		if err != nil {
			return err
		}
		entries = append(entries, string(encoded))
		// WalkDir never descends through a symbolic link: a link to a
		// directory is reported as a non-directory entry.
		return nil
	})
	if limitReached {
		return nil, ahdFSFail(operation, display, fmt.Sprintf("the directory tree has more than %d entries (maxEntries)", maxEntries))
	}
	if walkErr != nil {
		return nil, ahdFSFail(operation, display, ahdFSReason(walkErr))
	}
	return entries, nil
}

func ahdFileEntryDecode(data string) (ahdFileEntryData, error) {
	var entry ahdFileEntryData
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		return entry, errors.New("FileEntry storage is corrupted")
	}
	return entry, nil
}

// FileEntryPath and its siblings read one accessor of an encoded FileEntry.
func FileEntryPath(data string) (string, error) {
	entry, err := ahdFileEntryDecode(data)
	return entry.Path, err
}

func FileEntryRelativePath(data string) (string, error) {
	entry, err := ahdFileEntryDecode(data)
	return entry.RelativePath, err
}

func FileEntryKind(data string) (string, error) {
	entry, err := ahdFileEntryDecode(data)
	return entry.Kind, err
}

func FileEntrySize(data string) (int64, error) {
	entry, err := ahdFileEntryDecode(data)
	return entry.Size, err
}

func FileEntryIsSymlink(data string) (bool, error) {
	entry, err := ahdFileEntryDecode(data)
	return entry.Kind == "symlink", err
}

// --- native wrappers: raise the program's FileError ---

func ahdFSRaise(class *AhdClass, err error) {
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
}

func AhdFileCopy(class *AhdClass, source, destination string) {
	ahdFSRaise(class, FileCopy(source, destination))
}

func AhdFileAtomicWrite(class *AhdClass, path, content string) {
	ahdFSRaise(class, FileAtomicWrite(path, content))
}

func AhdFileAtomicMove(class *AhdClass, source, destination string) {
	ahdFSRaise(class, FileAtomicMove(source, destination))
}

func AhdFileSymlink(class *AhdClass, target, link string) {
	ahdFSRaise(class, FileSymlink(target, link))
}

func AhdFileReadLink(class *AhdClass, path string) string {
	target, err := FileReadLink(path)
	ahdFSRaise(class, err)
	return target
}

func AhdFileIsSymlink(class *AhdClass, path string) bool {
	result, err := FileIsSymlink(path)
	ahdFSRaise(class, err)
	return result
}

func AhdFilePermissions(class *AhdClass, path string) string {
	mode, err := FilePermissions(path)
	ahdFSRaise(class, err)
	return mode
}

func AhdFileSetPermissions(class *AhdClass, path, mode string) {
	ahdFSRaise(class, FileSetPermissions(path, mode))
}

func AhdFileWalk(class *AhdClass, root string, maxEntries int64) []string {
	entries, err := FileWalk(root, maxEntries)
	ahdFSRaise(class, err)
	return entries
}

func AhdFileEntryPath(class *AhdClass, data string) string {
	value, err := FileEntryPath(data)
	ahdFSRaise(class, err)
	return value
}

func AhdFileEntryRelativePath(class *AhdClass, data string) string {
	value, err := FileEntryRelativePath(data)
	ahdFSRaise(class, err)
	return value
}

func AhdFileEntryKind(class *AhdClass, data string) string {
	value, err := FileEntryKind(data)
	ahdFSRaise(class, err)
	return value
}

func AhdFileEntrySize(class *AhdClass, data string) int64 {
	value, err := FileEntrySize(data)
	ahdFSRaise(class, err)
	return value
}

func AhdFileEntryIsSymlink(class *AhdClass, data string) bool {
	value, err := FileEntryIsSymlink(data)
	ahdFSRaise(class, err)
	return value
}
