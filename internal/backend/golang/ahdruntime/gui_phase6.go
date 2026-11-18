package ahdruntime

import (
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// GUI timers follow the Graphics timer rules and use the same event queue as
// clicks and keys. A headless Window has no clock: its timers run only when
// the script sends a tick. A real Window's timers are goroutines that post
// one tick at a time; a callback that outlasts its interval skips the turns
// it missed, so a slow handler cannot pile up a backlog.

const (
	ahdGUIMinTimer       = int64(1)
	ahdGUIMaxTimer       = int64(24 * 60 * 60 * 1000)
	ahdGUIMaxTimers      = 64
	ahdGUIMaxCanvasSide  = 4096
	ahdGUIMaxCanvasOps   = 4096
	ahdGUIMaxCanvasCoord = 100000.0
)

// ahdGUICallbackLock is the GUI turn. Callbacks take it for one event, and a
// background task takes it for the whole function except at GUI.checkpoint.
var ahdGUICallbackLock sync.Mutex

// ahdGUITaskLock lets one task run at a time even while it is yielding.
var ahdGUITaskLock sync.Mutex

// ahdGUITaskOwner is the goroutine inside a task function. GUI requests from
// that goroutine are rejected.
var ahdGUITaskOwner atomic.Int64

var ahdGUICurrent atomic.Pointer[ahdGUITask]

// ahdGUINextTask gives every task a number that is unique in the process,
// so two Windows can each run a task without sharing an identifier.
var ahdGUINextTask atomic.Int64

type ahdGUITimer struct {
	handler func()
	stop    chan struct{}
	pending bool
}

// AhdGUIEvery starts a timer that runs handler every interval milliseconds
// while the program waits on the Window, and returns its number for
// Window.cancel. The interval is 1 millisecond to one day. A Window holds at
// most 64 timers. A headless Window does not start a clock.
func AhdGUIEvery(handle int64, milliseconds int64, handler func()) (int64, string) {
	if handler == nil {
		return 0, "Window.every needs a Function"
	}
	if milliseconds < ahdGUIMinTimer || milliseconds > ahdGUIMaxTimer {
		return 0, fmt.Sprintf("a timer interval must be between 1 millisecond and one day, not %d", milliseconds)
	}
	window, problem := ahdGUIWindowOf(handle)
	if problem != "" {
		return 0, problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return 0, ahdGUIClosedMessage
	}
	if len(window.timers) >= ahdGUIMaxTimers {
		return 0, fmt.Sprintf("a Window runs at most %d timers at a time; cancel one first", ahdGUIMaxTimers)
	}
	if window.timers == nil {
		window.timers = map[int64]*ahdGUITimer{}
	}
	window.nextTimer++
	id := window.nextTimer
	timer := &ahdGUITimer{handler: handler, stop: make(chan struct{})}
	window.timers[id] = timer
	window.timerOrder = append(window.timerOrder, id)
	if !window.headless {
		go window.runTimer(id, timer, time.Duration(milliseconds)*time.Millisecond)
	}
	return id, ""
}

func AhdGUIEveryChecked(handle int64, milliseconds int64, handler func()) int64 {
	id, problem := AhdGUIEvery(handle, milliseconds, handler)
	AhdGUICheck(problem)
	return id
}

// runTimer posts one tick per interval until the timer is cancelled. A tick
// that is still waiting is not posted again.
func (w *ahdGUIWindow) runTimer(id int64, timer *ahdGUITimer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-timer.stop:
			return
		case <-w.link.exited:
			return
		case <-ticker.C:
			if w.queueTick(id) {
				line := []byte(fmt.Sprintf(`{"event":"tick","timer":%d}`, id))
				w.link.events.push(line, false)
			}
		}
	}
}

// queueTick keeps at most one unhandled tick for a timer. The caller does
// not hold w.mu.
func (w *ahdGUIWindow) queueTick(id int64) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	timer := w.timers[id]
	if timer == nil || timer.pending || !w.alive {
		return false
	}
	timer.pending = true
	return true
}

// dueTimers returns the handlers for one tick. id 0 is every timer, in the
// order they were started. A timer cancelled since the tick was queued is
// skipped. The caller holds w.mu.
func (w *ahdGUIWindow) dueTimers(id int64) []func() {
	var due []func()
	for _, candidate := range w.timerOrder {
		timer := w.timers[candidate]
		if timer == nil || (id != 0 && id != candidate) {
			continue
		}
		timer.pending = false
		due = append(due, timer.handler)
	}
	return due
}

