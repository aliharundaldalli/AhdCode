package ahdruntime

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"io"
	"math/big"
	"net"
	"net/netip"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---- DNS ---------------------------------------------------------------

func TestDNSLookupRejectsMalformedHostsWithoutNetwork(t *testing.T) {
	calls := 0
	original := ahdDNSResolve
	ahdDNSResolve = func(ctx context.Context, host string) ([]netip.Addr, error) {
		calls++
		return nil, errors.New("must not be called")
	}
	defer func() { ahdDNSResolve = original }()
	cases := map[string]string{
		"":                               "the host is empty",
		"   ":                            "only whitespace",
		"exa\x00mple.com":                "control character",
		"example.com\n":                  "control character",
		"https://example.com":            "not a URL",
		"https://example.com/path":       "not a URL",
		"example.com/path":               "path, query, or fragment",
		"example.com?x=1":                "path, query, or fragment",
		"example.com:443":                "must not contain a port",
		"user@example.com":               "user information",
		"example.com;touch X":            "whitespace",
		"example.com;rm":                 "only letters, digits",
		"example.com && whoami":          "whitespace",
		"$(whoami)":                      "only letters, digits",
		"`whoami`":                       "only letters, digits",
		"ex ample.com":                   "whitespace",
		"exämple.com":                    "non-ASCII",
		"a..b":                           "empty label",
		"-bad.example":                   "hyphen",
		"bad-.example":                   "hyphen",
		strings.Repeat("a", 64) + ".io":  "longer than 63",
		strings.Repeat("a.", 130) + "io": "longer than 253",
		"fe80::1%en0":                    "zone",
		"1.2.3.4]":                       "only letters, digits",
		"[1.2.3.4":                       "only letters, digits",
		"[::1":                           "must not contain a port",
	}
	for host, want := range cases {
		if _, err := DNSLookup(host, 5); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("DNS.lookup(%q) error = %v; want %q", host, err, want)
		}
	}
	for _, timeout := range []int64{0, -1, 61} {
		if _, err := DNSLookup("example.com", timeout); err == nil || !strings.Contains(err.Error(), "between 1 and 60") {
			t.Fatalf("timeout %d accepted: %v", timeout, err)
		}
	}
	if calls != 0 {
		t.Fatalf("the resolver was called %d times for invalid input", calls)
	}
}

func TestDNSLookupIPLiteralsNeedNoResolver(t *testing.T) {
	original := ahdDNSResolve
	ahdDNSResolve = func(ctx context.Context, host string) ([]netip.Addr, error) {
		return nil, errors.New("must not be called")
	}
	defer func() { ahdDNSResolve = original }()
	for input, want := range map[string]string{"127.0.0.1": "127.0.0.1", "::1": "::1", "[::1]": "::1", "::ffff:10.0.0.1": "10.0.0.1", "2001:DB8::1": "2001:db8::1"} {
		data, err := DNSLookup(input, 1)
		if err != nil {
			t.Fatalf("%s: %v", input, err)
		}
		host, _ := DNSResultHost(data)
		addresses, _ := DNSResultAddresses(data)
		if host != want || len(addresses) != 1 || addresses[0] != want {
			t.Fatalf("%s -> host %q addresses %v", input, host, addresses)
		}
	}
}

func TestDNSLookupNormalizesOrdersAndDeduplicates(t *testing.T) {
	original := ahdDNSResolve
	defer func() { ahdDNSResolve = original }()
	var asked string
	ahdDNSResolve = func(ctx context.Context, host string) ([]netip.Addr, error) {
		asked = host
		return []netip.Addr{
			netip.MustParseAddr("2001:db8::2"), netip.MustParseAddr("192.0.2.10"), netip.MustParseAddr("::ffff:192.0.2.9"),
			netip.MustParseAddr("192.0.2.10"), netip.MustParseAddr("2001:db8::1"), netip.MustParseAddr("192.0.2.9"),
		}, nil
	}
	data, err := DNSLookup("Panel.Example.COM.", 5)
	if err != nil {
		t.Fatal(err)
	}
	host, _ := DNSResultHost(data)
	addresses, _ := DNSResultAddresses(data)
	four, _ := DNSResultIPv4(data)
	six, _ := DNSResultIPv6(data)
	if asked != "panel.example.com" || host != "panel.example.com" {
		t.Fatalf("host normalization: asked %q, host %q", asked, host)
	}
	if strings.Join(addresses, ",") != "192.0.2.9,192.0.2.10,2001:db8::1,2001:db8::2" {
		t.Fatalf("addresses = %v", addresses)
	}
	if strings.Join(four, ",") != "192.0.2.9,192.0.2.10" || strings.Join(six, ",") != "2001:db8::1,2001:db8::2" {
		t.Fatalf("families = %v / %v", four, six)
	}
}

