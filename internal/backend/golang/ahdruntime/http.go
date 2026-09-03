package ahdruntime

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

// HTTP server timeouts are internal v0.4.0 defaults. They are not a public
// configuration API. Local development stays usable; header reads are never
// left unlimited.
const (
	ahdHTTPReadHeaderTimeout = 10 * time.Second
	ahdHTTPReadTimeout       = 15 * time.Second
	ahdHTTPWriteTimeout      = 15 * time.Second
	ahdHTTPIdleTimeout       = 60 * time.Second
	ahdHTTPDefaultMaxBody    = 1048576
	ahdHTTPSessionIDBytes    = 32
	ahdHTTPDefaultCookiePath = "/"
)

type AhdHTTPHandler func(requestData string) string

type ahdHTTPRouteKey struct {
	method string
	path   string
}

type ahdHTTPServerState struct {
	mutex        sync.Mutex
	host         string
	port         int
	maxBodyBytes int64
	routes       map[ahdHTTPRouteKey]AhdHTTPHandler
	methods      map[string][]string
	started      bool
	httpServer   *http.Server
}

type ahdHTTPRequestData struct {
	Method    string              `json:"method"`
	Path      string              `json:"path"`
	Query     map[string][]string `json:"query"`
	Headers   map[string][]string `json:"headers"`
	Body      string              `json:"body"`
	BodyUTF8  bool                `json:"bodyUTF8"`
	Form      map[string][]string `json:"form"`
	FormOK    bool                `json:"formOK"`
	FormError string              `json:"formError,omitempty"`
	IsForm    bool                `json:"isForm"`
	// IsMultipart marks a multipart/form-data request. Its raw body is
	// legitimately binary, so the urlencoded "body must be UTF-8" rule does
	// not apply to it; each multipart text field is validated as UTF-8
	// individually while parsing instead.
	IsMultipart bool                 `json:"isMultipart,omitempty"`
	Cookies     []ahdHTTPCookieEntry `json:"cookies"`
	Files       []ahdHTTPUploadEntry `json:"files,omitempty"`
}

type ahdHTTPResponseData struct {
	Status      int                 `json:"status"`
	Body        string              `json:"body"`
	ContentType string              `json:"contentType"`
	Headers     []ahdHTTPHeaderPair `json:"headers"`
	Cookies     []ahdHTTPCookieData `json:"cookies"`
}

type ahdHTTPHeaderPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ahdHTTPCookieEntry struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ahdHTTPCookieData is the immutable Cookie value. MaxAge == nil means the
// attribute is omitted. MaxAge == 0 is the deletion encoding (Max-Age=0).
type ahdHTTPCookieData struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Path     string `json:"path"`
	HttpOnly bool   `json:"httpOnly"`
	Secure   bool   `json:"secure"`
	SameSite string `json:"sameSite"`
	MaxAge   *int64 `json:"maxAge"`
}

type ahdHTTPSessionData struct {
	StoreID   string            `json:"storeId"`
	ID        string            `json:"id"`
	Values    map[string]string `json:"values"`
	Dirty     bool              `json:"dirty"`
	Destroyed bool              `json:"destroyed"`
}

type ahdHTTPStoredSession struct {
	values    map[string]string
	expiresAt time.Time
}

type ahdHTTPSessionStoreState struct {
	mutex         sync.Mutex
	cookieName    string
	maxAgeSeconds int64
	secure        bool
	sameSite      string
	sessions      map[string]*ahdHTTPStoredSession
}

var (
	ahdHTTPSessionStores   = map[string]*ahdHTTPSessionStoreState{}
	ahdHTTPSessionStoresMu sync.Mutex
	ahdHTTPSessionNextID   atomic.Int64
	ahdHTTPTimeNow         = time.Now
)

var (
	ahdHTTPServers   = map[string]*ahdHTTPServerState{}
	ahdHTTPServersMu sync.Mutex
	ahdHTTPNextID    atomic.Int64
)

func AhdHTTPServer(class *AhdClass, host string, port int64, maxBodyBytes int64) string {
	if strings.TrimSpace(host) == "" {
		AhdRaiseClass(class, "HTTP host must not be empty")
	}
	if port < 1 || port > 65535 {
		AhdRaiseClass(class, "HTTP port must be in 1..65535")
	}
	if maxBodyBytes < 0 {
		AhdRaiseClass(class, "HTTP maxBodyBytes must not be negative")
	}
	id := strconv.FormatInt(ahdHTTPNextID.Add(1), 10)
	ahdHTTPServersMu.Lock()
	ahdHTTPServers[id] = &ahdHTTPServerState{
		host: host, port: int(port), maxBodyBytes: maxBodyBytes,
		routes:  make(map[ahdHTTPRouteKey]AhdHTTPHandler),
		methods: make(map[string][]string),
	}
	ahdHTTPServersMu.Unlock()
	return id
}

func AhdHTTPText(class *AhdClass, body string, status int64) string {
	return ahdHTTPEncodeResponse(class, ahdHTTPResponseData{
		Status: int(ahdHTTPRequireStatus(class, status)), Body: body,
		ContentType: "text/plain; charset=utf-8",
	})
}

func AhdHTTPHTML(class *AhdClass, body string, status int64) string {
	return ahdHTTPEncodeResponse(class, ahdHTTPResponseData{
		Status: int(ahdHTTPRequireStatus(class, status)), Body: body,
		ContentType: "text/html; charset=utf-8",
	})
}

func AhdHTTPResponse(class *AhdClass, status int64, body, contentType string) string {
	return ahdHTTPEncodeResponse(class, ahdHTTPResponseData{
		Status: int(ahdHTTPRequireStatus(class, status)), Body: body, ContentType: contentType,
	})
}

func AhdHTTPRedirect(class *AhdClass, location string, status int64) string {
	if status == 0 {
		status = 303
	}
	switch status {
	case 301, 302, 303, 307, 308:
	default:
		AhdRaiseClass(class, "HTTP redirect status must be 301, 302, 303, 307, or 308")
	}
	return ahdHTTPEncodeResponse(class, ahdHTTPResponseData{
		Status: int(status), Body: "", ContentType: "text/plain; charset=utf-8",
		Headers: []ahdHTTPHeaderPair{{Name: "Location", Value: location}},
	})
}

func AhdHTTPServerGet(class *AhdClass, handle, path string, handler AhdHTTPHandler) {
	ahdHTTPRegister(class, handle, "GET", path, handler)
}

func AhdHTTPServerPost(class *AhdClass, handle, path string, handler AhdHTTPHandler) {
	ahdHTTPRegister(class, handle, "POST", path, handler)
}

func AhdHTTPServerRoute(class *AhdClass, handle, method, path string, handler AhdHTTPHandler) {
	if !ahdHTTPMethodToken(method) {
		AhdRaiseClass(class, "HTTP method "+ahdHTMLQuote(method)+" is not a valid HTTP method token")
	}
	ahdHTTPRegister(class, handle, method, path, handler)
}

func AhdHTTPServerStart(class *AhdClass, handle string) {
	server := ahdHTTPLookup(class, handle)
	server.mutex.Lock()
	if server.started {
		server.mutex.Unlock()
		AhdRaiseClass(class, "HTTP server has already started")
	}
	server.started = true
	host := server.host
	port := server.port
	server.mutex.Unlock()

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		AhdRaiseClass(class, "HTTP server could not bind "+addr+": "+err.Error())
	}
	httpServer := &http.Server{
		Handler:           ahdHTTPDispatcher{class: class, state: server},
		ReadHeaderTimeout: ahdHTTPReadHeaderTimeout,
		ReadTimeout:       ahdHTTPReadTimeout,
		WriteTimeout:      ahdHTTPWriteTimeout,
		IdleTimeout:       ahdHTTPIdleTimeout,
	}
	server.mutex.Lock()
	server.httpServer = httpServer
	server.mutex.Unlock()
	err = httpServer.Serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		AhdRaiseClass(class, "HTTP server failed: "+err.Error())
	}
}

