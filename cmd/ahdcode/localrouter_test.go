package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"ahdcode/internal/localdev"
)

// backend starts a loopback application for the router to forward to and
// returns its host and port.
func backend(t *testing.T, handler http.HandlerFunc) (string, int) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatal(err)
	}
	return parsed.Hostname(), port
}

func routerServing(routes ...localdev.Route) *localRouter {
	allowed := make(map[string]localdev.Route, len(routes))
	for _, route := range routes {
		allowed[route.Hostname] = route
	}
	return &localRouter{
		allowed:   allowed,
		refreshed: time.Now(),
		live:      func(localdev.Route) bool { return true },
		stopped:   make(chan struct{}),
	}
}

// A. An allowlisted host reaches its application with the request intact:
// method, path, query, headers, and body all arrive unchanged, and the
// application sees the name the caller typed rather than the loopback port.
func TestRouterForwardsAnAllowlistedHostFaithfully(t *testing.T) {
	var seen *http.Request
	var seenBody []byte
	host, port := backend(t, func(writer http.ResponseWriter, request *http.Request) {
		seen = request
		seenBody, _ = io.ReadAll(request.Body)
		writer.Header().Set("X-Backend", "yes")
		writer.WriteHeader(http.StatusTeapot)
		_, _ = writer.Write([]byte("served by the application"))
	})

	router := routerServing(localdev.Route{
		Hostname: "ahdakademi.test", Kind: localdev.KindDev,
		BindHost: host, BindPort: port,
	})

	request := httptest.NewRequest(http.MethodPost, "/notes?page=2&q=a+b", strings.NewReader("title=hello"))
	request.Host = "ahdakademi.test"
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-Custom", "kept")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTeapot {
		t.Fatalf("status was %d; the request did not reach the application", recorder.Code)
	}
	if recorder.Header().Get("X-Backend") != "yes" {
		t.Error("the application's response headers were not passed back")
	}
	if !strings.Contains(recorder.Body.String(), "served by the application") {
		t.Errorf("the body was not passed back: %s", recorder.Body.String())
	}
	if seen == nil {
		t.Fatal("the application received nothing")
	}
	if seen.Method != http.MethodPost {
		t.Errorf("method arrived as %s", seen.Method)
	}
	if seen.URL.Path != "/notes" || seen.URL.RawQuery != "page=2&q=a+b" {
		t.Errorf("path/query arrived as %s?%s", seen.URL.Path, seen.URL.RawQuery)
	}
	if string(seenBody) != "title=hello" {
		t.Errorf("body arrived as %q", seenBody)
	}
	if seen.Header.Get("X-Custom") != "kept" {
		t.Error("a request header was dropped")
	}
	if seen.Host != "ahdakademi.test" {
		t.Errorf("the application saw Host %q; it should see the name that was typed", seen.Host)
	}
}

// B. A host that is not in the allowlist is refused, and nothing is dialled
// on its behalf. This is the property that keeps the router from being a
// general proxy: a request cannot name its own destination.
func TestRouterRefusesAnUnknownHost(t *testing.T) {
	reached := false
	host, port := backend(t, func(http.ResponseWriter, *http.Request) { reached = true })

	router := routerServing(localdev.Route{
		Hostname: "ahdakademi.test", Kind: localdev.KindDev, BindHost: host, BindPort: port,
	})

	for _, hostname := range []string{"example.com", "somebodyelse.test", "127.0.0.1:8080", ""} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Host = hostname
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNotFound {
			t.Errorf("Host %q produced status %d, expected 404", hostname, recorder.Code)
		}
	}
	if reached {
		t.Fatal("a refused request still reached an application")
	}
}

