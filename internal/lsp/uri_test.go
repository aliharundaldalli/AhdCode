package lsp

import "testing"

func TestURIToPathSimple(t *testing.T) {
	path, err := URIToPath("file:///Users/ali/main.ahd")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/Users/ali/main.ahd" {
		t.Fatalf("path = %q", path)
	}
}

func TestURIToPathSpaces(t *testing.T) {
	path, err := URIToPath("file:///Users/ali/my%20project/main.ahd")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/Users/ali/my project/main.ahd" {
		t.Fatalf("path = %q", path)
	}
}

func TestURIToPathUnicode(t *testing.T) {
	path, err := URIToPath("file:///Users/ali/%C4%B0sim/main.ahd")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/Users/ali/İsim/main.ahd" {
		t.Fatalf("path = %q", path)
	}
}

func TestURIToPathWindowsDriveLetter(t *testing.T) {
	path, err := URIToPath("file:///C:/Users/ali/main.ahd")
	if err != nil {
		t.Fatal(err)
	}
	want := "C:\\Users\\ali\\main.ahd"
	if path != want && path != "C:/Users/ali/main.ahd" {
		t.Fatalf("path = %q, want %q (filepath.FromSlash is platform-dependent)", path, want)
	}
}

func TestURIToPathUnsupportedScheme(t *testing.T) {
	if _, err := URIToPath("untitled:Untitled-1"); err == nil {
		t.Fatal("expected an error for a non-file scheme")
	}
	if _, err := URIToPath("http://example.com/main.ahd"); err == nil {
		t.Fatal("expected an error for a non-file scheme")
	}
}

func TestURIToPathMalformed(t *testing.T) {
	if _, err := URIToPath("file://%zz"); err == nil {
		t.Fatal("expected an error for a malformed URI")
	}
}

func TestPathToURISimple(t *testing.T) {
	uri := PathToURI("/Users/ali/main.ahd")
	if uri != "file:///Users/ali/main.ahd" {
		t.Fatalf("uri = %q", uri)
	}
}

func TestPathToURISpaces(t *testing.T) {
	uri := PathToURI("/Users/ali/my project/main.ahd")
	if uri != "file:///Users/ali/my%20project/main.ahd" {
		t.Fatalf("uri = %q", uri)
	}
}

func TestPathToURIUnicode(t *testing.T) {
	uri := PathToURI("/Users/ali/İsim/main.ahd")
	if uri != "file:///Users/ali/%C4%B0sim/main.ahd" {
		t.Fatalf("uri = %q", uri)
	}
}

func TestPathToURIWindowsDriveLetter(t *testing.T) {
	uri := PathToURI("C:\\Users\\ali\\main.ahd")
	if uri != "file:///C:/Users/ali/main.ahd" {
		t.Fatalf("uri = %q", uri)
	}
}

func TestURIRoundTrip(t *testing.T) {
	for _, original := range []string{
		"/Users/ali/main.ahd",
		"/Users/ali/my project/Rapor İsmi.ahd",
	} {
		uri := PathToURI(original)
		path, err := URIToPath(uri)
		if err != nil {
			t.Fatalf("%q: %v", original, err)
		}
		if path != original {
			t.Fatalf("round trip: %q -> %q -> %q", original, uri, path)
		}
	}
}
