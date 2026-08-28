# AhdCode Language Specification v0.1

**Status:** Draft frozen core specification for first implementation  
**Clarification revision:** 2026-08-28; clarified rules are normative for v0.1  
**Primary implementation target:** Go  
**File extension:** `.ahd`  
**Initial scope:** terminal/CLI language only. Web, HTTP, MySQL, SMTP, HTML layouts, JSON-specific web conveniences, and AhdWeb are explicitly deferred until the core language works reliably.

---

## 1. Design Philosophy

AhdCode is designed around a few strong rules:

1. **Readability over minimum line count.**
2. **Use plain English words when a short technical abbreviation adds no value.**
3. **Do not silently coerce unrelated types.**
4. **Make declaration and mutation visibly different.**
5. **Allow concise syntax when it is unambiguous, but provide one canonical formatter style.**
6. **Do not add a special language feature when an ordinary function can solve the problem cleanly.**
7. **Do not add features only because C, Python, Java, JavaScript, or another language has them.**
8. **The compiler may infer omitted type details only when it can do so safely and uniquely. It must never fall back to `Any`/dynamic behavior merely to make code compile.**
9. **The parser may accept several harmless presentation forms; the formatter decides what canonical AhdCode looks like.**
10. **Core language first. Web comes later as a runtime/library layer, not as the foundation of the grammar.**

AhdCode should feel approachable like Python, visually structured like C-family languages, and statically controlled without excessive ceremony.

---

## 2. Case Sensitivity and Identifiers

AhdCode is case-sensitive.

```ahd
student
Student
STUDENT
```

are three different identifiers.

AhdCode source files are UTF-8. Invalid UTF-8 is a lexical error.

Identifier rules follow Python's Unicode identifier model:

- the first character is `_` or a Unicode `XID_Start` code point;
- each following character is `_` or a Unicode `XID_Continue` code point;
- before identifier comparison and symbol lookup, identifiers are normalized with Unicode NFKC;
- normalization does not perform case folding; AhdCode remains case-sensitive;
- reserved AhdCode keywords cannot be used as identifiers.

Examples:

```ahd
öğrenci2: String := "Ali"
_private: Int := 5

Student: Class<> := {
}
```

Invalid:

```ahd
2student: Int := 5
```

### 2.1 Reserved and contextual keywords

AhdCode v0.1 reserves the following words in every syntactic context. They may not be used as identifiers with the same spelling and case:

```text
and or not same is in has
if else while until for break continue
state condition default
attempt except ultimately toss return
bring from all as
true false null
Int Real String Bool Nothing
List Pair Function Overload Override
Class Attributes Constant Local Global Confidential
Object Error
```

The following are contextual keywords:

| Word | Keyword context |
|---|---|
| `structure` | the `structure: Attributes := ...` declaration inside a Class |
| `attribute` | the implicit instance receiver inside Class structure bodies and methods |
| `SuperClass` | inherited attribute expansion and direct-super access inside a derived Class |

A contextual keyword has its special meaning only in the context listed above. Outside that context it may be used as an ordinary identifier.

Built-in and imported names such as `write`, `take`, `str`, `int`, `IndexError`, and module names are predeclared or imported identifiers, not reserved keywords.

---

## 3. Statements, Newlines, Commas, and Formatting

### 3.1 Statement boundaries

Newlines are significant and must remain available to the parser.

A newline terminates a statement unless the parser can determine that it occurs in a genuinely continuing expression or expression-level delimited construct. Inside a multiline call, collection literal, parameter list, generic argument list, index/slice, or grouped expression, a newline may instead separate items or continue the open construct according to that construct's grammar.

Block braces do **not** suppress statement newlines. Inside an executable `{ ... }` block, a newline continues to separate statements. The fact that a block's closing brace has not yet appeared does not make all lines in the block one expression.

A newline therefore terminates a **statement**, not necessarily an **expression**. The parser, rather than the lexer discarding newlines globally, makes this contextual decision.

A newline may continue an expression when an infix operator or an open expression delimiter makes the expression syntactically incomplete. Once an expression is complete, its following newline terminates the statement. A binary operator at the beginning of the next line never reaches backward to continue the preceding completed statement.

For example, this is not interpreted as `x = 5 + 2`:

```ahd
x = 5
+ 2
```

This is valid and preferred:

```ahd
area: Local Real := PI * square(radius)
```

AhdCode does not require `;`.

Multiple independent statements may not be packed onto one line.

Invalid:

```ahd
x: Int := 5 y: Int := 8 write(x + y)
```

### 3.2 Argument and element separators

For function arguments and collection elements:

- a comma may separate items on the same line;
- a newline may separate items;
- plain whitespace alone may **not** separate multiple arguments on the same line.

Valid:

```ahd
swap(a, b)
```

```ahd
swap(
    a
    b
)
```

```ahd
swap(a
    b)
```

Invalid:

```ahd
swap(a b)
```

The same concept applies to short versus multiline collection literals.

Valid:

```ahd
numbers: List<Int> := [1, 2, 3]
```

```ahd
numbers: List<Int> := [
    1
    2
    3
]
```

### 3.3 Canonical formatter policy

The parser is intentionally more permissive than the formatter.

The formatter should prefer:

- short calls/lists on one line with commas;
- multiline calls/lists with one item per line and no required trailing commas;
- multiline function declarations;
- consistent indentation;
- one statement per line.

Formatting must not change program semantics.

---

## 4. Comments

Single-line comment:

```ahd
// comment
```

Multiline comment:

```ahd
/*
multiline
comment
*/
```

Multiline comments do not nest. The first `*/` after an opening `/*` closes the comment. Any additional `/*` inside the comment is ordinary comment text.

Comments are formatter-preserved trivia. The lexer/token model must retain their source spans, text, and placement from the first implementation milestone; comments must not be irreversibly discarded before formatting.

