# CLI

[English](CLI.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Formatter](FORMATTER_TR.md) · [REPL](REPL_TR.md) · [Dil sunucusu](LSP_TR.md)

Mevcut komut yüzeyi (command surface) şudur:

```text
ahdcode
ahdcode init web [empty|basic|admin|mvc|crud]
ahdcode pwa init [--name NAME] [--short-name NAME]
ahdcode databases
ahdcode databases list
ahdcode databases add <file.db>
ahdcode databases remove <file.db>
ahdcode migrate new <name> [--dir <directory>]
ahdcode migrate status --db <file> [--dir <directory>] [--driver sqlite]
ahdcode migrate up --db <file> [--dir <directory>] [--driver sqlite]
ahdcode migrate down --db <file> [--dir <directory>] [--steps <n>] [--driver sqlite]
ahdcode build <entry.ahd> [-o <output>]
ahdcode check <entry.ahd> [--format text|json]
ahdcode package <entry.ahd> [--name <name>] [--output <folder>] [--icon <icon.png>]
                [--target <os-arch>] [--helpers <folder>] [--console] [--web [--pwa]]
ahdcode run <entry.ahd> [-- <args>...]
ahdcode dev [--https [--cert FILE --key FILE]] <entry.ahd>
ahdcode stop <app.dev|app.run>
ahdcode kill [--force] <app.dev|app.run>
ahdcode local status
ahdcode local hosts [apply|remove]
ahdcode format [--check] <file.ahd|folder>...
ahdcode lsp
ahdcode --help
ahdcode --version
```


`ahdcode init web` **bulunulan dizine** bir Web starter yazar. Bir TTY'de
Empty, Basic, Admin, MVC veya CRUD sorar. Etkileşimsiz kullanım `empty`,
`basic`, `admin`, `mvc` veya `crud` geçmelidir. Şablonlar, Bootstrap 5.3.3
ve AhdCode logosu gömülüdür: çevrimdışı, paket yöneticisi yok, üzerine
yazma yok. Her starter bu sürüme ait `AHDCODE.md` ve İngilizce
`Documents/AhdCode/` kopyasını da alır. Sonraki adım:
`ahdcode dev app.ahd`.

`package` (v2.0), bir giriş modülünden kendi başına çalışan bir masaüstü
uygulaması yapar: derlenmiş programı ve yalnızca kullandığı AhdCode
yardımcılarını içeren bir macOS `.app`'i ya da bir Windows veya Linux klasörü
ve arşivi; böylece AhdCode kurulu olmadan çalışır. Bkz.
[Paketleme](PACKAGING_TR.md).

