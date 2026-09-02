# AhdCode — Third-party notice for the SQLite standard module

The bundled `ahdsqlite` helper uses `github.com/ncruces/go-sqlite3` v0.35.4, a
CGO-free Go wrapper around SQLite. It is Copyright © 2023 Nuno Cruces and is
distributed under the MIT License; the upstream license is available at
<https://github.com/ncruces/go-sqlite3/blob/v0.35.4/LICENSE>.

That wrapper depends on `github.com/ncruces/go-sqlite3-wasm/v5` v5.0.35304,
a machine translation (via `wasm2go`) of the SQLite C library and its
supporting libraries into Go. SQLite itself (version 3.53.4) is in the public
domain (<https://sqlite.org/copyright.html>); the translated supporting code
retains its original licenses, and the remainder of that repository is
distributed under the MIT No Attribution License
(<https://github.com/ncruces/go-sqlite3-wasm/blob/v5.0.35304/LICENSE>).

The engine is linked into the helper only, not into stdlib-only AhdCode
generated workspaces. No system `libsqlite3`, `sqlite3` executable, or CGO is
used.
