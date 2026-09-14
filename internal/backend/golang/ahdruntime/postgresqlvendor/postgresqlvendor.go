// Package postgresqlvendor embeds the exact pinned source of the PostgreSQL
// module's third-party dependency graph: github.com/jackc/pgx/v5 v5.11.0 and
// the modules its pgx, pgconn, pgtype, and pgxpool packages import
// (jackc/pgpassfile, jackc/pgservicefile, jackc/puddle/v2, golang.org/x/sync,
// and golang.org/x/text, pinned to the versions this repository itself
// resolves, so a compiled program and the interactive evaluator link the same
// code). A generated PostgreSQL program builds it with `go build
// -mod=vendor`, so the build never fetches anything over the network or relies
// on the local module cache. The tree was produced once, at AhdCode development
// time, with `go mod vendor` in a throwaway module importing exactly those
// packages -- never regenerated on an end user's machine.
//
// See THIRD_PARTY_NOTICES_POSTGRESQL.md for the licenses of the vendored code.
package postgresqlvendor

import "embed"

// Vendor holds the vendor/ tree verbatim: vendor/modules.txt plus the source
// of every vendored package, as `go mod vendor` produced it.
//
//go:embed vendor
var Vendor embed.FS

// Requires lists the go.mod require lines this tree needs.
var Requires = []string{
	"github.com/jackc/pgpassfile v1.0.0 // indirect",
	"github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect",
	"github.com/jackc/pgx/v5 v5.11.0",
	"github.com/jackc/puddle/v2 v2.2.2 // indirect",
	"golang.org/x/sync v0.22.0 // indirect",
	"golang.org/x/text v0.41.0 // indirect",
}

// GoSum lists the module sums `go mod vendor` recorded alongside the tree.
const GoSum = `github.com/jackc/pgpassfile v1.0.0 h1:/6Hmqy13Ss2zCq62VdNG8tM1wchn8zjSGOBJ6icpsIM=
github.com/jackc/pgpassfile v1.0.0/go.mod h1:CEx0iS5ambNFdcRtxPj5JhEz+xB6uRky5eyVu/W2HEg=
github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 h1:iCEnooe7UlwOQYpKFhBabPMi4aNAfoODPEFNiAnClxo=
github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761/go.mod h1:5TJZWKEWniPve33vlWYSoGYefn3gLQRzjfDlhSJ9ZKM=
github.com/jackc/pgx/v5 v5.11.0 h1:IzBBtyK9AHqf98cctWFifYSci2hgQR/cd56wB4p+ogg=
github.com/jackc/pgx/v5 v5.11.0/go.mod h1:mal1tBGAFfLHvZzaYh77YS/eC6IX9OWbRV1QIIM0Jn4=
github.com/jackc/puddle/v2 v2.2.2 h1:PR8nw+E/1w0GLuRFSmiioY6UooMp6KJv0/61nB7icHo=
github.com/jackc/puddle/v2 v2.2.2/go.mod h1:vriiEXHvEE654aYKXXjOvZM39qJ0q+azkZFrfEOc3H4=
golang.org/x/sync v0.22.0 h1:SZjpbeLmrCk4xhRSZFNZW5gFUeCeFgjekvI/+gfScek=
golang.org/x/sync v0.22.0/go.mod h1:9xrNwdLfx4jkKbNva9FpL6vEN7evnE43NNNJQ2LF3+0=
golang.org/x/text v0.41.0 h1:vz/seA0lnX87Othu2f/0L24RcgrXD9/YFTSuGjj3rH8=
golang.org/x/text v0.41.0/go.mod h1:jvf1O8ajNzZqhSrQBPbutR/EB83Cc0CFrezNQIwbb5M=
`
