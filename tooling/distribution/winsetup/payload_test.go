package winsetup

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type entry struct {
	name    string
	content string
}

// buildPayload assembles a payload shaped exactly like the release builder's:
// files, a VERSION marker, and a FILES.json inventory covering every file.
func buildPayload(t *testing.T, entries []entry, corrupt map[string]string) *zip.Reader {
	t.Helper()
	inventory := map[string]string{}
	for _, e := range entries {
		digest := sha256.Sum256([]byte(e.content))
		inventory[e.name] = hex.EncodeToString(digest[:])
	}
	for name, value := range corrupt {
		inventory[name] = value
	}
	recorded, err := json.Marshal(inventory)
	if err != nil {
		t.Fatal(err)
	}
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, e := range entries {
		file, err := writer.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(file, e.content); err != nil {
			t.Fatal(err)
		}
	}
	file, err := writer.Create("FILES.json")
	if err != nil {
		t.Fatal(err)
	}
	file.Write(recorded)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return reader
}

func goodPayload(t *testing.T) *zip.Reader {
	return buildPayload(t, []entry{
		{"VERSION", "1.0.0-rc.1\n"},
		{"bin/ahdcode.exe", "the real command"},
		{"launcher/ahdcode.exe", "the stable launcher"},
		{"libexec/go/bin/go.exe", "private toolchain"},
		{"uninstall.ps1", "removal script"},
	}, nil)
}

// Scenario 1: a clean installation unpacks everything and verifies it.
func TestExtractWritesAndVerifiesEveryFile(t *testing.T) {
	archive := goodPayload(t)
	inventory, err := ReadInventory(archive)
	if err != nil {
		t.Fatal(err)
	}
	staging := t.TempDir()
	calls := 0
	if err := Extract(archive, staging, inventory, func(written, total int) { calls++ }); err != nil {
		t.Fatalf("Extract = %v", err)
	}
	if calls == 0 {
		t.Error("Extract never reported progress")
	}
	for _, name := range []string{"bin/ahdcode.exe", "launcher/ahdcode.exe", "libexec/go/bin/go.exe"} {
		if _, err := os.Stat(filepath.Join(staging, filepath.FromSlash(name))); err != nil {
			t.Errorf("%s was not installed: %v", name, err)
		}
	}
	content, err := os.ReadFile(filepath.Join(staging, "bin", "ahdcode.exe"))
	if err != nil || string(content) != "the real command" {
		t.Fatalf("installed content = %q, %v", content, err)
	}
	version, err := PayloadVersion(archive)
	if err != nil || version != "1.0.0-rc.1" {
		t.Fatalf("PayloadVersion = %q, %v", version, err)
	}
}

// Scenario 12: a damaged payload must fail, not produce a usable install.
func TestExtractRejectsAChecksumMismatch(t *testing.T) {
	archive := buildPayload(t, []entry{
		{"VERSION", "1.0.0-rc.1\n"},
		{"bin/ahdcode.exe", "the real command"},
	}, map[string]string{"bin/ahdcode.exe": strings.Repeat("00", 32)})
	inventory, err := ReadInventory(archive)
	if err != nil {
		t.Fatal(err)
	}
	err = Extract(archive, t.TempDir(), inventory, nil)
	if err == nil {
		t.Fatal("Extract accepted a file whose checksum did not match")
	}
	if !strings.Contains(err.Error(), "did not match its checksum") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractRejectsAnIncompletePayload(t *testing.T) {
	archive := buildPayload(t, []entry{
		{"VERSION", "1.0.0-rc.1\n"},
	}, map[string]string{"bin/ahdcode.exe": strings.Repeat("ab", 32)})
	inventory, err := ReadInventory(archive)
	if err != nil {
		t.Fatal(err)
	}
	err = Extract(archive, t.TempDir(), inventory, nil)
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("Extract on an incomplete payload = %v", err)
	}
}

func TestExtractRefusesToEscapeTheStagingDirectory(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, _ := writer.Create("../escaped.txt")
	io.WriteString(file, "should never be written")
	writer.Close()
	archive, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		t.Fatal(err)
	}
	staging := t.TempDir()
	if err := Extract(archive, staging, map[string]string{}, nil); err == nil {
		t.Fatal("Extract accepted a path that escapes the staging directory")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(staging), "escaped.txt")); err == nil {
		t.Fatal("a file was written outside the staging directory")
	}
}

func TestReadInventoryAndVersionRejectDamagedPayloads(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, _ := writer.Create("bin/ahdcode.exe")
	io.WriteString(file, "x")
	writer.Close()
	archive, _ := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if _, err := ReadInventory(archive); err == nil {
		t.Error("ReadInventory accepted a payload with no inventory")
	}
	if _, err := PayloadVersion(archive); err == nil {
		t.Error("PayloadVersion accepted a payload with no release identity")
	}
}

// Scenario 13: reinstalling must not disturb anything outside the version
// folder. Extract only ever writes inside the staging directory it is given.
func TestExtractOnlyTouchesItsStagingDirectory(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "my-project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	database := filepath.Join(project, "app.db")
	if err := os.WriteFile(database, []byte("user data"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(database)
	if err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(parent, "staging")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		t.Fatal(err)
	}
	archive := goodPayload(t)
	inventory, _ := ReadInventory(archive)
	if err := Extract(archive, staging, inventory, nil); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(database)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("the user's database changed during installation: %v", err)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("installation created %d entries beside the project, want 2", len(entries))
	}
}
