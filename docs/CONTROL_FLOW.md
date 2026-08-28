# Control flow

[Back to README](../README.md)

## Conditions

```ahd
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

Conditions require `Bool`.

## `while` and `until`

`while` checks before each iteration:

```ahd
count: Int := 0
while count < 3 {
    write(count)
    count++
}
```

`until` is a post-check loop. Its body runs at least once, then execution stops
when the condition becomes true:

```ahd
count: Int := 0
until count == 3 {
    count++
    write(count)
}
```

## `for` and `between`

```ahd
for value in [10, 20, 30] {
    write(value)
}

for value: Int in between(1, 6, 2) {
    write(value)
}
```

`between` is lazy, excludes its stop, supports negative steps, and raises
`DomainError` for a zero step. List, String, and Pair iteration uses a shallow
snapshot taken when the loop begins. Pair iteration yields keys.

`break` and `continue` affect the nearest loop.

## `state` / `condition`

```ahd
state status {
    condition "active" {
        write("Active")
    }
    condition default {
        write("Unknown")
    }
}
```

There is no fall-through and no `break` is needed.
