package ahdruntime

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHTTPCookieDefaultsAndBuilders(t *testing.T) {
	class := AhdClassHTTPError
	encoded := AhdHTTPCookie(class, "theme", "dark")
	cookie := ahdHTTPDecodeCookie(class, encoded)
	if cookie.Name != "theme" || cookie.Value != "dark" || cookie.Path != "/" || cookie.HttpOnly || cookie.Secure || cookie.SameSite != "Lax" || cookie.MaxAge != nil {
		t.Fatalf("defaults = %#v", cookie)
	}
	secured := ahdHTTPDecodeCookie(class, AhdHTTPCookieWithSecure(class, AhdHTTPCookieWithHttpOnly(class, encoded, true), true))
	if !secured.HttpOnly || !secured.Secure {
		t.Fatalf("builders mutated incorrectly: %#v", secured)
	}
	original := ahdHTTPDecodeCookie(class, encoded)
	if original.HttpOnly || original.Secure {
		t.Fatal("Cookie builders must not mutate the original")
	}
}

func TestHTTPCookieValidation(t *testing.T) {
	class := AhdClassHTTPError
	expectRaise(t, class, func() { AhdHTTPCookie(class, "bad name", "x") })
	expectRaise(t, class, func() { AhdHTTPCookie(class, "bad;name", "x") })
	expectRaise(t, class, func() { AhdHTTPCookie(class, "ok", "x\r\nInjected: 1") })
	expectRaise(t, class, func() { AhdHTTPCookie(class, "ok", "çağ") })
	expectRaise(t, class, func() {
		AhdHTTPCookieWithPath(class, AhdHTTPCookie(class, "ok", "x"), "/a\nb")
	})
	expectRaise(t, class, func() {
		AhdHTTPCookieWithSameSite(class, AhdHTTPCookie(class, "ok", "x"), "lax")
	})
	expectRaise(t, class, func() {
		AhdHTTPCookieWithSameSite(class, AhdHTTPCookie(class, "ok", "x"), "None")
	})
	none := AhdHTTPCookieWithSameSite(class, AhdHTTPCookieWithSecure(class, AhdHTTPCookie(class, "ok", "x"), true), "None")
	decoded := ahdHTTPDecodeCookie(class, none)
	if decoded.SameSite != "None" || !decoded.Secure {
		t.Fatalf("SameSite None = %#v", decoded)
	}
}

func TestHTTPCookieWireAttributes(t *testing.T) {
	class := AhdClassHTTPError
	cookie := ahdHTTPDecodeCookie(class, AhdHTTPCookieWithMaxAge(class, AhdHTTPCookieWithHttpOnly(class, AhdHTTPCookie(class, "theme", "dark"), true), 60))
	header := ahdHTTPCookieWire(cookie)
	for _, want := range []string{"theme=dark", "Path=/", "HttpOnly", "SameSite=Lax", "Max-Age=60"} {
		if !strings.Contains(header, want) {
			t.Fatalf("Set-Cookie %q missing %q", header, want)
		}
	}
	if strings.Contains(header, "Secure") {
		t.Fatalf("unexpected Secure: %q", header)
	}
}

func TestHTTPDeleteCookieWire(t *testing.T) {
	class := AhdClassHTTPError
	header := ahdHTTPCookieWire(ahdHTTPDecodeCookie(class, AhdHTTPDeleteCookie(class, "theme", "/")))
	if !strings.Contains(header, "Max-Age=0") {
		t.Fatalf("delete cookie = %q", header)
	}
}

