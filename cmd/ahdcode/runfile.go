package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// An AhdCode run descriptor records the one application `ahdcode run` is
// currently running, so `ahdcode kill app.run` can stop it without the user
// hunting for ports and process ids with lsof.
//
// The descriptor is a way to *find* the live supervisor, not authority over a
// process. It carries the loopback control port and the supervisor's random
// token; the recorded pid is diagnostic metadata only and is never signalled.
// See runcontrol.go for why: a pid in a file can be forged or reused, so
// trusting it would let a descriptor terminate an unrelated process.
//
// This is deliberately not a process manager and not a daemon registry: one
// descriptor sits next to one source file, it exists only while that run is
// alive, and it is internal CLI metadata rather than a language-level
// serialization contract. Nothing in the standard library can read or write
// it.
const (
	runDescriptorSchema  = "ahdcode.run"
	runDescriptorVersion = 2
	runDescriptorMaxPID  = 1 << 22
	runDescriptorSuffix  = ".run"
)

type runDescriptor struct {
	Schema  string `json:"schema"`
	Version int    `json:"version"`
	// PID is informational: it is shown in messages and never signalled.
	PID          int    `json:"pid"`
	Source       string `json:"source"`
	StartedAt    string `json:"startedAt"`
	ControlPort  int    `json:"controlPort"`
	ControlToken string `json:"controlToken"`
}

// runFileFor derives app.run from app.ahd, so `ahdcode run app.ahd` and
// `ahdcode kill app.run` name the same application in the same directory.
func runFileFor(entry string) string {
	absolute, err := filepath.Abs(entry)
	if err != nil {
		absolute = entry
	}
	extension := filepath.Ext(absolute)
	if extension == "" {
		return absolute + runDescriptorSuffix
	}
	return strings.TrimSuffix(absolute, extension) + runDescriptorSuffix
}

// writeRunDescriptor publishes a descriptor so that its bytes are never
// readable by anyone else, even for an instant.
//
// os.WriteFile's permission argument applies only when it creates the file:
// writing over an existing 0644 leftover keeps that 0644 inode, which would
// publish the control token world-readably, and chmod-after-write would still
// leave a window where the secret is exposed. Renaming an existing 0644 file
// into place would carry the wrong mode with it.
//
// So the descriptor is written to a fresh private file in the destination's
// own directory, forced to 0600 before anything secret reaches it, and then
// renamed over the destination. The rename is atomic on Unix and replaces on
// Windows, and it replaces a symlink at the destination rather than following
// it, so descriptor contents are never written through a link to somewhere
// else.
func writeRunDescriptor(path string, descriptor runDescriptor) error {
	encoded, err := json.Marshal(descriptor)
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')

	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	// CreateTemp already creates with 0600; the explicit Chmod states the
	// requirement rather than relying on that default, and runs before any
	// secret byte is written.
	temporary, err := os.CreateTemp(directory, ".ahdcode-run-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	cleanup := func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}
	if err := temporary.Chmod(0o600); err != nil {
		// Where chmod is unsupported the file is still created privately by
		// CreateTemp, so only a real failure aborts publication.
		if !errors.Is(err, os.ErrInvalid) && !errors.Is(err, errUnsupportedChmod) {
			cleanup()
			return err
		}
	}
	if _, err := temporary.Write(encoded); err != nil {
		cleanup()
		return err
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	return nil
}

// errUnsupportedChmod names the platforms where changing a mode is a no-op
// rather than a failure worth aborting publication for.
var errUnsupportedChmod = errors.ErrUnsupported

var errRunDescriptorShape = errors.New("not an AhdCode run file")

// readRunDescriptor parses and validates one descriptor. Anything that is not
// a well-formed AhdCode run file of a known version, carrying usable control
// metadata, is rejected outright.
func readRunDescriptor(path string) (runDescriptor, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return runDescriptor{}, err
	}
	if len(content) == 0 {
		return runDescriptor{}, errRunDescriptorShape
	}
	var descriptor runDescriptor
	if err := json.Unmarshal(content, &descriptor); err != nil {
		return runDescriptor{}, errRunDescriptorShape
	}
	if descriptor.Schema != runDescriptorSchema {
		return runDescriptor{}, errRunDescriptorShape
	}
	if descriptor.Version != runDescriptorVersion {
		return runDescriptor{}, fmt.Errorf("unsupported AhdCode run file version %d", descriptor.Version)
	}
	if descriptor.PID <= 0 || descriptor.PID > runDescriptorMaxPID {
		return runDescriptor{}, errRunDescriptorShape
	}
	if strings.TrimSpace(descriptor.Source) == "" {
		return runDescriptor{}, errRunDescriptorShape
	}
	if descriptor.ControlPort <= 0 || descriptor.ControlPort > 65535 {
		return runDescriptor{}, errRunDescriptorShape
	}
	if !validRunControlToken(descriptor.ControlToken) {
		return runDescriptor{}, errRunDescriptorShape
	}
	return descriptor, nil
}

// runDescriptorIsLive asks the recorded control endpoint to identify itself.
// Only a supervisor that accepts this descriptor's own token counts as live;
// a forged descriptor, a reused port, or an unrelated local service does not.
func runDescriptorIsLive(descriptor runDescriptor) bool {
	return requestRunControl(descriptor.ControlPort, descriptor.ControlToken, runControlPing) == nil
}

// claimRunFile prepares path for a new run. A descriptor whose supervisor
// still answers blocks the run instead of silently replacing a live
// application's identity; anything else is a leftover and is cleared. Process
// existence is never consulted, so a reused pid cannot masquerade as a live
// application.
func claimRunFile(path string) error {
	descriptor, err := readRunDescriptor(path)
	if err != nil {
		// Missing or malformed leftovers are not live applications.
		return nil
	}
	if runDescriptorIsLive(descriptor) {
		return fmt.Errorf("an AhdCode application is already running (pid %d)\nrun file: %s\nstop it with: ahdcode kill %s",
			descriptor.PID, path, filepath.Base(path))
	}
	return nil
}

func startRunDescriptor(path, source string, pid, controlPort int, controlToken string) error {
	absolute, err := filepath.Abs(source)
	if err != nil {
		absolute = source
	}
	return writeRunDescriptor(path, runDescriptor{
		Schema: runDescriptorSchema, Version: runDescriptorVersion,
		PID: pid, Source: absolute, StartedAt: time.Now().UTC().Format(time.RFC3339),
		ControlPort: controlPort, ControlToken: controlToken,
	})
}

// removeOwnRunDescriptor deletes the descriptor only when it still names this
// run's own control channel, so a concurrent replacement is never removed by
// an exiting process.
func removeOwnRunDescriptor(path string, controlPort int) {
	descriptor, err := readRunDescriptor(path)
	if err != nil || descriptor.ControlPort != controlPort {
		return
	}
	_ = os.Remove(path)
}
