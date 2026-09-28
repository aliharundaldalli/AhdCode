# File and Path modules

[English] · [Türkçe](FILESYSTEM_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Errors](ERRORS.md)

Import the modules explicitly:

```ahd
bring Path
bring File
from File bring (FileEntry, FileError)
```

## Path

`Path` performs pure, host-operating-system-aware path operations:

```text
Path.join(parts: List<String>) -> String
Path.ext(path: String)         -> String
Path.base(path: String)        -> String
Path.dir(path: String)         -> String
```

```ahd
filePath := Path.join(["reports", "result.txt"])
write(Path.ext(filePath))
write(Path.base(filePath))
write(Path.dir(filePath))
```

## File

```text
File.exists(path: String)                     -> Bool
File.readText(path: String)                   -> String
File.writeText(path: String, content: String) -> Nothing
File.append(path: String, content: String)    -> Nothing
File.delete(path: String)                     -> Nothing
File.createDir(path: String)                  -> Nothing
File.list(path: String)                       -> List<String>
```

Text is UTF-8. `File.list` returns immediate entry names in stable ascending
lexical order; it is not recursive. Relative paths use the process working
directory. In a REPL that is the directory from which `ahdcode` was launched.

```ahd
File.createDir("notes")
File.writeText("notes/today.txt", "first")
File.append("notes/today.txt", " second")
write(File.readText("notes/today.txt"))
write(File.list("notes"))
```

`File.exists` returns `false` for a missing path. Failures of the other File
operations raise `FileError`, which derives from `IOError` and `Error`:

```ahd
attempt {
    File.readText("missing.txt")
}
except FileError as error {
    write(error.message)
}
```

File operations never expose host error objects. File has no general OS
module, no ownership (`chown`) API, no file watching, and no generic binary
read/write; the deployment primitives below are deliberately narrow.

## Deployment primitives (v2.5.0)

```text
File.copy(source: String, destination: String)        -> Nothing
File.atomicWrite(path: String, content: String)       -> Nothing
File.atomicMove(source: String, destination: String)  -> Nothing
File.symlink(target: String, link: String)            -> Nothing
File.readLink(path: String)                           -> String
File.isSymlink(path: String)                          -> Bool
File.permissions(path: String)                        -> String   // "0755"
File.setPermissions(path: String, mode: String)       -> Nothing  // "0755" or "755"
File.walk(path: String, maxEntries: Int = 100000)     -> List<FileEntry>

FileEntry.path()         -> String   // root joined with the relative path
FileEntry.relativePath() -> String   // forward slashes, relative to the root
FileEntry.kind()         -> String   // "file", "directory", "symlink", or "other"
FileEntry.size()         -> Int      // bytes for files; 0 otherwise
FileEntry.isSymlink()    -> Bool
```

Every failure raises `FileError` with the operation, the path(s) as the
program wrote them, and one plain reason (`no such file or directory`,
`permission denied`, `the path already exists`, …).

### `File.copy` — binary-safe copy

`copy` duplicates one **regular file** byte for byte. It streams through a
fixed 64 KiB buffer, so it never loads the file into memory and never passes
through `String`: binaries, images, and archives are copied exactly.

- The source must be a regular file. A symbolic-link source is **refused, not
  followed**; a directory is refused.
- The destination must **not** exist: `copy` never overwrites. Replace a file
  deliberately with `File.atomicMove`.
- The copy is written to a temporary file beside the destination, flushed,
  and only then given its final name, so a failure never leaves a partial
  destination. On Unix the destination gets the source's permission bits.

### `File.atomicWrite` — replace a file in one step

`atomicWrite` writes UTF-8 text to a temporary file **in the same directory**
(so on the same filesystem) as `path`, flushes it to stable storage, and
renames it over `path`. Readers see either the old complete file or the new
complete file, never a mixture. The temporary file is removed on any failure.

- An existing file keeps its permission bits; a new file is created with
  `0644` on Unix.
- A symbolic link at `path` is replaced by the new file (the link itself, not
  its target). A directory at `path` is refused.
- On Unix, `rename` is atomic by POSIX and the directory is synced afterwards.
  On Windows the replacement uses `MoveFileEx` with replace-existing, which is
  the strongest step Windows offers but is not documented as atomic under
  every failure; AhdCode does not claim more than the platform provides.

