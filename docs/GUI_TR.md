# GUI standart modülü

[English](GUI.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Graphics](GRAPHICS_TR.md)

> v1.8.0 ile eklendi.

`GUI`, küçük bir bileşen kümesiyle gerçek masaüstü pencereleri açar: Column
ve Row içinde yerleşen bir Label, bir Button, tek satırlık bir TextInput ve
bir Checkbox. Bir Button tıklaması ve bir tuş basışı bir AhdCode Function'ını
çalıştırabilir.

AhdCode GUI; formlar, veri giriş araçları, basit veritabanı ön yüzleri,
yardımcı programlar ve API tabanlı uygulamalar gibi öğrenme ve küçük masaüstü
uygulamaları için tasarlanmıştır. Profesyonel video üretimi, 3B tasarım, oyun
motorları veya benzeri büyük yaratıcı uygulamaların gerektirdiği özel altyapıyı
sağlamayı amaçlamaz. Tkinter uyumlu bir araç takımı da değildir: yüzeyi
bilerek küçüktür ve yalnızca sıradan küçük programların ihtiyaç duyduğu yerde
büyür.

```ahd
bring GUI
from GUI bring (Window, Container, Label, Button, TextInput)

window: Window := GUI.window(title: "Greeter", width: 360, height: 180)
form: Container := window.column()
form.label("Your name")
name: TextInput := form.textInput(placeholder: "e.g. Ayşe")
greet: Button := form.button("Greet")
answer: Label := form.label("")

sayHello: Function := () -> Nothing {
    name: Global TextInput
    answer: Global Label
    answer.setText("Hello, {name.text()}!")
}

greet.onClick(sayHello)
window.wait()
```

GUI yalnızca kullanıcı etkileşimiyle ilgilenir. Bir TextInput String, bir
Checkbox Bool verir ve bir Button bir Function çalıştırır; o Function'ın bu
değerlerle yaptığı iş — [SQLite](SQLITE_TR.md) ile saklamak, dosya yazmak,
[HTTP](HTTP_TR.md) çağırmak, [Statistics](STATISTICS_TR.md) ile hesaplamak —
sıradan AhdCode'dur. GUI veritabanlarını, dosyaları veya Excel'i bilmez.

## Yüzey

```text
GUI.window(title: String := "AhdCode", width: Int := 800, height: Int := 600) -> Window

Window.column(spacing: Int := 8, padding: Int := 12) -> Container
Window.row(spacing: Int := 8, padding: Int := 12) -> Container
Window.onKey(handler: (key: String) -> Nothing) -> Nothing
Window.wait() -> Nothing
Window.close() -> Nothing
Window.isOpen() -> Bool
Window.setTitle(text: String) -> Nothing

Container.column(spacing: Int := 8, padding: Int := 0) -> Container
Container.row(spacing: Int := 8, padding: Int := 0) -> Container
Container.label(text: String) -> Label
Container.button(text: String) -> Button
Container.textInput(placeholder: String := "") -> TextInput
Container.checkbox(text: String, checked: Bool := false) -> Checkbox

Label.text() -> String              Label.setText(text: String) -> Nothing
Button.text() -> String             Button.setText(text: String) -> Nothing
Button.onClick(handler: () -> Nothing) -> Nothing
TextInput.text() -> String          TextInput.setText(text: String) -> Nothing
Checkbox.checked() -> Bool          Checkbox.setChecked(checked: Bool) -> Nothing
```

`GUIError`, `Error`'dan türer. Sınıfların hiçbirinin kurucusu yoktur: Window
`GUI.window`'dan, Container `column` veya `row`'dan ve her bileşen bir
Container'dan gelir. Her çağrı ya tamamen konumsal ya da tamamen
adlandırılmıştır.

## Pencereler

`GUI.window` pencereyi hemen açar ve döndürür; yazılacak bir ana döngü
yoktur. Genişlik ve yükseklik 1 ile 4096 arasında olmalıdır; başlık en fazla
256 karakterlik herhangi bir metindir. Geçersiz boyut `GUIError`'dır, hiçbir
zaman sessizce sınırlanmaz. v1.8'de pencere yeniden boyutlandırılamaz.

- `wait()`, kullanıcı kapatana kadar pencereyi gösterir, bu arada
  callback'leri çalıştırır (aşağıya bakın) ve ardından normal şekilde döner.
  Programın `wait()` öncesinde yazdığı her şey, pencere beklemeye başlamadan
  terminalde görünür; her callback'in çıktısı da callback döner dönmez
  görünür.
- `close()` pencereyi programdan kapatır. İki kez kapatmak bir şey yapmaz.
- `isOpen()`, `close()` sonrasında ve kullanıcı pencereyi kapattıktan sonra
  false'tur.
- `setTitle(text)` başlık çubuğunu değiştirir.

