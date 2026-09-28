# Filesystem deployment primitives (v2.5.0)

[English] · [Türkçe](README_TR.md)

Stages a release, copies a binary without overwriting, replaces a config file atomically, sets permissions with an octal String (`"0750"`), promotes staging to `releases/42` with one rename, walks the release without following links, and switches a `current` symbolic link atomically. Uses Unix symbolic links and permissions. See [File](../../../docs/FILESYSTEM.md).

Each run works in a fresh `run-<uuid>/` folder under the current directory,
so it can be repeated; delete those folders when you are done.

```bash
ahdcode run main.ahd
```
