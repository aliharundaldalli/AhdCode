# Dil turu

[English](LANGUAGE_TOUR.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Türler ve null](TYPES_AND_NULL_TR.md)

## Bildirimler ve değişiklik (mutation)

`:=` bir bağlama (binding) oluşturur. `=` mevcut bir bağlamayı değiştirir.

```ahd
name := "Ali"       // String olarak çıkarılır, hâlâ statik olarak tiplenmiş
count: Int := 3     // açık tür belirtimleri hâlâ kullanılabilir
count = 4
```

Çıkarım (inference), tam statik türü korur. `count = "four"` hâlâ bir
hatadır. Yalın `value := null` geçersizdir çünkü temel bir tür çıkarılamaz;
`value: User? := null` yazın.

Satır sonları aksi halde hiçbir anlam taşımaz -- girinti (indentation)
yalnızca okunabilirlik içindir ve `ahdcode format`
([Formatter](FORMATTER_TR.md)'a bakın) onu sizin için seçer -- ancak `:=`
veya `=`'nin sağındaki değer, işaretle aynı fiziksel satırda başlamalıdır:

```ahd
scores: List<Int> := [
    1
    2
]
```

bu geçerlidir (`[`, `:=`'den hemen sonra açılır), oysa `[`'i `:=`'den sonra
kendi satırında yazmak, genel bir ayrıştırma hatası yerine özel bir hatayla
reddedilir.

Çalıştırılabilir, iç içe geçmiş bir kapsam içinde, yeni bir bildirim
`Local` kullanır:

```ahd
if count > 0 {
    message: Local := "Ready"
    write(message)
}
```

Bir modül-kökü bağlamasını okuyan veya değiştiren bir Function, bu erişimi
`Global` ile bildirir:

```ahd
counter: Int := 0

increase: Function := (
) -> Nothing {
    counter: Global Int
    counter++
}
```

## Değerler ve koleksiyonlar

```ahd
score: Int := 90
average: Real := 87.5
passed: Bool := score >= 50
student: String := "Ayşe"

scores: List<Int> := [90, 85, 92]
grades: Pair<String, Int> := {
    "Ali": 90
    "Ayşe": 92
}
```

List'ler ve Pair'ler referans nesneleridir. Alias'lar değişikliği görür. Bir
`Constant` referans, ulaşılabilir nesne grafiğini derin dondurur.

## Koşullar ve döngüler

```ahd
if passed {
    write("Passed")
}
else {
    write("Try again")
}

for value in between(1, 4) {
    write(value)
}
```

Yalnızca `Bool` bir koşuldur; AhdCode'da truthiness yoktur.

## Function'lar ve Class'lar

```ahd
square: Function := (
    value: Int
) -> Int {
    return value^2
}
```

Tek bir ifade için `lambda`, aynı `Function` türünde bir değer oluşturur.
Parametreler açıkça tiplenir ve dönüş türü çıkarılır:

```ahd
squareShort := lambda (value: Int) -> value^2
values := [1, 2, 3]
squares := values.map(lambda (value: Int) -> value^2)
```

Lambda yalnızca bir ifade içerir: blok/deyim lambda'sı, ayrı bir Lambda türü
ve örtük zorlama yoktur.

Eğer lambda, kendi parametreleri dışındaki bir değişkeni kullanmak zorundaysa, bu değişkenleri parametrelerden hemen önce köşeli parantez içinde listelemelisiniz:

```ahd
minimum: Int := 50
passed := values.filter(lambda [#minimum] (score: Int) -> score >= minimum)
```

`Local` (değere göre) yakalama için `#` ve canlı `Global` bağımlılık için `@` kullanın. Hiçbir dış bağımlılık kendiliğinden dahil edilmez.

```ahd
Student: Class<> := {
    structure: Attributes := (
        name: String
        number: Constant Int
    )

    describe: Function := (
    ) -> String {
        return "{attribute.number}: {attribute.name}"
    }
}
```

Bir Class, `==`, sıralama, aritmetik, tekli (unary) `-` ve `str()`
davranışını on tam sayıda
[Class Protocol Methods](PROTOCOLS_TR.md) (`CEqual`, `CCompare`, `CAdd`,
`CSubtract`, `CMultiply`, `CDivide`, `CRemainder`, `CPower`, `CNegate`,
`CStr`) aracılığıyla tanımlayabilir -- sıradan Function sözdizimi kullanan
olağan bir metot:

```ahd
Vector2: Class<> := {
    structure: Attributes := (x: Real, y: Real)
    CAdd: Function := (
        other: Vector2
    ) -> Vector2 {
        return Vector2(x: attribute.x + other.x, y: attribute.y + other.y)
    }
}
```

## Hatalar ve modüller

Yakalanabilir hatalar için `attempt`, `except`, `ultimately` ve `toss`
kullanın. Bir isim uzayı (namespace) için `bring ModülAdı`, doğrudan bir
sembol için `from ModülAdı bring isim` kullanın. Yerel modüller kardeş
(sibling) dosyalardır. `Math`, `Time`, `Latex`, `Word`, `Excel`, `PDF`, `Archive`,
`Path`, `Regex`, `CSV`, `Data`, `File`, `Statistics`, `Plot`, `Numeric`, `JSON`,
`SQLite`, `XML`, `Env`, `Lists` ve `KeyValue` açık standart modüllerdir; alan ve dosya
hataları yakalanabilir AhdCode hatalarıdır.

Devamı için [Fonksiyonlar](FUNCTIONS_TR.md), [Sınıflar](CLASSES_TR.md),
[Class Protocol Methods](PROTOCOLS_TR.md), [Koleksiyonlar](COLLECTIONS_TR.md)
ve [Modüller](MODULES_TR.md)'e bakın. Time yerel, UTC ve sabit-ofsetli anları;
CSV String satırları ve kayıtlarını kapsar; Data ise bunları String
hücrelerden oluşan değiştirilemez bir `Table`'a dönüştürür; JSON ve XML tipli
yapılandırılmış-veri modelleridir, SQLite tipli bir yerel-veritabanı köprüsüdür,
Env ise işlem/`.env` yapılandırmasını okur;
Lists ve KeyValue ise `List` ve `Pair` üzerindeki saf yapısal dönüşüm
katmanıdır; Excel tipli, değiştirilemez XLSX çalışma kitapları ekler; PDF
değiştirilemez belgeler oluşturur ve bunları çevrimdışı gerçek `.pdf`
dosyalarına render eder (Latex'in konuşlandırılmış Tectonic render motorunu
paylaşarak), `PDF.fromWord`/`PDF.fromExcel` anlamsal dönüşümüyle birlikte;
Archive ise dosyaları yalnızca Go standart kütüphanesini kullanarak, yalnızca
oluşturma amaçlı, gerçek ZIP/TAR/TAR.GZ arşivlerine paketler. Ayrıca
[Time](TIME_TR.md), [CSV](CSV_TR.md), [Data](DATA_TR.md), [Statistics](STATISTICS_TR.md), [Plot](PLOT_TR.md), [Numeric](NUMERIC_TR.md), [Word](WORD_TR.md), [Excel](EXCEL_TR.md), [PDF](PDF_TR.md), [Archive](ARCHIVE_TR.md), [JSON](JSON_TR.md), [SQLite](SQLITE_TR.md), [XML](XML_TR.md), [Env](ENV_TR.md), [Lists](LISTS_TR.md), [KeyValue](KEYVALUE_TR.md) ve [tanılama rehberine](DIAGNOSTICS_TR.md) bakın.
