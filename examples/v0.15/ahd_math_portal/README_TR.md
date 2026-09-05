<p align="center">
  <img src="public/images/logo.png" alt="AhdCode" width="180">
</p>

<h1 align="center">Ahd Akademi Matematik</h1>

<p align="center">
  <strong><a href="https://github.com/aliharundaldalli/AhdCode">AhdCode</a> v0.15.0 ile yazıldı — <a href="https://github.com/aliharundaldalli/AhdCode/releases/tag/v0.15.0">Web Foundations</a></strong>
</p>

<p align="center">
  <a href="https://github.com/aliharundaldalli/AhdCode"><img src="https://img.shields.io/badge/AhdCode-v0.15.0-0d6efd?style=flat-square" alt="AhdCode v0.15.0"></a>
  <a href="https://github.com/aliharundaldalli/AhdCode/blob/main/docs/WEB.md"><img src="https://img.shields.io/badge/bring-Web-198754?style=flat-square" alt="bring Web"></a>
  <a href="https://github.com/aliharundaldalli/AhdCode/blob/main/docs/REQUIRE.md"><img src="https://img.shields.io/badge/require(...)-v0.14-6f42c1?style=flat-square" alt="require"></a>
  <a href="README.md"><img src="https://img.shields.io/badge/Language-English-0d6efd?style=flat-square" alt="English"></a>
</p>

Tamamı AhdCode ile yazılmış, sunucuda üretilen bir matematik portalı.
**v0.15 Web Foundations** için referans uygulama: `bring Web`, `Web.UI`,
Pages / Layouts / Components, `.env` + Config ve v0.14’te gelen
`require(...)` birleşimi.

npm, Node, React, VDOM, ORM veya paket kaydı yok. Derleyici Web
çerçevesini pakete gömer. Derlenen çalıştırılabilir dosya çerçeve
kaynağına bağımlı değildir.

