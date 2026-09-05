# Ahd Akademi Matematik

A full-stack mathematics portal built with the **released AhdCode v0.15.0** compiler. This reference application exercises Web/UI, MySQL, sessions, Security, multipart uploads, SMTP, and outbound HTTP/JSON without changing the framework. The interface is Turkish. [Türkçe](README_TR.md) · [Acceptance evidence](QA.md) · [v0.16 dogfood findings](DOGFOOD.md).

## Local setup

Requirements: the installed AhdCode v0.15.0 CLI, a reachable MySQL server, and a writable private upload directory. No npm, Node, CDN, remote JavaScript, or Bootstrap JavaScript is needed.

From this directory:

```sh
ahdcode --version
cp .env.example .env  # only if .env does not already exist
chmod 600 .env
```

Edit `.env` with your local connection settings. It is intentionally ignored by `.gitignore`; never stage it or copy real credentials into documentation. Existing process environment variables take precedence over `.env`, including explicitly empty values. Both entry points load configuration at runtime; builds embed no environment secrets. Restart the application after changing configuration.

The development defaults are `APP_NAME=Ahd Akademi Matematik`, `APP_ENV=development`, `APP_HOST=ahdakademi.com`, `APP_PROTOCOL=http`, `SERVER_HOST=127.0.0.1`, and `SERVER_PORT=8160`. Set `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USERNAME`, `DB_PASSWORD`, and `DB_SECURITY` for your MySQL installation. Do not assume the example has provisioned credentials.

Create the intended local database using a MySQL account with permission, supplying your username and password interactively:

```sql
CREATE DATABASE IF NOT EXISTS ahd_math_portal
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

Inspect existing tables before applying `database/schema.sql`. It creates five InnoDB tables with `IF NOT EXISTS` and seeds missing settings without replacing existing values. It does not migrate an incompatible older schema. Never drop a populated database to install this example.

```sh
mysql --host=127.0.0.1 --port=3306 --user=YOUR_USER -p ahd_math_portal < database/schema.sql
ahdcode dev app.ahd
```

Open **http://127.0.0.1:8160**. `http://ahdakademi.com.test` is a development identity only: v0.15 does not install local DNS/resolver routing for `.test`. Password-reset links in development use the working bind address. Normal local use needs MySQL; Gemini and SMTP are optional.

## Optional local HTTPS preview

For this Mac, `Caddyfile.local` serves **https://ahdakademi.com.test:8443** and proxies to `127.0.0.1:8161`. Install Caddy (`brew install caddy`) and use Python 3 for this optional helper; neither is a dependency of the core HTTP application.

```sh
mkdir -p .local
/Users/ahd/go/bin/ahdcode build app.ahd -o .local/portal
python3 scripts/local_https.py start
python3 scripts/local_https.py status
python3 scripts/local_https.py stop
```

The helper checks occupied ports and stops only recorded processes whose command identities still match. It does not build automatically or watch source edits; stop, rebuild, and restart after code changes. The preview uses internal port 8161 so it does not conflict with the default 8160 HTTP dev server.

The hostname requires an explicit local `127.0.0.1 ahdakademi.com.test` hosts entry (administrator authorization); AhdCode v0.15 does not create this mapping itself.

It preserves `.env` and overrides only the child process with production HTTPS behavior, `APP_HOST=ahdakademi.com.test`, and the application's optional `APP_PUBLIC_PORT=8443`. This yields Secure cookies and correct HTTPS reset links. The public port is separate from `SERVER_PORT`; an absent/invalid override preserves the usual canonical URL. This is an application convention, not a framework change.

Ignored `.local/` contains the binary, logs, PID state, and private Caddy CA keys. Keep it private. Caddy's admin API, HTTP redirect listener, and automatic trust-store installation are disabled. Browser trust requires an explicit, separate user decision. To trust only SSL/ahdakademi.com.test in the current user's keychain:

```sh
security add-trusted-cert -r trustRoot -p ssl -s ahdakademi.com.test \
  -k "$HOME/Library/Keychains/login.keychain-db" \
  .local/caddy-data/caddy/pki/authorities/local/root.crt
```

