# Archive standard module

[English] · Türkçe

[Back to README](README.md) · [Modules](MODULES.md) · [PDF](PDF.md)

Archive packages files into real ZIP, TAR, and TAR.GZ archives and, since
v2.5.0, lists and **safely extracts** ZIP, TAR, and TAR.GZ archives — offline,
using only the Go standard library (`archive/zip`, `archive/tar`,
`compress/gzip`). Import it explicitly:

```ahd
bring Archive
from Archive bring (ArchiveEntry, ArchiveError)
```

The canonical module identity is `builtin:Archive`; a sibling `Archive.ahd`
cannot shadow it.

Extraction is **safe by default and has no unsafe variant**: path traversal,
absolute and drive paths, links, and archive bombs are rejected, and a failed
extraction leaves nothing behind. See
[Listing and safe extraction](#listing-and-safe-extraction-v250).

## Surface

```text
Archive.zip(output: String, entries: Pair<String, String>)     -> Nothing
Archive.tar(output: String, entries: Pair<String, String>)     -> Nothing
Archive.tarGzip(output: String, entries: Pair<String, String>) -> Nothing

Archive.list(archive: String, maxFiles: Int = 10000)            -> List<ArchiveEntry>
Archive.extract(archive: String, destination: String,
                maxFiles: Int = 10000, maxBytes: Int = 1073741824) -> Nothing

ArchiveEntry.path() -> String   // the name exactly as stored in the archive
ArchiveEntry.kind() -> String   // "file", "directory", "symlink", "hardlink", or "other"
ArchiveEntry.size() -> Int      // uncompressed bytes; 0 for non-files

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

## Listing and safe extraction (v2.5.0)

`Archive.list` and `Archive.extract` read ZIP, TAR, and TAR.GZ. The format is
recognized from the file's **content** (its signature), never from its name,
so `release.bin` that is really a ZIP works and a renamed text file is
refused with `unsupported archive format`.

### Listing

```ahd
entries: List<ArchiveEntry> := Archive.list("release.zip")
for entry in entries {
    write(entry.path() + " " + entry.kind() + " " + str(entry.size()))
}
```

`list` writes nothing and reports every entry **as stored**, in archive order
— including symbolic links, hard links, and unsafe names such as
`../evil` — so a program can inspect an upload before deciding anything.
`ArchiveEntry` is an opaque value: it cannot be constructed directly and is
obtained only from `Archive.list`. At most `maxFiles` entries are accepted.

### Extracting

```ahd
Archive.extract(
    archive: "uploads/release-42.zip",
    destination: "releases/42",
    maxFiles: 5000,
    maxBytes: 1073741824
)
```

The contract:

- **The destination must not exist**; its parent directory must. `extract`
  only ever creates a new directory. It never merges into, overwrites, or
  deletes existing content — extracting twice to the same destination raises
  `ArchiveError`.
- **All or nothing.** Entries are written into a private staging directory
  beside the destination (`.<name>.ahdextract-…`), which is renamed to the
  destination only after every entry succeeded. On any failure the staging
  directory — created by this call — is removed, so a failed extraction never
  leaves a half-written release that looks successful.
- **Every entry path is validated before anything is written.** Rejected:
  empty names, NUL bytes, absolute paths (`/etc/x`), UNC paths
  (`//server/share`), drive paths (`C:/x`, `C:x`), any backslash (so
  `..\evil` and mixed `a/..\..\x` cannot slip through), any colon, and any
  `..` segment (`../x`, `a/../../x`). `.` segments and a leading `./` are
  harmless and accepted, as tar tools commonly write them. After joining, the
  target is re-checked with a real relative-path computation against the
  staging directory — never by string-prefix comparison, so a sibling such as
  `releases/42-evil` cannot pass as "inside" `releases/42`. On Windows, names
  that end in a dot or space or name a reserved device (`CON`, `NUL`, …) are
  also rejected.
- **Links are rejected, never followed or converted.** Symbolic-link and
  hard-link entries, and device, FIFO, and sparse entries, make the whole
  extraction fail. There is no `allowSymlinks` option and no unsafe variant.
  Because nothing extracted can be a link, no entry can redirect a later write
  outside the destination; each parent directory is additionally resolved and
  re-checked before a file is created.
- **Bounded.** `maxFiles` limits the number of entries (files and
  directories); `maxBytes` limits the total uncompressed size. Both are
  checked first against the archive's metadata — an absurd declared size is
  refused before any byte is written — and then again while streaming,
  because metadata can lie: an entry that produces more bytes than it
  declared, or a TAR.GZ stream that expands beyond its allowance, fails the
  extraction. Size accumulation is overflow-safe.
- Files are created with mode `0644`, or `0755` when the archive marks the
  entry executable; directories get `0755` (both before the process umask).
  setuid, setgid, and sticky bits are never applied, and archived owners are
  ignored. Adjust afterwards with `File.setPermissions`.
- A TAR PAX global header (the `pax_global_header` record `git archive`
  writes) is archive metadata, not an entry: it is neither listed nor
  written.
- Encrypted ZIP entries are refused. Duplicate file entries (including names
  that collide on a case-insensitive filesystem) are refused; a repeated
  directory entry is harmless.
- Reading a ZIP loads its central directory (the entry index) into memory, in
  proportion to the archive's size. Bound the size of untrusted uploads before
  listing or extracting them.

| Bound | Default | Allowed range |
|---|---|---|
| `maxFiles` | 10 000 | 1 – 1 000 000 |
| `maxBytes` | 1 073 741 824 (1 GiB) | 1 – 68 719 476 736 (64 GiB) |

A value outside the range raises `ArchiveError`; extraction is never
unbounded.

### Errors

Every failure is an `ArchiveError` whose message names the operation and the
archive, then one stable reason, for example:

```text
extract "evil.zip" failed: unsafe entry path "../escaped.txt": the path escapes the destination
extract "links.tar.gz" failed: entry "current" is a symbolic link; safe extraction rejects links
extract "big.zip" failed: file-count limit exceeded: the archive has more than 5000 entries (maxFiles)
extract "bomb.zip" failed: byte limit exceeded: the archive declares more than 1073741824 uncompressed bytes (maxBytes)
extract "notes.txt" failed: unsupported archive format; expected ZIP, TAR, or TAR.GZ
extract "broken.tar.gz" failed: malformed archive: invalid header
extract "release.zip" failed: destination "releases/42" already exists; Archive.extract only creates a new directory
```

Raw Go error text is never exposed.

## Not in this version

RAR, 7z, BZIP2, XZ, a standalone Compress module, encrypted or
password-protected archives, an archive object model with random access,
extraction that preserves links or ownership, an unsafe extraction mode, and
directory-source recursion for creation are not part of AhdCode.
