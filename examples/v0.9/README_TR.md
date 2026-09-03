# AhdCode v0.9 örnekleri

[English](README.md) · [Türkçe]

[Proje README'sine dön](../../README_TR.md)

Bu programlar v0.9.0 yalnızca-gönderim SMTP postasını tanıtır. Kendi başlarına
gerçek bir adrese posta göndermezler. Ortam değişkenleriyle yerel bir SMTP
test sunucusuna yöneltin:

```bash
export SMTP_HOST=127.0.0.1
export SMTP_PORT=2525
export SMTP_SECURITY=none
export SMTP_FROM=sender@example.com
export SMTP_TO=student@example.com
ahdcode run examples/v0.9/01_text_mail.ahd
```

Gerçek bir sunucu için `SMTP_SECURITY` değerini `starttls` veya `tls` yapın ve
`SMTP_USERNAME` / `SMTP_PASSWORD` değerlerini ortamdan verin. Gerçek bir
parolayı örnek kaynağa yazmayın.

| Örnek | Konu |
|---|---|
| `01_text_mail.ahd` | `SMTP.client` / `SMTP.message` ile düz metin ileti |
| `02_html_mail.ahd` | HTML ve multipart/alternative posta |
| `03_env_smtp.ahd` | Env ile host, port, güvenlik ve AUTH PLAIN |

`ahdsmtp` yardımcısı yoktur. TLS çalışma zamanı kütüphanelerini kullanır.
