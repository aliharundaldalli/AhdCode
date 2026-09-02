# CLI

[English](CLI.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Formatter](FORMATTER_TR.md) · [REPL](REPL_TR.md) · [Dil sunucusu](LSP_TR.md)

Mevcut komut yüzeyi (command surface) şudur:

```text
ahdcode
ahdcode build <entry.ahd> [-o <output>]
ahdcode run <entry.ahd> [-- <args>...]
ahdcode format [--check] <file.ahd>
ahdcode lsp
ahdcode --help
ahdcode --version
```

`run`, normal önyüz (frontend) ve Go arkayüzünden (backend) derler, ardından
yerel (native) sonucu çalıştırır. Giriş dosyasından sonraki argümanlar
(isteğe bağlı olarak `--`'den sonra) oluşturulan sürece iletilir; ancak v0.1
henüz dil düzeyinde bir argüman API'si yayınlamaz.

`build`, üretilen çalıştırılabilir dosyanın yolunu yazdırır. `-o` olmadan,
derleyici geçerli çalışma dizininde giriş modülünün temel (base) adını
kullanır.

Tanılamalar (diagnostics) sabit bir kod, kaynak konumu, bir alıntı (excerpt)
ve varsa bir ipucu içerir. Derleyici çağrıları, kabuk (shell) komut
dizeleri yerine argüman dizileri kullanır.

Herhangi bir komut olmadan `ahdcode` çalıştırmak REPL'i başlatır.

`lsp`, [Dil sunucusu rehberinde](LSP_TR.md) açıklanan dil sunucusunu
başlatır: yalnızca stdio üzerinden JSON-RPC, derleyici destekli tanılamalar,
hover, tanıma git, belge sembolleri, signature help, referans bulma ve
completion. İsteğe bağlı bir `--stdio` dışında argüman kabul etmez (kabul
edilir ve yok sayılır — gerçek LSP istemci kütüphaneleri, sunucuyu stdio
transport üzerinden başlatırken bunu otomatik olarak ekler; `ahdcode lsp`
zaten başka hiçbir transport'u desteklemediği için bu bayrak bir no-op'tur)
ve stdout'a protokol çerçeveleri dışında hiçbir şey yazmaz.
