# AhdCode v2.11.0 RC1 cross-platform validation

Create a fresh checkout on the target host. Use Go 1.26.4 or later and
Node.js with `node --test`. On macOS, install the Xcode Command Line Tools
for race-enabled builds and check `xcode-select -p`. On Linux, make a C
compiler available for the race detector.

```sh
git clone --branch release/v2.11.0-rc1 --single-branch https://github.com/aliharundaldalli/AhdCode.git
cd AhdCode
git rev-parse HEAD
go version
node --version
qa_dir="$(mktemp -d)"
go build -trimpath -o "$qa_dir/ahdcode" ./cmd/ahdcode
go build -trimpath -o "$qa_dir/ahdsqlite" ./cmd/ahdsqlite
export AHDCODE_BIN="$qa_dir/ahdcode"
export AHDCODE_SQLITE_RUNTIME="$qa_dir/ahdsqlite"
test "$("$AHDCODE_BIN" version)" = 'AhdCode v2.11.0'
node --test internal/initweb/pwa_worker_test.cjs
```

## macOS native verification

```sh
go test -count=1 -timeout 20m -run 'Test(BasicStarter|PWA|DevHTTPS|DevReload|HTTPNativeAndEvaluator|HTTPErrorPage|PersistentSession|WebSocket|WebPackage|WebAssets|WebPWA)' \
  ./cmd/ahdcode ./internal/backend/golang/ahdruntime ./internal/build ./internal/initweb ./internal/packaging
go test -race -count=1 -timeout 20m -run 'Test(DevHTTPS|HTTPShutdown|PersistentSession|WebSocket|WebPackage|WebAssets|WebPWA)' \
  ./cmd/ahdcode ./internal/backend/golang/ahdruntime ./internal/packaging
```

The packaging test moves the source and the generated `.app` before
launching the binary from `Contents/MacOS`. Record the macOS version,
architecture, and any filesystem-dependent skips. GitHub runs this
selection on ARM64.

## Linux verification

```sh
go list ./... | grep -v '^ahdcode/internal/build$' | xargs go test -count=1 -timeout 25m
go test -count=1 -timeout 45m ./internal/build
go test -race -count=1 -timeout 25m ./internal/backend/golang/ahdruntime ./internal/evaluator
go test -race -count=1 -timeout 25m -run 'WebSocket|HTTP|SMTP|V210' ./internal/build
go test -race -count=1 -timeout 20m ./internal/packaging ./internal/migrate
go vet ./...
git diff --check
```

The GitHub Linux workflow also covers nested helper modules and
PostgreSQL 17 and 18 against live service containers.

## Browser acceptance on either host

1. Generate a fresh Basic project with `"$AHDCODE_BIN" init web basic`.
   Generate and wire the PWA with `"$AHDCODE_BIN" pwa init --name "RC1 QA"`.
   [PWA integration steps](../../PWA.md)
2. Start development with `"$AHDCODE_BIN" dev --https app.ahd`. Trust
   the printed public CA in an isolated browser profile. Edit
   `Pages/Home.ahd` and verify live reload, then stop the dev server.
   [Local HTTPS](../../WEB.md#14-local-https)
3. Run `"$AHDCODE_BIN" package app.ahd --web --pwa --name RC1 --output "$qa_dir/deploy"`.
   Move the source directory and start the packaged binary with the
   deployment environment. The working directory is `RC1.app/Contents/MacOS`
   on macOS and `RC1/` on Linux. [Packaging](../../PACKAGING.md)
4. With the packaged app behind HTTPS, check `HEAD /ahdcode-sw.js`,
   install and launch in a supported browser, disconnect and reconnect,
   and verify that private responses do not enter Cache Storage.
   Record browser and OS versions separately from the Actions result.
