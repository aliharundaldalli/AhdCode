package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"

	"golang.org/x/image/font"
)

// A Tabs control holds at most this many pages, and a radio group at most
// this many options. A progress maximum stays inside the same bound a
// program can name without a silent clamp.
const (
	maxTabPages   = 32
	maxRadioItems = 64
	maxProgress   = 1_000_000
	minDateYear   = 1
	maxDateYear   = 9999
)

var (
	errTabIndex   = errors.New("the tab index is outside the pages")
	errRadioIndex = errors.New("the option index is outside the group")
	errBadDate    = errors.New("the date is not a real calendar day")
)

// validateNewControl checks a tabs page, radio option, progress bar, or date
// before it is stored. The caller holds m.mu.
func validateNewControl(m *model, parent int64, s spec) error {
	owner := m.widgets[parent]
	if owner != nil && owner.kind == kindTabs && s.kind != kindPage {
		return errors.New("a Tabs control only holds pages")
	}
	if owner != nil && owner.kind == kindRadioGroup && s.kind != kindRadio {
		return errors.New("only a radio group holds options")
	}
	switch s.kind {
	case kindPage:
		if owner == nil || owner.kind != kindTabs {
			return errors.New("a page belongs to a Tabs control")
		}
		if len([]rune(s.text)) == 0 {
			return errors.New("a tab needs a title")
		}
		if len([]rune(s.text)) > 256 {
			return errors.New("a tab title must be at most 256 characters")
		}
		if len(m.pages(owner)) >= maxTabPages {
			return errors.New("a Tabs control holds at most 32 pages")
		}
	case kindRadio:
		if owner == nil || owner.kind != kindRadioGroup {
			return errors.New("only a radio group holds options")
		}
		if len([]rune(s.text)) == 0 || len([]rune(s.text)) > 256 {
			return errors.New("a radio option must be between 1 and 256 characters")
		}
		if len(m.radios(owner)) >= maxRadioItems {
			return errors.New("a radio group holds at most 64 options")
		}
	case kindProgress:
		if s.maximum < 1 || s.maximum > maxProgress {
			return errors.New("progress maximum must be between 1 and 1000000")
		}
	case kindDate:
		if !realDate(s.year, s.month, s.day) {
			return errBadDate
		}
	case kindCanvas:
		if s.fitW < 1 || s.fitH < 1 || s.fitW > maxDimension || s.fitH > maxDimension {
			return errors.New("a Canvas must be between 1 and 4096 pixels on each side")
		}
	}
	return nil
}

func (m *model) pages(tabs *widget) []*widget   { return m.childrenOf(tabs, kindPage) }
func (m *model) radios(group *widget) []*widget { return m.childrenOf(group, kindRadio) }

func (m *model) childrenOf(parent *widget, kind string) []*widget {
	if parent == nil {
		return nil
	}
	var children []*widget
	for _, id := range parent.children {
		child := m.widgets[id]
		if child != nil && child.kind == kind {
			children = append(children, child)
		}
	}
	return children
}

func widgetIndex(widgets []*widget, id int64) int {
	for index, widget := range widgets {
		if widget.id == id {
			return index
		}
	}
	return -1
}

// ---- calendar ------------------------------------------------------------

func leapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func daysInMonth(year, month int) int {
	days := []int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if month < 1 || month > 12 {
		return 0
	}
	if month == 2 && leapYear(year) {
		return 29
	}
	return days[month]
}

func realDate(year, month, day int) bool {
	if year < minDateYear || year > maxDateYear {
		return false
	}
	return day >= 1 && day <= daysInMonth(year, month)
}

