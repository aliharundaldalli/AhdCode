package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ahdcode stop is the graceful counterpart to ahdcode kill: it asks the
// owning supervisor (a plain `run` or a `dev` controller) to shut down
// cleanly and, unlike kill, waits to confirm the process actually exited
// before reporting success. It never signals a pid read out of a file --
// every action goes through the same authenticated loopback control
// channel runcontrol.go/devcontrol.go already use for kill.
//
// Grace-period semantics are mandatory and deliberately not silently
// upgraded to a force-kill on timeout (section 18): if the process has not
// exited by the deadline, stop reports that plainly and points at
// `ahdcode kill` for the forced path.
func runStop(arguments []string, output, errorOutput io.Writer) int {
	target := ""
	for _, argument := range arguments {
		switch {
		case strings.HasPrefix(argument, "-"):
			fmt.Fprintf(errorOutput, "ahdcode stop: unknown flag %q\n", argument)
			return 2
		case target == "":
			target = argument
		default:
			fmt.Fprintln(errorOutput, "ahdcode stop: exactly one session is expected, as in: ahdcode stop app.dev")
			return 2
		}
	}
	if target == "" {
		fmt.Fprintln(errorOutput, "ahdcode stop: a session is required, as in: ahdcode stop app.dev (or app.run)")
		return 2
	}

	switch filepath.Ext(target) {
	case devDescriptorSuffix:
		return stopDevSession(target, output, errorOutput)
	case runDescriptorSuffix:
		return stopRunSession(target, output, errorOutput)
	}

	// Section 20's convenience: a bare source name (or anything else without
	// a recognized suffix) resolves against both possible descriptors next
	// to it. Ambiguity is never guessed through.
	devPath, runPath := devFileFor(target), runFileFor(target)
	devLive := isLiveDevDescriptor(devPath)
	runLive := isLiveRunDescriptor(runPath)
	switch {
	case devLive && runLive:
		fmt.Fprintf(errorOutput, "Multiple active sessions found for %s:\n  %s\n  %s\n\nSpecify a session explicitly.\n",
			target, filepath.Base(runPath), filepath.Base(devPath))
		return 1
	case devLive:
		return stopDevSession(devPath, output, errorOutput)
	case runLive:
		return stopRunSession(runPath, output, errorOutput)
	default:
		fmt.Fprintf(errorOutput, "ahdcode stop: no active session found for %s\n", target)
		return 1
	}
}

func isLiveRunDescriptor(path string) bool {
	descriptor, err := readRunDescriptor(path)
	return err == nil && runDescriptorIsLive(descriptor)
}

func isLiveDevDescriptor(path string) bool {
	descriptor, err := readDevDescriptor(path)
	return err == nil && devDescriptorIsLive(descriptor)
}

func stopRunSession(path string, output, errorOutput io.Writer) int {
	descriptor, err := readRunDescriptor(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(errorOutput, "ahdcode stop: no run file at %s\n", path)
			return 1
		}
		fmt.Fprintf(errorOutput, "ahdcode stop: %s is not a usable AhdCode run file (%v); no process was stopped\n", path, err)
		return 1
	}

	fmt.Fprintf(output, "Stopping %s...\n", filepath.Base(path))
	if err := requestRunControl(descriptor.ControlPort, descriptor.ControlToken, runControlStop); err != nil {
		if errors.Is(err, errRunControlUnreachable) {
			if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
				fmt.Fprintf(errorOutput, "ahdcode stop: the application is not running, but the stale run file could not be removed: %v\n", removeErr)
				return 1
			}
			fmt.Fprintf(output, "AhdCode application is not running; removed the stale run file %s (no process was signalled)\n", path)
			return 0
		}
		fmt.Fprintf(errorOutput, "ahdcode stop: %v\n", err)
		return 1
	}

	// The supervisor has already sent SIGTERM and retired its descriptor
	// synchronously inside that request; now confirm actual exit within the
	// grace period the same authenticated way -- once the child (and so the
	// whole `ahdcode run` process) has genuinely exited, its control
	// listener is gone and ping stops being answered at all.
	deadline := time.Now().Add(shutdownGracePeriod)
	for time.Now().Before(deadline) {
		if requestRunControl(descriptor.ControlPort, descriptor.ControlToken, runControlPing) != nil {
			fmt.Fprintln(output, "✓ Server stopped")
			return 0
		}
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Fprintf(errorOutput, "ahdcode stop: graceful shutdown did not complete within %s; the process may still be exiting\nUse: ahdcode kill %s\n",
		shutdownGracePeriod, filepath.Base(path))
	return 1
}

func stopDevSession(path string, output, errorOutput io.Writer) int {
	descriptor, err := readDevDescriptor(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(errorOutput, "ahdcode stop: no dev file at %s\n", path)
			return 1
		}
		fmt.Fprintf(errorOutput, "ahdcode stop: %s is not a usable AhdCode dev file (%v); no session was stopped\n", path, err)
		return 1
	}

	fmt.Fprintf(output, "Stopping %s...\n", filepath.Base(path))
	if err := requestDevControl(descriptor.ControlPort, descriptor.ControlToken, devControlStop); err != nil {
		if errors.Is(err, errDevControlUnreachable) {
			if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
				fmt.Fprintf(errorOutput, "ahdcode stop: the dev session is not running, but the stale dev file could not be removed: %v\n", removeErr)
				return 1
			}
			fmt.Fprintf(output, "AhdCode dev session is not running; removed the stale dev file %s (no process was signalled)\n", path)
			return 0
		}
		fmt.Fprintf(errorOutput, "ahdcode stop: %v\n", err)
		return 1
	}
	// requestDevControl's "ok" is sent only after the controller's onStop
	// callback -- the full watcher-stop, child-stop, descriptor-removal
	// sequence -- has actually finished, so both lines below are already
	// true by the time they print.
	fmt.Fprintln(output, "✓ Server stopped")
	fmt.Fprintln(output, "✓ Dev watcher stopped")
	return 0
}

// killDevSession is ahdcode kill's .dev counterpart. Unlike stop, it always
// requests immediate, forced termination of both the controller's current
// child and the controller itself -- "kill" is the forced command (section
// 14); there is no graceful variant to select with a flag here the way
// plain .run kill has, since that distinction is exactly what stop is for.
func killDevSession(target string, output, errorOutput io.Writer) int {
	descriptor, err := readDevDescriptor(target)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(errorOutput, "ahdcode kill: no dev file at %s\n", target)
			return 1
		}
		fmt.Fprintf(errorOutput, "ahdcode kill: %s is not a usable AhdCode dev file (%v); no session was stopped\n", target, err)
		return 1
	}
	if err := requestDevControl(descriptor.ControlPort, descriptor.ControlToken, devControlForceStop); err != nil {
		if errors.Is(err, errDevControlUnreachable) {
			if removeErr := os.Remove(target); removeErr != nil {
				fmt.Fprintf(errorOutput, "ahdcode kill: the dev session is not running, but the stale dev file could not be removed: %v\n", removeErr)
				return 1
			}
			fmt.Fprintf(output, "AhdCode dev session is not running; removed the stale dev file %s (no process was signalled)\n", target)
			return 0
		}
		fmt.Fprintf(errorOutput, "ahdcode kill: %v; no session was stopped\n", err)
		return 1
	}
	fmt.Fprintf(output, "Stopped the AhdCode dev session (controller pid %d) and removed %s\n", descriptor.ControllerPID, target)
	return 0
}
