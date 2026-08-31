<p align="center">
  <img src="editors/vscode/images/ahdcode-logo.png" alt="AhdCode logosu" width="360">
</p>

# AhdCode

[![CI](https://github.com/aliharundaldalli/AhdCode/actions/workflows/ci.yml/badge.svg)](https://github.com/aliharundaldalli/AhdCode/actions/workflows/ci.yml)

[English](README.md) · [Türkçe]

AhdCode; okunabilir sözdizimi, açık niyet (explicit intent), öngörülebilir
anlambilim (semantics) ve yerel (native) derlemeye odaklanan, deneysel,
statik olarak denetlenen genel amaçlı bir programlama dilidir.

Mevcut sürüm **v0.1.15**'tir. Çekirdek dil uçtan uca çalışır, ancak proje
üretime hazır değildir ve 1.0'dan önce kırıcı (breaking) değişiklikler
olabilir.

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
- Formatter, yorumları korurken tek bir kanonik (standart) sunum tanımlar.

## Kaynak koddan derleme

AhdCode şu anda Go 1.25 veya daha yeni bir sürüm gerektirir.

```bash
cd AhdCode
go test ./...
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot
```

Yukarıdaki komut, derleyiciyi ve yerel numeric/plot yardımcılarını (helpers) kurar.
Eğer `Latex` modülünü kullanmayı planlıyorsanız, çevrimdışı (offline) Latex çalışma zamanını da hazırlamanız (stage) gerekir. Bu işlem, sabitlenmiş ve doğrulanmış kaynakları indirmek için bir defaya mahsus ağ bağlantısı kullanır:

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

[CLI rehberine](docs/CLI_TR.md), [formatter rehberine](docs/FORMATTER_TR.md)
ve [REPL rehberine](docs/REPL_TR.md) bakın.

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
- [File ve Path modülleri](docs/FILESYSTEM_TR.md)
- [Regex modülü](docs/REGEX_TR.md)
- [CSV modülü](docs/CSV_TR.md)
- [Data modülü](docs/DATA_TR.md)
- [Statistics modülü](docs/STATISTICS_TR.md)
- [Plot modülü](docs/PLOT_TR.md)
- [Numeric modülü ve Complex skalerleri](docs/NUMERIC_TR.md)
- [Tanılamaları anlama](docs/DIAGNOSTICS_TR.md)
- [Yapay zekâ destekli yerel kurulum](FOR_AI.md)
- [Derlenmiş v0.1 örnekleri](examples/v0.1/README_TR.md)
- [Tam v0.1 dil spesifikasyonu](AHDCODE_LANGUAGE_SPEC_v0.1_TR.md)

## Editör eklentisi

[`editors/vscode`](editors/vscode) içindeki yerel, VS Code uyumlu eklenti,
`.ahd` dosyalarını tanır, sözdizimi vurgulama sağlar ve editör başlık
çubuğundaki oynat düğmesi, Komut Paleti veya `F6` ile aktif dosyayı
çalıştırır. Aynı VSIX, hem VS Code hem de Antigravity'i hedefler.
[Kurulum rehberine](editors/vscode/README_TR.md) bakın.

## Mevcut sınırlamalar

v0.1, kasıtlı olarak blok/deyim (statement) lambda'ları, örtük/genel değişken (mutable) closure hücreleri,
tuple dönüş değerleri, reflection, interface, çoklu kalıtım (multiple
inheritance), hata ayıklayıcı (debugger), LSP, paket arama yolları (package
search paths) veya web çalışma zamanına sahip değildir. Operatör davranışı
yalnızca on sabit
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
internal/          derleyici, çalışma zamanı, formatter ve REPL
editors/vscode/    VS Code / Antigravity eklentisi
docs/              son kullanıcı rehberleri
examples/v0.1/     derlenmiş çalışan programlar
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
