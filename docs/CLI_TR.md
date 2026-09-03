# CLI

[English](CLI.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Formatter](FORMATTER_TR.md) · [REPL](REPL_TR.md) · [Dil sunucusu](LSP_TR.md)

Mevcut komut yüzeyi (command surface) şudur:

```text
ahdcode
ahdcode build <entry.ahd> [-o <output>]
ahdcode run <entry.ahd> [-- <args>...]
ahdcode kill [--force] <app.run>
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
standart kütüphanede onu okuyan ya da yazan hiçbir şey yoktur; bir kontrol
yetkisi taşıdığı için `0600` izinle yazılır; bir süreç yöneticisi, arka plan
servis kaydı veya `dev`/izleme kipi değildir.

`build`, üretilen çalıştırılabilir dosyanın yolunu yazdırır. `-o` olmadan,
derleyici geçerli çalışma dizininde giriş modülünün temel (base) adını
kullanır.

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