[English](README.md) · [AhdCode](https://github.com/aliharundaldalli/AhdCode) · [v0.15.0 sürümü](https://github.com/aliharundaldalli/AhdCode/releases/tag/v0.15.0)

<p align="center">
  <img src="docs/screenshots/home.png" alt="Ahd Akademi Matematik ana sayfası" width="920">
</p>

## v0.15 gerçek bir uygulamada

| AhdCode yüzeyi | Bu portalda kullanımı |
|---|---|
| `bring Web` / `Web.UI` | Anlamlı sunucu HTML’i: form, nav, tablo, kart. Metin girişleri kaçışlanır. |
| `require("...")` | Config, Repositories, Services, Layouts, Components, Pages olarak bölünmüş tek program |
| `ahdcode dev` | Tüm require grafını izler, kayıtta yeniden derler |
| MySQL | Parametreli sorgular, InnoDB şema, UNIQUE kısıtları |
| Güvenlik | Argon2id parolalar, `secureEqual` ile CSRF, girişte `session.rotate()` |
| HTTP + HTML | Canlı Vikipedi matematik bülteni (dış istemci + kazıma) |
| SMTP | İsteğe bağlı parola sıfırlama; posta kapalıysa yanıt aynı kalır |
| HTTP istemcisi + JSON | İsteğe bağlı Gemini taslak yardımcısı; anahtar URL’ye yazılmaz |
| `Server.static` | Yalnızca `public/` altındaki yerel CSS/JS/görseller |

Arayüz Türkçe. Sorulardaki matematik yerel MathJax ile dizilir.

<p align="center">
  <img src="docs/screenshots/question.png" alt="MathJax ile yayımlanmış soru" width="720">
</p>

<p align="center">
  <img src="docs/screenshots/login.png" alt="Giriş" width="640">
</p>

<p align="center">
  <img src="docs/screenshots/admin.png" alt="Yönetici paneli" width="720">
</p>

<p align="center">
  <img src="docs/screenshots/admin-users.png" alt="Yönetici kullanıcı listesi; ad ve e-posta bulanık" width="720">
</p>

## Düzen

```
app.ahd                 giriş: require(...) grafı, rotalar, Server.static
Config/                 ortamı okuyan tek yer
Support/                satır yardımcıları ve doğrulama
Repositories/           her SQL ifadesi, bağlı parametreler
Services/               kimlik, oturum, CSRF, yükleme, posta, Gemini, kazıma
Layouts/                Main, Auth, Admin kabukları
Components/             navbar, kartlar, formlar, çözüm penceresi
Pages/                  rota başına bir Function
public/                 Bootstrap CSS, app.css, logo, yerel betikler
storage/solutions/      özel yüklemeler — statik dosya olarak eşlenmez
database/schema.sql     beş InnoDB tablo, IF NOT EXISTS
```

Her `require("...")` yolu, yazıldığı dosyaya değil, bu dizine (uygulama
köküne) görelidir. Her dosya kullandığı modülleri kendi getirir.

## Gereksinimler

- Kurulu **AhdCode v0.15.0** (`ahdcode --version`)
- Erişilebilir bir MySQL sunucusu
- Yazılabilir özel yükleme dizini

**`.env.example` → `.env`** kopyalayın. Örnek dosyada veritabanı, SMTP
ve Gemini değerleri bilerek boştur. Yalnızca yerel `.env` dosyanızı
doldurun. `.env` dosyasını veya gerçek parola, API anahtarı, bağlantı
dizgisini örneğe ya da README’ye koymayın.

```sh
ahdcode --version
cp .env.example .env   # yalnızca .env henüz yoksa
chmod 600 .env
```

Süreç ortamındaki değişkenler, boş olsalar bile `.env` değerinden
önceliklidir. `ahdcode build` bunları gömmez: çalıştırılabilir dosya
yapılandırmayı başlangıçta okur.

Şemayı oluşturun, sonra uygulamayı başlatın:

```sql
CREATE DATABASE IF NOT EXISTS ahd_math_portal
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

```sh
mysql --host=127.0.0.1 --port=3306 --user=KULLANICINIZ -p ahd_math_portal < database/schema.sql
ahdcode dev app.ahd
```

[http://127.0.0.1:8160](http://127.0.0.1:8160) adresini açın.

**Varsayılan yönetici yoktur.** Birini etkileşimli oluşturun:

```bash
read -r -p 'Yönetici adı: ' ADMIN_NAME
read -r -p 'Yönetici e-posta: ' ADMIN_EMAIL
read -r -s -p 'Yönetici parolası (en az 10 karakter): ' ADMIN_PASSWORD
printf '\n'
export ADMIN_NAME ADMIN_EMAIL ADMIN_PASSWORD
ahdcode run create_admin.ahd
unset ADMIN_PASSWORD ADMIN_EMAIL ADMIN_NAME
```

## Özellikler

- Her durum değişikliğinde CSRF ile kayıt / giriş / çıkış
- Ana sayfada yayımlanmış sorular; taslaklar yayımlanana kadar gizli
- PDF / PNG / JPEG çözüm yükleme (içerik koklama, 5 MiB, özel depolama)
- Yönetici kullanıcı, soru, ayar ve yetkili çözüm indirme
- İsteğe bağlı SMTP parola sıfırlama (30 dakikalık hash’li jetonlar)
- İsteğe bağlı Gemini taslak — üretmek yayımlamaz
- HTTP + HTML kazıma ile canlı matematik bülteni

## Güvenlik duruşu

Geliştirmede loopback bağlanır. Parolalar Argon2id ile hash’lenir.
Oturum girişte döner. CSRF `Security.secureEqual` ile karşılaştırılır.
Yüklemeler `public/` dışındadır. Yönetici dosya indirmeden önce yetki
kontrol eder. Site ayarları yalnızca görünen değerleri tutar — sır
tutmaz.

Bu bir dogfood referansıdır, üretim sertleştirme iddiası değildir.
Oturumlar bellektedir. Ölçülen sınırlar için [DOGFOOD.md](DOGFOOD.md).

## Derleme

```sh
ahdcode build app.ahd -o ./portal
```

Dağıtım dizininde çalıştırılabilir dosya, `public/`, yazılabilir
`storage/solutions/` ve çalışma zamanı yapılandırması (`.env` veya süreç
ortamı) gerekir. İkiliyi o dizinden başlatın.

## Aynı aile

- [AhdCode](https://github.com/aliharundaldalli/AhdCode) — dil ve derleyici
- [v0.15.0 — Web Foundations](https://github.com/aliharundaldalli/AhdCode/releases/tag/v0.15.0)
- [Ahd Akademi Matematik](https://github.com/aliharundaldalli/ahdcode-math-portal) — bu genel tanıtım
- [v0.4 Kütüphane Demosu](https://github.com/aliharundaldalli/ahdcode-library-demo)
- [v0.4 Seminer Demosu](https://github.com/aliharundaldalli/ahdcode-seminer-demo)

Bu uygulama AhdCode ağacında da
[`examples/v0.15/ahd_math_portal`](https://github.com/aliharundaldalli/AhdCode/tree/main/examples/v0.15/ahd_math_portal)
olarak durur.
