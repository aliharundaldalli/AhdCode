# AhdCode v0.3 examples

[English] · [Türkçe](README_TR.md)

[Back to project README](../../README.md)

These programs introduce the v0.3.0 SQLite persistence layer. Run them from a
**temporary directory** so `notes.db` is not created inside the repository:

```bash
scratch="$(mktemp -d)"
cp examples/v0.3/01_sqlite_notes.ahd "$scratch/"
cd "$scratch"
ahdcode run 01_sqlite_notes.ahd
ahdcode run 01_sqlite_notes.ahd
```

The second run still sees the notes written by the first, because they live in
`notes.db` on disk. `ahdsqlite` must be installed beside `ahdcode`
(`go install ./cmd/ahdsqlite`).

| Example | Topic |
|---|---|
| `01_sqlite_notes.ahd` | SQLite Notes App: create, insert with parameters, list, update, delete, search, close, and reopen |