func formatDate(year, month, day int) string {
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

// ---- progress and date writes --------------------------------------------

func (m *model) setAmount(id int64, value int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	if target.kind != kindProgress {
		return errWrongKind
	}
	if value < 0 || value > target.maximum {
		return errors.New("progress value must be between 0 and the maximum")
	}
	target.selected = value
	m.dirty = true
	return nil
}

func (m *model) setSpin(id int64, spin bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	if target.kind != kindProgress {
		return errWrongKind
	}
	target.spin = spin
	m.dirty = true
	return nil
}

func (m *model) setDate(id int64, year, month, day int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	if target.kind != kindDate {
		return errWrongKind
	}
	if !realDate(year, month, day) {
		return errBadDate
	}
	target.year, target.month, target.day = year, month, day
	target.digits = ""
	m.dirty = true
	return nil
}

// tickAnimation advances a visible indeterminate progress bar by one step.
func (m *model) tickAnimation() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range m.order {
		target := m.widgets[id]
		if target.kind == kindProgress && target.spin && m.shown(id) && m.tabShown(id) && target.w > 0 {
			m.phase++
			m.dirty = true
			return
		}
	}
}

// ---- tabs ----------------------------------------------------------------

func (m *model) measureTabs(target *widget) {
	width, height := rowHeight, 0
	pages := m.pages(target)
	for _, page := range pages {
		m.measureWidget(page.id)
	}
	if target.selected >= 0 && target.selected < len(pages) && !pages[target.selected].hidden {
		width = max(width, pages[target.selected].natW)
		height = pages[target.selected].natH
	}
	titles, shown := 0, 0
	for _, page := range pages {
		if page.hidden {
			continue
		}
		if shown > 0 {
			titles += 4
		}
		shown++
		titles += m.measure(string(page.text)) + 2*buttonPadding
	}
	target.natW = max(width, titles)
	target.natH = rowHeight + height
}

func (m *model) placeTabs(target *widget) {
	pages := m.pages(target)
	top := target.y + rowHeight
	height := max(target.h-rowHeight, 0)
	for index, page := range pages {
		if index == target.selected && !page.hidden {
			m.place(page.id, target.x, top, target.w, height)
			continue
		}
		m.zeroTree(page.id)
	}
}

func (m *model) zeroTree(id int64) {
	target := m.widgets[id]
	if target == nil {
		return
	}
	target.x, target.y, target.w, target.h = 0, 0, 0, 0
	for _, child := range target.children {
		m.zeroTree(child)
	}
}

// tabShown reports whether id sits on the active page of every Tabs ancestor.
// The caller holds m.mu. A hidden active page shows nothing.
func (m *model) tabShown(id int64) bool {
	for id != 0 {
		target := m.widgets[id]
		if target == nil {
			return false
		}
		parent := m.widgets[target.parent]
		if parent != nil && parent.kind == kindTabs && target.kind == kindPage {
			pages := m.pages(parent)
			if target.hidden || parent.selected < 0 || parent.selected >= len(pages) || pages[parent.selected].id != target.id {
				return false
			}
		}
		id = target.parent
	}
	return true
}

type tabHeader struct {
	index int
	x     int
	width int
}

func (m *model) tabHeaders(target *widget) []tabHeader {
	pages := m.pages(target)
	var headers []tabHeader
	total := 0
	for index, page := range pages {
		if page.hidden {
			continue
		}
		width := m.measure(string(page.text)) + 2*buttonPadding
		if len(headers) > 0 {
			total += 4
		}
		headers = append(headers, tabHeader{index: index, width: width})
		total += width
	}
	if total > target.w && total > 0 {
		used := 0
		for index := range headers {
			width := headers[index].width * target.w / total
			if width < 1 {
				width = 1
			}
			headers[index].width = width
			used += width
		}
		if len(headers) > 0 && used < target.w {
			headers[len(headers)-1].width += target.w - used
		}
	}
	x := target.x
	for index := range headers {
		headers[index].x = x
		x += headers[index].width
	}
	return headers
}

func (m *model) tabAt(x, y int) (int64, int, bool) {
	for index := len(m.order) - 1; index >= 0; index-- {
		target := m.widgets[m.order[index]]
		if target.kind != kindTabs || !m.shown(target.id) || !m.tabShown(target.id) {
			continue
		}
		if x < target.x || y < target.y || x >= target.x+target.w || y >= target.y+rowHeight || !m.insideScrolls(target.id, x, y) {
			continue
		}
		for _, header := range m.tabHeaders(target) {
			if x >= header.x && x < header.x+header.width {
				return target.id, header.index, true
			}
		}
	}
	return 0, 0, false
}

