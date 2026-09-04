// Package mysqlvendor embeds the exact pinned source of the MySQL module's
// one third-party dependency (github.com/go-sql-driver/mysql and its own
// dependency, filippo.io/edwards25519) so a generated MySQL program can be
// built with `go build -mod=vendor` and never fetch anything over the
// network or rely on the local machine's module cache being warm. This tree
// is produced once, at AhdCode development time, with `go mod vendor` in a
// throwaway module depending on the pinned driver version -- never
// regenerated on an end user's machine as part of an AhdCode build.
//
// See THIRD_PARTY_NOTICES_MYSQL.md for the licenses of the vendored code.
package mysqlvendor

import "embed"

// Vendor holds the vendor/ tree verbatim: vendor/modules.txt plus every
// vendored package's source, exactly as `go mod vendor` produced it for
// github.com/go-sql-driver/mysql v1.10.1 and filippo.io/edwards25519 v1.2.0.
//
//go:embed vendor
var Vendor embed.FS

// GoMod is the go.mod a generated program needs to build the vendored
// dependency graph in `-mod=vendor` mode: the same two require lines this
// repository's own go.mod carries for the identical pinned versions.
const GoMod = `module ahdcodeprogram

go 1.25

require github.com/go-sql-driver/mysql v1.10.1

require filippo.io/edwards25519 v1.2.0 // indirect
`

// GoSum is not required for a -mod=vendor build (vendor mode never consults
// it), but is included for the same reason `go mod vendor` writes one
// alongside vendor/: a generated workspace that looks like an ordinary
// vendored Go module if anyone inspects it by hand.
const GoSum = `filippo.io/edwards25519 v1.2.0 h1:crnVqOiS4jqYleHd9vaKZ+HKtHfllngJIiOpNpoJsjo=
filippo.io/edwards25519 v1.2.0/go.mod h1:xzAOLCNug/yB62zG1bQ8uziwrIqIuxhctzJT18Q77mc=
github.com/go-sql-driver/mysql v1.10.1 h1:arlSnNLq6a5yxGxV7qg9lF4j0C+KwD6NbQyKr9QL6ME=
github.com/go-sql-driver/mysql v1.10.1/go.mod h1:M+cqaI7+xxXGG9swrdeUIoPG3Y3KCkF0pZej+SK+nWk=
`
