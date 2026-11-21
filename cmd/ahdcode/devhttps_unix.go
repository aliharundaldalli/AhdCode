//go:build !windows

package main

import (
	"fmt"
	"os"
)

func mkdirPrivateDevHTTPS(path string) error {
	return os.Mkdir(path, 0o700)
}

func checkDevHTTPSPrivacy(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot inspect local HTTPS key storage: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("local HTTPS key storage must be accessible only to its owner: %s", path)
	}
	return nil
}
