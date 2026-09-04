# AhdCode editör eklentisi

[English](README.md) · [Türkçe]

Bu minimal eklenti, VS Code uyumlu editörlere AhdCode dosya tanıma, hafif
sözdizimi vurgulama, bir **AhdCode Dosyasını Çalıştır** oynat düğmesi ve
AhdCode dil sunucusuna (v0.2.2 özellik seti: tanılamalar, hover, otomatik
import ve erişim-farkında Class üyeleriyle completion, tanıma git, belge
sembolleri, signature help, referans bulma, rename, semantic vurgulama, inlay
hint, quick fix, biçimlendirme, workspace sembolleri, katlama ve seçim
aralıkları) bir bağlantı ekler.

Eklenti, AhdCode paket markalaşması ve açık/koyu dil simgeleri sağlar. Aktif
bir üçüncü taraf File Icon Theme, dil simgesini geçersiz kılabilir; dosya
tanıma ve tüm çalıştırma eylemleri normal şekilde çalışmaya devam eder.

## Aktif dosyayı çalıştırma

Kaydedilmiş bir `.ahd` dosyası açın ve şunlardan birini kullanın:

- editör başlık çubuğundaki oynat düğmesi;
- Komut Paleti'nde `Run AhdCode File`;
- `.ahd` editörü odaktayken `F6`.

Eklenti belgeyi kaydeder, ardından `AhdCode: Run filename.ahd` adında görünür
bir görev (task) başlatır. Çalıştırılabilir dosyayı şuna eşdeğer bir argüman
dizisiyle çağırır:

```text
ahdcode run /absolute/path/to/file.ahd
```

Hiçbir kabuk (shell) komutu oluşturulmaz. Görev, dosyanın bulunduğu dizini
çalışma dizini olarak kullanarak çalışır, bu yüzden yollardaki boşluklar ve
Unicode karakterleri kabuk tırnaklama sorunları olmadan geçirilir. Tekrarlanan
çalıştırmalar için özel, temizlenmiş bir görev terminali yeniden kullanılır.

## Çalıştırılabilir dosya keşfi

Varsayılan olarak, `ahdcode`, editörün miras aldığı ortam `PATH`'inde
bulunmalıdır. Editör, PATH değişmeden önce başlatıldıysa yeniden başlatın.

Alternatif olarak `ahdcode.executablePath`'i çalıştırılabilir dosyanın
mutlak yoluna ayarlayın. Ayar, kasıtlı olarak varsayılan olarak boştur ve
makineye özgü bir yol içermez.

## Dil sunucusu

Run File'a ek olarak, eklenti aktive olduğunda aynı `ahdcode` çalıştırılabilir
dosyasını arka planda bir [dil sunucusu](../../docs/LSP_TR.md) olarak
(`ahdcode lsp`) bir kez başlatır — tam olarak Run File'ın kullandığı aynı
`ahdcode.executablePath` ayarı / `PATH` araması üzerinden çözümlenir. Yalnızca
stdio üzerinden iletişim kurar; hiçbir şekilde bir ağ portu açmaz. Size
şunları sağlar:

- **Tanılamalar**, **Hover**, **Tanıma git**, **belge sembolleri**,
  **signature help**, **referans bulma** (derleme grafiği kapsamında),
  **completion** (otomatik import dahil), **rename**, **semantic vurgulama**,
  **inlay hint**, **quick fix**, **biçimlendirme**, **workspace sembolleri**,
  **katlama** ve **seçim aralığı** — hepsi standart LSP yetenek müzakeresiyle.

Sunucu, yalnızca analiz etmek için açık bir belgeyi asla diskteki dosyasına
geri yazmaz.

Çalıştırılabilir dosya bulunamazsa veya sunucu başlatılamazsa, tek bir kısa
hata mesajı gösterilir ve Run File normal şekilde çalışmaya devam eder.
Kapsam ve sınırlamalar için [dil sunucusu rehberine](../../docs/LSP_TR.md)
bakın.

## Geliştirme

1. VS Code'da `editors/vscode`'u açın.
2. `F5`'e basın ve istenirse **Run AhdCode Extension**'ı seçin.
3. Extension Development Host'ta bir `.ahd` dosyası açın ve oynat düğmesini
   kullanın.

Bağımlılıksız (dependency-free) testleri şununla çalıştırın:

```bash
npm test
```

## Paketleme ve yerel kurulum

`editors/vscode`'dan:

```bash
npm run package
```

Bu, `@vscode/vsce package`'ı çalıştırır ve yerel bir `.vsix` oluşturur; hiçbir
şey yayınlamaz (publish).

VS Code'da Komut Paleti'nden **Extensions: Install from VSIX...** ile veya
şununla kurun:

```bash
code --install-extension ahdcode-0.2.3.vsix
```

Google Antigravity IDE 1.107, aynı yerel VSIX CLI işlemini sunar:

```bash
antigravity-ide --install-extension ahdcode-0.2.3.vsix
```

macOS'ta, bu başlatıcılar PATH'te değilse, uygulama paketine gömülü
başlatıcıları kullanın:

```bash
/Applications/Visual\ Studio\ Code.app/Contents/Resources/app/bin/code --install-extension ahdcode-0.2.3.vsix
/Applications/Antigravity\ IDE.app/Contents/Resources/app/bin/antigravity-ide --install-extension ahdcode-0.2.3.vsix
```

Aynı paket her iki editör tarafından da kullanılır. Eklenti API temel
çizgisi (baseline) VS Code 1.107'dir ve test edilmiş Antigravity bağımsız
(standalone) eklenti sunucusuyla eşleşir.

## Kapsam ve sınırlamalar

Bu, kasıtlı olarak küçük bir eklentidir: sözdizimi vurgulama, Run File ve
tanılamalar, hover, tanıma git, belge sembolleri, signature help, referans
bulma ve completion içeren bir [dil sunucusu](../../docs/LSP_TR.md).
Yeniden adlandırma (rename), semantik vurgulama, bir hata ayıklayıcı
(debugger), kesme noktaları (breakpoints) veya Marketplace yayınlaması
sağlamaz; referans bulma da tüm workspace'i değil, yalnızca açık belgenin
kendi derleme grafiğini indeksler. Run File'ın kendi çıktısı (çalışan programın
`write`/hataları) hâlâ, öncekiyle tamamen aynı şekilde, yalnızca görev
terminali çıktısı olarak gösterilir — bu, çalışan bir programın
stdout/stderr'idir, bir derleyici tanılaması değildir. Derleyici
tanılamalarının kendisi (lexer/parser/modül/anlamsal hatalar) dil
sunucusundan gelir ve normal editör sorun (problem) girdileri olarak görünür.
