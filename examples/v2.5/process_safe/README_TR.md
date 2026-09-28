# Kabuksuz süreç çalıştırma (v2.5.0)

[English](README.md) · [Türkçe]

Kabuk sözdizimi içeren argümanlarla `/bin/echo` çalıştırır ve bunların düz metin olarak kaldığını gösterir, sıfır olmayan bir çıkış kodunu ve stderr'i sıradan bir sonuç olarak okur, eksik bir programı ve bir zaman aşımını `ProcessError` olarak yakalar. Unix programları kullanır; Windows'ta mutlak yolla gerçek bir `.exe` kullanın (toplu iş dosyaları reddedilir). Bkz. [Process](../../../docs/PROCESS_TR.md).

Her çalıştırma geçerli dizin altında yeni bir `run-<uuid>/` klasöründe
çalışır; bu yüzden tekrarlanabilir. İşiniz bitince bu klasörleri silin.

```bash
ahdcode run main.ahd
```
