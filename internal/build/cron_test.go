package build

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// cronDeterministicProgram needs no waiting: it exercises Cron.next and every
// validation and lifecycle failure a Scheduler can report before it runs.
const cronDeterministicProgram = `bring Cron
bring Time
from Cron bring (Scheduler, CronError)

after := Time.dateTimeUTC(2026, 9, 12, 10, 15, 30)
for schedule in ["* * * * *", "*/15 * * * *", "0 9 * * 1-5", "0 0 1,15 * 1", "0 0 */10 * 1", "0 12 * * 7", "0 0 29 2 *"] {
    write(schedule + " -> " + Cron.next(schedule, after).toString())
}
istanbul := Time.dateTimeOffset(2026, 9, 12, 180, 23, 59, 30)
write(Cron.next("0 0 * * *", istanbul).toString() + " " + str(Cron.next("0 0 * * *", istanbul).offsetMinutes))
jobs: Scheduler := Cron.scheduler()
tick: Function := () -> Nothing {
    write("tick")
}
for bad in ["", "* * * *", "60 * * * *", "*/0 * * * *", "30-10 * * * *", "* * 31 2 *", "@daily"] {
    attempt {
        jobs.add(bad, tick)
    } except CronError as error {
        write(error.message)
    }
}
attempt {
    jobs.run()
} except CronError as error {
    write(error.message)
}
jobs.stop()
jobs.stop()
write("done")
`

// cronNativeExpected is byte-identical to internal/repl's evaluator
// expectation for the same program.
const cronNativeExpected = `* * * * * -> 2026-09-12 10:16:00
*/15 * * * * -> 2026-09-12 10:30:00
0 9 * * 1-5 -> 2026-09-14 09:00:00
0 0 1,15 * 1 -> 2026-09-14 00:00:00
0 0 */10 * 1 -> 2026-09-21 00:00:00
0 12 * * 7 -> 2026-09-13 12:00:00
0 0 29 2 * -> 2028-02-29 00:00:00
2026-09-13 00:00:00 180
Scheduler.add: the schedule is empty
Scheduler.add: schedule "* * * *" has 4 fields; expected 5 (minute hour day-of-month month day-of-week)
Scheduler.add: schedule "60 * * * *" minute value 60 is outside 0..59
Scheduler.add: schedule "*/0 * * * *" minute step 0 is outside 1..59
Scheduler.add: schedule "30-10 * * * *" minute range 30-10 runs backwards
Scheduler.add: schedule "* * 31 2 *" can never run: none of its day-of-month values occurs in its months
Scheduler.add: schedule "@daily" has 1 field; expected 5 (minute hour day-of-month month day-of-week)
Scheduler.run: the Scheduler has no jobs; add one with Scheduler.add first
done
`

func TestCronRunsThroughNativeBackend(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": cronDeterministicProgram})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("Cron program failed: code=%d stderr=%q", code, stderr)
	}
	if stdout != cronNativeExpected {
		t.Fatalf("Cron native output mismatch:\n got: %q\nwant: %q", stdout, cronNativeExpected)
	}
}

func TestCronUncaughtErrorIsAhdCodeLevel(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": "bring Cron\nbring Time\nwrite(Cron.next(\"0 0 30 2 *\", Time.utc()).toString())\n"})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code == 0 {
		t.Fatalf("expected a failing exit code, got 0 (stdout=%q)", stdout)
	}
	assertNoGoInternals(t, stderr)
	if !containsAll(stderr, "CronError", `Cron.next: schedule "0 0 30 2 *" can never run: none of its day-of-month values occurs in its months`) {
		t.Fatalf("uncaught CronError was not reported at AhdCode level: %q", stderr)
	}
}

// A sibling Cron.ahd cannot hijack the builtin import.
func TestCronBuiltinWinsOverSiblingFile(t *testing.T) {
	directory := writeSources(t, map[string]string{
		"main.ahd": "bring Cron\nbring Time\nwrite(Cron.next(\"0 0 1 1 *\", Time.dateTimeUTC(2026, 1, 1)).toString())\n",
		"Cron.ahd": "Public next: Function := (expression: String) -> String {\n    return \"shadowed\"\n}\n",
	})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stdout != "2027-01-01 00:00:00\n" {
		t.Fatalf("builtin Cron did not take precedence: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

// These run compiled programs until their Scheduler fires at a real minute
// boundary, so each waits at most one minute; the two run in parallel.
func TestCronSchedulerFiresInACompiledProgram(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for a real minute boundary")
	}
	cases := []struct {
		name, program, want string
	}{
		{"stop", `bring Cron
from Cron bring Scheduler

jobs: Scheduler := Cron.scheduler()
beat: Function := () -> Nothing {
    jobs: Global Scheduler
    write("beat")
    jobs.stop()
    jobs.stop()
}
jobs.add("* * * * *", beat)
write("waiting")
jobs.run()
write("stopped")
`, "waiting\nbeat\nstopped\n"},
		{"task error", `bring Cron
from Cron bring Scheduler

jobs: Scheduler := Cron.scheduler()
first: Function := () -> Nothing {
    write("first")
}
second: Function := () -> Nothing {
    toss(ValueError("report failed"))
}
third: Function := () -> Nothing {
    write("third must not run")
}
jobs.add("* * * * *", first)
jobs.add("* * * * *", second)
jobs.add("* * * * *", third)
attempt {
    jobs.run()
} except ValueError as error {
    write("caught " + error.message)
}
jobs.add("0 0 1 1 *", first)
write("reusable")
`, "first\ncaught report failed\nreusable\n"},
	}
	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			directory := writeSources(t, map[string]string{"main.ahd": test.program})
			started := time.Now()
			stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
			if code != 0 || stderr != "" || stdout != test.want {
				t.Fatalf("code=%d stdout=%q stderr=%q, want stdout %q", code, stdout, stderr, test.want)
			}
			if elapsed := time.Since(started); elapsed > 3*time.Minute {
				t.Fatalf("scheduler run took %s", elapsed)
			}
		})
	}
}

// Interrupting a compiled program that waits in Scheduler.run ends it promptly
// and without a Go stack trace. Cron installs no signal handler of its own.
func TestCronCompiledProgramStopsOnInterrupt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.Interrupt cannot be delivered to a child process on Windows")
	}
	directory := writeSources(t, map[string]string{"main.ahd": `bring Cron
from Cron bring Scheduler

jobs: Scheduler := Cron.scheduler()
never: Function := () -> Nothing {
    write("far-future job ran")
}
jobs.add("0 0 1 1 *", never)
write("waiting")
jobs.run()
`})
	path, result := BuildProgram(filepath.Join(directory, "main.ahd"), filepath.Join(t.TempDir(), "program"))
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	command := exec.Command(path)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	// Scheduler.run flushes pending output before it waits, so the line
	// arrives through the pipe although the program never exits by itself.
	lines := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadString('\n')
		lines <- line
	}()
	select {
	case line := <-lines:
		if line != "waiting\n" {
			_ = command.Process.Kill()
			t.Fatalf("program did not reach Scheduler.run: line=%q", line)
		}
	case <-time.After(30 * time.Second):
		_ = command.Process.Kill()
		t.Fatal("output written before Scheduler.run was not flushed while it waited")
	}
	if err := command.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("an interrupted scheduler exited successfully")
		}
	case <-time.After(10 * time.Second):
		_ = command.Process.Kill()
		t.Fatal("an interrupted scheduler did not exit")
	}
	assertNoGoInternals(t, stderr.String())
}
