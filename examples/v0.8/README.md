# AhdCode v0.8 examples

[English] · [Türkçe](README_TR.md)

[Back to project README](../../README.md)

These programs introduce v0.8.0 multipart forms and safe file uploads:
`Request.file` / `Request.files` return `UploadedFile` values, and
`UploadedFile.save(directory)` persists one under a crypto-random name that
the uploader cannot influence. The browser-supplied filename, its extension,
and the declared Content-Type are all untrusted — `detectedContentType()`
sniffs the actual bytes. Uploaded bytes are never an AhdCode `String`, and
nothing unsaved outlives its request.

```bash
ahdcode run examples/v0.8/01_file_upload.ahd
```

Each example serves on `127.0.0.1:8080`. Stop one without hunting for the
port:

```bash
ahdcode kill examples/v0.8/01_file_upload.run
```

| Example | Topic |
|---|---|
| `01_file_upload.ahd` | Multipart form, text field + file field, upload metadata |
| `02_pdf_upload.ahd` | Detected-MIME validation, size policy, safe `save` |
| `03_upload_to_sqlite.ahd` | File on disk, path and metadata in SQLite — never a BLOB |

Examples 02 and 03 create `uploads/papers/` relative to the working
directory, and 03 also creates `papers.db`. Run them from a scratch
directory; neither the uploads nor the database belong in version control.
