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
sıradan denetimleri kapsar. v2.9 bu takımı sürdürür. Görünürlük, Label boyu
ve kalınlığı, program panosu, metin düzenleme, pencere menüleri, ızgara,
hizalama, kaydırılabilir alanlar, resimler, satır kaydıran etiketler,
sekmeler, radyo düğmeleri, ilerleme çubukları, tarih alanları, zamanlayıcılar,
bir arka plan görevi ve pencerenin içine gömülü bir Canvas vardır. Bir
TableView başlıklarından sıralanır ve `GUI.resource` uygulamanın kaynak
klasöründeki bir dosyayı adlandırır.

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
TableView.sort(column: Int, descending: Bool := false) -> Nothing
TableView.sortedColumn() -> Int?    TableView.sortDescending() -> Bool

GUI.openFile(title: String := "Open File", extensions: List<String> := []) -> String?
GUI.openFiles(title: String := "Open Files", extensions: List<String> := []) -> List<String>
GUI.selectFolder(title: String := "Select Folder") -> String?
GUI.saveFile(title: String := "Save File", suggestedName: String := "", extensions: List<String> := []) -> String?
GUI.message(title: String, text: String) -> Nothing
GUI.confirm(title: String, text: String) -> Bool
GUI.resource(name: String) -> String
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

## Yerleşim: tek kök, Column, Row ve ızgara

Bir Window'un en fazla bir kök Container'ı vardır: ilk `window.column()`,
`window.row()`, `window.grid()` veya `window.scroll()` onu oluşturur; bir
kök daha istemek `GUIError`'dır. Boş bir Window geçerlidir. Diğer her
Container bir Container'ın içinde oluşturulur.

- **Column** çocuklarını yukarıdan aşağıya, **Row** soldan sağa yerleştirir.
  **grid** onları sabit sayıda sütunda soldan sağa dizer ve satır dolunca
  alta geçer. `columnGap` sütunlar, `rowGap` satırlar arasındaki boşluktur;
  ikisinin varsayılanı 8'dir.
- `spacing`, Column, Row veya kaydırma alanının komşu çocukları arasındaki
  boşluk, `padding` ise kenar boşluğudur; ikisi de pencere noktası
  cinsindendir. Spacing, padding ve ızgara boşlukları 0 ile 1000 arasında
  olmalıdır. Bir ızgaranın 1 ile 32 arasında sütunu vardır.
- Her bileşenin doğal bir boyutu vardır: Label metni kadar geniştir, Button
  metni artı bir kenar boşluğu kadar (en az 64 nokta), TextInput ve
  PasswordInput 260 nokta genişliğindedir ve her tek satırlık bileşen 32
  nokta yüksekliğindedir. TextArea 360 × 136, ListBox 260 × 146 noktadır;
  Select en uzun öğesi kadar (200 ile 480 nokta), TableView sütunları kadar
  (200 ile 640 nokta) geniştir ve 222 nokta yüksekliğindedir.
- Açık bir hizalama yoksa Column'un çocukları sol kenarda, Row'un çocukları
  dikey olarak ortada kalır. Izgara hücresi çocuğu hücrenin sol üstüne koyar.
- Kök, pencerenin sol üstünden başlar. Pencerede menü çubuğu varsa kök onun
  28 nokta altından başlar; çubuk formun üzerine binmez. Sığmayan kısım
  kırpılır. Kök bir kaydırma alanıysa bu kuralın yerine aşağıdaki kural geçer.

`setMinWidth(width)` ve `setPreferredWidth(width)` doğal genişliği yükseltir.
Küçültmez: içerikten küçük bir tercih yok sayılır, 0 ayarı temizler. Negatif
bir genişlik veya 4096'dan büyük bir genişlik `GUIError`'dır.

`setAlign(horizontal, vertical := "")` hizalamayı seçer. Değerler `"start"`,
`"center"`, `"end"` veya `"stretch"` olabilir. Yatay hizalama Column'da ve
ızgara hücresinde, dikey hizalama Row'da ve ızgara hücresinde geçerlidir. Boş
dikey değer Row'un eski davranışını korur. Stretch, çocuğu o eksende yuvasına
yayar. Hizalama bir bileşeni küçültmez; yuvasından büyük bir bileşeni içeri
çekmez. Taşan kısım kırpılır veya bileşen bir kaydırma alanındaysa kayar.

`setSpan(columns, rows := 1)`, bir bileşenin kapladığı ızgara hücresi
sayısıdır; 1 ile 32 arasındadır. Izgaradan geniş bir span bütün sütunları
kullanır. Izgara dışında span saklanır ve yerleşimi değiştirmez.

Izgara kuralları, sırasıyla:

1. Gizli çocuklar atlanır. Sonraki görünen çocuk o hücreye gelir. Gizlemek
   boş bir delik bırakmaz.
2. Satıra sığmayan çocuk alta geçer. Bir satır span'inin kapladığı hücre
   atlanır.
3. Bir sütunun genişliği, o sütunu tek başına kullanan en geniş çocuktur.
   Bir satırın yüksekliği, o satırı tek başına kullanan en uzun çocuktur.
   Birden çok iz süren çocuk, toplam ve boşluklar onu alana kadar o izleri
   eşit yükseltir.
4. Izgaranın doğal boyutu izler, boşluklar ve padding'dir.
5. Izgaraya fazla genişlik verilirse her sütun eşit pay alır; kalan pay ilk
   sütunlara gider. Fazla yükseklik yalnızca büyüyen bir çocuk içeren
   satırlara gider (TextArea, ListBox, TableView veya bunları tutan bir
   Container). Aksi halde ızgara üste hizalı kalır.
6. Çocuk hücresinin içinde hizalanır. Büyüyen çocuk hücresini doldurur.
   Hiçbir şey doğal genişliğinin altına inmez.

