package ahdruntime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The test binary doubles as a portable child program: re-executed with
// AHD_PROCESS_HELPER=1, TestProcessHelperMain performs one scripted behavior
// and exits, so no test depends on the host's shell or coreutils.
func TestProcessHelperMain(t *testing.T) {
	if os.Getenv("AHD_PROCESS_HELPER") != "1" {
		return
	}
	args := os.Args
	for index, arg := range args {
		if arg == "--" {
			args = args[index+1:]
			break
		}
	}
	mode, rest := args[0], args[1:]
	switch mode {
	case "echo":
		for _, arg := range rest {
			fmt.Println(arg)
		}
		os.Exit(0)
	case "split":
		fmt.Fprint(os.Stdout, "to stdout")
		fmt.Fprint(os.Stderr, "to stderr")
		os.Exit(3)
	case "sleep":
		time.Sleep(60 * time.Second)
		os.Exit(0)
	case "flood":
		line := strings.Repeat("y", 1023) + "\n"
		for {
			if _, err := os.Stdout.WriteString(line); err != nil {
				os.Exit(1)
			}
		}
	case "spawn":
		// Start a grandchild in the same process group, report its pid,
		// then hang until killed.
		child := exec.Command(os.Args[0], "-test.run=^TestProcessHelperMain$", "--", "sleep")
		if err := child.Start(); err != nil {
			os.Exit(9)
		}
		fmt.Println(child.Process.Pid)
		time.Sleep(60 * time.Second)
		os.Exit(0)
	case "invalid-utf8":
		os.Stdout.Write([]byte{'o', 'k', 0xff, 0xfe})
		os.Exit(0)
	case "pwd":
		directory, _ := os.Getwd()
		fmt.Print(directory)
		os.Exit(0)
	}
	os.Exit(2)
}

func processHelper(t *testing.T) (string, []string) {
	t.Helper()
	t.Setenv("AHD_PROCESS_HELPER", "1")
	return os.Args[0], []string{"-test.run=^TestProcessHelperMain$", "--"}
}

