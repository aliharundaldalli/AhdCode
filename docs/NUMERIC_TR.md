# Numeric ve Complex

[English](NUMERIC.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Math](MATH_TR.md)

**Matematik yolu:** [Math](MATH_TR.md) → **Numeric** → [Statistics](STATISTICS_TR.md) → [Plot](PLOT_TR.md)

[Math](MATH_TR.md) **tek seferde bir sayıyla** hesap yapar: `Math.sqrt(2.0)`,
`Math.sin(x)`. `Numeric` ise **birçok sayıyla birlikte** hesap yapar; sayılar
bir `Vector` (bir sayı dizisi) ya da bir `Matrix` (bir sayı ızgarası) olarak
düzenlenir. Buna doğrusal cebir eklenir: çarpımlar, determinantlar, ters
matrisler, denklem sistemlerini çözme ve matris ayrışımları. Bu sayfa dilin
karmaşık sayı skaleri olan `Complex`'i de anlatır.

Yazdığınız matematik vektör ya da matris kullanıyorsa Numeric'i seçin: bir
doğrusal denklem sistemi, bir döndürme, matris biçiminde bir en küçük kareler
problemi, çizmek istediğiniz bir fonksiyonun örnek noktaları. Tek bir formül
için Math yeterlidir; bir veri kümesini özetlemek için
[Statistics](STATISTICS_TR.md) kullanın.

```ahd
bring Numeric

a := Numeric.matrix([[2, 1], [1, 3]])
b := Numeric.vector([3, 5])
write(a.solve(b))
```

```text
Vector([0.7999999999999999, 1.4000000000000001])
```

