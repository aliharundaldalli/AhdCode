package ahdruntime

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"io"
	"math/big"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const smtpQAPassword = "SMTP_QA_PASSWORD_7f91_secret"

type smtpCapture struct {
	mu           sync.Mutex
	connections  int
	commands     []string
	authSeen     bool
	authPayload  string
	mailFrom     string
	recipients   []string
	dataCount    int
	data         []byte
	transactions int
}

func (c *smtpCapture) addCommand(line string) {
	c.mu.Lock()
	c.commands = append(c.commands, line)
	c.mu.Unlock()
}

type smtpFixtureOptions struct {
	security           string
	advertiseSTARTTLS  bool
	advertiseAuthPlain bool
	authUser           string
	authPass           string
	rejectRecipient    string
	stall              time.Duration
	failDATA           bool
	certificate        *tls.Certificate
}

func startSMTPFixture(t *testing.T, options smtpFixtureOptions) (*smtpCapture, string, int) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if options.security == "tls" {
		if options.certificate == nil {
			t.Fatal("implicit TLS fixture requires a certificate")
		}
		listener = tls.NewListener(listener, &tls.Config{Certificates: []tls.Certificate{*options.certificate}})
	}
	capture := &smtpCapture{}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go serveSMTPFixture(conn, capture, options)
		}
	}()
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	portNum, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	return capture, host, portNum
}

func serveSMTPFixture(conn net.Conn, capture *smtpCapture, options smtpFixtureOptions) {
	defer conn.Close()
	capture.mu.Lock()
	capture.connections++
	capture.mu.Unlock()
	if options.stall > 0 {
		time.Sleep(options.stall)
	}
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	write := func(line string) {
		_, _ = writer.WriteString(line + "\r\n")
		_ = writer.Flush()
	}
	write("220 smtp.test ESMTP")
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		capture.addCommand(line)
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO") || strings.HasPrefix(upper, "HELO"):
			write("250-smtp.test")
			if options.advertiseSTARTTLS && options.security != "tls" {
				write("250-STARTTLS")
			}
			if options.advertiseAuthPlain {
				write("250-AUTH PLAIN")
			}
			write("250 OK")
		case strings.HasPrefix(upper, "STARTTLS"):
			if !options.advertiseSTARTTLS || options.certificate == nil {
				write("502 STARTTLS not available")
				continue
			}
			write("220 Ready to start TLS")
			tlsConn := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{*options.certificate}})
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			conn = tlsConn
			reader = bufio.NewReader(conn)
			writer = bufio.NewWriter(conn)
		case strings.HasPrefix(upper, "AUTH PLAIN"):
			capture.mu.Lock()
			capture.authSeen = true
			capture.authPayload = strings.TrimSpace(line[len("AUTH PLAIN"):])
			capture.mu.Unlock()
			decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(line[len("AUTH PLAIN"):]))
			if err != nil {
				write("501 invalid AUTH")
				continue
			}
			parts := bytes.Split(decoded, []byte{0})
			user, pass := "", ""
			if len(parts) == 3 {
				user, pass = string(parts[1]), string(parts[2])
			} else if len(parts) == 2 {
				user, pass = string(parts[0]), string(parts[1])
			}
			if options.authUser != "" && user == options.authUser && pass == options.authPass {
				write("235 Authentication successful")
			} else {
				write("535 Authentication failed")
			}
		case strings.HasPrefix(upper, "MAIL FROM:"):
			capture.mu.Lock()
			capture.mailFrom = strings.Trim(strings.TrimPrefix(line, line[:9]), " <>")
			if strings.Contains(upper, "<") {
				start := strings.Index(line, "<")
				end := strings.Index(line, ">")
				if start >= 0 && end > start {
					capture.mailFrom = line[start+1 : end]
				}
			}
			capture.mu.Unlock()
			write("250 OK")
		case strings.HasPrefix(upper, "RCPT TO:"):
			recipient := ""
			start := strings.Index(line, "<")
			end := strings.Index(line, ">")
			if start >= 0 && end > start {
				recipient = line[start+1 : end]
			}
			capture.mu.Lock()
			capture.recipients = append(capture.recipients, recipient)
			capture.mu.Unlock()
			if options.rejectRecipient != "" && strings.EqualFold(recipient, options.rejectRecipient) {
				write("550 recipient rejected")
				continue
			}
			write("250 OK")
		case upper == "DATA":
			capture.mu.Lock()
			capture.dataCount++
			capture.transactions++
			capture.mu.Unlock()
			if options.failDATA {
				write("554 DATA failed")
				continue
			}
			write("354 End data with <CR><LF>.<CR><LF>")
			var data bytes.Buffer
			for {
				raw, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if raw == ".\r\n" || raw == ".\n" {
					break
				}
				if strings.HasPrefix(raw, "..") {
					raw = raw[1:]
				}
				data.WriteString(raw)
			}
			capture.mu.Lock()
			capture.data = append([]byte(nil), data.Bytes()...)
			capture.mu.Unlock()
			write("250 OK")
		case upper == "RSET":
			write("250 OK")
		case upper == "QUIT":
			write("221 Bye")
			return
		default:
			write("250 OK")
		}
	}
}

