package main

import "testing"

func TestCalendarAndProgressBounds(t *testing.T) {
	valid := [][3]int{{2000, 2, 29}, {2024, 2, 29}, {2023, 2, 28}, {1, 1, 1}, {9999, 12, 31}, {2023, 4, 30}}
	for _, date := range valid {
		if !realDate(date[0], date[1], date[2]) {
			t.Fatalf("rejected real date %v", date)
		}
	}
	invalid := [][3]int{{1900, 2, 29}, {2023, 2, 29}, {2023, 4, 31}, {2023, 0, 1}, {0, 1, 1}, {10000, 1, 1}, {2023, 13, 1}, {2023, 1, 0}}
	for _, date := range invalid {
		if realDate(date[0], date[1], date[2]) {
			t.Fatalf("accepted %v", date)
		}
	}
	m := newModel("t", 400, 300, fixedMeasure)
	root := mustAdd(t, m, 0, kindColumn, "")
	for _, date := range invalid {
		if _, err := m.addSpec(root, spec{kind: kindDate, year: date[0], month: date[1], day: date[2], selected: -1}); err != errBadDate {
			t.Fatalf("date %v: %v", date, err)
		}
	}
	when := mustSpec(t, m, root, spec{kind: kindDate, year: 2024, month: 2, day: 29, selected: -1})
	if err := m.setDate(when, 2023, 2, 29); err != errBadDate {
		t.Fatalf("setDate: %v", err)
	}
	text, _, _, err := m.readAll(when)
	if err != nil || text != "2024-02-29" {
		t.Fatalf("date changed to %q (%v)", text, err)
	}
	if err := m.setDate(when, 2023, 1, 31); err != nil {
		t.Fatal(err)
	}
	if _, err := m.addSpec(root, spec{kind: kindProgress, maximum: 0, selected: -1}); err == nil {
		t.Fatal("accepted a zero maximum")
	}
	if _, err := m.addSpec(root, spec{kind: kindProgress, maximum: maxProgress + 1, selected: -1}); err == nil {
		t.Fatal("accepted a maximum past the limit")
	}
	bar := mustSpec(t, m, root, spec{kind: kindProgress, maximum: 10, selected: -1})
	if err := m.setAmount(bar, 4); err != nil {
		t.Fatal(err)
	}
	if err := m.setAmount(bar, 11); err == nil || err.Error() != "progress value must be between 0 and the maximum" {
		t.Fatalf("high value: %v", err)
	}
	if err := m.setAmount(bar, -1); err == nil {
		t.Fatal("accepted a negative value")
	}
	_, spin, value, err := m.readAll(bar)
	if err != nil || spin || value != 4 {
		t.Fatalf("progress reading spin %v value %d (%v)", spin, value, err)
	}
	if err := m.setSpin(bar, true); err != nil {
		t.Fatal(err)
	}
	if _, spin, _, err := m.readAll(bar); err != nil || !spin {
		t.Fatal("indeterminate did not stick")
	}
	if err := m.setEnabled(bar, false); err != nil {
		t.Fatal(err)
	}
	if err := m.setAmount(bar, 5); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if m.widgets[bar].w == 0 || m.widgets[bar].h == 0 {
		t.Fatal("a disabled progress bar left the layout")
	}
	m.mu.Lock()
	m.phase = 3
	img := render(frame{model: m, scale: 1})
	m.mu.Unlock()
	if img.Bounds().Empty() {
		t.Fatal("progress did not draw")
	}
}

