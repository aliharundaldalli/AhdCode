# Cron standard module

[English] · Türkçe

[Back to README](README.md) · [Modules](MODULES.md) · [Time](TIME.md) · [Web](WEB.md)

`Cron` is the compiler-registered `builtin:Cron` module, introduced in AhdCode
v1.2.0. It runs AhdCode Functions on classic five-field schedules while the
program that scheduled them is running. A sibling `Cron.ahd` cannot shadow it.

```ahd
bring Cron
from Cron bring (Scheduler, CronError)
```

## What Cron is, and what it is not

Cron is application-level scheduling inside one AhdCode process.

```text
AhdCode Cron  !=  the operating system's crontab
AhdCode Cron  !=  launchd
AhdCode Cron  !=  a systemd timer
AhdCode Cron  !=  Windows Task Scheduler
AhdCode Cron  !=  a persistent job queue or worker cluster
```

- Jobs run only while their program is running. When the process stops — it
  finishes, fails, is interrupted, or the machine restarts — its jobs stop too.
  Nothing is stored, and nothing resumes after a restart.
- Cron never installs, edits, or reads a system scheduler configuration, never
  starts a daemon or service, and opens no network endpoint.
- Work that must happen while no AhdCode program is running belongs to the
  operating system's own scheduler, which can start an ordinary AhdCode program.

## Surface

```text
Cron.scheduler()                                        -> Scheduler
Cron.next(expression: String, after: DateTime)          -> DateTime

Scheduler.add(expression: String, task: () -> Nothing)  -> Nothing
Scheduler.run()                                         -> Nothing
Scheduler.stop()                                        -> Nothing

CronError
```

`DateTime` is the [Time](TIME.md) module's Class. A `Scheduler` comes only from
`Cron.scheduler()`; constructing one directly is a compile-time error.

## Schedule syntax

A schedule has exactly five fields, separated by spaces or tabs:

```text
minute  hour  day-of-month  month  day-of-week
0-59    0-23  1-31          1-12   0-7 (0 and 7 are Sunday)
```

Each field is a comma-separated list of items:

| Item | Meaning | Example |
|---|---|---|
| `*` | every value | `*` |
| `n` | one value | `30` |
| `a-b` | an inclusive range with `a` ≤ `b` | `9-17` |
| `*/s` | every `s`-th value of the field | `*/15` |
| `a-b/s` | every `s`-th value within the range | `0-30/10` |
| `x,y` | a list of the items above | `0,30` |

Numbers are plain decimal digits. A step is at least `1` and no wider than the
range it steps over. Leading and trailing spaces or tabs are ignored.

```text
* * * * *        every minute
*/15 * * * *     every 15 minutes
0 9 * * 1-5      09:00 on weekdays
30 2 1 * *       02:30 on the first day of each month
0 0,12 * * *     midnight and noon
0 0 29 2 *       midnight on 29 February
```

**Day of month and day of week.** When both day fields are restricted — neither
begins with `*` — a day matches if **either** field matches: `0 0 1,15 * 1`
runs on the 1st, on the 15th, and on every Monday. When either day field begins
with `*`, a day must match **both**: `0 0 */10 * 1` runs on a 1st, 11th, 21st,
or 31st that is also a Monday. This is the traditional cron rule.

**Not supported.** Month or weekday names (`JAN`, `MON`), aliases such as
`@daily` or `@reboot`, a seconds field, `?`, `L`, `W`, `#`, and time zones
inside the expression. Each is rejected; none is reinterpreted.

**Validated before anything is scheduled.** `Scheduler.add` and `Cron.next`
check the whole schedule first. A malformed schedule raises `CronError` and is
never registered, so it cannot silently become another schedule, run at once,
or never run. A schedule whose day-of-month values occur in none of its months,
such as `0 0 30 2 *`, is rejected because it can never run.

## Cron.next

`Cron.next(expression, after)` returns the first matching minute strictly after
`after`. It is the calculation the scheduler itself uses, which makes it a
convenient way to validate a schedule or show when a job runs next.

```ahd
bring Cron
bring Time

after := Time.dateTimeUTC(2026, 9, 12, 10, 15, 30)
write(Cron.next("*/15 * * * *", after).toString())
write(Cron.next("0 9 * * 1-5", after).toString())
```

=>

```text
2026-09-12 10:30:00
2026-09-14 09:00:00
```

`next` evaluates the schedule in the fixed UTC offset that `after` carries, and
the result has the same offset: `Time.utc()` gives UTC, `Time.dateTimeOffset`
gives a fixed offset, and `Time.now()` gives the host's offset at that moment.
A `DateTime` has no daylight-saving rules of its own.

## Time zone and daylight saving

`Scheduler.run` evaluates schedules in the **host's local time zone** — the
clock `Time.now()` reads — including that zone's daylight-saving rules. There
is no per-schedule time zone. To schedule in UTC, run the program where the
local time zone is UTC.

Every occurrence is a real instant:

- A local time that a daylight-saving change **skips** does not occur. Where the
  clock jumps from 02:00 to 03:00, `30 2 * * *` does not run that night.
- A local time that a change **repeats** occurs twice, an hour apart. Where the
  clock falls back from 02:00 to 01:00, `30 1 * * *` runs at both 01:30s.

A fixed-offset zone without daylight saving, such as Turkey's, never meets
either case.

## Tasks

A task is a Function that takes nothing and returns `Nothing`, declared by name
and passed to `add`. Its shape is checked at compile time: a Function with a
parameter or a result is the ordinary `SEM004` type mismatch.

