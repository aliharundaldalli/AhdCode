package evaluator

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"ahdcode/internal/ir"
)

type argumentValue struct {
	value       any
	usesDefault bool
}

func (session *Session) evalArguments(arguments []ir.Argument, current *frame) []argumentValue {
	result := make([]argumentValue, len(arguments))
	for index, argument := range arguments {
		result[index].usesDefault = argument.UsesDefault
		if argument.Value != nil {
			result[index].value = session.eval(argument.Value, current)
		}
	}
	return result
}

func (session *Session) evalCall(call *ir.CallExpr, current *frame) any {
	if string(call.Callable) == "builtin:core::write" && len(call.Arguments) == 1 && call.Arguments[0].Value != nil {
		session.writeText(session.textOf(call.Arguments[0].Value, current) + "\n")
		return Nothing
	}
	arguments := session.evalArguments(call.Arguments, current)
	identity := string(call.Callable)
	if strings.HasPrefix(identity, "builtin:") {
		var receiver any
		if call.Callee != nil {
			// Type operations carry the raw receiver. Direct module functions do
			// not need a callee value at runtime.
			if _, member := call.Callee.(*ir.MemberExpr); !member || strings.HasPrefix(identity, "builtin:core::") {
				receiver = session.eval(call.Callee, current)
			}
		}
		if identity == "builtin:core::sum" && len(arguments) == 1 {
			if list, ok := arguments[0].value.(*List); ok && list != nil && len(list.Items) == 0 {
				if call.ExprMeta().Type.Kind == ir.RealType {
					return float64(0)
				}
				return int64(0)
			}
		}
		return session.builtin(call.Callable, receiver, arguments)
	}
	if call.Callee != nil {
		callee := session.eval(call.Callee, current)
		if function, ok := callee.(*FunctionValue); ok {
			return session.invoke(function, arguments)
		}
	}
	return session.invoke(&FunctionValue{Callable: call.Callable}, arguments)
}

func (session *Session) invoke(value *FunctionValue, arguments []argumentValue) any {
	if value == nil {
		session.raise("NullError", "Function value is null")
	}
	identity := value.Callable
	if value.Receiver != nil && !value.Direct {
		if table := session.dispatch[value.Receiver.Class]; table != nil && table[identity] != "" {
			identity = table[identity]
		}
	}
	if strings.HasPrefix(string(identity), "builtin:") {
		return session.builtin(identity, value.Receiver, arguments)
	}
	function := session.functions[identity]
	if function == nil {
		session.raise("Error", "missing callable "+string(identity))
	}
	if len(value.Captured) != 0 {
		// A capturing lambda's captures are the callable's leading parameters,
		// so they are bound ahead of the declared arguments. This mirrors the
		// native backend, where the closure passes them in the same positions.
		bound := make([]argumentValue, 0, len(value.Captured)+len(arguments))
		for _, captured := range value.Captured {
			bound = append(bound, argumentValue{value: captured})
		}
		arguments = append(bound, arguments...)
	}
	return session.invokeFunction(function, value.Receiver, arguments)
}

func (session *Session) invokeFunction(function *ir.Function, receiver *Instance, arguments []argumentValue) any {
	current := newFrame(nil)
	if function.Kind == ir.ConstructorFunction {
		current.constructing = receiver
	}
	if function.Receiver != "" {
		current.locals[function.Receiver] = &Cell{Value: receiver}
	}
	for index, parameter := range function.Parameters {
		var item argumentValue
		if index < len(arguments) {
			item = arguments[index]
		} else {
			item.usesDefault = true
		}
		value := item.value
		if item.usesDefault {
			if parameter.Default == nil {
				session.raise("Error", "missing argument for "+parameter.Name)
			}
			value = session.eval(parameter.Default, current)
		}
		current.locals[parameter.ID] = &Cell{Value: value}
	}
	_, _, outcome := session.execBlock(function.Body, current)
	if outcome.kind == flowReturn {
		return outcome.value
	}
	return Nothing
}

