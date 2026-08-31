# Complex and Numeric

[English] · [Türkçe](NUMERIC_TR.md)

[Back to README](../README.md) · [Statistics](STATISTICS.md) · [Plot](PLOT.md) · [Modules](MODULES.md)

## Complex

`Complex` is a language scalar. Only an uppercase `I` attached directly to a
number creates an imaginary literal:

```ahd
z := 2 + 3I
explicit: Complex := 7 - 3I
```

`3i`, `3 I`, and bare `I` are invalid. Safe widening is `Int -> Real`, `Int ->
Complex`, and `Real -> Complex`; there is no implicit `Complex -> Real` or
String conversion. Complex supports ordinary arithmetic/equality and
`Complex ^ Int`, but no ordering. Its operations are `real()`, `imag()`,
`conjugate()`, `magnitude()`, and `phase()`.

Text always contains canonical Real components: `2.0+3.0I`, `2.0-3.0I`, and
`0.0+5.0I`.

## Numeric

```ahd
bring Numeric

v := Numeric.vector([1, 2, 3])
m := Numeric.matrix([[1, 2], [3, 4]])
x := Numeric.linspace(0.0, 10.0, 101)
```

The canonical module identity is `builtin:Numeric`. It exports immutable,
Real-oriented `Vector` and `Matrix` values plus `NumericError`. Constructors
accept `List<Int>`/`List<Real>` (including nested Lists for Matrix), never
Strings. Other constructors are `zeros`, `ones`, and `identity`.

Vector provides `length`, `values`, `add`, `subtract`, `scale`, `dot`, `abs`,
`sqrt`, `exp`, `log`, `sum`, `min`, and `max`. Matrix provides `rowCount`,
`columnCount`, `rows`, `transpose`, `add`, `subtract`, `scale`, `matmul`,
`determinant`, `trace`, `inverse`, `solve`, `rank`, `lu`, `qr`, `cholesky`,
`svd`, `eigenvalues`, the elementwise operations, and reductions. Operations
never mutate constructor Lists or receivers. There is no broadcasting.

Decomposition contracts are insertion ordered: LU has `P`, `L`, `U`; QR has
`Q`, `R`; SVD has `U`, diagonal `S`, `V`; Cholesky returns the lower factor.
Eigenvalues are `List<Complex>` in Gonum backend order. That is a result order,
not a language ordering for Complex numbers.

Simple operations stay in the standard-library-only generated runtime.
Advanced linear algebra is delegated to the bundled `ahdnumeric` Gonum helper
using a bounded, deterministic JSON request. Helper discovery checks
`AHDCODE_NUMERIC_RUNTIME`, the compiler/runtime executable directory, and the
installed `libexec/ahdcode` directory. Failures become `NumericError`.

Plot adds `Vector` overloads for `Plot.line`, `Plot.scatter`, and matching
Chart methods without changing its existing List overloads.
