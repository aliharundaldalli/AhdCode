// Package ahdruntime is the AhdCode v0.1 Go backend runtime.
//
// This file is compiled twice: once as part of the compiler (so ordinary Go
// tooling checks it) and once as generated program source, where the package
// clause is rewritten to main. It must therefore depend only on the Go
// standard library and must not reference any other AhdCode package.
package ahdruntime

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// Class identity
// ---------------------------------------------------------------------------

// AhdClass is the canonical runtime identity of one AhdCode Class. Descriptors
// are compared by pointer, never by name.
//
// Members lists only the member names this Class itself declares. Inherited
// names are reached through Parent rather than copied, so one descriptor never
// restates its ancestors.
type AhdClass struct {
	Name    string
	Parent  *AhdClass
	Members []string
}

// The language-supplied Class catalog. Generated code aliases these
// descriptors so a runtime check and an AhdCode except clause agree on
// identity.
var (
	AhdClassObject              = &AhdClass{Name: "Object"}
	AhdClassError               = &AhdClass{Name: "Error", Parent: AhdClassObject, Members: []string{"message"}}
	AhdClassConstantError       = &AhdClass{Name: "ConstantError", Parent: AhdClassError}
	AhdClassDivisionByZeroError = &AhdClass{Name: "DivisionByZeroError", Parent: AhdClassError}
	AhdClassDomainError         = &AhdClass{Name: "DomainError", Parent: AhdClassError}
	AhdClassIndexError          = &AhdClass{Name: "IndexError", Parent: AhdClassError}
	AhdClassIOError             = &AhdClass{Name: "IOError", Parent: AhdClassError}
	AhdClassKeyError            = &AhdClass{Name: "KeyError", Parent: AhdClassError}
	AhdClassNullError           = &AhdClass{Name: "NullError", Parent: AhdClassError}
	AhdClassOverflowError       = &AhdClass{Name: "OverflowError", Parent: AhdClassError}
	AhdClassValueError          = &AhdClass{Name: "ValueError", Parent: AhdClassError}
	AhdClassLatexError          = &AhdClass{Name: "LatexError", Parent: AhdClassError}
	AhdClassFileError           = &AhdClass{Name: "FileError", Parent: AhdClassIOError}
	AhdClassRegexError          = &AhdClass{Name: "RegexError", Parent: AhdClassError}
	AhdClassCSVError            = &AhdClass{Name: "CSVError", Parent: AhdClassError}
	AhdClassDataError           = &AhdClass{Name: "DataError", Parent: AhdClassError}
	AhdClassStatisticsError     = &AhdClass{Name: "StatisticsError", Parent: AhdClassError}
	AhdClassPlotError           = &AhdClass{Name: "PlotError", Parent: AhdClassError}
	AhdClassNumericError        = &AhdClass{Name: "NumericError", Parent: AhdClassError}
	AhdClassWordError           = &AhdClass{Name: "WordError", Parent: AhdClassError}
	AhdClassJSONError           = &AhdClass{Name: "JSONError", Parent: AhdClassError}
	AhdClassXMLError            = &AhdClass{Name: "XMLError", Parent: AhdClassError}
	AhdClassEnvError            = &AhdClass{Name: "EnvError", Parent: AhdClassError}
	AhdClassListsError          = &AhdClass{Name: "ListsError", Parent: AhdClassError}
)

// AhdInstance is every AhdCode Class instance. The generated interface of each
// Class embeds it, so identity, freezing, and Class membership work uniformly.
type AhdInstance interface {
	AhdClassOf() *AhdClass
	AhdFreezeGraph(visited map[AhdFreezable]bool)
	AhdIdentity() int64
}

// AhdBase is embedded in the root of every generated Class struct.
type AhdBase struct {
	ahdClass  *AhdClass
	ahdFrozen bool
	ahdID     int64
}

// AhdClassOf returns the exact runtime Class of an instance.
func (base *AhdBase) AhdClassOf() *AhdClass { return base.ahdClass }

// AhdSetClass stamps the exact runtime Class during construction.
func (base *AhdBase) AhdSetClass(class *AhdClass) { base.ahdClass = class }

// AhdFrozen reports whether the instance is part of a deep-frozen graph.
func (base *AhdBase) AhdFrozen() bool { return base.ahdFrozen }

// AhdMarkFrozen freezes the instance itself and reports whether this call was
// the transition, so a graph walk visits each object once.
func (base *AhdBase) AhdMarkFrozen() bool {
	if base.ahdFrozen {
		return false
	}
	base.ahdFrozen = true
	return true
}

// AhdRequireMutable rejects mutation of a deep-frozen object.
func (base *AhdBase) AhdRequireMutable() {
	if base.ahdFrozen {
		AhdRaiseClass(AhdClassConstantError, "cannot mutate a Constant object")
	}
}

// AhdIsClass reports Class membership, including inheritance.
func AhdIsClass(value AhdInstance, target *AhdClass) bool {
	if value == nil || target == nil {
		return false
	}
	for current := value.AhdClassOf(); current != nil; current = current.Parent {
		if current == target {
			return true
		}
	}
	return false
}

// AhdHasMember implements has / has not. It reads the value's exact runtime
// Class rather than the static type of the expression, then walks the Parent
// chain, so an instance upcast to a parent type still reports the members its
// real Class declares. It is a lookup over published member names, not
// reflection over the Go object.
func AhdHasMember(value AhdInstance, name string) bool {
	if value == nil {
		return false
	}
	for current := value.AhdClassOf(); current != nil; current = current.Parent {
		for _, member := range current.Members {
			if member == name {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Runtime identity (id())
// ---------------------------------------------------------------------------

var (
	ahdIdentityMu      sync.Mutex
	ahdIdentityCounter int64
)

// ahdNextIdentity allocates the next opaque, process-local identity number.
// It is a plain incrementing counter; AhdCode programs must not depend on
// allocation order, only on equality/inequality between two identities.
func ahdNextIdentity() int64 {
	ahdIdentityMu.Lock()
	defer ahdIdentityMu.Unlock()
	if ahdIdentityCounter == math.MaxInt64 {
		AhdRaiseClass(AhdClassOverflowError, "runtime identity allocator overflowed signed 64-bit range")
	}
	ahdIdentityCounter++
	return ahdIdentityCounter
}

// AhdIdentity lazily assigns and then returns this instance's stable identity
// number. It never changes once assigned, regardless of later mutation.
func (base *AhdBase) AhdIdentity() int64 {
	if base.ahdID == 0 {
		base.ahdID = ahdNextIdentity()
	}
	return base.ahdID
}

// AhdIdentifiable is any AhdCode reference value with a meaningful runtime
// identity: a Class instance, a List, or a Pair.
type AhdIdentifiable interface{ AhdIdentity() int64 }

// AhdId implements the id() Fundamental.
func AhdId(value AhdIdentifiable) int64 { return value.AhdIdentity() }

// AhdEqInstance is Class reference identity.
func AhdEqInstance(left, right AhdInstance) bool { return left == right }

// AhdSameInstance is strict same: exact runtime Class and the same object.
func AhdSameInstance(left, right AhdInstance) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.AhdClassOf() == right.AhdClassOf() && left == right
}

// AhdStrInstance renders the canonical Class instance text.
func AhdStrInstance(value AhdInstance) string {
	if value == nil {
		return "null"
	}
	class := value.AhdClassOf()
	if class == nil {
		return "<Object>"
	}
	return "<" + class.Name + ">"
}

// AhdRequireInstance reads a nullable Class reference the frontend proved
// non-null.
func AhdRequireInstance[T any](value T, present bool) T {
	if !present {
		AhdRaiseClass(AhdClassNullError, "value is null")
	}
	return value
}

// ---------------------------------------------------------------------------
// Runtime errors
// ---------------------------------------------------------------------------

// AhdSignal is the isolated control value used to raise an AhdCode Error. An
// ordinary Go panic is never an AhdSignal, so a compiler or runtime defect can
// never be caught by an AhdCode except clause.
type AhdSignal struct {
	Instance AhdInstance
	Message  string
}

func (signal *AhdSignal) Error() string {
	return ahdSignalClassName(signal) + ": " + signal.Message
}

func ahdSignalClassName(signal *AhdSignal) string {
	if signal == nil || signal.Instance == nil || signal.Instance.AhdClassOf() == nil {
		return "Error"
	}
	return signal.Instance.AhdClassOf().Name
}

// ahdErrorConstructors lets a runtime check raise a real AhdCode Error
// instance. Generated code installs the built-in Error constructors before the
// program body runs; lookups are by descriptor pointer, never by iteration.
var ahdErrorConstructors = map[*AhdClass]func(string) AhdInstance{}

// AhdRegisterError installs the generated constructor of one built-in Error.
func AhdRegisterError(class *AhdClass, construct func(string) AhdInstance) {
	ahdErrorConstructors[class] = construct
}

// AhdRegisterErrorFallback installs a derived built-in Error using an
// available parent representation when its owning standard module is not
// otherwise present in the compilation. A later exact registration wins.
func AhdRegisterErrorFallback(class *AhdClass, construct func(string) AhdInstance) {
	if ahdErrorConstructors[class] != nil {
		return
	}
	ahdErrorConstructors[class] = func(message string) AhdInstance {
		instance := construct(message)
		setter, ok := instance.(interface{ AhdSetClass(*AhdClass) })
		if !ok {
			panic("ahdcode: fallback Error instance cannot be assigned a Class")
		}
		setter.AhdSetClass(class)
		return instance
	}
}

// AhdRaiseClass raises a built-in AhdCode runtime Error.
func AhdRaiseClass(class *AhdClass, message string) {
	construct := ahdErrorConstructors[class]
	if construct == nil {
		panic("ahdcode: built-in Error class " + class.Name + " has no registered constructor")
	}
	panic(&AhdSignal{Instance: construct(message), Message: message})
}

// AhdErrorInstance is an AhdCode Error instance, which always carries a
// message attribute inherited from the built-in Error Class.
type AhdErrorInstance interface {
	AhdInstance
	AhdErrorMessage() string
}

// AhdToss raises an AhdCode Error instance written by user code.
func AhdToss(instance AhdInstance) {
	if instance == nil {
		AhdRaiseClass(AhdClassNullError, "cannot toss a null Error")
	}
	message := ""
	if carrier, ok := instance.(AhdErrorInstance); ok {
		message = carrier.AhdErrorMessage()
	}
	panic(&AhdSignal{Instance: instance, Message: message})
}

// Control outcomes of an attempt body. They transfer a pending return, break,
// or continue across the error-handling boundary without skipping ultimately.
const (
	AhdOutcomeNormal   = 0
	AhdOutcomeReturn   = 1
	AhdOutcomeBreak    = 2
	AhdOutcomeContinue = 3
)

// AhdSignalOf converts a recovered value into an AhdCode error signal. An
// ordinary Go panic is re-raised unchanged rather than being handled as an
// AhdCode Error.
func AhdSignalOf(recovered any) *AhdSignal {
	if recovered == nil {
		return nil
	}
	signal, ok := recovered.(*AhdSignal)
	if !ok {
		panic(recovered)
	}
	return signal
}

// AhdMatches reports whether an error signal is handled by one except clause.
func AhdMatches(signal *AhdSignal, class *AhdClass) bool {
	return signal != nil && AhdIsClass(signal.Instance, class)
}

var ahdOut = bufio.NewWriter(os.Stdout)
var ahdIn = bufio.NewReader(os.Stdin)

// AhdFlush writes buffered program output.
func AhdFlush() {
	_ = ahdOut.Flush()
}

// AhdMain runs a generated program body and turns an uncaught AhdCode error
// into a diagnostic exit instead of a Go panic trace.
func AhdMain(install func(), body func()) {
	install()
	failed := false
	func() {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			signal, ok := recovered.(*AhdSignal)
			if !ok {
				AhdFlush()
				panic(recovered)
			}
			AhdFlush()
			fmt.Fprintln(os.Stderr, ahdSignalClassName(signal)+": "+signal.Message)
			failed = true
		}()
		body()
	}()
	AhdFlush()
	if failed {
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// Latex standard module
// ---------------------------------------------------------------------------

const ahdLatexLogLimit = 64 << 10

// Variable only so the runtime package can exercise timeout handling without
// making its unit test sleep for thirty seconds. Generated programs leave the
// v0.1.5 contract value unchanged.
var ahdLatexCompileTimeout = 30 * time.Second

// AhdLatexRuntimeHint is filled by the compiler when it builds a program that
// uses Latex. The environment override is reserved for packaging and tests;
// neither mechanism is AhdCode language syntax.
var AhdLatexRuntimeHint string

// AhdLatexEscape escapes ordinary text for LaTeX text context. It deliberately
// does not claim to sanitize raw mathematics.
func AhdLatexEscape(text string) string {
	var result strings.Builder
	for _, character := range text {
		switch character {
		case '\\':
			result.WriteString(`\textbackslash{}`)
		case '{':
			result.WriteString(`\{`)
		case '}':
			result.WriteString(`\}`)
		case '$':
			result.WriteString(`\$`)
		case '&':
			result.WriteString(`\&`)
		case '#':
			result.WriteString(`\#`)
		case '%':
			result.WriteString(`\%`)
		case '_':
			result.WriteString(`\_`)
		case '^':
			result.WriteString(`\textasciicircum{}`)
		case '~':
			result.WriteString(`\textasciitilde{}`)
		default:
			result.WriteRune(character)
		}
	}
	return result.String()
}

func AhdLatexSection(title string) string {
	return `\section{` + AhdLatexEscape(title) + "}\n"
}

func AhdLatexSubsection(title string) string {
	return `\subsection{` + AhdLatexEscape(title) + "}\n"
}

func AhdLatexChapter(title string) string { return `\chapter{` + AhdLatexEscape(title) + "}\n" }
func AhdLatexFrame(title, body string) string {
	return "\\begin{frame}{" + AhdLatexEscape(title) + "}\n" + body + func() string {
		if body != "" && !strings.HasSuffix(body, "\n") {
			return "\n"
		}
		return ""
	}() + "\\end{frame}\n"
}
func ahdLatexLabel(label string) string {
	if label == "" {
		return ""
	}
	return "\\label{" + AhdLatexEscape(label) + "}\n"
}
func AhdLatexEquation(source string, labels ...string) string {
	label := ""
	if len(labels) > 0 {
		label = labels[0]
	}
	return "\\begin{equation}\n" + source + "\n" + ahdLatexLabel(label) + "\\end{equation}\n"
}
func AhdLatexRef(label string) string { return "\\ref{" + AhdLatexEscape(label) + "}" }
func AhdLatexCite(key string) string  { return "\\cite{" + AhdLatexEscape(key) + "}" }
func AhdLatexCenter(body string) string {
	return "\\begin{center}\n" + body + func() string {
		if body != "" && !strings.HasSuffix(body, "\n") {
			return "\n"
		}
		return ""
	}() + "\\end{center}\n"
}
func AhdLatexPageBreak() string { return "\\clearpage\n" }
func AhdLatexContents() string  { return "\\tableofcontents\n" }
func AhdLatexMinipage(body string, width float64, alignment string) string {
	if width <= 0 {
		AhdRaiseClass(AhdClassValueError, "Latex.minipage width must be positive")
	}
	command := map[string]string{"left": "\\raggedright", "center": "\\centering", "right": "\\raggedleft"}[alignment]
	if command == "" {
		AhdRaiseClass(AhdClassValueError, "Latex.minipage alignment must be left, center, or right")
	}
	return "\\begin{minipage}{" + ahdFormatReal(width) + "cm}\n" + command + "\n" + body + func() string {
		if body != "" && !strings.HasSuffix(body, "\n") {
			return "\n"
		}
		return ""
	}() + "\\end{minipage}\n"
}

func ahdLatexTheoremID(name string) string {
	sum := sha256.Sum256([]byte(name))
	return fmt.Sprintf("ahdthm%x", sum[:6])
}
func AhdLatexTheorem(kind, body, label string) string {
	if kind == "" {
		AhdRaiseClass(AhdClassValueError, "Latex.theorem type must not be empty")
	}
	id := ahdLatexTheoremID(kind)
	return "\\begin{" + id + "}\n" + body + func() string {
		if body != "" && !strings.HasSuffix(body, "\n") {
			return "\n"
		}
		return ""
	}() + ahdLatexLabel(label) + "\\end{" + id + "}\n"
}

func ahdLatexSizes(size *AhdPair[string, float64]) string {
	if size == nil {
		return ""
	}
	size.require()
	options := []string{}
	known := map[string]bool{}
	for _, key := range size.keys {
		if key != "width" && key != "height" {
			AhdRaiseClass(AhdClassValueError, "Latex image size supports only width and height")
		}
		if known[key] {
			AhdRaiseClass(AhdClassValueError, "duplicate Latex image size option")
		}
		known[key] = true
		value := size.values[key]
		if value <= 0 {
			AhdRaiseClass(AhdClassValueError, "Latex image dimensions must be positive")
		}
		options = append(options, key+"="+ahdFormatReal(value)+"cm")
	}
	if len(options) == 0 {
		return ""
	}
	return "[" + strings.Join(options, ",") + "]"
}
func ahdLatexAsset(path string) (string, string) {
	if path == "" {
		AhdRaiseClass(AhdClassValueError, "Latex image path must not be empty")
	}
	extension := strings.ToLower(filepath.Ext(path))
	if extension != ".png" && extension != ".pdf" && extension != ".jpg" && extension != ".jpeg" {
		AhdRaiseClass(AhdClassValueError, "Latex image supports PNG, PDF, and JPEG assets")
	}
	sum := sha256.Sum256([]byte(path))
	staged := fmt.Sprintf("ahdasset-%x%s", sum[:8], extension)
	marker := "% AHDCODE_ASSET " + base64.RawStdEncoding.EncodeToString([]byte(path)) + " " + staged + "\n"
	return marker, staged
}
func AhdLatexImage(path string, size *AhdPair[string, float64]) string {
	marker, staged := ahdLatexAsset(path)
	return marker + "\\includegraphics" + ahdLatexSizes(size) + "{" + staged + "}\n"
}
func AhdLatexFigure(path, caption, label string, size *AhdPair[string, float64]) string {
	marker, staged := ahdLatexAsset(path)
	return marker + "\\begin{figure}[!ht]\n\\centering\n\\includegraphics" + ahdLatexSizes(size) + "{" + staged + "}\n\\caption{" + AhdLatexEscape(caption) + "}\n" + ahdLatexLabel(label) + "\\end{figure}\n"
}
func AhdLatexBibliography(references *AhdPair[string, string]) string {
	if references == nil {
		return ""
	}
	references.require()
	var out strings.Builder
	out.WriteString("\\begin{thebibliography}{99}\n")
	for _, key := range references.keys {
		out.WriteString("\\bibitem{" + AhdLatexEscape(key) + "} " + AhdLatexEscape(references.values[key]) + "\n")
	}
	out.WriteString("\\end{thebibliography}\n")
	return out.String()
}

// AhdLatexDocument returns one stable complete document. Font files are named
// explicitly so the supported baseline never depends on a system font.
func AhdLatexDocument(body, title, author string) string {
	return AhdLatexDocumentFull(body, title, author, "", "Article", 2.54, "", "", AhdBuildPair([]string{}, []string{}), "Default")
}

var ahdLatexBeamerThemes = map[string]bool{"Default": true, "Madrid": true, "Warsaw": true}

func AhdLatexDocumentFull(body, title, author, date, documentType string, margin float64, color, cover string, theorems *AhdPair[string, string], theme string) string {
	classes := map[string]string{"Article": "article", "Report": "report", "Beamer": "beamer"}
	documentClass := classes[documentType]
	if documentClass == "" {
		AhdRaiseClass(AhdClassValueError, "Latex.document type must be Article, Report, or Beamer")
	}
	if margin <= 0 {
		AhdRaiseClass(AhdClassValueError, "Latex.document margin must be positive")
	}
	if color != "" {
		matched, _ := regexp.MatchString(`^#[0-9A-Fa-f]{6}$`, color)
		if !matched {
			AhdRaiseClass(AhdClassValueError, "Latex.document color must use #RRGGBB")
		}
	}
	if !ahdLatexBeamerThemes[theme] {
		AhdRaiseClass(AhdClassValueError, "Latex.document theme must be Default, Madrid, or Warsaw")
	}
	if theme != "Default" && documentType != "Beamer" {
		AhdRaiseClass(AhdClassValueError, "Latex.document theme requires a Beamer document")
	}
	var result strings.Builder
	result.WriteString("\\documentclass{" + documentClass + "}\n")
	if theme != "Default" {
		// theme != "Default" already implies documentType == "Beamer", checked above.
		result.WriteString("\\usetheme{" + theme + "}\n")
	}
	result.WriteString("\\usepackage{fontspec}\n")
	result.WriteString("\\setmainfont{lmroman10-regular.otf}[BoldFont=lmroman10-bold.otf,ItalicFont=lmroman10-italic.otf,BoldItalicFont=lmroman10-bolditalic.otf]\n")
	result.WriteString("\\usepackage{amsmath,amssymb,mathtools}\n")
	result.WriteString("\\usepackage{geometry,graphicx,booktabs,array,xcolor,hyperref}\n")
	result.WriteString("\\geometry{margin=" + ahdFormatReal(margin) + "cm}\n")
	result.WriteString("\\hypersetup{hidelinks}\n")
	if color != "" {
		hex := strings.TrimPrefix(color, "#")
		result.WriteString("\\definecolor{ahdaccent}{HTML}{" + strings.ToUpper(hex) + "}\n")
		if documentType == "Beamer" {
			result.WriteString("\\setbeamercolor{structure}{fg=ahdaccent}\n")
		}
	}
	declared := map[string]string{}
	if theorems != nil {
		theorems.require()
		for _, display := range theorems.keys {
			if display == "" {
				AhdRaiseClass(AhdClassValueError, "theorem type name must not be empty")
			}
			id := ahdLatexTheoremID(display)
			rule := theorems.values[display]
			switch rule {
			case "":
				result.WriteString("\\newtheorem{" + id + "}{" + AhdLatexEscape(display) + "}\n")
			case "section", "subsection":
				result.WriteString("\\newtheorem{" + id + "}{" + AhdLatexEscape(display) + "}[" + rule + "]\n")
			case "chapter":
				if documentType != "Report" {
					AhdRaiseClass(AhdClassValueError, "chapter theorem counters require a Report document")
				}
				result.WriteString("\\newtheorem{" + id + "}{" + AhdLatexEscape(display) + "}[chapter]\n")
			default:
				shared := declared[rule]
				if shared == "" {
					AhdRaiseClass(AhdClassValueError, "theorem counter references an unknown or later type: "+rule)
				}
				result.WriteString("\\newtheorem{" + id + "}[" + shared + "]{" + AhdLatexEscape(display) + "}\n")
			}
			declared[display] = id
		}
	}
	knownTheorems := map[string]bool{}
	for _, id := range declared {
		knownTheorems[id] = true
	}
	theoremPattern := regexp.MustCompile(`\\begin\{(ahdthm[0-9a-f]+)\}`)
	for _, match := range theoremPattern.FindAllStringSubmatch(body, -1) {
		if !knownTheorems[match[1]] {
			AhdRaiseClass(AhdClassValueError, "document body uses an undeclared theorem type")
		}
	}
	if title != "" {
		result.WriteString("\\title{" + AhdLatexEscape(title) + "}\n")
	}
	if author != "" {
		result.WriteString("\\author{" + AhdLatexEscape(author) + "}\n")
	}
	result.WriteString("\\date{" + AhdLatexEscape(date) + "}\n\\begin{document}\n")
	if cover != "" {
		result.WriteString(cover)
		if !strings.HasSuffix(cover, "\n") {
			result.WriteByte('\n')
		}
		result.WriteString("\\clearpage\n")
	}
	if title != "" {
		if documentType == "Beamer" {
			result.WriteString("\\begin{frame}\n\\titlepage\n\\end{frame}\n")
		} else {
			result.WriteString("\\maketitle\n")
		}
	}
	result.WriteString(body)
	if body != "" && !strings.HasSuffix(body, "\n") {
		result.WriteByte('\n')
	}
	result.WriteString("\\end{document}\n")
	return result.String()
}

// AhdLatexTable creates deterministic booktabs source. List elements retain
// the ordinary nullable-element rule; a null row or cell raises NullError.
func AhdLatexTable(headers *AhdList[string], rows *AhdList[*AhdList[string]], mathColumns *AhdList[int64]) string {
	headerValues := headers.Snapshot()
	if len(headerValues) == 0 {
		AhdRaiseClass(AhdClassValueError, "Latex.table requires at least one header")
	}
	// A listed column is the caller's explicit opt-in to raw LaTeX math for
	// that column. Membership is a set, so a repeated index wraps a cell once.
	math := make(map[int64]bool)
	for _, index := range mathColumns.Snapshot() {
		if index < 0 || index >= int64(len(headerValues)) {
			AhdRaiseClass(AhdClassValueError, "Latex.table math column "+strconv.FormatInt(index, 10)+
				" is outside 0.."+strconv.FormatInt(int64(len(headerValues)-1), 10))
		}
		math[index] = true
	}
	rowValues := rows.Snapshot()
	var result strings.Builder
	result.WriteString("\\begin{tabular}{")
	result.WriteString(strings.Repeat("l", len(headerValues)))
	result.WriteString("}\n\\toprule\n")
	for index, value := range headerValues {
		if index != 0 {
			result.WriteString(" & ")
		}
		result.WriteString(AhdLatexEscape(value))
	}
	result.WriteString(" \\\\\n\\midrule\n")
	for _, row := range rowValues {
		nonNullRow := AhdNonNull(row)
		cells := nonNullRow.Snapshot()
		if len(cells) != len(headerValues) {
			AhdRaiseClass(AhdClassValueError, "Latex.table row column count does not match headers")
		}
		for index, value := range cells {
			if index != 0 {
				result.WriteString(" & ")
			}
			cell := value
			// A math column carries LaTeX source, so escaping it would destroy
			// the very commands it exists to typeset.
			if math[int64(index)] {
				result.WriteString("\\(" + cell + "\\)")
				continue
			}
			result.WriteString(AhdLatexEscape(cell))
		}
		result.WriteString(" \\\\\n")
	}
	result.WriteString("\\bottomrule\n\\end{tabular}\n")
	return result.String()
}

func AhdLatexPDF(source, output string) {
	if output == "" {
		AhdRaiseClass(AhdClassValueError, "Latex.pdf output path must not be empty")
	}
	directory, err := os.MkdirTemp("", "ahdcode-latex-source-*")
	if err != nil {
		ahdLatexRaise("could not create a secure temporary directory: " + err.Error())
	}
	defer os.RemoveAll(directory)
	working, err := os.Getwd()
	if err != nil {
		ahdLatexRaise("could not resolve the asset base directory: " + err.Error())
	}
	if err := ahdLatexStageAssets(source, working, directory); err != nil {
		ahdLatexRaise(err.Error())
	}
	input := filepath.Join(directory, "document.tex")
	if err := os.WriteFile(input, []byte(source), 0o600); err != nil {
		ahdLatexRaise("could not write temporary LaTeX source: " + err.Error())
	}
	ahdLatexCompile(input, directory, output)
}

func ahdLatexStageAssets(source, base, destination string) error {
	pattern := regexp.MustCompile(`(?m)^% AHDCODE_ASSET ([A-Za-z0-9_-]+) (ahdasset-[0-9a-f]+\.(?:png|pdf|jpg|jpeg))$`)
	seen := map[string]bool{}
	for _, match := range pattern.FindAllStringSubmatch(source, -1) {
		if seen[match[2]] {
			continue
		}
		seen[match[2]] = true
		decoded, err := base64.RawStdEncoding.DecodeString(match[1])
		if err != nil {
			return fmt.Errorf("invalid generated Latex asset marker")
		}
		path := string(decoded)
		if !filepath.IsAbs(path) {
			path = filepath.Join(base, path)
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("could not resolve Latex asset %s: %w", path, err)
		}
		info, err := os.Stat(absolute)
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("Latex asset is missing or not a regular file: %s", path)
		}
		input, err := os.Open(absolute)
		if err != nil {
			return fmt.Errorf("could not open Latex asset %s: %w", path, err)
		}
		output, err := os.OpenFile(filepath.Join(destination, match[2]), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			input.Close()
			return fmt.Errorf("could not stage Latex asset: %w", err)
		}
		_, copyError := io.Copy(output, input)
		closeInput := input.Close()
		closeOutput := output.Close()
		if copyError != nil {
			return fmt.Errorf("could not stage Latex asset: %w", copyError)
		}
		if closeInput != nil {
			return closeInput
		}
		if closeOutput != nil {
			return closeOutput
		}
	}
	return nil
}

func AhdLatexPDFFile(input, output string) {
	if input == "" {
		AhdRaiseClass(AhdClassValueError, "Latex.pdfFile input path must not be empty")
	}
	if output == "" {
		AhdRaiseClass(AhdClassValueError, "Latex.pdfFile output path must not be empty")
	}
	absolute, err := filepath.Abs(input)
	if err != nil {
		ahdLatexRaise("could not resolve input path: " + err.Error())
	}
	info, err := os.Stat(absolute)
	if err != nil || !info.Mode().IsRegular() {
		if err == nil {
			err = fmt.Errorf("not a regular file")
		}
		ahdLatexRaise("could not read LaTeX input " + input + ": " + err.Error())
	}
	ahdLatexCompile(absolute, filepath.Dir(absolute), output)
}

func ahdLatexCompile(input, workingDirectory, output string) {
	engine, bundle, err := ahdLatexRuntime()
	if err != nil {
		ahdLatexRaise(err.Error())
	}
	temporary, err := os.MkdirTemp("", "ahdcode-latex-build-*")
	if err != nil {
		ahdLatexRaise("could not create a secure build directory: " + err.Error())
	}
	defer os.RemoveAll(temporary)
	outputDirectory := filepath.Join(temporary, "output")
	cacheDirectory := filepath.Join(temporary, "cache")
	if err := os.Mkdir(outputDirectory, 0o700); err != nil {
		ahdLatexRaise("could not prepare temporary PDF output: " + err.Error())
	}
	if err := os.Mkdir(cacheDirectory, 0o700); err != nil {
		ahdLatexRaise("could not prepare isolated Tectonic cache: " + err.Error())
	}

	contextValue, cancel := context.WithTimeout(context.Background(), ahdLatexCompileTimeout)
	defer cancel()
	command := exec.CommandContext(contextValue, engine,
		"--untrusted", "--color", "never", "--bundle", bundle, "--only-cached",
		"--outdir", outputDirectory, input)
	command.Dir = workingDirectory
	command.Env = append(os.Environ(), "TECTONIC_CACHE_DIR="+cacheDirectory)
	log := &ahdLatexLog{}
	command.Stdout, command.Stderr = log, log
	err = command.Run()
	if contextValue.Err() == context.DeadlineExceeded {
		ahdLatexRaise("compilation timed out after 30 seconds")
	}
	if err != nil {
		message := ahdLatexDiagnostic(log.String())
		if message == "" {
			message = err.Error()
		}
		ahdLatexRaise("compilation failed: " + message)
	}

	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input)) + ".pdf"
	generated := filepath.Join(outputDirectory, base)
	if err := ahdLatexVerifyPDF(generated); err != nil {
		ahdLatexRaise("Tectonic did not produce a valid PDF: " + err.Error())
	}
	if err := ahdLatexPublish(generated, output); err != nil {
		ahdLatexRaise("could not write output PDF: " + err.Error())
	}
}

func ahdLatexRuntime() (string, string, error) {
	roots := []string{os.Getenv("AHDCODE_LATEX_RUNTIME"), AhdLatexRuntimeHint}
	if executable, err := os.Executable(); err == nil {
		bin := filepath.Dir(executable)
		roots = append(roots,
			filepath.Join(bin, "latex"),
			filepath.Join(bin, "..", "libexec", "ahdcode", "latex"),
		)
	}
	seen := make(map[string]bool)
	for _, root := range roots {
		if root == "" {
			continue
		}
		root = filepath.Clean(root)
		if seen[root] {
			continue
		}
		seen[root] = true
		engineName := "tectonic"
		if filepath.Ext(os.Args[0]) == ".exe" {
			engineName = "tectonic.exe"
		}
		engine := filepath.Join(root, engineName)
		bundle := filepath.Join(root, "ahdcode-latex.ttb")
		engineInfo, engineError := os.Stat(engine)
		bundleInfo, bundleError := os.Stat(bundle)
		if engineError == nil && bundleError == nil && engineInfo.Mode().IsRegular() && bundleInfo.Mode().IsRegular() {
			return engine, bundle, nil
		}
	}
	return "", "", fmt.Errorf("bundled Tectonic engine or local resource bundle is missing")
}

type ahdLatexLog struct {
	data      []byte
	truncated bool
}

func (log *ahdLatexLog) Write(value []byte) (int, error) {
	written := len(value)
	remaining := ahdLatexLogLimit - len(log.data)
	if remaining > 0 {
		if len(value) > remaining {
			value = value[:remaining]
			log.truncated = true
		}
		log.data = append(log.data, value...)
	} else {
		log.truncated = true
	}
	return written, nil
}

func (log *ahdLatexLog) String() string {
	result := string(log.data)
	if log.truncated {
		result += "\n[engine log truncated]"
	}
	return result
}

func ahdLatexDiagnostic(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	start := 0
	for index, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error:") || strings.HasPrefix(strings.TrimSpace(line), "!") {
			start = index
			break
		}
	}
	end := start + 20
	if end > len(lines) {
		end = len(lines)
	}
	result := strings.TrimSpace(strings.Join(lines[start:end], "\n"))
	if len(result) > 8192 {
		result = result[:8192] + "\n[diagnostic truncated]"
	}
	return result
}

