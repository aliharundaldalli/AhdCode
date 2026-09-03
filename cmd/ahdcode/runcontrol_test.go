package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// startTestSupervisor brings up a control server owning one disposable child,
// exactly as `ahdcode run` does, and writes its descriptor.
func startTestSupervisor(t *testing.T) (*runControlServer, *exec.Cmd, string) {
	t.Helper()
	server, err := startRunControlServer()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command("/bin/sleep", "600")
	if err := child.Start(); err != nil {
		server.close()
		t.Skipf("cannot start a child process here: %v", err)
	}
	server.attach(child.Process)
	path := filepath.Join(t.TempDir(), "app.run")
	server.ownDescriptor(path)
	if err := startRunDescriptor(path, "app.ahd", child.Process.Pid, server.port(), server.token); err != nil {
		server.close()
		_ = child.Process.Kill()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		server.close()
		_ = child.Process.Kill()
		_, _ = child.Process.Wait()
	})
	return server, child, path
}

func processStillRunning(t *testing.T, pid int) bool {
	t.Helper()
	// ps is only a test observation; the CLI itself never inspects pids.
	// A terminated child this test has not reaped is still listed by ps in
	// the "Z" (defunct) state; that is a stopped process, not a live one.
	output, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "stat=").CombinedOutput()
	if err != nil {
		return false
	}
	state := strings.TrimSpace(string(output))
	return state != "" && !strings.HasPrefix(state, "Z")
}

func TestRunControlAuthenticatedStopEndsOwnedChild(t *testing.T) {
	_, child, path := startTestSupervisor(t)
	pid := child.Process.Pid

	var out, errorOutput bytes.Buffer
	if code := runKill([]string{path}, &out, &errorOutput); code != 0 {
		t.Fatalf("authenticated kill failed: %s", errorOutput.String())
	}
	waitGone(t, pid)
	if processStillRunning(t, pid) {
		t.Fatal("the owned child survived an authenticated stop")
	}
}

func TestRunControlForceStopUsesTheSameAuthenticatedPath(t *testing.T) {
	_, child, path := startTestSupervisor(t)
	pid := child.Process.Pid

	var out, errorOutput bytes.Buffer
	if code := runKill([]string{"--force", path}, &out, &errorOutput); code != 0 {
		t.Fatalf("forced kill failed: %s", errorOutput.String())
	}
	waitGone(t, pid)
	if processStillRunning(t, pid) {
		t.Fatal("the owned child survived a forced stop")
	}
}

// TestForgedRunFileCannotKillAnUnrelatedProcess is the release-blocking
// regression: a structurally valid descriptor naming a live unrelated process,
// with no supervisor behind its control endpoint, must stop nothing.
func TestForgedRunFileCannotKillAnUnrelatedProcess(t *testing.T) {
	victim := exec.Command("/bin/sleep", "600")
	if err := victim.Start(); err != nil {
		t.Skipf("cannot start a victim process here: %v", err)
	}
	pid := victim.Process.Pid
	defer func() {
		_ = victim.Process.Kill()
		_, _ = victim.Process.Wait()
	}()

	token, err := newRunControlToken()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "forged.run")
	// Every field is well-formed; only the control capability is absent,
	// because port 1 has no AhdCode supervisor behind it.
	if err := writeRunDescriptor(path, runDescriptor{
		Schema: runDescriptorSchema, Version: runDescriptorVersion,
		PID: pid, Source: "/tmp/app.ahd", StartedAt: "2026-01-01T00:00:00Z",
		ControlPort: 1, ControlToken: token,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := readRunDescriptor(path); err != nil {
		t.Fatalf("the forged descriptor should parse; the defence is the control channel, not the shape: %v", err)
	}

	var out, errorOutput bytes.Buffer
	runKill([]string{path}, &out, &errorOutput)

	time.Sleep(300 * time.Millisecond)
	if !processStillRunning(t, pid) {
		t.Fatal("a forged run file terminated an unrelated process")
	}
	if !strings.Contains(out.String()+errorOutput.String(), "not running") {
		t.Fatalf("expected a not-running report; got %q / %q", out.String(), errorOutput.String())
	}
}

// TestWrongTokenCannotStopALiveApplication covers a descriptor that names a
// real supervisor but presents the wrong capability.
func TestWrongTokenCannotStopALiveApplication(t *testing.T) {
	server, child, _ := startTestSupervisor(t)
	pid := child.Process.Pid

	other, err := newRunControlToken()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "wrong.run")
	if err := writeRunDescriptor(path, runDescriptor{
		Schema: runDescriptorSchema, Version: runDescriptorVersion,
		PID: pid, Source: "/tmp/app.ahd", StartedAt: "2026-01-01T00:00:00Z",
		ControlPort: server.port(), ControlToken: other,
	}); err != nil {
		t.Fatal(err)
	}
	var out, errorOutput bytes.Buffer
	if code := runKill([]string{path}, &out, &errorOutput); code == 0 {
		t.Fatal("a wrong token must not report success")
	}
	if !strings.Contains(errorOutput.String(), "no process was stopped") {
		t.Fatalf("expected an explicit no-kill message; got %q", errorOutput.String())
	}
	time.Sleep(200 * time.Millisecond)
	if !processStillRunning(t, pid) {
		t.Fatal("a wrong token stopped the application")
	}
}