```ahd
bring Cron
from Cron bring Scheduler

jobs: Scheduler := Cron.scheduler()
tick: Function := () -> Nothing {
    write("tick")
}
jobs.add("* * * * *", tick)
```

A task reaches program state through an explicit `Global` dependency, like any
other Function. That is also how a task stops its own Scheduler:

```ahd
bring Cron
from Cron bring Scheduler

jobs: Scheduler := Cron.scheduler()
count: Int := 0

once: Function := () -> Nothing {
    jobs: Global Scheduler
    count: Global Int
    count += 1
    jobs.stop()
}
jobs.add("* * * * *", once)
```

## Lifecycle

**add** validates the schedule and appends the job. Several jobs can share one
Scheduler, each with its own schedule. A job cannot be added while its
Scheduler runs.

**run** blocks the program and runs the jobs:

- Jobs run on the program's own flow, one at a time. Cron starts no background
  thread and runs nothing in parallel.
- When several jobs are due in the same minute, each runs once, in the order it
  was added.
- One due occurrence means one run. When a task finishes, its next occurrence is
  computed from that moment, so an occurrence that passes while a task is still
  running is skipped, not queued.
- If the computer sleeps or the program is paused past several occurrences,
  each affected job runs once when the program resumes and then continues with
  its next future occurrence; missed occurrences are not replayed. A wall clock
  that moves backwards never makes an occurrence run a second time.
- Between occurrences the program waits on an ordinary timer and does not poll.
  Pending `write` output is flushed before each wait, so what a task writes
  reaches the terminal or a redirected log while the program keeps running.
- `run` raises `CronError` if the Scheduler has no jobs or is already running.

**stop** asks a running Scheduler to return. It takes effect when the current
task returns; jobs still due in that minute do not run. Calling `stop` on a
Scheduler that is not running, or calling it again, does nothing. After `run`
returns, the Scheduler can run again.

**A task that fails.** An error a task raises propagates out of `run` unchanged,
with its own Class and message, and ends the run; jobs still due in that minute
do not run. Handle it like any other error:

```ahd
bring Cron
from Cron bring Scheduler

jobs: Scheduler := Cron.scheduler()
report: Function := () -> Nothing {
    toss(ValueError("report failed"))
}
jobs.add("* * * * *", report)

attempt {
    jobs.run()
} except ValueError as error {
    write("job failed: " + error.message)
}
```

**Shutdown.** Cron installs no signal handler. Interrupting the program —
Ctrl+C, closing its terminal, `ahdcode kill`, or the operating system stopping
the process — ends it at once, like any AhdCode program; a task in progress
stops with it. Once the process is gone, nothing of the Scheduler remains.

## A bounded example

```ahd
bring Cron
bring Time
bring File
from Cron bring Scheduler

jobs: Scheduler := Cron.scheduler()
beats: Int := 0

heartbeat: Function := () -> Nothing {
    jobs: Global Scheduler
    beats: Global Int
    beats += 1
    line: Local String := "beat " + str(beats) + " at " + Time.now().toString()
    File.writeText("heartbeat.txt", line + "\n")
    write(line)
    if beats == 2 {
        jobs.stop()
    }
}

jobs.add("* * * * *", heartbeat)
jobs.run()
write("stopped after " + str(beats) + " beats")
```

It writes a heartbeat at the next two minute boundaries and then ends by itself.
The annotated version is `examples/v0.1/60_cron.ahd`.

## Cron in a Web application

`Scheduler.run` and a Web server both occupy the program that calls them, so in
v1.2.0 one AhdCode program does not serve requests and run a Scheduler at the
same time. Run scheduled work as a second AhdCode program beside the Web
application, sharing its files or database. See
[Web: scheduled application work](WEB.md#cron-scheduled-application-work).

## Errors

`CronError` derives from `Error`. Its messages are stable:

```text
Scheduler.add: the schedule is empty
Scheduler.add: schedule "* * * *" has 4 fields; expected 5 (minute hour day-of-month month day-of-week)
Scheduler.add: schedule "60 * * * *" minute value 60 is outside 0..59
Scheduler.add: schedule "a * * * *" minute field "a" is not a number, range, step, or list
Scheduler.add: schedule "*/0 * * * *" minute step 0 is outside 1..59
Scheduler.add: schedule "5/10 * * * *" minute step needs a range such as */5 or 0-30/5
Scheduler.add: schedule "30-10 * * * *" minute range 30-10 runs backwards
Scheduler.add: schedule "0 0 30 2 *" can never run: none of its day-of-month values occurs in its months
Scheduler.add: a job cannot be added while the Scheduler is running
Scheduler.run: the Scheduler has no jobs; add one with Scheduler.add first
Scheduler.run: the Scheduler is already running
Cron.next: schedule "0 0 29 2 *" has no occurrence before the year 10000
```

`Cron.next` reports schedule problems with the prefix `Cron.next:`. A schedule is
text, so an invalid schedule compiles and raises `CronError` when it is used; a
wrong argument type or task shape is a compile-time diagnostic.

## Execution modes

`ahdcode run`, `ahdcode build`, and a relocated compiled executable share one
Cron implementation: the same parser, occurrence search, and scheduler loop. The
persistent REPL runs that implementation too, but `Scheduler.run` blocks the
REPL session until a task stops it, and Ctrl+C ends the REPL itself. Explore
schedules with `Cron.next` interactively, and run long-lived Schedulers as
programs.

## Not in this module

No persistent jobs, retries, overlapping runs, worker pools, distributed
locking, job history, per-job time zones, seconds, cron aliases, month or
weekday names, HTTP triggers, shell-command jobs, or system scheduler
integration.
