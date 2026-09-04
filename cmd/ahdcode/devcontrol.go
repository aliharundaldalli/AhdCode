package main

import (
	"bufio"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// The dev control channel is runcontrol.go's loopback-token protocol reused
// for `ahdcode dev`, with its own magic string so a `.run` client can never
// mistake a dev controller for a plain run supervisor or vice versa. It
// authenticates a caller; it does not by itself decide what "stop" does --
// that is the single event-loop goroutine in dev.go, reached through the
// onStop callback below, which is the only thing that ever touches
// controller/child state. This keeps the same invariant runcontrol.go
// documents: a descriptor carries no authority over a pid, only the ability
// to ask an authenticated, already-identified supervisor to act.
const (
	devControlMagic   = "ahdcode-dev-control/1"
	devControlTimeout = 3 * time.Second

	devControlPing      = "ping"
	devControlStop      = "stop"
	devControlForceStop = "force-stop"
)

// devControlServer is the dev controller's listener half. Unlike
// runControlServer, "stop"/"force-stop" do not just end a child and reply --
// they ask the owning event loop to run the full shutdown sequence (stop
// watching, stop the current child, clean temporary artifacts, remove the
// descriptor) and only reply once that has actually finished, so a client
// that receives "ok" can trust the session is genuinely gone.
type devControlServer struct {
	listener net.Listener
	token    string
	// onStop is set once, before the server starts accepting requests that
	// matter (ping is harmless before that). It must block until shutdown is
	// complete.
	onStop func(force bool)
}

func startDevControlServer() (*devControlServer, error) {
	token, err := newRunControlToken()
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(runControlHost, "0"))
	if err != nil {
		return nil, err
	}
	server := &devControlServer{listener: listener, token: token}
	go server.serve()
	return server, nil
}

func (server *devControlServer) port() int {
	address, ok := server.listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0
	}
	return address.Port
}

// setOnStop installs the shutdown callback. Call it once, before publishing
// the dev descriptor, so no authenticated stop request can ever arrive
// before there is something to run it.
func (server *devControlServer) setOnStop(onStop func(force bool)) {
	server.onStop = onStop
}

func (server *devControlServer) close() {
	_ = server.listener.Close()
}

func (server *devControlServer) serve() {
	for {
		connection, err := server.listener.Accept()
		if err != nil {
			return
		}
		go server.handle(connection)
	}
}

func (server *devControlServer) handle(connection net.Conn) {
	defer func() { _ = connection.Close() }()

	// The request line itself must arrive promptly; once it has, the
	// deadline is cleared before any action that can legitimately take a
	// while (a graceful shutdown up to its grace period), so that duration
	// is bounded by dev.go's own timer, not silently cut off here.
	_ = connection.SetReadDeadline(time.Now().Add(devControlTimeout))
	line, err := bufio.NewReader(connection).ReadString('\n')
	if err != nil {
		return
	}
	_ = connection.SetDeadline(time.Time{})
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) != 3 || fields[0] != devControlMagic {
		server.reply(connection, "unrecognized request")
		return
	}
	action, token := fields[1], fields[2]
	if subtle.ConstantTimeCompare([]byte(token), []byte(server.token)) != 1 {
		server.reply(connection, "authentication failed")
		return
	}
	switch action {
	case devControlPing:
		// Deliberately no deadline on ping: a live controller must answer
		// identity checks even while a build or a graceful child shutdown
		// is in progress, so claimDevFile/devDescriptorIsLive never mistake
		// a busy session for a dead one. stop/force-stop below wait for
		// full shutdown and so stay undeadlined for the same reason -- a
		// slow but genuine shutdown must not read as "unreachable".
		server.replyOK(connection)
	case devControlStop, devControlForceStop:
		if server.onStop == nil {
			server.reply(connection, "controller is not ready yet")
			return
		}
		server.onStop(action == devControlForceStop)
		server.replyOK(connection)
	default:
		server.reply(connection, "unsupported action")
	}
}

func (server *devControlServer) replyOK(connection net.Conn) {
	_, _ = fmt.Fprintf(connection, "%s ok\n", devControlMagic)
}

func (server *devControlServer) reply(connection net.Conn, reason string) {
	_, _ = fmt.Fprintf(connection, "%s error %s\n", devControlMagic, reason)
}

var errDevControlUnreachable = errors.New("no AhdCode dev controller answered")

// requestDevControl is the client half used by `ahdcode stop`/`kill` for a
// .dev target. Mirrors requestRunControl exactly, including the fixed
// loopback host and the strict identity check on the reply.
func requestDevControl(port int, token, action string) error {
	if port <= 0 || port > 65535 {
		return errors.New("invalid control port")
	}
	if !validRunControlToken(token) {
		return errors.New("invalid control token")
	}
	address := net.JoinHostPort(runControlHost, fmt.Sprintf("%d", port))
	connection, err := net.DialTimeout("tcp", address, devControlTimeout)
	if err != nil {
		return errDevControlUnreachable
	}
	defer func() { _ = connection.Close() }()
	_ = connection.SetWriteDeadline(time.Now().Add(devControlTimeout))
	if _, err := fmt.Fprintf(connection, "%s %s %s\n", devControlMagic, action, token); err != nil {
		return errDevControlUnreachable
	}
	// A stop/force-stop reply can legitimately take up to the controller's
	// own shutdown grace period to arrive (it replies only once the child
	// has actually exited), plus a margin for the final cleanup and reply
	// write; the read deadline is set generously rather than removed
	// entirely, so a controller that hangs cannot block the client forever.
	_ = connection.SetReadDeadline(time.Now().Add(shutdownGracePeriod + 5*time.Second))
	line, err := bufio.NewReader(connection).ReadString('\n')
	if err != nil {
		return errDevControlUnreachable
	}
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 || fields[0] != devControlMagic {
		return errDevControlUnreachable
	}
	if fields[1] != "ok" {
		return errors.New("the AhdCode dev controller refused the request")
	}
	return nil
}