Mutlak konumlandırma yoktur. Kaydırmalı bileşenler — TextArea, ListBox ve
TableView — ve bunları içeren her Column, Row veya ızgara **büyür**. Pencere
içeriğinden büyük olduğunda Column'un fazladan yüksekliği büyüyen çocukları
arasında eşit paylaşılır; Row'un fazladan genişliği de öyle. Büyüyen bir
çocuk Column'unun genişliğini (Row'unun yüksekliğini) doldurur.

### Kaydırma alanları

`window.scroll(spacing := 8, padding := 12)` kaydırılabilir köktür.
`container.scroll(spacing := 8, padding := 0)` iç içe olandır. Çocuklar
doğal boyutlarında yukarıdan aşağıya dizilir; kaydırma alanının içinde
büyümezler. Yığının genişliği alanın doğal genişliğidir. Doğal yükseklik
yığının yüksekliğidir; yığın daha uzunsa 256 noktadır. Daha uzun bir kaydırma
alanı büyür, böylece bir Column ona büyük pencereden pay verebilir.

Kaydırma **kökü** ayrıdır: pencereyi doldurur; menü çubuğu varsa onun
altında. Formdan kısa bir pencere dikey, formdan dar bir pencere yatay
kayar. Geniş bir formu kaydırmanın olağan yolu budur. İç içe bir kaydırma
alanı içeriği kadar geniştir ve dikey kayar. İki yönde kaydırmak için kaydırma
alanını kök yapın.

Tekerlek, imlecin altındaki kaydırma alanını her iki eksende kaydırır ve
kaydırma çubuğu sürüklenebilir.
İmleç bir ListBox, TableView veya TextArea üzerindeyse o bileşen kendi
kaydırmasını korur. Bir çentik 72 noktadır. Devre dışı bir liste kaymaz.
İçerik alanın dışına çizilmez. Görünümün dışına kaymış bir bileşene tıklama
ona gitmez; tıklama, görünümün içinde o anda imlecin altındaki bileşene gider,
kaydırma çubuğuna değil. Tab ve betikli odak, odaklanan denetimi görünüme
getirir. Gizli bir denetim yer kaplamaz; tıklama ve Tab onu atlar. Devre
dışı bir denetim görünür kalır, yerini korur ve tıklama, tuş veya Tab kabul
etmez. Açık bir Select listesi ve menü çubuğu kayan içeriğin üstünde
çizilir.

`examples/v2.9/layout`, kaydırılabilir bir pencerede iki sütunlu bir formdur:
iç içe bir column, gizli bir düğme ve kaydırmaya yetecek kadar satır vardır.

### Yerleşimi elle denemek

Otomatik testler geometriyi, kırpmayı, tekerlek kaydırmasını, odağı ve bir
noktanın hangi bileşene denk geldiğini denetler. Fiziksel fareyi oynatmazlar.
Bu geliştirme Mac'inde bir kişi `examples/v2.9/layout` penceresini fiziksel
fare ve klavye ile denedi: File menüsü açıldı, metin alanına yazıldı,
Command+A, Command+C, Command+V ve Command+Z o alanı düzenledi, tekerlek
formu kaydırdı ve pencere temiz kapandı. Windows 11 ve Wayland
çalıştırılmadı.

Aynı pencere bu denetimi yinelemek içindir:

1. `examples/v2.9/layout` çalıştırılır. Menü çubuğu formun üstündedir.
2. Name alanına tıklanıp yazılır. macOS'ta Command, Windows ve Linux'ta Ctrl
   ile C alanı kopyalar. Aynı değiştirici ile S, Save'i çalıştırır.
3. Tekerlekle kaydırılır. İmlecin o anda altındaki denetime tıklanır ve
   tıklamayı onun aldığı görülür.
4. Tab'a, pencerenin altında başlayan bir denetim görünüme gelene kadar
   basılır.
5. Gizli Extra düğmesinin boş hücre bırakmadığı görülür.

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
- Gövdeye tıklamak bir satır seçer; tablo odaktayken ok tuşları, Home, End,
  PageUp ve PageDown seçimi taşır. Bir başlığa tıklamak o sütunu sıralar. İlk
  tıklama artandır. Aynı başlığa sonraki tıklama azalandır. Başka bir başlık
  o sütunu artan sıralar. Hücreler metin olarak, Unicode kod noktası sırasında
  ve büyük-küçük harfe duyarlı karşılaştırılır: artan sırada boş bir hücre
  diğer her metinden önce gelir ve `"10"`, `"2"`den önce gelir. Sıra sayısal
  değildir ve yerel ayara bağlı değildir. Eşit hücreler şu anki göreli
  sırasını korur. Seçili satır aynı satır kalır ve yeni indeksine taşınır.
  Başlık tıklaması `onSelect` çağırmaz. `sort(column, descending := false)`
  programın seçtiği aynı sıradır ve o da `onSelect` çağırmaz.
  `sortedColumn()`, tablo sıralanana kadar null'dır. `setRows` geçerli
  sıralamayı korur ve yeni satırlara uygular: yeni veride seçili indekse
  düşen satır seçili kalır, sonra sıralamayla taşınır. Devre dışı bir tablo
  başlık tıklamasını yok sayar. `sort` yine çalışır. Hücreler düzenlenemez:
  veriyi değiştirip `setRows` çağırın.
- `columns()` ve `rows()` ekrandaki sıranın kopyasını döndürür. Bir TableView
  en fazla 64 sütun,
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
  aynı anda çalışmaz. Bir arka plan görevi de bir callback ile aynı anda
  çalışmaz; pencerenin olayları işlemesine izin verdiği tek yer
  `GUI.checkpoint` çağrısıdır.
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

Bir Button, TextInput, Checkbox, PasswordInput, TextArea, Select, ListBox,
TableView, Image, RadioButton, DateField veya Progress devre dışı
bırakılabilir. Her bileşen etkin başlar; Label, Container, Tabs ve
RadioGroup için etkinlik durumu yoktur. Image ve Progress, etkin de olsa
devre dışı da olsa, Tab sırasında yer almaz.

