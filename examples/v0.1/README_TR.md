# AhdCode v0.1 örnekleri

[English](README.md) · [Türkçe]

[Proje README'sine dön](../../README_TR.md)

Bu programlar, v0.1 diline küçük, çalışan girişlerdir (introductions).

```bash
ahdcode run examples/v0.1/01_hello.ahd
```

Girdi örnekleri etkileşimli olarak çalıştırılabilir:

```bash
ahdcode run examples/v0.1/02_input.ahd
ahdcode run examples/v0.1/14_grade_app.ahd
```

`16_latex.ahd` örneğini çalıştırmadan önce, `Latex` modülünün çevrimdışı derleyiciyi bir kez hazırlamanızı (stage) gerektirdiğini unutmayın:

```bash
go run ./tooling/latex/cmd/package-latex --output "$(go env GOPATH)"
```

| Örnek | Konu |
|---|---|
| `01_hello.ahd` | bildirimler, interpolasyon, çıktı |
| `02_input.ahd` | `take`, `int`, terminal girdisi |
| `03_grade_average.ahd` | List'ler ve Fundamentals indirgemeleri (reductions) |
| `04_loops.ahd` | `while`, sonradan kontrol eden `until`, `for`, `between` |
| `05_functions.ahd` | Function'lar, varsayılanlar, isimlendirilmiş çağrılar, callback'ler |
| `06_list_api.ahd` | List değişikliği, map/filter, deterministik shuffle |
| `07_string_api.ahd` | değiştirilemez (immutable) String işlemleri |
| `08_pair.ahd` | ekleme-sıralı (insertion-ordered) Pair akışı |
| `09_class.ahd` | yapı öznitelikleri (structure attributes) ve metotlar |
| `10_errors.ahd` | `attempt`, `except`, `ultimately`, `toss` |
| `11_modules.ahd` | `Greeting.ahd`'den doğrudan içe aktarım |
| `12_math.ahd` | açık Math modülü ve tohumlama (seeding) |
| `13_null_safety.ahd` | akış-duyarlı (flow-sensitive) null daraltma (refinement) |
| `14_grade_app.ahd` | kompakt, etkileşimli bir CLI uygulaması |
| `15_time.ahd` | Time modülü: DateTime, Duration, Calendar, monotonic |
| `16_latex.ahd` | Latex modülü: modül takma adı, yardımcılar, PDF, LatexError |
| `17_filesystem.ahd` | çıkarımlı (inferred) bildirimler, Path, UTF-8 File G/Ç, FileError |
| `18_protocols.ahd` | Class Protocol Methods, `type()`, `id()` |
| `19_regex.ahd` | Regex modülü: `Pattern`, match/find/replace/split/groups, `RegexError` |
| `20_lambda.ahd` | yalnızca ifade içeren Function değerleri, çıkarım, callback'ler ve normal Function karşılaştırması |
| `21_time_utc.ahd` | UTC, Unix milisaniyesi, sabit ofsetler ve anı koruyan dönüşüm |
| `22_csv.ahd` | ham CSV taşıma, başlık kayıtları, tırnaklama, Unicode ve çok satırlı alanlar |
| `23_data.ahd` | Data tabloları: CSV'den `Table`'a, filter, anahtarlı sort, derive, groupBy ve açık dönüşüm |
| `24_capture.ahd` | açık lambda yakalama listeleri, değere göre yakalama ve lambda/Function ayrımı |
| `25_statistics.ahd` | Statistics: sum, mean, median, mode, dağılım, quantile ve tanımsız girdi hataları |
| `26_data_statistics.ahd` | Data ve Statistics birlikte: pivotCount, açık dönüşüm ve yakalanmış bir eşik |
| `27_raw_strings.ahd` | Ham (raw) String literalleri: kaçış yok, interpolasyon yok, Regex niceleyicileri ve LaTeX kaynağı |
| `28_plot.ahd` | Plot modülü: line, scatter, bar, histogram, box, errorBar, birden çok seri, save ve subplot'lar |
| `29_data_plot.ahd` | Data, Regex (ham bir String deseniyle), açık dönüşüm, Statistics ve Plot birlikte |
| `30_plot_show.ahd` | `chart.show()`/`figure.show()` için elle yapılan duman testi -- otomatik CI'nin parçası değildir |
| `31_complex.ahd` | Complex literalleri, genişletme, işlemler ve kanonik çıktı |
| `32_numeric.ahd` | Numeric Vector/Matrix, ayrıştırmalar, solve ve özdeğerler |
| `33_numeric_plot.ahd` | Doğrudan Plot'a geçirilen Numeric `Vector` girdileri |
| `34_latex_report.ahd` | Article/Report seçenekleri, kapak, figure, theorem, reference ve bibliography |
| `35_latex_beamer.ahd` | çevrimdışı Beamer frame, contents, equation, table ve vurgu rengi |
| `36_full_workflow.ahd` | Data → Numeric/Statistics → Plot → Latex Report akışı |
| `37_word_document.ahd` | Word heading, biçimlendirilmiş paragraf, hizalama, sayfa sonu ve DOCX save |
| `38_word_read.ahd` | Word DOCX anlamsal okuma: text, heading, paragraph ve table |
| `39_word_plot.ahd` | Immutable Word Document içine gömülen Plot PNG'si |
| `40_word_table_merge.ahd` | yatay/dikey Word tablo merge'leri ve hizalama |
| `41_latex_beamer_themes.ahd` | sınırlı Default/Madrid/Warsaw Beamer theme'leri ve özel renk |
| `42_json.ahd` | JSON modülü: object/array oluşturma, parse, stringify, tipli erişimciler, get/at, JSONError |
| `43_xml.ahd` | XML modülü: element/text oluşturma, öznitelikler, parse, karışık içerik, stringify, XMLError |

`Greeting.ahd`, `11_modules.ahd` tarafından kullanılan kardeş (sibling)
modüldür.
