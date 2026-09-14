# AhdCode — Third-party notice for the PostgreSQL standard module

The PostgreSQL module speaks the PostgreSQL wire protocol directly using
`github.com/jackc/pgx/v5` v5.11.0, a pure-Go client with no CGO and no `libpq`
dependency. It is Copyright (c) 2013-2021 Jack Christensen and is distributed
under the MIT license; the upstream license is available at
<https://github.com/jackc/pgx/blob/v5.11.0/LICENSE>.

The pgx packages the module uses depend on:

| Module | Version | Copyright | License |
| --- | --- | --- | --- |
| `github.com/jackc/pgpassfile` | v1.0.0 | Copyright (c) 2019 Jack Christensen | MIT |
| `github.com/jackc/pgservicefile` | v0.0.0-20240606120523-5a60cdf6a761 | Copyright (c) 2020 Jack Christensen | MIT |
| `github.com/jackc/puddle/v2` | v2.2.2 | Copyright (c) 2018 Jack Christensen | MIT |
| `golang.org/x/sync` | v0.22.0 | Copyright 2009 The Go Authors | 3-clause BSD |
| `golang.org/x/text` | v0.41.0 | Copyright 2009 The Go Authors | 3-clause BSD |

AhdCode never reads `~/.pgpass`, service files, or `PG*` environment
variables for a connection, even though the first two modules are part of
pgx's dependency graph.

All of these are embedded into AhdCode itself (see
`internal/backend/golang/ahdruntime/postgresqlvendor`) and copied verbatim into
a generated PostgreSQL program's build workspace as `vendor/`, so that program
builds with `go build -mod=vendor` and never fetches a dependency over the
network. Their LICENSE files travel with the vendored source and remain present
in that `vendor/` tree.

No `psql` CLI, `libpq`, or other external helper process is used or required.
The PostgreSQL *server* the program connects to is, naturally, still an
external network service.