func mustRaiseSMTP(t *testing.T, body func()) string {
	t.Helper()
	var message string
	func() {
		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatal("expected SMTPError")
			}
			signal, ok := recovered.(*AhdSignal)
			if !ok || signal.Instance.AhdClassOf() != AhdClassSMTPError {
				t.Fatalf("expected SMTPError; received %v", recovered)
			}
			message = signal.Message
		}()
		body()
	}()
	return message
}

func smtpTextMessage() string {
	return AhdSMTPMessageWithText(AhdClassSMTPError,
		AhdSMTPMessage(AhdClassSMTPError, "sender@example.com", []string{"student@example.com"}, "AhdCode Test"),
		"Merhaba AhdCode")
}

func TestSMTPConfigValidation(t *testing.T) {
	class := AhdClassSMTPError
	if msg := mustRaiseSMTP(t, func() { AhdSMTPClient(class, "", 25, "none", 1) }); !strings.Contains(msg, "host") {
		t.Fatalf("empty host: %s", msg)
	}
	if msg := mustRaiseSMTP(t, func() { AhdSMTPClient(class, "smtp://localhost", 25, "none", 1) }); !strings.Contains(msg, "URL") {
		t.Fatalf("URL host: %s", msg)
	}
	if msg := mustRaiseSMTP(t, func() { AhdSMTPClient(class, "127.0.0.1", 0, "none", 1) }); !strings.Contains(msg, "port") {
		t.Fatalf("port 0: %s", msg)
	}
	if msg := mustRaiseSMTP(t, func() { AhdSMTPClient(class, "127.0.0.1", 65536, "none", 1) }); !strings.Contains(msg, "port") {
		t.Fatalf("port 65536: %s", msg)
	}
	if AhdSMTPClient(class, "127.0.0.1", 1, "none", 1) == "" {
		t.Fatal("port 1 must be accepted")
	}
	if AhdSMTPClient(class, "127.0.0.1", 65535, "none", 1) == "" {
		t.Fatal("port 65535 must be accepted")
	}
	if msg := mustRaiseSMTP(t, func() { AhdSMTPClient(class, "127.0.0.1", 25, "none", 0) }); !strings.Contains(msg, "timeoutSeconds") {
		t.Fatalf("timeout 0: %s", msg)
	}
	if msg := mustRaiseSMTP(t, func() { AhdSMTPClient(class, "127.0.0.1", 25, "none", ahdSMTPMaxTimeoutSeconds+1) }); !strings.Contains(msg, "timeoutSeconds") {
		t.Fatalf("timeout overflow: %s", msg)
	}
	if AhdSMTPClient(class, "127.0.0.1", 25, "none", 1) == "" {
		t.Fatal("timeout 1 must be accepted")
	}
	if AhdSMTPClient(class, "127.0.0.1", 25, "none", ahdSMTPMaxTimeoutSeconds) == "" {
		t.Fatal("timeout 9223372036 must be accepted")
	}
	for _, security := range []string{"STARTTLS", "ssl", "auto", "bad"} {
		if msg := mustRaiseSMTP(t, func() { AhdSMTPClient(class, "127.0.0.1", 25, security, 1) }); !strings.Contains(strings.ToLower(msg), "security") {
			t.Fatalf("security %q: %s", security, msg)
		}
	}
}

