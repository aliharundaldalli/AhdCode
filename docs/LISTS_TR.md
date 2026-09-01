# Lists standart modülü

[English](LISTS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [KeyValue](KEYVALUE_TR.md) · [List API](LIST_API_TR.md)

`Lists`, derleyici tarafından kayıtlı `builtin:Lists` modülüdür. Açıktır ve
kardeş bir `Lists.ahd` dosyası onu gölgeleyemez:

```ahd
bring Lists
from Lists bring ListsError
```

`Lists`, çekirdek `List` türünün üye olarak doğal biçimde sunmadığı saf
yapısal dönüşümler için vardır. Bilinçli olarak hiçbir şeyi tekrarlamaz:
`add`, `eject`, `sort`, `reverse`, `shuffle`, `count`, `index`, `map`,
`filter`, dilimleme ve `List + List` tam olarak oldukları yerde, `List`
üzerinde kalır. Bunlar için [List API](LIST_API_TR.md) belgesine bakın.

## Yüzey

```text
Lists.chunk(values: List<T>, size: Int)          -> List<List<T>>
Lists.flatten(values: List<List<T>>)             -> List<T>
Lists.transpose(rows: List<List<T>>)             -> List<List<T>>
Lists.unique(values: List<T>)                    -> List<T>
Lists.valueCounts(values: List<K>)               -> Pair<K, Int>
Lists.groupBy(values: List<T>, key: Function(T) -> K) -> Pair<K, List<T>>
```

## İşlemler tür-yönelimlidir

Yukarıdaki `T` ve `K` AhdCode sözdizimi değildir. Kullanıcıya açık genel
(generic) `Function` sözdizimi yoktur ve `Lists.chunk<Int>(...)` gibi bir şey
asla yazılmaz. Bu işlemler derleyici tarafından sağlanır ve derleyici her
çağrının sonuç türünü gerçekten yazdığınız argüman türlerinden hesaplar:

```ahd
numbers: List<Int> := [1, 2, 3]
words: List<String> := ["a", "b"]

intChunks: List<List<Int>> := Lists.chunk(numbers, 2)
stringChunks: List<List<String>> := Lists.chunk(words, 1)
```

Her iki çağrı yerinin de tek bir kesin statik türü vardır. Hiçbir şey
`Object`, `Any` veya `dynamic` gösterimine silinmez, girişte hiçbir dönüşüm
yapılmaz ve `List<List<Int>>` asla `List<List<Real>>` biçimine genişlemez.

Her iki yazım da çalışır:

```ahd
bring Lists
from Lists bring chunk

parts := chunk([1, 2, 3], 2)
```

### Tek sınır

Sonuç türü her çağrı yerindeki argümanlardan hesaplandığı için bu işlemlerin
tek bir somut `Function` türü yoktur; dolayısıyla özelleştirilmemiş bir
`Function` değeri de yoktur:

```ahd
stored := Lists.chunk        // derleme zamanında reddedilir
```

Bu, çalışma zamanı sürprizi değil, derleme zamanı tanısıdır. İşlemi doğrudan
çağırın ya da istediğiniz kesin biçimi kendi `Function`'ınıza sarın:

```ahd
pageInts: Function := (values: List<Int>) -> List<List<Int>> {
    return Lists.chunk(values, 10)
}
```

## chunk

```ahd
write(Lists.chunk([1, 2, 3, 4, 5], 2))
```

```text
[[1, 2], [3, 4], [5]]
```

Kaynak sırası korunur. Son parça doldurulmaz, kısa kalır — uydurma bir dolgu
değeri üretilmez. Boş bir kaynak doğru türde boş bir `List` üretir; kaynaktan
büyük bir `size` ise her şeyi tutan tek bir parça üretir.

`size` sıfırdan büyük olmalıdır; `0` veya negatif bir değer, ne kastedildiğini
tahmin etmek yerine `ListsError` yükseltir.

## flatten

```ahd
empty: List<Int> := []
write(Lists.flatten([[1, 2], [3], empty, [4, 5]]))
```

```text
[1, 2, 3, 4, 5]
```

Bu tam olarak bir düzeylik düzleştirmedir. `List<List<List<Int>>>`,
`List<Int>` değil `List<List<Int>>` haline gelir: özyinelemeli düzleştirme
yoktur, çünkü bir yuvalanmanın derinliği çağıranın açıkça vermesi gereken bir
karardır.

İç Listeler null olamaz (`List<List<T>?>` değil, `List<List<T>>`). Null
olabilen bir iç List'in tanımlı bir katkısı yoktur — onu atlamak veriyi
sessizce düşürürdü — bu yüzden derleme zamanı tür hatasıdır.

## transpose

```ahd
write(Lists.transpose([[1, 2, 3], [4, 5, 6]]))
```

```text
[[1, 4], [2, 5], [3, 6]]
```

`transpose` dikdörtgen girdi ister: **her satırın uzunluğu tam olarak aynı
olmalıdır**. Düzensiz (ragged) girdi `ListsError` yükseltir:

```text
transpose requires rectangular rows: row 1 has 2 element(s); expected 3
```

Satır numarası kaynak `List` içindeki 0 tabanlı konumdur. Hiçbir şey
doldurulmaz, kırpılmaz veya tahmin edilmez. Sessiz dikdörtgenleştirme,
tablo verisinin sessizce kaybolma biçimidir; bu yüzden açıkça sunulmaz.

Sınır durumları:

```text
[]                -> []
[[], []]          -> []
[[1, 2, 3]]       -> [[1], [2], [3]]
```

İki kez transpoze etmek özgün biçimi geri getirir.

## unique

```ahd
write(Lists.unique([3, 1, 3, 2, 1]))
```

```text
[3, 1, 2]
```

Her farklı değerin **ilk** oluşumu, ilk-oluşum sırasında korunur. Farklılık
sıradan AhdCode `==` ile belirlenir — `in`, `List.count` ve `List.index` ile
aynı kural. Değerler asla metne çevrilmez, adres üzerinden hash'lenmez veya
uydurma bir kuralla karşılaştırılmaz; bu yüzden `List` ve `Pair` elemanları
derin karşılaştırılır, `Class` örnekleri ise `==` için zaten tanımlı olan
biçimde karşılaştırılır.

Null olabilen bir eleman türünde `null` sıradan, ayrı bir değerdir:

```ahd
values: List<String?> := ["a", null, "a", null, "b"]
write(Lists.unique(values))
```

```text
["a", null, "b"]
```

`List<Function>` derleme zamanında reddedilir: `==`, `Function` değerleri için
bir karşılaştırma tanımlamaz, dolayısıyla `unique`'in kullanabileceği bir şey
yoktur.

## valueCounts

```ahd
write(Lists.valueCounts([1, 1, 3, 2, 1, 3]))
```

```text
{1: 3, 3: 2, 2: 1}
```

Anahtarlar ilk-oluşum sırasında görünür. Eleman türü mevcut `Pair` anahtar
kurallarını sağlamalıdır — `String`, `Int` veya `Bool`, ve asla null olabilen:

```ahd
write(Lists.valueCounts(["Math", "Physics", "Math"]))
```

```text
{"Math": 2, "Physics": 1}
```

`List<Real>`, `List<Class>`, `List<List<Int>>` ve `List<String?>` hepsi
derleme zamanında reddedilir. Uydurmak için hiçbir şey `String`'e çevrilmez;
bu sürüm bir `Pair` anahtarının ne olabileceğini genişletmez.

## groupBy

```ahd
write(Lists.groupBy(["Ali", "Ayse", "Bora", "Ahmet"], lambda (name: String) -> name[0]))
```

```text
{"A": ["Ali", "Ayse", "Ahmet"], "B": ["Bora"]}
```

Anahtarlar ilk-anahtar-oluşum sırasında görünür ve her grubun içindeki
elemanlar kaynak sırasını korur.

Anahtar `Function`'ı tam olarak `List`'in eleman türünü — **null olabilirliği
dahil** — almalı ve null olamayan bir `Pair` anahtar türü döndürmelidir.
Kaynağın sığ (shallow) bir anlık görüntüsü üzerinde, soldan sağa, eleman
başına tam olarak bir kez çalışır — böylece kaynak `List`'i yapısal olarak
değiştiren bir geri çağırım, üzerinde gezilen şeyi değiştiremez; bu `List.map`
ve `List.filter` ile aynı davranıştır.

Geri çağırım hata yükseltirse, hata kendi türüyle değişmeden yayılır; kısmi
bir sonuç döndürülmez.

```ahd
values: List<String?> := ["a", null]
write(Lists.groupBy(values, lambda (value: String?) -> str(value)))
```

## Sıralama açık bir sözleşmedir

Bu sıralar garanti edilir, uygulama kazası değildir:

| İşlem | Sıra |
| --- | --- |
| `chunk` | kaynak sırası, ardışık dizilerde |
| `flatten` | dış sıra, sonra iç sıra |
| `transpose` | sütun indeksi, sonra satır indeksi |
| `unique` | ilk oluşum |
| `valueCounts` | her anahtarın ilk oluşumu |
| `groupBy` anahtarları | her anahtarın ilk oluşumu |
| `groupBy` üyeleri | kaynak sırası |

## Null olabilirlik

Yapısal null olabilirlik tam olarak korunur, asla silinmez ve asla düşürülmez:

```text
Lists.chunk(List<String?>, n)   -> List<List<String?>>
Lists.flatten(List<List<T?>>)   -> List<T?>
Lists.transpose(List<List<T?>>) -> List<List<T?>>
Lists.unique(List<String?>)     -> List<String?>
```

`Pair` anahtarları asla null olamaz; bu yüzden
`Lists.valueCounts(List<String?>)` geçersizdir ve `Lists.groupBy`'ın anahtar
`Function`'ı null olamayan bir anahtar türü döndürmelidir.

## Sığ yapısal anlambilim

Her işlem koleksiyon *yapısı* açısından saftır: kendisine verilen `List`'i
asla değiştirmez, bu yüzden bir `Constant List` güvenle geçirilebilir ve
döndürülen her `List` — dış ve iç — yeni, yapısal olarak bağımsız bir
koleksiyondur.

Dönüşüm **sığdır**. Başvurulan elemanlar derin kopyalanmaz, referansla
taşınır:

```ahd
boxes: List<Box> := [Box(label: "one"), Box(label: "two")]
parts: List<List<Box>> := Lists.chunk(boxes, 1)
parts[0][0].label = "changed"
write(boxes[0].label)          // changed
```

`List<Student>` parçalamak, *aynı* `Student` nesnelerini tutan yeni Listeler
üretir. Bu yapısal değişmezliktir, derin değişmezlik değil.

## Hatalar

`ListsError` doğrudan `Error`'dan türer ve modülün kendi yapısal
başarısızlıklarını kapsar:

- sıfır veya daha küçük bir `chunk` boyutu;
- düzensiz `transpose` girdisi.

Tür bakımından geçersiz çağrılar derleme zamanı tür hatası olarak kalır ve
asla `ListsError`'a ulaşmaz. Hata yükselten bir geri çağırım kendi hata
türünü korur — asla sarmalanmaz.

```ahd
attempt {
    write(Lists.transpose([[1, 2, 3], [4, 5]]))
}
except ListsError as error {
    write(error.message)
}
```

## Hedef olmayanlar

`Lists` genel amaçlı bir fonksiyonel koleksiyon kütüphanesi değildir.
Bilinçli olarak şunları eklemez:

- `zip` / `unzip` — AhdCode'da `Tuple` yoktur ve `Pair` anahtarları
  sınırlıdır, bu yüzden Python'un `zip` anlambiliminin henüz dürüst bir
  karşılığı yoktur;
- `sum` / `mean` / `min` / `max` — sayısal ve istatistiksel iş `Math`,
  `Statistics` ve `Numeric`'e aittir;
- özyinelemeli bir flatten, `invert` veya diğer spekülatif kolaylıklar;
- iteratörler, tembel koleksiyonlar, akışlar veya paralel koleksiyonlar.

## Ayrıca bakınız

- [KeyValue](KEYVALUE_TR.md) — `Pair` için aynı fikir
- [List API](LIST_API_TR.md) — çekirdek `List` üyeleri
- [Koleksiyonlar](COLLECTIONS_TR.md) — `List` ve `Pair` temelleri
- [Data](DATA_TR.md) — bu biçimler üzerine kurulu tablo verisi
- `examples/v0.1/47_lists.ahd`
