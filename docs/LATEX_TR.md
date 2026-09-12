# Latex standart modülü

[English](LATEX.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Time modülü](TIME_TR.md)

İlk kez öğreniyorsanız Article, Report, Beamer, denklem, tablo, Plot görseli
ve kaynakçayı tek akışta gösteren [Latex atölyesini](PRACTICAL_MODULES_TR.md#6-latex-akademik-pdf-ve-sunum-üretmek)
çalışın; bu sayfayı tam yardımcı ve derleme referansı olarak kullanın.

Latex, AhdCode String'lerini PDF belgelerine dönüştürür. Math ve Time gibi
açıktır (explicit) ve alias'lar dahil sıradan modül biçimleriyle çalışır:

```ahd
bring Latex
bring Latex as L
from Latex bring LatexError
```

Kanonik kimlik `builtin:Latex`'tir; kardeş bir `Latex.ahd` onun yerini
alamaz. Her argüman `NonNull` olmalıdır.

## Yüzey (Surface)

```text
pdf(source: String, output: String, sourceOutput: String = "") -> Nothing
pdfFile(input: String, output: String)   -> Nothing

escape(text: String)                     -> String

document(
    body: String, title: String = "", author: String = "", date: String = "",
    type: String = "Article", margin: Real = 2.54, color: String = "",
    cover: String = "", theorems: Pair<String, String> = {},
    theme: String = "Default", landscape: Bool = false,
    paper: String = "Letter", pageSize: Pair<String, Real> = {},
    margins: Pair<String, Real> = {}, subject: String = "",
    keywords: List<String> = [], creator: String = ""
)                                         -> String

chapter(title: String)                   -> String
section(title: String)                   -> String
subsection(title: String)                -> String
frame(title: String, body: String)       -> String

equation(source: String, label: String = "") -> String
theorem(type: String, body: String, label: String = "") -> String

table(headers: List<String>, rows: List<List<String>>, mathColumns: List<Int> = []) -> String
image(path: String, size: Pair<String, Real> = {}, transform: Pair<String, Real> = {}) -> String
figure(
    path: String, caption: String, label: String = "",
    size: Pair<String, Real> = {}, transform: Pair<String, Real> = {}
)                                         -> String

minipage(body: String, width: Real, alignment: String = "left") -> String
center(body: String)                     -> String
pageBreak()                              -> String
contents()                               -> String

ref(label: String)                       -> String
cite(key: String)                        -> String
bibliography(references: Pair<String, String>) -> String

tikz(source: String, libraries: List<String> = [])    -> String
overlay(source: String, libraries: List<String> = []) -> String
border(inset: Real = 1.0, thickness: Real = 1.0, color: String = "") -> String

LatexError
```

## Metin yardımcıları (Text helpers)

`escape`, **metin bağlamı (text-context)** kaçış (escaping) işlemidir.
TeX'e özel `\ { } $ & # % _ ^ ~` karakterlerini işler ve başka hiçbir şeyi
işlemez — ham matematiği (raw mathematics) sterilize ettiğini iddia etmez.

`chapter`, `section` ve `subsection`, başlıklarını (title) kaçış işlemine
tabi tutar. `equation` kasıtlı olarak kaçış işlemi **yapmaz**: amaç bu
olduğu için ham LaTeX matematik kaynağını olduğu gibi alır. Ham (raw)
String literalleri (v0.1.14) bunu yazmayı keyifli hale getirir, çünkü bir
ters eğik çizginin kendi kaçışına ihtiyacı yoktur:

```ahd
body += L.equation(
    r"\|x+y\| \leq \|x\|+\|y\|"
)
```

## Article, Report ve Beamer için TEK bir `document()`

Her desteklenen belge türü için, `type` parametresiyle seçilen TEK bir
`document(...)` fonksiyonu vardır — asla ayrı `Latex.report()` veya
`Latex.beamer()` fonksiyonları değil:

```ahd
source: String := L.document(
    body: body
    title: "Numerical Analysis"
    author: "Ali Harun"
    date: "31 August 2026"
    type: "Report"
    margin: 2.5
    color: "#1F4E79"
    cover: cover
    theorems: theoremTypes
    theme: "Default"
)
```

`type`, tam olarak `"Article"`, `"Report"` ve `"Beamer"` kabul eder;
varsayılan `"Article"`'dır. Var olan üç argümanlı bir çağrı,
`L.document(body, title, author)`, değişmeden çalışmaya devam eder ve her
yeni parametre varsayılan değerindeyken hâlâ bir `Article` üretir.

- **`date`** varsayılan olarak `""`'dir ve sistem tarihiyle otomatik olarak
  hiçbir zaman doldurulmaz — çıktı, çalıştırmalar ve makineler arasında
  belirlenimci (deterministic) kalır.
- **`margin`**, **santimetre** cinsinden tek bir belge-geneli değerdir,
  varsayılanı `2.54`'tür (etkin v0.1.14 yerleşimi). Pozitif olmalıdır.
  `margins` (v1.3.0) tek tek kenarları değiştirir, `paper` ve `pageSize`
  sayfayı seçer, yönelim ise ayrı `landscape` parametresidir.
- **`color`**, isteğe bağlı bir `#RRGGBB` vurgu rengidir (varsayılan olarak
  boş, v0.1.14 çıktısını tam olarak korur). Ayarlandığında, AhdCode
  tarafından üretilen vurgular için kullanılan bir `ahdaccent` rengi
  tanımlar — başlık/kapak alanı ve Beamer için, sunumun yapısal rengi.
  Geçersiz bir değer `ValueError` fırlatır.
- **`cover`**, sıradan üretilmiş LaTeX içeriğidir (varsayılan olarak boş),
  başlık sayfasından önce eklenir ve bir sayfa sonu izler; `cover` `""`
  olduğunda, başlık/yazar/tarih davranışı v0.1.14 ile bayt-bayt aynıdır.
  Sıralama her zaman kapak, sonra başlık, sonra gövdedir:
  ```ahd
  cover: String := L.center(
      L.image("logo.png", {"width": 5.0})
  )
  source := L.document(body: body, title: "Numerical Analysis", cover: cover)
  ```
- **`type: "Report"`**, `report` belge sınıfını kullanır ve `chapter`'ı
  etkinleştirir; **`type: "Beamer"`**, `beamer` belge sınıfını kullanır,
  başlığı `\maketitle` yerine bir başlık-sayfası çerçevesi olarak render
  eder ve aşağıda açıklanan dar slayt yüzeyini destekler.
- **`theme`**, tam olarak büyük/küçük harfe duyarlı `"Default"`, `"Madrid"`
  ve `"Warsaw"` değerlerini kabul eder, varsayılanı `"Default"`'tır ve onuncu
  konumsal parametredir; ardından yalnızca `landscape` gelir. Madrid ve Warsaw
  `type: "Beamer"` gerektirir; Article veya Report ile seçilmeleri
  `ValueError` fırlatır. Bilinmeyen tema adları da `ValueError` fırlatır ve
  LaTeX kaynağına hiçbir zaman doğrudan geçirilmez. Özel `color` theme'den
  sonra uygulanır; theme yerleşimini korurken yapısal vurgu rengini override
  eder.
- **`landscape`** (v1.2.0) varsayılan olarak `false`'tur. `true`, bir Article
  veya Report belgesinin her sayfasını aynı kağıt ve aynı `margin` ile yan
  çevirir. Genel bir yerleşim anahtarıdır, bir sertifika kipi değildir. Beamer
  slaytları zaten geniştir; bu yüzden `type: "Beamer"` ile `landscape: true`
  `ValueError` fırlatır.
- **`paper`** (v1.3.0) varsayılan olarak `"Letter"`'dır, yani her Latex
  belgesinin şimdiye kadar kullandığı US Letter sayfası; tam olarak `"A3"`,
  `"A4"`, `"A5"`, `"Letter"`, `"Legal"` veya `"Custom"` kabul eder.
  `"Custom"`, santimetre cinsinden hem `"width"` hem `"height"` içeren
  **`pageSize`** gerektirir; başka bir kağıtla `pageSize` vermek `ValueError`
  fırlatır.
- **`margins`** (v1.3.0), santimetre cinsinden `"top"`, `"right"`, `"bottom"`
  ve `"left"` anahtarlarından istediklerini kabul eder; verilmeyen kenar
  `margin` değerini kullanır. Her uzunluk 0'dan büyük ve en çok 1000 olmalı,
  kenar boşlukları içerik için yer bırakmalıdır. `type: "Beamer"` ile
  varsayılan dışı bir `paper`, bir `pageSize` veya `margins` `ValueError`
  fırlatır: slaytların kendi boyutu vardır.
- **`subject`**, **`keywords`** ve **`creator`** (v1.3.0) PDF belge
  özelliklerini ayarlar. Bunlardan biri ayarlandığında `title` ve `author` da
  özellik olarak kaydedilir. Her anahtar sözcük boş olmamalı ve virgül
  içermemelidir. Üçü de boş bırakılırsa belge, v1.3.0 öncesiyle bayt bayt
  aynıdır.

```ahd
source := L.document(
    body: body
    title: "Quarterly Report"
    author: "AhdCode Analytics"
    paper: "A4"
    margins: {"top": 3.0, "bottom": 2.5}
    subject: "Quarterly results"
    keywords: ["report", "2026"]
    creator: "AhdCode"
)
```

## Article, Report, Beamer

**Article**, `type` atlandığında veya `"Article"` olduğunda değişmeyen,
var olan v0.1.14 temel çizgisidir.

**Report**, gerçekten `report` belge sınıfını kullanır ve var olan
`section`/`subsection`'ın üzerine `chapter`'ı ekler:

```ahd
body += L.chapter("Introduction")
body += L.section("Background")
```

**Beamer**, paketlenmiş kaynak paketiyle gerçekten çevrimdışı derlenir —
sistem TeX yok, ağ yok, çalışma zamanı indirmesi yok. Kapsamı kasıtlı
olarak dardır: `document`, `frame`, `section`, `equation`, `table`,
`image` ve `contents`. Theme desteği bilinçli olarak Default, Madrid ve
Warsaw ile sınırlıdır; keyfi theme passthrough yoktur. Overlay, `\pause`,
geçiş, konuşmacı notu, özel navigasyon sembolleri veya bir columns
soyutlaması yoktur. `frame` bir slayt oluşturur:

```ahd
slides: String := ""
slides += L.frame("Contents", L.contents())
slides += L.frame("First Slide", L.equation(r"E = mc^2"))

presentation := L.document(
    body: slides
    title: "Talk"
    type: "Beamer"
    theme: "Madrid"
    color: "#1F4E79"
)
```

## Denklem etiketleri ve `ref`

`equation(source, label)`, isteğe bağlı bir etiket (label) alır. Tek bir
`ref(label)`, `equation`, `theorem` veya `figure` tarafından üretilen bir
etiketi çözer — ayrı `eqRef`/`theoremRef`/`figureRef` fonksiyonları yoktur:

```ahd
body += L.equation(
    r"\|x+y\| \leq \|x\|+\|y\|"
    "eq:triangle"
)
body += "See " + L.ref("eq:triangle") + "."
```

## Kullanıcı tanımlı teorem türleri

Tek bir genel `theorem(type, body, label)` yardımcısı vardır — asla ayrı
`lemma`/`definition`/`corollary`/`proposition`/`remark` fonksiyonları
değil. Mevcut teorem türleri ve her birinin sayacının (counter) nasıl
davrandığı, `document(theorems: ...)` aracılığıyla yapılandırılır:

```ahd
theoremTypes: Pair<String, String> := {
    "Theorem": "section"
    "Lemma": "Theorem"
    "Definition": "section"
    "Corollary": "Theorem"
}

source := L.document(body: body, type: "Article", theorems: theoremTypes)

body += L.theorem(type: "Theorem", body: "Every finite-dimensional normed space is complete.", label: "thm:finite")
```

Pair'in **anahtarı**, herkese açık teorem türü adıdır; **değeri**, sayaç
kuralıdır:

```text
""            -> bağımsız, belge-geneli bir sayaç
"section"     -> section ile sıfırlanır
"subsection"  -> subsection ile sıfırlanır
"chapter"     -> chapter ile sıfırlanır (yalnızca Report belgeleri)
"<tür adı>"   -> o (zaten bildirilmiş) türün sayacını paylaşır
```

`"Lemma": "Theorem"` ve `"Corollary": "Theorem"` ile birlikte
`"Theorem": "section"`, kavramsal olarak `Theorem 1.1`, `Lemma 1.2`,
`Corollary 1.3` gibi numaralandırır — üç tür, her bölümde sıfırlanan tek
bir sayacı paylaşır.

Bir görünen ad (display name) asla ham bir TeX tanımlayıcısı olmaz: her
teorem türü, üretilmiş, çakışmadan güvenli (collision-safe) bir dahili ad
alır. `document()`, boş bir tür adını, hiç kaydedilmemiş bir tür için
yapılan bir `theorem()` çağrısını, bilinmeyen veya henüz bildirilmemiş bir
türü adlandıran bir paylaşılan-sayaç kuralını (bu aynı zamanda kendine
referansı veya döngüsel bir referansı da yakalar) ve bir Report belgesi
dışındaki bir `"chapter"` kuralını `LatexError` olarak reddeder.

## Image ve figure

`image(path, size, transform)` numaralandırılmamış bir figür parçasıdır;
`figure(path, caption, label, size, transform)` numaralandırılmış, altyazılı
ve (bir etiketle) `ref` üzerinden referans verilebilir. `size`, yalnızca
`"width"`/`"height"` anahtarlarıyla `Pair<String, Real>`'dır, santimetre
cinsindendir: yalnızca genişlik veya yalnızca yükseklik en-boy oranını
korur, ikisi birden açıkça sığdırılır ve boş bir Pair görüntünün doğal
boyutunu kullanır.

```ahd
body += L.image("logo.png", {"width": 6.0})
body += L.figure("result.pdf", "Numerical solution", "fig:solution", {"width": 12.0})
```

Desteklenen biçimler PNG, PDF, JPEG ve (v1.3.0) vektör olarak kalan SVG'dir;
bkz. [SVG varlıkları](#svg-varlıkları-v130).

`transform` (v1.3.0) şu anahtarlara sahip bir `Pair<String, Real>`'dır:

| Anahtar | Anlamı | Aralık |
|---|---|---|
| `"rotation"` | saat yönünün tersine derece | -360 ile 360 |
| `"opacity"` | 0 görünmez, 1 tamamen opak | 0 ile 1 |
| `"trimLeft"`, `"trimTop"`, `"trimRight"`, `"trimBottom"` | boyutlandırılmış görselin o kenarından kesilen santimetre | 0 ile 1000 |

Görsel önce boyutlandırılır, sonra kırpılır, döndürülür ve soldurulur; vektör
içerik vektör olarak kalır. Bilinmeyen bir anahtar, aralık dışı bir değer veya
genişliğin ya da yüksekliğin tamamını kaldıracak kırpmalar `ValueError`
fırlatır. Dönüşüm verilmezse bir görsel parçası v1.3.0 öncesiyle tamamen
aynıdır.

```ahd
body += L.image("photo.jpg", {"width": 8.0}, {"trimTop": 1.0, "trimBottom": 1.0})
body += L.figure("stamp.png", "Approved", "fig:stamp", {"width": 4.0}, {"rotation": 12.0, "opacity": 0.6})
```

Subfigure'lar veya açığa çıkarılmış `graphicx`/float yerleştirme seçenekleri
yoktur.

### Varlık (asset) hazırlama

`pdf`/`pdfFile`, izole bir geçici çalışma alanında derlenir, bu yüzden bir
görüntü yolunun orada var olduğu varsayılamaz. `image`/`figure`, yollarını
derlenen programın çalışma dizinine göre çözer (`chart.save` ve `File`'ın
kullandığı aynı kural) ve o dosyanın bir kopyasını otomatik olarak derleme
çalışma alanına yerleştirir (stage) — hiçbir geliştirme-deposu (dev-repo)
yolu, kazara çalışma-dizini davranışı, sistem TeX veya ağ erişimi söz
konusu değildir:

```ahd
chart.save("chart.png")

body += L.figure("chart.png", "Results", "fig:results", {"width": 12.0})

source := L.document(body: body, type: "Report")
L.pdf(source: source, output: "report.pdf")
```

Eksik veya okunamayan bir varlık, sessizce bozuk bir PDF değil, derleme
zamanında fırlatılan bir `LatexError`'dır. `pdfFile`'ın var olan belgeye
göreli varlık çözümlemesi değişmeden kalır.

## Yerleşim (layout) yardımcıları

```ahd
left := L.minipage(leftBody, 7.0, "left")
right := L.minipage(rightBody, 7.0, "right")
body += left + right

body += L.center(
    L.minipage(content, 10.0, "center")
)

body += L.pageBreak()
```

`minipage`'in `width`'i santimetredir; `alignment` tam olarak `"left"`,
`"center"` veya `"right"`'tır, minipage içindeki içeriğe uygulanır.
`center`, ayrı, daha basit bir sarmalayıcıdır (wrapper). CSS benzeri bir
yerleşim sistemi ve grid/flex soyutlaması yoktur.

`contents()`, Article/Report için bir içindekiler (table of contents)
parçası üretir:

```ahd
body += L.contents()
```

Beamer için, `contents()` sessizce bir frame'e dönüşmez — frame'i açıkça
yazın:

```ahd
slides += L.frame("Contents", L.contents())
```

## Alıntılar (citations) ve kaynakça (bibliography)

`cite(key)`, `ref`'ten (bir denklem/teorem/figür etiketine dahili belge
referansı) ayrı tutulan bir kaynakça alıntısıdır:

```ahd
body += "As shown in " + L.cite("Hardy1934") + "."
```

`bibliography(references)`, ekleme sırasına (insertion order) göre bir
`Pair<String, String>` alıntı-anahtarı-metin çiftinden bir referans
listesi render eder:

```ahd
references: Pair<String, String> := {
    "Yildiz2016": "B. Yıldız, Article title, Journal Name, 2016."
    "Hardy1934": "G. H. Hardy, J. E. Littlewood and G. Pólya, Inequalities, 1934."
}

body += L.bibliography(references)
```

Latex asla referansları sıralamaz, yazar/yıl/dergi çıkarımı yapmaz, APA
veya IEEE biçimlendirmez, BibTeX kullanmaz, bir `.bib` dosyası
gerektirmez veya sağlanan metni yeniden yazmaz — değer tam olarak verildiği
gibi kullanılır.

## Table

`table`, v0.1.14'ten değişmemiştir: deterministik `booktabs` kaynağı, her
hücre kaçışlı ve `mathColumns: List<Int>`, belirli sıfır tabanlı sütunları
kaçış yerine ham satır içi matematiğe (`\( ... \)`) dahil eder. Yukarıdaki
v0.1.14 davranışına bakın; v0.1.15 için bu konuda hiçbir şey değişmedi.

## TikZ ile vektör grafik (v1.2.0)

Kenarlıklar, süslemeler, mühürler, rozetler, filigranlar, diyagramlar, oklar ve
konumlandırılmış etiketler metinle aynı Latex belgesine aittir. TikZ/PGF vektör
çizimin temelidir ve Latex çalışma zamanıyla birlikte çevrimdışı paketlenir; bu
yüzden bir PDF'i süslemek hiçbir zaman ikinci bir PDF kütüphanesi ya da önceden
üretilmiş bir kenarlık görseli gerektirmez.

```text
belge dizgisi (typography) -> Latex
vektör grafik              -> TikZ/PGF, Latex.tikz ve Latex.overlay ile
hazır süslemeler           -> pgfornament
raster görseller           -> Latex.image ve Latex.figure
PDF çıktısı                -> Latex.pdf
```

TikZ, TikZ olarak kalır. AhdCode çizim komutlarını çevirmez ve `line`,
`circle` veya `path` sarmalayıcıları yayınlamaz: aşağıdaki yardımcılar TikZ
kaynağınızı belgeye değiştirmeden yerleştirir ve yalnızca adını verdiği
paketlenmiş kütüphaneleri yükler.

### TikZ'i ham üçlü String'lerle yazın

Ham üçlü String, `r"""..."""`, ters eğik çizgileri, süslü ve köşeli
parantezleri ve `%` işaretini tam yazıldığı gibi korur, birden çok satıra yayılır
ve `{...}` interpolasyonu yapmaz; böylece bir düğüm içindeki `{AhdCode}` metin
olarak kalır. Normal bir String'de aynı süslü parantezler interpolasyon olurdu.

```ahd
bring Latex as L

drawing: String := L.tikz(r"""
\draw (0,0) rectangle (4,2);
\node at (2,1) {AhdCode};
""")
write(drawing)
```

Program verisini TikZ içine koymak için ham parçaları kaçışlanmış metinle
birleştirin; ham parçalar TikZ, veri ise metin olarak kalır:

```ahd
bring Latex as L

name: String := "Ayşe & Ali"
label: String := L.tikz(r"\node[draw] {" + L.escape(name) + r"};")
write(label)
```

### tikz

`tikz(source, libraries)`, metin akışında bir görsel gibi duran bir
`tikzpicture` parçası döndürür; bu yüzden `center`, `minipage` veya bir frame
içinde çalışır. Tüm resmin seçenekleri kaynağın içine yazılır, örneğin
`\begin{scope}[scale=2] ... \end{scope}`.

### overlay

`overlay(source, libraries)` metne göre değil, sayfanın kendisine göre çizer.
Kaynağı TikZ'in sayfa çapalarını kullanabilir: `current page.north`,
`current page.south west`, `current page.north east`, `current page.center` ve
diğerleri.

```ahd
bring Latex as L

watermark: String := L.overlay(r"""
\node[opacity=0.08, rotate=30, scale=8] at (current page.center) {DRAFT};
""")
body: String := r"\thispagestyle{empty}" + "\n" + watermark + L.section("Report")
write(L.document(body))
```

Bir overlay, parçaya ulaşıldığı anda doldurulmakta olan sayfaya çizilir ve o
sayfanın metnini asla kaydırmaz. Süslediği sayfanın başına koyun; birden çok
sayfa için her sayfaya bir tane ekleyin. Hiçbir şey rasterleştirilmez.

### border

`border(inset, thickness, color)`, tek bir dikdörtgen sayfa kenarlığı çizen
sıradan bir overlay'dir: `inset` her sayfa kenarından santimetre cinsinden
mesafedir (varsayılan `1.0`, negatif olamaz), `thickness` punto (point)
cinsinden çizgi kalınlığıdır (varsayılan `1.0`, pozitif) ve `color` isteğe bağlı
bir `#RRGGBB` değeridir. İki çağrı çift kenarlık verir. Daha ayrıntılı her şey —
yuvarlatılmış köşeler, kesik çizgiler, süslemeler — `overlay` içinde TikZ'dir.

### Paketlenmiş kütüphaneler ve pgfornament

`libraries`, büyük/küçük harfe duyarlı olarak tam olarak şu adları kabul eder:

```text
calc  positioning  arrows.meta  shapes.geometric
decorations.pathmorphing  decorations.pathreplacing
patterns  fit  backgrounds  pgfornament
```

`pgfornament`, pgfornament paketini ve onun 196 Vectorian süslemesini yükler;
bunlar `\pgfornament[width=3cm]{63}` ile çizilir, paketin `symmetry` seçeneği
bir süslemeyi her köşeye aynalar. Başka her ad, hiçbir şey derlenmeden
`ValueError` fırlatır.

`document()`, TikZ'i yalnızca gövdesinde veya kapağında bir `tikz`, `overlay`
veya `border` parçası varsa yükler; ardından istenen her kütüphaneyi yukarıdaki
sırayla bir kez yükler. Böyle bir parça içermeyen bir belge, v1.2.0 öncesiyle
bayt bayt aynıdır. Gövdeye elle yazılmış bir `\begin{tikzpicture}` TikZ'i sizin
yerinize yüklemez — `Latex.tikz` kullanın. `Latex.pdf`'e verdiğiniz eksiksiz
bir kaynak `\usepackage{tikz}`'i ve yukarıdaki kütüphaneleri kendisi
yükleyebilir.

### Bir sertifika

```ahd
bring Latex as L
from Latex bring LatexError

frame: String := L.border(inset: 0.8, thickness: 2.4, color: "#1F4E79")
frame += L.border(inset: 1.25, thickness: 0.6, color: "#B08D57")
corner: String := L.overlay(
    source: r"""
\node[anchor=north west] at ([shift={(1.5cm,-1.5cm)}]current page.north west)
    {\pgfornament[width=3cm]{63}};
"""
    libraries: ["pgfornament"]
)
title: String := r"{\Huge\bfseries Certificate of Achievement}\par\vspace{1cm}" + "\n"
title += r"{\LARGE\itshape " + L.escape("Ayşe Yılmaz") + r"}\par" + "\n"
body: String := r"\thispagestyle{empty}" + "\n" + frame + corner
body += r"\vspace*{\fill}" + "\n" + L.center(title) + r"\vspace*{\fill}" + "\n"

attempt {
    L.pdf(L.document(body: body, landscape: true), "certificate.pdf")
} except LatexError as error {
    write(error.message)
}
```

Her köşede süsleme, filigran, TikZ mührü ve imza satırları içeren eksiksiz
sertifika:
[`examples/v0.1/61_tikz_certificate.ahd`](../examples/v0.1/61_tikz_certificate.ahd).

### TikZ hataları ve güvenlik

Yardımcılar, bir parça oluşturulurken kendi girdilerini doğrular: paketlenmemiş
bir kütüphane adı, negatif bir kenarlık `inset`'i, pozitif olmayan bir
`thickness`, geçersiz bir kenarlık rengi ve Beamer ile `landscape`
`ValueError` fırlatır.

TikZ kaynağının kendisi Latex girdisidir ve AhdCode derleyicisi tarafından
denetlenmez. Bir TikZ sözdizimi hatası, elle yüklenen ve paketlenmemiş bir
kütüphane veya eksik bir paket, ilk TeX hatası korunarak `LatexError` ile
derlemeyi başarısız kılar:

```text
compilation failed: error: document.tex:13: Package tikz Error: Cannot parse this coordinate.
compilation failed: error: document.tex:3: Package tikz Error: I did not find the tikz library 'shadows'. ...
compilation failed: error: document.tex:3: ! LaTeX Error: File `tcolorbox.sty' not found.
```

TikZ bir kum havuzu (sandbox) değildir. Her Latex belgesiyle aynı güvenilmeyen
(untrusted) kip motorunda çalışır: kabuk kaçışı kullanılamaz ve hiçbir şey
indirilmez.

TCPDF veya FPDF, Canvas, SVG çizim modülü, tarayıcı tabanlı render, ikinci bir
PDF motoru, TikZ komutlarını taklit eden bir çizim API'si, sertifika modülü,
keyfi paket yükleme veya rasterleştirilmiş süsleme yoktur.

## Profesyonel belgeler (v1.3.0)

Sayfa numaralı üst ve alt bilgiler, sayfada tam konuma yerleştirilen içerik,
bağlantılar, PDF anahat yer imleri, QR kodları, barkodlar ve SVG logolar
sıradan Latex parçalarıdır: onları aşağıdaki yardımcılarla oluşturun ve diğer
parçalar gibi `document()` gövdesine ekleyin. `document()` destekleyici bir
paketi yalnızca bir parça ona ihtiyaç duyduğunda yükler; bu yüzden bu parçaları
içermeyen bir belge v1.3.0 öncesiyle bayt bayt aynıdır.

```text
üst/alt bilgi             -> header, footer, pageNumber, pageCount
sayfada tam konum         -> place
gezinme                   -> link, bookmark
makine tarafından okunan  -> qr, barcode
sayfa ve özellikler       -> document(paper:, pageSize:, margins:, subject:, keywords:, creator:)
vektör logolar            -> .svg yoluyla image ve figure
```

Hepsini içeren eksiksiz bir rapor
[`examples/v0.1/66_latex_professional_report.ahd`](../examples/v0.1/66_latex_professional_report.ahd),
doğrulama QR kodlu bir sertifika ise
[`examples/v0.1/64_verifiable_certificate.ahd`](../examples/v0.1/64_verifiable_certificate.ahd)
dosyasındadır.

### header, footer, pageNumber, pageCount

`header(left, center, right)` ve `footer(left, center, right)`, her sayfanın
üstündeki ve altındaki üç bölgeyi ayarlar. Bölgeler üretilmiş Latex taşır —
kaçışlanmış metin, `pageNumber()`, `pageCount()`, bir `link`, SVG logo gibi bir
`image` veya bir `qr` sembolü — bu yüzden sıradan metni `escape`'ten geçirin.
`pageNumber()` geçerli sayfa numarası, `pageCount()` ise belgenin toplam sayfa
sayısıdır; ikisi gövde metninde de çalışır.

```ahd
logo: String := L.image("logo.svg", {"height": 0.8})
header: String := L.header(logo, L.escape("Quarterly Report"), L.escape("Q3 2026"))
footer: String := L.footer(L.link("ahdcode.org", "https://ahdcode.org"), "", "Page " + L.pageNumber() + " of " + L.pageCount())
source := L.document(body: header + footer + body, paper: "A4")
```

Bir Latex belgesi sayfa numarasını varsayılan olarak alt bilginin ortasında
gösterir. Yalnızca `header` bunu korur; `footer` ise alt bilginin üç bölgesini
de ayarlar, bu yüzden numaranın görünmesi gereken yere `pageNumber()` koyun.
Ayarlar, parçaya ulaşılan sayfadan itibaren geçerli olur: her sayfayı kapsaması
için onları gövdenin başına koyun, sonraki sayfaları değiştirmek için daha
sonra başka bir `header` veya `footer` ekleyin. Üst ve alt bilgiler başlık ve
bölüm sayfalarına da uygulanır ve çizgi çizmez. Article veya Report belgesi
gerektirirler; Beamer ile `document()` `ValueError` fırlatır.

### place

`place(content, x, y, anchor)`, içeriği sayfada tam bir konuma yerleştirir.
`x` ve `y`, sayfanın sol üst köşesinden santimetre cinsinden, 0 ile 1000
arasındadır; `anchor` ise içeriğin o noktaya denk gelen yerini adlandırır:
`"north west"` (varsayılan), `"north"`, `"north east"`, `"west"`, `"center"`,
`"east"`, `"south west"`, `"south"` veya `"south east"`. Yerleştirilen içerik
sayfanın metnini asla kaydırmaz. Parçaya ulaşıldığında doldurulmakta olan
sayfaya çizilir; bu yüzden içeriği ait olduğu her sayfa için bir kez
yerleştirin. TikZ koordinatları ve sayfa çapalarıyla çizmek için `overlay`
kullanın.

```ahd
body += L.place(L.image("logo.svg", {"height": 2.0}), 2.0, 2.0)
body += L.place(L.qr("https://ahdcode.org/verify?id=42", 2.5, "Q"), 19.0, 27.5, "south east")
```

### link ve bookmark

`link(text, url)`, `text`'i tıklanabilir yapar; `text` kaçışlanır. `url`
`https://`, `http://` veya `mailto:` ile başlamalı ve kontrol karakteri
içermemelidir; `javascript:` ve `file:` dahil başka her şema `ValueError`
fırlatır. URL'deki özel karakterler kodlanır; böylece bağlantı tam olarak
verilen URL'yi açar ve belgenin içine taşamaz. Bağlantılar renkli kutu olmadan
çizilir.

`bookmark(title, level)`, PDF anahat görünümüne — görüntüleyicinin kenar
çubuğu — parçaya ulaşılan konumu gösteren bir girdi ekler. Seviye 1 en üst
düzey girdidir; 2 ile 4 arası seviyeler, kendinden küçük seviyedeki en yakın
önceki girdinin altına yerleşir. `chapter`, `section` ve `subsection` anahatta
zaten kendiliğinden görünür; kapak, tablo veya doğrulama kodu gibi başka her şey
için `bookmark` kullanın.

```ahd
body += L.bookmark("Verification code") + L.center(L.qr("https://ahdcode.org/verify?id=42"))
body += "Verify at " + L.link("ahdcode.org/verify", "https://ahdcode.org/verify?id=42") + "."
```

Sayfa boyutu, kenar boşlukları ve PDF özellikleri `document()` parametreleridir;
bkz. [Article, Report ve Beamer için TEK bir `document()`](#article-report-ve-beamer-için-tek-bir-document).

## QR kodları ve barkodlar (v1.3.0)

`qr(value, size, level)`, sessiz bölge dahil `size` santimetre kare (varsayılan
`3.0`) bir QR sembolünü `"L"`, `"M"` (varsayılan), `"Q"` veya `"H"` seviyesinde
çizer. `barcode(kind, value, width, height)`, sessiz bölgeler dahil `width` x
`height` santimetrelik (varsayılan `8.0` x `2.0`) bir `"Code128"`, `"EAN13"`
veya `"UPCA"` barkodu çizer. İkisi de [QR](QR_TR.md) ve [Barcode](BARCODE_TR.md)
modülleriyle aynı kodlayıcılardan üretilen TikZ vektör parçalarıdır — görsel
dosyası ve rasterleştirme yoktur — ve gövdede, `center` ve `minipage` içinde,
`header` ve `footer` içinde ve `place` ile çalışır. `bring QR` veya
`bring Barcode` gerekmez.

```ahd
body += L.center(L.qr("https://ahdcode.org", 3.0, "Q"))
body += L.barcode("EAN13", "590123412345", 6.0, 2.0)
```

Geçersiz bir değer — boş veya çok uzun bir QR değeri, bilinmeyen bir seviye ya
da tür, yanlış bir EAN-13 kontrol basamağı, ASCII olmayan bir Code 128
karakteri — parça oluşturulurken QR ve Barcode modüllerinin verdiği mesajla
aynı mesajı taşıyan bir `ValueError` fırlatır.

## SVG varlıkları (v1.3.0)

`image` ve `figure` bir `.svg` yolu kabul eder. Belge derlenirken SVG program
içinde PGF çizim komutlarına dönüştürülür; böylece PDF'te vektör olarak kalır.
Hiçbir şey rasterleştirilmez; tarayıcı, Inkscape, kabuk komutu veya ağ erişimi
kullanılmaz; dönüşüm yalnızca SVG dosyasının kendisini okur.
`PDFDocument.image` aynı dönüştürücüyü kullanır.

Desteklenenler:

- `svg`, `g`, `path` (yaylar dahil her path komutu), `rect` (yuvarlatılmış
  köşeler dahil), `circle`, `ellipse`, `line`, `polyline`, `polygon`, aynı
  dosyadaki bir öğeye başvuran `use` (`#id`), `defs`, `title`, `desc`,
  `metadata` ve `style` öğeleri;
- sunum öznitelikleri, satır içi `style` öznitelikleri ve seçicileri öğe adı,
  `.class`, `#id` veya `*` olan `<style>` kuralları;
- onaltılık renk, `rgb()`, CSS renk adı, `none` veya `currentColor` olarak
  `fill` ve `stroke`; `fill-rule`; `opacity`, `fill-opacity` ve
  `stroke-opacity`;
- `stroke-width`, `stroke-linecap`, `stroke-linejoin`, `stroke-miterlimit`,
  `stroke-dasharray` ve `stroke-dashoffset`; `display: none` ve `visibility`;
- `matrix`, `translate`, `scale`, `rotate`, `skewX` ve `skewY` ile
  `transform`; `px`, `pt`, `pc`, `mm`, `cm` veya `in` birimli uzunluklarla
  `viewBox`, `preserveAspectRatio`, `width` ve `height`.

Atılmak veya yaklaşık çizilmek yerine desteklenmeyen özelliği belirten bir
mesajla reddedilenler:

- metin (`text`, `tspan`, `textPath`) — metni SVG düzenleyicide dış hatlara
  (path) dönüştürün;
- gradyanlar, desenler, kırpma path'leri, maskeler, filtreler, işaretçiler,
  `symbol`, `switch` ve `#00000080` ya da `rgba(0, 0, 0, 0.5)` gibi kendi
  saydamlığı olan renkler — bunun yerine `fill-opacity` veya `stroke-opacity`
  kullanın;
- gömülü veya bağlantılı görseller, `script`, `foreignObject`, bağlantılar
  (`a`), animasyon, iç içe `svg`, `onload` gibi olay öznitelikleri,
  `url(...)` başvuruları ve dosya dışına her başvuru;
- `@import` gibi CSS at-kuralları, DOCTYPE veya entity bildirimleri ve işleme
  talimatları (processing instructions).

Bir SVG en çok 5 MiB, 100.000 öğe, 64 iç içe seviye ve 2.000.000 path parçası
olabilir. Latex bir SVG'yi belge derlenirken dönüştürür, bu yüzden
desteklenmeyen bir SVG `LatexError` fırlatır; `PDFDocument.image` onu
çağrıldığında dönüştürür ve `PDFError` fırlatır.

## Derleme

`pdf`, bir kaynak String'i derler; `pdfFile`, var olan bir `.tex` dosyasını
derler ve `\includegraphics` gibi belgeye göreli varlıkları (assets) girdi
dosyasının dizinine göre çözer.

`pdf`, isteğe bağlı üçüncü bir `sourceOutput` argümanı alır, `""` (varsayılan)
veya `"tex"`:

```ahd
pdf(source: String, output: String, sourceOutput: String = "") -> Nothing
```

`sourceOutput: ""` — mevcut, değişmemiş sözleşme: yalnızca `output` (bir
`.pdf`) yayınlanır. `sourceOutput: "tex"` ayrıca çağıranın tam `source`
baytlarını içeren kardeş bir `.tex` dosyası yayınlar:

```ahd
attempt {
    L.pdf(document, "cikti.pdf", "tex")
    write("PDF ve TEX başarıyla oluşturuldu!")
}
except LatexError as error {
    write("Dosyalar oluşturulamadı: {error.message}")
}
```

`cikti.pdf` ve `cikti.tex` üretir. Kardeş yol, sondaki `.pdf`'in `.tex` ile
değiştirilmesiyle türetilir (`report.pdf` → `report.tex`,
`folder/report.pdf` → `folder/report.tex`); `sourceOutput` `"tex"` olduğunda
`output` `.pdf` ile bitmelidir, aksi halde derlemeden önce `LatexError`
fırlatılır. `""` veya `"tex"`'in tam, büyük/küçük harfe duyarlı değeri
dışında herhangi bir `sourceOutput` değeri de `LatexError` fırlatır —
keyfi bir geçiş dizesi yoktur.

`.tex` yan dosyası, çağıranın kendi kaynağıdır, olduğu gibi: dönüştürülmüş
bir geçici Tectonic girdisi değil, derleyicinin eklediği hata ayıklama
içeriği değil, geçici bir yol değil ve render motoru meta verisi değil.
Yayınlama sırası: önce PDF'i derle ve doğrula, sonra — yalnızca PDF zaten
atomik olarak yayınlandıktan sonra — `.tex` yan dosyasını yaz. Bir derleme
hatası hiçbir dosyayı yayınlamaz ve mevcut bir hedefe asla dokunmaz.
Dosya sistemi genelinde iki dosyalı bir işlem olmadığından, başarılı bir PDF
yayınından sonraki bir `.tex` yan dosyası yazma hatası, yeni PDF'i yayınlanmış
ve yan dosyayı yazılmamış bırakır; bu, tek bir iki-dosyalı atomik işlem değil,
iki ayrı atomik yeniden adlandırmadır.

`pdfFile` değişmedi: hâlâ tam olarak `(input, output)` alır, çünkü çağıranı
`.tex` dosyasını zaten diskte sahiplenir.

**Sürüm paketleri bu runtime bileşenini içerir. Aşağıdaki hazırlama adımları yalnızca kaynaktan derleyen geliştiriciler içindir.**

Derleme, çevrimdışı Tectonic motoru ve yerel bir kaynak paketi tarafından yapılır. Kaynak koddan yapılan standart `go install` adımı LaTeX runtime dosyalarını kurmaz. LaTeX kullanmak isteyen kullanıcı, bunları `package-latex` aracıyla bir kez ayrıca hazırlar (stage):

```bash
go run ./tooling/latex/cmd/package-latex --output "$(go env GOPATH)"
```

Bu komut, sabitlenmiş (pinned) kaynakları getirmek ve doğrulamak için bir defaya mahsus bir ağ işlemi gerçekleştirir ve bunları Go binary dizininizde `ahdcode` ile yan yana yerleştirir:

```text
libexec/ahdcode/latex/tectonic
libexec/ahdcode/latex/ahdcode-latex.ttb
libexec/ahdcode/latex/THIRD_PARTY_NOTICES.txt
```

Hazırlandıktan (staged) sonra AhdCode, `PATH`'te bulunan bir `tectonic`'i asla çalıştırmaz, hiçbir zaman bir sistem TeX kurulumuna geri dönmez (fall back) ve çalışma zamanında hiçbir şey indirmez. Çevrimdışı motor veya paket eksikse, bu bir `LatexError`'dır.

## Yapısı gereği çevrimdışı

Motor, izole bir çağrı-başına (per-invocation) önbellek (cache) ve yalnızca
yerel-paket (local-bundle-only) politikasıyla çağrılır, bu yüzden
desteklenen bir belge, boş bir önbelleğe ve ağa sahip olmayan taze bir
makinede derlenir. Ayrıca kurulu bir TeX dağıtımı ve çalışma zamanı kaynak
indirmesi yoktur. Bu, Beamer'ı da kapsar: hazırlanan (staged) kaynak paketi
`beamer.cls`'i, `beamerbase*` bileşenlerini, üzerine inşa edildiği PGF/
TikZ çekirdeğini ve `translator`'ı taşır, bu yüzden bir Beamer sunumu tam
olarak Article/Report gibi derlenir — çevrimdışı, sistem TeX olmadan. v1.2.0'dan
beri paket ayrıca TikZ'i, `Latex.tikz`'in kabul ettiği dokuz TikZ kütüphanesini
ve Vectorian süslemeleriyle pgfornament'i, v1.3.0'dan beri de üst ve alt
bilgiler ile sayfa sayısı için `fancyhdr` ve `lastpage`'i taşır; her dosya
kaynak manifestinde sağlama toplamıyla sabitlenmiştir.

## Güvenlik

Motor güvenilmeyen (untrusted) modda çalışır, bu yüzden `\write18` kabuk
kaçışı (shell escape) kullanılamaz ve hiçbir AhdCode kaynak yapısı bunu
etkinleştiremez. Motor, bir kabuk komut dizesi değil bir argüman vektörüyle
başlatılır — bu yüzden boşluk, Unicode, tırnak, `$`, `;`, `&` veya parantez
içeren yollar güvende kalır. Varlık hazırlama, dosyaları asla bir kabuk
üzerinden değil, yol (path) üzerinden kopyalar ve derleme başlamadan önce
eksik, okunamayan veya desteklenmeyen biçimdeki bir varlığı reddeder. Bir SVG
varlığı, dosyanın baytlarından AhdCode'un kendisi tarafından dönüştürülür:
içindeki hiçbir şey çalıştırılmaz ve başka bir dosyayı ya da URL'yi yükleyemez.

Derleme, 30 saniyelik bir zaman aşımıyla (timeout) sınırlıdır. Zaman
aşımında motor süreci sonlandırılır, geçici dosyalar kaldırılır ve bir
`LatexError` fırlatılır.

## Çıktı güvenliği

Kaynak, başarı ve başarısızlıkta kaldırılan benzersiz, güvenli bir geçici
dizinde derlenir. PDF, geçici bir konuma üretilir, varlığı, sıradan-dosya
(regular-file) durumu, sıfır olmayan boyutu ve `%PDF-` imzası kontrol edilir
ve ancak o zaman istenen hedefe taşınır. Başarısız bir derleme bu yüzden
hiçbir zaman zaten geçerli bir hedef PDF'i yok etmez.

## ValueError ve LatexError

Girdi-alanı doğrulaması mevcut Latex API sözleşmesini izler ve `ValueError`
fırlatır: geçersiz `document()` type, margin, color veya theme; Beamer dışında
Default olmayan theme; geçersiz teorem kaydı/referansı; geçersiz table,
minipage veya image-size seçeneği; desteklenmeyen image uzantısı; v1.2.0'dan
beri paketlenmemiş bir TikZ kütüphane adı, geçersiz `border` değerleri ve
Beamer ile `landscape`; v1.3.0'dan beri de geçersiz bir `qr` veya `barcode`
değeri, `place` koordinatı veya çapası, `link` URL'si, `bookmark` seviyesi,
görsel `transform`'u, `paper`, `pageSize`, `margins` veya anahtar sözcük ile
Beamer'da sayfa düzeni, üst veya alt bilgi. Theme doğrulaması bilinçli olarak
farklı bir hata sınıfı eklemez.

`LatexError` yürütme hatalarını kapsar: derleme başarısızlığı, eksik
çevrimdışı motor veya paket, zaman aşımı, motor süreç başarısızlığı,
üretilmemiş PDF ve desteklenmeyen bir SVG dahil stage edilemeyen varlık
dosyası. Motor tanılamaları,
hatalı biçimlendirilmiş bir belgenin terminali doldurmasını önlemek için
sınırlıdır; bu sırada ilk faydalı TeX hatası korunur.

```ahd
bring Latex as L
from Latex bring LatexError

attempt {
    L.pdf(source: source, output: "report.pdf")
} except LatexError as error {
    write(error.message)
}
```

## Desteklenen temel çizgi (baseline)

`article`, `report`, `beamer`, `amsmath`/`amssymb`/`mathtools`,
`graphicx`, `booktabs`, `array`, `geometry`, `xcolor`, `hyperref`,
`fontspec`, `calc`, `positioning`, `arrows.meta`, `shapes.geometric`,
`decorations.pathmorphing`, `decorations.pathreplacing`, `patterns`, `fit` ve
`backgrounds` kütüphaneleriyle PGF ve TikZ, `pgfornament`, `translator`,
Default/Madrid/Warsaw Beamer theme kapanışı, Latin Modern yazı tipleri, Computer Modern
matematik ve heceleme (hyphenation) verisi. Türkçe dahil Unicode metin,
kutudan çıktığı gibi çalışır.

Bu sürümde olmayanlar: BibTeX, bir paket yöneticisi, TikZ komutlarını taklit
eden bir çizim API'si, yukarıdaki listenin dışındaki TikZ kütüphaneleri, keyfi
Beamer theme'leri, Beamer overlay'leri, konuşmacı notları, bir PDF düzenleyici
veya ayrıştırıcı ve Markdown veya HTML dönüşümü.
