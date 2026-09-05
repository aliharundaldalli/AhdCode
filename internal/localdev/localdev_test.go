package localdev

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Every test here runs against a throwaway home. Nothing in this package's
// suite may read or write the registry of whoever is running the tests.
func isolate(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv(HomeEnvKey, home)
	return home
}

func route(hostname, descriptor string, port int) Route {
	return Route{
		Kind: KindDev, Hostname: hostname, Descriptor: descriptor,
		BindHost: "127.0.0.1", BindPort: port,
	}
}

func allLive(Route) bool  { return true }
func noneLive(Route) bool { return false }

// A. The local name drops the registrable suffix rather than appending to it,
// so a project configured for ahdakademi.com develops as ahdakademi.test.
func TestDeriveHostname(t *testing.T) {
	for _, testCase := range []struct{ appHost, expected string }{
		{"ahdakademi.com", "ahdakademi.test"},
		{"ahdakademi.com.tr", "ahdakademi.com.test"},
		{"AhdAkademi.COM", "ahdakademi.test"},
		{"ahdakademi.com:8443", "ahdakademi.test"},
		{"ahdakademi.test", "ahdakademi.test"},
		{"localhost", "localhost.test"},
		{"ahd_akademi.com", "ahdakademi.test"},
		{"ahdakademi.com.", "ahdakademi.test"},
		{"", ""},
		{"127.0.0.1", ""},
		{"::1", ""},
		{"...", ""},
	} {
		if got := DeriveHostname(testCase.appHost); got != testCase.expected {
			t.Errorf("APP_HOST=%q derived %q, expected %q", testCase.appHost, got, testCase.expected)
		}
	}
}

// B. Only names under .test are routable, and only in the shape the router
// can serve. This is the gate an entry passes before it can reach the
// allowlist, so it is deliberately strict.
func TestValidHostname(t *testing.T) {
	for _, name := range []string{"a.test", "ahdakademi.test", "www.example.test", "a-b.test"} {
		if !ValidHostname(name) {
			t.Errorf("%q was refused", name)
		}
	}
	for _, name := range []string{"", "test", ".test", "ahdakademi.com", "ahdakademi.test.", "-a.test", "a-.test", "A.test", "a b.test", "a..test"} {
		if ValidHostname(name) {
			t.Errorf("%q was accepted", name)
		}
	}
}

// C. Only a loopback IP literal is a routable destination. A hostname is
// refused even when it looks local, because a name resolves somewhere and
// what it resolves to is not this package's to decide.
func TestIsLoopbackHost(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "127.1.2.3", "::1", "[::1]"} {
		if !IsLoopbackHost(host) {
			t.Errorf("%q was refused as a loopback destination", host)
		}
	}
	for _, host := range []string{"", "localhost", "0.0.0.0", "192.168.1.10", "example.com", "10.0.0.1"} {
		if IsLoopbackHost(host) {
			t.Errorf("%q was accepted as a loopback destination", host)
		}
	}
}

// D. A second project wanting the same name is given the next suffix, and the
// walk is deterministic: the smallest free index, every time.
func TestAllocateSuffixesOnCollision(t *testing.T) {
	isolate(t)
	first, err := Allocate("ahdakademi.test", route("", "/one/app.dev", 8001), allLive)
	if err != nil {
		t.Fatal(err)
	}
	if first.Hostname != "ahdakademi.test" {
		t.Fatalf("first session received %q", first.Hostname)
	}
	second, err := Allocate("ahdakademi.test", route("", "/two/app.dev", 8002), allLive)
	if err != nil {
		t.Fatal(err)
	}
	if second.Hostname != "ahdakademi1.test" {
		t.Fatalf("second session received %q, expected ahdakademi1.test", second.Hostname)
	}
	third, err := Allocate("ahdakademi.test", route("", "/three/app.dev", 8003), allLive)
	if err != nil {
		t.Fatal(err)
	}
	if third.Hostname != "ahdakademi2.test" {
		t.Fatalf("third session received %q, expected ahdakademi2.test", third.Hostname)
	}

	// Releasing the middle name frees exactly that one; the next claimant
	// takes it rather than continuing to climb.
	if err := Release("ahdakademi1.test", "/two/app.dev"); err != nil {
		t.Fatal(err)
	}
	fourth, err := Allocate("ahdakademi.test", route("", "/four/app.dev", 8004), allLive)
	if err != nil {
		t.Fatal(err)
	}
	if fourth.Hostname != "ahdakademi1.test" {
		t.Fatalf("a released name was not reused; received %q", fourth.Hostname)
	}
}

