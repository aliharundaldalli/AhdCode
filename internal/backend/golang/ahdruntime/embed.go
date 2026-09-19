package ahdruntime

import _ "embed"

// Source is the verbatim runtime source emitted into every generated program.
// The generator rewrites only its package clause.
//
//go:embed ahdruntime.go
var Source string

// ExcelSource is emitted as a separate generated Go file. Keeping the XLSX
// implementation separate makes the direct OOXML layer reviewable while it
// remains part of the same standard-library-only, relocation-safe runtime.
//
//go:embed excel.go
var ExcelSource string

// PDFSource is emitted as a separate generated Go file, the same way
// ExcelSource is. It shares the Latex module's low-level renderer
// (ahdLatexCompile and friends, in ahdruntime.go) but keeps its own document
// model and LaTeX-body construction reviewable on their own.
//
//go:embed pdf.go
var PDFSource string

// ArchiveSource is emitted as a separate generated Go file, the same way
// ExcelSource is. It depends only on the Go standard library's archive/zip,
// archive/tar, and compress/gzip packages.
//
//go:embed archive.go
var ArchiveSource string

// SQLiteSource is emitted as a separate generated Go file, the same way
// ExcelSource is. It is the stdlib-only client of the bundled ahdsqlite
// helper; the SQLite engine itself never enters a generated workspace.
//
//go:embed sqlite.go
var SQLiteSource string

//go:embed http.go
var HTTPSource string

//go:embed html.go
var HTMLSource string

//go:embed smtp.go
var SMTPSource string

// MySQLSource is emitted as a separate generated Go file, the same way
// SMTPSource is. github.com/go-sql-driver/mysql is a pure-Go client of the
// MySQL wire protocol, so unlike SQLiteSource this needs no bundled helper
// executable: the generated program links the driver directly.
//
//go:embed mysql.go
var MySQLSource string

// SecuritySource is emitted as a separate generated Go file. It contains a
// self-contained Argon2id implementation built only from the Go standard
// library (crypto/rand, crypto/subtle, math/bits, encoding/base64), so the
// generated workspace remains dependency-free.
//
//go:embed security.go
var SecuritySource string

//go:embed identity.go
var IdentitySource string

// UUIDSource is emitted as a separate generated Go file. It generates, parses,
// and compares RFC 9562 UUIDs using only bytes, crypto/rand, sync, and time from
// the standard library.
//
//go:embed uuid.go
var UUIDSource string

// WebSocketSource is emitted as a separate generated Go file. It holds the
// standard-library-only part of WebSocket server support: endpoints, route
// registration, and the WebSocket members, so the HTTP dispatcher always
// compiles.
//
//go:embed websocket.go
var WebSocketSource string

// WebSocketConnSource is emitted only into a program that creates a WebSocket
// endpoint. It imports the vendored github.com/coder/websocket (see
// ahdruntime/websocketvendor), the same way MySQLSource imports its driver.
//
//go:embed websocket_conn.go
var WebSocketConnSource string

// PostgreSQLSource is emitted only into a program that uses PostgreSQL. It
// imports the vendored github.com/jackc/pgx/v5 graph (see
// ahdruntime/postgresqlvendor), the same way MySQLSource imports its driver.
//
//go:embed postgresql.go
var PostgreSQLSource string

// BitsSource is emitted as a separate generated Go file. It provides the
// bitwise operations AhdCode's grammar has no operators for, using only
// math/bits from the standard library.
//
//go:embed bits.go
var BitsSource string

// CharactersSource is emitted as a separate generated Go file. It provides the
// Unicode code-point operations of the Characters module using only unicode
// and unicode/utf8 from the standard library.
//
//go:embed characters.go
var CharactersSource string

// CronSource is emitted as a separate generated Go file. It provides the
// Cron schedule parser, occurrence search, and blocking scheduler loop using
// only strconv, strings, sync, and time from the standard library.
//
//go:embed cron.go
var CronSource string

// CodesSource is emitted as a separate generated Go file only into a program
// that uses QR or barcode encoding. Unlike the standard-library-only runtime
// files above, it imports the vendored github.com/boombuler/barcode encoder
// (see ahdruntime/codesvendor), the same way MySQLSource imports its driver.
//
//go:embed codes.go
var CodesSource string

// SVGSource is emitted as a separate generated Go file. It converts SVG
// document assets to vector PGF drawing commands using only encoding/xml and
// other standard packages, so the generated workspace stays dependency-free.
//
//go:embed svg.go
var SVGSource string

// DocumentSource is emitted as a separate generated Go file. It holds the
// professional-document builders Latex and PDF share: page layout, headers
// and footers, placement, links, bookmarks, metadata, and image transforms.
//
//go:embed document.go
var DocumentSource string

// TerminalSource is emitted as a separate generated Go file. It implements the
// Terminal standard module over the same buffered standard output write uses,
// using only the standard library.
//
//go:embed terminal.go
var TerminalSource string

// The Terminal platform files each carry a build constraint and are emitted
// under an operating-system suffix, so a generated workspace compiles exactly
// the one for its target. They use only the standard library's syscall
// package.
//
//go:embed terminal_darwin.go
var TerminalDarwinSource string

//go:embed terminal_linux.go
var TerminalLinuxSource string

//go:embed terminal_windows.go
var TerminalWindowsSource string

//go:embed terminal_other.go
var TerminalOtherSource string

// GraphicsSource is emitted as a separate generated Go file. It is the AhdCode
// side of the Graphics standard module: argument validation, color parsing,
// Turtle state and geometry, and the protocol client of the bundled
// ahdgraphics window helper. It uses only the standard library.
//
//go:embed graphics.go
var GraphicsSource string

// FundamentalsSource is emitted as a separate generated Go file. It holds the
// v1.7 standard-library fundamentals shared with the evaluator: the added Math
// functions, strict ISO 8601 Time text and instant arithmetic, Vector and
// Matrix accessors, two-list Statistics, and Table concat and innerJoin. It
// uses only the standard library.
//
//go:embed fundamentals.go
var FundamentalsSource string

// BcryptSource is emitted only into a program that calls Security.bcryptHash
// or Security.bcryptVerify. It imports the vendored golang.org/x/crypto/bcrypt
// (see ahdruntime/bcryptvendor), the same way MySQLSource imports its driver.
//
//go:embed bcrypt.go
var BcryptSource string

// HelperLinkSource is emitted as a separate generated Go file. It is the
// standard-library-only transport to the bundled window helpers (ahdgraphics
// and ahdgui): request/response lines, event routing, and the bounded event
// queue that Canvas and Window callbacks are dispatched from.
//
//go:embed helperlink.go
var HelperLinkSource string

// GUISource is emitted as a separate generated Go file. It is the AhdCode
// side of the GUI standard module: argument validation, widget handles and
// values, callbacks, and the protocol client of the bundled ahdgui window
// helper. It uses only the standard library.
//
//go:embed gui.go
var GUISource string
