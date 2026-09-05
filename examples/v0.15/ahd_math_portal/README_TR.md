# Ahd Akademi Matematik

**Yayımlanmış AhdCode v0.15.0** ile hazırlanmış tam yığın matematik portalı. Web/UI, MySQL, oturum, Security, dosya yükleme, SMTP ve HTTP/JSON modüllerini gerçek bir uygulamada sınar; derleyici/framework değiştirilmez. [English](README.md) · [Kabul sonuçları](QA.md) · [v0.16 bulguları](DOGFOOD.md).

## Yerel kurulum ve `.env`

AhdCode v0.15.0, erişilebilir MySQL ve yazılabilir özel yükleme dizini gerekir. Uygulama npm, Node, CDN, uzak JavaScript veya Bootstrap JS kullanmaz.

Proje dizininde, mevcut `.env` dosyasının üzerine yazmadan:

```sh
ahdcode --version
cp .env.example .env  # yalnızca .env henüz yoksa
chmod 600 .env
```

`.env` makineye özeldir ve `.gitignore` tarafından bilinçli olarak dışlanır. Gerçek değerleri Git'e veya belgelere koymayın. Süreç ortamında tanımlı değişkenler, boş olsalar bile `.env` değerlerinden önceliklidir. Her iki giriş dosyası yapılandırmayı çalışma zamanında yükler; derleme gizli değerleri ikiliye gömmez. Ayar değişikliklerinden sonra uygulamayı yeniden başlatın.

Geliştirme değerleri: `APP_NAME=Ahd Akademi Matematik`, `APP_ENV=development`, `APP_HOST=ahdakademi.com`, `APP_PROTOCOL=http`, `SERVER_HOST=127.0.0.1`, `SERVER_PORT=8160`. `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USERNAME`, `DB_PASSWORD`, `DB_SECURITY` değerlerini kendi MySQL kurulumunuza göre girin. Hazır bir veritabanı kullanıcısı/parolası dağıtılmaz.

Yetkili MySQL hesabıyla hedef yerel veritabanını oluşturun:

```sql
CREATE DATABASE IF NOT EXISTS ahd_math_portal
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

Mevcut veritabanında önce tabloları inceleyin. Şema beş InnoDB tabloyu `IF NOT EXISTS` ile oluşturur; mevcut ayar değerlerini değiştirmez. Eski/uyumsuz bir şemayı otomatik taşımadığı için verili veritabanını silerek kurulum yapmayın.

```sh
mysql --host=127.0.0.1 --port=3306 --user=KULLANICINIZ -p ahd_math_portal < database/schema.sql
ahdcode dev app.ahd
```

Normal geliştirme adresi **http://127.0.0.1:8160**. `http://ahdakademi.com.test` yalnızca mantıksal geliştirme kimliğidir; v0.15 `.test` için DNS/çözümleyici kurmaz. Geliştirme parola sıfırlama bağlantıları gerçek dinleme adresini kullanır. MySQL gereklidir; SMTP ve Gemini isteğe bağlıdır.

## Bu Mac'te yerel HTTPS

İsteğe bağlı HTTPS önizlemesi **https://ahdakademi.com.test:8443** adresindedir. Caddy TLS bağlantısını karşılar ve istekleri `127.0.0.1:8161` üzerindeki uygulamaya aktarır. 8443 yönetici yetkisi gerektirmez. `Caddyfile.local` yalnızca loopback üzerinde dinler; dışarıya yayın yapmaz. Caddy yönetim API'si ve otomatik sertifika güveni kapalıdır.

Caddy ve Python 3 yalnızca bu yardımcı önizleme aracının gereksinimleridir; portalın kendisinin çalışma zamanı gereksinimleri değildir.

```sh
brew install caddy
mkdir -p .local
/Users/ahd/go/bin/ahdcode build app.ahd -o .local/portal
python3 scripts/local_https.py start
python3 scripts/local_https.py status
# İşi bitirince yalnızca bu önizlemenin kayıtlı süreçlerini durdurur:
python3 scripts/local_https.py stop
```

HTTPS önizlemesi iç port olarak 8161'i kullandığından varsayılan 8160 HTTP geliştirme sunucusuyla çakışmaz. Yardımcı araç meşgul porttaki başka bir süreci sonlandırmaz. Kaynak kod değişikliğinde önizlemeyi durdurup yeniden derleyin ve başlatın; bu ikili dosya canlı kaynak izleyicisi değildir.

`ahdakademi.com.test` için ayrıca yönetici onayıyla `/etc/hosts` içine `127.0.0.1 ahdakademi.com.test` yönlendirmesi gerekir. AhdCode v0.15 bunu kendiliğinden kurmaz.

Araç `.env` dosyasına yazmaz. Yalnızca başlattığı süreç için `APP_ENV=production`, `APP_HOST=ahdakademi.com.test`, `APP_PROTOCOL=https`, `APP_PUBLIC_PORT=8443` ve loopback dinleme değerlerini ayarlar. Böylece `Secure` oturum çerezleri ve `https://ahdakademi.com.test:8443` parola sıfırlama bağlantıları gerçek HTTPS davranışını sınar. `APP_PUBLIC_PORT` bu uygulamaya ait isteğe bağlı dış port ayarıdır; framework sözleşmesini değiştirmez. Boş/geçersiz değer varsayılan kanonik URL'yi korur.

