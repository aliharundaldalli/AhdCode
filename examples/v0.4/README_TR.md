# AhdCode v0.4 örnekleri

[English](README.md) · [Türkçe]

[Proje README'sine dön](../../README_TR.md)

Bu programlar v0.4.0 web temelini tanıtır: HTTP sunucusu, istek ve yanıt
değerleri ve güvenli yapılandırılmış HTML. Sayfanın yalnızca bu makinede
kalması için `127.0.0.1` kullanın, sonra tarayıcıda `http://127.0.0.1:8080/`
adresini açın. `Server.start()` programı durdurana kadar terminali meşgul
tutar.

Web Not Defteri uygulamasını **geçici bir dizinden** çalıştırın ki `notes.db`
depo içinde oluşmasın. `ahdsqlite`, `ahdcode` yanında kurulu olmalıdır
(`go install ./cmd/ahdsqlite`). Yalnızca HTTP örnekleri SQLite yardımcısına
ihtiyaç duymaz.

```bash
scratch="$(mktemp -d)"
cp examples/v0.4/03_web_notes.ahd "$scratch/"
cd "$scratch"
ahdcode run 03_web_notes.ahd
```

| Örnek | Konu |
|---|---|
| `01_http_hello.ahd` | HTTP üzerinden güvenilir statik HTML |
| `02_http_request.ahd` | Sorgu parametreleri ve `application/x-www-form-urlencoded` formlar |
| `03_web_notes.ahd` | Web Not Defteri: SQLite kalıcılığı, kaçırılmış HTML, POST-redirect-GET |
