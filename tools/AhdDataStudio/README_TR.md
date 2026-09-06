# AhdDataStudio

AhdDataStudio **yalnızca bu makinede** çalışan bir veritabanı geliştirme
uygulamasıdır. Derleyici yerleşik bir modülü, bir ORM veya bir veritabanı
soyutlama katmanı değildir. Yayınlanmış `HTTP`, `HTML`, `MySQL`, `SQLite`,
`Env`, `File`, `Path` ve `Security` modüllerini kullanan birinci taraf bir
AhdCode programıdır.

## Başlatma

```bash
ahdcode databases
```

Adres:

[http://ahddatabasestudio.test/](http://ahddatabasestudio.test/)

Bu ad, `ahdcode databases` çalıştığı sürece çalışan, yalnızca geri döngüyü
dinleyen AhdCode [yerel
yönlendiricisi](../../docs/CLI_TR.md#yerel-geliştirme-test-adları-ve-yönlendirici)
tarafından sunulur. Doğrudan adres her zaman desteklenir ve temiz ad burada
çözülmediğinde CLI zaten onu açar:

[http://127.0.0.1:8081/AhdDataStudio](http://127.0.0.1:8081/AhdDataStudio)

`GET /`, `/AhdDataStudio` adresine yönlendirir; böylece iki biçim de işe yarar
bir yere düşer.

Süreç hâlâ yalnızca **127.0.0.1:8081** dinler. `0.0.0.0` veya internete
açılmamalıdır.

Temiz adın çözülmesi için `ahddatabasestudio.test` geri döngüye eşlenmelidir.
Yerel konak bütünleştirmesi bir kez yetkilendirildikten sonra `ahdcode
databases` bu adı otomatik tutar. İlk uçbirim kullanımı herhangi bir yetki
sorusundan önce onay ister. `ahdcode local hosts apply` elle kurtarma
komutu olarak kalır, ya da satırı kendiniz ekleyebilirsiniz:

```text
127.0.0.1 ahddatabasestudio.test
```

Yönlendirici 80 portuna bağlanamazsa yedek portu kullanır ve adres onu taşır
(`http://ahddatabasestudio.test:7357/`). Hangisi olduğunu
`ahdcode local status` bildirir.

## Durdurma

`ahdcode run`, `app.ahd` yanına `app.run` yazar. Durdurmak için:

```bash
cd tools/AhdDataStudio
ahdcode kill app.run
```

Bu komut uygulamayı durdurur ve `app.run` dosyasını siler. Derlenmiş
ikiliyi kendiniz başlattıysanız o süreci durdurun (`lsof -nP -iTCP:8081
-sTCP:LISTEN` PID'yi gösterir).

## MySQL

Yapılandırma Env (veya bu dizindeki `.env`) iledir:

| Değişken | Varsayılan | Anlam |
|---|---|---|
| `AHD_DATA_MYSQL_HOST` | `127.0.0.1` | MySQL makinesi |
| `AHD_DATA_MYSQL_PORT` | `3306` | MySQL kapısı |
| `AHD_DATA_MYSQL_USER` | `root` | kullanıcı adı |
| `AHD_DATA_MYSQL_PASSWORD` | boş | parola — asla commit edilmez |
| `AHD_DATA_MYSQL_SECURITY` | `none` | `none` veya `tls` |

Bu değişkenler başlangıç değerlerini belirler. Studio çalışırken aynı
değerler **MySQL → Bağlantı ayarları** sayfasından değiştirilebilir; yerel bir
sunucuya bağlanmak için terminal gerekmez:

| Alan | Varsayılan |
|---|---|
| Sunucu | `127.0.0.1` |
| Port | `3306` |
| Kullanıcı adı | `root` |
| Parola | boş |
| Güvenlik | `none` |

**Bağlantıyı Sına** alanları doğrulayıp gerçek bir sunucu bağlantısı açar;
**Bağlan** aynısını yapıp şema listesini açar. Bağlantı `database: null` ile
kurulduğu için veritabanı adı sorulmaz.

Parola yalnızca yazılır. Forma geri basılmaz ve alan boş bırakıldığında
kullanımdaki parola korunur. Kimlik bilgileri yalnızca çalışan Studio
sürecinde tutulur: kayıt defterine, dosyaya veya çereze yazılmaz; Studio
kapanınca unutulur. Başarısız bağlantı, sunucunun söylediğini parola
çıkarılmış olarak bildirir.

Studio hiçbir zaman MySQL kurmaz, hesap oluşturmaz, parola sıfırlamaz veya
yetki değiştirmez. Yalnızca sizin verdiğiniz bilgilerle bağlanır.

Studio `database: null` ile bağlanır, ardından bu kimlik bilgilerinin
görebildiği her şemayı listeler (`SHOW DATABASES` / `INFORMATION_SCHEMA`).
MySQL izinleri korunur; Studio onları aşmaz.

Tablo bilgisi sıradan SQL'den gelir: `INFORMATION_SCHEMA.TABLES`,
`COLUMNS` ve `STATISTICS`. Tarama `LIMIT` 50 kullanır. `TABLE_ROWS`,
MySQL verdiğinde bir tahmindir; kesin `COUNT(*)` değildir.

Studio'nun ürettiği INSERT / UPDATE / DELETE değerleri `?` ile bağlar.
Kimlikler keşfedilmiş metadata'dan alınır ve tırnaklanır. Güvenilmeyen
form metninden ham SQL'e eklenmezler.

## SQLite

SQLite dosyaları yalnızca şu durumlarda görünür:

- **AhdCode veritabanı kayıt defterinde** iseler (`AHD_DATA_REGISTRY`;
  `ahdcode databases` tarafından otomatik ayarlanır), ve/veya
- `AHD_DATA_SQLITE_PATHS` içinde (virgülle ayrılmış), ve/veya
- `AHD_DATA_PROJECT_ROOT` altındaki **hemen** `.db` / `.sqlite` /
  `.sqlite3` çocukları

Üç kaynak bu sabit sırayla birleştirilip yinelenenler ayıklanır; böylece liste
her istekte aynıdır. Makine geneli tarama yoktur, özyinelemeli dolaşma yoktur,
parola dosyası araması yoktur. Sorgu/form yolu yalnızca bu izin listesiyle
eşleşirse kabul edilir. `..` bileşenleri ve dizin hedefleri reddedilir.

Bir ortam değişkenini artık elle düzenlememenizi sağlayan kaynak kayıt
defteridir:

```bash
ahdcode databases add ./database/app.db     # kaydet
ahdcode databases list                      # kayıtlıları gör
ahdcode databases remove ./database/app.db  # unut; dosya korunur
```

`ahdcode init web admin`, oluşturduğu SQLite veritabanını otomatik kaydeder;
böylece yeni üretilmiş bir proje burada zaten görünür.

Kayıt defteri yalnızca meta veri tutar — bir sürücü, bir yol, bir görünen ad.
Asla parola, belirteç veya veritabanı içeriği tutmaz ve **MySQL kimlik
bilgileri bilinçli olarak onun dışında**, bu dizindeki `.env` dosyasında
kalır. Bir kaydı silmek SQLite dosyasına asla dokunmaz. Şu anda mevcut olmayan
kayıtlı bir dosya, düşürülmek yerine "unavailable" olarak gösterilir; çünkü
bağlanmamış bir birim geri çekilme değildir.

Üretilen AhdCode programlarında SQLite, paketlenmiş `ahdsqlite`
yardımcısını kullanır. Yardımcı yoksa ilk `SQLite.open` `SQLiteError`
yükseltir ve `AHDCODE_SQLITE_RUNTIME` yolunu açıklar:

```bash
go install ./cmd/ahdcode ./cmd/ahdsqlite
```

SQLite `BLOB` değerleri Studio'da ayrı bir tür değildir: bir BLOB
sorgusu, yayınlanmış SQLite sözleşmesine uygun olarak `SQLiteError`
yükseltir. Tarama, bildirilen türünde `blob` geçen sütunları atlar.

## SQL konsolu

SQL konsolu yazdığınız SQL'i çalıştırır. Yıkıcı deyimler için ekstra
onay **istenmez**. Bu, yerel bir yönetim aracı için bilinçlidir.

Studio'nun ürettiği formlar (insert/update/delete/drop/truncate) farklıdır:
POST kullanırlar, CSRF jetonu vardır (`Security.token` +
`Security.secureEqual`) ve DROP/TRUNCATE için onay adımı vardır.

## Güvenlik duruşu

- Yalnızca yerel bağlama (`127.0.0.1`)
- Kimlik bilgisi tarama yok
- Hücre değerleri, adlar ve SQL hataları `HTML.text` ile çizilir
- MySQL parolası HTML'e geri yazılmaz
- MySQL bağlantı hatası SQLite'ı kapatmaz; tersi de geçerlidir

## Bugünkü sınırlar (neden sonraki Web işi var)

İlk sürüm tek bir `app.ahd` dosyasıdır; çünkü yerel kaynak `require(...)`
henüz yoktur. Rotalar tam yoldur; Pages/Components katmanı, statik varlık
borusu ve `Web.UI` yoktur. CSS, HTML üreticisinin içindeki güvenilir bir
ham String'dir. Path/File'da `realpath` / sembolik bağ API'si olmadığı
için SQLite keşfi ada dayalı ve özyinelemesizdir.

Bu kısıtlar sonraki bir Web cephesi için ürün kanıtıdır; bu sürümün
özellikleri değildir.

## Dil

[English](README.md) · [Türkçe](README_TR.md)