func (session *Session) construct(classID ir.ClassID, constructorID ir.CallableID, arguments []argumentValue) *Instance {
	instance := &Instance{Class: classID, Fields: make(map[ir.FieldID]any)}
	constructor := session.functions[constructorID]
	if constructor == nil {
		session.raise("Error", "missing constructor for "+className(classID))
	}
	if constructor.ParentConstructor != "" {
		parent := session.functions[constructor.ParentConstructor]
		mapped := make([]argumentValue, len(constructor.ParentArguments))
		for index, parameterIndex := range constructor.ParentArguments {
			if parameterIndex >= 0 && parameterIndex < len(arguments) {
				mapped[index] = arguments[parameterIndex]
			}
		}
		session.invokeFunction(parent, instance, mapped)
	}
	session.invokeFunction(constructor, instance, arguments)
	return instance
}

func values(arguments []argumentValue) []any {
	result := make([]any, len(arguments))
	for index := range arguments {
		result[index] = arguments[index].value
	}
	return result
}

func (session *Session) builtin(identity ir.CallableID, receiver any, arguments []argumentValue) any {
	name := string(identity)
	switch {
	case strings.HasPrefix(name, "builtin:core::"):
		return session.core(strings.TrimPrefix(name, "builtin:core::"), receiver, values(arguments))
	case strings.HasPrefix(name, "builtin:Math::"):
		return session.math(strings.TrimPrefix(name, "builtin:Math::"), values(arguments))
	case strings.HasPrefix(name, "builtin:CSV::"):
		return session.csvBuiltin(strings.TrimPrefix(name, "builtin:CSV::"), values(arguments))
	case strings.HasPrefix(name, "builtin:Time::"):
		return session.timeBuiltin(strings.TrimPrefix(name, "builtin:Time::"), values(arguments))
	case strings.HasPrefix(name, "builtin:Latex::"):
		return session.latexBuiltin(strings.TrimPrefix(name, "builtin:Latex::"), values(arguments))
	case strings.HasPrefix(name, "builtin:File::"):
		return session.fileBuiltin(strings.TrimPrefix(name, "builtin:File::"), values(arguments))
	case strings.HasPrefix(name, "builtin:Path::"):
		return session.pathBuiltin(strings.TrimPrefix(name, "builtin:Path::"), values(arguments))
	case strings.HasPrefix(name, "builtin:Regex::"):
		return session.regexBuiltin(strings.TrimPrefix(name, "builtin:Regex::"), values(arguments))
	case strings.HasPrefix(name, "builtin:Statistics::"):
		return session.statisticsBuiltin(strings.TrimPrefix(name, "builtin:Statistics::"), values(arguments))
	case strings.HasPrefix(name, "builtin:Data::"):
		return session.dataBuiltin(strings.TrimPrefix(name, "builtin:Data::"), values(arguments))
	case strings.HasPrefix(name, "builtin:Plot::"):
		return session.plotBuiltin(strings.TrimPrefix(name, "builtin:Plot::"), values(arguments))
	case strings.HasPrefix(name, "builtin:Numeric::"):
		return session.numericBuiltin(strings.TrimPrefix(name, "builtin:Numeric::"), values(arguments))
	case strings.HasPrefix(name, "builtin:Word::"):
		return session.wordBuiltin(strings.TrimPrefix(name, "builtin:Word::"), values(arguments))
	case strings.HasPrefix(name, "builtin:Excel::"):
		return session.excelBuiltin(strings.TrimPrefix(name, "builtin:Excel::"), values(arguments))
	case strings.HasPrefix(name, "builtin:PDF::"):
		return session.pdfBuiltin(strings.TrimPrefix(name, "builtin:PDF::"), values(arguments))
	case strings.HasPrefix(name, "builtin:Archive::"):
		return session.archiveBuiltin(strings.TrimPrefix(name, "builtin:Archive::"), values(arguments))
	case strings.HasPrefix(name, "builtin:JSON::"):
		return session.jsonBuiltin(strings.TrimPrefix(name, "builtin:JSON::"), values(arguments))
	case strings.HasPrefix(name, "builtin:XML::"):
		return session.xmlBuiltin(strings.TrimPrefix(name, "builtin:XML::"), values(arguments))
	case strings.HasPrefix(name, "builtin:Env::"):
		return session.envBuiltin(strings.TrimPrefix(name, "builtin:Env::"), values(arguments))
	case strings.HasPrefix(name, "builtin:Lists::"):
		return session.listsBuiltin(strings.TrimPrefix(name, "builtin:Lists::"), values(arguments))
	case strings.HasPrefix(name, "builtin:KeyValue::"):
		return session.keyValueBuiltin(strings.TrimPrefix(name, "builtin:KeyValue::"), values(arguments))
	case strings.HasPrefix(name, "builtin:SQLite::"):
		return session.sqliteBuiltin(strings.TrimPrefix(name, "builtin:SQLite::"), values(arguments))
	case strings.HasPrefix(name, "builtin:HTTP::"):
		return session.httpBuiltin(strings.TrimPrefix(name, "builtin:HTTP::"), values(arguments))
	case strings.HasPrefix(name, "builtin:HTML::"):
		return session.htmlBuiltin(strings.TrimPrefix(name, "builtin:HTML::"), values(arguments))
	}
	session.raise("Error", "unsupported builtin "+name)
	return nil
}

