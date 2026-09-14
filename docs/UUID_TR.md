# UUID standart modülü

[English](UUID.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Identity](IDENTITY_TR.md) · [Security](SECURITY_TR.md) · [PostgreSQL](POSTGRESQL_TR.md)

`UUID`, AhdCode v1.4.0 ile gelen ve derleyicide kayıtlı olan `builtin:UUID`
modülüdür. RFC 9562 UUID'lerini oluşturur, ayrıştırır ve karşılaştırır:
rastgele sürüm 4 değerleri ve zamana göre sıralı sürüm 7 değerleri. Yalnızca Go
standart kütüphanesiyle yazılmıştır ve `ahdcode run`, yerel derlemeler ve REPL
içinde aynı davranır.

## Genel arayüz

```text
bring UUID
from UUID bring (UUIDValue, UUIDError)

UUID.v4()                  -> UUIDValue
UUID.v7()                  -> UUIDValue
UUID.parse(text: String)   -> UUIDValue
UUID.isValid(text: String) -> Bool
UUID.zero()                -> UUIDValue

UUIDValue.string()                  -> String
UUIDValue.version()                 -> Int
UUIDValue.isZero()                  -> Bool
UUIDValue.equals(other: UUIDValue)  -> Bool
UUIDValue.compare(other: UUIDValue) -> Int

UUIDError  (Error'dan türer)
```

`UUIDValue` opak ve değiştirilemez yerleşik bir sınıftır. Kurucusu yoktur ve
`String`'e ya da `String`'den örtük dönüşümü yoktur.

## Hangi kimliği kullanmalı

| | `Identity.id()` | `Security.token()` | `UUID.v4()` | `UUID.v7()` |
| --- | --- | --- | --- | --- |
| Amaç | opak genel kimlik | gizli kimlik bilgisi | birlikte çalışabilir genel kimlik | zamana göre sıralı genel kimlik |
| Rastgelelik | 128 rastgele bit | 256 rastgele bit | 122 rastgele bit | en az 62 rastgele bit ve milisaniye |
| Metin | 22 karakter, base64url | 43 karakter, base64url | 36 karakter, küçük harfli onaltılık | 36 karakter, küçük harfli onaltılık |
| UUID uyumlu | hayır | hayır | evet | evet |
| Gizli | hayır | evet | hayır | hayır; oluşturulma zamanını gösterir |

- AhdCode'un kendi genel kimlikleri için [`Identity.id()`](IDENTITY_TR.md)
  kullanın. v1.4.0'da değişmedi ve Web başlangıç şablonları hâlâ onu kullanır.
- Gizli kalması gereken her şey için [`Security.token()`](SECURITY_TR.md)
  kullanın: oturumlar, parola sıfırlama bağlantıları, API token'ları. Bir UUID
  asla gizli değildir.
- Başka bir sistem UUID beklediğinde `UUID.v4()` ya da `UUID.v7()` kullanın;
  örneğin bir PostgreSQL `uuid` sütunu ya da harici bir API.

Bu değer türleri arasında dönüşüm yoktur.

## UUID oluşturma

```ahd
bring UUID
from UUID bring UUIDValue

random: UUIDValue := UUID.v4()
ordered: UUIDValue := UUID.v7()
write(ordered.string())      // 01a0a0c9-8fa6-733f-9f09-81191abcce97
write(ordered.version())     // 7
```

`v4`, 122 biti işletim sisteminin kriptografik rastgele kaynağından alır.

### Sürüm 7 sıralaması

Bir `v7` değeri milisaniye cinsinden Unix zamanı ve milisaniyenin altında bir
kesirle başlar, ardından 62 rastgele bit gelir.

- **Söz verilen:** tek bir süreç içinde her `UUID.v7()` bir öncekinden
  büyüktür; hem `compare` ile hem metin sırasıyla. Sistem saati geriye gitse
  bile bu geçerlidir: gömülü zaman, saat yetişene kadar verilen son değerden
  devam eder.
- **Söz verilmeyen:** aynı milisaniye içinde iki süreç ya da makine arasındaki
  sıra, tam oluşturulma zamanı ve tahmin edilemezlik. Bir `v7` değerini gören
  herkes aşağı yukarı ne zaman oluşturulduğunu okuyabilir.
- Milisaniyede 4096'dan fazla değer sürekli üretilirse gömülü zaman, üretim
  yoğunluğu kadar saatin önüne geçer.

