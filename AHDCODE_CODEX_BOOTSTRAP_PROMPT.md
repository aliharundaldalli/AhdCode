# Codex Bootstrap Prompt — Implement AhdCode v0.1

You are the primary implementation agent for a new programming language named **AhdCode**.

The authoritative language contract is:

`AHDCODE_LANGUAGE_SPEC_v0.1.md`

Read that file in full before writing implementation code.

This is not a request to casually invent a syntax or make a toy regex transpiler. The language design has already been discussed and intentionally frozen for v0.1. Your job is to implement it faithfully, incrementally, with strong tests, useful diagnostics, and clear separation between compiler layers.

If this prompt and the specification seem to disagree, stop and report the conflict rather than silently choosing one.

---

## Mission

Build the first serious AhdCode toolchain.

The initial target is **terminal/CLI programming**, not web development.

Do not implement:

- AhdWeb
- HTTP
- MySQL
- SMTP
- HTML layouts
- browser integration
- routing
- web-specific config conventions

until the language core works reliably.

The eventual CLI should support:

```bash
ahdcode
ahdcode run hello.ahd
ahdcode build hello.ahd
ahdcode format hello.ahd
```

The long-term compilation path is:

```text
AhdCode source
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
go build
    ↓
native executable
```

Use **Go** for the compiler/runtime implementation unless the repository already contains a deliberately approved different implementation.

Do not hide Python or JavaScript underneath AhdCode.

Do not implement the language with regex replacement.

---

## Non-negotiable engineering rules

### 1. The specification wins

Do not leak Go semantics just because they are convenient.

Examples of AhdCode semantics that must survive translation:

- `Int` is signed 64-bit with checked overflow.
- `Real` uses 64-bit floating point.
- `/` always returns Real.
- `%` is valid only for `Int % Int` and returns Int.
- `^` is right-associative and follows the exact result matrix in the specification.
- `++`/`--` are Int-only standalone statements.
- `int(3.7)` => `3`.
- `int(-3.7)` => `-3`.
- String is immutable.
- List, Pair, and class instances use reference semantics.
- `Constant` deep-freezes referenced structures.
- Pair preserves insertion order.
- missing Pair key => `KeyError`.
- missing List index => `IndexError`.
- class `==` means object identity.
- class `same` means exact type + same object.
- List/Pair `==` is deep comparison.
- `for` uses a shallow snapshot of iteration state.
- conditions must be Bool.
- `and`/`or` short-circuit.
- `until` is post-check and executes its body at least once.
- String + number is a type error.
- no hidden truthiness.
- no user-defined operator overloading.

### 2. Never invent public syntax silently

If a real ambiguity blocks implementation:

- identify it;
- show the conflicting or missing rule;
- propose the smallest options;
- recommend one;
- mark the implementation point as pending;
- ask for a language decision.

Do not silently copy Python/C/Go/JavaScript.

### 3. Error messages are product features

AhdCode is supposed to be readable and teachable.

Diagnostics should eventually include:

- stable error code/category;
- file path;
- line/column;
- source span;
- concise explanation;
- expected vs received type/token where relevant;
- useful recovery hint when safe.

Never expose raw Go panic output to an ordinary AhdCode user unless there is an internal compiler bug.

### 4. Parser can be flexible; formatter is canonical

The parser must accept presentation variants explicitly allowed by the spec.

Example valid forms:

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

But this is invalid:

```ahd
swap(a b)
```

Whitespace alone on the same line is not an argument separator.

---

## Suggested repository architecture

Use a clean Go compiler architecture. A reasonable shape is:

```text
/
├── cmd/
│   └── ahdcode/
├── internal/
│   ├── source/
│   ├── syntax/
│   │   ├── token/
│   │   └── ast/
│   ├── lexer/
│   ├── parser/
│   ├── semantic/
│   ├── types/
│   ├── diagnostics/
│   ├── ir/
│   ├── formatter/
│   ├── module/
│   ├── runtime/
│   └── codegen/
│       └── golang/
├── stdlib/
│   └── Fundamentals/
├── examples/
├── tests/
├── AHDCODE_LANGUAGE_SPEC_v0.1.md
└── README.md
```

You may choose a better Go layout if justified, but do not collapse lexer/parser/type-checker/codegen into one giant file.

---

# PHASE 0 — Spec audit

Before implementing:

1. Read the entire spec.
2. Extract the exact current:
   - keyword set;
   - operator set;
   - punctuation set;
   - literal kinds;
   - type forms;
   - statement forms;
   - expression forms;
   - declarations;
   - control-flow forms.
3. Create a compact internal implementation checklist.
4. Identify only genuine contradictions.
5. Do not “improve” the language while implementing it.

If useful, add `docs/implementation_notes.md`, but clearly distinguish compiler notes from public language rules.

---

# PHASE 1 — Lexer

Implement a real lexer with source spans.

Each token should preserve enough information for good diagnostics:

- source/file ID;
- start/end offset;
- line;
- column.

Must support:

### Unicode identifiers

