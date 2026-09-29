# Disk standard module

[English] · Türkçe

[Back to README](README.md) · [Modules](MODULES.md) · [Service](SERVICE.md) · [File](FILESYSTEM.md) · [Errors](ERRORS.md)

`Disk` (v2.7.0) reports the capacity of the filesystem that contains a path:
how big it is, how much is used, and how much is still free. It is read-only,
uses the operating system's own filesystem API, and never runs a shell or a
tool such as `df`, `diskutil`, `wmic`, or PowerShell. Import it explicitly:

```ahd
bring Disk
from Disk bring (DiskInfo, DiskError)
```

The canonical module identity is `builtin:Disk`; a sibling `Disk.ahd` cannot
shadow it. (Since v2.7.0 `Disk` is a standard module name: a local `Disk.ahd`
is no longer what `bring Disk` loads — rename such a file.)

## Surface

```text
Disk.inspect(path: String) -> DiskInfo

DiskInfo.path()           -> String   // the path as the program passed it
DiskInfo.totalBytes()     -> Int      // size of the filesystem
DiskInfo.usedBytes()      -> Int      // totalBytes - freeBytes
DiskInfo.freeBytes()      -> Int      // every free byte, including reserved space
DiskInfo.availableBytes() -> Int      // free bytes this program's user may use
DiskInfo.usedPercent()    -> Real     // usedBytes / totalBytes * 100

DiskError  (derives from Error)
```

`DiskInfo` is an opaque value produced only by `Disk.inspect`.

```ahd
disk: DiskInfo := Disk.inspect("/")
write(str(disk.usedPercent()) + "% used")
write(str(disk.availableBytes()) + " bytes available")
```

## Which filesystem

`Disk.inspect(path)` inspects the filesystem that **contains** `path`. The
path does not have to be a mount point: `Disk.inspect("/var/www/site")`
reports the filesystem that holds that directory, which may be `/` or a
separately mounted volume. The path must exist. A relative path is relative
to the working directory.

## The numbers

All values are bytes, never negative, and read in one call, so they are
consistent with each other:

- `totalBytes` — the size of the filesystem.
- `freeBytes` — all free space, **including** space the system reserves for
  its administrator (on Linux ext4 this is typically 5%).
- `availableBytes` — the free space an ordinary process of the current user
  can actually write. It is never larger than `freeBytes`.
- `usedBytes` — `totalBytes - freeBytes`.
- `usedPercent` — `usedBytes / totalBytes * 100`, between 0 and 100 (0 for a
  filesystem that reports no capacity).

`usedBytes` and `usedPercent` count reserved space as free, so `usedPercent`
can be slightly lower than the "Use%" column of `df`, which divides by
`used + available`. For "can this user still write?" decisions, use
`availableBytes`.

## Platform behavior

| Platform | Source | Notes |
|---|---|---|
| Linux | `statfs` | block counts in the fundamental block size; matches `df -B1` byte for byte |
| macOS | `statfs` | on APFS, volumes in one container share space, so several volumes can report the same free space |
| Windows | `GetDiskFreeSpaceExW` | `availableBytes` reflects the user's disk quota; `freeBytes` is the whole volume's free space |

A value that would not fit an `Int` is refused with a `DiskError` rather than
wrapped around.

## Errors

Every failure raises `DiskError`, with the message
`inspect disk of "<path>" failed: <reason>`:

```text
inspect disk of "" failed: the path is empty
inspect disk of "/missing" failed: no such file or directory
inspect disk of "/root/private" failed: permission denied
```

```ahd
attempt {
    disk: Local DiskInfo := Disk.inspect("/srv/data")
    if disk.usedPercent() > 90.0 {
        write("warning: /srv/data is " + str(disk.usedPercent()) + "% full")
    }
}
except DiskError as failure {
    write(failure.message)
}
```

Thresholds such as "warn above 90%" are the application's policy; the module
only reports facts.

## Security: worker-only for untrusted input

`Disk.inspect` reveals the size and fill level of any existing path it is
given. Do not pass untrusted web input to it. In a control panel, keep it in a
dedicated worker that inspects validated, allowlisted paths:

```text
web application
    -> authenticated control channel
        -> dedicated AhdCode worker
            -> validated path
                -> Disk.inspect
```

## Not in this version

Listing mounts or volumes, inode counts, per-directory usage (`du`), I/O
statistics, quotas management, and any change to disks are not part of
`Disk`.
