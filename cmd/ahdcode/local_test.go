package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/localdev"
)

// readOnlyHostsFile stages a hosts file the current user cannot write, so the
// permission-denied branch is exercised without ever going near the real one.
func readOnlyHostsFile(t *testing.T, content string) string {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("running as root; there is no unwritable path to exercise")
	}
	directory := t.TempDir()
	path := filepath.Join(directory, "hosts")
	if err := os.WriteFile(path, []byte(content), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(directory, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(directory, 0o700) })
	return path
}

// A. A non-interactive session never prompts and never elevates. It says what
// it would have done and stops, because a script blocking on a password
// nobody will type is worse than one that fails immediately.
func TestHostsApplyNeverPromptsWithoutATerminal(t *testing.T) {
	current := "127.0.0.1 localhost\n"
	updated := localdev.ApplyHostsBlock(current, localdev.RenderHostsBlock([]string{"ahdakademi.test"}))
	path := readOnlyHostsFile(t, current)

	var out, errOut bytes.Buffer
	// A pipe, not a terminal: isInteractive is false for it.
	code := writeSystemHosts(path, "apply", current, updated, strings.NewReader("y\ny\ny\n"), &out, &errOut)
	if code == 0 {
		t.Fatal("a non-interactive apply reported success without changing anything")
	}
	printed := out.String()
	if !strings.Contains(printed, "not an interactive terminal") {
		t.Errorf("the refusal did not explain itself:\n%s", printed)
	}
	if !strings.Contains(printed, "Nothing was changed") {
		t.Errorf("the refusal did not say the file was left alone:\n%s", printed)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != current {
		t.Fatalf("the hosts file was modified without consent:\n%s", content)
	}
}

// B. When the file is writable, the change is applied directly and no
// elevation is involved at all -- and only AhdCode's own block moves.
func TestHostsApplyWritesOnlyTheManagedBlock(t *testing.T) {
	current := "##\n# Host Database\n##\n127.0.0.1\tlocalhost\n10.0.0.5 build.internal\n"
	path := filepath.Join(t.TempDir(), "hosts")
	if err := os.WriteFile(path, []byte(current), 0o644); err != nil {
		t.Fatal(err)
	}
	updated := localdev.ApplyHostsBlock(current, localdev.RenderHostsBlock([]string{"ahdakademi.test", "ahddatabasestudio.test"}))

	var out, errOut bytes.Buffer
	if code := writeSystemHosts(path, "apply", current, updated, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("apply failed (%d): %s%s", code, out.String(), errOut.String())
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(content), current) {
		t.Fatalf("existing entries were not preserved verbatim:\n%s", content)
	}
	if !strings.Contains(string(content), "127.0.0.1 ahdakademi.test") {
		t.Fatalf("the managed block was not written:\n%s", content)
	}
	if !strings.Contains(out.String(), "+ 127.0.0.1 ahdakademi.test") {
		t.Errorf("the change was applied without showing what it was:\n%s", out.String())
	}

	// Removing it restores the file exactly.
	removed := localdev.ApplyHostsBlock(string(content), "")
	out.Reset()
	if code := writeSystemHosts(path, "remove", string(content), removed, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("remove failed (%d)", code)
	}
	final, _ := os.ReadFile(path)
	if string(final) != current {
		t.Fatalf("removal did not restore the original file:\n%q", final)
	}
}

// C. A no-op change touches nothing and says so.
func TestHostsApplyIsANoOpWhenAlreadyCorrect(t *testing.T) {
	current := "127.0.0.1 localhost\n"
	var out, errOut bytes.Buffer
	code := writeSystemHosts("/definitely/not/written", "apply", current, current, strings.NewReader(""), &out, &errOut)
	if code != 0 {
		t.Fatalf("a no-op apply returned %d", code)
	}
	if !strings.Contains(out.String(), "nothing was changed") {
		t.Errorf("a no-op apply did not say so:\n%s", out.String())
	}
}

// D. The union rule: a name AhdCode already manages stays in the block after
// its session stops, so the same project does not need administrator access
// again on its next run.
func TestManagedHostnamesAreUnioned(t *testing.T) {
	got := mergedHostnames([]string{"old.test", "ahddatabasestudio.test"}, []string{"new.test", "ahddatabasestudio.test", "not-a-name"})
	want := []string{"ahddatabasestudio.test", "new.test", "old.test"}
	if len(got) != len(want) {
		t.Fatalf("merged names were %#v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("merged names were %#v, expected %#v", got, want)
		}
	}
}

// E. Status is diagnostic: it reports what is there and changes nothing.
func TestLocalStatusReportsRoutesWithoutMutating(t *testing.T) {
	home := t.TempDir()
	t.Setenv(localdev.HomeEnvKey, home)
	if _, err := localdev.Allocate("ahdakademi.test", localdev.Route{
		Kind: localdev.KindDev, Source: "/projects/ahd/app.ahd",
		Descriptor: "/projects/ahd/app.dev", BindHost: "127.0.0.1", BindPort: 18437,
	}, func(localdev.Route) bool { return true }); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(home, "routes.json"))
	if err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if code := runLocalStatus(&out, &errOut); code != 0 {
		t.Fatalf("status returned %d: %s", code, errOut.String())
	}
	printed := out.String()
	// The descriptor names a session that does not exist, so the route is
	// reported as stale rather than as running -- liveness is never assumed.
	for _, expected := range []string{"AhdCode Local", "Router:", "Stale routes", "ahdakademi.test", "127.0.0.1:18437"} {
		if !strings.Contains(printed, expected) {
			t.Errorf("status did not report %q:\n%s", expected, printed)
		}
	}
	after, err := os.ReadFile(filepath.Join(home, "routes.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("a diagnostic command modified the route registry")
	}
}

// F. A route whose descriptor is gone is never live, whatever pid it recorded.
func TestLocalRouteIsNotLiveWithoutADescriptor(t *testing.T) {
	if localRouteIsLive(localdev.Route{Kind: localdev.KindDev, Descriptor: filepath.Join(t.TempDir(), "app.dev"), OwnerPID: os.Getpid()}) {
		t.Fatal("a route with no descriptor was reported live")
	}
	if localRouteIsLive(localdev.Route{Kind: localdev.KindDev, OwnerPID: os.Getpid()}) {
		t.Fatal("a route with an empty descriptor was reported live")
	}
}

// G. The dev descriptor carries the logical identity as well as the bind
// address, and the two stay distinct fields rather than one merged "URL".
func TestDevDescriptorCarriesTheLogicalIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.dev")
	local := devLocalRoute{
		route:      localdev.Route{Hostname: "ahdakademi.test", BindHost: "127.0.0.1", BindPort: 18437},
		routerPort: 7357,
	}
	firstToken, err := newRunControlToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := startDevDescriptor(path, "/projects/ahd/app.ahd", "/projects/ahd", os.Getpid(), 0, 51234,
		firstToken, local); err != nil {
		t.Fatal(err)
	}
	descriptor, err := readDevDescriptor(path)
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.LocalHostname != "ahdakademi.test" {
		t.Errorf("localHostname was %q", descriptor.LocalHostname)
	}
	if descriptor.LocalURL != "http://ahdakademi.test:7357/" {
		t.Errorf("localURL was %q", descriptor.LocalURL)
	}

	// A session with no routed identity simply omits both, and stays a
	// perfectly valid descriptor.
	plain := filepath.Join(t.TempDir(), "app.dev")
	secondToken, err := newRunControlToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := startDevDescriptor(plain, "/projects/plain/app.ahd", "/projects/plain", os.Getpid(), 0, 51235,
		secondToken, devLocalRoute{}); err != nil {
		t.Fatal(err)
	}
	descriptor, err = readDevDescriptor(plain)
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.LocalHostname != "" || descriptor.LocalURL != "" {
		t.Errorf("a session with no identity recorded one: %#v", descriptor)
	}
}
