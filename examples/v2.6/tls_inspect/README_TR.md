# TLS incelemesi (v2.6.0)

[English](README.md) · [Türkçe]

`example.com:443` uç noktasını `TLS.inspect` ile inceler; doğrulama sonucunu, konuyu, vereni, adları, protokolü ve bitiş tarihini yazdırır, ardından `Time.between` ile uygulamanın belirlediği bir yenileme politikası (7 ve 30 gün) uygular. TLS hizmeti olmayan bir port, `TLSError` ile geçersiz bir sertifika arasındaki farkı gösterir. Bkz. [TLS](../../../docs/TLS_TR.md).

**Ağ erişimi gerektirir.** Otomatik testler bu örneği yalnızca derler; DNS ve
TLS test takımları yerel sunucular kullanır, asla genel interneti kullanmaz.

```bash
ahdcode run main.ahd
```
