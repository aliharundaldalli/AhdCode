package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const numericModulePrefix = "builtin:Numeric::"
const (
	numericVectorClass = ir.ClassID("builtin:Numeric::class::Vector")
	numericMatrixClass = ir.ClassID("builtin:Numeric::class::Matrix")
)

var numericVectorField = ir.FieldID(string(numericVectorClass) + "::field::values")
var numericMatrixField = ir.FieldID(string(numericMatrixClass) + "::field::rows")

const numericErrorRuntime = "AhdClassNumericError"

func (generator *generator) numericCall(value *ir.CallExpr) string {
	name := strings.TrimPrefix(string(value.Callable), numericModulePrefix)
	meta := value.ExprMeta()
	arg := func(i int) ir.Expr {
		if i < len(value.Arguments) {
			return value.Arguments[i].Value
		}
		return nil
	}
	integer := func(i int) string { return generator.value(arg(i), ir.Type{Kind: ir.IntType}, false) }
	realValue := func(i int) string {
		e := arg(i)
		if e.ExprMeta().Type.Kind == ir.IntType {
			return "float64(" + generator.value(e, ir.Type{Kind: ir.IntType}, false) + ")"
		}
		return generator.value(e, ir.Type{Kind: ir.RealType}, false)
	}
	switch name {
	case "vector":
		return generator.numericVectorFrom("AhdNumericVector("+numericErrorRuntime+", "+generator.numericRealList(arg(0))+")", meta)
	case "matrix":
		return generator.numericMatrixFrom("AhdNumericMatrix("+numericErrorRuntime+", "+generator.numericRealGrid(arg(0))+")", meta)
	case "zeros", "ones":
		if len(value.Arguments) == 1 {
			return generator.numericVectorFrom("AhdNumeric"+title(name)+"Vector("+numericErrorRuntime+", "+integer(0)+")", meta)
		}
		return generator.numericMatrixFrom("AhdNumeric"+title(name)+"Matrix("+numericErrorRuntime+", "+integer(0)+", "+integer(1)+")", meta)
	case "identity":
		return generator.numericMatrixFrom("AhdNumericIdentity("+numericErrorRuntime+", "+integer(0)+")", meta)
	case "linspace":
		return generator.numericVectorFrom("AhdNumericLinspace("+numericErrorRuntime+", "+realValue(0)+", "+realValue(1)+", "+integer(2)+")", meta)
	}
	return generator.unsupported("Numeric function "+name, meta.Span)
}

func title(s string) string { return strings.ToUpper(s[:1]) + s[1:] }
func (generator *generator) numericRealList(e ir.Expr) string {
	if e.ExprMeta().Type.Element != nil && e.ExprMeta().Type.Element.Kind == ir.IntType {
		return "AhdNumericWidenList(" + generator.expr(e) + ")"
	}
	return generator.expr(e)
}
func (generator *generator) numericRealGrid(e ir.Expr) string {
	if e.ExprMeta().Type.Element != nil && e.ExprMeta().Type.Element.Kind == ir.ListType && e.ExprMeta().Type.Element.Element != nil && e.ExprMeta().Type.Element.Element.Kind == ir.IntType {
		return "AhdNumericWidenGrid(" + generator.expr(e) + ")"
	}
	return generator.expr(e)
}

func (generator *generator) numericVectorOf(e ir.Expr) string {
	return "AhdVector{Values: " + generator.expr(e) + "." + generator.fieldName(numericVectorField) + "_get()}"
}
func (generator *generator) numericMatrixOf(e ir.Expr) string {
	return "AhdMatrix{Rows: " + generator.expr(e) + "." + generator.fieldName(numericMatrixField) + "_get()}"
}
func (generator *generator) numericVectorFrom(code string, meta ir.ExprBase) string {
	h, ok := generator.numericHelper(numericVectorClass, "nv_")
	if !ok {
		return generator.unsupported("Vector without Class declaration", meta.Span)
	}
	return h + "(" + code + ")"
}
func (generator *generator) numericMatrixFrom(code string, meta ir.ExprBase) string {
	h, ok := generator.numericHelper(numericMatrixClass, "nm_")
	if !ok {
		return generator.unsupported("Matrix without Class declaration", meta.Span)
	}
	return h + "(" + code + ")"
}
func (generator *generator) numericHelper(class ir.ClassID, prefix string) (string, bool) {
	if generator.layouts[class] == nil {
		return "", false
	}
	if n, ok := generator.timeHelpers[class]; ok {
		return n, true
	}
	n := mangleNamed(prefix, generator.classDisplayName(class), string(class))
	generator.timeHelpers[class] = n
	return n, true
}

func (generator *generator) emitNumericHelpers(w *emitter) {
	for _, item := range []struct {
		class          ir.ClassID
		runtime, field string
	}{{numericVectorClass, "AhdVector", "Values"}, {numericMatrixClass, "AhdMatrix", "Rows"}} {
		n, ok := generator.timeHelpers[item.class]
		if !ok {
			continue
		}
		layout := generator.layouts[item.class]
		if layout == nil {
			continue
		}
		ctor := generator.functions[layout.class.Constructor]
		if ctor == nil {
			continue
		}
		w.open("func " + n + "(value " + item.runtime + ") " + generator.interfaceName(item.class) + " {")
		w.line("return " + generator.callableName(ctor) + "(value." + item.field + ")")
		w.close("}")
		w.blank()
	}
}

