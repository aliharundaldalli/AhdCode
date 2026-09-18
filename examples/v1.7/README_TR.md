# Standart kütüphane tamamlama (v1.7)

[English](README.md) · [Türkçe]

v1.7.0 ile [Math](../../docs/MATH_TR.md), [Numeric](../../docs/NUMERIC_TR.md),
[Time](../../docs/TIME_TR.md), [Data](../../docs/DATA_TR.md) ve
[Statistics](../../docs/STATISTICS_TR.md) modüllerine gelen eklemeler için üç
küçük program. Her biri derlendiğinde (`ahdcode build`) ve `ahdcode run` ile
aynı çıktıyı verir. bcrypt bilerek gösterilmez: yalnızca eski parola
veritabanlarıyla uyumluluk için vardır; yeni programlar
`Security.passwordHash` (Argon2id) kullanmalıdır.

## scientific_basics.ahd

```bash
ahdcode run scientific_basics.ahd
```

`sqrt`, `atan2` ve `degrees` ile çözülen yaslanmış bir merdiven; `gcd` ile
sadeleştirilen bir kesir ve `lcm` ile ortak paydada toplanan kesirler;
`Vector.cross` ile bir düzlem normali ve `Matrix.matvec` ile uygulanan 90
derecelik bir döndürme.

```text
height 4.0 m, angle 53.13 degrees
hypot check 5.0
asin(0.5) is 30.0 degrees
84/126 = 2/3
1/6 + 1/4 = 5/12
2^10 has log2 10.0
u x v = [0.0, 0.0, 1.0]
|(3, 4, 12)| = 13.0
rotated (1, 0) is (0.0, 1.0)
rotated point length 1.0 (a rotation keeps lengths)
```

## time_iso.ahd

```bash
ahdcode run time_iso.ahd
```

Bir kalkışı `Time.parseISO` ile okur, uçuş süresini (`Duration`) ekler ve
varışı `toISO()` ile hem özgün ofsette hem UTC'de yazar. Son satırlar yalnızca
katı ISO biçiminin kabul edildiğini gösterir.

```text
departs  2026-09-18T22:45:00.000+03:00
arrives  2026-09-19T03:15:00.000+03:00 (Istanbul clock)
arrives  2026-09-19T00:15:00.000Z (UTC)
same moment after a round trip: true
reminder 2026-09-18T20:45:00.000+03:00
minutes between reminder and departure: 120
a late reminder would be -7200000 ms, abs 7200000 ms
ok       2026-09-18T22:45:00.000Z
ok       2026-09-18T22:45:00.500-05:30
rejected 2026-09-18 22:45
rejected 18/09/2026
rejected Europe/Istanbul
```

## data_join.ahd

```bash
ahdcode run data_join.ahd
```

Geç gelen bir öğrenciyi `Table.concat` ile listeye ekler, çalışma kayıtlarını
`Table.innerJoin(other, leftKey, rightKey)` ile birleştirir (9 numaralı
öğrencinin listede satırı olmadığı için inner join onu dışarıda bırakır),
hücreleri açıkça Int'e çevirir ve `Statistics.linearRegression` ile
`score = slope * hours + intercept` doğrusunu uydurur.

```text
id,name,hours,score
1,Ada,2,58
2,Alan,5,81
3,Grace,3,66
4,Linus,6,90

correlation 0.9995
score = 7.9 * hours + 42.15
covariance 19.75
```