---

## 5. Core Scalar Types

AhdCode v0.1 provides:

```text
Int
Real
String
Bool
```

### 5.1 Numeric literal grammar

v0.1 numeric literals use ASCII decimal digits only:

```text
digit       := "0" ... "9"
digits      := digit+
exponent    := ("e" | "E") ("+" | "-")? digits
IntLiteral  := digits
RealLiteral := digits "." digits exponent?
             | digits exponent
```

Examples:

```ahd
12
0012
1.25
1e6
1.2e-3
```

Leading zeroes are allowed and remain decimal. `_` separators, `.5`, `5.`, binary/octal/hexadecimal prefixes, numeric suffixes, `NaN`, and infinity literals are not part of v0.1.

A leading `+` or `-` is a unary operator, not part of the numeric token. The lexer preserves the literal text and does not reject an Int token merely because its unsigned magnitude is greater than the positive `Int` maximum.

Signed constant expressions are evaluated during semantic analysis. Their final Int value must fit signed 64-bit range. In particular:

```ahd
minimum: Int := -9223372036854775808
```

is valid, while a positive Int value above `9223372036854775807` is a compile-time semantic error. A Real literal that cannot produce a finite `float64` value is likewise a compile-time semantic error.

### 5.2 Int

`Int` is implemented as signed 64-bit integer (`int64` semantics).

```ahd
count: Int := 10
```

Integer overflow must not silently wrap. It raises an overflow runtime error.

### 5.3 Real

`Real` is implemented with 64-bit floating-point representation (`float64` semantics).

```ahd
pi: Real := 3.14159
x: Real := 5
```

`Int -> Real` is an allowed safe widening conversion.

The name `Real` is a language-level abstraction. It does **not** claim exact representation of the mathematical real numbers.

### 5.4 Division

`/` always performs real division and returns `Real`.

```ahd
5 / 2
```

produces:

```text
2.5
```

### 5.5 Explicit numeric conversions

The predeclared Fundamentals conversion `int` accepts exactly one `Real` or `String` and returns `Int`. A `Real` input truncates toward zero.

```ahd
int(3.7)   // 3
int(-3.7)  // -3
```

This is conversion, not nearest-integer rounding.

For `String` input, `int` trims surrounding Unicode whitespace and then accepts only this ASCII-decimal grammar:

```text
IntText := ("+" | "-")? digits
digits  := ASCII digit+
```

There is no decimal point, exponent, underscore, or hexadecimal/binary/octal form. Invalid text raises `DomainError`; a mathematically valid decimal outside signed 64-bit range raises `OverflowError`.

The predeclared Fundamentals conversion `real` accepts exactly one `Int` or `String` and returns `Real`. An `Int` input is the explicit spelling of the language's safe `Int -> Real` widening conversion.

```ahd
real(2)         // 2.0
real("3")       // 3.0
real("3.14")    // 3.14
real("1e3")     // 1000.0
real("-2.5e-4") // -0.00025
```

For `String` input, `real` trims surrounding Unicode whitespace and accepts the ordinary ASCII-decimal numeric-text grammar: an optional leading sign, one or more digits, an optional fraction with digits on both sides of `.`, and an optional `e` or `E` exponent with its own optional sign and one or more digits. Decimal integer text is accepted. Underscores, hexadecimal/binary/octal forms, `NaN`, and infinity spellings are not accepted. Invalid text raises `DomainError`; a parsed result that is non-finite or outside `float64` range raises `OverflowError`.

These conversion names do not introduce general coercion. In particular, numeric operators still reject Strings, and `bool(...)` conversion and truthiness remain outside the v0.1 contract.

### 5.6 Bool

Only `Bool` values may be used as conditions.

There is no Python/JavaScript/PHP-style truthiness.

Invalid:

```ahd
if 5 {
}
```

Invalid:

```ahd
if "Ali" {
}
```

Valid:

```ahd
if age > 18 {
}
```

### 5.7 String

Strings are immutable.

```ahd
name: String := "Ali"
name += " Harun"
```

is valid because a new String value is produced and rebound.

Invalid:

```ahd
name[0] = "V"
```

A String index returns a one-character `String`. There is no `Char` type in v0.1.

---

## 6. String Literals, Escapes, Unicode, and Interpolation

Single and double quoted strings are equivalent:

```ahd
a: String := "hello"
b: String := 'hello'
```

The delimiter that opens a normal string must also close it.

Invalid:

```ahd
"hello'
'hello"
```

The exact v0.1 escape set is:

```text
\n
\r
\t
\\
\"
\'
\{
\}
```

This same escape set applies to normal and multiline strings. Any other escape sequence is a lexical error. v0.1 has no `\0`, `\b`, `\f`, `\x...`, or `\u...` escape form; Unicode characters are written directly in source text.

A normal single- or double-quoted string may not contain a physical newline. A matching triple-quoted string may contain physical newlines.

### 6.1 Multiline strings

Both triple-double-quote and triple-single-quote strings are supported.

```ahd
query: String := """
SELECT *
FROM students
WHERE name = '{name}'
"""
```

```ahd
text: String := '''
"double quotes" are allowed here.
'''
```

Inside `""" ... """`, ordinary `'` and `"` characters are allowed. Only the matching triple delimiter closes the literal.

Triple-string content is preserved exactly as it appears between the opening and closing delimiters. AhdCode performs no automatic dedent, trim, or removal of the first or last newline.

Only three consecutive matching, unescaped quote characters close a triple string. An escaped quote does not participate in a closing delimiter.

### 6.2 Interpolation

All normal and multiline strings support `{...}` interpolation.

```ahd
name: String := "Ali"
write("Hello {name}")
```

Literal braces may be escaped:

```ahd
text: String := "\{literal\}"
```

