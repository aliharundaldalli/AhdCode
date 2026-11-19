package ahdruntime

import (
	"context"
	"io"
	"mime"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"strconv"
	"strings"
)

// Development-server browser refresh.
//
// ahdcode dev puts AHDCODE_DEV_SESSION and AHDCODE_DEV_GENERATION on the
// child process before it starts. init captures them once. Env.load and any
// later os.Setenv cannot turn the hook on or change the identity the process
// already captured. A process that was not started with both values serves
// ordinary pages and leaves /__ahdcode/dev/reload to the application.
//
// The page gets a same-origin external script, not an inline script, so a
// page with script-src 'self' can run it. The script tag carries the session
// and generation that produced the page. A poll that reports a different
// pair reloads once. A new ahdcode dev process has a new session, so a tab
// left open across a full stop and start does not stay on the old page just
// because both processes numbered their first child 1.
//
// Injection uses the headers that will actually be sent. Compressed bodies,
// incompatible charsets, redirects, attachments, and a route the application
// already owns are left untouched. The probe and the script are answered
// only for a loopback client.

const (
	ahdDevReloadSessionEnv    = "AHDCODE_DEV_SESSION"
	ahdDevReloadGenerationEnv = "AHDCODE_DEV_GENERATION"
	ahdDevReloadPath          = "/__ahdcode/dev/reload"
	ahdDevReloadScriptPath    = "/__ahdcode/dev/reload.js"
	ahdDevReloadMarker        = `data-ahdcode-dev="reload"`
	ahdDevReloadPoll          = 500
)

var (
	ahdDevReloadOn             bool
	ahdDevReloadSession        string
	ahdDevReloadGeneration     string
	ahdDevReloadInitOn         bool
	ahdDevReloadInitSession    string
	ahdDevReloadInitGeneration string
)

func init() {
	ahdDevReloadCaptureFromEnvironment()
	ahdDevReloadInitOn = ahdDevReloadOn
	ahdDevReloadInitSession = ahdDevReloadSession
	ahdDevReloadInitGeneration = ahdDevReloadGeneration
}

func ahdDevReloadCaptureFromEnvironment() {
	session := strings.TrimSpace(os.Getenv(ahdDevReloadSessionEnv))
	generation := strings.TrimSpace(os.Getenv(ahdDevReloadGenerationEnv))
	if !ahdDevReloadHex(session) || len(session) != 32 || !ahdDevReloadDigits(generation) {
		ahdDevReloadOn = false
		ahdDevReloadSession = ""
		ahdDevReloadGeneration = ""
		return
	}
	ahdDevReloadOn = true
	ahdDevReloadSession = session
	ahdDevReloadGeneration = generation
}

func ahdDevReloadEnabled() bool { return ahdDevReloadOn }

func ahdDevReloadIdentity() (string, string, bool) {
	if !ahdDevReloadOn {
		return "", "", false
	}
	return ahdDevReloadSession, ahdDevReloadGeneration, true
}

// ahdDevReloadTestInstall sets the captured identity for an in-process test.
// Production code never calls it. The process-start snapshot is what a real
// binary uses; tests that need to prove that snapshot spawn a subprocess.
func ahdDevReloadTestInstall(session, generation string) {
	ahdDevReloadOn = ahdDevReloadHex(session) && len(session) == 32 && ahdDevReloadDigits(generation)
	if ahdDevReloadOn {
		ahdDevReloadSession = session
		ahdDevReloadGeneration = generation
		return
	}
	ahdDevReloadSession = ""
	ahdDevReloadGeneration = ""
}

func ahdDevReloadTestReset() {
	ahdDevReloadOn = ahdDevReloadInitOn
	ahdDevReloadSession = ahdDevReloadInitSession
	ahdDevReloadGeneration = ahdDevReloadInitGeneration
}

func ahdDevReloadHex(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}

func ahdDevReloadDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func ahdDevReloadLoopback(request *http.Request) bool {
	if request == nil {
		return false
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		host = request.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type ahdDevReloadServerKey struct{}

// ahdDevReloadRequest carries the server into the response writer so injection
// can see whether the application already owns a development path. The writer
// does not otherwise know which server produced the response.
func ahdDevReloadRequest(request *http.Request, state *ahdHTTPServerState) *http.Request {
	if request == nil {
		return nil
	}
	return request.WithContext(context.WithValue(request.Context(), ahdDevReloadServerKey{}, state))
}

func ahdDevReloadBlocked(request *http.Request) bool {
	if request == nil {
		return false
	}
	if !ahdDevReloadLoopback(request) {
		return true
	}
	state, _ := request.Context().Value(ahdDevReloadServerKey{}).(*ahdHTTPServerState)
	if state == nil {
		return false
	}
	return ahdDevReloadRouteOwned(state, ahdDevReloadPath) || ahdDevReloadRouteOwned(state, ahdDevReloadScriptPath)
}

// ahdDevReloadRouteOwned reports whether the application registered the path
// for any method. A POST-only registration still owns the path: the
// development handler must not answer it with 405, and the page must not
// gain a script that would call that path.
func ahdDevReloadRouteOwned(state *ahdHTTPServerState, path string) bool {
	if state == nil {
		return false
	}
	state.mutex.Lock()
	_, found, allowed := ahdHTTPLookupRoute(state, http.MethodGet, path)
	state.mutex.Unlock()
	return found || len(allowed) > 0
}

// ahdDevReloadServe answers the development probe and script for a loopback
// client when this process was started by ahdcode dev and the application
// did not register the path for any method. The same loopback decision
// gates HTML injection. It reports whether the request was consumed.
func ahdDevReloadServe(state *ahdHTTPServerState, writer http.ResponseWriter, request *http.Request) bool {
	if !ahdDevReloadEnabled() || request == nil || request.URL == nil || !ahdDevReloadLoopback(request) {
		return false
	}
	path := request.URL.Path
	if path != ahdDevReloadPath && path != ahdDevReloadScriptPath {
		return false
	}
	if ahdDevReloadRouteOwned(state, path) {
		return false
	}
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		ahdHTTPWritePlain(writer, http.StatusMethodNotAllowed, "Method Not Allowed")
		return true
	}
	var body string
	var contentType string
	if path == ahdDevReloadScriptPath {
		body = ahdDevReloadScript()
		contentType = "application/javascript; charset=utf-8"
	} else {
		body = ahdDevReloadSession + " " + ahdDevReloadGeneration
		contentType = "text/plain; charset=utf-8"
	}
	writer.Header().Set("Content-Type", contentType)
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	writer.WriteHeader(http.StatusOK)
	if request.Method != http.MethodHead {
		_, _ = io.WriteString(writer, body)
	}
	return true
}

// ahdDevReloadPrepare reports the body to send and whether this response was
// modified. Headers are adjusted by the caller only when modified is true.
func ahdDevReloadPrepare(contentType string, status int, headers []ahdHTTPHeaderPair, body string) (string, bool) {
	if !ahdDevReloadEnabled() || body == "" {
		return body, false
	}
	if status < 200 || (status >= 300 && status < 400) || status == http.StatusNoContent {
		return body, false
	}
	mediaType, charset, encoding, disposition := ahdDevReloadEffective(contentType, headers)
	if encoding != "" && !strings.EqualFold(encoding, "identity") {
		return body, false
	}
	if strings.Contains(strings.ToLower(disposition), "attachment") {
		return body, false
	}
	if !strings.EqualFold(mediaType, "text/html") || !ahdDevReloadCharsetOK(charset) {
		return body, false
	}
	if strings.Contains(body, ahdDevReloadMarker) {
		return body, false
	}
	updated := ahdDevReloadInjectHTML(body)
	if updated == body {
		return body, false
	}
	return updated, true
}

func ahdDevReloadEffective(contentType string, headers []ahdHTTPHeaderPair) (mediaType, charset, encoding, disposition string) {
	effectiveType := contentType
	for _, header := range headers {
		switch textproto.CanonicalMIMEHeaderKey(header.Name) {
		case "Content-Type":
			effectiveType = header.Value
		case "Content-Encoding":
			encoding = header.Value
		case "Content-Disposition":
			disposition = header.Value
		}
	}
	mediaType, params, err := mime.ParseMediaType(effectiveType)
	if err != nil {
		return "", "", encoding, disposition
	}
	return mediaType, params["charset"], encoding, disposition
}

func ahdDevReloadCharsetOK(charset string) bool {
	switch strings.ToLower(strings.TrimSpace(charset)) {
	case "", "utf-8", "utf8", "us-ascii", "ascii", "iso-8859-1", "latin1", "iso-8859-15", "windows-1252", "windows-1254":
		return true
	default:
		return false
	}
}

// ahdDevReloadInjectHTML inserts the external script before the last closing
// body tag. The search folds only the seven ASCII bytes of that tag, so a
// character such as İ, whose lowercase form has a different UTF-8 length,
// cannot move the insertion point.
func ahdDevReloadInjectHTML(body string) string {
	snippet := ahdDevReloadTag()
	if index := ahdDevReloadLastBodyClose(body); index >= 0 {
		return body[:index] + snippet + body[index:]
	}
	return body + snippet
}

func ahdDevReloadLastBodyClose(body string) int {
	const tag = "</body>"
	last := -1
	for i := 0; i+len(tag) <= len(body); i++ {
		match := true
		for j := 0; j < len(tag); j++ {
			if ahdDevReloadASCIIFold(body[i+j]) != tag[j] {
				match = false
				break
			}
		}
		if match {
			last = i
		}
	}
	return last
}

func ahdDevReloadASCIIFold(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

func ahdDevReloadTag() string {
	return `<script src="` + ahdDevReloadScriptPath + `" ` + ahdDevReloadMarker + ` data-session="` + ahdDevReloadSession + `" data-generation="` + ahdDevReloadGeneration + `"></script>`
}

func ahdDevReloadScript() string {
	return `(function(){var c=document.currentScript;if(!c){return;}var session=c.getAttribute("data-session")||"";var generation=c.getAttribute("data-generation")||"";var path="` + ahdDevReloadPath + `";var busy=false;var stopped=false;function tick(){if(busy||stopped){return;}busy=true;fetch(path,{cache:"no-store"}).then(function(r){if(!r.ok){throw new Error("reload");}return r.text();}).then(function(text){busy=false;var parts=text.replace(/^\s+|\s+$/g,"").split(" ");if(parts.length!==2){return;}if(!/^[0-9a-f]{32}$/.test(parts[0])||!/^[0-9]+$/.test(parts[1])){return;}if(parts[0]!==session||parts[1]!==generation){stopped=true;location.reload();}}).catch(function(){busy=false;});}setInterval(tick,` + strconv.Itoa(ahdDevReloadPoll) + `);tick();})();`
}