// C. A registry entry whose destination is not loopback is refused at the
// last moment too, so even a hand-edited registry that slipped past loading
// cannot make the router open a connection off this machine.
func TestRouterRefusesANonLoopbackDestination(t *testing.T) {
	router := routerServing(localdev.Route{
		Hostname: "leak.test", Kind: localdev.KindDev,
		BindHost: "203.0.113.9", BindPort: 80,
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Host = "leak.test"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("a remote destination produced status %d, expected 404", recorder.Code)
	}
}

// D. The router identifies itself on a fixed path, so one AhdCode process can
// tell whether the port is held by another AhdCode router or by an unrelated
// server. An allowlisted host keeps its whole path space.
func TestRouterProbeDoesNotShadowAnApplicationPath(t *testing.T) {
	reached := false
	host, port := backend(t, func(writer http.ResponseWriter, request *http.Request) {
		reached = true
		_, _ = writer.Write([]byte("application"))
	})
	router := routerServing(localdev.Route{
		Hostname: "ahdakademi.test", Kind: localdev.KindDev, BindHost: host, BindPort: port,
	})

	direct := httptest.NewRequest(http.MethodGet, localRouterProbePath, nil)
	direct.Host = "127.0.0.1:7357"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, direct)
	if !strings.Contains(recorder.Body.String(), localRouterProbeResponse) {
		t.Errorf("the router did not identify itself: %s", recorder.Body.String())
	}

	viaRoute := httptest.NewRequest(http.MethodGet, localRouterProbePath, nil)
	viaRoute.Host = "ahdakademi.test"
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, viaRoute)
	if !reached {
		t.Error("the probe path shadowed an application's own path")
	}
	if strings.Contains(recorder.Body.String(), localRouterProbeResponse) {
		t.Error("the router answered on behalf of an application")
	}
}

// E. The allowlist comes from the registry and only ever contains routes
// whose owner is live.
func TestRouterAllowlistFollowsLiveRoutesOnly(t *testing.T) {
	t.Setenv(localdev.HomeEnvKey, t.TempDir())
	for _, hostname := range []string{"alive.test", "dead.test"} {
		if _, err := localdev.Allocate(hostname, localdev.Route{
			Kind: localdev.KindDev, Descriptor: "/" + hostname + "/app.dev",
			BindHost: "127.0.0.1", BindPort: 9100,
		}, func(localdev.Route) bool { return true }); err != nil {
			t.Fatal(err)
		}
	}
	router := &localRouter{
		allowed: map[string]localdev.Route{},
		live:    func(route localdev.Route) bool { return route.Hostname == "alive.test" },
		stopped: make(chan struct{}),
	}
	router.refresh()
	if _, ok := router.lookup("alive.test"); !ok {
		t.Error("a live route was not served")
	}
	if _, ok := router.lookup("dead.test"); ok {
		t.Error("a route whose owner is gone was still served")
	}
}

// F. A Host header with a port, a trailing dot, or different casing names the
// same route: Host is case-insensitive and the allowlist is not.
func TestRequestHostnameNormalizes(t *testing.T) {
	for _, testCase := range []struct{ header, expected string }{
		{"ahdakademi.test", "ahdakademi.test"},
		{"AhdAkademi.Test", "ahdakademi.test"},
		{"ahdakademi.test:7357", "ahdakademi.test"},
		{"ahdakademi.test.", "ahdakademi.test"},
		{"", ""},
	} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Host = testCase.header
		if got := requestHostname(request); got != testCase.expected {
			t.Errorf("Host %q normalized to %q, expected %q", testCase.header, got, testCase.expected)
		}
	}
}

// G. The fallback port is deterministic and only an override in range moves
// it, so a bookmarked local URL keeps working across restarts.
func TestLocalRouterFallbackPortIsDeterministic(t *testing.T) {
	t.Setenv(localRouterPortEnvKey, "")
	_ = os.Unsetenv(localRouterPortEnvKey)
	if got := localRouterFallbackPortValue(); got != localRouterFallbackPort {
		t.Errorf("default fallback port was %d", got)
	}
	t.Setenv(localRouterPortEnvKey, "9099")
	if got := localRouterFallbackPortValue(); got != 9099 {
		t.Errorf("override produced %d", got)
	}
	for _, bad := range []string{"nonsense", "0", "70000", "-1"} {
		t.Setenv(localRouterPortEnvKey, bad)
		if got := localRouterFallbackPortValue(); got != localRouterFallbackPort {
			t.Errorf("override %q produced %d instead of the default", bad, got)
		}
	}
}