func decodeProcess(t *testing.T, data string) ahdProcessResultData {
	t.Helper()
	result, err := ahdProcessDecode(data)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestProcessRunPassesArgumentsLiterallyWithoutAShell(t *testing.T) {
	command, prefix := processHelper(t)
	directory := t.TempDir()
	hostile := []string{
		"; touch SHOULD_NOT_EXIST",
		"$(touch SHOULD_NOT_EXIST)",
		"`touch SHOULD_NOT_EXIST`",
		"a b  c",
		"| cat /etc/passwd",
		"&& echo pwned",
		"*",
		"$HOME",
		"%PATH%",
		"\"quoted\" 'single'",
		"",
	}
	data, err := ProcessRun(command, append(append([]string{}, prefix...), append([]string{"echo"}, hostile...)...), 30, AhdProcessDefaultMaxOutputBytes, directory)
	if err != nil {
		t.Fatal(err)
	}
	result := decodeProcess(t, data)
	if result.ExitCode != 0 {
		t.Fatalf("exit = %d, stderr %q", result.ExitCode, result.Stderr)
	}
	want := strings.Join(hostile, "\n") + "\n"
	if strings.ReplaceAll(result.Stdout, "\r\n", "\n") != want {
		t.Fatalf("arguments were not passed literally:\n got %q\nwant %q", result.Stdout, want)
	}
	if _, err := os.Stat(filepath.Join(directory, "SHOULD_NOT_EXIST")); err == nil {
		t.Fatal("a shell interpreted an argument")
	}

	if runtime.GOOS != "windows" {
		echo, err := exec.LookPath("echo")
		if err != nil {
			t.Skip("no echo executable")
		}
		data, err := ProcessRun(echo, []string{"; touch SHOULD_NOT_EXIST", "$(touch SHOULD_NOT_EXIST)"}, 10, 1024, directory)
		if err != nil {
			t.Fatal(err)
		}
		if got := decodeProcess(t, data).Stdout; got != "; touch SHOULD_NOT_EXIST $(touch SHOULD_NOT_EXIST)\n" {
			t.Fatalf("echo stdout = %q", got)
		}
		if _, err := os.Stat(filepath.Join(directory, "SHOULD_NOT_EXIST")); err == nil {
			t.Fatal("a shell interpreted an argument to echo")
		}
	}
}

func TestProcessRunSeparatesStreamsAndReportsExitCodes(t *testing.T) {
	command, prefix := processHelper(t)
	data, err := ProcessRun(command, append(prefix, "split"), 30, 4096, "")
	if err != nil {
		t.Fatal(err)
	}
	result := decodeProcess(t, data)
	if result.ExitCode != 3 || result.Stdout != "to stdout" || result.Stderr != "to stderr" {
		t.Fatalf("result = %+v", result)
	}
	data, err = ProcessRun(command, append(prefix, "invalid-utf8"), 30, 4096, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := decodeProcess(t, data).Stdout; got != "ok��" && got != "ok�" {
		t.Fatalf("invalid UTF-8 output = %q", got)
	}
	directory := t.TempDir()
	data, err = ProcessRun(command, append(prefix, "pwd"), 30, 4096, directory)
	if err != nil {
		t.Fatal(err)
	}
	real, _ := filepath.EvalSymlinks(directory)
	if got, _ := filepath.EvalSymlinks(decodeProcess(t, data).Stdout); got != real {
		t.Fatalf("working directory = %q; want %q", got, real)
	}
}

func TestProcessRunLaunchFailuresAreErrorsNotExitCodes(t *testing.T) {
	directory := t.TempDir()
	cases := []struct {
		command string
		args    []string
		want    string
	}{
		{filepath.Join(directory, "no-such-program"), nil, "not found"},
		{"ahd-no-such-program-v250", nil, "executable not found"},
		{"", nil, "command is empty"},
		{"echo", []string{"a\x00b"}, "NUL byte"},
	}
	if runtime.GOOS != "windows" {
		plain := filepath.Join(directory, "not-executable")
		if err := os.WriteFile(plain, []byte("#!/bin/sh\necho hi\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		cases = append(cases, struct {
			command string
			args    []string
			want    string
		}{plain, nil, "permission denied"})
	}
	for _, testCase := range cases {
		if _, err := ProcessRun(testCase.command, testCase.args, 5, 1024, ""); err == nil || !strings.Contains(err.Error(), testCase.want) {
			t.Fatalf("%q: error = %v; want %q", testCase.command, err, testCase.want)
		}
	}
	for _, bounds := range [][2]int64{{0, 10}, {AhdProcessHardMaxTimeoutSeconds + 1, 10}, {5, 0}, {5, AhdProcessHardMaxOutputBytes + 1}} {
		if _, err := ProcessRun("echo", nil, bounds[0], bounds[1], ""); err == nil || !strings.Contains(err.Error(), "must be between") {
			t.Fatalf("bounds %v accepted: %v", bounds, err)
		}
	}
	expectRaise(t, AhdClassError, func() { AhdProcessRun(AhdClassError, "", AhdNewList[string](), 5, 10) })
}

func TestProcessRunEnforcesTimeoutAndOutputBounds(t *testing.T) {
	command, prefix := processHelper(t)
	before := runtime.NumGoroutine()

	started := time.Now()
	_, err := ProcessRun(command, append(prefix, "sleep"), 1, 1024, "")
	if err == nil || !strings.Contains(err.Error(), "timed out after 1 seconds") {
		t.Fatalf("timeout: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 10*time.Second {
		t.Fatalf("timeout took %v", elapsed)
	}

	started = time.Now()
	_, err = ProcessRun(command, append(prefix, "flood"), 30, 10000, "")
	if err == nil || !strings.Contains(err.Error(), "output limit exceeded") {
		t.Fatalf("output bound: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 10*time.Second {
		t.Fatalf("output overflow took %v", elapsed)
	}

	// No goroutine outlives a run.
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before {
		t.Fatalf("goroutines grew from %d to %d", before, after)
	}
}

func TestProcessRunTimeoutKillsTheProcessGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-group termination is a Unix guarantee")
	}
	command, prefix := processHelper(t)
	started := time.Now()
	_, err := ProcessRun(command, append(prefix, "spawn"), 2, 4096, "")
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("timeout: %v", err)
	}
	if time.Since(started) > 10*time.Second {
		t.Fatal("a grandchild holding the pipes delayed the timeout")
	}
	// The grandchild's pid is not returned on failure; find it via ps is not
	// portable, so the stronger check is that Wait returned promptly above
	// (a surviving grandchild would keep the pipe open until WaitDelay) and
	// that no helper in "sleep" mode remains in our session.
	survivor := ""
	for attempt := 0; attempt < 30; attempt++ {
		survivor = ""
		output, _ := exec.Command("ps", "-A", "-o", "pid=,stat=,command=").Output()
		for _, line := range strings.Split(string(output), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 3 || strings.HasPrefix(fields[1], "Z") {
				continue // a zombie awaiting its reparented reaper is already dead
			}
			if strings.Contains(line, os.Args[0]) && strings.HasSuffix(strings.TrimSpace(line), "-- sleep") {
				if pid, err := strconv.Atoi(fields[0]); err == nil && pid != os.Getpid() {
					survivor = line
				}
			}
		}
		if survivor == "" {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("grandchild survived the group kill: %s", survivor)
}
