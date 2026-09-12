package ahdruntime

import (
	"runtime"
	"testing"
	"time"
)

// Expected instants below were computed independently by iterating real UTC
// minutes in Python's zoneinfo, not by this implementation.

func cronUTC(year, month, day, hour, minute, second int) time.Time {
	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
}

// cronRaised runs fn, requires a CronError signal, and returns its message.
func cronRaised(t *testing.T, fn func()) (message string) {
	t.Helper()
	defer func() {
		recovered := recover()
		signal, ok := recovered.(*AhdSignal)
		if !ok || ahdSignalClassName(signal) != "CronError" {
			t.Fatalf("expected a CronError signal, got %#v", recovered)
		}
		message = signal.Message
	}()
	fn()
	return ""
}

func TestCronNextMatchesAnIndependentCalendar(t *testing.T) {
	after := cronUTC(2026, 9, 12, 10, 15, 30) // a Saturday
	cases := []struct {
		expression string
		want       time.Time
	}{
		{"* * * * *", cronUTC(2026, 9, 12, 10, 16, 0)},
		{"*/15 * * * *", cronUTC(2026, 9, 12, 10, 30, 0)},
		{"0 9 * * 1-5", cronUTC(2026, 9, 14, 9, 0, 0)},
		// Both day fields restricted: day 15 OR Monday.
		{"0 0 1,15 * 1", cronUTC(2026, 9, 14, 0, 0, 0)},
		// Day-of-month begins with "*": its days AND Monday.
		{"0 0 */10 * 1", cronUTC(2026, 9, 21, 0, 0, 0)},
		{"0 12 * * 7", cronUTC(2026, 9, 13, 12, 0, 0)},
		{"0 12 * * 0", cronUTC(2026, 9, 13, 12, 0, 0)},
		{"5-7 * * * *", cronUTC(2026, 9, 12, 11, 5, 0)},
		{"0 0 * * 5-7", cronUTC(2026, 9, 13, 0, 0, 0)},
		{"0 0 29 2 *", cronUTC(2028, 2, 29, 0, 0, 0)},
		{" \t0  9\t* *   1-5 ", cronUTC(2026, 9, 14, 9, 0, 0)},
		{"00 09 * * 01-05", cronUTC(2026, 9, 14, 9, 0, 0)},
	}
	for _, test := range cases {
		got := AhdCronNextInstant(AhdClassCronError, test.expression, after)
		if !got.Equal(test.want) {
			t.Fatalf("next(%q) = %s, want %s", test.expression, got, test.want)
		}
	}
	// Strictly after: an instant exactly on a boundary moves to the next one.
	if got := AhdCronNextInstant(AhdClassCronError, "* * * * *", cronUTC(2026, 9, 12, 10, 16, 0)); !got.Equal(cronUTC(2026, 9, 12, 10, 17, 0)) {
		t.Fatalf("next on a boundary = %s", got)
	}
}

func TestCronNextUsesTheFixedOffsetOfAfter(t *testing.T) {
	istanbul := time.FixedZone("", 3*3600)
	got := AhdCronNextInstant(AhdClassCronError, "0 0 * * *", time.Date(2026, 9, 12, 23, 59, 30, 0, istanbul))
	if !got.Equal(time.Date(2026, 9, 13, 0, 0, 0, 0, istanbul)) {
		t.Fatalf("next in +03:00 = %s", got)
	}
	civil := AhdCronNext(AhdClassCronError, "0 9 * * 1-5", AhdCivilTime{Year: 2026, Month: 9, Day: 12, Hour: 10, Minute: 15, Second: 30, OffsetMinutes: 180})
	want := AhdCivilTime{Year: 2026, Month: 9, Day: 14, Hour: 9, Weekday: 1, OffsetMinutes: 180}
	if civil != want {
		t.Fatalf("civil next = %+v, want %+v", civil, want)
	}
}