func TestTabsKeyboardMouseAndFocus(t *testing.T) {
	m := newModel("t", 800, 640, fixedMeasure)
	root := mustAdd(t, m, 0, kindColumn, "")
	tabs := mustAdd(t, m, root, kindTabs, "")
	if _, err := m.add(tabs, kindButton, "No", "", false, 0, 0); err == nil {
		t.Fatal("a Tabs control accepted a button")
	}
	if _, err := m.addSpec(root, spec{kind: kindPage, text: "Loose", selected: -1}); err == nil {
		t.Fatal("a page was created outside Tabs")
	}
	one := mustAdd(t, m, tabs, kindPage, "One")
	first := mustAdd(t, m, one, kindButton, "Go")
	two := mustAdd(t, m, tabs, kindPage, "Two")
	second := mustAdd(t, m, two, kindButton, "Stay")
	if _, err := m.addSpec(tabs, spec{kind: kindPage, text: "", selected: -1}); err == nil {
		t.Fatal("accepted an empty title")
	}
	layout(t, m)
	if m.widgets[tabs].selected != 0 {
		t.Fatalf("first page was not selected: %d", m.widgets[tabs].selected)
	}
	// "One" and "Two" are 3 characters at 10 pixels plus the button padding,
	// so the second header starts at 58.
	m.press(10, 40)
	if m.focus != first {
		t.Fatalf("first page focus %d, want %d", m.focus, first)
	}
	m.press(70, 8)
	m.release(70, 8)
	if m.widgets[tabs].selected != 1 {
		t.Fatalf("click selected %d", m.widgets[tabs].selected)
	}
	if changes := m.takeChanges(); len(changes) != 1 || changes[0] != tabs {
		t.Fatalf("tab change %v", changes)
	}
	m.press(10, 40)
	if m.focus == first {
		t.Fatal("the inactive page received the click")
	}
	if m.focus != second {
		t.Fatalf("active page focus %d, want %d", m.focus, second)
	}
	m.press(10, 8)
	if m.focus != first {
		t.Fatalf("focus was not restored to %d (got %d)", first, m.focus)
	}
	m.focusNext()
	if m.focus == second {
		t.Fatal("Tab entered the inactive page")
	}
	m.mu.Lock()
	m.focus = tabs
	m.mu.Unlock()
	if !m.navigate("ArrowRight") || m.widgets[tabs].selected != 1 || m.focus != tabs {
		t.Fatalf("arrow selected %d focus %d", m.widgets[tabs].selected, m.focus)
	}
	m.takeChanges()
	if err := m.selectIndex(tabs, 0); err != nil {
		t.Fatal(err)
	}
	if len(m.takeChanges()) != 0 {
		t.Fatal("a programmatic tab change notified the user callback")
	}
	if m.focus == second {
		t.Fatal("a programmatic change left the focus on the hidden page")
	}
	if err := m.selectIndex(tabs, 4); err != errTabIndex {
		t.Fatalf("bad index: %v", err)
	}
	if err := m.setVisible(two, false); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if m.widgets[two].w != 0 {
		t.Fatal("a hidden page kept its box")
	}
	m.mu.Lock()
	m.focus = tabs
	m.mu.Unlock()
	before := m.widgets[tabs].selected
	m.navigate("ArrowRight")
	if m.widgets[tabs].selected != before {
		t.Fatal("an arrow selected the hidden page")
	}
	if err := m.selectIndex(tabs, 1); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if m.widgets[second].h != 0 {
		t.Fatal("a hidden active page still showed its button")
	}
}

