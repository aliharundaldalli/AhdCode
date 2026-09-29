package ahdruntime

// The v2.6.0 network-inspection primitives: DNS.lookup (address resolution)
// and TLS.inspect (certificate and handshake inspection). Built only on the Go
// standard library (net, net/netip, crypto/tls, crypto/x509, context), so this
// file is emitted verbatim into native programs and called directly by the
// evaluator. Nothing here runs a shell or an external tool (no dig, nslookup,
// or openssl), every operation has a finite timeout, and every connection is
// closed before the call returns.
//
// Inspection is not trust. TLS.inspect disables Go's automatic verification on
// the socket only to obtain the certificate chain the server presents, sends
// no application data, closes the connection, and then verifies the chain
// explicitly against the system roots, the requested host, and the current
// time. valid() is true only when that explicit verification succeeds.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Bounds. Both operations are always finite.
const (
	AhdDNSDefaultTimeoutSeconds = int64(5)
	AhdTLSDefaultTimeoutSeconds = int64(10)
	AhdTLSDefaultPort           = int64(443)
	ahdNetworkMaxTimeoutSeconds = int64(60)
	ahdNetworkMaxHostLength     = 253
)

// ahdDNSResolve is the resolver seam. Production uses the operating-system /
// Go resolver; tests replace it. It is unexported, so an AhdCode program can
// never reach it.
var ahdDNSResolve = func(ctx context.Context, host string) ([]netip.Addr, error) {
	return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
}

// ahdTLSRoots, when set by a test, replaces the system trust roots. It is
// unexported; production always verifies against the system roots.
var ahdTLSRoots *x509.CertPool

func ahdDNSFail(host, reason string) error {
	return errors.New("lookup " + strconv.Quote(host) + " failed: " + reason)
}

func ahdTLSFail(host string, port int64, reason string) error {
	return errors.New("inspect " + strconv.Quote(net.JoinHostPort(host, strconv.FormatInt(port, 10))) + " failed: " + reason)
}

// ahdNetworkHost validates a host argument and returns its canonical form:
// a lowercase host name without a trailing dot, or the canonical text of an IP
// literal. It never touches the network. URLs, paths, ports, user info,
// whitespace, control characters, and non-ASCII names are rejected with a
// reason that says what to pass instead.
func ahdNetworkHost(host string) (string, netip.Addr, string) {
	switch {
	case host == "":
		return "", netip.Addr{}, "the host is empty"
	case strings.TrimSpace(host) == "":
		return "", netip.Addr{}, "the host is only whitespace"
	case len(host) > ahdNetworkMaxHostLength+1:
		return "", netip.Addr{}, fmt.Sprintf("the host is longer than %d characters", ahdNetworkMaxHostLength)
	case strings.Contains(host, "://"):
		return "", netip.Addr{}, "the host must be a host name such as example.com, not a URL"
	}
	for _, r := range host {
		switch {
		case r < 0x20 || r == 0x7f:
			return "", netip.Addr{}, "the host contains a control character"
		case r == ' ' || r == '\t':
			return "", netip.Addr{}, "the host contains whitespace"
		case r > 0x7e:
			return "", netip.Addr{}, "the host contains a non-ASCII character; pass an internationalized name in its ASCII (xn--) form"
		}
	}
	if strings.ContainsAny(host, "/?#") {
		return "", netip.Addr{}, "the host must not contain a path, query, or fragment"
	}
	if strings.Contains(host, "@") {
		return "", netip.Addr{}, "the host must not contain user information"
	}
	literal := host
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		literal = host[1 : len(host)-1] // a bracketed IPv6 literal such as [::1]
	}
	if address, err := netip.ParseAddr(literal); err == nil {
		if address.Zone() != "" {
			return "", netip.Addr{}, "the host must not carry an IPv6 zone"
		}
		address = address.Unmap()
		return address.String(), address, ""
	}
	if strings.Contains(host, ":") {
		return "", netip.Addr{}, "the host must not contain a port; pass the port separately"
	}
	name := strings.ToLower(strings.TrimSuffix(host, "."))
	if name == "" || len(name) > ahdNetworkMaxHostLength {
		return "", netip.Addr{}, "the host is not a valid host name"
	}
	for _, label := range strings.Split(name, ".") {
		if label == "" {
			return "", netip.Addr{}, "the host has an empty label"
		}
		if len(label) > 63 {
			return "", netip.Addr{}, "a host label is longer than 63 characters"
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return "", netip.Addr{}, "a host label must not start or end with a hyphen"
		}
		for _, r := range label {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
				return "", netip.Addr{}, "the host contains " + strconv.QuoteRune(r) + "; only letters, digits, hyphens, underscores, and dots are allowed"
			}
		}
	}
	return name, netip.Addr{}, ""
}

