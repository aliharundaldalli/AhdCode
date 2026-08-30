# String API

[English] · [Türkçe](STRING_API_TR.md)

[Back to README](../README.md) · [Fundamentals](FUNDAMENTALS.md)

String is immutable and indexed by Unicode character, not UTF-8 byte. Every
operation below requires a non-null receiver and non-null arguments.

| Operation | Result |
|---|---|
| `trim()` | removes leading/trailing Unicode whitespace |
| `lower()` | locale-independent Unicode lowercase |
| `upper()` | locale-independent Unicode uppercase |
| `capitalize()` | uppercases only the first character |
| `split(separator)` | `List<String>` with empty fields preserved |
| `replace(old, new)` | replaces all non-overlapping matches |
| `contains(text)` | substring membership |
| `startsWith(prefix)` | prefix test |
| `endsWith(suffix)` | suffix test |
| `count(text)` | non-overlapping occurrence count |
| `index(text)` | first character index |

```ahd
text: String := "  Ali,Veli  "
write(text.trim().lower().split(","))
write("a✓b✓".index("✓"))
write("ali HARUN".capitalize())
```

`capitalize` preserves the remainder exactly (`"aHD"` becomes `"AHD"`).
`split("")`, `count("")`, and `index("")` raise `DomainError`. A missing
`index` search also raises `DomainError`; there is no `-1` sentinel.

These operations publish no named parameters, and v0.1 defines no aliases such
as `strip`, `toLowerCase`, or parameterless `split`.