func TestRadioGroup(t *testing.T) {
	m := newModel("t", 400, 300, fixedMeasure)
	root := mustAdd(t, m, 0, kindColumn, "")
	if _, err := m.add(root, kindRadio, "Loose", "", false, 0, 0); err == nil {
		t.Fatal("an option was created outside a group")
	}
	group := mustSpec(t, m, root, spec{kind: kindRadioGroup, spacing: 4, selected: -1})
	alpha := mustAdd(t, m, group, kindRadio, "Alpha")
	beta := mustAdd(t, m, group, kindRadio, "Beta")
	quiet := mustAdd(t, m, group, kindRadio, "Quiet")
	if err := m.setEnabled(quiet, false); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	quietBox := widgetBox(m, quiet)
	m.press(quietBox[0]+4, quietBox[1]+4)
	m.release(quietBox[0]+4, quietBox[1]+4)
	if m.widgets[group].selected != -1 || m.focus == quiet {
		t.Fatal("a disabled option was selected")
	}
	alphaBox := widgetBox(m, alpha)
	m.press(alphaBox[0]+4, alphaBox[1]+4)
	m.release(alphaBox[0]+4, alphaBox[1]+4)
	if m.widgets[group].selected != 0 || m.focus != alpha {
		t.Fatalf("click selected %d focus %d", m.widgets[group].selected, m.focus)
	}
	if !m.navigate("ArrowDown") || m.widgets[group].selected != 1 || m.focus != beta {
		t.Fatalf("arrow selected %d focus %d", m.widgets[group].selected, m.focus)
	}
	if !m.navigate("ArrowDown") || m.focus == quiet || m.widgets[group].selected == 2 {
		t.Fatal("an arrow selected the disabled option")
	}
	m.takeChanges()
	if err := m.selectIndex(group, 2); err != nil || m.widgets[group].selected != 2 {
		t.Fatalf("programmatic disabled select: %v index %d", err, m.widgets[group].selected)
	}
	if len(m.takeChanges()) != 0 {
		t.Fatal("a programmatic radio change notified the user callback")
	}
	if err := m.selectIndex(group, -1); err != nil || m.widgets[group].selected != -1 {
		t.Fatal("clearing the group failed")
	}
	if err := m.selectIndex(group, 3); err != errRadioIndex {
		t.Fatalf("bad option index: %v", err)
	}
	if err := m.choose(group, 2); err != nil || m.widgets[group].selected != -1 {
		t.Fatal("a scripted click selected the disabled option")
	}
	if err := m.choose(group, 1); err != nil || m.widgets[group].selected != 1 {
		t.Fatal("a scripted choice did not select")
	}
	if changes := m.takeChanges(); len(changes) != 1 || changes[0] != group {
		t.Fatalf("radio change %v", changes)
	}
	if err := m.setVisible(beta, false); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if m.widgets[beta].h != 0 || m.widgets[quiet].y <= m.widgets[alpha].y {
		t.Fatal("a hidden option kept its place")
	}
	if m.widgets[quiet].h == 0 {
		t.Fatal("a disabled option left the layout")
	}
}

func TestDateFieldInteraction(t *testing.T) {
	m := newModel("t", 400, 200, fixedMeasure)
	root := mustAdd(t, m, 0, kindColumn, "")
	when := mustSpec(t, m, root, spec{kind: kindDate, year: 2000, month: 1, day: 1, selected: -1})
	layout(t, m)
	box := widgetBox(m, when)
	m.press(box[0]+box[2]-8, box[1]+4)
	if m.focus != when || m.widgets[when].segment != 2 {
		t.Fatalf("day segment focus %d segment %d", m.focus, m.widgets[when].segment)
	}
	if !m.navigate("ArrowUp") || m.widgets[when].day != 2 {
		t.Fatalf("day %d", m.widgets[when].day)
	}
	if !m.navigate("ArrowLeft") || m.widgets[when].segment != 1 {
		t.Fatal("Left did not reach the month")
	}
	if err := m.setDate(when, 2023, 1, 31); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.widgets[when].segment = 1
	m.mu.Unlock()
	if !m.navigate("ArrowUp") || m.widgets[when].month != 2 || m.widgets[when].day != 28 {
		t.Fatalf("clamp gave %02d-%02d", m.widgets[when].month, m.widgets[when].day)
	}
	if err := m.setDate(when, 2024, 1, 31); err != nil {
		t.Fatal(err)
	}
	m.navigate("ArrowUp")
	if m.widgets[when].day != 29 || m.widgets[when].month != 2 {
		t.Fatalf("leap clamp gave %04d-%02d-%02d", m.widgets[when].year, m.widgets[when].month, m.widgets[when].day)
	}
	m.mu.Lock()
	m.widgets[when].segment = 0
	m.widgets[when].year = 1
	m.mu.Unlock()
	m.navigate("ArrowDown")
	if m.widgets[when].year != 1 {
		t.Fatal("the year moved below 1")
	}
	m.mu.Lock()
	m.widgets[when].year = 9999
	m.mu.Unlock()
	m.navigate("ArrowUp")
	if m.widgets[when].year != 9999 {
		t.Fatal("the year moved above 9999")
	}
	if err := m.typeInto(when, "2020"); err != nil {
		t.Fatal(err)
	}
	if m.widgets[when].year != 2020 {
		t.Fatalf("typed year %d", m.widgets[when].year)
	}
	if changes := m.takeChanges(); len(changes) != 1 || changes[0] != when {
		t.Fatalf("date change %v", changes)
	}
	m.mu.Lock()
	m.widgets[when].segment = 1
	m.mu.Unlock()
	if err := m.typeInto(when, "13"); err != nil {
		t.Fatal(err)
	}
	if m.widgets[when].month != 2 || m.widgets[when].digits != "" {
		t.Fatalf("invalid month stuck: month %d digits %q", m.widgets[when].month, m.widgets[when].digits)
	}
	if err := m.typeInto(when, "2"); err != nil {
		t.Fatal(err)
	}
	m.navigate("Backspace")
	m.navigate("ArrowRight")
	if m.widgets[when].year != 2020 || m.widgets[when].month != 2 {
		t.Fatal("a cleared buffer was committed")
	}
	if err := m.setEnabled(when, false); err != nil {
		t.Fatal(err)
	}
	m.press(box[0]+4, box[1]+4)
	if m.focus == when {
		t.Fatal("a disabled date took the focus")
	}
	if err := m.typeInto(when, "1999"); err != nil || m.widgets[when].year != 2020 {
		t.Fatal("a disabled date accepted digits")
	}
	if err := m.setVisible(when, false); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if m.widgets[when].h != 0 {
		t.Fatal("a hidden date kept its place")
	}
}

