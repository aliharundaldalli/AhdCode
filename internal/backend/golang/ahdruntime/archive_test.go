package ahdruntime

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func archiveTestPair(entries ...string) *AhdPair[string, string] {
	if len(entries)%2 != 0 {
		panic("archiveTestPair requires an even number of key/value strings")
	}
	keys := make([]string, 0, len(entries)/2)
	values := make([]string, 0, len(entries)/2)
	for index := 0; index < len(entries); index += 2 {
		keys = append(keys, entries[index])
		values = append(values, entries[index+1])
	}
	return AhdBuildPair(keys, values)
}

func archiveTestSourceFile(t *testing.T, directory, name, content string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestArchiveZipRoundTrip(t *testing.T) {
	directory := t.TempDir()
	reportPath := archiveTestSourceFile(t, directory, "report.pdf", "pdf-bytes")
	dataPath := archiveTestSourceFile(t, directory, "results.json", `{"ok":true}`)
	output := filepath.Join(directory, "out.zip")

	AhdArchiveZip(output, archiveTestPair(
		"report/report.pdf", reportPath,
		"data/results.json", dataPath,
	))

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"report/report.pdf": "pdf-bytes", "data/results.json": `{"ok":true}`}
	if len(reader.File) != len(want) {
		t.Fatalf("member count = %d; want %d", len(reader.File), len(want))
	}
	for _, file := range reader.File {
		expected, known := want[file.Name]
		if !known {
			t.Fatalf("unexpected member %q", file.Name)
		}
		opened, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(opened)
		_ = opened.Close()
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != expected {
			t.Fatalf("member %q content = %q; want %q", file.Name, content, expected)
		}
	}
}

func TestArchiveTarRoundTrip(t *testing.T) {
	directory := t.TempDir()
	path := archiveTestSourceFile(t, directory, "a.txt", "hello tar")
	output := filepath.Join(directory, "out.tar")
	AhdArchiveTar(output, archiveTestPair("nested/a.txt", path))

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	reader := tar.NewReader(bytes.NewReader(data))
	header, err := reader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if header.Name != "nested/a.txt" {
		t.Fatalf("member name = %q", header.Name)
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello tar" {
		t.Fatalf("member content = %q", content)
	}
	if _, err := reader.Next(); err != io.EOF {
		t.Fatalf("expected exactly one member, found more")
	}
}

func TestArchiveTarGzipRoundTrip(t *testing.T) {
	directory := t.TempDir()
	path := archiveTestSourceFile(t, directory, "a.txt", "hello tar gz")
	output := filepath.Join(directory, "out.tar.gz")
	AhdArchiveTarGzip(output, archiveTestPair("a.txt", path))

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	header, err := reader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if header.Name != "a.txt" {
		t.Fatalf("member name = %q", header.Name)
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello tar gz" {
		t.Fatalf("member content = %q", content)
	}
}

func TestArchiveEmptyEntriesProduceValidArchives(t *testing.T) {
	directory := t.TempDir()
	empty := AhdBuildPair([]string{}, []string{})

	zipPath := filepath.Join(directory, "empty.zip")
	AhdArchiveZip(zipPath, empty)
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.File) != 0 {
		t.Fatalf("empty zip has %d members", len(reader.File))
	}

	tarPath := filepath.Join(directory, "empty.tar")
	AhdArchiveTar(tarPath, empty)
	tarData, err := os.ReadFile(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	tarReader := tar.NewReader(bytes.NewReader(tarData))
	if _, err := tarReader.Next(); err != io.EOF {
		t.Fatalf("empty tar has a member")
	}

	tarGzipPath := filepath.Join(directory, "empty.tar.gz")
	AhdArchiveTarGzip(tarGzipPath, empty)
	tarGzipData, err := os.ReadFile(tarGzipPath)
	if err != nil {
		t.Fatal(err)
	}
	gzipReader, err := gzip.NewReader(bytes.NewReader(tarGzipData))
	if err != nil {
		t.Fatal(err)
	}
	defer gzipReader.Close()
	if _, err := tar.NewReader(gzipReader).Next(); err != io.EOF {
		t.Fatalf("empty tar.gz has a member")
	}
}

func TestArchivePreservesPairInsertionOrder(t *testing.T) {
	directory := t.TempDir()
	pathC := archiveTestSourceFile(t, directory, "c.txt", "c")
	pathA := archiveTestSourceFile(t, directory, "a.txt", "a")
	pathB := archiveTestSourceFile(t, directory, "b.txt", "b")
	output := filepath.Join(directory, "order.tar")
	// Deliberately not lexical: c, a, b.
	AhdArchiveTar(output, archiveTestPair("c.txt", pathC, "a.txt", pathA, "b.txt", pathB))

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	reader := tar.NewReader(bytes.NewReader(data))
	var order []string
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		order = append(order, header.Name)
	}
	want := []string{"c.txt", "a.txt", "b.txt"}
	if len(order) != len(want) {
		t.Fatalf("order = %v; want %v", order, want)
	}
	for index := range want {
		if order[index] != want[index] {
			t.Fatalf("order = %v; want %v", order, want)
		}
	}
}