Examples:

```ahd
öğrenci
çarpan2
_private
Student
student
```

Case-sensitive.

Source files are UTF-8; invalid UTF-8 is a lexical error. Identifier start is `_` or Unicode `XID_Start`, and continuation is `_` or `XID_Continue`. Normalize identifiers with NFKC before comparison and symbol lookup. Do not case-fold.

### Reserved and contextual keywords

Use this exact v0.1 reserved set:

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

Treat `structure`, `attribute`, and `SuperClass` as contextual keywords only in the contexts defined by the specification. Built-in functions, error class names, and module names remain ordinary predeclared/imported identifiers.

### Ordinary numeric literals

Implement exactly:

```text
digit       := "0" ... "9"
digits      := digit+
exponent    := ("e" | "E") ("+" | "-")? digits
IntLiteral  := digits
RealLiteral := digits "." digits exponent?
             | digits exponent
```

Leading zeroes are decimal and valid. Reject `.5`, `5.`, numeric `_` separators, base prefixes, suffixes, `NaN`, and infinity literals.

`+` and `-` are unary operators, not part of the numeric token. Preserve literal source text without applying signed Int64 range rejection in the lexer. The semantic constant evaluator must accept `-9223372036854775808` and reject final signed Int constant values outside Int64 range. Reject a Real literal semantically if it cannot produce a finite float64 value.

### Strings

Support:

```ahd
"hello"
'hello'
"""
hello
"""
'''
hello
'''
```

Matching delimiters are required.

The exact escape set for normal and triple strings is:

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

Reject every other escape form. Normal quoted strings cannot contain physical newlines; triple strings can.

Preserve triple-string content exactly between delimiters. Do not dedent, trim, or remove a first/last newline. Only three matching consecutive unescaped quotes close the string; an escaped quote cannot participate in the closing delimiter.

Handle interpolation with a lexer mode stack or an equivalently robust design, not regex replacement. An unescaped `{` in string text opens one non-empty expression and its matching depth-zero `}` closes it. Support nested delimiters, strings, comments, and nested string interpolation inside that expression. Braces inside nested strings/comments do not affect the outer depth. Reject empty interpolation, unmatched string-text `}`, statements/declarations inside interpolation, and missing closing `}`. `\{` and `\}` produce literal braces. Triple-string interpolation may span lines; ordinary-string interpolation may not. Interpolation uses `str(expression)` conversion semantics and rejects `Nothing`.

### Comments

```ahd
// comment
```

and:

```ahd
/*
multiline
*/
```

Multiline comments do not nest. The first `*/` closes the active comment; another `/*` inside it is ordinary comment text.

Comments are formatter-preserved trivia. Preserve their exact source spans, text, and placement in the token/trivia model from Milestone A; do not discard them irreversibly in the lexer.

### Operators/punctuation

At minimum:

```text
:
:=
=
->
+
-
*
/
%
^
+=
-=
*=
/=
%=
^=
++
--
==
!=
<
<=
>
>=
.
,
(
)
[
]
{
}
```

Word operators/keywords must include the relevant language words from the spec.

Keep multi-word constructs such as:

```text
is not
not in
has not
```

as parser-level combinations if that makes the lexer simpler and cleaner.

### Newlines

Newlines are significant.

Do not throw them away globally.

They may:

- terminate statements;
- separate multiline arguments;
- separate multiline list items;
- separate multiline Pair entries.

Whitespace alone on a single line may not substitute for comma/newline argument separation.

An executable `{ ... }` block does not suppress newlines. Newlines inside a block remain statement separators. Only a genuinely continuing expression or expression-level delimited construct may cause the parser to interpret a newline as continuation or item separation.

An infix operator or open expression delimiter may leave an expression incomplete across a newline. A newline after a complete expression terminates its statement, and a binary operator at the start of the next line never reaches backward to continue it.

### Lexer adversarial tests

Write many tests for:

- Unicode;
- case sensitivity;
- malformed quote closure;
- triple strings;
- ordinary quotes inside triple strings;
- escapes;
- braces in interpolation;
- braces escaped with `\{`;
- comments containing fake syntax;
- EOF in string;
- EOF in multiline comment;
- `:` vs `:=`;
- `+` vs `++` vs `+=`;
- `^` vs `^=`;
- comparison operators;
- newline retention.

Do not rush beyond lexer until it is stable.

---

# PHASE 2 — AST

Use typed Go structures/interfaces for AST.

Do not represent the AST as `map[string]any`.

Every AST node should keep a source span.

Model at least:

### Expressions
- literal
- identifier
- unary
- binary
- grouped
- call
- member
- index
- slice
- List literal
- Pair literal
- interpolated String

Do not create a parser-level `ClassConstructionExpr`. `Student(...)` and `calculate(...)` have identical call syntax and must both parse as a general `CallExpr`. Only semantic/name resolution may later determine that a call targets a Class and therefore constructs an instance. The parser must not perform symbol or type resolution.

