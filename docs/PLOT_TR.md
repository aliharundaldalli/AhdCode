# Plot standart modülü

[English](PLOT.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Statistics](STATISTICS_TR.md) · [Modüller](MODULES_TR.md)

Plot, tipli sayısal List'lerden grafik çizer. Math, Time, Regex, CSV, Data
ve Statistics gibi açıktır (explicit):

```ahd
bring Plot
from Plot bring Chart
from Plot bring Figure
from Plot bring PlotError
```

Kanonik kimlik `builtin:Plot`'tur; kardeş bir `Plot.ahd` dosyası onun yerini
alamaz (shadow edemez). Her argüman `NonNull`'dır.

Plot, Data'ya bağımlı **değildir**. Bir `Table` hücresi bir `String`'dir, bu
yüzden bir program bir sütunu çizmeden önce açıkça dönüştürür — Statistics'in
kullandığı ve aynı nedenle kullandığı disiplinin aynısı.

## Grafik türleri

```text
Plot.line(x, y)                              -> Chart
Plot.scatter(x, y)                           -> Chart
Plot.bar(labels: List<String>, values)       -> Chart
Plot.histogram(values, bins: Int)            -> Chart
Plot.box(values)                             -> Chart
Plot.errorBar(x, y, lowerErrors, upperErrors) -> Chart
Plot.new()                                   -> Chart
Plot.subplots(rows: Int, columns: Int, charts: List<Chart>) -> Figure
```

