package ahdruntime

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildMultipart returns one multipart body plus its content type.
func buildMultipart(t *testing.T, fields map[string]string, files []struct {
	Field, Filename, ContentType string
	Content                      []byte
}) ([]byte, string) {
	t.Helper()
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range files {
		header := make(map[string][]string)
		header["Content-Disposition"] = []string{
			`form-data; name="` + file.Field + `"; filename="` + file.Filename + `"`,
		}
		if file.ContentType != "" {
			header["Content-Type"] = []string{file.ContentType}
		}
		part, err := writer.CreatePart(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(file.Content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes(), writer.FormDataContentType()
}

type uploadFixture struct {
	Field, Filename, ContentType string
	Content                      []byte
}

// materializeUpload parses one multipart request exactly as the server does
// and returns the encoded request plus the ids to release.
func materializeUpload(t *testing.T, fields map[string]string, files []uploadFixture) (string, []string) {
	t.Helper()
	converted := make([]struct {
		Field, Filename, ContentType string
		Content                      []byte
	}, len(files))
	for index, file := range files {
		converted[index] = struct {
			Field, Filename, ContentType string
			Content                      []byte
		}{file.Field, file.Filename, file.ContentType, file.Content}
	}
	body, contentType := buildMultipart(t, fields, converted)
	request := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body))
	request.Header.Set("Content-Type", contentType)
	snapshot, ids, err := ahdHTTPMaterialize(request, body)
	if err != nil {
		t.Fatalf("multipart materialization failed: %v", err)
	}
	return ahdHTTPEncodeRequest(snapshot), ids
}

func TestMultipartUploadMetadataAndTextFields(t *testing.T) {
	class := AhdClassHTTPError
	pdf := append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte{0x00, 0xFF, 0x41}, 40)...)
	encoded, ids := materializeUpload(t,
		map[string]string{"title": "Banach spaces"},
		[]uploadFixture{{Field: "paper", Filename: "paper.pdf", ContentType: "application/pdf", Content: pdf}})
	defer ahdHTTPReleaseUploads(ids)

	// Multipart text fields reach the ordinary form API, not a second one.
	if value := AhdHTTPRequestForm(class, encoded, "title"); value == nil || *value != "Banach spaces" {
		t.Fatalf("form(title) = %v", value)
	}
	file := AhdHTTPRequestFile(class, encoded, "paper")
	if file == nil {
		t.Fatal("expected an uploaded file for field paper")
	}
	if got := AhdHTTPUploadedFileOriginalName(class, *file); got != "paper.pdf" {
		t.Fatalf("originalName = %q", got)
	}
	declared := AhdHTTPUploadedFileDeclaredContentType(class, *file)
	if declared == nil || *declared != "application/pdf" {
		t.Fatalf("declaredContentType = %v", declared)
	}
	if got := AhdHTTPUploadedFileDetectedContentType(class, *file); got != "application/pdf" {
		t.Fatalf("detectedContentType = %q", got)
	}
	if got := AhdHTTPUploadedFileSize(class, *file); got != int64(len(pdf)) {
		t.Fatalf("size = %d; want %d", got, len(pdf))
	}
	if missing := AhdHTTPRequestFile(class, encoded, "nothing"); missing != nil {
		t.Fatal("expected null for a field with no file")
	}
	if all := AhdHTTPRequestFiles(class, encoded, "nothing"); len(all) != 0 {
		t.Fatalf("files(nothing) = %d entries", len(all))
	}
}

