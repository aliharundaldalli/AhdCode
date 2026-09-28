package ahdruntime

// The v2.5.0 Archive reading side: Archive.list and the safe-by-default
// Archive.extract for ZIP, TAR, and TAR.GZ. Built only on the Go standard
// library (archive/zip, archive/tar, compress/gzip), so this file is emitted
// verbatim into native programs and called directly by the evaluator.
//
// Extraction policy (there is no unsafe variant):
//   - every entry path is validated before anything is written: absolute
//     paths, drive and UNC prefixes, backslashes, colons, NUL bytes, and ".."
//     segments are rejected, and the joined target is re-checked with
//     filepath.Rel against the staging directory, never by string prefix;
//   - symbolic-link, hard-link, device, FIFO, and sparse entries are rejected;
//   - the entry count and the total uncompressed size are bounded, first
//     from the archive's metadata (before any file is written) and again
//     while streaming, because metadata cannot be trusted;
//   - the archive is extracted into a fresh private staging directory beside
//     the destination and renamed to the destination only after every entry
//     succeeded. The destination must not exist beforehand, so no existing
//     content is ever merged into, overwritten, or deleted. On any failure the
//     staging directory (created by this call) is removed.

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Archive bounds. Defaults apply when a call omits maxFiles/maxBytes; an
// explicit value may lower or raise them within the hard limits.
const (
	AhdArchiveDefaultMaxFiles = int64(10000)
	AhdArchiveDefaultMaxBytes = int64(1) << 30 // 1 GiB
	AhdArchiveHardMaxFiles    = int64(1000000)
	AhdArchiveHardMaxBytes    = int64(64) << 30 // 64 GiB
	// ahdArchiveHeaderAllowance bounds the TAR header bytes (including PAX
	// records and GNU long names) read per entry, on top of maxBytes.
	ahdArchiveHeaderAllowance = int64(64 * 1024)
)

type ahdArchiveFailure struct{ message string }

func (failure *ahdArchiveFailure) Error() string { return failure.message }

func ahdArchiveFail(operation, archive, reason string) error {
	return &ahdArchiveFailure{message: operation + " " + strconv.Quote(archive) + " failed: " + reason}
}

// ahdArchiveMalformed phrases a reader error as "malformed archive: ...",
// without the Go package prefixes.
func ahdArchiveMalformed(err error) string {
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return "malformed archive: unexpected end of data"
	}
	text := err.Error()
	for _, prefix := range []string{"zip: ", "gzip: ", "archive/tar: ", "flate: "} {
		text = strings.TrimPrefix(text, prefix)
	}
	return "malformed archive: " + text
}

// ahdArchiveLimitError is raised by the streaming byte guard.
type ahdArchiveLimitError struct{ reason string }

func (failure *ahdArchiveLimitError) Error() string { return failure.reason }

// ahdArchiveGuard counts bytes flowing from a decompressor and fails once
// more than its limit was produced.
type ahdArchiveGuard struct {
	reader io.Reader
	left   int64
	reason string
}

func (guard *ahdArchiveGuard) Read(buffer []byte) (int, error) {
	if guard.left <= 0 {
		return 0, &ahdArchiveLimitError{reason: guard.reason}
	}
	if int64(len(buffer)) > guard.left {
		buffer = buffer[:guard.left]
	}
	count, err := guard.reader.Read(buffer)
	guard.left -= int64(count)
	return count, err
}

type ahdArchiveFormat int

const (
	ahdArchiveUnknown ahdArchiveFormat = iota
	ahdArchiveZip
	ahdArchiveTar
	ahdArchiveTarGzip
)

// ahdArchiveSniff identifies the format from its content, never from the
// file name.
func ahdArchiveSniff(file *os.File) (ahdArchiveFormat, error) {
	header := make([]byte, 512)
	count, err := io.ReadFull(file, header)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return ahdArchiveUnknown, err
	}
	header = header[:count]
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return ahdArchiveUnknown, err
	}
	switch {
	case len(header) >= 4 && string(header[:2]) == "PK" &&
		(string(header[2:4]) == "\x03\x04" || string(header[2:4]) == "\x05\x06"):
		return ahdArchiveZip, nil
	case len(header) >= 2 && header[0] == 0x1f && header[1] == 0x8b:
		return ahdArchiveTarGzip, nil
	case len(header) >= 262 && string(header[257:262]) == "ustar":
		return ahdArchiveTar, nil
	}
	return ahdArchiveUnknown, nil
}