// E. A name held by a session that is gone is reclaimed rather than avoided.
// This is what keeps a crashed dev session from pushing every future run onto
// a suffixed name forever.
func TestAllocateReclaimsAStaleRoute(t *testing.T) {
	isolate(t)
	if _, err := Allocate("ahdakademi.test", route("", "/dead/app.dev", 8001), allLive); err != nil {
		t.Fatal(err)
	}
	fresh, err := Allocate("ahdakademi.test", route("", "/live/app.dev", 8002), noneLive)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Hostname != "ahdakademi.test" {
		t.Fatalf("a stale route blocked the preferred name; received %q", fresh.Hostname)
	}
	routes, err := LoadRoutes()
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || routes[0].Descriptor != "/live/app.dev" {
		t.Fatalf("the stale entry survived: %#v", routes)
	}
}

// F. The same session restarting keeps its own name instead of climbing a
// suffix each time it rebuilds.
func TestAllocateIsIdempotentForOneDescriptor(t *testing.T) {
	isolate(t)
	for attempt := 0; attempt < 3; attempt++ {
		allocated, err := Allocate("ahdakademi.test", route("", "/one/app.dev", 8001), allLive)
		if err != nil {
			t.Fatal(err)
		}
		if allocated.Hostname != "ahdakademi.test" {
			t.Fatalf("restart %d received %q", attempt, allocated.Hostname)
		}
	}
	routes, _ := LoadRoutes()
	if len(routes) != 1 {
		t.Fatalf("restarting duplicated the entry: %#v", routes)
	}
}

// G. Concurrent starts never hand out the same hostname twice. This is the
// property the registry lock exists for.
func TestAllocateIsAtomicUnderConcurrency(t *testing.T) {
	isolate(t)
	const sessions = 8
	var group sync.WaitGroup
	results := make([]string, sessions)
	errs := make([]error, sessions)
	for index := 0; index < sessions; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			allocated, err := Allocate("ahdakademi.test",
				route("", filepath.Join("/session", string(rune('a'+index)), "app.dev"), 9000+index), allLive)
			results[index], errs[index] = allocated.Hostname, err
		}(index)
	}
	group.Wait()

	seen := map[string]bool{}
	for index, hostname := range results {
		if errs[index] != nil {
			t.Fatalf("session %d failed: %v", index, errs[index])
		}
		if seen[hostname] {
			t.Fatalf("hostname %q was handed to two sessions", hostname)
		}
		seen[hostname] = true
	}
	routes, err := LoadRoutes()
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != sessions {
		t.Fatalf("expected %d recorded routes, found %d", sessions, len(routes))
	}
}

// H. A destination that is not loopback is refused on write, and a registry
// that somehow contains one is refused again on read. The router's allowlist
// can therefore never point off this machine, even if the file is edited by
// hand.
func TestRoutesRefuseNonLoopbackDestinations(t *testing.T) {
	home := isolate(t)
	bad := route("", "/one/app.dev", 8001)
	bad.BindHost = "192.168.1.20"
	if _, err := Allocate("ahdakademi.test", bad, allLive); err == nil {
		t.Fatal("a non-loopback destination was registered")
	}

	path := filepath.Join(home, "routes.json")
	handEdited := `{"schema":"ahdcode.localroutes","version":1,"routes":[
	  {"hostname":"evil.test","kind":"dev","bindHost":"203.0.113.9","bindPort":80},
	  {"hostname":"ok.test","kind":"dev","bindHost":"127.0.0.1","bindPort":8080}]}`
	if err := os.WriteFile(path, []byte(handEdited), 0o600); err != nil {
		t.Fatal(err)
	}
	routes, err := LoadRoutes()
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || routes[0].Hostname != "ok.test" {
		t.Fatalf("a hand-edited remote destination survived loading: %#v", routes)
	}
}

