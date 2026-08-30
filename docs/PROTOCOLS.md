# Class Protocol Methods

[English] · [Türkçe](PROTOCOLS_TR.md)

[Back to README](../README.md) · [Classes](CLASSES.md) · [Fundamentals](FUNDAMENTALS.md)

AhdCode lets a Class define operator behavior through a small, closed set of
exact method names: **Class Protocol Methods**. This is a deliberately narrow
compiler extension surface, not a general reflection or metaprogramming
mechanism, and not Python's magic-method system. There is no
double-underscore naming convention, no `__eq__`/`__repr__`/`__radd__`-style
family, and no attempt to give every language mechanism (construction,
iteration, indexing, attribute access, calling) its own protocol name.

There are exactly ten Class Protocol Methods in v0.1.8:

```text
CEqual CCompare
CAdd CSubtract CMultiply CDivide CRemainder CPower
CNegate CStr
```

**Only these ten exact names are special, and only inside a Class method
slot.** The letter `C` itself is not reserved. `Calculate`, `Create`,
`CWhatever`, and `CustomMethod` are ordinary names with no special meaning,
including inside a Class. At module scope, even the exact names above (for
example a module-root `CAdd: Function := ...`) remain ordinary Functions --
protocol meaning exists only when one of the ten names occupies a Class
method slot.

Because the name is reserved there, a non-Function member that reuses one is
rejected at compile time rather than silently becoming an ordinary field:

```ahd
Bad: Class<> := {
    structure: Attributes := (x: Int)
    CAdd: Int := 5   // error: CAdd is a reserved Class Protocol Method slot
}
```

## Declaration syntax is unchanged

A Class Protocol Method is written with the same Function/method syntax as
any other method. There is no new declaration form:

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

Class syntax itself (`Person: Class<> := { ... }`) and Function declaration
syntax (`name: Function := (...) -> T { ... }`) do not change.

## Required signatures

| Protocol | Explicit parameters | Return type |
|---|---|---|
| `CEqual` | exactly 1 | `Bool` |
| `CCompare` | exactly 1 | `Int` |
| `CAdd`, `CSubtract`, `CMultiply`, `CDivide`, `CRemainder`, `CPower` | exactly 1 | the operator's result type (not required to be the Class itself) |
| `CNegate` | 0 | the operator's result type |
| `CStr` | 0 | `String` |

A malformed protocol declaration is a normal semantic diagnostic, never a
runtime panic: wrong arity, wrong return type, or a reserved name reused for
a non-Function are all caught during compilation.

The arithmetic protocols and `CNegate` do not have to return the containing
Class. A future `Matrix * Vector -> Vector` is legitimate; the operator
expression's static type is simply the selected method's declared return
type.

## Operator mapping

| Operator | Protocol |
|---|---|
| `==` | `CEqual` |
| `!=` | logical negation of the same `CEqual` call -- there is no `CNotEqual` |
| `<`, `<=`, `>`, `>=` | all four derive from one `CCompare` call |
| `+` | `CAdd` |
| `-` (binary) | `CSubtract` |
| `*` | `CMultiply` |
| `/` | `CDivide` |
| `%` | `CRemainder` |
| `^` | `CPower` |
| `-` (unary) | `CNegate` |

### Equality: `==` and `!=`

`a == b` calls `a.CEqual(b)` when `a`'s Class provides it. `a != b` is always
`!(a.CEqual(b))` -- the same call, negated -- so `==` and `!=` can never
disagree through two independently implemented methods. `CEqual` is never
derived from `CCompare`, and `CCompare` is never derived from `CEqual`: a
type can be meaningfully equality-comparable without a natural ordering, and
vice versa.

If a Class provides no `CEqual`, `==`/`!=` keep their pre-v0.1.8 behavior
(reference equality, the same as `same`) unchanged.

### Ordering: `<`, `<=`, `>`, `>=`

There is no `CLess`, `CGreater`, `CLessEqual`, or `CGreaterEqual`. All four
comparison operators are defined in terms of one `CCompare` call, evaluated
**exactly once** per expression:

