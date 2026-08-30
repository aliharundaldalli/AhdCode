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
List Pair Function lambda Overload Override
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

Absence is written explicitly in source types with `?`. Plain `T` is strictly
non-nullable; `T?` is the nullable form of the same underlying type.

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

### 6.3 String operations

`String` is immutable. Every operation below returns a new value and never modifies its receiver, so the receiver of one may be the result of another. The receiver and every argument must be `NonNull`.

```text
trim()                            -> String
lower()                           -> String
upper()                           -> String
capitalize()                      -> String
split(separator: String)          -> List<String>
replace(old: String, new: String) -> String
contains(text: String)            -> Bool
startsWith(prefix: String)        -> Bool
endsWith(suffix: String)          -> Bool
count(text: String)               -> Int
index(text: String)               -> Int
```

These are typed operations on the `String` type, selected by the receiver's static type. They are not Fundamentals functions, they publish no parameter names, and a named argument is rejected. A `Class` may still declare its own methods with these names, because the receiver type decides.

`trim` removes Unicode whitespace from the beginning and the end only; interior whitespace is preserved.

```ahd
write("[{"  Ali  ".trim()}]")
write("[{"\t Ali Harun \n".trim()}]")
```

=>

```text
[Ali]
[Ali Harun]
```

`lower` and `upper` are deterministic, locale-independent Unicode simple case mappings. v0.1 has no locale configuration and no Turkish-locale special casing.

```ahd
write("AhdCode".lower())
write("AhdCode".upper())
```

=>

```text
ahdcode
AHDCODE
```

`capitalize` uppercases the first character and leaves the remainder exactly as written. It is not Python's capitalize: the rest of the text is never lowercased. An empty String stays empty.

```ahd
write("ali HARUN".capitalize())
write("aHD".capitalize())
```

=>

```text
Ali HARUN
AHD
```

Normalization is written explicitly:

```ahd
write("ali HARUN".lower().capitalize())
```

=>

```text
Ali harun
```

`split` divides on every non-overlapping occurrence of the separator and preserves empty fields. The separator must not be empty; an empty separator raises the catchable `DomainError`. v0.1 has no parameterless whitespace-splitting form.

```ahd
write("a,b,c".split(","))
write("a,,b,".split(","))
write("".split(","))
```

=>

```text
["a", "b", "c"]
["a", "", "b", ""]
[""]
```

`replace` rewrites every non-overlapping occurrence, left to right. The searched text must not be empty and raises `DomainError` when it is; the replacement may be empty.

```ahd
write("banana".replace("na", "X"))
write("abc".replace("b", ""))
```

=>

```text
baXX
ac
```

`contains`, `startsWith`, and `endsWith` follow ordinary String mathematics, so an empty search text matches:

```ahd
write("abc".contains(""))
write("abc".startsWith(""))
write("abc".endsWith(""))
```

=>

```text
true
true
true
```

`count` counts non-overlapping occurrences. AhdCode does not adopt the length-plus-one rule for an empty search text: an empty text raises `DomainError`.

```ahd
write("banana".count("a"))
write("banana".count("na"))
write("banana".count("x"))
```

=>

```text
3
2
0
```

`index` returns the first occurrence as an AhdCode **character** index, never a UTF-8 byte offset.

```ahd
write("banana".index("na"))
write("a✓b✓".index("✓"))
```

=>

```text
2
1
```

`index` has no sentinel result. A search text the receiver does not contain raises the catchable `DomainError`, and so does an empty search text.

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
name: String? := null
age: Int? := null
student: Student? := null
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

These states are compiler analysis states. They refine the public nullable type
syntax; they do not replace it.

If the compiler can prove that a value is null or may be null, unsafe operations must be rejected at compile time whenever possible.

Example:

```ahd
age: Int? := null
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
student: Student? := findStudent()

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

Nullability composes structurally:

```text
List<User?>   non-null List whose elements may be null
List<User>?   nullable List whose elements are non-null
List<User?>?  nullable List whose elements may be null
Pair<String, User?>
```

`null` may appear only where the corresponding `?` makes the surrounding type
explicit.

```ahd
names: List<String?> := [
    "Ali"
    null
    "Ayşe"
]
```

A `null` key in a Pair is not allowed. `T` is assignable to `T?`; `T?` is not
assignable to `T` unless flow analysis has proven the expression non-null.

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

When an initializer has one unambiguous complete static type, the annotation
may be omitted:

```ahd
age := 28              // Int
user := findStudent()  // Student? when that is the Function return type
```

This is static inference, not a dynamic variable. Later assignment must still
match the inferred type. Bare `value := null` is invalid because it supplies
no underlying type; write `value: User? := null`. Scope intent remains
explicit: a nested inferred declaration is `name: Local := "Ali"`, while a
bare nested `name := "Ali"` remains invalid under §9.

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

`List<T>` has exactly these two element insertion/removal operations:

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

### 12.5 Ordering

`List<T>` has three ordering operations, and all rewrite the receiver in place:

```text
sort()                     -> Nothing
sort(key: Function(T) -> K) -> Nothing
reverse()                  -> Nothing
shuffle()                  -> Nothing
```

Like `add` and `eject`, they mutate the existing object, so every alias observes the new order, they return `Nothing`, and a `Constant` or otherwise frozen List rejects them. They publish no parameter names.

`reverse` reverses the current order.

```ahd
values: List<Int> := [1, 2, 3]
alias: List<Int> := values

values.reverse()

write(alias)
```

=>

```text
[3, 2, 1]
```

`shuffle` performs an unbiased in-place Fisher–Yates permutation. It walks
from the final element toward the second element and, for each index `i`, swaps
that element with an index selected uniformly from the inclusive interval
`0..i`. It uses the exact shared deterministic Math random sequence described
in §35.3; `shuffle`, `Math.random`, and `Math.randomInt` therefore advance one
common state in call order. `shuffle` does not require `bring Math`, but
`Math.seed(...)` may be used to reset the shared sequence before it.

```ahd
bring Math

Math.seed(42)
values: List<Int> := [1, 2, 3, 4, 5]
values.shuffle()

write(values)
```

=>

```text
[2, 3, 1, 5, 4]
```

An empty or single-element List remains unchanged and consumes no random
generator output. `shuffle` rearranges elements without inspecting or copying
them, so nullable elements retain their ordinary List semantics. The receiver
must still be `NonNull`, and a `Constant` or otherwise deep-frozen List rejects
the mutation before random state is consumed.

The natural form of `sort` orders ascending and is stable. Its element type must be `Int`, `Real`, or `String`; any other element type — including `Bool`, a `Class`, a `Pair`, or a nested `List` — is a compile-time rejection rather than a silent conversion to text. A `null` element has no natural order and raises the catchable `NullError`, leaving the List unchanged.

```ahd
values: List<Int> := [8, 3, 12, 5]

values.sort()

write(values)
```

=>

```text
[3, 5, 8, 12]
```

The key form orders by the result of a Function of one element. The key type `K` must be exactly `Int`, `Real`, or `String`; a `Bool` key is rejected, as is a key Function that may return `null`. v0.1 has no comparator form and no descending parameter, because a descending order is written with a negated or reversed key.

```ahd
gradeOf: Function := (
    student: Student
) -> Int {
    return student.grade
}