// ahdArchiveItem is one entry as the archive describes it.
type ahdArchiveItem struct {
	Name       string
	Kind       string
	Size       int64
	Executable bool
	Encrypted  bool
	open       func() (io.ReadCloser, error) // ZIP only
}

func ahdArchiveZipKind(file *zip.File) string {
	mode := file.Mode()
	switch {
	case mode&fs.ModeSymlink != 0:
		return "symlink"
	case mode.IsDir() || strings.HasSuffix(file.Name, "/"):
		return "directory"
	case mode&(fs.ModeNamedPipe|fs.ModeDevice|fs.ModeCharDevice|fs.ModeSocket|fs.ModeIrregular) != 0:
		return "other"
	}
	return "file"
}

func ahdArchiveTarKind(header *tar.Header) string {
	switch header.Typeflag {
	case tar.TypeReg, '\x00':
		return "file"
	case tar.TypeDir:
		return "directory"
	case tar.TypeSymlink:
		return "symlink"
	case tar.TypeLink:
		return "hardlink"
	}
	return "other"
}

// ahdArchiveVisit calls visit for every entry, in archive order. For TAR,
// the entry's content is readable from the tar reader during the callback.
// decompressedLimit bounds the bytes a TAR stream may produce in total.
func ahdArchiveVisit(archivePath string, decompressedLimit int64, visit func(item ahdArchiveItem, content io.Reader) error) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return errors.New("cannot open the archive: " + ahdFSReason(err))
	}
	defer file.Close()
	format, err := ahdArchiveSniff(file)
	if err != nil {
		return errors.New("cannot read the archive: " + ahdFSReason(err))
	}
	switch format {
	case ahdArchiveZip:
		info, err := file.Stat()
		if err != nil {
			return errors.New("cannot read the archive: " + ahdFSReason(err))
		}
		reader, err := zip.NewReader(file, info.Size())
		if err != nil {
			return errors.New(ahdArchiveMalformed(err))
		}
		for _, entry := range reader.File {
			size := int64(-1)
			if entry.UncompressedSize64 <= uint64(1<<62) {
				size = int64(entry.UncompressedSize64)
			}
			item := ahdArchiveItem{
				Name: entry.Name, Kind: ahdArchiveZipKind(entry), Size: size,
				Executable: entry.Mode().Perm()&0o111 != 0, Encrypted: entry.Flags&0x1 != 0,
				open: entry.Open,
			}
			if err := visit(item, nil); err != nil {
				return err
			}
		}
		return nil
	case ahdArchiveTar, ahdArchiveTarGzip:
		var stream io.Reader = file
		if format == ahdArchiveTarGzip {
			decompressor, err := gzip.NewReader(file)
			if err != nil {
				return errors.New(ahdArchiveMalformed(err))
			}
			defer decompressor.Close()
			stream = decompressor
		}
		guard := &ahdArchiveGuard{reader: stream, left: decompressedLimit,
			reason: "byte limit exceeded: the archive expands beyond its allowed uncompressed size (maxBytes)"}
		reader := tar.NewReader(guard)
		for {
			header, err := reader.Next()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				var limit *ahdArchiveLimitError
				if errors.As(err, &limit) {
					return limit
				}
				return errors.New(ahdArchiveMalformed(err))
			}
			if header.Typeflag == tar.TypeXGlobalHeader {
				// A PAX global header (git archive writes one) is metadata for
				// the whole archive, not an entry: nothing to list or write.
				continue
			}
			if header.Size < 0 {
				return errors.New("malformed archive: negative entry size")
			}
			item := ahdArchiveItem{
				Name: header.Name, Kind: ahdArchiveTarKind(header), Size: header.Size,
				Executable: header.Mode&0o111 != 0,
			}
			if err := visit(item, reader); err != nil {
				return err
			}
		}
	}
	return errors.New("unsupported archive format; expected ZIP, TAR, or TAR.GZ")
}