Remove that user trust setting with `security remove-trusted-cert .local/caddy-data/caddy/pki/authorities/local/root.crt`. Recreating the private CA requires a new trust decision. See [Caddy local HTTPS](https://caddyserver.com/docs/automatic-https#local-https) and the Turkish guide for details. This loopback preview does not publish the portal at the real public domain.

## First administrator

There is no default administrator or shipped password. `create_admin.ahd` reads the same local `.env` as the portal. It creates the named email if absent; if present, it updates that account's name/password, makes it an active administrator, and increments `auth_version`. Choose the email deliberately.

This Bash example reads the password without echoing it or storing it as a command in shell history:

```bash
read -r -p 'Administrator name: ' ADMIN_NAME
read -r -p 'Administrator email: ' ADMIN_EMAIL
read -r -s -p 'Administrator password (at least 10 characters): ' ADMIN_PASSWORD
printf '\n'
export ADMIN_NAME ADMIN_EMAIL ADMIN_PASSWORD
ahdcode run create_admin.ahd
unset ADMIN_PASSWORD ADMIN_EMAIL ADMIN_NAME
```

Sign in at `/login`, then open `/admin`. The continuation acceptance used a temporary setup account and removed it; it left no persistent test login. The setup helper reports its result as text; validation/duplicate-error branches do not all return a nonzero process exit status, so read the result.

## Features

- Registration normalizes email, requires a password of at least ten characters, and relies on the database email UNIQUE constraint. Login rejects wrong passwords and inactive accounts. Logout is a CSRF-protected POST.
- Every admin route checks the database role. User deactivation and password changes invalidate access on the next request. Sessions rotate on login; passwords use `Security.passwordHash`/`passwordVerify` (Argon2id).
- Administrators create/edit drafts and publish explicitly. Public lists and detail lookups select only published questions. Questions use query IDs such as `/question?id=1`.
- `site_name` and a strict `#RRGGBB` header color are editable in `/admin/settings`. These are display settings; deployment secrets never enter this table.
- Every state-changing route checks a session token with a constant-time comparison. Missing/wrong tokens return 403 after any required authorization guard. Input and generated content are escaped by semantic `Web.UI` nodes. Passwords are never repopulated into forms.

## Uploads

`UPLOAD_ROOT=storage/solutions` is private, outside `public/`; keep that separation in deployments. `UPLOAD_MAX_BYTES=5242880` limits each upload to 5 MiB. The request-body cap adds 64 KiB for the multipart envelope. PDF, PNG, and JPEG are accepted based on detected bytes, not declared MIME or the filename. This is content sniffing, not a complete document validator or malware scanner.

The server generates stored filenames. The uploader's filename is display metadata only. `UNIQUE(user_id, question_id)` enforces one solution per user per question even for direct duplicate POSTs. There is no replacement/delete/resubmission UI. An insert failure removes that request's newly saved file on a best-effort basis. Back up the database and upload directory together.

Only `/assets` maps to `public/`. Files under `storage/solutions/` are not public assets. `/admin/solutions/file?id=...` authorizes an administrator before reading the recorded file; ordinary users cannot download through that route. Never expose the private storage directory in a reverse-proxy static location.

## SMTP and password reset

Leave `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM_ADDRESS`, and `SMTP_FROM_NAME` empty when mail is unavailable. Set them to real deployment values only when you intend delivery; `SMTP_SECURITY` supports `none`, `starttls`, or `tls`. `SMTP_FROM_NAME` is currently read by configuration but not applied to the message's display name.

`/forgot-password` presents the same generic message for known/unknown addresses or delivery failure. Without SMTP there is no usable email delivery and no delivery-status UI. Links expire after 30 minutes and contain a selector plus a verifier; only the verifier's hash is stored. Expired, used, and wrong-verifier links are rejected. Successful reset increments the user's authentication version and invalidates other reset rows. Acceptance uses a loopback-only disposable SMTP sink; no real mail is sent.

## Optional Gemini drafting

Set both `GEMINI_API_KEY` and `GEMINI_MODEL` at runtime to enable the admin panel. No model or API credential is bundled. Empty values keep the rest of the application usable. The adapter uses AhdCode's HTTP client and JSON module; the key is an `x-goog-api-key` header, not a URL parameter. `GEMINI_BASE_URL` is an optional testing override for a local HTTP mock; keep it under operator control.

Generated text returns to editable title/body fields. Generation saves nothing and publishes nothing: the administrator must save a draft and then publish. Non-200 responses, malformed JSON, and unexpected response shapes return a message. Acceptance uses a local mock and a synthetic key; no real/billable Gemini request is made.

## Build, relocation, and production

```sh
/Users/ahd/go/bin/ahdcode build app.ahd -o /tmp/ahd_math_portal
```

A deployment directory needs only the executable, `public/`, writable private `storage/solutions/`, and runtime configuration (`.env` or process environment). Start the executable with that directory as its working directory so relative asset/storage paths resolve correctly. Schema installation is a provisioning step. Neither AhdCode/compiler sources nor framework `.ahd` files are runtime requirements. Offline operation still needs the configured local MySQL server; optional external mail/AI obviously requires its configured service.

In production use `APP_ENV=production`, your public `APP_HOST`, and `APP_PROTOCOL=https`. A reverse proxy such as Caddy/nginx or an appropriate tunnel terminates public TLS; the application binds its configured internal HTTP socket. Restrict that socket to the proxy. Secure session cookies follow the public protocol, and production reset links use the canonical public URL. Do not expose an internal HTTP endpoint as a public TLS service or expect `.test` routing.

## Assets and limitations

`public/css/bootstrap.min.css` is the actual Bootstrap **5.3.3** CSS asset (232,803 bytes). Its upstream copyright/MIT notice is preserved at the beginning of the file; see `public/css/bootstrap.LICENSE`. `public/css/app.css` contains the portal's local styling. All layouts link local stylesheets; there are no script tags or remote runtime assets. UI composition uses `Web.UI` forms, labels, navigation, tables, and text nodes rather than raw HTML.

This is a dogfood reference, not a production-hardening claim. Sessions are in memory and end on restart; multiple instances do not share sessions. There is no built-in rate limiting, mail retry/queue/status, transactional multi-step reset, upload malware scanning, pagination for all lists, or formula typesetting. Unknown-question pages render a message with HTTP 200. Several validation/database errors are intentionally coarse, and length bounds do not cover every schema column. Do not treat the generic reset message as proof of equal response timing. See [DOGFOOD.md](DOGFOOD.md) for measured friction and narrowly scoped v0.16 priorities.
