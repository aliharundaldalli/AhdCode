# Web

[English](WEB.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [HTTP](HTTP_TR.md) · [HTML](HTML_TR.md) · [Modüller](MODULES_TR.md) · [Env](ENV_TR.md) · [require(...)](REQUIRE_TR.md)

`Web`, AhdCode'un v0.15.0 ile gelen birinci taraf web çatısıdır. Sıradan bir
uygulama için tek bir içe aktarma yeter:

```ahd
bring Web
```

## 1. Web nedir

Web bir **cephedir** (facade). Mevcut `HTTP`, `HTML`, `Session`, `Security` ve
`Env` ilkellerini değiştirmez, onları birleştirir; o ilkellerin zaten
tanımladığı türleri yeniden dışa aktarır. `Web` üzerinden ulaşılan bir
`Request`, `HTTP`'nin bir işleyiciye verdiği `Request`'in **aynısıdır** —
kopyası değil. Bu yüzden `Web` ile yazılmış bir işleyici, çıplak bir
`HTTP.Server` üzerine hiçbir dönüşüm olmadan kaydolur.

Web'in neredeyse tamamı AhdCode ile yazılmıştır. Çatı, derleyiciye gömülü
AhdCode kaynağı olarak gelir ve sizin dosyalarınızla aynı geçişte derlenir. Go
yalnızca yalnızca Go'nun yapabileceğini üstlenir: soketler, TLS, dosya sistemi
sınırı, kriptografi ve düşük seviyeli HTTP.

## 2. Web ne değildir

v0.15 bilinçli olarak şunları **içermez**:

| İçermez | Durumu |
| --- | --- |
| ORM, sorgu kurucu, göç (migration) | `SQLite` / `MySQL` açıkça kullanılır |
| Form, doğrulama, eski girdi, hata torbaları | v0.16 için planlı |
| Ara katman zincirleri, rota grupları, yetki bekçileri | v0.17 için planlı |
| Şablon dili | `Web.document` bir kabuktur, şablon değil |
| Sanal DOM, hydration, tepkisel durum, hooks | Planlı değil; sayfalar sunucuda kurulur |
| Ön yüz paketleyici, npm, Node, CSS-in-JS | Planlı değil; varlıklar diskteki dosyalardır |
| Tarayıcı canlı yenileme / HMR | v0.15 için planlı değil |
| ACME, Let's Encrypt, sertifika yenileme | TLS'i ters vekilde sonlandırın |

Paket yöneticisi de yoktur. `bring Web` çevrimdışı çözülür: derleyiciye gömülü
baytlardan. Kayıt defteri, manifest, kilit dosyası veya indirme yoktur.

## 3. `bring Web`

`bring Web` bir ad alanı ve uygulamanın bildirdiği türleri verir:

```ahd
bring Web
from Web bring (Request, Response, HTMLNode, App, AppConfig)
```

Web şunları yeniden dışa aktarır: `Request`, `Response`, `Server`, `Session`,
`SessionStore`, `Cookie`, `UploadedFile`, `HTTPError`, `HTMLNode`,
`HTMLError`. Kendi türleri ise `App`, `AppConfig`, `UIKit` ve
`WebConfigError`'dır.

`Web` ayrılmış bir birinci taraf addır: projenizdeki kardeş bir `Web.ahd`
dosyası onu gölgeleyemez — tıpkı kardeş bir `HTTP.ahd`'nin `HTTP`'yi
gölgeleyememesi gibi.

## 4. Uygulama yapısı

Yayınlanmış [`require(...)`](REQUIRE_TR.md) kuralıyla kurulan sözleşme:

```
app.ahd
.env
.env.example

Config/
    App.ahd

Components/
    Navbar.ahd
    Notice.ahd

Layouts/
    Main.ahd

Pages/
    Home.ahd
    About.ahd

public/
    app.css
    logo.svg
```

Bu bir **sözleşmedir**, paket semantiği değil. `require("Pages/Home.ahd")`
zincir ne kadar derinleşirse derinleşsin uygulama kökünden — giriş dosyasının
kendi dizininden — çözülür.

## 5. Config

Kural tek yönlüdür:

```
.env / süreç ortamı
        ↓
     Config
        ↓
uygulama / sayfalar / bileşenler
```

Ortamı okuyan tek dosya `Config/App.ahd`'dir. Sayfalar, yerleşimler ve
bileşenler doğrulanmış bir `AppConfig` alır.

```ahd
bring Web
from Web bring AppConfig
bring Env

application: AppConfig := Web.configure()

configuration: Function := () -> AppConfig {
    application: Global AppConfig
    return application
}

tagline: Function := () -> String {
    return Env.getOr("APP_TAGLINE", "AhdCode ile yazılmış küçük bir akademi")
}
```

`Web.configure()` varsa `.env`'i yükler, sonra sözleşmenin tamamını doğrular;
hatalı anahtarı adıyla söyleyen bir `WebConfigError` fırlatır.

Derleyici v0.15'te Config sözleşmesini zorlamaz ve bunun için bir denetleyici
yoktur. Bu, kanonik yapının kolaylaştırdığı bir disiplindir.

## 6. Ortamlar

Altı anahtar dondurulmuştur:

| Anahtar | Değerler | Not |
| --- | --- | --- |
| `APP_NAME` | boş olmayan herhangi bir String | |
| `APP_ENV` | `development`, `test`, `production` | küçük harf, varsayılan yok |
| `APP_HOST` | bir konak, örn. `ahdakademi.com` | şema, yol veya port yok |
| `APP_PROTOCOL` | `http`, `https` | küçük harf, varsayılan yok |
| `SERVER_HOST` | örn. `127.0.0.1` | bu sürecin bağlandığı soket |
| `SERVER_PORT` | 1–65535 | açıkça dönüştürülür ve aralığı denetlenir |

Hiçbir şey sessizce varsayılana düşmez. Hangi ortamda çalıştığını söylemeyen
bir uygulama "muhtemelen development" değil, yanlış yapılandırılmıştır;
protokolünü söylemeyen bir uygulama da sessizce `http` üzerinden sunulmaz. Her
iki varsayılan da çerez güvenliğine ve yönlendirme hedeflerine kazara karar
verirdi.

`APP_HOST` bir **konaktır**, URL değil:

```
ahdakademi.com          kabul
admin.checkmate.tr      kabul

https://ahdakademi.com  ret — şema APP_PROTOCOL'dür
ahdakademi.com/path     ret — konağın yolu olmaz
ahdakademi.com:8080     ret — port SERVER_PORT'tur
```

Hatalı bir değer, doğru görünen bir şeye sessizce kırpılmaz; hangi kuralı
çiğnediğini söyleyen bir iletiyle reddedilir.

`APP_ENV=test`, otomatik testler için tanınan yalıtılmış bir ortamdır.
Production davranışını açmaz ve `.test` eklemez. v0.15 yeni bir test DSL'i
getirmez.

Yapılandırma hataları hatalı anahtarı adıyla söyler, değerini asla
yinelemez — böylece hatalı bir `DB_PASSWORD` hata yolundan bir günlüğe
sızamaz.

## 7. App

```ahd
require("Config/App.ahd")
require("Pages/Home.ahd")
bring Web
from Web bring App

academy: App := Web.app(application)

academy.assets("/assets", "public")
academy.get("/", homePage)
academy.get("/hakkinda", aboutPage)
academy.post("/selam", greetPage)
academy.start()
```

`App` iki şeye sahiptir: doğrulanmış yapılandırma ve bağlanmış sunucu.

| Yöntem | Anlamı |
| --- | --- |
| `get(path, handler)` | tam bir yol için GET kaydeder |
| `post(path, handler)` | tam bir yol için POST kaydeder |
| `route(method, path, handler)` | desteklenen herhangi bir yöntem |
| `assets(prefix, root)` | statik dosya dizinini sunar |
| `start()` | bağlanır ve sunar; geri dönmez |
| `configuration()` | doğrulanmış `AppConfig` |

Bir işleyici sıradan bir `Function(Request) -> Response`'tur ve `HTTP`'nin
denetlediği gibi tür denetiminden geçer — yanlış şekilli bir işleyici,
beklenen imzayı söyleyen bir derleme hatasıdır.

## 8. Yanıtlar

| Çağrı | Döndürür |
| --- | --- |
| `Web.html(node, status := 200)` | düğüm ağacından HTML yanıtı |
| `Web.page(title, body, head := [], status := 200)` | tam belge, yanıt olarak |
| `Web.document(title, body, head := [])` | tam belge, biçimlenmiş metin olarak |
| `Web.text(body, status := 200)` | düz metin yanıtı |
| `Web.redirect(location, status := 303)` | yönlendirme (303, POST sonrası için uygundur) |
| `Web.response(status, body, contentType)` | diğer her şey |
| `Web.render(node)` | düğüm ağacından biçimlenmiş metne |
| `Web.sessions(...)` | bir `SessionStore` |

## 9. Web.UI

`Web.UI` anlamsal HTML bileşen katmanıdır. İki şekil neredeyse her şeyi
kapsar:

```
metin öğeleri     tag(text, attributes := {})
kapsayıcı öğeler  tag(children, attributes := {})
```

```ahd
Web.UI.section(
    [
        Web.UI.h1("Ahd Akademi")
        Web.UI.p("Hoş geldiniz")
        Web.UI.img("/assets/logo.svg", "Ahd Akademi")
    ]
    {"class": "hero"}
)
```

Aynı ağacın düşük seviyeli modülle kurulmuş hâli:

```ahd
HTML.element(
    "section"
    {"class": "hero"}
    [
        HTML.element("h1", {}, [HTML.text("Ahd Akademi")])
        HTML.element("p", {}, [HTML.text("Hoş geldiniz")])
        HTML.element("img", {"src": "/assets/logo.svg", "alt": "Ahd Akademi"}, [])
    ]
)
```

İkisi aynı biçimlenmiş metni üretir. `HTML.element` kullanılabilir kalır ve
kaçış yolu olmayı sürdürür.

### Öznitelikler

Öznitelikler her zaman son parametredir ve her zaman isteğe bağlıdır: tek bir
türlü `Pair<String, String>`. Bu tek kapı `id`, `class`, `title`, `data-*`,
`aria-*` ve gerisini kapsar; `Any` yoktur, dinamik değer yoktur, etiket başına
parametre patlaması yoktur:

```ahd
Web.UI.p("Kayıt açıldı", {"class": "lead", "data-role": "notice", "aria-live": "polite"})
```

Bir yardımcının kendi öznitelikleri sizinkilerden sonra eklenir; sıralama
belirlenimlidir. Bir yardımcı size ait Pair'e asla yazmaz: Pair bir referanstır
ve paylaşılan bir öznitelik haritası aksi hâlde her öğenin özniteliklerini
biriktirirdi.

### Çocuklar

Metin öğesinin işaretleme çocuklarına da ihtiyacı olduğunda `Nodes` eşi
vardır:

```ahd
Web.UI.p("Düz metin")
Web.UI.pNodes([Web.UI.text("Merhaba "), Web.UI.strong("Ali")])
Web.UI.aNodes("/", [Web.UI.img("/logo.svg", "Ana sayfa")])
```

Bunun için `h1Nodes`…`h6Nodes`, `pNodes`, `spanNodes`, `liNodes`, `ddNodes`,
`aNodes`, `tdNodes`, `thNodes`, `labelNodes`, `buttonNodes`,
`blockquoteNodes`, `summaryNodes` ve `figcaptionNodes` bulunur.

### Öğe grupları

**Çekirdek** — `text`, `element`, `render`, `stylesheet`

**Belge** — `html`, `head`, `body`, `title`

**Sayfa yapısı** — `header`, `footer`, `main`, `section`, `article`, `aside`,
`nav`, `div`, `address`

**Başlıklar** — `h1`–`h6` (+ `Nodes`)

**Metin** — `p`, `span`, `blockquote` (+ `Nodes`), `strong`, `em`, `b`, `i`,
`u`, `small`, `mark`, `code`, `pre`, `time`, `br`, `hr`

**Listeler** — `ul`, `ol`, `dl`, `li` (+ `liNodes`), `dt`, `dd` (+ `ddNodes`)

**Bağlantı ve ortam** — `a` (+ `aNodes`), `img`, `picture`, `source`,
`figure`, `figcaption` (+ `Nodes`)

**Tablolar** — `table`, `caption`, `thead`, `tbody`, `tfoot`, `tr`, `th`, `td`
(+ `thNodes`, `tdNodes`)

**Formlar** — `form`, `formTo`, `label`, `labelFor`, `input`, `textarea`,
`select`, `option`, `button`, `fieldset`, `legend` (+ `labelNodes`,
`buttonNodes`)

**Açılır bölüm** — `details`, `summary` (+ `summaryNodes`)

### Kaçışlama

Sayfa içeriği olan her String `HTML.text`'ten geçer ve kaçışlanır:

```ahd
Web.UI.p("<script>alert(1)</script>")
```

şunu üretir:

```html
<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>
```

`Web` veya `HTML` içinde hiçbir yerde `raw()`, `unsafeHTML()`, `innerHTML()`
ya da `trustedHTML()` yoktur. Sıradan bir UI çağrısı, bir veritabanı veya form
değerini çalıştırılabilir HTML'e çeviremez ve v0.15 bundan çıkış yolu eklemez.

### Erişilebilirlik

`img`, `alt` ister. Dekoratif bir görsel bilinçli olarak `""` geçer; hiçbir
imza `alt`'ın unutulmasına izin vermez.

`labelFor(target, text)` `for` özniteliğini yazar; eşleşen `id` girdiye sizin
tarafınızdan konur:

```ahd
Web.UI.labelFor("isim", "Adınız")
Web.UI.input("text", "isim", "", {"id": "isim"})
```

Başlık hiyerarşisi sizin sorumluluğunuzdadır ve düğme ile bağlantı
anlambilimi açık kalır. Web, istemediğiniz bir erişilebilirlik özniteliğini
asla üretmez.

## 10. Sayfalar, Yerleşimler, Bileşenler

```
App        uygulamayı yönetir
Page       içeriği üretir
Layout     sayfayı sarar
Component  yeniden kullanılabilir parça üretir
```

**Bunlar sözleşmedir, dil yapısı değil.** Her biri `HTMLNode` (veya
`Response`) döndüren sıradan bir AhdCode `Function`'ıdır. Taban sınıf, yaşam
döngüsü, durum motoru, çizim zamanlayıcısı, hooks veya kayıt defteri yoktur.

Bir **Bileşen**:

```ahd
bring Web
from Web bring HTMLNode

notice: Function := (title: String, message: String) -> HTMLNode {
    return Web.UI.section([Web.UI.h2(title), Web.UI.p(message)], {"class": "notice"})
}
```

Bir **Yerleşim**:

```ahd
require("Components/Navbar.ahd")
bring Web
from Web bring (HTMLNode, Response, AppConfig)

mainLayout: Function := (config: AppConfig, title: String, current: String, content: List<HTMLNode>) -> Response {
    body: Local List<HTMLNode> := [
        navbar(config.name, current)
        Web.UI.main(content)
        Web.UI.footer([Web.UI.p("{config.name} — {config.effectiveURL()}")])
    ]
    return Web.page("{title} — {config.name}", body, [Web.UI.stylesheet("/assets/app.css")])
}
```

Bir **Sayfa**:

```ahd
homePage: Function := (request: Request) -> Response {
    content: Local List<HTMLNode> := [
        Web.UI.section([Web.UI.h1(configuration().name), Web.UI.p(tagline())])
    ]
    return mainLayout(configuration(), "Ana Sayfa", "/", content)
}
```

`Web.UI` ilkel HTML bileşenlerini tutar; sizin `Components/` dizininiz
uygulamaya özgü olanları. Bunlar bilinçli olarak ayrıdır.

## 11. Statik varlıklar

```ahd
academy.assets("/assets", "public")
```

`public/app.css` böylece `/assets/app.css` adresinden sunulur. Bu, yayınlanmış
`server.static`'e devredilir.

Statik bir dosyayı düzenlemek, tarayıcının bir sonraki istekte alacağını
değiştirir. AhdCode kaynağını **yeniden derlemez**, çünkü hiçbiri AhdCode
kaynağı değildir. Sıcak modül değişimi ve enjekte edilen tarayıcı yenilemesi
yoktur: sayfayı yeniden yükleyin.

## 12. Geliştirme kipi

```bash
ahdcode dev app.ahd
```

```
AhdCode Dev

Watching: /path/to/app.ahd

→ Building...
✓ Build succeeded
→ Starting...
✓ Running

AhdCode Web
  Ahd Akademi (development)

  http://ahdakademi.com.test

Waiting for changes...
```

v0.13/v0.14'ün bağımlılık farkında geliştirme denetleyicisi değişmemiştir ve
kaynak çizgesi, izleyici, son iyi yapı, yeniden derleme, yeniden başlatma ve
durdurma hâlâ ona aittir. Web bir başlık ve bir güvenlik denetimi ekler.

`require(...)` çizgesindeki herhangi bir kaynak dosyayı düzenlemek yeniden
derler ve yeniden başlatır. `public/app.css`'i düzenlemek bunu yapmaz.

`ahdcode dev`, `APP_ENV=production`'ı **reddeder**:

```
✗ APP_ENV is production, but this is the development command.
  ahdcode dev runs an application in development.
  Set APP_ENV=development for local work, or run the built
  executable directly for a production configuration.
  Nothing was started and APP_ENV was not changed.
```

Hiçbir şey başlatılmaz, `APP_ENV` yeniden yazılmaz ve komut sıfırdan farklı
bir kodla çıkar.

## 13. `.test`

`APP_ENV=development` için yerel kimlik, `APP_HOST`'a `.test` eklenmiş hâlidir:

```
APP_HOST=ahdakademi.com   →   ahdakademi.com.test
```

Böylece `APP_PROTOCOL=https`, mantıksal geliştirme adresi olarak
`https://ahdakademi.com.test`'i verir. `.test` ayrılmış özel amaçlı bir üst
düzey alan adıdır (RFC 6761); bunu kullanmak, geliştirme trafiğinin kazara
gerçek konağa çözülememesi demektir.

Production `APP_HOST`'u aynen kullanır ve bu son eki asla almaz.

`AppConfig` türetimleri sunar:

| Çağrı | `development` | `production` |
| --- | --- | --- |
| `url()` | `https://ahdakademi.com` | `https://ahdakademi.com` |
| `developmentURL()` | `https://ahdakademi.com.test` | `https://ahdakademi.com.test` |
| `developmentHost()` | `ahdakademi.com.test` | `ahdakademi.com.test` |
| `effectiveURL()` | `https://ahdakademi.com.test` | `https://ahdakademi.com` |
| `address()` | `127.0.0.1:8080` | `127.0.0.1:8080` |

## 14. Yerel HTTPS — mevcut sınır

`.test` kendiliğinden çözülmez ve v0.15 yerel bir sertifika otoritesi, bir
`.test` çözücüsü veya bir geliştirme geçidi **getirmez**. Bu sürümde
`ahdcode trust` komutu yoktur.

`https://<APP_HOST>.test` adresine görünür bir port olmadan ulaşmak, aynı anda
kalıcı olarak ayrıcalıklı üç sistem bileşeni gerektirir:

1. `.test` alanı için root ile kurulan bir çözücü (`/etc/resolver/test` artı
   bir çözücü süreci ya da root yönetimindeki `/etc/hosts` kayıtları),
2. ayrıcalıklı 443 portunda bir dinleyici veya root ile kurulan bir paket
   yönlendirmesi,
3. sistem güven deposunda bir sertifika otoritesi.

Bu, uzun ömürlü ve ayrıcalıklı bir yerel ağ artalan sürecidir. Yaklaşık bir
çözüm üretmek yerine ertelenmiştir: v0.15 hiçbir sistem durumu kurmaz, hiçbir
ayrıcalık istemez ve yerel güven artefaktı eklemez.

`APP_PROTOCOL=https` olduğunda `ahdcode dev`, ilk çalıştırmayı gizemli
bırakmak yerine durumu açıkça söyler:

```
  APP_PROTOCOL is https, which is the application's public identity.
  This machine has no local certificate authority or .test resolver,
  so https://ahdakademi.com.test does not open on its own yet.

  Until it does, either serve development over http by setting
  APP_PROTOCOL=http, or put a TLS-terminating proxy in front of
  127.0.0.1:8080. APP_PROTOCOL was not changed.
```

`https`'i asla sessizce `http`'ye düşürmez. Sessiz bir düşüş, güvenli çerez
veya karışık içerik sorununu production'a kadar gizlerdi.

Bugün yerel çalışma için `APP_PROTOCOL=http` kullanın ve
`http://127.0.0.1:SERVER_PORT` adresine gidin ya da hâlihazırda
çalıştırdığınız bir vekille TLS'i sonlandırın.

## 15. Production

```
     İnternet
        ↓
Cloudflare / Caddy / nginx
        ↓
    genel HTTPS
        ↓
  AhdCode uygulaması
        ↓
127.0.0.1:SERVER_PORT
```

Önemli olan ayrım:

| | Anahtarlar | Anlamı |
| --- | --- | --- |
| Genel kimlik | `APP_PROTOCOL`, `APP_HOST` | insanın yazdığı; bağlantıya konan |
| Süreç soketi | `SERVER_HOST`, `SERVER_PORT` | bu sürecin bağlandığı |

`APP_PROTOCOL=https` **genel URL'yi** tanımlar. AhdCode sürecinin TLS'i kendi
sonlandırdığı anlamına gelmez. Cloudflare, Caddy veya nginx arkasında
genellikle sonlandırmaz:

```
APP_PROTOCOL=https
APP_HOST=ahdakademi.com     →   https://ahdakademi.com

SERVER_HOST=127.0.0.1
SERVER_PORT=8080            →   127.0.0.1:8080
```

Genel bir URL'yi asla `SERVER_PORT`'tan türetmeyin.

v0.15 bir production sertifika yöneticisi değildir: ACME yok, Let's Encrypt
otomasyonu yok, DNS meydan okumaları yok, yenileme servisi yok. `HTTP`
ilkelleriniz doğrudan TLS'i zaten destekliyorsa, bu kullanılabilir ve
değişmeden kalır.

## 16. `.env`

`.env` **yerel bir geliştirme kolaylığıdır**.

- Gerçek bir `.env`'i işlemeyin (commit etmeyin). Adları ve yer tutucuları
  içeren, sır barındırmayan `.env.example`'ı işleyin.
- Süreç ortamında zaten bulunan bir değişken `.env`'e **üstün gelir**; böylece
  konteynerler ve CI, dosya düzenlemeden öngörülebilir biçimde geçersiz kılar.
- Dilbilgisi isteğe bağlı tırnaklı düz `KEY=value`'dur. Yerine koyma yoktur,
  komut ikamesi yoktur ve hiçbir şey çalıştırılmaz.
- `ahdcode build` `.env`'i **gömmez**. Üretilen çalıştırılabilir dosya ortamdan
  bağımsız bir üründür; farklı bir ortamla çalıştırın, farklı bir dağıtımdır.

Sırlar — `DB_PASSWORD`, API anahtarları, SMTP parolaları, jetonlar — çalışma
zamanı yapılandırması olarak kalır ve bir ikiliye asla girmez.

```bash
cp .env.example .env
```

## 17. Veritabanı

Belgelenen ama Web tarafından **tüketilmeyen** sözleşmeli anahtarlar:

```
DB_HOST
DB_PORT
DB_NAME
DB_USERNAME
DB_PASSWORD
```

Veritabanı erişimi açık kalır — `bring MySQL` veya `bring SQLite` — ve bunları
`Config/Database.ahd` okur ve doğrular. ORM yoktur, genel bir `Web.Database`
yoktur.

## 18. Hatalar

Web tam olarak bir hata türü ekler: eksik ya da hatalı bir ortam sözleşmesi
için `WebConfigError`. Geri kalan her şey onu zaten tanımlayan kimliği
korur — `HTTPError`, `HTMLError`, oturum hataları, `SecurityError` — çünkü her
şeyi kapsayan bir `WebError`, mevcut modüllerin zaten iyi bildirdiği hataları
yalnızca yeniden markalardı.

## 19. Güvenlik sınırları

- **Kaçışlama** isteğe bağlı değildir. Her metin giriş noktası kaçışlar; ham
  işaretleme yardımcısı yoktur.
- **Sırlar** bir ikiliye girmez ve yapılandırma hatalarında görünmez.
- **Oturumlar, CSRF ve parola özetleme** açık `Session`, `Security` ve `HTTP`
  ilkelleri olarak kalır. Web bunların etrafına sihir eklemez.
- **Statik varlıklar** yayınlanmış `server.static` sınırından geçer.
- **`bring Web`** gömülü baytlardan çevrimdışı çözülür. Hiçbir zaman indirme
  yoktur.

## 20. HTTP ve HTML ile ilişkisi

Web düşük seviyeli modülleri asla değiştirmez. Şu geçerli ve değişmemiş kalır:

```ahd
bring HTTP
bring HTML

server: Server := HTTP.server("127.0.0.1", 8080)
node: HTMLNode := HTML.element("p", {}, [HTML.text("merhaba")])
```

v0.4–v0.14 arası her uygulama derlenmeye ve çalışmaya devam eder. Web
eklemelidir; göç yoktur.

`Web.UI`'nin adlandırmadığı bir etiket veya öznitelik kalıbına ihtiyacınız
olduğunda ya da çatının yapılandırma sözleşmesi olmadan sunucuyu istediğinizde
düşük seviyeli modüllere uzanın.

## 21. Sırada ne var

| Sürüm | Alan |
| --- | --- |
| v0.16 | Formlar, doğrulama, CSRF kolaylıkları, flash, eski girdi, form hataları |
| v0.17 | Zengin yönlendirme, rota grupları, ara katman bileşimi, yetki bekçileri |

## Örnek

Tam bir uygulama
[`examples/v0.15/ahd_academi`](../examples/v0.15/ahd_academi) içindedir:
config katmanı, bir yerleşim, iki sayfa, iki bileşen, GET ve POST rotaları,
statik CSS ve `require(...)` bileşimi.

```bash
cd examples/v0.15/ahd_academi
cp .env.example .env
ahdcode dev app.ahd
```