func ahdNetworkTimeout(timeoutSeconds int64) string {
	if timeoutSeconds < 1 || timeoutSeconds > ahdNetworkMaxTimeoutSeconds {
		return fmt.Sprintf("timeoutSeconds must be between 1 and %d; received %d", ahdNetworkMaxTimeoutSeconds, timeoutSeconds)
	}
	return ""
}

// --- DNS ---------------------------------------------------------------

type ahdDNSResultData struct {
	Host      string   `json:"host"`
	Addresses []string `json:"addresses"`
}

// ahdDNSOrder sorts addresses deterministically: every IPv4 address before
// every IPv6 address, each family in ascending numeric order.
func ahdDNSOrder(addresses []netip.Addr) []string {
	seen := make(map[netip.Addr]bool, len(addresses))
	unique := make([]netip.Addr, 0, len(addresses))
	for _, address := range addresses {
		address = address.Unmap()
		if !address.IsValid() || seen[address] {
			continue
		}
		seen[address] = true
		unique = append(unique, address)
	}
	sort.Slice(unique, func(i, j int) bool {
		if unique[i].Is4() != unique[j].Is4() {
			return unique[i].Is4()
		}
		return unique[i].Less(unique[j])
	})
	result := make([]string, len(unique))
	for index, address := range unique {
		result[index] = address.String()
	}
	return result
}

// DNSLookup resolves host to its addresses (A and AAAA) with the operating
// system's resolver, bounded by timeoutSeconds. An IP literal is returned as
// itself without any network access. The addresses are deduplicated and
// sorted (IPv4 first, then IPv6, each ascending); a name with no addresses is
// an error, never an empty success.
func DNSLookup(host string, timeoutSeconds int64) (string, error) {
	name, literal, reason := ahdNetworkHost(host)
	if reason != "" {
		return "", ahdDNSFail(host, reason)
	}
	if reason := ahdNetworkTimeout(timeoutSeconds); reason != "" {
		return "", ahdDNSFail(host, reason)
	}
	var addresses []string
	if literal.IsValid() {
		addresses = []string{literal.String()}
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
		found, err := ahdDNSResolve(ctx, name)
		if err != nil {
			return "", ahdDNSFail(host, ahdDNSReason(err, ctx, timeoutSeconds))
		}
		addresses = ahdDNSOrder(found)
		if len(addresses) == 0 {
			return "", ahdDNSFail(host, "no addresses were found for the host")
		}
	}
	encoded, err := json.Marshal(ahdDNSResultData{Host: name, Addresses: addresses})
	if err != nil {
		return "", ahdDNSFail(host, err.Error())
	}
	return string(encoded), nil
}

// ahdDNSReason phrases a resolver failure without resolver configuration
// (such as the DNS server address Go puts in net.DNSError).
func ahdDNSReason(err error, ctx context.Context, timeoutSeconds int64) string {
	var dnsError *net.DNSError
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded):
		return fmt.Sprintf("timed out after %d seconds (timeoutSeconds)", timeoutSeconds)
	case errors.As(err, &dnsError) && dnsError.IsNotFound:
		return "host not found"
	case errors.As(err, &dnsError) && dnsError.IsTimeout:
		return fmt.Sprintf("timed out after %d seconds (timeoutSeconds)", timeoutSeconds)
	case errors.As(err, &dnsError) && dnsError.IsTemporary:
		return "the resolver could not answer (temporary failure)"
	}
	return "the resolver could not answer"
}

func ahdDNSDecode(data string) (ahdDNSResultData, error) {
	var result ahdDNSResultData
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return result, errors.New("DNSResult storage is corrupted")
	}
	return result, nil
}

func DNSResultHost(data string) (string, error) {
	result, err := ahdDNSDecode(data)
	return result.Host, err
}

func DNSResultAddresses(data string) ([]string, error) {
	result, err := ahdDNSDecode(data)
	return append([]string{}, result.Addresses...), err
}

func ahdDNSFamily(data string, four bool) ([]string, error) {
	result, err := ahdDNSDecode(data)
	selected := []string{}
	for _, text := range result.Addresses {
		if address, parseErr := netip.ParseAddr(text); parseErr == nil && address.Is4() == four {
			selected = append(selected, text)
		}
	}
	return selected, err
}

