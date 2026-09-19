# Plot standart modülü

[English](PLOT.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Statistics](STATISTICS_TR.md) · [Modüller](MODULES_TR.md)

İlk kez öğreniyorsanız Data'dan sayısal liste üretme, grafik türü seçme ve
aynı grafiği Word/Latex raporuna gömme akışını gösteren
[Plot atölyesini](PRACTICAL_MODULES_TR.md#3-plot-veriyi-okunabilir-bir-grafiğe-dönüştürmek)
çalışın; bu sayfayı bütün grafik seçeneklerinin referansı olarak kullanın.

Plot, tipli sayısal List'lerden grafik çizer. Math, Time, Regex, CSV, Data
ve Statistics gibi açıktır (explicit):

```ahd
bring Plot
from Plot bring Chart
from Plot bring Figure
from Plot bring Surface
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

> v1.9.0'dan itibaren. Önceki sürümler grafiği işletim sisteminin
> görüntüleyicisiyle açıyordu.

`show()`, grafiği AhdCode'un kendi etkileşimli görüntüleyicisinde açar:
**AhdCode Plot** adlı, AhdCode simgeli bir pencere. Grafik `save()` ile
kaydedildiği hâliyle çizilir ve görüntüleyici onu incelemenizi sağlar.
v2.0'dan itibaren pencerenin üstündeki bir araç çubuğu kontrolleri gösterir:

| Araç çubuğu düğmesi | Tuşlar ve fare | İşlem |
| --- | --- | --- |
| Save | — | Grafiği PNG, SVG veya PDF olarak kaydetme (aşağıya bakın) |
| Zoom Out / Zoom In | Fare tekerleği veya trackpad kaydırma | Uzaklaştırma ve yakınlaştırma (tekerlek imlecin çevresinde) |
| — | Sol tuşla sürükleme | Kaydırma (pan) |
| Rotate Left / Rotate Right | `Q` / `E` | Görünümü sola / sağa çeyrek tur döndürme |
| Fit | `R` | Sıfırlama: döndürme yok, grafik sığdırılmış, ortalanmış |
| — | `Escape` | Görüntüleyiciyi kapatma |

İmleç bir düğmenin üzerinde durduğunda düğmenin adı görünür. Araç çubuğu
görüntüleyiciye aittir: bir GUI bileşeni değildir ve programlar ona bir şey
ekleyemez.

Küçük bir gösterge yakınlaştırmayı ve dönüşü gösterir; ilk saniyelerde
kontrollerin tek satırlık bir özeti görünür. Görüntüleyici grafiğin
boyutunda (ekrana sığacak şekilde) açılır ve yeniden boyutlandırılabilir.
Yakınlaştırma, sığdırılmış boyutun dörtte birinden %1600'e kadardır;
pencereden küçük bir grafik ortada kalır, yakınlaştırılmış bir grafik ise
pencereyi her zaman kaplar, böylece asla kaybolmaz — `R` grafiğin tamamını
geri getirir. Dönüşler yalnızca çeyrek turdur (0°, 90°, 180°, 270°).

**Görüntüleyici yalnızca görünümü değiştirir.** Yakınlaştırma, kaydırma ve
döndürme `Chart`'ı veya `Figure`'ı, verisini ya da eksenlerini ve sonradan
kaydedilen dosyayı asla değiştirmez: `show()` sonrasında
`chart.save("result.png")`, `show()` hiç çağrılmamış gibi aynı dosyayı yazar.
Görüntüleyici için bir API yoktur; `show()` parametre almaz ve
görüntüleyicide menü, düzenleme veya veri seçimi yoktur.

**Save** (v2.0), sistemin kaydetme iletişim kutusunda PNG, SVG ve PDF
sunarak bir dosya adı sorar; biçimi uzantı belirler. Görüntüleyici kendi
penceresinin ekran görüntüsünü almaz: aynı render aracını grafiğin kendi
tanımıyla çalıştırır; böylece dosya, görünüm nasıl yakınlaştırılmış veya
döndürülmüş olursa olsun `chart.save(path)` veya `figure.save(path)`'in
yazdığıyla bayt bayt aynıdır. Kaydetme, `show()`'u çağıran program bittikten
sonra da çalışır. Sonuç — ya da neden başarısız olduğu — birkaç saniye araç
çubuğunun yanında görünür.

`show()`, görüntüleyici penceresi açılır açılmaz döner. Program devam eder
ve `show()`'u yeniden çağırabilir; her çağrı kendi görüntüleyicisini açar ve
bir görüntüleyici, program bittikten sonra bile siz kapatana kadar açık
kalır. `show()` grafiği sistem geçici dizininin AhdCode'a özgü bir alanında
geçici bir PNG'ye render eder; görüntüleyici onu tamamen okur ve `show()`
dönmeden önce siler, böylece önizleme dosyaları birikmez.

Görüntüleyici açılamadığında `show()` `PlotError` fırlatır: paketli
görüntüleyici eksiktir, bir ekran yoktur (CI, konteyner, uzak kabuk) ya da
grafik bir kenarda 6144 birimden büyüktür (grafiği küçültün veya
`save()` ile kaydedin). AhdCode hiçbir zaman başka bir uygulamaya geri
dönmez.

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

## Surface

> v2.0.0 ile eklendi.

```ahd
bring Plot
bring Math
bring Numeric
from Plot bring Surface

x: List<Real> := [-2.0, -1.0, 0.0, 1.0, 2.0]
y: List<Real> := [-1.0, 0.0, 1.0]
rows: List<List<Real>> := []
for yValue in y {
    row: Local List<Real> := []
    for xValue in x {
        row.add(xValue * xValue - yValue * yValue)
    }
    rows.add(row)
}
surface: Surface := Plot.surface(x, y, Numeric.matrix(rows)).title("Saddle").zLabel("height")
surface.save("saddle.png")
surface.show()
```

```text
Plot.surface(x, y, z: Matrix) -> Surface

Surface.title(text: String) -> Surface
Surface.xLabel(text: String) -> Surface
Surface.yLabel(text: String) -> Surface
Surface.zLabel(text: String) -> Surface
Surface.size(width: Int, height: Int) -> Surface
Surface.wireframe(enabled: Bool) -> Surface
Surface.save(path: String) -> Nothing
Surface.show() -> Nothing
```

`Plot.surface`, `z` yükseklik alanını bir ızgara üzerinde çizer: `x` ve `y`
her biri `List<Int>`, `List<Real>` veya bir Numeric `Vector`'dür; `z` ise
**her `y` değeri için bir satır ve her `x` değeri için bir sütun** içeren bir
[Numeric](NUMERIC_TR.md) `Matrix`'tir — `z[j][i]`, `(x[i], y[j])`
noktasındaki yüksekliktir.

- `x` ve `y` her biri kesin artan 2 ile 256 değer tutar; ızgara en fazla
  65.536 nokta içerir. Uyuşmayan bir Matrix, çok az veya çok fazla değer ya da
  sırasız koordinatlar `PlotError` fırlatır. Her değer sonludur (NaN ve
  sonsuz hiçbir zaman bir AhdCode programına ulaşmaz).
- Bir Surface, Chart gibi immutable'dır: `title`, etiketler, `size` ve
  `wireframe` yeni bir Surface döndürür. Eksen etiketleri varsayılan olarak
  `x`, `y` ve `z`'dir; boyut varsayılan olarak 800 × 600 birimdir ve bir
  kenarda en fazla 2000'dir.
- Yüzey sabit bir yükseklik renk ölçeğiyle (koyu maviden yeşile ve sarıya)
  doldurulur ve biçimi bir bakışta okunsun diye sabit bir ışıkla
  gölgelendirilir; `wireframe(true)` yalnızca yüksekliğe göre renklenen ızgara
  çizgilerini çizer. Kutunun tabanı, eksen adları ve her eksenin aralığı onu
  çerçeveler. Renk haritası, ışıklandırma, malzeme veya kamera ayarı yoktur.
- `save(path)` **yalnızca PNG** yazar, birim başına 4/3 piksel (800 × 600
  birim 1067 × 800 piksel eder), başlangıç görünümünden çizilir; başka bir
  uzantı `PlotError`'dır. Belirlenimcidir: aynı Surface aynı bilgisayarda
  hep aynı baytları yazar. Yalnızca bir resmi saran bir vektör dosyası
  üretmek yerine, Surface için SVG veya PDF yoktur.

`show()`, Surface'i aynı görüntüleyicide 3B olarak açar:

| Araç çubuğu düğmesi | Tuşlar ve fare | İşlem |
| --- | --- | --- |
| Save | — | Surface'in PNG'sini tam `save(path)`'in yazdığı gibi kaydetme |
| Zoom Out / Zoom In | Fare tekerleği | Uzaklaştırma ve yakınlaştırma |
| — | Sol tuşla sürükleme | Yüzeyin çevresinde döndürme (orbit) |
| — | Shift+sürükleme veya sağ tuşla sürükleme | Kaydırma (pan) |
| Reset View | `R` | Başlangıç görünümüne dönme |
| — | `Escape` | Görüntüleyiciyi kapatma |

Görünüm ortografik bir izdüşümdür. Tüm yüzey görünür olacak şekilde sabit bir
açıdan başlar; döndürme dikey eksen çevresinde serbesttir ve tam yukarıdan ve
tam aşağıdan biraz önce durur, böylece görünüm hiç ters dönmez;
yakınlaştırma 0,3× ile 6× arasındadır ve kaydırma yüzeyi erişilebilir
tutar. Grafik görüntüleyicisinde olduğu gibi kamera yalnızca görünüme
aittir: Surface'in parçası değildir, kamera API'si yoktur ve kaydetme her
zaman başlangıç görünümünü yazar.

Bu küçük bilimsel 3B çizimdir, bir 3B motoru değildir: ağ (mesh), içe
aktarılan modeller, dokular, ışık veya malzeme ayarları, sahne grafiği ya da
başka 3B ilkel nesneler yoktur ve v2.0'da 3B saçılım grafiği yoktur.

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
grafik, geçersiz bir Surface ızgarası veya boyutu, bir render hatası, bir geçici dosya hatası ve bir görüntüleyici-açma
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
Bir Surface'i hem ekranda hem de `Surface.save` için görüntüleyici yardımcısı
(`ahdplotview`) kendi küçük yazılım izdüşümüyle çizer; görüntüleyicinin Save
düğmesi kaydetme iletişim kutusu için GUI yardımcısını (`ahdgui`) kullanır.

## Plot'un olmadığı şeyler

Plot altı 2B grafik ailesini — line, scatter, bar, histogram, box ve error
bar — ve v2.0'dan itibaren 3B Surface'i destekler. Pie, heatmap, contour,
violin, stem, polar, 3B saçılım, candlestick veya area grafiği yoktur ve
keyfi özel plotter enjeksiyonu yoktur -- bunlar gelecekteki bir sürümde
değerlendirilebilir.
`Int`/`Real` genişletmesinin ötesinde sayısal bir skaler tip yoktur (bir
`Numeric` tipi yoktur), genel bir GUI çerçevesi yoktur ve ikincil eksenler
yoktur.
