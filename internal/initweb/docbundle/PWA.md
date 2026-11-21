# Progressive Web Apps

Türkçe

A Progressive Web App (PWA) is a website a supporting browser can install
and launch in its own window. A manifest describes its name, icon, and
launch behavior. A service worker can handle selected requests offline.
AhdCode still serves `.ahd` pages built with Web.UI and HTMLNode. Install
availability depends on the browser and platform. An installed web app
still runs in the browser and needs the server for application logic,
authentication, database queries, and actions. AhdCode generates web assets
for this feature. It does not produce APK or IPA files.

## Add a PWA to a project

Run from the project root:

```sh
ahdcode pwa init --name "Example Site" --short-name "Example"
```

The name defaults to the directory name. The short name defaults to the
name. The command creates a `PWA/` directory and refuses to replace an
existing one. It leaves the entry point and layouts for you to wire in.

1. Add `require("PWA/Integration.ahd")` to the entry point and the shared
   layout. For Basic the layout is `Layouts/Main.ahd`.
2. Call `ahdPwaMount(site)` after creating your Web App and before
   `site.start()`.
3. Include `ahdPwaManifest()` and `ahdPwaRegistration()` as two nodes in
   the layout head. They work in a Basic head list or the head argument
   of `Web.page` and `Web.document`.

The generated functions reuse existing Web, HTTP, and HTMLNode APIs. The
mount serves the manifest and worker at exact routes with explicit media
types and exposes only the separate `PWA/public` asset directory.
Use `ahdcode package app.ahd --web --pwa` to deploy the manifest, worker and
public files alongside the compiled application. [Web packaging](PACKAGING.md).
If your source graph imports the same HTTP types directly,
use a consistent import source for those types across the graph.

## Install and update

Edit `PWA/app.webmanifest` to configure the app name, short name, start URL,
scope, display mode, theme color, background color, and icons. The default
launch URL and scope are `/` with standalone display and local 192 and 512
pixel PNGs. One generated worker owns the origin root. Use a dedicated
origin for each app and avoid mounting it on an origin with another
root-scoped worker. The browser's installation command may appear in the
address bar or menus. A discovered manifest alone does not establish that
the browser offers installation. [Web App Manifest](https://www.w3.org/TR/appmanifest/).

Registration requires a secure top-level context. For local HTTPS use
`ahdcode dev --https app.ahd` and trust the local CA in the test browser.
Loopback HTTP can work in browsers that treat it as trustworthy. The
ordinary HTTP `.test` router is not a secure origin. The worker and all
generated URLs are same-origin. For production retain
`Browser HTTPS → Caddy → AhdCode HTTP`, set `APP_PROTOCOL=https`, and
proxy the generated paths unchanged alongside the app routes.
[Caddy reverse proxy](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy).

## Offline behavior and cache policy

The default worker uses network-first handling for a query-free navigation
to `/`. If the network fails it returns a small, generic offline response.
HTTP errors and redirects from a reachable server remain online responses.
Other navigations, API requests, WebSocket handshakes, writes, and AhdCode
live-reload traffic stay with the browser's normal network handling. The
worker never saves navigation or runtime fetch responses.
Keep `Cache-Control: no-store` on private application responses to control
the browser's ordinary HTTP cache as well.

Only explicit public files in the worker's `PUBLIC_ASSETS` list enter
Cache Storage. The starting list contains the two icons. Each entry
requires the expected MIME type and a SHA-256 integrity string. Install
fetches the files with credentials omitted and rejects redirects. The
worker associates the cache with its scope, version, and asset list.
It removes older caches in that namespace on activation. An update waits
for existing controlled tabs to close before taking over. An icon or worker
update must be deployed alongside any matching file changes. Use simple
filenames directly in `PWA/public` for precached assets. Update the integrity
value in `PWA/worker.js` whenever one of the selected files changes. For
example, with Node installed, compute the icon's SHA-256 value with:

```sh
node -e "const fs=require('node:fs'); const c=require('node:crypto'); console.log('sha256-'+c.createHash('sha256').update(fs.readFileSync(process.argv[1])).digest('base64'))" PWA/public/icon-192.png
```

Never add session-specific files, API output, tokens, or private data to the list.
[Service Workers](https://w3c.github.io/ServiceWorker/).

When the connection returns, reload or revisit the page to resume live
server behavior. Test offline with a separate browser profile, and close
all app tabs before checking worker activation. If you remove the feature
from a deployed origin, also retire its existing browser registration
and corresponding Cache Storage entries.
