# DNS sorgusu (v2.6.0)

[English](README.md) · [Türkçe]

`example.com` adını `DNS.lookup(host:, timeoutSeconds:)` ile çözer ve IPv4 ile IPv6 adreslerini (sıralı, tekilleştirilmiş) yazdırır; bir IP adresinin ağ gerektirmediğini ve bozuk girdinin (URL, `host:port`, kabuk benzeri metin) herhangi bir sorgudan önce reddedildiğini gösterir. Bkz. [DNS](../../../docs/DNS_TR.md).

**Ağ erişimi gerektirir.** Otomatik testler bu örneği yalnızca derler; DNS ve
TLS test takımları yerel sunucular kullanır, asla genel interneti kullanmaz.

```bash
ahdcode run main.ahd
```
