# Math standart modülü

[English](MATH.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [List API](LIST_API_TR.md)

Math açıktır (explicit):

```ahd
bring Math
write(Math.sqrt(25))
```

Doğrudan ve seçici içe aktarımlar da çalışır:

```ahd
from Math bring (
    PI
    sqrt
)
```

## Yüzey (Surface)

```text
PI E
round floor ceil
sqrt sin cos tan log log10 exp
asin acos atan atan2 sinh cosh tanh
hypot log2 cbrt radians degrees
gcd lcm
seed random randomInt
```

`round`, Real döndürür ve tam buçukları sıfırdan uzağa doğru yuvarlar. İsteğe
bağlı basamak (digits) argümanı `0..15` ile sınırlıdır. `floor` ve `ceil` Int
döndürür. Trigonometrik fonksiyonlar radyan kullanır. `log` doğal
logaritmadır; `log10` on tabanındadır. `^` üs almadır; `Math.pow` yoktur.
`abs`, `sum`, `min` ve `max`, [Temel İşlevler](FUNDAMENTALS_TR.md)'dendir,
Math üyesi değildir.

## Trigonometri, logaritmalar ve açılar

```text
Math.asin(value: Real) -> Real        Math.sinh(value: Real) -> Real
Math.acos(value: Real) -> Real        Math.cosh(value: Real) -> Real
Math.atan(value: Real) -> Real        Math.tanh(value: Real) -> Real
Math.atan2(y: Real, x: Real) -> Real  Math.hypot(x: Real, y: Real) -> Real
Math.log2(value: Real) -> Real        Math.cbrt(value: Real) -> Real
Math.radians(degrees: Real) -> Real   Math.degrees(radians: Real) -> Real
```

Her Real parametrede olduğu gibi Int argüman Real'e genişler.

- `asin` ve `acos`, `-1 <= value <= 1` ister; `asin` `[-π/2, π/2]`, `acos`
  `[0, π]` aralığında döner. `atan` `(-π/2, π/2)` aralığında döner.
- `atan2(y, x)`, `(x, y)` noktasının `[-π, π]` aralığındaki açısıdır.
  Argüman sırasına dikkat: önce `y`. `atan2(0, 0)` sonucu `0.0`'dır.
- `hypot(x, y)`, büyük girdilerde taşmadan `√(x² + y²)` hesaplar.
- `log2` sıfırdan büyük bir değer ister. `cbrt` negatif değerleri kabul eder:
  `Math.cbrt(-27.0)` sonucu `-3.0`'dır.
- `radians(d)`, `d * PI / 180`; `degrees(r)`, `r * 180 / PI` hesaplar. İkisi
  de normalleştirme yapmaz: `Math.degrees(Math.radians(720.0))` sonucu
  `720.0`'dır. Açılar düz Real değerlerdir; ayrı bir açı tipi yoktur.

Tanım kümesi dışındaki bir değer `DomainError` fırlatır. Sonlu bir Real'e
sığmayacak kadar büyük bir sonuç (örneğin `Math.cosh(1000.0)`)
`OverflowError` fırlatır; NaN ve sonsuz değerler programa hiçbir zaman
ulaşmaz.

```ahd
bring Math

write(Math.degrees(Math.atan2(1.0, 1.0)))
write(Math.hypot(3, 4))
write(Math.log2(1024.0))
write(Math.cbrt(-8.0))
attempt {
    write(Math.asin(2.0))
} except DomainError as error {
    write(error.message)
}
```

```text
45.0
5.0
10.0
-2.0
Math.asin requires a value between -1 and 1
```

## En büyük ortak bölen ve en küçük ortak kat

```text
Math.gcd(first: Int, second: Int) -> Int
Math.lcm(first: Int, second: Int) -> Int
```

İki sonuç da hiçbir zaman negatif değildir. `gcd(0, 0)` sonucu `0`'dır;
argümanlardan biri `0` ise `lcm` sonucu `0`'dır. `lcm`, denetimli aritmetikle
`|(first / gcd) * second|` olarak hesaplanır. Int'e sığmayan bir sonuç başa
sarmak (wrap around) yerine `OverflowError` fırlatır; buna büyüklüğü en büyük
Int'ten bir fazla olan `Math.gcd(-9223372036854775808, 0)` da dahildir.

```ahd
bring Math

write(Math.gcd(-12, 18))
write(Math.lcm(first: 4, second: 6))
write(Math.lcm(7, 0))
```

```text
6
12
0
```

## Rastgele durum (Random state)

Yeni bir yerel (native) süreç, işletim sistemi entropisinden tek, paylaşılan
bir SplitMix64 durumu başlatır. Herkese açık üretici (generator) sözde
rastgeledir (pseudo-random) ve kriptografik kullanım için uygun değildir.
Tohumlanmamış (unseeded) başlangıç tekrarlanabilir değildir.

```ahd
bring Math
write(Math.random())
write(Math.randomInt(1, 10))
```

`random()`, `0.0 <= value < 1.0` döndürür. `randomInt(min, max)` kapsayıcı
(inclusive) sınırlar kullanır ve ters çevrilmiş sınırlar için `DomainError`
fırlatır.

Testler ve simülasyonlar için açık tohumlama (seeding) kullanın:

```ahd
Math.seed(42)
write(Math.random())
```

Aynı Int ile yeniden tohumlamak (reseeding) aynı SplitMix64 dizisini
üretir. `Math.random`, `Math.randomInt` ve `List.shuffle` bu aynı program
genelindeki durumu tüketir. Eşit `randomInt` sınırları ve boş/tek elemanlı
shuffle hiçbir durum tüketmez.
