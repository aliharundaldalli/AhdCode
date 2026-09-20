package ahdruntime

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The download matrix. A download exists so a program can take bytes AhdCode
// has no type for, so every case here checks the file on disk byte for byte
// rather than anything the program could have seen.

// binaryFixtures are payloads a String could not have carried: a NUL byte, a
// lone 0xFF that is not valid UTF-8, and the leading bytes of the three
// formats an AhdCode program most often moves.
func binaryFixtures() map[string][]byte {
	png := append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte{0x00, 0xFF, 0x7F}, 300)...)
	zip := append([]byte{'P', 'K', 0x03, 0x04}, bytes.Repeat([]byte{0xC0, 0x80, 0x00}, 200)...)
	pdf := append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte{0x00, 0x01, 0xFE}, 200)...)
	random := make([]byte, 4096)
	for index := range random {
		random[index] = byte(index * 37 % 256)
	}
	return map[string][]byte{
		"empty":       {},
		"ascii":       []byte("plain text\nsecond line\n"),
		"nul":         {0x00, 0x00, 'a', 0x00},
		"invalidUTF8": {0xFF, 0xFE, 0xFD, 0x80},
		"png":         png,
		"zip":         zip,
		"pdf":         pdf,
		"random":      random,
		"crlf":        []byte("keep\r\nthese\r\nexact\r\n"),
	}
}

func downloadClient(t *testing.T, maxResponseBytes int64) string {
	t.Helper()
	return AhdHTTPClient(AhdClassHTTPError, 10, maxResponseBytes, true)
}

func TestHTTPDownloadWritesExactBytes(t *testing.T) {
	class := AhdClassHTTPError
	fixtures := binaryFixtures()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		payload, known := fixtures[strings.TrimPrefix(request.URL.Path, "/")]
		if !known {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		writer.Header().Set("Content-Type", "application/octet-stream")
		writer.Header().Add("X-Trace", "one")
		writer.Header().Add("X-Trace", "two")
		_, _ = writer.Write(payload)
	}))
	defer server.Close()
	handle := downloadClient(t, 1<<20)
	directory := t.TempDir()

	for name, payload := range fixtures {
		destination := filepath.Join(directory, name+".bin")
		response := AhdHTTPClientDownload(class, handle, server.URL+"/"+name, destination)
		if status := AhdHTTPClientFileResponseStatus(response); status != 200 {
			t.Fatalf("%s: status = %d", name, status)
		}
		if size := AhdHTTPClientFileResponseSize(class, response); size != int64(len(payload)) {
			t.Fatalf("%s: size = %d, want %d", name, size, len(payload))
		}
		written, err := os.ReadFile(destination)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !bytes.Equal(written, payload) {
			t.Fatalf("%s: the downloaded bytes differ from what the server sent", name)
		}
		if header := AhdHTTPClientFileResponseHeader(class, response, "content-type"); header == nil || *header != "application/octet-stream" {
			t.Fatalf("%s: Content-Type = %v", name, header)
		}
		if all := AhdHTTPClientFileResponseHeaderAll(class, response, "X-Trace").Snapshot(); len(all) != 2 {
			t.Fatalf("%s: duplicate headers = %v", name, all)
		}
		if url := AhdHTTPClientFileResponseURL(class, response); url != server.URL+"/"+name {
			t.Fatalf("%s: url = %s", name, url)
		}
	}
	if remaining := temporaryDownloadFiles(t, directory); len(remaining) != 0 {
		t.Fatalf("temporary files were left behind: %v", remaining)
	}
}

// temporaryDownloadFiles lists anything the download machinery left behind.
// A successful download renames its temporary file into place, and a failed
// one removes it, so this must always come back empty.
func temporaryDownloadFiles(t *testing.T, directory string) []string {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	var leftovers []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".ahdcode-download-") {
			leftovers = append(leftovers, entry.Name())
		}
	}
	return leftovers
}

