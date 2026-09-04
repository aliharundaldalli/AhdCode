# MySQL standart modülü

[English](MYSQL.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [SQLite](SQLITE_TR.md) · [Env](ENV_TR.md) · [Öğrenci Rehberi](STUDENT_GUIDE_TR.md#51-mysql-a%C4%9F-veritaban%C4%B1-sunucusu)

`MySQL`, AhdCode v0.11.0 ile gelen, derleyici tarafından kayıtlı
`builtin:MySQL` modülüdür. Gerçek bir MySQL sunucusuna, MySQL tel
protokolünün saf Go ile yazılmış bir uygulaması olan
`github.com/go-sql-driver/mysql` ile ağ üzerinden bağlanır; bu bağımlılık
AhdCode'un kendisine gömülüdür, böylece üretilen bir MySQL programı ağa hiç
dokunmadan derlenir (aşağıdaki "Çevrimdışı derlemeler" bölümüne bakın). Ne
`mysql` istemci kütüphanesi bağımlılığı, ne CGO, ne de dış bir yardımcı
süreç vardır.

`MySQL`, [`SQLite`](SQLITE_TR.md)'tan kasıtlı olarak ayrı bir modüldür;
paylaşılan bir `Database`/`Value` soyutlaması yerine kendi tip adlarını
kullanır (`MySQLDatabase`, `MySQLTransaction`, `MySQLResult`, `MySQLValue`,
`MySQLError`). SQLite yerel bir dosyadır; MySQL ise kendi bağlantı yaşam
döngüsü, kimlik doğrulaması ve işlem modeline sahip bir ağ sunucusudur —
ikisini tek bir genel arayüzde birleştirmek, gerçek farkları basitleştirmek
yerine bulanıklaştırır. Bir program ikisini de aynı anda `bring` edebilir,
hiçbir çakışma olmaz.

## Genel yüzey

```text
MySQL.connect(
    host: String
    username: String
    password: String
    port: Int := 3306
    database: String? := null
    security: String := "tls"
    timeoutSeconds: Int := 10
) -> MySQLDatabase

MySQL.nullValue()                    -> MySQLValue
MySQL.fromInt(value: Int)            -> MySQLValue
MySQL.fromReal(value: Real)          -> MySQLValue
MySQL.fromString(value: String)      -> MySQLValue

MySQLDatabase.ping()                                              -> Nothing
MySQLDatabase.execute(sql: String, params: List<MySQLValue> := []) -> MySQLResult
MySQLDatabase.query(sql: String, params: List<MySQLValue> := [])   -> List<Pair<String, MySQLValue>>
MySQLDatabase.begin()                                              -> MySQLTransaction
MySQLDatabase.close()                                              -> Nothing

MySQLTransaction.execute(sql: String, params: List<MySQLValue> := []) -> MySQLResult
MySQLTransaction.query(sql: String, params: List<MySQLValue> := [])   -> List<Pair<String, MySQLValue>>
MySQLTransaction.commit()                                              -> Nothing
MySQLTransaction.rollback()                                            -> Nothing

MySQLResult.affectedRows() -> Int
MySQLResult.lastInsertId() -> Int?

MySQLValue.kind()   -> String
MySQLValue.isNull() -> Bool
MySQLValue.int()    -> Int
MySQLValue.real()   -> Real
MySQLValue.string() -> String
MySQLValue.isBinary()     -> Bool
MySQLValue.binarySize()   -> Int
MySQLValue.binaryBase64() -> String

MySQLError  (Error'dan türer)
```

`MySQLDatabase`, `MySQLTransaction`, `MySQLResult` ve `MySQLValue` opak
yerleşik Class'lardır: doğrudan inşa edilemezler, herkese açık özniteliği
yoktur ve yalnızca yukarıdaki fonksiyon ve metotlardan elde edilirler.

## Bağlanma

```ahd
bring Env
bring MySQL
from MySQL bring MySQLDatabase

host := Env.getOr("MYSQL_HOST", "127.0.0.1")
username := Env.getOr("MYSQL_USERNAME", "app")
password := Env.getOr("MYSQL_PASSWORD", "")

db: MySQLDatabase := MySQL.connect(host, username, password)
```

Gerçek bir parolayı asla kaynak koduna yazmayın; [Env](ENV_TR.md) ile okuyun
— [SMTP](SMTP_TR.md)'nin kullandığı aynı kural.

`connect`, salt tembel bir tutamaç oluşturmakla kalmaz: dönmeden önce
sunucuya bağlanır ve sınırlı süreli bir ping çalıştırır, böylece elinizdeki
bir `MySQLDatabase`'in erişilebilir olduğu zaten bilinir. Erişilemeyen bir
sunucu, yanlış parola veya yanlış kullanıcı adı — hepsi `MySQLError`
yükseltir; bu, küçük bir kategori mesajı kümesine eşlenir — asla ham sürücü
hatası değil, çünkü o bağlantı ayrıntılarını geri yansıtabilir.

### `database` isteğe bağlıdır

```ahd
admin: MySQLDatabase := MySQL.connect(host, username, password, 3306, null, "tls")
rows := admin.query("SHOW DATABASES")
```

`null` geçmek (veya argümanı tamamen atlamak) hiçbir varsayılan veritabanı
seçmeden bağlanır — yalnızca kimlik bilgileri, bağlantının neyi
görebileceğine karar verir. Bu, bir veritabanı yönetim aracının, herhangi
biri seçilmeden önce bir kimlik bilgisi kümesinin görmeye yetkili olduğu her
veritabanını/şemayı `SHOW DATABASES` veya `INFORMATION_SCHEMA` ile
keşfetmesini sağlayan şeydir. Boş bir String (`""`), atlanmışsıyla aynı
şekilde davranır. Gerçek bir şema adı geçmek, o bağlantıdaki sonraki her
ifade için o veritabanını seçer — sıradan bir MySQL istemcisinde olduğu gibi.

### Güvenlik kipleri

Yalnızca kesin küçük harfli değerler — takma ad yok, düşürme yok:

| Değer | Anlamı |
|---|---|
| `"tls"` (varsayılan) | TLS zorunlu; sistem güven kökleri; ana bilgisayar adı doğrulanır; güvensiz atlama yok |
| `"none"` | Açıkça şifresiz bağlantı |

`security` `"tls"` ise ve sunucunun sertifikası güvenilmiyorsa, süresi
dolmuşsa veya kimliği bağlandığınız ana bilgisayarla eşleşmiyorsa, `connect`
`MySQLError` yükseltir. Herkese açık bir "her şeye güven" veya "ana
bilgisayar adını atla" anahtarı yoktur — [SMTP](SMTP_TR.md)'nin `"tls"`
kipinin aldığı aynı duruş. `"none"` açıktır ve güvenilir yerel geliştirme
içindir; güvenli olduğunu iddia etmez.

`port` `1..65535` aralığında olmalıdır. `timeoutSeconds` `1..9223372036`
aralığında olmalıdır ve o bağlantıdaki bağlanmayı, TLS el sıkışmasını ve
sonraki her ifadenin ağ G/Ç'sini sınırlar — [HTTP](HTTP_TR.md) ve
[SMTP](SMTP_TR.md)'nin kullandığı aynı süre sözleşmesi, bir
`time.Duration` dönüşümünün asla taşmaması için seçilmiştir.

## Parametreli sorgular

```ahd
db.execute(
    "INSERT INTO users (name, email) VALUES (?, ?)"
    [MySQL.fromString(name), MySQL.fromString(email)]
)
```

Her `?`, gerçek bir sunucu taraflı bağlı parametredir — SQL metni asla
yeniden yazılmaz ve bir değer asla içine eklenmez. Bu şekilde saklanan ve SQL
gibi görünen bir String (`Robert'); DROP TABLE users;--`), sıradan metin
olarak saklanır; tablo hiç dokunulmadan kalır. Bu, SQL enjeksiyonuna karşı
doğru savunmadır. AhdCode ayrıca SQL metnini temizlemez veya kaçış işlemi
yapmaz; uygulamalar güvenilmeyen girdiyi doğrudan SQL String'inin içine
birleştirerek sorgu oluşturmamalıdır.

`MySQL.nullValue()`, `.fromInt`, `.fromReal` ve `.fromString`, bir
`MySQLValue` oluşturur. Salt bir `String` veya `Int` argümanından örtük bir
dönüşüm yoktur ve `MySQL.fromBinary` yoktur: parametreli ikili girdi v0.11.0
kapsamı dışındadır, çünkü `MySQLValue.isBinary()` yalnızca bir sorgudan geri
okunan bir değerde görünür.

## Satırları okuma

```ahd
rows: List<Pair<String, MySQLValue>> := db.query(
    "SELECT id, name FROM users ORDER BY id"
)
for row in rows {
    write("{row["id"].int()}: {row["name"].string()}")
}
```

Her satır, anahtarları sonuç sütunu etiketleri olan (sonuç sırasında) bir
`Pair`'dir — [SQLite](SQLITE_TR.md)'ın kullandığı aynı şekil. Sonucunda aynı
etikete sahip iki sütun bulunan bir sorgu (`SELECT a.id, b.id`)
`MySQLError` yükseltir — AhdCode'un sessizce birini seçmesi yerine `AS` ile
birini takma adlandırın.

### Tip eşlemesi

| MySQL tipi | `kind()` | Şununla okuyun |
|---|---|---|
| `NULL` | `"Null"` | `isNull()` |
| `TINYINT` … `BIGINT`, `YEAR` (işaretli veya işaretsiz, `Int`'e sığar) | `"Int"` | `int()` |
| `FLOAT`, `DOUBLE` | `"Real"` | `real()` |
| `CHAR`, `VARCHAR`, `TEXT` ailesi, `ENUM`, `SET` | `"String"` | `string()` |
| `DECIMAL` / `NUMERIC` | `"String"` | `string()` |
| `DATE`, `TIME`, `DATETIME`, `TIMESTAMP` | `"String"` | `string()` |
| `JSON` | `"String"` | `string()` |
| `Int`'e sığmayacak kadar büyük işaretsiz tamsayı (`BIGINT UNSIGNED` tavanına yakın) | `"String"` | `string()` |
| `BLOB` ailesi, `BINARY`, `VARBINARY`, `BIT` | `"Binary"` | `isBinary()`, `binarySize()`, `binaryBase64()` |

Eşleme, sütunun kendi bildirilmiş tipine göre kararlaştırılır, metinden asla
tahmin edilmez — [SQLite](SQLITE_TR.md)'ın uyguladığı aynı disiplin. Yanlış
erişimciyi çağırmak (`"String"` bir değerde `int()`, `"Int"`'te `string()`)
sessizce dönüştürmek yerine `MySQLError` yükseltir.

**`DECIMAL` kasıtlı olarak bir `String` olarak kalır.** `"19.99"` gibi bir
`DECIMAL(10,2)` değeri asla `Real`'e zorlanmaz, çünkü ikili kayan nokta her
ondalık kesri tam olarak temsil edemez — zorlamak parayı ve diğer kesin
miktarları sessizce bozar. Gerçekten aritmetiğe ihtiyacınız varsa açıkça
dönüştürün (ve yaptığınızda hassasiyet kaybına dikkat edin).

**Tarihler ve saatler sade `String`'lerdir** (örn. `"2026-01-15 10:30:00"`),
tam olarak MySQL'in gönderdiği şekilde. v0.11.0'ın zamansal bir depolama
tipi yoktur; metni opak olarak ele alın veya kendiniz [Time](TIME_TR.md) ile
ayrıştırın.

### İkili değerler

```ahd
value: MySQLValue := rows[0]["payload"]
if value.isBinary() {
    write(str(value.binarySize()))
    encoded := value.binaryBase64()
}
```

Bir `BLOB`, `NUL` ve geçersiz UTF-8 dahil herhangi bir bayt içerebilir; asla
bir AhdCode `String`'e dönüşmez ve asla birinin içinden zorlanmaz.
`binaryBase64()`, onu metin olarak çıkarmanın tek yoludur. v0.11.0'da
herkese açık bir `Bytes` tipi yoktur ve bir sorguya *bağlanan* ikili
parametreler kapsam dışıdır — bu değer modeli salt okunurdur.

## Sonuçlar

```ahd
result: MySQLResult := db.execute("UPDATE users SET active = ? WHERE id = ?", [...])
changed: Int := result.affectedRows()
newID: Int? := result.lastInsertId()
```

`MySQLResult` değişmezdir ve onu üreten tek `execute` çağrısına aittir —
`MySQLDatabase`'e asılı değişebilir bir "son sonuç" asla değildir; bu, iki
isteğin aynı bağlantıyı eşzamanlı paylaştığı anda önem kazanır.
`lastInsertId()`, ifade yeni bir id üretmediğinde (bir `UPDATE`, veya
`AUTO_INCREMENT` sütunu olmayan bir tabloya `INSERT`) `null`'dır — asla
önceki bir ifadeden taşınan sahte bir değer değildir.

## İşlemler (Transactions)

```ahd
tx: MySQLTransaction := db.begin()
attempt {
    tx.execute("UPDATE accounts SET balance = balance - ? WHERE id = ?", [...])
    tx.execute("UPDATE accounts SET balance = balance + ? WHERE id = ?", [...])
    tx.commit()
} except MySQLError as error {
    tx.rollback()
    write(error.message)
}
```

`begin()`, kendi alttaki bağlantısını sabitleyen bağımsız bir
`MySQLTransaction` döndürür: aynı `MySQLDatabase`'den açılan iki işlem —
eşzamanlı iki istekten açılanlar dahil — birbirine asla karışmaz, yanlarında
çalışan sıradan bir `db.execute` de karışmaz. `commit()` ve `rollback()`
her biri tek seferliktir: birini ikinci kez çağırmak, ya da işlemi sonrasında
kullanmak, sessizce hiçbir şey yapmak yerine `MySQLError` yükseltir.

## Eşzamanlılık (Concurrency)

Bir `MySQLDatabase`, birden çok istekten eşzamanlı kullanım için güvenlidir
— Go'nun sıradan `database/sql` bağlantı havuzu tarafından desteklenir,
diğer her Go MySQL istemcisinin dayandığı aynı havuzlama. AhdCode onu tek
bir genel kilide sarmaz: bağımsız `execute`/`query` çağrıları eşzamanlı
çalışır, yalnızca açık bir `MySQLTransaction` ifadeleri bir arada gruplar.
v0.11.0'da herkese açık bir havuz ayarlama API'si yoktur; çalışma zamanının
kendi tutucu varsayılanları geçerlidir.

## Kapatma

```ahd
db.close()
```

Havuzlanmış bağlantıları serbest bırakır. İki kez kapatmak güvenle hiçbir
şey yapmaz; `close()`'dan sonra bir `MySQLDatabase` üzerindeki — veya
`commit()`/`rollback()`'tan sonra bir `MySQLTransaction` üzerindeki — herhangi
bir işlem, yıkılmış durumu yeniden kullanmak yerine `MySQLError` yükseltir.

## Hatalar

Her hata, `Error`'dan türeyen `MySQLError`'dır; küçük bir kategori mesajı
kümesiyle: bağlantı başarısız, bağlantı zaman aşımına uğradı, TLS
doğrulaması başarısız, sorgu başarısız, yürütme başarısız, işlem başarısız
(zaten kapalıyken yeniden kullanım dahil), veya bir değer-tipi uyumsuzluğu.
Sorgu/yürütme hataları, sunucunun kendi hata kodunu ve mesajını içerir, çünkü
o metin sunucuda oluşur ve asla parolanızı taşımaz; bağlantı aşaması
hataları sürücünün ham hata metnini asla içermez, böylece içine gömülü bir
parola veya adres hiçbir zaman bir tanılamaya, günlüğe veya yığın izine
sızamaz.

## Çevrimdışı derlemeler

El yazımı üçüncü taraf bir bağımlılığın aksine, gömülü sürücü AhdCode'un
kendisiyle birlikte gelir: `MySQL`'i `bring` eden bir programda `ahdcode
build`, tam olarak sabitlenmiş `github.com/go-sql-driver/mysql` ve
`filippo.io/edwards25519` kaynağını (`ahdcode` ikili dosyasının içine
`internal/backend/golang/ahdruntime/mysqlvendor`'da gömülü) o programın özel
derleme çalışma alanına `vendor/` olarak kopyalar, ardından `go build
-mod=vendor` ile derler. Derleme hiçbir zaman bir modül proxy'sine
bağlanmaz ve yerel Go modül önbelleğinin sıcak olmasına asla bağlı değildir
— MySQL kullanan bir AhdCode programı, diğer her üretilmiş program kadar
çevrimdışı yeniden üretilebilir şekilde derlenebilirdir. MySQL kullanmayan
bir program bundan hiç etkilenmez: bugünkü sade, bağımlılıksız `go.mod`'unu
korur.

Bu, AhdCode'un [`Latex.pdf`](LATEX_TR.md)'in Tectonic motoru için zaten
uyguladığı aynı tasarım ilkesidir: olgun bir protokol istemcisini veya
render motorunu sıfırdan yeniden uygulamak, dil değeri katmadan risk katar,
bu yüzden AhdCode gerçek uygulamayı — geliştirme zamanında bir kez
denetlenmiş, sabitlenmiş ve çevrimdışı gönderilmiş olarak — kendi küçük,
tipli, hata denetimli yüzeyinin arkasına gömer. MySQL tel protokolü (
`caching_sha2_password`'ın RSA anahtar değişimi dahil), tam olarak bu türden
koddur: AhdCode'un işi yukarıdaki güvenli, tipli
`connect`/`execute`/`query`/`begin` sözleşmesidir, üretim kalitesinde bir
protokol uygulamasını yeniden türetmek değil. Gömülü kodun lisansları için
[`THIRD_PARTY_NOTICES_MYSQL.md`](../THIRD_PARTY_NOTICES_MYSQL.md) dosyasına
bakın.

## Kapsam dışı olanlar

ORM veya active-record katmanı yok, model sınıfları yok, sorgu veya şema
oluşturucu yok, migration yok, bağlantı dizesi/DSN mini-dili yok, saklı
yordam veya tetikleyici çerçevesi yok, replikasyon veya binlog API'si yok,
yönetim sunucusu yok, bağlantı havuzu ayarlama API'si yok ve
[SQLite](SQLITE_TR.md) ile paylaşılan genel bir `Database` arayüzü yok.
Bağlı ikili parametreler ve herkese açık bir `Bytes` tipi de v0.11.0
kapsamı dışındadır. MariaDB aynı tel protokolü üzerinden tesadüfen
çalışabilir, ancak yalnızca MySQL 8.x test edilen hedeftir.

Ayrıca bakınız: [Öğrenci Rehberi — MySQL](STUDENT_GUIDE_TR.md#51-mysql-a%C4%9F-veritaban%C4%B1-sunucusu) ·
[`examples/v0.11`](../examples/v0.11/README_TR.md).
