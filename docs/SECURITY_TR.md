# Security standart modülü

[English](SECURITY.md) · [Türkçe]

[README'ye dön](../README.md) · [Modüller](MODULES_TR.md) · [HTTP](HTTP_TR.md) · [SQLite](SQLITE_TR.md) · [Öğrenci Rehberi](STUDENT_GUIDE_TR.md#50-güvenlik-parola-hashleme-ve-güvenli-belirteçler)

`Security`, AhdCode v0.10.0 ile eklenen derleyici tarafından kayıtlı
`builtin:Security` modülüdür. Explicit'tir; bir `Security.ahd` kardeş dosyası
onu gölgeleyemez:

```ahd
bring Security
from Security bring SecurityError
```

`Security`, dar bir kriptografik ilkeler kümesidir: Argon2id parola hashleme,
rastgele belirteç üretme ve sabit zamanlı karşılaştırma. Tam bir kimlik
doğrulama çerçevesi, JWT kütüphanesi veya şifreleme API'si **değildir**.

## ⚠ Kritik uyarılar

- **Asla düz metin parola saklamayın.** `passwordHash`'ten dönen PHC dizisini saklayın.
- **`Security.token`'ı JWT olarak kullanmayın.** Belirteçler iddia taşımaz ve imzalı değildir.
- **Parolaları veya ham belirteçleri asla loglamamayın.** Modülün hata mesajları bunları hiçbir zaman içermez.
- **Parola saklama için genel hash fonksiyonları (SHA-256, MD5) kullanmayın.**
  Bu fonksiyonlar hız için tasarlanmıştır; Argon2id yavaşlık için tasarlanmıştır.
- `Security` ilkeler sağlar. Üzerine eksiksiz bir kimlik doğrulama sistemi inşa edin.

## Genel arayüz

```text
Security.passwordHash(password: String)                  -> String
Security.passwordVerify(password: String, encodedHash: String) -> Bool
Security.token()                                         -> String
Security.secureEqual(expected: String, received: String) -> Bool
```

### Hata türü

```ahd
from Security bring SecurityError
```

`SecurityError`, `Error`'ı genişletir. Şunlarda yükseltilir:
- Hatalı biçimlendirilmiş veya eksik PHC dizileri
- Saklanmış hash'te desteklenmeyen algoritma veya sürüm
- Güvenli sınırlar dışındaki parametreler (Argon2 çalıştırılmadan önce kontrol edilir)
- Rastgele üretme sırasında entropi hatası (son derece nadir)

Yanlış parolalar `false` döndürür; asla `SecurityError` yükseltmez.

## passwordHash

```ahd
hash: String := Security.passwordHash("ornek_sahte_parola")
```

`password`'ü Argon2id ile hashler ve PHC (Password Hashing Competition) kodlu
bir dizi döndürür. Kodlama; algoritma, parametreler, tuz ve türetilmiş anahtarı
birlikte sakladığından `passwordVerify` yalnızca bu tek diziye ihtiyaç duyar.

**Argon2id parametreleri (v0.10.0):**

| Parametre | Değer | Anlam |
|-----------|-------|-------|
| algoritma | argon2id | Bellek yoğun, yan kanal dirençli |
| sürüm | v19 (0x13) | RFC 9106 |
| bellek | 65 536 KiB | Hash başına 64 MiB |
| yineleme | 3 | Zaman maliyeti |
| paralellik | 1 | İş parçacığı sayısı |
| tuz | 16 bayt | Hash başına kriptografik olarak rastgele |
| türetilmiş anahtar | 32 bayt | Çıkış uzunluğu |

**PHC dizi biçimi:**

```
$argon2id$v=19$m=65536,t=3,p=1$<base64-tuz>$<base64-anahtar>
```

**Parola boyut sınırı:** 1 MiB (1 048 576 bayt). Daha büyük girdiler,
herhangi bir hashleme başlamadan `SecurityError` yükseltir. Boş parolalara
izin verilir.

**UTF-8 davranışı:** Parola, AhdCode'daki diğer tüm `String`'ler gibi ham
UTF-8 baytları olarak ele alınır.

## passwordVerify

```ahd
ok: Bool := Security.passwordVerify(aday, sakliHash)
```

`storedHash`'i ayrıştırır, parametrelerini doğrular, saklanan tuzla Argon2id'yi
yeniden hesaplar ve sonucu `crypto/subtle.ConstantTimeCompare` ile karşılaştırır.

| Giriş koşulu | Sonuç |
|--------------|-------|
| Doğru parola | `true` |
| Yanlış parola | `false` |
| Hatalı PHC dizisi | `SecurityError` yükseltir |
| Desteklenmeyen algoritma (argon2id değil) | `SecurityError` yükseltir |
| Desteklenmeyen sürüm (v19 değil) | `SecurityError` yükseltir |
| Güvenli sınırlar dışındaki parametreler | `SecurityError` yükseltir (Argon2 çalışmadan önce) |

## token

```ahd
tok: String := Security.token()
```

`crypto/rand`'dan 32 rastgele bayt üretir ve `base64.RawURLEncoding` (dolgu
yok) ile kodlar. Sonuç her zaman 43 karakterdir, yalnızca URL güvenli
karakterler (`A–Z`, `a–z`, `0–9`, `-`, `_`) kullanır ve 256 bit entropi taşır.

`Security.token`, entropi hatasında kesin olarak başarısız olur; daha zayıf
bir kaynağa geri dönmez.

## secureEqual

```ahd
ayni: Bool := Security.secureEqual(beklenen, alinan)
```

`crypto/subtle.ConstantTimeCompare` kullanarak iki diziyi sabit zamanda
karşılaştırır. Her iki dizi bayt bayt özdeş olduğunda yalnızca `true` döner.
Farklı uzunluktaki girdilerde asla paniklemez.

## CSRF koruma kalıbı

```ahd
bring HTTP
bring Security

app := HTTP.server("127.0.0.1", 8080)
store := HTTP.sessionStore("SESSID", Env.getOr("SESSION_SECRET", "sadece-gelistirme"))

app.get("/form", fn(req) -> Response {
    session := store.session(req)
    tok := Security.token()
    session.set("csrf", tok)
    return HTTP.html("<form method='POST' action='/gonder'>" +
        "<input type='hidden' name='csrf' value='" + tok + "'/>" +
        "<button>Gönder</button></form>")
})

app.post("/gonder", fn(req) -> Response {
    session := store.session(req)
    saklanan: String?   := session.get("csrf")
    gonderilen: String? := req.field("csrf")
    if saklanan == null or gonderilen == null {
        return HTTP.text("reddedildi", 403)
    }
    if Security.secureEqual(saklanan, gonderilen) {
        session.set("csrf", Security.token())
        return HTTP.text("kabul edildi")
    }
    return HTTP.text("reddedildi", 403)
})
```

## Hata mesajları

| Mesaj | Anlam |
|-------|-------|
| `Security password hash is malformed` | PHC dizisi ayrıştırılamadı |
| `Security password hash uses an unsupported algorithm` | argon2id veya v19 değil |
| `Security password hash has unsafe parameters` | Parametreler güvenli sınırlar dışında |
| `Security password input is too large` | Parola 1 MiB'ı aştı |
| `Security random token generation failed` | İşletim sistemi entropi hatası |

Parolalar hiçbir zaman hata mesajlarında yer almaz.

## Ayrıca bakınız

- [v0.10 örnekleri](../examples/v0.10/README.md)
- [HTTP modülü](HTTP_TR.md)
- [SQLite modülü](SQLITE_TR.md)
- [Env modülü](ENV_TR.md)