func (session *Session) core(name string, receiver any, arguments []any) any {
	if strings.HasPrefix(name, "Vector.") || strings.HasPrefix(name, "Matrix.") {
		return session.numericOperation(name, session.requireInstance(receiver), arguments)
	}
	arg := func(index int) any {
		if index < len(arguments) {
			return arguments[index]
		}
		return nil
	}
	switch name {
	case "write":
		session.writeText(session.render(arg(0), false, make(map[visit]bool)) + "\n")
		return Nothing
	case "Complex.real":
		return real(receiver.(complex128))
	case "Complex.imag":
		return imag(receiver.(complex128))
	case "Complex.conjugate":
		v := receiver.(complex128)
		return complex(real(v), -imag(v))
	case "Complex.magnitude":
		return math.Hypot(real(receiver.(complex128)), imag(receiver.(complex128)))
	case "Complex.phase":
		return math.Atan2(imag(receiver.(complex128)), real(receiver.(complex128)))
	case "take":
		if len(arguments) != 0 {
			session.writeText(arg(0).(string))
		}
		line, _ := session.Input.ReadString('\n')
		line = strings.TrimSuffix(line, "\n")
		return strings.TrimSuffix(line, "\r")
	case "len":
		switch value := arg(0).(type) {
		case string:
			return int64(len([]rune(value)))
		case *List:
			return int64(len(session.requireList(value).Items))
		case *Pair:
			return int64(len(session.requirePair(value).Values))
		}
	case "clear":
		switch value := arg(0).(type) {
		case *List:
			session.requireMutable(value)
			value.Items = nil
		case *Pair:
			session.requireMutable(value)
			value.Keys, value.Values = nil, make(map[any]any)
		}
		return Nothing
	case "between":
		start, stop, step := int64(0), arg(0).(int64), int64(1)
		if len(arguments) >= 2 {
			start, stop = arg(0).(int64), arg(1).(int64)
		}
		if len(arguments) >= 3 {
			step = arg(2).(int64)
		}
		if step == 0 {
			session.raise("DomainError", "between requires a non-zero step")
		}
		return &Range{Current: start, Stop: stop, Step: step}
	case "abs":
		switch value := arg(0).(type) {
		case int64:
			if value == math.MinInt64 {
				session.raise("OverflowError", "abs overflowed signed 64-bit Int range")
			}
			if value < 0 {
				return -value
			}
			return value
		case float64:
			return math.Abs(value)
		}
	case "sum", "min", "max":
		return session.reduction(name, session.requireList(arg(0)))
	case "List.add":
		list := session.requireList(receiver)
		session.requireMutable(list)
		list.Items = append(list.Items, arg(0))
		return Nothing
	case "List.eject":
		list := session.requireList(receiver)
		session.requireMutable(list)
		position := resolveIndex(session, arg(0).(int64), len(list.Items))
		list.Items = append(list.Items[:position], list.Items[position+1:]...)
		return Nothing
	case "Pair.eject":
		pair := session.requirePair(receiver)
		session.requireMutable(pair)
		if !pairDelete(pair, arg(0)) {
			session.raise("KeyError", "Pair key was not found")
		}
		return Nothing
	}
	if strings.HasPrefix(name, "String.") {
		return session.stringOperation(strings.TrimPrefix(name, "String."), receiver.(string), arguments)
	}
	if strings.HasPrefix(name, "List.") {
		return session.listOperation(strings.TrimPrefix(name, "List."), session.requireList(receiver), arguments)
	}
	if strings.HasPrefix(name, "DateTime.") || strings.HasPrefix(name, "Calendar.") {
		return session.timeOperation(name, receiver, arguments)
	}
	if strings.HasPrefix(name, "Regex.") {
		return session.regexOperation(name, receiver, arguments)
	}
	if strings.HasPrefix(name, "Table.") {
		return session.dataOperation(name, receiver, arguments)
	}
	if strings.HasPrefix(name, "Chart.") || strings.HasPrefix(name, "Figure.") {
		return session.plotOperation(name, receiver, arguments)
	}
	if strings.HasPrefix(name, "Document.") {
		return session.wordOperation(name, receiver, arguments)
	}
	if strings.HasPrefix(name, "PDFDocument.") {
		return session.pdfOperation(name, receiver, arguments)
	}
	if strings.HasPrefix(name, "Workbook.") || strings.HasPrefix(name, "Sheet.") || strings.HasPrefix(name, "Cell.") ||
		strings.HasPrefix(name, "Range.") || strings.HasPrefix(name, "CellStyle.") {
		return session.excelOperation(name, receiver, arguments)
	}
	if strings.HasPrefix(name, "JSONValue.") {
		return session.jsonOperation(name, receiver, arguments)
	}
	if strings.HasPrefix(name, "XMLNode.") || strings.HasPrefix(name, "XMLDocument.") {
		return session.xmlOperation(name, receiver, arguments)
	}
	if strings.HasPrefix(name, "Database.") || strings.HasPrefix(name, "SQLiteValue.") {
		return session.sqliteOperation(name, receiver, arguments)
	}
	if strings.HasPrefix(name, "Server.") || strings.HasPrefix(name, "Request.") || strings.HasPrefix(name, "Response.") ||
		strings.HasPrefix(name, "Cookie.") || strings.HasPrefix(name, "SessionStore.") || strings.HasPrefix(name, "Session.") ||
		strings.HasPrefix(name, "Client.") || strings.HasPrefix(name, "ClientRequest.") || strings.HasPrefix(name, "ClientResponse.") || strings.HasPrefix(name, "UploadedFile.") {
		return session.httpOperation(name, receiver, arguments)
	}
	if strings.HasPrefix(name, "HTMLDocument.") || strings.HasPrefix(name, "HTMLElement.") {
		return session.htmlOperation(name, receiver, arguments)
	}
	session.raise("Error", "unsupported Fundamentals operation "+name)
	return nil
}

