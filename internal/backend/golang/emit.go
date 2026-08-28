package golang

import (
	"strconv"
	"strings"
)

// emitter is a minimal indentation-aware writer. Generated source is always
// re-parsed and formatted by gofmt, so the emitter only has to produce valid,
// deterministic Go text.
type emitter struct {
	out    strings.Builder
	indent int
}

func (writer *emitter) line(text string) {
	if text != "" {
		writer.out.WriteString(strings.Repeat("\t", writer.indent))
		writer.out.WriteString(text)
	}
	writer.out.WriteByte('\n')
}

func (writer *emitter) open(text string) {
	writer.line(text)
	writer.indent++
}

func (writer *emitter) close(text string) {
	if writer.indent > 0 {
		writer.indent--
	}
	writer.line(text)
}

func (writer *emitter) blank() { writer.out.WriteByte('\n') }

// raw appends already-emitted source at the current top level.
func (writer *emitter) raw(text string) { writer.out.WriteString(text) }

func (writer *emitter) String() string { return writer.out.String() }

func itoa(value int) string { return strconv.Itoa(value) }

func quote(value string) string { return strconv.Quote(value) }
