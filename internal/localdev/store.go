package localdev

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Registry files follow the same publish discipline the run/dev descriptors
// use: content is written to a fresh 0600 temporary file in the destination's
// own directory and renamed into place, so a reader never sees a partially
// written registry and the file is never briefly readable at a wider
// permission.
func writeAtomic(path string, payload any) error {
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')

	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".ahdcode-registry-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	cleanup := func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}
	if err := temporary.Chmod(0o600); err != nil && !errors.Is(err, os.ErrInvalid) {
		cleanup()
		return err
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

// readJSON loads a registry file. A missing file is not an error -- it is the
// ordinary state before anything has ever been registered -- and neither is
// an empty one. A file that exists but cannot be parsed *is* an error: it is
// user state, so it is reported rather than silently overwritten.
func readJSON(path string, into any) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if len(content) == 0 {
		return false, nil
	}
	if err := json.Unmarshal(content, into); err != nil {
		return false, fmt.Errorf("%s is not a readable AhdCode registry (%v)", path, err)
	}
	return true, nil
}

const (
	lockPollInterval = 15 * time.Millisecond
	lockTimeout      = 5 * time.Second
	// A lock older than this belonged to a process that died holding it.
	// The window is generous compared with the microseconds a
	// read-modify-write of a small JSON file actually takes.
	lockStaleAfter = 30 * time.Second
)

// withLock serializes read-modify-write cycles on one registry file so two
// concurrent `ahdcode dev` starts cannot both decide the same hostname is
// free. O_CREATE|O_EXCL on a sibling lock file is used rather than flock
// because it behaves identically on every platform the toolchain targets.
//
// A lock left behind by a process that died mid-write would otherwise wedge
// the registry permanently, so a lock file older than lockStaleAfter is
// broken. The window is far longer than any legitimate hold, and the
// operation it guards is idempotent enough that a mistaken break costs at
// most one lost concurrent update rather than a corrupted file -- the write
// itself is still atomic.
func withLock(path string, action func() error) error {
	lockPath := path + ".lock"
	deadline := time.Now().Add(lockTimeout)
	for {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = file.Close()
			defer func() { _ = os.Remove(lockPath) }()
			return action()
		}
		if !os.IsExist(err) {
			return err
		}
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > lockStaleAfter {
			_ = os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("another AhdCode process is still updating %s", filepath.Base(path))
		}
		time.Sleep(lockPollInterval)
	}
}

func nowStamp() string { return time.Now().UTC().Format(time.RFC3339) }