func ahdHTTPTestShutdown(handle string) {
	ahdHTTPServersMu.Lock()
	server := ahdHTTPServers[handle]
	ahdHTTPServersMu.Unlock()
	if server == nil {
		return
	}
	server.mutex.Lock()
	httpServer := server.httpServer
	server.mutex.Unlock()
	if httpServer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}

func AhdHTTPRequestMethod(data string) string { return ahdHTTPDecodeRequest(data).Method }
func AhdHTTPRequestPath(data string) string   { return ahdHTTPDecodeRequest(data).Path }

func AhdHTTPRequestQuery(class *AhdClass, data, name string) *string {
	return ahdHTTPFirst(ahdHTTPDecodeRequest(data).Query[name])
}
func AhdHTTPRequestQueryAll(class *AhdClass, data, name string) *AhdList[string] {
	return ahdHTTPList(ahdHTTPDecodeRequest(data).Query[name])
}
func AhdHTTPRequestHeader(class *AhdClass, data, name string) *string {
	return ahdHTTPFirst(ahdHTTPDecodeRequest(data).Headers[textproto.CanonicalMIMEHeaderKey(name)])
}
func AhdHTTPRequestHeaderAll(class *AhdClass, data, name string) *AhdList[string] {
	return ahdHTTPList(ahdHTTPDecodeRequest(data).Headers[textproto.CanonicalMIMEHeaderKey(name)])
}
func AhdHTTPRequestBody(class *AhdClass, data string) string {
	request := ahdHTTPDecodeRequest(data)
	if !request.BodyUTF8 {
		AhdRaiseClass(class, "HTTP request body is not valid UTF-8")
	}
	return request.Body
}
func AhdHTTPRequestForm(class *AhdClass, data, name string) *string {
	request := ahdHTTPRequireForm(class, data)
	return ahdHTTPFirst(request.Form[name])
}
func AhdHTTPRequestFormAll(class *AhdClass, data, name string) *AhdList[string] {
	request := ahdHTTPRequireForm(class, data)
	return ahdHTTPList(request.Form[name])
}

func AhdHTTPRequestCookie(class *AhdClass, data, name string) *string {
	request := ahdHTTPDecodeRequest(data)
	for _, cookie := range request.Cookies {
		if cookie.Name == name {
			value := cookie.Value
			return &value
		}
	}
	return nil
}

func AhdHTTPRequestCookieAll(class *AhdClass, data, name string) *AhdList[string] {
	request := ahdHTTPDecodeRequest(data)
	var values []string
	for _, cookie := range request.Cookies {
		if cookie.Name == name {
			values = append(values, cookie.Value)
		}
	}
	return ahdHTTPList(values)
}

func AhdHTTPResponseWithHeader(class *AhdClass, data, name, value string) string {
	if strings.EqualFold(name, "Set-Cookie") {
		AhdRaiseClass(class, "HTTP Set-Cookie must be set with Response.withCookie")
	}
	if !ahdHTTPHeaderNameOK(name) {
		AhdRaiseClass(class, "HTTP header name "+ahdHTMLQuote(name)+" is not valid")
	}
	if strings.ContainsAny(value, "\r\n") {
		AhdRaiseClass(class, "HTTP header value must not contain CR or LF")
	}
	response := ahdHTTPDecodeResponse(class, data)
	canonical := textproto.CanonicalMIMEHeaderKey(name)
	replaced := false
	headers := make([]ahdHTTPHeaderPair, 0, len(response.Headers)+1)
	for _, header := range response.Headers {
		if textproto.CanonicalMIMEHeaderKey(header.Name) == canonical {
			if !replaced {
				headers = append(headers, ahdHTTPHeaderPair{Name: name, Value: value})
				replaced = true
			}
			continue
		}
		headers = append(headers, header)
	}
	if !replaced {
		headers = append(headers, ahdHTTPHeaderPair{Name: name, Value: value})
	}
	response.Headers = headers
	return ahdHTTPEncodeResponse(class, response)
}

func AhdHTTPResponseWithCookie(class *AhdClass, data, cookieData string) string {
	response := ahdHTTPDecodeResponse(class, data)
	cookie := ahdHTTPDecodeCookie(class, cookieData)
	copied := make([]ahdHTTPCookieData, len(response.Cookies)+1)
	copy(copied, response.Cookies)
	copied[len(response.Cookies)] = cookie
	response.Cookies = copied
	return ahdHTTPEncodeResponse(class, response)
}

func ahdHTTPRegister(class *AhdClass, handle, method, path string, handler AhdHTTPHandler) {
	if handler == nil {
		AhdRaiseClass(class, "HTTP route handler is missing")
	}
	if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?#") {
		AhdRaiseClass(class, "HTTP route path "+ahdHTMLQuote(path)+" must begin with / and must not contain ? or #")
	}
	server := ahdHTTPLookup(class, handle)
	server.mutex.Lock()
	defer server.mutex.Unlock()
	if server.started {
		AhdRaiseClass(class, "HTTP routes cannot be changed after start")
	}
	key := ahdHTTPRouteKey{method: method, path: path}
	if _, exists := server.routes[key]; exists {
		AhdRaiseClass(class, "HTTP route "+method+" "+path+" is already registered")
	}
	server.routes[key] = handler
	if !ahdHTTPContains(server.methods[path], method) {
		server.methods[path] = append(server.methods[path], method)
	}
}

func ahdHTTPLookup(class *AhdClass, handle string) *ahdHTTPServerState {
	ahdHTTPServersMu.Lock()
	server := ahdHTTPServers[handle]
	ahdHTTPServersMu.Unlock()
	if server == nil {
		AhdRaiseClass(class, "HTTP Server storage is corrupted")
	}
	return server
}

type ahdHTTPDispatcher struct {
	class *AhdClass
	state *ahdHTTPServerState
}

func (dispatcher ahdHTTPDispatcher) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		ahdHTTPLogFailure(recovered)
		if !ahdHTTPHeaderWritten(writer) {
			ahdHTTPWritePlain(writer, http.StatusInternalServerError, "Internal Server Error")
		}
	}()

	path := request.URL.Path
	method := request.Method
	dispatcher.state.mutex.Lock()
	handler, found := dispatcher.state.routes[ahdHTTPRouteKey{method: method, path: path}]
	allowed := append([]string(nil), dispatcher.state.methods[path]...)
	maxBody := dispatcher.state.maxBodyBytes
	dispatcher.state.mutex.Unlock()

	if !found {
		if len(allowed) == 0 {
			ahdHTTPWritePlain(writer, http.StatusNotFound, "Not Found")
			return
		}
		writer.Header().Set("Allow", strings.Join(allowed, ", "))
		ahdHTTPWritePlain(writer, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	request.Body = http.MaxBytesReader(writer, request.Body, maxBody)
	body, err := io.ReadAll(request.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) || errors.Is(err, http.ErrBodyReadAfterClose) || ahdHTTPIsTooLarge(err) {
			ahdHTTPWritePlain(writer, http.StatusRequestEntityTooLarge, "Payload Too Large")
			return
		}
		ahdHTTPWritePlain(writer, http.StatusBadRequest, "Bad Request")
		return
	}

	snapshot, uploadIDs, err := ahdHTTPMaterialize(request, body)
	if err != nil {
		ahdHTTPWritePlain(writer, http.StatusBadRequest, "Bad Request")
		return
	}
	// Every upload this request registered is released once the handler is
	// done, on every path out: a normal response, a rejected upload, or a
	// panicking handler. Saved files have already moved out of the registry's
	// temporary storage and survive; everything else is deleted.
	defer ahdHTTPReleaseUploads(uploadIDs)
	var encoded string
	failed := false
	func() {
		dispatcher.state.mutex.Lock()
		defer dispatcher.state.mutex.Unlock()
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			failed = true
			ahdHTTPLogFailure(recovered)
		}()
		encoded = handler(ahdHTTPEncodeRequest(snapshot))
	}()
	if failed {
		if !ahdHTTPHeaderWritten(writer) {
			ahdHTTPWritePlain(writer, http.StatusInternalServerError, "Internal Server Error")
		}
		return
	}
	ahdHTTPWriteEncoded(writer, encoded)
}

