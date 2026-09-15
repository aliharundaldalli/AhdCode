# Terminal standard module

[English] · Türkçe

[Back to README](README.md) · [Modules](MODULES.md) · [Fundamentals](FUNDAMENTALS.md) · [Env](ENV.md)

`Terminal` is the compiler-registered `builtin:Terminal` module added in
AhdCode v1.5.0. It adds the terminal-specific behaviour the Fundamentals
`write`, `take`, and `str` deliberately do not expose: joining Strings onto
standard output, writing to standard error, flushing, asking whether standard
output is an interactive terminal and how large it is, styling text, and
laying out collections for people to read. It is written with the Go standard
library only and behaves the same under `ahdcode run`, in a native build, and in
the REPL.

## Public surface

```text
bring Terminal
from Terminal bring TerminalError

Terminal.emit(parts: List<String>, separator: String := " ", ending: String := "\n") -> Nothing
Terminal.error(text: String, ending: String := "\n")                               -> Nothing
Terminal.flush()                                                                   -> Nothing
Terminal.isInteractive()                                                           -> Bool
Terminal.width()                                                                   -> Int?
Terminal.height()                                                                  -> Int?
Terminal.supportsColor()                                                           -> Bool
Terminal.style(
    text: String
    foreground: String := "default"
    background: String := "default"
    bold: Bool := false
    underline: Bool := false
)                                                                                  -> String
Terminal.pretty(value)                                                             -> Nothing

TerminalError  (derived from Error)
```

`pretty` is a type-directed operation: `value` is any value `write` accepts.
Like `Lists.chunk`, it has no Function value of its own; call it directly.

## Relationship to write, take, and str

| | What it does |
| --- | --- |
| `str(value)` | the canonical, deterministic text of a value |
| `write(value)` | writes that text and a line break to standard output |
| `take()` / `take(prompt)` | reads one line from standard input |
| `Terminal` | terminal-specific behaviour the three above intentionally leave out |

`write`, `take`, and `str` are unchanged in v1.5.0. `Terminal` adds no second
way to do what they already do: `emit` takes Strings you have already produced,
and `pretty` reuses exactly the text `write` and `str` produce.

## Calling Terminal functions

An AhdCode call is either entirely positional or entirely named (language
specification §15.3). Both of these are valid:

```ahd
bring Terminal

Terminal.emit(["Ali", "95", "Passed"], " | ")
Terminal.emit(parts: ["Ali", "95", "Passed"], separator: " | ")
```

Writing a positional `parts` List together with a named `separator` in the same
call mixes the two forms and is a compile error. An omitted parameter takes its
default in either form.

## emit

`Terminal.emit(parts, separator, ending)` writes the parts in order, the
separator only between two parts, and the ending exactly once.

```ahd
bring Terminal

Terminal.emit(["Ali", "95", "Passed"])
Terminal.emit(["Ali", "95", "Passed"], " | ")
Terminal.emit(parts: ["Loading"], ending: "")
Terminal.emit(parts: ["..."], separator: "", ending: "\n")
```

writes exactly:

```text
Ali 95 Passed
Ali | 95 | Passed
Loading...
```

- An empty List writes only the ending.
- The separator and the ending may each be empty.
- Text, including Unicode, line breaks inside a part, and the separator itself,
  is written unchanged.
- `emit` takes `List<String>` and converts nothing. Produce the text yourself:
  `Terminal.emit([name, str(score)])` or
  `Terminal.emit(["{name}: {score}"])`. A `List<String?>` is rejected at compile
  time, and there is no variadic form.

`emit` writes to the same standard output as `write`, so the two keep their
order.

## error

`Terminal.error(text, ending)` writes the text and the ending to **standard
error**. Nothing is added: no prefix, no timestamp, no severity, no color.

```ahd
bring Terminal

Terminal.error("Configuration not found")
```

writes `Configuration not found` and a line break to standard error, and
nothing to standard output. With `ahdcode run app.ahd 2> errors.txt` the line
goes to `errors.txt`.

Before it writes, `Terminal.error` flushes pending standard output, so when both
streams reach the same terminal or file the text appears in program order.

## flush

`Terminal.flush()` writes any standard output AhdCode is still holding.

A compiled program — and `ahdcode run`, which compiles one — buffers standard
output. The buffer is written when the program ends, before `take` reads a
line, before `Terminal.error` writes, and before an uncaught error is
reported. A program that keeps running, such as a server, a Cron scheduler, or a
loop that waits, should call `Terminal.flush()` after output that must appear
now. Standard error is not buffered.

In the REPL, output is written as it is produced, so `flush` normally has
nothing to do. Calling `flush` repeatedly is harmless. It never reopens or
replaces a stream and never changes the terminal's mode. If standard output
cannot be written, `flush` raises `TerminalError`.

## isInteractive

`Terminal.isInteractive()` is `true` when **standard output** is attached to an
interactive terminal, and `false` when it is redirected to a file, a pipe, or
the null device. Standard input is not consulted.

```ahd
bring Terminal

if Terminal.isInteractive() {
    Terminal.emit(["Type your name and press Enter."])
}
```

`ahdcode run` gives the program the terminal's own standard output, so the
answer is the same as for the compiled program run directly. In the REPL it
describes the REPL's output. A missing terminal is never an error.

## width and height

`Terminal.width()` and `Terminal.height()` are the columns and rows of the
terminal on standard output, read at the moment of the call.

```ahd
bring Terminal

width: Int? := Terminal.width()
if width != null {
    Terminal.emit(["The terminal is {width} columns wide."])
}
```

