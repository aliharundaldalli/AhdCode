# Statistics standart modülü

[English](STATISTICS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Data](DATA_TR.md) · [Modüller](MODULES_TR.md)

Statistics, tipli sayısal List'ler üzerinde betimleyici istatistiktir. Math,
Time, Regex, CSV ve Data gibi açıktır (explicit):

```ahd
bring Statistics
from Statistics bring StatisticsError
```

Kanonik kimlik `builtin:Statistics`'tir; kardeş bir `Statistics.ahd` onu
gölgeleyemez. Her argüman `NonNull`'dır.

Statistics, Data'ya bağımlı **değildir**. Bir `Table` hücresi `String`'tir, bu
yüzden bir program bir istatistik istemeden önce açıkça dönüştürür -- her iki
modülü de dinamik bir sayısal değer tanıtmak yerine katı tutan şey budur.

## Tipli girdi, tipli sonuç

Her fonksiyon açık bir `Int`/`Real` aşırı yükleme çifti olarak yayınlanır ve
sıradan aşırı yükleme çözümlemesiyle seçilir. Zayıf tipli bir giriş noktası
yoktur; bu yüzden bir sonucun statik türü her zaman bilinir.

```text
sum(values: List<Int>)   -> Int        sum(values: List<Real>)   -> Real
min(values: List<Int>)   -> Int        min(values: List<Real>)   -> Real
max(values: List<Int>)   -> Int        max(values: List<Real>)   -> Real
range(values: List<Int>) -> Int        range(values: List<Real>) -> Real
mode(values: List<Int>)  -> Int        mode(values: List<Real>)  -> Real

mean(values)           -> Real
median(values)         -> Real
variance(values)       -> Real
sampleVariance(values) -> Real
stdDev(values)         -> Real
sampleStdDev(values)   -> Real
quantile(values, probability: Real) -> Real
```

Cevabı girdinin kendi değerlerinden biri olan bir istatistik -- `min`, `max`,
`mode` ve fark olan `range` -- eleman türünü korur. Ortalama alan veya dağılım
ölçen bir istatistik her zaman `Real`'dir, çünkü tam sayıların ortalaması
genellikle tam sayı değildir.

`Int` sonuçlar dilin sıradan denetimli aritmetiğini kullanır; bu yüzden
işaretli 64-bit aralığından çıkan bir `Int` toplamı veya aralığı sarmalamak
yerine `OverflowError` fırlatır.

## String zorlaması yok

Sayısal bir istatistik asla metin okumaz. Bu derlenmez:

```ahd
Statistics.mean(["10", "20", "30"])
```

Data hücrelerinden gelen değerler açıkça dönüştürülür:

```ahd
scores: List<Real> := students.column("score").map(
    lambda (value: String) -> real(value)
)

average: Real := Statistics.mean(scores)
```

## Matematiksel tanımlar

`mean`, aritmetik ortalamadır.

`median`, sıralanmış verinin ortadaki değeridir; sayı çift olduğunda ortadaki
iki değerin ortalamasını alır. Her zaman `Real`'dir, bu yüzden çift durumu ayrı
bir kural gerektirmez: `median([1, 2, 3, 4])` `2.5`, `median([1, 2, 3])` ise
`2.0`'dır.

`variance` ve `stdDev` **popülasyon** biçimleridir ve `n`'e böler.
`sampleVariance` ve `sampleStdDev` **örneklem** biçimleridir ve `n - 1`'e böler
(Bessel düzeltmesi). Tanımın asla örtük kalmaması için iki isim de yayınlanır:

```ahd
values: List<Int> := [3, 1, 4, 1, 5]

write(Statistics.variance(values))        // 2.56   popülasyon, / n
write(Statistics.sampleVariance(values))  // 3.2    örneklem, / (n - 1)
```

`stdDev`, `variance`'ın kareköküdür; `sampleStdDev` ise `sampleVariance`'ın
kareköküdür.

