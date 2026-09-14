# PostgreSQL standart modülü

[English](POSTGRESQL.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [MySQL](MYSQL_TR.md) · [SQLite](SQLITE_TR.md) · [Env](ENV_TR.md) · [UUID](UUID_TR.md)

`PostgreSQL`, AhdCode v1.4.0 ile gelen ve derleyicide kayıtlı olan
`builtin:PostgreSQL` modülüdür. Ağ üzerinden bir PostgreSQL sunucusuna
`github.com/jackc/pgx/v5` ile bağlanır. pgx, PostgreSQL protokolünün saf Go
uygulamasıdır ve AhdCode'un içine gömülüdür; bu yüzden PostgreSQL kullanan bir
program ağa hiç çıkmadan derlenir (aşağıdaki "Çevrimdışı derleme" bölümüne
bakın). `libpq`, CGO ya da harici bir yardımcı süreç yoktur.

`PostgreSQL`, [MySQL](MYSQL_TR.md) ailesinden ayrı bir modüldür; ortak bir
`Database` soyutlaması yerine kendi tür adlarını kullanır. İki modül bilerek
birbirine benzer ve sunucuların gerçekten farklı olduğu yerlerde ayrılır;
aşağıdaki "MySQL'den farkları" bölümüne bakın. Bir program MySQL, PostgreSQL
ve SQLite modüllerini çakışma olmadan birlikte `bring` edebilir.

## Genel arayüz

```text
PostgreSQL.connect(
    host: String
    username: String
    password: String
    port: Int := 5432
    database: String? := null
    security: String := "tls"
    timeoutSeconds: Int := 10
) -> PostgreSQLDatabase

PostgreSQL.nullValue()               -> PostgreSQLValue
PostgreSQL.fromInt(value: Int)       -> PostgreSQLValue
PostgreSQL.fromReal(value: Real)     -> PostgreSQLValue
PostgreSQL.fromString(value: String) -> PostgreSQLValue
PostgreSQL.fromBool(value: Bool)     -> PostgreSQLValue

PostgreSQLDatabase.ping()                                                    -> Nothing
PostgreSQLDatabase.execute(sql: String, params: List<PostgreSQLValue> := []) -> PostgreSQLResult
PostgreSQLDatabase.query(sql: String, params: List<PostgreSQLValue> := [])   -> List<Pair<String, PostgreSQLValue>>
PostgreSQLDatabase.begin()                                                   -> PostgreSQLTransaction
PostgreSQLDatabase.close()                                                   -> Nothing

PostgreSQLTransaction.execute(sql: String, params: List<PostgreSQLValue> := []) -> PostgreSQLResult
PostgreSQLTransaction.query(sql: String, params: List<PostgreSQLValue> := [])   -> List<Pair<String, PostgreSQLValue>>
PostgreSQLTransaction.commit()                                                  -> Nothing
PostgreSQLTransaction.rollback()                                                -> Nothing

PostgreSQLResult.affectedRows() -> Int

PostgreSQLValue.kind()         -> String
PostgreSQLValue.isNull()       -> Bool
PostgreSQLValue.bool()         -> Bool
PostgreSQLValue.int()          -> Int
PostgreSQLValue.real()         -> Real
PostgreSQLValue.string()       -> String
PostgreSQLValue.isBinary()     -> Bool
PostgreSQLValue.binarySize()   -> Int
PostgreSQLValue.binaryBase64() -> String

PostgreSQLError  (Error'dan türer)
```

`PostgreSQLDatabase`, `PostgreSQLTransaction`, `PostgreSQLResult` ve
`PostgreSQLValue` opak yerleşik sınıflardır: doğrudan oluşturulamazlar,
yalnızca yukarıdaki fonksiyon ve metotlardan elde edilirler.

## Bağlanma

```ahd
bring Env
bring PostgreSQL
from PostgreSQL bring PostgreSQLDatabase

host := Env.getOr("DB_HOST", "127.0.0.1")
username := Env.getOr("DB_USERNAME", "app")
password: String? := Env.secret("DB_PASSWORD")
if password != null {
    db: PostgreSQLDatabase := PostgreSQL.connect(host, username, password, 5432, "app")
}
```

Gerçek bir parolayı asla kaynak koda yazmayın. [`Env.secret`](ENV_TR.md),
`DB_PASSWORD` değişkenini ya da `DB_PASSWORD_FILE` ile adı verilen dosyayı
okur; konteyner platformları gizli değerleri bu şekilde bağlar.

`connect` sunucuya bağlanır, kimlik doğrular ve dönmeden önce süresi sınırlı
bir ping çalıştırır; yani elinizdeki bir `PostgreSQLDatabase` erişilebilir
olduğu bilinen bir veritabanıdır. Erişilemeyen bir sunucu, yanlış parola ya da
olmayan bir veritabanı `PostgreSQLError` oluşturur.

### `database` isteğe bağlıdır

`null` ya da `""` verildiğinde sunucu kendi varsayılanını seçer: rol ile aynı
adı taşıyan veritabanı. Başka bir değer, havuzdaki her bağlantı için o
veritabanını seçer.

### Güvenlik kipleri

Yalnızca tam, küçük harfli değerler geçerlidir. Takma ad ya da düşürme yoktur:

| Değer | Anlamı |
|---|---|
| `"tls"` (varsayılan) | TLS zorunlu; sistem güven kökleri; ana makine adı doğrulanır; TLS 1.2 ve üstü; doğrulamayı atlama yok |
| `"none"` | Açıkça şifresiz bağlantı |

`"tls"` kipinde TLS'i reddeden, güvenilmeyen ya da süresi dolmuş sertifika
sunan veya ana makine adıyla eşleşmeyen bir sunucu
`PostgreSQL TLS verification failed` hatası verir. Güven kökleri yalnızca
standart yolla, `SSL_CERT_FILE` ile genişletilebilir. `"none"` güvenilir yerel
geliştirme içindir; güvenliymiş gibi davranmaz.

`port` `1..65535` aralığında olmalıdır. `timeoutSeconds` `1..9223372036`
aralığında olmalıdır ve bağlanmayı, TLS el sıkışmasını ve her ifadeyi sınırlar.

### Ortam değişkenleri sunucuyu seçmez

Nereye ve nasıl bağlanılacağına yalnızca `connect` argümanları karar verir.
`PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`, `PGSSLMODE`,
`PGOPTIONS`, `PGTZ`, `PGAPPNAME` ve diğer tüm `PG*` değişkenlerinin,
`~/.pgpass` dosyasının, `~/.postgresql/` sertifikalarının ve servis
dosyalarının hiçbir etkisi yoktur.

Görünür tek bir istisna vardır: `PGSERVICE` okunamayan bir servisi
gösteriyorsa `connect` şu mesajla durur:
`PostgreSQL connection failed: PGSERVICE names a service that cannot be read; AhdCode never uses service files, so unset PGSERVICE`.
Okunabilen bir servis dosyası yine yok sayılır ve hiçbir zaman başka bir yere
bağlanılmaz.

## Parametreli sorgular

```ahd
db.execute(
    "INSERT INTO users (name, email, active) VALUES ($1, $2, $3)"
    [PostgreSQL.fromString(name), PostgreSQL.fromString(email), PostgreSQL.fromBool(true)]
)
```

Yer tutucular PostgreSQL'in kendi `$1`, `$2`, … biçimidir. Her ifade genişletilmiş
protokolle gönderilir ve değerleri sunucuda bağlanır; SQL metni hiçbir zaman
yeniden yazılmaz ve bir değer metnin içine eklenmez. `?` sıradan SQL metnidir
ve çevrilmez. Uygulamalar güvenilmeyen girdiyi String'e ekleyerek SQL
oluşturmamalıdır.

- Bir çağrı tam olarak bir ifade çalıştırır. `"UPDATE …; DROP TABLE …"`
  `PostgreSQL execution failed: (42601) cannot insert multiple commands into a prepared statement`
  hatası verir ve hiçbiri çalışmaz.
- Değer sayısı yer tutucularla eşleşmelidir:
  `PostgreSQL query failed: the statement has 2 placeholder(s) but 1 parameter(s) were passed`.
- `fromString`, sunucunun kullanıldığı yere göre dönüştürdüğü metni gönderir.
  Türün belirsiz olduğu yerlerde SQL içinde dönüştürün: `$1::uuid`,
  `$1::numeric`, `$1::timestamptz`.
- `fromBinary` yoktur; ikili değerler v1.4.0'da yalnızca okunur.

`lastInsertId` yoktur. Üretilen değerleri `RETURNING` ile isteyin ve `query`
ile okuyun:

```ahd
created := db.query(
    "INSERT INTO books (id, title) VALUES ($1::uuid, $2) RETURNING added_at"
    [PostgreSQL.fromString(UUID.v7().string()), PostgreSQL.fromString(title)]
)
write(created[0]["added_at"].string())
```

## Satırları okuma

Her satır, anahtarları sonuç sütun etiketleri olan bir `Pair`'dir; MySQL ve
SQLite ile aynı biçimdir. Aynı etikete sahip iki sütun (`SELECT a.id, b.id …`)
`query result has duplicate column "id"; alias it with AS` hatası verir.

### Tür eşlemesi

Eşleme her sütunun bildirilen türüne göre yapılır, içeriğine göre asla.

| PostgreSQL | `kind()` | Değer |
|---|---|---|
| NULL | `"Null"` | |
| `boolean` | `"Bool"` | `bool()` |
| `smallint`, `integer`, `bigint` | `"Int"` | `int()` |
| `real`, `double precision` | `"Real"` | `real()`; NaN ve ±Infinity sütun adıyla hata verir |
| `numeric` | `"String"` | `"19.990"` gibi tam metin; asla Real'e çevrilmez |
| `uuid` | `"String"` | küçük harfli kanonik metin |
| `json`, `jsonb` | `"String"` | sunucunun metni |
| `date` | `"String"` | `YYYY-MM-DD` |
| `timestamp` | `"String"` | `YYYY-MM-DD HH:MM:SS[.ffffff]` |
| `timestamptz` | `"String"` | UTC, `YYYY-MM-DD HH:MM:SS[.ffffff]+00` |
| `bytea` | `"Binary"` | `isBinary()`, `binarySize()`, `binaryBase64()` |
| metin ailesi, enum'lar, `time`, `interval`, `inet`, `money`, aralıklar, diğer skalerler | `"String"` | sunucunun metni |
| diziler, bileşik (composite) ve record değerler | — | sütun adıyla `PostgreSQLError` verir |

Yanlış erişimciyi çağırmak hata verir, örneğin
`int() requires kind Int; this PostgreSQLValue has kind String (check kind() first)`.
`real()`, `"Int"` bir değeri de okur.

**Tarih ve saatler oturum ayarlarına bağlı değildir.** `date`, `timestamp` ve
`timestamptz` PostgreSQL'in ikili biçiminde okunur ve metne AhdCode çevirir;
bu yüzden `DateStyle` ve `TimeZone` aldığınız metni değiştirmez. `timestamptz`
bir andır ve her zaman `+00` ile UTC olarak gösterilir; AhdCode oturumun
`TimeZone` ayarını hiç değiştirmez. `infinity`, `-infinity` ve MÖ tarihleri
(`"0044-03-15 BC"`) PostgreSQL'in yazımını korur. Kesir kısmının sondaki
sıfırları atılır: `07:30:00.120` değeri `"07:30:00.12"` olarak okunur.

**`numeric` bilerek String kalır**; nedeni MySQL'deki `DECIMAL` ile aynıdır:
ikili kayan nokta her ondalık kesri tam tutamaz.

**Diziler ve bileşik değerler** v1.4.0'da okunmaz. Bunları SQL içinde
dönüştürün: `array_to_json(tags)::text`, `row_to_json(address)::text`.

### İkili değerler

Bir `bytea` değeri herhangi bir baytı içerebilir; asla String'e dönüşmez.
Metin olarak almanın tek yolu `binaryBase64()`'tür.

## Sonuçlar

`PostgreSQLResult.affectedRows()`, sunucunun o tek `execute` çağrısı için
bildirdiği satır sayısıdır.

## İşlemler (transaction)

```ahd
tx: PostgreSQLTransaction := db.begin()
attempt {
    tx.execute("UPDATE accounts SET balance = balance - $1::numeric WHERE id = $2", [...])
    tx.execute("UPDATE accounts SET balance = balance + $1::numeric WHERE id = $2", [...])
    tx.commit()
} except PostgreSQLError as error {
    tx.rollback()
    write(error.message)
}
```

`begin()` havuzdan bir bağlantıyı sabitler ve `READ COMMITTED` bir işlem
başlatır. PostgreSQL başarısız bir ifadeyi MySQL'den farklı ele alır:

- Bir ifade başarısız olduktan sonra işlem iptal edilmiş olur. İçindeki sonraki
  her ifade
  `(25P02) current transaction is aborted, commands ignored until end of transaction block`
  hatası verir.
- İptal edilmiş bir işlemde `commit()`
  `PostgreSQL transaction was rolled back because an earlier statement failed`
  hatası verir ve işlemi bitirir; hiçbir şey kaydedilmemiştir.
- Başarısız bir `commit()` sonrasında `rollback()` hiçbir şey yapmaz ve hata
  vermez; bu yüzden yukarıdaki kalıp asla iki kez hata vermez.
- Bunun dışında `commit()` ve `rollback()` tek kullanımlıktır: ikinci çağrı ya
  da sonrasında çalıştırılan herhangi bir ifade
  `this PostgreSQLTransaction is already committed or rolled back` hatası verir.

v1.4.0'da kayıt noktası (savepoint) ve yalıtım düzeyi seçeneği yoktur.

## Eşzamanlılık ve zaman aşımları

Bir `PostgreSQLDatabase`, eşzamanlı kullanıma uygun bir bağlantı havuzudur;
genel bir kilit ve havuz ayar API'si yoktur. HTTP işleyicilerinden ya da
WebSocket geri çağrılarından çalıştırılan ifadeler yine birer birer çalışır,
çünkü bu geri çağrılar da öyle çalışır.

`timeoutSeconds` süresini aşan bir ifade sunucuda iptal edilir ve
`PostgreSQL connection timed out` hatası verir. Havuz kullanılabilir kalır.

## Kapatma

`db.close()`, veritabanında hâlâ açık olan her işlemi geri alır — hiçbir şey
örtük olarak kaydedilmez — ve havuzu serbest bırakır. İkinci kez kapatmak
başka bir şey yapmaz. Sonraki her kullanım `this PostgreSQLDatabase is closed`
hatası verir.

## Hatalar

Her hata `Error`'dan türeyen `PostgreSQLError`'dır:

| Mesaj şununla başlar | Ne zaman |
|---|---|
| `PostgreSQL connection failed` | bağlanma, kimlik doğrulama, olmayan veritabanı |
| `PostgreSQL connection timed out` | bağlanma ya da ifade `timeoutSeconds` süresini aştı |
| `PostgreSQL TLS verification failed` | `"tls"` kurulamadı ya da doğrulanamadı |
| `PostgreSQL query failed` | `query` |
| `PostgreSQL execution failed` | `execute` |
| `PostgreSQL transaction failed` | `begin`, `commit`, `rollback` |

Sunucu hataları SQLSTATE kodunu ve birincil mesajı ekler, örneğin
`PostgreSQL execution failed: (23505) duplicate key value violates unique constraint "users_email_key"`.
Sunucunun `DETAIL`, `HINT` ve `WHERE` alanları satır değerlerini
tekrarlayabildiği için hiçbir zaman eklenmez. Bağlantı aşamasındaki hatalar
sürücünün ham metnini içermez ve hiçbir mesaj parolayı içermez.

## MySQL'den farkları

| | MySQL | PostgreSQL |
|---|---|---|
| Yer tutucular | `?` | `$1`, `$2`, … |
| Varsayılan port | `3306` | `5432` |
| Üretilen kimlikler | `MySQLResult.lastInsertId()` | `query` ile `RETURNING` |
| Mantıksal değerler | tamsayılar | `boolean`, `"Bool"` olarak okunur; `fromBool` bağlar |
| Tarih ve saatler | sunucunun metni | sabit biçimler; `timestamptz` UTC |
| Diziler, bileşik değerler | — | sütun adıyla hata verir |
| İşlem içinde başarısız ifade | işlem kullanılabilir kalır | işlem iptal edilir; `commit()` hata verir |
| Çağrı başına birden fazla ifade | desteklenmez | hata verir ve hiçbiri çalışmaz |

## Çevrimdışı derleme

`PostgreSQL` modülünü `bring` eden bir programda `ahdcode build`, `ahdcode`
ikilisine gömülü sabit sürümlü `github.com/jackc/pgx/v5` v5.11.0 kaynağını ve
bağımlılıklarını (`jackc/pgpassfile`, `jackc/pgservicefile`,
`jackc/puddle/v2`, `golang.org/x/sync`, `golang.org/x/text`) programın derleme
alanına `vendor/` olarak kopyalar ve `-mod=vendor` ile derler. Derleme hiçbir
modül vekiline bağlanmaz. PostgreSQL kullanmayan bir program bu ağacı almaz.
Lisanslar için
[`THIRD_PARTY_NOTICES_POSTGRESQL.md`](../THIRD_PARTY_NOTICES_POSTGRESQL.md)
dosyasına bakın.

## Hedef dışı olanlar

ORM, sorgu ya da şema oluşturucu, geçiş (migration) sistemi, bağlantı dizesi ve
ortak bir `Database` arayüzü yoktur. v1.4.0'da olmayanlar: `LISTEN`/`NOTIFY`,
`COPY`, diziler, bileşik değerler, kayıt noktaları, yalıtım düzeyleri, havuz
ayarı, bağlanmış ikili parametreler, `schema` seçeneği ve deyim önbelleği
API'si. AhdDataStudio ve `ahdcode init web` henüz PostgreSQL sunmaz. Test
edilen sunucu PostgreSQL 18'dir; CI'da PostgreSQL 17 de çalıştırılır.

Ayrıca bakın: [`examples/v0.1/71_postgresql.ahd`](../examples/v0.1/71_postgresql.ahd) ·
[`examples/v1.4/realtime_attendance`](../examples/v1.4/realtime_attendance/README_TR.md).
