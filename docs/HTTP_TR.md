# HTTP standart modülü

[English](HTTP.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [HTML](HTML_TR.md) · [Öğrenci Rehberi](STUDENT_GUIDE_TR.md#36-küçük-bir-web-sayfası)

`HTTP`, AhdCode v0.4.0 ile gelen, derleyici tarafından kayıtlı
`builtin:HTTP` modülüdür. Açıkça getirilir ve yanındaki bir `HTTP.ahd`
onu gölgeleyemez:

```ahd
bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring HTTPError
```

`HTTP` küçük, tipli yerel bir web sunucusudur. Tam yolları kaydeder, her
isteğin bir anlık kopyasını okur ve bir `Response` döndürürsünüz. Middleware,
yönlendirici DSL'si, HTTPS, çerez, oturum, multipart, WebSocket veya yol
parametresi yoktur. Uygulama, AhdCode çalışma zamanının içindeki Go
`net/http` paketini kullanır; ayrı bir HTTP yardımcı süreci yoktur.

## Genel yüzey

```text
HTTP.server(host: String, port: Int, maxBodyBytes: Int := 1048576) -> Server
HTTP.response(status: Int, body: String, contentType: String)      -> Response
HTTP.text(body: String, status: Int := 200)                        -> Response
HTTP.html(body: String, status: Int := 200)                        -> Response
HTTP.redirect(location: String, status: Int := 303)                -> Response

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

Response.withHeader(name: String, value: String) -> Response

HTTPError  (Error'dan türer)
```

`Server`, `Request` ve `Response` opak yerleşik Sınıflardır: `Server()`,
`Request()` veya `Response()` ile oluşturulamazlar, herkese açık öznitelikleri
yoktur ve yalnızca yukarıdaki işlevlerden elde edilirler. Tüm argümanlar
konumsaldır. Atlanan `maxBodyBytes` `1048576`'dır. Atlanan `HTTP.text` /
`HTTP.html` durumu `200`'dür. Atlanan yönlendirme durumu `303`'tür.

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
korunarak saklanır.

`header` / `headerAll` büyük/küçük harfe duyarsızdır. `body()` UTF-8 gövdedir;
geçersiz UTF-8 `HTTPError` fırlatır (sessiz değiştirme yoktur).

Formlar yalnızca `Content-Type` `application/x-www-form-urlencoded` iken
ayrıştırılır. Gövde, işleyiciden **önce** `maxBodyBytes` ile sınırlanır;
aşım **413** döner.

## Yanıt

`HTTP.text` `text/plain; charset=utf-8` ayarlar. `HTTP.html`
`text/html; charset=utf-8` ayarlar ve gövdeyi **kaçırmaz**: güvenilir statik
işaretleme (`r"""..."""`) veya `HTML.document` / `HTML.render` çıktısı için
kullanın. Durum kodları `100..599` olmalıdır. `HTTP.redirect` yalnızca
`301`, `302`, `303`, `307` veya `308` kabul eder. `withHeader` yeni bir
`Response` döndürür; CR/LF yasaktır.

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
statik dosya sunucusu, çerez, oturum, kimlik doğrulama, CSRF, middleware, yol
parametreleri, joker, regex yolları, ters vekil, sıkıştırma API'si veya
önbellekleme yoktur. Gövdeler sınırlı String'lerdir.