Aynı anda birden çok Window açık olabilir; her biri bağımsızdır ve birini
kapatmak diğerlerini açık bırakır. Programın açık bıraktığı her pencere
program bitince kapanır.

Bir Window kapandıktan sonra TextInput'un `text()` ve Checkbox'ın `checked()`
çağrıları son değerleri döndürmeye devam eder; böylece program `wait()`
döndükten sonra formu okuyabilir. Kapalı bir Window'da bir bileşeni
değiştirmek, bileşen eklemek veya callback kaydetmek `GUIError`'dır.

## Yerleşim: tek kök, Column ve Row

Bir Window'un en fazla bir kök Container'ı vardır: ilk `window.column()` veya
`window.row()` onu oluşturur; ikisinden birini yeniden çağırmak `GUIError`'dır.
Boş bir Window geçerlidir. Diğer her Container bir Container'ın içinde
oluşturulur.

- **Column** çocuklarını yukarıdan aşağıya, **Row** soldan sağa yerleştirir.
- `spacing` komşu çocuklar arasındaki boşluk, `padding` Container kenarlarının
  içindeki kenar boşluğudur; ikisi de pencere noktası cinsindendir. İkisi de 0
  ile 1000 arasında olmalıdır; negatif değer `GUIError`'dır.
- Her bileşen doğal boyutunu korur: Label metni kadar geniştir, Button metni
  artı bir kenar boşluğu kadar (en az 64 nokta), TextInput 260 nokta
  genişliğindedir ve her bileşen 32 nokta yüksekliğindedir.
- Column'un çocukları sol kenarına hizalanır; Row'un çocukları dikey olarak
  ortalanır.
- Kök Container pencerenin sol üst köşesinden başlar. Pencereye sığmayan kısım
  kırpılır; forma uyan bir pencere boyutu seçin.

Izgara, mutlak konumlandırma, hizalama seçeneği, esnetme veya kaydırma yoktur.
Column ve Row ile formlar kolayca kurulur; her masaüstü yerleşimi bilerek
ifade edilemez.

```ahd
bring GUI
from GUI bring (Window, Container)

window: Window := GUI.window(title: "Layout", width: 420, height: 200)
form: Container := window.column(spacing: 10)
form.label("Name")
form.textInput(placeholder: "name")
buttons: Container := form.row(spacing: 12, padding: 0)
buttons.button("Save")
buttons.button("Cancel")
window.wait()
```

## Bileşenler

- **Label** tek satırlık metin gösterir; `setText` onu değiştirir.
- **Button** tek satırlık metin gösterir ve tıklama callback'ini çalıştırır.
  Tıklama, sol fare düğmesinin aynı Button üzerinde basılıp bırakılmasıdır;
  Enter veya Space odaktaki Button'ı etkinleştirir.
- **TextInput** tek satırlık bir metin alanıdır. Kullanıcı yazar, imleci ok
  tuşları, Home ve End ile taşır ve Backspace ile Delete ile siler. Yer tutucu
  (placeholder) yalnızca alan boş ve odak dışındayken gösterilir ve `text()`
  onu hiçbir zaman döndürmez. `setText` metnin tamamını değiştirir.
- **Checkbox** bir kutu ve bir başlık gösterir; bir tıklama veya odaktayken
  Space onu değiştirir. `checked()` onu okur, `setChecked` ayarlar.

Bir Button, TextInput veya Checkbox'a tıklamak onu odaklar; Tab odağı
oluşturulma sırasına göre sonrakine taşır. Metinler en fazla 4096
karakterdir. Metin AhdCode ile gelen Go Regular yazı tipiyle çizilir; bu
yüzden her bilgisayarda aynı görünür. v1.8'de yazı tipi, renk veya tema
seçeneği yoktur.

## Callback'ler

```text
Button.onClick(handler: () -> Nothing)
Window.onKey(handler: (key: String) -> Nothing)
```

İşleyicinin biçimi program derlenirken denetlenir: `onClick` Function'ı
argüman almaz, `onKey` Function'ı bir String alır ve ikisi de Nothing
döndürür. Uyuşmazlık çalışma zamanı hatası değil derleme hatasıdır.

- Her olayın en fazla bir callback'i vardır. `onClick` veya `onKey`'i yeniden
  kaydetmek önceki callback'in **yerine geçer**. v1.8'de bir callback'i
  kaldırmanın yolu yoktur.
- Callback'ler `wait()` sırasında, birer birer, olayların gerçekleştiği
  sırayla, programın kendi yürütme yolunda çalışır. İki callback hiçbir zaman
  aynı anda çalışmaz ve yönetilecek bir iş parçacığı yoktur.
- Bir callback herhangi bir bileşeni okuyup değiştirebilir, başka bir Window
  açabilir veya herhangi bir modülü çağırabilir.
