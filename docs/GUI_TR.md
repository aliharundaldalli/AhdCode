# GUI standart modülü

[English](GUI.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Graphics](GRAPHICS_TR.md)

> v1.8.0 ile eklendi. v2.0.0 ile tamamlandı.

`GUI`, küçük bir masaüstü uygulamasının sıradan denetimleriyle gerçek
masaüstü pencereleri açar: Column ve Row içinde yerleşen Label, Button, tek
ve çok satırlık metin, parola alanı, Checkbox, açılır Select, ListBox ve
TableView; ayrıca dosya, klasör, kaydetme, mesaj ve onay iletişim kutuları.
Tıklamalar, tuş basışları, düzenlemeler ve seçimler bir AhdCode Function'ını
çalıştırabilir.

v2.0 ile AhdCode'un birinci taraf GUI'si küçük formlar, veri giriş araçları,
veritabanı ön yüzleri ve dosya odaklı masaüstü yardımcıları için gereken
sıradan denetimleri kapsar. Sonraki sürümler gerçek bir kullanım gerektirdiğinde
odaklı bileşenler ekleyebilir; ancak GUI artık etkin bir temel yol haritası
değildir.

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

v1.9.0 ile eklendi (bkz. [Renkler](#renkler) ve
[Etkinlik durumu](#etkinlik-durumu)):

```text
Window.setBackground(color: String) -> Nothing
Container.setBackground(color: String) -> Nothing
Label.setForeground(color: String)      Label.setBackground(color: String)
Button.setForeground(color: String)     Button.setBackground(color: String)
TextInput.setForeground(color: String)  TextInput.setBackground(color: String)
Checkbox.setForeground(color: String)   Checkbox.setBackground(color: String)
Button.setEnabled(enabled: Bool)        Button.isEnabled() -> Bool
TextInput.setEnabled(enabled: Bool)     TextInput.isEnabled() -> Bool
Checkbox.setEnabled(enabled: Bool)      Checkbox.isEnabled() -> Bool
```

v2.0.0 ile eklenenler:

```text
Window.setResizable(resizable: Bool) -> Nothing
Window.isResizable() -> Bool

Container.passwordInput(placeholder: String := "") -> PasswordInput
Container.textArea(placeholder: String := "") -> TextArea
Container.select(items: List<String>, selectedIndex: Int? := null) -> Select
Container.listBox(items: List<String> := []) -> ListBox
Container.table(columns: List<String>, rows: List<List<String>> := []) -> TableView

TextInput.onChange(handler: (text: String) -> Nothing) -> Nothing
Checkbox.onChange(handler: (checked: Bool) -> Nothing) -> Nothing

PasswordInput.text() -> String      PasswordInput.setText(text: String) -> Nothing
PasswordInput.onChange(handler: (text: String) -> Nothing) -> Nothing
TextArea.text() -> String           TextArea.setText(text: String) -> Nothing
TextArea.onChange(handler: (text: String) -> Nothing) -> Nothing

Select.items() -> List<String>      Select.setItems(items: List<String>) -> Nothing
Select.selectedIndex() -> Int?      Select.selectedText() -> String?
Select.select(index: Int?) -> Nothing
Select.onChange(handler: (index: Int?, text: String?) -> Nothing) -> Nothing
ListBox.items() -> List<String>     ListBox.setItems(items: List<String>) -> Nothing
ListBox.selectedIndex() -> Int?     ListBox.selectedText() -> String?
ListBox.select(index: Int?) -> Nothing
ListBox.onChange(handler: (index: Int?, text: String?) -> Nothing) -> Nothing

TableView.columns() -> List<String>
TableView.rows() -> List<List<String>>
TableView.setRows(rows: List<List<String>>) -> Nothing
TableView.selectedRow() -> Int?     TableView.selectRow(index: Int?) -> Nothing
TableView.onSelect(handler: (row: Int?) -> Nothing) -> Nothing

GUI.openFile(title: String := "Open File", extensions: List<String> := []) -> String?
GUI.openFiles(title: String := "Open Files", extensions: List<String> := []) -> List<String>
GUI.selectFolder(title: String := "Select Folder") -> String?
GUI.saveFile(title: String := "Save File", suggestedName: String := "", extensions: List<String> := []) -> String?
GUI.message(title: String, text: String) -> Nothing
GUI.confirm(title: String, text: String) -> Bool
```

PasswordInput, TextArea, Select, ListBox ve TableView'da da v1.9 bileşenleri
gibi `setForeground`, `setBackground`, `setEnabled` ve `isEnabled` vardır.

`GUIError`, `Error`'dan türer. Sınıfların hiçbirinin kurucusu yoktur: Window
`GUI.window`'dan, Container `column` veya `row`'dan ve her bileşen bir
Container'dan gelir. Her çağrı ya tamamen konumsal ya da tamamen
adlandırılmıştır.

## Pencereler

`GUI.window` pencereyi hemen açar ve döndürür; yazılacak bir ana döngü
yoktur. Genişlik ve yükseklik 1 ile 4096 arasında olmalıdır; başlık en fazla
256 karakterlik herhangi bir metindir. Geçersiz boyut `GUIError`'dır, hiçbir
zaman sessizce sınırlanmaz. Program pencereyi yeniden boyutlandırılabilir
yapmadıkça boyutu sabittir (bkz. [Yeniden boyutlandırılabilir
pencereler](#yeniden-boyutlandırılabilir-pencereler)).

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

Bir Window kapandıktan sonra her metin alanının `text()`, Checkbox'ın
`checked()` ve her seçim çağrısı son değerleri döndürmeye devam eder; böylece program `wait()`
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
- Her bileşenin doğal bir boyutu vardır: Label metni kadar geniştir, Button
  metni artı bir kenar boşluğu kadar (en az 64 nokta), TextInput ve
  PasswordInput 260 nokta genişliğindedir ve her tek satırlık bileşen 32
  nokta yüksekliğindedir. TextArea 360 × 136, ListBox 260 × 146 noktadır;
  Select en uzun öğesi kadar (200 ile 480 nokta), TableView sütunları kadar
  (200 ile 640 nokta) geniştir ve 222 nokta yüksekliğindedir.
- Column'un çocukları sol kenarına hizalanır; Row'un çocukları dikey olarak
  ortalanır.
- Kök Container pencerenin sol üst köşesinden başlar. Pencereye sığmayan kısım
  kırpılır; forma uyan bir pencere boyutu seçin.

Kaydırmalı bileşenler — TextArea, ListBox ve TableView — ve bunları içeren
her Container **büyür**. Pencere içeriğinden büyük olduğunda (kullanıcının
büyüttüğü yeniden boyutlandırılabilir bir pencere veya büyük açılmış bir
pencere), Column'un fazladan yüksekliği büyüyen çocukları arasında eşit
paylaşılır; Row'un fazladan genişliği de öyle. Büyüyen bir çocuk ayrıca
Column'unun genişliğini (Row'unun yüksekliğini) doldurur. Label, Button,
metin alanları, Checkbox ve Select hiçbir zaman esnemez; hiçbir şey doğal
boyutunun altına küçülmez. Izgara, mutlak konumlandırma, hizalama veya büyüme
ayarı yoktur: bu tek kural bir tablonun veya listenin büyük bir pencerenin
alanını kullanmasını sağlar.

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

- **PasswordInput** (v2.0), her karakter için bir nokta gösteren bir
  TextInput'tur. `text()` yazılanı döndürür; yardımcı onu hiçbir zaman
  çizmez veya günlüğe yazmaz.
- **TextArea** (v2.0) çok satırlı metni düzenler. Enter yeni satır başlatır;
  ok tuşları, Home ve End (satırın), PageUp ve PageDown, Backspace ve Delete
  alışıldığı gibi çalışır, bir tıklama imleci yerleştirir. Uzun metin
  kaydırma çubuğuyla dikey, uzun bir satır imleçle birlikte yatay kayar;
  satır kaydırma, zengin metin veya sözdizimi renklendirmesi yoktur. Bir
  TextArea en fazla 100.000 karakter tutar; `setText`, `"\r\n"` ve `"\r"`
  dizilerini `"\n"` yapar.
- **Select** (v2.0), String'ler arasında açılır bir seçimdir. Tıklama, Enter
  veya Space listesini açar; tıklama veya Enter bir seçeneği seçer, Escape
  kapatır. Kapalıyken ok tuşları seçimi doğrudan değiştirir. Düzenlenemez.
  `container.select(...)` içindeki `selectedIndex` bir seçeneği önceden seçer.
- **ListBox** (v2.0), en fazla bir öğesi seçili bir String listesi gösterir.
  Tıklama bir öğe seçer; ok tuşları, Home, End, PageUp ve PageDown seçimi
  taşır. Uzun bir liste fare tekerleği veya kaydırma çubuğuyla kayar.
- **TableView** (v2.0) — bkz. [TableView](#tableview).

Etkileşimli bir bileşene tıklamak onu odaklar; Tab odağı oluşturulma sırasına
göre sonraki etkin bileşene, Shift+Tab öncekine taşır. Metinler en fazla 4096
karakterdir (TextArea'da 100.000). Metin AhdCode ile gelen Go Regular yazı
tipiyle çizilir, bu yüzden her bilgisayarda aynı görünür; her ekran ölçeğine
göre ölçüldüğü için Türkçe ve diğer harfler hiçbir zaman kırpılmaz. Yazı tipi
veya tema seçeneği yoktur.

## Seçimler

Select, ListBox veya TableView'da en fazla bir seçili girdi vardır; indeksiyle
tanımlanır ya da hiç yoktur, bu da `null`'dır:

- `selectedIndex()` (TableView için `selectedRow()`) indeksi veya `null`
  döndürür; `selectedText()` seçili öğenin metnini veya `null`.
- `select(index)` (`selectRow`) bir girdiyi seçer; `select(null)` seçimi
  temizler. Girdilerin dışındaki bir indeks `GUIError`'dır.
- `setItems` (`setRows`) girdileri değiştirir. Hâlâ var olan bir seçim
  korunur; artık var olmayan temizlenir.
- Değişiklik callback'ini yalnızca kullanıcının kendi eylemleri çalıştırır.
  `select`, `setItems`, `setRows`, `setText` ve `setChecked` hiçbir zaman
  çalıştırmaz; böylece bir callback kendini çağırmadan bileşenleri
  güncelleyebilir.

ListBox veya Select en fazla 1024 karakterlik 20.000 öğe tutar.

## TableView

`container.table(columns, rows)` bir başlık altında String satırları gösterir.
Bileşenin adı `TableView`'dur; `Table`, [Data](DATA_TR.md) modülünün tablosu
olarak kalır.

```ahd
bring GUI
bring Data
from GUI bring (Window, Container, TableView)

sales := Data.fromRows(["Month", "Amount"], [["Jan", "120"], ["Feb", "95"]])
// Table satırları kayıttır; TableView düz hücre satırları alır.
cells: List<List<String>> := []
for record in sales.rows() {
    row: Local List<String> := []
    for column in sales.columns() {
        row.add(record[column])
    }
    cells.add(row)
}
window: Window := GUI.window(title: "Sales", width: 480, height: 320)
window.setResizable(true)
form: Container := window.column()
table: TableView := form.table(sales.columns(), cells)
table.onSelect(lambda (row: Int?) -> write("row {row}"))
window.wait()
```

- En az bir sütun olmalı ve her satırda sütun başına tam bir hücre
  bulunmalıdır; aksi `GUIError`'dır. Sütun adları tekrarlanabilir: yalnızca
  etikettirler. Hücreler String'dir — sayıları ve tarihleri kendiniz
  biçimlendirin — ve `null` hücre değildir.
- Sütun genişlikleri başlık ve ilk 1000 satırla belirlenir: her biri en geniş
  metin artı bir kenar boşluğudur, 48 ile 320 nokta arasında. Daha geniş bir
  metin sütununda kırpılır.
- Tablo fare tekerleği, kaydırma çubuğu ve klavyeyle dikey; yana kaydırma
  veya Shift ile tekerlekle yatay kayar. Yalnızca görünen satırlar çizilir;
  bu yüzden on binlerce satır hızlı kalır.
- Tıklama bir satır seçer; tablo odaktayken ok tuşları, Home, End, PageUp ve
  PageDown seçimi taşır. Hücreler düzenlenemez: veriyi değiştirip `setRows`
  çağırın.
- `columns()` ve `rows()` kopya döndürür. Bir TableView en fazla 64 sütun,
  20.000 satır ve 1024 karakterlik 200.000 hücre tutar.

Bir [Data](DATA_TR.md) Table'ının `rows()` değeri kayıtlardır
(`List<Pair<String, String>>`); bu yüzden program onları yukarıdaki gibi
açıkça hücre satırlarına çevirir. GUI Data'ya bağlı değildir ve hiçbir şey
otomatik bağlanmaz.

## Callback'ler

```text
Button.onClick(handler: () -> Nothing)
Window.onKey(handler: (key: String) -> Nothing)
TextInput.onChange, PasswordInput.onChange, TextArea.onChange(handler: (text: String) -> Nothing)
Checkbox.onChange(handler: (checked: Bool) -> Nothing)
Select.onChange, ListBox.onChange(handler: (index: Int?, text: String?) -> Nothing)
TableView.onSelect(handler: (row: Int?) -> Nothing)
```

İşleyicinin biçimi program derlenirken denetlenir: `onClick` Function'ı
argüman almaz, `onKey` Function'ı bir String, metin `onChange`'i bir String,
Checkbox `onChange`'i bir Bool alır ve hepsi Nothing döndürür. Seçim
callback'i parametrelerini boş bırakılabilir bildirmelidir —
`lambda (index: Int?, text: String?) -> ...` — çünkü seçim boş olabilir;
uyuşmazlık çalışma zamanı hatası değil derleme hatasıdır.

- Metin alanının `onChange`'i kullanıcının her düzenlemesinde (yazma,
  Backspace, Delete, TextArea'da Enter) yeni metnin tamamıyla bir kez
  çalışır. Checkbox'ınki kullanıcı onu değiştirdiğinde; Select, ListBox veya
  TableView'unki kullanıcı seçimi değiştirdiğinde çalışır.
- Her olayın en fazla bir callback'i vardır. Bir callback'i yeniden kaydetmek
  öncekinin **yerine geçer**. Bir callback'i kaldırmanın yolu yoktur.
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

## Renkler

> v1.9.0 ile eklendi.

Renkler açıktır, bir stil çatısı değildir: Window ve Container'ın bir arka
planı vardır; Label, Button, TextInput ve Checkbox'ın bir ön planı (metni) ve
bir arka planı vardır.

```ahd
bring GUI
from GUI bring (Window, Container, Label, Button)

window: Window := GUI.window(title: "Renkler", width: 360, height: 160)
window.setBackground("#EEF2F7")
form: Container := window.column(spacing: 10, padding: 16)
title: Label := form.label("Yeni sipariş")
title.setForeground("#1F4E79")
save: Button := form.button("Kaydet")
save.setForeground("white")
save.setBackground("#2E7D32")
window.wait()
```

Bir renk tam olarak [Graphics](GRAPHICS_TR.md)'teki gibi yazılır: dokuz küçük
harfli addan biri (`black`, `white`, `red`, `green`, `blue`, `yellow`,
`cyan`, `magenta`, `gray`), `#RRGGBB` veya `#RRGGBBAA` (onaltılık rakamlar
büyük ya da küçük harf olabilir). Başka her şey — `"Red"`, `"purple"`,
`"#fff"`, boş metin — `GUIError`'dır; yerine başka bir renk konmaz. Bir rengi
yeniden ayarlamak öncekinin yerine geçer.

- Renk ayarlanmayan her şey v1.8'deki gibi görünür.
- Button'ın kendi arka planı gri dolgusunun yerine geçer; basılı tutulurken
  %15 daha koyu çizilir. TextInput'un arka planı beyaz alanının yerine geçer.
  Label ve Checkbox'ın arka planı tüm satır kutusunu doldurur; Checkbox'ın
  karesi kendi görünümünü korur.
- Boş bir TextInput'un yer tutucusu gri kalır, odak çerçeveleri mavi kalır.
- `#RRGGBBAA` renkleri olağan "source over" birleştirmesiyle karışır:
  pencere varsayılan açık gri renginde opak başlar, Window'un arka planı
  bunun üzerine boyanır ve her Container ile bileşen, oluşturulma sırasıyla
  üst öğesinin üzerine boyanır. Pencerenin kendisi hiçbir zaman saydam
  değildir.

Kapalı bir Window'da renk değiştirmek `GUIError`'dır.

## Etkinlik durumu

> v1.9.0 ile eklendi; v2.0 bileşenleri dahil.

Bir Button, TextInput, Checkbox, PasswordInput, TextArea, Select, ListBox
veya TableView devre dışı bırakılabilir. Her bileşen etkin başlar; Label ve
Container'ların etkinlik durumu yoktur.

- Devre dışı bir Button tıklanamaz ve Enter veya Space ile etkinleştirilemez;
  bu yüzden `onClick` callback'i çalışmaz.
- Devre dışı bir TextInput yazmayı ve düzenleme tuşlarını yok sayar.
- Devre dışı bir Checkbox tıklama veya Space ile değiştirilemez.
- Devre dışı bir Select, ListBox veya TableView kullanıcı tarafından
  açılamaz, seçilemez veya kaydırılamaz.
- Devre dışı bir bileşen odak alamaz: ona tıklamak odağı hiçbir yere
  taşımaz, Tab onu atlar ve odaktayken devre dışı bırakılan bir bileşen
  odağı kaybeder.
- Program devre dışı bir bileşeni yine değiştirebilir: `setText` ve
  `setChecked` çalışır, `text()` ve `checked()` onu okur.
- Devre dışı bir bileşen normal çizilir ve ardından pencerenin varsayılan
  rengine doğru soluklaştırılır.

`isEnabled()` durumu bildirir, Window kapandıktan sonra da; kapalı bir
Window'da `setEnabled` bir `GUIError`'dır. Görünürlük, gizleme veya gösterme
API'si yoktur.

## Yeniden boyutlandırılabilir pencereler

> v2.0.0 ile eklendi.

`window.setResizable(true)` kullanıcının pencereyi yeniden boyutlandırmasına
izin verir, `setResizable(false)` bunu durdurur; `isResizable()` ayarı
bildirir. Program istemedikçe pencereler v1.9'daki gibi yeniden
boyutlandırılamaz. Yeniden boyutlandırılabilir bir pencere en az 160 × 120
noktadır. Büyüdüğünde kaydırmalı bileşenler ve Container'ları
[Yerleşim](#yerleşim-tek-kök-column-ve-row) bölümündeki gibi fazladan alanı
alır; içeriğin altına küçüldüğünde içerik önceki gibi kırpılır.

## Dosya ve mesaj iletişim kutuları

> v2.0.0 ile eklendi.

```ahd
bring GUI
bring CSV

path: String? := GUI.openFile(title: "Open prices", extensions: ["csv"])
if path != null {
    rows: Local := CSV.read(path)
    write("{len(rows)} rows")
}
target: String? := GUI.saveFile(suggestedName: "report.csv", extensions: ["csv"])
if target != null and GUI.confirm("Export", "Write the report to {target}?") {
    CSV.write(target, [["name"], ["Ayşe"]])
    GUI.message("Export", "Saved.")
}
```

- `openFile` seçilen dosyayı, `openFiles` seçilen her dosyayı,
  `selectFolder` seçilen klasörü ve `saveFile` kaydedilecek yolu döndürür.
  Her yol mutlaktır ve sistemin kendi biçimindedir.
- **İptal etmek hata değildir.** İptal edilen `openFile`, `selectFolder` veya
  `saveFile` `null`, iptal edilen `openFiles` boş bir List döndürür.
  `GUIError`, iletişim kutusunun hiç gösterilemediği anlamına gelir.
- İletişim kutusu yalnızca bir yol seçer: hiçbir dosyayı açmaz, oluşturmaz
  veya değiştirmez. `saveFile` hiçbir şey oluşturmaz; okuma ve yazma
  programın kendi, ayrı adımlarıdır.
- `extensions` yalnızca bu uzantılara sahip dosyaları sunar, örneğin
  `["png", "jpg"]`: yalnızca harf ve rakam (baştaki nokta yok sayılır);
  `"*.png"` gibi başka her şey `GUIError`'dır. Boş List her dosyayı sunar.
  Sunulan bir uzantısı olmayan kaydedilmiş ada ilk uzantı eklenir.
- `suggestedName` klasörsüz bir dosya adıdır.
- `message` bir metni OK düğmesiyle gösterir ve kapatılınca döner. `confirm`
  OK ve Cancel gösterir ve OK için true döndürür.

İletişim kutuları macOS'ta (standart açma ve kaydetme panelleri ile uyarılar)
ve Windows'ta (ortak iletişim kutuları) sistemin kendisinindir. AhdCode'un
hiçbir masaüstü araç takımına bağlanmadığı Linux'ta yardımcı kendi iletişim
kutusunu GUI bileşenleriyle çizer: dosya iletişim kutusu her seferinde bir
klasörü listeler, Up üst klasöre çıkar ve seçili bir klasörde Open ona girer;
bu iletişim kutusu `openFiles` için de her seferinde bir dosya seçer.
İletişim kutusu program için modaldır ve program onu bekler: açık bir GUI
penceresi bu arada çizilmeye devam eder, ancak callback'leri iletişim kutusu
kapandıktan sonra çalışır.

## Hatalar

`GUIError` GUI sorunları için fırlatılır: geçersiz pencere boyutu, spacing
veya padding, desteklenmeyen renk, ikinci bir kök Container, yanlış
genişlikte bir TableView satırı, girdilerin dışında bir indeks, geçersiz bir
dosya uzantısı, kapalı bir Window'da değişiklik, gösterilemeyen bir iletişim
kutusu, eksik veya başarısız `ahdgui` yardımcısı ve bozuk yardımcı yanıtı.
Bir callback'in fırlattığı hata ve iptal edilen bir iletişim kutusu için
hiçbir zaman kullanılmaz.

## Sınırlar

| Ne | Sınır |
| --- | --- |
| Pencere genişliği ve yüksekliği | 1 ile 4096 nokta |
| Başlık | 256 karakter |
| Label, Button, Checkbox, TextInput, PasswordInput metni | 4096 karakter |
| TextArea metni | 100.000 karakter |
| ListBox ve Select öğeleri | 1024 karakterlik 20.000 öğe |
| TableView | 64 sütun, 20.000 satır, 1024 karakterlik 200.000 hücre |
| Bir Window'daki bileşenler | 1024 |
| İletişim kutusu başlığı, metni | 256, 4096 karakter |
| Önerilen dosya adı, uzantılar | 255 karakter, 16 karakterlik 32 uzantı |

Sınırı aşan bir istek `GUIError`'dır; hiçbir şey sessizce kısaltılmaz.

## Nasıl çalışır

Her açık Window, diğer AhdCode yardımcılarının yanına kurulan paketli
`ahdgui` yardımcısının bir süreci tarafından, her iletişim kutusu da aynı
yardımcının kısa ömürlü bir süreci tarafından çizilir. Program ile yardımcı,
yardımcının standart girdisi ve çıktısı üzerinden sınırlı JSON mesajları
alışverişi yapar; yardımcı tıklama, tuş, değişiklik ve kapanma olaylarını
geri gönderir.
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

- macOS: bu sürümün geliştirme Mac'inde canlı olarak test edildi. v1.9.0'dan
  itibaren bir Window odaktayken menü çubuğu **AhdCode**'u, Dock ise AhdCode
  simgesini gösterir (v1.8.0 yardımcının adını, `ahdgui`, gösteriyordu);
  v2.0'dan itibaren Dock simgesi macOS'un yuvarlatılmış biçimindedir.
  [`ahdcode package`](PACKAGING_TR.md) ile yapılan bir uygulamada bunların
  yerine uygulamanın kendi adı ve simgesi gösterilir.
- Windows ve Linux: v1.9.0'dan itibaren pencere, sistemin pencere simgesi
  gösterdiği yerlerde AhdCode simgesini kullanır.
- Windows ve Linux: yardımcı her ikisi için de derlenir; Linux masaüstü bir
  X11 ekranı (XWayland da sayılır) ve sistemin OpenGL kitaplıklarını gerektirir.
- Windows: yerel iletişim kutuları derlenir ve yapısal olarak test edilir,
  ancak bu sürüm için bir Windows makinesinde çalıştırılmamıştır.
- Otomatik testler: `AHDCODE_GUI_HEADLESS=1` her Window'u ekran olmadan açar,
  `AHDCODE_GUI_HEADLESS_DIALOGS` iletişim kutularını bir listeden yanıtlar.

## GUI ne değildir

Bunlar eksik temel özellikler değil, bilinçli sınırlardır:

- ağaç görünümü, sekmeler, menüler, şerit (ribbon), yerleştirme (docking)
  veya çok belgeli pencereler yoktur;
- zengin metin düzenleyici, sözdizimi renklendirme, gömülü tarayıcı veya
  web görünümü yoktur;
- tema veya CSS motoru, yazı tipleri veya açık renklerin ötesinde stiller
  yoktur;
- özel bileşen veya eklenti sistemi ve görsel GUI tasarımcısı yoktur;
- TableView'da düzenlenebilir hesap tablosu hücreleri, formüller, sıralama
  veya filtreleme çerçevesi ve veritabanlarına veya Data Table'lara otomatik
  bağlama yoktur;
- sürükle-bırak, pano API'si, animasyonlar veya Window içinde çizim yoktur
  (çizim için [Graphics](GRAPHICS_TR.md) kullanın);
- async, iş parçacıkları, future'lar veya arka plan görevleri yoktur:
  callback'ler birer birer çalışır ve uzun bir callback penceresini bekletir;
- video düzenleyiciler, 3B tasarım araçları veya oyun motorları gibi
  profesyonel yaratıcı uygulamalar için altyapı yoktur.

Radyo düğmelerinin işini Select görür; görünürlük (gizle/göster) v2.0'da
yoktur.
