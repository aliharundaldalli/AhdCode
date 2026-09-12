// Package codesvendor embeds the exact pinned source of the one third-party
// dependency behind AhdCode's machine-readable codes: the qr, code128, and ean
// packages of github.com/boombuler/barcode v1.1.0 and the utils package they
// share. A generated program that uses QR, Barcode, Latex.qr, Latex.barcode,
// PDFDocument.qr, or PDFDocument.barcode builds it with `go build
// -mod=vendor`, so the build never fetches anything over the network or
// relies on the local module cache. The tree was produced once, at AhdCode
// development time, with `go mod vendor` in a throwaway module importing
// exactly those packages -- never regenerated on an end user's machine.
//
// See THIRD_PARTY_NOTICES_CODES.md for the license of the vendored code.
package codesvendor

import "embed"

// Vendor holds the vendor/ tree verbatim: vendor/modules.txt plus the source
// of every vendored package, as `go mod vendor` produced it.
//
//go:embed vendor
var Vendor embed.FS

// Requires lists the go.mod require lines this tree needs.
var Requires = []string{"github.com/boombuler/barcode v1.1.0"}

// GoSum lists the module sums `go mod vendor` recorded alongside the tree.
const GoSum = `github.com/boombuler/barcode v1.1.0 h1:ChaYjBR63fr4LFyGn8E8nt7dBSt3MiU3zMOZqFvVkHo=
github.com/boombuler/barcode v1.1.0/go.mod h1:paBWMcWSl3LHKBqUq+rly7CNSldXjb2rDl3JlRe0mD8=
`
