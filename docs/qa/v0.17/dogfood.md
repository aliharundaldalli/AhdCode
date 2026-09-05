# v0.17 portal dogfood

Source measured: committed `examples/v0.15/ahd_math_portal` at
`3806e7c`, after accounting for the v0.16 RequestContext / Form / CSRF /
Flash contract. The user's dirty portal worktree and public Math Portal
repository were not modified.

## Residual repetition after v0.16

| Pattern | Count | Notes |
| --- | ---: | --- |
| `"/admin..."` route literals in `app.ahd` | 24 | prefix copied on every line |
| `sessionOpen` call sites | 40 | one per handler, plus the helper |
| `sessionCommit` call sites | 119 | every response path |
| `adminGuard` / `signedInGuard` uses | 28 | plus a refusal branch each time |
| `Web.context` / `context.respond` in committed tree | 0 | v0.16 API not yet applied in this snapshot |

v0.16 removes the sessionOpen/commit pair in favor of one
`RequestContext` and explicit `context.respond`. It does **not** remove
the `/admin` prefix copies or the per-handler guard branches.

## Why v0.17

Registration should show policy in one place:

```ahd
admin := routes.group("/admin")
admin.get("/users", adminUsers, authenticated, adminOnly)
admin.get("/questions", adminQuestions, authenticated, adminOnly)
```

Policy stays application code. Web does not know what an admin is.

## Representative after (scratch, no DB)

`portal_routes.ahd` is a compilable route table for the public, auth, and
admin surfaces. It does not query MySQL. Handler bodies are stubs that
only prove registration, guards, and `context.respond`.

| Pattern | Before (committed portal) | After (scratch table) |
| --- | ---: | ---: |
| `"/admin"` prefix literals | 24 | 1 (`group("/admin")`) |
| Manual session open in admin handlers | 21 | 0 (router opens one context) |
| Manual guard/refusal branches in admin handlers | 24 | 2 named Functions on each protected line |
| Context construction in those handlers | 21 | 0 |
| CSRF / flash | unchanged v0.16 helpers | unused in the stub; composition is the example app |

Readability: a reader of `app.ahd` sees the `/admin` prefix and the two
guards without opening every admin page.

Limitation: the scratch file is not the live Math Portal. External DB,
SMTP, and upload flows were not executed. The complete stub source
compiles.