func TestControlsInGridAndScroll(t *testing.T) {
	m := newModel("t", 420, 240, fixedMeasure)
	grid := mustSpec(t, m, 0, spec{kind: kindGrid, gridColumns: 2, spacing: 8, rowGap: 4, selected: -1})
	bar := mustSpec(t, m, grid, spec{kind: kindProgress, maximum: 100, selected: -1})
	when := mustSpec(t, m, grid, spec{kind: kindDate, year: 2000, month: 1, day: 1, selected: -1})
	if err := m.setAmount(bar, 100); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if m.widgets[bar].w == 0 || m.widgets[when].w == 0 || m.widgets[bar].x == m.widgets[when].x {
		t.Fatalf("grid placed progress %v date %v", widgetBox(m, bar), widgetBox(m, when))
	}
	m.focusNext()
	if m.focus == bar {
		t.Fatal("a progress bar took the focus")
	}
	if m.focus != when {
		t.Fatalf("Tab skipped the date: %d", m.focus)
	}
	img := render(frame{model: m, scale: 1})
	barBox := widgetBox(m, bar)
	if img.RGBAAt(barBox[0]+8, barBox[1]+barBox[3]/2) != colorFocus {
		t.Fatalf("full progress pixel %#v", img.RGBAAt(barBox[0]+8, barBox[1]+barBox[3]/2))
	}

	scrolled := newModel("t", 200, 80, fixedMeasure)
	body := mustSpec(t, scrolled, 0, spec{kind: kindScroll, spacing: 0, padding: 0, selected: -1})
	for i := 0; i < 6; i++ {
		mustAdd(t, scrolled, body, kindLabel, "Line")
	}
	date := mustSpec(t, scrolled, body, spec{kind: kindDate, year: 2000, month: 1, day: 1, selected: -1})
	layout(t, scrolled)
	below := widgetBox(scrolled, date)
	if scrolled.hit(below[0]+4, below[1]+4) == date {
		t.Fatal("a date below the viewport was hit")
	}
	scrolled.mu.Lock()
	scrolled.widgets[body].scrollY = below[1]
	scrolled.mu.Unlock()
	layout(t, scrolled)
	moved := widgetBox(scrolled, date)
	scrolled.press(moved[0]+4, moved[1]+4)
	if scrolled.focus != date {
		t.Fatalf("scrolled date focus %d box %v", scrolled.focus, moved)
	}
}

func TestControlLimits(t *testing.T) {
	m := newModel("t", 200, 200, fixedMeasure)
	root := mustAdd(t, m, 0, kindColumn, "")
	tabs := mustAdd(t, m, root, kindTabs, "")
	for i := 0; i < maxTabPages; i++ {
		mustAdd(t, m, tabs, kindPage, "P")
	}
	if _, err := m.add(tabs, kindPage, "P", "", false, 0, 0); err == nil {
		t.Fatal("accepted a 33rd page")
	}
	group := mustAdd(t, m, root, kindRadioGroup, "")
	for i := 0; i < maxRadioItems; i++ {
		mustAdd(t, m, group, kindRadio, "O")
	}
	if _, err := m.add(group, kindRadio, "O", "", false, 0, 0); err == nil {
		t.Fatal("accepted a 65th option")
	}
}