func TestSMTPClientAndMessageImmutability(t *testing.T) {
	class := AhdClassSMTPError
	base := AhdSMTPClient(class, "127.0.0.1", 2525, "none", 5)
	authenticated := AhdSMTPClientWithPlainAuth(class, base, "qa-user", smtpQAPassword)
	if authenticated == base {
		t.Fatal("withPlainAuth must return a new client handle")
	}
	baseConfig := ahdSMTPLookupClient(class, base)
	authConfig := ahdSMTPLookupClient(class, authenticated)
	if baseConfig.HasAuth {
		t.Fatal("base client must remain unauthenticated")
	}
	if !authConfig.HasAuth || authConfig.Username != "qa-user" {
		t.Fatal("authenticated client must carry AUTH PLAIN configuration")
	}

	m0 := AhdSMTPMessage(class, "sender@example.com", []string{"student@example.com"}, "Hello")
	m1 := AhdSMTPMessageWithCc(class, m0, []string{"cc@example.com"})
	m2 := AhdSMTPMessageWithText(class, m1, "Merhaba")
	m3 := AhdSMTPMessageWithHtml(class, m2, "<strong>Merhaba</strong>")
	if ahdSMTPDecodeMessage(class, m0).Cc != nil || ahdSMTPDecodeMessage(class, m0).HasText {
		t.Fatal("m0 must stay without Cc or body")
	}
	if len(ahdSMTPDecodeMessage(class, m1).Cc) != 1 || ahdSMTPDecodeMessage(class, m1).HasText {
		t.Fatal("m1 must keep Cc and no body")
	}
	if !ahdSMTPDecodeMessage(class, m2).HasText || ahdSMTPDecodeMessage(class, m2).HasHTML {
		t.Fatal("m2 must keep text only")
	}
	if !ahdSMTPDecodeMessage(class, m3).HasText || !ahdSMTPDecodeMessage(class, m3).HasHTML {
		t.Fatal("m3 must keep both bodies")
	}
}

func TestSMTPPlaintextSend(t *testing.T) {
	capture, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none"})
	client := AhdSMTPClient(AhdClassSMTPError, host, int64(port), "none", 5)
	AhdSMTPClientSend(AhdClassSMTPError, client, smtpTextMessage())
	capture.mu.Lock()
	defer capture.mu.Unlock()
	if capture.mailFrom != "sender@example.com" {
		t.Fatalf("MAIL FROM = %q", capture.mailFrom)
	}
	if len(capture.recipients) != 1 || capture.recipients[0] != "student@example.com" {
		t.Fatalf("RCPT = %#v", capture.recipients)
	}
	msg := smtpReadMessage(t, capture.data)
	if got := smtpDecodedSubject(t, msg); got != "AhdCode Test" {
		t.Fatalf("subject = %q", got)
	}
	if got := smtpDecodedBody(t, msg); got != "Merhaba AhdCode" {
		t.Fatalf("body = %q", got)
	}
	if capture.dataCount != 1 {
		t.Fatalf("DATA count = %d", capture.dataCount)
	}
}