func (session *Session) stringOperation(name, text string, arguments []any) any {
	arg := func(index int) string { return arguments[index].(string) }
	switch name {
	case "trim":
		return strings.TrimSpace(text)
	case "lower":
		return strings.ToLower(text)
	case "upper":
		return strings.ToUpper(text)
	case "capitalize":
		runes := []rune(text)
		if len(runes) != 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		return string(runes)
	case "split":
		if arg(0) == "" {
			session.raise("DomainError", "split requires a non-empty separator")
		}
		parts := strings.Split(text, arg(0))
		items := make([]any, len(parts))
		for index := range parts {
			items[index] = parts[index]
		}
		return &List{Items: items}
	case "replace":
		if arg(0) == "" {
			session.raise("DomainError", "replace requires non-empty search text")
		}
		return strings.ReplaceAll(text, arg(0), arg(1))
	case "contains":
		return strings.Contains(text, arg(0))
	case "startsWith":
		return strings.HasPrefix(text, arg(0))
	case "endsWith":
		return strings.HasSuffix(text, arg(0))
	case "count":
		if arg(0) == "" {
			session.raise("DomainError", "count requires a non-empty substring")
		}
		return int64(strings.Count(text, arg(0)))
	case "index":
		position := strings.Index(text, arg(0))
		if position < 0 {
			session.raise("DomainError", "index did not find the substring")
		}
		return int64(len([]rune(text[:position])))
	}
	return nil
}

