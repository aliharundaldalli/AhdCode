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