func (m *model) selectTab(target *widget, index int) error {
	pages := m.pages(target)
	if index < 0 || index >= len(pages) {
		return errTabIndex
	}
	target.selected = index
	if m.focus != 0 && !m.tabShown(m.focus) {
		m.focus = 0
	}
	m.dirty = true
	return nil
}

// userTab selects a visible page as the user. restore puts the focus back on
// that page's last focused descendant. The caller holds m.mu.
func (m *model) userTab(target *widget, index int, restore bool) {
	pages := m.pages(target)
	if index < 0 || index >= len(pages) || pages[index].hidden {
		return
	}
	if target.selected == index {
		m.focus = target.id
		m.dirty = true
		return
	}
	m.capturePageFocus(target)
	target.selected = index
	m.noteChange(target.id)
	if !m.tabShown(m.focus) {
		m.focus = target.id
	}
	if restore {
		m.restorePageFocus(pages[index])
	}
	m.dirty = true
}

func (m *model) capturePageFocus(tabs *widget) {
	id := m.focus
	for id != 0 {
		target := m.widgets[id]
		if target == nil {
			return
		}
		parent := m.widgets[target.parent]
		if parent != nil && parent.id == tabs.id && target.kind == kindPage {
			target.remember = m.focus
			return
		}
		id = target.parent
	}
}

func (m *model) restorePageFocus(page *widget) {
	remembered := m.widgets[page.remember]
	if remembered == nil || !remembered.focusable() || !m.shown(remembered.id) || !m.tabShown(remembered.id) || !m.inside(remembered.id, page.id) {
		return
	}
	m.focus = remembered.id
}

func (m *model) notePageFocus(id int64) {
	focus := m.focus
	for id != 0 {
		target := m.widgets[id]
		if target == nil {
			return
		}
		parent := m.widgets[target.parent]
		if parent != nil && parent.kind == kindTabs && target.kind == kindPage {
			target.remember = focus
			return
		}
		id = target.parent
	}
}

func (m *model) moveTabs(target *widget, key string) bool {
	delta := 0
	switch key {
	case "ArrowLeft":
		delta = -1
	case "ArrowRight":
		delta = 1
	default:
		return false
	}
	pages := m.pages(target)
	if len(pages) == 0 {
		return true
	}
	start := target.selected
	if start < 0 {
		start = 0
	}
	for step := 1; step <= len(pages); step++ {
		next := start + delta*step
		next = ((next % len(pages)) + len(pages)) % len(pages)
		if pages[next].hidden {
			continue
		}
		m.userTab(target, next, false)
		return true
	}
	return true
}

// ---- radio ---------------------------------------------------------------

func (m *model) selectRadio(target *widget, index int) error {
	options := m.radios(target)
	if index < -1 || index >= len(options) {
		return errRadioIndex
	}
	target.selected = index
	m.dirty = true
	return nil
}

func (m *model) chooseRadio(option *widget) {
	if option.disabled || option.hidden || !m.shown(option.id) || !m.tabShown(option.id) {
		return
	}
	group := m.widgets[option.parent]
	if group == nil || group.kind != kindRadioGroup {
		return
	}
	index := widgetIndex(m.radios(group), option.id)
	if index < 0 || group.selected == index {
		return
	}
	group.selected = index
	m.noteChange(group.id)
	m.dirty = true
}

func (m *model) userRadio(group *widget, index int) error {
	options := m.radios(group)
	if index < -1 || index >= len(options) {
		return errRadioIndex
	}
	if index >= 0 && (options[index].disabled || options[index].hidden) {
		return nil
	}
	if group.selected == index {
		return nil
	}
	group.selected = index
	m.noteChange(group.id)
	m.dirty = true
	return nil
}

