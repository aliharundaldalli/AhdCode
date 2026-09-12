package ahdruntime

// AhdCode Cron standard module runtime.
//
// This file is compiled twice: once as part of the compiler, where the
// interactive evaluator calls it directly, and once as generated program
// source. Both execution modes run the same parser, the same occurrence
// search, and the same scheduler loop, and it depends only on the Go standard
// library.
//
// Cron is deliberately bounded. A schedule is a classic five-field expression
// of numbers, ranges, steps, and lists. A Scheduler runs AhdCode Functions in
// the calling program: Scheduler.run blocks on the calling goroutine, starts
// no background goroutine, waits with an ordinary timer, and returns when a
// task calls Scheduler.stop or when a task raises. There is no daemon, no
// persistent queue, no system crontab, and no network endpoint.

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

// The interactive evaluator raises CronError through this runtime, so the
// class needs a constructor in the compiler process too. A generated program
// raises through its own generated descriptor instead.
func init() {
	AhdRegisterError(AhdClassCronError, func(message string) AhdInstance {
		instance := &ahdModuleError{message: message}
		instance.AhdSetClass(AhdClassCronError)
		return instance
	})
}

// ahdCronField is one of the five schedule positions and its accepted values.
type ahdCronField struct {
	name      string
	low, high int
}

var ahdCronFields = [5]ahdCronField{
	{"minute", 0, 59},
	{"hour", 0, 23},
	{"day-of-month", 1, 31},
	{"month", 1, 12},
	// 0 and 7 both mean Sunday.
	{"day-of-week", 0, 7},
}

// ahdCronMonthDays is the longest each month can be, so a day-of-month that
// never occurs in any selected month is rejected instead of never running.
var ahdCronMonthDays = [13]int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// ahdCronSchedule is one validated expression.
type ahdCronSchedule struct {
	minutes  [60]bool
	hours    [24]bool
	days     [32]bool
	months   [13]bool
	weekdays [7]bool
	// A day-of-month or day-of-week field that begins with "*" is
	// unrestricted. When both fields are restricted a day matches either one;
	// otherwise it must match both. This is the traditional cron rule.
	daysStar, weekdaysStar bool
}

// ahdCronParse validates expression completely before anything is scheduled.
func ahdCronParse(errorClass *AhdClass, operation, expression string) *ahdCronSchedule {
	quoted := strconv.Quote(expression)
	fail := func(detail string) {
		AhdRaiseClass(errorClass, operation+": schedule "+quoted+" "+detail)
	}
	// Fields are separated by spaces or tabs only. Any other character,
	// including a newline, is part of a field and is rejected there.
	fields := strings.FieldsFunc(expression, func(r rune) bool { return r == ' ' || r == '\t' })
	if len(fields) == 0 {
		AhdRaiseClass(errorClass, operation+": the schedule is empty")
	}
	if len(fields) != 5 {
		count := strconv.Itoa(len(fields)) + " fields"
		if len(fields) == 1 {
			count = "1 field"
		}
		fail("has " + count + "; expected 5 (minute hour day-of-month month day-of-week)")
	}
	schedule := &ahdCronSchedule{
		daysStar:     strings.HasPrefix(fields[2], "*"),
		weekdaysStar: strings.HasPrefix(fields[4], "*"),
	}
	sets := [5][]bool{schedule.minutes[:], schedule.hours[:], schedule.days[:], schedule.months[:], schedule.weekdays[:]}
	for index, text := range fields {
		ahdCronParseField(fail, ahdCronFields[index], text, sets[index])
	}
	if !schedule.daysStar && schedule.weekdaysStar {
		possible := false
		for month := 1; month <= 12 && !possible; month++ {
			for day := 1; day <= ahdCronMonthDays[month] && schedule.months[month]; day++ {
				if schedule.days[day] {
					possible = true
					break
				}
			}
		}
		if !possible {
			fail("can never run: none of its day-of-month values occurs in its months")
		}
	}
	return schedule
}

