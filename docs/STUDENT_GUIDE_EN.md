# AhdCode v0.1 English Student Guide

Türkçe sürüm: [Türkçe Öğrenci Rehberi](STUDENT_GUIDE_TR.md)

This guide helps you build small AhdCode command-line programs even if you
have little or no previous programming experience. Work through the chapters
in order, run each example, and try the small changes suggested along the way.

## 1. What is AhdCode?

AhdCode is a statically checked general-purpose language designed around
readable syntax, explicit intent, and predictable behavior. Before a program
runs, the compiler can catch many mistakes, including incompatible types and
unsafe use of a possibly absent value.

Version 0.1 is an experimental learning release. It can run small CLI programs
directly or compile them into native executables.

## 2. Installation and your first program

Building AhdCode from source currently requires Go 1.25 or newer:

```bash
cd AhdCode
go test ./...
go install ./cmd/ahdcode
export PATH="$(go env GOPATH)/bin:$PATH"
ahdcode --version
```

Create a file named `hello.ahd`:

```ahd
name: String := "AhdCode"
write("Hello {name}")
```

Run it:

```bash
ahdcode run hello.ahd
```

Expected output:

```text
Hello AhdCode
```

Try it: replace `AhdCode` with your own name and run the file again.

## 3. Declaring and changing variables: `:=` and `=`

Use `:=` to declare a new binding. Use `=` to change the value of an existing
mutable binding.

```ahd
score: Int := 70
write(score)

score = 85
write(score)
```

Expected output:

```text
70
85
```

Using only `=` on the first line is an error because there is no existing
binding to mutate. Using `:=` a second time in the same scope would instead
try to create a conflicting declaration.

## 4. Core types

These are the types you will meet most often at the beginning:

| Type | Meaning | Example |
|---|---|---|
| `Int` | Signed 64-bit integer | `42` |
| `Real` | Finite floating-point number | `3.5` |
| `String` | Immutable Unicode text | `"Ayşe"` |
| `Bool` | Logical value | `true`, `false` |
| `List<T>` | Ordered mutable collection of values | `[1, 2]` |
| `Pair<K, V>` | Insertion-ordered key/value collection | `{"Ali": 90}` |

```ahd
student: String := "Ayşe"
age: Int := 19
average: Real := 87.5
passed: Bool := average >= 50.0

write("{student}, {age}, {average}, {passed}")
```

Expected output:

```text
Ayşe, 19, 87.5, true
```

An `Int` can safely widen to `Real` where the language permits it. Mutable
generic collections are invariant, however: a `List<Int>` is not a
`List<Real>`.

## 5. Output and input with `write` and `take`

`write(value)` prints a value followed by a newline. `take()` reads one line of
text, while `take(prompt)` first displays a short prompt. The result of `take`
is always a `String`.

```ahd
name: String := take("Name: ")
age: Int := int(take("Age: "))

write("{name} is {age}")
```

Example interaction:

```text
Name: Ali
Age: 20
Ali is 20
```

Try it: ask for a city with a third call to `take`.

## 6. Conversions with `int`, `real`, and `str`

These functions are available in every module without a `bring` statement.

```ahd
write(int(3.7))
write(int(-3.7))
write(int(" +42 "))
write(real(2))
write(real("1e3"))
write(str(true))
```

Expected output:

```text
3
-3
42
2.0
1000.0
true
```

`int(Real)` truncates toward zero. `int(String)` trims surrounding whitespace
and accepts an optional sign followed only by decimal digits. It does not
accept a decimal point, exponent, underscore, or base prefix. `real(String)`
accepts decimal integers, fractions, and exponents, but not `NaN` or infinity.
Invalid text raises `DomainError`; an out-of-range result raises
`OverflowError`.

## 7. `if`, `else`, and `Bool` conditions

Every AhdCode condition must be a `Bool`. There is no truthiness for zero,
empty Strings, or empty Lists.

```ahd
score: Int := 72

if score >= 85 {
    write("Excellent")
}
else if score >= 50 {
    write("Passed")
}
else {
    write("Failed")
}
```

Expected output:

```text
Passed
```

`if score` is invalid. Write an explicit comparison such as `if score > 0`.

## 8. Loops with `while`, `until`, and `for`

`while` checks its condition before entering the body. `until` is a
**post-check** loop: its body runs at least once, and then the loop stops when
the condition becomes `true`.

```ahd
count: Int := 0

while count < 2 {
    write("while {count}")
    count++
}

until count == 4 {
    count++
    write("until {count}")
}

for value in [10, 20, 30] {
    write("for {value}")
}
```

Expected output:

```text
while 0
while 1
until 3
until 4
for 10
for 20
for 30
```

A declaration inside an executable nested scope needs `Local`. `break` exits
the nearest loop, and `continue` moves to its next iteration.

