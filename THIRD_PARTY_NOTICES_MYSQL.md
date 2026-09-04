# AhdCode — Third-party notice for the MySQL standard module

The MySQL module speaks the MySQL wire protocol directly using
`github.com/go-sql-driver/mysql` v1.10.1, a pure-Go client with no CGO and no
`mysql` client library dependency. It is distributed under the Mozilla
Public License 2.0; the upstream license is available at
<https://github.com/go-sql-driver/mysql/blob/v1.10.1/LICENSE>.

That driver depends on `filippo.io/edwards25519` v1.2.0, used for
`caching_sha2_password` authentication. It is Copyright © 2009 The Go
Authors and is distributed under a 3-clause BSD license; the upstream
license is available at
<https://github.com/FiloSottile/edwards25519/blob/v1.2.0/LICENSE>.

Both are embedded into AhdCode itself (see
`internal/backend/golang/ahdruntime/mysqlvendor`) and copied verbatim into a
generated MySQL program's build workspace as `vendor/`, so that program
builds with `go build -mod=vendor` and never fetches either dependency over
the network. Their LICENSE files travel with the vendored source and remain
present in that `vendor/` tree.

No `mysql` CLI, `mysqld`, or other external helper process is used or
required. The MySQL *server* the program connects to is, naturally, still an
external network service.
