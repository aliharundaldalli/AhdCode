# Math standart modülü

[English](MATH.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Temel İşlevler](FUNDAMENTALS_TR.md)

**Matematik yolu:** **Math** → [Numeric](NUMERIC_TR.md) → [Statistics](STATISTICS_TR.md) → [Plot](PLOT_TR.md)

`Math`, okul ve üniversite matematiğinin tek sayılık fonksiyonlarını içerir:
kökler, üsler, logaritmalar, trigonometri, yuvarlama, en büyük ortak bölen ve
rastgele sayılar. Her fonksiyon bir ya da iki sayı alır ve tek bir sayı
döndürür.

```ahd
bring Math

write(Math.sqrt(25))
write(Math.round(Math.PI, 2))
```

```text
5.0
3.14
```

Math açık (explicit) bir modüldür: dosyanın başına bir kez `bring Math` yazın
ve üyelerini `Math.ad(...)` biçiminde çağırın. Birkaç adı ön ek olmadan
kullanmak için onları seçerek getirin:

```ahd
from Math bring (PI, sqrt)

write(sqrt(PI))
```

```text
1.7724538509055159
```

## Hangi modüle ihtiyacım var?

| Yapmak istediğiniz… | Kullanın |
|---|---|
| tek sayılarla hesap: `√x`, `sin x`, `ln x`, yuvarlama | **Math** (bu sayfa) |
| `abs`, `min`, `max` ya da `sum` | [Temel İşlevler](FUNDAMENTALS_TR.md) — her zaman hazır, `bring` gerekmez |
| vektör, matris, doğrusal denklem sistemi ya da karmaşık sayı | [Numeric](NUMERIC_TR.md) |
| veriyi özetlemek: ortalama, medyan, standart sapma, regresyon | [Statistics](STATISTICS_TR.md) |
| bir fonksiyonu ya da veri kümesini çizmek | [Plot](PLOT_TR.md) |

## Bu sayfada

