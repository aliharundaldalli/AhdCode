package build

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"time"

	backend "ahdcode/internal/backend/golang"
)

// runCacheVersion namespaces the cache layout and the set of key inputs. It
// changes whenever either changes, so entries written by an older compiler are
// never reused rather than being silently reinterpreted.
const runCacheVersion = "run-v1"

// The persistent cache is bounded by both entry count and total size, so it
// can never grow without end and never needs manual cleanup. A generated
// executable is a few megabytes, and an edit-run cycle reuses a handful of
// programs, so a small bound is enough to keep the common case warm.
const (
	runCacheLimit = 16
	runCacheBytes = 64 << 20
)

// disableRunCache lets a user or a test opt out of the executable cache and
// take the ordinary build-and-discard path.
const disableRunCache = "AHDCODE_NO_RUN_CACHE"

// buildEnvironment is the set of environment variables that can change what
// the Go toolchain produces from identical sources. They are part of the cache
// identity so a differently configured build is never reused.
var buildEnvironment = []string{
	"GOOS", "GOARCH", "GOARM", "GOAMD64", "GOFLAGS", "GOEXPERIMENT", "CGO_ENABLED",
}

// runCache stores native executables of already-built generated programs.
//
// Rebuilding an unchanged program is pure waste twice over: the Go build runs
// again, and the operating system revalidates a newly written executable the
// first time it is launched. A cache hit avoids both, and it is safe because
// the key covers the complete generated program text — which is itself a
// deterministic function of every AhdCode source, every imported module, and
// the compiler's own code generation. Compilation, semantic analysis, and
// diagnostics still run in full on every invocation; only the native build is
// reused.
type runCache struct{ directory string }

// openRunCache prepares the per-user cache directory. It returns nil whenever
// caching is unavailable or disabled, and the caller then takes the ordinary
// path: the cache is an optimization and never a requirement.
func openRunCache() *runCache {
	if os.Getenv(disableRunCache) != "" {
		return nil
	}
	root, err := os.UserCacheDir()
	if err != nil {
		return nil
	}
	directory := filepath.Join(root, "ahdcode", runCacheVersion)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil
	}
	return &runCache{directory: directory}
}

// key is the identity of one built executable. It covers everything that can
// change the produced binary: the full generated source, the compiler that
// generated it, the Go toolchain that builds it, the target platform, and the
// build environment. Anything the key does not cover cannot affect the result.
func (cache *runCache) key(program *backend.GeneratedProgram, toolchain string) string {
	digest := sha256.New()
	write := func(part string) {
		var size [8]byte
		binary.LittleEndian.PutUint64(size[:], uint64(len(part)))
		_, _ = digest.Write(size[:])
		_, _ = digest.Write([]byte(part))
	}
	write(runCacheVersion)
	write(runtime.GOOS)
	write(runtime.GOARCH)
	// The compiler binary's own identity. Code generation changes usually show
	// up in the generated text below, but this also separates two compilers
	// that differ only in how they drive the build.
	write(fileIdentity(compilerPath()))
	write(fileIdentity(toolchain))
	for _, name := range buildEnvironment {
		write(name)
		write(os.Getenv(name))
	}
	write(strconv.Itoa(len(program.Files)))
	for _, file := range program.Files {
		write(file.Name)
		write(file.Content)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func compilerPath() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	return path
}

// fileIdentity describes a binary precisely enough to notice that it was
// replaced, without paying for a full content hash of a large executable.
func fileIdentity(path string) string {
	if path == "" {
		return "unknown"
	}
	info, err := os.Stat(path)
	if err != nil {
		return "missing:" + path
	}
	return path + "\x00" + strconv.FormatInt(info.Size(), 10) + "\x00" +
		strconv.FormatInt(info.ModTime().UnixNano(), 10)
}

func (cache *runCache) path(key string) string {
	return filepath.Join(cache.directory, key)
}

// lookup reports an existing executable for this key. A missing or unusable
// entry is simply a miss.
func (cache *runCache) lookup(key string) (string, bool) {
	executable := cache.path(key)
	info, err := os.Stat(executable)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return "", false
	}
	// Touch the entry so eviction keeps what is actually being run.
	now := time.Now()
	_ = os.Chtimes(executable, now, now)
	return executable, true
}

// reserve names a not-yet-built executable inside the cache directory. The
// build writes there directly, so publishing never copies the binary and never
// crosses a filesystem boundary.
func (cache *runCache) reserve() (string, bool) {
	temporary, err := os.CreateTemp(cache.directory, "partial-")
	if err != nil {
		return "", false
	}
	name := temporary.Name()
	_ = temporary.Close()
	// go build replaces this path; the placeholder only reserves the name.
	_ = os.Remove(name)
	return name, true
}

// publish moves a freshly built executable to its key in one step, so a
// concurrent run either sees no entry at all or sees a complete one. A failure
// to publish is not a run failure: the caller keeps using what it just built.
func (cache *runCache) publish(key, built string) (string, bool) {
	if err := os.Chmod(built, 0o700); err != nil {
		return "", false
	}
	final := cache.path(key)
	if err := os.Rename(built, final); err != nil {
		return "", false
	}
	if _, usable := cache.lookup(key); !usable {
		return "", false
	}
	cache.prune()
	return final, true
}

// prune keeps the most recently used entries that fit inside both bounds and
// removes the rest, so the directory stays small without any user maintenance.
func (cache *runCache) prune() {
	entries, err := os.ReadDir(cache.directory)
	if err != nil {
		return
	}
	type aged struct {
		name string
		used int64
		size int64
	}
	items := make([]aged, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || entry.IsDir() {
			continue
		}
		items = append(items, aged{name: entry.Name(), used: info.ModTime().UnixNano(), size: info.Size()})
	}
	// Most recently used first, so the entries that survive are the ones an
	// edit-run cycle is actually reusing.
	sort.Slice(items, func(left, right int) bool { return items[left].used > items[right].used })
	total := int64(0)
	for index, item := range items {
		total += item.size
		if index < runCacheLimit && total <= runCacheBytes {
			continue
		}
		_ = os.Remove(filepath.Join(cache.directory, item.name))
	}
}