func TestHTTPRequestCookiesPreserveOrderAndDuplicates(t *testing.T) {
	class := AhdClassHTTPError
	data := ahdHTTPEncodeRequest(ahdHTTPRequestData{
		Method: "GET", Path: "/",
		Cookies: []ahdHTTPCookieEntry{
			{Name: "a", Value: "1"},
			{Name: "b", Value: "2"},
			{Name: "a", Value: "3"},
		},
	})
	first := AhdHTTPRequestCookie(class, data, "a")
	if first == nil || *first != "1" {
		t.Fatalf("cookie(a) = %v", first)
	}
	if got := AhdHTTPRequestCookie(class, data, "missing"); got != nil {
		t.Fatalf("absent cookie = %v", got)
	}
	all := AhdHTTPRequestCookieAll(class, data, "a").Snapshot()
	if len(all) != 2 || all[0] != "1" || all[1] != "3" {
		t.Fatalf("cookieAll(a) = %#v", all)
	}
	if len(AhdHTTPRequestCookieAll(class, data, "missing").Snapshot()) != 0 {
		t.Fatal("absent cookieAll must be empty")
	}
}

func TestHTTPResponseWithCookieIsImmutableAndKeepsBoth(t *testing.T) {
	class := AhdClassHTTPError
	a := AhdHTTPText(class, "a", 200)
	b := AhdHTTPResponseWithCookie(class, a, AhdHTTPCookie(class, "one", "1"))
	c := AhdHTTPResponseWithCookie(class, b, AhdHTTPCookie(class, "two", "2"))
	if len(ahdHTTPDecodeResponse(class, a).Cookies) != 0 {
		t.Fatal("original response gained a cookie")
	}
	if len(ahdHTTPDecodeResponse(class, b).Cookies) != 1 {
		t.Fatal("withCookie mutated the previous response")
	}
	if len(ahdHTTPDecodeResponse(class, c).Cookies) != 2 {
		t.Fatal("second withCookie did not keep both cookies")
	}
}

func TestHTTPMultipleSetCookieHeadersReachTheClient(t *testing.T) {
	class := AhdClassHTTPError
	encoded := AhdHTTPResponseWithCookie(class,
		AhdHTTPResponseWithCookie(class, AhdHTTPText(class, "ok", 200), AhdHTTPCookie(class, "a", "1")),
		AhdHTTPCookie(class, "b", "2"),
	)
	recorder := httptest.NewRecorder()
	ahdHTTPWriteEncoded(recorder, encoded)
	headers := recorder.Result().Header["Set-Cookie"]
	if len(headers) != 2 {
		t.Fatalf("Set-Cookie headers = %#v", headers)
	}
}

