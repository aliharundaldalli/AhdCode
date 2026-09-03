# SMTP standard module

[English] · [Türkçe](SMTP_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Env](ENV.md) · [Student Guide](STUDENT_GUIDE_EN.md#41-sending-email-smtp)

`SMTP` is the compiler-registered `builtin:SMTP` module, introduced in
AhdCode v0.9.0. It is explicit and a sibling `SMTP.ahd` cannot shadow it:

```ahd
bring SMTP
from SMTP bring SMTPClient
from SMTP bring SMTPMessage
from SMTP bring SMTPError
```

`SMTP` is a send-only mail transport and message-composition primitive. It is
not a newsletter framework, not an inbox, not a mailbox client, not a
background queue, and not a provider-specific API. There is no IMAP, POP3,
attachment support, or mail helper executable. The implementation uses Go's
`net/smtp`, `crypto/tls`, and MIME libraries inside the AhdCode runtime.

## Public surface

```text
SMTP.client(
    host: String
    port: Int
    security: String := "starttls"
    timeoutSeconds: Int := 30
) -> SMTPClient

SMTP.message(
    from: String
    to: List<String>
    subject: String
) -> SMTPMessage

SMTPClient.withPlainAuth(username: String, password: String) -> SMTPClient
SMTPClient.send(message: SMTPMessage) -> Nothing

SMTPMessage.withCc(recipients: List<String>) -> SMTPMessage
SMTPMessage.withBcc(recipients: List<String>) -> SMTPMessage
SMTPMessage.withReplyTo(address: String) -> SMTPMessage
SMTPMessage.withText(body: String) -> SMTPMessage
SMTPMessage.withHtml(body: String) -> SMTPMessage
```

Failures use `SMTPError`. Direct construction of `SMTPClient` or `SMTPMessage`
is rejected.

## Client configuration

`SMTP.client` stores configuration. It does not connect. `withPlainAuth` also
does not connect. Network activity happens only on `client.send(message)`.

One send opens one SMTP connection, runs one transaction, then QUIT/closes.
There is no connection pool, persistent session, or hidden retry.

`SMTPClient` is immutable:

```ahd
base := SMTP.client("smtp.example.com", 587)
authenticated := base.withPlainAuth("user@example.com", password)
```

`base` remains unauthenticated.

`host` is a hostname or IP, never a URL such as `smtp://example.com`. `port`
must be in `1..65535`. `timeoutSeconds` must be in `1..9223372036` and covers
connect, TLS handshake, SMTP commands, and DATA.

## Security modes

Exact lowercase values only. User input is not silently lowercased, and
aliases such as `ssl`, `smtps`, `opportunistic`, or `auto` are rejected.

| Value | Meaning |
|---|---|
| `starttls` | Connect plaintext SMTP, require `STARTTLS`, upgrade, verify TLS |
| `tls` | Implicit TLS from connection start |
| `none` | Explicit plaintext SMTP |

If `security` is `starttls` and the server does not advertise STARTTLS, send
raises `SMTPError`. There is no opportunistic plaintext fallback.

Implicit TLS (`tls`) performs the TLS handshake before any SMTP commands.
Certificate chain and hostname/IP identity are verified against system
certificate roots. There is no public insecure-skip, trust-all, or
self-signed bypass. An untrusted certificate produces `SMTPError`.

`none` is intentionally explicit. It can be useful for a localhost
development fixture. Do not pretend it is secure.

## AUTH PLAIN

v0.9 supports only AUTH PLAIN, through `withPlainAuth`. There is no AUTH
LOGIN, CRAM-MD5, XOAUTH2, or provider OAuth.

AUTH is attempted only after an encrypted connection (`starttls` or `tls`).
If the client is authenticated and `security` is `none`, send raises
`SMTPError` before credentials are transmitted.

If the server does not advertise AUTH PLAIN, send raises `SMTPError`. There
is no fallback mechanism. Wrong credentials raise `SMTPError` with no retry.

Never hardcode a real SMTP password into source. Read it with [Env](ENV.md):

```ahd
bring Env
bring SMTP
from SMTP bring SMTPClient
from SMTP bring SMTPMessage

host := Env.getOr("SMTP_HOST", "127.0.0.1")
port := int(Env.getOr("SMTP_PORT", "2525"))
security := Env.getOr("SMTP_SECURITY", "starttls")
username := Env.getOr("SMTP_USERNAME", "")
password := Env.getOr("SMTP_PASSWORD", "")

client: SMTPClient := SMTP.client(host, port, security)
if username != "" {
    client = client.withPlainAuth(username, password)
}
```

Do not print the password. SMTP diagnostics never include it.

There is no `SMTP.fromEnv()`, `SMTP.gmail()`, or `SMTP.outlook()`.

## Messages

`SMTPMessage` is immutable. `withText` / `withHtml` / `withCc` / `withBcc` /
`withReplyTo` return new values.

Each `String` is exactly one mailbox. Use `user@example.com` or, when the
parser allows it, `Ali Daldallı <user@example.com>`. Do not put two recipients
in one list element.

Mailbox addr-spec must remain ASCII. Unicode display names may be encoded in
headers. A Unicode mailbox local-part raises `SMTPError`. SMTPUTF8 is not
implemented in v0.9.

`to` may be empty so Cc-only or Bcc-only delivery remains possible. At send
time there must be at least one recipient across To, Cc, and Bcc.

Envelope RCPT order is To, then Cc, then Bcc. Duplicate entries are preserved
as supplied; applications that do not want duplicate RCPT commands should not
supply duplicates.

`From` is exactly one mailbox and is used for both MAIL FROM and the From
header. The authenticated username is never inferred as From.

Bcc recipients are sent as envelope RCPT recipients. No Bcc header appears in
DATA.

`withReplyTo` sets one Reply-To mailbox. Repeating it replaces the previous
value. There is no automatic Reply-To.

Subject is UTF-8 and encoded with RFC 2047 encoded-words where required.
CR/LF in subject, From, To, Cc, Bcc, or Reply-To raises `SMTPError`. Header
injection does not reach DATA.

At least one body must be configured with `withText` or `withHtml`. An
explicit empty String counts. A message with neither raises `SMTPError`.

- text only: `text/plain; charset=utf-8`
- HTML only: `text/html; charset=utf-8`
- both: `multipart/alternative` with text/plain first, text/html second

UTF-8 bodies use quoted-printable transfer encoding. SMTP does **not**
escape or sanitize HTML. The supplied HTML is intentional mail markup.
Applications that handle user content must construct or sanitize that HTML
themselves. This does not go through `HTML.text`.

Date is generated automatically at send time. There is no public Date or
Message-ID API in v0.9. Message-ID is omitted rather than inventing a weak
identity scheme.

Attachments are out of scope.

## Command flow

connect → greeting → EHLO → TLS according to the security mode → AUTH if
configured → MAIL FROM → RCPT TO for To/Cc/Bcc → DATA → message → QUIT →
close.

A rejected SMTP command is `SMTPError`, not an HTTP-style Response. If any
RCPT recipient is rejected before DATA, the send aborts. There is no
partial-recipient delivery and no silent retry. Duplicate email delivery is
treated as a side effect to avoid.

Every send path closes the connection, including AUTH failure, RCPT
rejection, DATA failure, timeout, and TLS error.

Two concurrent `send` calls use independent connections. The client is
immutable configuration, not a shared session.

## Out of scope

IMAP, POP3, mailbox reading, attachments, AUTH LOGIN / CRAM-MD5 / XOAUTH2,
DKIM/SPF/DMARC, provider modules, mail queues, retries, templates, bounce
processing, and tracking are not part of v0.9.
