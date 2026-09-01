# JSON standard module

[English] · [Türkçe](JSON_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [XML](XML.md)

`JSON` is the compiler-registered `builtin:JSON` module. It is explicit and a
sibling `JSON.ahd` cannot shadow it:

```ahd
bring JSON
from JSON bring JSONValue
from JSON bring JSONError
```

JSON introduces no `Any`, no dynamic typing, and no reflection into AhdCode.
`JSONValue` is a closed, statically typed, immutable recursive value model;
every operation on it has a fixed, declared type, and a value never silently
becomes an unrelated type.

## The JSONValue model

`JSONValue` represents exactly the seven kinds JSON itself defines:

```text
Null
Bool
Int
Real
String
Array   (a List<JSONValue>)
Object  (a Pair<String, JSONValue>, insertion order preserved)
```

There is no eighth kind and no open extension point. A `JSONValue` is
immutable: every accessor that itself returns a `JSONValue` (or a
`List<JSONValue>`/`Pair<String, JSONValue>` of them) returns a fresh,
independent value, never an alias into the receiver.

## Parsing and reading

```text
JSON.parse(source: String) -> JSONValue
JSON.read(path: String)    -> JSONValue
```

A JSON document must contain exactly one top-level value; trailing
non-whitespace content is rejected. Object keys are duplicated-checked:
`{"a":1,"a":2}` raises `JSONError` rather than silently keeping the last
value. Object insertion order and Array order are always preserved. Standard
JSON escape sequences (`\"`, `\\`, `\/`, `\b`, `\f`, `\n`, `\r`, `\t`, and
`\uXXXX`, including surrogate pairs) are supported. `NaN` and `Infinity` are
not JSON literals and are rejected like any other malformed input.

Number literals are read as exactly two kinds:

```text
91      -> Int
-12     -> Int
3.14    -> Real
1e3     -> Real
1.0     -> Real
```

A lexeme is `Real` exactly when it has a fraction (`.`) or an exponent
(`e`/`E`); otherwise it is `Int`. An integer lexeme that does not fit
AhdCode's `Int` range raises `JSONError` — it never silently becomes a
`Real` or any other type. A real literal that is not finite once parsed
(effectively infinite) is likewise rejected.

Parsing is bounded: input larger than 8&nbsp;MiB, and Array/Object nesting
past 256 levels, both raise `JSONError` before completing.

Because AhdCode's ordinary quoted strings interpret `{` and `}` as
interpolation delimiters, write literal JSON text as a raw String so the
braces are ordinary characters:

```ahd
document: JSONValue := JSON.parse(r'{"a":1}')
```

(A raw String has no escape mechanism, so it cannot contain its own
delimiter — use `r'...'` for JSON text, which always contains `"`.)

## Construction

`JSONValue` is never constructed by converting an ordinary `String`, `Int`,
`Real`, `Bool`, `List`, or `Pair` implicitly. Every `JSONValue` is built by an
explicit JSON function:

```text
JSON.nullValue()                              -> JSONValue
JSON.fromBool(value: Bool)                    -> JSONValue
JSON.fromInt(value: Int)                      -> JSONValue
JSON.fromReal(value: Real)                    -> JSONValue
JSON.fromString(value: String)                -> JSONValue
JSON.array(values: List<JSONValue>)           -> JSONValue
JSON.object(values: Pair<String, JSONValue>)  -> JSONValue
```

(`JSON.nullValue()`, not `JSON.null()`: `null` is a reserved keyword in every
syntactic context, so it can never be written as a member name after `.`.)

`JSON.fromReal` rejects a non-finite `Real` (NaN or infinite) with
`JSONError` — AhdCode arithmetic that can produce one must convert it
explicitly, never implicitly, before handing it to JSON.

```ahd
student: JSONValue := JSON.object({
    "name": JSON.fromString("Ali")
    "score": JSON.fromInt(91)
    "active": JSON.fromBool(true)
})
```

## Accessors

```text
kind()   -> String
isNull() -> Bool

bool()   -> Bool
int()    -> Int
real()   -> Real
string() -> String

array()  -> List<JSONValue>
object() -> Pair<String, JSONValue>

get(key: String)   -> JSONValue?
at(index: Int)     -> JSONValue
```

`kind()` returns exactly one of `"Null"`, `"Bool"`, `"Int"`, `"Real"`,
`"String"`, `"Array"`, or `"Object"`.

Every accessor except `get` raises `JSONError` when the receiver's kind does
not match:

```ahd
attempt {
    JSON.fromString("x").int()
} except JSONError as error {
    write(error.message)
}
```

`real()` is the one deliberate exception to "wrong kind raises `JSONError`":
it also accepts an `Int` receiver and returns it widened to `Real`, the same
safe `Int -> Real` widening AhdCode already performs elsewhere. `int()` never
does the reverse — a `Real` value with an integer-looking value (`5.0`) still
raises `JSONError` from `int()`.

`get(key)` is `Object`-only (a non-`Object` receiver raises `JSONError`, the
same as every other wrong-kind access) and returns `JSONValue?`: `null` means
the key is absent, never that the key's value is JSON's own `Null` — a
present key holding `JSON.nullValue()` still returns a non-null `JSONValue`
whose `kind()` is `"Null"`.

`at(index)` is `Array`-only and follows ordinary List index rules (a negative
index counts back from the end); an out-of-range index raises `JSONError`.

```ahd
name := parsed.get("name")
if name != null {
    write(name.string())
}
```

## Serialization

```text
JSON.stringify(value: JSONValue, pretty: Bool = false) -> String
JSON.write(value: JSONValue, path: String, pretty: Bool = false) -> Nothing
```

Compact output (`pretty = false`, the default) has no insignificant
whitespace. Pretty output uses a fixed two-space indent. Both modes:

- preserve Array order and Object insertion order;
- escape String content correctly;
- never emit `NaN` or `Infinity` (unreachable in practice, since
  `JSON.fromReal` already rejects them at construction);
- produce valid JSON, deterministically — stringifying the same `JSONValue`
  twice always produces the same text, and `parse(stringify(value))` is
  semantically equal to `value`.

`JSON.write` stages its output and publishes it atomically, the same
temp-file-then-rename convention Word uses for `.docx`: a failed write never
disturbs a file that was already at the destination.

```ahd
text: String := JSON.stringify(student, true)
JSON.write(student, "student.json", true)
```

## Errors

`JSONError` derives directly from `Error` and covers every JSON-specific
failure: malformed input, duplicate keys, trailing content, depth/size
limits, integer overflow, a non-finite `Real`, a wrong-kind accessor, an
out-of-range `Array` index, and a missing/unreadable/unwritable file.
`JSON.read`/`JSON.write` raise `JSONError` directly rather than `FileError`,
so one error type covers the whole module end to end.

```ahd
attempt {
    JSON.read("missing.json")
} except JSONError as error {
    write(error.message)
}
```

## Security and limits

`JSON.parse`/`JSON.read` never access the network and never execute anything;
they only walk the JSON grammar. Input is rejected past 8&nbsp;MiB, and
Array/Object nesting is rejected past 256 levels — both limits are
implementation details of this release and may be revisited, but are always
enforced before a malformed or adversarial document could otherwise cause
pathological recursion or unbounded memory use.

## Updating an object without String surgery

A JSON object value is an ordinary `Pair<String, JSONValue>`, so
[KeyValue](KEYVALUE.md) updates a document without ever leaving the typed
representation:

```ahd
object: Pair<String, JSONValue> := data.object()

updatedObject: Pair<String, JSONValue> := KeyValue.with(
    object
    "books"
    JSON.array(newBooks)
)

JSON.write(JSON.object(updatedObject), "library.json", true)
```

`KeyValue.with` keeps the replaced key in its existing position and leaves
every other root field untouched. This is the intended way to update a JSON
document; `JSON.stringify` → String concatenation → `JSON.parse` is not, and
this release deliberately adds no JSON-specific mutation API, because
`KeyValue` solves the problem generally. See
`examples/v0.1/49_json_record_update.ahd`.

## Non-goals

JSON is a typed data-interchange module, not a schema or query language:
there is no JSON Schema/JSON Pointer/JSONPath support, no streaming parser,
and no `Any`/dynamic escape hatch. When you need to filter, sort, or derive
over decoded data, convert explicitly into `Data`'s `Table` or ordinary
AhdCode values first.
