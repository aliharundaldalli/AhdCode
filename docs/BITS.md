# Bits standard module

[English] · [Türkçe](BITS_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Math](MATH.md) · [Numeric](NUMERIC.md) · [Security](SECURITY.md)

`Bits` is the compiler-registered `builtin:Bits` module, introduced in AhdCode
v1.1.0. It is explicit and a sibling `Bits.ahd` cannot shadow it:

```ahd
bring Bits
from Bits bring BitsError
```

AhdCode's grammar has no bitwise operators — `and`, `or` and `not` are the
language's **logical** operators and `^` is exponentiation — so bit
manipulation is provided as named calls instead. Every operation is defined on
the language's only integer type, signed 64-bit `Int`, over its two's-complement
bit pattern.

## Public surface

```text
Bits.bitAnd(left: Int, right: Int)            -> Int
Bits.bitOr(left: Int, right: Int)             -> Int
Bits.bitXor(left: Int, right: Int)            -> Int
Bits.bitNot(value: Int)                       -> Int

Bits.shiftLeft(value: Int, distance: Int)          -> Int
Bits.shiftRight(value: Int, distance: Int)         -> Int
Bits.shiftRightUnsigned(value: Int, distance: Int) -> Int
Bits.rotateLeft(value: Int, distance: Int)         -> Int
Bits.rotateRight(value: Int, distance: Int)        -> Int

Bits.count(value: Int)                        -> Int
Bits.leadingZeros(value: Int)                 -> Int
Bits.trailingZeros(value: Int)                -> Int
```

The four logical operations are spelled `bitAnd`, `bitOr`, `bitXor` and
`bitNot` rather than `and`, `or`, `xor` and `not`, because three of those words
are reserved by the language itself.

## Shifts and rotations

`shiftRight` is **arithmetic**: the sign bit is replicated, so `-8` shifted
right by one is `-4`. `shiftRightUnsigned` fills with zeros instead, treating
the value as unsigned — that is the shift SHA-256 and similar algorithms
specify. Rotations move bits within the 64-bit word; no bit is lost.

```ahd
write(str(Bits.bitAnd(12, 10)))              // 8
write(str(Bits.shiftRight(-8, 1)))           // -4   (sign preserved)
write(str(Bits.shiftRightUnsigned(-1, 60)))  // 15   (zero filled)
write(str(Bits.rotateLeft(1, 63)))           // -9223372036854775808
```

A shift or rotate distance outside `0..63` raises `BitsError` rather than
silently producing zero, because "shift by 64" is almost always an arithmetic
mistake in the caller rather than a request for zero:

```ahd
attempt {
    write(str(Bits.shiftLeft(1, 64)))
} except BitsError as error {
    write(error.message)   // Bits shift distance must be between 0 and 63
}
```

## Counting bits

`count` is the population count — how many bits are set. `leadingZeros` and
`trailingZeros` return `64` for the value `0`, since every bit is a zero.

```ahd
write(str(Bits.count(255)))          // 8
write(str(Bits.count(-1)))           // 64  (every bit set)
write(str(Bits.leadingZeros(1)))     // 63
write(str(Bits.trailingZeros(8)))    // 3
write(str(Bits.trailingZeros(0)))    // 64
```

### Error type

```ahd
from Bits bring BitsError
```

`BitsError` derives from `Error`. Only the shift and rotate functions can
raise it; the logical operations and the counting functions are total.