### Statements
- declaration
- assignment
- compound assignment
- increment/decrement
- expression statement
- return
- break
- continue
- if/else-if/else
- while
- until
- for
- state/condition
- attempt/except/ultimately
- toss
- bring

### Declarations
- Function
- Overload Function
- Override Function
- Class
- structure/Attributes
- module-level values

---

# PHASE 3 — Parser

Use Pratt parsing, precedence climbing, or another real maintainable expression parser.

Do not build expression parsing out of ad-hoc regexes.

Respect precedence from the specification.

Critical tests:

```ahd
2^3^2
```

must parse as:

```ahd
2^(3^2)
```

and:

```ahd
not x == 5
```

must parse as:

```ahd
not (x == 5)
```

### Function calls

Accept:

```ahd
f(a, b)
```

```ahd
f(
    a
    b
)
```

```ahd
f(a
    b)
```

Reject:

```ahd
f(a b)
```

Accept entirely named calls:

```ahd
createUser(
    name: "Ali"
    age: 25
)
```

Reject calls that mix positional and named arguments:

```ahd
createUser("Ali", age: 25)
```

On the same line, named arguments must be comma-separated.

### Function declarations

Support:

```ahd
square: Function := (
    x: Real
) -> Real {
    return x^2
}
```

and the modifiers:

```text
Overload Function
Override Function
```

Ordinary Function declarations are module-root only; method declarations are Class members. Reject Function/Class declarations inside executable blocks and reject nested Class declarations. Existing named Function values may be assigned to Local Function bindings such as `operation: Local Function := add`. Do not add lambdas or nested function/class declarations.

### Class forms

Support:

```ahd
Person: Class := {
}
```

```ahd
Person: Class<> := {
}
```

```ahd
Student: Class<Person> := {
}
```

Class construction uses ordinary callable syntax and is checked against the class `structure` parameters:

```ahd
student: Student := Student(
    name: "Ali"
    age: 20
)
```

Also accept `Pair<String, Int>` and reject `Pair<String Int>`. Plain same-line whitespace is not a generic-argument separator.

### structure

Support:

```ahd
structure: Attributes := (
    name: String
    password: Local String
) {
    ...
}
```

### Error recovery

Add basic parser synchronization so one malformed statement does not necessarily destroy all later diagnostics.

---

# PHASE 4 — Type representation and semantic checker

Build a real internal type model.

Do not pass raw type names as arbitrary strings everywhere.

Required concepts:

```text
Int
Real
String
Bool
Nothing
typed-null state
List<T>
Pair<K,V>
Class
Function value
Object
Error hierarchy
Constant modifier
Local/Global/Confidential modifiers
```

## Numeric rules

- Int is signed 64-bit.
- Real uses float64 semantics.
- Int -> Real is safe widening.
- Real -> Int is not implicit.
- `/` returns Real.
- `Int ^ Int` always produces Int. Its type does not depend on Constant status, compile-time evaluation, or exponent sign. A negative exponent raises `DomainError` during evaluation and overflow raises `OverflowError`.
- `Real ^ Int`, `Int ^ Real`, and `Real ^ Real` return Real.
- `^` is right-associative.
- `%` is valid only as `Int % Int -> Int`.
- `++` and `--` accept only Int and only as standalone statements.
- `Real /= Int` and `Real /= Real` are valid; every `Int /= ...` form is a compile-time type error.
- `Int ^= Int` is valid for every Int right operand; negative exponents and overflow use the checked runtime error behavior. `Int ^= Real` remains invalid.
- `int()` truncates toward zero.

## Compile-time constant expressions

A constant expression is compile-time evaluable and scalar. Permit only scalar literals, parentheses, unary `+`/`-`/`not`, pure built-in scalar operators, and references to scalar Constant bindings whose initializers are constant expressions. Reject Function calls, mutable binding references, member/index access, List/Pair/Class construction, interpolation, and cyclic Constant dependencies.

Use this exact definition for signed Int range checking and other compile-time language requirements. Power typing depends only on operand types, never on optional optimizer or range-analysis proofs.

## null

`null` does not mean dynamic.

Example:

```ahd
student: Student := null
```

The variable type remains Student.

Track the internal analysis states `Null`, `MaybeNull`, and `NonNull`. Refine them across control flow, including `x != null` branches and the right operand of short-circuit `and`. Reject provably unsafe use statically; runtime null checks are only the final safety layer.

Preserve null-state information across modules for exported bindings and every concrete exported callable signature, including callable return state. Imported expressions must begin with the preserved state. This is compiler metadata, not public syntax.

### Implementation notes for cross-module null metadata

Keep the following compiler-mechanism details out of the public language contract:

- compute recursive callable return summaries to a fixed point across strongly connected call-graph components;
- use `MaybeNull` conservatively when a precise safe summary cannot be established;
- include semantic null-state metadata in the compiled module interface/cache identity;
- invalidate and re-check dependent modules when relevant exported metadata changes.

Do not permit:

```ahd
x: Constant Int := null
```

## Lists and Pairs

Never silently infer `Any`.