func DNSResultIPv4(data string) ([]string, error) { return ahdDNSFamily(data, true) }
func DNSResultIPv6(data string) ([]string, error) { return ahdDNSFamily(data, false) }

// --- TLS ---------------------------------------------------------------

// Verification statuses, in the order they are decided. The first matching
// problem is reported; "valid" means full verification succeeded.
const (
	AhdTLSStatusValid            = "valid"
	AhdTLSStatusExpired          = "expired"
	AhdTLSStatusNotYetValid      = "notYetValid"
	AhdTLSStatusHostnameMismatch = "hostnameMismatch"
	AhdTLSStatusUntrustedIssuer  = "untrustedIssuer"
	AhdTLSStatusInvalid          = "invalid"
)

type ahdTLSInfoData struct {
	Host        string   `json:"host"`
	Port        int64    `json:"port"`
	Valid       bool     `json:"valid"`
	Status      string   `json:"status"`
	Subject     string   `json:"subject"`
	Issuer      string   `json:"issuer"`
	DNSNames    []string `json:"dnsNames"`
	NotBefore   int64    `json:"notBefore"` // Unix milliseconds, UTC
	NotAfter    int64    `json:"notAfter"`
	Protocol    string   `json:"protocol"`
	CipherSuite string   `json:"cipherSuite"`
}

// ahdTLSVerify decides the verification status of a presented chain for host
// at now. valid is true only when x509 verification against the roots, with
// the peer's intermediates, the host name, and the current time succeeds.
func ahdTLSVerify(chain []*x509.Certificate, host string, literal netip.Addr, now time.Time) (bool, string) {
	leaf := chain[0]
	intermediates := x509.NewCertPool()
	for _, certificate := range chain[1:] {
		intermediates.AddCert(certificate)
	}
	options := x509.VerifyOptions{
		DNSName: host, Intermediates: intermediates, Roots: ahdTLSRoots, CurrentTime: now,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if _, err := leaf.Verify(options); err == nil {
		return true, AhdTLSStatusValid
	}
	switch {
	case now.After(leaf.NotAfter):
		return false, AhdTLSStatusExpired
	case now.Before(leaf.NotBefore):
		return false, AhdTLSStatusNotYetValid
	}
	hostCheck := host
	if literal.IsValid() {
		hostCheck = literal.String()
	}
	if leaf.VerifyHostname(hostCheck) != nil {
		return false, AhdTLSStatusHostnameMismatch
	}
	options.DNSName = ""
	if _, err := leaf.Verify(options); err != nil && ahdTLSUntrusted(err) {
		return false, AhdTLSStatusUntrustedIssuer
	}
	return false, AhdTLSStatusInvalid
}

// ahdTLSUntrusted recognizes "no trusted root" across verifiers: Go's own
// verifier (Linux, test roots) and Windows return UnknownAuthorityError; the
// macOS platform verifier reports "certificate is not trusted".
func ahdTLSUntrusted(err error) bool {
	var unknown x509.UnknownAuthorityError
	return errors.As(err, &unknown) || strings.HasSuffix(err.Error(), "certificate is not trusted")
}

// TLSInspect connects to host:port, performs a TLS handshake with SNI set to
// host (no SNI for an IP literal), collects the presented certificate chain,
// closes the connection, and verifies the chain explicitly. The whole
// operation (connect, handshake, verification) is bounded by timeoutSeconds.
// A reachable server with an invalid certificate is a successful inspection
// whose valid() is false; failures to obtain a certificate at all are errors.
func TLSInspect(host string, port, timeoutSeconds int64) (string, error) {
	name, literal, reason := ahdNetworkHost(host)
	if reason != "" {
		return "", ahdTLSFail(host, port, reason)
	}
	if port < 1 || port > 65535 {
		return "", ahdTLSFail(host, port, fmt.Sprintf("port must be between 1 and 65535; received %d", port))
	}
	if reason := ahdNetworkTimeout(timeoutSeconds); reason != "" {
		return "", ahdTLSFail(host, port, reason)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	deadline, _ := ctx.Deadline()
	address := net.JoinHostPort(name, strconv.FormatInt(port, 10))
	var dialer net.Dialer
	raw, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return "", ahdTLSFail(host, port, ahdTLSDialReason(err, ctx, timeoutSeconds))
	}
	defer raw.Close()
	_ = raw.SetDeadline(deadline)
	config := &tls.Config{
		// Automatic verification is off only to read the presented chain;
		// the chain is verified explicitly below and no application data
		// is ever exchanged over this connection.
		InsecureSkipVerify: true, //nolint:gosec
		MinVersion:         tls.VersionTLS10,
	}
	if !literal.IsValid() {
		config.ServerName = name
	}
	client := tls.Client(raw, config)
	handshakeErr := client.HandshakeContext(ctx)
	state := client.ConnectionState()
	_ = client.Close()
	if handshakeErr != nil {
		return "", ahdTLSFail(host, port, ahdTLSHandshakeReason(handshakeErr, ctx, timeoutSeconds))
	}
	if len(state.PeerCertificates) == 0 {
		return "", ahdTLSFail(host, port, "the server presented no certificate")
	}
	leaf := state.PeerCertificates[0]
	valid, status := ahdTLSVerify(state.PeerCertificates, name, literal, time.Now())
	names := append([]string{}, leaf.DNSNames...)
	encoded, err := json.Marshal(ahdTLSInfoData{
		Host: name, Port: port, Valid: valid, Status: status,
		Subject: leaf.Subject.String(), Issuer: leaf.Issuer.String(), DNSNames: names,
		NotBefore: leaf.NotBefore.UTC().UnixMilli(), NotAfter: leaf.NotAfter.UTC().UnixMilli(),
		Protocol: tls.VersionName(state.Version), CipherSuite: tls.CipherSuiteName(state.CipherSuite),
	})
	if err != nil {
		return "", ahdTLSFail(host, port, err.Error())
	}
	return string(encoded), nil
}

func ahdTLSDialReason(err error, ctx context.Context, timeoutSeconds int64) string {
	var dnsError *net.DNSError
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded):
		return fmt.Sprintf("timed out after %d seconds (timeoutSeconds)", timeoutSeconds)
	case errors.As(err, &dnsError) && dnsError.IsNotFound:
		return "host not found"
	case errors.As(err, &dnsError):
		return "the host could not be resolved"
	case ahdNetworkErrno(err, syscall.ECONNREFUSED, 10061): // WSAECONNREFUSED
		return "connection refused"
	case ahdNetworkErrno(err, syscall.ENETUNREACH, 10051) || ahdNetworkErrno(err, syscall.EHOSTUNREACH, 10065):
		return "the host is unreachable"
	}
	return "could not connect"
}