func ahdLatexVerifyPDF(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() < 5 {
		return fmt.Errorf("output is missing, empty, or not a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	signature := make([]byte, 5)
	if _, err := io.ReadFull(file, signature); err != nil {
		return err
	}
	if string(signature) != "%PDF-" {
		return fmt.Errorf("output has no PDF signature")
	}
	return nil
}

func ahdLatexPublish(generated, output string) error {
	absolute, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	directory := filepath.Dir(absolute)
	temporary, err := os.CreateTemp(directory, ".ahdcode-latex-output-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	cleanup := func() { _ = os.Remove(temporaryPath) }
	defer cleanup()
	source, err := os.Open(generated)
	if err != nil {
		_ = temporary.Close()
		return err
	}
	_, copyError := io.Copy(temporary, source)
	closeSourceError := source.Close()
	syncError := temporary.Sync()
	closeError := temporary.Close()
	for _, candidate := range []error{copyError, closeSourceError, syncError, closeError} {
		if candidate != nil {
			return candidate
		}
	}
	if err := os.Rename(temporaryPath, absolute); err != nil {
		return err
	}
	return nil
}

func ahdLatexRaise(message string) {
	AhdRaiseClass(AhdClassLatexError, message)
}

// ---------------------------------------------------------------------------
// Constant deep freeze
// ---------------------------------------------------------------------------

// AhdFreezable is every runtime object that participates in the Constant
// deep-freeze graph: List, Pair, and Class instances.
type AhdFreezable interface {
	AhdFreezeGraph(visited map[AhdFreezable]bool)
}

// AhdFreeze deep-freezes the object graph reachable from a Constant binding.
// The traversal is idempotent and terminates on cyclic graphs.
func AhdFreeze[T any](value T) T {
	if target, ok := any(value).(AhdFreezable); ok && target != nil {
		target.AhdFreezeGraph(make(map[AhdFreezable]bool))
	}
	return value
}

// AhdFreezeChild continues a freeze walk into one reachable value. Values with
// no reference identity, such as scalars, are skipped.
func AhdFreezeChild[T any](value T, visited map[AhdFreezable]bool) {
	if child, ok := any(value).(AhdFreezable); ok && child != nil {
		child.AhdFreezeGraph(visited)
	}
}

// AhdEnterFreeze reports whether a graph walk should descend into an object it
// has not visited yet.
func AhdEnterFreeze(target AhdFreezable, visited map[AhdFreezable]bool) bool {
	if target == nil || visited[target] {
		return false
	}
	visited[target] = true
	return true
}

func ahdRejectMutation() {
	AhdRaiseClass(AhdClassConstantError, "cannot mutate a Constant object")
}

// ---------------------------------------------------------------------------
// Null representation
// ---------------------------------------------------------------------------

// AhdBox stores a non-null scalar into a nullable slot.
func AhdBox[T any](value T) *T { return &value }

// AhdNonNull reads a nullable slot that the frontend proved non-null.
func AhdNonNull[T any](value *T) T {
	if value == nil {
		AhdRaiseClass(AhdClassNullError, "value is null")
	}
	return *value
}

// ---------------------------------------------------------------------------
// Terminal I/O
// ---------------------------------------------------------------------------

// AhdWrite prints one canonical line of program output.
func AhdWrite(text string) {
	_, _ = ahdOut.WriteString(text)
	_ = ahdOut.WriteByte('\n')
}

// AhdTake reads one line of terminal input.
func AhdTake() string { return ahdReadLine() }

// AhdTakePrompt writes a prompt, without adding a newline, and then reads one
// line of terminal input. The prompt is never part of the returned String.
func AhdTakePrompt(prompt string) string {
	_, _ = ahdOut.WriteString(prompt)
	return ahdReadLine()
}

// ahdReadLine is the single terminal read behind both take forms. Pending
// output is flushed first so a prompt is visible before the process blocks.
// Only the line terminator is removed, in both the LF and CRLF forms; ordinary
// whitespace in the entered text is preserved. End of input yields an empty
// String.
func ahdReadLine() string {
	AhdFlush()
	line, _ := ahdIn.ReadString('\n')
	line = strings.TrimSuffix(line, "\n")
	return strings.TrimSuffix(line, "\r")
}

// ---------------------------------------------------------------------------
// Checked Int arithmetic
// ---------------------------------------------------------------------------

func ahdOverflow() {
	AhdRaiseClass(AhdClassOverflowError, "Int arithmetic overflowed signed 64-bit range")
}

// AhdIntAdd adds two Int values without silent wrap-around.
func AhdIntAdd(left, right int64) int64 {
	sum := left + right
	if (sum > left) != (right > 0) {
		ahdOverflow()
	}
	return sum
}

// AhdIntSubtract subtracts two Int values without silent wrap-around.
func AhdIntSubtract(left, right int64) int64 {
	difference := left - right
	if (difference < left) != (right > 0) {
		ahdOverflow()
	}
	return difference
}

// AhdIntMultiply multiplies two Int values without silent wrap-around.
func AhdIntMultiply(left, right int64) int64 {
	if left == 0 || right == 0 {
		return 0
	}
	if (left == math.MinInt64 && right == -1) || (right == math.MinInt64 && left == -1) {
		ahdOverflow()
	}
	product := left * right
	if product/right != left {
		ahdOverflow()
	}
	return product
}

// AhdIntNegate negates an Int value without silent wrap-around.
func AhdIntNegate(value int64) int64 {
	if value == math.MinInt64 {
		ahdOverflow()
	}
	return -value
}

// AhdIntModulo is Int remainder with a zero-divisor error.
func AhdIntModulo(left, right int64) int64 {
	if right == 0 {
		AhdRaiseClass(AhdClassDivisionByZeroError, "Int modulo by zero")
	}
	if right == -1 {
		return 0
	}
	return left % right
}

// AhdIntPower raises an Int to a non-negative Int power without silent wrap.
func AhdIntPower(base, exponent int64) int64 {
	if exponent < 0 {
		AhdRaiseClass(AhdClassDomainError, "Int power requires a non-negative exponent")
	}
	result := int64(1)
	current := base
	remaining := exponent
	for remaining > 0 {
		if remaining&1 == 1 {
			result = AhdIntMultiply(result, current)
		}
		remaining >>= 1
		if remaining > 0 {
			current = AhdIntMultiply(current, current)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Checked Real arithmetic
// ---------------------------------------------------------------------------

func ahdRealCheck(result float64, operation string) float64 {
	if math.IsInf(result, 0) {
		AhdRaiseClass(AhdClassOverflowError, "Real "+operation+" produced a non-finite result")
	}
	if math.IsNaN(result) {
		AhdRaiseClass(AhdClassDomainError, "Real "+operation+" is not defined for these operands")
	}
	return result
}

// AhdRealAdd adds two Real values and rejects non-finite results.
func AhdRealAdd(left, right float64) float64 { return ahdRealCheck(left+right, "addition") }

// AhdRealSubtract subtracts two Real values and rejects non-finite results.
func AhdRealSubtract(left, right float64) float64 {
	return ahdRealCheck(left-right, "subtraction")
}

// AhdRealMultiply multiplies two Real values and rejects non-finite results.
func AhdRealMultiply(left, right float64) float64 {
	return ahdRealCheck(left*right, "multiplication")
}

// AhdRealDivide divides two Real values; a zero divisor is an error.
func AhdRealDivide(left, right float64) float64 {
	if right == 0 {
		AhdRaiseClass(AhdClassDivisionByZeroError, "division by zero")
	}
	return ahdRealCheck(left/right, "division")
}

// AhdRealPower raises a Real to a Real power and rejects non-finite results.
func AhdRealPower(base, exponent float64) float64 {
	return ahdRealCheck(math.Pow(base, exponent), "power")
}

// AhdRealNegate negates a Real value.
func AhdRealNegate(value float64) float64 { return -value }

func AhdIntToComplex(value int64) complex128               { return complex(float64(value), 0) }
func AhdRealToComplex(value float64) complex128            { return complex(value, 0) }
func AhdComplexNegate(value complex128) complex128         { return -value }
func AhdComplexAdd(left, right complex128) complex128      { return left + right }
func AhdComplexSubtract(left, right complex128) complex128 { return left - right }
func AhdComplexMultiply(left, right complex128) complex128 { return left * right }
func AhdComplexDivide(left, right complex128) complex128 {
	if right == 0 {
		AhdRaiseClass(AhdClassDivisionByZeroError, "Complex division by zero")
	}
	return left / right
}
func AhdComplexIntPower(base complex128, exponent int64) complex128 {
	if exponent == 0 {
		return complex(1, 0)
	}
	negative := exponent < 0
	var magnitude uint64
	if negative {
		magnitude = uint64(-(exponent + 1)) + 1
	} else {
		magnitude = uint64(exponent)
	}
	result := complex(1, 0)
	for magnitude != 0 {
		if magnitude&1 != 0 {
			result *= base
		}
		magnitude >>= 1
		if magnitude != 0 {
			base *= base
		}
	}
	if negative {
		if result == 0 {
			AhdRaiseClass(AhdClassDivisionByZeroError, "Complex negative power divides by zero")
		}
		return 1 / result
	}
	return result
}
func AhdComplexReal(value complex128) float64         { return real(value) }
func AhdComplexImag(value complex128) float64         { return imag(value) }
func AhdComplexConjugate(value complex128) complex128 { return complex(real(value), -imag(value)) }
func AhdComplexMagnitude(value complex128) float64    { return math.Hypot(real(value), imag(value)) }
func AhdComplexPhase(value complex128) float64        { return math.Atan2(imag(value), real(value)) }

// AhdIntToReal is the explicit Int -> Real widening conversion.
func AhdIntToReal(value int64) float64 { return float64(value) }

// AhdRealToInt truncates toward zero and preserves the checked Int contract.
func AhdRealToInt(value float64) int64 {
	if math.IsNaN(value) {
		AhdRaiseClass(AhdClassDomainError, "cannot convert a non-number Real to Int")
	}
	if math.IsInf(value, 0) || value < -9223372036854775808.0 || value >= 9223372036854775808.0 {
		AhdRaiseClass(AhdClassOverflowError, "Real value is outside signed 64-bit Int range")
	}
	return int64(math.Trunc(value))
}

// AhdStringToInt parses the exact v0.1 signed ASCII-decimal Int grammar.
func AhdStringToInt(value string) int64 {
	text := strings.TrimSpace(value)
	if !ahdValidIntText(text) {
		AhdRaiseClass(AhdClassDomainError, "String is not valid decimal Int text")
	}
	result, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		if number, ok := err.(*strconv.NumError); ok && number.Err == strconv.ErrRange {
			AhdRaiseClass(AhdClassOverflowError, "decimal String is outside signed 64-bit Int range")
		}
		AhdRaiseClass(AhdClassDomainError, "String is not valid decimal Int text")
	}
	return result
}

// AhdStringToReal parses the exact v0.1 finite ASCII-decimal Real grammar.
func AhdStringToReal(value string) float64 {
	text := strings.TrimSpace(value)
	if !ahdValidRealText(text) {
		AhdRaiseClass(AhdClassDomainError, "String is not valid decimal Real text")
	}
	result, err := strconv.ParseFloat(text, 64)
	if err != nil {
		if number, ok := err.(*strconv.NumError); ok && number.Err == strconv.ErrRange {
			AhdRaiseClass(AhdClassOverflowError, "decimal String is outside finite Real range")
		}
		AhdRaiseClass(AhdClassDomainError, "String is not valid decimal Real text")
	}
	if math.IsInf(result, 0) {
		AhdRaiseClass(AhdClassOverflowError, "decimal String is outside finite Real range")
	}
	if math.IsNaN(result) {
		AhdRaiseClass(AhdClassDomainError, "String is not valid decimal Real text")
	}
	if result == 0 && ahdNonzeroRealSignificand(text) {
		AhdRaiseClass(AhdClassOverflowError, "decimal String is outside finite Real range")
	}
	return result
}

func ahdValidIntText(text string) bool {
	index := ahdAfterSign(text)
	end, digits := ahdDecimalDigits(text, index)
	return digits && end == len(text)
}

func ahdValidRealText(text string) bool {
	index := ahdAfterSign(text)
	var digits bool
	index, digits = ahdDecimalDigits(text, index)
	if !digits {
		return false
	}
	if index < len(text) && text[index] == '.' {
		index++
		var fraction bool
		index, fraction = ahdDecimalDigits(text, index)
		if !fraction {
			return false
		}
	}
	if index < len(text) && (text[index] == 'e' || text[index] == 'E') {
		index++
		if index < len(text) && (text[index] == '+' || text[index] == '-') {
			index++
		}
		var exponent bool
		index, exponent = ahdDecimalDigits(text, index)
		if !exponent {
			return false
		}
	}
	return index == len(text)
}

func ahdAfterSign(text string) int {
	if len(text) > 0 && (text[0] == '+' || text[0] == '-') {
		return 1
	}
	return 0
}

func ahdDecimalDigits(text string, index int) (int, bool) {
	start := index
	for index < len(text) && text[index] >= '0' && text[index] <= '9' {
		index++
	}
	return index, index > start
}

func ahdNonzeroRealSignificand(text string) bool {
	for index := ahdAfterSign(text); index < len(text) && text[index] != 'e' && text[index] != 'E'; index++ {
		if text[index] >= '1' && text[index] <= '9' {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// String operations
// ---------------------------------------------------------------------------

// AhdStringRepeat implements String * Int.
func AhdStringRepeat(text string, count int64) string {
	if count < 0 {
		AhdRaiseClass(AhdClassValueError, "String repeat count must not be negative")
	}
	if count > math.MaxInt32 {
		AhdRaiseClass(AhdClassValueError, "String repeat count is too large")
	}
	return strings.Repeat(text, int(count))
}

// AhdStringChars splits a String into its one-character String elements.
func AhdStringChars(text string) []string {
	runes := []rune(text)
	result := make([]string, len(runes))
	for index, value := range runes {
		result[index] = string(value)
	}
	return result
}

// AhdStringTrim removes leading and trailing Unicode whitespace and preserves
// every interior character.
func AhdStringTrim(text string) string { return strings.TrimSpace(text) }

// AhdStringLower is deterministic, locale-independent Unicode lowercasing.
func AhdStringLower(text string) string { return strings.ToLower(text) }

// AhdStringUpper is deterministic, locale-independent Unicode uppercasing.
func AhdStringUpper(text string) string { return strings.ToUpper(text) }

// AhdStringCapitalize uppercases the first character and leaves the remainder
// exactly as written.
func AhdStringCapitalize(text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}
	return string(unicode.ToUpper(runes[0])) + string(runes[1:])
}

// AhdStringSplit divides a String on every non-overlapping occurrence of a
// non-empty separator, preserving empty fields. A split field is never null,
// so the result is a List<String>, not a List<String?>.
func AhdStringSplit(text, separator string) *AhdList[string] {
	if separator == "" {
		AhdRaiseClass(AhdClassDomainError, "split requires a non-empty separator")
	}
	parts := strings.Split(text, separator)
	items := make([]string, len(parts))
	copy(items, parts)
	return &AhdList[string]{items: items}
}

// AhdStringReplace rewrites every non-overlapping occurrence of a non-empty
// search text. The replacement may be empty.
func AhdStringReplace(text, old, replacement string) string {
	if old == "" {
		AhdRaiseClass(AhdClassDomainError, "replace requires a non-empty search text")
	}
	return strings.ReplaceAll(text, old, replacement)
}

// AhdStringStartsWith reports a prefix match.
func AhdStringStartsWith(text, prefix string) bool { return strings.HasPrefix(text, prefix) }

// AhdStringEndsWith reports a suffix match.
func AhdStringEndsWith(text, suffix string) bool { return strings.HasSuffix(text, suffix) }

// AhdStringCount counts non-overlapping occurrences. An empty search text has
// no occurrence count in AhdCode, so it is rejected rather than defined as one
// match per position.
func AhdStringCount(text, needle string) int64 {
	if needle == "" {
		AhdRaiseClass(AhdClassDomainError, "count requires a non-empty search text")
	}
	return int64(strings.Count(text, needle))
}

// AhdStringIndex is the first character index of a search text. The result is
// an AhdCode character index, never a UTF-8 byte offset, and a missing text is
// an error rather than a sentinel index.
func AhdStringIndex(text, needle string) int64 {
	if needle == "" {
		AhdRaiseClass(AhdClassDomainError, "index requires a non-empty search text")
	}
	position := strings.Index(text, needle)
	if position < 0 {
		AhdRaiseClass(AhdClassDomainError, "index did not find the search text")
	}
	return int64(len([]rune(text[:position])))
}

// AhdStringLen counts characters, not bytes.
func AhdStringLen(text string) int64 { return int64(len([]rune(text))) }

// AhdStringAt returns the one-character String at a possibly negative index.
func AhdStringAt(text string, index int64) string {
	runes := []rune(text)
	position := ahdResolveIndex(index, int64(len(runes)))
	return string(runes[position])
}

// AhdStringSlice slices by character position with optional bounds.
func AhdStringSlice(text string, start int64, hasStart bool, end int64, hasEnd bool) string {
	runes := []rune(text)
	low, high := ahdResolveRange(start, hasStart, end, hasEnd, int64(len(runes)))
	return string(runes[low:high])
}

func ahdResolveIndex(index, length int64) int64 {
	position := index
	if position < 0 {
		position += length
	}
	if position < 0 || position >= length {
		AhdRaiseClass(AhdClassIndexError, "index "+strconv.FormatInt(index, 10)+" is out of range for length "+strconv.FormatInt(length, 10))
	}
	return position
}

func ahdResolveRange(start int64, hasStart bool, end int64, hasEnd bool, length int64) (int64, int64) {
	low := int64(0)
	if hasStart {
		low = start
		if low < 0 {
			low += length
		}
	}
	high := length
	if hasEnd {
		high = end
		if high < 0 {
			high += length
		}
	}
	if low < 0 {
		low = 0
	}
	if high > length {
		high = length
	}
	if low > length {
		low = length
	}
	if high < low {
		high = low
	}
	return low, high
}

// ---------------------------------------------------------------------------
// Lazy integer iteration
// ---------------------------------------------------------------------------

// AhdRange is the lazy integer iteration produced by between. Its whole state
// is the current value, the excluded stop, and the step, so iterating any
// range costs O(1) memory no matter how many values it yields. It never
// materializes a List.
type AhdRange struct {
	current  int64
	stop     int64
	step     int64
	finished bool
}

// AhdBetween builds a lazy integer iteration. A zero step cannot make progress
// and is a DomainError rather than a silently substituted step of one.
func AhdBetween(start, stop, step int64) *AhdRange {
	if step == 0 {
		AhdRaiseClass(AhdClassDomainError, "between requires a non-zero step")
	}
	return &AhdRange{current: start, stop: stop, step: step}
}

// Next yields the next Int of the iteration. It computes each value on demand
// and stops before any step that would leave the signed 64-bit range, so a
// range near the Int boundaries terminates instead of wrapping.
func (iteration *AhdRange) Next() (int64, bool) {
	if iteration == nil || iteration.finished {
		return 0, false
	}
	if iteration.step > 0 && iteration.current >= iteration.stop {
		iteration.finished = true
		return 0, false
	}
	if iteration.step < 0 && iteration.current <= iteration.stop {
		iteration.finished = true
		return 0, false
	}
	value := iteration.current
	// A step that would overflow can only move past a stop that already fits in
	// Int, so the iteration is complete rather than in error.
	if iteration.step > 0 && iteration.current > math.MaxInt64-iteration.step {
		iteration.finished = true
		return value, true
	}
	if iteration.step < 0 && iteration.current < math.MinInt64-iteration.step {
		iteration.finished = true
		return value, true
	}
	iteration.current += iteration.step
	return value, true
}

// ---------------------------------------------------------------------------
// List: reference semantics with stable object identity
// ---------------------------------------------------------------------------

// AhdList is the pointer-backed runtime representation of List<T>. Aliases
// share one AhdList value, so in-place mutation is observed by every alias.
type AhdList[T any] struct {
	items  []T
	frozen bool
	id     int64
}

// AhdIdentity lazily assigns and then returns this List's stable identity
// number. It belongs to the List object itself, not its backing array, so
// growth and reallocation never change it.
func (list *AhdList[T]) AhdIdentity() int64 {
	if list.id == 0 {
		list.id = ahdNextIdentity()
	}
	return list.id
}

// AhdFreezeGraph deep-freezes this List and every reachable element.
func (list *AhdList[T]) AhdFreezeGraph(visited map[AhdFreezable]bool) {
	if list == nil || !AhdEnterFreeze(list, visited) {
		return
	}
	list.frozen = true
	for _, item := range list.items {
		AhdFreezeChild(item, visited)
	}
}

func (list *AhdList[T]) requireMutable() {
	list.require()
	if list.frozen {
		ahdRejectMutation()
	}
}

// AhdNewList builds a List from its literal elements.
func AhdNewList[T any](items ...T) *AhdList[T] {
	return &AhdList[T]{items: items}
}

func (list *AhdList[T]) require() {
	if list == nil {
		AhdRaiseClass(AhdClassNullError, "List value is null")
	}
}

// Len reports the element count.
func (list *AhdList[T]) Len() int64 {
	list.require()
	return int64(len(list.items))
}

// At reads a possibly negative index.
func (list *AhdList[T]) At(index int64) T {
	list.require()
	return list.items[ahdResolveIndex(index, int64(len(list.items)))]
}

// Set writes a possibly negative index.
func (list *AhdList[T]) Set(index int64, value T) {
	list.requireMutable()
	list.items[ahdResolveIndex(index, int64(len(list.items)))] = value
}

// Add appends one element, mutating the List in place so every alias observes
// the new element.
func (list *AhdList[T]) Add(value T) {
	list.requireMutable()
	list.items = append(list.items, value)
}

// Eject removes the element at a possibly negative index, mutating the List in
// place. An out-of-range index is an IndexError.
func (list *AhdList[T]) Eject(index int64) {
	list.requireMutable()
	position := ahdResolveIndex(index, int64(len(list.items)))
	list.items = append(list.items[:position], list.items[position+1:]...)
}

// Clear empties the List in place, preserving object identity.
func (list *AhdList[T]) Clear() {
	list.requireMutable()
	list.items = nil
}

// Snapshot returns the shallow iteration snapshot taken at loop entry.
func (list *AhdList[T]) Snapshot() []T {
	list.require()
	result := make([]T, len(list.items))
	copy(result, list.items)
	return result
}

// Slice produces a new List over a character-free index range.
func (list *AhdList[T]) Slice(start int64, hasStart bool, end int64, hasEnd bool) *AhdList[T] {
	list.require()
	low, high := ahdResolveRange(start, hasStart, end, hasEnd, int64(len(list.items)))
	items := make([]T, high-low)
	copy(items, list.items[low:high])
	return &AhdList[T]{items: items}
}

// AhdListConcat implements List<T> + List<T>.
func AhdListConcat[T any](left, right *AhdList[T]) *AhdList[T] {
	left.require()
	right.require()
	items := make([]T, 0, len(left.items)+len(right.items))
	items = append(items, left.items...)
	items = append(items, right.items...)
	return &AhdList[T]{items: items}
}

// AhdListContains implements value membership for List.
func AhdListContains[T any](list *AhdList[T], value T, equal func(T, T) bool) bool {
	list.require()
	for _, item := range list.items {
		if equal(item, value) {
			return true
		}
	}
	return false
}

// AhdListEqual implements deep List value equality.
func AhdListEqual[T any](left, right *AhdList[T], equal func(T, T) bool) bool {
	if left == nil || right == nil {
		return left == right
	}
	if len(left.items) != len(right.items) {
		return false
	}
	for index := range left.items {
		if !equal(left.items[index], right.items[index]) {
			return false
		}
	}
	return true
}

// Reverse reverses the List in place, so every alias observes the new order.
func (list *AhdList[T]) Reverse() {
	list.requireMutable()
	for left, right := 0, len(list.items)-1; left < right; left, right = left+1, right-1 {
		list.items[left], list.items[right] = list.items[right], list.items[left]
	}
}

// Shuffle permutes the List in place with an unbiased descending
// Fisher-Yates pass. AhdMathRandomInt owns all generator advancement, so this
// operation shares the exact Math RNG sequence with random and randomInt.
func (list *AhdList[T]) Shuffle() {
	list.requireMutable()
	for index := len(list.items) - 1; index > 0; index-- {
		selected := int(AhdMathRandomInt(0, int64(index)))
		list.items[index], list.items[selected] = list.items[selected], list.items[index]
	}
}

// AhdListCount counts equal elements with the ordinary AhdCode == semantics.
func AhdListCount[T any](list *AhdList[T], value T, equal func(T, T) bool) int64 {
	list.require()
	total := int64(0)
	for _, item := range list.items {
		if equal(item, value) {
			total++
		}
	}
	return total
}

// AhdListIndex is the first index of an equal element. A value the List does
// not contain is an error rather than a sentinel index.
func AhdListIndex[T any](list *AhdList[T], value T, equal func(T, T) bool) int64 {
	list.require()
	for index, item := range list.items {
		if equal(item, value) {
			return int64(index)
		}
	}
	AhdRaiseClass(AhdClassDomainError, "index did not find the value in the List")
	return 0
}

// AhdListMap builds a new List from a Function applied left to right to a
// shallow snapshot, so a callback that mutates the source cannot change what
// is iterated. The source List is never modified.
func AhdListMap[T any, U any](list *AhdList[T], transform func(T) U) *AhdList[U] {
	items := list.Snapshot()
	result := make([]U, 0, len(items))
	for _, item := range items {
		result = append(result, transform(item))
	}
	return &AhdList[U]{items: result}
}

// AhdListFilter builds a new List of the snapshot elements a predicate keeps.
// The predicate must produce a real Bool: AhdCode has no truthiness, so a null
// result is an error.
func AhdListFilter[T any](list *AhdList[T], keep func(T) *bool) *AhdList[T] {
	items := list.Snapshot()
	result := make([]T, 0, len(items))
	for _, item := range items {
		if ahdPredicate(keep(item)) {
			result = append(result, item)
		}
	}
	return &AhdList[T]{items: result}
}

func ahdPredicate(value *bool) bool {
	if value == nil {
		AhdRaiseClass(AhdClassNullError, "filter predicate returned null")
	}
	return *value
}

func ahdSortKey[K any](value *K) K {
	if value == nil {
		AhdRaiseClass(AhdClassNullError, "sort key Function returned null")
	}
	return *value
}

// ahdOrdered is the set of key types the v0.1 ordering supports.
type ahdOrdered interface{ ~int64 | ~float64 | ~string }

// ahdSortNatural orders a scalar List ascending and stably. Every element is
// read before the receiver is rewritten, so a null element leaves the original
// order untouched.
func ahdSortNatural[T ahdOrdered](list *AhdList[*T]) {
	list.requireMutable()
	items := list.items
	sorted := make([]*T, len(items))
	for index, item := range items {
		sorted[index] = ahdElementPointer(item, "sort")
	}
	sort.SliceStable(sorted, func(left, right int) bool { return *sorted[left] < *sorted[right] })
	list.items = sorted
}

func ahdElementPointer[T any](value *T, name string) *T {
	if value == nil {
		AhdRaiseClass(AhdClassNullError, name+" does not accept a null List element")
	}
	return value
}

// ahdSortNaturalNonNull is ahdSortNatural for a List<T> whose elements are
// never null, so no per-element null check is needed at all.
func ahdSortNaturalNonNull[T ahdOrdered](list *AhdList[T]) {
	list.requireMutable()
	sorted := make([]T, len(list.items))
	copy(sorted, list.items)
	sort.SliceStable(sorted, func(left, right int) bool { return sorted[left] < sorted[right] })
	list.items = sorted
}

// ahdSortByKey orders a List ascending and stably by a key Function. Every key
// is computed, left to right and exactly once per element, before the receiver
// is rewritten, so a key Function that raises leaves the original order
// unchanged.
func ahdSortByKey[T any, K ahdOrdered](list *AhdList[T], key func(T) *K) {
	list.requireMutable()
	items := list.items
	keys := make([]K, len(items))
	for index, item := range items {
		keys[index] = ahdSortKey(key(item))
	}
	order := make([]int, len(items))
	for index := range order {
		order[index] = index
	}
	sort.SliceStable(order, func(left, right int) bool { return keys[order[left]] < keys[order[right]] })
	sorted := make([]T, len(items))
	for position, index := range order {
		sorted[position] = items[index]
	}
	list.items = sorted
}

// AhdListSortInt orders a List<Int?> ascending in place.
func AhdListSortInt(list *AhdList[*int64]) { ahdSortNatural(list) }

// AhdListSortReal orders a List<Real?> ascending in place.
func AhdListSortReal(list *AhdList[*float64]) { ahdSortNatural(list) }

// AhdListSortString orders a List<String?> ascending in place.
func AhdListSortString(list *AhdList[*string]) { ahdSortNatural(list) }

// AhdListSortIntNonNull orders a List<Int> ascending in place.
func AhdListSortIntNonNull(list *AhdList[int64]) { ahdSortNaturalNonNull(list) }

// AhdListSortRealNonNull orders a List<Real> ascending in place.
func AhdListSortRealNonNull(list *AhdList[float64]) { ahdSortNaturalNonNull(list) }

// AhdListSortStringNonNull orders a List<String> ascending in place.
func AhdListSortStringNonNull(list *AhdList[string]) { ahdSortNaturalNonNull(list) }

// AhdListSortKeyInt orders a List by an Int key.
func AhdListSortKeyInt[T any](list *AhdList[T], key func(T) *int64) { ahdSortByKey(list, key) }

// AhdListSortKeyReal orders a List by a Real key.
func AhdListSortKeyReal[T any](list *AhdList[T], key func(T) *float64) { ahdSortByKey(list, key) }

// AhdListSortKeyString orders a List by a String key.
func AhdListSortKeyString[T any](list *AhdList[T], key func(T) *string) { ahdSortByKey(list, key) }

// ---------------------------------------------------------------------------
// Path and File standard modules
// ---------------------------------------------------------------------------

func AhdPathJoin(parts *AhdList[string]) string {
	return filepath.Join(parts.Snapshot()...)
}

func AhdPathExt(path string) string  { return filepath.Ext(path) }
func AhdPathBase(path string) string { return filepath.Base(path) }
func AhdPathDir(path string) string  { return filepath.Dir(path) }

func ahdFileFailure(class *AhdClass, operation, path string, err error) {
	message := operation + " " + strconv.Quote(path) + " failed"
	if err != nil {
		message += ": " + err.Error()
	}
	AhdRaiseClass(class, message)
}

func AhdFileExists(class *AhdClass, path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	ahdFileFailure(class, "stat", path, err)
	return false
}

func AhdFileReadText(class *AhdClass, path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		ahdFileFailure(class, "read", path, err)
	}
	if !utf8.Valid(content) {
		ahdFileFailure(class, "read", path, fmt.Errorf("content is not valid UTF-8"))
	}
	return string(content)
}

func AhdFileWriteText(class *AhdClass, path, content string) {
	if !utf8.ValidString(content) {
		ahdFileFailure(class, "write", path, fmt.Errorf("content is not valid UTF-8"))
	}
	if err := os.WriteFile(path, []byte(content), 0o666); err != nil {
		ahdFileFailure(class, "write", path, err)
	}
}

func AhdFileAppend(class *AhdClass, path, content string) {
	if !utf8.ValidString(content) {
		ahdFileFailure(class, "append", path, fmt.Errorf("content is not valid UTF-8"))
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		ahdFileFailure(class, "append", path, err)
	}
	if _, err = io.WriteString(file, content); err != nil {
		_ = file.Close()
		ahdFileFailure(class, "append", path, err)
	}
	if err = file.Close(); err != nil {
		ahdFileFailure(class, "close", path, err)
	}
}

func AhdFileDelete(class *AhdClass, path string) {
	if err := os.Remove(path); err != nil {
		ahdFileFailure(class, "delete", path, err)
	}
}

func AhdFileCreateDir(class *AhdClass, path string) {
	if err := os.MkdirAll(path, 0o777); err != nil {
		ahdFileFailure(class, "create directory", path, err)
	}
}

func AhdFileList(class *AhdClass, path string) *AhdList[string] {
	entries, err := os.ReadDir(path)
	if err != nil {
		ahdFileFailure(class, "list", path, err)
	}
	names := make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}
	sort.Strings(names)
	return AhdNewList(names...)
}

// ---------------------------------------------------------------------------
// CSV standard module
// ---------------------------------------------------------------------------

func ahdCSVDelimiter(class *AhdClass, delimiter string) rune {
	if !utf8.ValidString(delimiter) {
		AhdRaiseClass(class, "delimiter is not valid UTF-8")
	}
	runes := []rune(delimiter)
	if len(runes) != 1 || runes[0] == '\r' || runes[0] == '\n' || runes[0] == '"' || runes[0] == utf8.RuneError || runes[0] == 0 {
		AhdRaiseClass(class, "delimiter must be exactly one valid Unicode scalar other than quote, CR, or LF")
	}
	return runes[0]
}

func ahdCSVRows(class *AhdClass, text, delimiter string) [][]string {
	if !utf8.ValidString(text) {
		AhdRaiseClass(class, "CSV text is not valid UTF-8")
	}
	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = ahdCSVDelimiter(class, delimiter)
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		AhdRaiseClass(class, "invalid CSV: "+err.Error())
	}
	return rows
}

func ahdCSVList(rows [][]string) *AhdList[*AhdList[string]] {
	result := make([]*AhdList[string], len(rows))
	for index, row := range rows {
		result[index] = AhdNewList(row...)
	}
	return AhdNewList(result...)
}

func AhdCSVParse(class *AhdClass, text, delimiter string) *AhdList[*AhdList[string]] {
	return ahdCSVList(ahdCSVRows(class, text, delimiter))
}

func AhdCSVStringify(class *AhdClass, rows *AhdList[*AhdList[string]], delimiter string) string {
	comma := ahdCSVDelimiter(class, delimiter)
	if rows == nil {
		AhdRaiseClass(AhdClassNullError, "List value is null")
	}
	var output strings.Builder
	writer := csv.NewWriter(&output)
	writer.Comma = comma
	for _, row := range rows.Snapshot() {
		if row == nil {
			AhdRaiseClass(AhdClassNullError, "CSV row is null")
		}
		if err := writer.Write(row.Snapshot()); err != nil {
			AhdRaiseClass(class, "could not encode CSV: "+err.Error())
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		AhdRaiseClass(class, "could not encode CSV: "+err.Error())
	}
	return output.String()
}

func AhdCSVRead(csvClass, fileClass *AhdClass, path, delimiter string) *AhdList[*AhdList[string]] {
	content, err := os.ReadFile(path)
	if err != nil {
		ahdFileFailure(fileClass, "read", path, err)
	}
	return AhdCSVParse(csvClass, string(content), delimiter)
}

func AhdCSVWrite(csvClass, fileClass *AhdClass, path string, rows *AhdList[*AhdList[string]], delimiter string) {
	content := AhdCSVStringify(csvClass, rows, delimiter)
	if err := os.WriteFile(path, []byte(content), 0o666); err != nil {
		ahdFileFailure(fileClass, "write", path, err)
	}
}

func ahdCSVRecords(class *AhdClass, rows [][]string) *AhdList[*AhdPair[string, string]] {
	if len(rows) == 0 {
		return AhdNewList[*AhdPair[string, string]]()
	}
	headers := rows[0]
	seen := make(map[string]bool, len(headers))
	for _, header := range headers {
		if header == "" {
			AhdRaiseClass(class, "record headers must not be empty")
		}
		if seen[header] {
			AhdRaiseClass(class, "record headers must be unique; duplicate "+strconv.Quote(header))
		}
		seen[header] = true
	}
	result := make([]*AhdPair[string, string], 0, len(rows)-1)
	for rowIndex, row := range rows[1:] {
		if len(row) != len(headers) {
			AhdRaiseClass(class, "record row "+strconv.Itoa(rowIndex+2)+" has "+strconv.Itoa(len(row))+" fields; expected "+strconv.Itoa(len(headers)))
		}
		record := AhdNewPair[string, string]()
		for index, header := range headers {
			record.Set(header, row[index])
		}
		result = append(result, record)
	}
	return AhdNewList(result...)
}

func AhdCSVParseRecords(class *AhdClass, text, delimiter string) *AhdList[*AhdPair[string, string]] {
	return ahdCSVRecords(class, ahdCSVRows(class, text, delimiter))
}

func AhdCSVReadRecords(csvClass, fileClass *AhdClass, path, delimiter string) *AhdList[*AhdPair[string, string]] {
	content, err := os.ReadFile(path)
	if err != nil {
		ahdFileFailure(fileClass, "read", path, err)
	}
	return AhdCSVParseRecords(csvClass, string(content), delimiter)
}

func AhdCSVStringifyRecords(class *AhdClass, records *AhdList[*AhdPair[string, string]], delimiter string) string {
	ahdCSVDelimiter(class, delimiter)
	if records == nil {
		AhdRaiseClass(AhdClassNullError, "List value is null")
	}
	items := records.Snapshot()
	if len(items) == 0 {
		return ""
	}
	if items[0] == nil {
		AhdRaiseClass(AhdClassNullError, "CSV record is null")
	}
	headers := items[0].Keys()
	if len(headers) == 0 {
		AhdRaiseClass(class, "records must contain at least one column")
	}
	rows := make([]*AhdList[string], 0, len(items)+1)
	rows = append(rows, AhdNewList(headers...))
	for recordIndex, record := range items {
		if record == nil {
			AhdRaiseClass(AhdClassNullError, "CSV record is null")
		}
		keys := record.Keys()
		if len(keys) != len(headers) {
			AhdRaiseClass(class, "record "+strconv.Itoa(recordIndex+1)+" does not have the same key set as the first record")
		}
		row := make([]string, len(headers))
		for index, header := range headers {
			if !record.Has(header) {
				AhdRaiseClass(class, "record "+strconv.Itoa(recordIndex+1)+" is missing key "+strconv.Quote(header))
			}
			row[index] = record.Get(header)
		}
		rows = append(rows, AhdNewList(row...))
	}
	return AhdCSVStringify(class, AhdNewList(rows...), delimiter)
}

func AhdCSVWriteRecords(csvClass, fileClass *AhdClass, path string, records *AhdList[*AhdPair[string, string]], delimiter string) {
	content := AhdCSVStringifyRecords(csvClass, records, delimiter)
	if err := os.WriteFile(path, []byte(content), 0o666); err != nil {
		ahdFileFailure(fileClass, "write", path, err)
	}
}

// ---------------------------------------------------------------------------
// Regex standard module
// ---------------------------------------------------------------------------

var (
	ahdRegexMu    sync.Mutex
	ahdRegexCache = map[string]*regexp.Regexp{}
)

// ahdRegexCompiled compiles a pattern once and caches it by its exact source
// text, so a Regex value that is used many times pays the compilation cost
// only the first time. The cache is process-lifetime; AhdCode has no way to
// evict from it, and pattern texts in real programs are bounded in practice.
func ahdRegexCompiled(class *AhdClass, pattern string) *regexp.Regexp {
	ahdRegexMu.Lock()
	if cached, ok := ahdRegexCache[pattern]; ok {
		ahdRegexMu.Unlock()
		return cached
	}
	ahdRegexMu.Unlock()
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		AhdRaiseClass(class, fmt.Sprintf("invalid Regex pattern %q: %v", pattern, err))
	}
	ahdRegexMu.Lock()
	ahdRegexCache[pattern] = compiled
	ahdRegexMu.Unlock()
	return compiled
}

// AhdRegexValidate compiles (and caches) a pattern, raising the given Regex
// error class on invalid syntax, then returns the pattern unchanged so it
// composes directly as the generated Regex constructor's sole argument.
func AhdRegexValidate(class *AhdClass, pattern string) string {
	ahdRegexCompiled(class, pattern)
	return pattern
}

// AhdRegexMatches reports whether the pattern is found anywhere in text.
func AhdRegexMatches(class *AhdClass, pattern, text string) bool {
	return ahdRegexCompiled(class, pattern).MatchString(text)
}

// AhdRegexFind returns the first match, or nil (AhdCode null) if the pattern
// is not found. It uses the match's index rather than its text so an empty
// match is distinguished from no match at all.
func AhdRegexFind(class *AhdClass, pattern, text string) *string {
	location := ahdRegexCompiled(class, pattern).FindStringIndex(text)
	if location == nil {
		return nil
	}
	result := text[location[0]:location[1]]
	return &result
}

// AhdRegexFindAll returns every non-overlapping match, in order.
func AhdRegexFindAll(class *AhdClass, pattern, text string) *AhdList[string] {
	return AhdNewList(ahdRegexCompiled(class, pattern).FindAllString(text, -1)...)
}

// AhdRegexGroups returns the first match's full match followed by its
// capture groups, or nil (AhdCode null) if the pattern is not found. An
// unmatched optional group reports as an empty String, matching Go's own
// regexp/submatch convention.
func AhdRegexGroups(class *AhdClass, pattern, text string) *AhdList[string] {
	match := ahdRegexCompiled(class, pattern).FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	return AhdNewList(match...)
}

// AhdRegexReplace rewrites every match with a replacement, which may
// reference capture groups as $1, $2, and so on.
func AhdRegexReplace(class *AhdClass, pattern, text, replacement string) string {
	return ahdRegexCompiled(class, pattern).ReplaceAllString(text, replacement)
}

// AhdRegexSplit divides text on every match of the pattern.
func AhdRegexSplit(class *AhdClass, pattern, text string) *AhdList[string] {
	return AhdNewList(ahdRegexCompiled(class, pattern).Split(text, -1)...)
}

// ---------------------------------------------------------------------------
// Time standard module
// ---------------------------------------------------------------------------

// AhdCivilTime is the calendar breakdown of one instant in the host's local
// time. It is the only shape that crosses between the runtime clock and the
// generated DateTime value, so no Go time type ever reaches AhdCode.
type AhdCivilTime struct {
	Year, Month, Day, Hour, Minute, Second, Millisecond, Weekday, OffsetMinutes int64
	// OffsetSeconds carries the leftover seconds of a UTC offset that is not a
	// whole number of minutes. AhdCode publishes offsetMinutes only, so this
	// field is runtime representation rather than a visible attribute.
	OffsetSeconds int64
}

// ahdMonotonicOrigin anchors the monotonic clock. Go's time.Since uses the
// process monotonic reading, so the result never moves backwards even when the
// wall clock is adjusted.
var ahdMonotonicOrigin = time.Now()

// ahdCivilFrom converts a local instant into the calendar fields AhdCode
// publishes, using the Monday=1..Sunday=7 convention.
func ahdCivilFrom(value time.Time) AhdCivilTime {
	if value.Year() < 1 || value.Year() > 9999 {
		AhdRaiseClass(AhdClassValueError, "instant is outside the supported DateTime range")
	}
	weekday := int64(value.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	// A historical host zone can sit at a UTC offset that is not a whole
	// number of minutes: Europe/Istanbul is +01:55:52 (LMT) before 1880. The
	// published offsetMinutes stays minute-based, so the leftover seconds are
	// kept separately instead of being dropped, which would silently move the
	// instant, or rejected, which would refuse ordinary historical dates.
	// Go truncates toward zero and %  keeps the dividend's sign, so
	// minutes*60+seconds reproduces the original offset for both signs.
	_, offsetSeconds := value.Zone()
	return AhdCivilTime{
		Year: int64(value.Year()), Month: int64(value.Month()), Day: int64(value.Day()),
		Hour: int64(value.Hour()), Minute: int64(value.Minute()), Second: int64(value.Second()),
		Millisecond: int64(value.Nanosecond() / 1e6), Weekday: weekday,
		OffsetMinutes: int64(offsetSeconds / 60), OffsetSeconds: int64(offsetSeconds % 60),
	}
}

// AhdTimeNow reads the current local date and time.
func AhdTimeNow() AhdCivilTime { return ahdCivilFrom(time.Now()) }

// AhdTimeUTC reads the current civil date and time in UTC.
func AhdTimeUTC() AhdCivilTime { return ahdCivilFrom(time.Now().UTC()) }

// AhdTimeTimestamp is the current Unix timestamp in milliseconds.
func AhdTimeTimestamp() int64 { return time.Now().UnixMilli() }

// AhdTimeFromTimestamp converts a representable Unix millisecond timestamp to UTC.
func AhdTimeFromTimestamp(milliseconds int64) AhdCivilTime {
	value := time.UnixMilli(milliseconds).UTC()
	if value.Year() < 1 || value.Year() > 9999 {
		AhdRaiseClass(AhdClassValueError, "timestamp is outside the supported DateTime range")
	}
	return ahdCivilFrom(value)
}

// AhdTimeMonotonic is elapsed seconds on a clock that never moves backwards.
// Its absolute value has no calendar meaning; only differences do.
func AhdTimeMonotonic() float64 { return time.Since(ahdMonotonicOrigin).Seconds() }

// AhdTimeSleep pauses for a whole number of milliseconds. Zero returns at
// once, and a negative request is rejected rather than clamped.
func AhdTimeSleep(milliseconds int64) {
	if milliseconds < 0 {
		AhdRaiseClass(AhdClassValueError, "sleep requires a non-negative number of milliseconds")
	}
	if milliseconds == 0 {
		return
	}
	time.Sleep(time.Duration(milliseconds) * time.Millisecond)
}

// AhdTimeCivil validates a civil date and time and returns its fields. Every
// component is checked against the Gregorian calendar, so an impossible date
// is an error rather than a silently rolled-over one.
func AhdTimeCivil(year, month, day, hour, minute, second, millisecond int64) AhdCivilTime {
	return ahdTimeCivilIn(year, month, day, hour, minute, second, millisecond, time.Local)
}

// AhdTimeCivilUTC constructs a validated UTC civil time.
func AhdTimeCivilUTC(year, month, day, hour, minute, second, millisecond int64) AhdCivilTime {
	return ahdTimeCivilIn(year, month, day, hour, minute, second, millisecond, time.UTC)
}

// AhdTimeCivilOffset constructs a validated civil time at a fixed minute offset.
func AhdTimeCivilOffset(year, month, day, offsetMinutes, hour, minute, second, millisecond int64) AhdCivilTime {
	ahdRequireOffset(offsetMinutes)
	return ahdTimeCivilIn(year, month, day, hour, minute, second, millisecond,
		time.FixedZone("", int(offsetMinutes*60)))
}

func ahdTimeCivilIn(year, month, day, hour, minute, second, millisecond int64, location *time.Location) AhdCivilTime {
	ahdRequireRange(year, 1, 9999, "year")
	ahdRequireRange(month, 1, 12, "month")
	ahdRequireRange(day, 1, AhdCalendarDaysInMonth(year, month), "day")
	ahdRequireRange(hour, 0, 23, "hour")
	ahdRequireRange(minute, 0, 59, "minute")
	ahdRequireRange(second, 0, 59, "second")
	ahdRequireRange(millisecond, 0, 999, "millisecond")
	value := time.Date(int(year), time.Month(month), int(day), int(hour), int(minute), int(second),
		int(millisecond)*1e6, location)
	return ahdCivilFrom(value)
}

func ahdRequireOffset(offsetMinutes int64) {
	ahdRequireRange(offsetMinutes, -840, 840, "offsetMinutes")
}

func ahdRequireRange(value, low, high int64, name string) {
	if value < low || value > high {
		AhdRaiseClass(AhdClassValueError, name+" "+strconv.FormatInt(value, 10)+
			" is outside "+strconv.FormatInt(low, 10)+".."+strconv.FormatInt(high, 10))
	}
}

// ahdInstant rebuilds the instant a DateTime denotes from its civil fields and
// its offset. offsetSeconds is the sub-minute remainder of a historical local
// offset and is zero for every offset AhdCode source can name, so the instant
// stays exact without the published attributes growing.
func ahdInstant(year, month, day, hour, minute, second, millisecond, offsetMinutes, offsetSeconds int64) time.Time {
	ahdRequireOffset(offsetMinutes)
	return time.Date(int(year), time.Month(month), int(day), int(hour), int(minute), int(second),
		int(millisecond)*1e6, time.FixedZone("", int(offsetMinutes*60+offsetSeconds)))
}

// AhdTimeCompare orders two DateTime values: negative, zero, or positive.
func AhdTimeCompare(left, right time.Time) int64 {
	switch {
	case left.Before(right):
		return -1
	case left.After(right):
		return 1
	default:
		return 0
	}
}

// AhdTimeInstant is the generated code's entry point for rebuilding an instant.
func AhdTimeInstant(year, month, day, hour, minute, second, millisecond, offsetMinutes, offsetSeconds int64) time.Time {
	return ahdInstant(year, month, day, hour, minute, second, millisecond, offsetMinutes, offsetSeconds)
}

// AhdTimeInstantCivil rebuilds an instant from the runtime interchange shape.
func AhdTimeInstantCivil(value AhdCivilTime) time.Time {
	return AhdTimeInstant(value.Year, value.Month, value.Day, value.Hour, value.Minute,
		value.Second, value.Millisecond, value.OffsetMinutes, value.OffsetSeconds)
}

// AhdTimeInstantTimestamp returns one DateTime instant as Unix milliseconds.
func AhdTimeInstantTimestamp(value time.Time) int64 { return value.UnixMilli() }

// AhdTimeToUTC, AhdTimeToLocal, and AhdTimeToOffset preserve the instant while
// selecting the requested civil representation.
func AhdTimeToUTC(value time.Time) AhdCivilTime { return ahdCivilFrom(value.UTC()) }

func AhdTimeToLocal(value time.Time) AhdCivilTime { return ahdCivilFrom(value.In(time.Local)) }

func AhdTimeToOffset(value time.Time, offsetMinutes int64) AhdCivilTime {
	ahdRequireOffset(offsetMinutes)
	return ahdCivilFrom(value.In(time.FixedZone("", int(offsetMinutes*60))))
}

// AhdTimeDifference is the signed millisecond difference second minus first.
func AhdTimeDifference(first, second time.Time) int64 {
	return second.UnixMilli() - first.UnixMilli()
}

// AhdTimeText is the stable, locale-independent DateTime text.
func AhdTimeText(year, month, day, hour, minute, second int64) string {
	return ahdPad(year, 4) + "-" + ahdPad(month, 2) + "-" + ahdPad(day, 2) + " " +
		ahdPad(hour, 2) + ":" + ahdPad(minute, 2) + ":" + ahdPad(second, 2)
}

// AhdTimeCivilText formats the visible civil fields without appending an offset.
func AhdTimeCivilText(value AhdCivilTime) string {
	return AhdTimeText(value.Year, value.Month, value.Day, value.Hour, value.Minute, value.Second)
}

func ahdPad(value int64, width int) string {
	text := strconv.FormatInt(value, 10)
	for len(text) < width {
		text = "0" + text
	}
	return text
}

// AhdDurationSeconds converts a millisecond count to fractional seconds.
func AhdDurationSeconds(milliseconds int64) float64 { return float64(milliseconds) / 1000.0 }

// AhdCalendarIsLeapYear applies the Gregorian leap rule.
func AhdCalendarIsLeapYear(year int64) bool {
	ahdRequireRange(year, 1, 9999, "year")
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// AhdCalendarDaysInMonth is the length of one month of one year.
func AhdCalendarDaysInMonth(year, month int64) int64 {
	ahdRequireRange(year, 1, 9999, "year")
	ahdRequireRange(month, 1, 12, "month")
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	}
	if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
		return 29
	}
	return 28
}

// AhdCalendarWeekday is the Monday=1..Sunday=7 weekday of a civil date.
func AhdCalendarWeekday(year, month, day int64) int64 {
	return AhdTimeCivil(year, month, day, 0, 0, 0, 0).Weekday
}

// ---------------------------------------------------------------------------
// Math standard module
// ---------------------------------------------------------------------------

// AhdMathRound rounds to an integral Real. math.Round is explicitly the
// half-away-from-zero operation required by AhdCode, not bankers rounding.
func AhdMathRound(value float64) float64 {
	ahdMathRequireFinite("round", value)
	return math.Round(value)
}

// AhdMathRoundDigits rounds to 0..15 decimal places. When scaling a very large
// finite value would overflow, its float64 spacing is already much wider than
// the requested decimal place, so the value is returned unchanged.
func AhdMathRoundDigits(value float64, digits int64) float64 {
	ahdMathRequireFinite("round", value)
	if digits < 0 || digits > 15 {
		AhdRaiseClass(AhdClassDomainError, "Math.round digits must be in 0..15")
	}
	factor := math.Pow10(int(digits))
	if math.Abs(value) > math.MaxFloat64/factor {
		return value
	}
	return math.Round(value*factor) / factor
}

// AhdMathFloor returns the greatest Int not greater than value.
func AhdMathFloor(value float64) int64 { return ahdMathIntegral("floor", math.Floor(value)) }

// AhdMathCeil returns the least Int not less than value.
func AhdMathCeil(value float64) int64 { return ahdMathIntegral("ceil", math.Ceil(value)) }

func ahdMathIntegral(name string, value float64) int64 {
	ahdMathRequireFinite(name, value)
	limit := math.Ldexp(1, 63)
	if value < -limit || value >= limit {
		AhdRaiseClass(AhdClassOverflowError, "Math."+name+" result does not fit Int")
	}
	return int64(value)
}

// AhdMathSqrt computes the principal square root.
func AhdMathSqrt(value float64) float64 {
	ahdMathRequireFinite("sqrt", value)
	if value < 0 {
		AhdRaiseClass(AhdClassDomainError, "Math.sqrt requires a non-negative value")
	}
	return ahdMathFiniteResult("sqrt", math.Sqrt(value))
}

// AhdMathSin, AhdMathCos, and AhdMathTan use radians.
func AhdMathSin(value float64) float64 {
	ahdMathRequireFinite("sin", value)
	return ahdMathFiniteResult("sin", math.Sin(value))
}

func AhdMathCos(value float64) float64 {
	ahdMathRequireFinite("cos", value)
	return ahdMathFiniteResult("cos", math.Cos(value))
}

func AhdMathTan(value float64) float64 {
	ahdMathRequireFinite("tan", value)
	return ahdMathFiniteResult("tan", math.Tan(value))
}

// AhdMathLog is the natural logarithm; AhdMathLog10 is base ten.
func AhdMathLog(value float64) float64 {
	ahdMathRequireFinite("log", value)
	if value <= 0 {
		AhdRaiseClass(AhdClassDomainError, "Math.log requires a value greater than zero")
	}
	return ahdMathFiniteResult("log", math.Log(value))
}

func AhdMathLog10(value float64) float64 {
	ahdMathRequireFinite("log10", value)
	if value <= 0 {
		AhdRaiseClass(AhdClassDomainError, "Math.log10 requires a value greater than zero")
	}
	return ahdMathFiniteResult("log10", math.Log10(value))
}

// AhdMathExp computes e^value and preserves AhdCode's finite-Real contract.
func AhdMathExp(value float64) float64 {
	ahdMathRequireFinite("exp", value)
	return ahdMathFiniteResult("exp", math.Exp(value))
}

func ahdMathRequireFinite(name string, value float64) {
	if math.IsNaN(value) {
		AhdRaiseClass(AhdClassDomainError, "Math."+name+" received NaN")
	}
	if math.IsInf(value, 0) {
		AhdRaiseClass(AhdClassOverflowError, "Math."+name+" received a non-finite value")
	}
}

func ahdMathFiniteResult(name string, value float64) float64 {
	if math.IsNaN(value) {
		AhdRaiseClass(AhdClassDomainError, "Math."+name+" produced an undefined result")
	}
	if math.IsInf(value, 0) {
		AhdRaiseClass(AhdClassOverflowError, "Math."+name+" result exceeds finite Real range")
	}
	return value
}

const (
	ahdSplitMixIncrement uint64 = 0x9e3779b97f4a7c15
	ahdSplitMixFactor1   uint64 = 0xbf58476d1ce4e5b9
	ahdSplitMixFactor2   uint64 = 0x94d049bb133111eb
)

// ahdMathStateFromEntropy reads exactly one 64-bit seed in little-endian byte
// order. Accepting the reader here keeps startup entropy handling isolated and
// deterministic under test without exposing a public seed API.
func ahdMathStateFromEntropy(reader io.Reader) (uint64, error) {
	var seed [8]byte
	if _, err := io.ReadFull(reader, seed[:]); err != nil {
		return 0, fmt.Errorf("read 8 bytes of OS entropy: %w", err)
	}
	return binary.LittleEndian.Uint64(seed[:]), nil
}

func ahdMathStartupState(reader io.Reader, errorOutput io.Writer, exit func(int)) uint64 {
	state, err := ahdMathStateFromEntropy(reader)
	if err != nil {
		fmt.Fprintln(errorOutput, "AhdCode runtime: Math RNG initialization failed:", err)
		exit(1)
		return 0
	}
	return state
}

var ahdMathGenerator = struct {
	sync.Mutex
	state uint64
}{state: ahdMathStartupState(cryptorand.Reader, os.Stderr, os.Exit)}

// AhdMathSeed resets the one process-wide Math sequence. Converting the signed
// seed to uint64 is the specified two's-complement state mapping.
func AhdMathSeed(seed int64) {
	ahdMathGenerator.Lock()
	ahdMathGenerator.state = uint64(seed)
	ahdMathGenerator.Unlock()
}

// ahdSplitMix64 advances and mixes the generator state using the pinned v0.1
// SplitMix64 transition. The caller holds ahdMathGenerator's lock.
func ahdSplitMix64() uint64 {
	ahdMathGenerator.state += ahdSplitMixIncrement
	value := ahdMathGenerator.state
	value = (value ^ (value >> 30)) * ahdSplitMixFactor1
	value = (value ^ (value >> 27)) * ahdSplitMixFactor2
	return value ^ (value >> 31)
}

// AhdMathRandom constructs [0,1) from the high 53 bits of one raw output.
func AhdMathRandom() float64 {
	ahdMathGenerator.Lock()
	raw := ahdSplitMix64()
	ahdMathGenerator.Unlock()
	return float64(raw>>11) * (1.0 / (1 << 53))
}

// AhdMathRandomInt returns an unbiased value from the inclusive interval. A
// singleton interval consumes no generator output. Rejection sampling avoids
// modulo bias, and a zero uint64 span denotes all 2^64 signed Int values.
func AhdMathRandomInt(minimum, maximum int64) int64 {
	if minimum > maximum {
		AhdRaiseClass(AhdClassDomainError, "Math.randomInt minimum must not exceed maximum")
	}
	if minimum == maximum {
		return minimum
	}
	span := uint64(maximum) - uint64(minimum) + 1
	ahdMathGenerator.Lock()
	defer ahdMathGenerator.Unlock()
	if span == 0 {
		return int64(uint64(minimum) + ahdSplitMix64())
	}
	threshold := -span % span
	for {
		raw := ahdSplitMix64()
		if raw >= threshold {
			return int64(uint64(minimum) + raw%span)
		}
	}
}

// ---------------------------------------------------------------------------
// Numeric Fundamentals: abs and the List reductions
// ---------------------------------------------------------------------------

// AhdAbsInt is the Int magnitude. The minimum Int has no Int magnitude, so it
// reports the ordinary checked-arithmetic overflow rather than wrapping.
func AhdAbsInt(value int64) int64 {
	if value < 0 {
		return AhdIntNegate(value)
	}
	return value
}

// AhdAbsReal is the Real magnitude. It preserves the finite-Real contract and
// turns -0.0 into 0.0.
func AhdAbsReal(value float64) float64 { return math.Abs(value) }

// ahdElement reads one element of a numeric List. A null element is an error
// rather than a zero, so a reduction never invents a value.
func ahdElement[T any](value *T, name string) T {
	if value == nil {
		AhdRaiseClass(AhdClassNullError, name+" does not accept a null List element")
	}
	return *value
}

func ahdRequireElements[T any](list *AhdList[*T], name string) {
	list.require()
	if len(list.items) == 0 {
		AhdRaiseClass(AhdClassDomainError, name+" requires a non-empty List")
	}
}

func ahdRequireElementsNonNull[T any](list *AhdList[T], name string) {
	list.require()
	if len(list.items) == 0 {
		AhdRaiseClass(AhdClassDomainError, name+" requires a non-empty List")
	}
}

// AhdSumInt adds every element of a List<Int> with checked Int arithmetic. An
// empty List sums to the additive identity 0, and the List is not modified.
func AhdSumInt(list *AhdList[*int64]) int64 {
	list.require()
	total := int64(0)
	for _, item := range list.items {
		total = AhdIntAdd(total, ahdElement(item, "sum"))
	}
	return total
}

// AhdSumReal adds every element of a List<Real> with checked Real arithmetic,
// so a non-finite total is an error rather than Inf or NaN. An empty List sums
// to 0.0, and the List is not modified.
func AhdSumReal(list *AhdList[*float64]) float64 {
	list.require()
	total := 0.0
	for _, item := range list.items {
		total = AhdRealAdd(total, ahdElement(item, "sum"))
	}
	return total
}

// AhdMinInt is the least element of a non-empty List<Int>.
func AhdMinInt(list *AhdList[*int64]) int64 {
	ahdRequireElements(list, "min")
	result := ahdElement(list.items[0], "min")
	for _, item := range list.items[1:] {
		if value := ahdElement(item, "min"); value < result {
			result = value
		}
	}
	return result
}

// AhdMaxInt is the greatest element of a non-empty List<Int>.
func AhdMaxInt(list *AhdList[*int64]) int64 {
	ahdRequireElements(list, "max")
	result := ahdElement(list.items[0], "max")
	for _, item := range list.items[1:] {
		if value := ahdElement(item, "max"); value > result {
			result = value
		}
	}
	return result
}

// AhdMinReal is the least element of a non-empty List<Real>.
func AhdMinReal(list *AhdList[*float64]) float64 {
	ahdRequireElements(list, "min")
	result := ahdElement(list.items[0], "min")
	for _, item := range list.items[1:] {
		if value := ahdElement(item, "min"); value < result {
			result = value
		}
	}
	return result
}

// AhdMaxReal is the greatest element of a non-empty List<Real>.
func AhdMaxReal(list *AhdList[*float64]) float64 {
	ahdRequireElements(list, "max")
	result := ahdElement(list.items[0], "max")
	for _, item := range list.items[1:] {
		if value := ahdElement(item, "max"); value > result {
			result = value
		}
	}
	return result
}

// AhdSumIntNonNull is AhdSumInt for a List<Int>, whose elements are never
// null.
func AhdSumIntNonNull(list *AhdList[int64]) int64 {
	list.require()
	total := int64(0)
	for _, item := range list.items {
		total = AhdIntAdd(total, item)
	}
	return total
}

// AhdSumRealNonNull is AhdSumReal for a List<Real>.
func AhdSumRealNonNull(list *AhdList[float64]) float64 {
	list.require()
	total := 0.0
	for _, item := range list.items {
		total = AhdRealAdd(total, item)
	}
	return total
}

// AhdMinIntNonNull is AhdMinInt for a List<Int>.
func AhdMinIntNonNull(list *AhdList[int64]) int64 {
	ahdRequireElementsNonNull(list, "min")
	result := list.items[0]
	for _, item := range list.items[1:] {
		if item < result {
			result = item
		}
	}
	return result
}

// AhdMaxIntNonNull is AhdMaxInt for a List<Int>.
func AhdMaxIntNonNull(list *AhdList[int64]) int64 {
	ahdRequireElementsNonNull(list, "max")
	result := list.items[0]
	for _, item := range list.items[1:] {
		if item > result {
			result = item
		}
	}
	return result
}

// AhdMinRealNonNull is AhdMinReal for a List<Real>.
func AhdMinRealNonNull(list *AhdList[float64]) float64 {
	ahdRequireElementsNonNull(list, "min")
	result := list.items[0]
	for _, item := range list.items[1:] {
		if item < result {
			result = item
		}
	}
	return result
}

// AhdMaxRealNonNull is AhdMaxReal for a List<Real>.
func AhdMaxRealNonNull(list *AhdList[float64]) float64 {
	ahdRequireElementsNonNull(list, "max")
	result := list.items[0]
	for _, item := range list.items[1:] {
		if item > result {
			result = item
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Pair: reference semantics with insertion order
// ---------------------------------------------------------------------------

// AhdPair is the pointer-backed runtime representation of Pair<K, V>. Key
// order is insertion order; updating an existing key keeps its position.
type AhdPair[K comparable, V any] struct {
	keys   []K
	values map[K]V
	frozen bool
	id     int64
}

// AhdIdentity lazily assigns and then returns this Pair's stable identity
// number.
func (pair *AhdPair[K, V]) AhdIdentity() int64 {
	if pair.id == 0 {
		pair.id = ahdNextIdentity()
	}
	return pair.id
}

// AhdFreezeGraph deep-freezes this Pair and every reachable key and value.
func (pair *AhdPair[K, V]) AhdFreezeGraph(visited map[AhdFreezable]bool) {
	if pair == nil || !AhdEnterFreeze(pair, visited) {
		return
	}
	pair.require()
	pair.frozen = true
	for _, key := range pair.keys {
		AhdFreezeChild(key, visited)
		AhdFreezeChild(pair.values[key], visited)
	}
}

func (pair *AhdPair[K, V]) requireMutable() {
	pair.require()
	if pair.frozen {
		ahdRejectMutation()
	}
}

// AhdNewPair builds an empty Pair.
func AhdNewPair[K comparable, V any]() *AhdPair[K, V] {
	return &AhdPair[K, V]{values: make(map[K]V)}
}

func (pair *AhdPair[K, V]) require() {
	if pair == nil {
		AhdRaiseClass(AhdClassNullError, "Pair value is null")
	}
	if pair.values == nil {
		pair.values = make(map[K]V)
	}
}

// Len reports the entry count.
func (pair *AhdPair[K, V]) Len() int64 {
	pair.require()
	return int64(len(pair.keys))
}

// Set inserts or updates a key without moving an existing key.
func (pair *AhdPair[K, V]) Set(key K, value V) {
	pair.requireMutable()
	if _, exists := pair.values[key]; !exists {
		pair.keys = append(pair.keys, key)
	}
	pair.values[key] = value
}

// Get reads a key; a missing key is a KeyError.
func (pair *AhdPair[K, V]) Get(key K) V {
	pair.require()
	value, exists := pair.values[key]
	if !exists {
		AhdRaiseClass(AhdClassKeyError, "Pair has no key "+ahdKeyText(key))
	}
	return value
}

// Has reports key membership.
func (pair *AhdPair[K, V]) Has(key K) bool {
	pair.require()
	_, exists := pair.values[key]
	return exists
}

// Eject removes one key and its value, mutating the Pair in place. A missing
// key is a KeyError. Re-adding an ejected key appends it as a new final entry.
func (pair *AhdPair[K, V]) Eject(key K) {
	pair.requireMutable()
	if _, exists := pair.values[key]; !exists {
		AhdRaiseClass(AhdClassKeyError, "Pair has no key "+ahdKeyText(key))
	}
	pair.Remove(key)
}

// Remove deletes a key, keeping the order of the remaining keys.
func (pair *AhdPair[K, V]) Remove(key K) bool {
	pair.requireMutable()
	if _, exists := pair.values[key]; !exists {
		return false
	}
	delete(pair.values, key)
	for index, existing := range pair.keys {
		if existing == key {
			pair.keys = append(pair.keys[:index], pair.keys[index+1:]...)
			break
		}
	}
	return true
}

// Clear empties the Pair in place, preserving object identity.
func (pair *AhdPair[K, V]) Clear() {
	pair.requireMutable()
	pair.keys = nil
	pair.values = make(map[K]V)
}

// Keys returns the insertion-order iteration snapshot taken at loop entry.
func (pair *AhdPair[K, V]) Keys() []K {
	pair.require()
	result := make([]K, len(pair.keys))
	copy(result, pair.keys)
	return result
}

// AhdPairEqual implements deep Pair value equality; order is not significant.
func AhdPairEqual[K comparable, V any](left, right *AhdPair[K, V], equal func(V, V) bool) bool {
	if left == nil || right == nil {
		return left == right
	}
	left.require()
	right.require()
	if len(left.keys) != len(right.keys) {
		return false
	}
	for key, value := range left.values {
		other, exists := right.values[key]
		if !exists || !equal(value, other) {
			return false
		}
	}
	return true
}

func ahdKeyText(key any) string {
	switch typed := key.(type) {
	case string:
		return strconv.Quote(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return fmt.Sprint(key)
	}
}

// ---------------------------------------------------------------------------
// Canonical str rendering
// ---------------------------------------------------------------------------

// AhdStrInt renders an Int in base-10 decimal text.
func AhdStrInt(value int64) string { return strconv.FormatInt(value, 10) }

// AhdStrBool renders a Bool.
func AhdStrBool(value bool) string { return strconv.FormatBool(value) }

// AhdStrString renders a String as itself.
func AhdStrString(value string) string { return value }

// AhdStrQuoted renders a String nested inside a collection.
func AhdStrQuoted(value string) string { return ahdQuote(value) }

// AhdStrReal renders a Real with locale-independent, shortest round-trip text.
// An integral Real keeps a trailing .0 and negative zero is preserved.
func AhdStrReal(value float64) string {
	if math.IsNaN(value) {
		AhdRaiseClass(AhdClassDomainError, "Real value is not a number")
	}
	if math.IsInf(value, 0) {
		AhdRaiseClass(AhdClassOverflowError, "Real value is not finite")
	}
	return ahdFormatReal(value)
}

func AhdStrComplex(value complex128) string {
	realPart, imaginary := real(value), imag(value)
	if math.IsNaN(realPart) || math.IsNaN(imaginary) {
		AhdRaiseClass(AhdClassDomainError, "Complex component is not a number")
	}
	if math.IsInf(realPart, 0) || math.IsInf(imaginary, 0) {
		AhdRaiseClass(AhdClassOverflowError, "Complex component is not finite")
	}
	sign := "+"
	if math.Signbit(imaginary) {
		sign = "-"
		imaginary = -imaginary
	}
	return ahdFormatReal(realPart) + sign + ahdFormatReal(imaginary) + "I"
}

// ahdFormatReal renders the shortest round-trip decimal text for a finite
// Real. Fixed notation is used for ordinary magnitudes; scientific notation
// with a lowercase e is used only when a fixed rendering would be unwieldy.
func ahdFormatReal(value float64) string {
	scientific := strconv.FormatFloat(value, 'e', -1, 64)
	exponent := 0
	if marker := strings.IndexByte(scientific, 'e'); marker >= 0 {
		parsed, err := strconv.Atoi(scientific[marker+1:])
		if err == nil {
			exponent = parsed
		}
	}
	if exponent < -4 || exponent >= 21 {
		return scientific
	}
	text := strconv.FormatFloat(value, 'f', -1, 64)
	if !strings.Contains(text, ".") {
		text += ".0"
	}
	return text
}

// AhdStrNull lifts a renderer over a nullable slot.
func AhdStrNull[T any](render func(T) string) func(*T) string {
	return func(value *T) string {
		if value == nil {
			return "null"
		}
		return render(*value)
	}
}

// AhdStrRefInstance renders a nullable Class reference in canonical text.
func AhdStrRefInstance[T AhdInstance](value T) string {
	if any(value) == nil {
		return "null"
	}
	return AhdStrInstance(value)
}

// AhdStrFunction renders a named Function value.
func AhdStrFunction(name string) string { return "<Function " + name + ">" }

// AhdStrList renders a List with a canonical literal-like representation.
func AhdStrList[T any](render func(T) string) func(*AhdList[T]) string {
	return func(list *AhdList[T]) string {
		if list == nil {
			return "null"
		}
		parts := make([]string, len(list.items))
		for index, item := range list.items {
			parts[index] = render(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
}

// AhdStrPair renders a Pair in insertion order.
func AhdStrPair[K comparable, V any](renderKey func(K) string, renderValue func(V) string) func(*AhdPair[K, V]) string {
	return func(pair *AhdPair[K, V]) string {
		if pair == nil {
			return "null"
		}
		pair.require()
		parts := make([]string, 0, len(pair.keys))
		for _, key := range pair.keys {
			parts = append(parts, renderKey(key)+": "+renderValue(pair.values[key]))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
}

func ahdQuote(value string) string {
	var out strings.Builder
	out.WriteByte('"')
	for _, item := range value {
		switch item {
		case '"':
			out.WriteString("\\\"")
		case '\\':
			out.WriteString("\\\\")
		case '\n':
			out.WriteString("\\n")
		case '\r':
			out.WriteString("\\r")
		case '\t':
			out.WriteString("\\t")
		default:
			out.WriteRune(item)
		}
	}
	out.WriteByte('"')
	return out.String()
}

// ---------------------------------------------------------------------------
// Equality helpers
// ---------------------------------------------------------------------------

// AhdEqInt compares two Int values.
func AhdEqInt(left, right int64) bool { return left == right }

// AhdEqReal compares two Real values.
func AhdEqReal(left, right float64) bool { return left == right }

func AhdEqComplex(left, right complex128) bool { return left == right }

// AhdEqString compares two String values.
func AhdEqString(left, right string) bool { return left == right }

// AhdEqBool compares two Bool values.
func AhdEqBool(left, right bool) bool { return left == right }

// AhdEqNull lifts an equality over a nullable slot.
func AhdEqNull[T any](equal func(T, T) bool) func(*T, *T) bool {
	return func(left, right *T) bool {
		if left == nil || right == nil {
			return left == nil && right == nil
		}
		return equal(*left, *right)
	}
}

// AhdEqRef compares Class reference identity.
func AhdEqRef[T comparable]() func(T, T) bool {
	return func(left, right T) bool { return left == right }
}

// AhdEqList lifts an element equality to deep List equality.
func AhdEqList[T any](equal func(T, T) bool) func(*AhdList[T], *AhdList[T]) bool {
	return func(left, right *AhdList[T]) bool { return AhdListEqual(left, right, equal) }
}

// AhdEqPair lifts a value equality to deep Pair equality.
func AhdEqPair[K comparable, V any](equal func(V, V) bool) func(*AhdPair[K, V], *AhdPair[K, V]) bool {
	return func(left, right *AhdPair[K, V]) bool { return AhdPairEqual(left, right, equal) }
}

// AhdSameDifferent evaluates both operands of a statically type-distinct same.
func AhdSameDifferent[A any, B any](left A, right B) bool { return false }

// AhdConstBool evaluates a value for its effects and yields a statically
// resolved Bool result, such as Class member existence.
func AhdConstBool[T any](value T, result bool) bool { return result }

// AhdStringContains implements String substring membership.
func AhdStringContains(haystack, needle string) bool { return strings.Contains(haystack, needle) }

// AhdBuildPair builds a Pair literal in written key order.
func AhdBuildPair[K comparable, V any](keys []K, values []V) *AhdPair[K, V] {
	result := AhdNewPair[K, V]()
	for index, key := range keys {
		result.Set(key, values[index])
	}
	return result
}

// AhdUnreachable reports a Function that ended without returning a value. The
// frontend proves definite return, so reaching it indicates a compiler defect
// rather than a user program error.
func AhdUnreachable[T any]() T {
	panic("ahdcode: Function ended without returning a value")
}

// ---------------------------------------------------------------------------
// Data standard module
//
// A Table is an ordered list of column names plus one List of String cells per
// row. Every cell is a String: Data is a table structure layer, not a typed
// value system, so a program converts explicitly with int(...) / real(...).
//
// Every operation here is pure. Values arriving from AhdCode are copied on the
// way in and every result is freshly built, so a Table never shares mutable
// storage with the program that made it or with a snapshot it handed out.
// ---------------------------------------------------------------------------

// AhdTable is the runtime interchange shape of one Table's storage. The
// generated constructor spreads it into the Class's two hidden fields.
type AhdTable struct {
	Columns *AhdList[string]
	Cells   *AhdList[*AhdList[string]]
}

// ahdTableOf rebuilds the working representation from the stored Lists.
func ahdTableOf(value AhdTable) ([]string, [][]string) {
	var columns []string
	if value.Columns != nil {
		columns = value.Columns.Snapshot()
	}
	var cells [][]string
	if value.Cells != nil {
		for _, row := range value.Cells.Snapshot() {
			if row == nil {
				AhdRaiseClass(AhdClassNullError, "table row is null")
			}
			cells = append(cells, row.Snapshot())
		}
	}
	return columns, cells
}

// ahdTableValue packages a validated schema and grid as stored Lists.
func ahdTableValue(columns []string, cells [][]string) AhdTable {
	rows := make([]*AhdList[string], len(cells))
	for index, row := range cells {
		rows[index] = AhdNewList(row...)
	}
	return AhdTable{Columns: AhdNewList(columns...), Cells: AhdNewList(rows...)}
}

// ahdDataRequireSchema enforces the column rules every Table shares: a column
// name is non-empty, and no name repeats.
func ahdDataRequireSchema(class *AhdClass, columns []string) {
	seen := make(map[string]bool, len(columns))
	for _, name := range columns {
		if name == "" {
			AhdRaiseClass(class, "column name is empty")
		}
		if seen[name] {
			AhdRaiseClass(class, "duplicate column "+strconv.Quote(name))
		}
		seen[name] = true
	}
}

// ahdDataIndex is the column-name lookup every per-row operation shares, so a
// name is resolved once instead of scanning the schema for every row.
func ahdDataIndex(columns []string) map[string]int {
	index := make(map[string]int, len(columns))
	for position, name := range columns {
		index[name] = position
	}
	return index
}

func ahdDataColumnPosition(class *AhdClass, columns []string, name string) int {
	for position, column := range columns {
		if column == name {
			return position
		}
	}
	AhdRaiseClass(class, "Table has no column "+strconv.Quote(name))
	return -1
}

func ahdDataRequireWidth(class *AhdClass, columns []string, cells [][]string) {
	for number, row := range cells {
		if len(row) != len(columns) {
			AhdRaiseClass(class, "row "+strconv.FormatInt(int64(number), 10)+" has "+
				strconv.FormatInt(int64(len(row)), 10)+" cell(s); the table has "+
				strconv.FormatInt(int64(len(columns)), 10)+" column(s)")
		}
	}
}

// AhdDataFromRows builds a Table from an explicit schema and grid.
func AhdDataFromRows(class *AhdClass, columns *AhdList[string], rows *AhdList[*AhdList[string]]) AhdTable {
	if columns == nil || rows == nil {
		AhdRaiseClass(AhdClassNullError, "List value is null")
	}
	names := columns.Snapshot()
	ahdDataRequireSchema(class, names)
	var cells [][]string
	for _, row := range rows.Snapshot() {
		if row == nil {
			AhdRaiseClass(AhdClassNullError, "table row is null")
		}
		cells = append(cells, row.Snapshot())
	}
	ahdDataRequireWidth(class, names, cells)
	return ahdTableValue(names, cells)
}

// AhdDataFromRecords builds a Table from records. The first record fixes the
// column order; every later record must carry exactly the same key set, in any
// insertion order, and its values are copied into canonical order.
func AhdDataFromRecords(class *AhdClass, records *AhdList[*AhdPair[string, string]]) AhdTable {
	if records == nil {
		AhdRaiseClass(AhdClassNullError, "List value is null")
	}
	items := records.Snapshot()
	if len(items) == 0 {
		// No record means no schema to infer; an empty Table is the only
		// honest answer, rather than inventing column names.
		return ahdTableValue(nil, nil)
	}
	if items[0] == nil {
		AhdRaiseClass(AhdClassNullError, "record is null")
	}
	columns := items[0].Keys()
	ahdDataRequireSchema(class, columns)
	cells := make([][]string, 0, len(items))
	for number, record := range items {
		if record == nil {
			AhdRaiseClass(AhdClassNullError, "record is null")
		}
		if record.Len() != int64(len(columns)) {
			AhdRaiseClass(class, "record "+strconv.FormatInt(int64(number), 10)+" has "+
				strconv.FormatInt(record.Len(), 10)+" key(s); the first record has "+
				strconv.FormatInt(int64(len(columns)), 10))
		}
		row := make([]string, len(columns))
		for position, name := range columns {
			if !record.Has(name) {
				AhdRaiseClass(class, "record "+strconv.FormatInt(int64(number), 10)+
					" has no key "+strconv.Quote(name))
			}
			row[position] = record.Get(name)
		}
		cells = append(cells, row)
	}
	return ahdTableValue(columns, cells)
}

// ahdDataFromGrid turns parsed CSV rows into a Table. The first row is the
// header, so unlike CSV.parseRecords a header-only document keeps its schema.
func ahdDataFromGrid(class *AhdClass, grid [][]string) AhdTable {
	if len(grid) == 0 {
		return ahdTableValue(nil, nil)
	}
	columns := grid[0]
	ahdDataRequireSchema(class, columns)
	cells := grid[1:]
	ahdDataRequireWidth(class, columns, cells)
	return ahdTableValue(columns, cells)
}

// AhdDataFromCSV and AhdDataReadCSV reuse the CSV module's reader, so Data
// never defines a second CSV grammar. CSV syntax failures stay CSVError and
// filesystem failures stay FileError.
func AhdDataFromCSV(dataClass, csvClass *AhdClass, text, delimiter string) AhdTable {
	return ahdDataFromGrid(dataClass, ahdCSVRows(csvClass, text, delimiter))
}

func AhdDataReadCSV(dataClass, csvClass, fileClass *AhdClass, path, delimiter string) AhdTable {
	content, err := os.ReadFile(path)
	if err != nil {
		ahdFileFailure(fileClass, "read", path, err)
	}
	return ahdDataFromGrid(dataClass, ahdCSVRows(csvClass, string(content), delimiter))
}

// AhdDataRowCount and AhdDataColumnCount report the table's shape.
func AhdDataRowCount(value AhdTable) int64 {
	_, cells := ahdTableOf(value)
	return int64(len(cells))
}

func AhdDataColumnCount(value AhdTable) int64 {
	columns, _ := ahdTableOf(value)
	return int64(len(columns))
}

// AhdDataColumns returns a new List, so mutating it cannot reach the Table.
func AhdDataColumns(value AhdTable) *AhdList[string] {
	columns, _ := ahdTableOf(value)
	return AhdNewList(columns...)
}

func ahdDataRecord(columns, row []string) *AhdPair[string, string] {
	record := AhdNewPair[string, string]()
	for position, name := range columns {
		record.Set(name, row[position])
	}
	return record
}

// AhdDataRows returns a new List of new Pair snapshots.
func AhdDataRows(value AhdTable) *AhdList[*AhdPair[string, string]] {
	columns, cells := ahdTableOf(value)
	records := make([]*AhdPair[string, string], len(cells))
	for index, row := range cells {
		records[index] = ahdDataRecord(columns, row)
	}
	return AhdNewList(records...)
}

// AhdDataRow returns one row snapshot, using the ordinary List index rules so
// a negative index counts from the end and an invalid one is an IndexError.
func AhdDataRow(value AhdTable, index int64) *AhdPair[string, string] {
	columns, cells := ahdTableOf(value)
	position := index
	if position < 0 {
		position += int64(len(cells))
	}
	if position < 0 || position >= int64(len(cells)) {
		AhdRaiseClass(AhdClassIndexError, "row index "+strconv.FormatInt(index, 10)+" is out of range")
	}
	return ahdDataRecord(columns, cells[position])
}

// AhdDataColumn returns one column's cells, in row order.
func AhdDataColumn(class *AhdClass, value AhdTable, name string) *AhdList[string] {
	columns, cells := ahdTableOf(value)
	position := ahdDataColumnPosition(class, columns, name)
	result := make([]string, len(cells))
	for index, row := range cells {
		result[index] = row[position]
	}
	return AhdNewList(result...)
}

// AhdDataHead and AhdDataTail keep the first or last rows, preserving order
// and the full schema even when no row survives.
func AhdDataHead(class *AhdClass, value AhdTable, count int64) AhdTable {
	columns, cells := ahdTableOf(value)
	if count < 0 {
		AhdRaiseClass(class, "head requires a non-negative row count")
	}
	if count > int64(len(cells)) {
		count = int64(len(cells))
	}
	return ahdTableValue(columns, cells[:count])
}

func AhdDataTail(class *AhdClass, value AhdTable, count int64) AhdTable {
	columns, cells := ahdTableOf(value)
	if count < 0 {
		AhdRaiseClass(class, "tail requires a non-negative row count")
	}
	if count > int64(len(cells)) {
		count = int64(len(cells))
	}
	return ahdTableValue(columns, cells[int64(len(cells))-count:])
}

// AhdDataSelect keeps the requested columns, in the requested order.
func AhdDataSelect(class *AhdClass, value AhdTable, requested *AhdList[string]) AhdTable {
	if requested == nil {
		AhdRaiseClass(AhdClassNullError, "List value is null")
	}
	columns, cells := ahdTableOf(value)
	names := requested.Snapshot()
	seen := make(map[string]bool, len(names))
	positions := make([]int, len(names))
	for index, name := range names {
		if seen[name] {
			AhdRaiseClass(class, "duplicate column "+strconv.Quote(name)+" in select")
		}
		seen[name] = true
		positions[index] = ahdDataColumnPosition(class, columns, name)
	}
	result := make([][]string, len(cells))
	for index, row := range cells {
		selected := make([]string, len(positions))
		for target, position := range positions {
			selected[target] = row[position]
		}
		result[index] = selected
	}
	return ahdTableValue(names, result)
}

// AhdDataDrop removes the requested columns, keeping the original order of the
// columns that remain.
func AhdDataDrop(class *AhdClass, value AhdTable, requested *AhdList[string]) AhdTable {
	if requested == nil {
		AhdRaiseClass(AhdClassNullError, "List value is null")
	}
	columns, cells := ahdTableOf(value)
	removed := make(map[string]bool, requested.Len())
	for _, name := range requested.Snapshot() {
		if removed[name] {
			AhdRaiseClass(class, "duplicate column "+strconv.Quote(name)+" in drop")
		}
		ahdDataColumnPosition(class, columns, name)
		removed[name] = true
	}
	var kept []string
	var positions []int
	for position, name := range columns {
		if !removed[name] {
			kept = append(kept, name)
			positions = append(positions, position)
		}
	}
	result := make([][]string, len(cells))
	for index, row := range cells {
		remaining := make([]string, len(positions))
		for target, position := range positions {
			remaining[target] = row[position]
		}
		result[index] = remaining
	}
	return ahdTableValue(kept, result)
}

// AhdDataRename renames one column in place, preserving its position.
func AhdDataRename(class *AhdClass, value AhdTable, oldName, newName string) AhdTable {
	columns, cells := ahdTableOf(value)
	position := ahdDataColumnPosition(class, columns, oldName)
	if newName == "" {
		AhdRaiseClass(class, "column name is empty")
	}
	if newName != oldName {
		for _, name := range columns {
			if name == newName {
				AhdRaiseClass(class, "duplicate column "+strconv.Quote(newName))
			}
		}
	}
	renamed := append([]string(nil), columns...)
	renamed[position] = newName
	return ahdTableValue(renamed, cells)
}

// AhdDataReverse reverses row order and leaves the schema untouched.
func AhdDataReverse(value AhdTable) AhdTable {
	columns, cells := ahdTableOf(value)
	result := make([][]string, len(cells))
	for index, row := range cells {
		result[len(cells)-1-index] = row
	}
	return ahdTableValue(columns, result)
}

// AhdDataFilter keeps the rows the predicate accepts, in source order. The
// predicate sees a fresh row snapshot, so mutating it cannot reach the Table.
func AhdDataFilter(value AhdTable, keep func(*AhdPair[string, string]) *bool) AhdTable {
	columns, cells := ahdTableOf(value)
	var result [][]string
	for _, row := range cells {
		if ahdPredicate(keep(ahdDataRecord(columns, row))) {
			result = append(result, row)
		}
	}
	return ahdTableValue(columns, result)
}

// AhdDataSortColumn orders rows by one column's text, stably and ascending.
func AhdDataSortColumn(class *AhdClass, value AhdTable, name string) AhdTable {
	columns, cells := ahdTableOf(value)
	position := ahdDataColumnPosition(class, columns, name)
	order := make([]int, len(cells))
	for index := range order {
		order[index] = index
	}
	sort.SliceStable(order, func(left, right int) bool {
		return cells[order[left]][position] < cells[order[right]][position]
	})
	result := make([][]string, len(cells))
	for index, original := range order {
		result[index] = cells[original]
	}
	return ahdTableValue(columns, result)
}

// ahdDataSortByKey is the shared keyed ordering. The key Function runs exactly
// once per row, before any comparison, matching List's keyed sort.
func ahdDataSortByKey[K int64 | float64 | string](value AhdTable, key func(*AhdPair[string, string]) *K) AhdTable {
	columns, cells := ahdTableOf(value)
	keys := make([]K, len(cells))
	for index, row := range cells {
		computed := key(ahdDataRecord(columns, row))
		if computed == nil {
			AhdRaiseClass(AhdClassNullError, "sort key Function returned null")
		}
		keys[index] = *computed
	}
	order := make([]int, len(cells))
	for index := range order {
		order[index] = index
	}
	sort.SliceStable(order, func(left, right int) bool { return keys[order[left]] < keys[order[right]] })
	result := make([][]string, len(cells))
	for index, original := range order {
		result[index] = cells[original]
	}
	return ahdTableValue(columns, result)
}

// AhdDataSortKeyInt, AhdDataSortKeyReal, and AhdDataSortKeyString order rows by
// an Int, Real, or String key.
func AhdDataSortKeyInt(value AhdTable, key func(*AhdPair[string, string]) *int64) AhdTable {
	return ahdDataSortByKey(value, key)
}

func AhdDataSortKeyReal(value AhdTable, key func(*AhdPair[string, string]) *float64) AhdTable {
	return ahdDataSortByKey(value, key)
}

func AhdDataSortKeyString(value AhdTable, key func(*AhdPair[string, string]) *string) AhdTable {
	return ahdDataSortByKey(value, key)
}

// AhdDataTransform rewrites one column, leaving its position and every other
// column untouched.
func AhdDataTransform(class *AhdClass, value AhdTable, name string, convert func(string) *string) AhdTable {
	columns, cells := ahdTableOf(value)
	position := ahdDataColumnPosition(class, columns, name)
	result := make([][]string, len(cells))
	for index, row := range cells {
		replaced := append([]string(nil), row...)
		computed := convert(row[position])
		if computed == nil {
			AhdRaiseClass(AhdClassNullError, "transform Function returned null")
		}
		replaced[position] = *computed
		result[index] = replaced
	}
	return ahdTableValue(columns, result)
}

// AhdDataDerive appends a new column built from each complete row.
func AhdDataDerive(class *AhdClass, value AhdTable, name string, build func(*AhdPair[string, string]) *string) AhdTable {
	columns, cells := ahdTableOf(value)
	if name == "" {
		AhdRaiseClass(class, "column name is empty")
	}
	for _, column := range columns {
		if column == name {
			AhdRaiseClass(class, "column "+strconv.Quote(name)+
				" already exists; use transform to rewrite an existing column")
		}
	}
	result := make([][]string, len(cells))
	for index, row := range cells {
		computed := build(ahdDataRecord(columns, row))
		if computed == nil {
			AhdRaiseClass(AhdClassNullError, "derive Function returned null")
		}
		result[index] = append(append([]string(nil), row...), *computed)
	}
	return ahdTableValue(append(append([]string(nil), columns...), name), result)
}

// AhdDataUnique lists one column's distinct cells in first-occurrence order.
func AhdDataUnique(class *AhdClass, value AhdTable, name string) *AhdList[string] {
	columns, cells := ahdTableOf(value)
	position := ahdDataColumnPosition(class, columns, name)
	seen := make(map[string]bool, len(cells))
	var result []string
	for _, row := range cells {
		if !seen[row[position]] {
			seen[row[position]] = true
			result = append(result, row[position])
		}
	}
	return AhdNewList(result...)
}

// AhdDataValueCounts counts one column's cells, keyed in first-occurrence
// order.
func AhdDataValueCounts(class *AhdClass, value AhdTable, name string) *AhdPair[string, int64] {
	columns, cells := ahdTableOf(value)
	position := ahdDataColumnPosition(class, columns, name)
	counts := AhdNewPair[string, int64]()
	for _, row := range cells {
		if counts.Has(row[position]) {
			counts.Set(row[position], counts.Get(row[position])+1)
			continue
		}
		counts.Set(row[position], 1)
	}
	return counts
}

// ahdDataGroups partitions rows by one column, keeping first-occurrence key
// order and source row order inside each group.
func ahdDataGroups(class *AhdClass, value AhdTable, name string) ([]string, map[string][][]string, []string) {
	columns, cells := ahdTableOf(value)
	position := ahdDataColumnPosition(class, columns, name)
	var order []string
	groups := make(map[string][][]string)
	for _, row := range cells {
		key := row[position]
		if _, known := groups[key]; !known {
			order = append(order, key)
		}
		groups[key] = append(groups[key], row)
	}
	return columns, groups, order
}

// AhdDataGroupCount, AhdDataGroupKey, and AhdDataGroupTable let the generated
// code build the Pair<String, Table> without the runtime needing to know the
// generated Table Class type.
func AhdDataGroupKeys(class *AhdClass, value AhdTable, name string) *AhdList[string] {
	_, _, order := ahdDataGroups(class, value, name)
	return AhdNewList(order...)
}

func AhdDataGroupTable(class *AhdClass, value AhdTable, name, key string) AhdTable {
	columns, groups, _ := ahdDataGroups(class, value, name)
	return ahdTableValue(columns, groups[key])
}

// ahdDataGrid renders the header followed by the data rows, which is the shape
// the CSV writer serializes.
func ahdDataGrid(value AhdTable) *AhdList[*AhdList[string]] {
	columns, cells := ahdTableOf(value)
	if len(columns) == 0 && len(cells) == 0 {
		return AhdNewList[*AhdList[string]]()
	}
	rows := make([]*AhdList[string], 0, len(cells)+1)
	rows = append(rows, AhdNewList(columns...))
	for _, row := range cells {
		rows = append(rows, AhdNewList(row...))
	}
	return AhdNewList(rows...)
}

// AhdDataToCSV and AhdDataWriteCSV reuse the CSV module's writer, so quoting,
// delimiters, and line endings match CSV.stringify exactly.
func AhdDataToCSV(csvClass *AhdClass, value AhdTable, delimiter string) string {
	return AhdCSVStringify(csvClass, ahdDataGrid(value), delimiter)
}

func AhdDataWriteCSV(csvClass, fileClass *AhdClass, value AhdTable, path, delimiter string) {
	content := AhdCSVStringify(csvClass, ahdDataGrid(value), delimiter)
	if err := os.WriteFile(path, []byte(content), 0o666); err != nil {
		ahdFileFailure(fileClass, "write", path, err)
	}
}

// ---------------------------------------------------------------------------
// Statistics standard module
//
// Descriptive statistics over typed numeric Lists. Every function reads a
// snapshot, so ordering a List to find a median or quantile never reorders the
// caller's List. A statistic that is mathematically undefined for its input --
// most often because the List is empty -- raises StatisticsError rather than
// returning a placeholder.
// ---------------------------------------------------------------------------

// ahdStatisticsValues snapshots a numeric List, rejecting a null List or a null
// element before any arithmetic runs.
func ahdStatisticsValues[T int64 | float64](list *AhdList[T]) []T {
	if list == nil {
		AhdRaiseClass(AhdClassNullError, "List value is null")
	}
	return append([]T(nil), list.Snapshot()...)
}

func ahdStatisticsRequire(class *AhdClass, count int, statistic string) {
	if count == 0 {
		AhdRaiseClass(class, statistic+" is undefined for an empty List")
	}
}

// ahdStatisticsReal widens one numeric value for an averaging statistic.
func ahdStatisticsReal[T int64 | float64](value T) float64 { return float64(value) }

// AhdStatisticsSum adds every value. The empty sum is the additive identity,
// which is the one total that keeps sum(a) + sum(b) == sum(a ++ b).
func AhdStatisticsSumInt(values *AhdList[int64]) int64 {
	total := int64(0)
	for _, value := range ahdStatisticsValues(values) {
		total = AhdIntAdd(total, value)
	}
	return total
}

func AhdStatisticsSumReal(values *AhdList[float64]) float64 {
	total := float64(0)
	for _, value := range ahdStatisticsValues(values) {
		total = AhdRealAdd(total, value)
	}
	return total
}

func ahdStatisticsMin[T int64 | float64](class *AhdClass, list *AhdList[T]) T {
	values := ahdStatisticsValues(list)
	ahdStatisticsRequire(class, len(values), "min")
	smallest := values[0]
	for _, value := range values[1:] {
		if value < smallest {
			smallest = value
		}
	}
	return smallest
}

func ahdStatisticsMax[T int64 | float64](class *AhdClass, list *AhdList[T]) T {
	values := ahdStatisticsValues(list)
	ahdStatisticsRequire(class, len(values), "max")
	largest := values[0]
	for _, value := range values[1:] {
		if value > largest {
			largest = value
		}
	}
	return largest
}

func AhdStatisticsMinInt(class *AhdClass, values *AhdList[int64]) int64 {
	return ahdStatisticsMin(class, values)
}

func AhdStatisticsMinReal(class *AhdClass, values *AhdList[float64]) float64 {
	return ahdStatisticsMin(class, values)
}

func AhdStatisticsMaxInt(class *AhdClass, values *AhdList[int64]) int64 {
	return ahdStatisticsMax(class, values)
}

func AhdStatisticsMaxReal(class *AhdClass, values *AhdList[float64]) float64 {
	return ahdStatisticsMax(class, values)
}

// AhdStatisticsRange is max - min, using the language's checked arithmetic so an
// Int range that overflows is an error rather than a wrapped value.
func AhdStatisticsRangeInt(class *AhdClass, values *AhdList[int64]) int64 {
	// Checked before delegating, so an empty List reports "range" rather than
	// the name of whichever extreme happened to be computed first.
	ahdStatisticsRequire(class, len(ahdStatisticsValues(values)), "range")
	return AhdIntSubtract(ahdStatisticsMax(class, values), ahdStatisticsMin(class, values))
}

func AhdStatisticsRangeReal(class *AhdClass, values *AhdList[float64]) float64 {
	ahdStatisticsRequire(class, len(ahdStatisticsValues(values)), "range")
	return AhdRealSubtract(ahdStatisticsMax(class, values), ahdStatisticsMin(class, values))
}

// ahdStatisticsMean is the arithmetic mean, always Real: the average of whole
// numbers is generally not whole.
func ahdStatisticsMean[T int64 | float64](class *AhdClass, list *AhdList[T]) float64 {
	values := ahdStatisticsValues(list)
	ahdStatisticsRequire(class, len(values), "mean")
	total := float64(0)
	for _, value := range values {
		total += ahdStatisticsReal(value)
	}
	return ahdStatisticsFinite(class, total/float64(len(values)), "mean")
}

func AhdStatisticsMeanInt(class *AhdClass, values *AhdList[int64]) float64 {
	return ahdStatisticsMean(class, values)
}

func AhdStatisticsMeanReal(class *AhdClass, values *AhdList[float64]) float64 {
	return ahdStatisticsMean(class, values)
}

// ahdStatisticsSorted orders a snapshot, so the caller's List keeps its order.
func ahdStatisticsSorted[T int64 | float64](list *AhdList[T]) []T {
	values := ahdStatisticsValues(list)
	sort.Slice(values, func(left, right int) bool { return values[left] < values[right] })
	return values
}

// ahdStatisticsMedian is the middle value of the ordered data, averaging the two
// middle values when the count is even. It is always Real so the even case needs
// no separate rule.
func ahdStatisticsMedian[T int64 | float64](class *AhdClass, list *AhdList[T]) float64 {
	values := ahdStatisticsSorted(list)
	ahdStatisticsRequire(class, len(values), "median")
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return ahdStatisticsReal(values[middle])
	}
	low, high := ahdStatisticsReal(values[middle-1]), ahdStatisticsReal(values[middle])
	return ahdStatisticsFinite(class, low+(high-low)/2, "median")
}

func AhdStatisticsMedianInt(class *AhdClass, values *AhdList[int64]) float64 {
	return ahdStatisticsMedian(class, values)
}

func AhdStatisticsMedianReal(class *AhdClass, values *AhdList[float64]) float64 {
	return ahdStatisticsMedian(class, values)
}

// ahdStatisticsVariance is the mean squared deviation. sample divides by n-1
// (Bessel's correction) and needs at least two values; the population form
// divides by n and needs at least one.
func ahdStatisticsVariance[T int64 | float64](class *AhdClass, list *AhdList[T], sample bool) float64 {
	values := ahdStatisticsValues(list)
	statistic := "variance"
	if sample {
		statistic = "sampleVariance"
		if len(values) < 2 {
			AhdRaiseClass(class, "sampleVariance requires at least two values")
		}
	}
	ahdStatisticsRequire(class, len(values), statistic)
	mean := float64(0)
	for _, value := range values {
		mean += ahdStatisticsReal(value)
	}
	mean /= float64(len(values))
	total := float64(0)
	for _, value := range values {
		deviation := ahdStatisticsReal(value) - mean
		total += deviation * deviation
	}
	divisor := float64(len(values))
	if sample {
		divisor = float64(len(values) - 1)
	}
	return ahdStatisticsFinite(class, total/divisor, statistic)
}

func AhdStatisticsVarianceInt(class *AhdClass, values *AhdList[int64]) float64 {
	return ahdStatisticsVariance(class, values, false)
}

func AhdStatisticsVarianceReal(class *AhdClass, values *AhdList[float64]) float64 {
	return ahdStatisticsVariance(class, values, false)
}

func AhdStatisticsSampleVarianceInt(class *AhdClass, values *AhdList[int64]) float64 {
	return ahdStatisticsVariance(class, values, true)
}

func AhdStatisticsSampleVarianceReal(class *AhdClass, values *AhdList[float64]) float64 {
	return ahdStatisticsVariance(class, values, true)
}

func AhdStatisticsStdDevInt(class *AhdClass, values *AhdList[int64]) float64 {
	return ahdStatisticsFinite(class, math.Sqrt(ahdStatisticsVariance(class, values, false)), "stdDev")
}

func AhdStatisticsStdDevReal(class *AhdClass, values *AhdList[float64]) float64 {
	return ahdStatisticsFinite(class, math.Sqrt(ahdStatisticsVariance(class, values, false)), "stdDev")
}

func AhdStatisticsSampleStdDevInt(class *AhdClass, values *AhdList[int64]) float64 {
	return ahdStatisticsFinite(class, math.Sqrt(ahdStatisticsVariance(class, values, true)), "sampleStdDev")
}

func AhdStatisticsSampleStdDevReal(class *AhdClass, values *AhdList[float64]) float64 {
	return ahdStatisticsFinite(class, math.Sqrt(ahdStatisticsVariance(class, values, true)), "sampleStdDev")
}

// ahdStatisticsMode is the most frequent value. A tie is broken by first
// occurrence in the input, so the answer never depends on map iteration order.
func ahdStatisticsMode[T int64 | float64](class *AhdClass, list *AhdList[T]) T {
	values := ahdStatisticsValues(list)
	ahdStatisticsRequire(class, len(values), "mode")
	counts := make(map[T]int, len(values))
	for _, value := range values {
		counts[value]++
	}
	best := values[0]
	for _, value := range values {
		if counts[value] > counts[best] {
			best = value
		}
	}
	return best
}

func AhdStatisticsModeInt(class *AhdClass, values *AhdList[int64]) int64 {
	return ahdStatisticsMode(class, values)
}

func AhdStatisticsModeReal(class *AhdClass, values *AhdList[float64]) float64 {
	return ahdStatisticsMode(class, values)
}

// ahdStatisticsQuantile is the linear-interpolation quantile of the ordered
// data: position = probability * (count - 1), then interpolate between the two
// neighbouring order statistics. probability 0 is the minimum and 1 is the
// maximum, a single value is its own quantile, and a probability outside
// 0.0..1.0 is an error rather than a clamp.
func ahdStatisticsQuantile[T int64 | float64](class *AhdClass, list *AhdList[T], probability float64) float64 {
	values := ahdStatisticsSorted(list)
	ahdStatisticsRequire(class, len(values), "quantile")
	if !(probability >= 0 && probability <= 1) {
		AhdRaiseClass(class, "quantile probability must be between 0.0 and 1.0")
	}
	if len(values) == 1 {
		return ahdStatisticsReal(values[0])
	}
	position := probability * float64(len(values)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return ahdStatisticsReal(values[lower])
	}
	weight := position - float64(lower)
	low, high := ahdStatisticsReal(values[lower]), ahdStatisticsReal(values[upper])
	return ahdStatisticsFinite(class, low+(high-low)*weight, "quantile")
}

func AhdStatisticsQuantileInt(class *AhdClass, values *AhdList[int64], probability float64) float64 {
	return ahdStatisticsQuantile(class, values, probability)
}

func AhdStatisticsQuantileReal(class *AhdClass, values *AhdList[float64], probability float64) float64 {
	return ahdStatisticsQuantile(class, values, probability)
}

// ahdStatisticsFinite keeps the language's finite-Real contract: a statistic
// never hands back NaN or an infinity, matching how ordinary Real arithmetic
// reports an undefined or out-of-range result.
func ahdStatisticsFinite(class *AhdClass, value float64, statistic string) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		AhdRaiseClass(class, statistic+" has no finite Real value for this input")
	}
	return value
}

// AhdDataPivotCount is a strict count cross-tabulation: one row per distinct
// value of the row column, one generated column per distinct value of the
// column column, and each cell the number of source rows in that combination.
//
// It is deliberately not a general pivot. There is no aggregation callback, no
// value column, and no missing-data model: an absent combination counts zero,
// and every cell stays a String like every other Table cell.
func AhdDataPivotCount(class *AhdClass, value AhdTable, rowName, columnName string) AhdTable {
	columns, cells := ahdTableOf(value)
	rowPosition := ahdDataColumnPosition(class, columns, rowName)
	columnPosition := ahdDataColumnPosition(class, columns, columnName)
	if rowName == columnName {
		AhdRaiseClass(class, "pivotCount needs two different columns; received "+
			strconv.Quote(rowName)+" twice")
	}
	// First-occurrence order for both axes, matching groupBy and valueCounts,
	// so the result never depends on map iteration order.
	var rowOrder, columnOrder []string
	seenRow := make(map[string]bool)
	seenColumn := make(map[string]bool)
	counts := make(map[string]map[string]int64)
	for _, row := range cells {
		rowKey, columnKey := row[rowPosition], row[columnPosition]
		if !seenRow[rowKey] {
			seenRow[rowKey] = true
			rowOrder = append(rowOrder, rowKey)
			counts[rowKey] = make(map[string]int64)
		}
		if !seenColumn[columnKey] {
			seenColumn[columnKey] = true
			columnOrder = append(columnOrder, columnKey)
		}
		counts[rowKey][columnKey]++
	}
	schema := append([]string{rowName}, columnOrder...)
	result := make([][]string, 0, len(rowOrder))
	for _, rowKey := range rowOrder {
		line := make([]string, 0, len(schema))
		line = append(line, rowKey)
		for _, columnKey := range columnOrder {
			line = append(line, strconv.FormatInt(counts[rowKey][columnKey], 10))
		}
		result = append(result, line)
	}
	return ahdTableValue(schema, result)
}

// ---------------------------------------------------------------------------
// Plot standard module
// ---------------------------------------------------------------------------
//
// Gonum's plotting library cannot be linked into this file: it is embedded
// verbatim into every natively-compiled AhdCode program, which builds in an
// isolated, dependency-free workspace with no vendoring or network access.
// Rendering therefore happens out-of-process, in the bundled ahdplot helper.
// Both this runtime and the persistent evaluator drive that helper the same
// way: write a JSON request to a temporary file, run
// `ahdplot <request-file>`, and read a JSON response from its stdout. See
// internal/plotproto for the canonical protocol shape (duplicated here,
// field-for-field, since this file cannot import that package).

// AhdPlotRuntimeHint is filled by the compiler when it builds a program that
// uses Plot, mirroring AhdLatexRuntimeHint: os.Executable() inside a
// natively-compiled program can be a short-lived temporary binary (ahdcode
// run) or a copy moved away from the toolchain's own install directory
// (ahdcode build -o ... then relocated), so the compiler resolves the
// bundled helper's location once, at build time, from its own executable
// path, and bakes the result in. The environment override is reserved for
// packaging and tests; neither mechanism is AhdCode language syntax.
var AhdPlotRuntimeHint string

// AhdChart is the runtime interchange shape for one Chart: every field a
// Chart operation might read or write, laid out exactly like the AhdCode
// Class's hidden storage fields. Kind discriminates which family-specific
// fields are populated; the generated Chart struct's field getters produce
// this same shape (see the generator's plotChartOfCode), and its all-fields
// constructor consumes it (see emitPlotHelpers).
type AhdChart struct {
	Kind string

	SeriesKinds  *AhdList[string]
	SeriesLabels *AhdList[string]
	SeriesX      *AhdList[*AhdList[float64]]
	SeriesY      *AhdList[*AhdList[float64]]

	BarLabels *AhdList[string]
	BarValues *AhdList[float64]

	HistogramValues *AhdList[float64]
	HistogramBins   int64

	BoxValues *AhdList[float64]

	ErrorX, ErrorY, ErrorLower, ErrorUpper *AhdList[float64]

	Title, XLabel, YLabel string
	Legend                bool
	Width, Height         int64
}

// The wire protocol structs below mirror internal/plotproto field-for-field;
// see that package's doc comment for why this file cannot import it instead.
type ahdPlotSeriesSpec struct {
	Kind  string    `json:"kind"`
	Label string    `json:"label,omitempty"`
	X     []float64 `json:"x"`
	Y     []float64 `json:"y"`
}

type ahdPlotChartSpec struct {
	Present bool   `json:"present"`
	Kind    string `json:"kind"`

	Title  string `json:"title,omitempty"`
	XLabel string `json:"x_label,omitempty"`
	YLabel string `json:"y_label,omitempty"`
	Legend bool   `json:"legend,omitempty"`

	Series []ahdPlotSeriesSpec `json:"series,omitempty"`

	BarLabels []string  `json:"bar_labels,omitempty"`
	BarValues []float64 `json:"bar_values,omitempty"`

	HistogramValues []float64 `json:"histogram_values,omitempty"`
	HistogramBins   int       `json:"histogram_bins,omitempty"`

	BoxValues []float64 `json:"box_values,omitempty"`

	ErrorX     []float64 `json:"error_x,omitempty"`
	ErrorY     []float64 `json:"error_y,omitempty"`
	ErrorLower []float64 `json:"error_lower,omitempty"`
	ErrorUpper []float64 `json:"error_upper,omitempty"`
}

type ahdPlotRequest struct {
	OutputPath string             `json:"output_path"`
	Width      int                `json:"width"`
	Height     int                `json:"height"`
	Rows       int                `json:"rows"`
	Columns    int                `json:"columns"`
	Charts     []ahdPlotChartSpec `json:"charts"`
}

type ahdPlotResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

func ahdPlotFloats(list *AhdList[float64]) []float64 {
	if list == nil {
		return nil
	}
	return list.Snapshot()
}

func ahdPlotStrings(list *AhdList[string]) []string {
	if list == nil {
		return nil
	}
	return list.Snapshot()
}

func ahdPlotFloatGrid(list *AhdList[*AhdList[float64]]) [][]float64 {
	if list == nil {
		return nil
	}
	items := list.Snapshot()
	grid := make([][]float64, len(items))
	for index, row := range items {
		grid[index] = ahdPlotFloats(row)
	}
	return grid
}

func ahdPlotFloatListSlice(list *AhdList[*AhdList[float64]]) []*AhdList[float64] {
	if list == nil {
		return nil
	}
	return list.Snapshot()
}

func ahdPlotChartSpecOf(chart AhdChart) ahdPlotChartSpec {
	spec := ahdPlotChartSpec{
		Present: true, Kind: chart.Kind,
		Title: chart.Title, XLabel: chart.XLabel, YLabel: chart.YLabel, Legend: chart.Legend,
	}
	switch chart.Kind {
	case "line-scatter":
		kinds, labels := ahdPlotStrings(chart.SeriesKinds), ahdPlotStrings(chart.SeriesLabels)
		xs, ys := ahdPlotFloatGrid(chart.SeriesX), ahdPlotFloatGrid(chart.SeriesY)
		for index := range kinds {
			spec.Series = append(spec.Series, ahdPlotSeriesSpec{
				Kind: kinds[index], Label: labels[index], X: xs[index], Y: ys[index],
			})
		}
	case "bar":
		spec.BarLabels, spec.BarValues = ahdPlotStrings(chart.BarLabels), ahdPlotFloats(chart.BarValues)
	case "histogram":
		spec.HistogramValues = ahdPlotFloats(chart.HistogramValues)
		spec.HistogramBins = int(chart.HistogramBins)
	case "box":
		spec.BoxValues = ahdPlotFloats(chart.BoxValues)
	case "errorBar":
		spec.ErrorX = ahdPlotFloats(chart.ErrorX)
		spec.ErrorY = ahdPlotFloats(chart.ErrorY)
		spec.ErrorLower = ahdPlotFloats(chart.ErrorLower)
		spec.ErrorUpper = ahdPlotFloats(chart.ErrorUpper)
	}
	return spec
}

// ahdPlotTempDir is AhdCode's own temporary area for Plot render requests and
// Chart.show/Figure.show preview images.
func ahdPlotTempDir(class *AhdClass) string {
	dir := filepath.Join(os.TempDir(), "ahdcode", "plot")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		AhdRaiseClass(class, "creating temporary directory: "+err.Error())
	}
	return dir
}

// ahdPlotDiscoverRuntime locates the bundled ahdplot renderer helper: an
// explicit override, then a path relative to the running executable,
// mirroring this file's Latex-runtime discovery.
func ahdPlotDiscoverRuntime() (string, error) {
	name := "ahdplot"
	if runtime.GOOS == "windows" {
		name = "ahdplot.exe"
	}
	candidates := []string{os.Getenv("AHDCODE_PLOT_RUNTIME")}
	if AhdPlotRuntimeHint != "" {
		candidates = append(candidates, filepath.Join(AhdPlotRuntimeHint, name))
	}
	if executable, err := os.Executable(); err == nil {
		bin := filepath.Dir(executable)
		candidates = append(candidates,
			filepath.Join(bin, name),
			filepath.Join(bin, "..", "libexec", "ahdcode", name),
		)
	}
	seen := make(map[string]bool)
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		candidate = filepath.Clean(candidate)
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("the Plot renderer helper (ahdplot) was not found; set AHDCODE_PLOT_RUNTIME " +
		"or reinstall AhdCode with the bundled Plot renderer")
}

// ahdPlotRender hands one render request to the ahdplot helper. Every
// failure path -- missing helper, timeout, malformed response,
// renderer-reported error -- becomes a PlotError, never a leaked Go/
// filesystem error.
func ahdPlotRender(class *AhdClass, request ahdPlotRequest) {
	runtimePath, err := ahdPlotDiscoverRuntime()
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
	dir := ahdPlotTempDir(class)
	requestFile, err := os.CreateTemp(dir, "request-*.json")
	if err != nil {
		AhdRaiseClass(class, "writing render request: "+err.Error())
	}
	defer os.Remove(requestFile.Name())
	encoded, err := json.Marshal(request)
	if err == nil {
		_, err = requestFile.Write(encoded)
	}
	if closeErr := requestFile.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		AhdRaiseClass(class, "writing render request: "+err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, runErr := exec.CommandContext(ctx, runtimePath, requestFile.Name()).Output()
	var response ahdPlotResponse
	_ = json.Unmarshal(output, &response)
	if runErr != nil || !response.OK {
		message := response.Message
		if message == "" && runErr != nil {
			message = runErr.Error()
		}
		AhdRaiseClass(class, "rendering chart: "+message)
	}
}

// ahdPlotOpenViewer opens an image with the platform's standard
// image-opening mechanism, passing the path as an argument rather than
// through a shell string. A short timeout keeps a headless environment (no
// handler registered) from hanging.
//
// Windows deliberately does not go through "cmd /c start": cmd.exe re-scans
// its whole command line for its own metacharacters (&, |, ^, %, and so on)
// after argv-level quoting has already happened, so a path containing one of
// those could be reinterpreted as shell syntax even though it arrived as a
// single, properly quoted argument. rundll32's url.dll,FileProtocolHandler
// entry point invokes the same file-association mechanism "start" uses,
// without a cmd.exe shell in between.
func ahdPlotOpenViewer(class *AhdClass, path string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.CommandContext(ctx, "open", path)
	case "windows":
		command = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", path)
	default:
		command = exec.CommandContext(ctx, "xdg-open", path)
	}
	if err := command.Run(); err != nil {
		AhdRaiseClass(class, "opening chart viewer: "+err.Error())
	}
}

func ahdPlotTempImagePath(class *AhdClass) string {
	dir := ahdPlotTempDir(class)
	file, err := os.CreateTemp(dir, "chart-*.png")
	if err != nil {
		AhdRaiseClass(class, "creating temporary file: "+err.Error())
	}
	path := file.Name()
	file.Close()
	return path
}

func ahdPlotRequireNonEmpty(class *AhdClass, count int, what string) {
	if count == 0 {
		AhdRaiseClass(class, what+" must not be empty")
	}
}

func ahdPlotRequireNonNegative(class *AhdClass, values []float64, what string) {
	for _, value := range values {
		if value < 0 {
			AhdRaiseClass(class, what+" must be non-negative")
		}
	}
}

// AhdPlotWidenList widens a List<Int> argument to List<Real>, so every Plot
// rendering helper works on one canonical numeric representation regardless
// of which Int/Real overload the frontend selected.
func AhdPlotWidenList(values *AhdList[int64]) *AhdList[float64] {
	if values == nil {
		return nil
	}
	items := values.Snapshot()
	widened := make([]float64, len(items))
	for index, value := range items {
		widened[index] = float64(value)
	}
	return AhdNewList(widened...)
}

// AhdPlotNew is Plot.new(): an empty Chart, ready for Chart.line/Chart.scatter
// to build up a multi-series composite.
func AhdPlotNew() AhdChart {
	return AhdChart{Kind: "empty", Width: 800, Height: 600}
}

func AhdPlotLine(class *AhdClass, x, y *AhdList[float64]) AhdChart {
	return ahdPlotNewSeries(class, "line", x, y)
}

func AhdPlotScatter(class *AhdClass, x, y *AhdList[float64]) AhdChart {
	return ahdPlotNewSeries(class, "scatter", x, y)
}

func ahdPlotNewSeries(class *AhdClass, kind string, x, y *AhdList[float64]) AhdChart {
	xs, ys := ahdPlotFloats(x), ahdPlotFloats(y)
	if len(xs) != len(ys) {
		AhdRaiseClass(class, "x and y must have the same length")
	}
	ahdPlotRequireNonEmpty(class, len(xs), kind+" chart data")
	return AhdChart{
		Kind: "line-scatter", SeriesKinds: AhdNewList(kind), SeriesLabels: AhdNewList(""),
		SeriesX: AhdNewList(AhdNewList(xs...)), SeriesY: AhdNewList(AhdNewList(ys...)),
		Width: 800, Height: 600,
	}
}

func AhdPlotBar(class *AhdClass, labels *AhdList[string], values *AhdList[float64]) AhdChart {
	ls, vs := ahdPlotStrings(labels), ahdPlotFloats(values)
	if len(ls) != len(vs) {
		AhdRaiseClass(class, "bar labels and values must have the same length")
	}
	ahdPlotRequireNonEmpty(class, len(vs), "bar chart data")
	return AhdChart{Kind: "bar", BarLabels: AhdNewList(ls...), BarValues: AhdNewList(vs...), Width: 800, Height: 600}
}

func AhdPlotHistogram(class *AhdClass, values *AhdList[float64], bins int64) AhdChart {
	vs := ahdPlotFloats(values)
	if bins <= 0 {
		AhdRaiseClass(class, "histogram bin count must be positive")
	}
	ahdPlotRequireNonEmpty(class, len(vs), "histogram data")
	return AhdChart{Kind: "histogram", HistogramValues: AhdNewList(vs...), HistogramBins: bins, Width: 800, Height: 600}
}

func AhdPlotBox(class *AhdClass, values *AhdList[float64]) AhdChart {
	vs := ahdPlotFloats(values)
	ahdPlotRequireNonEmpty(class, len(vs), "box plot data")
	return AhdChart{Kind: "box", BoxValues: AhdNewList(vs...), Width: 800, Height: 600}
}

func AhdPlotErrorBar(class *AhdClass, x, y, lower, upper *AhdList[float64]) AhdChart {
	xs, ys, los, ups := ahdPlotFloats(x), ahdPlotFloats(y), ahdPlotFloats(lower), ahdPlotFloats(upper)
	if len(xs) != len(ys) || len(ys) != len(los) || len(los) != len(ups) {
		AhdRaiseClass(class, "errorBar x, y, lowerErrors, and upperErrors must have the same length")
	}
	ahdPlotRequireNonEmpty(class, len(xs), "errorBar data")
	ahdPlotRequireNonNegative(class, los, "lowerErrors")
	ahdPlotRequireNonNegative(class, ups, "upperErrors")
	return AhdChart{
		Kind: "errorBar", ErrorX: AhdNewList(xs...), ErrorY: AhdNewList(ys...),
		ErrorLower: AhdNewList(los...), ErrorUpper: AhdNewList(ups...), Width: 800, Height: 600,
	}
}

func AhdPlotChartTitle(chart AhdChart, text string) AhdChart  { chart.Title = text; return chart }
func AhdPlotChartXLabel(chart AhdChart, text string) AhdChart { chart.XLabel = text; return chart }
func AhdPlotChartYLabel(chart AhdChart, text string) AhdChart { chart.YLabel = text; return chart }
func AhdPlotChartLegend(chart AhdChart, enabled bool) AhdChart {
	chart.Legend = enabled
	return chart
}

func AhdPlotChartSize(class *AhdClass, chart AhdChart, width, height int64) AhdChart {
	if width <= 0 || height <= 0 {
		AhdRaiseClass(class, "chart size must be positive")
	}
	chart.Width, chart.Height = width, height
	return chart
}

// AhdPlotChartAddSeries implements Chart.line and Chart.scatter: append one
// more series to a Chart that is either empty or already a line/scatter
// composite. It never mutates the receiver's storage -- every List is
// rebuilt fresh -- so an alias to the original Chart is unaffected.
func AhdPlotChartAddSeries(class *AhdClass, chart AhdChart, kind string, x, y *AhdList[float64], label string) AhdChart {
	if chart.Kind != "empty" && chart.Kind != "line-scatter" {
		AhdRaiseClass(class, "cannot add a "+kind+" series to a "+chart.Kind+" chart")
	}
	xs, ys := ahdPlotFloats(x), ahdPlotFloats(y)
	if len(xs) != len(ys) {
		AhdRaiseClass(class, "x and y must have the same length")
	}
	ahdPlotRequireNonEmpty(class, len(xs), kind+" chart data")
	chart.Kind = "line-scatter"
	chart.SeriesKinds = AhdNewList(append(ahdPlotStrings(chart.SeriesKinds), kind)...)
	chart.SeriesLabels = AhdNewList(append(ahdPlotStrings(chart.SeriesLabels), label)...)
	chart.SeriesX = AhdNewList(append(ahdPlotFloatListSlice(chart.SeriesX), AhdNewList(xs...))...)
	chart.SeriesY = AhdNewList(append(ahdPlotFloatListSlice(chart.SeriesY), AhdNewList(ys...))...)
	return chart
}

func AhdPlotChartSave(class *AhdClass, chart AhdChart, path string) {
	ahdPlotRender(class, ahdPlotRequest{
		OutputPath: path, Width: int(chart.Width), Height: int(chart.Height),
		Rows: 1, Columns: 1, Charts: []ahdPlotChartSpec{ahdPlotChartSpecOf(chart)},
	})
}

func AhdPlotChartShow(class *AhdClass, chart AhdChart) {
	path := ahdPlotTempImagePath(class)
	ahdPlotRender(class, ahdPlotRequest{
		OutputPath: path, Width: int(chart.Width), Height: int(chart.Height),
		Rows: 1, Columns: 1, Charts: []ahdPlotChartSpec{ahdPlotChartSpecOf(chart)},
	})
	ahdPlotOpenViewer(class, path)
}

// AhdPlotSubplotsValidate checks Figure construction's domain rules (rows >
// 0, columns > 0, chart count <= rows*columns), generically over whichever
// generated Chart interface type the compiled program uses, and returns the
// List unchanged so the generated Figure constructor can use it directly.
func AhdPlotSubplotsValidate[T any](class *AhdClass, rows, columns int64, charts *AhdList[T]) *AhdList[T] {
	if rows <= 0 || columns <= 0 {
		AhdRaiseClass(class, "subplot rows and columns must be positive")
	}
	var count int64
	if charts != nil {
		count = int64(len(charts.Snapshot()))
	}
	if count > rows*columns {
		AhdRaiseClass(class, "more charts than subplot cells")
	}
	return charts
}

// AhdPlotFigureDefaultSize is Figure's one deterministic size rule: a fixed
// per-cell budget scaled by the grid dimensions. v0.1.14 publishes no
// Figure.size method, keeping subplot sizing to this one predictable formula
// rather than a second configuration surface.
func AhdPlotFigureDefaultSize(rows, columns int64) (int64, int64) {
	return columns * 500, rows * 400
}

// AhdPlotChartsFrom converts a List of the generated program's own Chart
// values into the runtime's AhdChart interchange shape. This file cannot
// know the generated Chart interface type, so the generator supplies the
// per-element conversion closure.
func AhdPlotChartsFrom[T any](list *AhdList[T], convert func(T) AhdChart) []AhdChart {
	if list == nil {
		return nil
	}
	items := list.Snapshot()
	charts := make([]AhdChart, len(items))
	for index, item := range items {
		charts[index] = convert(item)
	}
	return charts
}

func ahdPlotFigureRequest(rows, columns int64, charts []AhdChart, path string, width, height int) ahdPlotRequest {
	cells := make([]ahdPlotChartSpec, rows*columns)
	for index := range cells {
		if int64(index) < int64(len(charts)) {
			cells[index] = ahdPlotChartSpecOf(charts[index])
		}
	}
	return ahdPlotRequest{
		OutputPath: path, Width: width, Height: height, Rows: int(rows), Columns: int(columns), Charts: cells,
	}
}

func AhdPlotFigureSave(class *AhdClass, rows, columns int64, charts []AhdChart, path string) {
	width, height := AhdPlotFigureDefaultSize(rows, columns)
	ahdPlotRender(class, ahdPlotFigureRequest(rows, columns, charts, path, int(width), int(height)))
}

func AhdPlotFigureShow(class *AhdClass, rows, columns int64, charts []AhdChart) {
	path := ahdPlotTempImagePath(class)
	width, height := AhdPlotFigureDefaultSize(rows, columns)
	ahdPlotRender(class, ahdPlotFigureRequest(rows, columns, charts, path, int(width), int(height)))
	ahdPlotOpenViewer(class, path)
}

// ---------------------------------------------------------------------------
// Numeric standard module
// ---------------------------------------------------------------------------

type AhdVector struct{ Values *AhdList[float64] }
type AhdMatrix struct{ Rows *AhdList[*AhdList[float64]] }
type AhdMatrixPair struct {
	Keys   []string
	Values []AhdMatrix
}

var AhdNumericRuntimeHint string

// The wire types intentionally mirror internal/numericproto. This runtime is
// copied into dependency-free generated workspaces, so it cannot import that
// package directly.
type ahdNumericRequest struct {
	Operation string      `json:"operation"`
	Matrix    [][]float64 `json:"matrix"`
	Vector    []float64   `json:"vector,omitempty"`
}

type ahdNumericComplex struct {
	Real float64 `json:"real"`
	Imag float64 `json:"imag"`
}

type ahdNumericResponse struct {
	Error    string                 `json:"error,omitempty"`
	Scalar   *float64               `json:"scalar,omitempty"`
	Integer  *int                   `json:"integer,omitempty"`
	Vector   []float64              `json:"vector,omitempty"`
	Matrix   [][]float64            `json:"matrix,omitempty"`
	Matrices map[string][][]float64 `json:"matrices,omitempty"`
	Complex  []ahdNumericComplex    `json:"complex,omitempty"`
}

func ahdNumericDiscoverRuntime() (string, error) {
	name := "ahdnumeric"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	var candidates []string
	if custom := os.Getenv("AHDCODE_NUMERIC_RUNTIME"); custom != "" {
		candidates = append(candidates, custom, filepath.Join(custom, name))
	}
	if AhdNumericRuntimeHint != "" {
		candidates = append(candidates, filepath.Join(AhdNumericRuntimeHint, name))
	}
	if executable, err := os.Executable(); err == nil {
		bin := filepath.Dir(executable)
		candidates = append(candidates,
			filepath.Join(bin, name),
			filepath.Join(bin, "..", "libexec", "ahdcode", name),
		)
	}
	seen := make(map[string]bool)
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		candidate = filepath.Clean(candidate)
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("the Numeric helper (ahdnumeric) was not found; set AHDCODE_NUMERIC_RUNTIME or reinstall AhdCode with the bundled Numeric helper")
}

func ahdNumericCall(class *AhdClass, operation string, matrix [][]float64, vector []float64) ahdNumericResponse {
	helper, err := ahdNumericDiscoverRuntime()
	if err != nil {
		ahdNumericRaise(class, err.Error())
	}
	dir := filepath.Join(os.TempDir(), "ahdcode", "numeric")
	if err = os.MkdirAll(dir, 0o700); err != nil {
		ahdNumericRaise(class, "creating temporary directory: "+err.Error())
	}
	file, err := os.CreateTemp(dir, "request-*.json")
	if err != nil {
		ahdNumericRaise(class, "writing Numeric request: "+err.Error())
	}
	defer os.Remove(file.Name())
	encoded, err := json.Marshal(ahdNumericRequest{Operation: operation, Matrix: matrix, Vector: vector})
	if err == nil {
		_, err = file.Write(encoded)
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		ahdNumericRaise(class, "writing Numeric request: "+err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, runErr := exec.CommandContext(ctx, helper, file.Name()).Output()
	var response ahdNumericResponse
	decodeErr := json.Unmarshal(output, &response)
	if response.Error != "" {
		ahdNumericRaise(class, response.Error)
	}
	if ctx.Err() != nil {
		ahdNumericRaise(class, "Numeric helper timed out")
	}
	if decodeErr != nil {
		ahdNumericRaise(class, "Numeric helper returned an invalid response")
	}
	if runErr != nil {
		ahdNumericRaise(class, "Numeric helper failed: "+runErr.Error())
	}
	return response
}

func ahdNumericResponseMatrix(class *AhdClass, response ahdNumericResponse) AhdMatrix {
	if len(response.Matrix) == 0 {
		ahdNumericRaise(class, "Numeric helper omitted its Matrix result")
	}
	result := ahdNumericMatrix(response.Matrix)
	ahdNumericShape(class, result)
	return result
}

func ahdNumericResponseMatrices(class *AhdClass, response ahdNumericResponse, keys ...string) AhdMatrixPair {
	result := AhdMatrixPair{Keys: append([]string(nil), keys...), Values: make([]AhdMatrix, len(keys))}
	for index, key := range keys {
		rows, ok := response.Matrices[key]
		if !ok || len(rows) == 0 {
			ahdNumericRaise(class, "Numeric helper omitted its "+key+" Matrix result")
		}
		result.Values[index] = ahdNumericMatrix(rows)
		ahdNumericShape(class, result.Values[index])
	}
	return result
}

func ahdNumericRaise(class *AhdClass, message string) { AhdRaiseClass(class, message) }
func ahdNumericValues(vector AhdVector) []float64 {
	if vector.Values == nil {
		return nil
	}
	return vector.Values.Snapshot()
}
func ahdNumericRows(matrix AhdMatrix) [][]float64 {
	if matrix.Rows == nil {
		return nil
	}
	source := matrix.Rows.Snapshot()
	rows := make([][]float64, len(source))
	for i, row := range source {
		if row != nil {
			rows[i] = row.Snapshot()
		}
	}
	return rows
}
func ahdNumericVector(values []float64) AhdVector { return AhdVector{Values: AhdNewList(values...)} }
func ahdNumericMatrix(rows [][]float64) AhdMatrix {
	result := make([]*AhdList[float64], len(rows))
	for i, row := range rows {
		result[i] = AhdNewList(append([]float64(nil), row...)...)
	}
	return AhdMatrix{Rows: AhdNewList(result...)}
}
func ahdNumericShape(class *AhdClass, matrix AhdMatrix) ([][]float64, int, int) {
	rows := ahdNumericRows(matrix)
	if len(rows) == 0 {
		ahdNumericRaise(class, "matrix requires at least one row")
	}
	columns := len(rows[0])
	if columns == 0 {
		ahdNumericRaise(class, "matrix requires at least one column")
	}
	for _, row := range rows {
		if len(row) != columns {
			ahdNumericRaise(class, "matrix rows must have equal lengths")
		}
	}
	return rows, len(rows), columns
}
func ahdNumericSquare(class *AhdClass, m AhdMatrix) ([][]float64, int) {
	rows, r, c := ahdNumericShape(class, m)
	if r != c {
		ahdNumericRaise(class, "operation requires a square matrix")
	}
	return rows, r
}

func AhdNumericWidenList(values *AhdList[int64]) *AhdList[float64] {
	if values == nil {
		return nil
	}
	source := values.Snapshot()
	out := make([]float64, len(source))
	for i, v := range source {
		out[i] = float64(v)
	}
	return AhdNewList(out...)
}
func AhdNumericWidenGrid(values *AhdList[*AhdList[int64]]) *AhdList[*AhdList[float64]] {
	if values == nil {
		return nil
	}
	source := values.Snapshot()
	out := make([]*AhdList[float64], len(source))
	for i, row := range source {
		out[i] = AhdNumericWidenList(row)
	}
	return AhdNewList(out...)
}
func AhdNumericVector(class *AhdClass, values *AhdList[float64]) AhdVector {
	if values == nil {
		ahdNumericRaise(class, "vector values must not be null")
	}
	return ahdNumericVector(values.Snapshot())
}
func AhdNumericMatrix(class *AhdClass, rows *AhdList[*AhdList[float64]]) AhdMatrix {
	m := AhdMatrix{Rows: rows}
	grid, _, _ := ahdNumericShape(class, m)
	return ahdNumericMatrix(grid)
}
func ahdNumericSize(class *AhdClass, size int64) int {
	if size < 0 || size > int64(^uint(0)>>1) {
		ahdNumericRaise(class, "size must be a non-negative Int")
	}
	return int(size)
}
func AhdNumericZerosVector(class *AhdClass, size int64) AhdVector {
	return ahdNumericVector(make([]float64, ahdNumericSize(class, size)))
}
func AhdNumericOnesVector(class *AhdClass, size int64) AhdVector {
	v := make([]float64, ahdNumericSize(class, size))
	for i := range v {
		v[i] = 1
	}
	return ahdNumericVector(v)
}
func ahdNumericFilledMatrix(class *AhdClass, r, c int64, value float64) AhdMatrix {
	rows, cols := ahdNumericSize(class, r), ahdNumericSize(class, c)
	if rows == 0 || cols == 0 {
		ahdNumericRaise(class, "matrix dimensions must be positive")
	}
	grid := make([][]float64, rows)
	for i := range grid {
		grid[i] = make([]float64, cols)
		if value != 0 {
			for j := range grid[i] {
				grid[i][j] = value
			}
		}
	}
	return ahdNumericMatrix(grid)
}
func AhdNumericZerosMatrix(class *AhdClass, r, c int64) AhdMatrix {
	return ahdNumericFilledMatrix(class, r, c, 0)
}
func AhdNumericOnesMatrix(class *AhdClass, r, c int64) AhdMatrix {
	return ahdNumericFilledMatrix(class, r, c, 1)
}
func AhdNumericIdentity(class *AhdClass, size int64) AhdMatrix {
	n := ahdNumericSize(class, size)
	if n == 0 {
		ahdNumericRaise(class, "identity size must be positive")
	}
	m := make([][]float64, n)
	for i := range m {
		m[i] = make([]float64, n)
		m[i][i] = 1
	}
	return ahdNumericMatrix(m)
}
func AhdNumericLinspace(class *AhdClass, start, stop float64, count int64) AhdVector {
	if count <= 0 {
		ahdNumericRaise(class, "linspace count must be positive")
	}
	v := make([]float64, count)
	if count == 1 {
		v[0] = start
	} else {
		step := (stop - start) / float64(count-1)
		for i := range v {
			v[i] = start + float64(i)*step
		}
		v[len(v)-1] = stop
	}
	return ahdNumericVector(v)
}

func AhdNumericVectorLength(v AhdVector) int64             { return int64(len(ahdNumericValues(v))) }
func AhdNumericVectorValues(v AhdVector) *AhdList[float64] { return AhdNewList(ahdNumericValues(v)...) }
func ahdNumericVectorBinary(class *AhdClass, a, b AhdVector, subtract bool) AhdVector {
	x, y := ahdNumericValues(a), ahdNumericValues(b)
	if len(x) != len(y) {
		ahdNumericRaise(class, "vector lengths do not match")
	}
	out := make([]float64, len(x))
	for i := range x {
		if subtract {
			out[i] = x[i] - y[i]
		} else {
			out[i] = x[i] + y[i]
		}
	}
	return ahdNumericVector(out)
}
func AhdNumericVectorAdd(class *AhdClass, a, b AhdVector) AhdVector {
	return ahdNumericVectorBinary(class, a, b, false)
}
func AhdNumericVectorSubtract(class *AhdClass, a, b AhdVector) AhdVector {
	return ahdNumericVectorBinary(class, a, b, true)
}
func AhdNumericVectorScale(v AhdVector, f float64) AhdVector {
	x := ahdNumericValues(v)
	for i := range x {
		x[i] *= f
	}
	return ahdNumericVector(x)
}
func AhdNumericVectorDot(class *AhdClass, a, b AhdVector) float64 {
	x, y := ahdNumericValues(a), ahdNumericValues(b)
	if len(x) != len(y) {
		ahdNumericRaise(class, "vector lengths do not match")
	}
	sum := 0.0
	for i := range x {
		sum += x[i] * y[i]
	}
	return sum
}
func ahdNumericElement(class *AhdClass, value float64, operation string) float64 {
	switch operation {
	case "abs":
		return math.Abs(value)
	case "sqrt":
		if value < 0 {
			ahdNumericRaise(class, "sqrt requires non-negative values")
		}
		return math.Sqrt(value)
	case "exp":
		value = math.Exp(value)
	case "log":
		if value <= 0 {
			ahdNumericRaise(class, "log requires positive values")
		}
		value = math.Log(value)
	}
	if math.IsInf(value, 0) || math.IsNaN(value) {
		ahdNumericRaise(class, "elementwise operation produced a non-finite value")
	}
	return value
}
func AhdNumericVectorElementwise(class *AhdClass, v AhdVector, op string) AhdVector {
	x := ahdNumericValues(v)
	for i := range x {
		x[i] = ahdNumericElement(class, x[i], op)
	}
	return ahdNumericVector(x)
}
func AhdNumericVectorReduction(class *AhdClass, v AhdVector, op string) float64 {
	x := ahdNumericValues(v)
	if len(x) == 0 && (op == "min" || op == "max") {
		ahdNumericRaise(class, op+" requires a non-empty Vector")
	}
	result := 0.0
	if len(x) > 0 && (op == "min" || op == "max") {
		result = x[0]
	}
	for i, value := range x {
		if op == "sum" {
			result += value
		} else if op == "min" && value < result {
			result = value
		} else if op == "max" && value > result {
			result = value
		}
		_ = i
	}
	return result
}

func AhdNumericMatrixRowCount(m AhdMatrix) int64 { return int64(len(ahdNumericRows(m))) }
func AhdNumericMatrixColumnCount(m AhdMatrix) int64 {
	rows := ahdNumericRows(m)
	if len(rows) == 0 {
		return 0
	}
	return int64(len(rows[0]))
}
func AhdNumericMatrixRows(m AhdMatrix) *AhdList[*AhdList[float64]] {
	return ahdNumericMatrix(ahdNumericRows(m)).Rows
}
func AhdNumericMatrixTranspose(m AhdMatrix) AhdMatrix {
	a := ahdNumericRows(m)
	if len(a) == 0 {
		return ahdNumericMatrix(nil)
	}
	out := make([][]float64, len(a[0]))
	for j := range out {
		out[j] = make([]float64, len(a))
		for i := range a {
			out[j][i] = a[i][j]
		}
	}
	return ahdNumericMatrix(out)
}
func ahdNumericMatrixBinary(class *AhdClass, a, b AhdMatrix, subtract bool) AhdMatrix {
	x, r, c := ahdNumericShape(class, a)
	y, rr, cc := ahdNumericShape(class, b)
	if r != rr || c != cc {
		ahdNumericRaise(class, "matrix shapes do not match")
	}
	for i := range x {
		for j := range x[i] {
			if subtract {
				x[i][j] -= y[i][j]
			} else {
				x[i][j] += y[i][j]
			}
		}
	}
	return ahdNumericMatrix(x)
}
func AhdNumericMatrixAdd(class *AhdClass, a, b AhdMatrix) AhdMatrix {
	return ahdNumericMatrixBinary(class, a, b, false)
}
func AhdNumericMatrixSubtract(class *AhdClass, a, b AhdMatrix) AhdMatrix {
	return ahdNumericMatrixBinary(class, a, b, true)
}
func AhdNumericMatrixScale(m AhdMatrix, f float64) AhdMatrix {
	a := ahdNumericRows(m)
	for i := range a {
		for j := range a[i] {
			a[i][j] *= f
		}
	}
	return ahdNumericMatrix(a)
}
func AhdNumericMatrixMatmul(class *AhdClass, a, b AhdMatrix) AhdMatrix {
	x, r, k := ahdNumericShape(class, a)
	y, kk, c := ahdNumericShape(class, b)
	if k != kk {
		ahdNumericRaise(class, "matrix multiplication shapes do not match")
	}
	out := make([][]float64, r)
	for i := range out {
		out[i] = make([]float64, c)
		for j := 0; j < c; j++ {
			for q := 0; q < k; q++ {
				out[i][j] += x[i][q] * y[q][j]
			}
		}
	}
	return ahdNumericMatrix(out)
}
func AhdNumericMatrixTrace(class *AhdClass, m AhdMatrix) float64 {
	a, n := ahdNumericSquare(class, m)
	sum := 0.0
	for i := 0; i < n; i++ {
		sum += a[i][i]
	}
	return sum
}
func ahdNumericEliminate(a [][]float64) (rank int, det float64, swaps int) {
	rows, cols := len(a), len(a[0])
	det = 1
	pivotRow := 0
	for col := 0; col < cols && pivotRow < rows; col++ {
		pivot := pivotRow
		for i := pivotRow + 1; i < rows; i++ {
			if math.Abs(a[i][col]) > math.Abs(a[pivot][col]) {
				pivot = i
			}
		}
		if math.Abs(a[pivot][col]) <= 1e-12 {
			if rows == cols {
				det = 0
			}
			continue
		}
		if pivot != pivotRow {
			a[pivot], a[pivotRow] = a[pivotRow], a[pivot]
			swaps++
		}
		p := a[pivotRow][col]
		if rows == cols {
			det *= p
		}
		for i := pivotRow + 1; i < rows; i++ {
			factor := a[i][col] / p
			for j := col; j < cols; j++ {
				a[i][j] -= factor * a[pivotRow][j]
			}
		}
		pivotRow++
		rank++
	}
	if swaps%2 != 0 {
		det = -det
	}
	return
}
func AhdNumericMatrixDeterminant(class *AhdClass, m AhdMatrix) float64 {
	a, _ := ahdNumericSquare(class, m)
	response := ahdNumericCall(class, "determinant", a, nil)
	if response.Scalar == nil {
		ahdNumericRaise(class, "Numeric helper omitted its scalar result")
	}
	return *response.Scalar
}
func AhdNumericMatrixRank(class *AhdClass, m AhdMatrix) int64 {
	a, _, _ := ahdNumericShape(class, m)
	response := ahdNumericCall(class, "rank", a, nil)
	if response.Integer == nil {
		ahdNumericRaise(class, "Numeric helper omitted its Int result")
	}
	return int64(*response.Integer)
}
func ahdNumericSolve(class *AhdClass, a [][]float64, b []float64) []float64 {
	n := len(a)
	if len(b) != n {
		ahdNumericRaise(class, "system dimensions do not match")
	}
	for i := 0; i < n; i++ {
		a[i] = append(a[i], b[i])
	}
	for col := 0; col < n; col++ {
		pivot := col
		for i := col + 1; i < n; i++ {
			if math.Abs(a[i][col]) > math.Abs(a[pivot][col]) {
				pivot = i
			}
		}
		if math.Abs(a[pivot][col]) <= 1e-12 {
			ahdNumericRaise(class, "matrix is singular")
		}
		a[pivot], a[col] = a[col], a[pivot]
		p := a[col][col]
		for j := col; j <= n; j++ {
			a[col][j] /= p
		}
		for i := 0; i < n; i++ {
			if i == col {
				continue
			}
			f := a[i][col]
			for j := col; j <= n; j++ {
				a[i][j] -= f * a[col][j]
			}
		}
	}
	x := make([]float64, n)
	for i := range x {
		x[i] = a[i][n]
	}
	return x
}
func AhdNumericMatrixSolve(class *AhdClass, m AhdMatrix, v AhdVector) AhdVector {
	a, _ := ahdNumericSquare(class, m)
	response := ahdNumericCall(class, "solve", a, ahdNumericValues(v))
	if response.Vector == nil {
		ahdNumericRaise(class, "Numeric helper omitted its Vector result")
	}
	return ahdNumericVector(response.Vector)
}
func AhdNumericMatrixInverse(class *AhdClass, m AhdMatrix) AhdMatrix {
	a, _ := ahdNumericSquare(class, m)
	return ahdNumericResponseMatrix(class, ahdNumericCall(class, "inverse", a, nil))
}
func AhdNumericMatrixElementwise(class *AhdClass, m AhdMatrix, op string) AhdMatrix {
	a := ahdNumericRows(m)
	for i := range a {
		for j := range a[i] {
			a[i][j] = ahdNumericElement(class, a[i][j], op)
		}
	}
	return ahdNumericMatrix(a)
}
func AhdNumericMatrixReduction(class *AhdClass, m AhdMatrix, op string) float64 {
	a, _, _ := ahdNumericShape(class, m)
	flat := []float64{}
	for _, row := range a {
		flat = append(flat, row...)
	}
	return AhdNumericVectorReduction(class, ahdNumericVector(flat), op)
}

func AhdNumericMatrixCholesky(class *AhdClass, m AhdMatrix) AhdMatrix {
	a, _ := ahdNumericSquare(class, m)
	return ahdNumericResponseMatrix(class, ahdNumericCall(class, "cholesky", a, nil))
}
func ahdNumericQR(class *AhdClass, a [][]float64) ([][]float64, [][]float64) {
	m, n := len(a), len(a[0])
	q := make([][]float64, m)
	for i := range q {
		q[i] = make([]float64, n)
	}
	r := make([][]float64, n)
	for i := range r {
		r[i] = make([]float64, n)
	}
	for j := 0; j < n; j++ {
		v := make([]float64, m)
		for i := range v {
			v[i] = a[i][j]
		}
		for k := 0; k < j; k++ {
			for i := 0; i < m; i++ {
				r[k][j] += q[i][k] * v[i]
			}
			for i := 0; i < m; i++ {
				v[i] -= r[k][j] * q[i][k]
			}
		}
		for _, x := range v {
			r[j][j] += x * x
		}
		r[j][j] = math.Sqrt(r[j][j])
		if r[j][j] <= 1e-12 {
			ahdNumericRaise(class, "QR decomposition failed for rank-deficient matrix")
		}
		for i := 0; i < m; i++ {
			q[i][j] = v[i] / r[j][j]
		}
	}
	return q, r
}
func AhdNumericMatrixQR(class *AhdClass, m AhdMatrix) AhdMatrixPair {
	a, _, _ := ahdNumericShape(class, m)
	return ahdNumericResponseMatrices(class, ahdNumericCall(class, "qr", a, nil), "Q", "R")
}
func AhdNumericMatrixLU(class *AhdClass, m AhdMatrix) AhdMatrixPair {
	a, _ := ahdNumericSquare(class, m)
	return ahdNumericResponseMatrices(class, ahdNumericCall(class, "lu", a, nil), "P", "L", "U")
}
func AhdNumericMatrixSVD(class *AhdClass, m AhdMatrix) AhdMatrixPair {
	a, _, _ := ahdNumericShape(class, m)
	return ahdNumericResponseMatrices(class, ahdNumericCall(class, "svd", a, nil), "U", "S", "V")
}
func ahdNumericJacobi(a [][]float64) ([]float64, [][]float64) {
	n := len(a)
	v := make([][]float64, n)
	for i := range v {
		v[i] = make([]float64, n)
		v[i][i] = 1
	}
	for iteration := 0; iteration < 100*n*n; iteration++ {
		p, q := 0, 0
		largest := 0.0
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				if math.Abs(a[i][j]) > largest {
					largest = math.Abs(a[i][j])
					p, q = i, j
				}
			}
		}
		if largest < 1e-12 {
			break
		}
		angle := .5 * math.Atan2(2*a[p][q], a[q][q]-a[p][p])
		c, s := math.Cos(angle), math.Sin(angle)
		for i := 0; i < n; i++ {
			ap, aq := a[i][p], a[i][q]
			a[i][p], a[i][q] = c*ap-s*aq, s*ap+c*aq
		}
		for j := 0; j < n; j++ {
			ap, aq := a[p][j], a[q][j]
			a[p][j], a[q][j] = c*ap-s*aq, s*ap+c*aq
		}
		for i := 0; i < n; i++ {
			vp, vq := v[i][p], v[i][q]
			v[i][p], v[i][q] = c*vp-s*vq, s*vp+c*vq
		}
	}
	values := make([]float64, n)
	for i := range values {
		values[i] = a[i][i]
	}
	return values, v
}
func AhdNumericMatrixEigenvalues(class *AhdClass, m AhdMatrix) *AhdList[complex128] {
	a, _ := ahdNumericSquare(class, m)
	response := ahdNumericCall(class, "eigenvalues", a, nil)
	out := make([]complex128, len(response.Complex))
	if len(out) == 0 {
		ahdNumericRaise(class, "Numeric helper omitted its Complex result")
	}
	for i, value := range response.Complex {
		out[i] = complex(value.Real, value.Imag)
	}
	return AhdNewList(out...)
}
func AhdNumericWrapMatrixPair[T any](pair AhdMatrixPair, wrap func(AhdMatrix) T) *AhdPair[string, T] {
	values := make([]T, len(pair.Values))
	for i, v := range pair.Values {
		values[i] = wrap(v)
	}
	return AhdBuildPair(pair.Keys, values)
}

// ---------------------------------------------------------------------------
// Word standard module (WordprocessingML / DOCX)
// ---------------------------------------------------------------------------
//
// A Document's entire visible AhdCode surface is one immutable value plus
// WordError. Internally a Document carries exactly one hidden field, a
// List<String>, where every element is one JSON-encoded ahdWordBlock. That
// encoding is a private implementation detail: it is never read by AhdCode
// source (the field is hidden) and never round-tripped through any public
// operation, so it can change freely across releases without being a
// compatibility surface. Every operation appends to (or reads) that block
// list; nothing here shells out, links a third-party package, or depends on
// system Office software.

func ahdWordRaise(message string) { AhdRaiseClass(AhdClassWordError, message) }

// ahdWordBlock is the private content-block shape. Kind selects which of the
// remaining fields are meaningful; the zero value of every other field is
// simply omitted from the JSON encoding.
type ahdWordBlock struct {
	Kind      string     `json:"kind"`
	Text      string     `json:"text,omitempty"`
	Level     int        `json:"level,omitempty"`
	Align     string     `json:"align,omitempty"`
	Bold      bool       `json:"bold,omitempty"`
	Italic    bool       `json:"italic,omitempty"`
	Underline bool       `json:"underline,omitempty"`
	Headers   []string   `json:"headers,omitempty"`
	Rows      [][]string `json:"rows,omitempty"`
	Merges    [][4]int   `json:"merges,omitempty"`
	Media     []byte     `json:"media,omitempty"`
	MediaExt  string     `json:"mediaExt,omitempty"`
	WidthEMU  int64      `json:"widthEMU,omitempty"`
	HeightEMU int64      `json:"heightEMU,omitempty"`
}

// AhdWordDocument is the runtime interchange shape the generated backend
// reads and writes through the Document Class's one hidden field.
type AhdWordDocument struct {
	Blocks []string
}

// ahdWordAppend returns a new document with one more block. It always copies
// the existing block slice before appending, so two documents derived from
// the same base never share a backing array: appending to one can never
// retroactively change what the other one already produced.
func ahdWordAppend(doc AhdWordDocument, block ahdWordBlock) AhdWordDocument {
	// This concrete struct has no channel, function, or cyclic field, so
	// encoding it as JSON cannot fail.
	encoded, _ := json.Marshal(block)
	blocks := append(append([]string(nil), doc.Blocks...), string(encoded))
	return AhdWordDocument{Blocks: blocks}
}

func ahdWordDecodeBlocks(doc AhdWordDocument) []ahdWordBlock {
	blocks := make([]ahdWordBlock, len(doc.Blocks))
	for index, raw := range doc.Blocks {
		var block ahdWordBlock
		if err := json.Unmarshal([]byte(raw), &block); err != nil {
			ahdWordRaise("document storage is corrupted")
		}
		blocks[index] = block
	}
	return blocks
}

// AhdWordNew starts an empty Document. It is the only zero-content entry
// point: nothing else in the module constructs a Document from nothing.
func AhdWordNew() AhdWordDocument { return AhdWordDocument{} }

var ahdWordParagraphAlignments = map[string]bool{"left": true, "center": true, "right": true, "justify": true}
var ahdWordTableAlignments = map[string]bool{"left": true, "center": true, "right": true}

func AhdWordHeading(doc AhdWordDocument, text string, level int64) AhdWordDocument {
	if level < 1 || level > 6 {
		ahdWordRaise("heading level must be between 1 and 6")
	}
	return ahdWordAppend(doc, ahdWordBlock{Kind: "heading", Text: text, Level: int(level)})
}

func AhdWordParagraph(doc AhdWordDocument, text, align string, bold, italic, underline bool) AhdWordDocument {
	if !ahdWordParagraphAlignments[align] {
		ahdWordRaise("paragraph align must be left, center, right, or justify")
	}
	return ahdWordAppend(doc, ahdWordBlock{
		Kind: "paragraph", Text: text, Align: align, Bold: bold, Italic: italic, Underline: underline,
	})
}

// AhdWordTable validates shape and merge geometry before ever building XML,
// so a malformed table is always a clean WordError, never a corrupt package.
func AhdWordTable(doc AhdWordDocument, headers *AhdList[string], rows *AhdList[*AhdList[string]], merges *AhdList[*AhdList[int64]], align string) AhdWordDocument {
	if !ahdWordTableAlignments[align] {
		ahdWordRaise("table align must be left, center, or right")
	}
	headerValues := headers.Snapshot()
	if len(headerValues) == 0 {
		ahdWordRaise("table requires at least one column")
	}
	rowValues := rows.Snapshot()
	grid := make([][]string, len(rowValues))
	for index, row := range rowValues {
		nonNullRow := AhdNonNull(row)
		cells := nonNullRow.Snapshot()
		if len(cells) != len(headerValues) {
			ahdWordRaise("table row column count does not match headers")
		}
		grid[index] = cells
	}
	mergeValues := ahdWordValidateMerges(merges, 1+len(grid), len(headerValues))
	return ahdWordAppend(doc, ahdWordBlock{
		Kind: "table", Headers: headerValues, Rows: grid, Merges: mergeValues, Align: align,
	})
}

// ahdWordValidateMerges rejects every malformed or geometrically impossible
// descriptor explicitly: nothing is silently clipped, normalized, or
// dropped. rowCount includes the rendered header row (row 0).
func ahdWordValidateMerges(merges *AhdList[*AhdList[int64]], rowCount, columnCount int) [][4]int {
	entries := merges.Snapshot()
	result := make([][4]int, 0, len(entries))
	covered := make(map[[2]int]bool)
	for _, entry := range entries {
		nonNullEntry := AhdNonNull(entry)
		values := nonNullEntry.Snapshot()
		if len(values) != 4 {
			ahdWordRaise("a table merge descriptor must have exactly four Int values: row, column, rowSpan, columnSpan")
		}
		row, column, rowSpan, columnSpan := values[0], values[1], values[2], values[3]
		if row < 0 || column < 0 {
			ahdWordRaise("a table merge row and column must not be negative")
		}
		if rowSpan < 1 || columnSpan < 1 {
			ahdWordRaise("a table merge rowSpan and columnSpan must be at least 1")
		}
		if rowSpan == 1 && columnSpan == 1 {
			ahdWordRaise("a 1x1 table merge is meaningless")
		}
		if row+rowSpan > int64(rowCount) || column+columnSpan > int64(columnCount) {
			ahdWordRaise("a table merge extends outside the table")
		}
		for r := row; r < row+rowSpan; r++ {
			for c := column; c < column+columnSpan; c++ {
				key := [2]int{int(r), int(c)}
				if covered[key] {
					ahdWordRaise("table merge regions overlap")
				}
				covered[key] = true
			}
		}
		result = append(result, [4]int{int(row), int(column), int(rowSpan), int(columnSpan)})
	}
	return result
}

const (
	ahdWordEMUPerCentimeter = 360000
	ahdWordEMUPerPixel96DPI = 9525
)

// AhdWordImage reads and embeds the image bytes immediately, so the produced
// Document (and the DOCX it eventually saves to) never depends on the source
// file surviving: moving or deleting it afterward changes nothing.
func AhdWordImage(doc AhdWordDocument, path string, size *AhdPair[string, float64]) AhdWordDocument {
	if path == "" {
		ahdWordRaise("image path must not be empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		ahdWordRaise("could not read image: " + err.Error())
	}
	format, naturalWidth, naturalHeight := ahdWordDecodeImage(data)
	widthEMU, heightEMU := ahdWordImageExtent(size, naturalWidth, naturalHeight)
	return ahdWordAppend(doc, ahdWordBlock{
		Kind: "image", Media: data, MediaExt: format, WidthEMU: widthEMU, HeightEMU: heightEMU,
	})
}

func ahdWordDecodeImage(data []byte) (format string, width, height int) {
	config, formatName, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		ahdWordRaise("unsupported image format: Word supports PNG and JPEG")
	}
	switch formatName {
	case "png":
		return "png", config.Width, config.Height
	case "jpeg":
		return "jpeg", config.Width, config.Height
	default:
		ahdWordRaise("unsupported image format: Word supports PNG and JPEG")
		return "", 0, 0
	}
}

// ahdWordImageExtent resolves the four size.md-documented cases: an empty
// Pair keeps the image's natural pixel size (at 96 DPI, the OOXML default
// anchor), one dimension alone preserves the source aspect ratio, and both
// dimensions are used exactly as given.
func ahdWordImageExtent(size *AhdPair[string, float64], naturalWidth, naturalHeight int) (int64, int64) {
	size.require()
	var width, height float64
	var hasWidth, hasHeight bool
	for _, key := range size.keys {
		switch key {
		case "width":
			hasWidth = true
			width = size.values[key]
		case "height":
			hasHeight = true
			height = size.values[key]
		default:
			ahdWordRaise("image size supports only width and height")
		}
	}
	if hasWidth && width <= 0 {
		ahdWordRaise("image width must be positive")
	}
	if hasHeight && height <= 0 {
		ahdWordRaise("image height must be positive")
	}
	if naturalWidth <= 0 || naturalHeight <= 0 {
		naturalWidth, naturalHeight = 1, 1
	}
	aspect := float64(naturalHeight) / float64(naturalWidth)
	switch {
	case hasWidth && hasHeight:
		return int64(width * ahdWordEMUPerCentimeter), int64(height * ahdWordEMUPerCentimeter)
	case hasWidth:
		return int64(width * ahdWordEMUPerCentimeter), int64(width * aspect * ahdWordEMUPerCentimeter)
	case hasHeight:
		return int64(height / aspect * ahdWordEMUPerCentimeter), int64(height * ahdWordEMUPerCentimeter)
	default:
		return int64(naturalWidth) * ahdWordEMUPerPixel96DPI, int64(naturalHeight) * ahdWordEMUPerPixel96DPI
	}
}

func AhdWordPageBreak(doc AhdWordDocument) AhdWordDocument {
	return ahdWordAppend(doc, ahdWordBlock{Kind: "pageBreak"})
}

// ---------------------------------------------------------------------------
// Word: reading accessors
// ---------------------------------------------------------------------------

func AhdWordText(doc AhdWordDocument) string {
	var lines []string
	for _, block := range ahdWordDecodeBlocks(doc) {
		switch block.Kind {
		case "heading", "paragraph":
			lines = append(lines, block.Text)
		}
	}
	return strings.Join(lines, "\n")
}

func AhdWordParagraphs(doc AhdWordDocument) *AhdList[string] {
	var texts []string
	for _, block := range ahdWordDecodeBlocks(doc) {
		if block.Kind == "paragraph" {
			texts = append(texts, block.Text)
		}
	}
	return AhdNewList(texts...)
}

func AhdWordHeadings(doc AhdWordDocument) *AhdList[string] {
	var texts []string
	for _, block := range ahdWordDecodeBlocks(doc) {
		if block.Kind == "heading" {
			texts = append(texts, block.Text)
		}
	}
	return AhdNewList(texts...)
}

func AhdWordTables(doc AhdWordDocument) *AhdList[*AhdList[*AhdList[string]]] {
	var tables []*AhdList[*AhdList[string]]
	for _, block := range ahdWordDecodeBlocks(doc) {
		if block.Kind != "table" {
			continue
		}
		rows := make([]*AhdList[string], 0, 1+len(block.Rows))
		rows = append(rows, AhdNewList(block.Headers...))
		for _, row := range block.Rows {
			rows = append(rows, AhdNewList(row...))
		}
		tables = append(tables, AhdNewList(rows...))
	}
	return AhdNewList(tables...)
}

// ---------------------------------------------------------------------------
// Word: DOCX package generation
// ---------------------------------------------------------------------------

const ahdWordNamespaces = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
	`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" ` +
	`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" ` +
	`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ` +
	`xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"`

// ahdWordEscapeXML escapes the five predefined XML entities and drops any
// control character XML 1.0 forbids outright, so arbitrary AhdCode String
// content (including raw "<", "&", or stray control bytes) always produces
// well-formed XML.
func ahdWordEscapeXML(text string) string {
	var b strings.Builder
	for _, r := range text {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		case r == '"':
			b.WriteString("&quot;")
		case r == '\'':
			b.WriteString("&apos;")
		case r == '\t' || r == '\n' || r == '\r':
			b.WriteRune(r)
		case r < 0x20:
			// Not legal in XML 1.0 even escaped; drop it rather than emit an
			// invalid package.
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func ahdWordAlignValue(align string) string {
	if align == "justify" {
		return "both"
	}
	return align
}

func ahdWordHeadingXML(block ahdWordBlock) string {
	style := "Heading" + strconv.Itoa(block.Level)
	return `<w:p><w:pPr><w:pStyle w:val="` + style + `"/></w:pPr><w:r><w:t xml:space="preserve">` +
		ahdWordEscapeXML(block.Text) + `</w:t></w:r></w:p>`
}

func ahdWordParagraphXML(block ahdWordBlock) string {
	var run strings.Builder
	run.WriteString(`<w:r>`)
	var properties strings.Builder
	if block.Bold {
		properties.WriteString(`<w:b/>`)
	}
	if block.Italic {
		properties.WriteString(`<w:i/>`)
	}
	if block.Underline {
		properties.WriteString(`<w:u w:val="single"/>`)
	}
	if properties.Len() > 0 {
		run.WriteString(`<w:rPr>` + properties.String() + `</w:rPr>`)
	}
	run.WriteString(`<w:t xml:space="preserve">` + ahdWordEscapeXML(block.Text) + `</w:t></w:r>`)
	return `<w:p><w:pPr><w:jc w:val="` + ahdWordAlignValue(block.Align) + `"/></w:pPr>` + run.String() + `</w:p>`
}

func ahdWordPageBreakXML() string {
	return `<w:p><w:r><w:br w:type="page"/></w:r></w:p>`
}

func ahdWordSectionPropertiesXML() string {
	return `<w:sectPr><w:pgSz w:w="12240" w:h="15840"/>` +
		`<w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="720" w:footer="720" w:gutter="0"/>` +
		`</w:sectPr>`
}

// ahdWordColumnWidth splits a fixed default content width (roughly a Letter
// page with one-inch margins, in twentieths of a point) evenly across a
// table's columns, so every generated table has an explicit, valid width.
func ahdWordColumnWidth(columnCount int) int {
	const contentWidth = 9350
	if columnCount <= 0 {
		return contentWidth
	}
	return contentWidth / columnCount
}

// ahdWordTableXML renders one table block, including gridSpan/vMerge for
// every merge. A merge's anchor cell (its top-left corner) is the only cell
// that ever carries the merged region's text; every other covered position
// contributes no separate <w:tc> (horizontal continuation) or one empty,
// vMerge-continuation <w:tc> (vertical continuation), matching how Word
// itself represents a merged region.
func ahdWordTableXML(block ahdWordBlock) string {
	columnCount := len(block.Headers)
	rowCount := 1 + len(block.Rows)
	anchorMerge := make([][]int, rowCount)
	consumed := make([][]bool, rowCount)
	for r := 0; r < rowCount; r++ {
		anchorMerge[r] = make([]int, columnCount)
		consumed[r] = make([]bool, columnCount)
		for c := range anchorMerge[r] {
			anchorMerge[r][c] = -1
		}
	}
	for mergeIndex, merge := range block.Merges {
		row, column, rowSpan, columnSpan := merge[0], merge[1], merge[2], merge[3]
		for r := row; r < row+rowSpan; r++ {
			anchorMerge[r][column] = mergeIndex
			for c := column + 1; c < column+columnSpan; c++ {
				consumed[r][c] = true
			}
		}
	}
	var b strings.Builder
	b.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="0" w:type="auto"/><w:jc w:val="` + block.Align + `"/><w:tblBorders>`)
	for _, edge := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
		b.WriteString(`<w:` + edge + ` w:val="single" w:sz="4" w:space="0" w:color="auto"/>`)
	}
	b.WriteString(`</w:tblBorders></w:tblPr><w:tblGrid>`)
	columnWidth := ahdWordColumnWidth(columnCount)
	for i := 0; i < columnCount; i++ {
		b.WriteString(`<w:gridCol w:w="` + strconv.Itoa(columnWidth) + `"/>`)
	}
	b.WriteString(`</w:tblGrid>`)
	for r := 0; r < rowCount; r++ {
		var rowText []string
		if r == 0 {
			rowText = block.Headers
		} else {
			rowText = block.Rows[r-1]
		}
		b.WriteString(`<w:tr>`)
		for c := 0; c < columnCount; c++ {
			if consumed[r][c] {
				continue
			}
			mergeIndex := anchorMerge[r][c]
			columnSpan, rowSpan, mergeRow := 1, 1, r
			if mergeIndex >= 0 {
				merge := block.Merges[mergeIndex]
				mergeRow, rowSpan, columnSpan = merge[0], merge[2], merge[3]
			}
			var properties strings.Builder
			properties.WriteString(`<w:tcW w:w="0" w:type="auto"/>`)
			if columnSpan > 1 {
				properties.WriteString(`<w:gridSpan w:val="` + strconv.Itoa(columnSpan) + `"/>`)
			}
			if rowSpan > 1 {
				if r == mergeRow {
					properties.WriteString(`<w:vMerge w:val="restart"/>`)
				} else {
					properties.WriteString(`<w:vMerge/>`)
				}
			}
			paragraph := `<w:p>`
			// A vertical-merge continuation cell renders no text: the anchor
			// row already carries the merged region's content.
			if mergeIndex < 0 || r == mergeRow {
				cellText := ""
				if c < len(rowText) {
					cellText = rowText[c]
				}
				paragraph += `<w:r><w:t xml:space="preserve">` + ahdWordEscapeXML(cellText) + `</w:t></w:r>`
			}
			paragraph += `</w:p>`
			b.WriteString(`<w:tc><w:tcPr>` + properties.String() + `</w:tcPr>` + paragraph + `</w:tc>`)
		}
		b.WriteString(`</w:tr>`)
	}
	b.WriteString(`</w:tbl>`)
	return b.String()
}

func ahdWordImageXML(block ahdWordBlock, id int, relID string) string {
	name := ahdWordEscapeXML("Picture " + strconv.Itoa(id))
	cx, cy := strconv.FormatInt(block.WidthEMU, 10), strconv.FormatInt(block.HeightEMU, 10)
	idText := strconv.Itoa(id)
	return `<w:p><w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0">` +
		`<wp:extent cx="` + cx + `" cy="` + cy + `"/>` +
		`<wp:effectExtent l="0" t="0" r="0" b="0"/>` +
		`<wp:docPr id="` + idText + `" name="` + name + `"/>` +
		`<wp:cNvGraphicFramePr><a:graphicFrameLocks noChangeAspect="1"/></wp:cNvGraphicFramePr>` +
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
		`<pic:pic><pic:nvPicPr><pic:cNvPr id="` + idText + `" name="` + name + `"/><pic:cNvPicPr/></pic:nvPicPr>` +
		`<pic:blipFill><a:blip r:embed="` + relID + `"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>` +
		`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="` + cx + `" cy="` + cy + `"/></a:xfrm>` +
		`<a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic>` +
		`</a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>`
}

func ahdWordDocumentXML(blocks []ahdWordBlock, imageRelIDs []string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:document ` + ahdWordNamespaces + `><w:body>`)
	imageIndex := 0
	for _, block := range blocks {
		switch block.Kind {
		case "heading":
			b.WriteString(ahdWordHeadingXML(block))
		case "paragraph":
			b.WriteString(ahdWordParagraphXML(block))
		case "table":
			b.WriteString(ahdWordTableXML(block))
		case "image":
			b.WriteString(ahdWordImageXML(block, imageIndex+1, imageRelIDs[imageIndex]))
			imageIndex++
		case "pageBreak":
			b.WriteString(ahdWordPageBreakXML())
		}
	}
	b.WriteString(ahdWordSectionPropertiesXML())
	b.WriteString(`</w:body></w:document>`)
	return b.String()
}

// ahdWordStylesXML defines Normal and Heading1..Heading6 with a deterministic
// decreasing size scale, so document.xml can reference real WordprocessingML
// heading styles rather than hand-built bold paragraphs.
func ahdWordStylesXML() string {
	sizes := [6]int{32, 28, 26, 24, 22, 22}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	b.WriteString(`<w:docDefaults><w:rPrDefault><w:rPr><w:sz w:val="22"/><w:szCs w:val="22"/></w:rPr></w:rPrDefault></w:docDefaults>`)
	b.WriteString(`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/></w:style>`)
	for level := 1; level <= 6; level++ {
		id := "Heading" + strconv.Itoa(level)
		b.WriteString(`<w:style w:type="paragraph" w:styleId="` + id + `"><w:name w:val="heading ` + strconv.Itoa(level) + `"/>` +
			`<w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/>` +
			`<w:pPr><w:spacing w:before="240" w:after="120"/><w:outlineLvl w:val="` + strconv.Itoa(level-1) + `"/></w:pPr>` +
			`<w:rPr><w:b/><w:sz w:val="` + strconv.Itoa(sizes[level-1]) + `"/></w:rPr></w:style>`)
	}
	b.WriteString(`</w:styles>`)
	return b.String()
}

func ahdWordContentTypesXML(hasPNG, hasJPEG bool) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	b.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`)
	b.WriteString(`<Default Extension="xml" ContentType="application/xml"/>`)
	if hasPNG {
		b.WriteString(`<Default Extension="png" ContentType="image/png"/>`)
	}
	if hasJPEG {
		b.WriteString(`<Default Extension="jpeg" ContentType="image/jpeg"/>`)
	}
	b.WriteString(`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>`)
	b.WriteString(`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>`)
	b.WriteString(`</Types>`)
	return b.String()
}

const ahdWordPackageRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
	`</Relationships>`

// ahdWordDocumentRelsXML assigns relationship IDs in a fixed, deterministic
// order: rId1 is always the styles part, then one rIdN per image in document
// order.
func ahdWordDocumentRelsXML(extensions []string) (string, []string) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	b.WriteString(`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`)
	relIDs := make([]string, len(extensions))
	for index, extension := range extensions {
		relID := "rId" + strconv.Itoa(2+index)
		relIDs[index] = relID
		b.WriteString(`<Relationship Id="` + relID + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" ` +
			`Target="media/image` + strconv.Itoa(index+1) + `.` + extension + `"/>`)
	}
	b.WriteString(`</Relationships>`)
	return b.String(), relIDs
}

// ahdWordBuildPackage assembles the complete DOCX ZIP in a fixed member
// order, so identical Document content always produces byte-identical
// package bytes.
func ahdWordBuildPackage(blocks []ahdWordBlock) []byte {
	var images []ahdWordBlock
	for _, block := range blocks {
		if block.Kind == "image" {
			images = append(images, block)
		}
	}
	extensions := make([]string, len(images))
	hasPNG, hasJPEG := false, false
	for index, block := range images {
		extensions[index] = block.MediaExt
		hasPNG = hasPNG || block.MediaExt == "png"
		hasJPEG = hasJPEG || block.MediaExt == "jpeg"
	}
	documentRelsXML, relIDs := ahdWordDocumentRelsXML(extensions)
	documentXML := ahdWordDocumentXML(blocks, relIDs)

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	ahdWordWriteEntry(writer, "[Content_Types].xml", []byte(ahdWordContentTypesXML(hasPNG, hasJPEG)))
	ahdWordWriteEntry(writer, "_rels/.rels", []byte(ahdWordPackageRelsXML))
	ahdWordWriteEntry(writer, "word/document.xml", []byte(documentXML))
	ahdWordWriteEntry(writer, "word/_rels/document.xml.rels", []byte(documentRelsXML))
	ahdWordWriteEntry(writer, "word/styles.xml", []byte(ahdWordStylesXML()))
	for index, block := range images {
		name := "word/media/image" + strconv.Itoa(index+1) + "." + block.MediaExt
		ahdWordWriteEntry(writer, name, block.Media)
	}
	if err := writer.Close(); err != nil {
		ahdWordRaise("could not assemble the DOCX package: " + err.Error())
	}
	return buffer.Bytes()
}

func ahdWordWriteEntry(writer *zip.Writer, name string, content []byte) {
	part, err := writer.Create(name)
	if err != nil {
		ahdWordRaise("could not assemble the DOCX package: " + err.Error())
	}
	if _, err := part.Write(content); err != nil {
		ahdWordRaise("could not assemble the DOCX package: " + err.Error())
	}
}

// ahdWordValidatePackage is the last check before a generated package is
// allowed to replace a save destination: it must open as a ZIP, carry every
// required part, and word/document.xml must be non-empty, well-formed XML.
func ahdWordValidatePackage(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("package is empty")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("not a valid ZIP package: %w", err)
	}
	required := map[string]bool{"[Content_Types].xml": false, "_rels/.rels": false, "word/document.xml": false}
	var documentXML []byte
	for _, file := range reader.File {
		if _, known := required[file.Name]; known {
			required[file.Name] = true
		}
		if file.Name != "word/document.xml" {
			continue
		}
		opened, err := file.Open()
		if err != nil {
			return fmt.Errorf("could not read word/document.xml: %w", err)
		}
		documentXML, err = io.ReadAll(opened)
		closeErr := opened.Close()
		if err != nil {
			return fmt.Errorf("could not read word/document.xml: %w", err)
		}
		if closeErr != nil {
			return closeErr
		}
	}
	for name, present := range required {
		if !present {
			return fmt.Errorf("package is missing %s", name)
		}
	}
	if len(documentXML) == 0 {
		return fmt.Errorf("word/document.xml is empty")
	}
	decoder := xml.NewDecoder(bytes.NewReader(documentXML))
	for {
		if _, err := decoder.Token(); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("word/document.xml does not parse: %w", err)
		}
	}
	return nil
}

// AhdWordSave renders the Document to a real DOCX package and publishes it
// atomically: the destination is written only after the generated package
// has already passed AhdWordValidatePackage, and a failed save never
// disturbs a file that was already there.
func AhdWordSave(doc AhdWordDocument, path string) {
	if path == "" {
		ahdWordRaise("save path must not be empty")
	}
	if !strings.EqualFold(filepath.Ext(path), ".docx") {
		ahdWordRaise("Word.save only supports a .docx destination")
	}
	blocks := ahdWordDecodeBlocks(doc)
	if message := ahdWordRaggedTableMessage(blocks); message != "" {
		ahdWordRaise(message)
	}
	packageBytes := ahdWordBuildPackage(blocks)
	if err := ahdWordValidatePackage(packageBytes); err != nil {
		ahdWordRaise("failed to produce a valid DOCX: " + err.Error())
	}
	if err := ahdWordPublish(packageBytes, path); err != nil {
		ahdWordRaise("could not write the destination file: " + err.Error())
	}
}

// ahdWordPublish stages the package in the destination's own directory (so
// the final rename is on one filesystem, hence atomic) and only then renames
// it over the real destination.
func ahdWordPublish(data []byte, output string) error {
	absolute, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	directory := filepath.Dir(absolute)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".ahdcode-word-output-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	_, writeError := temporary.Write(data)
	syncError := temporary.Sync()
	closeError := temporary.Close()
	for _, candidate := range []error{writeError, syncError, closeError} {
		if candidate != nil {
			return candidate
		}
	}
	return os.Rename(temporaryPath, absolute)
}

// ---------------------------------------------------------------------------
// Word: DOCX reading
// ---------------------------------------------------------------------------
//
// Reading recovers ordinary paragraph text, heading text, and basic table
// cell text from a real DOCX package. It is not a fidelity-preserving
// editor: formatting it does not understand (custom fonts, unknown run
// properties, page layout, headers/footers, comments, unrecognized styles)
// is safely ignored rather than treated as a failure. Only a structurally
// broken package or a missing/unparseable word/document.xml raises
// WordError. Reading never touches the network: every relationship target,
// including an external one, is simply never followed.

const (
	ahdWordMaxArchiveSize       = 64 * 1024 * 1024  // total input file size
	ahdWordMaxEntryUncompressed = 32 * 1024 * 1024  // any single ZIP member, decompressed
	ahdWordMaxTotalUncompressed = 128 * 1024 * 1024 // every member's declared size, summed
	ahdWordMaxCompressionRatio  = 200               // reject a member that inflates more than 200x
	ahdWordMaxEntries           = 2000              // reject a package with an unreasonable member count
)

func AhdWordRead(path string) AhdWordDocument {
	if path == "" {
		ahdWordRaise("read path must not be empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		ahdWordRaise("could not open the DOCX file: " + err.Error())
	}
	if !info.Mode().IsRegular() {
		ahdWordRaise("the DOCX path is not a regular file")
	}
	if info.Size() > ahdWordMaxArchiveSize {
		ahdWordRaise("the DOCX file is larger than the supported limit")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		ahdWordRaise("could not read the DOCX file: " + err.Error())
	}
	documentXML := ahdWordExtractDocumentXML(data)
	blocks := ahdWordParseDocumentXML(documentXML)
	doc := AhdWordDocument{}
	for _, block := range blocks {
		doc = ahdWordAppend(doc, block)
	}
	return doc
}

// ahdWordExtractDocumentXML opens the ZIP package entirely in memory,
// rejects every unsafe or oversized member up front (path traversal,
// duplicates, declared sizes and compression ratios past the bounded
// limits), and returns only the bytes of word/document.xml, itself capped by
// a hard read limit that ignores whatever the ZIP header claims.
func ahdWordExtractDocumentXML(data []byte) []byte {
	if len(data) == 0 {
		ahdWordRaise("the DOCX file is empty")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		ahdWordRaise("not a valid DOCX package: " + err.Error())
	}
	if len(reader.File) == 0 {
		ahdWordRaise("the DOCX package has no members")
	}
	if len(reader.File) > ahdWordMaxEntries {
		ahdWordRaise("the DOCX package has too many members")
	}
	seen := make(map[string]bool, len(reader.File))
	var documentEntry *zip.File
	var totalUncompressed uint64
	for _, file := range reader.File {
		name := file.Name
		if name == "" || strings.HasPrefix(name, "/") || filepath.IsAbs(name) || strings.Contains(name, "..") {
			ahdWordRaise("the DOCX package contains an unsafe member path")
		}
		if seen[name] {
			ahdWordRaise("the DOCX package has a duplicate member")
		}
		seen[name] = true
		if file.UncompressedSize64 > ahdWordMaxEntryUncompressed {
			ahdWordRaise("the DOCX package has a member that is too large")
		}
		if file.CompressedSize64 > 0 && file.UncompressedSize64/file.CompressedSize64 > ahdWordMaxCompressionRatio {
			ahdWordRaise("the DOCX package has a member with an unreasonable compression ratio")
		}
		totalUncompressed += file.UncompressedSize64
		if totalUncompressed > ahdWordMaxTotalUncompressed {
			ahdWordRaise("the DOCX package is larger than the supported limit once decompressed")
		}
		if name == "word/document.xml" {
			documentEntry = file
		}
	}
	if documentEntry == nil {
		ahdWordRaise("the DOCX package has no word/document.xml")
	}
	opened, err := documentEntry.Open()
	if err != nil {
		ahdWordRaise("could not read word/document.xml: " + err.Error())
	}
	defer opened.Close()
	content, err := io.ReadAll(io.LimitReader(opened, ahdWordMaxEntryUncompressed+1))
	if err != nil {
		ahdWordRaise("could not read word/document.xml: " + err.Error())
	}
	if int64(len(content)) > ahdWordMaxEntryUncompressed {
		ahdWordRaise("word/document.xml is larger than the supported limit")
	}
	if len(content) == 0 {
		ahdWordRaise("word/document.xml is empty")
	}
	return content
}

var ahdWordHeadingStylePattern = regexp.MustCompile(`(?i)^heading\s*([1-6])$`)

func ahdWordHeadingLevel(styleID string) int {
	match := ahdWordHeadingStylePattern.FindStringSubmatch(strings.TrimSpace(styleID))
	if match == nil {
		return 0
	}
	level, _ := strconv.Atoi(match[1])
	return level
}

func ahdWordAttr(element xml.StartElement, local string) string {
	for _, attr := range element.Attr {
		if attr.Name.Local == local {
			return attr.Value
		}
	}
	return ""
}

// ahdWordParseDocumentXML walks the flat token stream once, dispatching each
// top-level <w:p> or <w:tbl> inside <w:body> to a dedicated subtree parser.
// Every other element - page layout, bookmarks, headers/footers references,
// comments ranges, unknown run properties - is simply never matched, which
// is what lets an unrelated, unsupported feature pass through instead of
// failing the read.
func ahdWordParseDocumentXML(data []byte) []ahdWordBlock {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var blocks []ahdWordBlock
	inBody := false
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			ahdWordRaise("word/document.xml does not parse: " + err.Error())
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "body":
			inBody = true
		case "p":
			if inBody {
				block := ahdWordParseParagraphElement(decoder)
				if block.Kind != "" {
					blocks = append(blocks, block)
				}
			}
		case "tbl":
			if inBody {
				blocks = append(blocks, ahdWordParseTableElement(decoder))
			}
		}
	}
	return blocks
}

// ahdWordParseParagraphElement consumes tokens up to and including the
// matching </w:p>, tracking a small local element stack so that only
// character data whose immediate parent is <w:t> is treated as text -
// whitespace between sibling elements in a pretty-printed foreign document
// is never mistaken for content.
func ahdWordParseParagraphElement(decoder *xml.Decoder) ahdWordBlock {
	styleID := ""
	var text strings.Builder
	var stack []string
	hasContent := false
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			ahdWordRaise("word/document.xml does not parse: " + err.Error())
		}
		switch element := token.(type) {
		case xml.StartElement:
			stack = append(stack, element.Name.Local)
			if element.Name.Local == "pStyle" {
				styleID = ahdWordAttr(element, "val")
			}
			if element.Name.Local == "t" {
				hasContent = true
			}
			if element.Name.Local == "tab" {
				hasContent = true
				text.WriteByte('\t')
			}
			if element.Name.Local == "br" && ahdWordAttr(element, "type") != "page" {
				hasContent = true
				text.WriteByte('\n')
			}
		case xml.EndElement:
			if len(stack) == 0 {
				return ahdWordFinishParagraph(styleID, text.String(), hasContent)
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 && stack[len(stack)-1] == "t" {
				text.Write(element)
			}
		}
	}
	return ahdWordFinishParagraph(styleID, text.String(), hasContent)
}

func ahdWordFinishParagraph(styleID, text string, hasContent bool) ahdWordBlock {
	if !hasContent {
		return ahdWordBlock{}
	}
	if level := ahdWordHeadingLevel(styleID); level > 0 {
		return ahdWordBlock{Kind: "heading", Text: text, Level: level}
	}
	return ahdWordBlock{Kind: "paragraph", Text: text, Align: "left"}
}

// ahdWordParseTableElement consumes tokens up to and including the matching
// </w:tbl>, collecting one logical column per <w:tc> encountered, grouped by
// <w:tr>. A <w:tc> carrying <w:gridSpan w:val="N"/> expands to N logical
// columns at the position where the span occurred - the cell's own text
// followed by N-1 empty columns - so a merged header still lines up with the
// unmerged data columns beneath it. ahdWordFinishTable then widens any row
// that is still short (a defensive fallback for markup whose spans do not
// fully account for the table's true width) so every row stays rectangular
// and no cell text is ever dropped.
func ahdWordParseTableElement(decoder *xml.Decoder) ahdWordBlock {
	var rows [][]string
	var currentRow []string
	var cellText *strings.Builder
	cellSpan := 1
	var stack []string
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			ahdWordRaise("word/document.xml does not parse: " + err.Error())
		}
		switch element := token.(type) {
		case xml.StartElement:
			stack = append(stack, element.Name.Local)
			switch element.Name.Local {
			case "tr":
				currentRow = nil
			case "tc":
				cellText = &strings.Builder{}
				cellSpan = 1
			case "gridSpan":
				if cellText != nil {
					if value, err := strconv.Atoi(ahdWordAttr(element, "val")); err == nil && value > 1 {
						cellSpan = value
					}
				}
			}
		case xml.EndElement:
			if len(stack) == 0 {
				return ahdWordFinishTable(rows)
			}
			closed := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if closed == "tc" && cellText != nil {
				currentRow = append(currentRow, cellText.String())
				for extra := 1; extra < cellSpan; extra++ {
					currentRow = append(currentRow, "")
				}
				cellText = nil
				cellSpan = 1
			}
			if closed == "tr" {
				rows = append(rows, currentRow)
			}
		case xml.CharData:
			if cellText != nil && len(stack) > 0 && stack[len(stack)-1] == "t" {
				cellText.Write(element)
			}
		}
	}
	return ahdWordFinishTable(rows)
}

func ahdWordFinishTable(rows [][]string) ahdWordBlock {
	if len(rows) == 0 {
		return ahdWordBlock{Kind: "table", Headers: []string{}, Align: "left"}
	}
	width := 0
	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}
	widened := make([][]string, len(rows))
	for index, row := range rows {
		if len(row) == width {
			widened[index] = row
			continue
		}
		padded := make([]string, width)
		copy(padded, row)
		widened[index] = padded
	}
	return ahdWordBlock{Kind: "table", Headers: widened[0], Rows: widened[1:], Align: "left"}
}

// ahdWordRaggedTableMessage reports the first table block whose rows are not
// all the same logical width, or "" if every table block is rectangular.
// ahdWordFinishTable always produces rectangular blocks and AhdWordTable
// rejects mismatched row widths at construction time, so this exists purely
// as a save-time defensive invariant: it must never be possible to silently
// truncate a ragged table into a shorter DOCX row.
func ahdWordRaggedTableMessage(blocks []ahdWordBlock) string {
	for _, block := range blocks {
		if block.Kind != "table" {
			continue
		}
		width := len(block.Headers)
		for _, row := range block.Rows {
			if len(row) != width {
				return "a table has rows with different widths and cannot be saved"
			}
		}
	}
	return ""
}

// --- JSON standard module ---
//
// A JSONValue is represented at rest as its own canonical, compact JSON
// text (never as a Go interface{} tree): every JSON module function that
// returns a JSONValue hands back that canonical text, which the generated
// program then wraps in one JSONValue instance through the class helper
// emitted by internal/backend/golang/json_module.go. Composing values
// (array/object construction) is therefore plain string concatenation of
// already-canonical child text, and every accessor works by re-parsing that
// text on demand. This mirrors the Word module's own "hidden String field,
// reparsed by helpers" pattern for Document.
//
// Parsing is hand-written rather than delegated to encoding/json because the
// public contract needs three things encoding/json's interface{} decoding
// does not give: Int/Real are distinct kinds (not "every number is
// float64"), Object key order is preserved, and duplicate Object keys are a
// hard parse error rather than last-write-wins.

const (
	// ahdJSONMaxInputBytes bounds JSON.parse/JSON.read input size.
	ahdJSONMaxInputBytes = 8 * 1024 * 1024
	// ahdJSONMaxDepth bounds Array/Object nesting to defend against
	// pathological recursion on adversarial input.
	ahdJSONMaxDepth = 256
)

// AhdJSONEntry is one ordered Object member, used to build a JSONValue from
// a Pair<String, JSONValue> in AhdJSONObject.
type AhdJSONEntry struct {
	Key  string
	Text string
}

// ahdJSONNode is the parsed interchange form of one JSON value, used only
// transiently while parsing or pretty-printing; a JSONValue's resting
// representation is always its canonical text, never this tree.
type ahdJSONNode struct {
	kind   string
	flag   bool
	number int64
	real   float64
	text   string
	items  []ahdJSONNode
	keys   []string
	values map[string]ahdJSONNode
}

// ---------------------------------------------------------------------------
// Parsing
// ---------------------------------------------------------------------------

type ahdJSONParser struct {
	class  *AhdClass
	source string
	pos    int
}

func ahdJSONParseDocument(class *AhdClass, source string) ahdJSONNode {
	if len(source) > ahdJSONMaxInputBytes {
		AhdRaiseClass(class, "JSON input is larger than the supported limit")
	}
	if !utf8.ValidString(source) {
		AhdRaiseClass(class, "JSON input is not valid UTF-8")
	}
	parser := &ahdJSONParser{class: class, source: source}
	parser.skipWhitespace()
	if parser.pos >= len(parser.source) {
		AhdRaiseClass(class, "JSON input is empty")
	}
	node := parser.parseValue(0)
	parser.skipWhitespace()
	if parser.pos != len(parser.source) {
		AhdRaiseClass(class, "JSON input has trailing content after its value")
	}
	return node
}

func (parser *ahdJSONParser) fail(message string) {
	AhdRaiseClass(parser.class, message)
}

func (parser *ahdJSONParser) skipWhitespace() {
	for parser.pos < len(parser.source) {
		switch parser.source[parser.pos] {
		case ' ', '\t', '\n', '\r':
			parser.pos++
		default:
			return
		}
	}
}

func (parser *ahdJSONParser) parseValue(depth int) ahdJSONNode {
	if depth > ahdJSONMaxDepth {
		parser.fail("JSON input exceeds the maximum supported nesting depth")
	}
	parser.skipWhitespace()
	if parser.pos >= len(parser.source) {
		parser.fail("JSON input ends where a value was expected")
	}
	switch character := parser.source[parser.pos]; {
	case character == '{':
		return parser.parseObject(depth)
	case character == '[':
		return parser.parseArray(depth)
	case character == '"':
		return ahdJSONNode{kind: "String", text: parser.parseString()}
	case character == 't':
		parser.expectLiteral("true")
		return ahdJSONNode{kind: "Bool", flag: true}
	case character == 'f':
		parser.expectLiteral("false")
		return ahdJSONNode{kind: "Bool", flag: false}
	case character == 'n':
		parser.expectLiteral("null")
		return ahdJSONNode{kind: "Null"}
	case character == '-' || (character >= '0' && character <= '9'):
		return parser.parseNumber()
	default:
		parser.fail("JSON input has an unexpected character")
		return ahdJSONNode{}
	}
}

func (parser *ahdJSONParser) expectLiteral(literal string) {
	if !strings.HasPrefix(parser.source[parser.pos:], literal) {
		parser.fail("JSON input has an invalid literal")
	}
	parser.pos += len(literal)
}

func (parser *ahdJSONParser) parseObject(depth int) ahdJSONNode {
	parser.pos++ // consume '{'
	node := ahdJSONNode{kind: "Object", values: make(map[string]ahdJSONNode)}
	parser.skipWhitespace()
	if parser.pos < len(parser.source) && parser.source[parser.pos] == '}' {
		parser.pos++
		return node
	}
	for {
		parser.skipWhitespace()
		if parser.pos >= len(parser.source) || parser.source[parser.pos] != '"' {
			parser.fail("JSON object key must be a String")
		}
		key := parser.parseString()
		parser.skipWhitespace()
		if parser.pos >= len(parser.source) || parser.source[parser.pos] != ':' {
			parser.fail("JSON object is missing ':' after a key")
		}
		parser.pos++
		value := parser.parseValue(depth + 1)
		if _, duplicate := node.values[key]; duplicate {
			parser.fail("JSON object has a duplicate key")
		}
		node.keys = append(node.keys, key)
		node.values[key] = value
		parser.skipWhitespace()
		if parser.pos >= len(parser.source) {
			parser.fail("JSON object is not closed")
		}
		switch parser.source[parser.pos] {
		case ',':
			parser.pos++
			continue
		case '}':
			parser.pos++
			return node
		default:
			parser.fail("JSON object is missing ',' or '}'")
		}
	}
}

func (parser *ahdJSONParser) parseArray(depth int) ahdJSONNode {
	parser.pos++ // consume '['
	node := ahdJSONNode{kind: "Array"}
	parser.skipWhitespace()
	if parser.pos < len(parser.source) && parser.source[parser.pos] == ']' {
		parser.pos++
		return node
	}
	for {
		node.items = append(node.items, parser.parseValue(depth+1))
		parser.skipWhitespace()
		if parser.pos >= len(parser.source) {
			parser.fail("JSON array is not closed")
		}
		switch parser.source[parser.pos] {
		case ',':
			parser.pos++
			continue
		case ']':
			parser.pos++
			return node
		default:
			parser.fail("JSON array is missing ',' or ']'")
		}
	}
}

func (parser *ahdJSONParser) parseString() string {
	parser.pos++ // consume opening quote
	var builder strings.Builder
	for {
		if parser.pos >= len(parser.source) {
			parser.fail("JSON String is not closed")
		}
		character := parser.source[parser.pos]
		switch {
		case character == '"':
			parser.pos++
			return builder.String()
		case character == '\\':
			parser.pos++
			if parser.pos >= len(parser.source) {
				parser.fail("JSON String has an incomplete escape sequence")
			}
			switch parser.source[parser.pos] {
			case '"':
				builder.WriteByte('"')
				parser.pos++
			case '\\':
				builder.WriteByte('\\')
				parser.pos++
			case '/':
				builder.WriteByte('/')
				parser.pos++
			case 'b':
				builder.WriteByte('\b')
				parser.pos++
			case 'f':
				builder.WriteByte('\f')
				parser.pos++
			case 'n':
				builder.WriteByte('\n')
				parser.pos++
			case 'r':
				builder.WriteByte('\r')
				parser.pos++
			case 't':
				builder.WriteByte('\t')
				parser.pos++
			case 'u':
				builder.WriteRune(parser.parseUnicodeEscape())
			default:
				parser.fail("JSON String has an invalid escape sequence")
			}
		case character < 0x20:
			parser.fail("JSON String contains an unescaped control character")
		default:
			_, width := utf8.DecodeRuneInString(parser.source[parser.pos:])
			builder.WriteString(parser.source[parser.pos : parser.pos+width])
			parser.pos += width
		}
	}
}

func (parser *ahdJSONParser) parseUnicodeEscape() rune {
	high := parser.parseHex4()
	if utf16.IsSurrogate(rune(high)) {
		if strings.HasPrefix(parser.source[parser.pos:], `\u`) {
			mark := parser.pos
			parser.pos += 2
			low := parser.parseHex4()
			combined := utf16.DecodeRune(rune(high), rune(low))
			if combined != utf8.RuneError {
				return combined
			}
			parser.pos = mark
		}
		return utf8.RuneError
	}
	return rune(high)
}

func (parser *ahdJSONParser) parseHex4() uint16 {
	parser.pos++ // consume 'u'
	if parser.pos+4 > len(parser.source) {
		parser.fail("JSON String has an incomplete \\u escape")
	}
	digits := parser.source[parser.pos : parser.pos+4]
	value, err := strconv.ParseUint(digits, 16, 32)
	if err != nil {
		parser.fail("JSON String has an invalid \\u escape")
	}
	parser.pos += 4
	return uint16(value)
}

func (parser *ahdJSONParser) parseNumber() ahdJSONNode {
	start := parser.pos
	if parser.pos < len(parser.source) && parser.source[parser.pos] == '-' {
		parser.pos++
	}
	if parser.pos >= len(parser.source) || parser.source[parser.pos] < '0' || parser.source[parser.pos] > '9' {
		parser.fail("JSON input has a malformed number")
	}
	if parser.source[parser.pos] == '0' {
		parser.pos++
	} else {
		for parser.pos < len(parser.source) && parser.source[parser.pos] >= '0' && parser.source[parser.pos] <= '9' {
			parser.pos++
		}
	}
	isReal := false
	if parser.pos < len(parser.source) && parser.source[parser.pos] == '.' {
		isReal = true
		parser.pos++
		digitStart := parser.pos
		for parser.pos < len(parser.source) && parser.source[parser.pos] >= '0' && parser.source[parser.pos] <= '9' {
			parser.pos++
		}
		if parser.pos == digitStart {
			parser.fail("JSON number has a malformed fraction")
		}
	}
	if parser.pos < len(parser.source) && (parser.source[parser.pos] == 'e' || parser.source[parser.pos] == 'E') {
		isReal = true
		parser.pos++
		if parser.pos < len(parser.source) && (parser.source[parser.pos] == '+' || parser.source[parser.pos] == '-') {
			parser.pos++
		}
		digitStart := parser.pos
		for parser.pos < len(parser.source) && parser.source[parser.pos] >= '0' && parser.source[parser.pos] <= '9' {
			parser.pos++
		}
		if parser.pos == digitStart {
			parser.fail("JSON number has a malformed exponent")
		}
	}
	lexeme := parser.source[start:parser.pos]
	if !isReal {
		value, err := strconv.ParseInt(lexeme, 10, 64)
		if err != nil {
			parser.fail("JSON integer literal " + lexeme + " does not fit AhdCode's Int range")
		}
		return ahdJSONNode{kind: "Int", number: value}
	}
	value, err := strconv.ParseFloat(lexeme, 64)
	if err != nil || math.IsInf(value, 0) {
		parser.fail("JSON real literal " + lexeme + " is out of range")
	}
	return ahdJSONNode{kind: "Real", real: value}
}

// ---------------------------------------------------------------------------
// Serialization
// ---------------------------------------------------------------------------

func ahdJSONFormatReal(value float64) string {
	text := strconv.FormatFloat(value, 'g', -1, 64)
	if !strings.ContainsAny(text, ".eE") {
		text += ".0"
	}
	return text
}

func ahdJSONEncodeString(value string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, character := range value {
		switch {
		case character == '"':
			builder.WriteString(`\"`)
		case character == '\\':
			builder.WriteString(`\\`)
		case character == '\n':
			builder.WriteString(`\n`)
		case character == '\r':
			builder.WriteString(`\r`)
		case character == '\t':
			builder.WriteString(`\t`)
		case character == '\b':
			builder.WriteString(`\b`)
		case character == '\f':
			builder.WriteString(`\f`)
		case character < 0x20:
			builder.WriteString(`\u`)
			builder.WriteString(strconv.FormatInt(int64(character), 16))
		default:
			builder.WriteRune(character)
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

func ahdJSONStringifyNode(node ahdJSONNode, pretty bool, depth int) string {
	indent := func(level int) string {
		if !pretty {
			return ""
		}
		return "\n" + strings.Repeat("  ", level)
	}
	switch node.kind {
	case "Null":
		return "null"
	case "Bool":
		if node.flag {
			return "true"
		}
		return "false"
	case "Int":
		return strconv.FormatInt(node.number, 10)
	case "Real":
		return ahdJSONFormatReal(node.real)
	case "String":
		return ahdJSONEncodeString(node.text)
	case "Array":
		if len(node.items) == 0 {
			return "[]"
		}
		var builder strings.Builder
		builder.WriteByte('[')
		for index, item := range node.items {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(indent(depth + 1))
			builder.WriteString(ahdJSONStringifyNode(item, pretty, depth+1))
		}
		builder.WriteString(indent(depth))
		builder.WriteByte(']')
		return builder.String()
	case "Object":
		if len(node.keys) == 0 {
			return "{}"
		}
		var builder strings.Builder
		builder.WriteByte('{')
		for index, key := range node.keys {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(indent(depth + 1))
			builder.WriteString(ahdJSONEncodeString(key))
			builder.WriteByte(':')
			if pretty {
				builder.WriteByte(' ')
			}
			builder.WriteString(ahdJSONStringifyNode(node.values[key], pretty, depth+1))
		}
		builder.WriteString(indent(depth))
		builder.WriteByte('}')
		return builder.String()
	}
	return "null"
}

func ahdJSONCanonicalText(node ahdJSONNode) string {
	return ahdJSONStringifyNode(node, false, 0)
}

// ---------------------------------------------------------------------------
// Public JSON module functions
// ---------------------------------------------------------------------------

// AhdJSONParse parses source and returns the resulting JSONValue's
// canonical compact text.
func AhdJSONParse(class *AhdClass, source string) string {
	return ahdJSONCanonicalText(ahdJSONParseDocument(class, source))
}

// AhdJSONRead reads path and parses its content the same way AhdJSONParse
// does. Unlike CSV/File, JSON owns its own error identity end to end: a
// missing or unreadable file raises JSONError, not FileError.
func AhdJSONRead(class *AhdClass, path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		AhdRaiseClass(class, "could not read the JSON file: "+err.Error())
	}
	return AhdJSONParse(class, string(content))
}

func AhdJSONNull() string { return "null" }

func AhdJSONFromBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func AhdJSONFromInt(value int64) string { return strconv.FormatInt(value, 10) }

func AhdJSONFromReal(class *AhdClass, value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		AhdRaiseClass(class, "JSON Real value must be finite")
	}
	return ahdJSONFormatReal(value)
}

func AhdJSONFromString(value string) string { return ahdJSONEncodeString(value) }

// AhdJSONArray builds a compact Array from each element's own already
// canonical text.
func AhdJSONArray(texts []string) string {
	return "[" + strings.Join(texts, ",") + "]"
}

// AhdJSONObject builds a compact Object from ordered, already-canonical
// entries. A Pair<String, JSONValue> cannot itself carry a duplicate key, so
// no duplicate check is needed here.
func AhdJSONObject(class *AhdClass, entries []AhdJSONEntry) string {
	var builder strings.Builder
	builder.WriteByte('{')
	for index, entry := range entries {
		if index > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(ahdJSONEncodeString(entry.Key))
		builder.WriteByte(':')
		builder.WriteString(entry.Text)
	}
	builder.WriteByte('}')
	return builder.String()
}

// AhdJSONStringify renders a JSONValue's own canonical text back as compact
// text (a no-op) or reparses it to render with two-space pretty indentation.
func AhdJSONStringify(text string, pretty bool) string {
	if !pretty {
		return text
	}
	node := ahdJSONParseDocument(AhdClassJSONError, text)
	return ahdJSONStringifyNode(node, true, 0)
}

// AhdJSONWrite stringifies and writes the result to path, staging the
// content in the destination directory and renaming into place so a failed
// write never disturbs an existing destination.
func AhdJSONWrite(class *AhdClass, text, path string, pretty bool) {
	content := AhdJSONStringify(text, pretty)
	if err := ahdJSONPublish([]byte(content), path); err != nil {
		AhdRaiseClass(class, "could not write the JSON file: "+err.Error())
	}
}

func ahdJSONPublish(data []byte, output string) error {
	absolute, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	directory := filepath.Dir(absolute)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".ahdcode-json-output-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	_, writeError := temporary.Write(data)
	syncError := temporary.Sync()
	closeError := temporary.Close()
	for _, candidate := range []error{writeError, syncError, closeError} {
		if candidate != nil {
			return candidate
		}
	}
	return os.Rename(temporaryPath, absolute)
}

// ---------------------------------------------------------------------------
// JSONValue accessors
// ---------------------------------------------------------------------------

func ahdJSONDecode(class *AhdClass, text string) ahdJSONNode {
	return ahdJSONParseDocument(class, text)
}

func ahdJSONWrongKind(class *AhdClass, operation, kind string) {
	AhdRaiseClass(class, operation+" cannot be called on a "+kind+" JSONValue")
}

func AhdJSONKind(class *AhdClass, text string) string {
	return ahdJSONDecode(class, text).kind
}

func AhdJSONIsNull(class *AhdClass, text string) bool {
	return ahdJSONDecode(class, text).kind == "Null"
}

func AhdJSONBool(class *AhdClass, text string) bool {
	node := ahdJSONDecode(class, text)
	if node.kind != "Bool" {
		ahdJSONWrongKind(class, "bool()", node.kind)
	}
	return node.flag
}

func AhdJSONInt(class *AhdClass, text string) int64 {
	node := ahdJSONDecode(class, text)
	if node.kind != "Int" {
		ahdJSONWrongKind(class, "int()", node.kind)
	}
	return node.number
}

// AhdJSONReal accepts both Int and Real JSONValues, widening an Int the same
// way AhdCode's Int -> Real assignment already does.
func AhdJSONReal(class *AhdClass, text string) float64 {
	node := ahdJSONDecode(class, text)
	switch node.kind {
	case "Real":
		return node.real
	case "Int":
		return float64(node.number)
	default:
		ahdJSONWrongKind(class, "real()", node.kind)
		return 0
	}
}

func AhdJSONString(class *AhdClass, text string) string {
	node := ahdJSONDecode(class, text)
	if node.kind != "String" {
		ahdJSONWrongKind(class, "string()", node.kind)
	}
	return node.text
}

func AhdJSONArrayElements(class *AhdClass, text string) []string {
	node := ahdJSONDecode(class, text)
	if node.kind != "Array" {
		ahdJSONWrongKind(class, "array()", node.kind)
	}
	result := make([]string, len(node.items))
	for index, item := range node.items {
		result[index] = ahdJSONCanonicalText(item)
	}
	return result
}

func AhdJSONObjectKeys(class *AhdClass, text string) []string {
	node := ahdJSONDecode(class, text)
	if node.kind != "Object" {
		ahdJSONWrongKind(class, "object()", node.kind)
	}
	keys := make([]string, len(node.keys))
	copy(keys, node.keys)
	return keys
}

func AhdJSONObjectValueTexts(class *AhdClass, text string) []string {
	node := ahdJSONDecode(class, text)
	if node.kind != "Object" {
		ahdJSONWrongKind(class, "object()", node.kind)
	}
	values := make([]string, len(node.keys))
	for index, key := range node.keys {
		values[index] = ahdJSONCanonicalText(node.values[key])
	}
	return values
}

// AhdJSONGet returns the canonical text of key's value, or nil if the
// receiver has no such key. The receiver must itself be an Object.
func AhdJSONGet(class *AhdClass, text, key string) *string {
	node := ahdJSONDecode(class, text)
	if node.kind != "Object" {
		ahdJSONWrongKind(class, "get()", node.kind)
	}
	value, found := node.values[key]
	if !found {
		return nil
	}
	result := ahdJSONCanonicalText(value)
	return &result
}

// AhdJSONAt returns the canonical text of the element at index, following
// List index rules (a negative index counts back from the end). The
// receiver must itself be an Array, and index must be in range.
func AhdJSONAt(class *AhdClass, text string, index int64) string {
	node := ahdJSONDecode(class, text)
	if node.kind != "Array" {
		ahdJSONWrongKind(class, "at()", node.kind)
	}
	length := int64(len(node.items))
	resolved := index
	if resolved < 0 {
		resolved += length
	}
	if resolved < 0 || resolved >= length {
		AhdRaiseClass(class, "JSONValue array index is out of range")
	}
	return ahdJSONCanonicalText(node.items[resolved])
}

// --- XML standard module ---
//
// An XMLNode/XMLDocument is represented at rest as one hidden field holding
// a private, opaque JSON encoding of an ahdXMLData tree (never published,
// never valid XML on its own) - the same "hidden String field, reparsed by
// helpers" pattern Word's Document uses for its block list and JSON's
// JSONValue uses for its own canonical text. Unlike JSONValue, XML's own
// encoding is a full nested struct rather than a flat canonical string,
// since composing an Element from already-built child XMLNodes needs to
// decode each child once and embed it directly - Go's encoding/json already
// walks nested struct slices recursively, so no extra composition helper is
// needed the way JSON's array()/object() needed one.
//
// Parsing uses encoding/xml.Decoder (the same token-walking style Word uses
// for DOCX), not a hand-written grammar: Go's decoder already resolves
// element namespaces to full URIs, already treats CDATA as ordinary
// CharData, already ignores comments/processing instructions as separate
// token kinds, and - critically for security - never expands a DTD's
// external subset or a custom general entity by default (an unknown entity
// is a parse error, not something to substitute), so it is not vulnerable
// to XXE/billion-laughs without any extra code here.

const (
	ahdXMLMaxInputBytes = 8 * 1024 * 1024
	ahdXMLMaxDepth      = 256
)

// ahdXMLData is the parsed/interchange form of one XMLNode, and is also
// exactly what an XMLNode's/XMLDocument's hidden field encodes.
type ahdXMLData struct {
	Kind      string       `json:"kind"`
	Name      string       `json:"name,omitempty"`
	Namespace string       `json:"namespace,omitempty"`
	Text      string       `json:"text,omitempty"`
	AttrKeys  []string     `json:"attrKeys,omitempty"`
	AttrVals  []string     `json:"attrVals,omitempty"`
	Children  []ahdXMLData `json:"children,omitempty"`
}

func ahdXMLEncode(node ahdXMLData) string {
	encoded, _ := json.Marshal(node)
	return string(encoded)
}

func ahdXMLDecode(class *AhdClass, data string) ahdXMLData {
	var node ahdXMLData
	if err := json.Unmarshal([]byte(data), &node); err != nil {
		AhdRaiseClass(class, "XML node storage is corrupted")
	}
	return node
}

func ahdXMLWrongKind(class *AhdClass, operation, kind string) {
	AhdRaiseClass(class, operation+" cannot be called on a "+kind+" XMLNode")
}

// ---------------------------------------------------------------------------
// Construction
// ---------------------------------------------------------------------------

func AhdXMLText(value string) string {
	return ahdXMLEncode(ahdXMLData{Kind: "Text", Text: value})
}

// AhdXMLElement builds an Element from already-encoded child node data,
// decoding each child once to embed it directly in the new node's tree.
func AhdXMLElement(class *AhdClass, name string, attrKeys, attrVals []string, childrenData []string) string {
	children := make([]ahdXMLData, len(childrenData))
	for index, data := range childrenData {
		children[index] = ahdXMLDecode(class, data)
	}
	return ahdXMLEncode(ahdXMLData{Kind: "Element", Name: name, AttrKeys: attrKeys, AttrVals: attrVals, Children: children})
}

// AhdXMLDocument validates that root is an Element and, since an
// XMLDocument's hidden field is exactly its root Element's own encoding,
// simply re-validates and returns it.
func AhdXMLDocument(class *AhdClass, rootData string) string {
	root := ahdXMLDecode(class, rootData)
	if root.Kind != "Element" {
		AhdRaiseClass(class, "an XMLDocument root must be an Element, not Text")
	}
	return rootData
}

// ---------------------------------------------------------------------------
// Parsing
// ---------------------------------------------------------------------------

func ahdXMLParseDocument(class *AhdClass, source string) ahdXMLData {
	if len(source) > ahdXMLMaxInputBytes {
		AhdRaiseClass(class, "XML input is larger than the supported limit")
	}
	if !utf8.ValidString(source) {
		AhdRaiseClass(class, "XML input is not valid UTF-8")
	}
	decoder := xml.NewDecoder(strings.NewReader(source))
	var root *ahdXMLData
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			AhdRaiseClass(class, "XML input does not parse: "+err.Error())
		}
		switch element := token.(type) {
		case xml.StartElement:
			if root != nil {
				AhdRaiseClass(class, "XML input has more than one root element")
			}
			node := ahdXMLParseElement(class, decoder, element, 1)
			root = &node
		case xml.CharData:
			if root == nil && strings.TrimSpace(string(element)) != "" {
				AhdRaiseClass(class, "XML input has content before its root element")
			}
		}
	}
	if root == nil {
		AhdRaiseClass(class, "XML input has no root element")
	}
	return *root
}

func ahdXMLParseElement(class *AhdClass, decoder *xml.Decoder, start xml.StartElement, depth int) ahdXMLData {
	if depth > ahdXMLMaxDepth {
		AhdRaiseClass(class, "XML input exceeds the maximum supported nesting depth")
	}
	node := ahdXMLData{Kind: "Element", Name: start.Name.Local, Namespace: start.Name.Space}
	seen := make(map[string]bool, len(start.Attr))
	for _, attr := range start.Attr {
		if attr.Name.Space == "xmlns" || attr.Name.Local == "xmlns" {
			continue
		}
		key := attr.Name.Local
		if seen[key] {
			AhdRaiseClass(class, "XML element has a duplicate attribute")
		}
		seen[key] = true
		node.AttrKeys = append(node.AttrKeys, key)
		node.AttrVals = append(node.AttrVals, attr.Value)
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				AhdRaiseClass(class, "XML input ends before an element is closed")
			}
			AhdRaiseClass(class, "XML input does not parse: "+err.Error())
		}
		switch element := token.(type) {
		case xml.StartElement:
			node.Children = append(node.Children, ahdXMLParseElement(class, decoder, element, depth+1))
		case xml.EndElement:
			return node
		case xml.CharData:
			if len(element) > 0 {
				node.Children = append(node.Children, ahdXMLData{Kind: "Text", Text: string(element)})
			}
		}
	}
}

