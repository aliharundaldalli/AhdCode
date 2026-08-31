# Word standart modülü

[English](WORD.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Plot](PLOT_TR.md)

Word, değiştirilemez belgeler oluşturur, gerçek `.docx` paketleri yazar ve
mevcut DOCX dosyalarının küçük bir anlamsal alt kümesini okur:

```ahd
bring Word
from Word bring Document
from Word bring WordError
```

Kanonik modül kimliği `builtin:Word`'dür; kardeş bir `Word.ahd` dosyası onu
gölgeleyemez. Word yalnızca Go standart kitaplığını kullanır; Office kurulumu,
harici süreç, ağ erişimi veya ek çalışma zamanı paketi gerektirmez.

Word, sınırlandırılmış anlamsal bir DOCX alt kümesi oluşturur ve okur. Bir
Office klonu değildir ve piksel düzeyinde kusursuz round-trip koruması vaat
etmez.

## Yüzey

```text
Word.new()                 -> Document
Word.read(path: String)    -> Document

Document.heading(text: String, level: Int) -> Document
Document.paragraph(
    text: String,
    align: String = "left",
    bold: Bool = false,
    italic: Bool = false,
    underline: Bool = false
) -> Document
Document.table(
    headers: List<String>,
    rows: List<List<String>>,
    merges: List<List<Int>> = [],
    align: String = "left"
) -> Document
Document.image(path: String, size: Pair<String, Real> = {}) -> Document
Document.pageBreak()       -> Document
Document.save(path: String) -> Nothing

Document.text()            -> String
Document.paragraphs()      -> List<String>
Document.headings()        -> List<String>
Document.tables()          -> List<List<List<String>>>

Document
WordError
```

Yukarıdaki etiketler argüman sırasını belgeler. `Document` işlemleri kaynak
kodda yalnızca konumsaldır:

```ahd
document = document.paragraph("Önemli", "center", true, false, false)
```

`document.paragraph(text: "Önemli", bold: true)` bilinçli olarak reddedilir.
String, List, Table, Chart, Vector ve Matrix üyeleri dahil derleyici tarafından
sağlanan bütün tip işlemleri aynı yalnız-konumsal dispatch mekanizmasını
kullanır. Yalnız Document için isimli argüman eklemek ortak API'yi yeniden
tasarlayacağı için Word mevcut dil kuralını izler. `Word.read(path:
"rapor.docx")` gibi modül Function'ları normal çağrı kurallarını kullanır.

## Değiştirilemez oluşturma

`Word.new()` boş bir Document döndürür. Her içerik işlemi yeni bir Document
döndürür ve alıcısını değiştirmez:

```ahd
base: Document := Word.new()
first: Document := base.paragraph("Bir")
second: Document := base.paragraph("İki")
```

Başlıklar, satırlar, merge tanımları ve görsel baytları işlem anında
kopyalanır. Bir girdi List'ini sonradan değiştirmek Document'ı etkileyemez;
`image()` sonrasında kaynak dosyanın silinmesi de DOCX içindeki görseli
kaldırmaz.

## Başlıklar ve paragraflar

Heading seviyesi `1`–`6` arasındadır; başka değer `WordError` fırlatır.
Paragraf hizası tam olarak `"left"`, `"center"`, `"right"` veya
`"justify"` olmalıdır. Bold, italic ve underline paragrafın tek metin run'ına
uygulanır.

```ahd
document: Document := Word.new()
document = document.heading("Üç aylık rapor", 1)
document = document.paragraph("Çevrimdışı hazırlandı.")
document = document.paragraph("Onaylandı", "right", true, true, true)
document = document.pageBreak()
```

Metin XML için kaçışlanır, XML 1.0'ın yasakladığı kontrol karakterleri atılır
ve Unicode korunur.

## Tablolar ve birleştirmeler

Her satır `headers` ile aynı hücre sayısına sahip olmalı ve en az bir sütun
bulunmalıdır. Tablo hizası `"left"`, `"center"` veya `"right"` olabilir.

Her merge tanımı dört Int değeridir:

```text
[row, column, rowSpan, columnSpan]
```

