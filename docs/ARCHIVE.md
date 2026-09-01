# Archive standard module

[English] · [Türkçe](ARCHIVE_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [PDF](PDF.md)

Archive packages files into real ZIP, TAR, and TAR.GZ archives, offline,
using only the Go standard library (`archive/zip`, `archive/tar`,
`compress/gzip`). Import it explicitly:

```ahd
bring Archive
from Archive bring ArchiveError
```

The canonical module identity is `builtin:Archive`; a sibling `Archive.ahd`
cannot shadow it.

Archive is **creation-only**: there is no extraction, listing, or archive
object model. `Archive.extract`, `Archive.open`, and similar do not exist and
will not be added to this module — see [Not in this version](#not-in-this-version).

## Surface

```text
Archive.zip(output: String, entries: Pair<String, String>)     -> Nothing
Archive.tar(output: String, entries: Pair<String, String>)     -> Nothing
Archive.tarGzip(output: String, entries: Pair<String, String>) -> Nothing

ArchiveError
```

## Entry mapping

`entries` is an ordinary `Pair<String, String>`: each key is the path *inside*
the archive, and each value is the *source filesystem path* to package. The
mapping is always explicit — Archive never guesses a destination name from a
source path.

```ahd
files := {
    "report/report.pdf": "output/report.pdf"
    "data/results.json": "results.json"
    "images/chart.png": "chart.png"
}

Archive.zip("submission.zip", files)
```

produces

```text
submission.zip
├── report/
│   └── report.pdf
├── data/
│   └── results.json
└── images/
    └── chart.png
```

## Regular files only

v0.1.20 Archive accepts regular files only — no directory sources, no
recursive expansion. A source that is a directory, a symbolic link, or any
other non-regular file raises `ArchiveError`. This keeps the safety argument
(path validation, symlink handling, ordering) small and fully auditable for
the first release; directory sources may be reconsidered in a future release
without changing today's contract.

## Entry path safety

Archive member names are canonical relative forward-slash paths. Every one of
the following is rejected outright — never silently normalized into something
else:

- empty name
- an absolute path (`/etc/...`)
- a `..` or `.` path segment (`../escape`, `a/../b`, `./file`)
- a doubled slash (`a//b`)
- a backslash (`a\b`)
- a NUL byte
- a Windows drive-prefix-like segment (`C:file`)

Source filesystem paths are ordinary paths; they are not member names and are
not subject to the same-slash-only rule.

## Symlinks

A source that is a symbolic link is rejected with `ArchiveError` rather than
followed, stored, or dereferenced silently. This avoids ever packaging a file
outside the caller's intended tree.

## Collisions

`Pair` already guarantees unique keys, so two entries cannot name the same
archive member; Archive additionally checks this defensively. There is no
`last wins`, `first wins`, or silent overwrite behavior to reason about.

## Determinism and ordering

Archive member order follows `Pair` insertion order exactly — the order the
entries were written in source. Archive metadata that would otherwise vary
run to run is normalized:

- ZIP: no per-entry modification timestamp, a fixed `0644` mode, standard
  Deflate compression.
- TAR: no modification time, owner, or group; a fixed `0644` mode.
- TAR.GZ: no gzip header name, comment, or modification time.

File **content** is preserved exactly; no other filesystem metadata (original
timestamps, permissions beyond the fixed mode, extended attributes) is
promised to survive. Two archives built from equivalent entries are
byte-for-byte identical.

## Format and extension

The function you call selects the format — Archive never guesses a format
from the destination extension — but a mismatched extension still raises
`ArchiveError` rather than silently writing the wrong bytes to the wrong
name:

```text
Archive.zip     ->  output must end in .zip
Archive.tar     ->  output must end in .tar
Archive.tarGzip ->  output must end in .tar.gz   (not .tgz)
```

An empty `entries` Pair produces a valid, empty archive in all three formats.

## Output safety

Archive builds the complete archive into a same-directory temporary file,
then atomically renames it over the destination — the same pattern
`Excel.save`/`Word.save`/`Latex.pdf` all use. A failed build never touches or
destroys an existing valid archive at the destination path, and a source path
that resolves to the destination archive itself is rejected before anything
is written.

## Errors

`ArchiveError` covers every Archive-specific failure: a missing or unreadable
source, an unsupported source type (directory or symlink), an invalid entry
path, a wrong output extension, and an archive-writer failure.

```ahd
attempt {
    Archive.zip("out.zip", {"a.txt": "missing.txt"})
}
except ArchiveError as error {
    write(error.message)
}
```

Static argument count and type mistakes remain compiler diagnostics; they do
not become runtime `ArchiveError` values.

## Not in this version

Archive extraction, archive listing, an archive object model, RAR, 7z,
BZIP2, XZ, a standalone Compress module, encrypted/password-protected
archives, and directory-source recursion are not part of v0.1.20.
