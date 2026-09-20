# Ekli posta

[English](README.md) · [Türkçe]

Bu örnek yerel bir SMTP sunucusu ister. Gerçek bir posta kutusunu tahmin
etmez ve hiçbir kimlik bilgisi içermez: ihtiyacı olan her şeyi ortamdan alır.

Yakalama yapan herhangi bir yerel sunucu iş görür. Python'ın yerleşik
sunucusuyla:

```bash
python3 -m smtpd -c DebuggingServer -n localhost:2525
```

Ardından başka bir terminalde:

```bash
export SMTP_HOST=127.0.0.1
export SMTP_PORT=2525
export SMTP_SECURITY=none
export SMTP_FROM=sender@example.com
export SMTP_TO=student@example.com
ahdcode run send.ahd
```

Yakalanan ileti bir `multipart/mixed`'dir: metin ve HTML gövdelerini taşıyan
bir `multipart/alternative`, ardından base64 ile kodlanmış iki ek.

## Kimlik bilgileri

`send.ahd`, `SMTP_USER` ve `SMTP_PASSWORD` değişkenlerini okur ve yalnızca
ikisi de ayarlıysa kimlik doğrular. `SMTP_PASSWORD`, `Env.secret` ile
okunur; böylece bir konteyner platformunun gizli dosyası da ortam değişkeni
kadar iyi çalışır — bkz. [Env](../../../docs/ENV_TR.md).

AhdCode şifrelenmemiş bir bağlantıda kimlik doğrulamayı reddeder; bu yüzden
gerçek bir sunucu için `SMTP_SECURITY=starttls` ya da `tls` gerekir. Bir
parolayı asla kaynak dosyaya yazmayın.

## Üretilen dosyalar

`summary.txt` ve `chart.png` programın yazdığı dosyalardır. Üretilen
dosyalardır; örneğin parçası değildir.