func TestSMTPSTARTTLSAndMissingCapability(t *testing.T) {
	caFile, cert := smtpTestCert(t, []string{}, []net.IP{net.ParseIP("127.0.0.1")})
	t.Setenv("SSL_CERT_FILE", caFile)
	capture, host, port := startSMTPFixture(t, smtpFixtureOptions{
		security: "starttls", advertiseSTARTTLS: true, certificate: &cert,
	})
	client := AhdSMTPClient(AhdClassSMTPError, host, int64(port), "starttls", 5)
	AhdSMTPClientSend(AhdClassSMTPError, client, smtpTextMessage())
	capture.mu.Lock()
	if capture.dataCount != 1 {
		t.Fatalf("STARTTLS DATA count = %d", capture.dataCount)
	}
	capture.mu.Unlock()

	missing, host2, port2 := startSMTPFixture(t, smtpFixtureOptions{security: "none", advertiseSTARTTLS: false})
	client2 := AhdSMTPClient(AhdClassSMTPError, host2, int64(port2), "starttls", 5)
	msg := mustRaiseSMTP(t, func() { AhdSMTPClientSend(AhdClassSMTPError, client2, smtpTextMessage()) })
	if !strings.Contains(msg, "STARTTLS") {
		t.Fatalf("missing STARTTLS: %s", msg)
	}
	missing.mu.Lock()
	defer missing.mu.Unlock()
	if missing.dataCount != 0 {
		t.Fatalf("plaintext DATA fallback occurred: %d", missing.dataCount)
	}
}

func TestSMTPImplicitTLS(t *testing.T) {
	caFile, cert := smtpTestCert(t, []string{}, []net.IP{net.ParseIP("127.0.0.1")})
	t.Setenv("SSL_CERT_FILE", caFile)
	capture, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "tls", certificate: &cert})
	client := AhdSMTPClient(AhdClassSMTPError, host, int64(port), "tls", 5)
	AhdSMTPClientSend(AhdClassSMTPError, client, smtpTextMessage())
	capture.mu.Lock()
	defer capture.mu.Unlock()
	if capture.dataCount != 1 {
		t.Fatalf("implicit TLS DATA count = %d", capture.dataCount)
	}
}

func TestSMTPUntrustedAndHostnameTLS(t *testing.T) {
	_, cert := smtpTestCert(t, []string{"smtp.example.test"}, nil)
	untrusted, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "tls", certificate: &cert})
	client := AhdSMTPClient(AhdClassSMTPError, host, int64(port), "tls", 5)
	msg := mustRaiseSMTP(t, func() { AhdSMTPClientSend(AhdClassSMTPError, client, smtpTextMessage()) })
	if !strings.Contains(msg, "TLS") {
		t.Fatalf("untrusted TLS: %s", msg)
	}
	untrusted.mu.Lock()
	if untrusted.dataCount != 0 {
		t.Fatal("untrusted TLS accepted DATA")
	}
	untrusted.mu.Unlock()

	caFile, mismatch := smtpTestCert(t, []string{"smtp.example.test"}, nil)
	t.Setenv("SSL_CERT_FILE", caFile)
	capture, host2, port2 := startSMTPFixture(t, smtpFixtureOptions{security: "tls", certificate: &mismatch})
	client2 := AhdSMTPClient(AhdClassSMTPError, host2, int64(port2), "tls", 5)
	msg = mustRaiseSMTP(t, func() { AhdSMTPClientSend(AhdClassSMTPError, client2, smtpTextMessage()) })
	if !strings.Contains(msg, "TLS") {
		t.Fatalf("hostname mismatch: %s", msg)
	}
	capture.mu.Lock()
	defer capture.mu.Unlock()
	if capture.dataCount != 0 {
		t.Fatal("hostname mismatch accepted DATA")
	}
}

