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
