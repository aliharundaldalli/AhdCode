# Progresif Web Uygulamaları

[English](PWA.md)

Progresif Web Uygulaması (PWA), destekleyen tarayıcılarda kurulabilen ve
kendi penceresinde açılabilen bir web sitesidir. Manifest uygulamanın adını,
simgelerini ve başlatma davranışını tanımlar. Service worker, seçili istekleri
çevrimdışı karşılayabilir. AhdCode sayfaları `.ahd`, Web.UI ve HTMLNode
ile oluşturulmaya devam eder. Kurulum tarayıcıya ve platforma bağlıdır.
Uygulama tarayıcı içinde çalışır. Sunucu mantığı, kimlik doğrulama, veritabanı
sorguları ve işlemler için bağlantı gerekir.
AhdCode bu özellik için web varlıkları üretir. APK veya IPA üretmez.

## Projeye PWA ekleme

Proje kökünde çalıştırın:

```sh
ahdcode pwa init --name "Örnek Site" --short-name "Örnek"
```

Varsayılan ad klasör adıdır. Kısa ad belirtilmezse uygulama adı kullanılır.
Komut bir `PWA/` klasörü oluşturur, mevcut bir klasörü değiştirmez.

1. Giriş dosyasına ve ortak yerleşime `require("PWA/Integration.ahd")`
   ekleyin. Basic starter yerleşimi `Layouts/Main.ahd` dosyasıdır.
2. Web App nesnesini oluşturduktan sonra, `site.start()` çağrısından önce
   `ahdPwaMount(site)` çağırın.
3. Yerleşimin başlık düğümlerine `ahdPwaManifest()` ve
   `ahdPwaRegistration()` ekleyin. Bu düğümleri Basic'in başlık listesine
   ya da `Web.page` ve `Web.document` çağrılarının başlık argümanına
   ekleyebilirsiniz.

Entegrasyon mevcut Web ve HTTP API'lerini kullanır. Manifest ve worker
belirli adreslerde doğru medya türleriyle sunulur. Genel PWA varlıkları
ayrı `PWA/public` klasöründen servis edilir.
Manifesti, worker'ı ve genel dosyaları derlenmiş uygulamayla dağıtmak için
`ahdcode package app.ahd --web --pwa` kullanın. [Web paketleme](PACKAGING_TR.md).
Kaynak çizgesi aynı HTTP türlerini doğrudan aktarıyorsa
bu türler için tüm çizgede tek bir aktarım kaynağı kullanın.

## Kurulum ve dağıtım

Ad, kısa ad, başlangıç adresi, kapsam, görünüm, renkler ve simgeler için
`PWA/app.webmanifest` dosyasını düzenleyin. Varsayılan başlangıç adresi
ve kapsam `/`, görünüm `standalone`, simgeler 192 ve 512 piksellik PNG
dosyalarıdır. Her uygulama için ayrı bir origin kullanın. Üretilen worker
kök kapsamı kullanır. Kurulum seçeneği tarayıcı menüsünde veya adres
çubuğunda görünebilir. Yalnızca manifestin yüklenmesi, kurulum seçeneğinin
sunulduğunu göstermez. [Web App Manifest](https://www.w3.org/TR/appmanifest/).

Kayıt için güvenli bir üst düzey bağlam gerekir. Yerel HTTPS için
`ahdcode dev --https app.ahd` çalıştırın ve yerel CA'ya test tarayıcısında
güvenin. Tarayıcıların güvenilir saydığı loopback HTTP adresleri de
çalışabilir. Düz HTTP `.test` adresi güvenli sayılmaz. Üretimde
`Tarayıcı HTTPS → Caddy → AhdCode HTTP` düzenini kullanın.
`APP_PROTOCOL=https` ayarlayın ve PWA adreslerini uygulama yollarıyla
birlikte değiştirmeden yönlendirin.
[Caddy ters vekil sunucusu](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy).

## Çevrimdışı kullanım ve önbellek

Worker, sorgu dizgesi içermeyen `/` sayfa isteğinde önce ağı kullanır.
Bağlantı kesilirse genel bir çevrimdışı sayfası gösterir. Sunucudan gelen
HTTP hataları ve yönlendirmeler korunur. Diğer sayfalar, API istekleri,
WebSocket bağlantıları, yazma işlemleri ve AhdCode canlı yenileme
trafiği tarayıcının normal ağ işlemlerine bırakılır. Çalışma sırasında
sunucudan gelen yanıtlar Cache Storage'a kaydedilmez.
Tarayıcının normal HTTP önbelleğini de denetlemek için özel uygulama
yanıtlarında `Cache-Control: no-store` kullanın.

Yalnızca worker'ın `PUBLIC_ASSETS` listesinde yer alan herkese açık
dosyalar önceden önbelleğe alınır. Başlangıçta listede iki simge bulunur.
Her dosya için medya türü ve SHA-256 bütünlük değeri gerekir. Önden indirme
kimlik bilgilerini göndermez ve yönlendirmeleri reddeder. Worker kapsamı,
sürümü ve varlık listesi kendi önbellek adını belirler. Yeni sürüm etkin
olunca yalnızca kendi eski önbellekleri silinir. Etkinleşme için eski worker'ın
denetlediği sekmeleri kapatın. Önceden önbelleğe alınacak dosyaları basit
dosya adlarıyla doğrudan `PWA/public` altına koyun. Bir dosya değişince
`PWA/worker.js` içindeki bütünlük değerini güncelleyin.
[İngilizce SHA-256 hesaplama örneği](PWA.md#offline-behavior-and-cache-policy).
Özel yanıtları, oturum verilerini ve token'ları listeye eklemeyin.
[Service Workers](https://w3c.github.io/ServiceWorker/).

Bağlantı geri dönünce sunucu işlevleri için sayfayı yeniden yükleyin.
Çevrimdışı davranışı ayrı bir tarayıcı profiliyle sınayın. Özelliği yayından
kaldırdığınızda mevcut worker kaydını ve ilgili önbellekleri de kaldırın.
