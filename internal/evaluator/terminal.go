package evaluator

import (
	"io"
	"os"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

// The Terminal standard module's REPL implementation. Output goes to the
// session's own writers: Terminal.emit and Terminal.pretty to Output, the
// writer write uses, and Terminal.error to ErrorOutput. The capability queries
// ask about Output when it is an operating-system file, through the same
// platform functions and color policy a compiled program uses, and report "not
// a terminal" for any other writer.

const terminalIndent = "    "

// errorWriter is the writer Terminal.error uses. Like Output, an unset
// ErrorOutput discards what is written.
func (session *Session) errorWriter() io.Writer {
	if session.ErrorOutput == nil {
		return io.Discard
	}
	return session.ErrorOutput
}

// outputTerminal reports the file descriptor behind Output, if Output is an
// operating-system file.
func (session *Session) outputTerminal() (uintptr, bool) {
	file, ok := session.Output.(*os.File)
	if !ok || file == nil {
		return 0, false
	}
	return file.Fd(), true
}

// flushOutput writes any text a buffered Output is holding. It reports false
// when the writer fails to flush.
func (session *Session) flushOutput() bool {
	if flusher, ok := session.Output.(interface{ Flush() error }); ok {
		return flusher.Flush() == nil
	}
	return true
}

func (session *Session) terminalBuiltin(name string, arguments []any) any {
	text := func(index int, fallback string) string {
		if index < len(arguments) && arguments[index] != nil {
			return arguments[index].(string)
		}
		return fallback
	}
	flag := func(index int) bool {
		return index < len(arguments) && arguments[index] != nil && arguments[index].(bool)
	}
	switch name {
	case "emit":
		parts := session.requireList(arguments[0])
		items := make([]string, len(parts.Items))
		for index, item := range parts.Items {
			items[index] = item.(string)
		}
		session.writeText(ahdruntime.AhdTerminalJoin(items, text(1, " "), text(2, "\n")))
		return Nothing
	case "error":
		// Pending output first, so both streams keep program order.
		session.flushOutput()
		_, _ = io.WriteString(session.errorWriter(), text(0, "")+text(1, "\n"))
		return Nothing
	case "flush":
		if !session.flushOutput() {
			session.raise("TerminalError", "standard output could not be flushed")
		}
		return Nothing
	case "isInteractive":
		fd, ok := session.outputTerminal()
		return ok && ahdruntime.AhdTerminalIsTerminal(fd)
	case "width", "height":
		fd, ok := session.outputTerminal()
		if !ok {
			return nil
		}
		width, height := ahdruntime.AhdTerminalDimensions(fd)
		if name == "height" {
			width = height
		}
		if width == nil {
			return nil
		}
		return *width
	case "supportsColor":
		return session.terminalColor()
	case "style":
		styled, problem := ahdruntime.AhdTerminalStyle(text(0, ""), text(1, "default"), text(2, "default"),
			flag(3), flag(4), session.terminalColor())
		if problem != "" {
			session.raise("TerminalError", problem)
		}
		return styled
	}
	session.raise("Error", "unsupported Terminal function "+name)
	return nil
}

func (session *Session) terminalColor() bool {
	fd, ok := session.outputTerminal()
	return ok && ahdruntime.AhdTerminalColorEnabled(fd)
}

// prettyOf lays out one Terminal.pretty argument. A List or Pair is spread
// over lines; every other value is exactly the text write produces, including
// a Class's own CStr, which is why this reads the expression rather than only
// its value.
func (session *Session) prettyOf(expression ir.Expr, current *frame) string {
	switch expression.ExprMeta().Type.Kind {
	case ir.ListType, ir.PairType:
		return session.prettyRender(session.eval(expression, current), "", make(map[visit]bool))
	}
	return session.textOf(expression, current)
}

// prettyRender mirrors the native AhdTerminalPrettyList and
// AhdTerminalPrettyPair layout. Values inside a collection use str's nested
// form, and a collection reached again through itself is shown as str shows
// it.
func (session *Session) prettyRender(value any, indent string, seen map[visit]bool) string {
	switch item := value.(type) {
	case *List:
		if item == nil {
			return "null"
		}
		if len(item.Items) == 0 {
			return "[]"
		}
		key := visit{'l', item}
		if seen[key] {
			return "[...]"
		}
		seen[key] = true
		defer delete(seen, key)
		inner := indent + terminalIndent
		result := "[\n"
		for index, element := range item.Items {
			result += inner + session.prettyRender(element, inner, seen)
			if index < len(item.Items)-1 {
				result += ","
			}
			result += "\n"
		}
		return result + indent + "]"
	case *Pair:
		if item == nil {
			return "null"
		}
		if len(item.Keys) == 0 {
			return "{}"
		}
		key := visit{'p', item}
		if seen[key] {
			return "{...}"
		}
		seen[key] = true
		defer delete(seen, key)
		inner := indent + terminalIndent
		result := "{\n"
		for index, pairKey := range item.Keys {
			result += inner + session.render(pairKey, true, seen) + ": " + session.prettyRender(item.Values[pairKey], inner, seen)
			if index < len(item.Keys)-1 {
				result += ","
			}
			result += "\n"
		}
		return result + indent + "}"
	}
	return session.render(value, true, seen)
}