// ahdArchiveSafePath validates one entry name and returns its cleaned,
// slash-separated relative path ("" for the archive root itself).
func ahdArchiveSafePath(name string) (string, error) {
	unsafe := func(reason string) (string, error) {
		return "", fmt.Errorf("unsafe entry path %s: %s", strconv.Quote(name), reason)
	}
	switch {
	case name == "":
		return unsafe("the path is empty")
	case strings.ContainsRune(name, 0):
		return unsafe("the path contains a NUL byte")
	case strings.Contains(name, "\\"):
		return unsafe("the path contains a backslash")
	case strings.HasPrefix(name, "/"):
		return unsafe("the path is absolute")
	case len(name) >= 2 && name[1] == ':':
		return unsafe("the path has a drive prefix")
	case strings.Contains(name, ":"):
		return unsafe("the path contains a colon")
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == ".." {
			return unsafe("the path escapes the destination")
		}
		if runtime.GOOS == "windows" && segment != "" && segment != "." {
			if strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") {
				return unsafe("a Windows path segment cannot end with a dot or space")
			}
			stem := strings.ToUpper(strings.SplitN(segment, ".", 2)[0])
			switch stem {
			case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
				"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
				return unsafe("the path names a reserved Windows device")
			}
		}
	}
	clean := path.Clean(name)
	if clean == "." {
		return "", nil
	}
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return unsafe("the path escapes the destination")
	}
	return clean, nil
}

// ahdArchiveCheck applies the safety policy to one item and the running
// totals. It is shared by the metadata preflight and the streaming pass.
func ahdArchiveCheck(item ahdArchiveItem, count, total *int64, maxFiles, maxBytes int64) (string, error) {
	*count++
	if *count > maxFiles {
		return "", fmt.Errorf("file-count limit exceeded: the archive has more than %d entries (maxFiles)", maxFiles)
	}
	relative, err := ahdArchiveSafePath(item.Name)
	if err != nil {
		return "", err
	}
	switch item.Kind {
	case "symlink":
		return "", fmt.Errorf("entry %s is a symbolic link; safe extraction rejects links", strconv.Quote(item.Name))
	case "hardlink":
		return "", fmt.Errorf("entry %s is a hard link; safe extraction rejects links", strconv.Quote(item.Name))
	case "other":
		return "", fmt.Errorf("entry %s is not a regular file or directory (device, FIFO, and sparse entries are rejected)", strconv.Quote(item.Name))
	case "file":
		if relative == "" {
			return "", fmt.Errorf("unsafe entry path %s: a file entry has no name", strconv.Quote(item.Name))
		}
		if item.Encrypted {
			return "", fmt.Errorf("entry %s is encrypted; encrypted entries are not supported", strconv.Quote(item.Name))
		}
		if item.Size < 0 || item.Size > maxBytes || *total > maxBytes-item.Size {
			return "", fmt.Errorf("byte limit exceeded: the archive declares more than %d uncompressed bytes (maxBytes)", maxBytes)
		}
		*total += item.Size
	}
	return relative, nil
}

func ahdArchiveCheckBounds(maxFiles, maxBytes int64) string {
	if maxFiles < 1 || maxFiles > AhdArchiveHardMaxFiles {
		return fmt.Sprintf("maxFiles must be between 1 and %d; received %d", AhdArchiveHardMaxFiles, maxFiles)
	}
	if maxBytes < 1 || maxBytes > AhdArchiveHardMaxBytes {
		return fmt.Sprintf("maxBytes must be between 1 and %d; received %d", AhdArchiveHardMaxBytes, maxBytes)
	}
	return ""
}

// ahdArchiveStreamLimit bounds the decompressed TAR stream: the content bytes
// plus a per-entry allowance for headers. Both inputs are within their hard
// limits, so the sum cannot overflow.
func ahdArchiveStreamLimit(maxFiles, maxBytes int64) int64 {
	return maxBytes + (maxFiles+2)*ahdArchiveHeaderAllowance
}

// ahdArchiveEntryData is the whole public surface of one ArchiveEntry.
type ahdArchiveEntryData struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
	Size int64  `json:"size"`
}

