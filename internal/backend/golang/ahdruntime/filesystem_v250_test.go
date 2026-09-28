package ahdruntime

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func requireNoTemporaryFiles(t *testing.T, directory string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".ahdtmp-") {
			t.Fatalf("temporary file left behind: %s", entry.Name())
		}
	}
}

func TestFileCopyIsByteExactAndNeverOverwrites(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "app.bin")
	payload := make([]byte, 3*64*1024+17) // spans several copy buffers
	if _, err := rand.Read(payload); err != nil {
		t.Fatal(err)
	}
	payload[0], payload[1] = 0x00, 0xff // bytes that are not valid UTF-8
	if err := os.WriteFile(source, payload, 0o750); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(directory, "copy.bin")
	if err := FileCopy(source, destination); err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(copied, payload) {
		t.Fatalf("copy is not byte-exact (err=%v)", err)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(destination)
		if info.Mode().Perm() != 0o750 {
			t.Fatalf("copy mode = %v; want the source's 0750", info.Mode().Perm())
		}
	}
	if err := FileCopy(source, destination); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("second copy must refuse to overwrite: %v", err)
	}
	if err := FileCopy(filepath.Join(directory, "missing"), filepath.Join(directory, "x")); err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("missing source: %v", err)
	}
	if err := FileCopy(directory, filepath.Join(directory, "y")); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("directory source: %v", err)
	}
	if runtime.GOOS != "windows" {
		link := filepath.Join(directory, "link")
		if err := os.Symlink(source, link); err != nil {
			t.Fatal(err)
		}
		if err := FileCopy(link, filepath.Join(directory, "z")); err == nil || !strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("symlink source must be refused: %v", err)
		}
	}
	if err := FileCopy(source, filepath.Join(directory, "no-such-dir", "copy")); err == nil {
		t.Fatal("copy into a missing directory succeeded")
	}
	requireNoTemporaryFiles(t, directory)
	expectRaise(t, AhdClassFileError, func() { AhdFileCopy(AhdClassFileError, source, destination) })
}

func TestFileAtomicWriteReplacesAndCleansUp(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "config.json")
	if err := FileAtomicWrite(path, `{"v":1}`); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if info, _ := os.Stat(path); info.Mode().Perm() != 0o644 {
			t.Fatalf("new file mode = %v", info.Mode().Perm())
		}
		if err := os.Chmod(path, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := FileAtomicWrite(path, `{"v":2,"ad":"Çağrı"}`); err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(path); string(content) != `{"v":2,"ad":"Çağrı"}` {
		t.Fatalf("content = %q", content)
	}
	if runtime.GOOS != "windows" {
		if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
			t.Fatalf("replacement must keep the existing mode; got %v", info.Mode().Perm())
		}
	}
	if err := FileAtomicWrite(path, "\xff\xfe"); err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("invalid UTF-8: %v", err)
	}
	if err := FileAtomicWrite(directory, "x"); err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("directory target: %v", err)
	}
	if err := FileAtomicWrite(filepath.Join(directory, "missing", "x"), "x"); err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("missing parent: %v", err)
	}
	if content, _ := os.ReadFile(path); string(content) != `{"v":2,"ad":"Çağrı"}` {
		t.Fatal("a failed atomic write changed the existing file")
	}
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		locked := filepath.Join(directory, "locked")
		if err := os.Mkdir(locked, 0o500); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(locked, 0o700)
		if err := FileAtomicWrite(filepath.Join(locked, "x"), "x"); err == nil || !strings.Contains(err.Error(), "permission denied") {
			t.Fatalf("read-only directory: %v", err)
		}
		requireNoTemporaryFiles(t, locked)
	}
	requireNoTemporaryFiles(t, directory)
}

