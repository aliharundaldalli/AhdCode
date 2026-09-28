# Dosya sistemi dağıtım temel işlemleri (v2.5.0)

[English](README.md) · [Türkçe]

Bir sürümü hazırlar, bir ikili dosyayı üzerine yazmadan kopyalar, bir yapılandırma dosyasını atomik olarak değiştirir, izinleri sekizlik bir String (`"0750"`) ile ayarlar, hazırlık dizinini tek bir yeniden adlandırmayla `releases/42` yapar, sürümü bağlantıları izlemeden gezer ve bir `current` sembolik bağlantısını atomik olarak değiştirir. Unix sembolik bağlantıları ve izinlerini kullanır. Bkz. [File](../../../docs/FILESYSTEM_TR.md).

Her çalıştırma geçerli dizin altında yeni bir `run-<uuid>/` klasöründe
çalışır; bu yüzden tekrarlanabilir. İşiniz bitince bu klasörleri silin.

```bash
ahdcode run main.ahd
```
