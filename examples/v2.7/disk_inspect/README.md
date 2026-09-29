# Disk inspection (v2.7.0)

[English] · [Türkçe](README_TR.md)

Reads the capacity of the filesystem that holds the current directory with `Disk.inspect`, prints total and available gigabytes and the used percentage, applies the program's own 90% warning threshold, and shows a missing path raising `DiskError`. Safe on any system. See [Disk](../../../docs/DISK.md).

```bash
ahdcode run main.ahd
```
