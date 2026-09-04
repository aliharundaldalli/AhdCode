# Modüller

[English](MODULES.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Math](MATH_TR.md) · [File ve Path](FILESYSTEM_TR.md)

Yerel bir modül, kardeş (sibling) bir `.ahd` dosyasıdır. Referans, büyük/küçük
harfe duyarlı tek bir tanımlayıcıdır (identifier): `Utilities`,
içe aktaran dosyanın yanındaki `Utilities.ahd` dosyasına karşılık gelir. v0.1,
noktalı yollara (dotted paths), paket-kökü aramasına (package-root search)
veya yapılandırılabilir bir modül yoluna sahip değildir.

İsim uzayı (namespace) içe aktarımı:

```ahd
bring Utilities
write(Utilities.greet("Ali"))
```

Doğrudan içe aktarım:

```ahd
from Utilities bring greet
write(greet("Ali"))
```

Seçici, çok satırlı içe aktarım:

```ahd
from Utilities bring (
    greet
    farewell
)
```

Herkese açık-hepsi (public-all) içe aktarım:

```ahd
from Utilities bring all
```

`all`, yalnızca herkese açık, `Confidential` olmayan sembolleri getirir. İçe
aktarım çakışmaları (import collisions) ve döngüsel bağımlılıklar (circular
dependencies) derleme zamanı hatalarıdır.

`Web` bunların ikisinden de farklıdır. O bir **gömülü birinci taraf
modüldür**: kaynağı AhdCode'dur, derleyiciye gömülüdür, sizin dosyalarınızla
aynı geçişte derlenir ve tek bir kendi kendine yeten çalıştırılabilir dosyaya
üretilir. Çevrimdışı çözülür -- kayıt defteri, manifest, kilit dosyası veya
indirme yoktur -- ve yerel bir `Web.ahd` onu gölgeleyemez; tıpkı yerel bir
`HTTP.ahd`'nin `HTTP`'yi gölgeleyememesi gibi. Bkz. [Web rehberi](WEB_TR.md).

```ahd
bring Web
from Web bring (Request, Response, HTMLNode)
```

`Web`, bileştirdiği `HTTP` ve `HTML` türlerini yeniden dışa aktarır; bunlar
kopya değil, aynı türlerdir: `Web` üzerinden ulaşılan bir `Request` çıplak bir
`HTTP.Server` üzerine değişmeden kaydolur. Yalnızca `Web` geneldir; çatının iç
modüllerine yalnızca çatı kaynağından erişilebilir.

`Math`, `Time`, `Latex`, `Word`, `Excel`, `PDF`, `Archive`, `Path`, `File`, `Regex`, `CSV`, `Data`, `Statistics`, `Plot`, `Numeric`, `JSON`, `SQLite`, `HTTP`, `HTML`, `SMTP`, `XML`, `Env`, `Lists` ve `KeyValue` derleyici tarafından
kayıtlıdır (compiler-registered) ve aynı içe aktarım biçimlerini kullanır.
Yerel bir dosya, aynı isimdeki standart bir modülün yerini alamaz (shadow
edemez). `HTTP` hem gelen sunucu (`Server` / `Request` / `Response`, çerezler,
oturumlar) hem de giden `Client` / `ClientRequest` / `ClientResponse`
yüzeyidir. `SMTP` yalnızca gönderim yapan postadır (`SMTPClient` /
`SMTPMessage`). Ayrıca sıradan isim uzayı takma adı (namespace alias) biçimini de
kullanabilirler:

```ahd
bring File as F
F.writeText("note.txt", "hello")
```

Tipli yüzeyleri ve yakalanabilir alan hataları için [Time](TIME_TR.md),
[CSV](CSV_TR.md), [Data](DATA_TR.md), [Statistics](STATISTICS_TR.md), [Plot](PLOT_TR.md), [Numeric](NUMERIC_TR.md), [Word](WORD_TR.md), [Excel](EXCEL_TR.md), [PDF](PDF_TR.md), [Archive](ARCHIVE_TR.md), [JSON](JSON_TR.md), [SQLite](SQLITE_TR.md), [HTTP](HTTP_TR.md), [HTML](HTML_TR.md), [SMTP](SMTP_TR.md), [XML](XML_TR.md), [Env](ENV_TR.md), [Lists](LISTS_TR.md), [KeyValue](KEYVALUE_TR.md) ve diğer modül referanslarına bakın.

CSV, Data, Plot, Excel, Word, Latex, HTTP(S) ve HTML'i tek tek API olarak
değil, birbirine bağlanan öğrenci projeleri içinde öğrenmek için
[Uygulamalı Modül Atölyeleri](PRACTICAL_MODULES_TR.md) belgesini kullanın.

`Lists` ve `KeyValue`, çekirdek `List` ve `Pair` türleri üzerindeki yapısal
dönüşüm katmanıdır. İşlemleri *tür-yönelimlidir*: derleyici her çağrının kesin
sonuç türünü o çağrı yerinde yazılan argüman türlerinden hesaplar; böylece
`Lists.chunk(List<Int>, 2)` `List<List<Int>>`, `Lists.chunk(List<String>, 2)`
ise `List<List<String>>` olur — dilde genel (generic) sözdizimi olmadan ve
hiçbir şey silinmeden. Bir çağrı kendi argümanlarına göre özelleştirildiği
için böyle bir işlemin özelleştirilmemiş bir `Function` değeri yoktur; birini
almak derleme zamanı tanısıdır.