In string-text mode, an unescaped `{` opens interpolation. The interpolation ends at its matching depth-zero `}`. Its contents must be one non-empty AhdCode expression; statements and declarations are not permitted.

Nested `()`, `[]`, and `{}` delimiters are supported inside the expression. Strings and comments inside the expression follow their normal lexical rules, and braces contained by them do not alter the surrounding interpolation depth. A string literal inside an interpolation may itself contain interpolation.

`\{` and `\}` produce literal braces in string text. An unescaped unmatched `}` in string text and an interpolation without a closing `}` are lexical errors reported at the relevant brace span.

An ordinary quoted string cannot contain a physical newline, including inside its interpolation. An interpolation in a triple-quoted string may span lines and follows the ordinary expression/newline rules.

The interpolation expression may produce any value accepted by `str`, but may not produce `Nothing`. Its textual conversion has the same semantics as `str(expression)` and does not introduce general String-number coercion for operators such as `+`.

---

## 7. Nothing and null

`Nothing` and `null` are intentionally different concepts.

### 7.1 Nothing

`Nothing` is AhdCode's equivalent of `void`.

It is a return type, not a runtime value.

```ahd
show: Function := (
    message: String
) -> Nothing {
    write(message)
    return
}
```

The trailing bare `return` is optional in a `Nothing` function.

Invalid concepts:

```ahd
x: Nothing := ...
return Nothing
```

### 7.2 null

`null` means:

> the declared type is known, but a value is currently absent.

Valid:

```ahd
name: String := null
age: Int := null
student: Student := null
```

The declared type remains unchanged.

```ahd
age = 25       // valid
age = "Ali"    // type error
```

`null` is not:

- zero,
- empty String,
- false,
- `Nothing`,
- an implicit dynamic type.

AhdCode performs **flow-sensitive null-state analysis** during semantic checking.

Internally, a binding may be tracked as one of these states:

```text
Null
MaybeNull
NonNull
```

These states are compiler analysis states, not public AhdCode types and not user-written syntax.

If the compiler can prove that a value is null or may be null, unsafe operations must be rejected at compile time whenever possible.

Example:

```ahd
age: Int := null
write(age + 5)
```

is a compile-time null-state error.

After a definite non-null assignment:

```ahd
age = 25
write(age + 5)
```

the value is considered `NonNull` and the operation is valid.

Branch conditions refine null state:

```ahd
student: Student := findStudent()

if student != null {
    write(student.name)
}
```

Inside the true branch, `student` is treated as non-null.

Short-circuit Boolean logic also participates in null-state refinement:

```ahd
if student != null and student.age >= 18 {
    write(student.name)
}
```

The right-hand side of `and` is checked under the knowledge that `student != null`.

If a function may return `null`, its call result is tracked as `MaybeNull` unless control-flow analysis proves otherwise.

Runtime null checks remain a final safety layer for cases that cannot be proven statically, but AhdCode should reject provably unsafe null usage during semantic analysis.

Null-state information must survive module boundaries. The compiler-visible contract for an exported binding and for every concrete exported callable signature must preserve the relevant `Null`, `MaybeNull`, or `NonNull` state, including callable return state. An imported expression begins semantic checking with that preserved state rather than being silently treated as non-null or dynamic.

This cross-module information is compiler metadata, not public AhdCode type syntax.

### 7.3 Constant null

A `Constant` may not be initialized to `null`.

Invalid:

```ahd
id: Constant Int := null
```

### 7.4 null in typed collections

`null` may appear where the surrounding type is known.

```ahd
names: List<String> := [
    "Ali"
    null
    "Ayşe"
]
```

A `null` key in a Pair is not allowed.

---

## 8. Declaration and Assignment

### 8.1 Declaration

A new named value uses:

```text
name: Type := value
```

Example:

```ahd
age: Int := 28
```

`:=` always indicates first declaration/binding.

### 8.2 Reassignment

An existing binding uses:

```ahd
age = 29
```

`=` does not declare a new variable.

### 8.3 No chained assignment

Invalid:

```ahd
a = b = c = 5
```

Invalid:

```ahd
a: Int := b: Int := 5
```

Write separate statements.

### 8.4 Compound assignment

Supported:

```text
+=
-=
*=
/=
%=
^=
```

Because `^` means exponentiation, `^=` means exponentiation assignment.

Compound assignment is valid only when the operator result is assignable back to the target's declared type.

Because `/` always produces `Real`:

- `Real /= Int` and `Real /= Real` are valid;
- `Int /= Int` and `Int /= Real` are compile-time type errors.

Because `%` exists only for `Int % Int`, `%=` requires both an `Int` target and an `Int` right operand.

### 8.5 Increment and decrement

Supported:

```ahd
i++
++i
i--
--i
```

`++` and `--` are defined only for `Int` bindings and may only appear as standalone statements.

Invalid:

```ahd
x: Int := ++i - j++
write(i++)
if ++i > 5 {
}
```

Prefix and postfix forms have the same effect in AhdCode because their use inside larger expressions is prohibited.

Using either form with a `Real` or any non-`Int` value is a compile-time type error.

---

## 9. Scope: Local, Global, and Shadowing

### 9.1 Parameters

Function and structure parameters are lexically local automatically. Function parameters do not use the `Local` modifier at the binding site. On a structure parameter, explicit `Local` has a separate Class-layout meaning: the constructor input remains lexically local but does not become an instance attribute. A `for` iteration variable and an `except ... as error` binding are also implicitly Local.

### 9.2 Local declarations

Only a declaration directly in module root scope omits a scope modifier:

```ahd
count: Int := 0
```

A new explicit declaration inside any executable lexical scope nested below module root uses `Local`. This includes function and method bodies, structure bodies, and module-level or callable-level `if`, loop, `state`, `attempt`, `except`, and `ultimately` blocks.

