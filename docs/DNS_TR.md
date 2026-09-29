# DNS standart modülü

[English](DNS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [TLS](TLS_TR.md) · [Hatalar](ERRORS_TR.md)

`DNS` (v2.6.0) tek bir soruyu yanıtlar: **bu ana makine adı şu anda hangi IP
adreslerine çözülüyor?** İşletim sisteminin çözümleyicisini kullanır, her zaman
bir zaman aşımıyla sınırlıdır ve asla bir kabuk ya da harici araç çalıştırmaz
(`dig` yok, `nslookup` yok). Açıkça içe aktarın:

```ahd
bring DNS
from DNS bring (DNSResult, DNSError)
```

Kanonik modül kimliği `builtin:DNS`'tir; bir kardeş `DNS.ahd` onu gölgeleyemez.
(v2.6.0'dan itibaren `DNS` bir standart modül adıdır: yerel bir `DNS.ahd` artık
`bring DNS`'in yüklediği şey değildir — böyle bir dosyayı yeniden adlandırın.)

## Yüzey

```text
DNS.lookup(host: String, timeoutSeconds: Int = 5) -> DNSResult

DNSResult.host()      -> String         // sorgulanan, normalleştirilmiş ana makine adı
DNSResult.addresses() -> List<String>   // tüm adresler, önce IPv4, sonra IPv6
DNSResult.ipv4()      -> List<String>
DNSResult.ipv6()      -> List<String>

DNSError  (Error'dan türer)
```

`DNSResult` yalnızca `DNS.lookup` tarafından üretilen opak bir değerdir.

```ahd
found: DNSResult := DNS.lookup(host: "example.com", timeoutSeconds: 5)
write(found.host())
write(found.addresses())
```

## Yalnızca adres sorgusu

v2.6.0 adresleri (bir bağlantının kullanacağı A ve AAAA kayıtlarını) çözer. MX,
TXT, NS, SRV ya da CNAME kayıtlarını sorgulamaz, DNS sunucusu seçmez ve DNS
verisini asla değiştirmez. Sonuçlar işletim sisteminin çözümleyicisinden gelir;
böylece makinenin kendi yapılandırmasına (hosts dosyası, arama alanları, üst
sunucular) uyar. Çözümleyicinin iç işleyişi macOS, Linux ve Windows arasında
farklıdır; genel sonuç biçimi farklı değildir.

## Sonuç anlamı

- Adresler **tekilleştirilir** ve **sıralanır**: önce tüm IPv4 adresleri, sonra
  tüm IPv6 adresleri, her aile artan sayısal sırada. Aynı yanıt her zaman aynı
  biçimde yazdırılır; REPL'de de derlenmiş programda da.
- Adresler kanonik metin kullanır (`192.0.2.10`, `2001:db8::1`); IPv4'e eşlenmiş
  bir IPv6 yanıtı düz IPv4 olarak bildirilir.
- `host()` normalleştirilmiş addır: küçük harf, sondaki nokta olmadan
  (`Panel.Example.COM.` → `panel.example.com`).
- Hiçbir adrese çözülmeyen bir ad, asla boş bir başarı değil, bir `DNSError`'dır
  (`no addresses were found for the host`).
- Bir **IP adresi** ağa hiç çıkmadan kendisi olarak döndürülür:
  `DNS.lookup("127.0.0.1")` `["127.0.0.1"]`, `DNS.lookup("[::1]")` ise `["::1"]`
  verir.

## Girdi doğrulama

Ana makine adı, herhangi bir ağ işinden önce denetlenir. `DNS.lookup` bir ana
makine adı ya da bir IP adresi alır — başka hiçbir şey değil:

| Girdi | Sonuç |
|---|---|
| `""`, `"   "` | `the host is empty` / `the host is only whitespace` |
| `"https://example.com/path"` | `the host must be a host name such as example.com, not a URL` |
| `"example.com/path"`, `"example.com?x"` | `the host must not contain a path, query, or fragment` |
| `"example.com:443"` | `the host must not contain a port; pass the port separately` |
| `"user@example.com"` | `the host must not contain user information` |
| boşluk, sekme, NUL, diğer denetim karakterleri | reddedilir |
| `;`, `&`, `$`, `` ` `` ve harf, rakam, `-`, `_`, `.` dışındaki her karakter | reddedilir |
| ASCII olmayan adlar | reddedilir — uluslararası bir adın ASCII (`xn--`) biçimini verin |
| 63 karakterden uzun etiketler, 253 karakterden uzun adlar, boş etiketler, `-` ile başlayan ya da biten etiketler | reddedilir |

Hiçbir yerde kabuk yoktur; bu yüzden `example.com;touch X` ya da `$(whoami)`
gibi bir metin yalnızca geçersiz bir ana makine adıdır.

## Zaman aşımı

Her sorgu sınırlıdır: `timeoutSeconds` varsayılan olarak **5**'tir ve **1 ile
60** arasında olmalıdır. Zamanında bitmeyen bir sorgu
`timed out after N seconds (timeoutSeconds)` iletisiyle `DNSError` fırlatır.

## Hatalar

Her hata, iletisi `lookup "<ana makine>" failed: <neden>` biçiminde olan bir
`DNSError` fırlatır:

```text
lookup "missing.example" failed: host not found
lookup "slow.example" failed: timed out after 5 seconds (timeoutSeconds)
lookup "broken.example" failed: the resolver could not answer (temporary failure)
lookup "empty.example" failed: no addresses were found for the host
lookup "example.com:443" failed: the host must not contain a port; pass the port separately
```

İletiler makinenin çözümleyici yapılandırmasını (yanıt veren DNS sunucusunun
adresi gibi) asla içermez.

```ahd
attempt {
    found: Local DNSResult := DNS.lookup("panel.example.com")
    if not ("203.0.113.10" in found.ipv4()) {
        write("panel.example.com does not point at this server")
    }
}
except DNSError as failure {
    write(failure.message)
}
```

## Güvenlik: güvenilmeyen girdi için yalnızca işçi

`DNS.lookup`, programın kendisine verilen her ad için ağ sorgusu göndermesine
yol açar. **Güvenilmeyen bir web isteğini doğrudan ona vermeyin.** Bir kontrol
paneli için önerilen yapı:

```text
web uygulaması (ayrıcalıksız)
    -> kimliği doğrulanmış kontrol kanalı
        -> özel bir AhdCode işçisi
            -> doğrulanmış / izin listesindeki alan adı
                -> DNS ve TLS incelemesi
```

Hangi alan adlarının incelenebileceğine işçi karar verir (örneğin yalnızca
isteği yapan hesaba kayıtlı alan adları). AhdCode'un kendine ait genel bir izin
listesi yoktur: bu politika uygulamaya aittir.

## Bu sürümde yok

Diğer kayıt türleri (MX, TXT, NS, SRV, CNAME), DNS sunucusu seçimi, DNSSEC
doğrulaması, ters sorgular, bölge aktarımları ve her türlü DNS değişikliği
(kayıtlar, bölgeler ya da Cloudflare gibi sağlayıcı API'leri) `DNS`'in parçası
değildir.
