# Numeric and Complex

[English] · [Türkçe](NUMERIC_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Math](MATH.md)

**Mathematics path:** [Math](MATH.md) → **Numeric** → [Statistics](STATISTICS.md) → [Plot](PLOT.md)

[Math](MATH.md) computes with **one number at a time**: `Math.sqrt(2.0)`,
`Math.sin(x)`. `Numeric` computes with **many numbers at once**, arranged as a
`Vector` (a row of numbers) or a `Matrix` (a grid of numbers), and adds linear
algebra: products, determinants, inverses, solving systems of equations, and
matrix decompositions. This page also covers `Complex`, the language's
complex-number scalar.

Choose Numeric when the mathematics you are writing down uses vectors or
matrices — a system of linear equations, a rotation, a least-squares problem
in matrix form, the sample points of a function you want to plot. For a single
formula, Math is enough; for summarizing a data set, use
[Statistics](STATISTICS.md).

```ahd
bring Numeric

a := Numeric.matrix([[2, 1], [1, 3]])
b := Numeric.vector([3, 5])
write(a.solve(b))
```

```text
Vector([0.7999999999999999, 1.4000000000000001])
```

That program solves the system `2x + y = 3`, `x + 3y = 5`; the exact answer is
`x = 0.8`, `y = 1.4`, and the last digits are floating-point rounding (see
[Int and Real](MATH.md#int-and-real-in-calculations)).

## Contents

- [Complex numbers](#complex-numbers)
- [Vectors](#vectors): [creating](#creating-vectors), [reading](#reading-a-vector),
  [arithmetic](#vector-arithmetic), [elementwise functions and totals](#elementwise-functions-and-totals),
  [geometry](#dot-product-length-and-cross-product)
- [Matrices](#matrices): [creating](#creating-matrices), [reading](#reading-a-matrix),
  [arithmetic](#matrix-arithmetic), [properties](#determinant-trace-rank-and-norm),
  [solving linear systems](#solving-linear-systems), [decompositions](#decompositions-and-eigenvalues)
- [Using Numeric with Math, Statistics, and Plot](#using-numeric-with-math-statistics-and-plot)
- [Errors](#errors)
- [How Numeric runs](#how-numeric-runs)
- [Quick reference](#quick-reference)

## Complex numbers

`Complex` is built into the language, so it needs no `bring`. An uppercase `I`
written directly after a number makes an imaginary literal:

```ahd
z := 2 + 3I
w: Complex := 1 - 1I

write(z)
write(z * w)
write(z / w)
write(z ^ 2)
write(z.magnitude())
write(z.conjugate())
```

```text
2.0+3.0I
5.0+1.0I
-0.5+2.5I
-5.0+12.0I
3.6055512754639896
2.0-3.0I
```

- Only `I` attached to a number is imaginary: `3i`, `3 I`, and a bare `I` are
  invalid.
- `Int` and `Real` widen to `Complex` automatically (`z + 0.5` works). There is
  no automatic `Complex -> Real` or `Complex -> String` conversion; read the
  parts explicitly.
- Complex values support `+`, `-`, `*`, `/`, `Complex ^ Int`, `==`, and `!=`.
  They have no ordering, so `<` and `>` do not compile.
- Methods: `real()` and `imag()` return the parts, `magnitude()` is `|z|`,
  `phase()` is the angle in radians (as [`Math.atan2`](MATH.md#inverse-functions)
  would give it), and `conjugate()` flips the sign of the imaginary part.
- Text always shows both parts as Reals: `2.0+3.0I`, `2.0-3.0I`, `0.0+5.0I`.

Matrix [eigenvalues](#decompositions-and-eigenvalues) are returned as
`List<Complex>`.

## Vectors

A `Vector` is an immutable, fixed-length sequence of `Real` values. Every
operation returns a new Vector or a number; nothing changes the Vector it is
called on.

### Creating vectors

```text
Numeric.vector(values: List<Int> | List<Real>) -> Vector
Numeric.zeros(size: Int) -> Vector
Numeric.ones(size: Int) -> Vector
Numeric.linspace(start, stop, count: Int) -> Vector
```

`linspace(start, stop, count)` gives `count` evenly spaced values from `start`
to `stop`, **including both ends** — the usual way to sample a function before
plotting it. `count` must be positive; `count` 1 gives just `start`.

```ahd
bring Numeric

write(Numeric.vector([1, 2, 3]))
write(Numeric.zeros(3))
write(Numeric.linspace(0, 1, 5))
```

```text
Vector([1.0, 2.0, 3.0])
Vector([0.0, 0.0, 0.0])
Vector([0.0, 0.25, 0.5, 0.75, 1.0])
```

Int input is stored as Real. A constructor copies the List, so changing the
List afterwards does not change the Vector. Strings are never accepted.

### Reading a vector

```text
vector.length() -> Int
vector.at(index: Int) -> Real
vector.values() -> List<Real>
```

Indexes follow List rules: they start at 0, a negative index counts from the
end (`-1` is the last), and an index out of range raises `IndexError`. There
is no indexed assignment. `values()` returns an ordinary `List<Real>` copy for
code that needs a List, such as [Statistics](STATISTICS.md).

```ahd
bring Numeric

v := Numeric.vector([3, 4, 12])
write(v.length())
write(v.at(0))
write(v.at(-1))
write(v.values())
```

```text
3
3.0
12.0
[3.0, 4.0, 12.0]
```

### Vector arithmetic

```text
a.add(b) -> Vector          a + b, entry by entry
a.subtract(b) -> Vector     a - b, entry by entry
a.scale(factor: Real) -> Vector   factor · a
```

Both Vectors must have the same length; otherwise `NumericError` is raised.
There is no broadcasting: adding a single number to every entry is written
explicitly, for example `a.add(Numeric.ones(a.length()).scale(5.0))`.

```ahd
bring Numeric

a := Numeric.vector([1, 2, 3])
b := Numeric.vector([4, 5, 6])
write(a.add(b))
write(b.subtract(a))
write(a.scale(2.0))
```

```text
Vector([5.0, 7.0, 9.0])
Vector([3.0, 3.0, 3.0])
Vector([2.0, 4.0, 6.0])
```

### Elementwise functions and totals

```text
vector.abs() -> Vector      |x| for every entry
vector.sqrt() -> Vector     √x for every entry, all entries >= 0
vector.exp() -> Vector      eˣ for every entry
vector.log() -> Vector      ln x for every entry, all entries > 0
vector.sum() -> Real
vector.min() -> Real
vector.max() -> Real
```

The elementwise functions apply the matching [Math](MATH.md) function to each
entry and keep the same domain rules; a violation raises `NumericError`.

```ahd
bring Numeric

v := Numeric.vector([1, 4, 9])
write(v.sqrt())
write(v.sum())
write(v.max())
```

```text
Vector([1.0, 2.0, 3.0])
14.0
9.0
```

### Dot product, length, and cross product

```text
a.dot(b) -> Real        a · b = Σ aᵢbᵢ
vector.norm() -> Real   Euclidean length √(Σ xᵢ²)
a.cross(b) -> Vector    a × b, both of length 3
a.outer(b) -> Matrix    the matrix of products aᵢ · bⱼ
```

- `norm()` is the Euclidean (L2) length. It is computed with internal scaling,
  so very large or very small entries do not overflow. No other norm is
  available.
- `cross` needs two Vectors of length 3 and is right-handed: x × y = z. Any
  other length raises `NumericError`.
- `outer` gives a `len(a)`-by-`len(b)` Matrix.

The angle between two vectors follows from cos θ = (a · b) / (|a| |b|):

```ahd
bring Math
bring Numeric

a := Numeric.vector([1, 0, 0])
b := Numeric.vector([1, 1, 0])
write(a.dot(b))
write(b.norm())
write(Math.degrees(Math.acos(a.dot(b) / (a.norm() * b.norm()))))
write(a.cross(b))
```

```text
1.0
1.4142135623730951
45.00000000000001
Vector([0.0, 0.0, 1.0])
```

## Matrices

A `Matrix` is an immutable, rectangular grid of `Real` values. Like a Vector,
it is never changed by its operations.

### Creating matrices

```text
Numeric.matrix(rows: List<List<Int>> | List<List<Real>>) -> Matrix
Numeric.zeros(rows: Int, columns: Int) -> Matrix
Numeric.ones(rows: Int, columns: Int) -> Matrix
Numeric.identity(size: Int) -> Matrix
```

`matrix` takes a List of rows; every row must have the same length, otherwise
`NumericError` is raised.

```ahd
bring Numeric

write(Numeric.matrix([[1, 2, 3], [4, 5, 6]]))
write(Numeric.identity(2))
write(Numeric.zeros(2, 3))
```

```text
Matrix([[1.0, 2.0, 3.0], [4.0, 5.0, 6.0]])
Matrix([[1.0, 0.0], [0.0, 1.0]])
Matrix([[0.0, 0.0, 0.0], [0.0, 0.0, 0.0]])
```

### Reading a matrix

```text
matrix.rowCount() -> Int
matrix.columnCount() -> Int
matrix.at(row: Int, column: Int) -> Real
matrix.row(index: Int) -> Vector
matrix.column(index: Int) -> Vector
matrix.diagonal() -> Vector
matrix.rows() -> List<List<Real>>
```

Indexes start at 0 and may be negative, as for Vectors. `diagonal()` has
`min(rows, columns)` entries, so it also works for a non-square matrix.
`rows()` returns an ordinary nested List copy.

```ahd
bring Numeric

m := Numeric.matrix([[1, 2, 3], [4, 5, 6]])
write("{m.rowCount()} x {m.columnCount()}")
write(m.at(row: 1, column: 0))
write(m.row(0))
write(m.column(2))
write(m.diagonal())
```

```text
2 x 3
4.0
Vector([1.0, 2.0, 3.0])
Vector([3.0, 6.0])
Vector([1.0, 5.0])
```

### Matrix arithmetic

```text
a.add(b) -> Matrix             entry by entry, same shape
a.subtract(b) -> Matrix        entry by entry, same shape
a.scale(factor: Real) -> Matrix
a.hadamard(b) -> Matrix        entry-by-entry product, same shape
a.matmul(b) -> Matrix          matrix product A·B, columns of A = rows of B
a.matvec(v) -> Vector          matrix-vector product A·v, columns of A = length of v
matrix.transpose() -> Matrix
```

`matmul` is the matrix product of linear algebra; `hadamard` multiplies
matching entries. Mixing them up is the most common mistake, so the two have
different names. A shape mismatch raises `NumericError`.

```ahd
bring Numeric

a := Numeric.matrix([[1, 2], [3, 4]])
b := Numeric.matrix([[0, 1], [1, 0]])
write(a.matmul(b))
write(a.hadamard(b))
write(a.matvec(Numeric.vector([1, 1])))
write(a.transpose())
```

```text
Matrix([[2.0, 1.0], [4.0, 3.0]])
Matrix([[0.0, 2.0], [3.0, 0.0]])
Vector([3.0, 7.0])
Matrix([[1.0, 3.0], [2.0, 4.0]])
```

A Matrix also has the elementwise `abs()`, `sqrt()`, `exp()`, and `log()` and
the totals `sum()`, `min()`, and `max()`, exactly as a Vector does. Note that
`matrix.exp()` takes eˣ of **each entry**; it is not the matrix exponential.

### Determinant, trace, rank, and norm

```text
matrix.determinant() -> Real   square matrices only
matrix.trace() -> Real         sum of the diagonal
matrix.rank() -> Int
matrix.norm() -> Real          Frobenius norm √(Σ aᵢⱼ²)
```

```ahd
bring Numeric

m := Numeric.matrix([[4, 2], [1, 3]])
write(m.determinant())
write(m.trace())
write(m.rank())
write(Numeric.matrix([[1, 2], [2, 4]]).rank())
```

```text
10.000000000000002
7.0
2
1
```

A determinant is computed by LU decomposition in floating point, so a matrix
whose exact determinant is zero may report a tiny value (or `-0.0`) instead.
To ask "is this matrix invertible?", use `rank()` or catch the `NumericError`
from `solve`/`inverse` rather than comparing the determinant with `0.0`.

### Solving linear systems

To solve `A·x = b`, call `A.solve(b)`. It is faster and more accurate than
computing `A.inverse().matvec(b)`, and it raises `NumericError` when the system
has no unique solution.

The system

```text
 x +  y +  z =  6
2x -  y + 3z =  9
      y + 2z =  8
```

is solved like this:

```ahd
bring Math
bring Numeric

a := Numeric.matrix([[1, 1, 1], [2, -1, 3], [0, 1, 2]])
b := Numeric.vector([6, 9, 8])
x := a.solve(b)
write(x.values().map(lambda (value: Real) -> Math.round(value, 9)))
write(a.matvec(x).subtract(b).norm() < 0.000000001)
```

```text
[1.0, 2.0, 3.0]
true
```

The second line checks the answer: the residual `A·x - b` is practically zero.
`inverse()` is there when you need the inverse matrix itself:

```ahd
bring Numeric

write(Numeric.matrix([[2, 1], [1, 3]]).inverse())
```

```text
Matrix([[0.6, -0.2], [-0.2, 0.4]])
```

### Decompositions and eigenvalues

```text
matrix.lu() -> Pair<String, Matrix>          keys "P", "L", "U":  P·A = L·U
matrix.qr() -> Pair<String, Matrix>          keys "Q", "R":       A = Q·R
matrix.svd() -> Pair<String, Matrix>         keys "U", "S", "V":  A = U·S·Vᵀ
matrix.cholesky() -> Matrix                  lower L with A = L·Lᵀ
matrix.eigenvalues() -> List<Complex>
```

The Pair results keep the key order shown; read a factor with `result["Q"]`.
`S` is a diagonal Matrix of singular values. `cholesky` needs a symmetric
positive-definite matrix. `eigenvalues` returns Complex values because a real
matrix can have complex eigenvalues — a 90° rotation has `±i`. Their order is
the order the numerical library produces; it is not sorted.

```ahd
bring Numeric

rotation := Numeric.matrix([[0, -1], [1, 0]])
write(rotation.eigenvalues())

a := Numeric.matrix([[2, 1], [1, 3]])
write(a.cholesky())
write(a.lu()["U"])
```

```text
[0.0+1.0I, 0.0-1.0I]
Matrix([[1.4142135623730951, 0.0], [0.7071067811865475, 1.5811388300841898]])
Matrix([[2.0, 1.0], [0.0, 2.5]])
```

## Using Numeric with Math, Statistics, and Plot

- **Math** works on single numbers. Apply it to every entry through a List:
  `v.values().map(lambda (x: Real) -> Math.sin(x))`, or use the elementwise
  Vector methods where one exists.
- **Statistics** takes Lists, so pass `v.values()`.
- **Plot** accepts a Vector directly for `x` and `y` in `Plot.line`,
  `Plot.scatter`, and the matching Chart methods, and a Matrix as the height
  grid of `Plot.surface`.

This example samples `sin x` on `[0, 2π]`, summarizes it, and draws it:

```ahd
bring Math
bring Numeric
bring Plot
bring Statistics

x := Numeric.linspace(0.0, 2 * Math.PI, 101)
y := Numeric.vector(x.values().map(lambda (value: Real) -> Math.sin(value)))
write(Math.round(Statistics.mean(y.values()), 6))
write(Math.round(y.max(), 6))
chart := Plot.line(x, y).title("$y=\\sin x$").xLabel("$x$").yLabel("$y$")
chart.save("sine.png")
```

```text
0.0
1.0
```

## Errors

Numeric raises `NumericError` (a direct subclass of `Error`) for every
numerical failure, and `IndexError` for an index out of range. To catch
`NumericError` by name, bring it: `from Numeric bring NumericError`.

| Situation | Message |
|---|---|
| Vector lengths differ | `vector lengths do not match` |
| matrix rows have different lengths | `matrix rows must have equal lengths` |
| `matmul` shapes do not fit | `matrix multiplication shapes do not match` |
| determinant, inverse, … of a non-square matrix | `operation requires a square matrix` |
| `inverse` of a singular matrix | `matrix is singular` |
| `solve` without a unique solution | `system is singular or unsolvable` |
| `cholesky` of a non-symmetric matrix | `Cholesky requires a symmetric matrix` |
| `sqrt` / `log` of an entry outside the domain | `sqrt requires non-negative values`, `log requires positive values` |
| `linspace` with `count <= 0` | `linspace count must be positive` |

A result that would be NaN or infinite also raises `NumericError`.

```ahd
bring Numeric
from Numeric bring NumericError

attempt {
    write(Numeric.matrix([[1, 2], [2, 4]]).inverse())
} except NumericError as error {
    write(error.message)
}
```

```text
matrix is singular
```

## How Numeric runs

Construction, reading, arithmetic, the elementwise functions, the totals,
norms, and products run inside the compiled program. Determinant, inverse,
`solve`, rank, and the decompositions are delegated to the bundled
`ahdnumeric` helper, which uses the Gonum numerical library. The helper is
found through `AHDCODE_NUMERIC_RUNTIME`, the compiler's own directory, or the
installed `libexec/ahdcode` directory; if it cannot run, the call raises
`NumericError`.

Numeric has no broadcasting, sparse matrices, GPU support, or automatic
differentiation.

## Quick reference

| Member | Result | Notes |
|---|---|---|
| `Numeric.vector(values)` | `Vector` | `List<Int>` or `List<Real>` |
| `Numeric.matrix(rows)` | `Matrix` | List of equal-length rows |
| `Numeric.zeros(size)`, `Numeric.ones(size)` | `Vector` | |
| `Numeric.zeros(rows, columns)`, `Numeric.ones(rows, columns)` | `Matrix` | |
| `Numeric.identity(size)` | `Matrix` | |
| `Numeric.linspace(start, stop, count)` | `Vector` | both ends included |
| `v.length()`, `v.at(i)`, `v.values()` | `Int`, `Real`, `List<Real>` | |
| `v.add(w)`, `v.subtract(w)`, `v.scale(k)` | `Vector` | same length |
| `v.dot(w)`, `v.norm()` | `Real` | |
| `v.cross(w)`, `v.outer(w)` | `Vector`, `Matrix` | cross: length 3 |
| `m.rowCount()`, `m.columnCount()`, `m.at(r, c)` | `Int`, `Int`, `Real` | |
| `m.row(i)`, `m.column(j)`, `m.diagonal()` | `Vector` | |
| `m.rows()` | `List<List<Real>>` | |
| `m.add(n)`, `m.subtract(n)`, `m.scale(k)`, `m.hadamard(n)` | `Matrix` | same shape |
| `m.matmul(n)`, `m.transpose()` | `Matrix` | |
| `m.matvec(v)`, `m.solve(b)` | `Vector` | |
| `m.determinant()`, `m.trace()`, `m.norm()` | `Real` | |
| `m.rank()` | `Int` | |
| `m.inverse()`, `m.cholesky()` | `Matrix` | |
| `m.lu()`, `m.qr()`, `m.svd()` | `Pair<String, Matrix>` | |
| `m.eigenvalues()` | `List<Complex>` | |
| `abs()`, `sqrt()`, `exp()`, `log()` | same kind | elementwise, Vector and Matrix |
| `sum()`, `min()`, `max()` | `Real` | Vector and Matrix |

**Next:** [Statistics](STATISTICS.md) — describing data with means, spreads,
quantiles, correlation, and a fitted line.
