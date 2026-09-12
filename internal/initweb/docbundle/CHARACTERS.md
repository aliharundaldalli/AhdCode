# Characters standard module

[English] · Türkçe

[Back to README](README.md) · [Modules](MODULES.md) · [String API](STRING_API.md) · [Regex](REGEX.md)

`Characters` is the compiler-registered `builtin:Characters` module, introduced
in AhdCode v1.2.0. It is the one explicit place for character and code-point
operations. A sibling `Characters.ahd` cannot shadow it.

```ahd
bring Characters
from Characters bring CharactersError
```

## The unit is a Unicode code point

A character in this module is one Unicode **code point** (a Unicode scalar
value). It is never a UTF-8 byte, and it is not an extended grapheme cluster.

- AhdCode has **no `Char` type**. One character is an ordinary `String` that
  holds exactly one code point.
- `"A"`, `"ş"`, `"ğ"`, `"İ"`, and `"😊"` are each one code point.
- A glyph that a reader sees as one symbol can be several code points. `"e"`
  followed by U+0301 COMBINING ACUTE ACCENT displays as `é` but is **two**
  characters here. Characters does not segment text into grapheme clusters.

This is the same unit `String` already uses: `len(text)`, `text[index]`, and
`for character in text` all count and index code points. Characters adds the
operations String does not have — conversion to a List, code-point values, and
Unicode classification — without a second model of text.

## Surface

```text
list(text: String)            -> List<String>
count(text: String)           -> Int

codePoint(character: String)  -> Int
fromCodePoint(value: Int)     -> String

isLetter(character: String)       -> Bool
isDigit(character: String)        -> Bool
isWhitespace(character: String)   -> Bool
isUpper(character: String)        -> Bool
isLower(character: String)        -> Bool
isAlphaNumeric(character: String) -> Bool
isPunctuation(character: String)  -> Bool
isSymbol(character: String)       -> Bool

CharactersError
```

Every argument is `NonNull`. The argument types are checked at compile time:
`Characters.codePoint(65)` or `Characters.fromCodePoint("A")` is the ordinary
`SEM004` type mismatch, not a runtime error.

## list and count

`list(text)` returns each code point of `text` as its own one-character String,
in source order. `text` is unchanged.

```ahd
bring Characters

write(Characters.list("Aş😊"))
write(Characters.count("Aş😊"))
```

=>

```text
["A", "ş", "😊"]
3
```

`count(text)` is the number of code points. It always equals `len(text)`;
`Characters.count` exists so character-oriented code reads as such. Neither one
counts UTF-8 bytes: `"çğıöşü"` is six characters and twelve bytes.

An empty String gives `[]` and `0`. Newlines, tabs, and spaces are characters
like any other.

## codePoint and fromCodePoint

`codePoint(character)` returns the Unicode scalar value of a String that holds
exactly one code point. `fromCodePoint(value)` is its inverse.

```ahd
bring Characters

write(Characters.codePoint("A"))
write(Characters.codePoint("ş"))
write(Characters.fromCodePoint(128522))
```

=>

```text
65
351
😊
```

`fromCodePoint` accepts only Unicode scalar values: `0..1114111` excluding the
surrogate range `55296..57343` (U+D800..U+DFFF). A negative value, a value above
U+10FFFF, or a surrogate raises `CharactersError`. Nothing is truncated,
wrapped, or replaced with U+FFFD.

## Classification

Each predicate takes exactly one character and uses the Unicode character
database built into the toolchain AhdCode ships with (Unicode 17.0.0 for Go
1.27.0). The compiler and a compiled program use the same tables.

| Function | True for |
|---|---|
| `isLetter` | General Category L: `Lu`, `Ll`, `Lt`, `Lm`, `Lo` |
| `isDigit` | General Category `Nd` only — decimal digits of any script, such as `7` and `٣` |
| `isWhitespace` | the Unicode `White_Space` property, the same definition `String.trim` uses |
| `isUpper` | General Category `Lu` |
| `isLower` | General Category `Ll` |
| `isAlphaNumeric` | a letter (`L`) or a decimal digit (`Nd`) |
| `isPunctuation` | General Category P: `Pc`, `Pd`, `Ps`, `Pe`, `Pi`, `Pf`, `Po` |
| `isSymbol` | General Category S: `Sm`, `Sc`, `Sk`, `So` |

`isDigit` is deliberately narrow: `"²"` (`No`) and `"Ⅻ"` (`Nl`) are numeric
characters but not decimal digits, so `isDigit` is `false` for them. A letter
without case, such as `"語"`, is a letter that is neither upper nor lower. An
emoji such as `"😊"` is a symbol (`So`). ASCII `+`, `$`, `^`, and `|` are
symbols, not punctuation.

Turkish letters classify by their own code points, with no locale rule:

```ahd
bring Characters

write(Characters.isUpper("İ"))
write(Characters.isLower("ı"))
write(Characters.isLetter("ğ"))
```

=>

```text
true
true
true
```

## Exactly one character, or CharactersError

A function documented as taking `character: String` requires a String that
holds exactly one code point. It never inspects only the first one, because
that would hide the caller's mistake.

```ahd
bring Characters
from Characters bring CharactersError

attempt {
    write(Characters.isLetter("AB"))
} except CharactersError as error {
    write(error.message)
}
```

=>

```text
Characters.isLetter requires exactly one character; received 2 characters
```

The messages are stable:

```text
Characters.codePoint requires exactly one character; received an empty String
Characters.isLetter requires exactly one character; received 2 characters
Characters.fromCodePoint: -1 is outside the Unicode range 0..1114111
Characters.fromCodePoint: 55296 is a surrogate code point, not a Unicode scalar value
```

`CharactersError` derives from `Error`. A wrong static type is a compile-time
diagnostic; a String with the wrong number of code points, or an Int that is
not a scalar value, is this runtime error.

## A small text inspection

```ahd
bring Characters

text: String := "AhdCode – Türkçe: çğıöşü 😊"
letters: Int := 0
spaces: Int := 0
for character in Characters.list(text) {
    if Characters.isLetter(character) {
        letters += 1
    }
    if Characters.isWhitespace(character) {
        spaces += 1
    }
}
write(str(Characters.count(text)) + " characters, " + str(letters) + " letters, " + str(spaces) + " spaces")
```

=>

```text
26 characters, 19 letters, 4 spaces
```

The complete runnable version is
`examples/v0.1/59_characters.ahd`.

## Execution modes

`ahdcode run`, `ahdcode build`, a relocated compiled executable, and the
persistent REPL all run the same Characters implementation and report the same
results and messages. Characters uses no filesystem, network, or helper
process.

## Not in this module

No `Char` type, no byte strings, no grapheme-cluster segmentation, no Unicode
normalization (NFC/NFD), no case conversion beyond `String.lower`/`upper`, no
locale-specific rules, and no character names or script properties.