students.sort(gradeOf)
```

The key form is stable and atomic. Every key is computed exactly once per element, left to right, before the receiver is rewritten, so a key Function that raises propagates its error and leaves the original order unchanged. The receiver expression is evaluated exactly once. A key Function that returns `null` at run time raises `NullError`.

### 12.6 Searching

```text
count(value: T) -> Int
index(value: T) -> Int
```

Both are pure reads: they never mutate, reorder, or copy the receiver, so a `NonNull` `Constant` List is a valid receiver. Both compare with the ordinary deep `==` semantics rather than the `same` object identity, and both require a `NonNull` argument.

```ahd
values: List<Int> := [5, 7, 5, 9]

write(values.count(5))
write(values.index(5))
```

=>

```text
2
0
```

`index` reports the first match and has no sentinel result: a value the List does not contain raises the catchable `DomainError` rather than returning `-1`.

### 12.7 map and filter

```text
map(transform: Function(T) -> U)   -> List<U>
filter(keep: Function(T) -> Bool)  -> List<T>
```

Both build a new mutable List and never modify the receiver, so a `Constant` List is a valid receiver. Both iterate a shallow snapshot taken when the operation starts, which is the same rule `for` follows, and invoke the callback left to right exactly once per snapshot element. A callback error propagates normally.

A callback is a compatible Function value: either an ordinary declared
Function or an expression lambda. Its parameter type must be exactly the
element type because `List` is invariant.

```ahd
double: Function := (
    x: Int
) -> Int {
    return x * 2
}

numbers: List<Int> := [1, 2, 3]
doubled: List<Int> := numbers.map(double)
squared: List<Int> := numbers.map(lambda (x: Int) -> x^2)
```

=>

```text
[2, 4, 6]
```

`map` may change the element type:

```ahd
describe: Function := (
    x: Int
) -> String {
    return "Sayi: {x}"
}

texts: List<String> := numbers.map(describe)
```

A mapped Function must return a value; a `Nothing` result is rejected.

`filter` requires a real `Bool` predicate. AhdCode has no truthiness, so a non-`Bool` result is a compile-time rejection and a `null` result at run time raises `NullError`.

```ahd
isEven: Function := (
    x: Int
) -> Bool {
    return x % 2 == 0
}

values: List<Int> := [1, 2, 3, 4]
evens: List<Int> := values.filter(isEven)
```

=>

```text
[2, 4]
```

A nullable List element (`List<T?>`) is passed to the callback as written rather than silently skipped. When the callback parameter is non-nullable, that element is rejected by the ordinary null-safety rules.

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

v0.1 has no nested Function declaration. Expression lambdas are specified in
§50 and do not change this declaration syntax.

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

### 15.5 Expression lambdas

`lambda (<typed parameters>) -> <expression>` creates an anonymous value of
the existing `Function` type. It has one expression body, whose static type
and null-state form the callable return contract. It is not a new type, and
normal named Function syntax remains unchanged. The complete v0.1.10 rules,
including the lexical-capture limitation, are in §50.

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

`between` yields `Int` values; see §32.

### 24.1 Iteration binding type

The iteration binding may carry an explicit type:

```ahd
for value: Int in values {
    write(value)
}
```

The annotation is optional. Both canonical forms are:

```text
for name in iterable
for name: Type in iterable
```

The binding is always implicitly `Local`, so no scope modifier is written; `for value: Local Int in ...` is invalid.

An explicit type is a compile-time constraint: it must be the type the iterable yields. Iterating a `List<String>` as `for value: Int` is a compile-time error, and no unrelated element type is silently converted.

### 24.2 Snapshot iteration

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

v0.1.8 adds a small, closed set of ten Class Protocol Methods (`CAdd`,
`CSubtract`, `CMultiply`, `CDivide`, `CRemainder`, `CPower`, and their
counterparts for `CEqual`/`CCompare`/`CNegate`/`CStr`) that let a Class define
these operators for its own instances. This is not general/unrestricted
operator overloading: only these ten exact reserved names carry protocol
meaning, and only when they occupy a Class method slot. See §47 for the full
specification.

---

## 29. Equality and Type Operators

### == / !=

- scalar: value equality;
- `5 == 5.0` is true through numeric compatibility;
- List/Pair: deep value equality;
- Class: object/reference identity, **unless** the Class provides the
  `CEqual` Class Protocol Method (§47), in which case `a == b` calls
  `a.CEqual(b)` and `a != b` is always its logical negation. A Class with no
  `CEqual` keeps the plain reference-identity rule above.

### same

Strict type + value/state. `same` is never affected by `CEqual`: it is always
the raw identity test, independent of any Class Protocol Method.

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

The left operand must be a non-null Class object according to the standard null-safety rules.

`has` inspects the object's **actual runtime Class and its inheritance chain**, not the static type of the expression it is written against. An instance held in a variable of a parent type therefore still reports the members its real Class declares.

```ahd
person: Person := Student(name: "Ali", number: 42)

write(person has name)     // inherited attribute
write(person has describe) // inherited method
write(person has number)   // attribute of the runtime Class
write(person has study)    // method of the runtime Class
write(person has nickname) // no such member
```

=>

```text
true
true
true
true
false
```

Attributes and methods are both members, an overridden method is a member of both Classes, and an ordinary parent instance never gains a subclass member. The right operand is an unquoted member designator: it names a member, is never evaluated as a binding, and executes nothing. The left expression is evaluated exactly once. `has not` is the exact logical negation of the same lookup.

`has` checks **member existence**, not access permission. A `Confidential` member counts as existing, so `object has secret` returns `true` if the member exists. This does **not** bypass access rules; `object.secret` remains restricted according to the normal `Confidential` contract.

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

`between` is available in v0.1. It is predeclared and needs no `bring`.

It has Python-like integer range semantics and accepts one to three `Int`
arguments:

```text
between(stop)
between(start, stop)
between(start, stop, step)
```

`start` defaults to `0` and `step` defaults to `1`. Stop is always excluded.

```ahd
for value in between(5) {
    write(value)
}
```

=> `0 1 2 3 4`

```ahd
for value in between(1, 5) {
    write(value)
}
```

=> `1 2 3 4`

```ahd
for value in between(0, 10, 2) {
    write(value)
}
```

=> `0 2 4 6 8`

A negative step counts down:

```ahd
for value in between(5, 0, -1) {
    write(value)
}
```

=> `5 4 3 2 1`

When the step cannot reach the stop, the iteration is empty. Both
`between(0, 5, -1)` and `between(5, 0, 1)` yield nothing.

A zero step makes no progress and raises a catchable `DomainError`. It is never
silently treated as a step of `1`.

### 32.1 Lazy iteration

`between` does not produce a `List`. It is a lazy iteration whose whole state is
its current value, its stop, and its step, so iterating any range uses constant
memory regardless of how many values it yields:

```ahd
for value in between(1, 10000000) {
    write(value)
}
```

allocates no collection. Each value is computed on demand, so `break` stops
immediately and `continue` advances to the next value without materializing the
remainder. Because there is no backing collection, the shallow snapshot rule of
§24.2 does not apply to it.

Iteration stops before any step that would leave the signed 64-bit `Int` range,
so a range near the `Int` boundaries terminates rather than wrapping.

The only public contract of a `between` value is that iterating it yields `Int`.
v0.1 defines no type syntax for it and no other operations on it: it cannot be
indexed, sliced, mutated, cleared, converted to a `List`, or rendered with
`str`.

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

### 33.2 Module aliases

A namespace import may bind the module under a different name:

```ahd
bring Time as T
bring Math as M
bring Engine as E
```

The alias must be a valid identifier and is case-sensitive. `bring Time as T`
binds `T` and does **not** bind `Time`; importing both names requires two
`bring` statements. An alias participates in the ordinary binding rules, so a
name already in use is the usual import collision diagnostic.

Aliases apply uniformly to compiler-supplied standard modules and to source
modules. The other forms are unchanged:

```ahd
bring Module
from Module bring name
from Module bring (
    first
    second
)
from Module bring all
```

v0.1 has no symbol alias (`from Time bring DateTime as DT`) and no
namespace-qualified type syntax. A type is still imported before it is named:

```ahd
bring Time as T
from Time bring DateTime

