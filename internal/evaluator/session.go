package evaluator

import (
	"bufio"
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"strings"

	"ahdcode/internal/ir"
)

// Session owns every value and initialization marker that must survive one
// REPL submission. Definitions are refreshed from each newly validated IR
// compilation, while storage and reference objects are never reconstructed.
type Session struct {
	Input  *bufio.Reader
	Output io.Writer
	CWD    string

	globals     map[ir.SymbolID]*Cell
	functions   map[ir.CallableID]*ir.Function
	classes     map[ir.ClassID]*ir.Class
	modules     map[ir.ModuleID]*ir.Module
	initialized map[ir.ModuleID]bool
	dispatch    map[ir.ClassID]map[ir.CallableID]ir.CallableID
	rngState    uint64
}

type ExecutionResult struct {
	Value    any
	HasValue bool
	Failure  *RuntimeError
}

func New(input *bufio.Reader, output io.Writer, cwd string) *Session {
	if input == nil {
		input = bufio.NewReader(strings.NewReader(""))
	}
	if output == nil {
		output = io.Discard
	}
	var seed [8]byte
	if _, err := io.ReadFull(cryptorand.Reader, seed[:]); err != nil {
		// This mirrors the native runtime's fatal entropy contract without
		// terminating the host REPL process from a library constructor.
		panic(fmt.Sprintf("AhdCode evaluator: Math RNG initialization failed: %v", err))
	}
	return &Session{
		Input: input, Output: output, CWD: cwd,
		globals: make(map[ir.SymbolID]*Cell), functions: make(map[ir.CallableID]*ir.Function),
		classes: make(map[ir.ClassID]*ir.Class), modules: make(map[ir.ModuleID]*ir.Module),
		initialized: make(map[ir.ModuleID]bool), dispatch: make(map[ir.ClassID]map[ir.CallableID]ir.CallableID),
		rngState: binary.LittleEndian.Uint64(seed[:]),
	}
}

// Execute installs one validated compilation and executes only module work
// that has not already happened. For the entry module, statements beginning
// before entryOffset belong to earlier REPL submissions and are not replayed.
func (session *Session) Execute(compilation *ir.Compilation, entryOffset int) (result ExecutionResult) {
	defer func() {
		if recovered := recover(); recovered != nil {
			switch value := recovered.(type) {
			case raised:
				result = ExecutionResult{Failure: value.failure}
			case *raised:
				result = ExecutionResult{Failure: value.failure}
			default:
				panic(recovered)
			}
		}
	}()
	if compilation == nil {
		return ExecutionResult{Failure: &RuntimeError{Name: "Error", Message: "nil IR compilation"}}
	}
	session.install(compilation)
	entry := session.modules[compilation.Entry]
	if entry == nil {
		return ExecutionResult{Failure: &RuntimeError{Name: "Error", Message: "entry IR module is missing"}}
	}
	for _, dependency := range entry.Dependencies {
		session.initializeModule(dependency)
	}
	frame := newFrame(nil)
	last, hasValue, flow := session.execBlockFiltered(entry.Init, frame, entryOffset)
	if flow.kind != flowNormal {
		session.raise("Error", "invalid control transfer at module root")
	}
	return ExecutionResult{Value: last, HasValue: hasValue}
}

func (session *Session) install(compilation *ir.Compilation) {
	for _, module := range compilation.Modules {
		if module == nil {
			continue
		}
		session.modules[module.ID] = module
		for _, function := range module.Functions {
			if function != nil {
				session.functions[function.ID] = function
			}
		}
		for _, class := range module.Classes {
			if class != nil {
				session.classes[class.ID] = class
			}
		}
	}
	session.buildDispatch()
}

func (session *Session) initializeModule(identity ir.ModuleID) {
	if session.initialized[identity] {
		return
	}
	module := session.modules[identity]
	if module == nil {
		session.raise("Error", "missing imported module "+string(identity))
	}
	for _, dependency := range module.Dependencies {
		session.initializeModule(dependency)
	}
	_, _, outcome := session.execBlock(module.Init, newFrame(nil))
	if outcome.kind != flowNormal {
		session.raise("Error", "invalid control transfer in module initializer")
	}
	session.initialized[identity] = true
}

func (session *Session) buildDispatch() {
	session.dispatch = make(map[ir.ClassID]map[ir.CallableID]ir.CallableID)
	ids := make([]ir.ClassID, 0, len(session.classes))
	for id := range session.classes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var build func(ir.ClassID) map[ir.CallableID]ir.CallableID
	building := make(map[ir.ClassID]bool)
	build = func(id ir.ClassID) map[ir.CallableID]ir.CallableID {
		if known := session.dispatch[id]; known != nil {
			return known
		}
		if building[id] {
			return map[ir.CallableID]ir.CallableID{}
		}
		building[id] = true
		result := make(map[ir.CallableID]ir.CallableID)
		class := session.classes[id]
		if class != nil && class.Parent != "" {
			for slot, implementation := range build(class.Parent) {
				result[slot] = implementation
			}
		}
		if class != nil {
			for _, methodID := range class.Methods {
				method := session.functions[methodID]
				result[methodID] = methodID
				if method != nil && method.Overrides != "" {
					result[method.Overrides] = methodID
					for slot, implementation := range result {
						if implementation == method.Overrides {
							result[slot] = methodID
						}
					}
				}
			}
		}
		delete(building, id)
		session.dispatch[id] = result
		return result
	}
	for _, id := range ids {
		build(id)
	}
}

func (session *Session) errorClass(name string) ir.ClassID {
	preferred := ir.ClassID("builtin:core::class::" + name)
	if session.classes[preferred] != nil {
		return preferred
	}
	for id := range session.classes {
		if className(id) == name {
			return id
		}
	}
	return preferred
}

func (session *Session) fieldNamed(classID ir.ClassID, name string) ir.FieldID {
	for current := classID; current != ""; {
		class := session.classes[current]
		if class == nil {
			break
		}
		for _, field := range class.Fields {
			if field.Name == name {
				return field.ID
			}
		}
		current = class.Parent
	}
	return ""
}

func (session *Session) isClass(value, target ir.ClassID) bool {
	for current := value; current != ""; {
		if current == target {
			return true
		}
		class := session.classes[current]
		if class == nil {
			return false
		}
		current = class.Parent
	}
	return false
}
