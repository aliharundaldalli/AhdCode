# Kontrol akışı

[English](CONTROL_FLOW.md) · [Türkçe]

[README'ye dön](../README_TR.md)

## Koşullar

```ahd
if score >= 85 {
    write("Excellent")
}
else if score >= 50 {
    write("Passed")
}
else {
    write("Failed")
}
```

Koşullar `Bool` türünde olmalıdır.

## `while` ve `until`

`while` her yinelemeden önce kontrol eder:

```ahd
count: Int := 0
while count < 3 {
    write(count)
    count++
}
```

`until` ise sonradan kontrol eden (post-check) bir döngüdür. Gövdesi en az bir
kez çalışır, ardından koşul doğru (true) olduğunda çalışma durur:

```ahd
count: Int := 0
until count == 3 {
    count++
    write(count)
}
```

## `for` ve `between`

```ahd
for value in [10, 20, 30] {
    write(value)
}

for value: Int in between(1, 6, 2) {
    write(value)
}
```

`between` tembeldir (lazy), bitiş değerini dışlar, negatif adımları
destekler ve sıfır adım için `DomainError` üretir. List, String ve Pair
üzerinde yineleme, döngü başladığı anda alınan sığ bir anlık görüntü
(shallow snapshot) kullanır. Pair üzerinde yineleme anahtarları (keys) verir.

`break` ve `continue` en yakın döngüyü etkiler.

## `state` / `condition`

```ahd
state status {
    condition "active" {
        write("Active")
    }
    condition default {
        write("Unknown")
    }
}
```

Sonraki koşula düşme (fall-through) yoktur ve `break` yazmaya gerek yoktur.
