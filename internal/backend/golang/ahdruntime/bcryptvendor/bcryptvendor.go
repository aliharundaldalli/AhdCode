// Package bcryptvendor embeds the exact pinned source behind Security's bcrypt
// functions: the bcrypt and blowfish packages of golang.org/x/crypto v0.56.0,
// the same module version AhdCode's own go.mod already requires, so it adds no
// dependency. A generated program that calls Security.bcryptHash or
// Security.bcryptVerify builds it with `go build -mod=vendor`, so the build
// never fetches anything over the network or relies on the local module cache.
// The tree was produced once, at AhdCode development time, with `go mod
// vendor` in a throwaway module importing exactly golang.org/x/crypto/bcrypt --
// never regenerated on an end user's machine.
//
// See THIRD_PARTY_NOTICES_BCRYPT.md for the license of the vendored code.
package bcryptvendor

import "embed"

// Vendor holds the vendor/ tree verbatim: vendor/modules.txt plus the source
// of every vendored package, as `go mod vendor` produced it.
//
//go:embed vendor
var Vendor embed.FS

// Requires lists the go.mod require lines this tree needs.
var Requires = []string{"golang.org/x/crypto v0.56.0"}

// GoSum lists the module sums `go mod vendor` recorded alongside the tree.
const GoSum = `golang.org/x/crypto v0.56.0 h1:GUh5Ii4J5jtcseSMiRqr1jXCNHoxjeV9Fmekc2oLy6Y=
golang.org/x/crypto v0.56.0/go.mod h1:OMW5y6CY9l38uPLmxU6l6pwcXp1obtLo3e6gT7gQR2I=
`
