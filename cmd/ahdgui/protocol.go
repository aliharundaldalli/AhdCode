package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"strings"
	"unicode/utf8"
)

// The protocol between AhdCode's GUI runtime and this helper is one JSON
// object per line in each direction. The runtime writes a request, carrying
// an id, to the helper's standard input; the helper answers each request with
// exactly one response line echoing that id. Besides responses, the helper
// writes event lines ({"event": ...}, never with an id): a click on a Button
// the program listens to, a key press when the program listens for keys, and
// one "closed" event when the window is gone. Standard error is never part of
// the protocol. The shape is duplicated field for field in
// internal/backend/golang/ahdruntime/gui.go, which cannot import this module.
const (
	protocolVersion = 2

	// maxRequestBytes bounds one request line; the longest field is a text
	// of at most maxTextRunes characters.
	maxRequestBytes = 64 << 10
	maxDimension    = 4096
	maxTitleRunes   = 256
	// maxTextRunes bounds every Label, Button, Checkbox, placeholder, and
	// TextInput text, so no request or event line can grow without bound.
	maxTextRunes = 4096
	// maxWidgets bounds the widgets of one Window.
	maxWidgets = 1024
	// maxSpacing bounds spacing and padding.
	maxSpacing = 1000
	// maxScriptEvents bounds the scripted events of a headless test Window.
	maxScriptEvents = 256
)

type request struct {
	Op string `json:"op"`
	ID int64  `json:"id,omitempty"`

	Version  int     `json:"version,omitempty"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	Title    string  `json:"title,omitempty"`
	Headless bool    `json:"headless,omitempty"`
	Script   []event `json:"script,omitempty"`

	Parent      int64  `json:"parent,omitempty"`
	Widget      int64  `json:"widget,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Text        string `json:"text,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Checked     *bool  `json:"checked,omitempty"`
	Spacing     int    `json:"spacing,omitempty"`
	Padding     int    `json:"padding,omitempty"`

	// Part and Color set one color: Part is foreground or background, and
	// Color is always #RRGGBBAA, already normalized by the runtime.
	Part    string `json:"part,omitempty"`
	Color   string `json:"color,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
}

// value is one TextInput or Checkbox reading.
type value struct {
	Widget  int64  `json:"widget"`
	Text    string `json:"text,omitempty"`
	Checked bool   `json:"checked,omitempty"`
}

type response struct {
	ID      int64   `json:"id,omitempty"`
	OK      bool    `json:"ok"`
	Error   string  `json:"error,omitempty"`
	Closed  bool    `json:"closed,omitempty"`
	Open    *bool   `json:"open,omitempty"`
	Widget  int64   `json:"widget,omitempty"`
	Text    *string `json:"text,omitempty"`
	Checked *bool   `json:"checked,omitempty"`
	// Values are the final TextInput and Checkbox readings, sent when the
	// window closes, so a program can still read them after wait returns.
	Values []value `json:"values,omitempty"`
}

// event is one line the helper writes on its own. In a headless test
// Window's script, "type" and "toggle" stand for the user typing into a
// TextInput and clicking a Checkbox; they change the model and are never sent.
type event struct {
	Event  string  `json:"event"`
	Widget int64   `json:"widget,omitempty"`
	Key    string  `json:"key,omitempty"`
	Text   string  `json:"text,omitempty"`
	Values []value `json:"values,omitempty"`
}

var errRequestTooLarge = errors.New("request exceeds the protocol size limit")

// readRequest reads and decodes one bounded request line. io.EOF means the
// runtime closed the pipe, which is how a helper learns its program ended.
func readRequest(reader *bufio.Reader) (request, error) {
	var line []byte
	for {
		chunk, isPrefix, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				break
			}
			return request{}, err
		}
		line = append(line, chunk...)
		if len(line) > maxRequestBytes {
			return request{}, errRequestTooLarge
		}
		if !isPrefix {
			break
		}
	}
	decoder := json.NewDecoder(strings.NewReader(string(line)))
	decoder.DisallowUnknownFields()
	var parsed request
	if err := decoder.Decode(&parsed); err != nil {
		return request{}, fmt.Errorf("malformed request: %w", err)
	}
	return parsed, nil
}

func writeLine(writer *bufio.Writer, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := writer.Write(append(encoded, '\n')); err != nil {
		return err
	}
	return writer.Flush()
}

// validText reports whether text is valid UTF-8 of at most limit characters
// with no NUL.
func validText(text string, limit int) bool {
	return utf8.ValidString(text) && utf8.RuneCountInString(text) <= limit && !strings.ContainsRune(text, 0)
}

// validateOpen checks the first request. The runtime has already validated
// everything; the helper checks again because it trusts no input.
func validateOpen(first request) error {
	if first.Op != "open" {
		return fmt.Errorf("the first request must be open, not %q", first.Op)
	}
	if first.Version != protocolVersion {
		return fmt.Errorf("protocol version %d is not supported (expected %d)", first.Version, protocolVersion)
	}
	if first.Width < 1 || first.Width > maxDimension || first.Height < 1 || first.Height > maxDimension {
		return fmt.Errorf("window size %dx%d is outside 1..%d", first.Width, first.Height, maxDimension)
	}
	if !validText(first.Title, maxTitleRunes) {
		return errors.New("invalid window title")
	}
	if len(first.Script) > 0 && !first.Headless {
		return errors.New("only a headless Window replays scripted events")
	}
	if len(first.Script) > maxScriptEvents {
		return errors.New("too many scripted events")
	}
	for _, scripted := range first.Script {
		switch {
		case scripted.Event == "click" || scripted.Event == "toggle":
		case scripted.Event == "type" && validText(scripted.Text, maxTextRunes):
		case scripted.Event == "key" && keyNameKnown(scripted.Key):
		default:
			return errors.New("invalid scripted event")
		}
	}
	return nil
}

// parseColor reads the runtime's normalized #RRGGBBAA form, a straight
// (non-premultiplied) color. The runtime
// accepts the nine names, #RRGGBB, and #RRGGBBAA and always sends the long
// form; the helper checks again because it trusts no input.
func parseColor(text string) (color.NRGBA, bool) {
	if len(text) != 9 || text[0] != '#' {
		return color.NRGBA{}, false
	}
	var channels [4]uint8
	for index := range channels {
		high, okHigh := hexDigit(text[1+2*index])
		low, okLow := hexDigit(text[2+2*index])
		if !okHigh || !okLow {
			return color.NRGBA{}, false
		}
		channels[index] = high<<4 | low
	}
	return color.NRGBA{channels[0], channels[1], channels[2], channels[3]}, true
}

func hexDigit(character byte) (uint8, bool) {
	switch {
	case character >= '0' && character <= '9':
		return character - '0', true
	case character >= 'a' && character <= 'f':
		return character - 'a' + 10, true
	case character >= 'A' && character <= 'F':
		return character - 'A' + 10, true
	}
	return 0, false
}
