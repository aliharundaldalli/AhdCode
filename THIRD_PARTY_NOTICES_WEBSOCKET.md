# AhdCode — Third-party notice for WebSocket server support

WebSocket endpoints registered through the `HTTP` module (and `Web`, which
passes through to it) use `github.com/coder/websocket` v1.8.15, a pure-Go
RFC 6455 implementation with no dependencies of its own. It is Copyright (c)
2025 Coder and is distributed under the ISC license; the upstream license is
available at
<https://github.com/coder/websocket/blob/v1.8.15/LICENSE.txt>.

It is embedded into AhdCode itself (see
`internal/backend/golang/ahdruntime/websocketvendor`) and copied verbatim into
the build workspace of a generated program that registers a WebSocket
endpoint, as `vendor/`, so that program builds with `go build -mod=vendor` and
never fetches the dependency over the network. Its LICENSE file travels with
the vendored source and remains present in that `vendor/` tree. A program that
registers no WebSocket endpoint does not receive this tree.

AhdCode performs its own origin validation, message and queue limits, and
connection lifecycle on top of the library; permessage-deflate compression is
never enabled.