func TestUploadedFileSaveIsSafeAndByteExact(t *testing.T) {
	class := AhdClassHTTPError
	payload := bytes.Repeat([]byte{0x00, 0xFF, 0xFE, 0x7F, 0x80}, 500)
	want := sha256.Sum256(payload)
	encoded, ids := materializeUpload(t, nil,
		[]uploadFixture{{Field: "paper", Filename: "../../evil.pdf", Content: payload}})
	defer ahdHTTPReleaseUploads(ids)

	file := AhdHTTPRequestFile(class, encoded, "paper")
	if file == nil {
		t.Fatal("expected an uploaded file")
	}
	// The browser filename is display metadata only, reduced to a basename.
	if got := AhdHTTPUploadedFileOriginalName(class, *file); got != "evil.pdf" {
		t.Fatalf("originalName = %q; traversal must not survive", got)
	}
	directory := filepath.Join(t.TempDir(), "uploads", "papers")
	stored := AhdHTTPUploadedFileSave(class, *file, directory)

	if filepath.Dir(stored) != directory {
		t.Fatalf("stored %q escaped %q", stored, directory)
	}
	if strings.Contains(filepath.Base(stored), "evil") {
		t.Fatalf("stored basename %q is attacker-controlled", filepath.Base(stored))
	}
	content, err := os.ReadFile(stored)
	if err != nil {
		t.Fatal(err)
	}
	if got := sha256.Sum256(content); got != want {
		t.Fatalf("saved bytes differ: %s vs %s", hex.EncodeToString(got[:]), hex.EncodeToString(want[:]))
	}
	// An upload persists once; a second save must not silently duplicate it.
	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected the second save to raise")
			}
		}()
		AhdHTTPUploadedFileSave(class, *file, directory)
	}()
}

func TestUploadedFileSameNameDoesNotOverwrite(t *testing.T) {
	class := AhdClassHTTPError
	directory := filepath.Join(t.TempDir(), "papers")
	seen := map[string]bool{}
	for round := 0; round < 3; round++ {
		encoded, ids := materializeUpload(t, nil,
			[]uploadFixture{{Field: "paper", Filename: "paper.pdf", Content: []byte("%PDF-1.7\nround")}})
		file := AhdHTTPRequestFile(class, encoded, "paper")
		if file == nil {
			t.Fatal("expected an uploaded file")
		}
		stored := AhdHTTPUploadedFileSave(class, *file, directory)
		if seen[stored] {
			t.Fatalf("stored path %q was reused; an upload overwrote another", stored)
		}
		seen[stored] = true
		ahdHTTPReleaseUploads(ids)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 stored files, found %d", len(entries))
	}
}

func TestUploadMIMEDetectionIgnoresNameAndDeclaration(t *testing.T) {
	class := AhdClassHTTPError
	cases := []struct {
		name     string
		filename string
		declared string
		content  []byte
		detected string
	}{
		{"real pdf", "paper.pdf", "application/pdf", []byte("%PDF-1.7\n\x00\x01binary"), "application/pdf"},
		{"fake pdf", "paper.pdf", "application/pdf", []byte("hello world, not a pdf"), "text/plain"},
		{"png", "image.png", "image/png", append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{7}, 64)...), "image/png"},
		{"jpeg", "photo.jpg", "image/jpeg", append([]byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00"), bytes.Repeat([]byte{7}, 64)...), "image/jpeg"},
		{"text", "notes.txt", "text/plain; charset=utf-8", []byte("plain ascii text"), "text/plain"},
		{"binary", "blob.bin", "application/octet-stream", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}, "application/octet-stream"},
		{"png name, text bytes", "image.png", "image/png", []byte("still just text"), "text/plain"},
		{"empty", "empty.pdf", "application/pdf", nil, "application/octet-stream"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			encoded, ids := materializeUpload(t, nil,
				[]uploadFixture{{Field: "f", Filename: test.filename, ContentType: test.declared, Content: test.content}})
			defer ahdHTTPReleaseUploads(ids)
			file := AhdHTTPRequestFile(class, encoded, "f")
			if file == nil {
				t.Fatal("expected an uploaded file")
			}
			if got := AhdHTTPUploadedFileDetectedContentType(class, *file); got != test.detected {
				t.Fatalf("detected = %q; want %q", got, test.detected)
			}
			// The declaration is reported verbatim (normalized), never merged
			// into detection.
			declared := AhdHTTPUploadedFileDeclaredContentType(class, *file)
			wantDeclared := ahdHTTPUploadMediaType(test.declared)
			if declared == nil || *declared != wantDeclared {
				t.Fatalf("declared = %v; want %q", declared, wantDeclared)
			}
		})
	}
}