func TestKillRejectsUnusableControlMetadata(t *testing.T) {
	victim := exec.Command("/bin/sleep", "600")
	if err := victim.Start(); err != nil {
		t.Skipf("cannot start a victim process here: %v", err)
	}
	pid := victim.Process.Pid
	defer func() {
		_ = victim.Process.Kill()
		_, _ = victim.Process.Wait()
	}()

	valid, err := newRunControlToken()
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]runDescriptor{
		"no control port":   {Schema: runDescriptorSchema, Version: runDescriptorVersion, PID: pid, Source: "/x", ControlToken: valid},
		"port out of range": {Schema: runDescriptorSchema, Version: runDescriptorVersion, PID: pid, Source: "/x", ControlPort: 70000, ControlToken: valid},
		"negative port":     {Schema: runDescriptorSchema, Version: runDescriptorVersion, PID: pid, Source: "/x", ControlPort: -1, ControlToken: valid},
		"missing token":     {Schema: runDescriptorSchema, Version: runDescriptorVersion, PID: pid, Source: "/x", ControlPort: 5000},
		"short token":       {Schema: runDescriptorSchema, Version: runDescriptorVersion, PID: pid, Source: "/x", ControlPort: 5000, ControlToken: "abc"},
		"bad token encoding": {Schema: runDescriptorSchema, Version: runDescriptorVersion, PID: pid, Source: "/x", ControlPort: 5000,
			ControlToken: strings.Repeat("!", 43)},
		"v1 descriptor": {Schema: runDescriptorSchema, Version: 1, PID: pid, Source: "/x"},
	}
	for name, descriptor := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "app.run")
			if err := writeRunDescriptor(path, descriptor); err != nil {
				t.Fatal(err)
			}
			if _, err := readRunDescriptor(path); err == nil {
				t.Fatal("expected the descriptor to be rejected")
			}
			var out, errorOutput bytes.Buffer
			if code := runKill([]string{path}, &out, &errorOutput); code == 0 {
				t.Fatalf("expected a non-zero exit; out %q", out.String())
			}
			if !strings.Contains(errorOutput.String(), "no process was stopped") {
				t.Fatalf("expected an explicit no-kill message; got %q", errorOutput.String())
			}
			if !processStillRunning(t, pid) {
				t.Fatal("an unusable descriptor terminated a process")
			}
		})
	}
}

func TestClaimRunFileUsesControlIdentityNotPID(t *testing.T) {
	server, child, path := startTestSupervisor(t)

	if err := claimRunFile(path); err == nil {
		t.Fatal("a live supervisor must block a duplicate run")
	} else if !strings.Contains(err.Error(), "already running") {
		t.Fatalf("unexpected message: %v", err)
	}

	// The child stays alive, but the supervisor goes away: the descriptor is
	// now stale even though its pid still exists, and must not block a run.
	server.close()
	if err := claimRunFile(path); err != nil {
		t.Fatalf("a descriptor with no live supervisor must not block a run: %v", err)
	}
	if !processStillRunning(t, child.Process.Pid) {
		t.Fatal("claimRunFile must never signal the recorded pid")
	}
}

func TestStaleDescriptorIsCleanedWithoutSignalling(t *testing.T) {
	server, child, path := startTestSupervisor(t)
	pid := child.Process.Pid
	// Supervisor gone, child still alive: exactly the pid-reuse shape.
	server.close()

	var out, errorOutput bytes.Buffer
	if code := runKill([]string{path}, &out, &errorOutput); code != 0 {
		t.Fatalf("stale cleanup should succeed; stderr %q", errorOutput.String())
	}
	if !strings.Contains(out.String(), "no process was signalled") {
		t.Fatalf("expected an explicit no-signal report; got %q", out.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("a stale descriptor must be removed")
	}
	time.Sleep(200 * time.Millisecond)
	if !processStillRunning(t, pid) {
		t.Fatal("a stale descriptor terminated the process its pid named")
	}
}

// TestRunControlRejectsForeignProtocol proves an unrelated local service is
// never mistaken for a supervisor, and that the supervisor ignores traffic
// that does not carry its own identity.
func TestRunControlRejectsForeignProtocol(t *testing.T) {
	server, child, _ := startTestSupervisor(t)
	if err := requestRunControl(server.port(), server.token, "bogus-action"); err == nil {
		t.Fatal("an unsupported action must be refused")
	}
	if !processStillRunning(t, child.Process.Pid) {
		t.Fatal("an unsupported action stopped the child")
	}
	if err := requestRunControl(server.port(), server.token, runControlPing); err != nil {
		t.Fatalf("ping with the real token should succeed: %v", err)
	}
}

func TestRunDescriptorTokenIsNotWrittenToOutput(t *testing.T) {
	server, _, path := startTestSupervisor(t)
	var out, errorOutput bytes.Buffer
	runKill([]string{path}, &out, &errorOutput)
	combined := out.String() + errorOutput.String()
	if strings.Contains(combined, server.token) {
		t.Fatal("the control token must never appear in CLI output")
	}
}

func waitGone(t *testing.T, pid int) {
	t.Helper()
	for attempt := 0; attempt < 50; attempt++ {
		if !processStillRunning(t, pid) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
