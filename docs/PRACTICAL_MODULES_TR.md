# AhdCode Uygulamalı Modül Atölyeleri

[README'ye dön](../README_TR.md) · [Öğrenci Rehberi](STUDENT_GUIDE_TR.md) · [Modüller](MODULES_TR.md)

Bu belge, AhdCode'un en çok birlikte kullanılan modüllerini yalnızca API
listesi olarak değil, küçük iş akışları üzerinden öğretir. Ana öğrenci
rehberindeki temelleri biliyorsanız buradaki atölyeleri sırayla yapabilirsiniz.

Her atölyede dört soru cevaplanır:

1. Bu modül hangi problemi çözer?
2. Veriyi hangi AhdCode türüyle alır ve geri verir?
3. En sık yapılan hata nedir?
4. Sonucu başka bir modüle nasıl taşırım?

Aynı atölye içindeki kısa kod blokları, aksi söylenmedikçe önceki blokta
oluşturulan değişkenlerle devam eder. Tek dosyada denemek için o atölyedeki
blokları yukarıdan aşağıya aynı `.ahd` dosyasında birleştirin.

## İçindekiler

- [Öğrenme haritası](#öğrenme-haritası)
- [Hazır çalışan örnekler](#hazır-çalışan-örnekler)
- [1. CSV: metin tablosunu güvenle taşımak](#1-csv-metin-tablosunu-güvenle-taşımak)
- [2. Data: String tablosunu şekillendirmek](#2-data-string-tablosunu-şekillendirmek)
- [3. Plot: veriyi okunabilir bir grafiğe dönüştürmek](#3-plot-veriyi-okunabilir-bir-grafiğe-dönüştürmek)
- [4. Excel: tipli hücrelerle gerçek XLSX üretmek](#4-excel-tipli-hücrelerle-gerçek-xlsx-üretmek)
- [5. Word: tablo ve görselli DOCX raporu](#5-word-tablo-ve-görselli-docx-raporu)
- [6. Latex: akademik PDF ve sunum üretmek](#6-latex-akademik-pdf-ve-sunum-üretmek)
- [7. HTTP ve HTTPS: istek, yanıt ve hata sınırı](#7-http-ve-https-istek-yanıt-ve-hata-sınırı)
- [8. HTML: güvenli sayfa kurmak ve belge ayrıştırmak](#8-html-güvenli-sayfa-kurmak-ve-belge-ayrıştırmak)
- [9. Bitirme projesi: veriden iki rapor üretmek](#9-bitirme-projesi-veriden-iki-rapor-üretmek)
- [Hangi belgeye ne zaman bakmalıyım?](#hangi-belgeye-ne-zaman-bakmalıyım)

## Öğrenme haritası

```text
CSV metni ──> CSV kayıtları ──> Data Table ──> sayısal List
                                  │                │
                                  │                ├──> Statistics / Plot
                                  │                │
                                  ├──> CSV çıktısı ├──> Word raporu
                                  │                │
                                  └───────────────> Excel çalışma kitabı

HTTPS adresi ──> HTTP ClientResponse.body() ──> HTML.parse ──> seçilmiş veriler

metin + tablo + görsel ──> Word (.docx) veya Latex (.pdf)
```

Bu oklar otomatik dönüşüm anlamına gelmez. AhdCode modül sınırlarını açık
tutar. Örneğin CSV hücresi `"91"` bir `String`'dir; Statistics'e vermeden önce
`int(...)` veya `real(...)` ile dönüştürmeniz gerekir.

## Hazır çalışan örnekler

Atölyeyi okurken aşağıdaki tam programları da çalıştırabilirsiniz:

- [CSV](../examples/v0.1/22_csv.ahd) ve [Data temelleri](../examples/v0.1/23_data.ahd)
- [Data + Plot](../examples/v0.1/29_data_plot.ahd)
- [Data → Statistics → Plot → Latex](../examples/v0.1/36_full_workflow.ahd)
- [Plot görselli Word](../examples/v0.1/39_word_plot.ahd)
- [JSON → Data → Statistics → Word](../examples/v0.1/45_structured_data_report.ahd)
- [Excel temel](../examples/v0.1/51_excel_basic.ahd),
  [stil](../examples/v0.1/52_excel_styles.ahd) ve
  [okuma/yazma turu](../examples/v0.1/54_excel_roundtrip.ahd)
- [HTTPS istemci örnekleri](../examples/v0.6/README_TR.md)
- [HTML ayrıştırma ve kazıma örnekleri](../examples/v0.7/README_TR.md)

## 1. CSV: metin tablosunu güvenle taşımak

CSV'nin görevi hesap yapmak değil, satır ve sütunlardan oluşan metni doğru
okuyup yazmaktır. Virgül içeren bir ad, tırnak içeren bir açıklama veya hücre
içindeki satır sonu elle `split(",")` ile güvenli ayrıştırılamaz. `CSV` bu
kuralları sizin için uygular.

### 1.1 Satır modeli ve kayıt modeli

Ham satır modelinde başlık da sıradan bir satırdır:

```ahd
bring CSV

source: String := "ad,sehir,puan\nAli,Adana,91\nAyse,Ankara,78\n"
rows: List<List<String>> := CSV.parse(source)

write(rows[0])     // ["ad", "sehir", "puan"]
write(rows[1][0])  // Ali
```

Başlıklarla çalışacaksanız kayıt modeli daha okunaklıdır:

```ahd
records: List<Pair<String, String>> := CSV.parseRecords(source)

write(records[0]["ad"])
write(records[0]["puan"])
```

Seçim kuralı basittir:

- Dosyanın ilk satırı özel bir başlık değilse `parse` / `read` kullanın.
- İlk satır sütun adlarıysa `parseRecords` / `readRecords` kullanın.
- Filtreleme, sıralama ve gruplama yapacaksanız `Data` katmanına geçin.

### 1.2 Tırnaklama neden önemlidir?

Şu CSV'de ikinci hücrenin içindeki virgül sütun ayırıcı değildir:

```ahd
bring CSV

source: String := "ad,not\nAli,\"hizli, dikkatli\"\n"
rows: List<List<String>> := CSV.parse(source)
write(rows[1][1])
```

Çıktı `hizli, dikkatli` olur. Bir hücre içindeki çift tırnak `""` ile
kaçırılır. `CSV.stringify(...)` ve `CSV.stringifyRecords(...)` gerektiğinde
bu tırnakları kendisi ekler; CSV metnini elle birleştirmeyin.

### 1.3 Sayıya açık dönüşüm

CSV, `"91"` değerinin puan mı, öğrenci numarası mı, yoksa metin mi olduğunu
bilemez. Dönüşümü verinin anlamını bilen kod yapar:

```ahd
bring CSV

records: List<Pair<String, String>> := CSV.parseRecords(
    "ad,puan\nAli,91\nAyse,78\n"
)

total: Int := 0
for record in records {
    total = total + int(record["puan"])
}
write(total)
```

Gerçek dosyada geçersiz bir değer bulunabilir. Dönüşüm hatasını veri hatası
olarak ele alın:

```ahd
attempt {
    score: Local Int := int(record["puan"])
    write(score)
}
except DomainError as error {
    badValue: Local String := record["puan"]
    write("gecersiz puan: {badValue}")
}
```

### 1.4 Ayraç ve dosya işlemleri

Noktalı virgüllü bir dosyayı aynı modülle okuyabilirsiniz:

```ahd
bring CSV

rows: List<List<String>> := CSV.read("ogrenciler.csv", ";")
CSV.write("kopya.csv", rows, ";")
```

Ayraç tek bir Unicode karakter olmalıdır. `","`, `";"` ve `"\t"` geçerli
örneklerdir; boş String veya iki karakterli `"||"` geçersizdir.

### 1.5 Yapısal hataları yakalamak

```ahd
bring CSV
from CSV bring CSVError

attempt {
    CSV.parseRecords("ad,puan\nAli\n")
}
except CSVError as error {
    write("CSV yapisi bozuk: {error.message}")
}
```

`CSVError` bozuk tırnaklama, geçersiz ayraç veya başlık genişliğiyle
uyuşmayan kayıt gibi CSV yapısı sorunlarını anlatır. Dosyanın bulunamaması ise
CSV yapısı değil dosya sistemi sorunudur ve `FileError` / `IOError` anlamını
korur.

### Atölye görevi

`ad;vize;final` başlıklı üç öğrencili bir metin oluşturun. `parseRecords` ile
okuyun, her öğrenci için `(vize + final) / 2` hesaplayın ve sonucu yazdırın.
Bir öğrencinin puanına `yok` yazarak dönüşüm hatasını yakaladığınızı doğrulayın.

Tam API ve sınır koşulları için [CSV referansına](CSV_TR.md) bakın.

## 2. Data: String tablosunu şekillendirmek

`Data`, CSV'nin üzerindeki tablo katmanıdır. Sütun adlarını bilir; satır
filtreleyebilir, sıralayabilir, yeni sütun türetebilir ve gruplandırabilir.
Hücreler yine `String` kalır.

### 2.1 Table oluşturmak ve incelemek

```ahd
bring Data
from Data bring Table

students: Table := Data.fromCSV(
    "ad,bolum,puan\nAli,Matematik,91\nAyse,Fizik,78\nDeniz,Matematik,85\n"
)

write(students.columns())
write(students.rowCount())
write(students.columnCount())
write(students.row(0))
write(students.column("puan"))
```

`row(-1)` son satırı verir. Olmayan satır `IndexError`, olmayan sütun
`DataError` üretir. `columns`, `rows`, `row` ve `column` sonuçları birer anlık
görüntüdür; onları değiştirmek tabloyu değiştirmez.

### 2.2 Filtrele, sırala, seç

```ahd
passed: Table := students.filter(
    lambda (row: Pair<String, String>) -> int(row["puan"]) >= 80
)

ranked: Table := passed.sort(
    lambda (row: Pair<String, String>) -> -int(row["puan"])
)

summary: Table := ranked.select(["ad", "puan"])
write(summary.toCSV())
write(students.rowCount())
```

Son satır hâlâ `3` yazar. `filter`, `sort` ve `select` yeni tablolar üretir;
kaynak tabloyu değiştirmez. `sort("puan")` yazsaydınız sözlüksel String
sıralaması yapılırdı: `"100"`, `"20"`den önce gelebilir. Sayısal sıralama
için yukarıdaki gibi anahtar fonksiyonunda açık dönüşüm yapın.

### 2.3 Temizle ve sütun türet

`transform`, var olan tek bir sütunun String değerini değiştirir. `derive`,
tam satırdan yeni bir sütun oluşturur:

```ahd
clean: Table := students.transform(
    "ad"
    lambda (value: String) -> value.trim().capitalize()
)

labelled: Table := clean.derive(
    "durum"
    lambda (row: Pair<String, String>) -> {
        if int(row["puan"]) >= 80 {
            return "gecti"
        }
        return "destek gerekli"
    }
)
```

Fonksiyon `String` döndürmelidir. `return 80` tablo hücresine sessizce
çevrilmez; gerekiyorsa `str(80)` yazın.

### 2.4 Gruplama ve sayım

```ahd
groups: Pair<String, Table> := students.groupBy("bolum")

for department in groups {
    group: Local Table := groups[department]
    write("{department}: {str(group.rowCount())}")
}

write(students.valueCounts("bolum"))
write(students.pivotCount("bolum", "puan").toCSV())
```

`groupBy` her anahtar için sıradan bir `Table` verir. Genel bir otomatik
toplama dili yoktur; ihtiyaç duyduğunuz sayıları grubun sütununu açıkça
dönüştürerek hesaplayın.

### 2.5 Statistics'e geçmek

```ahd
bring Statistics

scores: List<Real> := students.column("puan").map(
    lambda (value: String) -> real(value)
)

write(Statistics.mean(scores))
write(Statistics.median(scores))
write(Statistics.stdDev(scores))
```

Bu sınır önemlidir: `Data` tablo şeklini, `Statistics` ise sayıları bilir.

### 2.6 Kontrol noktaları

Bir veri akışını dışarı yazmadan önce en az şu kontrolleri yapın:

- Beklenen sütun adları var mı?
- Satır sayısı beklediğiniz aralıkta mı?
- Sayısal sütunların her değeri dönüştürülebiliyor mu?
- Boş String kabul ediliyor mu, yoksa hata mı olmalı?
- Sıra önemliyse sayısal mı, sözlüksel mi sıraladınız?

### Atölye görevi

Öğrencileri bölüme göre gruplayın. Her grup için puanları `List<Real>`'e
dönüştürüp ortalamayı hesaplayın. Ardından yalnızca puanı 80 ve üzeri olanları
azalan puan sırasıyla `gecenler.csv` dosyasına yazın.

Tam dönüşüm ve hata sözleşmeleri için [Data referansına](DATA_TR.md) bakın.

## 3. Plot: veriyi okunabilir bir grafiğe dönüştürmek

`Plot`, `List<Int>`, `List<Real>` veya Numeric Vector değerlerinden PNG, SVG
ve PDF grafik üretir. Grafik veri temizlemez ve `"91"` metnini sayıya
çevirmez; bu iş Data sınırında açıkça yapılır.

### 3.1 Data'dan Plot'a geçmek

```ahd
bring Data
from Data bring Table
bring Plot
from Plot bring Chart

students: Table := Data.fromCSV(
    "ad,puan\nAli,91\nAyse,78\nDeniz,85\n"
)

names: List<String> := students.column("ad")
scores: List<Real> := students.column("puan").map(
    lambda (value: String) -> real(value)
)

chart: Chart := Plot.bar(names, scores)
chart = chart.title("Sinav Puanlari")
chart = chart.xLabel("Ogrenci")
chart = chart.yLabel("Puan")
chart.save("puanlar.png")
```

`title`, `xLabel` ve `yLabel` yeni `Chart` döndürür; yeniden atama gerekir.
`save` dosyayı yazar ve `Nothing` döndürür.

### 3.2 Doğru grafik türünü seçmek

| Soru | Uygun başlangıç |
|---|---|
| Kategoriler arasında değer farkı ne? | `Plot.bar` |
| Değer zamanla veya sırayla nasıl değişiyor? | `Plot.line` |
| İki sayısal değişken birlikte nasıl hareket ediyor? | `Plot.scatter` |
| Tek bir sayısal dağılımın şekli ne? | `Plot.histogram` |
| Grupların dağılımları nasıl karşılaştırılıyor? | `Plot.box` |

Çizgi ve scatter grafiğinde x/y uzunlukları eşit olmalıdır. Boş veri, geçersiz
bin sayısı veya uyuşmayan uzunluklar `PlotError` üretir.

### 3.3 Statistics ile özet, Plot ile şekil

```ahd
bring Statistics

average: Real := Statistics.mean(scores)
spread: Real := Statistics.stdDev(scores)

histogram: Chart := Plot.histogram(scores, 5)
histogram = histogram.title(
    "Ortalama {str(average)}, std sapma {str(spread)}"
)
histogram.save("puan-dagilimi.svg")
```

Ortalama tek bir özet sayıdır; histogram aynı sayıların nasıl dağıldığını
gösterir. Biri diğerinin yerine geçmez.

### 3.4 Aynı grafiği Word ve Latex'te kullanmak

Grafiği bir kez PNG olarak kaydedip iki belgeye de ekleyebilirsiniz:

```ahd
bring Word
from Word bring Document
bring Latex as L

word: Document := Word.new()
word = word.heading("Puan Dagilimi", 1)
word = word.image("puanlar.png", {"width": 14.0})
word.save("puan-raporu.docx")

latexBody: String := L.section("Puan Dagilimi")
latexBody += L.figure(
    "puanlar.png"
    "Ogrencilerin sinav puanlari"
    "fig:scores"
    {"width": 12.0}
)
latexBody += "Sekil " + L.ref("fig:scores") + " sonuclari gosterir.\n"
```

Word ve Latex grafiği yeniden çizmez; kaydedilmiş görsel dosyasını belgeye
gömer. Bu nedenle önce `chart.save(...)` başarılı olmalıdır.

Excel modülünün bu sürümü XLSX içine grafik veya görsel gömmez. Aynı çalışmada
tipli veriyi Excel'e, görsel anlatımı Plot üzerinden Word/Latex'e yazın.

### 3.5 Grafik kalite kontrolü

- Eksen etiketlerinde ölçü birimini belirtin.
- Kategori sırasının ne anlama geldiğini kontrol edin.
- Çok fazla kategori varsa çubuk grafiği okunamaz hâle getirmeyin.
- Histogram bin sayısını değiştirip sonucun yanıltıcı olup olmadığına bakın.
- Grafiği kaydettikten sonra dosyayı gerçekten açıp başlık ve etiketleri
  görsel olarak kontrol edin.
- Yalnız grafik vermeyin; raporda veri kaynağını ve hesaplama kuralını yazın.

### Atölye görevi

Aynı puan listesinden bir çubuk grafik ve histogram üretin. İkisini PNG olarak
kaydedin; ortalamayı Statistics ile hesaplayıp başlığa ekleyin. Çubuk grafiği
hem Word hem Latex raporuna gömün.

Grafik türleri, stiller ve çoklu panel düzenleri için
[Plot referansına](PLOT_TR.md) bakın.

## 4. Excel: tipli hücrelerle gerçek XLSX üretmek

Excel'de üç nesneyi birbirinden ayırın:

```text
Workbook  dosyanın bütünü
Sheet     çalışma kitabındaki bir sayfa
Cell      Blank, String, Int, Real, Bool veya Formula değer
```

Koordinatlar 1 tabanlıdır: `(1, 1)` A1, `(2, 3)` C2'dir.

### 4.1 Değiştirilemezlik döngüsü

Bir sayfayı düzenleyip kitaba geri koyma kalıbı şöyledir:

```ahd
bring Excel
from Excel bring Workbook
from Excel bring Sheet

book: Workbook := Excel.new().addSheet("Puanlar")
sheet: Sheet := book.sheet("Puanlar")

sheet = sheet.setRow(1, 1, [
    Excel.fromString("Ad")
    Excel.fromString("Vize")
    Excel.fromString("Final")
    Excel.fromString("Ortalama")
])

sheet = sheet.setRow(2, 1, [
    Excel.fromString("Ali")
    Excel.fromInt(80)
    Excel.fromInt(92)
    Excel.formula("=AVERAGE(B2:C2)")
])

book = book.withSheet(sheet)
book.save("puanlar.xlsx")
```

Burada iki geri atama vardır: `setRow` yeni `Sheet`, `withSheet` yeni
`Workbook` döndürür. Bunlardan birini unutursanız değişiklik kaydedilecek
kitaba ulaşmaz.

### 4.2 Hücre türünü bilinçli seçmek

```ahd
textCell := Excel.fromString("91")
numberCell := Excel.fromInt(91)
decimalCell := Excel.fromReal(91.5)
flagCell := Excel.fromBool(true)
emptyCell := Excel.blank()
formulaCell := Excel.formula("=SUM(B2:B20)")
```

`Excel.fromString("=SUM(A1:A3)")` güvenli düz metindir.
`Excel.formula("=SUM(A1:A3)")` formüldür. AhdCode formülü saklar; sonucu
hesaplayan Excel, Numbers veya başka bir çalışma kitabı uygulamasıdır.

### 4.3 Başlık stili ve sayı biçimi

```ahd
from Excel bring CellStyle

header: CellStyle := Excel.style()
header = header.bold(true)
header = header.fillColor("#1F4E79").textColor("#FFFFFF")
header = header.horizontal("center").border("thin", "#000000")

sheet = sheet.style(sheet.range(1, 1, 1, 4), header)
sheet = sheet.style(
    sheet.range(2, 4, 20, 4)
    Excel.style().numberFormat("0.00")
)
sheet = sheet.columnWidth(1, 24.0)
sheet = sheet.columnWidth(2, 12.0)
```

Stil hücrenin AhdCode türünü değiştirmez. `yyyy-mm-dd` sayı biçimi vermek,
bir String'i tarih türüne çevirmek değildir.

### 4.4 Birleştirme ve aralık

```ahd
sheet = sheet.setCell(1, 1, Excel.fromString("Sinav Sonuclari"))
sheet = sheet.merge(sheet.range(1, 1, 1, 4))
```

Birleştirmede sol üst hücre korunur; kapsanan diğer hücreler önceden boş
olmalıdır. Çakışan birleştirmeler ve birleşmiş alanın sol üstü dışındaki bir
hücreye yazma `ExcelError` üretir.

### 4.5 Kaydedip geri doğrulamak

```ahd
bring Excel
from Excel bring Workbook
from Excel bring Sheet
from Excel bring Cell
from Excel bring Range

loaded: Workbook := Excel.read("puanlar.xlsx")
page: Sheet := loaded.sheet("Puanlar")
cell: Cell := page.cell(2, 2)

write(cell.kind())
write(cell.int())
used: Range? := page.usedRange()
if used != null {
    write(used.address())
}
```

`Cell.kind()` sonucuna uygun okuyucuyu kullanın. String hücrede `int()` veya
Int hücrede `string()` çağırmak `ExcelError` üretir. `usedRange()` boş bir
sayfada `null` olabileceğinden genel kodda null kontrolü yapın.

### Atölye görevi

Üç öğrencinin vize ve final notlarını yazın. Ortalama sütununu formül yapın,
başlığı renklendirin, sayı hücrelerine `0.00` biçimi uygulayın ve dosyayı
yeniden okuyarak ilk öğrencinin adını ve vize notunu doğrulayın.

Range, stiller, boyutlar ve XLSX okuma sınırları için
[Excel referansına](EXCEL_TR.md) bakın.

## 5. Word: tablo ve görselli DOCX raporu

`Word`, bir Word arayüzünü uzaktan kontrol etmez. AhdCode içindeki
`Document` değerinden gerçek `.docx` paketi üretir. Office kurulumu gerekmez.

### 5.1 Belgeyi adım adım kurmak

```ahd
bring Word
from Word bring Document

document: Document := Word.new()
document = document.heading("Laboratuvar Raporu", 1)
document = document.paragraph("Hazirlayan: Ayse Yilmaz")
document = document.heading("Sonuclar", 2)
document = document.table(
    ["Ornek", "pH", "Durum"]
    [["A", "7.1", "Normal"], ["B", "6.8", "Kontrol"]]
)
document = document.paragraph("Rapor sonu", "right", true)
document.save("laboratuvar.docx")
```

Her içerik metodu yeni `Document` döndürür. `save` ise `Nothing` döndürür;
`document = document.save(...)` yazılmaz.

### 5.2 Paragraf parametrelerinin sırası

`paragraph` parametreleri konumsaldır:

```text
paragraph(text, align, bold, italic, underline)
```

Örneğin ortalı, kalın ve italik metin:

```ahd
document = document.paragraph(
    "Onaylandi"
    "center"
    true
    true
    false
)
```

Hizalama `left`, `center`, `right` veya `justify`; başlık seviyesi `1` ile
`6` arasındadır.

### 5.3 Plot görselini gömmek

```ahd
bring Plot
from Plot bring Chart

chart: Chart := Plot.bar(
    ["Matematik", "Fizik", "Tarih"]
    [88.0, 74.5, 91.0]
)
chart = chart.title("Ders Ortalamalari")
chart.save("ortalamalar.png")

document = document.heading("Grafik", 2)
document = document.image("ortalamalar.png", {"width": 14.0})
```

Word PNG ve JPEG kabul eder. Tek boyut verirseniz en-boy oranı korunur;
`width` ve `height` santimetredir.

### 5.4 Data tablosunu Word'e taşımak

```ahd
bring Data
from Data bring Table

table: Table := Data.fromCSV("ad,puan\nAli,91\nAyse,78\n")
wordRows: List<List<String>> := table.rows().map(
    lambda (row: Pair<String, String>) -> [row["ad"], row["puan"]]
)

document = document.table(table.columns(), wordRows)
```

Word tablo hücreleri String olduğu için bu sınır Data ile doğal biçimde
uyumludur. Sütun sırasını açık yazmak, Pair ekleme sırasına yanlışlıkla
bağımlı kalmanızı önler.

### 5.5 Mevcut belgeyi okumak

```ahd
loaded: Document := Word.read("laboratuvar.docx")
write(loaded.headings())
write(loaded.paragraphs())
write(loaded.tables())
write(loaded.text())
```

Okuma anlamsaldır: başlık, paragraf ve tablo metnini geri alır. Özel stiller,
yorumlar, üstbilgi/altbilgi ve piksel düzeyindeki tüm Word düzeni korunmaz.
Bir belgeyi okuyup tekrar kaydetmeyi “Word dosyasını hiç değiştirmeden açıp
kapatmak” olarak düşünmeyin.

### Atölye görevi

Bir `Data` tablosundan geçen öğrencileri seçin. Ad ve puan sütunlarını Word
tablosuna ekleyin, Statistics ortalamasını paragraf olarak yazın ve Plot
grafiğini 12 cm genişlikle belgeye gömün.

Birleştirilmiş hücreler, görsel boyutları ve okuma güvenliği için
[Word referansına](WORD_TR.md) bakın.

## 6. Latex: akademik PDF ve sunum üretmek

`Latex` modülü yapı parçalarını String olarak üretir. Parçaları bir gövdede
birleştirir, `Latex.document` ile tam kaynak hâline getirir ve `Latex.pdf` ile
çevrimdışı derlersiniz.

### 6.1 Güvenli metin ile ham matematiği ayırmak

```ahd
bring Latex as L

body: String := ""
body += L.section("Deney Sonuclari")
body += L.escape("Basari %92, maliyet $5 ve A&B birlikte calisti.")
body += L.equation(r"\bar{x} = \frac{1}{n}\sum_{i=1}^{n} x_i", "eq:mean")
body += "Denklem " + L.ref("eq:mean") + " ortalamayi gosterir.\n"
```

`escape` sıradan metin içindir. `equation` ham LaTeX matematiği içindir.
Kullanıcıdan gelen metni ham gövdeye veya denkleme doğrudan eklemeyin.

### 6.2 Article, Report ve Beamer

```ahd
article: String := L.document(
    body: body
    title: "Kisa Makale"
    author: "Ayse Yilmaz"
    type: "Article"
)
```

- `Article`: kısa ödev, makale ve notlar.
- `Report`: bölüm (`chapter`) içeren uzun raporlar.
- `Beamer`: `frame(...)` parçalarından oluşan sunum.

Report örneği:

```ahd
reportBody: String := L.chapter("Bulgular")
reportBody += L.section("Ozet")
reportBody += L.escape("Uc deney tamamlandi.")

report: String := L.document(
    body: reportBody
    title: "Donem Raporu"
    type: "Report"
    margin: 2.5
    color: "#1F4E79"
)
```

Beamer örneği:

```ahd
slides: String := ""
slides += L.frame("Problem", L.escape("Neyi olcuyoruz?"))
slides += L.frame("Sonuc", L.equation(r"E = mc^2"))

deck: String := L.document(
    body: slides
    title: "Fizik Sunumu"
    type: "Beamer"
    theme: "Madrid"
)
```

### 6.3 Tablo, görsel ve kaynakça

Önce Plot ile görsel dosyasını üretin, sonra Latex figürüne ekleyin:

```ahd
bring Plot
from Plot bring Chart

scores: List<Real> := [91.0, 78.0, 85.0]
chart: Chart := Plot.bar(["Ali", "Ayse", "Deniz"], scores)
chart = chart.title("Sinav Puanlari").yLabel("Puan")
chart.save("ortalamalar.png")

body += L.figure(
    "ortalamalar.png"
    "Ogrencilerin sinav puanlari"
    "fig:averages"
    {"width": 12.0}
)
body += "Sekil " + L.ref("fig:averages") + " puanlari karsilastirir.\n"
```

Ardından aynı rapora tablo ve kaynakça ekleyebilirsiniz:

```ahd
body += L.table(
    ["Ogrenci", "Puan"]
    [["Ali", "91"], ["Ayse", "78"]]
)

body += "Bkz. " + L.ref("fig:averages") + ".\n"
body += "Yontem " + L.cite("Kaynak2026") + " ile uyumludur.\n"
body += L.bibliography({
    "Kaynak2026": "A. Yazar, Veri Analizi, 2026."
})
```

Tablo başlıkları ve normal hücreler kaçışlanır. Matematik içermesi gereken
sütunlar için `mathColumns` parametresini referanstan inceleyin.

### 6.4 Derlemek ve kaynak çıktısını saklamak

```ahd
from Latex bring LatexError

source: String := L.document(body: body, title: "Analiz", type: "Report")

attempt {
    L.pdf(source, "analiz.pdf", "analiz.tex")
    write("analiz.pdf hazir")
}
except LatexError as error {
    write("derleme basarisiz: {error.message}")
}
```

Üçüncü argüman `.tex` kaynağını da saklar; derleme hatasını incelemek ve
öğrenmek için faydalıdır. Render motorunun bir kez hazırlanmış olması gerekir;
kurulum adımı öğrenci rehberindedir.

### Atölye görevi

Bir başlık, iki bölüm, numaralı denklem, tablo, Plot görseli ve tek kaynakça
girdisi içeren Report üretin. Hem `.pdf` hem `.tex` çıktısını saklayın.

Tüm belge parametreleri, teoremler ve yerleşim yardımcıları için
[Latex referansına](LATEX_TR.md) bakın.

## 7. HTTP ve HTTPS: istek, yanıt ve hata sınırı

`HTTP` iki farklı rolde kullanılabilir:

```text
Server                 sizin programınız isteği kabul eder
Client / ClientRequest sizin programınız başka bir hizmete istek gönderir
```

Bu bölüm giden HTTP(S) isteğine odaklanır. Yerel sunucu için öğrenci
rehberindeki “Küçük bir web sayfası” bölümünü de çalışın.

### 7.1 URL'yi parçalarına ayırmak

```text
https://api.example.com:443/v1/students?active=true
\___/   \_____________/ \_/ \__________/ \_________/
şema          host      port     yol          sorgu
```

HTTPS, HTTP mesajını TLS ile korur. AhdCode sistemin güvenilen kök
sertifikalarıyla sunucuyu doğrular; doğrulamayı kapatan bir seçenek yoktur.

### 7.2 GET isteği ve durum kodu

```ahd
bring HTTP
from HTTP bring Client
from HTTP bring ClientResponse

client: Client := HTTP.client(10)
response: ClientResponse := client.get("https://example.com/")

write(response.status())
write(response.url())
contentType: String? := response.header("Content-Type")
if contentType != null {
    write(contentType)
}
write(response.body())
```

`HTTP.client(10)` tüm istek için 10 saniyelik zaman aşımı kullanır. Yanıt
başlığı bulunmayabilir; `header(...)` bu yüzden `String?` döndürür.

### 7.3 4xx/5xx ile taşıma hatasını ayırmak

```ahd
from HTTP bring HTTPError

attempt {
    response: Local ClientResponse := client.get("https://example.com/missing")
    if response.status() >= 400 {
        write("sunucu hata durumu: {str(response.status())}")
    } else {
        write(response.body())
    }
}
except HTTPError as error {
    write("istek gonderilemedi: {error.message}")
}
```

`404`, `429` ve `500` ulaşılmış sunucunun yanıtıdır; `ClientResponse` olarak
döner. DNS, TLS, bozuk URL veya zaman aşımı gibi durumlarda geçerli bir HTTP
yanıtı yoktur ve `HTTPError` oluşur.

### 7.4 JSON POST ve gizli değerler

```ahd
bring JSON
from JSON bring JSONValue
bring Env

token: String := Env.getOr("API_TOKEN", "")
payload: JSONValue := JSON.object({
    "name": JSON.fromString("Ayse")
    "score": JSON.fromInt(91)
})

request := HTTP.clientRequest("POST", "https://api.example.com/v1/results")
request = request.withHeader("Authorization", "Bearer {token}")
request = request.withHeader("Content-Type", "application/json")
request = request.withBody(JSON.stringify(payload))

response := client.send(request)
if response.status() >= 200 and response.status() < 300 {
    parsed: JSONValue := JSON.parse(response.body())
    write(JSON.stringify(parsed))
}
```

Jetonu kaynak dosyaya yazmayın. `withHeader` aynı adlı başlığı değiştirir;
`addHeader` aynı ada ikinci değer ekler. İstek değiştirilemez olduğu için
her iki metot da yeni `ClientRequest` döndürür.

### 7.5 Güvenli istemci kontrol listesi

- Dış isteğe mutlaka makul bir zaman aşımı verin.
- Başarılı gövdeyi ayrıştırmadan önce durum kodunu kontrol edin.
- JSON bekliyorsanız `Content-Type` ve gerçek gövdeyi birlikte doğrulayın.
- Parola ve API jetonlarını `Env` ile alın; günlükte yazdırmayın.
- Kullanıcıdan gelen tam URL'yi sınırsız biçimde çağırmak SSRF riski doğurur;
  izin verilen hostları uygulamanız belirlesin.
- HTTPS sertifika doğrulamasını aşmaya çalışmayın.

### Atölye görevi

Bir HTTPS sayfasını 5 saniyelik istemciyle alın. Durum kodunu ve son URL'yi
yazdırın; yalnızca 2xx durumunda gövdeyi işleyin. Ulaşılamayan bir host ve 404
yolu deneyerek iki hata kanalının farklı olduğunu gözlemleyin.

Sunucu, çerez, oturum, yükleme ve istemci sınırları için
[HTTP referansına](HTTP_TR.md) bakın.

## 8. HTML: güvenli sayfa kurmak ve belge ayrıştırmak

HTML modülünün iki bağımsız yönü vardır:

```text
HTML.text / element / document  güvenli işaretleme kurar
HTML.parse / select / first     var olan işaretlemeyi inceler
```

`HTML.parse` URL indirmez ve JavaScript çalıştırmaz. Ağ gerekiyorsa önce HTTP
Client ile String gövdeyi alırsınız.

### 8.1 Güvenli dinamik sayfa

```ahd
bring HTML

userName: String := "<script>alert(1)</script>"

page: String := HTML.document(
    "Ogrenci Paneli"
    [
        HTML.element("h1", {}, [HTML.text("Hos geldiniz")])
        HTML.element("p", {"class": "student"}, [HTML.text(userName)])
        HTML.element("a", {"href": "/results"}, [HTML.text("Sonuclar")])
    ]
)
```

`HTML.text`, `<`, `>`, `&` ve tırnakları kaçırır; kullanıcı adı kod olarak
çalışmaz. `HTTP.html(page)` yalnızca içerik türünü ayarlar, sonradan kaçış
yapmaz. Bu yüzden dinamik değeri ham String birleştirmesiyle etikete koymayın.

### 8.2 Ayrıştırma ve null kontrolü

```ahd
bring HTML
from HTML bring HTMLDocument

document: HTMLDocument := HTML.parse(
    "<article class=\"card\"><h2>Sinav</h2><a href=\"/1\">Ac</a></article>"
)

heading := document.first("article.card > h2")
if heading != null {
    write(heading.text())
}
```

`first` eşleşme yoksa `null`, `select` ise boş `List` döndürür. Bu iki sözleşme
farklıdır; `first(...).text()` şeklinde null kontrolünü atlamayın.

### 8.3 Seçiciler ve kapsam

Desteklenen küçük CSS seçici kümesi şunları kapsar:

- Etiket: `article`
- Kimlik: `#main`
- Sınıf: `.card`
- Öznitelik: `[href]` veya `[data-id="42"]`
- Birleşim: `article.card[data-id]`
- Alt öğe: `article a`
- Doğrudan çocuk: `article > h2`
- Alternatif: `h1, h2`

Bir `HTMLElement` üzerinde seçim yapmak aramayı o öğenin alt ağacıyla sınırlar:

```ahd
cards := document.select("article.card")
for card in cards {
    title: Local := card.first("h2")
    link: Local := card.first("a[href]")
    if title != null {
        if link != null {
            href: Local String? := link.attr("href")
            if href != null {
                write("{title.text()} -> {href}")
            }
        }
    }
}
```

`:nth-child`, `+` ve `~` desteklenmez; geçersiz veya desteklenmeyen seçici
`HTMLError` üretir.

### 8.4 HTTPS ile al, HTML ile ayrıştır

```ahd
bring HTTP
bring HTML
from HTTP bring ClientResponse
from HTML bring HTMLDocument

response: ClientResponse := HTTP.client(10).get("https://example.com/")
if response.status() == 200 {
    document: HTMLDocument := HTML.parse(response.body())
    title := document.first("h1")
    if title != null {
        write(title.text())
    }
}
```

HTML'deki göreli `/about` adresi otomatik olarak
`https://example.com/about` yapılmaz. Modül tarayıcı değildir; URL çözümleme,
JavaScript, CSS düzeni ve ekran görüntüsü üretmez.

### 8.5 Kazıma etiği ve dayanıklılık

- Sitenin kullanım koşullarına ve erişim kurallarına uyun.
- Çok hızlı ve sınırsız istek göndermeyin.
- Seçicilerin sayfa tasarımı değişince bozulabileceğini kabul edin.
- `first` sonuçlarında null kontrolü yapın.
- Aldığınız veri kritikse kaynak URL ve alınma zamanını da saklayın.
- Giriş gerektiren veya JavaScript ile sonradan oluşan içeriğin düz HTTP
  gövdesinde bulunmayabileceğini unutmayın.

### Atölye görevi

Küçük bir HTML String'inde üç `article.card` oluşturun. Her karttan başlık ve
`href` çıkarıp bir `List<Pair<String, String>>` hazırlayın; sonra
`CSV.stringifyRecords` ile CSV metnine dönüştürün.

Oluşturucu, ayrıştırıcı ve seçici sözleşmesi için
[HTML referansına](HTML_TR.md) bakın.

## 9. Bitirme projesi: veriden iki rapor üretmek

Bu proje modüllerin sorumluluklarını bir arada gösterir:

1. `CSV.readRecords` ile `sonuclar.csv` dosyasını okuyun.
2. `Data.fromRecords` ile `Table` oluşturun.
3. Sayısal puanları açıkça `List<Real>`'e dönüştürün.
4. `Statistics.mean` ile ortalamayı hesaplayın.
5. `Plot.bar` ile PNG grafik üretin.
6. `Excel` ile tipli hücreler ve formüller içeren `sonuclar.xlsx` yazın.
7. `Word` ile tabloyu ve grafiği içeren `sonuclar.docx` yazın.
8. İsterseniz aynı gövdeyi `Latex` ile `sonuclar.pdf` olarak üretin.

Projeyi tek seferde yazmayın. Her sınırda ara sonucu doğrulayın:

```text
CSV kayıt sayısı
    ↓
Table sütunları ve satır sayısı
    ↓
sayısal listenin uzunluğu ve ortalaması
    ↓
grafik dosyasının oluşması
    ↓
Excel dosyasını yeniden okuyarak hücre türleri
    ↓
Word belgesini yeniden okuyarak başlık ve tablo metni
```

Bu kontroller, “program çalıştı” ile “çıktı doğru” arasındaki farktır.

## Hangi belgeye ne zaman bakmalıyım?

| İhtiyaç | Önce burası | Sonra ayrıntılı referans |
|---|---|---|
| CSV okuyup yazmak | Atölye 1 | [CSV](CSV_TR.md) |
| Tablo filtrelemek/gruplamak | Atölye 2 | [Data](DATA_TR.md) |
| Grafik üretmek ve rapora gömmek | Atölye 3 | [Plot](PLOT_TR.md) |
| XLSX üretmek | Atölye 4 | [Excel](EXCEL_TR.md) |
| DOCX raporu üretmek | Atölye 5 | [Word](WORD_TR.md) |
| Akademik PDF/slayt | Atölye 6 | [Latex](LATEX_TR.md) |
| API veya web isteği | Atölye 7 | [HTTP](HTTP_TR.md) |
| HTML kurmak/ayrıştırmak | Atölye 8 | [HTML](HTML_TR.md) |

Ana dil temelleri için [Türkçe Öğrenci Rehberi](STUDENT_GUIDE_TR.md), çalışan
tam programlar için [örnekler dizini](../examples/v0.1/README_TR.md) başlangıç
noktasıdır.
