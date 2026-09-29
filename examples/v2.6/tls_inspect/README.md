# TLS inspection (v2.6.0)

[English] · [Türkçe](README_TR.md)

Inspects `example.com:443` with `TLS.inspect`, prints the verification result, subject, issuer, names, protocol, and expiry date, then applies an application-defined renewal policy (7 and 30 days) using `Time.between`. A port without a TLS service shows the difference between a `TLSError` and an invalid certificate. See [TLS](../../../docs/TLS.md).

**Requires network access.** The automated tests only compile this example;
the DNS and TLS test suites use local servers and never the public internet.

```bash
ahdcode run main.ahd
```
