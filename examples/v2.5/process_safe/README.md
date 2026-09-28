# Shell-free process execution (v2.5.0)

[English] · [Türkçe](README_TR.md)

Runs `/bin/echo` with arguments that contain shell syntax and shows they stay literal text, reads a non-zero exit code and stderr as an ordinary result, and catches a missing executable and a timeout as `ProcessError`. Uses Unix programs; on Windows substitute a real `.exe` by absolute path (batch files are refused). See [Process](../../../docs/PROCESS.md).

Each run works in a fresh `run-<uuid>/` folder under the current directory,
so it can be repeated; delete those folders when you are done.

```bash
ahdcode run main.ahd
```
