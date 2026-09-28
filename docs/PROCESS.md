# Process standard module

[English] · [Türkçe](PROCESS_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [File](FILESYSTEM.md) · [Errors](ERRORS.md)

`Process` (v2.5.0) runs one external program with an explicit list of
arguments, waits for it, and returns its exit code, standard output, and
standard error — with a finite timeout and a finite output budget, and
**never through a shell**. Import it explicitly:

```ahd
bring Process
from Process bring (ProcessResult, ProcessError)
```

The canonical module identity is `builtin:Process`; a sibling `Process.ahd`
cannot shadow it. (Since v2.5.0 `Process` is a standard module name: a local
`Process.ahd` is no longer what `bring Process` loads — rename such a file.)

## Surface

```text
Process.run(command: String,
            args: List<String> = [],
            timeoutSeconds: Int = 30,
            maxOutputBytes: Int = 1048576) -> ProcessResult

ProcessResult.exitCode() -> Int
ProcessResult.stdout()   -> String
ProcessResult.stderr()   -> String

ProcessError  (derives from Error)
```

`ProcessResult` is an opaque value produced only by `Process.run`. As with
every AhdCode call, arguments are either all positional or all named:

```ahd
result: ProcessResult := Process.run(
    command: "/usr/bin/systemctl",
    args: ["status", "example.service", "--no-pager"],
    timeoutSeconds: 15,
    maxOutputBytes: 1048576
)
write(str(result.exitCode()))
write(result.stdout())
write(result.stderr())
```

## No shell, ever

`Process.run` starts the executable **directly** with each element of `args`
as one separate argument. There is no `/bin/sh -c`, no `cmd.exe /C`, and no
PowerShell anywhere in the module, and there is no function that accepts a
command line as one String. Shell syntax in an argument is ordinary text:

```ahd
result := Process.run("/bin/echo", ["; touch SHOULD_NOT_EXIST", "$(whoami)"])
write(result.stdout())   // ; touch SHOULD_NOT_EXIST $(whoami)
```

Nothing is executed, expanded, globbed, or redirected. `*`, `|`, `&&`, `;`,
`$VAR`, `%VAR%`, backquotes, and quotes all reach the program exactly as
written. An argument with spaces stays one argument. Build argument lists
from data; never concatenate a command line.

On **Windows**, `.bat` and `.cmd` files are refused with `ProcessError`:
Windows runs them through `cmd.exe`, which would re-parse the arguments.
Run the real executable instead.

## Finding the executable

- An absolute path (`/usr/bin/systemctl`) is used as is. **Privileged workers
  should always use absolute paths.**
- A relative path containing a separator (`./bin/tool`, `bin/tool`) is
  resolved against the working directory.
- A bare name (`git`) is looked up in `PATH`. A program in the current
  directory is never found by bare name.

The child's working directory is the program's working directory (in the
REPL, the session directory). Standard input is the null device, so a child
that waits for input sees end-of-file instead of hanging. The environment is
inherited unchanged; `Process` offers no way to read, print, or modify it,
and error messages never include it.

## Exit codes versus errors

A program that runs and exits — with any status — is a **result**, not an
error:

```ahd
check := Process.run("/usr/bin/systemctl", ["is-active", "--quiet", "example.service"])
if check.exitCode() == 0 {
    write("active")
} else {
    write("not active: " + str(check.exitCode()))
}
```

`exitCode()` is the program's exit status. On Unix, a program killed by a
signal it did not handle reports `-1`.

`ProcessError` is raised only when there is no ordinary result to return:

| Condition | Message reason |
|---|---|
| executable missing | `executable not found` |
| not executable / no permission | `permission denied` |
| working directory missing | `executable or working directory not found` |
| empty command, NUL byte in command or argument | `the command is empty`, `argument 2 contains a NUL byte` |
| timeout | `timed out after N seconds (timeoutSeconds); the process was terminated` |
| output budget exceeded | `output limit exceeded: stdout and stderr together produced more than N bytes (maxOutputBytes); the process was terminated` |
| bound out of range | `timeoutSeconds must be between 1 and 3600; received 0` |
| Windows batch file | `batch files run through cmd.exe; Process.run never uses a shell` |

Every message has the form `run "<command>" failed: <reason>`. The argument
list is deliberately not repeated in messages, because arguments may carry
secrets.

## Timeout

Every run has a finite timeout: `timeoutSeconds` defaults to **30** and must
be between **1 and 3600**. When it expires the child is killed, then waited
for (reaped), so no zombie remains, and `ProcessError` is raised.

On **Unix** the child is started in its own process group and the whole
group is killed, so helpers it started are terminated too (a descendant that
deliberately moves to another session or group is outside this guarantee).
On **Windows** only the direct child is killed; its descendants are not
guaranteed to be terminated. If a surviving descendant keeps the output pipes
open, `run` stops reading after a two-second grace period rather than
waiting for it.

## Output budget

`stdout` and `stderr` are captured in memory, so they are bounded together:
`maxOutputBytes` (default **1 MiB**, allowed **1 – 67 108 864**) is the
combined total of both streams. A program that produces more is killed and
`ProcessError` is raised. Output is **never silently truncated** — you either
get all of it or an error.

Output is decoded as UTF-8 text; byte sequences that are not valid UTF-8 are
replaced with `U+FFFD`. `Process` is for programs whose output is text;
redirect binary output to a file with the program's own options.

## Security model: a capability, not a policy

`Process` is a capability. It deliberately does **not** decide which
programs or arguments are safe — there is no global allowlist inside the
language — because only the application knows that. The recommended
architecture for anything privileged, such as a hosting control panel:

```text
Web application (unprivileged)
    -> authenticated, narrow control channel (for example a Unix socket)
        -> dedicated AhdCode worker (privileged)
            -> allowlisted system programs, absolute paths, validated arguments
```

- Do not give request-handling web code a direct path to `Process.run`.
- In the worker, map each request to a fixed executable (by absolute path)
  and build its arguments from validated values:

```ahd
bring Process
bring Regex
from Process bring ProcessResult
from Regex bring Pattern

restartService: Function := (unit: String) -> ProcessResult {
    // Accept only plain unit names such as "ata-panel.service".
    unitName: Local Pattern := Regex.compile(r'^[a-z0-9][a-z0-9-]*\.service$')
    if not unitName.matches(unit) {
        toss ValueError("refused unit name: " + unit)
    }
    // A fixed program, by absolute path; the unit is one literal argument.
    return Process.run(command: "/usr/bin/systemctl", args: ["restart", unit], timeoutSeconds: 60)
}

attempt {
    restartService("x.service; rm -rf /")
} except ValueError as error {
    write(error.message)
}
```

- Validate every argument that comes from a request; an argument that starts
  with `-` can still be an option to the target program even without a shell.
- Keep `timeoutSeconds` and `maxOutputBytes` as small as the task allows.

## Not in this version

A shell or command-line String API, pipelines between processes, standard
input, streaming output, background or detached processes, per-call
environment changes, a working-directory parameter, process listing or
signalling, service-manager (`systemd`) abstractions, SSH, and containers
are not part of `Process`.