func AhdXMLParse(class *AhdClass, source string) string {
	return ahdXMLEncode(ahdXMLParseDocument(class, source))
}

func AhdXMLRead(class *AhdClass, path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		AhdRaiseClass(class, "could not read the XML file: "+err.Error())
	}
	return ahdXMLEncode(ahdXMLParseDocument(class, string(content)))
}

// ---------------------------------------------------------------------------
// Serialization
// ---------------------------------------------------------------------------

func ahdXMLEscapeText(value string) string {
	var builder strings.Builder
	for _, character := range value {
		switch character {
		case '&':
			builder.WriteString("&amp;")
		case '<':
			builder.WriteString("&lt;")
		case '>':
			builder.WriteString("&gt;")
		default:
			builder.WriteRune(character)
		}
	}
	return builder.String()
}

func ahdXMLEscapeAttr(value string) string {
	var builder strings.Builder
	for _, character := range value {
		switch character {
		case '&':
			builder.WriteString("&amp;")
		case '<':
			builder.WriteString("&lt;")
		case '"':
			builder.WriteString("&quot;")
		case '\n':
			builder.WriteString("&#10;")
		case '\r':
			builder.WriteString("&#13;")
		case '\t':
			builder.WriteString("&#9;")
		default:
			builder.WriteRune(character)
		}
	}
	return builder.String()
}