func TestFileAtomicMoveRenamesWithoutCopying(t *testing.T) {
	directory := t.TempDir()
	staging := filepath.Join(directory, "staging")
	if err := os.MkdirAll(filepath.Join(staging, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "app", "index.html"), []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	release := filepath.Join(directory, "release-2")
	if err := FileAtomicMove(staging, release); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(staging); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("source still exists after the move")
	}
	if content, _ := os.ReadFile(filepath.Join(release, "app", "index.html")); string(content) != "v2" {
		t.Fatal("moved tree is incomplete")
	}
	// A file replaces an existing file.
	first, second := filepath.Join(directory, "a.txt"), filepath.Join(directory, "b.txt")
	_ = os.WriteFile(first, []byte("new"), 0o644)
	_ = os.WriteFile(second, []byte("old"), 0o644)
	if err := FileAtomicMove(first, second); err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(second); string(content) != "new" {
		t.Fatal("replacement did not happen")
	}
	// An existing directory is never replaced, even an empty one.
	empty := filepath.Join(directory, "empty")
	_ = os.Mkdir(empty, 0o755)
	if err := FileAtomicMove(second, empty); err == nil || !strings.Contains(err.Error(), "existing directory") {
		t.Fatalf("directory destination: %v", err)
	}
	if err := FileAtomicMove(filepath.Join(directory, "missing"), filepath.Join(directory, "x")); err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("missing source: %v", err)
	}
	// Cross-device classification (EXDEV / ERROR_NOT_SAME_DEVICE).
	crossDevice := &os.LinkError{Op: "rename", Old: "a", New: "b", Err: syscall.Errno(18)}
	if runtime.GOOS == "windows" {
		crossDevice.Err = syscall.Errno(17)
	}
	if !ahdFSIsCrossDevice(crossDevice) || ahdFSReason(crossDevice) != "source and destination are on different filesystems" {
		t.Fatalf("cross-device error not recognized: %q", ahdFSReason(crossDevice))
	}
}

func TestFileSymlinkOperationsAndReleaseSwitch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic link creation needs Developer Mode on Windows")
	}
	directory := t.TempDir()
	for _, name := range []string{"releases/41", "releases/42"} {
		if err := os.MkdirAll(filepath.Join(directory, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	current := filepath.Join(directory, "current")
	if err := FileSymlink("releases/41", current); err != nil {
		t.Fatal(err)
	}
	if target, err := FileReadLink(current); err != nil || target != "releases/41" {
		t.Fatalf("readLink = %q, %v", target, err)
	}
	if is, err := FileIsSymlink(current); err != nil || !is {
		t.Fatalf("isSymlink(current) = %v, %v", is, err)
	}
	if is, err := FileIsSymlink(filepath.Join(directory, "releases")); err != nil || is {
		t.Fatal("a directory reported as a link")
	}
	if is, err := FileIsSymlink(filepath.Join(directory, "missing")); err != nil || is {
		t.Fatal("a missing path reported as a link")
	}
	if err := FileSymlink("releases/42", current); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("symlink over an existing path: %v", err)
	}
	// The deployment switch: new link beside, then atomic rename over it.
	next := filepath.Join(directory, ".current-next")
	if err := FileSymlink("releases/42", next); err != nil {
		t.Fatal(err)
	}
	if err := FileAtomicMove(next, current); err != nil {
		t.Fatal(err)
	}
	if target, _ := FileReadLink(current); target != "releases/42" {
		t.Fatalf("switched link = %q", target)
	}
	if _, err := FileReadLink(filepath.Join(directory, "releases")); err == nil || !strings.Contains(err.Error(), "not a symbolic link") {
		t.Fatalf("readLink on a directory: %v", err)
	}
	if err := FileSymlink("", filepath.Join(directory, "empty")); err == nil {
		t.Fatal("an empty target was accepted")
	}
}

func TestFilePermissionsUseOctalStringsHonestly(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "run.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		if _, err := FilePermissions(path); err == nil || !strings.Contains(err.Error(), "not supported on Windows") {
			t.Fatalf("Windows permissions must fail clearly: %v", err)
		}
		if err := FileSetPermissions(path, "0755"); err == nil || !strings.Contains(err.Error(), "not supported on Windows") {
			t.Fatalf("Windows setPermissions must fail clearly: %v", err)
		}
		return
	}
	for _, mode := range []string{"0755", "640", "0000", "777"} {
		if err := FileSetPermissions(path, mode); err != nil {
			t.Fatalf("mode %q: %v", mode, err)
		}
		got, err := FilePermissions(path)
		want := mode
		if len(want) == 3 {
			want = "0" + want
		}
		if err != nil || got != want {
			t.Fatalf("permissions after %q = %q, %v", mode, got, err)
		}
	}
	for _, bad := range []string{"", "755 ", "0o755", "0x1ed", "1755", "4755", "7777", "00755", "0800", "75", "rwxr-xr-x", "-755", "493"} {
		if err := FileSetPermissions(path, bad); err == nil || !strings.Contains(err.Error(), "three octal digits") {
			t.Fatalf("mode %q accepted or wrong error: %v", bad, err)
		}
	}
	if got, _ := FilePermissions(path); got != "0777" {
		t.Fatalf("a rejected mode changed the file: %s", got)
	}
	link := filepath.Join(directory, "link")
	_ = os.Symlink(path, link)
	if err := FileSetPermissions(link, "0644"); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("setPermissions through a link: %v", err)
	}
	if _, err := FilePermissions(link); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("permissions of a link: %v", err)
	}
	if got, _ := FilePermissions(path); got != "0777" {
		t.Fatal("a refused link operation changed its target")
	}
}

