package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"ahdcode/internal/localdev"
)

// The local router is what turns a registered `.test` name into something a
// browser can actually open. It is a few hundred lines of net/http, not a
// dependency and not a service: no Caddy, no nginx, nothing installed, and
// nothing left running once the AhdCode session that hosts it exits.
//
// It is emphatically not a general proxy, and three rules keep it that way:
//
//   - It listens on loopback only. Never 0.0.0.0, never an external
//     interface, so nothing off this machine can reach it at all.
//   - It forwards only to a destination the AhdCode route registry recorded,
//     and the registry only ever holds loopback destinations. A Host header
//     that is not in that allowlist is refused; there is no path by which a
//     request names its own destination.
//   - It re-checks the allowlist against live sessions, so a name never
//     keeps pointing at a port after the session that owned it is gone.
//
// v0.19 serves plaintext HTTP here. There is no local TLS, no certificate
// authority, and no ACME: introducing a locally trusted certificate is a
// change to the machine's trust store, which is a much larger decision than
// this release is making.
const (
	// localRouterPreferredPort is the port that produces the clean URL. It
	// is privileged on Unix, so binding it is attempted and never assumed.
	localRouterPreferredPort = 80

	// localRouterFallbackPort is the deterministic alternative used when
	// port 80 cannot be bound. It is fixed rather than randomly chosen so
	// the URL a person bookmarked keeps working across restarts.
	localRouterFallbackPort = 7357

	// LocalRouterPortEnvKey overrides the fallback port for an environment
	// where 7357 is already spoken for.
	localRouterPortEnvKey = "AHDCODE_LOCAL_ROUTER_PORT"

	localRouterHost         = "127.0.0.1"
	localRouterRefreshEvery = 2 * time.Second
	localRouterAcquireEvery = 3 * time.Second

	// localRouterProbePath lets one AhdCode process recognize another's
	// router rather than assuming whatever holds the port is one. On a
	// machine where some unrelated server already owns port 80, a plain
	// "something is listening" test would print a clean URL that does not
	// reach AhdCode at all.
	//
	// It answers only for a Host that is not itself a registered route, so
	// an application always keeps the whole of its own path space.
	localRouterProbePath     = "/.ahdcode-local-router"
	localRouterProbeResponse = "ahdcode-local-router/1"
)

// localRouter owns one bound listener and the allowlist it serves.
type localRouter struct {
	port     int
	listener net.Listener
	server   *http.Server
	live     localdev.LiveFunc

	mutex     sync.RWMutex
	allowed   map[string]localdev.Route
	refreshed time.Time

	stopOnce sync.Once
	stopped  chan struct{}
}

// localRouterFallbackPortValue resolves the configured fallback port. An
// unusable override is ignored rather than fatal: the router is a
// convenience, and refusing to start one because of a typo in an optional
// variable would be a worse outcome than using the documented default.
func localRouterFallbackPortValue() int {
	raw := strings.TrimSpace(os.Getenv(localRouterPortEnvKey))
	if raw == "" {
		return localRouterFallbackPort
	}
	port, err := strconv.Atoi(raw)
	if err != nil || port < 1 || port > 65535 {
		return localRouterFallbackPort
	}
	return port
}

// bindLocalRouter tries the clean port first and then the deterministic
// fallback.
//
// Nothing here elevates, retries indefinitely, or waits for a human. On a
// Unix machine an unprivileged process usually cannot bind 80 at all; that is
// a platform policy, not a failure to route around, so the fallback port is
// taken immediately and the caller reports the URL that actually works.
func bindLocalRouter() (net.Listener, int, error) {
	ports := []int{localRouterPreferredPort, localRouterFallbackPortValue()}
	var lastErr error
	for _, port := range ports {
		address := net.JoinHostPort(localRouterHost, strconv.Itoa(port))
		listener, err := net.Listen("tcp", address)
		if err == nil {
			return listener, port, nil
		}
		lastErr = err
	}
	return nil, 0, lastErr
}

