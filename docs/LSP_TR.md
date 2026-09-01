# Dil sunucusu (Language server)

[English](LSP.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [CLI](CLI_TR.md) · [Dil Turu](LANGUAGE_TOUR_TR.md)

```bash
ahdcode lsp
```

AhdCode dil sunucusunu başlatır. Standart
[Language Server Protocol](https://microsoft.github.io/language-server-protocol/)'ü
**yalnızca stdin/stdout üzerinden** konuşur — TCP portu, HTTP uç noktası,
tarayıcı, arka plan servisi (daemon) veya soket modu yoktur. `stdout`
yalnızca protokol çerçeveleri taşır; `ahdcode lsp` asla bir sürüm başlığı,
log satırı veya başka bir insan-okunabilir metin yazmaz. LSP konuşan
herhangi bir editör onu bir alt süreç (child process) olarak başlatıp
konuşabilir — [`editors/vscode`](../editors/vscode) içindeki VS Code
eklentisi olası istemcilerden yalnızca biridir, özel bir durum değildir.

## Neyi derleyici karar verir, LSP değil

Dil sunucusunun kendine ait bir ayrıştırıcısı (parser), tür denetleyicisi
veya sembol kataloğu yoktur. Her tanılama (diagnostic) ve her hover,
`ahdcode build` ve `ahdcode run`'ın kullandığı aynı sözcüksel çözümleyici
(lexer), ayrıştırıcı ve anlamsal analizciden — ince bir belge-farkında
derleme katmanı (`internal/analysis`) ve bir LSP protokol çeviri katmanı
(`internal/lsp`) aracılığıyla — doğrudan gelir. Bir standart modülün dışa
aktarılan üyeleri (`Math.PI`, `Excel.read` vb.) editör için asla elle
listelenmez: bunlar derleyicinin `bring`'i çözümlemek için zaten kullandığı
aynı `StandardModuleInterfaces()`'ten gelir, bu yüzden gelecekteki bir
standart modül, ayrı bir katalog güncellemeden otomatik olarak tanılama ve
hover'a katılır.

## v0.2.0 yetenekleri

Bu bir temel (foundation) sürümüdür. Tam olarak şunları uygular:

- **Belge senkronizasyonu** — `textDocument/didOpen`, `didChange`,
  `didClose`, **tam (full)** belge senkronizasyonu kullanılarak
  (`TextDocumentSyncKind.Full`). Her `didChange`, belgenin tam yeni metnini
  taşır; sunucu tüm belgeyi bu anlık görüntüden yeniden analiz eder.
  v0.2.0'da artımlı (incremental) düzenleme uygulaması ve artımlı bir
  derleyici yoktur — doğruluk bu optimizasyondan önce gelir.
- **Tanılamalar** (`textDocument/publishDiagnostics`) — lexer, parser,
  modül/import ve anlamsal tanılamalar; her biri derleyicinin kendi sabit
  kodunu, önem derecesini (severity), mesajını ve kaynak aralığını taşır.
  Bir tanılamanın `source`'u her zaman `"ahdcode"`'dur. Bir belgeyi
  düzeltmek boş bir tanılama listesi yayınlar, böylece eski işaretler
  temizlenir. İçe aktarılan bir modüldeki hata, içe aktaran dosyaya
  katıştırılmak yerine o modülün kendi belgesi altında yayınlanır.
- **Hover** (`textDocument/hover`) — derleyicinin güvenle gerçek bir sembole
  çözümlediği bir tanımlayıcı için: bir değişken, `Constant` veya `Local`
  bildirimi veya kullanımı; bir fonksiyon bildirimi veya çağrısı; bir
  fonksiyon veya structure parametresi; bir `Class`; veya içe aktarılan bir
  standart modül üyesi. Başka herhangi bir yeri (operatör, literal, boşluk)
  hover'lamak tahmin yerine hover döndürmez.

Sunucu **kaydedilmemiş editör metnini** analiz eder. Bir açık belgenin
tamponunu, yalnızca derlemek için asla diskteki dosyasına geri yazmaz — REPL'in
kendi oturum kaynağı için zaten kullandığı bellek-içi giriş (in-memory entry)
yaklaşımının, herhangi sayıda açık belgeye genelleştirilmiş hâli. Editörde de
açık olan içe aktarılmış bir modül de kendi kaydedilmemiş tamponundan analiz
edilir; açık olmayan her şey gerçek dosya sisteminden okunur.

## v0.2.0'da olmayanlar

Completion, tanıma git (go to definition), belge sembolleri, referans bulma,
yeniden adlandırma (rename), signature help, semantic token/vurgulama, inlay
hint, code action, quick fix, otomatik import, refactoring, workspace
genelinde indeksleme, artımlı bir derleyici veya ayrıştırıcı ve kalıcı bir
derleyici önbelleği bu sürümde uygulanmamıştır. `initialize` yanıtı tam
olarak yukarıdaki yetenekleri ve başka hiçbir şeyi duyurur, bu yüzden bir
istemci bu sürümde olmayan bir özelliği isteyebileceğine asla inanmaz.
Sonraki sürümlerin bu aynı temel üzerine tek tek özellik eklemesi
beklenmektedir.

## Konum kodlaması (position encoding)

AhdCode kaynak konumları, Unicode kod noktalarıyla sayılan bir-tabanlı
(one-based) satır/sütunlardır. LSP konumları sıfır-tabanlı satır/UTF-16 kod
birimleridir. Sunucu, her istekte gerçek kaynak metnini kullanarak ikisi
arasında dönüştürme yapar — asla yalın bir "birden çıkar" değil — böylece
BMP-dışı bir karakterden (çoğu emoji gibi) sonraki bir konum, bir UTF-16
vekil çift (surrogate pair) yarısı kadar kaymış değil, doğru editör
karakterine iner.
