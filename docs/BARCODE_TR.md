# Barcode standart modülü

[English](BARCODE.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [QR](QR_TR.md) · [Latex](LATEX_TR.md) · [PDF](PDF_TR.md)

Barcode (v1.3.0), doğrusal barkodları programın içinde oluşturur: ağ servisi,
harici araç veya yazı tipi yoktur. Açıkça içe aktarılır:

```ahd
bring Barcode
from Barcode bring BarcodeCode
from Barcode bring BarcodeError
```

Kanonik modül kimliği `builtin:Barcode`'dur; yan dizindeki bir `Barcode.ahd`
onu gölgeleyemez. Her argüman `NonNull` olmalıdır.

## Yüzey

```text
Barcode.code128(value: String) -> BarcodeCode
Barcode.ean13(value: String)   -> BarcodeCode
Barcode.upca(value: String)    -> BarcodeCode

BarcodeCode.kind()    -> String
BarcodeCode.value()   -> String
BarcodeCode.pattern() -> List<Bool>
BarcodeCode.savePNG(path: String, width: Int = 800, height: Int = 240) -> Nothing
BarcodeCode.saveSVG(path: String, width: Real = 8.0, height: Real = 2.4) -> Nothing

BarcodeCode
BarcodeError
```

`BarcodeCode` işlemleri yalnızca konumsal argüman alır.

## Semboloji türleri

| Fonksiyon | `kind()` | Kabul eder | Sessiz bölgeler (modül) | Tipik kullanım |
|---|---|---|---|---|
| `Barcode.code128` | `"Code128"` | 1 ile 80 arası ASCII karakter | solda 10, sağda 10 | sipariş numaraları, stok kodları, gönderi |
| `Barcode.ean13` | `"EAN13"` | 12 basamak veya kontrol basamağıyla 13 | solda 11, sağda 7 | perakende ürünler |
| `Barcode.upca` | `"UPCA"` | 11 basamak veya kontrol basamağıyla 12 | solda 9, sağda 9 | Kuzey Amerika'daki perakende ürünler |

```ahd
product: BarcodeCode := Barcode.ean13("590123412345")
grocery: BarcodeCode := Barcode.upca("03600029145")
order: BarcodeCode := Barcode.code128("AHD-2026-0042")
```

**Kontrol basamakları.** EAN-13 ve UPC-A, GS1 mod 10 kontrol basamağıyla
biter. Yalnızca veri basamaklarını verirseniz Barcode onu hesaplar; tam numarayı
verirseniz Barcode onu doğrular. Yanlış bir kontrol basamağı, doğrusunu
belirten bir `BarcodeError` fırlatır; asla sessizce değiştirilmez. `value()`
her zaman kontrol basamağını içerir:

```ahd
write(product.value())
```

`5901234123457` yazdırır. Bir UPC-A sembolünün çubukları, aynı numaranın başına
sıfır eklenmiş EAN-13 çubuklarıdır; EAN-13 okuyucuların UPC-A okuyabilmesinin
nedeni budur.

**Code 128**, en sıkı kod kümelerini kendisi seçerek ASCII metni kodlar. ASCII
dışındaki bir karakter, konumunu belirten bir `BarcodeError` fırlatır; Unicode
metin için bir [QR kodu](QR_TR.md) kullanın.

## Sembolü okuma

Bir `BarcodeCode` değişmezdir. `kind()` sembolojiyi adlandırır, `value()`
çubukların taşıdığı değerdir ve `pattern()` modülleri soldan sağa, koyu modül
için `true` olarak ve sessiz bölgeler olmadan döndürür. Bir EAN-13 veya UPC-A
deseni her zaman 95 modüldür. Her çağrı yeni bir List döndürür.

## PNG çıktısı

`savePNG(path, width, height)`, sessiz bölgeler dahil tam olarak `width` x
`height` piksellik bir görüntüye beyaz zemin üzerine siyah çubuklar yazar. Her
modül tam sayıda piksel genişliğindedir; bu yüzden çubuk kenarları keskin
kalır. En büyük tam modül genişliğinden artan pikseller sessiz bölgeleri eşit
olarak genişletir. `width`, sessiz bölgeler dahil modül sayısından az ve
10000'den çok olamaz; `height` 1 ile 10000 arasında olmalıdır.

```ahd
product.savePNG("product.png")
order.savePNG("order.png", 1200, 300)
```

## SVG çıktısı

`saveSVG(path, width, height)`, sessiz bölgeler dahil `width` x `height`
santimetrelik bir vektör SVG yazar. İkisi de 0'dan büyük ve en çok 1000
olmalıdır.

```ahd
product.saveSVG("product.svg", 6.0, 2.0)
```

İki kaydetme işlemi de uygun uzantıyı (`.png` veya `.svg`) ister, hedef
dizindeki geçici bir dosyaya yazar, kodlanmış baytları doğrular ve hedefi tek
adımda değiştirir; başarısız bir kaydetme asla yarım bir dosya bırakmaz.

## Belgelerde barkodlar

`Latex.barcode(kind, value, width, height)` ve
`PDFDocument.barcode(kind, value, width, height, align)`, aynı çubukları aynı
kodlayıcıdan PDF içinde vektör grafik olarak çizer. `kind`, `"Code128"`,
`"EAN13"` veya `"UPCA"`'dır. Yalnızca `bring Latex` veya `bring PDF` gerekir:

```ahd
body += L.barcode("EAN13", "590123412345", 6.0, 2.0)
doc = doc.barcode("Code128", "AHD-2026-0042", 7.0, 1.5, "left")
```

Bkz. [Latex](LATEX_TR.md#qr-kodları-ve-barkodlar-v130) ve
[PDF](PDF_TR.md#qr-kodları-ve-barkodlar).

## Hatalar

`BarcodeError`; boş, çok uzun, ASCII olmayan veya basamak dışı karakter içeren
değeri, yanlış basamak sayısını veya kontrol basamağını, geçersiz boyutu, yanlış
uzantıyı ve yazma hatalarını kapsar:

```ahd
attempt {
    Barcode.ean13("5901234123450")
}
except BarcodeError as error {
    write(error.message)
}
```

`Latex.barcode` aynı sorunları `ValueError`, `PDFDocument.barcode` ise
`PDFError` olarak bildirir. Statik argüman sayısı ve tür hataları derleyici
tanılaması olarak kalır.

## Uygulama ve lisans

Kodlayıcılar [`github.com/boombuler/barcode`](https://github.com/boombuler/barcode)
v1.1.0'dır (MIT Lisansı); AhdCode kaynak ağacına ve barkod kullanan her native
programa vendor edilir. Derleme veya çalışma sırasında hiçbir şey indirilmez.
Bkz. [`THIRD_PARTY_NOTICES_CODES.md`](../THIRD_PARTY_NOTICES_CODES.md).

## Bu sürümde yok

Barkod tarama veya çözme, çubukların altına basılan okunabilir rakamlar, EAN-8,
UPC-E, EAN-2/EAN-5 ekleri, ITF-14, Code 39, GS1-128 uygulama tanımlayıcıları,
QR dışındaki 2D sembolojiler (Data Matrix, PDF417, Aztec) ve renkler v1.3.0'ın
parçası değildir.
