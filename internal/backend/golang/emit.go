package golang

import "strings"

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

func (writer *emitter) String() string { return writer.out.String() }