func TestHTTPSessionLifecycleAndFixation(t *testing.T) {
	class := AhdClassHTTPError
	store := AhdHTTPSessions(class, "ahd_session", 60, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	anonymous := AhdHTTPSessionStoreOpen(class, store, empty)
	committed := AhdHTTPSessionStoreCommit(class, store, anonymous, AhdHTTPText(class, "anon", 200))
	if len(ahdHTTPDecodeResponse(class, committed).Cookies) != 0 {
		t.Fatal("lazy open+commit must not create a cookie")
	}
	if ahdHTTPTestStoreSize(store) != 0 {
		t.Fatal("lazy open allocated persistent state")
	}

	session := AhdHTTPSessionStoreOpen(class, store, empty)
	session = AhdHTTPSessionSet(class, session, "flash", "1")
	preLogin := AhdHTTPSessionStoreCommit(class, store, session, AhdHTTPText(class, "anon", 200))
	beforeRotate := ahdHTTPDecodeResponse(class, preLogin).Cookies[0].Value
	session = AhdHTTPSessionStoreOpen(class, store, ahdHTTPEncodeRequest(ahdHTTPRequestData{
		Method: "GET", Path: "/",
		Cookies: []ahdHTTPCookieEntry{{Name: "ahd_session", Value: beforeRotate}},
	}))
	session = AhdHTTPSessionRotate(class, session)
	session = AhdHTTPSessionSet(class, session, "name", "Ali")
	if got := AhdHTTPSessionGet(class, session, "name"); got == nil || *got != "Ali" {
		t.Fatalf("get after set = %v", got)
	}
	if !AhdHTTPSessionHas(class, session, "name") {
		t.Fatal("has after set")
	}
	afterRotate := ahdHTTPDecodeSession(class, session).ID
	if afterRotate == "" || afterRotate == beforeRotate {
		t.Fatalf("rotate did not issue a new id: %q -> %q", beforeRotate, afterRotate)
	}
	loggedIn := AhdHTTPSessionStoreCommit(class, store, session, AhdHTTPText(class, "in", 200))
	cookies := ahdHTTPDecodeResponse(class, loggedIn).Cookies
	if len(cookies) != 1 || cookies[0].Value != afterRotate || !cookies[0].HttpOnly {
		t.Fatalf("login cookie = %#v", cookies)
	}
	oldRequest := ahdHTTPEncodeRequest(ahdHTTPRequestData{
		Method: "GET", Path: "/",
		Cookies: []ahdHTTPCookieEntry{{Name: "ahd_session", Value: beforeRotate}},
	})
	reopenedOld := AhdHTTPSessionStoreOpen(class, store, oldRequest)
	if AhdHTTPSessionGet(class, reopenedOld, "name") != nil {
		t.Fatal("old session id recovered the logged-in session")
	}
	newRequest := ahdHTTPEncodeRequest(ahdHTTPRequestData{
		Method: "GET", Path: "/",
		Cookies: []ahdHTTPCookieEntry{{Name: "ahd_session", Value: afterRotate}},
	})
	reopenedNew := AhdHTTPSessionStoreOpen(class, store, newRequest)
	if got := AhdHTTPSessionGet(class, reopenedNew, "name"); got == nil || *got != "Ali" {
		t.Fatalf("new id lost values: %v", got)
	}
}

func TestHTTPSessionRemoveClearDestroy(t *testing.T) {
	class := AhdClassHTTPError
	store := AhdHTTPSessions(class, "s", 60, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	session := AhdHTTPSessionSet(class, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "a", "1"), "b", "2")
	session = AhdHTTPSessionRemove(class, session, "a")
	if AhdHTTPSessionHas(class, session, "a") || !AhdHTTPSessionHas(class, session, "b") {
		t.Fatal("remove must delete only one key")
	}
	session = AhdHTTPSessionClear(class, session)
	if AhdHTTPSessionHas(class, session, "b") {
		t.Fatal("clear must empty values")
	}
	committed := AhdHTTPSessionStoreCommit(class, store, session, AhdHTTPText(class, "ok", 200))
	id := ahdHTTPDecodeResponse(class, committed).Cookies[0].Value
	alive := AhdHTTPSessionStoreOpen(class, store, ahdHTTPEncodeRequest(ahdHTTPRequestData{
		Method: "GET", Path: "/", Cookies: []ahdHTTPCookieEntry{{Name: "s", Value: id}},
	}))
	if AhdHTTPSessionHas(class, alive, "b") {
		t.Fatal("cleared values survived reopen")
	}
	alive = AhdHTTPSessionDestroy(class, alive)
	if AhdHTTPSessionGet(class, alive, "b") != nil || AhdHTTPSessionHas(class, alive, "b") {
		t.Fatal("destroy must hide values")
	}
	expectRaise(t, class, func() { AhdHTTPSessionSet(class, alive, "x", "1") })
	deleted := AhdHTTPSessionStoreCommit(class, store, alive, AhdHTTPText(class, "bye", 200))
	header := ahdHTTPCookieWire(ahdHTTPDecodeResponse(class, deleted).Cookies[0])
	if !strings.Contains(header, "Max-Age=0") {
		t.Fatalf("destroy cookie = %q", header)
	}
	reopened := AhdHTTPSessionStoreOpen(class, store, ahdHTTPEncodeRequest(ahdHTTPRequestData{
		Method: "GET", Path: "/", Cookies: []ahdHTTPCookieEntry{{Name: "s", Value: id}},
	}))
	if AhdHTTPSessionGet(class, reopened, "b") != nil {
		t.Fatal("destroyed id remained valid")
	}
}

func TestHTTPSessionUnknownIDsDoNotAllocate(t *testing.T) {
	class := AhdClassHTTPError
	store := AhdHTTPSessions(class, "s", 60, false, "Lax")
	for i := 0; i < 200; i++ {
		req := ahdHTTPEncodeRequest(ahdHTTPRequestData{
			Method: "GET", Path: "/",
			Cookies: []ahdHTTPCookieEntry{{Name: "s", Value: "attacker-" + strings.Repeat("x", i%20)}},
		})
		session := AhdHTTPSessionStoreOpen(class, store, req)
		AhdHTTPSessionStoreCommit(class, store, session, AhdHTTPText(class, "ok", 200))
	}
	if ahdHTTPTestStoreSize(store) != 0 {
		t.Fatalf("unknown cookies allocated %d sessions", ahdHTTPTestStoreSize(store))
	}
}

func TestHTTPSessionExpiry(t *testing.T) {
	class := AhdClassHTTPError
	defer ahdHTTPTestResetClock()
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	ahdHTTPTimeNow = func() time.Time { return now }
	store := AhdHTTPSessions(class, "s", 1, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	session := AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "k", "v")
	committed := AhdHTTPSessionStoreCommit(class, store, session, AhdHTTPText(class, "ok", 200))
	id := ahdHTTPDecodeResponse(class, committed).Cookies[0].Value
	live := AhdHTTPSessionStoreOpen(class, store, ahdHTTPEncodeRequest(ahdHTTPRequestData{
		Method: "GET", Path: "/", Cookies: []ahdHTTPCookieEntry{{Name: "s", Value: id}},
	}))
	if got := AhdHTTPSessionGet(class, live, "k"); got == nil || *got != "v" {
		t.Fatal("value missing before expiry")
	}
	ahdHTTPTimeNow = func() time.Time { return now.Add(2 * time.Second) }
	expired := AhdHTTPSessionStoreOpen(class, store, ahdHTTPEncodeRequest(ahdHTTPRequestData{
		Method: "GET", Path: "/", Cookies: []ahdHTTPCookieEntry{{Name: "s", Value: id}},
	}))
	if AhdHTTPSessionGet(class, expired, "k") != nil {
		t.Fatal("expired id restored state")
	}
	if ahdHTTPTestStoreSize(store) != 0 {
		t.Fatal("expired session was not cleaned up")
	}
}

func TestHTTPSessionStoresAreIsolated(t *testing.T) {
	class := AhdClassHTTPError
	storeA := AhdHTTPSessions(class, "a", 60, false, "Lax")
	storeB := AhdHTTPSessions(class, "b", 60, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	sessionA := AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, storeA, empty), "who", "Ali")
	sessionB := AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, storeB, empty), "who", "Mehmet")
	cookieA := ahdHTTPDecodeResponse(class, AhdHTTPSessionStoreCommit(class, storeA, sessionA, AhdHTTPText(class, "a", 200))).Cookies[0].Value
	cookieB := ahdHTTPDecodeResponse(class, AhdHTTPSessionStoreCommit(class, storeB, sessionB, AhdHTTPText(class, "b", 200))).Cookies[0].Value
	fromA := AhdHTTPSessionStoreOpen(class, storeA, ahdHTTPEncodeRequest(ahdHTTPRequestData{
		Method: "GET", Path: "/", Cookies: []ahdHTTPCookieEntry{{Name: "a", Value: cookieA}, {Name: "b", Value: cookieB}},
	}))
	fromB := AhdHTTPSessionStoreOpen(class, storeB, ahdHTTPEncodeRequest(ahdHTTPRequestData{
		Method: "GET", Path: "/", Cookies: []ahdHTTPCookieEntry{{Name: "a", Value: cookieA}, {Name: "b", Value: cookieB}},
	}))
	if got := AhdHTTPSessionGet(class, fromA, "who"); got == nil || *got != "Ali" {
		t.Fatalf("store A = %v", got)
	}
	if got := AhdHTTPSessionGet(class, fromB, "who"); got == nil || *got != "Mehmet" {
		t.Fatalf("store B = %v", got)
	}
	destroyed := AhdHTTPSessionDestroy(class, fromA)
	AhdHTTPSessionStoreCommit(class, storeA, destroyed, AhdHTTPText(class, "bye", 200))
	stillB := AhdHTTPSessionStoreOpen(class, storeB, ahdHTTPEncodeRequest(ahdHTTPRequestData{
		Method: "GET", Path: "/", Cookies: []ahdHTTPCookieEntry{{Name: "b", Value: cookieB}},
	}))
	if got := AhdHTTPSessionGet(class, stillB, "who"); got == nil || *got != "Mehmet" {
		t.Fatal("destroy in A affected B")
	}
}

