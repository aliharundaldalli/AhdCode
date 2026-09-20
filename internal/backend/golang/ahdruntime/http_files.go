package ahdruntime

// Binary-safe outbound file transfer (v2.1.0).
//
// v0.6.0 gave AhdCode an HTTP client whose response body is a String, and
// v0.8.0 gave the server inbound multipart uploads whose bytes never become
// one. This file completes the pair in the other direction: a response body
// streamed straight to a file, and a request body streamed straight from
// files. Between them, an AhdCode program can move a PDF, a PNG, or a ZIP
// over HTTP without the language growing a Bytes type -- binary data stays
// opaque and travels file to file.
//
// Nothing here buffers a payload: a download is copied response -> temporary
// file -> destination, and an upload is written by mime/multipart into an
// io.Pipe the transport reads from. The only values that reach AhdCode are
// metadata: a status, headers, a final URL, and a byte count.
//
// This file is standard library only and joins every generated program,
// exactly like http.go.

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Bounds on what a program may configure. File *content* is streamed and is
// bounded by ahdHTTPMaxUploadTotalBytes alone; everything counted here is
// metadata that would otherwise be held in memory for the whole request.
const (
	// ahdHTTPMaxMultipartFields and ahdHTTPMaxMultipartFiles bound how many
	// parts one request may carry. A form with more than sixty-four text
	// fields or thirty-two files is a data feed, not a form.
	ahdHTTPMaxMultipartFields = 64
	ahdHTTPMaxMultipartFiles  = 32
	// ahdHTTPMaxMultipartNameBytes bounds a multipart field name, and
	// ahdHTTPMaxMultipartFileNameBytes the presentation filename. Both are
	// written into a part header, so both stay short.
	ahdHTTPMaxMultipartNameBytes     = 256
	ahdHTTPMaxMultipartFileNameBytes = 255
	ahdHTTPMaxMultipartTypeBytes     = 255
	// ahdHTTPMaxMultipartValueBytes bounds one text field's value. A text
	// field is already an AhdCode String, so this only keeps a program from
	// turning a megabyte-scale String into a form field by accident; a real
	// payload belongs in a file part.
	ahdHTTPMaxMultipartValueBytes = 1 << 20
	// ahdHTTPMaxUploadTotalBytes is the total raw size of every file part of
	// one request, measured before the body is streamed. Two gibibytes is
	// past any ordinary upload and still far below what a 64-bit counter or a
	// server's own limit would notice, so it bounds a runaway program without
	// getting in a real one's way. Each file's bytes are streamed; this is
	// not a memory bound.
	ahdHTTPMaxUploadTotalBytes = int64(2) << 30
)

// ahdHTTPMultipartPart is one configured part of an outbound
// multipart/form-data body. A file part carries the local path, never the
// file's bytes: the bytes are read at send time, straight from disk into the
// request.
type ahdHTTPMultipartPart struct {
	Name        string `json:"name"`
	IsFile      bool   `json:"isFile"`
	Value       string `json:"value,omitempty"`
	Path        string `json:"path,omitempty"`
	FileName    string `json:"fileName,omitempty"`
	ContentType string `json:"contentType,omitempty"`
}

// ahdHTTPClientFileResponseData is the whole public surface of a
// ClientFileResponse. There is no body and no stored path: the payload is in
// the file the caller named, and Size is what was actually written there.
type ahdHTTPClientFileResponseData struct {
	Status  int                 `json:"status"`
	Headers []ahdHTTPHeaderPair `json:"headers"`
	URL     string              `json:"url"`
	Size    int64               `json:"size"`
}

// --- ClientRequest multipart configuration ---

// AhdHTTPClientRequestWithMultipartField is
// ClientRequest.withMultipartField(name, value). It returns a new
// ClientRequest with one more text part appended; parts keep the order they
// were added in.
func AhdHTTPClientRequestWithMultipartField(class *AhdClass, data, name, value string) string {
	request := ahdHTTPDecodeClientRequest(class, data)
	ahdHTTPRequireMultipartFree(class, request)
	ahdHTTPRequireMultipartName(class, name)
	if len(value) > ahdHTTPMaxMultipartValueBytes {
		AhdRaiseClass(class, "HTTP multipart field value is too large; send large content as a file part")
	}
	if !utf8.ValidString(value) {
		AhdRaiseClass(class, "HTTP multipart field value is not valid UTF-8")
	}
	fields := 0
	for _, part := range request.Multipart {
		if !part.IsFile {
			fields++
		}
	}
	if fields+1 > ahdHTTPMaxMultipartFields {
		AhdRaiseClass(class, "HTTP request has too many multipart fields")
	}
	request.Multipart = append(append([]ahdHTTPMultipartPart(nil), request.Multipart...),
		ahdHTTPMultipartPart{Name: name, Value: value})
	return ahdHTTPEncodeClientRequest(class, request)
}

