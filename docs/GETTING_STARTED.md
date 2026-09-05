# Getting started

[English] · [Türkçe](GETTING_STARTED_TR.md)

[Back to README](../README.md) · [Language tour](LANGUAGE_TOUR.md) · [CLI](CLI.md)

## Install the compiler

AhdCode currently builds with Go 1.25 or newer.

```bash
cd AhdCode
go test ./...
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot ./cmd/ahdsqlite
```

If you plan to use the `Latex` module or the `PDF` module's `.save()` (they
share one offline renderer), you must also stage the offline Latex/Tectonic
runtime bundle. `Archive` needs no such staging. `SQLite` uses the bundled
`ahdsqlite` helper installed above; it does not need a system `sqlite3`. This
step performs a
one-time network fetch for pinned resources:

```bash
go run ./tooling/latex/cmd/package-latex --output "$(go env GOPATH)"
```

After staging, ordinary AhdCode Latex execution remains offline.

Ensure Go's binary directory is on `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

Confirm the installation:

```bash
ahdcode --version
```

## Your first program

Create `hello.ahd`:

```ahd
name := "AhdCode"
write("Hello {name}")
```

The compiler infers `String`; the binding is still statically typed. Write an
explicit annotation (`name: String := ...`) whenever it communicates intent or
inference is insufficient.

For a short reusable operation, an expression-only lambda creates a value of
the existing `Function` type:

```ahd
square := lambda (value: Int) -> value^2
write(square(5))
```

Lambda parameters require explicit types; the return type is inferred from
the single expression. Use a normal Function for a block or multiple steps.

Run it:

```bash
ahdcode run hello.ahd
```

Build a native executable:

```bash
ahdcode build hello.ahd -o hello
./hello
```

## Start a Web application

Initialize the current directory. No project name, package manager, or
network fetch:

```bash
mkdir my-app
cd my-app
ahdcode init web
ahdcode dev app.ahd
```

On a TTY the command asks Empty, Basic, or Admin, then the application name.
`ahdcode init web empty` and `ahdcode init web basic` skip the first question.
Admin continues with SQLite or MySQL and the administrator account.

Empty and Basic do not ask for a database and do not generate login or
`database/`. Admin initializes the schema and administrator immediately:
after `ahdcode dev app.ahd` the app is ready. Logout is POST `/logout` and
redirects to `/`.

`.env` is gitignored. Admin SQLite ignores `database/*.db` and keeps
`database/schema.sql` trackable. `.env.example` never contains entered
passwords. Existing files and existing databases are never overwritten;
there is no `--force`.

Open `http://127.0.0.1:8080`. Bootstrap 5.3.3 is local. `main.js` is an
ordinary static file, not a frontend runtime. There is no npm or CDN.
v0.17 route, guard, form, CSRF, and flash APIs are unchanged. This is a
pre-1.0 behavior change: bare `ahdcode init web` is now a wizard instead of
immediate generation.

## Input

`take` reads one line. It returns text, so numeric input uses an explicit
conversion:

```ahd
name := take("Name: ")
age := int(take("Age: "))

write("{name} is {age}")
```

## Format source

```bash
ahdcode format hello.ahd
ahdcode format --check hello.ahd
```

The first command updates the file atomically. The second only checks whether
the file is already canonical.

## Build a web application

```ahd
bring Web

home: Function := (request: Request) -> Response {
    return Web.html(Web.UI.h1("Merhaba"))
}
```

`bring Web` is the first-party web framework: routing, responses, and a
semantic HTML component layer in one import, resolved offline with no package
manager. See the [Web guide](WEB.md) and the runnable
[Ahd Akademi example](../examples/v0.15/ahd_academi).

Next: read the [language tour](LANGUAGE_TOUR.md), learn how to act on
[diagnostics](DIAGNOSTICS.md), build a [web application](WEB.md), or run the
[curated examples](../examples/v0.1/README.md), including UTC Time, CSV,
[Data tables](DATA.md), [PDF](PDF.md) generation, and
[Archive](ARCHIVE.md) packaging.

## v0.16: a complete form workflow

The [forms example](../examples/v0.16/forms_validation/README.md) runs with
`ahdcode run examples/v0.16/forms_validation/app.ahd` and needs no database.
Visit `http://127.0.0.1:8160/register`. GET creates a request context and renders
a hidden CSRF field. POST explicitly verifies CSRF and collects validation
errors. Invalid input re-renders only the selected name/email values through
Web.UI escaping; password and confirmation stay empty. Valid input sets a flash
message and redirects to `/profile`; its handler takes the message and commits
the removal, so refreshing shows no message.

Use one `Web.context(request, store)` per request and return
`context.respond(response)` on every response path. A duplicate finalization
raises `WebContextError`. `context.session` remains the ordinary Session value,
so application login and guards remain explicit. `Web.form(request)` is also
available without a session. `Form.integer` separates missing null from invalid
`FormValueError`; `Form.optional` separates missing from empty input.
`Web.errors()` supports required, length, matches, email shape, allowed values,
strict hex color, and custom field errors in deterministic order.

Choose `form.old(["name", "email"])` deliberately; never select passwords,
reset verifiers, or other secrets. There is no automatic old-input persistence,
flash rendering, middleware, or auth framework. The [Web guide](WEB.md) teaches
the complete flow and exact API; the v0.15 APIs remain source-compatible.

`Page`, `Layout` and `Component` are ways to organize an application, not
language constructs, and handler names are ordinary identifiers: the example
routes to `register`, `registerSubmit` and `profile` with no `Page` suffix,
while applications that already use `registerPage` keep working unchanged. See
[10.1 Naming](WEB.md#101-naming).