```ahd
result: Local Real := 0
```

The implicit binding introduced by `for` is scoped to the loop body. The implicit binding introduced by `except ... as error` is scoped to that except body.

A nested block may read and mutate a binding from an enclosing lexical scope within the same callable without declaring it `Global`.

### 9.3 Global access

A function must explicitly declare a global binding before using it, including read access.

Example:

```ahd
counter: Int := 0

increase: Function := (
) -> Nothing {
    counter: Global Int
    counter++
}
```

No hidden global capture is allowed.

`Global` refers specifically to a module-root binding from inside a function, method, or structure body. It is not used for an enclosing Local binding within the same callable. Code executing in a module-level control-flow block may access module-root bindings directly and does not declare them `Global`.

### 9.4 Shadowing

Shadowing across nested scopes is allowed.

```ahd
x: Int := 5

if true {
    x: Local Int := 20
    write(x)
}

write(x)
```

Redeclaring the same name in the same scope with `:=` is an error.

---

## 10. Constant

`Constant` is a modifier, not a separate type.

```ahd
PI: Constant Real := 3.14159
```

For scalar values, reassignment is forbidden.

For reference values, `Constant` deep-freezes the referenced object graph.

```ahd
numbers: Constant List<Int> := [1, 2, 3]
```

Invalid:

```ahd
numbers[0] = 50
numbers.add(4)
```

If another alias refers to the same object, it also cannot mutate the frozen object.

To modify an independent copy, a new copy must be explicitly created through library functionality such as `copy` or `deepCopy`.

A clone/copy is not automatically Constant unless declared Constant.

### 10.1 Compile-time constant expressions

A constant expression is a scalar expression that can be evaluated completely at compile time.

It may contain only:

- scalar literals;
- parentheses;
- unary `+`, `-`, and `not`;
- AhdCode's pure built-in scalar operators;
- references to scalar `Constant` bindings whose initializers are themselves constant expressions.

Function calls, mutable binding references, member or index access, List/Pair/Class construction, and interpolation are not constant expressions.

A cycle in the dependency graph of Constant initializers is a compile-time error.

This definition is normative wherever the language requires compile-time knowledge. Power typing does not depend on constant-expression evaluation.

---

## 11. Reference and Value Semantics

Value-like scalar types:

```text
Int
Real
Bool
String
```

Reference types:

```text
List
Pair
Class instances
```

Example:

```ahd
a: List<Int> := [1, 2, 3]
b: List<Int> := a

b[0] = 50
write(a[0])   // 50
```

### 11.1 Rebinding a function parameter

Reference objects are shared, but parameter bindings are local.

```ahd
change: Function := (
    values: List<Int>
) -> Nothing {
    values = [9, 9, 9]
}
```

This changes only the local binding `values`.

But:

```ahd
values[0] = 99
```

mutates the shared object.

---

## 12. List

`List<T>` is a homogeneous, dynamically sized ordered collection.

```ahd
numbers: List<Int> := [1, 2, 3]
```

All non-null elements must have the same element type.

Invalid:

```ahd
values: List := [1, "Ali"]
```

### 12.1 Type inference

Valid:

```ahd
numbers: List := [1, 2, 3]
```

Compiler infers:

```text
List<Int>
```

Invalid because the element type cannot be inferred:

```ahd
numbers: List := []
```

Use:

```ahd
numbers: List<Int> := []
```

A list containing only null values also requires explicit element type.

### 12.2 Indexing

```ahd
numbers[0]
numbers[-1]
```

Negative indexing is supported.

An invalid index raises `IndexError`.

### 12.3 Slicing

Supported:

```ahd
numbers[1:4]
numbers[:4]
numbers[2:]
numbers[-3:]
```

Slice-step syntax is not part of v0.1.

### 12.4 Mutation

`List<T>` has exactly two built-in mutation operations:

```text
add(value: T)      -> Nothing
eject(index: Int)  -> Nothing
```

`add` appends one element to the end.

```ahd
values: List<Int> := [
    10
    20
]

values.add(30)
```

produces:

```text
[10, 20, 30]
```

`eject` removes the element at an index.

```ahd
values: List<Int> := [
    10
    20
    30
]

values.eject(1)
```

produces:

```text
[10, 30]
```

`eject` accepts the same negative indexing as ordinary List indexing, so `values.eject(-1)` removes the final element. An out-of-range index raises `IndexError`. `eject` does not return the removed element in v0.1.

Both operations mutate the existing List object rather than producing a new one, so every alias observes the change. Both return `Nothing` and therefore cannot be used as values. The receiver must be `NonNull`, the argument follows ordinary element assignability, and a `Constant` or otherwise frozen List rejects both.

---

## 13. Pair

`Pair<K, V>` is AhdCode's ordered homogeneous key/value collection.

Generic type arguments on the same line must be separated with commas.

Valid:

```ahd
Pair<String, Int>
```

Invalid:

```ahd
Pair<String Int>
```

This follows the general separator rule of AhdCode: a comma may separate items on the same line, while a newline may separate items in multiline constructs. Plain whitespace alone is not a separator.

All keys have one key type `K`.  
All values have one value type `V`.

```ahd
scores: Pair<String, Int> := {
    "Ali": 85
    "Ayşe": 92
}
```

Pair is **not** a heterogeneous object/record container.

### 13.1 Nested Pair

Allowed:

```ahd
grades: Pair<String, Pair<String, Real>> := {
    "Ali": {
        "Analysis": 85.5
        "Algebra": 90
    }
}
```

Partial inference is allowed where safe:

```ahd
data: Pair<String, Pair> := {
    ...
}
```

### 13.2 Empty Pair

Invalid:

```ahd
scores: Pair := {}
```

Valid:

```ahd
scores: Pair<String, Int> := {}
```