current: DateTime := T.now()
```

The canonical formatting is `bring Time as T`.


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
between
abs
sum
min
max
```

Planned early Fundamentals functions include:

```text
bool
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

### 34.4 abs

`abs` is the numeric magnitude.

```text
abs(Int)   -> Int
abs(Real)  -> Real
```

The result type is exactly the argument type, so `abs` introduces no numeric widening of its own.

```ahd
write(abs(5))
write(abs(-5))
write(abs(2.5))
write(abs(-2.5))
```

=>

```text
5
5
2.5
2.5
```

`abs` does not accept `String`, `Bool`, `List`, `Pair`, `Class` instances, `Function` values, `null`, or `Nothing`. There is no implicit `String` conversion: `abs(int("-5"))` is written explicitly. A nullable value must be `NonNull` before `abs` is applied.

The minimum `Int` has no `Int` magnitude, so it follows the ordinary checked-arithmetic contract rather than wrapping:

```ahd
write(abs(-9223372036854775808))
```

raises the catchable `OverflowError`.

`abs(Real)` preserves the finite-`Real` rules and produces `0.0` for `-0.0`.

### 34.5 sum, min, and max

`sum`, `min`, and `max` reduce a numeric `List`.

```text
sum(List<Int>)   -> Int
sum(List<Real>)  -> Real

min(List<Int>)   -> Int
min(List<Real>)  -> Real

max(List<Int>)   -> Int
max(List<Real>)  -> Real
```

v0.1 has no vararg form: each takes exactly one `List`.

```ahd
values: List<Int> := [8, 3, 12, 5]

write(sum(values))
write(min(values))
write(max(values))
```

=>

```text
28
3
12
```

```ahd
values: List<Real> := [3.5, -2.0, 8.25]

write(sum(values))
write(min(values))
write(max(values))
```

=>

```text
9.75
-2.0
8.25
```

`List` generic invariance is unchanged: `List<Int>` is not a `List<Real>`, and no element is converted. `List<Bool>`, `List<String>`, `Pair`, `String`, and scalar values are rejected. The `List` argument must be `NonNull`.

`sum` of an empty `List` is the additive identity of its element type:

```ahd
ints: List<Int> := []
reals: List<Real> := []

write(sum(ints))
write(sum(reals))
```

=>

```text
0
0.0
```

`min` and `max` have no such identity, so an empty `List` raises the catchable `DomainError`:

```text
DomainError: min requires a non-empty List
DomainError: max requires a non-empty List
```

A `null` element encountered during a reduction raises the catchable `NullError` rather than being treated as zero or skipped.

`Int` summation uses checked `Int` arithmetic, so an overflowing total raises `OverflowError`. `Real` summation follows the finite-`Real` rules and never silently produces a non-finite total.

The three reductions are pure reads. They do not mutate, reorder, or copy the argument, object identity is unchanged, and a `NonNull` `Constant` `List` is therefore a valid argument:

```ahd
values: Constant List<Int> := [4, 1, 9]

write(sum(values))
write(min(values))
write(max(values))
```

=>

```text
14
1
9
```

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

## 35. Math Standard Module

`Math` is a compiler-registered standard module. It is not predeclared like
Fundamentals and must be imported through the ordinary module syntax:

```ahd
bring Math

write(Math.sqrt(25.0))
write(Math.PI)
```

`from Math bring sqrt` and the existing selective or `bring all` forms work
through the same module-interface rules as source modules. The canonical module
identity is `builtin:Math`; a sibling `Math.ahd` cannot shadow it. Every Math
argument must be `NonNull`. A parameter written as `Real` accepts only `Real`
and the existing implicit `Int -> Real` widening—never `String`, `Bool`, or a
new coercion.

The exact v0.1 public surface is:

```text
PI: Constant Real
E:  Constant Real

round(Real)      -> Real
round(Real, Int) -> Real
floor(Real)      -> Int
ceil(Real)       -> Int

sqrt(Real)  -> Real
sin(Real)   -> Real
cos(Real)   -> Real
tan(Real)   -> Real
log(Real)   -> Real
log10(Real) -> Real
exp(Real)   -> Real

