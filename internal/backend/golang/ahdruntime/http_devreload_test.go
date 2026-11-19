package ahdruntime

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

const devReloadTestSession = "0123456789abcdef0123456789abcdef"

func withDevReload(t *testing.T) {
	t.Helper()
	ahdDevReloadTestInstall(devReloadTestSession, "4")
	t.Cleanup(ahdDevReloadTestReset)
}

func TestDevReloadStaysOffWithoutAStartupSession(t *testing.T) {
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/", func(string) string {
			return AhdHTTPHTML(AhdClassHTTPError, "<!doctype html><html><body><h1>İstanbul</h1></body></html>", 200)
		})
	})
	body := httpGetText(t, base+"/")
	if strings.Contains(body, ahdDevReloadScriptPath) || strings.Contains(body, ahdDevReloadMarker) {
		t.Fatalf("ordinary HTML was instrumented: %s", body)
	}
	if !strings.Contains(body, "İstanbul") {
		t.Fatalf("page lost İstanbul: %s", body)
	}
	response, err := http.Get(base + ahdDevReloadPath)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("reload path status = %d, want 404", response.StatusCode)
	}
}

func TestDevReloadPreservesTurkishCapitalI(t *testing.T) {
	withDevReload(t)
	pageText := "<!doctype html><html><BODY><h1>" + strings.Repeat("İstanbul İletişim ", 12) + "</h1></BODY></html>"
	base, handle := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/", func(string) string {
			return AhdHTTPHTML(AhdClassHTTPError, pageText, 200)
		})
	})
	body := httpGetText(t, base+"/")
	if !utf8.ValidString(body) {
		t.Fatal("injected page is not valid UTF-8")
	}
	if strings.Count(body, "İstanbul") != 12 || strings.Count(body, "İletişim") != 12 {
		t.Fatalf("Turkish text was damaged: %s", body)
	}
	scriptAt := strings.Index(body, ahdDevReloadMarker)
	closeAt := strings.LastIndex(body, "</BODY>")
	if scriptAt < 0 || closeAt < 0 || scriptAt > closeAt {
		t.Fatalf("script was not placed before </BODY>: %s", body)
	}
	if strings.Count(body, "İ") != 24 {
		t.Fatalf("İ count = %d, want 24", strings.Count(body, "İ"))
	}
	if !strings.Contains(body, `data-session="`+devReloadTestSession+`" data-generation="4"`) {
		t.Fatalf("page did not carry its session and generation: %s", body)
	}
	probe := httpGetText(t, base+ahdDevReloadPath)
	if probe != devReloadTestSession+" 4" {
		t.Fatalf("probe = %q", probe)
	}
	script := httpGetText(t, base+ahdDevReloadScriptPath)
	if strings.Count(script, "setInterval") != 1 || !strings.Contains(script, "var busy=false") || !strings.Contains(script, "var stopped=false") {
		t.Fatalf("reload script does not keep a single poll and a single reload: %s", script)
	}
	state := ahdHTTPLookup(AhdClassHTTPError, handle)
	lan := httptest.NewRequest(http.MethodGet, ahdDevReloadPath, nil)
	lan.RemoteAddr = "192.168.1.20:4000"
	lanRecorder := httptest.NewRecorder()
	(ahdHTTPDispatcher{class: AhdClassHTTPError, state: state}).ServeHTTP(lanRecorder, lan)
	if lanRecorder.Code != http.StatusNotFound {
		t.Fatalf("LAN client received the development probe: %d %s", lanRecorder.Code, lanRecorder.Body.String())
	}
}