- **Uzun bir callback pencereyi bekletir.** Bir callback çalışırken pencere
  tepki vermez; bu arada gerçekleşen olaylar sonra, sırayla işlenir. HTTP
  yanıtı veya büyük bir sorgu bekleyen bir callback, dönene kadar pencereyi
  dondurur. Bu, basit sıralı modelin bedelidir.
- Bir callback hata fırlatırsa Window kapatılır ve hata `wait()`'ten
  değişmeden yayılır: bir `ValueError` `ValueError` olarak kalır ve `wait()`
  çevresinde `except ValueError` ile yakalanabilir.
- Pencere kapandığında hâlâ bekleyen olaylar atılır.

## Tuş adları

`Window.onKey` her tuş basışını bir kez bildirir (yalnızca basma; tuşu basılı
tutmak onu yinelemez). Tuş adları şunlardır:

```text
ArrowUp ArrowDown ArrowLeft ArrowRight Enter Escape Space Tab
Backspace Delete Home End PageUp PageDown
A B C ... Z        0 1 ... 9
```

Harfler ve rakamlar, etkin klavye düzeni ne olursa olsun tuşun ABD klavyesi
etiketiyle adlandırılır ve bir harf Shift ile veya Shift olmadan büyük harfle
bildirilir. Diğer tuşlar bildirilmez. Tuş bırakma olayı, değiştirici tuş API'si
veya tuş yineleme ayarı yoktur.

`Window.onKey`, odaktaki bir TextInput'a yazılan tuşları da alır: bir alana
"a" yazmak `onKey`'e "A" bildirir ve TextInput yine karakteri alır. `onKey`'de
harflere tepki veren bir program bunu hesaba katmalıdır; Escape, ok tuşları
ve Enter olağan seçimlerdir.

## Hatalar

`GUIError` GUI sorunları için fırlatılır: geçersiz pencere boyutu, spacing
veya padding, ikinci bir kök Container, kapalı bir Window'da değişiklik, eksik
veya başarısız `ahdgui` yardımcısı ve bozuk yardımcı yanıtı. Bir callback'in
fırlattığı hata için hiçbir zaman kullanılmaz.

## Nasıl çalışır

Her açık Window, diğer AhdCode yardımcılarının yanına kurulan paketli
`ahdgui` yardımcısının bir süreci tarafından çizilir. Program ile yardımcı,
yardımcının standart girdisi ve çıktısı üzerinden sınırlı JSON mesajları
alışverişi yapar; yardımcı tıklama, tuş ve kapanma olaylarını geri gönderir.
Yardımcı kabuk komutu çalıştırmaz, ağ kullanmaz, dosya veya ortam dosyası
okumaz ve kullanıcının yazdığı hiçbir şeyi günlüğe yazmaz. Graphics
yardımcısıyla aynı pencere kitaplığıyla (Ebitengine v2.10.2) derlenir ve Go
Regular yazı tipini içerir; bkz.
[THIRD_PARTY_NOTICES_GUI.md](../THIRD_PARTY_NOTICES_GUI.md).

`ahdcode run`, derlenmiş programlar ve REPL aynı gerçeklemeyi kullanır.
Derlenmiş bir program yardımcıyı kendisini derleyen kurulum üzerinden veya bir
AhdCode kurulumunda kendi yanında bulur; `PATH` hiçbir zaman aranmaz.
`AHDCODE_GUI_RUNTIME`, geliştirme ve testler için yardımcıyı açıkça belirtir.

## Platform notları

- macOS: bu sürümün geliştirme Mac'inde canlı olarak test edildi. Bir Window
  odaktayken menü çubuğu yardımcının adını, `ahdgui`, gösterir.
- Windows ve Linux: yardımcı her ikisi için de derlenir; Linux masaüstü bir
  X11 ekranı (XWayland da sayılır) ve sistemin OpenGL kitaplıklarını gerektirir.
- Otomatik testler: `AHDCODE_GUI_HEADLESS=1` her Window'u ekran olmadan açar.

## v1.8'de olmayanlar

GUI v1.8'de tablo veya ızgara bileşeni, liste kutusu, açılır liste, radyo
düğmeleri, sekmeler, ağaç, menüler, araç çubukları, durum çubukları, çok
satırlı metin, parola alanları, resim veya simgeler, pano, sürükle-bırak,
dosya veya klasör seçiciler, iletişim kutuları, kaydırma, yeniden
boyutlandırılabilir pencereler, temalar, stiller, yazı tipleri, animasyonlar,
Window içinde çizim, `TextInput.onChange`, `Checkbox.onChange`, arka plan
görevleri, iş parçacıkları veya uygulama paketleme yoktur. Bunların bazıları
sonraki sürümlerde gelebilir; programları masaüstü uygulaması olarak paketlemek
ayrıca planlanmaktadır.
