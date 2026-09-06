package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/localdev"
)

func isolatedLocalHome(t *testing.T) {
	t.Helper()
	t.Setenv(localdev.HomeEnvKey, t.TempDir())
}

func writeHosts(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hosts")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestHostSyncFirstUseInteractiveApproval(t *testing.T) {
	isolatedLocalHome(t)
	original := "127.0.0.1 localhost\n10.0.0.5 build.internal\n"
	path := writeHosts(t, original)
	var out, errOut bytes.Buffer
	result := syncManagedHostname(hostSyncRequest{
		hostname:    "bio.test",
		path:        path,
		input:       strings.NewReader("\n"),
		output:      &out,
		errorOutput: &errOut,
		interactive: true,
	})
	if !result.mapped || !result.mutated {
		t.Fatalf("first-use approval did not apply: %+v\n%s%s", result, out.String(), errOut.String())
	}
	if !localdev.HostsAuthorized() {
		t.Fatal("approval did not record host integration authorization")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(content), original) {
		t.Fatalf("unrelated hosts content changed:\n%s", content)
	}
	if !localdev.HostsMapsLoopback(string(content), "bio.test") {
		t.Fatalf("bio.test was not mapped:\n%s", content)
	}
	if !strings.Contains(out.String(), "Enable local .test names?") {
		t.Fatalf("first use did not ask for consent:\n%s", out.String())
	}
}

func TestHostSyncAlreadyAuthorizedAddsNewHostname(t *testing.T) {
	isolatedLocalHome(t)
	if err := localdev.MarkHostsAuthorized(); err != nil {
		t.Fatal(err)
	}
	path := writeHosts(t, "127.0.0.1 localhost\n# BEGIN AHDCODE LOCAL\n127.0.0.1 bio.test\n# END AHDCODE LOCAL\n")
	var out bytes.Buffer
	result := syncManagedHostname(hostSyncRequest{
		hostname:    "ahdakademi.test",
		path:        path,
		input:       strings.NewReader("this must not be read"),
		output:      &out,
		errorOutput: ioDiscard(),
		interactive: true,
	})
	if !result.mapped {
		t.Fatalf("authorized refresh failed: %+v\n%s", result, out.String())
	}
	if strings.Contains(out.String(), "Enable local .test names?") {
		t.Fatalf("already-authorized sync asked again:\n%s", out.String())
	}
	content, _ := os.ReadFile(path)
	if !localdev.HostsMapsLoopback(string(content), "ahdakademi.test") || !localdev.HostsMapsLoopback(string(content), "bio.test") {
		t.Fatalf("managed names were not preserved:\n%s", content)
	}
	if !strings.Contains(string(content), "127.0.0.1 localhost") {
		t.Fatalf("unrelated content was lost:\n%s", content)
	}
}

func TestHostSyncRefusalLeavesHostsAlone(t *testing.T) {
	isolatedLocalHome(t)
	original := "127.0.0.1 localhost\n"
	path := writeHosts(t, original)
	var out bytes.Buffer
	result := syncManagedHostname(hostSyncRequest{
		hostname:    "bio.test",
		path:        path,
		input:       strings.NewReader("n\n"),
		output:      &out,
		errorOutput: ioDiscard(),
		interactive: true,
	})
	if result.mutated || result.mapped || !result.refused {
		t.Fatalf("refusal result: %+v", result)
	}
	content, _ := os.ReadFile(path)
	if string(content) != original {
		t.Fatalf("refusal mutated hosts:\n%s", content)
	}
	if localdev.HostsAuthorized() {
		t.Fatal("refusal recorded authorization")
	}
}

func TestHostSyncNonTTYDoesNotMutate(t *testing.T) {
	isolatedLocalHome(t)
	original := "127.0.0.1 localhost\n"
	path := writeHosts(t, original)
	var out bytes.Buffer
	result := syncManagedHostname(hostSyncRequest{
		hostname:    "bio.test",
		path:        path,
		input:       strings.NewReader("y\n"),
		output:      &out,
		errorOutput: ioDiscard(),
		interactive: false,
	})
	if result.mutated || result.mapped || !result.nonTTY {
		t.Fatalf("non-TTY result: %+v", result)
	}
	content, _ := os.ReadFile(path)
	if string(content) != original {
		t.Fatalf("non-TTY mutated hosts:\n%s", content)
	}
}

func TestHostSyncPreservesUnrelatedBytes(t *testing.T) {
	isolatedLocalHome(t)
	if err := localdev.MarkHostsAuthorized(); err != nil {
		t.Fatal(err)
	}
	original := "##\n# Host Database\n##\n127.0.0.1\tlocalhost\n10.0.0.5 build.internal\n"
	path := writeHosts(t, original)
	_ = syncManagedHostname(hostSyncRequest{
		hostname:    "checkmate.test",
		path:        path,
		output:      ioDiscard(),
		errorOutput: ioDiscard(),
		interactive: false,
	})
	content, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(content), original) {
		t.Fatalf("bytes outside the managed block changed:\n%s", content)
	}
}

func ioDiscard() *bytes.Buffer { return &bytes.Buffer{} }