// ahdHTTPMaterialize snapshots one request. The returned id slice names every
// uploaded file registered for this request; the caller must release it once
// the handler has finished, so unsaved temporary files never outlive the
// request that produced them.
func ahdHTTPMaterialize(request *http.Request, body []byte) (ahdHTTPRequestData, []string, error) {
	var uploadIDs []string
	rawQuery := ""
	path := ""
	if request.URL != nil {
		rawQuery = request.URL.RawQuery
		path = request.URL.Path
	}
	query, err := ahdHTTPParseEncoded(rawQuery)
	if err != nil {
		return ahdHTTPRequestData{}, nil, err
	}
	headers := map[string][]string{}
	for name, values := range request.Header {
		headers[textproto.CanonicalMIMEHeaderKey(name)] = append([]string(nil), values...)
	}
	snapshot := ahdHTTPRequestData{
		Method: request.Method, Path: path,
		Query: query, Headers: headers, BodyUTF8: utf8.Valid(body),
	}
	if snapshot.BodyUTF8 {
		snapshot.Body = string(body)
	}
	for _, cookie := range request.Cookies() {
		snapshot.Cookies = append(snapshot.Cookies, ahdHTTPCookieEntry{Name: cookie.Name, Value: cookie.Value})
	}
	mediaType, parameters, _ := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if mediaType == "application/x-www-form-urlencoded" {
		snapshot.IsForm = true
		if !snapshot.BodyUTF8 {
			return ahdHTTPRequestData{}, nil, errHTTPEncodedUTF8
		}
		values, err := ahdHTTPParseEncoded(snapshot.Body)
		if err != nil {
			return ahdHTTPRequestData{}, nil, err
		}
		snapshot.Form = values
		snapshot.FormOK = true
	}
	if mediaType == "multipart/form-data" {
		boundary := parameters["boundary"]
		if boundary == "" {
			return ahdHTTPRequestData{}, nil, errHTTPMultipartBoundary
		}
		ids, err := ahdHTTPParseMultipart(body, boundary, &snapshot)
		if err != nil {
			return ahdHTTPRequestData{}, nil, err
		}
		uploadIDs = ids
	}
	return snapshot, uploadIDs, nil
}

var errHTTPMultipartBoundary = errors.New("multipart/form-data request has no boundary")

var errHTTPEncodedUTF8 = errors.New("HTTP encoded parameter is not valid UTF-8")

// ahdHTTPParseEncoded strictly decodes application/x-www-form-urlencoded data.
// It does not use URL.Query's error-discarding path. Malformed percent
// escapes and percent-decoded invalid UTF-8 are errors; valid UTF-8, '+',
// '%20', and duplicate keys are preserved in order.
func ahdHTTPParseEncoded(raw string) (map[string][]string, error) {
	parsed, err := url.ParseQuery(raw)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string, len(parsed))
	for key, values := range parsed {
		if !utf8.ValidString(key) {
			return nil, errHTTPEncodedUTF8
		}
		copied := make([]string, len(values))
		for index, value := range values {
			if !utf8.ValidString(value) {
				return nil, errHTTPEncodedUTF8
			}
			copied[index] = value
		}
		out[key] = copied
	}
	return out, nil
}

func ahdHTTPRequireForm(class *AhdClass, data string) ahdHTTPRequestData {
	request := ahdHTTPDecodeRequest(data)
	if request.IsForm && request.FormError != "" {
		AhdRaiseClass(class, request.FormError)
	}
	if !request.BodyUTF8 && request.IsForm && !request.IsMultipart {
		AhdRaiseClass(class, "HTTP request body is not valid UTF-8")
	}
	return request
}

func ahdHTTPRequireStatus(class *AhdClass, status int64) int64 {
	if status < 100 || status > 599 {
		AhdRaiseClass(class, "HTTP status must be in 100..599")
	}
	return status
}

func ahdHTTPMethodToken(method string) bool {
	if method == "" {
		return false
	}
	for i := 0; i < len(method); i++ {
		c := method[i]
		if !ahdHTTPTchar(c) {
			return false
		}
	}
	return true
}

func ahdHTTPTchar(c byte) bool {
	if c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
		return true
	}
	switch c {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	}
	return false
}

func ahdHTTPHeaderNameOK(name string) bool {
	return ahdHTTPMethodToken(name)
}

func ahdHTTPFirst(values []string) *string {
	if len(values) == 0 {
		return nil
	}
	value := values[0]
	return &value
}

func ahdHTTPList(values []string) *AhdList[string] {
	if len(values) == 0 {
		return AhdNewList[string]()
	}
	return AhdNewList(values...)
}

func ahdHTTPContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func ahdHTTPEncodeRequest(request ahdHTTPRequestData) string {
	encoded, _ := json.Marshal(request)
	return string(encoded)
}

func ahdHTTPDecodeRequest(data string) ahdHTTPRequestData {
	var request ahdHTTPRequestData
	_ = json.Unmarshal([]byte(data), &request)
	if request.Query == nil {
		request.Query = map[string][]string{}
	}
	if request.Headers == nil {
		request.Headers = map[string][]string{}
	}
	if request.Form == nil {
		request.Form = map[string][]string{}
	}
	if request.Cookies == nil {
		request.Cookies = []ahdHTTPCookieEntry{}
	}
	return request
}

func ahdHTTPEncodeResponse(class *AhdClass, response ahdHTTPResponseData) string {
	encoded, err := json.Marshal(response)
	if err != nil {
		AhdRaiseClass(class, "HTTP response could not be encoded")
	}
	return string(encoded)
}

func ahdHTTPDecodeResponse(class *AhdClass, data string) ahdHTTPResponseData {
	var response ahdHTTPResponseData
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		AhdRaiseClass(class, "HTTP Response storage is corrupted")
	}
	return response
}

func ahdHTTPWriteEncoded(writer http.ResponseWriter, data string) {
	var response ahdHTTPResponseData
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		ahdHTTPWritePlain(writer, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if response.ContentType != "" {
		writer.Header().Set("Content-Type", response.ContentType)
	}
	for _, header := range response.Headers {
		writer.Header().Set(header.Name, header.Value)
	}
	for _, cookie := range response.Cookies {
		writer.Header().Add("Set-Cookie", ahdHTTPCookieWire(cookie))
	}
	status := response.Status
	if status == 0 {
		status = http.StatusOK
	}
	writer.WriteHeader(status)
	_, _ = io.WriteString(writer, response.Body)
}

func ahdHTTPWritePlain(writer http.ResponseWriter, status int, body string) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(status)
	_, _ = io.WriteString(writer, body)
}

func ahdHTTPHeaderWritten(writer http.ResponseWriter) bool {
	if tracker, ok := writer.(interface{ Written() bool }); ok {
		return tracker.Written()
	}
	return false
}

func ahdHTTPIsTooLarge(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	return strings.Contains(text, "http: request body too large") || strings.Contains(text, "MaxBytesError")
}

func ahdHTTPLogFailure(recovered any) {
	if signal, ok := recovered.(*AhdSignal); ok && signal != nil {
		name := "Error"
		if signal.Instance != nil && signal.Instance.AhdClassOf() != nil {
			name = signal.Instance.AhdClassOf().Name
		}
		fmt.Fprintf(os.Stderr, "ahdcode: HTTP handler %s: %s\n", name, signal.Message)
		return
	}
	fmt.Fprintf(os.Stderr, "ahdcode: HTTP handler panic: %v\n", recovered)
}

func AhdHTTPCookie(class *AhdClass, name, value string) string {
	return ahdHTTPEncodeCookie(class, ahdHTTPRequireCookie(class, ahdHTTPCookieData{
		Name: name, Value: value, Path: ahdHTTPDefaultCookiePath, SameSite: "Lax",
	}))
}

func AhdHTTPDeleteCookie(class *AhdClass, name, path string) string {
	if path == "" {
		path = ahdHTTPDefaultCookiePath
	}
	zero := int64(0)
	return ahdHTTPEncodeCookie(class, ahdHTTPRequireCookie(class, ahdHTTPCookieData{
		Name: name, Value: "", Path: path, SameSite: "Lax", MaxAge: &zero,
	}))
}

