package build

import "testing"

// Widget ids follow creation order: column 1, tabs 2, pages 3 and 5,
// labels 4 and 6, radio group 7, options 8-10, progress 11, date 12.
func TestGUIControlsNativeAndEvaluatorAgree(t *testing.T) {
	useHeadlessGUI(t, `[{"event":"select","widget":2,"selected":1},{"event":"select","widget":7,"selected":0},{"event":"type","widget":12,"text":"2020"}]`)
	program := `bring GUI
from GUI bring (Window, Container, Tabs, RadioGroup, RadioButton, Progress, DateField, GUIError)

window: Window := GUI.window("Controls", 480, 420)
form: Container := window.column()
tabs: Tabs := form.tabs()
one: Container := tabs.page("One")
one.label("first")
two: Container := tabs.page("Two")
two.label("second")
group: RadioGroup := form.radioGroup()
group.option("Alpha")
group.option("Beta")
quiet: RadioButton := group.option("Quiet")
quiet.setEnabled(false)
bar: Progress := form.progress(maximum: 10)
when: DateField := form.date()

onTab: Function := (index: Int) -> Nothing {
    if index == 1 {
        write("tab")
    }
}
onRadio: Function := (index: Int?, text: String?) -> Nothing {
    if index != null and text != null {
        write("radio")
    }
}
onDate: Function := (year: Int, month: Int, day: Int) -> Nothing {
    if year == 2020 and month == 2 and day == 29 {
        write("date")
    }
}
tabs.onChange(onTab)
group.onChange(onRadio)
when.onChange(onDate)

tabs.select(1)
if tabs.selectedIndex() != 1 {
    write("tab select failed")
}
tabs.select(0)
group.select(1)
if group.selectedText() != "Beta" {
    write("radio text failed")
}
at: Int? := group.selectedIndex()
if at != 1 {
    write("radio select failed")
}
group.select(2)
disabledAt: Int? := group.selectedIndex()
if disabledAt != 2 {
    write("disabled program select failed")
}
group.select(null)
if group.selectedIndex() != null {
    write("clear failed")
}
bar.setValue(4)
if bar.value() != 4 or bar.maximum() != 10 {
    write("progress failed")
}
bar.setIndeterminate(true)
bar.setIndeterminate(false)
if bar.isIndeterminate() {
    write("still spinning")
}
when.setDate(2024, 2, 29)
if when.year() != 2024 or when.month() != 2 or when.day() != 29 {
    write("leap failed")
}
attempt {
    when.setDate(2023, 2, 29)
    write("accepted bad date")
} except GUIError as error {
    write(error.message)
}
if when.day() != 29 {
    write("date changed")
}
attempt {
    bar.setValue(11)
    write("accepted value")
} except GUIError as error {
    write(error.message)
}
attempt {
    bar.setValue(-1)
    write("accepted negative")
} except GUIError as error {
    write(error.message)
}
attempt {
    form.progress(maximum: 0)
    write("accepted maximum")
} except GUIError as error {
    write(error.message)
}
attempt {
    form.date(1900, 2, 29)
    write("accepted 1900")
} except GUIError as error {
    write(error.message)
}
attempt {
    tabs.page("")
    write("accepted title")
} except GUIError as error {
    write(error.message)
}
attempt {
    tabs.select(9)
    write("accepted tab")
} except GUIError as error {
    write(error.message)
}
attempt {
    group.select(9)
    write("accepted option")
} except GUIError as error {
    write(error.message)
}
window.wait()
if tabs.selectedIndex() != 1 {
    write("script tab failed")
}
if group.selectedText() != "Alpha" {
    write("script radio failed")
}
write(when.year())
write(when.month())
write(when.day())
write(bar.value())
if bar.isIndeterminate() {
    write("spinning")
} else {
    write("still")
}
write("ready")
`
	assertGUI(t, program, "the date is not a real calendar day\nprogress value must be between 0 and the maximum\nprogress value must be between 0 and the maximum\nprogress maximum must be between 1 and 1000000\nthe date is not a real calendar day\na tab needs a title\nthe tab index is outside the pages\nthe option index is outside the group\ntab\nradio\ndate\n2020\n2\n29\n4\nstill\nready\n")
}
