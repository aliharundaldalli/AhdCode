package evaluator

import "ahdcode/internal/backend/golang/ahdruntime"

// The Characters standard module's REPL implementation. It calls the native
// runtime's ahdruntime/characters.go directly, so the evaluator and a compiled
// program share one definition of every operation and every error message.

func (session *Session) charactersBuiltin(name string, args []any) any {
	defer session.charactersRecover()
	class := ahdruntime.AhdClassCharactersError
	character := func() string { return args[0].(string) }
	switch name {
	case "list":
		items := ahdruntime.AhdCharactersList(class, args[0].(string)).Snapshot()
		values := make([]any, len(items))
		for index, item := range items {
			values[index] = item
		}
		return &List{Items: values}
	case "count":
		return ahdruntime.AhdCharactersCount(class, args[0].(string))
	case "codePoint":
		return ahdruntime.AhdCharactersCodePoint(class, character())
	case "fromCodePoint":
		return ahdruntime.AhdCharactersFromCodePoint(class, args[0].(int64))
	case "isLetter":
		return ahdruntime.AhdCharactersIsLetter(class, character())
	case "isDigit":
		return ahdruntime.AhdCharactersIsDigit(class, character())
	case "isWhitespace":
		return ahdruntime.AhdCharactersIsWhitespace(class, character())
	case "isUpper":
		return ahdruntime.AhdCharactersIsUpper(class, character())
	case "isLower":
		return ahdruntime.AhdCharactersIsLower(class, character())
	case "isAlphaNumeric":
		return ahdruntime.AhdCharactersIsAlphaNumeric(class, character())
	case "isPunctuation":
		return ahdruntime.AhdCharactersIsPunctuation(class, character())
	case "isSymbol":
		return ahdruntime.AhdCharactersIsSymbol(class, character())
	}
	session.raise("Error", "unsupported Characters function "+name)
	return nil
}

// charactersRecover turns a runtime CharactersError signal into the
// evaluator's own catchable CharactersError with the identical message.
func (session *Session) charactersRecover() {
	recovered := recover()
	if recovered == nil {
		return
	}
	if signal, ok := recovered.(*ahdruntime.AhdSignal); ok {
		session.raise("CharactersError", signal.Message)
	}
	panic(recovered)
}
