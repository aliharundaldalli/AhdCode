module ahdcode

go 1.26.4

// The local fork adds support for a simple CFF program's embedded custom
// encoding. It lets controlled Tectonic PDFs render their own Type-1C glyphs
// without any host-font substitution.
replace github.com/SalvioniDigitalSolutions/gopdf => ./third_party/gopdf

require (
	github.com/SalvioniDigitalSolutions/gopdf v0.0.0-20260819123034-adfb3bf86a60
	github.com/boombuler/barcode v1.1.0
	github.com/coder/websocket v1.8.15
	github.com/go-sql-driver/mysql v1.10.1
	github.com/jackc/pgx/v5 v5.11.0
	github.com/ncruces/go-sqlite3 v0.35.4
	golang.org/x/crypto v0.56.0
	golang.org/x/image v0.32.0
	golang.org/x/sys v0.47.0
	golang.org/x/term v0.45.0
	golang.org/x/text v0.41.0
	gonum.org/v1/gonum v0.16.0
	gonum.org/v1/plot v0.17.0
)

require (
	codeberg.org/go-fonts/liberation v0.5.0 // indirect
	codeberg.org/go-latex/latex v0.2.0 // indirect
	codeberg.org/go-pdf/fpdf v0.11.1 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	git.sr.ht/~sbinet/gg v0.7.0 // indirect
	github.com/ajstarks/svgo v0.0.0-20211024235047-1546f124cd8b // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/ncruces/go-sqlite3-wasm/v5 v5.0.35304 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/sync v0.22.0 // indirect
)