Koordinatlar sıfır tabanlıdır ve `0`. satır header satırıdır. Bölge tablo
içinde kalmalı, span'ler pozitif olmalı, birden fazla hücreyi kapsamalı ve
başka bir merge ile çakışmamalıdır. Yatay birleştirmeler WordprocessingML
`gridSpan`, dikey birleştirmeler `vMerge` restart/continuation kullanır.

```ahd
document = document.table(
    ["Bölge", "Q1", "Q2"]
    [["Kuzey", "10", "12"], ["Güney", "8", "11"]]
    [[0, 0, 1, 3], [1, 0, 2, 1]]
    "center"
)
```

Bozuk tanımlar, negatif koordinatlar, sıfır span, 1x1 merge, sınır dışı
bölgeler, çakışmalar ve satır genişliği uyuşmazlıkları paket yazılmadan önce
`WordError` fırlatır.

## Görseller

Word PNG ve JPEG baytlarını hemen gömer. `size` yalnızca santimetre cinsinden
`"width"` ve `"height"` anahtarlarını kabul eder:

```ahd
document = document.image("chart.png")
document = document.image("chart.png", {"width": 12.0})
document = document.image("photo.jpg", {"height": 6.0})
document = document.image("logo.png", {"width": 4.0, "height": 3.0})
```

Tek boyut doğal en-boy oranını korur; iki boyut açık kutuyu kullanır; boş Pair
96 DPI'daki doğal piksel boyutunu kullanır. Boyutlar pozitif olmalıdır. Eksik
dosya, çözülemeyen veri, desteklenmeyen biçim/anahtar/boyut `WordError`
fırlatır. Plot entegrasyonu açık dosya sınırından geçer:

```ahd
chart.save("scores.png")
document = document.image("scores.png", {"width": 14.0})
```

## Kaydetme

`save(path)` bir `.docx` hedefi kabul eder ve `Nothing` döndürür; sonucunu bir
değişkene atamayın:

```ahd
document.save("rapor.docx")
```

Paketin relationship kimlikleri, medya adları, stilleri ve ZIP üye sırası
deterministiktir. Aynı Document'ı iki kez kaydetmek aynı baytları üretir. Word
paketin tamamını oluşturup doğruladıktan sonra hedef dizinde atomik olarak
yayımlar; başarısız save mevcut hedefi değiştirmez.

## Okuma ve erişim işlemleri

`Word.read(path)`, `word/document.xml` içinden paragraf metnini, Heading 1–6
metnini ve tablo hücre metnini kurtarır. Paragraf içindeki tab ve satır sonları
`\t` ve `\n` olur. Desteklenmeyen biçimlendirme, özel stiller, header/footer,
yorumlar, sayfa geometrisi, görseller ve bilinmeyen relationship'ler yeniden
üretilmez; güvenle yok sayılır.

```ahd
loaded: Document := Word.read("rapor.docx")
write(loaded.text())
write(loaded.headings())
write(loaded.paragraphs())
write(loaded.tables())
```

`text()` heading ve paragrafları newline ile birleştirir. `tables()` her tabloyu
fiziksel satırlarıyla döndürür; ilk satır `0`. indekstedir. Erişim List'leri
yeni snapshot'lardır.

Okuma sınırlandırılmıştır: arşiv baytı, üye sayısı, tekil/toplam açılmış boyut
ve sıkıştırma oranı limitleri vardır. Mutlak/path-traversal yollar, yinelenen
üyeler, geçersiz ZIP, eksik veya bozuk `word/document.xml`, aşırı büyük içerik
ve mantıksız sıkıştırma `WordError` ile reddedilir. Relationship izlenmez ve
ağa erişilmez.

## Hatalar

`WordError`, Word'e özgü doğrulama, görsel, DOCX paketleme, save ve read
hatalarını kapsar:

```ahd
attempt {
    Word.read("missing.docx")
}
except WordError as error {
    write(error.message)
}
```

Statik argüman sayısı ve tip hataları derleyici tanısı olarak kalır; runtime
`WordError` değerine dönüşmez.
