package ahdruntime

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ---- Disk ------------------------------------------------------------------

func requireDiskInfo(t *testing.T, path string) ahdDiskInfoData {
	t.Helper()
	data, err := DiskInspect(path)
	if err != nil {
		t.Fatalf("Disk.inspect(%q): %v", path, err)
	}
	info, err := ahdDiskDecode(data)
	if err != nil {
		t.Fatal(err)
	}
	if info.Total <= 0 || info.Free < 0 || info.Available < 0 || info.Used < 0 {
		t.Fatalf("%q: implausible sizes %+v", path, info)
	}
	if info.Free > info.Total || info.Available > info.Free || info.Used != info.Total-info.Free {
		t.Fatalf("%q: inconsistent sizes %+v", path, info)
	}
	percent, _ := DiskInfoUsedPercent(data)
	if percent < 0 || percent > 100 {
		t.Fatalf("%q: usedPercent %v outside 0..100", path, percent)
	}
	if path != info.Path {
		t.Fatalf("path() = %q; want %q", info.Path, path)
	}
	return info
}

func TestDiskInspectReportsConsistentSizes(t *testing.T) {
	root := string(filepath.Separator)
	if runtime.GOOS == "windows" {
		root = filepath.VolumeName(os.TempDir()) + `\`
	}
	requireDiskInfo(t, root)

	// Any path inside a filesystem works, not only a mount point, and it
	// reports the same filesystem as its parent directory.
	directory := t.TempDir()
	nested := filepath.Join(directory, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(nested, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	parent := requireDiskInfo(t, directory)
	for _, path := range []string{nested, file} {
		if got := requireDiskInfo(t, path); got.Total != parent.Total {
			t.Fatalf("%q reports a different filesystem size (%d vs %d)", path, got.Total, parent.Total)
		}
	}
	// Repeated inspection is stable in capacity and leaves nothing behind.
	for index := 0; index < 200; index++ {
		if got := requireDiskInfo(t, directory); got.Total != parent.Total {
			t.Fatal("capacity changed between inspections")
		}
	}
}

func TestDiskInspectRefusesBadPaths(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	for path, want := range map[string]string{
		"":            "the path is empty",
		"a\x00b":      "NUL byte",
		missing:       "no such file or directory",
		missing + "/": "no such file or directory",
	} {
		if _, err := DiskInspect(path); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("Disk.inspect(%q) = %v; want %q", path, err, want)
		}
	}
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		locked := filepath.Join(t.TempDir(), "locked")
		if err := os.MkdirAll(filepath.Join(locked, "inner"), 0o755); err != nil {
			t.Fatal(err)
		}
		_ = os.Chmod(locked, 0o000)
		defer os.Chmod(locked, 0o755)
		if _, err := DiskInspect(filepath.Join(locked, "inner")); err == nil || !strings.Contains(err.Error(), "permission denied") {
			t.Fatalf("inaccessible path: %v", err)
		}
	}
	if _, err := DiskInspectAs("/", "shown.path"); err != nil {
		t.Fatal(err)
	}
	if data, _ := DiskInspectAs(t.TempDir(), "relative/dir"); !strings.Contains(data, `"relative/dir"`) {
		t.Fatalf("display path not kept: %s", data)
	}
	expectRaise(t, AhdClassError, func() { AhdDiskInspect(AhdClassError, "") })
}

func TestDiskBytesRefusesOverflow(t *testing.T) {
	if _, ok := ahdDiskBytes(1<<62, 4); ok {
		t.Fatal("an overflowing byte count was accepted")
	}
	if value, ok := ahdDiskBytes(1000, 4096); !ok || value != 4096000 {
		t.Fatalf("ahdDiskBytes = %d, %v", value, ok)
	}
	data := `{"path":"x","total":0,"used":0,"free":0,"available":0}`
	if percent, err := DiskInfoUsedPercent(data); err != nil || percent != 0 {
		t.Fatalf("empty filesystem percent = %v, %v", percent, err)
	}
}

// ---- Service -----------------------------------------------------------------

func useServiceFixture(t *testing.T, platform string, show func(unit string) (int64, string, string, error)) {
	t.Helper()
	previousPlatform, previousShow := ahdServicePlatform, ahdServiceShow
	ahdServicePlatform, ahdServiceShow = platform, show
	t.Cleanup(func() { ahdServicePlatform, ahdServiceShow = previousPlatform, previousShow })
}

func fixtureOutput(id, load, active, sub, unitFile string) string {
	return "Id=" + id + "\nLoadState=" + load + "\nActiveState=" + active + "\nSubState=" + sub + "\nUnitFileState=" + unitFile + "\n"
}

func TestServiceStatusNormalizesSystemdStates(t *testing.T) {
	cases := []struct {
		label, output         string
		state, subState, name string
		running, enabled      bool
	}{
		{"active running enabled", fixtureOutput("nginx.service", "loaded", "active", "running", "enabled"), "active", "running", "nginx.service", true, true},
		{"inactive disabled", fixtureOutput("nginx.service", "loaded", "inactive", "dead", "disabled"), "inactive", "dead", "nginx.service", false, false},
		{"failed", fixtureOutput("app.service", "loaded", "failed", "failed", "enabled"), "failed", "failed", "app.service", false, true},
		{"activating", fixtureOutput("app.service", "loaded", "activating", "start", "enabled"), "activating", "start", "app.service", false, true},
		{"deactivating", fixtureOutput("app.service", "loaded", "deactivating", "stop-sigterm", "enabled"), "deactivating", "stop-sigterm", "app.service", false, true},
		{"reloading is active", fixtureOutput("caddy.service", "loaded", "reloading", "reload", "enabled"), "active", "reload", "caddy.service", false, true},
		{"one-shot exited", fixtureOutput("setup.service", "loaded", "active", "exited", "static"), "active", "exited", "setup.service", false, false},
		{"runtime enablement", fixtureOutput("x.service", "loaded", "active", "running", "enabled-runtime"), "active", "running", "x.service", true, true},
		{"masked", fixtureOutput("old.service", "masked", "inactive", "dead", "masked"), "inactive", "dead", "old.service", false, false},
		{"unknown active state", fixtureOutput("x.service", "loaded", "maintenance", "cleaning", "enabled"), "unknown", "cleaning", "x.service", false, true},
		{"odd substate", fixtureOutput("x.service", "loaded", "active", "Läuft 1", "enabled"), "active", "unknown", "x.service", false, true},
		{"crlf output", strings.ReplaceAll(fixtureOutput("x.service", "loaded", "active", "running", "enabled"), "\n", "\r\n"), "active", "running", "x.service", true, true},
	}
	for _, testCase := range cases {
		t.Run(testCase.label, func(t *testing.T) {
			useServiceFixture(t, "linux", func(unit string) (int64, string, string, error) { return 0, testCase.output, "", nil })
			data, err := ServiceStatus("anything.service")
			if err != nil {
				t.Fatal(err)
			}
			name, _ := ServiceInfoName(data)
			state, _ := ServiceInfoActiveState(data)
			sub, _ := ServiceInfoSubState(data)
			running, _ := ServiceInfoRunning(data)
			enabled, _ := ServiceInfoEnabled(data)
			if name != testCase.name || state != testCase.state || sub != testCase.subState || running != testCase.running || enabled != testCase.enabled {
				t.Fatalf("got %s %s/%s running=%v enabled=%v", name, state, sub, running, enabled)
			}
		})
	}
}

func TestServiceStatusFailuresAreServiceErrors(t *testing.T) {
	cases := []struct {
		label    string
		exitCode int64
		stdout   string
		stderr   string
		err      error
		want     string
	}{
		{"not found", 0, fixtureOutput("ghost.service", "not-found", "inactive", "dead", ""), "", nil, "service not found"},
		{"no unit information", 0, "", "", nil, "no unit information"},
		{"no systemd", 1, "", "System has not been booted with systemd as init system (PID 1). Can't operate.", nil, "systemd is not available"},
		{"bus unavailable", 1, "", "Failed to connect to bus: No such file or directory", nil, "systemd is not available"},
		{"access denied", 1, "", "Failed to get properties: Access denied", nil, "permission denied"},
		{"other failure", 4, "", "Unit name weird", nil, "exit code 4"},
		{"no systemctl", 0, "", "", errors.New("systemd is not available on this system (systemctl was not found)"), "systemctl was not found"},
		{"timeout", 0, "", "", errors.New("the inspection timed out after 10 seconds"), "timed out"},
	}
	for _, testCase := range cases {
		t.Run(testCase.label, func(t *testing.T) {
			useServiceFixture(t, "linux", func(unit string) (int64, string, string, error) {
				return testCase.exitCode, testCase.stdout, testCase.stderr, testCase.err
			})
			if _, err := ServiceStatus("ghost.service"); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v; want %q", err, testCase.want)
			}
		})
	}
	for _, platform := range []string{"darwin", "windows", "freebsd"} {
		useServiceFixture(t, platform, func(unit string) (int64, string, string, error) {
			t.Fatal("systemctl must not run on an unsupported platform")
			return 0, "", "", nil
		})
		if _, err := ServiceStatus("nginx.service"); err == nil || !strings.Contains(err.Error(), "supported only on Linux with systemd; this system is "+platform) {
			t.Fatalf("%s: %v", platform, err)
		}
	}
	expectRaise(t, AhdClassError, func() { AhdServiceStatus(AhdClassError, "") })
}

func TestServiceNamesAreDataNeverCommands(t *testing.T) {
	var asked []string
	useServiceFixture(t, "linux", func(unit string) (int64, string, string, error) {
		asked = append(asked, unit)
		return 0, fixtureOutput(unit, "loaded", "active", "running", "enabled"), "", nil
	})
	for _, hostile := range []string{
		"", "nginx.service; touch SHOULD_NOT_EXIST", "$(touch SHOULD_NOT_EXIST)", "`id`", "a b", "a|b", "a&&b",
		"../etc/passwd", "/usr/lib/systemd/system/x.service", "-H evil", "--version", ".hidden", "a\x00b", "a\nb",
		"'quoted'", "\"double\"", "tab\tname", strings.Repeat("a", 256), "ünicode.service",
	} {
		if _, err := ServiceStatus(hostile); err == nil || !strings.Contains(err.Error(), "the service name") {
			t.Fatalf("hostile name %q was accepted: %v", hostile, err)
		}
	}
	if len(asked) != 0 {
		t.Fatalf("systemctl was consulted for rejected names: %v", asked)
	}
	for _, valid := range []string{"nginx.service", "nginx", "getty@tty1.service", "sys-kernel-debug.mount", "dbus-org.freedesktop.login1.service", "a:b_c.timer"} {
		if _, err := ServiceStatus(valid); err != nil {
			t.Fatalf("valid name %q refused: %v", valid, err)
		}
	}
	if strings.Join(asked, ",") != "nginx.service,nginx,getty@tty1.service,sys-kernel-debug.mount,dbus-org.freedesktop.login1.service,a:b_c.timer" {
		t.Fatalf("names were not passed verbatim: %v", asked)
	}
	arguments := ahdServiceArguments("nginx.service")
	if strings.Join(arguments, " ") != "show --no-pager --property=Id,LoadState,ActiveState,SubState,UnitFileState -- nginx.service" {
		t.Fatalf("arguments = %v", arguments)
	}
}

// TestServiceStatusAgainstRealSystemd is a read-only smoke that runs only on
// a Linux host with systemd; it never changes any unit.
func TestServiceStatusAgainstRealSystemd(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("systemd inspection is Linux-only")
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		t.Skip("no systemctl on this host")
	}
	data, err := ServiceStatus("systemd-journald.service")
	if err != nil {
		t.Skipf("host systemd not usable here: %v", err)
	}
	if state, _ := ServiceInfoActiveState(data); state == "" {
		t.Fatalf("empty state: %s", data)
	}
	if _, err := ServiceStatus("ahdcode-definitely-missing-v270.service"); err == nil || !strings.Contains(err.Error(), "service not found") {
		t.Fatalf("missing unit: %v", err)
	}
}
