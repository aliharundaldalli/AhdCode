package localdev

import (
	"os"
	"strings"
)

const hostsAuthorizedFileName = "hosts-authorized"

// HostsAuthorized reports whether this user has already allowed AhdCode to
// maintain its managed loopback block. The flag is local metadata, not a
// secret, and never implies that /etc/hosts is currently writable.
func HostsAuthorized() bool {
	path, err := childOfHome(hostsAuthorizedFileName)
	if err != nil {
		return false
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	value := strings.ToLower(strings.TrimSpace(string(content)))
	return value == "1" || value == "true" || value == "yes"
}

// MarkHostsAuthorized records that the user consented to AhdCode-managed
// loopback hostnames. Subsequent sessions reuse this instead of asking again.
func MarkHostsAuthorized() error {
	path, err := childOfHome(hostsAuthorizedFileName)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte("1\n"), 0o600)
}

// HostsAuthorizedPath is the absolute path of the authorization marker.
func HostsAuthorizedPath() (string, error) { return childOfHome(hostsAuthorizedFileName) }
