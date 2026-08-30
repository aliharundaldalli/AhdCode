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
seed random randomInt
```

`round`, Real döndürür ve tam buçukları sıfırdan uzağa doğru yuvarlar. İsteğe
bağlı basamak (digits) argümanı `0..15` ile sınırlıdır. `floor` ve `ceil` Int
döndürür. Trigonometrik fonksiyonlar radyan kullanır. `log` doğal
logaritmadır; `log10` on tabanındadır. `^` üs almadır; `Math.pow` yoktur.
`abs`, `sum`, `min` ve `max`, [Temel İşlevler](FUNDAMENTALS_TR.md)'dendir,
Math üyesi değildir.

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
