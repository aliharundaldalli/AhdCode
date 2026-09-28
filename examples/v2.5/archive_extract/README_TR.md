# Güvenli arşiv çıkarma (v2.5.0)

[English](README.md) · [Türkçe]

Küçük bir ZIP oluşturur, hiçbir şey yazmadan listeler, açık `maxFiles`/`maxBytes` sınırlarıyla yeni bir dizine çıkarır ve iki reddi gösterir: mevcut bir dizinin üzerine çıkarma ve `maxFiles` sınırını aşma. Bkz. [Archive](../../../docs/ARCHIVE_TR.md).

Her çalıştırma geçerli dizin altında yeni bir `run-<uuid>/` klasöründe
çalışır; bu yüzden tekrarlanabilir. İşiniz bitince bu klasörleri silin.

```bash
ahdcode run main.ahd
```
