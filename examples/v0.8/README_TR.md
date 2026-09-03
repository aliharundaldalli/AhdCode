# AhdCode v0.8 örnekleri

[English](README.md) · [Türkçe]

[Proje README'sine dön](../../README.md)

Bu programlar v0.8.0 multipart formlarını ve güvenli dosya yüklemeyi tanıtır:
`Request.file` / `Request.files` `UploadedFile` değerleri döndürür ve
`UploadedFile.save(directory)` dosyayı, yükleyenin etkileyemeyeceği
kriptografik rastgele bir adla kaydeder. Tarayıcının verdiği dosya adı,
uzantısı ve bildirilen Content-Type'ı güvenilmezdir —
`detectedContentType()` gerçek baytları inceler. Yüklenen baytlar asla bir
AhdCode `String`'i değildir ve kaydedilmeyen hiçbir şey kendi isteğinden
uzun yaşamaz.

```bash
ahdcode run examples/v0.8/01_file_upload.ahd
```

Her örnek `127.0.0.1:8080` üzerinde hizmet verir. Portu aramadan durdurun:

```bash
ahdcode kill examples/v0.8/01_file_upload.run
```

| Örnek | Konu |
|---|---|
| `01_file_upload.ahd` | Multipart form, metin alanı + dosya alanı, yükleme meta verisi |
| `02_pdf_upload.ahd` | Algılanan MIME doğrulaması, boyut politikası, güvenli `save` |
| `03_upload_to_sqlite.ahd` | Dosya diskte, yol ve meta veri SQLite'ta — asla BLOB değil |

02 ve 03 örnekleri çalışma dizinine göre `uploads/papers/` oluşturur; 03
ayrıca `papers.db` oluşturur. Bunları geçici bir dizinde çalıştırın;
yüklemeler de veritabanı da sürüm kontrolüne girmemelidir.