seed(Int)           -> Nothing
random()            -> Real
randomInt(Int, Int) -> Int
```

`PI` and `E` are immutable float64 mathematical constants. They are ordinary
readable namespace members but cannot be assigned or updated. `abs`, `sum`,
`min`, and `max` remain Fundamentals; `Math` does not alias them. There is no
`Math.pow`; exponentiation uses `^`.

### 35.1 Rounding and integral bounds

`round(value)` returns an integral `Real`. Exact halves round away from zero:

```text
Math.round(3.4)   -> 3.0
Math.round(3.5)   -> 4.0
Math.round(-3.5)  -> -4.0
```

The two-argument form rounds to `digits` decimal places. `digits` must be in
`0..15`; otherwise it raises the catchable `DomainError`. The deterministic
float64 procedure multiplies by `10^digits`, applies half-away-from-zero
rounding, then divides by the same factor. If scaling a finite value would
overflow, the value is returned unchanged because its float64 spacing already
exceeds the requested decimal precision.

```text
Math.round(3.14159, 2) -> 3.14
Math.round(2.675, 2)   -> 2.68
```

`floor` and `ceil` return `Int` and follow their mathematical definitions for
positive and negative values. A result outside signed Int64 raises catchable
`OverflowError`; it never wraps.

### 35.2 Classic mathematics

`sqrt` returns the principal square root. A negative input raises
`DomainError`. `sin`, `cos`, and `tan` use radians; v0.1 has no degree mode.
`log` is the natural logarithm and `log10` is base ten. Both require a value
strictly greater than zero and otherwise raise `DomainError`. `exp` raises
`OverflowError` if its result exceeds the finite `Real` range.

Every Math result obeys the finite-`Real` contract. A runtime operation never
exposes `NaN` or infinity: undefined results become `DomainError`, and
out-of-range finite mathematics becomes `OverflowError`.

### 35.3 Pseudo-random sequence and explicit seeding

There is one shared Math pseudo-random sequence per native program execution.
At each fresh execution AhdCode reads exactly eight bytes from the operating
system's cryptographically secure entropy source and interprets them as one
little-endian unsigned 64-bit initial state. Failure to obtain those bytes is a
startup failure; the runtime never silently substitutes a fixed seed, time,
process identity, or clock value. Two unseeded executions are not required to
match and are not required to differ, because an entropy collision remains
theoretically possible. The initial seed is not exposed through a public API.

`Math.seed(value)` explicitly resets the shared state. Every signed Int64 seed
is valid and maps to the internal `uint64` state by its two's-complement bit
pattern. Reseeding with the same value reproduces the exact same sequence, so
explicit seeding is the supported mechanism for reproducible tests and
simulations. Calls made while initializing different modules and in the entry
module advance the same sequence in actual runtime order.

v0.1 pins SplitMix64. For state `s`, one output performs the following unsigned
64-bit wrapping operations exactly:

```text
s = s + 0x9e3779b97f4a7c15
z = s
z = (z xor (z >> 30)) * 0xbf58476d1ce4e5b9
z = (z xor (z >> 27)) * 0x94d049bb133111eb
z = z xor (z >> 31)
```

This algorithm and every explicitly seeded sequence are part of the v0.1
reproducibility contract across Go versions, operating systems, and supported
architectures. OS entropy initializes the state only; `Math.random`,
`Math.randomInt`, and `List.shuffle` remain pseudo-random and are not
cryptographically secure.

`random()` advances once and constructs a `Real` in `0.0 <= result < 1.0` from
the high 53 output bits:

```text
float64(z >> 11) * 2^-53
```

The first values for seed `557` are pinned as:

```text
0.4121990632081577
0.4686510900868295
0.5840201876345011
```

`randomInt(min, max)` uses inclusive bounds. `min > max` raises `DomainError`.
When `min == max`, it returns that value without consuming generator state.
For every other interval it uses rejection sampling before reducing modulo the
unsigned interval width, so the result has no modulo bias. Span arithmetic is
unsigned and supports intervals crossing zero and both Int64 boundaries. A
width of zero denotes the complete `2^64`-value Int64 domain and maps one raw
output across that full ordered interval.

---

## 36. Time Standard Module

`Time` is a compiler-registered standard module, imported exactly like `Math`
through the ordinary module syntax:

```ahd
bring Time
from Time bring DateTime
from Time bring Duration
```

The canonical module identity is `builtin:Time`; a sibling `Time.ahd` cannot
shadow it. AhdCode has no namespace-qualified type syntax, so a Time type is
written as `DateTime` after importing it, never as `Time.DateTime`. Every Time
argument must be `NonNull`.

The exact public surface is:

```text
Time.now()                        -> DateTime
Time.utc()                        -> DateTime
Time.timestamp()                  -> Int
Time.fromTimestamp(milliseconds: Int) -> DateTime
Time.monotonic()                  -> Real
Time.sleep(milliseconds: Int)     -> Nothing
Time.duration(milliseconds: Int)  -> Duration
Time.between(first: DateTime, second: DateTime) -> Duration
Time.dateTime(
    year: Int,
    month: Int,
    day: Int,
    hour: Int = 0,
    minute: Int = 0,
    second: Int = 0,
    millisecond: Int = 0
) -> DateTime
Time.dateTimeUTC(year, month, day, hour = 0, minute = 0, second = 0, millisecond = 0) -> DateTime
Time.dateTimeOffset(year, month, day, offsetMinutes, hour = 0, minute = 0, second = 0, millisecond = 0) -> DateTime
```

### 36.1 Local, UTC, fixed offset, and timestamps

`Time.now()` reports the host's **local** date and time, and `Time.dateTime`
builds a local civil moment. `Time.utc()` and `Time.dateTimeUTC` use UTC.
`Time.dateTimeOffset` uses a fixed whole-minute offset from -840 through 840.
A Unix timestamp is signed milliseconds since `1970-01-01 00:00:00 UTC`;
`Time.timestamp()` reads the current value and `Time.fromTimestamp` returns
its UTC representation. Negative timestamps are valid when representable as a
DateTime year 1..9999. v0.1.11 has no named/IANA timezone database.
The offset representation has minute precision. If a host's historical local
zone reports a seconds component, local construction/conversion raises
`ValueError` rather than truncating the offset and changing the instant.

### 36.2 DateTime

`DateTime` exposes nine read-only `Int` attributes:

```text
year  month  day  hour  minute  second  millisecond  weekday  offsetMinutes
```

`weekday` numbers the days from Monday:

| day | value |
|---|---|
| Monday | 1 |
| Tuesday | 2 |
| Wednesday | 3 |
| Thursday | 4 |
| Friday | 5 |
| Saturday | 6 |
| Sunday | 7 |

Every attribute is `Constant`, so assigning one is the ordinary Constant
diagnostic:

```ahd
current: DateTime := Time.now()

current.year = 2030
```

is invalid.

`offsetMinutes` is the value's fixed offset east of UTC. `DateTime` publishes
eight members:

```text
before(other: DateTime)     -> Bool
after(other: DateTime)      -> Bool
sameMoment(other: DateTime) -> Bool
timestamp()                 -> Int
toUTC()                     -> DateTime
toLocal()                   -> DateTime
toOffset(offsetMinutes: Int) -> DateTime
toString()                  -> String
```

`toString` is deterministic and locale-independent, and never names a timezone:

```text
YYYY-MM-DD HH:MM:SS
```

Milliseconds are read through the `millisecond` attribute rather than through
the text. `str(value)` renders a `DateTime` as `<DateTime>`, because §34.1
deliberately does not print Class attributes.

`DateTime` does not implement `CCompare` (§47), so `<` and `>` do not apply to
it. Ordering is written with `before` and `after`. `DateTime` also does not
implement `CEqual`, so `==` and `same` follow the ordinary Class rule of §29 —
object identity — and two separately built equal moments are **not** `==`.
Value comparison is `sameMoment`, and two `Duration` values are compared
through `milliseconds`.

`timestamp`, `toUTC`, `toLocal`, and `toOffset` preserve the represented
instant. `before`, `after`, `sameMoment`, and `Time.between` compare instants,
not displayed civil fields, even when the offsets differ.

### 36.3 Creating a DateTime

`Time.dateTime`, `Time.dateTimeUTC`, and `Time.dateTimeOffset` validate every component against the Gregorian calendar and
raises `ValueError` for an impossible moment rather than rolling it over:

```text
year         1..9999
month        1..12
day          1..(length of that month in that year)
hour         0..23
minute       0..59
second       0..59
millisecond  0..999
offsetMinutes -840..840 (fixed-offset constructor and conversion)
```

```ahd
Time.dateTime(year: 2028, month: 2, day: 29)
```

is valid, while `2026-02-29`, `2026-02-30`, month `13`, and hour `25` are all
`ValueError`. `DateTime` and `Duration` are never constructed directly; they
come only from the Time functions, which validate first.

### 36.4 Duration

`Duration` is elapsed time, not a calendar date. It exposes two read-only
attributes:

```text
milliseconds Int
seconds      Real
```

```ahd
wait: Duration := Time.duration(milliseconds: 1500)