// startLocalRouter binds a listener and begins serving. It returns nil, with
// no error surfaced to the user's foreground output, when no port is
// available: a session whose application started correctly must not be
// reported as broken because a convenience could not start.
func startLocalRouter(live localdev.LiveFunc) *localRouter {
	listener, port, err := bindLocalRouter()
	if err != nil {
		return nil
	}
	router := &localRouter{
		port:     port,
		listener: listener,
		live:     live,
		allowed:  map[string]localdev.Route{},
		stopped:  make(chan struct{}),
	}
	router.refresh()
	router.server = &http.Server{
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() { _ = router.server.Serve(listener) }()
	go router.refreshLoop()
	return router
}

func (router *localRouter) close() {
	router.stopOnce.Do(func() {
		close(router.stopped)
		if router.server != nil {
			context, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = router.server.Shutdown(context)
			return
		}
		_ = router.listener.Close()
	})
}

func (router *localRouter) refreshLoop() {
	ticker := time.NewTicker(localRouterRefreshEvery)
	defer ticker.Stop()
	for {
		select {
		case <-router.stopped:
			return
		case <-ticker.C:
			router.refresh()
		}
	}
}

// refresh rebuilds the allowlist from the registry, keeping only routes whose
// owning session still answers. This is the step that makes a crashed
// session's leftover name stop resolving to whatever now holds its port.
func (router *localRouter) refresh() {
	routes, err := localdev.LiveRoutes(router.live)
	if err != nil {
		return
	}
	allowed := make(map[string]localdev.Route, len(routes))
	for _, route := range routes {
		allowed[route.Hostname] = route
	}
	router.mutex.Lock()
	router.allowed = allowed
	router.refreshed = time.Now()
	router.mutex.Unlock()
}

func (router *localRouter) lookup(hostname string) (localdev.Route, bool) {
	router.mutex.RLock()
	route, ok := router.allowed[hostname]
	stale := time.Since(router.refreshed) > localRouterRefreshEvery
	router.mutex.RUnlock()
	if ok || !stale {
		return route, ok
	}
	// A name that is not in the allowlist may simply be newer than the last
	// refresh -- a session that started a moment ago. Re-read once before
	// refusing, so a freshly allocated hostname works immediately.
	router.refresh()
	router.mutex.RLock()
	route, ok = router.allowed[hostname]
	router.mutex.RUnlock()
	return route, ok
}

// requestHostname is the Host header reduced to a bare name. The port is
// dropped (a request to ahdakademi.test:7357 is still a request for
// ahdakademi.test) and the name is lowercased, because Host is
// case-insensitive and the allowlist is not.
func requestHostname(request *http.Request) string {
	host := request.Host
	if host == "" {
		return ""
	}
	if bare, _, err := net.SplitHostPort(host); err == nil {
		host = bare
	}
	return strings.ToLower(strings.Trim(strings.TrimSuffix(host, "."), "[]"))
}

func (router *localRouter) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	hostname := requestHostname(request)
	route, ok := router.lookup(hostname)
	if !ok {
		if request.URL.Path == localRouterProbePath {
			writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintln(writer, localRouterProbeResponse)
			return
		}
		router.refuse(writer, hostname)
		return
	}
	// Belt and braces: the registry already refuses a non-loopback
	// destination on write and again on load, and this is the last place it
	// could still matter, so it is checked once more before a connection is
	// made rather than trusted from storage.
	if !localdev.IsLoopbackHost(route.BindHost) {
		router.refuse(writer, hostname)
		return
	}
	router.proxyFor(route).ServeHTTP(writer, request)
}

func (router *localRouter) refuse(writer http.ResponseWriter, hostname string) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusNotFound)
	if hostname == "" {
		fmt.Fprintln(writer, "AhdCode local router: no host was requested.")
	} else {
		fmt.Fprintf(writer, "AhdCode local router: %s is not a running AhdCode local route.\n", hostname)
	}
	fmt.Fprintln(writer, "Run `ahdcode local status` to see the routes this machine currently serves.")
}

