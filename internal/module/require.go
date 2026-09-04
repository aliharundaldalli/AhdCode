package module

import (
	"fmt"
	"path/filepath"
	"strings"

	"ahdcode/internal/lexer"
	"ahdcode/internal/parser"
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
)

// requireState carries the state of one module's require(...) resolution:
// which canonical paths are already merged (dedup), which are currently
// being resolved (cycle detection), and the accumulated file/diagnostic
// results.
type requireState struct {
	appRoot    string
	resolved   map[string]bool
	stack      []string
	stackSet   map[string]bool
	order      []string
	files      map[string]source.File
	unresolved []string
}

// resolveRequires expands entryModule's own require(...) statements into one
// merged, deterministic statement list: each RequireStmt is replaced in
// place by the required file's own top-level statements, which may
// themselves contain further require(...) statements, resolved the same
// way. Every path is resolved relative to appRoot -- the entry module's own
// directory -- never relative to the requiring file, per the frozen
// app-root-relative rule (a nested require always means "from the app
// root", so the same literal always names the same file everywhere it
// appears). Identity is canonical: symlinks are resolved and "." segments
// removed before two paths are compared, so two spellings of one file merge
// as a single dependency.
func (compiler *Compiler) resolveRequires(identity SourceIdentity, entryModule *Module) {
	if entryModule.Parsed.Program == nil {
		return
	}
	// A bundled first-party module has no directory to be the app root, and
	// must never read the application's source tree. Framework sources
	// compose through `bring` alone, so there is nothing to expand here.
	if identity.Framework {
		return
	}
	entryCanonical := identity.Path
	state := &requireState{
		appRoot:  filepath.Dir(identity.Path),
		resolved: make(map[string]bool),
		stack:    []string{entryCanonical},
		stackSet: map[string]bool{entryCanonical: true},
		files:    make(map[string]source.File),
	}
	entryModule.Parsed.Program.Statements = compiler.expandRequires(entryModule.Parsed.Program.Statements, identity, state)
	for _, path := range state.order {
		entryModule.RequiredFiles = append(entryModule.RequiredFiles, state.files[path])
	}
	entryModule.UnresolvedRequires = state.unresolved
}

func (compiler *Compiler) expandRequires(statements []ast.Stmt, identity SourceIdentity, state *requireState) []ast.Stmt {
	result := make([]ast.Stmt, 0, len(statements))
	for _, statement := range statements {
		require, ok := statement.(*ast.RequireStmt)
		if !ok {
			result = append(result, statement)
			continue
		}
		if !require.HasLiteralPath {
			// The parser already reported a precise PAR014 diagnostic for
			// the malformed argument; dropping the statement keeps semantic
			// analysis from ever seeing an unresolved require.
			continue
		}
		resolvedPath, valid := compiler.resolveRequirePath(require, identity, state)
		if !valid {
			continue
		}
		// The cycle check must run before the dedup check: a file that
		// requires an ancestor still currently being resolved is both
		// "already marked resolved" (resolved is set as soon as its own
		// text is read, before its own requires are followed) and "on the
		// stack". Checking dedup first would silently treat that as an
		// ordinary already-merged dependency instead of the cycle it is.
		if state.stackSet[resolvedPath] {
			compiler.addDiagnostic(semantic.CodeRequireCycle,
				"require cycle detected:\n  "+strings.Join(requireCycleChain(state.appRoot, state.stack, resolvedPath), "\n  -> "),
				require.PathSpan, "remove one require(...) edge from the cycle", identity.ID, "", "", nil)
			continue
		}
		if state.resolved[resolvedPath] {
			// Compiled once: a second require edge to an already-merged
			// file, from this file or any other, contributes nothing more.
			continue
		}
		text, err := compiler.Loader.Load(SourceIdentity{ID: ModuleID("file:" + resolvedPath), Name: filepath.Base(resolvedPath), Path: resolvedPath})
		if err != nil {
			compiler.addDiagnostic(semantic.CodeRequireNotFound,
				fmt.Sprintf("required file not found: %q (resolved to %s)", require.Path, requireDisplayPath(state.appRoot, resolvedPath)),
				require.PathSpan,
				fmt.Sprintf("create %s, or fix the require(...) path", requireDisplayPath(state.appRoot, resolvedPath)),
				identity.ID, "", "", nil)
			state.unresolved = appendUniqueString(state.unresolved, resolvedPath)
			continue
		}
		fileID := compiler.nextFileID
		compiler.nextFileID++
		file := source.NewFile(fileID, resolvedPath, text)
		lexed := lexer.Lex(file)
		parsed := parser.Parse(file, lexed.Tokens)
		compiler.appendFrontendDiagnostics(identity.ID, lexed.Diagnostics)
		compiler.appendFrontendDiagnostics(identity.ID, parsed.Diagnostics)

		state.resolved[resolvedPath] = true
		state.files[resolvedPath] = file
		state.order = append(state.order, resolvedPath)

		var expanded []ast.Stmt
		if parsed.Program != nil {
			state.stack = append(state.stack, resolvedPath)
			state.stackSet[resolvedPath] = true
			expanded = compiler.expandRequires(parsed.Program.Statements, SourceIdentity{ID: identity.ID, Name: identity.Name, Path: resolvedPath}, state)
			state.stack = state.stack[:len(state.stack)-1]
			delete(state.stackSet, resolvedPath)
		}
		result = append(result, expanded...)
	}
	return result
}