write(wait.milliseconds)
write(wait.seconds)
```

=>

```text
1500
1.5
```

A `Duration` may be negative, because a signed difference is useful. A negative
value is preserved rather than being turned into its magnitude.

### 36.5 Difference between two moments

`Time.between(first, second)` is `second - first`:

```ahd
a: DateTime := Time.dateTime(year: 2026, month: 1, day: 1)
b: DateTime := Time.dateTime(year: 2026, month: 1, day: 2)

write(Time.between(a, b).milliseconds)
write(Time.between(b, a).milliseconds)
```

=>

```text
86400000
-86400000
```

The same moment on both sides produces a zero `Duration`.

### 36.6 Calendar

`Time.Calendar` answers questions about the Gregorian calendar itself, without
needing a `DateTime`:

```text
Time.Calendar.isLeapYear(year: Int)                     -> Bool
Time.Calendar.daysInMonth(year: Int, month: Int)        -> Int
Time.Calendar.weekday(year: Int, month: Int, day: Int)  -> Int
```

A leap year is divisible by 4, except a century year, which must be divisible
by 400: `2028` and `2000` are leap years, `2100` and `1900` are not.
`weekday` uses the same Monday=1..Sunday=7 numbering as the `weekday`
attribute. An invalid year, month, or date raises `ValueError`.

`Calendar` is read-only and is not constructed. v0.1 has no month or day names,
no localization, and no calendar rendering.

### 36.7 Elapsed time and waiting

```ahd
start: Real := Time.monotonic()

Time.sleep(100)

elapsed: Real := Time.monotonic() - start
```

`Time.monotonic()` is a **seconds** reading on a clock that never moves
backwards. Only differences are meaningful; the absolute value has no calendar
meaning.

`Time.sleep` takes **milliseconds**. Zero returns immediately, and a negative
request raises `ValueError` rather than being clamped to zero.

### 36.8 Not in this version

`Time` deliberately has no named/IANA zones, DST configuration objects,
format strings, `parse`, ISO-8601 or RFC 3339 reader, month or day names, or
natural-language dates. Existing `toString()` remains `YYYY-MM-DD HH:MM:SS`
and does not append an offset.

---

## 37. Latex Standard Module

`Latex` is a compiler-registered standard module, imported like `Math` and
`Time`, including with an alias:

```ahd
bring Latex as L
from Latex bring LatexError
```

The canonical identity is `builtin:Latex`. Every argument must be `NonNull`.

The exact public surface is:

```text
Latex.pdf(source: String, output: String)     -> Nothing
Latex.pdfFile(input: String, output: String)  -> Nothing
Latex.escape(text: String)                    -> String
Latex.section(title: String)                  -> String
Latex.subsection(title: String)               -> String
Latex.equation(source: String)                -> String
Latex.document(body: String, title: String = "", author: String = "") -> String
Latex.table(headers: List<String>, rows: List<List<String>>) -> String

LatexError
```

### 37.1 Text helpers

`escape` is text-context escaping for the TeX-special characters
`\ { } $ & # % _ ^ ~`. It does not claim to sanitize raw mathematics.

`section` and `subsection` escape their titles. `equation` deliberately does
not escape, because it accepts raw LaTeX math source. `table` produces
`booktabs` source and escapes every cell; a row whose column count differs from
the headers raises `ValueError`.

`document` returns a complete document whose preamble names the bundled Latin
Modern font files explicitly, so rendering never depends on a host system font.

### 37.2 Compilation

`pdf` compiles a source String and `pdfFile` compiles an existing `.tex` file,
resolving document-relative assets such as `\includegraphics` against the input
file's directory.

Compilation uses a Tectonic engine and a local resource bundle that ship with an
AhdCode installation. AhdCode never runs a `tectonic` found on `PATH`, never
falls back to a system TeX installation, and never downloads a resource at run
time. A supported document therefore compiles on a fresh machine with an empty
cache and no network. A missing bundled engine or bundle is a `LatexError`.

### 37.3 Security and limits

The engine runs in untrusted mode, so shell escape is unavailable and no AhdCode
construct can enable it. The engine is launched with an argument vector rather
than a shell command string, so paths containing spaces, Unicode, quotes, `$`,
`;`, `&`, or parentheses remain safe. Compilation is bounded by a 30-second
timeout; on timeout the process is terminated, temporary files are removed, and
`LatexError` is raised.

### 37.4 Output safety

Source compiles in a unique secure temporary directory that is removed on both
success and failure. The PDF is produced to a temporary location and checked for
existence, regular-file status, non-zero size, and the `%PDF-` signature before
it replaces the requested destination, so a failed compile never destroys an
already valid destination PDF.

### 37.5 LatexError

`LatexError` covers compilation failure, a missing bundled engine or bundle,
timeout, engine process failure, and a PDF that was not produced. Engine
diagnostics are bounded so a malformed document cannot flood the terminal, while
the first useful TeX error is preserved.

### 37.6 Not in this version

No BibTeX management, package manager, TikZ or Beamer abstraction, PDF editor or
parser, and no Markdown or HTML conversion.

---

### 37.7 Path and File Standard Modules

`Path` and `File` are compiler-registered standard modules imported through
the ordinary module system. They are not predeclared Fundamentals.

```ahd
bring Path
bring File
from File bring FileError
```

Their exact v0.1 public surfaces are:

```text
Path.join(parts: List<String>) -> String
Path.ext(path: String)         -> String
Path.base(path: String)        -> String
Path.dir(path: String)         -> String

File.exists(path: String)                  -> Bool
File.readText(path: String)                -> String
File.writeText(path: String, content: String) -> Nothing
File.append(path: String, content: String) -> Nothing
File.delete(path: String)                  -> Nothing
File.createDir(path: String)               -> Nothing
File.list(path: String)                    -> List<String>

FileError
```

Path operations use the host operating system's path rules and perform no
filesystem access. File text is UTF-8; reading invalid UTF-8 raises
`FileError`. `File.list` returns only the immediate entry names in stable
ascending lexical order and never recurses. Relative paths use the executing
process's current working directory, including the directory from which a REPL
session was launched.

Ordinary operating-system failures are catchable AhdCode errors. `FileError`
inherits `IOError`, which inherits `Error`; a missing path passed to
`File.exists` yields `false`, while failures of the other operations raise
`FileError`. No raw host error value or Go panic is exposed.

---

## 38. Core Terminal I/O

### 38.1 take

`take` is the terminal input function. It has exactly two forms:

```text
take()               -> String
take(prompt: String) -> String
```

Both read exactly one line from standard input and return it as a `String`.

```ahd
name: String := take()
```

```ahd
name: String := take("Name: ")
```

The prompt form writes the prompt first, without adding a newline of its own,
and the prompt is visible before the program blocks for input. The prompt is
never part of the returned text. The prompt argument must be a `NonNull`
`String`.