// I. Release only removes an entry that is still this session's own, so a
// late-exiting session cannot delete the route of whoever took its name next.
func TestReleaseOnlyRemovesItsOwnEntry(t *testing.T) {
	isolate(t)
	if _, err := Allocate("ahdakademi.test", route("", "/live/app.dev", 8001), allLive); err != nil {
		t.Fatal(err)
	}
	if err := Release("ahdakademi.test", "/somebody-else/app.dev"); err != nil {
		t.Fatal(err)
	}
	routes, _ := LoadRoutes()
	if len(routes) != 1 {
		t.Fatalf("another session's release removed a live route: %#v", routes)
	}
}

// J. Prune is the safety net for sessions that never got to release.
func TestPruneRemovesOnlyDeadRoutes(t *testing.T) {
	isolate(t)
	if _, err := Allocate("alive.test", route("", "/alive/app.dev", 8001), allLive); err != nil {
		t.Fatal(err)
	}
	if _, err := Allocate("dead.test", route("", "/dead/app.dev", 8002), allLive); err != nil {
		t.Fatal(err)
	}
	removed, err := Prune(func(candidate Route) bool { return candidate.Hostname == "alive.test" })
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("pruned %d routes, expected 1", removed)
	}
	running, stale, err := PartitionRoutes(allLive)
	if err != nil {
		t.Fatal(err)
	}
	if len(running) != 1 || len(stale) != 0 || running[0].Hostname != "alive.test" {
		t.Fatalf("running=%#v stale=%#v", running, stale)
	}
}

// K. The logical URL carries the router's port only when the router is not
// on 80, and never carries the bind port -- the two are different facts.
func TestRouteURLSeparatesLogicalFromBind(t *testing.T) {
	entry := route("ahdakademi.test", "/one/app.dev", 18437)
	if got := entry.URL(80); got != "http://ahdakademi.test/" {
		t.Errorf("URL on port 80 was %q", got)
	}
	if got := entry.URL(7357); got != "http://ahdakademi.test:7357/" {
		t.Errorf("URL on the fallback port was %q", got)
	}
	if got := entry.Destination(); got != "127.0.0.1:18437" {
		t.Errorf("destination was %q", got)
	}
}

// L. Registering the same file twice, however it is spelled, leaves one
// entry. Canonicalization is what makes "the same database" mean the same
// file rather than the same string.
func TestAddSQLiteDeduplicates(t *testing.T) {
	isolate(t)
	project := t.TempDir()
	database := filepath.Join(project, "app.db")
	if err := os.WriteFile(database, []byte("not really sqlite, but a real file"), 0o600); err != nil {
		t.Fatal(err)
	}

	// On a machine where the temporary directory is itself a symlink
	// (/var -> /private/var on macOS), the canonical path is the resolved
	// one. That is the point of canonicalizing, so the expectation follows
	// it rather than the spelling the test happened to use.
	canonical, err := CanonicalDatabasePath(database)
	if err != nil {
		t.Fatal(err)
	}
	entry, added, err := AddSQLite(database, project)
	if err != nil || !added {
		t.Fatalf("first add: added=%v err=%v", added, err)
	}
	if entry.Driver != DriverSQLite || entry.Path != canonical {
		t.Fatalf("unexpected entry %#v", entry)
	}

	for _, spelling := range []string{
		database,
		filepath.Join(project, ".", "app.db"),
		filepath.Join(project, "nested", "..", "app.db"),
	} {
		_, added, err := AddSQLite(spelling, project)
		if err != nil {
			t.Fatalf("re-adding %q failed: %v", spelling, err)
		}
		if added {
			t.Fatalf("%q was registered a second time", spelling)
		}
	}
	databases, err := LoadDatabases()
	if err != nil {
		t.Fatal(err)
	}
	if len(databases) != 1 {
		t.Fatalf("expected one entry, found %#v", databases)
	}
}

