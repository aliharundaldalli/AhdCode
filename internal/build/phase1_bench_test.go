package build

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	backend "ahdcode/internal/backend/golang"
	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/lowering"
	"ahdcode/internal/lsp"
	"ahdcode/internal/module"
)

// BenchmarkPhase1 measures an in-process frontend and an actual LSP exchange.
// Native subprocess allocations are outside Go's benchmark allocation totals.
func BenchmarkPhase1(b *testing.B) {
	for _, fixture := range []struct {
		name  string
		parts int
	}{{"Small", 1}, {"Medium", 24}} {
		b.Run(fixture.name, func(b *testing.B) {
			entry, contents := phase1Fixture(b, fixture.parts)
			frontend := func() module.CompilationResult {
				value := module.NewCompiler(module.FileResolver{}, module.FileLoader{}).Compile(entry)
				if value.HasErrors() {
					b.Fatalf("frontend: %+v", value.Diagnostics)
				}
				return value
			}
			checked := frontend()
			irResult := lowering.LowerCompilation(checked)
			if irResult.HasErrors() || irResult.Compilation == nil {
				b.Fatal(irResult.Diagnostics)
			}
			program, diags := backend.Generate(irResult.Compilation)
			requireBenchmarkStage(b, diags)
			b.Run("Frontend", func(b *testing.B) {
				for b.Loop() {
					frontend()
				}
			})
			b.Run("Lowering", func(b *testing.B) {
				for b.Loop() {
					if item := lowering.LowerCompilation(checked); item.HasErrors() {
						b.Fatal(item.Diagnostics)
					}
				}
			})
			b.Run("GenerateGo", func(b *testing.B) {
				for b.Loop() {
					_, issues := backend.Generate(irResult.Compilation)
					requireBenchmarkStage(b, issues)
				}
			})
			b.Run("Workspace", func(b *testing.B) {
				for b.Loop() {
					workspace, issues := NewWorkspace(program)
					requireBenchmarkStage(b, issues)
					workspace.Close()
				}
			})
			b.Run("NativeBuild", func(b *testing.B) {
				if os.Getenv("AHDCODE_BENCH_COLD_GO") == "1" {
					b.Setenv("GOCACHE", b.TempDir())
				}
				workspace, issues := NewWorkspace(program)
				requireBenchmarkStage(b, issues)
				defer workspace.Close()
				output := ExecutablePath(filepath.Join(b.TempDir(), "fixture"))
				for b.Loop() {
					requireBenchmarkStage(b, workspace.BuildExecutable(output))
				}
			})
			for _, mode := range []struct{ label, disabled string }{{"RunWithoutExecutableCache", "1"}, {"RunWithWarmExecutableCache", ""}} {
				b.Run(mode.label, func(b *testing.B) {
					b.Setenv(disableRunCache, mode.disabled)
					run := func() {
						code, result := RunProgramIO(entry, nil, strings.NewReader(""), io.Discard, io.Discard)
						requireBenchmarkStage(b, result.Diagnostics)
						if code != 0 {
							b.Fatalf("program exit %d", code)
						}
					}
					if mode.disabled == "" {
						run()
					}
					for b.Loop() {
						run()
					}
				})
			}
			b.Run("LSPEdit", func(b *testing.B) {
				client := newBenchmarkLSP(b, entry, contents)
				for index := 1; b.Loop(); index++ {
					changed := contents + fmt.Sprintf("\n// change %d\n", index)
					client.send("textDocument/didChange", 0, map[string]any{
						"textDocument":   map[string]any{"uri": client.uri, "version": index + 1},
						"contentChanges": []map[string]string{{"text": changed}},
					})
					client.wait(0)
				}
			})
			b.Run("LSPReferences", func(b *testing.B) {
				client := newBenchmarkLSP(b, entry, contents)
				var latencies []time.Duration
				for index := 2; b.Loop(); index++ {
					start := time.Now()
					client.send("textDocument/references", index, map[string]any{
						"textDocument": map[string]string{"uri": client.uri},
						"position":     map[string]int{"line": fixture.parts, "character": 1},
						"context":      map[string]bool{"includeDeclaration": true},
					})
					reply := client.wait(index)
					var locations []json.RawMessage
					if err := json.Unmarshal(reply, &locations); err != nil || len(locations) < 2 {
						b.Fatalf("unexpected references result: %s", reply)
					}
					latencies = append(latencies, time.Since(start))
				}
				slices.Sort(latencies)
				count := len(latencies)
				median := (latencies[(count-1)/2] + latencies[count/2]) / 2
				p95 := latencies[(95*count+99)/100-1]
				b.ReportMetric(float64(median)/float64(time.Millisecond), "median-ms")
				b.ReportMetric(float64(p95)/float64(time.Millisecond), "p95-ms")
			})
		})
	}
	b.Run("SQLite", func(b *testing.B) {
		if os.Getenv("AHDCODE_SQLITE_RUNTIME") == "" {
			b.Fatal("set AHDCODE_SQLITE_RUNTIME to the benchmark build of ahdsqlite")
		}
		path := filepath.Join(b.TempDir(), "sample.db")
		open := func() string {
			db, err := ahdruntime.SQLiteOpen(path)
			if err != nil {
				b.Fatal(err)
			}
			return db
		}
		close := func(db string) {
			if err := ahdruntime.SQLiteClose(db); err != nil {
				b.Fatal(err)
			}
		}
		db := open()
		defer close(db)
		for _, statement := range []string{
			"CREATE TABLE samples(id INTEGER PRIMARY KEY, value TEXT)",
			"WITH RECURSIVE n(i) AS (VALUES(1) UNION ALL SELECT i+1 FROM n WHERE i<1000) INSERT INTO samples SELECT i, 'record' FROM n",
		} {
			if _, err := ahdruntime.SQLiteExecute(db, statement, nil); err != nil {
				b.Fatal(err)
			}
		}
		b.Run("OpenClose", func(b *testing.B) {
			for b.Loop() {
				close(open())
			}
		})
		b.Run("PointQuery", func(b *testing.B) {
			bind := []string{ahdruntime.SQLiteFromInt(500)}
			for b.Loop() {
				if _, rows, err := ahdruntime.SQLiteQuery(db, "SELECT value FROM samples WHERE id = ?", bind); err != nil || len(rows) != 1 {
					b.Fatalf("SQLite point query: rows=%d err=%v", len(rows), err)
				}
			}
		})
	})
}