Yeni değerler sonra sıralandığı için `v7` birincil anahtarı, dizine eklemeleri
dizinin sonuna yakın tutar.

## Ayrıştırma ve doğrulama

```ahd
bring UUID
from UUID bring (UUIDValue, UUIDError)

id: UUIDValue := UUID.parse("017F22E2-79B0-7CC3-98C4-DC0C0C07398F")
write(id.string())                                   // 017f22e2-79b0-7cc3-98c4-dc0c0c07398f
write(UUID.isValid(r"{017f22e2-79b0-7cc3-98c4-dc0c0c07398f}"))  // false

attempt {
    UUID.parse("017f22e279b07cc398c4dc0c0c07398f")
}
except UUIDError as error {
    write(error.message)
}
```

UUID metni tam 36 karakterdir: `-` ile ayrılmış 8-4-4-4-12 gruplarında
onaltılık rakamlar. Girişteki rakamlar büyük ya da küçük harf olabilir; çıktı
her zaman küçük harftir. Çevreleyen boşluklar, süslü parantezler, `urn:uuid:`
ön eki ve tiresiz 32 rakamlı biçim reddedilir. Sürümü ya da varyantı ne olursa
olsun, sıfır UUID dahil her 128 bitlik değer ayrıştırılır.

`parse`, `UUIDError` verir ve reddedilen metni mesajda asla tekrarlamaz.
`isValid` hiçbir zaman hata vermez.

Sıradan bir AhdCode String'i içindeki `{` karakteri araya değer eklemeyi
başlatır. Süslü parantez içeren metni yukarıdaki gibi ham String olarak,
`r"{…}"` biçiminde yazın.

## Karşılaştırma

```ahd
a := UUID.parse("017f22e2-79b0-7cc3-98c4-dc0c0c07398f")
b := UUID.parse("017f22e2-79b0-7cc3-98c4-dc0c0c07398f")
write(a.equals(b))   // true
write(a == b)        // false: iki farklı UUIDValue nesnesi
write(a.compare(b))  // 0
write(str(a))        // <UUIDValue>
```

- `equals` 128 bitin tamamını karşılaştırır.
- `compare`, RFC 9562 bayt sırasına göre `-1`, `0` ya da `1` döndürür; bu,
  küçük harfli metni karşılaştırmakla aynıdır.
- `==`, her yerleşik sınıftaki kimlik anlamını korur; değerleri karşılaştırmak
  için `equals` kullanın.
- `str(id)`, diğer yerleşik sınıflar gibi `<UUIDValue>` olur. Metin için
  `id.string()` çağırın.
- `version()`, yazıldığı hâliyle 4 bitlik sürüm alanıdır (`0..15`). `isZero()`
  yalnızca `UUID.zero()`'nun döndürdüğü
  `00000000-0000-0000-0000-000000000000` için doğrudur.

## UUID saklama

36 karakterlik metni saklayın. [PostgreSQL](POSTGRESQL_TR.md) ile bir `uuid`
sütununa dönüşümle bağlayın ve geri okurken ayrıştırın:

```ahd
db.execute(
    "INSERT INTO check_ins (id, student) VALUES ($1::uuid, $2)"
    [PostgreSQL.fromString(UUID.v7().string()), PostgreSQL.fromString(student)]
)
rows := db.query("SELECT id FROM check_ins ORDER BY id")
first: UUIDValue := UUID.parse(rows[0]["id"].string())
```

MySQL ya da SQLite ile `CHAR(36)` veya `TEXT` sütunu yeterlidir. Metin küçük
harfli ve sabit genişlikte olduğundan, sıralandığında `v7` değerleri
oluşturulma zamanına göre sıralanır.

## Hatalar

Her hata `Error`'dan türeyen `UUIDError`'dır:

```text
UUID text must be 36 characters in the form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
UUID random generation failed
UUID v7 requires a system clock between 1970 and the year 10889
```

## Hedef dışı olanlar

Sürüm 1, 3, 5, 6 ya da 8 üretimi, isim tabanlı UUID'ler, 16 baytlık ikili biçim,
`Identity.id()` ile dönüşüm ve genel bir benzersizlik kaydı yoktur. v1.4.0,
`UUIDValue` için işleç eklemez: karşılaştırma `equals` ve `compare` ile yapılır.

Ayrıca bakın: [`examples/v0.1/69_uuid.ahd`](../examples/v0.1/69_uuid.ahd) ·
[`examples/v1.4/realtime_attendance`](../examples/v1.4/realtime_attendance/README_TR.md).
