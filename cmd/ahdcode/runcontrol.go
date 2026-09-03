package main

import (
	"bufio"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// The run control channel is how `ahdcode kill` stops an application without
// ever signalling a process id it read out of a file.
//
// A process id in a descriptor proves nothing: anyone who can write the file
// can name an unrelated process, and an operating system reuses ids, so a
// stale descriptor can come to name something else entirely. Signalling that
// id would terminate whatever now holds it.
//
// So the descriptor carries no authority. The live `ahdcode run` supervisor
// listens on a loopback-only port and holds a 256-bit random token. `kill`
// must connect to that port and present the token; only then does the
// supervisor terminate the child process it started and owns. A forged
// descriptor has no supervisor to answer it, and a wrong token is refused, so
// in both cases nothing is signalled at all.
//
// This is not a network API: the host is fixed at 127.0.0.1 and never read
// from the descriptor, the protocol is one strict line carrying an
// AhdCode-specific identity so an unrelated local service is never mistaken
// for a supervisor, and the token is capability data that is never printed,
// logged, or included in an error.
const (
	runControlMagic     = "ahdcode-run-control/2"
	runControlHost      = "127.0.0.1"
	runControlTokenSize = 32 // 256 bits
	runControlTimeout   = 3 * time.Second

	runControlPing      = "ping"
	runControlStop      = "stop"
	runControlForceStop = "force-stop"
)

var errRunControlUnreachable = errors.New("no AhdCode run supervisor answered")

func newRunControlToken() (string, error) {
	buffer := make([]byte, runControlTokenSize)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

// validRunControlToken checks only the shape of a token. It never reports the
// token itself.
func validRunControlToken(token string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil && len(decoded) == runControlTokenSize
}

// runControlServer is the supervisor half. It owns the running child process
// and is the only thing in AhdCode that terminates it.
type runControlServer struct {
	listener net.Listener
	token    string

	mutex          sync.Mutex
	child          *os.Process
	descriptorPath string
	stopped        bool
}

// startRunControlServer binds a loopback-only listener on an ephemeral port.
// The child is attached later, once it has actually started.
func startRunControlServer() (*runControlServer, error) {
	token, err := newRunControlToken()
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(runControlHost, "0"))
	if err != nil {
		return nil, err
	}
	server := &runControlServer{listener: listener, token: token}
	go server.serve()
	return server, nil
}

func (server *runControlServer) port() int {
	address, ok := server.listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0
	}
	return address.Port
}

func (server *runControlServer) attach(child *os.Process) {
	server.mutex.Lock()
	server.child = child
	server.mutex.Unlock()
}

// ownDescriptor records the descriptor this supervisor published, so a
// successful stop can retire it immediately rather than leaving the caller to
// wait for this process to exit.
func (server *runControlServer) ownDescriptor(path string) {
	server.mutex.Lock()
	server.descriptorPath = path
	server.mutex.Unlock()
}

func (server *runControlServer) close() {
	_ = server.listener.Close()
}

func (server *runControlServer) serve() {
	for {
		connection, err := server.listener.Accept()
		if err != nil {
			return
		}
		go server.handle(connection)
	}
}

// handle reads exactly one control line, authenticates it, and acts. Anything
// that does not parse as this protocol is refused without touching the child.
func (server *runControlServer) handle(connection net.Conn) {
	defer func() { _ = connection.Close() }()
	_ = connection.SetDeadline(time.Now().Add(runControlTimeout))

	line, err := bufio.NewReader(connection).ReadString('\n')
	if err != nil {
		return
	}
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) != 3 || fields[0] != runControlMagic {
		server.reply(connection, "unrecognized request")
		return
	}
	action, token := fields[1], fields[2]
	// Constant-time comparison: a token is a secret, not a string to match.
	if subtle.ConstantTimeCompare([]byte(token), []byte(server.token)) != 1 {
		server.reply(connection, "authentication failed")
		return
	}
	switch action {
	case runControlPing:
		server.replyOK(connection)
	case runControlStop, runControlForceStop:
		if err := server.terminateChild(action == runControlForceStop); err != nil {
			server.reply(connection, "could not stop the application")
			return
		}
		server.retireDescriptor()
		server.replyOK(connection)
	default:
		server.reply(connection, "unsupported action")
	}
}

// terminateChild stops the process this supervisor started. It is reached
// only after the caller authenticated, and it can only ever affect this
// supervisor's own child.
func (server *runControlServer) terminateChild(force bool) error {
	server.mutex.Lock()
	child := server.child
	if child == nil {
		server.mutex.Unlock()
		return errors.New("no child is running")
	}
	server.stopped = true
	server.mutex.Unlock()
	return terminateOwnedProcess(child, force)
}

// retireDescriptor removes this supervisor's own descriptor once its child
// has been stopped.
func (server *runControlServer) retireDescriptor() {
	server.mutex.Lock()
	path := server.descriptorPath
	server.mutex.Unlock()
	if path == "" {
		return
	}
	removeOwnRunDescriptor(path, server.port())
}

func (server *runControlServer) replyOK(connection net.Conn) {
	_, _ = fmt.Fprintf(connection, "%s ok\n", runControlMagic)
}

func (server *runControlServer) reply(connection net.Conn, reason string) {
	_, _ = fmt.Fprintf(connection, "%s error %s\n", runControlMagic, reason)
}

// requestRunControl is the client half used by `ahdcode kill`. The host is
// fixed here and never taken from the descriptor, so a descriptor can never
// redirect control traffic anywhere but this machine's loopback interface.
func requestRunControl(port int, token, action string) error {
	if port <= 0 || port > 65535 {
		return errors.New("invalid control port")
	}
	if !validRunControlToken(token) {
		return errors.New("invalid control token")
	}
	address := net.JoinHostPort(runControlHost, fmt.Sprintf("%d", port))
	connection, err := net.DialTimeout("tcp", address, runControlTimeout)
	if err != nil {
		return errRunControlUnreachable
	}
	defer func() { _ = connection.Close() }()
	_ = connection.SetDeadline(time.Now().Add(runControlTimeout))

	if _, err := fmt.Fprintf(connection, "%s %s %s\n", runControlMagic, action, token); err != nil {
		return errRunControlUnreachable
	}
	line, err := bufio.NewReader(connection).ReadString('\n')
	if err != nil {
		return errRunControlUnreachable
	}
	fields := strings.Fields(strings.TrimSpace(line))
	// Strict identity check: an unrelated local service listening on this
	// port cannot pass for an AhdCode run supervisor.
	if len(fields) < 2 || fields[0] != runControlMagic {
		return errRunControlUnreachable
	}
	if fields[1] != "ok" {
		return errors.New("the AhdCode run supervisor refused the request")
	}
	return nil
}