// ahdNetworkErrno matches a Unix errno or its Windows Winsock counterpart.
func ahdNetworkErrno(err error, unix syscall.Errno, winsock uintptr) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	return errno == unix || uintptr(errno) == winsock
}

func ahdTLSHandshakeReason(err error, ctx context.Context, timeoutSeconds int64) string {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, context.DeadlineExceeded):
		return fmt.Sprintf("timed out after %d seconds (timeoutSeconds)", timeoutSeconds)
	case errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF):
		return "the server closed the connection during the TLS handshake"
	}
	var record tls.RecordHeaderError
	if errors.As(err, &record) {
		return "the server did not answer with TLS"
	}
	var alert tls.AlertError
	if errors.As(err, &alert) {
		return "the server refused the TLS handshake (" + alert.Error() + ")"
	}
	return "the TLS handshake failed"
}

func ahdTLSDecode(data string) (ahdTLSInfoData, error) {
	var result ahdTLSInfoData
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return result, errors.New("TLSInfo storage is corrupted")
	}
	return result, nil
}

func TLSInfoHost(data string) (string, error) {
	info, err := ahdTLSDecode(data)
	return info.Host, err
}

func TLSInfoPort(data string) (int64, error) {
	info, err := ahdTLSDecode(data)
	return info.Port, err
}

func TLSInfoValid(data string) (bool, error) {
	info, err := ahdTLSDecode(data)
	return info.Valid, err
}

func TLSInfoVerificationStatus(data string) (string, error) {
	info, err := ahdTLSDecode(data)
	return info.Status, err
}

func TLSInfoSubject(data string) (string, error) {
	info, err := ahdTLSDecode(data)
	return info.Subject, err
}

func TLSInfoIssuer(data string) (string, error) {
	info, err := ahdTLSDecode(data)
	return info.Issuer, err
}

func TLSInfoDNSNames(data string) ([]string, error) {
	info, err := ahdTLSDecode(data)
	return append([]string{}, info.DNSNames...), err
}

