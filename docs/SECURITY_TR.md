# Security standart modülü

[English](SECURITY.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [HTTP](HTTP_TR.md) · [SQLite](SQLITE_TR.md) · [Öğrenci Rehberi](STUDENT_GUIDE_TR.md#50-güvenlik-parola-hashleme-ve-güvenli-belirteçler)

`Security`, AhdCode v0.10.0 ile eklenen, derleyiciye kayıtlı
`builtin:Security` modülüdür. Açıktır (explicit); kardeş bir `Security.ahd`
dosyası onu gölgeleyemez:

```ahd
bring Security
from Security bring SecurityError
```

`Security`, dar ve görüş sahibi (opinionated) bir kriptografik ilkeler
kümesidir: Argon2id parola hashleme, opak rastgele belirteçler, sabit zamanlı
karşılaştırma, SHA-2 özetleri, HMAC, yaygın kodlamalar, RS256 imzaları ve
AES-256-GCM şifreleme. Tam bir kimlik doğrulama çatısı ve bir JWT kütüphanesi
**değildir**: `rsaSignSHA256` ve `base64UrlEncode` bir JWT'nin yapıldığı
parçaları verir, ancak belirteçleri birleştirmek, doğrulamak ve yenilemek
programınızın işidir.

Bir kriptografik seçimin söz konusu olduğu yerde seçim bir kez, güvenli biçimde
yapılır ve ayar olarak dışarı açılmaz — tek bir özet ailesi, tek bir MAC, tek bir
imza şeması, tek bir doğrulamalı şifre. Bozuk bir kip seçemez veya bir doğrulama
etiketini (authentication tag) unutamazsınız.

## ⚠ Kritik uyarılar

- **Asla düz metin parola saklamayın.** Her zaman `passwordHash`'ten dönen PHC dizisini saklayın.
- **`Security.token`'ı asla JWT olarak kullanmayın.** Belirteçler iddia (claim) taşımaz ve imzalı değildir.
- **Parolaları veya ham belirteçleri asla loglamayın.** Bu modülün hata mesajları bunları hiçbir zaman içermez.
- **Parola saklamak için asla genel hash fonksiyonları (SHA-256, MD5) kullanmayın.**
  Bu fonksiyonlar hız için tasarlanmıştır; Argon2id ise yavaş olmak için tasarlanmıştır.
- `Security` ilkeler sağlar. Eksiksiz bir kimlik doğrulama sistemini bunların üzerine kurun.

## Genel yüzey

```text
Security.passwordHash(password: String)                  -> String
Security.passwordVerify(password: String, encodedHash: String) -> Bool
Security.token()                                         -> String
Security.secureEqual(expected: String, received: String) -> Bool

Security.sha256(text: String)                            -> String
Security.sha512(text: String)                            -> String
Security.hmacSHA256(key: String, message: String)        -> String
Security.hmacVerify(key: String, message: String, receivedHex: String) -> Bool

Security.base64Encode(text: String)                      -> String
Security.base64Decode(encoded: String)                   -> String
Security.base64UrlEncode(text: String)                   -> String
Security.base64UrlDecode(encoded: String)                -> String
Security.hexEncode(text: String)                         -> String
Security.hexDecode(encoded: String)                      -> String

Security.randomHex(count: Int)                           -> String

Security.rsaSignSHA256(privateKeyPem: String, message: String)   -> String
Security.rsaVerifySHA256(publicKeyPem: String, message: String, signature: String) -> Bool
Security.aesEncrypt(keyHex: String, plaintext: String)   -> String
Security.aesDecrypt(keyHex: String, payload: String)     -> String
```

### Özetler, MAC'ler ve kodlamalar

`sha256` ve `sha512` küçük harfli hex döndürür. `hmacSHA256` de küçük harfli hex
döndürür. Alınan bir MAC'i asla `==` ile değil, `hmacVerify` ile karşılaştırın:
`hmacVerify` sabit zamanlı karşılaştırma kullanır; bu yüzden baştaki kaç
karakterin eşleştiğini sızdırmaz.

`base64UrlEncode`, URL güvenli alfabeyi (`-` ve `_`) dolgu (padding)
**olmadan** üretir; JWT ve JWS parçalarının gerektirdiği budur.
`base64UrlDecode` dolgulu girdiyi de kabul eder, çünkü başka sistemler onu
üretir.

`randomHex(count)`, `count` adet kriptografik olarak rastgele baytı hex olarak
döndürür; bu yüzden dizi bayt sayısının iki katı uzunluğundadır. `count`, 1 ile
1024 arasında olmalıdır. Otuz iki bayt, bir AES-256 anahtarı veya bir HMAC gizi
için doğru boyuttur.

### İmzalar

`rsaSignSHA256`, SHA-256 üzerinde RSASSA-PKCS1-v1_5'tir — JSON Web Token'ların
**RS256** dediği algoritma — ve imzayı dolgusuz base64url olarak döndürür. Özel
anahtar PEM biçimindedir; PKCS#1 (`RSA PRIVATE KEY`) veya PKCS#8
(`PRIVATE KEY`) olabilir; servis hesabı (service-account) dosyaları PKCS#8
kullanır. `rsaVerifySHA256` bir PKIX veya PKCS#1 genel anahtar kabul eder ve
hatalı bir imza için `false` döndürür; yalnızca anahtarın veya kodlamanın
kendisi kullanılamaz olduğunda hata fırlatır.

`base64UrlEncode` ile birlikte bunlar imzalı bir JWT oluşturmak için yeterlidir:

```ahd
header: String := Security.base64UrlEncode("{\"alg\":\"RS256\",\"typ\":\"JWT\"}")
claims: String := Security.base64UrlEncode(claimsJson)
signature: String := Security.rsaSignSHA256(privateKeyPem, header + "." + claims)
token: String := header + "." + claims + "." + signature
```

### Simetrik şifreleme

`aesEncrypt` ve `aesDecrypt` AES-256-GCM'dir. Anahtar, 64 hex karakter olarak
verilen 32 bayttır; başka her uzunluk reddedilir. Nonce her çağrıda üretilir ve
döndürülen base64 yükünün (payload) içinde taşınır; bu yüzden çağıran onu hiç
yönetmez — aynı anahtarla bir nonce'u yeniden kullanmak GCM'nin güvenliğini yok
eder ve bu API o hatayı imkânsız kılar.

Şifre çözme doğrulamalıdır: değiştirilmiş veya kesilmiş bir yük, saldırganın
etkilediği düz metni döndürmek yerine `SecurityError` fırlatır. Kip seçimi
yoktur; bu yüzden ECB gibi bozuk bir kip yanlışlıkla seçilemez.

### Hata türü

```ahd
from Security bring SecurityError
```

`SecurityError`, `Error`'ı genişletir. Şu durumlarda fırlatılır:
- Hatalı biçimlendirilmiş veya kesilmiş PHC dizileri
- Saklanmış bir hash'te desteklenmeyen algoritma veya sürüm
- Güvenli sınırların dışındaki parametreler (Argon2 çalıştırılmadan önce denetlenir)
- Rastgele üretim sırasında entropi hatası (son derece nadir)
- `base64Decode`, `base64UrlDecode` veya `hexDecode`'a verilen hatalı girdi,
  `rsaVerifySHA256`'da base64url olmayan bir imza veya `aesDecrypt`'te standart
  Base64 olmayan bir yük
- 1..1024 aralığının dışındaki bir `randomHex` sayısı
- PEM olmayan, ayrıştırılamayan veya RSA anahtarı olmayan bir anahtar
  (`rsaSignSHA256`, `rsaVerifySHA256`) ya da ayrıştırılmış ama imzalama
  işleminin reddettiği bir anahtar
- Hex olmayan veya 32 bayt olmayan bir AES anahtarı (`aesEncrypt`,
  `aesDecrypt`); `aesDecrypt` anahtarı yükten önce denetler
- Çok kısa olan veya doğrulamadan geçemeyen bir `aesDecrypt` yükü — yanlış
  anahtar ya da değiştirilmiş veya kesilmiş veri

Yanlış parolalar `false` döndürür; **asla** `SecurityError` fırlatmaz. Aynı
şekilde `hmacVerify` ve `rsaVerifySHA256`, eşleşmeyen bir MAC veya imza için —
geçerli hex olmayan bir alınan MAC dahil — `false` döndürür.
`sha256`, `sha512`, `hmacSHA256`, `hmacVerify`, `base64Encode`,
`base64UrlEncode`, `hexEncode` ve `secureEqual` hiçbir zaman hata fırlatmaz.

## passwordHash

```ahd
hash: String := Security.passwordHash("ornek_sahte_parola")
```

`password`'ü Argon2id ile hashler ve PHC (Password Hashing Competition) kodlu
bir dizi döndürür. Kodlama algoritmayı, parametreleri, tuzu ve türetilmiş
anahtarı birlikte sakladığından `passwordVerify` yalnızca bu tek diziye ihtiyaç
duyar.

**Argon2id parametreleri (v0.10.0):**

| Parametre | Değer | Anlam |
|-----------|-------|-------|
| algoritma | argon2id | Bellek yoğun, yan kanal saldırılarına dirençli |
| sürüm | v19 (0x13) | RFC 9106 |
| bellek | 65 536 KiB | Hash başına 64 MiB |
| yineleme | 3 | Zaman maliyeti |
| paralellik | 1 | İş parçacığı sayısı |
| tuz | 16 bayt | Hash başına kriptografik olarak rastgele |
| türetilmiş anahtar | 32 bayt | Çıktı uzunluğu |

**PHC dizi biçimi:**

```
$argon2id$v=19$m=65536,t=3,p=1$<base64-tuz>$<base64-anahtar>
```

Hem `<base64-tuz>` hem de `<base64-anahtar>`, base64url değil, standart
dolgusuz Base64 (`base64.RawStdEncoding`) kullanır.

**Parola boyut sınırı:** 1 MiB (1 048 576 bayt). Daha büyük girdiler herhangi
bir hashleme başlamadan `SecurityError` fırlatır. Boş parolalara izin verilir.

**UTF-8 davranışı:** Parola, AhdCode'daki diğer her `String` gibi ham UTF-8
baytları olarak ele alınır.

## passwordVerify

```ahd
ok: Bool := Security.passwordVerify(aday, sakliHash)
```

`storedHash`'i ayrıştırır, parametrelerini doğrular, saklanan tuzla Argon2id'yi
yeniden hesaplar ve sonucu `crypto/subtle.ConstantTimeCompare` ile karşılaştırır.

| Girdi koşulu | Sonuç |
|--------------|-------|
| Doğru parola | `true` |
| Yanlış parola | `false` |
| Hatalı PHC dizisi | `SecurityError` fırlatır |
| Desteklenmeyen algoritma (argon2id değil) | `SecurityError` fırlatır |
| Desteklenmeyen sürüm (v19 değil) | `SecurityError` fırlatır |
| Güvenli sınırların dışındaki parametreler | `SecurityError` fırlatır (Argon2 çalışmadan önce) |

**Güvenli parametre sınırları (doğrulama):**

| Parametre | En az | En çok |
|-----------|-------|--------|
| bellek | 8 192 KiB | 262 144 KiB |
| yineleme | 1 | 10 |
| paralellik | 1 | 16 |
| tuz uzunluğu | 8 bayt | 64 bayt |
| hash uzunluğu | 16 bayt | 64 bayt |

Bu sınırlar, saklanmış bir hash'in doğrulayıcıyı aşırı kaynak harcamaya veya
patolojik derecede zayıf parametreler kullanmaya zorlamasını önler.

## token

```ahd
tok: String := Security.token()
```

`crypto/rand`'dan 32 rastgele bayt üretir ve bunları `base64.RawURLEncoding`
(dolgusuz) ile kodlar. Sonuç her zaman 43 karakterdir, yalnızca URL güvenli
karakterler (`A–Z`, `a–z`, `0–9`, `-`, `_`) kullanır ve 256 bit entropi taşır.

`Security.token`, entropi hatasında kesin olarak başarısız olur; asla daha zayıf
bir kaynağa geri dönmez.

Belirteçleri şunlar için kullanın:
- CSRF gizli alanları
- Parola sıfırlama bağlantıları
- Oturum kimlikleri (`HTTP.sessions` kullanmıyorsanız)

Belirteçleri JWT olarak **kullanmayın** — iddia taşımazlar, son kullanma
süreleri yoktur ve imzalı değildirler.

## secureEqual

```ahd
ayni: Bool := Security.secureEqual(beklenen, alinan)
```

İki diziyi `crypto/subtle.ConstantTimeCompare` kullanarak sabit zamanda
karşılaştırır. Yalnızca iki dizi bayt bayt özdeş olduğunda `true` döndürür.
Farklı uzunluktaki girdilerde asla paniklemez.

Güvenilmeyen bir kaynaktan gelen bir değeri bilinen bir gizle (CSRF belirteci,
API anahtarı, webhook imzası) karşılaştırdığınız her durumda `secureEqual`
kullanın. Sıradan `==` sabit zamanlı değildir ve zamanlama farkları üzerinden
gizle ilgili bilgi sızdırabilir.

## CSRF koruma kalıbı

```ahd
bring HTTP
bring Security
from HTTP bring (Server, Request, Response, SessionStore, Session)

sessions: SessionStore := HTTP.sessions("SESSID")

showForm: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    token: Local String := Security.token()
    session.set("csrf", token)
    page: Local String := """<form method="post" action="/submit">
<input type="hidden" name="csrf" value="{token}"/>
<button>Submit</button></form>"""
    return sessions.commit(session, HTTP.html(page))
}

handleSubmit: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    stored: Local String? := session.get("csrf")
    submitted: Local String? := request.form("csrf")
    if stored == null or submitted == null {
        return sessions.commit(session, HTTP.text("rejected", 403))
    }
    if Security.secureEqual(stored, submitted) {
        session.set("csrf", Security.token()) // rotate after use
        return sessions.commit(session, HTTP.text("ok"))
    }
    return sessions.commit(session, HTTP.text("rejected", 403))
}

app: Server := HTTP.server("127.0.0.1", 8080)
app.get("/form", showForm)
app.post("/submit", handleSubmit)
app.start()
```

İşleyiciler, `app.get` ve `app.post` ile kaydedilen sıradan adlandırılmış
Function'lardır. Her yanıt, oturum çerezini yazan `sessions.commit` üzerinden
geçer.

## SQLite saklama örneği

Yalnızca PHC dizisini saklayın. Ayrı bir tuz sütunu gerekmez.

```ahd
bring Security
bring SQLite
from Security bring SecurityError

db := SQLite.open("users.db")
db.execute("CREATE TABLE IF NOT EXISTS users (username TEXT PRIMARY KEY, hash TEXT NOT NULL)")

// Register
fn register(db: Database, username: String, password: String) {
    db.execute("INSERT INTO users (username, hash) VALUES (?, ?)",
        [SQLite.fromString(username), SQLite.fromString(Security.passwordHash(password))])
}

// Login
fn login(db: Database, username: String, attempt: String) -> Bool {
    rows := db.query("SELECT hash FROM users WHERE username = ?",
        [SQLite.fromString(username)])
    if len(rows) == 0 { return false }
    return Security.passwordVerify(attempt, rows[0]["hash"].string())
}
```

## Hata mesajları

| Mesaj | Anlam |
|-------|-------|
| `Security password hash is malformed` | PHC dizisi ayrıştırılamadı |
| `Security password hash uses an unsupported algorithm` | argon2id değil / v19 değil |
| `Security password hash has unsafe parameters` | Parametreler güvenli sınırların dışında |
| `Security password input is too large` | Parola 1 MiB'ı aştı |
| `Security random token generation failed` | İşletim sistemi entropi hatası (`token`, `passwordHash`, `randomHex`, `aesEncrypt`) |
| `Security base64 input is malformed` | `base64Decode` girdisi veya `aesDecrypt` yükü dolgulu standart Base64 değil |
| `Security base64url input is malformed` | `base64UrlDecode` girdisi veya `rsaVerifySHA256` imzası base64url değil |
| `Security hex input is malformed` | `hexDecode` tek sayıda karakter veya hex olmayan bir karakter aldı |
| `Security random byte count must be between 1 and 1024` | `randomHex` sayısı aralık dışında |
| `Security private key is not valid PEM` | `rsaSignSHA256` anahtarında PEM bloğu yok |
| `Security private key could not be parsed` | PEM bloğu ne PKCS#1 ne de PKCS#8 |
| `Security private key is not an RSA key` | EC gibi başka bir algoritmaya ait PKCS#8 anahtarı |
| `Security RSA signing failed` | Anahtar ayrıştırıldı ama imzalama onu reddetti; örneğin 512 bitlik bir anahtar |
| `Security public key is not valid PEM` | `rsaVerifySHA256` anahtarında PEM bloğu yok |
| `Security public key could not be parsed` | PEM bloğu ne PKCS#1 ne de PKIX |
| `Security public key is not an RSA key` | EC gibi başka bir algoritmaya ait PKIX anahtarı |
| `Security AES key must be hex encoded` | `aesEncrypt` / `aesDecrypt` anahtarı hex değil |
| `Security AES key must be 32 bytes (64 hex characters)` | Anahtar hex ama 32 bayt değil |
| `Security AES-GCM payload is too short` | Çözülmüş yük 12 baytlık nonce'tan kısa |
| `Security AES-GCM authentication failed` | Yanlış anahtar ya da değiştirilmiş veya kesilmiş şifreli metin |
| `Security AES cipher could not be created` | İç şifre kurulum hatası; anahtar geçerliyken beklenmez |
| `Security AES-GCM could not be created` | İç GCM kurulum hatası; anahtar geçerliyken beklenmez |

Parolalar, anahtarlar, düz metin ve MAC değerleri hiçbir zaman hata mesajlarında yer almaz.

## Ayrıca bakınız

- [v0.10 örnekleri](../examples/v0.10/README.md)
- [Öğrenci Rehberi — Güvenlik](STUDENT_GUIDE_TR.md#50-güvenlik-parola-hashleme-ve-güvenli-belirteçler)
- [HTTP modülü](HTTP_TR.md) — oturumlar, CSRF bağlamı
- [SQLite modülü](SQLITE_TR.md) — hash'leri saklama
- [Env modülü](ENV_TR.md) — gizleri ortamdan yükleme
