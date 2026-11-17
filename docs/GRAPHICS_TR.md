# Graphics standart modülü

[English](GRAPHICS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Plot](PLOT_TR.md) · [Math](MATH_TR.md)

`Graphics`, AhdCode v1.6.0 ile gelen, derleyicinin kaydettiği `builtin:Graphics`
modülüdür. Bir pencere içinde 2B bir Canvas (tuval) açar; üzerine matematiksel
(Kartezyen) koordinatlarda çizgi, çember, dikdörtgen, elips, çokgen, yay, metin
ve resim çizer, üzerinde bir Turtle (kaplumbağa) kalemi gezdirir ve çizimi PNG
ya da SVG olarak kaydeder.

Graphics görsel programlama ve 2B çizim içindir: koordinatlar, açılar,
geometri, dönüşümler, fraktallar ve bir matematik ya da programlama dersindeki
kaplumbağa programları. **Bir oyun motoru ya da GUI (arayüz) kiti değildir.**
Sprite, sahne, çarpışma, fizik, girdi yoklama, ses, arayüz bileşenleri ya da 3B
içermez ve bunlar onun için planlanmamıştır.
v1.8.0'dan itibaren bir Canvas tıklamaları ve tuş
basışlarını bir callback'e bildirebilir; böylece bir Turtle ok tuşlarıyla
yönlendirilebilir; bkz. [Tıklamalar ve tuş basışları](#tıklamalar-ve-tuş-basışları).
v2.8.0'dan itibaren metin, çokgen, elips, yay ve resim de çizer, işaretçinin
hareketini ve sürüklemeyi bildirir ve zamanlayıcı çalıştırır; hareketli bir
resim ya da fareyle yapılan bir çizim için bu kadarı yeter. Bkz.
[Metin](#metin), [Resimler](#resimler),
[İşaretçi hareketi ve sürükleme](#i̇şaretçi-hareketi-ve-sürükleme) ve
[Zamanlayıcılar ve animasyon](#zamanlayıcılar-ve-animasyon).
Düğmeli ve metin alanlı pencereler için [GUI](GUI_TR.md) kullanın.

## Genel yüzey

```text
bring Graphics
from Graphics bring (Canvas, Turtle, GraphicsError)

Graphics.open(
    width: Int := 800
    height: Int := 600
    title: String := "AhdCode Graphics"
    background: String := "white"
)                                                                    -> Canvas

canvas.clear(color: String := "white")                               -> Nothing
canvas.line(
    x1: Real, y1: Real, x2: Real, y2: Real
    color: String := "black"
    width: Real := 1.0
)                                                                    -> Nothing
canvas.circle(
    x: Real, y: Real, radius: Real
    stroke: String := "black"
    fill: String? := null
    width: Real := 1.0
)                                                                    -> Nothing
canvas.rectangle(
    x: Real, y: Real, width: Real, height: Real
    stroke: String := "black"
    fill: String? := null
    lineWidth: Real := 1.0
)                                                                    -> Nothing
canvas.ellipse(
    x: Real, y: Real, radiusX: Real, radiusY: Real
    stroke: String := "black"
    fill: String? := null
    width: Real := 1.0
)                                                                    -> Nothing
canvas.polygon(
    points: List<Real>            // x0, y0, x1, y1, ...; List<Int> de olur
    stroke: String := "black"
    fill: String? := null
    width: Real := 1.0
)                                                                    -> Nothing
canvas.arc(
    x: Real, y: Real, radius: Real, start: Real, end: Real
    color: String := "black"
    width: Real := 1.0
)                                                                    -> Nothing
canvas.text(
    x: Real, y: Real, text: String
    color: String := "black"
    size: Real := 16.0
    align: String := "left"
)                                                                    -> Nothing
canvas.image(
    path: String, x: Real, y: Real
    width: Real := 0.0
    height: Real := 0.0
)                                                                    -> Nothing
canvas.batch(handler: () -> Nothing)                                 -> Nothing
canvas.save(path: String)                                            -> Nothing
canvas.wait()                                                        -> Nothing
canvas.onClick(handler: (x: Real, y: Real) -> Nothing)               -> Nothing
canvas.onKey(handler: (key: String) -> Nothing)                      -> Nothing
canvas.onMove(handler: (x: Real, y: Real) -> Nothing)                -> Nothing
canvas.onDrag(handler: (x: Real, y: Real) -> Nothing)                -> Nothing
canvas.every(milliseconds: Int, handler: () -> Nothing)              -> Int
canvas.cancel(timer: Int)                                            -> Nothing
canvas.close()                                                       -> Nothing
canvas.isOpen()                                                      -> Bool
canvas.turtle()                                                      -> Turtle

turtle.forward(distance: Real)                                       -> Nothing
turtle.backward(distance: Real)                                      -> Nothing
turtle.left(degrees: Real)                                           -> Nothing
turtle.right(degrees: Real)                                          -> Nothing
turtle.moveTo(x: Real, y: Real)                                      -> Nothing
turtle.setHeading(degrees: Real)                                     -> Nothing
turtle.penUp()                                                       -> Nothing
turtle.penDown()                                                     -> Nothing
turtle.setColor(color: String)                                       -> Nothing
turtle.setWidth(width: Real)                                         -> Nothing
turtle.home()                                                        -> Nothing
turtle.x()                                                           -> Real
turtle.y()                                                           -> Real
turtle.heading()                                                     -> Real

GraphicsError  (Error'dan türer)
```

`Canvas` ve `Turtle`'ın kurucusu yoktur: bir Canvas `Graphics.open` ile, bir
Turtle `canvas.turtle()` ile elde edilir. `bring Turtle` diye bir şey yoktur;
Turtle, Graphics'e aittir.

## İlk çizim

```ahd
bring Graphics

canvas := Graphics.open(400, 300)
canvas.line(-200, 0, 200, 0)
canvas.line(0, -150, 0, 150)
canvas.circle(x: 0, y: 0, radius: 80, stroke: "blue", fill: "#ffff0080")

turtle := canvas.turtle()
turtle.setColor("red")
side := 0
while side < 4 {
    turtle.forward(100)
    turtle.left(90)
    side = side + 1
}

canvas.save("ilk.png")
canvas.wait()
```

Program iki eksen, yarı saydam sarı bir çember ve sol alt köşesi başlangıç
noktası olan kırmızı bir kare çizer, resmi kaydeder ve siz kapatana kadar
pencereyi açık tutar.

## Graphics'i çağırmak

AhdCode'daki her çağrı gibi, her çağrı ya tamamen konumsal ya da tamamen
isimli argümanlarla yapılır. Varsayılanı olan argümanlar yazılmayabilir; isimli
argümanlarla bunların herhangi biri, herhangi bir sırayla atlanabilir.

```ahd
bring Graphics

canvas := Graphics.open(800, 600)
labelled := Graphics.open(
    width: 640
    height: 480
    title: "Şekiller"
    background: "black"
)
canvas.line(0, 0, 100, 50, "red", 2)
canvas.line(x1: 0, y1: 0, x2: 100, y2: 50, width: 3)
canvas.circle(x: 0, y: 0, radius: 40, fill: "yellow")
```

İki biçimi karıştıran bir çağrı derleme hatasıdır. `Real` beklenen her yerde
tamsayı argümanlar da kabul edilir.

## Koordinatlar

Her Canvas matematiksel koordinatlar kullanır:

- başlangıç noktası `(0, 0)` Canvas'ın **merkezidir**;
- `+x` sağı, `-x` solu gösterir;
- `+y` **yukarıyı**, `-y` aşağıyı gösterir;
- bir birim, pencerenin bir pikselidir.

800 × 600'lük bir Canvas'ta `x` -400'den +400'e, `y` -300'den +300'e kadar
görünür. Canvas'ın dışına taşan çizim kırpılır; bu bir hata değildir.

`canvas.rectangle(x, y, width, height)` bu koordinatlardaki doğal köşe olan
**sol alt** köşeyi alır ve sağa ve yukarı doğru büyür. Hiçbir zaman sol üst köşe
olarak yorumlanmaz.

## Renkler

Her renk argümanı şunlardan biridir:

- tam olarak böyle (küçük harfle) yazılmış dokuz addan biri:
  `black`, `white`, `red`, `green`, `blue`, `yellow`, `cyan`, `magenta`, `gray`;
- `#RRGGBB`, altı onaltılık rakam (`#1e90ff`);
- `#RRGGBBAA`, son çifti opaklık olan sekiz rakam; `00` (saydam) ile `ff` (tam
  opak) arasındadır: `#ff000080` yarı saydam kırmızıdır.

| Ad        | Değer     |
| --------- | --------- |
| `black`   | `#000000` |
| `white`   | `#ffffff` |
| `red`     | `#ff0000` |
| `green`   | `#008000` |
| `blue`    | `#0000ff` |
| `yellow`  | `#ffff00` |
| `cyan`    | `#00ffff` |
| `magenta` | `#ff00ff` |
| `gray`    | `#808080` |

`"Red"`, `"purple"` ya da `"#fff"` gibi başka her metin `GraphicsError`
fırlatır; bir renk hiçbir zaman sessizce siyaha çevrilmez.

## Çizim

`canvas.line(x1, y1, x2, y2)` uçları yuvarlak düz bir çizgi çizer. `width` 0'dan
büyük olmalıdır. İki ucu aynı nokta olan bir çizgi, çizgi kalınlığında yuvarlak
bir noktadır.

`canvas.circle(x, y, radius)` `(x, y)` merkezli bir çember çizer. `stroke`
çevre çizgisinin rengi, `width` ise 0'dan büyük olması gereken kalınlığıdır.
`fill` içinin rengidir; varsayılan `null` içini boş bırakır. `radius` 0 ya da
daha büyük olmalıdır ve yarıçapı 0 olan bir çember hiçbir şey çizmez.

`canvas.rectangle(x, y, width, height)` sol alt köşesinden bir dikdörtgen
çizer. `width` ve `height` 0 ya da daha büyük olmalıdır ve bir kenarı sıfır olan
dikdörtgen hiçbir şey çizmez. `stroke`, `fill` ve `lineWidth` çemberin `stroke`,
`fill` ve `width` değerleri gibi çalışır.

`canvas.ellipse(x, y, radiusX, radiusY)` (v2.8.0'dan itibaren) `(x, y)`
çevresine yatay ve dikey yarıçapı olan bir elips çizer. İkisi de 0 ya da daha
büyük olmalıdır; yarıçaplarından biri sıfır olan elips hiçbir şey çizmez.
`stroke`, `fill` ve `width` çemberdeki gibi çalışır.

`canvas.polygon(points)` (v2.8.0'dan itibaren) köşelerinden geçen kapalı bir
şekil çizer. `points`, koordinatların tek bir düz listesidir:
`[x0, y0, x1, y1, ...]`; `Real` ya da `Int` listesi olabilir. Bir üçgen altı
sayıdır. Son köşe ilk köşeye bağlanır. Bir çokgenin 3 ile 2000 arasında köşesi
olmalıdır; tek sayıda sayı `GraphicsError` fırlatır. `stroke`, `fill` ve
`width` çemberdeki gibi çalışır. Beş köşeli bir yıldız gibi kendini kesen bir
çokgende çevrenin içine aldığı her bölge doldurulur.

`canvas.arc(x, y, radius, start, end)` (v2.8.0'dan itibaren) `(x, y)`
çevresindeki bir çemberin çevresinden bir parça çizer. `start` ve `end`,
matematikteki ve Turtle yönündeki gibi pozitif x ekseninden saat yönünün
tersine ölçülen derece cinsinden açılardır: 0 sağ, 90 yukarıdır. `end`,
`start`'tan büyükse yay saat yönünün tersine, küçükse saat yönünde ilerler ve
hiçbir zaman bir tam turu geçmez. Yayın uçları yuvarlaktır ve dolgusu yoktur.
`radius` 0 ya da daha büyük olmalıdır.

Bir şeklin hem dolgusu hem çevresi varsa önce dolgu, üzerine çevre çizilir.
Sonra çizilenler öncekilerin üstünü örter.

```ahd
bring Graphics

canvas := Graphics.open(400, 300)
canvas.text(0, 100, "Üçgen", "blue", 28, "center")
canvas.polygon([-100, -80, 100, -80, 0, 60], "black", "yellow", 2)
canvas.ellipse(0, -110, 120, 15, "gray")
canvas.arc(0, -80, 40, 0, 180, "red", 3)
canvas.save("ucgen.png")
canvas.wait()
```

`canvas.clear(color)` bütün çizimleri siler ve `color`'ı yeni arka plan yapar.
Pencereyi kapatmaz, hiçbir Turtle'ı hareket ettirmez ya da değiştirmez.

## Metin

`canvas.text(x, y, text)` (v2.8.0'dan itibaren) tek satır metin yazar. `(x, y)`
taban çizgisinin, yani harflerin üzerinde durduğu çizginin üstündedir: harfler
`y`'nin üstüne yükselir, yalnızca `g` ve `p` gibi harflerin kuyrukları altına
iner.

- `align`, metni `x` çevresinde yerleştirir: `"left"` (varsayılan) metni orada
  başlatır, `"center"` orada ortalar, `"right"` orada bitirir. Başka bir değer
  `GraphicsError` fırlatır.
- `size`, yazı tipinin Canvas birimi cinsinden yüksekliğidir; 0'dan büyük ve
  en çok 2000 olmalıdır.
- `color` herhangi bir Graphics rengidir.

Metin en çok 1000 karakterlik tek bir satırdır: satır sonu `GraphicsError`
fırlatır; ikinci satırı, boyutun yaklaşık 1,2 katı kadar aşağıya ikinci bir
çağrıyla yazın. Boş metin hiçbir şey çizmez.

Her Canvas aynı paketli yazı tipini (Go Regular) kullanır; böylece bir çizim her
bilgisayarda aynı görünür. Yazı tipi Türkçe harfler dahil Latin harflerini,
Yunan ve Kiril harflerini içerir; içermediği bir karakter boş bir kutu olarak
çizilir. SVG dosyasında metin, ana hattı olarak kaydedilir; dosyanın tam
görünmesi için yazı tipine gerek kalmaz ve özgün metin etiket olarak taşınır.

## Resimler

`canvas.image(path, x, y)` (v2.8.0'dan itibaren) bir PNG ya da JPEG dosyasını,
dikdörtgendeki gibi sol alt köşesi `(x, y)` olacak biçimde çizer. Göreli yol,
programın çalıştığı klasöre göredir.

- `width` ve `height` verilmezse resim kendi boyutunda, piksel başına bir
  Canvas birimiyle çizilir.
- Biri verilirse diğeri, resmin oranı korunacak biçimde hesaplanır.
- İkisi de verilirse resim tam o boyuta gerilir.

Bir dosya yalnızca ilk çizildiğinde okunur; onu yeniden çizmek, örneğin bir
animasyonun her karesinde, ek bir maliyet getirmez. Bir resim dosyası en çok
64 MB, bir kenarı en çok 8192 piksel ve toplamı en çok 32 milyon piksel
olabilir. Bulunmayan bir dosya, PNG ya
da JPEG olmayan bir dosya ya da negatif bir boyut `GraphicsError` fırlatır.
Saydamlığı olan bir PNG saydamlığını korur. SVG dosyasında resim dosyanın
içinde saklanır.

## Kaydetmek

`canvas.save(path)` Canvas'ı bir dosyaya yazar. Biçim, büyük/küçük harfe
bakılmaksızın uzantıdan anlaşılır:

- `.png` — pencere çerçevesi olmadan tam olarak `width` × `height` piksellik bir
  resim;
- `.svg` — çizgilerin, çemberlerin ve dikdörtgenlerin aynı koordinatlarda ayrı
  `line`, `circle` ve `rect` öğeleri olarak kaldığı bir vektör çizim.

Göreli bir yol, programın çalışma dizinine göredir. `.jpg` ya da `.pdf` gibi
başka her uzantı `GraphicsError` fırlatır. İki dosya da pencerenin gösterdiği
aynı çizim listesinden üretilir; bu yüzden pencere, PNG ve SVG birbiriyle
uyuşur. SVG betik ya da dış kaynak içermez.

Pencere kapanmadan önce kaydedin: kapanmış bir Canvas artık kaydedilemez.

## Turtle

Turtle, yönlendirdiğiniz bir kalemdir. `canvas.turtle()` yenisini başlangıç
noktasına, sağa bakacak şekilde (yön 0), kalemi aşağıda ve 1 kalınlığında siyah
çizgiler çizecek biçimde koyar. Her Turtle kendi konumunu, yönünü, kalemini,
rengini ve kalınlığını tutar ve yalnızca Canvas'ının `line`'ı üzerinden çizer.

Yönler, matematikteki açılar gibi ölçülen derecelerdir:

| Yön | Doğrultu |
| --- | -------- |
| 0   | sağ (+x) |
| 90  | yukarı (+y) |
| 180 | sol (-x) |
| 270 | aşağı (-y) |

- `left(a)` saat yönünün tersine döner: yön `a` kadar artar.
- `right(a)` saat yönünde döner: yön `a` kadar azalır.
- `setHeading(a)` hareket etmeden mutlak bir doğrultuya döner.
- `heading()` her zaman 0'dan 360'a kadar (360 hariç) bir değer bildirir:
  `left(450)` sonrasında 90'dır ve 90'dan `right(180)` 270 verir.

`forward(d)` yön boyunca `d` birim ilerler: yeni konum `x + d·cos(yön)`,
`y + d·sin(yön)` olur. `backward(d)` ters yönde ilerler. Negatif uzaklıklara ve
negatif açılara izin verilir: `forward(-20)`, `backward(20)`'dir; `left(-90)`
ise `right(90)`'dır.

`moveTo(x, y)` doğrudan bir noktaya gider. `home()` `(0, 0)`'a gider ve 0 yönüne
döner; Canvas'ı temizlemez ve kalemi, rengi ya da kalınlığı değiştirmez.

Kalem **aşağıdayken** her hareket (`forward`, `backward`, `moveTo`, `home`),
Turtle'ın bulunduğu yerden vardığı yere bir çizgi çizer. Kalem **yukarıdayken**
(`penUp()`) Turtle yalnızca hareket eder. `penDown()` çizmeyi yeniden başlatır.
`setColor` ve `setWidth` kendilerinden sonra çizilen çizgilere uygulanır;
kalınlık 0'dan büyük olmalıdır.

Turtle geometrisi pencereye, kare hızına ya da zamana bağlı değildir: aynı
komutlar her zaman aynı konumu verir. Dört eksen yönü tamdır; bu yüzden bir kare
tam olarak `(0, 0)`'da kapanır. Diğer açılar olağan `Real` aritmetiğini kullanır.

```ahd
bring Graphics

canvas := Graphics.open(500, 500)
turtle := canvas.turtle()
sides := 5
exteriorAngle := 360.0 / sides
count := 0
while count < sides {
    turtle.forward(150)
    turtle.left(exteriorAngle)
    count = count + 1
}
canvas.save("besgen.svg")
canvas.close()
```

Düzgün bir çokgenin dış açılarının toplamı 360 derece olduğundan Turtle,
`Real` aritmetiğinin yuvarlamaları dışında, başladığı yerde ve başladığı yöne
bakarak durur.

Görünür bir kaplumbağa simgesi yoktur; Turtle kalemin kendisidir.

## Pencere

`Graphics.open` gerçek bir pencere açar ve pencere hazır olduğunda döner; hiçbir
zaman olay döngüsü yazmazsınız. `width` ve `height` her biri 1 ile 4096
arasında olmalıdır, başlık en çok 256 karakterdir.

- `canvas.wait()` kullanıcı o pencereyi kapatana kadar bekler, sonra döner.
  Programın geri kalanı ardından devam eder.
- `canvas.close()` pencereyi programdan kapatır. Zaten kapanmış bir Canvas'ı
  kapatmak hiçbir şey yapmaz.
- `canvas.isOpen()`, Canvas'a çizilebildiği sürece `true`; `close()` sonrasında,
  `wait()` döndükten sonra ya da kullanıcı pencereyi kapattıktan sonra
  `false`'tur.

Kapanmış bir Canvas'ta çizim (`clear`, `line`, `circle`, `rectangle`, diğer
şekiller, `text`, `image`, `save` ve kalemi aşağıdaki bir Turtle hareketi) ile
`every` ve `batch` `GraphicsError` fırlatır. Kapanmış bir
Canvas kendiliğinden yeniden açılmaz; yerine yenisini açın. `wait()` döndükten
ya da kullanıcı pencereyi kapattıktan sonra kalemi yukarıdaki bir Turtle yine de
hareket edebilir ve dönebilir. `canvas.close()` Canvas'ı ve verdiği her Turtle'ı
serbest bırakır: ondan sonra bu Turtle'lara yapılan her çağrı, `x()`, `y()` ve
`heading()` dahil, `GraphicsError` fırlatır.

Her `Graphics.open` ayrı bir penceredir; aynı anda birkaç tane açık olabilir ve
birini kapatmak diğerlerini etkilemez. Program bittiğinde açık bıraktığı her
pencere kapanır. Bir pencereyi ekranda tutmak için `canvas.wait()` kullanın.

## Tıklamalar ve tuş basışları

v1.8.0'dan itibaren bir Canvas iki tür olayı bir callback'e
bildirir:

```text
canvas.onClick(handler: (x: Real, y: Real) -> Nothing)
canvas.onKey(handler: (key: String) -> Nothing)
```

- `onClick`, tıklanan noktayı Canvas'ın kendi Kartezyen koordinatlarında alır:
  `(0, 0)` merkezdir, `+x` sağa, `+y` yukarı. Yalnızca Canvas içindeki sol
  düğme basışı sayılır.
- `onKey`, Canvas penceresi odaktayken her tuş basışı için normalleştirilmiş
  bir tuş adı alır: `ArrowUp`, `ArrowDown`, `ArrowLeft`, `ArrowRight`,
  `Enter`, `Escape`, `Space`, `Tab`, `Backspace`, `Delete`, `Home`, `End`,
  `PageUp`, `PageDown`, `A`–`Z` harfleri ve `0`–`9` rakamları
  ([GUI](GUI_TR.md#tuş-adları) ile aynı adlar). Bir tuşu basılı tutmak onu
  yinelemez ve tuş bırakma olayı yoktur.

Callback'ler `canvas.wait()` içinde, birer birer ve sırayla, programın kendi
yürütme yolunda çalışır; bir callback çizebilir, Turtle taşıyabilir,
temizleyebilir veya kaydedebilir. `onClick` veya `onKey`'i yeniden kaydetmek
önceki callback'in yerine geçer. Uzun bir callback sonraki olayın işlenmesini
geciktirir. Bir callback hata fırlatırsa Canvas kapatılır ve hata `wait()`'ten
değişmeden yayılır. Pencere kapandığında hâlâ bekleyen olaylar atılır.
Callback kaydetmeyen bir program tam olarak eskisi gibi bekler.

```ahd
bring Graphics
from Graphics bring (Canvas, Turtle)

canvas: Canvas := Graphics.open(400, 400)
pen: Turtle := canvas.turtle()

step: Function := (key: String) -> Nothing {
    pen: Global Turtle
    state key {
        condition "ArrowUp" {
            pen.setHeading(90)
        }
        condition "ArrowDown" {
            pen.setHeading(270)
        }
        condition "ArrowLeft" {
            pen.setHeading(180)
        }
        condition "ArrowRight" {
            pen.setHeading(0)
        }
        condition default {
            return
        }
    }
    pen.forward(20)
}

canvas.onKey(step)
canvas.onClick(lambda [@pen] (x: Real, y: Real) -> pen.moveTo(x, y))
canvas.wait()
```

Her ok tuşu basışı tam olarak bir 20 birimlik çizgi çizer; terminalden girdi
okunmaz. Bkz. [`examples/v1.8/turtle_events`](../examples/v1.8/README_TR.md).
Girdi yoklama (`mouseX`, `isKeyDown`), tekerlek, çift tıklama ve `Turtle.speed`
hâlâ yoktur.

## İşaretçi hareketi ve sürükleme

v2.8.0'dan itibaren bir Canvas işaretçinin nereye gittiğini de bildirir:

```text
canvas.onMove(handler: (x: Real, y: Real) -> Nothing)
canvas.onDrag(handler: (x: Real, y: Real) -> Nothing)
```

- `onMove`, işaretçi hiçbir düğme basılı değilken Canvas üzerinde hareket
  ettiğinde konumunu alır.
- `onDrag`, işaretçi Canvas'ın içinde başlayan bir sol düğme basışından sonra
  hareket ettiğinde konumunu alır. Sürükleme, işaretçi dışarı çıksa da
  bildirilmeye devam eder; bu yüzden konum Canvas'ın dışında olabilir. Dışarıda
  başlayan bir basış, işaretçi içeri girince sürüklemeye dönüşmez. Basışın
  kendisi bir tıklamadır, sürükleme değildir.

İkisi de Canvas'ın Kartezyen koordinatlarını kullanır ve
[Tıklamalar ve tuş basışları](#tıklamalar-ve-tuş-basışları) bölümündeki bütün
kurallara uyar: `canvas.wait()` içinde, birer birer çalışırlar ve yeniden
kaydetmek önceki callback'in yerini alır. Bir konum, tick başına en çok bir kez
bildirilir ve pencere saniyede 60 kez tick atar. Sürüklemeyi başlatan basış bir tıklamadır; bu yüzden bir
çizginin başladığı yer `onClick`'tir. Hareket callback'i işaretçiden yavaşsa
henüz ulaşmadığı konumların yerini en yenisi alır; bir sürüklemenin noktaları
hiçbir zaman atılmaz.

```ahd
bring Graphics
from Graphics bring (Canvas)

canvas: Canvas := Graphics.open(600, 400)
lastX: Real := 0.0
lastY: Real := 0.0

start: Function := (x: Real, y: Real) -> Nothing uses [@lastX, @lastY] {
    lastX = x
    lastY = y
}

draw: Function := (x: Real, y: Real) -> Nothing uses [@canvas, @lastX, @lastY] {
    canvas.line(lastX, lastY, x, y, "blue", 3)
    lastX = x
    lastY = y
}

canvas.onClick(start)
canvas.onDrag(draw)
canvas.wait()
```

Bkz. [`examples/v2.8/free_drawing`](../examples/v2.8/README_TR.md).

## Zamanlayıcılar ve animasyon

v2.8.0'dan itibaren bir Canvas bir Function'ı tekrar tekrar çağırabilir:

```text
canvas.every(milliseconds: Int, handler: () -> Nothing) -> Int
canvas.cancel(timer: Int)
canvas.batch(handler: () -> Nothing)
```

`canvas.every(milliseconds, handler)` bir zamanlayıcı başlatır ve numarasını
döndürür. Program `canvas.wait()` içindeyken `handler` her `milliseconds`'ta
bir kez çalışır; 16, saniyede yaklaşık altmış kez demektir. Aralık 1 milisaniye
ile bir gün arasında olmalıdır. Zamanlayıcı callback'leri diğer bütün
callback'ler gibi çalışır: birer birer, tıklamalar ve tuşlarla sırayla,
programın kendi yürütme yolunda; bu yüzden kilitleme gerektirmezler. Aralığından
uzun süren bir callback yalnızca daha seyrek çalışır; kaçan turlar telafi
edilmez. `canvas.cancel(timer)` bir zamanlayıcıyı durdurur; kendi callback'inin
içinden de çağrılabilir ve çalışmayan bir zamanlayıcıyı iptal etmek hiçbir şey
yapmaz. Bir Canvas aynı anda en çok 64 zamanlayıcı çalıştırır; Canvas'ı
kapatmak hepsini durdurur.

Her zamanlayıcı callback'i bir karedir: çizdiği her şey, callback döndüğünde
birlikte görünür. Alışılmış animasyonu, yani temizleyip yeniden çizmeyi
titremesiz yapan budur:

```ahd
bring Graphics
from Graphics bring (Canvas)

canvas: Canvas := Graphics.open(400, 300)
x: Real := -180.0
speed: Real := 4.0

frame: Function := () -> Nothing uses [@canvas, @x, @speed] {
    x += speed
    if x > 180 or x < -180 {
        speed = -speed
    }
    canvas.clear("white")
    canvas.circle(x, 0, 20, "black", "red")
}

canvas.every(16, frame)
canvas.wait()
```

`canvas.batch(handler)` aynı güvenceyi başka her koda verir: `handler`
çalışırken pencere son tamamlanmış resmini göstermeyi sürdürür ve `handler`
döndüğünde sonucu bir kerede gösterir. Bir tıklama ya da tuş callback'inde çok
şekilden oluşan bir yeniden çizim için kullanın. Batch'ler iç içe olabilir;
içindeki bir hata da yayılmadan önce resmi serbest bırakır.

Temizlenen bir Canvas önceki çizimlerini unutur; bu yüzden her karede temizleyen
bir animasyon pencere açık kaldığı sürece çalışabilir. Bkz.
[`examples/v2.8/bouncing_ball`](../examples/v2.8/README_TR.md).

## Hatalar

Eksik bir argüman, `Real` gereken yerde `String`, null olamayan bir parametreye
`null` ya da var olmayan bir üye gibi derleyicinin görebildiği hatalar derleme
hatasıdır. Yalnızca çalışan programın görebildiği sorunlar `GraphicsError`
fırlatır:

- 1..4096 dışında bir Canvas boyutu;
- bilinmeyen bir renk;
- 0'dan büyük olmayan bir çizgi kalınlığı, çember çevre kalınlığı ya da Turtle
  kalınlığı;
- negatif bir yarıçap, genişlik ya da yükseklik;
- 3'ten az ya da 2000'den çok köşeli bir çokgen ya da tek sayıda sayı;
- satır sonu içeren, 1000 karakterden uzun, boyutu 0..2000 dışında ya da
  `align` değeri bilinmeyen bir metin;
- bulunmayan, fazla büyük ya da PNG ya da JPEG olmayan bir resim dosyası;
- 1 milisaniye..bir gün dışında bir zamanlayıcı aralığı ya da 65. zamanlayıcı;
- desteklenmeyen uzantılı bir kayıt yolu ya da yazılamayan bir dosya;
- kapanmış bir Canvas'a çizim ya da Canvas'ı `close()` ile kapatılmış bir
  Turtle'ı kullanmak;
- pencere yardımcısının bulunamaması, başlatılamaması ya da beklenmedik biçimde
  durması.

```ahd
bring Graphics
from Graphics bring (GraphicsError)

canvas := Graphics.open(200, 200)
attempt {
    canvas.circle(x: 0, y: 0, radius: 30, stroke: "orange")
}
except GraphicsError as error {
    write(error.message)
}
canvas.close()
```

## Nasıl çalışır

Açık her Canvas'ı, AhdCode'un onun için başlattığı ve standart girdi ile çıktısı
üzerinden konuştuğu küçük bir paketli program olan `ahdgraphics` çizer.
AhdCode her argümanı kendisi doğrular ve bütün Turtle geometrisini kendisi
hesaplar; `ahdgraphics` yalnızca çizim listesini tutar, pencerede gösterir ve
PNG ile SVG dosyalarını yazar. Kabuk komutu çalıştırmaz, `save` dışında dosya
yazmaz, programın çizdiği resimler dışında dosya okumaz ve ağ kullanmaz.

`ahdcode run`, yerel derlemeler ve REPL aynı gerçekleştirmeyi kullanır.
`ahdcode build` ile derlenen bir program, kurulu `ahdgraphics`'in yerini kaydeder;
onu başka bir bilgisayarda çalıştırmak için oraya da AhdCode kurun. Yardımcı
bulunamazsa `Graphics.open` `GraphicsError` fırlatır.

## Platform notları

- **macOS**: pencereler her uygulama penceresi gibi açılır. Yüksek yoğunluklu
  bir ekranda pencere ekranın çözünürlüğünde çizilir, böylece çizgiler keskin
  kalır; kaydedilen PNG dosyaları her zaman birim başına bir pikseldir. Hangi
  uygulamanın önde olacağına macOS karar verir: başka bir uygulama etkinken ya da
  Stage Manager açıkken yeni bir Canvas penceresi onun arkasında ya da Stage
  Manager şeridinde açılabilir. Öne getirmek için Dock'taki simgesine ya da
  küçük önizlemesine tıklayın.
- **Uygulama kimliği** (v1.9.0'dan itibaren): bir Canvas
  penceresi AhdCode adını ve simgesini gösterir — macOS'ta menü çubuğunda
  **AhdCode**'u ve Dock'ta AhdCode simgesini, Windows ve Linux'ta sistemin
  gösterdiği yerlerde pencere simgesini. Çizim, olaylar ve kaydedilen
  dosyalar değişmez.
- **Windows**: Windows 10 ya da daha yenisi gerekir.
- **Linux**: X11 ekranı olan (XWayland da sayılır) bir masaüstü oturumu ve
  sistemin OpenGL kitaplıkları gerekir. Ekran yoksa `Graphics.open`
  `GraphicsError` fırlatır.

`AHDCODE_GRAPHICS_HEADLESS=1`, otomatik testler ve ekranı olmayan sunucular için
her Canvas'ı pencere olmadan açar: çizim ve `save` her zamanki gibi çalışır;
kapatılacak pencere olmadığından `wait()` hemen döner. Penceresiz bir
Canvas'ın saati yoktur; bu yüzden zamanlayıcıları çalışmaz.

## Graphics'te olmayanlar

Graphics'te bilerek sprite, sahne, çarpışma, fizik, girdi yoklama (`mouseX`,
`isKeyDown`), tekerlek ya da çift tıklama olayları, ses, 3B, gölgelendirici,
arayüz bileşeni, seçilebilir yazı tipi, çok satırlı ya da döndürülmüş metin,
resim döndürme ya da kırpma ve `Turtle.speed` yoktur. Girdisi yukarıdaki
tıklama, tuş, hareket ve sürükleme callback'leri, tek saati de
`canvas.every`'dir.
