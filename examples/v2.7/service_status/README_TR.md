# Servis durumu (v2.7.0)

[English](README.md) · [Türkçe]

Birkaç systemd biriminin durumunu `Service.status` ile okur ve etkin olmayanları bildirir. **systemd'li Linux gerektirir**; macOS ve Windows'ta her çağrı `ServiceError` fırlatır, örnek bunu yakalayıp yazdırır. Hiçbir şeyi başlatmaz ya da durdurmaz. Bkz. [Service](../../../docs/SERVICE_TR.md).

```bash
ahdcode run main.ahd
```
