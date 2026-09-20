package ahdruntime

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The SMTP attachment matrix. Every case reads the MIME the fixture captured
// and decodes it the way a mail client would, because an attachment is only
// correct if the recipient gets the file back byte for byte.

// capturedMessage is one message the fixture received, taken apart.
type capturedMessage struct {
	raw         string
	topLevel    string
	text        string
	html        string
	hasText     bool
	hasHTML     bool
	attachments []capturedAttachment
}

type capturedAttachment struct {
	fileName    string
	contentType string
	disposition string
	encoding    string
	content     []byte
	// encodedLines are the raw base64 lines exactly as they were sent.
	encodedLines []string
}

func (message *capturedMessage) attachment(name string) *capturedAttachment {
	for index := range message.attachments {
		if message.attachments[index].fileName == name {
			return &message.attachments[index]
		}
	}
	return nil
}

// readCapturedMessage parses the captured DATA into the four shapes AhdCode
// produces, and fails the test on anything it does not recognize.
func readCapturedMessage(t *testing.T, data []byte) *capturedMessage {
	t.Helper()
	parsed, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("read message: %v\n%s", err, data)
	}
	captured := &capturedMessage{raw: string(data)}
	mediaType, parameters, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("Content-Type: %v", err)
	}
	captured.topLevel = mediaType
	switch mediaType {
	case "text/plain", "text/html":
		body := decodeTransfer(t, parsed.Body, parsed.Header.Get("Content-Transfer-Encoding"))
		if mediaType == "text/plain" {
			captured.text, captured.hasText = trimBodyEnd(string(body)), true
		} else {
			captured.html, captured.hasHTML = trimBodyEnd(string(body)), true
		}
	case "multipart/alternative":
		readAlternative(t, captured, multipart.NewReader(parsed.Body, parameters["boundary"]))
	case "multipart/mixed":
		readMixed(t, captured, multipart.NewReader(parsed.Body, parameters["boundary"]))
	default:
		t.Fatalf("unexpected top-level media type %q", mediaType)
	}
	return captured
}

func readAlternative(t *testing.T, captured *capturedMessage, reader *multipart.Reader) {
	t.Helper()
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Fatalf("alternative part: %v", err)
		}
		mediaType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("alternative part type: %v", err)
		}
		body := decodeTransfer(t, part, part.Header.Get("Content-Transfer-Encoding"))
		switch mediaType {
		case "text/plain":
			captured.text, captured.hasText = trimBodyEnd(string(body)), true
		case "text/html":
			captured.html, captured.hasHTML = trimBodyEnd(string(body)), true
		default:
			t.Fatalf("unexpected alternative part %q", mediaType)
		}
	}
}

func readMixed(t *testing.T, captured *capturedMessage, reader *multipart.Reader) {
	t.Helper()
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Fatalf("mixed part: %v", err)
		}
		mediaType, parameters, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("mixed part type: %v", err)
		}
		disposition := part.Header.Get("Content-Disposition")
		if disposition == "" {
			switch mediaType {
			case "multipart/alternative":
				readAlternative(t, captured, multipart.NewReader(part, parameters["boundary"]))
			case "text/plain":
				captured.text, captured.hasText = trimBodyEnd(string(decodeTransfer(t, part, part.Header.Get("Content-Transfer-Encoding")))), true
			case "text/html":
				captured.html, captured.hasHTML = trimBodyEnd(string(decodeTransfer(t, part, part.Header.Get("Content-Transfer-Encoding")))), true
			default:
				t.Fatalf("unexpected body part %q", mediaType)
			}
			continue
		}
		rawEncoded, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("attachment body: %v", err)
		}
		lines := strings.Split(strings.TrimSuffix(strings.ReplaceAll(string(rawEncoded), "\r\n", "\n"), "\n"), "\n")
		content, err := base64.StdEncoding.DecodeString(strings.Join(lines, ""))
		if err != nil {
			t.Fatalf("attachment base64: %v", err)
		}
		_, dispositionParameters, err := mime.ParseMediaType(disposition)
		if err != nil {
			t.Fatalf("Content-Disposition: %v", err)
		}
		captured.attachments = append(captured.attachments, capturedAttachment{
			fileName: dispositionParameters["filename"], contentType: mediaType,
			disposition: disposition, encoding: part.Header.Get("Content-Transfer-Encoding"),
			content: content, encodedLines: lines,
		})
	}
}

