# REPL

[English](REPL.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [CLI](CLI_TR.md) · [File ve Path](FILESYSTEM_TR.md)

Kalıcı (persistent) bir etkileşimli oturum başlatın:

```bash
ahdcode
```

Başlangıçta `ahdcode --version` ile eşleşen bir sürüm başlığı (şu anda
`AhdCode v0.9.0` yazdırılır, ardından `ahd>` istemi gelir:

```text
ahd> x := 5
ahd> x = x + 1
ahd> x
6
ahd> age := int(take("Age: "))
Age: 26
ahd> age + 1
27
ahd> square := lambda (x: Int) -> x^2
ahd> square(5)
25
```

Normal sözcüksel çözümleyici (lexer), ayrıştırıcı (parser), semantik
denetleyici ve tiplenmiş/düşürülmüş (lowered) IR, dosya derlemesiyle
paylaşılır. REPL, yeni doğrulanan (validated) IR'yi tek bir kalıcı
değerlendiricide (evaluator) çalıştırır. Başarılı kaynak geçmişini yeniden
çalıştırmaz, bu yüzden çıktı, girdi, değişiklik (mutation), Class oluşturma,
modül başlatma ve dosya işlemleri tam olarak bir kez gerçekleşir.

Değerler ve değiştirilebilir bağlamalar (mutable bindings) kalıcıdır.
List/Pair/Class nesneleri kimliğini korur, bu yüzden alias'lar sonraki
değişikliği görür. İsimlendirilmiş Function'lar, Class'lar, içe aktarımlar
ve paylaşılan Math RNG durumu da oturumda kalır:

```text
ahd> a := [1, 2]
ahd> b := a
ahd> a.add(3)
ahd> b
[1, 2, 3]
ahd> bring Math
ahd> Math.seed(42)
ahd> Math.random()
0.7415648787718233
ahd> Math.random()
...
```

Lambda Function değerleri de komutlar arasında kalır ve örneğin
`values.map(lambda (x: Int) -> x^2)` biçiminde doğrudan callback olarak
çalışır. Diğer yerlerde olduğu gibi yalnızca ifade içerir ve dış değişkenleri yakalamak için açıkça `#` veya `@` belirtimine ihtiyaç duyar.

`take`, isteğini (prompt) yazdırır ve temizler (flush), ardından gerçek
terminalden tam olarak bir cevap satırı tüketir. O cevap, başka bir REPL
komutu olarak ele alınmaz.

Yerel modüller, `ahdcode`'un başlatıldığı dizinden çözülür. Aynı dizin,
göreli File yolları için de temel (base) alınır:

```text
ahd> bring Engine
ahd> Engine.tick()
ahd> bring File
ahd> File.writeText("note.txt", "hello")
ahd> File.readText("note.txt")
hello
```

Çok satırlı Function'lar, Class'lar, bloklar ve ifadeler `...>` devam
istemini (continuation prompt) kullanır. Sıradan bildirim kuralları devam
eder: aynı kapsamda `x`'i yeniden bildirmek bir hatadır; değişiklik için
`=` kullanılır. Başarısız bir semantik gönderim (submission) veya
yakalanmamış bir AhdCode Error, REPL'i sonlandırmaz veya önceki başarıyla
işlenmiş kaynak bağlamını (context) silmez.

## REPL'de SQLite

`SQLite` kalıcı oturumda çalışır. Bellekteki bir `Database`, birbirini izleyen
başarılı girdilerde durur; başarısız bir SQL komutu `SQLiteError` bildirir
ve oturumu bozmaz:

```text
ahd> bring SQLite
ahd> db := SQLite.open(":memory:")
ahd> db.execute("CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)")
0
```

Göreli dosya yolları, REPL'in başlatıldığı dizine göre çözülür. Bkz.
[SQLite](SQLITE_TR.md).

## REPL'de Latex ve PDF

`Latex`'in markup oluşturma yardımcıları (`document`, `table`, `section`,
`escape` ve diğerleri) ve `PDF`'in belge oluşturma işlemleri (`PDF.new()`,
`heading`, `paragraph`, `table`, `image`, `pageBreak`, `PDF.fromWord`,
`PDF.fromExcel`) REPL'de normal şekilde çalışır — bunlar yalnızca bellekte
değer oluşturur. Gerçekten bir PDF derlemek, çevrimdışı Tectonic render
motorunu harici bir işlem olarak çağırır; kalıcı evaluator bunu desteklemez:
`Latex.pdf(...)`, `Latex.pdfFile(...)` ve `PDFDocument.save(...)`,
etkileşimli olarak çağrıldığında bir hata fırlatır (`LatexError`/`PDFError`).
Bunları bir `.ahd` dosyasından `ahdcode run` veya `ahdcode build` ile
çalıştırın. `Archive`'ın böyle bir sınırlaması yoktur — `Archive.zip`/`tar`/
`tarGzip` REPL'de tamamen çalışır, çünkü arşivleme yalnızca Go standart
kütüphanesini kullanır.