func TestHTTPSessionIDsAreCookieSafeAndUnique(t *testing.T) {
	class := AhdClassHTTPError
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		id := ahdHTTPNewSessionID(class)
		if seen[id] {
			t.Fatalf("duplicate session id %q", id)
		}
		seen[id] = true
		if len(id) != 43 || !ahdHTTPCookieValueOK(id) {
			t.Fatalf("id shape %q", id)
		}
		if strings.ContainsAny(id, "+/=") {
			t.Fatalf("id is not base64url without padding: %q", id)
		}
	}
}

func TestHTTPSessionStoreRace(t *testing.T) {
	class := AhdClassHTTPError
	store := AhdHTTPSessions(class, "s", 60, false, "Lax")
	var wait sync.WaitGroup
	for i := 0; i < 32; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			defer func() { _ = recover() }()
			empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
			session := AhdHTTPSessionStoreOpen(class, store, empty)
			session = AhdHTTPSessionSet(class, session, "n", "1")
			AhdHTTPSessionStoreCommit(class, store, session, AhdHTTPText(class, "ok", 200))
			attacker := ahdHTTPEncodeRequest(ahdHTTPRequestData{
				Method: "GET", Path: "/",
				Cookies: []ahdHTTPCookieEntry{{Name: "s", Value: "rand"}},
			})
			AhdHTTPSessionStoreOpen(class, store, attacker)
		}()
	}
	wait.Wait()
}

