package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"ahdcode/internal/build"
)

// ahdcode dev is process orchestration around the existing build pipeline
// (build.BuildProgram, the same function `ahdcode build` calls -- see
// section 3), not a second compiler or a relaxed one. One goroutine, the
// event loop in devController.run, is the only thing that ever reads or
// writes controller/child state: the watcher, the control-channel server,
// and the signal handler only ever *send* events into it. That single-writer
// design is what makes "builds never overlap", "shutdown cannot race into
// starting a new child", and "pending rebuild state is synchronized" true by
// construction rather than by careful locking -- see section 36.
const shutdownGracePeriod = 3 * time.Second

type devEventKind int

const (
	eventSourceChanged devEventKind = iota
	eventBuildFinished
	eventChildExited
	eventStopRequested
)

type devControllerEvent struct {
	kind devEventKind

	// eventBuildFinished
	buildOK     bool
	buildPath   string
	buildResult build.Result

	// eventChildExited: generation identifies which child exited, so a
	// stale notification from a child this controller has already
	// deliberately replaced or stopped is recognized and ignored rather
	// than misreported as a fresh runtime failure.
	generation int
	exitErr    error

	// eventStopRequested
	force bool
	done  chan struct{}
}

// devChild is one running candidate process together with the goroutine
// that will report its exit. Exactly one goroutine ever calls cmd.Wait();
// the event loop only ever reads exited/waitErr after that channel closes.
type devChild struct {
	cmd        *exec.Cmd
	generation int
	exited     chan struct{}
	waitErr    error
}

func startDevChild(binaryPath string, generation int, output, errorOutput io.Writer, events chan<- devControllerEvent) (*devChild, error) {
	cmd := exec.Command(binaryPath)
	cmd.Stdout = output
	cmd.Stderr = errorOutput
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	child := &devChild{cmd: cmd, generation: generation, exited: make(chan struct{})}
	go func() {
		child.waitErr = cmd.Wait()
		close(child.exited)
		select {
		case events <- devControllerEvent{kind: eventChildExited, generation: generation, exitErr: child.waitErr}:
		default:
			// The event loop is gone (shutdown already completed and closed
			// the channel's reader); nothing left to report to.
		}
	}()
	return child, nil
}

// stopDevChild sends SIGTERM and waits up to shutdownGracePeriod for the
// exit this child's own wait-goroutine will observe, then escalates to
// SIGKILL. This is the dev controller's *internal* child-replacement/
// shutdown plumbing, not the user-facing `ahdcode stop` command: stop's own
// graceful-first contract (section 18) is enforced at the CLI layer in
// runstop.go, which reports failure rather than escalating. Here, escalating
// is deliberate and necessary -- an unresponsive old candidate must never be
// able to wedge every future rebuild shut.
func stopDevChild(child *devChild) {
	if child == nil {
		return
	}
	_ = terminateOwnedProcess(child.cmd.Process, false)
	select {
	case <-child.exited:
		return
	case <-time.After(shutdownGracePeriod):
	}
	_ = terminateOwnedProcess(child.cmd.Process, true)
	<-child.exited
}

// stopDevChildForce skips the grace period entirely -- used only for the
// force-stop path `ahdcode kill app.dev` reaches, where the caller has
// explicitly asked for immediate termination.
func stopDevChildForce(child *devChild) {
	if child == nil {
		return
	}
	_ = terminateOwnedProcess(child.cmd.Process, true)
	<-child.exited
}

type devController struct {
	entry    string
	binDir   string
	output   io.Writer
	errorOut io.Writer

	// watcher's tracked path set is replaced after every completed build
	// attempt (section 25), success or failure, with the entry plus the
	// compiler's resolved require(...) graph plus any require(...) target
	// that attempt named but could not find yet. Set once, right after
	// construction, in runDev.
	watcher *devWatcher

	events chan devControllerEvent

	building        bool
	pendingRebuild  bool
	shuttingDown    bool
	child           *devChild
	childGeneration int
	buildGeneration int
	currentBinary   string
	descriptorPath  string
	controlPort     int
	devToken        string

	// stopWaiters accumulates every in-flight stop/Ctrl-C request's done
	// channel. Usually at most one, but nothing prevents a second request
	// arriving while the first is still waiting on an in-flight build; all
	// of them are released together, only once finishShutdown has actually
	// run -- never earlier, so a caller that observes completion can trust
	// the session is genuinely gone.
	stopWaiters  []chan struct{}
	shutdownDone chan struct{}

	// webAnnounced records that the Web banner has already been printed for
	// this session. The environment is inspected once, after the first
	// successful build, so an ordinary rebuild stays as quiet as it is for
	// any other program.
	webAnnounced   bool
	webBannerShown bool
	webEnvironment webEnvironment

	// exitStatus is the process exit code this session ends with. It stays 0
	// for every ordinary run, including a failed build, which dev recovers
	// from; a refused configuration is not recoverable and ends non-zero so
	// a script or CI notices.
	exitStatus int
}