### 13.3 Key types

v0.1 Pair keys may use stable simple scalar types such as:

```text
String
Int
Bool
```

`Real`, class instances, and `null` are not Pair keys in v0.1.

### 13.4 Missing keys

Accessing a missing key raises `KeyError`.

### 13.5 Ordering

Pair preserves insertion order.

Adding a new key appends it to the end.

Updating an existing key does not move it.

Removing a key and later adding it again appends it as a new final entry.

### 13.6 Duplicate keys in literals

Duplicate keys inside a single Pair literal are compile-time errors.

### 13.7 Mutation

`Pair<K, V>` is inserted into and updated through index assignment, and has one
built-in removal operation:

```text
pair[key] = value
eject(key: K)      -> Nothing
```

```ahd
scores: Pair<String, Int> := {}

scores["Ali"] = 85
scores["Ayşe"] = 92
```

Assigning an absent key appends a new entry at the end. Assigning a key that
already exists updates its value without moving it.

`eject` removes one key and its value.

```ahd
scores.eject("Ali")
```

produces:

```text
{"Ayşe": 92}
```

Ejecting a key that is not present raises `KeyError`. Combined with §13.5, an
ejected key that is added again becomes a new final entry:

```ahd
scores["Ali"] = 85
scores["Ayşe"] = 92
scores.eject("Ali")
scores["Ali"] = 100
```

leaves the order:

```text
Ayşe
Ali
```

There is no `pair.add`. `eject` mutates the existing Pair object, so every alias
observes the change; it returns `Nothing` and cannot be used as a value. The
receiver must be `NonNull`, the key follows the ordinary Pair key rules of
§13.3, and a `Constant` or otherwise frozen Pair rejects insertion, update, and
`eject`.

---

## 14. Generic Invariance

Safe scalar widening:

```text
Int -> Real
```

is allowed.

Mutable generic collections are invariant.

Invalid:

```ahd
integers: List<Int> := [1, 2, 3]
reals: List<Real> := integers
```

Likewise:

```text
Pair<String, Int> -> Pair<String, Real>
```

is not an implicit conversion.

Class inheritance assignment is allowed.

---

## 15. Functions

Function declaration syntax:

```ahd
square: Function := (
    x: Real
) -> Real {
    return x^2
}
```

An ordinary Function declaration may appear only at module root. A method Function declaration may appear in Class member scope. Executable blocks may not contain new Function declarations.

An existing named Function value may still be stored in a Local Function binding:

```ahd
operation: Local Function := add
```

v0.1 has no nested Function declaration and no lambda/anonymous Function syntax.

### 15.1 Return behavior

A function returns one value at most.

There is no multiple return syntax.

A `Nothing` function may use a bare `return` or simply reach the end.

A non-`Nothing` function must return a compatible value or typed `null` on every reachable path.

### 15.2 Default parameters

Allowed:

```ahd
greet: Function := (
    name: String
    title: String := "Student"
) -> String {
    return "Hello {title} {name}"
}
```

A required positional parameter may not follow a default parameter.

### 15.3 Positional vs named arguments

A call is either entirely positional or entirely named.

They may not be mixed.

Named argument order is irrelevant.

A named argument uses `name: expression` syntax:

```ahd
createUser(
    name: "Ali"
    age: 25
)
```

Named arguments on the same line use commas, following the general separator rule:

```ahd
createUser(name: "Ali", age: 25)
```

Invalid because positional and named arguments are mixed:

```ahd
createUser("Ali", age: 25)
```

### 15.4 First-class named functions

Named functions may be stored in variables and passed to other functions.

No lambda/anonymous function syntax exists in v0.1.

A parameter or binding may be typed simply as `Function`; the programmer does not write a public function-signature type such as `Function<Int -> Int>`.

However, `Function` is **not** a dynamic callable escape hatch.

Every `Function` binding must resolve at compile time to exactly one concrete callable signature known to the semantic checker.

Example:

```ahd
calculate: Function := (
    operation: Function
    a: Int
    b: Int
) -> Int {
    return operation(a, b)
}
```

From the use of `operation(a, b)` and the surrounding return type, the compiler must infer a signature compatible with:

```text
(Int, Int) -> Int
```

This inferred signature is an internal compiler type. The user does not need to write it.

If multiple callable signatures remain equally valid, compilation fails with an ambiguity error.

If there is not enough information to infer one safe signature, compilation fails with a function-inference error.

The compiler must never silently treat `Function` as dynamically callable or defer signature correctness to runtime.

---

## 16. Function Overloading

Base:

```ahd
calculate: Function := (
    x: Int
) -> Int {
    return x^2
}
```

Overload:

```ahd
calculate: Overload Function := (
    x: Real
) -> Real {
    return x^2
}
```

Rules:

1. exact type match preferred;
2. safe widening such as `Int -> Real` may be used;
3. equal best candidates => ambiguous overload compile error;
4. return type alone cannot distinguish overloads.

---

## 17. Classes

A Class declaration may appear only at module root. Executable blocks and Class member scopes may not contain nested Class declarations.

Root class:

```ahd
Person: Class := {
}
```

Equivalent:

```ahd
Person: Class<> := {
}
```

Both derive from built-in `Object`.

Inheritance:

```ahd
Student: Class<Person> := {
}
```

One direct superclass in v0.1.

### 17.1 structure / Attributes

```ahd
structure: Attributes := (
    name: String
    age: Int
)
```

All non-`Local` entries automatically become instance attributes.

`Constant` and `Confidential` on a non-`Local` structure entry apply to the generated instance attribute; they are not lexical-scope modifiers. A Constant reference attribute deep-freezes its reachable object graph when the constructor initializes it. `Global` is not valid on a structure parameter.

Class methods use:

```ahd
attribute.name
```

### 17.2 Construction