// stopTimers ends every timer. The caller holds w.mu.
func (w *ahdGUIWindow) stopTimers() {
	for _, timer := range w.timers {
		close(timer.stop)
	}
	w.timers, w.timerOrder = nil, nil
}

// AhdGUICancel stops one timer. Cancelling a timer that already ended, or a
// number that never named one, does nothing.
func AhdGUICancel(handle int64, id int64) string {
	window, problem := ahdGUIWindowOf(handle)
	if problem != "" {
		return ""
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	timer := window.timers[id]
	if timer == nil {
		return ""
	}
	close(timer.stop)
	delete(window.timers, id)
	for index, candidate := range window.timerOrder {
		if candidate == id {
			window.timerOrder = append(window.timerOrder[:index:index], window.timerOrder[index+1:]...)
			break
		}
	}
	return ""
}

func AhdGUICancelChecked(handle int64, id int64) {
	AhdGUICheck(AhdGUICancel(handle, id))
}

func ahdGoroutineID() int64 {
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

func ahdGUIInTask() bool {
	return ahdGUITaskOwner.Load() == ahdGoroutineID()
}

// AhdGUIInTask reports whether the caller is inside a background task.
func AhdGUIInTask() bool { return ahdGUIInTask() }

// ahdGUITask is one background call. done is closed when the goroutine
// leaves the function, whether or not its result is delivered.
type ahdGUITask struct {
	id        int64
	window    int64
	work      func() (int64, string)
	cancelled atomic.Bool
	finished  atomic.Bool
	closed    atomic.Bool
	done      chan struct{}
	// events is closed when the task has published its result or reached
	// its first checkpoint, so wait can read window events without losing
	// a result that was ready first.
	events chan struct{}
	once   sync.Once

	mu        sync.Mutex
	onDone    func(*int64, *string)
	ready     bool
	delivered bool
	value     *int64
	problem   string
}

// AhdGUITask starts work on a goroutine when the Window is already waiting,
// or queues it until Window.wait. The function is not interrupted by cancel
// or by the window closing. Its result is delivered only if it returns
// before either of those, and only through onDone on the GUI callback path.
// A Window runs one task at a time.
func AhdGUITask(handle int64, work func() (int64, string)) (int64, string) {
	if work == nil {
		return 0, "Window.task needs a Function"
	}
	window, problem := ahdGUIWindowOf(handle)
	if problem != "" {
		return 0, problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return 0, ahdGUIClosedMessage
	}
	for _, task := range window.tasks {
		if !task.finished.Load() {
			return 0, "a Window runs one background task at a time"
		}
	}
	task := &ahdGUITask{id: ahdGUINextTask.Add(1), window: handle, work: work, done: make(chan struct{}), events: make(chan struct{})}
	window.tasks = append(window.tasks, task)
	if window.waiting {
		go ahdGUIRunTask(window, task)
	} else {
		window.pending = append(window.pending, task)
	}
	return task.id, ""
}

func AhdGUITaskChecked(handle int64, work func() (int64, string)) int64 {
	id, problem := AhdGUITask(handle, work)
	AhdGUICheck(problem)
	return id
}

// startTasks releases the tasks queued before wait. The caller does not
// hold w.mu.
func (w *ahdGUIWindow) startTasks() {
	w.mu.Lock()
	w.waiting = true
	pending := w.pending
	w.pending = nil
	w.mu.Unlock()
	for _, task := range pending {
		// The task publishes its result, or reaches its first checkpoint,
		// before wait reads events. A headless script cannot close the
		// window between those two and drop a result that was already ready.
		go ahdGUIRunTask(w, task)
		<-task.events
	}
}

func (task *ahdGUITask) releaseEvents() {
	task.once.Do(func() { close(task.events) })
}

func ahdGUIRunTask(window *ahdGUIWindow, task *ahdGUITask) {
	defer task.finishWait()
	if task.cancelled.Load() {
		task.finished.Store(true)
		task.releaseEvents()
		return
	}
	ahdGUITaskLock.Lock()
	ahdGUICallbackLock.Lock()
	gid := ahdGoroutineID()
	ahdGUITaskOwner.Store(gid)
	ahdGUICurrent.Store(task)
	locked := true
	unlock := func() {
		if !locked {
			return
		}
		locked = false
		if ahdGUICurrent.Load() == task {
			ahdGUICurrent.Store(nil)
		}
		if ahdGUITaskOwner.Load() == gid {
			ahdGUITaskOwner.Store(0)
		}
		ahdGUICallbackLock.Unlock()
		ahdGUITaskLock.Unlock()
	}
	defer unlock()
	var result int64
	var problem string
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				signal := AhdSignalOf(recovered)
				problem = signal.Message
			}
		}()
		result, problem = task.work()
	}()
	task.finished.Store(true)
	// Publish before letting wait read events, and before giving the
	// callback lock back to the event loop.
	task.finish(result, problem)
	task.releaseEvents()
	unlock()
}

