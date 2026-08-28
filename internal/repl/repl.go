// Package repl implements an interactive AhdCode session by replaying the
// successfully committed session source through the ordinary compiler and
// native runtime. It deliberately contains no evaluator or semantic rules.
package repl

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"ahdcode/internal/build"
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/lexer"
	"ahdcode/internal/parser"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/syntax/token"
)

const (
	PrimaryPrompt      = "ahd> "
	ContinuationPrompt = "...> "
)

// Run starts a session and returns when input reaches EOF. Every successful
// submission is appended to one module source; failed submissions are never
// committed, so semantic/runtime failures leave the prior state available.
func Run(input io.Reader, output, errorOutput io.Writer, version string) int {
	directory, err := os.MkdirTemp("", "ahdcode-repl-")
	if err != nil {
		fmt.Fprintf(errorOutput, "error [REPL001]\ncould not create REPL workspace: %v\n", err)
		return 1
	}
	defer os.RemoveAll(directory)
	entry := filepath.Join(directory, "Session.ahd")
	reader := bufio.NewReader(input)
	committedSource := ""
	committedOutput := ""
	pending := ""
	fmt.Fprintln(output, version)

	for {
		prompt := PrimaryPrompt
		if pending != "" {
			prompt = ContinuationPrompt
		}
		fmt.Fprint(output, prompt)
		line, readError := reader.ReadString('\n')
		if len(line) == 0 && readError == io.EOF {
			if pending == "" {
				return 0
			}
			incomplete, optionalClause := incompleteState(pending)
			if incomplete && !optionalClause {
				reportSyntax(errorOutput, pending)
				return 0
			}
		}
		pending += line
		if strings.TrimSpace(pending) == "" {
			pending = ""
			if readError == io.EOF {
				return 0
			}
			continue
		}
		incomplete, optionalClause := incompleteState(pending)
		if incomplete && !(readError == io.EOF && optionalClause) {
			if readError == io.EOF {
				reportSyntax(errorOutput, pending)
				return 0
			}
			continue
		}

		candidate := appendSubmission(committedSource, pending)
		pending = ""
		if err := os.WriteFile(entry, []byte(candidate), 0o600); err != nil {
			fmt.Fprintf(errorOutput, "error [REPL001]\ncould not write REPL source: %v\n", err)
			return 1
		}
		var stdout, stderr bytes.Buffer
		// REPL command input belongs to the session reader; handing that reader to
		// an external executable would let os/exec prefetch future commands. The
		// v0.1 REPL therefore gives each replay an isolated EOF runtime input.
		code, result := build.RunProgramIO(entry, nil, strings.NewReader(""), &stdout, &stderr)
		if result.HasErrors() {
			reportDiagnostics(errorOutput, result)
		} else {
			current := stdout.String()
			if !strings.HasPrefix(current, committedOutput) {
				fmt.Fprintln(errorOutput, "error [REPL002]\nreplayed session output was not deterministic")
			} else {
				fmt.Fprint(output, current[len(committedOutput):])
				if code == 0 {
					committedSource = candidate
					committedOutput = current
				}
			}
			if stderr.Len() != 0 {
				fmt.Fprint(errorOutput, stderr.String())
			}
		}
		if readError == io.EOF {
			return 0
		}
	}
}

func appendSubmission(committed, next string) string {
	if committed != "" && !strings.HasSuffix(committed, "\n") {
		committed += "\n"
	}
	return committed + next
}

func reportDiagnostics(writer io.Writer, result build.Result) {
	files := result.Files
	for id, file := range files {
		if filepath.Base(file.Path) == "Session.ahd" {
			file.Path = "<repl>"
			files[id] = file
		}
	}
	for _, item := range result.Diagnostics {
		fmt.Fprintln(writer, diagnostics.Render(item, files))
	}
}