func (session *Session) listOperation(name string, list *List, arguments []any) any {
	switch name {
	case "reverse":
		session.requireMutable(list)
		for left, right := 0, len(list.Items)-1; left < right; left, right = left+1, right-1 {
			list.Items[left], list.Items[right] = list.Items[right], list.Items[left]
		}
		return Nothing
	case "shuffle":
		session.requireMutable(list)
		for index := len(list.Items) - 1; index > 0; index-- {
			selected := int(session.randomInt(0, int64(index)))
			list.Items[index], list.Items[selected] = list.Items[selected], list.Items[index]
		}
		return Nothing
	case "count":
		count := int64(0)
		for _, item := range list.Items {
			if session.equal(item, arguments[0]) {
				count++
			}
		}
		return count
	case "index":
		for index, item := range list.Items {
			if session.equal(item, arguments[0]) {
				return int64(index)
			}
		}
		session.raise("DomainError", "index did not find the value in the List")
	case "map":
		items := append([]any(nil), list.Items...)
		result := &List{Items: make([]any, 0, len(items))}
		callback := arguments[0].(*FunctionValue)
		for _, item := range items {
			result.Items = append(result.Items, session.invoke(callback, []argumentValue{{value: item}}))
		}
		return result
	case "filter":
		items := append([]any(nil), list.Items...)
		result := &List{}
		callback := arguments[0].(*FunctionValue)
		for _, item := range items {
			if session.boolean(session.invoke(callback, []argumentValue{{value: item}})) {
				result.Items = append(result.Items, item)
			}
		}
		return result
	case "sort":
		session.requireMutable(list)
		items := append([]any(nil), list.Items...)
		keys := append([]any(nil), items...)
		if len(arguments) == 1 {
			callback := arguments[0].(*FunctionValue)
			for index, item := range items {
				keys[index] = session.invoke(callback, []argumentValue{{value: item}})
				if keys[index] == nil {
					session.raise("NullError", "sort key Function returned null")
				}
			}
		}
		order := make([]int, len(items))
		for index := range order {
			order[index] = index
		}
		sort.SliceStable(order, func(left, right int) bool { return orderedLess(keys[order[left]], keys[order[right]]) })
		for index, original := range order {
			list.Items[index] = items[original]
		}
		return Nothing
	}
	return nil
}

func orderedLess(left, right any) bool {
	switch value := left.(type) {
	case int64:
		return value < right.(int64)
	case float64:
		return value < right.(float64)
	case string:
		return value < right.(string)
	}
	return false
}