Tek bir grafik — line, scatter, bar, histogram, box veya error bar — bir
`Chart` üretir. Çoklu-grafik kompozisyonu bir `Figure` üretir (bkz.
[Subplot'lar](#subplotlar)).

## Katı sayısal girdi, String zorlaması (coercion) yok

Her sayısal argüman `List<Int>` veya `List<Real>` kabul eder; bu, sıradan
overload çözümlemesiyle çözülür ve bir `Int` List dahili olarak güvenle
`Real`'a genişletilir. `x` ve `y` bağımsız olarak `List<Int>` veya
`List<Real>` olabilir:

```ahd
x: List<Int> := [1, 2, 3, 4]
y: List<Real> := [2.0, 5.0, 4.0, 8.0]

chart := Plot.line(x, y)
```

Bir `List<String>` -- rakam metni tutsa bile -- asla kabul edilmez. Bu
derlenmez:

```ahd
Plot.line(["1", "2", "3"], ["2", "5", "4"])
```

Data entegrasyonu, tıpkı Statistics'te olduğu gibi açık kalır:

```ahd
scores: List<Int> := table.column("score").map(
    lambda (value: String) -> int(value)
)

chart := Plot.histogram(scores, 10)
```

## Boş veri

Her grafik oluşturucusu (`Plot.line`, `Plot.scatter`, `Plot.bar`,
`Plot.histogram`, `Plot.box`, `Plot.errorBar`) ve `Chart.line`/`Chart.scatter`,
boş sayısal girdi için `PlotError` fırlatır. Çizilecek anlamlı bir şey
olmadığından, bu -- `Statistics.mean([])`'in olduğu gibi -- bir alan
(domain) hatasıdır:

```ahd
attempt {
    Plot.line(empty, empty)
} except PlotError as error {
    write(error.message)  // "line chart data must not be empty"
}
```

## Grafik meta verisi

```text
chart.title(text: String)   -> Chart
chart.xLabel(text: String)  -> Chart
chart.yLabel(text: String)  -> Chart
chart.legend(enabled: Bool) -> Chart
chart.size(width: Int, height: Int) -> Chart
```

Her Chart metodu saftır (pure): **yeni** bir Chart döndürür ve alıcısını
(receiver) asla değiştirmez -- [`Table`](DATA_TR.md)'ın her işlem için
kullandığı aynı kural. Bu yüzden yapılandırma, yeniden atama yoluyla
zincirlenir:

```ahd
chart := Plot.line(x, y)
chart = chart.title("Experiment")
chart = chart.xLabel("Time")
chart = chart.yLabel("Value")
```

`size`, PNG için çıktı boyutlarını piksel cinsinden, SVG/PDF için ise
eşdeğer sayfa boyutunu ayarlar; hem `width` hem `height` pozitif olmalıdır.
Bir Chart'ın varsayılan boyutu 800x600'dür.

## Birden çok seri

`chart.line(x, y, label)` ve `chart.scatter(x, y, label)`, bir Chart'a bir
seri daha ekler; böylece bir line ve bir scatter serisi -- veya birkaç line,
ya da birkaç scatter serisi -- bir legend ile tek bir Chart'ı paylaşabilir:

```ahd
chart := Plot.new()
chart = chart.line(x, y1, "Experiment")
chart = chart.scatter(x, y2, "Observation")
chart = chart.legend(true)
```

`Plot.line(x, y)` ve `Plot.scatter(x, y)`, etiketsiz tek bir seriyle bir
Chart başlatmanın kısayoludur; `chart.line`/`chart.scatter` onu genişletir
(veya bu şekilde zaten oluşturulmuş bir Chart'ı genişletir). `x` ve `y`,
diğer her sayısal argümanla aynı bağımsız `List<Int>`/`List<Real>` kuralını
izler.

Bir `bar`, `histogram`, `box` veya `errorBar` Chart'ına bir line veya
scatter serisi eklemek `PlotError` fırlatır: bu grafik türleri kendi
kendine yeterlidir ve seri modeliyle bileşmez.

## Save (kaydetme)

```text
chart.save(path: String) -> Nothing
figure.save(path: String) -> Nothing
```

Çıktı biçimi dosya uzantısından çıkarılır. Desteklenen biçimler PNG
(`.png`), SVG (`.svg`) ve PDF'dir (`.pdf`); başka herhangi bir şey
`PlotError` fırlatır:

```ahd
chart.save("result.png")
chart.save("result.svg")
chart.save("result.pdf")

attempt {
    chart.save("result.bmp")
} except PlotError as error {
    write(error.message)
}
```

Göreli bir yol, programın çalışma dizinine göre çözülür --
[`File`](FILESYSTEM_TR.md)'ın kullandığı aynı kural. Bir render veya
dosya sistemi hatası, ham bir Go hatası değil, her zaman `PlotError`
fırlatır.

## Show (gösterme)

```text
chart.show() -> Nothing
figure.show() -> Nothing
```

`show()`, benzersiz bir geçici PNG'ye render eder ve onu platformun standart
görüntü açma mekanizmasıyla açar (macOS'ta `open`, Linux'ta `xdg-open`,
Windows'ta kabuğun `start` komutu); böylece bir grafiği incelemek asla elle
kaydedip dosyayı bulmayı gerektirmez. Geçici görüntü, sistem geçici
dizininin altındaki AhdCode'a özgü bir alanda yaşar; otomatik olarak
silinmez, çünkü harici görüntüleyicinin `show()` döndükten sonra da onu
okumaya devam etmesi gerekir.

`show()` bir masaüstü oturumu gerektirir. Başsız (headless) bir ortam (CI,
görüntüsü olmayan bir konteyner, kayıtlı bir işleyicisi olmayan
`xdg-open`) askıda kalmak yerine temiz bir şekilde `PlotError` ile
başarısız olur -- her render/açma adımı kısa bir zaman aşımı altında
çalışır.

## Subplot'lar

```ahd
figure := Plot.subplots(
    2, 2,
    [
        Plot.line(x1, y1),
        Plot.scatter(x2, y2),
        Plot.histogram(values, 10),
        Plot.box(values)
    ]
)

figure.show()
figure.save("summary.pdf")
```

`charts` satır-öncelikli (row-major) sıradadır. `rows` ve `columns`'ın her
ikisi de pozitif olmalıdır ve grafik sayısı `rows * columns`'ı aşamaz; tam
bir sayı gerektirmek yerine, hücrelerden daha az grafiğe izin verilir ve
kalan hücreler boş bırakılır. Bir `Figure`, `Plot.subplots` tarafından
üretilen açık, immutable bir değerdir -- mutable global bir "geçerli
subplot" durumu yoktur.

Bir `Figure`'ın save/show boyutu, grid boyutlarından belirlenimci
(deterministic) şekilde türetilir (rows ve columns ile ölçeklenen sabit bir
hücre-başı bütçe); v0.1.14 bir `Figure.size` metodu yayımlamaz.

## PlotError

```ahd
bring Plot
from Plot bring PlotError
```

`PlotError`, doğrudan `Error`'dan türer. Plot, her plot'a özgü çalışma
zamanı hatası için onu fırlatır: eşleşmeyen `x`/`y` uzunlukları, boş grafik
verisi, geçersiz bir bin sayısı, eşleşmeyen bar etiketleri/değerleri,
eşleşmeyen error-bar verisi, negatif hata büyüklükleri, desteklenmeyen bir
çıktı biçimi, geçersiz subplot boyutları, subplot hücrelerinden daha fazla
grafik, bir render hatası, bir geçici dosya hatası ve bir görüntüleyici-açma
hatası. Statik bir tip uyuşmazlığı -- sayısal bir List beklenen yerde bir
`List<String>` geçmek -- sıradan bir derleme-zamanı tanılaması olarak kalır;
`PlotError`, tip denetleyicisinin önceden eleyemediği alan ve çalışma zamanı
hataları için ayrılmıştır.

## Girdi asla değiştirilmez

Her Plot fonksiyonu ve Chart metodu, List argümanlarının bir anlık
görüntüsünü (snapshot) okur; hiçbiri çağıranın List'ini yeniden sıralamaz
veya başka bir şekilde değiştirmez:

```ahd
values: List<Int> := [3, 1, 4, 1, 5]

chart := Plot.histogram(values, 5)
write(values)  // [3, 1, 4, 1, 5]
```

## Render (işleme)

Plot, `ahdcode` araç zincirinin yanında gönderilen küçük, gömülü bir render
yardımcısı (`ahdplot`) aracılığıyla, süreç dışında,
[Gonum](https://gonum.org)'un çizim kütüphanesiyle render eder. Bu,
uygulama arka ucunu dahili bir ayrıntı olarak tutar: hem kalıcı (persistent)
evaluator hem de doğal olarak derlenmiş programlar aynı yardımcıyı aynı
şekilde çalıştırır, böylece `Plot.*`, ister REPL ister
`ahdcode build`/`ahdcode run` üzerinden çalıştırılsın aynı şekilde davranır.

## Plot'un olmadığı şeyler

v0.1.14 tam olarak altı grafik ailesini destekler: line, scatter, bar,
histogram, box ve error bar. Pie, heatmap, contour, violin, stem, polar, 3D,
candlestick, area veya surface grafiği yoktur ve keyfi özel plotter
enjeksiyonu yoktur -- bunlar gelecekteki bir sürümde değerlendirilebilir.
`Int`/`Real` genişletmesinin ötesinde sayısal bir skaler tip yoktur (bir
`Numeric` tipi yoktur), genel bir GUI çerçevesi yoktur ve ikincil eksenler
yoktur.
