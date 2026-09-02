# AhdCode v0.6 examples

[English] · [Türkçe](README_TR.md)

[Back to project README](../../README.md)

These programs introduce v0.6.0 outbound HTTP: a reusable `Client`, immutable
`ClientRequest` / `ClientResponse` values, HTTPS with system certificate
verification, and explicit JSON/Env interoperability. There is no AI vendor
module and no HTTP helper process.

```bash
ahdcode run examples/v0.6/01_https_get.ahd
```

| Example | Topic |
|---|---|
| `01_https_get.ahd` | HTTPS GET; print status, final URL, and body |
| `02_custom_request.ahd` | `ClientRequest` headers and POST body |
| `03_json_api.ahd` | Existing JSON + Env token + HTTP Client |

Example 03 reads `API_URL` and `API_TOKEN`. Put them in the environment or a
local `.env` file. Never commit a real token.