func (generator *generator) numericOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	arg := func(i int) ir.Expr {
		if i < len(value.Arguments) {
			return value.Arguments[i].Value
		}
		return nil
	}
	if strings.HasPrefix(name, "Vector.") {
		op := strings.TrimPrefix(name, "Vector.")
		recv := generator.numericVectorOf(value.Callee)
		switch op {
		case "length":
			return "AhdNumericVectorLength(" + recv + ")"
		case "values":
			return "AhdNumericVectorValues(" + recv + ")"
		case "add", "subtract":
			return generator.numericVectorFrom("AhdNumericVector"+title(op)+"("+numericErrorRuntime+", "+recv+", "+generator.numericVectorOf(arg(0))+")", meta)
		case "scale":
			return generator.numericVectorFrom("AhdNumericVectorScale("+recv+", "+generator.value(arg(0), ir.Type{Kind: ir.RealType}, false)+")", meta)
		case "dot":
			return "AhdNumericVectorDot(" + numericErrorRuntime + ", " + recv + ", " + generator.numericVectorOf(arg(0)) + ")"
		case "abs", "sqrt", "exp", "log":
			return generator.numericVectorFrom("AhdNumericVectorElementwise("+numericErrorRuntime+", "+recv+", \""+op+"\")", meta)
		case "sum", "min", "max":
			return "AhdNumericVectorReduction(" + numericErrorRuntime + ", " + recv + ", \"" + op + "\")"
		}
	}
	op := strings.TrimPrefix(name, "Matrix.")
	recv := generator.numericMatrixOf(value.Callee)
	switch op {
	case "determinant", "inverse", "solve", "rank", "lu", "qr", "cholesky", "svd", "eigenvalues":
		generator.usesNumeric = true
	}
	switch op {
	case "rowCount":
		return "AhdNumericMatrixRowCount(" + recv + ")"
	case "columnCount":
		return "AhdNumericMatrixColumnCount(" + recv + ")"
	case "rows":
		return "AhdNumericMatrixRows(" + recv + ")"
	case "transpose":
		return generator.numericMatrixFrom("AhdNumericMatrixTranspose("+recv+")", meta)
	case "add", "subtract":
		return generator.numericMatrixFrom("AhdNumericMatrix"+title(op)+"("+numericErrorRuntime+", "+recv+", "+generator.numericMatrixOf(arg(0))+")", meta)
	case "scale":
		return generator.numericMatrixFrom("AhdNumericMatrixScale("+recv+", "+generator.value(arg(0), ir.Type{Kind: ir.RealType}, false)+")", meta)
	case "matmul":
		return generator.numericMatrixFrom("AhdNumericMatrixMatmul("+numericErrorRuntime+", "+recv+", "+generator.numericMatrixOf(arg(0))+")", meta)
	case "trace", "determinant":
		return "AhdNumericMatrix" + title(op) + "(" + numericErrorRuntime + ", " + recv + ")"
	case "inverse", "cholesky":
		return generator.numericMatrixFrom("AhdNumericMatrix"+title(op)+"("+numericErrorRuntime+", "+recv+")", meta)
	case "solve":
		return generator.numericVectorFrom("AhdNumericMatrixSolve("+numericErrorRuntime+", "+recv+", "+generator.numericVectorOf(arg(0))+")", meta)
	case "rank":
		return "AhdNumericMatrixRank(" + numericErrorRuntime + ", " + recv + ")"
	case "eigenvalues":
		return "AhdNumericMatrixEigenvalues(" + numericErrorRuntime + ", " + recv + ")"
	case "abs", "sqrt", "exp", "log":
		return generator.numericMatrixFrom("AhdNumericMatrixElementwise("+numericErrorRuntime+", "+recv+", \""+op+"\")", meta)
	case "sum", "min", "max":
		return "AhdNumericMatrixReduction(" + numericErrorRuntime + ", " + recv + ", \"" + op + "\")"
	case "lu", "qr", "svd":
		return generator.numericMatrixPair("AhdNumericMatrix"+strings.ToUpper(op)+"("+numericErrorRuntime+", "+recv+")", meta)
	}
	return generator.unsupported("Numeric operation "+name, meta.Span)
}

func (generator *generator) numericMatrixPair(code string, meta ir.ExprBase) string {
	h, ok := generator.numericHelper(numericMatrixClass, "nm_")
	if !ok {
		return generator.unsupported("Matrix decomposition", meta.Span)
	}
	matrixType := generator.interfaceName(numericMatrixClass)
	return "AhdNumericWrapMatrixPair(" + code + ", func(v AhdMatrix) " + matrixType + " { return " + h + "(v) })"
}