func TestSMTPAUTHPlain(t *testing.T) {
	caFile, cert := smtpTestCert(t, []string{}, []net.IP{net.ParseIP("127.0.0.1")})
	t.Setenv("SSL_CERT_FILE", caFile)
	okCap, host, port := startSMTPFixture(t, smtpFixtureOptions{
		security: "tls", certificate: &cert, advertiseAuthPlain: true,
		authUser: "qa-user", authPass: smtpQAPassword,
	})
	client := AhdSMTPClientWithPlainAuth(AhdClassSMTPError,
		AhdSMTPClient(AhdClassSMTPError, host, int64(port), "tls", 5),
		"qa-user", smtpQAPassword)
	AhdSMTPClientSend(AhdClassSMTPError, client, smtpTextMessage())
	okCap.mu.Lock()
	if okCap.dataCount != 1 || !okCap.authSeen {
		t.Fatalf("AUTH success DATA=%d auth=%v", okCap.dataCount, okCap.authSeen)
	}
	okCap.mu.Unlock()

	bad, host2, port2 := startSMTPFixture(t, smtpFixtureOptions{
		security: "tls", certificate: &cert, advertiseAuthPlain: true,
		authUser: "qa-user", authPass: smtpQAPassword,
	})
	wrong := AhdSMTPClientWithPlainAuth(AhdClassSMTPError,
		AhdSMTPClient(AhdClassSMTPError, host2, int64(port2), "tls", 5),
		"qa-user", "wrong-password")
	msg := mustRaiseSMTP(t, func() { AhdSMTPClientSend(AhdClassSMTPError, wrong, smtpTextMessage()) })
	if !strings.Contains(msg, "authentication") {
		t.Fatalf("wrong AUTH: %s", msg)
	}
	if strings.Contains(msg, smtpQAPassword) || strings.Contains(msg, "wrong-password") {
		t.Fatalf("AUTH error leaked a secret: %s", msg)
	}
	bad.mu.Lock()
	if bad.dataCount != 0 {
		t.Fatal("failed AUTH continued to DATA")
	}
	bad.mu.Unlock()

	missing, host3, port3 := startSMTPFixture(t, smtpFixtureOptions{
		security: "tls", certificate: &cert, advertiseAuthPlain: false,
	})
	noMech := AhdSMTPClientWithPlainAuth(AhdClassSMTPError,
		AhdSMTPClient(AhdClassSMTPError, host3, int64(port3), "tls", 5),
		"qa-user", smtpQAPassword)
	msg = mustRaiseSMTP(t, func() { AhdSMTPClientSend(AhdClassSMTPError, noMech, smtpTextMessage()) })
	if !strings.Contains(msg, "authentication") {
		t.Fatalf("missing AUTH PLAIN: %s", msg)
	}
	missing.mu.Lock()
	defer missing.mu.Unlock()
	if missing.authSeen || missing.dataCount != 0 {
		t.Fatalf("AUTH fallback occurred: auth=%v data=%d", missing.authSeen, missing.dataCount)
	}
}

func TestSMTPPlaintextAuthRejected(t *testing.T) {
	capture, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none", advertiseAuthPlain: true, authUser: "qa-user", authPass: smtpQAPassword})
	client := AhdSMTPClientWithPlainAuth(AhdClassSMTPError,
		AhdSMTPClient(AhdClassSMTPError, host, int64(port), "none", 5),
		"qa-user", smtpQAPassword)
	msg := mustRaiseSMTP(t, func() { AhdSMTPClientSend(AhdClassSMTPError, client, smtpTextMessage()) })
	if !strings.Contains(msg, "encrypted") {
		t.Fatalf("plaintext AUTH: %s", msg)
	}
	capture.mu.Lock()
	defer capture.mu.Unlock()
	if capture.authSeen {
		t.Fatal("credentials were transmitted over plaintext SMTP")
	}
	if capture.dataCount != 0 {
		t.Fatal("plaintext AUTH sent DATA")
	}
}

