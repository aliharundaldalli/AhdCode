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
