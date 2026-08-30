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
ve örtük zorlama yoktur. v0.1.10'da çevreleyen bir `Local` bağlamayı
yakalayamaz; normal bir Function kullanın veya değeri açıkça geçirin. Normal
Function bildirim sözdizimi değişmemiştir.

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
(sibling) dosyalardır. `Math`, `Time`, `Latex`, `Path`, `Regex` ve `File`
açık standart modüllerdir; `File` ve `Regex` hataları yakalanabilir.

Devamı için [Fonksiyonlar](FUNCTIONS_TR.md), [Sınıflar](CLASSES_TR.md),
[Class Protocol Methods](PROTOCOLS_TR.md), [Koleksiyonlar](COLLECTIONS_TR.md)
ve [Modüller](MODULES_TR.md)'e bakın.