// proxyFor builds the reverse proxy for one route.
//
// The inbound Host is preserved deliberately: the application should see the
// name the person typed, so any absolute URL it builds stays on the local
// identity instead of leaking the loopback port back into a link. The
// transport dials the registry's own loopback address and refuses anything
// else, so no name resolution the request could influence ever decides where
// bytes go.
func (router *localRouter) proxyFor(route localdev.Route) *httputil.ReverseProxy {
	destination := route.Destination()
	target := &url.URL{Scheme: "http", Host: destination}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			if address != destination {
				return nil, fmt.Errorf("refusing a non-registered destination")
			}
			host, _, err := net.SplitHostPort(address)
			if err != nil || !localdev.IsLoopbackHost(host) {
				return nil, fmt.Errorf("refusing a non-loopback destination")
			}
			dialer := &net.Dialer{Timeout: 5 * time.Second}
			return dialer.DialContext(ctx, network, address)
		},
		MaxIdleConns:          16,
		IdleConnTimeout:       30 * time.Second,
		ResponseHeaderTimeout: 0,
	}
	director := proxy.Director
	proxy.Director = func(request *http.Request) {
		inboundHost := request.Host
		director(request)
		// NewSingleHostReverseProxy rewrites Host to the target; put the
		// caller's own name back.
		request.Host = inboundHost
	}
	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(writer, "AhdCode local router: %s is registered but %s did not answer.\n",
			route.Hostname, route.Destination())
	}
	return proxy
}

// localRouterHolder keeps a router running for the lifetime of one command,
// re-acquiring the port if whichever AhdCode process currently owns it exits.
//
// Only one process can hold the router port, but every AhdCode session serves
// every registered route, so which one holds it does not matter -- what
// matters is that some session does. A session that starts second simply
// keeps trying in the background and takes over the moment the first one
// stops, which is what keeps a machine's local names working across the
// ordinary come-and-go of dev sessions.
type localRouterHolder struct {
	live     localdev.LiveFunc
	mutex    sync.Mutex
	router   *localRouter
	stopOnce sync.Once
	stopped  chan struct{}
}

func newLocalRouterHolder(live localdev.LiveFunc) *localRouterHolder {
	holder := &localRouterHolder{live: live, stopped: make(chan struct{})}
	holder.router = startLocalRouter(live)
	if holder.router == nil {
		go holder.acquireLoop()
	}
	return holder
}

func (holder *localRouterHolder) acquireLoop() {
	ticker := time.NewTicker(localRouterAcquireEvery)
	defer ticker.Stop()
	for {
		select {
		case <-holder.stopped:
			return
		case <-ticker.C:
			router := startLocalRouter(holder.live)
			if router == nil {
				continue
			}
			holder.mutex.Lock()
			holder.router = router
			holder.mutex.Unlock()
			return
		}
	}
}

func (holder *localRouterHolder) close() {
	holder.stopOnce.Do(func() {
		close(holder.stopped)
		holder.mutex.Lock()
		router := holder.router
		holder.router = nil
		holder.mutex.Unlock()
		if router != nil {
			router.close()
		}
	})
}

// port is the port a URL should be printed with: the one this process bound,
// or the one whichever other AhdCode session currently holds the router is
// serving on. Zero means no router is reachable at all.
func (holder *localRouterHolder) port() int {
	holder.mutex.Lock()
	router := holder.router
	holder.mutex.Unlock()
	if router != nil {
		return router.port
	}
	return probeLocalRouterPort()
}

// probeLocalRouterPort reports which port an AhdCode local router is
// currently served on, by asking. It identifies the router rather than
// merely finding an open port: an unrelated web server already holding
// port 80 must not make AhdCode advertise a clean URL that reaches it
// instead.
func probeLocalRouterPort() int {
	client := &http.Client{Timeout: 400 * time.Millisecond}
	for _, port := range []int{localRouterPreferredPort, localRouterFallbackPortValue()} {
		address := net.JoinHostPort(localRouterHost, strconv.Itoa(port))
		response, err := client.Get("http://" + address + localRouterProbePath)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(io.LimitReader(response.Body, 128))
		_ = response.Body.Close()
		if err == nil && strings.TrimSpace(string(body)) == localRouterProbeResponse {
			return port
		}
	}
	return 0
}
