//go:build darwin

package initweb

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

func TestReadSecretDisablesEchoOnInteractiveTerminal(t *testing.T) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	if err := unix.IoctlSetInt(int(master.Fd()), unix.TIOCPTYGRANT, 0); err != nil {
		t.Fatal(err)
	}
	if err := unix.IoctlSetInt(int(master.Fd()), unix.TIOCPTYUNLK, 0); err != nil {
		t.Fatal(err)
	}
	var buf [128]byte
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(master.Fd()),
		uintptr(unix.TIOCPTYGNAME),
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if errno != 0 {
		t.Fatalf("TIOCPTYGNAME: %v", errno)
	}
	n := 0
	for n < len(buf) && buf[n] != 0 {
		n++
	}
	slave, err := os.OpenFile(string(buf[:n]), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()

	before, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TIOCGETA)
	if err != nil {
		t.Fatal(err)
	}
	if before.Lflag&unix.ECHO == 0 {
		t.Fatal("test PTY started without ECHO")
	}

	done := make(chan error, 1)
	var out bytes.Buffer
	go func() {
		_, err := readSecret(bufio.NewReader(slave), &out, slave)
		done <- err
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		state, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TIOCGETA)
		if err == nil && state.Lflag&unix.ECHO == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("ECHO was not disabled on the interactive terminal")
		}
		time.Sleep(5 * time.Millisecond)
	}

	if _, err := io.WriteString(master, "terminal-secret\n"); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	after, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TIOCGETA)
	if err != nil {
		t.Fatal(err)
	}
	if after.Lflag&unix.ECHO == 0 {
		t.Fatal("ECHO was not restored after the secret was read")
	}
}
