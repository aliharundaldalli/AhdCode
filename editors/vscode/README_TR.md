# AhdCode editör eklentisi

[English](README.md) · [Türkçe]

Bu minimal eklenti, VS Code uyumlu editörlere AhdCode dosya tanıma, hafif
sözdizimi vurgulama (syntax highlighting), bir **AhdCode Dosyasını
Çalıştır** oynat düğmesi ve AhdCode dil sunucusuna (derleyici destekli
tanılamalar, hover, tanıma git, belge sembolleri, signature help, referans
bulma ve completion) bir bağlantı ekler.

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

- **Tanılamalar**: gerçek derleyici önyüzünden lexer, parser, modül/import ve
  anlamsal hatalar; normal editör sorun işaretleri olarak gösterilir —
  kaydedilmemiş tamponlar dahil yazarken canlı tutulur ve düzelttiğinizde
  otomatik olarak temizlenir.
- **Hover**: bir değişkeni, `Constant`/`Local` bildirimini veya kullanımını,
  bir fonksiyon bildirimini veya çağrısını, bir fonksiyon/structure
  parametresini, bir `Class`'ı veya içe aktarılan bir standart modül üyesini
  hover'lamak, derleyicinin çözümlediği türü veya imzasını gösterir.
- **Tanıma git**, **belge sembolleri** (outline görünümü), bir çağrının
  argümanlarını yazarken **signature help**, **referans bulma** (açık
  belgenin kendi derleme grafiğiyle sınırlı) ve modül adları,
  `from ... bring` dışa aktarımları, namespace/Class üyeleri, kapsamdaki
  yerel değişkenler/parametreler ve küçük bir anahtar kelime kümesi için
  **completion**.

Sunucu, yalnızca analiz etmek için açık bir belgeyi asla diskteki dosyasına
geri yazmaz.

Çalıştırılabilir dosya bulunamazsa veya sunucu başlatılamazsa, her tuş
vuruşunda değil, tek bir kısa hata mesajı gösterilir ve Run File normal
şekilde çalışmaya devam eder. Kasıtlı olarak hâlâ eksik olanlar (yeniden
adlandırma, semantic highlighting vb.) için
[dil sunucusunun sınırlamalarına](../../docs/LSP_TR.md#uygulanmayanlar)
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
code --install-extension ahdcode-0.2.1.vsix
```

Google Antigravity IDE 1.107, aynı yerel VSIX CLI işlemini sunar:

```bash
antigravity-ide --install-extension ahdcode-0.2.1.vsix
```

macOS'ta, bu başlatıcılar PATH'te değilse, uygulama paketine gömülü
başlatıcıları kullanın:

```bash
/Applications/Visual\ Studio\ Code.app/Contents/Resources/app/bin/code --install-extension ahdcode-0.2.1.vsix
/Applications/Antigravity\ IDE.app/Contents/Resources/app/bin/antigravity-ide --install-extension ahdcode-0.2.1.vsix
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
