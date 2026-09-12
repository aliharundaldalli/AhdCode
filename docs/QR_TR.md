# QR standart modülü

[English](QR.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Barcode](BARCODE_TR.md) · [Latex](LATEX_TR.md) · [PDF](PDF_TR.md)

QR (v1.3.0), QR kodlarını programın içinde oluşturur: ağ servisi, harici araç
veya tarayıcı yoktur. Açıkça içe aktarılır:

```ahd
bring QR
from QR bring QRCode
from QR bring QRError
```

Kanonik modül kimliği `builtin:QR`'dir; yan dizindeki bir `QR.ahd` onu
gölgeleyemez. Her argüman `NonNull` olmalıdır.

## Yüzey

```text
QR.create(value: String, level: String = "M") -> QRCode

QRCode.value()                                -> String
QRCode.level()                                -> String
QRCode.size()                                 -> Int
QRCode.matrix()                               -> List<List<Bool>>
QRCode.savePNG(path: String, pixels: Int = 512) -> Nothing
QRCode.saveSVG(path: String, size: Real = 5.0)  -> Nothing

QRCode
QRError
```

`QRCode` işlemleri, `PDFDocument` ve `Document` gibi yalnızca konumsal
argüman alır.

## Kod oluşturma

`QR.create(value, level)`, boş olmayan her UTF-8 String'i kodlar: bir URL,
düz metin, Türkçe harfler veya emoji. Kodlayıcı, değer için en küçük sembolü ve
en sıkı kodlamayı kendisi seçer.

`level`, hata düzeltme seviyesidir; tam olarak `"L"`, `"M"`, `"Q"` veya `"H"`
olmalıdır (büyük/küçük harf duyarlı). Daha yüksek bir seviye, okuyucunun hasar
görmüş veya kısmen kapanmış bir sembolü daha fazla kurtarmasını sağlar ve aynı
değer için daha çok modül gerektirir:

| Seviye | Yaklaşık kurtarma |
|---|---|
| `"L"` | %7 |
| `"M"` (varsayılan) | %15 |
| `"Q"` | %25 |
| `"H"` | %30 |

```ahd
site: QRCode := QR.create("https://ahdcode.org")
contact: QRCode := QR.create("Ayşe Yılmaz, +90 555 000 00 00", "Q")
```

`QR.create`, değeri doğrulamak için bir kez kodlar; bu yüzden her `QRCode`
değeri çizilebilir. İstenen seviyede en büyük sembole sığmayacak kadar uzun bir
değer `QRError` fırlatır.

## Sembolü okuma

Bir `QRCode` değişmezdir.

- `value()` ve `level()`, kodun oluşturulduğu değerleri döndürür.
- `size()`, sessiz bölge hariç bir kenardaki modül sayısıdır; en küçük sembolde
  21, en büyükte 177'dir.
- `matrix()`, modülleri yukarıdan aşağıya satır satır, her satırı soldan sağa,
  koyu modül için `true` olarak ve sessiz bölge olmadan döndürür. Her çağrı yeni
  bir List döndürür; onu değiştirmek `QRCode`'u asla değiştirmez.

```ahd
matrix: List<List<Bool>> := site.matrix()
write(str(site.size()) + " modules per side, " + str(len(matrix)) + " rows")
```

## PNG çıktısı

`savePNG(path, pixels)`, her kenarda dört açık modüllük standart sessiz bölge
dahil, tam olarak `pixels` x `pixels` boyutunda beyaz zemin üzerine siyah bir
PNG yazar. Her modül tam sayıda pikseldir; bu yüzden modül kenarları keskin
kalır. En büyük tam modül boyutundan artan pikseller sessiz bölgeyi eşit olarak
genişletir. `pixels` en az `size() + 8`, en çok 10000 olmalıdır.

```ahd
site.savePNG("site.png")
site.savePNG("site-print.png", 2048)
```

## SVG çıktısı

`saveSVG(path, size)`, sessiz bölge dahil `size` santimetre genişliğinde kare
bir vektör SVG yazar: beyaz bir kare ve modül dizilerinden oluşan tek bir siyah
path. `size` 0'dan büyük ve en çok 1000 olmalıdır.

```ahd
site.saveSVG("site.svg", 3.0)
```

İki kaydetme işlemi de uygun uzantıyı (`.png` veya `.svg`) ister, hedef
dizindeki geçici bir dosyaya yazar, kodlanmış baytları doğrular ve hedefi tek
adımda değiştirir; başarısız bir kaydetme asla yarım bir dosya bırakmaz.

## Belgelerde QR kodları

`Latex.qr(value, size, level)` ve `PDFDocument.qr(value, size, level, align)`,
aynı sembolü aynı kodlayıcıdan PDF içinde vektör grafik olarak çizer; bu yüzden
bir değer her yerde aynı modülleri üretir. Yalnızca `bring Latex` veya
`bring PDF` gerekir:

```ahd
body += L.place(L.qr("https://ahdcode.org/verify?id=42", 2.5, "Q"), 19.0, 26.0, "south east")
doc = doc.qr("https://ahdcode.org/verify?id=42", 2.5, "Q", "right")
```

Bkz. [Latex](LATEX_TR.md#qr-kodları-ve-barkodlar-v130) ve
[PDF](PDF_TR.md#qr-kodları-ve-barkodlar).

## Hatalar

`QRError`; geçersiz seviye, boş veya çok uzun değer, geçersiz `pixels` veya
`size`, yanlış uzantı ve yazma hatalarını kapsar:

```ahd
attempt {
    QR.create("https://ahdcode.org", "X")
}
except QRError as error {
    write(error.message)
}
```

`Latex.qr` aynı sorunları `ValueError`, `PDFDocument.qr` ise `PDFError` olarak
bildirir. Statik argüman sayısı ve tür hataları derleyici tanılaması olarak
kalır.

## Uygulama ve lisans

Kodlayıcı [`github.com/boombuler/barcode`](https://github.com/boombuler/barcode)
v1.1.0'dır (MIT Lisansı); AhdCode kaynak ağacına ve QR kodu kullanan her native
programa vendor edilir. QR kodu kullanmayan bir program ondan hiçbir şey
taşımaz. Derleme veya çalışma sırasında hiçbir şey indirilmez. Bkz.
[`THIRD_PARTY_NOTICES_CODES.md`](../THIRD_PARTY_NOTICES_CODES.md).

## Bu sürümde yok

Görüntülerden veya kameradan QR kodu okuma/çözme, renkler, logolar, yuvarlatılmış
veya stillendirilmiş modüller, Micro QR, structured append ve sembol sürümünü,
maskeyi ya da kodlama kipini elle seçme v1.3.0'ın parçası değildir.