// Occurrences are real instants in the host zone: a skipped wall-clock reading
// never occurs, and a repeated one occurs twice, an hour apart.
func TestCronNextAcrossDaylightSavingChanges(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("America/New_York zone data is unavailable: ", err)
	}
	spring := ahdCronParse(AhdClassCronError, "test", "30 2 * * *")
	got := spring.next(AhdClassCronError, "test", "30 2 * * *", time.Date(2026, 3, 8, 0, 0, 0, 0, newYork), newYork)
	if !got.Equal(cronUTC(2026, 3, 9, 6, 30, 0)) {
		t.Fatalf("spring-forward next = %s, want 2026-03-09 02:30 EDT", got)
	}
	fall := ahdCronParse(AhdClassCronError, "test", "30 1 * * *")
	first := fall.next(AhdClassCronError, "test", "30 1 * * *", time.Date(2026, 11, 1, 0, 0, 0, 0, newYork), newYork)
	second := fall.next(AhdClassCronError, "test", "30 1 * * *", first, newYork)
	third := fall.next(AhdClassCronError, "test", "30 1 * * *", second, newYork)
	for index, pair := range [][2]time.Time{
		{first, cronUTC(2026, 11, 1, 5, 30, 0)},
		{second, cronUTC(2026, 11, 1, 6, 30, 0)},
		{third, cronUTC(2026, 11, 2, 6, 30, 0)},
	} {
		if !pair[0].Equal(pair[1]) {
			t.Fatalf("fall-back occurrence %d = %s, want %s", index+1, pair[0].UTC(), pair[1])
		}
	}
}

func TestCronParseRejectsInvalidSchedulesExactly(t *testing.T) {
	const fields = "; expected 5 (minute hour day-of-month month day-of-week)"
	cases := []struct {
		expression string
		message    string
	}{
		{"", "Scheduler.add: the schedule is empty"},
		{" \t ", "Scheduler.add: the schedule is empty"},
		{"@daily", `Scheduler.add: schedule "@daily" has 1 field` + fields},
		{"* * * *", `Scheduler.add: schedule "* * * *" has 4 fields` + fields},
		{"* * * * * *", `Scheduler.add: schedule "* * * * * *" has 6 fields` + fields},
		{"* * * *\n*", `Scheduler.add: schedule "* * * *\n*" has 4 fields` + fields},
		{"60 * * * *", `Scheduler.add: schedule "60 * * * *" minute value 60 is outside 0..59`},
		{"* 24 * * *", `Scheduler.add: schedule "* 24 * * *" hour value 24 is outside 0..23`},
		{"* * 0 * *", `Scheduler.add: schedule "* * 0 * *" day-of-month value 0 is outside 1..31`},
		{"* * * 13 *", `Scheduler.add: schedule "* * * 13 *" month value 13 is outside 1..12`},
		{"* * * * 8", `Scheduler.add: schedule "* * * * 8" day-of-week value 8 is outside 0..7`},
		{"99999999999 * * * *", `Scheduler.add: schedule "99999999999 * * * *" minute value 99999999999 is outside 0..59`},
		{"a * * * *", `Scheduler.add: schedule "a * * * *" minute field "a" is not a number, range, step, or list`},
		{"* * * * MON", `Scheduler.add: schedule "* * * * MON" day-of-week field "MON" is not a number, range, step, or list`},
		{"-1 * * * *", `Scheduler.add: schedule "-1 * * * *" minute field "-1" is not a number, range, step, or list`},
		{"1,,2 * * * *", `Scheduler.add: schedule "1,,2 * * * *" minute field "1,,2" is not a number, range, step, or list`},
		{"? * * * *", `Scheduler.add: schedule "? * * * *" minute field "?" is not a number, range, step, or list`},
		{"*/2/3 * * * *", `Scheduler.add: schedule "*/2/3 * * * *" minute field "*/2/3" is not a number, range, step, or list`},
		{"1-2-3 * * * *", `Scheduler.add: schedule "1-2-3 * * * *" minute field "1-2-3" is not a number, range, step, or list`},
		{"*/0 * * * *", `Scheduler.add: schedule "*/0 * * * *" minute step 0 is outside 1..59`},
		{"*/60 * * * *", `Scheduler.add: schedule "*/60 * * * *" minute step 60 is outside 1..59`},
		{"5/10 * * * *", `Scheduler.add: schedule "5/10 * * * *" minute step needs a range such as */5 or 0-30/5`},
		{"5-5/1 * * * *", `Scheduler.add: schedule "5-5/1 * * * *" minute step needs a range such as */5 or 0-30/5`},
		{"30-10 * * * *", `Scheduler.add: schedule "30-10 * * * *" minute range 30-10 runs backwards`},
		{"* * 31 2 *", `Scheduler.add: schedule "* * 31 2 *" can never run: none of its day-of-month values occurs in its months`},
		{"0 0 30 2 *", `Scheduler.add: schedule "0 0 30 2 *" can never run: none of its day-of-month values occurs in its months`},
		{"0 0 31 4,6,9,11 *", `Scheduler.add: schedule "0 0 31 4,6,9,11 *" can never run: none of its day-of-month values occurs in its months`},
	}
	for _, test := range cases {
		got := cronRaised(t, func() { ahdCronParse(AhdClassCronError, "Scheduler.add", test.expression) })
		if got != test.message {
			t.Fatalf("parse(%q)\n got %q\nwant %q", test.expression, got, test.message)
		}
	}
	// A restricted day-of-week makes the day fields an OR, so Feb 31 plus
	// Monday is an ordinary Monday-in-February schedule, not an error.
	ahdCronParse(AhdClassCronError, "Scheduler.add", "0 0 31 2 1")
	if got := cronRaised(t, func() {
		AhdCronNextInstant(AhdClassCronError, "0 0 29 2 *", cronUTC(9996, 3, 1, 0, 0, 0))
	}); got != `Cron.next: schedule "0 0 29 2 *" has no occurrence before the year 10000` {
		t.Fatalf("year bound message = %q", got)
	}
}

