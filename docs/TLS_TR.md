# TLS standart modülü

[English](TLS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [DNS](DNS_TR.md) · [Time](TIME_TR.md) · [Hatalar](ERRORS_TR.md)

`TLS` (v2.6.0) bir TLS uç noktasını **inceler**: bağlanır, bir TLS el sıkışması
yapar, sunucunun sunduğu sertifikayı okur, doğrular ve bulduklarını bildirir —
sertifikanın ana makine için geçerli olup olmadığını, kimin verdiğini, hangi
adları kapsadığını, ne zaman sona erdiğini ve hangi TLS sürümünün
kararlaştırıldığını. Asla `openssl` ya da başka bir araç çalıştırmaz ve her
zaman bir zaman aşımıyla sınırlıdır. Açıkça içe aktarın:

```ahd
bring TLS
from TLS bring (TLSInfo, TLSError)
```

Kanonik modül kimliği `builtin:TLS`'tir; bir kardeş `TLS.ahd` onu gölgeleyemez.
(v2.6.0'dan itibaren `TLS` bir standart modül adıdır: yerel bir `TLS.ahd` artık
`bring TLS`'in yüklediği şey değildir — böyle bir dosyayı yeniden adlandırın.)

## Yüzey

```text
TLS.inspect(host: String, port: Int = 443, timeoutSeconds: Int = 10) -> TLSInfo

TLSInfo.host()               -> String          // normalleştirilmiş ana makine adı
TLSInfo.port()               -> Int
TLSInfo.valid()              -> Bool            // yalnızca doğrulama başarılıysa true
TLSInfo.verificationStatus() -> String          // aşağıya bakın
TLSInfo.subject()            -> String          // örn. "CN=example.com,O=Example"
TLSInfo.issuer()             -> String
TLSInfo.dnsNames()           -> List<String>    // sertifikadaki DNS adları, sertifika sırasıyla
TLSInfo.notBefore()          -> DateTime        // UTC
TLSInfo.notAfter()           -> DateTime        // UTC
TLSInfo.protocol()           -> String          // "TLS 1.3", "TLS 1.2", ...
TLSInfo.cipherSuite()        -> String          // örn. "TLS_AES_128_GCM_SHA256"

TLSError  (Error'dan türer)
```

`TLSInfo` yalnızca `TLS.inspect` tarafından üretilen opak bir değerdir.
Tarihler [Time](TIME_TR.md) modülünün UTC'deki `DateTime` değerleridir;
`bring TLS` onları `bring Time` olmadan kullanılabilir kılar, ancak türü
adlandırmak (`x: DateTime := ...`) ya da `Time` işlevlerini çağırmak her zamanki
gibi `bring Time` gerektirir.

```ahd
info: TLSInfo := TLS.inspect(host: "example.com", port: 443, timeoutSeconds: 5)
write(str(info.valid()) + " " + info.verificationStatus())
write(info.issuer())
write(info.notAfter().toISO())
```

## İnceleme güven demek değildir

`TLS.inspect` bozuk sertifikaları da bildirmelidir — süresi dolmuş ya da adı
uyuşmayan bir sertifika, bir izleyicinin görmesi gereken şeyin ta kendisidir.
Olağan TLS istemcileri böyle bir sertifikada durur ve fazlasını öğrenmez; bu
yüzden inceleme iki ayrı adımda çalışır:

1. **Okuma.** `host:port`'a bağlanır, `host`'u SNI sunucu adı olarak gönderir
   (bir IP adresi için SNI gönderilmez), el sıkışmasını otomatik doğrulama
   kapalıyken tamamlar, sunulan sertifika zincirini ve kararlaştırılan
   protokolü kaydeder ve bağlantıyı kapatır. Hiçbir uygulama verisi gönderilmez
   ya da alınmaz; sunucudan gelen hiçbir şeye güvenilmez ya da kullanılmaz.
2. **Doğrulama.** Ardından yaprak sertifikayı, sunucunun sunduğu ara
   sertifikaları, istenen ana makine adını ve geçerli zamanı kullanarak
   işletim sisteminin güvenilir köklerine karşı açıkça doğrular.

`valid()` yalnızca bu açık doğrulama başarılı olduğunda `true`'dur. Geçersiz bir
sertifika asla geçerli olarak bildirilmez ve asla "kabul edilmez" — yalnızca
betimlenir. `insecure` gibi bir seçenek yoktur.

`verificationStatus()` sonucu adlandırır; karar bu sabit sırayla verilir ve
bulunan ilk sorun bildirilir:

| Durum | Anlamı |
|---|---|
| `valid` | doğrulama başarılı; `valid()` `true`'dur |
| `expired` | geçerli zaman `notAfter()`'dan sonradır |
| `notYetValid` | geçerli zaman `notBefore()`'dan öncedir |
| `hostnameMismatch` | sertifika istenen ana makine adını kapsamıyor |
| `untrustedIssuer` | zincir güvenilir bir köke ulaşmıyor (kendinden imzalı, özel CA, eksik ara sertifika) |
| `invalid` | diğer her doğrulama hatası (örneğin platform doğrulayıcısının standartlara uygun bulmadığı bir sertifika) |

Her durumda üst veri — konu, veren, adlar, tarihler, protokol — kullanılabilir
kalır.

## Ana makine adı, SNI ve IP adresleri

`host`, tıpkı [`DNS.lookup`](DNS_TR.md#girdi-doğrulama) gibi doğrulanır: bir ana
makine adı ya da bir IP adresi; asla bir URL, bir yol ya da `host:port` çifti
değil. Aynı ana makine adı bağlantı, SNI ve ad denetimi için kullanılır; bu
yüzden bir sertifika her zaman sorduğunuz ada göre değerlendirilir. Bir IP
adresi (`TLS.inspect("203.0.113.10")`) o adrese bağlanır, SNI göndermez ve
sertifikanın IP adreslerine göre denetlenir.

`port` varsayılan olarak **443**'tür ve **1 ile 65535** arasında olmalıdır.
Servis keşfi yoktur.

## Zaman aşımı

Tek bir zaman aşımı sunucuyu içeren her şeyi kapsar — ana makine adını çözme,
bağlanma ve el sıkışması. `timeoutSeconds` varsayılan olarak **10**'dur ve **1
ile 60** arasında olmalıdır. Bağlantıyı kabul edip hiç yanıt vermeyen bir
sunucu, zaman aşımından sonra `TLSError` fırlatır; bağlantı `TLS.inspect`
dönmeden önce her zaman kapatılır. Doğrulama ardından, zaten alınmış
sertifikalar üzerinde yerel olarak çalışır; Windows'ta platform doğrulayıcısı
eksik bir ara sertifikayı işletim sisteminin kendi sınırları içinde kendisi
indirebilir.

## Sertifika bitiş tarihi

`TLS.inspect` politika değil, olgu bildirir: bir yenilemenin ne zaman "acil"
olduğuna karar vermez. Kalan süreyi Time modülüyle hesaplayın ve kendi
eşiklerinizi uygulayın:

```ahd
bring Math
bring TLS
bring Time

info := TLS.inspect("example.com")
remaining := Time.between(Time.utc(), info.notAfter())
days := Math.floor(remaining.milliseconds / 86400000)
if not info.valid() {
    write("certificate problem: " + info.verificationStatus())
} else if days <= 7 {
    write("renew now: " + str(days) + " days left")
} else if days <= 30 {
    write("renew soon: " + str(days) + " days left")
}
```

## Hatalar

`TLSError`, hiçbir sertifika elde edilemediği anlamına gelir. Kötü bir
sertifikası olan erişilebilir bir sunucu hata **değildir** — `valid()` değeri
`false` olan bir `TLSInfo`'dur. İletiler
`inspect "<ana makine>:<port>" failed: <neden>` biçimindedir:

```text
inspect "127.0.0.1:8443" failed: connection refused
inspect "example.com:443" failed: timed out after 10 seconds (timeoutSeconds)
inspect "example.com:80" failed: the server did not answer with TLS
inspect "example.com:443" failed: the server closed the connection during the TLS handshake
inspect "missing.example:443" failed: host not found
inspect "example.com:70000" failed: port must be between 1 and 65535; received 70000
```

## Platform notları

Doğrulama her işletim sisteminin güvenilir köklerini ve macOS ile Windows'ta
platformun kendi sertifika doğrulayıcısını kullanır. Doğrulayıcılar sıradan
sertifikalarda aynı sonuca varır; ancak platform politikaları uçlarda
ayrışabilir: örneğin macOS, geçerlilik süresi Apple'ın sınırlarını aşan bir
sunucu sertifikasını reddeder ve `TLS.inspect` orada `invalid` bildirir; Linux
aynı sertifikayı farklı bildirebilir. `valid()` her zaman platformun kararıdır
ve asla gevşetilmez.

## Güvenlik: güvenilmeyen girdi için yalnızca işçi

`TLS.inspect`, kendisine verilen her ana makine adı ve porta bir ağ bağlantısı
açar. **Güvenilmeyen bir web isteğini doğrudan ona vermeyin** — bu, bir
ziyaretçinin sunucunuzu iç adreslere bağlandırmasına izin verir. Onu yalnızca
doğrulanmış, izin listesindeki alan adlarını inceleyen özel bir işçide tutun;
mimari için [DNS](DNS_TR.md#güvenlik-güvenilmeyen-girdi-için-yalnızca-işçi)
bölümüne bakın. AhdCode'un kendine ait genel bir izin listesi yoktur.

## Bu sürümde yok

Sertifika verme ya da yenileme (ACME / Let's Encrypt), özel anahtarlar, istemci
sertifikaları, ayrı bir bağlantı adresi ve sunucu adı seçimi, özel güven
kökleri, OCSP ve CRL denetimleri, Certificate Transparency, ham sertifika ya da
uzantı dökümleri ve HTTP istekleri `TLS`'in parçası değildir.