// M. A path that does not exist is refused: the registry describes databases,
// it does not create them.
func TestAddSQLiteRequiresAnExistingFile(t *testing.T) {
	isolate(t)
	missing := filepath.Join(t.TempDir(), "nope.db")
	if _, _, err := AddSQLite(missing, ""); err == nil {
		t.Fatal("a missing file was registered")
	} else if !strings.Contains(err.Error(), "no such database file") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, _, err := AddSQLite(t.TempDir(), ""); err == nil {
		t.Fatal("a directory was registered")
	}
}

// N. Removing an entry forgets it and does nothing else. The file on disk is
// untouched -- byte for byte -- because forgetting a database and destroying
// one are different operations.
func TestRemoveSQLiteNeverTouchesTheFile(t *testing.T) {
	isolate(t)
	project := t.TempDir()
	database := filepath.Join(project, "app.db")
	original := []byte("precious rows")
	if err := os.WriteFile(database, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := AddSQLite(database, project); err != nil {
		t.Fatal(err)
	}

	canonical, err := CanonicalDatabasePath(database)
	if err != nil {
		t.Fatal(err)
	}
	removed, found, err := RemoveSQLite(database)
	if err != nil || !found {
		t.Fatalf("remove: found=%v err=%v", found, err)
	}
	if removed.Path != canonical {
		t.Fatalf("removed %q", removed.Path)
	}
	content, err := os.ReadFile(database)
	if err != nil {
		t.Fatalf("the database file was disturbed: %v", err)
	}
	if string(content) != string(original) {
		t.Fatalf("the database file content changed to %q", content)
	}
	databases, _ := LoadDatabases()
	if len(databases) != 0 {
		t.Fatalf("the entry survived removal: %#v", databases)
	}
	if _, found, _ := RemoveSQLite(database); found {
		t.Fatal("removing an unregistered database reported a match")
	}
}

// O. A registered file that is not currently there is reported as
// unavailable, not silently dropped: an unmounted volume is not a withdrawal.
func TestUnavailableDatabaseSurvivesInTheRegistry(t *testing.T) {
	isolate(t)
	project := t.TempDir()
	database := filepath.Join(project, "app.db")
	if err := os.WriteFile(database, []byte("rows"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := AddSQLite(database, project); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(database); err != nil {
		t.Fatal(err)
	}
	databases, err := LoadDatabases()
	if err != nil {
		t.Fatal(err)
	}
	if len(databases) != 1 {
		t.Fatalf("a temporarily missing database was dropped: %#v", databases)
	}
	if databases[0].Available() {
		t.Fatal("a missing database reported itself available")
	}
}

// P. The registry file itself is JSON with a schema AhdDataStudio can read,
// and it holds no credential of any kind.
func TestDatabaseRegistryShape(t *testing.T) {
	home := isolate(t)
	project := t.TempDir()
	database := filepath.Join(project, "app.db")
	if err := os.WriteFile(database, []byte("rows"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := AddSQLite(database, project); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(home, "databases.json"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	canonical, err := CanonicalDatabasePath(database)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"schema": "ahdcode.localdatabases"`, `"driver": "sqlite"`, canonical} {
		if !strings.Contains(content, expected) {
			t.Errorf("the registry did not contain %q:\n%s", expected, content)
		}
	}
	for _, forbidden := range []string{"password", "token", "secret", "AHD_DATA_MYSQL"} {
		if strings.Contains(strings.ToLower(content), strings.ToLower(forbidden)) {
			t.Errorf("the registry contained %q", forbidden)
		}
	}
	info, err := os.Stat(filepath.Join(home, "databases.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("registry permissions were %v", info.Mode().Perm())
	}
}

// Q. The managed hosts block is added, replaced, and removed without ever
// disturbing a line outside its markers. This runs entirely on strings and
// temporary files; the real /etc/hosts is never opened.
func TestHostsBlockLeavesEverythingElseAlone(t *testing.T) {
	original := "##\n# Host Database\n##\n127.0.0.1\tlocalhost\n255.255.255.255\tbroadcasthost\n::1 localhost\n" +
		"10.0.0.5 build.internal\n"

	added := ApplyHostsBlock(original, RenderHostsBlock([]string{"ahdakademi.test", "ahddatabasestudio.test"}))
	if !strings.HasPrefix(added, original) {
		t.Fatalf("the existing file was not preserved verbatim:\n%s", added)
	}
	for _, expected := range []string{
		"# BEGIN AHDCODE LOCAL",
		"127.0.0.1 ahdakademi.test",
		"127.0.0.1 ahddatabasestudio.test",
		"# END AHDCODE LOCAL",
	} {
		if !strings.Contains(added, expected) {
			t.Errorf("the managed block did not contain %q:\n%s", expected, added)
		}
	}

	// Replacing the block keeps its position and leaves the rest untouched.
	replaced := ApplyHostsBlock(added, RenderHostsBlock([]string{"ahdakademi.test"}))
	if strings.Contains(replaced, "ahddatabasestudio.test") {
		t.Errorf("a dropped name survived the rewrite:\n%s", replaced)
	}
	if !strings.Contains(replaced, "10.0.0.5 build.internal") {
		t.Errorf("an unrelated entry was lost:\n%s", replaced)
	}
	if strings.Count(replaced, hostsBeginMarker) != 1 {
		t.Errorf("the block was duplicated:\n%s", replaced)
	}

	// Removing the block restores exactly the original file.
	removed := ApplyHostsBlock(replaced, "")
	if removed != original {
		t.Errorf("removal did not restore the original file:\n%q\nvs\n%q", removed, original)
	}
	if ApplyHostsBlock(original, "") != original {
		t.Error("removing an absent block changed the file")
	}
}

// R. Only names AhdCode is willing to route reach the block, and only mapped
// to loopback.
func TestRenderHostsBlockRefusesEverythingElse(t *testing.T) {
	block := RenderHostsBlock([]string{"ahdakademi.test", "example.com", "", "127.0.0.1", "AHDAKADEMI.TEST"})
	if strings.Contains(block, "example.com") || strings.Contains(block, "127.0.0.1 127.0.0.1") {
		t.Errorf("an unroutable name reached the block:\n%s", block)
	}
	if strings.Count(block, "ahdakademi.test") != 1 {
		t.Errorf("a name was duplicated by case:\n%s", block)
	}
	for _, line := range strings.Split(block, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "127.0.0.1 ") {
			t.Errorf("a non-loopback mapping was written: %q", line)
		}
	}
	if RenderHostsBlock(nil) != "" {
		t.Error("an empty name set produced a block")
	}
}

// S. A half-written block from an interrupted edit is replaced rather than
// duplicated.
func TestApplyHostsBlockRepairsAnUnterminatedBlock(t *testing.T) {
	broken := "127.0.0.1 localhost\n# BEGIN AHDCODE LOCAL\n127.0.0.1 old.test\n"
	fixed := ApplyHostsBlock(broken, RenderHostsBlock([]string{"new.test"}))
	if strings.Count(fixed, hostsBeginMarker) != 1 || strings.Contains(fixed, "old.test") {
		t.Fatalf("an unterminated block was not repaired:\n%s", fixed)
	}
	if !strings.Contains(fixed, "127.0.0.1 localhost") {
		t.Fatalf("an unrelated entry was lost:\n%s", fixed)
	}
}

// T. The block's own names are readable back out, which is what lets the
// union of "already managed" and "running now" be computed.
func TestHostsBlockNames(t *testing.T) {
	content := ApplyHostsBlock("127.0.0.1 localhost\n", RenderHostsBlock([]string{"b.test", "a.test"}))
	names := HostsBlockNames(content)
	if len(names) != 2 || names[0] != "a.test" || names[1] != "b.test" {
		t.Fatalf("names were %#v", names)
	}
	if HostsBlockNames("127.0.0.1 localhost\n") != nil {
		t.Fatal("names were reported for a file with no managed block")
	}
}
