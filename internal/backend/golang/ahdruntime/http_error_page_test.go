package ahdruntime

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHTTPCustomErrorPagesKeepTheirStatus(t *testing.T) {
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/", func(string) string {
			return AhdHTTPHTML(AhdClassHTTPError, "<p>Ana sayfa</p>", 200)
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/boom", func(string) string {
			panic("secret stack /internal/config")
		})
		AhdHTTPServerNotFound(AhdClassHTTPError, handle, func(string) string {
			return AhdHTTPHTML(AhdClassHTTPError, "<p>Sayfa bulunamadı</p>", 200)
		})
		AhdHTTPServerInternalError(AhdClassHTTPError, handle, func(string) string {
			return AhdHTTPHTML(AhdClassHTTPError, "<p>Sunucu hatası</p>", 200)
		})
	})
	missing, err := http.Get(base + "/yok")
	if err != nil {
		t.Fatal(err)
	}
	defer missing.Body.Close()
	body, _ := io.ReadAll(missing.Body)
	if missing.StatusCode != 404 || !strings.Contains(string(body), "Sayfa bulunamadı") {
		t.Fatalf("404 = %d %s", missing.StatusCode, body)
	}
	failed, err := http.Get(base + "/boom")
	if err != nil {
		t.Fatal(err)
	}
	defer failed.Body.Close()
	body, _ = io.ReadAll(failed.Body)
	if failed.StatusCode != 500 || !strings.Contains(string(body), "Sunucu hatası") {
		t.Fatalf("500 = %d %s", failed.StatusCode, body)
	}
	if strings.Contains(string(body), "secret") || strings.Contains(string(body), "/internal") {
		t.Fatalf("error page leaked details: %s", body)
	}
}

func TestHTTPErrorPageDropsLocation(t *testing.T) {
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerNotFound(AhdClassHTTPError, handle, func(string) string {
			return AhdHTTPRedirect(AhdClassHTTPError, "https://example.invalid/elsewhere", 302)
		})
		AhdHTTPServerInternalError(AhdClassHTTPError, handle, func(string) string {
			return AhdHTTPRedirect(AhdClassHTTPError, "https://example.invalid/fail", 303)
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/boom", func(string) string {
			panic("x")
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/go", func(string) string {
			return AhdHTTPRedirect(AhdClassHTTPError, "https://example.invalid/ok", 303)
		})
	})
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	missing, err := client.Get(base + "/yok")
	if err != nil {
		t.Fatal(err)
	}
	defer missing.Body.Close()
	if missing.StatusCode != 404 || missing.Header.Get("Location") != "" {
		t.Fatalf("404 redirect kept: %d %v", missing.StatusCode, missing.Header)
	}
	failed, err := client.Get(base + "/boom")
	if err != nil {
		t.Fatal(err)
	}
	defer failed.Body.Close()
	if failed.StatusCode != 500 || failed.Header.Get("Location") != "" {
		t.Fatalf("500 redirect kept: %d %v", failed.StatusCode, failed.Header)
	}
	ok, err := client.Get(base + "/go")
	if err != nil {
		t.Fatal(err)
	}
	defer ok.Body.Close()
	if ok.StatusCode != 303 || ok.Header.Get("Location") != "https://example.invalid/ok" {
		t.Fatalf("normal redirect = %d %v", ok.StatusCode, ok.Header)
	}
}

func TestHTTPErrorPageFailureFallsBack(t *testing.T) {
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/boom", func(string) string {
			panic("first")
		})
		AhdHTTPServerNotFound(AhdClassHTTPError, handle, func(string) string {
			panic("missing page failed")
		})
		AhdHTTPServerInternalError(AhdClassHTTPError, handle, func(string) string {
			panic("error page failed /secret")
		})
	})
	missing, err := http.Get(base + "/yok")
	if err != nil {
		t.Fatal(err)
	}
	defer missing.Body.Close()
	body, _ := io.ReadAll(missing.Body)
	if missing.StatusCode != 404 || strings.TrimSpace(string(body)) != "Not Found" {
		t.Fatalf("fallback 404 = %d %q", missing.StatusCode, body)
	}
	failed, err := http.Get(base + "/boom")
	if err != nil {
		t.Fatal(err)
	}
	defer failed.Body.Close()
	body, _ = io.ReadAll(failed.Body)
	if failed.StatusCode != 500 || strings.TrimSpace(string(body)) != "Internal Server Error" || strings.Contains(string(body), "secret") {
		t.Fatalf("fallback 500 = %d %q", failed.StatusCode, body)
	}
	again, err := http.Get(base + "/boom")
	if err != nil {
		t.Fatal(err)
	}
	defer again.Body.Close()
	if again.StatusCode != 500 {
		t.Fatalf("second 500 = %d", again.StatusCode)
	}
}