func TestSMTPSecretSafety(t *testing.T) {
	_, cert := smtpTestCert(t, []string{"smtp.example.test"}, nil)
	capture, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "tls", certificate: &cert})
	client := AhdSMTPClientWithPlainAuth(AhdClassSMTPError,
		AhdSMTPClient(AhdClassSMTPError, host, int64(port), "tls", 5),
		"qa-user", smtpQAPassword)
	tlsMsg := mustRaiseSMTP(t, func() { AhdSMTPClientSend(AhdClassSMTPError, client, smtpTextMessage()) })
	caFile, good := smtpTestCert(t, []string{}, []net.IP{net.ParseIP("127.0.0.1")})
	t.Setenv("SSL_CERT_FILE", caFile)
	authCap, host2, port2 := startSMTPFixture(t, smtpFixtureOptions{
		security: "tls", certificate: &good, advertiseAuthPlain: true, authUser: "qa-user", authPass: "other",
	})
	authClient := AhdSMTPClientWithPlainAuth(AhdClassSMTPError,
		AhdSMTPClient(AhdClassSMTPError, host2, int64(port2), "tls", 5),
		"qa-user", smtpQAPassword)
	authMsg := mustRaiseSMTP(t, func() { AhdSMTPClientSend(AhdClassSMTPError, authClient, smtpTextMessage()) })
	stall, host3, port3 := startSMTPFixture(t, smtpFixtureOptions{security: "none", stall: 3 * time.Second})
	slow := AhdSMTPClient(AhdClassSMTPError, host3, int64(port3), "none", 1)
	timeoutMsg := mustRaiseSMTP(t, func() { AhdSMTPClientSend(AhdClassSMTPError, slow, smtpTextMessage()) })
	for _, msg := range []string{tlsMsg, authMsg, timeoutMsg} {
		if strings.Contains(msg, smtpQAPassword) {
			t.Fatalf("secret leaked in %q", msg)
		}
	}
	_ = capture
	_ = authCap
	_ = stall
}

func TestSMTPEnvelopeAndBccPrivacy(t *testing.T) {
	capture, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none"})
	client := AhdSMTPClient(AhdClassSMTPError, host, int64(port), "none", 5)
	message := AhdSMTPMessage(AhdClassSMTPError, "sender@example.com", []string{"alice@example.com"}, "Seminer")
	message = AhdSMTPMessageWithCc(AhdClassSMTPError, message, []string{"bob@example.com"})
	message = AhdSMTPMessageWithBcc(AhdClassSMTPError, message, []string{"secret@example.com"})
	message = AhdSMTPMessageWithText(AhdClassSMTPError, message, "Merhaba")
	AhdSMTPClientSend(AhdClassSMTPError, client, message)
	capture.mu.Lock()
	defer capture.mu.Unlock()
	want := []string{"alice@example.com", "bob@example.com", "secret@example.com"}
	if strings.Join(capture.recipients, ",") != strings.Join(want, ",") {
		t.Fatalf("envelope = %#v", capture.recipients)
	}
	raw := string(capture.data)
	if !strings.Contains(raw, "To:") || !strings.Contains(raw, "Cc:") {
		t.Fatalf("To/Cc missing from DATA:\n%s", raw)
	}
	if strings.Contains(strings.ToLower(raw), "bcc:") || strings.Contains(raw, "secret@example.com") {
		t.Fatalf("Bcc leaked into DATA:\n%s", raw)
	}
}

func TestSMTPUTF8AndMultipart(t *testing.T) {
	capture, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none"})
	client := AhdSMTPClient(AhdClassSMTPError, host, int64(port), "none", 5)
	subject := "Matematik ve Kodlama — Çukurova"
	text := "Merhaba, bugün hava güzel. 😊"
	html := "<p>Merhaba <strong>Hatay</strong> — ÇĞİÖŞÜ 😊</p>"
	message := AhdSMTPMessage(AhdClassSMTPError, "sender@example.com", []string{"student@example.com"}, subject)
	message = AhdSMTPMessageWithText(AhdClassSMTPError, message, text)
	message = AhdSMTPMessageWithHtml(AhdClassSMTPError, message, html)
	AhdSMTPClientSend(AhdClassSMTPError, client, message)
	msg := smtpReadMessage(t, capture.data)
	if got := smtpDecodedSubject(t, msg); got != subject {
		t.Fatalf("subject = %q", got)
	}
	plain, rich, media := smtpDecodedParts(t, msg)
	if media != "multipart/alternative" {
		t.Fatalf("media = %q", media)
	}
	if plain != text || rich != html {
		t.Fatalf("parts text=%q html=%q", plain, rich)
	}
}

