package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"ahdcode/internal/localdev"
)

// devLocalRoute is what `ahdcode dev` managed to arrange for one session's
// local identity. Every field is optional in the sense that matters: an
// application that is running correctly is reported as running correctly even
// when none of this worked, because a convenience failing is not the
// application failing.
type devLocalRoute struct {
	route      localdev.Route
	routerPort int
	// note explains, in one line, why there is no clean URL. Empty when
	// there is one.
	note string
}

func (local devLocalRoute) allocated() bool { return local.route.Hostname != "" }

// logicalURL is the address a person opens, or "" when this session has no
// routed identity. It is never the bind address: a caller that wants that
// asks for it by name.
func (local devLocalRoute) logicalURL() string {
	if !local.allocated() || local.routerPort == 0 {
		return ""
	}
	return local.route.URL(local.routerPort)
}

// loopbackBind reduces the application's configured bind address to the
// loopback address a router on this machine can actually reach.
//
// A wildcard bind is translated rather than refused: an application listening
// on 0.0.0.0 genuinely is reachable at 127.0.0.1, which is the same
// substitution the "Open:" line already makes. Anything else that is not
// loopback -- a bind pinned to one external interface -- yields no route at
// all, because the local router forwards to loopback and nowhere else.
func (environment webEnvironment) loopbackBind() (string, int, bool) {
	if !environment.hasBindAddress() {
		return "", 0, false
	}
	port, err := strconv.Atoi(environment.serverPort)
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, false
	}
	host := environment.serverHost
	switch host {
	case "0.0.0.0":
		host = "127.0.0.1"
	case "::", "[::]":
		host = "::1"
	}
	if !localdev.IsLoopbackHost(host) {
		return "", 0, false
	}
	return host, port, true
}

// preferredLocalHostname is the .test name this session would like.
func (environment webEnvironment) preferredLocalHostname() string {
	if !environment.hasDevelopmentIdentity() {
		return ""
	}
	return localdev.DeriveHostname(environment.host)
}

// establishLocalRoute claims a local hostname for this dev session and makes
// sure a router is serving it.
//
// The order matters. The route is registered only after the application's own
// child is running and its bind address is known, so a name never exists for
// something that is not there. Nothing here can fail the session: every
// failure path records a note for the banner and returns.
func (c *devController) establishLocalRoute(environment webEnvironment) {
	preferred := environment.preferredLocalHostname()
	if preferred == "" {
		return
	}
	bindHost, bindPort, ok := environment.loopbackBind()
	if !ok {
		c.localRoute.note = "the application does not bind a loopback address, so it cannot be routed locally"
		return
	}

	if c.routerHolder == nil {
		c.routerHolder = newLocalRouterHolder(localRouteIsLive)
	}

	route, err := localdev.Allocate(preferred, localdev.Route{
		Kind:        localdev.KindDev,
		Source:      c.entry,
		ProjectRoot: filepath.Dir(c.entry),
		Descriptor:  c.descriptorPath,
		BindHost:    bindHost,
		BindPort:    bindPort,
		ControlPort: c.controlPort,
		OwnerPID:    os.Getpid(),
	}, localRouteIsLive)
	if err != nil {
		c.localRoute.note = fmt.Sprintf("a local name could not be registered (%v)", err)
		return
	}
	c.localRoute.route = route
	c.localRoute.routerPort = c.routerHolder.port()
	if c.localRoute.routerPort == 0 {
		c.localRoute.note = "no local router port was available on this machine"
	}
}

func (c *devController) syncLocalHosts() {
	if !c.localRoute.allocated() {
		return
	}
	result := syncManagedHostname(hostSyncRequest{
		hostname:    c.localRoute.route.Hostname,
		input:       os.Stdin,
		output:      c.output,
		errorOutput: c.errorOut,
		interactive: isInteractive(os.Stdin),
	})
	if result.message != "" && c.localRoute.note == "" {
		c.localRoute.note = result.message
	}
}

// releaseLocalRoute gives the hostname back. It runs from the controller's
// own shutdown, so it covers every way a session ends that leaves the
// controller able to act: Ctrl-C, `ahdcode stop`, and `ahdcode kill`. A
// session that dies without running it leaves an entry behind, which the next
// `ahdcode dev` reclaims after finding its owner unreachable -- no name is
// held forever by a process that crashed.
func (c *devController) releaseLocalRoute() {
	if c.localRoute.allocated() {
		_ = localdev.Release(c.localRoute.route.Hostname, c.descriptorPath)
		c.localRoute.route = localdev.Route{}
	}
	if c.routerHolder != nil {
		c.routerHolder.close()
		c.routerHolder = nil
	}
}

// writeLocalIdentity prints the logical identity beneath the bind address.
//
// The two are always shown as two separate facts. The bind address is where
// the socket is; the local URL is the name this machine routes to it. Merging
// them into one line would be convenient right up until the port changes, at
// which point a single "URL" would be quietly wrong.
func writeLocalIdentity(output io.Writer, environment webEnvironment, local devLocalRoute) {
	if !environment.hasDevelopmentIdentity() {
		return
	}
	fmt.Fprintln(output)
	fmt.Fprintln(output, "  Local identity:")
	if !local.allocated() {
		fmt.Fprintf(output, "  %s\n", localdev.DeriveHostname(environment.host))
		if local.note != "" {
			fmt.Fprintf(output, "  (not routed: %s)\n", local.note)
		}
		return
	}
	fmt.Fprintf(output, "  %s\n", local.route.URL(local.routerPort))
	switch {
	case local.note != "":
		fmt.Fprintf(output, "  (%s)\n", local.note)
	case !localdev.HostsMapsLoopback(localdev.ReadSystemHosts(), local.route.Hostname):
		fmt.Fprintf(output, "  (%s is not mapped to 127.0.0.1; the bind address below still works)\n",
			local.route.Hostname)
	case local.routerPort != localRouterPreferredPort:
		fmt.Fprintf(output, "  (port %d was not available, so the local URL carries :%d)\n",
			localRouterPreferredPort, local.routerPort)
	}
	fmt.Fprintf(output, "  Bind: %s\n", net.JoinHostPort(local.route.BindHost, strconv.Itoa(local.route.BindPort)))
}