## 9. Integer ranges with `between`

`between(start, stop)` includes the start and **excludes** the stop. A third
argument sets the step.

```ahd
for value in between(1, 6, 2) {
    write(value)
}
```

Expected output:

```text
1
3
5
```

Negative steps are supported. A zero step raises `DomainError`.

## 10. Writing and calling Functions

Functions are named in v0.1; there are no nested Functions or lambdas. Each
parameter and return value has an explicit type.

```ahd
greet: Function := (
    name: String
    title: String := "Student"
) -> String {
    return "Hello {title} {name}"
}

write(greet("Ali"))
write(greet(name: "Ayşe", title: "Dr"))
```

Expected output:

```text
Hello Student Ali
Hello Dr Ayşe
```

A call must use either all positional arguments or all named arguments; it
cannot mix the two forms. A Function that does not return a value uses the
return type `Nothing`.

In a Function returning `Nothing`, a bare `return` ends the Function
immediately without returning a value. When no early exit is needed, writing
`return` is optional; reaching the end of the Function body is enough.

```ahd
showStatus: Function := (
    score: Int
) -> Nothing {
    if score < 0 {
        write("Invalid score")
        return
    }

    write("Score: {score}")
}

hello: Function := (
    name: String
) -> Nothing {
    write("Hello {name}")
}

showStatus(-5)
showStatus(80)
hello("Ayşe")
```

Expected output:

```text
Invalid score
Score: 80
Hello Ayşe
```

The bare `return` in the first `showStatus` call prevents the later `write`
from running. `hello` completes naturally by reaching the end of its body.

## 11. `Local` and `Global`

Function parameters already belong to the Function's lexical scope. New
bindings inside executable nested scopes use `Local`. When a Function needs a
module-root binding, it declares the required access with `Global`.

```ahd
counter: Int := 0

increase: Function := (
) -> Nothing {
    counter: Global Int
    next: Local Int := counter + 1
    counter = next
}

increase()
increase()
write(counter)
```

Expected output:

```text
2
```

`Global` does not create another value. It explicitly connects the Function
to the existing module-root binding.

## 12. String operations

String is immutable and indexed by Unicode character rather than UTF-8 byte.

```ahd
text: String := "  Ali,Veli,Ayşe  "
clean: String := text.trim()

write(clean.lower())
write(clean.split(","))
write(clean.replace("Veli", "Can"))
write(clean.contains("Ayşe"))
write("A✓B" [1])
write("a✓b✓".index("✓"))
```

Expected output:

```text
ali,veli,ayşe
["Ali", "Veli", "Ayşe"]
Ali,Can,Ayşe
true
✓
1
```

Other useful operations include `upper`, `capitalize`, `startsWith`,
`endsWith`, and `count`. A missing search with `String.index` raises
`DomainError` instead of returning `-1`. Ordinary invalid String indexing
raises `IndexError`.

## 13. Working with Lists

A List is ordered, its first index is `0`, and negative indexes are supported.
List is a reference object, so aliases that refer to the same List observe its
mutation.

```ahd
numbers: List<Int> := [10, 20, 30]
alias: List<Int> := numbers

alias[0] = 99
numbers.add(40)

write(numbers)
write(numbers[-1])
write(numbers same alias)
```

Expected output:

```text
[99, 20, 30, 40]
40
true
```

Ordinary invalid indexing raises `IndexError`. In contrast,
`List.index(value)` raises `DomainError` when the value is absent.

A `Constant List<T>` deep-freezes the reachable reference structure, not just
the binding. Mutation through an alias is blocked as well. Remember that
mutable generic collections are invariant.

## 14. `sort`, `reverse`, and `shuffle`

These operations mutate the original List instead of creating a new one.
Aliases therefore see the same change.

```ahd
bring Math

values: List<Int> := [4, 1, 3, 2]
alias: List<Int> := values

values.sort()
write(values)

values.reverse()
write(values)

Math.seed(42)
values.shuffle()
write(values)
write(alias)
```

Because the seed is explicit, the expected output is reproducible:

```text
[1, 2, 3, 4]
[4, 3, 2, 1]
[2, 4, 1, 3]
[2, 4, 1, 3]
```

`shuffle` consumes the same program-wide pseudo-random state as `Math.random`
and `Math.randomInt`. Without an explicit seed, a fresh native process starts
that state from operating-system entropy, so its sequence is not reproducible.
This generator is not suitable for cryptographic use. Shuffling an empty or
singleton List consumes no random state.

## 15. `map`, `filter`, and Function callbacks

Version 0.1 has no lambdas, so callbacks are named Function values. `map` and
`filter` return new Lists and do not mutate their source.

