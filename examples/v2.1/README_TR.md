# Uygulama G/Ç ve ağ (v2.1.0)

[English](README.md) · [Türkçe]

v2.1'in eklediklerini gösteren, kendi kendine yeten üç program: ikili
dosyaları [HTTP](../../docs/HTTP_TR.md) üzerinden iki yönde taşımak,
AhdCode'dan bir [WebSocket](../../docs/WEBSOCKET_TR.md) servisiyle
konuşmak ve dosya ekleyerek [posta](../../docs/SMTP_TR.md) göndermek.

Hiçbiri genel internete ihtiyaç duymaz. Her örnek, kendi başlattığınız
`127.0.0.1` üzerindeki bir servise karşı çalışır.

Bu örnekler AhdCode v2.1.0 sürümüyle yayımlanır.

## http_file_transfer

İki program. Önce servisi başlatın:

```bash
cd http_file_transfer
ahdcode run server.ahd
```

ve başka bir terminalde:

```bash
cd http_file_transfer
ahdcode run client.ahd
```

`client.ahd` bir QR kodu PNG'si üretir, bunu `withMultipartFile` ile bir
`multipart/form-data` dosya parçası olarak yükler ve saklanan dosyayı
`download` ile geri indirir. PNG hiçbir yönde AhdCode String'ine dönüşmez:
yükleme diskten doğrudan isteğe akıtılır, yanıt gövdesi de doğrudan
`returned.png` dosyasına yazılır.

`server.ahd` değişmemiş v0.8 yükleme API'si ve v0.9 dosya yanıtıdır.
Aldıklarını `received/` klasörüne yazar.

Servisi `ahdcode kill server.run` ile durdurun. `qr.png`, `returned.png` ve
`received/` üretilen dosyalardır; örneğin parçası değildir.

## websocket_client

Yine iki program:

```bash
cd websocket_client
ahdcode run server.ahd
```

```bash
cd websocket_client
ahdcode run client.ahd
```

`server.ahd` v1.4 WebSocket uç noktasıdır: el sıkışmada bir belirteç ister,
her bağlantıyı selamlar ve her iletiye JSON ile yanıt verir.

`client.ahd` yenidir. Bir `WebSocketClient`'ı o belirteçle yapılandırır,
bağlanır, üç soru gönderir, her yanıtı eşzamanlı bir `receive` ile okur,
sunucudan kapatmasını ister ve kapanış kodunu ve nedenini bildirir. Tarayıcı
gerekmez ve hiçbir şey kendiliğinden yeniden bağlanmaz.

## smtp_attachment

```bash
cd smtp_attachment
ahdcode run send.ahd
```

Metin gövdesi, HTML gövdesi ve iki ek — bir metin dosyası ve bir PNG —
içeren tek bir ileti gönderir. Önce [kendi README'sini](smtp_attachment/README_TR.md)
okuyun: yerel bir SMTP sunucusu ister ve bilinçli olarak hiçbir kimlik
bilgisi içermez.
