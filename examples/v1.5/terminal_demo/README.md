# v1.5.0 Terminal demo

[English] · [Türkçe](README_TR.md)

A small program that shows every part of the [`Terminal`](../../../docs/TERMINAL.md)
standard module next to the unchanged `write`.

## Run it

```bash
ahdcode run main.ahd
```

Then run it again with standard output redirected, and once more without color:

```bash
ahdcode run main.ahd > output.txt
NO_COLOR=1 ahdcode run main.ahd
```

## What to look for

| Line | Shows |
| --- | --- |
| 1 | `write` behaves exactly as before |
| 2–4 | `Terminal.emit` with the default separator, a custom separator, and an empty ending |
| 5 | `Terminal.error` writes to standard error, so it stays on the screen when output is redirected |
| 6–8 | `isInteractive`, `width`/`height`, and `supportsColor` describe standard output |
| 9 | `Terminal.style` colors words in a terminal and returns plain text otherwise |
| 10 | Unicode is written unchanged |
| 11 | `Terminal.pretty` lays out a `Pair<String, List<Int>>` over several lines |

Redirected, lines 6–8 read:

```text
6. interactive: false
7. size: unknown, because standard output is not a terminal
8. color: false
```

and `output.txt` contains no escape sequences.
