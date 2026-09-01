package semantic

import (
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

// NullState is flow-sensitive and deliberately orthogonal to types.Type.
type NullState uint8

const (
	MaybeNull NullState = iota
	Null
	NonNull
)

func (state NullState) String() string {
	switch state {
	case Null:
		return "Null"
	case NonNull:
		return "NonNull"
	default:
		return "MaybeNull"
	}
}

type SymbolKind uint8

const (
	BindingSymbol SymbolKind = iota
	ParameterSymbol
	FunctionSymbol
	ClassSymbol
	MemberSymbol
	ForSymbol
	ExceptSymbol
	BuiltinSymbol
	NamespaceSymbol
)

// Callable preserves concrete signatures and null-state metadata separately;
// it can later be serialized as a cross-module contract.
type Callable struct {
	Signature     *types.Signature
	ParameterNull []NullState
	ReturnNull    NullState
	Declaration   *ast.FunctionDecl
	Lambda        *ast.LambdaExpr
	Structure     *ast.StructureDecl
	// Captures are the enclosing bindings a lambda explicitly listed, in
	// source order. They are not part of Signature: a capture is closure
	// storage the caller never supplies, so the callable's public arity and
	// type are exactly what was written between the parentheses.
	Captures []*Capture
}

// Capture is one resolved entry of a lambda's explicit capture list. Outer is
// the enclosing binding being read; Inner is the read-only binding the lambda
// body sees under the same name.
type Capture struct {
	Name  string
	Outer *Symbol
	Inner *Symbol
}

// OverloadSet is distinct from a single Function type. Candidates remain
// concrete callables and declaration order has no ranking meaning.
type OverloadSet struct {
	Name       string
	Candidates []*Callable
}

// CandidateDecision records an overload applicability result without exposing
// raw Go/internal type dumps.
type CandidateDecision struct {
	Signature  string
	Applicable bool
	Reason     string
	Widenings  int
	Defaults   int
}

type ResolutionTrace struct {
	Candidates []CandidateDecision
	Selected   string
}

// Symbol is a resolved semantic declaration. Alias points from an explicit
// Global declaration to its module-root binding.
type Symbol struct {
	Name         string
	Kind         SymbolKind
	Type         types.Type
	Span         source.Span
	Declaration  ast.Node
	Constant     bool
	Confidential bool
	ModuleRoot   bool
	Builtin      bool
	// SuperClassBinding marks the implicit SuperClass binding installed inside
	// a Class callable. It designates the parent implementation rather than an
	// ordinary Class reference value.
	SuperClassBinding bool
	InitialNull       NullState
	// DeclaredNullable is the symbol's static, unnarrowed nullability -- true
	// only when its declared type carried a trailing `?`. Unlike InitialNull
	// (a flow-sensitive starting point) and the per-statement flowState (which
	// narrows and widens as code runs), this never changes after declaration.
	// It answers "may null legally be written here", independent of whatever
	// the current flow analysis currently believes about the value.
	DeclaredNullable bool
	// Captured marks a binding a lambda reads through its explicit capture
	// list. The lambda sees the enclosing binding's value; it does not gain
	// ownership of the outer variable, so the name cannot be reassigned from
	// inside the lambda.
	Captured       bool
	Alias          *Symbol
	Callable       *Callable
	OverloadSet    *OverloadSet
	Namespace      *ModuleInterface
	Class          *types.ClassSymbol
	OwnerClass     *types.ClassSymbol
	OriginModuleID string
	Members        map[string]*Symbol
	Constructor    *Callable
	// ConstructorAttributes records, per constructor parameter, the instance
	// attribute that parameter initializes. A Local structure parameter has a
	// nil entry. Inherited entries come from the parent construction contract.
	ConstructorAttributes []*Symbol
	ConstValue            *constantValue
	// BuiltinLiteral is the canonical source-independent scalar representation
	// of a compiler-supplied Constant. It lets lowering materialize standard
	// module constants without inventing filesystem-backed storage.
	BuiltinLiteral string
	// ModuleOperation names the compiler-supplied, type-directed standard
	// module function this symbol denotes. Such a function has no single fixed
	// signature: its result type is computed from the statically known
	// argument types at each call site (see ModuleOperation). It is empty for
	// every ordinary symbol, including the fixed-signature standard module
	// functions built by standardFunction.
	ModuleOperation ModuleOperation
	inference       *functionInference
}

// ModuleInterface is an in-memory, compile-time-only public contract. Identity
// is canonical and never derived from pointer addresses.
type ModuleInterface struct {
	ModuleID    string
	Name        string
	Exports     map[string]*Symbol
	Symbols     map[string]*Symbol // includes Confidential entries for access diagnostics
	Classes     map[string]*Symbol // canonical identity -> Class metadata, including ancestry support
	ExportNames []string
}

// TypeOperation is one built-in operation on a language type. These are typed
// language operations selected by the statically known receiver type rather
// than dynamic member lookups, so a user Class may still declare its own
// methods with the same names.
type TypeOperation string

const (
	// ListAdd appends one element to the end of a List.
	ListAdd TypeOperation = "List.add"
	// ListEject removes the element at an index, which may be negative.
	ListEject TypeOperation = "List.eject"
	// PairEject removes one key and its value from a Pair.
	PairEject TypeOperation = "Pair.eject"
	// ListSort orders a List in place, naturally or by a key Function.
	ListSort TypeOperation = "List.sort"
	// ListReverse reverses a List in place.
	ListReverse TypeOperation = "List.reverse"
	// ListShuffle permutes a List in place using the shared Math RNG.
	ListShuffle TypeOperation = "List.shuffle"
	// ListCount counts equal elements without mutating the List.
	ListCount TypeOperation = "List.count"
	// ListIndex is the first index of an equal element.
	ListIndex TypeOperation = "List.index"
	// ListMap builds a new List from a Function applied to a snapshot.
	ListMap TypeOperation = "List.map"
	// ListFilter builds a new List of the elements a predicate keeps.
	ListFilter TypeOperation = "List.filter"

	// StringTrim removes leading and trailing Unicode whitespace.
	StringTrim TypeOperation = "String.trim"
	// StringLower is locale-independent Unicode lowercase.
	StringLower TypeOperation = "String.lower"
	// StringUpper is locale-independent Unicode uppercase.
	StringUpper TypeOperation = "String.upper"
	// StringCapitalize uppercases only the first character.
	StringCapitalize TypeOperation = "String.capitalize"
	// StringSplit divides a String on every occurrence of a separator.
	StringSplit TypeOperation = "String.split"
	// StringReplace rewrites every occurrence of a search text.
	StringReplace TypeOperation = "String.replace"
	// StringContains reports substring membership.
	StringContains TypeOperation = "String.contains"
	// StringStartsWith reports a prefix match.
	StringStartsWith TypeOperation = "String.startsWith"
	// StringEndsWith reports a suffix match.
	StringEndsWith TypeOperation = "String.endsWith"
	// StringCount counts non-overlapping occurrences of a search text.
	StringCount TypeOperation = "String.count"
	// StringIndex is the first character index of a search text.
	StringIndex TypeOperation = "String.index"

	ComplexReal      TypeOperation = "Complex.real"
	ComplexImag      TypeOperation = "Complex.imag"
	ComplexConjugate TypeOperation = "Complex.conjugate"
	ComplexMagnitude TypeOperation = "Complex.magnitude"
	ComplexPhase     TypeOperation = "Complex.phase"

	NumericVectorLength      TypeOperation = "Vector.length"
	NumericVectorValues      TypeOperation = "Vector.values"
	NumericVectorAdd         TypeOperation = "Vector.add"
	NumericVectorSubtract    TypeOperation = "Vector.subtract"
	NumericVectorScale       TypeOperation = "Vector.scale"
	NumericVectorDot         TypeOperation = "Vector.dot"
	NumericVectorAbs         TypeOperation = "Vector.abs"
	NumericVectorSqrt        TypeOperation = "Vector.sqrt"
	NumericVectorExp         TypeOperation = "Vector.exp"
	NumericVectorLog         TypeOperation = "Vector.log"
	NumericVectorSum         TypeOperation = "Vector.sum"
	NumericVectorMin         TypeOperation = "Vector.min"
	NumericVectorMax         TypeOperation = "Vector.max"
	NumericMatrixRowCount    TypeOperation = "Matrix.rowCount"
	NumericMatrixColumnCount TypeOperation = "Matrix.columnCount"
	NumericMatrixRows        TypeOperation = "Matrix.rows"
	NumericMatrixTranspose   TypeOperation = "Matrix.transpose"
	NumericMatrixAdd         TypeOperation = "Matrix.add"
	NumericMatrixSubtract    TypeOperation = "Matrix.subtract"
	NumericMatrixScale       TypeOperation = "Matrix.scale"
	NumericMatrixMatmul      TypeOperation = "Matrix.matmul"
	NumericMatrixDeterminant TypeOperation = "Matrix.determinant"
	NumericMatrixTrace       TypeOperation = "Matrix.trace"
	NumericMatrixInverse     TypeOperation = "Matrix.inverse"
	NumericMatrixSolve       TypeOperation = "Matrix.solve"
	NumericMatrixRank        TypeOperation = "Matrix.rank"
	NumericMatrixLU          TypeOperation = "Matrix.lu"
	NumericMatrixQR          TypeOperation = "Matrix.qr"
	NumericMatrixCholesky    TypeOperation = "Matrix.cholesky"
	NumericMatrixSVD         TypeOperation = "Matrix.svd"
	NumericMatrixEigenvalues TypeOperation = "Matrix.eigenvalues"
	NumericMatrixAbs         TypeOperation = "Matrix.abs"
	NumericMatrixSqrt        TypeOperation = "Matrix.sqrt"
	NumericMatrixExp         TypeOperation = "Matrix.exp"
	NumericMatrixLog         TypeOperation = "Matrix.log"
	NumericMatrixSum         TypeOperation = "Matrix.sum"
	NumericMatrixMin         TypeOperation = "Matrix.min"
	NumericMatrixMax         TypeOperation = "Matrix.max"

	WordDocumentHeading    TypeOperation = "Document.heading"
	WordDocumentParagraph  TypeOperation = "Document.paragraph"
	WordDocumentTable      TypeOperation = "Document.table"
	WordDocumentImage      TypeOperation = "Document.image"
	WordDocumentPageBreak  TypeOperation = "Document.pageBreak"
	WordDocumentSave       TypeOperation = "Document.save"
	WordDocumentText       TypeOperation = "Document.text"
	WordDocumentParagraphs TypeOperation = "Document.paragraphs"
	WordDocumentHeadings   TypeOperation = "Document.headings"
	WordDocumentTables     TypeOperation = "Document.tables"

	// DateTimeBefore, DateTimeAfter, and DateTimeSameMoment order two moments
	// without giving AhdCode operator overloading.
	DateTimeBefore     TypeOperation = "DateTime.before"
	DateTimeAfter      TypeOperation = "DateTime.after"
	DateTimeSameMoment TypeOperation = "DateTime.sameMoment"
	DateTimeTimestamp  TypeOperation = "DateTime.timestamp"
	DateTimeToUTC      TypeOperation = "DateTime.toUTC"
	DateTimeToLocal    TypeOperation = "DateTime.toLocal"
	DateTimeToOffset   TypeOperation = "DateTime.toOffset"
	// DateTimeToString is the stable, locale-independent moment text.
	DateTimeToString TypeOperation = "DateTime.toString"
	// CalendarIsLeapYear, CalendarDaysInMonth, and CalendarWeekday are the
	// Calendar members. They are reached through the Calendar Class reference,
	// so the language gains no static-method concept.
	CalendarIsLeapYear  TypeOperation = "Calendar.isLeapYear"
	CalendarDaysInMonth TypeOperation = "Calendar.daysInMonth"
	CalendarWeekday     TypeOperation = "Calendar.weekday"

	// RegexMatches reports whether a pattern is found anywhere in a String.
	RegexMatches TypeOperation = "Regex.matches"
	// RegexFind is the first match, or null if the pattern is not found.
	RegexFind TypeOperation = "Regex.find"
	// RegexFindAll is every non-overlapping match, in order.
	RegexFindAll TypeOperation = "Regex.findAll"
	// RegexGroups is the first match's full match followed by its capture
	// groups, or null if the pattern is not found.
	RegexGroups TypeOperation = "Regex.groups"
	// RegexReplace rewrites every match with a replacement, which may
	// reference capture groups as $1, $2, and so on.
	RegexReplace TypeOperation = "Regex.replace"
	// RegexSplit divides a String on every match of the pattern.
	RegexSplit TypeOperation = "Regex.split"

	// The Data standard module's Table members. Every Table operation is pure:
	// it reads a snapshot and, where it produces a Table, returns a new one.
	// DataRowCount is the number of rows.
	DataRowCount TypeOperation = "Table.rowCount"
	// DataColumnCount is the number of columns.
	DataColumnCount TypeOperation = "Table.columnCount"
	// DataColumns is a new List snapshot of the column names, in order.
	DataColumns TypeOperation = "Table.columns"
	// DataRows is a new List of new Pair row snapshots.
	DataRows TypeOperation = "Table.rows"
	// DataRow is a new Pair snapshot of one row, by List index rules.
	DataRow TypeOperation = "Table.row"
	// DataColumn is a new List of one column's cells, in row order.
	DataColumn TypeOperation = "Table.column"
	// DataHead keeps the first rows; DataTail keeps the last rows.
	DataHead TypeOperation = "Table.head"
	DataTail TypeOperation = "Table.tail"
	// DataSelect keeps the requested columns in the requested order.
	DataSelect TypeOperation = "Table.select"
	// DataDrop removes the requested columns, keeping the original order.
	DataDrop TypeOperation = "Table.drop"
	// DataRename renames one column in place, preserving its position.
	DataRename TypeOperation = "Table.rename"
	// DataReverse reverses row order.
	DataReverse TypeOperation = "Table.reverse"
	// DataFilter keeps the rows a (Pair<String, String>) -> Bool predicate
	// accepts, in source order.
	DataFilter TypeOperation = "Table.filter"
	// DataSort orders rows by a column name or by an Int/Real/String key
	// Function, stably and ascending.
	DataSort TypeOperation = "Table.sort"
	// DataTransform rewrites one column through a (String) -> String Function.
	DataTransform TypeOperation = "Table.transform"
	// DataDerive appends a column built by a
	// (Pair<String, String>) -> String Function.
	DataDerive TypeOperation = "Table.derive"
	// DataUnique lists one column's distinct cells in first-occurrence order.
	DataUnique TypeOperation = "Table.unique"
	// DataValueCounts counts one column's cells in first-occurrence order.
	DataValueCounts TypeOperation = "Table.valueCounts"
	// DataGroupBy partitions rows into Tables keyed by one column's cells.
	DataGroupBy TypeOperation = "Table.groupBy"
	// DataPivotCount is a strict count cross-tabulation of two columns.
	DataPivotCount TypeOperation = "Table.pivotCount"
	// DataToCSV and DataWriteCSV serialize through the CSV module's writer.
	DataToCSV    TypeOperation = "Table.toCSV"
	DataWriteCSV TypeOperation = "Table.writeCSV"

	// The Plot standard module's Chart and Figure members. Every Chart
	// operation is pure: it returns a new Chart and leaves the receiver
	// untouched, matching the Table convention.
	PlotChartTitle  TypeOperation = "Chart.title"
	PlotChartXLabel TypeOperation = "Chart.xLabel"
	PlotChartYLabel TypeOperation = "Chart.yLabel"
	PlotChartLegend TypeOperation = "Chart.legend"
	PlotChartSize   TypeOperation = "Chart.size"
	// PlotChartLine and PlotChartScatter append one more series to a Chart
	// that is either empty (Plot.new()) or already a line/scatter composite.
	PlotChartLine    TypeOperation = "Chart.line"
	PlotChartScatter TypeOperation = "Chart.scatter"
	PlotChartSave    TypeOperation = "Chart.save"
	PlotChartShow    TypeOperation = "Chart.show"
	PlotFigureSave   TypeOperation = "Figure.save"
	PlotFigureShow   TypeOperation = "Figure.show"

	// The JSON standard module's JSONValue members. JSONValue is a closed,
	// immutable recursive value; every accessor below raises JSONError if the
	// receiver's kind does not match, except JSONValueGet, whose absence of a
	// key is a genuine, statically expected possibility rather than an error.
	JSONValueKind   TypeOperation = "JSONValue.kind"
	JSONValueIsNull TypeOperation = "JSONValue.isNull"
	JSONValueBool   TypeOperation = "JSONValue.bool"
	JSONValueInt    TypeOperation = "JSONValue.int"
	JSONValueReal   TypeOperation = "JSONValue.real"
	JSONValueString TypeOperation = "JSONValue.string"
	JSONValueArray  TypeOperation = "JSONValue.array"
	JSONValueObject TypeOperation = "JSONValue.object"
	// JSONValueGet looks up one Object key, or null if the key is absent.
	JSONValueGet TypeOperation = "JSONValue.get"
	// JSONValueAt indexes one Array element by List index rules.
	JSONValueAt TypeOperation = "JSONValue.at"

	// The XML standard module's XMLNode members. XMLNode is a closed,
	// immutable Element/Text value.
	XMLNodeKind TypeOperation = "XMLNode.kind"
	// XMLNodeName is the Element's local name; Text raises XMLError.
	XMLNodeName TypeOperation = "XMLNode.name"
	// XMLNodeNamespace is the Element's namespace URI, or "" when unqualified;
	// Text raises XMLError.
	XMLNodeNamespace TypeOperation = "XMLNode.namespace"
	// XMLNodeText is a Text node's own content, or an Element's direct Text
	// children concatenated in document order.
	XMLNodeText TypeOperation = "XMLNode.text"
	// XMLNodeAttribute looks up one Element attribute, or null if absent;
	// Text raises XMLError.
	XMLNodeAttribute TypeOperation = "XMLNode.attribute"
	// XMLNodeAttributes is every Element attribute in document order; Text
	// raises XMLError.
	XMLNodeAttributes TypeOperation = "XMLNode.attributes"
	// XMLNodeChildren is every Element/Text child in document order; Text
	// raises XMLError.
	XMLNodeChildren TypeOperation = "XMLNode.children"
	// XMLNodeElements is only the child Element nodes, in document order;
	// Text raises XMLError.
	XMLNodeElements TypeOperation = "XMLNode.elements"
	// XMLDocumentRoot is the XMLDocument's single root Element. The task
	// specification's XMLDocument surface (parse/read/stringify/write/
	// document) omits any way back from an XMLDocument to its XMLNode tree;
	// root() is the smallest addition that closes that gap.
	XMLDocumentRoot TypeOperation = "XMLDocument.root"
)

// ModuleOperation is one compiler-supplied standard module function whose
// result type is computed from the statically known argument types at each
// call site, rather than declared once as a fixed types.Signature.
//
// It is the module-function counterpart of TypeOperation: a TypeOperation is
// selected by a receiver's static type after a dot, while a ModuleOperation is
// selected by the resolved module symbol and then specialized against its
// arguments. Both stay entirely inside the compiler: every call site still
// resolves to one exact concrete type, so no Any, dynamic, or erased Object
// representation exists anywhere in the public semantic types.
type ModuleOperation string

const (
	// The Lists standard module. Every operation is a pure structural
	// transformation of a List: the source List is never modified, and each
	// returned List is a new structural collection over the same element
	// references.
	//
	// ListsChunk splits a List into consecutive Lists of at most one size.
	ListsChunk ModuleOperation = "Lists.chunk"
	// ListsFlatten concatenates a List of Lists, exactly one level deep.
	ListsFlatten ModuleOperation = "Lists.flatten"
	// ListsTranspose exchanges the rows and columns of a rectangular List of
	// Lists. Ragged input is a ListsError, never a silent repair.
	ListsTranspose ModuleOperation = "Lists.transpose"
	// ListsUnique keeps the first occurrence of each distinct element, using
	// ordinary AhdCode == equality.
	ListsUnique ModuleOperation = "Lists.unique"
	// ListsValueCounts counts equal elements into a Pair keyed by the element,
	// in first-occurrence order.
	ListsValueCounts ModuleOperation = "Lists.valueCounts"
	// ListsGroupBy partitions elements into Lists keyed by a key Function, in
	// first-key-occurrence order.
	ListsGroupBy ModuleOperation = "Lists.groupBy"
)

// listOperationMutates reports whether one List operation rewrites its
// receiver, so the Constant rules apply to it.
func listOperationMutates(operation TypeOperation) bool {
	switch operation {
	case ListAdd, ListEject, ListSort, ListReverse, ListShuffle:
		return true
	default:
		return false
	}
}

// Environment supplies already-resolved dependency interfaces to one semantic
// analysis run. Filesystem/module graph work stays outside this package.
type Environment struct {
	ModuleID      string
	ModuleName    string
	Imports       map[string]*ModuleInterface
	FailedImports map[string]bool
}

// Result is a side-table semantic model; Analyze never mutates the AST.
type Result struct {
	Diagnostics            []diagnostics.Diagnostic
	Symbols                []*Symbol
	ResolvedSymbols        map[ast.Node]*Symbol
	ExpressionTypes        map[ast.Expr]types.Type
	NullStates             map[ast.Expr]NullState
	SelectedCallables      map[*ast.CallExpr]*Callable
	SelectedFunctionValues map[ast.Expr]*Callable
	// SelectedAssignmentCallables records the resolved Class Protocol Method
	// for one compound assignment (+=, -=, and so on) whose target is a Class
	// instance, keyed by the AssignmentStmt itself since it is a statement
	// rather than an expression.
	SelectedAssignmentCallables map[*ast.AssignmentStmt]*Callable
	OverloadResolutions         map[*ast.CallExpr]ResolutionTrace
	// SuperCalls marks member expressions written as SuperClass.member, which
	// bind the current instance but call the parent implementation directly.
	SuperCalls map[ast.Expr]bool
	// TypeOperations records the built-in String, List, and Pair operations,
	// so lowering never has to rediscover them from member names.
	TypeOperations map[*ast.CallExpr]TypeOperation
	// ModuleOperations records the type-directed standard module calls, so a
	// consumer can see which operation a call selected without re-deriving it
	// from the resolved symbol. Lowering reads the specialized signature from
	// SelectedCallables instead, and identifies the operation by the stable
	// builtin CallableID.
	ModuleOperations map[*ast.CallExpr]ModuleOperation
	// LambdaExpressions preserves source order for deterministic lowering into
	// the existing Function IR/runtime representation.
	LambdaExpressions []*ast.LambdaExpr
}

func (result Result) HasErrors() bool {
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Severity == diagnostics.SeverityError {
			return true
		}
	}
	return false
}