The returned `String` excludes the line terminator, in both the `LF` and `CRLF`
forms. Ordinary whitespace inside the entered text is preserved:

| stdin | result |
|---|---|
| `Ali\n` | `"Ali"` |
| `  Ali  \n` | `"  Ali  "` |
| `\n` | `""` |
| end of input | `""` |

`take` never parses or converts what it reads. AhdCode stays strictly typed, so
numeric input goes through the ordinary conversions:

```ahd
age: Int := int(take("Age: "))
value: Real := real(take())
```

This is invalid, because a `String` is not implicitly an `Int`:

```ahd
age: Int := take()
```

`take` is the only terminal input function in v0.1. There is no `takeInt`,
`takeReal`, `input`, or `readLine`.

### 38.2 write

```ahd
write("Hello {name}")
```

---

## 39. Runtime Numerical Safety

AhdCode prefers explicit errors over surprising low-level numeric behavior.

- division by zero => error;
- Int64 overflow => error, no silent wrap;
- Real overflow => error in ordinary AhdCode operations;
- real-domain invalid operations should prefer a domain error over silently exposing NaN where practical.

Complex mathematics belongs to a Complex facility later.

---

## 40. Unsupported v0.1 Features

Intentionally excluded:

- web runtime / AhdWeb
- HTTP routing
- MySQL
- SMTP
- HTML layouts
- static class members
- Getter/Setter syntax
- block/statement lambdas and lexical closures
- general/unrestricted user-defined operator overloading (only the ten fixed
  Class Protocol Methods of §47 exist; there is no arbitrary operator
  definition, no reverse operators, and no in-place protocols)
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
- `reduce` and other fold operations
- comparator or descending forms of `sort`
- List/String operation aliases such as `append`, `push`, `remove`, `findIndex`, `foreach`, `select`, `where`, and `transform`

---

## 41. Planned Compiler Pipeline

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

## 42. CLI and interactive toolchain

```bash
ahdcode
```

REPL.

The REPL uses the same lexer, parser, semantic checker, and typed/lowered IR as
file compilation. Validated new IR executes in one persistent evaluator; prior
statements are not executed again. Values, aliases, Functions, Classes,
imports, module initialization, Math RNG state, working directory, and terminal
streams persist. It is not a separate mini-language. Session declarations
persist, and ordinary same-scope declaration rules remain in force: entering
`x := 5` and later `x := 7` is a duplicate declaration error. Reassignment is
written `x = 7`. A failed semantic check or catchable runtime Error does not
discard the last successfully committed session state or terminate the REPL.

`take()` reads the same real terminal input stream as the REPL. Its prompt is
flushed before blocking, and the one answer line it consumes is not parsed as a
subsequent REPL command. A top-level expression whose type is not `Nothing` is
printed in canonical form.

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

## 43. Example Program

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

## 44. Example Class

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

## 45. Implementation Rule: Do Not Invent Semantics Silently

When the specification is ambiguous:

1. do not silently copy Python, Go, C, Java, or JavaScript;
2. do not add public syntax merely because it is easy to implement;
3. isolate the ambiguity;
4. document it;
5. request a language-design decision before freezing it.

In particular, do not resolve uncertainty by introducing hidden `Any`, dynamic Function dispatch, or unchecked nullable behavior.

---

## 46. v0.1 Definition of Done

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

## 47. Class Protocol Methods (v0.1.8)

A Class may define operator behavior for its own instances through a small,
closed set of exactly ten reserved method names, called **Class Protocol
Methods**:

```text
CEqual CCompare
CAdd CSubtract CMultiply CDivide CRemainder CPower
CNegate CStr
```

No others exist. This is a deliberately narrow compiler extension, not
general/unrestricted operator overloading and not Python's magic-method
system: there is no `__eq__`/`__lt__`/`__repr__`/`__radd__`-style
double-underscore convention, and no attempt to give every language mechanism
(construction, iteration, indexing, attribute access, calling) its own
protocol name.

### 47.1 Where the names are reserved

The ten names are compiler-special **only** when they occupy a Class method
slot. At module scope, `CAdd: Function := ...` remains an ordinary Function.
Inside a Class, a name that is not one of the ten (`Calculate`, `Create`,
`CWhatever`, `CustomMethod`, and so on) remains an ordinary member; the
letter `C` itself carries no meaning. A Class body member that reuses one of
the ten names but is not a Function — for example `CAdd: Int := 5` — is a
compile-time error (reserved Class Protocol Method slot), not a silently
accepted field.

### 47.2 Declaration syntax is unchanged

A Class Protocol Method is written with the ordinary method syntax; there is
no new declaration form, and Function/Class declaration syntax do not change:

```ahd
Vector2: Class<> := {
    structure: Attributes := (
        x: Real
        y: Real
    )

    CEqual: Function := (
        other: Vector2
    ) -> Bool {
        return attribute.x == other.x and attribute.y == other.y
    }

    CAdd: Function := (
        other: Vector2
    ) -> Vector2 {
        return Vector2(x: attribute.x + other.x, y: attribute.y + other.y)
    }

    CNegate: Function := (
    ) -> Vector2 {
        return Vector2(x: -attribute.x, y: -attribute.y)
    }

    CStr: Function := (
    ) -> String {
        return "Vector2({attribute.x}, {attribute.y})"
    }
}
```

### 47.3 Required signatures

| Protocol | Explicit parameters | Return type |
|---|---|---|
| `CEqual` | exactly 1 | `Bool` |
| `CCompare` | exactly 1 | `Int` |
| `CAdd`, `CSubtract`, `CMultiply`, `CDivide`, `CRemainder`, `CPower` | exactly 1 | the operator's result type; not required to equal the containing Class |
| `CNegate` | 0 | the operator's result type |
| `CStr` | 0 | `String` |

A malformed declaration (wrong arity, wrong return type, or a reserved name
occupying a non-Function slot) is a normal semantic diagnostic, never a
runtime panic. The arithmetic protocols and `CNegate` may legitimately return
a different type than the containing Class — for example a future
`Matrix * Vector -> Vector` — and the operator expression's static type is
simply the selected method's declared return type.

### 47.4 Operator mapping

| Operator | Protocol |
|---|---|
| `==` | `CEqual` |
| `!=` | logical negation of the same `CEqual` call; there is no `CNotEqual` |
| `<`, `<=`, `>`, `>=` | all four derive from one `CCompare` call, evaluated exactly once per expression |
| `+` | `CAdd` |
| `-` (binary) | `CSubtract` |
| `*` | `CMultiply` |
| `/` | `CDivide` |
| `%` | `CRemainder` |
| `^` | `CPower` |
| `-` (unary) | `CNegate` |

`CCompare`'s result uses the conventional sign interpretation and is **not**
restricted to `-1`/`0`/`1`:

```text
a <  b   =>  a.CCompare(b) <  0
a <= b   =>  a.CCompare(b) <= 0
a >  b   =>  a.CCompare(b) >  0
a >= b   =>  a.CCompare(b) >= 0
```