Bu program `2x + y = 3`, `x + 3y = 5` sistemini çözer; tam yanıt `x = 0.8`,
`y = 1.4`'tür, son basamaklar kayan nokta yuvarlamasıdır (bkz.
[Int ve Real](MATH_TR.md#hesaplamada-int-ve-real)).

## Bu sayfada

- [Karmaşık sayılar](#karmaşık-sayılar)
- [Vektörler](#vektörler): [oluşturma](#vektör-oluşturma), [okuma](#bir-vektörü-okumak),
  [aritmetik](#vektör-aritmetiği), [öğe öğe fonksiyonlar ve toplamlar](#öğe-öğe-fonksiyonlar-ve-toplamlar),
  [geometri](#nokta-çarpım-uzunluk-ve-vektörel-çarpım)
- [Matrisler](#matrisler): [oluşturma](#matris-oluşturma), [okuma](#bir-matrisi-okumak),
  [aritmetik](#matris-aritmetiği), [özellikler](#determinant-iz-rank-ve-norm),
  [doğrusal sistem çözme](#doğrusal-sistem-çözme), [ayrışımlar](#ayrışımlar-ve-özdeğerler)
- [Numeric'i Math, Statistics ve Plot ile kullanmak](#numerici-math-statistics-ve-plot-ile-kullanmak)
- [Hatalar](#hatalar)
- [Numeric nasıl çalışır](#numeric-nasıl-çalışır)
- [Hızlı başvuru](#hızlı-başvuru)

## Karmaşık sayılar

`Complex` dilin içindedir, bu yüzden `bring` gerektirmez. Bir sayının hemen
arkasına yazılan büyük `I` sanal bir değişmez (literal) oluşturur:

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

- Yalnızca sayıya bitişik `I` sanaldır: `3i`, `3 I` ve tek başına `I`
  geçersizdir.
- `Int` ve `Real` kendiliğinden `Complex`'e genişler (`z + 0.5` çalışır).
  `Complex -> Real` ya da `Complex -> String` dönüşümü kendiliğinden olmaz;
  parçaları açıkça okuyun.
- Karmaşık değerler `+`, `-`, `*`, `/`, `Complex ^ Int`, `==` ve `!=`
  destekler. Sıralamaları yoktur; bu yüzden `<` ve `>` derlenmez.
- Metotlar: `real()` ve `imag()` parçaları döndürür, `magnitude()` `|z|`'dir,
  `phase()` radyan cinsinden açıdır ([`Math.atan2`](MATH_TR.md#ters-fonksiyonlar)'nin
  vereceği gibi), `conjugate()` sanal kısmın işaretini çevirir.
- Metin her zaman iki parçayı da Real olarak gösterir: `2.0+3.0I`,
  `2.0-3.0I`, `0.0+5.0I`.

Matris [özdeğerleri](#ayrışımlar-ve-özdeğerler) `List<Complex>` olarak
döndürülür.

## Vektörler

Bir `Vector`, `Real` değerlerden oluşan değişmez (immutable) ve sabit
uzunluklu bir dizidir. Her işlem yeni bir Vector ya da bir sayı döndürür;
üzerinde çağrıldığı Vector'ü hiçbir şey değiştirmez.

### Vektör oluşturma

```text
Numeric.vector(values: List<Int> | List<Real>) -> Vector
Numeric.zeros(size: Int) -> Vector
Numeric.ones(size: Int) -> Vector
Numeric.linspace(start, stop, count: Int) -> Vector
```

`linspace(start, stop, count)`, `start` ile `stop` arasında **iki ucu da
dahil** eşit aralıklı `count` değer verir; bir fonksiyonu çizmeden önce
örneklemenin olağan yolu budur. `count` pozitif olmalıdır; `count` 1 yalnızca
`start`'ı verir.

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

Int girdi Real olarak saklanır. Bir kurucu List'i kopyalar; bu yüzden List'i
sonradan değiştirmek Vector'ü değiştirmez. String hiçbir zaman kabul edilmez.

### Bir vektörü okumak

```text
vector.length() -> Int
vector.at(index: Int) -> Real
vector.values() -> List<Real>
```

İndeksler List kurallarına uyar: 0'dan başlar, negatif bir indeks sondan
sayar (`-1` sonuncudur) ve aralık dışındaki bir indeks `IndexError` verir.
İndeksle atama yoktur. `values()`, List isteyen kodlar için (örneğin
[Statistics](STATISTICS_TR.md)) sıradan bir `List<Real>` kopyası döndürür.

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

### Vektör aritmetiği

```text
a.add(b) -> Vector          a + b, öğe öğe
a.subtract(b) -> Vector     a - b, öğe öğe
a.scale(factor: Real) -> Vector   factor · a
```

İki Vector'ün uzunluğu aynı olmalıdır; aksi halde `NumericError` verilir.
Yayınlama (broadcasting) yoktur: her öğeye tek bir sayı eklemek açıkça
yazılır, örneğin `a.add(Numeric.ones(a.length()).scale(5.0))`.

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

### Öğe öğe fonksiyonlar ve toplamlar

```text
vector.abs() -> Vector      her öğe için |x|
vector.sqrt() -> Vector     her öğe için √x, bütün öğeler >= 0
vector.exp() -> Vector      her öğe için eˣ
vector.log() -> Vector      her öğe için ln x, bütün öğeler > 0
vector.sum() -> Real
vector.min() -> Real
vector.max() -> Real
```

Öğe öğe fonksiyonlar her öğeye karşılık gelen [Math](MATH_TR.md)
fonksiyonunu uygular ve aynı tanım kümesi kurallarını korur; bir ihlal
`NumericError` verir.

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

### Nokta çarpım, uzunluk ve vektörel çarpım

```text
a.dot(b) -> Real        a · b = Σ aᵢbᵢ
vector.norm() -> Real   Öklid uzunluğu √(Σ xᵢ²)
a.cross(b) -> Vector    a × b, ikisi de 3 uzunluklu
a.outer(b) -> Matrix    aᵢ · bⱼ çarpımlarının matrisi
```

- `norm()`, Öklid (L2) uzunluğudur. İç ölçekleme ile hesaplanır; bu yüzden
  çok büyük ya da çok küçük öğeler taşmaz. Başka bir norm yoktur.
- `cross` 3 uzunluklu iki Vector ister ve sağ el kuralına uyar: x × y = z.
  Başka bir uzunluk `NumericError` verir.
- `outer`, `len(a)`'ya `len(b)` boyutlu bir Matrix verir.

İki vektör arasındaki açı cos θ = (a · b) / (|a| |b|) bağıntısından çıkar:

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

## Matrisler

Bir `Matrix`, `Real` değerlerden oluşan değişmez, dikdörtgen bir ızgaradır.
Vector gibi, işlemleri onu hiçbir zaman değiştirmez.

### Matris oluşturma

```text
Numeric.matrix(rows: List<List<Int>> | List<List<Real>>) -> Matrix
Numeric.zeros(rows: Int, columns: Int) -> Matrix
Numeric.ones(rows: Int, columns: Int) -> Matrix
Numeric.identity(size: Int) -> Matrix
```

`matrix` satırlardan oluşan bir List alır; her satırın uzunluğu aynı
olmalıdır, aksi halde `NumericError` verilir.

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

### Bir matrisi okumak

```text
matrix.rowCount() -> Int
matrix.columnCount() -> Int
matrix.at(row: Int, column: Int) -> Real
matrix.row(index: Int) -> Vector
matrix.column(index: Int) -> Vector
matrix.diagonal() -> Vector
matrix.rows() -> List<List<Real>>
```

İndeksler Vector'deki gibi 0'dan başlar ve negatif olabilir. `diagonal()`,
`min(satır, sütun)` öğeye sahiptir; bu yüzden kare olmayan bir matriste de
çalışır. `rows()` sıradan, iç içe bir List kopyası döndürür.

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

### Matris aritmetiği

```text
a.add(b) -> Matrix             öğe öğe, aynı boyut
a.subtract(b) -> Matrix        öğe öğe, aynı boyut
a.scale(factor: Real) -> Matrix
a.hadamard(b) -> Matrix        öğe öğe çarpım, aynı boyut
a.matmul(b) -> Matrix          matris çarpımı A·B, A'nın sütunları = B'nin satırları
a.matvec(v) -> Vector          matris-vektör çarpımı A·v, A'nın sütunları = v'nin uzunluğu
matrix.transpose() -> Matrix
```

`matmul` doğrusal cebirdeki matris çarpımıdır; `hadamard` karşılıklı öğeleri
çarpar. Bu ikisini karıştırmak en yaygın hatadır; bu yüzden adları farklıdır.
Boyut uyuşmazlığı `NumericError` verir.

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

Bir Matrix de Vector gibi öğe öğe `abs()`, `sqrt()`, `exp()` ve `log()` ile
`sum()`, `min()` ve `max()` toplamlarına sahiptir. `matrix.exp()` **her
öğenin** eˣ değerini alır; matris üsteli değildir.

### Determinant, iz, rank ve norm

```text
matrix.determinant() -> Real   yalnızca kare matrisler
matrix.trace() -> Real         köşegenin toplamı (iz)
matrix.rank() -> Int
matrix.norm() -> Real          Frobenius normu √(Σ aᵢⱼ²)
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

Determinant kayan noktada LU ayrışımıyla hesaplanır; bu yüzden tam
determinantı sıfır olan bir matris bunun yerine çok küçük bir değer (ya da
`-0.0`) bildirebilir. "Bu matrisin tersi var mı?" sorusu için determinantı
`0.0` ile karşılaştırmak yerine `rank()` kullanın ya da `solve`/`inverse`'ün
`NumericError`'ını yakalayın.

### Doğrusal sistem çözme

`A·x = b` denklemini çözmek için `A.solve(b)` çağırın. Bu,
`A.inverse().matvec(b)` hesaplamaktan daha hızlı ve daha doğrudur; sistemin
tek bir çözümü yoksa `NumericError` verir.

Şu sistem

```text
 x +  y +  z =  6
2x -  y + 3z =  9
      y + 2z =  8
```

şöyle çözülür:

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

İkinci satır yanıtı denetler: `A·x - b` artığı pratikte sıfırdır. Ters
matrisin kendisine ihtiyacınız olduğunda `inverse()` vardır:

```ahd
bring Numeric

write(Numeric.matrix([[2, 1], [1, 3]]).inverse())
```

```text
Matrix([[0.6, -0.2], [-0.2, 0.4]])
```

### Ayrışımlar ve özdeğerler

```text
matrix.lu() -> Pair<String, Matrix>          anahtarlar "P", "L", "U":  P·A = L·U
matrix.qr() -> Pair<String, Matrix>          anahtarlar "Q", "R":       A = Q·R
matrix.svd() -> Pair<String, Matrix>         anahtarlar "U", "S", "V":  A = U·S·Vᵀ
matrix.cholesky() -> Matrix                  A = L·Lᵀ olan alt üçgensel L
matrix.eigenvalues() -> List<Complex>
```

Pair sonuçları gösterilen anahtar sırasını korur; bir çarpanı
`result["Q"]` ile okuyun. `S`, tekil değerlerin köşegen Matrix'idir.
`cholesky` simetrik ve pozitif tanımlı bir matris ister. `eigenvalues`
Complex değerler döndürür, çünkü gerçel bir matrisin karmaşık özdeğerleri
olabilir: 90°'lik bir döndürmenin özdeğerleri `±i`'dir. Sıraları sayısal
kütüphanenin ürettiği sıradır; sıralanmamıştır.

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

## Numeric'i Math, Statistics ve Plot ile kullanmak

- **Math** tek sayılarla çalışır. Her öğeye bir List üzerinden uygulayın:
  `v.values().map(lambda (x: Real) -> Math.sin(x))`, ya da varsa öğe öğe
  Vector metodunu kullanın.
- **Statistics** List alır; `v.values()` verin.
- **Plot**, `Plot.line`, `Plot.scatter` ve karşılık gelen Chart metotlarında
  `x` ve `y` için doğrudan Vector kabul eder; `Plot.surface` için yükseklik
  ızgarası bir Matrix'tir.

Bu örnek `sin x`'i `[0, 2π]` aralığında örnekler, özetler ve çizer:

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

## Hatalar

Numeric her sayısal hata için `NumericError` (`Error`'ın doğrudan alt
sınıfı), aralık dışındaki bir indeks için `IndexError` verir.
`NumericError`'ı adıyla yakalamak için onu getirin:
`from Numeric bring NumericError`.

| Durum | Mesaj |
|---|---|
| Vector uzunlukları farklı | `vector lengths do not match` |
| matris satırlarının uzunlukları farklı | `matrix rows must have equal lengths` |
| `matmul` boyutları uymuyor | `matrix multiplication shapes do not match` |
| kare olmayan matrisin determinantı, tersi, … | `operation requires a square matrix` |
| tekil bir matrisin `inverse`'ü | `matrix is singular` |
| tek çözümü olmayan `solve` | `system is singular or unsolvable` |
| simetrik olmayan bir matrisin `cholesky`'si | `Cholesky requires a symmetric matrix` |
| tanım kümesi dışındaki bir öğenin `sqrt` / `log`'u | `sqrt requires non-negative values`, `log requires positive values` |
| `count <= 0` ile `linspace` | `linspace count must be positive` |

NaN ya da sonsuz olacak bir sonuç da `NumericError` verir. Hata mesajları
İngilizcedir.

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

## Numeric nasıl çalışır

Oluşturma, okuma, aritmetik, öğe öğe fonksiyonlar, toplamlar, normlar ve
çarpımlar derlenmiş programın içinde çalışır. Determinant, ters matris,
`solve`, rank ve ayrışımlar, Gonum sayısal kütüphanesini kullanan paketli
`ahdnumeric` yardımcısına devredilir. Yardımcı `AHDCODE_NUMERIC_RUNTIME`,
derleyicinin kendi dizini ya da kurulu `libexec/ahdcode` dizini üzerinden
bulunur; çalıştırılamazsa çağrı `NumericError` verir.

Numeric'te yayınlama (broadcasting), seyrek matrisler, GPU desteği ya da
otomatik türev yoktur.

## Hızlı başvuru

| Üye | Sonuç | Notlar |
|---|---|---|
| `Numeric.vector(values)` | `Vector` | `List<Int>` ya da `List<Real>` |
| `Numeric.matrix(rows)` | `Matrix` | eşit uzunluklu satırların List'i |
| `Numeric.zeros(size)`, `Numeric.ones(size)` | `Vector` | |
| `Numeric.zeros(rows, columns)`, `Numeric.ones(rows, columns)` | `Matrix` | |
| `Numeric.identity(size)` | `Matrix` | |
| `Numeric.linspace(start, stop, count)` | `Vector` | iki uç da dahil |
| `v.length()`, `v.at(i)`, `v.values()` | `Int`, `Real`, `List<Real>` | |
| `v.add(w)`, `v.subtract(w)`, `v.scale(k)` | `Vector` | aynı uzunluk |
| `v.dot(w)`, `v.norm()` | `Real` | |
| `v.cross(w)`, `v.outer(w)` | `Vector`, `Matrix` | cross: 3 uzunluk |
| `m.rowCount()`, `m.columnCount()`, `m.at(r, c)` | `Int`, `Int`, `Real` | |
| `m.row(i)`, `m.column(j)`, `m.diagonal()` | `Vector` | |
| `m.rows()` | `List<List<Real>>` | |
| `m.add(n)`, `m.subtract(n)`, `m.scale(k)`, `m.hadamard(n)` | `Matrix` | aynı boyut |
| `m.matmul(n)`, `m.transpose()` | `Matrix` | |
| `m.matvec(v)`, `m.solve(b)` | `Vector` | |
| `m.determinant()`, `m.trace()`, `m.norm()` | `Real` | |
| `m.rank()` | `Int` | |
| `m.inverse()`, `m.cholesky()` | `Matrix` | |
| `m.lu()`, `m.qr()`, `m.svd()` | `Pair<String, Matrix>` | |
| `m.eigenvalues()` | `List<Complex>` | |
| `abs()`, `sqrt()`, `exp()`, `log()` | aynı tür | öğe öğe, Vector ve Matrix |
| `sum()`, `min()`, `max()` | `Real` | Vector ve Matrix |

**Sonraki:** [Statistics](STATISTICS_TR.md) — veriyi ortalama, yayılım,
çeyrekler, korelasyon ve uydurulmuş bir doğruyla betimlemek.
