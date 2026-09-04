# HTTP standard module

[English] · [Türkçe](HTTP_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [HTML](HTML.md) · [Student Guide](STUDENT_GUIDE_EN.md#38-http-client)

> **v0.15:** `HTTP` is the low-level module and stays exactly as documented
> here. The first-party [`Web`](WEB.md) framework composes it -- `Web.app`
> wraps `Server`, and `Web` re-exports `Request`, `Response`, `Session`,
> `SessionStore`, `Cookie`, and `UploadedFile` as the *same* types. Using
> `bring HTTP` directly is never a fallback or a legacy path; reach for it
> whenever you want the server without the framework's configuration
> contract.


If you are learning this module, start with the [HTTP/HTTPS workshop](PRACTICAL_MODULES.md#7-http-and-https-requests-responses-and-failures)
for URL structure, statuses, transport failures, JSON POST, and client safety;
use this page as the complete Server and Client reference.

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
from HTTP bring Client
from HTTP bring ClientRequest
from HTTP bring ClientResponse
```

`HTTP` is a small, typed local web server **and** an outbound HTTP/HTTPS
client. You register exact routes, read a snapshot of each request, and return
a `Response`. v0.5.0 adds HTTP cookies and an in-memory server-side
`SessionStore`. v0.6.0 adds `Client`, `ClientRequest`, and `ClientResponse` so
a program can call external HTTP and HTTPS APIs. v0.9.1 adds `HTTP.file` and
`HTTP.download`, binary-safe responses for a file already on disk. Server `Request`/`Response`
and outbound `ClientRequest`/`ClientResponse` are distinct types. There is no
middleware, router DSL, multipart, WebSocket, path parameters, authentication
framework, or AI vendor module. The implementation uses Go's `net/http` inside
the AhdCode runtime; there is no companion HTTP, cookie, session, or client
helper process.

## Public surface

```text
HTTP.server(host: String, port: Int, maxBodyBytes: Int := 1048576) -> Server
HTTP.response(status: Int, body: String, contentType: String)      -> Response
HTTP.text(body: String, status: Int := 200)                        -> Response
HTTP.html(body: String, status: Int := 200)                        -> Response
HTTP.redirect(location: String, status: Int := 303)                -> Response
HTTP.file(path: String, contentType: String)                       -> Response
HTTP.download(path: String, contentType: String, fileName: String) -> Response
HTTP.cookie(name: String, value: String)                           -> Cookie
HTTP.deleteCookie(name: String, path: String := "/")               -> Cookie
HTTP.sessions(
    cookieName: String := "ahd_session"
    maxAgeSeconds: Int := 86400
    secure: Bool := false
    sameSite: String := "Lax"
) -> SessionStore
HTTP.client(
    timeoutSeconds: Int := 30
    maxResponseBytes: Int := 8388608
    followRedirects: Bool := true
) -> Client
HTTP.clientRequest(method: String, url: String) -> ClientRequest

Request.file(name: String)  -> UploadedFile?
Request.files(name: String) -> List<UploadedFile>

UploadedFile.originalName()        -> String
UploadedFile.declaredContentType() -> String?
UploadedFile.detectedContentType() -> String
UploadedFile.size()                -> Int
UploadedFile.save(directory: String) -> String

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

Client.send(request: ClientRequest) -> ClientResponse
Client.get(url: String)             -> ClientResponse
Client.post(
    url: String
    body: String
    contentType: String := "text/plain; charset=utf-8"
) -> ClientResponse

ClientRequest.withHeader(name: String, value: String) -> ClientRequest
ClientRequest.addHeader(name: String, value: String)  -> ClientRequest
ClientRequest.withBody(body: String)                  -> ClientRequest

ClientResponse.status()              -> Int
ClientResponse.body()                -> String
ClientResponse.header(name: String)  -> String?
ClientResponse.headerAll(name: String) -> List<String>
ClientResponse.url()                 -> String

HTTPError  (derives from Error)
```

`Server`, `Request`, `Response`, `Cookie`, `SessionStore`, `Session`, `Client`,
`ClientRequest`, and `ClientResponse` are opaque built-in Classes: they cannot
be constructed with `Server()`, `Client()`, or the other type names, have no
public attributes, and are obtained only from the functions above. All
arguments are positional. Omitted `maxBodyBytes` is `1048576`. Omitted
`HTTP.text` / `HTTP.html` status is `200`. Omitted redirect status is `303`.
Omitted `HTTP.deleteCookie` path is `"/"`. Omitted `HTTP.sessions` arguments
are `ahd_session`, `86400`, `false`, and `"Lax"`. Omitted `HTTP.client`
arguments are `30`, `8388608`, and `true`. Omitted `Client.post` content type
is `text/plain; charset=utf-8`.

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

## Static files

`server.static(prefix, root)` (v0.14) maps every request whose path begins
with `prefix` to a file under the local directory `root`:
`server.static("/assets", "public")` serves a request for `/assets/app.css`
from `public/app.css`. This is a first-party, low-level primitive for the
application's own local files -- CSS, JavaScript, SVG, images, fonts -- not
a general file browser and not a router: there is no directory listing, no
`index.html` convention, and no wildcard/prefix mechanism beyond this one
purpose-built method.

An exact route always wins over a static prefix: `routes` is checked first
on every request, and `static()` is only consulted on a miss, so
`server.get("/assets/special.txt", handler)` can freely override one file
under an otherwise-static directory with zero ambiguity. `static()` serves
`GET`/`HEAD` only; other methods on a path under a static prefix fall
through to the normal 404.

Every request path segment is checked before it ever reaches the
filesystem: empty, `.`, `..`, and any dotfile/dot-directory segment
(`.env`, `.git/config`, `.DS_Store`, anywhere in the path, not only its
last component) are refused outright, which also defeats percent-encoded
traversal (`%2e%2e`), since the path is already decoded by the time this
check runs. The resulting candidate is still canonicalized and checked for
containment under `root`, and, if it turns out to be a symlink, checked
again after resolving it -- a symlink whose target escapes `root` is never
served, matching `require(...)`'s own containment model; one that resolves
back inside `root` is served normally. A request that maps to a directory
is refused (404), never listed.

Files stream through the same binary-safe `http.ServeContent` machinery as
`HTTP.file`/`HTTP.download`, so Range and conditional-request headers work,
and nothing passes through an AhdCode `String`. `Content-Type` is set from
the file's extension via Go's standard MIME registry; an unrecognized
extension gets no explicit `Content-Type`.

`static()` follows the same registration rules as routes: it cannot be
called after `start()`, `prefix` must begin with `/`, `root` must already
exist and be a directory, and two registered prefixes may not overlap.

Editing a static file never triggers `ahdcode dev`'s rebuild -- static
serving reads straight from disk on every request, so a manual browser
refresh is always enough. See the [CLI guide](CLI.md#dev-watch-scope).

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

Forms are parsed when `Content-Type` is `application/x-www-form-urlencoded`
or `multipart/form-data`. `form(name)` / `formAll(name)` then behave like
query accessors, and multipart **text** fields arrive through that same API --
there is no second multipart-only text accessor. The same strict
percent-decoding rules apply to urlencoded form bodies: malformed or non-UTF-8
form data returns **400** before the handler runs. Multipart **file** parts
are read with `file` / `files` (see below). JSON bodies are not a form API;
read `body()` for a raw String.

The body is limited with `http.MaxBytesReader` **before** the handler runs.
A body larger than `maxBodyBytes` returns **413** and the handler is not
called. The exact limit is accepted; one extra byte is 413.

## File uploads

A `multipart/form-data` request carries text fields and file parts together.
Text fields are read with `form`/`formAll`; file parts are read with
`file`/`files`:

```ahd
handle: Function := (request: Request) -> Response {
    title: Local String? := request.form("title")
    paper: Local UploadedFile? := request.file("paper")
    if paper == null {
        return HTTP.text("paper is required", 400)
    }
    if paper.detectedContentType() != "application/pdf" {
        return HTTP.text("rejected", 415)
    }
    storedPath: Local String := paper.save("uploads/papers")
    return HTTP.text("saved " + storedPath)
}
```

`file(name)` is the first uploaded file for that field, or `null` when the
field carried no file. `files(name)` is every uploaded file for that field in
request order, or `[]` -- so `<input type="file" name="papers" multiple>`
never silently discards duplicates.

`UploadedFile` is opaque and read-only. There is no `bytes()`, `raw()`,
`stream()`, `tempPath()`, file handle, or pointer: an uploaded PDF is not a
`String`, and AhdCode does not pretend binary content is text. The bytes
reach the filesystem only through `save`.

### The uploader controls the name; you control the path

`originalName()` is **display metadata only**. It is the browser-supplied
filename reduced to a safe basename: `/` and `\` are both treated as
separators regardless of platform, a `C:` drive prefix is dropped, `.`/`..`
and empty names collapse to `file`, and a NUL byte is rejected as
structurally invalid. `../../evil.pdf` therefore surfaces as `evil.pdf`.

Never build a path from it:

```ahd
storedPath := "uploads/" + paper.originalName()   // do not do this
storedPath := paper.save("uploads/papers")        // do this
```

`save(directory)` creates the directory if needed and writes the upload under
a **cryptographically random** basename such as
`uploads/papers/8e8f30c65c4d4d23...`, created exclusively so it can never
overwrite an existing file. It returns the actual stored path. Two uploads
with the same original name always get different stored paths. Because the
generated basename contains no separator, the stored file is always a direct
child of the directory the application named -- an uploaded filename cannot
reach a parent directory.

The directory argument is the application's decision, not the uploader's;
AhdCode does not sandbox filesystem access in general.

An upload is persisted **once**. Calling `save` a second time on the same
`UploadedFile` raises `HTTPError` rather than quietly creating a duplicate
copy. Metadata methods keep working after a save.

### Declared type versus detected type

```text
declaredContentType()  what the client claimed   (never trust this)
detectedContentType()  what the bytes look like  (decide with this)
```

`declaredContentType()` is the part's own `Content-Type` header, normalized
to a bare media type (`text/plain; charset=utf-8` becomes `text/plain`), or
`null` when the part declared none. `detectedContentType()` sniffs the
leading bytes with Go's `net/http.DetectContentType`, also normalized.

The filename and its extension never influence detection. A text file named
`malware.pdf` claiming `application/pdf` reports:

```text
originalName()         malware.pdf
declaredContentType()  application/pdf
detectedContentType()  text/plain
```

so the application can reject the mismatch. A zero-byte upload has no content
to resemble and reports the documented `application/octet-stream` fallback
with `size()` of `0`; it is never mistaken for a valid PDF.

Detection answers roughly *"what content family do these bytes resemble?"* It
is **not** malware scanning, virus detection, or format validation: it does
not tell you a PDF is safe to open, that an image will decode, or that a
document carries no macros. There is no antivirus integration.

### Limits, lifetime, and cleanup

The server's `maxBodyBytes` still bounds the **whole** request body, multipart
included, and still returns **413** before any handler or persistence runs;
uploads have no separate unbounded path. There is no built-in per-file limit:
check `size()` and apply your own policy. An application expecting PDFs should
raise the server limit explicitly, as in
`HTTP.server("127.0.0.1", 8080, 26214400)`.

Each upload is backed by a private temporary file for the lifetime of its
request. When the handler finishes -- whether it responded normally, rejected
the upload, or raised -- every upload it did not save is deleted and its
registry entry dropped. Saved files have already moved out of that lifetime
and survive. Saving after the request has ended raises `HTTPError` rather than
resurrecting a deleted temporary file.

Malformed multipart syntax (a missing or invalid boundary, a truncated part,
malformed part headers) returns **400** before the handler runs, like any
other malformed request. It never panics and never becomes a 500.

`body()` keeps its existing contract: it returns the UTF-8 request body and
raises `HTTPError` for a body that is not valid UTF-8. A binary multipart body
is therefore not readable through `body()` -- that is deliberate. Use
`file`/`files` for uploads and `form`/`formAll` for multipart text fields.

There is no outbound multipart in v0.8.0: `ClientRequest` cannot attach a file.

## Response

`HTTP.text` sets `text/plain; charset=utf-8`. `HTTP.html` sets
`text/html; charset=utf-8` and does **not** escape the body: use it for
trusted static markup (`r"""..."""`) or for a String already produced by
`HTML.document` / `HTML.render`. Status codes must be in `100..599`.

`HTTP.redirect` sets `Location` and accepts only `301`, `302`, `303`, `307`,
or `308`. `withHeader` returns a new `Response`; header names/values must not
contain CR or LF. `Set-Cookie` must be added with `withCookie`, not
`withHeader`, so multiple cookies are not collapsed.

## Binary file responses

`HTTP.text`, `HTTP.html`, and `HTTP.response` carry a `String` body. `HTTP.file`
and `HTTP.download`, added in v0.9.1, instead serve the exact bytes of a file
already on disk, without ever decoding them as UTF-8 or materializing them as
an AhdCode `String`:

```text
HTTP.file(path: String, contentType: String)                       -> Response
HTTP.download(path: String, contentType: String, fileName: String) -> Response
```

`HTTP.file` serves the file inline (no forced disposition). `HTTP.download`
adds `Content-Disposition: attachment` presented under `fileName`. Both return
an ordinary `Response`: `withHeader` and `withCookie` work on them exactly as
on a text response, and the result is just as immutable.

`path` is a **storage** path your application already trusts -- typically the
value `UploadedFile.save` returned, or a path your own code constructed. It is
not a static-files root, and request-supplied strings must never reach it
directly:

```ahd
storedPath := row["stored_path"].string()  // e.g. data/ozetler/14aa9132...
return HTTP.file(storedPath, "application/pdf")
```

`contentType` is your explicit declaration, sent verbatim as `Content-Type`.
It is never sniffed from the path, the file's bytes, or a client-supplied
name, and it must be a valid, header-safe media type: no CR/LF, no empty
string.

`fileName` in `HTTP.download` is presentation-only. It never affects the
filesystem lookup and is independent of `path` -- a completely opaque, unnamed
stored file can still download as `"ozet.pdf"`. It must not contain CR or LF.
Non-ASCII names (Turkish included) are encoded with a safe ASCII fallback plus
a standards-correct `filename*` (RFC 5987) for clients that support it, so
`"Özet Çalışması.pdf"` survives without risking header injection.

The file is opened only when the response is being sent, streamed to the
client without buffering the whole file in memory, and always closed
afterward -- including when the client disconnects mid-transfer. A missing
path is `404`. A directory, or a path that cannot be opened, is a contained
`500`; neither panics. `Content-Length` is set from the file's size when it is
an ordinary regular file. Range requests and conditional HEAD/GET arrive
through the same underlying HTTP primitive as everything else the server
sends, so a `Range` header gets a correct `206 Partial Content` without a
separate range API.

`HTTP.file`/`HTTP.download` do not add a static-files root, a URL-to-path
mapping, directory listing, or any web-root abstraction: they serve exactly
the one path the application passed, and only that path.

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
`body()` access raise `HTTPError`. Client contract and transport failures
(invalid URL, invalid client headers, DNS, connection, TLS verification,
timeout, more than 10 redirects, HTTPS-to-HTTP redirect, oversize body,
invalid response UTF-8) also raise `HTTPError`. An HTTP status of 400 or
higher is not an `HTTPError` by itself. Unknown or expired session cookies are
not errors; they become a new anonymous session.

If a handler raises any Error (or panics in the runtime), the client receives
**500** `Internal Server Error`. The internal message is written to stderr, not
to the client. The server keeps serving later requests.

## Concurrency

Go's `net/http` accepts connections concurrently. AhdCode handlers on one
`Server` are **serialized** with an unexported mutex. Two handlers never run
at the same time on the same server. There is no AhdCode thread, goroutine,
async, or lock API.

## Outbound client

`HTTP.client` builds a reusable `Client`. There is no `close()`. The runtime
reuses HTTP connections internally. `timeoutSeconds` is the **total** request
timeout and must be in `1..9223372036`. `maxResponseBytes` bounds the buffered
response body and must be in `1..9223372036854775806`. Values outside either
range raise `HTTPError`; they are not clamped. The default body limit is 8 MiB
(`8388608`). The runtime reads at most `maxResponseBytes + 1` bytes; exactly
`N` bytes succeed and `N + 1` raises `HTTPError`. There is no millisecond or
floating timeout API.

`HTTP.clientRequest(method, url)` does **not** uppercase the method. `"GET"`
is GET; `"get"` stays `"get"` if it is a valid HTTP token. The URL must be
absolute `http` or `https` with a non-empty host. Fragments, userinfo,
`file:`, `ftp:`, `data:`, `javascript:`, and malformed URLs raise `HTTPError`.
Send credentials with headers, not embedded userinfo.

`withHeader` replaces that header name case-insensitively. `addHeader`
appends another value. `withBody` replaces the String body. The original
request is unchanged. There is no binary body, automatic JSON, form encoding,
or multipart. Applications use the existing JSON module when they need JSON.

Header names and values are validated. CR, LF, invalid names, `Content-Length`,
and `Host` raise `HTTPError`. `Authorization`, `Content-Type`, `Accept`, and
`User-Agent` work normally. Failures never include secret header values in
`HTTPError.message` or stderr diagnostics.

`Client.send` performs one intended application request. There are **no
automatic retries**, including POST. HTTP status codes `400`, `401`, `403`,
`404`, `429`, `500`, and `503` still return `ClientResponse`. Applications
decide what those statuses mean. `HTTPError` is for invalid URLs, invalid
request contracts, DNS/connection/TLS failures, timeouts, redirect-policy
violations, a response larger than `maxResponseBytes`, and a body that is not
valid UTF-8. Invalid UTF-8 is rejected without U+FFFD replacement and without
truncation. An empty body is `""`; `204` works.

`Client.get(url)` is a GET with no custom headers or body. `Client.post`
sets the body and `Content-Type`. For anything richer, build a
`ClientRequest` and `send` it.

`ClientResponse` is an immutable buffered snapshot. `status()` is the status
code. `body()` is the UTF-8 String. `header(name)` is the first value or
`null`. `headerAll(name)` is every value, or `[]`, in observed order.
`url()` is the final response URL after redirects. There is no public stream
and no cookie jar.

### HTTPS and HTTP

HTTPS uses the platform/system trusted roots. Certificate chain and hostname
verification are on. There is no `insecureSkipVerify`, custom CA, or client
certificate API. An untrusted, self-signed, or expired certificate raises
`HTTPError`. Plain `http://` remains supported for localhost and tests.

### Redirects

`followRedirects = false` returns the first 3xx `ClientResponse`.
`followRedirects = true` follows ordinary redirects, at most 10, then
`HTTPError`. **HTTPS to HTTP redirects are rejected** with `HTTPError`.
`Authorization` and `Cookie` are not forwarded when the host or port changes.
Same-host HTTP redirects keep those headers.

## What this module does not do

The inbound server still has no HTTPS listener. Inbound uploads never become
a public Bytes type or a database BLOB, and nothing parses, renders, or scans
an uploaded document. `HTTP.file`/`HTTP.download` (v0.9.1) serve one
application-named path each; they are not a static-files root, a
URL-to-path mapping, a directory browser, a media-streaming framework, a
cache/ETag framework, or a progress API, and there is no chunked/resumable
upload. The outbound client has no
cookie jar, binary body, streaming API, SSE, WebSocket, multipart, file
upload, automatic retries, OAuth, custom CA, client certificates, insecure TLS
bypass, proxy API, or AI/OpenAI/Anthropic/Gemini module. There is no HTTP/2
or HTTP/3 API, database-backed sessions, authentication framework, CSRF,
middleware, path parameters, wildcards, regex routes, reverse proxy,
compression API, or caching. `server.static` (v0.14) serves local files
under one explicit root; it has no directory listing, `index.html`
convention, asset hashing, bundling, minification, or CDN. Bodies are
bounded Strings, not a binary type. Session values are Strings; structured
data is the application's conversion. JSON is never implied by HTTP.