func ahdCronParseField(fail func(string), field ahdCronField, text string, set []bool) {
	bounds := strconv.Itoa(field.low) + ".." + strconv.Itoa(field.high)
	invalid := func() {
		fail(field.name + " field " + strconv.Quote(text) + " is not a number, range, step, or list")
	}
	// number accepts plain decimal digits only: no sign, no name, no space. A
	// value too long to mean anything is reported as out of range instead of
	// overflowing.
	number := func(digits string) (int, bool) {
		if digits == "" {
			invalid()
		}
		for _, r := range digits {
			if r < '0' || r > '9' {
				invalid()
			}
		}
		if len(digits) > 9 {
			return 0, false
		}
		parsed, _ := strconv.Atoi(digits)
		return parsed, true
	}
	value := func(digits string) int {
		parsed, ok := number(digits)
		if !ok || parsed < field.low || parsed > field.high {
			fail(field.name + " value " + digits + " is outside " + bounds)
		}
		return parsed
	}
	for _, item := range strings.Split(text, ",") {
		rangeText, stepText, stepped := strings.Cut(item, "/")
		low, high := field.low, field.high
		if rangeText == "*" {
			if field.name == "day-of-week" {
				high = 6
			}
		} else {
			lowText, highText, isRange := strings.Cut(rangeText, "-")
			low = value(lowText)
			high = low
			if isRange {
				high = value(highText)
				if low > high {
					fail(field.name + " range " + rangeText + " runs backwards")
				}
			}
			if stepped && !isRange {
				fail(field.name + " step needs a range such as */5 or 0-30/5")
			}
		}
		step := 1
		if stepped {
			width := high - low
			if width == 0 {
				fail(field.name + " step needs a range such as */5 or 0-30/5")
			}
			parsed, ok := number(stepText)
			if !ok || parsed < 1 || parsed > width {
				fail(field.name + " step " + stepText + " is outside 1.." + strconv.Itoa(width))
			}
			step = parsed
		}
		for current := low; current <= high; current += step {
			if field.name == "day-of-week" {
				set[current%7] = true
			} else {
				set[current] = true
			}
		}
	}
}

func (schedule *ahdCronSchedule) dayMatches(day int, weekday time.Weekday) bool {
	if schedule.daysStar || schedule.weekdaysStar {
		return schedule.days[day] && schedule.weekdays[weekday]
	}
	return schedule.days[day] || schedule.weekdays[weekday]
}

// next returns the first minute boundary strictly after `after` whose civil
// reading in location matches every field. Occurrences are real instants: a
// wall-clock reading that a daylight-saving change skips does not occur, and
// one that a change repeats occurs twice.
func (schedule *ahdCronSchedule) next(errorClass *AhdClass, operation, expression string, after time.Time, location *time.Location) time.Time {
	local := after.In(location)
	candidate := local.Add(-time.Duration(local.Second())*time.Second - time.Duration(local.Nanosecond())).Add(time.Minute)
	// The bound is far above any search a validated schedule needs; it exists
	// so that no input can ever turn into an endless loop.
	for guard := 0; guard < 4_000_000; guard++ {
		year, month, day := candidate.Date()
		if year > 9999 {
			break
		}
		if !schedule.months[month] {
			candidate = ahdCronAdvance(candidate, time.Date(year, month+1, 1, 0, 0, 0, 0, location))
			continue
		}
		if !schedule.dayMatches(day, candidate.Weekday()) {
			candidate = ahdCronAdvance(candidate, time.Date(year, month, day+1, 0, 0, 0, 0, location))
			continue
		}
		if !schedule.hours[candidate.Hour()] {
			candidate = candidate.Add(time.Duration(60-candidate.Minute()) * time.Minute)
			continue
		}
		if !schedule.minutes[candidate.Minute()] {
			candidate = candidate.Add(time.Minute)
			continue
		}
		return candidate
	}
	AhdRaiseClass(errorClass, operation+": schedule "+strconv.Quote(expression)+" has no occurrence before the year 10000")
	return time.Time{}
}

