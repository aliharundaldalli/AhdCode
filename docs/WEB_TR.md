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
| Form, doğrulama, eski girdi, hata torbaları | v0.16'da yayımlandı (bkz. bölüm 16) |
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
academy.get("/", home)
academy.get("/hakkinda", about)
academy.post("/selam", greet)
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

*Adı* yönlendirme açısından önemsizdir; önerilen sözleşme için bkz.
[10.1 Adlandırma](#101-adlandırma).

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
home: Function := (request: Request) -> Response {
    content: Local List<HTMLNode> := [
        Web.UI.section([Web.UI.h1(configuration().name), Web.UI.p(tagline())])
    ]
    return mainLayout(configuration(), "Ana Sayfa", "/", content)
}
```

`Web.UI` ilkel HTML bileşenlerini tutar; sizin `Components/` dizininiz
uygulamaya özgü olanları. Bunlar bilinçli olarak ayrıdır.

### 10.1 Adlandırma

`Page`, `Layout` ve `Component` **uygulama düzenleme sözleşmeleridir, özel dil
yapıları değil.** Derleyicide, yönlendiricide veya çalışma zamanında hiçbir şey
bir ada bakmaz.

**İşleyici adları sıradan AhdCode tanımlayıcılarıdır. `Page` zorunlu bir sonek
değildir.** Bir rotaya bir `Function` **değeri** verilir:

```ahd
portal.get("/admin/users/edit/*", adminUserEdit)
```

O fonksiyonun adı ne olursa olsun — `adminUserEdit`, `adminUserEditPage`,
`admin_user_edit` — yönlendirici aynı değeri alır ve aynı şekilde davranır.
Önceki sürümlere göre yazılmış, `registerPage` veya `profilePage` kullanan
uygulamalar kaynak uyumlu kalır; hiçbir şey kullanımdan kaldırılmadı.

Bu belgelerdeki yeni örnekler gereksiz soneki bırakır ve bir ekran birden çok
işleyici gerektirdiğinde eylemi açıkça adlandırır:

```
register            // GET  /register
registerSubmit      // POST /register

adminQuestionEdit   // GET  düzenleme formu
adminQuestionSave   // POST düzenleme formu
```

**Önerilen dosya adları PascalCase'dir**; böylece işleyici adındaki sözcük
sınırları dosya adındaki sözcük sınırlarıyla aynı olur:

```
Pages/Admin/UserEdit.ahd        ->  adminUserEdit    (camelCase, tercih edilen)
                                ->  admin_user_edit  (snake_case, o da geçerli)
Pages/Admin/QuestionEdit.ahd    ->  adminQuestionEdit
Pages/Admin/Users.ahd           ->  adminUsers
```

Biçem terimleri, tam karşılıklarıyla:

```
adminUserEdit      camelCase
AdminUserEdit      PascalCase
admin_user_edit    snake_case
```

`admin_UserEdit` gibi karışık yazımlar iki sözleşmeyi anlam katmadan
birleştirir; örnekler bunlardan kaçınır. `adminUserEdit` yalnızca mevcut
AhdCode örnekleri ağırlıklı olarak camelCase olduğu için tercih edilir —
tanımlayıcı dil bilgisi alt çizgi kabul eder, dolayısıyla `admin_user_edit`
sıradan geçerli koddur ve seçim uygulamaya aittir.

Tanımlayıcılar **büyük/küçük harfe duyarlıdır** ve bu bir arama kuralı değil bir
adlandırma sözleşmesi olduğundan, birleştirilmiş veya harf katlanmış yazımlar
aynı adın alternatif yazımları değildir:

```
adminuseredit   adminUSEREDIT   adminUseredit   admin_UserEdit
```

Bunların hiçbiri `adminUserEdit` değildir. Otomatik rota keşfi, dosya adı
yansıması, dosya adından fonksiyona arama, büyük/küçük harf duyarsız veya
bulanık işleyici eşleştirmesi, `Page` açıklaması ya da `page` anahtar sözcüğü
yoktur. `require` zinciri açıktır ve rota argümanı düz bir değerdir.

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

  Open:
  http://127.0.0.1:8080

  Development identity:
  http://ahdakademi.com.test
  (.test is not locally routed in v0.15)

Waiting for changes...
```

**`Open:` altındaki adresi açın.** Bu, uygulamanın gerçekten bağlandığı soket
olan `SERVER_HOST` ve `SERVER_PORT`'tur ve bu makinede çalışan tek adrestir.

`Development identity:` altındaki satır, uygulamanın *yapılandırıldığı*
addır. v0.15 bu adı türetir ama çözmez — gömülü bir `.test` çözücüsü yoktur —
bu yüzden yalnızca bilgi olarak gösterilir ve öyle işaretlenir; asla
tıklanacak bir yer olarak değil.

v0.13/v0.14'ün bağımlılık farkında geliştirme denetleyicisi değişmemiştir ve
kaynak çizgesi, izleyici, son iyi yapı, yeniden derleme, yeniden başlatma ve
durdurma hâlâ ona aittir. Web bir başlık ve iki güvenlik denetimi ekler.

`require(...)` çizgesindeki herhangi bir kaynak dosyayı düzenlemek yeniden
derler ve yeniden başlatır. `public/app.css`'i düzenlemek bunu yapmaz.

### Reddedilen yapılandırmalar

`ahdcode dev`, `APP_ENV=production`'ı **reddeder**:

```
✗ APP_ENV is production, but this is the development command.
  ahdcode dev runs an application in development.
  Set APP_ENV=development for local work, or run the built
  executable directly for a production configuration.
  Nothing was started and APP_ENV was not changed.
```

`APP_PROTOCOL=https`'i de **reddeder**:

```
✗ Local HTTPS is not available in AhdCode v0.15.
  ahdcode dev serves plaintext HTTP, so it cannot honour
  APP_PROTOCOL=https.

  Configured identity:
  https://ahdakademi.com.test

  Set APP_PROTOCOL=http for local development, or terminate
  HTTPS with an external local proxy in front of 127.0.0.1:8080.
  Nothing was started and APP_PROTOCOL was not changed.
```

`ahdcode dev` uygulamayı başlatır ve uygulama düz metin bir HTTP soketine
bağlanır. v0.15'te `APP_PROTOCOL=https`'in burada TLS'e dönüştüğü bir yol
yoktur; alt süreci başlatmak, yapılandırma `https` derken `http` sunmak
olurdu. Düşürmek yerine reddeder: sessiz bir düşüş, güvenli çerez veya karışık
içerik sorununu production'a kadar gizlerdi.

Her iki rette de hiçbir şey başlatılmaz, hiçbir dinleyici açılmaz, geride
`.dev` tanımlayıcısı kalmaz, `APP_ENV` ve `APP_PROTOCOL` değişmez ve komut
sıfırdan farklı bir kodla çıkar.

`APP_ENV=test` normal çalışır. `APP_HOST`'u değiştirmeden kullanır; bu yüzden
başlık yalnızca bağlanma adresini bildirir ve hiçbir `.test` kimliği
göstermez.

## 13. `.test`

`APP_ENV=development` için yerel kimlik, `APP_HOST`'a `.test` eklenmiş hâlidir:

```
APP_HOST=ahdakademi.com   →   ahdakademi.com.test
```

`.test` ayrılmış özel amaçlı bir üst düzey alan adıdır (RFC 6761); bunu
kullanmak, geliştirme trafiğinin kazara gerçek konağa çözülememesi demektir.

**v0.15'te `.test` mantıksal bir kimliktir, yönlendirilebilir bir adres
değil.** AhdCode bunun için DNS, çözücü kaydı veya `/etc/hosts` girdisi
kurmaz; bu yüzden ad, olağan macOS çözümlemesiyle çözülmez ve tarayıcıda
açılamaz. Doğrudan kullanılabilir yerel adres `SERVER_HOST:SERVER_PORT`'tur ve
`ahdcode dev` önce onu yazar.

Production `APP_HOST`'u aynen kullanır — gerçek kanonik alan adını — ve bu son
eki asla almaz. `APP_ENV=test` de `APP_HOST`'u değiştirmeden kullanır.

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

Bu nedenle `APP_PROTOCOL=https`, `ahdcode dev`'in uygulamayı düz metin http
üzerinden sunup ona https demesi yerine
[başlatmayı reddetmesine](#reddedilen-yapılandırmalar) yol açar. `https`'i
asla sessizce `http`'ye düşürmez ve asla güvenilmeyen bir sertifika üretmez.
Sessiz bir düşüş, güvenli çerez veya karışık içerik sorununu production'a
kadar gizlerdi.

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
| v0.17 | `ahdcode init web`, bağlam duyarlı rotalar, rota grupları, sıralı bekçiler |
| v0.18 | Web starter'lar: Empty, Basic, Admin; yerel Bootstrap; Admin DB kurulumu |

## Bir proje başlatmak

```bash
mkdir my-app
cd my-app
ahdcode init web
ahdcode dev app.ahd
```

Bir terminalde `init web` Empty, Basic veya Admin sorar. Ayrıca
`ahdcode init web empty|basic|admin` çalıştırılabilir. Bu, v0.17'nin hemen
iskelet yazmasından 1.0 öncesi bir değişikliktir.

Şablonlar ve [Bootstrap 5.3.3](https://getbootstrap.com/) (MIT) CLI içindedir.
Üretilen sayfalar yalnızca yerel dosyaları yükler. `init web` sırasında CDN,
npm veya ağ indirmesi yoktur.

## 18. v0.18: Web starter'lar ve uygulama başlangıcı

v0.18 RequestContext, RouteSet, bekçiler, Forms, CSRF, Flash, Security,
SQLite, MySQL, HTTP veya SMTP'yi değiştirmez. Değişen **başlangıç deneyimidir**.

### Empty

Cilalı bir karşılama uygulaması. Veritabanı, giriş, pano, repository veya
posta anahtarı yoktur. `.env` altı uygulama anahtarında kalır.

### Basic

Aynı kabuk artı `Config/Mail.ahd` ve bu fonksiyonların gerçekten okuduğu
`MAIL_*` anahtarları. `MAIL_SECURITY` varsayılanı `starttls` (587). Boş
`MAIL_HOST` uygulamayı yine başlatır. Veritabanı ve kimlik doğrulama yoktur.

### Admin

Herkese açık Home, Login, Dashboard ve POST `/logout` (CSRF, sonra `/`).
`signedIn` üretilmiş sıradan uygulama kodudur.

Giriş Form, ValidationErrors, CSRF, oturum döndürme, Flash ve yalnızca e-posta
eski girdisini kullanır. Hatalar e-postanın var olup olmadığını söylemez.

#### SQLite

`database/<ad>.db` ve `database/schema.sql` oluşturur, şemayı uygular,
yöneticiyi `Security.passwordHash` ile ekler. Var olan bir `.db` dosyası
init'i durdurur. Üretilen `database/*.db` gitignore'dadır; `schema.sql` değil.

#### MySQL

Ana bilgisayar, port, veritabanı adı, kullanıcı adı ve parola (gizli) sorar.
Yayınlanmış MySQL sözleşmesini kullanır (`tls` veya `none`; varsayılan `tls`).
Yerel çakışma denetimlerinden sonra var olan bir veritabanını reddeder,
doğrulanmış bir tanıtıcı ile `CREATE DATABASE` yapar, şemayı ve yöneticiyi
kurar. MySQL kullanıcısı oluşturmaz, GRANT değiştirmez. Bu çağrı yeni bir
veritabanı oluşturup sonraki adım başarısız olursa veritabanı **silinmez**.

## 17. v0.17: bağlam rotaları, gruplar ve bekçiler

v0.17 eklemedir. `Function(Request) -> Response` ile `App.get` derlenmeye
devam eder. İkinci kayıt katmanı istek başına bir `RequestContext` açar;
çalışan yol hâlâ `context.respond` ister.

```ahd
routes := Web.routes(site, sessions)
routes.get("/profile", profile)

admin := routes.group("/admin")
admin.get("/users", adminUsers, authenticated, adminOnly)
```

| Parça | Sözleşme |
| --- | --- |
| işleyici | `Function(RequestContext) -> Response` |
| bekçi | `Function(RequestContext) -> Response?` |
| `null` | devam |
| `Response` | dur; değer `context.respond(...)` ile bitmiş olmalıdır |

Bekçi sırası `get`/`post`/`route` ek argümanları, sonra işleyicidir.
Varsayılan her isteğe izin verir. `next()`, `use()`, gizli sonlandırıcı,
Web içinde yetki politikası ve isimli rota parametresi yoktur. Function
alanları bekçi listesi tutamaz; kontroller kayıt satırında kalır.

Grup birleşimi açıktır. `/admin` + `/users` → `/admin/users`. `?`, `#`,
`//` veya tahminî onarım `WebRouteError` yükseltir. Eşleşmeyi hâlâ HTTP
yapar.

Odak örnek: [`examples/v0.17/routes_guards`](../examples/v0.17/routes_guards).

v0.17 genel ara katman zinciri, kimlik çerçevesi, ORM, otomatik rota
keşfi veya ön yüz çalışma zamanı **eklemez**.

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

## 16. v0.16: istek bağlamı, formlar, doğrulama, CSRF ve flash

v0.16, Matematik Portalında ölçülen oturum, doğrulama ve form tekrarlarını
azaltır. Olağan `Function -> HTMLNode` bileşimini ve açık veri akışını korur.
Yeni kolaylıkların tamamı derleyiciye gömülü AhdCode kaynaklarıdır; yerel HTTP
ayrıştırıcısı, SessionStore ve Security uygulaması değişmez. Yeni bağımlılık
veya dil sözdizimi eklenmez.

### Tam örneği çalıştırın

```bash
cd examples/v0.16/forms_validation
ahdcode run app.ahd
```

`http://127.0.0.1:8160/register` adresini açın. Önce geçersiz, sonra geçerli
veri gönderin; `/profile` yönlendirmesinden sonra sayfayı yenileyince mesajın
kaybolduğunu görün. Veritabanı, hesap oluşturma, `.env` dosyası veya indirilecek
kaynak gerekmez. Örnek loopback üzerinde yerel HTTP için açıkça Secure olmayan
çerez kullanır. HTTPS uygulamasında depoyu `secure: true` ile oluşturun veya
mevcut `AppConfig.isSecure()` politikasını kullanın. Kabuktaki isteğe bağlı
`SERVER_PORT` portu değiştirir; `.env.example` açıklamadır, otomatik yüklenmez.

### Bir bağlam, açıkça sonlandırılmış bir yanıt

Her gelen istek için `Web.context(request, store)` ile bir bağlam oluşturun.
Depoyu uygulama seçer; genellikle başlangıçta bir kez oluşturur. Bağlam özgün
`request`, açılan `session`, kaydetme yetkisine sahip `store` ve `finalized`
bayrağını taşır. Fonksiyonlara aktarılan sıradan bir referans değeridir; gizli
istek tekili, servis kapsayıcısı, örtük yetkilendirme veya kanca yoktur.

Yanıtı oluşturun ve yürütülen dönüş yolunda `context.respond(response)` çağrısını
tam bir kez yapın: 200, 403, 404, 422 ve yönlendirmeler dahil. Seçilen SessionStore
ile kaydeder, sonra bağlamı sonlandırılmış işaretler. Aynı değerin başka bir
referansı üzerinden de olsa ikinci çağrı, tekrar kaydetmeden `WebContextError`
fırlatır. Kaydetme hata verirse sonlandırma başarılı olmamıştır; otomatik yeniden
deneme yoktur. İşleyici sonlandırılmış yanıtı döndürmelidir.

Oturum değişikliklerini yanıttan **önce** bitirin. Mevcut giriş, rotate veya destroy
fonksiyonlarına `context.session` aktarabilirsiniz. Bağlamın CSRF/flash değiştiren
metotları sonlandırmadan sonraki kullanımı reddeder. Alanlar sıradan AhdCode sınıf
öznitelikleridir, erişim güvenliği sınırı değildir: session/store değerlerini
değiştirmeyin, bayrağı sıfırlamayın ve aynı yanıtta alt düzey commit ile bağlam
sonlandırmasını karıştırmayın. Alt düzey `HTTP`, `Request.form`,
`SessionStore.open/commit` ve bütün v0.15 `Web.UI` yardımcıları geçerlidir.
Tam yollar ve v0.15.1 son tek segment `/*` yolları aynı işleyicileri kabul eder;
tam yol eşleşmesi önceliklidir.

### GET, POST, doğrulama, yönlendirme

GET'te bağlam oluşturun, form içinde `Web.UI.csrfField(context)` üretin ve sayfayı
sonlandırın. POST'ta `context.form()` alın, CSRF'yi açıkça denetleyin ve doğrulayın.
Hatalarda mesajlarla seçilmiş eski girdiyi gösterin; başarıda uygulama değişikliğini
yapıp yönlendirin. Aşağıdaki işleyici [çalıştırılabilir tam örnekteki](../examples/v0.16/forms_validation/app.ahd)
görünümü ve depo erişim fonksiyonunu kullanır:

```ahd
registerSubmit: Function := (request: Request) -> Response {
    context: Local RequestContext := Web.context(request, exampleSessions())
    form: Local Form := context.form()
    if context.csrfValid() == false {
        return context.respond(Web.text("Forbidden", 403))
    }
    errors: Local ValidationErrors := Web.errors()
    errors.required("name", form.value("name"), "Name is required.")
    errors.email("email", form.value("email"), "Enter an email address.")
    errors.minLength("password", form.value("password"), 10, "Use at least 10 characters.")
    errors.matches("password_confirmation", form.value("password_confirmation"), form.value("password"), "Passwords must match.")
    if errors.any() {
        return context.respond(registerView(context, form.old(["name", "email"]), errors, 422))
    }
    context.flashSet("notice", "Form accepted. No account was created.")
    return context.respond(Web.redirect("/profile"))
}
```

`Web.form(request)` oturumsuz ve bağlamsız da çalışır; her form CSRF gerektirmez.
Mevcut ayrıştırıcının istek görüntüsünü sarar; gövdeyi tekrar ayrıştırmaz veya
dinamik değerler üretmez. `value` ilk gönderilen metni değiştirmeden döndürür;
yalnızca eksik alanda varsayılanı kullanır. `optional`, null ile boş metni ayırır;
`hasField` boş değer dahil alanın varlığını sınar. Gerektiğinde `.trim()` veya
`.lower()` ile açıkça normalleştirin. Uygulamanızın açık politikası değilse
parolaları kırpmayın. Sorgu dizesi ayrıdır: URL sorgusuyla çalışan arama formu için
`Request.query` kullanın.

`integer` eksik alanda null; boş, hatalı veya taşan girdide `FormValueError`
üretir ve mevcut `int(String)` dilbilgisini izler. `checkbox` eksik alanda false,
`on`/`true`/`1` için true, `off`/`false`/`0` için false döndürür; diğer metinler
`FormValueError` üretir. Büyük/küçük harf duyarlıdır. Eksik ile açık false ayrımı
gerekiyorsa önce `hasField` kullanın. Hatalar gönderilen değeri içermez.
Multipart dosyaları mevcut `Request.file/files` üzerinden alınır.

### Doğrulama açık ve sıralıdır

`Web.errors()` bağımsız bir koleksiyon oluşturur. Kural metotları verilen değer
başarısızsa hata ekler; istek okumaz ve gönderilen değerleri dönüştürmez.
`required` boş veya yalnızca boşluk içeren değerleri reddeder. Uzunluk kuralları
AhdCode `len(String)` anlamını ve dahil sınırları kullanır; negatif olmayan
sınırlar seçin. `matches` onay alanları için olağan eşitliktir, gizli belirteç
karşılaştırması değildir. `email` tek `@`, boş olmayan yerel/alan adı bölümleri,
boşluk/tab/CR/LF bulunmaması ve en az iki boş olmayan alan adı etiketi arar.
RFC doğrulaması veya teslim edilebilirlik kanıtı değildir. `hexColor` yalnızca
`#RRGGBB` kabul eder. `oneOf` büyük/küçük harf dahil tam metin üyeliğini sınar.

Hatalar kural/add çağrı sırasını korur. `first` yoksa null döndürür;
`forField` ve `messages` bağımsız metin listeleri döndürür. Aynı alanın birden
çok hatası korunur. `errors.add("email", "Zaten kayıtlı.")` ile veritabanı
benzersizliği ve iş kuralları uygulamada kalır. Mesajları örnekteki FormErrors
bileşeni gibi `Web.UI` metin düğümleriyle gösterin.

### Eski girdi açık bir izin listesidir

`form.old(["name", "email"])` yalnızca listede bulunan ve gönderilmiş alanlarla
yeni bir `OldInput` üretir. Alan listesi zorunludur; otomatik tümünü kopyalama
veya oturuma kaydetme yoktur. Parolaları, onaylarını, sıfırlama doğrulayıcılarını,
CSRF belirteçlerini ve diğer sırları listeye eklemeyerek dışarıda bırakın.
**Çerçeve hassas alan adlarını tahmin etmez ve açık seçiminizi değiştirmez.
Yalnızca güvenli alanları seçmek uygulamanın sorumluluğudur.** Boş form için boş
liste kullanın. OldInput görünümün açık parametresidir; gizli istek durumu değildir.

`old.value` sıradan metin döndürür. `Web.UI.input` bunu value özniteliğinde
kaçırır: `<script>alert(1)</script>` veri olarak kalır. Parola kontrollerine
örnekteki gibi `""` verin. Eski girdi yönlendirme sonrasında otomatik taşınmaz;
bu sürüm doğrulama hatasında aynı yanıtta yeniden göstermeyi öğretir.

### CSRF ve flash durumu

`csrfToken()` oturum başına `Security.token()` kullanır ve değeri ayrılmış
`__web_csrf` anahtarında tutar. Beklenen isteklerde ve oturum değerlerini koruyan
rotation sırasında sabittir. Oturumu temizlemek/yok etmek yeni belirteç gerektirir.
`Web.UI.csrfField(context)` kaçırılmış gizli `_csrf` alanı üretir; göndermez veya
doğrulamaz. `csrfValid()` beklenen/gönderilen değerin eksik veya boş olmasını
reddeder ve `Security.secureEqual` kullanır. Doğrulama belirteç üretmez.
Belirteç veya parola değerlerini günlüğe yazmayın.

`flashSet(key, value)` metni `__web_flash:{key}` altında saklar. `flashTake(key)`
nullable metni döndürüp kaldırır; ikinci alım null döndürür. Anahtarları ve görsel
kategorileri siz seçersiniz. Sonraki sayfa işleyicisi mesajı alıp yanıtını
sonlandırmalıdır; silme böyle kaydedilir ve yenilemede mesaj görünmez. Otomatik
gösterim veya istek yaşına göre süre sonu yoktur; mesaj açıkça tüketilene kadar
bekler. Eski uygulama yardımcılarıyla birlikte kullanımda da `__web_csrf` ve
`__web_flash:` önekini Web için ayırın.

Depo yayımlanmış bellek içi, süreç yerel ömrünü ve çerez politikasını korur.
Bağlam kalıcı oturum veya süreçler arası eşzamanlama eklemez. Kullanıcı/site
bağlamı, korumalar ve DB/JSON dönüşümleri açık kalır. Bu sürümde middleware, auth
çerçevesi, ORM, JSON/yönlendirme yeniden tasarımı veya varlık sistemi yoktur.

### Kesin v0.16 API'si

```text
Web.context(request: Request, store: SessionStore) -> RequestContext
Web.form(request: Request) -> Form
Web.errors() -> ValidationErrors
Web.UI.csrfField(context: RequestContext) -> HTMLNode

RequestContext.respond(response: Response) -> Response
RequestContext.form() -> Form
RequestContext.csrfToken() -> String
RequestContext.csrfValid() -> Bool
RequestContext.flashSet(key: String, value: String) -> Nothing
RequestContext.flashTake(key: String) -> String?

Form.optional(name: String) -> String?
Form.value(name: String, fallback: String := "") -> String
Form.hasField(name: String) -> Bool
Form.integer(name: String) -> Int?
Form.checkbox(name: String) -> Bool
Form.old(safeFields: List<String>) -> OldInput
OldInput.value(name: String, fallback: String := "") -> String

ValidationErrors.add(field: String, message: String) -> Nothing
ValidationErrors.any() -> Bool
ValidationErrors.hasField(field: String) -> Bool
ValidationErrors.first(field: String) -> String?
ValidationErrors.forField(field: String) -> List<String>
ValidationErrors.messages() -> List<String>
ValidationErrors.required(field: String, value: String, message: String) -> Nothing
ValidationErrors.minLength(field: String, value: String, minimum: Int, message: String) -> Nothing
ValidationErrors.maxLength(field: String, value: String, maximum: Int, message: String) -> Nothing
ValidationErrors.matches(field: String, value: String, expected: String, message: String) -> Nothing
ValidationErrors.email(field: String, value: String, message: String) -> Nothing
ValidationErrors.oneOf(field: String, value: String, allowed: List<String>, message: String) -> Nothing
ValidationErrors.hexColor(field: String, value: String, message: String) -> Nothing
```

Yeni somut türler ve kurucu öznitelikleri:

| Tür | Öznitelikler |
| --- | --- |
| `RequestContext` | `request: Request`, `session: Session`, `store: SessionStore`, `finalized: Bool := false` |
| `Form` | `request: Request` |
| `FormField` | `name: String`, `value: String` |
| `OldInput` | `fields: List<FormField>` |
| `FieldError` | `field: String`, `message: String` |
| `ValidationErrors` | `entries: List<FieldError>` |

Olağan kurucular `Web.context`, `Web.form`, `Web.errors` ve `Form.old` olmalıdır.
`WebContextError` ve `FormValueError`, `Error` ve özniteliklerini miras alır;
mevcut hata kimlikleri değişmez. Tamamlama, hover ve imza yardımı yeni API'leri
derlenmiş ModuleInterface üzerinden öğrenir.