func TestFileWalkNeverFollowsLinksAndIsBounded(t *testing.T) {
	directory := t.TempDir()
	root := filepath.Join(directory, "site")
	for _, name := range []string{"b/deep", "a"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.WriteFile(filepath.Join(root, "a", "index.html"), []byte("12345"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "b", "deep", "x.txt"), []byte("x"), 0o644)
	outside := filepath.Join(directory, "outside")
	_ = os.MkdirAll(outside, 0o755)
	_ = os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644)
	links := runtime.GOOS != "windows"
	if links {
		if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("..", filepath.Join(root, "b", "loop")); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := FileWalk(root, AhdFileWalkDefaultMaxEntries)
	if err != nil {
		t.Fatal(err)
	}
	var seen []string
	for _, entry := range entries {
		relative, _ := FileEntryRelativePath(entry)
		kind, _ := FileEntryKind(entry)
		seen = append(seen, relative+":"+kind)
		if strings.Contains(relative, "secret") {
			t.Fatal("walk followed a symbolic link out of the root")
		}
		path, _ := FileEntryPath(entry)
		if path != filepath.Join(root, filepath.FromSlash(relative)) {
			t.Fatalf("path %q does not match relative %q", path, relative)
		}
	}
	want := []string{"a:directory", "a/index.html:file", "b:directory", "b/deep:directory", "b/deep/x.txt:file"}
	if links {
		want = []string{"a:directory", "a/index.html:file", "b:directory", "b/deep:directory", "b/deep/x.txt:file", "b/loop:symlink", "escape:symlink"}
	}
	if strings.Join(seen, ",") != strings.Join(want, ",") {
		t.Fatalf("walk = %v\nwant   %v", seen, want)
	}
	if size, _ := FileEntrySize(entries[1]); size != 5 {
		t.Fatalf("file size = %d", size)
	}
	if links {
		if is, _ := FileEntryIsSymlink(entries[len(entries)-1]); !is {
			t.Fatal("isSymlink false for a link entry")
		}
	}
	if _, err := FileWalk(root, 3); err == nil || !strings.Contains(err.Error(), "more than 3 entries") {
		t.Fatalf("maxEntries bound: %v", err)
	}
	for _, bad := range []int64{0, -1, AhdFileWalkHardMaxEntries + 1} {
		if _, err := FileWalk(root, bad); err == nil || !strings.Contains(err.Error(), "must be between") {
			t.Fatalf("maxEntries %d accepted: %v", bad, err)
		}
	}
	if links {
		if _, err := FileWalk(filepath.Join(root, "escape"), 10); err == nil || !strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("symlink root: %v", err)
		}
	}
	if _, err := FileWalk(filepath.Join(root, "a", "index.html"), 10); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("file root: %v", err)
	}
	// The evaluator variant reports paths as the program wrote them.
	displayed, err := FileWalkAs(root, "site", 10)
	if err != nil {
		t.Fatal(err)
	}
	if path, _ := FileEntryPath(displayed[0]); path != filepath.Join("site", "a") {
		t.Fatalf("display path = %q", path)
	}
}
