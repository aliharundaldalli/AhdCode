package ahdruntime

import _ "embed"

// Source is the verbatim runtime source emitted into every generated program.
// The generator rewrites only its package clause.
//
//go:embed ahdruntime.go
var Source string