func TestArchiveNestedEntryPath(t *testing.T) {
	directory := t.TempDir()
	path := archiveTestSourceFile(t, directory, "chart.png", "png-bytes")
	output := filepath.Join(directory, "nested.zip")
	AhdArchiveZip(output, archiveTestPair("images/charts/2026/chart.png", path))

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.File) != 1 || reader.File[0].Name != "images/charts/2026/chart.png" {
		t.Fatalf("members = %+v", reader.File)
	}
}

func TestArchiveMemberPathTraversalRejected(t *testing.T) {
	directory := t.TempDir()
	path := archiveTestSourceFile(t, directory, "a.txt", "a")
	output := filepath.Join(directory, "out.zip")
	invalid := []string{
		"", "/absolute", "../escape", "a/../b", "./file", "a//b",
		"a\\b", "a\x00b", "C:file", ".", "..",
	}
	for _, member := range invalid {
		expectRaise(t, AhdClassArchiveError, func() { AhdArchiveZip(output, archiveTestPair(member, path)) })
	}
}

func TestArchiveMissingSourceRejected(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "out.zip")
	expectRaise(t, AhdClassArchiveError, func() {
		AhdArchiveZip(output, archiveTestPair("a.txt", filepath.Join(directory, "missing.txt")))
	})
}

func TestArchiveDirectorySourceRejected(t *testing.T) {
	directory := t.TempDir()
	sub := filepath.Join(directory, "sub")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(directory, "out.zip")
	expectRaise(t, AhdClassArchiveError, func() { AhdArchiveZip(output, archiveTestPair("sub", sub)) })
}

func TestArchiveSymlinkSourceRejected(t *testing.T) {
	directory := t.TempDir()
	target := archiveTestSourceFile(t, directory, "real.txt", "real")
	link := filepath.Join(directory, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("could not create a symlink to test against: %v", err)
	}
	output := filepath.Join(directory, "out.zip")
	expectRaise(t, AhdClassArchiveError, func() { AhdArchiveZip(output, archiveTestPair("link.txt", link)) })
}

func TestArchiveDestinationSelfInclusionRejected(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "self.zip")
	if err := os.WriteFile(output, []byte("PK\x05\x06"+string(make([]byte, 18))), 0o600); err != nil {
		t.Fatal(err)
	}
	expectRaise(t, AhdClassArchiveError, func() { AhdArchiveZip(output, archiveTestPair("self.zip", output)) })
}

func TestArchiveWrongExtensionRejected(t *testing.T) {
	directory := t.TempDir()
	path := archiveTestSourceFile(t, directory, "a.txt", "a")
	entries := archiveTestPair("a.txt", path)
	expectRaise(t, AhdClassArchiveError, func() { AhdArchiveZip(filepath.Join(directory, "out.tar"), entries) })
	expectRaise(t, AhdClassArchiveError, func() { AhdArchiveTar(filepath.Join(directory, "out.zip"), entries) })
	expectRaise(t, AhdClassArchiveError, func() { AhdArchiveTarGzip(filepath.Join(directory, "out.tgz"), entries) })
	expectRaise(t, AhdClassArchiveError, func() { AhdArchiveTarGzip(filepath.Join(directory, "out.gz"), entries) })
	// The correct extensions must still succeed.
	AhdArchiveZip(filepath.Join(directory, "ok.zip"), entries)
	AhdArchiveTar(filepath.Join(directory, "ok.tar"), entries)
	AhdArchiveTarGzip(filepath.Join(directory, "ok.tar.gz"), entries)
}

func TestArchiveDeterministicBytes(t *testing.T) {
	directory := t.TempDir()
	path := archiveTestSourceFile(t, directory, "a.txt", "deterministic content")
	entries := archiveTestPair("a.txt", path)

	firstZip := filepath.Join(directory, "first.zip")
	secondZip := filepath.Join(directory, "second.zip")
	AhdArchiveZip(firstZip, entries)
	AhdArchiveZip(secondZip, entries)
	firstData, err := os.ReadFile(firstZip)
	if err != nil {
		t.Fatal(err)
	}
	secondData, err := os.ReadFile(secondZip)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstData, secondData) {
		t.Fatal("two zips built from identical entries were not byte-identical")
	}

	firstTar := filepath.Join(directory, "first.tar")
	secondTar := filepath.Join(directory, "second.tar")
	AhdArchiveTar(firstTar, entries)
	AhdArchiveTar(secondTar, entries)
	firstTarData, _ := os.ReadFile(firstTar)
	secondTarData, _ := os.ReadFile(secondTar)
	if !bytes.Equal(firstTarData, secondTarData) {
		t.Fatal("two tars built from identical entries were not byte-identical")
	}

	firstGzip := filepath.Join(directory, "first.tar.gz")
	secondGzip := filepath.Join(directory, "second.tar.gz")
	AhdArchiveTarGzip(firstGzip, entries)
	AhdArchiveTarGzip(secondGzip, entries)
	firstGzipData, _ := os.ReadFile(firstGzip)
	secondGzipData, _ := os.ReadFile(secondGzip)
	if !bytes.Equal(firstGzipData, secondGzipData) {
		t.Fatal("two tar.gz archives built from identical entries were not byte-identical")
	}
}