func AhdHTTPCookieWithPath(class *AhdClass, data, path string) string {
	cookie := ahdHTTPDecodeCookie(class, data)
	cookie.Path = path
	return ahdHTTPEncodeCookie(class, ahdHTTPRequireCookie(class, cookie))
}

func AhdHTTPCookieWithHttpOnly(class *AhdClass, data string, value bool) string {
	cookie := ahdHTTPDecodeCookie(class, data)
	cookie.HttpOnly = value
	return ahdHTTPEncodeCookie(class, ahdHTTPRequireCookie(class, cookie))
}

func AhdHTTPCookieWithSecure(class *AhdClass, data string, value bool) string {
	cookie := ahdHTTPDecodeCookie(class, data)
	cookie.Secure = value
	return ahdHTTPEncodeCookie(class, ahdHTTPRequireCookie(class, cookie))
}

func AhdHTTPCookieWithSameSite(class *AhdClass, data, mode string) string {
	cookie := ahdHTTPDecodeCookie(class, data)
	cookie.SameSite = mode
	return ahdHTTPEncodeCookie(class, ahdHTTPRequireCookie(class, cookie))
}

func AhdHTTPCookieWithMaxAge(class *AhdClass, data string, seconds int64) string {
	if seconds < 0 {
		AhdRaiseClass(class, "HTTP cookie Max-Age must not be negative")
	}
	cookie := ahdHTTPDecodeCookie(class, data)
	cookie.MaxAge = &seconds
	return ahdHTTPEncodeCookie(class, ahdHTTPRequireCookie(class, cookie))
}

func AhdHTTPSessions(class *AhdClass, cookieName string, maxAgeSeconds int64, secure bool, sameSite string) string {
	ahdHTTPRequireCookieName(class, cookieName)
	if maxAgeSeconds <= 0 {
		AhdRaiseClass(class, "HTTP session maxAgeSeconds must be greater than 0")
	}
	ahdHTTPRequireSameSite(class, sameSite, secure)
	id := strconv.FormatInt(ahdHTTPSessionNextID.Add(1), 10)
	ahdHTTPSessionStoresMu.Lock()
	ahdHTTPSessionStores[id] = &ahdHTTPSessionStoreState{
		cookieName: cookieName, maxAgeSeconds: maxAgeSeconds, secure: secure, sameSite: sameSite,
		sessions: make(map[string]*ahdHTTPStoredSession),
	}
	ahdHTTPSessionStoresMu.Unlock()
	return id
}

func AhdHTTPSessionStoreOpen(class *AhdClass, storeHandle, requestData string) string {
	store := ahdHTTPLookupSessionStore(class, storeHandle)
	request := ahdHTTPDecodeRequest(requestData)
	now := ahdHTTPTimeNow()
	session := ahdHTTPSessionData{StoreID: storeHandle, Values: map[string]string{}}
	store.mutex.Lock()
	store.cleanupExpiredLocked(now)
	for _, cookie := range request.Cookies {
		if cookie.Name != store.cookieName {
			continue
		}
		if stored := store.sessions[cookie.Value]; stored != nil && stored.expiresAt.After(now) {
			session.ID = cookie.Value
			session.Values = ahdHTTPCopyStringMap(stored.values)
		}
		break
	}
	store.mutex.Unlock()
	return ahdHTTPEncodeSession(class, session)
}

func AhdHTTPSessionStoreCommit(class *AhdClass, storeHandle, sessionData, responseData string) string {
	store := ahdHTTPLookupSessionStore(class, storeHandle)
	session := ahdHTTPDecodeSession(class, sessionData)
	if session.StoreID != storeHandle {
		AhdRaiseClass(class, "HTTP session does not belong to this SessionStore")
	}
	now := ahdHTTPTimeNow()
	if session.Destroyed {
		store.mutex.Lock()
		store.cleanupExpiredLocked(now)
		if session.ID != "" {
			delete(store.sessions, session.ID)
		}
		store.mutex.Unlock()
		return AhdHTTPResponseWithCookie(class, responseData, AhdHTTPDeleteCookie(class, store.cookieName, ahdHTTPDefaultCookiePath))
	}
	if !session.Dirty {
		return responseData
	}
	if session.ID == "" {
		session.ID = ahdHTTPNewSessionID(class)
	}
	store.mutex.Lock()
	store.cleanupExpiredLocked(now)
	store.sessions[session.ID] = &ahdHTTPStoredSession{
		values: ahdHTTPCopyStringMap(session.Values), expiresAt: now.Add(time.Duration(store.maxAgeSeconds) * time.Second),
	}
	cookieName := store.cookieName
	maxAge := store.maxAgeSeconds
	secure := store.secure
	sameSite := store.sameSite
	store.mutex.Unlock()
	cookie := ahdHTTPRequireCookie(class, ahdHTTPCookieData{
		Name: cookieName, Value: session.ID, Path: ahdHTTPDefaultCookiePath,
		HttpOnly: true, Secure: secure, SameSite: sameSite, MaxAge: &maxAge,
	})
	response := ahdHTTPDecodeResponse(class, responseData)
	response.Cookies = append(append([]ahdHTTPCookieData(nil), response.Cookies...), cookie)
	return ahdHTTPEncodeResponse(class, response)
}

func AhdHTTPSessionGet(class *AhdClass, data, name string) *string {
	session := ahdHTTPDecodeSession(class, data)
	if session.Destroyed {
		return nil
	}
	value, ok := session.Values[name]
	if !ok {
		return nil
	}
	copied := value
	return &copied
}

func AhdHTTPSessionHas(class *AhdClass, data, name string) bool {
	session := ahdHTTPDecodeSession(class, data)
	if session.Destroyed {
		return false
	}
	_, ok := session.Values[name]
	return ok
}

func AhdHTTPSessionSet(class *AhdClass, data, name, value string) string {
	session := ahdHTTPRequireMutableSession(class, data)
	ahdHTTPRequireSessionKey(class, name)
	if session.Values == nil {
		session.Values = map[string]string{}
	}
	session.Values[name] = value
	session.Dirty = true
	return ahdHTTPEncodeSession(class, session)
}

func AhdHTTPSessionRemove(class *AhdClass, data, name string) string {
	session := ahdHTTPRequireMutableSession(class, data)
	ahdHTTPRequireSessionKey(class, name)
	if _, exists := session.Values[name]; exists {
		delete(session.Values, name)
		session.Dirty = true
	}
	return ahdHTTPEncodeSession(class, session)
}

func AhdHTTPSessionClear(class *AhdClass, data string) string {
	session := ahdHTTPRequireMutableSession(class, data)
	if len(session.Values) != 0 {
		session.Values = map[string]string{}
		session.Dirty = true
	}
	return ahdHTTPEncodeSession(class, session)
}

func AhdHTTPSessionRotate(class *AhdClass, data string) string {
	session := ahdHTTPRequireMutableSession(class, data)
	store := ahdHTTPLookupSessionStore(class, session.StoreID)
	oldID := session.ID
	session.ID = ahdHTTPNewSessionID(class)
	session.Dirty = true
	if oldID != "" {
		store.mutex.Lock()
		delete(store.sessions, oldID)
		store.mutex.Unlock()
	}
	return ahdHTTPEncodeSession(class, session)
}

func AhdHTTPSessionDestroy(class *AhdClass, data string) string {
	session := ahdHTTPRequireMutableSession(class, data)
	store := ahdHTTPLookupSessionStore(class, session.StoreID)
	oldID := session.ID
	session.Destroyed = true
	session.Dirty = false
	session.Values = map[string]string{}
	if oldID != "" {
		store.mutex.Lock()
		delete(store.sessions, oldID)
		store.mutex.Unlock()
	}
	return ahdHTTPEncodeSession(class, session)
}