// ahdCronAdvance moves the search forward, never backward, even when a
// daylight-saving change makes the calendar target ambiguous.
func ahdCronAdvance(current, target time.Time) time.Time {
	if target.After(current) {
		return target
	}
	return current.Add(time.Minute)
}

// AhdCronNextInstant returns the next occurrence of expression strictly after
// `after`, evaluated in the fixed UTC offset `after` carries.
func AhdCronNextInstant(errorClass *AhdClass, expression string, after time.Time) time.Time {
	schedule := ahdCronParse(errorClass, "Cron.next", expression)
	return schedule.next(errorClass, "Cron.next", expression, after, after.Location())
}

// AhdCronNext is the native entry point of Cron.next.
func AhdCronNext(errorClass *AhdClass, expression string, after AhdCivilTime) AhdCivilTime {
	return ahdCivilFrom(AhdCronNextInstant(errorClass, expression, AhdTimeInstantCivil(after)))
}

type ahdCronJob struct {
	expression string
	schedule   *ahdCronSchedule
	task       func()
	next       time.Time
}

type ahdCronScheduler struct {
	mutex    sync.Mutex
	jobs     []*ahdCronJob
	running  bool
	stopping bool
	stopped  chan struct{}
}

// ahdCronClock is the scheduler's only view of time, so the loop can be tested
// deterministically. The system clock uses host local time, like Time.now().
type ahdCronClock interface {
	now() time.Time
	location() *time.Location
	// wait blocks for duration, or until stop is closed; it reports whether
	// the whole duration elapsed.
	wait(duration time.Duration, stop <-chan struct{}) bool
}

type ahdCronSystemClock struct{}

func (ahdCronSystemClock) now() time.Time           { return time.Now() }
func (ahdCronSystemClock) location() *time.Location { return time.Local }
func (ahdCronSystemClock) wait(duration time.Duration, stop <-chan struct{}) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-stop:
		return false
	}
}

// ahdCronLongestWait bounds one sleep, so a wall-clock adjustment or a
// suspended machine is noticed within this long. It is not polling: a
// scheduler waiting for a daily job wakes at most twice a minute.
const ahdCronLongestWait = 30 * time.Second

var ahdCronRegistry = struct {
	sync.Mutex
	schedulers map[string]*ahdCronScheduler
	count      int64
}{schedulers: map[string]*ahdCronScheduler{}}

// AhdCronNewScheduler creates an empty scheduler and returns its handle.
func AhdCronNewScheduler() string {
	ahdCronRegistry.Lock()
	defer ahdCronRegistry.Unlock()
	ahdCronRegistry.count++
	handle := "cron-scheduler-" + strconv.FormatInt(ahdCronRegistry.count, 10)
	ahdCronRegistry.schedulers[handle] = &ahdCronScheduler{}
	return handle
}

func ahdCronLookup(errorClass *AhdClass, operation, handle string) *ahdCronScheduler {
	ahdCronRegistry.Lock()
	scheduler := ahdCronRegistry.schedulers[handle]
	ahdCronRegistry.Unlock()
	if scheduler == nil {
		AhdRaiseClass(errorClass, operation+": the Scheduler is not available")
	}
	return scheduler
}

// AhdCronSchedulerAdd validates expression and registers task. A job cannot be
// added while the scheduler runs, so the set of jobs a run uses is fixed.
func AhdCronSchedulerAdd(errorClass *AhdClass, handle, expression string, task func()) {
	scheduler := ahdCronLookup(errorClass, "Scheduler.add", handle)
	schedule := ahdCronParse(errorClass, "Scheduler.add", expression)
	if task == nil {
		AhdRaiseClass(errorClass, "Scheduler.add: the task Function is missing")
	}
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()
	if scheduler.running {
		AhdRaiseClass(errorClass, "Scheduler.add: a job cannot be added while the Scheduler is running")
	}
	scheduler.jobs = append(scheduler.jobs, &ahdCronJob{expression: expression, schedule: schedule, task: task})
}

