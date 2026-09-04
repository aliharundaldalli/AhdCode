package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ahdcode/internal/build"
	"ahdcode/internal/framework"
	"ahdcode/internal/module"
)

// Web awareness in `ahdcode dev`.
//
// This is a reporting and safety layer over the existing dev controller, not
// a second one. The v0.13/v0.14 controller still owns the source graph, the
// watcher, last-good, rebuild, restart, and stop; everything here happens
// once, after a successful build, and only for an application that actually
// imports the Web framework.
//
// It reads configuration the same way the application will -- process
// environment first, then the app-root .env -- but only ever to decide what
// to print and whether the session may run. It never sets a variable, never
// rewrites APP_ENV, and never passes anything to the child.

// webEnvironment is the subset of the application contract dev cares about.
type webEnvironment struct {
	name        string
	environment string
	host        string
	protocol    string
	serverHost  string
	serverPort  string
}

// isWebApplication reports whether this build's module graph contains the
// bundled Web framework. Asking the compiler is exact: an application that
// wrote `bring Web` gets the Web treatment and one that did not is left
// completely alone, so nothing here can misfire on a non-Web program that
// happens to have APP_ENV set in its environment.
func isWebApplication(result build.Result) bool {
	if result.Compilation == nil {
		return false
	}
	webID := module.ModuleID(framework.ModuleID("Web"))
	for id, item := range result.Compilation.Modules {
		if id == webID && item != nil {
			return true
		}
	}
	return false
}

// readWebEnvironment resolves the four APP_ keys with the application's own
// precedence: a variable already exported by this process wins, and the
// app-root .env only fills in what the environment does not already say.
func readWebEnvironment(entry string) webEnvironment {
	values := parseDevEnvFile(filepath.Join(filepath.Dir(entry), ".env"))
	lookup := func(key string) string {
		if value, present := os.LookupEnv(key); present {
			return strings.TrimSpace(value)
		}
		return strings.TrimSpace(values[key])
	}
	return webEnvironment{
		name:        lookup("APP_NAME"),
		environment: lookup("APP_ENV"),
		host:        lookup("APP_HOST"),
		protocol:    lookup("APP_PROTOCOL"),
		serverHost:  lookup("SERVER_HOST"),
		serverPort:  lookup("SERVER_PORT"),
	}
}

var devEnvKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// parseDevEnvFile reads the same KEY=value grammar the Env module's loader
// accepts, and deliberately no more: no interpolation, no command
// substitution, nothing executed. A malformed line is skipped rather than
// reported, because this reader exists only to label the dev banner -- the
// application's own Web.configure is what actually validates the contract,
// and it is the one that must produce the error the user acts on.
func parseDevEnvFile(path string) map[string]string {
	values := make(map[string]string)
	content, err := os.ReadFile(path)
	if err != nil {
		return values
	}
	for _, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimLeft(strings.TrimRight(rawLine, "\r"), " \t")
		if line == "" || line[0] == '#' {
			continue
		}
		equals := strings.IndexByte(line, '=')
		if equals < 0 {
			continue
		}
		key := line[:equals]
		if !devEnvKeyPattern.MatchString(key) {
			continue
		}
		values[key] = unquoteDevEnvValue(line[equals+1:])
	}
	return values
}

func unquoteDevEnvValue(rest string) string {
	if len(rest) == 0 {
		return ""
	}
	switch rest[0] {
	case '"':
		if closing := strings.IndexByte(rest[1:], '"'); closing >= 0 {
			return rest[1 : 1+closing]
		}
	case '\'':
		if closing := strings.IndexByte(rest[1:], '\''); closing >= 0 {
			return rest[1 : 1+closing]
		}
	}
	return strings.TrimSpace(rest)
}

// developmentHost is APP_HOST with .test appended -- the local name, never
// the real one, so development traffic cannot reach the production host by
// accident.
func (environment webEnvironment) developmentHost() string {
	return environment.host + ".test"
}

func (environment webEnvironment) developmentURL() string {
	return environment.protocol + "://" + environment.developmentHost()
}

// checkWebEnvironment refuses exactly one thing: running a configuration that
// declares itself production through the development command. Everything else
// is left to the application's own Web.configure, which validates the full
// contract and reports the offending key.
//
// The alternative -- quietly treating production as development, or quietly
// rewriting APP_ENV -- would make `ahdcode dev` a way to run production code
// under development semantics without saying so. It fails instead.
func checkWebEnvironment(environment webEnvironment) error {
	if environment.environment == "production" {
		return fmt.Errorf(
			"APP_ENV is production, but this is the development command.\n" +
				"  ahdcode dev runs an application in development.\n" +
				"  Set APP_ENV=development for local work, or run the built\n" +
				"  executable directly for a production configuration.\n" +
				"  Nothing was started and APP_ENV was not changed.")
	}
	return nil
}

// announceWebApplication prints the Web banner once the application is
// running. The canonical development URL is the line the user actually needs,
// so it gets its own line with nothing else on it.
func announceWebApplication(output io.Writer, environment webEnvironment) {
	fmt.Fprintln(output)
	fmt.Fprintln(output, "AhdCode Web")
	if environment.name != "" {
		fmt.Fprintf(output, "  %s (%s)\n", environment.name, environment.environment)
	}
	if environment.host != "" && environment.protocol != "" {
		fmt.Fprintln(output)
		fmt.Fprintf(output, "  %s\n", environment.developmentURL())
	}
	fmt.Fprintln(output)
	for _, line := range localHTTPSNotice(environment) {
		fmt.Fprintln(output, line)
	}
}

// localHTTPSNotice explains what an https development URL still needs on this
// machine. A first Web run must never be mysterious: if the name does not
// resolve, the reason and the way forward are printed here.
//
// It deliberately does not fall back to http, and never rewrites APP_PROTOCOL.
// An application that says APP_PROTOCOL=https is served over https or not at
// all: a silent downgrade would mean testing something other than what is
// configured, and would hide a secure-cookie or mixed-content problem until
// production.
//
// v0.15 does not ship the local certificate authority, the .test resolver, or
// the development gateway that would make this URL open by itself. Reaching
// https://<APP_HOST>.test with no port needs three permanently privileged
// pieces of system state at once -- a root-installed resolver for the .test
// domain, a listener on privileged port 443, and a certificate authority in
// the system trust store -- and that is a larger, security-sensitive change
// than this release should make quietly. It is deferred rather than
// approximated: nothing here installs, or asks for, any privilege.
func localHTTPSNotice(environment webEnvironment) []string {
	if environment.protocol != "https" {
		return nil
	}
	return []string{
		"  APP_PROTOCOL is https, which is the application's public identity.",
		"  This machine has no local certificate authority or .test resolver,",
		"  so " + environment.developmentURL() + " does not open on its own yet.",
		"",
		"  Until it does, either serve development over http by setting",
		"  APP_PROTOCOL=http, or put a TLS-terminating proxy in front of",
		"  " + environment.bindHint() + ". APP_PROTOCOL was not changed.",
		"",
	}
}

// bindHint names the socket the application binds, which is what a proxy in
// front of it forwards to. It is read from the same environment the
// application reads, and falls back to the shape of the keys rather than
// inventing a port.
func (environment webEnvironment) bindHint() string {
	if environment.serverHost != "" && environment.serverPort != "" {
		return environment.serverHost + ":" + environment.serverPort
	}
	return "SERVER_HOST:SERVER_PORT"
}