func ahdHTTPRequireCookie(class *AhdClass, cookie ahdHTTPCookieData) ahdHTTPCookieData {
	ahdHTTPRequireCookieName(class, cookie.Name)
	ahdHTTPRequireCookieValue(class, cookie.Value)
	ahdHTTPRequireCookiePath(class, cookie.Path)
	if cookie.MaxAge != nil && *cookie.MaxAge < 0 {
		AhdRaiseClass(class, "HTTP cookie Max-Age must not be negative")
	}
	ahdHTTPRequireSameSite(class, cookie.SameSite, cookie.Secure)
	return cookie
}

func ahdHTTPRequireCookieName(class *AhdClass, name string) {
	if !ahdHTTPCookieNameOK(name) {
		AhdRaiseClass(class, "HTTP cookie name "+ahdHTMLQuote(name)+" is not valid")
	}
}

func ahdHTTPRequireCookieValue(class *AhdClass, value string) {
	if !ahdHTTPCookieValueOK(value) {
		AhdRaiseClass(class, "HTTP cookie value is not a cookie-octet string")
	}
}

func ahdHTTPRequireCookiePath(class *AhdClass, path string) {
	if path == "" || !ahdHTTPCookiePathOK(path) {
		AhdRaiseClass(class, "HTTP cookie path "+ahdHTMLQuote(path)+" is not valid")
	}
}

func ahdHTTPRequireSameSite(class *AhdClass, mode string, secure bool) {
	switch mode {
	case "Lax", "Strict":
	case "None":
		if !secure {
			AhdRaiseClass(class, `HTTP SameSite "None" requires Secure=true`)
		}
	default:
		AhdRaiseClass(class, "HTTP SameSite must be Lax, Strict, or None")
	}
}

func ahdHTTPRequireSessionKey(class *AhdClass, name string) {
	if name == "" {
		AhdRaiseClass(class, "HTTP session key must not be empty")
	}
	for i := 0; i < len(name); i++ {
		if name[i] < 0x20 || name[i] == 0x7f {
			AhdRaiseClass(class, "HTTP session key must not contain control characters")
		}
	}
}

func ahdHTTPRequireMutableSession(class *AhdClass, data string) ahdHTTPSessionData {
	session := ahdHTTPDecodeSession(class, data)
	if session.Destroyed {
		AhdRaiseClass(class, "HTTP session has been destroyed")
	}
	if session.Values == nil {
		session.Values = map[string]string{}
	}
	return session
}

func ahdHTTPLookupSessionStore(class *AhdClass, handle string) *ahdHTTPSessionStoreState {
	ahdHTTPSessionStoresMu.Lock()
	store := ahdHTTPSessionStores[handle]
	ahdHTTPSessionStoresMu.Unlock()
	if store == nil {
		AhdRaiseClass(class, "HTTP SessionStore storage is corrupted")
	}
	return store
}

func (store *ahdHTTPSessionStoreState) cleanupExpiredLocked(now time.Time) {
	for id, session := range store.sessions {
		if !session.expiresAt.After(now) {
			delete(store.sessions, id)
		}
	}
}

func ahdHTTPNewSessionID(class *AhdClass) string {
	var raw [ahdHTTPSessionIDBytes]byte
	if _, err := rand.Read(raw[:]); err != nil {
		AhdRaiseClass(class, "HTTP session identifier could not be generated")
	}
	return base64.RawURLEncoding.EncodeToString(raw[:])
}

func ahdHTTPCookieNameOK(name string) bool {
	return ahdHTTPMethodToken(name)
}

func ahdHTTPCookieValueOK(value string) bool {
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c < 0x21 || c > 0x7e || c == '"' || c == ',' || c == ';' || c == '\\' {
			return false
		}
	}
	return true
}

func ahdHTTPCookiePathOK(path string) bool {
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c < 0x20 || c > 0x7e || c == ';' {
			return false
		}
	}
	return true
}

func ahdHTTPCookieWire(cookie ahdHTTPCookieData) string {
	item := http.Cookie{
		Name: cookie.Name, Value: cookie.Value, Path: cookie.Path,
		HttpOnly: cookie.HttpOnly, Secure: cookie.Secure,
	}
	switch cookie.SameSite {
	case "Strict":
		item.SameSite = http.SameSiteStrictMode
	case "None":
		item.SameSite = http.SameSiteNoneMode
	default:
		item.SameSite = http.SameSiteLaxMode
	}
	if cookie.MaxAge != nil {
		if *cookie.MaxAge == 0 {
			item.MaxAge = -1
			item.Expires = time.Unix(0, 0).UTC()
		} else {
			item.MaxAge = int(*cookie.MaxAge)
		}
	}
	return item.String()
}

func ahdHTTPEncodeCookie(class *AhdClass, cookie ahdHTTPCookieData) string {
	encoded, err := json.Marshal(cookie)
	if err != nil {
		AhdRaiseClass(class, "HTTP cookie could not be encoded")
	}
	return string(encoded)
}

func ahdHTTPDecodeCookie(class *AhdClass, data string) ahdHTTPCookieData {
	var cookie ahdHTTPCookieData
	if err := json.Unmarshal([]byte(data), &cookie); err != nil {
		AhdRaiseClass(class, "HTTP Cookie storage is corrupted")
	}
	return cookie
}

func ahdHTTPEncodeSession(class *AhdClass, session ahdHTTPSessionData) string {
	if session.Values == nil {
		session.Values = map[string]string{}
	}
	encoded, err := json.Marshal(session)
	if err != nil {
		AhdRaiseClass(class, "HTTP session could not be encoded")
	}
	return string(encoded)
}

func ahdHTTPDecodeSession(class *AhdClass, data string) ahdHTTPSessionData {
	var session ahdHTTPSessionData
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		AhdRaiseClass(class, "HTTP Session storage is corrupted")
	}
	if session.Values == nil {
		session.Values = map[string]string{}
	}
	return session
}

func ahdHTTPCopyStringMap(values map[string]string) map[string]string {
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func ahdHTTPTestStoreSize(handle string) int {
	ahdHTTPSessionStoresMu.Lock()
	store := ahdHTTPSessionStores[handle]
	ahdHTTPSessionStoresMu.Unlock()
	if store == nil {
		return 0
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return len(store.sessions)
}

func ahdHTTPTestResetClock() {
	ahdHTTPTimeNow = time.Now
}

const (
	ahdHTTPDefaultClientTimeout = 30
	ahdHTTPDefaultClientMaxBody = 8388608
	ahdHTTPMaxRedirects         = 10
	// ahdHTTPMaxClientTimeoutSeconds is the largest whole-second timeout that
	// still fits in a time.Duration after conversion to nanoseconds.
	ahdHTTPMaxClientTimeoutSeconds = 9223372036
	// ahdHTTPMaxClientResponseBytes leaves room for the bounded read of
	// maxResponseBytes + 1 used to detect an oversized body.
	ahdHTTPMaxClientResponseBytes = 9223372036854775806
)

var (
	errHTTPTooManyRedirects = errors.New("too many redirects")
	errHTTPHTTPSDowngrade   = errors.New("https to http redirect")
)

type ahdHTTPClientRequestData struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers []ahdHTTPHeaderPair `json:"headers"`
	Body    string              `json:"body"`
}

type ahdHTTPClientResponseData struct {
	Status  int                 `json:"status"`
	Body    string              `json:"body"`
	Headers []ahdHTTPHeaderPair `json:"headers"`
	URL     string              `json:"url"`
}

type ahdHTTPClientState struct {
	timeout          time.Duration
	maxResponseBytes int64
	followRedirects  bool
	http             *http.Client
}

var (
	ahdHTTPClients      = map[string]*ahdHTTPClientState{}
	ahdHTTPClientsMu    sync.Mutex
	ahdHTTPClientNextID atomic.Int64
)

func AhdHTTPClient(class *AhdClass, timeoutSeconds, maxResponseBytes int64, followRedirects bool) string {
	if timeoutSeconds < 1 || timeoutSeconds > ahdHTTPMaxClientTimeoutSeconds {
		AhdRaiseClass(class, "HTTP client timeoutSeconds must be between 1 and 9223372036")
	}
	if maxResponseBytes < 1 || maxResponseBytes > ahdHTTPMaxClientResponseBytes {
		AhdRaiseClass(class, "HTTP client maxResponseBytes must be between 1 and 9223372036854775806")
	}
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		AhdRaiseClass(class, "HTTP client transport is unavailable")
	}
	transport := base.Clone()
	state := &ahdHTTPClientState{
		timeout:          time.Duration(timeoutSeconds) * time.Second,
		maxResponseBytes: maxResponseBytes,
		followRedirects:  followRedirects,
	}
	state.http = &http.Client{
		Timeout:   state.timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return ahdHTTPClientRedirect(state, req, via)
		},
	}
	id := strconv.FormatInt(ahdHTTPClientNextID.Add(1), 10)
	ahdHTTPClientsMu.Lock()
	ahdHTTPClients[id] = state
	ahdHTTPClientsMu.Unlock()
	return id
}

