# WebSocket

[English] · [Türkçe](WEBSOCKET_TR.md)

[Back to README](../README.md) · [HTTP](HTTP.md) · [Web](WEB.md) · [Security](SECURITY.md) · [Env](ENV.md)

AhdCode v1.4.0 adds WebSocket **server** endpoints to the [HTTP](HTTP.md)
module, with a small pass-through in [Web](WEB.md). An endpoint lives on the
same `Server` as your routes, shares its handler mutex and lifecycle, and
exchanges text messages with browsers and other clients.

v2.1 adds the other end: a synchronous WebSocket **client**, so an AhdCode
program can connect to a service instead of only serving one. It is a
separate pair of types and changes nothing about the server; see
[The client](#the-client).

The protocol is implemented by `github.com/coder/websocket` v1.8.15, vendored
into AhdCode. Only a program that creates an endpoint or a client receives
it, and the build stays offline either way.

## Public surface

```text
HTTP.websocket(onMessage: Function(WebSocket, String) -> Nothing) -> WebSocketEndpoint

WebSocketEndpoint.withOpen(handler: Function(WebSocket, Request) -> Nothing)      -> WebSocketEndpoint
WebSocketEndpoint.withClose(handler: Function(WebSocket, Int, String) -> Nothing) -> WebSocketEndpoint
WebSocketEndpoint.withAccept(check: Function(Request) -> Response?)               -> WebSocketEndpoint
WebSocketEndpoint.withAllowedOrigins(origins: List<String>)                       -> WebSocketEndpoint
WebSocketEndpoint.withMaxMessageBytes(bytes: Int)                                 -> WebSocketEndpoint
WebSocketEndpoint.withMaxQueuedMessages(count: Int)                               -> WebSocketEndpoint
WebSocketEndpoint.withMaxConnections(count: Int)                                  -> WebSocketEndpoint

Server.websocket(path: String, endpoint: WebSocketEndpoint) -> Nothing

WebSocket.id()                                           -> String
WebSocket.send(text: String)                             -> Bool
WebSocket.close(code: Int := 1000, reason: String := "") -> Nothing
WebSocket.isOpen()                                       -> Bool

Web.websocket(onMessage: Function(WebSocket, String) -> Nothing) -> WebSocketEndpoint
App.websocket(path: String, endpoint: WebSocketEndpoint)         -> Nothing
```

The client (v2.1):

```text
HTTP.webSocketClient(url: String) -> WebSocketClient

WebSocketClient.withHeader(name: String, value: String) -> WebSocketClient
WebSocketClient.withTimeout(seconds: Int)               -> WebSocketClient
WebSocketClient.withMaxMessageBytes(bytes: Int)         -> WebSocketClient
WebSocketClient.connect()                               -> WebSocketConnection

WebSocketConnection.send(text: String)                             -> Bool
WebSocketConnection.receive(timeoutSeconds: Int := 0)              -> String?
WebSocketConnection.close(code: Int := 1000, reason: String := "") -> Nothing
WebSocketConnection.isOpen()                                       -> Bool
WebSocketConnection.closeCode()                                    -> Int?
WebSocketConnection.closeReason()                                  -> String
```

`Web` re-exports `WebSocket` and `WebSocketEndpoint`. A `WebSocketEndpoint`
is immutable configuration, like `Cookie`: each `with…` returns a new one, and
it holds no live connections. `WebSocket.id()` is an opaque identifier that is
unique in the process and never reused.

## A first endpoint

```ahd
bring HTTP
bring KeyValue
from HTTP bring (Server, Request, WebSocket, WebSocketEndpoint)

clients: Pair<String, WebSocket> := {}

joined: Function := (socket: WebSocket, request: Request) -> Nothing {
    clients: Global Pair<String, WebSocket>
    clients[socket.id()] = socket
}

received: Function := (socket: WebSocket, text: String) -> Nothing {
    socket.send("echo: " + text)
}

left: Function := (socket: WebSocket, code: Int, reason: String) -> Nothing {
    clients: Global Pair<String, WebSocket>
    clients = KeyValue.without(clients, socket.id())
}

endpoint: WebSocketEndpoint := HTTP.websocket(received)
endpoint = endpoint.withOpen(joined)
endpoint = endpoint.withClose(left)

server: Server := HTTP.server("127.0.0.1", 8172)
server.websocket("/live", endpoint)
server.start()
```

Register endpoints before `start()`. A path cannot be both a WebSocket endpoint
and a `GET` route; either mistake raises `HTTPError`. A non-`GET` request to an
endpoint path is answered `405` with `Allow`.

## Lifecycle

```mermaid
sequenceDiagram
    participant C as Client
    participant S as AhdCode server
    participant A as Your callbacks
    C->>S: GET /live (Upgrade: websocket)
    S->>S: route, upgrade headers, connection limit, origin
    S->>A: withAccept check(request)
    A-->>S: null accepts, or a Response refuses
    S->>A: onOpen(socket, request)
    S-->>C: 101 Switching Protocols
    S-->>C: messages sent during onOpen
    loop each message
        C->>S: text message
        S->>A: onMessage(socket, text)
    end
    C->>S: close, or the connection is lost
    S->>A: onClose(socket, code, reason)
```

The opening handshake runs its checks in this order:

1. The path matches an endpoint, with the same rules as HTTP routes.
2. Upgrade headers. A request that is not an upgrade attempt gets
   `426 Upgrade Required`; a malformed one gets `400`.
3. The connection limit. When it is reached: `503`.
4. The origin policy. When it fails: `403`.
5. The `withAccept` check, if any. A returned `Response` is sent as it is and
   no upgrade happens; `null` continues. An error raised in the check is a
   `500` and is logged like a handler failure.
6. `onOpen(socket, request)`. It runs **before** the upgrade response, so by
   the time the client sees the connection open, your `onOpen` has returned
   and the socket is in your registry. Messages sent from `onOpen` are
   delivered first.
7. `101 Switching Protocols`. If the upgrade fails after `onOpen`,
   `onClose(socket, 1006, "")` still runs.

For each socket, `onOpen` runs exactly once and first, `onMessage` runs once
per message in arrival order, and `onClose` runs exactly once and last. A
refused handshake runs none of them.

## Callbacks run one at a time

Every WebSocket callback holds the **same** per-server mutex as HTTP
handlers. No two handlers or callbacks on one `Server` ever run at the same
time. That is why the registry above can be an ordinary `Pair` with no
locking.

It also means a slow callback delays every request and every other socket,
exactly like a slow handler. Keep callbacks short. A database query in a
callback is fine; waiting on something slow is not.

The next message from a client is read only after `onMessage` for the
previous one has returned, so a slow application slows that client down
instead of growing a hidden queue.

## Sending and broadcasting

`send` never blocks. It puts the message on a bounded queue for that socket
and returns `true`, or returns `false` when the socket is closed or closing.
When the queue is full, the socket is closed with `1008` and the reason
`outbound queue full`, and `send` returns `false`: a client that cannot keep
up is disconnected rather than allowed to grow memory.

Broadcasting is application code over your own registry:

```ahd
broadcast: Function := (text: String) -> Int {
    clients: Global Pair<String, WebSocket>
    delivered: Local Int := 0
    for socket in KeyValue.values(clients) {
        if socket.send(text) {
            delivered += 1
        }
    }
    return delivered
}
```

An HTTP handler may call `broadcast`, because it holds the same mutex.

## Messages

- Only text messages are supported. A binary message closes the socket with
  `1003`.
- Text must be valid UTF-8; otherwise the socket closes with `1007`.
- A message larger than `maxMessageBytes` closes the socket with `1009`.
  Fragmented messages are reassembled up to that limit.
- Message contents are never logged.

To send structured data, build it with [JSON](JSON.md) and send the String.

## Closing

`close(code, reason)` accepts `1000`, `1001`, `1008`, `1011`, and
`3000..4999`, with a reason of at most 123 bytes of UTF-8; anything else raises
`HTTPError`. Closing a socket that is already closed or closing does nothing.
Messages already queued are sent before the close frame. `close` inside a
callback does not call `onClose` from within that callback; `onClose` runs
after the current callback returns. The close handshake waits at most five
seconds.

`onClose` receives the code and reason of the side that closed first. A lost
connection, a ping timeout, or a stalled write reports `1006` with an empty
reason; `1006` is never sent on the wire. After `onClose` starts, `isOpen()` is
`false` and `send` returns `false`.

An error raised in `onOpen` or `onMessage` is written to stderr, never to the
client; the socket is closed with `1011` and `onClose` still runs. An error in
`onClose` is only logged. The server keeps serving. A disconnect is not an
`HTTPError`.

Ending the process — Ctrl+C, `ahdcode kill`, or a `dev` rebuild — drops
connections without close frames. There is no automatic reconnection on
either side: a browser page reconnects only if its own script does.

## Keeping connections alive

The server pings every connection every 30 seconds. A client that does not
answer within 30 seconds is closed as `1006`. Client pings are answered.
Each outgoing message has a 10-second write deadline. The HTTP server's own
read and write timeouts do not apply to an upgraded connection, so a socket
stays open as long as the client does.

## Security

### Origins

Browsers send an `Origin` header, and a page on another site can try to open
your endpoint with the user's cookies. The default policy is same-origin:

- No `Origin` header is allowed; that is a non-browser client.
- Otherwise the `Origin` host, including a non-default port, must equal the
  request's `Host`, ignoring case.

`withAllowedOrigins(["https://app.example.com"])` replaces the default with an
exact list of `scheme://host[:port]` values, compared after lowercasing. There
are no wildcards and no paths; `"*"`, an empty list, or a malformed origin
raises `HTTPError` when you configure it.

### Authentication

Authenticate before the upgrade with `withAccept`:

```ahd
accept: Function := (request: Request) -> Response? {
    session: Local Session := sessions.open(request)
    if session.get("user_id") == null {
        return HTTP.text("sign in first", 401)
    }
    return null
}
endpoint = endpoint.withAccept(accept)
```

Sessions can be read in `withAccept` and `onOpen`. There is no response to
commit there, so changing the session in those callbacks is not supported.
For machine-to-machine use, a bearer token checked with
`request.header("Authorization")`, [`Env.secret`](ENV.md), and
[`Security.secureEqual`](SECURITY.md) works the same way.

### Limits

| Setting | Default | Range | When exceeded |
| --- | --- | --- | --- |
| `withMaxMessageBytes` | 65536 | `1..16777216` | close `1009` |
| `withMaxQueuedMessages` | 64 | `1..4096` | close `1008`, `outbound queue full` |
| `withMaxConnections` | 1024 | `1..1000000` | handshake `503` |

A value outside its range raises `HTTPError`. Compression is disabled.

## Using Web

```ahd
bring Web
from Web bring (App, WebSocket)

echo: Function := (socket: WebSocket, text: String) -> Nothing {
    socket.send(text)
}

site: App := Web.start()
site.websocket("/live", Web.websocket(echo))
site.start()
```

`App.websocket` registers the endpoint on the application's server, under the
same configuration and limits as its routes. Endpoints cannot be registered
on a `RouteSet` or `RouteGroup`: guards and `RequestContext.respond` produce an
HTTP response, which an upgraded connection cannot use. Put the same policy
in `withAccept` instead. The
[realtime attendance example](../examples/v1.4/realtime_attendance/README.md)
shows a complete application.

## Behind a reverse proxy

A reverse proxy must pass the upgrade through and should keep the original
`Host`, which the same-origin check compares against.

nginx:

```nginx
location /live {
    proxy_pass http://127.0.0.1:8140;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_read_timeout 120s;
}
```

Caddy's `reverse_proxy` passes WebSocket upgrades through without extra
settings. Keep any proxy idle timeout above the 30-second ping interval. When
the proxy terminates TLS, browsers connect with `wss://` to the proxy.

## The client

v2.1 lets an AhdCode program be the other end of a WebSocket connection.

The client is deliberately **not** the server's `WebSocket` value. A
`WebSocketClient` is immutable configuration with no socket behind it, and a
`WebSocketConnection` is one live connection that `connect()` produced.
Keeping them apart means a program can never call `receive` on something
that was never dialled, and the compiler says so.

```ahd
bring HTTP
from HTTP bring (WebSocketClient, WebSocketConnection, HTTPError)

socket: WebSocketClient := HTTP.webSocketClient("ws://127.0.0.1:8138/live")
authorized: WebSocketClient := socket.withHeader("Authorization", "Bearer token")
live: WebSocketConnection := authorized.withTimeout(5).connect()

if live.send("merhaba") {
    reply: String? := live.receive(5)
    if reply != null {
        write(reply)
    }
}
live.close()
```

### Connecting

`HTTP.webSocketClient(url)` performs no network activity: it validates the
URL and stores the configuration. Each `with…` returns a new
`WebSocketClient`, exactly as `WebSocketEndpoint` does. Only `connect()`
opens a connection, and it raises `HTTPError` when it cannot.

The URL must be `ws://` or `wss://`. `http://` and `https://` are refused
rather than quietly rewritten, so a program says which protocol it means, as
are `file:`, fragments, userinfo, and malformed URLs.

`withTimeout(seconds)` bounds the opening handshake, and each later `send`.
The default is 30 seconds. `withMaxMessageBytes(bytes)` bounds one incoming
message; the default is `65536` and the range `1..16777216`, the same as the
server endpoint's, so both ends of one connection are configured the same
way.

`withHeader` sets a handshake header such as `Authorization`. Setting a
header that is already present replaces it, exactly as
`ClientRequest.withHeader` does, so repeating a call never sends it twice.
Headers the protocol owns — `Connection`, `Upgrade`, `Host`,
`Content-Length`, and every `Sec-WebSocket-*` — are refused: setting one
would either be ignored or break the upgrade. CR and LF in a header value
are refused as everywhere else.

### Receiving

`receive` is synchronous. There is no callback, no background event bus, and
no async API: a program asks for the next message and waits.

```text
receive()                  waits until a message arrives or the peer closes
receive(timeoutSeconds)    waits at most that many seconds
```

- a text message returns that `String`
- a normal close by the peer returns `null`; `closeCode()` and
  `closeReason()` then describe it
- a lost connection, a protocol problem, or an **expired timeout** raises
  `HTTPError`

A timeout raises rather than returning `null`, so a program can always tell
"nothing arrived yet" from "the peer closed". The connection stays open
after a timeout and a later `receive` still picks the message up.

Messages arrive in the order the peer sent them. The next frame is read only
after the previous message was taken, so a slow program slows its peer
instead of growing a queue.

### Sending and closing

`send(text)` sends one complete UTF-8 text message. It returns `false` when
the connection is already closed or closing, and raises `HTTPError` when the
write itself fails — a lost connection is never reported as a delivered
message. There is no hidden retry and no automatic reconnect: reconnecting
is the program's decision, written in the program.

`close(code, reason)` follows the server's close-code policy: `1000`,
`1001`, `1008`, `1011`, or `3000..4999`, with a reason of at most 123 bytes
of UTF-8. Closing an already closed connection does nothing.

`closeCode()` is `null` while the connection is open, then the code that
ended it: the one this program sent when it closed, the peer's otherwise, or
`1006` when the connection was lost. `closeReason()` is the matching reason,
or `""`.

### Text only

The client is text-only, like the server. An incoming binary message closes
the connection with `1003` and raises `HTTPError`; an incoming message past
`maxMessageBytes` closes with `1009` and raises; a text message that is not
valid UTF-8 closes with `1007` and raises. `send` accepts only a `String`.
There is no `Bytes` type and no binary frame API.

### TLS

`wss://` verifies the certificate and the host name against the system
roots, exactly as the HTTP client's `https://` does. There is no
`insecureSkipVerify`, custom CA, or client certificate API, and an
untrusted, self-signed, or expired certificate raises `HTTPError`.

A failure never names a handshake header: the message says what stage
failed, not what the request carried, so an `Authorization` value cannot
reach a log through it. WebSocket has no HTTP redirect policy, so no
credential is ever forwarded to another host.

### Ping, pong, and keepalive

Control frames are handled inside the vendored library. There is no public
ping, pong, or heartbeat API on the client, and none is needed.

## Non-goals

Still absent, on both ends: binary messages, subprotocols, compression,
endpoints on `RouteSet` / `RouteGroup`, a graceful `1001` on shutdown,
automatic reconnection, and a publish/subscribe or event bus layer. The
client adds no callbacks, no background event loop, and no connection pool.

See also: [`examples/v0.1/72_websocket_echo.ahd`](../examples/v0.1/72_websocket_echo.ahd) ·
[`examples/v1.4/realtime_attendance`](../examples/v1.4/realtime_attendance/README.md) ·
[`examples/v2.1/websocket_client`](../examples/v2.1/websocket_client/) ·
[HTTP](HTTP.md) · [Web](WEB.md).
