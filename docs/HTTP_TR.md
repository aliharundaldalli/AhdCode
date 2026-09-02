# HTTP standart modülü

[English](HTTP.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [HTML](HTML_TR.md) · [Öğrenci Rehberi](STUDENT_GUIDE_TR.md#37-çerezler-ve-oturumlar)

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
```

`HTTP` küçük, tipli yerel bir web sunucusudur. Tam yolları kaydeder, her
isteğin bir anlık kopyasını okur ve bir `Response` döndürürsünüz. v0.5.0 HTTP
çerezleri ve bellek içi sunucu taraflı `SessionStore` ekler. Middleware,
yönlendirici DSL'si, HTTPS, multipart, WebSocket, yol parametresi veya kimlik
doğrulama çerçevesi yoktur. Uygulama, AhdCode çalışma zamanının içindeki Go
`net/http` paketini kullanır; ayrı bir HTTP, çerez veya oturum yardımcı
süreci yoktur.

## Genel yüzey

```text
HTTP.server(host: String, port: Int, maxBodyBytes: Int := 1048576) -> Server
HTTP.response(status: Int, body: String, contentType: String)      -> Response
HTTP.text(body: String, status: Int := 200)                        -> Response
HTTP.html(body: String, status: Int := 200)                        -> Response
HTTP.redirect(location: String, status: Int := 303)                -> Response
HTTP.cookie(name: String, value: String)                           -> Cookie
HTTP.deleteCookie(name: String, path: String := "/")               -> Cookie
HTTP.sessions(
    cookieName: String := "ahd_session"
    maxAgeSeconds: Int := 86400
    secure: Bool := false
    sameSite: String := "Lax"
) -> SessionStore

Server.get(path: String, handler: Function)   -> Nothing
Server.post(path: String, handler: Function)  -> Nothing
Server.route(method: String, path: String, handler: Function) -> Nothing
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

HTTPError  (Error'dan türer)
```

`Server`, `Request`, `Response`, `Cookie`, `SessionStore` ve `Session` opak
yerleşik Sınıflardır: `Server()`, `Request()`, `Response()`, `Cookie()`,
`SessionStore()` veya `Session()` ile oluşturulamazlar. Atlanan
`HTTP.sessions` argümanları `ahd_session`, `86400`, `false` ve `"Lax"`'tır.

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

`127.0.0.1` yalnızca bu makinedir. Aynı bilgisayarın tarayıcısında
`http://127.0.0.1:8080/` açın. `0.0.0.0` bağlamak portu ağdaki diğer
makinelere açar; bunu gelişigüzel yapmayın.

İç zaman aşımları (genel API değildir): ReadHeader 10s, Read 15s, Write 15s,
Idle 60s. Erişim günlüğü yoktur.

## Yönlendirme

Yollar tam eşleşmedir. Yol `/` ile başlamalı, `?` veya `#` içermemelidir.
`/notes` ile `/notes/` farklıdır. Sorgu dizesi yolun parçası değildir.
Aynı `method + path` tekrarı `HTTPError` fırlatır. `start()` sonrası yollar
değişmez.

Bilinmeyen yollar **404** döner. Başka bir yöntem için var olan bir yol
**405** ve `Allow` başlığı döner.

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

Formlar yalnızca `Content-Type` `application/x-www-form-urlencoded` iken
ayrıştırılır. `form` / `formAll` sorgu erişicileri gibi davranır. Aynı katı
yüzde kod çözümü form gövdesine de uygulanır: bozuk veya UTF-8 olmayan form
verisi işleyiciden **önce** **400** döner. Gövde, işleyiciden **önce**
`maxBodyBytes` ile sınırlanır; aşım **413** döner.

## Yanıt

`HTTP.text` `text/plain; charset=utf-8` ayarlar. `HTTP.html`
`text/html; charset=utf-8` ayarlar ve gövdeyi **kaçırmaz**: güvenilir statik
işaretleme (`r"""..."""`) veya `HTML.document` / `HTML.render` çıktısı için
kullanın. Durum kodları `100..599` olmalıdır. `HTTP.redirect` yalnızca
`301`, `302`, `303`, `307` veya `308` kabul eder. `withHeader` yeni bir
`Response` döndürür; CR/LF yasaktır. `Set-Cookie` `withCookie` ile eklenir;
`withHeader` birden fazla çerezi tek başlıkta ezmez.

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

`HTTP.sessions` bağımsız, bellek içi bir `SessionStore` oluşturur. İki store
durum paylaşmaz. Oturum çerezi yalnızca opak rastgele bir kimlik taşır
(`crypto/rand`, en az 256 bit, padding'siz base64url). Değerler yalnızca
String'dir ve sunucudadır. Süreç bitince kaybolur; bu tasarım gereğidir, hata
değildir. Kimlik doğrulama çerçevesi yoktur.

Varsayılan oturum çerezi: Path `/`, HttpOnly her zaman `true`, Max-Age =
`maxAgeSeconds`. HttpOnly kapatılamaz. Süresi dolmuş oturumlar kabul edilmez.
Temizlik `open`/`commit` sırasında tembeldir.

İşleyici içinde modül düzeyindeki store `Global` ile bildirilir:

```ahd
sessions: Global SessionStore
session: Local Session := sessions.open(request)
```

Bilinmeyen veya süresi dolmuş çerez yeni anonim oturum olur; saldırgan
kimliği sunucu kimliği olarak kullanılmaz. `set`/`remove`/`clear`/`rotate`
olmadan `open` kalıcı durum veya `Set-Cookie` üretmez. `Session.set` başlık
yazmaz; `commit` yeni bir `Response` döndürür.

`rotate()` eski kimliği geçersiz kılar, değerleri korur, yeni çerez gönderir.
`destroy()` sunucu durumunu siler; `commit` silme çerezi gönderir. Destroy
sonrası `get` `null`, `has` `false`; mutasyon `HTTPError` fırlatır.
`remove` bir anahtarı, `clear` değerleri, `destroy` oturumu siler.

## Hatalar ve sınırlama

`HTTPError`, `Error`'dan türer. İşleyici herhangi bir Error fırlatırsa istemci
**500** `Internal Server Error` alır. İç ileti stderr'e yazılır, istemciye
değil. Sunucu sonraki isteklere hizmet etmeye devam eder.

## Eşzamanlılık

Go `net/http` bağlantıları eşzamanlı kabul eder. Bir `Server` üzerindeki
AhdCode işleyicileri dışa kapalı bir kilit ile **serileştirilir**. Aynı
sunucuda iki işleyici aynı anda çalışmaz.

## Bu modülün yapmadıkları

HTTPS/TLS, HTTP/2 API, HTTP/3, WebSocket, SSE, multipart, dosya yükleme,
statik dosya sunucusu, veritabanı oturumu, kimlik doğrulama, CSRF, middleware, yol
parametreleri, joker, regex yolları, ters vekil, sıkıştırma API'si veya
önbellekleme yoktur. Oturum değerleri String'dir.
