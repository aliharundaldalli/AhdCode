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

// An AhdCode dev descriptor records the one development session `ahdcode
// dev` is currently running, so `ahdcode stop app.dev` (or `kill app.dev`)
// can find it. It follows runDescriptor's conventions exactly -- same
// atomic-private-write discipline, same "the pid is diagnostic metadata
// only, never signalled" contract -- extended with a second pid because a
// dev session owns two things that outlive a plain run: the controller
// itself, and whichever child it currently has running (which changes
// across rebuilds; 0 when no child is running, such as right after a
// compile failure with no prior successful build).
const (
	devDescriptorSchema  = "ahdcode.dev"
	devDescriptorVersion = 1
	devDescriptorMaxPID  = 1 << 22
	devDescriptorSuffix  = ".dev"
)

type devDescriptor struct {
	Schema  string `json:"schema"`
	Version int    `json:"version"`
	// ControllerPID and ChildPID are informational: shown in messages and
	// never signalled directly. See devcontrol.go for why -- the same
	// loopback control-channel authentication runDescriptor uses.
	ControllerPID    int    `json:"controllerPid"`
	ChildPID         int    `json:"childPid"`
	Source           string `json:"source"`
	WorkingDirectory string `json:"workingDirectory"`
	StartedAt        string `json:"startedAt"`
	ControlPort      int    `json:"controlPort"`
	ControlToken     string `json:"controlToken"`
}

// devFileFor derives app.dev from app.ahd, so `ahdcode dev app.ahd` and
// `ahdcode stop app.dev` name the same session in the same directory.
func devFileFor(entry string) string {
	absolute, err := filepath.Abs(entry)
	if err != nil {
		absolute = entry
	}
	extension := filepath.Ext(absolute)
	if extension == "" {
		return absolute + devDescriptorSuffix
	}
	return strings.TrimSuffix(absolute, extension) + devDescriptorSuffix
}

// writeDevDescriptor publishes a descriptor with the same atomic,
// private-before-visible discipline writeRunDescriptor uses: written to a
// fresh 0600 temporary file in the destination's own directory, then renamed
// into place, so the control token is never readable at any wider
// permission and never through a partially written file.
func writeDevDescriptor(path string, descriptor devDescriptor) error {
	encoded, err := json.Marshal(descriptor)
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')

	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".ahdcode-dev-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	cleanup := func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}
	if err := temporary.Chmod(0o600); err != nil {
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

var errDevDescriptorShape = errors.New("not an AhdCode dev file")

// readDevDescriptor parses and validates one descriptor, the same strict way
// readRunDescriptor does.
func readDevDescriptor(path string) (devDescriptor, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return devDescriptor{}, err
	}
	if len(content) == 0 {
		return devDescriptor{}, errDevDescriptorShape
	}
	var descriptor devDescriptor
	if err := json.Unmarshal(content, &descriptor); err != nil {
		return devDescriptor{}, errDevDescriptorShape
	}
	if descriptor.Schema != devDescriptorSchema {
		return devDescriptor{}, errDevDescriptorShape
	}
	if descriptor.Version != devDescriptorVersion {
		return devDescriptor{}, fmt.Errorf("unsupported AhdCode dev file version %d", descriptor.Version)
	}
	if descriptor.ControllerPID <= 0 || descriptor.ControllerPID > devDescriptorMaxPID {
		return devDescriptor{}, errDevDescriptorShape
	}
	if descriptor.ChildPID < 0 || descriptor.ChildPID > devDescriptorMaxPID {
		return devDescriptor{}, errDevDescriptorShape
	}
	if strings.TrimSpace(descriptor.Source) == "" {
		return devDescriptor{}, errDevDescriptorShape
	}
	if descriptor.ControlPort <= 0 || descriptor.ControlPort > 65535 {
		return devDescriptor{}, errDevDescriptorShape
	}
	if !validRunControlToken(descriptor.ControlToken) {
		return devDescriptor{}, errDevDescriptorShape
	}
	return descriptor, nil
}

// devDescriptorIsLive asks the recorded control endpoint to identify itself,
// the same liveness test runDescriptorIsLive uses. Only a controller that
// accepts this descriptor's own token counts as live.
func devDescriptorIsLive(descriptor devDescriptor) bool {
	return requestDevControl(descriptor.ControlPort, descriptor.ControlToken, devControlPing) == nil
}

// claimDevFile prepares path for a new dev session. A live controller blocks
// a second one from starting -- section 21's "duplicate dev session"
// rejection -- while a stale leftover is silently reclaimed.
func claimDevFile(path string) error {
	descriptor, err := readDevDescriptor(path)
	if err != nil {
		return nil
	}
	if devDescriptorIsLive(descriptor) {
		return fmt.Errorf("a development session for this source is already active\nrun file: %s\nstop it with: ahdcode stop %s",
			path, filepath.Base(path))
	}
	return nil
}

func startDevDescriptor(path, source, workingDirectory string, controllerPID, childPID, controlPort int, controlToken string) error {
	absoluteSource, err := filepath.Abs(source)
	if err != nil {
		absoluteSource = source
	}
	absoluteDirectory, err := filepath.Abs(workingDirectory)
	if err != nil {
		absoluteDirectory = workingDirectory
	}
	return writeDevDescriptor(path, devDescriptor{
		Schema: devDescriptorSchema, Version: devDescriptorVersion,
		ControllerPID: controllerPID, ChildPID: childPID,
		Source: absoluteSource, WorkingDirectory: absoluteDirectory,
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		ControlPort: controlPort, ControlToken: controlToken,
	})
}

// removeOwnDevDescriptor deletes the descriptor only when it still names
// this session's own control channel, the same guard
// removeOwnRunDescriptor uses against a concurrent replacement.
func removeOwnDevDescriptor(path string, controlPort int) {
	descriptor, err := readDevDescriptor(path)
	if err != nil || descriptor.ControlPort != controlPort {
		return
	}
	_ = os.Remove(path)
}
