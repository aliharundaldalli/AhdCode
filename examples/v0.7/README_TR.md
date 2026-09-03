# AhdCode v0.7 örnekleri

[English](README.md) · [Türkçe]

[Proje README'sine dön](../../README.md)

Bu programlar v0.7.0 HTML ayrıştırmayı tanıtır: `HTML.parse`, bir HTML
String'ini salt okunur bir `HTMLDocument`'e dönüştürür; `select`/`first` de
bunun içinde küçük bir CSS benzeri seçici diliyle (etiket, `#id`, `.class`,
`[attr]`, `[attr="değer"]`, soy/çocuk birleştiricileri, virgüllü listeler)
`HTMLElement` değerleri bulur. Ayrıştırma asla bir ağ isteği yapmaz ve asla
betik içeriği çalıştırmaz -- yalnızca kendisine verdiğiniz String'i okur.
Bir scraper modülü, tarayıcı veya JavaScript motoru yoktur.

```bash
ahdcode run examples/v0.7/01_parse_html.ahd
```

| Örnek | Konu |
|---|---|
| `01_parse_html.ahd` | Sabit bir HTML String'ini ayrıştırmak; `select`, `first`, `text()`, `attr()` |
| `02_http_scrape.ahd` | HTTP Client `get` + `HTML.parse`, iki bağımsız adım |
| `03_scrape_to_sqlite.ahd` | `HTML` ile çıkarmak, `SQLite` ile saklamak, bağlı parametreler |

Örnek 02 internet erişimi gerektirir. Varsayılan olarak küçük, kararlı, statik
bir sayfa olan `https://example.com/` kullanır; kendi kontrolünüzdeki bir
sayfayı işaret etmek için `SCRAPE_URL` ayarlayın. Örnek 03,
`SQLite.open(":memory:")` açar; böylece çalıştırmak geride bir veritabanı
dosyası bırakmaz -- gerçek bir dosyaya kalıcı kaydetmek için
[`examples/v0.3`](../v0.3/README_TR.md) örneğine bakın.
