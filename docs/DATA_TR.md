# Data standart modülü

[English](DATA.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [KeyValue](KEYVALUE_TR.md) · [CSV](CSV_TR.md) · [Modüller](MODULES_TR.md)

İlk kez öğreniyorsanız filtreleme, sayısal sıralama, türetme, gruplama ve
Statistics'e geçişi tek akışta gösteren [Data atölyesini](PRACTICAL_MODULES_TR.md#2-data-string-tablosunu-şekillendirmek)
çalışın; bu sayfayı eksiksiz davranış referansı olarak kullanın.

Data; mevcut String, List, Pair, Function, Lambda ve CSV altyapısı üzerine
kurulmuş küçük, katı (strict) ve değiştirilemez (immutable) bir tablo
katmanıdır. Math, Time, Regex ve CSV gibi açıktır (explicit):

```ahd
bring Data
from Data bring Table
from Data bring DataError
```

Kanonik kimlik `builtin:Data`'dır; kardeş bir `Data.ahd` onu gölgeleyemez
(shadow). Her argüman `NonNull`'dır.

İş bölümü kasıtlıdır:

```text
CSV          metin taşıma (transport)
Data         tablo yapısı ve dönüşümü
sizin kodunuz  açık dönüşümle sayısal işler
```

## Her hücre bir String'dir

Bir `Table` hücresi her zaman `String`'dir. Data asla `Int`, `Real`, `Bool`,
`DateTime` veya `null` çıkarımı yapmaz ve bir değeri asla örtük olarak
dönüştürmez:

```text
"95"     String kalır
"3.14"   String kalır
"true"   String kalır
""       boş bir String kalır
```

Boş bir hücre boş bir `String`'tir. `null` **değildir** ve v0.1.12'de Data'nın
eksik-değer (missing value) modeli yoktur -- `NA` yok, `fillNull` yok,
`dropNull` yok.

Bir sayı istediğinizde, AhdCode'un her yerinde olduğu gibi açıkça dönüştürün:

```ahd
total: Int := int(row["score"])
weight: Real := real(row["weight"])
```

Sayısal bir List de böyle üretilir:

```ahd
values: List<Real> := table.column("score").map(
    lambda (value: String) -> real(value)
)
```

Kanonik satır `Pair<String, String>`, satır koleksiyonu ise
`List<Pair<String, String>>`'tir.

## Bir Table oluşturmak

```text
Data.fromRows(columns: List<String>, rows: List<List<String>>) -> Table
Data.fromRecords(records: List<Pair<String, String>>)          -> Table
Data.fromCSV(text: String, delimiter: String = ",")            -> Table
Data.readCSV(path: String, delimiter: String = ",")            -> Table
```

`Table` derleyici tarafından sağlanır: asla doğrudan oluşturulmaz, yalnızca bu
fonksiyonlardan veya başka bir Table işleminden gelir.

`fromRows` verdiğiniz sütun sırasını korur. Sütun isimleri boş olmamalı ve
benzersiz olmalıdır; her satırda tam olarak `len(columns)` hücre bulunmalıdır,
uyuşmazlık `DataError` fırlatır. Satırlar asla doldurulmaz (pad) veya kırpılmaz
ve hiçbir sütun ismi uydurulmaz. Sıfır satır geçerlidir ve şemayı korur.

`fromRecords`, **ilk** kaydı kanonik sütun sırası olarak alır. Sonraki kayıtlar
herhangi bir ekleme sırası kullanabilir ancak tam olarak aynı anahtar kümesini
taşımalıdır; eksik veya fazla bir anahtar `DataError` fırlatır. Değerler
kanonik sıraya kopyalanır ve çağıranın Pair'leri asla değiştirilmez. Boş
kayıtlar, hiçbir şema çıkarılamayacağı için boş, sıfır sütunlu bir Table
üretir.

```ahd
table: Table := Data.fromRecords([{"b": "2", "a": "1"}, {"a": "3", "b": "4"}])
write(table.columns())
```

=>

```text
["b", "a"]
```

## CSV entegrasyonu

`fromCSV` ve `readCSV`, CSV modülünün okuyucusunu yeniden kullanır; bu yüzden
tırnaklama, kaçışlı tırnaklar, gömülü yeni satırlar, LF/CRLF, Unicode ve özel
ayraçlar tam olarak [CSV](CSV_TR.md)'deki gibi davranır. Data ikinci bir CSV
grameri tanımlamaz.

İlk CSV satırı başlıktır. `CSV.parseRecords`'un aksine Data, yalnızca başlıktan
oluşan bir belgenin şemasını korur:

```ahd
table: Table := Data.fromCSV("name,score\n")
write(table.columns())
write(table.rowCount())
```

=>

```text
["name", "score"]
0
```

Boş CSV girdisi sıfır sütun ve sıfır satır üretir. Başlık isimleri boş olmamalı
ve benzersiz olmalıdır; her veri satırı başlık genişliğiyle eşleşmelidir.

## Değiştirilemezlik ve anlık görüntüler (snapshots)

Her Table işlemi saftır (pure). `select`, `drop`, `rename`, `filter`, `sort`,
`reverse`, `derive`, `transform`, `head` ve `tail` **yeni** bir Table döndürür
ve kaynağa dokunmaz. v0.1.12'de `setCell`, `appendRow`, `deleteRow` veya
yerinde (in-place) mod yoktur.

Bir Table'ın geri verdiği her şey taze bir anlık görüntüdür; bu yüzden bir
sonucu değiştirmek asla Table'a ulaşamaz:

```ahd
columns: List<String> := table.columns()
columns.add("injected")
write(table.columns())
```

Table'ın kendi sütunları değişmez. Aynı kural `row()`, `rows()` ve `column()`
için de geçerlidir. Table'ın iç depolaması yayınlanan bir öznitelik değildir:
okunamaz ve `has` onu bildirmez.

## Biçim ve erişim

```text
rowCount()            -> Int
columnCount()         -> Int
columns()             -> List<String>
rows()                -> List<Pair<String, String>>
row(index: Int)       -> Pair<String, String>
column(name: String)  -> List<String>
```

`row`, sıradan List indeks kurallarını izler: negatif indeks sondan sayar ve
geçersiz bir indeks `IndexError` fırlatır. Bilinmeyen bir sütun ismi, sessizce
boş bir değer döndürmek yerine `DataError` fırlatır.

## head ve tail

```text
head(count: Int = 5) -> Table
tail(count: Int = 5) -> Table
```

`rowCount()`'tan büyük bir sayı tüm satırları döndürür; sıfır, satırı olmayan
ama aynı sütunlara sahip bir Table döndürür. Negatif bir sayı `DataError`
fırlatır. Satır sırası korunur.

## select, drop ve rename

```text
select(columns: List<String>) -> Table
drop(columns: List<String>)   -> Table
rename(oldName: String, newName: String) -> Table
```

`select`, istenen sırayı çıktı sırası olarak kullanır. `drop`, kalan sütunların
özgün sırasını korur. Her ikisi de adı geçen her sütunun var olmasını gerektirir
ve istekte tekrarlanan bir ismi reddeder; hiçbiri bilinmeyen bir ismi sessizce
yok saymaz.

`rename` sütunun konumunu korur. Yeni isim boş olmamalıdır ve var olan farklı
bir sütunla çakışamaz.

## filter

```text
filter(function: Function) -> Table
```

Sözleşme tam olarak `(Pair<String, String>) -> Bool`'dur ve derleme zamanında
denetlenir. Geri çağırma (callback) bir satır anlık görüntüsü alır, kaynak
sırasında satır başına tam olarak bir kez çalışır ve kabul ettiği satırlar
kaynak sırasında korunur.

```ahd
adults: Table := table.filter(
    lambda (row: Pair<String, String>) -> int(row["age"]) >= 18
)
```

## sort

```text
sort(column: String)     -> Table
sort(function: Function) -> Table
```

`sort(column)`, o sütunun kararlı (stable), artan, sözlüksel String
sıralamasıdır. `sort(function)`, `Int`, `Real` veya `String` döndüren bir
anahtar Function kullanır ve List'in anahtarlı sıralama sözleşmesini yeniden
kullanır: kararlı, artan ve anahtar satır başına tam olarak bir kez çalışır.
Karşılaştırıcı (comparator) geri çağırma ve azalan bayrağı yoktur -- sayısal
bir anahtarı negatifleyin veya `reverse()` kullanın:

```ahd
ranked: Table := table.sort(
    lambda (row: Pair<String, String>) -> -int(row["score"])
)
```

## reverse

```text
reverse() -> Table
```

Satır sırasını ters çevirir ve sütunları değiştirmez.

## transform ve derive

```text
transform(column: String, function: Function) -> Table
derive(name: String, function: Function)      -> Table
```

`transform`, var olan bir sütunu `(String) -> String` aracılığıyla yeniden
yazar. Yalnızca o sütun değişir ve konumu korunur. Geri çağırma bir `String`
döndürmelidir; `Int` veya `Real`'den örtük dönüşüm yoktur.

```ahd
cleaned: Table := table.transform("name", lambda (value: String) -> value.trim())
```

`derive`, her tam satırdan `(Pair<String, String>) -> String` ile oluşturulan
yeni bir sütun ekler. İsim boş olmamalı ve zaten var olmamalıdır -- var olan
bir sütunu yeniden yazmak `derive`'ın değil, `transform`'un işidir.

```ahd
labelled: Table := table.derive(
    "status",
    lambda (row: Pair<String, String>) -> str(int(row["score"]) >= 60)
)
```

## unique, valueCounts ve groupBy

```text
unique(column: String)      -> List<String>
valueCounts(column: String) -> Pair<String, Int>
groupBy(column: String)     -> Pair<String, Table>
pivotCount(rows: String, columns: String) -> Table
```

Üçü de sütunun String hücresine göre anahtarlanır ve ilk-görülme sırasını
kullanır. Bir grup içindeki satırlar kaynak sırasını korur ve gruplanan her
Table kaynakla aynı şemaya sahiptir. Bilinmeyen bir sütun `DataError` fırlatır;
boş bir tablo boş bir sonuç üretir.

```ahd
groups: Pair<String, Table> := table.groupBy("department")

for department in groups {
    group: Local Table := groups[department]
    write("{department}: {group.rowCount()}")
}
```

Toplama (aggregation) sözdizimi kasıtlı olarak yoktur; bir grup sıradan bir
Table'dır.

## pivotCount

```text
pivotCount(rows: String, columns: String) -> Table
```

Katı bir sayım çapraz tablosu (cross-tabulation): `rows` sütununun her farklı
değeri için bir satır, `columns` sütununun her farklı değeri için üretilmiş bir
sütun ve her hücrede o kombinasyondaki kaynak satır sayısı.

```ahd
students: Table := Data.fromCSV(
    "name,department,grade\nAli,Math,A\nAyse,Physics,B\nMehmet,Math,A\nZeynep,Physics,A\n"
)

write(students.pivotCount("department", "grade").toCSV())
```

=>

```text
department,A,B
Math,2,0
Physics,1,1
```

Her iki eksen de `groupBy` ve `valueCounts` ile eşleşerek ilk-görülme sırasını
kullanır; bu yüzden sonuç asla map yineleme sırasına bağlı değildir. Bulunmayan
bir kombinasyon `"0"` sayılır -- bu, eksik veri değil, sayım anlambilimidir.
Sayımlar diğer her hücre gibi `String` hücrelerdir; bu yüzden bir program
üzerlerinde aritmetik yapmak için açıkça dönüştürür. Argümanlar konumsaldır
(positional), çünkü yerleşik bir tür işlemi adlandırılmış argüman almaz; bu
metot için yeni bir sözdizimi eklenmemiştir.

Bilinmeyen bir sütun, onu isimlendiren bir `DataError` fırlatır ve her iki
eksen için aynı sütunu vermek, sessizce bir köşegen üretmek yerine reddedilir.

`pivotCount` kasıtlı olarak tek çapraz tablodur. Genel bir pivot değildir:
toplama (aggregation) callback'i, değer sütunu, multi-index veya eksik-değer
modeli yoktur.

## toCSV ve writeCSV

```text
toCSV(delimiter: String = ",")                  -> String
writeCSV(path: String, delimiter: String = ",") -> Nothing
```

İkisi de CSV modülünün yazıcısını kullanır; bu yüzden tırnaklama ve ayraçlar
tam olarak `CSV.stringify` ile eşleşir. Çıktı, Table sütun sırasında başlık
satırı ve ardından veri satırlarıdır. Yalnızca başlıktan oluşan bir Table
başlığını yazar; sıfır sütunlu, sıfır satırlı bir Table `""` üretir.

## Hatalar

`DataError` doğrudan `Error`'dan türer ve Data'ya özgü yapısal başarısızlıkları
kapsar: tekrarlanan veya boş sütun ismi, satır genişliği uyuşmazlığı, kayıt
anahtar kümesi uyuşmazlığı, bilinmeyen sütun, tekrarlanan `select`/`drop`
isteği, negatif `head`/`tail` sayısı ve zaten var olan bir `derive` hedefi.

Diğer alanlar kendi hata türlerini korur; böylece hata, gerçekten başarısız
olan katmanı isimlendirir:

| başarısızlık | hata |
|---|---|
| Data/Table yapısı | `DataError` |
| CSV sözdizimi veya geçersiz ayraç | `CSVError` |
| `readCSV`/`writeCSV` dosya sistemi erişimi | `FileError` / `IOError` |
| geçersiz `row()` indeksi | `IndexError` |

```ahd
attempt {
    write(table.column("age"))
} except DataError as error {
    write(error.message)
}
```

## Bir değer olarak Table

Table sıradan bir Class referansıdır. `type(table)`, `"Table"` bildirir;
`id()` ve `same`, diğer herhangi bir referansta olduğu gibi davranır. Table,
`CEqual`, `CCompare` veya `CStr` uygulamaz; bu yüzden `==` ve `same` sıradan
referans kimliğini korur -- Data, tablolar için değer eşitliği icat etmez.

## Data ne değildir

Data pandas **değildir** ve DataFrame uyumlu değildir. v0.1.12'de kasıtlı
olarak join, merge, concat, pivot, melt, MultiIndex, indeks etiketleri, sorgu
dizeleri, SQL, pencere (window) fonksiyonları, rolling, resample, kategorik
dtype, tembel (lazy) yürütme veya ifade ağaçları yoktur. Şema çıkarımı,
otomatik sayısal veya tarih ayrıştırma ve null çıkarımı yoktur.

İstatistik de yoktur. `sum`, `mean`, `median`, `variance`, `stdev`, `quantile`,
`correlation` ve `describe`, planlanan Statistics katmanına aittir; o katman
`List<Int>` ve `List<Real>` tüketecektir -- ki bu, açık bir dönüşümün zaten
ürettiği şeydir:

```ahd
scores: List<Real> := table.column("score").map(
    lambda (value: String) -> real(value)
)
```

Sayısal işleri açık tutmak, Data'nın dinamik bir değer sistemi hâline gelmek
yerine statik tipli kalmasını sağlayan şeydir.
