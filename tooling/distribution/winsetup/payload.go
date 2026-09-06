package winsetup

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// PayloadVersion reads the release identity the payload was built for.
func PayloadVersion(archive *zip.Reader) (string, error) {
	content, err := readEntry(archive, "VERSION")
	if err != nil {
		return "", errors.New("this setup file is damaged: it carries no release identity")
	}
	return ReadPointer(content)
}

// ReadInventory returns the builder's record of every payload file and its
// SHA-256 digest.
func ReadInventory(archive *zip.Reader) (map[string]string, error) {
	content, err := readEntry(archive, "FILES.json")
	if err != nil {
		return nil, errors.New("this setup file is damaged: it carries no file inventory")
	}
	inventory := map[string]string{}
	if err := json.Unmarshal(content, &inventory); err != nil {
		return nil, errors.New("this setup file is damaged: its file inventory could not be read")
	}
	return inventory, nil
}

func readEntry(archive *zip.Reader, name string) ([]byte, error) {
	for _, file := range archive.File {
		if file.Name != name {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	}
	return nil, os.ErrNotExist
}

// Extract unpacks the payload into staging and checks every inventoried file
// against its recorded digest while it is written, so the payload is read once
// rather than twice. Any unsafe name, any digest mismatch, and any inventoried
// file that never appeared is an error, and the caller is expected to discard
// the staging directory rather than activate it.
//
// progress may be nil; it is called with the number of files written so far and
// the total the payload contains.
func Extract(archive *zip.Reader, staging string, inventory map[string]string, progress func(written, total int)) error {
	total := 0
	for _, file := range archive.File {
		if !file.FileInfo().IsDir() {
			total++
		}
	}
	written := 0
	verified := 0
	for _, file := range archive.File {
		if err := CheckEntryName(file.Name); err != nil {
			return err
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return ErrUnsafeEntry
		}
		destination := filepath.Join(staging, filepath.FromSlash(file.Name))
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(destination, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		digest, err := writeEntry(file, destination)
		if err != nil {
			return err
		}
		if want, ok := inventory[file.Name]; ok {
			if digest != want {
				return fmt.Errorf("this setup file is damaged: %s did not match its checksum", file.Name)
			}
			verified++
		}
		written++
		if progress != nil && written%64 == 0 {
			progress(written, total)
		}
	}
	if missing := len(inventory) - verified; missing > 0 {
		return fmt.Errorf("this setup file is incomplete: %d expected files are missing", missing)
	}
	if progress != nil {
		progress(written, total)
	}
	return nil
}

func writeEntry(file *zip.File, destination string) (string, error) {
	source, err := file.Open()
	if err != nil {
		return "", err
	}
	defer source.Close()
	// O_EXCL: a payload that names the same file twice must fail loudly rather
	// than let a later entry quietly replace a verified one.
	sink, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	digest := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(sink, digest), source)
	closeErr := sink.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
