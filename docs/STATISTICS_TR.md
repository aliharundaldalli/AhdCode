# Statistics standart modülü

[English](STATISTICS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Data](DATA_TR.md)

**Matematik yolu:** [Math](MATH_TR.md) → [Numeric](NUMERIC_TR.md) → **Statistics** → [Plot](PLOT_TR.md)

`Statistics` bir veri kümesini betimler: merkezinin nerede olduğunu (ortalama,
medyan, mod), ne kadar yayıldığını (açıklık, varyans, standart sapma,
yüzdelikler) ve iki veri kümesinin birlikte nasıl değiştiğini (kovaryans,
korelasyon, uydurulan bir doğru). Girdisi, türü belli bir sayı List'idir.

```ahd
bring Statistics

scores: List<Int> := [72, 85, 90, 64, 85, 78, 95, 58]
write(Statistics.mean(scores))
write(Statistics.median(scores))
write(Statistics.mode(scores))
```

```text
78.375
81.5
85
```

Statistics açık (explicit) bir modüldür. Hata sınıfını adıyla yakalamak
istediğinizde onu da getirin:

```ahd
bring Statistics
from Statistics bring StatisticsError
```

Kanonik kimlik `builtin:Statistics`'tir; yanındaki bir `Statistics.ahd` onu
gölgeleyemez. Her argüman `NonNull`'dır ve hiçbir fonksiyon aldığı List'i
değiştirmez.

## Bu sayfada