func TestSMTPHeaderInjectionAndAddresses(t *testing.T) {
	class := AhdClassSMTPError
	mustRaiseSMTP(t, func() {
		AhdSMTPMessage(class, "sender@example.com", []string{"student@example.com"}, "Hello\r\nBcc: injected@example.com")
	})
	mustRaiseSMTP(t, func() {
		AhdSMTPMessage(class, "sender@example.com\r\nBcc: injected@example.com", []string{"student@example.com"}, "Hello")
	})
	mustRaiseSMTP(t, func() {
		AhdSMTPMessage(class, "sender@example.com", []string{"a@example.com, b@example.com"}, "Hello")
	})
	mustRaiseSMTP(t, func() {
		AhdSMTPMessage(class, "sender@example.com", []string{"öğrenci@example.com"}, "Hello")
	})
	mustRaiseSMTP(t, func() {
		AhdSMTPMessageWithReplyTo(class, AhdSMTPMessage(class, "sender@example.com", []string{"student@example.com"}, "Hello"), "ok@example.com\nBcc: x@example.com")
	})
	named := AhdSMTPMessage(class, "Ali Daldallı <sender@example.com>", []string{"Student <student@example.com>"}, "Hello")
	if ahdSMTPDecodeMessage(class, named).From == "" {
		t.Fatal("display-name addresses must be accepted")
	}
}

func TestSMTPRecipientRejectionAndRetry(t *testing.T) {
	reject, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none", rejectRecipient: "bob@example.com"})
	client := AhdSMTPClient(AhdClassSMTPError, host, int64(port), "none", 5)
	message := AhdSMTPMessage(AhdClassSMTPError, "sender@example.com", []string{"alice@example.com", "bob@example.com"}, "Hello")
	message = AhdSMTPMessageWithText(AhdClassSMTPError, message, "hi")
	mustRaiseSMTP(t, func() { AhdSMTPClientSend(AhdClassSMTPError, client, message) })
	reject.mu.Lock()
	if reject.dataCount != 0 {
		t.Fatalf("DATA after rejected RCPT = %d", reject.dataCount)
	}
	reject.mu.Unlock()

	fail, host2, port2 := startSMTPFixture(t, smtpFixtureOptions{security: "none", failDATA: true})
	failClient := AhdSMTPClient(AhdClassSMTPError, host2, int64(port2), "none", 5)
	mustRaiseSMTP(t, func() { AhdSMTPClientSend(AhdClassSMTPError, failClient, smtpTextMessage()) })
	fail.mu.Lock()
	if fail.dataCount != 1 || fail.transactions != 1 {
		t.Fatalf("failed send retries: data=%d tx=%d", fail.dataCount, fail.transactions)
	}
	fail.mu.Unlock()

	ok, host3, port3 := startSMTPFixture(t, smtpFixtureOptions{security: "none"})
	okClient := AhdSMTPClient(AhdClassSMTPError, host3, int64(port3), "none", 5)
	AhdSMTPClientSend(AhdClassSMTPError, okClient, smtpTextMessage())
	ok.mu.Lock()
	defer ok.mu.Unlock()
	if ok.dataCount != 1 {
		t.Fatalf("successful send DATA = %d", ok.dataCount)
	}
}

func TestSMTPTimeoutThenHealthySend(t *testing.T) {
	_, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none", stall: 3 * time.Second})
	client := AhdSMTPClient(AhdClassSMTPError, host, int64(port), "none", 1)
	started := time.Now()
	msg := mustRaiseSMTP(t, func() { AhdSMTPClientSend(AhdClassSMTPError, client, smtpTextMessage()) })
	elapsed := time.Since(started)
	if !strings.Contains(strings.ToLower(msg), "timed out") {
		t.Fatalf("timeout message: %s", msg)
	}
	if elapsed < 800*time.Millisecond || elapsed > 4*time.Second {
		t.Fatalf("timeout elapsed %s", elapsed)
	}
	healthy, host2, port2 := startSMTPFixture(t, smtpFixtureOptions{security: "none"})
	AhdSMTPClientSend(AhdClassSMTPError, AhdSMTPClient(AhdClassSMTPError, host2, int64(port2), "none", 5), smtpTextMessage())
	healthy.mu.Lock()
	defer healthy.mu.Unlock()
	if healthy.dataCount != 1 {
		t.Fatal("client unusable after timeout")
	}
}

