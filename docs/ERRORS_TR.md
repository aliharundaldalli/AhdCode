# Hatalar

[English](ERRORS.md) · [Türkçe]

[README'ye dön](../README_TR.md)

AhdCode hataları, `Error`'dan türeyen yakalanabilir (catchable) Class
değerleridir.

```ahd
attempt {
    toss(DomainError("invalid value"))
}
except DomainError as error {
    write(error.message)
}
ultimately {
    write("finished")
}
```

- `attempt` korunan kodu çalıştırır.
- `except ErrorType as error` eşleşen bir hatayı yakalar.
- `ultimately`, bekleyen bir `return` tamamlanmadan önce dahil, her zaman
  çalışır.
- `toss` bir Error örneği (instance) fırlatır.

Yaygın yerleşik (built-in) hatalar şunları içerir:

| Hata | Tipik neden |
|---|---|
| `DivisionByZeroError` | sıfıra bölme veya sıfırla mod alma |
| `OverflowError` | denetimli (checked) Int veya sonlu Real taşması |
| `DomainError` | türü geçerli ama matematiksel/arama alanı (domain) geçersiz |
| `IndexError` | geçersiz List/String indeksi |
| `IOError` | girdi/çıktı hatalarının temel (base) sınıfı |
| `FileError` | `File` modülü işlem hatası; `IOError`'dan türer |
| `RegexError` | `Regex.compile`'a geçersiz bir desen (pattern); `Error`'dan türer |
| `CSVError` | bozuk CSV, geçersiz ayraç veya geçersiz kayıt/başlık şekli; `Error`'dan türer |
| `UUIDError` | bozuk UUID metni ya da yeni bir UUID için rastgele kaynak veya geçerli saat olmaması; `Error`'dan türer |
| `PostgreSQLError` | PostgreSQL bağlantı, sorgu, çalıştırma, işlem ya da değer türü hatası; `Error`'dan türer |
| `ArchiveError` | güvensiz, bozuk, desteklenmeyen ya da sınırı aşan arşiv veya bir arşiv oluşturma/çıkarma hatası; `Error`'dan türer |
| `ProcessError` | `Process.run` programı başlatamadı, zaman aşımına uğradı ya da çıktı bütçesini aştı (sıfır olmayan çıkış kodu hata değil, sonuçtur); `Error`'dan türer |
| `DNSError` | `DNS.lookup`'ta geçersiz ana makine adı, bulunamayan ana makine, çözümleyici hatası ya da zaman aşımı; `Error`'dan türer |
| `TLSError` | `TLS.inspect` bir sertifika elde edemedi (geçersiz girdi, reddedilen bağlantı, zaman aşımı, TLS değil); geçersiz bir sertifika hata değil, sonuçtur; `Error`'dan türer |
| `DiskError` | `Disk.inspect` bir dosya sistemini okuyamadı (boş ya da eksik yol, izin reddi, bilgi alınamıyor); `Error`'dan türer |
| `ServiceError` | `Service.status` başarısız oldu (geçersiz ad, servis bulunamadı, systemd yok, izin reddi, zaman aşımı, desteklenmeyen platform); `Error`'dan türer |
| `KeyError` | eksik Pair anahtarı |
| `NullError` | çalışma zamanı null güvenliği sınırı |
| `ConstantError` | derin dondurulmuş (deep-frozen) bir referans üzerinden değişiklik |
| `ValueError` | negatif String tekrarı gibi geçersiz bir çalışma zamanı değeri |

Özel hatalar sıradan kalıtımı (inheritance) kullanır:

```ahd
InvalidAgeError: Class<Error> := {
    structure: Attributes := (
        message: String
    )
}
```