```ahd
double: Function := (
    value: Int
) -> Int {
    return value * 2
}

isEven: Function := (
    value: Int
) -> Bool {
    return value % 2 == 0
}

values: List<Int> := [1, 2, 3, 4]
doubled: List<Int> := values.map(double)
evens: List<Int> := values.filter(isEven)

write(values)
write(doubled)
write(evens)
```

Expected output:

```text
[1, 2, 3, 4]
[2, 4, 6, 8]
[2, 4]
```

## 16. Working with Pair

`Pair<K, V>` stores key/value associations and preserves insertion order. In
v0.1 its key type can only be `String`, `Int`, or `Bool`.

```ahd
scores: Pair<String, Int> := {
    "Ali": 85
    "Ayşe": 92
}

scores["Ali"] = 90
scores["Veli"] = 78

for name in scores {
    write("{name}: {scores[name]}")
}
```

Expected output:

```text
Ali: 90
Ayşe: 92
Veli: 78
```

Pair is also a reference object, so aliases observe mutation. A missing key
raises `KeyError`. Updating a key keeps its position; removing and re-adding it
moves it to the end. A `Constant Pair` deep-freezes its reachable reference
graph.

## 17. Class and attributes

A Class declares constructor inputs in `structure: Attributes`. Every
non-`Local` structure input becomes an instance attribute.

```ahd
Student: Class<> := {
    structure: Attributes := (
        name: String
        number: Constant Int
    )

    describe: Function := (
    ) -> String {
        return "#{attribute.number} {attribute.name}"
    }
}

student: Student := Student(name: "Ali", number: 42)
write(student.describe())
```

Expected output:

```text
#42 Ali
```

A `Constant` attribute cannot be changed and deep-freezes a reference value.
A `Local` structure input is constructor-local and does not become an
attribute. `Confidential` members are unavailable through ordinary external
access.

## 18. Null safety

`null` is not a separate nullable type. It is the state in which a value is
absent. The compiler follows whether a binding is `Null`, `MaybeNull`, or
`NonNull` as control flows through the program.

```ahd
message: String := null

if message == null {
    message = "ready"
}

if message != null and message.contains("read") {
    write(message.upper())
}
```

Expected output:

```text
READY
```

Member access, calls, and indexing on a maybe-null value are compile-time
errors until a check refines it. List elements and Pair values can be null, so
a value read from them may need refinement. A `Constant` cannot be initialized
with null.

## 19. Errors with `attempt`, `except`, `ultimately`, and `toss`

Runtime errors are catchable Class values. `attempt` protects a block,
`except` handles a matching error, `ultimately` runs regardless of the result,
and `toss` raises an Error.

```ahd
requirePositive: Function := (
    value: Int
) -> Int {
    if value <= 0 {
        toss (DomainError("value must be positive"))
    }

    return value
}

attempt {
    result: Local Int := requirePositive(-1)
    write(result)
}
except DomainError as error {
    write("Caught: {error.message}")
}
ultimately {
    write("Finished")
}
```

Expected output:

```text
Caught: value must be positive
Finished
```

Common built-in types include `DomainError`, `IndexError`, `KeyError`,
`OverflowError`, `DivisionByZeroError`, `NullError`, and `ConstantError`.

## 20. Modules and `bring`

A local module is a `.ahd` file in the same directory as its importer. For
example, the module name `Greeting` resolves to `Greeting.ahd`.

`Greeting.ahd`:

```ahd
greet: Function := (
    name: String
) -> String {
    return "Hello from a module, {name}"
}
```

`main.ahd`:

```ahd
from Greeting bring greet

write(greet("Ayşe"))
```

Running `main.ahd` produces:

```text
Hello from a module, Ayşe
```

`bring Greeting` imports a namespace, making the call
`Greeting.greet("Ayşe")`. `from Greeting bring greet` imports the name
directly. Selective multiline imports and `bring all` are supported; `all`
only imports public, non-`Confidential` names. Import collisions and circular
dependencies are errors.

## 21. The Math module

`Math` must be imported explicitly. `randomInt(min, max)` includes **both**
bounds.

```ahd
bring Math

write(Math.sqrt(81))
write(Math.round(3.14159, 2))

Math.seed(42)
write(Math.randomInt(1, 6))
write(Math.randomInt(1, 6))
```

Expected output:

```text
9.0
3.14
2
2
```

The same seed reproduces the same pseudo-random sequence. `Math.random()`
returns a value with `0.0 <= value < 1.0`. Without a seed, native process
startup uses operating-system entropy. This generator is not cryptographically
secure.

## 22. A small combined application: grade summary

The following program brings together input, String, List, Function, loops,
conditions, numeric reductions, and error handling:

