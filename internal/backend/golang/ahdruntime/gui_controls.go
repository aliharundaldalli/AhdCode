package ahdruntime

import "sort"

func ahdGUIKids(parentHandle int64, kind string) []*ahdGUIWidget {
	ahdGUI.Lock()
	defer ahdGUI.Unlock()
	var kids []*ahdGUIWidget
	for _, widget := range ahdGUI.widgets {
		if widget.parent == parentHandle && widget.kind == kind {
			kids = append(kids, widget)
		}
	}
	sort.Slice(kids, func(i, j int) bool { return kids[i].id < kids[j].id })
	return kids
}

func ahdGUILeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func ahdGUIDays(year, month int) int {
	days := []int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if month < 1 || month > 12 {
		return 0
	}
	if month == 2 && ahdGUILeap(year) {
		return 29
	}
	return days[month]
}

func ahdGUIRealDate(year, month, day int) bool {
	if year < 1 || year > 9999 {
		return false
	}
	return day >= 1 && day <= ahdGUIDays(year, month)
}

func ahdGUIParseDate(text string) (int, int, int, bool) {
	if len(text) != 10 || text[4] != '-' || text[7] != '-' {
		return 0, 0, 0, false
	}
	year, month, day := 0, 0, 0
	for index, part := range []struct {
		at, n int
		out   *int
	}{{0, 4, &year}, {5, 2, &month}, {8, 2, &day}} {
		_ = index
		for _, r := range text[part.at : part.at+part.n] {
			if r < '0' || r > '9' {
				return 0, 0, 0, false
			}
			*part.out = *part.out*10 + int(r-'0')
		}
	}
	if !ahdGUIRealDate(year, month, day) {
		return 0, 0, 0, false
	}
	return year, month, day, true
}

func AhdGUIAddTabs(container int64) (int64, string) {
	parent, problem := ahdGUIContainer(container)
	if problem != "" {
		return 0, problem
	}
	return ahdGUIAdd(parent.window, parent.id, container, "tabs", ahdGUIRequest{})
}

func AhdGUIAddPage(tabs int64, title string) (int64, string) {
	if len([]rune(title)) == 0 {
		return 0, "a tab needs a title"
	}
	if len([]rune(title)) > 256 {
		return 0, "a tab title must be at most 256 characters"
	}
	widget, _, problem := ahdGUIWidgetOf(tabs)
	if problem != "" {
		return 0, problem
	}
	if widget.kind != "tabs" {
		return 0, "a Tabs control only holds pages"
	}
	if len(ahdGUIKids(tabs, "page")) >= 32 {
		return 0, "a Tabs control holds at most 32 pages"
	}
	return ahdGUIAdd(widget.window, widget.id, tabs, "page", ahdGUIRequest{Text: title, Spacing: 8})
}

func AhdGUISelectTab(handle int64, index int64) string {
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	if widget.kind != "tabs" {
		return "the widget does not support that request"
	}
	count := int64(len(ahdGUIKids(handle, "page")))
	if count == 0 {
		return "a Tabs control has no pages"
	}
	if index < 0 || index >= count {
		return "the tab index is outside the pages"
	}
	selected := int(index)
	return ahdGUIChange(handle, ahdGUIRequest{Kind: "selected", Selected: &selected}, func(*ahdGUIWidget) {})
}

func AhdGUITabIndex(handle int64) (int64, string) {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return 0, problem
	}
	if widget.kind != "tabs" {
		return 0, "the widget does not support that request"
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	index := -1
	if final, ok := window.finals[widget.id]; ok || !window.alive {
		if final.Selected != nil {
			index = *final.Selected
		}
	} else {
		reply, problem := window.request(ahdGUIRequest{Op: "get", Widget: widget.id}, ahdGUIRequestTimeout)
		if problem != "" {
			return 0, problem
		}
		if reply.Selected == nil {
			return 0, "the GUI window helper sent an invalid response"
		}
		index = *reply.Selected
	}
	if index < 0 {
		return 0, "a Tabs control has no pages"
	}
	return int64(index), ""
}

func AhdGUIOnTabChange(handle int64, handler func(index int64)) string {
	if handler == nil {
		return "onChange needs a Function"
	}
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	if widget.kind != "tabs" {
		return "the widget does not support that request"
	}
	return ahdGUIListen(handle, "tab", func(event ahdGUIEvent) {
		if event.Selected == nil || *event.Selected < 0 {
			return
		}
		handler(int64(*event.Selected))
	})
}