A class is constructed through the same ordinary callable syntax used for functions. The class name is the callee, and the arguments are checked against its `structure: Attributes` parameters.

```ahd
student: Student := Student(
    name: "Ali"
    age: 20
)
```

Constructor calls may be entirely positional or entirely named. They follow the same separator and no-mixing rules as all other calls.

### 17.3 Local structure parameters

```ahd
User: Class<> := {
    structure: Attributes := (
        username: String
        password: Local String
    ) {
        attribute.passwordHash: Confidential String := hash(password)
    }
}
```

### 17.4 structure return

`return` is not allowed inside `structure`.

### 17.5 Inherited attributes

```ahd
Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )
}
```

### 17.6 Superclass methods

```ahd
SuperClass.describe()
```

calls the direct superclass implementation.

---

## 18. Override

Intentional replacement of an inherited function requires:

```ahd
describe: Override Function := (
) -> String {
    return "Student"
}
```

Plain inherited-signature collision is an error.

`Override Function` without a compatible inherited target is an error.

---

## 19. Confidential

Only one special visibility modifier exists in v0.1:

```text
Confidential
```

Default is public.

A Confidential class member:

- is accessible inside the defining class;
- is accessible to subclasses;
- is not accessible through ordinary external object access.

A module-level Confidential class/function is not public module API.

No separate `private`, `protected`, or `internal` keywords.

---

## 20. No Getter/Setter Syntax

AhdCode v0.1 has no dedicated Getter/Setter construct.

Use ordinary functions plus attributes, `Constant`, and `Confidential`.

---

## 21. Conditions

```ahd
if condition {
}
else if anotherCondition {
}
else {
}
```

Conditions must be Bool.

---

## 22. while

Pre-check loop.

```ahd
while count < 10 {
    count++
}
```

---

## 23. until

Post-check loop.

```ahd
until count == 10 {
    count++
}
```

Semantics:

1. execute body;
2. test condition;
3. if true stop;
4. otherwise repeat.

The body therefore executes at least once.

---

## 24. for and Iteration

Python-like iteration model.

List:

```ahd
for number in numbers {
    write(number)
}
```

String:

```ahd
for character in "AhdCode" {
    write(character)
}
```

Pair yields keys in insertion order.

### 24.1 Snapshot iteration

At loop start, AhdCode takes a shallow snapshot of the iterable's iteration view.

Structural mutation during the loop does not change what the active loop visits.

Referenced class objects remain shared references, not deep copies.

---

## 25. break and continue

Supported in `for`, `while`, and `until`.

They affect only the nearest enclosing loop.

No labels.

---

## 26. state / condition

```ahd
state status {
    condition "active" {
        write("Active")
    }

    condition "blocked" {
        write("Blocked")
    }

    condition default {
        write("Unknown")
    }
}
```

No fall-through.

No `break` required.

---

## 27. Error Handling

Keywords:

```text
attempt
except
ultimately
toss
```

Example:

```ahd
attempt {
    result: Local Real := divide(10, 0)
}
except DivisionByZeroError as error {
    write(error.message)
}
ultimately {
    write("Completed")
}
```

`toss` raises an Error.

AhdCode provides built-in `Error`.

Custom errors use normal inheritance:

```ahd
InvalidAgeError: Class<Error> := {
    structure: Attributes := (
        message: String
    )
}
```

`ultimately` is guaranteed to execute on success, handled error, propagating error, and before pending return completion.

If `ultimately` tosses a new error, the new error becomes active.

---

## 28. Operators

Arithmetic:

```text
+ - * / % ^
```

`^` is right-associative exponentiation.

Numeric result rules include:

| Left | Operator | Right | Result |
|---|---|---|---|
| `Int` | `/` | `Int` or `Real` | `Real` |
| `Real` | `/` | `Int` or `Real` | `Real` |
| `Int` | `%` | `Int` | `Int` |
| `Int` | `^` | `Int` | `Int` |
| `Real` | `^` | `Int` | `Real` |
| `Int` | `^` | `Real` | `Real` |
| `Real` | `^` | `Real` | `Real` |

No other `%` operand combination is valid. `/` always produces `Real`, including `Int / Int`.

`Int % Int` uses truncated-division, dividend-signed remainder semantics. The quotient is truncated toward zero, not floored. Consequently `-5 % 2` is `-1`, `5 % -2` is `1`, and `-5 % -2` is `-1`. For a nonzero divisor it is consistent with `a = trunc(a / b) * b + (a % b)`. This is not Python floor-mod semantics. A zero divisor raises the catchable `DivisionByZeroError`.

Power result types depend only on operand types; Constant status, compile-time evaluation, exponent sign, and optional optimizer/range analysis do not affect the result type. `Int ^ Int` uses checked Int arithmetic. A negative Int exponent raises `DomainError` during evaluation, and a result outside the signed 64-bit Int range raises `OverflowError`. It is not converted to `Real`.

`Int ^= Int` is valid for every Int right operand. A negative exponent raises `DomainError` and overflow raises `OverflowError` during evaluation. `Int ^= Real` is invalid because the operation produces `Real`, which cannot be assigned implicitly to an Int target. `Real ^= Int` and `Real ^= Real` are valid.

Other built-in meanings include:

- numeric arithmetic with safe `Int -> Real` where the operator permits it;
- `String + String`;
- `List<T> + List<T>` for compatible identical element type;
- `String * Int`.

No implicit String-number coercion.

No user-defined operator overloading in v0.1.

---

## 29. Equality and Type Operators

### == / !=

- scalar: value equality;
- `5 == 5.0` is true through numeric compatibility;
- List/Pair: deep value equality;
- Class: object/reference identity.

### same

Strict type + value/state.

```ahd
5 same 5       // true
5 same 5.0     // false
```

For class: exact runtime type + same instance.

