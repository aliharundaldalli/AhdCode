# Consistent SQLite backup (v2.5.0)

[English] · [Türkçe](README_TR.md)

Opens a WAL-mode database, writes rows, backs it up with `db.backupTo` while it stays open, changes it afterwards, and verifies that the backup holds exactly the snapshot and passes `PRAGMA integrity_check`. A second backup to the same name is refused. See [SQLite](../../../docs/SQLITE.md).

Each run works in a fresh `run-<uuid>/` folder under the current directory,
so it can be repeated; delete those folders when you are done.

```bash
ahdcode run main.ahd
```
