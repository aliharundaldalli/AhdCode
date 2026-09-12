package repl

import (
	"bytes"
	"strings"
	"testing"
)

// The persistent REPL runs Cron through the evaluator, which shares the native
// runtime's parser and occurrence search. Scheduler.run itself is not driven
// here: it blocks the session until a task stops it, exactly as it blocks a
// compiled program, and the scheduler loop is covered by the runtime tests.
func TestCronMatchesNativeInPersistentREPL(t *testing.T) {
	input := `bring Cron
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
write(Cron.next("0 0 30 2 *", after).toString())
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v1.2.0")
	rest := output.String()
	for _, line := range strings.SplitAfter(cronExpectedOutput, "\n") {
		if line == "" {
			continue
		}
		index := strings.Index(rest, line)
		if index < 0 {
			t.Fatalf("REPL output missing %q in order:\n%s\nerrors:\n%s", line, output.String(), errors.String())
		}
		rest = rest[index+len(line):]
	}
	if !strings.Contains(errors.String(), "CronError") ||
		!strings.Contains(errors.String(), `Cron.next: schedule "0 0 30 2 *" can never run: none of its day-of-month values occurs in its months`) {
		t.Fatalf("REPL did not report the uncaught CronError:\n%s", errors.String())
	}
	if strings.Contains(output.String(), "tick") {
		t.Fatal("a task ran although no Scheduler was started")
	}
	for _, forbidden := range []string{"panic:", "goroutine "} {
		if strings.Contains(errors.String(), forbidden) {
			t.Fatalf("REPL leaked Go internals: %s", errors.String())
		}
	}
}

// cronExpectedOutput is shared, byte for byte, with internal/build's native
// expectation for the same program.
const cronExpectedOutput = `* * * * * -> 2026-09-12 10:16:00
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
