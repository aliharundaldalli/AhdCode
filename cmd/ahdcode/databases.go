package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"ahdcode/internal/initweb"
	"ahdcode/internal/localdev"
)

// `ahdcode databases` still means "open AhdDataStudio" with no arguments --
// that is what it has always meant and nothing about it changes here. v0.19
// adds three small management subcommands beneath it so a database can become
// visible in Studio without hand-editing AHD_DATA_SQLITE_PATHS.
//
// It stays a registry, not a database administration CLI. It records where a
// SQLite file is; it does not create, migrate, inspect, back up, or delete
// one, and `remove` forgets an entry rather than touching the file.
func runDatabases(arguments []string, input io.Reader, output, errorOutput io.Writer) int {
	if len(arguments) == 0 {
		return startAhdDataStudio(input, output, errorOutput)
	}
	switch arguments[0] {
	case "list":
		if len(arguments) != 1 {
			fmt.Fprintln(errorOutput, "ahdcode databases list: no arguments are accepted")
			return 2
		}
		return runDatabasesList(output, errorOutput)
	case "add":
		if len(arguments) != 2 {
			fmt.Fprintln(errorOutput, "ahdcode databases add: exactly one SQLite file is expected, as in: ahdcode databases add app.db")
			return 2
		}
		return runDatabasesAdd(arguments[1], output, errorOutput)
	case "remove":
		if len(arguments) != 2 {
			fmt.Fprintln(errorOutput, "ahdcode databases remove: exactly one SQLite file is expected, as in: ahdcode databases remove app.db")
			return 2
		}
		return runDatabasesRemove(arguments[1], output, errorOutput)
	default:
		fmt.Fprintf(errorOutput, "ahdcode databases: unknown subcommand %q\n", arguments[0])
		fmt.Fprintln(errorOutput, "usage: ahdcode databases [list | add <file.db> | remove <file.db>]")
		return 2
	}
}

func runDatabasesList(output, errorOutput io.Writer) int {
	databases, err := localdev.LoadDatabases()
	if err != nil {
		fmt.Fprintf(errorOutput, "ahdcode databases list: %v\n", err)
		return 1
	}
	path, pathErr := localdev.DatabasesPath()
	if len(databases) == 0 {
		fmt.Fprintln(output, "No databases are registered.")
		fmt.Fprintln(output, "Register one with: ahdcode databases add app.db")
		if pathErr == nil {
			fmt.Fprintf(output, "Registry: %s\n", path)
		}
		return 0
	}
	for _, database := range databases {
		state := "available"
		if !database.Available() {
			// Reported, never removed: a database on a volume that is not
			// mounted right now has not been withdrawn.
			state = "unavailable"
		}
		fmt.Fprintf(output, "%s\t%s\t%s\n", database.Driver, database.Path, state)
	}
	if pathErr == nil {
		fmt.Fprintf(output, "\nRegistry: %s\n", path)
	}
	return 0
}

func runDatabasesAdd(path string, output, errorOutput io.Writer) int {
	database, added, err := localdev.AddSQLite(path, "")
	if err != nil {
		fmt.Fprintf(errorOutput, "ahdcode databases add: %v\n", err)
		return 1
	}
	if !added {
		fmt.Fprintf(output, "Already registered: %s\n", database.Path)
		return 0
	}
	fmt.Fprintf(output, "Registered: %s\n", database.Path)
	fmt.Fprintln(output, "Open it with: ahdcode databases")
	return 0
}

func runDatabasesRemove(path string, output, errorOutput io.Writer) int {
	database, removed, err := localdev.RemoveSQLite(path)
	if err != nil {
		fmt.Fprintf(errorOutput, "ahdcode databases remove: %v\n", err)
		return 1
	}
	if !removed {
		canonical, _ := localdev.CanonicalDatabasePath(path)
		fmt.Fprintf(errorOutput, "ahdcode databases remove: %s is not registered\n", canonical)
		return 1
	}
	fmt.Fprintf(output, "Removed from the registry: %s\n", database.Path)
	fmt.Fprintln(output, "The database file itself was not changed.")
	return 0
}