// ahdXMLStringifyNode renders one node. Pretty indentation is only inserted
// between a run of purely-Element children: inserting whitespace next to a
// Text child would add content that was not in the original tree, so mixed
// or text-only content is always rendered inline, in both modes. Compact
// output therefore always round-trips exactly; pretty output is a human
// readability convenience that may not for content mixing text and
// elements, the same well-known trade-off every XML pretty-printer makes.
func ahdXMLStringifyNode(node ahdXMLData, pretty bool, depth int) string {
	if node.Kind == "Text" {
		return ahdXMLEscapeText(node.Text)
	}
	var builder strings.Builder
	builder.WriteByte('<')
	builder.WriteString(node.Name)
	for index, key := range node.AttrKeys {
		builder.WriteByte(' ')
		builder.WriteString(key)
		builder.WriteString(`="`)
		builder.WriteString(ahdXMLEscapeAttr(node.AttrVals[index]))
		builder.WriteByte('"')
	}
	if len(node.Children) == 0 {
		builder.WriteString("/>")
		return builder.String()
	}
	builder.WriteByte('>')
	hasText := false
	for _, child := range node.Children {
		if child.Kind == "Text" {
			hasText = true
			break
		}
	}
	indent := func(level int) string {
		if !pretty || hasText {
			return ""
		}
		return "\n" + strings.Repeat("  ", level)
	}
	for _, child := range node.Children {
		builder.WriteString(indent(depth + 1))
		builder.WriteString(ahdXMLStringifyNode(child, pretty, depth+1))
	}
	builder.WriteString(indent(depth))
	builder.WriteString("</")
	builder.WriteString(node.Name)
	builder.WriteByte('>')
	return builder.String()
}