This must fail:

```ahd
x: List := [1, "A"]
```

Empty collection without enough type information must fail.

Partial nested Pair inference is allowed only when uniquely inferable.

## Generic invariance

Reject implicit:

```text
List<Int> -> List<Real>
Pair<String,Int> -> Pair<String,Real>
```

even though Int -> Real exists.

## Class subtyping

Allow derived-to-base assignment.

---

# PHASE 5 — Scope

Implement lexical scopes.

Rules:

- only direct module-root declarations omit a scope modifier;
- Function and structure parameters are lexically local automatically; explicit `Local` on a structure parameter instead means constructor-local-only and suppresses the instance attribute. Preserve `Constant` and `Confidential` as generated-attribute modifiers on every non-`Local` structure parameter;
- `for` variables and `except ... as error` bindings are implicitly Local and scoped to their bodies;
- every explicit declaration in an executable lexical scope below module root uses `Local`, including module-level control-flow blocks;
- nested blocks may access enclosing Local bindings in the same callable without `Global`;
- module-root bindings used inside a function, method, or structure body require explicit `Global`, including reads;
- module-level control-flow blocks access module-root bindings directly without `Global`;
- even reads of global values require declaration;
- no hidden global capture;
- shadowing across nested scopes allowed;
- same-scope duplicate `:=` declaration rejected.

Add tests for typo-like accidental new names and scope leakage.

---

# PHASE 6 — Function semantics

Implement:

- Function;
- one return value;
- Nothing;
- typed null return;
- default arguments;
- positional calls;
- named calls;
- no mixed positional+named calls;
- first-class named functions;
- no lambdas.

## Return-path checking

For non-Nothing functions, every reachable path must return a compatible value or typed null.

Do this in AhdCode semantic analysis rather than hoping Go catches it later.

## Function-valued parameters

A parameter may simply be typed:

```ahd
operation: Function
```

Infer callable compatibility from use/assignment/call context.

If impossible to infer safely, emit a type inference error.

Do not create a hidden dynamic callable escape hatch.

---

# PHASE 7 — Overloads

Implement deterministic overload resolution.

Priority:

1. exact types;
2. safe widening such as Int -> Real;
3. otherwise reject.

Equal best matches => ambiguity error.

Return type alone never distinguishes overloads.

Test defaults, named arguments, wrong arity, and ambiguity.

---

# PHASE 8 — Collections and references

## List

Implement:

- homogeneous type;
- dynamic size;
- ordering;
- negative index;
- IndexError;
- `[start:end]` slicing;
- no step syntax;
- compatible List concatenation;
- reference semantics.

## Pair

Implement:

- homogeneous key type;
- homogeneous value type;
- insertion order;
- allowed key types;
- duplicate literal key compile error;
- KeyError for missing key;
- update existing key without reordering;
- remove + re-add appends;
- reference semantics.

## Alias test

This must mutate the original:

```ahd
a: List<Int> := [1, 2, 3]
b: List<Int> := a
b[0] = 50
write(a[0])
```

But this:

```ahd
change: Function := (
    values: List<Int>
) -> Nothing {
    values = [9, 9]
}
```

must only rebind the local parameter.

---

# PHASE 9 — Constant deep freeze

This is important.

If two names point to the same mutable object and any reference declares the object Constant, the object itself becomes deeply frozen.

Do not implement Constant merely as “this variable cannot be rebound”.

Mutation through aliases must also fail.

Nested referenced contents must respect deep freeze.

Write adversarial tests.

A programmer who wants mutation must explicitly copy/deepCopy to a new mutable object.

---

# PHASE 10 — Classes

Provide built-in `Object`.

Class declarations are module-root only. Methods are Function declarations in Class member scope. Executable blocks and Class bodies may not contain nested Class declarations.

Treat:

```ahd
Person: Class := {
}
```

and:

```ahd
Person: Class<> := {
}
```

as root Object-derived class forms.

Construct instances through the normal callable form:

```ahd
student: Student := Student(
    name: "Ali"
    age: 20
)
```

Constructor arguments follow the ordinary positional-or-named, never-mixed call rules.

Support one direct superclass:

```ahd
Student: Class<Person> := {
}
```

## Attributes

Every non-Local structure entry automatically becomes an attribute.

Use internal class access:

```ahd
attribute.name
```

## Local constructor input

```ahd
password: Local String
```

must not become an attribute.

## Attributes created in structure body

Allow:

```ahd
attribute.passwordHash: Confidential String := ...
```

## SuperClass

Implement:

```ahd
SuperClass.attributes
```

for inherited structure expansion.

Implement:

```ahd
SuperClass.method()
```

for explicit direct-super invocation.

`structure` may not return a value and must reject `return`.

---

# PHASE 11 — Override

Require explicit override intent.

If a derived class declares a plain Function with the same compatible signature as an inherited method, reject it.

Require:

```ahd
describe: Override Function := (
) -> String {
    ...
}
```

Also reject Override if no compatible parent method exists.

This is deliberate protection against accidental override.

