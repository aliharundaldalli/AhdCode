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
`ahdcode kill --force app.run` uygulamayı hemen durdurur. Run dosyası
uygulamanın kimliğidir: çıplak bir pid bilinçli olarak kabul edilmez ve
düzgün biçimli bir AhdCode run tanımlayıcısı olmayan bir dosya, hiçbir şeye
sinyal gönderilmeden reddedilir. Süreci çoktan gitmiş bir tanımlayıcı, işlem
yapılmak yerine bayat (stale) olarak bildirilip silinir; canlı bir
tanımlayıcı varken ikinci bir `run` başlatmak ise portta sessizce çakışmak
yerine pid'i ve kullanılacak `kill` komutunu bildirerek başarısız olur.

Tanımlayıcı dahili CLI meta verisidir, dil düzeyinde bir biçim değildir:
standart kütüphanede onu okuyan ya da yazan hiçbir şey yoktur; bir süreç
yöneticisi, arka plan servis kaydı veya `dev`/izleme kipi değildir.

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
