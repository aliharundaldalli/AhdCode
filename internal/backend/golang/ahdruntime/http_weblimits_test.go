package ahdruntime

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebLimitSizeAndDurationParsing(t *testing.T) {
	size, err := ahdHTTPParseSize("AHD_WEB_MAX_BODY_SIZE", "16MB")
	if err != nil || size != 16<<20 {
		t.Fatalf("16MB = %d err=%v", size, err)
	}
	size, err = ahdHTTPParseSize("AHD_WEB_MAX_UPLOAD_SIZE", "512B")
	if err != nil || size != 512 {
		t.Fatalf("512B = %d err=%v", size, err)
	}
	if _, err := ahdHTTPParseSize("AHD_WEB_MAX_UPLOAD_SIZE", "potato"); err == nil {
		t.Fatal("potato was accepted")
	}
	if _, err := ahdHTTPParseSize("AHD_WEB_MAX_BODY_SIZE", "-1MB"); err == nil {
		t.Fatal("negative size was accepted")
	}
	if _, err := ahdHTTPParseSize("AHD_WEB_MAX_BODY_SIZE", "99999999999GB"); err == nil {
		t.Fatal("overflow size was accepted")
	}
	duration, err := ahdHTTPParseDuration("AHD_WEB_READ_TIMEOUT", "30s")
	if err != nil || duration != 30*time.Second {
		t.Fatalf("30s = %v err=%v", duration, err)
	}
	duration, err = ahdHTTPParseDuration("AHD_WEB_WRITE_TIMEOUT", "500ms")
	if err != nil || duration != 500*time.Millisecond {
		t.Fatalf("500ms = %v err=%v", duration, err)
	}
	if _, err := ahdHTTPParseDuration("AHD_WEB_IDLE_TIMEOUT", "nope"); err == nil {
		t.Fatal("invalid duration was accepted")
	}
	if _, err := ahdHTTPParseDuration("AHD_WEB_READ_TIMEOUT", "-1s"); err == nil {
		t.Fatal("negative duration was accepted")
	}
}

func TestWebLimitInconsistentUploadVsBodyRejected(t *testing.T) {
	handle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", 1, 1024)
	expectRaise(t, AhdClassHTTPError, func() {
		AhdHTTPServerSetLimits(AhdClassHTTPError, handle, 1024, 2048, 1, time.Second, time.Second, time.Second)
	})
}

func TestWebLimitBodyRejectedBeforeFullRead(t *testing.T) {
	base, _ := startHTTP(t, 32, func(handle string) {
		AhdHTTPServerPost(AhdClassHTTPError, handle, "/", func(string) string {
			return AhdHTTPText(AhdClassHTTPError, "ok", 200)
		})
		AhdHTTPServerSetLimits(AhdClassHTTPError, handle, 32, 16, 2, time.Second, time.Second, time.Second)
	})
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Post(base+"/", "text/plain", strings.NewReader(strings.Repeat("x", 64)))
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize body status = %d", response.StatusCode)
	}
}

func TestWebLimitExactBoundarySucceeds(t *testing.T) {
	body := strings.Repeat("a", 16)
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	request.Header.Set("Content-Type", "text/plain")
	snapshot, _, err := ahdHTTPMaterialize(request, []byte(body), 16, 1)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Body != body {
		t.Fatalf("boundary body = %q", snapshot.Body)
	}
}

func TestWebLimitFileAndCountRejected(t *testing.T) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	part, err := writer.CreateFormFile("file", "one.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("hello-world")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	body := buffer.Bytes()
	request := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if _, _, err := ahdHTTPMaterialize(request, body, 4, 1); err != errHTTPUploadTooLarge {
		t.Fatalf("oversize file err = %v", err)
	}

	buffer.Reset()
	writer = multipart.NewWriter(&buffer)
	for _, name := range []string{"a.txt", "b.txt"} {
		part, err := writer.CreateFormFile("file", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte("ok")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	body = buffer.Bytes()
	request = httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if _, _, err := ahdHTTPMaterialize(request, body, 64, 1); err != errHTTPTooManyUploads {
		t.Fatalf("too many files err = %v", err)
	}
}