func TestHTTPDownloadKeepsNonSuccessStatusAndBody(t *testing.T) {
	class := AhdClassHTTPError
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/missing":
			writer.WriteHeader(http.StatusNotFound)
			_, _ = writer.Write([]byte("no such thing"))
		default:
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte("it broke"))
		}
	}))
	defer server.Close()
	handle := downloadClient(t, 1<<20)
	directory := t.TempDir()

	for path, expected := range map[string]struct {
		status int64
		body   string
	}{"/missing": {404, "no such thing"}, "/broken": {500, "it broke"}} {
		destination := filepath.Join(directory, strings.TrimPrefix(path, "/"))
		response := AhdHTTPClientDownload(class, handle, server.URL+path, destination)
		if status := AhdHTTPClientFileResponseStatus(response); status != expected.status {
			t.Fatalf("%s: status = %d, want %d", path, status, expected.status)
		}
		written, err := os.ReadFile(destination)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if string(written) != expected.body {
			t.Fatalf("%s: body = %q", path, written)
		}
	}
}

func TestHTTPDownloadFollowsRedirectsAndReportsFinalURL(t *testing.T) {
	class := AhdClassHTTPError
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/start" {
			http.Redirect(writer, request, server.URL+"/final", http.StatusFound)
			return
		}
		_, _ = writer.Write([]byte{0x00, 0x01, 0x02})
	}))
	defer server.Close()
	destination := filepath.Join(t.TempDir(), "out.bin")
	response := AhdHTTPClientDownload(class, downloadClient(t, 1<<20), server.URL+"/start", destination)
	if url := AhdHTTPClientFileResponseURL(class, response); url != server.URL+"/final" {
		t.Fatalf("final url = %s", url)
	}
	written, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(written, []byte{0x00, 0x01, 0x02}) {
		t.Fatalf("body = %v, %v", written, err)
	}
}

func TestHTTPDownloadFailureLeavesTheDestinationAlone(t *testing.T) {
	class := AhdClassHTTPError
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/large":
			_, _ = writer.Write(bytes.Repeat([]byte{'x'}, 4096))
		case "/truncated":
			writer.Header().Set("Content-Length", "4096")
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write(bytes.Repeat([]byte{'y'}, 16))
			if flusher, ok := writer.(http.Flusher); ok {
				flusher.Flush()
			}
			panic(http.ErrAbortHandler)
		case "/slow":
			time.Sleep(3 * time.Second)
		}
	}))
	defer server.Close()
	directory := t.TempDir()
	original := []byte("the file that was already there")

	cases := []struct {
		name    string
		path    string
		timeout int64
		limit   int64
	}{
		{"over the response limit", "/large", 10, 1024},
		{"a truncated response", "/truncated", 10, 1 << 20},
		{"a timeout", "/slow", 1, 1 << 20},
	}
	for _, testCase := range cases {
		destination := filepath.Join(directory, "keep.bin")
		if err := os.WriteFile(destination, original, 0o600); err != nil {
			t.Fatal(err)
		}
		handle := AhdHTTPClient(class, testCase.timeout, testCase.limit, true)
		expectRaise(t, class, func() {
			AhdHTTPClientDownload(class, handle, server.URL+testCase.path, destination)
		})
		survived, err := os.ReadFile(destination)
		if err != nil {
			t.Fatalf("%s: the destination is gone: %v", testCase.name, err)
		}
		if !bytes.Equal(survived, original) {
			t.Fatalf("%s: the destination was overwritten by a failed download", testCase.name)
		}
		if remaining := temporaryDownloadFiles(t, directory); len(remaining) != 0 {
			t.Fatalf("%s: temporary files were left behind: %v", testCase.name, remaining)
		}
	}
}

func TestHTTPDownloadDestinationRules(t *testing.T) {
	class := AhdClassHTTPError
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("ok"))
	}))
	defer server.Close()
	handle := downloadClient(t, 1<<20)
	directory := t.TempDir()

	// A missing parent directory is refused rather than created: AhdCode does
	// not make directories a program did not ask for.
	expectRaise(t, class, func() {
		AhdHTTPClientDownload(class, handle, server.URL, filepath.Join(directory, "absent", "out.bin"))
	})
	// A directory is not a destination.
	expectRaise(t, class, func() {
		AhdHTTPClientDownload(class, handle, server.URL, directory)
	})
	// Neither is an empty path.
	expectRaise(t, class, func() { AhdHTTPClientDownload(class, handle, server.URL, "") })
	expectRaise(t, class, func() { AhdHTTPClientDownload(class, handle, server.URL, "  ") })

	// An unwritable directory fails without leaving anything behind.
	if os.Geteuid() != 0 {
		locked := filepath.Join(directory, "locked")
		if err := os.Mkdir(locked, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
		expectRaise(t, class, func() {
			AhdHTTPClientDownload(class, handle, server.URL, filepath.Join(locked, "out.bin"))
		})
	}
}

