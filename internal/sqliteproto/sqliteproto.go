// Package sqliteproto defines the narrow JSON contract between the
// stdlib-only generated runtime (and the persistent evaluator) and the bundled
// ahdsqlite helper, which links the pure-Go SQLite engine.
//
// The helper serves one long-lived session over its standard streams: the
// client writes one Request per line and reads one Response per line. Every
// open Database is one logical SQLite connection inside the helper, addressed
// by the handle the open response returned.
package sqliteproto

// Kinds are the exact public SQLiteValue kinds. They mirror SQLite's runtime
// storage classes; BLOB is deliberately absent because v0.3.0 rejects it.
const (
	KindNull   = "Null"
	KindInt    = "Int"
	KindReal   = "Real"
	KindString = "String"
)

// Value is one SQLite storage-class value crossing the wire. Kind selects
// which payload field is meaningful.
type Value struct {
	Kind   string  `json:"kind"`
	Int    int64   `json:"int,omitempty"`
	Real   float64 `json:"real,omitempty"`
	String string  `json:"string,omitempty"`
}

// Operations the helper understands.
const (
	OperationOpen         = "open"
	OperationExecute      = "execute"
	OperationQuery        = "query"
	OperationLastInsertID = "lastInsertId"
	OperationBegin        = "begin"
	OperationCommit       = "commit"
	OperationRollback     = "rollback"
	OperationClose        = "close"
)

// Request is one client instruction.
type Request struct {
	Operation  string  `json:"operation"`
	Database   int64   `json:"database,omitempty"`
	Path       string  `json:"path,omitempty"`
	SQL        string  `json:"sql,omitempty"`
	Parameters []Value `json:"parameters,omitempty"`
}

// Response is the helper's answer to exactly one Request. A non-empty Error
// means the operation failed and every other field is meaningless.
type Response struct {
	Error    string    `json:"error,omitempty"`
	Database int64     `json:"database,omitempty"`
	Changed  int64     `json:"changed,omitempty"`
	RowID    int64     `json:"rowId,omitempty"`
	Columns  []string  `json:"columns,omitempty"`
	Rows     [][]Value `json:"rows,omitempty"`
}