// startAhdDataStudio is the no-argument behaviour, unchanged in shape from
// v0.18: find Studio, make sure it has a .env, and run it. v0.19 adds the
// local router around it, so the canonical URL is a clean name rather than a
// loopback path, and hands Studio the registry location so registered
// databases appear without any environment configuration.
func startAhdDataStudio(input io.Reader, output, errorOutput io.Writer) int {
	dir, err := initweb.LocateAhdDataStudio()
	if err != nil {
		fmt.Fprintln(errorOutput, err.Error())
		return 1
	}
	if err := ensureStudioEnv(dir); err != nil {
		fmt.Fprintf(errorOutput, "ahdcode databases: %v\n", err)
		return 1
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(errorOutput, "ahdcode databases: %v\n", err)
		return 1
	}
	if err := os.Chdir(dir); err != nil {
		fmt.Fprintf(errorOutput, "ahdcode databases: %v\n", err)
		return 1
	}
	defer func() { _ = os.Chdir(cwd) }()

	// Studio is an AhdCode program and has no way to compute a per-user
	// configuration directory itself, so the CLI -- which does -- passes the
	// registry's location in the child's environment. The value is a path to
	// a file of local metadata; it is not a credential and grants nothing.
	if registry, err := localdev.DatabasesPath(); err == nil {
		_ = os.Setenv(studioRegistryEnvKey, registry)
	}

	holder := newLocalRouterHolder(localRouteIsLive)
	defer holder.close()
	routerPort := holder.port()

	openURL := studioOpenURL(routerPort)
	fmt.Fprintf(output, "AhdDataStudio: %s\n", localdev.Route{Hostname: localdev.StudioHost}.URL(routerPort))
	fmt.Fprintf(output, "Direct: %s\n", initweb.AhdDataStudioLoopbackURL())
	if openURL == initweb.AhdDataStudioLoopbackURL() {
		fmt.Fprintf(output, "%s is not mapped to 127.0.0.1 yet; run `ahdcode local hosts apply` to use the clean URL.\n",
			localdev.StudioHost)
	}
	fmt.Fprintf(output, "Using: %s\n", openURL)

	go registerStudioRoute(dir)
	go openStudioLater(openURL)
	return runRun([]string{"app.ahd"}, input, output, errorOutput)
}

// studioRegistryEnvKey names the database registry for AhdDataStudio.
const studioRegistryEnvKey = "AHD_DATA_REGISTRY"

// AhdDataStudio's fixed loopback socket and mount point. Both are properties
// of the application itself and are not configurable here: it binds
// 127.0.0.1 only, deliberately, and serving it on a wildcard address would
// publish a database browser to the network.
const (
	studioBindHost = "127.0.0.1"
	studioBindPort = 8081
	studioBasePath = "/AhdDataStudio"
)

// studioOpenURL picks the address to actually open. The clean name is used
// only when this machine genuinely resolves it and a router is serving it;
// otherwise the loopback URL, which always works, is used instead. Opening a
// browser at a name the machine cannot resolve would be a worse experience
// than showing the longer address.
func studioOpenURL(routerPort int) string {
	if routerPort == 0 {
		return initweb.AhdDataStudioLoopbackURL()
	}
	if !localdev.HostsMapsLoopback(localdev.ReadSystemHosts(), localdev.StudioHost) {
		return initweb.AhdDataStudioLoopbackURL()
	}
	return localdev.Route{Hostname: localdev.StudioHost}.URL(routerPort)
}

// registerStudioRoute publishes Studio's route once its run descriptor
// exists.
//
// The wait is what keeps the registry honest: the descriptor is written by
// `ahdcode run` only after Studio's control channel is up, so registering
// against it means the route is backed by a session that can be asked whether
// it is still alive. Registering earlier would create a name pointing at a
// port nothing had bound yet.
func registerStudioRoute(dir string) {
	descriptorPath := filepath.Join(dir, "app.run")
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		descriptor, err := readRunDescriptor(descriptorPath)
		if err == nil && runDescriptorIsLive(descriptor) {
			_, _ = localdev.Allocate(localdev.StudioHost, localdev.Route{
				Kind:        localdev.KindStudio,
				Source:      filepath.Join(dir, "app.ahd"),
				ProjectRoot: dir,
				Descriptor:  descriptorPath,
				BindHost:    studioBindHost,
				BindPort:    studioBindPort,
				BasePath:    studioBasePath,
				ControlPort: descriptor.ControlPort,
				OwnerPID:    os.Getpid(),
			}, localRouteIsLive)
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func openStudioLater(rawURL string) {
	time.Sleep(400 * time.Millisecond)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	_ = cmd.Start()
}

func ensureStudioEnv(dir string) error {
	envPath := filepath.Join(dir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	example := filepath.Join(dir, ".env.example")
	data, err := os.ReadFile(example)
	if err != nil {
		return nil
	}
	return os.WriteFile(envPath, data, 0o600)
}