// ArchiveList returns every entry of a ZIP, TAR, or TAR.GZ archive in
// archive order, encoded for ArchiveEntry. It reports entries as stored --
// including links and unsafe paths, so a program can inspect them -- and
// writes nothing. At most maxFiles entries are accepted.
func ArchiveList(archivePath string, maxFiles int64) ([]string, error) {
	const operation = "list"
	if maxFiles < 1 || maxFiles > AhdArchiveHardMaxFiles {
		return nil, ahdArchiveFail(operation, archivePath, fmt.Sprintf("maxFiles must be between 1 and %d; received %d", AhdArchiveHardMaxFiles, maxFiles))
	}
	entries := make([]string, 0)
	err := ahdArchiveVisit(archivePath, ahdArchiveStreamLimit(maxFiles, AhdArchiveHardMaxBytes), func(item ahdArchiveItem, _ io.Reader) error {
		if int64(len(entries)) >= maxFiles {
			return fmt.Errorf("file-count limit exceeded: the archive has more than %d entries (maxFiles)", maxFiles)
		}
		size := item.Size
		if item.Kind != "file" || size < 0 {
			size = 0
		}
		encoded, err := json.Marshal(ahdArchiveEntryData{Path: item.Name, Kind: item.Kind, Size: size})
		if err != nil {
			return err
		}
		entries = append(entries, string(encoded))
		return nil
	})
	if err != nil {
		return nil, ahdArchiveFail(operation, archivePath, err.Error())
	}
	return entries, nil
}

func ahdArchiveOutside(base, target string) bool {
	relative, err := filepath.Rel(base, target)
	return err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative)
}

// ArchiveExtract safely extracts a ZIP, TAR, or TAR.GZ archive into
// destination, which must not exist yet (its parent directory must). See the
// file comment for the complete policy.
func ArchiveExtract(archivePath, destination string, maxFiles, maxBytes int64) (resultErr error) {
	const operation = "extract"
	fail := func(reason string) error { return ahdArchiveFail(operation, archivePath, reason) }
	if reason := ahdArchiveCheckBounds(maxFiles, maxBytes); reason != "" {
		return fail(reason)
	}
	if destination == "" {
		return fail("the destination path is empty")
	}
	absolute, err := filepath.Abs(destination)
	if err != nil {
		return fail("destination: " + ahdFSReason(err))
	}
	if _, err := os.Lstat(absolute); err == nil {
		return fail("destination " + strconv.Quote(destination) + " already exists; Archive.extract only creates a new directory")
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fail("destination: " + ahdFSReason(err))
	}
	parent := filepath.Dir(absolute)
	if info, err := os.Stat(parent); err != nil {
		return fail("destination parent directory: " + ahdFSReason(err))
	} else if !info.IsDir() {
		return fail("destination parent is not a directory")
	}
	streamLimit := ahdArchiveStreamLimit(maxFiles, maxBytes)

	// Pass 1: metadata preflight. Nothing is written.
	var count, total int64
	if err := ahdArchiveVisit(archivePath, streamLimit, func(item ahdArchiveItem, _ io.Reader) error {
		_, err := ahdArchiveCheck(item, &count, &total, maxFiles, maxBytes)
		return err
	}); err != nil {
		return fail(err.Error())
	}

	// Pass 2: stream into a private staging directory.
	stage, err := os.MkdirTemp(parent, "."+filepath.Base(absolute)+".ahdextract-")
	if err != nil {
		return fail("cannot create the staging directory: " + ahdFSReason(err))
	}
	defer func() {
		if resultErr != nil {
			_ = os.RemoveAll(stage)
		}
	}()
	realStage, err := filepath.EvalSymlinks(stage)
	if err != nil {
		return fail("staging directory: " + ahdFSReason(err))
	}
	count, total = 0, 0
	written := int64(0)
	err = ahdArchiveVisit(archivePath, streamLimit, func(item ahdArchiveItem, content io.Reader) error {
		relative, err := ahdArchiveCheck(item, &count, &total, maxFiles, maxBytes)
		if err != nil {
			return err
		}
		if relative == "" {
			return nil
		}
		target := filepath.Join(stage, filepath.FromSlash(relative))
		if ahdArchiveOutside(stage, target) {
			return fmt.Errorf("unsafe entry path %s: the path escapes the destination", strconv.Quote(item.Name))
		}
		if item.Kind == "directory" {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("cannot create directory for entry %s: %s", strconv.Quote(item.Name), ahdFSReason(err))
			}
			return nil
		}
		directory := filepath.Dir(target)
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return fmt.Errorf("cannot create directory for entry %s: %s", strconv.Quote(item.Name), ahdFSReason(err))
		}
		// Defense in depth: the resolved parent must still be inside the
		// staging directory. No entry can create a link, so this holds
		// unless something outside this call interferes.
		realDirectory, err := filepath.EvalSymlinks(directory)
		if err != nil {
			return fmt.Errorf("cannot resolve directory for entry %s: %s", strconv.Quote(item.Name), ahdFSReason(err))
		}
		if ahdArchiveOutside(realStage, realDirectory) {
			return fmt.Errorf("unsafe entry path %s: the path escapes the destination", strconv.Quote(item.Name))
		}
		mode := fs.FileMode(0o644)
		if item.Executable {
			mode = 0o755
		}
		source := content
		if item.open != nil {
			opened, err := item.open()
			if err != nil {
				return errors.New(ahdArchiveMalformed(err))
			}
			defer opened.Close()
			source = opened
		}
		output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				return fmt.Errorf("entry %s appears more than once or collides with another entry", strconv.Quote(item.Name))
			}
			return fmt.Errorf("cannot create file for entry %s: %s", strconv.Quote(item.Name), ahdFSReason(err))
		}
		// Copy at most one byte more than declared, so an entry that
		// expands beyond its metadata is caught without writing it all.
		copied, copyErr := io.Copy(output, io.LimitReader(source, item.Size+1))
		closeErr := output.Close()
		written += copied
		switch {
		case copyErr != nil:
			var limit *ahdArchiveLimitError
			if errors.As(copyErr, &limit) {
				return limit
			}
			var pathError *fs.PathError
			if errors.As(copyErr, &pathError) {
				return fmt.Errorf("cannot write entry %s: %s", strconv.Quote(item.Name), ahdFSReason(copyErr))
			}
			return errors.New(ahdArchiveMalformed(copyErr))
		case copied > item.Size:
			return fmt.Errorf("byte limit exceeded: entry %s expands beyond its declared size", strconv.Quote(item.Name))
		case copied < item.Size:
			return errors.New("malformed archive: entry " + strconv.Quote(item.Name) + " is shorter than its declared size")
		case written > maxBytes:
			return fmt.Errorf("byte limit exceeded: the archive expands beyond %d uncompressed bytes (maxBytes)", maxBytes)
		case closeErr != nil:
			return fmt.Errorf("cannot write entry %s: %s", strconv.Quote(item.Name), ahdFSReason(closeErr))
		}
		return nil
	})
	if err != nil {
		return fail(err.Error())
	}
	if err := os.Chmod(stage, 0o755); err != nil {
		return fail("staging directory: " + ahdFSReason(err))
	}
	if _, err := os.Lstat(absolute); err == nil {
		return fail("destination " + strconv.Quote(destination) + " was created by someone else while extracting; it was left untouched")
	}
	if err := os.Rename(stage, absolute); err != nil {
		return fail("cannot move the extracted tree into place: " + ahdFSReason(err))
	}
	ahdFSSyncDir(parent)
	return nil
}

