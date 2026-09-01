# KeyValue standart modülü

[English](KEYVALUE.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Lists](LISTS_TR.md) · [Koleksiyonlar](COLLECTIONS_TR.md)

`KeyValue`, derleyici tarafından kayıtlı `builtin:KeyValue` modülüdür.
Açıktır ve kardeş bir `KeyValue.ahd` dosyası onu gölgeleyemez:

```ahd
bring KeyValue
from KeyValue bring KeyValueError
```

## Çekirdek tür Pair olarak kalır

`KeyValue` kendine ait hiçbir kapsayıcı getirmez. `Dictionary` yoktur, `Map`
yoktur, `Record` yoktur, `Struct` yoktur, `Tuple` yoktur ve `Any` anahtarlı
nesne torbası yoktur. Dilin mevcut sıralı, homojen `Pair<K, V>` türü üzerinde
çalışır ve her zaman bir `Pair` veya bir `List` döndürür.

`Pair`, Python'un `dict[Any, Any]`'si değildir: sıralıdır, anahtarları
`String`, `Int` veya `Bool`'dur ve asla null olamaz, değer türü ise tüm `Pair`
için tek bir türdür. `KeyValue` bunun üzerine kurulu saf bir dönüşüm
katmanıdır, fazlası değil.

## Yüzey

```text
KeyValue.keys(pair: Pair<K, V>)                  -> List<K>
KeyValue.values(pair: Pair<K, V>)                -> List<V>
KeyValue.combine(keys: List<K>, values: List<V>) -> Pair<K, V>

KeyValue.with(pair: Pair<K, V>, key: K, value: V) -> Pair<K, V>
KeyValue.without(pair: Pair<K, V>, key: K)        -> Pair<K, V>

KeyValue.select(pair: Pair<K, V>, keys: List<K>)  -> Pair<K, V>
KeyValue.drop(pair: Pair<K, V>, keys: List<K>)    -> Pair<K, V>
KeyValue.rename(pair: Pair<K, V>, oldKey: K, newKey: K) -> Pair<K, V>

KeyValue.mapValues(pair: Pair<K, V>, transform: Function(V) -> U) -> Pair<K, U>

KeyValue.merge(left: Pair<K, V>, right: Pair<K, V>)     -> Pair<K, V>
KeyValue.overlay(base: Pair<K, V>, changes: Pair<K, V>) -> Pair<K, V>
```

## İşlemler tür-yönelimlidir

`K`, `V` ve `U` AhdCode sözdizimi değildir; kullanıcıya açık genel `Function`
sözdizimi yoktur ve `KeyValue.keys<String>(...)` gibi bir şey asla yazılmaz.
Derleyici her çağrının sonuç türünü gerçekten yazılan argüman türlerinden
hesaplar:

```ahd
byName: Pair<String, Int> := {"a": 1}
byNumber: Pair<Int, String> := {1: "a"}

names: List<String> := KeyValue.keys(byName)
numbers: List<Int> := KeyValue.keys(byNumber)
```

Hiçbir şey silinmez ve `Pair` değişmezliği (invariance) korunur: bir
`Pair<String, Int>`, bu işlemlerden geçerek asla sessizce
`Pair<String, Real>` haline gelmez.

Her iki yazım da çalışır:

```ahd
bring KeyValue
from KeyValue bring combine

record := combine(["name"], ["Ali"])
```

### Tek sınır

Sonuç türü her çağrı yerinde hesaplandığı için bu işlemlerin tek bir somut
`Function` türü yoktur ve özelleştirilmemiş bir `Function` değeri olarak
saklanamazlar:

```ahd
stored := KeyValue.keys        // derleme zamanında reddedilir
```

Doğrudan çağırın ya da istediğiniz kesin biçimi kendi `Function`'ınıza sarın.

## keys ve values

```ahd
record: Pair<String, String> := {"name": "Ali", "score": "91"}
write(KeyValue.keys(record))
write(KeyValue.values(record))
```