`.local/` Git tarafından dışlanır; derlenmiş uygulama, süreç kimlikleri, günlükler ve Caddy'nin **özel CA anahtarları** burada kalır. Bu dizini paylaşmayın. Tarayıcı güveni ayrı bir işlemdir; Caddy otomatik olarak güven deposunu değiştirmez. Sertifika uyarısını kaldırmak için proje CA sertifikasını açık onayla yalnızca kullanıcıya, SSL/ahdakademi.com.test kuralıyla güvenilir ekleyebilirsiniz:

```sh
security add-trusted-cert -r trustRoot -p ssl -s ahdakademi.com.test \
  -k "$HOME/Library/Keychains/login.keychain-db" \
  .local/caddy-data/caddy/pki/authorities/local/root.crt
```

Bu güven tercihini kaldırmak için aynı sertifika dosyasıyla `security remove-trusted-cert .local/caddy-data/caddy/pki/authorities/local/root.crt` kullanın. İşlem sertifikanın kullanıcı güven kaydını değiştirir. CA dosyalarını silip yeniden üretirseniz eski güven kaydı yeni CA'yı kapsamaz. Kurulum davranışı için [Caddy yerel HTTPS belgesi](https://caddyserver.com/docs/automatic-https#local-https).

## İlk yönetici

Varsayılan yönetici/parola yoktur. `create_admin.ahd`, portal gibi yerel `.env` dosyasını yükler. E-posta yoksa yönetici oluşturur; varsa yalnızca o hesabın adını/parolasını değiştirir, hesabı etkin yönetici yapar ve `auth_version` değerini artırır. E-postayı dikkatli seçin.

Aşağıdaki örneği Bash içinde kullanın; parola ekrana yazılmaz ve bir komut satırına gömülmez:

```bash
read -r -p 'Yönetici adı: ' ADMIN_NAME
read -r -p 'Yönetici e-posta: ' ADMIN_EMAIL
read -r -s -p 'Parola (en az 10 karakter): ' ADMIN_PASSWORD
printf '\n'
export ADMIN_NAME ADMIN_EMAIL ADMIN_PASSWORD
ahdcode run create_admin.ahd
unset ADMIN_PASSWORD ADMIN_EMAIL ADMIN_NAME
```

Sonra `/login` ve `/admin` adreslerini kullanın. Kabul çalışması geçici kurulum hesabını oluşturup güncelleyerek doğrulamış, ardından yalnızca bu hesabı kaldırmıştır; kalıcı test hesabı bırakmaz. Yardımcı betiğin tüm doğrulama hataları sıfırdan farklı çıkış kodu üretmez; yazdığı sonucu da okuyun.

## Kullanıcılar, sorular ve ayarlar

Kayıt e-postayı normalleştirir; parola en az on karakterdir. E-posta benzersizliğini veritabanı belirler. Parolalar Argon2id ile özetlenir; hatalı parola ve pasif hesap reddedilir. Girişte oturum kimliği yenilenir. Rol/aktiflik/parola sürümü her istekte veritabanından okunur; kullanıcılar yönetim yollarına erişemez.

Yönetici taslak oluşturur/düzenler; yayımlama ayrı bir işlemdir. Genel liste ve `/question?id=1` sorgusu yalnızca yayımlanmış soruları gösterir. `/admin/settings` üzerinden site adı ve yalnızca `#RRGGBB` biçiminde başlık rengi değişir. Gizli bilgiler bu tabloya yazılmaz.

Durum değiştiren her yol oturum tabanlı CSRF doğrulaması yapar. Eksik/yanlış belirteç, gerekli yetki kontrolünden sonra 403 döndürür. Çıkış da CSRF alanı taşıyan POST formudur. `Web.UI` metin/öznitelik kaçışı yapar; kullanıcı ve model metni HTML olarak yürütülmez. Hatalı formda eski metin değerleri gösterilebilir; parola alanları geri doldurulmaz.

## Çözüm yüklemeleri

`UPLOAD_ROOT=storage/solutions`, `public/` dışında tutulur. `UPLOAD_MAX_BYTES=5242880` dosya başına 5 MiB sınırıdır; istek gövdesinde multipart zarfı için ek 64 KiB bulunur. PDF/PNG/JPEG algılanan içeriğe göre kabul edilir; istemcinin uzantısı veya MIME iddiası yeterli değildir. Bu kontrol tam belge doğrulaması ya da zararlı yazılım taraması değildir.

Dosya adını sunucu üretir; yükleyenin adı yalnızca gösterim bilgisidir. `UNIQUE(user_id, question_id)` her kullanıcı/soru çifti için tek çözümü zorunlu kılar. Yenileme/silme/yeniden gönderme arayüzü yoktur. Kayıt eklenemezse yalnızca o isteğin yeni dosyası mümkün olduğunca silinir. Veritabanını ve dosyaları birlikte yedekleyin.