// cronTestClock is a deterministic clock: waiting advances it instantly.
type cronTestClock struct {
	current time.Time
	waits   []time.Duration
	// jump is added once, on the first wait, to model a suspended machine.
	jump time.Duration
}

func (clock *cronTestClock) now() time.Time           { return clock.current }
func (clock *cronTestClock) location() *time.Location { return time.UTC }
func (clock *cronTestClock) wait(duration time.Duration, stop <-chan struct{}) bool {
	select {
	case <-stop:
		return false
	default:
	}
	clock.waits = append(clock.waits, duration)
	clock.current = clock.current.Add(duration + clock.jump)
	clock.jump = 0
	return true
}

func newCronTestScheduler() (*ahdCronScheduler, string) {
	handle := AhdCronNewScheduler()
	return ahdCronLookup(AhdClassCronError, "test", handle), handle
}

func cronTimes(t *testing.T, got, want []time.Time) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("runs = %v, want %v", got, want)
	}
	for index := range got {
		if !got[index].Equal(want[index]) {
			t.Fatalf("run %d at %s, want %s (all: %v)", index+1, got[index], want[index], got)
		}
	}
}

func TestCronSchedulerRunsEachOccurrenceExactlyOnce(t *testing.T) {
	scheduler, handle := newCronTestScheduler()
	clock := &cronTestClock{current: cronUTC(2026, 9, 12, 10, 15, 30)}
	var runs []time.Time
	AhdCronSchedulerAdd(AhdClassCronError, handle, "* * * * *", func() {
		runs = append(runs, clock.current)
		clock.current = clock.current.Add(2 * time.Second)
		if len(runs) == 3 {
			AhdCronSchedulerStop(AhdClassCronError, handle)
		}
	})
	scheduler.run(AhdClassCronError, clock, nil)
	cronTimes(t, runs, []time.Time{cronUTC(2026, 9, 12, 10, 16, 0), cronUTC(2026, 9, 12, 10, 17, 0), cronUTC(2026, 9, 12, 10, 18, 0)})
	for _, wait := range clock.waits {
		if wait <= 0 || wait > ahdCronLongestWait {
			t.Fatalf("scheduler waited %s; every wait must be positive and at most %s", wait, ahdCronLongestWait)
		}
	}
}

func TestCronSchedulerRunsIndependentJobsInRegistrationOrder(t *testing.T) {
	scheduler, handle := newCronTestScheduler()
	clock := &cronTestClock{current: cronUTC(2026, 9, 12, 10, 15, 30)}
	var log []string
	AhdCronSchedulerAdd(AhdClassCronError, handle, "*/2 * * * *", func() {
		log = append(log, "A"+clock.current.Format("15:04"))
		if clock.current.Minute() >= 24 {
			AhdCronSchedulerStop(AhdClassCronError, handle)
		}
	})
	AhdCronSchedulerAdd(AhdClassCronError, handle, "*/3 * * * *", func() {
		log = append(log, "B"+clock.current.Format("15:04"))
	})
	scheduler.run(AhdClassCronError, clock, nil)
	want := []string{"A10:16", "A10:18", "B10:18", "A10:20", "B10:21", "A10:22", "A10:24"}
	if len(log) != len(want) {
		t.Fatalf("log = %v, want %v", log, want)
	}
	for index := range want {
		if log[index] != want[index] {
			t.Fatalf("log = %v, want %v", log, want)
		}
	}
}

