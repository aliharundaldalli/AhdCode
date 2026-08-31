# AhdCode v0.1 English Student Guide

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
- [Word module](#word-module)
- [JSON module](#json-module)
- [XML module](#xml-module)
- [Env module](#env-module)
- [27. Code Formatter](#27-code-formatter)
- [28. Command line (CLI)](#28-command-line-cli)
- [29. Interactive shell (REPL)](#29-interactive-shell-repl)
- [30. Common beginner mistakes](#30-common-beginner-mistakes)
- [31. Small Projects](#31-small-projects)
- [32. Exercises](#32-exercises)
- [33. Solution Hints](#33-solution-hints)
- [34. Next steps and technical docs](#34-next-steps-and-technical-docs)

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

AhdCode v0.1 is still an evolving release. You can run small command-line programs directly or compile them into standalone local applications.

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

AhdCode's built-in `Path` and `File` modules are also used via `bring`:

```ahd
bring Path
bring File

path: String := Path.join(["notes", "today.txt"])
File.createDir("notes")
File.writeText(path, "hello")
write(File.readText(path))
```

`Path` works with path strings. `File` performs file and folder operations. If you want to catch errors in file operations, you can additionally import the `FileError` type:

```ahd
from File bring FileError

attempt {
    write(File.readText("missing.txt"))
}
except FileError as error {
    write("File could not be read")
}
```

`FileError` inherits from the `IOError` class. Relative paths use the working folder of the program or REPL session.

### A first look at Regex

AhdCode's built-in `Regex` module compiles a pattern into a `Pattern` value, then lets you ask questions about a String using it:

```ahd
bring Regex
from Regex bring Pattern

digits: Pattern := Regex.compile("[0-9]+")

write(digits.matches("order #482"))       // true
write(digits.find("order #482, item #7")) // "482"
write(digits.findAll("order #482, item #7")) // ["482", "7"]
```

The Class produced by `Regex.compile` is called `Pattern`, not `Regex` -- `bring Regex` already names the module itself, so the compiled-pattern type needs its own name (`from Regex bring Pattern`) to be written as a type.

`find` returns `String?` because there might be no match at all, so check it before use, exactly like any other nullable value:

```ahd
found: String? := digits.find("no numbers here")
if found == null {
    write("nothing found")
}
```

An invalid pattern raises `RegexError`:

```ahd
from Regex bring RegexError

attempt {
    Regex.compile("(unterminated")
}
except RegexError as error {
    write("could not compile: {error.message}")
}
```

See [the Regex module reference](REGEX.md) for `replace`, `split`, and `groups`.

### A first look at CSV

`CSV` transports text as raw String rows or header-keyed String records:

```ahd
bring CSV

rows: List<List<String>> := CSV.parse("name,age\nAli,42\n")
records: List<Pair<String, String>> := CSV.parseRecords("name,age\nAli,42\n")
write(records[0]["name"])
```

CSV never infers numbers or dates. Malformed input and invalid record shapes
raise `CSVError`. See [the CSV module reference](CSV.md).

### A first look at Data tables

Once text is in, `Data` gives you a `Table` to work with. Every cell is still a
`String`, and every operation returns a **new** table instead of changing the
one you had:

```ahd
bring Data
from Data bring Table

table: Table := Data.fromCSV("name,score\nAli,91\nAyse,78\n")

write(table.rowCount())
write(table.columns())

passed: Table := table.filter(
    lambda (row: Pair<String, String>) -> int(row["score"]) >= 80
)

write(passed.column("name"))
write(table.rowCount())
```

=>

```text
2
["name", "score"]
["Ali"]
2
```

The last line is the point: `table` still has both rows, because `filter`
returned a new table.

Notice `int(row["score"])`. Data never guesses that `"91"` is a number, so you
convert when you need one — the same rule you already know from the rest of the
language. A whole numeric column works the same way:

```ahd
scores: List<Real> := table.column("score").map(
    lambda (value: String) -> real(value)
)
```

There is also `sort`, `select`, `drop`, `rename`, `reverse`, `head`, `tail`,
`transform`, `derive`, `unique`, `valueCounts`, and `groupBy`. Asking for a
column that does not exist raises `DataError`. See
[the Data module reference](DATA.md).

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

### Generating random numbers

```ahd
bring Math

write(Math.randomInt(1, 6))
write(Math.random())
```

`randomInt(1, 6)` produces an integer in this range, including both `1` and `6`. `random()` is in the range `0.0 <= value < 1.0`.

If you want to obtain the exact same sequence of results again during testing, you can provide a seed:

```ahd
Math.seed(42)
```

If the same seed is provided again, the same sequence of random numbers starts over. If no seed is provided, a new program run takes its initial state from the operating system.

`Math.random`, `Math.randomInt`, and `List.shuffle` use the same shared randomness state; every call advances the sequence. `randomInt(5, 5)` or an empty/single-element `shuffle` do not consume the randomness state.

> **Caution:** Do not use this random number generator for cryptography or security purposes.

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
morning: DateTime := Time.dateTime(year: 2026, month: 1, day: 1, hour: 9)
evening: DateTime := Time.dateTime(year: 2026, month: 1, day: 1, hour: 21)

write(morning.before(evening))
write(morning.after(evening))
write(morning.sameMoment(morning))
```

Instead of `<` and `>`, these readable methods are used for dates.

### Duration between two times

```ahd
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
write(Time.Calendar.isLeapYear(2028))
write(Time.Calendar.daysInMonth(2028, 2))
write(Time.Calendar.weekday(2026, 8, 29))
```

### Delaying a task or measuring its duration

```ahd
start: Real := Time.monotonic()
Time.sleep(500)
elapsed: Real := Time.monotonic() - start
write(elapsed >= 0.5)
```

`Time.sleep(...)` uses **milliseconds**, whereas `Time.monotonic()` uses **seconds**. `Time.monotonic()` is not a date; it is used to calculate the duration between two measurements. A negative `sleep` value raises a `ValueError`.

## 23. Statistics module

The `Statistics` module provides descriptive statistics for `List<Int>` or `List<Real>`. It does not draw charts (for visualization, see `Plot`).

```ahd
bring Statistics

scores: List<Int> := [70, 80, 90]
write(Statistics.mean(scores))
write(Statistics.median(scores))
```

## 24. Plot module

The `Plot` module creates charts from numeric data and saves them as images (`.png`, `.svg`, `.pdf`).

```ahd
bring Plot

chart := Plot.line([1, 2, 3], [2, 4, 3])
chart.save("chart.png")
```

You can customize the chart before saving:

```ahd
chart = chart.title("Growth").xLabel("Days")
```

Plot only accepts numeric types (`List<Int>`, `List<Real>`, or `Numeric` Vectors). It does not automatically parse text.

## 25. Numeric module and Complex

The `Numeric` module provides linear algebra operations (Vectors and Matrices).

```ahd
bring Numeric

v := Numeric.vector([1, 2, 3])
m := Numeric.matrix([[1, 2], [3, 4]])
write(m.determinant())
```

A `Numeric` Vector can also be passed directly to `Plot.line` or `Plot.scatter` instead of a plain List.

### Complex Numbers

AhdCode also supports `Complex` numbers (composed of `Real` components) as a core type. To create one, attach an uppercase `I` directly to a number:

```ahd
z := 2 + 3I
write(z)       // 2.0+3.0I
```

- `3I` is valid.
- `3i` is invalid.
- `3 I` (with a space) is invalid.

An `Int` or `Real` can be safely used where a `Complex` is required. `Complex` values can be added, multiplied, and divided, but they are unordered (you cannot use `<` or `>`).

## 26. Latex module

If you want to directly produce a PDF with AhdCode, you can use the `Latex` module. The module brings the necessary engine with its own installation.

A basic example:

```ahd
bring Latex as L
from Latex bring LatexError

doc: String := L.document(
    type: "Article",
    contents: L.section("My First Document") +
              L.escape("Hello! This is a regular text section.")
)

attempt {
    L.pdf(doc, "output.pdf")
}
except LatexError as e {
    write("Could not create PDF: {e.message}")
}
```

Beyond basic articles, you can use `document(type: "Report")` or `document(type: "Beamer")`. Beamer supports the bounded `"Default"`, `"Madrid"`, and `"Warsaw"` themes. The module also supports `date`, `margin`, `color`, `figure`, `image`, `theorem`, `ref`, `cite`, and `bibliography`. For more advanced capabilities, see the [Latex module reference](LATEX.md).

## Word module

Word builds immutable DOCX documents without requiring Microsoft Office:

```ahd
bring Word
from Word bring Document

document: Document := Word.new()
document = document.heading("Class report", 1)
document = document.paragraph("Prepared with AhdCode.", "center", true)
document = document.table(["Name", "Score"], [["Ali", "91"]])
document.save("class-report.docx")
```

`save()` returns `Nothing`, so call it as a statement rather than assigning
its result. Document methods are positional-only and return a new Document.
Word can also embed Plot-generated PNG files and read text, headings, and
tables from a bounded semantic DOCX subset. See the [Word module
reference](WORD.md).

## JSON module

JSON reads and builds typed `JSONValue`s — no `Any`, no dynamic typing:

```ahd
bring JSON
from JSON bring JSONValue

student: JSONValue := JSON.object({
    "name": JSON.fromString("Ali")
    "score": JSON.fromInt(91)
})
text: String := JSON.stringify(student, true)
parsed: JSONValue := JSON.parse(text)
name: JSONValue? := parsed.get("name")
if name != null {
    write(name.string())
}
```

Every accessor (`int()`, `string()`, `array()`, ...) raises `JSONError` if
the value's `kind()` doesn't match; `get(key)` returns `JSONValue?` because a
key can genuinely be absent. Write literal JSON braces with a raw String
(`r'{"a":1}'`) so AhdCode doesn't read them as string interpolation. See the
[JSON module reference](JSON.md).

## XML module

XML builds and reads a small `Element`/`Text` node model:

```ahd
bring XML
from XML bring XMLNode

root: XMLNode := XML.element("student", {"id": "42"}, [XML.text("Ali")])
document := XML.document(root)
write(XML.stringify(document, true))
```

`XML.document(root)` requires an `Element` root. Every `XMLNode` accessor
except `kind()` and `text()` raises `XMLError` on a `Text` node. See the
[XML module reference](XML.md).

## Env module

Env reads process environment variables and `.env` files, always as
`String`:

```ahd
bring Env

Env.load(".env")
port: Int := int(Env.getOr("PORT", "8080"))
```

`Env.get(name)` returns `String?` so absence and an explicit empty value stay
distinguishable. See the [Env module reference](ENV.md).

## 24. Code Formatter

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

## 25. Command line (CLI)

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
ahdcode --help
ahdcode --version
```

Shows help and version information.

If you are a beginner, the command you will use most of the time will be `ahdcode run ...`.

## 26. Interactive shell (REPL)

You don't have to create a file every time you want to try something small. In the terminal, just run:

```bash
ahdcode
```

The REPL opens, and you can try AhdCode commands one by one:

```text
> x: Int := 5
> x = x + 1
> x
6
```

The REPL acts like a **session**. It remembers the values you created in previous successful commands:

```text
> name: String := "Ali"
> write(name)
Ali
```

Making a mistake in one command does not erase your previously working state:

```text
> x: Int := 5
> x: Int := 7
error: duplicate declaration
> x
5
```

The side effects of previous commands are not rerun. For example:

```text
> write("one")
one
> write("two")
two
```

`one` is not printed again on the second command.

`take()` waits for real user input even inside the REPL:

```text
> name: String := take("Name: ")
Name: Ali
> write(name)
Ali
```

Function and Class definitions, modules, List/Pair objects, and the Math randomness state are preserved throughout the session. Local modules and relative File paths are resolved relative to the folder where you started the `ahdcode` command.

The REPL is very useful when learning: you can quickly try an idea and see the result. For longer programs, using an `.ahd` file is more organized.

## 30. Common beginner mistakes

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

## 31. Small Projects

These small projects bring together what is taught in the guide. Try building them on your own!

1. **Grade Average Calculator**: Ask the user for 5 grades. Put them into a `List<Int>`. Filter out invalid grades (less than 0 or greater than 100). Print the average, minimum, and maximum value of the remaining grades, and finally whether the student passed (average >= 50).
2. **Simple Calculator**: Use `take()` to get two numbers and an operator (`+`, `-`, `*`, `/`). Use `state` on the operator to select the operation, and print the result. Manage the possibility of division by zero with `attempt`/`except`.
3. **Number Statistics**: Generate 100 random numbers with `Math.randomInt(1, 100)`. Count how many are odd and how many are even, and sort the list. Write a function that checks if a number is prime, and filter the list to show only primes.
4. **Word Analysis**: Ask the user to enter a sentence. Use `split(" ")` to separate the words. Find the word count, find the longest word, and create a `Pair<String, Int>` where each word is matched with its own length.
5. **Menu Program**: Make a small bank simulation using an `until` loop. Show a menu: 1. Deposit, 2. Withdraw, 3. Balance, 0. Exit. Store the balance in an `Int` and loop the program until the user enters 0.
6. **Student Registry with Classes**: Create a `Student` class and a `Course` class. Let the Course have a `List<Student>` inside it. Write a method to add a new student to the course, and another method to calculate the overall average grade of the course.
7. **Seeded Random Game**: Generate a "secret number" between 1 and 100 using `Math.seed(42)`. Ask the user to guess the number. Guide them with "higher" or "lower" until they guess correctly. Because a seed is used, the secret number will be the same every time you run the program—perfect for testing!

## 32. Exercises

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

## 33. Solution Hints

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

## 34. Next steps and technical docs

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
- [File and Path](FILESYSTEM.md)
- [Regex](REGEX.md)
- [CSV](CSV.md)
- [Data](DATA.md)
- [Diagnostics](DIAGNOSTICS.md)
- [CLI](CLI.md)
- [Formatter](FORMATTER.md)
- [REPL](REPL.md)
- [Full v0.1 specification](../AHDCODE_LANGUAGE_SPEC_v0.1.md)

Check the [curated v0.1 examples](../examples/v0.1/README.md) folder for more working examples.