`run`, normal önyüz (frontend) ve Go arkayüzünden (backend) derler, ardından
yerel (native) sonucu çalıştırır. Giriş dosyasından sonraki argümanlar
(isteğe bağlı olarak `--`'den sonra) oluşturulan sürece iletilir; ancak v0.1
henüz dil düzeyinde bir argüman API'si yayınlamaz.

`run` çalışırken, giriş modülünün yanında küçük bir `app.run` tanımlayıcısı
tutar (`app.ahd` aynı dizinde `app.run` üretir) ve çalışma bittiğinde onu
siler. `kill` bu tanımlayıcıyı kullanarak uygulamayı durdurur:

```bash
ahdcode run app.ahd
ahdcode kill app.run
```

Bu, süreci `lsof -i :8080` ile portundan bulup sonra `kill <pid>` çalıştırma
alışkanlığının yerini alır. `kill` nazik bir durdurma ister;
`ahdcode kill --force app.run` uygulamayı hemen durdurur.

**`kill`, run dosyasında yazan süreç kimliğine asla sinyal göndermez.** Bir
dosyadaki süreç kimliği hiçbir şey kanıtlamaz: dosyayı yazabilen herkes
ilgisiz bir süreci adlandırabilir ve işletim sistemleri kimlikleri yeniden
kullanır; bu yüzden bayat bir tanımlayıcı zamanla bambaşka bir şeyi
adlandırabilir. Bunun yerine, canlı bir `ahdcode run` yalnızca loopback'e
bağlı bir kontrol portunu dinler ve 256 bitlik rastgele bir jeton tutar;
tanımlayıcı da ona nasıl ulaşılacağını kaydeder. `kill`, `127.0.0.1`
üzerinden o porta bağlanır, jetonu sunar ve çalışan süpervizör, kendi
başlattığı ve sahibi olduğu çocuk süreci sonlandırır.

Sonuçları asıl meseledir:

- ilgisiz, canlı bir süreci adlandıran sahte bir tanımlayıcı hiçbir şeyi
  durdurmaz, çünkü onun adına yanıt veren bir süpervizör yoktur;
- yeniden kullanılmış bir süreç kimliği aynı nedenle zararsızdır;
- yanlış bir jeton reddedilir ve hiçbir şey durdurulmaz;
- canlı süpervizörü olmayan bir tanımlayıcı, hiçbir sürece sinyal
  gönderilmeden bayat olarak bildirilip silinir;
- düzgün biçimli bir AhdCode run tanımlayıcısı olmayan bir dosya — çıplak bir
  pid dahil — doğrudan reddedilir.

`--force` yalnızca süpervizörün kendi çocuğunu nasıl sonlandırdığını
değiştirir; dosyadan doğrudan sinyal göndermeyi asla geri getirmez. Bir
tanımlayıcının süpervizörü hâlâ yanıt verirken ikinci bir `run` başlatmak,
portta sessizce çakışmak yerine pid'i ve kullanılacak `kill` komutunu
bildirerek başarısız olur; süpervizörü gitmiş bir tanımlayıcı ise yeni
çalıştırma sürebilsin diye temizlenir.

Tanımlayıcı dahili CLI meta verisidir, dil düzeyinde bir biçim değildir:
standart kütüphanede onu okuyan ya da yazan hiçbir şey yoktur ve bir kontrol
yetkisi taşıdığı için `0600` izinle yazılır.

`build`, üretilen çalıştırılabilir dosyanın yolunu yazdırır. `-o` olmadan,
derleyici geçerli çalışma dizininde giriş modülünün temel (base) adını
kullanır.

### `dev`: izle, yeniden derle, yeniden başlat

`dev`, `build` ve `run`'ı önplanda bir izleme döngüsünde çalıştırır — bir
MAMP/Vite geliştirme sunucusu gibi — tamamen mevcut derleme hattının
üzerine kurulu bir orkestrasyon olarak; ikinci bir derleyici değildir:

```bash
ahdcode dev app.ahd
```

Giriş modülünü derler, sonucu başlatır ve ardından onu izler. Her kayıtta
yeniden derler:

- yeniden derleme **başarılı** olursa, önceden çalışan süreç durdurulur ve
  yenisi onun yerini alır;
- yeniden derleme **başarısız** olursa, tanılamalar olduğu yerde yazdırılır
  ve önceden çalışan (son-iyi) süreç dokunulmadan çalışmaya devam eder —
  bozuk bir kayıt, ilk derleme dahil, çalışan bir oturumu asla düşürmez;
- çalışan süreç başarılı bir derlemeden sonra kendiliğinden çıkarsa
  (örneğin bir çalışma zamanı çökmesi), `dev` bunu bildirir ve bir sonraki
  kaydı beklemeye döner; aynı bozuk ikiliyi yeniden deneyerek döngüye
  girmez.

`dev` ile başlatılan süreç, başlamadan önce `AHDCODE_DEV_SESSION` ve
`AHDCODE_DEV_GENERATION` alır. HTTP çalışma zamanı bu çifti bir kez, süreç
başlarken okur. `Env.load`, `Web.configure` ve `Web.start` daha sonra
çalışır; bir `.env` dosyası kancayı açamaz ve `dev`'in koyduğu oturumu
değiştiremez. `ahdcode run` bu çifti eklemez.

Üretim uyarısı: tek başına çalışan bir süreç, başlangıç ortamında geçerli
bir oturum ve nesil varsa kancayı açar. `AHDCODE_DEV_SESSION` ve
`AHDCODE_DEV_GENERATION` değerlerini bir üretim kabuğuna, servisine veya
kapsayıcısına vermeyin. İkili, geri döngü istemcilerine geliştirme betiğini
ekler. Uygulama yapılandırması süreç başladıktan sonra bunu yapamaz;
başlangıçta duran bir ortam yapabilir.

`bring Web` kullanan bir programın ilk başarılı başlangıcı, Web başlığında
`Browser refresh:` bölümünü yazar ve ilk `✓ Running` satırından sonra
`✓ Browser will refresh` yazmaz. Sonraki başarılı bir yeniden derleme
şunu yazar:

```
→ Restarting...
✓ Running
✓ Browser will refresh
```

Yalnızca geri döngü istemcisi için, ve uygulama bu yola herhangi bir yöntem
kaydetmemişse,
süreç `GET /__ahdcode/dev/reload` adresini `oturum nesil` olarak ve
`GET /__ahdcode/dev/reload.js` adresini aynı kökenden bir betik olarak
yanıtlar. Uygun HTML, satır içi betik yerine
`<script src="/__ahdcode/dev/reload.js" ...>` alır; `script-src 'self'` onu
çalıştırabilir. Etiket, sayfayı yazan oturumu ve nesli taşır. Aynı anda tek
sorgu çalışır. Farklı bir çift sayfayı bir kez yeniler. Başarısız bir
derleme aynı süreci tutar, çift değişmez ve son sayfa kalır. `ahdcode dev`
durdurulup yeniden başlatılınca yeni bir oturum oluşur; açık kalan sekme,
nesil `1` olduğu için eski sayfada kalmaz.

Betik, UTF-8 konumlarını kaydırmadan bulunan son `</body>` etiketinin
öncesine konur; yanıt bir HTML parçasıysa sona eklenir. 1xx veya 3xx
yanıta, `identity` dışında bir `Content-Encoding` değerine, bir eke, ya da
UTF-8, US-ASCII, ISO-8859-1, ISO-8859-15, Windows-1252 veya Windows-1254
dışındaki bir karakter kümesine eklenmez. Karakter kümesi yoksa uyumlu
sayılır. `withHeader` sonrasındaki son `Content-Type` karar verir.
`/__ahdcode/dev/reload` adresini yalnızca anan bir sayfa yine uygundur.
`HTTP.file` ve `server.static` yeniden yazılmaz. Geri döngü olmayan bir
istemci sıradan HTML'i değiştirilmeden alır: betik yok, oturum kimliği yok
ve geliştirme önbellek başlığı yok. Sondağı ve betik dosyasını da almaz.
Uygulama bu yollardan birine herhangi bir yöntemle kayıt yaptıysa o kayıt
her istemci için durur ve geliştirme işleyicisi yolu yanıtlamaz.

Yenileme tam sayfa yüklemesidir, sıcak modül değişimi değildir; form
girdisini ve kaydırma konumunu korumaz. Statik dosyalar izleme kümesinin
dışındadır, bu yüzden birini düzenlemek yeniden derlemez ve tarayıcıyı
yenilemez.

Yeni süreç portuna bağlanamazsa, HTTP sunucusunun kendi bağlanma hatasıyla
çıkar ve `dev` sonraki kaydı bekler. Yeni başarısız olan süreçten dinleyen
bir şey kalmaz. Ctrl+C ve `ahdcode stop` hem çocuğu hem izleyiciyi
durdurur. İlk oturumun tanımlayıcısı dururken aynı girişe ikinci bir `dev`
reddedilir.

Kayıtlar debounce edilir (~150-300ms), böylece bir editörden gelen ardışık
yazma patlaması tek bir yeniden derlemeye dönüşür, birkaçına değil; ve aynı
anda yalnızca bir derleme çalışır.

`run` gibi, canlı bir `dev` oturumu da giriş modülünün yanında küçük bir
tanımlayıcı tutar — `app.ahd`, `app.dev` üretir — kendi doğrulanmış
loopback kontrol kanalı üzerinden, oturum başlar başlamaz (ilk derleme
bitmeden önce bile) yayınlanır; böylece her zaman durdurulabilir ve aynı
kaynağa karşı ikinci bir `dev`, sessizce yarışmak yerine her zaman
saptanır. Temiz bir şekilde bitirmek için Ctrl+C'ye basın veya başka bir
yerden `ahdcode stop app.dev` çalıştırın.

#### Dev izleme kapsamı

`dev`; giriş dosyasını, derleyicinin çözümlenmiş
[`require(...)`](REQUIRE_TR.md) grafiğini ve en son derleme denemesinin
adlandırdığı ama henüz bulamadığı herhangi bir `require(...)` hedefini
izler — asla özyinelemeli, proje çapında bir tarama değil. İzlenen küme,
başarılı ya da başarısız her derleme denemesinden sonra yeniden hesaplanır,
bu yüzden:

- require edilen herhangi bir dosyayı düzenlemek (ne kadar derinlemesine iç
  içe olursa olsun) giriş dosyasını düzenlemekle aynı şekilde yeniden
  derler ve yeniden başlatır;
- önceden eksik olan require edilen bir dosyayı oluşturmak, onu require
  eden dosyaya başka bir düzenleme gerekmeden otomatik olarak yeniden
  derler;
- `require(...)` grafiğinden düşen bir dosya (`require(...)` satırı
  kaldırılan) izlenmeye devam etmez.

[`server.static`](HTTP_TR.md#statik-dosyalar) üzerinden sunulan statik
varlıklar hiçbir zaman bu grafiğin parçası değildir: birini düzenlemek asla
yeniden derlemeyi tetiklemez, çünkü statik dosyalar her istekte doğrudan
diskten okunur. Bu grafiğin izlediği birleştirme kuralları için bkz.
[`require(...)`](REQUIRE_TR.md).

`ahdcode dev` olmadan başlayan bir program yenileme betiğini eklemez.

Gömülü birinci taraf modüller de izlenmez. `bring Web`, derleyiciye gömülü
kaynaktan derlenir; diskte değişecek bir dosya yoktur.

#### Dev ve Web uygulamaları

Derlenen modül çizgesi birinci taraf [`Web`](WEB_TR.md) çatısını içerdiğinde
`dev`; uygulamayı, bağlandığı soketi ve bu makinenin artık ona yönlendirdiği
yerel adı adlandıran bir başlık ekler:

```
AhdCode Web
  Ahd Akademi (development)

  Open:
  http://127.0.0.1:8080

  Browser refresh:
  Open pages reload after a successful rebuild.
  A failed build keeps the last page.

  Local identity:
  http://ahdakademi.test/
  Bind: 127.0.0.1:8080
```

`Open:` altındaki adres `SERVER_HOST` ve `SERVER_PORT`'tan kurulur —
uygulamanın gerçekten bağlandığı soket — ve her zaman çalışır. Geri
döngüyü varsaymaz, yapılandırılmış konağı izler; joker bir bağlanma adresi
(`0.0.0.0`), gerçekten erişilebilir olduğu geri döngü adresi olarak gösterilir.

`Local identity:` altındaki satır, `APP_HOST`'tan türetilen ve
[yerel yönlendirici](#yerel-geliştirme-test-adları-ve-yönlendirici)
tarafından sunulan `.test` adıdır. İkisi bilinçli olarak ayrı bildirilir:
bağlanma adresi soketin nerede olduğunu, yerel adres ise bu makinenin ona
hangi adı yönlendirdiğini söyler; ikisini tek bir "URL" hâline getirmek, port
değiştiği anda sessizce yanlış olurdu. `APP_ENV=test`, `APP_HOST`'u
değiştirmeden kullanır; bu yüzden hiç kimlik satırı almaz.

#### Yerel ad nasıl türetilir

Kaydedilebilir sonek eklenmez, değiştirilir:

| `APP_HOST`                | yerel ad                |
| ------------------------- | ----------------------- |
| `ahdakademi.com`          | `ahdakademi.test`       |
| `ahdakademi.com.tr`       | `ahdakademi.test`       |
| `example.co.uk`           | `example.test`          |
| `www.example.com`         | `www.example.test`      |
| `admin.ahdakademi.com.tr` | `admin.ahdakademi.test` |
| `localhost`               | `localhost.test`        |
| `ahdakademi.test`         | `ahdakademi.test`       |

Çok etiketli sonek bütün olarak düşürülür: `ahdakademi.com.tr`,
`ahdakademi.com` ile aynı projedir; bu yüzden ikisi de `ahdakademi.test`
adresinde geliştirilir. İkinci etiket yalnızca iki harfli bir ülke kodunun
altında ve yalnızca bir kayıt etiketiyse (`com`, `co`, `org`, `edu`, `gov` ve
benzerleri) düşürülür; bu yüzden `www.example.com` `www` etiketini,
`admin.checkmate.tr` de `admin` etiketini korur.

O ada **canlı** bir AhdCode oturumu zaten sahipse, ilk boş sonek kullanılır:
`ahdakademi1.test`, sonra `ahdakademi2.test` — her zaman en küçük boş indis.
Sahiplik hosts dosyasıyla değil, AhdCode'un kendi rota kayıt defteriyle
belirlenir: artık çalışmayan bir projeden kalan `127.0.0.1 ahdakademi.test`
satırı zararsız, bayat bir eşlemedir ve yeni oturumu sonekli bir ada itmez.

Rota, ancak uygulama gerçekten dinlemeye başladığında alınır ve oturum
bittiğinde bırakılır — Ctrl+C, `ahdcode stop` ve `ahdcode kill` için aynı
şekilde. Çöken bir oturum geride bir kayıt bırakır; bir sonraki `ahdcode dev`,
sahibine ulaşamadığını görüp onu geri alır. Böylece hiçbir ad, ölmüş bir süreç
yüzünden sonsuza dek tutulmaz.

Yönlendirme hiç kurulamazsa oturum yine de çalışır ve başlık bunu tek satırda
söyler. Bir kolaylığın başarısız olması, düzgün başlamış bir uygulamayı asla
başarısız kılmaz.

`dev`, `APP_*` değerlerini uygulamanın kendi önceliğiyle okur — önce süreç
ortamı, sonra uygulama kökündeki `.env` — ve yalnızca ne yazacağına karar
vermek için. Önce süreç ortamı, sonra proje .env dosyası okunur.

Web geliştirmede ve açık HTTPS oturumlarında `APP_ENV=production`
reddedilir. Düz HTTP kipinde `APP_PROTOCOL=https` kullanılamaz.
Geri döngü TLS için [`dev --https`](#yerel-https) kullanın. Bu kip
`APP_PROTOCOL=https` değerini yalnızca alt süreç için ayarlar.
Yapılandırma reddedildiğinde çıkış kodu sıfırdan farklıdır.

### `stop`: nazik kapanış

```bash
ahdcode stop app.dev
ahdcode stop app.run
```

`stop`, `kill`'in nazik karşılığıdır: `kill`'in kullandığı aynı
doğrulanmış kontrol kanalı üzerinden, sahibi olan oturumdan (bir `dev`
denetleyicisi veya sade bir `run` süpervizörü) temiz bir şekilde
kapanmasını ister ve — `kill`'in aksine — başarıyı bildirmeden önce sürecin
gerçekten çıktığını doğrulamak için bekler. Nazik kapanış birkaç saniye
içinde tamamlanmazsa, `stop` bunu sessizce zorla durdurmaya yükseltmek
yerine açıkça bildirir; bunun için `ahdcode kill`'i kullanın. Çıplak bir
kaynak adı verildiğinde (`app.dev`/`app.run` yerine `app.ahd`), hangi
tanımlayıcı canlıysa ona göre çözümlenir; aynı ad için hem bir `dev` hem de
bir `run` oturumu canlıysa, tahmin etmeyi reddeder ve açık dosyayı ister.

`ahdcode kill app.dev`, hem dev denetleyicisini hem de o an sahibi olduğu
çocuk süreci hiçbir başıboş süreç bırakmadan zorla durdurur;
`ahdcode kill app.run` yukarıdaki açıklamadan değişmemiştir.

Tanılamalar (diagnostics) sabit bir kod, kaynak konumu, bir alıntı (excerpt)
ve varsa bir ipucu içerir. Derleyici çağrıları, kabuk (shell) komut
dizeleri yerine argüman dizileri kullanır.

Herhangi bir komut olmadan `ahdcode` çalıştırmak REPL'i başlatır.

`lsp`, [Dil sunucusu rehberinde](LSP_TR.md) açıklanan dil sunucusunu
başlatır: yalnızca stdio üzerinden JSON-RPC ve v0.2.2 pratik günlük özellik
seti (tanılamalar, hover, otomatik importlu completion, tanıma git, belge
sembolleri, signature help, referans bulma, rename, semantic token, inlay
hint, code action, biçimlendirme, workspace sembolleri, katlama ve seçim
aralıkları) — hepsi derleyici destekli. v0.4.0 modülleri (`HTTP` ve `HTML`
gibi) aynı derleyici modül arayüzünden görünür; v0.5.0 `cookie`/`sessions`
ve v0.6.0 `client`/`clientRequest`/`Client` dışa aktarımları da aynı
yoldandır. HTTP/çerez/oturum/istemciye özel bir LSP
kataloğu yoktur. v0.3.0'ın `SQLite`'ı aynı yolu kullanır.
İsteğe bağlı bir `--stdio` dışında argüman kabul etmez (kabul
edilir ve yok sayılır — gerçek LSP istemci kütüphaneleri, sunucuyu stdio
transport üzerinden başlatırken bunu otomatik olarak ekler; `ahdcode lsp`
zaten başka hiçbir transport'u desteklemediği için bu bayrak bir no-op'tur)
ve stdout'a protokol çerçeveleri dışında hiçbir şey yazmaz.

## Yerel geliştirme: `.test` adları ve yönlendirici

`ahdcode dev` ve `ahdcode databases`, çalıştıkları sürece küçük bir birinci
taraf ters yönlendirici (reverse router) barındırır. Go standart
kütüphanesinden kuruludur — Caddy yok, nginx yok, dış servis yok, makineye
kurulan bir şey yok — ve onu barındıran oturumla birlikte ortadan kalkar.

Yaptığı iş bilinçli olarak dardır:

- **yalnızca geri döngüyü** dinler; asla `0.0.0.0` ve asla dış bir arayüz
  değil. Bu makinenin dışından hiçbir şey ona erişemez;
- yalnızca AhdCode rota kayıt defterindeki hedeflere iletir ve kayıt defteri
  yalnızca geri döngü hedefi kabul eder — kayıt yazılırken ve okunurken iki
  kez denetlenir;
- izin listesinde olmayan bir `Host` başlığı `404` ile reddedilir. Bir isteğin
  kendi hedefini belirtmesinin hiçbir yolu yoktur; bu yüzden açık bir vekil
  (open proxy) değildir ve öyle hâle getirilemez;
- yöntem, yol, sorgu, başlıklar ve gövde değiştirilmeden iletilir; uygulama,
  geri döngü portunu değil yazılan adı görür.

Bu yönlendirici düz metin HTTP sunar. Doğrudan yerel HTTPS uygulamanın
geri döngü dinleyicisini kullanır.

### Hangi port

Temiz adresi 80 portu verir ve önce o denenir. Çoğu Unix makinesinde
ayrıcalıksız bir süreç bu porta bağlanamaz; bu bir platform politikasıdır,
etrafından dolaşılacak bir şey değil: AhdCode yetki yükseltmez, sonsuza dek
yeniden denemez ve kimseyi beklemez. Belirlenimci yedek port `7357`'yi hemen
alır ve gerçekten çalışan adresi yazar:

```text
http://ahdakademi.test:7357/
```

Makinenizde `7357` doluysa `AHDCODE_LOCAL_ROUTER_PORT` bu yedeği değiştirir.

Portu yalnızca bir süreç tutabilir; ama her AhdCode oturumu kayıtlı bütün
rotaları sunar, dolayısıyla portu hangisinin tuttuğu önemli değildir. İkinci
başlayan oturum arka planda sessizce denemeye devam eder ve ilki durduğu anda
devralır.

### `ahdcode local status`

Neyin çalıştığını ve nereye gittiğini bildirir; hiçbir şeyi değiştirmez:

```text
AhdCode Local

Router: running
Bind: 127.0.0.1:7357
  Port 80 was not available, so local URLs carry :7357.

System hosts: /etc/hosts
  managed block: absent
  ahdakademi.test: mapped to 127.0.0.1
  ahddatabasestudio.test: not mapped
  (`ahdcode local hosts` shows how to add the missing ones)

Routes:
  ahdakademi.test
    url: http://ahdakademi.test:7357/
    -> 127.0.0.1:18437
    source: /home/ada/projects/ahd/app.ahd

Route registry: ~/.config/ahdcode/routes.json
Database registry: ~/.config/ahdcode/databases.json
```

Sahibi artık çalışmayan bir rota ayrıca "stale" olarak listelenir. Yalnızca
kendi kimlik doğrulamalı denetim kanalına yanıt veren bir oturum canlı sayılır
— kayıtlı süreç kimliğine asla güvenilmez, çünkü işletim sistemleri onları
yeniden kullanır.

### `ahdcode local hosts`

Bir `.test` adının yine de çözülmesi gerekir. AhdCode, sistem hosts dosyasında
tam olarak bir sınırlanmış bloğu yönetir:

```text
# BEGIN AHDCODE LOCAL
127.0.0.1 ahdakademi.test
127.0.0.1 ahddatabasestudio.test
# END AHDCODE LOCAL
```

Yerel konak bütünleştirmesini bir kez onayladıktan sonra `ahdcode dev` geçerli
konak adını bu blokta otomatik tutar. İlk TTY oturumu sorar:

```text
Enable local .test names? [Y/n]
```

Onay, herhangi bir yetki sorusundan önce gelir. `checkmate.test` gibi sonraki
bir proje aynı yetkili mekanizma ile eklenir; her konak adı için ayrı bir
komut çalıştırmanız gerekmez. `ahdcode local hosts apply` ve `remove` elle
yönetim ve kurtarma olarak kalır.

`ahdcode local hosts` bloğu ve yerinde olup olmadığını yazar.
`ahdcode local hosts apply` onu yazar; `ahdcode local hosts remove` geri alır.
Üçünde de:

- **iki işaretin dışındaki her şey bayt bayt olduğu gibi bırakılır.** Dosya
  hiçbir zaman yeniden üretilmez, sırası değiştirilmez; sizin veya başka bir
  aracın koyduğu bir kayıt, AhdCode'un kendi bloğunun kaldırılması dâhil
  hayatta kalır;
- yalnızca `127.0.0.1` eşlemeleri ve yalnızca AhdCode'un gerçekten
  yönlendirdiği adlar yazılır;
- dosya zaten yazılabilirse değişiklik doğrudan uygulanır ve hiçbir yetki
  yükseltme söz konusu olmaz;
- aksi hâlde tam değişiklik gösterilir ve yönetici erişimi istenmeden önce
  **etkileşimli bir soruya açık bir evet** gerekir. `sudo` asla sessizce ve
  asla bu yanıt olmadan çağrılmaz;
- **etkileşimsiz** bir oturum asla sormaz ve asla yetki yükseltmez. Neyin
  gerektiğini yazıp çıkar; kimsenin yazmayacağı bir parolayı beklemez.

`.local` yerine `.test` bilinçli olarak kullanılır: `.test`, RFC 6761 ile
ayrılmıştır ve asla devredilmeyecektir; `.local` ise macOS'ta ve çoğu Linux
masaüstünde mDNS/Bonjour tarafından sahiplenilir.

Adlar her çalıştırmada budanmak yerine blokta birikir: arkasında bir şey
olmayan bir geri döngü eşlemesi zararsızdır ve onu silmek, aynı proje bir
sonraki çalıştığında yönetici erişimini yeniden istemek anlamına gelirdi.

Reddederseniz ya da platform desteklemiyorsa hiçbir şey bozulmaz — uygulama,
başlığın her zaman yazdığı kendi geri döngü adresinden erişilebilir kalır.

## PWA dosyaları ekleme

`ahdcode pwa init [--name NAME] [--short-name NAME]` komutu, uygulama
kökündeki `PWA/` klasörüne manifest, worker, simgeler ve AhdCode
entegrasyon modülü ekler. Mevcut proje dosyalarını korur ve mevcut bir
`PWA/` yolunun üzerine yazmaz. [PWA kurulumu ve önbellek politikası](PWA_TR.md).

## Yerel HTTPS

Doğrudan geri döngü TLS için `ahdcode dev --https app.ahd` komutunu
kullanın. Hazır bir PEM çifti için giriş dosyasından önce
`--cert FILE --key FILE` ekleyin. Varsayılan HTTP'dir. Sertifika üretimi,
tarayıcı güveni, yenileme, port çakışması ve Secure çerezler için
[Web yerel HTTPS](WEB_TR.md#14-yerel-https) bölümüne bakın.

## `ahdcode databases`

Bu kurulu AhdCode sürümüne ait AhdDataStudio'yu başlatır. Olağan yol AhdCode
deposuna, üst dizin taramasına veya `AHDCODE_ROOT` değerine ihtiyaç duymaz.
CLI, araç zincirine gömülü tam sürüm Studio'yu kullanıcı önbelleğine çıkarır
ve onu çalıştırır.

`AHDCODE_ROOT` bir geliştirici geçersiz kılmasıdır: ayarlıysa paketlenmiş
kopya yerine o ağaçtaki `tools/AhdDataStudio` kullanılır. Geçersiz kılma
ayarlıysa ve Studio orada yoksa komut başka yer aramaz, başarısız olur.
Olağan kurulumda `AHDCODE_ROOT` ayarlanmaz.

`ahdcode databases list`, `add` ve `remove` Studio'yu başlatmaz ve
materialize etmez. Yalnızca kullanıcıya özel SQLite kayıt defterini okur
ve yazar.

Başlatılırken `.env` yoksa `.env.example` dosyasından kopyalanır. Yeni dosya
Unix'te `0600` izinleriyle, Windows'ta ise mevcut kullanıcıya, SYSTEM ve
Administrators gruplarına erişim veren özel bir ACL ile oluşturulur. Mevcut
`.env` dosyasının içeriği ve izinleri korunur. Sunucu yalnızca
`127.0.0.1:8081` adresine bağlanır.

Kanonik adresi şudur:

```text
http://ahddatabasestudio.test/
```

ve yukarıda anlatılan yerel yönlendirici üzerinden sunulur. Doğrudan adres
tamamen desteklenmeye devam eder; temiz ad bu makinede çözülemediğinde CLI
zaten onu açar:

```text
http://127.0.0.1:8081/AhdDataStudio
```

Studio ayrıca `GET /` isteğine `/AhdDataStudio` yönlendirmesiyle yanıt verir;
böylece hem temiz ad hem de çıplak geri döngü kökü işe yarar bir yere düşer.

CLI, DNS sorgulamadan yerel hosts dosyasını okur. `ahddatabasestudio.test`
için çelişkisiz bir IPv4 eşlemesi bulamazsa doğrudan geri döngü adresini açar.
Yerel konak bütünleştirmesi zaten yetkilendirilmişse Studio adı otomatik
tutulur. İlk TTY kullanımı herhangi bir yetki sorusundan önce onay ister.
Reddetme veya etkileşimsiz oturum Studio'yu doğrudan bağ adresinde bırakır.

### `ahdcode databases list | add | remove`

Çıplak `ahdcode databases` hâlâ "AhdDataStudio'yu aç" demektir. Altında,
kullanıcıya özel **veritabanı kayıt defteri** üzerinde üç küçük komut vardır;
böylece bir veritabanı, hiçbir ortam değişkeni düzenlenmeden Studio'da
görünür olur:

```bash
ahdcode databases add ./database/app.db
ahdcode databases list
ahdcode databases remove ./database/app.db
```

- bu sürümde **yalnızca SQLite**. MySQL yapılandırması tam olarak olduğu yerde
  kalır — açık AhdDataStudio ayarları — çünkü bir MySQL kaynağı kimlik
  bilgilerinden ayrılamaz ve kimlik bilgilerinin bir kayıt defterinde yeri
  yoktur.
- `add`, dosyanın var olmasını gerektirir; kayıt defteri veritabanlarını
  tanımlar, oluşturmaz. Yol kanonikleştirilir (mutlak, temizlenmiş, sembolik
  bağlar çözülmüş); böylece iki farklı şekilde yazılan aynı dosya tek kayıt
  olarak kalır.
- `remove` **tam olarak** o kaydı unutur. **SQLite dosyasının kendisi asla
  açılmaz, taşınmaz veya silinmez.**
- Hiçbir şey taranmaz. Bir kayıt, ya bir starter o dosyayı oluşturduğu ya da
  siz burada adını verdiğiniz için vardır.
- Şu anda mevcut olmayan kayıtlı bir dosya, düşürülmek yerine `unavailable`
  olarak listelenir: bağlanmamış bir birimdeki veritabanı geri çekilmiş
  değildir ve onu yalnızca siz kaldırabilmelisiniz.

`list`, kayıt başına bir sekmeyle ayrılmış satır yazar (`driver`, yol, durum);
bir betikte `grep` veya `cut` için bu yeterlidir.

### `init web admin` ile otomatik kayıt

[`ahdcode init web admin`](WEB_TR.md) bir SQLite veritabanı oluşturduğunda o
tek dosyayı otomatik olarak kaydeder. Önce `AHD_DATA_SQLITE_PATHS` veya
`AHD_DATA_PROJECT_ROOT` içine bir şey eklemek gerekmez:

```bash
ahdcode init web admin
ahdcode databases      # yeni veritabanı zaten listelidir
```

Yalnızca komutun kendi oluşturduğu veritabanı kaydedilir — proje başkaları
için taranmaz. Kayıt, veritabanı güvenle yerine konduktan sonra yapılır ve
kayıt defteri yazılamazsa hata bildirilir, hiçbir şey geri alınmaz:
veritabanı da üretilen uygulama da gerçek ve doğrudur, mesaj da elle nasıl
kaydedileceğini söyler.

Bulunabilir bir Studio `.env` varsa listesi yine de güncellenir; böylece
`AHD_DATA_SQLITE_PATHS` üzerine kurulu bir v0.18 düzeni değişmeden çalışmaya
devam eder. `init web`, Studio `.env` dosyasını oluşturmaz.

### Kayıt defterleri nerede

Her iki kayıt defteri de işletim sisteminin yapılandırma dizini altındaki
kullanıcıya özel dosyalardır (Linux'ta `~/.config/ahdcode/`, macOS'ta
`~/Library/Application Support/ahdcode/`); dizin `0700` oluşturulur, her dosya
`0600` izinleriyle atomik yazılır. `AHDCODE_LOCAL_HOME` konumu değiştirir.
Yalnızca yerel geliştirme meta verisi tutarlar — konak adları, geri döngü
portları, dosya yolları — asla parola, belirteç veya veritabanı içeriği
tutmazlar. Bkz. [Env](ENV_TR.md).

## `ahdcode migrate`

Sürümlü SQL dosyalarını tek bir yerel SQLite veritabanına uygular. Bu bir
CLI iş akışıdır. `SQLite` ve `MySQL` dil modüllerinde migration metodu yoktur
ve `ahdcode databases` hâlâ yalnızca Studio kayıt defterini listeler, ekler
ve siler.

```bash
ahdcode migrate new create_notes
ahdcode migrate status --db notes.db
ahdcode migrate up --db notes.db
ahdcode migrate down --db notes.db --steps 1
```

Varsayılan dizin, çalışma dizininin yanındaki `migrations` klasörüdür.
`--dir` göreli veya mutlak yol alır. `--driver` varsayılanı `sqlite` olur.
`mysql` ve başka her sürücü, bağlantı açmadan bir tanı ile durur. Kullanım
hataları 2 ile çıkar. Başarısız bir migration 1 ile çıkar.

Ad, bir harfle başlar; ardından harf, rakam veya alt çizgi gelir ve en çok
64 karakterdir. `new`, sonraki kesintisiz sürümü iki dosya olarak yazar:

```text
migrations/0001_create_notes.up.sql
migrations/0001_create_notes.down.sql
```

Yalnızca bu adlar geçerlidir. Sürüm, öndeki tam sayıdır; `0002` ile `2`
aynı sürümdür. Sürümler boşluksuz ve tekrarsız `1..N` olmalıdır. Başka
`*.sql` adları, boş dosyalar ve geçerli UTF-8 olmayan dosyalar hatadır.
Sembolik bağ olan bir dizin girdisi reddedilir; bağ, migration kökünün
dışından bir betik çekemez. Migration kökü ya da bir üst dizin sembolik bağ
veya NTFS junction ise o da reddedilir. `filepath` temizliği bu bağları
izlemez. Son dosya, yeniden ayrıştırma noktası izlenmeden açılır ve çalışan
baytlar sağlanan baytlardır. Dizin kilit dosyası ve kurtarma sırasında yapılan
yeniden adlandırma da sembolik bağı, junction'ı, yeniden ayrıştırma noktasını
veya normal olmayan dosyayı reddeder. Yerine konmuş bir kilit dosyasını izlemez
ve gelen dosyayı başka bir dosyanın üzerine yeniden adlandırmaz. Denetimden
sonra bir üst dizinin bağla değiştirilmesi, kalan yarıştır; migrate bağı
görebildiğinde kapanır.
Veritabanı yolu boş olamaz, `:memory:` olamaz, dizin olamaz ve sembolik bağ
ya da junction olamaz. `up`, eksik veritabanı dosyasını oluşturur ve eksik
üst dizini reddeder. `status`, eksik bir veritabanını oluşturmaz: 0 ile çıkar
ve bekleyen dosyaları listeler. `down` da eksik bir veritabanını oluşturmaz.

`up`, bekleyen her dosyayı sürüm sırasında uygular. Sonraki bir dosya
başarısız olursa, daha önce işlenen sürümler hatadan önce yazılır ve geri
alınmaz. `status`, her sürüm için bir `applied` veya `pending` satırı basar.
Bekleyen bir şey yokken `up` yeniden çalışırsa `No pending migrations.` basar
ve hiçbir şeyi değiştirmez. Alt komuta ait olmayan bir bayrak ya da iki kez
verilen aynı bayrak 2 ile çıkar. `--help` ve `-h` bu kullanımı basar ve 0
ile çıkar.

Her dosya tek bir SQLite betiği olarak çalışır. AhdCode dosyayı noktalı
virgüllere bölmez; tetikleyiciler, tırnaklı dizeler ve SQL içindeki noktalı
virgüller olduğu gibi kalır. Yalnızca boşluk veya yorum olan bir dosyada
çalıştırılabilir deyim yoktur ve kaydedilmez. Yorumun ardından gelen SQL
çalıştırılabilir. Betiğin tamamı ile geçmiş satırı tek bir `BEGIN IMMEDIATE`
işlemi paylaşır. Betik çalışırken SQLite yetkilendiricisi `BEGIN`, `COMMIT`,
`ROLLBACK`, `END`, `ATTACH`, `DETACH`, `VACUUM` ve `ahd_schema_migrations`
yazmalarını reddeder. `SAVEPOINT`, `ROLLBACK TO` ve `RELEASE` dış işlemi
bitirmediği için serbesttir. Reddedilen ya da başarısız olan betik o işlemi
geri alır; sürüm kaydedilmez ve işlenmiş kısmi bir şema kalmaz. `CREATE TABLE`
gibi olağan DDL, SQLite'ın kendi şema tablosuna içeriden yazar; bu iç yazma
serbesttir. `sqlite_master` üzerinde doğrudan değişiklik `PRAGMA writable_schema`
ister ve o pragma reddedilir.

Bir tetikleyici oluşturmak gövdesini yetkilendirmez. Migration işlenmeden
önce migrate, bu SQLite derlemesinden, ana şemada veya geçici şemada
tetikleyicisi olan her tablo ve görünüm için bir `INSERT`, bir `UPDATE` ve
bir `DELETE` derlemesini ister. `UPDATE` her olağan sütunu kendisine atar.
`rowid` bulunan bir tabloda `rowid`, `oid` ve `_rowid_` de atanır; çünkü bu
adlar üzerindeki `UPDATE OF`, olağan bir sütun atamakla derlenmez. Üretilmiş
sütunlar atanmaz. Bu deyimler hazırlanır, çalıştırılmaz.

Kapsam, bu SQLite derlemesinin bayt koduna ait bir iç ayrıntıdır; kararlı bir
genel SQLite API'si değildir. Bir tetikleyici ancak `EXPLAIN` bilinen sekiz
sütunu verdiğinde ve adres 0'daki bir `Init` komutunun `p4` değeri tam olarak
`-- TRIGGER`, bir boşluk ve o tetikleyicinin adı olduğunda derlenmiş sayılır.
İşlem kodu, adres, `p1`, `p3`, `p5` ve boş açıklama da bu biçime uymalıdır.
Başka bir işlem kodundaki metin, bir `DEFAULT`, bir `CHECK` ya da başka bir
tetikleyicinin içindeki dizgi dahil, kapsam sayılmaz. Bayt kodu düzeni bu
komutun denendiği düzen değilse migration reddedilir. `ahd_schema_migrations`
yazan bir gövde reddedilir. Bu imle görünmeyen bir tetikleyici, atanamayan
üretilmiş bir sütun ya da olmayan bir sütun üzerindeki `UPDATE OF` dahil,
reddedilir. Bir hazırlama hatası, gövdenin güvenli olduğunun kanıtı değildir.

İm, şema adını taşımaz. SQLite'ın ASCII büyük/küçük harf katlamasında aynı
adı taşıyan iki tetikleyici, `main` ve `temp` içindeki aynı ad ya da yalnız
ASCII durumda ayrılan adlar dahil, reddedilir; çünkü derlenen bir gövde
diğerini kanıtlamaz. `BEFORE`, `AFTER` ve `INSTEAD OF` tetikleyicileri,
gerçek bir sütundaki `UPDATE OF`, görünümler, `WITHOUT ROWID` tablolar, birden
çok deyimli gövdeler, tırnaklı tanımlayıcılar, iç içe tetikleyiciler ve
sonraki bir migration'ın eklediği tetikleyici bu yolla denetlenir. Kabul
edilen tetikleyici olağan bir uygulama tetikleyicisidir. Bu denetim, gelecekteki
her SQLite bayt kodu biçimini tanıdığını söylemez.

Yetkilendirici, aynı bağlantıdaki sonraki bir migration'ın kilidini,
yetkisini veya bütünlüğünü değiştirecek bağlantı pragma'larını da reddeder:
`busy_timeout`, `locking_mode`, `trusted_schema`, `recursive_triggers`,
`foreign_keys`, `defer_foreign_keys`, `ignore_check_constraints`,
`query_only`, `read_uncommitted`, `cell_size_check`, `legacy_alter_table`,
`synchronous`, `writable_schema`, `journal_mode`, `wal_checkpoint`,
`wal_autocheckpoint`, `case_sensitive_like`, `secure_delete`,
`reverse_unordered_selects`, `temp_store`, `incremental_vacuum` ve
`schema_version`. Her migration kilidi almadan önce komut meşgul zaman
aşımını yeniden beş saniyeye kurar ve bu bağlantı ayarlarını dayandığı
varsayılanlara döndürür. `auto_vacuum` serbesttir: bu, komutun sıfırladığı
bir bağlantı bayrağı değil, veritabanı saklama ayarıdır. `table_info` gibi
şema sorguları serbesttir. Journal kipi değiştirilmez.

Geçmiş `ahd_schema_migrations` tablosundadır:

| sütun | anlamı |
| --- | --- |
| `version` | tamsayı birincil anahtar |
| `name` | sürüm önekinden sonraki ad |
| `up_checksum` | `.up.sql` baytlarının SHA-256 onaltılığı |
| `down_checksum` | `.down.sql` baytlarının SHA-256 onaltılığı; uygulama anında down dosyası yoksa boş |
| `applied_at` | Unix saniyesi |

`new`, migration dizininde işletim sistemi kilidi alır ve kilidi aldıktan
sonra yeniden tarar; iki oluşturucu aynı sürümü seçemez. Kilit, süreç
çöktüğünde de süreç bitince bırakılır. Down dosyasını yayımladıktan sonra
öldürülen bir oluşturucuyu sonraki `new` tamamlar; yayımlanmamış geçici
dosyalar atılır. Yalnızca down dosyası olmayan mevcut bir migration
silinmez.

Geçmiş adı, SQLite'ın büyük/küçük harfe duyarsız ASCII tanımlayıcı kurallarına
uyar. `AHD_SCHEMA_MIGRATIONS` ile `ahd_schema_migrations` aynı nesnedir. Bu
adda bir görünüm ya da birden çok nesne hatadır; migrate yerine yeni bir tablo
oluşturmaz. Tablo, ilk başarılı migration ile aynı işlemin içinde oluşturulur.
Sütunları,
bildirilen tipleri, `NOT NULL` kısıtları, birincil anahtarı ve dizinleri tam
olarak eşleşmelidir. Migrate sürümleri, sağlamaları ve zaman damgalarını
SQLite değer tipleriyle okur; metni tamsayıya ya da NULL'u boş dizgeye
dönüştürmez. Yanlış bir biçim ya da kötü bir değer komutu durdurur. Tablo
onarmak için değiştirilmez, silinmez ya da yeniden yazılmaz. Uygulanmış
sürümler tam olarak `1..N`
olmalıdır. Eşleşen dosyası olmayan bir veritabanı sürümü, yeniden adlandırılmış
bir dosya veya diskteki baytlarla artık eşleşmeyen bir sağlama, komutu durdurur.
Sağlamalar ham dosya baytlarıdır; CRLF'yi LF yapmak bir değişikliktir.
Meta veri sorguları bağlı parametre kullanır.

İki `ahdcode migrate up` süreci aynı dosyada `BEGIN IMMEDIATE` alır. Bekleyen
en çok beş saniye bekler. Kilidi alınca geçmişi yeniden okur ve diğer sürecin
kaydettiği sürümü atlar. Aynı sürüm iki kez uygulanmaz. İşlemin içinde
öldürülen bir süreç geçmiş satırı bırakmaz; sonraki `up` o sürümü uygular.

`down`, `up` başarısız olduğu için asla çalışmaz. Yalnızca son uygulanan
sürümü, ya da `--steps` kadarını, ve yalnızca uygulama anında kaydedilen
down betiği hâlâ bayt bayt aynı dosyaysa geri alır. Her adım kendi işlemidir:
down betiği çalışır, sonra o geçmiş satırı silinir. Sonraki bir adım
başarısız olursa önceki geri almalar kalır. İşlenen her geri alma yazılır,
ardından hata ve işlenen geri alma sayısı yazılır ve komut 1 ile çıkar.
Üretilen down dosyası, siz değiştirene kadar başarısız olan bir deyim içerir;
düzenlenmemiş bir down geçmişi silemez. Down dosyası olmadan uygulanan bir
migration, sonradan bir down dosyası görünse bile geri alınamaz.

`ahdcode migrate` journal kipini değiştirmez. Yukarıdaki bağlantı
pragma'larını, `foreign_keys` ve `synchronous` dahil, reddeder ve aynı
bağlantıdaki sonraki migration'dan önce onları geri yükler. Sizin yazdığınız
SQL dışında uygulama tablolarını silmez.
Kalıcı HTTP oturum tabloları (`ahd_session_meta`, `ahd_sessions`,
`ahd_session_revoked`) migration geçmişi değildir. Yazdığınız bir migration
onları ancak SQL'i öyle diyorsa değiştirir.

Sıradan SQLite migration'ları yönetici hakkı ve ağ istemez. Windows
`backupTo` URI yolu kullanılmaz.

Bkz. [`examples/v2.10/migrations`](../examples/v2.10/migrations).

## Derlemeden kaynak kodu denetleme

Proje yükseltmesinde kaynak doğrulaması için denetleyicinin çıkış kodunu ve
JSON raporunu kullanın. [v2.12 geçiş adımlarına](UPGRADE_2.12_TR.md) bakın.

```bash
ahdcode check app.ahd
ahdcode check "boşluk içeren yol/app.ahd" --format json
```

`check`, kaydedilmiş giriş dosyasını ve bağımlılıklarını okur, modülleri
çözümler, kaynak kodu ayrıştırır ve `build` ile
[dil sunucusunun](LSP_TR.md) kullandığı anlamsal analizi çalıştırır.
Dil sunucusu kaydedilmemiş düzenleyici içeriğini de inceleyebilir.
`format --check` standart biçimi denetler. Başarılı bir `check`, kaynak kodun
derleyicinin ön yüzünden geçtiğini gösterir. Go kodu üretimi, bağlama ve
çalışma zamanı davranışı sonraki aşamalarda denetlenir. Bu komut `PATH`
üzerinde Go gerektirmez, uygulama kodunu veya yardımcı süreçleri çalıştırmaz
ve derleme çıktısı yazmaz.

Varsayılan `--format text`, tanıları standart çıktıya
`yol:satır:sütun: önem KOD: ileti` biçiminde yazar. Varsa derleyicinin
ipucu izleyen satırda yer alır. Satır ve sütun 1’den başlar. Sütun, UTF-16 kod
birimiyle sayılır. Tanı yoksa metin yazılmaz.

`--format json`, geçerli ve geçersiz kaynaklar için standart çıktıya UTF-8
kodlamalı tek bir JSON belgesi yazar. Sürüm 1, tamsayı `schemaVersion` ve
boş olabilen `diagnostics` dizisini içerir. Her tanıda `severity`
(`error` veya `warning`), `code` (derleyici kodu), `message`, `path` ve
`range` bulunur. Derleyici ipucu sağladığında `hint` eklenir. Aralığın
`line` ve `character` değerleri sıfırdan başlar. Karakter konumu UTF-16
kod birimiyle sayılır ve bitiş konumu aralığa dahil değildir. Kaynakta
konumu bulunmayan giriş hataları için `"range": null` kullanılır.

Yollar `/` ayırıcısını kullanır ve çözümlenen giriş dizinine görelidir.
Başka bir disk birimindeki kaynak, dosya adıyla gösterilir. Gömülü
kaynaklar `<ahdcode>/...` biçimini kullanır. Rapor kaynak kod alıntısı
içermez. Tanılar yol, aralık (aralığı olmayanlar önce), önem, kod, ileti ve
ipucuna göre sıralanır.

`app.ahd` içindeki `x: Int := "bad"` için:

```json
{
  "schemaVersion": 1,
  "diagnostics": [
    {
      "severity": "error",
      "code": "SEM004",
      "message": "initializer of x expects Int; received String",
      "path": "app.ahd",
      "range": {"start": {"line": 0, "character": 10}, "end": {"line": 0, "character": 15}},
      "hint": "use a value assignable to the declared type"
    }
  ]
}
```

Çıkış kodu hata yoksa (yalnızca uyarı varsa da) `0`, kaynak veya giriş
hatası varsa `1`, geçersiz komut kullanımı varsa `2` olur. Kullanım ve
çıktı yazma hataları standart hata akışına gider. Kullanım hatalarında
JSON belgesi yazılmaz.
