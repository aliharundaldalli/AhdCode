# Formatter

[Back to README](../README.md) · [CLI](CLI.md)

Format a source file in place:

```bash
ahdcode format program.ahd
```

Check formatting without modifying the file:

```bash
ahdcode format --check program.ahd
```

The formatter is token/AST-aware. It preserves comments, string escapes,
interpolation, and exact multiline-string content. It prefers commas for short
same-line lists/calls and one item per line in multiline constructs.

Formatting is deterministic and idempotent. In-place updates use an atomic
temporary-file replacement and preserve the file's permission bits. Invalid
source is diagnosed and left unchanged.