```text
["name", "score"]
["Ali", "91"]
```

İkisi de değerleri `Pair` ekleme sırasında döndürür ve ikisi de taze `List`
anlık görüntüleridir: sonucu değiştirmek asla `Pair`'e ulaşmaz.

```ahd
snapshot := KeyValue.keys(record)
snapshot.add("injected")
write(record)                  // değişmedi
```

`values`, `Pair`'in değer null olabilirliğini korur: `Pair<K, V?>`,
`List<V?>` verir.

## combine

```ahd
record := KeyValue.combine(
    ["name", "score", "department"]
    ["Ali", "91", "Mathematics"]
)
```

```text
{"name": "Ali", "score": "91", "department": "Mathematics"}
```

Ekleme sırası anahtar `List`'ini izler. İki List'in uzunluğu tam olarak aynı
olmalıdır — hiçbir şey doldurulmaz, hiçbir şey kırpılmaz — ve tekrarlanan bir
anahtar, son-yazan-kazanır ile çözülmek yerine reddedilir. İkisi de
`KeyValueError`'dır.

`K` mevcut `Pair` anahtar kurallarını sağlamalıdır; bu yüzden `List<Real>`
veya `List<String?>` bir anahtar listesi derleme zamanı hatasıdır. Değer null
olabilirliği korunur: `combine(List<K>, List<V?>)`, `Pair<K, V?>` verir. İki
boş tipli List, doğru türde boş bir `Pair` verir.

## with

`with`, `pair[key] = value` ifadesinin saf karşılığıdır.

```ahd
base: Pair<String, String> := {"name": "Ali", "score": "90"}
updated := KeyValue.with(base, "score", "95")

write(base)      // {"name": "Ali", "score": "90"}
write(updated)   // {"name": "Ali", "score": "95"}
```

Var olan bir anahtar tam konumunu korur ve yeni değeri alır — hiçbir şey
yeniden sıralanmaz. Olmayan bir anahtar sona eklenir. Özgün `Pair` asla
değiştirilmez, bu yüzden bir `Constant Pair` güvenle geçirilebilir.

## without

```ahd
write(KeyValue.without({"a": 1, "b": 2, "c": 3}, "b"))
```

```text
{"a": 1, "c": 3}
```

Kalan her girdi kaynak sırasını korur. Olmayan bir anahtar, dilin mevcut
`KeyError`'ını yükseltir; bu `Pair`'in kendi `eject` anlambilimiyle
eşleşir — istek asla sessizce yok sayılmaz.

## select

```ahd
write(KeyValue.select({"a": 1, "b": 2, "c": 3}, ["c", "a"]))
```

```text
{"c": 3, "a": 1}
```

`select`'in çıktı sırası kaynak `Pair` sırasını değil, **istenen anahtar
List'ini** izler. Onu bir yeniden sıralama ve izdüşüm aracı yapan şey budur.

Bilinmeyen bir anahtar `KeyError` yükseltir. İki kez istenen bir anahtar
`KeyValueError` yükseltir: tekrarlanan bir istek, sessizce tekilleştirilecek
bir şey değil, çağıranın verisindeki bir hatadır. Boş bir anahtar `List`'i
doğru türde boş bir `Pair` verir.

## drop

```ahd
write(KeyValue.drop({"a": 1, "b": 2, "c": 3}, ["b"]))
```

```text
{"a": 1, "c": 3}
```

`drop`, korunan her girdi için **kaynak** `Pair`'in sırasını korur.
Bilinmeyen bir anahtar `KeyError`, tekrarlanan bir istek `KeyValueError`
yükseltir; gerekçeleri `select` ile aynıdır. Boş bir anahtar `List`'i özgün
`Pair`'in yapısal bir kopyasını döndürür.

## rename

```ahd
write(KeyValue.rename({"a": 1, "b": 2, "c": 3}, "b", "middle"))
```

