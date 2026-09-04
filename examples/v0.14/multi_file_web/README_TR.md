# v0.14 çok dosyalı web örneği

AhdCode v0.14'ün uygulama temellerini gösteren küçük, iki sayfalık bir site:
`require(...)` ile yerel kaynak birleştirme, bağımlılık farkında
`ahdcode dev` ve güvenli statik dosya sunumu.

```
app.ahd
Components/
    Layout.ahd       -- sayfa iskeleti; Shared/HTMLHelpers.ahd ve
                         Components/Navigation.ahd'yi require eder (iç içe
                         require)
    Navigation.ahd    -- gezinme çubuğu
Pages/
    Home.ahd
    About.ahd
Shared/
    HTMLHelpers.ahd   -- küçük, yeniden kullanılabilir HTML düğüm yardımcıları
public/
    app.css           -- Server.static üzerinden sunulur, derlenmiş bir rota
                         değil
```

## Çalıştırma

```bash
cd examples/v0.14/multi_file_web
ahdcode dev app.ahd
```

[http://127.0.0.1:8095/](http://127.0.0.1:8095/) adresini açın.
`ahdcode stop app.dev` temiz bir şekilde durdurur.

## Denenecekler

- `ahdcode dev` çalışırken `Components/Navigation.ahd`'yi düzenleyin (bir
  bağlantı ekleyin): `app.ahd`'nin kendisi hiç değişmemiş olsa bile otomatik
  olarak yeniden derlenir ve yeniden başlar -- `dev`, yalnızca giriş
  dosyasını değil, çözümlenmiş tüm `require(...)` grafiğini izler.
- Çalışırken `public/app.css`'i düzenleyin: uygulama yeniden derlenmez, ama
  bir tarayıcı yenilemesi (veya
  `curl http://127.0.0.1:8095/assets/app.css`) yeni içeriği hemen görür.
  Statik dosyalar doğrudan diskten sunulur.
- `Pages/About.ahd`'ye bir sözdizimi hatası ekleyin: `ahdcode dev` bunu
  bildirir ve önceden çalışan uygulamayı çalışır durumda tutar; dosyayı
  düzeltmek otomatik olarak kurtarır.

## Bu örneğin sınadığı require(...) kuralları

- **Her zaman uygulama köküne göreli.** `Components/Layout.ahd`,
  `Layout.ahd` kendisi bir dizin aşağıda yaşasa bile `app.ahd`'nin
  kullanacağı ile TAM AYNI yolları kullanarak `"Shared/HTMLHelpers.ahd"` ve
  `"Components/Navigation.ahd"`'yi require eder -- her require yolu,
  isteyen dosyaya göre değil, her zaman uygulama köküne (app.ahd'nin kendi
  dizini) göre çözümlenir.
- **Tek paylaşılan ad alanı.** `Pages/Home.ahd`, ne biri o dosyada
  tanımlanmamış olsa da `pageShell(...)` ve `paragraph(...)`'ı doğrudan,
  niteliksiz olarak çağırır -- her require edilen dosyanın üst düzey
  bildirimleri tek bir uygulama çapında ad alanını paylaşır.
- **Dosyaya özgü `bring`.** `Pages/Home.ahd` ve `Pages/About.ahd`,
  `Components/Layout.ahd` zaten aynı modülü getirse bile kendi
  `bring HTML` / `from HTML bring HTMLNode`'unu ayrı ayrı bildirir: require
  edilen bir dosyanın `bring`'i başka bir dosyaya asla sızmaz veya başka bir
  dosyadan miras alınmaz.
