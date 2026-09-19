package main

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ahdidentity"

	"github.com/hajimehoshi/ebiten/v2"
)

// The fallback dialog is an ordinary GUI window drawn by this helper, for
// systems without a usable dialog of their own (Linux: no desktop toolkit is
// linked, so none is assumed). It is built from the same widgets as a
// program's Window. A file dialog lists one folder at a time: Up goes to the
// parent folder, Open on a selected folder enters it, and Open, Save, or
// Select on a file or folder chooses it. The fallback chooses one file at a
// time, also for openFiles.

var errDialog = errors.New("the system dialog could not be shown")

// Fallback dialog labels.
const (
	folderSuffix = "/"
	maxListed    = maxItems
)

// dialogController drives one fallback dialog window.
type dialogController struct {
	req     dialogRequest
	session *session
	folder  string
	entries []string // listed names; a folder ends with folderSuffix
	result  dialogResult
	done    bool

	pathLabel, status        int64
	list, name               int64
	up, cancel, open, choose int64
}

// newDialogController builds the dialog's widgets on a fresh model. The
// session writes nowhere: its events go straight to the controller.
func newDialogController(req dialogRequest, start string) (*dialogController, error) {
	s := &session{
		model:      newModel(dialogTitle(req), 560, 420, measureText),
		in:         bufio.NewReader(strings.NewReader("")),
		out:        bufio.NewWriter(io.Discard),
		windowGone: make(chan struct{}),
		clicks:     map[int64]bool{},
		changes:    map[int64]bool{},
	}
	c := &dialogController{req: req, session: s, result: dialogResult{cancelled: true}}
	s.sink = c.handle
	m := s.model
	add := func(parent int64, value spec) int64 {
		value.selected = -1
		if value.kind == kindColumn || value.kind == kindRow {
			value.spacing = 8
		}
		id, err := m.addSpec(parent, value)
		if err != nil {
			panic("the fallback dialog could not build its widgets: " + err.Error())
		}
		return id
	}
	root, err := m.addSpec(0, spec{kind: kindColumn, spacing: 8, padding: 14, selected: -1})
	if err != nil {
		return nil, err
	}
	if req.kind == dialogMessage || req.kind == dialogConfirm {
		m.width, m.height = 440, 180
		for _, line := range strings.Split(req.text, "\n")[:min(strings.Count(req.text, "\n")+1, 20)] {
			add(root, spec{kind: kindLabel, text: line})
		}
		buttons := add(root, spec{kind: kindRow})
		if req.kind == dialogConfirm {
			c.cancel = add(buttons, spec{kind: kindButton, text: "Cancel"})
		}
		c.choose = add(buttons, spec{kind: kindButton, text: "OK"})
	} else {
		c.pathLabel = add(root, spec{kind: kindLabel})
		c.list = add(root, spec{kind: kindListBox})
		if req.kind == dialogSaveFile {
			c.name = add(root, spec{kind: kindTextInput, placeholder: "File name"})
			_ = m.setText(c.name, req.name)
		}
		c.status = add(root, spec{kind: kindLabel})
		buttons := add(root, spec{kind: kindRow})
		c.up = add(buttons, spec{kind: kindButton, text: "Up"})
		c.cancel = add(buttons, spec{kind: kindButton, text: "Cancel"})
		c.open = add(buttons, spec{kind: kindButton, text: "Open"})
		switch req.kind {
		case dialogSaveFile:
			c.choose = add(buttons, spec{kind: kindButton, text: "Save"})
		case dialogSelectFolder:
			c.choose = add(buttons, spec{kind: kindButton, text: "Select"})
		}
		m.setResizable(true)
		c.enter(start)
	}
	for _, id := range []int64{c.up, c.cancel, c.open, c.choose} {
		if id != 0 {
			s.clicks[id] = true
		}
	}
	if c.list != 0 {
		s.changes[c.list] = true
	}
	return c, nil
}

