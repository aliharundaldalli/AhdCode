package evaluator

import (
	"testing"
	"time"

	"ahdcode/internal/backend/golang/ahdruntime"
)

// The evaluator shares ahdruntime/cron.go with compiled programs; these tests
// pin the evaluator's own boundary: DateTime conversion, error classification,
// and unchanged task failures.

func TestCronNextThroughEvaluatorDateTimes(t *testing.T) {
	session := newSecurityTestSession()
	after := session.dateTime(time.Date(2026, 9, 12, 10, 15, 30, 0, time.UTC))
	cases := []struct {
		expression string
		want       time.Time
	}{
		{"*/15 * * * *", time.Date(2026, 9, 12, 10, 30, 0, 0, time.UTC)},
		{"0 9 * * 1-5", time.Date(2026, 9, 14, 9, 0, 0, 0, time.UTC)},
		{"0 0 29 2 *", time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC)},
	}
	for _, test := range cases {
		got := session.instant(session.cronBuiltin("next", []any{test.expression, after}))
		if !got.Equal(test.want) {
			t.Fatalf("next(%q) = %s, want %s", test.expression, got, test.want)
		}
	}
	istanbul := time.FixedZone("", 3*3600)
	result := session.cronBuiltin("next", []any{"0 0 * * *", session.dateTime(time.Date(2026, 9, 12, 23, 59, 30, 0, istanbul))}).(*Instance)
	if got := session.instant(result); !got.Equal(time.Date(2026, 9, 13, 0, 0, 0, 0, istanbul)) {
		t.Fatalf("next in +03:00 = %s", got)
	}
	if offset := result.Fields["builtin:Time::class::DateTime::field::offsetMinutes"].(int64); offset != 180 {
		t.Fatalf("next kept offset %d, want 180", offset)
	}
}

func TestCronEvaluatorErrorsAreCronErrors(t *testing.T) {
	session := newSecurityTestSession()
	after := session.dateTime(time.Date(2026, 9, 12, 10, 15, 30, 0, time.UTC))
	scheduler := session.cronBuiltin("scheduler", nil)
	task := &FunctionValue{}
	cases := []struct {
		name    string
		fn      func()
		message string
	}{
		{"next invalid", func() { session.cronBuiltin("next", []any{"61 * * * *", after}) },
			`Cron.next: schedule "61 * * * *" minute value 61 is outside 0..59`},
		{"add field count", func() { session.cronOperation("Scheduler.add", scheduler, []any{"* * * *", task}) },
			`Scheduler.add: schedule "* * * *" has 4 fields; expected 5 (minute hour day-of-month month day-of-week)`},
		{"add missing task", func() { session.cronOperation("Scheduler.add", scheduler, []any{"* * * * *", nil}) },
			"Scheduler.add: the task Function is missing"},
		{"run without jobs", func() { session.cronOperation("Scheduler.run", scheduler, nil) },
			"Scheduler.run: the Scheduler has no jobs; add one with Scheduler.add first"},
	}
	for _, test := range cases {
		if got := evaluatorRaisedMessage(t, "CronError", test.fn); got != test.message {
			t.Fatalf("%s message = %q, want %q", test.name, got, test.message)
		}
	}
	// stop on an idle Scheduler, twice, is a harmless no-op.
	if session.cronOperation("Scheduler.stop", scheduler, nil) != Nothing || session.cronOperation("Scheduler.stop", scheduler, nil) != Nothing {
		t.Fatal("Scheduler.stop did not return Nothing")
	}
}

// A task's own AhdCode error must leave Scheduler.run exactly as raised, never
// re-labelled as CronError.
func TestCronRecoverKeepsATaskFailureUnchanged(t *testing.T) {
	session := newSecurityTestSession()
	original := raised{failure: &RuntimeError{Name: "ValueError", Message: "task failed"}}
	defer func() {
		got, ok := recover().(raised)
		if !ok || got.failure == nil || got.failure.Name != "ValueError" || got.failure.Message != "task failed" {
			t.Fatalf("task failure was not re-raised unchanged: %#v", got)
		}
	}()
	func() {
		defer session.cronRecover()
		panic(cronTaskFailure{value: original})
	}()
}

func TestCronRecoverTurnsRuntimeSignalsIntoCronError(t *testing.T) {
	session := newSecurityTestSession()
	got := evaluatorRaisedMessage(t, "CronError", func() {
		defer session.cronRecover()
		ahdruntime.AhdRaiseClass(ahdruntime.AhdClassCronError, "Scheduler.run: the Scheduler is already running")
	})
	if got != "Scheduler.run: the Scheduler is already running" {
		t.Fatalf("message = %q", got)
	}
}
