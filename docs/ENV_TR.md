# Env standart modülü

[English](ENV.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [File ve Path](FILESYSTEM_TR.md)

`Env`, derleyici tarafından kayıtlı `builtin:Env` modülüdür. Açıktır ve
kardeş bir `Env.ahd` dosyası onu gölgeleyemez:

```ahd
bring Env
from Env bring EnvError
```

Env bilinçli olarak küçük kalır: işlem ortam değişkenleri ve sınırlandırılmış
bir `.env` dosya biçimi, her ikisi de düz `String` olarak, hiçbir sayısal/
boolean çıkarım, shell interpolation veya komut çalıştırma olmadan.

## Yüzey

```text
Env.get(name: String)    -> String?
Env.getOr(name: String, fallback: String) -> String
Env.exists(name: String) -> Bool

Env.set(name: String, value: String) -> Nothing
Env.unset(name: String)  -> Nothing

Env.read(path: String)   -> Pair<String, String>
Env.load(path: String, override: Bool = false) -> Nothing
```

(`Env.has` değil `Env.exists`: `has` saklı bir anahtar kelimedir — `x has y`
protokol operatörü — ve `.` sonrası bir üye adı olarak görünemez; `exists`,
mevcut `File.exists` adlandırmasıyla eşleşir.)

## get, getOr, exists

`Env.get(name)`, `String?` döndürür: `null`, değişkenin yok olduğu anlamına
gelir. `Env.exists(name)`, yokluğu açıkça mevcut ama boş bir değerden ayırt
eder — ikisi de gerçek, farklı durumlardır:

```ahd
found: String? := Env.get("PORT")
if found != null {
    write(found)
}
write(Env.exists("PORT"))
```

`Env.getOr(name, fallback)`, yalnızca değişken yokken `fallback` döndürür;
açıkça mevcut boş bir `String`, fallback ile değiştirilmeden `""` olarak
döndürülür. Otomatik sayısal veya boolean dönüşüm yoktur — açıkça
dönüştürün:

```ahd
port: Int := int(Env.getOr("PORT", "8080"))
```

## set ve unset

`Env.set`/`Env.unset`, mevcut AhdCode işleminin kendi ortamını değiştirir;
sonraki `Env.get`/`Env.exists` çağrıları ve sonrasında başlatılan alt
süreçler, OS semantiğinin izin verdiği yerde değişikliği görür. Bir ad,
herhangi bir şey değiştirilmeden önce doğrulanır: boş olmamalı ve bir NUL
baytı veya `=` içermemelidir. Değerler hata mesajlarında asla loglanmaz.

## `.env` biçimi

```text
KEY=value
KEY="value"
KEY='value'

# tam satır comment

EMPTY=
```

- Bir anahtar `[A-Za-z_][A-Za-z0-9_]*` ile eşleşir.
- Bir satırın başındaki boşluk yok sayılır; ilk boşluk-olmayan karakteri `#`
  olan bir satır tam satır comment'tir; boş bir satır yok sayılır.
- Tırnaksız bir değer, `=`'den satır sonuna kadar olan her şeydir, baştaki ve
  sondaki boşluklar kırpılır.
- Çift tırnaklı bir değer tam olarak `\\`, `\"`, `\n`, `\r` ve `\t`'yi
  destekler; başka herhangi bir kaçış reddedilir. Tek tırnaklı bir değer
  literaldir — açılış tırnağından sonra eşleşen kapanış tırnağına kadar
  hiçbir şey özel olarak işlenmez.
- Tırnaklı bir değerin kapanış tırnağından sonra, sondaki boşluk dışında
  hiçbir şey gelemez.

Bilinçli olarak `$(...)`, `` `...` ``, `${...}`, `$NAME` veya başka herhangi
bir shell-tarzı genişletme yoktur: bir `.env` değeri literal metin olarak
okunur, asla değerlendirilmez ve asla bir süreç başlatmaz.

```ahd
Env.load(".env")
databasePath: String := Env.getOr("DATABASE_PATH", "app.db")
```

## read ve load

`Env.read(path)`, bir dosyayı ayrıştırır ve onu ekleme sıralı bir
`Pair<String, String>` olarak döndürür, işlem ortamına hiç dokunmadan.

`Env.load(path, override = false)`, dosyanın tamamını ayrıştırır, tamamen
doğrular ve yalnızca ondan sonra uygular — bozuk bir sonraki satır, aynı
çağrıdan işlemi asla yarı güncellenmiş bırakamaz. `override = false` ile
(varsayılan), zaten mevcut bir değişken — `exists()`'in kullandığı aynı
yokluk-karşı-boş-farkında şekilde kontrol edilir — dokunulmadan bırakılır:
mevcut işlem/OS ortamı dosyaya karşı kazanır. `override = true` ile, `.env`
değeri orada ne olursa olsun her zaman onun yerini alır.

```ahd
Env.load(".env")            -- mevcut ortam kazanır
Env.load(".env", true)      -- .env her zaman kazanır
```

Bir `.env` dosyası içindeki yinelenen anahtarlar, sonuncunun sessizce
kazanmasına izin vermek yerine `EnvError` ile reddedilir.

## Hatalar

`EnvError` doğrudan `Error`'dan türer ve şunları kapsar: eksik/okunamayan
bir `.env` dosyası, bozuk bir atama, geçersiz bir anahtar, sonlandırılmamış
tırnaklı bir değer, yinelenen bir anahtar, geçersiz bir kaçış dizisi ve bir
OS düzeyinde `set`/`unset` hatası. Hata mesajları asla değişkenin değerini
içermez.

```ahd
attempt {
    Env.load("missing.env")
} except EnvError as error {
    write(error.message)
}
```

## AhdCode araç zincirinin kendi okuduğu değişkenler

Bunlar `Env` modülü tarafından değil, `ahdcode` komut satırı tarafından
okunur. Burada listelenmelerinin nedeni, insanların onlara burada bakmasıdır.

### Uygulama yapılandırması

`ahdcode dev`; `APP_NAME`, `APP_ENV`, `APP_HOST`, `APP_PROTOCOL`,
`SERVER_HOST` ve `SERVER_PORT` değerlerini uygulamanın kendi önceliğiyle
okur — önce süreç ortamı, sonra uygulama kökündeki `.env` — ve yalnızca ne
yazacağına ve oturumun çalışıp çalışamayacağına karar vermek için. Hiçbirini
dışa aktarmaz ve alt sürece geçirmez. Her birinin anlamı için bkz.
[Web](WEB_TR.md).

`APP_HOST` ayrıca bir geliştirme oturumunun yönlendirileceği yerel `.test`
adını belirler: `ahdakademi.com`, `ahdakademi.test` olarak geliştirilir. Bkz.
[CLI](CLI_TR.md#yerel-geliştirme-test-adları-ve-yönlendirici).

### Yerel geliştirme

| Değişken | Okuyan | Anlamı |
| --- | --- | --- |
| `AHDCODE_LOCAL_HOME` | CLI | Rota ve veritabanı kayıt defterlerini tutan kullanıcıya özel dizini değiştirir. Mutlak olmalıdır. Olağan kullanımda ayarlanmaz. |
| `AHDCODE_LOCAL_ROUTER_PORT` | CLI | 80 portuna bağlanılamadığında yerel yönlendiricinin yedek portu. Varsayılan `7357`. 1–65535 dışındaki bir değer ölümcül değildir, yok sayılır. |
| `AHDCODE_ROOT` | CLI | Geliştirici geçersiz kılması: paketlenmiş sürüm-eş Studio yerine bu kaynak ağacındaki `tools/AhdDataStudio` kullanılır. Olağan kurulumda ayarlanmaz. |
| `AHDCODE_STUDIO_CACHE` | CLI | Materialize edilmiş Studio önbelleğinin mutlak dizini. Test ve paketleme içindir; olağan kullanımda ayarlanmaz. |
| `AHDCODE_SQLITE_RUNTIME` | çalışma zamanı | `ahdcode` ile birlikte kurulu değilse paketlenmiş `ahdsqlite` yardımcısının yolu. |

### AhdDataStudio

| Değişken | Anlamı |
| --- | --- |
| `AHD_DATA_REGISTRY` | AhdCode veritabanı kayıt defterinin yolu. `ahdcode databases` tarafından otomatik ayarlanır; Studio kayıtlı SQLite veritabanlarını buradan okur. |
| `AHD_DATA_SQLITE_PATHS` | Virgülle ayrılmış açık SQLite dosyaları. Desteklenmeye devam eder ve artık elle düzenlenmesi gereken bir şey değildir — bkz. `ahdcode databases add`. |
| `AHD_DATA_PROJECT_ROOT` | **Hemen** altındaki `.db`/`.sqlite`/`.sqlite3` dosyaları listelenen klasör. Özyinelemeli değildir ve bu kökün dışına asla çıkılmaz. |
| `AHD_DATA_MYSQL_HOST` | Yalnızca MySQL sunucu adresi (`127.0.0.1`); asla `host:port` değil. |
| `AHD_DATA_MYSQL_PORT` | MySQL portu. |
| `AHD_DATA_MYSQL_USER`, `AHD_DATA_MYSQL_PASSWORD` | MySQL kimlik bilgileri. Bunlar Studio'nun `.env` dosyasında yaşar ve bilinçli olarak **hiçbir** AhdCode kayıt defterinde bulunmaz. |
| `AHD_DATA_MYSQL_SECURITY` | `tls` veya `none`. |

Studio üç SQLite kaynağını — kayıt defteri, `AHD_DATA_SQLITE_PATHS`,
`AHD_DATA_PROJECT_ROOT` — bu sabit sırayla birleştirip yinelenenleri ayıklar.
Özyinelemeli veya makine geneli hiçbir tarama yapılmaz.

### Web çalışma zamanı sınırları

`Web.app` bunları HTTP sunucusu oluşturulduktan sonra uygular. Ayarlanmamış
değer varsayılanı kullanır. Bozuk bir değer süreci durdurur; sessizce yok
sayılmaz.

| Değişken | Varsayılan | Anlamı |
| --- | --- | --- |
| `AHD_WEB_MAX_BODY_SIZE` | `16MB` | En büyük istek gövdesi. Birimler 1024 tabanlıdır (`B`, `KB`, `MB`, `GB`). |
| `AHD_WEB_MAX_UPLOAD_SIZE` | `8MB` | Bir yüklenen dosyanın en büyük boyutu. `AHD_WEB_MAX_BODY_SIZE`'ı aşamaz. |
| `AHD_WEB_MAX_UPLOAD_FILES` | `10` | Bir istekteki en fazla yüklenen dosya sayısı. |
| `AHD_WEB_READ_TIMEOUT` | `30s` | Sunucu okuma zaman aşımı (`ms`, `s`, `m`). |
| `AHD_WEB_WRITE_TIMEOUT` | `30s` | Sunucu yazma zaman aşımı. |
| `AHD_WEB_IDLE_TIMEOUT` | `60s` | Sunucu boşta zaman aşımı. |

Ham `HTTP.server`, `applyWebLimits()` çağrılana kadar tarihsel 1MiB gövde
ve 15s/15s/60s zaman aşımlarını korur.