- Devre dışı bir Button tıklanamaz ve Enter veya Space ile etkinleştirilemez;
  bu yüzden `onClick` callback'i çalışmaz.
- Devre dışı bir TextInput yazmayı ve düzenleme tuşlarını yok sayar.
- Devre dışı bir Checkbox tıklama veya Space ile değiştirilemez.
- Devre dışı bir Select, ListBox veya TableView kullanıcı tarafından
  açılamaz, seçilemez veya kaydırılamaz.
- Devre dışı bir bileşen odak alamaz: ona tıklamak odağı hiçbir yere
  taşımaz, Tab onu atlar ve odaktayken devre dışı bırakılan bir bileşen
  odağı kaybeder.
- Devre dışı bir RadioButton tıklama, Space veya ok tuşlarıyla seçilemez.
  Program onu yine seçebilir.
- Devre dışı bir DateField tıklamayı ve yazmayı yok sayar. Program yine
  `setDate` çağırabilir.
- Devre dışı bir Progress soluk çizilir. Program değerini yine ayarlayabilir.
- Program devre dışı bir bileşeni yine değiştirebilir: `setText` ve
  `setChecked` çalışır, `text()` ve `checked()` onu okur.
- Devre dışı bir bileşen normal çizilir ve ardından pencerenin varsayılan
  rengine doğru soluklaştırılır.

`isEnabled()` durumu bildirir, Window kapandıktan sonra da; kapalı bir
Window'da `setEnabled` bir `GUIError`'dır. Devre dışı bir denetim görünür
kalır ve yerleşimdeki yerini korur. Gizlemek ayrıdır: `setVisible` denetimi
yerleşimden, tıklamadan ve Tab'dan çıkarır.

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
kutusu, eksik, bozuk veya desteklenmeyen bir resim, eksik veya başarısız
`ahdgui` yardımcısı, bozuk bir yardımcı yanıtı, gerçek bir takvim günü
olmayan bir tarih, sınırın dışındaki bir ilerleme değeri veya üst sınırı,
denetimin dışındaki bir sekme veya radyo indeksi, boş bir sekme başlığı
veya radyo yazısı.
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
| Resim dosyası | PNG veya JPEG, en çok 64 MiB, bir kenarı en çok 8192 piksel |
| Resim yolu | 4096 bayt |
| Resim genişliği ve yüksekliği | 0 ile 4096 nokta; 0 o kenarın verilmediği anlamına gelir |
| Tabs | 32 sayfa; başlık 1 ile 256 karakter |
| Radyo grubu | 64 seçenek; yazı 1 ile 256 karakter |
| Progress üst sınırı | 1 ile 1000000; değer 0 ile bu üst sınır arasındadır |
| Tarih | yıl 1 ile 9999, gerçek bir Gregoryen ayı ve günü |

Sınırı aşan bir istek `GUIError`'dır; hiçbir şey sessizce kısaltılmaz.

## Nasıl çalışır

Her açık Window, diğer AhdCode yardımcılarının yanına kurulan paketli
`ahdgui` yardımcısının bir süreci tarafından, her iletişim kutusu da aynı
yardımcının kısa ömürlü bir süreci tarafından çizilir. Program ile yardımcı,
yardımcının standart girdisi ve çıktısı üzerinden sınırlı JSON mesajları
alışverişi yapar; yardımcı tıklama, tuş, değişiklik ve kapanma olaylarını
geri gönderir.
Yardımcı kabuk komutu çalıştırmaz, ağ kullanmaz ve kullanıcının yazdığı
hiçbir şeyi günlüğe yazmaz. Okuduğu tek dosya, programın göstermesini
istediği resimdir. Graphics
yardımcısıyla aynı pencere kitaplığıyla (Ebitengine v2.10.2) derlenir ve Go
Regular yazı tipini içerir; bkz.
[THIRD_PARTY_NOTICES_GUI.md](../THIRD_PARTY_NOTICES_GUI.md).

`ahdcode run`, derlenmiş programlar ve REPL aynı gerçeklemeyi kullanır.
Derlenmiş bir program yardımcıyı kendisini derleyen kurulum üzerinden veya bir
AhdCode kurulumunda kendi yanında bulur; `PATH` hiçbir zaman aranmaz.
`AHDCODE_GUI_RUNTIME`, geliştirme ve testler için yardımcıyı açıkça belirtir.

## Platform notları

- macOS: bu sürümün geliştirme Mac'inde canlı olarak test edildi. Fiziksel
  fare ve klavye ile pencere içindeki File menüsü, metin düzenleme,
  Command+A, C, V ve Z, kaydırma ve pencereyi kapatma da denendi. v1.9.0'dan
  itibaren bir Window odaktayken menü çubuğu **AhdCode**'u, Dock ise AhdCode
  simgesini gösterir (v1.8.0 yardımcının adını, `ahdgui`, gösteriyordu);
  v2.0'dan itibaren Dock simgesi macOS'un yuvarlatılmış biçimindedir.
  Bu sistem menü çubuğu yalnızca uygulamanın adıdır. AhdCode menüleri her
  platformda pencerenin içinde çizilir. [`ahdcode package`](PACKAGING_TR.md)
  ile yapılan bir uygulamada bunların yerine uygulamanın kendi adı ve
  simgesi gösterilir.
- Windows ve Linux: v1.9.0'dan itibaren pencere, sistemin pencere simgesi
  gösterdiği yerlerde AhdCode simgesini kullanır.
- Windows ve Linux: yardımcı her ikisi için de derlenir; Linux masaüstü bir
  X11 ekranı (XWayland da sayılır) ve sistemin OpenGL kitaplıklarını gerektirir.
- Windows: yerel iletişim kutuları derlenir ve yapısal olarak test edilir,
  ancak bu sürüm için bir Windows makinesinde çalıştırılmamıştır.