func AhdXMLStringify(class *AhdClass, documentData string, pretty bool) string {
	return ahdXMLStringifyNode(ahdXMLDecode(class, documentData), pretty, 0)
}

func ahdXMLPublish(data []byte, output string) error {
	absolute, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	directory := filepath.Dir(absolute)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".ahdcode-xml-output-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	_, writeError := temporary.Write(data)
	syncError := temporary.Sync()
	closeError := temporary.Close()
	for _, candidate := range []error{writeError, syncError, closeError} {
		if candidate != nil {
			return candidate
		}
	}
	return os.Rename(temporaryPath, absolute)
}

func AhdXMLWrite(class *AhdClass, documentData, path string, pretty bool) {
	content := AhdXMLStringify(class, documentData, pretty)
	if err := ahdXMLPublish([]byte(content), path); err != nil {
		AhdRaiseClass(class, "could not write the XML file: "+err.Error())
	}
}

// ---------------------------------------------------------------------------
// XMLNode/XMLDocument accessors
// ---------------------------------------------------------------------------

func AhdXMLKind(class *AhdClass, data string) string {
	return ahdXMLDecode(class, data).Kind
}

func AhdXMLName(class *AhdClass, data string) string {
	node := ahdXMLDecode(class, data)
	if node.Kind != "Element" {
		ahdXMLWrongKind(class, "name()", node.Kind)
	}
	return node.Name
}

