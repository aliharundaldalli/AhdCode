package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunFileForDerivesSiblingDescriptor(t *testing.T) {
	directory := t.TempDir()
	entry := filepath.Join(directory, "app.ahd")
	if got, want := runFileFor(entry), filepath.Join(directory, "app.run"); got != want {
		t.Fatalf("runFileFor(app.ahd) = %q; want %q", got, want)
	}
	if got, want := runFileFor(filepath.Join(directory, "server")), filepath.Join(directory, "server.run"); got != want {
		t.Fatalf("runFileFor(server) = %q; want %q", got, want)
	}
}

func TestRunDescriptorRoundTripAndRemoval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.run")
	if err := startRunDescriptor(path, "app.ahd", os.Getpid()); err != nil {
		t.Fatal(err)
	}
	descriptor, err := readRunDescriptor(path)
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.Schema != runDescriptorSchema || descriptor.Version != runDescriptorVersion {
		t.Fatalf("descriptor identity = %+v", descriptor)
	}
	if descriptor.PID != os.Getpid() || !filepath.IsAbs(descriptor.Source) {
		t.Fatalf("descriptor payload = %+v", descriptor)
	}
	// A descriptor is only removed by the run it actually names.
	removeOwnRunDescriptor(path, os.Getpid()+1)
	if _, err := os.Stat(path); err != nil {
		t.Fatal("a foreign pid must not remove this descriptor")
	}
	removeOwnRunDescriptor(path, os.Getpid())
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("the owning run must remove its descriptor")
	}
}

// TestKillRejectsMalformedRunFiles is the safety gate: nothing that is not a
// well-formed AhdCode descriptor may lead to signalling a process.
func TestKillRejectsMalformedRunFiles(t *testing.T) {
	cases := map[string]string{
		"empty":           "",
		"not json":        "definitely not json",
		"pid zero":        `{"schema":"ahdcode.run","version":1,"pid":0,"source":"/x"}`,
		"negative pid":    `{"schema":"ahdcode.run","version":1,"pid":-5,"source":"/x"}`,
		"huge pid":        `{"schema":"ahdcode.run","version":1,"pid":999999999,"source":"/x"}`,
		"missing source":  `{"schema":"ahdcode.run","version":1,"pid":10}`,
		"foreign schema":  `{"schema":"something-else","version":1,"pid":10,"source":"/x"}`,
		"future version":  `{"schema":"ahdcode.run","version":99,"pid":10,"source":"/x"}`,
		"bare pid number": "12345",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "app.run")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := readRunDescriptor(path); err == nil {
				t.Fatal("expected the malformed descriptor to be rejected")
			}
			var out, errorOutput bytes.Buffer
			if code := runKill([]string{path}, &out, &errorOutput); code == 0 {
				t.Fatalf("expected a non-zero exit; output %q", out.String())
			}
			if !strings.Contains(errorOutput.String(), "no process was stopped") {
				t.Fatalf("expected an explicit no-kill message; got %q", errorOutput.String())
			}
			// The malformed file is left alone rather than acted upon.
			if _, err := os.Stat(path); err != nil {
				t.Fatal("a malformed descriptor must not be silently consumed")
			}
		})
	}
}

func TestKillCleansStaleDescriptor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.run")
	// A pid that cannot be running: writeRunDescriptor accepts it, and the
	// liveness check is what decides.
	stale := runDescriptorMaxPID - 1
	if processAlive(stale) {
		t.Skip("chosen stale pid happens to be alive on this machine")
	}
	if err := writeRunDescriptor(path, runDescriptor{
		Schema: runDescriptorSchema, Version: runDescriptorVersion,
		PID: stale, Source: "/tmp/app.ahd", StartedAt: "2026-01-01T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	var out, errorOutput bytes.Buffer
	if code := runKill([]string{path}, &out, &errorOutput); code != 0 {
		t.Fatalf("stale cleanup should succeed; stderr %q", errorOutput.String())
	}
	if !strings.Contains(out.String(), "not running") {
		t.Fatalf("expected a stale report; got %q", out.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("a stale descriptor must be removed")
	}
}

func TestKillRequiresARunFileArgument(t *testing.T) {
	var out, errorOutput bytes.Buffer
	if code := runKill(nil, &out, &errorOutput); code != 2 {
		t.Fatalf("expected usage exit 2; got %d", code)
	}
	// A bare pid is deliberately not an accepted interface.
	errorOutput.Reset()
	if code := runKill([]string{"12345"}, &out, &errorOutput); code == 0 {
		t.Fatal("a bare pid must not be accepted as a kill target")
	}
}

func TestClaimRunFileBlocksLiveDuplicateAndClearsStale(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.run")
	if err := claimRunFile(path); err != nil {
		t.Fatalf("a missing descriptor must not block a run: %v", err)
	}
	if err := startRunDescriptor(path, "app.ahd", os.Getpid()); err != nil {
		t.Fatal(err)
	}
	err := claimRunFile(path)
	if err == nil {
		t.Fatal("a live descriptor must block a duplicate run")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Fatalf("expected an already-running message; got %v", err)
	}
	stale := runDescriptorMaxPID - 1
	if processAlive(stale) {
		t.Skip("chosen stale pid happens to be alive on this machine")
	}
	if err := writeRunDescriptor(path, runDescriptor{
		Schema: runDescriptorSchema, Version: runDescriptorVersion,
		PID: stale, Source: "/tmp/app.ahd", StartedAt: "2026-01-01T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	if err := claimRunFile(path); err != nil {
		t.Fatalf("a stale descriptor must not block a new run: %v", err)
	}
}
