# PDF standart modülü

[English](PDF.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Latex](LATEX_TR.md) · [Word](WORD_TR.md) · [Excel](EXCEL_TR.md)

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
PDFDocument.image(path: String, size: Pair<String, Real> = {}) -> PDFDocument
PDFDocument.pageBreak()                -> PDFDocument
PDFDocument.save(path: String)         -> Nothing

PDFDocument
PDFError
```

`PDFDocument` işlemleri yalnızca konumsaldır; bu, `Document`, `String`,
`List`, `Table` ve `Chart`'ın zaten kullandığı aynı kuraldır:

```ahd
doc = doc.paragraph("Important", "center", true, false, false)
```

## Değişmez inşa

`PDF.new()` boş bir PDFDocument döndürür. Her içerik işlemi yeni bir
PDFDocument döndürür ve alıcısını değiştirmeden bırakır; bir List girdisini
sonradan değiştirmek veya `image()`'dan sonra kaynak dosyayı silmek, zaten
inşa edilmiş bir PDFDocument'i değiştiremez.

## Sayfa düzeni

v0.1.20 kasıtlı olarak sabit bir düzen kullanır: A4, dikey, 2.54cm kenar
boşluğu (`Latex.document()`'ın zaten kullandığı varsayılan değerle aynı). Bu
sürümde sayfa boyutu, yönü veya kenar boşluğu yapılandırması yoktur.

## Başlıklar, paragraflar ve metin güvenliği

Başlık seviyeleri `1`'den `6`'ya kadardır; başka bir değer `PDFError`
fırlatır. Paragraf hizalaması tam olarak `"left"`, `"center"`, `"right"` veya
`"justify"`'dır.

Bir PDFDocument'e ulaşan her String — başlık metni, paragraf metni, tablo
hücre metni — render kaynağı olmadan önce kaçışlanır. `\ { } $ & # % _ ^ ~`
içeren bir String sıradan metin olarak görünür; hiçbiri asla bir render
komutu olarak yorumlanmaz. PDF'in ham içerik veya ham işaretleme kaçış yolu
yoktur.

## Tablo

Her satır, `headers` ile tam olarak aynı sayıda hücreye sahip olmalı ve en
az bir sütun gereklidir. Hizalama `"left"`, `"center"` veya `"right"`'dır,
her sütuna uygulanır. Hücre birleştirme veya kapsama yoktur; düzensiz bir
satır hiçbir şey render edilmeden önce `PDFError` fırlatır — asla
doldurulmaz, kesilmez veya onarılmaz.

## Görsel

PDF, PNG ve JPEG baytlarını `Word.image`'ın yaptığı gibi hemen gömer; böylece
bir PDFDocument, kaynak dosyanın hayatta kalmasına veya çalışma dizininin
aynı kalmasına asla bağımlı olmaz. `size` yalnızca `"width"` ve `"height"`
anahtarlarını kabul eder, santimetre cinsinden.

Bir boyut doğal en-boy oranını korur; her iki boyut açık kutuyu kullanır;
boyut verilmezse render motorunun kendi doğal boyutlandırması kullanılır.
Boyutlar pozitif olmalıdır. Eksik dosya, çözülemeyen veri, desteklenmeyen
biçim, anahtar veya boyut `PDFError` fırlatır.

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

## Render motoru

`PDF`, düşük seviye render motorunu `Latex.pdf` ile paylaşır: aynı
konuşlandırılmış çevrimdışı Tectonic motoru, aynı `--untrusted` çağrısı,
aynı güvenli geçici çalışma alanı ve aynı atomik yayın. Tam sözleşme için
[Latex'in çevrimdışı/güvenlik/çıktı güvenliği bölümlerine](LATEX_TR.md#yapısı-gereği-çevrimdışı)
bakın — PDF için hiçbiri farklı değildir.

## Hatalar

`PDFError`, PDF'e özgü doğrulama, görsel, render ve kaydetme hatalarını
kapsar:

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

PDF okuma/ayrıştırma, düzenleme, açıklamalar, formlar, imzalar, şifreleme,
birleştirme, bölme, sayfa düzeni yapılandırması, hücre başına tablo
birleştirmeleri, OCR, HTML/URL/tarayıcı render ve JavaScript v0.1.20'nin
parçası değildir.