// AhdHTTPClientRequestWithMultipartFile is
// ClientRequest.withMultipartFile(name, path, fileName, contentType). The
// path names a local file that is read when the request is sent; fileName is
// presentation metadata the server sees and is never used as a path here.
func AhdHTTPClientRequestWithMultipartFile(class *AhdClass, data, name, path, fileName, contentType string) string {
	request := ahdHTTPDecodeClientRequest(class, data)
	ahdHTTPRequireMultipartFree(class, request)
	ahdHTTPRequireMultipartName(class, name)
	if strings.TrimSpace(path) == "" {
		AhdRaiseClass(class, "HTTP multipart file path must not be empty")
	}
	if strings.IndexByte(path, 0) >= 0 {
		AhdRaiseClass(class, "HTTP multipart file path must not contain a NUL byte")
	}
	presented := ahdHTTPMultipartPresentedName(class, path, fileName)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	ahdHTTPRequireMultipartContentType(class, contentType)
	files := 0
	for _, part := range request.Multipart {
		if part.IsFile {
			files++
		}
	}
	if files+1 > ahdHTTPMaxMultipartFiles {
		AhdRaiseClass(class, "HTTP request has too many multipart files")
	}
	request.Multipart = append(append([]ahdHTTPMultipartPart(nil), request.Multipart...),
		ahdHTTPMultipartPart{Name: name, IsFile: true, Path: path, FileName: presented, ContentType: contentType})
	return ahdHTTPEncodeClientRequest(class, request)
}

// ahdHTTPMultipartPresentedName resolves the filename the server is told.
// An empty fileName takes the path's basename; either way the result is
// reduced to a plain basename, so nothing a caller passes can travel as a
// path component in the part header.
func ahdHTTPMultipartPresentedName(class *AhdClass, path, fileName string) string {
	raw := fileName
	if strings.TrimSpace(raw) == "" {
		raw = filepath.Base(path)
	}
	safe, err := ahdHTTPUploadSafeName(raw)
	if err != nil {
		AhdRaiseClass(class, "HTTP multipart file name is not valid")
	}
	if len(safe) > ahdHTTPMaxMultipartFileNameBytes {
		AhdRaiseClass(class, "HTTP multipart file name is too long")
	}
	return safe
}

func ahdHTTPRequireMultipartName(class *AhdClass, name string) {
	if name == "" {
		AhdRaiseClass(class, "HTTP multipart field name must not be empty")
	}
	if len(name) > ahdHTTPMaxMultipartNameBytes {
		AhdRaiseClass(class, "HTTP multipart field name is too long")
	}
	if !utf8.ValidString(name) {
		AhdRaiseClass(class, "HTTP multipart field name is not valid UTF-8")
	}
	if strings.ContainsAny(name, "\r\n\"") || strings.IndexByte(name, 0) >= 0 {
		AhdRaiseClass(class, "HTTP multipart field name must not contain CR, LF, a quote, or a NUL byte")
	}
}

func ahdHTTPRequireMultipartContentType(class *AhdClass, contentType string) {
	if len(contentType) > ahdHTTPMaxMultipartTypeBytes {
		AhdRaiseClass(class, "HTTP multipart content type is too long")
	}
	if strings.ContainsAny(contentType, "\r\n") || strings.IndexByte(contentType, 0) >= 0 {
		AhdRaiseClass(class, "HTTP multipart content type must not contain CR, LF, or a NUL byte")
	}
	if _, _, err := mime.ParseMediaType(contentType); err != nil {
		AhdRaiseClass(class, "HTTP multipart content type "+ahdHTMLQuote(contentType)+" is not a valid media type")
	}
}

// ahdHTTPRequireMultipartFree refuses to add a multipart part to a request
// that already owns its body another way. AhdCode writes the multipart body
// and its Content-Type together, so an explicit body or an explicit
// Content-Type would have to be silently overwritten -- which is exactly what
// this refuses to do.
func ahdHTTPRequireMultipartFree(class *AhdClass, request ahdHTTPClientRequestData) {
	if request.HasBody {
		AhdRaiseClass(class, "HTTP request already has a body from withBody; a request uses either withBody or multipart parts, not both")
	}
	for _, header := range request.Headers {
		if textproto.CanonicalMIMEHeaderKey(header.Name) == "Content-Type" {
			AhdRaiseClass(class, "HTTP request already sets Content-Type; a multipart request's Content-Type and boundary are set by AhdCode")
		}
	}
}

