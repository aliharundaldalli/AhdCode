# Distribution third-party notices

AhdCode is MIT licensed; the full license is included as `LICENSE`.

Private Go 1.27.0 is the unmodified official binary distribution under the
Go BSD-style license. Its `LICENSE`, `PATENTS`, and embedded third-party source
notices remain in `libexec/go`. Official origin: https://go.dev/dl/ .

Bootstrap 5.3.3 is MIT licensed. Its complete license is in
`licenses/bootstrap/LICENSE`, and remains embedded with the starter assets.
No npm or CDN runtime is included.

The exact Go module versions and complete license/notice texts are inventoried
in `licenses/modules.json` and `licenses/modules/`. They include Go's x packages
(BSD), go-sql-driver/mysql (MPL 2.0), edwards25519 (BSD), go-sqlite3 and its WASM
component (MIT), gonum/plot (BSD), font and PDF dependencies. The unchanged MySQL
source and license accompany this distribution under `licenses/mysql-source`;
see `THIRD_PARTY_NOTICES_MYSQL.md`. SQLite, numeric, and plot-specific notices
are included alongside this document. Source URLs/versions are preserved in
the module inventory and notices. No database server is redistributed.

The unmodified Tectonic 0.17.0 engine and the minimal offline resource bundle
live in `libexec/ahdcode/latex`. That directory carries its complete staged
notices and license texts. Tectonic is MIT; TeX macro packages include LPPL,
fonts include GUST and Computer Modern/AMS terms, and data files carry their
respective Unicode and hyphenation notices. The exact pinned resources are
listed in `licenses/latex-resources.json`. Engine binaries are checked against
https://github.com/tectonic-typesetting/tectonic/releases/tag/tectonic%400.17.0 .
The bundle is 5,546,077 bytes and does not fetch resources during execution.

The Windows installer is AhdCode-owned Go code and includes Go/x/sys notices.
No NSIS, commercial installer runtime, or third-party installer plug-in is used.
