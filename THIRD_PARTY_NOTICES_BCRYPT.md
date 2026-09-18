# AhdCode — Third-party notice for Security's bcrypt functions

`Security.bcryptHash` and `Security.bcryptVerify` use the `bcrypt` and
`blowfish` packages of `golang.org/x/crypto` v0.56.0: pure Go, with no CGO,
no external process, and no network access. This is the same module version
AhdCode's own `go.mod` already requires (the evaluator's Argon2id comes from
it), so bcrypt adds no dependency. The code is Copyright 2009 The Go Authors
and is distributed under the BSD 3-Clause License; the upstream license is
available at <https://cs.opensource.google/go/x/crypto/+/refs/tags/v0.56.0:LICENSE>.

The two packages are embedded into AhdCode itself (see
`internal/backend/golang/ahdruntime/bcryptvendor`) and copied verbatim into a
generated program's build workspace as `vendor/` only when that program calls
a bcrypt function, so the program builds with `go build -mod=vendor` and never
fetches the dependency over the network. The LICENSE and PATENTS files travel
with the vendored source and remain present in that `vendor/` tree.

AhdCode does not implement bcrypt itself. New applications should prefer
Argon2id (`Security.passwordHash`); bcrypt is provided for compatibility and
migration.