func ahdHTTPClientRedirect(state *ahdHTTPClientState, req *http.Request, via []*http.Request) error {
	if !state.followRedirects {
		return http.ErrUseLastResponse
	}
	if len(via) >= ahdHTTPMaxRedirects {
		return errHTTPTooManyRedirects
	}
	previous := via[len(via)-1]
	if previous.URL.Scheme == "https" && req.URL.Scheme == "http" {
		return errHTTPHTTPSDowngrade
	}
	if !ahdHTTPSameRedirectHost(previous.URL, req.URL) {
		req.Header.Del("Authorization")
		req.Header.Del("Cookie")
	}
	return nil
}

func ahdHTTPSameRedirectHost(from, to *url.URL) bool {
	return strings.EqualFold(from.Hostname(), to.Hostname()) && from.Port() == to.Port()
}

func AhdHTTPClientRequest(class *AhdClass, method, rawURL string) string {
	if !ahdHTTPMethodToken(method) {
		AhdRaiseClass(class, "HTTP client method "+ahdHTMLQuote(method)+" is not valid")
	}
	ahdHTTPRequireClientURL(class, rawURL)
	return ahdHTTPEncodeClientRequest(class, ahdHTTPClientRequestData{Method: method, URL: rawURL})
}

func AhdHTTPClientRequestWithHeader(class *AhdClass, data, name, value string) string {
	ahdHTTPRequireClientHeader(class, name, value)
	request := ahdHTTPDecodeClientRequest(class, data)
	canonical := textproto.CanonicalMIMEHeaderKey(name)
	replaced := false
	headers := make([]ahdHTTPHeaderPair, 0, len(request.Headers)+1)
	for _, header := range request.Headers {
		if textproto.CanonicalMIMEHeaderKey(header.Name) == canonical {
			if !replaced {
				headers = append(headers, ahdHTTPHeaderPair{Name: name, Value: value})
				replaced = true
			}
			continue
		}
		headers = append(headers, header)
	}
	if !replaced {
		headers = append(headers, ahdHTTPHeaderPair{Name: name, Value: value})
	}
	request.Headers = headers
	return ahdHTTPEncodeClientRequest(class, request)
}

func AhdHTTPClientRequestAddHeader(class *AhdClass, data, name, value string) string {
	ahdHTTPRequireClientHeader(class, name, value)
	request := ahdHTTPDecodeClientRequest(class, data)
	request.Headers = append(append([]ahdHTTPHeaderPair(nil), request.Headers...), ahdHTTPHeaderPair{Name: name, Value: value})
	return ahdHTTPEncodeClientRequest(class, request)
}

func AhdHTTPClientRequestWithBody(class *AhdClass, data, body string) string {
	request := ahdHTTPDecodeClientRequest(class, data)
	request.Body = body
	return ahdHTTPEncodeClientRequest(class, request)
}

func AhdHTTPClientSend(class *AhdClass, handle, requestData string) string {
	state := ahdHTTPLookupClient(class, handle)
	request := ahdHTTPDecodeClientRequest(class, requestData)
	ahdHTTPRequireClientURL(class, request.URL)
	var body io.Reader
	if request.Body != "" || request.Method == "POST" || request.Method == "PUT" || request.Method == "PATCH" {
		body = io.NopCloser(strings.NewReader(request.Body))
	}
	httpRequest, err := http.NewRequest(request.Method, request.URL, body)
	if err != nil {
		ahdHTTPRaiseClientFailure(class, err)
	}
	httpRequest.GetBody = nil
	if body != nil {
		httpRequest.ContentLength = int64(len(request.Body))
	}
	httpRequest.Header = make(http.Header)
	for _, header := range request.Headers {
		httpRequest.Header.Add(header.Name, header.Value)
	}
	httpRequest.Header.Del("Content-Length")
	response, err := state.http.Do(httpRequest)
	if err != nil {
		ahdHTTPRaiseClientFailure(class, err)
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, state.maxResponseBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		ahdHTTPRaiseClientFailure(class, err)
	}
	if int64(len(raw)) > state.maxResponseBytes {
		AhdRaiseClass(class, "HTTP response body exceeds maxResponseBytes")
	}
	if !utf8.Valid(raw) {
		AhdRaiseClass(class, "HTTP response body is not valid UTF-8")
	}
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
	return ahdHTTPEncodeClientResponse(class, ahdHTTPClientResponseData{
		Status: response.StatusCode, Body: string(raw), Headers: headers, URL: finalURL,
	})
}

func AhdHTTPClientGet(class *AhdClass, handle, rawURL string) string {
	return AhdHTTPClientSend(class, handle, AhdHTTPClientRequest(class, "GET", rawURL))
}

func AhdHTTPClientPost(class *AhdClass, handle, rawURL, body, contentType string) string {
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	request := AhdHTTPClientRequest(class, "POST", rawURL)
	request = AhdHTTPClientRequestWithHeader(class, request, "Content-Type", contentType)
	request = AhdHTTPClientRequestWithBody(class, request, body)
	return AhdHTTPClientSend(class, handle, request)
}

func AhdHTTPClientResponseStatus(data string) int64 {
	return int64(ahdHTTPDecodeClientResponse(AhdClassHTTPError, data).Status)
}

func AhdHTTPClientResponseBody(class *AhdClass, data string) string {
	return ahdHTTPDecodeClientResponse(class, data).Body
}

func AhdHTTPClientResponseURL(class *AhdClass, data string) string {
	return ahdHTTPDecodeClientResponse(class, data).URL
}

func AhdHTTPClientResponseHeader(class *AhdClass, data, name string) *string {
	canonical := textproto.CanonicalMIMEHeaderKey(name)
	for _, header := range ahdHTTPDecodeClientResponse(class, data).Headers {
		if textproto.CanonicalMIMEHeaderKey(header.Name) == canonical {
			value := header.Value
			return &value
		}
	}
	return nil
}

func AhdHTTPClientResponseHeaderAll(class *AhdClass, data, name string) *AhdList[string] {
	canonical := textproto.CanonicalMIMEHeaderKey(name)
	var values []string
	for _, header := range ahdHTTPDecodeClientResponse(class, data).Headers {
		if textproto.CanonicalMIMEHeaderKey(header.Name) == canonical {
			values = append(values, header.Value)
		}
	}
	return ahdHTTPList(values)
}

func ahdHTTPRequireClientURL(class *AhdClass, raw string) *url.URL {
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		AhdRaiseClass(class, "HTTP client URL must be an absolute http or https URL with a host")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		AhdRaiseClass(class, "HTTP client URL scheme must be http or https")
	}
	if parsed.Fragment != "" || strings.Contains(raw, "#") {
		AhdRaiseClass(class, "HTTP client URL must not contain a fragment")
	}
	if parsed.User != nil {
		AhdRaiseClass(class, "HTTP client URL must not contain userinfo; use an Authorization header")
	}
	if parsed.Hostname() == "" {
		AhdRaiseClass(class, "HTTP client URL must include a host")
	}
	return parsed
}