```text
{"a": 1, "middle": 2, "c": 3}
```

Yeniden adlandırılan girdi tam olarak eski konumunu ve değerini korur.
Olmayan bir eski anahtar `KeyError` yükseltir. Yeni anahtar zaten *başka* bir
girdiye aitse, bu ikisinden birini sessizce atmak yerine `KeyValueError`
yükseltir. Bir anahtarı kendisiyle yeniden adlandırmak zararsız bir işlemdir
ve taze, yapısal olarak eşdeğer bir `Pair` döndürür.

## mapValues

```ahd
write(KeyValue.mapValues({"a": "10", "b": "20"}, lambda (value: String) -> int(value)))
```

```text
{"a": 10, "b": 20}
```

Anahtar kümesi ve sırası değişmez; yalnızca değer türü değişir, `V -> U`.
Geri çağırım, `Pair` ekleme sırasında değer başına tam olarak bir kez çalışır
ve hataları kendi türüyle değişmeden yayılır. `Nothing` döndüren bir geri
çağırım derleme zamanı hatasıdır.

Geri çağırımın parametresi `Pair`'in değer türüyle — null olabilirliği dahil —
eşleşmelidir ve geri çağırımın kendi sonuç null olabilirliği korunur:

```text
KeyValue.mapValues(Pair<K, V>, Function(V) -> U?)  -> Pair<K, U?>
```

## merge ve overlay

Bunlar bilinçli olarak iki ayrı fonksiyondur, `Bool` bayraklı tek bir
fonksiyon değil; çünkü okuyucuya hangi niyetin kastedildiğini söyleyen şey
addır.

**`merge` güvenli bir ayrık birleşimdir.** İki `Pair`'de birden bulunan bir
anahtar `KeyValueError`'dır — modül sessizce sol-kazanır veya sağ-kazanır
seçimi yapmaz:

```ahd
write(KeyValue.merge({"a": 1, "b": 2}, {"c": 3}))
```

```text
{"a": 1, "b": 2, "c": 3}
```

```ahd
write(KeyValue.merge({"a": 1}, {"a": 9}))     // KeyValueError
```

Sıra önce sol sıra, sonra sağ sıradır.

**`overlay` açıkça adlandırılmış değişiklikler-kazanır işlemidir:**

```ahd
write(KeyValue.overlay({"a": 1, "b": 2}, {"b": 9, "c": 3}))
```

```text
{"a": 1, "b": 9, "c": 3}
```

Var olan bir taban anahtarı konumunu korur ve yeni değeri alır; yalnızca
`changes` içinde olan bir anahtar `changes` ekleme sırasında sona eklenir.
Hiçbir kaynak değiştirilmez.

İkisi de değer null olabilirliği dahil tam olarak aynı `Pair` türünü ister.

## Sıralama açık bir sözleşmedir

| İşlem | Sıra |
| --- | --- |
| `keys`, `values` | `Pair` ekleme sırası |
| `combine` | anahtar List sırası |
| `with`, var olan anahtar | mevcut konumu |
| `with`, yeni anahtar | sona eklenir |
| `without` | kalanların kaynak sırası |
| `select` | istenen anahtar List sırası |
| `drop` | kalanların kaynak sırası |
| `rename` | özgün anahtarın konumu |
| `mapValues` | özgün anahtar sırası |
| `merge` | sol sıra, sonra sağ sıra |
| `overlay`, var olan anahtar | taban konumu |
| `overlay`, yeni anahtar | `changes` sırasında sona eklenir |

## Null olabilirlik

`Pair` anahtarları asla null olamaz. Değer null olabilirliği yapısaldır ve tam
olarak korunur:

```text
KeyValue.values(Pair<K, V?>)                 -> List<V?>
KeyValue.combine(List<K>, List<V?>)          -> Pair<K, V?>
KeyValue.mapValues(Pair<K, V>, V -> U?)      -> Pair<K, U?>
```

