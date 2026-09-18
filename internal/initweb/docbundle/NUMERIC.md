# Complex and Numeric

[English] · Türkçe

[Back to README](README.md) · [Statistics](STATISTICS.md) · [Plot](PLOT.md) · [Modules](MODULES.md)

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

### Reading entries, norms, and products

```text
vector.at(index: Int) -> Real
vector.norm() -> Real
vector.outer(other: Vector) -> Matrix
vector.cross(other: Vector) -> Vector
matrix.at(row: Int, column: Int) -> Real
matrix.row(index: Int) -> Vector
matrix.column(index: Int) -> Vector
matrix.diagonal() -> Vector
matrix.norm() -> Real
matrix.hadamard(other: Matrix) -> Matrix
matrix.matvec(vector: Vector) -> Vector
```

- Indexes follow List rules: they start at 0, a negative index counts from the
  end (`-1` is the last), and an index out of range raises `IndexError`.
  Entries are read-only; there is no indexed assignment.
- `vector.norm()` is the Euclidean (L2) length and `matrix.norm()` the
  Frobenius norm. Both are scaled internally so large or tiny entries do not
  overflow. There is no argument that selects another norm.
- `outer` is the `len(a)`-by-`len(b)` Matrix of products `a[i] * b[j]`.
- `cross` needs two Vectors of length 3 and is right-handed: x × y = z.
  Any other length raises `NumericError`.
- `diagonal()` has `min(rows, columns)` entries, so it works for non-square
  matrices.
- `hadamard` multiplies entries of two same-shape matrices; `matvec` multiplies
  an m-by-n Matrix by a length-n Vector. A shape mismatch raises
  `NumericError`, and a non-finite result raises `NumericError`.

```ahd
bring Numeric

v := Numeric.vector([3, 4, 12])
m := Numeric.matrix([[1, 2, 3], [4, 5, 6]])
write(v.at(-1))
write(v.norm())
write(m.at(row: 1, column: 0))
write(m.column(2).values())
write(m.diagonal().values())
write(m.matvec(Numeric.vector([1, 0, -1])).values())
write(Numeric.vector([1, 0, 0]).cross(Numeric.vector([0, 1, 0])).values())
```

```text
12.0
13.0
4.0
[3.0, 6.0]
[1.0, 5.0]
[-2.0, -2.0]
[0.0, 0.0, 1.0]
```

These operations run in the generated program itself; they never start the
`ahdnumeric` helper. Numeric has no broadcasting, sparse matrices, GPU
support, or automatic differentiation.

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
