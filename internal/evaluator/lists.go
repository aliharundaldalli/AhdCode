package evaluator

import "strconv"

// The Lists standard module. Every operation is a pure structural
// transformation of a List: it reads its source List and builds a new one,
// carrying element references over unchanged. The messages and error classes
// match the native backend's helpers exactly, so a program reports the same
// failure through the REPL and through a compiled binary.

func (session *Session) listsBuiltin(name string, arguments []any) any {
	switch name {
	case "chunk":
		values := session.requireList(arguments[0])
		size := arguments[1].(int64)
		if size <= 0 {
			session.raise("ListsError", "chunk requires a size greater than zero; received "+strconv.FormatInt(size, 10))
		}
		result := &List{}
		length := int64(len(values.Items))
		for start := int64(0); start < length; start += size {
			end := start + size
			if end > length || end < start {
				end = length
			}
			result.Items = append(result.Items, &List{Items: append([]any(nil), values.Items[start:end]...)})
		}
		return result
	case "flatten":
		rows := session.requireList(arguments[0])
		result := &List{}
		for _, row := range rows.Items {
			result.Items = append(result.Items, session.requireList(row).Items...)
		}
		return result
	case "transpose":
		return session.listsTranspose(session.requireList(arguments[0]))
	case "unique":
		values := session.requireList(arguments[0])
		result := &List{}
		for _, item := range values.Items {
			if !session.containsEqual(result.Items, item) {
				result.Items = append(result.Items, item)
			}
		}
		return result
	case "valueCounts":
		values := session.requireList(arguments[0])
		result := &Pair{Values: make(map[any]any)}
		for _, item := range values.Items {
			count, exists := result.Values[item]
			if !exists {
				result.Keys = append(result.Keys, item)
				result.Values[item] = int64(1)
				continue
			}
			result.Values[item] = count.(int64) + 1
		}
		return result
	case "groupBy":
		values := session.requireList(arguments[0])
		callback := arguments[1].(*FunctionValue)
		// A shallow snapshot, exactly like List.map/filter: a callback that
		// structurally mutates the source cannot change what is iterated.
		items := append([]any(nil), values.Items...)
		result := &Pair{Values: make(map[any]any)}
		for _, item := range items {
			key := session.invoke(callback, []argumentValue{{value: item}})
			group, exists := result.Values[key]
			if !exists {
				group = &List{}
				result.Keys = append(result.Keys, key)
				result.Values[key] = group
			}
			list := group.(*List)
			list.Items = append(list.Items, item)
		}
		return result
	}
	session.raise("Error", "unsupported Lists operation "+name)
	return nil
}

func (session *Session) listsTranspose(rows *List) any {
	result := &List{}
	if len(rows.Items) == 0 {
		return result
	}
	width := len(session.requireList(rows.Items[0]).Items)
	for index, row := range rows.Items {
		length := len(session.requireList(row).Items)
		if length != width {
			session.raise("ListsError", "transpose requires rectangular rows: row "+strconv.Itoa(index)+
				" has "+strconv.Itoa(length)+" element(s); expected "+strconv.Itoa(width))
		}
	}
	for column := 0; column < width; column++ {
		items := make([]any, len(rows.Items))
		for index, row := range rows.Items {
			items[index] = session.requireList(row).Items[column]
		}
		result.Items = append(result.Items, &List{Items: items})
	}
	return result
}

func (session *Session) containsEqual(items []any, value any) bool {
	for _, item := range items {
		if session.equal(item, value) {
			return true
		}
	}
	return false
}
