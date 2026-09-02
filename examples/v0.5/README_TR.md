# AhdCode v0.5 örnekleri

[English](README.md) · [Türkçe]

[Proje README'sine dön](../../README_TR.md)

Bu programlar v0.5.0 web durumunu tanıtır: HTTP çerezleri ve bellek içi
sunucu taraflı oturum. `127.0.0.1` bağlayın, ardından
`http://127.0.0.1:8080/` açın. `Server.start()` programı durdurana kadar
terminali meşgul eder.

Oturum değerleri yalnızca o süreç çalışırken vardır. Programı yeniden
başlatmak tüm oturumları unutturur. Bu beklenen davranıştır. Yalnızca HTTP
kullanan örnekler SQLite yardımcısı veya çerez/oturum yardımcısı gerektirmez.

```bash
ahdcode run examples/v0.5/01_cookie.ahd
```

| Örnek | Konu |
|---|---|
| `01_cookie.ahd` | Çerez okuma, yazma, silme |
| `02_session_counter.ahd` | Tarayıcı başına sayaç; Int dönüşümü açıktır |
| `03_session_login.ahd` | continue'da rotate, çıkışta destroy, iki tarayıcı bağımsızlığı |

Bu üretim kimlik doğrulaması değildir. Oturum, yazdığınız adı saklar.
