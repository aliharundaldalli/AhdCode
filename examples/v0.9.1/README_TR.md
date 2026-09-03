# AhdCode v0.9.1 örnekleri

[English](README.md) · Türkçe

[Proje README'sine dön](../../README_TR.md)

Bu program, v0.9.1'in ikili-güvenli dosya yanıtı ilkellerini tanıtır:
`HTTP.file` ve `HTTP.download`. Yüklemelerin depolanma biçimini değiştirmez:
`UploadedFile.save`, v0.8.0'daki gibi opak, uzantısız bir yol döndürmeye
devam eder.

```bash
ahdcode run examples/v0.9.1/01_upload_and_serve.ahd
curl -F "paper=@paper.pdf" http://127.0.0.1:8080/upload
curl http://127.0.0.1:8080/view?id=1 -o view.pdf
curl http://127.0.0.1:8080/download?id=1 -o ozet.pdf
```

| Örnek | Konu |
|---|---|
| `01_upload_and_serve.ahd` | Bir PDF yükle, opak yolunu ve içerik türünü SQLite'ta sakla, sonra satır içi (`HTTP.file`) ve adlandırılmış indirme (`HTTP.download`) olarak sun |

Depolanan yol hiçbir zaman bir uzantı taşımaz veya buna ihtiyaç duymaz.
İndirmenin sunum dosya adı (burada `"ozet.pdf"`) ondan bağımsızdır.
