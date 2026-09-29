package build

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// localUntrustedTLSServer serves a self-signed certificate for localhost with
// fixed validity dates, so the result is identical in the evaluator and in a
// native child process (neither trusts it, so no trust override is needed).
func localUntrustedTLSServer(t *testing.T, notBefore, notAfter time.Time) int {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(42), Subject: pkix.Name{CommonName: "localhost", Organization: []string{"AhdCode Parity"}},
		DNSNames: []string{"localhost", "panel.localhost"}, NotBefore: notBefore, NotAfter: notAfter,
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}, MaxVersion: tls.VersionTLS13,
	})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				_ = conn.(*tls.Conn).Handshake()
				_, _ = io.Copy(io.Discard, conn)
				_ = conn.Close()
			}()
		}
	}()
	t.Cleanup(func() { _ = listener.Close() })
	return listener.Addr().(*net.TCPAddr).Port
}

// TestNetworkInspectionMatchesBetweenEvaluatorAndNative runs one DNS/TLS
// program natively and in the evaluator and requires identical output. It
// deliberately does not bring Time: TLSInfo.notAfter() still returns a
// DateTime through the implicit TLS -> Time dependency.
func TestNetworkInspectionMatchesBetweenEvaluatorAndNative(t *testing.T) {
	expired := localUntrustedTLSServer(t, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2021, 6, 30, 12, 0, 0, 0, time.UTC))
	// A normal 30-day validity: macOS's platform verifier classes very long
	// server-certificate validity as not standards compliant ("invalid").
	currentUntil := time.Now().Add(30 * 24 * time.Hour)
	current := localUntrustedTLSServer(t, time.Now().Add(-time.Hour), currentUntil)
	refused, _ := net.Listen("tcp", "127.0.0.1:0")
	refusedPort := refused.Addr().(*net.TCPAddr).Port
	_ = refused.Close()
	source := `bring DNS
bring TLS
from DNS bring (DNSResult, DNSError)
from TLS bring (TLSInfo, TLSError)

literal: DNSResult := DNS.lookup(host: "127.0.0.1", timeoutSeconds: 2)
write(literal.host() + " " + str(literal.addresses()) + " " + str(literal.ipv4()) + " " + str(literal.ipv6()))
local := DNS.lookup("localhost")
write("localhost has loopback: " + str("127.0.0.1" in local.addresses() or "::1" in local.addresses()))
for bad in ["", "https://example.com/path", "example.com:443", "example.com;touch X", "$(whoami)"] {
    attempt {
        DNS.lookup(bad)
    }
    except DNSError as failure {
        write(failure.message)
    }
}
attempt {
    DNS.lookup(host: "example.com", timeoutSeconds: 0)
}
except Error as failure {
    write(failure.message)
}

info: TLSInfo := TLS.inspect(host: "localhost", port: ` + strconv.Itoa(expired) + `, timeoutSeconds: 5)
write(str(info.valid()) + " " + info.verificationStatus() + " " + info.subject() + " | " + info.issuer())
write(str(info.dnsNames()) + " " + info.protocol() + " " + str(info.port()) + " " + info.host())
write(str(info.notBefore().year) + "-" + str(info.notBefore().month) + " .. " + str(info.notAfter().year) + "-" + str(info.notAfter().month) + "-" + str(info.notAfter().day) + " offset " + str(info.notAfter().offsetMinutes))
fresh := TLS.inspect("localhost", ` + strconv.Itoa(current) + `)
write(str(fresh.valid()) + " " + fresh.verificationStatus() + " until " + str(fresh.notAfter().year))
attempt {
    TLS.inspect(host: "127.0.0.1", port: ` + strconv.Itoa(refusedPort) + `, timeoutSeconds: 5)
}
except TLSError as failure {
    write(failure.message)
}
attempt {
    TLS.inspect(host: "localhost", port: 70000)
}
except Error as failure {
    write(failure.message)
}
`
	directory := t.TempDir()
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	native, stderr, code := buildAndRunIn(t, entry, directory)
	if code != 0 || stderr != "" {
		t.Fatalf("native exit %d stderr %q:\n%s", code, stderr, native)
	}
	var output, errorOutput bytes.Buffer
	runEvaluatorIn(t, directory, source, &output, &errorOutput)
	if output.String() != native {
		t.Fatalf("evaluator and native differ\nnative:\n%s\nevaluator:\n%s\nstderr: %s", native, output.String(), errorOutput.String())
	}
	for _, want := range []string{
		`127.0.0.1 ["127.0.0.1"] ["127.0.0.1"] []`,
		"localhost has loopback: true",
		`lookup "" failed: the host is empty`,
		`lookup "https://example.com/path" failed: the host must be a host name such as example.com, not a URL`,
		`lookup "example.com:443" failed: the host must not contain a port; pass the port separately`,
		`lookup "example.com;touch X" failed: the host contains whitespace`,
		`lookup "$(whoami)" failed: the host contains '$'`,
		`timeoutSeconds must be between 1 and 60; received 0`,
		"false expired CN=localhost,O=AhdCode Parity | CN=localhost,O=AhdCode Parity",
		`["localhost", "panel.localhost"] TLS 1.3 ` + strconv.Itoa(expired) + " localhost",
		"2020-1 .. 2021-6-30 offset 0",
		"false untrustedIssuer until " + strconv.Itoa(currentUntil.UTC().Year()),
		"connection refused",
		"port must be between 1 and 65535; received 70000",
	} {
		if !strings.Contains(native, want) {
			t.Fatalf("output lacks %q:\n%s", want, native)
		}
	}
}

// TestV260ExamplesCompile keeps examples/v2.6 compiling. They are only
// compiled: running them needs public network access.
func TestV260ExamplesCompile(t *testing.T) {
	for _, name := range []string{"dns_lookup", "tls_inspect"} {
		entry, err := filepath.Abs(filepath.Join("..", "..", "examples", "v2.6", name, "main.ahd"))
		if err != nil {
			t.Fatal(err)
		}
		if _, result := BuildProgram(entry, filepath.Join(t.TempDir(), name)); result.HasErrors() {
			t.Fatalf("%s does not compile:\n%s", name, diagnosticText(result.Diagnostics))
		}
	}
}