// An occurrence that passes while a task runs is skipped, never queued.
func TestCronSchedulerSkipsOccurrencesThatPassDuringATask(t *testing.T) {
	scheduler, handle := newCronTestScheduler()
	clock := &cronTestClock{current: cronUTC(2026, 9, 12, 10, 15, 30)}
	var runs []time.Time
	AhdCronSchedulerAdd(AhdClassCronError, handle, "* * * * *", func() {
		runs = append(runs, clock.current)
		if len(runs) == 1 {
			clock.current = clock.current.Add(150 * time.Second)
			return
		}
		AhdCronSchedulerStop(AhdClassCronError, handle)
	})
	scheduler.run(AhdClassCronError, clock, nil)
	cronTimes(t, runs, []time.Time{cronUTC(2026, 9, 12, 10, 16, 0), cronUTC(2026, 9, 12, 10, 19, 0)})
}

// A machine that sleeps through many occurrences runs the late one once when
// it wakes and does not replay the backlog.
func TestCronSchedulerRunsALateOccurrenceOnceAfterSuspension(t *testing.T) {
	scheduler, handle := newCronTestScheduler()
	clock := &cronTestClock{current: cronUTC(2026, 9, 12, 10, 15, 30), jump: 3 * time.Hour}
	var runs []time.Time
	AhdCronSchedulerAdd(AhdClassCronError, handle, "*/5 * * * *", func() {
		runs = append(runs, clock.current)
		if len(runs) == 2 {
			AhdCronSchedulerStop(AhdClassCronError, handle)
		}
	})
	scheduler.run(AhdClassCronError, clock, nil)
	cronTimes(t, runs, []time.Time{cronUTC(2026, 9, 12, 13, 16, 0), cronUTC(2026, 9, 12, 13, 20, 0)})
}

// A wall clock moved backwards never makes an occurrence run again.
func TestCronSchedulerNeverRepeatsAnOccurrenceWhenTheClockMovesBack(t *testing.T) {
	scheduler, handle := newCronTestScheduler()
	clock := &cronTestClock{current: cronUTC(2026, 9, 12, 10, 15, 30)}
	var runs []time.Time
	AhdCronSchedulerAdd(AhdClassCronError, handle, "* * * * *", func() {
		runs = append(runs, clock.current)
		if len(runs) == 1 {
			clock.current = clock.current.Add(-time.Hour)
			return
		}
		AhdCronSchedulerStop(AhdClassCronError, handle)
	})
	scheduler.run(AhdClassCronError, clock, nil)
	cronTimes(t, runs, []time.Time{cronUTC(2026, 9, 12, 10, 16, 0), cronUTC(2026, 9, 12, 10, 17, 0)})
}

func TestCronSchedulerLifecycleErrors(t *testing.T) {
	scheduler, handle := newCronTestScheduler()
	if got := cronRaised(t, func() { AhdCronSchedulerRun(AhdClassCronError, handle, nil) }); got != "Scheduler.run: the Scheduler has no jobs; add one with Scheduler.add first" {
		t.Fatalf("empty run message = %q", got)
	}
	// stop before run and a repeated stop are harmless no-ops.
	AhdCronSchedulerStop(AhdClassCronError, handle)
	AhdCronSchedulerStop(AhdClassCronError, handle)
	// An invalid schedule never registers a job.
	cronRaised(t, func() { AhdCronSchedulerAdd(AhdClassCronError, handle, "61 * * * *", func() {}) })
	if len(scheduler.jobs) != 0 {
		t.Fatalf("an invalid schedule registered a job")
	}

	clock := &cronTestClock{current: cronUTC(2026, 9, 12, 10, 15, 30)}
	var addMessage, runMessage string
	runs := 0
	AhdCronSchedulerAdd(AhdClassCronError, handle, "* * * * *", func() {
		runs++
		addMessage = cronRaised(t, func() { AhdCronSchedulerAdd(AhdClassCronError, handle, "* * * * *", func() {}) })
		runMessage = cronRaised(t, func() { scheduler.run(AhdClassCronError, clock, nil) })
		AhdCronSchedulerStop(AhdClassCronError, handle)
		AhdCronSchedulerStop(AhdClassCronError, handle)
	})
	scheduler.run(AhdClassCronError, clock, nil)
	if runs != 1 {
		t.Fatalf("task ran %d times, want 1", runs)
	}
	if addMessage != "Scheduler.add: a job cannot be added while the Scheduler is running" {
		t.Fatalf("add-while-running message = %q", addMessage)
	}
	if runMessage != "Scheduler.run: the Scheduler is already running" {
		t.Fatalf("nested run message = %q", runMessage)
	}
	// After run returns the Scheduler is idle and reusable.
	if scheduler.running || scheduler.stopping {
		t.Fatal("scheduler state was not reset after run returned")
	}
	AhdCronSchedulerAdd(AhdClassCronError, handle, "0 0 1 1 *", func() {})
}