func TestSMTPDotStuffing(t *testing.T) {
	capture, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none"})
	body := ".first\n..\n.normal\nline"
	message := AhdSMTPMessageWithText(AhdClassSMTPError,
		AhdSMTPMessage(AhdClassSMTPError, "sender@example.com", []string{"student@example.com"}, "Dots"),
		body)
	AhdSMTPClientSend(AhdClassSMTPError, AhdSMTPClient(AhdClassSMTPError, host, int64(port), "none", 5), message)
	msg := smtpReadMessage(t, capture.data)
	if got := smtpDecodedBody(t, msg); got != body {
		t.Fatalf("dot-stuffed body = %q", got)
	}
}

func TestSMTPConcurrentSends(t *testing.T) {
	capture, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none"})
	client := AhdSMTPClient(AhdClassSMTPError, host, int64(port), "none", 5)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			message := AhdSMTPMessageWithText(AhdClassSMTPError,
				AhdSMTPMessage(AhdClassSMTPError, "sender@example.com", []string{"user" + string(rune('a'+i)) + "@example.com"}, "N"+string(rune('1'+i))),
				"body-"+string(rune('1'+i)))
			AhdSMTPClientSend(AhdClassSMTPError, client, message)
		}(i)
	}
	wg.Wait()
	capture.mu.Lock()
	defer capture.mu.Unlock()
	if capture.dataCount != 4 || capture.connections != 4 {
		t.Fatalf("concurrent data=%d connections=%d", capture.dataCount, capture.connections)
	}
}

func smtpReadMessage(t *testing.T, data []byte) *mail.Message {
	t.Helper()
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("read message: %v\n%s", err, data)
	}
	return msg
}

func smtpDecodedSubject(t *testing.T, msg *mail.Message) string {
	t.Helper()
	decoder := new(mime.WordDecoder)
	got, err := decoder.DecodeHeader(msg.Header.Get("Subject"))
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func smtpDecodedBody(t *testing.T, msg *mail.Message) string {
	t.Helper()
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		t.Fatal(err)
	}
	media, params, _ := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if strings.HasPrefix(media, "text/") && msg.Header.Get("Content-Transfer-Encoding") == "quoted-printable" {
		decoded, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimRight(strings.ReplaceAll(string(decoded), "\r\n", "\n"), "\n")
	}
	_ = params
	return strings.TrimRight(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n")
}

func smtpDecodedParts(t *testing.T, msg *mail.Message) (text, html, media string) {
	t.Helper()
	media, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	if media != "multipart/alternative" {
		return smtpDecodedBody(t, msg), "", media
	}
	reader := multipart.NewReader(msg.Body, params["boundary"])
	part, err := reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	text = smtpDecodePart(t, part)
	part, err = reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	html = smtpDecodePart(t, part)
	if _, err := reader.NextPart(); err != io.EOF {
		t.Fatal("unexpected extra MIME part")
	}
	return text, html, media
}

func smtpDecodePart(t *testing.T, part *multipart.Part) string {
	t.Helper()
	body, err := io.ReadAll(part)
	if err != nil {
		t.Fatal(err)
	}
	if strings.EqualFold(part.Header.Get("Content-Transfer-Encoding"), "quoted-printable") {
		decoded, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		if err != nil {
			t.Fatal(err)
		}
		body = decoded
	}
	return strings.TrimRight(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n")
}

func smtpTestCert(t *testing.T, dns []string, ips []net.IP) (caFile string, certificate tls.Certificate) {
	t.Helper()
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "AhdCode SMTP Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "smtp.test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     dns,
		IPAddresses:  ips,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(leafKey)})
	certificate, err = tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	file, err := os.CreateTemp(t.TempDir(), "smtp-ca-*.pem")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(caPEM); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return file.Name(), certificate
}
