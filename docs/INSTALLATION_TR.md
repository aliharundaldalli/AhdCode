# Kurulum, yükseltme ve kaldırma

İşletim sisteminize uygun AhdCode v2.7.0 paketini indirin. Her paket
kendi kendine yeterlidir: derleyici, özel Go araç zinciri, AhdDataStudio,
SQLite/numeric/plot yardımcıları, çevrimdışı LaTeX motoru, proje starter'ları
ve İngilizce belgeler paketin içindedir.

## macOS (Apple Silicon)

macOS paketi Apple Silicon Mac'leri hedefler — M1, M2, M3, M4 ve sonraki
arm64 modeller. Intel sürümü yoktur.

`AhdCode-2.7.0-macos-arm64.pkg` dosyasına çift tıklayıp kurulumu izleyin. Paket
yayımlanan önerilen kurulum biçimidir. Developer ID ile imzalanmış ve Apple
tarafından noterlenmiştir; noter bileti pakete zımbalanmıştır, bu yüzden
Gatekeeper uyarı göstermeden kabul eder. Yalnızca sizin hesabınıza kurar,
yönetici şifresi istemez ve ev dizininizin dışına hiçbir şey yazmaz. İndirilen
dosyayı yayımlanan SHA-256 değeriyle doğrulayın; sistem korumalarını asla
kapatmayın veya atlatmayın.

Dosyalar `~/Library/AhdCode/versions/2.7.0` altına kurulur. `current` etkin
sürümü seçer, sabit komut `~/Library/AhdCode/bin/ahdcode`'dur. PATH'e yalnızca
bu tek dizin eklenir.

Ardından **yeni** bir Terminal açıp `ahdcode --version` çalıştırın. Zaten açık
olan bir terminal başlatıldığı ortamı korur; yeni açılan değişikliği hemen
görür.

`AhdCode-2.7.0-macos-arm64.dmg`, disk imajı tercih edenler için aynı içeriğe
sahip, imzalı ve noterlenmiş bir disk imajıdır: açın ve `Install.command`'ı
çalıştırın (`install.sh --setup-path` çalıştırır). Önerilen kurulum biçimi
`.pkg`'dir.

## Windows x64

`AhdCode-2.7.0-windows-x64.exe` dosyasına Dosya Gezgini'nde çift tıklayın.
Kurulum küçük bir grafik programdır: ne kuracağını gösterir, gömülü paketi
ilerleme penceresiyle açıp doğrular ve bir onay penceresiyle biter. Konsol,
terminal veya komut yazmak gerekmez.

Bu sürümün kurulum programı Authenticode imzalı değildir ve Windows SmartScreen “yayıncı
bilinmiyor” uyarısı gösterebilir. Sistemin **Daha fazla bilgi → Yine de
çalıştır** yolunu kullanmadan önce dosyayı yayımlanan SHA-256 değeriyle
doğrulayın; Windows güvenlik korumalarını kapatmayın.

Dosyalar `%LOCALAPPDATA%\AhdCode\versions\2.7.0` altına kurulur. Sabit komut
`%LOCALAPPDATA%\AhdCode\bin\ahdcode.exe`'dir ve kullanıcı PATH'ine yalnızca bu
tek dizin, yalnızca ilk kurulumda eklenir. Yönetici izni, Git veya sistem Go
kurulumu gerekmez; kaldırma kaydı Installed Apps içine yazılır.

Ardından **yeni** bir PowerShell veya Komut İstemi açıp `ahdcode --version`
çalıştırın.
Kurulum, kullanıcı ortam değişikliğini yayınlar; ancak zaten açık terminaller
ve IDE'ler başladıkları ortamı korur. Entegre terminalden denemeden önce bu
programları yeniden başlatın. Yeni bir kabukta sabit başlatıcının bulunduğunu
`where.exe ahdcode` ile doğrulayabilirsiniz.

Onarım veya yükseltme güvenlidir: kurulum programını yeniden çalıştırmak mevcut
PATH girdilerini aynen korur ve eşdeğer tek bir AhdCode girdisi bırakır.
Kaldırma yalnızca eşdeğer `%LOCALAPPDATA%\AhdCode\bin` girdilerini kaldırır,
diğer PATH girdilerine ve makine PATH'ine dokunmaz.