Write JSON and other brace-containing text as a raw String so the braces are
not interpolation: `File.atomicWrite("config.json", r'{"port": 8080}')`.

### `File.atomicMove` — rename, never copy

`atomicMove` is the operating system's single `rename`. It is the primitive
for "staging → live":

- Same filesystem: the move is atomic. An existing **file or symbolic link**
  at `destination` is replaced atomically.
- An existing **directory** at `destination` is always refused — even an
  empty one — so the result never depends on the platform's empty-directory
  rules.
- Different filesystems: `FileError` with
  `source and destination are on different filesystems; File.atomicMove never copies`.
  There is no hidden copy-and-delete.

### Symbolic links

```ahd
File.symlink(target: "releases/42", link: "current")
write(File.readLink("current"))      // releases/42
write(str(File.isSymlink("current"))) // true
```

**Argument order is `target`, then `link`** — the same order as
`ln -s target link`: `link` is the new path being created, `target` is the
text it points to. Using the parameter names, as above, makes every call
self-explanatory. The target is stored verbatim; a relative target is
resolved by the operating system relative to the link's own directory, and
it need not exist. The link path must not exist yet.

`readLink` returns the stored target without resolving it and fails on a path
that is not a link. `isSymlink` inspects the path itself (it never follows
it) and returns `false` for a missing path, like `File.exists`.
`File.delete` removes a link itself, never its target.

The deployment switch pattern creates the new link beside the live one and
renames it over the live one in one atomic step:

```ahd
File.symlink(target: "releases/43", link: "current.next")
File.atomicMove(source: "current.next", destination: "current")
```

On **Windows**, creating symbolic links requires Developer Mode or the
`SeCreateSymbolicLinkPrivilege`; without it `File.symlink` raises `FileError`
saying so. It never falls back to copying.

### Permissions — an octal String

AhdCode has no octal number literals, and a decimal `Int` is dangerous here:
`0444` written as an Int would be the decimal number 444, which is mode
`0674`. File permissions therefore use a **String of octal digits**:

```ahd
File.setPermissions("releases/42/app/bin/server", "0750")
write(File.permissions("releases/42/app/bin/server"))   // 0750
```

- Accepted: exactly three octal digits, optionally preceded by one `0`
  (`"755"`, `"0755"`, `"0640"`, `"0000"`).
- Rejected with `FileError`: `"0o755"`, `"0x1ed"`, `"493"`, `"7777"`,
  `"4755"`, `"00755"`, `"rwxr-xr-x"`, empty text, and anything else. setuid,
  setgid, and sticky bits cannot be set.
- `permissions` always returns four digits (`"0644"`).
- Symbolic links are refused by both functions rather than followed, so a
  link can never redirect a permission change to another file.
- **Windows:** both functions raise `FileError`
  (`Unix permission bits are not supported on Windows`). Windows access
  control lists cannot be represented honestly as mode bits, so AhdCode does
  not pretend `chmod` exists there.

### `File.walk` — recursive listing that never follows links

```ahd
entries: List<FileEntry> := File.walk("releases/42")
for entry in entries {
    write(entry.relativePath() + " " + entry.kind() + " " + str(entry.size()))
}
```

- Lists every entry below the root (the root itself excluded) in
  deterministic lexical order, depth first.
- **Symbolic links are never followed.** A link is reported as one entry of
  kind `"symlink"` and is not descended into, whether it points to a file, a
  directory, outside the root, or back up the tree. Link cycles therefore
  cannot loop the walk. A root that is itself a link is refused (read it with
  `File.readLink` and walk the target explicitly if that is what you mean).
- The result is materialized, so it is bounded: more than `maxEntries`
  entries raises `FileError`. The default is 100 000; the allowed range is
  1 – 1 000 000.
- Mount points are not detected; a walk continues into a mounted directory.

## Security notes for privileged programs

These primitives are designed for deployment workers that may run with
elevated rights: nothing follows a symbolic link implicitly except
`File.exists`, `readText`, `writeText`, and `append` (which keep their
original behavior); `copy`, `permissions`, `setPermissions`, and `walk`
refuse links. Paths are still checked and then used, so an attacker who can
rewrite the directories you operate on concurrently can race any
filesystem API — operate on directories only your worker can write.
