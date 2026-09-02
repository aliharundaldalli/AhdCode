# AhdCode v0.3 örnekleri

[English](README.md) · [Türkçe]

[Proje README'sine dön](../../README_TR.md)

Bu programlar v0.3.0 SQLite kalıcılık katmanını tanıtır. `notes.db` deposunun
içinde oluşmasın diye onları bir **geçici dizinden** çalıştırın:

```bash
scratch="$(mktemp -d)"
cp examples/v0.3/01_sqlite_notes.ahd "$scratch/"
cd "$scratch"
ahdcode run 01_sqlite_notes.ahd
ahdcode run 01_sqlite_notes.ahd
```

İkinci çalışma, birincinin yazdığı notları hâlâ görür; çünkü notlar diskteki
`notes.db` dosyasındadır. `ahdsqlite`, `ahdcode` yanında kurulu olmalıdır
(`go install ./cmd/ahdsqlite`).

| Örnek | Konu |
|---|---|
| `01_sqlite_notes.ahd` | SQLite Not Defteri: oluştur, parametreyle ekle, listele, güncelle, sil, ara, kapat ve yeniden aç |
