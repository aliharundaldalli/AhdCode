package ahdruntime

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

type archiveFixtureEntry struct {
	name    string
	body    string
	mode    fs.FileMode
	kind    byte // tar type flag; 0 = regular file
	link    string
	declare int64 // ZIP: declared uncompressed size override (0 = actual)
}

func writeZipFixture(t *testing.T, path string, entries ...archiveFixtureEntry) {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Store}
		mode := entry.mode
		if mode == 0 {
			mode = 0o644
		}
		header.SetMode(mode)
		if entry.declare != 0 {
			header.UncompressedSize64 = uint64(entry.declare)
			header.CompressedSize64 = uint64(len(entry.body))
			raw, err := writer.CreateRaw(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := raw.Write([]byte(entry.body)); err != nil {
				t.Fatal(err)
			}
			continue
		}
		file, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeTarGzipFixture(t *testing.T, path string, entries ...archiveFixtureEntry) {
	t.Helper()
	var buffer bytes.Buffer
	compressor := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(compressor)
	for _, entry := range entries {
		mode := int64(entry.mode.Perm())
		if mode == 0 {
			mode = 0o644
		}
		header := &tar.Header{Name: entry.name, Mode: mode, Typeflag: entry.kind, Linkname: entry.link, Format: tar.FormatPAX}
		if header.Typeflag == 0 {
			header.Typeflag = tar.TypeReg
		}
		if header.Typeflag == tar.TypeReg {
			header.Size = int64(len(entry.body))
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Typeflag == tar.TypeReg {
			if _, err := writer.Write([]byte(entry.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressor.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// treeSnapshot lists every path below root, so a test can prove a failed
// extraction left the surrounding directory exactly as it was.
func treeSnapshot(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(root, path)
		paths = append(paths, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	return paths
}

func TestArchiveExtractZipAndTarGzipWritesExactTree(t *testing.T) {
	for _, format := range []string{"zip", "tar.gz"} {
		t.Run(format, func(t *testing.T) {
			directory := t.TempDir()
			archive := filepath.Join(directory, "release."+format)
			entries := []archiveFixtureEntry{
				{name: "app/", kind: tar.TypeDir, mode: fs.ModeDir | 0o755},
				{name: "app/bin/server", body: "\x7fELF\x00\x01binary\xff", mode: 0o755},
				{name: "app/public/index.html", body: "<h1>Merhaba</h1>"},
				{name: "./app/config/settings.json", body: `{"ok":true}`},
			}
			if format == "zip" {
				writeZipFixture(t, archive, entries...)
			} else {
				writeTarGzipFixture(t, archive, entries...)
			}
			listed, err := ArchiveList(archive, AhdArchiveDefaultMaxFiles)
			if err != nil {
				t.Fatal(err)
			}
			if len(listed) != len(entries) {
				t.Fatalf("listed %d entries; want %d", len(listed), len(entries))
			}
			if kind, _ := ArchiveEntryKind(listed[0]); kind != "directory" {
				t.Fatalf("first entry kind = %q", kind)
			}
			if size, _ := ArchiveEntrySize(listed[1]); size != int64(len(entries[1].body)) {
				t.Fatalf("server size = %d", size)
			}
			destination := filepath.Join(directory, "staging")
			if err := ArchiveExtract(archive, destination, AhdArchiveDefaultMaxFiles, AhdArchiveDefaultMaxBytes); err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries[1:] {
				name := strings.TrimPrefix(entry.name, "./")
				content, err := os.ReadFile(filepath.Join(destination, filepath.FromSlash(name)))
				if err != nil || string(content) != entry.body {
					t.Fatalf("%s = %q, %v", name, content, err)
				}
			}
			if runtime.GOOS != "windows" {
				info, err := os.Stat(filepath.Join(destination, "app", "bin", "server"))
				if err != nil || info.Mode().Perm()&0o100 == 0 {
					t.Fatalf("executable bit not preserved: %v %v", info.Mode(), err)
				}
			}
			snapshot := treeSnapshot(t, directory)
			for _, path := range snapshot {
				if strings.Contains(path, ".ahdextract-") {
					t.Fatalf("staging directory left behind: %v", snapshot)
				}
			}
		})
	}
}

func TestArchiveExtractRejectsEveryUnsafeEntryWithoutWriting(t *testing.T) {
	cases := []struct {
		label, format string
		entry         archiveFixtureEntry
		want          string
	}{
		{"zip parent", "zip", archiveFixtureEntry{name: "../evil", body: "x"}, "escapes the destination"},
		{"zip grandparent", "zip", archiveFixtureEntry{name: "../../evil", body: "x"}, "escapes the destination"},
		{"zip nested parent", "zip", archiveFixtureEntry{name: "a/../../evil", body: "x"}, "escapes the destination"},
		{"zip absolute", "zip", archiveFixtureEntry{name: "/tmp/evil", body: "x"}, "absolute"},
		{"zip drive", "zip", archiveFixtureEntry{name: "C:/evil", body: "x"}, "drive prefix"},
		{"zip drive relative", "zip", archiveFixtureEntry{name: "C:evil", body: "x"}, "drive prefix"},
		{"zip UNC", "zip", archiveFixtureEntry{name: "//server/share/evil", body: "x"}, "absolute"},
		{"zip backslash", "zip", archiveFixtureEntry{name: "..\\evil", body: "x"}, "backslash"},
		{"zip mixed slashes", "zip", archiveFixtureEntry{name: "a/..\\..\\evil", body: "x"}, "backslash"},
		{"zip prefix collision", "zip", archiveFixtureEntry{name: "../out-evil/x", body: "x"}, "escapes the destination"},
		{"zip symlink", "zip", archiveFixtureEntry{name: "link", body: "/etc/passwd", mode: fs.ModeSymlink | 0o777}, "symbolic link"},
		{"tar parent", "tar.gz", archiveFixtureEntry{name: "../evil", body: "x"}, "escapes the destination"},
		{"tar absolute", "tar.gz", archiveFixtureEntry{name: "/tmp/evil", body: "x"}, "absolute"},
		{"tar symlink", "tar.gz", archiveFixtureEntry{name: "link", kind: tar.TypeSymlink, link: "../../etc"}, "symbolic link"},
		{"tar hardlink", "tar.gz", archiveFixtureEntry{name: "hard", kind: tar.TypeLink, link: "/etc/passwd"}, "hard link"},
		{"tar fifo", "tar.gz", archiveFixtureEntry{name: "pipe", kind: tar.TypeFifo}, "not a regular file"},
	}
	for _, testCase := range cases {
		t.Run(testCase.label, func(t *testing.T) {
			directory := t.TempDir()
			archive := filepath.Join(directory, "bad."+testCase.format)
			safe := archiveFixtureEntry{name: "ok.txt", body: "fine"}
			if testCase.format == "zip" {
				writeZipFixture(t, archive, safe, testCase.entry)
			} else {
				writeTarGzipFixture(t, archive, safe, testCase.entry)
			}
			root := filepath.Join(directory, "root")
			if err := os.Mkdir(root, 0o755); err != nil {
				t.Fatal(err)
			}
			before := treeSnapshot(t, directory)
			err := ArchiveExtract(archive, filepath.Join(root, "out"), AhdArchiveDefaultMaxFiles, AhdArchiveDefaultMaxBytes)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("extract error = %v; want %q", err, testCase.want)
			}
			if after := treeSnapshot(t, directory); strings.Join(after, "\n") != strings.Join(before, "\n") {
				t.Fatalf("a rejected archive changed the filesystem:\nbefore %v\nafter  %v", before, after)
			}
		})
	}
}

func TestArchiveExtractEnforcesCountAndByteBounds(t *testing.T) {
	directory := t.TempDir()
	many := filepath.Join(directory, "many.zip")
	var entries []archiveFixtureEntry
	for index := 0; index < 5; index++ {
		entries = append(entries, archiveFixtureEntry{name: "f" + string(rune('a'+index)), body: "x"})
	}
	writeZipFixture(t, many, entries...)
	if err := ArchiveExtract(many, filepath.Join(directory, "a"), 3, AhdArchiveDefaultMaxBytes); err == nil || !strings.Contains(err.Error(), "file-count limit exceeded") {
		t.Fatalf("count bound: %v", err)
	}
	if _, err := ArchiveList(many, 3); err == nil || !strings.Contains(err.Error(), "file-count limit exceeded") {
		t.Fatalf("list count bound: %v", err)
	}

	// A ZIP whose metadata declares an absurd size is refused before any byte
	// is written.
	huge := filepath.Join(directory, "huge.zip")
	writeZipFixture(t, huge, archiveFixtureEntry{name: "bomb", body: "tiny", declare: 1 << 50})
	if err := ArchiveExtract(huge, filepath.Join(directory, "b"), 10, AhdArchiveDefaultMaxBytes); err == nil || !strings.Contains(err.Error(), "byte limit exceeded") {
		t.Fatalf("declared size bound: %v", err)
	}

	// Content that streams out larger than its declared size is caught.
	liar := filepath.Join(directory, "liar.zip")
	writeZipFixture(t, liar, archiveFixtureEntry{name: "liar", body: strings.Repeat("A", 4096), declare: 16})
	if err := ArchiveExtract(liar, filepath.Join(directory, "c"), 10, AhdArchiveDefaultMaxBytes); err == nil {
		t.Fatal("an entry expanding beyond its declared size was accepted")
	}

	// TAR.GZ content above maxBytes.
	big := filepath.Join(directory, "big.tar.gz")
	writeTarGzipFixture(t, big, archiveFixtureEntry{name: "big.bin", body: strings.Repeat("Z", 10000)})
	if err := ArchiveExtract(big, filepath.Join(directory, "d"), 10, 1000); err == nil || !strings.Contains(err.Error(), "byte limit exceeded") {
		t.Fatalf("tar byte bound: %v", err)
	}
	for _, bad := range [][2]int64{{0, 10}, {AhdArchiveHardMaxFiles + 1, 10}, {10, 0}, {10, AhdArchiveHardMaxBytes + 1}} {
		if err := ArchiveExtract(big, filepath.Join(directory, "e"), bad[0], bad[1]); err == nil || !strings.Contains(err.Error(), "must be between") {
			t.Fatalf("bounds %v accepted: %v", bad, err)
		}
	}
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		if _, err := os.Lstat(filepath.Join(directory, name)); err == nil {
			t.Fatalf("failed extraction left destination %s", name)
		}
	}
	for _, path := range treeSnapshot(t, directory) {
		if strings.Contains(path, ".ahdextract-") {
			t.Fatalf("staging directory left behind: %s", path)
		}
	}
}

func TestArchiveExtractRefusesExistingDestinationAndMalformedInput(t *testing.T) {
	directory := t.TempDir()
	archive := filepath.Join(directory, "ok.zip")
	writeZipFixture(t, archive, archiveFixtureEntry{name: "a.txt", body: "a"})
	existing := filepath.Join(directory, "existing")
	if err := os.Mkdir(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(existing, "keep.txt")
	if err := os.WriteFile(keep, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ArchiveExtract(archive, existing, 10, 1000); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing destination: %v", err)
	}
	if content, _ := os.ReadFile(keep); string(content) != "keep" {
		t.Fatal("existing destination content was touched")
	}

	gzipBad := filepath.Join(directory, "bad.tar.gz")
	if err := os.WriteFile(gzipBad, []byte("\x1f\x8bnot really gzip at all"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ArchiveExtract(gzipBad, filepath.Join(directory, "x"), 10, 1000); err == nil || !strings.Contains(err.Error(), "malformed archive") {
		t.Fatalf("malformed gzip: %v", err)
	}
	var truncated bytes.Buffer
	compressor := gzip.NewWriter(&truncated)
	_, _ = compressor.Write(bytes.Repeat([]byte("garbage-tar"), 100))
	_ = compressor.Close()
	tarBad := filepath.Join(directory, "bad2.tar.gz")
	if err := os.WriteFile(tarBad, truncated.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ArchiveExtract(tarBad, filepath.Join(directory, "y"), 10, 100000); err == nil || !strings.Contains(err.Error(), "malformed archive") {
		t.Fatalf("malformed tar: %v", err)
	}
	unknown := filepath.Join(directory, "notes.txt")
	if err := os.WriteFile(unknown, []byte("plain text"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ArchiveList(unknown, 10); err == nil || !strings.Contains(err.Error(), "unsupported archive format") {
		t.Fatalf("unknown format: %v", err)
	}
	missing := filepath.Join(directory, "missing.zip")
	if err := ArchiveExtract(missing, filepath.Join(directory, "z"), 10, 100); err == nil ||
		!strings.Contains(err.Error(), "cannot open the archive: no such file or directory") || strings.Contains(err.Error(), "open "+missing) {
		t.Fatalf("missing archive must fail without raw Go error text: %v", err)
	}
}

func TestArchiveListReportsLinksWithoutFollowingThem(t *testing.T) {
	directory := t.TempDir()
	archive := filepath.Join(directory, "links.tar.gz")
	writeTarGzipFixture(t, archive,
		archiveFixtureEntry{name: "link", kind: tar.TypeSymlink, link: "/etc/passwd"},
		archiveFixtureEntry{name: "hard", kind: tar.TypeLink, link: "link"},
		archiveFixtureEntry{name: "../evil", body: "x"})
	entries, err := ArchiveList(archive, 10)
	if err != nil {
		t.Fatal(err)
	}
	kinds := []string{}
	for _, entry := range entries {
		kind, _ := ArchiveEntryKind(entry)
		kinds = append(kinds, kind)
	}
	if strings.Join(kinds, ",") != "symlink,hardlink,file" {
		t.Fatalf("kinds = %v", kinds)
	}
	if path, _ := ArchiveEntryPath(entries[2]); path != "../evil" {
		t.Fatalf("list must report the stored path verbatim; got %q", path)
	}
	expectRaise(t, AhdClassArchiveError, func() { AhdArchiveExtract(AhdClassArchiveError, archive, filepath.Join(directory, "out"), 10, 100) })
}

// TestArchiveExtractAcceptsPAXGlobalHeaders covers real-world tarballs:
// `git archive` writes a pax_global_header record (Typeflag 'g') that carries
// metadata, not a file. It must be skipped, not rejected or listed.
func TestArchiveExtractAcceptsPAXGlobalHeaders(t *testing.T) {
	directory := t.TempDir()
	var buffer bytes.Buffer
	compressor := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(compressor)
	if err := writer.WriteHeader(&tar.Header{Typeflag: tar.TypeXGlobalHeader, Name: "pax_global_header",
		PAXRecords: map[string]string{"comment": "0123456789abcdef0123456789abcdef01234567"}}); err != nil {
		t.Fatal(err)
	}
	body := "release"
	if err := writer.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: "app/VERSION", Mode: 0o644, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	_, _ = writer.Write([]byte(body))
	_ = writer.Close()
	_ = compressor.Close()
	archive := filepath.Join(directory, "git.tar.gz")
	if err := os.WriteFile(archive, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := ArchiveList(archive, 10)
	if err != nil || len(entries) != 1 {
		t.Fatalf("list = %v, %v; want only app/VERSION", entries, err)
	}
	if err := ArchiveExtract(archive, filepath.Join(directory, "out"), 10, 1000); err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(filepath.Join(directory, "out", "app", "VERSION")); string(content) != body {
		t.Fatalf("content = %q", content)
	}
}