`KeyValue.with(pair, key, null)` yalnızca `Pair`'in değer türü null olabilir
olduğunda kabul edilir.

## Sığ yapısal anlambilim

Her işlem koleksiyon *yapısı* açısından saftır: kendisine verilen `Pair` veya
`List`'i asla değiştirmez, bu yüzden bir `Constant` koleksiyon güvenle
geçirilebilir ve döndürülen her koleksiyon yapısal olarak bağımsızdır.

Dönüşüm **sığdır**. Anahtarlar ve değerler derin kopyalanmaz, referansla
taşınır:

```ahd
byName: Pair<String, Student> := {"ali": student}
copied := KeyValue.with(byName, "ayse", other)
write(copied["ali"] same student)     // true
```

`with`, `select`, `drop` ve diğerleri `Pair` yapısını kopyalar, `Student`
nesnesini değil. Bu yapısal değişmezliktir, derin değişmezlik değil.

## Hatalar

`KeyValueError` doğrudan `Error`'dan türer ve modülün kendi yapısal
başarısızlıklarını kapsar:

- `combine` uzunluk uyuşmazlığı;
- tekrarlanan bir `combine` anahtarı;
- tekrarlanan bir `select` veya `drop` isteği;
- var olan farklı bir anahtarla `rename` çakışması;
- `merge` anahtar çakışması.

Gerçekten **olmayan** bir `Pair` anahtarı dilin mevcut `KeyError`'ını
kullanır; böylece `without`, `select`, `drop` ve `rename` olmayan bir anahtarı
tam olarak `Pair`'in kendisinin zaten yaptığı gibi bildirir.

Tür bakımından geçersiz çağrılar derleme zamanı tür hatası olarak kalır. Hata
yükselten bir geri çağırım kendi hata türünü korur — asla sarmalanmaz.

## String cerrahisi olmadan JSON güncelleme

Bir `JSONValue` nesnesi sıradan bir `Pair<String, JSONValue>`'dur; bu yüzden
`KeyValue`, tipli gösterimden hiç çıkmadan bir JSON belgesini günceller:

```ahd
object: Pair<String, JSONValue> := data.object()

updatedObject: Pair<String, JSONValue> := KeyValue.with(
    object
    "books"
    JSON.array(newBooks)
)

JSON.write(JSON.object(updatedObject), "library.json", true)
```

Diğer tüm kök alanlar dokunulmadan kalır ve konumlarını korur. Bu,
`JSON.stringify` → String birleştirme → `JSON.parse` gidiş-dönüşünü tamamen
ortadan kaldırır; bkz. `examples/v0.1/49_json_record_update.ahd`.

## Hedef olmayanlar

`KeyValue` bilinçli olarak şunları eklemez:

- `entries` — AhdCode'da bir "girdi" için gerçek bir tür olana kadar dürüst
  bir karşılığı yoktur; tek elemanlı bir `Pair` bu değildir;
- `invert` veya diğer spekülatif kolaylıklar;
- herhangi bir heterojen kapsayıcı, `Tuple`, `Set` veya `Any` tipli torba;
- JSON'a, Data'ya veya SQL'e özgü davranış. `KeyValue` genel bir `Pair`
  modülü olarak kalır; böylece JSON, Data, CSV ve gelecekteki depolama
  modülleri her biri kendi dönüşüm katmanını icat etmeden onu yeniden
  kullanabilir.

## Ayrıca bakınız

- [Lists](LISTS_TR.md) — `List` için aynı fikir
- [Koleksiyonlar](COLLECTIONS_TR.md) — `List` ve `Pair` temelleri
- [JSON](JSON_TR.md) — tipli JSON belgeleri
- [Data](DATA_TR.md) — `Pair<String, String>` satırları üzerine kurulu tablo verisi
- `examples/v0.1/48_key_value.ahd`, `examples/v0.1/49_json_record_update.ahd`,
  `examples/v0.1/50_data_records.ahd`
