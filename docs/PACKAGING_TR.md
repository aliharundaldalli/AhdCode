# AhdCode uygulamalarını paketleme

[English](PACKAGING.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [CLI](CLI_TR.md) · [GUI](GUI_TR.md)

> v2.0.0 ile eklendi.

`ahdcode package`, bir AhdCode programını AhdCode kurulu olmayan bir
bilgisayarda çalışan bir masaüstü uygulamasına dönüştürür.

```text
ahdcode package <entry.ahd> [--name <name>] [--output <folder>] [--icon <icon.png>]
                [--target <os-arch>] [--helpers <folder>] [--console] [--web [--pwa]]
```

```sh
ahdcode package examples/v2.0/ledger_app/main.ahd --name Ledger --icon ledger.png
```

| Seçenek | Anlamı | Varsayılan |
| --- | --- | --- |
| `--name` | Uygulamanın adı: en fazla 64 harf, rakam, boşluk, nokta, alt çizgi veya kısa çizgi | giriş dosyasının adı |
| `--output`, `-o` | Uygulamanın yazılacağı klasör | `dist` |
| `--icon` | Kare bir PNG simge, bir kenarda 16 ile 4096 piksel | AhdCode simgesi |
| `--target` | Paketlenecek platform (bkz. [Hedefler](#hedefler)) | bu bilgisayar |
| `--helpers` | O platformun AhdCode yardımcılarını içeren klasör | bu kurulumunki |
| `--console` | Yalnızca Windows: masaüstü programı için konsol penceresini koru | konsol yok |
| `--web` | Gözden geçirilmiş genel varlıkları pakete ekle | kapalı |
| `--pwa` | PWA varlıklarını ekle ve dosya düzenini denetle. `--web` gerekir | kapalı |

Manifest dosyası, paket kayıt defteri veya bağımlılık yöneticisi yoktur:
yapılandırmanın tamamı komut satırıdır.

## Web uygulaması dağıtımı

```sh
ahdcode package app.ahd --web --name Site --output releases/v2.11
# PWA entegrasyonu eklenmişse:
ahdcode package app.ahd --web --pwa --name Site --output releases/v2.11-pwa
```

`--web`, giriş dosyasının yanındaki gözden geçirilmiş `public/` ağacını
göreli yollarıyla kopyalar. `--pwa` ayrıca `PWA/app.webmanifest`,
`PWA/worker.js` ve `PWA/public/` dosyalarını ekler. Manifest geçerli
olmalı ve yerel `/ahd-pwa/` simgelerini göstermelidir.
`PWA/Integration.ahd` derleme sırasında kullanılır. [PWA kurulumu](PWA_TR.md).

Çalışma dizini yürütülebilir dosyanın bulunduğu dizindir. Windows ve
Linux'ta paket kökünü, macOS'ta `Name.app/Contents/MacOS` yolunu kullanın.
Windows'ta yürütülebilir dosya `Site.exe` olur. Örneğin Linux veya macOS'ta:

```sh
APP_NAME=Site APP_ENV=production APP_PROTOCOL=https APP_HOST=example.com \
  SERVER_HOST=127.0.0.1 SERVER_PORT=8080 ./Site
```

Yapılandırmayı süreç ortamından sağlayın. Uygulama `.env` okuyorsa özel
dosyayı dağıtımda ayrıca oluşturun. Admin, MVC ve CRUD için `DB_*`
değerlerini ve hazırlanmış veritabanını ayrı sağlayın. Kalıcı oturum
dosyaları için yazılabilir özel bir dizin kullanın. Caddy dış HTTPS'i
sonlandırırken AhdCode loopback HTTP üzerinden çalışabilir.
[HTTPS ayarları](WEB_TR.md#14-yerel-https).

Genel ağaçlara CSS, JavaScript, statik HTML, web manifesti, yaygın görsel
ve yazı tipi dosyaları ile `LICENSE` dosyaları eklenebilir. Sembolik bağlar,
junction'lar, gizli yollar, özel dizin adları, büyük/küçük harfe duyarsız
dosya sistemlerinde çakışan yollar, desteklenmeyen türler ve
PEM özel anahtarı içeren dosyalar reddedilir. Genel varlıkların içeriğini
gözden geçirin. Paketleyici tüm gizli değerleri tanıyamaz. Proje ayarları,
kaynak modülleri ve veritabanları otomatik kopyalanmaz. Özel varlık
köklerini yapılandırılmış yollarına ayrıca yerleştirin.

Mevcut Web paketinin veya arşivinin üzerine yazılmaz. Yeni dağıtım için
yeni bir hedef seçin. Elle dağıtımda `ahdcode build app.ahd -o Site`
yalnızca yürütülebilir dosyayı oluşturur. Varlıkları ve gereken yardımcıları
belirtilen dizin düzeniyle ayrıca sağlayın.

## Bir masaüstü uygulaması neler içerir

| Platform | Uygulama |
| --- | --- |
| macOS | `Name.app` — `Contents/MacOS` programı ve yardımcılarını, `Contents/Resources` simgeyi (`AppIcon.icns`), `Contents/Resources/app` paketli resimleri içerir |
| Windows | `Name.exe`, `runtime/` ve paketli resimler için `resources/` içeren bir `Name/` klasörü ve aynısı `Name-windows-x64.zip` olarak |
| Linux | `Name`, `runtime/` ve paketli resimler için `resources/` içeren bir `Name/` klasörü ve aynısı `Name-linux-x64.tar.gz` olarak |

Bir uygulama tam olarak şunları içerir:

- derlenmiş program;
- programın kullandığı ve derlenmiş programın kendisinden belirlenen paketli
  yardımcılar: GUI için `ahdgui`; Graphics için `ahdgraphics`; Plot için
  `ahdplot`, `ahdplotview` ve (görüntüleyicinin Save iletişim kutusu için)
  `ahdgui`; SQLite için `ahdsqlite`; Numeric için `ahdnumeric`; Latex için
  LaTeX motoru;
- `ahdcode-app.json` (ad ve simge dosyasının adı), simge ve macOS'ta
  `Info.plist`, `PkgInfo` ve `AppIcon.icns`.

AhdCode derleyicisini, Go araç zincirini, dil sunucusunu, belgeleri veya
AhdDataStudio'yu içermez. Giriş modülünün yanından yalnızca o modülün
yanındaki `resources` klasöründeki `.png`, `.jpg` ve `.jpeg` dosyaları
kopyalanır. Bu resimler uygulamanın kendi kaynak klasörüne gider: macOS'ta
`Contents/Resources/app`, Windows ve Linux'ta programın yanındaki
`resources`. O klasördeki başka bir dosya, `.env`, `.git`, veritabanları,
hesap tabloları, CSV dosyaları, günlükler ve kaynak dosyalar dahil,
paketlemeyi durdurur. O klasördeki bir sembolik bağlantı, kendisi sembolik
bağlantı olan bir `resources` klasörü dahil, paketlemeyi durdurur. Bağlantı
izlenmez; bu yüzden paketleme klasörün dışında yaşayan bir dosyayı
kopyalayamaz.
Bir veri dosyasına ihtiyaç duyan program onu oluşturur
ya da [GUI iletişim kutularıyla](GUI_TR.md#dosya-ve-mesaj-iletişim-kutuları)
sorar. `GUI.resource` ile yüklenen bir resim bu klasörü kullanır.
`image("mark.png")` yine çalışma dizinini kullanır. Hiçbir şey
yazılmadan önce bir sızıntı denetimi, başka bir dosya, yasak adlı bir dosya
veya gizli bilgiye benzeyen meta veri içerecek uygulamayı reddeder. Derlenmiş
programın içine bakamaz: programın kendi kaynağına yazılmış bir parola
programın parçasıdır.

Bir uygulamayı yeniden paketlemek, daha önce yaptığı uygulamanın yerine
geçer. Aynı yerde `ahdcode package`'ın yapmadığı bir klasör hiçbir zaman
silinmez; başka bir `--output` veya `--name` seçin.

## AhdCode olmadan çalışma

Paketlenmiş bir program yardımcılarını yalnızca kendi uygulamasının içinde
bulur — yürütülebilir dosyasının yanında veya `runtime/` içinde. Hiçbir
zaman bir AhdCode kurulumunu, `PATH`'i, `AHDCODE_ROOT`'u veya
`AHDCODE_*_RUNTIME` değişkenlerini kullanmaz; bu yüzden uygulama herhangi
bir yere ve aynı platformdaki başka bir bilgisayara kopyalanabilir. Eksik
bir yardımcı, uygulamada eksik olarak bildirilir.

macOS Finder'dan açılan bir uygulama kök klasör yerine kullanıcının ana
klasöründe başlar; böylece `"ledger.db"` gibi göreli bir yol ana klasördeki
bir dosya anlamına gelir. Bir terminalden başlatıldığında, her program gibi
terminalin geçerli klasörünü kullanır. Paketli resimler bu klasörü kullanmaz.
`GUI.resource` uygulamanın kaynak klasörünü okur.

## Ad ve simge

Uygulamanın pencereleri — GUI pencereleri, Graphics tuvalleri, Plot
görüntüleyicileri ve iletişim kutuları — AhdCode'unkini değil, uygulamanın
adını ve simgesini gösterir:

- macOS: paketin `Info.plist` dosyası uygulamayı adlandırır (tanımlayıcı
  `org.ahdcode.app.<ad>`), Dock ise `AppIcon.icns`'i gösterir. Paketin ana
  programı kendi penceresini açmaz — yardımcıları açar — bu yüzden paket bir
  ajan (`LSUIElement`) olarak işaretlenir ve yardımcı pencereler uygulamanın
  kimliğini taşır. Varsayılan AhdCode simgesine macOS'un yuvarlatılmış biçimi
  verilir; `--icon` ile verilen simge olduğu gibi kullanılır.
- Windows: simge `Name.exe` içine bağlanır ve yardımcı pencereler onu
  kullanır. GUI, Graphics veya Plot kullanan bir program konsol penceresi
  açmaz; birini korumak için `--console` kullanın.
- Linux: yardımcı pencereler, masaüstünün pencere simgesi gösterdiği yerde
  simgeyi kullanır.

## Hedefler

| Hedef | Çıktı | Not |
| --- | --- | --- |
| `macos-arm64`, `windows-x64`, `linux-x64`, `linux-arm64` | paket ve platform arşivi | standart hedefler (`--target` varsayılan olarak geçerli bilgisayarı seçer) |
| `macos-x64`, `windows-arm64` | `.app`, klasör veya `.zip` | uyumlu `--helpers` dizini verildiğinde çapraz derlenen hedefler |

Varsayılan olarak `ahdcode package`, çalıştığı bilgisayar için bu kurulumun
yardımcılarıyla paketler. Bir program C kodu içermeyen düz Go'dur; bu yüzden
AhdCode onu başka bir platform için de derleyebilir; ancak yardımcılar her
biri tek bir platform için derlenir. Başka bir platform için paketlemek üzere
`--target`'ı, hedef için derlenmiş uyumlu yardımcıların klasörünü gösteren
`--helpers` ile verin:

```sh
ahdcode package app.ahd --target windows-x64 --helpers ./helpers/windows-x64
```

Kurulum programı, `.msi`, `.deb`, `.rpm`, AppImage veya Flatpak çıktısı
yoktur.

## macOS: imzalama ve Gatekeeper

AhdCode bir Apple geliştirici hesabı istemez. Apple Silicon'da her programın
bir imzası olmalıdır; bu yüzden uygulamanın programı derlenirken geçici
(ad hoc) olarak imzalanır ve paketli yardımcılar AhdCode'un kendi
imzalarını korur; paketin tamamı bir geliştirici sertifikasıyla imzalanmaz
veya noter onayından geçirilmez.

Paketlediğiniz uygulama kendi Mac'inizde açılır. Tarayıcı indirmesi, AirDrop
veya e-posta ile başka bir Mac'e kopyalandığında macOS onu indirilmiş olarak
işaretler ve Gatekeeper noter onayı olmayan bir uygulamayı açmayı reddeder;
alıcı ona **Sistem Ayarları › Gizlilik ve Güvenlik** bölümünden bir kez izin
verebilir. Kendi Developer ID'nizle imzalamak ve noter onayı almak
`ahdcode package`'ın kapsamı dışındadır.

## Hatalar

Bir paketleme sorunu `PKG001` koduyla bildirilir ve hiçbir şey yazılmaz:
geçersiz bir ad veya simge, bilinmeyen bir hedef, eksik bir yardımcı,
`--helpers` olmadan başka bir platform hedefi, yolda duran bir klasör veya
başarısız bir sızıntı denetimi. Programdaki derleme hataları `ahdcode build`
için olduğu gibi bildirilir.