func (task *ahdGUITask) finishWait() {
	if task.closed.CompareAndSwap(false, true) {
		close(task.done)
	}
}

// finish queues the completion when the task was not cancelled and the
// Window can still deliver it.
func (task *ahdGUITask) finish(result int64, problem string) {
	if task.cancelled.Load() {
		return
	}
	window, failure := ahdGUIWindowOf(task.window)
	if failure != "" {
		return
	}
	window.mu.Lock()
	if task.cancelled.Load() || !window.alive {
		window.mu.Unlock()
		return
	}
	task.mu.Lock()
	value := result
	task.value = &value
	task.problem = problem
	task.ready = true
	handler := task.onDone
	if handler != nil {
		task.delivered = true
	}
	task.mu.Unlock()
	window.mu.Unlock()
	if handler == nil {
		return
	}
	task.push()
}

func (task *ahdGUITask) push() {
	window, problem := ahdGUIWindowOf(task.window)
	if problem != "" {
		return
	}
	task.mu.Lock()
	value, text := task.value, task.problem
	task.mu.Unlock()
	payload := struct {
		Event string `json:"event"`
		Task  int64  `json:"task"`
		Value *int64 `json:"value,omitempty"`
		Text  string `json:"text,omitempty"`
	}{Event: "task", Task: task.id, Text: text}
	if text == "" {
		payload.Value = value
	}
	line, err := json.Marshal(payload)
	if err != nil {
		return
	}
	window.link.events.push(line, false)
}

// AhdGUITaskOnDone registers the completion callback. If the task has
// already finished successfully, the callback is queued at once.
func AhdGUITaskOnDone(handle int64, handler func(*int64, *string)) string {
	if handler == nil {
		return "Task.onDone needs a Function"
	}
	window, task, problem := ahdGUITaskOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	task.mu.Lock()
	task.onDone = handler
	ready := task.ready && !task.delivered && !task.cancelled.Load() && window.alive
	if ready {
		task.delivered = true
	}
	task.mu.Unlock()
	window.mu.Unlock()
	if ready {
		task.push()
	}
	return ""
}

func AhdGUITaskOnDoneChecked(handle int64, handler func(*int64, *string)) {
	AhdGUICheck(AhdGUITaskOnDone(handle, handler))
}

// AhdGUITaskCancel asks that the result be discarded. It does not stop the
// function. Cancelling a task that already finished, or a number that never
// named one, does nothing.
func AhdGUITaskCancel(handle int64) string {
	_, task, problem := ahdGUITaskOf(handle)
	if problem != "" {
		return ""
	}
	task.cancelled.Store(true)
	return ""
}

func AhdGUITaskCancelChecked(handle int64) { AhdGUICheck(AhdGUITaskCancel(handle)) }

// AhdGUITaskCancelled reports whether the task was cancelled or its window
// began closing before the function returned.
func AhdGUITaskCancelled(handle int64) bool {
	_, task, problem := ahdGUITaskOf(handle)
	if problem != "" {
		return true
	}
	return task.cancelled.Load()
}

func AhdGUITaskCancelledChecked(handle int64) bool { return AhdGUITaskCancelled(handle) }

func ahdGUITaskOf(id int64) (*ahdGUIWindow, *ahdGUITask, string) {
	ahdGUI.Lock()
	defer ahdGUI.Unlock()
	for _, window := range ahdGUI.windows {
		for _, task := range window.tasks {
			if task.id == id {
				return window, task, ""
			}
		}
	}
	return nil, nil, "the background task does not exist"
}

// taskHandler returns the completion callback. The caller holds w.mu.
func (w *ahdGUIWindow) taskHandler(id int64) func(*int64, *string) {
	for _, task := range w.tasks {
		if task.id == id {
			task.mu.Lock()
			handler := task.onDone
			task.mu.Unlock()
			return handler
		}
	}
	return nil
}