```text
a <  b   =>  a.CCompare(b) <  0
a <= b   =>  a.CCompare(b) <= 0
a >  b   =>  a.CCompare(b) >  0
a >= b   =>  a.CCompare(b) >= 0
```

Any negative, zero, or positive `Int` is a valid `CCompare` result -- it is
not restricted to exactly `-1`, `0`, `1`:

```ahd
Score: Class<> := {
    structure: Attributes := (value: Int)
    CCompare: Function := (
        other: Score
    ) -> Int {
        return attribute.value - other.value   // any sign is fine
    }
}
```

### Arithmetic and unary negation

`+`, `-`, `*`, `/`, `%`, and `^` map to `CAdd`, `CSubtract`, `CMultiply`,
`CDivide`, `CRemainder`, and `CPower`. Unary `-` maps to `CNegate`.

Dispatch is **left-operand based**. There are no reverse-operator protocols
(`CReverseAdd`/`CRAdd`/etc.):

```ahd
value + 3   // works if value's Class has CAdd(Int)
3 + value   // does NOT try value's CAdd -- this is an ordinary type error
            // unless it is independently valid under the primitive rules
```

### Overloading

Class Protocol Methods reuse the ordinary method overload mechanism -- there
is no second overload system. A Class may declare more than one `CAdd`
overload as long as the existing overload rules allow it:

```ahd
CAdd: Function := (
    other: Vector2
) -> Vector2 { ... }

CAdd: Overload Function := (
    scalar: Real
) -> Vector2 { ... }
```

Operator resolution uses the same static overload-resolution rules as an
ordinary call. Ambiguity is a compile-time error, exactly as it already is
for overloaded method calls.

### Inheritance and overriding

A protocol method is inherited and overridden exactly like any other method.
`Override` is required to replace an inherited protocol method, and operator
dispatch uses the same dynamic dispatch as an ordinary method call, so a more
derived override runs when the runtime object is a subclass:

```ahd
Animal: Class<> := {
    structure: Attributes := (name: String)
    CStr: Function := () -> String { return "Animal({attribute.name})" }
}
Dog: Class<Animal> := {
    structure: Attributes := (SuperClass.attributes)
    CStr: Override Function := () -> String { return "Dog({attribute.name})" }
}

pet: Animal := Dog(name: "Rex")
write(str(pet))   // "Dog(Rex)" -- dynamic dispatch, not the static Animal type
```

### Compound assignment

`+=`, `-=`, `*=`, `/=`, `%=`, and `^=` on a Class target reuse the matching
arithmetic protocol -- conceptually `a += b` behaves like `a = a + b`,
subject to the normal assignment-compatibility rules. The target's receiver
is evaluated exactly once, exactly like every other compound assignment;
there is no separate in-place protocol (no `CIAdd`-style method). `++` and
`--` are unrelated and are not extended to Class values in v0.1.8.

## Nullability

Protocol dispatch never weakens v0.1.7 null safety. If the left operand is
nullable, it must be narrowed to `NonNull` by ordinary flow analysis before a
protocol method can be invoked on it -- exactly the same requirement as any
other method call:

```ahd
user: User?

user + other   // still invalid unless user is narrowed to non-null
```

The right-hand argument uses its own declared parameter type and
nullability normally; a protocol may explicitly accept a nullable argument
(`CEqual(other: User?) -> Bool`) if that is what the Class wants, but nothing
about that acceptance is inferred automatically.

## What this is not

- Not Python magic methods: no `__eq__`, `__lt__`, `__repr__`,
  `__radd__`/`__iadd__`, and no double-underscore convention at all.
- Not a general reflection or metaprogramming system: there is no
  `CStructure`, `CConstructor`, `CGetAttribute`, `CSetAttribute`, `CIterator`,
  `CLength`, `CCall`, `CEnter`, or `CExit`. Only the ten names above carry
  protocol meaning.
- No reverse operators, no in-place protocols, no `CRepr`/`CAbs`.
- `str(value)` is the only conversion `CStr` participates in; there is no
  separate developer/debug string protocol.

Everything else beginning with `C` -- or beginning with any other letter --
remains a perfectly ordinary identifier.