func TestDNSLookupFailuresAreClearAndBounded(t *testing.T) {
	original := ahdDNSResolve
	defer func() { ahdDNSResolve = original }()

	ahdDNSResolve = func(ctx context.Context, host string) ([]netip.Addr, error) {
		return nil, &net.DNSError{Err: "no such host", Name: host, Server: "10.9.8.7:53", IsNotFound: true}
	}
	if _, err := DNSLookup("missing.example", 5); err == nil || !strings.Contains(err.Error(), "host not found") || strings.Contains(err.Error(), "10.9.8.7") {
		t.Fatalf("not found: %v", err)
	}
	ahdDNSResolve = func(ctx context.Context, host string) ([]netip.Addr, error) {
		return nil, &net.DNSError{Err: "server misbehaving", Name: host, Server: "10.9.8.7:53", IsTemporary: true}
	}
	if _, err := DNSLookup("broken.example", 5); err == nil || !strings.Contains(err.Error(), "resolver could not answer") || strings.Contains(err.Error(), "10.9.8.7") {
		t.Fatalf("resolver failure: %v", err)
	}
	ahdDNSResolve = func(ctx context.Context, host string) ([]netip.Addr, error) { return nil, nil }
	if _, err := DNSLookup("empty.example", 5); err == nil || !strings.Contains(err.Error(), "no addresses") {
		t.Fatalf("empty answer must be an error: %v", err)
	}
	ahdDNSResolve = func(ctx context.Context, host string) ([]netip.Addr, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	started := time.Now()
	if _, err := DNSLookup("slow.example", 1); err == nil || !strings.Contains(err.Error(), "timed out after 1 seconds") {
		t.Fatalf("timeout: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("timeout took %v", elapsed)
	}
	expectRaise(t, AhdClassError, func() { AhdDNSLookup(AhdClassError, "", 5) })
}

func TestDNSLookupLocalhostThroughTheOperatingSystemResolver(t *testing.T) {
	data, err := DNSLookup("localhost", 5)
	if err != nil {
		t.Skipf("this host cannot resolve localhost: %v", err)
	}
	addresses, _ := DNSResultAddresses(data)
	found := false
	for _, address := range addresses {
		if address == "127.0.0.1" || address == "::1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("localhost resolved to %v", addresses)
	}
}

// ---- TLS fixtures --------------------------------------------------------

type tlsFixture struct {
	root     *x509.Certificate
	rootKey  *ecdsa.PrivateKey
	rootPool *x509.CertPool
}

func newTLSFixture(t *testing.T, name string) *tlsFixture {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: name, Organization: []string{"AhdCode Test"}},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := x509.ParseCertificate(der)
	pool := x509.NewCertPool()
	pool.AddCert(root)
	return &tlsFixture{root: root, rootKey: key, rootPool: pool}
}

func (fixture *tlsFixture) leaf(t *testing.T, names []string, ips []net.IP, notBefore, notAfter time.Time) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: names[0]},
		DNSNames: names, IPAddresses: ips, NotBefore: notBefore, NotAfter: notAfter,
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, fixture.root, &key.PublicKey, fixture.rootKey)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

// tlsServer serves one certificate on 127.0.0.1 and counts connections that
// were fully closed, so tests can prove inspection releases its socket.
type tlsServer struct {
	listener net.Listener
	port     int64
	closed   atomic.Int64
	group    sync.WaitGroup
}

func startTLSServer(t *testing.T, certificate tls.Certificate, maxVersion uint16) *tlsServer {
	t.Helper()
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{certificate}, MaxVersion: maxVersion})
	if err != nil {
		t.Fatal(err)
	}
	server := &tlsServer{listener: listener, port: int64(listener.Addr().(*net.TCPAddr).Port)}
	server.group.Add(1)
	go func() {
		defer server.group.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			server.group.Add(1)
			go func() {
				defer server.group.Done()
				_ = conn.(*tls.Conn).Handshake()
				_, _ = io.Copy(io.Discard, conn) // until the client closes
				_ = conn.Close()
				server.closed.Add(1)
			}()
		}
	}()
	t.Cleanup(func() { _ = listener.Close(); server.group.Wait() })
	return server
}

