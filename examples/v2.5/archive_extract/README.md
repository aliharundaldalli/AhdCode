# Safe archive extraction (v2.5.0)

[English] · [Türkçe](README_TR.md)

Builds a small ZIP, lists it without writing anything, extracts it into a new directory with explicit `maxFiles`/`maxBytes` bounds, and shows two refusals: extracting onto an existing directory, and exceeding `maxFiles`. See [Archive](../../../docs/ARCHIVE.md).

Each run works in a fresh `run-<uuid>/` folder under the current directory,
so it can be repeated; delete those folders when you are done.

```bash
ahdcode run main.ahd
```
