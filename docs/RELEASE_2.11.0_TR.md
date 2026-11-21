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
uyumlu yardımcıyı CLI'ın yanında tutun. Mevcut imzalı kurucular v2.7.0'da
kalır. Araç zincirinin ürettiği Web uygulama paketi ayrı bir çıktıdır.

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