```ahd
checkGrade: Function := (
    grade: Int
) -> Int {
    if grade < 0 or grade > 100 {
        toss (DomainError("grade must be between 0 and 100"))
    }

    return grade
}

name: String := take("Student: ").trim().capitalize()
grades: List<Int> := []

for index in between(1, 4) {
    attempt {
        grade: Local Int := checkGrade(int(take("Grade {index}: ")))
        grades.add(grade)
    }
    except DomainError as error {
        write("Invalid input: {error.message}")
    }
}

if len(grades) > 0 {
    average: Local Real := sum(grades) / len(grades)
    write("{name}: {average}")
    write("Lowest: {min(grades)}")
    write("Highest: {max(grades)}")

    if average >= 50.0 {
        write("Passed")
    }
    else {
        write("Failed")
    }
}
else {
    write("No valid grades were entered")
}
```

Example interaction for the inputs `ali`, `90`, `80`, and `70`:

```text
Student: ali
Grade 1: 90
Grade 2: 80
Grade 3: 70
Ali: 80.0
Lowest: 70
Highest: 90
Passed
```

Try it: enter one invalid grade and observe the `except` branch.

## 23. Common beginner mistakes

- Use `:=`, not `=`, when introducing a binding.
- Do not write `if value`; produce a `Bool` with a comparison such as
  `value > 0`.
- Add `Local` to a new declaration in an executable nested scope.
- Declare the required `Global` access when a Function uses a module-root
  binding.
- Remember that an `until` body always runs at least once.
- Remember that `between` excludes its stop.
- List indexing starts at `0`; negative indexes count from the end.
- Missing `List.index`/`String.index` searches raise `DomainError`, while
  ordinary invalid indexing raises `IndexError`.
- `sort`, `reverse`, and `shuffle` mutate their List; `map` and `filter` do not.
- Use a named Function callback instead of a lambda.
- Do not mix named and positional arguments in one call.
- Refine a maybe-null value before member access or indexing.

## 24. Exercises

Build each program in small steps rather than looking for a complete solution
immediately.

1. **Name and age:** Read a user's name and age, then print how old they will be
   next year.
2. **Celsius conversion:** Read a Celsius value as `Real` and calculate its
   Fahrenheit equivalent.
3. **Odd or even:** Read an `Int` and use `%` to print whether it is odd or
   even.
4. **Grade average:** Add three grades to a List and calculate the average with
   `sum` and `len`.
5. **Minimum and maximum:** Display `min` and `max` for a grade List while
   guarding against an empty List.
6. **Simple menu loop:** Use `until` to show a menu at least once and stop when
   the user enters `0`.
7. **String normalization:** Write a Function that trims a name, converts it to
   lowercase, and then capitalizes its first character.
8. **Student-score Pair:** Associate names with scores, update one score, and
   print entries in insertion order.
9. **Deterministic dice:** Call `Math.seed(42)`, generate ten rolls with
   inclusive `randomInt(1, 6)`, and confirm that another run repeats them.
10. **Class-based record:** Create a `Student` Class with `name` and a
    `Constant number` attribute plus a method that returns a summary.

## 25. Solution Hints

1. `take` returns String; convert the age with `int(...)` and add `1`.
2. Break the formula into small parts. Start with `real(take(...))` and use
   Real literals.
3. `value % 2 == 0` produces a `Bool`.
4. Give an empty `List<Int>` an explicit type and append each input with `add`.
5. `min` and `max` raise `DomainError` for an empty List; first check
   `len(grades) > 0`.
6. Because `until` is post-check, the menu can be printed at the start of its
   body.
7. Try chaining `trim`, `lower`, and `capitalize` in one return expression.
8. Use `Pair<String, Int>`; a Pair `for` loop yields keys in insertion order.
9. Set the seed once before rolling. Inclusive bounds mean the arguments can
   be exactly `1, 6`.
10. Use the curated Class example as a model for `structure: Attributes`, named
    construction, and `attribute.name`.

## 26. Next steps and technical documentation

After completing this guide, continue with the detailed project documents:

- [Getting Started](GETTING_STARTED.md)
- [Language Tour](LANGUAGE_TOUR.md)
- [Types and Null](TYPES_AND_NULL.md)
- [Control Flow](CONTROL_FLOW.md)
- [Functions](FUNCTIONS.md)
- [Classes](CLASSES.md)
- [Collections](COLLECTIONS.md)
- [Modules](MODULES.md)
- [Errors](ERRORS.md)
- [Fundamentals](FUNDAMENTALS.md)
- [String API](STRING_API.md)
- [List API](LIST_API.md)
- [Math](MATH.md)
- [CLI](CLI.md)
- [Formatter](FORMATTER.md)
- [REPL](REPL.md)
- [Full v0.1 specification](../AHDCODE_LANGUAGE_SPEC_v0.1.md)

For more working programs, explore the
[curated v0.1 examples](../examples/v0.1/README.md).