func AhdGUIAddRadioGroup(container int64) (int64, string) {
	parent, problem := ahdGUIContainer(container)
	if problem != "" {
		return 0, problem
	}
	return ahdGUIAdd(parent.window, parent.id, container, "radioGroup", ahdGUIRequest{Spacing: 4})
}

func AhdGUIAddOption(group int64, text string) (int64, string) {
	if len([]rune(text)) == 0 || len([]rune(text)) > 256 {
		return 0, "a radio option must be between 1 and 256 characters"
	}
	widget, _, problem := ahdGUIWidgetOf(group)
	if problem != "" {
		return 0, problem
	}
	if widget.kind != "radioGroup" {
		return 0, "only a radio group holds options"
	}
	if len(ahdGUIKids(group, "radio")) >= 64 {
		return 0, "a radio group holds at most 64 options"
	}
	handle, problem := ahdGUIAdd(widget.window, widget.id, group, "radio", ahdGUIRequest{Text: text})
	if problem != "" {
		return 0, problem
	}
	ahdGUI.Lock()
	widget.items = append(widget.items, text)
	ahdGUI.Unlock()
	return handle, ""
}

func AhdGUISetRadioText(handle int64, text string) string {
	if len([]rune(text)) == 0 || len([]rune(text)) > 256 {
		return "a radio option must be between 1 and 256 characters"
	}
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	if widget.kind != "radio" {
		return "the widget does not support that request"
	}
	if problem := AhdGUISetText(handle, text); problem != "" {
		return problem
	}
	index := -1
	for at, kid := range ahdGUIKids(widget.parent, "radio") {
		if kid.id == widget.id {
			index = at
		}
	}
	ahdGUI.Lock()
	group := ahdGUI.widgets[widget.parent]
	if group != nil && index >= 0 && index < len(group.items) {
		group.items[index] = text
	}
	ahdGUI.Unlock()
	return ""
}

func AhdGUIAddProgress(container int64, maximum int64) (int64, string) {
	if maximum < 1 || maximum > 1_000_000 {
		return 0, "progress maximum must be between 1 and 1000000"
	}
	parent, problem := ahdGUIContainer(container)
	if problem != "" {
		return 0, problem
	}
	handle, problem := ahdGUIAdd(parent.window, parent.id, container, "progress", ahdGUIRequest{Width: int(maximum)})
	if problem != "" {
		return 0, problem
	}
	ahdGUI.Lock()
	if created := ahdGUI.widgets[handle]; created != nil {
		created.maximum = int(maximum)
	}
	ahdGUI.Unlock()
	return handle, ""
}

func AhdGUISetProgressValue(handle int64, value int64) string {
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	if widget.kind != "progress" {
		return "the widget does not support that request"
	}
	if value < 0 || value > int64(widget.maximum) {
		return "progress value must be between 0 and the maximum"
	}
	selected := int(value)
	return ahdGUIChange(handle, ahdGUIRequest{Kind: "amount", Selected: &selected}, func(*ahdGUIWidget) {})
}

func AhdGUIProgressValue(handle int64) (int64, string) {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return 0, problem
	}
	if widget.kind != "progress" {
		return 0, "the widget does not support that request"
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if final, ok := window.finals[widget.id]; ok || !window.alive {
		if final.Selected == nil || *final.Selected < 0 {
			return 0, ""
		}
		return int64(*final.Selected), ""
	}
	reply, problem := window.request(ahdGUIRequest{Op: "get", Widget: widget.id}, ahdGUIRequestTimeout)
	if problem != "" {
		return 0, problem
	}
	if reply.Selected == nil || *reply.Selected < 0 {
		return 0, "the GUI window helper sent an invalid response"
	}
	return int64(*reply.Selected), ""
}

func AhdGUIProgressMaximum(handle int64) (int64, string) {
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return 0, problem
	}
	if widget.kind != "progress" {
		return 0, "the widget does not support that request"
	}
	return int64(widget.maximum), ""
}

func AhdGUISetProgressSpin(handle int64, spin bool) string {
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	if widget.kind != "progress" {
		return "the widget does not support that request"
	}
	return ahdGUIChange(handle, ahdGUIRequest{Kind: "indeterminate", Checked: &spin}, func(*ahdGUIWidget) {})
}

func AhdGUIProgressSpin(handle int64) (bool, string) {
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return false, problem
	}
	if widget.kind != "progress" {
		return false, "the widget does not support that request"
	}
	return AhdGUIChecked(handle)
}

