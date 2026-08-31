# Latex standart modülü

[English](LATEX.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Time modülü](TIME_TR.md)

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
pdf(source: String, output: String)      -> Nothing
pdfFile(input: String, output: String)   -> Nothing

escape(text: String)                     -> String

document(
    body: String, title: String = "", author: String = "", date: String = "",
    type: String = "Article", margin: Real = 2.54, color: String = "",
    cover: String = "", theorems: Pair<String, String> = {},
    theme: String = "Default"
)                                         -> String

chapter(title: String)                   -> String
section(title: String)                   -> String
subsection(title: String)                -> String
frame(title: String, body: String)       -> String

equation(source: String, label: String = "") -> String
theorem(type: String, body: String, label: String = "") -> String

table(headers: List<String>, rows: List<List<String>>, mathColumns: List<Int> = []) -> String
image(path: String, size: Pair<String, Real> = {})    -> String
figure(path: String, caption: String, label: String = "", size: Pair<String, Real> = {}) -> String

minipage(body: String, width: Real, alignment: String = "left") -> String
center(body: String)                     -> String
pageBreak()                              -> String
contents()                               -> String

ref(label: String)                       -> String
cite(key: String)                        -> String
bibliography(references: Pair<String, String>) -> String

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
  varsayılanı `2.54`'tür (etkin v0.1.14 yerleşimi); sayfa-başı kenar boşluğu,
  kağıt boyutu veya yönelim kontrolü yoktur. Pozitif olmalıdır.
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
  ve `"Warsaw"` değerlerini kabul eder, varsayılanı `"Default"`'tır ve son
  konumsal parametredir. Madrid ve Warsaw `type: "Beamer"` gerektirir;
  Article veya Report ile seçilmeleri `ValueError` fırlatır. Bilinmeyen tema
  adları da `ValueError` fırlatır ve LaTeX kaynağına hiçbir zaman doğrudan
  geçirilmez. Özel `color` theme'den sonra uygulanır; theme yerleşimini
  korurken yapısal vurgu rengini override eder.

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

`image(path, size)` numaralandırılmamış bir figür parçasıdır;
`figure(path, caption, label, size)` numaralandırılmış, altyazılı ve (bir
etiketle) `ref` üzerinden referans verilebilir. `size`, yalnızca
`"width"`/`"height"` anahtarlarıyla `Pair<String, Real>`'dır, santimetre
cinsindendir: yalnızca genişlik veya yalnızca yükseklik en-boy oranını
korur, ikisi birden açıkça sığdırılır ve boş bir Pair görüntünün doğal
boyutunu kullanır.

```ahd
body += L.image("logo.png", {"width": 6.0})
body += L.figure("result.pdf", "Numerical solution", "fig:solution", {"width": 12.0})
```

Desteklenen biçimler PNG, PDF ve JPEG'dir. Kırpma (crop), kesme (trim),
döndürme (rotation), subfigure'lar veya açığa çıkarılmış `graphicx`/float
yerleştirme seçenekleri yoktur.

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

## Derleme

`pdf`, bir kaynak String'i derler; `pdfFile`, var olan bir `.tex` dosyasını
derler ve `\includegraphics` gibi belgeye göreli varlıkları (assets) girdi
dosyasının dizinine göre çözer.

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
olarak Article/Report gibi derlenir — çevrimdışı, sistem TeX olmadan.

## Güvenlik

Motor güvenilmeyen (untrusted) modda çalışır, bu yüzden `\write18` kabuk
kaçışı (shell escape) kullanılamaz ve hiçbir AhdCode kaynak yapısı bunu
etkinleştiremez. Motor, bir kabuk komut dizesi değil bir argüman vektörüyle
başlatılır — bu yüzden boşluk, Unicode, tırnak, `$`, `;`, `&` veya parantez
içeren yollar güvende kalır. Varlık hazırlama, dosyaları asla bir kabuk
üzerinden değil, yol (path) üzerinden kopyalar ve derleme başlamadan önce
eksik, okunamayan veya desteklenmeyen biçimdeki bir varlığı reddeder.

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
minipage veya image-size seçeneği ve desteklenmeyen image uzantısı. Theme
doğrulaması bilinçli olarak farklı bir hata sınıfı eklemez.

`LatexError` yürütme hatalarını kapsar: derleme başarısızlığı, eksik
çevrimdışı motor veya paket, zaman aşımı, motor süreç başarısızlığı,
üretilmemiş PDF ve stage edilemeyen varlık dosyası. Motor tanılamaları,
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
`fontspec`, Beamer'ın üzerine inşa edildiği PGF/TikZ çekirdeği ve
`translator` paketleri, Default/Madrid/Warsaw Beamer theme kapanışı, Latin Modern yazı tipleri, Computer Modern
matematik ve heceleme (hyphenation) verisi. Türkçe dahil Unicode metin,
kutudan çıktığı gibi çalışır.

Bu sürümde olmayanlar: BibTeX, bir paket yöneticisi, genel bir TikZ çizim
API'si, keyfi Beamer theme'leri/overlay'leri/konuşmacı notları, bir PDF düzenleyici
veya ayrıştırıcı ve Markdown veya HTML dönüşümü.