// ahdHTTPRequireBodyFree is the same rule seen from the other side: withBody
// on a request that already carries multipart parts.
func ahdHTTPRequireBodyFree(class *AhdClass, request ahdHTTPClientRequestData) {
	if len(request.Multipart) > 0 {
		AhdRaiseClass(class, "HTTP request already has multipart parts; a request uses either withBody or multipart parts, not both")
	}
}

// ahdHTTPRequireContentTypeFree refuses a manual Content-Type on a request
// that carries multipart parts, so the header can never disagree with the
// boundary AhdCode generated.
func ahdHTTPRequireContentTypeFree(class *AhdClass, request ahdHTTPClientRequestData, name string) {
	if len(request.Multipart) == 0 {
		return
	}
	if textproto.CanonicalMIMEHeaderKey(name) == "Content-Type" {
		AhdRaiseClass(class, "HTTP request has multipart parts; its Content-Type and boundary are set by AhdCode")
	}
}

// --- streaming the multipart body ---

// ahdHTTPMultipartSource is everything one send needs to write the body: the
// configured parts and the boundary the Content-Type header names. The same
// boundary is reused for a redirect replay, so the replayed body still
// matches the header the transport carries over.
type ahdHTTPMultipartSource struct {
	parts    []ahdHTTPMultipartPart
	boundary string
}

// ahdHTTPPrepareMultipart validates every file part before a byte is sent:
// each path must be a readable regular file, and their total size must stay
// within ahdHTTPMaxUploadTotalBytes. Failing here means the request is never
// started, which is a much clearer failure than a body that dies mid-stream.
func ahdHTTPPrepareMultipart(class *AhdClass, parts []ahdHTTPMultipartPart) *ahdHTTPMultipartSource {
	var total int64
	for _, part := range parts {
		if !part.IsFile {
			continue
		}
		info, err := os.Stat(part.Path)
		if err != nil {
			if os.IsNotExist(err) {
				AhdRaiseClass(class, "HTTP multipart file does not exist")
			}
			AhdRaiseClass(class, "HTTP multipart file could not be read")
		}
		if info.IsDir() {
			AhdRaiseClass(class, "HTTP multipart file path is a directory")
		}
		if !info.Mode().IsRegular() {
			AhdRaiseClass(class, "HTTP multipart file path is not a regular file")
		}
		total += info.Size()
		if total > ahdHTTPMaxUploadTotalBytes {
			AhdRaiseClass(class, "HTTP multipart upload exceeds the total upload size limit")
		}
		file, err := os.Open(part.Path)
		if err != nil {
			AhdRaiseClass(class, "HTTP multipart file could not be opened")
		}
		_ = file.Close()
	}
	boundary, err := ahdHTTPUploadRandomHex(16)
	if err != nil {
		AhdRaiseClass(class, "HTTP multipart boundary could not be generated")
	}
	return &ahdHTTPMultipartSource{parts: parts, boundary: "ahdcode-" + boundary}
}

func (source *ahdHTTPMultipartSource) contentType() string {
	return "multipart/form-data; boundary=" + source.boundary
}

// body returns a reader the transport consumes. Writing happens in its own
// goroutine so nothing is buffered: each file is copied from disk into the
// pipe as the connection drains it. A failure closes the pipe with that
// error, which the transport reports as a failed request.
func (source *ahdHTTPMultipartSource) body() io.ReadCloser {
	reader, writer := io.Pipe()
	go func() {
		writer.CloseWithError(source.write(writer))
	}()
	return reader
}

func (source *ahdHTTPMultipartSource) write(sink io.Writer) error {
	form := multipart.NewWriter(sink)
	if err := form.SetBoundary(source.boundary); err != nil {
		return err
	}
	for _, part := range source.parts {
		if !part.IsFile {
			if err := form.WriteField(part.Name, part.Value); err != nil {
				return err
			}
			continue
		}
		if err := source.writeFile(form, part); err != nil {
			return err
		}
	}
	return form.Close()
}