// TLSInfoNotBefore and TLSInfoNotAfter return Unix milliseconds (UTC); the
// callers turn them into the Time module's DateTime.
func TLSInfoNotBefore(data string) (int64, error) {
	info, err := ahdTLSDecode(data)
	return info.NotBefore, err
}

func TLSInfoNotAfter(data string) (int64, error) {
	info, err := ahdTLSDecode(data)
	return info.NotAfter, err
}

func TLSInfoProtocol(data string) (string, error) {
	info, err := ahdTLSDecode(data)
	return info.Protocol, err
}

func TLSInfoCipherSuite(data string) (string, error) {
	info, err := ahdTLSDecode(data)
	return info.CipherSuite, err
}

// --- native wrappers: raise the program's DNSError / TLSError ---

func ahdNetworkRaise(class *AhdClass, err error) {
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
}

func AhdDNSLookup(class *AhdClass, host string, timeoutSeconds int64) string {
	data, err := DNSLookup(host, timeoutSeconds)
	ahdNetworkRaise(class, err)
	return data
}

func AhdTLSInspect(class *AhdClass, host string, port, timeoutSeconds int64) string {
	data, err := TLSInspect(host, port, timeoutSeconds)
	ahdNetworkRaise(class, err)
	return data
}

func ahdNetworkStrings(class *AhdClass, values []string, err error) *AhdList[string] {
	ahdNetworkRaise(class, err)
	return AhdNewList(values...)
}

func AhdDNSResultHost(class *AhdClass, data string) string {
	value, err := DNSResultHost(data)
	ahdNetworkRaise(class, err)
	return value
}

func AhdDNSResultAddresses(class *AhdClass, data string) *AhdList[string] {
	values, err := DNSResultAddresses(data)
	return ahdNetworkStrings(class, values, err)
}

func AhdDNSResultIPv4(class *AhdClass, data string) *AhdList[string] {
	values, err := DNSResultIPv4(data)
	return ahdNetworkStrings(class, values, err)
}

func AhdDNSResultIPv6(class *AhdClass, data string) *AhdList[string] {
	values, err := DNSResultIPv6(data)
	return ahdNetworkStrings(class, values, err)
}

func AhdTLSInfoString(class *AhdClass, data string, read func(string) (string, error)) string {
	value, err := read(data)
	ahdNetworkRaise(class, err)
	return value
}

func AhdTLSInfoHost(class *AhdClass, data string) string {
	return AhdTLSInfoString(class, data, TLSInfoHost)
}

func AhdTLSInfoVerificationStatus(class *AhdClass, data string) string {
	return AhdTLSInfoString(class, data, TLSInfoVerificationStatus)
}

func AhdTLSInfoSubject(class *AhdClass, data string) string {
	return AhdTLSInfoString(class, data, TLSInfoSubject)
}

func AhdTLSInfoIssuer(class *AhdClass, data string) string {
	return AhdTLSInfoString(class, data, TLSInfoIssuer)
}

func AhdTLSInfoProtocol(class *AhdClass, data string) string {
	return AhdTLSInfoString(class, data, TLSInfoProtocol)
}

func AhdTLSInfoCipherSuite(class *AhdClass, data string) string {
	return AhdTLSInfoString(class, data, TLSInfoCipherSuite)
}

func AhdTLSInfoPort(class *AhdClass, data string) int64 {
	value, err := TLSInfoPort(data)
	ahdNetworkRaise(class, err)
	return value
}

func AhdTLSInfoValid(class *AhdClass, data string) bool {
	value, err := TLSInfoValid(data)
	ahdNetworkRaise(class, err)
	return value
}

func AhdTLSInfoDNSNames(class *AhdClass, data string) *AhdList[string] {
	values, err := TLSInfoDNSNames(data)
	return ahdNetworkStrings(class, values, err)
}

// AhdTLSInfoNotBefore and AhdTLSInfoNotAfter return the civil UTC reading the
// generated DateTime helper turns into a DateTime value.
func AhdTLSInfoNotBefore(class *AhdClass, data string) AhdCivilTime {
	value, err := TLSInfoNotBefore(data)
	ahdNetworkRaise(class, err)
	return AhdTimeFromTimestamp(value)
}

func AhdTLSInfoNotAfter(class *AhdClass, data string) AhdCivilTime {
	value, err := TLSInfoNotAfter(data)
	ahdNetworkRaise(class, err)
	return AhdTimeFromTimestamp(value)
}