`CEqual` is never derived from `CCompare`, and `CCompare` is never derived
from `CEqual`: a type may be meaningfully equality-comparable without a
natural ordering, and vice versa. If a Class provides no `CEqual`, `==`/`!=`
keep the pre-v0.1.8 reference-equality rule of §29 unchanged; if it provides
no `CCompare`, `<`/`<=`/`>`/`>=` remain the ordinary static type error.

### 47.5 Dispatch is left-operand based

There is no reverse-operator protocol (`CReverseAdd`/`CRAdd`, etc.):

```ahd
value + 3   // works if value's Class declares CAdd(Int)
3 + value   // does NOT try value's CAdd; ordinary primitive-operator rules
            // apply, and this is a static type error unless independently valid
```

### 47.6 Overloading, inheritance, and dynamic dispatch

Class Protocol Methods reuse the ordinary method overload-resolution,
inheritance, and dynamic-dispatch machinery; there is no second system. A
Class may declare more than one `CAdd` overload under the existing
`Overload Function` rules (§26), operator resolution uses the same static
overload-resolution rules as an explicit call, and ambiguity is the ordinary
compile-time error. A protocol method is inherited and overridden exactly
like any other method (`Override` is required to replace one), and operator
dispatch uses the same dynamic dispatch as an ordinary method call: if a
statically valid protocol method is overridden by a subclass, the override
runs when the runtime object is that subclass, exactly as in §26.

### 47.7 Compound assignment

`+=`, `-=`, `*=`, `/=`, `%=`, and `^=` on a Class target reuse the matching
arithmetic protocol: `a += b` behaves like `a = a + b`, subject to the normal
assignment-compatibility rules, with the receiver evaluated exactly once
(including a member or indexed target). There is no separate in-place
protocol (no `CIAdd`-style method). `++` and `--` are unrelated and are not
extended to Class values.

### 47.8 Nullability

Protocol dispatch never weakens the null-safety rules of §§17-24. If the left
operand is nullable, it must be narrowed to `NonNull` by ordinary flow
analysis before a protocol method can be invoked on it — the same requirement
as any other method call:

```ahd
user: User?

user + other   // invalid unless user is narrowed to non-null
```

The right-hand argument uses its own declared parameter type and nullability
normally; a protocol may explicitly accept a nullable argument
(`CEqual(other: User?) -> Bool`), but nothing about that acceptance is
inferred automatically.

### 47.9 CStr and str()

For a Class instance whose statically declared type resolves `CStr`, the
Fundamental `str(value)` (§34) dispatches to it, and `write` benefits through
the same shared conversion path. There is no `repr()`/`CRepr`, and no second
developer/debug string protocol. A Class with no `CStr` keeps the existing
`<ClassName>` rendering of §34.1.

---

## 48. Runtime Introspection Fundamentals: `type()` and `id()` (v0.1.8)

### 48.1 `type(value) -> String`

Returns the canonical AhdCode type name of `value` as a `String`. It is a
small runtime/introspection aid, not a reflection framework: it never returns
a first-class type object, and there is no metaclass or reflection API.

```text
type(5)      -> "Int"
type(5.0)    -> "Real"
type("Ali")  -> "String"
type(true)   -> "Bool"
type(null)   -> "Null"
```

Collections use canonical AhdCode generic notation wherever the language has
sufficient static type information: `List<Int>`, `List<Int?>`,
`Pair<String, Int>`. `type` never exposes a backend/Go implementation type.

For a Class instance, `type` reports the **most-derived runtime Class**, not
the static/declared type of the expression:

```ahd
animal: Animal := Dog(name: "Rex")
write(type(animal))   // "Dog", not "Animal"
```

For a nullable value that currently holds a non-null value, `type` reports
the contained value's own type, never the declared type's trailing `?`. For
`null` itself, `type` reports the literal String `"Null"`; this is an
intrinsic Fundamental special case and does **not** introduce `Null` as an
ordinary source-level declaration type, and does not weaken the `x := null`
inference rejection of §17.

A Function value's `type()` text reuses the same canonical signature
formatting already used elsewhere in diagnostics (`Function(ParamType, ...)
-> ReturnType`), never a Go-level representation.

### 48.2 `id(reference) -> Int`

Returns an opaque, runtime-managed identity number for a **List, Pair, or
Class instance**. `Int`, `Real`, `String`, and `Bool` are not accepted --
`id(5)` and `id("x")` are compile-time errors, and no primitive is boxed
merely to manufacture an identity.

The returned number:

- is opaque and only meaningful within the current process or REPL session;
- is **not** a memory address, and never derived from one;
- is not guaranteed to be stable across separate program executions;
- is not serialization data or a persistent/database identifier;
- may happen to be produced by an incrementing allocator internally, but a
  program must never depend on allocation order, only on equality/inequality
  between two identities.

Within one process or REPL session, an alias shares its identity with the
original (`id(a) == id(b)` when `b := a`), and two distinct, simultaneously
existing objects have distinct identities. The identity is stable for an
object's entire lifetime and is unaffected by mutation (mutating a List,
Pair, or Class instance never changes its `id()`). `id()` requires a
non-null identity-bearing reference under the ordinary nullable-use rules of
§§17-24.

`id()` does not replace `same` (§29): `same` is the ordinary programmatic
identity test; `id()` exists for debugging, logging, and introspection. For
supported live reference values, `(a same b) == (id(a) == id(b))`.

---

## 49. Regex Standard Module (v0.1.9)

`Regex` is explicit, like `Math` and `Time` (§33, §35, §36): it must be
imported with `bring Regex` before use, and its canonical identity is
`builtin:Regex`, so a sibling `Regex.ahd` file cannot shadow it.

```ahd
bring Regex
from Regex bring Pattern
from Regex bring RegexError
```

### 49.1 Compiling a pattern

```text
Regex.compile(pattern: String) -> Pattern
```

`Regex.compile` compiles a pattern using Go `regexp` (RE2) syntax and returns
a `Pattern` instance. An invalid pattern raises the catchable `RegexError`,
which derives directly from `Error` (not from `IOError`). `Pattern` is a
compiler-supplied Class: it is never constructed directly, only produced by
`Regex.compile`, exactly like `Time.dateTime` and `DateTime` in §36.

The Class is named `Pattern`, not `Regex`, specifically so it can be named
independently of the module's own namespace (`bring Regex` already binds the
name `Regex`; `from Regex bring Pattern` is required to name the type).

### 49.2 Pattern members

```text
matches(text: String)                    -> Bool
find(text: String)                       -> String?
findAll(text: String)                    -> List<String>
groups(text: String)                     -> List<String>?
replace(text: String, replacement: String) -> String
split(text: String)                      -> List<String>
```

- `matches` reports whether the pattern is found anywhere in `text` (not an
  implicit full-string anchor; write `^...$` in the pattern for that).
- `find` returns the first match, or `null` if the pattern does not occur in
  `text`. The result is `String?`; ordinary null-safety narrowing (§§17-24)
  applies before use.
- `findAll` returns every non-overlapping match, in order; an empty
  `List<String>` if there is no match.
