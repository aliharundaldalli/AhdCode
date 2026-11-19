# AhdCode v2.10.0 — Web Geliştirme ve Uygulama Güvenilirliği

[English](RELEASE_2.10.0.md) · [Türkçe]

v2.10.0 kararlı bir kaynak sürümüdür. GitHub ZIP ve tar.gz arşivlerini
sunar. Son imzalı kurulum paketleri v2.7.0'a aittir. v2.10.0'ı etiketli
kaynaktan go.mod dosyasında belirtilen Go sürümüyle derleyin. Windows'ta
çıktı adı `ahdcode.exe` olmalıdır. `ahdcode version` komutu `AhdCode v2.10.0`
yazar. SQLite kullanan yerel programlar için aynı kaynaktan derlenen
`ahdsqlite` yardımcısı gerekir.

## Web ve HTTP

- `ahdcode dev` başarılı derlemelerden sonra yerel geliştirme istemcilerine
  sunulan uygun HTML sayfalarını yeniler. `templates/` altındaki HTML
  dosyalarında oluşturma, düzenleme, silme, geri yükleme ve yeniden
  adlandırmayı izler.
- HTTP, WebSocket ve değerlendiricinin eşzamanlaması iyileştirildi.
- `HTTP.persistentSessions` ve `Web.persistentSessions` kaydedilen oturum
  verilerini ve iptal bilgilerini SQLite'ta saklar. Bu bilgiler süreç
  yeniden başlatıldığında korunur. Oturum kimliği yenilendiğinde eski
  kimlik geçersiz olur. Bellek içi oturumlar kullanılmaya devam edilebilir.
- `server.notFound` ve `server.internalError` özel HTTP 404 ve 500
  sayfaları sunar. İşleyici hata verirse varsayılan yanıt döner.
- `ahdcode init web basic` Türkçe sayfalar, ortak bir gezinme bileşeni,
  paylaşılan yerleşim ve özel hata sayfaları içeren bir site oluşturur.

## Şema göçü ve şablonlar

```sh
ahdcode migrate new create_notes
ahdcode migrate status --db notes.db
ahdcode migrate up --db notes.db
ahdcode migrate down --db notes.db --steps 1
```

`new` komutu `migrations/` altında ileri ve geri SQL dosyalarını oluşturur.
Uygulamadan önce ikisini de düzenleyin. Var olan veritabanını yedekleyin.
Kalıcı oturumlar için ayrı bir SQLite dosyası kullanın.

```ahd
bring Templates
from Templates bring TemplateSet

pages: TemplateSet := Templates.files("templates")
markup: String := pages.layout("layouts/site.html", "pages/home.html", Templates.object({
    "title": Templates.fromString("Ana sayfa")
}))
```

Yerleşimdeki `{{block "content" .}}{{end}}` bloğunu sayfadaki
`{{define "content"}}<h1>{{.title}}</h1>{{end}}` tanımı doldurur. Sıradan
veriler HTML için kaçışlanır. `layout` çağrısının sayfa adı `pages/` altında
olmalıdır. Her dosya kendi şablon adına sahiptir. Paylaşılan dosyalar
`pages/` altında tanım yapamaz. Bir geliştirme derlemesinden geçerken
çakışan tanımları yeniden adlandırın.

## Bilinen sınırlamalar

- Şema göçü komutları SQLite ile çalışır.
- Windows'ta tam test çalıştırmalarında çalıştırılabilir dosya yolu,
  izinler, simgesel bağlantılar ve platforma özel test düzenekleriyle
  ilgili bilinen hatalar vardır. Bağlantı oluşturma izni yoksa bazı
  testler atlanır.
- Windows'ta şablon dosyası açılırken son bileşendeki reparse noktaları
  reddedilir. Üst dizinler yol üzerinden denetlendiği için denetim ile
  kullanım arasında dizin değiştirilebilir.
- `ahdcode package` harici `templates/` ve `public/` dizinlerini otomatik
  paketlemez. Bu dizinleri uygulamayla birlikte dağıtın ve çalışma dizinini
  buna göre ayarlayın.
- Son bağımsız sürüm denetimi engelleyici olmayan üç küçük bulguyla geçti:
  HTTP/WebSocket `closing` denetimi ile callback girişi arasındaki dar kapanış
  TOCTOU penceresi (veri yarışı, SQLite bozulması veya kalıcı tanıtıcı
  sızıntısı oluşturmaz; ileride sıkılaştırılacaktır), eşzamanlı `net.Listen`
  bağlanma hatasında sahiplik temizlenmeden önce oluşan dar dinleme-öncesi
  kapanış yarışı ve yalnızca testte görülen mevcut
  `TestWebSocketClockIsImmutablePerConnection` zamanlama kararsızlığı
  (üretim kodu WebSocket saatini hiçbir zaman değiştirmez).
- Yerel HTTPS, PWA, daha kapsamlı çalışma zamanı birleştirmesi ve sonraki
  dil belirtimi çalışmaları ertelendi.

[CLI](CLI_TR.md) · [HTTP](HTTP_TR.md) · [Şablonlar](TEMPLATES_TR.md) · [Web](WEB_TR.md)
