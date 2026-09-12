# PDF standart modülü

[English](PDF.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Latex](LATEX_TR.md) · [Word](WORD_TR.md) · [Excel](EXCEL_TR.md) · [QR](QR_TR.md) · [Barcode](BARCODE_TR.md)

PDF, değişmez belgeler oluşturur ve gerçek `.pdf` dosyalarını çevrimdışı
üretir. Açıkça içe aktarın:

```ahd
bring PDF
from PDF bring PDFDocument
from PDF bring PDFError
```

Kanonik modül kimliği `builtin:PDF`'tir; bir kardeş `PDF.ahd` dosyası onu
gölgeleyemez. PDF metni her zaman sıradan metindir: çağıranın verdiği her
String, render motoruna ulaşmadan önce kaçışlanır; bu yüzden `PDF` asla ham
LaTeX/TeX enjeksiyonuna izin vermez. Gerçek LaTeX kaynak denetimi isteniyorsa
doğrudan [`Latex`](LATEX_TR.md) kullanın.

## Yüzey

```text
PDF.new()                              -> PDFDocument
PDF.fromWord(document: Word.Document)  -> PDFDocument
PDF.fromExcel(workbook: Excel.Workbook) -> PDFDocument

PDFDocument.heading(text: String, level: Int) -> PDFDocument
PDFDocument.paragraph(
    text: String,
    align: String = "left",
    bold: Bool = false,
    italic: Bool = false,
    underline: Bool = false
) -> PDFDocument
PDFDocument.table(
    headers: List<String>,
    rows: List<List<String>>,
    align: String = "left"
) -> PDFDocument
PDFDocument.image(
    path: String,
    size: Pair<String, Real> = {},
    transform: Pair<String, Real> = {}
) -> PDFDocument
PDFDocument.pageBreak()                -> PDFDocument
PDFDocument.qr(value: String, size: Real = 3.0, level: String = "M", align: String = "center") -> PDFDocument
PDFDocument.barcode(
    kind: String,
    value: String,
    width: Real = 8.0,
    height: Real = 2.0,
    align: String = "center"
) -> PDFDocument
PDFDocument.link(text: String, url: String, align: String = "left") -> PDFDocument
PDFDocument.bookmark(title: String, level: Int = 1) -> PDFDocument

PDFDocument.layout(
    paper: String,
    landscape: Bool = false,
    pageSize: Pair<String, Real> = {},
    margins: Pair<String, Real> = {}
) -> PDFDocument
PDFDocument.header(left: String, center: String = "", right: String = "") -> PDFDocument
PDFDocument.footer(left: String, center: String = "", right: String = "") -> PDFDocument
PDFDocument.pageNumbers(align: String = "center", total: Bool = false) -> PDFDocument
PDFDocument.metadata(
    title: String,
    author: String = "",
    subject: String = "",
    keywords: List<String> = [],
    creator: String = ""
) -> PDFDocument
PDFDocument.save(path: String)         -> Nothing

PDFDocument
PDFError
```

`PDFDocument` işlemleri yalnızca konumsaldır; bu, `Document`, `String`,
`List`, `Table` ve `Chart`'ın zaten kullandığı aynı kuraldır:

```ahd
doc = doc.paragraph("Important", "center", true, false, false)
```

`qr`, `barcode`, `link`, `bookmark`, `layout`, `header`, `footer`,
`pageNumbers`, `metadata` ve `image` dönüşümü v1.3.0'da eklendi.

## Değişmez inşa

`PDF.new()` boş bir PDFDocument döndürür. Her işlem yeni bir PDFDocument
döndürür ve alıcısını değiştirmeden bırakır; bir List girdisini sonradan
değiştirmek veya `image()`'dan sonra kaynak dosyayı silmek, zaten inşa edilmiş
bir PDFDocument'i değiştiremez:

```ahd
base: PDFDocument := PDF.new()
first: PDFDocument := base.paragraph("One")
second: PDFDocument := base.paragraph("Two")
```

