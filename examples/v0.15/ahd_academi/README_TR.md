# Ahd Akademi — AhdCode v0.15 Web örneği

[English](README.md) · [Türkçe]

Küçük ama gerçek bir Web uygulaması: tek bir içe aktarma, bir config katmanı,
bir yerleşim, iki sayfa, iki bileşen, GET ve POST rotaları ve statik CSS.

## Çalıştırma

```bash
cp .env.example .env
ahdcode dev app.ahd
```

Ardından `http://127.0.0.1:8080/` adresini açın.

`ahdcode dev` kanonik geliştirme adresini yazar
(`http://ahdakademi.com.test`). `.test`, `APP_HOST`'un yerel kimliğini
adlandırır; v0.15 bu adresi kendiliğinden açan çözücüyü getirmediği için
şimdilik geri döngü adresini ve portunu kullanın. Bkz.
[docs/WEB_TR.md](../../../docs/WEB_TR.md#14-yerel-https--mevcut-sınır).

Durdurmak için `ahdcode stop app.dev`.

## Neyi gösterir

```
app.ahd            bring Web, rotalar, statik varlıklar, start
.env.example       ortam sözleşmesi, sır yok
Config/App.ahd     ortamı okuyan tek dosya
Layouts/Main.ahd   ortak belge kabuğu
Pages/Home.ahd     GET / ve POST /selam
Pages/About.ahd    GET /hakkinda
Components/        navbar ve notice, sıradan Function'lar
public/            app.css ve logo.svg, diskten sunulur
```

- **`bring Web`** içe aktarmanın tamamıdır. `bring HTTP` yok, `bring HTML`
  yok.
- **Config katmanı.** `Config/App.ahd` `Web.configure()`'ı bir kez çağırır ve
  `configuration()` ile sunar. Hiçbir sayfa `Env`'e uzanmaz.
- **Sayfalar, Yerleşimler, Bileşenler** `HTMLNode` veya `Response` döndüren
  sıradan Function'lardır. Taban sınıf, yaşam döngüsü, kayıt defteri yoktur.
- **`Web.UI`** ağacı kurar: `nav`, `main`, `section`, `h1`, `h2`, `p`, `a`,
  `img`, `ul`, `li`, `table`, `formTo`, `labelFor`, `input`, `button`,
  `footer`.
- **Bir POST rotası** alanı, yayınlanmış düşük seviyeli `request.form("isim")`
  ile okur; bu, açıkça ele alınan boş değer alabilir bir String döndürür.
  v0.15'te Form çatısı yoktur.
- **Kaçışlama.** Selam formuna `<script>alert(1)</script>` yazın; sayfa bu
  karakterleri metin olarak gösterir.
- **`require(...)`** her dosyayı uygulama kökünden bileştirir.
- **Statik varlıklar.** `public/app.css`'i düzenlemek, AhdCode kaynağını
  yeniden derlemeden bir sonraki istekte yeni baytları sunar. Herhangi bir
  `.ahd` dosyasını düzenlemek yeniden derler ve yeniden başlatır.

## Ortamlar

```bash
# development: yerel kimlik .test kazanır
APP_ENV=development APP_HOST=ahdakademi.com   →  ahdakademi.com.test

# production: APP_HOST aynen, asla .test değil
APP_ENV=production  APP_HOST=ahdakademi.com   →  ahdakademi.com
```

`ahdcode dev`, bir production yapılandırmasını development anlambilimiyle
çalıştırmak yerine `APP_ENV=production`'ı reddeder.

Genel URL ile bağlanma adresi ayrıdır: `APP_PROTOCOL`/`APP_HOST` bir insanın
yazdığını, `SERVER_HOST`/`SERVER_PORT` bu sürecin bağlandığını söyler. Ters
vekil arkasında bunlar tasarım gereği farklıdır.

## Notlar

Gerçek bir `.env`'i işlemeyin. `.env.example` yalnızca adlar ve yer tutucular
taşır; `ahdcode build` ikisini de asla gömmez — üretilen çalıştırılabilir
dosya yapılandırmasını başlangıçta ortamdan okur.