// trimBodyEnd drops the trailing line break the SMTP DATA writer adds when
// the message does not already end with one. It has been there since
// v0.9.0 and is not part of what a body says.
func trimBodyEnd(body string) string {
	return strings.TrimRight(body, "\r\n")
}

func decodeTransfer(t *testing.T, reader io.Reader, encoding string) []byte {
	t.Helper()
	if strings.EqualFold(encoding, "quoted-printable") {
		decoded, err := io.ReadAll(quotedprintable.NewReader(reader))
		if err != nil {
			t.Fatalf("quoted-printable: %v", err)
		}
		return decoded
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read part: %v", err)
	}
	return raw
}

func attachmentFixtures(t *testing.T) (directory string, pdf, png, empty, invalid string, payloads map[string][]byte) {
	t.Helper()
	directory = t.TempDir()
	payloads = map[string][]byte{
		"report.pdf":  append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte{0x00, 0xFF, 0x10, 0x0A, 0x0D}, 400)...),
		"chart.png":   append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte{0x7F, 0x80, 0x00}, 300)...),
		"empty.dat":   {},
		"invalid.dat": {0xFF, 0xFE, 0xFD, 0x00, 0x80},
	}
	for name, content := range payloads {
		if err := os.WriteFile(filepath.Join(directory, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return directory, filepath.Join(directory, "report.pdf"), filepath.Join(directory, "chart.png"),
		filepath.Join(directory, "empty.dat"), filepath.Join(directory, "invalid.dat"), payloads
}

// A message with no attachment must produce exactly the MIME AhdCode has
// produced since v0.9.0: the streaming rewrite changed how the bytes are
// written, never what they are.
func TestSMTPWithoutAttachmentsKeepsItsShape(t *testing.T) {
	class := AhdClassSMTPError
	cases := []struct {
		name     string
		text     string
		html     string
		topLevel string
	}{
		{"text only", "plain body", "", "text/plain"},
		{"HTML only", "", "<p>rich</p>", "text/html"},
		{"text and HTML", "plain body", "<p>rich</p>", "multipart/alternative"},
	}
	for _, testCase := range cases {
		capture, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none"})
		client := AhdSMTPClient(class, host, int64(port), "none", 5)
		message := AhdSMTPMessage(class, "sender@example.com", []string{"student@example.com"}, "Subject")
		if testCase.text != "" {
			message = AhdSMTPMessageWithText(class, message, testCase.text)
		}
		if testCase.html != "" {
			message = AhdSMTPMessageWithHtml(class, message, testCase.html)
		}
		AhdSMTPClientSend(class, client, message)
		captured := readCapturedMessage(t, capture.data)
		if captured.topLevel != testCase.topLevel {
			t.Fatalf("%s: top-level = %q, want %q", testCase.name, captured.topLevel, testCase.topLevel)
		}
		if len(captured.attachments) != 0 {
			t.Fatalf("%s: a message with no attachment grew one", testCase.name)
		}
		if testCase.text != "" && captured.text != testCase.text {
			t.Fatalf("%s: text = %q", testCase.name, captured.text)
		}
		if testCase.html != "" && captured.html != testCase.html {
			t.Fatalf("%s: html = %q", testCase.name, captured.html)
		}
		if strings.Contains(captured.raw, "multipart/mixed") {
			t.Fatalf("%s: a message with no attachment gained a mixed wrapper", testCase.name)
		}
	}
}

func TestSMTPAttachmentMIMEStructure(t *testing.T) {
	class := AhdClassSMTPError
	_, pdf, png, _, _, payloads := attachmentFixtures(t)
	cases := []struct {
		name      string
		text      string
		html      string
		nested    string
		wantText  bool
		wantHTML  bool
		wantInner bool
	}{
		{"text and attachment", "plain body", "", "", true, false, false},
		{"HTML and attachment", "", "<p>rich</p>", "", false, true, false},
		{"text, HTML and attachment", "plain body", "<p>rich</p>", "multipart/alternative", true, true, true},
	}
	for _, testCase := range cases {
		capture, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none"})
		client := AhdSMTPClient(class, host, int64(port), "none", 5)
		message := AhdSMTPMessage(class, "sender@example.com", []string{"student@example.com"}, "Subject")
		if testCase.text != "" {
			message = AhdSMTPMessageWithText(class, message, testCase.text)
		}
		if testCase.html != "" {
			message = AhdSMTPMessageWithHtml(class, message, testCase.html)
		}
		message = AhdSMTPMessageWithAttachment(class, message, pdf, "", "application/pdf")
		message = AhdSMTPMessageWithAttachment(class, message, png, "", "image/png")
		AhdSMTPClientSend(class, client, message)

		captured := readCapturedMessage(t, capture.data)
		if captured.topLevel != "multipart/mixed" {
			t.Fatalf("%s: top-level = %q", testCase.name, captured.topLevel)
		}
		if testCase.wantInner && !strings.Contains(captured.raw, "multipart/alternative") {
			t.Fatalf("%s: text and HTML were flattened out of their alternative", testCase.name)
		}
		if !testCase.wantInner && strings.Contains(captured.raw, "multipart/alternative") {
			t.Fatalf("%s: an alternative appeared where there is one body", testCase.name)
		}
		if testCase.wantText && captured.text != testCase.text {
			t.Fatalf("%s: text = %q", testCase.name, captured.text)
		}
		if testCase.wantHTML && captured.html != testCase.html {
			t.Fatalf("%s: html = %q", testCase.name, captured.html)
		}
		if len(captured.attachments) != 2 {
			t.Fatalf("%s: %d attachments", testCase.name, len(captured.attachments))
		}
		// Attachments keep the order they were added in.
		if captured.attachments[0].fileName != "report.pdf" || captured.attachments[1].fileName != "chart.png" {
			t.Fatalf("%s: order = %s, %s", testCase.name,
				captured.attachments[0].fileName, captured.attachments[1].fileName)
		}
		for _, attachment := range captured.attachments {
			expected := payloads[attachment.fileName]
			if !bytes.Equal(attachment.content, expected) {
				t.Fatalf("%s: %s did not survive the round trip", testCase.name, attachment.fileName)
			}
			if attachment.encoding != "base64" {
				t.Fatalf("%s: %s encoding = %q", testCase.name, attachment.fileName, attachment.encoding)
			}
			if !strings.HasPrefix(attachment.disposition, "attachment") {
				t.Fatalf("%s: %s disposition = %q", testCase.name, attachment.fileName, attachment.disposition)
			}
			for _, line := range attachment.encodedLines {
				if len(line) > ahdSMTPBase64LineBytes {
					t.Fatalf("%s: a base64 line is %d characters long", testCase.name, len(line))
				}
			}
		}
		if captured.attachment("report.pdf").contentType != "application/pdf" {
			t.Fatalf("%s: pdf content type", testCase.name)
		}
		if captured.attachment("chart.png").contentType != "image/png" {
			t.Fatalf("%s: png content type", testCase.name)
		}
		// Every line of the message is CRLF-terminated, as MIME requires.
		if strings.Contains(strings.ReplaceAll(captured.raw, "\r\n", ""), "\n") {
			t.Fatalf("%s: the message contains a bare LF", testCase.name)
		}
	}
}

func TestSMTPAttachmentNamesAndTypes(t *testing.T) {
	class := AhdClassSMTPError
	_, pdf, _, empty, invalid, payloads := attachmentFixtures(t)
	capture, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none"})
	client := AhdSMTPClient(class, host, int64(port), "none", 5)
	message := AhdSMTPMessage(class, "sender@example.com", []string{"student@example.com"}, "Subject")
	message = AhdSMTPMessageWithText(class, message, "see attached")
	// An empty presentation name takes the basename; an explicit one is used
	// as given, including a name that is not ASCII.
	message = AhdSMTPMessageWithAttachment(class, message, pdf, "", "application/pdf")
	message = AhdSMTPMessageWithAttachment(class, message, invalid, "Ölçüm Raporu — Çukurova.dat", "")
	message = AhdSMTPMessageWithAttachment(class, message, empty, "nothing.dat", "text/plain")
	AhdSMTPClientSend(class, client, message)

	captured := readCapturedMessage(t, capture.data)
	if len(captured.attachments) != 3 {
		t.Fatalf("%d attachments", len(captured.attachments))
	}
	if name := captured.attachments[0].fileName; name != "report.pdf" {
		t.Fatalf("default name = %q", name)
	}
	unicode := captured.attachments[1]
	if unicode.fileName != "Ölçüm Raporu — Çukurova.dat" {
		t.Fatalf("unicode name = %q", unicode.fileName)
	}
	if unicode.contentType != "application/octet-stream" {
		t.Fatalf("default content type = %q", unicode.contentType)
	}
	if !bytes.Equal(unicode.content, payloads["invalid.dat"]) {
		t.Fatal("an attachment that is not valid UTF-8 did not survive")
	}
	blank := captured.attachments[2]
	if blank.fileName != "nothing.dat" || len(blank.content) != 0 || blank.contentType != "text/plain" {
		t.Fatalf("zero-byte attachment = %#v", blank)
	}
}

func TestSMTPAttachmentKeepsSMTPMessageImmutable(t *testing.T) {
	class := AhdClassSMTPError
	_, pdf, png, _, _, _ := attachmentFixtures(t)
	base := AhdSMTPMessage(class, "sender@example.com", []string{"student@example.com"}, "Subject")
	one := AhdSMTPMessageWithAttachment(class, base, pdf, "", "application/pdf")
	two := AhdSMTPMessageWithAttachment(class, one, png, "", "image/png")

	if got := ahdSMTPDecodeMessage(class, base).Attachments; len(got) != 0 {
		t.Fatalf("the original message gained attachments: %v", got)
	}
	if got := ahdSMTPDecodeMessage(class, one).Attachments; len(got) != 1 || got[0].FileName != "report.pdf" {
		t.Fatalf("one = %#v", got)
	}
	got := ahdSMTPDecodeMessage(class, two).Attachments
	if len(got) != 2 || got[1].FileName != "chart.png" || got[1].ContentType != "image/png" {
		t.Fatalf("two = %#v", got)
	}
	// The configured message holds paths, never file content.
	if strings.Contains(two, "%PDF") || strings.Contains(two, "PNG") {
		t.Fatalf("the message value carries file content: %s", two)
	}
}

func TestSMTPAttachmentConfigurationIsValidated(t *testing.T) {
	class := AhdClassSMTPError
	directory, pdf, _, _, _, _ := attachmentFixtures(t)
	base := AhdSMTPMessage(class, "sender@example.com", []string{"student@example.com"}, "Subject")

	// A path is required and must not carry a NUL.
	mustRaiseSMTP(t, func() { AhdSMTPMessageWithAttachment(class, base, "", "", "") })
	mustRaiseSMTP(t, func() { AhdSMTPMessageWithAttachment(class, base, "   ", "", "") })
	mustRaiseSMTP(t, func() { AhdSMTPMessageWithAttachment(class, base, "a\x00b", "", "") })
	// A display name must be usable in a header.
	mustRaiseSMTP(t, func() { AhdSMTPMessageWithAttachment(class, base, pdf, "bad\x00name", "") })
	mustRaiseSMTP(t, func() {
		AhdSMTPMessageWithAttachment(class, base, pdf, strings.Repeat("n", ahdSMTPMaxAttachmentNameBytes+1), "")
	})
	// A content type must be a real media type on one line.
	for _, contentType := range []string{"not a media type", "text/plain\r\nX: y", "text/plain\x00",
		strings.Repeat("t", ahdSMTPMaxAttachmentTypeBytes+1)} {
		value := contentType
		mustRaiseSMTP(t, func() { AhdSMTPMessageWithAttachment(class, base, pdf, "", value) })
	}
	// The attachment count is bounded.
	message := base
	for index := 0; index < ahdSMTPMaxAttachments; index++ {
		message = AhdSMTPMessageWithAttachment(class, message, pdf, "", "")
	}
	full := message
	mustRaiseSMTP(t, func() { AhdSMTPMessageWithAttachment(class, full, pdf, "", "") })

	// A missing file and a directory are found when the message is sent,
	// before the connection is opened.
	_, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none"})
	client := AhdSMTPClient(class, host, int64(port), "none", 5)
	missing := AhdSMTPMessageWithText(class,
		AhdSMTPMessageWithAttachment(class, base, filepath.Join(directory, "absent"), "", ""), "body")
	mustRaiseSMTP(t, func() { AhdSMTPClientSend(class, client, missing) })
	folder := AhdSMTPMessageWithText(class, AhdSMTPMessageWithAttachment(class, base, directory, "", ""), "body")
	mustRaiseSMTP(t, func() { AhdSMTPClientSend(class, client, folder) })
}

// A display name that looks like a path is reduced to a basename, so an
// attachment can never suggest a location to a mail client.
func TestSMTPAttachmentNameIsNeverAPath(t *testing.T) {
	class := AhdClassSMTPError
	_, pdf, _, _, _, _ := attachmentFixtures(t)
	for presented, expected := range map[string]string{
		"../../etc/passwd":      "passwd",
		`..\..\windows\hosts`:   "hosts",
		`C:\Users\a\secret.txt`: "secret.txt",
	} {
		capture, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none"})
		client := AhdSMTPClient(class, host, int64(port), "none", 5)
		message := AhdSMTPMessageWithText(class,
			AhdSMTPMessage(class, "sender@example.com", []string{"student@example.com"}, "Subject"), "body")
		message = AhdSMTPMessageWithAttachment(class, message, pdf, presented, "")
		AhdSMTPClientSend(class, client, message)
		captured := readCapturedMessage(t, capture.data)
		if len(captured.attachments) != 1 || captured.attachments[0].fileName != expected {
			t.Fatalf("%q was presented as %#v, want %q", presented, captured.attachments, expected)
		}
	}
}

func TestSMTPAttachmentsKeepBccHiddenAndThePasswordQuiet(t *testing.T) {
	class := AhdClassSMTPError
	_, pdf, _, _, _, _ := attachmentFixtures(t)
	capture, host, port := startSMTPFixture(t, smtpFixtureOptions{
		security: "none", advertiseAuthPlain: true,
	})
	client := AhdSMTPClient(class, host, int64(port), "none", 5)
	message := AhdSMTPMessage(class, "sender@example.com", []string{"student@example.com"}, "Subject")
	message = AhdSMTPMessageWithBcc(class, message, []string{"secret@example.com"})
	message = AhdSMTPMessageWithText(class, message, "body")
	message = AhdSMTPMessageWithAttachment(class, message, pdf, "", "application/pdf")
	AhdSMTPClientSend(class, client, message)

	raw := string(capture.data)
	if strings.Contains(strings.ToLower(raw), "bcc:") || strings.Contains(raw, "secret@example.com") {
		t.Fatalf("Bcc leaked into a message with an attachment:\n%s", raw)
	}
	// The envelope still reaches the blind recipient.
	found := false
	for _, recipient := range capture.recipients {
		if recipient == "secret@example.com" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the blind recipient was dropped: %v", capture.recipients)
	}

	// An attachment failure on an authenticated client never names the
	// password, whatever the file path is.
	authenticated := AhdSMTPClientWithPlainAuth(class,
		AhdSMTPClient(class, host, int64(port), "starttls", 5), "user", smtpQAPassword)
	broken := AhdSMTPMessageWithText(class,
		AhdSMTPMessageWithAttachment(class,
			AhdSMTPMessage(class, "sender@example.com", []string{"student@example.com"}, "Subject"),
			filepath.Join(t.TempDir(), smtpQAPassword+".pdf"), "", ""), "body")
	message2 := broken
	failure := captureSMTPRaise(t, func() { AhdSMTPClientSend(class, authenticated, message2) })
	if strings.Contains(failure, smtpQAPassword) {
		t.Fatalf("the failure names the password: %s", failure)
	}
}

// An attachment much larger than any buffer the runtime holds still arrives
// intact, because the file is encoded straight into the DATA stream.
func TestSMTPAttachmentStreamsALargeFile(t *testing.T) {
	class := AhdClassSMTPError
	directory := t.TempDir()
	payload := make([]byte, 3<<20)
	for index := range payload {
		payload[index] = byte(index * 17 % 253)
	}
	path := filepath.Join(directory, "large.bin")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	capture, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none"})
	client := AhdSMTPClient(class, host, int64(port), "none", 30)
	message := AhdSMTPMessageWithText(class,
		AhdSMTPMessage(class, "sender@example.com", []string{"student@example.com"}, "Subject"), "body")
	message = AhdSMTPMessageWithAttachment(class, message, path, "", "")
	AhdSMTPClientSend(class, client, message)

	captured := readCapturedMessage(t, capture.data)
	if len(captured.attachments) != 1 {
		t.Fatalf("%d attachments", len(captured.attachments))
	}
	if !bytes.Equal(captured.attachments[0].content, payload) {
		t.Fatal("the streamed attachment differs from the file")
	}
}

func TestSMTPAttachmentSizeLimits(t *testing.T) {
	class := AhdClassSMTPError
	directory := t.TempDir()
	path := filepath.Join(directory, "sparse.bin")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	// A sparse file is a cheap way to have a file whose reported size is past
	// the single-attachment limit without writing that many bytes.
	if err := file.Truncate(ahdSMTPMaxAttachmentBytes + 1); err != nil {
		_ = file.Close()
		t.Skipf("this filesystem cannot make a sparse file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	_, host, port := startSMTPFixture(t, smtpFixtureOptions{security: "none"})
	client := AhdSMTPClient(class, host, int64(port), "none", 5)
	message := AhdSMTPMessageWithText(class,
		AhdSMTPMessage(class, "sender@example.com", []string{"student@example.com"}, "Subject"), "body")
	oversize := AhdSMTPMessageWithAttachment(class, message, path, "", "")
	failure := captureSMTPRaise(t, func() { AhdSMTPClientSend(class, client, oversize) })
	if !strings.Contains(failure, "128 MiB") {
		t.Fatalf("single-attachment limit = %q", failure)
	}

	// Three of them are inside the single limit but past the total.
	total := message
	for index := 0; index < 3; index++ {
		total = AhdSMTPMessageWithAttachment(class, total, path, "", "")
	}
	combined := total
	failure = captureSMTPRaise(t, func() { AhdSMTPClientSend(class, client, combined) })
	if failure == "" {
		t.Fatal("the total attachment limit was not enforced")
	}
}

// captureSMTPRaise runs body, requires that it raised SMTPError, and returns
// the message.
func captureSMTPRaise(t *testing.T, body func()) (message string) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected an AhdCode SMTPError")
		}
		signal, ok := recovered.(*AhdSignal)
		if !ok {
			t.Fatalf("expected an AhdSignal; received %v", recovered)
		}
		if signal.Instance.AhdClassOf() != AhdClassSMTPError {
			t.Fatalf("expected SMTPError; received %s", signal.Instance.AhdClassOf().Name)
		}
		message = signal.Message
	}()
	body()
	return ""
}