func TestDevReloadUsesTheFinalContentTypeAndSkipsUnsafeBodies(t *testing.T) {
	withDevReload(t)
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte("<html><body>İstanbul</body></html>")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	gzipBody := compressed.String()
	if prepared, modified := ahdDevReloadPrepare("text/html; charset=utf-8", 200, []ahdHTTPHeaderPair{{Name: "Content-Encoding", Value: "gzip"}}, gzipBody); modified || prepared != gzipBody {
		t.Fatal("gzip bytes were rewritten before they reached the socket")
	}
	utf16 := utf16LEString("<html><body>İstanbul</body></html>")
	if prepared, modified := ahdDevReloadPrepare("text/html; charset=utf-16", 200, nil, utf16); modified || prepared != utf16 {
		t.Fatal("UTF-16 bytes were rewritten")
	}
	for _, status := range []int{101, 204, 302} {
		if prepared, modified := ahdDevReloadPrepare("text/html", status, nil, "<html><body>İstanbul</body></html>"); modified || strings.Contains(prepared, "<script") {
			t.Fatalf("status %d was instrumented", status)
		}
	}
	fragment, fragmentModified := ahdDevReloadPrepare("text/html; charset=utf-8", 200, nil, "<p>İstanbul</p>")
	if !fragmentModified || !strings.HasPrefix(fragment, "<p>İstanbul</p><script ") {
		t.Fatalf("HTML fragment was not left intact with the script appended: %q", fragment)
	}
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/json", func(string) string {
			response := AhdHTTPResponse(AhdClassHTTPError, 200, `{"city":"İstanbul"}`, "text/html; charset=utf-8")
			return AhdHTTPResponseWithHeader(AhdClassHTTPError, response, "Content-Type", "application/json")
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/html", func(string) string {
			response := AhdHTTPResponse(AhdClassHTTPError, 200, "<html><body>İstanbul</body></html>", "text/plain")
			return AhdHTTPResponseWithHeader(AhdClassHTTPError, response, "Content-Type", "text/html; charset=utf-8")
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/gzip", func(string) string {
			response := AhdHTTPHTML(AhdClassHTTPError, "<html><body>İstanbul</body></html>", 200)
			return AhdHTTPResponseWithHeader(AhdClassHTTPError, response, "Content-Encoding", "gzip")
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/utf16", func(string) string {
			return AhdHTTPResponse(AhdClassHTTPError, 200, "<html><body>İstanbul</body></html>", "text/html; charset=utf-16")
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/redirect", func(string) string {
			response := AhdHTTPResponse(AhdClassHTTPError, 302, "<html><body>İstanbul</body></html>", "text/html")
			return AhdHTTPResponseWithHeader(AhdClassHTTPError, response, "Location", "/next")
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/mention", func(string) string {
			return AhdHTTPHTML(AhdClassHTTPError, "<html><body>See /__ahdcode/dev/reload in the docs.</body></html>", 200)
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/marked", func(string) string {
			return AhdHTTPHTML(AhdClassHTTPError, `<html><body>already `+ahdDevReloadMarker+`</body></html>`, 200)
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/file", func(string) string {
			response := AhdHTTPHTML(AhdClassHTTPError, "<html><body>İstanbul</body></html>", 200)
			return AhdHTTPResponseWithHeader(AhdClassHTTPError, response, "Content-Disposition", `attachment; filename="page.html"`)
		})
	})
	client := &http.Client{
		Transport: &http.Transport{DisableCompression: true},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	jsonBody, jsonHeader := httpGetDetailClient(t, client, base+"/json")
	if jsonBody != `{"city":"İstanbul"}` {
		t.Fatalf("JSON changed: %q", jsonBody)
	}
	if jsonHeader.Get("Content-Type") != "application/json" {
		t.Fatalf("JSON content-type = %q", jsonHeader.Get("Content-Type"))
	}
	if jsonHeader.Get("Cache-Control") == "no-store" {
		t.Fatal("JSON gained a development cache header")
	}

	htmlBody, htmlHeader := httpGetDetailClient(t, client, base+"/html")
	if !strings.Contains(htmlBody, "İstanbul") || !strings.Contains(htmlBody, ahdDevReloadScriptPath) {
		t.Fatalf("eligible HTML was not instrumented: %s", htmlBody)
	}
	if htmlHeader.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("HTML content-type = %q", htmlHeader.Get("Content-Type"))
	}
	if htmlHeader.Get("Content-Length") != strconv.Itoa(len(htmlBody)) {
		t.Fatalf("content-length = %q, body = %d", htmlHeader.Get("Content-Length"), len(htmlBody))
	}

	gzipGot, gzipHeader := httpGetDetailClient(t, client, base+"/gzip")
	if gzipGot != "<html><body>İstanbul</body></html>" {
		t.Fatalf("compressed response body changed: %q", gzipGot)
	}
	if gzipHeader.Get("Content-Encoding") != "gzip" {
		t.Fatalf("content-encoding = %q", gzipHeader.Get("Content-Encoding"))
	}
	if strings.Contains(gzipGot, "<script") {
		t.Fatal("plaintext script was appended to a compressed response")
	}

	utf16Got, _ := httpGetDetailClient(t, client, base+"/utf16")
	if utf16Got != "<html><body>İstanbul</body></html>" || strings.Contains(utf16Got, "<script") {
		t.Fatalf("UTF-16 labeled response was modified: %q", utf16Got)
	}

	redirect, redirectHeader := httpGetDetailClient(t, client, base+"/redirect")
	if redirect != "<html><body>İstanbul</body></html>" {
		t.Fatalf("redirect body changed: %q", redirect)
	}
	if redirectHeader.Get("Cache-Control") == "no-store" {
		t.Fatal("redirect gained a development cache header")
	}

	mention := httpGetText(t, base+"/mention")
	if strings.Count(mention, ahdDevReloadScriptPath) != 1 || !strings.Contains(mention, "See /__ahdcode/dev/reload") {
		t.Fatalf("mentioning the path blocked or duplicated injection: %s", mention)
	}

	marked, markedHeader := httpGetDetailClient(t, client, base+"/marked")
	if strings.Count(marked, ahdDevReloadScriptPath) != 0 {
		t.Fatalf("marker page was injected again: %s", marked)
	}
	if markedHeader.Get("Cache-Control") == "no-store" {
		t.Fatal("an unchanged body gained a development cache header")
	}

	download := httpGetText(t, base+"/file")
	if strings.Contains(download, ahdDevReloadScriptPath) {
		t.Fatalf("attachment was instrumented: %s", download)
	}
}

