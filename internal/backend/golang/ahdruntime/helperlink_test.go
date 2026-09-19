package ahdruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// startEchoLink starts /bin/cat as a stand-in helper: every request comes
// back unchanged, so it is a valid response carrying its own id, and a line
// written straight to the helper's input comes back as an event.
func startEchoLink(t *testing.T) *ahdHelperLink {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in helper is /bin/cat")
	}
	path, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("no cat")
	}
	link, err := ahdStartHelper(path, 1<<16)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { link.shutdown(time.Second) })
	return link
}

// TestHelperLinkRoutesEventsWhileRequestsArePending interleaves a stream of
// events with synchronous requests from another goroutine, the situation a
// callback that updates a widget creates. Every request must get its own
// response, every event must reach the queue in order, and nothing may
// deadlock.
func TestHelperLinkRoutesEventsWhileRequestsArePending(t *testing.T) {
	link := startEchoLink(t)
	var writeMu sync.Mutex
	write := func(line string) {
		writeMu.Lock()
		defer writeMu.Unlock()
		if _, err := link.input.Write([]byte(line + "\n")); err != nil {
			t.Error(err)
		}
	}
	const events = 300
	done := make(chan struct{})
	go func() {
		defer close(done)
		for index := 0; index < events; index++ {
			write(fmt.Sprintf(`{"event":"key","key":"K%d"}`, index))
		}
	}()
	var callMu sync.Mutex
	for index := 0; index < 200; index++ {
		callMu.Lock()
		writeMu.Lock()
		reply, err := link.call(map[string]any{"op": "status", "n": index}, 5*time.Second)
		writeMu.Unlock()
		callMu.Unlock()
		if err != nil {
			t.Fatalf("call %d: %v", index, err)
		}
		var decoded struct {
			N int `json:"n"`
		}
		if json.Unmarshal(reply, &decoded) != nil || decoded.N != index {
			t.Fatalf("call %d got %s", index, reply)
		}
	}
	<-done
	for index := 0; index < events; index++ {
		line, kind := link.events.next(link.exited)
		if kind != ahdEventLine || string(line) != fmt.Sprintf(`{"event":"key","key":"K%d"}`, index) {
			t.Fatalf("event %d: %q %v", index, line, kind)
		}
	}
	write(`{"event":"closed","values":[{"widget":1,"text":"x"}]}`)
	line, kind := link.events.next(link.exited)
	if kind != ahdEventClosed || len(line) == 0 {
		t.Fatalf("closed notice: %q %v", line, kind)
	}
}

func TestHelperLinkQueueIsBounded(t *testing.T) {
	queue := newAhdEventQueue()
	for index := 0; index < ahdEventQueueLimit+10; index++ {
		queue.push([]byte(`{"event":"key","key":"A"}`), false)
	}
	queue.push([]byte(`{"event":"closed"}`), true)
	if len(queue.items) != ahdEventQueueLimit+1 || !queue.closedPending() {
		t.Fatalf("queue kept %d items", len(queue.items))
	}
	if _, kind := queue.next(nil); kind != ahdEventOverflow {
		t.Fatalf("an overflowing queue reported %v", kind)
	}
}

func TestHelperLinkReportsAStoppedHelper(t *testing.T) {
	link := startEchoLink(t)
	_ = link.input.Close()
	if _, kind := link.events.next(link.exited); kind != ahdEventExited {
		t.Fatalf("a finished helper reported %v", kind)
	}
	if _, err := link.call(map[string]any{"op": "status"}, time.Second); err == nil {
		t.Fatal("a request to a finished helper succeeded")
	}
	if link.process.ProcessState == nil {
		t.Fatal("the finished helper was not reaped")
	}
}

func TestHelperLinkRejectsAMismatchedResponse(t *testing.T) {
	link := startEchoLink(t)
	// The next id will be 1; a response with another id is a protocol error.
	link.nextID = 41
	go func() { _, _ = link.input.Write([]byte(`{"id":7,"ok":true}` + "\n")) }()
	time.Sleep(50 * time.Millisecond)
	if _, err := link.call(map[string]any{"op": "status"}, time.Second); err != errAhdHelperInvalid {
		t.Fatalf("mismatched id: %v", err)
	}
}

// TestHelperLinkNeverSearchesPath starts a helper named only by a bare file
// name in the working directory. exec.Command would look such a name up in
// PATH; the link must start the file that discovery found instead.
func TestHelperLinkNeverSearchesPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in helper is /bin/cat")
	}
	directory := t.TempDir()
	stand := []byte("#!/bin/sh\nexec /bin/cat\n")
	if err := os.WriteFile(filepath.Join(directory, "ahdtesthelper"), stand, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(directory)
	t.Setenv("PATH", t.TempDir())
	link, err := ahdStartHelper("ahdtesthelper", 1024)
	if err != nil {
		t.Fatalf("a helper in the working directory was not started directly: %v", err)
	}
	defer link.shutdown(time.Second)
	if _, err := link.call(map[string]string{"op": "ping"}, 5*time.Second); err != nil {
		t.Fatal(err)
	}
}
