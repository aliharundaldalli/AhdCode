<p align="center">
  <img src="editors/vscode/images/ahdcode-logo.png" alt="AhdCode logosu" width="360">
</p>

# AhdCode

[![CI](https://github.com/aliharundaldalli/AhdCode/actions/workflows/ci.yml/badge.svg)](https://github.com/aliharundaldalli/AhdCode/actions/workflows/ci.yml)

[English](README.md) · [Türkçe]

AhdCode; okunabilir sözdizimi, açık niyet (explicit intent), öngörülebilir
anlambilim (semantics) ve yerel (native) derlemeye odaklanan, deneysel,
statik olarak denetlenen genel amaçlı bir programlama dilidir.

Mevcut sürüm **v0.3.0**'dır. Çekirdek dil uçtan uca çalışır, ancak proje
üretime hazır değildir ve 1.0'dan önce kırıcı (breaking) değişiklikler
olabilir.

v0.3.0, AhdCode'un pratik uygulama geliştirme evresini tipli bir
[SQLite](docs/SQLITE_TR.md) köprüsüyle başlatır: gerçek yerel veritabanı,
parametreli SQL, CRUD, işlemler (transactions) ve kalıcı bir not defteri
örneği. HTTP bu sürümde yoktur; SQLite'tan sonra gelir.

v0.2.2, v0.2.1'in tanılama, hover, completion, tanıma git, belge sembolleri,
signature help ve referans bulma özelliklerinin üzerine pratik günlük AhdCode
dil sunucusunu tamamlar. v0.2.2; yeniden adlandırma, semantic vurgulama,
inlay hint, code action/quick fix, otomatik import, belge biçimlendirme,
workspace sembol araması, katlama aralıkları ve seçim aralıkları ekler — hepsi
gerçek derleyici önyüzü tarafından desteklenir, kaydedilmemiş editör
tamponları üzerinde, tam belge senkronizasyonuyla. Kapsam ve dürüst
sınırlamalar için [`docs/LSP_TR.md`](docs/LSP_TR.md)'ye bakın (derleme
grafiğiyle sınırlı referans/rename, isteğe bağlı modül keşfi, kalıcı
workspace indeksi yok). Birlikte gelen [VS Code eklentisi](editors/vscode)
aynı sunucuyu başlatır. Dil anlambilimi v0.1.20'den değişmemiştir; v0.1.20,
[PDF](docs/PDF_TR.md) ve [Archive](docs/ARCHIVE_TR.md) modüllerini ve bir
`Latex.pdf` kaynak yan dosyasını eklemişti.

```ahd
greet: Function := (
    name: String
) -> String {
    return "Hello {name}"
}

names: List<String> := ["Ali", "Ayşe"]

for name in names {
    write(greet(name))
}
```

## Neden AhdCode?

- Bildirim ve değişiklik (mutation) farklı görünür: `:=` bildirir, `=`
  değiştirir.
- Statik denetim, ilgisiz örtük dönüşümleri (implicit conversions) ve
  truthiness'i reddeder.
- Açık null olabilen türler (`T?`), koleksiyonlarla birleşirken,
  akış-duyarlı (flow-sensitive) kontroller kanıtlanmış null olmayan
  değerleri daraltır (narrow eder).
- List'ler, Pair'ler, Class'lar, Function'lar, modüller, hatalar ve yerel
  çalıştırılabilir dosyalar v0.1 çekirdeğinin bir parçasıdır.
- Yalnızca ifade içeren `lambda (<tipli parametreler>) -> <ifade>`, mevcut
  `Function` türünde bir değer oluşturur; ayrı bir çağrılabilir tür değildir.
- Küçük, kapalı bir [Class Protocol Methods](docs/PROTOCOLS_TR.md) kümesi,
  bir Class'ın `==`, sıralama, aritmetik, tekli `-` ve `str()` davranışını
  tanımlamasına izin verir.
- Bir [Regex modülü](docs/REGEX_TR.md), desenleri (patterns) `matches`,
  `find`, `findAll`, `groups`, `replace` ve `split` içeren bir `Pattern`
  değerine derler.
- [Time](docs/TIME_TR.md), saat dilimi veritabanı eklemeden yerel, UTC, sabit
  dakika ofseti ve Unix milisaniye gösterimlerini destekler.
- Sıkı [CSV modülü](docs/CSV_TR.md), ham String satırlarını veya başlık
  anahtarlı String kayıtlarını native ve kalıcı REPL uyumuyla taşır.
- Bir ifade lambda'sı, dışarıdaki değerleri açık bir bağımlılık listesiyle
  okuyabilir: lexical yakalama için `#name`/`Local name`, modül bağlaması için
  `@name`/`Global name`, örneğin
  `lambda [#minimum, @Maximum] (score: Int) -> score >= minimum and score <= Maximum`;
  her iki tür de asla çıkarılmaz ve asla örtük değildir.
- [Statistics modülü](docs/STATISTICS_TR.md), `List<Int>` ve `List<Real>`
  üzerinde tipli betimleyici istatistik sağlar; String zorlaması yoktur.
- [Numeric modülü](docs/NUMERIC_TR.md), immutable ve Real yönelimli Vector/
  Matrix değerleri, doğrusal cebir ve Plot için ek `Vector` overload'ları sağlar.
- [Word modülü](docs/WORD_TR.md), Office veya harici çalışma zamanı
  gerektirmeden immutable biçimlendirilmiş belgeler, merge edilmiş tablolar,
  gömülü Plot görselleri ve sınırlandırılmış anlamsal DOCX okuma sağlar.
- [Excel modülü](docs/EXCEL_TR.md), tipli ve değiştirilemez Workbook/Sheet/
  Cell/Range değerleriyle gerçek `.xlsx` paketlerini okur ve yazar. Formula
  niyeti açıktır, merge değer kaybını reddeder ve native çalıştırılabilirler
  çevrimdışı ve taşınabilir kalır.
- [PDF modülü](docs/PDF_TR.md), değiştirilemez `PDFDocument` değerleri
  oluşturur ve bunları `Latex`'in kullandığı aynı konuşlandırılmış Tectonic
  render motoru üzerinden çevrimdışı gerçek `.pdf` dosyalarına render eder;
  ayrıca başka bir modülün kendi tipli belgesinin anlamsal dönüşümü olan
  `PDF.fromWord`/`PDF.fromExcel` sağlar.
- [Archive modülü](docs/ARCHIVE_TR.md), dosyaları yalnızca Go standart
  kütüphanesini kullanarak çevrimdışı gerçek ZIP, TAR ve TAR.GZ arşivlerine
  paketler, yalnızca oluşturma amaçlıdır.
- [Lists](docs/LISTS_TR.md) ve [KeyValue](docs/KEYVALUE_TR.md), `List` ve
  `Pair` üzerinde saf yapısal dönüşümler ekler — `chunk`, `flatten`,
  `transpose`, `unique`, `valueCounts`, `groupBy` ve `keys`, `values`,
  `combine`, `with`, `select`, `drop`, `rename`, `mapValues`, `merge`,
  `overlay`. Tür-yönelimlidirler: her çağrının kesin sonuç türü argüman
  türlerinden hesaplanır, genel (generic) sözdizimi olmadan ve hiçbir şey
  silinmeden.
- Formatter, yorumları korurken tek bir kanonik (standart) sunum tanımlar.
- [Dil sunucusu](docs/LSP_TR.md) (`ahdcode lsp`), derleyicinin kendi
  tanılamalarını, hover'ını, tanıma git özelliğini, belge sembollerini,
  signature help'ini, referans bulmasını ve completion'ını standart stdio
  LSP üzerinden sunar — ikinci bir ayrıştırıcı yok, elle bakımı yapılan bir
  sembol kataloğu yok ve bir belge editörde açık ve kaydedilmemişken
  dosyasına asla yazılmaz.

## Kaynak koddan derleme

AhdCode şu anda Go 1.25 veya daha yeni bir sürüm gerektirir.

```bash
cd AhdCode
go test ./...
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot ./cmd/ahdsqlite
```

Yukarıdaki komut, derleyiciyi ve yerel numeric, plot ve SQLite yardımcılarını (helpers) kurar.
Eğer `Latex` modülünü **veya** `PDF` modülünün `.save()` metodunu kullanmayı
planlıyorsanız (ikisi de aynı çevrimdışı render motorunu paylaşır),
çevrimdışı (offline) Latex/Tectonic çalışma zamanını da hazırlamanız (stage)
gerekir. `Archive` böyle bir hazırlığa ihtiyaç duymaz — yalnızca Go standart
kütüphanesini kullanır. Hazırlık işlemi, sabitlenmiş ve doğrulanmış
kaynakları indirmek için bir defaya mahsus ağ bağlantısı kullanır:

```bash
go run ./tooling/latex/cmd/package-latex --output "$(go env GOPATH)"
```

Hazırlık (staging) aşamasından sonra, AhdCode'un normal Latex işlemleri tamamen çevrimdışı çalışmaya devam eder.

Go'nun ikili dosya (binary) dizininin `PATH`'te olduğundan emin olun:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
ahdcode --version
```

## CLI hızlı başlangıç

```bash
ahdcode run examples/v0.1/01_hello.ahd
ahdcode build examples/v0.1/01_hello.ahd -o hello
ahdcode format examples/v0.1/01_hello.ahd
ahdcode format --check examples/v0.1/01_hello.ahd
ahdcode
```

[CLI rehberine](docs/CLI_TR.md), [formatter rehberine](docs/FORMATTER_TR.md),
[REPL rehberine](docs/REPL_TR.md) ve [dil sunucusu rehberine](docs/LSP_TR.md)
bakın.

## Dokümantasyon

- [Türkçe Öğrenci Rehberi](docs/STUDENT_GUIDE_TR.md)
- [English Student Guide](docs/STUDENT_GUIDE_EN.md)
- [Başlangıç](docs/GETTING_STARTED_TR.md)
- [Dil Turu](docs/LANGUAGE_TOUR_TR.md)
- [Türler ve Null Güvenliği](docs/TYPES_AND_NULL_TR.md)
- [Kontrol Akışı](docs/CONTROL_FLOW_TR.md)
- [Fonksiyonlar](docs/FUNCTIONS_TR.md)
- [Sınıflar](docs/CLASSES_TR.md)
- [Class Protocol Methods](docs/PROTOCOLS_TR.md)
- [Koleksiyonlar](docs/COLLECTIONS_TR.md)
- [Modüller](docs/MODULES_TR.md)
- [Hatalar](docs/ERRORS_TR.md)
- [Temel İşlevler](docs/FUNDAMENTALS_TR.md)
- [String API](docs/STRING_API_TR.md)
- [List API](docs/LIST_API_TR.md)
- [Math modülü](docs/MATH_TR.md)
- [Time modülü](docs/TIME_TR.md)
- [Latex modülü](docs/LATEX_TR.md)
- [Word modülü](docs/WORD_TR.md)
- [Excel modülü](docs/EXCEL_TR.md)
- [PDF modülü](docs/PDF_TR.md)
- [Archive modülü](docs/ARCHIVE_TR.md)
- [File ve Path modülleri](docs/FILESYSTEM_TR.md)
- [Regex modülü](docs/REGEX_TR.md)
- [CSV modülü](docs/CSV_TR.md)
- [Data modülü](docs/DATA_TR.md)
- [Statistics modülü](docs/STATISTICS_TR.md)
- [Plot modülü](docs/PLOT_TR.md)
- [Numeric modülü ve Complex skalerleri](docs/NUMERIC_TR.md)
- [JSON modülü](docs/JSON_TR.md)
- [SQLite modülü](docs/SQLITE_TR.md)
- [XML modülü](docs/XML_TR.md)
- [Env modülü](docs/ENV_TR.md)
- [Lists modülü](docs/LISTS_TR.md)
- [KeyValue modülü](docs/KEYVALUE_TR.md)
- [Tanılamaları anlama](docs/DIAGNOSTICS_TR.md)
- [Dil sunucusu](docs/LSP_TR.md)
- [Yapay zekâ destekli yerel kurulum](FOR_AI.md)
- [Derlenmiş v0.1 örnekleri](examples/v0.1/README_TR.md)
- [v0.3 SQLite Not Defteri](examples/v0.3/README_TR.md)
- [Tam v0.1 dil spesifikasyonu](AHDCODE_LANGUAGE_SPEC_v0.1_TR.md)

## Editör eklentisi

[`editors/vscode`](editors/vscode) içindeki yerel, VS Code uyumlu eklenti,
`.ahd` dosyalarını tanır, sözdizimi vurgulama sağlar, editör başlık
çubuğundaki oynat düğmesi, Komut Paleti veya `F6` ile aktif dosyayı
çalıştırır ve derleyici destekli tanılamalar ve hover için
[dil sunucusuna](docs/LSP_TR.md) (`ahdcode lsp`) bağlanır. Aynı VSIX, hem
VS Code hem de Antigravity'i hedefler.
[Kurulum rehberine](editors/vscode/README_TR.md) bakın.

## Mevcut sınırlamalar

v0.1, kasıtlı olarak blok/deyim (statement) lambda'ları, örtük/genel değişken (mutable) closure hücreleri,
tuple dönüş değerleri, reflection, interface, çoklu kalıtım (multiple
inheritance), hata ayıklayıcı (debugger), paket arama yolları (package
search paths) veya web çalışma zamanına sahip değildir.
[Dil sunucusunda](docs/LSP_TR.md) henüz yeniden adlandırma (rename) veya
semantic token yoktur, ve referans bulma workspace genelinde bir indeks
yerine bir belgenin kendi derleme grafiğiyle sınırlıdır. Operatör
davranışı yalnızca on sabit
[Class Protocol Methods](docs/PROTOCOLS_TR.md) aracılığıyla
kullanıcı-tanımlıdır (user-definable), genel bir aşırı yükleme (overloading)
mekanizması değildir. Modüller kardeş (sibling) `.ahd` dosyalarıdır ve
editör eklentisi hafif bir çalıştır-ve-vurgula entegrasyonudur. Bkz.
[spesifikasyonun desteklenmeyen özellik listesi](AHDCODE_LANGUAGE_SPEC_v0.1_TR.md#40-desteklenmeyen-v01-özellikleri).

## Depo haritası

```text
cmd/ahdcode/       CLI giriş noktası
cmd/ahdnumeric/    paketli ileri doğrusal-cebir yardımcısı
cmd/ahdplot/       paketli grafik render yardımcısı
cmd/ahdsqlite/     paketli CGO'suz SQLite yardımcısı
internal/          derleyici, çalışma zamanı, formatter ve REPL
editors/vscode/    VS Code / Antigravity eklentisi
docs/              son kullanıcı rehberleri
examples/v0.1/     derlenmiş çalışan programlar
examples/v0.3/     SQLite Not Defteri
AHDCODE_LANGUAGE_SPEC_v0.1.md
                   yetkili (authoritative) dil sözleşmesi
```

## Geliştirme ve katkılar

AhdCode, Ali Harun Daldallı tarafından tasarlanmış ve spesifikasyonu
oluşturulmuştur. Uygulama, dokümantasyon ve test süreçlerinde OpenAI Codex,
Anthropic Claude ve Google Gemini dahil olmak üzere yoğun yapay zekâ
desteğinden yararlanılmıştır. Araçların katkı biçimi göreve göre değişmekte;
dil tasarımı ve nihai teknik kararlar proje yazarına aittir.

## Lisans

AhdCode, [MIT Lisansı](LICENSE) altında kullanılabilir.
