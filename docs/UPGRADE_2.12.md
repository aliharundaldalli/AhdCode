# Upgrade to AhdCode v2.12.0

Verify your downloaded release artifacts against `SHA256SUMS.txt` before upgrading. Read the [installation guide](INSTALLATION.md) and [release notes](RELEASE_2.12.0.md).

1. Record the current compiler with `ahdcode version`. Back up your application database and verify restoration in a separate directory.
2. Install or extract v2.12.0 into an isolated test prefix first if evaluating a production application. Keep your working v2.11 installation and application data available for rollback.
3. Run `ahdcode check app.ahd` and then build the application with the Go version specified in `go.mod`. Deploy the compiler and SQLite helper from the same release package if you adopt `SQLite.openWithOptions`.
4. Exercise application routes, session restart and logout, authorized file downloads, and the PWA's offline behavior against temporary data before updating production.

The JSON checker uses 0 for success, 1 for source/input errors, and 2 for usage errors. See the [CLI schema](CLI.md#check-source-without-building) and [SQLite options and backup guide](SQLITE.md). Private files belong outside public asset mounts and should be opened only after a server-side authorization check. Always keep Gatekeeper enabled on macOS and verify Developer ID signatures (`spctl`, `pkgutil`, `stapler`) before running downloaded installers.