İçerik işlemleri — `heading`, `paragraph`, `table`, `image`, `pageBreak`,
`qr`, `barcode`, `link` ve `bookmark` — eklendikleri sırayla görünür.
`layout`, `header`, `footer`, `pageNumbers` ve `metadata` belge ayarlarıdır:
nerede çağrılırlarsa çağrılsınlar bütün belgeye uygulanırlar ve biri birden çok
kez çağrılırsa son çağrı geçerli olur.

## Sayfa düzeni

`layout` olmadan her sayfa, v1.3.0 öncesinde olduğu gibi 2.54 cm kenar
boşluklu dikey A4'tür. (`Latex.document` kendi varsayılanı olan US Letter'ı
korur; iki modül aynı varsayılan sayfayı paylaşmaz.)

`layout(paper, landscape, pageSize, margins)` bütün belgenin sayfasını
değiştirir:

| `paper` | Boyut (cm) |
|---|---|
| `"A3"` | 29.7 x 42 |
| `"A4"` | 21 x 29.7 |
| `"A5"` | 14.8 x 21 |
| `"Letter"` | 21.59 x 27.94 |
| `"Legal"` | 21.59 x 35.56 |
| `"Custom"` | `pageSize` |

- `paper` büyük/küçük harf duyarlıdır.
- `landscape: true` sayfayı yan çevirir.
- `pageSize` yalnızca `"Custom"` ile kullanılır ve santimetre cinsinden hem
  `"width"` hem `"height"` gerektirir.
- `margins`, santimetre cinsinden `"top"`, `"right"`, `"bottom"` ve `"left"`
  anahtarlarından istediklerini kabul eder; verilmeyen kenar 2.54 cm kalır.
- Her uzunluk 0'dan büyük ve en çok 1000 olmalıdır; kenar boşlukları içerik
  için yer bırakmalıdır.

```ahd
doc = doc.layout("A5", true)
doc = doc.layout("Letter", false, {}, {"top": 3.0, "bottom": 2.0})
doc = doc.layout("Custom", false, {"width": 10.0, "height": 6.0}, {"top": 0.5, "right": 0.5, "bottom": 0.5, "left": 0.5})
```

## Üst bilgi, alt bilgi ve sayfa numaraları

`header(left, center, right)` ve `footer(left, center, right)`, her sayfanın
üç bölgesine düz metin yerleştirir. Metin, tüm PDF metinleri gibi kaçışlanır ve
tek satır olmalıdır.

Bir PDF, sayfa numarasını varsayılan olarak alt bilginin ortasında gösterir.
Yalnızca üst bilgi eklemek bunu korur. Bir alt bilgi onun yerini alır;
`pageNumbers(align, total)` ise numarayı alt bilginin bir bölgesine yerleştirir:
`"3"` veya `total: true` ile `"3 / 12"`. Alt bilginin o bölgesi boş olmalıdır;
aksi halde `save` `PDFError` fırlatır. `footer("")` sayfa numarası dahil alt
bilgiyi kaldırır.

```ahd
doc = doc.header("AhdCode Analytics", "Quarterly Report", "Q3 2026")
doc = doc.footer("Confidential").pageNumbers("right", true)
```

