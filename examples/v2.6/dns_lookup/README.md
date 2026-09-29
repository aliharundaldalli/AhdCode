# DNS lookup (v2.6.0)

[English] · [Türkçe](README_TR.md)

Resolves `example.com` with `DNS.lookup(host:, timeoutSeconds:)` and prints its IPv4 and IPv6 addresses (sorted, deduplicated), shows that an IP literal needs no network, and shows malformed input (a URL, a `host:port`, shell-like text) refused before any lookup. See [DNS](../../../docs/DNS.md).

**Requires network access.** The automated tests only compile this example;
the DNS and TLS test suites use local servers and never the public internet.

```bash
ahdcode run main.ahd
```
