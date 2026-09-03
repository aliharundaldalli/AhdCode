package ahdruntime

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
)

const (
	// ahdSMTPMaxTimeoutSeconds is the largest whole-second timeout that still
	// fits in a time.Duration after conversion to nanoseconds.
	ahdSMTPMaxTimeoutSeconds = 9223372036
)

type ahdSMTPClientConfig struct {
	Host           string
	Port           int64
	Security       string
	TimeoutSeconds int64
	Username       string
	Password       string
	HasAuth        bool
}

type ahdSMTPMessageData struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Cc      []string `json:"cc,omitempty"`
	Bcc     []string `json:"bcc,omitempty"`
	ReplyTo string   `json:"replyTo,omitempty"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
	HTML    string   `json:"html"`
	HasText bool     `json:"hasText"`
	HasHTML bool     `json:"hasHTML"`
}

type ahdSMTPMailbox struct {
	raw     string
	header  string
	address string
}

type ahdSMTPPlainAuth struct {
	username string
	password string
}

var (
	ahdSMTPClients   = map[string]ahdSMTPClientConfig{}
	ahdSMTPClientsMu sync.Mutex
	ahdSMTPNextID    atomic.Int64
)

func (auth ahdSMTPPlainAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	_ = server
	return "PLAIN", []byte("\x00" + auth.username + "\x00" + auth.password), nil
}

func (auth ahdSMTPPlainAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		return nil, errors.New("unexpected server challenge")
	}
	return nil, nil
}

func AhdSMTPClient(class *AhdClass, host string, port int64, security string, timeoutSeconds int64) string {
	ahdSMTPValidateClient(class, host, port, security, timeoutSeconds)
	id := strconv.FormatInt(ahdSMTPNextID.Add(1), 10)
	ahdSMTPClientsMu.Lock()
	ahdSMTPClients[id] = ahdSMTPClientConfig{
		Host: host, Port: port, Security: security, TimeoutSeconds: timeoutSeconds,
	}
	ahdSMTPClientsMu.Unlock()
	return id
}

func AhdSMTPClientWithPlainAuth(class *AhdClass, handle, username, password string) string {
	config := ahdSMTPLookupClient(class, handle)
	config.Username = username
	config.Password = password
	config.HasAuth = true
	id := strconv.FormatInt(ahdSMTPNextID.Add(1), 10)
	ahdSMTPClientsMu.Lock()
	ahdSMTPClients[id] = config
	ahdSMTPClientsMu.Unlock()
	return id
}

func AhdSMTPMessage(class *AhdClass, from string, to []string, subject string) string {
	message := ahdSMTPMessageData{From: from, To: ahdSMTPCopyStrings(to), Subject: subject}
	ahdSMTPValidateMessageFields(class, message)
	return ahdSMTPEncodeMessage(class, message)
}

func AhdSMTPMessageWithCc(class *AhdClass, data string, recipients []string) string {
	message := ahdSMTPDecodeMessage(class, data)
	message.Cc = ahdSMTPCopyStrings(recipients)
	ahdSMTPValidateMessageFields(class, message)
	return ahdSMTPEncodeMessage(class, message)
}

func AhdSMTPMessageWithBcc(class *AhdClass, data string, recipients []string) string {
	message := ahdSMTPDecodeMessage(class, data)
	message.Bcc = ahdSMTPCopyStrings(recipients)
	ahdSMTPValidateMessageFields(class, message)
	return ahdSMTPEncodeMessage(class, message)
}

func AhdSMTPMessageWithReplyTo(class *AhdClass, data, address string) string {
	message := ahdSMTPDecodeMessage(class, data)
	message.ReplyTo = address
	ahdSMTPValidateMessageFields(class, message)
	return ahdSMTPEncodeMessage(class, message)
}

func AhdSMTPMessageWithText(class *AhdClass, data, body string) string {
	message := ahdSMTPDecodeMessage(class, data)
	message.Text = body
	message.HasText = true
	return ahdSMTPEncodeMessage(class, message)
}

func AhdSMTPMessageWithHtml(class *AhdClass, data, body string) string {
	message := ahdSMTPDecodeMessage(class, data)
	message.HTML = body
	message.HasHTML = true
	return ahdSMTPEncodeMessage(class, message)
}

func AhdSMTPClientSend(class *AhdClass, handle, messageData string) {
	config := ahdSMTPLookupClient(class, handle)
	message := ahdSMTPDecodeMessage(class, messageData)
	ahdSMTPValidateClient(class, config.Host, config.Port, config.Security, config.TimeoutSeconds)
	ahdSMTPValidateMessageFields(class, message)
	if !message.HasText && !message.HasHTML {
		ahdSMTPRaise(class, "SMTP message has no body", config.Password)
	}
	parsed, envelope := ahdSMTPMaterialize(class, message)
	if len(envelope) == 0 {
		ahdSMTPRaise(class, "SMTP message has no recipients", config.Password)
	}
	ahdSMTPDeliver(class, config, parsed, envelope)
}

func ahdSMTPValidateClient(class *AhdClass, host string, port int64, security string, timeoutSeconds int64) {
	if err := ahdSMTPHostError(host); err != "" {
		ahdSMTPRaise(class, err, "")
	}
	if port < 1 || port > 65535 {
		ahdSMTPRaise(class, "SMTP port must be in 1..65535", "")
	}
	switch security {
	case "starttls", "tls", "none":
	default:
		ahdSMTPRaise(class, "SMTP security must be starttls, tls, or none", "")
	}
	if timeoutSeconds < 1 || timeoutSeconds > ahdSMTPMaxTimeoutSeconds {
		ahdSMTPRaise(class, "SMTP timeoutSeconds must be between 1 and 9223372036", "")
	}
}

func ahdSMTPHostError(host string) string {
	if host == "" {
		return "SMTP host must not be empty"
	}
	if strings.Contains(host, "://") {
		return "SMTP host must not be a URL"
	}
	if strings.TrimSpace(host) != host {
		return "SMTP host is not valid"
	}
	for _, r := range host {
		if r < 32 || r == 127 || r == '/' || unicode.IsSpace(r) {
			return "SMTP host is not valid"
		}
	}
	return ""
}

func ahdSMTPValidateMessageFields(class *AhdClass, message ahdSMTPMessageData) {
	ahdSMTPParseMailbox(class, message.From, "From")
	if ahdSMTPHasCRLF(message.Subject) {
		ahdSMTPRaise(class, "SMTP subject must not contain a line break", "")
	}
	for _, raw := range message.To {
		ahdSMTPParseMailbox(class, raw, "To")
	}
	for _, raw := range message.Cc {
		ahdSMTPParseMailbox(class, raw, "Cc")
	}
	for _, raw := range message.Bcc {
		ahdSMTPParseMailbox(class, raw, "Bcc")
	}
	if message.ReplyTo != "" {
		ahdSMTPParseMailbox(class, message.ReplyTo, "Reply-To")
	}
}

func ahdSMTPMaterialize(class *AhdClass, message ahdSMTPMessageData) (ahdSMTPMessageParsed, []string) {
	parsed := ahdSMTPMessageParsed{
		from:    ahdSMTPParseMailbox(class, message.From, "From"),
		subject: message.Subject,
		text:    message.Text,
		html:    message.HTML,
		hasText: message.HasText,
		hasHTML: message.HasHTML,
	}
	var envelope []string
	for _, raw := range message.To {
		box := ahdSMTPParseMailbox(class, raw, "To")
		parsed.to = append(parsed.to, box)
		envelope = append(envelope, box.address)
	}
	for _, raw := range message.Cc {
		box := ahdSMTPParseMailbox(class, raw, "Cc")
		parsed.cc = append(parsed.cc, box)
		envelope = append(envelope, box.address)
	}
	for _, raw := range message.Bcc {
		box := ahdSMTPParseMailbox(class, raw, "Bcc")
		parsed.bcc = append(parsed.bcc, box)
		envelope = append(envelope, box.address)
	}
	if message.ReplyTo != "" {
		box := ahdSMTPParseMailbox(class, message.ReplyTo, "Reply-To")
		parsed.replyTo = &box
	}
	return parsed, envelope
}

type ahdSMTPMessageParsed struct {
	from    ahdSMTPMailbox
	to      []ahdSMTPMailbox
	cc      []ahdSMTPMailbox
	bcc     []ahdSMTPMailbox
	replyTo *ahdSMTPMailbox
	subject string
	text    string
	html    string
	hasText bool
	hasHTML bool
}

func ahdSMTPParseMailbox(class *AhdClass, raw, field string) ahdSMTPMailbox {
	if strings.TrimSpace(raw) == "" {
		ahdSMTPRaise(class, "SMTP "+field+" mailbox must not be empty", "")
	}
	if ahdSMTPHasCRLF(raw) {
		ahdSMTPRaise(class, "SMTP "+field+" must not contain a line break", "")
	}
	parsed, err := mail.ParseAddress(raw)
	if err != nil {
		ahdSMTPRaise(class, "SMTP "+field+" mailbox is not valid", "")
	}
	if parsed == nil || parsed.Address == "" {
		ahdSMTPRaise(class, "SMTP "+field+" mailbox is not valid", "")
	}
	if !ahdSMTPASCII(parsed.Address) {
		ahdSMTPRaise(class, "SMTP "+field+" mailbox must be ASCII; SMTPUTF8 is not supported", "")
	}
	return ahdSMTPMailbox{raw: raw, header: parsed.String(), address: parsed.Address}
}

func ahdSMTPHasCRLF(value string) bool {
	return strings.ContainsAny(value, "\r\n")
}

func ahdSMTPASCII(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] > 127 {
			return false
		}
	}
	return true
}

func ahdSMTPCopyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}

func ahdSMTPLookupClient(class *AhdClass, handle string) ahdSMTPClientConfig {
	ahdSMTPClientsMu.Lock()
	config, ok := ahdSMTPClients[handle]
	ahdSMTPClientsMu.Unlock()
	if !ok {
		ahdSMTPRaise(class, "SMTP client storage is corrupted", "")
	}
	return config
}

func ahdSMTPEncodeMessage(class *AhdClass, message ahdSMTPMessageData) string {
	encoded, err := json.Marshal(message)
	if err != nil {
		ahdSMTPRaise(class, "SMTP message storage is corrupted", "")
	}
	return string(encoded)
}

func ahdSMTPDecodeMessage(class *AhdClass, data string) ahdSMTPMessageData {
	var message ahdSMTPMessageData
	if err := json.Unmarshal([]byte(data), &message); err != nil {
		ahdSMTPRaise(class, "SMTP message storage is corrupted", "")
	}
	return message
}

func ahdSMTPDeliver(class *AhdClass, config ahdSMTPClientConfig, parsed ahdSMTPMessageParsed, envelope []string) {
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	deadline := time.Now().Add(timeout)
	addr := net.JoinHostPort(config.Host, strconv.FormatInt(config.Port, 10))
	dialer := &net.Dialer{Timeout: timeout}
	var conn net.Conn
	var err error
	if config.Security == "tls" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, ahdSMTPTlsConfig(class, config.Host, config.Password))
		if err != nil {
			ahdSMTPRaiseMapped(class, err, "tls", config.Password)
		}
	} else {
		conn, err = dialer.Dial("tcp", addr)
		if err != nil {
			ahdSMTPRaiseMapped(class, err, "connect", config.Password)
		}
	}
	defer conn.Close()
	if err := conn.SetDeadline(deadline); err != nil {
		ahdSMTPRaiseMapped(class, err, "connect", config.Password)
	}

	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		ahdSMTPRaiseMapped(class, err, "greeting", config.Password)
	}
	defer func() {
		_ = client.Close()
	}()

	if config.Security == "starttls" {
		ok, _ := client.Extension("STARTTLS")
		if !ok {
			_ = client.Quit()
			ahdSMTPRaise(class, "SMTP STARTTLS is required but not supported", config.Password)
		}
		if err := client.StartTLS(ahdSMTPTlsConfig(class, config.Host, config.Password)); err != nil {
			ahdSMTPRaiseMapped(class, err, "tls", config.Password)
		}
	}

	if config.HasAuth {
		if config.Security == "none" {
			_ = client.Quit()
			ahdSMTPRaise(class, "SMTP authentication requires an encrypted connection", config.Password)
		}
		ok, params := client.Extension("AUTH")
		if !ok || !ahdSMTPHasMechanism(params, "PLAIN") {
			_ = client.Quit()
			ahdSMTPRaise(class, "SMTP authentication failed", config.Password)
		}
		auth := ahdSMTPPlainAuth{username: config.Username, password: config.Password}
		if err := client.Auth(auth); err != nil {
			ahdSMTPRaiseMapped(class, err, "auth", config.Password)
		}
	}

	if err := client.Mail(parsed.from.address); err != nil {
		ahdSMTPRaiseMapped(class, err, "mail", config.Password)
	}
	for _, recipient := range envelope {
		if err := client.Rcpt(recipient); err != nil {
			_ = client.Reset()
			ahdSMTPRaiseMapped(class, err, "rcpt", config.Password)
		}
	}
	writer, err := client.Data()
	if err != nil {
		ahdSMTPRaiseMapped(class, err, "data", config.Password)
	}
	payload := ahdSMTPBuildMIME(parsed)
	if _, err := writer.Write(payload); err != nil {
		_ = writer.Close()
		ahdSMTPRaiseMapped(class, err, "data", config.Password)
	}
	if err := writer.Close(); err != nil {
		ahdSMTPRaiseMapped(class, err, "data", config.Password)
	}
	if err := client.Quit(); err != nil {
		ahdSMTPRaiseMapped(class, err, "quit", config.Password)
	}
}

func ahdSMTPHasMechanism(params, want string) bool {
	for _, mechanism := range strings.Fields(params) {
		if strings.EqualFold(mechanism, want) {
			return true
		}
	}
	return false
}

func ahdSMTPTlsConfig(class *AhdClass, host, password string) *tls.Config {
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		roots = x509.NewCertPool()
	}
	if path := strings.TrimSpace(os.Getenv("SSL_CERT_FILE")); path != "" {
		pem, readErr := os.ReadFile(path)
		if readErr != nil || !roots.AppendCertsFromPEM(pem) {
			ahdSMTPRaise(class, "SMTP TLS verification failed", password)
		}
	}
	serverName := host
	if strings.HasPrefix(serverName, "[") && strings.HasSuffix(serverName, "]") {
		serverName = strings.TrimSuffix(strings.TrimPrefix(serverName, "["), "]")
	}
	return &tls.Config{
		RootCAs:    roots,
		ServerName: serverName,
		MinVersion: tls.VersionTLS12,
	}
}

func ahdSMTPBuildMIME(parsed ahdSMTPMessageParsed) []byte {
	var buf bytes.Buffer
	writeHeader := func(name, value string) {
		buf.WriteString(name)
		buf.WriteString(": ")
		buf.WriteString(value)
		buf.WriteString("\r\n")
	}
	writeHeader("Date", time.Now().Format(time.RFC1123Z))
	writeHeader("From", parsed.from.header)
	if len(parsed.to) > 0 {
		writeHeader("To", ahdSMTPFormatList(parsed.to))
	}
	if len(parsed.cc) > 0 {
		writeHeader("Cc", ahdSMTPFormatList(parsed.cc))
	}
	if parsed.replyTo != nil {
		writeHeader("Reply-To", parsed.replyTo.header)
	}
	writeHeader("Subject", mime.QEncoding.Encode("utf-8", parsed.subject))
	writeHeader("MIME-Version", "1.0")

	switch {
	case parsed.hasText && parsed.hasHTML:
		var body bytes.Buffer
		mp := multipart.NewWriter(&body)
		ahdSMTPWritePart(mp, "text/plain; charset=utf-8", parsed.text)
		ahdSMTPWritePart(mp, "text/html; charset=utf-8", parsed.html)
		_ = mp.Close()
		writeHeader("Content-Type", "multipart/alternative; boundary="+mp.Boundary())
		buf.WriteString("\r\n")
		buf.Write(body.Bytes())
	case parsed.hasHTML:
		writeHeader("Content-Type", "text/html; charset=utf-8")
		writeHeader("Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		buf.Write(ahdSMTPQuotedPrintable(parsed.html))
	default:
		writeHeader("Content-Type", "text/plain; charset=utf-8")
		writeHeader("Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		buf.Write(ahdSMTPQuotedPrintable(parsed.text))
	}
	return buf.Bytes()
}

func ahdSMTPFormatList(boxes []ahdSMTPMailbox) string {
	parts := make([]string, len(boxes))
	for i, box := range boxes {
		parts[i] = box.header
	}
	return strings.Join(parts, ", ")
}

func ahdSMTPWritePart(mp *multipart.Writer, contentType, body string) {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", contentType)
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := mp.CreatePart(header)
	if err != nil {
		return
	}
	_, _ = part.Write(ahdSMTPQuotedPrintable(body))
}

func ahdSMTPQuotedPrintable(body string) []byte {
	var buf bytes.Buffer
	writer := quotedprintable.NewWriter(&buf)
	_, _ = writer.Write([]byte(ahdSMTPCRLF(body)))
	_ = writer.Close()
	return buf.Bytes()
}

func ahdSMTPCRLF(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.ReplaceAll(text, "\n", "\r\n")
}

func ahdSMTPRaiseMapped(class *AhdClass, err error, stage, password string) {
	if err == nil {
		return
	}
	message := ahdSMTPStageMessage(err, stage)
	ahdSMTPRaise(class, message, password)
}

func ahdSMTPStageMessage(err error, stage string) string {
	if ahdSMTPTimedOut(err) {
		return "SMTP request timed out"
	}
	if ahdSMTPTLSFailed(err) || stage == "tls" {
		return "SMTP TLS verification failed"
	}
	switch stage {
	case "connect", "greeting":
		return "SMTP connection failed"
	case "auth":
		return "SMTP authentication failed"
	case "mail":
		return "SMTP MAIL FROM was rejected"
	case "rcpt":
		return "SMTP recipient was rejected"
	case "data":
		return "SMTP DATA failed"
	case "quit":
		return "SMTP connection failed"
	default:
		return "SMTP connection failed"
	}
}

func ahdSMTPTimedOut(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "timeout") || strings.Contains(text, "deadline exceeded") || strings.Contains(text, "i/o timeout")
}

func ahdSMTPTLSFailed(err error) bool {
	if err == nil {
		return false
	}
	var unknown x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalid x509.CertificateInvalidError
	if errors.As(err, &unknown) || errors.As(err, &hostname) || errors.As(err, &invalid) {
		return true
	}
	var certErr x509.CertificateInvalidError
	_ = certErr
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "certificate") || strings.Contains(text, "tls:") || strings.Contains(text, "x509:")
}

func ahdSMTPRaise(class *AhdClass, message, password string) {
	if password != "" {
		message = strings.ReplaceAll(message, password, "")
	}
	AhdRaiseClass(class, message)
}
