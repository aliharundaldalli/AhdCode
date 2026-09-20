package ahdruntime

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// The outbound multipart matrix. Every case checks what the server actually
// received, because the point of streaming a file part is that the bytes on
// the wire are the bytes on disk.

// multipartCapture is what one fixture request carried.
type multipartCapture struct {
	mutex       sync.Mutex
	contentType string
	fields      map[string][]string
	files       []capturedFile
	failure     string
}

type capturedFile struct {
	field       string
	fileName    string
	contentType string
	content     []byte
}

func (capture *multipartCapture) file(field string) *capturedFile {
	capture.mutex.Lock()
	defer capture.mutex.Unlock()
	for index := range capture.files {
		if capture.files[index].field == field {
			return &capture.files[index]
		}
	}
	return nil
}

// startMultipartFixture serves one endpoint that parses whatever multipart
// body it is sent and records it exactly.
func startMultipartFixture(t *testing.T) (*multipartCapture, *httptest.Server) {
	t.Helper()
	capture := &multipartCapture{fields: map[string][]string{}}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		capture.mutex.Lock()
		capture.contentType = request.Header.Get("Content-Type")
		capture.fields = map[string][]string{}
		capture.files = nil
		capture.failure = ""
		capture.mutex.Unlock()

		mediaType, parameters, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			capture.mutex.Lock()
			capture.failure = "not multipart: " + request.Header.Get("Content-Type")
			capture.mutex.Unlock()
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		reader := multipart.NewReader(request.Body, parameters["boundary"])
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				capture.mutex.Lock()
				capture.failure = err.Error()
				capture.mutex.Unlock()
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			content, readErr := io.ReadAll(part)
			if readErr != nil {
				capture.mutex.Lock()
				capture.failure = readErr.Error()
				capture.mutex.Unlock()
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			capture.mutex.Lock()
			if part.FileName() == "" {
				capture.fields[part.FormName()] = append(capture.fields[part.FormName()], string(content))
			} else {
				capture.files = append(capture.files, capturedFile{
					field: part.FormName(), fileName: part.FileName(),
					contentType: part.Header.Get("Content-Type"), content: content,
				})
			}
			capture.mutex.Unlock()
			_ = part.Close()
		}
		_, _ = writer.Write([]byte("received"))
	}))
	t.Cleanup(server.Close)
	return capture, server
}

func writeFixtureFile(t *testing.T, directory, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestHTTPMultipartSendsFieldsAndFilesExactly(t *testing.T) {
	class := AhdClassHTTPError
	capture, server := startMultipartFixture(t)
	directory := t.TempDir()
	binary := []byte{0x89, 'P', 'N', 'G', 0x00, 0xFF, 0x80, 0x0A, 0x0D}
	invalid := []byte{0xFF, 0xFE, 0x00}
	pngPath := writeFixtureFile(t, directory, "picture.png", binary)
	rawPath := writeFixtureFile(t, directory, "raw.dat", invalid)
	emptyPath := writeFixtureFile(t, directory, "empty.dat", nil)

	request := AhdHTTPClientRequest(class, "POST", server.URL)
	request = AhdHTTPClientRequestWithMultipartField(class, request, "title", "a plain field")
	request = AhdHTTPClientRequestWithMultipartField(class, request, "note", "Türkçe ölçüm — ünicode")
	request = AhdHTTPClientRequestWithMultipartField(class, request, "tag", "first")
	request = AhdHTTPClientRequestWithMultipartField(class, request, "tag", "second")
	request = AhdHTTPClientRequestWithMultipartFile(class, request, "picture", pngPath, "", "image/png")
	request = AhdHTTPClientRequestWithMultipartFile(class, request, "raw", rawPath, "renamed.bin", "")
	request = AhdHTTPClientRequestWithMultipartFile(class, request, "blank", emptyPath, "", "")

	response := AhdHTTPClientSend(class, downloadClient(t, 1<<20), request)
	if AhdHTTPClientResponseStatus(response) != 200 {
		t.Fatalf("status = %d, failure = %s", AhdHTTPClientResponseStatus(response), capture.failure)
	}
	if capture.failure != "" {
		t.Fatalf("the fixture could not parse the body: %s", capture.failure)
	}
	if !strings.HasPrefix(capture.contentType, "multipart/form-data; boundary=ahdcode-") {
		t.Fatalf("Content-Type = %s", capture.contentType)
	}
	if got := capture.fields["title"]; len(got) != 1 || got[0] != "a plain field" {
		t.Fatalf("title = %v", got)
	}
	if got := capture.fields["note"]; len(got) != 1 || got[0] != "Türkçe ölçüm — ünicode" {
		t.Fatalf("unicode field = %v", got)
	}
	// Repeating a field name appends, in order, the way an HTML form does.
	if got := capture.fields["tag"]; len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("repeated field = %v", got)
	}

	picture := capture.file("picture")
	if picture == nil || !bytes.Equal(picture.content, binary) {
		t.Fatalf("picture bytes = %v", picture)
	}
	// An empty presentation name takes the path's basename.
	if picture.fileName != "picture.png" || picture.contentType != "image/png" {
		t.Fatalf("picture metadata = %s %s", picture.fileName, picture.contentType)
	}
	raw := capture.file("raw")
	if raw == nil || !bytes.Equal(raw.content, invalid) {
		t.Fatalf("raw bytes = %v", raw)
	}
	// An explicit presentation name is used, and an empty content type falls
	// back to application/octet-stream.
	if raw.fileName != "renamed.bin" || raw.contentType != "application/octet-stream" {
		t.Fatalf("raw metadata = %s %s", raw.fileName, raw.contentType)
	}
	blank := capture.file("blank")
	if blank == nil || len(blank.content) != 0 {
		t.Fatalf("zero-byte file = %v", blank)
	}
}