func useTestRoots(t *testing.T, pool *x509.CertPool) {
	t.Helper()
	previous := ahdTLSRoots
	ahdTLSRoots = pool
	t.Cleanup(func() { ahdTLSRoots = previous })
}

func inspectTLS(t *testing.T, host string, port int64) ahdTLSInfoData {
	t.Helper()
	data, err := TLSInspect(host, port, 5)
	if err != nil {
		t.Fatal(err)
	}
	info, err := ahdTLSDecode(data)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

// ---- TLS tests -----------------------------------------------------------

func TestTLSInspectValidCertificateWithSeveralNames(t *testing.T) {
	fixture := newTLSFixture(t, "AhdCode Test Root")
	useTestRoots(t, fixture.rootPool)
	notBefore := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	notAfter := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second)
	server := startTLSServer(t, fixture.leaf(t, []string{"localhost", "panel.localhost", "www.localhost"}, nil, notBefore, notAfter), 0)
	info := inspectTLS(t, "localhost", server.port)
	if !info.Valid || info.Status != AhdTLSStatusValid {
		t.Fatalf("valid certificate reported %v/%s", info.Valid, info.Status)
	}
	if info.Host != "localhost" || info.Port != server.port || info.Subject != "CN=localhost" ||
		info.Issuer != "CN=AhdCode Test Root,O=AhdCode Test" {
		t.Fatalf("metadata = %+v", info)
	}
	if strings.Join(info.DNSNames, ",") != "localhost,panel.localhost,www.localhost" {
		t.Fatalf("dnsNames = %v", info.DNSNames)
	}
	if info.NotBefore != notBefore.UnixMilli() || info.NotAfter != notAfter.UnixMilli() {
		t.Fatalf("validity = %d..%d; want %d..%d", info.NotBefore, info.NotAfter, notBefore.UnixMilli(), notAfter.UnixMilli())
	}
	if info.Protocol != "TLS 1.3" || info.CipherSuite == "" {
		t.Fatalf("protocol %q cipher %q", info.Protocol, info.CipherSuite)
	}
	older := startTLSServer(t, fixture.leaf(t, []string{"localhost"}, nil, notBefore, notAfter), tls.VersionTLS12)
	if got := inspectTLS(t, "localhost", older.port).Protocol; got != "TLS 1.2" {
		t.Fatalf("TLS 1.2 server reported %q", got)
	}
	// An IP literal is verified against the certificate's IP addresses.
	byIP := startTLSServer(t, fixture.leaf(t, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, notBefore, notAfter), 0)
	if info := inspectTLS(t, "127.0.0.1", byIP.port); !info.Valid {
		t.Fatalf("IP-literal inspection = %+v", info)
	}
	if info := inspectTLS(t, "127.0.0.1", server.port); info.Valid || info.Status != AhdTLSStatusHostnameMismatch {
		t.Fatalf("certificate without the IP must not verify for it: %+v", info)
	}
}

func TestTLSInspectKeepsMetadataForInvalidCertificates(t *testing.T) {
	fixture := newTLSFixture(t, "AhdCode Test Root")
	useTestRoots(t, fixture.rootPool)
	now := time.Now()
	cases := []struct {
		label  string
		cert   tls.Certificate
		status string
	}{
		{"expired", fixture.leaf(t, []string{"localhost"}, nil, now.Add(-48*time.Hour), now.Add(-24*time.Hour)), AhdTLSStatusExpired},
		{"not yet valid", fixture.leaf(t, []string{"localhost"}, nil, now.Add(24*time.Hour), now.Add(48*time.Hour)), AhdTLSStatusNotYetValid},
		{"hostname mismatch", fixture.leaf(t, []string{"other.example", "second.example"}, nil, now.Add(-time.Hour), now.Add(time.Hour)), AhdTLSStatusHostnameMismatch},
		{"untrusted issuer", newTLSFixture(t, "Stranger Root").leaf(t, []string{"localhost"}, nil, now.Add(-time.Hour), now.Add(time.Hour)), AhdTLSStatusUntrustedIssuer},
	}
	for _, testCase := range cases {
		server := startTLSServer(t, testCase.cert, 0)
		info := inspectTLS(t, "localhost", server.port)
		if info.Valid || info.Status != testCase.status {
			t.Fatalf("%s: valid=%v status=%q; want false/%q", testCase.label, info.Valid, info.Status, testCase.status)
		}
		if info.Subject == "" || info.Issuer == "" || len(info.DNSNames) == 0 || info.NotAfter == 0 || info.Protocol == "" {
			t.Fatalf("%s lost its metadata: %+v", testCase.label, info)
		}
	}
	// Without the test root (the system roots), the test CA is untrusted: an
	// inspection never trusts what the server presents on its own.
	ahdTLSRoots = nil
	server := startTLSServer(t, fixture.leaf(t, []string{"localhost"}, nil, now.Add(-time.Hour), now.Add(time.Hour)), 0)
	if info := inspectTLS(t, "localhost", server.port); info.Valid || info.Status != AhdTLSStatusUntrustedIssuer {
		t.Fatalf("system roots trusted a private test CA: %+v", info)
	}
}