func TestCronTaskFailurePropagatesUnchangedAndResetsTheScheduler(t *testing.T) {
	scheduler, handle := newCronTestScheduler()
	clock := &cronTestClock{current: cronUTC(2026, 9, 12, 10, 15, 30)}
	calls, later := 0, 0
	AhdCronSchedulerAdd(AhdClassCronError, handle, "* * * * *", func() {
		calls++
		panic("task failed")
	})
	AhdCronSchedulerAdd(AhdClassCronError, handle, "* * * * *", func() { later++ })
	func() {
		defer func() {
			if recovered := recover(); recovered != "task failed" {
				t.Fatalf("task failure did not propagate unchanged: %#v", recovered)
			}
		}()
		scheduler.run(AhdClassCronError, clock, nil)
	}()
	if calls != 1 || later != 0 {
		t.Fatalf("after a failing task: calls=%d later=%d, want 1 and 0", calls, later)
	}
	if scheduler.running {
		t.Fatal("a failed run left the scheduler running")
	}
	AhdCronSchedulerAdd(AhdClassCronError, handle, "0 0 1 1 *", func() {})
}

func TestCronStopWakesAWaitingSchedulerWithoutLeakingGoroutines(t *testing.T) {
	before := runtime.NumGoroutine()
	scheduler, handle := newCronTestScheduler()
	AhdCronSchedulerAdd(AhdClassCronError, handle, "0 0 1 1 *", func() { t.Error("a far-future job must not run") })
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		scheduler.run(AhdClassCronError, ahdCronSystemClock{}, nil)
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		scheduler.mutex.Lock()
		running := scheduler.running
		scheduler.mutex.Unlock()
		if running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("scheduler never started")
		}
		time.Sleep(time.Millisecond)
	}
	AhdCronSchedulerStop(AhdClassCronError, handle)
	select {
	case recovered := <-done:
		if recovered != nil {
			t.Fatalf("run failed after stop: %#v", recovered)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stop did not wake the waiting scheduler")
	}
	for attempt := 0; attempt < 200 && runtime.NumGoroutine() > before; attempt++ {
		time.Sleep(5 * time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before {
		t.Fatalf("goroutines leaked: %d before, %d after", before, after)
	}
}

// cronShiftedClock is real time moved so a minute boundary is moments away,
// which exercises the real timer without waiting up to a minute.
type cronShiftedClock struct{ shift time.Duration }

func (clock cronShiftedClock) now() time.Time           { return time.Now().Add(clock.shift) }
func (clock cronShiftedClock) location() *time.Location { return time.UTC }
func (clock cronShiftedClock) wait(duration time.Duration, stop <-chan struct{}) bool {
	return ahdCronSystemClock{}.wait(duration, stop)
}

func TestCronSchedulerFiresThroughARealTimer(t *testing.T) {
	now := time.Now()
	boundary := now.Truncate(time.Minute).Add(time.Minute)
	clock := cronShiftedClock{shift: boundary.Add(-150 * time.Millisecond).Sub(now)}
	scheduler, handle := newCronTestScheduler()
	runs := 0
	AhdCronSchedulerAdd(AhdClassCronError, handle, "* * * * *", func() {
		runs++
		AhdCronSchedulerStop(AhdClassCronError, handle)
	})
	started := time.Now()
	scheduler.run(AhdClassCronError, clock, nil)
	if elapsed := time.Since(started); runs != 1 || elapsed > 5*time.Second {
		t.Fatalf("real-timer run: runs=%d elapsed=%s", runs, elapsed)
	}
}

// Pending program output is flushed before every wait, so what a task writes
// reaches a terminal or log while the program keeps running.
func TestCronSchedulerFlushesOutputBeforeEveryWait(t *testing.T) {
	scheduler, handle := newCronTestScheduler()
	clock := &cronTestClock{current: cronUTC(2026, 9, 12, 10, 15, 30)}
	flushes, flushedBeforeFirstRun := 0, false
	runs := 0
	AhdCronSchedulerAdd(AhdClassCronError, handle, "* * * * *", func() {
		runs++
		if runs == 1 {
			flushedBeforeFirstRun = flushes > 0
		}
		if runs == 2 {
			AhdCronSchedulerStop(AhdClassCronError, handle)
		}
	})
	scheduler.run(AhdClassCronError, clock, func() { flushes++ })
	if !flushedBeforeFirstRun {
		t.Fatal("output was not flushed before the scheduler first waited")
	}
	if flushes != len(clock.waits) {
		t.Fatalf("flushed %d times for %d waits", flushes, len(clock.waits))
	}
}
