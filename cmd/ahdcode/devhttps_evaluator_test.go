package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/evaluator"
	"ahdcode/internal/lowering"
	"ahdcode/internal/module"
)

func TestDevHTTPSEvaluatorUsesStartupTransport(t *testing.T) {
	newDevHTTPSTestHome(t)
	files, ca, err := prepareDevHTTPS(t.TempDir()+string(os.PathSeparator)+"app.ahd", devHTTPSFiles{})
	if err != nil {
		t.Fatal(err)
	}
	caPEM, err := os.ReadFile(ca)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caPEM) {
		t.Fatal("missing test CA")
	}
	for _, tc := range []struct{ name, scheme, session, generation string }{
		{"valid startup identity", "https", "00000000000000000000000000000001", "1"},
		{"certificate variables alone", "http", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			portListener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			port := strconv.Itoa(portListener.Addr().(*net.TCPAddr).Port)
			if err := portListener.Close(); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDevHTTPSEvaluatorChild$")
			cmd.Env = append(os.Environ(), "AHDCODE_TEST_HTTPS_EVALUATOR_PORT="+port,
				"AHDCODE_DEV_SESSION="+tc.session, "AHDCODE_DEV_GENERATION="+tc.generation,
				ahdruntime.AhdDevTLSCertEnv+"="+files.cert, ahdruntime.AhdDevTLSKeyEnv+"="+files.key)
			var log bytes.Buffer
			cmd.Stdout, cmd.Stderr = &log, &log
			input, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			joined := false
			t.Cleanup(func() {
				if !joined {
					_ = input.Close()
					<-done
				}
			})

			transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}
			defer transport.CloseIdleConnections()
			client := &http.Client{Transport: transport, Timeout: time.Second}
			endpoint := tc.scheme + "://127.0.0.1:" + port + "/parity"
			ticker := time.NewTicker(20 * time.Millisecond)
			defer ticker.Stop()
			ready := false
			deadline := time.After(8 * time.Second)
			for !ready {
				select {
				case err := <-done:
					joined = true
					t.Fatalf("evaluator exited before readiness: %v\n%s", err, log.String())
				case <-deadline:
					t.Fatal("evaluator HTTP transport never became ready")
				case <-ticker.C:
					response, err := client.Get(endpoint)
					if err != nil {
						continue
					}
					body, err := io.ReadAll(response.Body)
					_ = response.Body.Close()
					if err != nil || response.StatusCode != http.StatusOK || string(body) != "parity İstanbul" || (response.TLS != nil) != (tc.scheme == "https") {
						t.Fatalf("unexpected evaluator result: %d %q %v", response.StatusCode, body, err)
					}
					ready = true
				}
			}
			_ = input.Close()
			if err := <-done; err != nil {
				joined = true
				t.Fatalf("evaluator shutdown failed: %v\n%s", err, log.String())
			}
			joined = true
		})
	}
}

func TestDevHTTPSEvaluatorChild(t *testing.T) {
	port := os.Getenv("AHDCODE_TEST_HTTPS_EVALUATOR_PORT")
	if port == "" {
		return
	}
	source := fmt.Sprintf("bring HTTP\nfrom HTTP bring (Server, Request, Response)\nhome: Function := (request: Request) -> Response { return HTTP.text(\"parity İstanbul\") }\nserver: Server := HTTP.server(\"127.0.0.1\", %s)\nserver.get(\"/parity\", home)\nserver.start()\n", port)
	workspace := module.NewInMemoryWorkspace(map[string]string{"/Main.ahd": source})
	frontend := module.NewCompiler(workspace, workspace).Compile("/Main.ahd")
	if frontend.HasErrors() {
		t.Fatalf("compile: %+v", frontend.Diagnostics)
	}
	lowered := lowering.LowerCompilation(frontend)
	if lowered.HasErrors() {
		t.Fatalf("lowering: %+v", lowered.Diagnostics)
	}
	session := evaluator.New(bufio.NewReader(strings.NewReader("")), io.Discard, t.TempDir())
	finished := make(chan evaluator.ExecutionResult, 1)
	go func() { finished <- session.Execute(lowered.Compilation, 0) }()
	go func() {
		_, _ = io.Copy(io.Discard, os.Stdin)
		number, _ := strconv.Atoi(port)
		_ = ahdruntime.AhdHTTPShutdownAddress("127.0.0.1", number)
	}()
	select {
	case result := <-finished:
		if result.Failure != nil {
			t.Fatalf("evaluator HTTP startup: %s", result.Failure.Message)
		}
	case <-time.After(12 * time.Second):
		t.Fatal("evaluator HTTPS server did not return")
	}
}