Üst bilgide logo veya başka grafikler, "Sayfa 3 / 12" gibi özel ifadeler ya da
sayfada tam bir konuma yerleştirilmiş içerik için
[`Latex.header`, `Latex.footer` ve `Latex.place`](LATEX_TR.md#profesyonel-belgeler-v130)
kullanın.

## Başlıklar, paragraflar ve metin güvenliği

Başlık seviyeleri `1`'den `6`'ya kadardır; başka bir değer `PDFError`
fırlatır. Paragraf hizalaması tam olarak `"left"`, `"center"`, `"right"` veya
`"justify"`'dır.

```ahd
doc: PDFDocument := PDF.new()
doc = doc.heading("Quarterly report", 1)
doc = doc.paragraph("Prepared offline.")
doc = doc.paragraph("Approved", "right", true, true, true)
doc = doc.pageBreak()
```

Bir PDFDocument'e ulaşan her String — başlık ve paragraf metni, tablo
hücreleri, üst ve alt bilgi metni, bağlantı metni, yer imi başlıkları ve belge
özellikleri — render kaynağı olmadan önce kaçışlanır. `\ { } $ & # % _ ^ ~`
içeren bir String sıradan metin olarak görünür; hiçbiri asla bir render
komutu olarak yorumlanmaz. PDF'in ham içerik veya ham işaretleme kaçış yolu
yoktur.

## Tablo

Her satır, `headers` ile tam olarak aynı sayıda hücreye sahip olmalı ve en
az bir sütun gereklidir. Hizalama `"left"`, `"center"` veya `"right"`'dır,
her sütuna uygulanır. Hücre birleştirme veya kapsama yoktur; düzensiz bir
satır hiçbir şey render edilmeden önce `PDFError` fırlatır — asla
doldurulmaz, kesilmez veya onarılmaz.

```ahd
doc = doc.table(
    ["Region", "Q1", "Q2"]
    [
        ["North", "10", "12"]
        ["South", "8", "11"]
    ]
    "center"
)
```

## Görsel

PDF, PNG, JPEG ve SVG görselleri kabul eder. PNG ve JPEG baytları
`Word.image`'ın yaptığı gibi hemen gömülür; böylece bir PDFDocument, kaynak
dosyanın hayatta kalmasına veya çalışma dizininin aynı kalmasına asla bağımlı
olmaz. Bir SVG, `image()` çağrıldığı anda vektör çizim komutlarına dönüştürülür;
bu yüzden desteklenmeyen bir SVG hemen orada `PDFError` fırlatır; bkz.
[SVG görseller](#svg-görseller).

`size` yalnızca `"width"` ve `"height"` anahtarlarını kabul eder, santimetre
cinsinden:

```ahd
doc = doc.image("chart.png")
doc = doc.image("chart.png", {"width": 12.0})
doc = doc.image("logo.svg", {"width": 4.0, "height": 3.0})
```

Bir boyut doğal en-boy oranını korur; her iki boyut açık kutuyu kullanır;
boyut verilmezse görselin doğal boyutu kullanılır. Boyutlar pozitif olmalıdır.
Eksik dosya, çözülemeyen veri, desteklenmeyen biçim, anahtar veya boyut
`PDFError` fırlatır.

`transform` (v1.3.0) şu anahtarları kabul eder:

| Anahtar | Anlamı | Aralık |
|---|---|---|
| `"rotation"` | saat yönünün tersine derece | -360 ile 360 |
| `"opacity"` | 0 görünmez, 1 tamamen opak | 0 ile 1 |
| `"trimLeft"`, `"trimTop"`, `"trimRight"`, `"trimBottom"` | boyutlandırılmış görselin o kenarından kesilen santimetre | 0 ile 1000 |

Görsel önce boyutlandırılır, sonra kırpılır, döndürülür ve soldurulur. Genişliğin
veya yüksekliğin tamamını kaldıracak bir kırpma `PDFError` fırlatır.

```ahd
doc = doc.image("photo.jpg", {"width": 8.0}, {"trimTop": 1.0, "trimBottom": 1.0})
doc = doc.image("stamp.png", {"width": 4.0}, {"rotation": 12.0, "opacity": 0.6})
```

Görseller kelimeler gibi akar: art arda eklenen iki görsel aynı satırı
paylaşır. Birini diğerinin altına yerleştirmek için aralarına bir paragraf veya
sayfa sonu ekleyin.

### SVG görseller

Bir SVG, PDF içinde vektör çizim komutlarına dönüşür. Asla rasterleştirilmez;
tarayıcı, Inkscape veya başka bir harici dönüştürücü kullanılmaz. Şekiller,
path'ler (yaylar dahil), dönüşümlü gruplar, dolgular, çizgiler, kesik çizgiler
ve opaklık desteklenir. Metin, gradyanlar, desenler, maskeler, kırpma,
filtreler, işaretçiler, gömülü görseller, betikler, animasyon ve harici
referanslar, atılmak veya yaklaşık çizilmek yerine desteklenmeyen özelliği
belirten bir `PDFError` ile reddedilir. Tam liste
[Latex'in SVG varlıkları](LATEX_TR.md#svg-varlıkları-v130) bölümündedir; PDF
ve Latex aynı dönüştürücüyü kullanır.

## QR kodları ve barkodlar

`qr(value, size, level, align)` ve `barcode(kind, value, width, height,
align)` (v1.3.0), [QR](QR_TR.md) ve [Barcode](BARCODE_TR.md) modülleriyle aynı
kodlayıcılardan vektör semboller çizer; bu yüzden bir değer her yerde aynı
modülleri üretir. Boyutlar sessiz bölgeler dahil santimetredir ve `align`
`"left"`, `"center"` veya `"right"`'tır. `kind`, `"Code128"`, `"EAN13"` veya
`"UPCA"`'dır. `bring QR` veya `bring Barcode` gerekmez.

```ahd
doc = doc.qr("https://ahdcode.org/verify?id=42")
doc = doc.qr("https://ahdcode.org/verify?id=42", 2.5, "Q", "right")
doc = doc.barcode("EAN13", "590123412345", 6.0, 2.0, "left")
doc = doc.barcode("Code128", "AHD-2026-0042")
```

Geçersiz bir değer — boş bir QR değeri, yanlış bir EAN-13 kontrol basamağı,
ASCII olmayan bir Code 128 karakteri — işlem çağrıldığında, QR ve Barcode
modüllerinin verdiği mesajla aynı mesajı taşıyan bir `PDFError` fırlatır.

## Bağlantılar, yer imleri ve belge özellikleri

`link(text, url, align)` (v1.3.0) tıklanabilir bir metin satırı ekler. `url`
`https://`, `http://` veya `mailto:` ile başlamalıdır; `javascript:` ve `file:`
dahil başka her şema ve kontrol karakterleri `PDFError` fırlatır. Özel
karakterler güvenle kodlanır; böylece bağlantı tam olarak verilen URL'yi açar.
Bağlantılar renkli kutu olmadan düz metin olarak çizilir.

`bookmark(title, level)`, PDF anahat görünümüne (görüntüleyicinin kenar
çubuğu) eklendiği konuma atlayan bir girdi ekler. Seviye 1 en üst düzey
girdidir; 2 ile 4 arası seviyeler, kendinden küçük seviyedeki en yakın önceki
girdinin altına yerleşir. Anahat tam olarak `bookmark` çağrılarından oluşur:
başlıklar kendiliğinden girdi eklemez.

`metadata(title, author, subject, keywords, creator)`, PDF görüntüleyicilerinin
gösterdiği belge özelliklerini ayarlar. Her anahtar sözcük boş olmamalı ve
virgül içermemelidir; her değer tek satır olmalıdır.

```ahd
doc = doc.bookmark("Summary").heading("Summary", 1)
doc = doc.link("ahdcode.org", "https://ahdcode.org", "center")
doc = doc.metadata("Quarterly Report", "AhdCode Analytics", "Q3 results", ["report", "2026"], "AhdCode")
```

## Kaydetme

`save(path)` bir `.pdf` hedefi kabul eder ve `Nothing` döndürür:

```ahd
doc.save("report.pdf")
```

`save`, PDFDocument'in içeriğini dahili olarak (asla dışa açılmayan) bir
LaTeX gövdesine dönüştürür, bunu AhdCode'un mevcut çevrimdışı Tectonic render
motoru üzerinden derler — `Latex.pdf`'in kullandığı aynı düşük seviye motor
çağrısı, güvenli geçici çalışma alanı ve atomik aynı-dizin yayını — ve
yayınlamadan önce `%PDF-` imzasını doğrular. Başarısız bir derleme mevcut
bir hedefi asla değiştirmez. PDF asla bir `.tex` yan dosyası üretmez; tam
LaTeX kaynağı da isteniyorsa [`Latex.pdf(source, output, "tex")`](LATEX_TR.md#derleme)
kullanın.

Bağlantılar, üst bilgiler, vektör kodlar, SVG görseller ve görsel dönüşümleri
için render paketleri yalnızca belge onları kullandığında eklenir. Yalnızca
v1.2.0'da var olan işlemlerle oluşturulmuş bir belge, v1.2.0'da olduğu gibi
render edilir.

## Word ve Excel dönüşümü

`PDF.fromWord` ve `PDF.fromExcel`, başka bir modülün kendi tipli belgesinin
anlamsal dönüşümleridir — Office/Excel yazdırma taklidi değildir ve
piksel-mükemmel bir DOCX/XLSX-PDF render motoru değildir. İkisi de kaynak
belgeyi okumaz veya yazmaz; ikisi de onu tamamen değişmeden bırakır.

### `PDF.fromWord`

Başlıkları, paragraf metni/hizalaması/kalın/italik/altı çizili durumunu,
tablo içeriğini, görselleri (Word'ün gömülü baytlarından ve EMU
boyutlarından dönüştürülür) ve sayfa sonlarını korur. Bir tablonun birleştirme
geometrisinin PDF karşılığı yoktur ve atılır; tablonun hücre metni her
durumda tam olarak korunur.

```ahd
wordDocument := Word.new()
wordDocument = wordDocument.heading("Report", 1)
wordDocument = wordDocument.paragraph("Hello")

pdfDocument := PDF.fromWord(wordDocument)
pdfDocument.save("report.pdf")
```

### `PDF.fromExcel`

Her Sheet, Workbook sırasına göre bir başlık (Sheet adı) ve ardından
kullanılan aralık üzerinde bir tablo olur. Kullanılan aralığın ilk satırı
tablo başlığı, kalan satırlar ise gövde olur — bu yalnızca sunumsal bir
tercihtir; Excel çalışma kitaplarının resmi bir başlık satırı kavramı yoktur
ve hiçbir durumda hiçbir hücre kaybolmaz. String/Int/Real/Bool hücreleri
deterministik olarak gösterilir, Blank boş kalır ve bir Formula hücresi
formül *kaynak metnini* gösterir — asla uydurulmuş veya önbelleğe alınmış bir
sonuç değil, çünkü AhdCode Excel formüllerini hesaplamaz. Bir birleştirmenin
çapa olmayan hücreleri zaten Excel'in kendi modeli tarafından Blank olmaya
garanti edilir, bu yüzden düz ızgara hiçbir değeri kaybetmez; PDF, çıktı
tablosunda çoklu sütun kapsamı denemez. Sıfır Sheet'li bir Workbook `PDFError`
fırlatır. Kullanılan aralığı 10 sütundan geniş olan bir Sheet de sütunları
sessizce atmak veya en iyi çaba çok sayfalı bir düzen denemek yerine
`PDFError` fırlatır.

İki dönüşüm de sıradan bir PDFDocument döndürür; bu yüzden `layout`,
`header`, `footer`, `pageNumbers`, `metadata` ve diğer işlemler onlara da
uygulanır.

## Render motoru

`PDF`, düşük seviye render motorunu `Latex.pdf` ile paylaşır: aynı
konuşlandırılmış çevrimdışı Tectonic motoru, aynı `--untrusted` çağrısı,
aynı güvenli geçici çalışma alanı ve aynı atomik yayın. Tam sözleşme için
[Latex'in çevrimdışı/güvenlik/çıktı güvenliği bölümlerine](LATEX_TR.md#yapısı-gereği-çevrimdışı)
bakın — PDF için hiçbiri farklı değildir.

## Hatalar

`PDFError`, PDF'e özgü doğrulamayı — başlık seviyeleri, hizalamalar, tablo
biçimi, sayfa düzeni, üst ve alt bilgi metni, URL'ler, yer imi seviyeleri,
belge özellikleri, QR ve barkod değerleri, görseller ve SVG içeriği — ve
render ile kaydetme hatalarını kapsar:

```ahd
attempt {
    doc.save("report.txt")
}
except PDFError as error {
    write(error.message)
}
```

Statik argüman sayısı ve tip hataları derleyici tanılamaları olarak kalır;
çalışma zamanı `PDFError` değerlerine dönüşmezler.

## Bu sürümde yok

PDF okuma/ayrıştırma, düzenleme, formlar, imzalar, şifreleme, birleştirme,
bölme, hücre başına tablo birleştirmeleri, OCR, HTML/URL/tarayıcı render,
JavaScript, ham LaTeX/TeX, üst ve alt bilgilerde grafik veya özel sayfa
numarası ifadesi ve mutlak konumlandırma v1.3.0'ın parçası değildir. Son üçü
için [`Latex`](LATEX_TR.md) kullanın.