For `List` and `Pair`, `==`/`!=` perform deep value comparison, while `same` performs object/reference identity comparison. Two distinct collections with equal contents compare equal with `==` but false with `same`; aliases of the same collection compare true with `same`.

### is / is not

Type membership, including inheritance.

### in / not in

- List => value membership;
- String => substring membership;
- Pair => key membership.

### has / has not

Class/object member existence only.

Not used for Pair.

---

## 30. Boolean Operators and Short-Circuit

```text
and
or
not
```

Bool only.

`and` and `or` short-circuit.

`not x == 5` means:

```ahd
not (x == 5)
```

---

## 31. Operator Precedence

High to low conceptually:

1. call/group/index/member
2. `^` right-associative
3. unary numeric signs
4. `* / %`
5. `+ -`
6. `< <= > >=`
7. `== != same is/is not in/not in has/has not`
8. `not` over the resulting Boolean comparison/expression
9. `and`
10. `or`

`++`/`--` are standalone statements.

---

## 32. between

Fundamentals provides `between` with Python-like range semantics.

```ahd
between(5)
```

=> `0 1 2 3 4`

```ahd
between(0, 5)
```

=> `0 1 2 3 4`

```ahd
between(0, 10, 2)
```

=> `0 2 4 6 8`

Stop is excluded. Zero step is an error.

---

## 33. bring

AhdCode uses `bring`, not `import`.

### 33.1 v0.1 module-name resolution

A v0.1 module reference is exactly the single case-sensitive identifier already accepted by the `bring` grammar. For a local module, `ModuleName` resolves to the file `ModuleName.ahd` in the directory containing the importing source file. Identifier normalization follows the ordinary AhdCode NFKC identifier rule before resolution.

v0.1 has no dotted module paths, relative-path syntax, package-root search, configurable source search path, or implicit directory-module convention. Those features require a later explicit language revision rather than an implementation-specific search heuristic.

Compiler-registered built-in module names resolve through the built-in module registry. A local file cannot shadow a registered built-in module name. Module-name and filename matching remains case-sensitive.

```ahd
bring Mathematics
```

This imports the module as a namespace. Its public members are accessed through that namespace:

```ahd
result: Real := Mathematics.sqrt(25)
```

```ahd
from Utilities bring all
```

```ahd
from Mathematics bring (
    sqrt
    sin
    cos
)
```

`from Mathematics bring sqrt` imports the selected public symbol directly, so it is called as `sqrt(...)` rather than `Mathematics.sqrt(...)`. The multiline form above applies the same rule to every listed symbol. `bring all` imports every non-`Confidential` public symbol directly.

Circular brings are compile-time errors.

Name collisions introduced by bring are compile-time errors.

Confidential module-level symbols are not externally bringable.

---

## 34. Fundamentals

The functions listed as available below are predeclared in every module and do not require `bring`.

Core terminal I/O:

```text
write
take
```

Available Fundamentals functions:

```text
str
int
real
len
clear
```

Planned early Fundamentals functions include:

```text
bool
max
min
sum
abs
round
between
swap
combine
merge
jump
copy
deepCopy
```

### 34.1 Canonical str semantics

`str` is locale-independent and deterministic. v0.1 has no user-defined or custom `str` override mechanism.

Scalar and null conversion:

| Input | Result |
|---|---|
| `String` | the String value itself |
| `Int` | base-10 decimal text |
| `Real` | shortest round-trip decimal text; an integral Real retains `.0` |
| `Bool` | `"true"` or `"false"` |
| `null` | `"null"` |

The Real decimal separator is always `.`. Canonical scientific notation may be used when needed and uses lowercase `e`. Negative zero is preserved.

Examples:

```text
str(5)     -> "5"
str(5.0)   -> "5.0"
str(-0.0)  -> "-0.0"
str(true)  -> "true"
str(null)  -> "null"
```

Collections use a canonical literal-like representation:

```text
str([1, 2, 3])
-> "[1, 2, 3]"

str(["Ali", "Ayşe"])
-> "[\"Ali\", \"Ayşe\"]"

str({"Ali": 90, "Ayşe": 95})
-> "{\"Ali\": 90, \"Ayşe\": 95}"
```

List order and Pair insertion order are preserved. Nested values use this same representation recursively. A String nested inside a collection is surrounded with double quotes and escaped with the exact AhdCode string escape rules.

The default Class instance representation is `<ClassName>`. For example, a Student instance becomes:

```text
<Student>
```

Attributes are not printed automatically. This prevents Confidential member disclosure and avoids recursive object-graph traversal.

A named Function value is represented as:

```text
<Function functionName>
```

`Nothing` is not accepted by `str`.

### 34.2 len

`len` reports the size of a sized value.

```text
len(String)     -> Int
len(List<T>)    -> Int
len(Pair<K,V>)  -> Int
```

`len(String)` counts characters, not bytes. `len(List<T>)` counts elements and `len(Pair<K,V>)` counts entries.

`len` does not accept scalar numeric types, `Bool`, `Class` instances, `Function` values, or `Nothing`. A nullable value must be `NonNull` before `len` is applied.

```ahd
write(len("añb"))
```

=> `3`

### 34.3 clear

`clear` empties a collection in place.

```text
clear(List<T>)    -> Nothing
clear(Pair<K,V>)  -> Nothing
```

`clear` does not create a new object. Object identity is unchanged, so every alias of the collection observes the emptied state.

```ahd
a: List<Int> := [1, 2, 3]
b: List<Int> := a

clear(a)

write(len(b))
```

=> `0`

The same reference semantics apply to `Pair`:

```ahd
scores: Pair<String, Int> := {
    "Ali": 85
}

alias: Pair<String, Int> := scores

clear(scores)

write(len(alias))
```

=> `0`

