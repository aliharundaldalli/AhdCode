# SQLite standart modülü

[English](SQLITE.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [JSON](JSON_TR.md) · [Öğrenci Rehberi](STUDENT_GUIDE_TR.md#35-sqlite-hatırlayan-bir-veritabanı)

`SQLite`, AhdCode v0.3.0 ile gelen, derleyici tarafından kayıtlı
`builtin:SQLite` modülüdür. Açıktır ve kardeş bir `SQLite.ahd` dosyası onu
gölgeleyemez:

```ahd
bring SQLite
from SQLite bring Database
from SQLite bring SQLiteValue
from SQLite bring SQLiteError
```

`SQLite`, gerçek bir yerel SQLite veritabanına güvenli ve tipli bir köprüdür.
Siz sıradan SQL yazarsınız; AhdCode parametre bağlamayı ve tipli değer
dönüşümünü yapar, başka hiçbir şey yapmaz. ORM yok, sorgu oluşturucu yok,
şema çıkarımı yok, migration çatısı yok ve `Any` yok: SQL'den AhdCode'a geçen
her değer, türünü açıkça okuduğunuz bir `SQLiteValue`'dur.

## Genel yüzey

```text
SQLite.open(path: String)         -> Database
SQLite.nullValue()                -> SQLiteValue
SQLite.fromInt(value: Int)        -> SQLiteValue
SQLite.fromReal(value: Real)      -> SQLiteValue
SQLite.fromString(value: String)  -> SQLiteValue

Database.execute(sql: String, parameters: List<SQLiteValue> = []) -> Int
Database.query(sql: String, parameters: List<SQLiteValue> = [])   -> List<Pair<String, SQLiteValue>>
Database.lastInsertId()                                           -> Int
Database.begin()                                                  -> Nothing
Database.commit()                                                 -> Nothing
Database.rollback()                                               -> Nothing
Database.close()                                                  -> Nothing

SQLiteValue.kind()    -> String     // "Null", "Int", "Real" veya "String"
SQLiteValue.isNull()  -> Bool
SQLiteValue.int()     -> Int
SQLiteValue.real()    -> Real
SQLiteValue.string()  -> String

SQLiteError  (Error'dan türer)
```

`Database` ve `SQLiteValue` opak yerleşik Class'lardır: `Database()` ya da
`SQLiteValue()` ile kurulamazlar, genel özellikleri yoktur ve yalnızca
yukarıdaki fonksiyonlardan elde edilirler. Tüm argümanlar konumsaldır.
`SQLite.fromReal` için `Int`, tıpkı sıradan bir `x: Real := 3` atamasındaki
gibi `Real`'e genişler.

## Veritabanı açma

```ahd
db: Database := SQLite.open("notlar.db")
bellek: Database := SQLite.open(":memory:")
```

`path` bir dosya sistemi yoludur ya da `Database` kapandığında veya program
bittiğinde yok olan özel bir bellek içi veritabanı için tam olarak
`":memory:"` işaretidir. Sıradan dosya davranışı SQLite'ı izler:

- eksik veritabanı dosyası oluşturulur;
- üst dizinler **oluşturulmaz**; `veri/` yokken `veri/uygulama.db` açmak
  `SQLiteError` fırlatır;
- göreli yol programın çalışma dizinine göre çözülür (REPL'de, REPL'in
  başlatıldığı dizin);
- yollar boşluk ve ASCII dışı karakter içerebilir.

Yol asla URI ya da DSN olarak yorumlanmaz; sorgu dizesi sözdizimi yoktur.
AhdCode bir veritabanı dizini icat etmez ve adını vermediğiniz hiçbir dosyayı
yazmaz.

## Bildirilen tipler değil, saklama sınıfları

SQLite her bir değeri beş çalışma zamanı saklama sınıfından biriyle saklar.
AhdCode, bildirilen sütun tipini değil, **her değerin saklama sınıfını**
eşler:

| SQLite saklama sınıfı | `SQLiteValue.kind()` | Okuma yolu            |
| --------------------- | -------------------- | --------------------- |
| `NULL`                | `"Null"`             | `isNull()`            |
| `INTEGER`             | `"Int"`              | `int()` veya `real()` |
| `REAL`                | `"Real"`             | `real()`              |
| `TEXT`                | `"String"`           | `string()`            |
| `BLOB`                | — desteklenmez —     | `SQLiteError` fırlatır |

Beklemeniz gereken sonuçlar:

- `BOOLEAN` olarak bildirilen bir sütun `INTEGER` `0`/`1` değerleri tutar;
  bunları `int()` ile okursunuz. Bool çıkarımı yoktur.
- `DATE` ya da `DATETIME` olarak bildirilen ve `'2026-09-02'` tutan bir sütun
  `TEXT`'tir; `string()` ile okur ve uygulamanın gerektirdiği yerde kendiniz
  ayrıştırırsınız. Tarih çıkarımı yoktur.
- `REAL` olarak bildirilen bir sütun bağlanan `Int` `7`'yi `REAL` `7.0` olarak
  saklar; bu yüzden `kind()` `"Real"` bildirir. Bu SQLite'ın kendi tip
  yakınlığıdır.
- Tipsiz bildirilen bir sütun (ya da `'12'` tutan bir `TEXT`), eklenen değerin
  saklama sınıfını korur. `'12'` `"String"` kalır; üzerinde `int()` çağırmak
  ayrıştırmak yerine `SQLiteError` fırlatır.
- `real()`, `Real` türünü ve (`Real := Int` gibi genişletilerek) `Int` türünü
  kabul eder. `int()` yalnızca `Int`, `string()` yalnızca `String` kabul eder.
  `Null` hiçbir zaman sayı ya da metin olarak okunamaz.
- SQLite'ın sonsuz ya da NaN olarak hesapladığı bir `REAL` (örneğin
  `1e308 * 10`) AhdCode `Real`'i olamaz ve `SQLiteError` fırlatır.
- Bir `BLOB` değerini sorgulamak sütunu adlandıran bir `SQLiteError` fırlatır.
  Baytlar asla sessizce metne ya da base64'e çevrilmez; aynı tablonun diğer
  sütunları okunabilir kalır.

SQL `NULL`, AhdCode `null`'ı **değil**, `Null` türünde bir `SQLiteValue`'dur.
Sorgu satırı yapısal olarak her zaman `Pair<String, SQLiteValue>`'dur ve
dilin nullable sistemi işin içine girmez.

## Parametreler: gerçek bağlama, asla metin birleştirme değil

```ahd
db.execute(
    "INSERT INTO students (name, score) VALUES (?, ?)",
    [
        SQLite.fromString("Ayşe")
        SQLite.fromReal(91.5)
    ]
)

satirlar := db.query(
    "SELECT id, name, score FROM students WHERE score >= ? ORDER BY id",
    [SQLite.fromReal(80.0)]
)
```

Desteklenen genel stil konumsal `?` yer tutucularıdır. Her `?`, SQLite'ın
kendi parametre bağlama API'siyle aynı konumdaki `SQLiteValue`'ya bağlanır.
SQL metni SQLite'a değiştirilmeden iletilir; parametre değerleri ayrı gider ve
asla metne eklenmez, kaçışlanmaz ya da tırnaklanmaz. Dolayısıyla

```text
Robert'); DROP TABLE notes;--
```

gibi bir değer tam olarak bu metin olarak saklanır: veridir, asla SQL
değildir. Aynısı tırnaklar, noktalı virgüller, satır sonları, ters bölüler,
Türkçe karakterler ve emojiler için de geçerlidir.

Yer tutucu sayısı parametre sayısına eşit olmalıdır; aksi hâlde hiçbir şey
çalışmadan `SQLiteError` fırlatılır. Adlı parametreler (`:ad`, `@ad`, `$ad`)
v0.3.0 genel API'sinin parçası değildir.

## Bir çağrı, bir ifade

`execute` ve `query` tam olarak bir SQL ifadesi çalıştırır. İkinci bir ifade
içeren metin (`"DELETE FROM a; DELETE FROM b"`) hiçbir şey çalıştırmadan
`SQLiteError` fırlatır; böylece bir parametre listesi asla kastettiğinizden
farklı bir ifadeye uygulanmaz. Sondaki `;`, boşluk ya da yorum sorun değildir.
Birden çok ifadeyi birden çok çağrı olarak çalıştırın; birlikte başarılı ya da
başarısız olmaları gerekiyorsa bir işlem (transaction) kullanın.

## execute

```text
Database.execute(sql: String, parameters: List<SQLiteValue> = []) -> Int
```

Bir ifade çalıştırır ve eklediği, güncellediği ya da sildiği satır sayısını
(tetikleyicilerin değiştirdiği satırlar dâhil) döndürür. `CREATE TABLE`,
`CREATE INDEX`, `DROP TABLE`, `PRAGMA` ve satır değiştirmeyen diğer ifadeler
`0` döndürür.

## query

```text
Database.query(sql: String, parameters: List<SQLiteValue> = []) -> List<Pair<String, SQLiteValue>>
```

Bir ifade çalıştırır ve dönmeden önce tüm sonuç satırlarını tamamen
somutlaştırır. Her satır, anahtarları **sonuç sütunu sırasındaki** sütun
etiketleri olan bir `Pair<String, SQLiteValue>`'dur; `List`, satırları
**SQLite'ın döndürdüğü sırada** tutar. Eşleşen satırı olmayan bir sorgu boş
bir `List` döndürür.

```ahd
for satir in db.query("SELECT id, title FROM notes ORDER BY id") {
    write("{satir["id"].int()}: {satir["title"].string()}")
}
```

### Satır sırası bir SQL sözleşmesidir

SQLite, ifadede `ORDER BY` yoksa hiçbir satır sırası vaat etmez. Basit bir
tabloda satırlar çoğu zaman ekleme sırasında gelir; ama bu bir saklama
ayrıntısıdır, garanti değildir ve indekslerle, silmelerle ve sorgu planlarıyla
değişir. AhdCode `List`'i SQLite'ın ürettiği sırayı korur; asla yeni bir sıra
icat etmez. Sıra önemliyse her zaman `ORDER BY` yazın.

### Yinelenen sütun etiketleri

Bir `Pair` aynı anahtarla iki giriş tutamaz; bu yüzden aynı etikete sahip iki
sütunu olan bir sonuç satırı `SQLiteError` fırlatır:

```sql
SELECT a.id, b.id FROM a JOIN b ON b.a_id = a.id      -- SQLiteError: yinelenen "id"
```

Her sütuna `AS` ile ayrı bir ad verin:

```sql
SELECT a.id AS a_id, b.id AS b_id FROM a JOIN b ON b.a_id = a.id
```

AhdCode sizin yerinize sütunların üzerine yazmaz, yeniden adlandırmaz ya da
numaralandırmaz.

## lastInsertId

```text
Database.lastInsertId() -> Int
```

SQLite'ın bağlantıya özel `last_insert_rowid()` değerini döndürür: **bu**
`Database` üzerindeki en son başarılı `INSERT`'in satır kimliği. Genel bir
birincil anahtar keşif API'si değildir. İlgili `INSERT`'ten hemen sonra
çağırın; aynı `Database` üzerindeki başka bir `INSERT` değeri değiştirir ve
farklı bir `Database` onu hiç görmez. `INTEGER PRIMARY KEY AUTOINCREMENT` ile
bu, yeni satırın `id`'sidir. Hiç ekleme yapılmadan önce `0` döndürür.

## İşlemler (transaction)

```ahd
db.begin()

attempt {
    db.execute(
        "UPDATE accounts SET balance = balance - ? WHERE id = ?",
        [SQLite.fromReal(10.0), SQLite.fromInt(1)]
    )
    db.execute(
        "UPDATE accounts SET balance = balance + ? WHERE id = ?",
        [SQLite.fromReal(10.0), SQLite.fromInt(2)]
    )
    db.commit()
}
except SQLiteError as error {
    db.rollback()
    write(error.message)
}
```

Anlambilim:

- Bir `Database`'in en fazla bir etkin işlemi olur. Biri etkinken `begin()`
  `SQLiteError` fırlatır; iç içe işlem ya da savepoint yoktur.
- Etkin işlem yokken `commit()` ya da `rollback()` `SQLiteError` fırlatır.
- `begin()` ile `commit()`/`rollback()` arasındaki her `execute`/`query` o
  işlemin içinde çalışır ve kendi henüz kalıcılaşmamış değişikliklerini görür.
- `commit()`, `begin()`'den bu yana yapılan her değişikliği yayımlar;
  `rollback()` hepsini — sonraki bir ifade başarısız olmadan önce başarılı
  olanlar dâhil — atar.
- Başarısız bir ifade işlemi kendi başına **bitirmez**. `SQLiteError`'ı
  yakalayın ve `rollback()` (ya da önceki işi tutmak istiyorsanız `commit()`)
  çağırın.
- `begin()`/`commit()` dışında her ifade kendi başına otomatik kalıcılaşan bir
  işlemdir.

## Bağlantı modeli ve close

Her `Database` tam olarak bir mantıksal SQLite bağlantısıdır; gizli bir havuz
yoktur. `:memory:` veritabanlarını ve işlem durumunu öngörülebilir kılan
budur: bir `Database` üzerindeki `begin`, `execute`, `query` ve `commit` her
zaman aynı bağlantıyı gözlemler.

Atama aynı bağlantıya takma ad verir; asla ikinci bir bağlantı açmaz:

```ahd
db: Database := SQLite.open(":memory:")
ayni: Database := db
db.close()
ayni.execute("SELECT 1")     // SQLiteError: the Database is closed
```

`close()` davranışı:

- `close()` bağlantıyı serbest bırakır. Sonrasında o `Database` (ve her takma
  adı) üzerindeki `execute`, `query`, `lastInsertId`, `begin`, `commit` ve
  `rollback`, `the Database is closed` iletisiyle `SQLiteError` fırlatır.
- `close()` idempotenttir: zaten kapalı bir `Database`'i kapatmak başarılıdır.
- Etkin bir işlem varken `close()` `SQLiteError` fırlatır ve işlemi olduğu
  gibi bırakır. Hiçbir şey örtük olarak kalıcılaşmaz ya da atılmaz; önce
  `commit()` ya da `rollback()` çağırın.
- Aynı yol üzerinde bile iki `SQLite.open` çağrısı, bağımsız işlemleri ve
  bağımsız `lastInsertId()` değerleri olan iki ayrı bağlantıdır. İki
  `SQLite.open(":memory:")` çağrısı birbiriyle ilgisiz iki boş veritabanıdır.

Program (ya da REPL oturumu) bittiğinde hâlâ tuttuğu her bağlantı işletim
sistemi tarafından serbest bırakılır; SQLite'ın günlüğü dosyayı tutarlı tutar,
ama niyet görünür olsun diye yine de açıkça `close()` çağırmalısınız.

## Hatalar

`SQLiteError`, `Error`'dan türer ve modülün fırlattığı tek Error sınıfıdır.
`message`'ı, SQLite bir metin üretmişse onun kendi metnini
(`no such table: notes`, `UNIQUE constraint failed: notes.title`,
`near "SELEC": syntax error`), aksi hâlde açık bir AhdCode açıklamasını tutar.
Fırlatıldığı durumlar:

- bozuk SQL;
- eksik tablo, sütun ya da fonksiyon;
- `UNIQUE`, `NOT NULL`, `CHECK` ve `FOREIGN KEY` kısıt ihlalleri;
- yer tutucu/parametre sayısı uyuşmazlığı ya da sonlu olmayan bir `Real`
  parametre;
- birden çok ifade içeren ya da hiç ifade içermeyen metin;
- bir `BLOB` sonuç değeri ya da sonlu olmayan bir `REAL` sonucu;
- yinelenen sonuç sütunu etiketleri;
- yanlış türdeki bir `SQLiteValue` üzerinde `int()`, `real()` ya da
  `string()`;
- geçersiz işlem durumunda `begin`, `commit`, `rollback` ya da `close`;
- kapalı bir `Database` üzerindeki herhangi bir işlem;
- `SQLite.open`'da boş yol, yazılamayan yol ya da eksik üst dizin;
- paketli SQLite yardımcısının bulunamaması (aşağıya bakın).

Go sürücüsünün hataları asla doğrudan açığa çıkmaz; sınıflandırma her zaman
`SQLiteError`'dır. REPL'de yakalanmayan bir `SQLiteError` diğer hatalar gibi
bildirilir ve oturum, değişkenleri ve açık `Database` hayatta kalır.

## REPL

Modül kalıcı REPL'de aynı şekilde çalışır:

```text
ahd> bring SQLite
ahd> db := SQLite.open(":memory:")
ahd> db.execute("CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)")
0
ahd> db.execute("INSERT INTO items (name) VALUES (?)", [SQLite.fromString("Çay")])
1
ahd> db.query("SELECT id, name FROM items ORDER BY id")[0]["name"].string()
Çay
```

`Database` değeri ve bellek içi veritabanı başarılı girişler boyunca kalıcıdır;
başarısız bir SQL girişi `SQLiteError` bildirir ve oturumu olduğu gibi bırakır.
`SQLiteError`, `attempt`/`except` içinde adlandırılmadan önce, tıpkı
`JSONError` gibi içe alınmalıdır (`from SQLite bring SQLiteError`).

## Editör desteği

`SQLite` sıradan bir derleyici kayıtlı modül olduğu için AhdCode dil sunucusu
(v0.2.2 ve sonrası) onu SQLite'a özel hiçbir kod olmadan keşfeder: `bring SQL`
`SQLite`'a tamamlanır, `SQLite.` gerçek üyeleri imzalarıyla listeler,
`from SQLite bring SQL` `SQLiteError` ve `SQLiteValue` önerir; üzerine gelme ve
imza yardımı derleyicinin kendi tiplerini gösterir. v0.3.0 için editör
eklentisinde değişiklik gerekmedi.

## Bağımlılık ve taşınabilirlik modeli

Üretilen AhdCode programları yalnızca Go standart kütüphanesini kullanmaya
devam eder. SQLite motoru, paketli `ahdsqlite` yardımcısında
(`cmd/ahdsqlite`) yaşar; yardımcı
[`github.com/ncruces/go-sqlite3`](https://github.com/ncruces/go-sqlite3)
v0.35.4'ü (MIT) bağlar: saf Go, CGO'suz bir SQLite. Gerçek SQLite C
kütüphanesi (SQLite 3.53.x, kamu malı) WebAssembly'ye derlenir ve ardından
`wasm2go` ile sıradan Go kaynağına çevrilir; böylece Go araç zinciri onu her
paket gibi derler. Sistem `libsqlite3`'ü, `sqlite3` komut satırı aracı, CGO ya
da ağ erişimi yoktur. `CGO_ENABLED=0 go build ./...` başarılıdır. Yardımcı,
derleyicinin yanına kurulur:

```bash
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot ./cmd/ahdsqlite
```

İlk `SQLite.open`'da program bir yardımcı süreci başlatır ve onunla, tıpkı
`Numeric` ve `Plot` modüllerinin kendi üçüncü taraf bağımlılıklarını yalıttığı
gibi, dar bir JSON protokolü (`internal/sqliteproto`) üzerinden konuşur.
Yardımcı keşfi `AHDCODE_SQLITE_RUNTIME`'ı (bir dosya ya da dizini), derleme
anında kaydedilen dizini, çalışan yürütülebilirin dizinini ve kurulu
`libexec/ahdcode` dizinini denetler. Yardımcı bulunamazsa ilk veritabanı
işlemi, `AHDCODE_SQLITE_RUNTIME`'ın nasıl ayarlanacağını açıklayan bir
`SQLiteError` fırlatır.

AhdCode'un yazdığı veritabanı dosyaları sıradan SQLite 3 dosyalarıdır; başka
her SQLite uygulaması (örneğin Python'un `sqlite3` modülü ya da `sqlite3`
CLI'si) onları okuyabilir ve AhdCode da bu araçların ürettiği dosyaları okur.
Bkz. [`THIRD_PARTY_NOTICES_SQLITE.md`](../THIRD_PARTY_NOTICES_SQLITE.md).

## Kapsam dışı

Bu sürüm bilinçli olarak bir SQL köprüsüdür, daha fazlası değildir: ORM ya da
active-record katmanı yok, model sınıfları ya da satırdan Class'a eşleme yok,
sorgu ya da şema oluşturucu yok, migration yok, ilişkiler yok, bağlantı
havuzu API'si yok, eşzamansız SQL ya da arka plan iş parçacıkları yok,
şifreleme katmanı yok, SQL yeniden yazımı yok (AhdCode asla `LIMIT`,
`ORDER BY` ya da tırnak eklemez), AhdCode'un bir ikili veri tipi olana kadar
`BLOB` desteği yok, adlı parametre API'si yok, savepoint yok ve gelecekteki
bir MySQL modülüyle paylaşılan genel bir veritabanı arayüzü yok. `SQLite`
kendi başına bir modüldür.

Ayrıca bkz. [Öğrenci Rehberi — SQLite](STUDENT_GUIDE_TR.md#35-sqlite-hatırlayan-bir-veritabanı) ·
[`examples/v0.3/01_sqlite_notes.ahd`](../examples/v0.3/01_sqlite_notes.ahd).