---

# PHASE 12 — Confidential

There is one special visibility concept.

Do not invent private/protected/internal keywords.

Default is public.

Class/member Confidential:

- visible to defining class;
- visible to subclasses;
- not visible through ordinary external access.

Module-level Confidential:

- module implementation detail;
- not publicly bringable.

---

# PHASE 13 — Control flow

Implement:

```text
if
else if
else
while
until
for
break
continue
state
condition
```

Conditions must be Bool.

No truthiness.

## until

Post-check, at least one execution.

## break / continue

Nearest loop only.

No labels.

## state / condition

No fall-through.

No `break` required.

---

# PHASE 14 — for snapshot semantics

At the beginning of `for`, build a shallow iteration snapshot.

If original List/Pair structure changes during iteration, the active loop still walks its starting snapshot.

Example:

```ahd
numbers: List<Int> := [1, 2, 3]

for number in numbers {
    numbers.add(10)
    write(number)
}
```

prints only:

```text
1
2
3
```

Do not deep-copy class objects inside a List. Snapshot references.

Pair iteration snapshots keys in insertion order.

This rule must have dedicated integration tests.

---

# PHASE 15 — Error model

Provide built-in `Error`.

Support user subclasses through ordinary Class inheritance.

Implement:

```text
attempt
except
ultimately
toss
```

## ultimately

It must run:

- on success;
- after handled error;
- before unhandled error propagates;
- before a pending return completes.

If ultimately tosses a new error, that new error becomes active.

Never expose routine Go panics as AhdCode error handling.

---

# PHASE 16 — Operators

Implement exactly the current built-in semantics.

## Numeric
- + - * / % ^
- ^ is exponentiation
- right-associative
- `Int ^ Int -> Int` for every Int exponent
- `Real ^ Int -> Real`
- `Int ^ Real -> Real`
- `Real ^ Real -> Real`
- `/` always returns Real
- `%` only permits `Int % Int -> Int`
- Int/Real widening where the exact operator rule permits it
- safe numeric errors

## Compound
- += -= *= /= %= ^=
- `Real /= Int|Real` is valid
- `Int /= ...` is a compile-time type error
- `%=` requires an Int target and Int right operand
- `Int ^= Int` is valid for every Int exponent; negative exponent raises `DomainError` and overflow raises `OverflowError` during evaluation
- `Int ^= Real` is invalid

## ++ / --
- prefix and postfix spellings accepted;
- Int operands only;
- standalone statements only;
- same increment/decrement behavior;
- forbidden inside expressions.

## String
- String + String
- String * Int
- immutable indexing result is String

## List
- List<T> + List<T> where element types are compatible and equal

No user-defined operator overload.

No implicit String-number conversion.

---

# PHASE 17 — Equality / membership / type operations

Implement:

## ==
- scalar value equality;
- numeric compatible comparison;
- deep List/Pair equality;
- class identity.

## same
- strict type + value/state;
- class exact type + same object.

## is / is not
- type test;
- inheritance-aware.

## in / not in
- List value;
- String substring;
- Pair key.

## has / has not
- Class/Object member existence only.

Pair does not use `has`.

---

# PHASE 18 — Boolean semantics

Only Bool values participate.

`and` and `or` short-circuit.

Test specifically:

```ahd
if user != null and user.age >= 18 {
}
```

The second operand must not run when user is null.

Parse:

```ahd
not x == 5
```

as:

```ahd
not (x == 5)
```

---

# PHASE 19 — Numeric safety

Do not blindly accept Go's default low-level arithmetic behavior where it violates AhdCode.

Implement:

- division by zero error;
- checked Int64 overflow;
- Real overflow handling;
- domain errors instead of silently returning NaN where practical for ordinary Real math.

Test numeric boundaries.

---

# PHASE 20 — bring / modules

v0.1 local module resolution is deliberately minimal: a normalized single identifier `ModuleName` resolves to sibling `ModuleName.ahd` relative to the importing source. Do not add dotted paths, search paths, package roots, or path syntax. Registered built-in module names resolve through the built-in registry and are not shadowed by local files.

Implementation must canonicalize resolved source identities before graph/cache lookup so spelling aliases do not analyze one physical module twice. Keep canonical filesystem identity, dependency traversal state, graph caching, and invalidation policy out of the public language contract. The resolver and source loader must remain injectable so in-memory tests and built-in interfaces use the same module-graph pipeline without direct filesystem calls inside semantic analysis.

Represent namespace imports with an explicit semantic module/namespace symbol, not as a Class or runtime Object. Analyze each reachable canonical module at most once, preserve an explicit resolving/resolved/failed state, and reject dependency cycles with their dependency chain. Successful modules expose a deterministic compile-time-only interface containing public symbol types, constants, concrete callable and overload metadata, null-state summaries, and canonical Class/member metadata. Do not serialize AST pointers, runtime values, or Go pointer addresses as identities; failed modules must not publish a successful interface.

Implement:

