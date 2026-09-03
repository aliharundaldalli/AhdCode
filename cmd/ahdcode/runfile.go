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
// This is deliberately not a process manager and not a daemon registry: one
// descriptor sits next to one source file, it exists only while that run is
// alive, and it is internal CLI metadata rather than a language-level
// serialization contract. Nothing in the standard library can read or write
// it.
const (
	runDescriptorSchema  = "ahdcode.run"
	runDescriptorVersion = 1
	runDescriptorMaxPID  = 1 << 22
	runDescriptorSuffix  = ".run"
)

type runDescriptor struct {
	Schema    string `json:"schema"`
	Version   int    `json:"version"`
	PID       int    `json:"pid"`
	Source    string `json:"source"`
	StartedAt string `json:"startedAt"`
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

func writeRunDescriptor(path string, descriptor runDescriptor) error {
	encoded, err := json.Marshal(descriptor)
	if err != nil {
		return err
	}
	// 0600: the descriptor names a process this user may signal, so it is not
	// world-readable where the platform enforces permissions.
	return os.WriteFile(path, append(encoded, '\n'), 0o600)
}

var errRunDescriptorShape = errors.New("not an AhdCode run file")

// readRunDescriptor parses and validates one descriptor. Anything that is not
// a well-formed AhdCode run file of a known version, naming a plausible
// process id, is rejected outright: a malformed file must never be followed
// into signalling an arbitrary process.
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
	return descriptor, nil
}

// claimRunFile prepares path for a new run. An existing descriptor whose
// process is still alive blocks the run instead of silently overwriting a
// live application's identity; a stale or malformed one is cleared.
func claimRunFile(path string) error {
	descriptor, err := readRunDescriptor(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		// A malformed leftover is not a live application; replace it.
		return nil
	}
	if processAlive(descriptor.PID) {
		return fmt.Errorf("an AhdCode application is already running (pid %d)\nrun file: %s\nstop it with: ahdcode kill %s",
			descriptor.PID, path, filepath.Base(path))
	}
	return nil
}

func startRunDescriptor(path, source string, pid int) error {
	absolute, err := filepath.Abs(source)
	if err != nil {
		absolute = source
	}
	return writeRunDescriptor(path, runDescriptor{
		Schema: runDescriptorSchema, Version: runDescriptorVersion,
		PID: pid, Source: absolute, StartedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

// removeOwnRunDescriptor deletes the descriptor only when it still names this
// run, so a concurrent replacement is never removed by an exiting process.
func removeOwnRunDescriptor(path string, pid int) {
	descriptor, err := readRunDescriptor(path)
	if err != nil || descriptor.PID != pid {
		return
	}
	_ = os.Remove(path)
}
