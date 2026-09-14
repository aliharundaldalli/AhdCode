# v1.4.0 realtime attendance

A small Web application that records check-ins in PostgreSQL and shows new
ones live on every open dashboard over a WebSocket. It is the v1.4.0 dogfood
application: Web, PostgreSQL, WebSocket, UUID v7, `Env.secret`, and Cron in
one place. No npm, CDN, or download at build or run time.

## Run it

```bash
cp .env.example .env
```

Set `DB_PASSWORD` (or `DB_PASSWORD_FILE`, the path of a file that holds it)
and `NOTIFY_TOKEN` in `.env`, then create the table once:

```bash
psql "host=127.0.0.1 dbname=attendance user=attendance" -f schema.sql
```

```bash
ahdcode dev app.ahd
```

Optionally, in a second terminal:

```bash
ahdcode run jobs.ahd
```

## What it does

| Request | Result |
| --- | --- |
| `GET /` | the sign-in form, or the dashboard with the latest 20 check-ins |
| `POST /sign-in` | CSRF-checked; stores `user_id` and `user_name` in the session |
| `POST /check-in` | CSRF-checked; inserts with `UUID.v7()`, reads `created_at` back with `RETURNING`, broadcasts one JSON line, redirects |
| `GET /live` (WebSocket) | same-origin only; refused with `401` without a signed-in session; read-only |
| `POST /internal/notify` | needs `Authorization: Bearer <NOTIFY_TOKEN>`; broadcasts the body as a notice |
| `POST /sign-out` | CSRF-checked; clears the session |

`jobs.ahd` is a second program. A Web server and a Cron scheduler each occupy
the program that runs them, so the two run side by side and share the
database. Once a minute it counts the last hour's check-ins and posts the
summary to `/internal/notify`.

## Files

| File | Role |
| --- | --- |
| `app.ahd` | configuration, routes, the `/live` endpoint, managed assets |
| `Config/Database.ahd` | the shared PostgreSQL pool; the password comes from `Env.secret("DB_PASSWORD")` |
| `Config/Sessions.ahd` | the session store used by routes and by the `/live` accept check |
| `Live.ahd` | the WebSocket endpoint, its accept check, and the socket registry |
| `Pages/` | sign-in, dashboard, check-in, and the bearer-token notify endpoint |
| `jobs.ahd` | the Cron program that posts a summary through `HTTP.client` |
| `public/js/live.js` | the dashboard's live list; reconnects with a visible countdown |
| `schema.sql` | the `check_ins` table |

## Notes

- Live messages are built with `JSON.object`, and `live.js` writes them with
  `textContent`, so a student name is always text. Server-rendered names are
  escaped by `Web.UI`.
- WebSocket callbacks run one at a time together with HTTP handlers, so the
  socket registry is an ordinary `Pair<String, WebSocket>`.
- The notify guard is a pattern built from `request.header`, `Env.secret`,
  and `Security.secureEqual`. It is not a scoped API authentication framework.
- Behind a reverse proxy, forward the `Upgrade` and `Connection` headers for
  `/live`; see [WebSocket](../../../docs/WEBSOCKET.md).
