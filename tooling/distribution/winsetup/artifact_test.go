package winsetup

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestRealWindowsPayload runs a built Windows setup program's own payload
// through the code the installer uses on Windows. It is skipped unless
// AHDCODE_WINDOWS_SETUP names a built artifact, so ordinary `go test` stays
// fast while release verification can exercise the real bytes.
func TestRealWindowsPayload(t *testing.T) {
	artifact := os.Getenv("AHDCODE_WINDOWS_SETUP")
	if artifact == "" {
		t.Skip("set AHDCODE_WINDOWS_SETUP to a built Windows setup program to run this")
	}
	data, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := embeddedPayload(data)
	if err != nil {
		t.Fatal(err)
	}
	version, err := PayloadVersion(archive)
	if err != nil {
		t.Fatalf("PayloadVersion = %v", err)
	}
	t.Logf("payload release identity: %s", version)

	inventory, err := ReadInventory(archive)
	if err != nil {
		t.Fatalf("ReadInventory = %v", err)
	}
	staging := t.TempDir()
	if err := Extract(archive, staging, inventory, nil); err != nil {
		t.Fatalf("Extract on the real payload = %v", err)
	}
	// The two executables the installed product depends on must be present:
	// the stable launcher that goes on PATH, and the release it activates.
	for _, name := range []string{
		filepath.Join("launcher", "ahdcode.exe"),
		filepath.Join("bin", "ahdcode.exe"),
		filepath.Join("libexec", "go", "bin", "go.exe"),
		"uninstall.ps1",
	} {
		information, err := os.Stat(filepath.Join(staging, name))
		if err != nil || !information.Mode().IsRegular() || information.Size() == 0 {
			t.Errorf("%s missing or empty: %v", name, err)
		}
	}
	t.Logf("verified %d inventoried files", len(inventory))
}

// embeddedPayload finds the zip the setup program carries.
func embeddedPayload(data []byte) (*zip.Reader, error) {
	end := bytes.LastIndex(data, []byte("PK\x05\x06"))
	start := bytes.Index(data, []byte("PK\x03\x04"))
	var lastErr error
	for start >= 0 && end > start {
		section := data[start : end+22]
		reader, err := zip.NewReader(bytes.NewReader(section), int64(len(section)))
		if err == nil {
			return reader, nil
		}
		lastErr = err
		next := bytes.Index(data[start+1:], []byte("PK\x03\x04"))
		if next < 0 {
			break
		}
		start += 1 + next
	}
	return nil, lastErr
}
