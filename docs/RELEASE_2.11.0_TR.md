# AhdCode v2.11.0 — Web Dağıtımı ve Progresif Web Uygulamaları

[English](RELEASE_2.11.0.md)

Araç zinciri kaynak kod olarak dağıtılır. v2.11.0 kaynaklarını
`go.mod` dosyasında belirtilen Go sürümüyle derleyin:

```sh
go build -o ahdcode ./cmd/ahdcode
go build -o ahdsqlite ./cmd/ahdsqlite
./ahdcode version
```

Windows'ta `ahdcode.exe` ve `ahdsqlite.exe` adlarını kullanın.
Sürüm komutu `AhdCode v2.11.0` gösterir. SQLite gereken uygulamalar için
uyumlu yardımcıyı CLI'ın yanında tutun. Yayımlanmış mevcut imzalı kararlı
kurucular, nihai imzalı sürüm yayımlanana kadar v2.7.0'da kalır. Araç
zincirinin ürettiği Web uygulama paketi ayrı bir çıktıdır.

## macOS dağıtım paketleme (.pkg, .dmg, .tar.gz)

`tooling/distribution/build.py --platform darwin`, Windows ve Linux dağıtım
davranışını değiştirmeden macOS ARM64 (ve isteğe bağlı çapraz derlenen
`x86_64`) araç zinciri paketleri üretir:

- `AhdCode-<sürüm>-macos-<mimari>[-unsigned].tar.gz`: `install.sh` içeren ve
  `~/Library/AhdCode/versions/<sürüm>` altına kuran taşınabilir arşiv.
- `AhdCode-<sürüm>-macos-<mimari>[-unsigned].pkg`: yönetici şifresi istemeyen
  (`auth="none"`) ev dizini `.pkg` kurucusu; `2.9.0` gibi mevcut sürümleri
  korur, `current` bağını ve `~/Library/AhdCode/bin/ahdcode` yolunu günceller,
  kabuk PATH bloklarını idempotent biçimde ekler ve kaldırma için
  `install.sh` içerir.
- `AhdCode-<sürüm>-macos-<mimari>[-unsigned].dmg`: `.pkg`, `INSTALL.txt` ve
  `VERSION` dosyalarını içeren UDZO disk imajı.
- `SHA256SUMS.txt`: üretilen her dosya için SHA-256 doğrulama listesi.

`--sign-app`, `--sign-installer` ve `--notary-profile` verildiğinde Apple'ın
standart `codesign`, `productbuild --sign`, `xcrun notarytool` ve
`xcrun stapler` iş akışı kullanılır. Noter kimlikleri yoksa çıktılar açıkça
`-unsigned` olarak etiketlenir ve Gatekeeper asla atlatılmaz. Kaldırmak için:

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
