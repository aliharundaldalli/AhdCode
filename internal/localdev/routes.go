package localdev

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The route registry is the single source of truth for which local `.test`
// names AhdCode currently owns and where each one points.
//
// Two properties matter and everything below exists to keep them:
//
//   - It decides ownership, not /etc/hosts. A hosts entry left behind by a
//     project that is no longer running is a harmless stale mapping, not a
//     live application, so it never pushes a new session onto a suffixed
//     name. Only a registry entry backed by a controller that answers its
//     own authenticated control channel counts as taken.
//
//   - It never grants authority. An entry records a loopback port and the
//     path of the descriptor that owns it; it carries no token and no
//     ability to signal anything. Liveness is always re-derived from the
//     descriptor, never assumed from the recorded pid, which the OS is free
//     to reuse.
const (
	routesSchema  = "ahdcode.localroutes"
	routesVersion = 1

	// KindDev is an `ahdcode dev` session's application route.
	KindDev = "dev"
	// KindStudio is AhdDataStudio's fixed route.
	KindStudio = "studio"

	// maxHostnameSuffix bounds the collision walk. Reaching it means
	// something is registering routes in a loop, which is a bug to report
	// rather than a search to continue.
	maxHostnameSuffix = 999
)

// Route is one local hostname and the loopback destination behind it.
//
// BindHost/BindPort are the actual socket the application listens on and are
// kept strictly separate from Hostname/URL: the logical identity a person
// types and the address a socket is bound to are different facts, and
// merging them is what makes a development URL start lying after a port
// changes.
type Route struct {
	Hostname string `json:"hostname"`
	Kind     string `json:"kind"`

	// Source and ProjectRoot identify whose route this is, for diagnostics
	// and for the stop path that releases it.
	Source      string `json:"source,omitempty"`
	ProjectRoot string `json:"projectRoot,omitempty"`

	// Descriptor is the .dev/.run file whose authenticated control channel
	// decides whether this route is still live. Empty for a route whose
	// liveness is established some other way (Studio's own port).
	Descriptor string `json:"descriptor,omitempty"`

	BindHost string `json:"bindHost"`
	BindPort int    `json:"bindPort"`

	// BasePath is displayed with the destination and is never used to
	// rewrite a proxied request: the router forwards paths unchanged.
	BasePath string `json:"basePath,omitempty"`

	ControlPort int    `json:"controlPort,omitempty"`
	OwnerPID    int    `json:"ownerPid,omitempty"`
	StartedAt   string `json:"startedAt,omitempty"`
}

// URL is the logical address a person opens. routerPort is the port the
// local router actually managed to bind; 80 is omitted so the clean form is
// printed whenever it is the truth.
func (route Route) URL(routerPort int) string {
	if routerPort == 80 || routerPort == 0 {
		return "http://" + route.Hostname + "/"
	}
	return fmt.Sprintf("http://%s:%d/", route.Hostname, routerPort)
}

// Destination is the loopback address the router forwards to.
func (route Route) Destination() string {
	return fmt.Sprintf("%s:%d", route.BindHost, route.BindPort)
}

// DisplayDestination adds the base path, for status output only.
func (route Route) DisplayDestination() string {
	if route.BasePath == "" {
		return route.Destination()
	}
	return route.Destination() + route.BasePath
}

// Valid rejects anything the router must not be asked to serve. It runs on
// load as well as on write, so a hand-edited or corrupted registry cannot
// widen what the router will reach: a non-loopback destination is dropped
// rather than honoured.
func (route Route) Valid() bool {
	if !ValidHostname(route.Hostname) {
		return false
	}
	if route.Kind != KindDev && route.Kind != KindStudio {
		return false
	}
	if !IsLoopbackHost(route.BindHost) {
		return false
	}
	if route.BindPort <= 0 || route.BindPort > 65535 {
		return false
	}
	if route.BasePath != "" && !strings.HasPrefix(route.BasePath, "/") {
		return false
	}
	return true
}

type routeFile struct {
	Schema  string  `json:"schema"`
	Version int     `json:"version"`
	Routes  []Route `json:"routes"`
}

// LiveFunc reports whether a recorded route still has a running owner. The
// route registry never answers this itself: the authoritative test is the
// descriptor's own authenticated control channel, which lives with the
// command that owns it.
type LiveFunc func(Route) bool

func loadRoutesFrom(path string) ([]Route, error) {
	var file routeFile
	present, err := readJSON(path, &file)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, nil
	}
	if file.Schema != routesSchema {
		return nil, fmt.Errorf("%s is not an AhdCode route registry", path)
	}
	if file.Version != routesVersion {
		return nil, fmt.Errorf("unsupported AhdCode route registry version %d", file.Version)
	}
	kept := make([]Route, 0, len(file.Routes))
	seen := map[string]bool{}
	for _, route := range file.Routes {
		if !route.Valid() || seen[route.Hostname] {
			continue
		}
		seen[route.Hostname] = true
		kept = append(kept, route)
	}
	return kept, nil
}

