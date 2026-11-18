package evaluator

import (
	"runtime"

	"ahdcode/internal/backend/golang/ahdruntime"
)

// The interpreter is one Session. A background task runs on another
// goroutine, so evaluation takes this lock. It is reentrant for the
// goroutine that holds it: a function call from inside a function does not
// lock twice. Window.wait and GUI.checkpoint release it, which is when the
// other goroutine may run.

func evalGoroutine() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	var id int64
	for index := len("goroutine "); index < n; index++ {
		if buf[index] < '0' || buf[index] > '9' {
			break
		}
		id = id*10 + int64(buf[index]-'0')
	}
	return id
}

func (session *Session) enter() {
	gid := evalGoroutine()
	if session.owner.Load() == gid {
		session.depth.Add(1)
		return
	}
	session.mu.Lock()
	session.owner.Store(gid)
	session.depth.Store(1)
}

func (session *Session) leave() {
	if session.owner.Load() != evalGoroutine() {
		return
	}
	if session.depth.Add(-1) == 0 {
		session.owner.Store(0)
		session.mu.Unlock()
	}
}

// yield releases the interpreter lock. The returned depth is handed back to
// resume by the same goroutine.
func (session *Session) yield() int64 {
	if session.owner.Load() != evalGoroutine() {
		return 0
	}
	depth := session.depth.Swap(0)
	session.owner.Store(0)
	if depth > 0 {
		session.mu.Unlock()
	}
	return depth
}

func (session *Session) resume(depth int64) {
	if depth == 0 {
		return
	}
	session.mu.Lock()
	session.owner.Store(evalGoroutine())
	session.depth.Store(depth)
}

// guiCheckpoint yields the interpreter around the runtime's checkpoint so a
// GUI callback can run while a task is in the middle of its function.
func (session *Session) guiCheckpoint() bool {
	if !ahdruntime.AhdGUIInTask() {
		return true
	}
	depth := session.yield()
	ok := ahdruntime.AhdGUICheckpoint()
	session.resume(depth)
	return ok
}
