# AhdCode v0.9 examples

[English] · [Türkçe](README_TR.md)

[Back to project README](../../README.md)

These programs introduce v0.9.0 send-only SMTP mail. They do **not** send
mail to a real address by themselves. Point them at a local SMTP test server
with environment variables:

```bash
export SMTP_HOST=127.0.0.1
export SMTP_PORT=2525
export SMTP_SECURITY=none
export SMTP_FROM=sender@example.com
export SMTP_TO=student@example.com
ahdcode run examples/v0.9/01_text_mail.ahd
```

For a real server, set `SMTP_SECURITY` to `starttls` or `tls` and supply
`SMTP_USERNAME` / `SMTP_PASSWORD` through the environment. Never put a real
password in the example source.

| Example | Topic |
|---|---|
| `01_text_mail.ahd` | Plain-text message through `SMTP.client` / `SMTP.message` |
| `02_html_mail.ahd` | HTML and multipart/alternative mail |
| `03_env_smtp.ahd` | Env-configured host, port, security, and AUTH PLAIN |

There is no `ahdsmtp` helper. TLS uses the runtime libraries.
