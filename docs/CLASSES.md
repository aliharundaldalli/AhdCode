# Classes

[Back to README](../README.md) · [Functions](FUNCTIONS.md) · [Class Protocol Methods](PROTOCOLS.md)

## Structure and construction

```ahd
Person: Class<> := {
    structure: Attributes := (
        name: String
        age: Int
    )

    describe: Function := (
    ) -> String {
        return "{attribute.name}, {attribute.age}"
    }
}

person: Person := Person(name: "Ali", age: 28)
write(person.describe())
```

Construction uses ordinary call syntax. The parser does not need a separate
constructor-call form.

Non-`Local` structure entries become attributes. `Constant` makes an attribute
immutable and deep-freezes a reference value. `Local` keeps a constructor input
available to the structure body without creating an attribute.

```ahd
Account: Class<> := {
    structure: Attributes := (
        id: Constant Int
        password: Local String
    ) {
        attribute.passwordHint: Confidential String := password[0]
    }
}
```

## Inheritance and overrides

```ahd
Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Constant Int
    )

    describe: Override Function := (
    ) -> String {
        base: Local String := SuperClass.describe()
        return "{base}, #{attribute.number}"
    }
}
```

v0.1 has one direct superclass. Replacing an inherited method requires
`Override`. `SuperClass.describe()` calls the direct-super implementation.

## Confidential members

`Confidential` members are accessible in the defining class and subclasses,
but not through ordinary external access. Module-root Confidential declarations
are not public module exports.

## Operator behavior

A Class defines `==`, ordering (`<`/`<=`/`>`/`>=`), arithmetic, unary `-`,
and `str()` behavior through ten exact, compiler-recognized method names --
`CEqual`, `CCompare`, `CAdd`, `CSubtract`, `CMultiply`, `CDivide`,
`CRemainder`, `CPower`, `CNegate`, `CStr`. See
[Class Protocol Methods](PROTOCOLS.md) for the required signatures and
operator semantics. No other name, including one that merely starts with
`C`, carries special meaning.