func AhdXMLNamespace(class *AhdClass, data string) string {
	node := ahdXMLDecode(class, data)
	if node.Kind != "Element" {
		ahdXMLWrongKind(class, "namespace()", node.Kind)
	}
	return node.Namespace
}

// AhdXMLNodeText is the XMLNode.text() accessor (distinct from the
// AhdXMLText constructor): a Text node's own content, or an Element's
// direct Text children concatenated in document order.
func AhdXMLNodeText(class *AhdClass, data string) string {
	node := ahdXMLDecode(class, data)
	if node.Kind == "Text" {
		return node.Text
	}
	var builder strings.Builder
	for _, child := range node.Children {
		if child.Kind == "Text" {
			builder.WriteString(child.Text)
		}
	}
	return builder.String()
}

func AhdXMLAttribute(class *AhdClass, data, key string) *string {
	node := ahdXMLDecode(class, data)
	if node.Kind != "Element" {
		ahdXMLWrongKind(class, "attribute()", node.Kind)
	}
	for index, candidate := range node.AttrKeys {
		if candidate == key {
			value := node.AttrVals[index]
			return &value
		}
	}
	return nil
}

func AhdXMLAttributeKeys(class *AhdClass, data string) []string {
	node := ahdXMLDecode(class, data)
	if node.Kind != "Element" {
		ahdXMLWrongKind(class, "attributes()", node.Kind)
	}
	keys := make([]string, len(node.AttrKeys))
	copy(keys, node.AttrKeys)
	return keys
}

