# Application I/O and network (v2.1.0)

[English] · [Türkçe](README_TR.md)

Three self-contained programs for what v2.1 adds: moving binary files over
[HTTP](../../docs/HTTP.md) in both directions, talking to a
[WebSocket](../../docs/WEBSOCKET.md) service from AhdCode, and sending
[mail](../../docs/SMTP.md) with files attached.

Nothing here needs the public internet. Every example runs against a service
on `127.0.0.1` that you start yourself.

This example set is shipped with the AhdCode v2.1.0 release.

## http_file_transfer

Two programs. Start the service first:

```bash
cd http_file_transfer
ahdcode run server.ahd
```

and in another terminal:

```bash
cd http_file_transfer
ahdcode run client.ahd
```

`client.ahd` makes a QR code PNG, uploads it as a `multipart/form-data` file
part with `withMultipartFile`, and downloads the stored file back with
`download`. The PNG never becomes an AhdCode String in either direction: the
upload is streamed from disk into the request, and the response body is
streamed straight into `returned.png`.

`server.ahd` is the v0.8 upload API and the v0.9 file response, unchanged. It
writes what it receives into `received/`.

Stop the service with `ahdcode kill server.run`. `qr.png`, `returned.png`,
and `received/` are generated; they are not part of the example.

## websocket_client

Two programs again:

```bash
cd websocket_client
ahdcode run server.ahd
```

```bash
cd websocket_client
ahdcode run client.ahd
```

`server.ahd` is the v1.4 WebSocket endpoint: it requires a token in the
handshake, greets each connection, and answers every message with JSON.

`client.ahd` is new. It configures a `WebSocketClient` with that token,
connects, sends three questions, reads each answer with a synchronous
`receive`, asks the server to close, and reports the close code and reason.
No browser is involved, and nothing reconnects by itself.

## smtp_attachment

```bash
cd smtp_attachment
ahdcode run send.ahd
```

Sends one message with a text body, an HTML body, and two attachments — a
text file and a PNG. Read [its README](smtp_attachment/README.md) first: it
needs a local SMTP server, and it deliberately contains no credentials.