func TestHTTPMultipartKeepsClientRequestImmutable(t *testing.T) {
	class := AhdClassHTTPError
	directory := t.TempDir()
	path := writeFixtureFile(t, directory, "one.bin", []byte{1, 2, 3})
	base := AhdHTTPClientRequest(class, "POST", "http://127.0.0.1:1/")
	withField := AhdHTTPClientRequestWithMultipartField(class, base, "a", "1")
	withFile := AhdHTTPClientRequestWithMultipartFile(class, withField, "f", path, "", "")

	if parts := ahdHTTPDecodeClientRequest(class, base).Multipart; len(parts) != 0 {
		t.Fatalf("the original request gained parts: %v", parts)
	}
	if parts := ahdHTTPDecodeClientRequest(class, withField).Multipart; len(parts) != 1 {
		t.Fatalf("the field-only request has %d parts", len(parts))
	}
	parts := ahdHTTPDecodeClientRequest(class, withFile).Multipart
	if len(parts) != 2 || parts[0].Name != "a" || !parts[1].IsFile || parts[1].Path != path {
		t.Fatalf("parts = %#v", parts)
	}
	// The configured request holds the path, never the file's bytes.
	if strings.Contains(withFile, "\\u0001\\u0002\\u0003") {
		t.Fatalf("the request value carries file content: %s", withFile)
	}
}

func TestHTTPMultipartRefusesAmbiguousBodyOwnership(t *testing.T) {
	class := AhdClassHTTPError
	directory := t.TempDir()
	path := writeFixtureFile(t, directory, "one.bin", []byte{1})
	base := AhdHTTPClientRequest(class, "POST", "http://127.0.0.1:1/")

	// A body already set, then a multipart part.
	withBody := AhdHTTPClientRequestWithBody(class, base, "explicit")
	expectRaise(t, class, func() { AhdHTTPClientRequestWithMultipartField(class, withBody, "a", "1") })
	expectRaise(t, class, func() { AhdHTTPClientRequestWithMultipartFile(class, withBody, "f", path, "", "") })
	// An empty explicit body still counts as a body.
	emptyBody := AhdHTTPClientRequestWithBody(class, base, "")
	expectRaise(t, class, func() { AhdHTTPClientRequestWithMultipartField(class, emptyBody, "a", "1") })

	// Multipart parts already set, then a body.
	withPart := AhdHTTPClientRequestWithMultipartField(class, base, "a", "1")
	expectRaise(t, class, func() { AhdHTTPClientRequestWithBody(class, withPart, "explicit") })

	// A manual Content-Type, in either order.
	typed := AhdHTTPClientRequestWithHeader(class, base, "Content-Type", "application/json")
	expectRaise(t, class, func() { AhdHTTPClientRequestWithMultipartField(class, typed, "a", "1") })
	expectRaise(t, class, func() { AhdHTTPClientRequestWithHeader(class, withPart, "content-type", "application/json") })
	expectRaise(t, class, func() {
		AhdHTTPClientRequestWithHeader(class, withPart, "Content-Type", "multipart/form-data; boundary=zzz")
	})
	expectRaise(t, class, func() { AhdHTTPClientRequestAddHeader(class, withPart, "Content-Type", "application/json") })
	// An unrelated header is still fine on a multipart request.
	if AhdHTTPClientRequestWithHeader(class, withPart, "Authorization", "Bearer x") == "" {
		t.Fatal("an unrelated header was refused")
	}
}