- `groups` returns the first match's full match text followed by its capture
  groups (index `0` is the whole match), or `null` if the pattern does not
  occur. An unmatched optional group reports as an empty `String`, matching
  the underlying RE2 submatch convention.
- `replace` rewrites every match with `replacement`, which may reference
  capture groups as `$1`, `$2`, and so on.
- `split` divides `text` on every match of the pattern.

Every argument is `String` and `NonNull`. `has`/`has not` (§29) report these
six names as existing members of a `Pattern` instance.

### 49.3 Caching and determinism

A `Pattern`'s only observable state is its source pattern text (readable
through ordinary Class semantics); the compiled matcher itself is an
implementation detail cached internally by pattern text, so repeated use of
the same `Pattern` value -- or repeated `Regex.compile` calls with an
identical pattern string -- does not repeatedly pay compilation cost.
Matching, replacement, and splitting are deterministic for a given pattern
and input.

### 49.4 RegexError

```ahd
attempt {
    Regex.compile("(unterminated")
}
except RegexError as error {
    write(error.message)
}
```

`RegexError` is raised only by `Regex.compile` on invalid pattern syntax. No
other `Pattern` operation raises it: match, find, replace, and split never
fail once a `Pattern` exists.

---

## 50. Expression Lambdas (v0.1.10)

Lambda is a concise expression syntax for creating a value of AhdCode's
existing `Function` type. It is not a `Lambda` type, a second callable family,
or a replacement for named Function declarations.

### 50.1 Grammar

```text
lambda-expression  ::= "lambda" "(" [ lambda-parameter-list ] ")"
                       "->" expression
lambda-parameter   ::= identifier ":" type-reference
```

The parameter list uses the ordinary comma/newline separator rules. Every
parameter has an explicit type; zero parameters are valid. The existing
Function parameter types apply. Parameter declaration modifiers and default
values are not accepted in v0.1.10; use a named Function declaration when a
default parameter is needed.

```ahd
positive := lambda (x: Int) -> x > 0
difference := lambda (x: Int, y: Int) -> x^2 - y^2
now := lambda () -> Time.now()
```

There is no written lambda return annotation. The static return type and
return null-state are inferred from the single body expression. An invalid or
unresolved body type is a semantic error; it never falls back to dynamic
typing. Existing assignability, nullability, arity, and conversion rules apply
unchanged, including the lack of hidden String/numeric/truthiness coercions.

### 50.2 Expression-only boundary

A lambda body is exactly one expression. A `{ ... }` block, `return`, `if`,
loop, declaration, `attempt`, or any other statement body is invalid. Logic
that requires statements uses the unchanged named Function form:

```ahd
positive: Function := (x: Int) -> Bool {
    if x <= 0 {
        return false
    }
    return true
}
```

### 50.3 Function compatibility and callbacks

The compiler preserves the concrete callable signature internally while the
public source type remains `Function`:

```ahd
positive: Function := lambda (x: Int) -> x > 0
inferred := lambda (x: Int) -> x > 0
values.filter(lambda (x: Int) -> x > 0)
values.map(lambda (x: Int) -> x^2)
values.sort(lambda (x: Int) -> -x)
```

A lambda works anywhere that exact Function signature is accepted. `map`,
`filter`, and keyed `sort` keep their existing contracts; no collection API is
added or redesigned.

### 50.4 Scope and lexical capture

Lambda is an expression, not a declaration. Assigning it follows the ordinary
module-root and explicit `Local` declaration rules. Lambda parameters are
implicitly local to the lambda.

v0.1.10 does not introduce a closure environment. A lambda may not capture a
binding from an enclosing callable's lexical scope, including an enclosing
Function parameter or `Local`; such use is a semantic error. Pass the value as
an explicit lambda parameter or use a named Function. Module bindings,
Functions, Classes, and imports retain the existing visibility rules. In
particular, the existing explicit `Global` rule is not weakened merely because
the expression is a lambda.

### 50.5 Implementation and tools

`lambda` is a reserved keyword and parses to a real `LambdaExpr` AST node. The
semantic checker produces the same concrete callable signature used for every
Function value. Lowering emits an ordinary typed Function IR callable and a
`FunctionValueExpr`; the native Go backend and persistent evaluator therefore
reuse their existing Function adapters and invocation paths. No source rewrite
or runtime Lambda identity exists, and `id()` is not extended to Functions.

The persistent REPL retains lambda Function values between commands. The
formatter uses the ordinary Function parameter-list layout and is idempotent:

```ahd
lambda (x: Int) -> x > 0
```

Long parameter lists break according to the existing 80-column policy; the
single expression remains after `->`.

---

## 51. CSV Standard Module (v0.1.11)

`CSV` is the explicit compiler-registered `builtin:CSV` module; a sibling
`CSV.ahd` cannot shadow it. It transports Strings only and performs no type
inference or DataFrame/table modeling.

```text
parse(text: String, delimiter: String = ",") -> List<List<String>>
stringify(rows: List<List<String>>, delimiter: String = ",") -> String
read(path: String, delimiter: String = ",") -> List<List<String>>
write(path: String, rows: List<List<String>>, delimiter: String = ",") -> Nothing
parseRecords(text: String, delimiter: String = ",") -> List<Pair<String, String>>
readRecords(path: String, delimiter: String = ",") -> List<Pair<String, String>>
stringifyRecords(records: List<Pair<String, String>>, delimiter: String = ",") -> String
writeRecords(path: String, records: List<Pair<String, String>>, delimiter: String = ",") -> Nothing
```

Raw parsing supports standard quoting, escaped quotes, embedded delimiters and
newlines, LF/CRLF, Unicode, empty fields, and variable-width rows. Empty raw
rows stringify to `""`; encoding uses deterministic Go `encoding/csv` output.

Record parsing uses the first row as non-empty unique headers. Empty input and
header-only input return an empty List. Every data row must have exactly the
header width. Record writing takes column order from the first Pair; every
later Pair must have exactly the same key set, although insertion order may
differ. Empty records stringify to `""`.

The delimiter is exactly one valid Unicode scalar and cannot be quote, CR, or
LF. Invalid delimiters, malformed CSV, invalid UTF-8 CSV content, and invalid
header/record shape raise `CSVError`, which derives directly from `Error`.
File access failures preserve `FileError`/`IOError` semantics. Relative paths
use process working directory, including the REPL launch directory.

## 52. Diagnostic quality and recovery (v0.1.11)

Diagnostics carry code, severity, message, hint, and precise source span.
Established codes retain their rules: `PAR010` identifies an assigned/default
expression that begins after its operator's physical line; `SEM022` retains the
no-lexical-capture rule. `PAR013` identifies unsupported leading-dot newline
continuation and recovers at the statement boundary to suppress derivative
parse/semantic cascades without hiding independent later errors.

When a construct is known, incomplete initializer, assignment, binary operand,
index, lambda body, call/list/pair/group, and delimiter messages name the
missing part. Lambda blocks remain rejected as expression-lambda syntax.
Runtime domain failures remain AhdCode Errors (`RegexError`, `ValueError`,
`CSVError`, or `FileError`) and must not expose Go panics or stack traces.

---

# End of AhdCode v0.1 Core Specification
