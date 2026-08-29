# AhdCode v0.1 English Student Guide

This guide is designed for beginners. It will walk you through the AhdCode language step-by-step using everyday language, starting from your very first line of code.

## Table of Contents
- [1. What is AhdCode?](#1-what-is-ahdcode)
- [2. Installation and your first program](#2-installation-and-your-first-program)
- [3. Source basics](#3-source-basics)
- [4. Core types](#4-core-types)
- [5. Operators](#5-operators)
- [6. Strings](#6-strings)
- [7. Input, output, and conversions](#7-input-output-and-conversions)
- [8. Conditions: if and state](#8-conditions-if-and-state)
- [9. Loops: while, until, and for](#9-loops-while-until-and-for)
- [10. Writing and calling Functions](#10-writing-and-calling-functions)
- [11. Local and Global](#11-local-and-global)
- [12. Working with Lists](#12-working-with-lists)
- [13. Reference Behavior](#13-reference-behavior)
- [14. Working with Pair](#14-working-with-pair)
- [15. Constant](#15-constant)
- [16. Null safety](#16-null-safety)
- [17. Class and attributes](#17-class-and-attributes)
- [18. Errors with attempt, except, ultimately, and toss](#18-errors-with-attempt-except-ultimately-and-toss)
- [19. Modules and bring](#19-modules-and-bring)
- [20. Fundamentals Module](#20-fundamentals-module)
- [21. Math Module](#21-math-module)
- [22. Time Module](#22-time-module)
- [23. Latex Module](#23-latex-module)
- [24. Formatter](#24-formatter)
- [25. CLI](#25-cli)
- [26. REPL](#26-repl)
- [27. Common beginner mistakes](#27-common-beginner-mistakes)
- [28. Mini Projects](#28-mini-projects)
- [29. Exercises](#29-exercises)
- [30. Solution Hints](#30-solution-hints)
- [31. Next steps and technical documentation](#31-next-steps-and-technical-documentation)

Türkçe sürüm: [Türkçe Öğrenci Rehberi](STUDENT_GUIDE_TR.md)

This guide helps you build small AhdCode command-line programs even if you
have little or no previous programming experience. Work through the chapters
in order, run each example, and try the small changes suggested along the way.

## 1. What is AhdCode?

AhdCode is a general-purpose language designed for readable code and
predictable behavior. The compiler checks your program before running it. This
catches many mistakes early, such as using the wrong kind of value or trusting
a value that might be `null`.

> **Technical note:** Checking types before the program runs is called static
> checking.

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

## 3. Source basics

Every AhdCode program is written in a `.ahd` file.
Statements are usually written one per line. You do not need to end lines with
semicolons. Blocks of code are wrapped in braces `{` and `}`.

If you want to leave a note for yourself or other programmers, you can start a single-line comment with `//`. For longer notes, you can use multiline comments starting with `/*` and ending with `*/`. The compiler ignores comments. (Note: multiline comments do not nest).

```ahd
// This is a comment. It is ignored by the compiler.
write("This line runs")
```

When you need to remember a value, you create a variable. An identifier (a variable's name) can start with a letter or `_`. Subsequent characters can contain letters, numbers, and `_`.

### Declaring and changing variables: `:=` and `=`

Use `:=` to create a new variable. Use `=` to give a new value to a variable
that already exists and can be changed.

```ahd
score: Int := 10
name: String := "Ayşe"

write(score)

score = 20
write(score)
```

Expected output:

```text
10
20
```

Using only `=` on the first line is an error because `score` has not been
created yet. For example, `score = 10` without a previous declaration produces a
semantic error. Using `:=` a second time in the same block would try to create the
same variable again, which is also an error.

AhdCode requires you to write the type explicitly (like `Int` and `String` above):

```ahd
age: Int := 19
// The compiler ensures age remains an Int.
```

> **Technical note:** The region where a name can be used is called its scope.
> Letting the compiler determine the type automatically is called type inference.

## 4. Core types

These are the types you will meet most often at the beginning:

| Type | Meaning | Example |
|---|---|---|
| `Int` | Signed 64-bit integer | `42` |
| `Real` | Finite floating-point number | `3.5` |
| `String` | Unicode text that is not changed in place | `"Ayşe"` |
| `Bool` | Logical value | `true`, `false` |
| `List<T>` | Ordered mutable collection of values | `[1, 2]` |
| `Pair<K, V>` | Insertion-ordered key/value collection | `{"Ali": 90}` |
| `Function` | A block of reusable code | |
| `Class` | A custom data structure | |
| `Nothing` | A type for a Function that returns no value | |

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

An `Int` can safely be used as a `Real` where the language permits it. However,
`List<Int>` and `List<Real>` are different types. You cannot directly pass a
`List<Int>` where a `List<Real>` is required.

> **Technical note:** This strict rule for generic collections is called invariance.

In AhdCode, `null` is a state a variable can be in, rather than a normal type of its own. We will cover this in the Null Safety section.

## 5. Operators

AhdCode supports the standard mathematical and logical operators.

### Arithmetic
- `+` addition
- `-` subtraction
- `*` multiplication
- `/` division (always returns a `Real`, so `5 / 2` is `2.5`)
- `%` remainder (requires `Int` values)
- `^` power (right-associative, so `2 ^ 3 ^ 2` means `2 ^ (3 ^ 2)`)

```ahd
write(10 + 5)
write(10 / 4)
write(10 % 3)
write(2 ^ 3)
```

Expected output:
```text
15
2.5
1
8
```

AhdCode checks integer math for overflow. If a result is too large or too small to fit in an `Int`, the program safely stops with an `OverflowError` instead of producing wrong numbers.

> **Watch out:** Division `/` always returns a `Real`. If you need an integer division, use `int(a / b)`.

### Assignment and compound assignment
You can change variables directly with math:
- `+=`, `-=`, `*=`, `/=`, `%=`, `^=`

```ahd
score: Int := 10
score += 5
write(score)
```

> **Watch out:** Because `/` always returns a `Real`, you cannot use `/=` on an `Int` variable. `score /= 2` is invalid if `score` is an `Int`. It is only valid for `Real` variables.

### Increment and decrement
To add or subtract exactly `1` from an `Int`, use `++` or `--`. These must stand alone on their own line; they cannot be used inside another expression.

```ahd
count: Int := 0
count++
write(count)
```

### Comparison, Identity, and Membership

- `==` equals (checks if the values/content are the same according to type)
- `!=` not equals
- `<` less than
- `<=` less than or equal to
- `>` greater than
- `>=` greater than or equal to

AhdCode also provides operators for checking identity, types, and membership:

- `same` checks if two variables point to the exact same object in memory.
- `is` / `is not` check if an object is of a specific Class type.
- `in` / `not in` check if a value exists inside a collection or a String.
- `has` / `has not` check if a Class object has a specific member (attribute or method).

**Using `in` and `not in`:**
You can check if a value is in a `List`, a substring is in a `String`, or a key is in a `Pair`. For `Pair`, `in` checks the keys, not the values.

```ahd
numbers: List<Int> := [10, 20, 30]
write(20 in numbers)
write(99 not in numbers)

text: String := "AhdCode"
write("Code" in text)

scores: Pair<String, Int> := {
    "Ali": 90
    "Ayşe": 95
}
write("Ali" in scores)
```

**Using `has` and `has not`:**
These are used exclusively to check if a Class object has a specific member. The right side must be the actual name of the member, not a String.

`has` checks the object's actual Class at runtime, so storing a Student in a Person variable does not hide Student-only members from `has`. Inherited members from parent Classes also count as existing.

```ahd
Person: Class<> := {
    structure: Attributes := ( name: String )
}
Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )
}

person: Person := Student(name: "Ali", number: 42)

write(person has number) // true
write(person has not nickname) // true
```

*Note: `person is Student` asks "Is this object a Student?", while `person has number` asks "Does this object have a member named number?".*

### Logical operators
- `and` (true if both are true)
- `or` (true if at least one is true)
- `not` (flips true to false, and false to true)

```ahd
age: Int := 20
hasTicket: Bool := true

if age >= 18 and hasTicket {
    write("Welcome!")
}
```

## 6. Strings

A String holds text. AhdCode fully supports Unicode, meaning letters from any language and emojis work naturally. Strings are immutable: once created, a String cannot be changed in place. Operations on Strings return a new String.

You can write strings in three ways:
1. `"double quotes"`
2. `'single quotes'`
3. `"""triple quotes"""` for multiline text

```ahd
greeting: String := "Hello"
letter: String := 'A'
poem: String := """
Roses are red,
Violets are blue.
"""
```

### Escapes and Interpolation
Use a backslash `\` to escape special characters, such as `\"` or `\n` (newline).
You can insert variables directly into strings using braces `{ }`. This is called interpolation.

```ahd
name: String := "Ali"
write("Hello, {name}!")
```

### String API (methods)
AhdCode provides many useful methods for Strings:

```ahd
text: String := "  Ali,Veli,Ayşe  "
clean: String := text.trim()

write(clean.lower())
write(clean.upper())
write(clean.capitalize())
write(clean.split(","))
write(clean.replace("Veli", "Can"))
write(clean.contains("Ayşe"))
write(clean.startsWith("Ali"))
write(clean.endsWith("Can"))
write(clean.count("i"))
write("a✓b✓".index("✓"))
```

Expected output:

```text
ali,veli,ayşe
ALI,VELI,AYŞE
Ali,veli,ayşe
["Ali", "Veli", "Ayşe"]
Ali,Can,Ayşe
true
true
false
1
1
```

A missing search with `String.index` raises a `DomainError`.

### Indexing and Length
You can access a single character using `[ ]` brackets, or find the total number of characters using `len()`. Indexes count from `0`. You can also use negative indexes to count from the end of the string (`-1` is the last character).

```ahd
word: String := "AhdCode"
write(len(word))
write(word[0])
write(word[-1])
```

Expected output:
```text
7
A
e
```

Ordinary invalid String indexing raises `IndexError`.

## 7. Input, output, and conversions

`write(value)` prints a value followed by a newline. `take()` reads one line of
text from the user, while `take(prompt)` first displays a short prompt. The result of `take`
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

### Conversions with `int`, `real`, and `str`

These functions are available everywhere without needing to import them.

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

`int(Real)` cuts off the decimal part (truncates toward zero). 

`int(String)` ignores surrounding spaces and accepts an optional `+` or `-` sign followed only by digits. It does **not** accept a decimal point, exponent, underscore, or base prefix.

`real(String)` accepts decimal integers, fractions, and exponents, but not `NaN` or infinity. 

Invalid text raises a `DomainError`; a number that is too large raises an `OverflowError`. AhdCode does not automatically convert text to numbers; you must be explicit.

## 8. Conditions: `if` and `state`

### `if` and `else`
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

`if score` is invalid because `score` is an `Int`, not a `Bool`. Write an explicit comparison such as `if score > 0`.

### `state`, `condition`, and `default`
When you need to compare one value against many specific matches, use `state`. This is cleaner than writing many `else if` chains.

```ahd
status: String := "active"

state status {
    condition "active" {
        write("Account is active")
    }
    condition "blocked" {
        write("Account is blocked")
    }
    condition default {
        write("Unknown status")
    }
}
```

Expected output:
```text
Account is active
```

The `state` block executes only the first matching `condition` block. It does not fall through to the next condition, so you do not need to write `break`. The `condition default` runs if no other condition matches. 
> **Watch out:** Do not call `state`, `condition`, or `default` "variables". They are control flow keywords, similar to `if` and `else`.

## 9. Loops: `while`, `until`, and `for`

Loops let you repeat code.

### `while` and `until`
`while` checks its condition before running the code inside it. `until` uses
the opposite order: it runs its body first and checks the condition afterward.
Its body therefore runs at least once, and the loop stops when the condition
becomes `true`.

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
```

Expected output:

```text
while 0
while 1
until 3
until 4
```

> **Technical note:** `while` is a pre-check loop, while `until` is a
> post-check loop.

### `break` and `continue`
You can stop a loop early with `break`, or skip the rest of the current round and move to the next one with `continue`.

```ahd
count: Int := 0
while count < 10 {
    count++
    if count == 2 {
        continue
    }
    if count == 4 {
        break
    }
    write("count is {count}")
}
```

Expected output:
```text
count is 1
count is 3
```

### `for` and `between`
Use `for` to loop over a collection of items, or over a range of numbers. 
`between(start, stop)` creates a range that includes the start and **excludes** the stop. A third argument sets the step.

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

Negative steps are supported (e.g., `between(10, 0, -2)` counts down). A zero step raises a `DomainError`. `between` is highly efficient; it doesn't build a massive List in memory, it just counts lazily.

You usually do not need to write the type of a `for` variable; the compiler can
learn it from the values being visited. You may also write the type explicitly:

```ahd
for value in [10, 20, 30] {
    write(value)
}

for value: Int in [10, 20, 30] {
    write(value)
}
```

In both forms, `value` is created only for the loop. It is already local. Do not add `Local` to it.
This form is invalid:

```ahd
// INVALID:
for value: Local Int in [10, 20] {
    write(value)
}
```

> **Technical note:** Letting the compiler determine the type is called type
> inference. A `for` variable is already treated as Local. Snapshot iteration is used for lists, meaning you iterate over the values present at the start of the loop.


## 10. Writing and calling Functions

Every Function has a name in v0.1. You cannot declare a new Function inside
another Function, and there are no lambdas. Each parameter and return value
has an explicit type.

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
cannot mix the two forms. `title` has a default value (`"Student"`), so it is optional when calling the function. 

A Function that does not return a value uses the return type `Nothing`. In a Function returning `Nothing`, a bare `return` ends the Function
immediately without returning a value. When no early exit is needed, writing
`return` is optional; reaching the end of the Function body is enough (natural fall-through).

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

### Recursion
Functions can call themselves. This is called recursion. You must ensure there is a condition that stops the recursion so it doesn't run forever.

```ahd
countdown: Function := (
    n: Int
) -> Nothing {
    if n <= 0 {
        write("Blastoff!")
        return
    }
    write(n)
    countdown(n - 1)
}

countdown(3)
```

Expected output:
```text
3
2
1
Blastoff!
```

### More than one Function with the same name: overloads

You can define several versions of one Function name when their parameter
types differ. Write the first as an ordinary `Function` and the later version
as an `Overload Function`:

```ahd
describe: Function := (
    value: Int
) -> String {
    return "Int {value}"
}

describe: Overload Function := (
    value: Real
) -> String {
    return "Real {value}"
}

write(describe(2))
write(describe(2.5))
```

Expected output:

```text
Int 2
Real 2.5
```

The compiler first chooses a version whose parameter type matches exactly. It
may use the safe `Int`-to-`Real` conversion (widening) when needed. It also prefers versions with fewer default parameters if multiple versions match. If two versions are
equally good, the call is ambiguous and compilation stops. A return type alone
cannot select a version.

> **Technical note:** This selection is called overload resolution. A
> user-written `Function` is not dynamic; before the program runs, the compiler
> must determine the one version each call will use.

## 11. `Local` and `Global`

You can use Function parameters directly inside the Function; do not add
`Local` to them. Write `Local` when you create a variable inside a Function or
an inner block such as `if`, `for`, or `while`. To use a variable created at
the top level of the file from inside a Function, declare that access with
`Global`.

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

Here, `counter` is created at the top level of the file. The line
`counter: Global Int` does not create another counter; it tells the Function to
use the existing variable. `next` is created only inside the Function, so it
uses `Local`.

> **Technical note:** These rules describe a variable's scope: the parts of
> the program where it can be used. `Global` does not make a hidden copy, it refers to the module-root binding.

## 12. Working with Lists

A List is an ordered collection of values. Its first index is `0`, and negative indexes count from the end (`-1` is the last item).

### Adding, sorting, and reversing
```ahd
bring Math

values: List<Int> := [4, 1, 3]
values.add(2)
values.sort()
write(values)

values.reverse()
write(values)

Math.seed(42)
values.shuffle()
write(values)
```
Because the seed is explicit, the output is reproducible:
```text
[1, 2, 3, 4]
[4, 3, 2, 1]
[2, 4, 1, 3]
```

These operations (`sort`, `reverse`, `shuffle`) do not create a new List. They change the order of the List you already have. `sort` uses natural ascending order.

### Clearing, Ejecting, and Slicing
You can remove items from a List. `eject(index)` removes a single item in place. `clear(list)` empties the whole collection.

```ahd
letters: List<String> := ["A", "B", "C", "D"]
letters.eject(1)
write(letters)

clear(letters)
write(letters)
```
Expected output:
```text
["A", "C", "D"]
[]
```

You can also take a slice of a List using `[start:stop]`. This returns a new List.
```ahd
nums: List<Int> := [10, 20, 30, 40, 50]
slice: List<Int> := nums[1:4]
write(slice)
```
Expected output:
```text
[20, 30, 40]
```

### Searching and Counting
`count(value)` returns how many times a value appears in the List. `index(value)` finds the first position of the value.
```ahd
data: List<Int> := [7, 8, 7, 9]
write(data.count(7))
write(data.index(8))
```
Expected output:
```text
2
1
```
> **Watch out:** If `index()` cannot find the value, it does not return `-1`. It raises a `DomainError`.

### Map, Filter, and Keyed Sort
Version 0.1 has no lambdas, so callbacks are named Function values. `map` and
`filter` return new Lists and do not mutate their source. `sort(keyFunction)` sorts the List based on the results of your function (stable sort).

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

absSort: Function := (
    value: Int
) -> Int {
    return abs(value)
}

values: List<Int> := [3, -1, 4, -2]
doubled: List<Int> := values.map(double)
evens: List<Int> := values.filter(isEven)

values.sort(absSort)

write(doubled)
write(evens)
write(values)
```

Expected output:

```text
[6, -2, 8, -4]
[4, -2]
[-1, -2, 3, 4]
```

## 13. Reference Behavior

If two variables are connected to the same List, they both see the same
collection. A change made through one variable is visible through the other.

```ahd
numbers: List<Int> := [10, 20, 30]
alias: List<Int> := numbers

alias[0] = 99
numbers.add(40)

write(numbers)
write(alias)
write(numbers same alias)
write(numbers == alias)
```

Expected output:

```text
[99, 20, 30, 40]
[99, 20, 30, 40]
true
true
```

`same` checks if both variables point to the exact same object in memory. `==` checks if their contents are deeply equal. In this case, since they share the object, both are true.

> **Technical note:** Sharing the same List this way is called reference
> semantics. A second name for the same object is often called an alias.

## 14. Working with Pair

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

If two variables point to the same Pair, a change through one is visible
through the other (Reference Behavior). A missing key raises `KeyError`. Updating a key keeps its
position; removing and re-adding it moves it to the end. You can remove a key with `eject(key)` and empty the Pair with `clear(pair)`.

```ahd
scores: Pair<String, Int> := {"Ali": 85}
scores.eject("Ali")
clear(scores)
```

## 15. Constant

A `Constant` collection cannot be changed. If you try to modify it directly, the compiler will reject it during checking.

```ahd
locked: Constant List<Int> := [1, 2, 3]
// locked[0] = 99 // This causes a compile-time error
```

The entire reachable object graph is deep-frozen. If a `Constant` contains other collections or objects, those inner objects are also frozen. If you attempt to mutate an already frozen object through another variable (an alias) that isn't explicitly marked as Constant, it may produce a `ConstantError` at runtime.

> **Technical note:** A `Constant` value cannot be initialized with `null`.

## 16. Null safety

`null` means “this variable has a known type, but it has no value right now.”
There is no separate `null` type written beside types such as `String` or
`Student`. As the program moves through different branches, the compiler
tracks whether a value is definitely present, definitely `null`, or possibly
`null`.

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

Member access, calls, and indexing are compile-time errors when the value might
be `null`. After a check such as `message != null`, the compiler knows the
value is present inside that block. List elements and Pair values can also be
null, so values read from them may need the same kind of check. 

If you try to use a possibly null value without checking, you will get a compile-time error.

> **Technical note:** Documentation names the three possibilities `Null`,
> `MaybeNull`, and `NonNull`. The compiler learning more after a check is
> called null refinement.


## 17. Class and attributes

A Class defines a custom data structure and groups related functions (methods) together. It declares constructor inputs in `structure: Attributes`. Every non-`Local` structure input becomes an instance attribute.

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

A `Constant` attribute cannot be changed later. A `Local`
structure input is used only while constructing the object and does not become
an attribute. `Confidential` members are unavailable through ordinary access
from outside the Class.

### Parent and child Classes

One Class can receive the features of another Class. Read the parent Class
first, followed by the child that extends it:

```ahd
Person: Class<> := {
    structure: Attributes := (
        name: String
    )

    describe: Function := (
    ) -> String {
        return "Person {attribute.name}"
    }
}

Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )

    describe: Override Function := (
    ) -> String {
        return "{SuperClass.describe()} #{attribute.number}"
    }
}

student: Student := Student(name: "Ayşe", number: 7)
person: Person := student
write(person.describe())
```

Expected output:

```text
Person Ayşe #7
```

`Student` is a child of `Person`. `SuperClass.attributes` carries the parent's
constructor inputs forward. `Override` says that an inherited method is being
replaced intentionally. `SuperClass.describe()` calls the parent's version.

Although the variable `person` has type `Person`, the actual object is a
`Student`, so `Student.describe` runs. 

You can check the true type of an object using the `is` keyword:
```ahd
Person: Class<> := { structure: Attributes := ( name: String ) }
Student: Class<Person> := { structure: Attributes := ( SuperClass.attributes ) }

person: Person := Student(name: "Ayşe")
if person is Student {
    write("This person is a student!")
}
```

> **Technical note:** Keeping a child object in a parent-typed variable is
> called upcasting. Choosing the method for the actual object is called dynamic
> dispatch.

## 18. Errors with `attempt`, `except`, `ultimately`, and `toss`

If code inside `attempt` produces an error, a suitable `except` block can run.
`ultimately` performs a final step whether or not there was an error. Use
`toss` when your own code needs to raise an Error deliberately.

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
    write("Domain error: {error.message}")
}
except IndexError as error {
    write("Index error: {error.message}")
}
ultimately {
    write("Finished")
}
```

Expected output:

```text
Domain error: value must be positive
Finished
```

Common built-in types include `DomainError`, `ValueError`, `IndexError`, `KeyError`,
`OverflowError`, `DivisionByZeroError`, `NullError`, and `ConstantError`. You can have multiple `except` blocks to handle different errors differently.

You can also create your own custom errors by inheriting from the built-in `Error` class:

```ahd
InvalidAgeError: Class<Error> := {
    structure: Attributes := (
        message: String
    )
}

attempt {
    age: Local Int := -5
    if age < 0 {
        toss (InvalidAgeError("Age cannot be negative"))
    }
}
except InvalidAgeError as error {
    write("Custom error caught: {error.message}")
}
```

> **Technical note:** AhdCode runtime errors are catchable Class values.

## 19. Modules and `bring`

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

There are several ways to import things from a module:
- `bring Greeting` imports a namespace, making the call `Greeting.greet("Ayşe")`.
- `from Greeting bring greet` imports the name directly.
- `from Greeting bring ( greet, farewell )` allows you to selectively import multiple names on multiple lines.
- `from Greeting bring all` imports all public, non-`Confidential` names.

Import collisions and circular dependencies are errors.


### Module aliases

You can rename a module namespace when you import it using `as`. This is useful for keeping your code concise.

```ahd
bring Time as T

write(T.Calendar.isLeapYear(2028))
```

The alias replaces the original namespace binding for that import. Writing `bring Time as T` gives you `T`, but it does not automatically give you `Time`. The normal `bring Time` still works if you prefer the full name. You can also alias your own local modules.

This feature does not create qualified type syntax. For types, preserve the current style by importing the name directly:

```ahd
bring Time as T
from Time bring DateTime

current: DateTime := T.now()
```

Do not write `T.DateTime` as a type; that is invalid. Also, you cannot alias individual items like `from Time bring DateTime as DT`.

## 20. Fundamentals Module

These names are predeclared in every module and require no `bring`. They cover standard input/output, text manipulation, and numeric reductions.

```text
write take str int real len clear between abs sum min max
```

| Function | Behavior |
|---|---|
| `write(value)` | writes one value followed by a newline |
| `take()` / `take(prompt)` | reads one line as String |
| `str(value)` | canonical deterministic text |
| `int(Real)` | truncates toward zero |
| `int(String)` | strict signed ASCII-decimal parse |
| `real(Int)` | explicit safe widening |
| `real(String)` | strict decimal/fraction/exponent parse |
| `len(value)` | String characters, List elements, or Pair entries |
| `clear(collection)` | empties List or Pair in place |
| `between(...)` | lazy stop-exclusive Int iteration |
| `abs(number)` | numeric magnitude with exact result type |
| `sum(list)` | numeric reduction; empty List gives `0` or `0.0` |
| `min(list)` / `max(list)` | numeric extrema; empty List raises `DomainError` |

`abs`, `sum`, `min`, and `max` work on both `Int` and `Real` types. `clear` mutates the existing collection, so any other variable sharing it will see it empty. The numeric reductions (`sum`, `min`, `max`) are pure reads, so they work safely on `Constant List` as well.

## 21. Math Module

The `Math` module provides advanced mathematical and random operations. It must be explicitly imported.

### Functions and Constants

```ahd
bring Math

write(Math.PI)
write(Math.sqrt(81))
write(Math.round(3.14159, 2))
```

Expected output:

```text
3.141592653589793
9.0
3.14
```

Here is the complete Math surface available:

| Item | Description |
|---|---|
| `PI`, `E` | Mathematical constants. |
| `round`, `floor`, `ceil` | `round(value, digits)` rounds exact halves away from zero; digits is optional (0..15). `floor` and `ceil` return an `Int`. |
| `sqrt`, `exp` | Square root and exponential function ($e^x$). |
| `sin`, `cos`, `tan` | Trigonometric functions using radians. |
| `log`, `log10` | Natural logarithm and base ten logarithm. |
| `seed`, `random`, `randomInt` | Random number generation functions. |

For exponentiation, use the `^` operator, there is no `Math.pow`. Functions like `abs`, `sum`, `min`, and `max` are in the Fundamentals module, not Math.

### Random State

```ahd
bring Math

Math.seed(42)
write(Math.randomInt(1, 6))
write(Math.random())
```
`randomInt(min, max)` includes **both** bounds. `random()` returns a value with `0.0 <= value < 1.0`.

Using the same seed again reproduces the same sequence of random values. Without a seed, each new program run gets its starting value from the operating system. This number generator must not be used for security or encryption.

`Math.random`, `Math.randomInt`, and `List.shuffle` consume this same program-wide state. Equal `randomInt` bounds and empty/singleton shuffle consume no state.

> **Technical note:** The sequence is pseudo-random. An unseeded run takes its starting value from operating-system entropy.


## 22. Time Module

The `Time` module lets your program read the clock, work with dates, and wait. Like `Math`, it must be imported.

AhdCode has no `Time.DateTime` type syntax, so you import the types you want to name:

```ahd
bring Time
from Time bring DateTime
from Time bring Duration
```

### The current date and time

```ahd
bring Time
from Time bring DateTime

current: DateTime := Time.now()

write(current.year)
write(current.month)
write(current.day)
```

`Time.now()` gives you your computer's **local** time. Version 0.1 has no timezone features at all, so there is nothing to configure.

A `DateTime` has eight pieces of information you can read:

```text
year  month  day  hour  minute  second  millisecond  weekday
```

`weekday` counts from Monday:

| day | number |
|---|---|
| Monday | 1 |
| Tuesday | 2 |
| Wednesday | 3 |
| Thursday | 4 |
| Friday | 5 |
| Saturday | 6 |
| Sunday | 7 |

These are read-only. Writing `current.year = 2030` is an error, exactly like changing any other `Constant`.

### Building a date yourself

```ahd
bring Time
from Time bring DateTime

birthday: DateTime := Time.dateTime(
    year: 2028,
    month: 2,
    day: 29
)

write(birthday.toString())
```

Expected output:

```text
2028-02-29 00:00:00
```

`hour`, `minute`, `second`, and `millisecond` are optional and start at `0`.

If you ask for a date that does not exist, you get a `ValueError`. `2026-02-29` is not a real day, because 2026 is not a leap year, so AhdCode refuses it instead of quietly sliding to March 1.

`toString()` always writes `YYYY-MM-DD HH:MM:SS`, in every language and on every computer.

### Comparing two moments

AhdCode does not use `<` and `>` on dates. You ask the question in words:

```ahd
bring Time
from Time bring DateTime

morning: DateTime := Time.dateTime(year: 2026, month: 1, day: 1, hour: 9)
evening: DateTime := Time.dateTime(year: 2026, month: 1, day: 1, hour: 21)

write(morning.before(evening))
write(morning.after(evening))
write(morning.sameMoment(morning))
```

Expected output:

```text
true
false
true
```

Use `sameMoment` to ask whether two dates mean the same moment. `==` asks a different question — whether they are the same object — which is the normal rule for every Class.

### How much time is between them

```ahd
bring Time
from Time bring DateTime
from Time bring Duration

first: DateTime := Time.dateTime(year: 2026, month: 1, day: 1)
second: DateTime := Time.dateTime(year: 2026, month: 1, day: 2)

gap: Duration := Time.between(first, second)

write(gap.milliseconds)
write(gap.seconds)
```

Expected output:

```text
86400000
86400.0
```

`Time.between(first, second)` means "second minus first". Swapping the two gives a negative `Duration`, which is useful when you want to know that something is in the past.

You can also make a `Duration` directly with `Time.duration(milliseconds: 1500)`.

### Asking about the calendar

Sometimes you only want to know something about the calendar, not about a particular date:

```ahd
bring Time

write(Time.Calendar.isLeapYear(2028))
write(Time.Calendar.isLeapYear(2100))
write(Time.Calendar.daysInMonth(2028, 2))
write(Time.Calendar.weekday(2026, 8, 29))
```

Expected output:

```text
true
false
29
6
```

A leap year divides by 4, but a year ending in `00` must divide by 400. That is why 2000 is a leap year and 2100 is not.

### Measuring and waiting

```ahd
bring Time

start: Real := Time.monotonic()

Time.sleep(500)

elapsed: Real := Time.monotonic() - start

write(elapsed >= 0.5)
```

Watch the units, because they are different on purpose:

- `Time.sleep(...)` waits for a number of **milliseconds**.
- `Time.monotonic()` reports **seconds**.

`Time.monotonic()` on its own is not a date and means nothing by itself. It is only useful as a difference between two readings, which is exactly what you want for measuring how long something took.

`Time.sleep(0)` returns immediately. A negative wait is a `ValueError`, because waiting "minus one millisecond" is not a real request.


## 23. Latex Module

The `Latex` standard library allows you to create PDF documents directly from your AhdCode programs. It uses a bundled offline Tectonic engine, so you do not need to install TeX Live, MiKTeX, or any other external software. Shell escape is intentionally disabled for safety.

The module provides a practical, beginner-friendly API to generate documents safely:

```ahd
bring Latex as L

document: String := L.document(
    L.section("My First Document") +
    L.escape("Hello! This is an ordinary text section.") +
    L.subsection("Math Example") +
    L.equation("E = mc^2")
)

attempt {
    L.pdfFile("output.pdf", document)
    write("PDF created!")
}
except LatexError as error {
    write("Failed to create PDF: {error.message}")
}
```

- `Latex.escape(text)`: Safely escapes ordinary text so it won't be treated as LaTeX code.
- `Latex.section(text)` and `Latex.subsection(text)`: Create headings.
- `Latex.equation(math)`: Accepts raw LaTeX math source.
- `Latex.document(content)`: Wraps your content in a complete, ready-to-compile document.
- `Latex.pdf(document)`: Compiles the document and returns the PDF bytes.
- `Latex.pdfFile(path, document)`: Compiles the document and saves it directly to a file.
- `LatexError`: Raised if compilation fails (for example, if there is a syntax error in your math).

## 24. Formatter

Your program may work even when its spacing and line layout are untidy. The formatter rewrites the file into AhdCode's shared style while preserving your comments:

```bash
ahdcode format hello.ahd
ahdcode format --check hello.ahd
```

The first command updates the file directly (it is idempotent: running it again changes nothing). The second command checks the style without changing anything, which is useful in team environments to ensure code style is followed.

### Valid syntax vs. recommended style

AhdCode's grammar is flexible about spelling: a comma between two items on the same line is required, but everywhere else -- between a Function's parameters, a call's arguments, and a List's or Pair's entries -- a comma is only needed when two items share a line, and a trailing comma is always optional. All three of these calls mean the same thing:

```ahd
add(2, 3)

add(
    2,
    3
)

add(
    2
    3
)
```

Only one placement is not free to choose: the value after `:=` or `=` has to start on the same line as the operator (see [Declarations and mutation](LANGUAGE_TOUR.md) in the language tour).

You never have to pick a style by hand -- write whichever one is comfortable (or whatever an AI assistant hands you), then run `ahdcode format`. It always produces the same result: short constructs collapse onto one line, and anything too long to fit breaks into one item per line with no trailing comma. For example,

```ahd
calculate: Function := (first: Int, second: Int, description: String, flag: Bool) -> Real {
    return first
}
```

becomes

```ahd
calculate: Function := (
    first: Int
    second: Int
    description: String
    flag: Bool
) -> Real {
    return first
}
```

while a short signature like `check: Function := (x: Int) -> Bool { ... }` stays exactly on one line.

## 25. CLI

AhdCode comes with a simple command-line interface.

- `ahdcode run file.ahd`: Runs a program directly.
- `ahdcode build file.ahd`: Compiles a program into a native executable that can be run on its own without needing the AhdCode compiler installed.
- `ahdcode format file.ahd`: Formats the file.
- `ahdcode --help`: Shows help for all commands.
- `ahdcode --version`: Shows the current compiler version.

## 26. REPL

Run `ahdcode` by itself to open the REPL (Read-Eval-Print Loop) for small experiments without a file. 

The REPL uses the exact same rules as the file compiler. Successful commands remain in the session. A failed command does not erase the last working state, so you can just try again. 

```text
> x: Int := 5
> x: Int := 7
error: duplicate declaration
> x = 7
> write(x)
7
```

> **Watch out:** Because the REPL replays successful work when you enter new lines, random behavior might replay unless you use `Math.seed(...)` for deterministic operations. Test interactive input with `take` in a `.ahd` file rather than in the REPL.

## 27. Common beginner mistakes

Here are common errors you might see and how to fix them:

**1. Using `=` before declaration**
- Wrong: `score = 10`
- Why: You must create a variable before assigning to it.
- Correct: `score: Int := 10`

**2. Duplicate declaration**
- Wrong: `score: Int := 10 \n score: Int := 20`
- Why: `score` already exists in this block.
- Correct: `score: Int := 10 \n score = 20`

**3. Missing `Local` in inner block**
- Wrong: `if true { result: Int := 1 }`
- Why: New variables in inner blocks need `Local`.
- Correct: `if true { result: Local Int := 1 }`

**4. Wrong `Local` in `for` loop**
- Wrong: `for item: Local Int in items`
- Why: The `for` variable is already local by design.
- Correct: `for item: Int in items`

**5. Missing `Global` for module variables**
- Wrong: `count: Int := 0 \n f: Function := () -> Nothing { count = 1 }`
- Why: To modify the module-root variable, you must explicitly declare access.
- Correct: `f: Function := () -> Nothing { count: Global Int \n count = 1 }`

**6. Truthiness**
- Wrong: `if 1 { write("Yes") }`
- Why: Conditions must be strictly `Bool`.
- Correct: `if 1 > 0 { write("Yes") }`

**7. Unsafe null use**
- Wrong: `name: String := null \n write(name.upper())`
- Why: `name` might be null, causing a crash. The compiler rejects this.
- Correct: `if name != null { write(name.upper()) }`

**8. Mixed positional and named arguments**
- Wrong: `greet("Ali", title: "Dr")`
- Why: You must use all positional or all named arguments.
- Correct: `greet(name: "Ali", title: "Dr")`

**9. Overload ambiguity**
- Wrong: Having `f(Int)` and `f(Real)` with defaults, then calling `f()`.
- Why: The compiler cannot guess which one you meant.
- Correct: Provide arguments to make the match exact.

**10. Wrong List element type**
- Wrong: `list: List<Int> := [1, 2.5]`
- Why: `2.5` is a `Real`, not an `Int`.
- Correct: Use `List<Real> := [1.0, 2.5]` or `int(2.5)`.

**11. Modifying Constant**
- Wrong: `locked: Constant List<Int> := [1] \n locked[0] = 2`
- Why: Constants cannot be changed in place.
- Correct: Remove `Constant` if you intend to modify it.

**12. Zero step in between**
- Wrong: `between(1, 10, 0)`
- Why: A step of 0 creates an infinite loop, which `between` rejects.
- Correct: `between(1, 10, 1)`

**13. Invalid int string**
- Wrong: `int("3.14")`
- Why: `int()` parsing is strict and does not accept decimal points.
- Correct: `int(real("3.14"))`

**14. Modulo on Real**
- Wrong: `5.5 % 2.0`
- Why: The `%` operator is for `Int` only.
- Correct: Use `Int` values, e.g. `5 % 2`.

**15. Int division assignment**
- Wrong: `count: Int := 4 \n count /= 2`
- Why: `/` returns a `Real`, which cannot be assigned to an `Int`.
- Correct: `count = int(count / 2)`

**16. Missing module**
- Wrong: `bring Math` without it existing locally (Math is built-in, but for your own `bring Greeting`, `Greeting.ahd` must exist).
- Why: Modules must be sibling files.
- Correct: Ensure `Greeting.ahd` is in the same folder.

**17. Wrong Class Override usage**
- Wrong: Writing `Override` on a method that does not exist in the parent Class.
- Why: `Override` strictly means replacing an existing parent method.
- Correct: Remove `Override` if it is a new method.

**18. Invalid return**
- Wrong: `return "Done"` inside a `-> Nothing` function.
- Why: The function promised to return `Nothing`.
- Correct: Use a bare `return`.

**19. String mutation**
- Wrong: `name[0] = 'B'`
- Why: Strings are immutable.
- Correct: Use `replace` or build a new String.

**20. Unseeded reproducible random**
- Wrong: Expecting `Math.randomInt(1,6)` to be the same without `Math.seed(42)`.
- Why: Unseeded random uses the OS entropy and is not reproducible.
- Correct: Use `Math.seed(42)` before rolling.

## 28. Mini Projects

These mini projects combine the ideas taught in this guide. Try building them on your own!

1. **Grade Average Calculator**: Ask the user for 5 grades. Put them in a `List<Int>`. Filter out invalid grades (less than 0 or greater than 100). Calculate the average, print the minimum, maximum, and whether the student passed (average >= 50).
2. **Simple Calculator**: Use `take()` to get two numbers and an operator (`+`, `-`, `*`, `/`). Use `state` on the operator to perform the math and write the result. Handle division by zero using `attempt`/`except`.
3. **Number Statistics**: Generate 100 random numbers using `Math.randomInt(1, 100)`. Count how many are even, how many are odd, and sort them. Use a function to check if a number is a prime number, and filter the list to show only primes.
4. **Word Analysis**: Ask for a sentence. Use `split(" ")` to get the words. Count the number of words, find the longest word, and build a `Pair<String, Int>` that maps each word to its length.
5. **Menu-Driven Program**: Create a banking simulation using an `until` loop. Show a menu: 1. Deposit, 2. Withdraw, 3. Balance, 0. Exit. Keep track of the balance in an `Int` and stop when the user chooses 0.
6. **Student Record with Class**: Create a `Student` Class and a `Course` Class. A Course contains a `List<Student>`. Write a method to add a student, and a method to calculate the class average.
7. **Seeded Random Game**: Use `Math.seed(42)` to generate a "secret number" between 1 and 100. Ask the user to guess it. Tell them "higher" or "lower" until they guess it. Because it is seeded, the secret number will be the same every time you run it—great for testing!

## 29. Exercises

Build each program in small steps rather than looking for a complete solution immediately.

### Beginner
1. Read a user's name and age, then print how old they will be next year.
2. Read a Celsius value as `Real` and calculate its Fahrenheit equivalent (`C * 9/5 + 32`).
3. Read an `Int` and use `%` to print whether it is odd or even.
4. Use `until` to show a menu at least once and stop when the user enters `0`.
5. Write a Function that trims a name, converts it to lowercase, and capitalizes its first character.
6. Create an empty `List<Int>`, add 3 numbers, and print the `sum` and `len`.
7. Iterate over `between(10, 0, -1)` to print a countdown.

### Intermediate
8. Read a sentence and replace all spaces with underscores.
9. Display `min` and `max` for a grade List while guarding against an empty List.
10. Associate names with scores in a `Pair`, update one score, and print entries.
11. Call `Math.seed(42)`, generate ten rolls with `randomInt(1, 6)`, and confirm that another run repeats them.
12. Use `map` to square all numbers in a List.
13. Create a `Student` Class with `name` and a `Constant number` attribute plus a method that returns a summary.
14. Write a recursive Function to calculate the factorial of a number.

### Challenge
15. Use `attempt` to safely handle `DomainError` when calling `int()` on invalid user input.
16. Sort a `List<String>` by the length of the string using a `keyFunction`.
17. Create a `Shape` parent Class and a `Circle` child Class with an `Override` method for area.
18. Write a function that accepts a `String` and returns a `Pair` counting how many times each character appears.
19. Demonstrate `break` and `continue` by finding the first 5 even numbers in a large range, skipping numbers divisible by 3.
20. Create a module `MathUtils.ahd` with a function for calculating the area of a rectangle, and `bring` it into a `main.ahd` file to use it.

## 30. Solution Hints

1. `take` returns String; convert the age with `int(...)` and add `1`.
2. Break the formula into small parts. Start with `real(take(...))` and use Real literals.
3. `value % 2 == 0` produces a `Bool`.
4. `until` is post-check, so the menu can be printed at the start of its body.
5. Try chaining `trim`, `lower`, and `capitalize` in one return expression.
6. Give an empty `List<Int>` an explicit type and append each input with `add`.
7. Negative step counts down; remember `between` excludes the stop value.
8. Use `String.replace(" ", "_")`.
9. `min` and `max` raise `DomainError` for an empty List; first check `len(grades) > 0`.
10. Use `Pair<String, Int>`; a Pair `for` loop yields keys in insertion order.
11. Set the seed once before rolling. Inclusive bounds mean the arguments can be exactly `1, 6`.
12. Your callback function should return `value * value`.
13. Use the curated Class example as a model for `structure: Attributes`.
14. The recursion base case is `n <= 1`, returning 1.
15. Put the `int(take())` inside `attempt` and handle `DomainError` in `except`.
16. The key function should take a `String` and return `len(value)`.
17. Use `Math.PI * (radius ^ 2)`.
18. Iterate through the string, check if the character is in the Pair, and add 1.
19. `if i % 3 == 0 { continue }`. `if count == 5 { break }`.
20. Use `from MathUtils bring calculateArea`.

## 31. Next steps and technical documentation

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

For more working programs, explore the [curated v0.1 examples](../examples/v0.1/README.md).