func TestHTTPDownloadReplacesAnExistingFile(t *testing.T) {
	class := AhdClassHTTPError
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	}))
	defer server.Close()
	destination := filepath.Join(t.TempDir(), "out.bin")
	if err := os.WriteFile(destination, []byte("older, longer content"), 0o600); err != nil {
		t.Fatal(err)
	}
	AhdHTTPClientDownload(class, downloadClient(t, 1<<20), server.URL, destination)
	written, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(written, []byte{0xDE, 0xAD, 0xBE, 0xEF}) {
		t.Fatalf("replacement = %v, %v", written, err)
	}
}

func TestHTTPSendToFileCarriesHeadersAndMethod(t *testing.T) {
	class := AhdClassHTTPError
	var seenMethod, seenHeader, seenBody string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		seenMethod = request.Method
		seenHeader = request.Header.Get("X-Token")
		body, _ := readAllLimited(request, 1024)
		seenBody = string(body)
		_, _ = writer.Write([]byte{0x01, 0x02, 0x03})
	}))
	defer server.Close()
	request := AhdHTTPClientRequest(class, "POST", server.URL)
	request = AhdHTTPClientRequestWithHeader(class, request, "X-Token", "abc")
	request = AhdHTTPClientRequestWithBody(class, request, "question")
	destination := filepath.Join(t.TempDir(), "answer.bin")
	response := AhdHTTPClientSendToFile(class, downloadClient(t, 1<<20), request, destination)
	if AhdHTTPClientFileResponseStatus(response) != 200 {
		t.Fatal("status")
	}
	if seenMethod != "POST" || seenHeader != "abc" || seenBody != "question" {
		t.Fatalf("request = %s %s %q", seenMethod, seenHeader, seenBody)
	}
	written, _ := os.ReadFile(destination)
	if !bytes.Equal(written, []byte{0x01, 0x02, 0x03}) {
		t.Fatalf("body = %v", written)
	}
}

func readAllLimited(request *http.Request, limit int64) ([]byte, error) {
	defer func() { _ = request.Body.Close() }()
	var buffer bytes.Buffer
	_, err := buffer.ReadFrom(newLimitedReader(request, limit))
	return buffer.Bytes(), err
}

func newLimitedReader(request *http.Request, limit int64) *limitedRequestReader {
	return &limitedRequestReader{request: request, remaining: limit}
}

type limitedRequestReader struct {
	request   *http.Request
	remaining int64
}

func (reader *limitedRequestReader) Read(buffer []byte) (int, error) {
	if reader.remaining <= 0 {
		return 0, fmt.Errorf("body exceeded the test limit")
	}
	if int64(len(buffer)) > reader.remaining {
		buffer = buffer[:reader.remaining]
	}
	count, err := reader.request.Body.Read(buffer)
	reader.remaining -= int64(count)
	return count, err
}

func TestHTTPClientFileResponseIsOpaqueMetadataOnly(t *testing.T) {
	class := AhdClassHTTPError
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte{0xFF, 0x00, 0xFF})
	}))
	defer server.Close()
	destination := filepath.Join(t.TempDir(), "out.bin")
	response := AhdHTTPClientDownload(class, downloadClient(t, 1<<20), server.URL, destination)
	// The encoded value a ClientFileResponse carries must not contain the
	// payload: that is the whole point of writing it to a file.
	if strings.ContainsAny(response, "\xFF") {
		t.Fatalf("the response value carries payload bytes: %q", response)
	}
	if strings.Contains(response, destination) {
		t.Fatalf("the response value carries the destination path: %q", response)
	}
	if AhdHTTPClientFileResponseSize(class, response) != 3 {
		t.Fatal("size")
	}
	// Corrupted storage is reported, never silently read as something else.
	expectRaise(t, class, func() { AhdHTTPClientFileResponseURL(class, "not json") })
}
