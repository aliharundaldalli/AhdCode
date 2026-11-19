# HTTP standart modülü

[English](HTTP.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [HTML](HTML_TR.md) · [Öğrenci Rehberi](STUDENT_GUIDE_TR.md#38-http-client)

> **v0.15:** `HTTP` düşük seviyeli modüldür ve burada belgelendiği gibi
> kalır. Birinci taraf [`Web`](WEB_TR.md) çatısı onu bileştirir -- `Web.app`
> `Server`'ı sarar ve `Web`, `Request`, `Response`, `Session`,
> `SessionStore`, `Cookie` ve `UploadedFile`'ı *aynı* türler olarak yeniden
> dışa aktarır. Doğrudan `bring HTTP` kullanmak asla bir geri düşüş ya da
> eski yol değildir; çatının yapılandırma sözleşmesi olmadan sunucuyu
> istediğinizde ona uzanın.

İlk kez öğreniyorsanız URL yapısı, durum kodları, taşıma hataları, JSON POST
ve istemci güvenliğini birlikte gösteren [HTTP/HTTPS atölyesini](PRACTICAL_MODULES_TR.md#7-http-ve-https-istek-yanıt-ve-hata-sınırı)
çalışın; bu sayfayı sunucu ve istemcinin eksiksiz referansı olarak kullanın.

`HTTP`, AhdCode v0.4.0 ile gelen, derleyici tarafından kayıtlı
`builtin:HTTP` modülüdür. Açıkça getirilir ve yanındaki bir `HTTP.ahd`
onu gölgeleyemez:

```ahd
bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring HTTPError
from HTTP bring Cookie
from HTTP bring SessionStore
from HTTP bring Session
from HTTP bring Client
from HTTP bring ClientRequest
from HTTP bring ClientResponse
from HTTP bring WebSocket
from HTTP bring WebSocketEndpoint
```

`HTTP` küçük, tipli yerel bir web sunucusu **ve** giden bir HTTP/HTTPS
istemcisidir. Tam yolları kaydeder, her isteğin bir anlık kopyasını okur ve
bir `Response` döndürürsünüz. v0.5.0 HTTP çerezleri ve bellek içi sunucu
taraflı `SessionStore` ekler. v0.6.0, dış HTTP ve HTTPS API'lerini çağırmak
için `Client`, `ClientRequest` ve `ClientResponse` ekler. v0.9.1, diskteki bir
dosya için ikili-güvenli yanıtlar sağlayan `HTTP.file` ve `HTTP.download`
ekler. Sunucu `Request`/`Response` ile giden `ClientRequest`/`ClientResponse`
ayrı türlerdir. v1.4.0, aynı `Server` üzerinde WebSocket sunucu uç noktaları
ekler; bkz. [WebSocket](WEBSOCKET_TR.md).
v2.1, dosya aktarımının giden yarısını ekler: `Client.download` ve
`Client.sendToFile` bir yanıt gövdesini doğrudan dosyaya yazar,
`ClientRequest.withMultipartField` ve `withMultipartFile` ise bir
`multipart/form-data` gövdesi gönderir.

Böylece multipart iki yönde de desteklenir ve her zaman bayt türüyle değil
dosyalarla: **gelen** dosya yüklemeleri `Request.file` / `Request.files`
ile ulaşır (v0.8.0), **giden** dosya yüklemeleri ise `withMultipartFile`
tarafından diskten akıtılır (v2.1). Düşük seviyeli HTTP her ikisinde de açık
ve tür güvenlidir: middleware, yönlendirici DSL'si, yol parametresi, kimlik
doğrulama çerçevesi veya yapay zeka satıcı modülü yoktur. Uygulama, AhdCode
çalışma zamanının içindeki Go `net/http` paketini kullanır; ayrı bir HTTP,
çerez, oturum, istemci veya yükleme yardımcı süreci yoktur.

## Genel yüzey

```text
HTTP.server(host: String, port: Int, maxBodyBytes: Int := 1048576) -> Server
HTTP.response(status: Int, body: String, contentType: String)      -> Response
HTTP.text(body: String, status: Int := 200)                        -> Response
HTTP.html(body: String, status: Int := 200)                        -> Response
HTTP.redirect(location: String, status: Int := 303)                -> Response
HTTP.file(path: String, contentType: String)                       -> Response
HTTP.download(path: String, contentType: String, fileName: String) -> Response
HTTP.websocket(onMessage: Function(WebSocket, String) -> Nothing)  -> WebSocketEndpoint
HTTP.cookie(name: String, value: String)                           -> Cookie
HTTP.deleteCookie(name: String, path: String := "/")               -> Cookie
HTTP.sessions(
    cookieName: String := "ahd_session"
    maxAgeSeconds: Int := 86400
    secure: Bool := false
    sameSite: String := "Lax"
) -> SessionStore
HTTP.persistentSessions(
    path: String
    cookieName: String := "ahd_session"
    maxAgeSeconds: Int := 86400
    secure: Bool := false
    sameSite: String := "Lax"
) -> SessionStore
HTTP.client(
    timeoutSeconds: Int := 30
    maxResponseBytes: Int := 8388608
    followRedirects: Bool := true
) -> Client
HTTP.clientRequest(method: String, url: String) -> ClientRequest
HTTP.webSocketClient(url: String) -> WebSocketClient
HTTP.contextHandler(
    store: SessionStore
    opener: Function(Request, SessionStore) -> RequestContext
    handler: Function(RequestContext) -> Response
    first: Function(RequestContext) -> Response
    second: Function(RequestContext) -> Response
) -> Function(Request) -> Response

Request.file(name: String)  -> UploadedFile?
Request.files(name: String) -> List<UploadedFile>

UploadedFile.originalName()        -> String
UploadedFile.declaredContentType() -> String?
UploadedFile.detectedContentType() -> String
UploadedFile.size()                -> Int
UploadedFile.save(directory: String) -> String

Server.get(path: String, handler: Function)   -> Nothing
Server.post(path: String, handler: Function)  -> Nothing
Server.route(method: String, path: String, handler: Function) -> Nothing
Server.static(prefix: String, root: String)   -> Nothing
Server.managed(prefix: String, root: String)  -> Nothing
Server.websocket(path: String, endpoint: WebSocketEndpoint) -> Nothing
Server.applyWebLimits()                       -> Nothing
Server.start()                                -> Nothing

Request.method()                   -> String
Request.path()                     -> String
Request.query(name: String)        -> String?
Request.queryAll(name: String)     -> List<String>
Request.header(name: String)       -> String?
Request.headerAll(name: String)    -> List<String>
Request.body()                     -> String
Request.form(name: String)         -> String?
Request.formAll(name: String)      -> List<String>
Request.cookie(name: String)       -> String?
Request.cookieAll(name: String)    -> List<String>

Response.withHeader(name: String, value: String) -> Response
Response.withCookie(cookie: Cookie)              -> Response

Cookie.withPath(path: String)       -> Cookie
Cookie.withHttpOnly(value: Bool)    -> Cookie
Cookie.withSecure(value: Bool)      -> Cookie
Cookie.withSameSite(mode: String)   -> Cookie
Cookie.withMaxAge(seconds: Int)     -> Cookie

SessionStore.open(request: Request) -> Session
SessionStore.commit(session: Session, response: Response) -> Response

Session.get(name: String)                 -> String?
Session.has(name: String)                 -> Bool
Session.set(name: String, value: String)  -> Nothing
Session.remove(name: String)              -> Nothing
Session.clear()                           -> Nothing
Session.rotate()                          -> Nothing
Session.destroy()                         -> Nothing

Client.send(request: ClientRequest) -> ClientResponse
Client.get(url: String)             -> ClientResponse
Client.post(
    url: String
    body: String
    contentType: String := "text/plain; charset=utf-8"
) -> ClientResponse
Client.download(url: String, path: String)              -> ClientFileResponse
Client.sendToFile(request: ClientRequest, path: String) -> ClientFileResponse

ClientRequest.withHeader(name: String, value: String) -> ClientRequest
ClientRequest.addHeader(name: String, value: String)  -> ClientRequest
ClientRequest.withBody(body: String)                  -> ClientRequest
ClientRequest.withMultipartField(name: String, value: String) -> ClientRequest
ClientRequest.withMultipartFile(
    name: String
    path: String
    fileName: String := ""
    contentType: String := "application/octet-stream"
) -> ClientRequest

ClientResponse.status()              -> Int
ClientResponse.body()                -> String
ClientResponse.header(name: String)  -> String?
ClientResponse.headerAll(name: String) -> List<String>
ClientResponse.url()                 -> String

ClientFileResponse.status()              -> Int
ClientFileResponse.header(name: String)  -> String?
ClientFileResponse.headerAll(name: String) -> List<String>
ClientFileResponse.url()                 -> String
ClientFileResponse.size()                -> Int

HTTPError  (Error'dan türer)
```

`Server`, `Request`, `Response`, `Cookie`, `SessionStore`, `Session`,
`Client`, `ClientRequest`, `ClientResponse` ve `ClientFileResponse` opak
yerleşik Sınıflardır. Atlanan `HTTP.sessions` argümanları `ahd_session`,
`86400`, `false` ve `"Lax"`'tır. Atlanan `HTTP.client` argümanları `30`,
`8388608` ve `true`'dur. Atlanan `withMultipartFile` argümanları `""`
(yolun kendi taban adı) ve `application/octet-stream`'dir.

`HTTP.contextHandler`, `Web.routes` tarafından kullanılan v0.17 kayıt
adaptörüdür. Oturumu kendisi sonlandırmaz ve genel bir middleware zinciri
değildir. Sıradan uygulamalar bağlam yollarını Web üzerinden kaydeder.

## İşleyici imzası

Her yol işleyicisi sıradan bir Fonksiyondur:

```text
(request: Request) -> Response
```

Derleyici bu şekli statik olarak denetler. Eksik parametre, `String` dönüşü
veya `Int` parametresi derleme zamanı hatasıdır.

```ahd
home: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

app: Server := HTTP.server("127.0.0.1", 8080)
app.get("/", home)
app.start()
```

`app.get` ve `app.post` `"GET"` ve `"POST"` kaydeder. `app.route` yöntemi
verildiği gibi kullanır: `"get"`, `"GET"` değildir.

## Sunucu ömrü

`HTTP.server` host, port ve gövde sınırını kaydeder. Henüz bağlanmaz.
`start()` bağlanır, sonra **bloklar**. Tarayıcı açmaz ve bir başlangıç
afişi yazdırmaz. Program durdurulana kadar terminal meşgul kalır.

`host` boş olamaz. `port` `1..65535` aralığında olmalıdır (port `0` geçersizdir).
Bağlanma hataları `HTTPError` fırlatır.

Bir istek işleyicisi aynı sunucuda `start()` çağıramaz ve yol kaydedemez.
Aynı kural bir WebSocket `accept`, `onOpen`, `onMessage` veya `onClose`
geri çağrısı için de geçerlidir. Çağrı `HTTPError` fırlatır; çalışan sunucu
değişmez. İşleyici başka bir sunucuda `start()` çağırabilir. Bu çağrı
yalnızca o isteği, iç sunucu durana kadar bekletir. Bir sunucuyu durdurmak
diğerini durdurmaz. Başlamış bir sunucuya yol kaydetmek de `HTTPError`
fırlatır ve o sunucunun sürmekte olan geri çağrısının bitmesini beklemez.

`HTTP.server` ile verilen host, sunucunun kimliğidir. Adlar büyük-küçük
harften bağımsız eşleşir. Bir IP yalnızca o IP ile eşleşir. `localhost`,
`127.0.0.1`, `::1`, `0.0.0.0` ve `::` farklı sunuculardır. `127.0.0.1`
durdurmak, `localhost` veya joker adresle kurulan sunucuyu durdurmaz.

`127.0.0.1` yalnızca bu makinedir. Aynı bilgisayarın tarayıcısında
`http://127.0.0.1:8080/` açın. `0.0.0.0` bağlamak portu ağdaki diğer
makinelere açar; bunu gelişigüzel yapmayın.

İç zaman aşımları (genel API değildir): ReadHeader 10s, Read 15s, Write 15s,
Idle 60s. Erişim günlüğü yoktur.

## Yönlendirme

Yollar tam eşleşmedir. Yol `/` ile başlamalı, `?` veya `#` içermemelidir.
`/notes` ile `/notes/` farklıdır. Sorgu dizesi yolun parçası değildir.
Yol `/*` ile bitiyorsa tam olarak bir ek segment eşleşir (`/question/*`,
`/question/2` ile eşleşir; `/question` veya `/question/2/extra` ile değil).
Aynı `method + path` tekrarı `HTTPError` fırlatır. `start()` sonrası yollar
değişmez.

Bilinmeyen yollar **404** döner. Başka bir yöntem için var olan bir yol
**405** ve `Allow` başlığı döner.

`server.notFound(handler)` ve `server.internalError(handler)`, eşleşmeyen
yolun ve başarısız bir işleyicinin düz metnini değiştirir. İşleyici
`Function(Request) -> Response` biçimindedir. Yazılan durum, işleyici başka
bir durum seçse bile 404 veya 500'tür. İşleyicinin `Location` başlığı
düşürülür; böylece zorlanmış durum bir yönlendirme olamaz. Dosya yanıtı
kullanılmaz. Bu işleyici de başarısız olursa istemci düz `Not Found` veya
`Internal Server Error` metnini alır ve diğer hata sayfası çağrılmaz. Sayfa,
başarısızlığın metnini almaz. `static` veya `managed` tarafından zaten
yanıtlanmış bir önekteki eksik dosya, mevcut düz 404 olarak kalır. Bkz.
[Templates](TEMPLATES_TR.md).

## Statik dosyalar

`server.static(prefix, root)` (v0.14), yolu `prefix` ile başlayan her isteği
yerel `root` dizini altındaki bir dosyaya eşler:
`server.static("/assets", "public")`, `/assets/app.css` isteğini
`public/app.css`'ten sunar. Bu, uygulamanın kendi yerel dosyaları -- CSS,
JavaScript, SVG, görseller, fontlar -- için birinci taraf, düşük seviyeli
bir ilkeldir; genel bir dosya tarayıcısı ya da yönlendirici değildir: dizin
listelemesi, `index.html` kuralı ya da bu tek amaç dışında joker/önek
mekanizması yoktur.

Tam bir rota her zaman statik bir önek karşısında kazanır: her istekte önce
`routes` denetlenir, `static()` yalnızca eşleşme olmadığında devreye girer;
böylece `server.get("/assets/special.txt", handler)`, aksi halde statik
olan bir dizin altındaki tek bir dosyayı hiçbir belirsizlik olmadan
serbestçe geçersiz kılabilir. `static()` yalnızca `GET`/`HEAD` sunar; statik
bir önek altındaki bir yolda başka yöntemler normal 404'e düşer.

Her istek yol parçası, dosya sistemine ulaşmadan önce denetlenir: boş, `.`,
`..` ve herhangi bir gizli dosya/dizin parçası (`.env`, `.git/config`,
`.DS_Store`, yalnızca son bileşende değil, yolun herhangi bir yerinde) doğrudan
reddedilir -- bu, yüzde-kodlanmış geçiş denemelerini de (`%2e%2e`) etkisiz
kılar, çünkü bu denetim çalıştığında yol zaten çözülmüştür. Ortaya çıkan aday
yol yine de `root` altında kapsanma açısından kanonikleştirilip denetlenir
ve bir sembolik bağ olduğu ortaya çıkarsa, çözüldükten sonra tekrar
denetlenir -- hedefi `root`'un dışına kaçan bir sembolik bağ asla sunulmaz,
tıpkı `require(...)`'in kendi kapsanma modeli gibi; `root`'un içine geri
dönen biri normal şekilde sunulur. Bir dizine eşlenen bir istek reddedilir
(404), asla listelenmez.

Dosyalar, `HTTP.file`/`HTTP.download` ile aynı ikili-güvenli
`http.ServeContent` altyapısından akar, böylece Range ve koşullu istek
başlıkları çalışır ve hiçbir şey bir AhdCode `String`'inden geçmez.
`Content-Type`, dosyanın uzantısından Go'nun standart MIME kayıt defteri
üzerinden ayarlanır; tanınmayan bir uzantı açık bir `Content-Type` almaz.

`static()`, rotalarla aynı kayıt kurallarını izler: `start()` sonrası
çağrılamaz, `prefix` `/` ile başlamalıdır, `root` zaten var olmalı ve bir
dizin olmalıdır, ve iki kayıtlı önek çakışamaz.

Statik bir dosyayı düzenlemek `ahdcode dev`'in yeniden derlemesini asla
tetiklemez -- statik sunum her istekte doğrudan diskten okur. Tarayıcı
yenilemesi başarılı bir kaynak yeniden derlemesini izler, bu yüzden statik
dosya düzenlemesi hâlâ elle yenileme ister. `HTTP.file`, `HTTP.download`
ve `server.static` yanıtları yeniden yazılmaz. Yönlendirme, 1xx durumu,
sıkıştırılmış gövde, ek veya
[CLI](CLI_TR.md#dev-izle-yeniden-derle-yeniden-başlat) listesinin dışındaki
bir karakter kümesi olduğu gibi bırakılır. `</body>` içermeyen bir HTML
parçası betiği gövdenin sonuna alır. Yenileme yolunu yalnızca anan bir
sayfa yine uygundur. Eklenen etiket aynı kökenden harici bir betiktir;
`script-src 'self'` onu çalıştırabilir. `HEAD`, gövde uzunluğu biliniyorsa
`Content-Length` taşır.

## Yönetilen dosyalar

`server.managed(prefix, root)` (v0.20) `static()` ile aynı kapsama, dizin
gezinme ve gizli dosya kurallarını kullanır, ancak yalnızca birleşik
belgenin `HTML.asset` / `Web.Assets` ile kaydettiği dosyaları sunar. `root`
içinde duran bir dosya tek başına tarayıcıdan okunamaz. Dizin listesi yine
yoktur.

`applyWebLimits()` [Web çalışma zamanı sınırlarını](ENV_TR.md) kurar.
`Web.app` bunu çağırır. Ham `HTTP.server`, bu çağrılana kadar tarihsel
1MiB gövde ve 15s/15s/60s zaman aşımlarını korur.

## İstek kopyası

Her işleyici, canlı bir Go isteği değil, değişmez bir kopya alır. `path()`
sorgu olmadan URL yoludur. `query(name)` ilk değerdir, yoksa `null`.
`queryAll(name)` tüm değerlerdir, yoksa boş liste. Yinelenen anahtarlar sıra
korunarak saklanır. Geçerli UTF-8 yüzde kod çözümü değişmez (`Ay%C5%9Fe`,
`%20`, `+`, emoji).

Sorgudaki bozuk yüzde kodlaması (`%`, `%2`, `%ZZ`) ve yüzde çözülmüş geçersiz
UTF-8 (`%80`, `%C0%80`) **400** döner; işleyici çalışmaz. U+FFFD yerine koyma
ve anahtarları sessizce düşürme yoktur.

`header` / `headerAll` büyük/küçük harfe duyarsızdır. `cookie(name)` o addaki
ilk Cookie değeri veya `null`'dır. `cookieAll` istek sırasındaki tüm
eşleşmelerdir, yoksa `[]`. Çerez adları büyük/küçük harfe duyarlıdır.

`body()` UTF-8 gövdedir;
geçersiz UTF-8 `HTTPError` fırlatır (sessiz değiştirme yoktur).

Formlar `Content-Type` `application/x-www-form-urlencoded` veya
`multipart/form-data` iken ayrıştırılır. `form` / `formAll` sorgu erişicileri
gibi davranır ve multipart **metin** alanları da aynı API'den gelir --
multipart'a özel ikinci bir metin erişicisi yoktur. Aynı katı yüzde kod
çözümü urlencoded form gövdesine uygulanır: bozuk veya UTF-8 olmayan form
verisi işleyiciden **önce** **400** döner. Multipart **dosya** parçaları
`file` / `files` ile okunur (aşağıya bakın). Gövde, işleyiciden **önce**
`maxBodyBytes` ile sınırlanır; aşım **413** döner.

## Dosya yüklemeleri

Bir `multipart/form-data` isteği metin alanlarını ve dosya parçalarını
birlikte taşır. Metin alanları `form`/`formAll` ile, dosya parçaları
`file`/`files` ile okunur:

```ahd
handle: Function := (request: Request) -> Response {
    title: Local String? := request.form("title")
    paper: Local UploadedFile? := request.file("paper")
    if paper == null {
        return HTTP.text("paper is required", 400)
    }
    if paper.detectedContentType() != "application/pdf" {
        return HTTP.text("rejected", 415)
    }
    storedPath: Local String := paper.save("uploads/papers")
    return HTTP.text("saved " + storedPath)
}
```

`file(name)` o alandaki ilk yüklenen dosyadır; alan dosya taşımıyorsa
`null`'dır. `files(name)` o alandaki tüm dosyaları istek sırasıyla verir,
yoksa `[]` -- yani `<input type="file" name="papers" multiple>` yinelenenleri
asla sessizce atmaz.

`UploadedFile` opak ve salt okunurdur. `bytes()`, `raw()`, `stream()`,
`tempPath()`, dosya tanıtıcısı veya işaretçi yoktur: yüklenen bir PDF bir
`String` değildir ve AhdCode ikili içeriği metinmiş gibi göstermez. Baytlar
dosya sistemine yalnızca `save` ile ulaşır.

### Adı yükleyen belirler, yolu siz belirlersiniz

`originalName()` **yalnızca görüntüleme meta verisidir**. Tarayıcının verdiği
dosya adının güvenli bir temel ada indirgenmiş halidir: platformdan bağımsız
olarak hem `/` hem `\` ayırıcı sayılır, `C:` sürücü öneki atılır, `.`/`..`
ve boş adlar `file`'a dönüşür ve NUL baytı yapısal olarak geçersiz sayılıp
reddedilir. Bu yüzden `../../evil.pdf` dışarıya `evil.pdf` olarak yansır.

Ondan asla yol kurmayın:

```ahd
storedPath := "uploads/" + paper.originalName()   // bunu yapmayın
storedPath := paper.save("uploads/papers")        // bunu yapın
```

`save(directory)` gerekirse dizini oluşturur ve yüklemeyi
`uploads/papers/8e8f30c65c4d4d23...` gibi **kriptografik rastgele** bir temel
adla, mevcut bir dosyayı asla ezemeyecek şekilde dışlamalı (exclusive) olarak
yazar. Gerçek kayıt yolunu döndürür. Aynı orijinal ada sahip iki yükleme her
zaman farklı kayıt yolları alır. Üretilen temel ad ayırıcı içermediği için
kaydedilen dosya her zaman uygulamanın belirttiği dizinin doğrudan çocuğudur
-- yüklenen bir dosya adı üst dizine ulaşamaz.

Dizin argümanı yükleyenin değil, uygulamanın kararıdır; AhdCode genel olarak
dosya sistemi erişimini kum havuzuna almaz.

Bir yükleme **bir kez** kalıcılaştırılır. Aynı `UploadedFile` üzerinde ikinci
kez `save` çağırmak, sessizce kopya oluşturmak yerine `HTTPError` fırlatır.
Meta veri metotları kayıttan sonra da çalışır.

### Bildirilen tür ve algılanan tür

```text
declaredContentType()  istemcinin iddiası      (buna asla güvenmeyin)
detectedContentType()  baytların görünümü      (kararı bununla verin)
```

`declaredContentType()` parçanın kendi `Content-Type` başlığıdır; yalın medya
türüne normalleştirilir (`text/plain; charset=utf-8` → `text/plain`) veya
parça tür bildirmediyse `null`'dır. `detectedContentType()` baştaki baytları
Go'nun `net/http.DetectContentType` işleviyle inceler ve aynı şekilde
normalleştirir.

Dosya adı ve uzantısı algılamayı asla etkilemez. `application/pdf` iddia eden,
`malware.pdf` adlı bir metin dosyası şunu bildirir:

```text
originalName()         malware.pdf
declaredContentType()  application/pdf
detectedContentType()  text/plain
```

böylece uygulama uyuşmazlığı reddedebilir. Sıfır baytlık bir yüklemenin
benzeyeceği içerik yoktur; belgelenmiş `application/octet-stream` geri
dönüşünü ve `0` `size()` değerini bildirir, asla geçerli bir PDF sanılmaz.

Algılama kabaca *"bu baytlar hangi içerik ailesine benziyor?"* sorusunu
yanıtlar. Kötü amaçlı yazılım taraması, virüs tespiti veya biçim doğrulaması
**değildir**: bir PDF'in açılmasının güvenli olduğunu, bir görüntünün
çözüleceğini veya bir belgenin makro taşımadığını söylemez. Antivirüs
entegrasyonu yoktur.

### Sınırlar, ömür ve temizlik

Sunucunun `maxBodyBytes` değeri multipart dahil **tüm** istek gövdesini
sınırlamaya devam eder ve herhangi bir işleyici veya kalıcılaştırma
çalışmadan **önce** **413** döndürür; yüklemelerin ayrı, sınırsız bir yolu
yoktur. Yerleşik dosya başına sınır yoktur: `size()` değerini kontrol edip
kendi politikanızı uygulayın. PDF bekleyen bir uygulama sunucu sınırını
açıkça yükseltmelidir; örneğin
`HTTP.server("127.0.0.1", 8080, 26214400)`.

Her yükleme, kendi isteğinin ömrü boyunca özel bir geçici dosyayla
desteklenir. İşleyici bittiğinde -- normal yanıt verse de, yüklemeyi
reddetse de, hata fırlatsa da -- kaydetmediği her yükleme silinir ve kaydı
düşürülür. Kaydedilen dosyalar bu ömrün dışına çıkmıştır ve yaşamaya devam
eder. İstek bittikten sonra kaydetmek, silinmiş geçici dosyayı diriltmek
yerine `HTTPError` fırlatır.

Bozuk multipart söz dizimi (eksik veya geçersiz sınır, kesilmiş parça, bozuk
parça başlıkları) işleyici çalışmadan **önce** **400** döndürür; tıpkı başka
herhangi bir bozuk istek gibi. Asla panic üretmez ve asla 500'e dönüşmez.

`body()` mevcut sözleşmesini korur: UTF-8 istek gövdesini döndürür ve geçerli
UTF-8 olmayan bir gövde için `HTTPError` fırlatır. Bu nedenle ikili bir
multipart gövdesi `body()` ile okunamaz -- bu bilinçlidir. Yüklemeler için
`file`/`files`, multipart metin alanları için `form`/`formAll` kullanın.

v0.8.0'da giden multipart yoktu. v2.1 bunu ekler: bkz.
[Giden multipart ve dosya yükleme](#giden-multipart-ve-dosya-yükleme).

## Yanıt

`HTTP.text` `text/plain; charset=utf-8` ayarlar. `HTTP.html`
`text/html; charset=utf-8` ayarlar ve gövdeyi **kaçırmaz**: güvenilir statik
işaretleme (`r"""..."""`) veya `HTML.document` / `HTML.render` çıktısı için
kullanın. Durum kodları `100..599` olmalıdır. `HTTP.redirect` yalnızca
`301`, `302`, `303`, `307` veya `308` kabul eder. `withHeader` yeni bir
`Response` döndürür; CR/LF yasaktır. `Set-Cookie` `withCookie` ile eklenir;
`withHeader` birden fazla çerezi tek başlıkta ezmez.

## İkili dosya yanıtları

`HTTP.text`, `HTTP.html` ve `HTTP.response` bir `String` gövde taşır. v0.9.1
ile eklenen `HTTP.file` ve `HTTP.download` ise diskteki bir dosyanın tam
baytlarını, hiçbir zaman UTF-8 olarak çözmeden veya bir AhdCode `String`ine
dönüştürmeden sunar:

```text
HTTP.file(path: String, contentType: String)                       -> Response
HTTP.download(path: String, contentType: String, fileName: String) -> Response
```

`HTTP.file` dosyayı satır içi sunar (zorunlu bir disposition yoktur).
`HTTP.download` `fileName` altında sunulan `Content-Disposition: attachment`
ekler. İkisi de sıradan bir `Response` döndürür: `withHeader` ve `withCookie`
tıpkı metin yanıtlarında olduğu gibi çalışır ve sonuç aynı şekilde
değişmezdir.

`path`, uygulamanızın zaten güvendiği bir **depolama** yoludur -- genellikle
`UploadedFile.save`'in döndürdüğü değer. Bu bir statik dosya kökü değildir;
istekten gelen dizeler doğrudan buraya asla ulaşmamalıdır.

`contentType` sizin açık beyanınızdır ve `Content-Type` olarak olduğu gibi
gönderilir; yoldan, dosya baytlarından veya istemcinin verdiği bir addan asla
koklanmaz (sniff edilmez). Geçerli, başlık için güvenli bir medya türü
olmalıdır: CR/LF yasaktır, boş olamaz.

`HTTP.download`'daki `fileName` yalnızca sunum amaçlıdır; dosya sisteminde
aramayı asla etkilemez ve `path`'ten bağımsızdır. CR/LF içeremez. ASCII
olmayan adlar (Türkçe dahil) güvenli bir ASCII yedeğiyle birlikte
standartlara uygun bir `filename*` (RFC 5987) ile kodlanır.

Dosya yalnızca yanıt gönderilirken açılır, tüm dosyayı belleğe almadan
istemciye akıtılır ve -- istemci bağlantıyı yarıda kesse bile -- her zaman
kapatılır. Eksik bir yol `404`; bir dizin veya açılamayan bir yol denetimli
bir `500`'dür; hiçbiri panic üretmez.

`HTTP.file`/`HTTP.download` bir statik dosya kökü, URL-yol eşlemesi, dizin
listeleme veya herhangi bir web-root soyutlaması eklemez: yalnızca
uygulamanın verdiği tek yolu sunar.

## Çerezler

`HTTP.cookie(name, value)` değişmez bir `Cookie` üretir. Oluşturucular yeni
bir Cookie döndürür; asıl değer değişmez.

Varsayılanlar: Path `/`, HttpOnly `false`, Secure `false`, SameSite `"Lax"`,
Max-Age yok. SameSite yalnızca `"Lax"`, `"Strict"`, `"None"` kabul eder.
`"lax"` sessizce düzeltilmez. `"None"` için `Secure=true` gerekir.

Çerez değerleri cookie-octet'tir (ASCII; `"`, `,`, `;`, `\` yok). CR/LF,
denetim karakterleri ve rastgele Unicode `HTTPError` fırlatır; sessizce
silinmez veya kodlanmaz. Oturum kimlikleri bu sözleşmeyi sağlar.

`HTTP.deleteCookie(name)` `Max-Age=0` içeren bir silme çerezi döndürür.
`withCookie` ile gönderin.

## Oturumlar

`HTTP.sessions` bağımsız, **bellek içi** bir `SessionStore` oluşturur.
Değerler süreç bitince kaybolur; bu v0.5.0 sözleşmesidir, hata değildir.

`HTTP.persistentSessions(path, ...)` aynı `open` / `commit` / `rotate` /
`destroy` işlemlerini `path` SQLite dosyasında saklar. Başarıyla commit
edilmiş değerler ve iptaller süreç kapansa da kalır; aynı dosyayı ve aynı
çerez adını açan sonraki süreç onları görür. `HTTP.sessions` veritabanı
oluşturmaz ve yeniden başlatmada veri tutmaz.

```ahd
sessions: SessionStore := HTTP.persistentSessions("data/sessions.db", "ahd_session", 86400, false, "Lax")
```

`path` bir dosya yoludur; URI ya da `":memory:"` değildir. Üst dizin önceden
var olmalıdır. Yol boşluk, Windows sürücü harfi ve Unicode içerebilir. Göreli
yol store oluşturulurken bir kez çözülür. Yerel program süreç çalışma dizinini
kullanır. Değerlendirici, `SQLite.open` ve `File` gibi oturum dizinini kullanır.
Mutlak yol, Windows sürücü harfi dahil, olduğu gibi kullanılır. Farklı çerez
adları aynı dosyayı paylaşabilir; satırları ayrıdır. Aynı çerez adı ve aynı
dosya, yeniden başlayan sürecin eski oturumları bulduğu addır.

Şema sürümü `1`'dir. Sürüm `1` değilse `HTTPError` fırlatılır ve dosyaya
otomatik göç uygulanmaz. Günlük kipi WAL'dir. `open`, `commit`, `rotate` ve
`destroy` her biri `BEGIN IMMEDIATE` alır. Bu bir yazma kilididir: WAL,
`open`'ın süren bir yazıcının önünden okumasını sağlamaz. Kilit 5 saniyeden
uzun sürerse çağrı `HTTPError` fırlatır. Depolama hatası boş ya da kimliği
doğrulanmış bir oturuma çevrilmez. Bir dosya süreç içinde tek bağlantı kullanır.
Bağlantı, bir HTTP sunucusu gerçekten çalışırken, istek içindeki bir işleyicisi
henüz dönmemişken ya da değerlendirici programı yürütürken açık kalır.
Oluşturulup hiç başlatılmamış bir sunucu nesnesi dosyayı açık tutmaz. Son
çalışan sunucu durduğunda, işleyicileri döndüğünde ve süren bir değerlendirici
yürütmesi kalmadığında bağlantı kapanır; dosya yerinde kalır. Zarif kapanma
süresi dolduktan sonra hâlâ çalışan bir işleyici, dönene kadar dosyayı açık
tutar. Sonraki oturum işlemi aynı dosyayı yeniden açar. Kapatmak commit edilmiş
satırları silmez.

Bu bir kimlik doğrulama çerçevesi değildir. SameSite, uygulama düzeyinde CSRF
kontrolünün yerine geçmez.

Çerez yalnızca opak rastgele bir kimlik taşır (`crypto/rand`, 256 bit,
padding'siz base64url). Veritabanı çerezin kendisini değil, SHA-256 özetini
saklar. Değerler yalnızca String'dir. Anahtar en çok 128 bayt, bir değer en
çok 65536 bayt, kayıtlı harita en çok 262144 bayt JSON olabilir. Kimlikler
olağan tanı iletilerine yazılmaz.

Varsayılan oturum çerezi: Path `/`, HttpOnly her zaman `true`, Max-Age =
`maxAgeSeconds`. HttpOnly kapatılamaz. Düz HTTP geliştirme `secure: false`
ister; tarayıcı `http://` üzerinde Secure çerezi geri göndermez. HTTPS
dağıtımı `secure: true` geçirir.

Süre sunucuda ölçülür ve yalnızca başarılı `commit` ile yenilenir; `open`
süreyi uzatmaz. Süresi dolmuş ya da iptal edilmiş kimlik, yeniden başlatmadan
sonra da kabul edilmez. Böyle bir kimliği `commit` etmek satırı yeniden
oluşturmaz, `HTTPError` fırlatır. `commit` haritanın tamamını değiştirir;
anahtarları başka bir istekle birleştirmez. Aynı `Server` üzerindeki
işleyiciler seri çalışır. İki süreç araya girebilir; geçerli kimliğin sonraki
başarılı `commit`'i haritanın tümünü yazar. Bozuk JSON, oturum açılmış sayılmaz.

İşleyici içinde modül düzeyindeki store `Global` ile bildirilir:

```ahd
sessions: Global SessionStore
session: Local Session := sessions.open(request)
```

Bilinmeyen veya süresi dolmuş çerez yeni anonim oturum olur; saldırgan
kimliği sunucu kimliği olarak kullanılmaz. `set`/`remove`/`clear`/`rotate`
olmadan `open` kalıcı durum veya `Set-Cookie` üretmez. `Session.set` başlık
yazmaz; `commit` yeni bir `Response` döndürür.

`set`, `remove` ve `clear` yalnızca isteğin kopyasını değiştirir. `commit`
olmadan süreç kapanırsa bu değişiklikler kaybolur; önceki commit durur.
`commit` haritanın tamamını yazar. Anahtar birleştirme ve karşılaştır-değiştir
yoktur. Süresi dolmuş, iptal edilmiş veya döndürülmüş bir kimliğin `commit`'i
`HTTPError` fırlatır ve satır yazmaz.

Kayıtlı bir oturumda `rotate()`, aynı transaction içinde satırın hâlâ var
olduğunu, süresinin dolmadığını ve iptal edilmediğini doğrular. Ancak ondan
sonra eski kimliği iptal eder ve yeni kimlik döndürür. Bellek içi store aynı
kuralı kullanır. Eksik, süresi dolmuş, iptal edilmiş veya daha önce döndürülmüş
bir kimlik `HTTPError` fırlatır ve yeni kimliği doğrulanmış oturum olmaz.
İptal commit edildikten sonra, yeni kimlik `commit` edilmeden süreç durursa
eski kimlik geçersiz kalır ve yeni kimlik hiç yazılmaz. İstemci yeniden
oturum açmalıdır. Bu kasıtlı kapalı-başarısız sonuçtur.

`destroy()` kayıtlı kimliği hemen iptal eder. `commit` silme çerezini aynı ad,
`Path=/`, HttpOnly, Secure ve SameSite ile, `Max-Age=0` olarak gönderir.
Oturum çerezinin Domain alanı yoktur; silme çerezinin de yoktur. `__Host-` ve
`__Secure-` adları `secure: true` ister. Eski kimlik, geç kalmış `rotate` veya
`commit` ile de kullanılamaz. Destroy sonrası `get` `null`, `has` `false`;
mutasyon `HTTPError` fırlatır.

## Giden istemci

`HTTP.client` yeniden kullanılabilir bir `Client` üretir. `close()` yoktur.
`timeoutSeconds` toplam istek zaman aşımıdır ve `1..9223372036` aralığında
olmalıdır. `maxResponseBytes` tamponlanan gövde sınırıdır ve
`1..9223372036854775806` aralığında olmalıdır (varsayılan 8 MiB). Aralık dışı
değerler `HTTPError` fırlatır; sıkıştırılmaz. Tam `N` bayt başarılıdır;
`N + 1` `HTTPError` fırlatır. Otomatik yeniden deneme yoktur.

`HTTP.clientRequest` yöntemi sessizce büyültmez. URL mutlak `http` veya
`https` olmalı, konak boş olmamalıdır. Parça, userinfo, `file:` / `ftp:` /
`data:` / `javascript:` ve bozuk URL'ler `HTTPError` fırlatır.

`withHeader` o başlık adını büyük/küçük harf duyarsız değiştirir.
`addHeader` ekler. `withBody` String gövdeyi değiştirir. `Content-Length` ve
`Host` reddedilir. Gizli başlık değerleri hata iletilerine yazılmaz.

`400`/`401`/`403`/`404`/`429`/`500`/`503` yine `ClientResponse` döner.
`HTTPError` geçersiz sözleşme, DNS, bağlantı, TLS, zaman aşımı, yönlendirme
politikası, aşırı büyük gövde ve geçersiz UTF-8 içindir. Geçersiz UTF-8
U+FFFD ile değiştirilmez. HTTPS sistem köklerini doğrular; güvensiz TLS seçeneği
yoktur. `http://` localhost için desteklenir.

`followRedirects = false` ilk 3xx yanıtını döndürür. `true` en fazla 10
yönlendirme izler. **HTTPS → HTTP yönlendirmesi reddedilir.** Konak veya
port değişince `Authorization` ve `Cookie` iletilmez.

### İkili-güvenli indirmeler

`ClientResponse.body()` bir `String`'dir, yani metin taşır. PNG, ZIP veya
PDF metin değildir; rastgele baytları bir `String`'e zorlamak onları ya
bozar ya da hata fırlatır. v2.1 bunun yerine dosya odaklı ikinci bir yanıt
yolu ekler:

```text
Client.download(url: String, path: String)              -> ClientFileResponse
Client.sendToFile(request: ClientRequest, path: String) -> ClientFileResponse
```

`download(url, path)`, gövdesi `path` dosyasına yazılan bir GET'tir.
`sendToFile(request, path)` aynısını herhangi bir `ClientRequest` için
yapar; böylece bir indirme kendi başlıklarını, yöntemini veya gövdesini
taşıyabilir.

İkisi de opak bir `ClientFileResponse` döndürür ve bu türün bilinçli olarak
**`body()`'si yoktur**: yük, adını verdiğiniz dosyadadır.

```ahd
client: Client := HTTP.client(timeoutSeconds: 30)
answer: ClientFileResponse := client.download("https://example.com/report.pdf", "report.pdf")
write(str(answer.status()))
write(str(answer.size()) + " bayt yazıldı")
```

`size()`, hedefe gerçekten yazılan bayt sayısıdır; bir `Content-Length`
iddiası değildir. `status()`, `header`, `headerAll` ve `url`, bir
`ClientResponse` üzerindeki davranışlarının aynısını gösterir.

Gövde akıtılır: bağlantıdan dosyaya parça parça gider ve hiçbir zaman
bütünüyle bellekte tutulmaz.

**Durum kodu yine bir yanıttır, başarısızlık değil.** `download` modülün
mevcut felsefesini sürdürür: `404` ve `500` geçerli HTTP yanıtlarıdır, o
durumla bir `ClientFileResponse` döner ve sunucunun gönderdiği gövde —
genellikle bir hata sayfası — dosyaya yazılır. Dosyaya güvenmeden önce
`status()` değerine bakın. Yalnızca taşıma, TLS, sınır veya dosya sistemi
hataları `HTTPError` fırlatır.

**Başarısız bir indirme hedefe asla zarar vermez.** Baytlar önce hedefin
yanındaki geçici bir dosyaya gider; hedef ancak aktarım tamamlandığında
değiştirilir. Kesilen bir istek, `maxResponseBytes` sınırını aşan bir yanıt
veya bir dosya sistemi hatası `HTTPError` fırlatır, geçici dosyayı siler ve
o yoldaki mevcut dosyayı olduğu gibi bırakır.

`maxResponseBytes`, akıtılan bir yanıtı tamponlanan bir yanıtı sınırladığı
gibi sınırlar: akıtılan, sınırsız demek değildir. Üst dizin önceden var
olmalıdır — AhdCode, programın istemediği bir dizini oluşturmaz — ve
kendisi dizin olan bir hedef `HTTPError` fırlatır. Göreli bir yol,
AhdCode'daki her yol gibi süreç çalışma dizinine göre çözülür.

### Giden multipart ve dosya yükleme

Bir `ClientRequest`, multipart parçaları taşıdığında `multipart/form-data`
gövdesi gönderir:

```text
ClientRequest.withMultipartField(name: String, value: String) -> ClientRequest
ClientRequest.withMultipartFile(
    name: String
    path: String
    fileName: String := ""
    contentType: String := "application/octet-stream"
) -> ClientRequest
```

İkisi de **yeni** bir `ClientRequest` döndürür; her `with*` gibi özgün
istek değişmez. Parçalar eklendikleri sırayla gönderilir ve bir alan adı,
bir HTML formunun aynı onay kutusu adını yinelemesi gibi yinelenebilir.

```ahd
upload: ClientRequest := HTTP.clientRequest("POST", "https://example.com/upload")
withTitle: ClientRequest := upload.withMultipartField("title", "Ölçüm raporu")
withFile: ClientRequest := withTitle.withMultipartFile("file", "report.pdf", "", "application/pdf")
answer: ClientResponse := client.send(withFile)
```

`withMultipartFile` içinde `path`, okunacak yerel dosyadır; `fileName` ise
**yalnızca sunum verisidir** — sunucuya bildirilen ad. Boş bir `fileName`,
yolun kendi taban adını kullanır. Her iki durumda da ad düz bir taban adına
indirgenir; böylece `../../etc/passwd` gibi bir ad `passwd` olarak gider ve
asla bir yol gibi davranamaz. `contentType` geçerli bir ortam türü olmalıdır.

Dosyanın baytları hiçbir zaman AhdCode `String`'ine dönüşmez. Bağlantı
boşaldıkça diskten doğrudan akıtılırlar; böylece çalışma zamanının tuttuğu
her tampondan çok daha büyük bir dosya yüklenmeden gönderilir.

**Multipart bir isteğin gövdesi AhdCode'a aittir.** Bir istek ya `withBody`
ya da multipart parçaları kullanır, ikisini birden değil; `Content-Type` ve
sınır (boundary) AhdCode'undur:

- zaten multipart parçaları olan bir istekte `withBody` `HTTPError` fırlatır
- zaten gövdesi olan bir istekte `withMultipartField` / `withMultipartFile`
  `HTTPError` fırlatır
- multipart parçaları olan bir isteğe `Content-Type` koymak ya da zaten
  `Content-Type` koyan bir isteğe parça eklemek `HTTPError` fırlatır

Hiçbir şey sessizce ezilmez: belirsiz bir istek reddedilir.

Gövdenin uzunluğu dosyalar okunmadan bilinemediği için multipart bir istek
parçalı (chunked) gönderilir. `307` veya `308` yönlendirmesi, aynı
dosyaları yeniden açıp aynı sınırı yeniden yazarak isteği tekrarlar; böylece
tekrarlanan gövde tam olarak `Content-Type` başlığının tarif ettiğidir. Bu
arada okunamaz hale gelen bir dosya, boş gövde göndermek yerine isteği
başarısız kılar.

Her dosya parçası istek başlamadan önce denetlenir: her yol okunabilir bir
düzenli dosya olmalıdır; eksik bir dosya, bir dizin ya da düzenli olmayan
bir şey, ağa hiçbir şey ulaşmadan `HTTPError` fırlatır.

Üst veri sınırlıdır, böylece istek istek olarak kalır: en fazla 64 metin
alanı ve 32 dosya, en fazla 256 baytlık alan adı, en fazla 255 baytlık
sunum dosya adı, en fazla 255 baytlık içerik türü ve en fazla 1 MiB'lık alan
değeri (büyük bir değer dosya parçasına aittir). Dosya *içeriği* akıtılır ve
yalnızca tüm dosya parçaları için, gövde gönderilmeden önce ölçülen 2 GiB'lık
toplam yükleme sınırıyla sınırlanır. Bu sınır sıradan her yüklemenin çok
ötesindedir ve yine de kaçak bir programı sınırlar; bir bellek sınırı
değildir.

Alan adları ve dosya adları CR, LF, tırnak ya da NUL baytı içeriyorsa
reddedilir; böylece programın geçirdiği hiçbir şey parça başlığı
enjekte edemez.

## Hatalar ve sınırlama

`HTTPError`, `Error`'dan türer. İşleyici herhangi bir Error fırlatırsa istemci
**500** `Internal Server Error` alır. İç ileti stderr'e yazılır, istemciye
değil. Sunucu sonraki isteklere hizmet etmeye devam eder. 400 ve üzeri HTTP
durum kodu tek başına `HTTPError` değildir.

## WebSocket uç noktaları

v1.4.0 WebSocket sunucu uç noktaları ekler. `HTTP.websocket(onMessage)`
değiştirilemez bir `WebSocketEndpoint` kurar; `withOpen`, `withClose`,
`withAccept`, `withAllowedOrigins` ve üç sınır yapılandırılmış kopyalar
döndürür; `Server.websocket(path, endpoint)` onu `start()` çağrısından önce
kaydeder. [WebSocket rehberi](WEBSOCKET_TR.md) yüzeyi, el sıkışmayı ve yaşam
döngüsünü, kökenleri, sınırları ve ters vekil arkasında dağıtımı anlatır.

## Eşzamanlılık

Go `net/http` bağlantıları eşzamanlı kabul eder. Bir `Server` üzerindeki
AhdCode işleyicileri dışa kapalı bir kilit ile **serileştirilir**. Aynı
sunucuda iki işleyici aynı anda çalışmaz; WebSocket geri çağrıları da aynı
kilidi paylaşır.

## Bu modülün yapmadıkları

Gelen sunucunun HTTPS dinleyicisi yoktur. Gelen yüklemeler asla genel bir
Bytes tipine ya da veritabanı BLOB'una dönüşmez; yüklenen bir belgeyi hiçbir
şey ayrıştırmaz, çizmez veya taramaz. `HTTP.file`/`HTTP.download` (v0.9.1)
yalnızca uygulamanın adlandırdığı tek bir yolu sunar; bunlar bir statik
dosya kökü, URL-yol eşlemesi, dizin tarayıcı, medya akış çerçevesi, önbellek/
ETag çerçevesi ya da ilerleme API'si değildir ve parçalı/sürdürülebilir
yükleme yoktur. Giden istemci multipart dosya yüklemeleri gönderir ve
indirmeleri dosyaya yazar (v2.1); ancak çerez kavanozu, ikili gövde veya
genel bayt tipi, okuyucu/yazıcı akış API'si, SSE, otomatik yeniden deneme,
OAuth, özel CA, istemci sertifikası, güvensiz TLS baypası, vekil API'si
veya yapay zeka satıcı modülü yoktur. WebSocket istemcisi ayrı bir tür
çiftidir ve [WebSocket](WEBSOCKET_TR.md) belgesinde anlatılır. HTTP/2 veya HTTP/3 API'si,
veritabanı oturumu, kimlik doğrulama çerçevesi, CSRF, middleware, yol
parametreleri, joker, regex yolları, ters vekil, sıkıştırma API'si veya
önbellekleme yoktur. `server.static` (v0.14), tek bir açık kök altındaki
yerel dosyaları sunar; dizin listelemesi, `index.html` kuralı, varlık
hash'leme, paketleme, küçültme veya CDN yoktur. JSON, HTTP tarafından ima
edilmez.
