package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"ahdcode/internal/localdev"
)

// `ahdcode local` is the visible half of v0.19's local development
// integration: it reports what the toolchain has set up on this machine, and
// it offers the one system change AhdCode cannot make on its own.
//
// It is deliberately small. There is no start, no stop, no restart, and no
// daemon to manage: the router lives inside whichever `ahdcode dev` or
// `ahdcode databases` session is running, and it disappears with them. What
// remains worth asking is "what is running, and where does it point", which
// is what status answers.

func runLocal(arguments []string, input io.Reader, output, errorOutput io.Writer) int {
	if len(arguments) == 0 {
		fmt.Fprintln(errorOutput, "ahdcode local: a subcommand is required, as in: ahdcode local status")
		return 2
	}
	switch arguments[0] {
	case "status":
		if len(arguments) != 1 {
			fmt.Fprintln(errorOutput, "ahdcode local status: no arguments are accepted")
			return 2
		}
		return runLocalStatus(output, errorOutput)
	case "hosts":
		return runLocalHosts(arguments[1:], input, output, errorOutput)
	default:
		fmt.Fprintf(errorOutput, "ahdcode local: unknown subcommand %q\n", arguments[0])
		fmt.Fprintln(errorOutput, "usage: ahdcode local status | ahdcode local hosts [apply|remove]")
		return 2
	}
}

// localRouteIsLive is the authoritative liveness test for a registered route.
//
// It never trusts the recorded pid: the operating system reuses pids, so a
// route whose owner died could otherwise appear live because something
// unrelated inherited its number. Instead it re-reads the descriptor the
// route names and asks that descriptor's own authenticated control channel to
// identify itself -- exactly the test `ahdcode stop` already uses, so a route
// is live on precisely the same terms a session is.
func localRouteIsLive(route localdev.Route) bool {
	if !localdev.DescriptorExists(route.Descriptor) {
		return false
	}
	switch route.Kind {
	case localdev.KindDev:
		descriptor, err := readDevDescriptor(route.Descriptor)
		if err != nil {
			return false
		}
		if route.ControlPort != 0 && descriptor.ControlPort != route.ControlPort {
			// The descriptor has been replaced by a newer session; this
			// entry belongs to the one that is gone.
			return false
		}
		return devDescriptorIsLive(descriptor)
	case localdev.KindStudio:
		descriptor, err := readRunDescriptor(route.Descriptor)
		if err != nil {
			return false
		}
		if route.ControlPort != 0 && descriptor.ControlPort != route.ControlPort {
			return false
		}
		return runDescriptorIsLive(descriptor)
	}
	return false
}

func runLocalStatus(output, errorOutput io.Writer) int {
	running, stale, err := localdev.PartitionRoutes(localRouteIsLive)
	if err != nil {
		fmt.Fprintf(errorOutput, "ahdcode local status: %v\n", err)
		return 1
	}
	port := probeLocalRouterPort()

	fmt.Fprintln(output, "AhdCode Local")
	fmt.Fprintln(output)
	if port == 0 {
		fmt.Fprintln(output, "Router: not running")
		fmt.Fprintln(output, "Bind: -")
		fmt.Fprintln(output, "  A router runs inside `ahdcode dev` and `ahdcode databases`.")
		fmt.Fprintln(output, "  Start one of those to serve the routes below.")
	} else {
		fmt.Fprintln(output, "Router: running")
		fmt.Fprintf(output, "Bind: %s:%d\n", localRouterHost, port)
		if port != localRouterPreferredPort {
			fmt.Fprintf(output, "  Port %d was not available, so local URLs carry :%d.\n",
				localRouterPreferredPort, port)
		}
	}

	fmt.Fprintln(output)
	writeLocalHostsStatus(output, hostnamesOf(running))

	fmt.Fprintln(output)
	if len(running) == 0 {
		fmt.Fprintln(output, "Routes: none")
	} else {
		fmt.Fprintln(output, "Routes:")
		for _, route := range running {
			fmt.Fprintf(output, "  %s\n", route.Hostname)
			fmt.Fprintf(output, "    url: %s\n", route.URL(port))
			fmt.Fprintf(output, "    -> %s\n", route.DisplayDestination())
			if route.Source != "" {
				fmt.Fprintf(output, "    source: %s\n", route.Source)
			}
		}
	}
	if len(stale) > 0 {
		fmt.Fprintln(output)
		fmt.Fprintln(output, "Stale routes (owner is no longer running; reclaimed on the next dev start):")
		for _, route := range stale {
			fmt.Fprintf(output, "  %s -> %s\n", route.Hostname, route.DisplayDestination())
		}
	}

	fmt.Fprintln(output)
	if path, err := localdev.RoutesPath(); err == nil {
		fmt.Fprintf(output, "Route registry: %s\n", path)
	}
	if path, err := localdev.DatabasesPath(); err == nil {
		fmt.Fprintf(output, "Database registry: %s\n", path)
	}
	return 0
}