- [Hesaplamada Int ve Real](#hesaplamada-int-ve-real)
- [Sabitler](#sabitler)
- [Üsler ve kökler](#üsler-ve-kökler)
- [Üstel fonksiyon ve logaritmalar](#üstel-fonksiyon-ve-logaritmalar)
- [Trigonometri](#trigonometri)
- [Yuvarlama](#yuvarlama)
- [Mutlak değer, en küçük, en büyük](#mutlak-değer-en-küçük-en-büyük)
- [En büyük ortak bölen ve en küçük ortak kat](#en-büyük-ortak-bölen-ve-en-küçük-ortak-kat)
- [Rastgele sayılar](#rastgele-sayılar)
- [Tanım kümesi ve taşma hataları](#tanım-kümesi-ve-taşma-hataları)
- [Çözümlü örnekler](#çözümlü-örnekler)
- [Hızlı başvuru](#hızlı-başvuru)

## Hesaplamada Int ve Real

AhdCode'da iki sayı türü vardır. `Int` bir tam sayıdır (işaretli 64 bit),
`Real` ise ondalıklı bir sayıdır (64 bit kayan nokta). Bir ifadenin hangisini
ürettiğini bilmek, Math'in davranışının büyük kısmını açıklar.

- `Real` beklenen her yerde `Int` kabul edilir; bu yüzden `Math.sqrt(16)`
  çalışır ve `4.0` döndürür. Tersi hiçbir zaman kendiliğinden olmaz: `int(x)`
  (kesirli kısmı sıfıra doğru atar) ya da aşağıdaki yuvarlama fonksiyonlarından
  birini kullanın.
- `/` her zaman `Real` üretir, iki Int için bile: `7 / 2` değeri `3.5`,
  `6 / 3` değeri `2.0`'dır.
- `%`, Int kalanıdır. İşareti sol işlenene uyar: `7 % 3` değeri `1`,
  `-7 % 3` değeri `-1`'dir.
- `^` üs almadır (`Math.pow` yoktur). `Int ^ Int` sonucu `Int` kalır ve
  negatif olmayan bir üs ister; iki taraftan biri `Real` ise sonuç `Real` olur.
- Sıfıra bölme, Int ve Real için aynı biçimde `DivisionByZeroError` verir.
- `Int` aralığının dışına çıkan bir sonuç başa sarmak yerine `OverflowError`
  verir. Taşan bir sabit ifade zaten derleme hatasıdır.
- Bir `Real` sonuç her zaman sonludur: AhdCode programları NaN ya da sonsuzluk
  görmez. Başka bir dilin bunlardan birini döndüreceği yerde AhdCode hata verir
  (bkz. [Tanım kümesi ve taşma hataları](#tanım-kümesi-ve-taşma-hataları)).

```ahd
write(7 / 2)
write(type(6 / 3))
write(-7 % 3)
write(2 ^ 10)
write(2.0 ^ 0.5)
```

```text
3.5
Real
-1
1024
1.4142135623730951
```

`Real` ikili kayan noktalı bir sayıdır; bu yüzden `0.1` gibi bir ondalık kesir
yaklaşık olarak saklanır ve bir sonucun son basamağı tam değerden farklı
olabilir: `Math.sin(Math.PI / 6)` çıktısı `0.5` değil
`0.49999999999999994`'tür. Yalnızca bir değeri gösterirken yuvarlayın;
hesaplarken tam hassasiyeti koruyun.

## Sabitler

```text
Math.PI  -> Real    π = 3.141592653589793
Math.E   -> Real    e = 2.718281828459045, doğal logaritmanın tabanı
```

İkisi de sabittir: bunlara atama yapılamaz.

```ahd
bring Math

radius := 3.0
write(Math.round(Math.PI * radius ^ 2, 2))
write(Math.round(2 * Math.PI * radius, 2))
```

```text
28.27
18.85
```

## Üsler ve kökler

```text
x ^ n                      üs (işleç)
Math.sqrt(value) -> Real   karekök, value >= 0
Math.cbrt(value) -> Real   küpkök, her işaret
Math.hypot(x, y) -> Real   √(x² + y²)
```

- `sqrt`, negatif değerleri `DomainError` ile reddeder; Math karmaşık kök
  döndürmez.
- `cbrt` negatif değerleri kabul eder: `Math.cbrt(-27.0)` değeri `-3.0`'dır.
  İşleç biçimi `(-27.0) ^ (1.0 / 3.0)` ise `DomainError` verir, çünkü negatif
  bir tabanın kesirli bir üssü genel olarak Real bir değer değildir.
- `hypot(x, y)`, `(x, y)` vektörünün uzunluğudur ve çok büyük girdilerde
  `Math.sqrt(x ^ 2 + y ^ 2)` ifadesinin yaşayabileceği taşma olmadan hesaplanır.
- Diğer kökler birer üstür: 81'in dördüncü kökü `81.0 ^ 0.25`'tir.

```ahd
bring Math

write(Math.sqrt(2))
write(Math.cbrt(-8.0))
write(Math.hypot(3, 4))
write(81.0 ^ 0.25)
```

```text
1.4142135623730951
-2.0
5.0
3.0000000000000004
```

`81.0 ^ 0.25` matematiksel olarak `3`'tür; son basamaktaki `4`,
[Hesaplamada Int ve Real](#hesaplamada-int-ve-real) bölümünde anlatılan kayan
nokta yuvarlamasıdır. Böyle bir değeri gösterirken
`Math.round(81.0 ^ 0.25, 10)` kullanın.

## Üstel fonksiyon ve logaritmalar

```text
Math.exp(value) -> Real     eˣ
Math.log(value) -> Real     doğal logaritma ln x, value > 0
Math.log10(value) -> Real   10 tabanında logaritma, value > 0
Math.log2(value) -> Real    2 tabanında logaritma, value > 0
```

`log`, **doğal** logaritmadır (e tabanında); 10 tabanında değildir. Her
logaritma sıfırdan büyük bir değer ister; sıfır ya da negatif bir değer için
`DomainError` verir. `eˣ` sonlu bir `Real` için fazla büyüdüğünde
(yaklaşık `x = 710`'dan itibaren) `exp`, `OverflowError` verir.

Başka bir taban için iki logaritmayı bölün: log_b(x) = ln x / ln b. Bölüm bir
kayan nokta değeridir, bu yüzden `log₃ 81` çıktısı `4.000000000000001` olur;
bir tam sayıyla karşılaştırmadan önce yuvarlayın.

```ahd
bring Math

write(Math.exp(1))
write(Math.log(Math.E))
write(Math.log10(1000))
write(Math.log2(1024.0))
write(Math.log(81.0) / Math.log(3.0))
```

```text
2.718281828459045
1.0
3.0
10.0
4.000000000000001
```

## Trigonometri

### Açılar radyan cinsindendir

Bütün trigonometrik fonksiyonlar **radyan** ile çalışır. Açı sıradan bir
`Real`'dir; ayrı bir açı türü yoktur. Dereceyi `radians` ile radyana,
radyanı `degrees` ile dereceye çevirin:

```text
Math.radians(degrees) -> Real   degrees * π / 180
Math.degrees(radians) -> Real   radians * 180 / π
```

İkisi de normalleştirme yapmaz: `Math.degrees(Math.radians(720.0))` değeri
`0.0` değil `720.0`'dır.

### Sinüs, kosinüs, tanjant

```text
Math.sin(value) -> Real
Math.cos(value) -> Real
Math.tan(value) -> Real
```

```ahd
bring Math

for degrees in between(0, 91, 30) {
    angle: Local := Math.radians(degrees)
    write("{degrees}°  sin={Math.round(Math.sin(angle), 4)}  cos={Math.round(Math.cos(angle), 4)}")
}
```

```text
0°  sin=0.0  cos=1.0
30°  sin=0.5  cos=0.866
60°  sin=0.866  cos=0.5
90°  sin=1.0  cos=0.0
```

`Math.PI / 2`, π/2'ye en yakın `Real` değerdir; bu yüzden
`Math.tan(Math.PI / 2)` bir hata değil, çok büyük sonlu bir sayıdır.

### Ters fonksiyonlar

```text
Math.asin(value) -> Real      value -1..1 aralığında, sonuç [-π/2, π/2]
Math.acos(value) -> Real      value -1..1 aralığında, sonuç [0, π]
Math.atan(value) -> Real      sonuç (-π/2, π/2)
Math.atan2(y, x) -> Real      (x, y) noktasının açısı, sonuç [-π, π]
```

`atan2`, "`(x, y)` noktası hangi yönde?" sorusunu yanıtlar ve `atan(y / x)`
ifadesinden farklı olarak bölgeyi (çeyreği) bilir, `x = 0` durumunu da
karşılar. Argüman sırasına dikkat edin: **önce `y`**. `Math.atan2(0, 0)`
değeri `0.0`'dır.

```ahd
bring Math

write(Math.degrees(Math.atan2(1.0, 1.0)))
write(Math.degrees(Math.atan2(1.0, -1.0)))
write(Math.degrees(Math.asin(0.5)))
```

```text
45.0
135.0
30.000000000000004
```

### Hiperbolik fonksiyonlar

```text
Math.sinh(value) -> Real
Math.cosh(value) -> Real
Math.tanh(value) -> Real
```

Sonlu bir `Real` için fazla büyük bir sonuç, örneğin `Math.cosh(1000.0)`,
`OverflowError` verir.

## Yuvarlama

```text
Math.round(value) -> Real               en yakın tam değer
Math.round(value, digits: Int) -> Real  `digits` ondalık basamaklı en yakın değer, digits 0..15
Math.floor(value) -> Int                value'dan küçük ya da eşit en büyük Int
Math.ceil(value) -> Int                 value'dan büyük ya da eşit en küçük Int
int(value) -> Int                       kesirli kısmı atar (sıfıra doğru), bring gerekmez
```

Dördü negatif sayılarda ve tam buçuklarda birbirinden ayrılır:

| değer | `Math.round` | `Math.floor` | `Math.ceil` | `int` |
|---|---|---|---|---|
| `2.5` | `3.0` | `2` | `3` | `2` |
| `-2.5` | `-3.0` | `-3` | `-2` | `-2` |
| `-2.7` | `-3.0` | `-3` | `-2` | `-2` |

- `round` bir **Real** döndürür ve tam buçukları sıfırdan uzağa yuvarlar. Bir
  değeri göstermek için kullanın: `Math.round(3.14159, 2)` değeri `3.14`'tür.
  `0..15` dışındaki bir `digits` değeri `DomainError` verir.
- `floor` ve `ceil` bir **Int** döndürür; böylece bir Real'i sayabileceğiniz
  ya da indeks olarak kullanabileceğiniz bir tam sayıya çevirir. `Int`
  aralığının dışındaki bir değer `OverflowError` verir.
- `/` her zaman `Real` ürettiği için, negatif olmayan değerlerde tam sayı
  bölmesi `Math.floor(a / b)` ile yapılır.

```ahd
bring Math

write(Math.round(3.14159, 2))
write(Math.floor(-2.5))
write(Math.ceil(-2.5))
write(int(-2.7))
write(Math.floor(17 / 5))
```

```text
3.14
-3
-2
-2
3
```

## Mutlak değer, en küçük, en büyük

`abs`, `min`, `max` ve `sum` Math üyesi değildir. Bunlar her dosyada `bring`
olmadan kullanılabilen [Temel İşlevler](FUNDAMENTALS_TR.md)'dir ve öğe
türünü korurlar: `abs(-3)` Int `3`, `abs(-3.5)` ise `3.5`'tir. `min` ve `max`
bir List alır; boş bir List `DomainError` verir, boş bir List'in `sum`
değeri ise `0`'dır (ya da `0.0`).

```ahd
write(abs(-3))
write(min([3, 1, 2]))
write(max([1.5, 2.5]))
write(sum([1, 2, 3, 4]))
```

```text
3
1
2.5
10
```

Bir veri kümesinin ortalaması, medyanı, yayılımı ve daha fazlası için
[Statistics](STATISTICS_TR.md) ile devam edin.

## En büyük ortak bölen ve en küçük ortak kat

```text
Math.gcd(first: Int, second: Int) -> Int
Math.lcm(first: Int, second: Int) -> Int
```

İkisi de Int ile çalışır ve hiçbir zaman negatif sayı döndürmez. `gcd(0, 0)`
değeri `0`'dır; argümanlardan biri `0` ise `lcm` de `0`'dır. `Int`'e sığmayan
bir sonuç başa sarmak yerine `OverflowError` verir; buna büyüklüğü en büyük
Int'ten bir fazla olan `Math.gcd(-9223372036854775808, 0)` da dahildir.

Yaygın bir kullanım bir kesri sadeleştirmektir:

```ahd
bring Math

numerator := 42
denominator := 56
divisor := Math.gcd(numerator, denominator)
write("{Math.floor(numerator / divisor)}/{Math.floor(denominator / divisor)}")
write(Math.lcm(first: 4, second: 6))
```

```text
3/4
12
```

## Rastgele sayılar

```text
Math.random() -> Real                  0.0 <= value < 1.0
Math.randomInt(min: Int, max: Int) -> Int  min <= value <= max (iki sınır da dahil)
Math.seed(value: Int) -> Nothing       diziyi yeniden başlatır
```

Yeni bir program işletim sisteminin entropisiyle başlar; bu yüzden iki
çalıştırma farklı sayılar verir. `Math.seed(n)` diziyi tekrarlanabilir yapar;
testlerin ve simülasyonların ihtiyacı budur: aynı tohum her zaman aynı diziyi
üretir.

```ahd
bring Math

Math.seed(42)
first := [Math.randomInt(1, 6), Math.randomInt(1, 6), Math.randomInt(1, 6)]
Math.seed(42)
second := [Math.randomInt(1, 6), Math.randomInt(1, 6), Math.randomInt(1, 6)]
write(first == second)
```

```text
true
```

- `min > max` olduğunda `randomInt` `DomainError` verir; eşit sınırlar o
  değeri döndürür.
- Üreteç (SplitMix64) sözde rastgeledir ve **kriptografi için uygun
  değildir**. Belirteçler ve gizli değerler için [Security](SECURITY_TR.md)
  kullanın.
- `Math.random`, `Math.randomInt` ve `List.shuffle` program genelinde tek bir
  durumu paylaşır. Eşit `randomInt` sınırları ve boş ya da tek öğeli bir
  List'i karıştırmak durum tüketmez.

## Tanım kümesi ve taşma hataları

Math hiçbir zaman NaN ya da sonsuzluk döndürmez. Bir fonksiyonun tanım
kümesi dışındaki girdi `DomainError`, türü için fazla büyük bir sonuç
`OverflowError` verir. İkisi de yakalanabilir:

```ahd
bring Math

attempt {
    write(Math.sqrt(-1.0))
} except DomainError as error {
    write(error.message)
}
attempt {
    write(Math.exp(1000.0))
} except OverflowError as error {
    write(error.message)
}
```

```text
Math.sqrt requires a non-negative value
Math.exp result exceeds finite Real range
```

| Çağrı | Koşul | Aksi halde |
|---|---|---|
| `sqrt(x)` | `x >= 0` | `DomainError` |
| `log(x)`, `log10(x)`, `log2(x)` | `x > 0` | `DomainError` |
| `asin(x)`, `acos(x)` | `-1 <= x <= 1` | `DomainError` |
| `round(x, digits)` | `digits` `0..15` aralığında | `DomainError` |
| `randomInt(min, max)` | `min <= max` | `DomainError` |
| `exp`, `sinh`, `cosh`, `Real ^ Real` | sonlu sonuç | `OverflowError` |
| `floor`, `ceil` | sonuç `Int`'e sığar | `OverflowError` |
| `gcd`, `lcm`, `Int ^ Int` | sonuç `Int`'e sığar | `OverflowError` |
| `Int ^ Int` | üs `>= 0` | `DomainError` |
| `a / b`, `a % b` | `b != 0` | `DivisionByZeroError` |

Hata mesajları İngilizcedir; programınız bunları `error.message` ile okuyabilir.

## Çözümlü örnekler

### İkinci dereceden denklemin kökleri

```ahd
bring Math

solveQuadratic: Function := (a: Real, b: Real, c: Real) -> List<Real> {
    discriminant: Local := b ^ 2 - 4 * a * c
    if discriminant < 0 {
        return []
    }
    root: Local := Math.sqrt(discriminant)
    return [(-b + root) / (2 * a), (-b - root) / (2 * a)]
}

write(solveQuadratic(1, -3, 2))
write(solveQuadratic(1, 2, 5))
```

```text
[2.0, 1.0]
[]
```

`x² + 2x + 5` için boş List, Real kök olmadığını söyler. Önce diskriminantı
denetlemek, `Math.sqrt`'yi tanım kümesinin içinde tutar.

### İki nokta arasındaki uzaklık ve yön

```ahd
bring Math

dx := 4.0 - 1.0
dy := 6.0 - 2.0
write(Math.hypot(dx, dy))
write(Math.round(Math.degrees(Math.atan2(dy, dx)), 1))
```

```text
5.0
53.1
```

### Bileşik faiz

Yıllık %5 faizle 10 yıl duran 1000'lik bir bakiye 1000 · 1,05¹⁰ olur; onu
ikiye katlamak için gereken yıl sayısı ln 2 / ln 1,05'tir.

```ahd
bring Math

balance := 1000.0 * 1.05 ^ 10
write(Math.round(balance, 2))
years := Math.log(2.0) / Math.log(1.05)
write(Math.ceil(years))
```

```text
1628.89
15
```

Bir fonksiyonu çok sayıda noktada tablolamak ya da çizmek için x değerlerini
[`Numeric.linspace`](NUMERIC_TR.md#vektör-oluşturma) ile üretin ve
[Plot](PLOT_TR.md#hızlı-başlangıç-bir-fonksiyon-çizmek)'a verin.

## Hızlı başvuru

| Üye | İmza | Anlamı |
|---|---|---|
| `PI`, `E` | `Real` | π ve e |
| `sqrt` | `(value: Real) -> Real` | √x, `x >= 0` |
| `cbrt` | `(value: Real) -> Real` | ∛x, her işaret |
| `hypot` | `(x: Real, y: Real) -> Real` | √(x² + y²) |
| `exp` | `(value: Real) -> Real` | eˣ |
| `log` | `(value: Real) -> Real` | ln x, `x > 0` |
| `log10` | `(value: Real) -> Real` | log₁₀ x, `x > 0` |
| `log2` | `(value: Real) -> Real` | log₂ x, `x > 0` |
| `sin`, `cos`, `tan` | `(value: Real) -> Real` | radyan |
| `asin`, `acos` | `(value: Real) -> Real` | `-1 <= x <= 1` |
| `atan` | `(value: Real) -> Real` | ters tanjant |
| `atan2` | `(y: Real, x: Real) -> Real` | `(x, y)` yönü |
| `sinh`, `cosh`, `tanh` | `(value: Real) -> Real` | hiperbolik fonksiyonlar |
| `radians` | `(degrees: Real) -> Real` | derece → radyan |
| `degrees` | `(radians: Real) -> Real` | radyan → derece |
| `round` | `(value: Real) -> Real`, `(value: Real, digits: Int) -> Real` | buçuklar sıfırdan uzağa |
| `floor`, `ceil` | `(value: Real) -> Int` | aşağı / yukarı Int'e yuvarlar |
| `gcd`, `lcm` | `(first: Int, second: Int) -> Int` | hiçbir zaman negatif değil |
| `random` | `() -> Real` | `0.0 <= r < 1.0` |
| `randomInt` | `(min: Int, max: Int) -> Int` | sınırlar dahil |
| `seed` | `(value: Int) -> Nothing` | tekrarlanabilir dizi |

Tabloda `Real` yazan her yerde bir `Int` argüman da kabul edilir.

**Sonraki:** [Numeric](NUMERIC_TR.md) — vektörler, matrisler, doğrusal
denklem sistemleri ve karmaşık sayılar.