func (m *model) moveRadio(option *widget, key string) bool {
	delta := 0
	switch key {
	case "ArrowUp", "ArrowLeft":
		delta = -1
	case "ArrowDown", "ArrowRight":
		delta = 1
	default:
		return false
	}
	group := m.widgets[option.parent]
	if group == nil || group.kind != kindRadioGroup {
		return true
	}
	options := m.radios(group)
	start := widgetIndex(options, option.id)
	if start < 0 || len(options) == 0 {
		return true
	}
	for step := 1; step <= len(options); step++ {
		next := start + delta*step
		next = ((next % len(options)) + len(options)) % len(options)
		candidate := options[next]
		if candidate.disabled || candidate.hidden || !m.shown(candidate.id) {
			continue
		}
		if group.selected != next {
			group.selected = next
			m.noteChange(group.id)
		}
		m.focus = candidate.id
		m.dirty = true
		return true
	}
	return true
}

func (m *model) radioOn(option *widget) bool {
	group := m.widgets[option.parent]
	if group == nil || group.selected < 0 {
		return false
	}
	options := m.radios(group)
	return group.selected < len(options) && options[group.selected].id == option.id
}

// ---- date keys -----------------------------------------------------------

func (m *model) pickSegment(target *widget, x int) {
	m.commitDigits(target)
	target.segment = datePart(target, x)
	m.dirty = true
}

func datePart(target *widget, x int) int {
	if target.w <= 0 {
		return 0
	}
	part := target.w / 3
	if part < 1 {
		part = 1
	}
	offset := x - target.x
	if offset < part {
		return 0
	}
	if offset < 2*part {
		return 1
	}
	return 2
}

func (m *model) shiftSegment(target *widget, delta int) {
	m.commitDigits(target)
	target.segment = ((target.segment+delta)%3 + 3) % 3
	m.dirty = true
}

func (m *model) adjustDate(target *widget, delta int) {
	m.commitDigits(target)
	year, month, day := target.year, target.month, target.day
	switch target.segment {
	case 0:
		year += delta
		if year < minDateYear {
			year = minDateYear
		}
		if year > maxDateYear {
			year = maxDateYear
		}
	case 1:
		month += delta
		if month < 1 {
			month = 1
		}
		if month > 12 {
			month = 12
		}
	default:
		day += delta
	}
	limit := daysInMonth(year, month)
	if day > limit {
		day = limit
	}
	if day < 1 {
		day = 1
	}
	if year == target.year && month == target.month && day == target.day {
		return
	}
	target.year, target.month, target.day = year, month, day
	m.noteChange(target.id)
	m.dirty = true
}

func (m *model) typeDate(target *widget, runes []rune) {
	for _, r := range runes {
		if r < '0' || r > '9' {
			continue
		}
		limit := 4
		if target.segment != 0 {
			limit = 2
		}
		if len(target.digits) >= limit {
			target.digits = ""
		}
		target.digits += string(r)
		if len(target.digits) == limit {
			m.commitDigits(target)
		}
	}
	m.dirty = true
}

func (m *model) commitDigits(target *widget) {
	if target.digits == "" {
		return
	}
	number := 0
	for _, r := range target.digits {
		number = number*10 + int(r-'0')
	}
	year, month, day := target.year, target.month, target.day
	switch target.segment {
	case 0:
		year = number
	case 1:
		month = number
	default:
		day = number
	}
	target.digits = ""
	if !realDate(year, month, day) || (year == target.year && month == target.month && day == target.day) {
		m.dirty = true
		return
	}
	target.year, target.month, target.day = year, month, day
	m.noteChange(target.id)
	m.dirty = true
}

func (m *model) dateKey(target *widget, key string) bool {
	switch key {
	case "ArrowLeft":
		m.shiftSegment(target, -1)
	case "ArrowRight":
		m.shiftSegment(target, 1)
	case "ArrowUp":
		m.adjustDate(target, 1)
	case "ArrowDown":
		m.adjustDate(target, -1)
	case "Backspace":
		if target.digits != "" {
			target.digits = ""
			m.dirty = true
		}
	default:
		return false
	}
	return true
}

// ---- drawing -------------------------------------------------------------

