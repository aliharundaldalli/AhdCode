# Kurulum, yükseltme ve kaldırma

RC paketleri bağımsız QA için adaydır; final v1.0.0 henüz yayımlanmamıştır.
Windows canlı kurulum testi tamamlanmadan Windows QA geçti denmez.

macOS Apple Silicon: `AhdCode-1.0.0-rc.1-macos-arm64.dmg` dosyasını açın,
`Install.command` çalıştırın. Kurulum kullanıcı Library dizinindeki
`Application Support/AhdCode/versions/<sürüm>` altına yapılır. `current` etkin
sürümü seçer; `bin/ahdcode` sabit başlatıcıdır. `~/.zprofile` içine yalnızca
AhdCode'a ait, kaldırılabilir bir PATH bloğu eklenir. Yeni giriş kabuğunda
`ahdcode --version` çalıştırın. RC henüz Developer ID imzalı/noter onaylı değildir;
normal güvenilir macOS dağıtımı için bu işlem final öncesinde tamamlanmalıdır.

Windows x64: `AhdCode-1.0.0-rc.1-windows-x64.exe` dosyasına Dosya
Gezgini'nde çift tıklayın. Kurulum küçük bir grafik programdır: ne kuracağını
gösterir, gömülü paketi ilerleme penceresiyle açıp doğrular ve bir onay
penceresiyle biter. Konsol, terminal veya komut yazmak gerekmez.

Dosyalar `%LOCALAPPDATA%\AhdCode\versions\<sürüm>` altına kurulur. Sabit komut
`%LOCALAPPDATA%\AhdCode\bin\ahdcode.exe` olup kullanıcı PATH değerine yalnızca
bu tek dizin, yalnızca ilk kurulumda eklenir. Yönetici izni, Git veya sistem Go
gerekmez; kaldırma kaydı Installed Apps içine yazılır.

Ardından **yeni** bir PowerShell veya Komut İstemi açıp `ahdcode --version`
çalıştırın. Zaten açık olan bir terminal başlatıldığı ortamı korur; bu Windows'un
normal davranışıdır ve yeni açılan terminal değişikliği hemen görür.

`AhdCode-1.0.0-rc.1-windows-x64.exe --silent` hiçbir pencere açmadan kurar.
Kurulum grafik bir program olduğu için, betikten çağırırken bitmesini beklemek
isterseniz `Start-Process -Wait` kullanın.

Windows canlı kurulum/kaldırma QA gereklidir; RC imzasızdır.

Linux x64: `AhdCode-1.0.0-rc.1-linux-x64.tar.gz` arşivini açın ve
`sh install.sh --setup-path` çalıştırın. Kök `~/.local/share/ahdcode` dizinidir;
PATH bloğu `~/.profile` dosyasına eklenir.

Paket; özel Go 1.27.0, `ahdsqlite`, `ahdnumeric`, `ahdplot`, çevrimdışı Tectonic
0.17.0 ve 5.546.077 baytlık kaynak paketini içerir. CLI; sürüm-eş Studio,
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
sh "$HOME/Library/Application Support/AhdCode/current/install.sh" --uninstall
```

Linux için yol `~/.local/share/ahdcode/current/install.sh` olur. Windows'ta
Installed Apps veya kurulum kökündeki `uninstall.ps1` kullanılır; `YES` onayı
istenir. Yalnızca ürün kökü ve kendi PATH girdisi kaldırılır. Projeler,
veritabanları, ürün kökü dışındaki `.env`, kaynak depoları ve kullanıcı
registry/cache verileri korunur. Önce AhdCode uygulamalarını kapatın.

İzole macOS/Linux QA için `--prefix /mutlak/yeni/kurulum-koku` kullanın;
kabuk profili değişmez. Aynı prefix ve `--uninstall` ile kaldırın.
Paket üretimi ve ayrıntılar: [Installation](INSTALLATION.md).