func ahdHTTPRequireClientHeader(class *AhdClass, name, value string) {
	if !ahdHTTPHeaderNameOK(name) {
		AhdRaiseClass(class, "HTTP header name "+ahdHTMLQuote(name)+" is not valid")
	}
	if strings.ContainsAny(value, "\r\n") {
		AhdRaiseClass(class, "HTTP header value must not contain CR or LF")
	}
	if strings.EqualFold(name, "Content-Length") {
		AhdRaiseClass(class, "HTTP Content-Length is set from the request body")
	}
	if strings.EqualFold(name, "Host") {
		AhdRaiseClass(class, "HTTP Host is taken from the request URL")
	}
}

func ahdHTTPLookupClient(class *AhdClass, handle string) *ahdHTTPClientState {
	ahdHTTPClientsMu.Lock()
	state := ahdHTTPClients[handle]
	ahdHTTPClientsMu.Unlock()
	if state == nil {
		AhdRaiseClass(class, "HTTP Client storage is corrupted")
	}
	return state
}

func ahdHTTPRaiseClientFailure(class *AhdClass, err error) {
	if err == nil {
		AhdRaiseClass(class, "HTTP request failed")
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr != nil && urlErr.Timeout() {
		AhdRaiseClass(class, "HTTP request timed out")
	}
	if errors.Is(err, errHTTPTooManyRedirects) {
		AhdRaiseClass(class, "HTTP request exceeded 10 redirects")
	}
	if errors.Is(err, errHTTPHTTPSDowngrade) {
		AhdRaiseClass(class, "HTTP HTTPS to HTTP redirect is not allowed")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		AhdRaiseClass(class, "HTTP request timed out")
	}
	var timeout net.Error
	if errors.As(err, &timeout) && timeout.Timeout() {
		AhdRaiseClass(class, "HTTP request timed out")
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "tls") || strings.Contains(message, "certificate") || strings.Contains(message, "x509") {
		AhdRaiseClass(class, "HTTP TLS verification failed")
	}
	AhdRaiseClass(class, "HTTP request failed")
}

func ahdHTTPEncodeClientRequest(class *AhdClass, request ahdHTTPClientRequestData) string {
	encoded, err := json.Marshal(request)
	if err != nil {
		AhdRaiseClass(class, "HTTP ClientRequest storage is corrupted")
	}
	return string(encoded)
}

func ahdHTTPDecodeClientRequest(class *AhdClass, data string) ahdHTTPClientRequestData {
	var request ahdHTTPClientRequestData
	if err := json.Unmarshal([]byte(data), &request); err != nil {
		AhdRaiseClass(class, "HTTP ClientRequest storage is corrupted")
	}
	return request
}

func ahdHTTPEncodeClientResponse(class *AhdClass, response ahdHTTPClientResponseData) string {
	encoded, err := json.Marshal(response)
	if err != nil {
		AhdRaiseClass(class, "HTTP ClientResponse storage is corrupted")
	}
	return string(encoded)
}

func ahdHTTPDecodeClientResponse(class *AhdClass, data string) ahdHTTPClientResponseData {
	var response ahdHTTPClientResponseData
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		AhdRaiseClass(class, "HTTP ClientResponse storage is corrupted")
	}
	return response
}

// --- multipart/form-data uploads (v0.8.0) ---
//
// An uploaded file is never carried as an AhdCode String: a PDF is not text.
// Each file part is streamed to a private temporary file during request
// materialization, and the request snapshot carries only metadata plus an
// opaque, unguessable registry id. UploadedFile's hidden field encodes that
// metadata and id, so no Go file handle, multipart.File, or pointer is ever
// reachable from AhdCode.
//
// The registry entry lives exactly as long as the request: ServeHTTP
// releases every id it created once the handler returns, whether the handler
// responded normally, rejected the upload, or panicked. Saving moves the
// bytes out of that lifetime; everything unsaved is deleted.
//
// The stored basename is always crypto/rand hex, never the browser-supplied
// filename, so an attacker-controlled name can neither escape the
// application's directory nor overwrite an existing file.

const (
	ahdHTTPUploadIDBytes    = 16
	ahdHTTPUploadNameBytes  = 16
	ahdHTTPUploadSniffBytes = 512
	ahdHTTPUploadNameTries  = 8
)

// ahdHTTPUploadEntry is one uploaded file as seen by the request snapshot.
// Size is the exact payload length in bytes. Declared is the client-supplied
// Content-Type (never trusted); Detected is sniffed from the leading bytes.
type ahdHTTPUploadEntry struct {
	Field        string `json:"field"`
	ID           string `json:"id"`
	OriginalName string `json:"originalName"`
	Declared     string `json:"declared,omitempty"`
	HasDeclared  bool   `json:"hasDeclared"`
	Detected     string `json:"detected"`
	Size         int64  `json:"size"`
}

// ahdHTTPUploadRecord is the private server-side backing for one uploaded
// file. It never leaves the runtime.
type ahdHTTPUploadRecord struct {
	tempPath string
	saved    bool
}

var ahdHTTPUploads = struct {
	mutex   sync.Mutex
	records map[string]*ahdHTTPUploadRecord
}{records: map[string]*ahdHTTPUploadRecord{}}

func ahdHTTPUploadRandomHex(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", buffer), nil
}

