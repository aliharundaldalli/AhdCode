package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const numericModuleID = "builtin:Numeric"

var (
	numericErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error", Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	numericErrorClass  = &types.ClassSymbol{ModuleID: numericModuleID, Name: "NumericError", Parent: numericErrorParent}
	numericVectorClass = &types.ClassSymbol{ModuleID: numericModuleID, Name: "Vector"}
	numericMatrixClass = &types.ClassSymbol{ModuleID: numericModuleID, Name: "Matrix"}
)

func NumericErrorIdentity() *types.ClassSymbol  { return numericErrorClass }
func NumericVectorIdentity() *types.ClassSymbol { return numericVectorClass }
func NumericMatrixIdentity() *types.ClassSymbol { return numericMatrixClass }
func numericVectorType() types.Type             { return types.Class{Symbol: numericVectorClass} }
func numericMatrixType() types.Type             { return types.Class{Symbol: numericMatrixClass} }

var NumericVectorOperations = []string{"length", "values", "add", "subtract", "scale", "dot", "abs", "sqrt", "exp", "log", "sum", "min", "max"}
var NumericMatrixOperations = []string{"rowCount", "columnCount", "rows", "transpose", "add", "subtract", "scale", "matmul", "determinant", "trace", "inverse", "solve", "rank", "lu", "qr", "cholesky", "svd", "eigenvalues", "abs", "sqrt", "exp", "log", "sum", "min", "max"}

func numericModuleInterface() *ModuleInterface {
	module := standardInterface(numericModuleID, "Numeric")
	for _, item := range []struct {
		name        string
		identity    *types.ClassSymbol
		constructor *Callable
	}{
		{"NumericError", numericErrorClass, builtinErrorConstructor()}, {"Vector", numericVectorClass, nil}, {"Matrix", numericMatrixClass, nil},
	} {
		symbol := &Symbol{Name: item.name, Kind: ClassSymbol, Class: item.identity, Type: types.Class{Symbol: item.identity, Reference: true}, ModuleRoot: true, Builtin: true, InitialNull: NonNull, OriginModuleID: numericModuleID, Members: map[string]*Symbol{}, Constructor: item.constructor}
		module.Classes[numericModuleID+"\x00"+item.name] = symbol
		addStandardExport(module, symbol)
	}
	vector, matrix := numericVectorType(), numericMatrixType()
	listInt, listReal := types.List{Element: types.Int}, types.List{Element: types.Real}
	gridInt, gridReal := types.List{Element: listInt}, types.List{Element: listReal}
	addStandardExport(module, numericFunction("vector", signature(vector, param("values", listInt)), signature(vector, param("values", listReal))))
	addStandardExport(module, numericFunction("matrix", signature(matrix, param("rows", gridInt)), signature(matrix, param("rows", gridReal))))
	addStandardExport(module, numericFunction("zeros", signature(vector, param("size", types.Int)), signature(matrix, param("rows", types.Int), param("columns", types.Int))))
	addStandardExport(module, numericFunction("ones", signature(vector, param("size", types.Int)), signature(matrix, param("rows", types.Int), param("columns", types.Int))))
	addStandardExport(module, numericFunction("identity", signature(matrix, param("size", types.Int))))
	var line []*types.Signature
	for _, start := range []types.Type{types.Int, types.Real} {
		for _, stop := range []types.Type{types.Int, types.Real} {
			line = append(line, signature(vector, param("start", start), param("stop", stop), param("count", types.Int)))
		}
	}
	addStandardExport(module, numericFunction("linspace", line...))
	sort.Strings(module.ExportNames)
	return module
}

func param(name string, value types.Type) types.Parameter {
	return types.Parameter{Name: name, Type: value}
}
func signature(result types.Type, parameters ...types.Parameter) *types.Signature {
	return &types.Signature{Parameters: parameters, Return: result}
}
func numericFunction(name string, signatures ...*types.Signature) *Symbol {
	s := &Symbol{Name: name, Kind: FunctionSymbol, Type: types.Function{}, ModuleRoot: true, Builtin: true, InitialNull: NonNull, OriginModuleID: numericModuleID}
	for _, sig := range signatures {
		c := &Callable{Signature: sig, ParameterNull: nonNullParameters(len(sig.Parameters)), ReturnNull: NonNull}
		if s.Callable == nil {
			s.Callable = c
		}
		if len(signatures) > 1 {
			if s.OverloadSet == nil {
				s.OverloadSet = &OverloadSet{Name: name}
			}
			s.OverloadSet.Candidates = append(s.OverloadSet.Candidates, c)
		}
	}
	if len(signatures) == 1 {
		s.Type = types.Function{Signature: signatures[0]}
	}
	return s
}

var numericOperationNames = map[string]map[string]TypeOperation{
	"Vector": {"length": NumericVectorLength, "values": NumericVectorValues, "add": NumericVectorAdd, "subtract": NumericVectorSubtract, "scale": NumericVectorScale, "dot": NumericVectorDot, "abs": NumericVectorAbs, "sqrt": NumericVectorSqrt, "exp": NumericVectorExp, "log": NumericVectorLog, "sum": NumericVectorSum, "min": NumericVectorMin, "max": NumericVectorMax},
	"Matrix": {"rowCount": NumericMatrixRowCount, "columnCount": NumericMatrixColumnCount, "rows": NumericMatrixRows, "transpose": NumericMatrixTranspose, "add": NumericMatrixAdd, "subtract": NumericMatrixSubtract, "scale": NumericMatrixScale, "matmul": NumericMatrixMatmul, "determinant": NumericMatrixDeterminant, "trace": NumericMatrixTrace, "inverse": NumericMatrixInverse, "solve": NumericMatrixSolve, "rank": NumericMatrixRank, "lu": NumericMatrixLU, "qr": NumericMatrixQR, "cholesky": NumericMatrixCholesky, "svd": NumericMatrixSVD, "eigenvalues": NumericMatrixEigenvalues, "abs": NumericMatrixAbs, "sqrt": NumericMatrixSqrt, "exp": NumericMatrixExp, "log": NumericMatrixLog, "sum": NumericMatrixSum, "min": NumericMatrixMin, "max": NumericMatrixMax},
}

func numericOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	c, ok := receiver.(types.Class)
	if !ok || c.Reference || c.Symbol == nil || c.Symbol.ModuleID != numericModuleID {
		return "", false
	}
	op, ok := numericOperationNames[c.Symbol.Name][name]
	return op, ok
}

type numericShape struct {
	parameters []types.Type
	result     types.Type
	hint       string
}

func numericShapes() map[TypeOperation]numericShape {
	v, m := numericVectorType(), numericMatrixType()
	realList := types.List{Element: types.Real}
	grid := types.List{Element: realList}
	matrixPair := types.Pair{Key: types.String, Value: m}
	return map[TypeOperation]numericShape{
		NumericVectorLength: {nil, types.Int, "call length with no argument"}, NumericVectorValues: {nil, realList, "call values with no argument"}, NumericVectorAdd: {[]types.Type{v}, v, "pass one Vector"}, NumericVectorSubtract: {[]types.Type{v}, v, "pass one Vector"}, NumericVectorScale: {[]types.Type{types.Real}, v, "pass one Real factor"}, NumericVectorDot: {[]types.Type{v}, types.Real, "pass one Vector"},
		NumericVectorAbs: {nil, v, "call with no argument"}, NumericVectorSqrt: {nil, v, "call with no argument"}, NumericVectorExp: {nil, v, "call with no argument"}, NumericVectorLog: {nil, v, "call with no argument"}, NumericVectorSum: {nil, types.Real, "call with no argument"}, NumericVectorMin: {nil, types.Real, "call with no argument"}, NumericVectorMax: {nil, types.Real, "call with no argument"},
		NumericMatrixRowCount: {nil, types.Int, "call with no argument"}, NumericMatrixColumnCount: {nil, types.Int, "call with no argument"}, NumericMatrixRows: {nil, grid, "call with no argument"}, NumericMatrixTranspose: {nil, m, "call with no argument"}, NumericMatrixAdd: {[]types.Type{m}, m, "pass one Matrix"}, NumericMatrixSubtract: {[]types.Type{m}, m, "pass one Matrix"}, NumericMatrixScale: {[]types.Type{types.Real}, m, "pass one Real factor"}, NumericMatrixMatmul: {[]types.Type{m}, m, "pass one Matrix"},
		NumericMatrixDeterminant: {nil, types.Real, "call with no argument"}, NumericMatrixTrace: {nil, types.Real, "call with no argument"}, NumericMatrixInverse: {nil, m, "call with no argument"}, NumericMatrixSolve: {[]types.Type{v}, v, "pass one Vector"}, NumericMatrixRank: {nil, types.Int, "call with no argument"}, NumericMatrixLU: {nil, matrixPair, "call with no argument"}, NumericMatrixQR: {nil, matrixPair, "call with no argument"}, NumericMatrixCholesky: {nil, m, "call with no argument"}, NumericMatrixSVD: {nil, matrixPair, "call with no argument"}, NumericMatrixEigenvalues: {nil, types.List{Element: types.Complex}, "call with no argument"},
		NumericMatrixAbs: {nil, m, "call with no argument"}, NumericMatrixSqrt: {nil, m, "call with no argument"}, NumericMatrixExp: {nil, m, "call with no argument"}, NumericMatrixLog: {nil, m, "call with no argument"}, NumericMatrixSum: {nil, types.Real, "call with no argument"}, NumericMatrixMin: {nil, types.Real, "call with no argument"}, NumericMatrixMax: {nil, types.Real, "call with no argument"},
	}
}

func (a *analyzer) analyzeNumericOperation(call *ast.CallExpr, operation TypeOperation, shape numericShape, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: shape.result, nullState: NonNull}
	if len(call.Arguments) != len(shape.parameters) {
		a.error(codeCallArguments, fmt.Sprintf("%s expects %d argument(s); received %d", operation, len(shape.parameters), len(call.Arguments)), call.Span(), shape.hint)
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return result
	}
	for i, expected := range shape.parameters {
		info := a.analyzeExpressionExpected(call.Arguments[i].Value, current, flow, expected)
		if info.invalid() {
			continue
		}
		if info.nullState != NonNull {
			a.nullableError(string(operation), call.Arguments[i].Value, info.nullState)
		} else if !types.Assignable(expected, info.typeValue) {
			a.typeMismatch(call.Arguments[i].Span(), expected, info.typeValue, string(operation)+" argument")
		}
	}
	return result
}

func numericConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != numericModuleID {
		return "", false
	}
	if identity.Name == "Vector" {
		return "create a Vector with Numeric.vector, zeros, ones, or linspace", true
	}
	if identity.Name == "Matrix" {
		return "create a Matrix with Numeric.matrix, zeros, ones, or identity", true
	}
	return "", false
}
