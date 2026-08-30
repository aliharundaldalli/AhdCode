# String API

[English](STRING_API.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Temel İşlevler](FUNDAMENTALS_TR.md)

String değiştirilemezdir (immutable) ve UTF-8 byte'ına göre değil, Unicode
karakterine göre indekslenir. Aşağıdaki her işlem, null olmayan bir alıcı
(receiver) ve null olmayan argümanlar gerektirir.

| İşlem | Sonuç |
|---|---|
| `trim()` | baştaki/sondaki Unicode boşluklarını kaldırır |
| `lower()` | yerelden bağımsız (locale-independent) Unicode küçük harf |
| `upper()` | yerelden bağımsız Unicode büyük harf |
| `capitalize()` | yalnızca ilk karakteri büyütür |
| `split(separator)` | boş alanları koruyan bir `List<String>` |
| `replace(old, new)` | çakışmayan (non-overlapping) tüm eşleşmeleri değiştirir |
| `contains(text)` | alt metin (substring) üyeliği |
| `startsWith(prefix)` | önek (prefix) testi |
| `endsWith(suffix)` | sonek (suffix) testi |
| `count(text)` | çakışmayan geçiş sayısı |
| `index(text)` | ilk karakter indeksi |

```ahd
text: String := "  Ali,Veli  "
write(text.trim().lower().split(","))
write("a✓b✓".index("✓"))
write("ali HARUN".capitalize())
```

`capitalize` geri kalanı tam olarak korur (`"aHD"` → `"AHD"` olur).
`split("")`, `count("")` ve `index("")` `DomainError` fırlatır. Bulunamayan
bir `index` araması da `DomainError` fırlatır; `-1` gibi bir gösterge
(sentinel) değeri yoktur.

Bu işlemler isimlendirilmiş parametre yayınlamaz ve v0.1, `strip`,
`toLowerCase` veya parametresiz bir `split` gibi takma adlar (alias)
tanımlamaz.
