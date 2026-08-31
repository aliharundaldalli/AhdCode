# AhdCode editör eklentisi

[English](README.md) · [Türkçe]

Bu minimal eklenti, VS Code uyumlu editörlere AhdCode dosya tanıma, hafif
sözdizimi vurgulama (syntax highlighting) ve bir **AhdCode Dosyasını
Çalıştır** oynat düğmesi ekler.

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
code --install-extension ahdcode-0.1.4.vsix
```

Google Antigravity IDE 1.107, aynı yerel VSIX CLI işlemini sunar:

```bash
antigravity-ide --install-extension ahdcode-0.1.4.vsix
```

macOS'ta, bu başlatıcılar PATH'te değilse, uygulama paketine gömülü
başlatıcıları kullanın:

```bash
/Applications/Visual\ Studio\ Code.app/Contents/Resources/app/bin/code --install-extension ahdcode-0.1.4.vsix
/Applications/Antigravity\ IDE.app/Contents/Resources/app/bin/antigravity-ide --install-extension ahdcode-0.1.4.vsix
```

Aynı paket her iki editör tarafından da kullanılır. Eklenti API temel
çizgisi (baseline) VS Code 1.107'dir ve test edilmiş Antigravity bağımsız
(standalone) eklenti sunucusuyla eşleşir.

## Kapsam ve sınırlamalar

Bu, kasıtlı olarak küçük bir çalıştır-ve-vurgula (run-and-highlight)
entegrasyonudur. LSP, tamamlama (completion), semantik vurgulama, bir hata
ayıklayıcı (debugger), kesme noktaları (breakpoints) veya Marketplace
yayınlaması sağlamaz. Derleyici ve çalışma zamanı tanılamaları, normal görev
terminali çıktısı olarak gösterilir; editör sorun (problem) girdilerine
dönüştürülmezler.