func reportSyntax(writer io.Writer, text string) {
	file := source.NewFile(1, "<repl>", text)
	lexed := lexer.Lex(file)
	parsed := parser.Parse(file, lexed.Tokens)
	for _, item := range append(lexed.Diagnostics, parsed.Diagnostics...) {
		fmt.Fprintln(writer, diagnostics.Render(item, map[source.FileID]source.File{1: file}))
	}
}

// Incomplete distinguishes input that needs a continuation prompt from input
// that is complete (possibly invalid). It uses lexer tokens and parser EOF
// diagnostics rather than a second grammar.
func Incomplete(text string) bool {
	incomplete, _ := incompleteState(text)
	return incomplete
}

func incompleteState(text string) (bool, bool) {
	file := source.NewFile(1, "<repl>", text)
	lexed := lexer.Lex(file)
	for _, item := range lexed.Diagnostics {
		switch item.Code {
		case "LEX003", "LEX006", "LEX009":
			return true, false
		}
	}
	stack := make([]token.Kind, 0, 4)
	last := token.Invalid
	for _, item := range lexed.Tokens {
		if item.Kind == token.Newline || item.Kind == token.EOF || item.Synthetic {
			continue
		}
		last = item.Kind
		switch item.Kind {
		case token.LeftParen, token.LeftBracket, token.LeftBrace, token.InterpolationStart:
			stack = append(stack, item.Kind)
		case token.RightParen, token.RightBracket, token.RightBrace, token.InterpolationEnd:
			if len(stack) == 0 || !matching(stack[len(stack)-1], item.Kind) {
				return false, false
			}
			stack = stack[:len(stack)-1]
		}
	}
	if len(stack) != 0 || continuesExpression(last) {
		return true, false
	}
	parsed := parser.Parse(file, lexed.Tokens)
	for _, item := range parsed.Diagnostics {
		if item.Code == "PAR002" && item.Span.Start.Offset >= len(text) {
			return true, false
		}
	}
	if parsed.Program != nil && len(parsed.Program.Statements) != 0 && !strings.HasSuffix(text, "\n\n") {
		switch last := parsed.Program.Statements[len(parsed.Program.Statements)-1].(type) {
		case *ast.IfStmt:
			if len(parsed.Diagnostics) == 0 && last.Else == nil {
				return true, true
			}
		case *ast.AttemptStmt:
			if len(last.Excepts) == 0 && last.Ultimately == nil && onlyAttemptClauseDiagnostic(parsed.Diagnostics) {
				return true, false
			}
			if len(parsed.Diagnostics) == 0 && last.Ultimately == nil {
				return true, true
			}
		}
	}
	return false, false
}

func onlyAttemptClauseDiagnostic(items []diagnostics.Diagnostic) bool {
	if len(items) == 0 {
		return true
	}
	for _, item := range items {
		if item.Code != "PAR009" || !strings.Contains(item.Message, "attempt requires") {
			return false
		}
	}
	return true
}

func matching(opening, closing token.Kind) bool {
	return opening == token.LeftParen && closing == token.RightParen ||
		opening == token.LeftBracket && closing == token.RightBracket ||
		opening == token.LeftBrace && closing == token.RightBrace ||
		opening == token.InterpolationStart && closing == token.InterpolationEnd
}

func continuesExpression(kind token.Kind) bool {
	switch kind {
	case token.Colon, token.Declare, token.Assign, token.Arrow, token.Comma, token.Dot,
		token.Plus, token.Minus, token.Star, token.Slash, token.Percent, token.Caret,
		token.PlusAssign, token.MinusAssign, token.StarAssign, token.SlashAssign,
		token.PercentAssign, token.CaretAssign, token.Equal, token.NotEqual,
		token.Less, token.LessEqual, token.Greater, token.GreaterEqual,
		token.KeywordAnd, token.KeywordOr, token.KeywordNot, token.KeywordSame,
		token.KeywordIs, token.KeywordIn, token.KeywordHas:
		return true
	default:
		return false
	}
}