func TestHTTPDeleteCookieClearsARealJar(t *testing.T) {
	class := AhdClassHTTPError
	mux := http.NewServeMux()
	mux.HandleFunc("/set", func(writer http.ResponseWriter, request *http.Request) {
		ahdHTTPWriteEncoded(writer, AhdHTTPResponseWithCookie(class, AhdHTTPText(class, "set", 200), AhdHTTPCookie(class, "theme", "dark")))
	})
	mux.HandleFunc("/delete", func(writer http.ResponseWriter, request *http.Request) {
		ahdHTTPWriteEncoded(writer, AhdHTTPResponseWithCookie(class, AhdHTTPText(class, "del", 200), AhdHTTPDeleteCookie(class, "theme", "/")))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}
	if _, err := client.Get(server.URL + "/set"); err != nil {
		t.Fatal(err)
	}
	if len(jar.Cookies(mustParseURL(t, server.URL+"/set"))) != 1 {
		t.Fatal("jar did not store the cookie")
	}
	if _, err := client.Get(server.URL + "/delete"); err != nil {
		t.Fatal(err)
	}
	if remaining := jar.Cookies(mustParseURL(t, server.URL+"/delete")); len(remaining) != 0 {
		t.Fatalf("jar still has %#v", remaining)
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestHTTPCookieJSONRoundTripKeepsMaxAgePointer(t *testing.T) {
	class := AhdClassHTTPError
	var cookie ahdHTTPCookieData
	if err := json.Unmarshal([]byte(AhdHTTPDeleteCookie(class, "x", "/")), &cookie); err != nil {
		t.Fatal(err)
	}
	if cookie.MaxAge == nil || *cookie.MaxAge != 0 {
		t.Fatalf("delete MaxAge = %#v", cookie.MaxAge)
	}
}
