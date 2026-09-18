package semantic

import (
	"ahdcode/internal/types"
)

// The v1.7 members of DateTime, Duration, Vector, Matrix, and Table publish
// real parameter names, the way Canvas and Turtle members do: a call binds its
// arguments like a module function call (all positional or all named), the
// editor completes the member, hover renders its signature, and signature help
// shows the selected overload. They reuse the Graphics member machinery rather
// than a second one.
const (
	DateTimeToISO    TypeOperation = "DateTime.toISO"
	DateTimeAdd      TypeOperation = "DateTime.add"
	DateTimeSubtract TypeOperation = "DateTime.subtract"

	DurationAdd      TypeOperation = "Duration.add"
	DurationSubtract TypeOperation = "Duration.subtract"
	DurationNegate   TypeOperation = "Duration.negate"
	DurationAbs      TypeOperation = "Duration.abs"

	NumericVectorAt    TypeOperation = "Vector.at"
	NumericVectorNorm  TypeOperation = "Vector.norm"
	NumericVectorOuter TypeOperation = "Vector.outer"
	NumericVectorCross TypeOperation = "Vector.cross"

	NumericMatrixAt       TypeOperation = "Matrix.at"
	NumericMatrixRow      TypeOperation = "Matrix.row"
	NumericMatrixColumn   TypeOperation = "Matrix.column"
	NumericMatrixDiagonal TypeOperation = "Matrix.diagonal"
	NumericMatrixNorm     TypeOperation = "Matrix.norm"
	NumericMatrixHadamard TypeOperation = "Matrix.hadamard"
	NumericMatrixMatvec   TypeOperation = "Matrix.matvec"

	DataConcat    TypeOperation = "Table.concat"
	DataInnerJoin TypeOperation = "Table.innerJoin"
)

// The member names each Class gained in v1.7, in documentation order.
var (
	TimeDateTimeCompletionOperations  = []string{"toISO", "add", "subtract"}
	TimeDurationOperations            = []string{"add", "subtract", "negate", "abs"}
	NumericVectorCompletionOperations = []string{"at", "norm", "outer", "cross"}
	NumericMatrixCompletionOperations = []string{"at", "row", "column", "diagonal", "norm", "hadamard", "matvec"}
	DataTableCompletionOperations     = []string{"concat", "innerJoin"}
)

// completionMember builds one published member with a single signature.
func completionMember(module, name string, result types.Type, parameters ...types.Parameter) *Symbol {
	return completionOverloads(module, name, &types.Signature{Parameters: parameters, Return: result})
}

// completionOverloads builds one published member with one or more
// signatures; several signatures form an ordinary OverloadSet.
func completionOverloads(module, name string, signatures ...*types.Signature) *Symbol {
	symbol := &Symbol{
		Name: name, Kind: FunctionSymbol, Type: types.Function{},
		Builtin: true, InitialNull: NonNull, OriginModuleID: module,
	}
	for _, signature := range signatures {
		callable := &Callable{Signature: signature, ParameterNull: nonNullParameters(len(signature.Parameters)), ReturnNull: NonNull}
		if symbol.Callable == nil {
			symbol.Callable = callable
		}
		if len(signatures) > 1 {
			if symbol.OverloadSet == nil {
				symbol.OverloadSet = &OverloadSet{Name: name}
			}
			symbol.OverloadSet.Candidates = append(symbol.OverloadSet.Candidates, callable)
		}
	}
	if len(signatures) == 1 {
		symbol.Type = types.Function{Signature: signatures[0]}
	}
	return symbol
}

var completionMembers = func() map[TypeOperation]*Symbol {
	dateTime := types.Class{Symbol: timeDateTimeClass}
	duration := types.Class{Symbol: timeDurationClass}
	vector, matrix := numericVectorType(), numericMatrixType()
	table := dataTableType()
	p := func(name string, typ types.Type) types.Parameter { return types.Parameter{Name: name, Type: typ} }
	return map[TypeOperation]*Symbol{
		DateTimeToISO:    completionMember(timeModuleID, "toISO", types.String),
		DateTimeAdd:      completionMember(timeModuleID, "add", dateTime, p("duration", duration)),
		DateTimeSubtract: completionMember(timeModuleID, "subtract", dateTime, p("duration", duration)),

		DurationAdd:      completionMember(timeModuleID, "add", duration, p("other", duration)),
		DurationSubtract: completionMember(timeModuleID, "subtract", duration, p("other", duration)),
		DurationNegate:   completionMember(timeModuleID, "negate", duration),
		DurationAbs:      completionMember(timeModuleID, "abs", duration),

		NumericVectorAt:    completionMember(numericModuleID, "at", types.Real, p("index", types.Int)),
		NumericVectorNorm:  completionMember(numericModuleID, "norm", types.Real),
		NumericVectorOuter: completionMember(numericModuleID, "outer", matrix, p("other", vector)),
		NumericVectorCross: completionMember(numericModuleID, "cross", vector, p("other", vector)),

		NumericMatrixAt:       completionMember(numericModuleID, "at", types.Real, p("row", types.Int), p("column", types.Int)),
		NumericMatrixRow:      completionMember(numericModuleID, "row", vector, p("index", types.Int)),
		NumericMatrixColumn:   completionMember(numericModuleID, "column", vector, p("index", types.Int)),
		NumericMatrixDiagonal: completionMember(numericModuleID, "diagonal", vector),
		NumericMatrixNorm:     completionMember(numericModuleID, "norm", types.Real),
		NumericMatrixHadamard: completionMember(numericModuleID, "hadamard", matrix, p("other", matrix)),
		NumericMatrixMatvec:   completionMember(numericModuleID, "matvec", vector, p("vector", vector)),

		DataConcat: completionMember(dataModuleID, "concat", table, p("other", table)),
		DataInnerJoin: completionOverloads(dataModuleID, "innerJoin",
			&types.Signature{Parameters: []types.Parameter{p("other", table), p("key", types.String)}, Return: table},
			&types.Signature{Parameters: []types.Parameter{p("other", table), p("leftKey", types.String), p("rightKey", types.String)}, Return: table}),
	}
}()

// publishedMember returns the Symbol of a built-in member that publishes
// parameter names, or nil.
func publishedMember(operation TypeOperation) *Symbol {
	if symbol, ok := graphicsMembers[operation]; ok {
		return symbol
	}
	return completionMembers[operation]
}

// publishedMemberNames lists the published members of one compiler-supplied
// Class identity, for editor completion.
func publishedMemberNames(identity *types.ClassSymbol) []string {
	switch {
	case identity.ModuleID == graphicsModuleID && identity.Name == "Canvas":
		return GraphicsCanvasOperations
	case identity.ModuleID == graphicsModuleID && identity.Name == "Turtle":
		return GraphicsTurtleOperations
	case identity.ModuleID == timeModuleID && identity.Name == "DateTime":
		return TimeDateTimeCompletionOperations
	case identity.ModuleID == timeModuleID && identity.Name == "Duration":
		return TimeDurationOperations
	case identity.ModuleID == numericModuleID && identity.Name == "Vector":
		return NumericVectorCompletionOperations
	case identity.ModuleID == numericModuleID && identity.Name == "Matrix":
		return NumericMatrixCompletionOperations
	case identity.ModuleID == dataModuleID && identity.Name == "Table":
		return DataTableCompletionOperations
	}
	return nil
}