func TestArchiveFailedBuildPreservesExistingDestination(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "existing.zip")
	if err := os.WriteFile(output, []byte("existing valid content"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectRaise(t, AhdClassArchiveError, func() {
		AhdArchiveZip(output, archiveTestPair("a.txt", filepath.Join(directory, "missing.txt")))
	})
	content, err := os.ReadFile(output)
	if err != nil || string(content) != "existing valid content" {
		t.Fatalf("a failed build changed the destination: %q, %v", content, err)
	}
}

func TestArchiveDuplicateMemberNameRejected(t *testing.T) {
	// Pair itself guarantees unique keys, so this exercises the defensive
	// check directly rather than via the public API (which cannot construct
	// a colliding Pair in the first place). ahdArchiveEntries is the shared,
	// non-panicking core, so this checks the returned error directly rather
	// than expecting a panic.
	directory := t.TempDir()
	path := archiveTestSourceFile(t, directory, "a.txt", "a")
	output := filepath.Join(directory, "out.zip")
	absolute, err := filepath.Abs(output)
	if err != nil {
		t.Fatal(err)
	}
	pair := archiveTestPair("a.txt", path)
	pair.keys = append(pair.keys, "a.txt")
	pair.values["a.txt"] = path
	if _, err := ahdArchiveEntries(absolute, pair); err == nil {
		t.Fatal("expected a duplicate-member error")
	}
}

// TestArchiveCoreReturnsErrorsInsteadOfPanicking proves ArchiveZip/Tar/
// TarGzip -- the shared core the evaluator now calls directly -- report every
// documented failure category as a returned Go error rather than a panic,
// while the success path through that same core still works.
func TestArchiveCoreReturnsErrorsInsteadOfPanicking(t *testing.T) {
	directory := t.TempDir()
	path := archiveTestSourceFile(t, directory, "a.txt", "a")
	entries := archiveTestPair("a.txt", path)

	if err := ArchiveZip(filepath.Join(directory, "out.tar"), entries); err == nil {
		t.Fatal("expected a wrong-extension error")
	}
	if err := ArchiveZip(filepath.Join(directory, "out.zip"), archiveTestPair("a.txt", filepath.Join(directory, "missing.txt"))); err == nil {
		t.Fatal("expected a missing-source error")
	}
	if err := ArchiveZip(filepath.Join(directory, "out.zip"), archiveTestPair("../escape", path)); err == nil {
		t.Fatal("expected an unsafe-member-path error")
	}
	if err := ArchiveTar(filepath.Join(directory, "out.tar.gz"), entries); err == nil {
		t.Fatal("expected a wrong-extension error for Archive.tar")
	}
	if err := ArchiveTarGzip(filepath.Join(directory, "out.tgz"), entries); err == nil {
		t.Fatal("expected a wrong-extension error for Archive.tarGzip")
	}

	link := filepath.Join(directory, "link.txt")
	if err := os.Symlink(path, link); err != nil {
		t.Skipf("could not create a symlink to test against: %v", err)
	}
	if err := ArchiveZip(filepath.Join(directory, "out.zip"), archiveTestPair("link.txt", link)); err == nil {
		t.Fatal("expected a symlink-source error")
	}

	// The success path must still work through the same core.
	if err := ArchiveZip(filepath.Join(directory, "ok.zip"), entries); err != nil {
		t.Fatalf("unexpected error on the success path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "ok.zip")); err != nil {
		t.Fatalf("success path did not write a real archive: %v", err)
	}
}

// TestArchiveNativeWrapperRaisesTheSameMessageTheCoreReturns proves the
// panicking native wrapper (AhdArchiveZip, used by generated programs) and
// the error-returning core (ArchiveZip, used directly by the evaluator) stay
// in lockstep: one always raises exactly the message the other returns, so
// native and REPL Archive failures can never drift apart.
func TestArchiveNativeWrapperRaisesTheSameMessageTheCoreReturns(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "out.zip")
	entries := archiveTestPair("a.txt", filepath.Join(directory, "missing.txt"))

	coreErr := ArchiveZip(output, entries)
	if coreErr == nil {
		t.Fatal("expected the shared core to return an error")
	}

	var raisedMessage string
	func() {
		defer func() {
			recovered := recover()
			signal, ok := recovered.(*AhdSignal)
			if !ok {
				t.Fatalf("expected an AhdSignal; received %v", recovered)
			}
			if signal.Instance.AhdClassOf() != AhdClassArchiveError {
				t.Fatalf("expected ArchiveError; received %s", signal.Instance.AhdClassOf().Name)
			}
			raisedMessage = signal.Message
		}()
		AhdArchiveZip(output, entries)
	}()

	if raisedMessage != coreErr.Error() {
		t.Fatalf("native raise message %q != core error message %q", raisedMessage, coreErr.Error())
	}
}