func ahdArchiveEntryDecode(data string) (ahdArchiveEntryData, error) {
	var entry ahdArchiveEntryData
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		return entry, errors.New("ArchiveEntry storage is corrupted")
	}
	return entry, nil
}

func ArchiveEntryPath(data string) (string, error) {
	entry, err := ahdArchiveEntryDecode(data)
	return entry.Path, err
}

func ArchiveEntryKind(data string) (string, error) {
	entry, err := ahdArchiveEntryDecode(data)
	return entry.Kind, err
}

func ArchiveEntrySize(data string) (int64, error) {
	entry, err := ahdArchiveEntryDecode(data)
	return entry.Size, err
}

// --- native wrappers: raise the program's ArchiveError ---

func ahdArchiveRaiseErr(class *AhdClass, err error) {
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
}

func AhdArchiveList(class *AhdClass, archivePath string, maxFiles int64) []string {
	entries, err := ArchiveList(archivePath, maxFiles)
	ahdArchiveRaiseErr(class, err)
	return entries
}

func AhdArchiveExtract(class *AhdClass, archivePath, destination string, maxFiles, maxBytes int64) {
	ahdArchiveRaiseErr(class, ArchiveExtract(archivePath, destination, maxFiles, maxBytes))
}

func AhdArchiveEntryPath(class *AhdClass, data string) string {
	value, err := ArchiveEntryPath(data)
	ahdArchiveRaiseErr(class, err)
	return value
}

func AhdArchiveEntryKind(class *AhdClass, data string) string {
	value, err := ArchiveEntryKind(data)
	ahdArchiveRaiseErr(class, err)
	return value
}

func AhdArchiveEntrySize(class *AhdClass, data string) int64 {
	value, err := ArchiveEntrySize(data)
	ahdArchiveRaiseErr(class, err)
	return value
}
