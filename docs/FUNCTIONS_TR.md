# Fonksiyonlar

[English](FUNCTIONS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Sınıflar](CLASSES_TR.md)

Function'lar modül kökündeki (module root) isimlendirilmiş bildirimlerdir;
metotlar bir Class içindeki isimlendirilmiş bildirimlerdir. v0.1'de iç içe
Function bildirimleri veya lambda'lar yoktur.

```ahd
greet: Function := (
    name: String
    title: String := "Student"
) -> String {
    return "Hello {title} {name}"
}
```

Parametreler arasındaki virgüller, aralarında zaten bir satır sonu varsa
isteğe bağlıdır ve sondaki virgüle her zaman izin verilir -- tek satırda
`(name: String, title: String := "Student")`, aynı bildirimdir. `ahdcode
format` ([Formatter](FORMATTER_TR.md)'a bakın), hangi yazımı kullanırsanız
kullanın, tek önerilen stile göre işler: sığıyorsa tek satır, aksi halde
hiç virgül olmadan yukarıdaki gibi her parametre kendi satırında.

Çağrılar ya tamamen sıralı (positional) ya da tamamen isimlendirilmiş
(named) olmalıdır:

```ahd
write(greet("Ali"))
write(greet(name: "Ali", title: "Dr"))
```

`greet("Ali", title: "Dr")` geçersizdir çünkü iki biçimi karıştırır. Zorunlu
parametreler varsayılan (default) parametrelerden önce gelir.

## Dönüş davranışı

Değer döndüren bir Function, her ulaşılabilir (reachable) yolda uyumlu bir
değer döndürmelidir. `null`'u yalnızca bildirilen dönüş türü `User?` gibi
null olabilen bir türse döndürebilir. Bir `Nothing` Function, yalın bir
`return` kullanabilir veya sonuna ulaşabilir.

## Aşırı yüklemeler (Overloads)

```ahd
double: Function := (
    value: Int
) -> Int {
    return value * 2
}

double: Overload Function := (
    value: Real
) -> Real {
    return value * 2.0
}
```

Tam argüman eşleşmeleri, güvenli genişletmeyi (widening) yener. Eşit en iyi
eşleşmeler derleme zamanı belirsizlikleridir (ambiguities) ve yalnızca
dönüş türü, bir aşırı yüklemeyi seçmek için yeterli değildir.

## Function değerleri ve callback'ler

```ahd
apply: Function := (
    operation: Function
    value: Int
) -> Int {
    return operation(value)
}

result: Int := apply(double, 5)
```

Kullanıcılar yalnızca `Function` yazar, ancak bu dinamik değildir. Her
Function bağlaması veya parametresi, derleme zamanında tek bir somut
(concrete) çağrılabilir imzaya çözülmelidir.
