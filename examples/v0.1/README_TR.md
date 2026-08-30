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

`Greeting.ahd`, `11_modules.ahd` tarafından kullanılan kardeş (sibling)
modüldür.