func hostResolvesLoopback(name string) bool {
	ips, err := net.LookupHost(name)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if ip == "127.0.0.1" || ip == "::1" {
			return true
		}
	}
	return false
}

func hostnamesOf(routes []localdev.Route) []string {
	names := make([]string, 0, len(routes))
	for _, route := range routes {
		names = append(names, route.Hostname)
	}
	return names
}

func writeLocalHostsStatus(output io.Writer, active []string) {
	content := localdev.ReadSystemHosts()
	fmt.Fprintf(output, "System hosts: %s\n", localdev.SystemHostsPath())
	fmt.Fprintf(output, "  Host integration: %s\n", hostIntegrationState(content, localdev.HostsAuthorized()))
	managed := localdev.HostsBlockNames(content)
	if len(managed) == 0 {
		fmt.Fprintln(output, "  managed block: absent")
	} else {
		fmt.Fprintf(output, "  managed block: present (%d name(s))\n", len(managed))
	}
	for _, name := range mergedHostnames(managed, append(active, localdev.StudioHost)) {
		state := "not mapped"
		if localdev.HostsMapsLoopback(content, name) {
			state = "mapped to 127.0.0.1"
			if hostResolvesLoopback(name) {
				state += "; resolvable"
			} else {
				state += "; not yet resolvable"
			}
		}
		fmt.Fprintf(output, "  %s: %s\n", name, state)
	}
}

