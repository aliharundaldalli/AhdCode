# v2.5.0 release examples

[English] · [Türkçe](README_TR.md)

Small, focused examples of the v2.5.0 deployment and systems primitives.
Together they cover what a deployment worker needs, without being one:

- [Safe archive extraction](archive_extract/README.md): `Archive.list` and
  `Archive.extract` with bounds, and the refusals.
- [Filesystem deployment](filesystem_deploy/README.md): copy, atomic write,
  permissions, atomic move, walk, and an atomic `current` link switch.
- [Shell-free processes](process_safe/README.md): `Process.run` with an
  argument list, exit codes, stderr, and bounded failures.
- [SQLite backup](sqlite_backup/README.md): a consistent `db.backupTo`
  snapshot of a live WAL database.
