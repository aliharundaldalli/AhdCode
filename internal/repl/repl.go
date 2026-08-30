// Package repl implements a persistent interactive AhdCode session over the
// same validated and lowered IR consumed by the native backend.
package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"ahdcode/internal/build"
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/evaluator"
	"ahdcode/internal/lexer"
	"ahdcode/internal/lowering"
	"ahdcode/internal/module"
	"ahdcode/internal/parser"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/syntax/token"
)

const (
	PrimaryPrompt      = "ahd> "
	ContinuationPrompt = "...> "
)

// Run starts a session and returns when input reaches EOF. Frontend validation
// sees aggregate source for static context, while only new IR spans execute.
func Run(input io.Reader, output, errorOutput io.Writer, version string) int {
	directory, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(errorOutput, "error [REPL001]\ncould not determine REPL working directory: %v\n", err)
		return 1
	}
	entry := filepath.Join(directory, ".ahdcode-repl-session.ahd")
	reader := bufio.NewReader(input)
	session := evaluator.New(reader, output, directory)
	committedSource := ""
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

		prefix := committedSource
		if prefix != "" && !strings.HasSuffix(prefix, "\n") {
			prefix += "\n"
		}
		entryOffset := len(prefix)
		candidate := prefix + pending
		pending = ""
		result := compileSession(entry, candidate)
		if result.HasErrors() {
			reportDiagnostics(errorOutput, result)
		} else {
			execution := session.Execute(result.IR, entryOffset)
			if execution.Failure != nil {
				fmt.Fprintf(errorOutput, "error [RUN001]\n%s\n", execution.Failure)
			} else {
				committedSource = candidate
				if execution.HasValue {
					fmt.Fprintln(output, session.Render(execution.Value))
				}
			}
		}
		if readError == io.EOF {
			return 0
		}
	}
}

// overlayWorkspace keeps the synthetic entry in memory while ordinary local
// modules resolve relative to the real directory where the REPL was launched.
type overlayWorkspace struct {
	entryPath string
	text      string
	resolver  module.FileResolver
	loader    module.FileLoader
}

func (workspace overlayWorkspace) CanonicalEntry(entryPath string) (module.SourceIdentity, error) {
	return workspace.resolver.CanonicalEntry(entryPath)
}

func (workspace overlayWorkspace) Resolve(importer module.SourceIdentity, moduleName string) (module.SourceIdentity, error) {
	return workspace.resolver.Resolve(importer, moduleName)
}

func (workspace overlayWorkspace) Load(identity module.SourceIdentity) (string, error) {
	if filepath.Clean(identity.Path) == filepath.Clean(workspace.entryPath) {
		return workspace.text, nil
	}
	return workspace.loader.Load(identity)
}

func compileSession(entry, text string) build.Result {
	workspace := overlayWorkspace{entryPath: entry, text: text}
	frontend := module.NewCompiler(workspace, workspace).Compile(entry)
	result := build.Result{Compilation: &frontend, Files: make(map[source.FileID]source.File)}
	for _, current := range frontend.Modules {
		if current != nil && current.File.ID != 0 {
			result.Files[current.File.ID] = current.File
		}
	}
	for _, item := range frontend.Diagnostics {
		result.Diagnostics = append(result.Diagnostics, item.Diagnostic)
	}
	if result.HasErrors() {
		return result
	}
	lowered := lowering.LowerCompilation(frontend)
	result.Diagnostics = append(result.Diagnostics, lowered.Diagnostics...)
	if !result.HasErrors() {
		result.IR = lowered.Compilation
	}
	return result
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
		if filepath.Base(file.Path) == ".ahdcode-repl-session.ahd" {
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