// AhdCronSchedulerRun runs the scheduler with the system clock. flush writes
// pending program output; it runs before every wait, the same rule take
// follows before it blocks, so what a task writes reaches a terminal or log
// file while the program keeps running.
func AhdCronSchedulerRun(errorClass *AhdClass, handle string, flush func()) {
	ahdCronLookup(errorClass, "Scheduler.run", handle).run(errorClass, ahdCronSystemClock{}, flush)
}

// AhdCronSchedulerStop asks a running scheduler to return. It is a no-op when
// the scheduler is not running or is already stopping.
func AhdCronSchedulerStop(errorClass *AhdClass, handle string) {
	ahdCronLookup(errorClass, "Scheduler.stop", handle).stop()
}

func (scheduler *ahdCronScheduler) stop() {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()
	if scheduler.running && !scheduler.stopping {
		scheduler.stopping = true
		close(scheduler.stopped)
	}
}

func (scheduler *ahdCronScheduler) stopRequested() bool {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()
	return scheduler.stopping
}

// run is the whole scheduler: it waits for the earliest due job, runs every
// due job once in registration order, and repeats. A task runs on this
// goroutine, so an error it raises propagates out of run unchanged and ends
// the run. After a job runs, its next occurrence is computed from the time
// the task finished, so an occurrence that passed while a task was running is
// skipped rather than queued, and no occurrence ever runs twice.
func (scheduler *ahdCronScheduler) run(errorClass *AhdClass, clock ahdCronClock, flush func()) {
	scheduler.mutex.Lock()
	if scheduler.running {
		scheduler.mutex.Unlock()
		AhdRaiseClass(errorClass, "Scheduler.run: the Scheduler is already running")
	}
	if len(scheduler.jobs) == 0 {
		scheduler.mutex.Unlock()
		AhdRaiseClass(errorClass, "Scheduler.run: the Scheduler has no jobs; add one with Scheduler.add first")
	}
	stop := make(chan struct{})
	scheduler.running, scheduler.stopping, scheduler.stopped = true, false, stop
	jobs := append([]*ahdCronJob(nil), scheduler.jobs...)
	scheduler.mutex.Unlock()
	defer func() {
		scheduler.mutex.Lock()
		scheduler.running, scheduler.stopping, scheduler.stopped = false, false, nil
		scheduler.mutex.Unlock()
	}()

	location := clock.location()
	started := clock.now()
	for _, job := range jobs {
		job.next = job.schedule.next(errorClass, "Scheduler.run", job.expression, started, location)
	}
	latest := started
	for {
		due := jobs[0].next
		for _, job := range jobs[1:] {
			if job.next.Before(due) {
				due = job.next
			}
		}
		for {
			now := clock.now()
			if !now.Before(due) {
				break
			}
			wait := due.Sub(now)
			if wait > ahdCronLongestWait {
				wait = ahdCronLongestWait
			}
			if flush != nil {
				flush()
			}
			if !clock.wait(wait, stop) {
				return
			}
		}
		if now := clock.now(); now.After(latest) {
			latest = now
		}
		for _, job := range jobs {
			if job.next.After(latest) {
				continue
			}
			if scheduler.stopRequested() {
				return
			}
			job.task()
			if scheduler.stopRequested() {
				return
			}
			// A wall clock moved backwards must not make an occurrence due
			// again, so the search starts from the latest time seen.
			if now := clock.now(); now.After(latest) {
				latest = now
			}
			job.next = job.schedule.next(errorClass, "Scheduler.run", job.expression, latest, location)
		}
	}
}
