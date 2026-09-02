package ahdruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
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
}

type ahdHTTPResponseData struct {
	Status      int                 `json:"status"`
	Body        string              `json:"body"`
	ContentType string              `json:"contentType"`
	Headers     []ahdHTTPHeaderPair `json:"headers"`
}

type ahdHTTPHeaderPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

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

func AhdHTTPResponseWithHeader(class *AhdClass, data, name, value string) string {
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

	snapshot := ahdHTTPMaterialize(request, body)
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

func ahdHTTPMaterialize(request *http.Request, body []byte) ahdHTTPRequestData {
	query := request.URL.Query()
	headers := map[string][]string{}
	for name, values := range request.Header {
		headers[textproto.CanonicalMIMEHeaderKey(name)] = append([]string(nil), values...)
	}
	snapshot := ahdHTTPRequestData{
		Method: request.Method, Path: request.URL.Path,
		Query: query, Headers: headers, BodyUTF8: utf8.Valid(body),
	}
	if snapshot.BodyUTF8 {
		snapshot.Body = string(body)
	}
	mediaType, _, _ := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if mediaType == "application/x-www-form-urlencoded" {
		snapshot.IsForm = true
		if !snapshot.BodyUTF8 {
			snapshot.FormError = "HTTP form body is not valid UTF-8"
			return snapshot
		}
		values, err := url.ParseQuery(snapshot.Body)
		if err != nil {
			snapshot.FormError = "HTTP form body is not valid URL encoding"
			return snapshot
		}
		snapshot.Form = values
		snapshot.FormOK = true
	}
	return snapshot
}

func ahdHTTPRequireForm(class *AhdClass, data string) ahdHTTPRequestData {
	request := ahdHTTPDecodeRequest(data)
	if request.IsForm && request.FormError != "" {
		AhdRaiseClass(class, request.FormError)
	}
	if !request.BodyUTF8 && request.IsForm {
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
