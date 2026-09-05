package localdev

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// AhdCode's system-hosts integration is deliberately the smallest thing that
// can work: one clearly delimited block, appended at the end, containing only
// loopback mappings for names AhdCode itself owns.
//
// Everything outside the markers is copied through byte for byte. The file is
// never regenerated, never reordered, and never "cleaned up" -- an unrelated
// entry a user or another tool put there survives every operation here,
// including removal of the AhdCode block itself. That is the whole reason
// this is a block rather than a rewrite.
const (
	hostsBeginMarker = "# BEGIN AHDCODE LOCAL"
	hostsEndMarker   = "# END AHDCODE LOCAL"
)

// SystemHostsPath is the platform's hosts file.
func SystemHostsPath() string {
	if runtime.GOOS == "windows" {
		root := os.Getenv("SystemRoot")
		if root == "" {
			root = `C:\Windows`
		}
		return filepath.Join(root, "System32", "drivers", "etc", "hosts")
	}
	return "/etc/hosts"
}

// RenderHostsBlock builds the managed block for a set of hostnames. Only
// loopback is ever mapped, and the names are sorted so the same set always
// produces the same text -- a rewrite that changes nothing must produce a
// file that differs in nothing.
func RenderHostsBlock(hostnames []string) string {
	unique := make([]string, 0, len(hostnames))
	seen := map[string]bool{}
	for _, hostname := range hostnames {
		name := strings.ToLower(strings.TrimSpace(hostname))
		if !ValidHostname(name) || seen[name] {
			continue
		}
		seen[name] = true
		unique = append(unique, name)
	}
	if len(unique) == 0 {
		return ""
	}
	sort.Strings(unique)

	var builder strings.Builder
	builder.WriteString(hostsBeginMarker)
	builder.WriteString("\n")
	for _, name := range unique {
		fmt.Fprintf(&builder, "127.0.0.1 %s\n", name)
	}
	builder.WriteString(hostsEndMarker)
	builder.WriteString("\n")
	return builder.String()
}

// ApplyHostsBlock returns the new content for a hosts file with AhdCode's
// managed block set to exactly block.
//
// An existing managed block is replaced in place, keeping its position, so
// nothing above or below it moves. When there is none, the block is appended.
// An empty block removes AhdCode's section entirely and leaves the rest of
// the file as it was.
func ApplyHostsBlock(existing, block string) string {
	before, after, found := splitManagedBlock(existing)
	if !found {
		if block == "" {
			return existing
		}
		prefix := existing
		if prefix != "" && !strings.HasSuffix(prefix, "\n") {
			prefix += "\n"
		}
		return prefix + block
	}
	return before + block + after
}

// splitManagedBlock finds the managed section and returns everything before
// and after it. An unterminated begin marker is treated as running to the end
// of the file: a half-written block from an interrupted edit must still be
// replaceable rather than duplicated.
func splitManagedBlock(content string) (before, after string, found bool) {
	lines := strings.Split(content, "\n")
	start, end := -1, -1
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if start < 0 && trimmed == hostsBeginMarker {
			start = index
			continue
		}
		if start >= 0 && trimmed == hostsEndMarker {
			end = index
			break
		}
	}
	if start < 0 {
		return content, "", false
	}
	if end < 0 {
		end = len(lines) - 1
	}
	before = strings.Join(lines[:start], "\n")
	if before != "" && !strings.HasSuffix(before, "\n") {
		before += "\n"
	}
	after = strings.Join(lines[end+1:], "\n")
	return before, after, true
}

// HostsBlockNames lists the hostnames AhdCode's managed block currently maps.
func HostsBlockNames(content string) []string {
	_, _, found := splitManagedBlock(content)
	if !found {
		return nil
	}
	inside := false
	var names []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		trimmed := strings.TrimSpace(scanner.Text())
		if trimmed == hostsBeginMarker {
			inside = true
			continue
		}
		if trimmed == hostsEndMarker {
			break
		}
		if !inside {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 || fields[0] != "127.0.0.1" {
			continue
		}
		names = append(names, fields[1:]...)
	}
	return names
}

// HostsMapsLoopback reports whether the file as a whole -- managed block or
// not -- already points hostname at 127.0.0.1.
//
// A mapping to anything else answers false rather than true: AhdCode's local
// services bind IPv4 loopback only, so a name a user has pointed somewhere
// else is a name AhdCode must not claim works. The whole file is consulted,
// not just the managed block, because an entry a person added by hand is
// just as good as one AhdCode wrote.
func HostsMapsLoopback(content, hostname string) bool {
	want := strings.ToLower(strings.TrimSpace(hostname))
	if want == "" {
		return false
	}
	found := false
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line, _, _ := strings.Cut(scanner.Text(), "#")
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		for _, name := range fields[1:] {
			if !strings.EqualFold(strings.TrimSuffix(name, "."), want) {
				continue
			}
			if fields[0] != "127.0.0.1" {
				return false
			}
			found = true
		}
	}
	return found
}

// ReadSystemHosts loads the hosts file. A file that cannot be read is
// reported as empty rather than as an error: not being able to read
// /etc/hosts means AhdCode cannot confirm a mapping, which is exactly the
// same outcome for every caller as the mapping being absent.
func ReadSystemHosts() string {
	content, err := os.ReadFile(SystemHostsPath())
	if err != nil {
		return ""
	}
	return string(content)
}
