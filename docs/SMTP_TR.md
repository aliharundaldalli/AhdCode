# SMTP standart modülü

[English](SMTP.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Env](ENV_TR.md) · [Öğrenci Rehberi](STUDENT_GUIDE_TR.md#41-e-posta-gönderme-smtp)

`SMTP`, AhdCode v0.9.0 ile gelen derleyici kayıtlı `builtin:SMTP` modülüdür.
Açıktır (explicit); kardeş bir `SMTP.ahd` onu gölgeleyemez:

```ahd
bring SMTP
from SMTP bring SMTPClient
from SMTP bring SMTPMessage
from SMTP bring SMTPError
```

`SMTP` yalnızca gönderim yapan bir posta taşıması ve ileti bileşim
ilkelidir. Bülten çerçevesi, gelen kutusu, mailbox istemcisi, arka plan
kuyruğu veya sağlayıcıya özel bir API değildir. IMAP, POP3, ek (attachment)
desteği veya bir posta yardımcı çalıştırılabilir dosyası yoktur. Uygulama,
AhdCode çalışma zamanının içindeki Go `net/smtp`, `crypto/tls` ve MIME
kütüphanelerini kullanır.

## Genel yüzey

```text
SMTP.client(
    host: String
    port: Int
    security: String := "starttls"
    timeoutSeconds: Int := 30
) -> SMTPClient

SMTP.message(
    from: String
    to: List<String>
    subject: String
) -> SMTPMessage

SMTPClient.withPlainAuth(username: String, password: String) -> SMTPClient
SMTPClient.send(message: SMTPMessage) -> Nothing

SMTPMessage.withCc(recipients: List<String>) -> SMTPMessage
SMTPMessage.withBcc(recipients: List<String>) -> SMTPMessage
SMTPMessage.withReplyTo(address: String) -> SMTPMessage
SMTPMessage.withText(body: String) -> SMTPMessage
SMTPMessage.withHtml(body: String) -> SMTPMessage
```

Başarısızlıklar `SMTPError` kullanır. `SMTPClient` veya `SMTPMessage` doğrudan
oluşturulamaz.

## İstemci yapılandırması

`SMTP.client` yapılandırma saklar; bağlanmaz. `withPlainAuth` da bağlanmaz.
Ağ etkinliği yalnızca `client.send(message)` ile olur.

Bir gönderim bir SMTP bağlantısı açar, bir işlem çalıştırır, sonra QUIT/kapatır.
Bağlantı havuzu, kalıcı oturum veya gizli yeniden deneme yoktur.

`SMTPClient` değiştirilemezdir:

```ahd
base := SMTP.client("smtp.example.com", 587)
authenticated := base.withPlainAuth("user@example.com", password)
```

`base` kimlik doğrulamasız kalır.

`host` bir makine adı veya IP'dir; `smtp://example.com` gibi bir URL değildir.
`port` `1..65535` aralığında olmalıdır. `timeoutSeconds` `1..9223372036`
aralığındadır ve bağlanma, TLS el sıkışması, SMTP komutları ve DATA'yı kapsar.

## Güvenlik kipleri

Yalnızca tam küçük harf değerler. Girdi sessizce küçültülmez; `ssl`, `smtps`,
`opportunistic` veya `auto` gibi takma adlar reddedilir.

| Değer | Anlam |
|---|---|
| `starttls` | Düz metin SMTP ile bağlan, STARTTLS iste, yükselt, TLS doğrula |
| `tls` | Bağlantı başından örtük TLS |
| `none` | Açık düz metin SMTP |

`security` `starttls` ise ve sunucu STARTTLS ilan etmezse gönderim `SMTPError`
yükseltir. Fırsatçı düz metin geri dönüşü yoktur.

Örtük TLS (`tls`), herhangi bir SMTP komutundan önce TLS el sıkışması yapar.
Sertifika zinciri ve makine adı/IP kimliği sistem kökleriyle doğrulanır.
Herkese açık güvensiz atlama, hepsine güven veya kendinden imzalı kabul
yoktur. Güvenilmeyen bir sertifika `SMTPError` üretir.

`none` kasıtlı olarak açıktır. Yerel geliştirme armatürü için yararlı
olabilir. Güvenli olduğunu iddia etmeyin.

## AUTH PLAIN

v0.9 yalnızca AUTH PLAIN destekler (`withPlainAuth`). AUTH LOGIN, CRAM-MD5,
XOAUTH2 veya sağlayıcı OAuth yoktur.

AUTH yalnızca şifreli bir bağlantıdan sonra denenir (`starttls` veya `tls`).
İstemci kimlik doğrulamalıysa ve `security` `none` ise, gönderim kimlik
bilgileri iletilmeden `SMTPError` yükseltir.

Sunucu AUTH PLAIN ilan etmezse `SMTPError` yükselir. Yedek mekanizma yoktur.
Yanlış kimlik bilgileri yeniden denemeden `SMTPError` yükseltir.

Gerçek bir SMTP parolasını kaynağa yazmayın. [Env](ENV_TR.md) ile okuyun.

Parolayı yazdırmayın. SMTP tanıları onu asla içermez.

`SMTP.fromEnv()`, `SMTP.gmail()` veya `SMTP.outlook()` yoktur.

## İletiler

`SMTPMessage` değiştirilemezdir.

Her `String` tam olarak bir postadır. Bir liste öğesine iki alıcı koymayın.

Posta addr-spec ASCII kalmalıdır. Unicode görünen adlar başlıklarda
kodlanabilir. Unicode bir yerel parça `SMTPError` yükseltir. v0.9'da
SMTPUTF8 yoktur.

`to` boş olabilir; böylece yalnızca Cc/Bcc teslimi mümkün kalır. Gönderim
anında To, Cc ve Bcc arasında en az bir alıcı olmalıdır.

Zarf RCPT sırası To, sonra Cc, sonra Bcc'dir. Yinelenen girdiler korunur.

`From` tam olarak bir postadır ve hem MAIL FROM hem From başlığı için
kullanılır. Kimliği doğrulanmış kullanıcı adı From olarak çıkarılmaz.

Bcc alıcıları zarf RCPT alıcıları olarak gönderilir. DATA'da Bcc başlığı
görünmez.

Konu UTF-8'dir ve gerektiğinde RFC 2047 encoded-word kullanır. Konu, From,
To, Cc, Bcc veya Reply-To içindeki CR/LF `SMTPError` yükseltir.

En az bir gövde `withText` veya `withHtml` ile yapılandırılmış olmalıdır.

- yalnızca metin: `text/plain; charset=utf-8`
- yalnızca HTML: `text/html; charset=utf-8`
- ikisi: `multipart/alternative` (önce text/plain, sonra text/html)

UTF-8 gövdeler quoted-printable kullanır. SMTP HTML'i kaçışlamaz veya
temizlemez. Verilen HTML kasıtlı posta işaretlemesidir.

Date gönderim anında üretilir. v0.9'da genel Date veya Message-ID API'si
yoktur. Message-ID, zayıf bir kimlik uydurmak yerine atlanır.

Ekler kapsam dışıdır.

## Komut akışı

bağlan → selamlama → EHLO → güvenlik kipine göre TLS → yapılandırıldıysa AUTH
→ MAIL FROM → To/Cc/Bcc için RCPT TO → DATA → ileti → QUIT → kapat.

Reddedilen bir SMTP komutu `SMTPError`'dur. DATA öncesi herhangi bir RCPT
reddedilirse gönderim durur. Kısmi alıcı teslimi ve sessiz yeniden deneme
yoktur.

Her gönderim yolu bağlantıyı kapatır. İki eşzamanlı `send` bağımsız
bağlantılar kullanır.

## Kapsam dışı

IMAP, POP3, ekler, AUTH LOGIN / CRAM-MD5 / XOAUTH2, DKIM/SPF/DMARC, sağlayıcı
modülleri, kuyruklar, yeniden denemeler, şablonlar ve izleme v0.9'un parçası
değildir.
