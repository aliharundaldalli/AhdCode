# HTTP standard module

[English] · [Türkçe](HTTP_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [HTML](HTML.md) · [Student Guide](STUDENT_GUIDE_EN.md#37-cookies-and-sessions)

`HTTP` is the compiler-registered `builtin:HTTP` module, introduced in
AhdCode v0.4.0. It is explicit and a sibling `HTTP.ahd` cannot shadow it:

```ahd
bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring HTTPError
from HTTP bring Cookie
from HTTP bring SessionStore
from HTTP bring Session
```

`HTTP` is a small, typed local web server. You register exact routes, read a
snapshot of each request, and return a `Response`. v0.5.0 adds HTTP cookies
and an in-memory server-side `SessionStore`. There is no middleware, router
DSL, HTTPS, multipart, WebSocket, path parameters, or authentication
framework. The implementation uses Go's `net/http` inside the AhdCode
runtime; there is no companion HTTP, cookie, or session helper process.

## Public surface

```text
HTTP.server(host: String, port: Int, maxBodyBytes: Int := 1048576) -> Server
HTTP.response(status: Int, body: String, contentType: String)      -> Response
HTTP.text(body: String, status: Int := 200)                        -> Response
HTTP.html(body: String, status: Int := 200)                        -> Response
HTTP.redirect(location: String, status: Int := 303)                -> Response
HTTP.cookie(name: String, value: String)                           -> Cookie
HTTP.deleteCookie(name: String, path: String := "/")               -> Cookie
HTTP.sessions(
    cookieName: String := "ahd_session"
    maxAgeSeconds: Int := 86400
    secure: Bool := false
    sameSite: String := "Lax"
) -> SessionStore

Server.get(path: String, handler: Function)   -> Nothing
Server.post(path: String, handler: Function)  -> Nothing
Server.route(method: String, path: String, handler: Function) -> Nothing
Server.start()                                -> Nothing

Request.method()                   -> String
Request.path()                     -> String
Request.query(name: String)        -> String?
Request.queryAll(name: String)     -> List<String>
Request.header(name: String)       -> String?
Request.headerAll(name: String)    -> List<String>
Request.body()                     -> String
Request.form(name: String)         -> String?
Request.formAll(name: String)      -> List<String>
Request.cookie(name: String)       -> String?
Request.cookieAll(name: String)    -> List<String>

Response.withHeader(name: String, value: String) -> Response
Response.withCookie(cookie: Cookie)              -> Response

Cookie.withPath(path: String)       -> Cookie
Cookie.withHttpOnly(value: Bool)    -> Cookie
Cookie.withSecure(value: Bool)      -> Cookie
Cookie.withSameSite(mode: String)   -> Cookie
Cookie.withMaxAge(seconds: Int)     -> Cookie

SessionStore.open(request: Request) -> Session
SessionStore.commit(session: Session, response: Response) -> Response

Session.get(name: String)                 -> String?
Session.has(name: String)                 -> Bool
Session.set(name: String, value: String)  -> Nothing
Session.remove(name: String)              -> Nothing
Session.clear()                           -> Nothing
Session.rotate()                          -> Nothing
Session.destroy()                         -> Nothing

HTTPError  (derives from Error)
```

`Server`, `Request`, `Response`, `Cookie`, `SessionStore`, and `Session` are
opaque built-in Classes: they cannot be constructed with `Server()`,
`Request()`, `Response()`, `Cookie()`, `SessionStore()`, or `Session()`, have
no public attributes, and are obtained only from the functions above. All
arguments are positional. Omitted `maxBodyBytes` is `1048576`. Omitted
`HTTP.text` / `HTTP.html` status is `200`. Omitted redirect status is `303`.
Omitted `HTTP.deleteCookie` path is `"/"`. Omitted `HTTP.sessions` arguments
are `ahd_session`, `86400`, `false`, and `"Lax"`.

## Handler signature

Every route handler is an ordinary Function:

```text
(request: Request) -> Response
```

The compiler checks that shape statically. A missing parameter, a `String`
return, or an `Int` parameter is a compile-time error. Named arguments are
rejected on type operations (`app.get(path: "/", handler: home)` is invalid).

```ahd
home: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

app: Server := HTTP.server("127.0.0.1", 8080)
app.get("/", home)
app.start()
```

`app.get` and `app.post` register `"GET"` and `"POST"`. `app.route` uses the
method string as given: `"get"` is not `"GET"`. Method tokens follow RFC 9110
(no spaces).

## Server lifetime

`HTTP.server` records host, port, and body limit. It does not bind yet.
`start()` binds, then **blocks**. It does not open a browser and does not
print a startup banner. The terminal stays occupied until the process is
stopped.

`host` must be non-empty. `port` must be in `1..65535` (port `0` is invalid).
`maxBodyBytes` must not be negative. Bind failures raise `HTTPError`.

`127.0.0.1` is this machine only. Open `http://127.0.0.1:8080/` in a browser
on the same computer. Binding `0.0.0.0` makes the port reachable from other
machines on the network; do not do that casually.

Internal timeouts (not a public API): ReadHeader 10s, Read 15s, Write 15s,
Idle 60s. There are no access logs.

## Routing

Routes are exact path matches. The path must begin with `/` and must not
contain `?` or `#`. `/notes` and `/notes/` are different. Query strings are
not part of the route: `GET /notes?q=x` still matches `/notes`. Duplicate
`method + path` pairs raise `HTTPError`. Routes cannot change after `start()`.

Unknown paths return **404**. A path that exists for another method returns
**405** with an `Allow` header listing the registered methods.

## Request snapshot

Each handler receives an immutable snapshot, not a live Go request. `path()`
is the URL path without the query. `query(name)` is the first value, or
`null` if the name is absent. `queryAll(name)` is every value, or an empty
list. Duplicate keys are preserved in order. Valid UTF-8 percent-decoding is
unchanged (`Ay%C5%9Fe`, `%20`, `+`, emoji).

Malformed percent-encoding (`%`, `%2`, `%ZZ`) and percent-decoded invalid
UTF-8 (`%80`, `%C0%80`) in the query string return **400** and the handler is
not called. There is no U+FFFD replacement and no silent dropping of keys.

`header(name)` / `headerAll(name)` are case-insensitive. `cookie(name)` is the
first Cookie-header value with that exact name, or `null`. `cookieAll(name)`
is every matching value in request order, or `[]`. Cookie names are
case-sensitive. Cookie parsing is part of the immutable request snapshot.

`body()` is the UTF-8
request body; invalid UTF-8 raises `HTTPError` (no silent replacement).

Forms are parsed only when `Content-Type` is `application/x-www-form-urlencoded`.
`form(name)` / `formAll(name)` then behave like query accessors. The same
strict percent-decoding rules apply to form bodies: malformed or non-UTF-8
form data returns **400** before the handler runs. Multipart, files, and JSON
bodies are not a form API; read `body()` for a raw String.

The body is limited with `http.MaxBytesReader` **before** the handler runs.
A body larger than `maxBodyBytes` returns **413** and the handler is not
called. The exact limit is accepted; one extra byte is 413.

## Response

`HTTP.text` sets `text/plain; charset=utf-8`. `HTTP.html` sets
`text/html; charset=utf-8` and does **not** escape the body: use it for
trusted static markup (`r"""..."""`) or for a String already produced by
`HTML.document` / `HTML.render`. Status codes must be in `100..599`.

`HTTP.redirect` sets `Location` and accepts only `301`, `302`, `303`, `307`,
or `308`. `withHeader` returns a new `Response`; header names/values must not
contain CR or LF. `Set-Cookie` must be added with `withCookie`, not
`withHeader`, so multiple cookies are not collapsed.

## Cookies

`HTTP.cookie(name, value)` builds an immutable `Cookie`. Builders return a
new Cookie; the original is unchanged.

Defaults:

- Path `/`
- HttpOnly `false`
- Secure `false`
- SameSite `"Lax"`
- no Max-Age

SameSite accepts exactly `"Lax"`, `"Strict"`, or `"None"`. `"lax"` is not
normalized. `"None"` requires `Secure=true`.

Cookie names follow HTTP token rules (no whitespace, no `;`). Cookie values
are **cookie-octets**: ASCII `0x21–0x7E` except `"`, `,`, `;`, and `\`. CR,
LF, controls, spaces, and arbitrary Unicode are rejected with `HTTPError`.
AhdCode does not silently strip, quote, or percent-encode a dangerous value.
Internally generated session identifiers always satisfy this contract.

`HTTP.deleteCookie(name)` returns a Cookie whose wire form includes
`Max-Age=0` (and Go's compatible expiry). Send it with `withCookie`. There is
no separate `Response.deleteCookie`.

`withCookie` returns a new Response. Several cookies can be attached:

```ahd
response := HTTP.text("ok")
response = response.withCookie(cookieA)
response = response.withCookie(cookieB)
```

Both `Set-Cookie` headers reach the client.

## Sessions

`HTTP.sessions` creates an independent in-memory `SessionStore`. Two stores
do not share identifiers or values, even if cookie names differ only by
choice. The store is not tied to one `Server`.

A session cookie contains **only** an opaque random identifier (256 bits from
`crypto/rand`, base64url without padding). Application values are String-only
and live on the server. They are lost when the process exits. That is the
v0.5.0 contract, not a bug. SQLite data in the same program can persist
independently.

This is not authentication. There is no User, Login, Password, Role, or Guard
type. An application may store `user_id` itself after it verifies a password.

Default session cookie: Path `/`, HttpOnly **always true**, Secure and
SameSite as configured, Max-Age = `maxAgeSeconds`. HttpOnly cannot be
disabled. `maxAgeSeconds` must be greater than 0 and controls both the
browser cookie and server-side expiry. Expired sessions are not accepted.
Cleanup runs lazily during `open` and `commit`; there is no background
goroutine.

`Session.has(name)` is a method. After `.`, `has` is a member name; the `x has y`
operator is unchanged.

### Open, lazy creation, commit

```ahd
sessions: Global SessionStore
session: Local Session := sessions.open(request)
```

Inside a handler, a module-level store is declared with `Global`, like any
other module binding.

- Known, non-expired cookie: attach to that server-side session.
- Missing cookie: new anonymous logical session.
- Unknown, random, or expired cookie: new anonymous session. The supplied
  identifier is never trusted as a new server-side ID.

Opening a session that never calls `set`, `remove`, `clear`, or `rotate` does
not create stored state or a `Set-Cookie` header. Persistence happens when
state actually changes, or when `rotate()` needs a concrete identifier.

`Session.set` does not write HTTP headers. `SessionStore.commit(session, response)`
returns a **new** Response. That explicit boundary is required.

Values are String only. Convert explicitly: `int(session.get("count"))` after
a null check. There is no Any, JSON, or implicit Int.

### Rotate, destroy, clear

`rotate()` invalidates the old identifier immediately, issues a new random
id, keeps values, and `commit` sends the new cookie. A typical login-style
flow:

```ahd
session.rotate()
session.set("user_id", "{userId}")
return sessions.commit(session, HTTP.redirect("/panel"))
```

The pre-login cookie must not recover the logged-in session.

`remove(name)` deletes one key. `clear()` deletes all values but keeps the
session. `destroy()` invalidates server state; `commit` sends a deletion
cookie; the old id cannot be reused. After destroy, `get` is `null` and `has`
is `false`. Further mutating operations raise `HTTPError`.

```ahd
session.destroy()
return sessions.commit(session, HTTP.redirect("/"))
```

Unknown attacker cookies do not allocate persistent sessions. Only legitimate
mutation creates stored state.

## Errors and containment

`HTTPError` derives from `Error`. Invalid host/port, invalid or duplicate
routes, invalid status, invalid redirect status, invalid headers, invalid
cookies (name, value, path, SameSite, Max-Age), invalid session options,
session lifecycle misuse after `destroy`, bind failures, and invalid UTF-8
`body()` access raise `HTTPError`. Unknown or expired session cookies are not
errors; they become a new anonymous session.

If a handler raises any Error (or panics in the runtime), the client receives
**500** `Internal Server Error`. The internal message is written to stderr, not
to the client. The server keeps serving later requests.

## Concurrency

Go's `net/http` accepts connections concurrently. AhdCode handlers on one
`Server` are **serialized** with an unexported mutex. Two handlers never run
at the same time on the same server. There is no AhdCode thread, goroutine,
async, or lock API.

## What this module does not do

No HTTPS/TLS, HTTP/2 API, HTTP/3, WebSocket, SSE, multipart, file upload,
static-file server, database-backed sessions, authentication, CSRF, middleware, path
parameters, wildcards, regex routes, reverse proxy, compression API, or
caching. Bodies are bounded Strings, not a binary type. Session values are
Strings; structured data is the application's conversion.