func newDevController(entry, binDir, descriptorPath string, output, errorOut io.Writer) *devController {
	return &devController{
		entry: entry, binDir: binDir, descriptorPath: descriptorPath,
		output: output, errorOut: errorOut,
		events:       make(chan devControllerEvent, 8),
		shutdownDone: make(chan struct{}),
	}
}

func (c *devController) printf(format string, args ...any) {
	fmt.Fprintf(c.output, format, args...)
}

// startBuild launches one compile in its own goroutine and reports the
// result back through the event channel. The event loop only ever calls
// this when c.building is false, so at most one is ever in flight -- see
// section 11 and section 36.
func (c *devController) startBuild() {
	c.building = true
	c.buildGeneration++
	c.printf("→ Building...\n")
	outputPath := filepath.Join(c.binDir, fmt.Sprintf("candidate-%d", c.buildGeneration))
	go func() {
		path, result := build.BuildProgram(c.entry, outputPath)
		c.events <- devControllerEvent{kind: eventBuildFinished, buildOK: !result.HasErrors(), buildPath: path, buildResult: result}
	}()
}

// updateWatchSet replaces the watcher's tracked files with the entry plus
// this build attempt's full dependency picture. The entry is always
// included even if, in some degenerate failure, SourcePaths came back
// empty -- dev must always be able to notice the entry itself being fixed.
// Order does not matter to the watcher, so a plain dedup is enough to keep
// the set deterministic without needing the compiler's own resolution
// order.
func (c *devController) updateWatchSet(result build.Result) {
	if c.watcher == nil {
		return
	}
	seen := map[string]bool{c.entry: true}
	paths := []string{c.entry}
	for _, path := range result.SourcePaths {
		if seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	c.watcher.setPathsAsync(paths)
}

func (c *devController) reportDiagnostics(result build.Result) {
	for _, item := range result.Diagnostics {
		fmt.Fprintln(c.errorOut, format(item, result.Files))
	}
}

// run is the single event-loop goroutine. Every state transition described
// in sections 5-7, 11, and 36-39 happens here and only here.
func (c *devController) run() {
	defer close(c.shutdownDone)
	// Published before the first build even starts, with no child yet
	// (childPid 0): otherwise a session whose very first build fails (see
	// section 4) would never have a discoverable descriptor at all, making
	// it unstoppable via `ahdcode stop`/`kill` and invisible to duplicate-
	// session detection for as long as it stayed broken.
	c.publishDescriptor()
	c.startBuild()

	for event := range c.events {
		switch event.kind {
		case eventSourceChanged:
			if c.shuttingDown {
				continue
			}
			if c.building {
				c.pendingRebuild = true
				continue
			}
			c.printf("Source changed\n")
			c.startBuild()

		case eventBuildFinished:
			c.building = false
			if c.shuttingDown {
				c.discardCandidate(event.buildPath, false)
				c.finishShutdown()
				return
			}
			c.updateWatchSet(event.buildResult)
			if event.buildOK {
				c.printf("✓ Build succeeded\n")
				if !c.webAnnounced && isWebApplication(event.buildResult) {
					environment := readWebEnvironment(c.entry)
					if err := checkWebEnvironment(environment); err != nil {
						// A production configuration reached the development
						// command. Refuse it rather than run it: this is a
						// mismatch the user has to resolve, and starting the
						// child anyway would run production configuration
						// under a development session without saying so.
						fmt.Fprintf(c.errorOut, "✗ %v\n", err)
						c.discardCandidate(event.buildPath, false)
						c.shuttingDown = true
						c.exitStatus = 1
						c.finishShutdown()
						return
					}
					c.webAnnounced = true
					c.webEnvironment = environment
				}
				if c.child != nil {
					c.printf("→ Restarting...\n")
					stopDevChild(c.child)
					c.removeCandidate(c.currentBinary)
				} else {
					c.printf("→ Starting...\n")
				}
				c.childGeneration++
				child, err := startDevChild(event.buildPath, c.childGeneration, c.output, c.errorOut, c.events)
				if err != nil {
					c.printf("✗ Could not start the application: %v\n", err)
					c.child = nil
					c.currentBinary = ""
				} else {
					c.child = child
					c.currentBinary = event.buildPath
					c.printf("✓ Running\n")
					c.publishDescriptor()
					if c.webAnnounced && !c.webBannerShown {
						c.webBannerShown = true
						announceWebApplication(c.output, c.webEnvironment)
					}
				}
			} else {
				c.printf("✗ Build failed\n\n")
				c.reportDiagnostics(event.buildResult)
				fmt.Fprintln(c.errorOut)
				c.discardCandidate(event.buildPath, false)
				if c.child != nil {
					c.printf("✓ Previous successful build is still running\n")
				}
			}
			if c.pendingRebuild {
				c.pendingRebuild = false
				c.printf("Source changed\n")
				c.startBuild()
			} else {
				c.printf("Waiting for changes...\n")
			}

		case eventChildExited:
			if event.generation != c.childGeneration || c.child == nil {
				// A deliberate stop/replace already forgot this child;
				// this is its trailing exit notification, not news.
				continue
			}
			c.child = nil
			c.currentBinary = ""
			if c.shuttingDown {
				continue
			}
			c.printf("✗ Application exited with code %s\n", exitCodeText(event.exitErr))
			c.printf("Waiting for changes...\n")

		case eventStopRequested:
			c.shuttingDown = true
			if event.done != nil {
				c.stopWaiters = append(c.stopWaiters, event.done)
			}
			if c.child != nil {
				if event.force {
					stopDevChildForce(c.child)
				} else {
					stopDevChild(c.child)
				}
				c.removeCandidate(c.currentBinary)
				c.child = nil
				c.currentBinary = ""
			}
			if !c.building {
				c.finishShutdown()
				return
			}
			// A build is still in flight: it is left to finish (compilation
			// has no clean early-cancel hook, and letting it complete is
			// simpler and safer than interrupting mid-write); its result is
			// discarded in the eventBuildFinished branch above, which is
			// where shutdown actually completes and every waiter is
			// released.
		}
	}
}

func (c *devController) removeCandidate(path string) {
	if path == "" {
		return
	}
	_ = os.Remove(path)
}

func (c *devController) discardCandidate(path string, keep bool) {
	if keep || path == "" {
		return
	}
	_ = os.Remove(path)
}

func (c *devController) publishDescriptor() {
	controllerPID := os.Getpid()
	childPID := 0
	if c.child != nil {
		childPID = c.child.cmd.Process.Pid
	}
	_ = startDevDescriptor(c.descriptorPath, c.entry, filepath.Dir(c.entry), controllerPID, childPID, c.controlPort, c.devToken)
}

// finishShutdown removes every temporary/descriptor trace of this session
// and releases every waiter (the control-channel handler and/or the Ctrl-C
// goroutine) that is blocked on the stop actually being complete.
func (c *devController) finishShutdown() {
	c.removeCandidate(c.currentBinary)
	removeOwnDevDescriptor(c.descriptorPath, c.controlPort)
	_ = os.RemoveAll(c.binDir)
	for _, waiter := range c.stopWaiters {
		close(waiter)
	}
	c.stopWaiters = nil
}

func exitCodeText(err error) string {
	if err == nil {
		return "0"
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return fmt.Sprintf("%d", exitErr.ExitCode())
	}
	return err.Error()
}

// runDev implements `ahdcode dev <entry.ahd>`.
func runDev(arguments []string, output, errorOutput io.Writer) int {
	if len(arguments) != 1 || strings.HasPrefix(arguments[0], "-") {
		fmt.Fprintln(errorOutput, "ahdcode dev: exactly one entry module is expected, as in: ahdcode dev app.ahd")
		return 2
	}
	entry := arguments[0]
	absoluteEntry, err := filepath.Abs(entry)
	if err != nil {
		absoluteEntry = entry
	}

	descriptorPath := devFileFor(entry)
	if err := claimDevFile(descriptorPath); err != nil {
		fmt.Fprintf(errorOutput, "ahdcode dev: %v\n", err)
		return 1
	}

	binDir, err := os.MkdirTemp("", "ahdcode-dev-*")
	if err != nil {
		fmt.Fprintf(errorOutput, "ahdcode dev: could not create a build workspace: %v\n", err)
		return 1
	}

	control, err := startDevControlServer()
	if err != nil {
		_ = os.RemoveAll(binDir)
		fmt.Fprintf(errorOutput, "ahdcode dev: could not start the dev control channel: %v\n", err)
		return 1
	}
	defer control.close()

	controller := newDevController(absoluteEntry, binDir, descriptorPath, output, errorOutput)
	controller.controlPort = control.port()
	controller.devToken = control.token
	control.setOnStop(func(force bool) {
		done := make(chan struct{})
		controller.events <- devControllerEvent{kind: eventStopRequested, force: force, done: done}
		<-done
	})

	stopSignals := make(chan os.Signal, 1)
	signal.Notify(stopSignals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stopSignals)
	go func() {
		if _, ok := <-stopSignals; ok {
			fmt.Fprintln(output, "^C")
			fmt.Fprintln(output, "Stopping...")
			done := make(chan struct{})
			controller.events <- devControllerEvent{kind: eventStopRequested, done: done}
			<-done
			fmt.Fprintln(output, "✓ Server stopped")
			fmt.Fprintln(output, "✓ Dev watcher stopped")
			os.Exit(0)
		}
	}()

	watcher := newDevWatcher([]string{absoluteEntry}, controller.events)
	controller.watcher = watcher
	watcher.start()

	fmt.Fprintln(output, "AhdCode Dev")
	fmt.Fprintln(output)
	fmt.Fprintf(output, "Watching: %s\n", absoluteEntry)
	fmt.Fprintln(output)

	controller.run()
	watcher.close()
	return controller.exitStatus
}
