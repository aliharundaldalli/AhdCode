# Mail with attachments

[English] · [Türkçe](README_TR.md)

This example needs a local SMTP server. It will not guess a real mailbox,
and it contains no credentials: everything it needs comes from the
environment.

Any local capture server works. With Python's built-in one:

```bash
python3 -m smtpd -c DebuggingServer -n localhost:2525
```

Then, in another terminal:

```bash
export SMTP_HOST=127.0.0.1
export SMTP_PORT=2525
export SMTP_SECURITY=none
export SMTP_FROM=sender@example.com
export SMTP_TO=student@example.com
ahdcode run send.ahd
```

The captured message is a `multipart/mixed`: a `multipart/alternative` with
the text and HTML bodies, then the two attachments, base64-encoded.

## Credentials

`send.ahd` reads `SMTP_USER` and `SMTP_PASSWORD` and authenticates only when
both are set. `SMTP_PASSWORD` is read with `Env.secret`, so a container
platform's secret file works as well as an environment variable — see
[Env](../../../docs/ENV.md).

AhdCode refuses to authenticate over an unencrypted connection, so a real
server needs `SMTP_SECURITY=starttls` or `tls`. Never put a password in a
source file.

## Generated files

`summary.txt` and `chart.png` are written by the program. They are generated;
they are not part of the example.