// mergedHostnames is the union AhdCode's managed block should contain: every
// name it already manages plus every name currently in use.
//
// The union is deliberate. A name is left in the block after its session
// stops, because a hosts entry pointing at loopback with nothing behind it is
// harmless -- and removing it would mean asking for administrator access
// again the next time the same project runs. Ownership of a name is decided
// by the route registry, never by this file.
func mergedHostnames(existing, wanted []string) []string {
	seen := map[string]bool{}
	var names []string
	for _, group := range [][]string{existing, wanted} {
		for _, raw := range group {
			name := strings.ToLower(strings.TrimSpace(raw))
			if !localdev.ValidHostname(name) || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func runLocalHosts(arguments []string, input io.Reader, output, errorOutput io.Writer) int {
	action := "show"
	if len(arguments) == 1 {
		action = arguments[0]
	} else if len(arguments) > 1 {
		fmt.Fprintln(errorOutput, "ahdcode local hosts: at most one of apply or remove is accepted")
		return 2
	}

	running, _, err := localdev.PartitionRoutes(localRouteIsLive)
	if err != nil {
		fmt.Fprintf(errorOutput, "ahdcode local hosts: %v\n", err)
		return 1
	}
	content := localdev.ReadSystemHosts()
	existing := localdev.HostsBlockNames(content)

	var block string
	if action != "remove" {
		block = localdev.RenderHostsBlock(mergedHostnames(existing, append(hostnamesOf(running), localdev.StudioHost)))
	}
	updated := localdev.ApplyHostsBlock(content, block)

	switch action {
	case "show":
		fmt.Fprintf(output, "AhdCode manages one block in %s:\n\n", localdev.SystemHostsPath())
		fmt.Fprint(output, block)
		fmt.Fprintln(output)
		if updated == content {
			fmt.Fprintln(output, "This block is already in place; nothing to do.")
			return 0
		}
		fmt.Fprintln(output, "It is not in place yet. Apply it with:")
		fmt.Fprintln(output, "  ahdcode local hosts apply")
		fmt.Fprintln(output, "Everything outside the two markers is left exactly as it is.")
		return 0
	case "apply", "remove":
		return writeSystemHosts(localdev.SystemHostsPath(), action, content, updated, input, output, errorOutput)
	default:
		fmt.Fprintf(errorOutput, "ahdcode local hosts: unknown action %q (expected apply or remove)\n", action)
		return 2
	}
}

// writeSystemHosts performs the one system change AhdCode offers, under
// rules that do not bend:
//
//   - Nothing outside the managed markers is altered; the new content is the
//     old content with one block replaced.
//   - Only 127.0.0.1 mappings are ever written.
//   - The change is applied directly when the user can already write the
//     file, and otherwise requires an explicit yes at an interactive prompt
//     before administrator access is requested. sudo is never invoked
//     silently and never invoked at all without that answer.
//   - A non-interactive session never prompts and never elevates. It prints
//     the exact command to run and exits, because a script that blocks
//     waiting for a password that will never be typed is worse than one that
//     fails immediately.
//
// The hosts file path is a parameter rather than read from the platform
// inside this function, so the whole decision tree above -- including the
// paths that would otherwise shell out -- is exercisable against a temporary
// file. No test in this repository writes to the real hosts file.
func writeSystemHosts(path, action, current, updated string, input io.Reader, output, errorOutput io.Writer) int {
	if updated == current {
		fmt.Fprintf(output, "%s is already what AhdCode would write; nothing was changed.\n", path)
		return 0
	}

	fmt.Fprintf(output, "AhdCode wants to update its managed block in %s.\n\n", path)
	writeHostsDiff(output, current, updated)
	fmt.Fprintln(output)

	staged, err := stageHostsFile(updated)
	if err != nil {
		fmt.Fprintf(errorOutput, "ahdcode local hosts: %v\n", err)
		return 1
	}
	defer func() { _ = os.Remove(staged) }()

	// The straightforward path: on a machine where the user can already
	// write the file, no elevation is involved at all.
	if err := replaceFileContents(path, updated); err == nil {
		fmt.Fprintf(output, "✓ Updated %s\n", path)
		return 0
	}

	if !isInteractive(input) {
		fmt.Fprintln(output, "This change needs administrator access, and this is not an interactive terminal.")
		fmt.Fprintln(output, "Nothing was changed. Run this yourself, or re-run in a terminal:")
		fmt.Fprintf(output, "  sudo cp %s %s\n", staged, path)
		fmt.Fprintln(output, "  (that staged file is removed when this command exits; re-run `ahdcode local hosts apply` in a terminal instead)")
		return 1
	}
	if runtime.GOOS == "windows" {
		fmt.Fprintln(output, "This change needs Administrator access on Windows.")
		fmt.Fprintf(output, "Open an elevated prompt and edit %s to contain the block above.\n", path)
		return 1
	}

	fmt.Fprintf(output, "Applying it requires administrator access (sudo).\n")
	if !confirmYes(input, output, "Apply this change now? [y/N]: ") {
		fmt.Fprintln(output, "Nothing was changed.")
		return 1
	}
	if err := elevatedReplace(staged, path); err != nil {
		fmt.Fprintf(errorOutput, "ahdcode local hosts: %v\n", err)
		fmt.Fprintln(errorOutput, "Nothing was changed.")
		return 1
	}
	fmt.Fprintf(output, "✓ Updated %s\n", path)
	if action == "remove" {
		fmt.Fprintln(output, "AhdCode's managed block was removed. Local names now fall back to loopback URLs.")
	}
	return 0
}

func writeHostsDiff(output io.Writer, current, updated string) {
	currentLines := map[string]bool{}
	for _, line := range strings.Split(current, "\n") {
		currentLines[line] = true
	}
	updatedLines := map[string]bool{}
	for _, line := range strings.Split(updated, "\n") {
		updatedLines[line] = true
	}
	for _, line := range strings.Split(updated, "\n") {
		if line != "" && !currentLines[line] {
			fmt.Fprintf(output, "  + %s\n", line)
		}
	}
	for _, line := range strings.Split(current, "\n") {
		if line != "" && !updatedLines[line] {
			fmt.Fprintf(output, "  - %s\n", line)
		}
	}
}

func stageHostsFile(content string) (string, error) {
	file, err := os.CreateTemp("", "ahdcode-hosts-*")
	if err != nil {
		return "", err
	}
	path := file.Name()
	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	// Readable so that the elevated copy below can read it; it holds only
	// loopback hostname mappings, which is the same information the file it
	// is about to become already publishes.
	_ = os.Chmod(path, 0o644)
	return path, nil
}

// replaceFileContents rewrites a file in place, keeping its inode, owner, and
// mode. /etc/hosts is a system file: replacing it with a fresh one owned by
// the current user would be a permissions change nobody asked for.
func replaceFileContents(path, content string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

// elevatedReplace runs exactly one narrow command, with a fixed argument
// shape, after the user has said yes. `cp` writes through the existing file
// rather than replacing it, so ownership and mode survive.
func elevatedReplace(staged, path string) error {
	sudo, err := exec.LookPath("sudo")
	if err != nil {
		return fmt.Errorf("sudo is not available on this machine; edit %s yourself", path)
	}
	copier, err := exec.LookPath("cp")
	if err != nil {
		return fmt.Errorf("cp is not available on this machine; edit %s yourself", path)
	}
	command := exec.Command(sudo, copier, filepath.Clean(staged), filepath.Clean(path))
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("the elevated copy did not complete: %v", err)
	}
	return nil
}

func confirmYes(input io.Reader, output io.Writer, prompt string) bool {
	fmt.Fprint(output, prompt)
	reader := bufio.NewReader(input)
	line, err := reader.ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
