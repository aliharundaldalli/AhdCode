# Excel

[English](EXCEL.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Lists](LISTS_TR.md) · [KeyValue](KEYVALUE_TR.md) · [Data](DATA_TR.md) · [JSON](JSON_TR.md)

İlk kez öğreniyorsanız Workbook–Sheet–Cell döngüsünü, tipli hücreleri,
formülleri, stilleri ve yeniden okuma kontrolünü birlikte gösteren
[Excel atölyesini](PRACTICAL_MODULES_TR.md#4-excel-tipli-hücrelerle-gerçek-xlsx-üretmek)
çalışın; bu sayfayı ayrıntılı API referansı olarak kullanın.

`bring Excel`, güçlü tipli, değiştirilemez ve çevrimdışı bir XLSX katmanı
sağlar. Yerel çalışma zamanı gerçek Excel-uyumlu `.xlsx` ZIP/XML paketleri
oluşturur ve anlamsal olarak okur; Microsoft Excel, LibreOffice, Python,
yardımcı çalıştırılabilir dosya veya ağ gerekmez.

```ahd
bring Excel
from Excel bring Workbook
from Excel bring Sheet

book: Workbook := Excel.new().addSheet("Students")
sheet: Sheet := book.sheet("Students")
sheet = sheet.setCell(1, 1, Excel.fromString("Name"))
sheet = sheet.setCell(1, 2, Excel.fromString("Score"))
sheet = sheet.setCell(2, 1, Excel.fromString("Ali"))
sheet = sheet.setCell(2, 2, Excel.fromInt(91))
book = book.withSheet(sheet)
book.save("students.xlsx")
```

## Türler ve değiştirilemezlik

Açık türler `Workbook`, `Sheet`, `Cell`, `Range`, `CellStyle` ve
`ExcelError`'dır. `Workbook` ve `Sheet` dönüşümleri bağımsız yeni değerler
döndürür. `book.sheet(name)` ile alınan Sheet'in gizli geri bağlantısı yoktur;
düzenlemeden sonra `book.withSheet(sheet)` ile açıkça geri yerleştirin.

`Excel.new()` boştur ve kendiliğinden `Sheet1` oluşturmaz. `addSheet` yeni boş
Sheet'i sona ekler; `withSheet`, aynı tam ada sahip mevcut Sheet'i konumunu
değiştirmeden yeniler. Bilinmeyen veya yinelenen adlar `ExcelError` yükseltir.
Adlar büyük/küçük harfe duyarsız olarak benzersizdir, en çok 31 Unicode
karakter içerir ve `: \ / ? * [ ]` ya da güvensiz XML denetim karakterlerini
içeremez.

```text
Excel.new()                    -> Workbook
Excel.read(path)              -> Workbook
Workbook.addSheet(name)       -> Workbook
Workbook.sheet(name)          -> Sheet
Workbook.withSheet(sheet)     -> Workbook
Workbook.sheets()             -> List<String>
Workbook.sheetCount()         -> Int
Workbook.save(path)           -> Nothing
```

Sıfır Sheet içeren Workbook'u veya `.xlsx` olmayan hedefi kaydetmek
`ExcelError` yükseltir.

## Tipli Cell değerleri

Bir Cell tam olarak `Blank`, `String`, `Int`, `Real`, `Bool` veya `Formula`
türlerinden biridir. `Any`, dinamik hücre değeri ya da skalerden Cell'e örtük
dönüşüm yoktur.

```text
Excel.blank()
Excel.fromString(value)
Excel.fromInt(value)
Excel.fromReal(value)
Excel.fromBool(value)
Excel.formula(expression)

Cell.kind()       Cell.isBlank()
Cell.string()     Cell.int()       Cell.real()
Cell.bool()       Cell.formula()
```

Yanlış tür erişimi `ExcelError` yükseltir. `real()`, Int'i güvenle genişletir;
`int()`, Real'i asla daraltmaz. Sonlu olmayan Real'ler, güvensiz XML metni ve
fazla uzun String'ler dönüştürülmek veya kesilmek yerine reddedilir.

`=` ile başlayan String, düz metin olarak kalır:

```ahd
safeText := Excel.fromString("=SUM(A1:A100)") // String
formula := Excel.formula("=SUM(A1:A100)")    // Formula
```

Formula metni `=` ile başlamalı, sonrasında içerik bulunmalı ve Excel uzunluk
sınırına uymalıdır. AhdCode ifadeyi saklar ve XML için kaçışlar; ayrıştırmaz,
tip denetlemez, hesaplamaz, bağlantı çalıştırmaz veya ağ içeriği almaz. Okuma,
başındaki `=` işaretini döndürür ve önbelleklenmiş sonucu yok sayar.

Kaydedilen çalışma kitapları her Formula Cell'i yeniden hesaplama için
işaretler: üretilen XLSX bir yer tutucu önbellek değeri ve çalışma kitabı
hesaplama meta verisi taşır (`fullCalcOnLoad`, `forceFullCalc`, gerçek bir
`calcId`), böylece Excel, Numbers ve diğer elektronik tablo uygulamaları
dosya açılır açılmaz gerçek sonucu yeniden hesaplayıp gösterir; kullanıcının
F9'a basmasına veya formülü yeniden girmesine gerek kalmaz. Yer tutucu yalnızca
XLSX birlikte çalışabilirlik meta verisidir; AhdCode onu asla hesaplamaz ve
`Cell.formula()` onu asla döndürmez.

## Koordinatlar, Range ve toplu yazma

Excel koordinatları bilinçli olarak 1 tabanlıdır: `(1, 1)`, `A1`'dir. Satırlar
`1..1048576`; sütunlar `1..16384` (`XFD`) aralığındadır. Sınır dışı değerler
`ExcelError` yükseltir.

```text
Sheet.name()                                      -> String
Sheet.cell(row, column)                           -> Cell
Sheet.setCell(row, column, value)                 -> Sheet
Sheet.range(startRow, startColumn, endRow, endColumn) -> Range
Sheet.setRow(row, startColumn, values)            -> Sheet
Sheet.setRange(range, values)                     -> Sheet
Sheet.cells(range)                                -> List<List<Cell>>
Sheet.usedRange()                                 -> Range?
```

Ayarlanmamış koordinatlar açık Blank Cell olarak okunur. `setRow`,
`List<Cell>` alır. `setRange`, Range'in satır sayısını ve her satırda tam sütun
sayısını ister; veriyi doldurmaz, kesmez, düzensiz satırı onarmaz ve kaynak
Sheet'i kısmen değiştirmez. `cells`, Blank koordinatlar dahil tam dikdörtgenin
yeni bir kopyasını döndürür.

`Range`; `startRow`, `startColumn`, `endRow`, `endColumn`, `rowCount`,
`columnCount` ve `address` erişimcilerini sunar. `usedRange()` boş Sheet için
`null` döndürür; aksi halde Blank olmayan, formula içeren, stilli ve merge
edilmiş hücreleri kapsar. Yalnızca satır/sütun boyutları alanı genişletmez.

## Merge ve stiller

`sheet.merge(range)`, sol üst anchor hücreyi korur. Kapsanan diğer tüm Cell
değerleri önceden Blank olmalı ve merge alanları çakışmamalıdır. Blank olmayan
kapsanmış Cell, `ExcelError` yükseltir; değer atılmaz veya taşınmaz. Merge
edilmiş anchor olmayan koordinata yazmak da reddedilir. `merges()`, sıralı yeni
bir `List<Range>` döndürür.

`Excel.style()` değiştirilemez bir yama oluşturur. Her işlem yalnızca bir
özelliği belirtir; böylece yalnızca bold yaması mevcut fill'i korur. Açık
`bold(false)` bold'u kapatır; belirtilmemiş bold mevcut değeri değiştirmez.

```text
bold(Bool)              italic(Bool)          underline(Bool)
fontSize(Real)          textColor("#RRGGBB") fillColor("#RRGGBB")
horizontal(String)      vertical(String)      wrap(Bool)
numberFormat(String)    border(style, color)
```

Yatay değerler `left`, `center`, `right`; dikey değerler `top`, `center`,
`bottom`'dır. Kenarlık stilleri `none`, `thin`, `medium`, `thick`, `dashed`,
`dotted`, `double`'dır. Renkler büyük harfli `#RRGGBB` kullanır. `General`,
`0`, `0.00`, `0%`, `yyyy-mm-dd` gibi sayı biçimleri açık Excel biçim String
değerleridir; Int/Real Cell'i tarih, para birimi veya yüzde türüne dönüştürmez.

`sheet.style(range, patch)`, `sheet.columnWidth(column, width)` ve
`sheet.rowHeight(row, height)` kullanın. Boyutlar desteklenen Excel sınırları
içinde pozitif, sonlu Real olmalıdır.

## XLSX okuma, yazma ve güvenlik

Üretilen paketler belirlenimci Sheet/ilişki/stil kimlikleri, belirlenimci ZIP
üye sırası ve inline String kullanır. Atomik hedef değişiminden önce üretilen
paketin tamamı doğrulanır; başarısız kayıt mevcut dosyayı yok etmez.

`Excel.read`; sıralı Sheet'leri, shared/inline String'leri, Blank/Int/Real/
Bool/Formula Cell'leri, merge'leri, belgelenen stil alt kümesini ve boyutları
destekler. Sayısal sözcüksel yazım Int/Real ayrımını belirler: `91` Int;
`1.0`, `3.14`, `1e3` Real'dir. Stil biçimleri tarih veya başka tür çıkarmaz.
Harici bir uygulama `1.0` değerini `1` olarak yeniden yazarsa bu sözcüksel
niyet artık geri kazanılamaz.

Okuma anlamsaldır, piksel-kusursuz değildir. Desteklenmeyen sunum özellikleri
korunmaz. Güvenle yeniden kurulamayan desteklenmeyen Cell türleri veya ileri
shared/array Formula gösterimleri sessiz değer/formula kaybı yerine
`ExcelError` yükseltir. Girdi; arşiv, üye, açılmış toplam boyut ve sıkıştırma
oranı sınırlarına tabidir. Yinelenen/kaçış yapan ZIP üyeleri, bozuk XML,
DTD'ler, bozuk iç ilişkiler ve harici worksheet/shared-string/style hedefleri
reddedilir. Harici bağlantılar açılmaz ve ağ isteği yapılmaz.

## Bileşim ve kapsam

`List<List<Cell>>` yapısı için `Lists.transpose`, kayıtlar için
`KeyValue.keys/values`, String tablo anlamı için `Data` ve Cell kurucusunu
seçmeden önce açık JSONValue tür denetimi kullanın. Excel bu modülleri
çoğaltmaz ve `Table`, `Pair` veya `JSONValue` için `Any` köprüsü kabul etmez.

v0.1.20 yalnızca XLSX'tir. `.xls`, `.xlsm`, makrolar, grafikler, görseller,
pivot tablolar, zengin metin parçaları, formula hesaplama, tarih çıkarımı,
şifreleme, baskı düzeni veya kullanıcıya açık ZIP API'si içermez. `Excel`'in
kendisinin PDF dışa aktarımı yoktur, ancak `PDF.fromExcel(workbook)`, Excel'i
hiç kullanmadan bir Workbook'u PDF belgesine dönüştürür — bkz.
[PDF](PDF_TR.md).
