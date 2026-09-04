# Modules

[English] · [Türkçe](MODULES_TR.md)

[Back to README](../README.md) · [Math](MATH.md) · [File and Path](FILESYSTEM.md)

A local module is a sibling `.ahd` file. The reference is one case-sensitive
identifier: `Utilities` resolves to `Utilities.ahd` beside the importing file.
v0.1 has no dotted paths, package-root search, or configurable module path.

Namespace import:

```ahd
bring Utilities
write(Utilities.greet("Ali"))
```

Direct import:

```ahd
from Utilities bring greet
write(greet("Ali"))
```

Selective multiline import:

```ahd
from Utilities bring (
    greet
    farewell
)
```

Public-all import:

```ahd
from Utilities bring all
```

`all` brings only public, non-`Confidential` symbols. Import collisions and
circular dependencies are compile-time errors.

`Web` is different from both of these. It is a **bundled first-party module**:
its source is AhdCode, embedded in the compiler, compiled in the same pass as
your own files, and generated into the one self-contained executable. It
resolves offline -- no registry, manifest, lockfile, or download -- and a
local `Web.ahd` cannot shadow it, exactly as a local `HTTP.ahd` cannot shadow
`HTTP`. See the [Web guide](WEB.md).

```ahd
bring Web
from Web bring (Request, Response, HTMLNode)
```

`Web` re-exports the `HTTP` and `HTML` types it composes, and those are the
same types, not copies: a `Request` reached through `Web` registers on a bare
`HTTP.Server` unchanged. Only `Web` is public; the framework's internal
modules are reachable from framework source alone.

`Math`, `Time`, `Latex`, `Word`, `Excel`, `PDF`, `Archive`, `Path`, `File`, `Regex`, `CSV`, `Data`, `Statistics`, `Plot`, `Numeric`, `JSON`, `SQLite`, `HTTP`, `HTML`, `SMTP`, `XML`, `Env`, `Lists`, and `KeyValue` are compiler-registered and use
these same import forms. A local file cannot shadow a standard module of the
same name. `HTTP` is both the inbound server (`Server` / `Request` /
`Response`, cookies, sessions) and the outbound `Client` / `ClientRequest` /
`ClientResponse` surface. `SMTP` is send-only mail (`SMTPClient` /
`SMTPMessage`). They can also use the ordinary namespace alias form:

```ahd
bring File as F
F.writeText("note.txt", "hello")
```

See [Time](TIME.md), [CSV](CSV.md), [Data](DATA.md), [Statistics](STATISTICS.md), [Plot](PLOT.md), [Numeric](NUMERIC.md), [Word](WORD.md), [Excel](EXCEL.md), [PDF](PDF.md), [Archive](ARCHIVE.md), [JSON](JSON.md), [SQLite](SQLITE.md), [HTTP](HTTP.md), [HTML](HTML.md), [SMTP](SMTP.md), [XML](XML.md), [Env](ENV.md), [Lists](LISTS.md), [KeyValue](KEYVALUE.md), and the other module-specific references
for their typed surfaces and catchable domain errors.

To learn CSV, Data, Plot, Excel, Word, Latex, HTTP(S), and HTML as connected
student projects rather than isolated APIs, use the
[Practical Module Workshops](PRACTICAL_MODULES.md).

`Lists` and `KeyValue` are the structural transformation layer over the core
`List` and `Pair` types. Their operations are *type-directed*: the compiler
computes each call's exact result type from the argument types written at that
call site, so `Lists.chunk(List<Int>, 2)` is `List<List<Int>>` and
`Lists.chunk(List<String>, 2)` is `List<List<String>>`, with no generic syntax
in the language and nothing erased. Because a call is specialized against its
own arguments, such an operation has no unspecialized `Function` value; taking
one is a compile-time diagnostic.