func TestHTTPMultipartRejectsUnusableMetadata(t *testing.T) {
	class := AhdClassHTTPError
	directory := t.TempDir()
	path := writeFixtureFile(t, directory, "one.bin", []byte{1})
	base := AhdHTTPClientRequest(class, "POST", "http://127.0.0.1:1/")

	// Header injection through a field name or a presentation filename.
	for _, name := range []string{"", "bad\rname", "bad\nname", "bad\r\nname", "quo\"ted", "nul\x00name",
		strings.Repeat("n", ahdHTTPMaxMultipartNameBytes+1)} {
		expectRaise(t, class, func() { AhdHTTPClientRequestWithMultipartField(class, base, name, "v") })
		expectRaise(t, class, func() { AhdHTTPClientRequestWithMultipartFile(class, base, name, path, "", "") })
	}
	for _, fileName := range []string{"bad\x00name", strings.Repeat("f", ahdHTTPMaxMultipartFileNameBytes+1)} {
		expectRaise(t, class, func() { AhdHTTPClientRequestWithMultipartFile(class, base, "f", path, fileName, "") })
	}
	// A path is required and must not carry a NUL.
	expectRaise(t, class, func() { AhdHTTPClientRequestWithMultipartFile(class, base, "f", "", "", "") })
	expectRaise(t, class, func() { AhdHTTPClientRequestWithMultipartFile(class, base, "f", "a\x00b", "", "") })
	// The content type must be a media type, on one line, of bounded length.
	for _, contentType := range []string{"not a media type", "text/plain\r\nX: y", "text/plain\x00",
		strings.Repeat("t", ahdHTTPMaxMultipartTypeBytes+1)} {
		expectRaise(t, class, func() { AhdHTTPClientRequestWithMultipartFile(class, base, "f", path, "", contentType) })
	}
	// A field value is bounded, and must be text.
	expectRaise(t, class, func() {
		AhdHTTPClientRequestWithMultipartField(class, base, "f", strings.Repeat("v", ahdHTTPMaxMultipartValueBytes+1))
	})
	expectRaise(t, class, func() { AhdHTTPClientRequestWithMultipartField(class, base, "f", "\xff\xfe") })
}

// A presentation filename that looks like a path is reduced to a basename,
// so it can never travel as a path component the server might act on.
func TestHTTPMultipartFileNameIsNeverAPath(t *testing.T) {
	class := AhdClassHTTPError
	capture, server := startMultipartFixture(t)
	directory := t.TempDir()
	path := writeFixtureFile(t, directory, "one.bin", []byte{7})

	for presented, expected := range map[string]string{
		"../../etc/passwd":      "passwd",
		`..\..\windows\hosts`:   "hosts",
		`C:\Users\a\secret.txt`: "secret.txt",
		"..":                    "file",
		"/":                     "file",
	} {
		request := AhdHTTPClientRequest(class, "POST", server.URL)
		request = AhdHTTPClientRequestWithMultipartFile(class, request, "f", path, presented, "")
		AhdHTTPClientSend(class, downloadClient(t, 1<<20), request)
		file := capture.file("f")
		if file == nil || file.fileName != expected {
			t.Fatalf("%q was presented as %v, want %q", presented, file, expected)
		}
	}
}

func TestHTTPMultipartRejectsUnreadableFilesAtSendTime(t *testing.T) {
	class := AhdClassHTTPError
	_, server := startMultipartFixture(t)
	directory := t.TempDir()
	handle := downloadClient(t, 1<<20)

	// A missing file and a directory both fail before anything is sent.
	missing := AhdHTTPClientRequestWithMultipartFile(class,
		AhdHTTPClientRequest(class, "POST", server.URL), "f", filepath.Join(directory, "absent"), "", "")
	expectRaise(t, class, func() { AhdHTTPClientSend(class, handle, missing) })

	folder := AhdHTTPClientRequestWithMultipartFile(class,
		AhdHTTPClientRequest(class, "POST", server.URL), "f", directory, "", "")
	expectRaise(t, class, func() { AhdHTTPClientSend(class, handle, folder) })
}