// enter lists a folder.
func (c *dialogController) enter(folder string) {
	m := c.session.model
	folder = filepath.Clean(folder)
	files, err := os.ReadDir(folder)
	if err != nil {
		_ = m.setText(c.status, "This folder cannot be read.")
		return
	}
	c.folder = folder
	var folders, names []string
	for _, file := range files {
		name := file.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		isFolder := file.IsDir()
		if file.Type()&os.ModeSymlink != 0 {
			if info, err := os.Stat(filepath.Join(folder, name)); err == nil {
				isFolder = info.IsDir()
			}
		}
		switch {
		case isFolder:
			folders = append(folders, name+folderSuffix)
		case c.req.kind != dialogSelectFolder && (len(c.req.extensions) == 0 || hasExtension(name, c.req.extensions)):
			names = append(names, name)
		}
	}
	sort.Strings(folders)
	sort.Strings(names)
	c.entries = append(folders, names...)
	if len(c.entries) > maxListed {
		c.entries = c.entries[:maxListed]
	}
	shown := make([]string, len(c.entries))
	for index, entry := range c.entries {
		shown[index] = firstRunes(entry, maxItemRunes)
	}
	_ = m.setItems(c.list, shown)
	_ = m.setText(c.pathLabel, firstRunes(folder, maxTextRunes))
	_ = m.setText(c.status, "")
}

func firstRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return text
}

// selectedEntry is the selected name, or "".
func (c *dialogController) selectedEntry() string {
	_, _, selected, _ := c.session.model.readAll(c.list)
	if selected < 0 || selected >= len(c.entries) {
		return ""
	}
	return c.entries[selected]
}

// handle reacts to a click or a change in the dialog window.
func (c *dialogController) handle(e event) {
	m := c.session.model
	switch {
	case e.Event == "change" && e.Widget == c.list && c.name != 0:
		// Choosing a file in a save dialog puts its name in the name field.
		if entry := c.selectedEntry(); entry != "" && !strings.HasSuffix(entry, folderSuffix) {
			_ = m.setText(c.name, entry)
		}
	case e.Event != "click":
	case e.Widget == c.up:
		c.enter(filepath.Dir(c.folder))
	case e.Widget == c.cancel:
		c.finish(dialogResult{cancelled: true})
	case e.Widget == c.open:
		entry := c.selectedEntry()
		switch {
		case strings.HasSuffix(entry, folderSuffix):
			c.enter(filepath.Join(c.folder, strings.TrimSuffix(entry, folderSuffix)))
		case entry != "" && c.req.kind != dialogSaveFile:
			c.finish(dialogResult{paths: []string{filepath.Join(c.folder, entry)}})
		}
	case e.Widget == c.choose:
		switch c.req.kind {
		case dialogMessage, dialogConfirm:
			c.finish(dialogResult{confirmed: true})
		case dialogSelectFolder:
			chosen := c.folder
			if entry := c.selectedEntry(); entry != "" {
				chosen = filepath.Join(c.folder, strings.TrimSuffix(entry, folderSuffix))
			}
			c.finish(dialogResult{paths: []string{chosen}})
		case dialogSaveFile:
			name, _, _ := m.read(c.name)
			name = strings.TrimSpace(name)
			if !validFileName(name) || name == "" {
				_ = m.setText(c.status, "Type a file name.")
				return
			}
			c.finish(dialogResult{paths: []string{filepath.Join(c.folder, name)}})
		}
	}
}

func (c *dialogController) finish(result dialogResult) {
	c.result, c.done = result, true
	c.session.closeRequested.Store(true)
}

// startFolder is where a file dialog opens: the working directory, or the
// home folder.
func startFolder() string {
	if folder, err := os.Getwd(); err == nil {
		return folder
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return string(filepath.Separator)
}

// fallbackDialog shows the dialog in a window of its own.
func fallbackDialog(req dialogRequest) (dialogResult, error) {
	c, err := newDialogController(req, startFolder())
	if err != nil {
		return dialogResult{}, err
	}
	m := c.session.model
	w := &window{session: c.session, ready: make(chan struct{}), scale: 1, title: m.title}
	ebiten.SetWindowTitle(m.title)
	ebiten.SetWindowSize(m.width, m.height)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	ahdidentity.Prepare()
	ebiten.SetTPS(30)
	if err := runGame(w); err != nil {
		return dialogResult{}, errDialog
	}
	select {
	case <-w.ready:
	default:
		return dialogResult{}, errDialog
	}
	if !c.done {
		return dialogResult{cancelled: true, confirmed: false}, nil
	}
	return c.result, nil
}
