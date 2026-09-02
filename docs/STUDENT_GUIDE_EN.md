# AhdCode v0.2.2 English Student Guide

This guide is designed so that **even someone who has never programmed before** can follow along. You can read it in order from beginning to end; in each section, you will first see what we want to achieve, then write a working example, and finally learn the necessary rules.

You will occasionally see English technical terms next to the code. You do not need to memorize these on your first read. If a box starts with **Technical note**, you don't necessarily have to know that detail immediately to run the program.

The best way to learn is not just by reading the examples, but by running them. Copy an example, change a number or a text inside it, and observe how the result changes.

## Table of Contents
- [1. What is AhdCode?](#1-what-is-ahdcode)
- [2. Installation and your first program](#2-installation-and-your-first-program)
- [3. Code writing basics](#3-code-writing-basics)
- [4. Core types](#4-core-types)
- [5. Operators](#5-operators)
- [6. Strings](#6-strings)
- [7. Input, output, and conversions](#7-input-output-and-conversions)
- [8. Conditions: if and state](#8-conditions-if-and-state)
- [9. Loops: while, until, and for](#9-loops-while-until-and-for)
- [10. Writing and calling functions](#10-writing-and-calling-functions)
- [11. Local and Global](#11-local-and-global)
- [12. Working with Lists (List)](#12-working-with-lists-list)
- [13. Reference Behavior](#13-reference-behavior)
- [14. Working with Pair](#14-working-with-pair)
- [15. Constants (Constant)](#15-constants-constant)
- [16. Null safety](#16-null-safety)
- [17. Classes (Class) and Attributes](#17-classes-class-and-attributes)
- [18. Error handling (`attempt`, `except`, `ultimately`, and `toss`)](#18-error-handling-attempt-except-ultimately-and-toss)
- [19. Modules and bring](#19-modules-and-bring)
- [20. Fundamentals module](#20-fundamentals-module)
- [21. Math module](#21-math-module)
- [22. Time module](#22-time-module)
- [23. Statistics module](#23-statistics-module)
- [24. Plot module](#24-plot-module)
- [25. Numeric module and Complex](#25-numeric-module-and-complex)
- [26. Latex module](#26-latex-module)
- [27. Word module](#27-word-module)
- [28. Excel module](#28-excel-module)
- [29. PDF module](#29-pdf-module)
- [30. Archive module](#30-archive-module)
- [31. JSON module](#31-json-module)
- [32. XML module](#32-xml-module)
- [33. Env module](#33-env-module)
- [34. Lists and KeyValue modules](#34-lists-and-keyvalue-modules)
- [35. Code Formatter](#35-code-formatter)
- [36. Command line (CLI)](#36-command-line-cli)
- [37. Interactive shell (REPL)](#37-interactive-shell-repl)
- [38. Common beginner mistakes](#38-common-beginner-mistakes)
- [39. Small Projects](#39-small-projects)
- [40. Exercises](#40-exercises)
- [41. Solution Hints](#41-solution-hints)
- [42. Next steps and technical docs](#42-next-steps-and-technical-docs)

## 1. What is AhdCode?

A programming language is a way of telling the computer what we want it to do. With AhdCode, for example, you can have it perform calculations, get information from the user, read a file, or create your own small programs.

Your first AhdCode program could be just a single line:

```ahd
write("Hello!")
```

This program prints the following to the screen:

```text
Hello!
```

AhdCode checks the code you wrote before running the program. For example, if you try to use text like a number, or if you use a value that could be `null` without checking it, it will tell you the error before the program even starts, whenever possible. But you don't need to think about these details at the beginning; we'll see examples in later sections.

AhdCode v0.2.2 is the current release. You can run small command-line programs, compile them into standalone local applications, and use the language server (`ahdcode lsp`) from an editor such as VS Code.

> **Technical note:** Checking types before the program runs is called *static checking*.

## 2. Installation and your first program

To build AhdCode from source, you must have Go 1.25 or newer installed on your computer. Run the following commands in the project folder:

```bash
cd AhdCode
go test ./...
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot
export PATH="$(go env GOPATH)/bin:$PATH"
ahdcode --version
```

If you want to use the `Latex` module, you must also stage the offline Latex runtime bundle. This step performs a one-time network fetch to download pinned resources:

```bash
go run ./tooling/latex/cmd/package-latex --output "$(go env GOPATH)"
```

If the last command shows the AhdCode version, you are ready.

Now create a file named `hello.ahd` and write this inside:

```ahd
name: String := "AhdCode"
write("Hello {name}")
```

Run the file:

```bash
ahdcode run hello.ahd
```

Expected output:

```text
Hello AhdCode
```

Here we stored a small piece of information in a name, and then placed it inside the text. You don't need to memorize why the `:=` sign is used for now; we'll learn it in the next section.

**Try it yourself:** Replace `AhdCode` with your own name and run the program again.

## 3. Code writing basics

AhdCode programs are written in files with the `.ahd` extension. Each operation generally sits on its own line; you don't need to put a `;` at the end of a line.

### Notes the computer won't read: comments

You can leave notes for yourself inside the code. A section starting with `//` is a comment on that line:

```ahd
// This line just informs us.
write("This line works")
```

Longer comments can start with `/*` and end with `*/`:

```ahd
/*
This explanation can span
several lines.
*/
write("Continue")
```

### Storing information: variables

We can store a piece of information under a name for the program to use later:

```ahd
name: String := "Ayşe"
age: Int := 19
```

Here `name` and `age` are **variables**. We can use the values on the right with these names later:

```ahd
name: String := "Ayşe"
age: Int := 19

write(name)
write(age)
```

### `:=` creates a new variable, `=` modifies an existing one

When creating it for the first time, use `:=`:

```ahd
score: Int := 10
```

Later, to change the value of the same variable, use `=`:

```ahd
score: Int := 10
write(score)

score = 20
write(score)
```

Output:

```text
10
20
```

You cannot directly write `score = 10` for a variable that has not yet been created. Writing `score: Int := ...` a second time in the same place also means trying to create a new variable, which is an error.

### Writing the type explicitly: a recommended habit, not a requirement

When you're starting out, it's recommended to write the type explicitly. This lets you see at a glance what kind of value a variable holds, and helps you catch mistakes early:

```ahd
age: Int := 19
name: String := "Ayşe"
active: Bool := true
```

This is not a requirement, though. When the initializer's type is already unambiguous, you can leave the type out entirely and just use `:=`; the compiler infers it for you:

```ahd
age := 19       // inferred as Int
name := "Ayşe"  // inferred as String
```

Both forms are completely equivalent. This guide will usually write the type explicitly for readability, but which one you use is up to you.

Whether the type is written explicitly or inferred, once AhdCode has settled on a type for a variable, you cannot change it to a value of a different type:

```ahd
name: String := "Ayşe"
// name = 5   // ERROR: name was created as a String
```

We will also learn about the `Local` rule for inner blocks in section 11; for now, focus on the examples at the top level of the file.

> **Technical note:** Letting the compiler work out the type from the initializer is called *type inference*. This does not mean the variable can later hold a different kind of value at runtime -- AhdCode keeps static types. `name = 5` is still an error, because `name` was inferred as `String` once and for all.

## 4. Core types

We use **types** to specify what kind of information we hold in a variable. It's enough to know the following types in the first stage:

| Type | What it holds | Example |
|---|---|---|
| `Int` | Integer | `42`, `-3` |
| `Real` | Decimal number | `3.5`, `-0.25` |
| `String` | Text | `"Ayşe"` |
| `Bool` | True/false value | `true`, `false` |
| `Complex` | Complex number | `2 + 3I` |
| `List<T>` | Ordered values | `[1, 2, 3]` |
| `Pair<K, V>` | Key-value pairs | `{"Ali": 90}` |
| `Function` | An executable function | we'll see later |
| `Class` | Defines your own data structure | we'll see later |
| `Nothing` | States that a function returns no value | we'll see later |

A simple example:

```ahd
student: String := "Ayşe"
age: Int := 19
average: Real := 87.5
passed: Bool := average >= 50.0

write("{student}, {age}, {average}, {passed}")
```

Output:

```text
Ayşe, 19, 87.5, true
```

Here we didn't write `true` directly into the `passed` variable. Since the result of the comparison `average >= 50.0` is already true or false, AhdCode understands it as a `Bool`.

In some cases, an `Int` can be safely used where a `Real` is required. But collections have distinct types; for example, `List<Int>` and `List<Real>` are not the same type.

If a value might not exist at all, you can add `?` to the end of the type. For example, `String?` means "a String or `null`". We will cover this with examples in section 16.

> **Technical note:** Generic types like `List<Int>` and `List<Real>` not automatically converting to each other is called *invariance*.

## 5. Operators

Operators are symbols and words we use to do math, compare values, or combine conditions.

### Arithmetic and other numeric operations

```ahd
write(10 + 5)   // 15
write(10 - 5)   // 5
write(10 * 5)   // 50
write(10 / 4)   // 2.5
write(10 % 3)   // 1
write(2 ^ 3)    // 8
```

The `/` operation always results in a `Real`. So the result of `5 / 2` is `2.5`, not `2`.

`%` is only used with `Int` values and yields the remainder. `^` is exponentiation.

AhdCode checks for overflow in integer calculations. If a result exceeds `Int` bounds, it raises an `OverflowError` instead of producing the wrong number.

### Shortcut for updating an existing value

```ahd
score: Int := 10
score += 5
write(score) // 15
```

The forms `+=`, `-=`, `*=`, `/=`, `%=`, `^=` exist. Since the result of `/` is a `Real`, you cannot use `/=` on an `Int` variable.

To increment or decrement an `Int` value by exactly one:

```ahd
count: Int := 0
count++
count++
count--
write(count) // 1
```

`++` and `--` are used by themselves on their own line.

### Comparing values

```ahd
age: Int := 20

write(age == 20) // true
write(age != 18) // true
write(age > 18)  // true
write(age <= 30) // true
```

Basic comparisons: `==`, `!=`, `<`, `<=`, `>`, `>=`.

### Asking if a value exists somewhere

`in` and `not in` are very readable:

```ahd
numbers: List<Int> := [10, 20, 30]
write(20 in numbers)      // true
write(99 not in numbers)  // true

text: String := "AhdCode"
write("Code" in text)     // true
```

On a `Pair`, `in` searches keys, not values:

```ahd
scores: Pair<String, Int> := {
    "Ali": 90
    "Ayşe": 95
}

write("Ali" in scores) // true
```

### `and`, `or`, `not`

You can combine multiple true/false conditions:

```ahd
age: Int := 20
hasTicket: Bool := true

if age >= 18 and hasTicket {
    write("Welcome!")
}
```

- `and`: `true` if both sides are true
- `or`: `true` if at least one side is true
- `not`: reverses the result

### When will we need `same`, `is`, and `has`?

These are useful a bit further along:

- `same`: asks if two variables point to the **same object**.
- `is` / `is not`: asks if an object is of a specific Class.
- `has` / `has not`: asks if a specific attribute or method exists in a Class object.

You will see these concepts again with examples in the List references and Classes sections. You do not need to memorize them on your first read.

## 6. Strings

`String` is used to hold text:

```ahd
name: String := "Ayşe"
city: String := 'Hatay'
```

Single quotes and double quotes can both be used. For multi-line text, you can use triple quotes:

```ahd
poem: String := """
Roses are red,
Violets are blue.
"""
```

### Placing a variable inside text

Use curly braces:

```ahd
name: String := "Ali"
age: Int := 20
write("{name} is {age} years old")
```

Output:

```text
Ali is 20 years old
```

This is called *interpolation*.

### Common String operations

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
write(clean.endsWith("Ayşe"))
write(clean.count("i"))
```

`String` values are not modified in place. For example, `clean.upper()` produces a new String; it does not change the `clean` variable itself.

### Accessing a character

The index of the first character is `0`:

```ahd
word: String := "AhdCode"
write(len(word)) // 7
write(word[0])   // A
write(word[-1])  // e
```

Negative indices count from the end; `-1` is the last character. An invalid index raises an `IndexError`. If `String.index(...)` cannot find what it is looking for, it raises a `DomainError`.

### Escape characters

You can use `\` to write special characters inside text. For example, `\n` is a newline, and `\"` is the double quote character.

> **Technical note:** AhdCode Strings work with Unicode characters and are immutable; they cannot be changed in place.

### Raw Strings

A normal String decodes `\` escapes and `{...}` interpolation. Sometimes you want the exact text instead -- for example, a regular-expression pattern full of `{3}`-style quantifiers, or LaTeX source full of backslashes. Prefix the opening quote with `r` for a raw String, and neither escapes nor interpolation are processed:

```ahd
name: String := "Ali"

write(r"{name}")   // {name}, not Ali -- no interpolation
write(r"\n")       // \n as two characters, not a newline

pattern: String := r"^MATH-[0-9]{3}$"
formula: String := r"\frac{x+1}{x-1}"
```

Raw multiline strings work the same way, with `r"""..."""` or `r'''...'''`. In short:

```text
normal String = escapes + interpolation
raw String    = neither escapes nor interpolation
```

A raw String is still an ordinary `String`; there is no separate raw type.

## 7. Input, output, and conversions

Now let our program talk to the user.

### Writing to the screen: `write`

```ahd
write("Hello")
write(42)
```

`write(...)` prints the value to the screen and moves to a new line.

### Getting information from the user: `take`

```ahd
name: String := take("Name: ")
write("Hello {name}")
```

Example usage:

```text
Name: Ali
Hello Ali
```

`take()` always returns a **String**. Even if the user types `20`, initially we have the text `"20"`, not a number.

If we want to do math with the age, we convert it to a number:

```ahd
age: Int := int(take("Age: "))
write("You will be {age + 1} years old next year.")
```

### Type conversions

```ahd
write(int(3.7))       // 3
write(int(" +42 "))  // 42
write(real(2))        // 2.0
write(real("1e3"))   // 1000.0
write(str(true))      // true
```

- `int(...)`: produces an `Int`.
- `real(...)`: produces a `Real`.
- `str(...)`: produces a String.

AhdCode does not automatically convert text to numbers. If the user enters invalid text for a number, such as `int("hello")`, a `DomainError` occurs. For very large values, an `OverflowError` might occur. We will see how to catch errors in section 18.

> **Detail:** `int(String)` only accepts integer texts consisting of signs and digits; it does not accept text like `"3.14"` directly. `real(String)` accepts decimal and exponential notations but does not accept `NaN` or infinity.

## 8. Conditions: `if` and `state`

Programs don't always need to do the same thing. We can make them behave differently based on a condition.

### `if` and `else`

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

Output:

```text
Passed
```

After `if`, there must always be an expression that produces `true` or `false`. Therefore, instead of:

```ahd
// if score { ... }  // invalid
```

we write:

```ahd
if score > 0 {
    write("Positive")
}
```

In AhdCode, `0`, empty text, or an empty list are not automatically considered true/false.

### Comparing the same value with many options: `state`

In cases like a menu selection, `state` can be readable:

```ahd
choice: Int := 2

state choice {
    condition 1 {
        write("New game")
    }
    condition 2 {
        write("Settings")
    }
    condition default {
        write("Unknown choice")
    }
}
```

The first matching `condition` executes. If none matches, `condition default` executes. Because there is no automatic fall-through to the next option, `break` is not required.

## 9. Loops: `while`, `until`, and `for`

A loop does the same job over and over again.

### `while`: keep going as long as the condition is true

```ahd
count: Int := 1

while count <= 3 {
    write(count)
    count++
}
```

Output:

```text
1
2
3
```

`while` checks the condition first. If the condition is false initially, the body might not run at all.

### `until`: run at least once, then check the stop condition

```ahd
count: Int := 0

until count == 3 {
    count++
    write(count)
}
```

Output:

```text
1
2
3
```

`until` runs the body first, then checks the condition. Therefore, the body runs **at least once**. When the condition is `true`, the loop ends.

### `for`: use values from a list or a range sequentially

```ahd
for value in [10, 20, 30] {
    write(value)
}
```

and for a range of numbers:

```ahd
for value in between(1, 6, 2) {
    write(value)
}
```

Output:

```text
1
3
5
```

`between(start, stop)` includes the start but not the stop. The third value is the step amount. You can go backward with a negative step; a step of `0` raises a `DomainError`.

You generally don't need to write the type of the `for` variable:

```ahd
for value in [10, 20, 30] {
    write(value)
}
```

You can write it explicitly if you want:

```ahd
for value: Int in [10, 20, 30] {
    write(value)
}
```

The `for` variable is already considered belonging to that loop; `Local` is not prepended to it.

### Ending a loop early or skipping a turn

```ahd
for value in between(1, 10) {
    if value == 2 {
        continue
    }
    if value == 5 {
        break
    }
    write(value)
}
```

- `continue`: skips the rest of that turn.
- `break`: completely ends the loop.

> **Technical note:** `while` is a pre-check loop, `until` is a post-check loop. `between` doesn't pre-fill all numbers into a List; it produces values as needed. When a `for` loop begins on a List, iteration progresses over a snapshot.

## 10. Writing and calling functions

Instead of writing the same code over and over again, we can give it a name and run it whenever we want. This is called a **function**.

First, the smallest example:

```ahd
square: Function := (number: Int) -> Int {
    return number * number
}

write(square(5))
```

Output:

```text
25
```

Let's break down this line:

```text
square: Function := (number: Int) -> Int
^^^^^^^              ^^^^^^^^^^^     ^^^
function name         received value  returned type
```

`number` is the name of the value we send into the function. `return` gives back the result of the function.

### Multiple parameters

```ahd
add: Function := (a: Int, b: Int) -> Int {
    return a + b
}

write(add(2, 3))
```

You can also write parameters on separate lines:

```ahd
greet: Function := (
    name: String
    title: String := "Student"
) -> String {
    return "Hello {title} {name}"
}
```

Since we provided a default value for `title`, we don't have to provide it in the call:

```ahd
write(greet("Ali"))
write(greet(name: "Ayşe", title: "Dr"))
```

In a single call, either provide all values in order, or provide all of them by their names. Both of these are valid:

```ahd
greet("Ali")
greet(name: "Ayşe", title: "Dr")
```

But don't mix positional and named formats in the same call.

### A function that doesn't return a result: `Nothing`

A function that just does a job and doesn't return a value uses `Nothing`:

```ahd
sayHello: Function := (name: String) -> Nothing {
    write("Hello {name}")
}

sayHello("Ali")
```

You do not have to write `return` at the end of such a function. But if you want to exit early, you can use a bare `return`:

```ahd
showScore: Function := (score: Int) -> Nothing {
    if score < 0 {
        write("Invalid score")
        return
    }

    write("Score: {score}")
}
```

### A function calling itself

A function can call itself again. For example:

```ahd
countdown: Function := (n: Int) -> Nothing {
    if n <= 0 {
        write("Fire!")
        return
    }

    write(n)
    countdown(n - 1)
}

countdown(3)
```

This is called **recursion**. There must be a condition where it stops.

### Functions with the same name but different parameters

For more advanced use, you can use the same name with different parameter types:

```ahd
describe: Function := (value: Int) -> String {
    return "Int {value}"
}

describe: Overload Function := (value: Real) -> String {
    return "Real {value}"
}
```

AhdCode determines which version of the call it belongs to based on the parameters. Exact matches come first; safe `Int -> Real` widening can be used when needed. If it's ambiguous which version should be selected, the compiler will yield an error instead of guessing.

> **Technical note:** This selection is called *overload resolution*. Named Function declarations keep the `name: Function := (...) -> T { ... }` form and cannot be nested. For one expression, `lambda (x: Int) -> x > 0` creates an anonymous value of the same `Function` type. Lambda parameters are explicit, its return type is inferred, and it has no block or statements.

## 11. `Local` and `Global`

This section might seem a bit different at first glance. The logic is actually simple: AhdCode wants you to explicitly see **where** a variable lives.

### The function's parameter is already inside the function

```ahd
greet: Function := (name: String) -> Nothing {
    write(name)
}
```

Here `name` is the function's parameter. We don't separately define it as `Local`; it is already clear that it was sent into the function.

### If you are creating a new variable inside the function, write `Local`

```ahd
greet: Function := (name: String) -> Nothing {
    message: Local String := "Hello {name}"
    write(message)
}
```

`message` was created to be used solely inside this function. That's why we wrote `Local`.

The same rule applies to new variables you create inside inner blocks like `if` or `while`:

```ahd
if true {
    message: Local String := "This variable belongs to this block"
    write(message)
}
```

### Do not write `Local` for the `for` variable

`for` already creates its own variable:

```ahd
for value in [10, 20, 30] {
    write(value)
}
```

Therefore, do not write `for value: Local Int ...`.

### Using a variable from outside inside the function: `Global`

Now let's create a counter at the very top of the file:

```ahd
counter: Int := 0
```

If we want a function to modify this **same** counter, we say this explicitly inside the function with `Global`:

```ahd
counter: Int := 0

increase: Function := () -> Nothing {
    counter: Global Int
    counter++
}

increase()
increase()
write(counter)
```

Output:

```text
2
```

`counter: Global Int` does not create a new counter. It means "I will use the `counter` variable that is outside this function."

In short:

| Situation | What is written? |
|---|---|
| Function parameter | No extra `Local` required |
| New variable inside function / `if` / `while` | `Local` |
| `for` variable | `Local` is not written |
| Inner access to variable at file root | `Global` |

> **Technical note:** The region of code where a name can be used is called *scope*. In AhdCode, `Local` and `Global` are scope intents, not types.

## 12. Working with Lists (List)

You can use `List` to hold multiple values sequentially:

```ahd
numbers: List<Int> := [10, 20, 30]
write(numbers)
```

The index of the first element is `0`:

```ahd
write(numbers[0])  // 10
write(numbers[-1]) // 30
```

### Adding and removing elements

```ahd
numbers: List<Int> := [10, 20]
numbers.add(30)
write(numbers)

numbers.eject(1)
write(numbers)
```

Output:

```text
[10, 20, 30]
[10, 30]
```

`clear(numbers)` empties the entire list.

### Taking a portion of the list

```ahd
nums: List<Int> := [10, 20, 30, 40, 50]
part: List<Int> := nums[1:4]
write(part)
```

Output:

```text
[20, 30, 40]
```

This operation produces a new List.

### Sorting, reversing, and shuffling

```ahd
bring Math

values: List<Int> := [4, 1, 3, 2]
values.sort()
write(values)

values.reverse()
write(values)

Math.seed(42)
values.shuffle()
write(values)
```

`sort`, `reverse`, and `shuffle` modify the existing List in place.

### Searching

```ahd
data: List<Int> := [7, 8, 7, 9]
write(data.count(7)) // 2
write(data.index(8)) // 1
```

If `index()` cannot find the value it is looking for, it raises a `DomainError`.

### `map`, `filter`, and sorting by key

You can use `map` to transform every value in a list, and `filter` to select
some of them. A named Function remains useful for multi-step logic:

```ahd
double: Function := (value: Int) -> Int {
    return value * 2
}

isEven: Function := (value: Int) -> Bool {
    return value % 2 == 0
}

values: List<Int> := [3, -1, 4, -2]
write(values.map(double))
write(values.filter(isEven))
```

For a short single expression, pass a lambda directly:

```ahd
squares := values.map(lambda (value: Int) -> value^2)
positive := values.filter(lambda (value: Int) -> value > 0)
values.sort(lambda (value: Int) -> -value)
```

Lambda is not a separate type: it creates a value of the existing `Function`
type. Each parameter needs an explicit static type, the return type comes from
the expression, and ordinary strict typing/null-safety rules still apply. A
lambda has no block or statements.

If a lambda needs to use a variable from outside its own parameters, you must list it explicitly in square brackets before the parameters:

```ahd
minimum: Int := 50
passed := values.filter(lambda [#minimum] (score: Int) -> score >= minimum)
```

- `#` is for `Local` captures: it copies the value at the moment the lambda is created.
- `@` is for `Global` captures: it links to a live module variable.

No dependencies are inferred automatically; you must declare them.

`map` and `filter` do not alter the source List; they return a new List.

You can also provide a sorting key:

```ahd
absSort: Function := (value: Int) -> Int {
    return abs(value)
}

values: List<Int> := [3, -1, 4, -2]
values.sort(absSort)
write(values)
```

This sort is stable.

## 13. Reference Behavior

Look at this example:

```ahd
numbers: List<Int> := [10, 20, 30]
alias: List<Int> := numbers

alias[0] = 99
write(numbers)
```

Output:

```text
[99, 20, 30]
```

You might think, "I changed the `alias` variable, why did `numbers` change?" Because when we wrote `alias: List<Int> := numbers`, we did not create a second copy of the List. Both variables point to the **same List**.

We can explicitly check this:

```ahd
write(numbers same alias) // true
```

`same` asks whether two variables point to the same object.

`==` checks if their contents are equal:

```ahd
a: List<Int> := [1, 2]
b: List<Int> := [1, 2]

write(a == b)    // true: their contents are the same
write(a same b)  // false: two separate Lists
```

This distinction is especially important when modifying List, Pair, and Class objects.

> **Technical note:** Using multiple names for the same object is called *aliasing*, and this behavior is called *reference semantics*.

## 14. Working with Pair

When you want to pair a student's name with their grade, a product code with its price, or a setting with its value, you can use `Pair`.

```ahd
scores: Pair<String, Int> := {
    "Ali": 85
    "Ayşe": 92
}
```

Here `"Ali"` is the key, and `85` is the value bound to that key.

### Reading and modifying the value

```ahd
write(scores["Ali"])

scores["Ali"] = 90
scores["Veli"] = 78

write(scores["Ali"])
```

If you try to read a non-existent key, a `KeyError` occurs.

### Iterating over a Pair

```ahd
for name in scores {
    write("{name}: {scores[name]}")
}
```

`for` yields the Pair's keys in insertion order.

To delete a key use `eject(key)`, to empty the entire Pair use `clear(pair)`.

`Pair` exhibits reference behavior just like a List: if two variables point to the same Pair, changes made from one are visible from the other.

In v0.1, Pair keys can be `String`, `Int`, or `Bool`.

## 15. Constants (Constant)

Sometimes you do not want a value to be changed after it is created. `Constant` explicitly states this:

```ahd
locked: Constant List<Int> := [1, 2, 3]
```

It is now an error to attempt to change this List:

```ahd
// locked[0] = 99  // ERROR
```

`Constant` not only protects the outer List but also objects accessible from inside it. Meaning, if there are other List, Pair, or Class objects inside a constant structure, they are also frozen.

Therefore, even if you access the same object through another variable, attempting to change it can raise a `ConstantError` at runtime.

```ahd
values: List<Int> := [1, 2, 3]
locked: Constant List<Int> := values

// values.add(4) might raise a ConstantError at runtime;
// because the same List is now deep-frozen.
```

A `Constant` cannot initially be `null`.

> **Technical note:** Freezing the entire reachable object graph is called *deep freeze*.

## 16. Null safety

Sometimes it is normal for a value not to exist yet. For example, we search for a user but there is no record:

```text
"Ali" found    -> there is a User
no record      -> no value
```

In AhdCode, `null` means "there is no value here right now."

You can assign `null` to a regular variable, but AhdCode won't allow you to use it immediately:

```ahd
name: String := null
```

This tells us that the `name` variable can hold a `String` or `null`.

### Check before using

```ahd
message: String := null

if message == null {
    message = "ready"
}

if message != null {
    write(message.upper())
}
```

The compiler knows that there is indeed a String inside this block after the `message != null` check.

Writing the following without checking is invalid:

```ahd
message: String := null
// write(message.upper()) // ERROR: message could be null
```

### `null` alone does not specify its type

The following usage is invalid:

```ahd
// value := null // ERROR: variables must have explicit types
```

Because AhdCode wouldn't know if this is a `String`, a `User`, or some other type. Specify the type:

```ahd
value: String := null
```

If a function returns `User` but the result could be `null`:

```ahd
user: User := fetchUser()
```

### Null usage in collections

In collections, for example `List<User>`, the list itself could be `null`, or the list could be a valid object but the elements inside it could be `null`. This distinction is managed entirely by the same flow analysis (null refinement) logic in AhdCode.

> **Technical note:** The compiler gaining more precise information about a nullable value after a check is called *null refinement*. In the documentation, flow states can be referred to as `Null`, `MaybeNull`, and `NonNull`.

## 17. Classes (Class) and Attributes

Imagine you want to hold a student not just by their name, but by their name and number together. Instead of constantly carrying these two pieces of information in separate variables, you can define your own data structure. In AhdCode, `Class` is used for this.

### Our first Class

```ahd
Student: Class<> := {
    structure: Attributes := (
        name: String
        number: Int
    )
}
```

This definition says that every `Student` object will have `name` and `number` information.

Now let's create a student:

```ahd
student: Student := Student(name: "Ali", number: 42)
```

Inside the class, these properties are accessed via `attribute`. Let's add a method:

```ahd
Student: Class<> := {
    structure: Attributes := (
        name: String
        number: Constant Int
    )

    describe: Function := () -> String {
        return "#{attribute.number} {attribute.name}"
    }
}

student: Student := Student(name: "Ali", number: 42)
write(student.describe())
```

Output:

```text
#42 Ali
```

Because `number: Constant Int`, the number cannot be changed after the student is created.

If `Local` is prepended to a `structure` entry, it can only be used while the object is being created; it doesn't become a permanent attribute of the object. `Confidential` blocks normal access to the member from outside the class.

### Extending one Class from another

First let's define a general `Person`:

```ahd
Person: Class<> := {
    structure: Attributes := (
        name: String
    )

    describe: Function := () -> String {
        return "Person {attribute.name}"
    }
}
```

`Student` can inherit what `Person` has:

```ahd
Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )

    describe: Override Function := () -> String {
        return "{SuperClass.describe()} #{attribute.number}"
    }
}
```

- `Class<Person>`: `Student` is derived from `Person`.
- `SuperClass.attributes`: brings in the creation inputs of the superclass.
- `Override`: explicitly states that we knowingly modify a method from the superclass.
- `SuperClass.describe()`: calls the superclass's own version.

```ahd
student: Student := Student(name: "Ayşe", number: 7)
person: Person := student
write(person.describe())
```

Since the object is actually a `Student`, `Student.describe()` runs.

To query the actual type:

```ahd
if person is Student {
    write("This person is a student!")
}
```

To ask if it has a member, `has` can be used:

```ahd
write(person has name)
```

> **Technical note:** Holding a subclass object in a superclass type is called *upcasting*, and selecting the method to call based on the true object type is called *dynamic dispatch*. You do not have to memorize these terms on your first read.

### Class Protocol Methods: making `+`, `==`, `<`, and friends work for your own Class

You can make your own Class work with operators like `+`, `==`, and `<`. AhdCode does this with exactly ten reserved names, called **Class Protocol Methods**:

```text
CEqual CCompare CAdd CSubtract CMultiply CDivide CRemainder CPower CNegate CStr
```

These names carry special meaning only when they occupy a method slot inside a Class. Everywhere else they are ordinary identifiers -- the letter `C` itself is not reserved, so `Calculate`, `Create`, and `CWhatever` all remain perfectly normal names.

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

a: Vector2 := Vector2(x: 1.0, y: 2.0)
b: Vector2 := Vector2(x: 3.0, y: 4.0)

write(a + b)
write(-a)
write(a == b)
write(str(a))
```

Expected output:

```text
Vector2(4.0, 6.0)
Vector2(-1.0, -2.0)
false
Vector2(1.0, 2.0)
```

Quick summary:
- `==` and `!=` map to `CEqual` (`!=` is always the negation of `CEqual`'s result; there is no separate `CNotEqual`)
- `<`, `<=`, `>`, and `>=` all derive from one `CCompare` call (there are no separate `CLess` and similar names)
- `+ - * / % ^` map to `CAdd CSubtract CMultiply CDivide CRemainder CPower`
- Unary `-` maps to `CNegate`
- `str(object)` maps to `CStr`

Dispatch always looks at the **left-hand** operand: if `vector + 3` works, that does not mean `3 + vector` also works -- there is no reverse-operator rule. Inheritance and `Override` behave exactly like ordinary methods. See [Class Protocol Methods](PROTOCOLS.md) for the full reference.

## 18. Error handling (`attempt`, `except`, `ultimately`, and `toss`)

Some errors might not appear while writing the program, but while the program is running. For example, when waiting for a number from the user, they might type `abc`.

```ahd
age: Int := int(take("Age: "))
```

If the user types a number, there is no problem. If they type invalid text, a `DomainError` occurs. We can determine what the program will do in this situation.

### Catching an error: `attempt` and `except`

```ahd
attempt {
    age: Local Int := int(take("Age: "))
    write("Your age: {age}")
}
except DomainError as error {
    write("Please enter a valid integer.")
}
```

The code inside `attempt` is tried. If the specified error occurs, the appropriate `except` block runs.

You can write multiple `except` blocks for multiple error types:

```ahd
attempt {
    // operations that might raise errors
}
except DomainError as error {
    write("Number is invalid")
}
except IndexError as error {
    write("Index is invalid")
}
```

### The part that runs no matter what: `ultimately`

```ahd
attempt {
    write("Trying operation")
}
except DomainError as error {
    write("An error occurred")
}
ultimately {
    write("This line runs in every case")
}
```

### Raising an error on our own will: `toss`

```ahd
requirePositive: Function := (value: Int) -> Int {
    if value <= 0 {
        toss (DomainError("value must be positive"))
    }

    return value
}
```

Common error types in AhdCode include `DomainError`, `ValueError`, `IndexError`, `KeyError`, `OverflowError`, `DivisionByZeroError`, `NullError`, and `ConstantError`.

### Creating your own error type

When needed, you can derive a new error type from the `Error` class:

```ahd
InvalidAgeError: Class<Error> := {
    structure: Attributes := (
        message: String
    )
}
```

and then:

```ahd
attempt {
    age: Local Int := -5
    if age < 0 {
        toss (InvalidAgeError("Age cannot be negative"))
    }
}
except InvalidAgeError as error {
    write(error.message)
}
```

> **Technical note:** AhdCode runtime errors are modeled as normal catchable Class values.

## 19. Modules and `bring`

As a program grows, you won't want to write everything in a single file. You can put a task into a separate `.ahd` file and use it from another file. We can call this a **module**.

### Creating your own module

Imagine having two files in the same folder:

```text
main.ahd
Greeting.ahd
```

`Greeting.ahd`:

```ahd
greet: Function := (name: String) -> String {
    return "Hello from module, {name}"
}
```

`main.ahd`:

```ahd
from Greeting bring greet

write(greet("Ayşe"))
```

Output:

```text
Hello from module, Ayşe
```

### In what formats can I import a module?

As a namespace:

```ahd
bring Greeting
write(Greeting.greet("Ayşe"))
```

Only bringing the name you want:

```ahd
from Greeting bring greet
write(greet("Ayşe"))
```

Multiple names:

```ahd
from Greeting bring (
    greet
    farewell
)
```

All public names:

```ahd
from Greeting bring all
```

Imports that lead to the collision of identical names and circular module dependencies are compile-time errors.

### Giving a short name to a module

```ahd
bring Time as T

write(T.Calendar.isLeapYear(2028))
```

When you use `as T`, you use `T` instead of `Time` for this import.

This abbreviation is not used in type declarations. For example:

```ahd
bring Time as T
from Time bring DateTime

current: DateTime := T.now()
```

`T.DateTime` is not a valid type declaration syntax; import the type separately.

### A first look at File and Path

Sooner or later a program needs to remember something after it exits: a note, a small log, a file the user asked you to read. `Path` builds and inspects path *Strings*. `File` actually talks to the filesystem. They are two modules on purpose — a path is just text until `File` uses it.

Relative paths are resolved against the working folder of the program or REPL session (the folder where you ran `ahdcode`, not necessarily the folder that contains the `.ahd` file).

A tiny notes workflow:

```ahd
bring Path
bring File

notesDir: String := "notes"
path: String := Path.join([notesDir, "today.txt"])

if not File.exists(notesDir) {
    File.createDir(notesDir)
}

File.writeText(path, "Buy milk.\n")
write(File.readText(path))
write(Path.base(path))
write(File.exists(path))
```

Expected output:

```text
Buy milk.

today.txt
true
```

`Path.join` builds a path from parts. `Path.base` is the last component (`today.txt`); `Path.dir` and `Path.ext` are the other common inspections. `File.exists` returns `false` for a missing path instead of raising. `File.createDir` makes a directory. `File.writeText` overwrites (or creates) a UTF-8 text file; `File.readText` reads it back. `File.append` adds to the end without replacing the whole file.

When the path is missing *and* you asked to read it, that is a `FileError` (it inherits from `IOError`):

```ahd
bring File
from File bring FileError

attempt {
    write(File.readText("missing.txt"))
}
except FileError as error {
    write("File could not be read")
}
```

**Try it yourself:** Change the note text, run the program twice, then switch `writeText` to `append` and see both lines in the file.

The full list of operations lives in the [File and Path reference](FILESYSTEM.md).

### A first look at Regex

A regular expression is a tiny language for describing text shapes: “one or more digits”, “starts with a letter”, “an email-like token”. In AhdCode you compile that description once into a `Pattern`, then ask questions about Strings with it.

```ahd
bring Regex
from Regex bring Pattern

digits: Pattern := Regex.compile("[0-9]+")

write(digits.matches("order #482"))
write(digits.find("order #482, item #7"))
write(digits.findAll("order #482, item #7"))
write(digits.replace("room 12 and room 7", "N"))
write(digits.split("a12b34c"))
```

Expected output:

```text
true
482
["482", "7"]
room N and room N
["a", "b", "c"]
```

The Class produced by `Regex.compile` is called `Pattern`, not `Regex` — `bring Regex` already names the module, so the compiled-pattern type needs its own name (`from Regex bring Pattern`) when you write it as a type.

`matches` is true if the pattern occurs *anywhere* in the text (anchor it yourself with `^` / `$` when you need “the whole string”). `find` returns `String?` because there might be no match — check it before use, exactly like any other nullable value:

```ahd
bring Regex
from Regex bring Pattern

digits: Pattern := Regex.compile("[0-9]+")
found: String? := digits.find("no numbers here")
if found == null {
    write("nothing found")
}
```

`findAll` returns every match. `replace` returns a **new** String (the original text is unchanged). `split` cuts the text at each match. `groups` returns the captured groups of the first match as `List<String>?`.

An invalid pattern raises `RegexError` at compile-pattern time, not later during `find`:

```ahd
bring Regex
from Regex bring RegexError

attempt {
    Regex.compile("(unterminated")
}
except RegexError as error {
    write("could not compile: {error.message}")
}
```

**Try it yourself:** Compile a pattern that matches a simple three-letter word and test it against `"cat"` and `"catalog"`.

See [the Regex module reference](REGEX.md) for `groups` details and the full pattern syntax.

### A first look at CSV

CSV is the spreadsheet-as-text format: commas (or another delimiter) between cells, newlines between rows. `CSV` only transports **String**. It never decides that `"42"` is an `Int` or that `"2026-01-01"` is a date — you convert when *you* know what the column means.

There are two first-use shapes:

- **rows** — `List<List<String>>`: each inner list is one row, including the header
- **records** — `List<Pair<String, String>>`: each Pair uses the header as keys

```ahd
bring CSV

text: String := "name,age\nAli,42\nMerve,19\n"

rows: List<List<String>> := CSV.parse(text)
records: List<Pair<String, String>> := CSV.parseRecords(text)

write(rows[1][0])
write(records[0]["name"])

ages: List<Int> := []
for record in records {
    ages.add(int(record["age"]))
}
write(sum(ages))
```

Expected output:

```text
Ali
Ali
61
```

Malformed quoting, a bad delimiter, or a record that does not match the header raises `CSVError`. You can also `CSV.read` / `CSV.write` a file path, and `stringify` / `stringifyRecords` to go back to text.

**Try it yourself:** Add a third person to the CSV text and print that record's `age` after converting with `int(...)`.

See [the CSV module reference](CSV.md) for delimiters, files, and error details.

### A first look at Data tables

Once text is in, `Data` gives you a `Table`: named columns, rows you can filter and reshape. Every cell is still a `String`. Every transformation returns a **new** table — the one you already have does not change.

```ahd
bring Data
from Data bring Table

table: Table := Data.fromCSV("name,score,city\nAli,91,Adana\nAyse,78,Ankara\nDeniz,85,Adana\n")

write(table.rowCount())
write(table.columns())

passed: Table := table.filter(
    lambda (row: Pair<String, String>) -> int(row["score"]) >= 80
)
namesOnly: Table := passed.select(["name", "score"])

write(namesOnly.column("name"))
write(table.rowCount())
```

Expected output:

```text
3
["name", "score", "city"]
["Ali", "Deniz"]
3
```

The last line is the point: `table` still has three rows, because `filter` and `select` returned new tables. `drop(["city"])` would hide a column instead of keeping a list of names.

Notice `int(row["score"])`. Data never guesses that `"91"` is a number. A whole numeric column works the same way:

```ahd
bring Data
from Data bring Table

table: Table := Data.fromCSV("name,score\nAli,91\nAyse,78\n")
scores: List<Real> := table.column("score").map(
    lambda (value: String) -> real(value)
)
write(scores)
```

A realistic second step is “keep the people who passed, drop the city, write CSV text”:

```ahd
bring Data
from Data bring Table

table: Table := Data.fromCSV("name,score,city\nAli,91,Adana\nAyse,78,Ankara\n")
passed: Table := table.filter(
    lambda (row: Pair<String, String>) -> int(row["score"]) >= 80
)
namesOnly: Table := passed.select(["name", "score"])
write(namesOnly.toCSV())
```

Asking for a column that does not exist raises `DataError`:

```ahd
bring Data
from Data bring Table
from Data bring DataError

table: Table := Data.fromCSV("name,score\nAli,91\n")
attempt {
    write(table.column("grade"))
}
except DataError as error {
    write("no such column")
}
```

**Try it yourself:** Change the filter to `>= 90` and print how many rows remain in `passed` with `rowCount()`.

There is also `sort`, `rename`, `reverse`, `head`, `tail`, `transform`, `derive`, `unique`, `valueCounts`, and `groupBy`. See [the Data module reference](DATA.md).

## 20. Fundamentals module

To use some tools, you don't need to write any `bring`. These are directly available in an AhdCode program:

```text
write take str int real len clear between abs sum min max type id
```

We have already used most of them. A brief summary:

| Function | What it does |
|---|---|
| `write(value)` | prints value to the screen |
| `take()` / `take(prompt)` | reads a line of String from the user |
| `str(value)` | converts value to String |
| `int(...)` | converts suitable value to `Int` |
| `real(...)` | converts suitable value to `Real` |
| `len(value)` | returns length of String/List/Pair |
| `clear(collection)` | empties List or Pair in place |
| `between(...)` | produces a number range |
| `abs(number)` | returns absolute value |
| `sum(list)` | sums numbers in a list |
| `min(list)` / `max(list)` | finds the smallest / largest value |

Example:

```ahd
numbers: List<Int> := [3, -5, 10]

write(len(numbers))
write(sum(numbers))
write(min(numbers))
write(max(numbers))
write(abs(-8))
```

`sum` gives `0` or `0.0` for an empty List. `min` and `max` produce a `DomainError` on an empty List.

Because `clear` modifies the existing collection in place, other aliases pointing to the same collection will also see it emptied. `sum`, `min`, and `max` only read; they do not modify the List.

### `type(value)`: finding out a value's type

`type(value)` returns a value's AhdCode type as a `String`. It's especially handy while experimenting in the REPL:

```ahd
write(type(5))
write(type(5.0))
write(type("Ali"))
write(type(true))
write(type(null))
write(type([1, 2, 3]))
```

Expected output:

```text
Int
Real
String
Bool
Null
List<Int>
```

For a Class instance, `type` always reports the object's **actual runtime Class**, not the variable's declared type:

```ahd
Animal: Class<> := { structure: Attributes := (name: String) }
Dog: Class<Animal> := { structure: Attributes := (SuperClass.attributes) }

pet: Animal := Dog(name: "Rex")
write(type(pet)) // prints "Dog", not "Animal"
```

`type` does not return a reflection object of any kind -- only a plain String.

### `id(reference)`: runtime identity

`id(reference)` returns an opaque identity number (`Int`) for a List, Pair, or Class instance. It cannot be used on primitive values such as `Int`, `Real`, `String`, or `Bool`.

```ahd
a: List<Int> := [1, 2]
b: List<Int> := a
c: List<Int> := [1, 2]

write(id(a) == id(b))
write(id(a) == id(c))

a.add(3)
write(id(a) == id(b))
```

Expected output:

```text
true
false
true
```

`a` and `b` refer to the same List, so their identities match; `c` is a DIFFERENT List even though its contents look the same, so its identity differs. Mutating `a` (with `add`) never changes its identity.

This number is not a memory address, is not preserved between program runs, and is only meaningful during the current run/session. Use `same` for ordinary identity comparisons; `id` is mainly meant for debugging and logging.

## 21. Math module

Use the `Math` module for tools like square root, trigonometric functions, or random numbers:

```ahd
bring Math

write(Math.PI)
write(Math.sqrt(81))
write(Math.round(3.14159, 2))
```

Output:

```text
3.141592653589793
9.0
3.14
```

Commonly used ones:

| Item | What it does |
|---|---|
| `PI`, `E` | mathematical constants |
| `round`, `floor`, `ceil` | rounding operations |
| `sqrt`, `exp` | square root and $e^x$ |
| `sin`, `cos`, `tan` | trigonometric functions (radians) |
| `log`, `log10` | natural and base-10 logarithm |
| `seed`, `random`, `randomInt` | random number generation |

For exponentiation, use the language's `^` operator instead of `Math.pow`. `abs`, `sum`, `min`, and `max` are not in `Math`; they are directly available Fundamentals functions.

A tiny numerical example — hypotenuse:

```ahd
bring Math

a: Real := 3.0
b: Real := 4.0
c: Real := Math.sqrt((a ^ 2.0) + (b ^ 2.0))
write(c)
```

```text
5.0
```

### Generating random numbers

```ahd
bring Math

write(Math.randomInt(1, 6))
write(Math.random())
```

`randomInt(1, 6)` produces an integer in this range, including both `1` and `6`. `random()` is in the range `0.0 <= value < 1.0`.

If you want to obtain the exact same sequence of results again during testing, you can provide a seed:

```ahd
bring Math

Math.seed(42)
```

If the same seed is provided again, the same sequence of random numbers starts over. If no seed is provided, a new program run takes its initial state from the operating system.

`Math.random`, `Math.randomInt`, and `List.shuffle` use the same shared randomness state; every call advances the sequence. `randomInt(5, 5)` or an empty/single-element `shuffle` do not consume the randomness state.

> **Caution:** Do not use this random number generator for cryptography or security purposes.

**Try it yourself:** Call `Math.seed(1)`, print `randomInt(1, 6)` three times, run the program again, and confirm you get the same three numbers.

## 22. Time module

The `Time` module is used for date, time, and wait operations.

### Getting the current time

```ahd
bring Time
from Time bring DateTime

current: DateTime := Time.now()

write(current.year)
write(current.month)
write(current.day)
write(current.hour)
```

`Time.now()` gives your computer's local time. `Time.utc()` gives UTC.
`Time.timestamp()` is signed Unix milliseconds. AhdCode supports fixed minute
offsets but deliberately has no named/IANA timezone database.

A `DateTime` contains the following information:

```text
year  month  day  hour  minute  second  millisecond  weekday  offsetMinutes
```

`weekday` uses the value `1` for Monday, and `7` for Sunday. These fields are read-only.

`Time.fromTimestamp(milliseconds)` returns UTC. `dateTimeUTC(...)` constructs
UTC, while `dateTimeOffset(..., offsetMinutes: 180)` constructs a fixed-offset
value. `toUTC()`, `toLocal()`, and `toOffset(...)` preserve the instant;
`timestamp()` recovers its Unix milliseconds. Offsets must be -840..840.

### Creating a specific date

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

Output:

```text
2028-02-29 00:00:00
```

If `hour`, `minute`, `second`, and `millisecond` are not provided, they default to `0`. An invalid date raises a `ValueError`; for example, AhdCode rejects `2026-02-29` instead of silently changing it to another day.

### Comparing two times

```ahd
bring Time
from Time bring DateTime

morning: DateTime := Time.dateTime(year: 2026, month: 1, day: 1, hour: 9)
evening: DateTime := Time.dateTime(year: 2026, month: 1, day: 1, hour: 21)

write(morning.before(evening))
write(morning.after(evening))
write(morning.sameMoment(morning))
```

Instead of `<` and `>`, these readable methods are used for dates.

### Duration between two times

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

`Time.between(first, second)` subtracts the first time from the second. You can directly create a duration with `Time.duration(milliseconds: 1500)`.

### Calendar information

```ahd
bring Time

write(Time.Calendar.isLeapYear(2028))
write(Time.Calendar.daysInMonth(2028, 2))
write(Time.Calendar.weekday(2026, 8, 29))
```

### Delaying a task or measuring its duration

```ahd
bring Time

start: Real := Time.monotonic()
Time.sleep(500)
elapsed: Real := Time.monotonic() - start
write(elapsed >= 0.5)
```

`Time.sleep(...)` uses **milliseconds**, whereas `Time.monotonic()` uses **seconds**. `Time.monotonic()` is not a date; it is used to calculate the duration between two measurements. A negative `sleep` value raises a `ValueError`.

## 23. Statistics module

A list of numbers is not yet an answer. “What is typical?”, “how spread out is this?”, “what is the middle value?” — those questions belong to `Statistics`. The module does **not** draw pictures (that is `Plot`) and it does **not** read table cells as text (that is `Data`). You hand it a `List<Int>` or a `List<Real>` and it returns a number.

```ahd
bring Statistics

scores: List<Int> := [70, 80, 80, 90, 100]

write(Statistics.mean(scores))
write(Statistics.median(scores))
write(Statistics.mode(scores))
write(Statistics.stdDev(scores))
write(Statistics.min(scores))
write(Statistics.max(scores))
```

Expected output:

```text
84.0
80.0
80
10.198039027185569
70
100
```

What these mean in ordinary language:

- `mean` — arithmetic average (always `Real`, even for `List<Int>`)
- `median` — middle value after sorting; with an even count it averages the two middle values, so it is also always `Real` (`median([1, 2, 3, 4])` is `2.5`)
- `mode` — the most common value; keeps the element type (`Int` here)
- `stdDev` / `variance` — **population** spread (divide by `n`)
- `sampleStdDev` / `sampleVariance` — **sample** spread (divide by `n - 1`; need at least two values)
- `min` / `max` / `range` — smallest, largest, and `max - min`, keeping the element type
- `quantile(values, probability: p)` — value at probability `p` between `0.0` and `1.0`

`sum` here is the Statistics overload (empty list → `0` / `0.0`), not a different spelling of the Fundamentals `sum`. Empty input is otherwise a `StatisticsError` — not a silent zero:

```ahd
bring Statistics
from Statistics bring StatisticsError

empty: List<Int> := []
attempt {
    write(Statistics.mean(empty))
}
except StatisticsError as error {
    write("no mean of an empty list")
}
```

This does **not** compile, because Statistics never parses digit text:

```text
Statistics.mean(["10", "20"])
```

Convert first, the same way you already do with Data:

```ahd
bring Data
from Data bring Table
bring Statistics

table: Table := Data.fromCSV("name,score\nAli,91\nAyse,78\n")
numbers: List<Int> := table.column("score").map(
    lambda (value: String) -> int(value)
)
write(Statistics.mean(numbers))
```

**Try it yourself:** Compute `median` of `[4, 1, 3, 2]` and check that you get `2.5`.

See [the Statistics module reference](STATISTICS.md) for the exact formulas.

## 24. Plot module

`Plot` turns numeric lists into a picture file: `.png`, `.svg`, or `.pdf`. A beginner can produce a useful graph from this section alone. Like Statistics, Plot never parses `"91"` as a number.

```ahd
bring Plot
from Plot bring Chart

weeks: List<Int> := [1, 2, 3, 4]
scores: List<Int> := [62, 71, 68, 80]

chart: Chart := Plot.line(weeks, scores)
chart = chart.title("Exam scores")
chart = chart.xLabel("Week")
chart = chart.yLabel("Score")
chart.save("exam-scores.png")
```

`title`, `xLabel`, and `yLabel` each return a **new** `Chart`. The original value does not change, so you reassign (`chart = chart.title(...)`) the same way you reassign a Word document. `save` returns `Nothing` — call it as a statement, not `chart = chart.save(...)`.

A second common chart is a bar chart of named categories:

```ahd
bring Plot
from Plot bring Chart

subjects: List<String> := ["Math", "Physics", "History"]
averages: List<Real> := [88.0, 74.5, 91.0]

bars: Chart := Plot.bar(subjects, averages)
bars = bars.title("Class averages")
bars.save("averages.svg")
```

`Plot.scatter` is the same shape as `line` (x and y numeric lists) when you want points rather than a polyline. `List<Int>` and `List<Real>` may be mixed; a `Numeric` Vector is also accepted. Empty data raises `PlotError`.

**Try it yourself:** Change the title to your course name and save as `"exam-scores.pdf"` instead of `.png`.

See [the Plot module reference](PLOT.md) for histogram, box, error bars, and subplots.

## 25. Numeric module and Complex

`Numeric` is linear algebra: a `Vector` is an ordered list of numbers with a length, a `Matrix` is a grid with rows and columns. Elements are `Int` or `Real`. Operations return new values; they do not rewrite the Vector or Matrix you already have.

```ahd
bring Numeric
from Numeric bring Vector
from Numeric bring Matrix

v: Vector := Numeric.vector([1, 2, 3])
write(v.length())

m: Matrix := Numeric.matrix([[1, 2], [3, 4]])
write(m.determinant())
write(m.transpose().rowCount())
```

`determinant` is defined on a square matrix. A shape mismatch — adding two vectors of different length, or multiplying matrices whose inner sizes do not agree — raises `NumericError` instead of silently padding.

A `Vector` can be passed to `Plot.line` or `Plot.scatter` in place of a plain List, which is useful when the same numbers are both computed and drawn.

### Complex numbers

`Complex` is a **core type**, not a `Numeric` submodule. A value has a real part and an imaginary part, both `Real`. Write an uppercase `I` glued to a number:

```ahd
z: Complex := 2 + 3I
write(z)
write((z * z))
```

Expected output:

```text
2.0+3.0I
-5.0+12.0I
```

- `3I` is valid.
- `3i` is invalid.
- `3 I` (with a space) is invalid.

An `Int` or `Real` can be used where a `Complex` is required (safe widening). You can add, multiply, and divide Complex values, but they are unordered — `<` and `>` are not allowed.

See [the Numeric module reference](NUMERIC.md) for inverse, solve, and decompositions.

## 26. Latex module

If you want to produce a PDF from structured text — a short article, a report, a slide deck — `Latex` is the module that writes real LaTeX and compiles it **offline**. You do not install TeX Live. Staging the renderer is the one-time step in [Installation and your first program](#2-installation-and-your-first-program).

A document is a String you assemble, then wrap with `document`, then compile with `pdf`:

```ahd
bring Latex as L
from Latex bring LatexError

body: String := L.section("My First Document") +
    L.escape("Hello! Cost is $5, and 100% of the class passed.") +
    L.subsection("Energy") +
    L.equation("E = mc^2")

doc: String := L.document(body: body, title: "Notes", type: "Article")

attempt {
    L.pdf(doc, "output.pdf")
}
except LatexError as error {
    write("Could not create PDF: {error.message}")
}
```

Two different kinds of text:

- **`escape`** — ordinary language. Characters like `$ % & { }` become visible text instead of LaTeX commands.
- **`equation`** — raw math. It is *not* escaped, because `$` and `^` are the math.

`section` / `subsection` titles are escaped for you. `document`'s first real payload is named `body` (a positional `L.document(body)` still works). `type` may be `"Article"` (default), `"Report"`, or `"Beamer"`. Beamer themes are only `"Default"`, `"Madrid"`, and `"Warsaw"`.

`L.pdf(doc, "output.pdf", "tex")` also writes `output.tex` beside the PDF — useful when you want the exact source. `LatexError` is the compile/engine failure; invalid theme or type is a `ValueError`.

**Try it yourself:** Change `type` to `"Report"` and add a second `subsection`.

Do not copy every theorem/table/bibliography helper into a first program. See [the Latex module reference](LATEX.md).

## 27. Word module

Word builds **immutable** `.docx` documents. Microsoft Office is not required. Each method returns a **new** `Document`; the previous value stays as it was. `save` writes the file and returns `Nothing`, so you call it as a statement.

```ahd
bring Word
from Word bring Document

document: Document := Word.new()
document = document.heading("Lab report", 1)
document = document.paragraph("Prepared with AhdCode.", "center", true)
document = document.table(
    ["Sample", "pH"]
    [["A", "7.1"], ["B", "6.8"]]
)
document.save("lab-report.docx")
```

`heading(text, level)` uses heading levels `1`..`6`. `paragraph` is positional-only: text, then optional align (`"left"` / `"center"` / `"right"`), then optional `bold`, `italic`, `underline`. `table` takes header strings and a list of rows. `image(path)` embeds a PNG (including one you just saved from Plot) when you need a figure.

`Word.read(path)` loads a bounded semantic subset (paragraphs, headings, tables) back into a `Document`. That is useful for “open, add a heading, save again” — not for pixel-perfect round-trips with every Word feature.

**Try it yourself:** Add a second heading at level `2` and a short paragraph under it before `save`.

See [the Word module reference](WORD.md) for merges, image sizing, and `WordError`.

## 28. Excel module

Excel creates and reads real `.xlsx` workbooks without Microsoft Office. The mental model has three layers:

- **Workbook** — the file
- **Sheet** — a named grid inside the workbook
- **Cell** — one typed value at a 1-based row/column (`(1, 1)` is A1)

`Excel.new()` starts **empty** — there is no automatic `Sheet1`. Transformations return new values. A sheet you get with `book.sheet("Scores")` has no hidden back-pointer; after you edit it, put it back with `book.withSheet(sheet)`.

```ahd
bring Excel
from Excel bring Workbook
from Excel bring Sheet
from Excel bring Cell

book: Workbook := Excel.new().addSheet("Scores")
sheet: Sheet := book.sheet("Scores")
sheet = sheet.setRow(1, 1, [Excel.fromString("Name"), Excel.fromString("Score")])
sheet = sheet.setRow(2, 1, [Excel.fromString("Ali"), Excel.fromInt(91)])
sheet = sheet.setCell(3, 1, Excel.fromString("Merve"))
sheet = sheet.setCell(3, 2, Excel.fromInt(88))
sheet = sheet.setCell(4, 2, Excel.formula("=SUM(B2:B3)"))
book = book.withSheet(sheet)
book.save("scores.xlsx")
```

Cell values are explicit constructors: `fromString`, `fromInt`, `fromReal`, `fromBool`, `blank`, `formula`. A String that happens to start with `=` is still **text** unless you use `Excel.formula(...)`. AhdCode **stores** formulas; it does not evaluate them like Excel the application would.

Read the same file back:

```ahd
bring Excel
from Excel bring Workbook
from Excel bring Sheet
from Excel bring Cell

book: Workbook := Excel.new().addSheet("Scores")
sheet: Sheet := book.sheet("Scores")
sheet = sheet.setCell(1, 1, Excel.fromString("Ali"))
sheet = sheet.setCell(1, 2, Excel.fromInt(91))
book = book.withSheet(sheet)
book.save("scores.xlsx")

loaded: Workbook := Excel.read("scores.xlsx")
page: Sheet := loaded.sheet("Scores")
score: Cell := page.cell(1, 2)
write(score.kind())
write(score.int())
```

Wrong-kind access (`int()` on a String cell) raises `ExcelError`. Unknown sheet names and saving a workbook with zero sheets also raise `ExcelError`.

**Try it yourself:** Add a third student row, keep the formula covering the new cell, and save again.

Merge and style exist for later; they are not required for a first grade sheet. See [the Excel module reference](EXCEL.md).

## 29. PDF module

`PDF` builds an immutable `PDFDocument` (headings, paragraphs, tables, images,
page breaks) and renders it offline to a real `.pdf` file -- no Microsoft
Office, LibreOffice, or network access:

```ahd
bring PDF
from PDF bring PDFDocument

doc: PDFDocument := PDF.new()
doc = doc.heading("Quarterly Report", 1)
doc = doc.paragraph("Prepared offline, no Office dependency.")
doc = doc.table(["Region", "Q1", "Q2"], [["North", "10", "12"]])
doc.save("report.pdf")
```

Like `Document` and `Table`, every `PDFDocument` operation is positional-only
and returns a *new* `PDFDocument` -- the receiver never changes. `save()`
returns `Nothing`, so call it as a statement. Every String you pass in
(headings, paragraphs, table cells) is escaped before rendering, so
characters like `\ { } $ & #` always show up as ordinary text -- `PDF` has no
way to inject raw LaTeX.

`PDF` can also build a document directly from another module's own typed
document, with no manual copying:

```ahd
bring Word
from Word bring Document

wordDocument: Document := Word.new()
wordDocument = wordDocument.heading("Report", 1)
wordDocument = wordDocument.paragraph("Hello")

pdfFromWord: PDFDocument := PDF.fromWord(wordDocument)
pdfFromWord.save("report-from-word.pdf")
```

`.save()` uses the same offline renderer as `Latex.pdf` -- see [installation
and your first program](#2-installation-and-your-first-program) for the
one-time staging step. See the [PDF module reference](PDF.md) for image
sizing, page layout, and error details.

## 30. Archive module

`Archive` packages files into real, deterministic `.zip`, `.tar`, or
`.tar.gz` archives -- creation-only, and it needs no renderer or extra setup
at all, since it only uses the Go standard library:

```ahd
bring Archive
bring File

File.writeText("notes.txt", "pack me")
File.writeText("data.csv", "a,b\n1,2\n")

files: Pair<String, String> := {
    "notes.txt": "notes.txt"
    "data.csv": "data.csv"
}

Archive.zip("submission.zip", files)
write(File.exists("submission.zip"))
```

`Archive.zip`, `Archive.tar`, and `Archive.tarGzip` are the three creation
calls. The output extension must match (`.zip` / `.tar` / `.tar.gz`).
The key in `files` is the path *inside* the archive; the value is the source
file's path on disk. Unsafe entry paths (like `../secret`) and symlink
sources are rejected with an `ArchiveError` rather than silently skipped.
There is no extraction, listing, or read API — `Archive` only
creates archives. See the [Archive module reference](ARCHIVE.md).

### Putting it together

Two small workflows show how PDF, Excel, and Archive combine in real
programs.

**Report packaging** -- turn a Workbook into a PDF, then bundle both into one
ZIP:

```ahd
bring Excel
bring PDF
bring Archive
from Excel bring Workbook
from Excel bring Sheet

book: Workbook := Excel.new().addSheet("Scores")
sheet: Sheet := book.sheet("Scores")
sheet = sheet.setRow(1, 1, [Excel.fromString("Name"), Excel.fromInt(91)])
book = book.withSheet(sheet)
book.save("report.xlsx")

pdf := PDF.fromExcel(book)
pdf.save("report.pdf")

files: Pair<String, String> := {
    "report.xlsx": "report.xlsx"
    "report.pdf": "report.pdf"
}

Archive.zip("report.zip", files)
```

**Latex source sidecar** -- publish the compiled PDF together with its exact
LaTeX source:

```ahd
bring Latex as L

source: String := L.document(body: L.section("Findings"), title: "Report")
L.pdf(source, "article.pdf", "tex")
```

This produces both `article.pdf` and `article.tex` from one call; the third
argument only accepts `""` (the default, PDF-only) or `"tex"`. Pack them the
same way with `Archive.zip("article-bundle.zip", {"article.pdf": "article.pdf", "article.tex": "article.tex"})`.

## 31. JSON module

JSON reads and builds typed `JSONValue`s. There is no `Any` and no dynamic typing: a JSON value is exactly one of Null, Bool, Int, Real, String, Array, or Object, and the compiler knows which accessors are legal only after you ask `kind()` or pick a typed getter.

```ahd
bring JSON
from JSON bring JSONValue
from JSON bring JSONError

student: JSONValue := JSON.object({
    "name": JSON.fromString("Ali")
    "score": JSON.fromInt(91)
    "passed": JSON.fromBool(true)
})

text: String := JSON.stringify(student, true)
parsed: JSONValue := JSON.parse(text)
write(parsed.kind())

name: JSONValue? := parsed.get("name")
if name != null {
    write(name.string())
}

missing: JSONValue? := parsed.get("nickname")
if missing == null {
    write("no nickname")
}
```

Expected output (pretty-printed `stringify` may add spaces/newlines; `kind` and the two writes are stable):

```text
Object
Ali
no nickname
```

Constructors you will actually use: `fromString`, `fromInt`, `fromReal`, `fromBool`, `array`, `object`, and `nullValue` (not `JSON.null()`). `parse` reads a String; use a **raw** String for literal JSON so `{` is not interpolation: `JSON.parse(r'{"a":1}')`.

`get(key)` returns `JSONValue?` because a key can be absent. After you have a non-null `JSONValue`, `string()`, `int()`, `array()`, … raise `JSONError` when `kind()` does not match. That is the beginner trap: `"Ali"` is JSON String, so `.int()` fails.

```ahd
bring JSON
from JSON bring JSONError

attempt {
    write(JSON.fromString("Ali").int())
}
except JSONError as error {
    write("wrong kind")
}
```

**Try it yourself:** `parse` a small object, request a missing key, and print a default name when the result is `null`.

See [the JSON module reference](JSON.md).

## 32. XML module

XML is a small closed node model: every node is either an **Element** (a named tag, optional attributes, children) or **Text** (character data). There is no `Any`, no full DOM, and no comment/CDATA public types. You build nodes, wrap one Element as a document, stringify or parse.

```ahd
bring XML
from XML bring XMLNode
from XML bring XMLDocument

student: XMLNode := XML.element(
    "student"
    {"id": "42"}
    [
        XML.element("name", {}, [XML.text("Ali")])
        XML.element("score", {}, [XML.text("91")])
    ]
)
document: XMLDocument := XML.document(student)
write(XML.stringify(document, true))
write(student.kind())
idAttr: String? := student.attribute("id")
if idAttr != null {
    write(idAttr)
}
```

`XML.document(root)` requires an Element root — a Text node there raises `XMLError`. `kind()` works on every node. `name`, `attribute`, `children`, `elements` are for Elements. On a Text node, those accessors raise `XMLError`; use `text()` for the character data.

`XML.parse` reads a String back into an `XMLDocument` (exactly one root element). Attributes you construct are unqualified (no namespace).

**Try it yourself:** Add a `city` child element and print `stringify` again.

See [the XML module reference](XML.md).

## 33. Env module

An **environment variable** is a named String the operating system (or a `.env` file) gives your program: a port number, a data folder, a flag. `Env` always returns `String`. It never decides that `"8080"` is an `Int`.

```ahd
bring Env

found: String? := Env.get("PORT")
if found == null {
    write("PORT is not set")
}

port: Int := int(Env.getOr("PORT", "8080"))
write(port)
```

- `get` → `String?`. `null` means the name is absent.
- `getOr(name, fallback)` uses the fallback only when the name is absent. An explicitly empty value `""` is returned as empty, not replaced.
- `exists(name)` is the Bool test (`has` is a reserved keyword, so the method is not called `has`).

`Env.load(".env")` reads a file of `NAME=value` lines into the process environment. By default it does not override names that are already set (`override` defaults to `false`). A malformed `.env` raises `EnvError`. `Env.read(path)` returns the pairs without changing the process environment.

Do not hard-code secrets into source files; a `.env` file that you do not commit is the usual place for local configuration.

**Try it yourself:** Call `getOr` with a name you have not set and confirm you see the fallback, then `int(...)` that String.

See [the Env module reference](ENV.md).

## 34. Lists and KeyValue modules

These two **modules** are not the same thing as the List and Pair *types* you already use. `list.add(...)` changes the existing List. `Lists.chunk(...)` and `KeyValue.with(...)` return **new** values and leave the original alone.

```ahd
bring Lists
bring KeyValue

numbers: List<Int> := [1, 2, 3, 4, 5]
write(Lists.chunk(numbers, 2))
write(Lists.flatten([[1, 2], [3]]))
write(Lists.unique([1, 1, 2, 2, 3]))
write(Lists.valueCounts(["Math", "Physics", "Math"]))
write(numbers)
```

Expected output:

```text
[[1, 2], [3, 4], [5]]
[1, 2, 3]
[1, 2, 3]
{"Math": 2, "Physics": 1}
[1, 2, 3, 4, 5]
```

`numbers` is unchanged. `transpose` turns rows into columns (ragged input raises `ListsError`). `groupBy` groups elements by a callback key.

For Pair, `KeyValue` is the structural toolkit:

```ahd
bring KeyValue

record: Pair<String, String> := KeyValue.combine(["name", "score"], ["Ali", "91"])
updated: Pair<String, String> := KeyValue.with(record, "score", "95")
slim: Pair<String, String> := KeyValue.select(updated, ["name"])
write(KeyValue.keys(record))
write(updated)
write(record)
```

Expected output:

```text
["name", "score"]
{"name": "Ali", "score": "95"}
{"name": "Ali", "score": "91"}
```

`with` / `without` add or remove a key. `select` / `drop` keep or hide keys. `rename` and `mapValues` rewrite names or values. `merge` requires disjoint keys; `overlay` lets the second Pair win on collisions.

**Try it yourself:** `chunk` the list with size `3`, then print the original list again to prove it did not change.

See [Lists](LISTS.md) and [KeyValue](KEYVALUE.md) for every signature.

## 35. Code Formatter

Even if the code works, if everyone uses different spacing and line layouts, it becomes hard to read. The AhdCode formatter converts valid code to a common style:

```bash
ahdcode format hello.ahd
```

This command edits the file. To check only:

```bash
ahdcode format --check hello.ahd
```

When writing AhdCode, you don't have to manually adjust every comma or line break perfectly. For example, all three of these calls are valid:

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

The formatter keeps short structures on a single line and breaks long structures readably.

There is one important layout rule: the value following a `:=` or `=` sign **must start on the same line**.

Valid:

```ahd
values: List<Int> := [
    1
    2
    3
]
```

Invalid:

```text
values: List<Int> :=
    [1, 2, 3]
```

The formatter is idempotent; running it again on the same file produces no new changes.

## 36. Command line (CLI)

You can use AhdCode from the terminal with a few basic commands:

```text
ahdcode run file.ahd
```

Runs the program.

```text
ahdcode build file.ahd
```

Compiles the program into a standalone local executable.

```text
ahdcode format file.ahd
```

Formats the file according to the common style.

```text
ahdcode lsp
```

Starts the language server used by editors (VS Code / Antigravity). It talks over stdio only; optional `--stdio` is accepted and ignored. See the [language server guide](LSP.md) for the v0.2.2 feature set.

```text
ahdcode --help
ahdcode --version
```

Shows help and version information.

If you are a beginner, the command you will use most of the time will be `ahdcode run ...`.

## 37. Interactive shell (REPL)

You don't have to create a file every time you want to try something small. In the terminal, just run:

```bash
ahdcode
```

Starting it prints a version banner matching `ahdcode --version`, then the
REPL opens and you can try AhdCode commands one by one at the `ahd>` prompt:

```text
ahd> x: Int := 5
ahd> x = x + 1
ahd> x
6
```

The REPL acts like a **session**. It remembers the values you created in previous successful commands:

```text
ahd> name: String := "Ali"
ahd> write(name)
Ali
```

Making a mistake in one command does not erase your previously working state:

```text
ahd> x: Int := 5
ahd> x: Int := 7
error: duplicate declaration
ahd> x
5
```

The side effects of previous commands are not rerun. For example:

```text
ahd> write("one")
one
ahd> write("two")
two
```

`one` is not printed again on the second command.

`take()` waits for real user input even inside the REPL:

```text
ahd> name: String := take("Name: ")
Name: Ali
ahd> write(name)
Ali
```

Function and Class definitions, modules, List/Pair objects, and the Math randomness state are preserved throughout the session. Local modules and relative File paths are resolved relative to the folder where you started the `ahdcode` command.

The REPL is very useful when learning: you can quickly try an idea and see the result. For longer programs, using an `.ahd` file is more organized.

Building a `PDFDocument` or Latex source String works fine in the REPL, but
actually compiling to a `.pdf` does not: `Latex.pdf(...)`, `Latex.pdfFile(...)`,
and `PDFDocument.save(...)` all raise an error interactively, because
compiling invokes an external renderer that the persistent evaluator does
not support. Run those calls from a `.ahd` file instead. `Archive` has no
such limit -- it works fully in the REPL. See the [REPL reference](REPL.md)
for details.

## 38. Common beginner mistakes

Seeing an error message is a normal part of programming. Most errors simply tell you that the computer couldn't understand what you wanted. The following examples show common situations beginners encounter and how to fix them:

**1. Using `=` without declaring a variable**
- Wrong: `score = 10`
- Why: You must create a variable before assigning a value to it.
- Correct: `score: Int := 10`

**2. Defining the same variable twice (Duplicate declaration)**
- Wrong: `score: Int := 10 \n score: Int := 20`
- Why: `score` already exists in that block.
- Correct: `score: Int := 10 \n score = 20`

**3. Missing `Local` in inner blocks**
- Wrong: `if true { result: Int := 1 }`
- Why: New variables created in inner blocks like `if` require `Local`.
- Correct: `if true { result: Local Int := 1 }`

**4. Wrong use of `Local` in `for` loops**
- Wrong: `for item: Local Int in items`
- Why: The `for` variable is already local by design.
- Correct: `for item: Int in items`

**5. Missing `Global` for module-level variables**
- Wrong: `count: Int := 0 \n f: Function := () -> Nothing { count = 1 }`
- Why: To modify a variable at the module root, you must explicitly declare it with `Global`.
- Correct: `f: Function := () -> Nothing { count: Global Int \n count = 1 }`

**6. Truthiness (Automatic true/false assumption)**
- Wrong: `if 1 { write("Yes") }`
- Why: Conditions must be strictly of the `Bool` type.
- Correct: `if 1 > 0 { write("Yes") }`

**7. Unsafe `null` usage**
- Wrong: `name: String? := null \n write(name.upper())`
- Why: `name` can be null and this could lead to a crash. The compiler does not allow this.
- Correct: `if name != null { write(name.upper()) }`

**8. Mixing positional and named parameters**
- Wrong: `greet("Ali", title: "Dr")`
- Why: You must use either all arguments by their order or all arguments by their names.
- Correct: `greet(name: "Ali", title: "Dr")`

**9. Overload ambiguity**
- Wrong: Calling `f()` when you have both `f(Int)` and `f(Real)` functions with default values.
- Why: The compiler cannot guess which one you mean (ambiguous).
- Correct: Provide arguments in the call so the match is exact.

**10. Putting wrong type elements into a List**
- Wrong: `list: List<Int> := [1, 2.5]`
- Why: `2.5` is a `Real`, not an `Int`.
- Correct: Use `List<Real> := [1.0, 2.5]` or convert with `int(2.5)`.

**11. Trying to change a Constant**
- Wrong: `locked: Constant List<Int> := [1] \n locked[0] = 2`
- Why: Constants cannot be modified in place.
- Correct: If you intend to change it, remove the `Constant` keyword.

**12. Zero step in `between`**
- Wrong: `between(1, 10, 0)`
- Why: A loop with zero step would go on forever, so `between` rejects it.
- Correct: `between(1, 10, 1)`

**13. Invalid number conversion**
- Wrong: `int("3.14")`
- Why: The `int()` conversion is very strict and does not accept a decimal point.
- Correct: `int(real("3.14"))`

**14. Using `%` on `Real`**
- Wrong: `5.5 % 2.0`
- Why: The `%` (remainder) operator only works with `Int`.
- Correct: Use `Int` values, for example `5 % 2`.

**15. Division assignment to an `Int` variable**
- Wrong: `count: Int := 4 \n count /= 2`
- Why: The `/` operation returns a `Real`, and a `Real` cannot be directly assigned to an `Int` variable.
- Correct: `count = int(count / 2)`

**16. Missing module file**
- Wrong: Writing `bring Greeting` when there is no module file in the same folder. (`Math` is built-in, but your modules must have a file).
- Why: Modules must be in the same folder (sibling file) as the importing code.
- Correct: Make sure the `Greeting.ahd` file is in the same folder in your project.

**17. Wrong use of the `Override` keyword**
- Wrong: Writing `Override` on a method that does not exist in the parent Class.
- Why: `Override` strictly means "I am changing an existing method from the superclass."
- Correct: If you are adding a brand new method, delete the `Override` keyword.

**18. Invalid `return`**
- Wrong: Using `return "Done"` inside a function marked `-> Nothing`.
- Why: The function promised to return "nothing".
- Correct: Use a bare `return`.

**19. Trying to change a String in place**
- Wrong: `name[0] = 'B'`
- Why: Texts (Strings) are immutable.
- Correct: Use `replace` to swap a character, or create a new text.

**20. Expecting unseeded random numbers to be repeatable**
- Wrong: Expecting the code `Math.randomInt(1,6)` to remain the same on every run even though you didn't use `Math.seed(42)`.
- Why: Unseeded randomness uses OS entropy and cannot be repeated.
- Correct: Provide a seed value like `Math.seed(42)` before throwing the dice.

## 39. Small Projects

These small projects bring together what is taught in the guide. Try building them on your own!

1. **Grade Average Calculator**: Ask the user for 5 grades. Put them into a `List<Int>`. Filter out invalid grades (less than 0 or greater than 100). Print the average, minimum, and maximum value of the remaining grades, and finally whether the student passed (average >= 50).
2. **Simple Calculator**: Use `take()` to get two numbers and an operator (`+`, `-`, `*`, `/`). Use `state` on the operator to select the operation, and print the result. Manage the possibility of division by zero with `attempt`/`except`.
3. **Number Statistics**: Generate 100 random numbers with `Math.randomInt(1, 100)`. Count how many are odd and how many are even, and sort the list. Write a function that checks if a number is prime, and filter the list to show only primes.
4. **Word Analysis**: Ask the user to enter a sentence. Use `split(" ")` to separate the words. Find the word count, find the longest word, and create a `Pair<String, Int>` where each word is matched with its own length.
5. **Menu Program**: Make a small bank simulation using an `until` loop. Show a menu: 1. Deposit, 2. Withdraw, 3. Balance, 0. Exit. Store the balance in an `Int` and loop the program until the user enters 0.
6. **Student Registry with Classes**: Create a `Student` class and a `Course` class. Let the Course have a `List<Student>` inside it. Write a method to add a new student to the course, and another method to calculate the overall average grade of the course.
7. **Seeded Random Game**: Generate a "secret number" between 1 and 100 using `Math.seed(42)`. Ask the user to guess the number. Guide them with "higher" or "lower" until they guess correctly. Because a seed is used, the secret number will be the same every time you run the program—perfect for testing!

## 40. Exercises

Instead of immediately looking for full solutions, build each program in small steps.

### Beginner Level
1. Take the user's name and age; print how old they will be next year.
2. Read a Celsius value as `Real` and calculate its Fahrenheit equivalent (`C * 9/5 + 32`).
3. Read an `Int` and print whether it is odd or even using `%`.
4. Build a small menu that appears at least once using `until`, and stops when the user enters `0`.
5. Write a Function that trims the surrounding whitespace, converts a name to lowercase, then capitalizes the first character.
6. Create an empty `List<Int>`, add 3 numbers into it, and print `sum` and `len`.
7. Print a countdown by looping over `between(10, 0, -1)`.

### Intermediate Level
8. Read a sentence and replace all spaces inside it with underscores.
9. Show the `min` and `max` results in a List of grades; but check for the possibility of the List being empty first.
10. Link names to grades using a `Pair`, update a grade, and print all records.
11. Call the `Math.seed(42)` function, and throw a die ten times with `randomInt(1, 6)`. Verify that you get the exact same 10 results when you run the program again.
12. Use `map` to square all numbers in a List.
13. Write a `Student` Class with `name` and `Constant number` attributes, and a method that returns a summary.
14. Write a recursive function that calculates the factorial of a number.

### Advanced Level (Challenge)
15. Use `attempt` to prevent getting a `DomainError` when converting invalid number characters that might arise when the user types input with `int()`.
16. Sort texts in a `List<String>` by their lengths by calling `sort` with a `keyFunction`.
17. Create a `Shape` parent Class, then create a `Circle` subclass with an `Override` method that performs area calculation.
18. Write a function that takes a `String` as a parameter and returns a `Pair` counting how many times each character inside the text appears.
19. Use `break` and `continue` to find the first 5 even numbers within a very large range, but skip multiples of 3 with `continue`.
20. Create a module named `MathUtils.ahd` containing a function that calculates rectangle area, and use it from inside `main.ahd` by calling it with `bring`.

## 41. Solution Hints

1. The result of `take` is a String; use `int(...)` for age, and `+ 1` for the new age.
2. Break the formula into small parts; start with `real(take(...))` and use Real numbers.
3. `value % 2 == 0` produces a `Bool`.
4. Since `until` is post-check, you can place the menu text at the beginning of the body.
5. Try chaining `trim`, `lower`, and `capitalize` operations on a single `return` line.
6. Explicitly write the type for an empty `List<Int>`; add each entry with `add`.
7. Negative steps count backwards; remember that `between` does not include the stop value.
8. Use `String.replace(" ", "_")`.
9. `min` and `max` raise `DomainError` on an empty List; do a `len(grades) > 0` check first.
10. Use `Pair<String, Int>`; `for` on a Pair yields keys in insertion order.
11. Set the seed exactly once right before rolling the dice. Since bounds are inclusive, you can directly use `1, 6`.
12. The callback function you write should return `value * value`.
13. Take the class example from the beginning as a model for `structure: Attributes`.
14. The base condition of the recursive function should be `n <= 1` and should return 1.
15. Put the `int(take())` part inside `attempt` and catch `except DomainError`.
16. Your key function must take a `String` parameter and return `len(value)`.
17. For area, use `Math.PI * (radius ^ 2)`.
18. Loop over every letter in the String, check if it exists in the Pair, and increment its count by 1.
19. `if i % 3 == 0 { continue }`. `if count == 5 { break }`.
20. You can use `from MathUtils bring calculateArea`.

## 42. Next steps and technical docs

After finishing this guide, you can deepen your knowledge of the language details from these documents:

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
- [Class Protocol Methods](PROTOCOLS.md)
- [String API](STRING_API.md)
- [List API](LIST_API.md)
- [Math](MATH.md)
- [Time](TIME.md)
- [Statistics](STATISTICS.md)
- [Plot](PLOT.md)
- [Numeric](NUMERIC.md)
- [Latex](LATEX.md)
- [Word](WORD.md)
- [Excel](EXCEL.md)
- [PDF](PDF.md)
- [Archive](ARCHIVE.md)
- [JSON](JSON.md)
- [XML](XML.md)
- [Env](ENV.md)
- [Lists](LISTS.md)
- [KeyValue](KEYVALUE.md)
- [File and Path](FILESYSTEM.md)
- [Regex](REGEX.md)
- [CSV](CSV.md)
- [Data](DATA.md)
- [Diagnostics](DIAGNOSTICS.md)
- [CLI](CLI.md)
- [Formatter](FORMATTER.md)
- [REPL](REPL.md)
- [Language server](LSP.md)
- [Full v0.1 specification](../AHDCODE_LANGUAGE_SPEC_v0.1.md)

Check the [curated v0.1 examples](../examples/v0.1/README.md) folder for more working examples.
