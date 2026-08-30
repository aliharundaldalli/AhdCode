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

## Açık lambda bağımlılıkları

Bir lambda, kendi parametrelerinin dışındaki bir bağlamayı yalnızca o bağlamayı
`lambda` ile parametreler arasına yazılan açık bir bağımlılık listesinde
listeleyerek okur. Her girdi kendi türünü belirtir:

- `#name` veya `Local name` -- çevreleyen bir bağlamanın değere göre yakalanması.
- `@name` veya `Global name` -- bir modül/global bağlamasına açık bir
  bağımlılık; sıradan bir Function'ın modül durumuna dokunmak için zaten
  ihtiyaç duyduğu `Global` bildirimini yansıtır.

```ahd
keepAbove: Function := (
    minimum: Int
    scores: List<Int>
) -> List<Int> {
    return scores.filter(lambda [#minimum] (score: Int) -> score >= minimum)
}

Maximum: Int := 100
inRange: Function := lambda [#minimum, @Maximum] (score: Int) -> score >= minimum and score <= Maximum
```

Birden çok girdi virgülle ayrılır: `lambda [#low, #high] (v: Int) -> ...`.
`#name`/`Local name` ve `@name`/`Global name` aynı bağımlılığın alternatif
yazımlarıdır; bir liste kısa ve tam yazımları serbestçe karıştırabilir.
Formatter, kaynağın hangi yazımı kullandığını korur.

Bir bağımlılık asla çıkarılmaz (infer edilmez) ve yalın bir isim
(`lambda [minimum] (...)`, yayınlanmamış v0.1.13-öncesi yazım) reddedilir: her
girdi Local mi yoksa Global mi olduğunu belirtmelidir. Listenin atladığı
çevreleyen bir `Local`'i veya Function parametresini okumak, bağlamayı
isimlendiren bir derleme zamanı hatasıdır; böylece bir lambda'nın neye bağlı
olduğu -- ve nasıl -- lambda'nın yazıldığı yerde görünür olur:

```text
SEM043  local "minimum" is not a lambda dependency
SEM007  module binding "Maximum" requires an explicit Global dependency
```

Bağımlılık listesi olmadan yazılan bir lambda, kendi parametrelerinin
dışında hiçbir şey okumaz; bu yüzden v0.1.13'ten önce derlenen her lambda
değişmeden derlenmeye devam eder. `lambda [] (...)` kabul edilir ve aynı
anlama gelir.

`#`/`Local` yalnızca çevreleyen bir çağrılabilirin bağlamasını isimlendirir:
bir Function parametresi, bir `Local` veya bir `for`/`except` bağlaması.
`@`/`Global` yalnızca modül kökündeki bir değer bağlamasını isimlendirir.
Modül kökündeki bir Class, Function veya isim uzayı sıradan aramayla erişilir
ve her iki türde de listelenmemelidir.

**Bir `#`/`Local` yakalaması değere göredir.** Lambda değeri
oluşturulduğunda bağlamanın tuttuğu şeyi okur; bu yüzden o bağlamadaki
sonraki bir değişiklik lambda'nın içinde görünmez:

```ahd
step: Local Int := 1
first: Local Function := lambda [#step] (x: Int) -> x + step
step = step + 100
second: Local Function := lambda [#step] (x: Int) -> x + step
// first(0) 1'dir, second(0) ise 101
```

Referans değerler dilin sıradan kurallarını izler: bir `List`, `Pair` veya
Class örneğini yakalamak, tam olarak onu bir parametre olarak geçirmek gibi
referansı kopyalar; bu yüzden referans verilen nesne paylaşılmaya devam eder.

Yakalanan bir isim lambda içinde salt okunurdur. `#`/`Local`, lambda'ya
çevreleyen değeri verir, çevreleyen değişkenin sahipliğini değil; v0.1.13
değiştirilebilir bir closure hücresi veya referans kutusu eklemez.

**Bir `@`/`Global` bağımlılığı bir yakalama değildir.** Modül bağlamasını
anlık görüntülemez veya closure depolamasına kopyalamaz; gerçek bağlamayı
AhdCode'un sıradan global-mutasyon kuralları altında okur; bu yüzden lambda
oluşturulduktan sonraki yasal bir mutasyon, lambda bir sonraki çalıştığında
görünür olur:

```ahd
Maximum: Int := 100
check: Function := lambda [@Maximum] (score: Int) -> score <= Maximum

check(50)  // true, Maximum 100'dür
Maximum = 40
check(50)  // false, @Maximum canlı bağlamayı gözlemler
```

Bağımlılıklar mevcut her callback ile çalışır; buna [Data](DATA_TR.md)'nın
`filter`, `sort`, `transform` ve `derive` üyeleri dahildir:

```ahd
strong: Local Table := table.filter(
    lambda [#minimum] (row: Pair<String, String>) -> int(row["score"]) >= minimum
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
