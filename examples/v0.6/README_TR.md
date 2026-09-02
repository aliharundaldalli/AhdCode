# AhdCode v0.6 örnekleri

[English](README.md) · [Türkçe]

[Proje README'sine dön](../../README.md)

Bu programlar v0.6.0 giden HTTP'yi tanıtır: yeniden kullanılabilir bir
`Client`, değişmez `ClientRequest` / `ClientResponse` değerleri, sistem
sertifika doğrulamalı HTTPS ve açık JSON/Env birlikte çalışması. Yapay zeka
satıcı modülü ve HTTP yardımcı süreci yoktur.

```bash
ahdcode run examples/v0.6/01_https_get.ahd
```

| Örnek | Konu |
|---|---|
| `01_https_get.ahd` | HTTPS GET; durum, son URL ve gövde |
| `02_custom_request.ahd` | `ClientRequest` başlıkları ve POST gövdesi |
| `03_json_api.ahd` | Mevcut JSON + Env jetonu + HTTP Client |

Örnek 03 `API_URL` ve `API_TOKEN` okur. Bunları ortama veya yerel bir `.env`
dosyasına koyun. Gerçek bir jetonu asla commit etmeyin.