func (m *model) drawTabs(img *image.RGBA, face font.Face, f frame, w *widget, box image.Rectangle, line int, focused bool, foreground color.Color) {
	if w.background != nil {
		fill(img, box, *w.background)
	}
	headers := m.tabHeaders(w)
	for _, header := range headers {
		rect := scaled(f.scale, header.x, w.y, header.width, rowHeight).Intersect(box)
		if rect.Empty() {
			continue
		}
		paint := colorButton
		if header.index == w.selected {
			paint = colorField
		}
		fill(img, rect, paint)
		border := colorBorder
		if focused && header.index == w.selected {
			border = colorFocus
		}
		outline(img, rect, border, line)
		page := m.pages(w)[header.index]
		text(img, face, rect, rect.Min.X+int(buttonPadding*f.scale), rect, string(page.text), foreground)
	}
}

func (m *model) drawRadio(img *image.RGBA, face font.Face, f frame, w *widget, box image.Rectangle, line int, focused bool, foreground color.Color) {
	if w.background != nil {
		fill(img, box, *w.background)
	}
	size := int(checkboxBox * f.scale)
	if size < 1 {
		size = 1
	}
	cx := box.Min.X + size/2
	cy := box.Min.Y + box.Dy()/2
	radius := size / 2
	if m.radioOn(w) {
		fillCircle(img, cx, cy, radius, colorFocus)
		fillCircle(img, cx, cy, max(radius/2, 1), colorField)
	} else {
		fillCircle(img, cx, cy, radius, colorBorder)
		fillCircle(img, cx, cy, max(radius-line*2, 1), colorField)
	}
	if focused {
		fillCircle(img, cx, cy, radius+2*line, colorFocus)
		if m.radioOn(w) {
			fillCircle(img, cx, cy, radius, colorFocus)
			fillCircle(img, cx, cy, max(radius/2, 1), colorField)
		} else {
			fillCircle(img, cx, cy, radius, colorField)
		}
	}
	x := box.Min.X + size + int(checkboxGap*f.scale)
	text(img, face, box, x, box, string(w.text), foreground)
}

func (m *model) drawProgress(img *image.RGBA, f frame, w *widget, box image.Rectangle, line int) {
	fill(img, box, colorField)
	outline(img, box, colorBorder, line)
	inner := box.Inset(2 * line)
	if inner.Empty() {
		return
	}
	if w.spin {
		span := inner.Dx()
		block := max(span/3, 4*line)
		travel := span + block
		pos := 0
		if travel > 0 {
			pos = (m.phase * max(line*3, 3)) % travel
		}
		start := inner.Min.X + pos - block
		fill(img, image.Rect(start, inner.Min.Y, start+block, inner.Max.Y).Intersect(inner), colorFocus)
		return
	}
	span := inner.Dx()
	filled := 0
	if w.maximum > 0 {
		filled = span * w.selected / w.maximum
	}
	if filled > 0 {
		fill(img, image.Rect(inner.Min.X, inner.Min.Y, inner.Min.X+filled, inner.Max.Y), colorFocus)
	}
}

func (m *model) drawDate(img *image.RGBA, face font.Face, f frame, w *widget, box image.Rectangle, line int, focused bool, foreground color.Color) {
	fill(img, box, or(w.background, colorField))
	border := colorBorder
	if focused {
		border = colorFocus
	}
	outline(img, box, border, line)
	part := max(w.w/3, 1)
	labels := []string{
		fmt.Sprintf("%04d", w.year),
		fmt.Sprintf("%02d", w.month),
		fmt.Sprintf("%02d", w.day),
	}
	for index := range labels {
		if focused && index == w.segment && w.digits != "" {
			labels[index] = w.digits
		}
		rect := scaled(f.scale, w.x+index*part, w.y, part, w.h).Intersect(box)
		if focused && index == w.segment {
			fill(img, rect.Inset(line), colorTextSelect)
		}
		text(img, face, rect, rect.Min.X+int(textInputInset*f.scale), rect, labels[index], foreground)
	}
}

func fillCircle(img *image.RGBA, cx, cy, radius int, c color.Color) {
	if radius < 1 {
		return
	}
	bounds := img.Bounds()
	limit := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			if x < bounds.Min.X || y < bounds.Min.Y || x >= bounds.Max.X || y >= bounds.Max.Y {
				continue
			}
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= limit {
				img.Set(x, y, c)
			}
		}
	}
}