func TestHTTPMultipartBoundsThePartCount(t *testing.T) {
	class := AhdClassHTTPError
	directory := t.TempDir()
	path := writeFixtureFile(t, directory, "one.bin", []byte{1})
	request := AhdHTTPClientRequest(class, "POST", "http://127.0.0.1:1/")
	for index := 0; index < ahdHTTPMaxMultipartFields; index++ {
		request = AhdHTTPClientRequestWithMultipartField(class, request, "field", "value")
	}
	full := request
	expectRaise(t, class, func() { AhdHTTPClientRequestWithMultipartField(class, full, "one more", "value") })

	files := AhdHTTPClientRequest(class, "POST", "http://127.0.0.1:1/")
	for index := 0; index < ahdHTTPMaxMultipartFiles; index++ {
		files = AhdHTTPClientRequestWithMultipartFile(class, files, "file", path, "", "")
	}
	tooMany := files
	expectRaise(t, class, func() { AhdHTTPClientRequestWithMultipartFile(class, tooMany, "one more", path, "", "") })
}

// 307 and 308 keep the method and the body, so the transport replays it. A
// replayed multipart body must be the same bytes under the same boundary, or
// the server sees a body that does not match the Content-Type it was given.
func TestHTTPMultipartReplaysTheSameBodyOnA307Redirect(t *testing.T) {
	class := AhdClassHTTPError
	capture := &multipartCapture{fields: map[string][]string{}}
	var bodies []string
	var types []string
	var mutex sync.Mutex
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/start" {
			http.Redirect(writer, request, server.URL+"/final", http.StatusTemporaryRedirect)
			return
		}
		raw, _ := io.ReadAll(request.Body)
		mutex.Lock()
		bodies = append(bodies, string(raw))
		types = append(types, request.Header.Get("Content-Type"))
		mutex.Unlock()

		mediaType, parameters, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		reader := multipart.NewReader(bytes.NewReader(raw), parameters["boundary"])
		for {
			part, err := reader.NextPart()
			if err != nil {
				break
			}
			content, _ := io.ReadAll(part)
			capture.mutex.Lock()
			if part.FileName() == "" {
				capture.fields[part.FormName()] = append(capture.fields[part.FormName()], string(content))
			} else {
				capture.files = append(capture.files, capturedFile{field: part.FormName(), content: content})
			}
			capture.mutex.Unlock()
		}
	}))
	defer server.Close()

	directory := t.TempDir()
	payload := bytes.Repeat([]byte{0x00, 0xFF}, 64)
	path := writeFixtureFile(t, directory, "one.bin", payload)
	request := AhdHTTPClientRequest(class, "POST", server.URL+"/start")
	request = AhdHTTPClientRequestWithMultipartField(class, request, "a", "1")
	request = AhdHTTPClientRequestWithMultipartFile(class, request, "f", path, "", "")

	response := AhdHTTPClientSend(class, downloadClient(t, 1<<20), request)
	if AhdHTTPClientResponseStatus(response) != 200 {
		t.Fatalf("status = %d", AhdHTTPClientResponseStatus(response))
	}
	mutex.Lock()
	defer mutex.Unlock()
	if len(bodies) != 1 {
		t.Fatalf("the final endpoint saw %d requests", len(bodies))
	}
	if bodies[0] == "" {
		t.Fatal("the replayed request sent an empty body")
	}
	if !strings.Contains(bodies[0], "boundary") && !strings.Contains(types[0], "boundary") {
		t.Fatalf("Content-Type = %s", types[0])
	}
	// The boundary named in the header must be the one the body actually uses.
	_, parameters, err := mime.ParseMediaType(types[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bodies[0], parameters["boundary"]) {
		t.Fatalf("the replayed body does not use the boundary the header names")
	}
	file := capture.file("f")
	if file == nil || !bytes.Equal(file.content, payload) {
		t.Fatalf("the replayed file part is wrong: %v", file)
	}
	if got := capture.fields["a"]; len(got) != 1 || got[0] != "1" {
		t.Fatalf("the replayed field is wrong: %v", got)
	}
}