- Otomatik testler: `AHDCODE_GUI_HEADLESS=1` her Window'u ekran olmadan açar,
  `AHDCODE_GUI_HEADLESS_DIALOGS` iletişim kutularını bir listeden yanıtlar.

## Görünürlük, Label boyu ve pano

Her bileşenin ve her Container'ın `setVisible(visible)` ve `isVisible()`
üyesi vardır. Gizli bir bileşen Column, Row, ızgara veya kaydırma alanında
yer kaplamaz, tıklama almaz ve Tab ile atlanır. Bir Container gizlenince içindeki her şey de
gizlenir; çocuk üzerindeki `isVisible()` o Container yeniden gösterilene
kadar false döner. Gösterilen bileşen kardeşleriyle birlikte yeniden yerleşir.
Gizli bir bileşen, devre dışı bir bileşen değildir: devre dışı bileşen görünür
kalır ve yerini korur.

Bir Label'ın `setSize(size)` ve `setBold(bold)` üyesi vardır. `size`
`"small"`, `"normal"`, `"large"` veya `"title"` olur. Başka bir metin
GUIError'dur. Serbest yazı tipi seçimi yoktur.

`GUI.setClipboard(text)` programın metin panosunu değiştirir,
`GUI.clipboard()` onu okur. Metin en çok 100000 karakterdir. TextInput,
PasswordInput ve TextArea bu panoyu paylaşır.

Tıklama imleci koyar. Sürüklemek seçer, çift tıklama işaretçinin altındaki
sözcüğü seçer. Shift ile ok tuşları, Home ve End seçimi büyütür. Backspace,
Delete ve yazma seçimin tamamına uygulanır. İmleç yazı birimiyle ilerler;
birleşen işaret veya emoji taban karakterinden ayrılmaz. Alana sığmayan bir
yapıştırma reddedilir ve metin olduğu gibi kalır. PasswordInput yapıştırmayı
kabul eder; kopyalamaz ve kesmez.

Art arda yazmak tek bir geri alma adımıdır. Yapıştırma, silme veya seçimin
yazmayla değiştirilmesi ayrı birer adımdır. Geri alma ve yineleme metni ve
imleci geri getirir. macOS kısayolları Command+A, C, X, V ve Z'dir;
Command+Shift+Z yinelemedir. Windows ve Linux kısayolları Control+A, C, X, V
ve Z'dir; Control+Y ve Control+Shift+Z yinelemedir. Gizli veya etkisiz bir
alan bu tuşları yok sayar.

```ahd
bring GUI
form: Container := GUI.window("Form").column()
note: Label := form.label("Ready")
note.setSize("title")
note.setBold(true)
save: Button := form.button("Save")
save.setVisible(false)
GUI.setClipboard(note.text())
```

## Menüler ve kısayollar

Bir menü, Window'unun üstünde bir çubukta çizilir. macOS, Windows ve Linux
aynı çubuğu kullanır. Bu, işletim sisteminin menü çubuğu değildir.

```ahd
file: Menu := window.menu("File")
save: MenuItem := file.item("Save", "S")
file.separator()
view: Menu := file.menu("View")
bold: MenuItem := view.checkItem("Bold title", "", false)
save.onClick(onSave)
```

`menu(title)` çubuğa bir menü veya bir menünün içine bir alt menü ekler.
Bir alt menü başka bir menü içeremez. `item(label, shortcut := "")` bir
öğe ekler. `checkItem(label, shortcut := "", checked := false)` işaretli
bir öğe ekler. `separator()` bir çizgi çizer. Bir öğenin `onClick`,
`setEnabled`, `isEnabled` ve `setText` üyesi vardır. İşaretli bir öğenin
ayrıca `setChecked` ve `checked` üyesi vardır. Düz bir öğede `setChecked`
GUIError'dur.