`clear` does not accept `String`, scalar types, `Class` instances, `Function` values, `null`, or `Nothing`. Strings are immutable, so an empty String is produced by rebinding rather than by `clear`.

A nullable `List` or `Pair` must be `NonNull` before `clear` is applied.

Clearing a directly known `Constant` target is a compile-time mutation error, because `clear` mutates the collection rather than rebinding it:

```ahd
values: Constant List<Int> := [1, 2, 3]
clear(values)
```

is invalid.

`clear` returns `Nothing`, so its result cannot be bound or used as a value.

Planned later Fundamentals/data-structure features may include:

```text
FSet
FLinkedList
FStack
FQueue
FDeque
Matrix
Complex
```

### swap

No tuple assignment.

Use:

```ahd
swap(a, b)
```

### combine

Homogeneous key List + homogeneous value List => Pair.

Length mismatch is an error.

### jump

Core slicing has no step syntax. Stepped selection belongs in a library function such as `jump`.

---

## 35. Core Terminal I/O

Input:

```ahd
name: String := take("Name: ")
```

Output:

```ahd
write("Hello {name}")
```

---

## 36. Runtime Numerical Safety

AhdCode prefers explicit errors over surprising low-level numeric behavior.

- division by zero => error;
- Int64 overflow => error, no silent wrap;
- Real overflow => error in ordinary AhdCode operations;
- real-domain invalid operations should prefer a domain error over silently exposing NaN where practical.

Complex mathematics belongs to a Complex facility later.

---

## 37. Unsupported v0.1 Features

Intentionally excluded:

- web runtime / AhdWeb
- HTTP routing
- MySQL
- SMTP
- HTML layouts
- static class members
- Getter/Setter syntax
- lambdas
- user-defined operator overloads
- multiple return values
- tuple/multiple assignment
- chained assignment
- goto/labels
- labeled break/continue
- slice-step syntax
- Char type
- JS-style implicit coercion
- `===`
- traits/interfaces/mixins
- decorators/annotations
- multiple inheritance

---

## 38. Planned Compiler Pipeline

```text
AhdCode source (.ahd)
        ↓
Lexer
        ↓
Parser
        ↓
AST
        ↓
Semantic/type analysis
        ↓
Typed/lowered IR
        ↓
Go code generation
        ↓
Go compiler
        ↓
native executable
```

The IR is an internal compiler layer, not public AhdCode syntax. It may make checked arithmetic, runtime null safety, deep freeze, snapshot iteration, class identity, and other AhdCode semantics explicit before Go generation.

The implementation itself may be written in Go.

AhdCode must not be a thin Python/JavaScript eval wrapper or regex-only translator.

---

## 39. CLI and interactive toolchain

```bash
ahdcode
```

REPL.

The REPL uses the same lexer, parser, semantic checker, lowering, backend, and runtime behavior as file compilation. It is not a separate mini-language. Session declarations persist, and ordinary same-scope declaration rules remain in force: entering `x: Int := 5` and later `x: Int := 7` is a duplicate declaration error. Reassignment is written `x = 7`. A failed semantic check or catchable runtime Error does not discard the last successfully committed session state or terminate the REPL.

```bash
ahdcode run hello.ahd
```

Run file.

```bash
ahdcode build hello.ahd
```

Build native executable through Go backend.

```bash
ahdcode format hello.ahd
```

Canonical in-place formatter. `ahdcode format --check hello.ahd` performs the same validation and canonicalization comparison without modifying the file; it succeeds only when the source is already canonical.

`ahdcode --help` describes the supported commands and `ahdcode --version` prints the canonical v0.1 version string. Unknown commands and flags fail without invoking a shell or treating arguments as source text.

---

## 40. Example Program

```ahd
PI: Constant Real := 3.14159

square: Function := (
    x: Real
) -> Real {
    return x^2
}

radiusInput: Int := 5
radius: Real := real(radiusInput)

if radius > 0 {
    area: Local Real := PI * square(radius)
    write("Area: {area}")
}
else {
    write("Radius must be positive")
}
```

---

## 41. Example Class

```ahd
Person: Class<> := {
    structure: Attributes := (
        name: String
        age: Int
    )

    describe: Function := (
    ) -> String {
        return "{attribute.name} - {attribute.age}"
    }
}

Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Constant Int
        password: Local String
    ) {
        attribute.passwordHash: Confidential String := hash(password)
    }

    describe: Override Function := (
    ) -> String {
        base: Local String := SuperClass.describe()
        return "{base} - {attribute.number}"
    }
}
```

---

## 42. Implementation Rule: Do Not Invent Semantics Silently

When the specification is ambiguous:

1. do not silently copy Python, Go, C, Java, or JavaScript;
2. do not add public syntax merely because it is easy to implement;
3. isolate the ambiguity;
4. document it;
5. request a language-design decision before freezing it.

In particular, do not resolve uncertainty by introducing hidden `Any`, dynamic Function dispatch, or unchecked nullable behavior.

---

## 43. v0.1 Definition of Done

Core v0.1 is meaningfully alive when:

1. Lexer handles Unicode identifiers, numbers, strings, triple strings, escapes, comments, keywords, operators, newline/comma separation.
2. Parser builds a real AST for core declarations, expressions, Functions, Classes, control flow, collections, bring, and error handling.
3. Semantic checker enforces types, flow-sensitive null-state rules, Function signature inference, scope, overload, inheritance, Constant, and visibility.
4. `write` and `take` work.
5. Functions, classes, Lists, Pairs, loops, and error handling execute correctly.
6. Go code generation builds representative `.ahd` programs.
7. REPL works for ordinary core code.
8. Formatter is deterministic/idempotent.
9. Tests cover malformed syntax and adversarial semantic cases.
10. No web functionality is required for core completion.

---

# End of AhdCode v0.1 Core Specification
