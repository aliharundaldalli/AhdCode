# Complex ve Numeric

[English](NUMERIC.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Statistics](STATISTICS_TR.md) · [Plot](PLOT_TR.md) · [Modüller](MODULES_TR.md)

## Complex

`Complex` bir dil skaleridir. Yalnızca sayıya aralıksız eklenen büyük `I` sanal
literal oluşturur:

```ahd
z := 2 + 3I
explicit: Complex := 7 - 3I
```

`3i`, `3 I` ve tek başına `I` geçersizdir. Güvenli genişletmeler `Int -> Real`,
`Int -> Complex`, `Real -> Complex`'tir; örtük `Complex -> Real` veya String
dönüşümü yoktur. Complex normal aritmetik/eşitlik ve `Complex ^ Int` destekler,
sıralama desteklemez. İşlemleri `real()`, `imag()`, `conjugate()`,
`magnitude()` ve `phase()`'dır.

Metin her zaman kanonik Real bileşenlerini içerir: `2.0+3.0I`, `2.0-3.0I`,
`0.0+5.0I`.

## Numeric

```ahd
bring Numeric

v := Numeric.vector([1, 2, 3])
m := Numeric.matrix([[1, 2], [3, 4]])
x := Numeric.linspace(0.0, 10.0, 101)
```

Kanonik modül kimliği `builtin:Numeric`'tir. Immutable, Real yönelimli
`Vector`, `Matrix` ve `NumericError` yayımlar. Kurucular `List<Int>`/
`List<Real>` (Matrix için iç içe List) kabul eder, String kabul etmez. Diğer
kurucular `zeros`, `ones`, `identity`'dir.

Vector; `length`, `values`, `add`, `subtract`, `scale`, `dot`, `abs`, `sqrt`,
`exp`, `log`, `sum`, `min`, `max` sağlar. Matrix; `rowCount`, `columnCount`,
`rows`, `transpose`, `add`, `subtract`, `scale`, `matmul`, `determinant`,
`trace`, `inverse`, `solve`, `rank`, `lu`, `qr`, `cholesky`, `svd`,
`eigenvalues`, eleman işlemleri ve indirgemeleri sağlar. İşlemler kurucu
List'lerini veya alıcıyı değiştirmez. Broadcasting yoktur.

### Elemanları okuma, normlar ve çarpımlar

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

- İndeksler List kurallarına uyar: 0'dan başlar, negatif indeks sondan sayar
  (`-1` sonuncudur), aralık dışındaki indeks `IndexError` fırlatır. Elemanlar
  salt okunurdur; indeksle atama yoktur.
- `vector.norm()` Öklid (L2) uzunluğu, `matrix.norm()` Frobenius normudur.
  İkisi de çok büyük veya çok küçük elemanlarda taşmamak için içeride
  ölçeklenir. Başka bir norm seçen argüman yoktur.
- `outer`, `a[i] * b[j]` çarpımlarından oluşan `len(a)` x `len(b)` Matrix'tir.
- `cross` uzunluğu 3 olan iki Vector ister ve sağ el kuralına uyar: x × y = z.
  Başka uzunluklar `NumericError` fırlatır.
- `diagonal()` `min(satır, sütun)` elemanlıdır; kare olmayan matrislerde de
  çalışır.
- `hadamard` aynı boyuttaki iki matrisin elemanlarını çarpar; `matvec` m x n
  bir Matrix'i n uzunluğunda bir Vector ile çarpar. Boyut uyuşmazlığı ve
  sonlu olmayan sonuç `NumericError` fırlatır.

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

Bu işlemler üretilen programın kendisinde çalışır; `ahdnumeric` yardımcısını
hiç başlatmaz. Numeric'te broadcasting, seyrek matris, GPU desteği veya
otomatik türev yoktur.

Ayrıştırma sözleşmeleri ekleme sıralıdır: LU `P`, `L`, `U`; QR `Q`, `R`; SVD
`U`, diyagonal `S`, `V` içerir; Cholesky alt çarpanı döndürür. Özdeğerler Gonum
backend sırasında `List<Complex>`'tir. Bu, Complex için dil sıralaması değildir.

Basit işlemler yalnızca standart kütüphaneli üretilen çalışma zamanında kalır.
İleri doğrusal cebir, sınırlı ve belirlenimci JSON isteğiyle paketli Gonum
`ahdnumeric` yardımcısına devredilir. Bulma sırası `AHDCODE_NUMERIC_RUNTIME`,
derleyici/çalışma zamanı executable dizini ve kurulu `libexec/ahdcode`
dizinidir. Hatalar `NumericError` olur.

Plot, mevcut List overload'larını değiştirmeden `Plot.line`, `Plot.scatter` ve
karşılık gelen Chart metotlarına `Vector` overload'ları ekler.
