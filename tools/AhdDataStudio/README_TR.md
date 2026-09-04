# AhdDataStudio

AhdDataStudio **yalnızca bu makinede** çalışan bir veritabanı geliştirme
uygulamasıdır. Derleyici yerleşik bir modülü, bir ORM veya bir veritabanı
soyutlama katmanı değildir. Yayınlanmış `HTTP`, `HTML`, `MySQL`, `SQLite`,
`Env`, `File`, `Path` ve `Security` modüllerini kullanan birinci taraf bir
AhdCode programıdır.

## Başlatma

```bash
cd tools/AhdDataStudio
cp .env.example .env   # ardından yer tutucuları düzenleyin
ahdcode run app.ahd
```

Adres:

[http://127.0.0.1:8081/AhdDataStudio](http://127.0.0.1:8081/AhdDataStudio)

Sunucu yalnızca **127.0.0.1:8081** dinler. Bu bir yerel geliştirme aracıdır;
`0.0.0.0` veya internete açılmamalıdır.

Bu sürümde `ahdcode studio` komutu yoktur.

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

- `AHD_DATA_SQLITE_PATHS` içinde (virgülle ayrılmış), ve/veya
- `AHD_DATA_PROJECT_ROOT` altındaki **hemen** `.db` / `.sqlite` /
  `.sqlite3` çocukları

Makine geneli tarama yoktur, özyinelemeli dolaşma yoktur, parola dosyası
araması yoktur. Sorgu/form yolu yalnızca bu izin listesiyle eşleşirse
kabul edilir. `..` bileşenleri ve dizin hedefleri reddedilir.

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
