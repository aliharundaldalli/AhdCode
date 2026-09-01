# List API

[English](LIST_API.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Lists](LISTS_TR.md) · [Koleksiyonlar](COLLECTIONS_TR.md) · [Math](MATH_TR.md)

`List<T>` için:

| İşlem | Davranış |
|---|---|
| `add(value)` | yerinde (in place) sona ekler |
| `eject(index)` | belirtilen indeksi yerinde siler; negatif indeks kabul edilir |
| `reverse()` | yerinde ters çevirir |
| `sort()` | Int, Real veya String için kararlı (stable) artan doğal sıralama |
| `sort(keyFunction)` | Int/Real/String anahtara (key) göre kararlı artan sıralama |
| `shuffle()` | yerinde, önyargısız (unbiased) Fisher–Yates permütasyonu |
| `count(value)` | derin eşitlik (deep-equality) eşleşme sayısı |
| `index(value)` | ilk derin eşitlik eşleşmesi |
| `map(function)` | dönüştürülmüş anlık görüntü (snapshot) elemanlarından oluşan yeni bir List |
| `filter(function)` | korunan anlık görüntü elemanlarından oluşan yeni bir List |

Değiştiren (mutating) işlemler nesne kimliğini (object identity) korur, bu
yüzden alias'lar bu değişiklikleri görür. `map` ve `filter` alıcıyı
(receiver) hiçbir zaman değiştirmez. Uyumlu isimli bir Function veya ifade
lambda'sı geçirin; `sort(keyFunction)` da aynı biçimi kabul eder.

```ahd
double: Function := (
    value: Int
) -> Int {
    return value * 2
}

values: List<Int> := [3, 1, 2]
mapped: List<Int> := values.map(double)
positive: List<Int> := values.filter(lambda (value: Int) -> value > 0)
values.sort()

write(values)
write(mapped)
```

Hiçbir değer eşleşmediğinde `index` `DomainError` fırlatır. Doğal sıralama
(natural sort) desteklenmeyen eleman türlerini reddeder. Constant veya derin
dondurulmuş (deep-frozen) List'ler her türlü değişikliği reddeder.

`shuffle`, `Math.random` ve `Math.randomInt` ile aynı program genelindeki
sözde rastgele (pseudo-random) durumu kullanır. Açık tohumlama (seeding) bunu
tekrarlanabilir hale getirir:

```ahd
bring Math
Math.seed(42)
values.shuffle()
```

Boş veya tek elemanlı bir shuffle hiçbir rastgele durum tüketmez.