// A multipart upload is streamed: the request body is produced as the
// connection drains it, so a payload much larger than any buffer the runtime
// holds still arrives intact.
func TestHTTPMultipartStreamsALargeFile(t *testing.T) {
	class := AhdClassHTTPError
	var received int64
	var digest byte
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mediaType, parameters, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		// The body is read a chunk at a time, exactly as a streaming server
		// would, and never held whole.
		reader := multipart.NewReader(request.Body, parameters["boundary"])
		for {
			part, err := reader.NextPart()
			if err != nil {
				break
			}
			buffer := make([]byte, 32*1024)
			for {
				count, readErr := part.Read(buffer)
				for index := 0; index < count; index++ {
					digest ^= buffer[index]
				}
				received += int64(count)
				if readErr != nil {
					break
				}
			}
		}
	}))
	defer server.Close()

	directory := t.TempDir()
	payload := make([]byte, 6<<20)
	for index := range payload {
		payload[index] = byte(index * 31 % 251)
	}
	path := writeFixtureFile(t, directory, "large.bin", payload)
	request := AhdHTTPClientRequestWithMultipartFile(class,
		AhdHTTPClientRequest(class, "POST", server.URL), "f", path, "", "")
	AhdHTTPClientSend(class, downloadClient(t, 1<<20), request)

	if received != int64(len(payload)) {
		t.Fatalf("the server received %d bytes, want %d", received, len(payload))
	}
	var expected byte
	for _, value := range payload {
		expected ^= value
	}
	if digest != expected {
		t.Fatal("the streamed bytes differ from the file")
	}
}

// An ordinary request keeps working exactly as it did before multipart
// existed: same body, same Content-Length, no multipart header.
func TestHTTPClientSendWithoutMultipartIsUnchanged(t *testing.T) {
	class := AhdClassHTTPError
	var seen struct {
		contentType   string
		contentLength int64
		body          string
		encoding      []string
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		seen.contentType = request.Header.Get("Content-Type")
		seen.contentLength = request.ContentLength
		seen.body = string(body)
		seen.encoding = request.TransferEncoding
		_, _ = writer.Write([]byte("ok"))
	}))
	defer server.Close()
	request := AhdHTTPClientRequest(class, "POST", server.URL)
	request = AhdHTTPClientRequestWithHeader(class, request, "Content-Type", "application/json")
	request = AhdHTTPClientRequestWithBody(class, request, `{"a":1}`)
	AhdHTTPClientSend(class, downloadClient(t, 1<<20), request)
	if seen.contentType != "application/json" || seen.body != `{"a":1}` {
		t.Fatalf("request = %s %q", seen.contentType, seen.body)
	}
	if seen.contentLength != int64(len(`{"a":1}`)) {
		t.Fatalf("Content-Length = %d", seen.contentLength)
	}
	if len(seen.encoding) != 0 {
		t.Fatalf("an ordinary request became %v", seen.encoding)
	}
}

// A file part streams straight from disk, so a multipart send can also be
// combined with a download in the same program without either buffering.
func TestHTTPMultipartAndDownloadRoundTrip(t *testing.T) {
	class := AhdClassHTTPError
	stored := map[string][]byte{}
	var mutex sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/upload":
			_, parameters, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
			if err != nil {
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			reader := multipart.NewReader(request.Body, parameters["boundary"])
			for {
				part, err := reader.NextPart()
				if err != nil {
					break
				}
				if part.FileName() == "" {
					continue
				}
				content, _ := io.ReadAll(part)
				mutex.Lock()
				stored[part.FileName()] = content
				mutex.Unlock()
			}
			_, _ = writer.Write([]byte("stored"))
		case "/download":
			mutex.Lock()
			content := stored[request.URL.Query().Get("name")]
			mutex.Unlock()
			writer.Header().Set("Content-Type", "application/octet-stream")
			_, _ = writer.Write(content)
		}
	}))
	defer server.Close()

	directory := t.TempDir()
	payload := append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte{0x00, 0xFF, 0x10}, 512)...)
	source := writeFixtureFile(t, directory, "report.pdf", payload)
	handle := downloadClient(t, 1<<20)

	request := AhdHTTPClientRequestWithMultipartFile(class,
		AhdHTTPClientRequest(class, "POST", server.URL+"/upload"), "file", source, "", "application/pdf")
	if AhdHTTPClientResponseStatus(AhdHTTPClientSend(class, handle, request)) != 200 {
		t.Fatal("upload")
	}
	destination := filepath.Join(directory, "returned.pdf")
	response := AhdHTTPClientDownload(class, handle, server.URL+"/download?name=report.pdf", destination)
	if AhdHTTPClientFileResponseSize(class, response) != int64(len(payload)) {
		t.Fatalf("size = %d", AhdHTTPClientFileResponseSize(class, response))
	}
	returned, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(returned, payload) {
		t.Fatal("the round trip did not preserve the bytes")
	}
}