func TestDevReloadHeadMatchesTheGetLength(t *testing.T) {
	withDevReload(t)
	page := "<html><body>" + strings.Repeat("İstanbul ", 4000) + "</body></html>"
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		pageHandler := func(string) string {
			return AhdHTTPHTML(AhdClassHTTPError, page, 200)
		}
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/", pageHandler)
		AhdHTTPServerRoute(AhdClassHTTPError, handle, http.MethodHead, "/", pageHandler)
	})
	getBody, getHeader := httpGetDetail(t, base+"/")
	headRequest, err := http.NewRequest(http.MethodHead, base+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	head, err := http.DefaultClient.Do(headRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer head.Body.Close()
	headBody, _ := io.ReadAll(head.Body)
	if len(headBody) != 0 {
		t.Fatalf("HEAD body = %d bytes", len(headBody))
	}
	if head.StatusCode != http.StatusOK {
		t.Fatalf("HEAD status = %d", head.StatusCode)
	}
	if head.Header.Get("Content-Length") != strconv.Itoa(len(getBody)) || getHeader.Get("Content-Length") != strconv.Itoa(len(getBody)) {
		t.Fatalf("HEAD length %q, GET length %q, body %d", head.Header.Get("Content-Length"), getHeader.Get("Content-Length"), len(getBody))
	}
	probe, err := http.NewRequest(http.MethodHead, base+ahdDevReloadPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	probeHead, err := http.DefaultClient.Do(probe)
	if err != nil {
		t.Fatal(err)
	}
	defer probeHead.Body.Close()
	wantProbe := devReloadTestSession + " 4"
	if probeHead.Header.Get("Content-Length") != strconv.Itoa(len(wantProbe)) {
		t.Fatalf("probe HEAD length = %q", probeHead.Header.Get("Content-Length"))
	}
}

func TestDevReloadLoopbackAndRouteCollision(t *testing.T) {
	withDevReload(t)
	base, handle := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/", func(string) string {
			return AhdHTTPHTML(AhdClassHTTPError, "<html><body>İstanbul</body></html>", 200)
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, ahdDevReloadPath, func(string) string {
			return AhdHTTPText(AhdClassHTTPError, "owned-by-the-application", 200)
		})
	})
	owned := httpGetText(t, base+ahdDevReloadPath)
	if owned != "owned-by-the-application" {
		t.Fatalf("application route was replaced: %q", owned)
	}
	page := httpGetText(t, base+"/")
	if strings.Contains(page, ahdDevReloadScriptPath) {
		t.Fatalf("script injected while the probe route belongs to the application: %s", page)
	}

	state := ahdHTTPLookup(AhdClassHTTPError, handle)
	request := httptest.NewRequest(http.MethodGet, ahdDevReloadScriptPath, nil)
	request.RemoteAddr = "192.168.1.20:4000"
	recorder := httptest.NewRecorder()
	(ahdHTTPDispatcher{class: AhdClassHTTPError, state: state}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("LAN client received the development script: %d %s", recorder.Code, recorder.Body.String())
	}
	loopback := httptest.NewRequest(http.MethodGet, ahdDevReloadScriptPath, nil)
	loopback.RemoteAddr = "127.0.0.1:4000"
	loopRecorder := httptest.NewRecorder()
	(ahdHTTPDispatcher{class: AhdClassHTTPError, state: state}).ServeHTTP(loopRecorder, loopback)
	if loopRecorder.Code != http.StatusOK || !strings.Contains(loopRecorder.Body.String(), "location.reload") {
		t.Fatalf("loopback script = %d %s", loopRecorder.Code, loopRecorder.Body.String())
	}
}