// resolveRequirePath validates and canonicalizes one require(...) literal
// against the application root: it must be a relative, non-empty, .ahd path
// that stays inside the root after both plain "../" traversal and symlink
// resolution are accounted for. It returns the canonical absolute path even
// when the target does not exist yet, so a missing-file diagnostic can name
// the expected location and dev-mode watching can later notice the file
// appearing there.
func (compiler *Compiler) resolveRequirePath(require *ast.RequireStmt, identity SourceIdentity, state *requireState) (string, bool) {
	literal := require.Path
	invalid := func(message string) (string, bool) {
		compiler.addDiagnostic(semantic.CodeRequireInvalidPath, message, require.PathSpan,
			"require paths are relative to the application root and must stay inside it, e.g. require(\"Pages/Home.ahd\")",
			identity.ID, "", "", nil)
		return "", false
	}
	if literal == "" {
		return invalid("require path must not be empty")
	}
	if !strings.HasSuffix(literal, ".ahd") {
		return invalid(fmt.Sprintf("require only loads .ahd source files, got %q", literal))
	}
	if filepath.IsAbs(literal) || strings.HasPrefix(literal, "/") || strings.HasPrefix(literal, "\\") ||
		(len(literal) >= 2 && literal[1] == ':') {
		return invalid(fmt.Sprintf("require path must be relative to the application root, not absolute: %q", literal))
	}

	root, err := filepath.Abs(state.appRoot)
	if err != nil {
		return invalid(fmt.Sprintf("application root could not be resolved: %v", err))
	}
	candidate, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(literal)))
	if err != nil || !isWithinRoot(root, candidate) {
		return invalid(fmt.Sprintf("require path escapes the application root: %q", literal))
	}

	canonicalRoot := root
	if evaluated, evalErr := filepath.EvalSymlinks(root); evalErr == nil {
		canonicalRoot = evaluated
	}
	canonicalCandidate := candidate
	if evaluated, evalErr := filepath.EvalSymlinks(candidate); evalErr == nil {
		// The target exists: resolve through any symlink so identity and
		// containment are both judged on the real file, never the link.
		canonicalCandidate = evaluated
	}
	if !isWithinRoot(canonicalRoot, canonicalCandidate) {
		return invalid(fmt.Sprintf("require path escapes the application root through a symlink: %q", literal))
	}
	return canonicalCandidate, true
}

// isWithinRoot reports whether candidate is root itself or a descendant of
// it, using a purely lexical comparison of already-cleaned/absolute paths.
func isWithinRoot(root, candidate string) bool {
	root = filepath.Clean(root)
	candidate = filepath.Clean(candidate)
	if candidate == root {
		return true
	}
	relative, err := filepath.Rel(root, candidate)
	if err != nil || filepath.IsAbs(relative) {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func requireDisplayPath(appRoot, absolute string) string {
	if relative, err := filepath.Rel(appRoot, absolute); err == nil && !strings.HasPrefix(relative, "..") {
		return filepath.ToSlash(relative)
	}
	return absolute
}

func requireCycleChain(appRoot string, stack []string, closing string) []string {
	displayed := make([]string, 0, len(stack)+1)
	for _, path := range stack {
		displayed = append(displayed, requireDisplayPath(appRoot, path))
	}
	displayed = append(displayed, requireDisplayPath(appRoot, closing))
	return displayed
}

func appendUniqueString(list []string, value string) []string {
	for _, existing := range list {
		if existing == value {
			return list
		}
	}
	return append(list, value)
}