- [Bir bakışta](#bir-bakışta)
- [Int ve Real girdi, türü belli sonuçlar](#int-ve-real-girdi-türü-belli-sonuçlar)
- [Merkez: ortalama, medyan, mod](#merkez-ortalama-medyan-mod)
- [Yayılım: açıklık, varyans, standart sapma](#yayılım-açıklık-varyans-standart-sapma)
- [Yüzdelikler (quantile)](#yüzdelikler-quantile)
- [Veri çiftleri: kovaryans, korelasyon ve uydurulan doğru](#veri-çiftleri-kovaryans-korelasyon-ve-uydurulan-doğru)
- [Boş ve tanımsız girdi](#boş-ve-tanımsız-girdi)
- [Metinden, CSV'den ve tablolardan veri](#metinden-csvden-ve-tablolardan-veri)
- [Çözümlü örnek: bir sınav raporu](#çözümlü-örnek-bir-sınav-raporu)
- [Statistics ne değildir](#statistics-ne-değildir)

## Bir bakışta

| Fonksiyon | Sonuç | Anlamı |
|---|---|---|
| `sum(values)` | öğe türü | toplam; boş List için `0` / `0.0` |
| `min(values)`, `max(values)` | öğe türü | en küçük / en büyük değer |
| `range(values)` | öğe türü | `max - min` (açıklık) |
| `mode(values)` | öğe türü | en sık değer; eşitlikte ilk görülen kazanır |
| `mean(values)` | `Real` | aritmetik ortalama |
| `median(values)` | `Real` | sıralı verinin ortası |
| `variance(values)` | `Real` | popülasyon varyansı, `n`'e böler |
| `sampleVariance(values)` | `Real` | örneklem varyansı, `n - 1`'e böler |
| `stdDev(values)` | `Real` | √`variance` |
| `sampleStdDev(values)` | `Real` | √`sampleVariance` |
| `quantile(values, probability)` | `Real` | verinin o oranının altında kaldığı değer |
| `covariance(x, y)` | `Real` | popülasyon kovaryansı |
| `sampleCovariance(x, y)` | `Real` | örneklem kovaryansı |
| `correlation(x, y)` | `Real` | Pearson r, `-1.0..1.0` aralığında |
| `linearRegression(x, y)` | `Pair<String, Real>` | en küçük kareler doğrusu: `slope`, `intercept` |

`values`, `x` ve `y`'nin her biri bir `List<Int>` ya da bir `List<Real>`'dir.

## Int ve Real girdi, türü belli sonuçlar

Her fonksiyon açık bir `Int`/`Real` aşırı yükleme (overload) çifti olarak
yayınlanır; böylece bir sonucun statik türü her zaman bilinir:

- Yanıtı girdinin kendi değerlerinden biri olan istatistikler — `min`, `max`,
  `mode` ve fark olan `range` — **öğe türünü korur**: Int List Int verir.
- Ortalama alan ya da yayılım ölçen istatistikler — `mean`, `median`,
  varyanslar, standart sapmalar, `quantile` ve iki List'li fonksiyonlar — her
  zaman **`Real`**'dir, çünkü tam sayıların ortalaması genellikle tam sayı
  değildir.
- İşaretli 64 bit aralığının dışına çıkan bir `Int` `sum` ya da `range`, başa
  sarmak yerine `OverflowError` verir.

```ahd
bring Statistics

write(type(Statistics.max([3, 8, 5])))
write(type(Statistics.mean([3, 8, 5])))
write(Statistics.median([1, 2, 3]))
```

```text
Int
Real
2.0
```

Sayısal bir istatistik hiçbir zaman metin okumaz. Stringler rakam içerse bile
`List<String>` vermek derleme hatasıdır:

```ahd
bring Statistics

write(Statistics.mean(["10", "20", "30"]))
```

Açık dönüşüm için [Metinden, CSV'den ve tablolardan veri](#metinden-csvden-ve-tablolardan-veri)
bölümüne bakın.

## Merkez: ortalama, medyan, mod

- `mean` aritmetik ortalamadır: toplamın adede bölümü.
- `median` sıralanmış verinin ortadaki değeridir; adet çiftse ortadaki iki
  değerin ortalamasını alır: `median([1, 2, 3, 4])` değeri `2.5`'tir. Sıralama
  bir kopya üzerinde yapılır; List'iniz sırasını korur.
- `mode` en sık görülen değerdir. Birden çok değer en yüksek sıklıkta
  eşitlenirse **girdide ilk görülen** kazanır; böylece sonuç hiçbir gizli
  sıralamaya bağlı olmaz.

```ahd
bring Statistics

values: List<Int> := [3, 1, 2]
write(Statistics.median(values))
write(values)
write(Statistics.mode([2, 3, 3, 2]))
write(Statistics.mode([3, 2, 2, 3]))
```

```text
2.0
[3, 1, 2]
2
3
```

Ortalama uç değerlerden etkilenir, medyan etkilenmez; ikisini karşılaştırmak
aykırı değerler için hızlı bir denetimdir.

## Yayılım: açıklık, varyans, standart sapma

`range`, `max - min`'dir. `variance` ve `stdDev` **popülasyon**
biçimleridir ve `n`'e böler; `sampleVariance` ve `sampleStdDev` **örneklem**
biçimleridir ve `n - 1`'e böler (Bessel düzeltmesi). List ilgilendiğiniz
grubun tamamıysa popülasyon biçimini, daha büyük bir popülasyonu tahmin etmek
için kullanılan bir örneklemse örneklem biçimini kullanın. Tanımın hiçbir
zaman örtük kalmaması için iki ad da yayınlanır.

```ahd
bring Statistics

values: List<Int> := [3, 1, 4, 1, 5]
write(Statistics.range(values))
write(Statistics.variance(values))
write(Statistics.sampleVariance(values))
write(Statistics.stdDev([2, 4, 4, 4, 5, 5, 7, 9]))
```

```text
4
2.56
3.2
2.0
```

## Yüzdelikler (quantile)

`quantile(values, probability)`, verinin verilen oranının altında kaldığı
değeri döndürür: `0.5` medyandır, `0.25` ve `0.75` birinci ve üçüncü
çeyreklerdir. Doğrusal ara değerleme kullanır: veri artan sırada ve `n` değer
varken konum `probability * (n - 1)`'dir; iki değer arasına düşen bir konum,
kesirli kısmı oranında ikisi arasında ara değer alır.

- `probability` `0.0..1.0` aralığında olmalıdır; başka bir değer
  kırpılmak yerine `StatisticsError` verir.
- `0.0` en küçük, `1.0` en büyük değeri verir.
- Tek değerli bir List, her geçerli olasılık için kendi yüzdeliğidir.

```ahd
bring Statistics

values: List<Int> := [1, 2, 3, 4]
write(Statistics.quantile(values, 0.0))
write(Statistics.quantile(values, 0.25))
write(Statistics.quantile(values, 0.5))
write(Statistics.quantile(values, 1.0))
```

```text
1.0
1.75
2.5
4.0
```

Aykırı değerlerden etkilenmeyen bir yayılım ölçüsü olan çeyrekler açıklığı
`quantile(values, 0.75) - quantile(values, 0.25)`'tir.

## Veri çiftleri: kovaryans, korelasyon ve uydurulan doğru

```text
covariance(first, second)       -> Real
sampleCovariance(first, second) -> Real
correlation(first, second)      -> Real
linearRegression(x, y)          -> Pair<String, Real>
```

Her argüman, herhangi bir birleşimle, bir `List<Int>` ya da bir
`List<Real>`'dir. İki List'in uzunluğu aynı olmalıdır; aynı konumdaki
değerler bir `(xᵢ, yᵢ)` çifti oluşturur.

- `covariance` **popülasyon** kovaryansıdır, `n`'e böler; en az bir çift
  ister. `sampleCovariance` `n - 1`'e böler ve en az iki çift ister.
- `correlation` Pearson korelasyon katsayısı `r`'dir: `1.0` kusursuz artan
  bir doğru, `-1.0` kusursuz azalan bir doğru, `0.0` doğrusal ilişki yok
  demektir. En az iki çift ister ve List'lerden birinin varyansı sıfırsa (bütün
  değerler eşit) `StatisticsError` verir, çünkü `r` o zaman tanımsızdır. Son
  bitteki bir yuvarlama taşması sınıra kırpılır.
- `linearRegression(x, y)`, `y = slope · x + intercept` doğrusunu sıradan en
  küçük kareler yöntemiyle uydurur ve bu anahtar sırasıyla
  `{"slope": …, "intercept": …}` Pair'ini döndürür. En az iki çift ve hepsi
  eşit olmayan x değerleri ister. Bütün y değerleri aynıysa eğim `0.0`,
  kesişim tam olarak o değerdir.

Her fonksiyon çarpmadan önce veriyi ortalamasına göre merkezler (iki geçişli
hesap); böylece yıllar ya da zaman damgaları gibi büyük kaymalar sonucu
bozmaz.

```ahd
bring Math
bring Statistics

hours: List<Int> := [1, 2, 3, 4, 5]
scores: List<Real> := [52.0, 57.5, 61.0, 68.5, 71.0]
write(Statistics.covariance(hours, scores))
write(Math.round(Statistics.correlation(hours, scores), 3))
fit := Statistics.linearRegression(x: hours, y: scores)
write(fit)
predicted := fit["slope"] * 6 + fit["intercept"]
write("after 6 hours: {Math.round(predicted, 1)}")
```

```text
9.8
0.991
{"slope": 4.9, "intercept": 47.3}
after 6 hours: 76.7
```

Veriyi ve uydurulan doğruyu çizmek için
[Plot](PLOT_TR.md#birden-çok-seri) sayfasına bakın.

## Boş ve tanımsız girdi

Boş bir List'in `sum`'ı toplama birim öğesidir — `Int` için `0`, `Real` için
`0.0` — çünkü bu, `sum(a) + sum(b)`'nin birleşik değerlerin toplamına eşit
kalmasını sağlar.

Diğer her istatistik boş bir List için matematiksel olarak tanımsızdır ve
`StatisticsError` verir: `mean`, `median`, `min`, `max`, `range`, `mode`,
`variance`, `stdDev` ve `quantile`. `sampleVariance` ve `sampleStdDev` ayrıca
en az iki değer ister, çünkü tek değer için `n - 1`'e bölmek tanımsızdır.
Uzunlukları farklı List'ler ve yukarıdaki iki List'li durumlar da
`StatisticsError` verir.

```ahd
bring Statistics
from Statistics bring StatisticsError

empty: List<Real> := []
write(Statistics.sum(empty))
attempt {
    write(Statistics.mean(empty))
} except StatisticsError as error {
    write(error.message)
}
attempt {
    write(Statistics.sampleVariance([5]))
} except StatisticsError as error {
    write(error.message)
}
```

```text
0.0
mean is undefined for an empty List
sampleVariance requires at least two values
```

`StatisticsError` doğrudan `Error`'dan türer. Yalnızca girdisi için tanımsız
olan istatistiklerde kullanılır; Data, CSV ya da dosya sistemi hataları için
yeniden kullanılmaz. Hata mesajları İngilizcedir.

AhdCode'un `Real`'i her zaman sonludur: sıradan aritmetik `NaN` ya da sonsuzluk
üretmek yerine bir tanım kümesi ya da aralık hatası bildirir ve Statistics bu
sözleşmeyi korur — bir istatistik hiçbir zaman `NaN` ya da sonsuzluk
döndürmez, böyle bir durum ortaya çıkacaksa `StatisticsError` bildirir.

## Metinden, CSV'den ve tablolardan veri

Statistics, [Data](DATA_TR.md) ya da [CSV](CSV_TR.md)'ye **bağlı değildir**.
Bir `Table` hücresi ve bir CSV alanı `String`'dir; bu yüzden bir istatistik
istemeden önce onları `int` ya da `real` ile açıkça dönüştürün. İki modülü
dinamik bir sayısal değer tanıtmak yerine katı tutan budur.

```ahd
bring Statistics

raw: List<String> := ["72", "85.5", "90"]
values: List<Real> := raw.map(lambda (value: String) -> real(value))
write(Statistics.mean(values))
```

```text
82.5
```

Bir Data `Table`'ında aynı dönüşüm bir sütuna uygulanır:

```ahd
scores: List<Real> := students.column("score").map(
    lambda (value: String) -> real(value)
)
average: Real := Statistics.mean(scores)
```

`real`, sayı olmayan metin için `DomainError` verir; böylece hatalı girdi bir
istatistiğin içinde değil, okunduğu yerde bildirilir. Bir Numeric `Vector`
`vector.values()` olarak verilir.

## Çözümlü örnek: bir sınav raporu

```ahd
bring Math
bring Statistics

scores: List<Int> := [72, 85, 90, 64, 85, 78, 95, 58]

write("students: {len(scores)}")
write("mean:     {Statistics.mean(scores)}")
write("median:   {Statistics.median(scores)}")
write("range:    {Statistics.min(scores)}..{Statistics.max(scores)}")
write("std dev:  {Math.round(Statistics.sampleStdDev(scores), 2)}")
write("Q1..Q3:   {Statistics.quantile(scores, 0.25)}..{Statistics.quantile(scores, 0.75)}")
passed := scores.filter(lambda (score: Int) -> score >= 70)
write("passed:   {len(passed)}")
```

```text
students: 8
mean:     78.375
median:   81.5
range:    58..95
std dev:  12.88
Q1..Q3:   70.0..86.25
passed:   6
```

Burada örneklem standart sapması kullanılır, çünkü bir sınıf genellikle
sınava girebilecek bütün öğrencilerden alınmış bir örneklem olarak okunur.
Dağılımı göstermek için [`Plot.histogram` ve `Plot.box`](PLOT_TR.md#grafik-türleri)
ile devam edin.

## Statistics ne değildir

`Statistics` modülü yalnızca betimleyici istatistik ve basit bir en küçük
kareler doğrusudur. Çıkarımsal test (p-değeri ya da hipotez testi); çoklu,
polinom ya da lojistik regresyon; `rSquared` ya da bir regresyon nesnesi;
olasılık dağılımları; rastgele örnekleme; makine öğrenmesi ve çizim yoktur.
`frequency` fonksiyonu da yoktur: bir sıklık tablosu `Pair<K, Int>` olurdu ve
bir Pair anahtarı `String`, `Int` ya da `Bool` olmak zorundadır; bu yüzden
`List<Real>` girdinin ifade edilebilir bir sonucu yoktur. Yaygın ihtiyacı
`mode` karşılar; String hücreleri [`Table.valueCounts`](DATA_TR.md) sayar.
Rastgele sayılar için [Math](MATH_TR.md#rastgele-sayılar) sayfasına bakın.

**Sonraki:** [Plot](PLOT_TR.md) — fonksiyonları ve verileri çizmek.
