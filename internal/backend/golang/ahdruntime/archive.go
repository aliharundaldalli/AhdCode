package ahdruntime

// The Archive standard module: a creation-only ZIP/TAR/TAR.GZ writer built
// entirely on the Go standard library (archive/zip, archive/tar,
// compress/gzip). There is no extraction, listing, or archive object model.
// This file is also emitted verbatim into native programs (with only its
// package clause rewritten), so it intentionally depends on the Go standard
// library and the sibling AhdCode runtime only.

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func ahdArchiveRaise(message string) { AhdRaiseClass(AhdClassArchiveError, message) }

// ahdArchiveEntry is one validated (member path, source path) pair, kept in
// Pair insertion order -- the chosen deterministic archive entry order.
type ahdArchiveEntry struct {
	Member string
	Source string
}

// ahdArchiveValidateMemberName rejects every unsafe or ambiguous archive
// member path outright rather than silently normalizing it.
func ahdArchiveValidateMemberName(name string) error {
	if name == "" {
		return fmt.Errorf("archive member name must not be empty")
	}
	if strings.ContainsRune(name, 0) {
		return fmt.Errorf("archive member name %s must not contain a NUL byte", strconv.Quote(name))
	}
	if strings.Contains(name, "\\") {
		return fmt.Errorf("archive member name %s must use forward slashes, not backslashes", strconv.Quote(name))
	}
	if strings.HasPrefix(name, "/") {
		return fmt.Errorf("archive member name %s must not be absolute", strconv.Quote(name))
	}
	if strings.Contains(name, "//") {
		return fmt.Errorf("archive member name %s must not contain an empty path segment", strconv.Quote(name))
	}
	if len(name) >= 2 && name[1] == ':' {
		return fmt.Errorf("archive member name %s must not contain a drive prefix", strconv.Quote(name))
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "." || segment == ".." {
			return fmt.Errorf("archive member name %s must not contain a %q path segment", strconv.Quote(name), segment)
		}
	}
	return nil
}

// ahdArchiveEntries validates and resolves one Pair<String,String> of
// (member path, source filesystem path) into ordered, safety-checked
// entries. Pair already guarantees unique keys, and v0.1.20 has no directory
// expansion step that could create new ones, so a member-name collision is
// structurally unreachable here; the check below documents that invariant
// rather than papering over a real gap.
func ahdArchiveEntries(absoluteOutput string, entries *AhdPair[string, string]) []ahdArchiveEntry {
	entries.require()
	seen := make(map[string]bool, len(entries.keys))
	result := make([]ahdArchiveEntry, 0, len(entries.keys))
	for _, member := range entries.keys {
		if err := ahdArchiveValidateMemberName(member); err != nil {
			ahdArchiveRaise(err.Error())
		}
		if seen[member] {
			ahdArchiveRaise("duplicate archive member name " + strconv.Quote(member))
		}
		seen[member] = true
		source := entries.values[member]
		info, err := os.Lstat(source)
		if err != nil {
			ahdArchiveRaise("could not read archive source " + strconv.Quote(source) + ": " + err.Error())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			ahdArchiveRaise("archive source " + strconv.Quote(source) +
				" is a symbolic link; v0.1.20 rejects symlink sources rather than following them")
		}
		if !info.Mode().IsRegular() {
			ahdArchiveRaise("archive source " + strconv.Quote(source) +
				" is not a regular file; v0.1.20 Archive accepts regular files only")
		}
		if absoluteSource, err := filepath.Abs(source); err == nil && absoluteSource == absoluteOutput {
			ahdArchiveRaise("archive source " + strconv.Quote(source) + " must not be the destination archive itself")
		}
		result = append(result, ahdArchiveEntry{Member: member, Source: source})
	}
	return result
}