func saveRoutesTo(path string, routes []Route) error {
	sort.Slice(routes, func(i, j int) bool { return routes[i].Hostname < routes[j].Hostname })
	return writeAtomic(path, routeFile{Schema: routesSchema, Version: routesVersion, Routes: routes})
}

// LoadRoutes returns every recorded route, valid ones only, without deciding
// whether any of them is still live.
func LoadRoutes() ([]Route, error) {
	path, err := RoutesPath()
	if err != nil {
		return nil, err
	}
	return loadRoutesFrom(path)
}

// PartitionRoutes splits the registry into the routes whose owner is still
// answering and the leftovers from sessions that are gone.
func PartitionRoutes(live LiveFunc) (running, stale []Route, err error) {
	routes, err := LoadRoutes()
	if err != nil {
		return nil, nil, err
	}
	for _, route := range routes {
		if live != nil && live(route) {
			running = append(running, route)
			continue
		}
		stale = append(stale, route)
	}
	return running, stale, nil
}

// LiveRoutes is the router's allowlist: only routes with a live owner are
// served, so a leftover entry can never keep a hostname pointed at a port
// some unrelated program has since been given.
func LiveRoutes(live LiveFunc) ([]Route, error) {
	running, _, err := PartitionRoutes(live)
	return running, err
}

// Allocate claims the first free local hostname for one session and records
// it, replacing any entry whose owner is no longer live.
//
// The whole read-decide-write cycle happens under the registry lock, so two
// `ahdcode dev` starts racing for the same APP_HOST cannot both observe the
// preferred name as free: the loser sees the winner's entry and takes the
// next suffix.
func Allocate(preferred string, request Route, live LiveFunc) (Route, error) {
	if !ValidHostname(preferred) {
		return Route{}, fmt.Errorf("%q is not a usable local hostname", preferred)
	}
	request.Hostname = preferred
	if !request.Valid() {
		return Route{}, fmt.Errorf("refusing to register a route to %s", request.Destination())
	}

	path, err := RoutesPath()
	if err != nil {
		return Route{}, err
	}
	var allocated Route
	err = withLock(path, func() error {
		routes, err := loadRoutesFrom(path)
		if err != nil {
			return err
		}
		taken := map[string]bool{}
		kept := make([]Route, 0, len(routes))
		for _, route := range routes {
			// This session's own previous entry (same descriptor) is
			// always reclaimed: a restart must not walk up the suffixes.
			if request.Descriptor != "" && route.Descriptor == request.Descriptor {
				continue
			}
			if live != nil && !live(route) {
				continue
			}
			taken[route.Hostname] = true
			kept = append(kept, route)
		}
		request.Hostname = ""
		for index := 0; index <= maxHostnameSuffix; index++ {
			candidate := SuffixHostname(preferred, index)
			if taken[candidate] {
				continue
			}
			request.Hostname = candidate
			break
		}
		if request.Hostname == "" {
			return fmt.Errorf("no free local hostname derived from %s", preferred)
		}
		request.StartedAt = nowStamp()
		allocated = request
		return saveRoutesTo(path, append(kept, request))
	})
	if err != nil {
		return Route{}, err
	}
	return allocated, nil
}

// Release drops one session's route. The descriptor must match the recorded
// one, so a session that has already been replaced by a newer owner of the
// same hostname cannot delete that owner's entry on its way out.
func Release(hostname, descriptor string) error {
	path, err := RoutesPath()
	if err != nil {
		return err
	}
	return withLock(path, func() error {
		routes, err := loadRoutesFrom(path)
		if err != nil {
			return err
		}
		kept := make([]Route, 0, len(routes))
		for _, route := range routes {
			if route.Hostname == hostname && route.Descriptor == descriptor {
				continue
			}
			kept = append(kept, route)
		}
		if len(kept) == len(routes) {
			return nil
		}
		return saveRoutesTo(path, kept)
	})
}

// Prune removes every entry whose owner is gone. It is the safety net that
// keeps a crashed session from holding a name forever; ordinary shutdown
// releases its own route explicitly.
func Prune(live LiveFunc) (int, error) {
	path, err := RoutesPath()
	if err != nil {
		return 0, err
	}
	removed := 0
	err = withLock(path, func() error {
		routes, err := loadRoutesFrom(path)
		if err != nil {
			return err
		}
		kept := make([]Route, 0, len(routes))
		for _, route := range routes {
			if live != nil && !live(route) {
				removed++
				continue
			}
			kept = append(kept, route)
		}
		if removed == 0 {
			return nil
		}
		return saveRoutesTo(path, kept)
	})
	if err != nil {
		return 0, err
	}
	return removed, nil
}

// DescriptorExists is the cheap half of a liveness test: a descriptor file
// that is gone means the session that owned it finished. Callers still ask
// the control channel before concluding a surviving file is live.
func DescriptorExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Lstat(filepath.Clean(path))
	return err == nil && info.Mode().IsRegular()
}