Yalnızca `/assets` → `public/` statik eşlemesi vardır. Özel çözümler statik olarak sunulmaz. `/admin/solutions/file?id=...` dosya okumadan önce yönetici yetkisi ister; normal kullanıcı indiremez. Reverse proxy üzerinden `storage/` dizinini ayrıca yayınlamayın.

## SMTP ve parola sıfırlama

Posta yapılandırılmayacaksa `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM_ADDRESS`, `SMTP_FROM_NAME` boş bırakılır. `SMTP_SECURITY`: `none`, `starttls` veya `tls`. Sunucunun beklediği güvenlik türünü kullanın; bu makinedeki 587 portunda STARTTLS el sıkışması doğrulanmıştır. `SMTP_FROM_NAME` yapılandırmada okunur, fakat gönderilen mesajın görünen adına henüz uygulanmaz.

Bilinen/bilinmeyen hesap ve teslim hatası için aynı genel yanıt gösterilir. SMTP yokken mesaj teslim edilmez; teslim durumu arayüzü yoktur. Bağlantı 30 dakika geçerlidir. Seçici ve gizli doğrulayıcı ayrı tutulur; veritabanında doğrulayıcının yalnızca özeti vardır. Kullanılmış/süresi dolmuş/yanlış doğrulayıcılı bağlantı reddedilir. Başarılı sıfırlama eski oturumları ve diğer sıfırlama kayıtlarını geçersiz kılar. Otomatik kabul testleri yerel SMTP alıcısı kullanır; gerçek posta göndermez.

## İsteğe bağlı Gemini

`GEMINI_API_KEY` ve `GEMINI_MODEL` birlikte tanımlanırsa yönetici taslak paneli açılır. Gerçek anahtar veya sabit model dağıtılmaz. Boşken portalın geri kalanı çalışır. `GEMINI_BASE_URL` yerel HTTP taklidi için isteğe bağlı, işletmeci kontrolündeki uç nokta ayarıdır.

İstek AhdCode HTTP istemcisi ve JSON modülünden geçer; anahtar `x-goog-api-key` başlığındadır. Üretilen metin düzenlenebilir başlık/gövde alanlarına döner; kendiliğinden kaydedilmez veya yayımlanmaz. Yönetici önce taslağı kaydeder, ardından açıkça yayımlar. Bozuk JSON, beklenmeyen veri ve HTTP hatası mesaj olarak gösterilir. Kabul çalışması gerçek/billable üretim çağrısı yapmaz; kullanıcının gerçek model/anahtarının servis tarafından kabul edildiğini iddia etmez.

## Bağımsız çalıştırma ve üretim

```sh
/Users/ahd/go/bin/ahdcode build app.ahd -o /tmp/ahd_math_portal
```

Taşınmış çalışma dizininde yalnızca çalıştırılabilir dosya, `public/`, özel yazılabilir `storage/solutions/` ve çalışma zamanı yapılandırması gerekir. Göreli yollar için bu dizinden başlatın. Şema önceden uygulanır; derleyici deposu, framework `.ahd` dosyaları, Internet, npm veya CDN çalışma zamanı gereksinimi değildir. MySQL yerel bağımlılık olarak kalır; harici posta/AI hizmetleri kendi bağlantılarını gerektirir.

Gerçek üretimde `APP_ENV=production`, herkese açık `APP_HOST` ve `APP_PROTOCOL=https` kullanın. TLS, Caddy/nginx veya uygun bir tünel/reverse proxy üzerinde sonlanır; uygulama iç HTTP soketinde kalır. Soketi yalnızca proxy erişimine açın. Güvenli çerezler herkese açık protokolü, üretim sıfırlama bağlantıları kanonik adresi kullanır. Bu projedeki `Caddyfile.local` Internet'e açık üretim yayını değildir.

## Bootstrap, dogfood ve bilinen sınırlar

Gerçek yerel varlık `public/css/bootstrap.min.css`: **Bootstrap 5.3.3**, 232.803 bayt. Dosya başındaki telif/MIT bildirimi korunur; tam lisans `public/css/bootstrap.LICENSE` içindedir. `public/css/app.css` yerel ek stillerdir. Yerel stiller dışında uzak varlık ve script etiketi yoktur; formlar, etiketler, gezinme ve tablolar semantik `Web.UI` düğümleridir.

Bu bir üretim güvenliği sertifikası değil, referans/dogfood uygulamasıdır. Oturumlar bellektedir; yeniden başlatma oturumları bitirir ve çoklu örnekler oturum paylaşmaz. Dahili hız sınırı, posta kuyruğu/yeniden deneme/teslim gözlemi, çok adımlı sıfırlama işlemi için transaction, zararlı dosya taraması, bütün listelerde sayfalama veya formül dizgisi yoktur. Bazı şema uzunluk sınırları forma yansıtılmamıştır; bazı DB hataları kaba mesajlara dönüştürülür. Bulunamayan genel soru sayfası mesajla HTTP 200 dönebilir. Genel sıfırlama mesajı aynı yanıt süresi garantisi değildir. [DOGFOOD.md](DOGFOOD.md) tekrar sayılarını ve v0.16 için somut öncelikleri kaydeder.