```ahd
bring Mathematics
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

Rules:

- `bring Mathematics` imports a namespace used as `Mathematics.sqrt(...)`;
- `from Mathematics bring sqrt` imports `sqrt` directly for `sqrt(...)` calls;
- `from ... bring all` directly imports all non-Confidential public symbols;
- circular bring => compile error;
- imported name collision => compile error;
- Confidential => not publicly exported;
- local `.ahd` modules;
- built-in Fundamentals module.

Do not add web folder conventions.

---

# PHASE 21 — Minimal Fundamentals

Do not implement the entire future standard library at once.

The implemented v0.1 Fundamentals surface is:

```text
str
int
real
len
clear
```

These functions, together with core terminal functions `write` and `take`, are predeclared and need no `bring`. `int` accepts `Real` (truncate toward zero) or strict signed ASCII-decimal `String`; `real` accepts `Int` (safe widening) or strict decimal `String` with optional fraction and `e`/`E` exponent. Both String forms trim surrounding Unicode whitespace, reject underscores and non-decimal bases, raise `DomainError` for invalid text, and raise `OverflowError` for out-of-range results. `real(String)` rejects `NaN` and infinity text. Do not turn these entry points into general coercion. Do not add `bool(...)`, truthiness, or other planned Fundamentals until their contracts are normatively specified.

## Canonical str

Implement `str` as locale-independent and deterministic, with no user-defined override in v0.1:

```text
String       -> itself
Int          -> base-10 decimal
Real         -> shortest round-trip decimal, retaining .0 for integral values
Bool         -> true or false
null         -> null
Class value  -> <ClassName>
Function     -> <Function functionName>
```

Preserve `-0.0`, always use `.` as the decimal separator, and use lowercase `e` when canonical scientific notation is needed. Lists and Pairs use recursive literal-like representations while preserving List order and Pair insertion order. Quote nested Strings with double quotes and the exact AhdCode escapes. Do not print Class attributes. Reject `Nothing`. Do not add custom/user-defined `str` dispatch.

Later:

```text
FSet
FLinkedList
FStack
FQueue
FDeque
Matrix
Complex
```

## between

Python range semantics, name is `between`.

## swap

Because multiple assignment is intentionally absent, `swap` must support actual assignable values/locations.

A normal value-only helper is insufficient for general variable swapping. Implement it cleanly as an intrinsic or another compiler-supported lvalue operation if necessary.

## combine

keys List + values List => Pair.

Length mismatch => error.

## jump

Provides stepped selection that core slicing intentionally omits.

---

# PHASE 22 — take / write

Core terminal I/O is:

```ahd
name: String := take("Name: ")
write("Hello {name}")
```

Public AhdCode syntax must use `take` and `write`, not `input` and `print`.

---

# CROSS-CUTTING BACKEND TRACK — IR and Go code generation

Do not wait until every language feature is parsed before creating any backend. After the parser and basic semantic checker are credible, establish a small vertical backend through typed/lowered IR, runtime support, Go generation, and an integration test.

After that first vertical slice, implement later features end to end through:

```text
lexer -> parser -> semantic analysis -> IR/runtime -> Go codegen -> integration test
```

Generated Go does not need to be beautiful.

It does need to preserve AhdCode semantics.

Use a small runtime layer for difficult semantics, including:

- ordered Pair;
- null state;
- deep Constant freeze;
- checked arithmetic;
- class identity;
- Ahd errors;
- snapshot iteration;
- collection helpers.

Do not rely on generated Go accidentally behaving correctly.

The IR should make AhdCode-specific operations explicit where useful, especially checked arithmetic, runtime null checks, deep freeze, snapshot iteration, class identity, and structured error flow. Do not expose IR as public AhdCode syntax.

Use a backend-oriented structured typed IR rather than copying the syntax AST or prematurely forcing SSA. Lowering must consume semantic side tables for resolved symbols, concrete callables, overload choices, null state, and canonical Class identity. Make accepted `Int -> Real` widening explicit, normalize named/default arguments to parameter order, distinguish ordinary calls from Class construction, and preserve single-evaluation lvalue targets. Represent post-check `until`, shallow-snapshot `for`, no-fallthrough `state`, and always-run `ultimately` semantics explicitly. Stable IR identities and debug output must not depend on Go pointer addresses or map iteration order. Validate structural IR invariants before any backend consumes the result.

Add golden codegen tests where useful.

## Implemented Go backend architecture (Milestone G)

The Go backend is an implementation detail. None of the representations below are AhdCode language surface and none of them belong in the public specification.

```text
ir.Compilation
      |
      v
internal/backend/golang.Generate
      |
      v
GeneratedProgram (ahdcode_program.go + ahdcode_runtime.go)
      |
      v
internal/build temporary workspace
      |
      v
go build
      |
      v
