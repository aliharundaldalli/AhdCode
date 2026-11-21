package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/localdev"
)

func TestDevHTTPSArgumentsAndLoopbackURL(t *testing.T) {
	for _, tc := range []struct {
		args          []string
		valid, secure bool
	}{
		{[]string{"app.ahd"}, true, false},
		{[]string{"--https", "app.ahd"}, true, true},
		{[]string{"--https", "--cert", "a.pem", "--key", "a.key", "app.ahd"}, true, true},
		{[]string{"--key", "a.key", "app.ahd"}, false, false},
		{[]string{"--https", "--cert", "a.pem", "app.ahd"}, false, true},
		{[]string{"--https", "--cert=", "--key=", "app.ahd"}, false, true},
		{[]string{"app.ahd", "--https"}, false, false},
	} {
		entry, https, _, err := parseDevArguments(tc.args, io.Discard)
		if (err == nil) != tc.valid || https != tc.secure || (tc.valid && entry != "app.ahd") {
			t.Errorf("%v: entry=%q https=%v err=%v", tc.args, entry, https, err)
		}
	}
	for host, want := range map[string]string{
		"localhost": "https://localhost:8443", "127.0.0.1": "https://127.0.0.1:8443",
		"::1": "https://[::1]:8443", "0.0.0.0": "", "example.test": "",
	} {
		got, err := devHTTPSWebURL(webEnvironment{serverHost: host, serverPort: "8443"})
		if got != want || (err == nil) != (want != "") {
			t.Errorf("%q: url=%q err=%v", host, got, err)
		}
	}
}

func newDevHTTPSTestHome(t *testing.T) string {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(base, "private")
	if err := mkdirPrivateDevHTTPS(path); err != nil {
		t.Fatal(err)
	}
	t.Setenv(localdev.HomeEnvKey, path)
	return path
}

func TestDevHTTPSCertificateStore(t *testing.T) {
	home := newDevHTTPSTestHome(t)
	entry := filepath.Join(t.TempDir(), "app.ahd")
	files, ca, err := prepareDevHTTPS(entry, devHTTPSFiles{})
	if err != nil {
		t.Fatal(err)
	}
	if files.key != filepath.Join(home, "https", "localhost-key.pem") || ca != filepath.Join(home, "https", "root-ca.pem") {
		t.Fatalf("unexpected layout: %+v %q", files, ca)
	}
	if err := checkDevHTTPSPrivacy(files.key); err != nil {
		t.Fatal(err)
	}
	caBytes, err := os.ReadFile(ca)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(caBytes, []byte("PRIVATE KEY")) {
		t.Fatal("CA file contains a private key")
	}
	entries, err := os.ReadDir(filepath.Dir(ca))
	if err != nil || len(entries) != 3 {
		t.Fatalf("store entries = %v, %v", entries, err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caBytes) {
		t.Fatal("invalid public CA")
	}
	config, err := ahdruntime.AhdHTTPDevLoadTLS(files.cert, files.key, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	rootPEM, _ := pem.Decode(caBytes)
	root, err := x509.ParseCertificate(rootPEM.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(root.SubjectKeyId) == 0 || !bytes.Equal(config.Certificates[0].Leaf.AuthorityKeyId, root.SubjectKeyId) {
		t.Fatal("the leaf authority key identifier must match the issuing CA's subject key identifier")
	}
	for _, host := range []string{"localhost", "127.0.0.1", "::1"} {
		if _, err := config.Certificates[0].Leaf.Verify(x509.VerifyOptions{Roots: roots, DNSName: host}); err != nil {
			t.Fatalf("%s: %v", host, err)
		}
	}
	if err := config.Certificates[0].Leaf.VerifyHostname("outside.test"); err == nil {
		t.Fatal("non-loopback identity matched")
	}
	if config.MinVersion != tls.VersionTLS12 || len(config.NextProtos) != 1 || config.NextProtos[0] != "http/1.1" {
		t.Fatalf("unexpected TLS policy: %x %v", config.MinVersion, config.NextProtos)
	}
	again, public, err := prepareDevHTTPS(entry, devHTTPSFiles{})
	if err != nil || again != files || public != ca {
		t.Fatalf("reuse: %+v %q %v", again, public, err)
	}
	after, err := os.ReadFile(ca)
	if err != nil || !bytes.Equal(caBytes, after) {
		t.Fatal("existing CA was replaced")
	}
	if err := os.WriteFile(ca, []byte("corrupted"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := prepareDevHTTPS(entry, devHTTPSFiles{}); err == nil || !strings.Contains(err.Error(), "CA") {
		t.Fatalf("corrupt CA accepted: %v", err)
	}
}

func TestDevHTTPSInvalidCredentials(t *testing.T) {
	home := newDevHTTPSTestHome(t)
	current := time.Now()
	entry := filepath.Join(t.TempDir(), "app.ahd")
	files, _, err := prepareDevHTTPS(entry, devHTTPSFiles{})
	if err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(home, "other")
	if err := mkdirPrivateDevHTTPS(other); err != nil {
		t.Fatal(err)
	}
	if err := writeDevHTTPSCertificate(other, current); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, cert, key, message string
		now                      time.Time
	}{
		{"expired", files.cert, files.key, "expired", current.AddDate(0, 0, 31)},
		{"future", files.cert, files.key, "not yet valid", current.Add(-time.Hour)},
		{"wrong key", files.cert, filepath.Join(other, "localhost-key.pem"), "matching PEM pair", current},
		{"missing certificate", filepath.Join(other, "missing-cert.pem"), files.key, "certificate", current},
		{"missing key", files.cert, filepath.Join(other, "missing.pem"), "private key", current},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ahdruntime.AhdHTTPDevLoadTLS(tc.cert, tc.key, tc.now); err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("expected %q: %v", tc.message, err)
			}
		})
	}
	certPEM, err := os.ReadFile(files.cert)
	if err != nil {
		t.Fatal(err)
	}
	mixed := filepath.Join(other, "mixed.pem")
	marker := []byte("PRIVATE-KEY-FIXTURE-MUST-NOT-APPEAR-IN-DIAGNOSTICS")
	if err := os.WriteFile(mixed, append(certPEM, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: marker})...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ahdruntime.AhdHTTPDevLoadTLS(mixed, files.key, current); err == nil || !strings.Contains(err.Error(), "only PEM certificates") {
		t.Fatalf("mixed certificate and private key file accepted: %v", err)
	}
	keyInProject := filepath.Join(filepath.Dir(entry), "public", "server-key.pem")
	if _, _, err := prepareDevHTTPS(entry, devHTTPSFiles{files.cert, keyInProject}); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("project key not refused: %v", err)
	}
	linked := filepath.Join(other, "alias.pem")
	t.Run("linked key", func(t *testing.T) {
		if err := os.Symlink(files.key, linked); err != nil {
			t.Skipf("file symlinks unavailable: %v", err)
		}
		if _, _, err := prepareDevHTTPS(entry, devHTTPSFiles{files.cert, linked}); err == nil {
			t.Fatal("symbolic link accepted")
		}
	})
	const secretMarker = "must-not-echo-corrupted-private-key"
	if err := os.WriteFile(files.key, []byte(secretMarker), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := prepareDevHTTPS(entry, devHTTPSFiles{files.cert, files.key}); err == nil || strings.Contains(err.Error(), secretMarker) {
		t.Fatalf("invalid key accepted or leaked in error: %v", err)
	}
}