// taskCancelled reports a task whose result must not be delivered. The
// caller holds w.mu.
func (w *ahdGUIWindow) taskCancelled(id int64) bool {
	for _, task := range w.tasks {
		if task.id == id {
			return task.cancelled.Load()
		}
	}
	return true
}

// cancelUnfinishedTasks drops results of tasks that have not returned and
// of tasks that have not started. The caller holds w.mu.
func (w *ahdGUIWindow) cancelUnfinishedTasks() {
	for _, task := range w.tasks {
		if !task.finished.Load() {
			task.cancelled.Store(true)
		}
	}
	for _, task := range w.pending {
		task.cancelled.Store(true)
		task.finished.Store(true)
		task.finishWait()
	}
	w.pending = nil
}

func ahdGUIJoinTasks(handle int64) {
	window, problem := ahdGUIWindowOf(handle)
	if problem != "" {
		return
	}
	window.mu.Lock()
	tasks := append([]*ahdGUITask(nil), window.tasks...)
	window.mu.Unlock()
	for _, task := range tasks {
		<-task.done
	}
}

// AhdGUICheckpoint lets the Window run callbacks that are waiting. Outside a
// task it does nothing and returns true. Inside a task it returns false when
// the task was cancelled or the Window began to close. It does not stop the
// function; the function decides whether to return.
func AhdGUICheckpoint() bool {
	if !ahdGUIInTask() {
		return true
	}
	task := ahdGUICurrent.Load()
	if task != nil && task.cancelled.Load() {
		return false
	}
	if task != nil {
		task.releaseEvents()
	}
	ahdGUICallbackLock.Unlock()
	runtime.Gosched()
	time.Sleep(time.Millisecond)
	ahdGUICallbackLock.Lock()
	if task != nil && task.cancelled.Load() {
		return false
	}
	if task == nil {
		return true
	}
	window, problem := ahdGUIWindowOf(task.window)
	if problem != "" {
		return false
	}
	window.mu.Lock()
	open := window.usable && window.alive
	window.mu.Unlock()
	return open && !task.cancelled.Load()
}

func AhdGUICheckpointChecked() bool { return AhdGUICheckpoint() }

// AhdGUIAddCanvas adds a drawing surface. Width and height are its natural
// size, from 1 to 4096.
func AhdGUIAddCanvas(parent int64, width, height int64) (int64, string) {
	if width < 1 || height < 1 || width > ahdGUIMaxCanvasSide || height > ahdGUIMaxCanvasSide {
		return 0, fmt.Sprintf("a Canvas must be between 1 and %d pixels on each side", ahdGUIMaxCanvasSide)
	}
	widget, _, problem := ahdGUIWidgetOf(parent)
	if problem != "" {
		return 0, problem
	}
	return ahdGUIAdd(widget.window, widget.id, parent, "canvas", ahdGUIRequest{Width: int(width), Height: int(height)})
}

func AhdGUIAddCanvasChecked(parent, width, height int64) int64 {
	id, problem := AhdGUIAddCanvas(parent, width, height)
	AhdGUICheck(problem)
	return id
}

func ahdGUICanvasColor(role, text string) (string, string) {
	parsed, ok := AhdGraphicsParseColor(text)
	if !ok {
		return "", ahdGraphicsColorProblem(role, text)
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", parsed[0], parsed[1], parsed[2], parsed[3]), ""
}

func ahdGUIFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && math.Abs(value) <= ahdGUIMaxCanvasCoord
}

