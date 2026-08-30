# Fonksiyonlar

[English](FUNCTIONS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Sınıflar](CLASSES_TR.md)

Function'lar modül kökündeki (module root) isimlendirilmiş bildirimlerdir;
metotlar bir Class içindeki isimlendirilmiş bildirimlerdir. İç içe Function
bildirimleri desteklenmez. Lambda, mevcut `Function` türünde isimsiz bir değer
oluşturan, yalnızca ifade içeren kısa yazımdır; yeni bir tür veya ikinci bir
çağrılabilir sistem değildir.

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

## İfade lambda'ları

```ahd
square := lambda (x: Int) -> x^2
positive: Function := lambda (x: Int) -> x > 0

values := [1, 2, 3]
squares := values.map(lambda (x: Int) -> x^2)
```

Kesin sözdizimi `lambda (<tipli parametreler>) -> <ifade>` biçimindedir. Her
parametrenin açık bir statik türü vardır; dönüş türü ve dönüş null olabilirliği
tek gövde ifadesinden çıkarılır. Sıfır parametre ve Function için zaten geçerli
olan her parametre türü desteklenir. Uyumlu bir lambda; `map`, `filter` ve
mevcut `sort` anahtar callback'i dahil, `Function` değeri kabul edilen her
yerde çalışır. Örtük zorlama (implicit coercion) eklenmez.

Lambda parametreleri zorunludur; varsayılan değerli parametreler v0.1.11'de
ifade lambda'larında değil, isimli Function bildirimlerinde kullanılabilir.

Bir lambda blok veya deyim (statement) içeremez. Kontrol akışı, bildirimler,
döngüler, hata işleme veya birden çok adım gerektiğinde değişmeyen isimli
Function sözdizimini kullanın. Sıradan `Local`/`Global` görünürlük kuralları
değişmez.

## Açık lambda yakalama (capture)

Bir lambda, çevreleyen çağrılabilirdeki bir bağlamayı yalnızca o bağlamayı
`lambda` ile parametreler arasına yazılan açık bir yakalama listesinde
listeleyerek okur:

```ahd
keepAbove: Function := (
    minimum: Int
    scores: List<Int>
) -> List<Int> {
    return scores.filter(lambda [minimum] (score: Int) -> score >= minimum)
}
```

Birden çok isim virgülle ayrılır: `lambda [low, high] (v: Int) -> ...`.

Yakalama asla çıkarılmaz (infer edilmez). Listenin atladığı çevreleyen bir
`Local`'i veya Function parametresini okumak, bağlamayı isimlendiren bir
derleme zamanı hatasıdır; böylece bir lambda'nın neye bağlı olduğu, lambda'nın
yazıldığı yerde görünür olur:

```text
SEM043  local "minimum" is not captured by this lambda
```

Yakalama listesi olmadan yazılan bir lambda hiçbir şey yakalamaz; bu yüzden
v0.1.13'ten önce derlenen her lambda değişmeden derlenmeye devam eder.
`lambda [] (...)` kabul edilir ve aynı anlama gelir.

Yalnızca çevreleyen bir çağrılabilirin bağlaması yakalanır: bir Function
parametresi, bir `Local` veya bir `for`/`except` bağlaması. Modül kökündeki bir
isim, Class veya isim uzayı sıradan aramayla erişilir ve listelenmemelidir --
modül bağlamaları mevcut `Global` kuralını izlemeye devam eder.

**Yakalama değere göredir.** Bir yakalama, lambda değeri oluşturulduğunda
bağlamanın tuttuğu şeyi okur; bu yüzden o bağlamadaki sonraki bir değişiklik
lambda'nın içinde görünmez:

```ahd
step: Local Int := 1
first: Local Function := lambda [step] (x: Int) -> x + step
step = step + 100
second: Local Function := lambda [step] (x: Int) -> x + step
// first(0) 1'dir, second(0) ise 101
```

Referans değerler dilin sıradan kurallarını izler: bir `List`, `Pair` veya
Class örneğini yakalamak, tam olarak onu bir parametre olarak geçirmek gibi
referansı kopyalar; bu yüzden referans verilen nesne paylaşılmaya devam eder.

Yakalanan bir isim lambda içinde salt okunurdur. Açık yakalama, lambda'ya
çevreleyen değeri verir, çevreleyen değişkenin sahipliğini değil; v0.1.13
değiştirilebilir bir closure hücresi veya referans kutusu eklemez.

Yakalamalar mevcut her callback ile çalışır; buna [Data](DATA_TR.md)'nın
`filter`, `sort`, `transform` ve `derive` üyeleri dahildir:

```ahd
strong: Local Table := table.filter(
    lambda [minimum] (row: Pair<String, String>) -> int(row["score"]) >= minimum
)
```

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
