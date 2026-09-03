# AhdCode v0.9.1 examples

[English] · [Türkçe](README_TR.md)

[Back to project README](../../README.md)

This program introduces the v0.9.1 binary-safe file response primitives,
`HTTP.file` and `HTTP.download`. It does not change how uploads are stored:
`UploadedFile.save` still returns an opaque, extensionless path, exactly as
in v0.8.0.

```bash
ahdcode run examples/v0.9.1/01_upload_and_serve.ahd
curl -F "paper=@paper.pdf" http://127.0.0.1:8080/upload
curl http://127.0.0.1:8080/view?id=1 -o view.pdf
curl http://127.0.0.1:8080/download?id=1 -o ozet.pdf
```

| Example | Topic |
|---|---|
| `01_upload_and_serve.ahd` | Upload a PDF, store its opaque path and content type in SQLite, then serve it inline (`HTTP.file`) and as a named download (`HTTP.download`) |

The stored path never carries or needs an extension. The download's
presentation filename (`"ozet.pdf"` here) is independent of it.