// AhdGUICanvasDraw sends one drawing command. An empty fill draws no fill.
// An empty color on clear uses white.
func AhdGUICanvasDraw(handle int64, kind string, x, y, x2, y2, radius, stroke float64, color, fill, text string) string {
	for _, value := range []float64{x, y, x2, y2, radius} {
		if !ahdGUIFinite(value) {
			return "a Canvas coordinate is out of range"
		}
	}
	if !ahdGUIFinite(stroke) {
		return "a line width must be between 1 and 64"
	}
	request := ahdGUIRequest{Op: "draw", Kind: kind, X: x, Y: y, X2: x2, Y2: y2, Radius: radius, StrokeW: stroke, Text: text}
	switch kind {
	case "clear":
		if color == "" {
			color = "white"
		}
		normalized, problem := ahdGUICanvasColor("fill", color)
		if problem != "" {
			return problem
		}
		request.Color = normalized
	case "line":
		if stroke == 0 {
			stroke = 1
		}
		if stroke < 1 || stroke > 64 {
			return "a line width must be between 1 and 64"
		}
		request.StrokeW = stroke
		if color == "" {
			color = "black"
		}
		normalized, problem := ahdGUICanvasColor("stroke", color)
		if problem != "" {
			return problem
		}
		request.Color = normalized
	case "rectangle", "circle":
		if kind == "rectangle" && (x2 <= 0 || y2 <= 0) {
			return "a rectangle's width and height must be greater than 0"
		}
		if kind == "circle" && (radius <= 0 || radius > ahdGUIMaxCanvasCoord) {
			return "a circle radius must be greater than 0 and at most 100000"
		}
		if stroke == 0 {
			stroke = 1
		}
		if stroke < 1 || stroke > 64 {
			return "a line width must be between 1 and 64"
		}
		request.StrokeW = stroke
		if color == "" {
			color = "black"
		}
		normalized, problem := ahdGUICanvasColor("stroke", color)
		if problem != "" {
			return problem
		}
		request.Color = normalized
		if fill != "" {
			normalized, problem = ahdGUICanvasColor("fill", fill)
			if problem != "" {
				return problem
			}
			request.Fill = normalized
		}
	case "text":
		if text == "" || len([]rune(text)) > 1000 {
			return "Canvas text must be between 1 and 1000 characters"
		}
		if stroke == 0 {
			stroke = 16
		}
		if stroke < 1 || stroke > 200 {
			return "Canvas text size must be between 1 and 200"
		}
		request.StrokeW = stroke
		if color == "" {
			color = "black"
		}
		normalized, problem := ahdGUICanvasColor("fill", color)
		if problem != "" {
			return problem
		}
		request.Color = normalized
	default:
		return "unknown Canvas drawing command"
	}
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	request.Widget = widget.id
	_, problem = window.request(request, ahdGUIRequestTimeout)
	return problem
}

func AhdGUICanvasDrawChecked(handle int64, kind string, x, y, x2, y2, radius, stroke float64, color, fill, text string) {
	AhdGUICheck(AhdGUICanvasDraw(handle, kind, x, y, x2, y2, radius, stroke, color, fill, text))
}

func ahdGUICanvasListen(handle int64, kind, member string, handler func(float64, float64), store func(*ahdGUIWindow, int64)) string {
	if handler == nil {
		return member + " needs a Function"
	}
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	if widget.kind != "canvas" {
		return "the widget is not a Canvas"
	}
	if _, problem := window.request(ahdGUIRequest{Op: "listen", Kind: kind, Widget: widget.id}, ahdGUIRequestTimeout); problem != "" {
		return problem
	}
	store(window, widget.id)
	return ""
}

// AhdGUICanvasOnMove registers the pointer-move callback.
func AhdGUICanvasOnMove(handle int64, handler func(float64, float64)) string {
	return ahdGUICanvasListen(handle, "move", "Canvas.onMove", handler, func(window *ahdGUIWindow, id int64) {
		window.canvasMove[id] = handler
	})
}

func AhdGUICanvasOnMoveChecked(handle int64, handler func(float64, float64)) {
	AhdGUICheck(AhdGUICanvasOnMove(handle, handler))
}

// AhdGUICanvasOnDrag registers the drag callback.
func AhdGUICanvasOnDrag(handle int64, handler func(float64, float64)) string {
	return ahdGUICanvasListen(handle, "drag", "Canvas.onDrag", handler, func(window *ahdGUIWindow, id int64) {
		window.canvasDrag[id] = handler
	})
}

func AhdGUICanvasOnDragChecked(handle int64, handler func(float64, float64)) {
	AhdGUICheck(AhdGUICanvasOnDrag(handle, handler))
}

// AhdGUICanvasOnClick registers the click callback. The press is not a drag.
func AhdGUICanvasOnClick(handle int64, handler func(float64, float64)) string {
	return ahdGUICanvasListen(handle, "click", "Canvas.onClick", handler, func(window *ahdGUIWindow, id int64) {
		window.canvasClick[id] = handler
	})
}

func AhdGUICanvasOnClickChecked(handle int64, handler func(float64, float64)) {
	AhdGUICheck(AhdGUICanvasOnClick(handle, handler))
}