func phase1Fixture(b *testing.B, parts int) (string, string) {
	b.Helper()
	dir := b.TempDir()
	var body strings.Builder
	write := func(name, text string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o600); err != nil {
			b.Fatal(err)
		}
	}
	for index := range parts {
		name := fmt.Sprintf("Part%02d.ahd", index)
		fmt.Fprintf(&body, "require(%q)\n", name)
		var source strings.Builder
		for value := range 12 {
			fmt.Fprintf(&source, "value%d_%d: Int := %d\n", index, value, value)
		}
		write(name, source.String())
	}
	body.WriteString("score: Int := 42\n")
	if parts > 1 {
		body.WriteString("bring Web\nwrite(Web.render(Web.UI.p(\"fixture {score}\")))\n")
	} else {
		body.WriteString("write(score)\n")
	}
	write("main.ahd", body.String())
	return filepath.Join(dir, "main.ahd"), body.String()
}

func requireBenchmarkStage(b *testing.B, items []diagnostics.Diagnostic) {
	b.Helper()
	if len(items) != 0 {
		b.Fatal(diagnosticText(items))
	}
}

type benchmarkLSP struct {
	b      *testing.B
	uri    string
	input  io.Writer
	output <-chan []byte
}

func newBenchmarkLSP(b *testing.B, path, contents string) benchmarkLSP {
	b.Helper()
	in, input := io.Pipe()
	output, out := io.Pipe()
	done := make(chan error, 1)
	go func() {
		defer out.Close()
		done <- lsp.NewServer(io.Discard).Run(in, out)
	}()
	b.Cleanup(func() {
		_ = input.Close()
		if err := <-done; err != nil {
			b.Errorf("LSP close: %v", err)
		}
		_ = output.Close()
	})
	replies := make(chan []byte, 128)
	go func() {
		defer close(replies)
		reader := textproto.NewReader(bufio.NewReader(output))
		for {
			headers, err := reader.ReadMIMEHeader()
			if err != nil {
				return
			}
			size, err := strconv.Atoi(headers.Get("Content-Length"))
			if err != nil || size < 0 {
				return
			}
			body := make([]byte, size)
			if _, err := io.ReadFull(reader.R, body); err != nil {
				return
			}
			replies <- body
		}
	}()
	client := benchmarkLSP{b: b, uri: lsp.PathToURI(path), input: input, output: replies}
	client.send("initialize", 1, map[string]string{"rootUri": lsp.PathToURI(filepath.Dir(path))})
	client.wait(1)
	client.send("textDocument/didOpen", 0, map[string]any{
		"textDocument": map[string]any{"uri": client.uri, "languageId": "ahdcode", "version": 1, "text": contents},
	})
	client.wait(0)
	return client
}

func (client benchmarkLSP) send(method string, id int, params any) {
	data := map[string]any{"jsonrpc": "2.0", "method": method, "params": params}
	if id > 0 {
		data["id"] = id
	}
	body, err := json.Marshal(data)
	if err != nil {
		client.b.Fatal(err)
	}
	if _, err := fmt.Fprintf(client.input, "Content-Length: %d\r\n\r\n%s", len(body), body); err != nil {
		client.b.Fatal(err)
	}
}

func (client benchmarkLSP) wait(id int) json.RawMessage {
	for body := range client.output {
		var message struct {
			ID     int             `json:"id"`
			Method string          `json:"method"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
			Params struct {
				URI         string            `json:"uri"`
				Diagnostics []json.RawMessage `json:"diagnostics"`
			} `json:"params"`
		}
		if err := json.Unmarshal(body, &message); err != nil {
			client.b.Fatal(err)
		}
		if id != 0 && id == message.ID {
			if len(message.Error) != 0 {
				client.b.Fatalf("LSP error: %s", message.Error)
			}
			return message.Result
		}
		if id == 0 && message.Method == "textDocument/publishDiagnostics" && message.Params.URI == client.uri {
			if len(message.Params.Diagnostics) != 0 {
				client.b.Fatalf("fixture diagnostics: %s", body)
			}
			return nil
		}
	}
	client.b.Fatal("LSP closed before sending its response")
	return nil
}