`mode`, en sık görülen değerdir. Birden fazla değer en yüksek frekansta
eşitlendiğinde, girdide **ilk geçen** kazanır; bu yüzden sonuç asla map
yineleme sırasına bağlı değildir:

```ahd
write(Statistics.mode([2, 3, 3, 2]))  // 2
write(Statistics.mode([3, 2, 2, 3]))  // 3
```

`quantile(values, probability)`, komşu sıra istatistikleri arasında doğrusal
interpolasyon kullanır. Veri artan sırada ve `n` değer varken konum
`probability * (n - 1)`'dir; bu konum iki değerin arasına düştüğünde sonuç,
kesirli kısma göre aralarında interpolasyon yapar.

- `probability`, `0.0..1.0` aralığında olmalıdır; başka bir şey kırpılmak
  yerine `StatisticsError` fırlatır.
- `probability` `0.0` minimum, `1.0` maksimumdur.
- Tek değerli bir List, her geçerli olasılık için kendi quantile'ıdır.

```ahd
values: List<Int> := [1, 2, 3, 4]

write(Statistics.quantile(values, 0.0))   // 1.0
write(Statistics.quantile(values, 0.25))  // 1.75
write(Statistics.quantile(values, 0.5))   // 2.5
write(Statistics.quantile(values, 1.0))   // 4.0
```

## Boş ve tanımsız girdi

Boş bir List'in `sum`'ı toplamsal birim öğedir -- `Int` için `0`, `Real` için
`0.0` -- çünkü `sum(a) + sum(b)`'yi birleşik değerlerin toplamına eşit tutan
tek toplam budur.

Diğer her istatistik boş bir List için matematiksel olarak tanımsızdır ve
`StatisticsError` fırlatır:

```text
mean([])      median([])      min([])       max([])
range([])     variance([])    stdDev([])    mode([])
quantile([], p)
```

`sampleVariance` ve `sampleStdDev` ek olarak en az iki değer gerektirir, çünkü
`n - 1`'e bölmek tek bir değer için tanımsızdır.

```ahd
attempt {
    write(Statistics.mean(empty))
} except StatisticsError as error {
    write(error.message)
}
```

`StatisticsError` doğrudan `Error`'dan türer. Yalnızca girdisi için tanımsız
olan istatistikler için kullanılır; Data, CSV veya dosya sistemi
başarısızlıkları için yeniden kullanılmaz.

## Sonlu sonuçlar

AhdCode'un `Real`'i dilin mevcut sözleşmesi gereği sonludur: sıradan aritmetik,
`NaN` veya sonsuzluk üretmek yerine bir alan (domain) veya aralık hatası
bildirir. Statistics bu sözleşmeyi korur -- bir istatistik asla `NaN` veya
sonsuzluk döndürmez; böyle bir sonuç oluşacaksa `StatisticsError` bildirir.

## Girdi asla değiştirilmez

Bir median veya quantile için veriyi sıralamak bir anlık görüntü üzerinde
çalışır, bu yüzden çağıranın List'i sırasını korur:

```ahd
values: List<Int> := [3, 1, 2]

write(Statistics.median(values))  // 2.0
write(values)                     // [3, 1, 2]
```

## Statistics ne değildir

`Statistics` modülü yalnızca betimleyici istatistiktir. Çıkarımsal test, regresyon,
dağılım, rastgele örnekleme ve grafik çizimi yoktur. Bir `frequency` fonksiyonu
da yoktur: bir frekans tablosu `Pair<K, Int>` olurdu ve bir Pair anahtarı
`String`, `Int` veya `Bool` olmak zorundadır; bu yüzden `List<Real>` girdisinin
ifade edilebilir bir sonucu yoktur. `mode` yaygın ihtiyacı karşılar ve
[`Table.valueCounts`](DATA_TR.md) String hücreleri sayar.
