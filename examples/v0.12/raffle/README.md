# MySQL raffle showcase

A local, server-rendered raffle that demonstrates the AhdCode modules used
in a small web application: Env, HTTP, HTML, sessions, CSRF, Security, and
MySQL (including transactions and `MySQLResult`).

This is an example program, not a product and not part of AhdDataStudio.

## Purpose

Study one realistic application that covers:

- joining a raffle and receiving a participation code
- verifying a ticket without listing other codes
- admin lifecycle: idle → open → closed → drawn
- previous winners that survive the next round
- an admin audit log

## Features demonstrated

See [What this example demonstrates](#what-this-example-demonstrates).

## Requirements

- AhdCode v0.12.0 or later (`ahdcode version`)
- A reachable MySQL server
- A database the configured user can create tables in

## Database creation

Create an empty schema, then point `MYSQL_DATABASE` at it:

```sql
CREATE DATABASE ahdcode_demo;
```

Tables are created on first start:

- `raffle_admin` — Argon2id password hashes only
- `raffle` — singleton current round (`idle` / `open` / `closed` / `drawn`)
- `raffle_entry` — current participants
- `raffle_history` — completed draws
- `raffle_audit` — administrative actions (no passwords, no session IDs)

You can inspect them in [AhdDataStudio](../../../tools/AhdDataStudio/README.md).

## Environment setup

```bash
cd examples/v0.12/raffle
cp .env.example .env
```

Edit placeholders. **Never commit `.env`.**

`RAFFLE_ADMIN_PASSWORD` is required only when `raffle_admin` is empty. If it
is missing on that first run, startup stops with a configuration error and
does not print the password. An existing admin row is left unchanged.

## Run

```bash
cd examples/v0.12/raffle
ahdcode run app.ahd
```

Open [http://127.0.0.1:8082/](http://127.0.0.1:8082/)

Admin: [http://127.0.0.1:8082/admin](http://127.0.0.1:8082/admin)

Stop:

```bash
cd examples/v0.12/raffle
ahdcode kill app.run
```

The server binds **127.0.0.1:8082** only. It does not listen on `0.0.0.0`
and does not collide with AhdDataStudio on 8081.

## Routes

Public:

| Method | Path | Purpose |
|---|---|---|
| GET | `/` | Status, join form, or winner |
| POST | `/join` | Join while the raffle is open |
| GET | `/ticket` | Participation receipt |
| GET | `/verify` | Ticket check form |
| POST | `/verify` | Verify a current participation code |
| GET | `/history` | Previous winners (latest 25) |

Admin:

| Method | Path | Purpose |
|---|---|---|
| GET | `/admin` | Login or dashboard |
| POST | `/admin/login` | Argon2id login |
| POST | `/admin/logout` | End the admin session |
| GET | `/admin/participants` | Bounded participant list + search |
| POST | `/admin/participant/delete` | Remove one current entry |
| GET | `/admin/history` | Previous winners (latest 50) |
| GET | `/admin/audit` | Latest 100 audit rows |
| POST | `/admin/start` | Open a new round (`idle` only) |
| POST | `/admin/close` | Stop registration (`open` only) |
| POST | `/admin/draw` | Transactional winner selection (`closed` only) |
| POST | `/admin/reset` | Prepare the next round (`drawn` only) |

## Security model

- Bind address is loopback only.
- Admin passwords are hashed with `Security.passwordHash` (Argon2id). The
  plaintext is never stored or rendered.
- Login failures are generic: **Login failed**.
- Successful login calls `session.rotate()` and issues a new CSRF token.
- Every browser state change is POST + CSRF compared with
  `Security.secureEqual`.
- Participation codes come from `Security.token()` and are unique.
- Ticket checks compare the stored code with `Security.secureEqual`.
- Participant tables do not list full codes; admin sees a masked suffix.
- Names, notes, codes, and audit detail go through `HTML.text` (attributes
  are escaped by the HTML builder).
- SQL values are bound parameters. Table/column names are application
  constants.
- `redact` strips MySQL and admin passwords from startup error text.
- Queries that list rows use `LIMIT 25`, `50`, or `100`.

## Local-only nature

This program is for a trusted local machine. It is not hardened for the
public internet: sessions are the in-memory HTTP store, TLS to MySQL is
opt-in via `MYSQL_SECURITY`, and the HTTP server itself is plaintext on
127.0.0.1.

## What this example demonstrates

- MySQL connection (`MySQL.connect`, `db.ping()`)
- parameterized queries (`?` + `MySQL.fromString` / `fromInt`)
- transactions (`begin`, `execute`/`query` on `MySQLTransaction`, `commit`, `rollback`)
- `MySQLResult` (`affectedRows()`, `lastInsertId()`)
- Argon2id passwords (`Security.passwordHash` / `passwordVerify`)
- secure random tokens (`Security.token`)
- sessions and `session.rotate()` after login
- CSRF (`Security.token` + `Security.secureEqual`)
- safe HTML (`HTML.text` / escaped attributes)
- bounded queries (`LIMIT`)
- lifecycle/state management (`idle` / `open` / `closed` / `drawn`)
- error redaction (`redact`)
- nullable database values (`MySQLValue.isNull`, DECIMAL `AVG` kept as String)

## Language

[English](README.md) · [Türkçe](README_TR.md)
