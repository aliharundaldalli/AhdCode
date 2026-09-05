# v0.16 naming-convention probe

Evidence that the `Page` suffix and identifier style are **conventions only**:
no compiler, router, or runtime code reads a handler's name.

## Probe source

```ahd
bring Web
bring HTTP
from Web bring (Request, Response, Server)
// camelCase handler
adminUserEdit: Function := (request: Request) -> Response {
    return Web.text("camelCase handler", 200)
}
// snake_case handler: ordinary identifier grammar, no special support
admin_user_edit: Function := (request: Request) -> Response {
    return Web.text("snake_case handler", 200)
}
// PascalCase-free, no Page suffix anywhere
profile: Function := (request: Request) -> Response {
    return Web.text("no Page suffix", 200)
}

server: Server := HTTP.server("127.0.0.1", 8199)
server.get("/admin/users/edit/*", adminUserEdit)
server.get("/admin/users/edit2/*", admin_user_edit)
server.get("/profile", profile)
server.start()
```

## Results

| Check | Result |
| --- | --- |
| `ahdcode build` on the probe | succeeds; all three handlers route |
| `ahdcode format` on the probe | rewrites blank lines only; identifiers byte-identical |
| `userEdit` and `useredit` in one file | two distinct bindings — identifiers are case-sensitive |
| `git diff -- '*.go'` for this cleanup | no compiler/router/runtime change |

The formatter performs no case folding, no collapsing, and no suffix
normalization. There is no filename-to-function lookup to probe, because none
exists: a route is handed a `Function` value by an explicit `require` chain.

## Renamed v0.16 example

`examples/v0.16/forms_validation` drops the redundant suffix
(`registerPage` -> `register`, `profilePage` -> `profile`; `registerSubmit`
keeps its explicit action name). PascalCase file names are unchanged, so word
boundaries still line up: `Pages/Register.ahd` -> `register`,
`Pages/Profile.ahd` -> `profile`.

Verification after the rename:

| Check | Result |
| --- | --- |
| `go test ./internal/build -run TestWebV016` | ok |
| `tools/qa/v016_forms.py` against the rebuilt fixture | PASS, 66 HTTP assertions |
| `ahdcode format --check` on all four renamed sources | canonical (exit 0) |
| Live example: GET `/register`, GET `/profile`, POST without CSRF | 200, 200, 403 |
| `go build ./... && go test ./...` | green |

`examples/v0.15/ahd_math_portal` keeps its historical `...Page` names and still
compiles unchanged, which is the source-compatibility check.