native executable
```

`internal/backend/golang/ahdruntime` holds the runtime as an ordinary compiled Go package, so the repository's own `go build`, `go vet`, and tests check it. Its source is embedded verbatim and emitted into every generated program with only its package clause rewritten to `main`. The runtime depends on the Go standard library alone.

### Deterministic name mangling

Generated identifiers are derived from stable IR identities, never from raw AhdCode spelling:

```text
gv_<readable>_<digest>   module global
lv_<readable>_<digest>   local, parameter, receiver, iteration binding
fn_<readable>_<digest>   Function and method
ct_<readable>_<digest>   constructor
Cl_<readable>_<digest>   Class struct
Fd_<readable>_<digest>   Class field
md_<readable>_<digest>   module initialization
```

The readable fragment is decorative and ASCII-filtered; uniqueness comes from a 64-bit FNV-1a digest of the full `ModuleID`/`SymbolID`/`CallableID`/`ClassID`/`FieldID`. A Go keyword, a Unicode identifier, or a cross-module name collision therefore cannot corrupt the generated program. Every runtime identifier is prefixed `Ahd`, which no mangled identifier can produce.

### Type representation

```text
Int          -> int64
Real         -> float64
String       -> string
Bool         -> bool
Nothing      -> no value
List<E>      -> *AhdList[rep(E, nullable)]
Pair<K,V>    -> *AhdPair[rep(K, non-null), rep(V, nullable)]
Class C      -> *Cl_C
Function sig -> func(...) ...
```

A scalar storage slot is boxed as `*T` when its declaration allows null, or when any assignment in the compilation stores a possibly-null value into it. Reference types are already nil-able, so both representations coincide. Collection elements and Pair values always use the nullable representation because element access is `MaybeNull` in the frontend null-state model; Pair keys are never null.

Function parameters and results use the non-null representation, because `ir.Signature` carries no per-parameter or return null-state. A concrete Function whose parameter or return is nullable is reported as an unsupported backend node rather than miscompiled.

### Runtime semantics preserved by helpers

- checked `Int` arithmetic with overflow, modulo-by-zero, and negative-exponent errors;
- `Real` arithmetic that rejects division by zero and non-finite or undefined results instead of exposing `Inf`/`NaN`;
- `AhdList`, pointer-backed so `clear` and indexed writes are observed by every alias;
- `AhdPair`, insertion-ordered, where updating an existing key keeps its position and re-adding a removed key appends;
- shallow iteration snapshots for `for` over List elements, String characters, and Pair keys;
- canonical `str` rendering, including quoted nested Strings and `<ClassName>` instance text;
- character-based String indexing, slicing, and `len`.

Normative runtime decisions are: `%` follows truncated-division, dividend-signed remainder semantics; `take` returns the input line without its terminator and yields an empty String at end of input; `List`/`Pair` `==` is deep value equality while `same` compares object identity.

### Control flow

`until` is emitted as a Go loop whose condition is checked at the top of every iteration after the first, so the body always runs once and `continue` still reaches the condition check. `state` evaluates its subject into one temporary and lowers to a `switch` without fall-through. Compound assignment and `++`/`--` evaluate their receiver and index exactly once through explicit temporaries.

### Backend diagnostics

`BCK001` unsupported IR node, `BCK002` invalid runtime representation, `BCK003` code generation failure, `BCK004` generated-source formatting failure, `BCK005` `go build` failure, `BCK006` missing Go toolchain, `BCK007` build workspace failure. A backend never panics, never emits a silently wrong program, and never silently ignores a node it cannot lower.

### Build pipeline

`internal/build` discovers the Go toolchain through `PATH` first and only then through well-known install locations, materializes the generated program in a private temporary workspace with its own `go.mod`, and invokes `go build` through argument-safe process invocation with no shell string concatenation. The workspace is removed afterwards and the user's source tree is never written to.

---

# PHASE 24 — REPL

Use the same lexer/parser/semantic engine as files.

Do not create a second incompatible mini-language.

Persist successful session declarations and statements through the shared compilation pipeline. Ordinary declaration semantics do not change in the REPL: duplicate same-scope `:=` is an error and mutation uses `=`. Failed semantic input or a catchable runtime Error must leave the last successful session state available.

Target:

```text
$ ahdcode

