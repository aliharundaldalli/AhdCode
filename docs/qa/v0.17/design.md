# v0.17 Web foundations design

## Dogfood problem

The committed Math Portal (`examples/v0.15/ahd_math_portal`, after mentally
applying the v0.16 RequestContext / Form / CSRF / Flash contract) still
repeats three things the framework does not own as policy:

- 24 literal `"/admin..."` route registrations in `app.ahd`
- `sessionOpen` / `sessionCommit` at every handler boundary (40+ / 100+ calls)
- `adminGuard` / `signedInGuard` plus a refusal branch in every protected
  handler (24+ call sites)

v0.16 already made finalization explicit (`context.respond`). What remained
was registration noise and copied guard branches, not a missing auth product.

See `dogfood.md` for the before/after counts.

## Selected API

```ahd
routes: RouteSet := Web.routes(site, sessionStore)

routes.get(path, handler)
routes.get(path, handler, firstGuard, secondGuard)
routes.post(...)
routes.route(method, path, handler, firstGuard := allow, secondGuard := allow)

admin: RouteGroup := routes.group("/admin")
admin.get("/users", adminUsers, authenticated, adminOnly)
```

`List<Function>` is rejected by the language. Function-typed Class
attributes have no inferable signature (SEM023), so `.guard(fn)` cannot
store a live chain. Guards are therefore the extra parameters of
`get`/`post`/`route`, in order. Two slots cover the portal's
authenticated-then-admin pattern. More checks compose in application
code as one `Function(RequestContext) -> Response?`.

`App.get` / `App.post` / `App.route` stay `Function(Request) -> Response`.

Native adapter (not an application API):

```ahd
HTTP.contextHandler(store, opener, handler, first, second) -> Function
```

`opener` is `openContext` in WebRoutes (same construction as `Web.context`).
The adapter never calls `context.respond`.

## Handler signature

```ahd
handler: Function := (context: RequestContext) -> Response
```

The context-aware layer constructs one `RequestContext` per request with
`Web.context`. It never commits the session.

The handler (or a refusing guard) must return `context.respond(...)`.
A second `context.respond` on the same instance still raises
`WebContextError`.

## Guard signature

```ahd
check: Function := (context: RequestContext) -> Response?
```

- `null` — continue
- `Response` — stop; the value must already have been finalized by
  `context.respond` in the guard

No truthiness. No reflection. A wrong static shape is a compile error.

## Route group semantics

Grouping is registration convenience over the existing HTTP router.

- Prefix and child are explicit canonical fragments.
- Final path is `prefix + child` with one documented join rule.
- No silent `//` collapse, no query/fragment, no rewrite engine.
- Exact / trailing `/*` semantics stay exactly as v0.15.1 HTTP.
- Duplicate final `method + path` remains the existing HTTP error.

Join:

| prefix | child | final |
| --- | --- | --- |
| `/` | `/users` | `/users` |
| `/admin` | `/` | `/admin` |
| `/admin` | `/users` | `/admin/users` |
| `/admin/questions` | `/*` | `/admin/questions/*` |
| `/admin` | `/*` | `/admin/*` |

A final path of `/*` is rejected (HTTP already rejects it). The child
fragment `/*` is valid when joined to a non-root prefix.

Invalid fragments raise `WebRouteError` before any route is registered.

## Guard ordering

On one `get`/`post`/`route` call:

1. `first` guard
2. `second` guard
3. handler, once, if both returned `null`

Defaults are `allowAllGuard` (`null`). There is no hidden group-wide
store. Repeat the same two Functions on each protected route, or wrap
them in one application Function.

## Context finalization rule

Exactly one `context.respond` per request on the taken path.

The framework does not auto-commit after the handler returns. If a handler
returns a raw `Response`, the session is not committed — that is a
programmer error, not a hidden fix.

## Compatibility boundary

Old `Function(Request) -> Response` handlers keep compiling.
Context-aware registration is additive. `init web` stays a one-route
starter with no groups or guards.

## Rejected alternatives

- Hidden auto-finalization after handler return
- Global / thread-local current request
- Auth, roles, or “admin” policy inside Web
- `next()` / `use()` middleware stacks
- Route discovery, annotations, filename routing
- Named parameters, `{id}`, regex, `/**`
- New language syntax
- `List<Function>` guard lists (`List<Function>` is rejected by the language)
- Native Go router rewrite

AhdCode cannot store `Function` on a Class (SEM023) and cannot wrap an
unspecialized Function parameter in a lambda. The selected surface
therefore keeps guards as `get`/`post`/`route` parameters. A narrow
native adapter, `HTTP.contextHandler`, binds those already-checked
Functions to one `Function(Request) -> Response`. It creates the
context through an AhdCode opener and never commits the session.
HTTP still owns matching.
