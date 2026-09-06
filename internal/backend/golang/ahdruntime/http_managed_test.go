package ahdruntime

import (
	"io"
	"net/http"
	"testing"
	"time"
)

func TestHTTPManagedExposesOnlyDeclaredAssets(t *testing.T) {
	t.Cleanup(ahdHTMLResetExposedAssets)
	root := t.TempDir()
	writeStaticFixture(t, root, "declared.css", []byte("body{}"))
	writeStaticFixture(t, root, "declared.js", []byte("console.log(1)"))
	writeStaticFixture(t, root, "secret.txt", []byte("nope"))
	writeStaticFixture(t, root, "unused.css", []byte(".x{}"))
	AhdHTMLAsset(AhdClassHTMLError, "stylesheet", "declared.css")
	AhdHTMLAsset(AhdClassHTMLError, "script", "declared.js")

	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerManaged(AhdClassHTMLError, handle, "/assets", root)
	})
	client := &http.Client{Timeout: 2 * time.Second}

	mustStatus := func(path string, want int) {
		t.Helper()
		response, err := client.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if response.StatusCode != want {
			t.Fatalf("%s status = %d body %q; want %d", path, response.StatusCode, body, want)
		}
	}
	mustStatus("/assets/declared.css", 200)
	mustStatus("/assets/declared.js", 200)
	mustStatus("/assets/secret.txt", 404)
	mustStatus("/assets/unused.css", 404)
	mustStatus("/assets/", 404)
	mustStatus("/assets/../secret.txt", 404)
	mustStatus("/assets/.hidden", 404)
}

func TestHTTPManagedDoesNotWeakenStaticMounts(t *testing.T) {
	root := t.TempDir()
	writeStaticFixture(t, root, "public.txt", []byte("ok"))
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerStatic(AhdClassHTMLError, handle, "/static", root)
	})
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(base + "/static/public.txt")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("explicit static mount status = %d", response.StatusCode)
	}
}
