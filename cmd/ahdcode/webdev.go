package main

import (
	"fmt"
	"io"
	"net"
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
//
// v0.15 derives this identity but does not resolve it: there is no bundled
// .test resolver, so the name does not open in a browser on its own. Every
// caller below therefore has to say which of the two it is showing -- the
// address that works, or the identity the application is configured with.
func (environment webEnvironment) developmentHost() string {
	return environment.host + ".test"
}

func (environment webEnvironment) developmentURL() string {
	return environment.protocol + "://" + environment.developmentHost()
}

// hasDevelopmentIdentity reports whether this environment is the one that
// derives a .test name. Only development does, which is exactly what
// AppConfig.effectiveURL already decides -- test and production use APP_HOST
// unchanged, so advertising a .test name for them would contradict the
// application's own configuration.
func (environment webEnvironment) hasDevelopmentIdentity() bool {
	return environment.environment == "development" && environment.host != "" && environment.protocol != ""
}

// hasBindAddress reports whether the socket keys are both present. They are
// required configuration, so a missing one means the child has already failed
// its own validation and there is no address worth printing.
func (environment webEnvironment) hasBindAddress() bool {
	return environment.serverHost != "" && environment.serverPort != ""
}

// bindAddress is SERVER_HOST:SERVER_PORT, the socket the application actually
// binds -- the same pair AppConfig.address reports. An IPv6 literal is
// bracketed so the result stays a valid authority.
func (environment webEnvironment) bindAddress() string {
	if !environment.hasBindAddress() {
		return "SERVER_HOST:SERVER_PORT"
	}
	return net.JoinHostPort(environment.serverHost, environment.serverPort)
}

// openURL is the address a person can actually open right now. It is built
// from the real SERVER_HOST and SERVER_PORT rather than a hardcoded loopback,
// with one display substitution: a wildcard bind address is not something a
// browser can navigate to, so it is shown as the loopback the server is in
// fact reachable on. Nothing is invented -- an application bound to 0.0.0.0
// is genuinely served at 127.0.0.1.
func (environment webEnvironment) openURL() string {
	host := environment.serverHost
	switch host {
	case "0.0.0.0":
		host = "127.0.0.1"
	case "::", "[::]":
		host = "::1"
	}
	scheme := environment.protocol
	if scheme == "" {
		scheme = "http"
	}
	return scheme + "://" + net.JoinHostPort(host, environment.serverPort)
}

// checkWebEnvironment decides whether this configuration may run under the
// development command at all. It runs before the child is started, so a
// refusal means no process, no listener, and no descriptor.
//
// Two configurations are refused, both for the same underlying reason: dev
// would otherwise run something other than what the environment describes.
//
// The alternative in each case -- quietly treating production as development,
// or quietly serving an https application over plaintext http -- would make
// `ahdcode dev` lie about what it is running. It fails instead, and never
// rewrites APP_ENV or APP_PROTOCOL.
func checkWebEnvironment(environment webEnvironment) error {
	if environment.environment == "production" {
		return fmt.Errorf(
			"APP_ENV is production, but this is the development command.\n" +
				"  ahdcode dev runs an application in development.\n" +
				"  Set APP_ENV=development for local work, or run the built\n" +
				"  executable directly for a production configuration.\n" +
				"  Nothing was started and APP_ENV was not changed.")
	}
	if environment.protocol == "https" {
		// `ahdcode dev` starts the application, and the application binds a
		// plaintext HTTP socket. There is no path in v0.15 by which
		// APP_PROTOCOL=https results in TLS here, so starting the child would
		// mean serving http while the configuration -- and any URL printed
		// from it -- says https. Refusing is the honest outcome; downgrading
		// silently would hide a secure-cookie or mixed-content problem until
		// production.
		identity := "https://" + environment.host
		if environment.environment == "development" && environment.host != "" {
			identity = environment.developmentURL()
		}
		message := "Local HTTPS is not available in AhdCode v0.15.\n" +
			"  ahdcode dev serves plaintext HTTP, so it cannot honour\n" +
			"  APP_PROTOCOL=https.\n"
		if environment.host != "" {
			message += "\n  Configured identity:\n  " + identity + "\n"
		}
		message += "\n  Set APP_PROTOCOL=http for local development, or terminate\n" +
			"  HTTPS with an external local proxy in front of " + environment.bindAddress() + ".\n" +
			"  Nothing was started and APP_PROTOCOL was not changed."
		return fmt.Errorf("%s", message)
	}
	return nil
}

// announceWebApplication prints the Web banner once the application is
// running.
//
// The address that actually works comes first and is labelled as the one to
// open. The .test identity is shown after it, labelled as the configured
// identity and marked as not locally routed, because v0.15 ships no resolver
// for it -- presenting it as the primary URL would send a reader to a name
// their machine cannot resolve.
//
// Only development has a .test identity. For test the configuration uses
// APP_HOST unchanged, and printing a .test name there would contradict
// AppConfig, so the banner shows the bind address alone.
func announceWebApplication(output io.Writer, environment webEnvironment) {
	fmt.Fprintln(output)
	fmt.Fprintln(output, "AhdCode Web")
	if environment.name != "" {
		fmt.Fprintf(output, "  %s (%s)\n", environment.name, environment.environment)
	}
	if environment.hasBindAddress() {
		fmt.Fprintln(output)
		fmt.Fprintln(output, "  Open:")
		fmt.Fprintf(output, "  %s\n", environment.openURL())
	}
	if environment.hasDevelopmentIdentity() {
		fmt.Fprintln(output)
		fmt.Fprintln(output, "  Development identity:")
		fmt.Fprintf(output, "  %s\n", environment.developmentURL())
		fmt.Fprintln(output, "  (.test is not locally routed in v0.15)")
	}
	fmt.Fprintln(output)
}