func (source *ahdHTTPMultipartSource) writeFile(form *multipart.Writer, part ahdHTTPMultipartPart) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
		ahdHTTPMultipartEscape(part.Name), ahdHTTPMultipartEscape(part.FileName)))
	header.Set("Content-Type", part.ContentType)
	target, err := form.CreatePart(header)
	if err != nil {
		return err
	}
	file, err := os.Open(part.Path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	_, err = io.Copy(target, file)
	return err
}

// ahdHTTPMultipartEscape is the quoting mime/multipart itself uses for a
// Content-Disposition parameter. The configuration members already reject CR,
// LF, and a quote in a field name, and reduce a filename to a safe basename,
// so this is the last of several layers rather than the only one.
func ahdHTTPMultipartEscape(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", `"`, "\\\"", "\r", "", "\n", "")
	return replacer.Replace(value)
}

// --- Client.download and Client.sendToFile ---

// AhdHTTPClientDownload is Client.download(url, path): a GET whose response
// body is written to path.
func AhdHTTPClientDownload(class *AhdClass, handle, rawURL, path string) string {
	return AhdHTTPClientSendToFile(class, handle, AhdHTTPClientRequest(class, "GET", rawURL), path)
}

// AhdHTTPClientSendToFile is Client.sendToFile(request, path). The response
// body is streamed to path; only metadata comes back to AhdCode.
//
// An HTTP status is not a transport failure here any more than it is for
// Client.send: a 404's error page is written to the file and the 404 is
// reported through status(). Only a transport, TLS, limit, or filesystem
// failure raises HTTPError.
func AhdHTTPClientSendToFile(class *AhdClass, handle, requestData, path string) string {
	state := ahdHTTPLookupClient(class, handle)
	request := ahdHTTPDecodeClientRequest(class, requestData)
	ahdHTTPRequireClientURL(class, request.URL)
	destination := ahdHTTPRequireDestination(class, path)

	httpRequest, _ := ahdHTTPBuildClientRequest(class, request)
	response, err := state.http.Do(httpRequest)
	if err != nil {
		ahdHTTPRaiseClientFailure(class, err)
	}
	defer func() { _ = response.Body.Close() }()

	size := ahdHTTPStreamToFile(class, response.Body, state.maxResponseBytes, destination)

	headers := make([]ahdHTTPHeaderPair, 0)
	for name, values := range response.Header {
		for _, value := range values {
			headers = append(headers, ahdHTTPHeaderPair{Name: name, Value: value})
		}
	}
	finalURL := request.URL
	if response.Request != nil && response.Request.URL != nil {
		finalURL = response.Request.URL.String()
	}
	return ahdHTTPEncodeClientFileResponse(class, ahdHTTPClientFileResponseData{
		Status: response.StatusCode, Headers: headers, URL: finalURL, Size: size,
	})
}

// ahdHTTPRequireDestination checks the destination before the request is
// made. The parent directory must already exist -- AhdCode does not create
// directories behind a program's back -- and the destination itself must not
// be a directory.
func ahdHTTPRequireDestination(class *AhdClass, path string) string {
	if strings.TrimSpace(path) == "" {
		AhdRaiseClass(class, "HTTP download path must not be empty")
	}
	if strings.IndexByte(path, 0) >= 0 {
		AhdRaiseClass(class, "HTTP download path must not contain a NUL byte")
	}
	directory := filepath.Dir(path)
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		AhdRaiseClass(class, "HTTP download directory does not exist; create it before downloading into it")
	}
	if existing, err := os.Stat(path); err == nil && existing.IsDir() {
		AhdRaiseClass(class, "HTTP download path is a directory")
	}
	return path
}

// ahdHTTPStreamToFile copies the response body to a temporary file beside the
// destination and only then replaces the destination, so a failed transfer
// never leaves a half-written file where a program expects a whole one, and
// an existing file at that path survives untouched.
func ahdHTTPStreamToFile(class *AhdClass, body io.Reader, maxResponseBytes int64, destination string) int64 {
	directory := filepath.Dir(destination)
	temporary, err := os.CreateTemp(directory, ".ahdcode-download-*")
	if err != nil {
		AhdRaiseClass(class, "HTTP download could not create a temporary file: "+err.Error())
	}
	temporaryPath := temporary.Name()
	fail := func(message string) {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
		AhdRaiseClass(class, message)
	}

	// One byte past the limit is read so an oversize body is recognized
	// rather than silently truncated to the limit.
	written, copyErr := io.Copy(temporary, io.LimitReader(body, maxResponseBytes+1))
	if copyErr != nil {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
		ahdHTTPRaiseClientFailure(class, copyErr)
	}
	if written > maxResponseBytes {
		fail("HTTP response body exceeds maxResponseBytes")
	}
	if err := temporary.Sync(); err != nil {
		fail("HTTP download could not be written: " + err.Error())
	}
	if err := temporary.Close(); err != nil {
		fail("HTTP download could not be written: " + err.Error())
	}
	if err := os.Chmod(temporaryPath, 0o644); err != nil {
		_ = os.Remove(temporaryPath)
		AhdRaiseClass(class, "HTTP download could not be written: "+err.Error())
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		_ = os.Remove(temporaryPath)
		AhdRaiseClass(class, "HTTP download could not replace "+ahdHTMLQuote(destination)+": "+err.Error())
	}
	return written
}

// ahdHTTPBuildClientRequest turns one decoded ClientRequest into an
// *http.Request. It is shared by Client.send and Client.sendToFile so both
// carry a body, headers, and redirect replay the same way.
func ahdHTTPBuildClientRequest(class *AhdClass, request ahdHTTPClientRequestData) (*http.Request, *ahdHTTPMultipartSource) {
	var body io.Reader
	var source *ahdHTTPMultipartSource
	if len(request.Multipart) > 0 {
		source = ahdHTTPPrepareMultipart(class, request.Multipart)
		body = source.body()
	} else if request.Body != "" || request.Method == "POST" || request.Method == "PUT" || request.Method == "PATCH" {
		body = io.NopCloser(strings.NewReader(request.Body))
	}
	httpRequest, err := http.NewRequest(request.Method, request.URL, body)
	if err != nil {
		ahdHTTPRaiseClientFailure(class, err)
	}
	httpRequest.GetBody = nil
	switch {
	case source != nil:
		// The length is not known before the files are read, so the body is
		// sent chunked. A redirect that must replay the body reopens the same
		// files and rewrites the same boundary, so the replayed request is
		// byte for byte the one the Content-Type header describes.
		httpRequest.ContentLength = -1
		replay := source
		httpRequest.GetBody = func() (io.ReadCloser, error) { return replay.body(), nil }
	case body != nil:
		httpRequest.ContentLength = int64(len(request.Body))
	}
	httpRequest.Header = make(http.Header)
	for _, header := range request.Headers {
		httpRequest.Header.Add(header.Name, header.Value)
	}
	if source != nil {
		httpRequest.Header.Set("Content-Type", source.contentType())
	}
	httpRequest.Header.Del("Content-Length")
	return httpRequest, source
}

// --- ClientFileResponse members ---

func AhdHTTPClientFileResponseStatus(data string) int64 {
	return int64(ahdHTTPDecodeClientFileResponse(AhdClassHTTPError, data).Status)
}

func AhdHTTPClientFileResponseURL(class *AhdClass, data string) string {
	return ahdHTTPDecodeClientFileResponse(class, data).URL
}

func AhdHTTPClientFileResponseSize(class *AhdClass, data string) int64 {
	return ahdHTTPDecodeClientFileResponse(class, data).Size
}

func AhdHTTPClientFileResponseHeader(class *AhdClass, data, name string) *string {
	canonical := textproto.CanonicalMIMEHeaderKey(name)
	for _, header := range ahdHTTPDecodeClientFileResponse(class, data).Headers {
		if textproto.CanonicalMIMEHeaderKey(header.Name) == canonical {
			value := header.Value
			return &value
		}
	}
	return nil
}

func AhdHTTPClientFileResponseHeaderAll(class *AhdClass, data, name string) *AhdList[string] {
	canonical := textproto.CanonicalMIMEHeaderKey(name)
	var values []string
	for _, header := range ahdHTTPDecodeClientFileResponse(class, data).Headers {
		if textproto.CanonicalMIMEHeaderKey(header.Name) == canonical {
			values = append(values, header.Value)
		}
	}
	return ahdHTTPList(values)
}

func ahdHTTPEncodeClientFileResponse(class *AhdClass, response ahdHTTPClientFileResponseData) string {
	encoded, err := json.Marshal(response)
	if err != nil {
		AhdRaiseClass(class, "HTTP ClientFileResponse storage is corrupted")
	}
	return string(encoded)
}

func ahdHTTPDecodeClientFileResponse(class *AhdClass, data string) ahdHTTPClientFileResponseData {
	var response ahdHTTPClientFileResponseData
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		AhdRaiseClass(class, "HTTP ClientFileResponse storage is corrupted")
	}
	return response
}