func ahdArchiveWriteZIP(entries []ahdArchiveEntry, destination io.Writer) error {
	writer := zip.NewWriter(destination)
	for _, entry := range entries {
		data, err := os.ReadFile(entry.Source)
		if err != nil {
			return err
		}
		header := &zip.FileHeader{Name: entry.Member, Method: zip.Deflate}
		header.SetMode(0o644)
		part, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err := part.Write(data); err != nil {
			return err
		}
	}
	return writer.Close()
}

func ahdArchiveWriteTAR(entries []ahdArchiveEntry, destination io.Writer) error {
	writer := tar.NewWriter(destination)
	for _, entry := range entries {
		data, err := os.ReadFile(entry.Source)
		if err != nil {
			return err
		}
		header := &tar.Header{Name: entry.Member, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(data))}
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		if _, err := writer.Write(data); err != nil {
			return err
		}
	}
	return writer.Close()
}

func ahdArchiveWriteTARGzip(entries []ahdArchiveEntry, destination io.Writer) error {
	gzipWriter, err := gzip.NewWriterLevel(destination, gzip.BestCompression)
	if err != nil {
		return err
	}
	// Name, Comment, ModTime, and OS are left at their zero values so
	// identical entries produce byte-identical output on every run and host.
	if err := ahdArchiveWriteTAR(entries, gzipWriter); err != nil {
		_ = gzipWriter.Close()
		return err
	}
	return gzipWriter.Close()
}

// ahdArchiveBuild writes the whole archive into a same-directory temporary
// file, then atomically renames it over the destination -- the same
// build-then-rename pattern excelAtomicWrite already uses for XLSX. A failed
// build never touches an existing valid archive.
func ahdArchiveBuild(output string, entries *AhdPair[string, string], write func([]ahdArchiveEntry, io.Writer) error) {
	absolute, err := filepath.Abs(output)
	if err != nil {
		ahdArchiveRaise("could not resolve destination path: " + err.Error())
	}
	resolved := ahdArchiveEntries(absolute, entries)
	directory := filepath.Dir(absolute)
	temporary, err := os.CreateTemp(directory, ".ahdcode-archive-output-*")
	if err != nil {
		ahdArchiveRaise("could not create a secure temporary file: " + err.Error())
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		ahdArchiveRaise("could not prepare the atomic output file: " + err.Error())
	}
	writeErr := write(resolved, temporary)
	syncErr := temporary.Sync()
	closeErr := temporary.Close()
	if writeErr != nil {
		ahdArchiveRaise("could not write archive: " + writeErr.Error())
	}
	if syncErr != nil {
		ahdArchiveRaise("could not write archive: " + syncErr.Error())
	}
	if closeErr != nil {
		ahdArchiveRaise("could not write archive: " + closeErr.Error())
	}
	if err := os.Rename(temporaryPath, absolute); err != nil {
		ahdArchiveRaise("could not atomically replace the destination archive: " + err.Error())
	}
}

func ahdArchiveHasTarGzipExtension(output string) bool {
	return strings.HasSuffix(strings.ToLower(output), ".tar.gz")
}

func AhdArchiveZip(output string, entries *AhdPair[string, string]) {
	if !strings.EqualFold(filepath.Ext(output), ".zip") {
		ahdArchiveRaise("Archive.zip destination must use the .zip extension")
	}
	ahdArchiveBuild(output, entries, ahdArchiveWriteZIP)
}

func AhdArchiveTar(output string, entries *AhdPair[string, string]) {
	if !strings.EqualFold(filepath.Ext(output), ".tar") {
		ahdArchiveRaise("Archive.tar destination must use the .tar extension")
	}
	ahdArchiveBuild(output, entries, ahdArchiveWriteTAR)
}

func AhdArchiveTarGzip(output string, entries *AhdPair[string, string]) {
	if !ahdArchiveHasTarGzipExtension(output) {
		ahdArchiveRaise("Archive.tarGzip destination must use the .tar.gz extension")
	}
	ahdArchiveBuild(output, entries, ahdArchiveWriteTARGzip)
}
