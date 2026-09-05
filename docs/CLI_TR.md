# CLI

[English](CLI.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Formatter](FORMATTER_TR.md) · [REPL](REPL_TR.md) · [Dil sunucusu](LSP_TR.md)

Mevcut komut yüzeyi (command surface) şudur:

```text
ahdcode
ahdcode init web [empty|basic|admin]
ahdcode databases
ahdcode databases list
ahdcode databases add <file.db>
ahdcode databases remove <file.db>
ahdcode build <entry.ahd> [-o <output>]
ahdcode run <entry.ahd> [-- <args>...]
ahdcode dev <entry.ahd>
ahdcode stop <app.dev|app.run>
ahdcode kill [--force] <app.dev|app.run>
ahdcode local status
ahdcode local hosts [apply|remove]
ahdcode format [--check] <file.ahd>
ahdcode lsp
ahdcode --help
ahdcode --version
```

`ahdcode init web` **bulunulan dizine** bir Web starter yazar. Bir TTY'de
Empty, Basic veya Admin sorar. Etkileşimsiz kullanım `empty`, `basic` veya
`admin` geçmelidir. Şablonlar, Bootstrap 5.3.3 ve AhdCode logosu gömülüdür:
çevrimdışı, paket yöneticisi yok, üzerine yazma yok. Sonraki adım:
`ahdcode dev app.ahd`.

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

| `APP_HOST`          | yerel ad               |
| ------------------- | ---------------------- |
| `ahdakademi.com`    | `ahdakademi.test`      |
| `ahdakademi.com.tr` | `ahdakademi.com.test`  |
| `localhost`         | `localhost.test`       |
| `ahdakademi.test`   | `ahdakademi.test`      |

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
vermek için. Hiçbir değişkeni dışa aktarmaz ve alt sürece hiçbir şey geçirmez.

Bir şey başlatmadan önce iki yapılandırmayı reddeder:

- `APP_ENV=production`. Bir production sözleşmesini geliştirme komutuyla
  çalıştırmak, ya onu development saymak ya da `APP_ENV`'i yeniden yazmak
  olurdu.
- `APP_PROTOCOL=https`. `dev` düz metin HTTP sunar; alt süreci başlatmak,
  yapılandırma `https` derken `http` sunmak olurdu. v0.19 `.test` adlarını
  düz metin HTTP üzerinden yönlendirir; hâlâ yerel bir sertifika otoritesi
  veya sertifika yönetimi getirmez ve `dev` ne protokolü düşürür ne de
  güvenilmeyen bir sertifika üretir — bkz.
  [Web](WEB_TR.md#14-yerel-https--mevcut-sınır).

Her iki durumda da `dev` uyuşmazlığı bildirir, alt süreç başlatmaz, dinleyici
açmaz, geride `.dev` tanımlayıcısı bırakmaz, iki değişkeni de değiştirmez ve
sıfırdan farklı bir kodla çıkar.

Hiç `bring Web` yazmamış bir program, ortamında `APP_ENV` bulunsa bile
bunların hiçbirinden etkilenmez.

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

v0.19'da yerel geliştirme **düz metin HTTP**'dir. Yerel TLS, sertifika
otoritesi ve ACME yoktur.

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

## `ahdcode databases`

Paketle gelen AhdDataStudio veritabanı çalışma alanını başlatır. Kaynak keşfi
önce `$AHDCODE_ROOT/tools/AhdDataStudio/app.ahd` dosyasına, sonra geçerli
dizinden yukarı doğru `tools/AhdDataStudio` dizinine (veya Studio dizininin
kendisine) bakar. Başka bir projede çalışırken `AHDCODE_ROOT` değerini AhdCode
kaynak deposuna ayarlayın. Makine veya ev dizini taraması yapılmaz.

Başlatılırken `.env` yoksa `.env.example` dosyasından `0600` izinleriyle
kopyalanır; mevcut `.env` korunur. Sunucu yalnızca `127.0.0.1:8081` adresine
bağlanır.

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
Hosts dosyalarını otomatik değiştirmez — bunun için `ahdcode local hosts apply`
vardır ve o da önce sorar.

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

- v0.19'da **yalnızca SQLite**. MySQL yapılandırması tam olarak olduğu yerde
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
