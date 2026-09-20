package ahdruntime

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
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
	// Attachment limits (v2.1.0). They are AhdCode's own deterministic
	// bounds, not a guess at a provider's: a message that fits these may
	// still be refused by a server, and the refusal is reported as it is for
	// any other rejected message. Thirty-two attachments and a quarter of a
	// gibibyte in total are far past ordinary mail and still bounded.
	ahdSMTPMaxAttachments      = 32
	ahdSMTPMaxAttachmentBytes  = int64(128) << 20
	ahdSMTPMaxAttachmentsBytes = int64(256) << 20
	// ahdSMTPMaxAttachmentNameBytes and ahdSMTPMaxAttachmentTypeBytes bound
	// the two pieces of metadata that go into a part header.
	ahdSMTPMaxAttachmentNameBytes = 255
	ahdSMTPMaxAttachmentTypeBytes = 255
	// ahdSMTPBase64LineBytes is the base64 line length RFC 2045 asks for.
	ahdSMTPBase64LineBytes = 76
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
	// Attachments (v2.1.0) name local files. Their bytes are read at send
	// time and base64-encoded straight into the DATA stream; nothing here
	// ever holds a file's content.
	Attachments []ahdSMTPAttachment `json:"attachments,omitempty"`
}

// ahdSMTPAttachment is one configured attachment: where to read it from, the
// filename the recipient sees, and the media type it is declared as.
type ahdSMTPAttachment struct {
	Path        string `json:"path"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
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

// AhdSMTPMessageWithAttachment is
// SMTPMessage.withAttachment(path, fileName, contentType). Each call appends
// one attachment and returns a new SMTPMessage, so an SMTPMessage stays
// immutable and attachments keep the order they were added in.
//
// The file is not read here. It is opened when the message is sent, so a
// configured message holds a path, never a payload.
func AhdSMTPMessageWithAttachment(class *AhdClass, data, path, fileName, contentType string) string {
	message := ahdSMTPDecodeMessage(class, data)
	if strings.TrimSpace(path) == "" {
		ahdSMTPRaise(class, "SMTP attachment path must not be empty", "")
	}
	if strings.IndexByte(path, 0) >= 0 {
		ahdSMTPRaise(class, "SMTP attachment path must not contain a NUL byte", "")
	}
	if len(message.Attachments)+1 > ahdSMTPMaxAttachments {
		ahdSMTPRaise(class, "SMTP message has too many attachments", "")
	}
	presented := ahdSMTPAttachmentName(class, path, fileName)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if len(contentType) > ahdSMTPMaxAttachmentTypeBytes {
		ahdSMTPRaise(class, "SMTP attachment content type is too long", "")
	}
	if ahdSMTPHasCRLF(contentType) || strings.IndexByte(contentType, 0) >= 0 {
		ahdSMTPRaise(class, "SMTP attachment content type must not contain a line break", "")
	}
	if _, _, err := mime.ParseMediaType(contentType); err != nil {
		ahdSMTPRaise(class, "SMTP attachment content type is not a valid media type", "")
	}
	message.Attachments = append(append([]ahdSMTPAttachment(nil), message.Attachments...),
		ahdSMTPAttachment{Path: path, FileName: presented, ContentType: contentType})
	return ahdSMTPEncodeMessage(class, message)
}

// ahdSMTPAttachmentName resolves the filename the recipient sees. An empty
// fileName takes the path's basename; either way the result is reduced to a
// plain basename by the same rule an inbound upload's filename is, so nothing
// a caller passes travels as a path component in a part header.
func ahdSMTPAttachmentName(class *AhdClass, path, fileName string) string {
	raw := fileName
	if strings.TrimSpace(raw) == "" {
		raw = filepath.Base(path)
	}
	safe, err := ahdHTTPUploadSafeName(raw)
	if err != nil {
		ahdSMTPRaise(class, "SMTP attachment file name is not valid", "")
	}
	if len(safe) > ahdSMTPMaxAttachmentNameBytes {
		ahdSMTPRaise(class, "SMTP attachment file name is too long", "")
	}
	return safe
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
	ahdSMTPCheckAttachments(class, parsed.attachments, config.Password)
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

		attachments: message.Attachments,
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
	from        ahdSMTPMailbox
	to          []ahdSMTPMailbox
	cc          []ahdSMTPMailbox
	bcc         []ahdSMTPMailbox
	replyTo     *ahdSMTPMailbox
	subject     string
	text        string
	html        string
	hasText     bool
	hasHTML     bool
	attachments []ahdSMTPAttachment
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
	if err := ahdSMTPWriteMIME(writer, parsed); err != nil {
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

// ahdSMTPCheckAttachments verifies every attachment before the connection is
// opened: each path must be a readable regular file, each file must stay
// within ahdSMTPMaxAttachmentBytes, and their total within
// ahdSMTPMaxAttachmentsBytes. Failing here means nothing was sent, which is a
// far clearer failure than a message that dies in the middle of DATA.
func ahdSMTPCheckAttachments(class *AhdClass, attachments []ahdSMTPAttachment, password string) {
	if len(attachments) > ahdSMTPMaxAttachments {
		ahdSMTPRaise(class, "SMTP message has too many attachments", password)
	}
	var total int64
	for _, attachment := range attachments {
		info, err := os.Stat(attachment.Path)
		if err != nil {
			if os.IsNotExist(err) {
				ahdSMTPRaise(class, "SMTP attachment file does not exist", password)
			}
			ahdSMTPRaise(class, "SMTP attachment file could not be read", password)
		}
		if info.IsDir() {
			ahdSMTPRaise(class, "SMTP attachment path is a directory", password)
		}
		if !info.Mode().IsRegular() {
			ahdSMTPRaise(class, "SMTP attachment path is not a regular file", password)
		}
		if info.Size() > ahdSMTPMaxAttachmentBytes {
			ahdSMTPRaise(class, "SMTP attachment is larger than 128 MiB", password)
		}
		total += info.Size()
		if total > ahdSMTPMaxAttachmentsBytes {
			ahdSMTPRaise(class, "SMTP attachments are larger than 256 MiB in total", password)
		}
		file, err := os.Open(attachment.Path)
		if err != nil {
			ahdSMTPRaise(class, "SMTP attachment file could not be opened", password)
		}
		_ = file.Close()
	}
}

// ahdSMTPWriteMIME writes the whole message to the DATA writer, incrementally.
// A message without attachments produces exactly the bytes AhdCode has
// produced since v0.9.0; attachments add a multipart/mixed wrapper around
// that same body and are base64-encoded straight from disk, so a 100 MB file
// never becomes a 100 MB buffer.
//
// The four shapes are:
//
//	text only                 text/plain
//	HTML only                 text/html
//	text and HTML             multipart/alternative
//	any of those + files      multipart/mixed { the above, attachments... }
func ahdSMTPWriteMIME(sink io.Writer, parsed ahdSMTPMessageParsed) error {
	var failure error
	writeHeader := func(name, value string) {
		if failure != nil {
			return
		}
		_, failure = io.WriteString(sink, name+": "+value+"\r\n")
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
	if failure != nil {
		return failure
	}

	if len(parsed.attachments) == 0 {
		return ahdSMTPWriteBody(sink, parsed, writeHeader, &failure)
	}

	// A multipart.Writer writes nothing until its first part, so its boundary
	// can be named in the Content-Type header before any part exists.
	mixed := multipart.NewWriter(sink)
	writeHeader("Content-Type", "multipart/mixed; boundary="+mixed.Boundary())
	if failure != nil {
		return failure
	}
	if _, err := io.WriteString(sink, "\r\n"); err != nil {
		return err
	}
	if err := ahdSMTPWriteBodyPart(mixed, parsed); err != nil {
		return err
	}
	for _, attachment := range parsed.attachments {
		if err := ahdSMTPWriteAttachment(mixed, attachment); err != nil {
			return err
		}
	}
	return mixed.Close()
}

// ahdSMTPWriteBody writes a message with no attachments: the body is the
// message, so its Content-Type is a top-level header.
func ahdSMTPWriteBody(sink io.Writer, parsed ahdSMTPMessageParsed, writeHeader func(string, string), failure *error) error {
	switch {
	case parsed.hasText && parsed.hasHTML:
		alternative := multipart.NewWriter(sink)
		writeHeader("Content-Type", "multipart/alternative; boundary="+alternative.Boundary())
		if *failure != nil {
			return *failure
		}
		if _, err := io.WriteString(sink, "\r\n"); err != nil {
			return err
		}
		if err := ahdSMTPWriteTextPart(alternative, "text/plain; charset=utf-8", parsed.text); err != nil {
			return err
		}
		if err := ahdSMTPWriteTextPart(alternative, "text/html; charset=utf-8", parsed.html); err != nil {
			return err
		}
		return alternative.Close()
	case parsed.hasHTML:
		writeHeader("Content-Type", "text/html; charset=utf-8")
		writeHeader("Content-Transfer-Encoding", "quoted-printable")
		if *failure != nil {
			return *failure
		}
		if _, err := io.WriteString(sink, "\r\n"); err != nil {
			return err
		}
		return ahdSMTPWriteQuotedPrintable(sink, parsed.html)
	default:
		writeHeader("Content-Type", "text/plain; charset=utf-8")
		writeHeader("Content-Transfer-Encoding", "quoted-printable")
		if *failure != nil {
			return *failure
		}
		if _, err := io.WriteString(sink, "\r\n"); err != nil {
			return err
		}
		return ahdSMTPWriteQuotedPrintable(sink, parsed.text)
	}
}

// ahdSMTPWriteBodyPart writes the same body as one part of a
// multipart/mixed. A message with both a text and an HTML body keeps its
// multipart/alternative, nested inside the mixed part rather than flattened
// beside the attachments.
func ahdSMTPWriteBodyPart(mixed *multipart.Writer, parsed ahdSMTPMessageParsed) error {
	if parsed.hasText && parsed.hasHTML {
		// A throwaway writer is the standard library's only way to ask for a
		// fresh random boundary; nothing is ever written to it.
		boundary := multipart.NewWriter(io.Discard).Boundary()
		header := make(textproto.MIMEHeader)
		header.Set("Content-Type", "multipart/alternative; boundary="+boundary)
		part, err := mixed.CreatePart(header)
		if err != nil {
			return err
		}
		alternative := multipart.NewWriter(part)
		if err := alternative.SetBoundary(boundary); err != nil {
			return err
		}
		if err := ahdSMTPWriteTextPart(alternative, "text/plain; charset=utf-8", parsed.text); err != nil {
			return err
		}
		if err := ahdSMTPWriteTextPart(alternative, "text/html; charset=utf-8", parsed.html); err != nil {
			return err
		}
		return alternative.Close()
	}
	contentType, body := "text/plain; charset=utf-8", parsed.text
	if parsed.hasHTML {
		contentType, body = "text/html; charset=utf-8", parsed.html
	}
	return ahdSMTPWriteTextPart(mixed, contentType, body)
}

func ahdSMTPWriteTextPart(writer *multipart.Writer, contentType, body string) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", contentType)
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	return ahdSMTPWriteQuotedPrintable(part, body)
}

// ahdSMTPWriteAttachment streams one file into the message. The bytes go
// file -> base64 -> line wrapper -> DATA writer, so only a small buffer
// exists at any moment and nothing ever becomes an AhdCode String.
func ahdSMTPWriteAttachment(mixed *multipart.Writer, attachment ahdSMTPAttachment) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", attachment.ContentType)
	header.Set("Content-Transfer-Encoding", "base64")
	header.Set("Content-Disposition", ahdSMTPDisposition(attachment.FileName))
	part, err := mixed.CreatePart(header)
	if err != nil {
		return err
	}
	file, err := os.Open(attachment.Path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	wrapper := &ahdSMTPLineWriter{sink: part, width: ahdSMTPBase64LineBytes}
	encoder := base64.NewEncoder(base64.StdEncoding, wrapper)
	if _, err := io.Copy(encoder, file); err != nil {
		_ = encoder.Close()
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	return wrapper.finish()
}

// ahdSMTPDisposition writes the Content-Disposition header. An ASCII name
// that needs no escaping becomes a plain quoted string; anything else uses
// the RFC 2231 encoded form, which is how a non-ASCII filename survives
// intact. The name is already a basename with no CR, LF, or NUL, so this
// cannot produce a header that spans a line.
func ahdSMTPDisposition(name string) string {
	formatted := mime.FormatMediaType("attachment", map[string]string{"filename": name})
	if formatted == "" {
		return "attachment"
	}
	return formatted
}

// ahdSMTPLineWriter breaks a base64 stream into CRLF-terminated lines of at
// most width characters, as RFC 2045 asks, without holding the whole encoded
// payload.
type ahdSMTPLineWriter struct {
	sink   io.Writer
	width  int
	filled int
}

func (writer *ahdSMTPLineWriter) Write(data []byte) (int, error) {
	written := 0
	for len(data) > 0 {
		room := writer.width - writer.filled
		chunk := data
		if len(chunk) > room {
			chunk = chunk[:room]
		}
		count, err := writer.sink.Write(chunk)
		written += count
		writer.filled += count
		if err != nil {
			return written, err
		}
		data = data[len(chunk):]
		if writer.filled == writer.width {
			if _, err := io.WriteString(writer.sink, "\r\n"); err != nil {
				return written, err
			}
			writer.filled = 0
		}
	}
	return written, nil
}

// finish ends a partly filled last line.
func (writer *ahdSMTPLineWriter) finish() error {
	if writer.filled == 0 {
		return nil
	}
	writer.filled = 0
	_, err := io.WriteString(writer.sink, "\r\n")
	return err
}

func ahdSMTPFormatList(boxes []ahdSMTPMailbox) string {
	parts := make([]string, len(boxes))
	for i, box := range boxes {
		parts[i] = box.header
	}
	return strings.Join(parts, ", ")
}

// ahdSMTPWriteQuotedPrintable encodes one body straight into the message.
// The bytes it produces are exactly what the buffered v0.9.0 path produced.
func ahdSMTPWriteQuotedPrintable(sink io.Writer, body string) error {
	writer := quotedprintable.NewWriter(sink)
	if _, err := writer.Write([]byte(ahdSMTPCRLF(body))); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
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