Each is a positive `Int`, or `null` when standard output is not a terminal or
the terminal does not report a size; a reported size of zero is `null` too.
There is no fallback: AhdCode never invents 80 columns or 24 rows, never treats
`COLUMNS` or `LINES` as the answer, and never runs `tput`, `stty`, or
PowerShell. On Windows the size is that of the console window, not of its
scroll-back buffer.

## supportsColor

`Terminal.supportsColor()` answers whether styled text should be used on
standard output. The v1.5.0 policy is exactly:

1. standard output is an interactive terminal, and
2. the terminal interprets escape sequences: always on macOS and Linux; on
   Windows, only when the console already has virtual terminal processing
   enabled, which AhdCode reads and never changes, and
3. `NO_COLOR` is unset or empty, following the NO_COLOR convention, and
4. `TERM` is not `dumb`.

No other environment variable is consulted: `FORCE_COLOR`, `CLICOLOR`, and
`COLORTERM` have no effect. The policy is evaluated at every call; there is no
stored color setting and no function to turn color on or off.

## style

`Terminal.style(...)` returns a String; it does not write anything.

```ahd
bring Terminal

success := Terminal.style(text: "Success", foreground: "green", bold: true)
warning := Terminal.style(
    text: "Warning"
    foreground: "yellow"
    underline: true
)
Terminal.emit([success, warning], " / ")
```

The color names are a closed set, written exactly in lower case:

| Name | Foreground | Background |
| --- | --- | --- |
| `default` | no color code | no color code |
| `black` | 30 | 40 |
| `red` | 31 | 41 |
| `green` | 32 | 42 |
| `yellow` | 33 | 43 |
| `blue` | 34 | 44 |
| `magenta` | 35 | 45 |
| `cyan` | 36 | 46 |
| `white` | 37 | 47 |

- Any other name, including `"Red"`, `"bright-red"`, a number, or `""`, raises
  `TerminalError`. The name is checked even when color is unavailable, so a
  misspelling fails the same way on every machine. An unknown name is never
  treated as `default`.
- When `supportsColor()` is `true` and at least one attribute is requested, the
  result is `ESC[` + the codes + `m`, then the text, then the reset `ESC[0m`.
  The codes appear in the order bold (`1`), underline (`4`), foreground,
  background: `Terminal.style("OK", "green", "white", true, true)` is
  `ESC[1;4;32;47mOK ESC[0m` (shown here with a space for readability; there is
  none).
- When `supportsColor()` is `false` — redirected output, `NO_COLOR`,
  `TERM=dumb` — the result is the text itself, so `command > output.txt` and
  `command | other-command` receive no escape sequences.
- With every argument at its default, the result is the text itself.
- Every styled String ends with a reset, so styling never leaks into later
  output. There is no 256-color, RGB, or theme support.

**Nesting.** When the text passed to `style` already contains a styled String,
the outer style is applied again after each inner reset, so it resumes where
the inner style ends:

```ahd
bring Terminal

status := Terminal.style(
    text: "Status: " + Terminal.style(
        text: "late"
        foreground: "yellow"
    ) + " today"
    bold: true
)
```

Here `Status:` and `today` are both bold.

**Standard error.** `supportsColor` and `style` describe standard output. When
standard output is a terminal but standard error is redirected, text styled for
`Terminal.error` would carry escape sequences into that file; style error text
only when both streams go to the same terminal.

## pretty

`Terminal.pretty(value)` writes one value laid out for reading, followed by a
line break, to standard output.

```ahd
bring Terminal

grades: Pair<String, List<Int>> := {"Ali": [90, 95, 100], "Ayşe": [85, 91, 97]}
Terminal.pretty(grades)
```

writes:

```text
{
    "Ali": [
        90,
        95,
        100
    ],
    "Ayşe": [
        85,
        91,
        97
    ]
}
```

- A non-empty `List` or `Pair` is written one element per line with four spaces
  of indentation per level; an empty one stays `[]` or `{}`.
- Everything inside a collection is written exactly as `str` writes it there:
  Strings in double quotes with `\"`, `\\`, `\n`, `\r`, and `\t` escaped, `null`
  as `null`, and a Class value as `<ClassName>`.
- A value that is not a List or Pair is written exactly as `write` writes it,
  including a Class's own `CStr`.
- `pretty` never looks inside a Class, so `Confidential` attributes are never
  shown.
- It is a layout for people, not a serialization format; use
  [JSON](JSON.md) to produce data for programs.

## Errors

Every error is a `TerminalError`, derived from `Error`:

```text
unsupported foreground color "purple"; use default, black, red, green, yellow, blue, magenta, cyan, or white
unsupported background color "purple"; use default, black, red, green, yellow, blue, magenta, cyan, or white
standard output could not be flushed
```

The absence of a terminal is not an error: `isInteractive()` is `false`,
`width()` and `height()` are `null`, and `supportsColor()` is `false`. Mistakes
in types or argument counts are compile-time diagnostics.

## Platform notes

- **macOS and Linux:** terminal detection and size come from the kernel's
  terminal requests on standard output.
- **Windows:** detection and size come from the console API; escape sequences
  are used only when the console already interprets them.
- Terminal starts no process, reads no file, and makes no network request. The
  only environment variables it reads are `NO_COLOR` and `TERM`. It adds no
  dependency: generated programs use the Go standard library.

## Not in v1.5.0

Cursor movement, screen clearing, raw keyboard input, mouse input, progress
bars, spinners, menus, logging, 256-color and RGB color, color switches,
separate standard-error capability queries, and a GUI.

See also: `examples/v1.5/terminal_demo` ·
[Fundamentals](FUNDAMENTALS.md).
