# v0.17 route groups and guards

Context-aware routes, one `/admin` group, and two ordered guards on each
protected registration line. No database, npm, or CDN.

```bash
cp .env.example .env
ahdcode dev app.ahd
```

| Request | Result |
| --- | --- |
| `GET /` | `200 home` |
| `GET /admin` without a session | `401` — `adminOnly` never runs |
| `POST /login` as `user=ada&role=member` then `GET /admin` | `403` — signed in, not admin |
| `POST /login` as `user=ada&role=admin` then `GET /admin` | `200 admin dashboard` |
| `GET /admin/secret` as admin | `200 admin secret` |

Guards and the handler each call `context.respond` on the path they take.
There is no hidden finalization and no general middleware chain.
