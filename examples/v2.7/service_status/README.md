# Service status (v2.7.0)

[English] · [Türkçe](README_TR.md)

Reads the state of a few systemd units with `Service.status` and reports any that are not active. **Requires Linux with systemd**; on macOS and Windows every call raises `ServiceError`, which the example catches and prints. It never starts or stops anything. See [Service](../../../docs/SERVICE.md).

```bash
ahdcode run main.ahd
```