`AhdCode-2.7.0-windows-x64.exe --silent` hiçbir pencere açmadan kurar.

Linux (x86-64 ve arm64): işlemcinize uygun paketi seçin — `uname -m`, `amd64`
için `x86_64`, `arm64` için `aarch64` yazdırır:
`AhdCode-v2.7.0-linux-amd64.tar.gz` (64 bit Intel/AMD) ya da
`AhdCode-v2.7.0-linux-arm64.tar.gz` (64 bit ARM; örneğin Ampere, AWS Graviton,
64 bit işletim sistemli Raspberry Pi 4/5). Arşivi açın, `AhdCode-2.7.0`
dizinine girin ve `sh install.sh --setup-path` çalıştırın. Kurulum programı
hiçbir şey kurmadan önce paketin bu makinede çalıştığını denetler. Kök
`~/.local/share/ahdcode` dizinidir; PATH bloğu `~/.profile` dosyasına eklenir.

VS Code eklentisi: her pakette kurulum kökü altındaki `vscode/` klasöründe bir
`.vsix` dosyası ve kısa bir `README.txt` bulunur. AhdCode için gerekli değildir
ve kurulum onu sizin yerinize yüklemez. VS Code'da Eklentiler görünümünü açın,
`...` menüsünden **Install from VSIX...** seçin ve kurulum kökündeki
`vscode/ahdcode-0.2.3.vsix` dosyasını gösterin. Antigravity IDE aynı işlemi
sunar. Dosya gereken her şeyi taşır; npm veya ağ gerekmez.

Paket; özel Go 1.27.0, `ahdsqlite`, `ahdnumeric`, `ahdplot`, `ahdgraphics`, `ahdgui`, `ahdplotview`, çevrimdışı Tectonic
0.17.0 ve sabitlenmiş kaynak paketini içerir; paket, TikZ'i, dokuz TikZ
kütüphanesini ve pgfornament'i ekleyen v1.2.0'ın üzerine fancyhdr ve lastpage'i
ekleyen v1.3.0'dan beri 6.350.367 bayttır. CLI; sürüm-eş Studio,
starter, Bootstrap, framework ve İngilizce proje belgelerini gömer. Sistem Go
kurulumu ve ayarları değişmez. Temel derleme, SQLite, Studio ve LaTeX çevrimdışıdır.
MySQL/SMTP için yapılandırdığınız harici sunucular gereklidir. Lisanslar paketin
`THIRD_PARTY_NOTICES.md` ve `licenses/` dizinindedir.

```sh
ahdcode --version
mkdir demo
cd demo
ahdcode init web mvc
ahdcode dev app.ahd
```

Başka terminalde `ahdcode databases` Studio'yu açar. Başka boş dizinde
`ahdcode init web crud` aynı uygulamayı daha düz kaynak düzeniyle oluşturur.
SQLite otomatik kaydedilir. `AHDCODE_ROOT` normal kullanımda gerekmez.
`.test` adları mevcut izin akışına bağlıdır; izin yoksa yazdırılan loopback URL
çalışır. Yerel geliştirme HTTP'dir; her proje için hosts komutu gerekmez.

Yeni sürümü aynı köke kurmak eski sürümü koruyup başlatıcıyı günceller. Aynı
sürümün dosyaları üzerine yazılmaz. Eski Go kurulumundaki CLI silinmez; PATH
sırası hangi CLI'ın çalışacağını belirler.

macOS kaldırma:

```sh
sh "$HOME/Library/AhdCode/current/install.sh" --uninstall
```

Linux için yol `~/.local/share/ahdcode/current/install.sh` olur. Windows'ta
Installed Apps veya kurulum kökündeki `uninstall.ps1` kullanılır; `YES` onayı
istenir. Yalnızca ürün kökü ve kendi PATH girdisi kaldırılır. Projeler,
veritabanları, ürün kökü dışındaki `.env`, kaynak depoları ve kullanıcı
registry/cache verileri korunur. Önce AhdCode uygulamalarını kapatın.

İzole macOS/Linux QA için `--prefix /mutlak/yeni/kurulum-koku` kullanın;
kabuk profili değişmez. Aynı prefix ve `--uninstall` ile kaldırın.
Paket üretimi ve ayrıntılar: [Installation](INSTALLATION.md).