func TestUploadReleaseRemovesUnsavedTemporaryFiles(t *testing.T) {
	class := AhdClassHTTPError
	encoded, ids := materializeUpload(t, nil,
		[]uploadFixture{{Field: "paper", Filename: "paper.pdf", Content: []byte("%PDF-1.7\nunsaved")}})
	file := AhdHTTPRequestFile(class, encoded, "paper")
	if file == nil {
		t.Fatal("expected an uploaded file")
	}
	entry := ahdHTTPUploadEntryFor(class, *file)
	ahdHTTPUploads.mutex.Lock()
	record := ahdHTTPUploads.records[entry.ID]
	ahdHTTPUploads.mutex.Unlock()
	if record == nil {
		t.Fatal("expected a registry record while the request is live")
	}
	tempPath := record.tempPath
	if _, err := os.Stat(tempPath); err != nil {
		t.Fatalf("temporary backing missing: %v", err)
	}

	ahdHTTPReleaseUploads(ids)

	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("unsaved temporary file survived release: %v", err)
	}
	ahdHTTPUploads.mutex.Lock()
	_, stillRegistered := ahdHTTPUploads.records[entry.ID]
	ahdHTTPUploads.mutex.Unlock()
	if stillRegistered {
		t.Fatal("registry entry survived release")
	}
	// Saving after release is refused rather than resurrecting the upload.
	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected save after release to raise")
			}
		}()
		AhdHTTPUploadedFileSave(class, *file, t.TempDir())
	}()
}

func TestUploadMultipleFilesUnderOneField(t *testing.T) {
	class := AhdClassHTTPError
	encoded, ids := materializeUpload(t, nil, []uploadFixture{
		{Field: "papers", Filename: "a.pdf", Content: []byte("%PDF-1.7\na")},
		{Field: "papers", Filename: "b.pdf", Content: []byte("%PDF-1.7\nb")},
		{Field: "other", Filename: "c.pdf", Content: []byte("%PDF-1.7\nc")},
	})
	defer ahdHTTPReleaseUploads(ids)

	all := AhdHTTPRequestFiles(class, encoded, "papers")
	if len(all) != 2 {
		t.Fatalf("files(papers) = %d; want 2", len(all))
	}
	if name := AhdHTTPUploadedFileOriginalName(class, all[0]); name != "a.pdf" {
		t.Fatalf("request order not preserved: first is %q", name)
	}
	if name := AhdHTTPUploadedFileOriginalName(class, all[1]); name != "b.pdf" {
		t.Fatalf("request order not preserved: second is %q", name)
	}
	first := AhdHTTPRequestFile(class, encoded, "papers")
	if first == nil || AhdHTTPUploadedFileOriginalName(class, *first) != "a.pdf" {
		t.Fatal("file(papers) must be the first uploaded file")
	}
}

func TestUploadSafeNameReducesHostilePaths(t *testing.T) {
	cases := map[string]string{
		"paper.pdf":                "paper.pdf",
		"../../evil.pdf":           "evil.pdf",
		`..\..\evil.pdf`:           "evil.pdf",
		`C:\Users\Alice\paper.pdf`: "paper.pdf",
		"folder/paper.pdf":         "paper.pdf",
		"..":                       "file",
		".":                        "file",
		"":                         "file",
	}
	for input, want := range cases {
		got, err := ahdHTTPUploadSafeName(input)
		if err != nil {
			t.Fatalf("%q: unexpected error %v", input, err)
		}
		if got != want {
			t.Fatalf("%q -> %q; want %q", input, got, want)
		}
		if strings.ContainsAny(got, `/\`) {
			t.Fatalf("%q produced a separator: %q", input, got)
		}
	}
	if _, err := ahdHTTPUploadSafeName("paper\x00.pdf"); err == nil {
		t.Fatal("a NUL byte in a filename must be rejected")
	}
}

func TestMultipartMalformedIsRejectedBeforeHandler(t *testing.T) {
	cases := map[string]struct {
		contentType string
		body        string
	}{
		"missing boundary": {"multipart/form-data", "garbage"},
		"invalid boundary": {"multipart/form-data; boundary=zzz", "--nope\r\ngarbage"},
		"truncated part": {"multipart/form-data; boundary=X",
			"--X\r\nContent-Disposition: form-data; name=\"p\"; filename=\"a.pdf\"\r\n\r\ntruncated"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			body := []byte(test.body)
			request := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body))
			request.Header.Set("Content-Type", test.contentType)
			if _, _, err := ahdHTTPMaterialize(request, body); err == nil {
				t.Fatal("expected malformed multipart to fail materialization (400), not reach a handler")
			}
		})
	}
}