func TestDevReloadLeavesNonLoopbackHTMLUntouched(t *testing.T) {
	withDevReload(t)
	page := "<html><body>İstanbul İletişim</body></html>"
	_, handle := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/", func(string) string {
			return AhdHTTPHTML(AhdClassHTTPError, page, 200)
		})
	})
	state := ahdHTTPLookup(AhdClassHTTPError, handle)
	cases := []struct {
		remote string
		inject bool
	}{
		{remote: "127.0.0.1:4000", inject: true},
		{remote: "[::1]:4000", inject: true},
		{remote: "192.168.1.20:4000", inject: false},
		{remote: "[2001:db8::1]:4000", inject: false},
	}
	for _, testCase := range cases {
		t.Run(testCase.remote, func(t *testing.T) {
			pageResponse := devReloadExchange(t, state, http.MethodGet, "/", testCase.remote)
			body := pageResponse.Body.String()
			if testCase.inject {
				if pageResponse.Code != http.StatusOK || !strings.Contains(body, ahdDevReloadScriptPath) || !strings.Contains(body, `data-session="`+devReloadTestSession+`"`) {
					t.Fatalf("loopback page was not instrumented: %d %s", pageResponse.Code, body)
				}
			} else if body != page || strings.Contains(body, "data-session") || strings.Contains(body, ahdDevReloadScriptPath) || pageResponse.Header().Get("Cache-Control") == "no-store" {
				t.Fatalf("non-loopback page was instrumented: %d cache %q body %s", pageResponse.Code, pageResponse.Header().Get("Cache-Control"), body)
			}
			probe := devReloadExchange(t, state, http.MethodGet, ahdDevReloadPath, testCase.remote)
			script := devReloadExchange(t, state, http.MethodGet, ahdDevReloadScriptPath, testCase.remote)
			if testCase.inject {
				if probe.Code != http.StatusOK || probe.Body.String() != devReloadTestSession+" 4" || script.Code != http.StatusOK {
					t.Fatalf("loopback endpoints = probe %d %q script %d", probe.Code, probe.Body.String(), script.Code)
				}
				return
			}
			if probe.Code != http.StatusNotFound || script.Code != http.StatusNotFound || strings.Contains(probe.Body.String(), devReloadTestSession) {
				t.Fatalf("non-loopback endpoints = probe %d %q script %d %q", probe.Code, probe.Body.String(), script.Code, script.Body.String())
			}
		})
	}
}

func TestDevReloadPreservesANonGetApplicationRoute(t *testing.T) {
	withDevReload(t)
	page := "<html><body>İstanbul</body></html>"
	_, handle := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/", func(string) string {
			return AhdHTTPHTML(AhdClassHTTPError, page, 200)
		})
		AhdHTTPServerPost(AhdClassHTTPError, handle, ahdDevReloadPath, func(string) string {
			return AhdHTTPText(AhdClassHTTPError, "posted-by-the-application", 200)
		})
		AhdHTTPServerRoute(AhdClassHTTPError, handle, http.MethodPut, ahdDevReloadScriptPath, func(string) string {
			return AhdHTTPText(AhdClassHTTPError, "put-by-the-application", 200)
		})
	})
	state := ahdHTTPLookup(AhdClassHTTPError, handle)
	for _, remote := range []string{"127.0.0.1:4000", "[::1]:4000", "192.168.1.20:4000", "[2001:db8::1]:4000"} {
		posted := devReloadExchange(t, state, http.MethodPost, ahdDevReloadPath, remote)
		if posted.Code != http.StatusOK || posted.Body.String() != "posted-by-the-application" {
			t.Fatalf("%s POST = %d %q", remote, posted.Code, posted.Body.String())
		}
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			response := devReloadExchange(t, state, method, ahdDevReloadPath, remote)
			if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodPost || strings.Contains(response.Body.String(), devReloadTestSession) {
				t.Fatalf("%s %s probe = %d allow %q body %q", remote, method, response.Code, response.Header().Get("Allow"), response.Body.String())
			}
		}
		put := devReloadExchange(t, state, http.MethodPut, ahdDevReloadScriptPath, remote)
		if put.Code != http.StatusOK || put.Body.String() != "put-by-the-application" {
			t.Fatalf("%s PUT = %d %q", remote, put.Code, put.Body.String())
		}
		script := devReloadExchange(t, state, http.MethodGet, ahdDevReloadScriptPath, remote)
		if script.Code != http.StatusMethodNotAllowed || script.Header().Get("Allow") != http.MethodPut || strings.Contains(script.Body.String(), "location.reload") {
			t.Fatalf("%s GET script = %d allow %q body %q", remote, script.Code, script.Header().Get("Allow"), script.Body.String())
		}
		html := devReloadExchange(t, state, http.MethodGet, "/", remote)
		if html.Body.String() != page || html.Header().Get("Cache-Control") == "no-store" {
			t.Fatalf("%s HTML changed while the development paths are owned: %s", remote, html.Body.String())
		}
	}
}

