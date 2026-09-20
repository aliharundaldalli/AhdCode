<p align="center">
  <img src="editors/vscode/images/ahdcode-logo.png" alt="AhdCode logosu" width="360">
</p>

# AhdCode

[![CI](https://github.com/aliharundaldalli/AhdCode/actions/workflows/ci.yml/badge.svg)](https://github.com/aliharundaldalli/AhdCode/actions/workflows/ci.yml)

[English](README.md) · [Türkçe]

AhdCode; okunabilir sözdizimi, açık niyet (explicit intent), öngörülebilir
anlambilim (semantics) ve yerel (native) derlemeye odaklanan, bağımsız olarak
geliştirilen, statik olarak denetlenen genel amaçlı bir programlama dilidir.
Kendi araç zinciri, standart kütüphanesi, dil sunucusu, Web çatısı,
veritabanı modülleri, GUI'si, Graphics'i, etkileşimli Plot görüntüleyicisi ve
masaüstü uygulaması paketlemesiyle gelir. Küçük bir topluluk tarafından
pratikte kullanılmaktadır; yaygın (mainstream) bir dil değildir.

Bu, **v2.3.0**, **Dil Ergonomisi ve Plot Cilası** sürümüdür. Açık isimli
Function `uses` capture'ları, capture-aware derleyici ve LSP araçları,
workspace referans/rename, mevcut Surface görünümünü kaydeden Save ve
Plot/Surface etiketleri için sınırlı matematiksel metin ekler. Önceki
**Plot'un Tamamlanması** sürümü olan v2.2.0 da kullanılabilir durumdadır.

v2.0.0 bir ana (major) sürümdür, **Masaüstü Uygulamasının Tamamlanması**:
birinci taraf masaüstü uygulaması temelini tamamlar — GUI'de ListBox, Select,
TextArea, PasswordInput, TableView, iletişim kutuları, değişiklik
callback'leri ve yeniden boyutlandırılabilir pencereler; Plot
görüntüleyicisinde araç çubuğu ve 3B Surface çizimi; ve kendi başına çalışan
masaüstü uygulamaları için `ahdcode package` — yeni sözdizimi eklemez ve tip
sistemini değiştirmez. Bkz. [v2.0.0 ile gelenler](#v200-ile-gelenler).

v2.1.0 ikili güvenli giden indirmeler, giden multipart yüklemeler, SMTP
ekleri, WebSocket istemcisi ekler ve v2.0 Plot Save çalışma zamanı bulma
gerilemesini düzeltir. Bkz. [v2.1.0 ile gelenler](#v210-ile-gelenler).

Ürün, kendi kendine yeten bir platform paketi olarak dağıtılır: `ahdcode` CLI,
özel Go 1.27.0 araç zinciri, AhdDataStudio, `ahdsqlite`, `ahdnumeric`, `ahdplot`,
`ahdgraphics` ve `ahdgui` yardımcıları, etkileşimli Plot görüntüleyicisi `ahdplotview`, sabitlenmiş kaynak paketiyle çevrimdışı Tectonic LaTeX
motoru, Web starter'ları ve bu sürüme ait İngilizce belge paketi.
Bkz. [Kurulum](docs/INSTALLATION_TR.md).

1.0 Web yüzeyi, v0.20.0'ın **Web Varlıkları, Kaynak Sınırları ve Uygulama
Örüntüleri** sürümüyle geldi. Bileşenler HTML döndüren sıradan fonksiyonlar
olarak kalır. Düzenler CSS ve JavaScript'i `Web.Assets` ile bildirir;
`managedAssets` yalnızca bildirilen dosyaları sunar. `Identity.id()` herkese
açık tanımlayıcı üretir. Web uygulamaları gövde, yükleme ve zaman aşımı
sınırlarını açıkça uygular. `ahdcode init web` artık Empty, Basic, Admin,
MVC ve CRUD sunar; her proje bu sürüme ait İngilizce belge paketini alır.

İlk TTY `ahdcode dev` oturumu yerel `.test` adlarını etkinleştirmeden önce
bir kez sorar. Onaydan sonra yeni proje konak adları otomatik tutulur.
`ahdcode local hosts apply` mutlu yolda zorunlu bir adım değil, kurtarma
komutudur. `ahdcode databases`, kurulu CLI'ye gömülü tam sürüm
AhdDataStudio'yu başlatır; AhdCode deposuna veya `AHDCODE_ROOT` değerine
ihtiyaç duymaz. `AHDCODE_ROOT` yalnızca geliştirici geçersiz kılmasıdır.

Yerel geliştirme hâlâ HTTP'dir: yerel TLS, sertifika otoritesi veya ACME
yoktur. Bu sürümde dil sözdizimi değişmemiştir. Bkz.
[CLI](docs/CLI_TR.md#yerel-geliştirme-test-adları-ve-yönlendirici) ve
[Web](docs/WEB_TR.md).

v0.18.5, **Web Starter ve Uygulama Başlangıcı**, `ahdcode init web` komutunu
Empty, Basic veya Admin seçen bir sihirbaza çevirir. Empty cilalı bir
karşılama uygulamasıdır. Basic ortak uygulama ve posta yapılandırmasını
ekler. Admin giriş, pano ve SQLite veya MySQL üzerinde bir yönetici hesabı
ekler. v0.17 rota, bekçi, form, CSRF ve flash API'leri değişmez.
[Web](docs/WEB_TR.md#18-v018-web-starterlar-ve-uygulama-başlangıcı) bölümüne bakın.

v0.17.0, **Web Init, Bağlam Rotaları, Gruplar ve Bekçiler**, `ahdcode init web`
ile bağlam duyarlı rota kaydı, rota grupları ve sıralı, politikasız bekçiler
ekler. Sonlandırıcı hâlâ yalnızca `context.respond`'dur. Genel ara katman
zinciri, kimlik çerçevesi veya ORM yoktur.
[Web](docs/WEB_TR.md#17-v017-bağlam-rotaları-gruplar-ve-bekçiler) ve
[`examples/v0.17/routes_guards`](examples/v0.17/routes_guards) bölümüne bakın.

v0.16.0, **İstek Bağlamı, Formlar, Doğrulama, CSRF ve Flash**, açık istek/oturum
bağlamı (`RequestContext`), tek seferlik yanıt ve oturum sonlandırma
(`context.respond`), türlenmiş form okuma (`Forms`), sıralı doğrulama hataları
(`ValidationErrors`), seçilmiş güvenli eski girdi (`OldInput`), oturuma bağlı
CSRF (`Web.UI.csrfField`) ve tüketilen flash mesajları (`context.flashSet`,
`context.flashTake`) ekler. Uygulama gömülü AhdCode'dur; mevcut
HTTP/Session/Web.UI API'leri ve v0.15.1 temiz yolları uyumludur.
[İş akışı ve kesin API](docs/WEB_TR.md#16-v016-istek-bağlamı-formlar-doğrulama-csrf-ve-flash)
ile [çalıştırılabilir örneğe](examples/v0.16/forms_validation/README_TR.md) bakın.

v0.15.1, URL sorgu parametreleri yerine kaynak rotaları için temiz yol
parametreleri sağlayan tek-segmentli HTTP yol sonu joker karakterini (`/*`) ekler.

v0.15.0, **Web Temelleri (Web Foundations)**, [`Web`](docs/WEB_TR.md)'i
ekler: çoğunlukla AhdCode'un kendisiyle yazılmış ve derleyiciye gömülü
birinci taraf bir web çatısı. Böylece `bring Web` çevrimdışı çözülür — paket
yöneticisi, kayıt defteri, manifest veya kilit dosyası yoktur — ve üretilen
çalıştırılabilir dosya çatı kaynağına çalışma zamanında bağımlı kalmaz.
Mevcut HTTP ve HTML ilkellerini değiştirmez, bileştirir ve türlerini
değiştirmeden yeniden dışa aktarır: `Web` üzerinden ulaşılan bir `Request`,
HTTP'nin `Request`'idir.
[`Web.UI`](docs/WEB_TR.md#9-webui) anlamsal bir HTML bileşen katmanıdır —
`section`, `h1`, `p`, `a`, `img`, `table`, `form` ve gerisi — her metin giriş
noktası kaçışlar ve hiçbir yerde ham işaretleme yardımcısı yoktur.
Sayfalar, Yerleşimler ve Bileşenler `HTMLNode` döndüren sıradan
Function'lardır: sanal DOM yok, hydration yok, şablon dili yok, JavaScript
çalışma zamanı yok. Dondurulmuş bir ortam sözleşmesi (`APP_NAME`, `APP_ENV`,
`APP_HOST`, `APP_PROTOCOL`, `SERVER_HOST`, `SERVER_PORT`) ters vekil
dağıtımları için genel URL ile bağlanma adresini ayrı tutar; sessiz
varsayılan yoktur ve hiçbir `.env` değeri bir ikiliye gömülmez. Bilinçli
olarak kapsam dışı kalanlar: ORM, ara katman, yetkilendirme, paketleyici ve
tarayıcı canlı yenilemesi. `<APP_HOST>.test` için yerel
güvenilir HTTPS, yaklaşık bir çözüm üretmek yerine
[ertelenmiştir](docs/WEB_TR.md#14-yerel-https--mevcut-sınır): kalıcı olarak
ayrıcalıklı sistem durumu gerektirirdi.

v0.14.1, `require(...)` için bir araç düzeltmesidir (dil sunucusu,
biçimlendirici ve editör vurgulaması). Dil anlambilimi v0.14.0 ile aynıdır.

v0.14.0, **Uygulama Temelleri (Application Foundations)**, daha büyük
sunucu taraflı AhdCode uygulamaları için kalan çerçeveden bağımsız
temelleri ekler: paket yöneticisi olmadan bir programın dosyalara
bölünebilmesi için derleme-zamanı yerel kaynak birleştirmesi
([`require(...)`](docs/REQUIRE_TR.md)); `ahdcode dev` (v0.13) artık yalnızca
giriş dosyasını değil, çözümlenmiş tüm `require(...)` grafiğini izler, bu
yüzden require edilen herhangi bir dosyayı düzenlemek otomatik olarak
yeniden derler ve yeniden başlatır; ve [`server.static`](docs/HTTP_TR.md#statik-dosyalar),
yerel statik varlıkları (CSS, JS, SVG, görseller, fontlar) yol geçişi,
sembolik bağ kaçışı ve gizli dosya korumasıyla tek bir açık dosya sistemi
kökünden sunar, böylece her uygulama bunu kendisi yeniden uygulamak zorunda
kalmaz. Bu kasıtlı olarak bir Web çerçevesi sürümü değildir: şablon dili,
form/middleware/router çerçevesi, paket yöneticisi veya tarayıcı canlı
yenileme yoktur. [AhdDataStudio](tools/AhdDataStudio/README_TR.md), kendi
dogfood'u olarak bu temeller üzerine yeniden yapılandırılır; tek dosyadan
sorumluluğa göre gruplanmış dosyalara bölünür, davranış değişikliği olmadan.
v0.13.0, `ahdcode dev` ekler: `ahdcode build`/`run`'ın zaten kullandığı aynı
derleme hattı etrafında bir orkestrasyon olarak kurulmuş, MAMP/Vite
tarzı önplan izle-yeniden derle-yeniden başlat döngüsü; artı `ahdcode stop`,
mevcut (varsayılan olarak zorlamalı) `kill`'in nazik karşılığı — `stop`,
yalnızca sinyal göndermek yerine sürecin gerçekten çıktığını doğrulamak için
bekler. Başarısız bir yeniden derleme, ilk derleme dahil, önceden çalışan
süreci her zaman dokunulmadan çalışır bırakır; başarılı bir derlemeden
sonraki bir çalışma zamanı çökmesi, aynı ikili dosyayı yeniden denemeden
bildirilir.
v0.12.0, [AhdDataStudio](tools/AhdDataStudio/README_TR.md) ekler: AhdCode ile
yazılmış birinci taraf, yalnızca localhost MySQL + SQLite geliştirme
uygulaması — derleyici yerleşik bir modülü değildir. `ahdcode databases` ile
başlatılır ve
[http://ahddatabasestudio.test/](http://ahddatabasestudio.test/) (ya da
[http://127.0.0.1:8081/AhdDataStudio](http://127.0.0.1:8081/AhdDataStudio))
adresinde açılır. Yalnızca `127.0.0.1` dinler, MySQL şemalarını
`database: null` ile keşfeder, SQLite dosyalarını yapılandırılmış proje
yollarıyla sınırlar ve üretilen CRUD için CSRF korumalı POST formları
kullanır. Bu sürüm ayrıca iç içe hatalı String değişmezlerinde ayrıştırıcının
takılı kalmasını düzeltir. v0.11 MySQL, gömülü vendor sürücü sayesinde
çevrimdışı derlenebilir kalır. v0.11.0, [MySQL](docs/MYSQL_TR.md) ekler: `MySQL.connect` gerçek bir
sunucuyu arar ve döndürmeden önce erişilebilir olduğunu doğrular; `database`
`null` olabilir, böylece bir bağlantı herhangi biri seçilmeden önce kimlik
bilgilerinin görebildiği her veritabanını `SHOW DATABASES` ile listeleyebilir;
her sorgu sunucu taraflı parametre bağlıdır; bağımsız bir
`MySQLTransaction` eşzamanlı isteklerle asla değişebilir durum paylaşmaz; ve
`DECIMAL`/ikili değerler zorlanmak yerine kesin kalır. Gömülü
`github.com/go-sql-driver/mysql` sürücüsü AhdCode'un kendisine yerleşiktir
ve üretilen bir programın derlemesine `vendor/` olarak kopyalanır, böylece
MySQL kullanan bir program yine de tamamen çevrimdışı derlenir — diğer her
üretilmiş programın zaten sahip olduğu aynı garanti. v0.10.0,
[Security](docs/SECURITY_TR.md) ekler: `Security.passwordHash` /
`passwordVerify`, Argon2id parola hashlemeyi kendini tanımlayan tek bir
saklanan dize arkasına sarar; `Security.token`, 256 bitlik URL-güvenli
rastgele bir belirteç döndürür; ve `Security.secureEqual`, CSRF belirteçleri
ve benzerleri için iki String'i sabit zamanda karşılaştırır — üç odaklı
ilkel, bir kimlik doğrulama çerçevesi değil. v0.9.1, ikili-güvenli
[HTTP](docs/HTTP_TR.md) dosya yanıtları ekler: `HTTP.file` ve
`HTTP.download`, saklanan bir dosyanın tam baytlarını, hiçbir zaman bir
AhdCode `String`'inden geçirmeden istemciye geri akıtır; açık bir
`contentType` ve, `download` için, saklanan yoldan bağımsız ve ASCII
olmayan isimler için bile güvenle kodlanmış bir sunum dosya adı ile.

v0.9.0, yalnızca gönderim yapan [SMTP](docs/SMTP_TR.md) postası ekler:
değiştirilemez bir `SMTPClient` host, port ve güvenliği (`starttls`, `tls`
veya açık `none`) yapılandırır; değiştirilemez bir `SMTPMessage` To/Cc/Bcc,
Reply-To, UTF-8 konu ve metin ve/veya HTML gövde taşır; `send` her ileti
için bir SMTP bağlantısı açar ve AUTH PLAIN yalnızca TLS sonrasındadır.
IMAP, ek, posta kuyruğu ve sağlayıcı kısayolu yoktur. v0.8.0, multipart form işlemeyi ve güvenli dosya yüklemeyi ekler: bir
işleyici yüklenen dosyaları `Request.file` / `Request.files` ile okur;
`originalName`, `size` ve hem bildirilen hem de içerikten algılanan MIME
türünü inceler; birini `UploadedFile.save(directory)` ile yükleyenin
etkileyemeyeceği kriptografik rastgele bir adla kalıcılaştırır — böylece
düşmanca bir dosya adı ne dizinden kaçabilir ne de mevcut bir dosyayı ezebilir.
Yüklenen baytlar asla bir AhdCode `String`'i ve asla veritabanı BLOB'u
değildir: uygulamalar dosyayı diskte tutar, [SQLite](docs/SQLITE_TR.md)'ta
yalnızca yolunu ve meta verisini saklar. Ayrıca `ahdcode run` ile başlatılan
bir uygulamayı port ve pid aramadan durduran `ahdcode kill app.run` gelir.
v0.7.0, mevcut [HTML](docs/HTML_TR.md) oluşturucusunun üzerine HTML
ayrıştırma ve küçük bir CSS benzeri seçici dili ekler: `HTML.parse(source)`,
bir HTML String'ini -- tipik olarak bir `HTTP` Client yanıt gövdesini --
salt okunur bir `HTMLDocument`'e dönüştürür; `select`/`first` de bunun
içinde etiket, id, class, öznitelik ve soy/çocuk birleştiricileriyle
`HTMLElement` değerleri bulur. Ayrıştırma asla bir ağ kaynağı getirmez ve
asla betik içeriği çalıştırmaz; yalnızca belirteçler ve bir ağaç kurar.
v0.6.0, HTTPS, zaman aşımı ve açık JSON/Env API birlikte çalışması olan giden
bir [HTTP](docs/HTTP_TR.md) Client ekler. v0.5.0, v0.4.0 web temelinin üzerine
HTTP çerezleri ve bellek içi sunucu taraflı oturumlar eklemişti: tipli bir
HTTP sunucusu, Request/Response değerleri ve küçük, güvenli bir
[HTML](docs/HTML_TR.md) oluşturucu; böylece bir AhdCode programı bu makinenin
tarayıcısında açılabilir. Bir oturum çerezi yalnızca opak rastgele bir kimlik
taşır; oturum değerleri sunucuda kalır ve süreç bitince kaybolur. Bu bir
kimlik doğrulama çerçevesi değildir. v0.3.0 pratik uygulama geliştirmeyi tipli
bir [SQLite](docs/SQLITE_TR.md) köprüsüyle başlatmıştı. HTTP, çalışma zamanının
içindeki Go `net/http` paketini kullanır; ayrı bir HTTP, çerez, oturum veya
istemci yardımcısı yoktur. Gelen multipart yüklemeler v0.8.0, WebSocket sunucu
uç noktaları v1.4.0 ile geldi; giden dosya ekleri, bir WebSocket istemcisi ve
bir yapay zeka satıcı modülü hâlâ sürümün parçası değildir.

v0.2.2, v0.2.1'in tanılama, hover, completion, tanıma git, belge sembolleri,
signature help ve referans bulma özelliklerinin üzerine pratik günlük AhdCode
dil sunucusunu tamamlar. v0.2.2; yeniden adlandırma, semantic vurgulama,
inlay hint, code action/quick fix, otomatik import, belge biçimlendirme,
workspace sembol araması, katlama aralıkları ve seçim aralıkları ekler — hepsi
gerçek derleyici önyüzü tarafından desteklenir, kaydedilmemiş editör
tamponları üzerinde, tam belge senkronizasyonuyla. Kapsam ve dürüst
sınırlamalar için [`docs/LSP_TR.md`](docs/LSP_TR.md)'ye bakın (derleme
grafiğiyle sınırlı referans/rename, isteğe bağlı modül keşfi, kalıcı
workspace indeksi yok). Birlikte gelen [VS Code eklentisi](editors/vscode)
aynı sunucuyu başlatır. Dil anlambilimi v0.1.20'den değişmemiştir; v0.1.20,
[PDF](docs/PDF_TR.md) ve [Archive](docs/ARCHIVE_TR.md) modüllerini ve bir
`Latex.pdf` kaynak yan dosyasını eklemişti.

```ahd
greet: Function := (
    name: String
) -> String {
    return "Hello {name}"
}

names: List<String> := ["Ali", "Ayşe"]

for name in names {
    write(greet(name))
}
```

## Neden AhdCode?

- Bildirim ve değişiklik (mutation) farklı görünür: `:=` bildirir, `=`
  değiştirir.
- Statik denetim, ilgisiz örtük dönüşümleri (implicit conversions) ve
  truthiness'i reddeder.
- Açık null olabilen türler (`T?`), koleksiyonlarla birleşirken,
  akış-duyarlı (flow-sensitive) kontroller kanıtlanmış null olmayan
  değerleri daraltır (narrow eder).
- List'ler, Pair'ler, Class'lar, Function'lar, modüller, hatalar ve yerel
  çalıştırılabilir dosyalar v0.1 çekirdeğinin bir parçasıdır.
- Yalnızca ifade içeren `lambda (<tipli parametreler>) -> <ifade>`, mevcut
  `Function` türünde bir değer oluşturur; ayrı bir çağrılabilir tür değildir.
- Küçük, kapalı bir [Class Protocol Methods](docs/PROTOCOLS_TR.md) kümesi,
  bir Class'ın `==`, sıralama, aritmetik, tekli `-` ve `str()` davranışını
  tanımlamasına izin verir.
- Bir [Regex modülü](docs/REGEX_TR.md), desenleri (patterns) `matches`,
  `find`, `findAll`, `groups`, `replace` ve `split` içeren bir `Pattern`
  değerine derler.
- [Time](docs/TIME_TR.md), saat dilimi veritabanı eklemeden yerel, UTC, sabit
  dakika ofseti ve Unix milisaniye gösterimlerini destekler.
- Sıkı [CSV modülü](docs/CSV_TR.md), ham String satırlarını veya başlık
  anahtarlı String kayıtlarını native ve kalıcı REPL uyumuyla taşır.
- Bir ifade lambda'sı, dışarıdaki değerleri açık bir bağımlılık listesiyle
  okuyabilir: lexical yakalama için `#name`/`Local name`, modül bağlaması için
  `@name`/`Global name`, örneğin
  `lambda [#minimum, @Maximum] (score: Int) -> score >= minimum and score <= Maximum`;
  her iki tür de asla çıkarılmaz ve asla örtük değildir.
- [Statistics modülü](docs/STATISTICS_TR.md), `List<Int>` ve `List<Real>`
  üzerinde tipli betimleyici istatistik sağlar; String zorlaması yoktur.
- [Numeric modülü](docs/NUMERIC_TR.md), immutable ve Real yönelimli Vector/
  Matrix değerleri, doğrusal cebir ve Plot için ek `Vector` overload'ları sağlar.
- [Word modülü](docs/WORD_TR.md), Office veya harici çalışma zamanı
  gerektirmeden immutable biçimlendirilmiş belgeler, merge edilmiş tablolar,
  gömülü Plot görselleri ve sınırlandırılmış anlamsal DOCX okuma sağlar.
- [Excel modülü](docs/EXCEL_TR.md), tipli ve değiştirilemez Workbook/Sheet/
  Cell/Range değerleriyle gerçek `.xlsx` paketlerini okur ve yazar. Formula
  niyeti açıktır, merge değer kaybını reddeder ve native çalıştırılabilirler
  çevrimdışı ve taşınabilir kalır.
- [PDF modülü](docs/PDF_TR.md), değiştirilemez `PDFDocument` değerleri
  oluşturur ve bunları `Latex`'in kullandığı aynı konuşlandırılmış Tectonic
  render motoru üzerinden çevrimdışı gerçek `.pdf` dosyalarına render eder;
  ayrıca başka bir modülün kendi tipli belgesinin anlamsal dönüşümü olan
  `PDF.fromWord`/`PDF.fromExcel` sağlar.
- [Archive modülü](docs/ARCHIVE_TR.md), dosyaları yalnızca Go standart
  kütüphanesini kullanarak çevrimdışı gerçek ZIP, TAR ve TAR.GZ arşivlerine
  paketler, yalnızca oluşturma amaçlıdır.
- [Lists](docs/LISTS_TR.md) ve [KeyValue](docs/KEYVALUE_TR.md), `List` ve
  `Pair` üzerinde saf yapısal dönüşümler ekler — `chunk`, `flatten`,
  `transpose`, `unique`, `valueCounts`, `groupBy` ve `keys`, `values`,
  `combine`, `with`, `select`, `drop`, `rename`, `mapValues`, `merge`,
  `overlay`. Tür-yönelimlidirler: her çağrının kesin sonuç türü argüman
  türlerinden hesaplanır, genel (generic) sözdizimi olmadan ve hiçbir şey
  silinmeden.
- Formatter, yorumları korurken tek bir kanonik (standart) sunum tanımlar.
- [Dil sunucusu](docs/LSP_TR.md) (`ahdcode lsp`), derleyicinin kendi
  tanılamalarını, hover'ını, tanıma git özelliğini, belge sembollerini,
  signature help'ini, referans bulmasını ve completion'ını standart stdio
  LSP üzerinden sunar — ikinci bir ayrıştırıcı yok, elle bakımı yapılan bir
  sembol kataloğu yok ve bir belge editörde açık ve kaydedilmemişken
  dosyasına asla yazılmaz.

## Tasarım İlkeleri: Sabit İlkeler, Gelişen 1.0 Öncesi Yüzey

AhdCode kalıcı tasarım ilkelerine dayanır:

- **Minimum satır sayısı yerine okunabilirlik:** Sözdizimi yapay kısalık yerine netliği ve yapıyı tercih eder.
- **Açık niyet (explicit intent):** Sade İngilizce anahtar sözcükler; bildirim (`:=`) ve değişiklik (`=`) görsel olarak ayırt edilir.
- **Katı statik tipleme:** `Any`/dinamik geri çekilme (fallback) yoktur; ilgisiz sessiz tür zorlaması (coercion) yoktur; truthiness yoktur.
- **Yalnızca güvenli ve tekil çıkarım:** Atlanan tür belirtimleri yalnızca belirsizlik olmadığında çıkarılır; derleyici asla tahmin yürütmez.
- **Belirlenirci (deterministic) davranış:** Gizli değişken çalışma zamanı durumu yoktur, sihirli küresel yan etkiler yoktur.
- **Pratik olan her yerde yeni sözdizimi yerine sıradan Function'lar ve kütüphaneler.**
- **Kanonik biçimlendirme:** `ahdcode format` tarafından uygulanan tek bir yetkili sunum stili.
- **Ürün davranışı olarak tanılama:** Eyleme geçirilebilir ipuçlarıyla yapıya duyarlı net hatalar.

### 1.0 Öncesi Dil Evrimi

AhdCode, 1.0 öncesi dönemi kalıcı olarak dondurulmuş bir durum olarak görmedi; sözdizimini gelişigüzel de değiştirmedi. Yukarıdaki temel ilkeler sabittir. Gerçek uygulama, dogfooding ve pratik uygulama ihtiyaçları somut eksikleri gösterdiğinde, 1.0 öncesi dil kararları bilinçli olarak revize edildi. Bildirim tür çıkarımı, açık null olabilen türler (`T?`), açık leksikal/küresel bağımlılık listelerine sahip yalnızca-ifade lambda'lar (`#isim`, `@isim`) ve kapalı Class Protocol Methods kümesi gibi yetenekler; statik tiplemeyi, belirlenirciliği, açıklığı ve gizli sihrin reddedilmesini kesin olarak koruyan bilinçli evrimleri yansıtır.

## Mimari Taksonomi

Kavramsal netliği korumak için AhdCode'un yetenekleri dört belirgin mimari katmanda düzenlenmiştir:

1. **Çekirdek Dil (Core Language):**
   - Açık bildirimler (`:=`) ve değişiklik (`=`), açık iç içe kapsam (`Local`, `Global`)
   - Statik tür sistemi ve null güvenliği: null olamayan `T`, açık null olabilen `T?`, `Nothing` ve akış-duyarlı null daraltması
   - İsimli Function'lar ve açık yakalamalara (`#isim`, `@isim`) sahip yalnızca-ifade lambda'lar (`lambda (...) -> ifade`)
   - Class'lar, tekli kalıtım ve on sabit [Class Protocol Method](docs/PROTOCOLS_TR.md) (`CEqual`, `CCompare`, `CAdd`, `CSubtract`, `CMultiply`, `CDivide`, `CRemainder`, `CPower`, `CNegate`, `CStr`)
   - Belirlenirci kontrol akışı (`if`/`else`, `for`/`between`, `attempt`/`except`/`ultimately`/`toss`)
   - Önceden tanımlanmış temel işlevler (`write`, `take`, `str`, `int`, `real`, `len`, `clear`, `abs`, `sum`, `min`, `max`, `type`, `id`) ve yapılandırılmış hata taksonomisi
   - Modül çözümleme (`bring`, `from ... bring`) ve derleme zamanı yerel kaynak birleştirme ([`require(...)`](docs/REQUIRE_TR.md))

2. **Standart Kütüphane (Birinci Taraf Gömülü Modüller):**
   - **Matematik ve Hesaplama:** [`Math`](docs/MATH_TR.md), [`Bits`](docs/BITS_TR.md) (`Int` üzerinde bit işlemleri), [`Regex`](docs/REGEX_TR.md), [`Statistics`](docs/STATISTICS_TR.md), [`Numeric`](docs/NUMERIC_TR.md), [`Plot`](docs/PLOT_TR.md), [`Graphics`](docs/GRAPHICS_TR.md) (Canvas pencereleri ve Turtle ile çizim), [`GUI`](docs/GUI_TR.md) (tıklama ve tuş geri çağırmalı küçük masaüstü pencereleri)
   - **Veri ve Koleksiyonlar:** [`Lists`](docs/LISTS_TR.md), [`KeyValue`](docs/KEYVALUE_TR.md), [`Characters`](docs/CHARACTERS_TR.md) (Unicode kod noktaları ve sınıflandırma), [`CSV`](docs/CSV_TR.md), [`Data`](docs/DATA_TR.md), [`JSON`](docs/JSON_TR.md), [`XML`](docs/XML_TR.md), [`UUID`](docs/UUID_TR.md) (RFC 9562 sürüm 4 ve zamana göre sıralı sürüm 7 kimlikleri)
   - **Belge Üretimi:** [`Word`](docs/WORD_TR.md), [`Excel`](docs/EXCEL_TR.md), [`PDF`](docs/PDF_TR.md), [`Latex`](docs/LATEX_TR.md), [`QR`](docs/QR_TR.md) (QR kodları), [`Barcode`](docs/BARCODE_TR.md) (Code 128, EAN-13, UPC-A), [`Archive`](docs/ARCHIVE_TR.md)
   - **Sistem ve Ortam:** [`Time`](docs/TIME_TR.md), [`Cron`](docs/CRON_TR.md) (sınırlı, süreç içi zamanlama), [`Path`](docs/FILESYSTEM_TR.md), [`File`](docs/FILESYSTEM_TR.md), [`Env`](docs/ENV_TR.md), [`Terminal`](docs/TERMINAL_TR.md) (v1.5.0: standart hata, tampon boşaltma, terminal tespiti ve boyutu, biçimli metin, okunabilir düzen)

3. **Birinci Taraf Çalışma Zamanı / Çatı Modülleri:**
   - **Ağ, Sunucu ve Depolama İlkelleri:** [`HTTP`](docs/HTTP_TR.md) (bellek içi sunucu, istek/yanıt, çerezler, oturumlar, statik dosya sunucusu, [WebSocket uç noktaları](docs/WEBSOCKET_TR.md), client), [`HTML`](docs/HTML_TR.md) (anlamsal kurucu, ayrıştırıcı, seçici motoru), [`Security`](docs/SECURITY_TR.md) (Argon2id özetleme, bcrypt uyumluluğu, güvenli token'lar, sabit zamanlı karşılaştırma, SHA-2 özetleri, HMAC, kodlamalar, RS256 imzaları, AES-256-GCM), [`SQLite`](docs/SQLITE_TR.md) (yerel tipli veritabanı köprüsü), [`MySQL`](docs/MYSQL_TR.md) (bağlantı havuzu ve işlemlerle ağ veritabanı), [`PostgreSQL`](docs/POSTGRESQL_TR.md) (bağlantı havuzu ve işlemlerle ağ veritabanı), [`SMTP`](docs/SMTP_TR.md) (yalnızca gönderim yapan posta istemcisi)
   - **Web Uygulama Çatısı:** [`Web`](docs/WEB_TR.md) (birinci taraf gömülü web çatısı, [`Web.UI`](docs/WEB_TR.md#9-webui) anlamsal bileşenleri, `RequestContext`, tipli `Forms`, sıralı `ValidationErrors`, seçilmiş `OldInput`, oturuma bağlı CSRF ve flash yaşam döngüsü)

4. **Geliştirici Araçları:**
   - **Derleyici ve Araç Zinciri:** `ahdcode build`, `ahdcode run`, `ahdcode dev` (otomatik `.test` kimlikleriyle izleme-yeniden derleme döngüsü), `ahdcode stop`, `ahdcode local` (yerel rotalar ve yönetilen konak adları)
   - **Kanonik Biçimlendirici:** `ahdcode format` (sözdizim ağacı güdümlü, yorum koruyan)
   - **Etkileşimli REPL:** `ahdcode repl` (kalıcı çok satırlı ortam)
   - **Editör ve Dil Sunucusu:** `ahdcode lsp`, resmi VS Code / Antigravity eklentisi (`editors/vscode`)
   - **Tanılama Motoru:** Net hata kodları ve ipuçlarıyla yapıya duyarlı derleyici tanılamaları
   - **Yerel Geliştirici Arayüzü:** [AhdDataStudio](tools/AhdDataStudio/README_TR.md) (localhost MySQL ve SQLite yönetim aracı), `ahdcode databases list|add|remove` (kullanıcıya özel SQLite kayıt defteri)

## Kurulum

[Kurulum](docs/INSTALLATION_TR.md) belgesindeki platform paketini kullanın. Özel Go araç zinciri, çevrimdışı LaTeX, yardımcı programlar, Studio, starter ve belgeler pakete dahildir.

## Kaynak koddan derleme

AhdCode şu anda Go 1.26 veya daha yeni bir sürüm gerektirir.

```bash
cd AhdCode
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot ./cmd/ahdsqlite
go -C cmd/ahdgraphics install .
go -C cmd/ahdgui install .
go -C cmd/ahdplotview install .
```

Yukarıdaki komutlar derleyiciyi ve yerel numeric, plot, SQLite, Graphics pencere ve GUI pencere yardımcılarını (helpers) ve etkileşimli Plot görüntüleyicisini kurar. Bu pencere yardımcıları kendi Go modülleri olduğundan `go -C cmd/ahdgraphics install .`, `go -C cmd/ahdgui install .` ve `go -C cmd/ahdplotview install .` ile kurulur.
Eğer `Latex` modülünü **veya** `PDF` modülünün `.save()` metodunu kullanmayı
planlıyorsanız (ikisi de aynı çevrimdışı render motorunu paylaşır),
çevrimdışı (offline) Latex/Tectonic çalışma zamanını da hazırlamanız (stage)
gerekir. `Archive` böyle bir hazırlığa ihtiyaç duymaz — yalnızca Go standart
kütüphanesini kullanır. Hazırlık işlemi, sabitlenmiş ve doğrulanmış
kaynakları indirmek için bir defaya mahsus ağ bağlantısı kullanır:

```bash
go run ./tooling/latex/cmd/package-latex --output "$(go env GOPATH)"
```

Hazırlık (staging) aşamasından sonra, AhdCode'un normal Latex işlemleri tamamen çevrimdışı çalışmaya devam eder.

Go'nun ikili dosya (binary) dizininin `PATH`'te olduğundan emin olun:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
ahdcode --version
```

## CLI hızlı başlangıç

```bash
mkdir my-app
cd my-app
ahdcode init web
ahdcode dev app.ahd
```

`dev`, hem bağlandığı soketi hem de artık yönlendirdiği yerel adı yazar:

```text
  Open:
  http://127.0.0.1:8080

  Local identity:
  http://ahdakademi.test/
  Bind: 127.0.0.1:8080
```

Bir terminalde `ahdcode init web` hangi starter'ın yazılacağını sorar:

- **Empty** — karşılama uygulaması; veritabanı ve giriş yok
- **Basic** — aynı kabuk artı ortak `.env` / posta yapılandırması
- **Admin** — Home, Login, Dashboard ve SQLite veya MySQL üzerinde bir yönetici

Starter'ı doğrudan da verebilirsiniz: `ahdcode init web empty`, `basic` veya
`admin`. Şablonlar ve Bootstrap 5.3.3 CLI içindedir (MIT, yerel dosyalar,
CDN yok). `.env` gitignore'dadır; Admin SQLite `database/*.db` dosyaları da
yok sayılır. Var olan dosyaların ve var olan veritabanlarının üzerine
yazılmaz.

```bash
ahdcode run examples/v0.1/01_hello.ahd
ahdcode dev examples/v0.1/01_hello.ahd
ahdcode build examples/v0.1/01_hello.ahd -o hello
ahdcode format examples/v0.1/01_hello.ahd
ahdcode format --check examples/v0.1/01_hello.ahd
ahdcode
```

[CLI rehberine](docs/CLI_TR.md), [formatter rehberine](docs/FORMATTER_TR.md),
[REPL rehberine](docs/REPL_TR.md) ve [dil sunucusu rehberine](docs/LSP_TR.md)
bakın.

## Dokümantasyon

- [Türkçe Öğrenci Rehberi](docs/STUDENT_GUIDE_TR.md)
- [Web — birinci taraf web çatısı](docs/WEB_TR.md)
- [require(...) — yerel kaynak birleştirme](docs/REQUIRE_TR.md)
- [Uygulamalı Modül Atölyeleri](docs/PRACTICAL_MODULES_TR.md) — CSV, Data,
  Plot, Excel, Word, Latex, HTTP(S) ve HTML'i uçtan uca projelerle öğrenin
- [English Student Guide](docs/STUDENT_GUIDE_EN.md)
- [Başlangıç](docs/GETTING_STARTED_TR.md)
- [Dil Turu](docs/LANGUAGE_TOUR_TR.md)
- [Türler ve Null Güvenliği](docs/TYPES_AND_NULL_TR.md)
- [Kontrol Akışı](docs/CONTROL_FLOW_TR.md)
- [Fonksiyonlar](docs/FUNCTIONS_TR.md)
- [Sınıflar](docs/CLASSES_TR.md)
- [Class Protocol Methods](docs/PROTOCOLS_TR.md)
- [Koleksiyonlar](docs/COLLECTIONS_TR.md)
- [Modüller](docs/MODULES_TR.md)
- [Hatalar](docs/ERRORS_TR.md)
- [Temel İşlevler](docs/FUNDAMENTALS_TR.md)
- [String API](docs/STRING_API_TR.md)
- [List API](docs/LIST_API_TR.md)
- [Math modülü](docs/MATH_TR.md)
- [Time modülü](docs/TIME_TR.md)
- [Latex modülü](docs/LATEX_TR.md)
- [Word modülü](docs/WORD_TR.md)
- [Excel modülü](docs/EXCEL_TR.md)
- [PDF modülü](docs/PDF_TR.md)
- [QR modülü](docs/QR_TR.md)
- [Barcode modülü](docs/BARCODE_TR.md)
- [Archive modülü](docs/ARCHIVE_TR.md)
- [File ve Path modülleri](docs/FILESYSTEM_TR.md)
- [Regex modülü](docs/REGEX_TR.md)
- [CSV modülü](docs/CSV_TR.md)
- [Data modülü](docs/DATA_TR.md)
- [Statistics modülü](docs/STATISTICS_TR.md)
- [Plot modülü](docs/PLOT_TR.md)
- [Numeric modülü ve Complex skalerleri](docs/NUMERIC_TR.md)
- [JSON modülü](docs/JSON_TR.md)
- [SQLite modülü](docs/SQLITE_TR.md)
- [PostgreSQL modülü](docs/POSTGRESQL_TR.md)
- [HTTP modülü](docs/HTTP_TR.md)
- [WebSocket uç noktaları](docs/WEBSOCKET_TR.md)
- [HTML modülü](docs/HTML_TR.md)
- [SMTP modülü](docs/SMTP_TR.md)
- [XML modülü](docs/XML_TR.md)
- [Env modülü](docs/ENV_TR.md)
- [Lists modülü](docs/LISTS_TR.md)
- [KeyValue modülü](docs/KEYVALUE_TR.md)
- [UUID modülü](docs/UUID_TR.md)
- [Terminal modülü](docs/TERMINAL_TR.md)
- [Graphics modülü](docs/GRAPHICS_TR.md)
- [GUI modülü](docs/GUI_TR.md)
- [Masaüstü uygulaması paketleme](docs/PACKAGING_TR.md)
- [Tanılamaları anlama](docs/DIAGNOSTICS_TR.md)
- [Dil sunucusu](docs/LSP_TR.md)
- [Yapay zekâ destekli yerel kurulum](FOR_AI.md)
- [Derlenmiş v0.1 örnekleri](examples/v0.1/README_TR.md)
- [v0.3 SQLite Not Defteri](examples/v0.3/README_TR.md)
- [v0.4 Web Not Defteri](examples/v0.4/README_TR.md)
- [v0.5 çerezler ve oturumlar](examples/v0.5/README_TR.md)
- [v0.6 HTTP Client](examples/v0.6/README_TR.md)
- [v0.7 HTML ayrıştırma ve web kazıma](examples/v0.7/README_TR.md)
- [v0.8 multipart formlar ve dosya yükleme](examples/v0.8/README_TR.md)
- [v0.9 SMTP posta gönderimi](examples/v0.9/README_TR.md)
- [v0.12 MySQL çekiliş](examples/v0.12/raffle/README_TR.md) — katılım kodu, hash’li yönetici girişi, kazanan ilanı
- [v1.4 gerçek zamanlı yoklama](examples/v1.4/realtime_attendance/README_TR.md) — Web, PostgreSQL, WebSocket, UUID v7, `Env.secret` ve Cron
- [v1.5 Terminal tanıtımı](examples/v1.5/terminal_demo/README_TR.md) — `Terminal.emit`, standart hata, terminal tespiti, biçimli metin ve okunabilir düzen
- [v1.7 standart kütüphane tamamlama](examples/v1.7/README_TR.md) — trigonometri ve ebob/ekok, katı ISO zaman metni ve an aritmetiği, istatistikli Table join
- [v2.0 masaüstü uygulamaları](examples/v2.0/README_TR.md) — TableView, iletişim kutuları ve CSV dışa aktarma içeren bir SQLite defteri; bir 3B Surface; ve paketlenecek küçük bir uygulama
- [v2.1 uygulama G/Ç ve ağ](examples/v2.1/README_TR.md) — HTTP üzerinden ikili bir dosya turu, bir WebSocket istemci/sunucu çifti ve ekli posta
- [v2.2 öğrenci performansı](examples/v2.2/student_performance/README_TR.md) — aynı notları çizgi, çubuk, histogram, kutu, hata çubuğu, pasta, ısı haritası ve 3B yüzey olarak gösteren tek bir masaüstü uygulaması
- [v2.3 örnekleri](examples/v2.3/README_TR.md) — isimli Function `uses` capture'ları ve Plot/Surface matematik etiketleri
- [v1.9 GUI renkleri ve etkileşimli Plot](examples/v1.9/README_TR.md) — renkli ve devre dışı Kaydet düğmeli bir sipariş formu ile AhdCode'un kendi görüntüleyicisinde bir grafik
- [v1.8 GUI ve olaylar](examples/v1.8/README_TR.md) — SQLite destekli küçük bir GUI defteri ve ok tuşları ile tıklamalarla yönetilen Turtle
- [v1.6 Graphics ve Turtle](examples/v1.6/graphics_turtle/README_TR.md) — Kartezyen Canvas, şekiller, Turtle ile yıldız ve spiral, PNG/SVG kaydı ve düzgün çokgen dersi
- [AhdDataStudio](tools/AhdDataStudio/README_TR.md) — yerel MySQL + SQLite geliştirme arayüzü
- [v0.4 Kütüphane Demosu](https://github.com/aliharundaldalli/ahdcode-library-demo) (ayrı başlangıç web uygulaması)
- [v0.4 Seminer Demosu](https://github.com/aliharundaldalli/ahdcode-seminer-demo) (Hatay, çok sayfalı)
- [v0.16 Matematik Portalı](https://github.com/aliharundaldalli/ahdcode-math-portal) (RequestContext, form, doğrulama, CSRF, flash)
- [Tam v0.1 dil spesifikasyonu](AHDCODE_LANGUAGE_SPEC_v0.1_TR.md)

## Editör eklentisi

[`editors/vscode`](editors/vscode) içindeki yerel, VS Code uyumlu eklenti,
`.ahd` dosyalarını tanır, sözdizimi vurgulama sağlar, editör başlık
çubuğundaki oynat düğmesi, Komut Paleti veya `F6` ile aktif dosyayı
çalıştırır ve derleyici destekli tanılamalar ve hover için
[dil sunucusuna](docs/LSP_TR.md) (`ahdcode lsp`) bağlanır. Aynı VSIX, hem
VS Code hem de Antigravity'i hedefler.
[Kurulum rehberine](editors/vscode/README_TR.md) bakın.

## Mevcut sınırlamalar

## v2.3.0 ile gelenler <a id="v23-ile-gelenler"></a>

v2.3.0, **Dil Ergonomisi ve Plot Cilası**, odaklanmış bir dil ve
görselleştirme sürümüdür:

- yürütülebilir blok içindeki isimli Function'lar, lambda capture modeliyle
  aynı `#` lexical ve `@` module capture'larını açık `uses` listesiyle
  bildirebilir;
- derleyici, biçimlendirici, tanılamalar ve dil sunucusu capture'ları
  completion, definition, hover, references, rename, semantic token ve eksik
  capture quick fix düzeylerinde tanır;
- referans ve rename, arka plan indeksi olmadan, derleyici kaynaklı sınırlı
  çalışma alanı görüntülerinde ve içe aktarılan dosyalarda çalışır;
- Plot Surface görüntüleyicisindeki Save mevcut görünür kamera görünümünü
  kaydeder; programatik `Surface.save(path)` kanonik ve belirlenimci kalır;
- Plot ve Surface etiketleri mevcut çevrimdışı math-text yolu üzerinden
  bütün String'i saran `$...$` matematiğini kabul eder. Karışık zengin metin,
  animasyon ve genel 3B kamera/sahne API'si kapsam dışıdır.

[v2.3 örneklerine](examples/v2.3/README_TR.md),
[Fonksiyonlar](docs/FUNCTIONS_TR.md), [LSP](docs/LSP_TR.md) ve
[Plot](docs/PLOT_TR.md) referanslarına bakın.

## v2.2 ile gelenler <a id="v22-ile-gelenler"></a>

v2.2.0, **Plot'un Tamamlanması**, bilinçli olarak küçük bir görselleştirme
sürümüdür. AhdCode'un kendi GUI ve Plot modüllerini birlikte kullanırken
ortaya çıkan boşlukları kapatır; yeni sözdizimi ve tip sistemi değişikliği
eklemez.

- [`Plot.pie(labels, values)`](docs/PLOT_TR.md#pasta): verilen sırayla
  kategori başına bir dilim; kategori göstergesi varsayılan olarak açıktır
  ve her dilimin üzerinde payı yazar. Pastanın ekseni yoktur; bu yüzden
  üzerinde `xLabel` ve `yLabel` sessizce yok sayılmak yerine `PlotError`
  fırlatır.
- [`Plot.heatmap(xLabels, yLabels, values)`](docs/PLOT_TR.md#isı-haritası):
  rengi sayıları taşıyan etiketli bir ızgara ve bir renk ölçeği göstergesi.
  Matrix, y etiketi başına bir satır ve x etiketi başına bir sütundur.
- [`Surface.xCategories` ve `Surface.yCategories`](docs/PLOT_TR.md#koordinatları-adlandırmak):
  bir Surface'in x ve y koordinatlarına ad verir; böylece dersler ve
  yıllardan oluşan bir ızgara `1`–`5` diye etiketlenmekten kurtulur.
  Yalnızca sunumdur — geometri değişmez ve `xLabel`/`yLabel` eksen başlığı
  olarak kalır.

Yeni grafiklerin ikisi de sıradan `Chart` değerleridir: `title`, `legend`,
`size`, PNG/SVG/PDF'e `save`, etkileşimli görüntüleyicide `show` ve bir
`Plot.subplots` Figure'ında bir hücre alırlar.

Bkz. [v2.2 örneği](examples/v2.2/student_performance/README_TR.md).

## v2.1.0 ile gelenler <a id="v210-ile-gelenler"></a>

v2.0 masaüstü uygulaması temelini tamamladı. v2.1, **Uygulama G/Ç ve Ağ
Tamamlanması**, CLI, Web ve masaüstü programlarının paylaştığı pratik dosya
ve ağ boşluklarını kapatır. Yeni sözdizimi, tip sistemi değişikliği ya da
bayt tipi eklemez: ikili veri opak kalır ve dosyadan dosyaya taşınır.

- [`HTTP`](docs/HTTP_TR.md): `Client.download(url, path)` ve
  `Client.sendToFile(request, path)` bir yanıt gövdesini doğrudan dosyaya
  akıtır ve bir `ClientFileResponse` döndürür — durum, başlıklar, son URL ve
  yazılan bayt sayısı; `body()` yoktur, çünkü yük dosyadadır. Başarısız bir
  aktarım hedefe asla zarar vermez.
- [`HTTP`](docs/HTTP_TR.md): `ClientRequest.withMultipartField` ve
  `withMultipartFile` bir `multipart/form-data` gövdesi gönderir ve her
  dosyayı diskten akıtır. Giden multipart, v0.8'in eklediği gelen
  yüklemelerle çifti tamamlar.
- [`WebSocket`](docs/WEBSOCKET_TR.md): eşzamanlı bir istemci —
  `HTTP.webSocketClient(url)` yapılandırır, `connect()` bir
  `WebSocketConnection` açar ve `receive()` sıradaki mesajı bekler. Geri
  çağrı, arka plan olay döngüsü ya da yeniden bağlanma yoktur.
- [`SMTP`](docs/SMTP_TR.md): `SMTPMessage.withAttachment(path, fileName, contentType)`
  gerçek dosyaları, doğrudan diskten base64 ile kodlayarak ekler. Eksiz bir
  ileti her zamanki MIME'ın tam olarak aynısını üretir.
- v2.0 Plot görüntüleyicisinin Save'i, Plot kullanan ama GUI kullanmayan bir
  program için düzeltildi; hiçbir ortam değişkenine ihtiyaç duymaz.

Bkz. [v2.1 örnekleri](examples/v2.1/README_TR.md).

## v2.0.0 ile gelenler <a id="v200-ile-gelenler"></a>

v2.0.0 bir **ana** sürümdür, **Masaüstü Uygulamasının Tamamlanması**. Yeni
sözdizimi veya tip sistemi değişikliği olmadan temel masaüstü yol haritasını
kapatır; her v1.9 programı çalışmaya devam eder:

- [`GUI`](docs/GUI_TR.md): ListBox, Select, TextArea, PasswordInput ve
  kaydırma ile seçim içeren okuma odaklı bir TableView; metin alanları ve
  Checkbox'lar için `onChange` ile seçim callback'leri; tablo ve listeleri
  fazladan alanı kullanan yeniden boyutlandırılabilir pencereler; Shift+Tab;
  ve yerel dosya, klasör, kaydetme, mesaj ve onay iletişim kutuları. Temel
  GUI yol haritası tamamlanmıştır: sonraki eklemeler eksik platform temelleri
  değil, kullanım odaklı bileşenlerdir.
- [`Plot`](docs/PLOT_TR.md): görüntüleyici bir araç çubuğu kazanır — Save
  (PNG, SVG, PDF; render aracı tarafından tam `save()` gibi yazılır), Zoom
  Out, Zoom In, Rotate Left, Rotate Right ve Fit — ve `Plot.surface` bir
  Numeric Matrix'ten 3B yüzey çizer; döndürme/kaydırma/yakınlaştırma
  görüntüleyicisi, tel kafes kipi ve PNG dışa aktarma ile.
- [`ahdcode package`](docs/PACKAGING_TR.md), bir programı masaüstü
  uygulamasına dönüştürür — bir macOS `.app`'i ya da bir Windows veya Linux
  klasörü ve arşivi — yalnızca gereken yardımcıları içerir ve AhdCode kurulu
  olmadan çalışır.
- Editörler her Chart ve Figure üyesini tamamlar, üzerine gelince gösterir ve
  imzasını gösterir; v1.8 ve v1.9'daki bir boşluk giderilmiştir.

Bkz. [v2.0 örnekleri](examples/v2.0/README_TR.md).

## v1.9.0 ile gelenler <a id="v190-ile-gelenler"></a>

v1.9.0 bir **ara (minor)** sürümdür, **Masaüstü İnceltmeleri + Etkileşimli
Plot**. `show()`'un bir grafiği sunma biçimini değiştirir, GUI'ye renkler ve
etkinlik durumu ekler ve AhdCode'un kendi pencerelerini adlandırır; çekirdek
dilbilgisi ve tip sistemi değişmez.

- [`Plot`](docs/PLOT_TR.md): `chart.show()` ve `figure.show()`, işletim
  sisteminin görüntüleyicisi yerine AhdCode'un kendi etkileşimli
  görüntüleyicisini açar: imleç çevresinde yakınlaştırmak için kaydırın,
  kaydırmak için sürükleyin, görünümü çeyrek tur döndürmek için Q/E,
  sıfırlamak için R, kapatmak için Escape. Bu etkileşimler yalnızca
  görüntüleyiciyi değiştirir; Chart'ı veya Figure'ı, verilerini ya da dışa
  aktarılan PNG, SVG veya PDF dosyalarını değiştirmez. `save()` değişmez.
- [`GUI`](docs/GUI_TR.md): temel renkler — Window ve Container'da
  `setBackground`, Label, Button, TextInput ve Checkbox'ta
  `setForeground`/`setBackground`; Graphics renkleri gibi yazılır — ve
  Button, TextInput ve Checkbox için etkinlik durumu
  (`setEnabled`/`isEnabled`). AhdCode GUI; formlar, yardımcı araçlar ve
  eğitim amaçlı masaüstü uygulamaları için bilinçli olarak küçük bir araç
  takımı olarak kalır.
- GUI, Graphics ve Plot görüntüleyici pencereleri AhdCode adını ve simgesini
  gösterir: macOS'ta menü çubuğunda **AhdCode**'u ve Dock'ta AhdCode
  simgesini, Windows ve Linux'ta sistemin gösterdiği yerlerde pencere
  simgesini. Bu, AhdCode'un kendi pencerelerini adlandırır; kullanıcı
  programlarını paketlemez.

Bkz. [v1.9 örnekleri](examples/v1.9/README_TR.md).

## v1.8.0 ile gelenler <a id="v180-ile-gelenler"></a>

v1.8.0 bir **ara (minor)** sürümdür, **GUI Temelleri + Asgari Olaylar**. Bir
standart modül, iki Graphics Canvas üyesi ve beş değer Class'ı için okunabilir
çıktı ekler; çekirdek dilbilgisi ve tip sistemi değişmez.

- [`GUI`](docs/GUI_TR.md): küçük masaüstü pencereleri için yeni standart
  modül. Label, Button, tek satırlık TextInput ve Checkbox bileşenleri Column
  ve Row'larla yerleştirilir; `Button.onClick` ve `Window.onKey` geri
  çağırmaları vardır. Öğrenme ve formlar, veri giriş araçları, basit
  veritabanı ön yüzleri ve yardımcı araçlar gibi küçük masaüstü uygulamaları
  içindir; Tkinter uyumlu bir araç takımı, oyun motoru veya profesyonel
  yaratıcı uygulama çatısı değildir. Her Window'u paketli `ahdgui`
  yardımcısı çizer.
- [`Graphics`](docs/GRAPHICS_TR.md): `Canvas.onClick` ve `Canvas.onKey` bir
  tıklamada (Kartezyen koordinatlarla) veya tuş basışında bir Function
  çalıştırır; böylece Turtle, terminal okunmadan ok tuşlarıyla yönetilebilir.
- Geri çağırmalar programın kendi akışında sırayla çalışır; biçimleri derleme
  zamanında denetlenir ve bir geri çağırmanın hatası `wait()` dışına aynen
  yayılır.
- `str`, `write` ve `Terminal.pretty` artık Vector, Matrix, DateTime,
  Duration ve Table değerlerinin içeriğini gösterir; örneğin
  `Vector([3.0, 4.0])` ve `Duration(1500 ms)`. Yalnızca Class adı gösterilmez.

Bkz. [v1.8 örnekleri](examples/v1.8/README_TR.md).

## v1.7.0 ile gelenler <a id="v170-ile-gelenler"></a>

v1.7.0 bir **ara (minor)** sürümdür, **Standart Kütüphane Tamamlama**. Mevcut
altı modülü eklemeli fonksiyon ve üyelerle güçlendirir; temel dilbilgisi, tip
sistemi ve daha önce yayımlanan her fonksiyon değişmeden kalır ve hiçbir
bağımlılık eklenmez.

- [`Math`](docs/MATH_TR.md): `asin`, `acos`, `atan`, `atan2`, `sinh`, `cosh`,
  `tanh`, `hypot`, `log2`, `cbrt`, `radians`, `degrees`, `gcd` ve `lcm`.
  Açılar düz Real değerler olarak kalır; ayrı bir açı tipi ve `Math.pow`
  yoktur.
- [`Time`](docs/TIME_TR.md): katı bir RFC 3339 alt kümesi için
  `Time.parseISO` ve `DateTime.toISO` (`Z` veya `±HH:MM` göstergesi
  zorunludur), `DateTime.add` ve `subtract` ile an aritmetiği, Duration için
  tam `add`, `subtract`, `negate` ve `abs`. Adlandırılmış saat dilimi
  veritabanı hâlâ yoktur.
- [`Numeric`](docs/NUMERIC_TR.md): `Vector.at`, `norm`, `outer` ve `cross`;
  `Matrix.at`, `row`, `column`, `diagonal`, `norm` (Frobenius), `hadamard` ve
  `matvec`. Broadcasting yoktur.
- [`Statistics`](docs/STATISTICS_TR.md): `covariance`, `sampleCovariance`,
  `correlation` (Pearson) ve `{"slope", "intercept"}` döndüren
  `linearRegression`. İstatistiksel test paketi yoktur.
- [`Data`](docs/DATA_TR.md): tek ortak anahtarla veya sol ve sağ anahtarla
  `Table.concat` ve `Table.innerJoin`. Yalnızca inner join; eksik değer ve
  otomatik sütun son eki yoktur.
- [`Security`](docs/SECURITY_TR.md): zaten sabitlenmiş `golang.org/x/crypto`
  üzerinden, uyumluluk ve geçiş için `bcryptHash` ve `bcryptVerify`. Yeni
  uygulamalar Argon2id'yi tercih etmelidir.

Her ekleme derlenmiş programda, `ahdcode run`'da ve REPL'de aynı davranır;
editörler her biri için tamamlama, hover ve imza yardımı gösterir. Bkz.
[v1.7 örnekleri](examples/v1.7/README_TR.md).

## v1.6.0 ile gelenler <a id="v160-ile-gelenler"></a>

v1.6.0 bir **ara (minor)** sürümdür, **Graphics + Turtle**. Tek bir standart
modül ekler; çekirdek dilbilgisini, tip sistemini ve daha önce yayımlanmış her
fonksiyonun davranışını korur.

**Yeni standart modül: [`Graphics`](docs/GRAPHICS_TR.md)** — görsel programlama
ve 2B çizim:

- `Graphics.open(width, height, title, background)` koordinatları Kartezyen
  olan bir Canvas içeren bir pencere açar: `(0, 0)` merkezdir ve `+y` yukarıyı
  gösterir.
- `canvas.line`, `canvas.circle` ve `canvas.rectangle` renk adları ya da
  `#RRGGBB`/`#RRGGBBAA` ile çizer; `fill` isteğe bağlıdır (`String?`) ve bir
  dikdörtgen sol alt köşesiyle verilir.
- `canvas.save("x.png")` ve `canvas.save("x.svg")` çizimi kaydeder;
  `canvas.wait()` pencere kapanana kadar açık tutar, `canvas.close()` kapatır.
- `canvas.turtle()` geometrisi belirlenimci olan, zamana ya da kare hızına
  bağlı olmayan bir Turtle kalemi döndürür (`forward`, `left`, `right`,
  `moveTo`, `penUp`, `penDown`, `setColor`, `setWidth`, `home`, `x`, `y`,
  `heading`).

Canvas ve Turtle çağrıları her zamanki kurala uyar: ya tamamen konumsal ya da
tamamen isimli. Derlenmiş programlar, `ahdcode run` ve REPL tek bir
gerçekleştirmeyi paylaşır. Her Canvas'ı, pencere kitaplığını derleyicinin ve
derlenmiş programların dışında tutan ayrı bir program olan paketli
`ahdgraphics` yardımcısı çizer. Graphics'te sprite, animasyon döngüsü, klavye
ya da fare girdisi, ses, arayüz bileşeni ya da 3B yoktur. Bkz.
[Graphics ve Turtle örnekleri](examples/v1.6/graphics_turtle/README_TR.md).

## v1.5.0 ile gelenler <a id="v150-ile-gelenler"></a>

v1.5.0 bir **ara (minor)** sürümdür, **Terminal**. Tek bir standart modül ekler;
çekirdek dilbilgisi, tip sistemi ve daha önce yayımlanmış her fonksiyon
davranışını korur.

**Yeni standart modül: [`Terminal`](docs/TERMINAL_TR.md)** — `write`, `take` ve
`str`'nin bilerek dışarıda bıraktığı terminale özgü davranışlar:

- `Terminal.emit(parts, separator, ending)` bir `List<String>`'i standart
  çıktıya birleştirerek yazar ve hiçbir şeyi dönüştürmez.
- `Terminal.error(text, ending)` standart hataya yazar, `Terminal.flush()` ise
  tamponlanmış standart çıktıyı hemen yazar.
- `Terminal.isInteractive()`, `Terminal.width()` ve `Terminal.height()` standart
  çıktıyı anlatır; boyutlar `Int?`'dir ve asla tahmin edilmez.
- `Terminal.supportsColor()` ve `Terminal.style(...)` rengi yalnızca etkileşimli
  bir terminalde ekler, `NO_COLOR` ve `TERM=dumb`'a uyar ve çıktı
  yönlendirildiğinde düz metin döndürür.
- `Terminal.pretty(value)` bir List ya da Pair'i okunmak üzere satırlara yayar.

Terminal bir TUI çatısı değildir: imleç denetimi, ham klavye girdisi ya da
ilerleme çıktısı yoktur. Bağımlılık eklemez, derlenmiş programlar ve REPL aynı
çıktıyı üretir ve editörler artık üzerine gelme (hover) ile imza yardımında
null olabilen parametre ve dönüşlerin `?` işaretini gösterir. `Terminal` artık
bir standart modül adıdır: bir programın yanındaki yerel `Terminal.ahd`,
`bring Terminal` ile yüklenen şey değildir. Bkz.
[Terminal tanıtımı](examples/v1.5/terminal_demo/README_TR.md).

## v1.4.0 ile gelenler <a id="v140-ile-gelenler"></a>

v1.4.0 bir **ara (minor)** sürümdür, **Gerçek Zamanlı Web ve Veri**. İki
standart modül, WebSocket sunucu uç noktaları ve `Env.secret` ekler. Çekirdek
dilbilgisi, tip sistemi ve daha önce yayımlanmış her fonksiyon davranışını
korur.

**Yeni standart modül: [`UUID`](docs/UUID_TR.md)** — `UUID.v4()` ve zamana göre
sıralı `UUID.v7()` değiştirilemez `UUIDValue` değerleri üretir. `UUID.parse`
yalnızca kanonik 36 karakterlik biçimi kabul eder ve değerler `equals` ile
`compare` kullanılarak karşılaştırılır. `Identity.id()` değişmedi.

**Yeni standart modül: [`PostgreSQL`](docs/POSTGRESQL_TR.md)** — MySQL
ailesinde `connect`, `execute`, `query` ve `begin`; `$1` yer tutucuları,
`boolean`, tam `numeric`, UTC `timestamptz`, `RETURNING` ve PostgreSQL'in
iptal edilmiş işlem kuralı açıkça tanımlanmış olarak. Sunucuyu yalnızca
`connect` argümanları seçer: `PG*` değişkenlerinin ve `~/.pgpass`'in etkisi
yoktur. pgx gömülüdür; derlemeler çevrimdışı kalır.

**[`HTTP`](docs/WEBSOCKET_TR.md) içinde WebSocket uç noktaları** —
`HTTP.websocket`, `Server.websocket` ve `App.websocket`, rotalarınızla aynı
sunucuda metin mesajlı uç noktalar barındırır. Geri çağrılar HTTP
işleyicileriyle birlikte tek tek çalışır; istemci bağlantıyı açık gördüğünde
`onOpen` dönmüştür; varsayılan köken politikası aynı kökendir; `withAccept`
yükseltmeden önce kimlik doğrular; mesaj boyutu, kuyruk ve bağlantı sayısı
sınırlıdır.

**Genişletilen modül: [`Env`](docs/ENV_TR.md#secret)** — `Env.secret(name)`,
`NAME` değişkenini ya da konteyner platformlarının bağlanan gizli değerler için
kullandığı `NAME_FILE` ile adı verilen dosyayı okur.

`UUID` ve `PostgreSQL` artık standart modül adlarıdır: bir programın yanındaki
yerel `UUID.ahd` ya da `PostgreSQL.ahd`, `bring UUID` veya `bring PostgreSQL`
ile yüklenmez; v1.3.0'daki `QR` ve `Barcode` gibi. Bkz.
[`examples/v0.1/69_uuid.ahd`](examples/v0.1/69_uuid.ahd) ile
[`examples/v0.1/72_websocket_echo.ahd`](examples/v0.1/72_websocket_echo.ahd)
arası ve [gerçek zamanlı yoklama uygulaması](examples/v1.4/realtime_attendance/README_TR.md).

## v1.3.0 ile gelenler <a id="v130-ile-gelenler"></a>

v1.3.0 bir **ara (minor)** sürümdür, **Profesyonel Belgeler ve Makine
Kodları**. İki standart modül ve `Latex` ile `PDF` için profesyonel belge
özellikleri ekler. Çekirdek dilbilgisi, tip sistemi ve daha önce yayımlanmış
her fonksiyon davranışını korur; yeni parametreler isteğe bağlıdır ve sona
eklenir, yeni özellikleri kullanmayan bir belge önceki gibi render edilir.

**Yeni standart modül: [`QR`](docs/QR_TR.md)** — `QR.create(value, level)`,
L, M, Q veya H hata düzeltme seviyesinde değişmez bir `QRCode` oluşturur,
modül matrisini sunar ve keskin PNG ya da vektör SVG dosyaları kaydeder;
`QRError` ile.

**Yeni standart modül: [`Barcode`](docs/BARCODE_TR.md)** — `Barcode.code128`,
`Barcode.ean13` ve `Barcode.upca`, hesaplanan veya doğrulanan kontrol
basamaklarıyla değişmez `BarcodeCode` değerleri oluşturur, çubuk desenini
sunar ve PNG ya da SVG dosyaları kaydeder; `BarcodeError` ile.

**Genişletilen modül: [`Latex`](docs/LATEX_TR.md#profesyonel-belgeler-v130)** —
`qr`, `barcode`, `place`, `header`, `footer`, `pageNumber`, `pageCount`,
`link` ve `bookmark`; `document(paper:, pageSize:, margins:, subject:,
keywords:, creator:)`; SVG dosyaları ile döndürme, opaklık ve kırpma içeren
bir `transform` destekleyen `image`/`figure`. Çevrimdışı paket `fancyhdr` ve
`lastpage`'i ekler.

**Genişletilen modül: [`PDF`](docs/PDF_TR.md)** — `layout`, `header`,
`footer`, `pageNumbers`, `qr`, `barcode`, `link`, `bookmark`, `metadata` ile
SVG dosyaları ve `transform` destekleyen `image`; her String yine kaçışlanır
ve ham TeX yoktur.

SVG dosyaları program içinde vektör çizime dönüşür: asla rasterleştirilmez;
tarayıcı, Inkscape veya başka bir harici dönüştürücü kullanılmaz. QR kodları
ve barkodlar, vendor edilmiş MIT lisanslı bir kodlayıcıyla çevrimdışı
üretilir. Bkz. [`examples/v0.1/62_qr.ahd`](examples/v0.1/62_qr.ahd) ile
[`examples/v0.1/68_svg_assets.ahd`](examples/v0.1/68_svg_assets.ahd) arası.

## v1.2.0 ile gelenler <a id="v120-ile-gelenler"></a>

v1.2.0 bir **ara (minor)** sürümdür, **Zamanlama, Vektör Belgeler ve Karakter
Araçları**. İki standart modül ve `Latex` için vektör grafik ekler. Çekirdek
dilbilgisi, tür sistemi ve daha önce yayımlanmış her fonksiyon davranışını
korur; `Latex.document` yalnızca isteğe bağlı bir son parametre kazanır.

**Yeni standart modül: [`Cron`](docs/CRON_TR.md)** — klasik beş alanlı
zamanlamalarla sınırlı, süreç içi zamanlama: `Cron.scheduler()`,
`Scheduler.add(expression, task)`, `Scheduler.run()`, `Scheduler.stop()`,
`Cron.next(expression, after)` ve `CronError`. İşler, onları zamanlayan
program çalıştığı sürece çalışır. Cron işletim sisteminin crontab'ı, bir
daemon veya bir iş kuyruğu değildir ve sistem zamanlama yapılandırmasını asla
değiştirmez.

**Yeni standart modül: [`Characters`](docs/CHARACTERS_TR.md)** — `Char` türü
olmadan Unicode kod noktası işlemleri: `list`, `count`, `codePoint`,
`fromCodePoint`, `isLetter`, `isDigit`, `isWhitespace`, `isUpper`, `isLower`,
`isAlphaNumeric`, `isPunctuation`, `isSymbol` ve `CharactersError`. Bir
karakter, tek kod noktası taşıyan bir `String`'dir; grafem kümeleri bölütlenmez.

**Genişletilen modül: [`Latex`](docs/LATEX_TR.md#tikz-ile-vektör-grafik-v120)** —
aynı çevrimdışı hat üzerinden TikZ/PGF vektör grafikleri: `Latex.tikz`,
`Latex.overlay`, `Latex.border` ve `Latex.document(..., landscape: true)`.
Çevrimdışı kaynak paketi artık TikZ'i, dokuz TikZ kütüphanesini ve
süslemeleriyle pgfornament'i de taşır; sertifikalar, sayfa kenarlıkları,
filigranlar ve diyagramlar ne ikinci bir PDF kütüphanesi ne de önceden
üretilmiş bir görsel gerektirir. Bkz.
[`examples/v0.1/61_tikz_certificate.ahd`](examples/v0.1/61_tikz_certificate.ahd).

## v1.1.0 ile gelenler <a id="v110-ile-gelenler"></a>

v1.1.0 bir **ara (minor)** sürümdür. Bir yeni standart modül ekler, bir
mevcut modülü genişletir. Çekirdek dilbilgisi, tür sistemi ve daha önce
yayımlanmış her fonksiyon v1.0.0 davranışını korur.

**Yeni standart modül: [`Bits`](docs/BITS_TR.md)** — dilin işaretli 64 bit `Int`
türü üzerinde bit işlemleri. AhdCode dilbilgisinde bit operatörü yoktur
(`and`, `or` ve `not` mantıksal operatörlerdir, `^` üs almadır); bu yüzden
işlemler adlandırılmış çağrılardır:

`bitAnd`, `bitOr`, `bitXor`, `bitNot`, `shiftLeft`, `shiftRight`,
`shiftRightUnsigned`, `rotateLeft`, `rotateRight`, `count`, `leadingZeros`,
`trailingZeros` ve kaydırma/döndürme mesafesi `0..63` dışına çıktığında
yükselen `BitsError` hata türü.

**Genişletilen modül: [`Security`](docs/SECURITY_TR.md)** — parola primitifleri
değişmedi; modül artık özetler (digest), mesaj doğrulama, yaygın kodlamalar,
RS256 imzaları ve doğrulamalı simetrik şifrelemeyi de kapsıyor:

`sha256`, `sha512`, `hmacSHA256`, `hmacVerify`, `base64Encode`, `base64Decode`,
`base64UrlEncode`, `base64UrlDecode`, `hexEncode`, `hexDecode`, `randomHex`,
`rsaSignSHA256`, `rsaVerifySHA256`, `aesEncrypt`, `aesDecrypt`.

Kriptografik bir seçim varsa bir kez yapılır ve ayar olarak dışarı açılmaz:
özet için SHA-2, MAC için HMAC-SHA256, imza için SHA-256 üzerinde
RSASSA-PKCS1-v1_5 (JWT'nin RS256 dediği algoritma) ve şifreleme için
AES-256-GCM.

---

AhdCode v1.0.0 ilk kararlı sürümdür. Aşağıdaki dışarıda bırakmalar bilinçli tasarım kararlarıdır; sonraki bir sürümü bekleyen eksikler değildir.

Dil içinde AhdCode; kasıtlı olarak blok/deyim lambda'larını, keyfi/örtük değişken closure'larını, genel kullanıcı-tanımlı operatör aşırı yüklemesini (on sabit Class Protocol Method dışında), çoklu dönüş değerlerini/tuple'ları, reflection'ı, trait/interface'leri ve çoklu kalıtımı hariç tutar. Araçlarda ise AhdCode harici bir paket yöneticisi veya uzak kayıt defteri yerine derleme zamanı yerel kaynak birleştirmesi ([`require(...)`](docs/REQUIRE_TR.md)) ve paketli çevrimdışı modülleri kullanır. Dil sunucusu referans bulma ve yeniden adlandırma, derleyici tarafından çözümlenen, sınırlı sayıdaki isteğe bağlı çalışma alanı görüntüsüyle çalışır; arka planda sürekli bir indeks başlatmaz. Bkz. [spesifikasyonun desteklenmeyen özellik listesi](AHDCODE_LANGUAGE_SPEC_v0.1_TR.md#40-desteklenmeyen-v01-özellikleri).

## Depo haritası

```text
cmd/ahdcode/         CLI giriş noktası ve komut yönlendirici
cmd/ahdnumeric/      paketli ileri doğrusal-cebir yardımcısı
cmd/ahdplot/         paketli grafik render yardımcısı
cmd/ahdgraphics/     paketli Graphics pencere yardımcısı (kendi Go modülü)
cmd/ahdgui/          paketli GUI pencere yardımcısı (kendi Go modülü)
cmd/ahdplotview/     paketli etkileşimli Plot görüntüleyicisi (kendi Go modülü)
cmd/ahdidentity/     pencere yardımcılarının ortak AhdCode adı ve simgesi
cmd/ahdsqlite/       paketli CGO'suz SQLite yardımcısı
internal/            derleyici ön yüzü, arka yüzü, çalışma zamanı, biçimlendirici, LSP ve REPL
internal/framework/  paketli birinci taraf Web çatısı kaynağı
editors/vscode/      VS Code / Antigravity editör eklentisi
docs/                yetkili referans rehberleri ve eğitimler
examples/v0.1/       derlenmiş çekirdek dil programları
examples/v0.3/       SQLite Not Defteri
examples/v0.4/       Web Not Defteri
examples/v0.5/       çerezler ve bellek içi oturumlar
examples/v0.6/       giden HTTP Client ve JSON API'ler
examples/v0.7/       HTML ayrıştırma, seçiciler ve web kazıma
examples/v0.8/       multipart formlar, dosya yükleme ve yükleme meta verisi
examples/v0.9/       Env yapılandırmalı SMTP metin/HTML postası
examples/v0.12/      MySQL çekiliş örneği (katılım kodu ve kazanan ilanı)
examples/v0.14/      require(...) ve statik varlıklarla çok dosyalı web uygulaması
examples/v0.15/      Matematik Portalı dogfood uygulaması
examples/v0.16/      formlar, doğrulama, CSRF ve flash iş akışı
examples/v2.3/       v2.3 sürüm örnekleri
tools/AhdDataStudio/ birinci taraf yerel MySQL + SQLite geliştirme arayüzü
AHDCODE_LANGUAGE_SPEC_v0.1.md
                     yetkili çekirdek dil sözleşmesi
```

## Geliştirme ve katkılar

AhdCode, Ali Harun Daldallı tarafından tasarlanmış ve spesifikasyonu
oluşturulmuştur. Uygulama, dokümantasyon ve test süreçlerinde OpenAI Codex,
Anthropic Claude ve Google Gemini dahil olmak üzere yoğun yapay zekâ
desteğinden yararlanılmıştır. Araçların katkı biçimi göreve göre değişmekte;
dil tasarımı ve nihai teknik kararlar proje yazarına aittir.

## Lisans

AhdCode, [MIT Lisansı](LICENSE) altında kullanılabilir.