Kısayol metni platformun komut tuşunu içermez. `"S"`, macOS'ta Command+S,
Windows ve Linux'ta Ctrl+S demektir. `"Shift+S"`, `"Alt+S"` ve
`"Alt+Shift+S"` Shift ve Alt (macOS'ta Option) ekler. `Ctrl`, `Command`
ve `Cmd` reddedilir; komut tuşu iki kez yazılmaz. Bir harf büyük veya
küçük yazılabilir. İki öğe aynı kısayolu kullanamaz; ikincisi GUIError'dur.
Boş kısayol çakışma değildir.

Etkisiz bir öğe kısayolunu tutar ve kısayol bir şey yapmaz. İşaretli bir
öğe çalışınca, `onClick`'ten önce işareti değişir. Geri çağrılar Button
tıklamaları gibi teker teker çalışır ve başka bir Function ile aynı
`Global` adları görür. Menü açmak yerel değişken yakalamaz.

Odaktaki bir Window için kısayol sırası:

1. Odakta bir TextInput, TextArea veya PasswordInput varsa ve akor bir
   düzenleme akoruysa alanı onu işler, menü öğesi çalışmaz. Bu akorlar
   her platformda A, C, X, V ve Z, her platformda Shift+Z ve Windows ile
   Linux'ta Y'dir. PasswordInput yine kopyalamaz ve kesmez; C ve X'i yine
   de tüketir.
2. Aksi halde akora sahip tek etkin öğe bir kez çalışır. Tuş yazılmaz ve
   `Window.onKey`'e iletilmez.
3. Akorun tek öğesi etkisizse akor bir şey yapmaz.
4. Odakta olmayan bir Window kısayol almaz. Gizli veya etkisiz bir alan
   odakta değildir, bu yüzden bir düzenleme akoru almaz.

`examples/v2.9/menu` küçük bir not penceresidir: Save, TextInput'u okur
ve kısayolu Command/Ctrl+S'tir.

## Resimler

`container.image(path, width := 0, height := 0)` bir PNG veya JPEG dosyası
gösterir. Image bir çizim yüzeyi değildir ve bir Button değildir: tıklama
callback'i yoktur ve Tab sırasında yer almaz.

`width` ve `height` pencere noktasıdır, 0 ile 4096 arasındadır. Sıfır, o
kenarın verilmediği anlamına gelir.

- İkisi de verilmezse kutu resmin kendi piksel boyudur; iki kenar da 4096'yı
  aşmayacak biçimde küçültülür. Oran korunur.
- Yalnız genişlik verilirse yükseklik resme uyar. Yalnız yükseklik verilirse
  genişlik resme uyar. Uyan kenar 4096'yı aşacaksa o kenar 4096 olur ve resim
  kutunun içine sığdırılır.
- İkisi de verilirse kutu tam o boydur. Resim kutunun içinde ortalanır ve
  oranı korunur. Boş kalan yer Image'ın arka planını, arka plan yoksa
  arkasında ne varsa onu gösterir.

`setSize(width := 0, height := 0)` bu seçimi sonradan değiştirir.
`setMinWidth` ve `setPreferredWidth` kutuyu resimden daha geniş yapabilir.
Resim ortada kalır ve gerilmez. `setAlign` ve `setSpan` diğer bileşenlerdeki
gibi çalışır. Image bir Column, Row, ızgara veya kaydırma alanında durabilir
ve bir kaydırma alanı onu Label gibi kırpar.

Resim, Image oluşturulurken okunur ve saklanır. Sonraki kareler dosyayı
yeniden okumaz ve yeniden çözmez. Eksik bir dosya, bir dizin, 64 MiB'den
büyük bir dosya, bir kenarı 8192 pikselden büyük bir resim, bozuk bir dosya
veya PNG ya da JPEG olmayan bir dosya GUIError'dır ve Image oluşturulmaz.
GIF ve diğer biçimler reddedilir.

Yol, Graphics ve File ile aynı kural olarak programın çalışma dizinine
göredir. Çalışma zamanı yolu mutlak yapar, yardımcı ondan sonra okur.
Yardımcının kendi klasörü aranmaz. Finder'dan açılan paketlenmiş bir
uygulamanın çalışma dizini ev klasörüdür; bir terminalden başlatılan
uygulamanınki o terminalin klasörüdür. `image` uygulamanın kaynak klasörüne
bakmaz.

`GUI.resource(name)` o kaynak klasöründeki bir dosyanın kurallı mutlak
yolunu döndürür. `name` klasöre görelidir; `..` ve mutlak yol kabul edilmez.
Çalışma dizini değildir ve ev dizini değildir. Klasörün kendisi ve dosya,
sembolik bağlantılar çözülerek bulunur. Bir bağlantı, dosyanın kurallı yolu
klasörün kurallı yolunun içinde kaldığında kabul edilir; iç içe bir dizin
bağlantısı da buna dahildir. Dışarıyı gösteren bir bağlantı GUIError'dır.
Hedefi olmayan bir bağlantı, eksik bir dosya veya bir dizin de GUIError'dır.
`ahdcode run`, giriş modülünün yanındaki `resources` klasörünü kullanır.
Paketlenmiş bir macOS uygulaması `Contents/Resources/app` kullanır.
Paketlenmiş bir Windows veya Linux uygulaması, `runtime` değil, programın
yanındaki `resources` klasörünü kullanır. Paketlenmiş bir resim isteyen
program o yolu `image`'a verir:

```ahd
mark: Image := form.image(GUI.resource("icons/mark.png"))
```

`ahdcode package`, girişin yanındaki `resources` klasöründen `.png`, `.jpg`
ve `.jpeg` dosyalarını uygulamanın bu klasörüne kopyalar. Oradaki başka bir
dosya hatadır. Veri dosyaları kopyalanmaz.

`setEnabled(false)` Image'ı görünür, yerinde ve soluk bırakır. Yine tıklama
veya Tab kabul etmez. `setVisible(false)` onu yerleşimden çıkarır.

```ahd
mark: Image := form.image("mark.png", 160, 0)
mark.setSize(120, 80)
```

## Satır kaydıran etiketler

`label.setWrap(wrap)` satır kaydırmayı açar veya kapatır. Program açmadıkça
kaydırma kapalıdır. Kaydırmayan bir Label tek satırdır; `setSize` ve
`setBold` önceki gibi çalışır.

Kaydırma açıkken:

- Kaydırma genişliği, ikisinden biri sıfırdan büyükse tercih edilen genişlik
  ile en küçük genişliğin büyüğüdür. Metin daha geniş olsa da bu genişlik
  kullanılır. Label'ın genişliği bu kaydırma genişliği olur; yüksekliği satır
  sayısı çarpı Label'ın satır yüksekliğidir.
- İki genişlik de verilmemişse Label yalnızca açık satır sonlarında kırılır.
  Genişliği en uzun satırıdır.
- `\r\n` ve `\r`, `\n` gibi satır sonudur. Boş bir satır yüksekliğini korur.
- Boşluklar kelimeleri ayırır; art arda gelen boşluklar tek boşluk olur.
  Satırdan geniş bir kelime grafik kümelere bölünür, bu yüzden bir Türkçe
  harf veya bir emoji ortadan kesilmez. Satırdan geniş tek bir küme kendi
  satırında kalır ve Label'ın kutusuna kırpılır. Gömülü yazı tipinde emoji
  glifi yoktur; o küme yazı tipinin eksik karakter kutusu olarak çizilir.
- Metin kutunun üstünden, satır satır çizilir ve kutuya kırpılır. Bir
  kaydırma alanı onu bir kez daha kırpar. Label gerilirse kutu genişler ve
  metin o genişliğe göre yeniden kayar. Kutu genişlediğinde yükseklik
  küçülmez.
- Pencere veya bir kaydırma görünümü, ölçülen genişlikten dar olduğunda
  Label, altındaki bileşenler yerleştirilmeden önce bu dar genişlikte yeniden
  ölçülür. Yeni yükseklik yeni satır sayısıdır; alttaki bileşenler aşağı
  kayar ve satırlar onların üzerine binmez. Label, en küçük genişliğinden
  veya en geniş grafik kümesinden daha dar olmaz. Kaydırmayan bir alan,
  düğme, resim veya başka bir bileşen ölçülen genişliğini korur ve bir ızgara
  sütunu o bileşene yetecek kadar geniş kalır. Label bundan sonra o
  bileşenlerin bıraktığı yuvaya kayar. Yuva pencereden hâlâ genişse fazlalık
  kayar veya kırpılır. Bir kutu, kendisine verilen yükseklikten dar olursa
  çizim ölçülen satır sonlarını korumaya devam eder.
- `setSize` ve `setBold` satır yüksekliğini ve kalınlığı değiştirmeye devam
  eder. Kalın bir Label'ın ek genişliği, tek satırlık bir Label'ın ek
  genişliğiyle aynıdır.

Label dışında bir bileşende `setWrap` bir GUIError'dır. Argümansız
`setWrap()` derleyici tarafından reddedilir.

`examples/v2.9/gallery` bir pencerede resim, satır kaydıran etiket, ızgara
ve kaydırma alanını bir araya getirir.

## GUI ne değildir

Bunlar eksik temel özellikler değil, bilinçli sınırlardır:

- ağaç görünümü, şerit (ribbon), yerleştirme (docking) veya çok belgeli
  pencereler yoktur; işletim sistemi menü çubuğu uygulama adının ötesinde
  yoktur;
- zengin metin düzenleyici, sözdizimi renklendirme, gömülü tarayıcı veya
  web görünümü yoktur;
- tema veya CSS motoru yoktur; yazı tipi bir Label'ın dört boyu ve
  kalınlığıyla sınırlıdır;
- özel bileşen veya eklenti sistemi ve görsel GUI tasarımcısı yoktur;
- TableView'da düzenlenebilir hesap tablosu hücreleri, formüller veya
  filtreleme çerçevesi yoktur. Başlık sıralaması metni karşılaştırır; yukarıda
  anlatılmıştır. Veritabanlarına veya Data Table'lara otomatik bağlama yoktur;
- sürükle-bırak yoktur ve genel bir animasyon çerçevesi yoktur.
  `window.every` bir zamanlayıcıdır. Belirsiz ilerleme çubuğu kendi hareketli
  bloğunu çizmeye devam eder. Image bir resim dosyası gösterir. Pencerenin
  içindeki Canvas küçük bir şekil kümesi çizer; tam çizim penceresi
  [Graphics](GRAPHICS_TR.md) olmaya devam eder;
- pencere başına işbirlikli bir arka plan görevi vardır; async, iş
  parçacıkları, future'lar veya genel çoklu iş parçacığı yoktur. Bir callback
  hâlâ birer birer çalışır ve uzun bir callback o pencereyi, dönene kadar
  bekletir;
- video düzenleyiciler, 3B tasarım araçları veya oyun motorları gibi
  profesyonel yaratıcı uygulamalar için altyapı yoktur.

## Sekmeler, radyo düğmeleri, ilerleme ve tarih

`container.tabs()` bir Tabs denetimi döndürür. `tabs.page(title)` bir sayfa
ekler ve bir Container döndürür; bu sayfa bir Column'un tuttuğu bileşenleri
tutar. Başlık 1 ile 256 karakterdir ve bir Tabs en çok 32 sayfa tutar. İlk
sayfa seçilidir. `select(index)` sayfayı oluşturulma sırasındaki indeksiyle
seçer, gizli bir sayfa da buna dahildir; `selectedIndex()` bu indeksi
döndürür. Sayfa yoksa veya indeks dışarıdaysa ikisi de `GUIError` fırlatır.

Yalnızca seçili sayfa çizilir ve tıklama, tuş ve Tab alır. Diğer sayfalar
bileşenlerini tutar ama o bileşenler tıklanamaz ve Tab sırasında yoktur.
Bir başlığa tıklamak o sayfayı seçer. Sol ve Sağ görünür başlıklar arasında
gezer ve odağı sekme çubuğunda bırakır. Bir başlığa tıklayarak bir sayfaya
dönmek, o sayfada odağı en son tutan ve hâlâ odak alabilen denetime odağı
geri verir. Gizli bir sayfanın başlığı yoktur. Seçili sayfa gizliyse içerik
alanı boştur. `onChange` yeni indeksi, sayfayı kullanıcı değiştirdiğinde
alır. `select` onu çağırmaz.

`container.radioGroup()` bir RadioGroup döndürür. `group.option(text)` yazısı
1 ile 256 karakter olan bir RadioButton ekler. Bir grup en çok 64 seçenek
tutar. Kullanıcı veya program seçene kadar hiçbir şey seçili değildir.
`select(index)` `Int?` alır; `null` seçimi temizler. `selectedIndex()` ve
`selectedText()`, hiçbir şey seçili değilse null'dır. İndeks gizli olanlar
dahil her seçeneği sayar. Tıklama veya Space etkin bir seçeneği seçer. Yukarı
ve Aşağı, etkin ve görünür seçenekler arasında gezer ve vardığı seçeneği
seçer. Devre dışı bir seçeneği kullanıcı seçemez; `select` onu yine
seçebilir. `onChange` indeksi ve yazıyı alır; seçim temizlenince ikisi de
null'dır ve yalnızca kullanıcının değişikliği için çalışır. Bir RadioButton'ın
yazısı, renkleri, etkinliği ve görünürlüğü vardır. Grubun kendisi etkin veya
devre dışı yapılmaz.

`container.progress(maximum := 100)` bir Progress çubuğu döndürür. Üst sınır
1 ile 1000000 arasında bir Int'tir. Değer 0'dan başlar. `setValue`, 0 ile üst
sınır arasında bir Int ister; dışı `GUIError` fırlatır ve değeri değiştirmez.
`setIndeterminate(true)` çubuğun bir kesri yerine hareketli bir blok çizer.
Program, blok hareket ederken de değeri okuyup ayarlayabilir. Değeri ayarlamak
bir callback çalıştırmaz ve beklemez: pencere yeni değeri kendi sonraki
çerçevesinde çizer. Progress bir Tab durağı değildir. Onu devre dışı bırakmak
soluklaştırır ve programın değeri ayarlamasını durdurmaz. Doğal boyu bir metin
alanı genişliğinde ve bir satır yüksekliğindedir.

`container.date(year := 2000, month := 1, day := 1)` bir DateField döndürür.
Tarih, proleptik Gregoryen takviminde bir gündür: yıl 1 ile 9999, ay 1 ile 12
ve o ayda bulunan bir gün. 29 Şubat, yıl 4'e bölünüyorsa geçerlidir; 100'e
bölünen yıllar ancak 400'e de bölünüyorsa geçerlidir. Denetim her zaman gerçek
bir tarih tutar. Geçersiz bir tarih `GUIError` fırlatır. `setDate` o zaman
önceki tarihi yerinde bırakır. Alan `YYYY-MM-DD` olarak, sıfırlarla ve ay
adları olmadan çizilir; macOS, Windows ve Linux aynı rakamları gösterir. Tek
bir Tab durağıdır ve üç eşit parçası vardır. Tıklama bir parça seçer. Sol ve
Sağ parçalar arasında gezer. Yukarı ve Aşağı o parçayı bir değiştirir. Yıl
1 ile 9999, ay 1 ile 12 arasında kalır; gün ayın uzunluğuna kıstırılır, ay
veya yıl değişince de. Rakamlar parçayı değiştirir: yıl için dört, ay veya
gün için iki. Ancak ortaya çıkan tarih gerçekse uygulanır. `year()`,
`month()` ve `day()` onu okur. `onChange` üçünü de, tarihi kullanıcı
değiştirince alır. `setDate` onu çağırmaz.

`examples/v2.9/controls` dört denetimi bir pencerede bir araya getirir.

## Zamanlayıcılar, arka plan görevleri ve gömülü Canvas

`window.every(milliseconds, handler)` yinelenen bir zamanlayıcı kaydeder ve
Int kimliğini döndürür. `window.cancel(timer)` o zamanlayıcıyı durdurur.
Hiçbir zamanlayıcıyı adlandırmayan bir kimliği iptal etmek bir şey yapmaz.
Aralık 1 milisaniye ile bir gün (86.400.000 milisaniye) arasındadır. Bir
pencere en çok 64 zamanlayıcı tutar; biri iptal edilene kadar sonraki
`every` çağrısı `GUIError` fırlatır. İşleyici `() -> Nothing` şeklindedir.

Gerçek bir pencere, tıklamalar ve tuşlarla aynı olay kuyruğuna her seferinde
bir tik koyar. İşleyici `wait` içinde, o olaylarla sırayla ve birer birer
çalışır; bileşenleri okuyup değiştirebilir. Aralığından uzun süren bir
işleyici kuyruk biriktirmez: kaçırdığı tikler atlanır. İşleyicinin içinden
iptal, sonraki tikten önce geçerli olur. Pencere kapanınca her zamanlayıcı
durur ve kapanıştan sonra kuyruğa girmiş bir tik çalışmaz. Başsız bir
pencerenin saati yoktur. Zamanlayıcıları yalnızca betik
`{"event":"tick"}` gönderince çalışır. `timer` alanı olmayan veya `timer`
değeri 0 olan bir tik, başlatma sırasıyla her zamanlayıcıyı çalıştırır. Sıfır
olmayan bir `timer` yalnızca onu çalıştırır.

`window.task(work)` bir arka plan görevi başlatır ve bir `Task` döndürür.
`work`, `() -> Int` şeklindedir. Bir pencere aynı anda bir görev çalıştırır;
ilki dönmeden yapılan ikinci `task` çağrısı `GUIError` fırlatır. Fonksiyon,
`wait` çalışırken çalışır; `wait` öncesinde oluşturulan bir görev de buna
dahildir. Bir tıklama, bir zamanlayıcı veya başka bir GUI callback'i ile aynı
anda çalışmaz. `GUI.checkpoint()` bu callback'lerin çalışmasına izin verme
yoludur. Görev dışında `checkpoint` true döndürür ve hiçbir şeyi değiştirmez.
Görev içinde, görev iptal edildiyse veya pencere kapanmaya başladıysa false,
pencere hâlâ bir sonuç teslim edebiliyorsa true döndürür. Fonksiyonu durdurmaz.
Dönüp dönmemeye fonksiyon karar verir. `checkpoint` çağırmayan bir kesit,
tek bir dosya veya veritabanı çağrısı da buna dahil, GUI callback'lerini o
kesit dönene kadar tutar. Pencere süreci çizmeye devam eder; program o kesitte
tıklamaları ve zamanlayıcıları işlemez.

`task.onDone(handler)`, `(value: Int?, problem: String?) -> Nothing` işleyicisini
kaydeder. Yeniden kaydetmek öncekinin yerine geçer. İşleyici diğer
callback'lerle aynı yolda çalışır, bu yüzden bileşenleri değiştirebilir.
Başarıda `value` Int'tir ve `problem` null'dır. `work` bir Error fırlatırsa
`value` null'dır ve `problem` o hatanın mesajıdır; başka bir başarısızlık
yayılmaya devam eder. `onDone` yalnızca fonksiyon dönmüşse, görev iptal
edilmemişse ve pencere hâlâ açıksa çalışır. `onDone` kaydedildiğinde görev
bitmişse, aynı kurallarla o anda teslim edilir.

`task.cancel()` sonucun atılmasını ister. Fonksiyonu kesmez. Fonksiyon
`cancel` sonrasında dönerse `onDone` çalışmaz. `task.cancelled()`, `cancel`
sonrasında, fonksiyon hâlâ çalışırken pencere kapanmaya başladıktan sonra ve
hiçbir görevi adlandırmayan bir kimlik için true'dur. Bitmiş bir görevi veya
bilinmeyen bir görevi iptal etmek bir şey yapmaz.

Görev pencereyi değiştirmemelidir. Görevden yapılan bir bileşen çağrısı
`GUIError` fırlatır (`a background task cannot change the window`). Label,
ilerleme çubuğu ve diğer bileşenleri `onDone` içinden, bir zamanlayıcıdan
veya başka bir callback'ten güncelleyin. Görev ile bu callback'ler gizli bir
değişken depo paylaşmaz: yalnızca programın kendi atadığı değerleri kullanırlar
ve `checkpoint` callback'lere izin verdiği süre dışında aynı anda çalışmazlar.
`checkpoint` boyunca aynı değişkeni iki taraftan okuyup yazmayın; program her
iki sırayı da kabul edemiyorsa.

Pencereyi kapatmak, henüz dönmemiş bir görevi iptal eder. `wait` bundan sonra
fonksiyon dönene kadar bekler; `checkpoint`'i yok sayan ve hiç dönmeyen bir
fonksiyon `wait`'in de dönmesini engeller. Sonuç atılır ve `onDone` çalışmaz.
Pencere açıkken bitmiş ve tamamlanması kuyruğa girmiş bir görev, `wait`
dönmeden `onDone`'ını hâlâ çalıştırabilir.

`container.canvas(width := 240, height := 160)` bir Canvas döndürür ve bir
Graphics penceresi açmaz. Bir Column, Row, Grid, Scroll veya Tabs sayfasına
eklenebilir. Kök olamaz. Genişlik ve yükseklik doğal boydur, 1 ile 4096
arasındadır. Yerleşim onu gerebilir. Koordinatlar yerleştirilen boyu kullanır:
başlangıç merkezdir, x sağa, y yukarı büyür; Graphics ile aynı düzen.
Çizim Canvas'a ve onu saran bir kaydırma görünümüne kırpılır. Gizli bir
Canvas yer kaplamaz ve işaretçi olayı almaz. Devre dışı bir Canvas görünür
kalır, soluklaşır ve hiç işaretçi olayı almaz.

Bu Canvas, Graphics çiziminin bir alt kümesidir. Kaplumbağa, resim, yay,
çokgen, elips, SVG veya batch yoktur. Dikdörtgen ve daire çizgisi bir birim
kalınlığındadır. Bir doğrunun kalınlığı 1 ile 64 arasındadır.

```text
Canvas.clear(color: String := "") -> Nothing
Canvas.line(x1: Real, y1: Real, x2: Real, y2: Real, stroke: String := "black", width: Real := 1) -> Nothing
Canvas.rectangle(x: Real, y: Real, width: Real, height: Real, stroke: String := "black", fill: String := "") -> Nothing
Canvas.circle(x: Real, y: Real, radius: Real, stroke: String := "black", fill: String := "") -> Nothing
Canvas.text(x: Real, y: Real, text: String, fill: String := "black", size: Real := 16) -> Nothing
Canvas.onMove(handler: (x: Real, y: Real) -> Nothing) -> Nothing
Canvas.onClick(handler: (x: Real, y: Real) -> Nothing) -> Nothing
Canvas.onDrag(handler: (x: Real, y: Real) -> Nothing) -> Nothing
```

Boş bir `clear` rengi beyazdır. Boş bir dolgu, dolgu çizmez. Renkler Graphics
adları ile `#RRGGBB` ve `#RRGGBBAA` biçimleridir. Koordinatlar sonlu olmalı
ve mutlak değeri en çok 100.000 olmalıdır. Bir dikdörtgenin genişliği ve
yüksekliği 0'dan büyük olmalıdır. Bir dairenin yarıçapı 0'dan büyük ve en
çok 100.000 olmalıdır. Metin 1 ile 1000 karakterdir ve boyu 1 ile 200
arasındadır. Metin için `(x, y)` taban çizgisidir. Bir Canvas en çok 4096
çizim komutu tutar; `clear` yeniden başlatır. `setBackground`, `setEnabled`,
`setVisible`, `setMinWidth`, `setPreferredWidth`, `setAlign` ve `setSpan`
diğer bileşenlerdeki üyelerdir. `setForeground` yoktur. Canvas bir Tab
durağı değildir.

`onMove`, hiçbir düğme basılı değilken işaretçiyi bildirir. `onClick`,
Canvas üzerinde başlayan bir basışı bildirir. `onDrag`, o basış sürerken
işaretçiyi bildirir; işaretçi Canvas'tan çıksa da, bu yüzden bir koordinat
dışarıda olabilir. Devre dışı, gizli veya etkin olmayan bir sayfadaki Canvas
bunların hiçbirini bildirmez. Başlamış bir sürükleme, Canvas devre dışı
kalırsa veya gizlenirse durur. Koordinatlar Canvas koordinatlarıdır; Canvas
kaydırılmış bir kabın içindeyse de öyledir: bileşenin yeri kaydırmayı zaten
içerir ve işaretçi Canvas'ın merkezinden ölçülür. GUI'nin Canvas'ı ile
Graphics'in Canvas'ı aynı ada sahip farklı sınıflardır. `from GUI bring Canvas`
ile `from Graphics bring Canvas` aynı kapsamda birlikte bulunamaz. Yalnızca
gömülü yüzeye ihtiyacı olan bir program GUI'nin Canvas'ını getirir veya sınıfı
hiç adlandırmaz ve `container.canvas` değerini kullanır.

`examples/v2.9/interactive` bir zamanlayıcıyı, bir ilerleme çubuğunu, bir
arka plan toplamını ve hareket eden bir daireyi, Start, Stop ve Cancel ile
tek bir Canvas üzerinde bir araya getirir.

## Açık kalan sınırlar

1. Bu GUI çalışması Windows 11'de ve bir Wayland oturumunda çalıştırılmadı.
   macOS'ta otomatik testler ve yerleşim örneği, Controls, Interactive ve
   defter için fiziksel fare ve klavye geçişleri vardır.