func devReloadExchange(t *testing.T, state *ahdHTTPServerState, method, path, remote string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	request.RemoteAddr = remote
	recorder := httptest.NewRecorder()
	(ahdHTTPDispatcher{class: AhdClassHTTPError, state: state}).ServeHTTP(recorder, request)
	return recorder
}

func TestDevReloadSnapshotIsFixedAtProcessStart(t *testing.T) {
	if mode := os.Getenv("AHDCODE_DEV_SNAPSHOT_MODE"); mode != "" {
		ahdDevReloadSnapshotChild(mode)
		return
	}
	absent := exec.Command(os.Args[0], "-test.run=TestDevReloadSnapshotIsFixedAtProcessStart")
	absent.Env = append(os.Environ(), "AHDCODE_DEV_SNAPSHOT_MODE=absent")
	if output, err := absent.CombinedOutput(); err != nil {
		t.Fatalf("absent child: %v\n%s", err, output)
	}
	session := strings.Repeat("cd", 16)
	present := exec.Command(os.Args[0], "-test.run=TestDevReloadSnapshotIsFixedAtProcessStart")
	present.Env = append(os.Environ(),
		"AHDCODE_DEV_SNAPSHOT_MODE=present",
		"AHDCODE_DEV_SNAPSHOT_SESSION="+session,
		"AHDCODE_DEV_SNAPSHOT_GENERATION=3",
		ahdDevReloadSessionEnv+"="+session,
		ahdDevReloadGenerationEnv+"=3",
	)
	if output, err := present.CombinedOutput(); err != nil {
		t.Fatalf("present child: %v\n%s", err, output)
	}
}

func ahdDevReloadSnapshotChild(mode string) {
	switch mode {
	case "absent":
		_ = os.Setenv(ahdDevReloadSessionEnv, devReloadTestSession)
		_ = os.Setenv(ahdDevReloadGenerationEnv, "9")
		if ahdDevReloadEnabled() {
			os.Exit(2)
		}
		os.Exit(0)
	case "present":
		_ = os.Setenv(ahdDevReloadSessionEnv, devReloadTestSession)
		_ = os.Setenv(ahdDevReloadGenerationEnv, "1")
		session, generation, ok := ahdDevReloadIdentity()
		if !ok || session != os.Getenv("AHDCODE_DEV_SNAPSHOT_SESSION") || generation != os.Getenv("AHDCODE_DEV_SNAPSHOT_GENERATION") {
			os.Exit(3)
		}
		os.Exit(0)
	default:
		os.Exit(4)
	}
}

func utf16LEString(text string) string {
	var buf bytes.Buffer
	buf.Write([]byte{0xFF, 0xFE})
	for _, r := range text {
		buf.WriteByte(byte(r))
		buf.WriteByte(byte(r >> 8))
	}
	return buf.String()
}

func httpGetText(t *testing.T, url string) string {
	t.Helper()
	body, _ := httpGetDetail(t, url)
	return body
}

func httpGetDetail(t *testing.T, url string) (string, http.Header) {
	t.Helper()
	return httpGetDetailClient(t, http.DefaultClient, url)
}

func httpGetDetailClient(t *testing.T, client *http.Client, url string) (string, http.Header) {
	t.Helper()
	response, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusFound {
		t.Fatalf("%s status = %d body = %s", url, response.StatusCode, body)
	}
	return string(body), response.Header.Clone()
}
