# REPL

[Back to README](../README.md) · [CLI](CLI.md)

Start an interactive session:

```bash
ahdcode
```

```text
AhdCode v0.1
ahd> x: Int := 5
ahd> write(x^2)
25
ahd> x = 7
```

The REPL uses the normal lexer, parser, semantic checker, lowering, backend,
and native runtime. Successful submissions are accumulated and recompiled.
Failed semantic or runtime submissions are not committed, so the last good
state remains available. Ordinary rules still apply: redeclaring `x` in the
same scope is an error; mutate it with `=`.

Multiline Functions, Classes, blocks, and expressions use the continuation
prompt `...>`.

## v0.1 replay limitations

Each replayed native process receives an isolated end-of-input stream. A
`take()` inside the REPL therefore returns `""` instead of consuming the next
REPL command. Use `take` in a file run for interactive input.

Previously emitted output must replay deterministically. Seed random work
explicitly before `Math.random`, `Math.randomInt`, or `List.shuffle` in a REPL
session; entropy-initialized random output may otherwise fail the replay
consistency check.
