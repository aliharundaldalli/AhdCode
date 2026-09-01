package evaluator

import "strconv"

// The KeyValue standard module. Every operation is a pure structural
// transformation of the core ordered Pair type: it reads its source Pair and
// builds a new one, carrying key and value references over unchanged. The
// messages and error classes match the native backend's helpers exactly, so a
// program reports the same failure through the REPL and a compiled binary.

func (session *Session) keyValueBuiltin(name string, arguments []any) any {
	if name == "combine" {
		keys := session.requireList(arguments[0])
		values := session.requireList(arguments[1])
		if len(keys.Items) != len(values.Items) {
			session.raise("KeyValueError", "combine requires equal lengths; received "+
				strconv.Itoa(len(keys.Items))+" key(s) and "+strconv.Itoa(len(values.Items))+" value(s)")
		}
		result := &Pair{Values: make(map[any]any)}
		for index, key := range keys.Items {
			if _, exists := result.Values[key]; exists {
				session.raise("KeyValueError", "combine received the duplicate key "+keyText(key))
			}
			result.Keys = append(result.Keys, key)
			result.Values[key] = values.Items[index]
		}
		return result
	}
	pair := session.requirePair(arguments[0])
	switch name {
	case "keys":
		return &List{Items: append([]any(nil), pair.Keys...)}
	case "values":
		items := make([]any, len(pair.Keys))
		for index, key := range pair.Keys {
			items[index] = pair.Values[key]
		}
		return &List{Items: items}
	case "with":
		result := copyPair(pair)
		if _, exists := result.Values[arguments[1]]; !exists {
			result.Keys = append(result.Keys, arguments[1])
		}
		result.Values[arguments[1]] = arguments[2]
		return result
	case "without":
		if _, exists := pair.Values[arguments[1]]; !exists {
			session.raise("KeyError", "Pair has no key "+keyText(arguments[1]))
		}
		result := &Pair{Values: make(map[any]any)}
		for _, key := range pair.Keys {
			if key == arguments[1] {
				continue
			}
			result.Keys = append(result.Keys, key)
			result.Values[key] = pair.Values[key]
		}
		return result
	case "select", "drop":
		keys := session.requireList(arguments[1])
		requested := session.requestedKeys(name, pair, keys)
		result := &Pair{Values: make(map[any]any)}
		if name == "select" {
			for _, key := range keys.Items {
				result.Keys = append(result.Keys, key)
				result.Values[key] = pair.Values[key]
			}
			return result
		}
		for _, key := range pair.Keys {
			if requested[key] {
				continue
			}
			result.Keys = append(result.Keys, key)
			result.Values[key] = pair.Values[key]
		}
		return result
	case "rename":
		oldKey, newKey := arguments[1], arguments[2]
		if _, exists := pair.Values[oldKey]; !exists {
			session.raise("KeyError", "Pair has no key "+keyText(oldKey))
		}
		if oldKey == newKey {
			return copyPair(pair)
		}
		if _, exists := pair.Values[newKey]; exists {
			session.raise("KeyValueError", "rename cannot rename "+keyText(oldKey)+" to "+keyText(newKey)+
				"; that key already exists")
		}
		result := &Pair{Values: make(map[any]any)}
		for _, key := range pair.Keys {
			if key == oldKey {
				result.Keys = append(result.Keys, newKey)
				result.Values[newKey] = pair.Values[oldKey]
				continue
			}
			result.Keys = append(result.Keys, key)
			result.Values[key] = pair.Values[key]
		}
		return result
	case "mapValues":
		callback := arguments[1].(*FunctionValue)
		result := &Pair{Values: make(map[any]any)}
		for _, key := range append([]any(nil), pair.Keys...) {
			result.Keys = append(result.Keys, key)
			result.Values[key] = session.invoke(callback, []argumentValue{{value: pair.Values[key]}})
		}
		return result
	case "merge":
		right := session.requirePair(arguments[1])
		result := copyPair(pair)
		for _, key := range right.Keys {
			if _, exists := result.Values[key]; exists {
				session.raise("KeyValueError", "merge received the key "+keyText(key)+" in both Pairs")
			}
			result.Keys = append(result.Keys, key)
			result.Values[key] = right.Values[key]
		}
		return result
	case "overlay":
		changes := session.requirePair(arguments[1])
		result := copyPair(pair)
		for _, key := range changes.Keys {
			if _, exists := result.Values[key]; !exists {
				result.Keys = append(result.Keys, key)
			}
			result.Values[key] = changes.Values[key]
		}
		return result
	}
	session.raise("Error", "unsupported KeyValue operation "+name)
	return nil
}

// requestedKeys validates a requested key List left to right: every key must
// exist in the source Pair, and no key may be requested twice.
func (session *Session) requestedKeys(operation string, pair *Pair, keys *List) map[any]bool {
	requested := make(map[any]bool, len(keys.Items))
	for _, key := range keys.Items {
		if _, exists := pair.Values[key]; !exists {
			session.raise("KeyError", "Pair has no key "+keyText(key))
		}
		if requested[key] {
			session.raise("KeyValueError", operation+" received the duplicate key "+keyText(key))
		}
		requested[key] = true
	}
	return requested
}

// copyPair is the shallow structural copy every KeyValue operation starts from.
func copyPair(pair *Pair) *Pair {
	result := &Pair{Keys: append([]any(nil), pair.Keys...), Values: make(map[any]any, len(pair.Values))}
	for key, value := range pair.Values {
		result.Values[key] = value
	}
	return result
}

// keyText renders one Pair key inside a diagnostic, matching the runtime's
// ahdKeyText so both execution paths report identical messages.
func keyText(key any) string {
	switch typed := key.(type) {
	case string:
		return strconv.Quote(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return ""
	}
}
