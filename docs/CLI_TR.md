# CLI

[English](CLI.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Formatter](FORMATTER_TR.md) · [REPL](REPL_TR.md) · [Dil sunucusu](LSP_TR.md)

Mevcut komut yüzeyi (command surface) şudur:

```text
ahdcode
ahdcode build <entry.ahd> [-o <output>]
ahdcode run <entry.ahd> [-- <args>...]
ahdcode dev <entry.ahd>
ahdcode stop <app.dev|app.run>
ahdcode kill [--force] <app.dev|app.run>
ahdcode format [--check] <file.ahd>
ahdcode lsp
ahdcode --help
ahdcode --version
```

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
`dev`, uygulamayı ve kanonik geliştirme adresini adlandıran bir başlık ekler:

```
AhdCode Web
  Ahd Akademi (development)

  http://ahdakademi.com.test
```

Adres, `APP_PROTOCOL` ile `APP_HOST`'a `.test` eklenmiş hâlidir; bu,
uygulamanın genel konağının yerel kimliğidir. `dev`, `APP_*` değerlerini
uygulamanın kendi önceliğiyle okur — önce süreç ortamı, sonra uygulama
kökündeki `.env` — ve yalnızca ne yazacağına karar vermek için. Hiçbir
değişkeni dışa aktarmaz ve alt sürece hiçbir şey geçirmez.

Tek bir yapılandırmayı reddeder: `APP_ENV=production`. Bir production
sözleşmesini geliştirme komutuyla çalıştırmak, ya onu development saymak ya da
`APP_ENV`'i yeniden yazmak olurdu; bu yüzden `dev` uyuşmazlığı bildirir,
hiçbir şey başlatmaz ve sıfırdan farklı bir kodla çıkar.

`https` bir geliştirme adresi düşürülmez, açıklanır. v0.15 yerel bir sertifika
otoritesi, `.test` çözücüsü veya geliştirme geçidi getirmez; `dev` eksik olanı
söyler ve `APP_PROTOCOL`'ü olduğu gibi bırakır — bkz.
[Web](WEB_TR.md#14-yerel-https--mevcut-sınır).

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
