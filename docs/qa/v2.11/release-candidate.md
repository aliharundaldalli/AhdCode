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

To build and verify the unsigned macOS CI distribution artifacts (`.tar.gz`,
`.pkg`, `.dmg` + `SHA256SUMS.txt`):

```sh
python3 -B tooling/distribution/leak_gate.py --self-test
python3 -B tooling/distribution/build.py --platform darwin --macos-arch both --without-bundled-runtimes --without-vsix --output "$qa_dir/macos-dist"
AHDCODE_TEST_MACOS_DIST_DIR="$qa_dir/macos-dist" python3 -B -m unittest -v tooling/distribution/test_macos_dist.py
```

When Developer ID certificates and a `notarytool` keychain profile are
configured on a macOS signing host, produce and verify the signed and
Apple-notarized artifacts with Gatekeeper enabled:

```sh
spctl --status
python3 -B tooling/distribution/build.py \
  --platform darwin \
  --macos-arch both \
  --without-bundled-runtimes \
  --without-vsix \
  --sign-app "Developer ID Application: <Team Name> (<Team ID>)" \
  --sign-installer "Developer ID Installer: <Team Name> (<Team ID>)" \
  --notary-profile "<notarytool-keychain-profile>" \
  --output "$qa_dir/macos-dist-signed"
AHDCODE_TEST_MACOS_DIST_DIR="$qa_dir/macos-dist-signed" python3 -B -m unittest -v tooling/distribution/test_macos_dist.py
(cd "$qa_dir/macos-dist-signed" && shasum -a 256 -c SHA256SUMS.txt)
pkgutil --check-signature "$qa_dir/macos-dist-signed"/AhdCode-*.pkg
xcrun stapler validate "$qa_dir/macos-dist-signed"/AhdCode-*.pkg "$qa_dir/macos-dist-signed"/AhdCode-*.dmg
spctl -a -vv -t install "$qa_dir/macos-dist-signed"/AhdCode-*-macos-arm64.pkg
spctl -a -vv -t open --context context:primary-signature "$qa_dir/macos-dist-signed"/AhdCode-*-macos-arm64.dmg
```

Note: When built on Apple Silicon (`arm64`) without an Intel runner, `x86_64`
binary execution is cross-compiled and layout-verified while native runtime
execution on `x86_64` hardware remains unverified.

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
