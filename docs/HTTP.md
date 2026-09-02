# HTTP standard module

[English] · [Türkçe](HTTP_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [HTML](HTML.md) · [Student Guide](STUDENT_GUIDE_EN.md#36-a-small-web-page)

`HTTP` is the compiler-registered `builtin:HTTP` module, introduced in
AhdCode v0.4.0. It is explicit and a sibling `HTTP.ahd` cannot shadow it:

```ahd
bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring HTTPError
```

`HTTP` is a small, typed local web server. You register exact routes, read a
snapshot of each request, and return a `Response`. There is no middleware,
router DSL, HTTPS, cookies, sessions, multipart, WebSocket, or path
parameters. The implementation uses Go's `net/http` inside the AhdCode
runtime; there is no companion HTTP helper process.

## Public surface

```text
HTTP.server(host: String, port: Int, maxBodyBytes: Int := 1048576) -> Server
HTTP.response(status: Int, body: String, contentType: String)      -> Response
HTTP.text(body: String, status: Int := 200)                        -> Response
HTTP.html(body: String, status: Int := 200)                        -> Response
HTTP.redirect(location: String, status: Int := 303)                -> Response

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

Response.withHeader(name: String, value: String) -> Response

HTTPError  (derives from Error)
```

`Server`, `Request`, and `Response` are opaque built-in Classes: they cannot
be constructed with `Server()`, `Request()`, or `Response()`, have no public
attributes, and are obtained only from the functions above. All arguments are
positional. Omitted `maxBodyBytes` is `1048576`. Omitted `HTTP.text` /
`HTTP.html` status is `200`. Omitted redirect status is `303`.

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
list. Duplicate keys are preserved in order.

`header(name)` / `headerAll(name)` are case-insensitive. `body()` is the UTF-8
request body; invalid UTF-8 raises `HTTPError` (no silent replacement).

Forms are parsed only when `Content-Type` is `application/x-www-form-urlencoded`.
`form(name)` / `formAll(name)` then behave like query accessors. A malformed
or non-UTF-8 form body raises `HTTPError` when the handler calls `form` /
`formAll`. Multipart, files, and JSON bodies are not a form API; read
`body()` for a raw String.

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
contain CR or LF.

## Errors and containment

`HTTPError` derives from `Error`. Invalid host/port, invalid or duplicate
routes, invalid status, invalid redirect status, invalid headers, bind
failures, and invalid UTF-8 body/form access raise `HTTPError`.

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
static-file server, cookies, sessions, authentication, CSRF, middleware, path
parameters, wildcards, regex routes, reverse proxy, compression API, or
caching. Bodies are bounded Strings, not a binary type.
