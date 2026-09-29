# Plot standart modülü

[English](PLOT.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Statistics](STATISTICS_TR.md) · [Modüller](MODULES_TR.md)

**Matematik yolu:** [Math](MATH_TR.md) → [Numeric](NUMERIC_TR.md) → [Statistics](STATISTICS_TR.md) → **Plot**

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

## Bu sayfada

- Başlarken: [bir fonksiyon çizmek](#hızlı-başlangıç-bir-fonksiyon-çizmek),
  [veri noktalarını çizmek](#hızlı-başlangıç-veri-noktalarını-çizmek)
- Grafikler: [grafik türleri](#grafik-türleri), [sayısal girdi](#katı-sayısal-girdi-string-zorlaması-coercion-yok),
  [boş veri](#boş-veri), [başlık, etiket, legend, boyut](#grafik-meta-verisi),
  [stil](#plot-stili), [birden çok seri](#birden-çok-seri)
- [Matematiksel metin](#matematiksel-metin)
- Çıktı: [save](#save-kaydetme) (PNG, SVG, PDF), [show](#show-gösterme) (etkileşimli görüntüleyici)
- Diğer grafikler: [pasta](#pasta), [ısı haritası](#heatmap-ısı-haritası), [subplot'lar](#subplotlar),
  [Surface (3B)](#surface)
- Başvuru: [PlotError](#ploterror), [girdi asla değiştirilmez](#girdi-asla-değiştirilmez),
  [render](#render-işleme), [Plot'un olmadığı şeyler](#plotun-olmadığı-şeyler)

## Hızlı başlangıç: bir fonksiyon çizmek

`y = f(x)` çizmek için x'i [`Numeric.linspace`](NUMERIC_TR.md#vektör-oluşturma)
ile örnekleyin, her örnek için y'yi hesaplayın ve ikisini `Plot.line`'a verin.
Daha çok örnek daha düzgün bir eğri verir; çoğu fonksiyon için yüz kadarı
yeterlidir.

```ahd
bring Numeric
bring Plot

x := Numeric.linspace(-3.0, 3.0, 121)
y := Numeric.vector(x.values().map(lambda (value: Real) -> value ^ 2 - 2))
chart := Plot.line(x, y)
chart = chart.title(r'$f(x)=x^{2}-2$').xLabel("$x$").yLabel("$f(x)$")
chart.save("parabola.png")
chart.save("parabola.svg")
chart.save("parabola.pdf")
write("saved parabola.png, parabola.svg, parabola.pdf")
```

```text
saved parabola.png, parabola.svg, parabola.pdf
```

- Bir Chart değişmezdir: `title`, `xLabel` ve diğer metotlar yeni bir Chart
  döndürür; bu yüzden sonuç yeniden `chart`'a atanır. Bir metot zinciri tek
  satırda kalmalıdır; birkaç satırda yapılandırmak için her satırda yeniden
  atayın.
- Tamamı `$...$` arasında yazılan bir etiket matematik olarak dizilir (bkz.
  [Matematiksel metin](#matematiksel-metin)). Başlık `r'...'` ham String'i
  kullanır, çünkü sıradan bir String'deki `{2}`
  [String interpolasyonu](#ahdcode-stringlerinde-tex) olarak okunurdu.
- `save`, biçimi uzantıdan seçer. Grafiği etkileşimli
  [görüntüleyicide](#show-gösterme) açmak için bunun yerine `chart.show()`
  kullanın.

Math fonksiyonları da aynı biçimde kullanılır: `bring Math` ile
`lambda (value: Real) -> Math.sin(value)` bir sinüs eğrisi çizer.

## Hızlı başlangıç: veri noktalarını çizmek

Ölçülmüş veri için noktaları `Plot.scatter` ile çizin. Bir Chart birden çok
seri tutabilir; `chart.line` ve `chart.scatter` her biri bir legend
etiketiyle bir seri ekler. Bu örnek ölçümleri,
[`Statistics.linearRegression`](STATISTICS_TR.md#veri-çiftleri-kovaryans-korelasyon-ve-uydurulan-doğru)
ile uydurulan doğruyla birlikte çizer:

```ahd
bring Plot
bring Statistics
from Plot bring (Chart, LegendPosition)

hours: List<Int> := [1, 2, 3, 4, 5]
scores: List<Real> := [52.0, 57.5, 61.0, 68.5, 71.0]
fit := Statistics.linearRegression(x: hours, y: scores)
fitted: List<Real> := hours.map(lambda [@fit] (hour: Int) -> fit["slope"] * hour + fit["intercept"])

chart: Chart := Plot.new()
chart = chart.scatter(hours, scores, "measured")
chart = chart.line(hours, fitted, "least-squares line")
chart = chart.title("Study time and score").xLabel("hours").yLabel("score")
chart = chart.legend(true).legendPosition(LegendPosition.topLeft)
chart.save("fit.png")
write("saved fit.png")
```

```text
saved fit.png
```

`x` ve `y`'nin her biri bir `List<Int>`, bir `List<Real>` ya da bir Numeric
`Vector` olabilir. Tek bir List'in dağılımı için [`Plot.histogram` ya da
`Plot.box`](#grafik-türleri) kullanın.

## Grafik türleri

```text
Plot.line(x, y)                              -> Chart
Plot.scatter(x, y)                           -> Chart
Plot.bar(labels: List<String>, values)       -> Chart
Plot.pie(labels: List<String>, values)       -> Chart
Plot.heatmap(
    xLabels: List<String>
    yLabels: List<String>
    values: Matrix
) -> Chart
Plot.histogram(values, bins: Int)            -> Chart
Plot.box(values)                             -> Chart
Plot.errorBar(x, y, lowerErrors, upperErrors) -> Chart
Plot.new()                                   -> Chart
Plot.subplots(rows: Int, columns: Int, charts: List<Chart>) -> Figure
```

Tek bir grafik — line, scatter, bar, pie, heatmap, histogram, box veya error
bar — bir `Chart` üretir. Çoklu-grafik kompozisyonu bir `Figure` üretir
(bkz. [Subplot'lar](#subplotlar)).

[Pasta](#pasta) ve [Isı haritası](#heatmap-ısı-haritası) aşağıda anlatılır.

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
chart.legendPosition(position: LegendPosition) -> Chart
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

`legend` v2.2'nin iki grafiği dışında her grafik ailesi için varsayılan
olarak kapalıdır: bir [pastanın](#pasta) kategori anahtarı ve bir
[ısı haritasının](#heatmap-ısı-haritası) renk ölçeği, program kapatmadıkça açıktır;
çünkü ikisi de anahtarı olmadan okunamaz.

Legend açık olduğunda v2.4.0 varsayılan olarak onu grafik kenarlarından küçük
bir iç boşlukla `topRight` konumuna yerleştirir. Başka bir köşe gerekiyorsa
güçlü tipli konumlardan birini seçin:

```text
LegendPosition.topRight | LegendPosition.topLeft
LegendPosition.bottomRight | LegendPosition.bottomLeft
```

Örneğin: `chart = chart.legend(true).legendPosition(LegendPosition.topLeft)`.
Çizici, matematiksel etiketlerin kenara yapışmaması için girişler arasına ve
çizgi/marker örneği ile metin arasına da boşluk koyar.

Bir [pastanın](#pasta) Kartezyen ekseni yoktur; bu yüzden `xLabel` ve
`yLabel` bir pastada sessizce yok sayılmak yerine `PlotError` fırlatır.

## Plot stili

Line ve scatter serilerinin küçük, güçlü tipli bir stil API'si vardır. Stil
değerleri rastgele String değildir:

```text
LineStyle.solid | LineStyle.dashed | LineStyle.dotted | LineStyle.dashDot
Marker.none | Marker.circle | Marker.square | Marker.triangle | Marker.diamond | Marker.cross

chart.lineStyle(style: LineStyle) -> Chart
chart.lineWidth(width: Real)    -> Chart
chart.marker(shape: Marker)     -> Chart
chart.markerSize(size: Real)    -> Chart
```

Bu metotlar saftır ve zincirlenebilir. Bir line solid ve markersız başlar;
scatter v2.3'teki daire görünümünü korur. Genişlik ve marker boyutu sonlu,
pozitif çizim değerleridir; line'a özel seçeneği yalnızca scatter olan bir
Chart'a (veya tersi) uygulamak çalışma zamanında `PlotError` üretir. Legend,
serinin aynı line ve marker birleşimini kullandığından anahtar gerçeği korur:

```ahd
bring Plot
from Plot bring (Chart, LineStyle, Marker, LegendPosition)

x: List<Int> := [1, 2, 3, 4]
y: List<Real> := [2.0, 5.0, 4.0, 8.0]
chart: Chart := Plot.new().line(x, y, "measurement")
chart = chart.lineStyle(LineStyle.dashDot).lineWidth(2.0)
chart = chart.marker(Marker.diamond).markerSize(7.0)
chart = chart.legend(true).legendPosition(LegendPosition.topLeft)
chart.save("styled.png")
```

## Matematiksel metin

**Tüm** içeriği tek bir `$...$` çifti arasında olan bir metin argümanı, pakete
gömülü çevrimdışı Tectonic (TeX) motoruyla matematik olarak dizilir:

```ahd
bring Plot

x: List<Int> := [0, 1, 2, 3]
y: List<Int> := [2, 6, 12, 20]
chart := Plot.new().line(x, y, "$f(x)$")
chart = chart.title(r'$f(x)=x^{2}+3x+2$').xLabel("$x$").yLabel("$y$")
chart = chart.legend(true)
chart.save("math.png")
write("saved math.png")
```

```text
saved math.png
```

**Nerede çalışır.** Chart başlıkları ve eksen etiketleri; line ve scatter
legend etiketleri; pasta etiketleri ve legend'ları; bar kategorileri; ısı
haritası kategorileri; Surface başlıkları, eksen etiketleri, z etiketleri ve
`xCategories`/`yCategories`.

**Bütün String kuralı.** Yalnızca `$` ile başlayıp eşleşen `$` ile biten bir
String matematiktir:

| Metin | Sonuç |
|---|---|
| `"$\\sigma^2$"` | dizilmiş σ² |
| `"Cost $100"` | sıradan metin |
| `"Variance is $\\sigma^2$"` | sıradan metin; dolar işaretleri ve ters bölü olduğu gibi görünür |

Tek bir String içinde karışık metin ve matematik **desteklenmez**: parçalara
ayrılmaz, reddedilmez de; yalnızca sıradan metin olarak çizilir. Bunun yerine
etiketin tamamını matematik olarak yazın; sözcükleri `\mathrm{...}` ile,
boşluğu `\ ` ile yazın: `r'$\mathrm{Varyans}\ \sigma^{2}$'`. (`\text{...}`
paketli fontlarla kullanılamaz ve `PlotError` verir.)

### AhdCode String'lerinde TeX

TeX `\` ve `{}` kullanır ve ikisinin de sıradan bir AhdCode String'inde bir
anlamı vardır: `\` bir kaçış dizisi başlatır, `{...}` ise interpolasyondur.
`"$x^{2}$"` içinde `{2}`, `2` ile değiştirilir; `"$\\frac{a}{b}$"` ise `a` ve
`b` değişkenlerini okumaya çalışır. TeX'i ikisinin de özel olmadığı bir **ham
String** içinde yazın:

```ahd
write(r'$\frac{a}{b}$')
write("$x^{2}$")
```

```text
$\frac{a}{b}$
$x^2$
```

Sıradan bir String'de `\\` tek bir ters bölü, `\{`/`\}` ise düz süslü parantez
verir: `"$e^\{i\\pi\}$"`, `r'$e^{i\pi}$'` ile aynıdır.

**Desteklenen TeX.** Kesirler (`\frac`), kökler (`\sqrt`), toplamlar,
integraller, alt ve üst indisler, Yunan harfleri ve diğer sıradan matematik
kipi komutları display style ile çizilir. Bir parçanın boyutu sınırlıdır,
kontrollü bir komut kümesine göre denetlenir ve yalnızca paketli çevrimdışı
bundle ile izole bir geçici dizinde derlenir; dosya, shell, belge, ağ ve paket
yükleme komutları reddedilir. Bozuk ya da reddedilen TeX `PlotError` verir;
Plot hiçbir zaman ham TeX'e ya da bir Unicode yaklaşımına geri düşmez. Plot ve
Surface sınırlı tek bir renderer cache'ini paylaşır; aynı etiket iki kez
derlenmez.

**Çıktı biçimleri.** PNG, SVG ve PDF aynı Tectonic sonucunu kullanır. SVG ve
PDF'de grafik vektör olarak kalır, her matematik etiketi ise saydam, yüksek
çözünürlüklü bir görsel olarak gömülür. Glyph'ler yalnızca bundle'a gömülü
fontlardan gelir — makinenin font dizinlerinden, harici bir PDF sürecinden ya
da ağdan asla — ve eksik bir glyph sessiz bir yedek yerine `PlotError`'dır.

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

Çıktı biçimi dosya uzantısından çıkarılır:

| Çıktı | Chart / Figure | Surface |
|---|---|---|
| `.png` | raster görsel | raster görsel (tek biçim) |
| `.svg` | vektör; matematik etiketleri görsel olarak gömülü | yok |
| `.pdf` | vektör; matematik etiketleri görsel olarak gömülü | yok |

Başka her uzantı `PlotError` fırlatır:

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

## Pasta

> v2.2'de eklendi.

`Plot.pie(labels, values)`, verilen sırayla, saat on iki yönünden saat
yönünde kategori başına bir dilim çizer:

```ahd
chart := Plot.pie(
    ["Analiz", "Cebir", "Geometri", "İstatistik", "Programlama"],
    [84, 81, 88, 83, 95]
)
chart = chart.title("3. Yıl ders dağılımı")
chart.show()
```

`labels` ve `values` aynı uzunlukta olmalı ve boş olmamalıdır. `values`,
diğer her sayısal Plot argümanı gibi `List<Int>` ya da `List<Real>`'dir. Her
değer **sonlu ve negatif olmayan** bir sayı olmalı ve **en az biri sıfırdan
büyük** olmalıdır — yalnızca sıfırlardan oluşan bir pastanın çizilecek
şekli yoktur ve `PlotError` fırlatır. Pozitif değerler arasındaki bir sıfır
sıradan veridir: o kategorinin dilimi olmaz, gösterge yine de onu adlandırır.

Pasta bir `Chart`'tır; `title`, `legend`, `size`, `save` ve `show` üzerinde
tam olarak bir çubuk grafikteki gibi çalışır.

**Gösterge varsayılan olarak açıktır.** Dilimler renkle ayırt edilir ve bir
renk, yanındaki ad olmadan bir şey ifade etmez; bu yüzden `Plot.pie`
anahtarı sizin için açar. `chart.legend(false)` onu gizler.

**Bir pastanın ekseni yoktur.** Bir pastada `chart.xLabel(...)` ve
`chart.yLabel(...)` `PlotError` fırlatır:

```ahd
attempt {
    Plot.pie(["A", "B"], [1, 2]).xLabel("kategori")
}
except PlotError as error {
    write(error.message)
}
```

Bu bilinçlidir: çağrıyı sessizce yok saymak, grafikle ilgili bir yanlış
anlamayı gizlerdi.

Toplamın yüzde beşinden büyük her dilim, payını tam sayı yüzde olarak
taşır. Renkler, on iki tonluk tek bir sabit kategorik paletten gelir ve
daha çok dilimli bir pastada sırayla yeniden kullanılır; v2.2'de palet
argümanı yoktur, bu yüzden aynı veri her zaman aynı resmi çizer. Bir pasta
en fazla 64 dilim çizer.

Bir pasta, diğer her Chart gibi PNG, SVG ve PDF'e kaydedilir ve `show()`
onu sıradan [görüntüleyicide](#show-gösterme) açar.

## Heatmap (ısı haritası)

> v2.2'de eklendi.

`Plot.heatmap(xLabels, yLabels, values)`, rengi sayıları taşıyan etiketli
bir ızgara çizer:

```ahd
scores := Numeric.matrix([
    [72.0, 78.0, 84.0]
    [68.0, 75.0, 81.0]
    [80.0, 82.0, 88.0]
    [65.0, 74.0, 83.0]
    [85.0, 91.0, 95.0]
])

chart := Plot.heatmap(
    ["1. Yıl", "2. Yıl", "3. Yıl"]
    ["Analiz", "Cebir", "Geometri", "İstatistik", "Programlama"]
    scores
)
chart = chart.title("Öğrenci performansı")
chart = chart.xLabel("Akademik yıl")
chart = chart.yLabel("Ders")
chart.show()
```

**Şekil kuralı: y etiketi başına bir satır, x etiketi başına bir sütun.**
`values[row][column]` hücresi `yLabels[row]` ve `xLabels[column]`'a aittir —
yukarıdaki örnekte aşağı doğru beş ders, yana doğru üç yıl. Başka bir
şekildeki Matrix, hem sahip olduğu şekli hem etiketlerin gerektirdiğini
adlandıran bir `PlotError` fırlatır.

`values` bir [`Numeric`](NUMERIC_TR.md) `Matrix`'tir, yani hücreleri zaten
`Real`'dir; iç içe bir `List<List<Real>>` yayımlanmış argüman değildir ve bir
programın birini kurma yolu `Numeric.matrix(rows)`'tur. Negatif hücreler
geçerlidir; her hücresi eşit bir ızgara da geçerlidir. NaN ya da sonsuz bir
hücre çizilemez ve `PlotError` fırlatır.

Etiket listelerinin hiçbiri boş olamaz. Bir ısı haritasının her ekseninde en
fazla 256 etiket ve toplamda en fazla 65.536 hücre bulunur.

Kategoriler tam olarak verilen sırayla, soldan sağa ve aşağıdan yukarıya,
her hücrenin merkezinde bir tick ile çizilir.

**Renk ölçeği sabittir ve göstergesi varsayılan olarak açıktır.** Ölçek
koyudan parlağa gider — Moreland'in kara cisim ölçeği; parlaklığı tekdüze
arttığı için gri tonlamada ve yaygın renk körlüklerinde de okunur — ve
ızgaranın yanındaki çubuk her rengin ne anlama geldiğini gösterir.
`chart.legend(false)` çubuğu gizler; hücreler aynı renkleri korur. v2.2'de
colormap, en küçük ya da en büyük değer argümanı yoktur: ölçek her zaman
veriyi kaplar.

Isı haritası bir `Chart`'tır; `title`, `xLabel`, `yLabel`, `legend`, `size`,
`save` ve `show` üzerinde çalışır, PNG, SVG ve PDF'e kaydedilir ve bir
[Figure](#subplotlar)'ın hücresi olabilir.

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
hücre-başı bütçe); `Figure.size` metodu yoktur.

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
Surface.xCategories(labels: List<String>) -> Surface
Surface.yCategories(labels: List<String>) -> Surface
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
| Save | — | Görüntüleyicideki mevcut kamera görünümünü PNG olarak kaydetme |
| Zoom Out / Zoom In | Fare tekerleği | Uzaklaştırma ve yakınlaştırma |
| — | Sol tuşla sürükleme | Yüzeyin çevresinde döndürme (orbit) |
| — | Shift+sürükleme veya sağ tuşla sürükleme | Kaydırma (pan) |
| Reset View | `R` | Başlangıç görünümüne dönme |
| — | `Escape` | Görüntüleyiciyi kapatma |

Görünüm ortografik bir izdüşümdür. Tüm yüzey görünür olacak şekilde sabit bir
açıdan başlar; döndürme dikey eksen çevresinde serbesttir ve tam yukarıdan ve
tam aşağıdan biraz önce durur, böylece görünüm hiç ters dönmez;
yakınlaştırma 0,3× ile 6× arasındadır ve kaydırma yüzeyi erişilebilir
tutar. Kamera yalnızca görüntüleyici görünümüne aittir: Surface'in parçası
değildir, kamera API'si yoktur. Görüntüleyicideki **Save**, o anki orbit,
eğim, zoom ve pan görünümünü kaydeder; programatik `Surface.save(path)` ise
her zaman başlangıç görünümünü yazar.

Bu küçük bilimsel 3B çizimdir, bir 3B motoru değildir: ağ (mesh), içe
aktarılan modeller, dokular, ışık veya malzeme ayarları, sahne grafiği ya da
başka 3B ilkel nesneler yoktur ve 3B saçılım grafiği yoktur.

### Koordinatları adlandırmak

> v2.2'de eklendi.

Bir Surface'in x ve y'si sayıdır; bu, iki değişkenli bir fonksiyon için
doğru, bir kategori ızgarası için yanlıştır: beş ders ve üç yıl üzerindeki
bir yüzey `1`–`5` ve `1`–`3` diye etiketlenir ve başlığının sayıların ne
anlama geldiğini açıklaması gerekir. `xCategories` ve `yCategories` o
koordinatlara ad verir:

```ahd
surface := Plot.surface([1, 2, 3], [1, 2, 3, 4, 5], scores)
surface = surface.xCategories(["1. Yıl", "2. Yıl", "3. Yıl"])
surface = surface.yCategories(["Analiz", "Cebir", "Geometri", "İstatistik", "Programlama"])
surface = surface.xLabel("Akademik yıl").yLabel("Ders").zLabel("Not")
surface.show()
```

**Yalnızca sunumdur.** Geometri değişmez: koordinatlar değerlerini ve
aralıklarını korur, Matrix değişmez ve çizilen şekil, onlarsız çizilenin tam
olarak aynısıdır. Yalnızca her eksenin yanındaki metin değişir.

**`xLabel` ve `yLabel` eksenleri adlandırmaya devam eder.**
`xLabel("Akademik yıl")` eksenin başlığıdır; `xCategories(["1. Yıl", …])` ise
eksen üzerindeki noktaların etiketleridir. İkisi, bilinçli olarak ayrı
şeyler için bilinçli olarak ayrı adlardır.

**Koordinat başına tam olarak bir etiket** olmalıdır. Başka uzunluktaki bir
liste, ekseni yanlış etiketlemek yerine `PlotError` fırlatır. Etiketler
yinelenebilir ve sıraları koordinatların sırasıdır.

Sekize kadar kategorinin hepsi, her biri kendi koordinatının yanında
çizilir; bundan sonrasında yalnızca ilk ve son çizilir, çünkü fazlası üst
üste binerdi — ki bu da sayısal bir eksenin her zaman gösterdiğidir,
sayılar yerine sözcüklerle.

Hiç kategori verilmezse eksen, ilk ve son değerini sayı olarak göstererek
v2.2'den önceki hâlinde kalır.

Kategoriler hem `show()` hem `save()` içinde çalışır. `Surface.save` yine
**yalnızca PNG** yazar.

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
hatası. v2.2 şunları ekler: eşleşmeyen pasta etiketleri/değerleri, boş pasta
verisi, negatif ya da sonlu olmayan bir pasta değeri, yalnızca sıfırlardan
oluşan bir pasta, bir pastada `xLabel` ya da `yLabel`, boş ısı haritası
etiketleri, etiketleriyle şekli uyuşmayan bir ısı haritası Matrix'i, sonlu
olmayan bir ısı haritası hücresi ve koordinatlarıyla uzunluğu uyuşmayan bir
Surface kategori listesi. Statik bir tip uyuşmazlığı -- sayısal bir List beklenen yerde bir
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

Plot sekiz 2B grafik ailesini — line, scatter, bar, pie, heatmap,
histogram, box ve error bar — ve v2.0'dan itibaren 3B Surface'i destekler.
Contour, violin, stem, polar, 3B saçılım, candlestick ya da area grafiği,
donut ya da patlatılmış pasta, ısı haritası açıklamaları ya da kümeleme ve
keyfi özel plotter enjeksiyonu yoktur -- bunlar gelecekteki bir sürümde
değerlendirilebilir. Renk sabittir: palet ya da colormap argümanı, tema ve
yazı tipi API'si yoktur. Eksenler de sabittir: biçimlendirici geri çağrıları,
keyfi tick yerleşimi, genel bir Axis nesnesi ve ikincil eksenler yoktur.
Plot genel bir GUI çerçevesi değildir; pencereler ve denetimler için
[GUI](GUI_TR.md) sayfasına bakın.

**İlgili:** çizdiğiniz fonksiyonlar için [Math](MATH_TR.md), örnek noktalar ve
Surface ızgaraları için [Numeric](NUMERIC_TR.md), çizdiğiniz veriyi özetlemek
için [Statistics](STATISTICS_TR.md).
