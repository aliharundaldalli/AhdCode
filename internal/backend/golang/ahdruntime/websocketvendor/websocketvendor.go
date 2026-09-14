// Package websocketvendor embeds the exact pinned source of the one third-party
// dependency behind AhdCode's WebSocket server support:
// github.com/coder/websocket v1.8.15, which itself has no dependencies. A
// generated program that registers a WebSocket endpoint builds it with `go
// build -mod=vendor`, so the build never fetches anything over the network or
// relies on the local module cache. The tree was produced once, at AhdCode
// development time, with `go mod vendor` in a throwaway module importing
// exactly that package -- never regenerated on an end user's machine.
//
// See THIRD_PARTY_NOTICES_WEBSOCKET.md for the license of the vendored code.
package websocketvendor

import "embed"

// Vendor holds the vendor/ tree verbatim: vendor/modules.txt plus the source
// of every vendored package, as `go mod vendor` produced it.
//
//go:embed vendor
var Vendor embed.FS

// Requires lists the go.mod require lines this tree needs.
var Requires = []string{"github.com/coder/websocket v1.8.15"}

// GoSum lists the module sums `go mod vendor` recorded alongside the tree.
const GoSum = `github.com/coder/websocket v1.8.15 h1:6B2JPeOGlpff2Uz6vOEH1Vzpi0iUz20A+lPVhPHtNUA=
github.com/coder/websocket v1.8.15/go.mod h1:NX3SzP+inril6yawo5CQXx8+fk145lPDC6pumgx0mVg=
`
