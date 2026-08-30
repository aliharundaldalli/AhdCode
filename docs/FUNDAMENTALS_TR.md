# Temel İşlevler (Fundamentals)

[English](FUNDAMENTALS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [String API](STRING_API_TR.md) · [List API](LIST_API_TR.md)

Bu isimler her modülde önceden bildirilmiştir (predeclared) ve `bring`
gerektirmez:

```text
write take str int real len clear between abs sum min max type id
```

| Fonksiyon | Davranış |
|---|---|
| `write(value)` | bir değeri, ardından bir satır sonu (newline) yazar |
| `take()` / `take(prompt)` | bir satırı String olarak okur |
| `str(value)` | kanonik, deterministik metin |
| `int(Real)` | sıfıra doğru keser (truncate) |
| `int(String)` | katı (strict), işaretli ASCII-ondalık ayrıştırma |
| `real(Int)` | açık, güvenli genişletme (widening) |
| `real(String)` | katı ondalık/kesir/üs (exponent) ayrıştırması |
| `len(value)` | String karakterleri, List elemanları veya Pair girdileri |
| `clear(collection)` | List veya Pair'i yerinde boşaltır |
| `between(...)` | tembel (lazy), bitişi hariç tutan Int yinelemesi |
| `abs(number)` | tam sonuç türüyle sayısal büyüklük (magnitude) |
| `sum(list)` | sayısal indirgeme (reduction); boş List `0` veya `0.0` verir |
| `min(list)` / `max(list)` | sayısal uç değerler; boş List `DomainError` fırlatır |
| `type(value)` | kanonik AhdCode tür adı, String olarak |
| `id(reference)` | bir Class örneğinin, List'in veya Pair'in opak çalışma zamanı kimliği |

Dönüşümler, String girdisindeki çevreleyen Unicode boşluklarını kırpar
(trim). `int(String)` kesirleri, üsleri, alt çizgileri (underscore) veya
taban (base) öneklerini kabul etmez. `real(String)` ondalık tam sayı, kesir
ve üs biçimlerini kabul eder ama NaN veya sonsuzluğu (infinity) kabul etmez.
Geçersiz metin `DomainError` fırlatır; aralık dışı metin `OverflowError`
fırlatır.

`str`, List sırasını ve Pair ekleme sırasını korur, iç içe geçmiş String'leri
tırnak içine alır, özellikleri (attributes) asla göstermez ve bir Class
örneğini, o Class `CStr` [Class Protocol Method](PROTOCOLS_TR.md)'unu
uygulamadıkça `<ClassName>` olarak yazdırır; uyguluyorsa `str` (ve bu yüzden
aynı dönüşümü paylaşan `write` ile String interpolasyonu) onun yerine ona
yönlendirilir (dispatch).

`clear`, mevcut koleksiyonu değiştirir, bu yüzden alias'lar bunu görür ve
Constant koleksiyonlar bunu reddeder. Sayısal indirgemeler saf okumalardır
(pure reads) ve null olmayan bir Constant List'i kabul eder.

## `type(value)`

`type(value) -> String`, özellikle REPL'de faydalı bir çalışma
zamanı/içgözlem (introspection) aracıdır. Bir reflection çerçevesi değildir:
kanonik AhdCode tür adını bir String olarak döndürür, asla birinci sınıf bir
Type nesnesi veya bir Go uygulama adı döndürmez.

```ahd
write(type(5))          // "Int"
write(type(5.0))        // "Real"
write(type("Ali"))      // "String"
write(type(true))       // "Bool"
write(type(null))       // "Null"

numbers: List<Int> := [1, 2]
write(type(numbers))    // "List<Int>"
```

Bir Class örneği için `type`, statik olarak bildirilen türü değil, **en
türetilmiş (most-derived) çalışma zamanı Class adını** bildirir:

```ahd
Animal: Class<> := { structure: Attributes := (name: String) }
Dog: Class<Animal> := { structure: Attributes := (SuperClass.attributes) }

pet: Animal := Dog(name: "Rex")
write(type(pet))        // "Animal" değil, "Dog"
```

Şu anda null olmayan bir değer tutan null olabilen bir değer için, `type`,
bildirilen türün `?`'sini değil, o değerin kendi türünü bildirir:

```ahd
x: Int? := 5
write(type(x))          // "Int"
```

`type(null)` her zaman `"Null"` bildirir. Bu, Fundamental'ın kendi içindeki
içkin (intrinsic) bir durumdur, yeni bir kaynak seviyesi `Null` bildirim
türü değildir -- `x := null`, v0.1.7'deki gibi hâlâ reddedilir.

## `id(reference)`

`id(reference) -> Int`, hata ayıklama (debugging), günlükleme (logging) ve
içgözlem için çalışma zamanı tarafından yönetilen bir kimlik numarası
döndürür. Bir bellek adresi **değildir** ve mevcut süreç veya REPL
oturumunun ötesinde hiçbir garanti taşımaz.

v0.1.8'de yalnızca anlamlı bir AhdCode kimliğine sahip referans değerleri
kabul edilir: bir Class örneği, bir List veya bir Pair. Bir ilkel (primitive)
değerin (`Int`, `Real`, `Bool`, `String`) bildirecek bir kimliği yoktur ve
bu bir derleme zamanı hatasıdır:

```ahd
id(5)       // derleme zamanı hatası
id("Ali")   // derleme zamanı hatası
```

```ahd
a := [1, 2]
b := a
c := [1, 2]

write(id(a) == id(b))   // true -- aynı nesne
write(id(a) == id(c))   // false -- ayrı ayrı tahsis edilmiş, farklı nesneler

a.add(3)
write(id(a) == id(b))   // hâlâ true -- değişiklik kimliği asla değiştirmez
```

Sayı opaktır ve süreç/oturuma özgüdür (process/session-local): bir bellek
adresi değildir, ayrı program çalıştırmaları arasında kararlı olacağı garanti
edilmez, serileştirme (serialization) verisi değildir ve kalıcı bir veritabanı
tanımlayıcısı değildir. Sıralamasına veya herhangi bir belirli başlangıç
değerine bağlı kalmayın.

`id`, null olabilen bir referans üzerindeki diğer her işlem gibi, kanıtlanmış
`NonNull` bir argüman gerektirir:

```ahd
user: User? := fetchUser()
id(user)                 // derleme zamanı hatası: user null olabilir
if user != null {
    write(id(user))      // daraltıldıktan (narrowed) sonra sorun yok
}
```

`id`, `same`'in yerini almaz. `a same b`, günlük AhdCode kodunda kullanılan
sıradan programatik kimlik testidir; `id(a)` özellikle o kimliği yazdırmak
veya günlüklemek için vardır. Herhangi iki canlı List, Pair veya Class
örneği değeri için `(a same b) == (id(a) == id(b))`.