func TestTLSInspectConnectionFailuresAreErrors(t *testing.T) {
	// Connection refused: a port that was just released.
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	refusedPort := int64(listener.Addr().(*net.TCPAddr).Port)
	_ = listener.Close()
	if _, err := TLSInspect("127.0.0.1", refusedPort, 5); err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("refused: %v", err)
	}

	// A stalled server accepts TCP and never speaks.
	stalled, _ := net.Listen("tcp", "127.0.0.1:0")
	var held []net.Conn
	var mutex sync.Mutex
	go func() {
		for {
			conn, err := stalled.Accept()
			if err != nil {
				return
			}
			mutex.Lock()
			held = append(held, conn)
			mutex.Unlock()
		}
	}()
	defer func() {
		_ = stalled.Close()
		mutex.Lock()
		for _, conn := range held {
			_ = conn.Close()
		}
		mutex.Unlock()
	}()
	started := time.Now()
	if _, err := TLSInspect("127.0.0.1", int64(stalled.Addr().(*net.TCPAddr).Port), 1); err == nil || !strings.Contains(err.Error(), "timed out after 1 seconds") {
		t.Fatalf("stalled server: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 4*time.Second {
		t.Fatalf("timeout took %v", elapsed)
	}

	// A plain-text (non-TLS) server.
	plain, _ := net.Listen("tcp", "127.0.0.1:0")
	go func() {
		for {
			conn, err := plain.Accept()
			if err != nil {
				return
			}
			_, _ = conn.Write([]byte("HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\n\r\n"))
			_ = conn.Close()
		}
	}()
	defer plain.Close()
	if _, err := TLSInspect("127.0.0.1", int64(plain.Addr().(*net.TCPAddr).Port), 5); err == nil || !strings.Contains(err.Error(), "did not answer with TLS") {
		t.Fatalf("plain server: %v", err)
	}

	for _, bad := range []int64{0, -1, 65536} {
		if _, err := TLSInspect("localhost", bad, 5); err == nil || !strings.Contains(err.Error(), "port must be between 1 and 65535") {
			t.Fatalf("port %d accepted: %v", bad, err)
		}
	}
	for _, bad := range []int64{0, 61} {
		if _, err := TLSInspect("localhost", 443, bad); err == nil || !strings.Contains(err.Error(), "between 1 and 60") {
			t.Fatalf("timeout %d accepted: %v", bad, err)
		}
	}
	for _, host := range []string{"https://example.com", "example.com:443", "example.com/path", "$(whoami)", "example.com;touch X"} {
		if _, err := TLSInspect(host, 443, 5); err == nil || !strings.Contains(err.Error(), "failed: the host") {
			t.Fatalf("host %q accepted: %v", host, err)
		}
	}
	expectRaise(t, AhdClassError, func() { AhdTLSInspect(AhdClassError, "", 443, 5) })
}

func TestTLSInspectClosesItsConnectionsAndLeaksNoGoroutines(t *testing.T) {
	fixture := newTLSFixture(t, "AhdCode Test Root")
	useTestRoots(t, fixture.rootPool)
	server := startTLSServer(t, fixture.leaf(t, []string{"localhost"}, nil, time.Now().Add(-time.Hour), time.Now().Add(time.Hour)), 0)
	before := runtime.NumGoroutine()
	const rounds = 20
	for index := 0; index < rounds; index++ {
		inspectTLS(t, "127.0.0.1", server.port)
	}
	deadline := time.Now().Add(5 * time.Second)
	for server.closed.Load() < rounds && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if closed := server.closed.Load(); closed != rounds {
		t.Fatalf("the server saw %d of %d connections closed", closed, rounds)
	}
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before {
		t.Fatalf("goroutines grew from %d to %d", before, after)
	}
	_ = strconv.Itoa
}