AhdCode v0.1
ahd> x: Int := 5
ahd> write(x^2)
25
```

The v0.1 REPL must support ordinary core declarations, Functions, Classes, multiline constructs, mutation, and recoverable semantic/runtime failures through the shared compilation path. Do not weaken declaration rules for interactive convenience.

---

# PHASE 25 — Formatter

Implement an AST/token-aware formatter, not regex formatting.

Canonical goals:

Short:

```ahd
swap(a, b)
```

Multiline:

```ahd
createStudent(
    name: "Ali"
    number: 123
    active: true
)
```

Function:

```ahd
calculateAverage: Function := (
    values: List<Real>
) -> Real {
    return sum(values) / len(values)
}
```

Formatter must preserve:

- comments;
- multiline strings;
- interpolation;
- escapes;
- semantics.

Require idempotence:

```text
format(format(source)) == format(source)
```

---

# Testing requirements

Testing is mandatory, not optional cleanup.

For each feature aim for:

1. valid parse;
2. invalid parse;
3. valid semantic case;
4. type/scope rejection;
5. runtime behavior;
6. adversarial edge case.

Must test heavily:

- Unicode identifiers;
- case sensitivity;
- quote mismatch;
- triple strings;
- escapes;
- interpolation;
- null arithmetic;
- Constant null rejection;
- Constant alias mutation;
- nested deep freeze;
- local/global scope;
- same-scope redeclaration;
- shadowing;
- positional calls;
- named calls;
- mixed named/positional rejection;
- default args;
- first-class Function inference;
- overload exact vs widening;
- overload ambiguity;
- List homogeneous inference;
- Pair homogeneous inference;
- partial nested Pair inference;
- duplicate Pair keys;
- Pair insertion order;
- KeyError;
- IndexError;
- negative indexing;
- slicing;
- class identity equality;
- same;
- is with inheritance;
- has;
- in/not in;
- override mistakes;
- SuperClass;
- Confidential subclass access;
- Bool-only conditions;
- short-circuiting;
- until executes at least once;
- nearest-loop break/continue;
- for snapshot behavior;
- toss/except;
- ultimately on return;
- division by zero;
- Int overflow;
- Real overflow;
- formatter idempotence;
- comment/trivia preservation through formatting;
- `Pair<String, Int>` acceptance and `Pair<String Int>` rejection;
- constructor calls with named arguments;
- namespace bring versus direct from-bring;
- exact exponent, remainder, increment/decrement, and compound-division typing;
- minimum Int constant `-9223372036854775808` and out-of-range constant rejection;
- exact numeric literal and escape grammars;
- nested interpolation and malformed interpolation diagnostics;
- Local scope boundaries and cross-module null-state preservation.

Use fuzz/property tests for lexer/parser if practical.

---

# Milestones

Do not dump one giant patch.

## Milestone A — Lexer foundation
- Go module/project skeleton
- diagnostics
- tokens
- formatter-preserved comment trivia
- lexer
- lexer tests

## Milestone B — Parser/AST basics
- expressions
- declarations
- assignment
- primitives
- parser tests

## Milestone C — Semantic core
- types
- Local/Global
- null/Nothing
- arithmetic/comparison
- basic Functions
- typed/lowered IR contracts
- minimal vertical Go backend after the basic checker is stable

## Milestone D — Functions
- returns
- defaults
- named calls
- first-class functions
- overloads

## Milestone E — Collections
- List
- Pair
- inference
- indexing/slicing
- reference behavior
- Constant
- end-to-end IR/runtime/codegen integration

## Milestone F — Classes
- Object
- Class<>
- Class<Parent>
- Attributes
- structure Local
- Confidential
- Override
- SuperClass
- end-to-end IR/runtime/codegen integration

## Milestone G — Control and errors
- if/else if/else
- while
- until
- for
- snapshot
- state/condition
- attempt/except/ultimately/toss
- end-to-end IR/runtime/codegen integration

## Milestone H — Modules and Fundamentals
- bring
- minimal Fundamentals
- take/write
- between/swap/combine
- end-to-end IR/runtime/codegen integration

## Milestone I — Toolchain
- run
- build
- backend completion and hardening
- REPL
- formatter
- integration suite

At the end of each milestone:

- run all tests;
- summarize changed files;
- summarize implemented language rules;
- list any unresolved spec issue;
- do not silently continue into unrelated speculative features.

---

# Things you must NOT add

Do not add:

- let
- var
- public `const` keyword replacing Constant
- semicolons
- Float/Double public types
- void instead of Nothing
- print instead of write
- input instead of take
- import instead of bring
- truthiness
- JavaScript coercion
- ===
- tuple assignment
- multiple returns
- lambdas
- static members
- getters/setters
- goto
- labeled break
- slice step
- Char
- user operator overload
- traits
- interfaces
- mixins
- decorators
- multiple inheritance
- web features

unless the language owner explicitly changes the specification.

---

# Quality bar

A smaller correct implementation is preferred over fake completeness.

It is acceptable to say:

> “This feature is not implemented yet.”

It is not acceptable to parse syntax and then execute it with the wrong semantics.

Do not mark a milestone complete because the happy path works.

The language should survive hostile tests.

Treat every semantic rule as a contract.

---

# Your first response/action

Before coding:

1. Read `AHDCODE_LANGUAGE_SPEC_v0.1.md`.
2. Inspect the repository.
3. Produce the concrete compiler architecture you will use.
4. Produce the exact current keyword/token list derived from the specification.
5. Identify genuine ambiguities or contradictions only.
6. Propose Milestone A file layout and tests.
7. Then implement **Milestone A only** unless existing code already provides a stable equivalent.

Do not begin with AhdWeb.

Do not begin with code generation.

Do not invent missing syntax.

The first important victory is not a flashy demo. It is:

> AhdCode source is tokenized correctly, source spans are reliable, malformed input fails cleanly, and the lexer is covered by strong tests.

After that, proceed milestone by milestone.

# End of Codex Bootstrap Prompt