func (session *Session) reduction(name string, list *List) any {
	if len(list.Items) == 0 {
		// evalCall handles the statically typed empty-sum identity value.
		session.raise("DomainError", name+" requires a non-empty List")
	}
	for _, item := range list.Items {
		if item == nil {
			session.raise("NullError", name+" does not accept a null List element")
		}
	}
	result := list.Items[0]
	if name == "sum" {
		if _, real := result.(float64); real {
			result = float64(0)
		} else {
			result = int64(0)
		}
	}
	for index, item := range list.Items {
		if name == "sum" {
			if left, ok := result.(int64); ok {
				result = session.intAdd(left, item.(int64))
			} else {
				result = session.realCheck(result.(float64)+item.(float64), "addition")
			}
		} else if index > 0 && (name == "min" && orderedLess(item, result) || name == "max" && orderedLess(result, item)) {
			result = item
		}
	}
	return result
}

func (session *Session) math(name string, arguments []any) any {
	argReal := func(index int) float64 { return arguments[index].(float64) }
	switch name {
	case "round":
		value := argReal(0)
		if len(arguments) == 1 {
			return session.realCheck(math.Round(value), "round")
		}
		digits := arguments[1].(int64)
		if digits < 0 || digits > 15 {
			session.raise("DomainError", "round digits must be between 0 and 15")
		}
		factor := math.Pow10(int(digits))
		return session.realCheck(math.Round(value*factor)/factor, "round")
	case "floor":
		return session.mathIntegral("floor", math.Floor(argReal(0)))
	case "ceil":
		return session.mathIntegral("ceil", math.Ceil(argReal(0)))
	case "sqrt":
		if argReal(0) < 0 {
			session.raise("DomainError", "sqrt requires a non-negative value")
		}
		return math.Sqrt(argReal(0))
	case "sin":
		return math.Sin(argReal(0))
	case "cos":
		return math.Cos(argReal(0))
	case "tan":
		return session.realCheck(math.Tan(argReal(0)), "tan")
	case "log":
		if argReal(0) <= 0 {
			session.raise("DomainError", "log requires a positive value")
		}
		return math.Log(argReal(0))
	case "log10":
		if argReal(0) <= 0 {
			session.raise("DomainError", "log10 requires a positive value")
		}
		return math.Log10(argReal(0))
	case "exp":
		return session.realCheck(math.Exp(argReal(0)), "exp")
	case "seed":
		session.rngState = uint64(arguments[0].(int64))
		return Nothing
	case "random":
		return float64(session.randomRaw()>>11) * (1.0 / (1 << 53))
	case "randomInt":
		return session.randomInt(arguments[0].(int64), arguments[1].(int64))
	}
	session.raise("Error", "unsupported Math operation "+name)
	return nil
}

func (session *Session) mathIntegral(name string, value float64) int64 {
	if math.IsNaN(value) {
		session.raise("DomainError", "Math."+name+" produced an undefined result")
	}
	if math.IsInf(value, 0) || value < -9223372036854775808.0 || value >= 9223372036854775808.0 {
		session.raise("OverflowError", "Math."+name+" result does not fit Int")
	}
	return int64(value)
}

func (session *Session) randomRaw() uint64 {
	session.rngState += 0x9e3779b97f4a7c15
	value := session.rngState
	value = (value ^ value>>30) * 0xbf58476d1ce4e5b9
	value = (value ^ value>>27) * 0x94d049bb133111eb
	return value ^ value>>31
}

func (session *Session) randomInt(minimum, maximum int64) int64 {
	if minimum > maximum {
		session.raise("DomainError", "randomInt minimum must not exceed maximum")
	}
	if minimum == maximum {
		return minimum
	}
	width := uint64(maximum) - uint64(minimum) + 1
	if width == 0 {
		return int64(session.randomRaw())
	}
	limit := -width % width
	for {
		value := session.randomRaw()
		if value >= limit {
			return int64(uint64(minimum) + value%width)
		}
	}
}

func (session *Session) timeBuiltin(name string, arguments []any) any {
	// Full Time value construction is handled in time.go; retaining this
	// dispatch point keeps all standard modules on the same evaluator path.
	return session.evalTime(name, arguments)
}