func AhdXMLAttributeValues(class *AhdClass, data string) []string {
	node := ahdXMLDecode(class, data)
	if node.Kind != "Element" {
		ahdXMLWrongKind(class, "attributes()", node.Kind)
	}
	values := make([]string, len(node.AttrVals))
	copy(values, node.AttrVals)
	return values
}

func AhdXMLChildrenData(class *AhdClass, data string) []string {
	node := ahdXMLDecode(class, data)
	if node.Kind != "Element" {
		ahdXMLWrongKind(class, "children()", node.Kind)
	}
	result := make([]string, len(node.Children))
	for index, child := range node.Children {
		result[index] = ahdXMLEncode(child)
	}
	return result
}

func AhdXMLElementsData(class *AhdClass, data string) []string {
	node := ahdXMLDecode(class, data)
	if node.Kind != "Element" {
		ahdXMLWrongKind(class, "elements()", node.Kind)
	}
	var result []string
	for _, child := range node.Children {
		if child.Kind == "Element" {
			result = append(result, ahdXMLEncode(child))
		}
	}
	return result
}

// --- Env standard module ---
//
// Env is deliberately small: it has no data-carrying Class (unlike Word/
// JSON/XML), just plain String/Bool/Nothing/Pair<String,String> functions
// over the process environment and a bounded .env file grammar. There is
// no shell interpolation, no command substitution, and no variable
// expansion - a .env value is read as literal text (with a small, explicit
// escape set inside double quotes), never evaluated.

var ahdEnvKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// AhdEnvEntry is one ordered .env file assignment.
type AhdEnvEntry struct {
	Key   string
	Value string
}

func ahdEnvValidateName(class *AhdClass, name string) {
	if name == "" {
		AhdRaiseClass(class, "environment variable name must not be empty")
	}
	if strings.IndexByte(name, 0) >= 0 {
		AhdRaiseClass(class, "environment variable name must not contain a NUL byte")
	}
	if strings.IndexByte(name, '=') >= 0 {
		AhdRaiseClass(class, "environment variable name must not contain '='")
	}
}

func AhdEnvGet(name string) *string {
	value, present := os.LookupEnv(name)
	if !present {
		return nil
	}
	return &value
}

func AhdEnvGetOr(name, fallback string) string {
	if value, present := os.LookupEnv(name); present {
		return value
	}
	return fallback
}

func AhdEnvHas(name string) bool {
	_, present := os.LookupEnv(name)
	return present
}

func AhdEnvSet(class *AhdClass, name, value string) {
	ahdEnvValidateName(class, name)
	if err := os.Setenv(name, value); err != nil {
		AhdRaiseClass(class, "could not set the environment variable")
	}
}

func AhdEnvUnset(class *AhdClass, name string) {
	ahdEnvValidateName(class, name)
	if err := os.Unsetenv(name); err != nil {
		AhdRaiseClass(class, "could not unset the environment variable")
	}
}

// ---------------------------------------------------------------------------
// .env parsing
// ---------------------------------------------------------------------------

func ahdEnvParseFile(class *AhdClass, content string) []AhdEnvEntry {
	var entries []AhdEnvEntry
	seen := make(map[string]bool)
	for index, rawLine := range strings.Split(content, "\n") {
		lineNumber := index + 1
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || trimmed[0] == '#' {
			continue
		}
		key, value := ahdEnvParseAssignment(class, trimmed, lineNumber)
		if seen[key] {
			AhdRaiseClass(class, fmt.Sprintf("line %d: duplicate key %q in .env file", lineNumber, key))
		}
		seen[key] = true
		entries = append(entries, AhdEnvEntry{Key: key, Value: value})
	}
	return entries
}

func ahdEnvParseAssignment(class *AhdClass, line string, lineNumber int) (string, string) {
	equals := strings.IndexByte(line, '=')
	if equals < 0 {
		AhdRaiseClass(class, fmt.Sprintf("line %d is not a valid KEY=value assignment", lineNumber))
	}
	key := line[:equals]
	if !ahdEnvKeyPattern.MatchString(key) {
		AhdRaiseClass(class, fmt.Sprintf("line %d has an invalid key %q", lineNumber, key))
	}
	return key, ahdEnvParseValue(class, line[equals+1:], lineNumber)
}

func ahdEnvParseValue(class *AhdClass, rest string, lineNumber int) string {
	if len(rest) == 0 {
		return ""
	}
	switch rest[0] {
	case '"':
		return ahdEnvParseDoubleQuoted(class, rest, lineNumber)
	case '\'':
		return ahdEnvParseSingleQuoted(class, rest, lineNumber)
	default:
		return strings.TrimSpace(rest)
	}
}

func ahdEnvParseDoubleQuoted(class *AhdClass, rest string, lineNumber int) string {
	var builder strings.Builder
	index := 1
	for index < len(rest) {
		character := rest[index]
		switch character {
		case '"':
			if strings.TrimSpace(rest[index+1:]) != "" {
				AhdRaiseClass(class, fmt.Sprintf("line %d has content after a closing quote", lineNumber))
			}
			return builder.String()
		case '\\':
			index++
			if index >= len(rest) {
				AhdRaiseClass(class, fmt.Sprintf("line %d has an incomplete escape sequence", lineNumber))
			}
			switch rest[index] {
			case '\\':
				builder.WriteByte('\\')
			case '"':
				builder.WriteByte('"')
			case 'n':
				builder.WriteByte('\n')
			case 'r':
				builder.WriteByte('\r')
			case 't':
				builder.WriteByte('\t')
			default:
				AhdRaiseClass(class, fmt.Sprintf("line %d has an invalid escape sequence", lineNumber))
			}
			index++
			continue
		default:
			builder.WriteByte(character)
		}
		index++
	}
	AhdRaiseClass(class, fmt.Sprintf("line %d has an unterminated double-quoted value", lineNumber))
	return ""
}

func ahdEnvParseSingleQuoted(class *AhdClass, rest string, lineNumber int) string {
	closing := strings.IndexByte(rest[1:], '\'')
	if closing < 0 {
		AhdRaiseClass(class, fmt.Sprintf("line %d has an unterminated single-quoted value", lineNumber))
	}
	value := rest[1 : 1+closing]
	if strings.TrimSpace(rest[1+closing+1:]) != "" {
		AhdRaiseClass(class, fmt.Sprintf("line %d has content after a closing quote", lineNumber))
	}
	return value
}

func ahdEnvReadFile(class *AhdClass, path string) []AhdEnvEntry {
	content, err := os.ReadFile(path)
	if err != nil {
		AhdRaiseClass(class, "could not read the .env file: "+err.Error())
	}
	return ahdEnvParseFile(class, string(content))
}

// AhdEnvReadEntries parses path without touching the process environment.
func AhdEnvReadEntries(class *AhdClass, path string) []AhdEnvEntry {
	return ahdEnvReadFile(class, path)
}

// AhdEnvLoad parses the entire file first (so a later malformed line can
// never leave the process environment half-applied), then applies every
// entry: with override=false an already-present variable (checked with
// LookupEnv, so an explicitly empty existing value still counts as
// present) is left untouched; with override=true the .env value always
// wins.
func AhdEnvLoad(class *AhdClass, path string, override bool) {
	entries := ahdEnvReadFile(class, path)
	for _, entry := range entries {
		if !override {
			if _, present := os.LookupEnv(entry.Key); present {
				continue
			}
		}
		if err := os.Setenv(entry.Key, entry.Value); err != nil {
			AhdRaiseClass(class, "could not set the environment variable")
		}
	}
}

// ---------------------------------------------------------------------------
// Lists and KeyValue: pure structural collection transformations
//
// Every helper below reads its source List/Pair and builds a new structural
// collection; none mutates a source, so a Constant collection may be passed
// safely. The transformation is shallow: element and value references are
// carried over unchanged, never deep-copied.
// ---------------------------------------------------------------------------

// AhdListsChunk splits a List into consecutive Lists of at most size elements.
// The final chunk is short rather than padded, and every returned inner List
// is a new List over the same element references.
func AhdListsChunk[T any](class *AhdClass, values *AhdList[T], size int64) *AhdList[*AhdList[T]] {
	values.require()
	if size <= 0 {
		AhdRaiseClass(class, "chunk requires a size greater than zero; received "+strconv.FormatInt(size, 10))
	}
	result := AhdNewList[*AhdList[T]]()
	length := int64(len(values.items))
	for start := int64(0); start < length; start += size {
		end := start + size
		if end > length || end < start {
			end = length
		}
		part := make([]T, end-start)
		copy(part, values.items[start:end])
		result.items = append(result.items, &AhdList[T]{items: part})
	}
	return result
}

// AhdListsFlatten concatenates a List of Lists, exactly one level deep.
func AhdListsFlatten[T any](rows *AhdList[*AhdList[T]]) *AhdList[T] {
	rows.require()
	result := AhdNewList[T]()
	for _, row := range rows.items {
		row.require()
		result.items = append(result.items, row.items...)
	}
	return result
}

// AhdListsTranspose exchanges rows and columns. Ragged input is a ListsError:
// padding or truncating it would silently invent or destroy data.
func AhdListsTranspose[T any](class *AhdClass, rows *AhdList[*AhdList[T]]) *AhdList[*AhdList[T]] {
	rows.require()
	result := AhdNewList[*AhdList[T]]()
	if len(rows.items) == 0 {
		return result
	}
	rows.items[0].require()
	width := len(rows.items[0].items)
	for index, row := range rows.items {
		row.require()
		if len(row.items) != width {
			AhdRaiseClass(class, "transpose requires rectangular rows: row "+strconv.Itoa(index)+
				" has "+strconv.Itoa(len(row.items))+" element(s); expected "+strconv.Itoa(width))
		}
	}
	for column := 0; column < width; column++ {
		items := make([]T, len(rows.items))
		for index, row := range rows.items {
			items[index] = row.items[column]
		}
		result.items = append(result.items, &AhdList[T]{items: items})
	}
	return result
}

// AhdListsUnique keeps the first occurrence of each distinct element, using
// the same == equality the language defines for that element type.
func AhdListsUnique[T any](values *AhdList[T], equal func(T, T) bool) *AhdList[T] {
	values.require()
	result := AhdNewList[T]()
	for _, item := range values.items {
		seen := false
		for _, kept := range result.items {
			if equal(kept, item) {
				seen = true
				break
			}
		}
		if !seen {
			result.items = append(result.items, item)
		}
	}
	return result
}

// AhdListsValueCounts counts equal elements, keyed in first-occurrence order.
func AhdListsValueCounts[K comparable](values *AhdList[K]) *AhdPair[K, int64] {
	values.require()
	result := AhdNewPair[K, int64]()
	for _, item := range values.items {
		result.values[item]++
		if result.values[item] == 1 {
			result.keys = append(result.keys, item)
		}
	}
	return result
}

// AhdListsGroupBy partitions a snapshot of the elements into new Lists keyed by
// a key Function, in first-key-occurrence order. Elements inside each group
// keep their source order.
func AhdListsGroupBy[T any, K comparable](values *AhdList[T], key func(T) K) *AhdPair[K, *AhdList[T]] {
	items := values.Snapshot()
	result := AhdNewPair[K, *AhdList[T]]()
	for _, item := range items {
		selected := key(item)
		group, exists := result.values[selected]
		if !exists {
			group = AhdNewList[T]()
			result.keys = append(result.keys, selected)
			result.values[selected] = group
		}
		group.items = append(group.items, item)
	}
	return result
}
