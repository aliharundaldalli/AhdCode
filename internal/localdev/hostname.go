package localdev

import (
	"net"
	"strconv"
	"strings"
)

// LocalTLD is the canonical local development suffix.
//
// `.test` is reserved by RFC 6761 for exactly this purpose and is guaranteed
// never to be delegated, so a name under it can never collide with a real
// domain the user might one day need. `.local` is deliberately *not* used:
// it is claimed by mDNS/Bonjour on macOS and on most Linux desktops, and
// mapping names under it in /etc/hosts fights the resolver rather than
// cooperating with it.
const LocalTLD = "test"

// StudioHost is AhdDataStudio's fixed local identity. Unlike an application
// hostname it is never suffixed on collision: there is one Studio per user,
// bound to one loopback port, so a second claimant is a conflict to report
// rather than a name to work around.
const StudioHost = "ahddatabasestudio.test"

// DeriveHostname turns an application's configured APP_HOST into the local
// development name AhdCode will route.
//
// The registrable suffix is dropped rather than kept, so a project
// configured for ahdakademi.com develops at ahdakademi.test and not at
// ahdakademi.com.test: the local name should read as the same project, not
// as the production name with something bolted on. A single-label host has
// no suffix to drop and is used whole.
//
// The result is always a syntactically valid host: labels are lowercased and
// reduced to letters, digits, and interior hyphens. An APP_HOST that reduces
// to nothing yields "" and the caller falls back rather than inventing a name.
func DeriveHostname(appHost string) string {
	host := strings.TrimSpace(appHost)
	if host == "" {
		return ""
	}
	// A configured host may carry a port; the name never does.
	if bare, _, err := net.SplitHostPort(host); err == nil {
		host = bare
	}
	host = strings.ToLower(strings.TrimSuffix(strings.Trim(host, "[]"), "."))
	if host == "" {
		return ""
	}
	// An IP literal has no name to derive from.
	if net.ParseIP(host) != nil {
		return ""
	}
	// Drop the registrable suffix. A host already written as a local name
	// loses its own "test" here and has it put straight back below, so
	// APP_HOST=ahdakademi.test derives ahdakademi.test unchanged.
	labels := strings.Split(host, ".")
	if len(labels) > 1 {
		labels = labels[:len(labels)-1]
	}
	var kept []string
	for _, label := range labels {
		if cleaned := sanitizeLabel(label); cleaned != "" {
			kept = append(kept, cleaned)
		}
	}
	if len(kept) == 0 {
		return ""
	}
	return strings.Join(kept, ".") + "." + LocalTLD
}

func sanitizeLabel(label string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(label) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' && builder.Len() > 0:
			builder.WriteRune(r)
		}
	}
	return strings.Trim(builder.String(), "-")
}

// SuffixHostname produces the nth alternative for a base local name:
// ahdakademi.test, then ahdakademi1.test, ahdakademi2.test, and so on. The
// digits attach to the name itself rather than becoming a new label, so
// every alternative stays visibly the same project.
func SuffixHostname(hostname string, index int) string {
	if index <= 0 {
		return hostname
	}
	stem := strings.TrimSuffix(hostname, "."+LocalTLD)
	return stem + strconv.Itoa(index) + "." + LocalTLD
}

// ValidHostname accepts exactly the shape this package is willing to route:
// a non-empty, lowercase, dot-separated name under .test. Anything else --
// an absolute name from somewhere else, an IP literal, a name with a port --
// is refused before it can reach the router's allowlist.
func ValidHostname(hostname string) bool {
	if hostname == "" || len(hostname) > 253 {
		return false
	}
	if !strings.HasSuffix(hostname, "."+LocalTLD) {
		return false
	}
	stem := strings.TrimSuffix(hostname, "."+LocalTLD)
	if stem == "" {
		return false
	}
	for _, label := range strings.Split(stem, ".") {
		if label == "" || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			switch {
			case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-':
			default:
				return false
			}
		}
	}
	return true
}

// IsLoopbackHost reports whether an address literal names this machine's
// loopback interface. It is the single gate every routing destination passes
// through: the local router must never be able to reach anything else, so a
// hostname (which could resolve anywhere, now or later) is refused outright
// and only a loopback IP literal is accepted.
func IsLoopbackHost(host string) bool {
	trimmed := strings.Trim(strings.TrimSpace(host), "[]")
	if trimmed == "" {
		return false
	}
	ip := net.ParseIP(trimmed)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}
