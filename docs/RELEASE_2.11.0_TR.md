# AhdCode v2.11.0 — Web Dağıtımı ve Progresif Web Uygulamaları

[English](RELEASE_2.11.0.md)

AhdCode `v2.11.0`; macOS, Windows ve Linux için araç zinciri dağıtım
paketleri sunar ve `v2.11.0` etiketli kaynaktan `go.mod` içindeki Go
sürümüyle de derlenebilir:

```sh
go build -o ahdcode ./cmd/ahdcode
go build -o ahdsqlite ./cmd/ahdsqlite
./ahdcode version
```

Windows'ta `ahdcode.exe` ve `ahdsqlite.exe` adlarını kullanın.
Sürüm komutu `AhdCode v2.11.0` gösterir. Kaynaktan derlediğinizde uyumlu
yardımcıları CLI'ın yanında tutun. Araç zinciri paketleri yerel
`ahdcode build` ve `ahdcode package` komutları için `PATH` üzerinde
`go.mod` içindeki Go sürümünü gerektirir; özel Go, çevrimdışı Tectonic LaTeX
ve VSIX içeren tam kendi kendine yeten paketler `v2.7.0`'da mevcuttur ve
`v3.0`'da geri döner. `ahdcode package app.ahd --web` ile üretilen Web
uygulama paketi ayrı bir dağıtım çıktısıdır.

## Çoklu platform dağıtım paketleme

`tooling/distribution/build.py`; macOS, Windows ve Linux için yerel araç
zinciri paketleri üretir:

- **macOS (`arm64` ve çapraz derlenen `x64`):**
  - `AhdCode-2.11.0-macos-<mimari>.pkg`: yönetici şifresi istemeyen
    (`auth="none"`) ev dizini `.pkg` kurucusu; `2.9.0` gibi mevcut sürümleri
    korur, `current` bağını ve `~/Library/AhdCode/bin/ahdcode` yolunu
    günceller, kabuk PATH bloklarını idempotent biçimde ekler ve kaldırma için
    `install.sh` içerir. Apple Developer ID Installer ile imzalı, Apple noter
    onaylı ve bilet zımbalıdır.
  - `AhdCode-2.11.0-macos-<mimari>.dmg`: `.pkg`, `INSTALL.txt` ve `VERSION`
    dosyalarını içeren UDZO disk imajı. Apple Developer ID Application ile
    imzalı ve noter onaylıdır.
  - `AhdCode-2.11.0-macos-<mimari>.tar.gz`: sertleştirilmiş çalışma zamanı ve
    zaman damgalı Developer ID imzalı Mach-O ikilileri içeren taşınabilir arşiv
    + `install.sh`.
- **Windows (`x64`):** `AhdCode-2.11.0-windows-x64.exe` kullanıcı başına
  kurulum programı ve sabit başlatıcı (`--silent` destekli). Authenticode
  imzalı değildir; çalıştırmadan önce SHA-256 özetini doğrulayın.
- **Linux (`amd64` ve `arm64`):** `AhdCode-2.11.0-linux-amd64.tar.gz` ve
  `AhdCode-2.11.0-linux-arm64.tar.gz` arşivleri (`install.sh` ile `--setup-path`,
  `--prefix`, `--uninstall`).
- **`SHA256SUMS.txt`:** Yayımlanan her dosya için SHA-256 doğrulama listesi.

`--sign-app`, `--sign-installer` ve `--notary-profile` verildiğinde Apple'ın
standart `codesign --verify --deep --strict`, `pkgutil --check-signature`,
`xcrun notarytool submit --wait`, `xcrun stapler staple`/`validate` ve
Gatekeeper `spctl` denetimleri zorunlu olarak çalıştırılır: `.tar.gz`
arşivleri sertleştirilmiş çalışma zamanı ve zaman damgalı Developer ID imzalı
ikililer taşır; `.pkg` ve `.dmg` paketleri ise imzalanır, Apple noter onayından
geçer ve bilet zımbalanır. Noter kimlikleri yoksa çıktılar açıkça `-unsigned`
olarak etiketlenir ve Gatekeeper asla atlatılmaz. Kaldırmak için:

```sh
sh "$HOME/Library/AhdCode/current/install.sh" --uninstall
```

## v2.10.0 sonrasındaki değişiklikler

- Geliştirme için loopback HTTPS ve isteğe bağlı sertifika ile anahtar
  desteği eklendi. Yerel CA'ya tarayıcıda ayrıca güvenilir. Üretimde HTTPS
  sonlandırma Caddy'de kalabilir.
- HTTP ve WebSocket kapanışında geri çağrı sahipliği güçlendirildi. SQLite
  kaynakları kabul edilmiş geri çağrılar bitene kadar açık kalır.
- `ahdcode pwa init` ile tarayıcıdan kurulum ve sınırlı çevrimdışı
  destek eklendi. Worker yalnızca açıkça seçilmiş, bütünlüğü sabitlenmiş
  genel varlıkları önbelleğe alır. Canlı yenileme isteklerine dokunmaz.
- `ahdcode package app.ahd --web` ve isteğe bağlı `--pwa` ile derlenmiş
  uygulama ve gözden geçirilmiş genel varlıklar paketlenebilir. Mevcut
  Web dağıtımları üzerine yazılmaz. Manifest ve worker yolları GET ve
  HEAD kabul eder.
- Genel Templates modülü kaldırıldı. Basic sayfaları, yerleşimleri ve
  bileşenleri `.ahd`, Web.UI ve HTMLNode ile oluşturulur. Önceki
  Templates API'sini kullanan uygulamalar geçiş yapmalıdır.

Tarayıcı kurulumu ve çevrimdışı davranış platforma bağlıdır. Özel
yapılandırmayı, kimlik bilgilerini ve veritabanlarını dağıtımın yönettiği
depolamada saklayın. [Paketleme](PACKAGING_TR.md), [PWA kurulumu](PWA_TR.md)
ve [Templates geçişi](UPGRADE_2.11_TR.md).