// ahdHTTPUploadSafeName reduces a browser-supplied filename to a display-only
// basename. Both '/' and '\\' are treated as separators regardless of host
// platform, a Windows drive prefix is dropped, and "."/".."/empty collapse to
// a neutral name, so originalName can never carry operative path traversal.
// A NUL byte is a structurally invalid filename and is rejected.
func ahdHTTPUploadSafeName(raw string) (string, error) {
	if strings.IndexByte(raw, 0) >= 0 {
		return "", errors.New("multipart filename contains a NUL byte")
	}
	name := raw
	if index := strings.LastIndexAny(name, `/\`); index >= 0 {
		name = name[index+1:]
	}
	if len(name) >= 2 && name[1] == ':' {
		name = name[2:]
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "file", nil
	}
	if !utf8.ValidString(name) {
		return "", errors.New("multipart filename is not valid UTF-8")
	}
	return name, nil
}

// ahdHTTPUploadMediaType returns the bare media type of a MIME header value,
// dropping parameters such as charset. An unparsable value yields "".
func ahdHTTPUploadMediaType(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(mediaType))
}

// ahdHTTPUploadDetect sniffs a media type from the leading bytes only. The
// filename and the client-declared Content-Type deliberately play no part.
// Zero bytes have no content to resemble, so they report the documented
// application/octet-stream fallback rather than being called text.
func ahdHTTPUploadDetect(head []byte, size int64) string {
	if size == 0 {
		return "application/octet-stream"
	}
	detected := ahdHTTPUploadMediaType(http.DetectContentType(head))
	if detected == "" {
		return "application/octet-stream"
	}
	return detected
}

// ahdHTTPParseMultipart streams every part of a multipart/form-data body.
// Text parts join the ordinary form/formAll values; file parts are written to
// private temporary files and registered. The caller owns releasing the
// returned ids.
func ahdHTTPParseMultipart(body []byte, boundary string, snapshot *ahdHTTPRequestData) ([]string, error) {
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	form := map[string][]string{}
	var ids []string
	release := func() {
		ahdHTTPReleaseUploads(ids)
	}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			release()
			return nil, err
		}
		field := part.FormName()
		if part.FileName() == "" {
			value, readErr := io.ReadAll(part)
			_ = part.Close()
			if readErr != nil {
				release()
				return nil, readErr
			}
			if !utf8.Valid(value) {
				release()
				return nil, errHTTPEncodedUTF8
			}
			form[field] = append(form[field], string(value))
			continue
		}
		entry, id, fileErr := ahdHTTPStoreUploadPart(part, field)
		_ = part.Close()
		if fileErr != nil {
			release()
			return nil, fileErr
		}
		ids = append(ids, id)
		snapshot.Files = append(snapshot.Files, entry)
	}
	snapshot.Form = form
	snapshot.FormOK = true
	snapshot.IsForm = true
	snapshot.IsMultipart = true
	return ids, nil
}

// ahdHTTPStoreUploadPart writes one file part to a temporary file and
// registers it, returning the metadata the request snapshot carries.
func ahdHTTPStoreUploadPart(part *multipart.Part, field string) (ahdHTTPUploadEntry, string, error) {
	name, err := ahdHTTPUploadSafeName(part.FileName())
	if err != nil {
		return ahdHTTPUploadEntry{}, "", err
	}
	id, err := ahdHTTPUploadRandomHex(ahdHTTPUploadIDBytes)
	if err != nil {
		return ahdHTTPUploadEntry{}, "", err
	}
	file, err := os.CreateTemp("", "ahdcode-upload-*")
	if err != nil {
		return ahdHTTPUploadEntry{}, "", err
	}
	tempPath := file.Name()
	head := make([]byte, 0, ahdHTTPUploadSniffBytes)
	buffer := make([]byte, 32*1024)
	var size int64
	for {
		read, readErr := part.Read(buffer)
		if read > 0 {
			if len(head) < ahdHTTPUploadSniffBytes {
				room := ahdHTTPUploadSniffBytes - len(head)
				if room > read {
					room = read
				}
				head = append(head, buffer[:room]...)
			}
			written, writeErr := file.Write(buffer[:read])
			size += int64(written)
			if writeErr != nil {
				_ = file.Close()
				_ = os.Remove(tempPath)
				return ahdHTTPUploadEntry{}, "", writeErr
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = file.Close()
			_ = os.Remove(tempPath)
			return ahdHTTPUploadEntry{}, "", readErr
		}
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return ahdHTTPUploadEntry{}, "", err
	}
	declared := ahdHTTPUploadMediaType(part.Header.Get("Content-Type"))
	entry := ahdHTTPUploadEntry{
		Field: field, ID: id, OriginalName: name,
		Declared: declared, HasDeclared: declared != "",
		Detected: ahdHTTPUploadDetect(head, size), Size: size,
	}
	ahdHTTPUploads.mutex.Lock()
	ahdHTTPUploads.records[id] = &ahdHTTPUploadRecord{tempPath: tempPath}
	ahdHTTPUploads.mutex.Unlock()
	return entry, id, nil
}

// ahdHTTPReleaseUploads drops every listed upload from the registry and
// deletes any temporary file that was never saved. It is safe to call twice.
func ahdHTTPReleaseUploads(ids []string) {
	if len(ids) == 0 {
		return
	}
	var stale []string
	ahdHTTPUploads.mutex.Lock()
	for _, id := range ids {
		record, found := ahdHTTPUploads.records[id]
		if !found {
			continue
		}
		delete(ahdHTTPUploads.records, id)
		if !record.saved {
			stale = append(stale, record.tempPath)
		}
	}
	ahdHTTPUploads.mutex.Unlock()
	for _, path := range stale {
		_ = os.Remove(path)
	}
}

func ahdHTTPUploadEntryFor(class *AhdClass, data string) ahdHTTPUploadEntry {
	var entry ahdHTTPUploadEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		AhdRaiseClass(class, "HTTP UploadedFile storage is corrupted")
	}
	return entry
}

func ahdHTTPEncodeUpload(class *AhdClass, entry ahdHTTPUploadEntry) string {
	encoded, err := json.Marshal(entry)
	if err != nil {
		AhdRaiseClass(class, "HTTP UploadedFile storage is corrupted")
	}
	return string(encoded)
}

// AhdHTTPRequestFile is Request.file(name): the first uploaded file for that
// field, or null when the field carried no file.
func AhdHTTPRequestFile(class *AhdClass, data, name string) *string {
	request := ahdHTTPRequireForm(class, data)
	for _, entry := range request.Files {
		if entry.Field == name {
			encoded := ahdHTTPEncodeUpload(class, entry)
			return &encoded
		}
	}
	return nil
}

// AhdHTTPRequestFiles is Request.files(name): every uploaded file for that
// field in request order, or an empty List.
func AhdHTTPRequestFiles(class *AhdClass, data, name string) []string {
	request := ahdHTTPRequireForm(class, data)
	var result []string
	for _, entry := range request.Files {
		if entry.Field == name {
			result = append(result, ahdHTTPEncodeUpload(class, entry))
		}
	}
	return result
}

func AhdHTTPUploadedFileOriginalName(class *AhdClass, data string) string {
	return ahdHTTPUploadEntryFor(class, data).OriginalName
}

func AhdHTTPUploadedFileDeclaredContentType(class *AhdClass, data string) *string {
	entry := ahdHTTPUploadEntryFor(class, data)
	if !entry.HasDeclared {
		return nil
	}
	declared := entry.Declared
	return &declared
}

func AhdHTTPUploadedFileDetectedContentType(class *AhdClass, data string) string {
	return ahdHTTPUploadEntryFor(class, data).Detected
}

func AhdHTTPUploadedFileSize(class *AhdClass, data string) int64 {
	return ahdHTTPUploadEntryFor(class, data).Size
}

// AhdHTTPUploadedFileSave persists the upload inside directory under a
// crypto-random basename and returns the stored path. The browser-supplied
// filename never reaches the filesystem. An upload can be saved once: a
// second save raises rather than silently writing a duplicate copy.
func AhdHTTPUploadedFileSave(class *AhdClass, data, directory string) string {
	entry := ahdHTTPUploadEntryFor(class, data)
	if strings.TrimSpace(directory) == "" {
		AhdRaiseClass(class, "UploadedFile.save requires a directory")
	}
	ahdHTTPUploads.mutex.Lock()
	record, found := ahdHTTPUploads.records[entry.ID]
	if found && record.saved {
		ahdHTTPUploads.mutex.Unlock()
		AhdRaiseClass(class, "this uploaded file has already been saved")
	}
	if !found {
		ahdHTTPUploads.mutex.Unlock()
		AhdRaiseClass(class, "this uploaded file is no longer available; save it while its request is still being handled")
	}
	tempPath := record.tempPath
	ahdHTTPUploads.mutex.Unlock()

	if err := os.MkdirAll(directory, 0o755); err != nil {
		AhdRaiseClass(class, "could not create the upload directory: "+err.Error())
	}
	source, err := os.Open(tempPath)
	if err != nil {
		AhdRaiseClass(class, "could not read the uploaded file: "+err.Error())
	}
	defer func() { _ = source.Close() }()

	var stored *os.File
	storedPath := ""
	for attempt := 0; attempt < ahdHTTPUploadNameTries; attempt++ {
		basename, randErr := ahdHTTPUploadRandomHex(ahdHTTPUploadNameBytes)
		if randErr != nil {
			AhdRaiseClass(class, "could not generate a stored file name: "+randErr.Error())
		}
		candidate := filepath.Join(directory, basename)
		file, openErr := os.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if openErr == nil {
			stored, storedPath = file, candidate
			break
		}
		if !os.IsExist(openErr) {
			AhdRaiseClass(class, "could not create the stored file: "+openErr.Error())
		}
	}
	if stored == nil {
		AhdRaiseClass(class, "could not create a unique stored file name")
	}
	if _, err := io.Copy(stored, source); err != nil {
		_ = stored.Close()
		_ = os.Remove(storedPath)
		AhdRaiseClass(class, "could not write the stored file: "+err.Error())
	}
	if err := stored.Close(); err != nil {
		_ = os.Remove(storedPath)
		AhdRaiseClass(class, "could not finish writing the stored file: "+err.Error())
	}

	ahdHTTPUploads.mutex.Lock()
	if current, ok := ahdHTTPUploads.records[entry.ID]; ok {
		current.saved = true
	}
	ahdHTTPUploads.mutex.Unlock()
	_ = os.Remove(tempPath)
	return storedPath
}

type ahdModuleError struct {
	AhdBase
	message string
}

func (value *ahdModuleError) AhdErrorMessage() string { return value.message }

func (value *ahdModuleError) AhdFreezeGraph(visited map[AhdFreezable]bool) {
	if !AhdEnterFreeze(value, visited) {
		return
	}
	value.AhdMarkFrozen()
}

func init() {
	for _, class := range []*AhdClass{AhdClassHTTPError, AhdClassHTMLError} {
		target := class
		AhdRegisterError(target, func(message string) AhdInstance {
			instance := &ahdModuleError{message: message}
			instance.AhdSetClass(target)
			return instance
		})
	}
}
