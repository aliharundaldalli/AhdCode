package build

import "testing"

// Widget ids follow creation order: column 1, canvas 2, progress 3.
func TestGUITimersTasksAndCanvasAgree(t *testing.T) {
	useHeadlessGUI(t, `[{"event":"tick"},{"event":"tick"},{"event":"click","widget":2,"x":50,"y":40}]`)
	program := `bring GUI
from GUI bring (Window, Container, Progress, Canvas, Task, GUIError)

window: Window := GUI.window("Interactive", 400, 300)
form: Container := window.column(0, 0)
board: Canvas := form.canvas(100, 80)
bar: Progress := form.progress(maximum: 10)
ticks: Int := 0
result: Int := 0
origin: Bool := false

onTick: Function := () -> Nothing {
    ticks: Global Int
    bar: Global Progress
    ticks = ticks + 1
    bar.setValue(ticks)
}
onClick: Function := (x: Real, y: Real) -> Nothing {
    origin: Global Bool
    if x == 0.0 and y == 0.0 {
        origin = true
    }
}
onDone: Function := (value: Int?, problem: String?) -> Nothing {
    result: Global Int
    if value != null and problem == null {
        result = value
    }
}
window.every(1000, onTick)
board.onClick(onClick)
work: Function := () -> Int {
    total: Local Int := 0
    n: Local Int := 1
    while n <= 10 {
        total = total + n
        n = n + 1
    }
    return total
}
job: Task := window.task(work)
job.onDone(onDone)
window.wait()
write(ticks)
write(bar.value())
write(result)
if origin {
    write("origin")
} else {
    write("off")
}
`
	assertGUI(t, program, "2\n2\n55\norigin\n")
}