func AhdGUIAddDate(container int64, year, month, day int64) (int64, string) {
	if year > 9999 || month > 12 || day > 31 || !ahdGUIRealDate(int(year), int(month), int(day)) {
		return 0, "the date is not a real calendar day"
	}
	parent, problem := ahdGUIContainer(container)
	if problem != "" {
		return 0, problem
	}
	selected := int(day)
	return ahdGUIAdd(parent.window, parent.id, container, "date", ahdGUIRequest{Width: int(year), Height: int(month), Selected: &selected})
}

func AhdGUISetDate(handle int64, year, month, day int64) string {
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	if widget.kind != "date" {
		return "the widget does not support that request"
	}
	if year > 9999 || month > 12 || day > 31 || !ahdGUIRealDate(int(year), int(month), int(day)) {
		return "the date is not a real calendar day"
	}
	selected := int(day)
	return ahdGUIChange(handle, ahdGUIRequest{Kind: "date", Width: int(year), Height: int(month), Selected: &selected}, func(*ahdGUIWidget) {})
}

func AhdGUIDate(handle int64) (int64, int64, int64, string) {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return 0, 0, 0, problem
	}
	if widget.kind != "date" {
		return 0, 0, 0, "the widget does not support that request"
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	text := ""
	if final, ok := window.finals[widget.id]; ok || !window.alive {
		text = final.Text
	} else {
		reply, problem := window.request(ahdGUIRequest{Op: "get", Widget: widget.id}, ahdGUIRequestTimeout)
		if problem != "" {
			return 0, 0, 0, problem
		}
		if reply.Text == nil {
			return 0, 0, 0, "the GUI window helper sent an invalid response"
		}
		text = *reply.Text
	}
	year, month, day, ok := ahdGUIParseDate(text)
	if !ok {
		return 0, 0, 0, "the GUI window helper sent an invalid response"
	}
	return int64(year), int64(month), int64(day), ""
}

func AhdGUIOnDateChange(handle int64, handler func(year, month, day int64)) string {
	if handler == nil {
		return "onChange needs a Function"
	}
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	if widget.kind != "date" {
		return "the widget does not support that request"
	}
	return ahdGUIListen(handle, "date", func(event ahdGUIEvent) {
		year, month, day, ok := ahdGUIParseDate(event.Text)
		if !ok {
			return
		}
		handler(int64(year), int64(month), int64(day))
	})
}

func AhdGUIAddTabsChecked(container int64) int64 {
	handle, problem := AhdGUIAddTabs(container)
	AhdGUICheck(problem)
	return handle
}

func AhdGUIAddPageChecked(tabs int64, title string) int64 {
	handle, problem := AhdGUIAddPage(tabs, title)
	AhdGUICheck(problem)
	return handle
}

func AhdGUITabIndexChecked(handle int64) int64 {
	index, problem := AhdGUITabIndex(handle)
	AhdGUICheck(problem)
	return index
}

func AhdGUIAddRadioGroupChecked(container int64) int64 {
	handle, problem := AhdGUIAddRadioGroup(container)
	AhdGUICheck(problem)
	return handle
}

func AhdGUIAddOptionChecked(group int64, text string) int64 {
	handle, problem := AhdGUIAddOption(group, text)
	AhdGUICheck(problem)
	return handle
}

func AhdGUIAddProgressChecked(container int64, maximum int64) int64 {
	handle, problem := AhdGUIAddProgress(container, maximum)
	AhdGUICheck(problem)
	return handle
}

func AhdGUIProgressValueChecked(handle int64) int64 {
	value, problem := AhdGUIProgressValue(handle)
	AhdGUICheck(problem)
	return value
}

func AhdGUIProgressMaximumChecked(handle int64) int64 {
	value, problem := AhdGUIProgressMaximum(handle)
	AhdGUICheck(problem)
	return value
}

func AhdGUIProgressSpinChecked(handle int64) bool {
	spin, problem := AhdGUIProgressSpin(handle)
	AhdGUICheck(problem)
	return spin
}

func AhdGUIAddDateChecked(container int64, year, month, day int64) int64 {
	handle, problem := AhdGUIAddDate(container, year, month, day)
	AhdGUICheck(problem)
	return handle
}

func AhdGUIDateYearChecked(handle int64) int64 {
	year, _, _, problem := AhdGUIDate(handle)
	AhdGUICheck(problem)
	return year
}

func AhdGUIDateMonthChecked(handle int64) int64 {
	_, month, _, problem := AhdGUIDate(handle)
	AhdGUICheck(problem)
	return month
}

func AhdGUIDateDayChecked(handle int64) int64 {
	_, _, day, problem := AhdGUIDate(handle)
	AhdGUICheck(problem)
	return day
}
