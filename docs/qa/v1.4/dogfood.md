# v1.4.0 realtime dogfood

Source measured: this working tree, with the WebSocket `onOpen` ordering fix
described below. Nothing was committed, tagged, or published.

Environment: macOS 26 (arm64); native binaries from `ahdcode build`; a
throwaway PostgreSQL 18.6 cluster (Homebrew) on `127.0.0.1:55432` with
SCRAM authentication, SSL off, and session `TimeZone` `Europe/Istanbul`; the
in-app Chromium browser for the manual check.

## What was built

| Artifact | Exercises |
| --- | --- |
| [`examples/v1.4/realtime_attendance`](../../../examples/v1.4/realtime_attendance/README.md) | Web routes, CSRF, sessions, PostgreSQL pool and `RETURNING`, `UUID.v7()`, `Env.secret` through `DB_PASSWORD_FILE`, a `/live` WebSocket with `withAccept`, a bearer-token notify endpoint, managed assets, Cron in `jobs.ahd` posting through `HTTP.client`, and `live.js` with a visible reconnect |
| `examples/v0.1/69_uuid.ahd` | v4, v7 ordering, canonical parsing, `equals`/`compare`, zero, rejection |
| `examples/v0.1/70_env_secret.ahd` | `NAME`, `NAME_FILE`, trailing line break, the both-set error |
| `examples/v0.1/71_postgresql.ahd` | opt-in message without a server; NUMERIC, BYTEA, TIMESTAMPTZ, `RETURNING`, a rolled-back transaction |
| `examples/v0.1/72_websocket_echo.ahd` | echo, broadcast from an HTTP route, limits, close codes |

The planned numbering `66`–`69` was already used by v1.3.0, so the acceptance
programs are `69`–`72`.

## End-to-end run: 25 checks

A Go probe drove the native application and `jobs.ahd` against PostgreSQL over
HTTP and WebSocket. The final run passed all 25 checks:

| Area | Checks |
| --- | --- |
| Sign-in | anonymous `GET /` shows the form; `POST /sign-in` without CSRF is 403; with CSRF, 303 to `/`; the dashboard shows the user |
| Assets | the dashboard declares `live.js` as a module; `/assets/js/live.js` is 200; undeclared paths are 404 |
| `/live` | 401 without a session; 403 from another origin; opens with the session cookie; 401 again after sign-out; a client that sends is closed with `1008` and its reason |
| Check-in | 303 after `POST /check-in`; the socket receives the check-in as JSON; the id is a UUID v7; `created_at` came back through `RETURNING` in UTC (`+00`, although the session TimeZone is Istanbul); the dashboard lists the name HTML-escaped (`&lt;b&gt;Grace&lt;/b&gt;`); the flash appears once; an empty name is refused with a flash; an 81-character name fails the table's `CHECK` and becomes a flash, not a 500 |
| Notify | 401 with `WWW-Authenticate: Bearer` without a token; 401 with a wrong token; with the token, "delivered to 1 dashboard(s)" and the socket receives the notice |
| Cron | `jobs.ahd` posts "1 check-in(s) in the last hour" at the next minute and the socket receives it |

Closing from a callback was also repeated in isolation: 400 rounds (200 with
code `1000` on the echo example, 200 with `1008` on the application), a close
after 40 seconds open (one server ping), and a close after 65 seconds (two
pings), each ending with the expected close frame.

## Manual browser check

In the in-app browser: sign-in; a check-in through the form (flash and list
entry, status `Live`); a notice sent while the page was open appeared without a
reload. Stopping the server showed
`Disconnected (code 1006). Reconnecting in N s.` with a `Reconnect now` button,
backing off, and after three failures the hint to reload and sign in again.
After a restart the in-memory session was gone, so reconnecting was refused
until signing in again, as the hint says; after signing in the page was `Live`
and a new notice arrived.

## Findings

| # | Finding | Status |
| --- | --- | --- |
| 1 | **WebSocket: a broadcast could miss a just-connected client.** The frozen pipeline sent `101` before running `onOpen`, so the client saw the connection open before the application had registered the socket. Evidence: 1 of 200 "connect, then broadcast immediately" rounds reported "delivered to 0 dashboard(s)"; a long-running probe failed the same way on its first broadcast. | **Fixed with approval.** `onOpen` now runs before `101` (design.md steps 6–7 updated). Regression tests `TestWebSocketOnOpenCompletesBeforeTheClientSeesOpen` (fails on the old order, passes now) and `TestWebSocketPeerLostDuringOnOpenStillClosesOnce`. After the fix: 500 of 500 rounds delivered, and the 25-check run passed. |
| 2 | The first 25-check run reported the `1008` close check as failed with status `-1`. That probe could not tell a received message from a read error. | **Not reproduced** in the isolated close runs above or in three later full runs. The probe now prints every message before the close and the exact error. |
| 3 | Native `write` output is buffered and flushed at exit, before terminal input, and after each Cron job. Output written by a long-running server's handlers or WebSocket callbacks appears late, or not at all when the process is killed. | Pre-existing, unchanged in v1.4.0. `jobs.ahd` output appears because Cron flushes. Candidate for a later release. |
| 4 | The formatter breaks an empty argument list across lines when a long expression overflows, leaving a whitespace-only line. | Pre-existing. **Fix integrated** into this tree from a separate session: empty constructs are atomic in `delimitedGroup`, with regression test `TestEmptyArgumentListsAreNeverBrokenOpen`. Seven `examples/v0.15/ahd_math_portal` files whose canonical output changed were reformatted by the fixed formatter (layout only). The v1.4 examples are canonical under the fixed formatter and did not change. |
| 5 | Sessions are in memory, so restarting the application signs everyone out and `/live` refuses reconnects until the user signs in again. | Expected behaviour; `live.js` shows the hint after three failed attempts. |
| 6 | In the in-app browser automation, pressing Enter in the sign-in field did not submit the form; clicking the button did. The rendered page has one `<form>` with its submit button and text input. | Recorded as an automation observation, not a product defect. |
| 7 | Final QA: `TestWebSocketPolicyCloseCodes` failed once under full-suite load. Every client received the right close code and reason; the test also required four different connections to report `onClose` in one global order, which the design never promises (ordering is per socket). | **Test corrected** to compare the four events as a set. 400 stress runs, three package runs, and a race run pass. |

## Ergonomics observations (no API added)

- Top-level code in a `require`d file runs before `Web.start()` loads `.env`,
  so `Config/Database.ahd` and `Config/Sessions.ahd` each use a lazy nullable
  `Global` accessor (about 15 lines apiece) instead of a plain top-level value.
- The CSRF check and its 403 response are repeated in three handlers.
- The socket registry (`Pair<String, WebSocket>` plus `onOpen`/`onClose`
  bookkeeping and a broadcast loop) is about 25 lines and reads clearly; it did
  not need a framework.

These are notes for future design work. Scope was frozen, so v1.4.0 adds no API
for them.

## Limitations

- `security: "tls"` was exercised against a plaintext server, which refuses
  TLS; a server with a trusted certificate was not available. The TLS
  configuration itself is unit-tested.
- PostgreSQL 17 and Linux/Windows runners were not run locally; the CI
  `postgresql` job covers PostgreSQL 18 and 17 on Linux.
- `ahdcode dev` hot restart with an open WebSocket was not exercised manually;
  the local router's WebSocket passthrough is covered by
  `cmd/ahdcode/localrouter_websocket_test.go`.
